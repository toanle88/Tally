package main

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/events"
	"github.com/toanle88/Tally/internal/platform/integration"
	"github.com/toanle88/Tally/internal/workflow"
)

type evaluatorWorkflowAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

func (authorizer evaluatorWorkflowAuthorizer) AuthorizeApprovalPolicy(ctx context.Context, actor workflow.Actor, command workflow.ApprovalPolicyCommand, current *workflow.ApprovalPolicy) (workflow.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyAuthorizationUnavailable
	}
	scope := command.Scope
	if current != nil && (command.Action == workflow.ApprovalPolicyActionSubmitForApproval || command.Action == workflow.ApprovalPolicyActionActivate || ((command.Action == workflow.ApprovalPolicyActionUpdate || command.Action == workflow.ApprovalPolicyActionMaintain) && !workflowPolicyCommandHasContent(command))) {
		scope = current.Scope
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: workflow.MaintainApprovalPoliciesPermission, RequestedScopeIDs: scope.CanonicalScopeIDs(), DataClassification: "internal"})
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyAuthorizationUnavailable
	}
	return workflowAuthorizationDecision(decision), nil
}

type evaluatorWorkflowRequestAuthorizer struct {
	evaluator   identity.AuthorizationEvaluator
	segregation identity.SegregationDecisionPort
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalDecision(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest) (workflow.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	permission := workflow.CreateApprovalRequestsPermission
	if actor.UserID != request.Requester.UserID {
		permission = "finance.wfa.decide.approval.request"
	}
	classification := strings.TrimSpace(request.DataClassification)
	if classification == "" {
		classification = "internal"
	}
	transactionType := strings.TrimSpace(request.TransactionType)
	var transactionTypePtr *string
	if transactionType != "" {
		transactionTypePtr = &transactionType
	}
	currency := strings.TrimSpace(request.Currency)
	var currencyPtr *string
	if currency != "" {
		currencyPtr = &currency
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{
		ActorID:            actor.UserID,
		Permission:         permission,
		RequestedScopeIDs:  request.Scope.CanonicalScopeIDs(),
		TransactionType:    transactionTypePtr,
		Amount:             request.Amount,
		Currency:           currencyPtr,
		DataClassification: classification,
	})
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	workflowDecision := workflowAuthorizationDecision(decision)
	if !workflowDecision.Allowed || actor.UserID == request.Requester.UserID || request.SubjectType != "approval-policy" {
		return workflowDecision, nil
	}
	if authorizer.segregation == nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	subjectID, err := uuid.Parse(request.SubjectID)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	segregationDecision, err := authorizer.segregation.Evaluate(ctx, identity.SegregationDecisionInput{
		ActorID:     actor.UserID,
		Action:      identity.SegregationActionPolicyApprove,
		SubjectID:   subjectID,
		ScopeIDs:    request.Scope.CanonicalScopeIDs(),
		Permissions: []string{"finance.wfa.decide.approval.request"},
		History:     []identity.SegregationHistoryEntry{},
		ProposerID:  request.Requester.UserID,
		ApproverID:  actor.UserID,
	})
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	if segregationDecision.Outcome == identity.AuthorizationUnavailable {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	if segregationDecision.Outcome == identity.AuthorizationStale {
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationStale, Permission: permission, DecisionReference: segregationDecision.DecisionReference}, nil
	}
	if !segregationDecision.Allowed {
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: permission, DecisionReference: segregationDecision.DecisionReference}, nil
	}
	return workflowDecision, nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalRequestRead(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest) (workflow.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyAuthorizationUnavailable
	}
	classification := request.DataClassification
	if strings.TrimSpace(classification) == "" {
		classification = "internal"
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{
		ActorID:            actor.UserID,
		Permission:         workflow.ReadApprovalRequestsPermission,
		RequestedScopeIDs:  request.Scope.CanonicalScopeIDs(),
		DataClassification: classification,
	})
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyAuthorizationUnavailable
	}
	return workflowAuthorizationDecision(decision), nil
}

func workflowAuthorizationDecision(decision identity.AuthorizationDecision) workflow.AuthorizationDecision {
	return workflow.AuthorizationDecision{
		Allowed:              decision.Allowed,
		Outcome:              workflow.AuthorizationOutcome(decision.Outcome),
		Permission:           decision.Permission,
		ApprovedScopeIDs:     append([]string(nil), decision.ApprovedScopeIDs...),
		PolicyReference:      decision.PolicyReference,
		PolicyVersion:        decision.PolicyVersion,
		DecisionReference:    decision.DecisionReference,
		ReasonCode:           decision.ReasonCode,
		Reason:               decision.Reason,
		ApplicableDimensions: append([]string(nil), decision.ApplicableDimensions...),
	}
}

func workflowPolicyCommandHasContent(command workflow.ApprovalPolicyCommand) bool {
	return strings.TrimSpace(command.Code) != "" || strings.TrimSpace(command.Name) != "" || command.Applicability.SourceContext != "" || len(command.Scope.ScopeIDs) > 0 || len(command.Scope.AccountingScopeIDs) > 0 || len(command.Scope.LegalEntityIDs) > 0 || len(command.Scope.BusinessUnitIDs) > 0 || len(command.Scope.SegmentIDs) > 0 || len(command.Scope.AccountIDs) > 0 || len(command.Scope.FiscalPeriodIDs) > 0 || len(command.Rules) > 0 || len(command.Steps) > 0 || !command.EffectiveFrom.IsZero() || command.EffectiveTo != nil
}

