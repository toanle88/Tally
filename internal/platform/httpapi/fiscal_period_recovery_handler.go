package httpapi

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

type fpmTakeOverPeriodControlCommandData struct {
	FiscalPeriodID                  string    `json:"fiscalPeriodId"`
	Target                          string    `json:"target"`
	ProcessID                       string    `json:"processId"`
	ExpectedProcessVersion          int64     `json:"expectedProcessVersion"`
	ExpectedGateVersion             int64     `json:"expectedGateVersion"`
	ExpectedControlAuthorityVersion int64     `json:"expectedControlAuthorityVersion"`
	ExpectedControlAuthorityEpoch   int64     `json:"expectedControlAuthorityEpoch"`
	Reason                          string    `json:"reason"`
	EvidenceReference               string    `json:"evidenceReference"`
	NewAuthorityExpiresAt           time.Time `json:"newAuthorityExpiresAt"`
	ApprovalRequestID               string    `json:"approvalRequestId"`
	DecisionID                      string    `json:"decisionId"`
	DecisionVersion                 int64     `json:"decisionVersion"`
	Outcome                         string    `json:"outcome"`
	DecisionReason                  string    `json:"decisionReason"`
	ApprovalPolicyVersion           string    `json:"approvalPolicyVersion"`
	SubjectVersion                  int64     `json:"subjectVersion"`
	CandidateFingerprint            string    `json:"candidateFingerprint"`
	DecisionReference               string    `json:"decisionReference"`
	ApproverUserID                  string    `json:"approverUserId"`
	SeparationOfDutiesReference     string    `json:"separationOfDutiesReference"`
	RecordedAt                      time.Time `json:"recordedAt"`
}

type fpmExtendCloseExceptionCommandData struct {
	FiscalPeriodID                  string    `json:"fiscalPeriodId"`
	CloseRunID                      string    `json:"closeRunId"`
	ExceptionID                     string    `json:"exceptionId"`
	ExpectedCloseRunVersion         int64     `json:"expectedCloseRunVersion"`
	ExpectedGateVersion             int64     `json:"expectedGateVersion"`
	ExpectedControlAuthorityVersion int64     `json:"expectedControlAuthorityVersion"`
	ExpectedControlAuthorityEpoch   int64     `json:"expectedControlAuthorityEpoch"`
	Justification                   string    `json:"justification"`
	NewDueAt                        time.Time `json:"newDueAt"`
	NewExpiresAt                    time.Time `json:"newExpiresAt"`
	EvidenceReference               string    `json:"evidenceReference"`
	ApprovalRequestID               string    `json:"approvalRequestId"`
	DecisionID                      string    `json:"decisionId"`
	DecisionVersion                 int64     `json:"decisionVersion"`
	Outcome                         string    `json:"outcome"`
	DecisionReason                  string    `json:"decisionReason"`
	ApprovalPolicyVersion           string    `json:"approvalPolicyVersion"`
	SubjectVersion                  int64     `json:"subjectVersion"`
	CandidateFingerprint            string    `json:"candidateFingerprint"`
	DecisionReference               string    `json:"decisionReference"`
	ApproverUserID                  string    `json:"approverUserId"`
	SeparationOfDutiesReference     string    `json:"separationOfDutiesReference"`
	RecordedAt                      time.Time `json:"recordedAt"`
}

