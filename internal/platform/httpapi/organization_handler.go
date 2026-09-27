package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type legalEntityCommandData struct {
	Action               string                                  `json:"action"`
	LegalEntityID        string                                  `json:"legalEntityId"`
	LegalName            string                                  `json:"legalName"`
	FunctionalCurrency   string                                  `json:"functionalCurrency"`
	PresentationCurrency string                                  `json:"presentationCurrency"`
	TaxRegistrationID    string                                  `json:"taxRegistrationId"`
	EffectiveFrom        string                                  `json:"effectiveFrom"`
	EffectiveTo          *string                                 `json:"effectiveTo"`
	Registrations        []legalEntityRegistrationData           `json:"registrations"`
	Addresses            []legalEntityAddressData                `json:"addresses"`
	OwnershipInterests   []legalEntityOwnershipData              `json:"ownershipInterests"`
	Approval             *organization.ApprovalDecisionReference `json:"approval"`
}

type legalEntityRegistrationData struct {
	Type          string  `json:"type"`
	Identifier    string  `json:"identifier"`
	Jurisdiction  string  `json:"jurisdiction"`
	EffectiveFrom string  `json:"effectiveFrom"`
	EffectiveTo   *string `json:"effectiveTo"`
}
type legalEntityAddressData struct {
	Type          string  `json:"type"`
	Line1         string  `json:"line1"`
	Line2         string  `json:"line2"`
	Locality      string  `json:"locality"`
	Region        string  `json:"region"`
	PostalCode    string  `json:"postalCode"`
	CountryCode   string  `json:"countryCode"`
	EffectiveFrom string  `json:"effectiveFrom"`
	EffectiveTo   *string `json:"effectiveTo"`
}
type legalEntityOwnershipData struct {
	OwnerReference string  `json:"ownerReference"`
	Percentage     string  `json:"percentage"`
	EffectiveFrom  string  `json:"effectiveFrom"`
	EffectiveTo    *string `json:"effectiveTo"`
}

func (handler IdentityHandler) OmdMaintainLegalEntities(ctx context.Context, request *generated.CommandRequest, params generated.OmdMaintainLegalEntitiesParams) (generated.OmdMaintainLegalEntitiesRes, error) {
	if handler.OrganizationService == nil {
		return omdProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity service is unavailable.", correlationFromOmdParams(params)), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationFromOmdParams(params)), nil
	}
	var payload legalEntityCommandData
	data, err := json.Marshal(request.Data)
	if err != nil {
		return omdProblem(http.StatusBadRequest, "INVALID_REQUEST", "The legal-entity command data is invalid.", correlationFromOmdParams(params)), nil
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return omdProblem(http.StatusBadRequest, "INVALID_REQUEST", "The legal-entity command data is invalid.", correlationFromOmdParams(params)), nil
	}
	command, err := legalEntityCommandFromTransport(request, params, payload)
	if err != nil {
		return omdProblem(http.StatusBadRequest, "INVALID_REQUEST", "The legal-entity command data is invalid.", correlationFromOmdParams(params)), nil
	}
	result, err := handler.OrganizationService.Execute(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapOmdError(err, correlationFromOmdParams(params)), nil
	}
	return establishedLegalEntityResult(result, correlationFromOmdParams(params)), nil
}

