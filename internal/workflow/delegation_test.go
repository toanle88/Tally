package workflow

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

type delegationTestAuthorizer struct {
	decision AuthorizationDecision
	err      error
}

func (authorizer delegationTestAuthorizer) AuthorizeApprovalDelegation(context.Context, Actor, DelegateApprovalCommand, ApprovalPolicy) (AuthorizationDecision, error) {
	return authorizer.decision, authorizer.err
}

func delegationTestPolicy(now time.Time, id uuid.UUID) ApprovalPolicy {
	policy := testActiveApprovalPolicy(now, "v1", "finance.approver")
	policy.ID = id
	policy.Scope = ApprovalPolicyScope{ScopeIDs: []string{"scope-1", "scope-2"}}
	policy.Steps[0].RequiredRoles[0].AllowDelegation = true
	policy.Steps[0].Delegation = DelegationConfiguration{Enabled: true, DelegateToRoles: []string{"finance.delegate"}, MaximumDepth: 1}
	return policy
}

func TestDelegateApprovalEstablishesFiniteImmutableAuthorityAndReplays(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	policyID := uuid.New()
	policyRepository := NewMemoryApprovalPolicyRepository()
	policy := delegationTestPolicy(now, policyID)
	policyRepository.policies[policy.ID] = policy
	audit := &MemoryDelegationAuditRecorder{}
	events := &MemoryDelegationEventRecorder{}
	repository := NewMemoryDelegationRepository()
	repository.BindPorts(audit, events)
	decisionReference := uuid.New()
	service, err := NewDelegationService(repository, policyRepository, delegationTestAuthorizer{decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: DelegateApprovalPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: decisionReference}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "delegator-subject"}
	command := DelegateApprovalCommand{DelegationID: uuid.New(), DelegateUserID: uuid.New(), PolicyID: policyID, PolicyVersion: policy.PolicyVersion, Scope: ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, EffectiveFrom: now.Add(time.Hour), EffectiveTo: now.Add(8 * time.Hour), IdempotencyKey: "delegate-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	result, err := service.DelegateApproval(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Delegation.ID != command.DelegationID || result.Delegation.DelegatorUserID != actor.UserID || result.Delegation.Version.Value() != 1 || result.Delegation.ActiveAt(now.Add(2*time.Hour)) == false || result.Delegation.AuditReference == uuid.Nil {
		t.Fatalf("delegation result = %#v", result)
	}
	if len(audit.Records) != 1 || len(events.Events) != 1 || events.Events[0].Type != ApprovalDelegatedEventType {
		t.Fatalf("audit/events = %#v / %#v", audit.Records, events.Events)
	}
	replayed, err := service.DelegateApproval(context.Background(), actor, command)
	if err != nil || !replayed.Replayed || replayed.Delegation.ID != result.Delegation.ID {
		t.Fatalf("replayed result = %#v, err = %v", replayed, err)
	}
	changed := command
	changed.DelegateUserID = uuid.New()
	if _, err := service.DelegateApproval(context.Background(), actor, changed); !errors.Is(err, ErrDelegationIdempotencyConflict) {
		t.Fatalf("changed fingerprint error = %v", err)
	}
}

func TestDelegateApprovalRejectsSelfExpiredScopeAndNonDelegablePolicy(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	policyID := uuid.New()
	policyRepository := NewMemoryApprovalPolicyRepository()
	policy := delegationTestPolicy(now, policyID)
	policyRepository.policies[policy.ID] = policy
	repository := NewMemoryDelegationRepository()
	repository.BindPorts(&MemoryDelegationAuditRecorder{}, &MemoryDelegationEventRecorder{})
	allow := delegationTestAuthorizer{decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: DelegateApprovalPermission, ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()}}
	service, err := NewDelegationService(repository, policyRepository, allow, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	base := DelegateApprovalCommand{DelegationID: uuid.New(), DelegateUserID: uuid.New(), PolicyID: policyID, PolicyVersion: policy.PolicyVersion, Scope: ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, EffectiveFrom: now, EffectiveTo: now.Add(time.Hour), IdempotencyKey: "case"}
	self := base
	// The service derives the delegator from the actor, so use the same identity
	// for the self-delegation assertion.
	actor := Actor{UserID: uuid.New(), SubjectReference: "delegator"}
	self.DelegateUserID = actor.UserID
	if _, err := service.DelegateApproval(context.Background(), actor, self); !errors.Is(err, ErrInvalidDelegation) {
		t.Fatalf("self-delegation error = %v", err)
	}
	expired := base
	expired.DelegationID = uuid.New()
	expired.EffectiveTo = now.Add(-time.Minute)
	if _, err := service.DelegateApproval(context.Background(), actor, expired); !errors.Is(err, ErrInvalidDelegationCommand) {
		t.Fatalf("expired interval validation error = %v", err)
	}
	outOfScope := base
	outOfScope.DelegationID = uuid.New()
	outOfScope.IdempotencyKey = "out-of-scope"
	outOfScope.Scope = ApprovalPolicyScope{ScopeIDs: []string{"outside"}}
	if _, err := service.DelegateApproval(context.Background(), actor, outOfScope); !errors.Is(err, ErrDelegationScopeConflict) {
		t.Fatalf("out-of-scope error = %v", err)
	}
	nonDelegable := policy
	nonDelegable.ID = uuid.New()
	nonDelegable.Steps[0].Delegation = DelegationConfiguration{}
	nonDelegable.Steps[0].RequiredRoles[0].AllowDelegation = false
	policyRepository.policies[nonDelegable.ID] = nonDelegable
	blocked := base
	blocked.DelegationID = uuid.New()
	blocked.PolicyID = nonDelegable.ID
	blocked.IdempotencyKey = "non-delegable"
	if _, err := service.DelegateApproval(context.Background(), actor, blocked); !errors.Is(err, ErrDelegationPolicyIncompatible) {
		t.Fatalf("non-delegable policy error = %v", err)
	}
}

