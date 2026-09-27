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

func TestCustomerProfileServiceMaintainsAndReplaysIdempotently(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	partyResult, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "customer"))
	if err != nil {
		t.Fatal(err)
	}

	audit := &MemoryCustomerProfileAuditRecorder{}
	service, err := NewCustomerProfileService(
		NewMemoryCustomerProfileRepository(partyRepository),
		partyRepository,
		MemoryCustomerProfileAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: CustomerProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		MemoryCustomerProfileFieldAuthorizer{Decision: CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true}},
		AllowAllCustomerProfileApprovalValidator{}, audit, profileTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "profile-actor"}
	create := customerProfileTestCommand(scopeID, partyResult.Party.ID, partyResult.Party.Version.Value(), CustomerProfileActionCreate)
	first, err := service.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if first.CustomerProfile.Version.Value() != 1 || first.CustomerProfile.PartyVersion.Value() != 1 || first.CustomerProfile.CreditLimit == nil || first.CustomerProfile.CreditLimit.Amount != "1000" {
		t.Fatalf("created customer profile = %#v", first.CustomerProfile)
	}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "accountNumber") || strings.Contains(string(body), "TAX-PRIVATE") {
		t.Fatalf("restricted party data leaked into customer profile result: %s", body)
	}
	replay, err := service.Execute(context.Background(), actor, create)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	maintain := customerProfileTestCommand(scopeID, partyResult.Party.ID, partyResult.Party.Version.Value(), CustomerProfileActionMaintain)
	maintain.CustomerProfileID = first.CustomerProfile.ID
	maintain.ExpectedVersion = aggregateVersionPointer(1)
	maintain.IdempotencyKey = "profile-maintain-1"
	maintain.CreditLimit = CreditLimit{Amount: "1250.00", Currency: "USD"}
	updated, err := service.Execute(context.Background(), actor, maintain)
	if err != nil {
		t.Fatal(err)
	}
	if updated.CustomerProfile.Version.Value() != 2 || updated.CustomerProfile.CreditLimit.Amount != "1250" {
		t.Fatalf("maintained customer profile = %#v", updated.CustomerProfile)
	}
	stored, err := service.repository.Get(context.Background(), first.CustomerProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Revisions) != 2 || len(audit.Records) != 2 {
		t.Fatalf("stored revisions/audit = %d/%d, want 2/2", len(stored.Revisions), len(audit.Records))
	}
}

func TestCustomerProfileServiceRejectsStalePartyAndDoesNotMutate(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	partyActor := Actor{UserID: uuid.New(), SubjectReference: "party-actor"}
	partyResult, err := partyService.Execute(context.Background(), partyActor, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "customer"))
	if err != nil {
		t.Fatal(err)
	}
	profileRepository := NewMemoryCustomerProfileRepository(partyRepository)
	profileService, err := NewCustomerProfileService(profileRepository, partyRepository, allowCustomerProfileAuthorization{scope: scopeID}, allowCustomerProfileFields{CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true}}, AllowAllCustomerProfileApprovalValidator{}, &MemoryCustomerProfileAuditRecorder{}, profileTestClock)
	if err != nil {
		t.Fatal(err)
	}
	profileActor := Actor{UserID: uuid.New(), SubjectReference: "profile-actor"}
	create := customerProfileTestCommand(scopeID, partyResult.Party.ID, 1, CustomerProfileActionCreate)
	created, err := profileService.Execute(context.Background(), profileActor, create)
	if err != nil {
		t.Fatal(err)
	}

	partyMaintain := customerProfileTestPartyCommand(scopeID, PartyActionMaintain, "customer")
	partyMaintain.PartyID = partyResult.Party.ID
	partyMaintain.ExpectedVersion = aggregateVersionPointer(1)
	partyMaintain.IdempotencyKey = "party-update-before-profile"
	partyMaintain.Name = "Example Customer Updated"
	if _, err := partyService.Execute(context.Background(), partyActor, partyMaintain); err != nil {
		t.Fatal(err)
	}

	maintain := customerProfileTestCommand(scopeID, partyResult.Party.ID, 1, CustomerProfileActionMaintain)
	maintain.CustomerProfileID = created.CustomerProfile.ID
	maintain.ExpectedVersion = aggregateVersionPointer(1)
	maintain.IdempotencyKey = "profile-stale-party"
	if _, err := profileService.Execute(context.Background(), profileActor, maintain); !errors.Is(err, ErrCustomerProfilePartyVersionConflict) {
		t.Fatalf("stale Party error = %v, want Party version conflict", err)
	}
	stored, err := profileRepository.Get(context.Background(), created.CustomerProfile.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 1 || stored.PartyVersion.Value() != 1 {
		t.Fatalf("stale Party changed profile = %#v", stored)
	}
}

