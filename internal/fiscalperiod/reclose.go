package fiscalperiod

import (
	"context"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

// StartRecloseCommand is the FPM-owned command that closes an approved reopen
// gate and, when GL proves that a correction was admitted, transfers ownership
// to a new non-releasable reclose run.
type StartRecloseCommand struct {
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

func (command StartRecloseCommand) canonical() StartRecloseCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	return command
}

func (command StartRecloseCommand) Validate() error {
	command = command.canonical()
	if command.CommandID == uuid.Nil || command.AccountingScopeID == uuid.Nil || command.FiscalPeriodID == uuid.Nil || command.ReopenRequestID == uuid.Nil || command.ExpectedPeriodVersion < 1 || command.ExpectedReopenRequestVersion < 1 || command.ExpectedGateVersion < 1 || command.IdempotencyKey == "" || command.CorrelationID == "" || command.CausationID == "" {
		return ErrInvalidFiscalPeriodCommand
	}
	return nil
}

func isReopenGateCloseOperation(operation string) bool {
	return operation == gl.ScopedReopenGateCloseOperation || operation == gl.OperationalReopenGateCloseOperation
}

func reopenGateCloseOperation(mode string) string {
	if mode == ReopenModeOperational {
		return gl.OperationalReopenGateCloseOperation
	}
	return gl.ScopedReopenGateCloseOperation
}

func reopenGateCloseCommandType(mode string) string {
	if mode == ReopenModeOperational {
		return gl.OperationalReopenGateCloseCommandIdentity
	}
	return gl.ScopedReopenGateCloseCommandIdentity
}

func reopenGateMode(mode string) string {
	if mode == ReopenModeOperational {
		return gl.PostingGateOperationalReopen
	}
	return gl.PostingGateScopedReopen
}

func reopenGateOwnerType(mode string) string {
	if mode == ReopenModeOperational {
		return gl.OperationalReopenGateOwnerType
	}
	return gl.ScopedReopenGateOwnerType
}

func (service *Service) StartReclose(ctx context.Context, actor Actor, command StartRecloseCommand) (CommandResult, error) {
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
	fingerprint := startRecloseFingerprint(command)
	durable, replay, unlock, err := service.beginCloseCommand(ctx, actor, command.AccountingScopeID, command.IdempotencyKey, fingerprint, StartRecloseOperation)
	if err != nil || replay != nil {
		return closeCommandReturn(replay, err)
	}
	if unlock != nil {
		defer unlock()
	}

	decision, err := service.authorizer.Authorize(ctx, actor, PermissionStartReclose, command.AccountingScopeID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodAuthorizationUnavailable)
		return CommandResult{}, ErrFiscalPeriodAuthorizationUnavailable
	}
	if err := authorizeFiscalPeriodDecision(decision, PermissionStartReclose); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	state, err := service.repository.Get(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if err := validateStartRecloseState(state, command); err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	request := state.ReopenRequest
	if request == nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}

	if request.Status == ReopenRequestCompletedNoChange || request.Status == ReopenRequestCompleted || request.Status == ReopenRequestRecloseInProgress {
		result := recloseReplayResult(state, decision)
		if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
			return CommandResult{}, err
		}
		service.rememberCommand(actor, command.AccountingScopeID, StartRecloseOperation, command.IdempotencyKey, fingerprint, result)
		return result, nil
	}

	closeResult, err := service.closeApprovedReopenGate(ctx, actor, state, command)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	if request.Status == ReopenRequestActive || request.Status == ReopenRequestExpiredPendingClosure {
		applyCommand := ApplyPostingGateResultCommand{
			CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
			ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedGateVersion: command.ExpectedGateVersion,
			ReopenRequestID: command.ReopenRequestID, ExpectedReopenRequestVersion: command.ExpectedReopenRequestVersion,
			Result: closeResult, IdempotencyKey: command.IdempotencyKey + ":close-result", CorrelationID: command.CorrelationID, CausationID: command.CausationID,
		}
		applied, applyErr := service.applyPostingGateResult(ctx, actor, decision, applyCommand, false, nil)
		if applyErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, applyErr)
			return CommandResult{}, applyErr
		}
		state = applied.State
		request = state.ReopenRequest
		closeResult = applied.GateResult
	}

	if closeResult.Gate.GateMode == gl.PostingGateHardClosed {
		result, completeErr := service.completeReopenNoChange(ctx, actor, decision, state, closeResult, command, durable, fingerprint)
		if completeErr != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, completeErr)
			return CommandResult{}, completeErr
		}
		service.rememberCommand(actor, command.AccountingScopeID, StartRecloseOperation, command.IdempotencyKey, fingerprint, result)
		return result, nil
	}

	if request.Status == ReopenRequestGateClosing {
		candidate := service.newRecloseCandidate(command, fingerprint)
		working := cloneControlState(state)
		updated := cloneReopenRequest(request)
		updated.Status = ReopenRequestRecloseHandoff
		updated.AggregateVersion++
		updated.UpdatedAt = service.clock().UTC()
		working.ReopenRequest = &updated
		working.CloseRun = &candidate
		working.Period.AggregateVersion++
		working.Period.UpdatedAt = updated.UpdatedAt
		now := updated.UpdatedAt
		payload := closeStatePayload(working, closeResult)
		mutation := Mutation{
			Before: state, After: working, Operation: StartRecloseOperation, CommandFingerprint: fingerprint,
			Audit:  service.auditRecord(actor, decision, command.AccountingScopeID, candidate.ID, StartRecloseOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID),
			Events: []Event{{EventType: "RecloseHandoffStarted", AggregateID: updated.ID, AggregateVersion: updated.AggregateVersion, AccountingScopeID: updated.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: now, Payload: payload}},
		}
		if err := service.repository.Commit(ctx, mutation); err != nil {
			service.finalizeDurableFailureIfNeeded(ctx, durable, err)
			return CommandResult{}, err
		}
		state = working
		request = state.ReopenRequest
	}

	if request == nil || request.Status != ReopenRequestRecloseHandoff || state.CloseRun == nil || state.CloseRun.CloseType != gl.CloseTypeReclose {
		service.finalizeDurableFailureIfNeeded(ctx, durable, ErrFiscalPeriodOwnerConflict)
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	beginResult, err := service.beginRecloseGate(ctx, actor, state, command)
	if err != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, err)
		return CommandResult{}, err
	}
	result, applyErr := service.applyRecloseHandoff(ctx, actor, decision, state, beginResult, command, durable, fingerprint)
	if applyErr != nil {
		service.finalizeDurableFailureIfNeeded(ctx, durable, applyErr)
		return CommandResult{}, applyErr
	}
	service.rememberCommand(actor, command.AccountingScopeID, StartRecloseOperation, command.IdempotencyKey, fingerprint, result)
	return result, nil
}

