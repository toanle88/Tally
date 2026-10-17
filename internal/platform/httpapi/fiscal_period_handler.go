package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

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

type fpmStartRecloseCommandData struct {
	FiscalPeriodID               string `json:"fiscalPeriodId"`
	ReopenRequestID              string `json:"reopenRequestId"`
	ExpectedPeriodVersion        int64  `json:"expectedPeriodVersion"`
	ExpectedReopenRequestVersion int64  `json:"expectedReopenRequestVersion"`
	ExpectedGateVersion          int64  `json:"expectedGateVersion"`
}

type fpmApplyPostingGateResultCommandData struct {
	FiscalPeriodID               string                `json:"fiscalPeriodId"`
	SoftCloseRunID               string                `json:"softCloseRunId"`
	CloseRunID                   string                `json:"closeRunId"`
	ReopenRequestID              string                `json:"reopenRequestId"`
	ExpectedReopenRequestVersion int64                 `json:"expectedReopenRequestVersion"`
	SoftCloseControlEpoch        int64                 `json:"softCloseControlEpoch"`
	PolicyVersion                string                `json:"policyVersion"`
	ExpectedGateVersion          int64                 `json:"expectedGateVersion"`
	PostingGateResult            json.RawMessage       `json:"postingGateResult"`
	Gate                         *gl.PeriodPostingGate `json:"gate"`
	Operation                    string                `json:"operation"`
	DecisionReference            uuid.UUID             `json:"decisionReference"`
	PolicyReference              string                `json:"policyReference"`
	AuthorizationPolicyVersion   string                `json:"authorizationPolicyVersion"`
	ValidationOutcome            string                `json:"validationOutcome"`
	AuditReference               uuid.UUID             `json:"auditReference"`
	NextAction                   string                `json:"nextAction"`
}

