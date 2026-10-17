package fiscalperiod

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

type recoveryDecisionPortStub struct {
	err error
}

type recoveryCommitFailOnceRepository struct {
	delegate *MemoryRepository
	failOn   int
	commits  int
	failure  error
}

func (repository *recoveryCommitFailOnceRepository) Get(ctx context.Context, scopeID, periodID uuid.UUID) (ControlState, error) {
	return repository.delegate.Get(ctx, scopeID, periodID)
}

func (repository *recoveryCommitFailOnceRepository) Commit(ctx context.Context, mutation Mutation) error {
	repository.commits++
	if repository.failOn > 0 && repository.commits == repository.failOn {
		repository.failOn = 0
		return repository.failure
	}
	return repository.delegate.Commit(ctx, mutation)
}

func (stub recoveryDecisionPortStub) Validate(_ context.Context, _ Actor, decision ControlRecoveryApprovalDecision) error {
	if stub.err != nil {
		return stub.err
	}
	return decision.Validate()
}

func recoveryAuthorityFixture(now time.Time, owner Actor) *ControlAuthority {
	return &ControlAuthority{CurrentOwner: owner, AuthorityEpoch: 1, Version: 1, ExpiresAt: now.Add(-time.Minute), CutoffAt: now.Add(-time.Hour)}
}

func recoverySoftGate(state ControlState, gateVersion int64) gl.SoftCloseGateResult {
	softRunID := state.SoftClose.ID
	epoch := state.ControlEpoch.Number
	return gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{
		ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID,
		GateMode: gl.PostingGateSoftClosePolicy, PeriodStateVersion: state.Period.StateVersion, GateVersion: gateVersion, AggregateVersion: gateVersion, NextLedgerPosition: 100,
		ActiveSoftCloseRunID: &softRunID, ActiveSoftCloseControlEpoch: epoch, SoftClosePolicyVersion: state.SoftClose.PolicyVersion,
		ActiveAdmissionCounters: gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.SoftCloseGateOwnerType, ControlOwnerID: softRunID, ControlOwnerEpoch: &epoch},
		CreatedAt:               time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC),
	}}
}

func recoveryCloseGate(state ControlState, gateVersion int64) gl.SoftCloseGateResult {
	softRunID := state.SoftClose.ID
	closeRunID := state.CloseRun.ID
	epoch := state.ControlEpoch.Number
	summary := gl.FrozenGateAdmissionSummary{ControlOwnerType: gl.SoftCloseGateOwnerType, ControlOwnerID: softRunID, ControlOwnerEpoch: &epoch, FrozenAtGateVersion: gateVersion, FrozenAt: time.Date(2026, 10, 16, 6, 1, 0, 0, time.UTC), FrozenByCommandFingerprint: "sha256:gl-acquire", FrozenByCommandType: gl.PostingBarrierAcquireCommandIdentity}
	return gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{
		ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID,
		GateMode: gl.PostingGateCloseOnly, PeriodStateVersion: state.Period.StateVersion, GateVersion: gateVersion, AggregateVersion: gateVersion, NextLedgerPosition: 100,
		ActiveCloseRunID: &closeRunID, ActiveCloseType: state.CloseRun.CloseType, PriorControlProcessID: &softRunID, PriorControlProcessType: gl.SoftCloseGateOwnerType, PriorControlEpoch: &epoch, PriorGateMode: gl.PostingGateSoftClosePolicy,
		BarrierLedgerPosition: 99, ActiveAdmissionCounters: gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.PostingBarrierOwnerType, ControlOwnerID: closeRunID}, LastFrozenAdmissionSummary: &summary,
		CreatedAt: time.Date(2026, 10, 16, 6, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 10, 16, 6, 1, 0, 0, time.UTC),
	}}
}

func recoveryApproval(candidate string, subjectVersion int64, approver Actor, now time.Time) ControlRecoveryApprovalDecision {
	return ControlRecoveryApprovalDecision{RequestID: uuid.New(), DecisionID: uuid.New(), DecisionVersion: 1, Outcome: CloseApprovalApproved, Reason: "Independent recovery review completed.", PolicyVersion: "recovery-policy-v1", SubjectVersion: subjectVersion, CandidateFingerprint: candidate, ApproverUserID: approver.UserID, DecisionReference: uuid.New(), SeparationOfDutiesReference: "sod-recovery-1", RecordedAt: now}
}

