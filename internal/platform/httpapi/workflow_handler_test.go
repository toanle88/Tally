package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/workflow"
)

func TestWorkflowPolicyHandlerEstablishesReplaysAndMapsConflicts(t *testing.T) {
	service, _ := newHTTPWorkflowPolicyService(t, workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.MaintainApprovalPoliciesPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()})
	handler := IdentityHandler{ApprovalPolicyService: service}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}})
	if err != nil {
		t.Fatal(err)
	}
	request := newHTTPWorkflowPolicyRequest(workflowPolicyActionCreate, uuid.Nil, 0)
	params := generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-http-1", XCorrelationID: generated.NewOptUUID(generated.UUID(uuid.New()))}
	response, err := handler.WfaMaintainApprovalPolicies(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	established, ok := response.(*generated.WfaMaintainApprovalPoliciesEstablishedResult)
	if !ok || established.Status != "established" || established.Data.Policy.Status != generated.WfaApprovalPolicyProjectionStatusDraft || !established.Data.AuditReference.Set {
		t.Fatalf("established response = %#v", response)
	}
	if established.Data.Policy.Rules[0].Threshold.Minimum.Value != generated.Money("0.1") {
		t.Fatalf("threshold projection = %#v", established.Data.Policy.Rules[0].Threshold)
	}

	replay, err := handler.WfaMaintainApprovalPolicies(actorContext, request, params)
	if err != nil {
		t.Fatal(err)
	}
	replayed, ok := replay.(*generated.WfaMaintainApprovalPoliciesEstablishedResult)
	if !ok || !replayed.Data.Replayed.Set || !replayed.Data.Replayed.Value {
		t.Fatalf("replay response = %#v", replay)
	}

	changed := *request
	changed.Data.Name = "changed content"
	conflict, err := handler.WfaMaintainApprovalPolicies(actorContext, &changed, params)
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := conflict.(*generated.WfaMaintainApprovalPoliciesConflict); !ok || value.Code != "IDEMPOTENCY_CONFLICT" {
		t.Fatalf("idempotency response = %#v", conflict)
	}

	invalid := *request
	invalid.Data.Rules = []generated.WfaApprovalRule{{ID: generated.UUID(uuid.New()), Threshold: generated.WfaApprovalRuleThreshold{Minimum: generated.NewOptMoney(generated.Money("not-decimal")), Currency: generated.CurrencyCode("USD")}, RequiredRoles: request.Data.Rules[0].RequiredRoles}}
	invalidResponse, err := handler.WfaMaintainApprovalPolicies(actorContext, &invalid, generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-http-invalid"})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := invalidResponse.(*generated.WfaMaintainApprovalPoliciesBadRequest); !ok || value.Code != "INVALID_REQUEST" {
		t.Fatalf("invalid response = %#v", invalidResponse)
	}
}

