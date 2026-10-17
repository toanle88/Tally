package fiscalperiod

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
)

type reopenGateStub struct {
	*hardCloseGateStub
	openResult           gl.SoftCloseGateResult
	openErr              error
	openScopedCalls      int
	openOperationalCalls int
}

func (stub *reopenGateStub) OpenScopedReopenGate(_ context.Context, _ gl.Actor, command gl.OpenScopedReopenGateCommand) (gl.SoftCloseGateResult, error) {
	stub.openScopedCalls++
	return stub.openReopenResult(command.ExpectedGateVersion, command.ReopenRequestID, command.AuthorizationExpiry, command.ControlAuthorityEpoch, gl.PostingGateScopedReopen, gl.ScopedReopenGateOpenOperation)
}

func (stub *reopenGateStub) OpenOperationalReopenGate(_ context.Context, _ gl.Actor, command gl.OpenOperationalReopenGateCommand) (gl.SoftCloseGateResult, error) {
	stub.openOperationalCalls++
	return stub.openReopenResult(command.ExpectedGateVersion, command.ReopenRequestID, command.AuthorizationExpiry, command.ControlAuthorityEpoch, gl.PostingGateOperationalReopen, gl.OperationalReopenGateOpenOperation)
}

func (stub *reopenGateStub) openReopenResult(expectedGateVersion int64, requestID uuid.UUID, expiry time.Time, epoch int64, mode, operation string) (gl.SoftCloseGateResult, error) {
	if stub.openErr != nil {
		return gl.SoftCloseGateResult{}, stub.openErr
	}
	result := stub.openResult
	result.Operation = operation
	result.Gate.GateMode = mode
	result.Gate.GateVersion = expectedGateVersion + 1
	result.Gate.AggregateVersion = expectedGateVersion + 1
	result.Gate.ActiveReopenRequestID = cloneUUID(&requestID)
	result.Gate.ReopenAuthorizationExpiry = cloneTime(&expiry)
	result.Gate.ReopenControlAuthorityEpoch = cloneInt64(&epoch)
	stub.openResult = result
	return result, nil
}

func (stub *reopenGateStub) GetPostingGateStatus(ctx context.Context, scopeID, periodID uuid.UUID) (gl.PostingGateStatus, error) {
	if stub.openScopedCalls > 0 || stub.openOperationalCalls > 0 {
		return stub.openResult.Gate, stub.err
	}
	return stub.hardCloseGateStub.GetPostingGateStatus(ctx, scopeID, periodID)
}

type reopenDecisionStub struct {
	err error
}

func (stub reopenDecisionStub) Validate(context.Context, Actor, ReopenRequest, ReopenApprovalDecision) error {
	return stub.err
}

func TestRequestReopenCreatesPendingApprovalWithoutOpeningGate(t *testing.T) {
	state, command, now := reopenFixture(ReopenModeScopedCorrection)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, command.ExpectedGateVersion)}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}

	result, err := service.RequestReopen(context.Background(), testActor(), command)
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateHardClosed || result.State.ReopenRequest == nil || result.State.ReopenRequest.Status != ReopenRequestPendingApproval {
		t.Fatalf("pending reopen result = %#v", result.State)
	}
	if gate.openScopedCalls != 0 || gate.openOperationalCalls != 0 {
		t.Fatalf("pending request opened GL gate: scoped=%d operational=%d", gate.openScopedCalls, gate.openOperationalCalls)
	}
	if events := repository.Events(); len(events) != 1 || events[0].EventType != "ReopenRequested" {
		t.Fatalf("reopen events = %#v", events)
	}
}