func (handler IdentityHandler) FpmTakeOverPeriodControl(ctx context.Context, request *generated.CommandRequest, params generated.FpmTakeOverPeriodControlParams) (generated.FpmTakeOverPeriodControlRes, error) {
	ctx, finish := beginFPMRecoveryTelemetry(handler.Instrumentation, ctx, "fpmTakeOverPeriodControl")
	outcome := "failed"
	defer func() { finish(outcome) }()
	correlationID := fpmRecoveryCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmTakeOverPeriodControlProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmTakeOverPeriodControlProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The period-control recovery actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set || request.ExpectedVersion.Value < 1 {
		return fpmTakeOverPeriodControlProblem(http.StatusBadRequest, "INVALID_REQUEST", "The takeover command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmTakeOverPeriodControlCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmTakeOverPeriodControlProblem(http.StatusBadRequest, "INVALID_REQUEST", "The period-control takeover data is invalid.", correlationID), nil
	}
	periodID, processID, approval, err := decodeFPMRecoveryEvidence(payload.FiscalPeriodID, payload.ProcessID, payload.ApprovalRequestID, payload.DecisionID, payload.DecisionReference, payload.ApproverUserID, payload.RecordedAt, payload.DecisionVersion, payload.Outcome, payload.DecisionReason, payload.ApprovalPolicyVersion, payload.SubjectVersion, payload.CandidateFingerprint, payload.SeparationOfDutiesReference)
	if err != nil {
		return fpmTakeOverPeriodControlProblem(http.StatusBadRequest, "INVALID_REQUEST", "The takeover identifiers or approval evidence are invalid.", correlationID), nil
	}
	command := fiscalperiod.TakeOverPeriodControlCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), FiscalPeriodID: periodID,
		Target: payload.Target, ProcessID: processID, ExpectedPeriodVersion: int64(request.ExpectedVersion.Value), ExpectedProcessVersion: payload.ExpectedProcessVersion,
		ExpectedGateVersion: payload.ExpectedGateVersion, ExpectedControlAuthorityVersion: payload.ExpectedControlAuthorityVersion, ExpectedControlAuthorityEpoch: payload.ExpectedControlAuthorityEpoch,
		Reason: payload.Reason, EvidenceReference: payload.EvidenceReference, NewAuthorityExpiresAt: payload.NewAuthorityExpiresAt, Approval: approval,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.FiscalPeriodService.TakeOverPeriodControl(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		outcome = fpmRecoveryOutcome(err)
		return mapFPMTakeOverPeriodControlError(err, correlationID), nil
	}
	outcome = map[bool]string{true: "replayed", false: "established"}[result.Replayed]
	observeFPMRecoveryAging(handler.Instrumentation, ctx, "fpmTakeOverPeriodControl", result)
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/take-over-period-control"), nil
}

func (handler IdentityHandler) FpmExtendCloseException(ctx context.Context, request *generated.CommandRequest, params generated.FpmExtendCloseExceptionParams) (generated.FpmExtendCloseExceptionRes, error) {
	ctx, finish := beginFPMRecoveryTelemetry(handler.Instrumentation, ctx, "fpmExtendCloseException")
	outcome := "failed"
	defer func() { finish(outcome) }()
	correlationID := fpmRecoveryCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmExtendCloseExceptionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmExtendCloseExceptionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The close-exception recovery actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set || request.ExpectedVersion.Value < 1 {
		return fpmExtendCloseExceptionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The exception-extension command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmExtendCloseExceptionCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmExtendCloseExceptionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close-exception extension data is invalid.", correlationID), nil
	}
	periodID, closeRunID, approval, err := decodeFPMRecoveryEvidence(payload.FiscalPeriodID, payload.CloseRunID, payload.ApprovalRequestID, payload.DecisionID, payload.DecisionReference, payload.ApproverUserID, payload.RecordedAt, payload.DecisionVersion, payload.Outcome, payload.DecisionReason, payload.ApprovalPolicyVersion, payload.SubjectVersion, payload.CandidateFingerprint, payload.SeparationOfDutiesReference)
	if err != nil {
		return fpmExtendCloseExceptionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The exception identifiers or approval evidence are invalid.", correlationID), nil
	}
	exceptionID, err := parseFPMUUID(payload.ExceptionID)
	if err != nil {
		return fpmExtendCloseExceptionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The close exception identifier is invalid.", correlationID), nil
	}
	command := fiscalperiod.ExtendCloseExceptionCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), FiscalPeriodID: periodID,
		CloseRunID: closeRunID, ExceptionID: exceptionID, ExpectedPeriodVersion: int64(request.ExpectedVersion.Value), ExpectedCloseRunVersion: payload.ExpectedCloseRunVersion,
		ExpectedGateVersion: payload.ExpectedGateVersion, ExpectedControlAuthorityVersion: payload.ExpectedControlAuthorityVersion, ExpectedControlAuthorityEpoch: payload.ExpectedControlAuthorityEpoch,
		Justification: payload.Justification, NewDueAt: payload.NewDueAt, NewExpiresAt: payload.NewExpiresAt, EvidenceReference: payload.EvidenceReference, Approval: approval,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.FiscalPeriodService.ExtendCloseException(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		outcome = fpmRecoveryOutcome(err)
		return mapFPMExtendCloseExceptionError(err, correlationID), nil
	}
	outcome = map[bool]string{true: "replayed", false: "established"}[result.Replayed]
	observeFPMRecoveryAging(handler.Instrumentation, ctx, "fpmExtendCloseException", result)
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/extend-close-exception"), nil
}

