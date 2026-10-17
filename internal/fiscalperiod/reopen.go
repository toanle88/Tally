package fiscalperiod

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	ReopenModeScopedCorrection = "ScopedCorrection"
	ReopenModeOperational      = "Operational"

	ReopenRequestDraft                 = "Draft"
	ReopenRequestPendingApproval       = "PendingApproval"
	ReopenRequestApproved              = "Approved"
	ReopenRequestGateOpening           = "GateOpening"
	ReopenRequestActive                = "Active"
	ReopenRequestExpiredPendingClosure = "ExpiredPendingClosure"
	ReopenRequestGateClosing           = "GateClosing"
	ReopenRequestCompletedNoChange     = "CompletedNoChange"
	ReopenRequestRecloseHandoff        = "RecloseHandoff"
	ReopenRequestRecloseInProgress     = "RecloseInProgress"
	ReopenRequestCompleted             = "Completed"
	ReopenRequestRejected              = "Rejected"
	ReopenRequestCancelled             = "Cancelled"
	ReopenRequestFailed                = "Failed"

	ReopenApprovalApproved = "approved"
	ReopenApprovalRejected = "rejected"
)

// ReopenScope is the FPM-owned request envelope. GL receives the appropriate
// mode-specific scope through the PostingGateApplicationPort only after the
// Workflow decision has been revalidated.
type ReopenScope struct {
	AccountIDs         []uuid.UUID `json:"accountIds,omitempty"`
	TransactionClasses []string    `json:"transactionClasses,omitempty"`
	ActorIDs           []uuid.UUID `json:"actorIds,omitempty"`
}