type fpmCloseRunCommandData struct {
	FiscalPeriodID              string    `json:"fiscalPeriodId"`
	SoftCloseRunID              string    `json:"softCloseRunId"`
	CloseRunID                  string    `json:"closeRunId"`
	SoftCloseControlEpoch       int64     `json:"softCloseControlEpoch"`
	PolicyVersion               string    `json:"policyVersion"`
	ApprovalPolicyVersion       string    `json:"approvalPolicyVersion"`
	ExpectedGateVersion         int64     `json:"expectedGateVersion"`
	ExpectedCloseRunVersion     int64     `json:"expectedCloseRunVersion"`
	ExceptionID                 string    `json:"exceptionId"`
	ApprovalRequestID           string    `json:"approvalRequestId"`
	DecisionID                  string    `json:"decisionId"`
	DecisionVersion             int64     `json:"decisionVersion"`
	Outcome                     string    `json:"outcome"`
	DecisionReason              string    `json:"decisionReason"`
	DecisionReference           string    `json:"decisionReference"`
	ApproverUserID              string    `json:"approverUserId"`
	SeparationOfDutiesReference string    `json:"separationOfDutiesReference"`
	CandidateFingerprint        string    `json:"candidateFingerprint"`
	SubjectVersion              int64     `json:"subjectVersion"`
	FinalLedgerWatermark        int64     `json:"finalLedgerWatermark"`
	RecordedAt                  time.Time `json:"recordedAt"`
	Reason                      string    `json:"reason"`
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

func (handler IdentityHandler) FpmStartReclose(ctx context.Context, request *generated.CommandRequest, params generated.FpmStartRecloseParams) (generated.FpmStartRecloseRes, error) {
	correlationID := fpmStartRecloseCorrelation(params)
	if handler.FiscalPeriodService == nil {
		return fpmStartRecloseProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmStartRecloseProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The reclose actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return fpmStartRecloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reclose command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmStartRecloseCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmStartRecloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reclose command data is invalid.", correlationID), nil
	}
	periodID, err := parseFPMUUID(payload.FiscalPeriodID)
	if err != nil {
		return fpmStartRecloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID), nil
	}
	reopenRequestID, err := parseFPMUUID(payload.ReopenRequestID)
	if err != nil {
		return fpmStartRecloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID), nil
	}
	expectedPeriodVersion := payload.ExpectedPeriodVersion
	if expectedPeriodVersion < 1 {
		expectedPeriodVersion = int64(request.ExpectedVersion.Value)
	}
	if expectedPeriodVersion < 1 || payload.ExpectedReopenRequestVersion < 1 || payload.ExpectedGateVersion < 1 {
		return fpmStartRecloseProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected period, reopen-request, and gate versions are required.", correlationID), nil
	}
	result, err := handler.FiscalPeriodService.StartReclose(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, fiscalperiod.StartRecloseCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), FiscalPeriodID: periodID, ReopenRequestID: reopenRequestID,
		ExpectedPeriodVersion: expectedPeriodVersion, ExpectedReopenRequestVersion: payload.ExpectedReopenRequestVersion, ExpectedGateVersion: payload.ExpectedGateVersion,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapFPMStartRecloseError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/start-reclose"), nil
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
	result, err := fpmPostingGateResult(payload)
	if err != nil {
		return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The authoritative posting-gate result is invalid.", correlationID), nil
	}
	command := fiscalperiod.ApplyPostingGateResultCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: scopeID, FiscalPeriodID: periodID,
		ExpectedPeriodVersion: int64(request.ExpectedVersion.Value), ExpectedGateVersion: payload.ExpectedGateVersion,
		Result: result, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	if result.Operation == gl.ScopedReopenGateCloseOperation || result.Operation == gl.OperationalReopenGateCloseOperation {
		command.ReopenRequestID, err = parseFPMUUID(payload.ReopenRequestID)
		if err != nil || payload.ExpectedReopenRequestVersion < 1 || payload.ExpectedGateVersion < 1 {
			return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request and expected gate versions are required for a reopen-gate close result.", correlationID), nil
		}
		command.ExpectedReopenRequestVersion = payload.ExpectedReopenRequestVersion
	} else {
		softCloseRunID, parseErr := parseFPMUUID(payload.SoftCloseRunID)
		if parseErr != nil {
			return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The soft-close run identifier is invalid.", correlationID), nil
		}
		closeRunID, parseErr := parseFPMUUID(payload.CloseRunID)
		if parseErr != nil {
			return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close-run identifier is invalid.", correlationID), nil
		}
		if payload.ExpectedGateVersion < 1 || payload.SoftCloseControlEpoch < 1 || strings.TrimSpace(payload.PolicyVersion) == "" {
			return fpmApplyPostingGateResultProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected gate version, control epoch, and policy version are required.", correlationID), nil
		}
		command.SoftCloseRunID = softCloseRunID
		command.CloseRunID = closeRunID
		command.SoftCloseControlEpoch = payload.SoftCloseControlEpoch
		command.SoftClosePolicyVersion = payload.PolicyVersion
	}
	resultValue, err := handler.FiscalPeriodService.ApplyPostingGateResult(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapFPMApplyPostingGateResultError(err, correlationID), nil
	}
	return establishedFPMResult(resultValue, correlationID, "/api/v1/fiscal-periods/actions/apply-posting-gate-result"), nil
}

func (handler IdentityHandler) FpmResumeCloseRun(ctx context.Context, request *generated.CommandRequest, params generated.FpmResumeCloseRunParams) (generated.FpmResumeCloseRunRes, error) {
	correlationID := fpmCloseCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmResumeCloseRunProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmResumeCloseRunProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The close-run actor is not authorized.", correlationID), nil
	}
	payload, base, err := decodeFPMCloseRunCommand(request, params.IdempotencyKey, correlationID)
	if err != nil {
		return fpmResumeCloseRunProblem(http.StatusBadRequest, "INVALID_REQUEST", "The resume close-run command is invalid.", correlationID), nil
	}
	result, err := handler.FiscalPeriodService.ResumeCloseRun(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, base)
	if err != nil {
		if errors.Is(err, fiscalperiod.ErrCloseTaskFailed) && result.State.CloseRun != nil {
			return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/resume-close-run"), nil
		}
		return mapFPMResumeCloseRunError(err, correlationID), nil
	}
	_ = payload
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/resume-close-run"), nil
}

