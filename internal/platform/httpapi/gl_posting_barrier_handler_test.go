package httpapi

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestGLPostingBarrierHandlersAcquireReleaseAndExposeEvidence(t *testing.T) {
	scopeID, periodID, softCloseRunID, closeRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := gl.NewMemorySoftCloseGateRepository()
	if err := repository.Seed(gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(repository, gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "policy", PolicyVersion: "policy-v1"}}, gl.AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SoftCloseGateService: service}

	enter := softCloseGateRequest(scopeID, uuid.New(), 1, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "controlEpoch": 3, "policyVersion": "policy-v1", "restrictions": map[string]any{}})
	if response, err := handler.GlEnterSoftCloseGate(ctx, enter, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "handler-enter"}); err != nil {
		t.Fatal(err)
	} else if established, ok := response.(*generated.EstablishedResult); !ok || established.AggregateVersion != 2 {
		t.Fatalf("entry response = %#v", response)
	}

	acquire := softCloseGateRequest(scopeID, uuid.New(), 2, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "softCloseControlEpoch": 3, "policyVersion": "policy-v1", "closeRunId": closeRunID.String(), "closeType": gl.CloseTypeInitialHardClose})
	response, err := handler.GlAcquirePostingBarrier(ctx, acquire, generated.GlAcquirePostingBarrierParams{IdempotencyKey: "handler-acquire"})
	if err != nil {
		t.Fatal(err)
	}
	acquired, ok := response.(*generated.EstablishedResult)
	if !ok || acquired.AggregateVersion != 3 {
		t.Fatalf("acquire response = %#v", response)
	}
	var mode string
	if err := json.Unmarshal([]byte(acquired.Data["gateMode"]), &mode); err != nil || mode != gl.PostingGateCloseOnly {
		t.Fatalf("acquire mode = %q, err = %v", mode, err)
	}
	var owner string
	if err := json.Unmarshal([]byte(acquired.Data["ownerId"]), &owner); err != nil || owner != closeRunID.String() {
		t.Fatalf("acquire owner = %q, err = %v", owner, err)
	}

	release := softCloseGateRequest(scopeID, uuid.New(), 3, map[string]any{"fiscalPeriodId": periodID.String(), "closeRunId": closeRunID.String()})
	releaseResponse, err := handler.GlReleasePostingBarrier(ctx, release, generated.GlReleasePostingBarrierParams{IdempotencyKey: "handler-release"})
	if err != nil {
		t.Fatal(err)
	}
	released, ok := releaseResponse.(*generated.EstablishedResult)
	if !ok || released.AggregateVersion != 4 {
		t.Fatalf("release response = %#v", releaseResponse)
	}
	var epoch int64
	if err := json.Unmarshal([]byte(released.Data["controlEpoch"]), &epoch); err != nil || epoch != 4 {
		t.Fatalf("restored epoch = %d, err = %v", epoch, err)
	}
}

func TestGLPostingBarrierHandlersRejectInvalidCloseTypeAndFinalize(t *testing.T) {
	scopeID, periodID, softCloseRunID, closeRunID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := gl.NewMemorySoftCloseGateRepository()
	if err := repository.Seed(gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(repository, gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, gl.AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SoftCloseGateService: service}
	enter := softCloseGateRequest(scopeID, uuid.New(), 1, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "controlEpoch": 1, "policyVersion": "v1", "restrictions": map[string]any{}})
	if _, err := handler.GlEnterSoftCloseGate(ctx, enter, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "finalize-enter"}); err != nil {
		t.Fatal(err)
	}
	invalid := softCloseGateRequest(scopeID, uuid.New(), 2, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "softCloseControlEpoch": 1, "policyVersion": "v1", "closeRunId": closeRunID.String(), "closeType": gl.CloseTypeReclose})
	if response, err := handler.GlAcquirePostingBarrier(ctx, invalid, generated.GlAcquirePostingBarrierParams{IdempotencyKey: "invalid-close-type"}); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlAcquirePostingBarrierUnprocessableEntity); !ok || problem.Code != "VALIDATION_FAILED" {
		t.Fatalf("invalid type response = %#v", response)
	}

	valid := softCloseGateRequest(scopeID, uuid.New(), 2, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": softCloseRunID.String(), "softCloseControlEpoch": 1, "policyVersion": "v1", "closeRunId": closeRunID.String(), "closeType": gl.CloseTypeInitialHardClose})
	if _, err := handler.GlAcquirePostingBarrier(ctx, valid, generated.GlAcquirePostingBarrierParams{IdempotencyKey: "finalize-acquire"}); err != nil {
		t.Fatal(err)
	}
	finalize := softCloseGateRequest(scopeID, uuid.New(), 3, map[string]any{"fiscalPeriodId": periodID.String(), "closeRunId": closeRunID.String(), "finalLedgerWatermark": 9})
	response, err := handler.GlFinalizePostingGate(ctx, finalize, generated.GlFinalizePostingGateParams{IdempotencyKey: "finalize-gate"})
	if err != nil {
		t.Fatal(err)
	}
	if established, ok := response.(*generated.EstablishedResult); !ok || established.AggregateVersion != 4 {
		t.Fatalf("finalize response = %#v", response)
	}
}