func validateStartRecloseState(state ControlState, command StartRecloseCommand) error {
	if state.Period.AccountingScopeID != command.AccountingScopeID || state.Period.ID != command.FiscalPeriodID || state.ReopenRequest == nil || state.ReopenRequest.ID != command.ReopenRequestID || state.ReopenRequest.AccountingScopeID != command.AccountingScopeID || state.ReopenRequest.FiscalPeriodID != command.FiscalPeriodID {
		return ErrFiscalPeriodScopeConflict
	}
	request := state.ReopenRequest
	switch request.Status {
	case ReopenRequestActive, ReopenRequestExpiredPendingClosure:
		if state.Period.State != PeriodStateReopening || state.Period.AggregateVersion != command.ExpectedPeriodVersion || request.AggregateVersion != command.ExpectedReopenRequestVersion || request.GateVersion != command.ExpectedGateVersion {
			return ErrFiscalPeriodVersionConflict
		}
	case ReopenRequestGateClosing, ReopenRequestRecloseHandoff:
		if request.Status == ReopenRequestGateClosing {
			if state.Period.State != PeriodStateReopening || request.GateResult == nil || request.GateVersion != request.GateResult.Gate.GateVersion {
				return ErrFiscalPeriodStateConflict
			}
		} else if state.Period.State != PeriodStateReopening || request.GateResult == nil || state.CloseRun == nil || state.CloseRun.CloseType != gl.CloseTypeReclose || state.CloseRun.Status != CloseRunInitiating {
			return ErrFiscalPeriodStateConflict
		}
		if !recloseContinuationVersionsMatch(state, command) {
			return ErrFiscalPeriodVersionConflict
		}
	case ReopenRequestRecloseInProgress, ReopenRequestCompleted:
		if state.CloseRun == nil || state.CloseRun.CloseType != gl.CloseTypeReclose || state.CloseRun.BarrierEvidence == nil || state.CloseRun.BarrierEvidence.GateVersion != request.GateVersion {
			return ErrFiscalPeriodStateConflict
		}
		if request.Status == ReopenRequestRecloseInProgress {
			if state.Period.State != PeriodStateClosing || state.CloseRun.Status != CloseRunBarrierAcquired {
				return ErrFiscalPeriodStateConflict
			}
		} else if state.Period.State != PeriodStateHardClosed || state.CloseRun.Status != CloseRunCompleted || state.CloseRun.FinalizationGateResult == nil {
			return ErrFiscalPeriodStateConflict
		}
		if !recloseContinuationVersionsMatch(state, command) {
			return ErrFiscalPeriodVersionConflict
		}
	case ReopenRequestCompletedNoChange:
		if state.Period.State != PeriodStateHardClosed || state.CloseRun == nil || state.CloseRun.CloseType != gl.CloseTypeInitialHardClose || request.GateResult == nil || request.GateVersion != request.GateResult.Gate.GateVersion {
			return ErrFiscalPeriodStateConflict
		}
		if !recloseContinuationVersionsMatch(state, command) {
			return ErrFiscalPeriodVersionConflict
		}
	default:
		return ErrFiscalPeriodStateConflict
	}
	return nil
}

