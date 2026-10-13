package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/workflow"
)

const workflowMaintainApprovalPoliciesRoute = "/api/v1/approvals/configuration/maintain-approval-policies"

// WfaMaintainApprovalPolicies owns the typed HTTP boundary for policy-family
// changes. It deliberately exposes only the command/result route; policy
// reads remain owned by a future authorized workflow projection.
func (handler IdentityHandler) WfaMaintainApprovalPolicies(ctx context.Context, request *generated.WfaMaintainApprovalPoliciesCommandRequest, params generated.WfaMaintainApprovalPoliciesParams) (generated.WfaMaintainApprovalPoliciesRes, error) {
	correlationID := workflowCorrelation(params)
	if handler.ApprovalPolicyService == nil {
		return workflowPolicyProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-policy service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return workflowPolicyProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return workflowPolicyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval-policy command requires a command identifier.", correlationID), nil
	}
	command, err := workflowCommandFromGenerated(request, params, correlationID)
	if err != nil {
		if errors.Is(err, workflow.ErrApprovalPolicyVersionConflict) {
			return mapWorkflowPolicyError(err, correlationID), nil
		}
		return workflowPolicyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval-policy command is malformed.", correlationID), nil
	}
	result, err := handler.ApprovalPolicyService.Execute(ctx, workflow.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapWorkflowPolicyError(err, correlationID), nil
	}
	return establishedWorkflowPolicyResult(result, correlationID), nil
}

func workflowCorrelation(params generated.WfaMaintainApprovalPoliciesParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func workflowCommandFromGenerated(request *generated.WfaMaintainApprovalPoliciesCommandRequest, params generated.WfaMaintainApprovalPoliciesParams, correlationID uuid.UUID) (workflow.ApprovalPolicyCommand, error) {
	data := request.Data
	policyID := uuid.Nil
	if data.PolicyId.Set {
		policyID = uuid.UUID(data.PolicyId.Value)
		if policyID == uuid.Nil {
			return workflow.ApprovalPolicyCommand{}, errors.New("policy id is nil")
		}
	}
	policyVersion := ""
	if data.PolicyVersion.Set {
		policyVersion = data.PolicyVersion.Value
	}

	var expectedVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return workflow.ApprovalPolicyCommand{}, err
		}
		expectedVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return workflow.ApprovalPolicyCommand{}, err
		}
		if expectedVersion != nil && expectedVersion.Value() != headerVersion.Value() {
			return workflow.ApprovalPolicyCommand{}, workflow.ErrApprovalPolicyVersionConflict
		}
		expectedVersion = &headerVersion
	}

	applicability := workflow.ApprovalApplicability{
		SourceContext:       data.Applicability.SourceContext,
		SubjectType:         data.Applicability.SubjectType,
		Operation:           data.Applicability.Operation,
		TransactionTypes:    append([]string(nil), data.Applicability.TransactionTypes...),
		DataClassifications: append([]string(nil), data.Applicability.DataClassifications...),
		Dimensions:          map[string]string{},
	}
	for _, currency := range data.Applicability.Currencies {
		applicability.Currencies = append(applicability.Currencies, string(currency))
	}
	if data.Applicability.Dimensions.Set {
		for key, value := range data.Applicability.Dimensions.Value {
			applicability.Dimensions[key] = value
		}
	}
	if len(applicability.Dimensions) == 0 {
		applicability.Dimensions = nil
	}

	scope := workflow.ApprovalPolicyScope{
		ScopeIDs: append([]string(nil), data.Scope.ScopeIds...),
	}
	scope.AccountingScopeIDs = workflowUUIDs(data.Scope.AccountingScopeIds)
	scope.LegalEntityIDs = workflowUUIDs(data.Scope.LegalEntityIds)
	scope.BusinessUnitIDs = workflowUUIDs(data.Scope.BusinessUnitIds)
	scope.SegmentIDs = workflowUUIDs(data.Scope.SegmentIds)
	scope.AccountIDs = workflowUUIDs(data.Scope.AccountIds)
	scope.FiscalPeriodIDs = workflowUUIDs(data.Scope.FiscalPeriodIds)

	rules := make([]workflow.ApprovalRule, len(data.Rules))
	for index, rule := range data.Rules {
		threshold, err := workflowThresholdFromGenerated(rule.Threshold)
		if err != nil {
			return workflow.ApprovalPolicyCommand{}, err
		}
		rules[index] = workflow.ApprovalRule{
			ID:            uuid.UUID(rule.ID),
			Name:          rule.Name.Or(""),
			Threshold:     threshold,
			RequiredRoles: workflowRolesFromGenerated(rule.RequiredRoles),
			Delegation:    workflowDelegationFromGenerated(rule.Delegation),
			Escalation:    workflowEscalationFromGenerated(rule.Escalation),
		}
	}
	steps := make([]workflow.ApprovalStepDefinition, len(data.Steps))
	for index, step := range data.Steps {
		steps[index] = workflow.ApprovalStepDefinition{
			ID:               uuid.UUID(step.ID),
			Sequence:         step.Sequence,
			Name:             step.Name,
			RequiredRoles:    workflowRolesFromGenerated(step.RequiredRoles),
			MinimumApprovals: step.MinimumApprovals,
			Delegation:       workflowDelegationFromGenerated(step.Delegation),
			Escalation:       workflowEscalationFromGenerated(step.Escalation),
		}
	}

	var effectiveTo *time.Time
	if data.EffectiveTo.Set {
		value := data.EffectiveTo.Value.UTC()
		effectiveTo = &value
	}
	var approval *workflow.ApprovalDecisionReference
	if data.Approval.Set {
		value := data.Approval.Value
		approval = &workflow.ApprovalDecisionReference{
			ApprovalRequestID:    uuid.UUID(value.ApprovalRequestId),
			DecisionID:           uuid.UUID(value.DecisionId),
			PolicyVersion:        value.PolicyVersion,
			DecisionVersion:      int64(value.DecisionVersion),
			SubjectVersion:       int64(value.SubjectVersion),
			CandidateFingerprint: value.CandidateFingerprint,
			ApproverUserID:       uuid.UUID(value.ApproverUserId),
		}
	}

	submitForApproval := data.SubmitForApproval.Set && data.SubmitForApproval.Value
	reason := data.Reason.Or("")
	impact := data.DependentWorkflowImpact.Or("")
	return workflow.ApprovalPolicyCommand{
		Action:                  string(data.Action),
		PolicyID:                policyID,
		Code:                    data.Code,
		Name:                    data.Name,
		PolicyVersion:           policyVersion,
		Applicability:           applicability,
		Scope:                   scope,
		Rules:                   rules,
		Steps:                   steps,
		EffectiveFrom:           data.EffectiveFrom.UTC(),
		EffectiveTo:             effectiveTo,
		Approval:                approval,
		ExpectedVersion:         expectedVersion,
		SubmitForApproval:       submitForApproval,
		Reason:                  reason,
		DependentWorkflowImpact: impact,
		IdempotencyKey:          params.IdempotencyKey,
		CorrelationID:           correlationID.String(),
		CausationID:             uuid.UUID(request.CommandId).String(),
	}, nil
}

