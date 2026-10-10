package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

var (
	errGLScopeConflict   = errors.New("accounting scope mismatch")
	errGLVersionConflict = errors.New("body and header versions do not agree")
)

func (handler IdentityHandler) GlMaintainLedgers(ctx context.Context, request *generated.GlMaintainLedgersCommandRequest, params generated.GlMaintainLedgersParams) (generated.GlMaintainLedgersRes, error) {
	correlationID := glLedgerCorrelation(params)
	if handler.LedgerService == nil {
		return glLedgerProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The ledger service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glLedgerProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return glLedgerProblem(http.StatusBadRequest, "INVALID_REQUEST", "The ledger command envelope is invalid.", correlationID), nil
	}
	scopeID, err := glScopeID(request.AccountingScopeId, params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glLedgerProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return glLedgerProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}
	expectedVersion, err := glExpectedVersion(request.ExpectedVersion, params.IfMatch)
	if err != nil {
		if errors.Is(err, errGLVersionConflict) {
			return glLedgerProblem(http.StatusConflict, "VERSION_CONFLICT", "The expected and If-Match versions do not agree.", correlationID), nil
		}
		return glLedgerProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected or If-Match version is invalid.", correlationID), nil
	}
	data := request.Data
	ledgerID := uuid.Nil
	if data.LedgerId.Set {
		ledgerID = uuid.UUID(data.LedgerId.Value)
	}
	effectiveTo := glEffectiveDateTo(data.EffectiveDateTo)
	command := gl.LedgerCommand{
		Action:             string(data.Action),
		LedgerID:           ledgerID,
		AccountingScopeID:  scopeID,
		LegalEntityID:      uuid.UUID(data.LegalEntityId),
		LedgerType:         data.LedgerType,
		FunctionalCurrency: string(data.FunctionalCurrency),
		FiscalCalendarID:   uuid.UUID(data.FiscalCalendarId),
		LifecycleStatus:    string(data.LifecycleStatus),
		EffectiveDateFrom:  data.EffectiveDateFrom,
		EffectiveDateTo:    effectiveTo,
		Approval:           glApproval(data.Approval),
		ExpectedVersion:    expectedVersion,
		IdempotencyKey:     params.IdempotencyKey,
		CorrelationID:      correlationID.String(),
		CausationID:        uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.LedgerService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlLedgerError(err, correlationID), nil
	}
	return establishedGlLedgerResult(result, correlationID), nil
}

func (handler IdentityHandler) GlMaintainAccountingBooks(ctx context.Context, request *generated.GlMaintainAccountingBooksCommandRequest, params generated.GlMaintainAccountingBooksParams) (generated.GlMaintainAccountingBooksRes, error) {
	correlationID := glBookCorrelation(params)
	if handler.AccountingBookService == nil {
		return glAccountingBookProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The accounting-book service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glAccountingBookProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return glAccountingBookProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting-book command envelope is invalid.", correlationID), nil
	}
	scopeID, err := glScopeID(request.AccountingScopeId, params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glAccountingBookProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return glAccountingBookProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}
	expectedVersion, err := glExpectedVersion(request.ExpectedVersion, params.IfMatch)
	if err != nil {
		if errors.Is(err, errGLVersionConflict) {
			return glAccountingBookProblem(http.StatusConflict, "VERSION_CONFLICT", "The expected and If-Match versions do not agree.", correlationID), nil
		}
		return glAccountingBookProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected or If-Match version is invalid.", correlationID), nil
	}
	data := request.Data
	accountingBookID := uuid.Nil
	if data.AccountingBookId.Set {
		accountingBookID = uuid.UUID(data.AccountingBookId.Value)
	}
	command := gl.AccountingBookCommand{
		Action:               string(data.Action),
		AccountingBookID:     accountingBookID,
		AccountingScopeID:    scopeID,
		LedgerID:             uuid.UUID(data.LedgerId),
		BookType:             data.BookType,
		AccountingBasis:      data.AccountingBasis,
		PostingPolicyVersion: data.PostingPolicyVersion,
		LifecycleStatus:      string(data.LifecycleStatus),
		EffectiveDateFrom:    data.EffectiveDateFrom,
		EffectiveDateTo:      glEffectiveDateTo(data.EffectiveDateTo),
		Approval:             glApproval(data.Approval),
		ExpectedVersion:      expectedVersion,
		IdempotencyKey:       params.IdempotencyKey,
		CorrelationID:        correlationID.String(),
		CausationID:          uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.AccountingBookService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlAccountingBookError(err, correlationID), nil
	}
	return establishedGlAccountingBookResult(result, correlationID), nil
}

