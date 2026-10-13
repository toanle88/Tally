package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestApprovalPolicyDomainValidatesExactThresholdsAndScope(t *testing.T) {
	minimum := decimal.RequireFromString("0.10")
	maximum := decimal.RequireFromString("100.20")
	ruleID, stepID := uuid.New(), uuid.New()
	policy, err := NewApprovalPolicy(uuid.New(), "gl-posting", "GL posting approvals", "v1", ApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []string{"USD"}}, ApprovalPolicyScope{AccountingScopeIDs: []uuid.UUID{uuid.New()}}, []ApprovalRule{{ID: ruleID, Threshold: AmountThreshold{Minimum: &minimum, Maximum: &maximum, Currency: "USD"}, RequiredRoles: []RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}}}, []ApprovalStepDefinition{{ID: stepID, Sequence: 1, Name: "Independent approval", RequiredRoles: []RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1}}, time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), nil, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got := policy.CandidateFingerprint(); got == "" || got[:7] != "sha256:" {
		t.Fatalf("candidate fingerprint = %q", got)
	}
	if !policy.Scope.Contains([]string{policy.Scope.AccountingScopeIDs[0].String()}) {
		t.Fatal("typed accounting scope should contain its canonical scope identity")
	}

	bad := policy
	bad.Rules = cloneRules(policy.Rules)
	bad.Rules[0].Threshold.Minimum = ptrDecimal(decimal.RequireFromString("100.21"))
	if !errors.Is(bad.Validate(), ErrInvalidApprovalPolicy) {
		t.Fatalf("invalid threshold error = %v", bad.Validate())
	}
	badApplicability := policy
	badApplicability.Applicability.SourceContext = " general-ledger"
	if !errors.Is(badApplicability.Validate(), ErrInvalidApprovalPolicy) {
		t.Fatalf("malformed applicability error = %v", badApplicability.Validate())
	}
	badScope := policy
	badScope.Scope = ApprovalPolicyScope{ScopeIDs: []string{"*", "narrow-scope"}}
	if !errors.Is(badScope.Validate(), ErrInvalidApprovalPolicy) {
		t.Fatalf("wildcard scope error = %v", badScope.Validate())
	}
	badScope.Scope = ApprovalPolicyScope{ScopeIDs: []string{"*"}, AccountingScopeIDs: []uuid.UUID{uuid.New()}}
	if !errors.Is(badScope.Validate(), ErrInvalidApprovalPolicy) {
		t.Fatalf("wildcard typed scope error = %v", badScope.Validate())
	}
}

