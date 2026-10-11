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

type glEnterSoftCloseGateCommandData struct {
	FiscalPeriodID string          `json:"fiscalPeriodId"`
	SoftCloseRunID string          `json:"softCloseRunId"`
	ControlEpoch   int64           `json:"controlEpoch"`
	PolicyVersion  string          `json:"policyVersion"`
	Restrictions   json.RawMessage `json:"restrictions"`
}

type glExitSoftCloseGateCommandData struct {
	FiscalPeriodID string `json:"fiscalPeriodId"`
	SoftCloseRunID string `json:"softCloseRunId"`
	ControlEpoch   int64  `json:"controlEpoch"`
}

func (handler IdentityHandler) GlEnterSoftCloseGate(ctx context.Context, request *generated.CommandRequest, params generated.GlEnterSoftCloseGateParams) (generated.GlEnterSoftCloseGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glEnterSoftCloseGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The soft-close gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glEnterSoftCloseGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The entering actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glEnterSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close entry command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glEnterSoftCloseGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glEnterSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	payloadBytes, err := json.Marshal(request.Data)
	if err != nil {
		return glEnterSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close entry data is invalid.", correlationID, nil), nil
	}
	var payload glEnterSoftCloseGateCommandData
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return glEnterSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close entry data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glEnterSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	runID, err := parseSoftCloseGateUUID(payload.SoftCloseRunID)
	if err != nil {
		return glEnterSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close run identifier is invalid.", correlationID, nil), nil
	}
	command := gl.EnterSoftCloseGateCommand{CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID, ExpectedGateVersion: int64(request.ExpectedVersion.Value), SoftCloseRunID: runID, ControlEpoch: payload.ControlEpoch, PolicyVersion: payload.PolicyVersion, Restrictions: payload.Restrictions, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String()}
	result, err := handler.SoftCloseGateService.EnterSoftCloseGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlEnterSoftCloseGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/enter-soft-close-gate"), nil
}

func (handler IdentityHandler) GlExitSoftCloseGate(ctx context.Context, request *generated.CommandRequest, params generated.GlExitSoftCloseGateParams) (generated.GlExitSoftCloseGateRes, error) {
	correlationID := glSoftCloseGateCorrelation(params.XCorrelationID)
	if handler.SoftCloseGateService == nil {
		return glExitSoftCloseGateProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The soft-close gate service is unavailable.", correlationID, nil), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glExitSoftCloseGateProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The exiting actor is not authorized.", correlationID, nil), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glExitSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close exit command requires command, scope, and expected-version fields.", correlationID, nil), nil
	}
	scopeID, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glExitSoftCloseGateProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID, nil), nil
		}
		return glExitSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID, nil), nil
	}
	payloadBytes, err := json.Marshal(request.Data)
	if err != nil {
		return glExitSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close exit data is invalid.", correlationID, nil), nil
	}
	var payload glExitSoftCloseGateCommandData
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return glExitSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close exit data is invalid.", correlationID, nil), nil
	}
	fiscalPeriodID, err := parseSoftCloseGateUUID(payload.FiscalPeriodID)
	if err != nil {
		return glExitSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID, nil), nil
	}
	runID, err := parseSoftCloseGateUUID(payload.SoftCloseRunID)
	if err != nil {
		return glExitSoftCloseGateProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close run identifier is invalid.", correlationID, nil), nil
	}
	command := gl.ExitSoftCloseGateCommand{CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: fiscalPeriodID, ExpectedGateVersion: int64(request.ExpectedVersion.Value), SoftCloseRunID: runID, ControlEpoch: payload.ControlEpoch, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String()}
	result, err := handler.SoftCloseGateService.ExitSoftCloseGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlExitSoftCloseGateError(err, correlationID), nil
	}
	return establishedGlSoftCloseGateResult(result, correlationID, "/api/v1/general-ledger/actions/exit-soft-close-gate"), nil
}