func glScopeID(requestScope generated.UUID, headerScope generated.OptUUID) (uuid.UUID, error) {
	scopeID := uuid.UUID(requestScope)
	if scopeID == uuid.Nil {
		return uuid.Nil, errors.New("accounting scope is required")
	}
	if headerScope.Set && uuid.UUID(headerScope.Value) != scopeID {
		return uuid.Nil, errGLScopeConflict
	}
	return scopeID, nil
}

func glExpectedVersion(body generated.OptInt, ifMatch generated.OptString) (*aggregateversion.AggregateVersion, error) {
	var result *aggregateversion.AggregateVersion
	if body.Set {
		value, err := aggregateversion.FromInt64(int64(body.Value))
		if err != nil {
			return nil, err
		}
		result = &value
	}
	if ifMatch.Set {
		value, err := parseIfMatchVersion(ifMatch.Value)
		if err != nil {
			return nil, err
		}
		if result != nil && result.Value() != value.Value() {
			return nil, errGLVersionConflict
		}
		result = &value
	}
	return result, nil
}

func glEffectiveDateTo(value generated.OptNilDate) *time.Time {
	if !value.Set || value.Null {
		return nil
	}
	result := value.Value
	return &result
}

func glApproval(value generated.OptIamApprovalDecisionReference) *gl.ApprovalDecisionReference {
	if !value.Set {
		return nil
	}
	approval := value.Value
	return &gl.ApprovalDecisionReference{ApprovalRequestID: uuid.UUID(approval.ApprovalRequestId), DecisionID: uuid.UUID(approval.DecisionId), PolicyVersion: approval.PolicyVersion, DecisionVersion: int64(approval.DecisionVersion), SubjectVersion: int64(approval.SubjectVersion), CandidateFingerprint: approval.CandidateFingerprint, ApproverUserID: uuid.UUID(approval.ApproverUserId)}
}

func glActorSubject(actor identity.ApplicationActor) string {
	if strings.TrimSpace(actor.Subject.Sub) != "" {
		return actor.Subject.Sub
	}
	return actor.UserID.String()
}