func TestCustomerProfileServiceRejectsNonCustomerPartyAndFieldAccess(t *testing.T) {
	scopeID := uuid.New()
	partyRepository, partyService := newCustomerProfileTestPartyService(t, scopeID)
	vendor, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "vendor"))
	if err != nil {
		t.Fatal(err)
	}
	service, err := NewCustomerProfileService(
		NewMemoryCustomerProfileRepository(partyRepository), partyRepository,
		allowCustomerProfileAuthorization{scope: scopeID}, allowCustomerProfileFields{CustomerProfileFieldAuthorization{CreditTerms: true}}, AllowAllCustomerProfileApprovalValidator{}, &MemoryCustomerProfileAuditRecorder{}, profileTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	command := customerProfileTestCommand(scopeID, vendor.Party.ID, vendor.Party.Version.Value(), CustomerProfileActionCreate)
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "profile-actor"}, command); !errors.Is(err, ErrCustomerProfilePartyInvalid) {
		t.Fatalf("vendor Party error = %v, want invalid Party", err)
	}

	customer, err := partyService.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "party-actor-2"}, customerProfileTestPartyCommand(scopeID, PartyActionCreate, "customer"))
	if err != nil {
		t.Fatal(err)
	}
	command = customerProfileTestCommand(scopeID, customer.Party.ID, customer.Party.Version.Value(), CustomerProfileActionCreate)
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "profile-actor-2"}, command); !errors.Is(err, ErrCustomerProfileFieldAuthorizationDenied) {
		t.Fatalf("field authorization error = %v, want field denial", err)
	}
}

func TestCustomerProfileRejectsNegativeCreditLimit(t *testing.T) {
	limit := CreditLimit{Amount: "-0.01", Currency: "USD"}
	if _, err := limit.Canonicalize(); !errors.Is(err, ErrInvalidCustomerProfile) {
		t.Fatalf("negative credit limit error = %v, want invalid profile", err)
	}
}

func TestCustomerProfileCreateRejectsEndDate(t *testing.T) {
	command := customerProfileTestCommand(uuid.New(), uuid.New(), 1, CustomerProfileActionCreate)
	endDate := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)
	command.EffectiveTo = &endDate
	if err := command.Validate(); !errors.Is(err, ErrInvalidCustomerProfileCommand) {
		t.Fatalf("create end-date error = %v, want invalid customer profile command", err)
	}
}

type allowCustomerProfileAuthorization struct{ scope uuid.UUID }

func (authorization allowCustomerProfileAuthorization) AuthorizeCustomerProfile(context.Context, Actor, CustomerProfileCommand, *CustomerProfile) (AuthorizationDecision, error) {
	return AuthorizationDecision{Allowed: true, Permission: CustomerProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{authorization.scope}}, nil
}

type allowCustomerProfileFields struct {
	CustomerProfileFieldAuthorization
}

func (authorization allowCustomerProfileFields) AuthorizeCustomerProfileFields(context.Context, Actor, CustomerProfileCommand, *CustomerProfile) (CustomerProfileFieldAuthorization, error) {
	return authorization.CustomerProfileFieldAuthorization, nil
}

func newCustomerProfileTestPartyService(t *testing.T, scopeID uuid.UUID) (*MemoryPartyRepository, *PartyService) {
	t.Helper()
	partyRepository := NewMemoryPartyRepository()
	partyService, err := NewPartyService(
		partyRepository,
		MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		AllowAllPartyFieldAuthorizer{}, AllowAllPartyBankReferenceValidator{}, AllowAllPartyBankControlEvaluator{}, &MemoryPartyAuditRecorder{}, profileTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	return partyRepository, partyService
}

func customerProfileTestPartyCommand(scopeID uuid.UUID, action string, partyType ...string) PartyCommand {
	typeValue := "customer"
	if len(partyType) > 0 {
		typeValue = partyType[0]
	}
	name := "Example Customer"
	if typeValue == "vendor" {
		name = "Example Vendor"
	}
	return PartyCommand{Action: action, ScopeID: scopeID, Name: name, PartyType: PartyType(typeValue), Status: "active", ContactMethods: []PartyContactMethod{{Type: "email", Value: "finance@example.test"}}, Addresses: []PartyAddress{{Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN"}}, Classifications: []PartyClassification{{Code: "segment", Value: typeValue}}, IdempotencyKey: "party-" + action + "-" + typeValue, CorrelationID: "correlation", CausationID: "causation"}
}

func customerProfileTestCommand(scopeID, partyID uuid.UUID, partyVersion int64, action string) CustomerProfileCommand {
	return CustomerProfileCommand{Action: action, ScopeID: scopeID, PartyID: partyID, ExpectedPartyVersion: aggregateVersionPointer(partyVersion), CreditTerms: "net_30", CreditLimit: CreditLimit{Amount: "1000.00", Currency: "USD"}, BillingPreference: "invoice", TaxTreatment: "standard", EffectiveFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), IdempotencyKey: "profile-" + action, CorrelationID: "correlation", CausationID: "causation"}
}

func profileTestClock() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}
