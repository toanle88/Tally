package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestOperationalReopenGateRequiresActorAndClassAndTransfersPositiveSummaryToReclose(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID, recloseRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "operations-controller"}
	repository := NewMemorySoftCloseGateRepository()
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
	gate.GateMode = PostingGateHardClosed
	gate.GateVersion = 3
	gate.AggregateVersion = 3
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, allowScopedReopenForTest(scopeID), AllowAllSoftClosePolicyEvaluator{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	open := validOpenOperationalReopenCommand(scopeID, periodID, requestID, actor.UserID, 3, now.Add(time.Hour))
	opened, err := service.OpenOperationalReopenGate(context.Background(), actor, open)
	if err != nil {
		t.Fatal(err)
	}
	replayedOpen, err := service.OpenOperationalReopenGate(context.Background(), actor, open)
	if err != nil || !replayedOpen.Replayed {
		t.Fatalf("replayed open = %#v, err = %v", replayedOpen, err)
	}
	conflictingOpen := open
	conflictingOpen.AuthorizationScope.TransactionClasses = []string{"different-recovery-class"}
	if _, err := service.OpenOperationalReopenGate(context.Background(), actor, conflictingOpen); !errors.Is(err, ErrSoftCloseGateIdempotencyConflict) {
		t.Fatalf("changed open fingerprint error = %v", err)
	}
	if opened.Gate.GateMode != PostingGateOperationalReopen || opened.Gate.ActiveReopenRequestID == nil || *opened.Gate.ActiveReopenRequestID != requestID {
		t.Fatalf("opened gate = %#v", opened.Gate)
	}
	if opened.Gate.ActiveAdmissionCounters.ControlOwnerType != OperationalReopenGateOwnerType || opened.Gate.ActiveAdmissionCounters.ControlOwnerID != requestID || opened.Gate.ActiveAdmissionCounters.ControlOwnerEpoch == nil || *opened.Gate.ActiveAdmissionCounters.ControlOwnerEpoch != 9 || opened.Gate.ActiveAdmissionCounters.AdmittedPostingCount != 0 {
		t.Fatalf("opened counters = %#v", opened.Gate.ActiveAdmissionCounters)
	}

	posting := PostingRequest{
		FiscalPeriodID: periodID, PeriodStateVersion: 4, PostingGateVersion: 4,
		PostingPurpose: PostingPurposeOperationalReopen, SourceAggregateType: "operational-recovery",
		OperationalReopenRequestID: &requestID, ControlAuthorityEpoch: int64Pointer(9),
	}
	evidence := PostingGateEvidence{
		FiscalPeriodID: periodID, PeriodStateVersion: 4, PostingGateVersion: 4, GateMode: PostingGateOperationalReopen,
		ReopenRequestID: &requestID, AuthorizationScope: opened.Gate.ReopenAuthorizationScope,
		AuthorizationExpiry: opened.Gate.ReopenAuthorizationExpiry, ControlAuthorityEpoch: opened.Gate.ReopenControlAuthorityEpoch,
	}
	if err := validatePostingGateEvidenceForActorAt(posting, evidence, actor, now); err != nil {
		t.Fatalf("in-scope operational posting evidence rejected: %v", err)
	}
	wrongActor := actor
	wrongActor.UserID = uuid.New()
	if err := validatePostingGateEvidenceForActorAt(posting, evidence, wrongActor, now); !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("wrong actor error = %v", err)
	}
	wrongClass := posting
	wrongClass.SourceAggregateType = "manual-journal"
	if err := validatePostingGateEvidenceForActorAt(wrongClass, evidence, actor, now); !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("wrong class error = %v", err)
	}

	active, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	active.ActiveAdmissionCounters.AdmittedPostingCount = 1
	active.ActiveAdmissionCounters.ReopenPostingAdmitted = true
	active.ActiveAdmissionCounters.FirstAdmittedLedgerPosition = 4
	active.ActiveAdmissionCounters.LastAdmittedLedgerPosition = 4
	active.NextLedgerPosition = 5
	if err := repository.Seed(active); err != nil {
		t.Fatal(err)
	}
	closed, err := service.CloseOperationalReopenGate(context.Background(), actor, CloseOperationalReopenGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 4,
		ReopenRequestID: requestID, IdempotencyKey: "close-operational", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Gate.GateMode != PostingGateCloseOnly || closed.Gate.ActiveReopenRequestID == nil || *closed.Gate.ActiveReopenRequestID != requestID || closed.Gate.LastFrozenAdmissionSummary == nil || closed.Gate.LastFrozenAdmissionSummary.AdmittedPostingCount != 1 {
		t.Fatalf("closed gate = %#v", closed.Gate)
	}
	priorSummary := *closed.Gate.LastFrozenAdmissionSummary

	beginCommand := BeginRecloseGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 5,
		ReopenRequestID: requestID, RecloseRunID: recloseRunID, IdempotencyKey: "begin-reclose", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	begun, err := service.BeginRecloseGate(context.Background(), actor, beginCommand)
	if err != nil {
		t.Fatal(err)
	}
	replayedBegin, err := service.BeginRecloseGate(context.Background(), actor, beginCommand)
	if err != nil || !replayedBegin.Replayed {
		t.Fatalf("replayed begin-reclose = %#v, err = %v", replayedBegin, err)
	}
	conflictingBegin := beginCommand
	conflictingBegin.RecloseRunID = uuid.New()
	if _, err := service.BeginRecloseGate(context.Background(), actor, conflictingBegin); !errors.Is(err, ErrSoftCloseGateIdempotencyConflict) {
		t.Fatalf("changed begin-reclose fingerprint error = %v", err)
	}
	if begun.Gate.GateMode != PostingGateCloseOnly || begun.Gate.ActiveCloseRunID == nil || *begun.Gate.ActiveCloseRunID != recloseRunID || begun.Gate.ActiveCloseType != CloseTypeReclose || begun.Gate.ActiveReopenRequestID != nil {
		t.Fatalf("begun gate = %#v", begun.Gate)
	}
	if begun.Gate.ActiveAdmissionCounters.ControlOwnerType != PostingBarrierOwnerType || begun.Gate.ActiveAdmissionCounters.ControlOwnerID != recloseRunID || begun.Gate.ActiveAdmissionCounters.AdmittedPostingCount != 0 {
		t.Fatalf("reclose counters = %#v", begun.Gate.ActiveAdmissionCounters)
	}
	if begun.Gate.BarrierLedgerPosition != 4 || begun.Gate.LastFrozenAdmissionSummary == nil || begun.Gate.LastFrozenAdmissionSummary.FrozenByCommandFingerprint != priorSummary.FrozenByCommandFingerprint {
		t.Fatalf("reclose handoff summary = %#v", begun.Gate.LastFrozenAdmissionSummary)
	}
	if _, err := service.ReleasePostingBarrier(context.Background(), actor, ReleasePostingBarrierCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 6,
		CloseRunID: recloseRunID, IdempotencyKey: "release-reclose", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}); !errors.Is(err, ErrPostingBarrierNotReleasable) {
		t.Fatalf("reclose release error = %v", err)
	}
	if events := repository.Events(); len(events) != 3 || events[0].EventType != OperationalReopenGateOpenedEventType || events[1].EventType != OperationalReopenGateClosedEventType || events[2].EventType != RecloseGateBegunEventType {
		t.Fatalf("events = %#v", events)
	}
}

