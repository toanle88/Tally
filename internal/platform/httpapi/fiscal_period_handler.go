package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type fpmStartHardCloseCommandData struct {
	FiscalPeriodID        string `json:"fiscalPeriodId"`
	SoftCloseRunID        string `json:"softCloseRunId"`
	SoftCloseControlEpoch int64  `json:"softCloseControlEpoch"`
	PolicyVersion         string `json:"policyVersion"`
	ExpectedGateVersion   int64  `json:"expectedGateVersion"`
}

type fpmApplyPostingGateResultCommandData struct {
	FiscalPeriodID             string                `json:"fiscalPeriodId"`
	SoftCloseRunID             string                `json:"softCloseRunId"`
	CloseRunID                 string                `json:"closeRunId"`
	SoftCloseControlEpoch      int64                 `json:"softCloseControlEpoch"`
	PolicyVersion              string                `json:"policyVersion"`
	ExpectedGateVersion        int64                 `json:"expectedGateVersion"`
	PostingGateResult          json.RawMessage       `json:"postingGateResult"`
	Gate                       *gl.PeriodPostingGate `json:"gate"`
	Operation                  string                `json:"operation"`
	DecisionReference          uuid.UUID             `json:"decisionReference"`
	PolicyReference            string                `json:"policyReference"`
	AuthorizationPolicyVersion string                `json:"authorizationPolicyVersion"`
	ValidationOutcome          string                `json:"validationOutcome"`
	AuditReference             uuid.UUID             `json:"auditReference"`
	NextAction                 string                `json:"nextAction"`
}

func (handler IdentityHandler) FpmStartHardClose(ctx context.Context, request *generated.CommandRequest, params generated.FpmStartHardCloseParams) (generated.FpmStartHardCloseRes, error) {
	correlationID := fpmStartHardCloseCorrelation(params)
	if handler.FiscalPeriodService == nil {
		return fpmStartHardCloseProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmStartHardCloseProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The hard-close actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return fpmStartHardCloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The hard-close command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmStartHardCloseCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmStartHardCloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The hard-close command data is invalid.", correlationID), nil
	}
	scopeID := uuid.UUID(request.AccountingScopeId.Value)
	periodID, err := parseFPMUUID(payload.FiscalPeriodID)
	if err != nil {
		return fpmStartHardCloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID), nil
	}
	softCloseRunID, err := parseFPMUUID(payload.SoftCloseRunID)
	if err != nil {
		return fpmStartHardCloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close run identifier is invalid.", correlationID), nil
	}
	if payload.ExpectedGateVersion < 1 || payload.SoftCloseControlEpoch < 1 || strings.TrimSpace(payload.PolicyVersion) == "" {
		return fpmStartHardCloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected gate version, control epoch, and policy version are required.", correlationID), nil
	}
	result, err := handler.FiscalPeriodService.StartHardClose(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, fiscalperiod.StartHardCloseCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: periodID, SoftCloseRunID: softCloseRunID,
		ExpectedPeriodVersion: int64(request.ExpectedVersion.Value), ExpectedGateVersion: payload.ExpectedGateVersion,
		SoftCloseControlEpoch: payload.SoftCloseControlEpoch, SoftClosePolicyVersion: payload.PolicyVersion,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapFPMStartHardCloseError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/start-hard-close"), nil
}

