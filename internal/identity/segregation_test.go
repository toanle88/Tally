package identity

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestSegregationEvaluatorEnforcesMinimumRules(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	repository, err := NewMemorySegregationRuleRepository(DefaultSegregationRules(now)...)
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := NewSegregationEvaluator(repository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	subject := uuid.New()
	amount := decimal.NewFromInt(10001)
	tests := []struct {
		name    string
		input   SegregationDecisionInput
		allowed bool
		outcome AuthorizationOutcome
	}{
		{name: "payment batch", input: SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchApprove, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchPrepare}}}, allowed: false, outcome: AuthorizationDenied},
		{name: "fiscal reopen", input: SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionFiscalReopenApprove, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionFiscalReopenRequest}}}, allowed: false, outcome: AuthorizationDenied},
		{name: "vendor cooling off", input: SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentRelease, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionVendorBankDetailChange, At: now.Add(-time.Hour)}}}, allowed: false, outcome: AuthorizationDenied},
		{name: "manual journal above threshold", input: SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionManualJournalApprove, Amount: &amount, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionManualJournalPrepare}}}, allowed: false, outcome: AuthorizationDenied},
		{name: "payroll and summary ledger", input: SegregationDecisionInput{ActorID: actor, Action: "access", Permissions: []string{"finance.payr.maintain.employee.payroll.profiles", "finance.rpt.generate.and.publish.ledger.financial.statements"}}, allowed: false, outcome: AuthorizationDenied},
		{name: "policy self approval", input: SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPolicyApprove, ProposerID: actor, ApproverID: actor, History: []SegregationHistoryEntry{}}, allowed: false, outcome: AuthorizationDenied},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			decision, err := evaluator.Evaluate(context.Background(), test.input)
			if err != nil {
				t.Fatal(err)
			}
			if decision.Allowed != test.allowed || decision.Outcome != test.outcome {
				t.Fatalf("decision = %#v, want allowed=%v outcome=%q", decision, test.allowed, test.outcome)
			}
		})
	}
}

func TestSegregationEvaluatorAllowsControlsAndApprovedException(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	rules := DefaultSegregationRules(now)
	repository, err := NewMemorySegregationRuleRepository(rules...)
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := NewSegregationEvaluator(repository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	subject := uuid.New()
	decision, err := evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchApprove, History: []SegregationHistoryEntry{{ActorID: uuid.New(), SubjectID: subject, Action: SegregationActionPaymentBatchPrepare}}})
	if err != nil || !decision.Allowed {
		t.Fatalf("different actor control = %#v, err=%v", decision, err)
	}
	exception := &SegregationException{Active: true, Reason: "incident bridge", ApprovedBy: uuid.New(), ExpiresAt: now.Add(time.Hour), RuleVersion: rules[0].Version}
	decision, err = evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchApprove, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchPrepare}}, Exception: exception})
	if err != nil || !decision.Allowed || decision.ReasonCode != SegregationReasonAllowed {
		t.Fatalf("approved exception = %#v, err=%v", decision, err)
	}
	exception.ExpiresAt = now.Add(-time.Minute)
	decision, err = evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchApprove, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchPrepare}}, Exception: exception})
	if err != nil || decision.Allowed || decision.ReasonCode != SegregationReasonExpiredException {
		t.Fatalf("expired exception = %#v, err=%v", decision, err)
	}
	decision, err = evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentRelease, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionVendorBankDetailChange, At: now.Add(-25 * time.Hour)}}})
	if err != nil || !decision.Allowed {
		t.Fatalf("cooled-off vendor change = %#v, err=%v", decision, err)
	}
	decision, err = evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentRelease, ScopeIDs: []string{"entity-2"}, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, ScopeIDs: []string{"entity-1"}, Action: SegregationActionVendorBankDetailChange, At: now.Add(-time.Hour)}}})
	if err != nil || !decision.Allowed {
		t.Fatalf("out-of-scope vendor change = %#v, err=%v", decision, err)
	}
	underThreshold := decimal.NewFromInt(10000)
	decision, err = evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionManualJournalApprove, Amount: &underThreshold, History: []SegregationHistoryEntry{{ActorID: actor, SubjectID: subject, Action: SegregationActionManualJournalPrepare}}})
	if err != nil || !decision.Allowed {
		t.Fatalf("threshold control = %#v, err=%v", decision, err)
	}
}

