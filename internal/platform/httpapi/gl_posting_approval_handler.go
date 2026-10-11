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

type glApplyJournalApprovalDecisionCommandData struct {
	JournalID            string `json:"journalId"`
	Outcome              string `json:"outcome"`
	ApprovalRequestID    string `json:"approvalRequestId"`
	DecisionID           string `json:"decisionId"`
	PolicyVersion        string `json:"policyVersion"`
	DecisionVersion      int64  `json:"decisionVersion"`
	SubjectVersion       int64  `json:"subjectVersion"`
	CandidateFingerprint string `json:"candidateFingerprint"`
	ApproverUserID       string `json:"approverUserId"`
	Reason               string `json:"reason"`
}

func (handler IdentityHandler) GlApplyJournalApprovalDecision(ctx context.Context, request *generated.CommandRequest, params generated.GlApplyJournalApprovalDecisionParams) (generated.GlApplyJournalApprovalDecisionRes, error) {
	correlationID := glApprovalDecisionCorrelation(params)
	if handler.PostingApprovalDecisionService == nil {
		return glApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The journal approval decision service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The applying actor is not authorized.", correlationID), nil
	}
	if request == nil || request.CommandId == (generated.UUID{}) || !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil || !request.ExpectedVersion.Set {
		return glApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command requires command, scope, and expected-version fields.", correlationID), nil
	}
	if _, err := glScopeID(generated.UUID(request.AccountingScopeId.Value), params.XAccountingScopeID); err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glApprovalDecisionProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return glApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}

	data, err := json.Marshal(request.Data)
	if err != nil {
		return glApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision data is invalid.", correlationID), nil
	}
	var payload glApplyJournalApprovalDecisionCommandData
	if err := json.Unmarshal(data, &payload); err != nil {
		return glApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision data is invalid.", correlationID), nil
	}
	command, err := glApplyJournalApprovalDecisionCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		return glApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision data is invalid.", correlationID), nil
	}
	result, err := handler.PostingApprovalDecisionService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlApprovalDecisionError(err, correlationID), nil
	}
	return establishedGlApprovalDecisionResult(result, correlationID), nil
}

func glApplyJournalApprovalDecisionCommandFromTransport(request *generated.CommandRequest, params generated.GlApplyJournalApprovalDecisionParams, payload glApplyJournalApprovalDecisionCommandData, correlationID uuid.UUID) (gl.ApplyJournalApprovalDecisionCommand, error) {
	parseRequired := func(value string) (uuid.UUID, error) {
		if strings.TrimSpace(value) == "" {
			return uuid.Nil, errors.New("required UUID is missing")
		}
		return uuid.Parse(strings.TrimSpace(value))
	}
	journalID, err := parseRequired(payload.JournalID)
	if err != nil {
		return gl.ApplyJournalApprovalDecisionCommand{}, err
	}
	approvalRequestID, err := parseRequired(payload.ApprovalRequestID)
	if err != nil {
		return gl.ApplyJournalApprovalDecisionCommand{}, err
	}
	decisionID, err := parseRequired(payload.DecisionID)
	if err != nil {
		return gl.ApplyJournalApprovalDecisionCommand{}, err
	}
	approverUserID, err := parseRequired(payload.ApproverUserID)
	if err != nil {
		return gl.ApplyJournalApprovalDecisionCommand{}, err
	}
	return gl.ApplyJournalApprovalDecisionCommand{
		CommandID: uuid.UUID(request.CommandId), JournalID: journalID, AccountingScopeID: uuid.UUID(request.AccountingScopeId.Value), ExpectedJournalVersion: int64(request.ExpectedVersion.Value),
		Decision: gl.PostingApprovalDecision{Outcome: payload.Outcome, Reason: payload.Reason, Reference: gl.ApprovalDecisionReference{
			ApprovalRequestID: approvalRequestID, DecisionID: decisionID, PolicyVersion: payload.PolicyVersion, DecisionVersion: payload.DecisionVersion,
			SubjectVersion: payload.SubjectVersion, CandidateFingerprint: payload.CandidateFingerprint, ApproverUserID: approverUserID,
		}},
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}, nil
}

