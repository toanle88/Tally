package workflow

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestEscalateApprovalRecordsPolicyEvidenceWithoutDecisionAndReplays(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	request := escalationTestRequest(now, now.Add(-time.Second))
	service, repository, audit, events := escalationTestService(t, now, request)
	actor := Actor{UserID: uuid.New(), SubjectReference: "approval-owner"}
	expected := request.Version
	command := EscalateApprovalCommand{
		RequestID: request.ID, ExpectedVersion: &expected, EscalationID: uuid.New(), StepID: request.Steps[0].ID,
		Reason: ApprovalEscalationReasonOverdue, PolicyVersion: request.PolicyVersion, SubjectVersion: request.SubjectVersion,
		CandidateFingerprint: request.SubjectFingerprint, IdempotencyKey: "escalate-overdue-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}

	result, err := service.EscalateApproval(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.Replayed || result.Request.Status != ApprovalRequestStatusPending || len(result.Request.Decisions) != 0 || result.Request.Steps[0].Status != ApprovalRequestStepStatusEscalated {
		t.Fatalf("escalation result = %#v", result)
	}
	if result.Escalation == nil || result.Escalation.Reason != ApprovalEscalationReasonOverdue || result.Escalation.TargetRoles[0] != "finance.controller" || result.Escalation.AuditReference == uuid.Nil {
		t.Fatalf("escalation evidence = %#v", result.Escalation)
	}
	stored, err := repository.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Escalations) != 1 || stored.Escalations[0].AuditReference != result.Escalation.AuditReference {
		t.Fatalf("stored escalation = %#v", stored.Escalations)
	}
	if len(audit.Records) != 1 || audit.Records[0].ApprovalEscalationID != command.EscalationID || len(events.Events) != 1 || events.Events[0].Type != ApprovalEscalatedEventType {
		t.Fatalf("audit/events = %#v / %#v", audit.Records, events.Events)
	}

	replayed, err := service.EscalateApproval(context.Background(), actor, command)
	if err != nil || !replayed.Replayed || replayed.Escalation == nil || replayed.Escalation.ID != command.EscalationID {
		t.Fatalf("replayed escalation = %#v, err = %v", replayed, err)
	}
	changed := command
	changed.Blocker = "changed command evidence"
	if _, err := service.EscalateApproval(context.Background(), actor, changed); !errors.Is(err, ErrApprovalPolicyIdempotencyConflict) {
		t.Fatalf("changed escalation fingerprint error = %v", err)
	}
}

func TestEscalateApprovalEnforcesOverdueBlockedAndPolicyBoundaries(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	actor := Actor{UserID: uuid.New(), SubjectReference: "approval-owner"}

	tooEarly := escalationTestRequest(now, now.Add(time.Minute))
	service, _, _, _ := escalationTestService(t, now, tooEarly)
	expected := tooEarly.Version
	if _, err := service.EscalateApproval(context.Background(), actor, EscalateApprovalCommand{RequestID: tooEarly.ID, ExpectedVersion: &expected, EscalationID: uuid.New(), StepID: tooEarly.Steps[0].ID, Reason: ApprovalEscalationReasonOverdue, PolicyVersion: tooEarly.PolicyVersion, SubjectVersion: tooEarly.SubjectVersion, CandidateFingerprint: tooEarly.SubjectFingerprint, IdempotencyKey: "too-early"}); !errors.Is(err, ErrApprovalPolicyEscalationNotEligible) {
		t.Fatalf("early escalation error = %v", err)
	}

	blocked := escalationTestRequest(now, now.Add(time.Minute))
	service, _, _, _ = escalationTestService(t, now, blocked)
	expected = blocked.Version
	blockedCommand := EscalateApprovalCommand{RequestID: blocked.ID, ExpectedVersion: &expected, EscalationID: uuid.New(), StepID: blocked.Steps[0].ID, Reason: ApprovalEscalationReasonBlocked, Blocker: "Controller assignment is unavailable", PolicyVersion: blocked.PolicyVersion, SubjectVersion: blocked.SubjectVersion, CandidateFingerprint: blocked.SubjectFingerprint, IdempotencyKey: "blocked-1"}
	if result, err := service.EscalateApproval(context.Background(), actor, blockedCommand); err != nil || result.Escalation == nil || result.Escalation.Blocker != blockedCommand.Blocker {
		t.Fatalf("blocked escalation = %#v, err = %v", result, err)
	}

	invalid := blockedCommand
	invalid.Blocker = ""
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidApprovalPolicyCommand) {
		t.Fatalf("missing blocker validation error = %v", err)
	}

	disabled := escalationTestRequest(now, now.Add(-time.Minute))
	disabled.Steps[0].Escalation.Enabled = false
	service, _, _, _ = escalationTestService(t, now, disabled)
	expected = disabled.Version
	if _, err := service.EscalateApproval(context.Background(), actor, EscalateApprovalCommand{RequestID: disabled.ID, ExpectedVersion: &expected, EscalationID: uuid.New(), StepID: disabled.Steps[0].ID, Reason: ApprovalEscalationReasonOverdue, PolicyVersion: disabled.PolicyVersion, SubjectVersion: disabled.SubjectVersion, CandidateFingerprint: disabled.SubjectFingerprint, IdempotencyKey: "disabled"}); !errors.Is(err, ErrApprovalPolicyEscalationNotEligible) {
		t.Fatalf("disabled escalation error = %v", err)
	}
}

