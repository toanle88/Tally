package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/workflow"
)

func TestApprovalEscalationHandlerEstablishesTypedEvidenceAndReplays(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	requestID := uuid.New()
	stepID := uuid.New()
	policyID := uuid.New()
	request := workflow.ApprovalRequest{
		ID: requestID, SubjectType: "posting-request", SubjectID: "posting-1", SubjectVersion: 1, SubjectFingerprint: "sha256:subject-v1",
		PolicyID: policyID, PolicyVersion: "v1", Scope: workflow.ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}}, Requester: workflow.Actor{UserID: uuid.New(), SubjectReference: "requester"},
		Steps:            []workflow.ApprovalRequestStep{{ID: stepID, Sequence: 1, Name: "Review", RequiredRoles: []workflow.RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1, Escalation: workflow.EscalationConfiguration{Enabled: true, AfterSeconds: 60, EscalateToRoles: []string{"finance.controller"}}, Status: workflow.ApprovalRequestStatusPending}},
		ResponsibleRoles: []string{"finance.approver"}, Status: workflow.ApprovalRequestStatusPending, Version: 1, DueAt: timePtr(now.Add(-time.Minute)), CreatedAt: now.Add(-time.Hour), UpdatedAt: now.Add(-time.Hour),
	}
	authorizer := workflow.MemoryApprovalRequestAuthorizer{Decision: workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.EscalateApprovalPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()}}
	repository := workflow.NewMemoryApprovalRequestRepository()
	audit := &workflow.MemoryApprovalRequestAuditRecorder{}
	events := &workflow.MemoryApprovalEventRecorder{}
	repository.BindPorts(audit, events)
	if _, err := repository.Save(context.Background(), request, nil, workflow.ApprovalRequestAuditRecord{ApprovalRequestID: requestID}, workflow.ApprovalEvent{Type: workflow.ApprovalRequestedEventType, RequestID: requestID, OccurredAt: now}); err != nil {
		t.Fatal(err)
	}
	service, err := workflow.NewApprovalRequestService(repository, authorizer, audit, events, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{ApprovalRequestService: service}
	actorID := uuid.New()
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: actorID, Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "approval-owner"}})
	if err != nil {
		t.Fatal(err)
	}
	requestBody := &generated.WfaEscalateApprovalCommandRequest{
		CommandId: generated.UUID(uuid.New()), ExpectedVersion: 1,
		Data: generated.WfaEscalateApprovalCommandData{RequestId: generated.UUID(requestID), EscalationId: generated.UUID(uuid.New()), StepId: generated.UUID(stepID), Reason: generated.WfaEscalateApprovalCommandDataReasonOverdue, PolicyVersion: "v1", SubjectVersion: 1, CandidateFingerprint: request.SubjectFingerprint},
	}
	params := generated.WfaEscalateApprovalParams{IdempotencyKey: "http-escalation-1", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))}
	response, err := handler.WfaEscalateApproval(actorContext, requestBody, params)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.WfaEscalateApprovalEstablishedResult)
	if !ok || established.Data.Request.Status != generated.WfaApprovalRequestProjectionStatusPending || established.Data.Request.Steps[0].Status != workflow.ApprovalRequestStepStatusEscalated || established.Data.Escalation.TargetRoles[0] != "finance.controller" || established.Data.Escalation.AuditReference == (generated.UUID(uuid.Nil)) || !established.Data.Escalation.ActorUserId.Set || uuid.UUID(established.Data.Escalation.ActorUserId.Value) != actorID || !established.Data.Escalation.ActorSubjectReference.Set || established.Data.Escalation.ActorSubjectReference.Value != "approval-owner" || established.Data.Replayed {
		t.Fatalf("escalation response = %#v", response)
	}

	replayedResponse, err := handler.WfaEscalateApproval(actorContext, requestBody, params)
	if err != nil {
		t.Fatal(err)
	}
	if replayed, ok := replayedResponse.(*generated.WfaEscalateApprovalEstablishedResult); !ok || !replayed.Data.Replayed {
		t.Fatalf("replayed escalation response = %#v", replayedResponse)
	}
}

func TestApprovalProjectionKeepsHistoricalEscalationSeparateFromCompletedState(t *testing.T) {
	now := time.Date(2026, 10, 15, 10, 0, 0, 0, time.UTC)
	request := workflow.ApprovalRequest{
		ID: uuid.New(), SubjectType: "posting-request", SubjectID: "posting-1", SubjectVersion: 1, SubjectFingerprint: "sha256:subject-v1",
		PolicyID: uuid.New(), PolicyVersion: "v1", Scope: workflow.ApprovalPolicyScope{ScopeIDs: []string{"scope-1"}},
		Steps: []workflow.ApprovalRequestStep{{ID: uuid.New(), Sequence: 1, Name: "Review", RequiredRoles: []workflow.RoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1, Status: workflow.ApprovalRequestStatusApproved}},
		Status: workflow.ApprovalRequestStatusApproved, Version: 2, CreatedAt: now.Add(-time.Hour), UpdatedAt: now,
		Escalations: []workflow.ApprovalEscalation{{ID: uuid.New(), StepID: uuid.New(), Reason: workflow.ApprovalEscalationReasonOverdue, ResponsibleRoles: []string{"finance.approver"}, TargetRoles: []string{"finance.controller"}, PolicyID: uuid.Nil, PolicyVersion: "v1", SubjectVersion: 1, CandidateFingerprint: "sha256:subject-v1", ActorUserID: uuid.New(), ActorSubjectReference: "approval-owner", OccurredAt: now.Add(-time.Minute), AuditReference: uuid.New()}},
	}
	request.Escalations[0].StepID = request.Steps[0].ID
	request.Escalations[0].PolicyID = request.PolicyID

	projection := generatedWorkflowApprovalRequestProjection(request)
	if projection.Status != generated.WfaApprovalRequestProjectionStatusApproved || len(projection.EscalationTargetRoles) != 0 || projection.NextAction.Set || projection.RecoveryPath.Set {
		t.Fatalf("completed projection exposed current escalation state: %#v", projection)
	}
	if len(projection.Escalations) != 1 {
		t.Fatalf("completed projection lost escalation history: %#v", projection.Escalations)
	}
}

func timePtr(value time.Time) *time.Time { return &value }
