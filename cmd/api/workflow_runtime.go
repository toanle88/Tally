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
	clock       func() time.Time
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalDecision(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest) (workflow.AuthorizationDecision, error) {
	roleIDs, err := authorizer.approvalRoleIDs(ctx, actor, false)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	return authorizer.evaluateRequest(ctx, actor, request, workflow.CreateApprovalRequestsPermission, roleIDs)
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalRequestDecision(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, step workflow.ApprovalRequestStep, delegationReference string) (workflow.AuthorizationDecision, error) {
	return authorizer.AuthorizeApprovalRequestDecisionWithDelegation(ctx, actor, request, step, delegationReference, nil)
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalRequestDecisionWithDelegation(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, step workflow.ApprovalRequestStep, delegationReference string, delegation *workflow.Delegation) (workflow.AuthorizationDecision, error) {
	roleIDs, err := authorizer.approvalRoleIDs(ctx, actor, true)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	decision, err := authorizer.evaluateRequest(ctx, actor, request, workflow.DecideApprovalRequestsPermission, roleIDs)
	if err != nil || !decision.Allowed {
		return decision, err
	}
	if authorizer.users != nil && authorizer.roles != nil {
		eligible, err := authorizer.actorEligibleForStep(ctx, actor, request, step, delegationReference, delegation)
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

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalEscalation(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, step workflow.ApprovalRequestStep) (workflow.AuthorizationDecision, error) {
	roleIDs, err := authorizer.approvalRoleIDs(ctx, actor, true)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	decision, err := authorizer.evaluateRequest(ctx, actor, request, workflow.EscalateApprovalPermission, roleIDs)
	if err != nil || !decision.Allowed || authorizer.users == nil || authorizer.roles == nil {
		return decision, err
	}
	eligible, err := authorizer.actorEligibleForStep(ctx, actor, request, step, "", nil)
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyApprovalUnavailable
	}
	if !eligible {
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.EscalateApprovalPermission, DecisionReference: decision.DecisionReference, ReasonCode: "approval-step-role-required"}, nil
	}
	return decision, nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalEscalationProcess(_ context.Context, processReference string, request workflow.ApprovalRequest, _ workflow.ApprovalRequestStep) (workflow.AuthorizationDecision, error) {
	if strings.TrimSpace(processReference) != workflow.ApprovalEscalationProcess {
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.EscalateApprovalPermission, DecisionReference: uuid.New(), ReasonCode: "process-not-approved"}, nil
	}
	return workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.EscalateApprovalPermission, ApprovedScopeIDs: request.Scope.CanonicalScopeIDs(), PolicyReference: workflow.ApprovalEscalationProcess, DecisionReference: uuid.New()}, nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) AuthorizeApprovalDelegation(ctx context.Context, actor workflow.Actor, command workflow.DelegateApprovalCommand, policy workflow.ApprovalPolicy) (workflow.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return workflow.AuthorizationDecision{}, workflow.ErrDelegationAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{
		ActorID:            actor.UserID,
		Permission:         workflow.DelegateApprovalPermission,
		RequestedScopeIDs:  command.Scope.CanonicalScopeIDs(),
		DataClassification: "internal",
		SubjectActorIDs:    []uuid.UUID{command.DelegateUserID},
	})
	if err != nil {
		return workflow.AuthorizationDecision{}, workflow.ErrDelegationAuthorizationUnavailable
	}
	if !decision.Allowed {
		return workflowAuthorizationDecision(decision), nil
	}
	if authorizer.users == nil || authorizer.roles == nil {
		return workflowAuthorizationDecision(decision), nil
	}
	delegate, err := authorizer.users.Get(ctx, command.DelegateUserID)
	if err != nil || delegate.Status != identity.UserStatusActive {
		return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.DelegateApprovalPermission, DecisionReference: decision.DecisionReference, ReasonCode: "delegate-inactive"}, nil
	}
	requestedScopes := command.Scope.CanonicalScopeIDs()
	for _, assignment := range delegate.Assignments {
		if !approvalAssignmentCoversScopes(assignment.Scopes, requestedScopes) {
			continue
		}
		role, roleErr := authorizer.roles.Get(ctx, assignment.RoleID)
		if roleErr != nil || role.Status != identity.RoleStatusActive {
			continue
		}
		for _, step := range policy.Steps {
			if !step.Delegation.Enabled || !containsApprovalRole(step.Delegation.DelegateToRoles, role.Name) {
				continue
			}
			for _, requirement := range step.RequiredRoles {
				if requirement.AllowDelegation {
					return workflowAuthorizationDecision(decision), nil
				}
			}
		}
	}
	return workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.DelegateApprovalPermission, DecisionReference: decision.DecisionReference, ReasonCode: "delegate-role-required"}, nil
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

