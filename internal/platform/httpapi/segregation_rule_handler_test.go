package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestIdentityHandlerManagesTypedSegregationRulesAndMapsSafeOutcomes(t *testing.T) {
	now := time.Date(2026, 9, 26, 8, 0, 0, 0, time.UTC)
	repository := memorySegregationRepoForAPI(t)
	authorizer := identity.MemorySegregationRuleAuthorizer{Decision: identity.AuthorizationDecision{Allowed: true, Outcome: identity.AuthorizationAllowed, Permission: identity.SegregationRuleManagementPermission, PolicyReference: "segregation-policy-v1", PolicyVersion: "segregation-policy-v1", DecisionReference: uuid.New()}}
	service, err := identity.NewSegregationRuleService(repository, authorizer, apiSegregationRuleApproval{}, &identity.MemorySegregationRuleAuditRecorder{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	handler := IdentityHandler{SegregationRuleService: service}
	request := testSegregationRuleRequest(generated.IamManageSegregationRulesCommandDataActionCreate, generated.OptUUID{}, "api-rule", "API rule")
	created, err := handler.IamManageSegregationRules(ctx, request, generated.IamManageSegregationRulesParams{IdempotencyKey: "api-rule-create"})
	if err != nil {
		t.Fatal(err)
	}
	createdResult, ok := created.(*generated.EstablishedResult)
	if !ok || createdResult.AggregateVersion != 1 {
		t.Fatalf("created response = %#v", created)
	}
	update := testSegregationRuleRequest(generated.IamManageSegregationRulesCommandDataActionUpdate, generated.NewOptUUID(createdResult.AggregateId), "api-rule", "API rule v2")
	update.ExpectedVersion = generated.NewOptInt(1)
	updated, err := handler.IamManageSegregationRules(ctx, update, generated.IamManageSegregationRulesParams{IdempotencyKey: "api-rule-update", IfMatch: generated.NewOptString("\"1\"")})
	if err != nil {
		t.Fatal(err)
	}
	updatedResult, ok := updated.(*generated.EstablishedResult)
	if !ok || updatedResult.AggregateVersion != 2 {
		t.Fatalf("updated response = %#v", updated)
	}
	conflictRequest := testSegregationRuleRequest(generated.IamManageSegregationRulesCommandDataActionUpdate, generated.NewOptUUID(createdResult.AggregateId), "api-rule", "API rule conflict")
	conflictRequest.ExpectedVersion = generated.NewOptInt(1)
	conflict, err := handler.IamManageSegregationRules(ctx, conflictRequest, generated.IamManageSegregationRulesParams{IdempotencyKey: "api-rule-conflict", IfMatch: generated.NewOptString("\"2\"")})
	if err != nil {
		t.Fatal(err)
	}
	if response, ok := conflict.(*generated.IamManageSegregationRulesConflict); !ok || response.Code != "VERSION_CONFLICT" {
		t.Fatalf("conflict response = %#v", conflict)
	}

	deniedAuthorizer := identity.MemorySegregationRuleAuthorizer{Decision: identity.AuthorizationDecision{Allowed: false, Outcome: identity.AuthorizationDenied, Permission: identity.SegregationRuleManagementPermission, DecisionReference: uuid.New()}}
	deniedService, err := identity.NewSegregationRuleService(memorySegregationRepoForAPI(t), deniedAuthorizer, apiSegregationRuleApproval{}, &identity.MemorySegregationRuleAuditRecorder{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	denied, err := (IdentityHandler{SegregationRuleService: deniedService}).IamManageSegregationRules(ctx, request, generated.IamManageSegregationRulesParams{IdempotencyKey: "api-rule-denied"})
	if err != nil {
		t.Fatal(err)
	}
	if response, ok := denied.(*generated.IamManageSegregationRulesForbidden); !ok || response.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denied response = %#v", denied)
	}

	staleAuthorizer := identity.MemorySegregationRuleAuthorizer{Decision: identity.AuthorizationDecision{Allowed: false, Outcome: identity.AuthorizationStale, Permission: identity.SegregationRuleManagementPermission, DecisionReference: uuid.New()}}
	staleService, err := identity.NewSegregationRuleService(memorySegregationRepoForAPI(t), staleAuthorizer, apiSegregationRuleApproval{}, &identity.MemorySegregationRuleAuditRecorder{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	stale, err := (IdentityHandler{SegregationRuleService: staleService}).IamManageSegregationRules(ctx, request, generated.IamManageSegregationRulesParams{IdempotencyKey: "api-rule-stale"})
	if err != nil {
		t.Fatal(err)
	}
	if response, ok := stale.(*generated.IamManageSegregationRulesConflict); !ok || response.Code != "POLICY_STALE" {
		t.Fatalf("stale response = %#v", stale)
	}
}

func TestIdentityGeneratedServerRequiresSegregationRuleIdempotencyHeader(t *testing.T) {
	repository := memorySegregationRepoForAPI(t)
	service, err := identity.NewSegregationRuleService(repository, identity.MemorySegregationRuleAuthorizer{Decision: identity.AuthorizationDecision{Allowed: true, Outcome: identity.AuthorizationAllowed, Permission: identity.SegregationRuleManagementPermission, DecisionReference: uuid.New()}}, apiSegregationRuleApproval{}, &identity.MemorySegregationRuleAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "sub"}}
	server, err := generated.NewServer(IdentityHandler{SegregationRuleService: service}, testIdentitySecurity{actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	request := testSegregationRuleRequest(generated.IamManageSegregationRulesCommandDataActionCreate, generated.OptUUID{}, "generated-rule", "Generated rule")
	body := mustJSON(t, request)
	httpRequest := httptest.NewRequest(http.MethodPost, "/identity-access/actions/manage-segregation-rules", bytes.NewReader(body))
	httpRequest.Header.Set("Authorization", "Bearer fixture")
	httpRequest.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, httpRequest)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency status = %d, want 400", response.Code)
	}
}

type apiSegregationRuleApproval struct{}

func (apiSegregationRuleApproval) ValidateSegregationRuleApproval(context.Context, identity.ApplicationActor, identity.SegregationRuleCommand, *identity.SegregationRule, string) error {
	return nil
}

func testSegregationRuleRequest(action generated.IamManageSegregationRulesCommandDataAction, ruleID generated.OptUUID, code, name string) *generated.IamManageSegregationRulesCommandRequest {
	return &generated.IamManageSegregationRulesCommandRequest{CommandId: generated.UUID(uuid.New()), Data: generated.IamManageSegregationRulesCommandData{Action: action, RuleId: ruleID, Code: code, Name: name, ConflictingPermissions: []string{"finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"}, EnforcementMode: generated.IamManageSegregationRulesCommandDataEnforcementModeBlock, EffectiveFrom: time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC), Approval: generated.IamApprovalDecisionReference{ApprovalRequestId: generated.UUID(uuid.New()), DecisionId: generated.UUID(uuid.New()), PolicyVersion: "rule-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:api", ApproverUserId: generated.UUID(uuid.New())}}}
}

func mustJSON(t *testing.T, value any) []byte {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func memorySegregationRepoForAPI(t *testing.T) *identity.MemorySegregationRuleRepository {
	t.Helper()
	repository, err := identity.NewMemorySegregationRuleRepository()
	if err != nil {
		t.Fatal(err)
	}
	return repository
}
