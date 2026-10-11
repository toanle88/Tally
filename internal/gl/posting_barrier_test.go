package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestPostingBarrierAcquireAndReleaseRestoresNextSoftCloseEpoch(t *testing.T) {
	service, repository, scopeID, periodID, softCloseRunID, actor := newSoftCloseGateTestService(t)
	enterSoftCloseForBarrierTest(t, service, actor, scopeID, periodID, softCloseRunID, 1, 4)
	closeRunID := uuid.New()

	acquired, err := service.AcquirePostingBarrier(context.Background(), actor, AcquirePostingBarrierCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2,
		SoftCloseRunID: softCloseRunID, SoftCloseControlEpoch: 4, SoftClosePolicyVersion: "policy-v1",
		CloseRunID: closeRunID, CloseType: CloseTypeInitialHardClose, IdempotencyKey: "barrier-acquire-1",
		CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if acquired.Gate.GateMode != PostingGateCloseOnly || acquired.Gate.GateVersion != 3 || acquired.Gate.ActiveCloseRunID == nil || *acquired.Gate.ActiveCloseRunID != closeRunID {
		t.Fatalf("acquired gate = %#v", acquired.Gate)
	}
	if acquired.Gate.ActiveSoftCloseRunID != nil || acquired.Gate.PriorControlProcessID == nil || *acquired.Gate.PriorControlProcessID != softCloseRunID || acquired.Gate.PriorControlEpoch == nil || *acquired.Gate.PriorControlEpoch != 4 || acquired.Gate.BarrierLedgerPosition != 0 {
		t.Fatalf("acquisition evidence = %#v", acquired.Gate)
	}
	if acquired.Gate.ActiveAdmissionCounters.ControlOwnerType != PostingBarrierOwnerType || acquired.Gate.ActiveAdmissionCounters.ControlOwnerID != closeRunID || acquired.Gate.ActiveAdmissionCounters.ControlOwnerEpoch != nil {
		t.Fatalf("close counters = %#v", acquired.Gate.ActiveAdmissionCounters)
	}

	released, err := service.ReleasePostingBarrier(context.Background(), actor, ReleasePostingBarrierCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 3,
		CloseRunID: closeRunID, IdempotencyKey: "barrier-release-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if released.Gate.GateMode != PostingGateSoftClosePolicy || released.Gate.GateVersion != 4 || released.Gate.ActiveSoftCloseRunID == nil || *released.Gate.ActiveSoftCloseRunID != softCloseRunID || released.Gate.ActiveSoftCloseControlEpoch != 5 {
		t.Fatalf("released gate = %#v", released.Gate)
	}
	if released.Gate.LastFrozenAdmissionSummary == nil || released.Gate.LastFrozenAdmissionSummary.ControlOwnerID != closeRunID || released.Gate.LastFrozenAdmissionSummary.ControlOwnerEpoch != nil || len(released.Gate.FrozenAdmissionSummaryHistory) != 2 {
		t.Fatalf("release summary = %#v", released.Gate)
	}
	if released.Gate.ActiveAdmissionCounters.ControlOwnerEpoch == nil || *released.Gate.ActiveAdmissionCounters.ControlOwnerEpoch != 5 {
		t.Fatalf("restored counters = %#v", released.Gate.ActiveAdmissionCounters)
	}
	status, err := service.GetPostingGateStatus(context.Background(), scopeID, periodID)
	if err != nil || status.GateVersion != 4 {
		t.Fatalf("status = %#v, err = %v", status, err)
	}
	if events := repository.Events(); len(events) != 3 || events[1].EventType != PostingBarrierAcquiredEventType || events[2].EventType != PostingBarrierReleasedEventType {
		t.Fatalf("events = %#v", events)
	}
}

func TestPostingBarrierFinalizationRetainsWatermarkAndReplays(t *testing.T) {
	service, repository, scopeID, periodID, softCloseRunID, actor := newSoftCloseGateTestService(t)
	enterSoftCloseForBarrierTest(t, service, actor, scopeID, periodID, softCloseRunID, 1, 1)
	closeRunID := uuid.New()
	_, err := service.AcquirePostingBarrier(context.Background(), actor, AcquirePostingBarrierCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2,
		SoftCloseRunID: softCloseRunID, SoftCloseControlEpoch: 1, SoftClosePolicyVersion: "policy-v1", CloseRunID: closeRunID,
		CloseType: CloseTypeInitialHardClose, IdempotencyKey: "barrier-acquire-finalize", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	command := FinalizePostingGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 3, CloseRunID: closeRunID, FinalLedgerWatermark: 12, IdempotencyKey: "barrier-finalize-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	finalized, err := service.FinalizePostingGate(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if finalized.Gate.GateMode != PostingGateHardClosed || finalized.Gate.GateVersion != 4 || finalized.Gate.ActiveCloseRunID != nil || finalized.Gate.FinalizedCloseRunID == nil || *finalized.Gate.FinalizedCloseRunID != closeRunID || finalized.Gate.FinalLedgerWatermark == nil || *finalized.Gate.FinalLedgerWatermark != 12 || finalized.Gate.ActiveAdmissionCounters.ControlOwnerType != "" {
		t.Fatalf("finalized gate = %#v", finalized.Gate)
	}
	if finalized.Gate.LastFrozenAdmissionSummary == nil || finalized.Gate.LastFrozenAdmissionSummary.FrozenByCommandType != PostingBarrierFinalizeCommandIdentity {
		t.Fatalf("finalized summary = %#v", finalized.Gate.LastFrozenAdmissionSummary)
	}
	replay, err := service.FinalizePostingGate(context.Background(), actor, command)
	if err != nil || !replay.Replayed || replay.Gate.FinalizationCommandFingerprint != finalized.Gate.FinalizationCommandFingerprint {
		t.Fatalf("replay = %#v, err = %v", replay, err)
	}
	if _, err := service.ReleasePostingBarrier(context.Background(), actor, ReleasePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 4, CloseRunID: closeRunID, IdempotencyKey: "release-finalized", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); !errors.Is(err, ErrPostingBarrierNotReleasable) {
		t.Fatalf("release after finalization error = %v", err)
	}
	if events := repository.Events(); len(events) != 3 || events[2].EventType != PostingGateFinalizedEventType {
		t.Fatalf("events = %#v", events)
	}
}

func TestPostingBarrierRejectsStaleEpochAndAdmittedRelease(t *testing.T) {
	service, repository, scopeID, periodID, softCloseRunID, actor := newSoftCloseGateTestService(t)
	enterSoftCloseForBarrierTest(t, service, actor, scopeID, periodID, softCloseRunID, 1, 2)
	before, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	closeRunID := uuid.New()
	_, err = service.AcquirePostingBarrier(context.Background(), actor, AcquirePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2, SoftCloseRunID: softCloseRunID, SoftCloseControlEpoch: 9, SoftClosePolicyVersion: "policy-v1", CloseRunID: closeRunID, CloseType: CloseTypeInitialHardClose, IdempotencyKey: "stale-acquire", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if !errors.Is(err, ErrSoftCloseGateEpochConflict) {
		t.Fatalf("stale epoch error = %v", err)
	}
	after, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil || after.GateVersion != before.GateVersion || after.ActiveSoftCloseRunID == nil || *after.ActiveSoftCloseRunID != softCloseRunID {
		t.Fatalf("gate after stale acquire = %#v, err = %v", after, err)
	}

	_, err = service.AcquirePostingBarrier(context.Background(), actor, AcquirePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2, SoftCloseRunID: softCloseRunID, SoftCloseControlEpoch: 2, SoftClosePolicyVersion: "policy-v1", CloseRunID: closeRunID, CloseType: CloseTypeInitialHardClose, IdempotencyKey: "valid-acquire", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()})
	if err != nil {
		t.Fatal(err)
	}
	gate, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	gate.ActiveAdmissionCounters.ClosePostingAdmitted = true
	gate.ActiveAdmissionCounters.AdmittedPostingCount = 1
	gate.ActiveAdmissionCounters.FirstAdmittedLedgerPosition = 1
	gate.ActiveAdmissionCounters.LastAdmittedLedgerPosition = 1
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReleasePostingBarrier(context.Background(), actor, ReleasePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 3, CloseRunID: closeRunID, IdempotencyKey: "blocked-release", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); !errors.Is(err, ErrPostingBarrierAdmissionConflict) {
		t.Fatalf("admitted release error = %v", err)
	}
}

func TestPostingBarrierRejectsRecloseReleaseAndInvalidWatermark(t *testing.T) {
	scopeID, periodID, closeRunID := uuid.New(), uuid.New(), uuid.New()
	repository := NewMemorySoftCloseGateRepository()
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 1)
	gate.GateMode = PostingGateCloseOnly
	gate.GateVersion = 2
	gate.AggregateVersion = 2
	gate.ActiveCloseRunID = &closeRunID
	gate.ActiveCloseType = CloseTypeReclose
	gate.BarrierLedgerPosition = 8
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: PostingBarrierOwnerType, ControlOwnerID: closeRunID}
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, MemorySoftCloseGateAuthorizer{Decision: AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	if _, err := service.ReleasePostingBarrier(context.Background(), actor, ReleasePostingBarrierCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2, CloseRunID: closeRunID, IdempotencyKey: "reclose-release", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); !errors.Is(err, ErrPostingBarrierNotReleasable) {
		t.Fatalf("reclose release error = %v", err)
	}
	if _, err := service.FinalizePostingGate(context.Background(), actor, FinalizePostingGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2, CloseRunID: closeRunID, FinalLedgerWatermark: 7, IdempotencyKey: "low-watermark", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); !errors.Is(err, ErrPostingBarrierWatermarkConflict) {
		t.Fatalf("low watermark error = %v", err)
	}
}

func enterSoftCloseForBarrierTest(t *testing.T, service *SoftCloseGateService, actor Actor, scopeID, periodID, runID uuid.UUID, expectedVersion, epoch int64) {
	t.Helper()
	if _, err := service.EnterSoftCloseGate(context.Background(), actor, EnterSoftCloseGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: expectedVersion, SoftCloseRunID: runID, ControlEpoch: epoch, PolicyVersion: "policy-v1", Restrictions: []byte(`{"ordinaryPosting":"evaluator-controlled"}`), IdempotencyKey: "soft-close-for-barrier", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}); err != nil {
		t.Fatal(err)
	}
}