func (authorizer evaluatorWorkflowRequestAuthorizer) actorEligibleForStep(ctx context.Context, actor workflow.Actor, request workflow.ApprovalRequest, step workflow.ApprovalRequestStep, delegationReference string, delegation *workflow.Delegation) (bool, error) {
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
	if delegation != nil {
		if delegation.DelegateUserID != actor.UserID || delegation.PolicyID != request.PolicyID || delegation.PolicyVersion != request.PolicyVersion || !delegation.ActiveAt(authorizer.now()) || !delegation.Scope.Contains(request.Scope.CanonicalScopeIDs()) {
			return false, nil
		}
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
			if role.Name == requirement.Role && (delegation == nil || delegation.DelegateUserID == actor.UserID) {
				return true, nil
			}
			if requirement.AllowDelegation && step.Delegation.Enabled && strings.TrimSpace(delegationReference) != "" && delegation != nil && containsApprovalRole(step.Delegation.DelegateToRoles, role.Name) {
				return true, nil
			}
			// Preserve the pre-delegation adapter contract for callers that do
			// not yet bind a delegation reader. Production decision commands
			// always pass the resolved aggregate above.
			if requirement.AllowDelegation && step.Delegation.Enabled && strings.TrimSpace(delegationReference) != "" && delegation == nil && containsApprovalRole(step.Delegation.DelegateToRoles, role.Name) {
				return true, nil
			}
		}
	}
	return false, nil
}

