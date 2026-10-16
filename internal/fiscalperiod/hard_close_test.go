package fiscalperiod

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

type hardCloseGateStub struct {
	result gl.SoftCloseGateResult
	err    error
	calls  []gl.AcquirePostingBarrierCommand
}

type fiscalPeriodAuthorizerStub struct {
	decision AuthorizationDecision
	err      error
}

func (stub fiscalPeriodAuthorizerStub) Authorize(context.Context, Actor, string, uuid.UUID) (AuthorizationDecision, error) {
	return stub.decision, stub.err
}

func (stub *hardCloseGateStub) AcquirePostingBarrier(_ context.Context, _ gl.Actor, command gl.AcquirePostingBarrierCommand) (gl.SoftCloseGateResult, error) {
	stub.calls = append(stub.calls, command)
	if stub.err != nil {
		return gl.SoftCloseGateResult{}, stub.err
	}
	result := stub.result
	closeRunID := command.CloseRunID
	softCloseRunID := command.SoftCloseRunID
	epoch := command.SoftCloseControlEpoch
	result.Gate.ActiveCloseRunID = &closeRunID
	result.Gate.PriorControlProcessID = &softCloseRunID
	result.Gate.PriorControlEpoch = &epoch
	if result.Gate.LastFrozenAdmissionSummary != nil {
		result.Gate.LastFrozenAdmissionSummary.ControlOwnerID = command.SoftCloseRunID
		result.Gate.LastFrozenAdmissionSummary.ControlOwnerEpoch = &epoch
	}
	result.Gate.ActiveAdmissionCounters.ControlOwnerID = command.CloseRunID
	return result, nil
}

func (stub *hardCloseGateStub) GetPostingGateStatus(_ context.Context, _, _ uuid.UUID) (gl.PostingGateStatus, error) {
	return stub.result.Gate, stub.err
}

func TestStartHardCloseCreatesCandidateAndAppliesAuthoritativeBarrier(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{}
	gate.result = hardCloseResult(command, uuid.New())
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC) })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.StartHardClose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateClosing || result.State.Period.StateVersion != 2 || result.State.Period.AggregateVersion != 2 {
		t.Fatalf("period after hard-close handoff = %#v", result.State.Period)
	}
	if result.State.SoftClose.Status != SoftCloseRunHandoffPending || result.State.ControlEpoch.Status != ControlEpochHandoffPending {
		t.Fatalf("soft-close handoff = %#v / %#v", result.State.SoftClose, result.State.ControlEpoch)
	}
	if result.State.CloseRun == nil || result.State.CloseRun.Status != CloseRunBarrierAcquired || result.State.CloseRun.BarrierEvidence == nil {
		t.Fatalf("close run after barrier = %#v", result.State.CloseRun)
	}
	if len(gate.calls) != 1 || gate.calls[0].CloseRunID != result.State.CloseRun.ID || gate.calls[0].ExpectedGateVersion != command.ExpectedGateVersion {
		t.Fatalf("barrier calls = %#v", gate.calls)
	}
	events := repository.Events()
	if len(events) != 3 || events[0].EventType != "GateAdmissionSummaryRecorded" || events[1].EventType != "SoftCloseHandoffStarted" || events[2].EventType != "PeriodStateChanged" {
		t.Fatalf("events = %#v", events)
	}
}