func postgresWorkflowPolicyAuditWriter(auditWriter identity.PostgresAuditWriter) workflow.PostgresApprovalPolicyAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record workflow.ApprovalPolicyAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, workflow.ErrApprovalPolicyAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        record.PolicyID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorAuthenticationSubjectRef,
			Action:                        record.Action,
			ScopeIDs:                      append([]string(nil), record.ScopeIDs...),
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyReference,
			PolicyVersion:                 record.PolicyVersion,
			RevisionVersion:               record.RevisionVersion,
			DecisionReference:             record.DecisionReference,
			ApprovalRequestID:             record.ApprovalRequestID,
			ApprovalDecisionID:            record.ApprovalDecisionID,
			ApproverUserID:                record.ApproverUserID,
			Reason:                        record.Reason,
			EffectiveFrom:                 record.EffectiveFrom,
			EffectiveTo:                   record.EffectiveTo,
			BeforeFingerprint:             record.BeforeFingerprint,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
}

func postgresWorkflowRequestAuditWriter(auditWriter identity.PostgresAuditWriter) workflow.PostgresApprovalRequestAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record workflow.ApprovalRequestAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, workflow.ErrApprovalPolicyAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        record.ApprovalRequestID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorAuthenticationSubjectRef,
			Action:                        record.Action,
			ScopeIDs:                      append([]string(nil), record.ScopeIDs...),
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyReference,
			DecisionReference:             record.DecisionReference,
			PolicyVersion:                 record.PolicyVersion,
			RevisionVersion:               record.RevisionVersion,
			ApprovalRequestID:             record.ApprovalRequestID,
			ApprovalDecisionID:            record.ApprovalDecisionID,
			ApproverUserID:                record.ApproverUserID,
			Reason:                        record.Reason,
			BeforeFingerprint:             record.BeforeFingerprint,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
}

func postgresWorkflowEventWriter(ctx context.Context, tx pgx.Tx, event workflow.ApprovalEvent) error {
	decisionID := ""
	if event.DecisionID != uuid.Nil {
		decisionID = event.DecisionID.String()
	}
	approverUserID := ""
	if event.ApproverUserID != uuid.Nil {
		approverUserID = event.ApproverUserID.String()
	}
	payload, err := json.Marshal(struct {
		ApprovalRequestID    uuid.UUID `json:"approvalRequestId"`
		DecisionID           string    `json:"decisionId,omitempty"`
		DecisionVersion      int64     `json:"decisionVersion,omitempty"`
		Outcome              string    `json:"outcome,omitempty"`
		ApproverUserID       string    `json:"approverUserId,omitempty"`
		PolicyID             uuid.UUID `json:"policyId"`
		PolicyVersion        string    `json:"policyVersion"`
		SubjectID            string    `json:"subjectId"`
		SubjectVersion       int64     `json:"subjectVersion"`
		CandidateFingerprint string    `json:"candidateFingerprint"`
		ScopeIDs             []string  `json:"scopeIds"`
		DataClassification   string    `json:"dataClassification"`
	}{ApprovalRequestID: event.RequestID, DecisionID: decisionID, DecisionVersion: event.DecisionVersion, Outcome: event.Outcome, ApproverUserID: approverUserID, PolicyID: event.PolicyID, PolicyVersion: event.PolicyVersion, SubjectID: event.SubjectID, SubjectVersion: event.SubjectVersion, CandidateFingerprint: event.CandidateFingerprint, ScopeIDs: append([]string(nil), event.ScopeIDs...), DataClassification: event.DataClassification})
	if err != nil {
		return err
	}
	fingerprint, err := events.ComputePayloadFingerprint(payload)
	if err != nil {
		return err
	}
	if _, err := uuid.Parse(event.CorrelationID); err != nil {
		return err
	}
	if _, err := uuid.Parse(event.CausationID); err != nil {
		return err
	}
	classification := strings.TrimSpace(event.DataClassification)
	if classification == "" {
		classification = string(events.Internal)
	}
	envelope, err := events.NewEnvelope(events.EnvelopeInput{
		MessageID:          uuid.NewString(),
		EventType:          event.Type,
		EventVersion:       1,
		OccurredAt:         event.OccurredAt,
		SourceContext:      "workflow",
		AggregateID:        event.RequestID.String(),
		AggregateVersion:   event.AggregateVersion,
		CorrelationID:      event.CorrelationID,
		CausationID:        event.CausationID,
		DataClassification: events.Classification(classification),
		PayloadFingerprint: fingerprint,
		Data:               payload,
	})
	if err != nil {
		return err
	}
	return integration.WritePublication(ctx, tx, integration.Publication{Event: envelope, AvailableAt: event.OccurredAt})
}

var _ workflow.ApprovalPolicyAuthorizer = evaluatorWorkflowAuthorizer{}
var _ workflow.ApprovalRequestAuthorizer = evaluatorWorkflowRequestAuthorizer{}
var _ workflow.PostgresApprovalPolicyAuditWriter = postgresWorkflowPolicyAuditWriter(nil)
var _ workflow.PostgresApprovalRequestAuditWriter = postgresWorkflowRequestAuditWriter(nil)
var _ workflow.PostgresApprovalEventWriter = postgresWorkflowEventWriter