func (scope ReopenScope) Canonical() (ReopenScope, error) {
	accounts, err := canonicalUUIDs(scope.AccountIDs, "accountIds")
	if err != nil {
		return ReopenScope{}, err
	}
	actors, err := canonicalUUIDs(scope.ActorIDs, "actorIds")
	if err != nil {
		return ReopenScope{}, err
	}
	classes := make([]string, 0, len(scope.TransactionClasses))
	seen := make(map[string]struct{}, len(scope.TransactionClasses))
	for _, value := range scope.TransactionClasses {
		value = strings.TrimSpace(value)
		if value == "" {
			return ReopenScope{}, fmt.Errorf("%w: transaction class is empty", ErrFiscalPeriodPolicyConflict)
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		classes = append(classes, value)
	}
	sort.Strings(classes)
	return ReopenScope{AccountIDs: accounts, TransactionClasses: classes, ActorIDs: actors}, nil
}

func (scope ReopenScope) Validate(mode string) error {
	canonical, err := scope.Canonical()
	if err != nil {
		return err
	}
	switch mode {
	case ReopenModeScopedCorrection:
		if len(canonical.AccountIDs) == 0 && len(canonical.TransactionClasses) == 0 {
			return fmt.Errorf("%w: scoped reopen requires an account or transaction class", ErrFiscalPeriodPolicyConflict)
		}
		if len(canonical.ActorIDs) > 0 {
			return fmt.Errorf("%w: scoped reopen cannot include actor scope", ErrFiscalPeriodPolicyConflict)
		}
	case ReopenModeOperational:
		if len(canonical.ActorIDs) == 0 || len(canonical.TransactionClasses) == 0 {
			return fmt.Errorf("%w: operational reopen requires actors and transaction classes", ErrFiscalPeriodPolicyConflict)
		}
		if len(canonical.AccountIDs) > 0 {
			return fmt.Errorf("%w: operational reopen cannot include account scope", ErrFiscalPeriodPolicyConflict)
		}
	default:
		return fmt.Errorf("%w: unsupported reopen mode %q", ErrFiscalPeriodPolicyConflict, mode)
	}
	return nil
}

func canonicalUUIDs(values []uuid.UUID, field string) ([]uuid.UUID, error) {
	result := append([]uuid.UUID(nil), values...)
	for _, value := range result {
		if value == uuid.Nil {
			return nil, fmt.Errorf("%w: %s contains a nil UUID", ErrFiscalPeriodPolicyConflict, field)
		}
	}
	sort.Slice(result, func(left, right int) bool { return result[left].String() < result[right].String() })
	if len(result) < 2 {
		return result, nil
	}
	unique := result[:1]
	for _, value := range result[1:] {
		if value != unique[len(unique)-1] {
			unique = append(unique, value)
		}
	}
	return unique, nil
}

type ReopenApprovalDecision struct {
	RequestID                   uuid.UUID `json:"requestId"`
	DecisionID                  uuid.UUID `json:"decisionId"`
	DecisionVersion             int64     `json:"decisionVersion"`
	Outcome                     string    `json:"outcome"`
	Reason                      string    `json:"reason,omitempty"`
	PolicyVersion               string    `json:"policyVersion"`
	SubjectVersion              int64     `json:"subjectVersion"`
	CandidateFingerprint        string    `json:"candidateFingerprint"`
	ApproverUserID              uuid.UUID `json:"approverUserId"`
	DecisionReference           uuid.UUID `json:"decisionReference"`
	SeparationOfDutiesReference string    `json:"separationOfDutiesReference,omitempty"`
	RecordedAt                  time.Time `json:"recordedAt"`
	DecidedAt                   time.Time `json:"decidedAt,omitempty"`
}

func (decision ReopenApprovalDecision) Validate() error {
	if decision.RequestID == uuid.Nil || decision.DecisionID == uuid.Nil || decision.DecisionReference == uuid.Nil || decision.ApproverUserID == uuid.Nil || decision.DecisionVersion < 1 || decision.SubjectVersion < 1 || strings.TrimSpace(decision.PolicyVersion) == "" || strings.TrimSpace(decision.CandidateFingerprint) == "" || strings.TrimSpace(decision.SeparationOfDutiesReference) == "" || strings.TrimSpace(decision.Outcome) == "" || decision.RecordedAt.IsZero() {
		return ErrReopenDecisionRejected
	}
	if decision.Outcome != ReopenApprovalApproved && decision.Outcome != ReopenApprovalRejected {
		return ErrReopenDecisionRejected
	}
	if decision.Outcome == ReopenApprovalRejected && strings.TrimSpace(decision.Reason) == "" {
		return ErrReopenDecisionRejected
	}
	return nil
}

type ReopenRequest struct {
	ID                    uuid.UUID               `json:"id"`
	AccountingScopeID     uuid.UUID               `json:"accountingScopeId"`
	FiscalPeriodID        uuid.UUID               `json:"fiscalPeriodId"`
	OriginalCloseRunID    uuid.UUID               `json:"originalCloseRunId"`
	Mode                  string                  `json:"mode"`
	Status                string                  `json:"status"`
	Reason                string                  `json:"reason"`
	Scope                 ReopenScope             `json:"scope"`
	ProposedCorrections   json.RawMessage         `json:"proposedCorrections"`
	Impact                json.RawMessage         `json:"impact"`
	AuthorizationExpiry   time.Time               `json:"authorizationExpiry"`
	ExpectedPeriodVersion int64                   `json:"expectedPeriodVersion"`
	ExpectedGateVersion   int64                   `json:"expectedGateVersion"`
	ControlAuthorityEpoch int64                   `json:"controlAuthorityEpoch"`
	RequestedBy           Actor                   `json:"requestedBy"`
	RequestFingerprint    string                  `json:"requestFingerprint"`
	ApprovalRequestID     uuid.UUID               `json:"approvalRequestId,omitempty"`
	ApprovalDecision      *ReopenApprovalDecision `json:"approvalDecision,omitempty"`
	GateResult            *gl.SoftCloseGateResult `json:"gateResult,omitempty"`
	GateVersion           int64                   `json:"gateVersion,omitempty"`
	AggregateVersion      int64                   `json:"aggregateVersion"`
	CreatedAt             time.Time               `json:"createdAt"`
	UpdatedAt             time.Time               `json:"updatedAt"`
	CorrelationID         string                  `json:"correlationId"`
	CausationID           string                  `json:"causationId"`
}

func (request ReopenRequest) Validate() error {
	if request.ID == uuid.Nil || request.AccountingScopeID == uuid.Nil || request.FiscalPeriodID == uuid.Nil || request.OriginalCloseRunID == uuid.Nil {
		return ErrFiscalPeriodNotFound
	}
	if request.Mode != ReopenModeScopedCorrection && request.Mode != ReopenModeOperational {
		return ErrFiscalPeriodPolicyConflict
	}
	if request.Status == "" || !validReopenRequestStatus(request.Status) {
		return ErrFiscalPeriodStateConflict
	}
	if strings.TrimSpace(request.Reason) == "" || strings.TrimSpace(request.RequestFingerprint) == "" || request.ExpectedPeriodVersion < 1 || request.ExpectedGateVersion < 1 || request.ControlAuthorityEpoch < 1 || request.AggregateVersion < 1 || request.AuthorizationExpiry.IsZero() {
		return ErrFiscalPeriodPolicyConflict
	}
	if err := request.Scope.Validate(request.Mode); err != nil {
		return err
	}
	if err := request.RequestedBy.Validate(); err != nil {
		return err
	}
	if !validJSONDocument(request.ProposedCorrections) || !validJSONDocument(request.Impact) {
		return ErrFiscalPeriodPolicyConflict
	}
	if request.ApprovalDecision != nil {
		if err := request.ApprovalDecision.Validate(); err != nil {
			return err
		}
		if request.ApprovalRequestID != request.ApprovalDecision.RequestID || request.ApprovalDecision.CandidateFingerprint != request.RequestFingerprint {
			return ErrFiscalPeriodScopeConflict
		}
	} else if request.Status == ReopenRequestApproved || request.Status == ReopenRequestGateOpening || request.Status == ReopenRequestActive || request.Status == ReopenRequestExpiredPendingClosure || request.Status == ReopenRequestRejected {
		return ErrReopenDecisionRejected
	}
	if request.Status == ReopenRequestActive || request.Status == ReopenRequestExpiredPendingClosure {
		if request.GateResult == nil || request.GateVersion < 1 {
			return ErrFiscalPeriodStateConflict
		}
		expectedMode := gl.PostingGateScopedReopen
		if request.Mode == ReopenModeOperational {
			expectedMode = gl.PostingGateOperationalReopen
		}
		gate := request.GateResult.Gate
		if err := gate.Validate(); err != nil || gate.AccountingScopeID != request.AccountingScopeID || gate.FiscalPeriodID != request.FiscalPeriodID || gate.GateMode != expectedMode || gate.GateVersion != request.GateVersion || gate.ActiveReopenRequestID == nil || *gate.ActiveReopenRequestID != request.ID || gate.ReopenAuthorizationExpiry == nil || !gate.ReopenAuthorizationExpiry.Equal(request.AuthorizationExpiry) || gate.ReopenControlAuthorityEpoch == nil || *gate.ReopenControlAuthorityEpoch != request.ControlAuthorityEpoch {
			return ErrFiscalPeriodStateConflict
		}
	}
	if request.GateResult != nil && request.GateResult.Gate.GateVersion < 1 {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func validReopenRequestStatus(status string) bool {
	switch status {
	case ReopenRequestDraft, ReopenRequestPendingApproval, ReopenRequestApproved, ReopenRequestGateOpening, ReopenRequestActive, ReopenRequestExpiredPendingClosure, ReopenRequestGateClosing, ReopenRequestCompletedNoChange, ReopenRequestRecloseHandoff, ReopenRequestRecloseInProgress, ReopenRequestCompleted, ReopenRequestRejected, ReopenRequestCancelled, ReopenRequestFailed:
		return true
	default:
		return false
	}
}

func validJSONDocument(value json.RawMessage) bool {
	_, err := canonicalJSONDocument(value)
	return err == nil
}

func canonicalJSONDocument(value json.RawMessage) (json.RawMessage, error) {
	if len(value) == 0 || string(value) == "null" {
		return nil, ErrFiscalPeriodPolicyConflict
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(value, &document); err != nil || document == nil {
		return nil, ErrFiscalPeriodPolicyConflict
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return nil, ErrFiscalPeriodPolicyConflict
	}
	return canonical, nil
}

type RequestReopenCommand struct {
	CommandID             uuid.UUID
	AccountingScopeID     uuid.UUID
	FiscalPeriodID        uuid.UUID
	Mode                  string
	Reason                string
	Scope                 ReopenScope
	ProposedCorrections   json.RawMessage
	Impact                json.RawMessage
	AuthorizationExpiry   time.Time
	ExpectedPeriodVersion int64
	ExpectedGateVersion   int64
	ControlAuthorityEpoch int64
	IdempotencyKey        string
	CorrelationID         string
	CausationID           string
}

func (command RequestReopenCommand) canonical() RequestReopenCommand {
	command.Mode = strings.TrimSpace(command.Mode)
	command.Reason = strings.TrimSpace(command.Reason)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.AuthorizationExpiry = command.AuthorizationExpiry.UTC()
	if canonical, err := command.Scope.Canonical(); err == nil {
		command.Scope = canonical
	}
	if canonical, err := canonicalJSONDocument(command.ProposedCorrections); err == nil {
		command.ProposedCorrections = canonical
	}
	if canonical, err := canonicalJSONDocument(command.Impact); err == nil {
		command.Impact = canonical
	}
	return command
}

func (command RequestReopenCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.ExpectedPeriodVersion < 1 || command.ExpectedGateVersion < 1 || command.ControlAuthorityEpoch < 1 || command.AuthorizationExpiry.IsZero() || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" || command.Reason == "" || !validJSONDocument(command.ProposedCorrections) || !validJSONDocument(command.Impact) {
		return ErrInvalidFiscalPeriodCommand
	}
	if err := command.Scope.Validate(command.Mode); err != nil {
		return err
	}
	return nil
}

type ApplyReopenApprovalDecisionCommand struct {
	CommandID                    uuid.UUID
	AccountingScopeID            uuid.UUID
	FiscalPeriodID               uuid.UUID
	ReopenRequestID              uuid.UUID
	ExpectedPeriodVersion        int64
	ExpectedReopenRequestVersion int64
	ExpectedGateVersion          int64
	Decision                     ReopenApprovalDecision
	IdempotencyKey               string
	CorrelationID                string
	CausationID                  string
}

// ReconcileReopenExpiryCommand is an internal FPM reconciliation seam. It is
// intentionally not exposed as an HTTP action: a scheduler, event consumer,
// or recovery operator supplies the current expected versions and FPM checks
// the authoritative GL owner before recording expiry. US5 owns gate closure,
// no-change completion, and reclose handoff after this transition.
type ReconcileReopenExpiryCommand struct {
	CommandID                    uuid.UUID
	AccountingScopeID            uuid.UUID
	FiscalPeriodID               uuid.UUID
	ReopenRequestID              uuid.UUID
	ExpectedPeriodVersion        int64
	ExpectedReopenRequestVersion int64
	ExpectedGateVersion          int64
	IdempotencyKey               string
	CorrelationID                string
	CausationID                  string
}

func (command ApplyReopenApprovalDecisionCommand) canonical() ApplyReopenApprovalDecisionCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Decision.Outcome = strings.TrimSpace(command.Decision.Outcome)
	command.Decision.Reason = strings.TrimSpace(command.Decision.Reason)
	command.Decision.PolicyVersion = strings.TrimSpace(command.Decision.PolicyVersion)
	command.Decision.CandidateFingerprint = strings.TrimSpace(command.Decision.CandidateFingerprint)
	command.Decision.SeparationOfDutiesReference = strings.TrimSpace(command.Decision.SeparationOfDutiesReference)
	return command
}

func (command ApplyReopenApprovalDecisionCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.ReopenRequestID == uuid.Nil || command.ExpectedPeriodVersion < 1 || command.ExpectedReopenRequestVersion < 1 || command.ExpectedGateVersion < 1 || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return command.Decision.Validate()
}

func (command ReconcileReopenExpiryCommand) canonical() ReconcileReopenExpiryCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command ReconcileReopenExpiryCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.ReopenRequestID == uuid.Nil || command.ExpectedPeriodVersion < 1 || command.ExpectedReopenRequestVersion < 1 || command.ExpectedGateVersion < 1 || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func (service *Service) RequestReopen(ctx context.Context, actor Actor, command RequestReopenCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := requestReopenFingerprint(command)
	identity := service.idempotencyIdentity(RequestReopenOperation, actor.UserID, command.AccountingScopeID, command.IdempotencyKey)
	var durable *durableCommand
	if service.durable != nil {
		acquisition, replay, err := service.acquireDurableCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, RequestReopenOperation)
		if err != nil {
			return CommandResult{}, err
		}
		if replay != nil {
			return *replay, nil
		}
		durable = &acquisition
	} else if replay, ok, conflict := service.reopenIdempotency(identity, fingerprint); ok || conflict {
		if conflict {
			return CommandResult{}, ErrFiscalPeriodIdempotencyConflict
		}
		return replay, nil
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionRequestReopen, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionRequestReopen); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if state.ReopenRequest != nil && state.ReopenRequest.RequestFingerprint == fingerprint {
		result := CommandResult{State: state, Decision: decision, Replayed: true}
		if state.ReopenRequest.GateResult != nil {
			result.GateResult = *cloneGateResult(state.ReopenRequest.GateResult)
		}
		if durable != nil {
			if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
				return CommandResult{}, err
			}
		}
		service.rememberReopenCommand(identity, fingerprint, result)
		return result, nil
	}
	if err := validateRequestReopenPreconditions(state, command); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenGateUnavailable)
		return CommandResult{}, ErrReopenGateUnavailable
	}
	if err := validateHardClosedGate(status, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	now := service.clock().UTC()
	if !command.AuthorizationExpiry.After(now) {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodPolicyConflict)
		return CommandResult{}, ErrFiscalPeriodPolicyConflict
	}
	request := ReopenRequest{ID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, OriginalCloseRunID: state.CloseRun.ID, Mode: command.Mode, Status: ReopenRequestPendingApproval, Reason: command.Reason, Scope: command.Scope, ProposedCorrections: append(json.RawMessage(nil), command.ProposedCorrections...), Impact: append(json.RawMessage(nil), command.Impact...), AuthorizationExpiry: command.AuthorizationExpiry, ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedGateVersion: command.ExpectedGateVersion, ControlAuthorityEpoch: command.ControlAuthorityEpoch, RequestedBy: actor, RequestFingerprint: fingerprint, AggregateVersion: 1, CreatedAt: now, UpdatedAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	working := cloneControlState(state)
	working.Period.AggregateVersion++
	working.Period.UpdatedAt = now
	working.ReopenRequest = &request
	payload := reopenEventPayload(request, status)
	result := CommandResult{State: working, GateResult: reopenStatusResult(status, "await-reopen-approval"), Decision: decision}
	mutation := Mutation{Before: state, After: working, Operation: RequestReopenOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, request.ID, RequestReopenOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: []Event{{EventType: "ReopenRequested", AggregateID: request.ID, AggregateVersion: request.AggregateVersion, AccountingScopeID: command.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}}}
	if err := service.commitReopenMutation(ctx, mutation, durable, result); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberReopenCommand(identity, fingerprint, result)
	return result, nil
}

