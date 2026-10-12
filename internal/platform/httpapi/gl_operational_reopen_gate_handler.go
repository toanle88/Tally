package httpapi

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type glOpenOperationalReopenGateCommandData struct {
	FiscalPeriodID         string                       `json:"fiscalPeriodId"`
	ReopenRequestID        string                       `json:"reopenRequestId"`
	AuthorizationScope     gl.OperationalReopenScope    `json:"authorizationScope"`
	AuthorizationExpiry    time.Time                    `json:"authorizationExpiry"`
	ControlAuthorityEpoch  int64                        `json:"controlAuthorityEpoch"`
	Approval               gl.ApprovalDecisionReference `json:"approval"`
	NoChangeClosePermitted bool                         `json:"noChangeClosePermitted"`
}

type glCloseOperationalReopenGateCommandData struct {
	FiscalPeriodID  string `json:"fiscalPeriodId"`
	ReopenRequestID string `json:"reopenRequestId"`
}

type glBeginRecloseGateCommandData struct {
	FiscalPeriodID  string `json:"fiscalPeriodId"`
	ReopenRequestID string `json:"reopenRequestId"`
	RecloseRunID    string `json:"recloseRunId"`
}

func (handler IdentityHandler) GlOpenOperationalReopenGate(ctx context.Context, request *generated.CommandRequest, params generated.GlOpenOperationalReopenGateParams) (generated.GlOpenOperationalReopenGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glOpenOperationalReopenGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The operational-reopen gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glOpenOperationalReopenGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The operational-reopen gate actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glOpenOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The operational-reopen opening command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glOpenOperationalReopenGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glOpenOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glOpenOperationalReopenGateCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glOpenOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The operational-reopen opening data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glOpenOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	reopenRequestID, err := parseSoftCloseGateUUID(payload.ReopenRequestID)
	if err != nil {
		return glOpenOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.OpenOperationalReopenGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.OpenOperationalReopenGateCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), ReopenRequestID: reopenRequestID,
		AuthorizationScope: payload.AuthorizationScope, AuthorizationExpiry: payload.AuthorizationExpiry,
		ControlAuthorityEpoch: payload.ControlAuthorityEpoch, Approval: payload.Approval,
		NoChangeClosePermitted: payload.NoChangeClosePermitted, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlOpenOperationalReopenGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/open-operational-reopen-gate"), nil
}

func (handler IdentityHandler) GlCloseOperationalReopenGate(ctx context.Context, request *generated.CommandRequest, params generated.GlCloseOperationalReopenGateParams) (generated.GlCloseOperationalReopenGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glCloseOperationalReopenGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The operational-reopen gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glCloseOperationalReopenGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The operational-reopen gate actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glCloseOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The operational-reopen closing command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glCloseOperationalReopenGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glCloseOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glCloseOperationalReopenGateCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glCloseOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The operational-reopen closing data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glCloseOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	reopenRequestID, err := parseSoftCloseGateUUID(payload.ReopenRequestID)
	if err != nil {
		return glCloseOperationalReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.CloseOperationalReopenGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.CloseOperationalReopenGateCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), ReopenRequestID: reopenRequestID,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlCloseOperationalReopenGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/close-operational-reopen-gate"), nil
}

func (handler IdentityHandler) GlBeginRecloseGate(ctx context.Context, request *generated.CommandRequest, params generated.GlBeginRecloseGateParams) (generated.GlBeginRecloseGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glBeginRecloseGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The reclose gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glBeginRecloseGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The reclose gate actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glBeginRecloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reclose command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glBeginRecloseGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glBeginRecloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glBeginRecloseGateCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glBeginRecloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reclose data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glBeginRecloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	reopenRequestID, err := parseSoftCloseGateUUID(payload.ReopenRequestID)
	if err != nil {
		return glBeginRecloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID, nil), nil
	}
	recloseRunID, err := parseSoftCloseGateUUID(payload.RecloseRunID)
	if err != nil {
		return glBeginRecloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reclose run identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.BeginRecloseGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.BeginRecloseGateCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), ReopenRequestID: reopenRequestID, RecloseRunID: recloseRunID,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlBeginRecloseGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/begin-reclose-gate"), nil
}

func mapGlOpenOperationalReopenGateError(err error, correlationID uuid.UUID) generated.GlOpenOperationalReopenGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glOpenOperationalReopenGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The operational-reopen opening command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glOpenOperationalReopenGateProblem(status, code, detail, correlationID, nil)
}

func mapGlCloseOperationalReopenGateError(err error, correlationID uuid.UUID) generated.GlCloseOperationalReopenGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glCloseOperationalReopenGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The operational-reopen closing command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glCloseOperationalReopenGateProblem(status, code, detail, correlationID, nil)
}

func mapGlBeginRecloseGateError(err error, correlationID uuid.UUID) generated.GlBeginRecloseGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glBeginRecloseGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The reclose command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glBeginRecloseGateProblem(status, code, detail, correlationID, nil)
}

func glOpenOperationalReopenGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlOpenOperationalReopenGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlOpenOperationalReopenGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlOpenOperationalReopenGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlOpenOperationalReopenGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlOpenOperationalReopenGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlOpenOperationalReopenGateServiceUnavailable(problem)
		return &value
	}
}

func glCloseOperationalReopenGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlCloseOperationalReopenGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlCloseOperationalReopenGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlCloseOperationalReopenGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlCloseOperationalReopenGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlCloseOperationalReopenGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlCloseOperationalReopenGateServiceUnavailable(problem)
		return &value
	}
}

func glBeginRecloseGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlBeginRecloseGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlBeginRecloseGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlBeginRecloseGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlBeginRecloseGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlBeginRecloseGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlBeginRecloseGateServiceUnavailable(problem)
		return &value
	}
}