func parseSoftCloseGateUUID(value string) (uuid.UUID, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return uuid.Nil, errors.New("identifier is required")
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return uuid.Nil, errors.New("identifier is invalid")
	}
	return parsed, nil
}

func glSoftCloseGateCorrelation(value generated.OptUUID) uuid.UUID {
	if value.Set {
		return uuid.UUID(value.Value)
	}
	return uuid.New()
}

func establishedGlSoftCloseGateResult(result gl.SoftCloseGateResult, correlationID uuid.UUID, self string) *generated.EstablishedResult {
	gate := result.Gate
	data := generated.EstablishedResultData{
		"operation":                      mustRaw(result.Operation),
		"gate":                           mustRaw(gate),
		"gateMode":                       mustRaw(gate.GateMode),
		"accountingScopeId":              mustRaw(gate.AccountingScopeID.String()),
		"fiscalPeriodId":                 mustRaw(gate.FiscalPeriodID.String()),
		"ownerType":                      mustRaw(ownerType(gate)),
		"ownerId":                        mustRaw(ownerID(gate)),
		"controlEpoch":                   mustRaw(gate.ActiveSoftCloseControlEpoch),
		"controlAuthorityEpoch":          mustRaw(gate.ReopenControlAuthorityEpoch),
		"reopenRequestId":                mustRaw(uuidPointerString(gate.ActiveReopenRequestID)),
		"authorizationScope":             mustRaw(gate.ReopenAuthorizationScope),
		"authorizationExpiry":            mustRaw(gate.ReopenAuthorizationExpiry),
		"reopenApproval":                 mustRaw(gate.ReopenApproval),
		"reopenNoChangeClosePermitted":   mustRaw(gate.ReopenNoChangeClosePermitted),
		"closeType":                      mustRaw(gate.ActiveCloseType),
		"priorOwnerType":                 mustRaw(gate.PriorControlProcessType),
		"priorOwnerId":                   mustRaw(uuidPointerString(gate.PriorControlProcessID)),
		"priorControlEpoch":              mustRaw(gate.PriorControlEpoch),
		"priorGateMode":                  mustRaw(gate.PriorGateMode),
		"barrierLedgerPosition":          mustRaw(gate.BarrierLedgerPosition),
		"finalLedgerWatermark":           mustRaw(gate.FinalLedgerWatermark),
		"finalizedCloseRunId":            mustRaw(uuidPointerString(gate.FinalizedCloseRunID)),
		"finalizationCommandFingerprint": mustRaw(gate.FinalizationCommandFingerprint),
		"gateVersion":                    mustRaw(gate.GateVersion),
		"periodStateVersion":             mustRaw(gate.PeriodStateVersion),
		"policyVersion":                  mustRaw(gate.SoftClosePolicyVersion),
		"restrictions":                   mustRaw(gate.SoftCloseRestrictions),
		"activeAdmissionCounters":        mustRaw(gate.ActiveAdmissionCounters),
		"lastFrozenAdmissionSummary":     mustRaw(gate.LastFrozenAdmissionSummary),
		"frozenAdmissionSummaryHistory":  mustRaw(gate.FrozenAdmissionSummaryHistory),
		"decisionReference":              mustRaw(result.DecisionReference.String()),
		"policyReference":                mustRaw(result.PolicyReference),
		"authorizationPolicyVersion":     mustRaw(result.AuthorizationPolicyVersion),
		"validationOutcome":              mustRaw(result.ValidationOutcome),
		"auditReference":                 mustRaw(result.AuditReference.String()),
		"nextAction":                     mustRaw(result.NextAction),
		"ordinaryPostingAdmission":       mustRaw(ordinaryPostingAdmission(gate)),
		"reopenPostingAdmission":         mustRaw(reopenPostingAdmission(gate)),
		"replayed":                       mustRaw(result.Replayed),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(gate.ID), AggregateVersion: int(result.AggregateVersion()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: self}, Data: data}
}