func (handler IdentityHandler) FpmAbortCloseRun(ctx context.Context, request *generated.CommandRequest, params generated.FpmAbortCloseRunParams) (generated.FpmAbortCloseRunRes, error) {
	correlationID := fpmCloseCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmAbortCloseRunProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmAbortCloseRunProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The close-run actor is not authorized.", correlationID), nil
	}
	payload, base, err := decodeFPMCloseRunCommand(request, params.IdempotencyKey, correlationID)
	if err != nil || strings.TrimSpace(payload.Reason) == "" {
		return fpmAbortCloseRunProblem(http.StatusBadRequest, "INVALID_REQUEST", "The abort close-run command requires a reason and current run evidence.", correlationID), nil
	}
	result, err := handler.FiscalPeriodService.AbortCloseRun(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, fiscalperiod.AbortCloseRunCommand{ResumeCloseRunCommand: base, Reason: payload.Reason})
	if err != nil {
		return mapFPMAbortCloseRunError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/abort-close-run"), nil
}

func (handler IdentityHandler) FpmApplyCloseExceptionApprovalDecision(ctx context.Context, request *generated.CommandRequest, params generated.FpmApplyCloseExceptionApprovalDecisionParams) (generated.FpmApplyCloseExceptionApprovalDecisionRes, error) {
	correlationID := fpmCloseCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmApplyCloseExceptionApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmApplyCloseExceptionApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The close-approval actor is not authorized.", correlationID), nil
	}
	payload, base, err := decodeFPMCloseRunCommand(request, params.IdempotencyKey, correlationID)
	if err != nil {
		return fpmApplyCloseExceptionApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close-exception approval decision is invalid.", correlationID), nil
	}
	exceptionID, err := parseFPMUUID(payload.ExceptionID)
	if err != nil {
		return fpmApplyCloseExceptionApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close exception identifier is invalid.", correlationID), nil
	}
	decision, err := fpmCloseApprovalFields(payload)
	if err != nil {
		return fpmApplyCloseExceptionApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The immutable Workflow decision evidence is incomplete.", correlationID), nil
	}
	command := fiscalperiod.ApplyCloseExceptionApprovalDecisionCommand{ResumeCloseRunCommand: base, ExceptionID: exceptionID, ApprovalRequestID: decision.RequestID, DecisionID: decision.DecisionID, DecisionVersion: decision.DecisionVersion, Outcome: decision.Outcome, DecisionReason: decision.Reason, PolicyVersion: decision.PolicyVersion, SubjectVersion: decision.SubjectVersion, CandidateFingerprint: decision.CandidateFingerprint, DecisionReference: decision.DecisionReference, ApproverUserID: decision.ApproverUserID, SeparationOfDutiesReference: decision.SeparationOfDutiesReference, RecordedAt: decision.RecordedAt}
	result, err := handler.FiscalPeriodService.ApplyCloseExceptionApprovalDecision(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapFPMApplyCloseExceptionApprovalDecisionError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/apply-close-exception-approval-decision"), nil
}

func (handler IdentityHandler) FpmApplyCloseApprovalDecision(ctx context.Context, request *generated.CommandRequest, params generated.FpmApplyCloseApprovalDecisionParams) (generated.FpmApplyCloseApprovalDecisionRes, error) {
	correlationID := fpmCloseCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmApplyCloseApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmApplyCloseApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The close-approval actor is not authorized.", correlationID), nil
	}
	payload, base, err := decodeFPMCloseRunCommand(request, params.IdempotencyKey, correlationID)
	if err != nil {
		return fpmApplyCloseApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close approval decision is invalid.", correlationID), nil
	}
	decision, err := fpmCloseApprovalFields(payload)
	if err != nil || payload.FinalLedgerWatermark < 0 || (decision.Outcome == fiscalperiod.CloseApprovalApproved && payload.FinalLedgerWatermark < 1) {
		return fpmApplyCloseApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The final approval and ledger watermark evidence are incomplete.", correlationID), nil
	}
	command := fiscalperiod.ApplyCloseApprovalDecisionCommand{ResumeCloseRunCommand: base, ApprovalRequestID: decision.RequestID, DecisionID: decision.DecisionID, DecisionVersion: decision.DecisionVersion, Outcome: decision.Outcome, DecisionReason: decision.Reason, PolicyVersion: decision.PolicyVersion, SubjectVersion: decision.SubjectVersion, CandidateFingerprint: decision.CandidateFingerprint, DecisionReference: decision.DecisionReference, ApproverUserID: decision.ApproverUserID, SeparationOfDutiesReference: decision.SeparationOfDutiesReference, FinalLedgerWatermark: payload.FinalLedgerWatermark, RecordedAt: decision.RecordedAt}
	result, err := handler.FiscalPeriodService.ApplyCloseApprovalDecision(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapFPMApplyCloseApprovalDecisionError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/apply-close-approval-decision"), nil
}