func workflowUUIDs(values []generated.UUID) []uuid.UUID {
	if len(values) == 0 {
		return nil
	}
	result := make([]uuid.UUID, len(values))
	for index, value := range values {
		result[index] = uuid.UUID(value)
	}
	return result
}

func workflowThresholdFromGenerated(value generated.WfaApprovalRuleThreshold) (workflow.AmountThreshold, error) {
	minimum, err := workflowMoneyPointer(value.Minimum)
	if err != nil {
		return workflow.AmountThreshold{}, err
	}
	maximum, err := workflowMoneyPointer(value.Maximum)
	if err != nil {
		return workflow.AmountThreshold{}, err
	}
	return workflow.AmountThreshold{Minimum: minimum, Maximum: maximum, Currency: string(value.Currency)}, nil
}

func workflowMoneyPointer(value generated.OptMoney) (*decimal.Decimal, error) {
	if !value.Set {
		return nil, nil
	}
	parsed, err := decimal.NewFromString(string(value.Value))
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func workflowRolesFromGenerated(values []generated.WfaRoleRequirement) []workflow.RoleRequirement {
	result := make([]workflow.RoleRequirement, len(values))
	for index, value := range values {
		result[index] = workflow.RoleRequirement{Role: value.Role, MinimumApprovals: value.MinimumApprovals, AllowDelegation: value.AllowDelegation}
	}
	return result
}

func workflowDelegationFromGenerated(value generated.WfaDelegationConfiguration) workflow.DelegationConfiguration {
	return workflow.DelegationConfiguration{Enabled: value.Enabled, DelegateToRoles: append([]string(nil), value.DelegateToRoles...), MaximumDepth: value.MaximumDepth.Or(0), RequiresIndependentApproval: value.RequiresIndependentApproval}
}

func workflowEscalationFromGenerated(value generated.WfaEscalationConfiguration) workflow.EscalationConfiguration {
	return workflow.EscalationConfiguration{Enabled: value.Enabled, AfterSeconds: int64(value.AfterSeconds.Or(0)), EscalateToRoles: append([]string(nil), value.EscalateToRoles...)}
}

func establishedWorkflowPolicyResult(result workflow.ApprovalPolicyCommandResult, correlationID uuid.UUID) *generated.WfaMaintainApprovalPoliciesEstablishedResult {
	policy := result.Policy
	projection := generated.WfaApprovalPolicyProjection{
		ID:                generated.UUID(policy.ID),
		Code:              policy.Code,
		Name:              policy.Name,
		PolicyVersion:     policy.PolicyVersion,
		Status:            generated.WfaApprovalPolicyProjectionStatus(policy.Status),
		Applicability:     generatedWorkflowApplicability(policy.Applicability),
		Scope:             generatedWorkflowScope(policy.Scope),
		Rules:             generatedWorkflowRules(policy.Rules),
		Steps:             generatedWorkflowSteps(policy.Steps),
		EffectiveFrom:     policy.EffectiveFrom,
		ActivationOutcome: policy.ActivationOutcome,
		CandidateFingerprint: policy.CandidateFingerprint(),
		AggregateVersion:  generated.NewOptInt(int(policy.Version.Value())),
	}
	if policy.EffectiveTo != nil {
		projection.EffectiveTo = generated.NewOptDateTime(*policy.EffectiveTo)
	}
	if policy.Approval != nil {
		projection.Approval = generated.NewOptIamApprovalDecisionReference(generatedWorkflowApproval(*policy.Approval))
	}
	if policy.ApprovalRequestID != uuid.Nil {
		projection.ApprovalRequestId = generated.NewOptUUID(generated.UUID(policy.ApprovalRequestID))
	}
	data := generated.WfaMaintainApprovalPoliciesResultData{
		Policy:                  projection,
		PolicyVersion:           policy.PolicyVersion,
		Scope:                   generatedWorkflowScope(policy.Scope),
		Thresholds:              generatedWorkflowThresholds(policy.Rules),
		Roles:                   generatedWorkflowRoles(policy),
		EffectiveFrom:           generated.NewOptDateTime(policy.EffectiveFrom),
		ApprovalStatus:          result.ApprovalStatus,
		ActivationOutcome:       result.ActivationOutcome,
		DependentWorkflowImpact: result.DependentWorkflowImpact,
		Replayed:                generated.NewOptBool(result.Replayed),
	}
	if policy.EffectiveTo != nil {
		data.EffectiveTo = generated.NewOptDateTime(*policy.EffectiveTo)
	}
	if policy.ApprovalRequestID != uuid.Nil {
		data.ApprovalRequestId = generated.NewOptUUID(generated.UUID(policy.ApprovalRequestID))
	}
	if result.DecisionReference != uuid.Nil {
		data.DecisionReference = generated.NewOptUUID(generated.UUID(result.DecisionReference))
	}
	if result.AuditReference != uuid.Nil {
		data.AuditReference = generated.NewOptUUID(generated.UUID(result.AuditReference))
	}
	return &generated.WfaMaintainApprovalPoliciesEstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(policy.ID),
		AggregateVersion: int(policy.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: workflowMaintainApprovalPoliciesRoute},
		Data:             data,
	}
}

