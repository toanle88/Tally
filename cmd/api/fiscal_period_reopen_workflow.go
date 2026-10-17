package main

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/workflow"
)

// fiscalPeriodReopenWorkflowValidator is the composition-root adapter for the
// FPM-owned decision port. It asks Workflow for its authorized current
// approval request and verifies the immutable decision evidence there; FPM is
// not given a Workflow repository or schema handle.
type fiscalPeriodReopenWorkflowValidator struct {
	queries *workflow.ApprovalRequestQueryService
}

func (validator fiscalPeriodReopenWorkflowValidator) Validate(ctx context.Context, actor fiscalperiod.Actor, request fiscalperiod.ReopenRequest, decision fiscalperiod.ReopenApprovalDecision) error {
	if validator.queries == nil {
		return fiscalperiod.ErrReopenDecisionUnavailable
	}
	approvalRequest, err := validator.queries.Get(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, decision.RequestID)
	if err != nil {
		if errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied) || errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale) || errors.Is(err, workflow.ErrApprovalPolicyNotFound) || errors.Is(err, workflow.ErrApprovalPolicySubjectConflict) || errors.Is(err, workflow.ErrApprovalPolicyApprovalRejected) {
			return fiscalperiod.ErrReopenDecisionRejected
		}
		return fiscalperiod.ErrReopenDecisionUnavailable
	}
	if request.ApprovalRequestID != uuid.Nil && request.ApprovalRequestID != approvalRequest.ID {
		return fiscalperiod.ErrReopenDecisionRejected
	}
	if approvalRequest.SubjectFingerprint != request.RequestFingerprint || approvalRequest.SubjectVersion < 1 || approvalRequest.PolicyVersion != decision.PolicyVersion || approvalRequest.SubjectVersion != decision.SubjectVersion {
		return fiscalperiod.ErrReopenDecisionRejected
	}
	if approvalRequest.Requester.UserID == decision.ApproverUserID {
		return fiscalperiod.ErrReopenDecisionRejected
	}
	wantStatus := workflow.ApprovalRequestStatusApproved
	wantOutcome := workflow.ApprovalDecisionApproved
	if decision.Outcome == fiscalperiod.ReopenApprovalRejected {
		wantStatus = workflow.ApprovalRequestStatusRejected
		wantOutcome = workflow.ApprovalDecisionRejected
	}
	if approvalRequest.Status != wantStatus {
		return fiscalperiod.ErrReopenDecisionRejected
	}
	for _, recorded := range approvalRequest.Decisions {
		if recorded.ID == decision.DecisionID && recorded.DecisionVersion == decision.DecisionVersion && recorded.Outcome == wantOutcome && recorded.Reason == decision.Reason && recorded.Approver.UserID == decision.ApproverUserID && recorded.PolicyVersion == decision.PolicyVersion && recorded.SubjectVersion == decision.SubjectVersion && recorded.CandidateFingerprint == decision.CandidateFingerprint && recorded.RecordedAt.Equal(decision.RecordedAt) {
			return nil
		}
	}
	return fiscalperiod.ErrReopenDecisionRejected
}

var _ fiscalperiod.ReopenDecisionPort = fiscalPeriodReopenWorkflowValidator{}
