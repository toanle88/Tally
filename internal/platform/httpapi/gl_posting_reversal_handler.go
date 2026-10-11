package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type glReverseJournalEntryCommandData struct {
	JournalID              string `json:"journalId"`
	Reason                 string `json:"reason"`
	EffectivePostingDate   string `json:"effectivePostingDate"`
	PostingDate            string `json:"postingDate"`
	FiscalPeriodID         string `json:"fiscalPeriodId"`
	PeriodStateVersion     int64  `json:"periodStateVersion"`
	PostingGateVersion     int64  `json:"postingGateVersion"`
	PostingAuthorizationID string `json:"postingAuthorizationId"`
	SourceContext          string `json:"sourceContext"`
	SourceAggregateType    string `json:"sourceAggregateType"`
	SourceAggregateID      string `json:"sourceAggregateId"`
	SourceVersion          int64  `json:"sourceVersion"`
}

func (handler IdentityHandler) GlReverseJournalEntry(ctx context.Context, request *generated.CommandRequest, params generated.GlReverseJournalEntryParams) (generated.GlReverseJournalEntryRes, error) {
	correlationID := glReverseJournalEntryCorrelation(params)
	if handler.JournalReversalService == nil {
		return glReverseJournalEntryProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The journal reversal service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glReverseJournalEntryProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The reversing actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glReverseJournalEntryProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reversal command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glReverseJournalEntryProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glReverseJournalEntryProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}

	data, err := json.Marshal(request.Data)
	if err != nil {
		return glReverseJournalEntryProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reversal data is invalid.", correlationID, nil), nil
	}
	var payload glReverseJournalEntryCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return glReverseJournalEntryProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reversal data is invalid.", correlationID, nil), nil
	}
	command, err := glReverseJournalEntryCommandFromTransport(request, params, payload, scopeID, correlationID)
	if err != nil {
		return glReverseJournalEntryProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reversal data is invalid.", correlationID, nil), nil
	}
	result, err := handler.JournalReversalService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlReverseJournalEntryError(err, correlationID), nil
	}
	return establishedGlReverseJournalEntryResult(result, correlationID), nil
}

func glReverseJournalEntryCommandFromTransport(request *generated.CommandRequest, params generated.GlReverseJournalEntryParams, payload glReverseJournalEntryCommandData, scopeID, correlationID uuid.UUID) (gl.ReverseJournalEntryCommand, error) {
	parseRequiredUUID := func(value string) (uuid.UUID, error) {
		value = strings.TrimSpace(value)
		if value == "" {
			return uuid.Nil, errors.New("required UUID is missing")
		}
		parsed, err := uuid.Parse(value)
		if err != nil || parsed == uuid.Nil {
			return uuid.Nil, errors.New("required UUID is invalid")
		}
		return parsed, nil
	}
	journalID, err := parseRequiredUUID(payload.JournalID)
	if err != nil {
		return gl.ReverseJournalEntryCommand{}, err
	}
	fiscalPeriodID, err := parseRequiredUUID(payload.FiscalPeriodID)
	if err != nil {
		return gl.ReverseJournalEntryCommand{}, err
	}
	authorizationID, err := parseRequiredUUID(payload.PostingAuthorizationID)
	if err != nil {
		return gl.ReverseJournalEntryCommand{}, err
	}
	sourceAggregateID, err := parseRequiredUUID(payload.SourceAggregateID)
	if err != nil {
		return gl.ReverseJournalEntryCommand{}, err
	}
	postingDate := strings.TrimSpace(payload.EffectivePostingDate)
	if postingDate == "" {
		postingDate = strings.TrimSpace(payload.PostingDate)
	}
	effectivePostingDate, err := time.Parse("2006-01-02", postingDate)
	if err != nil {
		return gl.ReverseJournalEntryCommand{}, err
	}
	return gl.ReverseJournalEntryCommand{
		CommandID: uuid.UUID(request.CommandId), OriginalJournalID: journalID, AccountingScopeID: scopeID, ExpectedJournalVersion: int64(request.ExpectedVersion.Value),
		CorrectionReason: payload.Reason, EffectivePostingDate: effectivePostingDate, FiscalPeriodID: fiscalPeriodID,
		PeriodStateVersion: payload.PeriodStateVersion, PostingGateVersion: payload.PostingGateVersion, PostingAuthorizationID: authorizationID,
		SourceContext: payload.SourceContext, SourceAggregateType: payload.SourceAggregateType, SourceAggregateID: sourceAggregateID, SourceVersion: payload.SourceVersion,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}, nil
}

