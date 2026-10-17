package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type fpmRequestReopenCommandData struct {
	FiscalPeriodID        string          `json:"fiscalPeriodId"`
	ReopenMode            string          `json:"reopenMode"`
	Reason                string          `json:"reason"`
	AccountIDs            []uuid.UUID     `json:"accountIds"`
	TransactionClasses    []string        `json:"transactionClasses"`
	ActorIDs              []uuid.UUID     `json:"actorIds"`
	ProposedCorrections   json.RawMessage `json:"proposedCorrections"`
	Impact                json.RawMessage `json:"impact"`
	AuthorizationExpiry   time.Time       `json:"authorizationExpiry"`
	ExpectedGateVersion   int64           `json:"expectedGateVersion"`
	ControlAuthorityEpoch int64           `json:"controlAuthorityEpoch"`
}

type fpmApplyReopenApprovalDecisionCommandData struct {
	FiscalPeriodID               string    `json:"fiscalPeriodId"`
	ReopenRequestID              string    `json:"reopenRequestId"`
	ApprovalRequestID            string    `json:"approvalRequestId"`
	DecisionID                   string    `json:"decisionId"`
	DecisionVersion              int64     `json:"decisionVersion"`
	Outcome                      string    `json:"outcome"`
	DecisionReason               string    `json:"decisionReason"`
	PolicyVersion                string    `json:"policyVersion"`
	SubjectVersion               int64     `json:"subjectVersion"`
	CandidateFingerprint         string    `json:"candidateFingerprint"`
	DecisionReference            string    `json:"decisionReference"`
	ApproverUserID               string    `json:"approverUserId"`
	SeparationOfDutiesReference  string    `json:"separationOfDutiesReference"`
	RecordedAt                   time.Time `json:"recordedAt"`
	DecidedAt                    time.Time `json:"decidedAt"`
	ExpectedReopenRequestVersion int64     `json:"expectedReopenRequestVersion"`
	ExpectedPeriodVersion        int64     `json:"expectedPeriodVersion"`
	ExpectedGateVersion          int64     `json:"expectedGateVersion"`
}

func (handler IdentityHandler) FpmRequestReopen(ctx context.Context, request *generated.CommandRequest, params generated.FpmRequestReopenParams) (generated.FpmRequestReopenRes, error) {
	correlationID := fpmReopenCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmRequestReopenProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmRequestReopenProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The reopen requester is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return fpmRequestReopenProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmRequestReopenCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmRequestReopenProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request data is invalid.", correlationID), nil
	}
	periodID, err := parseFPMUUID(payload.FiscalPeriodID)
	if err != nil {
		return fpmRequestReopenProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID), nil
	}
	if payload.ExpectedGateVersion < 1 || payload.ControlAuthorityEpoch < 1 || payload.AuthorizationExpiry.IsZero() || strings.TrimSpace(payload.ReopenMode) == "" {
		return fpmRequestReopenProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen mode, expiry, expected gate version, and control authority epoch are required.", correlationID), nil
	}
	result, err := handler.FiscalPeriodService.RequestReopen(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, fiscalperiod.RequestReopenCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), FiscalPeriodID: periodID,
		Mode: payload.ReopenMode, Reason: payload.Reason, Scope: fiscalperiod.ReopenScope{AccountIDs: payload.AccountIDs, TransactionClasses: payload.TransactionClasses, ActorIDs: payload.ActorIDs},
		ProposedCorrections: payload.ProposedCorrections, Impact: payload.Impact, AuthorizationExpiry: payload.AuthorizationExpiry,
		ExpectedPeriodVersion: int64(request.ExpectedVersion.Value), ExpectedGateVersion: payload.ExpectedGateVersion, ControlAuthorityEpoch: payload.ControlAuthorityEpoch,
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapFPMRequestReopenError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/request-reopen"), nil
}

