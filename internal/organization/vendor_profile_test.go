package organization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestVendorProfileServiceMaintainsReplaysAndPreservesRevisions(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	partyResult, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "vendor"))
	if err != nil {
		t.Fatal(err)
	}
	audit := &MemoryVendorProfileAuditRecorder{}
	service, err := NewVendorProfileService(
		NewMemoryVendorProfileRepository(partyRepository),
		partyRepository,
		MemoryVendorProfileAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		MemoryVendorProfileFieldAuthorizer{Decision: VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}},
		AllowAllVendorProfileApprovalValidator{}, audit, profileTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "vendor-profile-actor"}
	create := vendorProfileTestCommand(scopeID, partyResult.Party.ID, partyResult.Party.Version.Value(), VendorProfileActionCreate)
	first, err := service.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if first.VendorProfile.Version.Value() != 1 || first.VendorProfile.PartyVersion.Value() != 1 || first.VendorProfile.PaymentTerms != "net_30" {
		t.Fatalf("created vendor profile = %#v", first.VendorProfile)
	}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "accountNumber") || strings.Contains(string(body), "TAX-PRIVATE") {
		t.Fatalf("restricted Party data leaked into vendor profile result: %s", body)
	}
	replay, err := service.Execute(context.Background(), actor, create)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	maintain := vendorProfileTestCommand(scopeID, partyResult.Party.ID, partyResult.Party.Version.Value(), VendorProfileActionMaintain)
	maintain.VendorProfileID = first.VendorProfile.ID
	maintain.ExpectedVersion = aggregateVersionPointer(1)
	maintain.IdempotencyKey = "vendor-profile-maintain-1"
	maintain.PaymentTerms = "net_45"
	updated, err := service.Execute(context.Background(), actor, maintain)
	if err != nil {
		t.Fatal(err)
	}
	if updated.VendorProfile.Version.Value() != 2 || updated.VendorProfile.PaymentTerms != "net_45" {
		t.Fatalf("maintained vendor profile = %#v", updated.VendorProfile)
	}
	stored, err := service.repository.Get(context.Background(), first.VendorProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Revisions) != 2 || len(audit.Records) != 2 || audit.Records[1].BeforeFingerprint == audit.Records[1].AfterFingerprint {
		t.Fatalf("stored revisions/audit = %d/%d, audit=%#v", len(stored.Revisions), len(audit.Records), audit.Records)
	}
}

func TestVendorProfileServiceRejectsWrongPartyAndRestrictedFieldAccess(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	customer, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "customer"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewVendorProfileService(
		NewMemoryVendorProfileRepository(partyRepository), partyRepository,
		allowVendorProfileAuthorization{scope: scopeID},
		allowVendorProfileFields{VendorProfileFieldAuthorization{PaymentTerms: true}},
		AllowAllVendorProfileApprovalValidator{}, &MemoryVendorProfileAuditRecorder{}, profileTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	command := vendorProfileTestCommand(scopeID, customer.Party.ID, customer.Party.Version.Value(), VendorProfileActionCreate)
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "vendor-profile-actor"}, command); !errors.Is(err, ErrVendorProfilePartyInvalid) {
		t.Fatalf("customer Party error = %v, want invalid Party", err)
	}

	vendor, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor-2"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "vendor"))
	if err != nil {
		t.Fatal(err)
	}
	command = vendorProfileTestCommand(scopeID, vendor.Party.ID, vendor.Party.Version.Value(), VendorProfileActionCreate)
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "vendor-profile-actor-2"}, command); !errors.Is(err, ErrVendorProfileFieldAuthorizationDenied) {
		t.Fatalf("field authorization error = %v, want field denial", err)
	}
}