func takeoverCommand(state ControlState, target string, now time.Time, actor Actor) TakeOverPeriodControlCommand {
	processID := state.SoftClose.ID
	processVersion := state.SoftClose.AggregateVersion
	if target == ControlProcessCloseRun {
		processID = state.CloseRun.ID
		processVersion = state.CloseRun.AggregateVersion
	}
	command := TakeOverPeriodControlCommand{CommandID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, Target: target, ProcessID: processID, ExpectedPeriodVersion: state.Period.AggregateVersion, ExpectedProcessVersion: processVersion, ExpectedGateVersion: 2, ExpectedControlAuthorityVersion: 1, ExpectedControlAuthorityEpoch: 1, Reason: "Recover the interrupted owner", EvidenceReference: "incident://recovery-1", NewAuthorityExpiresAt: now.Add(time.Hour), IdempotencyKey: "takeover-" + target, CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	command.Approval = recoveryApproval(takeoverCandidateFingerprint(command), command.ExpectedProcessVersion, Actor{UserID: uuid.New(), SubjectReference: "approver"}, now)
	_ = actor
	return command
}

func TestTakeOverSoftClosePreservesIdentityAndNeverMutatesGL(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, _ := hardCloseFixture()
	priorOwner := Actor{UserID: uuid.New(), SubjectReference: "prior-owner"}
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, priorOwner)
	gate := &hardCloseGateStub{result: recoverySoftGate(state, 2)}
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	command := takeoverCommand(state, ControlProcessSoftClose, now, actor)
	result, err := service.TakeOverPeriodControl(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.SoftClose.ID != state.SoftClose.ID || result.State.SoftClose.ControlAuthority == nil || result.State.SoftClose.ControlAuthority.CurrentOwner.UserID != actor.UserID || result.State.SoftClose.ControlAuthority.AuthorityEpoch != 2 || result.State.SoftClose.ControlAuthority.Version != 2 || len(result.State.SoftClose.ControlAuthority.OwnershipHistory) != 1 {
		t.Fatalf("takeover projection = %#v", result.State.SoftClose)
	}
	if len(gate.calls) != 0 || gate.releaseCalls != 0 || gate.finalizeCalls != 0 {
		t.Fatalf("takeover mutated GL: acquire=%d release=%d finalize=%d", len(gate.calls), gate.releaseCalls, gate.finalizeCalls)
	}
	events := repository.Events()
	if len(events) != 1 || events[0].EventType != "SoftCloseResumed" {
		t.Fatalf("takeover events = %#v", events)
	}
	replay, err := service.TakeOverPeriodControl(context.Background(), actor, command)
	if err != nil || !replay.Replayed || len(repository.Events()) != 1 {
		t.Fatalf("takeover replay = %#v, err=%v, events=%d", replay, err, len(repository.Events()))
	}
	changed := command
	changed.Reason = "changed reason"
	if _, err := service.TakeOverPeriodControl(context.Background(), actor, changed); !errors.Is(err, ErrFiscalPeriodIdempotencyConflict) {
		t.Fatalf("changed duplicate error = %v", err)
	}
}

func TestTakeOverRejectsActiveOrMissingAuthorityAndGateMismatch(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	actor := testActor()
	newService := func(state ControlState, gate *hardCloseGateStub) *Service {
		repository := NewMemoryRepository()
		if err := repository.Seed(state); err != nil {
			t.Fatal(err)
		}
		service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		return service
	}

	state, _ := hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
	gate := &hardCloseGateStub{result: recoverySoftGate(state, 3)}
	service := newService(state, gate)
	command := takeoverCommand(state, ControlProcessSoftClose, now, actor)
	if _, err := service.TakeOverPeriodControl(context.Background(), actor, command); !errors.Is(err, ErrPostingBarrierResultConflict) {
		t.Fatalf("gate mismatch error = %v", err)
	}
	if len(gate.calls) != 0 {
		t.Fatalf("gate mismatch called a GL mutation: %#v", gate.calls)
	}

	state, _ = hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
	state.SoftClose.ControlAuthority.ExpiresAt = now.Add(time.Hour)
	gate = &hardCloseGateStub{result: recoverySoftGate(state, 2)}
	service = newService(state, gate)
	command = takeoverCommand(state, ControlProcessSoftClose, now, actor)
	if _, err := service.TakeOverPeriodControl(context.Background(), actor, command); !errors.Is(err, ErrFiscalPeriodRecoveryNotEligible) {
		t.Fatalf("active authority error = %v", err)
	}
	if len(gate.calls) != 0 {
		t.Fatalf("active authority called GL: %#v", gate.calls)
	}

	state, _ = hardCloseFixture()
	gate = &hardCloseGateStub{result: recoverySoftGate(state, 2)}
	service = newService(state, gate)
	command = takeoverCommand(state, ControlProcessSoftClose, now, actor)
	if _, err := service.TakeOverPeriodControl(context.Background(), actor, command); !errors.Is(err, ErrFiscalPeriodControlAuthorityMissing) {
		t.Fatalf("missing authority error = %v", err)
	}
	if len(gate.calls) != 0 {
		t.Fatalf("missing authority called GL: %#v", gate.calls)
	}

	state, _ = hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
	state.CloseRun = &CloseRun{ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunInitiating, AggregateVersion: 1, CommandFingerprint: "sha256:legacy-close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "legacy-close", SealStatus: CloseRunSealNotRequested, CreatedAt: now, UpdatedAt: now}
	gate = &hardCloseGateStub{result: recoveryCloseGate(state, 2)}
	service = newService(state, gate)
	command = takeoverCommand(state, ControlProcessCloseRun, now, actor)
	if _, err := service.TakeOverPeriodControl(context.Background(), actor, command); !errors.Is(err, ErrFiscalPeriodControlAuthorityMissing) {
		t.Fatalf("legacy close-run authority error = %v", err)
	}
	if len(gate.calls) != 0 {
		t.Fatalf("legacy close-run authority called GL: %#v", gate.calls)
	}
}

func TestTakeOverCloseRunReconcilesExistingGLBarrierWithoutMutation(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, _ := hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
	closeRunID := uuid.New()
	state.CloseRun = &CloseRun{ID: closeRunID, AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunInitiating, AggregateVersion: 1, CommandFingerprint: "sha256:close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "close-barrier", ControlAuthority: cloneControlAuthority(state.SoftClose.ControlAuthority), CreatedAt: now, UpdatedAt: now}
	gate := &hardCloseGateStub{result: recoveryCloseGate(state, 3)}
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	command := takeoverCommand(state, ControlProcessCloseRun, now, actor)
	result, err := service.TakeOverPeriodControl(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateClosing || result.State.CloseRun == nil || result.State.CloseRun.BarrierEvidence == nil || result.State.CloseRun.ControlAuthority == nil || result.State.CloseRun.ControlAuthority.CurrentOwner.UserID != actor.UserID {
		t.Fatalf("close takeover state = %#v", result.State)
	}
	if len(gate.calls) != 0 || gate.releaseCalls != 0 || gate.finalizeCalls != 0 {
		t.Fatalf("close takeover mutated GL: acquire=%d release=%d finalize=%d", len(gate.calls), gate.releaseCalls, gate.finalizeCalls)
	}
	events := repository.Events()
	if len(events) != 4 || events[len(events)-1].EventType != "CloseRunResumed" {
		t.Fatalf("close takeover events = %#v", events)
	}
}

func TestTakeOverCloseRunRetriesAfterReconciliationCommitFailure(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, _ := hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
	state.CloseRun = &CloseRun{ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunInitiating, AggregateVersion: 1, CommandFingerprint: "sha256:close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "close-barrier", ControlAuthority: cloneControlAuthority(state.SoftClose.ControlAuthority), CreatedAt: now, UpdatedAt: now}
	gate := &hardCloseGateStub{result: recoveryCloseGate(state, 3)}
	base := NewMemoryRepository()
	if err := base.Seed(state); err != nil {
		t.Fatal(err)
	}
	repository := &recoveryCommitFailOnceRepository{delegate: base, failOn: 2, failure: ErrFiscalPeriodAuditUnavailable}
	service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	command := takeoverCommand(state, ControlProcessCloseRun, now, actor)
	if _, err := service.TakeOverPeriodControl(context.Background(), actor, command); !errors.Is(err, ErrFiscalPeriodAuditUnavailable) {
		t.Fatalf("first takeover error = %v", err)
	}
	interrupted, err := base.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
	if err != nil {
		t.Fatal(err)
	}
	if interrupted.CloseRun == nil || interrupted.CloseRun.BarrierEvidence == nil || interrupted.CloseRun.ControlAuthority.CurrentOwner.UserID == actor.UserID {
		t.Fatalf("partial reconciliation did not preserve retry state = %#v", interrupted)
	}

	result, err := service.TakeOverPeriodControl(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.CloseRun == nil || result.State.CloseRun.ControlAuthority == nil || result.State.CloseRun.ControlAuthority.CurrentOwner.UserID != actor.UserID {
		t.Fatalf("retry did not complete takeover = %#v", result.State.CloseRun)
	}
	if events := base.Events(); len(events) != 4 || events[3].EventType != "CloseRunResumed" {
		t.Fatalf("retry duplicated reconciliation events = %#v", events)
	}
}

func extensionFixture(now time.Time, status string) (ControlState, uuid.UUID) {
	state, _ := hardCloseFixture()
	state.Period.State = PeriodStateClosing
	state.Period.ActiveSoftCloseRunID = nil
	closeRunID := uuid.New()
	exceptionID := uuid.New()
	owner := Actor{UserID: uuid.New(), SubjectReference: "close-owner"}
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, owner)
	approvalRequestID := uuid.New()
	exception := CloseException{ID: exceptionID, CloseRunID: closeRunID, Code: "FX_BLOCKER", Status: status, ApprovalRequestID: approvalRequestID, DueAt: timePtr(now.Add(-2 * time.Hour)), ExpiresAt: timePtr(now.Add(-time.Hour)), CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour), Blocker: "Awaiting owner evidence"}
	if status == CloseExceptionApproved || status == CloseExceptionRejected {
		exception.ApprovalDecision = &CloseApprovalDecision{RequestID: approvalRequestID, DecisionID: uuid.New(), DecisionVersion: 1, Outcome: map[string]string{CloseExceptionApproved: CloseApprovalApproved, CloseExceptionRejected: CloseApprovalRejected}[status], Reason: "Original decision", PolicyVersion: "close-policy-v1", SubjectVersion: 1, CandidateFingerprint: "sha256:original", ApproverUserID: uuid.New(), DecisionReference: uuid.New(), RecordedAt: now.Add(-2 * time.Hour)}
	}
	state.CloseRun = &CloseRun{ID: closeRunID, AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunInProgress, AggregateVersion: 1, CommandFingerprint: "sha256:close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "close-barrier", Exceptions: []CloseException{exception}, ControlAuthority: cloneControlAuthority(state.SoftClose.ControlAuthority), SealStatus: CloseRunSealNotRequested, CreatedAt: now.Add(-3 * time.Hour), UpdatedAt: now.Add(-2 * time.Hour)}
	return state, exceptionID
}

func timePtr(value time.Time) *time.Time { return &value }

func extensionCommand(state ControlState, exceptionID uuid.UUID, now time.Time) ExtendCloseExceptionCommand {
	command := ExtendCloseExceptionCommand{CommandID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, CloseRunID: state.CloseRun.ID, ExceptionID: exceptionID, ExpectedPeriodVersion: state.Period.AggregateVersion, ExpectedCloseRunVersion: state.CloseRun.AggregateVersion, ExpectedGateVersion: 2, ExpectedControlAuthorityVersion: 1, ExpectedControlAuthorityEpoch: 1, Justification: "The owning capability needs a documented recovery window.", NewDueAt: now.Add(2 * time.Hour), NewExpiresAt: now.Add(4 * time.Hour), EvidenceReference: "incident://exception-extension-1", IdempotencyKey: "extend-exception-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	command.Approval = recoveryApproval(extensionCandidateFingerprint(command), command.ExpectedCloseRunVersion, Actor{UserID: uuid.New(), SubjectReference: "approver"}, now)
	return command
}

func TestExtendCloseExceptionPreservesUnresolvedStateAndHistory(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, exceptionID := extensionFixture(now, CloseExceptionUnresolved)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	command := extensionCommand(state, exceptionID, now)
	actor := testActor()
	result, err := service.ExtendCloseException(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	exception := result.State.CloseRun.Exceptions[0]
	if exception.Status != CloseExceptionUnresolved || len(exception.ExtensionHistory) != 1 || exception.DueAt == nil || !exception.DueAt.Equal(command.NewDueAt) || exception.ExpiresAt == nil || !exception.ExpiresAt.Equal(command.NewExpiresAt) {
		t.Fatalf("extended exception = %#v", exception)
	}
	if len(repository.Events()) != 0 || len(repository.Audits()) != 1 {
		t.Fatalf("extension events/audits = %d/%d", len(repository.Events()), len(repository.Audits()))
	}
	replay, err := service.ExtendCloseException(context.Background(), actor, command)
	if err != nil || !replay.Replayed || len(replay.State.CloseRun.Exceptions[0].ExtensionHistory) != 1 {
		t.Fatalf("extension replay = %#v, err=%v", replay, err)
	}
	changed := command
	changed.Justification = "changed justification"
	if _, err := service.ExtendCloseException(context.Background(), actor, changed); !errors.Is(err, ErrFiscalPeriodIdempotencyConflict) {
		t.Fatalf("changed extension duplicate = %v", err)
	}
}

func TestExtendApprovedExpiredExceptionAndRejectTerminalCases(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, exceptionID := extensionFixture(now, CloseExceptionApproved)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	command := extensionCommand(state, exceptionID, now)
	result, err := service.ExtendCloseException(context.Background(), testActor(), command)
	if err != nil || result.State.CloseRun.Exceptions[0].Status != CloseExceptionApproved {
		t.Fatalf("approved expired extension = %#v, err=%v", result.State.CloseRun.Exceptions[0], err)
	}

	for _, status := range []string{CloseExceptionRejected} {
		state, exceptionID = extensionFixture(now, status)
		repository = NewMemoryRepository()
		if err := repository.Seed(state); err != nil {
			t.Fatal(err)
		}
		service, err = NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		if _, err := service.ExtendCloseException(context.Background(), testActor(), extensionCommand(state, exceptionID, now)); !errors.Is(err, ErrCloseExceptionExtensionRejected) {
			t.Fatalf("rejected exception error = %v", err)
		}
	}
	state, exceptionID = extensionFixture(now, CloseExceptionUnresolved)
	state.CloseRun.Status = CloseRunFinalizing
	repository = NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err = NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ExtendCloseException(context.Background(), testActor(), extensionCommand(state, exceptionID, now)); !errors.Is(err, ErrCloseExceptionExtensionRejected) {
		t.Fatalf("finalizing close-run error = %v", err)
	}
}

func TestExtendCloseExceptionRejectsInvalidFutureWindow(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	cases := []struct {
		name   string
		mutate func(*ExtendCloseExceptionCommand)
	}{
		{name: "due in the past", mutate: func(command *ExtendCloseExceptionCommand) { command.NewDueAt = now.Add(-time.Minute) }},
		{name: "expiry in the past", mutate: func(command *ExtendCloseExceptionCommand) { command.NewExpiresAt = now.Add(-time.Minute) }},
		{name: "due after expiry", mutate: func(command *ExtendCloseExceptionCommand) {
			command.NewDueAt = now.Add(4 * time.Hour)
			command.NewExpiresAt = now.Add(2 * time.Hour)
		}},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state, exceptionID := extensionFixture(now, CloseExceptionUnresolved)
			repository := NewMemoryRepository()
			if err := repository.Seed(state); err != nil {
				t.Fatal(err)
			}
			service, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			command := extensionCommand(state, exceptionID, now)
			test.mutate(&command)
			if _, err := service.ExtendCloseException(context.Background(), testActor(), command); !errors.Is(err, ErrInvalidFiscalPeriodCommand) {
				t.Fatalf("invalid window error = %v", err)
			}
			if len(repository.Events()) != 0 || len(repository.Audits()) != 0 {
				t.Fatalf("invalid window established evidence: events=%d audits=%d", len(repository.Events()), len(repository.Audits()))
			}
		})
	}
}

func TestExtendCloseExceptionAllowsEqualFutureDueAndExpiry(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, exceptionID := extensionFixture(now, CloseExceptionUnresolved)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	command := extensionCommand(state, exceptionID, now)
	command.NewExpiresAt = command.NewDueAt
	command.Approval = recoveryApproval(extensionCandidateFingerprint(command), command.ExpectedCloseRunVersion, Actor{UserID: uuid.New(), SubjectReference: "approver"}, now)
	if _, err := service.ExtendCloseException(context.Background(), testActor(), command); err != nil {
		t.Fatalf("equal future due/expiry was rejected: %v", err)
	}
}

func TestRecoverySerializationPreservesAuthorityAndExceptionHistory(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, exceptionID := extensionFixture(now, CloseExceptionUnresolved)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ExtendCloseException(context.Background(), testActor(), extensionCommand(state, exceptionID, now))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := marshalCloseRunDetails(result.State.CloseRun)
	if err != nil {
		t.Fatal(err)
	}
	roundTripped := CloseRun{}
	if err := unmarshalCloseRunDetails(&roundTripped, encoded); err != nil {
		t.Fatal(err)
	}
	if roundTripped.ControlAuthority == nil || roundTripped.ControlAuthority.CurrentOwner.UserID != state.CloseRun.ControlAuthority.CurrentOwner.UserID || len(roundTripped.Exceptions) != 1 || len(roundTripped.Exceptions[0].ExtensionHistory) != 1 {
		t.Fatalf("recovery details round trip = %#v", roundTripped)
	}
}

func TestControlAuthorityCutoffIsImmutable(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, _ := hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "owner"})
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	after := cloneControlState(state)
	after.SoftClose.ControlAuthority.CutoffAt = now
	if err := repository.Commit(context.Background(), Mutation{Before: state, After: after}); !errors.Is(err, ErrFiscalPeriodPolicyConflict) {
		t.Fatalf("cutoff mutation error = %v", err)
	}
}

func TestRecoveryHistoriesAreAppendOnly(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, _ := hardCloseFixture()
	priorOwner := Actor{UserID: uuid.New(), SubjectReference: "prior-owner"}
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, priorOwner)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}

	change := ControlOwnershipChange{PriorOwner: priorOwner, NewOwner: testActor(), ChangedAt: now, ExpectedGateVersion: 2, Reason: "recover", EvidenceReference: "incident://recovery", AuthorizationReference: uuid.New(), AuthorizationPolicyReference: "policy", AuthorizationPolicyVersion: "v1", ApprovalRequestID: uuid.New(), ApprovalDecisionID: uuid.New(), ApprovalDecisionVersion: 1, ApprovalDecisionReference: uuid.New()}
	withHistory := cloneControlState(state)
	withHistory.SoftClose.ControlAuthority.OwnershipHistory = []ControlOwnershipChange{change}
	withHistory.SoftClose.ControlAuthority.CurrentOwner = change.NewOwner
	withHistory.SoftClose.ControlAuthority.AuthorityEpoch++
	withHistory.SoftClose.ControlAuthority.Version++
	withHistory.SoftClose.ControlAuthority.ExpiresAt = now.Add(time.Hour)
	if err := repository.Commit(context.Background(), Mutation{Before: state, After: withHistory}); err != nil {
		t.Fatal(err)
	}

	mutated := cloneControlState(withHistory)
	mutated.SoftClose.ControlAuthority.OwnershipHistory[0].Reason = "rewritten"
	if err := repository.Commit(context.Background(), Mutation{Before: withHistory, After: mutated}); !errors.Is(err, ErrFiscalPeriodPolicyConflict) {
		t.Fatalf("ownership history mutation error = %v", err)
	}

	truncated := cloneControlState(withHistory)
	truncated.SoftClose.ControlAuthority.OwnershipHistory = nil
	if err := repository.Commit(context.Background(), Mutation{Before: withHistory, After: truncated}); !errors.Is(err, ErrFiscalPeriodPolicyConflict) {
		t.Fatalf("ownership history truncation error = %v", err)
	}
}

func TestRecoveryApprovalAndDependencyFailuresRemainRetryable(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state, _ := hardCloseFixture()
	state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
	gate := &hardCloseGateStub{err: errors.New("GL status unavailable")}
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	service.BindControlRecoveryDecisionPort(recoveryDecisionPortStub{err: ErrControlRecoveryDecisionUnavailable})
	command := takeoverCommand(state, ControlProcessSoftClose, now, testActor())
	if _, err := service.TakeOverPeriodControl(context.Background(), testActor(), command); !errors.Is(err, ErrPostingBarrierUnavailable) {
		t.Fatalf("dependency error = %v", err)
	}
	if len(gate.calls) != 0 {
		t.Fatalf("dependency failure called a GL mutation: %#v", gate.calls)
	}
}

func TestTakeoverRejectsInvalidRecoveryApprovalEvidence(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	actor := testActor()
	cases := []struct {
		name    string
		mutate  func(*TakeOverPeriodControlCommand)
		portErr error
		wantErr error
	}{
		{name: "self approval", mutate: func(command *TakeOverPeriodControlCommand) { command.Approval.ApproverUserID = actor.UserID }, wantErr: ErrControlRecoveryDecisionRejected},
		{name: "rejected outcome", mutate: func(command *TakeOverPeriodControlCommand) { command.Approval.Outcome = CloseApprovalRejected }, wantErr: ErrControlRecoveryDecisionRejected},
		{name: "stale subject", mutate: func(command *TakeOverPeriodControlCommand) { command.Approval.SubjectVersion++ }, wantErr: ErrControlRecoveryDecisionRejected},
		{name: "changed candidate", mutate: func(command *TakeOverPeriodControlCommand) { command.Approval.CandidateFingerprint = "sha256:changed" }, wantErr: ErrControlRecoveryDecisionRejected},
		{name: "workflow unavailable", portErr: ErrControlRecoveryDecisionUnavailable, wantErr: ErrControlRecoveryDecisionUnavailable},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			state, _ := hardCloseFixture()
			state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
			repository := NewMemoryRepository()
			if err := repository.Seed(state); err != nil {
				t.Fatal(err)
			}
			gate := &hardCloseGateStub{result: recoverySoftGate(state, 2)}
			service, err := NewService(repository, AllowAllAuthorizer{}, gate, func() time.Time { return now })
			if err != nil {
				t.Fatal(err)
			}
			if test.portErr != nil {
				if err := service.BindControlRecoveryDecisionPort(recoveryDecisionPortStub{err: test.portErr}); err != nil {
					t.Fatal(err)
				}
			}
			command := takeoverCommand(state, ControlProcessSoftClose, now, actor)
			if test.mutate != nil {
				test.mutate(&command)
			}
			if _, err := service.TakeOverPeriodControl(context.Background(), actor, command); !errors.Is(err, test.wantErr) {
				t.Fatalf("approval error = %v, want %v", err, test.wantErr)
			}
			if len(repository.Events()) != 0 || len(repository.Audits()) != 0 {
				t.Fatalf("invalid approval established evidence: events=%d audits=%d", len(repository.Events()), len(repository.Audits()))
			}
		})
	}
}

