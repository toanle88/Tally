package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func (handler IdentityHandler) GlSubmitPostingRequest(ctx context.Context, request *generated.GlSubmitPostingRequestCommandRequest, params generated.GlSubmitPostingRequestParams) (generated.GlSubmitPostingRequestRes, error) {
	correlationID := glPostingCorrelation(params)
	if handler.PostingService == nil {
		return glPostingProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The posting service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glPostingProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The posting actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return glPostingProblem(http.StatusBadRequest, "INVALID_REQUEST", "The posting command envelope is invalid.", correlationID, nil), nil
	}
	if _, err := glScopeID(request.AccountingScopeId, params.XAccountingScopeID); err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glPostingProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glPostingProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}

	postingRequest, err := glPostingRequestFromGenerated(request, params, correlationID)
	if err != nil {
		return glPostingProblem(http.StatusBadRequest, "INVALID_REQUEST", "The posting request is malformed.", correlationID, nil), nil
	}
	result, err := handler.PostingService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, postingRequest)
	if err != nil {
		return mapGlPostingError(err, correlationID), nil
	}
	return establishedGlPostingResult(result, correlationID), nil
}

func glPostingRequestFromGenerated(request *generated.GlSubmitPostingRequestCommandRequest, params generated.GlSubmitPostingRequestParams, correlationID uuid.UUID) (gl.PostingRequest, error) {
	data := request.Data
	scope, err := accountingscope.New(uuid.UUID(data.TenantId), uuid.UUID(data.LegalEntityId), uuid.UUID(data.LedgerId), uuid.UUID(data.AccountingBookId), string(data.FunctionalCurrency))
	if err != nil {
		return gl.PostingRequest{}, err
	}
	result := gl.PostingRequest{
		ContractVersion: int(data.ContractVersion), RequestID: uuid.UUID(data.RequestId),
		SourceContext: data.SourceContext, SourceAggregateType: data.SourceAggregateType, SourceAggregateID: uuid.UUID(data.SourceAggregateId), SourceVersion: int64(data.SourceVersion),
		AccountingScope: scope, AccountingScopeID: uuid.UUID(request.AccountingScopeId), PostingDate: data.PostingDate, FiscalPeriodID: uuid.UUID(data.FiscalPeriodId), PeriodStateVersion: int64(data.PeriodStateVersion), PostingGateVersion: int64(data.PostingGateVersion),
		PostingPurpose: string(data.PostingPurpose), TransactionCurrency: string(data.TransactionCurrency), IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	if data.AdjustmentPeriodIndicator.Set {
		result.AdjustmentPeriodIndicator = data.AdjustmentPeriodIndicator.Value
	}
	result.PostingAuthorizationID = generatedPostingUUID(data.PostingAuthorizationId)
	result.CloseRunID = generatedPostingUUID(data.CloseRunId)
	result.ReopenRequestID = generatedPostingUUID(data.ReopenRequestId)
	result.OperationalReopenRequestID = generatedPostingUUID(data.OperationalReopenRequestId)
	if data.ControlAuthorityEpoch.Set {
		value := int64(data.ControlAuthorityEpoch.Value)
		result.ControlAuthorityEpoch = &value
	}
	if data.Description.Set {
		result.Description = data.Description.Value
	}
	result.ReversalOfJournalEntryID = generatedPostingUUID(data.ReversalOfJournalEntryId)
	if data.AutomaticReversalDate.Set {
		value := data.AutomaticReversalDate.Value
		result.AutomaticReversalDate = &value
	}
	if data.ConversionEvidence.Set {
		evidence := data.ConversionEvidence.Value
		result.ConversionEvidence = &gl.PostingConversionEvidence{RateSetID: uuid.UUID(evidence.RateSetId), RateType: evidence.RateType, ConversionDate: evidence.ConversionDate, ConversionTimestamp: evidence.ConversionTimestamp}
	}
	result.Lines = make([]gl.PostingLine, len(data.Lines))
	for index, line := range data.Lines {
		result.Lines[index] = gl.PostingLine{
			AccountID: uuid.UUID(line.AccountId), DebitOrCredit: string(line.DebitOrCredit), LineCurrencyMode: string(line.LineCurrencyMode),
			TransactionAmount: string(line.TransactionAmount), FunctionalAmount: string(line.FunctionalAmount), SegmentCombinationID: uuid.UUID(line.SegmentCombinationId),
		}
		if line.LineReference.Set {
			result.Lines[index].LineReference = line.LineReference.Value
		}
	}
	return result, nil
}

func generatedPostingUUID(value generated.OptUUID) *uuid.UUID {
	if !value.Set {
		return nil
	}
	result := uuid.UUID(value.Value)
	return &result
}

func glPostingCorrelation(params generated.GlSubmitPostingRequestParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedGlPostingResult(result gl.PostingResult, correlationID uuid.UUID) *generated.GlSubmitPostingRequestEstablishedResult {
	journalID := result.AggregateID()
	data := generated.GlSubmitPostingRequestResultData{
		Outcome:           generated.GlSubmitPostingRequestResultDataOutcome(result.Outcome),
		LifecycleStatus:   generated.GlSubmitPostingRequestResultDataLifecycleStatus(result.LifecycleStatus()),
		ValidationOutcome: result.ValidationOutcome, ApprovalStatus: result.ApprovalStatus, SourceReference: result.SourceReference,
		GateEvidence: generated.GlPostingGateEvidence{FiscalPeriodId: generated.UUID(result.GateEvidence.FiscalPeriodID), PeriodStateVersion: int(result.GateEvidence.PeriodStateVersion), PostingGateVersion: int(result.GateEvidence.PostingGateVersion), GateMode: result.GateEvidence.GateMode},
		Replayed:     result.Replayed,
	}
	if journalID != uuid.Nil {
		data.JournalId = generated.NewOptUUID(generated.UUID(journalID))
	}
	if result.Journal != nil {
		data.JournalNumber = generated.NewOptString(result.Journal.Number)
		data.JournalVersion = generated.NewOptInt(int(result.Journal.Version))
		if result.Journal.LedgerPosition > 0 {
			data.LedgerPosition = generated.NewOptInt(int(result.Journal.LedgerPosition))
		}
	}
	if result.ApprovalRequestID != uuid.Nil {
		data.ApprovalRequestId = generated.NewOptUUID(generated.UUID(result.ApprovalRequestID))
	}
	if result.NextAction != "" {
		data.NextAction = generated.NewOptString(result.NextAction)
	}
	if result.AuditReference != uuid.Nil {
		data.AuditReference = generated.NewOptUUID(generated.UUID(result.AuditReference))
	}
	if len(result.Issues) > 0 {
		data.Issues = make([]generated.GlPostingValidationIssue, len(result.Issues))
		for index, issue := range result.Issues {
			data.Issues[index] = generated.GlPostingValidationIssue{Code: issue.Code, Field: issue.Field, Message: issue.Message}
		}
	}
	return &generated.GlSubmitPostingRequestEstablishedResult{
		Status: "established", AggregateId: generated.UUID(journalID), AggregateVersion: int(result.AggregateVersion()), CorrelationId: generated.UUID(correlationID),
		Links: generated.Links{Self: "/api/v1/general-ledger/actions/submit-posting-request"}, Data: data,
	}
}

func mapGlPostingError(err error, correlationID uuid.UUID) generated.GlSubmitPostingRequestRes {
	var validation *gl.PostingValidationError
	if errors.As(err, &validation) {
		return glPostingProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The posting request violates a general-ledger rule.", correlationID, validation.Issues)
	}
	switch {
	case errors.Is(err, gl.ErrPostingAuthorizationDenied):
		return glPostingProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The posting actor is not authorized for this accounting effect.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingAuthorizationUnavailable), errors.Is(err, gl.ErrPostingReferenceUnavailable), errors.Is(err, gl.ErrPostingApprovalUnavailable), errors.Is(err, gl.ErrPostingAuditUnavailable), errors.Is(err, gl.ErrPostingOutboxUnavailable), errors.Is(err, gl.ErrInvalidPostingService):
		return glPostingProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The posting operation could not be completed.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingIdempotencyConflict):
		return glPostingProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different posting data.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingCommandInProgress):
		return glPostingProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The posting command is still being finalized. Retry with the same idempotency key.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingSourceDuplicate):
		return glPostingProblem(http.StatusConflict, "SOURCE_DUPLICATE", "The source aggregate already owns an accounting effect.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingPeriodConflict), errors.Is(err, gl.ErrPostingGateConflict):
		return glPostingProblem(http.StatusConflict, "VERSION_CONFLICT", "The period or posting gate changed. Refresh and retry.", correlationID, nil)
	case errors.Is(err, gl.ErrPostingGateClosed), errors.Is(err, gl.ErrPostingReferenceInvalid), errors.Is(err, gl.ErrPostingValidation):
		return glPostingProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The posting request violates a general-ledger rule.", correlationID, nil)
	case errors.Is(err, gl.ErrInvalidPostingRequest):
		return glPostingProblem(http.StatusBadRequest, "INVALID_REQUEST", "The posting request is invalid.", correlationID, nil)
	default:
		return glPostingProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The posting operation could not be completed.", correlationID, nil)
	}
}

func glPostingProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlSubmitPostingRequestRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	if len(issues) > 0 {
		problem.FieldErrors = make([]generated.ProblemFieldError, len(issues))
		for index, issue := range issues {
			problem.FieldErrors[index] = generated.ProblemFieldError{Code: generated.NewOptString(issue.Code), Field: issue.Field, Message: issue.Message}
		}
	}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlSubmitPostingRequestBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlSubmitPostingRequestForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlSubmitPostingRequestConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlSubmitPostingRequestUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlSubmitPostingRequestServiceUnavailable(problem)
		return &value
	}
}
