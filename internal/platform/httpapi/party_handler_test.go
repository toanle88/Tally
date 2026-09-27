package httpapi

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestPartyHandlerReturnsSafeEstablishedResultAndBankControl(t *testing.T) {
	scopeID := uuid.New()
	control := organization.MemoryPartyBankControlEvaluator{State: organization.BankDetailControlState{Status: organization.PartyBankControlPending}}
	service, err := organization.NewPartyService(
		organization.NewMemoryPartyRepository(),
		organization.MemoryPartyAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.AllowAllPartyFieldAuthorizer{}, organization.AllowAllPartyBankReferenceValidator{}, &control, &organization.MemoryPartyAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{PartyService: service}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action": mustRaw("create"), "name": mustRaw("Example Vendor"), "partyType": mustRaw("vendor"), "status": mustRaw("active"), "taxIdentifier": mustRaw("TAX-PRIVATE-1234"),
		"contactMethods":       mustRaw([]map[string]any{{"type": "email", "value": "finance@example.test"}}),
		"addresses":            mustRaw([]map[string]any{{"type": "registered", "line1": "1 Main Street", "locality": "Hanoi", "postalCode": "100000", "countryCode": "VN"}}),
		"classifications":      mustRaw([]map[string]any{{"code": "segment", "value": "supplier"}}),
		"bankDetailReferences": mustRaw([]map[string]any{{"reference": "provider-ref-001", "providerCode": "provider-a", "consentReference": "consent-ref-001"}}),
	}}
	response, err := handler.OmdMaintainParties(ctx, request, generated.OmdMaintainPartiesParams{IdempotencyKey: "party-api-create-1"})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("response = %T, want established result", response)
	}
	body, err := json.Marshal(result)
	if err != nil {
		t.Fatal(err)
	}
	bodyText := string(body)
	if strings.Contains(bodyText, "TAX-PRIVATE-1234") || strings.Contains(bodyText, "accountNumber") || strings.Contains(bodyText, "consent-ref-001") {
		t.Fatalf("restricted party value leaked: %s", bodyText)
	}
	if !strings.Contains(bodyText, "provider-ref-001") || !strings.Contains(bodyText, "pending") || result.AggregateVersion != 1 {
		t.Fatalf("safe party result = %s", bodyText)
	}
}

func TestPartyHandlerRejectsRawBankKeysAndConflictingScope(t *testing.T) {
	scopeID := uuid.New()
	service, err := organization.NewPartyService(
		organization.NewMemoryPartyRepository(),
		organization.MemoryPartyAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.AllowAllPartyFieldAuthorizer{}, organization.AllowAllPartyBankReferenceValidator{}, organization.AllowAllPartyBankControlEvaluator{}, &organization.MemoryPartyAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{PartyService: service}
	raw := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action": mustRaw("create"), "name": mustRaw("Example Vendor"), "partyType": mustRaw("vendor"), "status": mustRaw("active"),
		"bankDetailReferences": mustRaw([]map[string]any{{"reference": "provider-ref-001", "accountNumber": "123456789"}}),
	}}
	response, err := handler.OmdMaintainParties(ctx, raw, generated.OmdMaintainPartiesParams{IdempotencyKey: "party-api-raw-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.OmdMaintainPartiesBadRequest); !ok {
		t.Fatalf("raw bank response = %T, want bad request", response)
	}
	conflicting := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action": mustRaw("create"), "scopeId": mustRaw(uuid.New().String()), "name": mustRaw("Example Vendor"), "partyType": mustRaw("vendor"), "status": mustRaw("active"),
	}}
	response, err = handler.OmdMaintainParties(ctx, conflicting, generated.OmdMaintainPartiesParams{IdempotencyKey: "party-api-scope-1"})
	if err != nil {
		t.Fatal(err)
	}
	if conflict, ok := response.(*generated.OmdMaintainPartiesConflict); !ok || conflict.Code != "VERSION_CONFLICT" {
		t.Fatalf("scope conflict response = %#v", response)
	}
}
