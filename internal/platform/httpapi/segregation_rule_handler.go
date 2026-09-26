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
)

func (handler IdentityHandler) IamManageSegregationRules(ctx context.Context, request *generated.IamManageSegregationRulesCommandRequest, params generated.IamManageSegregationRulesParams) (generated.IamManageSegregationRulesRes, error) {
	correlationID := segregationCorrelationFromParams(params)
	if handler.SegregationRuleService == nil {
		return iamSegregationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segregation-rule service is unavailable.", correlationID, nil)
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return iamSegregationProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID, nil)
	}

	data := request.Data
	ruleID := uuid.Nil
	if data.RuleId.Set {
		ruleID = uuid.UUID(data.RuleId.Value)
	}
	var expectedVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return iamSegregationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected version is invalid.", correlationID, nil)
		}
		expectedVersion = &version
	}
	if data.RuleId.Set && params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return iamSegregationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The If-Match version is invalid.", correlationID, nil)
		}
		if expectedVersion != nil && expectedVersion.Value() != headerVersion.Value() {
			return iamSegregationProblem(http.StatusConflict, "VERSION_CONFLICT", "The body version and If-Match version do not agree.", correlationID, nil)
		}
		expectedVersion = &headerVersion
	}

	threshold, err := segregationAmount(data.AmountThreshold)
	if err != nil {
		return iamSegregationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The amount threshold is invalid.", correlationID, nil)
	}
	approval := identity.ApprovalDecisionReference{
		ApprovalRequestID:    uuid.UUID(data.Approval.ApprovalRequestId),
		DecisionID:           uuid.UUID(data.Approval.DecisionId),
		PolicyVersion:        data.Approval.PolicyVersion,
		DecisionVersion:      int64(data.Approval.DecisionVersion),
		SubjectVersion:       int64(data.Approval.SubjectVersion),
		CandidateFingerprint: data.Approval.CandidateFingerprint,
		ApproverUserID:       uuid.UUID(data.Approval.ApproverUserId),
	}
	var effectiveTo *time.Time
	if data.EffectiveTo.Set {
		value := data.EffectiveTo.Value.UTC()
		effectiveTo = &value
	}
	coolingOff := time.Duration(0)
	if data.CoolingOffSeconds.Set {
		coolingOff = time.Duration(data.CoolingOffSeconds.Value) * time.Second
	}
	command := identity.SegregationRuleCommand{
		Action:                 string(data.Action),
		RuleID:                 ruleID,
		Code:                   data.Code,
		Name:                   data.Name,
		ConflictingPermissions: append([]string(nil), data.ConflictingPermissions...),
		EnforcementMode:        identity.SegregationEnforcementMode(data.EnforcementMode),
		ScopeIDs:               append([]string(nil), data.ScopeIds...),
		AmountThreshold:        threshold,
		CoolingOff:             coolingOff,
		EffectiveFrom:          data.EffectiveFrom.UTC(),
		EffectiveTo:            effectiveTo,
		Approval:               approval,
		ExpectedVersion:        expectedVersion,
		IdempotencyKey:         params.IdempotencyKey,
		CorrelationID:          correlationID.String(),
		CausationID:            uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.SegregationRuleService.Execute(ctx, actor, command)
	if err != nil {
		return mapSegregationRuleError(err, correlationID)
	}
	return establishedSegregationRuleResult(result, correlationID), nil
}

func segregationAmount(value generated.OptMoney) (*decimal.Decimal, error) {
	if !value.Set {
		return nil, nil
	}
	parsed, err := decimal.NewFromString(string(value.Value))
	if err != nil || parsed.IsNegative() {
		return nil, errors.New("invalid exact-decimal amount")
	}
	return &parsed, nil
}

