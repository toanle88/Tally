package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	ApprovalEscalationReasonOverdue             = "overdue"
	ApprovalEscalationReasonBlocked             = "blocked"
	ApprovalEscalationProcess                   = "workflow.approval-escalation"
	approvalEscalationBlockerMaxLength          = 500
	approvalEscalationFingerprintMaxLength      = 240
	approvalEscalationProcessReferenceMaxLength = 160
)

var (
	ErrApprovalPolicyEscalationNotEligible = errors.New("approval request is not eligible for escalation")
	ErrApprovalPolicyAlreadyEscalated      = errors.New("approval request step is already escalated")
)

type approvalEscalationPrincipal struct {
	actor            *Actor
	processReference string
}

func (principal approvalEscalationPrincipal) validate() error {
	if principal.actor == nil && strings.TrimSpace(principal.processReference) == "" {
		return fmt.Errorf("%w: escalation actor or process is required", ErrInvalidApprovalPolicyCommand)
	}
	if principal.actor != nil {
		return principal.actor.Validate()
	}
	if len(strings.TrimSpace(principal.processReference)) > approvalEscalationProcessReferenceMaxLength {
		return fmt.Errorf("%w: escalation process reference is too long", ErrInvalidApprovalPolicyCommand)
	}
	return nil
}

func (principal approvalEscalationPrincipal) identity() string {
	if principal.actor != nil {
		return "actor:" + principal.actor.UserID.String()
	}
	return "process:" + strings.TrimSpace(principal.processReference)
}

func (principal approvalEscalationPrincipal) commandIdentity(key string) string {
	return principal.identity() + ":" + key
}

func (principal approvalEscalationPrincipal) auditFields() (uuid.UUID, string, string) {
	if principal.actor != nil {
		return principal.actor.UserID, principal.actor.SubjectReference, ""
	}
	return uuid.Nil, "", strings.TrimSpace(principal.processReference)
}

func (service *ApprovalRequestService) EscalateApproval(ctx context.Context, actor Actor, command EscalateApprovalCommand) (ApprovalRequestCommandResult, error) {
	return service.escalateApproval(ctx, approvalEscalationPrincipal{actor: &actor}, command)
}

// EscalateApprovalAsProcess is used only by the approved workflow worker. The
// process still passes through policy, authorization, optimistic concurrency,
// audit, and outbox persistence; it is not a persistence shortcut.
func (service *ApprovalRequestService) EscalateApprovalAsProcess(ctx context.Context, processReference string, command EscalateApprovalCommand) (ApprovalRequestCommandResult, error) {
	return service.escalateApproval(ctx, approvalEscalationPrincipal{processReference: processReference}, command)
}

