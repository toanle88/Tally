package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func TestEffectiveApprovalPolicyResolverRequiresExactlyOneEffectiveMatch(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	repository := NewMemoryApprovalPolicyRepository()
	matching := testActiveApprovalPolicy(now, "v1", "finance.approver")
	repository.policies[matching.ID] = matching
	resolver, err := NewEffectiveApprovalPolicyResolver(repository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	selection := testApprovalPolicySelection()
	resolved, err := resolver.Resolve(context.Background(), selection)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ID != matching.ID || resolved.PolicyVersion != "v1" {
		t.Fatalf("resolved policy = %#v", resolved)
	}

	second := testActiveApprovalPolicy(now, "v2", "finance.approver")
	second.ID = uuid.New()
	repository.policies[second.ID] = second
	if _, err := resolver.Resolve(context.Background(), selection); !errors.Is(err, ErrApprovalPolicyAmbiguous) {
		t.Fatalf("ambiguous resolution error = %v", err)
	}

	delete(repository.policies, second.ID)
	selection.Amount = ptrDecimal(decimal.RequireFromString("1000.01"))
	if _, err := resolver.Resolve(context.Background(), selection); !errors.Is(err, ErrApprovalPolicyNotApplicable) {
		t.Fatalf("non-matching threshold error = %v", err)
	}

	selection.Amount = ptrDecimal(decimal.RequireFromString("1000.00"))
	matching.EffectiveTo = ptrTime(now.Add(-time.Minute))
	repository.policies[matching.ID] = matching
	if _, err := resolver.Resolve(context.Background(), selection); !errors.Is(err, ErrApprovalPolicyNotApplicable) {
		t.Fatalf("expired policy error = %v", err)
	}
}

func TestEffectiveApprovalPolicyResolverClassifiesRepositoryFailure(t *testing.T) {
	resolver, err := NewEffectiveApprovalPolicyResolver(unavailableApprovalPolicyRepository{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.Resolve(context.Background(), testApprovalPolicySelection()); !errors.Is(err, ErrApprovalPolicyResolverUnavailable) {
		t.Fatalf("resolver error = %v, want ErrApprovalPolicyResolverUnavailable", err)
	}
}

func TestCreateApprovalRequestResolvesPolicyBindsEvidenceAndReplaysBeforeResolution(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	policyRepository := NewMemoryApprovalPolicyRepository()
	policy := testActiveApprovalPolicy(now, "v1", "finance.approver")
	policyRepository.policies[policy.ID] = policy
	resolver, err := NewEffectiveApprovalPolicyResolver(policyRepository, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	authorizer := MemoryApprovalRequestAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: CreateApprovalRequestsPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}
	repository := NewMemoryApprovalRequestRepository()
	audit := &MemoryApprovalRequestAuditRecorder{}
	events := &MemoryApprovalEventRecorder{}
	repository.BindPorts(audit, events)
	service, err := NewApprovalRequestService(repository, authorizer, audit, events, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := service.BindPolicyResolver(resolver); err != nil {
		t.Fatal(err)
	}
	amount := decimal.RequireFromString("100.00")
	command := CreateApprovalRequestCommand{
		RequestID: uuid.New(), SubjectType: "posting-request", SubjectID: "posting-1", SubjectVersion: 3, SubjectFingerprint: "sha256:source-v3",
		Scope: ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, ResolvePolicy: true,
		Applicability:   ApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []string{"USD"}, DataClassifications: []string{"internal"}},
		TransactionType: "ordinary", DataClassification: "internal", Amount: &amount, Currency: "USD",
		SubjectSnapshot: ApprovalSubjectSnapshot{Label: "Posting 1", Reference: "safe-reference", Summary: "Safe projection", Amount: &amount, Currency: "USD"},
		IdempotencyKey:  "create-approval-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor-subject"}
	result, err := service.CreateApprovalRequest(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Request.PolicyID != policy.ID || result.Request.PolicyVersion != policy.PolicyVersion || result.Request.SnapshotFingerprint == "" || result.Request.DueAt == nil {
		t.Fatalf("resolved request evidence = %#v", result.Request)
	}
	if len(result.Request.ResponsibleRoles) != 1 || result.Request.ResponsibleRoles[0] != "finance.approver" {
		t.Fatalf("responsible roles = %#v", result.Request.ResponsibleRoles)
	}
	if len(events.Events) != 1 || events.Events[0].Type != ApprovalRequestedEventType || events.Events[0].DataClassification != "internal" {
		t.Fatalf("outbox events = %#v", events.Events)
	}

	policy.PolicyVersion = "v2"
	policy.ID = uuid.New()
	policyRepository.policies[policy.ID] = policy
	replayed, err := service.CreateApprovalRequest(context.Background(), actor, command)
	if err != nil || !replayed.Replayed || replayed.Request.PolicyID != result.Request.PolicyID {
		t.Fatalf("replayed request = %#v, err = %v", replayed, err)
	}

	changed := command
	changed.SubjectFingerprint = "sha256:source-v4"
	if _, err := service.CreateApprovalRequest(context.Background(), actor, changed); !errors.Is(err, ErrApprovalPolicyIdempotencyConflict) {
		t.Fatalf("changed source evidence error = %v", err)
	}
}

func TestApprovalRequestQueryServiceAuthorizesFiltersAndPages(t *testing.T) {
	now := time.Date(2026, 10, 14, 0, 0, 0, 0, time.UTC)
	repository := NewMemoryApprovalRequestRepository()
	pending := testApprovalRequest(now.Add(-time.Hour), ApprovalRequestStatusPending, "scope-1")
	pending.DueAt = ptrTime(now.Add(-time.Minute))
	pending.ResponsibleRoles = []string{"finance.approver"}
	completed := testApprovalRequest(now.Add(-2*time.Hour), ApprovalRequestStatusApproved, "scope-1")
	completed.ResponsibleRoles = []string{"finance.approver"}
	otherScope := testApprovalRequest(now, ApprovalRequestStatusPending, "scope-2")
	otherScope.ResponsibleRoles = []string{"finance.approver"}
	repository.requests[pending.ID] = pending
	repository.requests[completed.ID] = completed
	repository.requests[otherScope.ID] = otherScope
	authorizer := MemoryApprovalRequestAuthorizer{ReadDecision: &AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: ReadApprovalRequestsPermission, ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()}}
	service, err := NewApprovalRequestQueryService(repository, authorizer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.List(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "reader"}, ApprovalRequestListFilter{ScopeIDs: []string{"scope-1"}, Assignment: ApprovalRequestAssignmentAll, PageSize: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Requests) != 1 || !page.HasMore || page.After == "" || page.Requests[0].ID != pending.ID {
		t.Fatalf("first page = %#v", page)
	}
	next, err := service.List(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "reader"}, ApprovalRequestListFilter{ScopeIDs: []string{"scope-1"}, Assignment: ApprovalRequestAssignmentAll, PageSize: 2, PageAfter: page.After})
	if err != nil {
		t.Fatal(err)
	}
	if len(next.Requests) != 1 || next.Requests[0].ID != completed.ID {
		t.Fatalf("next page = %#v", next)
	}
	due, err := service.List(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "reader"}, ApprovalRequestListFilter{ScopeIDs: []string{"scope-1"}, Due: ApprovalRequestDueDue})
	if err != nil || len(due.Requests) != 1 || due.Requests[0].ID != pending.ID {
		t.Fatalf("due page = %#v, err = %v", due, err)
	}
	if _, err := service.List(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "reader"}, ApprovalRequestListFilter{PageAfter: "not-a-uuid"}); !errors.Is(err, ErrApprovalPolicyInvalidPage) {
		t.Fatalf("invalid cursor error = %v", err)
	}
}

func TestApprovalRequestAuthorizationRequiresApprovedScope(t *testing.T) {
	decision := AuthorizationDecision{
		Allowed:           true,
		Outcome:           AuthorizationAllowed,
		Permission:        CreateApprovalRequestsPermission,
		ApprovedScopeIDs:  []string{"scope-1"},
		DecisionReference: uuid.New(),
	}
	if err := validateAuthorization(decision, CreateApprovalRequestsPermission, []string{"scope-2"}); !errors.Is(err, ErrApprovalPolicyAuthorizationDenied) {
		t.Fatalf("out-of-scope authorization error = %v", err)
	}
}

func TestCloneApprovalRequestDeepCopiesStepRoutingRoles(t *testing.T) {
	request := testApprovalRequest(time.Now().UTC(), ApprovalRequestStatusPending, "scope-1")
	request.Steps[0].Delegation = DelegationConfiguration{DelegateToRoles: []string{"finance.delegate"}}
	request.Steps[0].Escalation = EscalationConfiguration{EscalateToRoles: []string{"finance.controller"}}

	copy := cloneApprovalRequest(request)
	copy.Steps[0].Delegation.DelegateToRoles[0] = "changed.delegate"
	copy.Steps[0].Escalation.EscalateToRoles[0] = "changed.controller"
	if request.Steps[0].Delegation.DelegateToRoles[0] != "finance.delegate" || request.Steps[0].Escalation.EscalateToRoles[0] != "finance.controller" {
		t.Fatalf("step routing roles were aliased: %#v", request.Steps[0])
	}
}

func TestDurableApprovalRequestFailureCodesRoundTripResolutionOutcomes(t *testing.T) {
	tests := []struct {
		err  error
		code string
	}{
		{ErrApprovalPolicyResolverUnavailable, "POLICY_RESOLUTION_UNAVAILABLE"},
		{ErrApprovalPolicyNotApplicable, "POLICY_NOT_APPLICABLE"},
		{ErrApprovalPolicyAmbiguous, "POLICY_AMBIGUOUS"},
		{ErrApprovalPolicySubjectUnavailable, "SUBJECT_UNAVAILABLE"},
		{ErrApprovalPolicySubjectConflict, "SUBJECT_CONFLICT"},
	}
	for _, test := range tests {
		t.Run(test.code, func(t *testing.T) {
			code, ok := durableApprovalRequestFailureCode(test.err)
			if !ok || code != test.code {
				t.Fatalf("failure code = %q, ok = %t, want %q", code, ok, test.code)
			}
			body, err := json.Marshal(struct {
				Code string `json:"code"`
			}{Code: code})
			if err != nil {
				t.Fatal(err)
			}
			if recovered := durableApprovalRequestFailureError(body); !errors.Is(recovered, test.err) {
				t.Fatalf("recovered error = %v, want %v", recovered, test.err)
			}
		})
	}
}

func testActiveApprovalPolicy(now time.Time, version, role string) ApprovalPolicy {
	minimum := decimal.RequireFromString("0.00")
	maximum := decimal.RequireFromString("1000.00")
	policy, err := NewApprovalPolicy(uuid.New(), "policy-"+version+"-"+uuid.NewString(), "Test policy", version, ApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []string{"USD"}, DataClassifications: []string{"internal"}}, ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, []ApprovalRule{{ID: uuid.New(), Threshold: AmountThreshold{Minimum: &minimum, Maximum: &maximum, Currency: "USD"}, RequiredRoles: []RoleRequirement{{Role: role, MinimumApprovals: 1}}}}, []ApprovalStepDefinition{{ID: uuid.New(), Sequence: 1, Name: "Review", RequiredRoles: []RoleRequirement{{Role: role, MinimumApprovals: 1}}, MinimumApprovals: 1, Escalation: EscalationConfiguration{Enabled: true, AfterSeconds: 60, EscalateToRoles: []string{"finance.controller"}}}}, now.Add(-time.Hour), nil, now)
	if err != nil {
		panic(err)
	}
	policy.Status = ApprovalPolicyStatusActive
	policy.Approval = &ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: version, DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: policy.CandidateFingerprint(), ApproverUserID: uuid.New()}
	return policy
}