func (handler IdentityHandler) FpmApplyPostingGateResult(ctx context.Context, request *generated.CommandRequest, params generated.FpmApplyPostingGateResultParams) (generated.FpmApplyPostingGateResultRes, error) {
	correlationID := fpmApplyPostingGateResultCorrelation(params)
	if handler.FiscalPeriodService == nil {
		return fpmApplyPostingGateResultProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmApplyPostingGateResultProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The posting-gate result actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The posting-gate result command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmApplyPostingGateResultCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The posting-gate result data is invalid.", correlationID), nil
	}
	scopeID := uuid.UUID(request.AccountingScopeId.Value)
	periodID, err := parseFPMUUID(payload.FiscalPeriodID)
	if err != nil {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID), nil
	}
	softCloseRunID, err := parseFPMUUID(payload.SoftCloseRunID)
	if err != nil {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close run identifier is invalid.", correlationID), nil
	}
	closeRunID, err := parseFPMUUID(payload.CloseRunID)
	if err != nil {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close-run identifier is invalid.", correlationID), nil
	}
	result, err := fpmPostingGateResult(payload)
	if err != nil {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The authoritative posting-gate result is invalid.", correlationID), nil
	}
	if payload.ExpectedGateVersion < 1 || payload.SoftCloseControlEpoch < 1 || strings.TrimSpace(payload.PolicyVersion) == "" {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected gate version, control epoch, and policy version are required.", correlationID), nil
	}
	command := fiscalperiod.ApplyPostingGateResultCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: periodID,
		SoftCloseRunID: softCloseRunID, CloseRunID: closeRunID, ExpectedPeriodVersion: int64(request.ExpectedVersion.Value),
		ExpectedGateVersion: payload.ExpectedGateVersion, SoftCloseControlEpoch: payload.SoftCloseControlEpoch,
		SoftClosePolicyVersion: payload.PolicyVersion, Result: result, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	resultValue, err := handler.FiscalPeriodService.ApplyPostingGateResult(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapFPMApplyPostingGateResultError(err, correlationID), nil
	}
	return establishedFPMResult(resultValue, correlationID, "/api/v1/fiscal-periods/actions/apply-posting-gate-result"), nil
}

func decodeFPMCommandData(request *generated.CommandRequest, target any) error {
	if request == nil {
		return errors.New("request is required")
	}
	data, err := json.Marshal(request.Data)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

func fpmPostingGateResult(payload fpmApplyPostingGateResultCommandData) (gl.SoftCloseGateResult, error) {
	if len(payload.PostingGateResult) > 0 && string(payload.PostingGateResult) != "null" {
		var result gl.SoftCloseGateResult
		if err := json.Unmarshal(payload.PostingGateResult, &result); err != nil {
			return gl.SoftCloseGateResult{}, err
		}
		return result, nil
	}
	if payload.Gate == nil {
		return gl.SoftCloseGateResult{}, errors.New("postingGateResult or gate is required")
	}
	return gl.SoftCloseGateResult{Operation: payload.Operation, Gate: *payload.Gate, DecisionReference: payload.DecisionReference, PolicyReference: payload.PolicyReference, AuthorizationPolicyVersion: payload.AuthorizationPolicyVersion, ValidationOutcome: payload.ValidationOutcome, AuditReference: payload.AuditReference, NextAction: payload.NextAction}, nil
}

func establishedFPMResult(result fiscalperiod.CommandResult, correlationID uuid.UUID, self string) *generated.EstablishedResult {
	period := result.State.Period
	data := generated.EstablishedResultData{
		"periodId": mustRaw(period.ID.String()), "accountingScopeId": mustRaw(period.AccountingScopeID.String()), "periodState": mustRaw(period.State),
		"periodStateVersion": mustRaw(period.StateVersion), "periodAggregateVersion": mustRaw(period.AggregateVersion),
		"softCloseRunId": mustRaw(result.State.SoftClose.ID.String()), "softCloseStatus": mustRaw(result.State.SoftClose.Status),
		"controlEpoch": mustRaw(result.State.ControlEpoch.Number), "controlEpochStatus": mustRaw(result.State.ControlEpoch.Status),
		"closeRunId": mustRaw(result.State.CloseRun.ID.String()), "closeRunStatus": mustRaw(result.State.CloseRun.Status),
		"gateMode": mustRaw(result.GateResult.Gate.GateMode), "gateVersion": mustRaw(result.GateResult.Gate.GateVersion),
		"barrierLedgerPosition": mustRaw(result.GateResult.Gate.BarrierLedgerPosition), "frozenAdmissionSummary": mustRaw(result.State.CloseRun.BarrierEvidence),
		"validationOutcome": mustRaw(result.GateResult.ValidationOutcome), "decisionReference": mustRaw(result.GateResult.DecisionReference.String()),
		"policyReference": mustRaw(result.GateResult.PolicyReference), "auditReference": mustRaw(result.GateResult.AuditReference.String()),
		"nextAction": mustRaw(result.GateResult.NextAction), "replayed": mustRaw(result.Replayed),
	}
	if result.State.CloseRun.BarrierEvidence != nil {
		data["priorControlProcessId"] = mustRaw(result.State.CloseRun.BarrierEvidence.PriorControlProcessID)
		data["priorControlEpoch"] = mustRaw(result.State.CloseRun.BarrierEvidence.PriorControlEpoch)
		data["frozenAdmissionSummary"] = mustRaw(result.State.CloseRun.BarrierEvidence.FrozenSummary)
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(period.ID), AggregateVersion: int(period.AggregateVersion), ProcessId: generated.NewOptNilUUID(generated.UUID(result.State.CloseRun.ID)), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: self}, Data: data}
}