func decodeFPMCloseRunCommand(request *generated.CommandRequest, idempotencyKey string, correlationID uuid.UUID) (fpmCloseRunCommandData, fiscalperiod.ResumeCloseRunCommand, error) {
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return fpmCloseRunCommandData{}, fiscalperiod.ResumeCloseRunCommand{}, errors.New("command, scope, and expected period version are required")
	}
	var payload fpmCloseRunCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmCloseRunCommandData{}, fiscalperiod.ResumeCloseRunCommand{}, err
	}
	periodID, err := parseFPMUUID(payload.FiscalPeriodID)
	if err != nil {
		return fpmCloseRunCommandData{}, fiscalperiod.ResumeCloseRunCommand{}, err
	}
	softCloseRunID, err := parseFPMUUID(payload.SoftCloseRunID)
	if err != nil {
		return fpmCloseRunCommandData{}, fiscalperiod.ResumeCloseRunCommand{}, err
	}
	closeRunID, err := parseFPMUUID(payload.CloseRunID)
	if err != nil {
		return fpmCloseRunCommandData{}, fiscalperiod.ResumeCloseRunCommand{}, err
	}
	policyVersion := strings.TrimSpace(payload.PolicyVersion)
	closeRunVersion := payload.ExpectedCloseRunVersion
	if closeRunVersion < 1 {
		closeRunVersion = int64(request.ExpectedVersion.Value)
	}
	if payload.ExpectedGateVersion < 1 || payload.SoftCloseControlEpoch < 1 || policyVersion == "" || closeRunVersion < 1 {
		return fpmCloseRunCommandData{}, fiscalperiod.ResumeCloseRunCommand{}, errors.New("current gate, close-run version, epoch, and policy are required")
	}
	return payload, fiscalperiod.ResumeCloseRunCommand{CommandID: uuid.UUID(request.CommandId), AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), FiscalPeriodID: periodID, SoftCloseRunID: softCloseRunID, CloseRunID: closeRunID, ExpectedPeriodVersion: int64(request.ExpectedVersion.Value), ExpectedCloseRunVersion: closeRunVersion, ExpectedGateVersion: payload.ExpectedGateVersion, SoftCloseControlEpoch: payload.SoftCloseControlEpoch, SoftClosePolicyVersion: policyVersion, IdempotencyKey: idempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String()}, nil
}