func TestSegregationEvaluatorFailsClosedForStaleMissingAndUnavailableState(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	repository, err := NewMemorySegregationRuleRepository(DefaultSegregationRules(now)...)
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := NewSegregationEvaluator(repository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := uuid.New()
	subject := uuid.New()
	version := versionForTest(2)
	decision, err := evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchApprove, ExpectedRuleVersion: version, History: []SegregationHistoryEntry{}})
	if err != nil || decision.Allowed || decision.Outcome != AuthorizationStale {
		t.Fatalf("stale decision = %#v, err=%v", decision, err)
	}
	decision, err = evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, SubjectID: subject, Action: SegregationActionPaymentBatchApprove})
	if err != nil || decision.Allowed || decision.Outcome != AuthorizationUnavailable {
		t.Fatalf("missing history decision = %#v, err=%v", decision, err)
	}
	unavailable, err := NewSegregationEvaluator(failingSegregationRuleStore{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	decision, err = unavailable.Evaluate(context.Background(), SegregationDecisionInput{ActorID: actor, Action: "access"})
	if err != nil || decision.Allowed || decision.Outcome != AuthorizationUnavailable {
		t.Fatalf("unavailable decision = %#v, err=%v", decision, err)
	}
}

func TestSegregationEvaluatorReportsOnlyBoundedOutcome(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	repository, err := NewMemorySegregationRuleRepository(DefaultSegregationRules(now)...)
	if err != nil {
		t.Fatal(err)
	}
	var outcomes []AuthorizationOutcome
	evaluator, err := NewSegregationEvaluator(repository, func() time.Time { return now }, func(_ context.Context, decision SegregationDecision, err error) {
		if err != nil {
			t.Errorf("observer error = %v", err)
		}
		outcomes = append(outcomes, decision.Outcome)
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := evaluator.Evaluate(context.Background(), SegregationDecisionInput{ActorID: uuid.New(), Action: "access"}); err != nil {
		t.Fatal(err)
	}
	if len(outcomes) != 1 || outcomes[0] != AuthorizationAllowed {
		t.Fatalf("outcomes = %#v, want one allowed outcome", outcomes)
	}
}

type failingSegregationRuleStore struct{}

func (failingSegregationRuleStore) List(context.Context) ([]SegregationRule, error) {
	return nil, errors.New("database unavailable")
}

func TestSegregationRuleServiceProvidesImmutableAuditLinkedReplay(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	repository, err := NewMemorySegregationRuleRepository(DefaultSegregationRules(now)...)
	if err != nil {
		t.Fatal(err)
	}
	audit := &MemorySegregationRuleAuditRecorder{}
	authorizer := MemorySegregationRuleAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: SegregationRuleManagementPermission, PolicyReference: "iam-policy", PolicyVersion: "iam-policy-v1", DecisionReference: uuid.New()}}
	service, err := NewSegregationRuleService(repository, authorizer, testSegregationRuleApproval{}, audit, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor(t)
	command := SegregationRuleCommand{Action: SegregationRuleActionCreate, Code: "test-rule", Name: "Test rule", ConflictingPermissions: []string{"finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"}, EnforcementMode: SegregationEnforcementBlock, EffectiveFrom: now, Approval: ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "rule-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:test", ApproverUserID: uuid.New()}, IdempotencyKey: "rule-1", CorrelationID: "corr-1", CausationID: "cause-1"}
	created, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil || !replay.Replayed || replay.Rule.ID != created.Rule.ID {
		t.Fatalf("replay = %#v, err=%v", replay, err)
	}
	command.Name = "different"
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrSegregationRuleIdempotency) {
		t.Fatalf("changed replay error = %v", err)
	}
	if len(audit.Records) != 1 || audit.Records[0].RuleID != created.Rule.ID || audit.Records[0].AfterFingerprint == "" {
		t.Fatalf("audit records = %#v", audit.Records)
	}
}