func TestStartHardCloseRejectsAnAuthorizationDecisionWithTheWrongPermission(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{result: hardCloseResult(command, uuid.New())}
	service, err := NewService(repository, fiscalPeriodAuthorizerStub{decision: AuthorizationDecision{Allowed: true, Permission: "finance.fpm.unrelated", DecisionReference: uuid.New()}}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartHardClose(context.Background(), testActor(), command); !errors.Is(err, ErrFiscalPeriodAuthorizationDenied) {
		t.Fatalf("authorization error = %v", err)
	}
	current, err := repository.Get(context.Background(), command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	if current.CloseRun != nil || len(gate.calls) != 0 {
		t.Fatalf("unauthorized hard-close mutated state or called GL: state=%#v calls=%d", current, len(gate.calls))
	}
}

func TestStartHardCloseReconcilesAnAlreadyAcquiredBarrierFromGLStatus(t *testing.T) {
	state, command := hardCloseFixture()
	closeRunID := uuid.New()
	now := time.Date(2026, 10, 16, 7, 1, 0, 0, time.UTC)
	state.CloseRun = &CloseRun{
		ID: closeRunID, AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunInitiating, AggregateVersion: 1,
		CommandFingerprint: startHardCloseFingerprint(command), BarrierCommandID: uuid.New(),
		BarrierIdempotencyKey: command.IdempotencyKey + ":posting-barrier", CreatedAt: now, UpdatedAt: now,
	}
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{result: hardCloseResult(command, closeRunID)}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.StartHardClose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if len(gate.calls) != 0 || result.State.CloseRun.ID != closeRunID || result.State.Period.State != PeriodStateClosing {
		t.Fatalf("status recovery result = %#v, acquire calls = %#v", result, gate.calls)
	}
	if result.GateResult.ValidationOutcome != "reconciled" {
		t.Fatalf("recovery outcome = %q", result.GateResult.ValidationOutcome)
	}
}

func TestStartHardCloseIsIdempotentAndChangedFingerprintConflicts(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{}
	gate.result = hardCloseResult(command, uuid.New())
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	first, err := service.StartHardClose(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.StartHardClose(context.Background(), actor, command)
	if err != nil || !replay.Replayed || replay.State.CloseRun.ID != first.State.CloseRun.ID {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	if len(gate.calls) != 1 {
		t.Fatalf("barrier calls after replay = %d", len(gate.calls))
	}

	changed := command
	changed.SoftClosePolicyVersion = "soft-close-policy-v2"
	if _, err := service.StartHardClose(context.Background(), actor, changed); !errors.Is(err, ErrFiscalPeriodIdempotencyConflict) {
		t.Fatalf("changed retry error = %v", err)
	}
}

func TestStartHardCloseDistinguishesBarrierRejectionFromUnavailableDependency(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{err: gl.ErrSoftCloseGateVersionConflict}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartHardClose(context.Background(), testActor(), command); !errors.Is(err, ErrPostingBarrierRejected) {
		t.Fatalf("barrier rejection error = %v", err)
	}
}

func TestApplyPostingGateResultRejectsMismatchedAuthoritativeEvidenceWithoutMutation(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{err: errors.New("GL unavailable")}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.StartHardClose(context.Background(), testActor(), command); !errors.Is(err, ErrPostingBarrierUnavailable) {
		t.Fatalf("barrier failure = %v", err)
	}
	failed, err := repository.Get(context.Background(), command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Period.State != PeriodStateSoftClosed || failed.SoftClose.Status != SoftCloseRunActive || failed.CloseRun == nil || failed.CloseRun.Status != CloseRunFailed {
		t.Fatalf("failed handoff state = %#v", failed)
	}

	state, command = hardCloseFixture()
	repository = NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate = &hardCloseGateStub{}
	gate.result = hardCloseResult(command, uuid.New())
	service, err = NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartHardClose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	mismatched := started.GateResult
	mismatched.Gate.PriorControlProcessID = cloneUUID(&uuid.UUID{})
	apply := ApplyPostingGateResultCommand{
		CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		SoftCloseRunID: command.SoftCloseRunID, CloseRunID: started.State.CloseRun.ID, ExpectedPeriodVersion: 2,
		ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseControlEpoch: command.SoftCloseControlEpoch,
		SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: mismatched, IdempotencyKey: "different-apply",
		CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	if _, err := service.ApplyPostingGateResult(context.Background(), testActor(), apply); !errors.Is(err, ErrPostingBarrierResultConflict) {
		t.Fatalf("mismatched result error = %v", err)
	}
	current, err := repository.Get(context.Background(), command.AccountingScopeID, command.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Period.State != PeriodStateClosing || current.CloseRun.Status != CloseRunBarrierAcquired {
		t.Fatalf("state after mismatched replay = %#v", current)
	}
}

func TestApplyPostingGateResultReplaysSameEvidence(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{}
	gate.result = hardCloseResult(command, uuid.New())
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartHardClose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	replayCommand := ApplyPostingGateResultCommand{
		CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		SoftCloseRunID: command.SoftCloseRunID, CloseRunID: started.State.CloseRun.ID, ExpectedPeriodVersion: 2,
		ExpectedGateVersion: command.ExpectedGateVersion, SoftCloseControlEpoch: command.SoftCloseControlEpoch,
		SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: started.GateResult, IdempotencyKey: "manual-apply-replay",
		CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	replay, err := service.ApplyPostingGateResult(context.Background(), testActor(), replayCommand)
	if err != nil || !replay.Replayed || len(repository.Events()) != 3 {
		t.Fatalf("result replay = %#v, err = %v, events = %#v", replay, err, repository.Events())
	}
}

func TestApplyPostingGateResultRejectsChangedCommandAfterStateRecovery(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{}
	gate.result = hardCloseResult(command, uuid.New())
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartHardClose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	changedResult := started.GateResult
	changedResult.Gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint = "sha256:changed"
	changed := ApplyPostingGateResultCommand{
		CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		SoftCloseRunID: command.SoftCloseRunID, CloseRunID: started.State.CloseRun.ID, ExpectedPeriodVersion: command.ExpectedPeriodVersion,
		ExpectedGateVersion: command.ExpectedGateVersion + 1, SoftCloseControlEpoch: command.SoftCloseControlEpoch,
		SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: changedResult, IdempotencyKey: "changed-after-recovery",
		CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	if _, err := service.ApplyPostingGateResult(context.Background(), testActor(), changed); !errors.Is(err, ErrPostingBarrierResultConflict) {
		t.Fatalf("changed recovered result error = %v", err)
	}
}

func TestApplyPostingGateResultReplaysAuthoritativeEvidenceAfterServiceRestart(t *testing.T) {
	state, command := hardCloseFixture()
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &hardCloseGateStub{result: hardCloseResult(command, uuid.New())}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartHardClose(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}

	restarted, err := NewService(repository, AllowAllAuthorizer{}, gate, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	startReplay, err := restarted.StartHardClose(context.Background(), testActor(), command)
	if err != nil || !startReplay.Replayed || startReplay.GateResult.Gate.ID != started.GateResult.Gate.ID {
		t.Fatalf("restarted start replay = %#v, err = %v", startReplay, err)
	}
	replayCommand := ApplyPostingGateResultCommand{
		CommandID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID,
		SoftCloseRunID: command.SoftCloseRunID, CloseRunID: started.State.CloseRun.ID, ExpectedPeriodVersion: 99,
		ExpectedGateVersion: 99, SoftCloseControlEpoch: command.SoftCloseControlEpoch,
		SoftClosePolicyVersion: command.SoftClosePolicyVersion, Result: started.GateResult, IdempotencyKey: "restart-apply-replay",
		CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	result, err := restarted.ApplyPostingGateResult(context.Background(), testActor(), replayCommand)
	if err != nil || !result.Replayed || result.GateResult.Gate.ID != started.GateResult.Gate.ID {
		t.Fatalf("restarted replay = %#v, err = %v", result, err)
	}
	if len(repository.Events()) != 3 {
		t.Fatalf("restarted replay emitted events = %d", len(repository.Events()))
	}
}

func hardCloseFixture() (ControlState, StartHardCloseCommand) {
	scopeID, periodID, softCloseRunID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 16, 7, 0, 0, 0, time.UTC)
	return ControlState{
		Period:       FiscalPeriod{ID: periodID, AccountingScopeID: scopeID, State: PeriodStateSoftClosed, StateVersion: 1, AggregateVersion: 1, ActiveSoftCloseRunID: cloneUUID(&softCloseRunID), CreatedAt: now, UpdatedAt: now},
		SoftClose:    SoftCloseRun{ID: softCloseRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Status: SoftCloseRunActive, ControlEpoch: 1, OpenedGateVersion: 2, PolicyVersion: "soft-close-policy-v1", AggregateVersion: 1, CreatedAt: now, UpdatedAt: now},
		ControlEpoch: SoftCloseControlEpoch{SoftCloseRunID: softCloseRunID, FiscalPeriodID: periodID, Number: 1, OpenedGateVersion: 2, PolicyVersion: "soft-close-policy-v1", Status: ControlEpochActive, AggregateVersion: 1, CreatedAt: now, UpdatedAt: now},
	}, StartHardCloseCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, SoftCloseRunID: softCloseRunID, ExpectedPeriodVersion: 1, ExpectedGateVersion: 2, SoftCloseControlEpoch: 1, SoftClosePolicyVersion: "soft-close-policy-v1", IdempotencyKey: "hard-close-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
}

func hardCloseResult(command StartHardCloseCommand, closeRunID uuid.UUID) gl.SoftCloseGateResult {
	priorEpoch := command.SoftCloseControlEpoch
	priorOwner := command.SoftCloseRunID
	closeOwner := closeRunID
	now := time.Date(2026, 10, 16, 7, 1, 0, 0, time.UTC)
	summary := gl.FrozenGateAdmissionSummary{ControlOwnerType: gl.SoftCloseGateOwnerType, ControlOwnerID: command.SoftCloseRunID, ControlOwnerEpoch: &priorEpoch, FrozenAtGateVersion: command.ExpectedGateVersion + 1, FrozenAt: now, FrozenByCommandFingerprint: "sha256:gl-acquire", FrozenByCommandType: gl.PostingBarrierAcquireCommandIdentity}
	return gl.SoftCloseGateResult{Operation: gl.PostingBarrierAcquireOperation, Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: command.AccountingScopeID, FiscalPeriodID: command.FiscalPeriodID, GateMode: gl.PostingGateCloseOnly, PeriodStateVersion: command.ExpectedPeriodVersion, GateVersion: command.ExpectedGateVersion + 1, AggregateVersion: command.ExpectedGateVersion + 1, NextLedgerPosition: 20, ActiveCloseRunID: &closeOwner, ActiveCloseType: gl.CloseTypeInitialHardClose, PriorControlProcessID: &priorOwner, PriorControlProcessType: gl.SoftCloseGateOwnerType, PriorControlEpoch: &priorEpoch, PriorGateMode: gl.PostingGateSoftClosePolicy, BarrierLedgerPosition: 19, ActiveAdmissionCounters: gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.PostingBarrierOwnerType, ControlOwnerID: closeRunID}, LastFrozenAdmissionSummary: &summary, CreatedAt: now, UpdatedAt: now}, DecisionReference: uuid.New(), PolicyReference: "gl-policy", AuthorizationPolicyVersion: "gl-policy-v1", ValidationOutcome: "valid", AuditReference: uuid.New(), NextAction: "continue-close"}
}

func testActor() Actor { return Actor{UserID: uuid.New(), SubjectReference: "test-subject"} }

var _ PostingGateApplicationPort = (*hardCloseGateStub)(nil)