func beginFPMRecoveryTelemetry(instrumentation *telemetry.Instrumentation, ctx context.Context, operation string) (context.Context, func(string)) {
	if instrumentation == nil {
		return ctx, func(string) {}
	}
	ctx, span := instrumentation.StartSpan(ctx, "fiscalperiod."+operation, telemetry.SpanAttributes{Module: "fiscalperiod", Operation: operation, DataClassification: "internal"})
	finished := false
	return ctx, func(result string) {
		if finished {
			return
		}
		finished = true
		instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{Module: "fiscalperiod", Operation: operation, Result: result, DataClassification: "internal"})
		instrumentation.AddCommand(ctx, 1, "fiscalperiod", operation, result)
		span.End()
	}
}

func fpmRecoveryOutcome(err error) string {
	switch {
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodAuthorizationDenied):
		return "authorization_denied"
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodAuthorizationUnavailable), errors.Is(err, fiscalperiod.ErrControlRecoveryDecisionUnavailable), errors.Is(err, fiscalperiod.ErrPostingBarrierUnavailable), errors.Is(err, fiscalperiod.ErrFiscalPeriodAuditUnavailable), errors.Is(err, fiscalperiod.ErrFiscalPeriodOutboxUnavailable):
		return "dependency_unavailable"
	case errors.Is(err, fiscalperiod.ErrControlRecoveryDecisionRejected):
		return "approval_rejected"
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodIdempotencyConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodCommandInProgress), errors.Is(err, fiscalperiod.ErrFiscalPeriodVersionConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodOwnerConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodScopeConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodEpochConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodPolicyConflict), errors.Is(err, fiscalperiod.ErrFiscalPeriodStateConflict), errors.Is(err, fiscalperiod.ErrPostingBarrierResultConflict):
		return "conflict"
	case errors.Is(err, fiscalperiod.ErrInvalidFiscalPeriodCommand):
		return "invalid_request"
	case errors.Is(err, fiscalperiod.ErrFiscalPeriodControlAuthorityMissing), errors.Is(err, fiscalperiod.ErrFiscalPeriodRecoveryNotEligible), errors.Is(err, fiscalperiod.ErrCloseExceptionExtensionRejected), errors.Is(err, fiscalperiod.ErrPostingBarrierRejected):
		return "rejected"
	default:
		return "internal_failure"
	}
}

func observeFPMRecoveryAging(instrumentation *telemetry.Instrumentation, ctx context.Context, operation string, result fiscalperiod.CommandResult) {
	if instrumentation == nil {
		return
	}
	now := time.Now().UTC()
	var authority *fiscalperiod.ControlAuthority
	if result.State.CloseRun != nil {
		authority = result.State.CloseRun.ControlAuthority
	} else {
		authority = result.State.SoftClose.ControlAuthority
	}
	if authority != nil {
		instrumentation.AddCommand(ctx, 1, "fiscalperiod", operation+".authority_age."+fpmAgeBucket(now.Sub(authority.CutoffAt)), "observed")
	}
	if result.State.CloseRun != nil {
		for _, exception := range result.State.CloseRun.Exceptions {
			if !exception.CreatedAt.IsZero() {
				instrumentation.AddCommand(ctx, 1, "fiscalperiod", operation+".exception_age."+fpmAgeBucket(now.Sub(exception.CreatedAt)), "observed")
			}
		}
	}
}

