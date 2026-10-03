package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type coaSegmentDefinitionCommandData struct {
	Action              string  `json:"action"`
	SegmentDefinitionID string  `json:"segmentDefinitionId"`
	SegmentType         string  `json:"segmentType"`
	Code                string  `json:"code"`
	Name                string  `json:"name"`
	Status              string  `json:"status"`
	EffectiveDateFrom   string  `json:"effectiveDateFrom"`
	EffectiveDateTo     *string `json:"effectiveDateTo"`
}

func (handler IdentityHandler) CoaMaintainSegmentDefinitions(ctx context.Context, request *generated.CommandRequest, params generated.CoaMaintainSegmentDefinitionsParams) (generated.CoaMaintainSegmentDefinitionsRes, error) {
	correlationID := correlationFromCoaParams(params)
	if handler.SegmentDefinitionService == nil {
		return coaProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-definition service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return coaProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command data is invalid.", correlationID), nil
	}
	var payload coaSegmentDefinitionCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command data is invalid.", correlationID), nil
	}
	command, err := coaSegmentDefinitionCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command data is invalid.", correlationID), nil
	}
	result, err := handler.SegmentDefinitionService.Execute(ctx, coa.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapCoaSegmentDefinitionError(err, correlationID), nil
	}
	return establishedCoaSegmentDefinitionResult(result, correlationID), nil
}

func coaSegmentDefinitionCommandFromTransport(request *generated.CommandRequest, params generated.CoaMaintainSegmentDefinitionsParams, payload coaSegmentDefinitionCommandData, correlationID uuid.UUID) (coa.SegmentDefinitionCommand, error) {
	command := coa.SegmentDefinitionCommand{
		Action:         payload.Action,
		SegmentType:    payload.SegmentType,
		Code:           payload.Code,
		Name:           payload.Name,
		Status:         coa.SegmentStatus(payload.Status),
		IdempotencyKey: params.IdempotencyKey,
		CorrelationID:  correlationID.String(),
		CausationID:    uuid.UUID(request.CommandId).String(),
	}
	if request.AccountingScopeId.Set {
		command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	}
	if payload.SegmentDefinitionID != "" {
		parsed, err := uuid.Parse(strings.TrimSpace(payload.SegmentDefinitionID))
		if err != nil {
			return command, err
		}
		command.SegmentDefinitionID = parsed
	}
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, err
		}
		command.ExpectedVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, err
		}
		if command.ExpectedVersion != nil && command.ExpectedVersion.Value() != headerVersion.Value() {
			return command, errors.New("If-Match and expectedVersion do not agree")
		}
		command.ExpectedVersion = &headerVersion
	}
	from, err := parseCoaDate(payload.EffectiveDateFrom)
	if err != nil {
		return command, err
	}
	command.EffectiveDateFrom = from
	if payload.EffectiveDateTo != nil && strings.TrimSpace(*payload.EffectiveDateTo) != "" {
		to, err := parseCoaDate(*payload.EffectiveDateTo)
		if err != nil {
			return command, err
		}
		command.EffectiveDateTo = &to
	}
	return command, nil
}

func parseCoaDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}

func correlationFromCoaParams(params generated.CoaMaintainSegmentDefinitionsParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedCoaSegmentDefinitionResult(result coa.SegmentDefinitionCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["segmentDefinition"] = mustRaw(result.SegmentDefinition)
	data["status"] = mustRaw(result.SegmentDefinition.Status)
	data["aggregateVersion"] = mustRaw(result.SegmentDefinition.Version.Value())
	data["effectiveDateFrom"] = mustRaw(result.SegmentDefinition.EffectiveDateFrom)
	data["effectiveDateTo"] = mustRaw(result.SegmentDefinition.EffectiveDateTo)
	data["approvalStatus"] = mustRaw(result.SegmentDefinition.ApprovalStatus)
	data["validationOutcome"] = mustRaw(result.ValidationOutcome)
	data["nextAction"] = mustRaw(result.SegmentDefinition.NextAction)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["policyReference"] = mustRaw(result.PolicyReference)
	return &generated.EstablishedResult{
		Status:           "established",
		AggregateId:      generated.UUID(result.SegmentDefinition.ID),
		AggregateVersion: int(result.SegmentDefinition.Version.Value()),
		CorrelationId:    generated.UUID(correlationID),
		Links:            generated.Links{Self: "/api/v1/coa-segments/configuration/maintain-segment-definitions"},
		Data:             data,
	}
}

func mapCoaSegmentDefinitionError(err error, correlationID uuid.UUID) generated.CoaMaintainSegmentDefinitionsRes {
	switch {
	case errors.Is(err, coa.ErrSegmentDefinitionAuthorizationUnavailable):
		return coaProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionAuthorizationStale):
		return coaProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionAuthorizationDenied):
		return coaProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested segment-definition scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionVersionConflict):
		return coaProblem(http.StatusConflict, "VERSION_CONFLICT", "The segment definition changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionIdempotencyConflict):
		return coaProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionCommandInProgress):
		return coaProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, coa.ErrSegmentDefinitionNotFound):
		return coaProblem(http.StatusConflict, "SEGMENT_DEFINITION_NOT_FOUND", "The requested segment definition does not exist.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentDefinitionCommand):
		return coaProblem(http.StatusBadRequest, "INVALID_REQUEST", "The segment-definition command is invalid.", correlationID)
	case errors.Is(err, coa.ErrInvalidSegmentDefinition), errors.Is(err, coa.ErrSegmentDefinitionDuplicate), errors.Is(err, coa.ErrSegmentDefinitionDurableCommandFailed):
		return coaProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The segment-definition command violates a COA rule.", correlationID)
	default:
		return coaProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The segment-definition operation could not be completed.", correlationID)
	}
}

func coaProblem(status int, code, detail string, correlationID uuid.UUID) generated.CoaMaintainSegmentDefinitionsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.CoaMaintainSegmentDefinitionsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.CoaMaintainSegmentDefinitionsForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.CoaMaintainSegmentDefinitionsConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.CoaMaintainSegmentDefinitionsUnprocessableEntity(problem)
		return &value
	default:
		value := generated.CoaMaintainSegmentDefinitionsServiceUnavailable(problem)
		return &value
	}
}