func TestApplyReopenApprovalRejectionRetainsHardClosedPeriod(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeScopedCorrection)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestReopen(context.Background(), testActor(), requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	decision := reopenDecision(requested.State.ReopenRequest, ReopenApprovalRejected, testActor().UserID)
	decision.Reason = "The correction evidence is insufficient."
	result, err := service.ApplyReopenApprovalDecision(context.Background(), testActor(), ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: requested.State.ReopenRequest.ID,
		ExpectedPeriodVersion: requested.State.Period.AggregateVersion, ExpectedReopenRequestVersion: requested.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: requestCommand.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: "apply-reopen-rejection", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateHardClosed || result.State.ReopenRequest.Status != ReopenRequestRejected || result.State.ReopenRequest.ApprovalDecision == nil {
		t.Fatalf("rejected reopen result = %#v", result.State)
	}
	if gate.openScopedCalls != 0 || gate.openOperationalCalls != 0 {
		t.Fatalf("rejected approval opened GL gate: scoped=%d operational=%d", gate.openScopedCalls, gate.openOperationalCalls)
	}

	retry := requestCommand
	retry.CommandID = uuid.New()
	retry.ExpectedPeriodVersion = result.State.Period.AggregateVersion
	retry.IdempotencyKey = "request-reopen-after-rejection"
	retry.CorrelationID = uuid.NewString()
	retry.CausationID = uuid.NewString()
	reopened, err := service.RequestReopen(context.Background(), testActor(), retry)
	if err != nil {
		t.Fatalf("request after rejection: %v", err)
	}
	if reopened.State.Period.State != PeriodStateHardClosed || reopened.State.ReopenRequest.Status != ReopenRequestPendingApproval || reopened.State.ReopenRequest.ID == result.State.ReopenRequest.ID {
		t.Fatalf("request after rejection result = %#v", reopened.State)
	}
}

func TestApplyApprovedScopedReopenOpensOnlyTheScopedGate(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeScopedCorrection)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}, openResult: gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, PeriodStateVersion: state.Period.StateVersion, NextLedgerPosition: 100}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestReopen(context.Background(), testActor(), requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	decision := reopenDecision(requested.State.ReopenRequest, ReopenApprovalApproved, uuid.New())
	result, err := service.ApplyReopenApprovalDecision(context.Background(), testActor(), ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: requested.State.ReopenRequest.ID,
		ExpectedPeriodVersion: requested.State.Period.AggregateVersion, ExpectedReopenRequestVersion: requested.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: requestCommand.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: "apply-scoped-reopen", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.State.Period.State != PeriodStateReopening || result.State.ReopenRequest.Status != ReopenRequestActive || result.State.ReopenRequest.GateVersion != requestCommand.ExpectedGateVersion+1 {
		t.Fatalf("active scoped reopen result = %#v", result.State)
	}
	if gate.openScopedCalls != 1 || gate.openOperationalCalls != 0 {
		t.Fatalf("scoped gate calls = scoped %d, operational %d", gate.openScopedCalls, gate.openOperationalCalls)
	}
	if events := repository.Events(); len(events) != 3 || events[1].EventType != "PeriodStateChanged" || events[2].EventType != "PeriodReopened" {
		t.Fatalf("scoped reopen events = %#v", events)
	}
}

func TestApplyApprovedOperationalReopenUsesOperationalGate(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeOperational)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}, openResult: gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, PeriodStateVersion: state.Period.StateVersion, NextLedgerPosition: 100}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestReopen(context.Background(), testActor(), requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	decision := reopenDecision(requested.State.ReopenRequest, ReopenApprovalApproved, uuid.New())
	if _, err := service.ApplyReopenApprovalDecision(context.Background(), testActor(), ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: requested.State.ReopenRequest.ID,
		ExpectedPeriodVersion: requested.State.Period.AggregateVersion, ExpectedReopenRequestVersion: requested.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: requestCommand.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: "apply-operational-reopen", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	}); err != nil {
		t.Fatal(err)
	}
	if gate.openScopedCalls != 0 || gate.openOperationalCalls != 1 {
		t.Fatalf("operational gate calls = scoped %d, operational %d", gate.openScopedCalls, gate.openOperationalCalls)
	}
}

