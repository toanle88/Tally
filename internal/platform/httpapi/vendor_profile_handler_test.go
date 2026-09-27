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

func TestVendorProfileHandlerReturnsEstablishedSafeResult(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newHTTPTestPartyService(t, scopeID)
	party, err := partyService.Execute(context.Background(), organization.Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, organization.PartyCommand{Action: organization.PartyActionCreate, ScopeID: scopeID, Name: "Example Vendor", PartyType: "vendor", Status: "active", TaxIdentifier: httpStringPointer("TAX-PRIVATE-1234"), ContactMethods: []organization.PartyContactMethod{{Type: "email", Value: "finance@example.test"}}, Addresses: []organization.PartyAddress{{Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN"}}, BankDetailReferences: []organization.PartyBankDetailReference{{ID: uuid.New(), Reference: "provider-ref-001", ProviderCode: "vault"}}, IdempotencyKey: "http-vendor-party-create", CorrelationID: "correlation", CausationID: "causation"})
	if err != nil {
		t.Fatal(err)
	}
	profileService, err := organization.NewVendorProfileService(
		organization.NewMemoryVendorProfileRepository(partyRepository), partyRepository,
		organization.MemoryVendorProfileAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.MemoryVendorProfileFieldAuthorizer{Decision: organization.VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}},
		organization.AllowAllVendorProfileApprovalValidator{}, &organization.MemoryVendorProfileAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{VendorProfileService: profileService}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":               mustRaw("create"),
		"partyId":              mustRaw(party.Party.ID.String()),
		"partyVersion":         mustRaw(1),
		"paymentTerms":         mustRaw("net_30"),
		"withholdingTreatment": mustRaw("standard"),
		"remittancePreference": mustRaw("bank_transfer"),
		"effectiveFrom":        mustRaw("2026-01-01"),
	}}
	response, err := handler.OmdMaintainVendorProfiles(ctx, request, generated.OmdMaintainVendorProfilesParams{IdempotencyKey: "vendor-profile-http-create"})
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
	if !strings.Contains(bodyText, "net_30") || !strings.Contains(bodyText, "partyVersion") || strings.Contains(bodyText, "TAX-PRIVATE-1234") || strings.Contains(bodyText, "provider-ref-001") || strings.Contains(bodyText, "accountNumber") {
		t.Fatalf("vendor profile result = %s", bodyText)
	}
	if result.AggregateVersion != 1 {
		t.Fatalf("aggregate version = %d, want 1", result.AggregateVersion)
	}
}

func TestVendorProfileHandlerRejectsRawBankDataAndScopeConflict(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, _ := newHTTPTestPartyService(t, scopeID)
	profileService, err := organization.NewVendorProfileService(
		organization.NewMemoryVendorProfileRepository(partyRepository), partyRepository,
		organization.MemoryVendorProfileAuthorizer{Decision: organization.AuthorizationDecision{Allowed: true, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		organization.MemoryVendorProfileFieldAuthorizer{Decision: organization.VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}},
		organization.AllowAllVendorProfileApprovalValidator{}, &organization.MemoryVendorProfileAuditRecorder{}, time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{VendorProfileService: profileService}
	request := &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":               mustRaw("create"),
		"partyId":              mustRaw(uuid.New().String()),
		"partyVersion":         mustRaw(1),
		"paymentTerms":         mustRaw("net_30"),
		"withholdingTreatment": mustRaw("standard"),
		"remittancePreference": mustRaw("bank_transfer"),
		"effectiveFrom":        mustRaw("2026-01-01"),
		"bankDetails":          mustRaw(map[string]any{"accountNumber": "123456789"}),
	}}
	response, err := handler.OmdMaintainVendorProfiles(ctx, request, generated.OmdMaintainVendorProfilesParams{IdempotencyKey: "vendor-profile-http-raw"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.OmdMaintainVendorProfilesBadRequest); !ok {
		t.Fatalf("forbidden-key response = %T, want bad request", response)
	}

	request = &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":               mustRaw("create"),
		"scopeId":              mustRaw(uuid.New().String()),
		"partyId":              mustRaw(uuid.New().String()),
		"partyVersion":         mustRaw(1),
		"paymentTerms":         mustRaw("net_30"),
		"withholdingTreatment": mustRaw("standard"),
		"remittancePreference": mustRaw("bank_transfer"),
		"effectiveFrom":        mustRaw("2026-01-01"),
	}}
	response, err = handler.OmdMaintainVendorProfiles(ctx, request, generated.OmdMaintainVendorProfilesParams{IdempotencyKey: "vendor-profile-http-scope"})
	if err != nil {
		t.Fatal(err)
	}
	conflict, ok := response.(*generated.OmdMaintainVendorProfilesConflict)
	if !ok || conflict.Code != "VERSION_CONFLICT" {
		t.Fatalf("scope conflict response = %#v", response)
	}
}

func TestVendorProfileHandlerMapsPolicyDeniedStaleAndUnavailableOutcomes(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newHTTPTestPartyService(t, scopeID)
	party, err := partyService.Execute(context.Background(), organization.Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, organization.PartyCommand{Action: organization.PartyActionCreate, ScopeID: scopeID, Name: "Policy Vendor", PartyType: "vendor", Status: "active", ContactMethods: []organization.PartyContactMethod{{Type: "email", Value: "finance@example.test"}}, Addresses: []organization.PartyAddress{{Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN"}}, IdempotencyKey: "policy-vendor-party", CorrelationID: "correlation", CausationID: "causation"})
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name       string
		outcome    string
		wantStatus any
	}{
		{name: "denied", outcome: "denied", wantStatus: (*generated.OmdMaintainVendorProfilesForbidden)(nil)},
		{name: "stale", outcome: "stale", wantStatus: (*generated.OmdMaintainVendorProfilesConflict)(nil)},
		{name: "unavailable", outcome: "unavailable", wantStatus: (*generated.OmdMaintainVendorProfilesServiceUnavailable)(nil)},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			profileService, err := organization.NewVendorProfileService(
				organization.NewMemoryVendorProfileRepository(partyRepository), partyRepository,
				organization.MemoryVendorProfileAuthorizer{Decision: organization.AuthorizationDecision{Outcome: testCase.outcome, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
				organization.MemoryVendorProfileFieldAuthorizer{Decision: organization.VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}},
				organization.AllowAllVendorProfileApprovalValidator{}, &organization.MemoryVendorProfileAuditRecorder{}, time.Now,
			)
			if err != nil {
				t.Fatal(err)
			}
			response, err := (IdentityHandler{VendorProfileService: profileService}).OmdMaintainVendorProfiles(ctx, vendorProfileHTTPCreateRequest(scopeID, party.Party.ID), generated.OmdMaintainVendorProfilesParams{IdempotencyKey: "policy-" + testCase.name})
			if err != nil {
				t.Fatal(err)
			}
			switch testCase.wantStatus.(type) {
			case *generated.OmdMaintainVendorProfilesForbidden:
				if _, ok := response.(*generated.OmdMaintainVendorProfilesForbidden); !ok {
					t.Fatalf("response = %T, want forbidden", response)
				}
			case *generated.OmdMaintainVendorProfilesConflict:
				if result, ok := response.(*generated.OmdMaintainVendorProfilesConflict); !ok || result.Code != "POLICY_STALE" {
					t.Fatalf("response = %#v, want policy-stale conflict", response)
				}
			case *generated.OmdMaintainVendorProfilesServiceUnavailable:
				if _, ok := response.(*generated.OmdMaintainVendorProfilesServiceUnavailable); !ok {
					t.Fatalf("response = %T, want service unavailable", response)
				}
			}
		})
	}
}

func vendorProfileHTTPCreateRequest(scopeID, partyID uuid.UUID) *generated.CommandRequest {
	return &generated.CommandRequest{CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: generated.CommandRequestData{
		"action":               mustRaw("create"),
		"partyId":              mustRaw(partyID.String()),
		"partyVersion":         mustRaw(1),
		"paymentTerms":         mustRaw("net_30"),
		"withholdingTreatment": mustRaw("standard"),
		"remittancePreference": mustRaw("bank_transfer"),
		"effectiveFrom":        mustRaw("2026-01-01"),
	}}
}