func generatedWorkflowApplicability(value workflow.ApprovalApplicability) generated.WfaApprovalApplicability {
	result := generated.WfaApprovalApplicability{
		SourceContext:       value.SourceContext,
		SubjectType:         value.SubjectType,
		Operation:           value.Operation,
		TransactionTypes:    append([]string(nil), value.TransactionTypes...),
		DataClassifications: append([]string(nil), value.DataClassifications...),
	}
	for _, currency := range value.Currencies {
		result.Currencies = append(result.Currencies, generated.CurrencyCode(currency))
	}
	if len(value.Dimensions) > 0 {
		dimensions := generated.WfaApprovalApplicabilityDimensions{}
		for key, item := range value.Dimensions {
			dimensions[key] = item
		}
		result.Dimensions = generated.NewOptWfaApprovalApplicabilityDimensions(dimensions)
	}
	return result
}

func generatedWorkflowScope(value workflow.ApprovalPolicyScope) generated.WfaApprovalScope {
	return generated.WfaApprovalScope{
		ScopeIds:           append([]string(nil), value.ScopeIDs...),
		AccountingScopeIds: generatedUUIDs(value.AccountingScopeIDs),
		LegalEntityIds:     generatedUUIDs(value.LegalEntityIDs),
		BusinessUnitIds:    generatedUUIDs(value.BusinessUnitIDs),
		SegmentIds:         generatedUUIDs(value.SegmentIDs),
		AccountIds:         generatedUUIDs(value.AccountIDs),
		FiscalPeriodIds:    generatedUUIDs(value.FiscalPeriodIDs),
	}
}