func TestRejectedGateOpeningRestoresApprovedRetryState(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeScopedCorrection)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}, openErr: gl.ErrSoftCloseGateVersionConflict}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestReopen(context.Background(), testActor(), requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	decision := reopenDecision(requested.State.ReopenRequest, ReopenApprovalApproved, uuid.New())
	_, err = service.ApplyReopenApprovalDecision(context.Background(), testActor(), ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: requested.State.ReopenRequest.ID,
		ExpectedPeriodVersion: requested.State.Period.AggregateVersion, ExpectedReopenRequestVersion: requested.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: requestCommand.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: "apply-reopen-gate-rejection", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if !errors.Is(err, ErrReopenGateRejected) {
		t.Fatalf("gate rejection = %v", err)
	}
	stored, err := repository.Get(context.Background(), requestCommand.AccountingScopeID, requestCommand.FiscalPeriodID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Period.State != PeriodStateHardClosed || stored.ReopenRequest.Status != ReopenRequestApproved {
		t.Fatalf("retryable gate rejection state = %#v", stored)
	}
}

func TestReopenRejectsSelfApprovalAndChangedIdempotency(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeScopedCorrection)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := testActor()
	requested, err := service.RequestReopen(context.Background(), actor, requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	changed := requestCommand
	changed.Reason = "changed reason"
	if _, err := service.RequestReopen(context.Background(), actor, changed); !errors.Is(err, ErrFiscalPeriodIdempotencyConflict) {
		t.Fatalf("changed request idempotency error = %v", err)
	}
	decision := reopenDecision(requested.State.ReopenRequest, ReopenApprovalApproved, actor.UserID)
	selfApprovalResult, err := service.ApplyReopenApprovalDecision(context.Background(), actor, ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: requested.State.ReopenRequest.ID,
		ExpectedPeriodVersion: requested.State.Period.AggregateVersion, ExpectedReopenRequestVersion: requested.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: requestCommand.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: "self-approved-reopen", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if !errors.Is(err, ErrReopenDecisionRejected) || gate.openScopedCalls != 0 {
		t.Fatalf("self approval = result=%#v err=%v gateCalls=%d", selfApprovalResult, err, gate.openScopedCalls)
	}
}

func TestReconcileReopenExpiryRetainsRestrictiveOwner(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeOperational)
	current := now
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}, openResult: gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: state.Period.AccountingScopeID, FiscalPeriodID: state.Period.ID, PeriodStateVersion: state.Period.StateVersion, NextLedgerPosition: 100}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return current })
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestReopen(context.Background(), testActor(), requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	decision := reopenDecision(requested.State.ReopenRequest, ReopenApprovalApproved, uuid.New())
	active, err := service.ApplyReopenApprovalDecision(context.Background(), testActor(), ApplyReopenApprovalDecisionCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: requested.State.ReopenRequest.ID,
		ExpectedPeriodVersion: requested.State.Period.AggregateVersion, ExpectedReopenRequestVersion: requested.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: requestCommand.ExpectedGateVersion,
		Decision: decision, IdempotencyKey: "apply-operational-expiry", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	current = requestCommand.AuthorizationExpiry.Add(time.Minute)
	expired, err := service.ReconcileReopenExpiry(context.Background(), testActor(), ReconcileReopenExpiryCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: active.State.ReopenRequest.ID,
		ExpectedPeriodVersion: active.State.Period.AggregateVersion, ExpectedReopenRequestVersion: active.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: active.State.ReopenRequest.GateVersion,
		IdempotencyKey: "reconcile-operational-expiry", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if expired.State.Period.State != PeriodStateReopening || expired.State.ReopenRequest.Status != ReopenRequestExpiredPendingClosure || expired.GateResult.Gate.GateMode != gl.PostingGateOperationalReopen {
		t.Fatalf("expired reopen result = %#v / %#v", expired.State.Period, expired.State.ReopenRequest)
	}
	if events := repository.Events(); len(events) != 4 || events[3].EventType != "OperationalReopenRequestExpired" {
		t.Fatalf("expiry events = %#v", events)
	}
	gate.err = ErrReopenGateUnavailable
	replayed, err := service.ReconcileReopenExpiry(context.Background(), testActor(), ReconcileReopenExpiryCommand{
		CommandID: uuid.New(), AccountingScopeID: requestCommand.AccountingScopeID, FiscalPeriodID: requestCommand.FiscalPeriodID, ReopenRequestID: active.State.ReopenRequest.ID,
		ExpectedPeriodVersion: active.State.Period.AggregateVersion, ExpectedReopenRequestVersion: active.State.ReopenRequest.AggregateVersion, ExpectedGateVersion: active.State.ReopenRequest.GateVersion,
		IdempotencyKey: "reconcile-operational-expiry", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(),
	})
	if err != nil || !replayed.Replayed || replayed.State.ReopenRequest.Status != ReopenRequestExpiredPendingClosure || replayed.GateResult.Gate.ID != expired.GateResult.Gate.ID {
		t.Fatalf("expiry replay = %#v / %v", replayed, err)
	}
}

func TestReopenEvidenceCannotBeMutatedAfterRequest(t *testing.T) {
	state, requestCommand, now := reopenFixture(ReopenModeScopedCorrection)
	repository := NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &reopenGateStub{hardCloseGateStub: &hardCloseGateStub{result: gl.SoftCloseGateResult{Gate: hardClosedGate(state, requestCommand.ExpectedGateVersion)}}}
	service, err := NewServiceWithDependencies(repository, AllowAllAuthorizer{}, gate, CloseRunDependencies{ReopenWorkflow: reopenDecisionStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	requested, err := service.RequestReopen(context.Background(), testActor(), requestCommand)
	if err != nil {
		t.Fatal(err)
	}
	mutated := cloneControlState(requested.State)
	mutated.ReopenRequest.Reason = "changed after request"
	mutated.ReopenRequest.AggregateVersion++
	mutated.Period.AggregateVersion++
	if err := repository.Commit(context.Background(), Mutation{Before: requested.State, After: mutated}); !errors.Is(err, ErrFiscalPeriodPolicyConflict) {
		t.Fatalf("mutated reopen evidence error = %v", err)
	}
	stored, err := repository.Get(context.Background(), requestCommand.AccountingScopeID, requestCommand.FiscalPeriodID)
	if err != nil || stored.ReopenRequest.Reason != requestCommand.Reason {
		t.Fatalf("stored reopen evidence = %#v, err = %v", stored.ReopenRequest, err)
	}
}

func reopenFixture(mode string) (ControlState, RequestReopenCommand, time.Time) {
	scopeID, periodID, softCloseRunID, closeRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	watermark := int64(99)
	finalGate := gl.SoftCloseGateResult{Gate: hardClosedGateValues(scopeID, periodID, 7)}
	state := ControlState{
		Period:       FiscalPeriod{ID: periodID, AccountingScopeID: scopeID, State: PeriodStateHardClosed, StateVersion: 3, AggregateVersion: 3, CreatedAt: now, UpdatedAt: now},
		SoftClose:    SoftCloseRun{ID: softCloseRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Status: SoftCloseRunCompleted, ControlEpoch: 2, OpenedGateVersion: 6, PolicyVersion: "soft-close-policy-v1", AggregateVersion: 2, CreatedAt: now, UpdatedAt: now},
		ControlEpoch: SoftCloseControlEpoch{SoftCloseRunID: softCloseRunID, FiscalPeriodID: periodID, Number: 2, OpenedGateVersion: 6, PolicyVersion: "soft-close-policy-v1", Status: ControlEpochActive, AggregateVersion: 2, CreatedAt: now, UpdatedAt: now},
		CloseRun:     &CloseRun{ID: closeRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, CloseType: gl.CloseTypeInitialHardClose, Status: CloseRunCompleted, AggregateVersion: 4, CommandFingerprint: "sha256:close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "barrier", FinalLedgerWatermark: &watermark, FinalizationGateResult: &finalGate, SealStatus: CloseRunSealSealed, CreatedAt: now, UpdatedAt: now},
	}
	command := RequestReopenCommand{CommandID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, Mode: mode, Reason: "Correct a bounded post-close classification error", Scope: ReopenScope{TransactionClasses: []string{"ReopenCorrection"}}, ProposedCorrections: []byte(`{"replacement":"classification"}`), Impact: []byte(`{"materiality":"low"}`), AuthorizationExpiry: now.Add(time.Hour), ExpectedPeriodVersion: 3, ExpectedGateVersion: 7, ControlAuthorityEpoch: 2, IdempotencyKey: "request-reopen", CorrelationID: uuid.NewString(), CausationID: uuid.NewString()}
	if mode == ReopenModeScopedCorrection {
		command.Scope.AccountIDs = []uuid.UUID{uuid.New()}
	} else {
		command.Scope.ActorIDs = []uuid.UUID{uuid.New()}
	}
	return state, command, now
}

func reopenDecision(request *ReopenRequest, outcome string, approverID uuid.UUID) ReopenApprovalDecision {
	return ReopenApprovalDecision{RequestID: uuid.New(), DecisionID: uuid.New(), DecisionVersion: 1, Outcome: outcome, Reason: "Independent controller review", PolicyVersion: "reopen-policy-v1", SubjectVersion: 1, CandidateFingerprint: request.RequestFingerprint, ApproverUserID: approverID, DecisionReference: uuid.New(), SeparationOfDutiesReference: "sod-1", RecordedAt: request.UpdatedAt.Add(time.Minute), DecidedAt: request.UpdatedAt.Add(time.Minute)}
}

func hardClosedGate(state ControlState, version int64) gl.PeriodPostingGate {
	return hardClosedGateValues(state.Period.AccountingScopeID, state.Period.ID, version)
}

func hardClosedGateValues(scopeID, periodID uuid.UUID, version int64) gl.PeriodPostingGate {
	return gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateHardClosed, PeriodStateVersion: 3, GateVersion: version, AggregateVersion: version, NextLedgerPosition: 100}
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

var _ PostingGateApplicationPort = (*reopenGateStub)(nil)
var _ ReopenDecisionPort = reopenDecisionStub{}