func fpmCloseApprovalFields(payload fpmCloseRunCommandData) (fiscalperiod.CloseApprovalDecision, error) {
	requestID, err := parseFPMUUID(payload.ApprovalRequestID)
	if err != nil {
		return fiscalperiod.CloseApprovalDecision{}, err
	}
	decisionID, err := parseFPMUUID(payload.DecisionID)
	if err != nil {
		return fiscalperiod.CloseApprovalDecision{}, err
	}
	decisionReference, err := parseFPMUUID(payload.DecisionReference)
	if err != nil {
		return fiscalperiod.CloseApprovalDecision{}, err
	}
	approverID, err := parseFPMUUID(payload.ApproverUserID)
	if err != nil {
		return fiscalperiod.CloseApprovalDecision{}, err
	}
	policyVersion := strings.TrimSpace(payload.ApprovalPolicyVersion)
	if policyVersion == "" {
		policyVersion = strings.TrimSpace(payload.PolicyVersion)
	}
	if policyVersion == "" || payload.SubjectVersion < 1 || payload.RecordedAt.IsZero() {
		return fiscalperiod.CloseApprovalDecision{}, errors.New("policy version, subject version, and recorded-at evidence are required")
	}
	return fiscalperiod.CloseApprovalDecision{RequestID: requestID, DecisionID: decisionID, DecisionVersion: payload.DecisionVersion, Outcome: strings.TrimSpace(payload.Outcome), Reason: strings.TrimSpace(payload.DecisionReason), PolicyVersion: policyVersion, SubjectVersion: payload.SubjectVersion, CandidateFingerprint: strings.TrimSpace(payload.CandidateFingerprint), ApproverUserID: approverID, DecisionReference: decisionReference, SeparationOfDutiesReference: strings.TrimSpace(payload.SeparationOfDutiesReference), RecordedAt: payload.RecordedAt}, nil
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
	var closeRun fiscalperiod.CloseRun
	if result.State.CloseRun != nil {
		closeRun = *result.State.CloseRun
	}
	gate := result.GateResult
	if gate.Gate.ID == uuid.Nil && closeRun.FinalizationGateResult != nil {
		gate = *closeRun.FinalizationGateResult
	}
	if gate.Gate.ID == uuid.Nil && closeRun.ReleaseGateResult != nil {
		gate = *closeRun.ReleaseGateResult
	}
	if gate.Gate.ID == uuid.Nil && closeRun.BarrierEvidence != nil {
		gate = fiscalperiodGateEvidenceResult(closeRun)
	}
	nextAction := gate.NextAction
	var reopenRequest *fiscalperiod.ReopenRequest
	if result.State.ReopenRequest != nil {
		reopenRequest = result.State.ReopenRequest
	}
	var authority *fiscalperiod.ControlAuthority
	if result.State.CloseRun != nil {
		authority = closeRun.ControlAuthority
	} else {
		authority = result.State.SoftClose.ControlAuthority
	}
	if nextAction == "" {
		if reopenRequest != nil {
			switch reopenRequest.Status {
			case fiscalperiod.ReopenRequestPendingApproval:
				nextAction = "await-reopen-approval"
			case fiscalperiod.ReopenRequestApproved, fiscalperiod.ReopenRequestGateOpening:
				nextAction = "open-reopen-gate"
			case fiscalperiod.ReopenRequestActive:
				nextAction = "perform-authorized-reopen-work"
			case fiscalperiod.ReopenRequestExpiredPendingClosure, fiscalperiod.ReopenRequestGateClosing, fiscalperiod.ReopenRequestRecloseHandoff:
				nextAction = "start-reclose"
			case fiscalperiod.ReopenRequestRecloseInProgress:
				nextAction = "resume-close-run"
			case fiscalperiod.ReopenRequestCompletedNoChange, fiscalperiod.ReopenRequestCompleted:
				nextAction = "review-reopen-state"
			case fiscalperiod.ReopenRequestRejected:
				nextAction = "request-new-reopen"
			default:
				nextAction = "review-reopen-state"
			}
		}
	}
	if nextAction == "" {
		switch {
		case closeRun.SealStatus == fiscalperiod.CloseRunSealPending:
			nextAction = "await-audit-seal"
		case closeRun.SealStatus == fiscalperiod.CloseRunSealFailed:
			nextAction = "retry-audit-seal"
		case closeRun.Status == fiscalperiod.CloseRunAwaitingApproval:
			nextAction = "apply-close-approval"
		case closeRun.Status == fiscalperiod.CloseRunCompleted:
			nextAction = "await-audit-seal"
		default:
			nextAction = "resume-close-run"
		}
	}
	data := generated.EstablishedResultData{
		"periodId": mustRaw(period.ID.String()), "accountingScopeId": mustRaw(period.AccountingScopeID.String()), "periodState": mustRaw(period.State),
		"periodStateVersion": mustRaw(period.StateVersion), "periodAggregateVersion": mustRaw(period.AggregateVersion),
		"softCloseRunId": mustRaw(result.State.SoftClose.ID.String()), "softCloseStatus": mustRaw(result.State.SoftClose.Status),
		"controlEpoch": mustRaw(result.State.ControlEpoch.Number), "controlEpochStatus": mustRaw(result.State.ControlEpoch.Status),
		"closeRunId": mustRaw(closeRun.ID.String()), "closeRunStatus": mustRaw(closeRun.Status), "closeRunAggregateVersion": mustRaw(closeRun.AggregateVersion), "closeRunCommandFingerprint": mustRaw(closeRun.CommandFingerprint),
		"closeType": mustRaw(closeRun.CloseType),
		"gateMode":  mustRaw(gate.Gate.GateMode), "gateVersion": mustRaw(gate.Gate.GateVersion),
		"barrierLedgerPosition": mustRaw(gate.Gate.BarrierLedgerPosition), "frozenAdmissionSummary": mustRaw(closeRun.BarrierEvidence),
		"validationOutcome": mustRaw(gate.ValidationOutcome), "decisionReference": mustRaw(gate.DecisionReference.String()),
		"policyReference": mustRaw(gate.PolicyReference), "auditReference": mustRaw(gate.AuditReference.String()),
		"nextAction": mustRaw(nextAction), "replayed": mustRaw(result.Replayed), "steps": mustRaw(closeRun.Steps), "exceptions": mustRaw(closeRun.Exceptions), "approvalRequestIds": mustRaw(closeRun.ApprovalRequestIDs), "approvalDecisions": mustRaw(closeRun.ApprovalDecisions), "approvalPolicyVersion": mustRaw(closeRun.ApprovalPolicyVersion), "finalApproval": mustRaw(closeRun.FinalApproval), "finalLedgerWatermark": mustRaw(closeRun.FinalLedgerWatermark), "finalizationGateResult": mustRaw(closeRun.FinalizationGateResult), "releaseGateResult": mustRaw(closeRun.ReleaseGateResult), "sealStatus": mustRaw(closeRun.SealStatus), "sealRequestId": mustRaw(closeRun.SealRequestID), "sealProofReference": mustRaw(closeRun.SealProofReference), "sealFailureCode": mustRaw(closeRun.SealFailureCode), "sealFailureDetail": mustRaw(closeRun.SealFailureDetail), "lastFailureCode": mustRaw(closeRun.LastFailureCode), "lastFailureDetail": mustRaw(closeRun.LastFailureDetail),
		"controlAuthority": mustRaw(authority), "authoritativeGateStatus": mustRaw(gate.Gate), "pendingReconciliation": mustRaw(false), "replayStatus": mustRaw(map[bool]string{true: "replayed", false: "established"}[result.Replayed]), "exceptionExtensionHistory": mustRaw(exceptionExtensionHistory(closeRun.Exceptions)),
	}
	if authority != nil {
		data["ownerId"] = mustRaw(authority.CurrentOwner.UserID.String())
		data["ownerSubjectReference"] = mustRaw(authority.CurrentOwner.SubjectReference)
		data["authorityEpoch"] = mustRaw(authority.AuthorityEpoch)
		data["authorityVersion"] = mustRaw(authority.Version)
		data["authorityExpiresAt"] = mustRaw(authority.ExpiresAt)
		data["controlCutoffAt"] = mustRaw(authority.CutoffAt)
		data["ownershipHistory"] = mustRaw(authority.OwnershipHistory)
	}
	if reopenRequest != nil {
		data["reopenRequestId"] = mustRaw(reopenRequest.ID.String())
		data["reopenStatus"] = mustRaw(reopenRequest.Status)
		data["reopenMode"] = mustRaw(reopenRequest.Mode)
		data["reopenRequest"] = mustRaw(reopenRequest)
		data["reopenScope"] = mustRaw(reopenRequest.Scope)
		data["authorizationExpiry"] = mustRaw(reopenRequest.AuthorizationExpiry)
		data["controlAuthorityEpoch"] = mustRaw(reopenRequest.ControlAuthorityEpoch)
		var approvalRequestID any
		if reopenRequest.ApprovalRequestID != uuid.Nil {
			approvalRequestID = reopenRequest.ApprovalRequestID.String()
		}
		data["approvalRequestId"] = mustRaw(approvalRequestID)
		data["approvalDecision"] = mustRaw(reopenRequest.ApprovalDecision)
		data["reopenGateResult"] = mustRaw(reopenRequest.GateResult)
		data["reopenGateVersion"] = mustRaw(reopenRequest.GateVersion)
	}
	if closeRun.BarrierEvidence != nil {
		data["priorControlProcessId"] = mustRaw(closeRun.BarrierEvidence.PriorControlProcessID)
		data["priorControlEpoch"] = mustRaw(closeRun.BarrierEvidence.PriorControlEpoch)
		data["frozenAdmissionSummary"] = mustRaw(closeRun.BarrierEvidence.FrozenSummary)
	}
	processID := closeRun.ID
	if processID == uuid.Nil {
		processID = result.State.SoftClose.ID
	}
	if processID == uuid.Nil && reopenRequest != nil {
		processID = reopenRequest.ID
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(period.ID), AggregateVersion: int(period.AggregateVersion), ProcessId: generated.NewOptNilUUID(generated.UUID(processID)), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: self}, Data: data}
}