func TestApprovalPolicyLifecycleUsesGenericApprovalEvidenceAndPreservesHistory(t *testing.T) {
	requester := Actor{UserID: uuid.New(), SubjectReference: "requester-subject"}
	approver := Actor{UserID: uuid.New(), SubjectReference: "approver-subject"}
	authorizer := MemoryApprovalPolicyAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: MaintainApprovalPoliciesPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}
	repository := NewMemoryApprovalPolicyRepository()
	approvalWorkflow, err := NewMemoryApprovalWorkflow()
	if err != nil {
		t.Fatal(err)
	}
	audit := &MemoryApprovalPolicyAuditRecorder{}
	service, err := NewApprovalPolicyService(repository, authorizer, approvalWorkflow, audit, func() time.Time { return time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}
	command := policyCommand("create-policy", ApprovalPolicyActionCreate, uuid.Nil, nil)
	command.SubmitForApproval = true
	created, err := service.Execute(context.Background(), requester, command)
	if err != nil {
		t.Fatal(err)
	}
	if created.Policy.Status != ApprovalPolicyStatusPendingApproval || created.ApprovalRequest == nil || created.Policy.ApprovalRequestID != created.ApprovalRequest.ID {
		t.Fatalf("pending result = %#v request = %#v", created.Policy, created.ApprovalRequest)
	}

	expectedRequestVersion := created.ApprovalRequest.Version
	decision, err := approvalWorkflow.Service.DecideApprovalRequest(context.Background(), approver, DecideApprovalRequestCommand{RequestID: created.ApprovalRequest.ID, ExpectedVersion: &expectedRequestVersion, DecisionID: uuid.New(), Outcome: ApprovalDecisionApproved, Reason: "Independent review complete", PolicyVersion: created.ApprovalRequest.PolicyVersion, SubjectVersion: created.ApprovalRequest.SubjectVersion, CandidateFingerprint: created.ApprovalRequest.SubjectFingerprint, IdempotencyKey: "decide-policy"})
	if err != nil {
		t.Fatal(err)
	}
	if decision.Reference.ApprovalRequestID != created.Policy.ApprovalRequestID || decision.Request.Status != ApprovalRequestStatusApproved {
		t.Fatalf("decision evidence = %#v", decision)
	}

	activateVersion := created.Policy.Version
	activated, err := service.Execute(context.Background(), requester, ApprovalPolicyCommand{Action: ApprovalPolicyActionActivate, PolicyID: created.Policy.ID, ExpectedVersion: &activateVersion, Approval: &decision.Reference, Reason: "Apply approved version", IdempotencyKey: "activate-policy"})
	if err != nil {
		t.Fatal(err)
	}
	if activated.Policy.Status != ApprovalPolicyStatusActive || activated.DecisionReference != decision.Reference.DecisionID || activated.AuditReference == uuid.Nil {
		t.Fatalf("activated result = %#v", activated)
	}

	replayed, err := service.Execute(context.Background(), requester, command)
	if err != nil || !replayed.Replayed || replayed.Policy.Status != ApprovalPolicyStatusPendingApproval {
		t.Fatalf("replayed create = %#v, err = %v", replayed, err)
	}
	headerReplay := command
	headerReplay.CorrelationID = uuid.NewString()
	headerReplay.CausationID = uuid.NewString()
	replayed, err = service.Execute(context.Background(), requester, headerReplay)
	if err != nil || !replayed.Replayed || replayed.Policy.Status != ApprovalPolicyStatusPendingApproval {
		t.Fatalf("trace-header replay = %#v, err = %v", replayed, err)
	}
	changed := command
	changed.Name = "changed content"
	if !errors.Is(func() error { _, err := service.Execute(context.Background(), requester, changed); return err }(), ErrApprovalPolicyIdempotencyConflict) {
		t.Fatal("changed idempotent content should be rejected")
	}

	updateVersion := activated.Policy.Version
	updated, err := service.Execute(context.Background(), requester, ApprovalPolicyCommand{Action: ApprovalPolicyActionUpdate, PolicyID: activated.Policy.ID, PolicyVersion: "v2", ExpectedVersion: &updateVersion, SubmitForApproval: true, IdempotencyKey: "update-policy"})
	if err != nil {
		t.Fatal(err)
	}
	selectable, err := repository.ListSelectable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(selectable) != 1 || selectable[0].PolicyVersion != activated.Policy.PolicyVersion {
		t.Fatalf("selectable history = %#v", selectable)
	}
	historicalRequest, err := approvalWorkflow.Requests.Get(context.Background(), created.Policy.ApprovalRequestID)
	if err != nil || historicalRequest.PolicyVersion != activated.Policy.PolicyVersion {
		t.Fatalf("historical request = %#v, err = %v", historicalRequest, err)
	}
	if updated.Policy.ApprovalRequestID == created.Policy.ApprovalRequestID {
		t.Fatal("new candidate must have a new approval request identity")
	}
	sameVersion := updated.Policy.Version
	if _, err := service.Execute(context.Background(), requester, ApprovalPolicyCommand{Action: ApprovalPolicyActionUpdate, PolicyID: updated.Policy.ID, PolicyVersion: updated.Policy.PolicyVersion, ExpectedVersion: &sameVersion, IdempotencyKey: "same-policy-version"}); !errors.Is(err, ErrApprovalPolicyVersionConflict) {
		t.Fatalf("reusing a policy version = %v, want version conflict", err)
	}
	decisionVersion := updated.ApprovalRequest.Version
	updatedDecision, err := approvalWorkflow.Service.DecideApprovalRequest(context.Background(), approver, DecideApprovalRequestCommand{RequestID: updated.ApprovalRequest.ID, ExpectedVersion: &decisionVersion, DecisionID: uuid.New(), Outcome: ApprovalDecisionApproved, Reason: "Independent review complete", PolicyVersion: updated.ApprovalRequest.PolicyVersion, SubjectVersion: updated.ApprovalRequest.SubjectVersion, CandidateFingerprint: updated.ApprovalRequest.SubjectFingerprint, IdempotencyKey: "decide-policy-v2"})
	if err != nil {
		t.Fatal(err)
	}
	activeVersion := updated.Policy.Version
	activatedV2, err := service.Execute(context.Background(), requester, ApprovalPolicyCommand{Action: ApprovalPolicyActionActivate, PolicyID: updated.Policy.ID, ExpectedVersion: &activeVersion, Approval: &updatedDecision.Reference, Reason: "Apply approved version", IdempotencyKey: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	if activatedV2.Policy.Status != ApprovalPolicyStatusActive {
		t.Fatalf("second policy activation = %#v", activatedV2.Policy)
	}
	selectable, err = repository.ListSelectable(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(selectable) != 1 || selectable[0].PolicyVersion != updated.Policy.PolicyVersion {
		t.Fatalf("selectable replacement history = %#v", selectable)
	}
	historicalVersion := activated.Policy.Version
	if _, err := service.Execute(context.Background(), requester, ApprovalPolicyCommand{Action: ApprovalPolicyActionUpdate, PolicyID: activatedV2.Policy.ID, PolicyVersion: activated.Policy.PolicyVersion, ExpectedVersion: &historicalVersion, IdempotencyKey: "reuse-historical-policy-version"}); !errors.Is(err, ErrApprovalPolicyVersionConflict) {
		t.Fatalf("reusing a historical policy version = %v, want version conflict", err)
	}
}

func TestApprovalPolicyOverlapUsesInclusiveEffectiveBoundaries(t *testing.T) {
	base := policyCommand("base", ApprovalPolicyActionCreate, uuid.New(), nil)
	left, err := NewApprovalPolicy(base.PolicyID, base.Code, base.Name, base.PolicyVersion, base.Applicability, base.Scope, base.Rules, base.Steps, base.EffectiveFrom, base.EffectiveTo, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	approval := ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: left.PolicyVersion, DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: left.CandidateFingerprint(), ApproverUserID: uuid.New()}
	left.Status = ApprovalPolicyStatusActive
	left.Approval = &approval
	right := left
	right.ID = uuid.New()
	right.PolicyVersion = "v2"
	right.EffectiveFrom = left.EffectiveToValue()
	right.Approval = &approval
	right.Status = ApprovalPolicyStatusActive
	if !PoliciesOverlap(left, right) {
		t.Fatal("equal effective boundary should overlap")
	}
}

func TestSupersededPolicyFamilyHasNoSelectableVersions(t *testing.T) {
	command := policyCommand("superseded", ApprovalPolicyActionCreate, uuid.New(), nil)
	policy, err := NewApprovalPolicy(command.PolicyID, command.Code, command.Name, command.PolicyVersion, command.Applicability, command.Scope, command.Rules, command.Steps, command.EffectiveFrom, command.EffectiveTo, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	approval := ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: policy.PolicyVersion, DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: policy.CandidateFingerprint(), ApproverUserID: uuid.New()}
	policy.Status = ApprovalPolicyStatusActive
	policy.Approval = &approval
	if err := policy.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := policy.Supersede(time.Now()); err != nil {
		t.Fatal(err)
	}
	policy.History = []ApprovalPolicyRevision{{PolicyVersion: approval.PolicyVersion, Version: aggregateversion.Initial(), Snapshot: ApprovalPolicySnapshot{ID: policy.ID, PolicyVersion: approval.PolicyVersion, Status: ApprovalPolicyStatusActive}}}
	if selectable := selectablePolicyVersions(policy); len(selectable) != 0 {
		t.Fatalf("superseded policy family selectable versions = %#v", selectable)
	}
}

func TestSupersededPolicyFamilyDoesNotBlockNewSelectablePolicy(t *testing.T) {
	repository := NewMemoryApprovalPolicyRepository()
	repository.BindAuditRecorder(&MemoryApprovalPolicyAuditRecorder{})
	command := policyCommand("superseded-family", ApprovalPolicyActionCreate, uuid.New(), nil)
	first, err := NewApprovalPolicy(command.PolicyID, command.Code, command.Name, command.PolicyVersion, command.Applicability, command.Scope, command.Rules, command.Steps, command.EffectiveFrom, command.EffectiveTo, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	first.Status = ApprovalPolicyStatusActive
	first.Approval = &ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: first.PolicyVersion, DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: first.CandidateFingerprint(), ApproverUserID: uuid.New()}
	if err := first.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), ApprovalPolicyMutation{After: first, Audit: ApprovalPolicyAuditRecord{PolicyID: first.ID}}); err != nil {
		t.Fatal(err)
	}

	superseded := cloneApprovalPolicy(first)
	if err := superseded.Supersede(time.Now()); err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), ApprovalPolicyMutation{Before: &first, After: superseded, Audit: ApprovalPolicyAuditRecord{PolicyID: superseded.ID}}); err != nil {
		t.Fatal(err)
	}

	secondCommand := command
	secondCommand.PolicyID = uuid.New()
	secondCommand.Code = "policy-selectable-after-supersession"
	second, err := NewApprovalPolicy(secondCommand.PolicyID, secondCommand.Code, secondCommand.Name, secondCommand.PolicyVersion, secondCommand.Applicability, secondCommand.Scope, secondCommand.Rules, secondCommand.Steps, secondCommand.EffectiveFrom, secondCommand.EffectiveTo, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	second.Status = ApprovalPolicyStatusActive
	second.Approval = &ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: second.PolicyVersion, DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: second.CandidateFingerprint(), ApproverUserID: uuid.New()}
	if err := second.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := repository.Commit(context.Background(), ApprovalPolicyMutation{After: second, Audit: ApprovalPolicyAuditRecord{PolicyID: second.ID}}); err != nil {
		t.Fatalf("superseded family should not block a new selectable policy: %v", err)
	}
}

