package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/fiscalperiod"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestFPMRecoveryHandlersFailClosedWithoutAnApplicationActor(t *testing.T) {
	service, err := fiscalperiod.NewService(fiscalperiod.NewMemoryRepository(), fiscalperiod.AllowAllAuthorizer{}, &fiscalPeriodHandlerGateStub{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{FiscalPeriodService: service}
	takeover, err := handler.FpmTakeOverPeriodControl(context.Background(), &generated.CommandRequest{}, generated.FpmTakeOverPeriodControlParams{IdempotencyKey: "takeover-unauthorized"})
	if err != nil {
		t.Fatal(err)
	}
	if problem, ok := takeover.(*generated.FpmTakeOverPeriodControlForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("takeover unauthorized response = %#v", takeover)
	}
	extension, err := handler.FpmExtendCloseException(context.Background(), &generated.CommandRequest{}, generated.FpmExtendCloseExceptionParams{IdempotencyKey: "extension-unauthorized"})
	if err != nil {
		t.Fatal(err)
	}
	if problem, ok := extension.(*generated.FpmExtendCloseExceptionForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("extension unauthorized response = %#v", extension)
	}
}

func TestFPMRecoveryProblemMappingUsesTypedDependencyAndApprovalOutcomes(t *testing.T) {
	correlationID := uuid.New()
	takeover := mapFPMTakeOverPeriodControlError(fiscalperiod.ErrControlRecoveryDecisionUnavailable, correlationID)
	if problem, ok := takeover.(*generated.FpmTakeOverPeriodControlServiceUnavailable); !ok || problem.Code != "DEPENDENCY_UNAVAILABLE" || problem.CorrelationId != generated.UUID(correlationID) {
		t.Fatalf("takeover dependency response = %#v", takeover)
	}
	extension := mapFPMExtendCloseExceptionError(fiscalperiod.ErrCloseExceptionExtensionRejected, correlationID)
	if problem, ok := extension.(*generated.FpmExtendCloseExceptionConflict); !ok || problem.Code != "EXCEPTION_EXTENSION_REJECTED" {
		t.Fatalf("extension rejection response = %#v", extension)
	}
	approval := mapFPMTakeOverPeriodControlError(fiscalperiod.ErrControlRecoveryDecisionRejected, correlationID)
	if problem, ok := approval.(*generated.FpmTakeOverPeriodControlConflict); !ok || problem.Code != "APPROVAL_INVALIDATED" {
		t.Fatalf("approval response = %#v", approval)
	}
	if errors.Is(fiscalperiod.ErrControlRecoveryDecisionUnavailable, fiscalperiod.ErrControlRecoveryDecisionRejected) {
		t.Fatal("recovery dependency and approval errors unexpectedly overlap")
	}
}

func TestEstablishedFPMResultIncludesControlRecoveryProjection(t *testing.T) {
	scopeID, periodID, softRunID := uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	owner := fiscalperiod.Actor{UserID: uuid.New(), SubjectReference: "owner"}
	authority := &fiscalperiod.ControlAuthority{CurrentOwner: owner, AuthorityEpoch: 3, Version: 2, ExpiresAt: now.Add(time.Hour), CutoffAt: now.Add(-time.Hour)}
	state := fiscalperiod.ControlState{Period: fiscalperiod.FiscalPeriod{ID: periodID, AccountingScopeID: scopeID, State: fiscalperiod.PeriodStateSoftClosed, StateVersion: 1, AggregateVersion: 2, CreatedAt: now, UpdatedAt: now}, SoftClose: fiscalperiod.SoftCloseRun{ID: softRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Status: fiscalperiod.SoftCloseRunActive, ControlEpoch: 1, OpenedGateVersion: 2, PolicyVersion: "soft-close-policy-v1", AggregateVersion: 2, ControlAuthority: authority, CreatedAt: now, UpdatedAt: now}, ControlEpoch: fiscalperiod.SoftCloseControlEpoch{SoftCloseRunID: softRunID, FiscalPeriodID: periodID, Number: 1, OpenedGateVersion: 2, PolicyVersion: "soft-close-policy-v1", Status: fiscalperiod.ControlEpochActive, AggregateVersion: 1, CreatedAt: now, UpdatedAt: now}}
	gate := gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateSoftClosePolicy, PeriodStateVersion: 1, GateVersion: 2, AggregateVersion: 2, NextLedgerPosition: 10, ActiveSoftCloseRunID: &softRunID, ActiveSoftCloseControlEpoch: 1, SoftClosePolicyVersion: "soft-close-policy-v1"}}
	result := establishedFPMResult(fiscalperiod.CommandResult{State: state, GateResult: gate}, uuid.New(), "/fiscal-periods/actions/take-over-period-control")
	for _, key := range []string{"ownerId", "authorityEpoch", "authorityVersion", "authorityExpiresAt", "controlCutoffAt", "controlAuthority", "authoritativeGateStatus", "pendingReconciliation", "nextAction", "replayStatus"} {
		if _, ok := result.Data[key]; !ok {
			t.Fatalf("established result missing %q: %#v", key, result.Data)
		}
	}
	var ownerID string
	if err := json.Unmarshal([]byte(result.Data["ownerId"]), &ownerID); err != nil || ownerID != owner.UserID.String() {
		t.Fatalf("owner projection = %q, err=%v", ownerID, err)
	}
}