func generatedUUIDs(values []uuid.UUID) []generated.UUID {
	result := make([]generated.UUID, len(values))
	for index, value := range values {
		result[index] = generated.UUID(value)
	}
	return result
}

func generatedWorkflowRules(values []workflow.ApprovalRule) []generated.WfaApprovalRule {
	result := make([]generated.WfaApprovalRule, len(values))
	for index, value := range values {
		minimum := value.Threshold.Minimum
		maximum := value.Threshold.Maximum
		currency := value.Threshold.Currency
		if minimum == nil {
			minimum = value.MinAmount
		}
		if maximum == nil {
			maximum = value.MaxAmount
		}
		if currency == "" {
			currency = value.Currency
		}
		threshold := generated.WfaApprovalRuleThreshold{Currency: generated.CurrencyCode(currency)}
		if minimum != nil {
			threshold.Minimum = generated.NewOptMoney(generated.Money(minimum.String()))
		}
		if maximum != nil {
			threshold.Maximum = generated.NewOptMoney(generated.Money(maximum.String()))
		}
		result[index] = generated.WfaApprovalRule{ID: generated.UUID(value.ID), Threshold: threshold, RequiredRoles: generatedWorkflowRolesForRequirements(value.RequiredRoles), Delegation: generatedWorkflowDelegation(value.Delegation), Escalation: generatedWorkflowEscalation(value.Escalation)}
		if value.Name != "" {
			result[index].Name = generated.NewOptString(value.Name)
		}
	}
	return result
}

func generatedWorkflowSteps(values []workflow.ApprovalStepDefinition) []generated.WfaApprovalStepDefinition {
	result := make([]generated.WfaApprovalStepDefinition, len(values))
	for index, value := range values {
		result[index] = generated.WfaApprovalStepDefinition{ID: generated.UUID(value.ID), Sequence: value.Sequence, Name: value.Name, RequiredRoles: generatedWorkflowRolesForRequirements(value.RequiredRoles), MinimumApprovals: value.MinimumApprovals, Delegation: generatedWorkflowDelegation(value.Delegation), Escalation: generatedWorkflowEscalation(value.Escalation)}
	}
	return result
}

func generatedWorkflowRolesForRequirements(values []workflow.RoleRequirement) []generated.WfaRoleRequirement {
	result := make([]generated.WfaRoleRequirement, len(values))
	for index, value := range values {
		result[index] = generated.WfaRoleRequirement{Role: value.Role, MinimumApprovals: value.MinimumApprovals, AllowDelegation: value.AllowDelegation}
	}
	return result
}

func generatedWorkflowDelegation(value workflow.DelegationConfiguration) generated.WfaDelegationConfiguration {
	result := generated.WfaDelegationConfiguration{Enabled: value.Enabled, DelegateToRoles: append([]string(nil), value.DelegateToRoles...), RequiresIndependentApproval: value.RequiresIndependentApproval}
	if value.MaximumDepth != 0 {
		result.MaximumDepth = generated.NewOptInt(value.MaximumDepth)
	}
	return result
}

func generatedWorkflowEscalation(value workflow.EscalationConfiguration) generated.WfaEscalationConfiguration {
	result := generated.WfaEscalationConfiguration{Enabled: value.Enabled, EscalateToRoles: append([]string(nil), value.EscalateToRoles...)}
	if value.AfterSeconds != 0 {
		result.AfterSeconds = generated.NewOptInt(int(value.AfterSeconds))
	}
	return result
}

func generatedWorkflowApproval(value workflow.ApprovalDecisionReference) generated.IamApprovalDecisionReference {
	return generated.IamApprovalDecisionReference{ApprovalRequestId: generated.UUID(value.ApprovalRequestID), DecisionId: generated.UUID(value.DecisionID), PolicyVersion: value.PolicyVersion, DecisionVersion: int(value.DecisionVersion), SubjectVersion: int(value.SubjectVersion), CandidateFingerprint: value.CandidateFingerprint, ApproverUserId: generated.UUID(value.ApproverUserID)}
}