type recloseVersionDelta struct {
	period  int64
	request int64
	gate    int64
}

// A StartReclose retry may arrive with the versions from the original command
// after one or more local handoff commits have already succeeded. It may also
// be a new recovery command carrying the versions currently persisted. Accept
// only those two forms; arbitrary stale versions remain conflicts.
func recloseContinuationVersionsMatch(state ControlState, command StartRecloseCommand) bool {
	delta, ok := recloseVersionDeltaForStatus(state.ReopenRequest.Status)
	if !ok {
		return false
	}
	return recloseVersionTupleMatches(
		state.Period.AggregateVersion, state.ReopenRequest.AggregateVersion, state.ReopenRequest.GateVersion,
		command.ExpectedPeriodVersion, command.ExpectedReopenRequestVersion, command.ExpectedGateVersion, delta,
	)
}

func recloseVersionDeltaForStatus(status string) (recloseVersionDelta, bool) {
	switch status {
	case ReopenRequestGateClosing:
		return recloseVersionDelta{period: 1, request: 1, gate: 1}, true
	case ReopenRequestRecloseHandoff:
		return recloseVersionDelta{period: 2, request: 2, gate: 1}, true
	case ReopenRequestRecloseInProgress:
		return recloseVersionDelta{period: 3, request: 3, gate: 2}, true
	case ReopenRequestCompletedNoChange:
		return recloseVersionDelta{period: 2, request: 2, gate: 1}, true
	case ReopenRequestCompleted:
		return recloseVersionDelta{period: 4, request: 4, gate: 2}, true
	default:
		return recloseVersionDelta{}, false
	}
}

func recloseVersionTupleMatches(currentPeriod, currentRequest, currentGate, expectedPeriod, expectedRequest, expectedGate int64, delta recloseVersionDelta) bool {
	if currentPeriod == expectedPeriod && currentRequest == expectedRequest && currentGate == expectedGate {
		return true
	}
	return currentPeriod == expectedPeriod+delta.period && currentRequest == expectedRequest+delta.request && currentGate == expectedGate+delta.gate
}

func (service *Service) newRecloseCandidate(command StartRecloseCommand, fingerprint string) CloseRun {
	now := service.clock().UTC()
	return CloseRun{
		ID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		CloseType: gl.CloseTypeReclose, Status: CloseRunInitiating, AggregateVersion: 1,
		CommandFingerprint: fingerprint, BarrierCommandID: uuid.New(), BarrierIdempotencyKey: command.IdempotencyKey + ":begin-reclose",
		Steps: defaultCloseSteps(), SealStatus: CloseRunSealNotRequested, CreatedAt: now, UpdatedAt: now,
	}
}

