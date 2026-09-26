package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestPolicyEvaluatorRequiresEveryConstrainedDimension(t *testing.T) {
	actor := uuid.New()
	values := struct {
		accountingScope uuid.UUID
		legalEntity     uuid.UUID
		segment         uuid.UUID
		account         uuid.UUID
		fiscalPeriod    uuid.UUID
	}{uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()}
	cases := []struct {
		name      string
		rule      AccessRule
		apply     func(*DecisionInput)
		dimension string
	}{
		{"scope", AccessRule{ScopeIDs: []string{"scope-a"}}, func(input *DecisionInput) { input.RequestedScopeIDs = []string{"scope-a"} }, AuthorizationDimensionScope},
		{"accounting scope", AccessRule{AccountingScopeIDs: []uuid.UUID{values.accountingScope}}, func(input *DecisionInput) { input.AccountingScopeID = &values.accountingScope }, AuthorizationDimensionAccountingScope},
		{"legal entity", AccessRule{LegalEntityIDs: []uuid.UUID{values.legalEntity}}, func(input *DecisionInput) { input.LegalEntityID = &values.legalEntity }, AuthorizationDimensionLegalEntity},
		{"segment", AccessRule{SegmentIDs: []uuid.UUID{values.segment}}, func(input *DecisionInput) { input.SegmentIDs = []uuid.UUID{values.segment} }, AuthorizationDimensionSegment},
		{"account", AccessRule{AccountIDs: []uuid.UUID{values.account}}, func(input *DecisionInput) { input.AccountID = &values.account }, AuthorizationDimensionAccount},
		{"account class", AccessRule{AccountClasses: []string{"asset"}}, func(input *DecisionInput) { value := "asset"; input.AccountClass = &value }, AuthorizationDimensionAccountClass},
		{"transaction type", AccessRule{TransactionTypes: []string{"journal"}}, func(input *DecisionInput) { value := "journal"; input.TransactionType = &value }, AuthorizationDimensionTransactionType},
		{"amount", AccessRule{MinAmount: decimalPointer("10.00"), MaxAmount: decimalPointer("20.00")}, func(input *DecisionInput) { input.Amount = decimalPointer("10.00") }, AuthorizationDimensionAmount},
		{"currency", AccessRule{Currencies: []string{"USD"}}, func(input *DecisionInput) { value := "USD"; input.Currency = &value }, AuthorizationDimensionCurrency},
		{"fiscal period", AccessRule{FiscalPeriodIDs: []uuid.UUID{values.fiscalPeriod}}, func(input *DecisionInput) { input.FiscalPeriodID = &values.fiscalPeriod }, AuthorizationDimensionFiscalPeriod},
		{"data classification", AccessRule{DataClassifications: []string{"restricted"}}, func(input *DecisionInput) { input.DataClassification = "restricted" }, AuthorizationDimensionDataClassification},
		{"subject actor", AccessRule{SubjectActorIDs: []uuid.UUID{actor}}, func(input *DecisionInput) { input.SubjectActorIDs = []uuid.UUID{actor} }, AuthorizationDimensionSubjectActor},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			evaluator := newTestEvaluator(t, AccessPolicy{ID: uuid.New(), Version: "v1", Status: AccessPolicyStatusActive, Permissions: []string{"finance.test"}, EffectiveFrom: time.Now().Add(-time.Hour), Rules: []AccessRule{testCase.rule}})
			input := DecisionInput{ActorID: actor, Permission: "finance.test"}
			testCase.apply(&input)
			decision, err := evaluator.Evaluate(context.Background(), input)
			if err != nil || decision.Outcome != AuthorizationAllowed {
				t.Fatalf("allowed evaluation = %#v, err=%v", decision, err)
			}
			input = DecisionInput{ActorID: actor, Permission: "finance.test"}
			decision, err = evaluator.Evaluate(context.Background(), input)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Outcome != AuthorizationDenied {
				t.Fatalf("missing %s outcome = %s, want denied", testCase.dimension, decision.Outcome)
			}
		})
	}
}

func TestPolicyEvaluatorAmountBoundariesAndDefaultDeny(t *testing.T) {
	evaluator := newTestEvaluator(t, AccessPolicy{
		ID: uuid.New(), Version: "v1", Status: AccessPolicyStatusActive, Permissions: []string{"finance.test"}, EffectiveFrom: time.Now().Add(-time.Hour),
		Rules: []AccessRule{{MinAmount: decimalPointer("10.00"), MaxAmount: decimalPointer("20.00")}},
	})
	for _, value := range []string{"10.00", "20.00"} {
		decision, err := evaluator.Evaluate(context.Background(), DecisionInput{ActorID: uuid.New(), Permission: "finance.test", Amount: decimalPointer(value)})
		if err != nil || decision.Outcome != AuthorizationAllowed {
			t.Fatalf("amount %s decision = %#v, err=%v", value, decision, err)
		}
	}
	for _, value := range []string{"9.999", "20.001"} {
		decision, err := evaluator.Evaluate(context.Background(), DecisionInput{ActorID: uuid.New(), Permission: "finance.test", Amount: decimalPointer(value)})
		if err != nil || decision.Outcome != AuthorizationDenied {
			t.Fatalf("amount %s decision = %#v, err=%v", value, decision, err)
		}
	}
	decision, err := evaluator.Evaluate(context.Background(), DecisionInput{ActorID: uuid.New(), Permission: "finance.unknown"})
	if err != nil || decision.Outcome != AuthorizationDenied || decision.DecisionReference == uuid.Nil {
		t.Fatalf("default deny decision = %#v, err=%v", decision, err)
	}
}