func glApprovalDecisionCorrelation(params generated.GlApplyJournalApprovalDecisionParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedGlApprovalDecisionResult(result gl.PostingApprovalDecisionResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"outcome":              mustRaw(result.Outcome),
		"applicationStatus":    mustRaw(result.ApplicationStatus),
		"lifecycleStatus":      mustRaw(result.LifecycleStatus()),
		"approval":             mustRaw(result.Approval),
		"approvalRequestId":    mustRaw(result.ApprovalRequestID.String()),
		"approvalDecisionId":   mustRaw(result.Approval.DecisionID.String()),
		"subjectVersion":       mustRaw(result.Approval.SubjectVersion),
		"candidateFingerprint": mustRaw(result.Approval.CandidateFingerprint),
		"policyReference":      mustRaw(result.PolicyReference),
		"policyVersion":        mustRaw(result.Approval.PolicyVersion),
		"validationOutcome":    mustRaw(result.ValidationOutcome),
		"approvalStatus":       mustRaw(result.ApprovalStatus),
		"sourceReference":      mustRaw(result.SourceReference),
		"gateEvidence":         mustRaw(result.GateEvidence),
		"auditReference":       mustRaw(result.AuditReference.String()),
		"nextAction":           mustRaw(result.NextAction),
		"replayed":             mustRaw(result.Replayed),
	}
	if len(result.Issues) > 0 {
		data["issues"] = mustRaw(result.Issues)
	}
	if result.Journal != nil && result.Journal.Approval != nil && result.Journal.Approval.DecisionReason != "" {
		data["decisionReason"] = mustRaw(result.Journal.Approval.DecisionReason)
	}
	if result.Journal != nil {
		data["journalId"] = mustRaw(result.Journal.ID.String())
		data["journalNumber"] = mustRaw(result.Journal.Number)
		data["journalVersion"] = mustRaw(result.Journal.Version)
		data["ledgerPosition"] = mustRaw(result.Journal.LedgerPosition)
	}
	return &generated.EstablishedResult{
		Status: "established", AggregateId: generated.UUID(result.AggregateID()), AggregateVersion: int(result.AggregateVersion()), CorrelationId: generated.UUID(correlationID),
		Links: generated.Links{Self: "/api/v1/general-ledger/actions/apply-journal-approval-decision"}, Data: data,
	}
}

func mapGlApprovalDecisionError(err error, correlationID uuid.UUID) generated.GlApplyJournalApprovalDecisionRes {
	switch {
	case errors.Is(err, gl.ErrPostingApprovalDecisionAuthorizationDenied), errors.Is(err, gl.ErrPostingAuthorizationDenied):
		return glApprovalDecisionProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The applying actor is not authorized for this journal approval decision.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionUnavailable), errors.Is(err, gl.ErrPostingApprovalDecisionAuthorizationMissing), errors.Is(err, gl.ErrPostingApprovalDecisionAuditUnavailable), errors.Is(err, gl.ErrPostingApprovalDecisionOutboxUnavailable), errors.Is(err, gl.ErrInvalidPostingApprovalDecisionService), errors.Is(err, gl.ErrPostingAuthorizationUnavailable), errors.Is(err, gl.ErrPostingReferenceUnavailable), errors.Is(err, gl.ErrPostingGatePolicyUnavailable):
		return glApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The journal approval decision could not be completed.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionIdempotencyConflict):
		return glApprovalDecisionProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different approval-decision data.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionCommandInProgress):
		return glApprovalDecisionProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The approval decision is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionNotFound):
		return glApprovalDecisionProblem(http.StatusConflict, "REQUEST_NOT_FOUND", "The pending journal does not exist.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionAlreadyApplied):
		return glApprovalDecisionProblem(http.StatusConflict, "DUPLICATE_DECISION", "The journal already has a terminal approval decision.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionVersionConflict), errors.Is(err, gl.ErrPostingApprovalDecisionScopeConflict), errors.Is(err, gl.ErrPostingPeriodConflict), errors.Is(err, gl.ErrPostingGateConflict):
		return glApprovalDecisionProblem(http.StatusConflict, "VERSION_CONFLICT", "The pending journal or posting gate changed. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrPostingApprovalDecisionReferenceInvalid):
		return glApprovalDecisionProblem(http.StatusUnprocessableEntity, "INVALID_APPROVAL_REFERENCE", "The Workflow approval decision does not match the stored journal proposal.", correlationID)
	case errors.Is(err, gl.ErrPostingGateClosed), errors.Is(err, gl.ErrPostingGatePolicyRejected), errors.Is(err, gl.ErrPostingReferenceInvalid), errors.Is(err, gl.ErrPostingValidation), errors.Is(err, gl.ErrPostingApprovalDecisionValidation):
		return glApprovalDecisionProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The approval decision violates a general-ledger rule.", correlationID)
	case errors.Is(err, gl.ErrInvalidPostingApprovalDecisionCommand):
		return glApprovalDecisionProblem(http.StatusBadRequest, "INVALID_REQUEST", "The approval decision command is invalid.", correlationID)
	default:
		return glApprovalDecisionProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The journal approval decision could not be completed.", correlationID)
	}
}

func glApprovalDecisionProblem(status int, code, detail string, correlationID uuid.UUID) generated.GlApplyJournalApprovalDecisionRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlApplyJournalApprovalDecisionBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlApplyJournalApprovalDecisionForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlApplyJournalApprovalDecisionConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlApplyJournalApprovalDecisionUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlApplyJournalApprovalDecisionServiceUnavailable(problem)
		return &value
	}
}