func (service *Service) ApplyReopenApprovalDecision(ctx context.Context, actor Actor, command ApplyReopenApprovalDecisionCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	fingerprint := applyReopenApprovalFingerprint(command)
	identity := service.idempotencyIdentity(ApplyReopenApprovalDecisionOperation, actor.UserID, command.AccountingScopeID, command.IdempotencyKey)
	var durable *durableCommand
	if service.durable != nil {
		acquisition, replay, err := service.acquireDurableCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, ApplyReopenApprovalDecisionOperation)
		if err != nil {
			return CommandResult{}, err
		}
		if replay != nil {
			return *replay, nil
		}
		durable = &acquisition
	} else if replay, ok, conflict := service.reopenIdempotency(identity, fingerprint); ok || conflict {
		if conflict {
			return CommandResult{}, ErrFiscalPeriodIdempotencyConflict
		}
		return replay, nil
	}
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionApplyReopenApprovalDecision, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionApplyReopenApprovalDecision); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	request := state.ReopenRequest
	if request == nil || request.ID != command.ReopenRequestID || request.AccountingScopeID != command.AccountingScopeID || request.FiscalPeriodID != command.FiscalPeriodID {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion || request.AggregateVersion != command.ExpectedReopenRequestVersion || request.ExpectedGateVersion != command.ExpectedGateVersion {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodVersionConflict)
		return CommandResult{}, ErrFiscalPeriodVersionConflict
	}
	if request.Status != ReopenRequestPendingApproval && request.Status != ReopenRequestApproved && request.Status != ReopenRequestGateOpening {
		if request.ApprovalDecision != nil && sameReopenDecision(*request.ApprovalDecision, command.Decision) {
			result := CommandResult{State: state, Decision: decision, Replayed: true}
			return result, nil
		}
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodStateConflict)
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	if command.Decision.RequestID != request.ApprovalRequestID && request.ApprovalRequestID != uuid.Nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenDecisionRejected)
		return CommandResult{}, ErrReopenDecisionRejected
	}
	if command.Decision.CandidateFingerprint != request.RequestFingerprint {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenDecisionRejected)
		return CommandResult{}, ErrReopenDecisionRejected
	}
	if request.ApprovalDecision != nil && !sameReopenDecision(*request.ApprovalDecision, command.Decision) {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenDecisionRejected)
		return CommandResult{}, ErrReopenDecisionRejected
	}
	if command.Decision.ApproverUserID == request.RequestedBy.UserID {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenDecisionRejected)
		return CommandResult{}, ErrReopenDecisionRejected
	}
	if command.Decision.Outcome == ReopenApprovalRejected && request.Status == ReopenRequestPendingApproval {
		status, statusErr := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
		if statusErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenGateUnavailable)
			return CommandResult{}, ErrReopenGateUnavailable
		}
		if statusErr := validateHardClosedGate(status, command.AccountingScopeID, command.FiscalPeriodID, command.ExpectedGateVersion); statusErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, statusErr)
			return CommandResult{}, statusErr
		}
	}
	if service.reopenWorkflow == nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrReopenDecisionUnavailable)
		return CommandResult{}, ErrReopenDecisionUnavailable
	}
	if err := service.reopenWorkflow.Validate(ctx, actor, *request, command.Decision); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if command.Decision.Outcome == ReopenApprovalRejected {
		working := cloneControlState(state)
		updated := cloneReopenRequest(request)
		updated.Status = ReopenRequestRejected
		updated.ApprovalRequestID = command.Decision.RequestID
		updated.ApprovalDecision = cloneReopenDecision(&command.Decision)
		updated.AggregateVersion++
		updated.UpdatedAt = service.clock().UTC()
		working.ReopenRequest = &updated
		working.Period.AggregateVersion++
		working.Period.UpdatedAt = updated.UpdatedAt
		result := CommandResult{State: working, Decision: decision}
		mutation := Mutation{Before: state, After: working, Operation: ApplyReopenApprovalDecisionOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ApplyReopenApprovalDecisionOperation, request.RequestFingerprint, fingerprint, command.CorrelationID, command.CausationID)}
		if err := service.commitReopenMutation(ctx, mutation, durable, result); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		service.rememberReopenCommand(identity, fingerprint, result)
		return result, nil
	}

	working := state
	if request.ApprovalDecision == nil {
		approved := cloneControlState(state)
		updated := cloneReopenRequest(request)
		updated.Status = ReopenRequestApproved
		updated.ApprovalRequestID = command.Decision.RequestID
		updated.ApprovalDecision = cloneReopenDecision(&command.Decision)
		updated.AggregateVersion++
		updated.UpdatedAt = service.clock().UTC()
		approved.ReopenRequest = &updated
		approved.Period.AggregateVersion++
		approved.Period.UpdatedAt = updated.UpdatedAt
		if err := service.repository.Commit(ctx, Mutation{Before: state, After: approved, Operation: ApplyReopenApprovalDecisionOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ApplyReopenApprovalDecisionOperation, request.RequestFingerprint, fingerprint, command.CorrelationID, command.CausationID)}); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		working = approved
		request = working.ReopenRequest
	}
	if request.Status == ReopenRequestApproved {
		opening := cloneControlState(working)
		updated := cloneReopenRequest(request)
		updated.Status = ReopenRequestGateOpening
		updated.AggregateVersion++
		updated.UpdatedAt = service.clock().UTC()
		opening.ReopenRequest = &updated
		opening.Period.AggregateVersion++
		opening.Period.UpdatedAt = updated.UpdatedAt
		if err := service.repository.Commit(ctx, Mutation{Before: working, After: opening, Operation: ApplyReopenApprovalDecisionOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ApplyReopenApprovalDecisionOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID)}); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		working = opening
		request = working.ReopenRequest
	}

	gateResult, err := service.openReopenGate(ctx, actor, *request, command)
	if err != nil {
		err = classifyReopenGateError(err)
		if errors.Is(err, ErrReopenGateRejected) {
			if rollbackErr := service.restoreApprovedReopen(ctx, actor, decision, working, *request, command); rollbackErr != nil {
				service.finalizeDurableFailureIfNeeded(ctx, durable, rollbackErr)
				return CommandResult{}, rollbackErr
			}
		}
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validateReopenGateResult(*request, gateResult); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	now := service.clock().UTC()
	completed := cloneControlState(working)
	updated := cloneReopenRequest(request)
	updated.Status = ReopenRequestActive
	updated.GateResult = cloneGateResult(&gateResult)
	updated.GateVersion = gateResult.Gate.GateVersion
	updated.AggregateVersion++
	updated.UpdatedAt = now
	completed.ReopenRequest = &updated
	completed.Period.State = PeriodStateReopening
	completed.Period.StateVersion++
	completed.Period.AggregateVersion++
	completed.Period.UpdatedAt = now
	payload := reopenEventPayload(updated, gateResult.Gate)
	events := []Event{{EventType: "PeriodStateChanged", AggregateID: completed.Period.ID, AggregateVersion: completed.Period.AggregateVersion, AccountingScopeID: completed.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}}
	if updated.Mode == ReopenModeOperational {
		events = append(events, Event{EventType: "OperationalReopenActivated", AggregateID: updated.ID, AggregateVersion: updated.AggregateVersion, AccountingScopeID: updated.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload})
	} else {
		events = append(events, Event{EventType: "PeriodReopened", AggregateID: updated.ID, AggregateVersion: updated.AggregateVersion, AccountingScopeID: updated.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload})
	}
	result := CommandResult{State: completed, GateResult: gateResult, Decision: decision}
	mutation := Mutation{Before: working, After: completed, Operation: ApplyReopenApprovalDecisionOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ApplyReopenApprovalDecisionOperation, request.RequestFingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: events}
	if err := service.commitReopenMutation(ctx, mutation, durable, result); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	service.rememberReopenCommand(identity, fingerprint, result)
	return result, nil
}

