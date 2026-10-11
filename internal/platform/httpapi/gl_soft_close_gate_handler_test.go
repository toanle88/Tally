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

func TestGLSoftCloseGateHandlersReturnActionResultStateAndReplaySafely(t *testing.T) {
	scopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New()
	repository := gl.NewMemorySoftCloseGateRepository()
	if err := repository.Seed(gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(
		repository,
		gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "gate-policy", PolicyVersion: "gate-policy-v1"}},
		gl.AllowAllSoftClosePolicyEvaluator{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SoftCloseGateService: service}
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}

	commandID := uuid.New()
	correlationID := uuid.New()
	enterRequest := softCloseGateRequest(scopeID, uuid.New(), 1, map[string]any{
		"fiscalPeriodId": periodID.String(), "softCloseRunId": runID.String(), "controlEpoch": 4,
		"policyVersion": "soft-close-v1", "restrictions": map[string]any{"ordinaryPosting": "evaluator-controlled"},
	})
	enterRequest.CommandId = generated.UUID(commandID)
	enterResponse, err := handler.GlEnterSoftCloseGate(ctx, enterRequest, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "http-soft-close-enter", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID))})
	if err != nil {
		t.Fatal(err)
	}
	entered, ok := enterResponse.(*generated.EstablishedResult)
	if !ok || entered.Status != "established" || entered.AggregateVersion != 2 || entered.CorrelationId != generated.UUID(correlationID) {
		t.Fatalf("entry response = %#v", enterResponse)
	}
	var mode string
	if err := json.Unmarshal([]byte(entered.Data["gateMode"]), &mode); err != nil || mode != gl.PostingGateSoftClosePolicy {
		t.Fatalf("entry gate mode = %q, err = %v", mode, err)
	}
	var restrictions map[string]string
	if err := json.Unmarshal([]byte(entered.Data["restrictions"]), &restrictions); err != nil || restrictions["ordinaryPosting"] != "evaluator-controlled" {
		t.Fatalf("entry restrictions = %#v, err = %v", restrictions, err)
	}

	replayedResponse, err := handler.GlEnterSoftCloseGate(ctx, enterRequest, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "http-soft-close-enter", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID))})
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replayedResponse.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("replay response = %#v", replayedResponse)
	}
	var replayedValue bool
	if err := json.Unmarshal([]byte(replayed.Data["replayed"]), &replayedValue); err != nil || !replayedValue {
		t.Fatalf("replay marker = %v, err = %v", replayedValue, err)
	}

	exitRequest := softCloseGateRequest(scopeID, uuid.New(), 2, map[string]any{
		"fiscalPeriodId": periodID.String(), "softCloseRunId": runID.String(), "controlEpoch": 4,
	})
	exitResponse, err := handler.GlExitSoftCloseGate(ctx, exitRequest, generated.GlExitSoftCloseGateParams{IdempotencyKey: "http-soft-close-exit"})
	if err != nil {
		t.Fatal(err)
	}
	exited, ok := exitResponse.(*generated.EstablishedResult)
	if !ok || exited.AggregateVersion != 3 {
		t.Fatalf("exit response = %#v", exitResponse)
	}
	if err := json.Unmarshal([]byte(exited.Data["gateMode"]), &mode); err != nil || mode != gl.PostingGateOpen {
		t.Fatalf("exit gate mode = %q, err = %v", mode, err)
	}
	var summary gl.FrozenGateAdmissionSummary
	if err := json.Unmarshal([]byte(exited.Data["lastFrozenAdmissionSummary"]), &summary); err != nil || summary.FrozenAtGateVersion != 3 || summary.ControlOwnerID != runID {
		t.Fatalf("exit frozen summary = %#v, err = %v", summary, err)
	}

	changed := *enterRequest
	changed.Data["policyVersion"] = mustRaw("different-policy")
	conflictResponse, err := handler.GlEnterSoftCloseGate(ctx, &changed, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "http-soft-close-enter", XCorrelationID: generated.NewOptUUID(generated.UUID(correlationID))})
	if err != nil {
		t.Fatal(err)
	}
	if conflict, ok := conflictResponse.(*generated.GlEnterSoftCloseGateConflict); !ok || conflict.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("conflict response = %#v", conflictResponse)
	}
}