func testApprovalPolicySelection() ApprovalPolicySelection {
	amount := decimal.RequireFromString("1000.00")
	return ApprovalPolicySelection{Applicability: ApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", DataClassifications: []string{"internal"}}, Scope: ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, Amount: &amount, Currency: "USD"}
}

func testApprovalRequest(createdAt time.Time, status, scopeID string) ApprovalRequest {
	return ApprovalRequest{ID: uuid.New(), SubjectType: "posting-request", SubjectID: uuid.NewString(), SubjectVersion: 1, SubjectFingerprint: "sha256:subject", PolicyID: uuid.New(), PolicyVersion: "v1", Scope: ApprovalPolicyScope{ScopeIDs: []string{scopeID}}, Requester: Actor{UserID: uuid.New(), SubjectReference: "requester"}, Steps: []ApprovalRequestStep{{ID: uuid.New(), Sequence: 1, Name: "Review", RequiredRoles: []RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1, Status: ApprovalRequestStatusPending}}, Status: status, Version: 1, CreatedAt: createdAt, UpdatedAt: createdAt}
}

func ptrTime(value time.Time) *time.Time { return &value }

type unavailableApprovalPolicyRepository struct{}

func (unavailableApprovalPolicyRepository) Get(context.Context, uuid.UUID) (ApprovalPolicy, error) {
	return ApprovalPolicy{}, ErrApprovalPolicyNotFound
}

func (unavailableApprovalPolicyRepository) List(context.Context) ([]ApprovalPolicy, error) {
	return nil, nil
}

func (unavailableApprovalPolicyRepository) ListSelectable(context.Context) ([]ApprovalPolicy, error) {
	return nil, errors.New("policy store unavailable")
}

func (unavailableApprovalPolicyRepository) Commit(context.Context, ApprovalPolicyMutation) error {
	return nil
}