func TestWorkflowPolicyHandlerMapsAuthorizationLifecycleAndHeaders(t *testing.T) {
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	deniedService, _ := newHTTPWorkflowPolicyService(t, workflow.AuthorizationDecision{Allowed: false, Outcome: workflow.AuthorizationDenied, Permission: workflow.MaintainApprovalPoliciesPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()})
	denied, err := (IdentityHandler{ApprovalPolicyService: deniedService}).WfaMaintainApprovalPolicies(ctx, newHTTPWorkflowPolicyRequest(workflowPolicyActionCreate, uuid.Nil, 0), generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-denied"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := denied.(*generated.WfaMaintainApprovalPoliciesForbidden); !ok {
		t.Fatalf("denied response = %#v", denied)
	}

	service, repository := newHTTPWorkflowPolicyService(t, workflow.AuthorizationDecision{Allowed: true, Outcome: workflow.AuthorizationAllowed, Permission: workflow.MaintainApprovalPoliciesPermission, ApprovedScopeIDs: []string{"scope-1"}, DecisionReference: uuid.New()})
	created, err := (IdentityHandler{ApprovalPolicyService: service}).WfaMaintainApprovalPolicies(ctx, newHTTPWorkflowPolicyRequest(workflowPolicyActionCreate, uuid.Nil, 0), generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-create"})
	if err != nil {
		t.Fatal(err)
	}
	createdResult := created.(*generated.WfaMaintainApprovalPoliciesEstablishedResult)
	stale := newHTTPWorkflowPolicyRequest(workflowPolicyActionUpdate, uuid.UUID(createdResult.AggregateId), createdResult.AggregateVersion+1)
	stale.Data.PolicyVersion = generated.NewOptString("v2")
	staleResponse, err := (IdentityHandler{ApprovalPolicyService: service}).WfaMaintainApprovalPolicies(ctx, stale, generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-stale", IfMatch: generated.NewOptString(`"99"`)})
	if err != nil {
		t.Fatal(err)
	}
	if value, ok := staleResponse.(*generated.WfaMaintainApprovalPoliciesConflict); !ok || value.Code != "VERSION_CONFLICT" {
		t.Fatalf("stale response = %#v", staleResponse)
	}

	activate := newHTTPWorkflowPolicyRequest(workflowPolicyActionActivate, uuid.UUID(createdResult.AggregateId), createdResult.AggregateVersion)
	activate.Data.PolicyVersion = generated.NewOptString("v1")
	activateResponse, err := (IdentityHandler{ApprovalPolicyService: service}).WfaMaintainApprovalPolicies(ctx, activate, generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-activate"})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := activateResponse.(*generated.WfaMaintainApprovalPoliciesUnprocessableEntity); !ok {
		t.Fatalf("missing approval response = %#v", activateResponse)
	}
	if _, err := repository.Get(context.Background(), uuid.UUID(createdResult.AggregateId)); err != nil {
		t.Fatal(err)
	}
}

func TestWorkflowPolicyHandlerFailsClosedForActorAndService(t *testing.T) {
	request := newHTTPWorkflowPolicyRequest(workflowPolicyActionCreate, uuid.Nil, 0)
	params := generated.WfaMaintainApprovalPoliciesParams{IdempotencyKey: "wfa-auth"}
	if response, err := (IdentityHandler{ApprovalPolicyService: nil}).WfaMaintainApprovalPolicies(context.Background(), request, params); err != nil {
		t.Fatal(err)
	} else if _, ok := response.(*generated.WfaMaintainApprovalPoliciesServiceUnavailable); !ok {
		t.Fatalf("missing service response = %#v", response)
	}
	service, _ := newHTTPWorkflowPolicyService(t, workflow.AuthorizationDecision{}, workflow.ErrApprovalPolicyAuthorizationUnavailable)
	if response, err := (IdentityHandler{ApprovalPolicyService: service}).WfaMaintainApprovalPolicies(context.Background(), request, params); err != nil {
		t.Fatal(err)
	} else if _, ok := response.(*generated.WfaMaintainApprovalPoliciesForbidden); !ok {
		t.Fatalf("missing actor response = %#v", response)
	}
}

const (
	workflowPolicyActionCreate   = generated.WfaMaintainApprovalPoliciesCommandDataActionCreate
	workflowPolicyActionUpdate   = generated.WfaMaintainApprovalPoliciesCommandDataActionUpdate
	workflowPolicyActionActivate = generated.WfaMaintainApprovalPoliciesCommandDataActionActivate
)

func newHTTPWorkflowPolicyService(t *testing.T, decision workflow.AuthorizationDecision, authorizationError ...error) (*workflow.ApprovalPolicyService, *workflow.MemoryApprovalPolicyRepository) {
	t.Helper()
	repository := workflow.NewMemoryApprovalPolicyRepository()
	approvalWorkflow, err := workflow.NewMemoryApprovalWorkflow()
	if err != nil {
		t.Fatal(err)
	}
	authorizer := workflow.MemoryApprovalPolicyAuthorizer{Decision: decision}
	if len(authorizationError) > 0 {
		authorizer.Err = authorizationError[0]
	}
	service, err := workflow.NewApprovalPolicyService(repository, authorizer, approvalWorkflow, &workflow.MemoryApprovalPolicyAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	return service, repository
}

func newHTTPWorkflowPolicyRequest(action generated.WfaMaintainApprovalPoliciesCommandDataAction, policyID uuid.UUID, expectedVersion int) *generated.WfaMaintainApprovalPoliciesCommandRequest {
	request := &generated.WfaMaintainApprovalPoliciesCommandRequest{CommandId: generated.UUID(uuid.New()), Data: generated.WfaMaintainApprovalPoliciesCommandData{Action: action, Code: "gl-posting", Name: "GL posting policy", PolicyVersion: generated.NewOptString("v1"), Applicability: generated.WfaApprovalApplicability{SourceContext: "general-ledger", SubjectType: "posting-request", Operation: "submit", Currencies: []generated.CurrencyCode{"USD"}}, Scope: generated.WfaApprovalScope{ScopeIds: []string{"scope-1"}}, Rules: []generated.WfaApprovalRule{{ID: generated.UUID(uuid.New()), Threshold: generated.WfaApprovalRuleThreshold{Minimum: generated.NewOptMoney(generated.Money("0.10")), Maximum: generated.NewOptMoney(generated.Money("1000.00")), Currency: generated.CurrencyCode("USD")}, RequiredRoles: []generated.WfaRoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}}}, Steps: []generated.WfaApprovalStepDefinition{{ID: generated.UUID(uuid.New()), Sequence: 1, Name: "Review", RequiredRoles: []generated.WfaRoleRequirement{{Role: "finance.approver", MinimumApprovals: 1}}, MinimumApprovals: 1}}, EffectiveFrom: time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)}}
	if policyID != uuid.Nil {
		request.Data.PolicyId = generated.NewOptUUID(generated.UUID(policyID))
	}
	if expectedVersion > 0 {
		request.ExpectedVersion = generated.NewOptInt(expectedVersion)
	}
	return request
}