// ReconcileReopenExpiry records the restrictive FPM-side expiry state after
// the authorization cutoff has passed. It does not close or mutate the GL
// gate; GL remains authoritative for rejecting new admissions and US5 owns
// the subsequent close/no-change/reclose outcome.
func (service *Service) ReconcileReopenExpiry(ctx context.Context, actor Actor, command ReconcileReopenExpiryCommand) (CommandResult, error) {
	command = command.canonical()
	if service == nil || service.repository == nil || service.authorizer == nil || service.postingGate == nil {
		return CommandResult{}, ErrInvalidFiscalPeriodService
	}
	if err := actor.Validate(); err != nil {
		return CommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return CommandResult{}, err
	}
	// Expiry reconciliation is a material reopen lifecycle transition. It
	// reuses the existing apply-reopen authorization boundary because the
	// transition is part of applying/operating the approved reopen process and
	// has no separate approved public permission or route.
	decision, err := service.authorizer.Authorize(ctx, actor, PermissionApplyReopenApprovalDecision, command.AccountingScopeID)
	if err != nil {
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionApplyReopenApprovalDecision); err != nil {
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		return CommandResult{}, err
	}
	request := state.ReopenRequest
	if request == nil || request.ID != command.ReopenRequestID || request.AccountingScopeID != command.AccountingScopeID || request.FiscalPeriodID != command.FiscalPeriodID {
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if request.Status == ReopenRequestExpiredPendingClosure {
		var gateResult gl.SoftCloseGateResult
		if status, statusErr := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID); statusErr == nil {
			gateResult = reopenStatusResult(status, "close-expired-reopen-gate")
		} else if request.GateResult != nil {
			gateResult = *cloneGateResult(request.GateResult)
		} else {
			return CommandResult{}, ErrReopenGateUnavailable
		}
		gateResult.NextAction = "close-expired-reopen-gate"
		gateResult.ValidationOutcome = "expired"
		return CommandResult{State: state, GateResult: gateResult, Decision: decision, Replayed: true}, nil
	}
	if state.Period.State != PeriodStateReopening || request.Status != ReopenRequestActive {
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion || request.AggregateVersion != command.ExpectedReopenRequestVersion || request.GateVersion != command.ExpectedGateVersion {
		return CommandResult{}, ErrFiscalPeriodVersionConflict
	}
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		return CommandResult{}, ErrReopenGateUnavailable
	}
	if err := validateActiveReopenGateStatus(*request, status, command.ExpectedGateVersion); err != nil {
		return CommandResult{}, err
	}
	now := service.clock().UTC()
	if now.Before(request.AuthorizationExpiry.UTC()) {
		return CommandResult{}, ErrFiscalPeriodStateConflict
	}
	updated := cloneReopenRequest(request)
	updated.Status = ReopenRequestExpiredPendingClosure
	updated.AggregateVersion++
	updated.UpdatedAt = now
	working := cloneControlState(state)
	working.ReopenRequest = &updated
	working.Period.AggregateVersion++
	working.Period.UpdatedAt = now
	gateResult := reopenStatusResult(status, "close-expired-reopen-gate")
	gateResult.ValidationOutcome = "expired"
	result := CommandResult{State: working, GateResult: gateResult, Decision: decision}
	commandFingerprint := reconcileReopenExpiryFingerprint(command)
	events := []Event{}
	if updated.Mode == ReopenModeOperational {
		events = append(events, Event{EventType: "OperationalReopenRequestExpired", AggregateID: updated.ID, AggregateVersion: updated.AggregateVersion, AccountingScopeID: updated.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: reopenEventPayload(updated, status)})
	}
	mutation := Mutation{Before: state, After: working, Operation: ReconcileReopenExpiryOperation, CommandFingerprint: commandFingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ReconcileReopenExpiryOperation, request.RequestFingerprint, fingerprint(updated), command.CorrelationID, command.CausationID), Events: events}
	if err := service.repository.Commit(ctx, mutation); err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func validateRequestReopenPreconditions(state ControlState, command RequestReopenCommand) error {
	if state.Period.AccountingScopeID != command.AccountingScopeID || state.Period.ID != command.FiscalPeriodID {
		return ErrFiscalPeriodScopeConflict
	}
	if state.Period.State != PeriodStateHardClosed {
		return ErrFiscalPeriodStateConflict
	}
	if state.Period.AggregateVersion != command.ExpectedPeriodVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if state.CloseRun == nil || state.CloseRun.Status != CloseRunCompleted || state.CloseRun.FinalizationGateResult == nil {
		return ErrFiscalPeriodOwnerConflict
	}
	if state.ReopenRequest != nil && isReopenOwnerStatus(state.ReopenRequest.Status) {
		if state.ReopenRequest.RequestFingerprint == requestReopenFingerprint(command) {
			return ErrFiscalPeriodIdempotencyConflict
		}
		return ErrFiscalPeriodOwnerConflict
	}
	return nil
}

