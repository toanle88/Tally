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

type glOpenScopedReopenGateCommandData struct {
	FiscalPeriodID         string                       `json:"fiscalPeriodId"`
	ReopenRequestID        string                       `json:"reopenRequestId"`
	AuthorizationScope     gl.ScopedReopenScope         `json:"authorizationScope"`
	AuthorizationExpiry    time.Time                    `json:"authorizationExpiry"`
	ControlAuthorityEpoch  int64                        `json:"controlAuthorityEpoch"`
	Approval               gl.ApprovalDecisionReference `json:"approval"`
	NoChangeClosePermitted bool                         `json:"noChangeClosePermitted"`
}

type glCloseScopedReopenGateCommandData struct {
	FiscalPeriodID  string `json:"fiscalPeriodId"`
	ReopenRequestID string `json:"reopenRequestId"`
}

func (handler IdentityHandler) GlOpenScopedReopenGate(ctx context.Context, request *generated.CommandRequest, params generated.GlOpenScopedReopenGateParams) (generated.GlOpenScopedReopenGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glOpenScopedReopenGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The scoped-reopen gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glOpenScopedReopenGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The scoped-reopen gate actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glOpenScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The scoped-reopen opening command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glOpenScopedReopenGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glOpenScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glOpenScopedReopenGateCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glOpenScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The scoped-reopen opening data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glOpenScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	reopenRequestID, err := parseSoftCloseGateUUID(payload.ReopenRequestID)
	if err != nil {
		return glOpenScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.OpenScopedReopenGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.OpenScopedReopenGateCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), ReopenRequestID: reopenRequestID,
		AuthorizationScope: payload.AuthorizationScope, AuthorizationExpiry: payload.AuthorizationExpiry,
		ControlAuthorityEpoch: payload.ControlAuthorityEpoch, Approval: payload.Approval,
		NoChangeClosePermitted: payload.NoChangeClosePermitted, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlOpenScopedReopenGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/open-scoped-reopen-gate"), nil
}

func (handler IdentityHandler) GlCloseScopedReopenGate(ctx context.Context, request *generated.CommandRequest, params generated.GlCloseScopedReopenGateParams) (generated.GlCloseScopedReopenGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glCloseScopedReopenGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The scoped-reopen gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glCloseScopedReopenGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The scoped-reopen gate actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glCloseScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The scoped-reopen closing command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glCloseScopedReopenGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glCloseScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glCloseScopedReopenGateCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glCloseScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The scoped-reopen closing data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glCloseScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	reopenRequestID, err := parseSoftCloseGateUUID(payload.ReopenRequestID)
	if err != nil {
		return glCloseScopedReopenGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.CloseScopedReopenGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.CloseScopedReopenGateCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), ReopenRequestID: reopenRequestID,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlCloseScopedReopenGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/close-scoped-reopen-gate"), nil
}

func mapGlOpenScopedReopenGateError(err error, correlationID uuid.UUID) generated.GlOpenScopedReopenGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glOpenScopedReopenGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The scoped-reopen opening command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glOpenScopedReopenGateProblem(status, code, detail, correlationID, nil)
}

func mapGlCloseScopedReopenGateError(err error, correlationID uuid.UUID) generated.GlCloseScopedReopenGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glCloseScopedReopenGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The scoped-reopen closing command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glCloseScopedReopenGateProblem(status, code, detail, correlationID, nil)
}

func glOpenScopedReopenGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlOpenScopedReopenGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlOpenScopedReopenGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlOpenScopedReopenGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlOpenScopedReopenGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlOpenScopedReopenGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlOpenScopedReopenGateServiceUnavailable(problem)
		return &value
	}
}

func glCloseScopedReopenGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlCloseScopedReopenGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlCloseScopedReopenGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlCloseScopedReopenGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlCloseScopedReopenGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlCloseScopedReopenGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlCloseScopedReopenGateServiceUnavailable(problem)
		return &value
	}
}