func TestConcurrentRecoveryCommandsHaveOneWinner(t *testing.T) {
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)

	t.Run("takeover", func(t *testing.T) {
		state, _ := hardCloseFixture()
		state.SoftClose.ControlAuthority = recoveryAuthorityFixture(now, Actor{UserID: uuid.New(), SubjectReference: "prior-owner"})
		repository := NewMemoryRepository()
		if err := repository.Seed(state); err != nil {
			t.Fatal(err)
		}
		serviceA, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{result: recoverySoftGate(state, 2)}, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		serviceB, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{result: recoverySoftGate(state, 2)}, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		actorA, actorB := testActor(), testActor()
		commandA := takeoverCommand(state, ControlProcessSoftClose, now, actorA)
		commandB := takeoverCommand(state, ControlProcessSoftClose, now, actorB)
		commandB.Reason = "Recover the interrupted owner through the secondary recovery path"
		commandB.IdempotencyKey = "takeover-concurrent-b"
		commandB.Approval = recoveryApproval(takeoverCandidateFingerprint(commandB), commandB.ExpectedProcessVersion, Actor{UserID: uuid.New(), SubjectReference: "approver-b"}, now)

		errorsCh := make(chan error, 2)
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			_, err := serviceA.TakeOverPeriodControl(context.Background(), actorA, commandA)
			errorsCh <- err
		}()
		go func() {
			defer wait.Done()
			_, err := serviceB.TakeOverPeriodControl(context.Background(), actorB, commandB)
			errorsCh <- err
		}()
		wait.Wait()
		close(errorsCh)
		var winners, conflicts int
		for err := range errorsCh {
			if err == nil {
				winners++
			} else if errors.Is(err, ErrFiscalPeriodVersionConflict) {
				conflicts++
			} else {
				t.Fatalf("concurrent takeover error = %v", err)
			}
		}
		if winners != 1 || conflicts != 1 || len(repository.Events()) != 1 {
			t.Fatalf("concurrent takeover outcome winners=%d conflicts=%d events=%d", winners, conflicts, len(repository.Events()))
		}
	})

	t.Run("extension", func(t *testing.T) {
		state, exceptionID := extensionFixture(now, CloseExceptionUnresolved)
		repository := NewMemoryRepository()
		if err := repository.Seed(state); err != nil {
			t.Fatal(err)
		}
		serviceA, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		serviceB, err := NewService(repository, AllowAllAuthorizer{}, &hardCloseGateStub{}, func() time.Time { return now })
		if err != nil {
			t.Fatal(err)
		}
		actorA, actorB := testActor(), testActor()
		commandA := extensionCommand(state, exceptionID, now)
		commandB := extensionCommand(state, exceptionID, now)
		commandB.Justification = "A second recovery operator documented the same blocker window."
		commandB.IdempotencyKey = "extend-exception-concurrent-b"
		commandB.Approval = recoveryApproval(extensionCandidateFingerprint(commandB), commandB.ExpectedCloseRunVersion, Actor{UserID: uuid.New(), SubjectReference: "approver-b"}, now)

		errorsCh := make(chan error, 2)
		var wait sync.WaitGroup
		wait.Add(2)
		go func() {
			defer wait.Done()
			_, err := serviceA.ExtendCloseException(context.Background(), actorA, commandA)
			errorsCh <- err
		}()
		go func() {
			defer wait.Done()
			_, err := serviceB.ExtendCloseException(context.Background(), actorB, commandB)
			errorsCh <- err
		}()
		wait.Wait()
		close(errorsCh)
		var winners, conflicts int
		for err := range errorsCh {
			if err == nil {
				winners++
			} else if errors.Is(err, ErrFiscalPeriodVersionConflict) {
				conflicts++
			} else {
				t.Fatalf("concurrent extension error = %v", err)
			}
		}
		current, err := repository.Get(context.Background(), state.Period.AccountingScopeID, state.Period.ID)
		if err != nil {
			t.Fatal(err)
		}
		if winners != 1 || conflicts != 1 || current.CloseRun == nil || len(current.CloseRun.Exceptions[0].ExtensionHistory) != 1 {
			t.Fatalf("concurrent extension outcome winners=%d conflicts=%d history=%d", winners, conflicts, len(current.CloseRun.Exceptions[0].ExtensionHistory))
		}
	})
}