func TestApprovalPolicyUpdateRecognizesTypedContent(t *testing.T) {
	expected := aggregateversion.Initial()
	command := ApprovalPolicyCommand{
		Action:          ApprovalPolicyActionUpdate,
		PolicyID:        uuid.New(),
		PolicyVersion:   "v2",
		ExpectedVersion: &expected,
		Applicability:   ApprovalApplicability{SubjectType: "posting-request"},
		IdempotencyKey:  "typed-only-update",
	}
	if err := command.Validate(); !errors.Is(err, ErrInvalidApprovalPolicy) {
		t.Fatalf("typed-only policy content should be validated as content, got %v", err)
	}
}

func (policy ApprovalPolicy) EffectiveToValue() time.Time {
	if policy.EffectiveTo == nil {
		return policy.EffectiveFrom.Add(24 * time.Hour)
	}
	return *policy.EffectiveTo
}

func policyCommand(id string, action string, policyID uuid.UUID, expected *aggregateversion.AggregateVersion) ApprovalPolicyCommand {
	minimum := decimal.RequireFromString("0.00")
	maximum := decimal.RequireFromString("1000.00")
	return ApprovalPolicyCommand{Action: action, PolicyID: policyID, Code: "policy-" + id, Name: "Policy " + id, PolicyVersion: "v1", Applicability: ApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []string{"USD"}}, Scope: ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, Rules: []ApprovalRule{{ID: uuid.New(), Threshold: AmountThreshold{Minimum: &minimum, Maximum: &maximum, Currency: "USD"}, RequiredRoles: []RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}}}, Steps: []ApprovalStepDefinition{{ID: uuid.New(), Sequence: 1, Name: "Review", RequiredRoles: []RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1}}, EffectiveFrom: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC), ExpectedVersion: expected, IdempotencyKey: id, CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
}

func ptrDecimal(value decimal.Decimal) *decimal.Decimal { return &value }