func ownerType(gate gl.PeriodPostingGate) string {
	if gate.ActiveCloseRunID != nil {
		return gl.PostingBarrierOwnerType
	}
	if gate.ActiveReopenRequestID != nil {
		return gl.ScopedReopenGateOwnerType
	}
	if gate.ActiveSoftCloseRunID == nil {
		return ""
	}
	return gl.SoftCloseGateOwnerType
}

func ownerID(gate gl.PeriodPostingGate) any {
	if gate.ActiveCloseRunID != nil {
		return gate.ActiveCloseRunID.String()
	}
	if gate.ActiveReopenRequestID != nil {
		return gate.ActiveReopenRequestID.String()
	}
	if gate.ActiveSoftCloseRunID == nil {
		return nil
	}
	return gate.ActiveSoftCloseRunID.String()
}

func ordinaryPostingAdmission(gate gl.PeriodPostingGate) map[string]string {
	if gate.GateMode == gl.PostingGateSoftClosePolicy {
		return map[string]string{"status": "policy-evaluated", "blockedAction": "ordinary-posting", "reason": "The active soft-close policy controls ordinary posting admission."}
	}
	if gate.GateMode == gl.PostingGateCloseOnly {
		if gate.ActiveReopenRequestID != nil {
			return map[string]string{"status": "blocked", "blockedAction": "ordinary-posting", "reason": "The scoped reopen correction window is closed; mandatory reclose is required."}
		}
		return map[string]string{"status": "blocked", "blockedAction": "ordinary-posting", "reason": "The posting barrier admits only authorized close postings."}
	}
	if gate.GateMode == gl.PostingGateHardClosed {
		return map[string]string{"status": "blocked", "blockedAction": "ordinary-posting", "reason": "The posting gate is hard closed; correction requires an approved reopen workflow."}
	}
	return map[string]string{"status": "allowed", "reason": "The posting gate is open."}
}

func reopenPostingAdmission(gate gl.PeriodPostingGate) map[string]string {
	if gate.GateMode != gl.PostingGateScopedReopen || gate.ActiveReopenRequestID == nil {
		return map[string]string{"status": "blocked", "postingPurpose": gl.PostingPurposeReopenCorrection, "reason": "An active scoped reopen gate is required for correction postings."}
	}
	if gate.ReopenAuthorizationExpiry == nil || !time.Now().UTC().Before(gate.ReopenAuthorizationExpiry.UTC()) {
		return map[string]string{"status": "expired", "postingPurpose": gl.PostingPurposeReopenCorrection, "reason": "The scoped reopen authorization has expired; close the gate before reclose handoff."}
	}
	return map[string]string{"status": "allowed", "postingPurpose": gl.PostingPurposeReopenCorrection, "requestId": gate.ActiveReopenRequestID.String(), "reason": "Only corrections matching the approved account or transaction-class scope are admitted."}
}

func uuidPointerString(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return value.String()
}

func mapGlEnterSoftCloseGateError(err error, correlationID uuid.UUID) generated.GlEnterSoftCloseGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glEnterSoftCloseGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The soft-close entry violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glEnterSoftCloseGateProblem(status, code, detail, correlationID, nil)
}

func mapGlExitSoftCloseGateError(err error, correlationID uuid.UUID) generated.GlExitSoftCloseGateRes {
	var validation *gl.SoftCloseGateValidationError
	if errors.As(err, &validation) {
		return glExitSoftCloseGateProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The soft-close exit violates a general-ledger rule.", correlationID, validation.Issues)
	}
	status, code, detail := softCloseGateErrorOutcome(err)
	return glExitSoftCloseGateProblem(status, code, detail, correlationID, nil)
}

