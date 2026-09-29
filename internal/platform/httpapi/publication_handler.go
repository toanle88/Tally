package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type masterDataPublicationCommandData struct {
	AggregateType string `json:"aggregateType"`
	AggregateID   string `json:"aggregateId"`
}

func (handler IdentityHandler) OmdPublishApprovedMasterDataChanges(ctx context.Context, request *generated.CommandRequest, params generated.OmdPublishApprovedMasterDataChangesParams) (generated.OmdPublishApprovedMasterDataChangesRes, error) {
	correlationID := correlationFromOmdPublicationParams(params)
	if handler.PublicationService == nil {
		return omdPublicationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The master-data publication service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdPublicationProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The publication command data is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The publication command data is invalid.", correlationID), nil
	}
	var payload masterDataPublicationCommandData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil || strings.TrimSpace(payload.AggregateType) == "" || strings.TrimSpace(payload.AggregateID) == "" {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The publication command data is invalid.", correlationID), nil
	}
	aggregateID, err := uuid.Parse(strings.TrimSpace(payload.AggregateID))
	if err != nil {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The publication aggregate identifier is invalid.", correlationID), nil
	}
	if !request.AccountingScopeId.Set {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope is required for publication.", correlationID), nil
	}
	if !request.ExpectedVersion.Set {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected aggregate version is required for publication.", correlationID), nil
	}
	expectedVersion, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
	if err != nil {
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected aggregate version is invalid.", correlationID), nil
	}
	command := organization.MasterDataPublicationCommand{
		AggregateType:   organization.MasterDataAggregateType(strings.ToLower(strings.TrimSpace(payload.AggregateType))),
		AggregateID:     aggregateID,
		ScopeID:         uuid.UUID(request.AccountingScopeId.Value),
		ExpectedVersion: &expectedVersion,
		IdempotencyKey:  params.IdempotencyKey,
		CorrelationID:   correlationID.String(),
		CausationID:     uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.PublicationService.Execute(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapOmdPublicationError(err, correlationID), nil
	}
	return establishedMasterDataPublicationResult(result, correlationID), nil
}

func establishedMasterDataPublicationResult(result organization.MasterDataPublicationResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["publicationId"] = mustRaw(result.PublicationID)
	data["messageId"] = mustRaw(result.MessageID)
	data["aggregateType"] = mustRaw(result.AggregateType)
	data["status"] = mustRaw(result.Status)
	if result.EffectiveFrom != nil {
		data["effectiveFrom"] = mustRaw(result.EffectiveFrom.Format("2006-01-02"))
	}
	data["dependentAvailability"] = mustRaw(result.DependentAvailability)
	data["eventType"] = mustRaw(result.EventType)
	return &generated.EstablishedResult{Status: result.Status, AggregateId: generated.UUID(result.AggregateID), AggregateVersion: int(result.AggregateVersion.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/actions/publish-approved-master-data-changes/" + result.PublicationID.String()}, Data: data}
}

func mapOmdPublicationError(err error, correlationID uuid.UUID) generated.OmdPublishApprovedMasterDataChangesRes {
	switch {
	case errors.Is(err, organization.ErrMasterDataPublicationAuthorizationDenied):
		return omdPublicationProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested master-data scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationAuthorizationUnavailable), errors.Is(err, organization.ErrMasterDataPublicationAuditUnavailable):
		return omdPublicationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The master-data publication operation could not be completed.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationAuthorizationStale):
		return omdPublicationProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationVersionConflict), errors.Is(err, organization.ErrMasterDataPublicationSourceChanged), errors.Is(err, organization.ErrMasterDataPublicationAlreadyPublished):
		return omdPublicationProblem(http.StatusConflict, "VERSION_CONFLICT", "The approved master-data version changed or has already been published.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationIdempotencyConflict):
		return omdPublicationProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different publication data.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationCommandInProgress):
		return omdPublicationProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The publication command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationNotFound):
		return omdPublicationProblem(http.StatusConflict, "MASTER_DATA_NOT_FOUND", "The requested master-data record does not exist.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationApprovalRequired):
		return omdPublicationProblem(http.StatusUnprocessableEntity, "APPROVAL_REQUIRED", "Only an approved master-data version can be published.", correlationID)
	case errors.Is(err, organization.ErrMasterDataPublicationApprovalStale):
		return omdPublicationProblem(http.StatusUnprocessableEntity, "APPROVAL_STALE", "The approval evidence no longer matches the current master-data version.", correlationID)
	case errors.Is(err, organization.ErrInvalidMasterDataPublicationCommand):
		return omdPublicationProblem(http.StatusBadRequest, "INVALID_REQUEST", "The publication command data is invalid.", correlationID)
	default:
		return omdPublicationProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The master-data publication operation could not be completed.", correlationID)
	}
}

func omdPublicationProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdPublishApprovedMasterDataChangesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdPublishApprovedMasterDataChangesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdPublishApprovedMasterDataChangesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.OmdPublishApprovedMasterDataChangesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.OmdPublishApprovedMasterDataChangesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.OmdPublishApprovedMasterDataChangesServiceUnavailable(problem)
		return &value
	}
}

func correlationFromOmdPublicationParams(params generated.OmdPublishApprovedMasterDataChangesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}
