package fiscalperiod

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

type recloseGateStub struct {
	*hardCloseGateStub
	closeResult gl.SoftCloseGateResult
	beginResult gl.SoftCloseGateResult
	err         error
	closeCalls  int
	beginCalls  int
}

func (stub *recloseGateStub) CloseScopedReopenGate(context.Context, gl.Actor, gl.CloseScopedReopenGateCommand) (gl.SoftCloseGateResult, error) {
	stub.closeCalls++
	if stub.err != nil {
		return gl.SoftCloseGateResult{}, stub.err
	}
	stub.hardCloseGateStub.result = stub.closeResult
	return stub.closeResult, nil
}

func (stub *recloseGateStub) CloseOperationalReopenGate(context.Context, gl.Actor, gl.CloseOperationalReopenGateCommand) (gl.SoftCloseGateResult, error) {
	stub.closeCalls++
	if stub.err != nil {
		return gl.SoftCloseGateResult{}, stub.err
	}
	stub.hardCloseGateStub.result = stub.closeResult
	return stub.closeResult, nil
}

func (stub *recloseGateStub) BeginRecloseGate(_ context.Context, _ gl.Actor, command gl.BeginRecloseGateCommand) (gl.SoftCloseGateResult, error) {
	stub.beginCalls++
	if stub.err != nil {
		return gl.SoftCloseGateResult{}, stub.err
	}
	result := stub.beginResult
	result.Gate.ActiveCloseRunID = cloneUUID(&command.RecloseRunID)
	result.Gate.ActiveAdmissionCounters.ControlOwnerID = command.RecloseRunID
	stub.hardCloseGateStub.result = result
	return result, nil
}

type recloseFailOnceRepository struct {
	delegate *MemoryRepository
	failOn   int
	commits  int
	failure  error
}

func (repository *recloseFailOnceRepository) Get(ctx context.Context, scopeID, periodID uuid.UUID) (ControlState, error) {
	return repository.delegate.Get(ctx, scopeID, periodID)
}

func (repository *recloseFailOnceRepository) Commit(ctx context.Context, mutation Mutation) error {
	repository.commits++
	if repository.failOn > 0 && repository.commits == repository.failOn {
		repository.failOn = 0
		return repository.failure
	}
	return repository.delegate.Commit(ctx, mutation)
}