func TestEscalateApprovalRequiresEscalationAuthorization(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	request := escalationTestRequest(now, now.Add(-time.Minute))
	policy := testActiveApprovalPolicy(now, request.PolicyVersion, "finance.approver")
	policy.ID = request.PolicyID
	policyRepository := NewMemoryApprovalPolicyRepository()
	policyRepository.policies[policy.ID] = policy
	repository := NewMemoryApprovalRequestRepository()
	audit := &MemoryApprovalRequestAuditRecorder{}
	events := &MemoryApprovalEventRecorder{}
	repository.BindPorts(audit, events)
	repository.requests[request.ID] = request
	authorizer := MemoryApprovalRequestAuthorizer{Decision: AuthorizationDecision{Allowed: false, Outcome: AuthorizationDenied, Permission: EscalateApprovalPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}
	service, err := NewApprovalRequestService(repository, authorizer, audit, events, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := service.BindPolicyRepository(policyRepository); err != nil {
		t.Fatal(err)
	}
	expected := request.Version
	command := EscalateApprovalCommand{RequestID: request.ID, ExpectedVersion: &expected, EscalationID: uuid.New(), StepID: request.Steps[0].ID, Reason: ApprovalEscalationReasonOverdue, PolicyVersion: request.PolicyVersion, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, IdempotencyKey: "authorization-denied"}
	if _, err := service.EscalateApproval(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "unauthorized-owner"}, command); !errors.Is(err, ErrApprovalPolicyAuthorizationDenied) {
		t.Fatalf("authorization error = %v", err)
	}
	stored, err := repository.Get(context.Background(), request.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Escalations) != 0 || len(audit.Records) != 0 || len(events.Events) != 0 {
		t.Fatalf("authorization denial changed request evidence: %#v / %#v / %#v", stored.Escalations, audit.Records, events.Events)
	}
}

func TestApprovalEscalationWorkerUsesDeterministicIdentityAndSkipsRaces(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	request := escalationTestRequest(now, now.Add(-time.Minute))
	service, repository, _, _ := escalationTestService(t, now, request)
	worker, err := NewApprovalEscalationWorker(service, repository, func() time.Time { return now }, ApprovalEscalationWorkerConfig{Interval: time.Minute, BatchSize: 100, ProcessReference: ApprovalEscalationProcess})
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := repository.ListDueApprovalEscalationCandidates(context.Background(), now, 100)
	if err != nil || len(candidates) != 1 {
		t.Fatalf("candidates = %#v, err = %v", candidates, err)
	}
	firstCommand := worker.command(candidates[0])
	secondCommand := worker.command(candidates[0])
	if firstCommand.EscalationID != secondCommand.EscalationID || firstCommand.IdempotencyKey != secondCommand.IdempotencyKey {
		t.Fatalf("worker identity is not deterministic: %#v / %#v", firstCommand, secondCommand)
	}
	result, err := worker.RunBatch(context.Background())
	if err != nil || result.Examined != 1 || result.Escalated != 1 || result.Skipped != 0 {
		t.Fatalf("worker result = %#v, err = %v", result, err)
	}
	if candidates, err := repository.ListDueApprovalEscalationCandidates(context.Background(), now, 100); err != nil || len(candidates) != 0 {
		t.Fatalf("post-worker candidates = %#v, err = %v", candidates, err)
	}
}

func TestApprovalRequestQueryTreatsExplicitBlockedEscalationAsBlocked(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	request := escalationTestRequest(now, now.Add(time.Minute))
	request.Steps[0].Status = ApprovalRequestStepStatusEscalated
	request.Escalations = []ApprovalEscalation{{ID: uuid.New(), StepID: request.Steps[0].ID, Reason: ApprovalEscalationReasonBlocked, Blocker: "No eligible owner", ResponsibleRoles: []string{"finance.approver"}, TargetRoles: []string{"finance.controller"}, PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, ActorUserID: uuid.New(), ActorSubjectReference: "owner", OccurredAt: now, AuditReference: uuid.New()}}
	repository := NewMemoryApprovalRequestRepository()
	repository.requests[request.ID] = request
	authorizer := MemoryApprovalRequestAuthorizer{ReadDecision: &AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: ReadApprovalRequestsPermission, ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()}}
	service, err := NewApprovalRequestQueryService(repository, authorizer, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	page, err := service.List(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "reader"}, ApprovalRequestListFilter{Assignment: ApprovalRequestAssignmentAll, State: ApprovalRequestStateBlocked})
	if err != nil || len(page.Requests) != 1 || page.Requests[0].ID != request.ID {
		t.Fatalf("blocked page = %#v, err = %v", page, err)
	}
}

func escalationTestService(t *testing.T, now time.Time, request ApprovalRequest) (*ApprovalRequestService, *MemoryApprovalRequestRepository, *MemoryApprovalRequestAuditRecorder, *MemoryApprovalEventRecorder) {
	t.Helper()
	policy := testActiveApprovalPolicy(now, request.PolicyVersion, "finance.approver")
	policy.ID = request.PolicyID
	policyRepository := NewMemoryApprovalPolicyRepository()
	policyRepository.policies[policy.ID] = policy
	repository := NewMemoryApprovalRequestRepository()
	audit := &MemoryApprovalRequestAuditRecorder{}
	events := &MemoryApprovalEventRecorder{}
	repository.BindPorts(audit, events)
	repository.requests[request.ID] = request
	authorizer := MemoryApprovalRequestAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: EscalateApprovalPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}
	service, err := NewApprovalRequestService(repository, authorizer, audit, events, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if err := service.BindPolicyRepository(policyRepository); err != nil {
		t.Fatal(err)
	}
	return service, repository, audit, events
}

func escalationTestRequest(now, dueAt time.Time) ApprovalRequest {
	request := testApprovalRequest(now.Add(-time.Hour), ApprovalRequestStatusPending, "scope-1")
	request.Steps[0].Escalation = EscalationConfiguration{Enabled: true, AfterSeconds: 60, EscalateToRoles: []string{"finance.controller"}}
	request.DueAt = ptrTime(dueAt)
	return request
}