func TestVendorProfileEndDateStopsFurtherMaintenance(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	party, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "vendor"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewVendorProfileService(
		NewMemoryVendorProfileRepository(partyRepository), partyRepository,
		MemoryVendorProfileAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		MemoryVendorProfileFieldAuthorizer{Decision: VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}},
		AllowAllVendorProfileApprovalValidator{}, &MemoryVendorProfileAuditRecorder{}, profileTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "vendor-profile-actor"}
	created, err := service.Execute(context.Background(), actor, vendorProfileTestCommand(scopeID, party.Party.ID, party.Party.Version.Value(), VendorProfileActionCreate))
	if err != nil {
		t.Fatal(err)
	}
	end := vendorProfileTestCommand(scopeID, party.Party.ID, party.Party.Version.Value(), VendorProfileActionMaintain)
	end.VendorProfileID, end.ExpectedVersion, end.IdempotencyKey = created.VendorProfile.ID, aggregateVersionPointer(1), "vendor-profile-end"
	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	end.EffectiveTo = &endDate
	ended, err := service.Execute(context.Background(), actor, end)
	if err != nil || ended.VendorProfile.Status != VendorProfileStatusEndDated || ended.VendorProfile.NextAction != "view history" {
		t.Fatalf("end-dated result = %#v, err=%v", ended, err)
	}
	next := vendorProfileTestCommand(scopeID, party.Party.ID, party.Party.Version.Value(), VendorProfileActionMaintain)
	next.VendorProfileID, next.ExpectedVersion, next.IdempotencyKey = created.VendorProfile.ID, aggregateVersionPointer(2), "vendor-profile-after-end"
	if _, err := service.Execute(context.Background(), actor, next); !errors.Is(err, ErrInvalidVendorProfile) {
		t.Fatalf("post-end maintenance error = %v, want invalid profile", err)
	}
}

func TestVendorProfileServiceFailsClosedForStaleAndUnavailableAuthorization(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	party, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "vendor"))
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name    string
		outcome string
		want    error
	}{
		{name: "stale", outcome: "stale", want: ErrVendorProfileAuthorizationStale},
		{name: "unavailable", outcome: "unavailable", want: ErrVendorProfileAuthorizationUnavailable},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			service, err := NewVendorProfileService(
				NewMemoryVendorProfileRepository(partyRepository), partyRepository,
				MemoryVendorProfileAuthorizer{Decision: AuthorizationDecision{Outcome: testCase.outcome, Permission: VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
				MemoryVendorProfileFieldAuthorizer{Decision: VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}},
				AllowAllVendorProfileApprovalValidator{}, &MemoryVendorProfileAuditRecorder{}, profileTestClock,
			)
			if err != nil {
				t.Fatal(err)
			}
			command := vendorProfileTestCommand(scopeID, party.Party.ID, party.Party.Version.Value(), VendorProfileActionCreate)
			if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "vendor-profile-actor"}, command); !errors.Is(err, testCase.want) {
				t.Fatalf("authorization error = %v, want %v", err, testCase.want)
			}
		})
	}
}

type allowVendorProfileAuthorization struct{ scope uuid.UUID }

func (authorization allowVendorProfileAuthorization) AuthorizeVendorProfile(context.Context, Actor, VendorProfileCommand, *VendorProfile) (AuthorizationDecision, error) {
	return AuthorizationDecision{Allowed: true, Permission: VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{authorization.scope}}, nil
}

type allowVendorProfileFields struct {
	VendorProfileFieldAuthorization
}

func (authorization allowVendorProfileFields) AuthorizeVendorProfileFields(context.Context, Actor, VendorProfileCommand, *VendorProfile) (VendorProfileFieldAuthorization, error) {
	return authorization.VendorProfileFieldAuthorization, nil
}

func vendorProfileTestCommand(scopeID, partyID uuid.UUID, partyVersion int64, action string) VendorProfileCommand {
	return VendorProfileCommand{Action: action, ScopeID: scopeID, PartyID: partyID, ExpectedPartyVersion: aggregateVersionPointer(partyVersion), PaymentTerms: "net_30", WithholdingTreatment: "standard", RemittancePreference: "bank_transfer", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), IdempotencyKey: "vendor-profile-" + action, CorrelationID: "correlation", CausationID: "causation"}
}
