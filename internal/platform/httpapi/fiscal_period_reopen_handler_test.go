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

type fiscalPeriodReopenHandlerWorkflowStub struct{}

func (fiscalPeriodReopenHandlerWorkflowStub) Validate(context.Context, fiscalperiod.Actor, fiscalperiod.ReopenRequest, fiscalperiod.ReopenApprovalDecision) error {
	return nil
}

func TestFPMRequestAndRejectReopenHandlersUseGenericCommandEnvelope(t *testing.T) {
	scopeID, periodID, softCloseRunID, closeRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	now := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	watermark := int64(99)
	state := fiscalperiod.ControlState{
		Period:       fiscalperiod.FiscalPeriod{ID: periodID, AccountingScopeID: scopeID, State: fiscalperiod.PeriodStateHardClosed, StateVersion: 3, AggregateVersion: 3, CreatedAt: now, UpdatedAt: now},
		SoftClose:    fiscalperiod.SoftCloseRun{ID: softCloseRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, Status: fiscalperiod.SoftCloseRunCompleted, ControlEpoch: 2, OpenedGateVersion: 6, PolicyVersion: "soft-close-policy-v1", AggregateVersion: 2, CreatedAt: now, UpdatedAt: now},
		ControlEpoch: fiscalperiod.SoftCloseControlEpoch{SoftCloseRunID: softCloseRunID, FiscalPeriodID: periodID, Number: 2, OpenedGateVersion: 6, PolicyVersion: "soft-close-policy-v1", Status: fiscalperiod.ControlEpochActive, AggregateVersion: 2, CreatedAt: now, UpdatedAt: now},
		CloseRun:     &fiscalperiod.CloseRun{ID: closeRunID, AccountingScopeID: scopeID, FiscalPeriodID: periodID, CloseType: gl.CloseTypeInitialHardClose, Status: fiscalperiod.CloseRunCompleted, AggregateVersion: 4, CommandFingerprint: "sha256:close", BarrierCommandID: uuid.New(), BarrierIdempotencyKey: "barrier", FinalLedgerWatermark: &watermark, FinalizationGateResult: &gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateHardClosed, PeriodStateVersion: 3, GateVersion: 7, AggregateVersion: 7, NextLedgerPosition: 100}}, SealStatus: fiscalperiod.CloseRunSealSealed, CreatedAt: now, UpdatedAt: now},
	}
	repository := fiscalperiod.NewMemoryRepository()
	if err := repository.Seed(state); err != nil {
		t.Fatal(err)
	}
	gate := &fiscalPeriodHandlerGateStub{result: gl.SoftCloseGateResult{Gate: gl.PeriodPostingGate{ID: uuid.New(), AccountingScopeID: scopeID, FiscalPeriodID: periodID, GateMode: gl.PostingGateHardClosed, PeriodStateVersion: 3, GateVersion: 7, AggregateVersion: 7, NextLedgerPosition: 100}}}
	service, err := fiscalperiod.NewServiceWithDependencies(repository, fiscalperiod.AllowAllAuthorizer{}, gate, fiscalperiod.CloseRunDependencies{ReopenWorkflow: fiscalPeriodReopenHandlerWorkflowStub{}}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{FiscalPeriodService: service}
	request := fpmHandlerRequest(scopeID, uuid.New(), 3, map[string]any{
		"fiscalPeriodId": periodID.String(), "reopenMode": fiscalperiod.ReopenModeScopedCorrection, "reason": "Correct a bounded classification error",
		"accountIds": []string{uuid.NewString()}, "transactionClasses": []string{"ReopenCorrection"}, "actorIds": []string{},
		"proposedCorrections": map[string]any{"replacement": "classification"}, "impact": map[string]any{"materiality": "low"},
		"authorizationExpiry": now.Add(time.Hour).Format(time.RFC3339), "expectedGateVersion": 7, "controlAuthorityEpoch": 2,
	})
	requestResponse, err := handler.FpmRequestReopen(ctx, request, generated.FpmRequestReopenParams{IdempotencyKey: "handler-request-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	requested, ok := requestResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("request response = %#v", requestResponse)
	}
	var requestStatus string
	if err := json.Unmarshal([]byte(requested.Data["reopenStatus"]), &requestStatus); err != nil || requestStatus != fiscalperiod.ReopenRequestPendingApproval {
		t.Fatalf("request status = %q, err = %v", requestStatus, err)
	}

	stored, err := repository.Get(context.Background(), scopeID, periodID)
	if err != nil {
		t.Fatal(err)
	}
	approvalRequestID, decisionID, decisionReference, approverID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	apply := fpmHandlerRequest(scopeID, uuid.New(), int(stored.Period.AggregateVersion), map[string]any{
		"fiscalPeriodId": periodID.String(), "reopenRequestId": stored.ReopenRequest.ID.String(), "approvalRequestId": approvalRequestID.String(),
		"decisionId": decisionID.String(), "decisionVersion": 1, "outcome": fiscalperiod.ReopenApprovalRejected, "decisionReason": "Insufficient evidence",
		"policyVersion": "reopen-policy-v1", "subjectVersion": 1, "candidateFingerprint": stored.ReopenRequest.RequestFingerprint, "decisionReference": decisionReference.String(),
		"approverUserId": approverID.String(), "separationOfDutiesReference": "sod-1", "recordedAt": now.Add(2 * time.Minute).Format(time.RFC3339),
		"expectedReopenRequestVersion": 1, "expectedGateVersion": 7,
	})
	applyResponse, err := handler.FpmApplyReopenApprovalDecision(ctx, apply, generated.FpmApplyReopenApprovalDecisionParams{IdempotencyKey: "handler-reject-reopen"})
	if err != nil {
		t.Fatal(err)
	}
	rejected, ok := applyResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("apply response = %#v", applyResponse)
	}
	if err := json.Unmarshal([]byte(rejected.Data["reopenStatus"]), &requestStatus); err != nil || requestStatus != fiscalperiod.ReopenRequestRejected {
		t.Fatalf("rejected status = %q, err = %v", requestStatus, err)
	}
	if stored, err = repository.Get(context.Background(), scopeID, periodID); err != nil || stored.Period.State != fiscalperiod.PeriodStateHardClosed || stored.ReopenRequest.Status != fiscalperiod.ReopenRequestRejected {
		t.Fatalf("rejected state = %#v, err = %v", stored, err)
	}
}