func (handler IdentityHandler) FpmApplyReopenApprovalDecision(ctx context.Context, request *generated.CommandRequest, params generated.FpmApplyReopenApprovalDecisionParams) (generated.FpmApplyReopenApprovalDecisionRes, error) {
	correlationID := fpmReopenCorrelation(params.XCorrelationID)
	if handler.FiscalPeriodService == nil {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The fiscal-period service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The reopen approval actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval command requires command, scope, and expected period version fields.", correlationID), nil
	}
	var payload fpmApplyReopenApprovalDecisionCommandData
	if err := decodeFPMCommandData(request, &payload); err != nil {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen approval decision data is invalid.", correlationID), nil
	}
	periodID, err := parseFPMUUID(payload.FiscalPeriodID)
	if err != nil {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The fiscal period identifier is invalid.", correlationID), nil
	}
	reopenRequestID, err := parseFPMUUID(payload.ReopenRequestID)
	if err != nil {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The reopen request identifier is invalid.", correlationID), nil
	}
	decision, err := fpmReopenApprovalFields(payload)
	if err != nil {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The immutable Workflow decision evidence is incomplete.", correlationID), nil
	}
	expectedPeriodVersion := payload.ExpectedPeriodVersion
	if expectedPeriodVersion < 1 {
		expectedPeriodVersion = int64(request.ExpectedVersion.Value)
	}
	if payload.ExpectedReopenRequestVersion < 1 || payload.ExpectedGateVersion < 1 || expectedPeriodVersion < 1 {
		return fpmApplyReopenApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected period, reopen-request, and gate versions are required.", correlationID), nil
	}
	result, err := handler.FiscalPeriodService.ApplyReopenApprovalDecision(ctx, fiscalperiod.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, fiscalperiod.ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.UUID(request.CommandId), AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), FiscalPeriodID: periodID, ReopenRequestID: reopenRequestID,
		ExpectedPeriodVersion: expectedPeriodVersion, ExpectedReopenRequestVersion: payload.ExpectedReopenRequestVersion, ExpectedGateVersion: payload.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	})
	if err != nil {
		return mapFPMApplyReopenApprovalDecisionError(err, correlationID), nil
	}
	return establishedFPMResult(result, correlationID, "/api/v1/fiscal-periods/actions/apply-reopen-approval-decision"), nil
}

func fpmReopenApprovalFields(payload fpmApplyReopenApprovalDecisionCommandData) (fiscalperiod.ReopenApprovalDecision, error) {
	requestID, err := parseFPMUUID(payload.ApprovalRequestID)
	if err != nil {
		return fiscalperiod.ReopenApprovalDecision{}, err
	}
	decisionID, err := parseFPMUUID(payload.DecisionID)
	if err != nil {
		return fiscalperiod.ReopenApprovalDecision{}, err
	}
	decisionReference, err := parseFPMUUID(payload.DecisionReference)
	if err != nil {
		return fiscalperiod.ReopenApprovalDecision{}, err
	}
	approverID, err := parseFPMUUID(payload.ApproverUserID)
	if err != nil {
		return fiscalperiod.ReopenApprovalDecision{}, err
	}
	return fiscalperiod.ReopenApprovalDecision{
		RequestID: requestID, DecisionID: decisionID, DecisionVersion: payload.DecisionVersion, Outcome: strings.TrimSpace(payload.Outcome), Reason: strings.TrimSpace(payload.DecisionReason),
		PolicyVersion: strings.TrimSpace(payload.PolicyVersion), SubjectVersion: payload.SubjectVersion, CandidateFingerprint: strings.TrimSpace(payload.CandidateFingerprint), ApproverUserID: approverID,
		DecisionReference: decisionReference, SeparationOfDutiesReference: strings.TrimSpace(payload.SeparationOfDutiesReference), RecordedAt: payload.RecordedAt, DecidedAt: payload.DecidedAt,
	}, nil
}

func fpmReopenCorrelation(value generated.OptUUID) uuid.UUID {
	if value.Set && uuid.UUID(value.Value) != uuid.Nil {
		return uuid.UUID(value.Value)
	}
	return uuid.New()
}

func mapFPMRequestReopenError(err error, correlationID uuid.UUID) generated.FpmRequestReopenRes {
	return mapFPMError(err, correlationID, fpmRequestReopenProblem)
}

func mapFPMApplyReopenApprovalDecisionError(err error, correlationID uuid.UUID) generated.FpmApplyReopenApprovalDecisionRes {
	return mapFPMError(err, correlationID, fpmApplyReopenApprovalDecisionProblem)
}

func fpmRequestReopenProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmRequestReopenRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmRequestReopenBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmRequestReopenForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmRequestReopenConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmRequestReopenUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmRequestReopenServiceUnavailable(problem)
		return &value
	}
}

func fpmApplyReopenApprovalDecisionProblem(status int, code, detail string, correlationID uuid.UUID) generated.FpmApplyReopenApprovalDecisionRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + strings.ToLower(code), Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.FpmApplyReopenApprovalDecisionBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.FpmApplyReopenApprovalDecisionForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.FpmApplyReopenApprovalDecisionConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.FpmApplyReopenApprovalDecisionUnprocessableEntity(problem)
		return &value
	default:
		value := generated.FpmApplyReopenApprovalDecisionServiceUnavailable(problem)
		return &value
	}
}

var _ generated.FpmRequestReopenRes = (*generated.EstablishedResult)(nil)
var _ generated.FpmApplyReopenApprovalDecisionRes = (*generated.EstablishedResult)(nil)