func fpmAgeBucket(age time.Duration) string {
	switch {
	case age < 0:
		return "future"
	case age < time.Hour:
		return "under_1h"
	case age < 24*time.Hour:
		return "1h_to_24h"
	case age < 7*24*time.Hour:
		return "1d_to_7d"
	default:
		return "7d_plus"
	}
}

func decodeFPMRecoveryEvidence(periodText, processText, requestText, decisionText, referenceText, approverText string, recordedAt time.Time, decisionVersion int64, outcome, reason, policyVersion string, subjectVersion int64, candidateFingerprint, separationReference string) (uuid.UUID, uuid.UUID, fiscalperiod.ControlRecoveryApprovalDecision, error) {
	periodID, err := parseFPMUUID(periodText)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiscalperiod.ControlRecoveryApprovalDecision{}, err
	}
	processID, err := parseFPMUUID(processText)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiscalperiod.ControlRecoveryApprovalDecision{}, err
	}
	requestID, err := parseFPMUUID(requestText)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiscalperiod.ControlRecoveryApprovalDecision{}, err
	}
	decisionID, err := parseFPMUUID(decisionText)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiscalperiod.ControlRecoveryApprovalDecision{}, err
	}
	decisionReference, err := parseFPMUUID(referenceText)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiscalperiod.ControlRecoveryApprovalDecision{}, err
	}
	approverUserID, err := parseFPMUUID(approverText)
	if err != nil {
		return uuid.Nil, uuid.Nil, fiscalperiod.ControlRecoveryApprovalDecision{}, err
	}
	return periodID, processID, fiscalperiod.ControlRecoveryApprovalDecision{RequestID: requestID, DecisionID: decisionID, DecisionVersion: decisionVersion, Outcome: strings.TrimSpace(outcome), Reason: strings.TrimSpace(reason), PolicyVersion: strings.TrimSpace(policyVersion), SubjectVersion: subjectVersion, CandidateFingerprint: strings.TrimSpace(candidateFingerprint), ApproverUserID: approverUserID, DecisionReference: decisionReference, SeparationOfDutiesReference: strings.TrimSpace(separationReference), RecordedAt: recordedAt}, nil
}

func fpmRecoveryCorrelation(value generated.OptUUID) uuid.UUID {
	if value.Set && uuid.UUID(value.Value) != uuid.Nil {
		return uuid.UUID(value.Value)
	}
	return uuid.New()
}

func mapFPMTakeOverPeriodControlError(err error, correlationID uuid.UUID) generated.FpmTakeOverPeriodControlRes {
	return mapFPMError(err, correlationID, fpmTakeOverPeriodControlProblem)
}

func mapFPMExtendCloseExceptionError(err error, correlationID uuid.UUID) generated.FpmExtendCloseExceptionRes {
	return mapFPMError(err, correlationID, fpmExtendCloseExceptionProblem)
}

func fpmTakeOverPeriodControlProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmTakeOverPeriodControlRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmTakeOverPeriodControlBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmTakeOverPeriodControlForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmTakeOverPeriodControlConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmTakeOverPeriodControlUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmTakeOverPeriodControlServiceUnavailable(problem)
		return &value
	}
}

func fpmExtendCloseExceptionProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmExtendCloseExceptionRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmExtendCloseExceptionBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmExtendCloseExceptionForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmExtendCloseExceptionConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmExtendCloseExceptionUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmExtendCloseExceptionServiceUnavailable(problem)
		return &value
	}
}