func validateHardClosedGate(status gl.PostingGateStatus, scopeID, periodID uuid.UUID, expectedVersion int64) error {
	if err := status.Validate(); err != nil || status.AccountingScopeID != scopeID || status.FiscalPeriodID != periodID || status.GateMode != gl.PostingGateHardClosed || status.GateVersion != expectedVersion || status.ActiveReopenRequestID != nil || status.ActiveCloseRunID != nil {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func validateActiveReopenGateStatus(request ReopenRequest, status gl.PostingGateStatus, expectedVersion int64) error {
	if err := status.Validate(); err != nil {
		return ErrReopenGateRejected
	}
	expectedMode := gl.PostingGateScopedReopen
	if request.Mode == ReopenModeOperational {
		expectedMode = gl.PostingGateOperationalReopen
	}
	if status.AccountingScopeID != request.AccountingScopeID || status.FiscalPeriodID != request.FiscalPeriodID || status.GateMode != expectedMode || status.GateVersion != expectedVersion || status.ActiveReopenRequestID == nil || *status.ActiveReopenRequestID != request.ID || status.ReopenAuthorizationExpiry == nil || !status.ReopenAuthorizationExpiry.Equal(request.AuthorizationExpiry) || status.ReopenControlAuthorityEpoch == nil || *status.ReopenControlAuthorityEpoch != request.ControlAuthorityEpoch {
		return ErrReopenGateRejected
	}
	return nil
}

func isReopenOwnerStatus(status string) bool {
	switch status {
	case ReopenRequestDraft, ReopenRequestPendingApproval, ReopenRequestApproved, ReopenRequestGateOpening, ReopenRequestActive, ReopenRequestExpiredPendingClosure, ReopenRequestGateClosing, ReopenRequestRecloseHandoff, ReopenRequestRecloseInProgress, ReopenRequestFailed:
		return true
	default:
		return false
	}
}

func (service *Service) openReopenGate(ctx context.Context, actor Actor, request ReopenRequest, command ApplyReopenApprovalDecisionCommand) (gl.SoftCloseGateResult, error) {
	approval := gl.ApprovalDecisionReference{ApprovalRequestID: request.ApprovalDecision.RequestID, DecisionID: request.ApprovalDecision.DecisionID, PolicyVersion: request.ApprovalDecision.PolicyVersion, DecisionVersion: request.ApprovalDecision.DecisionVersion, SubjectVersion: request.ApprovalDecision.SubjectVersion, CandidateFingerprint: request.ApprovalDecision.CandidateFingerprint, ApproverUserID: request.ApprovalDecision.ApproverUserID}
	gateActor := gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}
	key := "reopen-gate:" + request.ID.String()
	if request.Mode == ReopenModeOperational {
		return service.postingGate.OpenOperationalReopenGate(ctx, gateActor, gl.OpenOperationalReopenGateCommand{CommandID: request.ID, AccountingScopeID: request.AccountingScopeID, FiscalPeriodID: request.FiscalPeriodID, ExpectedGateVersion: request.ExpectedGateVersion, ReopenRequestID: request.ID, AuthorizationScope: gl.OperationalReopenScope{ActorIDs: append([]uuid.UUID(nil), request.Scope.ActorIDs...), TransactionClasses: append([]string(nil), request.Scope.TransactionClasses...)}, AuthorizationExpiry: request.AuthorizationExpiry, ControlAuthorityEpoch: request.ControlAuthorityEpoch, Approval: approval, NoChangeClosePermitted: true, IdempotencyKey: key, CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	}
	return service.postingGate.OpenScopedReopenGate(ctx, gateActor, gl.OpenScopedReopenGateCommand{CommandID: request.ID, AccountingScopeID: request.AccountingScopeID, FiscalPeriodID: request.FiscalPeriodID, ExpectedGateVersion: request.ExpectedGateVersion, ReopenRequestID: request.ID, AuthorizationScope: gl.ScopedReopenScope{AccountIDs: append([]uuid.UUID(nil), request.Scope.AccountIDs...), TransactionClasses: append([]string(nil), request.Scope.TransactionClasses...)}, AuthorizationExpiry: request.AuthorizationExpiry, ControlAuthorityEpoch: request.ControlAuthorityEpoch, Approval: approval, NoChangeClosePermitted: true, IdempotencyKey: key, CorrelationID: command.CorrelationID, CausationID: command.CausationID})
}

func validateReopenGateResult(request ReopenRequest, result gl.SoftCloseGateResult) error {
	if err := result.Gate.Validate(); err != nil {
		return ErrReopenGateRejected
	}
	expectedMode := gl.PostingGateScopedReopen
	expectedOperation := gl.ScopedReopenGateOpenOperation
	if request.Mode == ReopenModeOperational {
		expectedMode = gl.PostingGateOperationalReopen
		expectedOperation = gl.OperationalReopenGateOpenOperation
	}
	if result.Operation != expectedOperation || result.Gate.AccountingScopeID != request.AccountingScopeID || result.Gate.FiscalPeriodID != request.FiscalPeriodID || result.Gate.GateMode != expectedMode || result.Gate.GateVersion != request.ExpectedGateVersion+1 || result.Gate.ActiveReopenRequestID == nil || *result.Gate.ActiveReopenRequestID != request.ID || result.Gate.ReopenAuthorizationExpiry == nil || !result.Gate.ReopenAuthorizationExpiry.Equal(request.AuthorizationExpiry) || result.Gate.ReopenControlAuthorityEpoch == nil || *result.Gate.ReopenControlAuthorityEpoch != request.ControlAuthorityEpoch {
		return ErrReopenGateRejected
	}
	return nil
}

func (service *Service) commitReopenMutation(ctx context.Context, mutation Mutation, durable *durableCommand, result CommandResult) error {
	if durable == nil {
		return service.repository.Commit(ctx, mutation)
	}
	committer, ok := service.repository.(DurableRepository)
	if !ok {
		return ErrInvalidFiscalPeriodService
	}
	metadata, err := service.establishedMetadata(*durable, result)
	if err != nil {
		return err
	}
	return committer.CommitWithIdempotency(ctx, mutation, DurableMutationCommit{Coordinator: service.durable.Coordinator, Acquisitions: []platformidempotency.DurableAcquisition{durable.acquisition}, Results: []platformidempotency.CommandResultMetadata{metadata}})
}

func (service *Service) restoreApprovedReopen(ctx context.Context, actor Actor, decision AuthorizationDecision, state ControlState, request ReopenRequest, command ApplyReopenApprovalDecisionCommand) error {
	if request.Status != ReopenRequestGateOpening {
		return nil
	}
	working := cloneControlState(state)
	updated := cloneReopenRequest(&request)
	updated.Status = ReopenRequestApproved
	updated.AggregateVersion++
	updated.UpdatedAt = service.clock().UTC()
	working.ReopenRequest = &updated
	working.Period.AggregateVersion++
	working.Period.UpdatedAt = updated.UpdatedAt
	return service.repository.Commit(ctx, Mutation{
		Before: state, After: working, Operation: ApplyReopenApprovalDecisionOperation,
		CommandFingerprint: applyReopenApprovalFingerprint(command),
		Audit:              service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ApplyReopenApprovalDecisionOperation, request.RequestFingerprint, request.RequestFingerprint, command.CorrelationID, command.CausationID),
	})
}

func (service *Service) reopenIdempotency(identity, fingerprint string) (CommandResult, bool, bool) {
	service.mu.Lock()
	defer service.mu.Unlock()
	existing, ok := service.idempotency[identity]
	if !ok {
		return CommandResult{}, false, false
	}
	if existing.Fingerprint != fingerprint {
		return CommandResult{}, false, true
	}
	existing.Result.Replayed = true
	return existing.Result, true, false
}

func (service *Service) rememberReopenCommand(identity, fingerprint string, result CommandResult) {
	if service == nil || service.durable != nil {
		return
	}
	service.mu.Lock()
	defer service.mu.Unlock()
	service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
}

func requestReopenFingerprint(command RequestReopenCommand) string {
	return fingerprint(struct {
		Operation             string
		AccountingScopeID     uuid.UUID
		FiscalPeriodID        uuid.UUID
		Mode                  string
		Reason                string
		Scope                 ReopenScope
		ProposedCorrections   json.RawMessage
		Impact                json.RawMessage
		AuthorizationExpiry   time.Time
		ExpectedPeriodVersion int64
		ExpectedGateVersion   int64
		ControlAuthorityEpoch int64
	}{RequestReopenCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.Mode, command.Reason, command.Scope, command.ProposedCorrections, command.Impact, command.AuthorizationExpiry.UTC(), command.ExpectedPeriodVersion, command.ExpectedGateVersion, command.ControlAuthorityEpoch})
}

func applyReopenApprovalFingerprint(command ApplyReopenApprovalDecisionCommand) string {
	return fingerprint(struct {
		Operation                    string
		AccountingScopeID            uuid.UUID
		FiscalPeriodID               uuid.UUID
		ReopenRequestID              uuid.UUID
		ExpectedPeriodVersion        int64
		ExpectedReopenRequestVersion int64
		ExpectedGateVersion          int64
		Decision                     ReopenApprovalDecision
	}{ApplyReopenApprovalDecisionIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.ReopenRequestID, command.ExpectedPeriodVersion, command.ExpectedReopenRequestVersion, command.ExpectedGateVersion, command.Decision})
}

func reconcileReopenExpiryFingerprint(command ReconcileReopenExpiryCommand) string {
	return fingerprint(struct {
		Operation                    string
		AccountingScopeID            uuid.UUID
		FiscalPeriodID               uuid.UUID
		ReopenRequestID              uuid.UUID
		ExpectedPeriodVersion        int64
		ExpectedReopenRequestVersion int64
		ExpectedGateVersion          int64
	}{ReconcileReopenExpiryCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.ReopenRequestID, command.ExpectedPeriodVersion, command.ExpectedReopenRequestVersion, command.ExpectedGateVersion})
}

func sameReopenDecision(left, right ReopenApprovalDecision) bool {
	leftData, leftErr := json.Marshal(left)
	rightData, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftData) == string(rightData)
}

func validateReopenRequestMutation(before, after *ReopenRequest) error {
	if before == nil || after == nil {
		return nil
	}
	if before.ID != after.ID {
		if !isReopenTerminalStatus(before.Status) || after.Status != ReopenRequestPendingApproval || after.AggregateVersion != 1 || before.AccountingScopeID != after.AccountingScopeID || before.FiscalPeriodID != after.FiscalPeriodID || before.OriginalCloseRunID != after.OriginalCloseRunID {
			return ErrFiscalPeriodOwnerConflict
		}
		return nil
	}
	if after.AggregateVersion != before.AggregateVersion+1 {
		return ErrFiscalPeriodVersionConflict
	}
	if before.AccountingScopeID != after.AccountingScopeID || before.FiscalPeriodID != after.FiscalPeriodID || before.OriginalCloseRunID != after.OriginalCloseRunID || before.Mode != after.Mode || before.Reason != after.Reason || !sameReopenScope(before.Scope, after.Scope) || string(before.ProposedCorrections) != string(after.ProposedCorrections) || string(before.Impact) != string(after.Impact) || !before.AuthorizationExpiry.Equal(after.AuthorizationExpiry) || before.ExpectedPeriodVersion != after.ExpectedPeriodVersion || before.ExpectedGateVersion != after.ExpectedGateVersion || before.ControlAuthorityEpoch != after.ControlAuthorityEpoch || before.RequestedBy != after.RequestedBy || before.RequestFingerprint != after.RequestFingerprint || !before.CreatedAt.Equal(after.CreatedAt) || before.CorrelationID != after.CorrelationID || before.CausationID != after.CausationID {
		return ErrFiscalPeriodPolicyConflict
	}
	if before.ApprovalRequestID != uuid.Nil && before.ApprovalRequestID != after.ApprovalRequestID {
		return ErrFiscalPeriodScopeConflict
	}
	if before.ApprovalDecision != nil && (after.ApprovalDecision == nil || !sameReopenDecision(*before.ApprovalDecision, *after.ApprovalDecision)) {
		return ErrFiscalPeriodScopeConflict
	}
	gateChanged := before.GateResult != nil && (after.GateResult == nil || !sameGateResult(*before.GateResult, *after.GateResult))
	if gateChanged {
		if before.GateResult == nil || after.GateResult == nil || (before.Status != ReopenRequestActive && before.Status != ReopenRequestExpiredPendingClosure) || after.Status != ReopenRequestGateClosing || after.GateVersion != before.GateVersion+1 || after.GateResult.Gate.GateVersion != after.GateVersion || after.GateResult.Operation != reopenGateCloseOperation(before.Mode) {
			return ErrFiscalPeriodScopeConflict
		}
	}
	if before.GateVersion > 0 && before.GateVersion != after.GateVersion && !gateChanged {
		if before.Status != ReopenRequestRecloseHandoff || after.Status != ReopenRequestRecloseInProgress || after.GateVersion != before.GateVersion+1 || before.GateResult == nil {
			return ErrFiscalPeriodVersionConflict
		}
	}
	return nil
}

func sameReopenScope(left, right ReopenScope) bool {
	leftData, leftErr := json.Marshal(left)
	rightData, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftData) == string(rightData)
}

