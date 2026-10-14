package main

import (
	"context"
	"encoding/json"
	"strings"
	"time"

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
	users       identity.UserRepository
	roles       identity.RoleRepository
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalDecision(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest) (workflow.AuthorizationDecision, error) {
	roleIDs, err := authorizer.approvalRoleIDs(ctx, actor, false)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	return authorizer.evaluateRequest(ctx, actor, request, workflow.CreateApprovalRequestsPermission, roleIDs)
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalRequestDecision(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, step workflow.ApprovalRequestStep, delegationReference string) (workflow.AuthorizationDecision, error) {
	roleIDs, err := authorizer.approvalRoleIDs(ctx, actor, true)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	decision, err := authorizer.evaluateRequest(ctx, actor, request, workflow.DecideApprovalRequestsPermission, roleIDs)
	if err != nil || !decision.Allowed {
		return decision, err
	}
	if authorizer.users != nil && authorizer.roles != nil {
		eligible, err := authorizer.actorEligibleForStep(ctx, actor, request, step, delegationReference)
		if err != nil {
			return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
		}
		if !eligible {
			return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.DecideApprovalRequestsPermission, DecisionReference: decision.DecisionReference, ReasonCode: "approval-step-role-required"}, nil
		}
	}
	if request.SubjectType != "approval-policy" || authorizer.segregation == nil {
		return decision, nil
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
		Permissions: []string{workflow.DecideApprovalRequestsPermission},
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
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationStale, Permission: workflow.DecideApprovalRequestsPermission, DecisionReference: segregationDecision.DecisionReference}, nil
	}
	if !segregationDecision.Allowed {
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.DecideApprovalRequestsPermission, DecisionReference: segregationDecision.DecisionReference}, nil
	}
	return decision, nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) evaluateRequest(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, permission string, roleIDs []uuid.UUID) (workflow.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
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
		RoleIDs:            append([]uuid.UUID(nil), roleIDs...),
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
	return workflowAuthorizationDecision(decision), nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) approvalRoleIDs(ctx context.Context, actor workflow.Actor, required bool) ([]uuid.UUID, error) {
	if authorizer.users == nil || authorizer.roles == nil {
		return nil, nil
	}
	user, err := authorizer.users.Get(ctx, actor.UserID)
	if err != nil {
		if required {
			return nil, err
		}
		return nil, nil
	}
	if user.Status != identity.UserStatusActive {
		if required {
			return nil, identity.ErrAuthorizationDenied
		}
		return nil, nil
	}
	roleIDs := make([]uuid.UUID, 0, len(user.Assignments))
	for _, assignment := range user.Assignments {
		role, err := authorizer.roles.Get(ctx, assignment.RoleID)
		if err != nil {
			if required {
				return nil, err
			}
			continue
		}
		if role.Status == identity.RoleStatusActive {
			roleIDs = append(roleIDs, role.ID)
		}
	}
	return roleIDs, nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) actorEligibleForStep(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, step workflow.ApprovalRequestStep, delegationReference string) (bool, error) {
	if authorizer.users == nil || authorizer.roles == nil {
		return true, nil
	}
	user, err := authorizer.users.Get(ctx, actor.UserID)
	if err != nil {
		return false, err
	}
	if user.Status != identity.UserStatusActive {
		return false, nil
	}
	requestedScopes := request.Scope.CanonicalScopeIDs()
	for _, assignment := range user.Assignments {
		role, err := authorizer.roles.Get(ctx, assignment.RoleID)
		if err != nil {
			return false, err
		}
		if role.Status != identity.RoleStatusActive || !approvalAssignmentCoversScopes(assignment.Scopes, requestedScopes) {
			continue
		}
		for _, requirement := range step.RequiredRoles {
			if role.Name == requirement.Role {
				return true, nil
			}
			if requirement.AllowDelegation && step.Delegation.Enabled && strings.TrimSpace(delegationReference) != "" && containsApprovalRole(step.Delegation.DelegateToRoles, role.Name) {
				return true, nil
			}
		}
	}
	return false, nil
}

func approvalAssignmentCoversScopes(scopes []identity.EntityAccessScope, requested []string) bool {
	allowed := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		allowed[strings.TrimSpace(scope.ScopeID)] = struct{}{}
	}
	if _, ok := allowed["*"]; ok {
		return true
	}
	for _, scope := range requested {
		if _, ok := allowed[strings.TrimSpace(scope)]; !ok {
			return false
		}
	}
	return true
}

func containsApprovalRole(roles []string, candidate string) bool {
	for _, role := range roles {
		if strings.TrimSpace(role) == strings.TrimSpace(candidate) {
			return true
		}
	}
	return false
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
	var decidedAt *time.Time
	if !event.DecidedAt.IsZero() {
		value := event.DecidedAt
		decidedAt = &value
	}
	payload, err := json.Marshal(struct {
		ApprovalRequestID    uuid.UUID  `json:"approvalRequestId"`
		DecisionID           string     `json:"decisionId,omitempty"`
		DecisionVersion      int64      `json:"decisionVersion,omitempty"`
		Outcome              string     `json:"outcome,omitempty"`
		ApproverUserID       string     `json:"approverUserId,omitempty"`
		StepID               string     `json:"stepId,omitempty"`
		DelegationReference  string     `json:"delegationReference,omitempty"`
		DecidedAt            *time.Time `json:"decidedAt,omitempty"`
		PolicyID             uuid.UUID  `json:"policyId"`
		PolicyVersion        string     `json:"policyVersion"`
		SubjectID            string     `json:"subjectId"`
		SubjectVersion       int64      `json:"subjectVersion"`
		CandidateFingerprint string     `json:"candidateFingerprint"`
		ScopeIDs             []string   `json:"scopeIds"`
		DataClassification   string     `json:"dataClassification"`
	}{ApprovalRequestID: event.RequestID, DecisionID: decisionID, DecisionVersion: event.DecisionVersion, Outcome: event.Outcome, ApproverUserID: approverUserID, StepID: nullableUUIDString(event.StepID), DelegationReference: event.DelegationReference, DecidedAt: decidedAt, PolicyID: event.PolicyID, PolicyVersion: event.PolicyVersion, SubjectID: event.SubjectID, SubjectVersion: event.SubjectVersion, CandidateFingerprint: event.CandidateFingerprint, ScopeIDs: append([]string(nil), event.ScopeIDs...), DataClassification: event.DataClassification})
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

func nullableUUIDString(value uuid.UUID) string {
	if value == uuid.Nil {
		return ""
	}
	return value.String()
}

var _ workflow.ApprovalPolicyAuthorizer = evaluatorWorkflowAuthorizer{}
var _ workflow.ApprovalRequestAuthorizer = evaluatorWorkflowRequestAuthorizer{}
var _ workflow.PostgresApprovalPolicyAuditWriter = postgresWorkflowPolicyAuditWriter(nil)
var _ workflow.PostgresApprovalRequestAuditWriter = postgresWorkflowRequestAuditWriter(nil)
var _ workflow.PostgresApprovalEventWriter = postgresWorkflowEventWriter