func TestOperationalReopenGateNoChangeAndExpiryEvent(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	currentTime := now
	scopeID, periodID, requestID := uuid.New(), uuid.New(), uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "operations-controller"}
	repository := NewMemorySoftCloseGateRepository()
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
	gate.GateMode = PostingGateHardClosed
	gate.GateVersion = 7
	gate.AggregateVersion = 7
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, allowScopedReopenForTest(scopeID), AllowAllSoftClosePolicyEvaluator{}, func() time.Time { return currentTime })
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.OpenOperationalReopenGate(context.Background(), actor, validOpenOperationalReopenCommand(scopeID, periodID, requestID, actor.UserID, 7, now.Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	currentTime = now.Add(2 * time.Hour)
	posting := PostingRequest{
		FiscalPeriodID: periodID, PeriodStateVersion: 4, PostingGateVersion: 8,
		PostingPurpose: PostingPurposeOperationalReopen, SourceAggregateType: "operational-recovery",
		OperationalReopenRequestID: &requestID, ControlAuthorityEpoch: int64Pointer(9),
	}
	evidence := PostingGateEvidence{
		FiscalPeriodID: periodID, PeriodStateVersion: 4, PostingGateVersion: 8, GateMode: PostingGateOperationalReopen,
		ReopenRequestID: &requestID, AuthorizationScope: opened.Gate.ReopenAuthorizationScope,
		AuthorizationExpiry: opened.Gate.ReopenAuthorizationExpiry, ControlAuthorityEpoch: opened.Gate.ReopenControlAuthorityEpoch,
	}
	if err := validatePostingGateEvidenceForActorAt(posting, evidence, actor, currentTime); !errors.Is(err, ErrPostingGateClosed) {
		t.Fatalf("expired operational posting error = %v", err)
	}
	closed, err := service.CloseOperationalReopenGate(context.Background(), actor, CloseOperationalReopenGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 8,
		ReopenRequestID: requestID, IdempotencyKey: "close-expired", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if closed.Gate.GateMode != PostingGateHardClosed || closed.Gate.ActiveReopenRequestID != nil || closed.Gate.LastFrozenAdmissionSummary == nil || closed.Gate.LastFrozenAdmissionSummary.AdmittedPostingCount != 0 {
		t.Fatalf("expired no-change close = %#v", closed.Gate)
	}
	if events := repository.Events(); len(events) != 3 || events[1].EventType != OperationalReopenGateExpiredEventType || events[2].EventType != OperationalReopenGateClosedEventType {
		t.Fatalf("expiry events = %#v", events)
	}
}

func TestScopedReopenGateBeginReclosePreservesScopedPriorOwner(t *testing.T) {
	now := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	scopeID, periodID, requestID, recloseRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	epoch := int64(9)
	repository := NewMemorySoftCloseGateRepository()
	gate := NewOpenMemorySoftCloseGate(scopeID, periodID, 4)
	gate.GateMode = PostingGateCloseOnly
	gate.GateVersion = 12
	gate.AggregateVersion = 12
	gate.ActiveReopenRequestID = &requestID
	gate.ReopenControlAuthorityEpoch = &epoch
	gate.ActiveAdmissionCounters = ActiveGateAdmissionCounters{ControlOwnerType: ScopedReopenGateOwnerType, ControlOwnerID: requestID, ControlOwnerEpoch: &epoch, AdmittedPostingCount: 1, ReopenPostingAdmitted: true, FirstAdmittedLedgerPosition: 10, LastAdmittedLedgerPosition: 10}
	gate.LastFrozenAdmissionSummary = &FrozenGateAdmissionSummary{ControlOwnerType: ScopedReopenGateOwnerType, ControlOwnerID: requestID, ControlOwnerEpoch: &epoch, AdmittedPostingCount: 1, ReopenPostingAdmitted: true, FirstAdmittedLedgerPosition: 10, LastAdmittedLedgerPosition: 10, FrozenAtGateVersion: 12, FrozenAt: now, FrozenByCommandFingerprint: "sha256:scoped-close", FrozenByCommandType: ScopedReopenGateCloseCommandIdentity}
	if err := repository.Seed(gate); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, allowScopedReopenForTest(scopeID), AllowAllSoftClosePolicyEvaluator{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.BeginRecloseGate(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "controller"}, BeginRecloseGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 12, ReopenRequestID: requestID, RecloseRunID: recloseRunID,
		IdempotencyKey: "scoped-begin-reclose", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Gate.PriorControlProcessType != ScopedReopenGateOwnerType || result.Gate.PriorGateMode != PostingGateScopedReopen || result.Gate.ActiveCloseType != CloseTypeReclose {
		t.Fatalf("scoped reclose lineage = %#v", result.Gate)
	}
}

func validOpenOperationalReopenCommand(scopeID, periodID, requestID, actorID uuid.UUID, expectedVersion int64, expiry time.Time) OpenOperationalReopenGateCommand {
	return OpenOperationalReopenGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: expectedVersion,
		ReopenRequestID: requestID, AuthorizationScope: OperationalReopenScope{ActorIDs: []uuid.UUID{actorID}, TransactionClasses: []string{"operational-recovery"}},
		AuthorizationExpiry: expiry, ControlAuthorityEpoch: 9,
		Approval:               ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "reopen-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:approved", ApproverUserID: uuid.New()},
		NoChangeClosePermitted: true, IdempotencyKey: "open-operational-reopen", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
}