func generatedWorkflowThresholds(values []workflow.ApprovalRule) []generated.Money {
	result := make([]generated.Money, 0, len(values))
	for _, rule := range values {
		minimum := rule.Threshold.Minimum
		maximum := rule.Threshold.Maximum
		if minimum == nil {
			minimum = rule.MinAmount
		}
		if maximum == nil {
			maximum = rule.MaxAmount
		}
		if minimum != nil {
			result = append(result, generated.Money(minimum.String()))
		}
		if maximum != nil {
			result = append(result, generated.Money(maximum.String()))
		}
	}
	return result
}

func generatedWorkflowRoles(policy workflow.ApprovalPolicy) []string {
	seen := map[string]struct{}{}
	var result []string
	for _, rule := range policy.Rules {
		for _, role := range rule.RequiredRoles {
			if _, ok := seen[role.Role]; !ok {
				seen[role.Role] = struct{}{}
				result = append(result, role.Role)
			}
		}
	}
	for _, step := range policy.Steps {
		for _, role := range step.RequiredRoles {
			if _, ok := seen[role.Role]; !ok {
				seen[role.Role] = struct{}{}
				result = append(result, role.Role)
			}
		}
	}
	return result
}

func mapWorkflowPolicyError(err error, correlationID uuid.UUID) generated.WfaMaintainApprovalPoliciesRes {
	switch {
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationDenied), errors.Is(err, workflow.ErrApprovalPolicySelfApproval):
		status, code, detail := http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested policy change is outside the administering actor's authority."
		if errors.Is(err, workflow.ErrApprovalPolicySelfApproval) {
			status, code, detail = http.StatusUnprocessableEntity, "SELF_APPROVAL_DENIED", "The policy requester cannot approve the same policy change."
		}
		return workflowPolicyProblem(status, code, detail, correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationStale):
		return workflowPolicyProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyAuthorizationUnavailable), errors.Is(err, workflow.ErrApprovalPolicyAuditUnavailable), errors.Is(err, workflow.ErrApprovalPolicyApprovalUnavailable), errors.Is(err, workflow.ErrInvalidApprovalPolicyService):
		return workflowPolicyProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-policy operation could not reach a required dependency.", correlationID)
	case errors.Is(err, workflow.ErrApprovalPolicyVersionConflict), errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict), errors.Is(err, workflow.ErrApprovalPolicyCommandInProgress), errors.Is(err, workflow.ErrApprovalPolicyOverlap), errors.Is(err, workflow.ErrApprovalPolicyDuplicate), errors.Is(err, workflow.ErrApprovalPolicyNotFound):
		code := "VERSION_CONFLICT"
		detail := "The policy changed or the requested policy identity is no longer current. Refresh and retry."
		if errors.Is(err, workflow.ErrApprovalPolicyIdempotencyConflict) {
			code, detail = "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data."
		} else if errors.Is(err, workflow.ErrApprovalPolicyCommandInProgress) {
			code, detail = "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key."
		} else if errors.Is(err, workflow.ErrApprovalPolicyOverlap) {
			code, detail = "POLICY_OVERLAP", "The active effective interval overlaps another selectable policy."
		} else if errors.Is(err, workflow.ErrApprovalPolicyNotFound) {
			code, detail = "POLICY_NOT_FOUND", "The requested policy version is not available."
		}
		return workflowPolicyProblem(http.StatusConflict, code, detail, correlationID)
	case errors.Is(err, workflow.ErrInvalidApprovalPolicyCommand):
		return workflowPolicyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval-policy command is invalid.", correlationID)
	case errors.Is(err, workflow.ErrInvalidApprovalPolicy), errors.Is(err, workflow.ErrApprovalPolicyApprovalRequired), errors.Is(err, workflow.ErrApprovalPolicyApprovalRejected), errors.Is(err, workflow.ErrApprovalPolicyImmutable), errors.Is(err, workflow.ErrApprovalPolicyDurableCommandFailed):
		return workflowPolicyProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The approval-policy command violates a workflow rule.", correlationID)
	default:
		return workflowPolicyProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The approval-policy operation could not be completed.", correlationID)
	}
}

func workflowPolicyProblem(status int, code, detail string, correlationID uuid.UUID) generated.WfaMaintainApprovalPoliciesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.WfaMaintainApprovalPoliciesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.WfaMaintainApprovalPoliciesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.WfaMaintainApprovalPoliciesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.WfaMaintainApprovalPoliciesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.WfaMaintainApprovalPoliciesServiceUnavailable(problem)
		return &value
	}
}
