package httpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

type fiscalPeriodHandlerGateStub struct {
	result gl.SoftCloseGateResult
	err    error
}

func (stub *fiscalPeriodHandlerGateStub) AcquirePostingBarrier(_ context.Context, _ gl.Actor, command gl.AcquirePostingBarrierCommand) (gl.SoftCloseGateResult, error) {
	if stub.err != nil {
		return gl.SoftCloseGateResult{}, stub.err
	}
	result := stub.result
	result.Gate.ActiveCloseRunID = cloneHandlerUUID(&command.CloseRunID)
	result.Gate.PriorControlProcessID = cloneHandlerUUID(&command.SoftCloseRunID)
	result.Gate.PriorControlEpoch = cloneHandlerInt64(&command.SoftCloseControlEpoch)
	result.Gate.LastFrozenAdmissionSummary.ControlOwnerID = command.SoftCloseRunID
	result.Gate.LastFrozenAdmissionSummary.ControlOwnerEpoch = cloneHandlerInt64(&command.SoftCloseControlEpoch)
	result.Gate.ActiveAdmissionCounters.ControlOwnerType = gl.PostingBarrierOwnerType
	result.Gate.ActiveAdmissionCounters.ControlOwnerID = command.CloseRunID
	return result, nil
}

func (stub *fiscalPeriodHandlerGateStub) GetPostingGateStatus(_ context.Context, _, _ uuid.UUID) (gl.PostingGateStatus, error) {
	return stub.result.Gate, stub.err
}