func (service *ApprovalRequestService) escalateApproval(ctx context.Context, principal approvalEscalationPrincipal, command EscalateApprovalCommand) (result ApprovalRequestCommandResult, returnErr error) {
	var durableAcquisition platformidempotency.DurableAcquisition
	defer func() {
		if service != nil && returnErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, returnErr)
		}
	}()
	if service == nil {
		return ApprovalRequestCommandResult{}, ErrInvalidApprovalPolicyService
	}
	if err := principal.validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	command.Reason = strings.ToLower(strings.TrimSpace(command.Reason))
	command.Blocker = strings.TrimSpace(command.Blocker)
	if err := command.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	commandFP := approvalEscalationCommandFingerprint(command)
	key := principal.commandIdentity(command.IdempotencyKey)
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		identity, err := approvalEscalationDurableIdentity(principal, command.IdempotencyKey)
		if err != nil {
			return ApprovalRequestCommandResult{}, err
		}
		durableIdentity = identity
		var acquisitionErr error
		durableAcquisition, acquisitionErr = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.EscalateOperationID, service.durable.Policy)
		if errors.Is(acquisitionErr, platformidempotency.ErrIdempotencyConflict) {
			return ApprovalRequestCommandResult{}, ErrApprovalPolicyIdempotencyConflict
		}
		if acquisitionErr != nil {
			return ApprovalRequestCommandResult{}, acquisitionErr
		}
		if durableAcquisition.Decision() == platformidempotency.DecisionReturn {
			switch durableAcquisition.Result().State() {
			case platformidempotency.StateFailed:
				return ApprovalRequestCommandResult{}, durableApprovalRequestFailureError(durableAcquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return ApprovalRequestCommandResult{}, ErrApprovalPolicyCommandInProgress
			case platformidempotency.StateEstablished:
				stored, decodeErr := decodeApprovalRequestCommandResult(durableAcquisition.Result().ResultBody())
				if decodeErr != nil {
					return ApprovalRequestCommandResult{}, decodeErr
				}
				stored.Replayed = true
				return stored, nil
			}
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != commandFP {
				return ApprovalRequestCommandResult{}, ErrApprovalPolicyIdempotencyConflict
			}
			stored := cloneApprovalRequestResult(previous.result)
			stored.Replayed = true
			return stored, nil
		}
	}

	request, err := service.repository.Get(ctx, command.RequestID)
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if !request.Version.Matches(*command.ExpectedVersion) {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyVersionConflict
	}
	if request.Status != ApprovalRequestStatusPending {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyEscalationNotEligible
	}
	if request.PolicyVersion != command.PolicyVersion || request.SubjectVersion != command.SubjectVersion || request.SubjectFingerprint != command.CandidateFingerprint {
		return ApprovalRequestCommandResult{}, fmt.Errorf("%w: escalation evidence does not match the immutable request", ErrApprovalPolicySubjectConflict)
	}
	if err := service.validateDecisionPolicy(ctx, request); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	step, err := actionableApprovalRequestStep(request, command.StepID)
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if step.Status != ApprovalRequestStatusPending {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyAlreadyEscalated
	}
	if !step.Escalation.Enabled || step.Escalation.AfterSeconds < 1 || len(step.Escalation.EscalateToRoles) == 0 {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyEscalationNotEligible
	}
	for _, escalation := range request.Escalations {
		if escalation.StepID == step.ID {
			return ApprovalRequestCommandResult{}, ErrApprovalPolicyAlreadyEscalated
		}
	}
	now := service.clock().UTC()
	if command.Reason == ApprovalEscalationReasonOverdue && (request.DueAt == nil || now.Before(request.DueAt.UTC())) {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyEscalationNotEligible
	}
	authorization, err := authorizeApprovalEscalation(ctx, service.authorizer, principal, request, step)
	if err != nil {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyAuthorizationUnavailable
	}
	if err := validateAuthorization(authorization, EscalateApprovalPermission, request.Scope.CanonicalScopeIDs()); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	nextVersion, err := request.Version.Advance()
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	requestBefore := cloneApprovalRequest(request)
	request.Version = nextVersion
	request.UpdatedAt = now
	for index := range request.Steps {
		if request.Steps[index].ID == step.ID {
			request.Steps[index].Status = ApprovalRequestStepStatusEscalated
			break
		}
	}
	actorUserID, actorSubjectReference, processReference := principal.auditFields()
	escalation := ApprovalEscalation{
		ID:                    command.EscalationID,
		StepID:                step.ID,
		Reason:                command.Reason,
		Blocker:               strings.TrimSpace(command.Blocker),
		ResponsibleRoles:      approvalRequestStepRoles(step),
		TargetRoles:           append([]string(nil), step.Escalation.EscalateToRoles...),
		ActorUserID:           actorUserID,
		ActorSubjectReference: actorSubjectReference,
		ProcessReference:      processReference,
		PolicyID:              request.PolicyID,
		PolicyVersion:         request.PolicyVersion,
		SubjectVersion:        request.SubjectVersion,
		CandidateFingerprint:  request.SubjectFingerprint,
		DueAt:                 cloneTime(request.DueAt),
		OccurredAt:            now,
		CorrelationID:         command.CorrelationID,
		CausationID:           command.CausationID,
	}
	request.Escalations = append(request.Escalations, escalation)
	if err := request.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	auditRecord := ApprovalRequestAuditRecord{
		ApprovalRequestID: request.ID, ApprovalEscalationID: escalation.ID,
		ActorUserID: actorUserID, ActorAuthenticationSubjectRef: actorSubjectReference,
		Action: "EscalateApproval", ScopeIDs: request.Scope.CanonicalScopeIDs(),
		Reason: command.Reason, StepID: step.ID, Permission: authorization.Permission,
		PolicyReference: authorization.PolicyReference, DecisionReference: authorization.DecisionReference,
		PolicyVersion: command.PolicyVersion, RevisionVersion: request.Version.Value(), SubjectVersion: command.SubjectVersion,
		BeforeFingerprint: fingerprint(requestBefore), AfterFingerprint: fingerprint(request),
		CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	event := ApprovalEvent{
		Type: ApprovalEscalatedEventType, RequestID: request.ID, EscalationID: escalation.ID,
		StepID: step.ID, EscalationReason: command.Reason, EscalationBlocker: strings.TrimSpace(command.Blocker),
		EscalationTargetRoles: append([]string(nil), step.Escalation.EscalateToRoles...),
		AggregateVersion:      request.Version.Value(), PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion,
		SubjectID: request.SubjectID, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint,
		ScopeIDs: request.Scope.CanonicalScopeIDs(), DataClassification: request.DataClassification,
		OccurredAt: now, EscalatedAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	escalationResult := cloneApprovalEscalation(escalation)
	result = ApprovalRequestCommandResult{Request: cloneApprovalRequest(request), Escalation: &escalationResult}
	if service.durable != nil {
		committer, ok := service.repository.(DurableApprovalRequestRepository)
		if !ok {
			return ApprovalRequestCommandResult{}, ErrInvalidApprovalPolicyService
		}
		aggregateID := request.ID
		resultBuilder := func(auditReference uuid.UUID) (platformidempotency.CommandResultMetadata, error) {
			result.Escalation.AuditReference = auditReference
			for index := range result.Request.Escalations {
				if result.Request.Escalations[index].ID == result.Escalation.ID {
					result.Request.Escalations[index].AuditReference = auditReference
					break
				}
			}
			body, marshalErr := jsonMarshalApprovalRequestResult(result)
			if marshalErr != nil {
				return platformidempotency.CommandResultMetadata{}, marshalErr
			}
			status := 200
			return platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.EscalateOperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		}
		metadata, builderErr := resultBuilder(uuid.Nil)
		if builderErr != nil {
			return ApprovalRequestCommandResult{}, builderErr
		}
		auditReference, saveErr := committer.SaveWithIdempotency(ctx, request, &requestBefore, auditRecord, event, DurableApprovalRequestCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata, ResultBuilder: resultBuilder})
		if saveErr != nil {
			return ApprovalRequestCommandResult{}, saveErr
		}
		result.Escalation.AuditReference = auditReference
		for index := range result.Request.Escalations {
			if result.Request.Escalations[index].ID == result.Escalation.ID {
				result.Request.Escalations[index].AuditReference = auditReference
				break
			}
		}
	} else {
		auditReference, saveErr := service.repository.Save(ctx, request, &requestBefore, auditRecord, event)
		if saveErr != nil {
			return ApprovalRequestCommandResult{}, saveErr
		}
		result.Escalation.AuditReference = auditReference
		for index := range result.Request.Escalations {
			if result.Request.Escalations[index].ID == result.Escalation.ID {
				result.Request.Escalations[index].AuditReference = auditReference
				break
			}
		}
	}
	if service.durable == nil {
		service.idempotency[key] = storedApprovalRequestResult{fingerprint: commandFP, result: cloneApprovalRequestResult(result)}
	}
	return result, nil
}