func softCloseGateErrorOutcome(err error) (int, string, string) {
	switch {
	case errors.Is(err, gl.ErrSoftCloseGateAuthorizationDenied):
		return http.StatusForbidden, "AUTHORIZATION_DENIED", "The actor is not authorized for this soft-close gate action."
	case errors.Is(err, gl.ErrSoftCloseGateAuthorizationUnavailable), errors.Is(err, gl.ErrSoftCloseGateAuthorizationStale), errors.Is(err, gl.ErrSoftCloseGatePolicyUnavailable), errors.Is(err, gl.ErrSoftCloseGateReferenceUnavailable), errors.Is(err, gl.ErrSoftCloseGateNotFound), errors.Is(err, gl.ErrSoftCloseGateAuditUnavailable), errors.Is(err, gl.ErrSoftCloseGateOutboxUnavailable), errors.Is(err, gl.ErrInvalidSoftCloseGateService), errors.Is(err, gl.ErrSoftCloseGateDurableCommandFailed):
		return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The soft-close gate action could not be completed."
	case errors.Is(err, gl.ErrSoftCloseGateIdempotencyConflict):
		return http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different soft-close gate data."
	case errors.Is(err, gl.ErrSoftCloseGateCommandInProgress):
		return http.StatusConflict, "COMMAND_IN_PROGRESS", "The soft-close gate command is still being finalized. Retry with the same idempotency key."
	case errors.Is(err, gl.ErrSoftCloseGateVersionConflict), errors.Is(err, gl.ErrSoftCloseGateScopeConflict), errors.Is(err, gl.ErrSoftCloseGateOwnerConflict), errors.Is(err, gl.ErrSoftCloseGateEpochConflict), errors.Is(err, gl.ErrSoftCloseGateModeConflict), errors.Is(err, gl.ErrSoftCloseGatePolicyConflict), errors.Is(err, gl.ErrSoftCloseGateAuthorizationExpired):
		return http.StatusConflict, "VERSION_CONFLICT", "The posting gate changed or the supplied owner and control epoch do not match. Refresh and retry safely."
	case errors.Is(err, gl.ErrPostingBarrierNotReleasable), errors.Is(err, gl.ErrPostingBarrierAdmissionConflict), errors.Is(err, gl.ErrPostingBarrierWatermarkConflict), errors.Is(err, gl.ErrPostingBarrierFinalized):
		return http.StatusConflict, "BARRIER_CONFLICT", "The posting barrier changed or the requested recovery action is not allowed. Use the authoritative gate result before retrying."
	case errors.Is(err, gl.ErrSoftCloseGatePolicyRejected):
		return http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The soft-close policy was rejected by the GL policy evaluator."
	case errors.Is(err, gl.ErrInvalidSoftCloseGateCommand):
		return http.StatusBadRequest, "INVALID_REQUEST", "The soft-close gate command is invalid."
	default:
		return http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The soft-close gate action could not be completed."
	}
}

func glSoftCloseGateProblemDetails(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.ProblemDetails {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	if len(issues) > 0 {
		problem.FieldErrors = make([]generated.ProblemFieldError, len(issues))
		for index, issue := range issues {
			problem.FieldErrors[index] = generated.ProblemFieldError{Code: generated.NewOptString(issue.Code), Field: issue.Field, Message: issue.Message}
		}
	}
	return problem
}

func glEnterSoftCloseGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlEnterSoftCloseGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlEnterSoftCloseGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlEnterSoftCloseGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlEnterSoftCloseGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlEnterSoftCloseGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlEnterSoftCloseGateServiceUnavailable(problem)
		return &value
	}
}

func glExitSoftCloseGateProblem(status int, code, detail string, correlationID uuid.UUID, issues []gl.PostingIssue) generated.GlExitSoftCloseGateRes {
	problem := glSoftCloseGateProblemDetails(status, code, detail, correlationID, issues)
	switch status {
	case http.StatusBadRequest:
		value := generated.GlExitSoftCloseGateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlExitSoftCloseGateForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlExitSoftCloseGateConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlExitSoftCloseGateUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlExitSoftCloseGateServiceUnavailable(problem)
		return &value
	}
}
