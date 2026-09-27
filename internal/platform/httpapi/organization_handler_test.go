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

func TestOrganizationHandlerMaintainsAndReadsSafeLegalEntityProjection(t *testing.T) {
	scopeID := uuid.New()
	audit := &organization.MemoryAuditRecorder{}
	service, err := organization.NewLegalEntityService(
		organization.NewMemoryLegalEntityRepository(),
		organization.MemoryAuthorizer{Decision: organization.AuthorizationDecision{
			Allowed: true, Permission: organization.LegalEntityManagementPermission,
			DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
		}},
		organization.AllowAllApprovalValidator{}, audit,
		func() time.Time { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{OrganizationService: service}
	request := &generated.CommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)),
		Data: generated.CommandRequestData{
			"action":               mustRaw("create"),
			"legalName":            mustRaw("Safe Projection Holdings"),
			"functionalCurrency":   mustRaw("VND"),
			"presentationCurrency": mustRaw("VND"),
			"taxRegistrationId":    mustRaw("VN-PRIVATE-999999"),
			"effectiveFrom":        mustRaw("2026-01-01"),
			"registrations": mustRaw([]map[string]any{{
				"type": "company", "identifier": "123456789", "jurisdiction": "VN", "effectiveFrom": "2026-01-01",
			}}),
			"addresses": mustRaw([]map[string]any{{
				"type": "registered", "line1": "1 Main Street", "locality": "Hanoi", "postalCode": "100000", "countryCode": "VN", "effectiveFrom": "2026-01-01",
			}}),
			"ownershipInterests": mustRaw([]map[string]any{{
				"ownerReference": "owner-1", "percentage": "100", "effectiveFrom": "2026-01-01",
			}}),
		},
	}
	created, err := handler.OmdMaintainLegalEntities(ctx, request, generated.OmdMaintainLegalEntitiesParams{IdempotencyKey: "omd-safe-projection-1"})
	if err != nil {
		t.Fatal(err)
	}
	createdResult, ok := created.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("create response = %T, want established result", created)
	}
	createdBody, err := json.Marshal(createdResult)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(createdBody), "VN-PRIVATE-999999") || strings.Contains(string(createdBody), "taxRegistrationId") {
		t.Fatalf("restricted tax registration leaked from create response: %s", createdBody)
	}

	list, err := handler.OmdListLegalEntities(ctx, generated.OmdListLegalEntitiesParams{
		XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID)),
	})
	if err != nil {
		t.Fatal(err)
	}
	listResult, ok := list.(*generated.EstablishedResult)
	if !ok || !strings.Contains(string(listResult.Data["items"]), "Safe Projection Holdings") {
		t.Fatalf("list response = %#v, want safe legal-entity item", list)
	}

	get, err := handler.OmdGetLegalEntity(ctx, generated.OmdGetLegalEntityParams{
		LegalEntityId:      createdResult.AggregateId,
		XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID)),
	})
	if err != nil {
		t.Fatal(err)
	}
	getResult, ok := get.(*generated.EstablishedResult)
	if !ok || !strings.Contains(string(getResult.Data["legalEntity"]), "identifierMasked") {
		t.Fatalf("get response = %#v, want masked registration", get)
	}
	if strings.Contains(string(getResult.Data["legalEntity"]), "123456789") {
		t.Fatalf("restricted registration leaked from get response: %s", getResult.Data["legalEntity"])
	}
	if len(audit.Records) != 1 {
		t.Fatalf("audit records = %d, want one mutation record", len(audit.Records))
	}
}

func TestOrganizationHandlerDeniesUnauthorizedLegalEntityRead(t *testing.T) {
	service, err := organization.NewLegalEntityService(
		organization.NewMemoryLegalEntityRepository(),
		organization.MemoryAuthorizer{Decision: organization.AuthorizationDecision{
			Allowed: false, Permission: organization.LegalEntityReadPermission, DecisionReference: uuid.New(),
		}},
		organization.AllowAllApprovalValidator{}, &organization.MemoryAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	response, err := (IdentityHandler{OrganizationService: service}).OmdListLegalEntities(ctx, generated.OmdListLegalEntitiesParams{})
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok || string(result.Data["items"]) != "[]" {
		t.Fatalf("response = %#v, want an empty filtered projection", response)
	}
}