func TestFPMHardCloseHandlersStartAndApplyAuthoritativeResult(t *testing.T) {
	scopeID, periodID, softCloseRunID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	state := fiscalperiod.ControlState{
		Period:       fiscalperiod.FiscalPeriod{ID: periodID, AccountingScopeID: scopeID, State: fiscalperiod.PeriodStateSoftClosed, StateVersion: 1, AggregateVersion: 1, ActiveSoftCloseRunID: &softCloseRunID, CreatedAt: now, UpdatedAt: now},
		SoftClose:    fiscalperiod.SoftCloseRun{ID: softCloseRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Status: fiscalperiod.SoftCloseRunActive, ControlEpoch: 1, OpenedGateVersion: 2, PolicyVersion: "soft-close-policy-v1", AggregateVersion: 1, CreatedAt: now, UpdatedAt: now},
		ControlEpoch: fiscalperiod.SoftCloseControlEpoch{SoftCloseRunID: softCloseRunID, FiscalPeriodID: periodID, Number: 1, OpenedGateVersion: 2, PolicyVersion: "soft-close-policy-v1", Status: fiscalperiod.ControlEpochActive, AggregateVersion: 1, CreatedAt: now, UpdatedAt: now},
	}
	repository := fiscalperiod.NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	closeRunID := uuid.New()
	epoch := int64(1)
	summary := gl.FrozenGateAdmissionSummary{ControlOwnerType: gl.SoftCloseGateOwnerType, ControlOwnerID: softCloseRunID, ControlOwnerEpoch: &epoch, FrozenAtGateVersion: 3, FrozenByCommandFingerprint: "sha256:barrier", FrozenByCommandType: gl.PostingBarrierAcquireCommandIdentity}
	gate := &fiscalPeriodHandlerGateStub{result: gl.SoftCloseGateResult{Operation: gl.PostingBarrierAcquireOperation, Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateCloseOnly, PeriodStateVersion: 1, GateVersion: 3, AggregateVersion: 3, NextLedgerPosition: 30, ActiveCloseRunID: &closeRunID, ActiveCloseType: gl.CloseTypeInitialHardClose, PriorControlProcessID: &softCloseRunID, PriorControlProcessType: gl.SoftCloseGateOwnerType, PriorControlEpoch: &epoch, PriorGateMode: gl.PostingGateSoftClosePolicy, BarrierLedgerPosition: 29, ActiveAdmissionCounters: gl.ActiveGateAdmissionCounters{ControlOwnerType: gl.PostingBarrierOwnerType, ControlOwnerID: closeRunID}, LastFrozenAdmissionSummary: &summary}, DecisionReference: uuid.New(), PolicyReference: "gl-policy", AuthorizationPolicyVersion: "gl-policy-v1", ValidationOutcome: "accepted", AuditReference: uuid.New(), NextAction: "continue-close"}}
	service, err := fiscalperiod.NewService(repository, fiscalperiod.AllowAllAuthorizer{}, gate, func() time.Time { return now.Add(time.Minute) })
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{FiscalPeriodService: service}
	startRequest := fpmHandlerRequest(scopeID, uuid.New(), 1, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "softCloseControlEpoch": 1, "policyVersion": "soft-close-policy-v1", "expectedGateVersion": 2})
	startResponse, err := handler.FpmStartHardClose(ctx, startRequest, generated.FpmStartHardCloseParams{IdempotencyKey: "handler-hard-close"})
	if err != nil {
		t.Fatal(err)
	}
	started, ok := startResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("start response = %#v", startResponse)
	}
	var periodState string
	if err := json.Unmarshal([]byte(started.Data["periodState"]), &periodState); err != nil || periodState != fiscalperiod.PeriodStateClosing {
		t.Fatalf("period state = %q, err = %v", periodState, err)
	}

	var startedState fiscalperiod.ControlState
	startedState, err = repository.Get(context.Background(), scopeID, periodID)
	if err != nil || startedState.CloseRun == nil {
		t.Fatalf("started state = %#v, err = %v", startedState, err)
	}
	authoritative := gate.result
	authoritative.Gate.ActiveCloseRunID = cloneHandlerUUID(&startedState.CloseRun.ID)
	authoritative.Gate.PriorControlProcessID = cloneHandlerUUID(&softCloseRunID)
	authoritative.Gate.PriorControlEpoch = cloneHandlerInt64(&epoch)
	authoritative.Gate.ActiveAdmissionCounters.ControlOwnerType = gl.PostingBarrierOwnerType
	authoritative.Gate.ActiveAdmissionCounters.ControlOwnerID = startedState.CloseRun.ID
	applyRequest := fpmHandlerRequest(scopeID, uuid.New(), 1, map[string]any{
		"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "closeRunId": startedState.CloseRun.ID.String(),
		"softCloseControlEpoch": 1, "policyVersion": "soft-close-policy-v1", "expectedGateVersion": 2,
		"postingGateResult": authoritative,
	})
	applyResponse, err := handler.FpmApplyPostingGateResult(ctx, applyRequest, generated.FpmApplyPostingGateResultParams{IdempotencyKey: "handler-hard-close-apply"})
	if err != nil {
		t.Fatal(err)
	}
	if established, ok := applyResponse.(*generated.EstablishedResult); !ok || established.AggregateVersion != started.AggregateVersion {
		t.Fatalf("apply response = %#v", applyResponse)
	}
}

func TestFPMHardCloseHandlerRequiresApplicationActor(t *testing.T) {
	service, err := fiscalperiod.NewService(fiscalperiod.NewMemoryRepository(), fiscalperiod.AllowAllAuthorizer{}, &fiscalPeriodHandlerGateStub{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{FiscalPeriodService: service}
	response, err := handler.FpmStartHardClose(context.Background(), &generated.CommandRequest{}, generated.FpmStartHardCloseParams{IdempotencyKey: "unauthorized"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := response.(*generated.FpmStartHardCloseServiceUnavailable); ok {
		t.Fatalf("missing actor incorrectly reported service unavailable: %#v", response)
	}
	if problem, ok := response.(*generated.FpmStartHardCloseForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("missing actor response = %#v", response)
	}
}

func fpmHandlerRequest(scopeID, commandID uuid.UUID, expectedVersion int, data map[string]any) *generated.CommandRequest {
	requestData := generated.CommandRequestData{}
	for key, value := range data {
		requestData[key] = mustRaw(value)
	}
	return &generated.CommandRequest{CommandId: generated.UUID(commandID), ExpectedVersion: generated.NewOptInt(expectedVersion), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: requestData}
}

func cloneHandlerUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneHandlerInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
