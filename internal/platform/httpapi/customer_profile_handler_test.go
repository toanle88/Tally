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

func TestCustomerProfileHandlerReturnsEstablishedSafeResult(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newHTTPTestPartyService(t, scopeID)
	party, err := partyService.Execute(context.Background(), organization.Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, organization.PartyCommand{Action: organization.PartyActionCreate, ScopeID: scopeID, Name: "Example Customer", PartyType: "customer", Status: "active", TaxIdentifier: httpStringPointer("TAX-PRIVATE-1234"), ContactMethods: []organization.PartyContactMethod{{Type: "email", Value: "finance@example.test"}}, Addresses: []organization.PartyAddress{{Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN"}}, IdempotencyKey: "http-party-create", CorrelationID: "correlation", CausationID: "causation"})
	if err != nil {
		t.Fatal(err)
	}
	profileService, err := organization.NewCustomerProfileService(
		organization.NewMemoryCustomerProfileRepository(partyRepository), partyRepository,
		organization.MemoryCustomerProfileAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.CustomerProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.MemoryCustomerProfileFieldAuthorizer{Decision: organization.CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true}},
		organization.AllowAllCustomerProfileApprovalValidator{}, &organization.MemoryCustomerProfileAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{CustomerProfileService: profileService}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":            mustRaw("create"),
		"partyId":           mustRaw(party.Party.ID.String()),
		"partyVersion":      mustRaw(1),
		"creditTerms":       mustRaw("net_30"),
		"creditLimit":       mustRaw(map[string]any{"amount": "1000.00", "currency": "USD"}),
		"billingPreference": mustRaw("invoice"),
		"taxTreatment":      mustRaw("standard"),
		"effectiveFrom":     mustRaw("2026-01-01"),
	}}
	response, err := handler.OmdMaintainCustomerProfiles(ctx, request, generated.OmdMaintainCustomerProfilesParams{IdempotencyKey: "profile-http-create"})
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
	if !strings.Contains(bodyText, "net_30") || !strings.Contains(bodyText, "partyVersion") || !strings.Contains(bodyText, "1000") || strings.Contains(bodyText, "TAX-PRIVATE-1234") {
		t.Fatalf("customer profile result = %s", bodyText)
	}
	if result.AggregateVersion != 1 {
		t.Fatalf("aggregate version = %d, want 1", result.AggregateVersion)
	}
}

func TestCustomerProfileHandlerRejectsForbiddenKeysAndScopeConflict(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, _ := newHTTPTestPartyService(t, scopeID)
	profileService, err := organization.NewCustomerProfileService(
		organization.NewMemoryCustomerProfileRepository(partyRepository), partyRepository,
		organization.MemoryCustomerProfileAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.CustomerProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.MemoryCustomerProfileFieldAuthorizer{Decision: organization.CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true}},
		organization.AllowAllCustomerProfileApprovalValidator{}, &organization.MemoryCustomerProfileAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{CustomerProfileService: profileService}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":            mustRaw("create"),
		"partyId":           mustRaw(uuid.New().String()),
		"partyVersion":      mustRaw(1),
		"creditTerms":       mustRaw("net_30"),
		"creditLimit":       mustRaw(map[string]any{"amount": "1", "currency": "USD", "accountNumber": "123456789"}),
		"billingPreference": mustRaw("invoice"),
		"taxTreatment":      mustRaw("standard"),
		"effectiveFrom":     mustRaw("2026-01-01"),
	}}
	response, err := handler.OmdMaintainCustomerProfiles(ctx, request, generated.OmdMaintainCustomerProfilesParams{IdempotencyKey: "profile-http-raw"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.OmdMaintainCustomerProfilesBadRequest); !ok {
		t.Fatalf("forbidden-key response = %T, want bad request", response)
	}

	request = &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":            mustRaw("create"),
		"scopeId":           mustRaw(uuid.New().String()),
		"partyId":           mustRaw(uuid.New().String()),
		"partyVersion":      mustRaw(1),
		"creditTerms":       mustRaw("net_30"),
		"creditLimit":       mustRaw(map[string]any{"amount": "1", "currency": "USD"}),
		"billingPreference": mustRaw("invoice"),
		"taxTreatment":      mustRaw("standard"),
		"effectiveFrom":     mustRaw("2026-01-01"),
	}}
	response, err = handler.OmdMaintainCustomerProfiles(ctx, request, generated.OmdMaintainCustomerProfilesParams{IdempotencyKey: "profile-http-scope"})
	if err != nil {
		t.Fatal(err)
	}
	conflict, ok := response.(*generated.OmdMaintainCustomerProfilesConflict)
	if !ok || conflict.Code != "VERSION_CONFLICT" {
		t.Fatalf("scope conflict response = %#v", response)
	}
}

func newHTTPTestPartyService(t *testing.T, scopeID uuid.UUID) (*organization.MemoryPartyRepository, *organization.PartyService) {
	t.Helper()
	partyRepository := organization.NewMemoryPartyRepository()
	partyService, err := organization.NewPartyService(
		partyRepository,
		organization.MemoryPartyAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.AllowAllPartyFieldAuthorizer{}, organization.AllowAllPartyBankReferenceValidator{}, organization.AllowAllPartyBankControlEvaluator{}, &organization.MemoryPartyAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	return partyRepository, partyService
}

func httpStringPointer(value string) *string {
	return &value
}