func TestApprovalDecisionRequiresLiveDelegationWhenReaderIsBound(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	request := testApprovalRequest(now, ApprovalRequestStatusPending, "scope-1")
	policy := testActiveApprovalPolicy(now, request.PolicyVersion, "finance.approver")
	policy.ID = request.PolicyID
	policyRepository := NewMemoryApprovalPolicyRepository()
	policyRepository.policies[policy.ID] = policy
	repository := NewMemoryApprovalRequestRepository()
	audit := &MemoryApprovalRequestAuditRecorder{}
	events := &MemoryApprovalEventRecorder{}
	repository.BindPorts(audit, events)
	repository.requests[request.ID] = request
	delegationRepository := NewMemoryDelegationRepository()
	delegation := Delegation{ID: uuid.New(), DelegatorUserID: uuid.New(), DelegateUserID: uuid.New(), PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion, Scope: request.Scope, EffectiveFrom: now.Add(-time.Hour), EffectiveTo: now.Add(time.Hour), Version: 1, CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour)}
	delegationRepository.delegations[delegation.ID] = delegation
	authorizer := MemoryApprovalRequestAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: DecideApprovalRequestsPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}
	service, err := NewApprovalRequestService(repository, authorizer, audit, events, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := service.BindPolicyRepository(policyRepository); err != nil {
		t.Fatal(err)
	}
	if err := service.BindDelegationReader(delegationRepository); err != nil {
		t.Fatal(err)
	}
	expected := request.Version
	command := DecideApprovalRequestCommand{RequestID: request.ID, ExpectedVersion: &expected, DecisionID: uuid.New(), StepID: request.Steps[0].ID, Outcome: ApprovalDecisionApproved, PolicyVersion: request.PolicyVersion, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, DelegationID: delegation.ID, IdempotencyKey: "live-delegation"}
	decisionResult, err := service.DecideApprovalRequest(context.Background(), Actor{UserID: delegation.DelegateUserID, SubjectReference: "delegate"}, command)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisionResult.Request.Decisions) != 1 || decisionResult.Request.Decisions[0].DelegationReference != delegation.ID.String() {
		t.Fatalf("delegated decision evidence = %#v", decisionResult.Request.Decisions)
	}

	request = testApprovalRequest(now, ApprovalRequestStatusPending, "scope-1")
	request.PolicyID = policy.ID
	request.PolicyVersion = policy.PolicyVersion
	request.Steps[0].RequiredRoles = policy.Steps[0].RequiredRoles
	request.Steps[0].Delegation = policy.Steps[0].Delegation
	request.Version = 1
	repository.requests[request.ID] = request
	expired := delegation
	expired.ID = uuid.New()
	expired.EffectiveTo = now.Add(-time.Minute)
	delegationRepository.delegations[expired.ID] = expired
	expected = request.Version
	command.RequestID = request.ID
	command.DecisionID = uuid.New()
	command.StepID = request.Steps[0].ID
	command.DelegationID = expired.ID
	command.ExpectedVersion = &expected
	command.IdempotencyKey = "expired-live-delegation"
	if _, err := service.DecideApprovalRequest(context.Background(), Actor{UserID: expired.DelegateUserID, SubjectReference: "delegate"}, command); !errors.Is(err, ErrDelegationExpired) {
		t.Fatalf("expired delegation error = %v", err)
	}
}

func TestDelegateApprovalConcurrentEquivalentCommandsEstablishOnce(t *testing.T) {
	now := time.Date(2026, 10, 15, 9, 0, 0, 0, time.UTC)
	policyID := uuid.New()
	policyRepository := NewMemoryApprovalPolicyRepository()
	policy := delegationTestPolicy(now, policyID)
	policyRepository.policies[policy.ID] = policy
	repository := NewMemoryDelegationRepository()
	audit := &MemoryDelegationAuditRecorder{}
	events := &MemoryDelegationEventRecorder{}
	repository.BindPorts(audit, events)
	service, err := NewDelegationService(repository, policyRepository, delegationTestAuthorizer{decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: DelegateApprovalPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "delegator"}
	command := DelegateApprovalCommand{DelegationID: uuid.New(), DelegateUserID: uuid.New(), PolicyID: policyID, PolicyVersion: policy.PolicyVersion, Scope: ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, EffectiveFrom: now.Add(time.Hour), EffectiveTo: now.Add(8 * time.Hour), IdempotencyKey: "concurrent-delegation", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	results := make(chan DelegationCommandResult, 2)
	errorsCh := make(chan error, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, callErr := service.DelegateApproval(context.Background(), actor, command)
			results <- result
			errorsCh <- callErr
		}()
	}
	wait.Wait()
	close(results)
	close(errorsCh)
	var established, replayed int
	for result := range results {
		if result.Replayed {
			replayed++
		} else {
			established++
		}
	}
	for callErr := range errorsCh {
		if callErr != nil {
			t.Fatalf("concurrent delegation error = %v", callErr)
		}
	}
	if established != 1 || replayed != 1 || len(audit.Records) != 1 || len(events.Events) != 1 {
		t.Fatalf("concurrent delegation evidence = established %d, replayed %d, audit %d, events %d", established, replayed, len(audit.Records), len(events.Events))
	}
}