func legalEntityCommandFromTransport(request *generated.CommandRequest, params generated.OmdMaintainLegalEntitiesParams, payload legalEntityCommandData) (organization.LegalEntityCommand, error) {
	command := organization.LegalEntityCommand{Action: payload.Action, LegalName: payload.LegalName, FunctionalCurrency: payload.FunctionalCurrency, PresentationCurrency: payload.PresentationCurrency, TaxRegistrationID: payload.TaxRegistrationID, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationFromOmdParams(params).String(), CausationID: uuid.UUID(request.CommandId).String(), Approval: payload.Approval}
	if params.IfMatch.Set {
		version, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, err
		}
		command.ExpectedVersion = &version
	} else if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, err
		}
		command.ExpectedVersion = &version
	}
	if request.AccountingScopeId.Set {
		command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	}
	if payload.LegalEntityID != "" {
		parsed, err := uuid.Parse(payload.LegalEntityID)
		if err != nil {
			return command, err
		}
		command.LegalEntityID = parsed
	}
	if payload.EffectiveFrom != "" {
		parsed, err := parseLegalEntityDate(payload.EffectiveFrom)
		if err != nil {
			return command, err
		}
		command.EffectiveFrom = parsed
	}
	if payload.EffectiveTo != nil {
		parsed, err := parseLegalEntityDate(*payload.EffectiveTo)
		if err != nil {
			return command, err
		}
		command.EffectiveTo = &parsed
	}
	for _, value := range payload.Registrations {
		from, err := parseLegalEntityDate(value.EffectiveFrom)
		if err != nil {
			return command, err
		}
		var to *time.Time
		if value.EffectiveTo != nil {
			parsed, parseErr := parseLegalEntityDate(*value.EffectiveTo)
			if parseErr != nil {
				return command, parseErr
			}
			to = &parsed
		}
		command.Registrations = append(command.Registrations, organization.LegalEntityRegistration{Type: value.Type, Identifier: value.Identifier, Jurisdiction: value.Jurisdiction, EffectiveFrom: from, EffectiveTo: to})
	}
	for _, value := range payload.Addresses {
		from, err := parseLegalEntityDate(value.EffectiveFrom)
		if err != nil {
			return command, err
		}
		var to *time.Time
		if value.EffectiveTo != nil {
			parsed, parseErr := parseLegalEntityDate(*value.EffectiveTo)
			if parseErr != nil {
				return command, parseErr
			}
			to = &parsed
		}
		command.Addresses = append(command.Addresses, organization.LegalEntityAddress{Type: value.Type, Line1: value.Line1, Line2: value.Line2, Locality: value.Locality, Region: value.Region, PostalCode: value.PostalCode, CountryCode: value.CountryCode, EffectiveFrom: from, EffectiveTo: to})
	}
	for _, value := range payload.OwnershipInterests {
		from, err := parseLegalEntityDate(value.EffectiveFrom)
		if err != nil {
			return command, err
		}
		var to *time.Time
		if value.EffectiveTo != nil {
			parsed, parseErr := parseLegalEntityDate(*value.EffectiveTo)
			if parseErr != nil {
				return command, parseErr
			}
			to = &parsed
		}
		command.OwnershipInterests = append(command.OwnershipInterests, organization.LegalEntityOwnershipInterest{OwnerReference: value.OwnerReference, Percentage: value.Percentage, EffectiveFrom: from, EffectiveTo: to})
	}
	if payload.Action != organization.LegalEntityActionCreate && !params.IfMatch.Set {
		return command, errors.New("If-Match is required for a mutable legal-entity command")
	}
	return command, nil
}

func parseLegalEntityDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, err
	}
	return parsed.UTC(), nil
}
func correlationFromOmdParams(params generated.OmdMaintainLegalEntitiesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}
func establishedLegalEntityResult(result organization.LegalEntityCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{}
	data["legalEntity"] = mustRaw(result.LegalEntity)
	data["decisionReference"] = mustRaw(result.DecisionReference.String())
	data["policyReference"] = mustRaw(result.PolicyReference)
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.LegalEntity.ID), AggregateVersion: int(result.LegalEntity.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/reference/legal-entities/" + result.LegalEntity.ID.String()}, Data: data}
}

func mapOmdError(err error, correlationID uuid.UUID) generated.OmdMaintainLegalEntitiesRes {
	switch {
	case errors.Is(err, organization.ErrLegalEntityAuthorizationDenied):
		return omdProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested legal-entity scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityAuthorizationStale):
		return omdProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityAuthorizationUnavailable), errors.Is(err, organization.ErrLegalEntityAuditUnavailable), errors.Is(err, organization.ErrLegalEntityApprovalUnavailable):
		return omdProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity operation could not be completed.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityVersionConflict):
		return omdProblem(http.StatusConflict, "VERSION_CONFLICT", "The legal entity changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityIdempotencyConflict):
		return omdProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityCommandInProgress):
		return omdProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityNotFound):
		return omdProblem(http.StatusConflict, "LEGAL_ENTITY_NOT_FOUND", "The requested legal entity does not exist.", correlationID)
	case errors.Is(err, organization.ErrLegalEntityApprovalRequired), errors.Is(err, organization.ErrLegalEntityApprovalRejected), errors.Is(err, organization.ErrInvalidLegalEntity), errors.Is(err, organization.ErrInvalidLegalEntityCommand), errors.Is(err, organization.ErrLegalEntityDuplicate):
		return omdProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The legal-entity command violates an organization rule.", correlationID)
	default:
		return omdProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity operation could not be completed.", correlationID)
	}
}
func omdProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdMaintainLegalEntitiesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdMaintainLegalEntitiesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdMaintainLegalEntitiesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.OmdMaintainLegalEntitiesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.OmdMaintainLegalEntitiesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.OmdMaintainLegalEntitiesServiceUnavailable(problem)
		return &value
	}
}