func sameGateResult(left, right gl.SoftCloseGateResult) bool {
	leftData, leftErr := json.Marshal(left)
	rightData, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftData) == string(rightData)
}

func isReopenTerminalStatus(status string) bool {
	switch status {
	case ReopenRequestCompletedNoChange, ReopenRequestCompleted, ReopenRequestRejected, ReopenRequestCancelled:
		return true
	default:
		return false
	}
}

func cloneReopenDecision(value *ReopenApprovalDecision) *ReopenApprovalDecision {
	if value == nil {
		return nil
	}
	clone := *value
	return &clone
}

func cloneReopenRequest(value *ReopenRequest) ReopenRequest {
	clone := *value
	clone.Scope.AccountIDs = append([]uuid.UUID(nil), value.Scope.AccountIDs...)
	clone.Scope.ActorIDs = append([]uuid.UUID(nil), value.Scope.ActorIDs...)
	clone.Scope.TransactionClasses = append([]string(nil), value.Scope.TransactionClasses...)
	clone.ProposedCorrections = append(json.RawMessage(nil), value.ProposedCorrections...)
	clone.Impact = append(json.RawMessage(nil), value.Impact...)
	clone.ApprovalDecision = cloneReopenDecision(value.ApprovalDecision)
	clone.GateResult = cloneGateResult(value.GateResult)
	return clone
}