func (authorizer evaluatorWorkflowRequestAuthorizer) now() time.Time {
	if authorizer.clock != nil {
		return authorizer.clock().UTC()
	}
	return time.Now().UTC()
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
			ApprovalEscalationID:          record.ApprovalEscalationID,
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
	escalationID := ""
	if event.EscalationID != uuid.Nil {
		escalationID = event.EscalationID.String()
	}
	var decidedAt *time.Time
	if !event.DecidedAt.IsZero() {
		value := event.DecidedAt
		decidedAt = &value
	}
	var escalatedAt *time.Time
	if !event.EscalatedAt.IsZero() {
		value := event.EscalatedAt
		escalatedAt = &value
	}
	payload, err := json.Marshal(struct {
		ApprovalRequestID     uuid.UUID  `json:"approvalRequestId"`
		DecisionID            string     `json:"decisionId,omitempty"`
		EscalationID          string     `json:"escalationId,omitempty"`
		DecisionVersion       int64      `json:"decisionVersion,omitempty"`
		Outcome               string     `json:"outcome,omitempty"`
		EscalationReason      string     `json:"escalationReason,omitempty"`
		EscalationBlocker     string     `json:"escalationBlocker,omitempty"`
		EscalationTargetRoles []string   `json:"escalationTargetRoles,omitempty"`
		ApproverUserID        string     `json:"approverUserId,omitempty"`
		StepID                string     `json:"stepId,omitempty"`
		DelegationReference   string     `json:"delegationReference,omitempty"`
		DecidedAt             *time.Time `json:"decidedAt,omitempty"`
		EscalatedAt           *time.Time `json:"escalatedAt,omitempty"`
		PolicyID              uuid.UUID  `json:"policyId"`
		PolicyVersion         string     `json:"policyVersion"`
		SubjectID             string     `json:"subjectId"`
		SubjectVersion        int64      `json:"subjectVersion"`
		CandidateFingerprint  string     `json:"candidateFingerprint"`
		ScopeIDs              []string   `json:"scopeIds"`
		DataClassification    string     `json:"dataClassification"`
	}{ApprovalRequestID: event.RequestID, DecisionID: decisionID, EscalationID: escalationID, DecisionVersion: event.DecisionVersion, Outcome: event.Outcome, EscalationReason: event.EscalationReason, EscalationBlocker: event.EscalationBlocker, EscalationTargetRoles: append([]string(nil), event.EscalationTargetRoles...), ApproverUserID: approverUserID, StepID: nullableUUIDString(event.StepID), DelegationReference: event.DelegationReference, DecidedAt: decidedAt, EscalatedAt: escalatedAt, PolicyID: event.PolicyID, PolicyVersion: event.PolicyVersion, SubjectID: event.SubjectID, SubjectVersion: event.SubjectVersion, CandidateFingerprint: event.CandidateFingerprint, ScopeIDs: append([]string(nil), event.ScopeIDs...), DataClassification: event.DataClassification})
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

func postgresWorkflowDelegationAuditWriter(auditWriter identity.PostgresAuditWriter) workflow.PostgresDelegationAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record workflow.DelegationAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, workflow.ErrDelegationAuditUnavailable
		}
		to := record.EffectiveTo
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        record.DelegationID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorAuthenticationSubjectRef,
			Action:                        "DelegateApproval",
			ScopeIDs:                      append([]string(nil), record.ScopeIDs...),
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyID.String(),
			PolicyVersion:                 record.PolicyVersion,
			DecisionReference:             record.DecisionReference,
			ApproverUserID:                record.DelegateUserID,
			EffectiveFrom:                 record.EffectiveFrom,
			EffectiveTo:                   &to,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
}

func postgresWorkflowDelegationEventWriter(ctx context.Context, tx pgx.Tx, event workflow.DelegationEvent) error {
	payload, err := json.Marshal(struct {
		DelegationID    uuid.UUID `json:"delegationId"`
		DelegatorUserID uuid.UUID `json:"delegatorUserId"`
		DelegateUserID  uuid.UUID `json:"delegateUserId"`
		PolicyID        uuid.UUID `json:"policyId"`
		PolicyVersion   string    `json:"policyVersion"`
		ScopeIDs        []string  `json:"scopeIds"`
		EffectiveFrom   time.Time `json:"effectiveFrom"`
		EffectiveTo     time.Time `json:"effectiveTo"`
	}{DelegationID: event.DelegationID, DelegatorUserID: event.DelegatorUserID, DelegateUserID: event.DelegateUserID, PolicyID: event.PolicyID, PolicyVersion: event.PolicyVersion, ScopeIDs: append([]string(nil), event.ScopeIDs...), EffectiveFrom: event.EffectiveFrom, EffectiveTo: event.EffectiveTo})
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
		AggregateID:        event.DelegationID.String(),
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
var _ workflow.DelegationAuthorizer = evaluatorWorkflowRequestAuthorizer{}
var _ workflow.PostgresApprovalPolicyAuditWriter = postgresWorkflowPolicyAuditWriter(nil)
var _ workflow.PostgresApprovalRequestAuditWriter = postgresWorkflowRequestAuditWriter(nil)
var _ workflow.PostgresApprovalEventWriter = postgresWorkflowEventWriter
var _ workflow.PostgresDelegationAuditWriter = postgresWorkflowDelegationAuditWriter(nil)
var _ workflow.PostgresDelegationEventWriter = postgresWorkflowDelegationEventWriter