func TestGLSoftCloseGateHandlersFailClosedWithoutActorOrService(t *testing.T) {
	scopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New()
	request := softCloseGateRequest(scopeID, uuid.New(), 1, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": runID.String(), "controlEpoch": 1, "policyVersion": "v1", "restrictions": map[string]any{}})
	if response, err := (IdentityHandler{}).GlEnterSoftCloseGate(context.Background(), request, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "unavailable"}); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlEnterSoftCloseGateServiceUnavailable); !ok || problem.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("unavailable response = %#v", response)
	}

	repository := gl.NewMemorySoftCloseGateRepository()
	if err := repository.Seed(gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(repository, gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, gl.AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := (IdentityHandler{SoftCloseGateService: service}).GlEnterSoftCloseGate(context.Background(), request, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "no-actor"}); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlEnterSoftCloseGateForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("authorization response = %#v", response)
	}
}

func TestGLSoftCloseGateHandlerMapsScopeAndAuthorizationDependencies(t *testing.T) {
	scopeID, wrongScopeID, periodID, runID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	repository := gl.NewMemorySoftCloseGateRepository()
	if err := repository.Seed(gl.NewOpenMemorySoftCloseGate(scopeID, periodID, 1)); err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewSoftCloseGateService(repository, gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: false, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, gl.AllowAllSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := softCloseGateRequest(scopeID, uuid.New(), 1, map[string]any{"fiscalPeriodId": periodID.String(), "softCloseRunId": runID.String(), "controlEpoch": 1, "policyVersion": "v1", "restrictions": map[string]any{}})
	wrongScopeRequest := *request
	wrongScopeRequest.AccountingScopeId = generated.NewOptUUID(generated.UUID(wrongScopeID))
	if response, err := (IdentityHandler{SoftCloseGateService: service}).GlEnterSoftCloseGate(ctx, &wrongScopeRequest, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "scope-conflict", XAccountingScopeID: generated.NewOptUUID(generated.UUID(scopeID))}); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlEnterSoftCloseGateConflict); !ok || problem.Code != "SCOPE_CONFLICT" {
		t.Fatalf("scope response = %#v", response)
	}
	if response, err := (IdentityHandler{SoftCloseGateService: service}).GlEnterSoftCloseGate(ctx, request, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "authorization-denied"}); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlEnterSoftCloseGateForbidden); !ok || problem.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("authorization response = %#v", response)
	}

	policyService, err := gl.NewSoftCloseGateService(repository, gl.MemorySoftCloseGateAuthorizer{Decision: gl.AuthorizationDecision{Allowed: true, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, gl.UnavailableSoftClosePolicyEvaluator{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if response, err := (IdentityHandler{SoftCloseGateService: policyService}).GlEnterSoftCloseGate(ctx, request, generated.GlEnterSoftCloseGateParams{IdempotencyKey: "policy-unavailable"}); err != nil {
		t.Fatal(err)
	} else if problem, ok := response.(*generated.GlEnterSoftCloseGateServiceUnavailable); !ok || problem.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("policy response = %#v", response)
	}
}

func softCloseGateRequest(scopeID, commandID uuid.UUID, expectedVersion int, values map[string]any) *generated.CommandRequest {
	data := generated.CommandRequestData{}
	for key, value := range values {
		data[key] = mustRaw(value)
	}
	return &generated.CommandRequest{CommandId: generated.UUID(commandID), ExpectedVersion: generated.NewOptInt(expectedVersion), AccountingScopeId: generated.NewOptUUID(generated.UUID(scopeID)), Data: data}
}