func segregationCorrelationFromParams(params generated.IamManageSegregationRulesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedSegregationRuleResult(result identity.SegregationRuleCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["ruleId"] = mustRaw(result.Rule.ID.String())
	data["code"] = mustRaw(result.Rule.Code)
	data["name"] = mustRaw(result.Rule.Name)
	data["status"] = mustRaw(result.Rule.Status)
	data["aggregateVersion"] = mustRaw(result.Rule.Version.Value())
	data["enforcementMode"] = mustRaw(result.Rule.EnforcementMode)
	data["conflictingPermissionCount"] = mustRaw(len(result.Rule.ConflictingPermissions))
	data["policyReference"] = mustRaw(result.PolicyReference)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["approval"] = mustRaw(map[string]any{
		"approvalRequestId":    result.Rule.Approval.ApprovalRequestID.String(),
		"decisionId":           result.Rule.Approval.DecisionID.String(),
		"policyVersion":        result.Rule.Approval.PolicyVersion,
		"decisionVersion":      result.Rule.Approval.DecisionVersion,
		"subjectVersion":       result.Rule.Approval.SubjectVersion,
		"candidateFingerprint": result.Rule.Approval.CandidateFingerprint,
		"approverUserId":       result.Rule.Approval.ApproverUserID.String(),
	})
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.Rule.ID),
		AggregateVersion: int(result.Rule.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/identity-access/actions/manage-segregation-rules"},
		Data:             data,
	}
}

func mapSegregationRuleError(err error, correlationID uuid.UUID) (generated.IamManageSegregationRulesRes, error) {
	switch {
	case errors.Is(err, identity.ErrAuthorizationStale), errors.Is(err, identity.ErrSegregationRuleStale):
		return iamSegregationProblem(http.StatusConflict, "POLICY_STALE", "The authorization or segregation policy changed. Refresh and retry.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationExpired):
		return iamSegregationProblem(http.StatusForbidden, "POLICY_EXPIRED", "The authorization policy is no longer effective.", correlationID, nil)
	case errors.Is(err, identity.ErrAuthorizationUnavailable), errors.Is(err, identity.ErrSegregationRuleUnavailable), errors.Is(err, identity.ErrSegregationRuleAudit):
		return iamSegregationProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The segregation policy could not be evaluated or recorded. Retry later.", correlationID, nil)
	case errors.Is(err, identity.ErrSegregationRuleAuthorization):
		return iamSegregationProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized to change segregation rules.", correlationID, nil)
	case errors.Is(err, identity.ErrSegregationRuleVersion):
		return iamSegregationProblem(http.StatusConflict, "VERSION_CONFLICT", "The segregation rule changed after it was loaded. Refresh and retry.", correlationID, nil)
	case errors.Is(err, identity.ErrSegregationRuleIdempotency):
		return iamSegregationProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID, nil)
	case errors.Is(err, identity.ErrSegregationRuleInProgress):
		return iamSegregationProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID, nil)
	case errors.Is(err, identity.ErrSegregationRuleNotFound):
		return iamSegregationProblem(http.StatusConflict, "RULE_NOT_FOUND", "The requested segregation rule does not exist.", correlationID, nil)
	case errors.Is(err, identity.ErrInvalidSegregationCommand):
		return iamSegregationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The manage-segregation-rules command is invalid.", correlationID, nil)
	case errors.Is(err, identity.ErrInvalidSegregationRule), errors.Is(err, identity.ErrSegregationRuleRetired), errors.Is(err, identity.ErrSegregationRuleApproval), errors.Is(err, identity.ErrApprovalRequired):
		return iamSegregationProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The segregation-rule command violates an identity rule.", correlationID, nil)
	default:
		return iamSegregationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segregation-rule operation could not be completed.", correlationID, nil)
	}
}

func iamSegregationProblem(status int, code, detail string, correlationID uuid.UUID, currentVersion *int) (generated.IamManageSegregationRulesRes, error) {
	problem := generated.ProblemDetails{
		Type:          "https://tally.local/problems/" + code,
		Title:         http.StatusText(status),
		Status:        status,
		Code:          code,
		Detail:        detail,
		CorrelationId: generated.UUID(correlationID),
	}
	if currentVersion != nil {
		problem.CurrentVersion.SetTo(*currentVersion)
	}
	switch status {
	case http.StatusBadRequest:
		value := generated.IamManageSegregationRulesBadRequest(problem)
		return &value, nil
	case http.StatusForbidden:
		value := generated.IamManageSegregationRulesForbidden(problem)
		return &value, nil
	case http.StatusConflict:
		value := generated.IamManageSegregationRulesConflict(problem)
		return &value, nil
	case http.StatusUnprocessableEntity:
		value := generated.IamManageSegregationRulesUnprocessableEntity(problem)
		return &value, nil
	default:
		value := generated.IamManageSegregationRulesServiceUnavailable(problem)
		return &value, nil
	}
}