func TestAllowAllSegregationRuleApprovalBindsCreateFingerprintAndVersion(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	actor := testActor(t)
	approval := ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "rule-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:preview", ApproverUserID: uuid.New()}
	command := SegregationRuleCommand{Action: SegregationRuleActionCreate, Code: "approval-bound-rule", Name: "Approval-bound rule", ConflictingPermissions: []string{"finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"}, EnforcementMode: SegregationEnforcementBlock, EffectiveFrom: now, Approval: approval, IdempotencyKey: "approval-bound-rule-create"}
	preview, err := NewSegregationRule(uuid.New(), command.Code, command.Name, command.ConflictingPermissions, command.EnforcementMode, nil, nil, 0, command.EffectiveFrom, command.Approval, now)
	if err != nil {
		t.Fatal(err)
	}
	command.Approval.CandidateFingerprint = candidateSegregationRuleFingerprint(command, preview)
	repository, err := NewMemorySegregationRuleRepository()
	if err != nil {
		t.Fatal(err)
	}
	audit := &MemorySegregationRuleAuditRecorder{}
	service, err := NewSegregationRuleService(repository, MemorySegregationRuleAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: SegregationRuleManagementPermission, DecisionReference: uuid.New()}}, AllowAllSegregationRuleApprovalPort{}, audit, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Rule.Version.Value() != 1 || len(audit.Records) != 1 {
		t.Fatalf("result = %#v, audit records = %#v", result, audit.Records)
	}

	command.IdempotencyKey = "approval-bound-rule-wrong-version"
	command.Approval.SubjectVersion = 2
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrSegregationRuleApproval) {
		t.Fatalf("wrong subject version error = %v", err)
	}
}

func TestSegregationDurableFailureOutcomesRoundTrip(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want error
	}{
		{name: "expired authorization", err: ErrAuthorizationExpired, want: ErrAuthorizationExpired},
		{name: "unavailable authorization", err: ErrAuthorizationUnavailable, want: ErrSegregationRuleUnavailable},
		{name: "idempotency conflict", err: ErrSegregationRuleIdempotency, want: ErrSegregationRuleIdempotency},
	} {
		t.Run(test.name, func(t *testing.T) {
			code, ok := segregationDurableFailureCode(test.err)
			if !ok {
				t.Fatalf("failure code was not established for %v", test.err)
			}
			body, err := json.Marshal(durableFailurePayload{Code: code})
			if err != nil {
				t.Fatal(err)
			}
			if roundTrip := segregationDurableFailureError(body); !errors.Is(roundTrip, test.want) {
				t.Fatalf("round-trip error = %v, want %v", roundTrip, test.want)
			}
		})
	}
}

type testSegregationRuleApproval struct{}

func (testSegregationRuleApproval) ValidateSegregationRuleApproval(context.Context, ApplicationActor, SegregationRuleCommand, *SegregationRule, string) error {
	return nil
}

func TestRuleBackedRoleSegregationRejectsBeforePersistence(t *testing.T) {
	now := time.Date(2026, 9, 26, 12, 0, 0, 0, time.UTC)
	rules, err := NewMemorySegregationRuleRepository(DefaultSegregationRules(now)...)
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := NewSegregationEvaluator(rules, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	roles := NewMemoryRoleRepository()
	authorizer := MemoryRoleAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: RoleManagementPermission, ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()}}
	service, err := NewRoleService(roles, authorizer, AllowAllRoleApprovalPort{}, evaluator, &MemoryRoleAuditRecorder{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	grants := []PermissionGrant{{Permission: "finance.pcm.prepare.payment.batch", ScopeIDs: []string{"entity-1"}, EffectiveFrom: now}, {Permission: "finance.pcm.apply.payment.batch.approval.decision", ScopeIDs: []string{"entity-1"}, EffectiveFrom: now}}
	if _, err := service.Execute(context.Background(), testActor(t), roleCreateCommand("conflict", "Conflicting role", grants, testActor(t))); !errors.Is(err, ErrSegregationConflict) {
		t.Fatalf("role conflict error = %v", err)
	}
	stored, err := roles.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(stored) != 0 {
		t.Fatalf("stored roles after conflict = %d, want 0", len(stored))
	}
}