func glLedgerCorrelation(params generated.GlMaintainLedgersParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func glBookCorrelation(params generated.GlMaintainAccountingBooksParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedGlLedgerResult(result gl.LedgerCommandResult, correlationID uuid.UUID) *generated.GlMaintainLedgersEstablishedResult {
	data := generated.GlMaintainLedgersResultData{Ledger: generated.GlLedgerProjection{ID: generated.UUID(result.Ledger.ID), AccountingScopeId: generated.UUID(result.Ledger.AccountingScopeID), LegalEntityId: generated.UUID(result.Ledger.LegalEntityID), LedgerType: result.Ledger.LedgerType, FunctionalCurrency: generated.CurrencyCode(result.Ledger.FunctionalCurrency), FiscalCalendarId: generated.UUID(result.Ledger.FiscalCalendarID), LifecycleStatus: result.Ledger.LifecycleStatus, EffectiveDateFrom: result.Ledger.EffectiveDateFrom, ApprovalStatus: result.Ledger.ApprovalStatus, ValidationOutcome: result.Ledger.ValidationOutcome, NextAction: result.Ledger.NextAction, Version: int(result.Ledger.Version.Value()), RevisionNumber: int(result.Ledger.RevisionNumber)}, ValidationOutcome: result.ValidationOutcome, ApprovalStatus: result.ApprovalStatus}
	data.Ledger.EffectiveDateTo = generatedGlDate(result.Ledger.EffectiveDateTo)
	if result.DecisionReference != uuid.Nil {
		data.DecisionReference = generated.NewOptUUID(generated.UUID(result.DecisionReference))
	}
	if result.PolicyReference != "" {
		data.PolicyReference = generated.NewOptString(result.PolicyReference)
	}
	if result.Replayed {
		data.Replayed = generated.NewOptBool(true)
	}
	return &generated.GlMaintainLedgersEstablishedResult{Status: "established", AggregateId: generated.UUID(result.Ledger.ID), AggregateVersion: int(result.Ledger.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/general-ledger/configuration/maintain-ledgers"}, Data: data}
}

func establishedGlAccountingBookResult(result gl.AccountingBookCommandResult, correlationID uuid.UUID) *generated.GlMaintainAccountingBooksEstablishedResult {
	data := generated.GlMaintainAccountingBooksResultData{AccountingBook: generated.GlAccountingBookProjection{ID: generated.UUID(result.AccountingBook.ID), AccountingScopeId: generated.UUID(result.AccountingBook.AccountingScopeID), LedgerId: generated.UUID(result.AccountingBook.LedgerID), BookType: result.AccountingBook.BookType, AccountingBasis: result.AccountingBook.AccountingBasis, PostingPolicyVersion: result.AccountingBook.PostingPolicyVersion, LifecycleStatus: result.AccountingBook.LifecycleStatus, EffectiveDateFrom: result.AccountingBook.EffectiveDateFrom, ApprovalStatus: result.AccountingBook.ApprovalStatus, ValidationOutcome: result.AccountingBook.ValidationOutcome, NextAction: result.AccountingBook.NextAction, Version: int(result.AccountingBook.Version.Value()), RevisionNumber: int(result.AccountingBook.RevisionNumber)}, ValidationOutcome: result.ValidationOutcome, ApprovalStatus: result.ApprovalStatus}
	data.AccountingBook.EffectiveDateTo = generatedGlDate(result.AccountingBook.EffectiveDateTo)
	if result.DecisionReference != uuid.Nil {
		data.DecisionReference = generated.NewOptUUID(generated.UUID(result.DecisionReference))
	}
	if result.PolicyReference != "" {
		data.PolicyReference = generated.NewOptString(result.PolicyReference)
	}
	if result.Replayed {
		data.Replayed = generated.NewOptBool(true)
	}
	return &generated.GlMaintainAccountingBooksEstablishedResult{Status: "established", AggregateId: generated.UUID(result.AccountingBook.ID), AggregateVersion: int(result.AccountingBook.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/general-ledger/configuration/maintain-accounting-books"}, Data: data}
}

func generatedGlDate(value *time.Time) generated.OptNilDate {
	if value == nil {
		return generated.OptNilDate{Set: true, Null: true}
	}
	return generated.NewOptNilDate(*value)
}

func mapGlLedgerError(err error, correlationID uuid.UUID) generated.GlMaintainLedgersRes {
	switch {
	case errors.Is(err, gl.ErrLedgerAuthorizationUnavailable):
		return glLedgerProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, gl.ErrLedgerAuthorizationStale):
		return glLedgerProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrLedgerAuthorizationDenied):
		return glLedgerProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested ledger scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, gl.ErrLedgerVersionConflict):
		return glLedgerProblem(http.StatusConflict, "VERSION_CONFLICT", "The ledger changed after it was loaded. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrLedgerIdempotencyConflict):
		return glLedgerProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different ledger data.", correlationID)
	case errors.Is(err, gl.ErrLedgerCommandInProgress):
		return glLedgerProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The ledger command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, gl.ErrLedgerNotFound):
		return glLedgerProblem(http.StatusConflict, "LEDGER_NOT_FOUND", "The requested ledger does not exist.", correlationID)
	case errors.Is(err, gl.ErrInvalidLedgerCommand):
		return glLedgerProblem(http.StatusBadRequest, "INVALID_REQUEST", "The ledger command is invalid.", correlationID)
	case errors.Is(err, gl.ErrInvalidLedger), errors.Is(err, gl.ErrLedgerDuplicate), errors.Is(err, gl.ErrLedgerReferenceInvalid), errors.Is(err, gl.ErrLedgerApprovalRequired), errors.Is(err, gl.ErrLedgerApprovalInvalid), errors.Is(err, gl.ErrLedgerDurableCommandFailed):
		return glLedgerProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The ledger command violates a general-ledger rule.", correlationID)
	case errors.Is(err, gl.ErrLedgerReferenceUnavailable), errors.Is(err, gl.ErrLedgerApprovalUnavailable), errors.Is(err, gl.ErrLedgerAuditUnavailable), errors.Is(err, gl.ErrInvalidLedgerService):
		return glLedgerProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The ledger operation could not be completed.", correlationID)
	default:
		return glLedgerProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The ledger operation could not be completed.", correlationID)
	}
}

func mapGlAccountingBookError(err error, correlationID uuid.UUID) generated.GlMaintainAccountingBooksRes {
	switch {
	case errors.Is(err, gl.ErrAccountingBookAuthorizationUnavailable):
		return glAccountingBookProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookAuthorizationStale):
		return glAccountingBookProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookAuthorizationDenied):
		return glAccountingBookProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested accounting-book scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookVersionConflict):
		return glAccountingBookProblem(http.StatusConflict, "VERSION_CONFLICT", "The accounting book changed after it was loaded. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookIdempotencyConflict):
		return glAccountingBookProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different accounting-book data.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookCommandInProgress):
		return glAccountingBookProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The accounting-book command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookNotFound):
		return glAccountingBookProblem(http.StatusConflict, "ACCOUNTING_BOOK_NOT_FOUND", "The requested accounting book does not exist.", correlationID)
	case errors.Is(err, gl.ErrInvalidAccountingBookCommand):
		return glAccountingBookProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting-book command is invalid.", correlationID)
	case errors.Is(err, gl.ErrInvalidAccountingBook), errors.Is(err, gl.ErrAccountingBookDuplicate), errors.Is(err, gl.ErrAccountingBookReferenceInvalid), errors.Is(err, gl.ErrAccountingBookApprovalRequired), errors.Is(err, gl.ErrAccountingBookApprovalInvalid), errors.Is(err, gl.ErrAccountingBookDurableCommandFailed):
		return glAccountingBookProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The accounting-book command violates a general-ledger rule.", correlationID)
	case errors.Is(err, gl.ErrAccountingBookReferenceUnavailable), errors.Is(err, gl.ErrAccountingBookApprovalUnavailable), errors.Is(err, gl.ErrAccountingBookAuditUnavailable), errors.Is(err, gl.ErrInvalidAccountingBookService):
		return glAccountingBookProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The accounting-book operation could not be completed.", correlationID)
	default:
		return glAccountingBookProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The accounting-book operation could not be completed.", correlationID)
	}
}

func glLedgerProblem(status int, code, detail string, correlationID uuid.UUID) generated.GlMaintainLedgersRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlMaintainLedgersBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlMaintainLedgersForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlMaintainLedgersConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlMaintainLedgersUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlMaintainLedgersServiceUnavailable(problem)
		return &value
	}
}

func glAccountingBookProblem(status int, code, detail string, correlationID uuid.UUID) generated.GlMaintainAccountingBooksRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlMaintainAccountingBooksBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlMaintainAccountingBooksForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlMaintainAccountingBooksConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlMaintainAccountingBooksUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlMaintainAccountingBooksServiceUnavailable(problem)
		return &value
	}
}