func (service *Service) closeApprovedReopenGate(ctx context.Context, actor Actor, state ControlState, command StartRecloseCommand) (gl.SoftCloseGateResult, error) {
	request := state.ReopenRequest
	if request == nil {
		return gl.SoftCloseGateResult{}, ErrFiscalPeriodOwnerConflict
	}
	if request.GateResult != nil && request.Status != ReopenRequestActive && request.Status != ReopenRequestExpiredPendingClosure {
		return *cloneGateResult(request.GateResult), nil
	}
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		return gl.SoftCloseGateResult{}, ErrReopenGateUnavailable
	}
	if status.GateVersion == command.ExpectedGateVersion+1 && status.LastFrozenAdmissionSummary != nil && status.LastFrozenAdmissionSummary.FrozenByCommandType == reopenGateCloseCommandType(request.Mode) {
		result := reopenCloseStatusResult(status, request.Mode, "reconciled")
		if err := validateReopenCloseGateResultForResult(startRecloseApplyCommand(command, result), state, result); err == nil {
			return result, nil
		}
	}
	if err := validateActiveReopenGateStatus(*request, status, command.ExpectedGateVersion); err != nil {
		return gl.SoftCloseGateResult{}, err
	}
	gateActor := gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}
	var result gl.SoftCloseGateResult
	if request.Mode == ReopenModeOperational {
		result, err = service.postingGate.CloseOperationalReopenGate(ctx, gateActor, gl.CloseOperationalReopenGateCommand{CommandID: command.CommandID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, ExpectedGateVersion: command.ExpectedGateVersion, ReopenRequestID: command.ReopenRequestID, IdempotencyKey: "reopen-close:" + request.ID.String(), CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	} else {
		result, err = service.postingGate.CloseScopedReopenGate(ctx, gateActor, gl.CloseScopedReopenGateCommand{CommandID: command.CommandID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, ExpectedGateVersion: command.ExpectedGateVersion, ReopenRequestID: command.ReopenRequestID, IdempotencyKey: "reopen-close:" + request.ID.String(), CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	}
	if err != nil {
		return gl.SoftCloseGateResult{}, classifyReopenGateError(err)
	}
	if err := validateReopenCloseGateResultForResult(startRecloseApplyCommand(command, result), state, result); err != nil {
		return gl.SoftCloseGateResult{}, err
	}
	return result, nil
}

func startRecloseApplyCommand(command StartRecloseCommand, result gl.SoftCloseGateResult) ApplyPostingGateResultCommand {
	return ApplyPostingGateResultCommand{
		CommandID: command.CommandID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedGateVersion: command.ExpectedGateVersion,
		ReopenRequestID: command.ReopenRequestID, ExpectedReopenRequestVersion: command.ExpectedReopenRequestVersion,
		Result: result, IdempotencyKey: command.IdempotencyKey + ":close-result", CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
}

func reopenCloseStatusResult(status gl.PostingGateStatus, mode, validation string) gl.SoftCloseGateResult {
	return gl.SoftCloseGateResult{Operation: reopenGateCloseOperation(mode), Gate: status, PolicyReference: "reconciled-from-gl-status", ValidationOutcome: validation, NextAction: "evaluate-reopen-admission-summary"}
}

func (service *Service) applyReopenCloseGateResult(ctx context.Context, actor Actor, decision AuthorizationDecision, command ApplyPostingGateResultCommand, state ControlState, checkIdempotency bool, durable *durableCommand) (CommandResult, error) {
	fingerprint := applyPostingGateResultFingerprint(command)
	identity := service.idempotencyIdentity(ApplyPostingGateResultOperation, actor.UserID, command.AccountingScopeID, command.IdempotencyKey)
	request := state.ReopenRequest
	if request == nil {
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if request.Status == ReopenRequestGateClosing || request.Status == ReopenRequestRecloseHandoff || request.Status == ReopenRequestRecloseInProgress || request.Status == ReopenRequestCompletedNoChange || request.Status == ReopenRequestCompleted {
		if err := validateReplayedReopenCloseGateResult(command, state); err != nil {
			return CommandResult{}, err
		}
		result := CommandResult{State: state, GateResult: command.Result, Decision: decision, Replayed: true}
		if durable != nil {
			if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
				return CommandResult{}, err
			}
		} else if checkIdempotency {
			service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
		}
		return result, nil
	}
	if err := validateReopenCloseGateResultForResult(command, state, command.Result); err != nil {
		return CommandResult{}, err
	}
	before := state
	after := cloneControlState(state)
	updated := cloneReopenRequest(after.ReopenRequest)
	updated.Status = ReopenRequestGateClosing
	updated.GateResult = cloneGateResult(&command.Result)
	updated.GateVersion = command.Result.Gate.GateVersion
	updated.AggregateVersion++
	updated.UpdatedAt = service.clock().UTC()
	after.ReopenRequest = &updated
	after.Period.AggregateVersion++
	after.Period.UpdatedAt = updated.UpdatedAt
	payload := reopenEventPayload(updated, command.Result.Gate)
	result := CommandResult{State: after, GateResult: command.Result, Decision: decision}
	mutation := Mutation{
		Before: before, After: after, Operation: ApplyPostingGateResultOperation, CommandFingerprint: fingerprint,
		Audit:  service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, ApplyPostingGateResultOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID),
		Events: []Event{{EventType: "GateAdmissionSummaryRecorded", AggregateID: updated.ID, AggregateVersion: updated.AggregateVersion, AccountingScopeID: updated.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: updated.UpdatedAt, Payload: payload}},
	}
	if err := service.commitCloseMutation(ctx, mutation, durable, result); err != nil {
		return CommandResult{}, err
	}
	if checkIdempotency {
		service.idempotency[identity] = storedResult{Fingerprint: fingerprint, Result: result}
	}
	return result, nil
}

func validateReopenCloseGateResult(command ApplyPostingGateResultCommand, state ControlState) error {
	if state.ReopenRequest == nil {
		return ErrFiscalPeriodOwnerConflict
	}
	if state.ReopenRequest.Status == ReopenRequestGateClosing || state.ReopenRequest.Status == ReopenRequestRecloseHandoff || state.ReopenRequest.Status == ReopenRequestRecloseInProgress || state.ReopenRequest.Status == ReopenRequestCompletedNoChange || state.ReopenRequest.Status == ReopenRequestCompleted {
		return validateReplayedReopenCloseGateResult(command, state)
	}
	return validateReopenCloseGateResultForResult(command, state, command.Result)
}

func validateReplayedReopenCloseGateResult(command ApplyPostingGateResultCommand, state ControlState) error {
	request := state.ReopenRequest
	if request == nil || request.ID != command.ReopenRequestID || request.AccountingScopeID != command.AccountingScopeID || request.FiscalPeriodID != command.FiscalPeriodID {
		return ErrFiscalPeriodOwnerConflict
	}
	if request.GateResult == nil || !sameGateResult(*request.GateResult, command.Result) {
		return ErrPostingBarrierResultConflict
	}
	delta, ok := recloseVersionDeltaForStatus(request.Status)
	if !ok || !recloseVersionTupleMatches(state.Period.AggregateVersion, request.AggregateVersion, request.GateVersion, command.ExpectedPeriodVersion, command.ExpectedReopenRequestVersion, command.ExpectedGateVersion, delta) {
		return ErrFiscalPeriodVersionConflict
	}
	return nil
}

func validateReopenCloseGateResultForResult(command ApplyPostingGateResultCommand, state ControlState, result gl.SoftCloseGateResult) error {
	request := state.ReopenRequest
	if request == nil || request.ID != command.ReopenRequestID || request.AccountingScopeID != command.AccountingScopeID || request.FiscalPeriodID != command.FiscalPeriodID {
		return ErrFiscalPeriodOwnerConflict
	}
	if request.Status != ReopenRequestActive && request.Status != ReopenRequestExpiredPendingClosure {
		return ErrFiscalPeriodStateConflict
	}
	if state.Period.State != PeriodStateReopening || state.Period.AggregateVersion != command.ExpectedPeriodVersion || request.AggregateVersion != command.ExpectedReopenRequestVersion || request.GateVersion != command.ExpectedGateVersion {
		return ErrFiscalPeriodVersionConflict
	}
	if result.Operation != reopenGateCloseOperation(request.Mode) {
		return ErrPostingBarrierResultConflict
	}
	if err := result.Gate.Validate(); err != nil || result.Gate.AccountingScopeID != command.AccountingScopeID || result.Gate.FiscalPeriodID != command.FiscalPeriodID || result.Gate.GateVersion != command.ExpectedGateVersion+1 || result.Gate.ActiveCloseRunID != nil || result.Gate.ActiveCloseType != "" || result.Gate.LastFrozenAdmissionSummary == nil {
		return ErrPostingBarrierResultConflict
	}
	summary := result.Gate.LastFrozenAdmissionSummary
	if summary.ControlOwnerType != reopenGateOwnerType(request.Mode) || summary.ControlOwnerID != request.ID || summary.ControlOwnerEpoch == nil || *summary.ControlOwnerEpoch != request.ControlAuthorityEpoch || summary.FrozenAtGateVersion != result.Gate.GateVersion || summary.FrozenByCommandType != reopenGateCloseCommandType(request.Mode) || strings.TrimSpace(summary.FrozenByCommandFingerprint) == "" {
		return ErrPostingBarrierResultConflict
	}
	noAdmission := summary.AdmittedPostingCount == 0 && !summary.ReopenPostingAdmitted && !summary.ClosePostingAdmitted && summary.FirstAdmittedLedgerPosition == 0 && summary.LastAdmittedLedgerPosition == 0
	if noAdmission {
		if result.Gate.GateMode != gl.PostingGateHardClosed || result.Gate.ActiveReopenRequestID != nil || result.Gate.ActiveAdmissionCounters != (gl.ActiveGateAdmissionCounters{}) {
			return ErrPostingBarrierResultConflict
		}
		return nil
	}
	if result.Gate.GateMode != gl.PostingGateCloseOnly || result.Gate.ActiveReopenRequestID == nil || *result.Gate.ActiveReopenRequestID != request.ID || result.Gate.ActiveAdmissionCounters.ControlOwnerType != reopenGateOwnerType(request.Mode) || result.Gate.ActiveAdmissionCounters.ControlOwnerID != request.ID {
		return ErrPostingBarrierResultConflict
	}
	return nil
}

func (service *Service) completeReopenNoChange(ctx context.Context, actor Actor, decision AuthorizationDecision, state ControlState, closeResult gl.SoftCloseGateResult, command StartRecloseCommand, durable *durableCommand, fingerprint string) (CommandResult, error) {
	if state.ReopenRequest == nil || closeResult.Gate.GateMode != gl.PostingGateHardClosed {
		return CommandResult{}, ErrPostingBarrierResultConflict
	}
	before := state
	after := cloneControlState(state)
	updated := cloneReopenRequest(after.ReopenRequest)
	updated.Status = ReopenRequestCompletedNoChange
	updated.AggregateVersion++
	updated.UpdatedAt = service.clock().UTC()
	after.ReopenRequest = &updated
	after.Period.State = PeriodStateHardClosed
	after.Period.StateVersion++
	after.Period.AggregateVersion++
	after.Period.ActiveSoftCloseRunID = nil
	after.Period.ActiveCloseRunID = nil
	after.Period.UpdatedAt = updated.UpdatedAt
	payload := reopenEventPayload(updated, closeResult.Gate)
	result := CommandResult{State: after, GateResult: closeResult, Decision: decision}
	mutation := Mutation{
		Before: before, After: after, Operation: StartRecloseOperation, CommandFingerprint: fingerprint,
		Audit: service.auditRecord(actor, decision, command.AccountingScopeID, updated.ID, StartRecloseOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID),
		Events: []Event{
			{EventType: "ReopenCompletedNoChange", AggregateID: updated.ID, AggregateVersion: updated.AggregateVersion, AccountingScopeID: updated.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: updated.UpdatedAt, Payload: payload},
			{EventType: "PeriodStateChanged", AggregateID: after.Period.ID, AggregateVersion: after.Period.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: updated.UpdatedAt, Payload: payload},
		},
	}
	if err := service.commitCloseMutation(ctx, mutation, durable, result); err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func (service *Service) beginRecloseGate(ctx context.Context, actor Actor, state ControlState, command StartRecloseCommand) (gl.SoftCloseGateResult, error) {
	request := state.ReopenRequest
	closeRun := state.CloseRun
	if request == nil || closeRun == nil || request.GateResult == nil || closeRun.CloseType != gl.CloseTypeReclose {
		return gl.SoftCloseGateResult{}, ErrFiscalPeriodOwnerConflict
	}
	if closeRun.Status == CloseRunBarrierAcquired && closeRun.BarrierEvidence != nil {
		return evidenceResult(*closeRun), nil
	}
	closeGate := *cloneGateResult(request.GateResult)
	status, err := service.postingGate.GetPostingGateStatus(ctx, command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		return gl.SoftCloseGateResult{}, ErrReopenGateUnavailable
	}
	if status.GateVersion == closeGate.Gate.GateVersion+1 && status.ActiveCloseRunID != nil && *status.ActiveCloseRunID == closeRun.ID && status.ActiveCloseType == gl.CloseTypeReclose {
		reconciled := gl.SoftCloseGateResult{Operation: gl.BeginRecloseGateOperation, Gate: status, PolicyReference: "reconciled-from-gl-status", ValidationOutcome: "reconciled", NextAction: "continue-reclose"}
		if err := validateBeginRecloseGateResult(state, reconciled); err == nil {
			return reconciled, nil
		}
	}
	if status.GateMode != gl.PostingGateCloseOnly || status.GateVersion != closeGate.Gate.GateVersion || status.ActiveReopenRequestID == nil || *status.ActiveReopenRequestID != request.ID {
		return gl.SoftCloseGateResult{}, ErrFiscalPeriodOwnerConflict
	}
	result, err := service.postingGate.BeginRecloseGate(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: actor.SubjectReference}, gl.BeginRecloseGateCommand{
		CommandID: closeRun.BarrierCommandID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		ExpectedGateVersion: closeGate.Gate.GateVersion, ReopenRequestID: request.ID, RecloseRunID: closeRun.ID,
		IdempotencyKey: closeRun.BarrierIdempotencyKey, CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	})
	if err != nil {
		return gl.SoftCloseGateResult{}, classifyReopenGateError(err)
	}
	if err := validateBeginRecloseGateResult(state, result); err != nil {
		return gl.SoftCloseGateResult{}, err
	}
	return result, nil
}

func validateBeginRecloseGateResult(state ControlState, result gl.SoftCloseGateResult) error {
	request := state.ReopenRequest
	closeRun := state.CloseRun
	if request == nil || closeRun == nil || request.GateResult == nil || closeRun.CloseType != gl.CloseTypeReclose || result.Operation != gl.BeginRecloseGateOperation {
		return ErrPostingBarrierResultConflict
	}
	if err := result.Gate.Validate(); err != nil || result.Gate.AccountingScopeID != state.Period.AccountingScopeID || result.Gate.FiscalPeriodID != state.Period.ID || result.Gate.GateMode != gl.PostingGateCloseOnly || result.Gate.GateVersion != request.GateVersion+1 || result.Gate.ActiveCloseRunID == nil || *result.Gate.ActiveCloseRunID != closeRun.ID || result.Gate.ActiveCloseType != gl.CloseTypeReclose || result.Gate.ActiveReopenRequestID != nil || result.Gate.ActiveAdmissionCounters.ControlOwnerType != gl.PostingBarrierOwnerType || result.Gate.ActiveAdmissionCounters.ControlOwnerID != closeRun.ID || result.Gate.PriorControlProcessID == nil || *result.Gate.PriorControlProcessID != request.ID || result.Gate.PriorControlProcessType != reopenGateOwnerType(request.Mode) || result.Gate.PriorControlEpoch == nil || *result.Gate.PriorControlEpoch != request.ControlAuthorityEpoch || result.Gate.PriorGateMode != reopenGateMode(request.Mode) || result.Gate.LastFrozenAdmissionSummary == nil {
		return ErrPostingBarrierResultConflict
	}
	if !sameGateSummary(result.Gate.LastFrozenAdmissionSummary, request.GateResult.Gate.LastFrozenAdmissionSummary) {
		return ErrPostingBarrierResultConflict
	}
	if result.Gate.LastFrozenAdmissionSummary.AdmittedPostingCount == 0 && !result.Gate.LastFrozenAdmissionSummary.ReopenPostingAdmitted {
		return ErrPostingBarrierResultConflict
	}
	return nil
}

func sameGateSummary(left, right *gl.FrozenGateAdmissionSummary) bool {
	if left == nil || right == nil {
		return false
	}
	return left.ControlOwnerType == right.ControlOwnerType && left.ControlOwnerID == right.ControlOwnerID && sameInt64Pointer(left.ControlOwnerEpoch, right.ControlOwnerEpoch) && left.ClosePostingAdmitted == right.ClosePostingAdmitted && left.ReopenPostingAdmitted == right.ReopenPostingAdmitted && left.AdmittedPostingCount == right.AdmittedPostingCount && left.FirstAdmittedLedgerPosition == right.FirstAdmittedLedgerPosition && left.LastAdmittedLedgerPosition == right.LastAdmittedLedgerPosition && left.FrozenAtGateVersion == right.FrozenAtGateVersion && left.FrozenAt.Equal(right.FrozenAt) && left.FrozenByCommandFingerprint == right.FrozenByCommandFingerprint && left.FrozenByCommandType == right.FrozenByCommandType
}

func sameInt64Pointer(left, right *int64) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (service *Service) applyRecloseHandoff(ctx context.Context, actor Actor, decision AuthorizationDecision, state ControlState, beginResult gl.SoftCloseGateResult, command StartRecloseCommand, durable *durableCommand, fingerprint string) (CommandResult, error) {
	if err := validateBeginRecloseGateResult(state, beginResult); err != nil {
		return CommandResult{}, err
	}
	if state.ReopenRequest == nil || state.CloseRun == nil {
		return CommandResult{}, ErrFiscalPeriodOwnerConflict
	}
	if state.ReopenRequest.Status == ReopenRequestRecloseInProgress && state.CloseRun.Status == CloseRunBarrierAcquired && state.CloseRun.BarrierEvidence != nil {
		result := CommandResult{State: state, GateResult: beginResult, Decision: decision, Replayed: true}
		if err := service.finalizeDurableResult(ctx, durable, result); err != nil {
			return CommandResult{}, err
		}
		return result, nil
	}
	before := state
	after := cloneControlState(state)
	request := cloneReopenRequest(after.ReopenRequest)
	request.Status = ReopenRequestRecloseInProgress
	request.GateVersion = beginResult.Gate.GateVersion
	request.AggregateVersion++
	request.UpdatedAt = service.clock().UTC()
	after.ReopenRequest = &request
	closeRun := after.CloseRun
	closeRun.Status = CloseRunBarrierAcquired
	closeRun.AggregateVersion++
	closeRun.UpdatedAt = request.UpdatedAt
	closeRun.LastFailureCode = ""
	closeRun.LastFailureDetail = ""
	summary := beginResult.Gate.LastFrozenAdmissionSummary
	closeRun.BarrierEvidence = &BarrierEvidence{
		Operation: beginResult.Operation, GateMode: beginResult.Gate.GateMode, GateVersion: beginResult.Gate.GateVersion,
		BarrierLedgerPosition: beginResult.Gate.BarrierLedgerPosition, PriorControlProcessID: cloneUUID(beginResult.Gate.PriorControlProcessID),
		PriorControlProcessType: beginResult.Gate.PriorControlProcessType, PriorControlEpoch: cloneInt64(beginResult.Gate.PriorControlEpoch),
		FrozenSummary: *summary, DecisionReference: beginResult.DecisionReference, PolicyReference: beginResult.PolicyReference,
		AuthorizationPolicyVersion: beginResult.AuthorizationPolicyVersion, AuditReference: beginResult.AuditReference,
		CommandFingerprint:            request.GateResult.Gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint,
		CommandType:                   request.GateResult.Gate.LastFrozenAdmissionSummary.FrozenByCommandType,
		ApplicationCommandFingerprint: postingGateResultFingerprint(beginResult), ValidationOutcome: beginResult.ValidationOutcome,
		NextAction: beginResult.NextAction, AuthoritativeResult: canonicalPostingGateResult(beginResult),
	}
	after.Period.State = PeriodStateClosing
	after.Period.StateVersion++
	after.Period.AggregateVersion++
	after.Period.ActiveSoftCloseRunID = nil
	after.Period.ActiveCloseRunID = cloneUUID(&closeRun.ID)
	after.Period.UpdatedAt = request.UpdatedAt
	after.ControlEpoch.HandoffCloseRunID = cloneUUID(&closeRun.ID)
	after.ControlEpoch.AggregateVersion++
	after.ControlEpoch.UpdatedAt = request.UpdatedAt
	payload := closeStatePayload(after, beginResult)
	events := []Event{
		{EventType: "GateAdmissionSummaryRecorded", AggregateID: closeRun.ID, AggregateVersion: closeRun.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: request.UpdatedAt, Payload: payload},
		{EventType: "PeriodStateChanged", AggregateID: after.Period.ID, AggregateVersion: after.Period.AggregateVersion, AccountingScopeID: after.Period.AccountingScopeID, CorrelationID: command.CorrelationID, CausationID: command.CausationID, OccurredAt: request.UpdatedAt, Payload: payload},
	}
	result := CommandResult{State: after, GateResult: beginResult, Decision: decision}
	mutation := Mutation{Before: before, After: after, Operation: StartRecloseOperation, CommandFingerprint: fingerprint, Audit: service.auditRecord(actor, decision, command.AccountingScopeID, closeRun.ID, StartRecloseOperation, fingerprint, fingerprint, command.CorrelationID, command.CausationID), Events: events}
	if err := service.commitCloseMutation(ctx, mutation, durable, result); err != nil {
		return CommandResult{}, err
	}
	return result, nil
}

func recloseReplayResult(state ControlState, decision AuthorizationDecision) CommandResult {
	gate := gl.SoftCloseGateResult{}
	if state.CloseRun != nil && state.ReopenRequest != nil && state.ReopenRequest.Status == ReopenRequestCompleted && state.CloseRun.FinalizationGateResult != nil {
		gate = *cloneGateResult(state.CloseRun.FinalizationGateResult)
	} else if state.ReopenRequest != nil && state.ReopenRequest.GateResult != nil {
		gate = *cloneGateResult(state.ReopenRequest.GateResult)
	}
	if state.CloseRun != nil && state.CloseRun.BarrierEvidence != nil && state.ReopenRequest != nil && state.ReopenRequest.Status == ReopenRequestRecloseInProgress {
		gate = evidenceResult(*state.CloseRun)
	}
	return CommandResult{State: state, GateResult: gate, Decision: decision, Replayed: true}
}

func startRecloseFingerprint(command StartRecloseCommand) string {
	return fingerprint(struct {
		Operation                    string
		AccountingScopeID            uuid.UUID
		FiscalPeriodID               uuid.UUID
		ReopenRequestID              uuid.UUID
		ExpectedPeriodVersion        int64
		ExpectedReopenRequestVersion int64
		ExpectedGateVersion          int64
	}{StartRecloseCommandIdentity, command.AccountingScopeID, command.FiscalPeriodID, command.ReopenRequestID, command.ExpectedPeriodVersion, command.ExpectedReopenRequestVersion, command.ExpectedGateVersion})
}
