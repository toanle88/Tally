package organization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestPartyServiceReturnsSafeProjectionAndReplaysIdempotently(t *testing.T) {
	scopeID := uuid.New()
	audit := &MemoryPartyAuditRecorder{}
	service, err := NewPartyService(
		NewMemoryPartyRepository(),
		MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		AllowAllPartyFieldAuthorizer{},
		AllowAllPartyBankReferenceValidator{},
		AllowAllPartyBankControlEvaluator{},
		audit,
		partyTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	command := partyTestCommand(scopeID, PartyActionCreate)
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Party.Name != "Example Vendor" || first.Party.Version.Value() != 1 || first.ValidationOutcome != "accepted" {
		t.Fatalf("first result = %#v", first)
	}
	body, err := json.Marshal(first)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(body), "TAX-PRIVATE-1234") || strings.Contains(string(body), "account") {
		t.Fatalf("restricted party data leaked: %s", body)
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil || !replay.Replayed {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	if len(audit.Records) != 1 {
		t.Fatalf("audit records = %d, want one", len(audit.Records))
	}
	record := audit.Records[0]
	if record.ActorUserID != actor.UserID || record.ScopeID != scopeID || record.Permission != PartyManagementPermission || record.CorrelationID != command.CorrelationID || record.CausationID != command.CausationID {
		t.Fatalf("audit evidence = %#v, want actor/scope/permission/correlation/causation", record)
	}
	auditBody, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(auditBody), "TAX-PRIVATE-1234") || strings.Contains(string(auditBody), "accountNumber") {
		t.Fatalf("audit evidence leaked restricted data: %s", auditBody)
	}
}

func TestPartyServiceFieldAuthorizationIsIndependent(t *testing.T) {
	scopeID := uuid.New()
	authorizer := MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}
	fields := MemoryPartyFieldAuthorizer{Decision: PartyFieldAuthorization{Identity: true, PersonalData: true, Classifications: true, BankDetailReferences: true}}
	service, err := NewPartyService(NewMemoryPartyRepository(), authorizer, fields, AllowAllPartyBankReferenceValidator{}, AllowAllPartyBankControlEvaluator{}, &MemoryPartyAuditRecorder{}, partyTestClock)
	if err != nil {
		t.Fatal(err)
	}
	command := partyTestCommand(scopeID, PartyActionCreate)
	command.TaxIdentifier = nil
	result, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Party.Name != "Example Vendor" || result.Party.TaxIdentifierMasked != "" {
		t.Fatalf("field projection = %#v, want identity visible and tax omitted", result.Party)
	}

	command = partyTestCommand(scopeID, PartyActionCreate)
	command.TaxIdentifier = stringPointer("TAX-PRIVATE-1234")
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor-2"}, command); !errors.Is(err, ErrPartyFieldAuthorizationDenied) {
		t.Fatalf("restricted create error = %v, want field authorization denial", err)
	}
}

func TestPartyServiceBankControlFailsClosedAndDoesNotMutate(t *testing.T) {
	scopeID := uuid.New()
	repository := NewMemoryPartyRepository()
	control := MemoryPartyBankControlEvaluator{State: BankDetailControlState{Status: PartyBankControlPending}}
	service, err := NewPartyService(
		repository,
		MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		AllowAllPartyFieldAuthorizer{}, AllowAllPartyBankReferenceValidator{}, &control, &MemoryPartyAuditRecorder{}, partyTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	create := partyTestCommand(scopeID, PartyActionCreate)
	create.BankDetailReferences = nil
	created, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, create)
	if err != nil {
		t.Fatal(err)
	}
	version := aggregateversion.AggregateVersion(1)
	maintain := partyTestCommand(scopeID, PartyActionMaintain)
	maintain.PartyID = created.Party.ID
	maintain.ExpectedVersion = &version
	maintain.IdempotencyKey = "maintain-bank-pending"
	maintain.BankDetailReferences = []PartyBankDetailReference{{Reference: "bank-ref-001", ProviderCode: "provider-a", ConsentReference: "consent-ref-001"}}
	result, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, maintain)
	if err != nil {
		t.Fatal(err)
	}
	if result.BankControl.Status != PartyBankControlPending || result.Party.Version.Value() != 2 {
		t.Fatalf("pending bank result = %#v", result)
	}

	control.State = BankDetailControlState{Status: PartyBankControlRejected}
	maintain.ExpectedVersion = aggregateVersionPointer(2)
	maintain.IdempotencyKey = "maintain-bank-rejected"
	maintain.BankDetailReferences = []PartyBankDetailReference{{Reference: "bank-ref-002", ProviderCode: "provider-a"}}
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, maintain); !errors.Is(err, ErrPartyBankApprovalRejected) {
		t.Fatalf("rejected bank error = %v", err)
	}
	current, err := repository.Get(context.Background(), created.Party.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Version.Value() != 2 || current.BankDetailReferences[0].Reference != "bank-ref-001" {
		t.Fatalf("rejected bank mutated party: %#v", current)
	}
}