func TestPolicyEvaluatorReportsExpiredStaleAndUnavailableSafely(t *testing.T) {
	actor := uuid.New()
	expired := time.Now().Add(-time.Hour)
	expiredTo := time.Now().Add(-time.Minute)
	evaluator := newTestEvaluator(t, AccessPolicy{ID: uuid.New(), Version: "v1", Status: AccessPolicyStatusActive, Permissions: []string{"finance.test"}, EffectiveFrom: expired, EffectiveTo: &expiredTo, Rules: []AccessRule{{}}})
	decision, err := evaluator.Evaluate(context.Background(), DecisionInput{ActorID: actor, Permission: "finance.test"})
	if err != nil || decision.Outcome != AuthorizationExpired || decision.ReasonCode != AuthorizationReasonPolicyExpired {
		t.Fatalf("expired decision = %#v, err=%v", decision, err)
	}

	staleStore, err := NewMemoryAccessPolicyStore(AccessPolicy{ID: uuid.New(), Version: "v2", Status: AccessPolicyStatusActive, Permissions: []string{"finance.test"}, EffectiveFrom: expired, Rules: []AccessRule{{}}})
	if err != nil {
		t.Fatal(err)
	}
	staleEvaluator, err := NewPolicyEvaluator(staleStore, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	decision, err = staleEvaluator.Evaluate(context.Background(), DecisionInput{ActorID: actor, Permission: "finance.test", ExpectedPolicyVersion: "v1"})
	if err != nil || decision.Outcome != AuthorizationStale || decision.PolicyReference == "" || decision.PolicyVersion != "v2" {
		t.Fatalf("stale decision = %#v, err=%v", decision, err)
	}

	unavailableStore, err := NewMemoryAccessPolicyStore()
	if err != nil {
		t.Fatal(err)
	}
	unavailableStore.SetError(errors.New("database details must not escape"))
	unavailableEvaluator, err := NewPolicyEvaluator(unavailableStore, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	decision, err = unavailableEvaluator.Evaluate(context.Background(), DecisionInput{ActorID: actor, Permission: "finance.test"})
	if err != nil || decision.Outcome != AuthorizationUnavailable || decision.ReasonCode != AuthorizationReasonPolicyUnavailable {
		t.Fatalf("unavailable decision = %#v, err=%v", decision, err)
	}
	if decision.Reason == "database details must not escape" {
		t.Fatal("raw policy dependency error escaped decision")
	}
}

func TestPolicyEvaluatorFieldAccessIsIndependentFromRecordAccess(t *testing.T) {
	evaluator := newTestEvaluator(t, AccessPolicy{ID: uuid.New(), Version: "v1", Status: AccessPolicyStatusActive, Permissions: []string{"record.read"}, EffectiveFrom: time.Now().Add(-time.Hour), Rules: []AccessRule{{}}})
	record, err := evaluator.Evaluate(context.Background(), DecisionInput{ActorID: uuid.New(), Permission: "record.read"})
	if err != nil || record.Outcome != AuthorizationAllowed {
		t.Fatalf("record decision = %#v, err=%v", record, err)
	}
	field, err := evaluator.EvaluateField(context.Background(), FieldAccessInput{DecisionInput: DecisionInput{ActorID: uuid.New()}, FieldClassification: "restricted", FieldAction: "reveal"})
	if err != nil || field.Reveal || field.Export || field.Outcome != AuthorizationDenied {
		t.Fatalf("field decision = %#v, err=%v", field, err)
	}
}

func TestMemoryAccessPolicyStoreKeepsRevisionsImmutable(t *testing.T) {
	policy := AccessPolicy{ID: uuid.New(), Version: "v1", Status: AccessPolicyStatusActive, Permissions: []string{"finance.test"}, EffectiveFrom: time.Now().Add(-time.Hour), Rules: []AccessRule{{}}}
	store, err := NewMemoryAccessPolicyStore(policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Add(policy); !errors.Is(err, ErrAccessPolicyRevision) {
		t.Fatalf("duplicate revision error = %v, want %v", err, ErrAccessPolicyRevision)
	}
	listed, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	listed[0].Permissions[0] = "mutated"
	again, err := store.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if again[0].Permissions[0] != "finance.test" {
		t.Fatal("policy store returned mutable internal revision")
	}
}

func newTestEvaluator(t *testing.T, policy AccessPolicy) *PolicyEvaluator {
	t.Helper()
	store, err := NewMemoryAccessPolicyStore(policy)
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := NewPolicyEvaluator(store, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return evaluator
}

func decimalPointer(value string) *decimal.Decimal {
	parsed := decimal.RequireFromString(value)
	return &parsed
}