func authorizeApprovalEscalation(ctx context.Context, authorizer ApprovalRequestAuthorizer, principal approvalEscalationPrincipal, request ApprovalRequest, step ApprovalRequestStep) (AuthorizationDecision, error) {
	if principal.actor != nil {
		escalationAuthorizer, ok := authorizer.(ApprovalRequestEscalationAuthorizer)
		if !ok {
			return AuthorizationDecision{}, ErrApprovalPolicyAuthorizationUnavailable
		}
		return escalationAuthorizer.AuthorizeApprovalEscalation(ctx, *principal.actor, request, step)
	}
	processAuthorizer, ok := authorizer.(ApprovalRequestEscalationProcessAuthorizer)
	if !ok {
		return AuthorizationDecision{}, ErrApprovalPolicyAuthorizationUnavailable
	}
	return processAuthorizer.AuthorizeApprovalEscalationProcess(ctx, principal.processReference, request, step)
}

func approvalEscalationDurableIdentity(principal approvalEscalationPrincipal, key string) (platformidempotency.IdempotencyIdentity, error) {
	scopeKey := struct {
		Module    string `json:"module"`
		Principal string `json:"principal"`
		Operation string `json:"operation"`
	}{Module: "workflow", Principal: principal.identity(), Operation: "escalate"}
	return platformidempotency.NewOpaqueIdentity(string(mustJSON(scopeKey)), key)
}

func approvalEscalationCommandFingerprint(command EscalateApprovalCommand) string {
	command.IdempotencyKey = ""
	command.CorrelationID = ""
	command.CausationID = ""
	return commandFingerprint(command)
}

func approvalRequestStepRoles(step ApprovalRequestStep) []string {
	roles := make([]string, 0, len(step.RequiredRoles))
	seen := make(map[string]struct{}, len(step.RequiredRoles))
	for _, requirement := range step.RequiredRoles {
		role := strings.TrimSpace(requirement.Role)
		if role == "" {
			continue
		}
		if _, exists := seen[role]; exists {
			continue
		}
		seen[role] = struct{}{}
		roles = append(roles, role)
	}
	return roles
}

func cloneApprovalEscalation(escalation ApprovalEscalation) ApprovalEscalation {
	escalation.ResponsibleRoles = append([]string(nil), escalation.ResponsibleRoles...)
	escalation.TargetRoles = append([]string(nil), escalation.TargetRoles...)
	escalation.DueAt = cloneTime(escalation.DueAt)
	return escalation
}

func jsonMarshalApprovalRequestResult(result ApprovalRequestCommandResult) ([]byte, error) {
	return json.Marshal(result)
}

func mustJSON(value any) []byte {
	data, _ := json.Marshal(value)
	return data
}