func TestStartRecloseCompletesAuthoritativeNoChangeWithoutNewCloseRun(t *testing.T) {
	state, command, now, requestID, originalCloseRunID := recloseFixture(t)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, false)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.ReopenRequest.Status != ReopenRequestCompletedNoChange || result.State.Period.State != PeriodStateHardClosed {
		t.Fatalf("no-change result = %#v", result.State)
	}
	if result.State.CloseRun == nil || result.State.CloseRun.ID != originalCloseRunID || result.State.CloseRun.CloseType != gl.CloseTypeInitialHardClose {
		t.Fatalf("no-change close run = %#v", result.State.CloseRun)
	}
	if result.State.CloseRun.FinalLedgerWatermark == nil || *result.State.CloseRun.FinalLedgerWatermark != 99 || result.State.CloseRun.SealStatus != CloseRunSealSealed {
		t.Fatalf("prior close evidence was not retained = %#v", result.State.CloseRun)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 0 {
		t.Fatalf("gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
	if events := repository.Events(); len(events) != 3 || events[1].EventType != "ReopenCompletedNoChange" {
		t.Fatalf("events = %#v", events)
	}
}

func TestStartRecloseDependencyFailureDoesNotBecomeNoChange(t *testing.T) {
	state, command, now, requestID, originalCloseRunID := recloseFixture(t)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, err: ErrReopenGateUnavailable}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.StartReclose(context.Background(), testActor(), command); err == nil || !errors.Is(err, ErrReopenGateUnavailable) {
		t.Fatalf("dependency failure = %v", err)
	}
	stored, err := repository.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Period.State != PeriodStateReopening || stored.ReopenRequest == nil || stored.ReopenRequest.ID != requestID || stored.ReopenRequest.Status != ReopenRequestActive || stored.CloseRun == nil || stored.CloseRun.ID != originalCloseRunID {
		t.Fatalf("dependency failure mutated control state = %#v", stored)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 0 || len(repository.Events()) != 0 {
		t.Fatalf("dependency failure side effects close=%d begin=%d events=%d", gate.closeCalls, gate.beginCalls, len(repository.Events()))
	}
}

func TestStartRecloseReconcilesAlreadyClosedGateFromAuthoritativeStatus(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: closeResult}, closeResult: closeResult, beginResult: beginResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.ReopenRequest.Status != ReopenRequestRecloseInProgress || result.GateResult.Gate.ActiveCloseType != gl.CloseTypeReclose {
		t.Fatalf("reconciled reclose state = %#v / %#v", result.State, result.GateResult)
	}
	if gate.closeCalls != 0 || gate.beginCalls != 1 {
		t.Fatalf("reconciled gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
}

func TestStartRecloseClosesExpiredReopenWithoutTreatingItAsActive(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	state.ReopenRequest.Status = ReopenRequestExpiredPendingClosure
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, false)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.ReopenRequest.Status != ReopenRequestCompletedNoChange || result.State.Period.State != PeriodStateHardClosed {
		t.Fatalf("expired reclose state = %#v", result.State)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 0 {
		t.Fatalf("expired gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
}

func TestStartRecloseCreatesScopedMandatoryRecloseRun(t *testing.T) {
	state, command, now, requestID, originalCloseRunID := recloseFixture(t)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult, beginResult: beginResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	actor := testActor()
	result, err := service.StartReclose(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateClosing || result.State.ReopenRequest.Status != ReopenRequestRecloseInProgress {
		t.Fatalf("reclose state = %#v", result.State)
	}
	if result.State.CloseRun == nil || result.State.CloseRun.ID == originalCloseRunID || result.State.CloseRun.CloseType != gl.CloseTypeReclose || result.State.CloseRun.Status != CloseRunBarrierAcquired {
		t.Fatalf("reclose close run = %#v", result.State.CloseRun)
	}
	if result.State.CloseRun.BarrierEvidence == nil || result.State.CloseRun.BarrierEvidence.PriorControlProcessID == nil || *result.State.CloseRun.BarrierEvidence.PriorControlProcessID != requestID || result.State.CloseRun.BarrierEvidence.PriorControlProcessType != gl.ScopedReopenGateOwnerType {
		t.Fatalf("reclose lineage = %#v", result.State.CloseRun.BarrierEvidence)
	}
	if result.State.ReopenRequest.GateVersion != command.ExpectedGateVersion+2 || result.GateResult.Gate.ActiveCloseType != gl.CloseTypeReclose {
		t.Fatalf("reclose gate version/result = %#v / %#v", result.State.ReopenRequest, result.GateResult)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 1 {
		t.Fatalf("gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
	replayed, err := service.StartReclose(context.Background(), actor, command)
	if err != nil || !replayed.Replayed || gate.closeCalls != 1 || gate.beginCalls != 1 {
		t.Fatalf("duplicate start-reclose = %#v, err=%v, gate calls close=%d begin=%d", replayed, err, gate.closeCalls, gate.beginCalls)
	}
	if events := repository.Events(); len(events) != 4 || events[1].EventType != "RecloseHandoffStarted" || events[2].EventType != "GateAdmissionSummaryRecorded" {
		t.Fatalf("events = %#v", events)
	}
}

func TestStartRecloseCreatesOperationalMandatoryRecloseRun(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	state.ReopenRequest.Mode = ReopenModeOperational
	state.ReopenRequest.Scope = ReopenScope{ActorIDs: []uuid.UUID{uuid.New()}, TransactionClasses: []string{"OperationalCorrection"}}
	state.ReopenRequest.GateResult.Operation = gl.OperationalReopenGateOpenOperation
	state.ReopenRequest.GateResult.Gate.GateMode = gl.PostingGateOperationalReopen
	state.ReopenRequest.GateResult.Gate.ActiveAdmissionCounters.ControlOwnerType = gl.OperationalReopenGateOwnerType
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult, beginResult: beginResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.ReopenRequest.Status != ReopenRequestRecloseInProgress || result.State.CloseRun == nil || result.State.CloseRun.BarrierEvidence == nil {
		t.Fatalf("operational reclose state = %#v", result.State)
	}
	if result.State.CloseRun.BarrierEvidence.PriorControlProcessType != gl.OperationalReopenGateOwnerType || result.GateResult.Gate.PriorGateMode != gl.PostingGateOperationalReopen {
		t.Fatalf("operational reclose lineage = %#v / %#v", result.State.CloseRun.BarrierEvidence, result.GateResult.Gate)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 1 {
		t.Fatalf("operational gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
}

func TestStartRecloseResumesOriginalCommandAfterGateResultCommit(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	delegate := NewMemoryRepository()
	if err := delegate.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult, beginResult: beginResult}
	repository := &recloseFailOnceRepository{delegate: delegate, failOn: 2, failure: errors.New("simulated interruption after gate result")}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.StartReclose(context.Background(), testActor(), command); err == nil {
		t.Fatal("expected the interrupted handoff to return an error")
	}
	interrupted, err := delegate.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.ReopenRequest == nil || interrupted.ReopenRequest.Status != ReopenRequestGateClosing || interrupted.Period.AggregateVersion != command.ExpectedPeriodVersion+1 {
		t.Fatalf("interrupted state = %#v", interrupted)
	}

	resumed, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State.ReopenRequest.Status != ReopenRequestRecloseInProgress || resumed.State.CloseRun == nil || resumed.State.CloseRun.Status != CloseRunBarrierAcquired {
		t.Fatalf("resumed state = %#v", resumed.State)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 1 {
		t.Fatalf("resumed gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
}

func TestStartRecloseReconcilesOriginalCommandAfterBeginRecloseCommitInterruption(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	delegate := NewMemoryRepository()
	if err := delegate.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult, beginResult: beginResult}
	repository := &recloseFailOnceRepository{delegate: delegate, failOn: 3, failure: errors.New("simulated interruption after GL handoff")}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.StartReclose(context.Background(), testActor(), command); err == nil {
		t.Fatal("expected the interrupted reclose handoff to return an error")
	}
	interrupted, err := delegate.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.ReopenRequest == nil || interrupted.ReopenRequest.Status != ReopenRequestRecloseHandoff || interrupted.CloseRun == nil || interrupted.CloseRun.CloseType != gl.CloseTypeReclose {
		t.Fatalf("interrupted handoff state = %#v", interrupted)
	}

	resumed, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.State.ReopenRequest.Status != ReopenRequestRecloseInProgress || resumed.State.CloseRun == nil || resumed.State.CloseRun.Status != CloseRunBarrierAcquired {
		t.Fatalf("resumed state = %#v", resumed.State)
	}
	if gate.closeCalls != 1 || gate.beginCalls != 1 {
		t.Fatalf("reconciled gate calls close=%d begin=%d", gate.closeCalls, gate.beginCalls)
	}
}

func TestApplyPostingGateResultRejectsWrongIdentityAndStaleReplayedRecloseEvidence(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult, beginResult: beginResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartReclose(context.Background(), testActor(), command); err != nil {
		t.Fatal(err)
	}

	wrongIdentity := ApplyPostingGateResultCommand{CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, ReopenRequestID: uuid.New(), ExpectedPeriodVersion: command.ExpectedPeriodVersion, ExpectedReopenRequestVersion: command.ExpectedReopenRequestVersion, ExpectedGateVersion: command.ExpectedGateVersion, Result: closeResult, IdempotencyKey: "wrong-reopen-identity", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	if _, err := service.ApplyPostingGateResult(context.Background(), testActor(), wrongIdentity); !errors.Is(err, ErrFiscalPeriodOwnerConflict) {
		t.Fatalf("wrong replay identity error = %v", err)
	}

	stale := wrongIdentity
	stale.ReopenRequestID = requestID
	stale.IdempotencyKey = "stale-replay-version"
	stale.ExpectedPeriodVersion = command.ExpectedPeriodVersion + 1
	if _, err := service.ApplyPostingGateResult(context.Background(), testActor(), stale); !errors.Is(err, ErrFiscalPeriodVersionConflict) {
		t.Fatalf("stale replay version error = %v", err)
	}
}

func TestRecloseFinalizationCompletesRequestAndEmitsPeriodReclosed(t *testing.T) {
	state, command, now, requestID, _ := recloseFixture(t)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeResult := recloseCloseResult(state, requestID, command.ExpectedGateVersion, now, true)
	beginResult := recloseBeginResult(state, closeResult, now)
	gate := &recloseGateStub{hardCloseGateStub: &hardCloseGateStub{result: *cloneGateResult(state.ReopenRequest.GateResult)}, closeResult: closeResult, beginResult: beginResult}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now.Add(2 * time.Hour) })
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartReclose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	watermark := int64(120)
	finalizedGate := started.GateResult.Gate
	finalizedGate.GateVersion++
	finalizedGate.AggregateVersion = finalizedGate.GateVersion
	finalizedGate.GateMode = gl.PostingGateHardClosed
	finalizedGate.ActiveCloseRunID = nil
	finalizedGate.ActiveCloseType = ""
	finalizedGate.ActiveAdmissionCounters = gl.ActiveGateAdmissionCounters{}
	finalizedGate.FinalizedCloseRunID = cloneUUID(&started.State.CloseRun.ID)
	finalizedGate.FinalLedgerWatermark = &watermark
	finalizedGate.FinalizationCommandFingerprint = "sha256:reclose-finalize"
	finalizedGate.LastFrozenAdmissionSummary = &gl.FrozenGateAdmissionSummary{ControlOwnerType: gl.PostingBarrierOwnerType, ControlOwnerID: started.State.CloseRun.ID, FrozenAtGateVersion: finalizedGate.GateVersion, FrozenByCommandFingerprint: "sha256:reclose-finalize", FrozenByCommandType: gl.PostingBarrierFinalizeCommandIdentity}
	seal := &us3SealStub{result: AuditSealResult{Status: CloseRunSealSealed, RequestID: uuid.New(), ProofReference: "seal://reclose"}}
	service.seal = seal
	finalized, err := service.applyAuthoritativeFinalization(context.Background(), testActor(), AuthorizationDecision{Allowed: true, Permission: PermissionResumeCloseRun, PolicyReference: "close-policy", PolicyVersion: "close-v1", DecisionReference: uuid.New()}, started.State, gl.SoftCloseGateResult{Operation: gl.PostingBarrierFinalizeOperation, Gate: finalizedGate, ValidationOutcome: "authoritative"}, uuid.NewString(), uuid.NewString(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.State.Period.State != PeriodStateHardClosed || finalized.State.ReopenRequest.Status != ReopenRequestCompleted || finalized.State.CloseRun.Status != CloseRunCompleted || finalized.State.CloseRun.FinalizationGateResult == nil || finalized.State.CloseRun.SealProofReference != "seal://reclose" {
		t.Fatalf("finalized reclose = %#v", finalized.State)
	}
	if events := repository.Events(); len(events) != 7 || events[len(events)-1].EventType != "PeriodReclosed" {
		t.Fatalf("finalization events = %#v", events)
	}
}

func recloseFixture(t *testing.T) (ControlState, StartRecloseCommand, time.Time, uuid.UUID, uuid.UUID) {
	t.Helper()
	scopeID, periodID, softCloseRunID, originalCloseRunID, requestID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, time.October, 16, 8, 0, 0, 0, time.UTC)
	actor := testActor()
	expiry := now.Add(time.Hour)
	epoch := int64(2)
	approvalRequestID := uuid.New()
	requestFingerprint := "sha256:reopen-request"
	openGate := gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateScopedReopen, PeriodStateVersion: 4, GateVersion: 8, AggregateVersion: 8, NextLedgerPosition: 100, ActiveReopenRequestID: cloneUUID(&requestID), ReopenAuthorizationExpiry: &expiry, ReopenControlAuthorityEpoch: &epoch, ActiveAdmissionCounters: gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.ScopedReopenGateOwnerType, ControlOwnerID: requestID, ControlOwnerEpoch: cloneInt64(&epoch)}}
	approval := &ReopenApprovalDecision{RequestID: approvalRequestID, DecisionID: uuid.New(), DecisionVersion: 1, Outcome: ReopenApprovalApproved, PolicyVersion: "reopen-policy-v1", SubjectVersion: 1, CandidateFingerprint: requestFingerprint, ApproverUserID: uuid.New(), DecisionReference: uuid.New(), SeparationOfDutiesReference: "sod-1", RecordedAt: now.Add(-time.Minute)}
	request := &ReopenRequest{ID: requestID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, OriginalCloseRunID: originalCloseRunID, Mode: ReopenModeScopedCorrection, Status: ReopenRequestActive, Reason: "Correct approved reopen scope", Scope: ReopenScope{AccountIDs: []uuid.UUID{uuid.New()}, TransactionClasses: []string{"ReopenCorrection"}}, ProposedCorrections: []byte(`{"replacement":"classification"}`), Impact: []byte(`{"materiality":"low"}`), AuthorizationExpiry: expiry, ExpectedPeriodVersion: 3, ExpectedGateVersion: 7, ControlAuthorityEpoch: epoch, RequestedBy: actor, RequestFingerprint: requestFingerprint, ApprovalRequestID: approvalRequestID, ApprovalDecision: approval, GateResult: &gl.SoftCloseGateResult{Operation: gl.ScopedReopenGateOpenOperation, Gate: openGate}, GateVersion: openGate.GateVersion, AggregateVersion: 5, CreatedAt: now, UpdatedAt: now, CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	watermark := int64(99)
	state := ControlState{Period: FiscalPeriod{ID: periodID, AccountingScopeID: scopeID, State: PeriodStateReopening, StateVersion: 4, AggregateVersion: 4, CreatedAt: now, UpdatedAt: now}, SoftClose: SoftCloseRun{ID: softCloseRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Status: SoftCloseRunSuperseded, ControlEpoch: 2, OpenedGateVersion: 6, PolicyVersion: "soft-close-policy-v1", AggregateVersion: 2, CreatedAt: now, UpdatedAt: now}, ControlEpoch: SoftCloseControlEpoch{SoftCloseRunID: softCloseRunID, FiscalPeriodID: periodID, Number: 2, OpenedGateVersion: 6, PolicyVersion: "soft-close-policy-v1", Status: ControlEpochHandoffPending, AggregateVersion: 3, CreatedAt: now, UpdatedAt: now}, CloseRun: &CloseRun{ID: originalCloseRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunCompleted, AggregateVersion: 4, CommandFingerprint: "sha256:initial-close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "initial-close-barrier", FinalLedgerWatermark: &watermark, FinalizationGateResult: &gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateHardClosed, GateVersion: 7, AggregateVersion: 7, NextLedgerPosition: 100}}, SealStatus: CloseRunSealSealed, SealRequestID: uuid.New(), SealProofReference: "seal://initial", CreatedAt: now, UpdatedAt: now}, ReopenRequest: request}
	command := StartRecloseCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ReopenRequestID: requestID, ExpectedPeriodVersion: state.Period.AggregateVersion, ExpectedReopenRequestVersion: request.AggregateVersion, ExpectedGateVersion: request.GateVersion, IdempotencyKey: "start-reclose", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	return state, command, now, requestID, originalCloseRunID
}

func recloseCloseResult(state ControlState, requestID uuid.UUID, expectedGateVersion int64, now time.Time, positive bool) gl.SoftCloseGateResult {
	epoch := state.ReopenRequest.ControlAuthorityEpoch
	count := int64(0)
	first, last := int64(0), int64(0)
	mode := gl.PostingGateHardClosed
	if positive {
		count, first, last, mode = 1, 96, 96, gl.PostingGateCloseOnly
	}
	ownerType := gl.ScopedReopenGateOwnerType
	closeOperation := gl.ScopedReopenGateCloseOperation
	closeCommandType := gl.ScopedReopenGateCloseCommandIdentity
	if state.ReopenRequest.Mode == ReopenModeOperational {
		ownerType = gl.OperationalReopenGateOwnerType
		closeOperation = gl.OperationalReopenGateCloseOperation
		closeCommandType = gl.OperationalReopenGateCloseCommandIdentity
	}
	summary := gl.FrozenGateAdmissionSummary{ControlOwnerType: ownerType, ControlOwnerID: requestID, ControlOwnerEpoch: cloneInt64(&epoch), AdmittedPostingCount: count, ReopenPostingAdmitted: positive, FirstAdmittedLedgerPosition: first, LastAdmittedLedgerPosition: last, FrozenAtGateVersion: expectedGateVersion + 1, FrozenAt: now, FrozenByCommandFingerprint: "sha256:reopen-close", FrozenByCommandType: closeCommandType}
	gate := gl.PeriodPostingGate{ID: state.ReopenRequest.GateResult.Gate.ID, AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, GateMode: mode, PeriodStateVersion: state.Period.StateVersion, GateVersion: expectedGateVersion + 1, AggregateVersion: expectedGateVersion + 1, NextLedgerPosition: 100, LastFrozenAdmissionSummary: &summary}
	if positive {
		gate.ActiveReopenRequestID = cloneUUID(&requestID)
		gate.ActiveAdmissionCounters = gl.ActiveGateAdmissionCounters{ControlOwnerType: ownerType, ControlOwnerID: requestID, ControlOwnerEpoch: cloneInt64(&epoch), AdmittedPostingCount: count, ReopenPostingAdmitted: true, FirstAdmittedLedgerPosition: first, LastAdmittedLedgerPosition: last}
	}
	return gl.SoftCloseGateResult{Operation: closeOperation, Gate: gate, PolicyReference: "reopen-policy", ValidationOutcome: "authoritative", NextAction: "evaluate-reopen-admission-summary"}
}

func recloseBeginResult(state ControlState, closeResult gl.SoftCloseGateResult, now time.Time) gl.SoftCloseGateResult {
	requestID := state.ReopenRequest.ID
	epoch := state.ReopenRequest.ControlAuthorityEpoch
	summary := cloneSummary(closeResult.Gate.LastFrozenAdmissionSummary)
	priorOwnerType := gl.ScopedReopenGateOwnerType
	priorGateMode := gl.PostingGateScopedReopen
	if state.ReopenRequest.Mode == ReopenModeOperational {
		priorOwnerType = gl.OperationalReopenGateOwnerType
		priorGateMode = gl.PostingGateOperationalReopen
	}
	return gl.SoftCloseGateResult{Operation: gl.BeginRecloseGateOperation, Gate: gl.PeriodPostingGate{ID: closeResult.Gate.ID, AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, GateMode: gl.PostingGateCloseOnly, PeriodStateVersion: state.Period.StateVersion, GateVersion: closeResult.Gate.GateVersion + 1, AggregateVersion: closeResult.Gate.GateVersion + 1, NextLedgerPosition: 100, BarrierLedgerPosition: 99, PriorControlProcessID: cloneUUID(&requestID), PriorControlProcessType: priorOwnerType, PriorControlEpoch: cloneInt64(&epoch), PriorGateMode: priorGateMode, ActiveCloseType: gl.CloseTypeReclose, ActiveAdmissionCounters: gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.PostingBarrierOwnerType}, LastFrozenAdmissionSummary: summary}, PolicyReference: "reclose-policy", ValidationOutcome: "authoritative", NextAction: "resume-reclose"}
}