func cloneGateResult(value *gl.SoftCloseGateResult) *gl.SoftCloseGateResult {
	if value == nil {
		return nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		clone := *value
		return &clone
	}
	var clone gl.SoftCloseGateResult
	if err := json.Unmarshal(data, &clone); err != nil {
		fallback := *value
		return &fallback
	}
	return &clone
}

func reopenStatusResult(status gl.PostingGateStatus, nextAction string) gl.SoftCloseGateResult {
	return gl.SoftCloseGateResult{Operation: "get-posting-gate-status", Gate: status, ValidationOutcome: "validated", NextAction: nextAction}
}

func reopenEventPayload(request ReopenRequest, gate any) []byte {
	requestEvidence := struct {
		ID                    uuid.UUID   `json:"id"`
		AccountingScopeID     uuid.UUID   `json:"accountingScopeId"`
		FiscalPeriodID        uuid.UUID   `json:"fiscalPeriodId"`
		OriginalCloseRunID    uuid.UUID   `json:"originalCloseRunId"`
		Mode                  string      `json:"mode"`
		Status                string      `json:"status"`
		Scope                 ReopenScope `json:"scope"`
		AuthorizationExpiry   time.Time   `json:"authorizationExpiry"`
		ExpectedPeriodVersion int64       `json:"expectedPeriodVersion"`
		ExpectedGateVersion   int64       `json:"expectedGateVersion"`
		ControlAuthorityEpoch int64       `json:"controlAuthorityEpoch"`
		RequestFingerprint    string      `json:"requestFingerprint"`
		ApprovalRequestID     uuid.UUID   `json:"approvalRequestId,omitempty"`
		GateVersion           int64       `json:"gateVersion,omitempty"`
		AggregateVersion      int64       `json:"aggregateVersion"`
	}{
		ID: request.ID, AccountingScopeID: request.AccountingScopeID, FiscalPeriodID: request.FiscalPeriodID,
		OriginalCloseRunID: request.OriginalCloseRunID, Mode: request.Mode, Status: request.Status, Scope: request.Scope,
		AuthorizationExpiry: request.AuthorizationExpiry, ExpectedPeriodVersion: request.ExpectedPeriodVersion,
		ExpectedGateVersion: request.ExpectedGateVersion, ControlAuthorityEpoch: request.ControlAuthorityEpoch,
		RequestFingerprint: request.RequestFingerprint, ApprovalRequestID: request.ApprovalRequestID,
		GateVersion: request.GateVersion, AggregateVersion: request.AggregateVersion,
	}
	payload, err := json.Marshal(struct {
		Request any `json:"request"`
		Gate    any `json:"gate,omitempty"`
	}{Request: requestEvidence, Gate: gate})
	if err != nil {
		return []byte(`{"requestId":"invalid"}`)
	}
	return payload
}

func classifyReopenGateError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gl.ErrSoftCloseGateVersionConflict) || errors.Is(err, gl.ErrSoftCloseGateScopeConflict) || errors.Is(err, gl.ErrSoftCloseGateModeConflict) || errors.Is(err, gl.ErrSoftCloseGateOwnerConflict) || errors.Is(err, gl.ErrSoftCloseGateEpochConflict) || errors.Is(err, gl.ErrSoftCloseGatePolicyConflict) || errors.Is(err, gl.ErrSoftCloseGatePolicyRejected) || errors.Is(err, gl.ErrSoftCloseGateAuthorizationDenied) || errors.Is(err, gl.ErrSoftCloseGateAuthorizationExpired) || errors.Is(err, gl.ErrSoftCloseGateIdempotencyConflict) || errors.Is(err, gl.ErrSoftCloseGateRecloseSummaryConflict) {
		return fmt.Errorf("%w: %v", ErrReopenGateRejected, err)
	}
	return fmt.Errorf("%w: %v", ErrReopenGateUnavailable, err)
}

var _ ReopenDecisionPort = UnavailableReopenDecisionPort{}