func TestPartyServiceBankControlOutcomes(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		wantStatus string
		wantErr    error
	}{
		{name: "approved", status: PartyBankControlApproved, wantStatus: PartyBankControlApproved},
		{name: "pending", status: PartyBankControlPending, wantStatus: PartyBankControlPending},
		{name: "rejected", status: PartyBankControlRejected, wantErr: ErrPartyBankApprovalRejected},
		{name: "stale", status: PartyBankControlStale, wantErr: ErrPartyBankApprovalStale},
		{name: "unavailable", status: PartyBankControlUnavailable, wantErr: ErrPartyBankControlUnavailable},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			scopeID := uuid.New()
			repository := NewMemoryPartyRepository()
			service, err := NewPartyService(
				repository,
				MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
				AllowAllPartyFieldAuthorizer{}, AllowAllPartyBankReferenceValidator{},
				MemoryPartyBankControlEvaluator{State: BankDetailControlState{Status: testCase.status}},
				&MemoryPartyAuditRecorder{}, partyTestClock,
			)
			if err != nil {
				t.Fatal(err)
			}
			create := partyTestCommand(scopeID, PartyActionCreate)
			create.BankDetailReferences = nil
			created, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, create)
			if err != nil {
				t.Fatal(err)
			}

			maintain := partyTestCommand(scopeID, PartyActionMaintain)
			maintain.PartyID = created.Party.ID
			maintain.ExpectedVersion = aggregateVersionPointer(1)
			maintain.IdempotencyKey = "bank-control-" + testCase.name
			maintain.BankDetailReferences = []PartyBankDetailReference{{Reference: "bank-ref-" + testCase.name, ProviderCode: "provider-a"}}
			result, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, maintain)
			if testCase.wantErr != nil {
				if !errors.Is(err, testCase.wantErr) {
					t.Fatalf("error = %v, want %v", err, testCase.wantErr)
				}
				current, getErr := repository.Get(context.Background(), created.Party.ID)
				if getErr != nil {
					t.Fatal(getErr)
				}
				if current.Version.Value() != 1 || len(current.BankDetailReferences) != 0 {
					t.Fatalf("failed bank control changed party = %#v", current)
				}
				return
			}
			if err != nil || result.BankControl.Status != testCase.wantStatus || result.Party.Version.Value() != 2 {
				t.Fatalf("result = %#v, err = %v", result, err)
			}
		})
	}
}

func TestPartyServiceAuditFailureLeavesMemoryRepositoryUnchanged(t *testing.T) {
	scopeID := uuid.New()
	repository := NewMemoryPartyRepository()
	audit := &MemoryPartyAuditRecorder{Err: ErrPartyAuditUnavailable}
	service, err := NewPartyService(
		repository,
		MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		AllowAllPartyFieldAuthorizer{}, AllowAllPartyBankReferenceValidator{}, AllowAllPartyBankControlEvaluator{}, audit, partyTestClock,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, partyTestCommand(scopeID, PartyActionCreate)); !errors.Is(err, ErrPartyAuditUnavailable) {
		t.Fatalf("audit error = %v, want audit unavailable", err)
	}
	parties, err := repository.List(context.Background(), &scopeID)
	if err != nil {
		t.Fatal(err)
	}
	if len(parties) != 0 {
		t.Fatalf("parties after audit failure = %d, want zero", len(parties))
	}
}

func TestPartyServiceRejectsRawBankReferences(t *testing.T) {
	scopeID := uuid.New()
	service, err := NewPartyService(NewMemoryPartyRepository(), MemoryPartyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllPartyFieldAuthorizer{}, AllowAllPartyBankReferenceValidator{}, AllowAllPartyBankControlEvaluator{}, &MemoryPartyAuditRecorder{}, partyTestClock)
	if err != nil {
		t.Fatal(err)
	}
	command := partyTestCommand(scopeID, PartyActionCreate)
	command.TaxIdentifier = nil
	for _, reference := range []string{"123456789", "1234-5678"} {
		command.BankDetailReferences = []PartyBankDetailReference{{Reference: reference, ProviderCode: "provider-a"}}
		command.IdempotencyKey = "raw-bank-" + reference
		if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, command); !errors.Is(err, ErrPartyBankReferenceInvalid) {
			t.Fatalf("raw bank reference %q error = %v, want invalid reference", reference, err)
		}
	}
}

func partyTestCommand(scopeID uuid.UUID, action string) PartyCommand {
	command := PartyCommand{Action: action, ScopeID: scopeID, Name: "Example Vendor", PartyType: "vendor", Status: "active", TaxIdentifier: stringPointer("TAX-PRIVATE-1234"), ContactMethods: []PartyContactMethod{{Type: "email", Value: "finance@example.test"}}, Addresses: []PartyAddress{{Type: "registered", Line1: "1 Main Street", Locality: "Hanoi", PostalCode: "100000", CountryCode: "VN"}}, Classifications: []PartyClassification{{Code: "segment", Value: "supplier"}}, IdempotencyKey: action + "-1", CorrelationID: "correlation", CausationID: "causation"}
	return command
}

func partyTestClock() time.Time {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
}

func stringPointer(value string) *string { return &value }

func aggregateVersionPointer(value int64) *aggregateversion.AggregateVersion {
	version := aggregateversion.AggregateVersion(value)
	return &version
}