func (handler IdentityHandler) OmdListLegalEntities(ctx context.Context, params generated.OmdListLegalEntitiesParams) (generated.OmdListLegalEntitiesRes, error) {
	correlationID := omdListCorrelation(params)
	if handler.OrganizationService == nil {
		return omdListProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdListProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	scopeID := optionalScopeID(params.XAccountingScopeID)
	values, err := handler.OrganizationService.ListSafe(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, scopeID)
	if err != nil {
		return mapOmdListReadError(err, correlationID), nil
	}
	if params.PageAfter.Set {
		after, parseErr := uuid.Parse(strings.TrimSpace(params.PageAfter.Value))
		if parseErr != nil {
			return omdListProblem(http.StatusBadRequest, "INVALID_PAGE", "The page cursor is invalid.", correlationID), nil
		}
		filtered := values[:0]
		for _, value := range values {
			if bytes.Compare(value.ID[:], after[:]) > 0 {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	pageSize := 50
	if params.PageSize.Set {
		pageSize = params.PageSize.Value
	}
	hasMore := len(values) > pageSize
	if hasMore {
		values = values[:pageSize]
	}
	page := map[string]any{"size": pageSize, "hasMore": hasMore}
	if hasMore && len(values) > 0 {
		page["after"] = values[len(values)-1].ID.String()
	}
	data := generated.EstablishedResultData{"items": mustRaw(values), "page": mustRaw(page)}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(uuid.Nil), AggregateVersion: 0, CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/reference/legal-entities"}, Data: data}, nil
}

func (handler IdentityHandler) OmdGetLegalEntity(ctx context.Context, params generated.OmdGetLegalEntityParams) (generated.OmdGetLegalEntityRes, error) {
	correlationID := omdGetCorrelation(params)
	if handler.OrganizationService == nil {
		return omdGetProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdGetProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	value, err := handler.OrganizationService.GetSafe(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, uuid.UUID(params.LegalEntityId), optionalScopeID(params.XAccountingScopeID))
	if err != nil {
		return mapOmdGetReadError(err, correlationID), nil
	}
	data := generated.EstablishedResultData{"legalEntity": mustRaw(value)}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(value.ID), AggregateVersion: int(value.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/reference/legal-entities/" + value.ID.String()}, Data: data}, nil
}

func optionalScopeID(value generated.OptUUID) *uuid.UUID {
	if !value.Set {
		return nil
	}
	scopeID := uuid.UUID(value.Value)
	return &scopeID
}

func omdListCorrelation(params generated.OmdListLegalEntitiesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func omdGetCorrelation(params generated.OmdGetLegalEntityParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func mapOmdListReadError(err error, correlationID uuid.UUID) generated.OmdListLegalEntitiesRes {
	status, code, detail := http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity read could not be completed."
	if errors.Is(err, organization.ErrLegalEntityAuthorizationDenied) {
		status, code, detail = http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested legal-entity scope is outside the actor scope."
	}
	return omdListProblem(status, code, detail, correlationID)
}

func mapOmdGetReadError(err error, correlationID uuid.UUID) generated.OmdGetLegalEntityRes {
	status, code, detail := http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The legal-entity read could not be completed."
	switch {
	case errors.Is(err, organization.ErrLegalEntityNotFound):
		status, code, detail = http.StatusNotFound, "LEGAL_ENTITY_NOT_FOUND", "The requested legal entity does not exist."
	case errors.Is(err, organization.ErrLegalEntityAuthorizationDenied):
		status, code, detail = http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested legal-entity scope is outside the actor scope."
	}
	return omdGetProblem(status, code, detail, correlationID)
}

func omdListProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdListLegalEntitiesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdListLegalEntitiesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdListLegalEntitiesForbidden(problem)
		return &value
	default:
		value := generated.OmdListLegalEntitiesServiceUnavailable(problem)
		return &value
	}
}

func omdGetProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdGetLegalEntityRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdGetLegalEntityBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdGetLegalEntityForbidden(problem)
		return &value
	case http.StatusNotFound:
		value := generated.OmdGetLegalEntityNotFound(problem)
		return &value
	default:
		value := generated.OmdGetLegalEntityServiceUnavailable(problem)
		return &value
	}
}
