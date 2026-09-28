package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

type fiscalCalendarCommandData struct {
	Action           string                                  `json:"action"`
	FiscalCalendarID string                                  `json:"fiscalCalendarId"`
	ScopeID          string                                  `json:"scopeId"`
	CalendarType     string                                  `json:"calendarType"`
	PeriodPattern    string                                  `json:"periodPattern"`
	EffectiveFrom    string                                  `json:"effectiveFrom"`
	EffectiveTo      *string                                 `json:"effectiveTo"`
	Periods          []fiscalCalendarPeriodData              `json:"periods"`
	Approval         *organization.ApprovalDecisionReference `json:"approval"`
}

type fiscalCalendarPeriodData struct {
	ID        string `json:"id"`
	Reference string `json:"reference"`
	Ordinal   int    `json:"ordinal"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
}

var (
	errFiscalCalendarTransportConflict = errors.New("fiscal calendar transport conflict")
	errFiscalCalendarTransportInvalid  = errors.New("fiscal calendar transport invalid")
)

func (handler IdentityHandler) OmdMaintainFiscalCalendars(ctx context.Context, request *generated.CommandRequest, params generated.OmdMaintainFiscalCalendarsParams) (generated.OmdMaintainFiscalCalendarsRes, error) {
	correlationID := correlationFromOmdFiscalCalendarsParams(params)
	ctx, finish := beginFiscalCalendarTelemetry(handler.Instrumentation, ctx)
	outcome := "failed"
	defer func() { finish(outcome) }()
	if handler.FiscalCalendarService == nil {
		return omdFiscalCalendarProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-calendar service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdFiscalCalendarProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return omdFiscalCalendarProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal-calendar command data is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return omdFiscalCalendarProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal-calendar command data is invalid.", correlationID), nil
	}
	payload, err := decodeFiscalCalendarCommandData(data)
	if err != nil {
		return omdFiscalCalendarProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal-calendar command data is invalid.", correlationID), nil
	}
	command, err := fiscalCalendarCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		if errors.Is(err, errFiscalCalendarTransportConflict) {
			return omdFiscalCalendarProblem(http.StatusConflict, "VERSION_CONFLICT", "The fiscal-calendar version or accounting scope does not agree with the command envelope.", correlationID), nil
		}
		return omdFiscalCalendarProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal-calendar command data is invalid.", correlationID), nil
	}
	if _, present := telemetry.FromContext(ctx); present {
		if enriched, causationErr := telemetry.WithCausation(ctx, uuid.UUID(request.CommandId)); causationErr == nil {
			ctx = enriched
		}
	}
	result, err := handler.FiscalCalendarService.Execute(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapOmdFiscalCalendarError(err, correlationID), nil
	}
	outcome = "established"
	if result.Replayed {
		outcome = "replayed"
	}
	return establishedFiscalCalendarResult(result, correlationID), nil
}

func decodeFiscalCalendarCommandData(data []byte) (fiscalCalendarCommandData, error) {
	var payload fiscalCalendarCommandData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return fiscalCalendarCommandData{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return fiscalCalendarCommandData{}, errFiscalCalendarTransportInvalid
	}
	return payload, nil
}

func fiscalCalendarCommandFromTransport(request *generated.CommandRequest, params generated.OmdMaintainFiscalCalendarsParams, payload fiscalCalendarCommandData, correlationID uuid.UUID) (organization.FiscalCalendarCommand, error) {
	command := organization.FiscalCalendarCommand{
		Action: strings.ToLower(strings.TrimSpace(payload.Action)), ScopeID: uuid.Nil,
		CalendarType: strings.ToLower(strings.TrimSpace(payload.CalendarType)), PeriodPattern: strings.ToLower(strings.TrimSpace(payload.PeriodPattern)),
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(), Approval: payload.Approval,
	}
	if !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil {
		return command, errFiscalCalendarTransportInvalid
	}
	command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	if payload.ScopeID != "" {
		payloadScope, err := uuid.Parse(strings.TrimSpace(payload.ScopeID))
		if err != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		if payloadScope != command.ScopeID {
			return command, errFiscalCalendarTransportConflict
		}
	}
	if payload.FiscalCalendarID != "" {
		calendarID, err := uuid.Parse(strings.TrimSpace(payload.FiscalCalendarID))
		if err != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		command.FiscalCalendarID = calendarID
	}
	var err error
	command.EffectiveFrom, err = parseFiscalCalendarDate(payload.EffectiveFrom)
	if err != nil {
		return command, errFiscalCalendarTransportInvalid
	}
	if payload.EffectiveTo != nil && strings.TrimSpace(*payload.EffectiveTo) != "" {
		value, parseErr := parseFiscalCalendarDate(*payload.EffectiveTo)
		if parseErr != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		command.EffectiveTo = &value
	}
	for _, value := range payload.Periods {
		if value.Ordinal < 1 || strings.TrimSpace(value.Reference) == "" {
			return command, errFiscalCalendarTransportInvalid
		}
		startDate, parseErr := parseFiscalCalendarDate(value.StartDate)
		if parseErr != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		endDate, parseErr := parseFiscalCalendarDate(value.EndDate)
		if parseErr != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		period := organization.CalendarPeriod{Reference: strings.ToLower(strings.TrimSpace(value.Reference)), Ordinal: value.Ordinal, StartDate: startDate, EndDate: endDate}
		if strings.TrimSpace(value.ID) != "" {
			period.ID, parseErr = uuid.Parse(strings.TrimSpace(value.ID))
			if parseErr != nil {
				return command, errFiscalCalendarTransportInvalid
			}
		}
		command.Periods = append(command.Periods, period)
	}
	var bodyVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		bodyVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, errFiscalCalendarTransportInvalid
		}
		if bodyVersion != nil && bodyVersion.Value() != headerVersion.Value() {
			return command, errFiscalCalendarTransportConflict
		}
		command.ExpectedVersion = &headerVersion
	} else {
		command.ExpectedVersion = bodyVersion
	}
	if command.Action == organization.FiscalCalendarActionCreate && command.ExpectedVersion != nil {
		return command, errFiscalCalendarTransportInvalid
	}
	if command.Action == organization.FiscalCalendarActionMaintain && !params.IfMatch.Set {
		return command, errFiscalCalendarTransportInvalid
	}
	if err := command.Validate(); err != nil {
		return command, errFiscalCalendarTransportInvalid
	}
	return command, nil
}

func parseFiscalCalendarDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func establishedFiscalCalendarResult(result organization.FiscalCalendarCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"fiscalCalendar":    mustRaw(result.FiscalCalendar),
		"periodCount":       mustRaw(len(result.FiscalCalendar.Periods)),
		"status":            mustRaw(result.FiscalCalendar.Status),
		"aggregateVersion":  mustRaw(result.FiscalCalendar.Version.Value()),
		"validationOutcome": mustRaw(result.ValidationOutcome),
		"impact":            mustRaw(result.Impact),
		"decisionReference": mustRaw(result.DecisionReference.String()),
		"policyReference":   mustRaw(result.PolicyReference),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.FiscalCalendar.ID), AggregateVersion: int(result.FiscalCalendar.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/configuration/maintain-fiscal-calendars"}, Data: data}
}

func mapOmdFiscalCalendarError(err error, correlationID uuid.UUID) generated.OmdMaintainFiscalCalendarsRes {
	switch {
	case errors.Is(err, organization.ErrFiscalCalendarAuthorizationDenied):
		return omdFiscalCalendarProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested fiscal-calendar scope is outside the administering actor authority.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarAuthorizationStale):
		return omdFiscalCalendarProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarVersionConflict):
		return omdFiscalCalendarProblem(http.StatusConflict, "VERSION_CONFLICT", "The fiscal calendar changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarIdempotencyConflict):
		return omdFiscalCalendarProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarCommandInProgress):
		return omdFiscalCalendarProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarNotFound):
		return omdFiscalCalendarProblem(http.StatusConflict, "CALENDAR_NOT_FOUND", "The requested fiscal calendar does not exist.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarDuplicate):
		return omdFiscalCalendarProblem(http.StatusConflict, "DUPLICATE_CALENDAR", "An active fiscal calendar with this type already exists in the accounting scope.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarAuthorizationUnavailable), errors.Is(err, organization.ErrFiscalCalendarAuditUnavailable), errors.Is(err, organization.ErrFiscalCalendarDurableCommandFailed), errors.Is(err, organization.ErrInvalidFiscalCalendarService):
		return omdFiscalCalendarProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-calendar operation could not be completed.", correlationID)
	case errors.Is(err, organization.ErrFiscalCalendarApprovalRequired), errors.Is(err, organization.ErrFiscalCalendarApprovalRejected), errors.Is(err, organization.ErrInvalidFiscalCalendar), errors.Is(err, organization.ErrInvalidFiscalCalendarCommand):
		return omdFiscalCalendarProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The fiscal-calendar command violates an organization rule.", correlationID)
	default:
		return omdFiscalCalendarProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-calendar operation could not be completed.", correlationID)
	}
}

func omdFiscalCalendarProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdMaintainFiscalCalendarsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdMaintainFiscalCalendarsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdMaintainFiscalCalendarsForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.OmdMaintainFiscalCalendarsConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.OmdMaintainFiscalCalendarsUnprocessableEntity(problem)
		return &value
	default:
		value := generated.OmdMaintainFiscalCalendarsServiceUnavailable(problem)
		return &value
	}
}

func correlationFromOmdFiscalCalendarsParams(params generated.OmdMaintainFiscalCalendarsParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func beginFiscalCalendarTelemetry(instrumentation *telemetry.Instrumentation, ctx context.Context) (context.Context, func(string)) {
	if instrumentation == nil {
		return ctx, func(string) {}
	}
	ctx, span := instrumentation.StartSpan(ctx, "organization.omd_maintain_fiscal_calendars", telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainFiscalCalendars", DataClassification: "internal"})
	finished := false
	return ctx, func(result string) {
		if finished {
			return
		}
		finished = true
		instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainFiscalCalendars", Result: result, DataClassification: "internal"})
		instrumentation.AddCommand(ctx, 1, "organization", "omdMaintainFiscalCalendars", result)
		span.End()
	}
}