func exceptionExtensionHistory(exceptions []fiscalperiod.CloseException) []fiscalperiod.CloseExceptionExtension {
	var history []fiscalperiod.CloseExceptionExtension
	for _, exception := range exceptions {
		history = append(history, exception.ExtensionHistory...)
	}
	return history
}

func fiscalperiodGateEvidenceResult(run fiscalperiod.CloseRun) gl.SoftCloseGateResult {
	if run.BarrierEvidence == nil {
		return gl.SoftCloseGateResult{}
	}
	return gl.SoftCloseGateResult{Operation: run.BarrierEvidence.Operation, Gate: run.BarrierEvidence.AuthoritativeResult.Gate, DecisionReference: run.BarrierEvidence.DecisionReference, PolicyReference: run.BarrierEvidence.PolicyReference, AuthorizationPolicyVersion: run.BarrierEvidence.AuthorizationPolicyVersion, AuditReference: run.BarrierEvidence.AuditReference, ValidationOutcome: run.BarrierEvidence.ValidationOutcome, NextAction: run.BarrierEvidence.NextAction}
}

func fpmStartHardCloseCorrelation(params generated.FpmStartHardCloseParams) uuid.UUID {
	if params.XCorrelationID.Set && uuid.UUID(params.XCorrelationID.Value) != uuid.Nil {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func fpmStartRecloseCorrelation(params generated.FpmStartRecloseParams) uuid.UUID {
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

func fpmCloseCorrelation(value generated.OptUUID) uuid.UUID {
	if value.Set && uuid.UUID(value.Value) != uuid.Nil {
		return uuid.UUID(value.Value)
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

func mapFPMStartRecloseError(err error, correlationID uuid.UUID) generated.FpmStartRecloseRes {
	return mapFPMError(err, correlationID, fpmStartRecloseProblem)
}

func mapFPMApplyPostingGateResultError(err error, correlationID uuid.UUID) generated.FpmApplyPostingGateResultRes {
	return mapFPMError(err, correlationID, fpmApplyPostingGateResultProblem)
}

func mapFPMResumeCloseRunError(err error, correlationID uuid.UUID) generated.FpmResumeCloseRunRes {
	return mapFPMError(err, correlationID, fpmResumeCloseRunProblem)
}

func mapFPMAbortCloseRunError(err error, correlationID uuid.UUID) generated.FpmAbortCloseRunRes {
	return mapFPMError(err, correlationID, fpmAbortCloseRunProblem)
}

func mapFPMApplyCloseExceptionApprovalDecisionError(err error, correlationID uuid.UUID) generated.FpmApplyCloseExceptionApprovalDecisionRes {
	return mapFPMError(err, correlationID, fpmApplyCloseExceptionApprovalDecisionProblem)
}

func mapFPMApplyCloseApprovalDecisionError(err error, correlationID uuid.UUID) generated.FpmApplyCloseApprovalDecisionRes {
	return mapFPMError(err, correlationID, fpmApplyCloseApprovalDecisionProblem)
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
	case errors.Is(err, fiscalperiod.ErrCloseTaskUnavailable), errors.Is(err, fiscalperiod.ErrCloseDecisionUnavailable):
		code, detail = "DEPENDENCY_UNAVAILABLE", "A required close dependency is unavailable. The close run remains restrictive and retryable."
	case errors.Is(err, fiscalperiod.ErrCloseTaskFailed):
		status, code, detail = http.StatusConflict, "CLOSE_TASK_FAILED", "A mandatory close task failed. Resolve the task and resume the existing close run."
	case errors.Is(err, fiscalperiod.ErrCloseDecisionRejected):
		status, code, detail = http.StatusConflict, "APPROVAL_INVALIDATED", "The approval decision is stale, mismatched, rejected, or violates the current close-run evidence."
	case errors.Is(err, fiscalperiod.ErrControlRecoveryDecisionRejected):
		status, code, detail = http.StatusConflict, "APPROVAL_INVALIDATED", "The recovery approval decision is stale, mismatched, rejected, or violates the current recovery candidate evidence."
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodControlAuthorityMissing), errors.Is(err, fiscalperiod.ErrFiscalPeriodRecoveryNotEligible):
		status, code, detail = http.StatusConflict, "RECOVERY_NOT_ELIGIBLE", "The process has no eligible control authority or is not in an interrupted recoverable state."
	case errors.Is(err, fiscalperiod.ErrCloseExceptionExtensionRejected):
		status, code, detail = http.StatusConflict, "EXCEPTION_EXTENSION_REJECTED", "The close exception or close run cannot be extended in its current state."
	case errors.Is(err, fiscalperiod.ErrReopenDecisionRejected):
		status, code, detail = http.StatusConflict, "APPROVAL_INVALIDATED", "The reopen approval decision is stale, mismatched, rejected, or violates the current reopen evidence."
	case errors.Is(err, fiscalperiod.ErrReopenGateRejected):
		status, code, detail = http.StatusUnprocessableEntity, "REOPEN_GATE_REJECTED", "GL rejected the reopen gate request. The period remains restrictive."
	case errors.Is(err, fiscalperiod.ErrReopenDecisionUnavailable), errors.Is(err, fiscalperiod.ErrReopenGateUnavailable), errors.Is(err, fiscalperiod.ErrControlRecoveryDecisionUnavailable):
		code, detail = "DEPENDENCY_UNAVAILABLE", "A required period-control dependency is unavailable. The period remains restrictive and the request is retryable."
	case errors.Is(err, fiscalperiod.ErrCloseFinalizationPending):
		status, code, detail = http.StatusConflict, "FINALIZATION_PENDING", "The authoritative finalization result is not yet confirmed. Resume the existing close run."
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

func fpmStartRecloseProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmStartRecloseRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmStartRecloseBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmStartRecloseForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmStartRecloseConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmStartRecloseUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmStartRecloseServiceUnavailable(problem)
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

func fpmResumeCloseRunProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmResumeCloseRunRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmResumeCloseRunBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmResumeCloseRunForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmResumeCloseRunConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmResumeCloseRunUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmResumeCloseRunServiceUnavailable(problem)
		return &value
	}
}

func fpmAbortCloseRunProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmAbortCloseRunRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmAbortCloseRunBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmAbortCloseRunForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmAbortCloseRunConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmAbortCloseRunUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmAbortCloseRunServiceUnavailable(problem)
		return &value
	}
}

func fpmApplyCloseExceptionApprovalDecisionProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmApplyCloseExceptionApprovalDecisionRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmApplyCloseExceptionApprovalDecisionBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmApplyCloseExceptionApprovalDecisionForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmApplyCloseExceptionApprovalDecisionConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmApplyCloseExceptionApprovalDecisionUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmApplyCloseExceptionApprovalDecisionServiceUnavailable(problem)
		return &value
	}
}

func fpmApplyCloseApprovalDecisionProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmApplyCloseApprovalDecisionRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmApplyCloseApprovalDecisionBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmApplyCloseApprovalDecisionForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmApplyCloseApprovalDecisionConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmApplyCloseApprovalDecisionUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmApplyCloseApprovalDecisionServiceUnavailable(problem)
		return &value
	}
}
