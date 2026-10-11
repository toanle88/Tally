package gl

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestSoftCloseGateEnterAndExitPreserveBoundaryEvidence(t *testing.T) {
	service, repository, scopeID, periodID, runID, actor := newSoftCloseGateTestService(t)
	fixedNow := time.Date(2026, 10, 11, 7, 30, 0, 0, time.UTC)
	service.clock = func() time.Time { return fixedNow }

	enter, err := service.EnterSoftCloseGate(context.Background(), actor, EnterSoftCloseGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 1,
		SoftCloseRunID: runID, ControlEpoch: 7, PolicyVersion: "soft-close-v3",
		Restrictions:   json.RawMessage(`{"ordinaryPosting":"evaluator-controlled","closePosting":false}`),
		IdempotencyKey: "soft-close-enter-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if enter.Gate.GateMode != PostingGateSoftClosePolicy || enter.Gate.GateVersion != 2 || enter.Gate.AggregateVersion != 2 {
		t.Fatalf("entry gate = %#v", enter.Gate)
	}
	if enter.Gate.ActiveSoftCloseRunID == nil || *enter.Gate.ActiveSoftCloseRunID != runID || enter.Gate.ActiveSoftCloseControlEpoch != 7 {
		t.Fatalf("entry owner = %#v", enter.Gate)
	}
	if enter.Gate.ActiveAdmissionCounters.ControlOwnerType != SoftCloseGateOwnerType || enter.Gate.ActiveAdmissionCounters.ControlOwnerID != runID || enter.Gate.ActiveAdmissionCounters.ControlOwnerEpoch != 7 {
		t.Fatalf("entry counters = %#v", enter.Gate.ActiveAdmissionCounters)
	}
	if string(enter.Gate.SoftCloseRestrictions) != `{"closePosting":false,"ordinaryPosting":"evaluator-controlled"}` {
		t.Fatalf("canonical restrictions = %s", enter.Gate.SoftCloseRestrictions)
	}

	exit, err := service.ExitSoftCloseGate(context.Background(), actor, ExitSoftCloseGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2,
		SoftCloseRunID: runID, ControlEpoch: 7, IdempotencyKey: "soft-close-exit-1",
		CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if exit.Gate.GateMode != PostingGateOpen || exit.Gate.GateVersion != 3 || exit.Gate.AggregateVersion != 3 {
		t.Fatalf("exit gate = %#v", exit.Gate)
	}
	if exit.Gate.ActiveSoftCloseRunID != nil || exit.Gate.ActiveSoftCloseControlEpoch != 0 || exit.Gate.SoftClosePolicyVersion != "" || string(exit.Gate.SoftCloseRestrictions) != `{}` {
		t.Fatalf("cleared active owner = %#v", exit.Gate)
	}
	if exit.Gate.LastFrozenAdmissionSummary == nil || exit.Gate.LastFrozenAdmissionSummary.FrozenAtGateVersion != 3 || len(exit.Gate.FrozenAdmissionSummaryHistory) != 1 {
		t.Fatalf("frozen summary = %#v", exit.Gate)
	}
	if exit.Gate.LastFrozenAdmissionSummary.ControlOwnerID != runID || exit.Gate.LastFrozenAdmissionSummary.ControlOwnerEpoch != 7 {
		t.Fatalf("frozen owner = %#v", exit.Gate.LastFrozenAdmissionSummary)
	}

	events := repository.Events()
	if len(events) != 2 || events[0].EventType != SoftCloseGateEnteredEventType || events[1].EventType != SoftCloseGateExitedEventType {
		t.Fatalf("events = %#v", events)
	}
}

func TestSoftCloseGateIdempotencyAndTypedConflicts(t *testing.T) {
	service, repository, scopeID, periodID, runID, actor := newSoftCloseGateTestService(t)
	command := EnterSoftCloseGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 1,
		SoftCloseRunID: runID, ControlEpoch: 1, PolicyVersion: "policy-v1", Restrictions: json.RawMessage(`{"ordinaryPosting":"review"}`),
		IdempotencyKey: "same-enter", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	first, err := service.EnterSoftCloseGate(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := service.EnterSoftCloseGate(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.Gate.GateVersion != first.Gate.GateVersion || len(repository.Events()) != 1 {
		t.Fatalf("replay = %#v, events = %#v", replay, repository.Events())
	}

	changed := command
	changed.PolicyVersion = "policy-v2"
	if _, err := service.EnterSoftCloseGate(context.Background(), actor, changed); !errors.Is(err, ErrSoftCloseGateIdempotencyConflict) {
		t.Fatalf("changed retry error = %v", err)
	}
	policyConflict := command
	policyConflict.ExpectedGateVersion = 2
	policyConflict.PolicyVersion = "policy-v2"
	policyConflict.IdempotencyKey = "different-enter-policy"
	if _, err := service.EnterSoftCloseGate(context.Background(), actor, policyConflict); !errors.Is(err, ErrSoftCloseGatePolicyConflict) {
		t.Fatalf("changed active policy error = %v, want policy conflict", err)
	}

	wrongOwner := ExitSoftCloseGateCommand{
		CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 2,
		SoftCloseRunID: uuid.New(), ControlEpoch: 1, IdempotencyKey: "wrong-owner", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}
	if _, err := service.ExitSoftCloseGate(context.Background(), actor, wrongOwner); !errors.Is(err, ErrSoftCloseGateOwnerConflict) {
		t.Fatalf("wrong owner error = %v", err)
	}

	wrongEpoch := wrongOwner
	wrongEpoch.SoftCloseRunID = runID
	wrongEpoch.ControlEpoch = 9
	wrongEpoch.IdempotencyKey = "wrong-epoch"
	if _, err := service.ExitSoftCloseGate(context.Background(), actor, wrongEpoch); !errors.Is(err, ErrSoftCloseGateEpochConflict) {
		t.Fatalf("wrong epoch error = %v", err)
	}
}

func TestSoftCloseGateAuditAndOutboxFailuresDoNotMutateGate(t *testing.T) {
	audit := &MemorySoftCloseGateAuditRecorder{Err: ErrSoftCloseGateAuditUnavailable}
	scopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New()
	repository := NewMemorySoftCloseGateRepository(audit)
	if err := repository.Seed(NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, MemorySoftCloseGateAuthorizer{Decision: AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject"}
	command := EnterSoftCloseGateCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, ExpectedGateVersion: 1, SoftCloseRunID: runID, ControlEpoch: 1, PolicyVersion: "policy-v1", Restrictions: json.RawMessage(`{}`), IdempotencyKey: "audit-failure", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	if _, err := service.EnterSoftCloseGate(context.Background(), actor, command); !errors.Is(err, ErrSoftCloseGateAuditUnavailable) {
		t.Fatalf("audit error = %v", err)
	}
	gate, err := repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil || gate.GateMode != PostingGateOpen || gate.GateVersion != 1 {
		t.Fatalf("gate after audit failure = %#v, err = %v", gate, err)
	}

	audit.Err = nil
	repository.SetOutboxError(ErrSoftCloseGateOutboxUnavailable)
	command.IdempotencyKey = "outbox-failure"
	if _, err := service.EnterSoftCloseGate(context.Background(), actor, command); !errors.Is(err, ErrSoftCloseGateOutboxUnavailable) {
		t.Fatalf("outbox error = %v", err)
	}
	gate, err = repository.GetSoftCloseGate(context.Background(), scopeID, periodID)
	if err != nil || gate.GateMode != PostingGateOpen || gate.GateVersion != 1 || len(repository.Events()) != 0 {
		t.Fatalf("gate after outbox failure = %#v, events = %#v, err = %v", gate, repository.Events(), err)
	}
	if len(audit.Records) != 0 {
		t.Fatalf("audit records = %#v, want no record when outbox is unavailable", audit.Records)
	}
}

func newSoftCloseGateTestService(t *testing.T) (*SoftCloseGateService, *MemorySoftCloseGateRepository, uuid.UUID, uuid.UUID, uuid.UUID, Actor) {
	t.Helper()
	scopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New()
	repository := NewMemorySoftCloseGateRepository()
	if err := repository.Seed(NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := NewSoftCloseGateService(repository, MemorySoftCloseGateAuthorizer{Decision: AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "gl-policy", PolicyVersion: "gl-policy-v1"}}, AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, scopeID, periodID, runID, Actor{UserID: uuid.New(), SubjectReference: "subject"}
}
