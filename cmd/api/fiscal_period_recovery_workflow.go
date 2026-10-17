package main

import (
	"context"
	"errors"
	"strings"

	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/workflow"
)

// fiscalPeriodRecoveryWorkflowValidator is the composition-root adapter for
// the dedicated US6 approval evidence. FPM receives immutable evidence only;
// Workflow remains the owner of approval requests, decisions, and policy
// authorization.
type fiscalPeriodRecoveryWorkflowValidator struct {
	queries *workflow.ApprovalRequestQueryService
}

func (validator fiscalPeriodRecoveryWorkflowValidator) Validate(ctx context.Context, actor fiscalperiod.Actor, decision fiscalperiod.ControlRecoveryApprovalDecision) error {
	if validator.queries == nil {
		return fiscalperiod.ErrControlRecoveryDecisionUnavailable
	}
	approvalRequest, err := validator.queries.Get(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, decision.RequestID)
	if err != nil {
		if errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied) || errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale) || errors.Is(err, workflow.ErrApprovalPolicyNotFound) || errors.Is(err, workflow.ErrApprovalPolicySubjectConflict) || errors.Is(err, workflow.ErrApprovalPolicyApprovalRejected) {
			return fiscalperiod.ErrControlRecoveryDecisionRejected
		}
		return fiscalperiod.ErrControlRecoveryDecisionUnavailable
	}
	if approvalRequest.ID != decision.RequestID || approvalRequest.SubjectFingerprint != decision.CandidateFingerprint || approvalRequest.SubjectVersion != decision.SubjectVersion || approvalRequest.PolicyVersion != decision.PolicyVersion || strings.TrimSpace(decision.SeparationOfDutiesReference) == "" || approvalRequest.Requester.UserID == decision.ApproverUserID {
		return fiscalperiod.ErrControlRecoveryDecisionRejected
	}
	wantStatus := workflow.ApprovalRequestStatusApproved
	wantOutcome := workflow.ApprovalDecisionApproved
	if decision.Outcome == fiscalperiod.CloseApprovalRejected {
		wantStatus = workflow.ApprovalRequestStatusRejected
		wantOutcome = workflow.ApprovalDecisionRejected
	}
	if approvalRequest.Status != wantStatus {
		return fiscalperiod.ErrControlRecoveryDecisionRejected
	}
	for _, recorded := range approvalRequest.Decisions {
		if recorded.ID == decision.DecisionID && recorded.DecisionVersion == decision.DecisionVersion && recorded.Outcome == wantOutcome && recorded.Reason == decision.Reason && recorded.Approver.UserID == decision.ApproverUserID && recorded.PolicyVersion == decision.PolicyVersion && recorded.SubjectVersion == decision.SubjectVersion && recorded.CandidateFingerprint == decision.CandidateFingerprint && recorded.RecordedAt.Equal(decision.RecordedAt) {
			return nil
		}
	}
	return fiscalperiod.ErrControlRecoveryDecisionRejected
}

var _ fiscalperiod.ControlRecoveryDecisionPort = fiscalPeriodRecoveryWorkflowValidator{}
