package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type glAcquirePostingBarrierCommandData struct {
	FiscalPeriodID        string `json:"fiscalPeriodId"`
	SoftCloseRunID        string `json:"softCloseRunId"`
	SoftCloseControlEpoch int64  `json:"softCloseControlEpoch"`
	PolicyVersion         string `json:"policyVersion"`
	CloseRunID            string `json:"closeRunId"`
	CloseType             string `json:"closeType"`
}

type glReleasePostingBarrierCommandData struct {
	FiscalPeriodID string `json:"fiscalPeriodId"`
	CloseRunID     string `json:"closeRunId"`
}

type glFinalizePostingGateCommandData struct {
	FiscalPeriodID       string `json:"fiscalPeriodId"`
	CloseRunID           string `json:"closeRunId"`
	FinalLedgerWatermark *int64 `json:"finalLedgerWatermark"`
}

func (handler IdentityHandler) GlAcquirePostingBarrier(ctx context.Context, request *generated.CommandRequest, params generated.GlAcquirePostingBarrierParams) (generated.GlAcquirePostingBarrierRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glAcquirePostingBarrierProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The posting barrier service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glAcquirePostingBarrierProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The barrier-acquiring actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glAcquirePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The barrier-acquisition command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glAcquirePostingBarrierProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glAcquirePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glAcquirePostingBarrierCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glAcquirePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The barrier-acquisition data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glAcquirePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	softCloseRunID, err := parseSoftCloseGateUUID(payload.SoftCloseRunID)
	if err != nil {
		return glAcquirePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close run identifier is invalid.", correlationID, nil), nil
	}
	closeRunID, err := parseSoftCloseGateUUID(payload.CloseRunID)
	if err != nil {
		return glAcquirePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close run identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.AcquirePostingBarrier(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.AcquirePostingBarrierCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), SoftCloseRunID: softCloseRunID,
		SoftCloseControlEpoch: payload.SoftCloseControlEpoch, SoftClosePolicyVersion: payload.PolicyVersion,
		CloseRunID: closeRunID, CloseType: payload.CloseType, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlAcquirePostingBarrierError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/acquire-posting-barrier"), nil
}

func (handler IdentityHandler) GlReleasePostingBarrier(ctx context.Context, request *generated.CommandRequest, params generated.GlReleasePostingBarrierParams) (generated.GlReleasePostingBarrierRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glReleasePostingBarrierProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The posting barrier service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glReleasePostingBarrierProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The barrier-releasing actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glReleasePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The barrier-release command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glReleasePostingBarrierProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glReleasePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glReleasePostingBarrierCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil {
		return glReleasePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The barrier-release data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glReleasePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	closeRunID, err := parseSoftCloseGateUUID(payload.CloseRunID)
	if err != nil {
		return glReleasePostingBarrierProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close run identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.ReleasePostingBarrier(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.ReleasePostingBarrierCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), CloseRunID: closeRunID,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlReleasePostingBarrierError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/release-posting-barrier"), nil
}

func (handler IdentityHandler) GlFinalizePostingGate(ctx context.Context, request *generated.CommandRequest, params generated.GlFinalizePostingGateParams) (generated.GlFinalizePostingGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glFinalizePostingGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The posting barrier service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glFinalizePostingGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The finalizing actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glFinalizePostingGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The gate-finalization command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glFinalizePostingGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glFinalizePostingGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	var payload glFinalizePostingGateCommandData
	if err := decodePostingBarrierData(request, &payload); err != nil || payload.FinalLedgerWatermark == nil {
		return glFinalizePostingGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The finalization data must include a final ledger watermark.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glFinalizePostingGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	closeRunID, err := parseSoftCloseGateUUID(payload.CloseRunID)
	if err != nil {
		return glFinalizePostingGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close run identifier is invalid.", correlationID, nil), nil
	}
	result, err := handler.SoftCloseGateService.FinalizePostingGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, gl.FinalizePostingGateCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID,
		ExpectedGateVersion: int64(request.ExpectedVersion.Value), CloseRunID: closeRunID,
		FinalLedgerWatermark: *payload.FinalLedgerWatermark, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapGlFinalizePostingGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/finalize-posting-gate"), nil
}

func decodePostingBarrierData(request *generated.CommandRequest, target any) error {
	if request == nil {
		return errors.New("request is required")
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func postingBarrierErrorOutcome(err error) (int, string, string) {
	if errors.Is(err, gl.ErrPostingBarrierNotReleasable) || errors.Is(err, gl.ErrPostingBarrierAdmissionConflict) || errors.Is(err, gl.ErrPostingBarrierWatermarkConflict) || errors.Is(err, gl.ErrPostingBarrierFinalized) {
		return http.StatusConflict, "BARRIER_CONFLICT", "The posting barrier changed or the requested recovery action is not allowed. Use the authoritative gate result before retrying."
	}
	return softCloseGateErrorOutcome(err)
}

func mapGlAcquirePostingBarrierError(err error, correlationID uuid.UUID) generated.GlAcquirePostingBarrierRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glAcquirePostingBarrierProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The barrier-acquisition command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := postingBarrierErrorOutcome(err)
	return glAcquirePostingBarrierProblem(status, code, detail, correlationID, nil)
}

func mapGlReleasePostingBarrierError(err error, correlationID uuid.UUID) generated.GlReleasePostingBarrierRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glReleasePostingBarrierProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The barrier-release command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := postingBarrierErrorOutcome(err)
	return glReleasePostingBarrierProblem(status, code, detail, correlationID, nil)
}

func mapGlFinalizePostingGateError(err error, correlationID uuid.UUID) generated.GlFinalizePostingGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glFinalizePostingGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The gate-finalization command violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := postingBarrierErrorOutcome(err)
	return glFinalizePostingGateProblem(status, code, detail, correlationID, nil)
}

func postingBarrierProblemDetails(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.ProblemDetails {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	problem.Type = "https://tally.local/problems/" + strings.ToLower(code)
	return problem
}

func glAcquirePostingBarrierProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlAcquirePostingBarrierRes {
	problem := postingBarrierProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlAcquirePostingBarrierBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlAcquirePostingBarrierForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlAcquirePostingBarrierConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlAcquirePostingBarrierUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlAcquirePostingBarrierServiceUnavailable(problem)
		return &value
	}
}

func glReleasePostingBarrierProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlReleasePostingBarrierRes {
	problem := postingBarrierProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlReleasePostingBarrierBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlReleasePostingBarrierForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlReleasePostingBarrierConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlReleasePostingBarrierUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlReleasePostingBarrierServiceUnavailable(problem)
		return &value
	}
}

func glFinalizePostingGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlFinalizePostingGateRes {
	problem := postingBarrierProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlFinalizePostingGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlFinalizePostingGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlFinalizePostingGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlFinalizePostingGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlFinalizePostingGateServiceUnavailable(problem)
		return &value
	}
}