func fpmStartHardCloseCorrelation(params generated.FpmStartHardCloseParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func fpmApplyPostingGateResultCorrelation(params generated.FpmApplyPostingGateResultParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func parseFPMUUID(value string) (uuid.UUID, error) {
	parsed, err := uuid.Parse(strings.TrimSpace(value))
	if err != nil || parsed == uuid.Nil {
		return uuid.Nil, errors.New("invalid UUID")
	}
	return parsed, nil
}

func mapFPMStartHardCloseError(err error, correlationID uuid.UUID) generated.FpmStartHardCloseRes {
	return mapFPMError(err, correlationID, fpmStartHardCloseProblem)
}

func mapFPMApplyPostingGateResultError(err error, correlationID uuid.UUID) generated.FpmApplyPostingGateResultRes {
	return mapFPMError(err, correlationID, fpmApplyPostingGateResultProblem)
}

type fpmProblemFactory[T any] func(int, string, string, uuid.UUID) T

func mapFPMError[T any](err error, correlationID uuid.UUID, problem fpmProblemFactory[T]) T {
	status, code, detail := http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period operation could not be completed."
	switch {
	case errors.Is(err, fiscalperiod.ErrInvalidFiscalPeriodCommand):
		status, code, detail = http.StatusBadRequest, "INVALID_REQUEST", "The fiscal-period command is invalid."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodAuthorizationDenied):
		status, code, detail = http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested fiscal-period scope is outside the actor's authorization."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodAuthorizationUnavailable):
		code, detail = "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodIdempotencyConflict):
		status, code, detail = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodCommandInProgress):
		status, code, detail = http.StatusConflict, "COMMAND_IN_PROGRESS", "The fiscal-period command is still being finalized. Retry with the same idempotency key."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodDurableCommandFailed):
		status, code, detail = http.StatusConflict, "COMMAND_FAILED", "The fiscal-period command has a durably recorded failure. Use the recovery workflow before submitting a new command."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodNotFound):
		status, code, detail = http.StatusConflict, "PERIOD_NOT_FOUND", "The requested fiscal period does not exist in the selected accounting scope."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodVersionConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodOwnerConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodScopeConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodEpochConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodPolicyConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodStateConflict), errors.Is(err, fiscalperiod.ErrPostingBarrierResultConflict):
		status, code, detail = http.StatusConflict, "FISCAL_PERIOD_CONFLICT", "The fiscal-period or authoritative posting-gate state changed. Refresh before retrying."
	case errors.Is(err, fiscalperiod.ErrPostingBarrierRejected):
		status, code, detail = http.StatusUnprocessableEntity, "POSTING_BARRIER_REJECTED", "GL rejected the barrier request. The current owner remains visible and restrictive."
	case errors.Is(err, fiscalperiod.ErrPostingBarrierUnavailable):
		code, detail = "DEPENDENCY_UNAVAILABLE", "The authoritative posting barrier could not be reached. The current owner remains visible and restrictive."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodAuditUnavailable), errors.Is(err, fiscalperiod.ErrFiscalPeriodOutboxUnavailable):
		code, detail = "SERVICE_UNAVAILABLE", "The fiscal-period evidence could not be durably recorded. Retry safely."
	}
	return problem(status, code, detail, correlationID)
}

func fpmStartHardCloseProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmStartHardCloseRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmStartHardCloseBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmStartHardCloseForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmStartHardCloseConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmStartHardCloseUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmStartHardCloseServiceUnavailable(problem)
		return &value
	}
}

func fpmApplyPostingGateResultProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmApplyPostingGateResultRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmApplyPostingGateResultBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmApplyPostingGateResultForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmApplyPostingGateResultConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmApplyPostingGateResultUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmApplyPostingGateResultServiceUnavailable(problem)
		return &value
	}
}