func glReverseJournalEntryCorrelation(params generated.GlReverseJournalEntryParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

type glReverseJournalEntryLine struct {
	AccountID            uuid.UUID `json:"accountId"`
	DebitOrCredit        string    `json:"debitOrCredit"`
	LineCurrencyMode     string    `json:"lineCurrencyMode"`
	TransactionAmount    string    `json:"transactionAmount"`
	FunctionalAmount     string    `json:"functionalAmount"`
	SegmentCombinationID uuid.UUID `json:"segmentCombinationId"`
	LineReference        string    `json:"lineReference,omitempty"`
}

func glReverseJournalEntryLines(lines []gl.PostingLine) []glReverseJournalEntryLine {
	result := make([]glReverseJournalEntryLine, len(lines))
	for index, line := range lines {
		result[index] = glReverseJournalEntryLine{
			AccountID: line.AccountID, DebitOrCredit: line.DebitOrCredit, LineCurrencyMode: line.LineCurrencyMode,
			TransactionAmount: line.TransactionAmount, FunctionalAmount: line.FunctionalAmount, SegmentCombinationID: line.SegmentCombinationID, LineReference: line.LineReference,
		}
	}
	return result
}

func establishedGlReverseJournalEntryResult(result gl.JournalReversalResult, correlationID uuid.UUID) *generated.EstablishedResult {
	original := result.Original.Journal
	reversal := result.Reversal.Journal
	data := generated.EstablishedResultData{
		"outcome":                  mustRaw(result.Outcome),
		"originalJournalId":        mustRaw(original.ID.String()),
		"originalJournalNumber":    mustRaw(original.Number),
		"originalJournalVersion":   mustRaw(original.Version),
		"originalLifecycleStatus":  mustRaw(original.Status),
		"originalSourceReference":  mustRaw(original.SourceReference),
		"originalLedgerPosition":   mustRaw(original.LedgerPosition),
		"originalLines":            mustRaw(glReverseJournalEntryLines(result.Original.Lines)),
		"reversalJournalId":        mustRaw(reversal.ID.String()),
		"reversalJournalNumber":    mustRaw(reversal.Number),
		"reversalJournalVersion":   mustRaw(reversal.Version),
		"reversalLifecycleStatus":  mustRaw(reversal.Status),
		"reversalOfJournalEntryId": mustRaw(reversal.ReversalOfJournalEntryID),
		"reversalSourceReference":  mustRaw(reversal.SourceReference),
		"reversalLedgerPosition":   mustRaw(reversal.LedgerPosition),
		"reversalLines":            mustRaw(glReverseJournalEntryLines(result.Reversal.Lines)),
		"correctionReason":         mustRaw(result.CorrectionReason),
		"authorizationReference":   mustRaw(result.AuthorizationReference.String()),
		"gateEvidence":             mustRaw(result.GateEvidence),
		"auditReference":           mustRaw(result.AuditReference.String()),
		"ownerUserId":              mustRaw(reversal.SubmittedByUserID.String()),
		"sourceContext":            mustRaw(result.Reversal.Request.SourceContext),
		"sourceAggregateType":      mustRaw(result.Reversal.Request.SourceAggregateType),
		"sourceAggregateId":        mustRaw(result.Reversal.Request.SourceAggregateID.String()),
		"sourceVersion":            mustRaw(result.Reversal.Request.SourceVersion),
		"nextAction":               mustRaw(result.NextAction),
		"replayed":                 mustRaw(result.Replayed),
	}
	return &generated.EstablishedResult{
		Status: "established", AggregateId: generated.UUID(reversal.ID), AggregateVersion: int(reversal.Version), CorrelationId: generated.UUID(correlationID),
		Links: generated.Links{Self: "/api/v1/general-ledger/actions/reverse-journal-entry"}, Data: data,
	}
}

func mapGlReverseJournalEntryError(err error, correlationID uuid.UUID) generated.GlReverseJournalEntryRes {
	var validation *gl.PostingValidationError
	if errors.As(err, &validation) {
		return glReverseJournalEntryProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The reversal request violates a general-ledger rule.", correlationID, validation.Issues)
	}
	switch {
	case errors.Is(err, gl.ErrPostingAuthorizationDenied):
		return glReverseJournalEntryProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The actor is not authorized to reverse this journal.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingAuthorizationUnavailable), errors.Is(err, gl.ErrPostingReferenceUnavailable), errors.Is(err, gl.ErrPostingAuditUnavailable), errors.Is(err, gl.ErrPostingOutboxUnavailable), errors.Is(err, gl.ErrInvalidPostingService):
		return glReverseJournalEntryProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The journal reversal could not be completed.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingIdempotencyConflict):
		return glReverseJournalEntryProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different reversal data.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingSourceDuplicate):
		return glReverseJournalEntryProblem(http.StatusConflict, "SOURCE_DUPLICATE", "The correction source already owns an accounting effect.", correlationID, nil)
	case errors.Is(err, gl.ErrJournalEntryAlreadyReversed):
		return glReverseJournalEntryProblem(http.StatusConflict, "ALREADY_REVERSED", "The journal already has a linked reversal.", correlationID, nil)
	case errors.Is(err, gl.ErrJournalReversalVersionConflict), errors.Is(err, gl.ErrPostingPeriodConflict), errors.Is(err, gl.ErrPostingGateConflict):
		return glReverseJournalEntryProblem(http.StatusConflict, "VERSION_CONFLICT", "The journal, period, or posting gate changed. Refresh and retry.", correlationID, nil)
	case errors.Is(err, gl.ErrJournalEntryNotFound), errors.Is(err, gl.ErrJournalEntryNotPosted), errors.Is(err, gl.ErrPostingGateClosed), errors.Is(err, gl.ErrPostingReferenceInvalid), errors.Is(err, gl.ErrPostingValidation):
		return glReverseJournalEntryProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The journal is not eligible for reversal in the requested state.", correlationID, nil)
	case errors.Is(err, gl.ErrInvalidJournalReversalCommand):
		return glReverseJournalEntryProblem(http.StatusBadRequest, "INVALID_REQUEST", "The journal reversal command is invalid.", correlationID, nil)
	default:
		return glReverseJournalEntryProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The journal reversal could not be completed.", correlationID, nil)
	}
}

func glReverseJournalEntryProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlReverseJournalEntryRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	if len(issues) > 0 {
		problem.FieldErrors = make([]generated.ProblemFieldError, len(issues))
		for index, issue := range issues {
			problem.FieldErrors[index] = generated.ProblemFieldError{Code: generated.NewOptString(issue.Code), Field: issue.Field, Message: issue.Message}
		}
	}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlReverseJournalEntryBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlReverseJournalEntryForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlReverseJournalEntryConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlReverseJournalEntryUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlReverseJournalEntryServiceUnavailable(problem)
		return &value
	}
}
