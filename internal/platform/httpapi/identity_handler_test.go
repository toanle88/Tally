package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestIdentityHandlerUsesTypedActionsAndMasksAuthenticationSubject(t *testing.T) {
	service, actor := testIdentityHTTPService(t, true)
	handler := IdentityHandler{Service: service}
	request := &generated.IamManageUsersCommandRequest{
		CommandId: generated.UUID(uuid.New()),
		Data: generated.IamManageUsersCommandData{
			Action: generated.IamManageUsersCommandDataActionCreate,
			AuthenticationSubject: generated.NewOptIamAuthenticationSubject(generated.IamAuthenticationSubject{
				Oid: "oid-secret", Tid: "tenant-secret", Sub: "subject-secret",
			}),
		},
	}
	params := generated.IamManageUsersParams{IdempotencyKey: "api-create-1"}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}

	response, err := handler.IamManageUsers(ctx, request, params)
	if err != nil {
		t.Fatal(err)
	}
	result, ok := response.(*generated.EstablishedResult)
	if !ok {
		t.Fatalf("response = %T, want *generated.EstablishedResult", response)
	}
	if result.Status != "established" || result.AggregateVersion != 1 {
		t.Fatalf("result = %#v, want established version 1", result)
	}
	data := result.Data["authenticationSubject"]
	if string(data) != `{"oid":"masked","sub":"masked","tid":"masked"}` {
		t.Fatalf("masked subject = %s, want masked fields only", data)
	}
	if string(data) == "oid-secret" || string(data) == "subject-secret" {
		t.Fatal("authentication subject leaked through established result")
	}

	userID := uuid.UUID(result.AggregateId)
	update := &generated.IamManageUsersCommandRequest{
		CommandId:       generated.UUID(uuid.New()),
		ExpectedVersion: generated.NewOptInt(1),
		Data: generated.IamManageUsersCommandData{
			Action:      generated.IamManageUsersCommandDataActionUpdate,
			UserId:      generated.NewOptUUID(generated.UUID(userID)),
			Assignments: []generated.IamRoleAssignment{},
		},
	}
	updateParams := generated.IamManageUsersParams{
		IdempotencyKey: "api-update-1",
		IfMatch:        generated.NewOptString(`"1"`),
	}
	updated, err := handler.IamManageUsers(ctx, update, updateParams)
	if err != nil {
		t.Fatal(err)
	}
	updatedResult, ok := updated.(*generated.EstablishedResult)
	if !ok || updatedResult.AggregateVersion != 2 {
		t.Fatalf("update response = %#v, want established version 2", updated)
	}
}

func TestIdentityHandlerMapsDenialAndVersionConflict(t *testing.T) {
	service, actor := testIdentityHTTPService(t, false)
	handler := IdentityHandler{Service: service}
	request := &generated.IamManageUsersCommandRequest{
		CommandId: generated.UUID(uuid.New()),
		Data: generated.IamManageUsersCommandData{
			Action: generated.IamManageUsersCommandDataActionCreate,
			AuthenticationSubject: generated.NewOptIamAuthenticationSubject(generated.IamAuthenticationSubject{
				Oid: "oid", Tid: "tenant", Sub: "subject",
			}),
		},
	}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	response, err := handler.IamManageUsers(ctx, request, generated.IamManageUsersParams{IdempotencyKey: "api-denied-1"})
	if err != nil {
		t.Fatal(err)
	}
	denied, ok := response.(*generated.IamManageUsersForbidden)
	if !ok || denied.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denial response = %#v, want typed AUTHORIZATION_DENIED", response)
	}

	service, actor = testIdentityHTTPService(t, true)
	handler = IdentityHandler{Service: service}
	ctx, err = identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	created, err := handler.IamManageUsers(ctx, request, generated.IamManageUsersParams{IdempotencyKey: "api-conflict-create"})
	if err != nil {
		t.Fatal(err)
	}
	createdResult := created.(*generated.EstablishedResult)
	activationRequest := &generated.IamManageUsersCommandRequest{
		CommandId:       generated.UUID(uuid.New()),
		ExpectedVersion: generated.NewOptInt(1),
		Data: generated.IamManageUsersCommandData{
			Action: generated.IamManageUsersCommandDataActionActivate,
			UserId: generated.NewOptUUID(createdResult.AggregateId),
		},
	}
	if _, err := handler.IamManageUsers(ctx, activationRequest, generated.IamManageUsersParams{
		IdempotencyKey: "api-conflict-activate", IfMatch: generated.NewOptString(`"1"`),
	}); err != nil {
		t.Fatal(err)
	}
	conflictRequest := &generated.IamManageUsersCommandRequest{
		CommandId:       generated.UUID(uuid.New()),
		ExpectedVersion: generated.NewOptInt(1),
		Data: generated.IamManageUsersCommandData{
			Action:      generated.IamManageUsersCommandDataActionUpdate,
			UserId:      generated.NewOptUUID(createdResult.AggregateId),
			Assignments: []generated.IamRoleAssignment{},
		},
	}
	conflict, err := handler.IamManageUsers(ctx, conflictRequest, generated.IamManageUsersParams{
		IdempotencyKey: "api-conflict-update", IfMatch: generated.NewOptString(`"1"`),
	})
	if err != nil {
		t.Fatal(err)
	}
	versionConflict, ok := conflict.(*generated.IamManageUsersConflict)
	if !ok || versionConflict.Code != "VERSION_CONFLICT" {
		t.Fatalf("version conflict response = %#v, want typed VERSION_CONFLICT", conflict)
	}
}

func TestIdentityHandlerMapsScopedPolicyOutcomesWithoutMutation(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		outcome identity.AuthorizationOutcome
		status  int
		code    string
	}{
		{name: "expired", outcome: identity.AuthorizationExpired, status: http.StatusForbidden, code: "POLICY_EXPIRED"},
		{name: "stale", outcome: identity.AuthorizationStale, status: http.StatusConflict, code: "POLICY_STALE"},
		{name: "unavailable", outcome: identity.AuthorizationUnavailable, status: http.StatusServiceUnavailable, code: "POLICY_UNAVAILABLE"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			repository := identity.NewMemoryUserRepository()
			audit := &identity.MemoryAuditRecorder{}
			service, err := identity.NewUserService(repository, identity.MemoryUserAuthorizer{Decision: identity.AuthorizationDecision{
				Permission: identity.UserManagementPermission, Outcome: testCase.outcome, DecisionReference: uuid.New(), ReasonCode: string(testCase.outcome),
			}}, audit, time.Now)
			if err != nil {
				t.Fatal(err)
			}
			actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tid", Sub: "sub"}}
			ctx, err := identity.WithActor(context.Background(), actor)
			if err != nil {
				t.Fatal(err)
			}
			request := &generated.IamManageUsersCommandRequest{CommandId: generated.UUID(uuid.New()), Data: generated.IamManageUsersCommandData{
				Action:                generated.IamManageUsersCommandDataActionCreate,
				AuthenticationSubject: generated.NewOptIamAuthenticationSubject(generated.IamAuthenticationSubject{Oid: "oid", Tid: "tid", Sub: "subject"}),
			}}
			response, err := (IdentityHandler{Service: service}).IamManageUsers(ctx, request, generated.IamManageUsersParams{IdempotencyKey: "outcome-" + testCase.name})
			if err != nil {
				t.Fatal(err)
			}
			body, err := json.Marshal(response)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Contains(body, []byte(testCase.code)) || !bytes.Contains(body, []byte(fmt.Sprintf("\"status\":%d", testCase.status))) {
				t.Fatalf("response body = %s, want status %d and safe code %s", body, testCase.status, testCase.code)
			}
			users, err := repository.List(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			if len(users) != 0 || len(audit.Records) != 0 {
				t.Fatalf("denied outcome mutated state: users=%d audit=%d", len(users), len(audit.Records))
			}
		})
	}
}

func TestIdentityGeneratedServerRequiresIdempotencyHeader(t *testing.T) {
	service, actor := testIdentityHTTPService(t, true)
	server, err := generated.NewServer(IdentityHandler{Service: service}, testIdentitySecurity{actor: actor})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]any{
		"commandId": uuid.New().String(),
		"data": map[string]any{
			"action":                "create",
			"authenticationSubject": map[string]string{"oid": "oid", "tid": "tenant", "sub": "subject"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/identity-access/actions/manage-users", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer fixture")
	request.Header.Set("Content-Type", "application/json")
	withoutKey := httptest.NewRecorder()
	server.ServeHTTP(withoutKey, request)
	if withoutKey.Code != http.StatusBadRequest {
		t.Fatalf("missing idempotency status = %d, want 400", withoutKey.Code)
	}

	request = httptest.NewRequest(http.MethodPost, "/identity-access/actions/manage-users", bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer fixture")
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "generated-server-create")
	withKey := httptest.NewRecorder()
	server.ServeHTTP(withKey, request)
	if withKey.Code != http.StatusOK {
		t.Fatalf("typed command status = %d, want 200: %s", withKey.Code, withKey.Body.String())
	}
}

type testIdentitySecurity struct {
	actor identity.ApplicationActor
}

func (security testIdentitySecurity) HandleBearerAuth(ctx context.Context, _ generated.OperationName, _ generated.BearerAuth) (context.Context, error) {
	return identity.WithActor(ctx, security.actor)
}

func testIdentityHTTPService(t *testing.T, allowed bool) (*identity.UserService, identity.ApplicationActor) {
	t.Helper()
	repository := identity.NewMemoryUserRepository()
	audit := &identity.MemoryAuditRecorder{}
	decision := identity.AuthorizationDecision{
		Allowed: allowed, Permission: identity.UserManagementPermission,
		ApprovedScopeIDs: []string{"*"}, PolicyReference: "policy-v1", DecisionReference: uuid.New(),
	}
	service, err := identity.NewUserService(repository, identity.MemoryUserAuthorizer{Decision: decision}, audit, func() time.Time {
		return time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "admin-oid", TID: "tenant", Sub: "admin-sub"}}
	return service, actor
}

type testRoleApproval struct{}

func (testRoleApproval) ValidateRoleApproval(context.Context, identity.ApplicationActor, identity.RoleCommand, *identity.Role, string) error {
	return nil
}

func testRoleHTTPService(t *testing.T, allowed bool) (*identity.RoleService, identity.ApplicationActor) {
	t.Helper()
	repository := identity.NewMemoryRoleRepository()
	service, err := identity.NewRoleService(
		repository,
		identity.MemoryRoleAuthorizer{Decision: identity.AuthorizationDecision{
			Allowed: allowed, Permission: identity.RoleManagementPermission,
			ApprovedScopeIDs: []string{"*"}, PolicyReference: "role-policy-v1", DecisionReference: uuid.New(),
		}},
		testRoleApproval{},
		identity.AllowAllRoleSegregationPort{},
		&identity.MemoryRoleAuditRecorder{},
		func() time.Time { return time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "role-admin-oid", TID: "tenant", Sub: "role-admin-sub"}}
	return service, actor
}

func testRoleRequest(action generated.IamManageRolesCommandDataAction, roleID generated.OptUUID, name string, scope string) *generated.IamManageRolesCommandRequest {
	return &generated.IamManageRolesCommandRequest{
		CommandId: generated.UUID(uuid.New()),
		Data: generated.IamManageRolesCommandData{
			Action: action, RoleId: roleID, Name: name,
			Grants: []generated.IamPermissionGrant{{
				Permission: "finance.iam.manage.roles", ScopeIds: []string{scope},
				EffectiveFrom: time.Date(2026, 9, 25, 0, 0, 0, 0, time.UTC),
			}},
			Approval: generated.IamApprovalDecisionReference{
				ApprovalRequestId: generated.UUID(uuid.New()), DecisionId: generated.UUID(uuid.New()),
				PolicyVersion: "role-policy-v1", DecisionVersion: 1, SubjectVersion: 1,
				CandidateFingerprint: "test-candidate", ApproverUserId: generated.UUID(uuid.New()),
			},
		},
	}
}

func TestIdentityHandlerManagesTypedRolesAndMapsSafeOutcomes(t *testing.T) {
	service, actor := testRoleHTTPService(t, true)
	handler := IdentityHandler{RoleService: service}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}

	createRequest := testRoleRequest(generated.IamManageRolesCommandDataActionCreate, generated.OptUUID{}, "Scoped role", "entity-1")
	created, err := handler.IamManageRoles(ctx, createRequest, generated.IamManageRolesParams{IdempotencyKey: "role-http-create"})
	if err != nil {
		t.Fatal(err)
	}
	createdResult, ok := created.(*generated.EstablishedResult)
	if !ok || createdResult.AggregateVersion != 1 {
		t.Fatalf("create response = %#v, want typed version 1 result", created)
	}

	updateRequest := testRoleRequest(
		generated.IamManageRolesCommandDataActionUpdate,
		generated.NewOptUUID(createdResult.AggregateId),
		"Scoped role v2", "entity-1",
	)
	updateRequest.ExpectedVersion = generated.NewOptInt(1)
	updateRequest.Data.Approval.SubjectVersion = 2
	updated, err := handler.IamManageRoles(ctx, updateRequest, generated.IamManageRolesParams{
		IdempotencyKey: "role-http-update", IfMatch: generated.NewOptString("\"1\""),
	})
	if err != nil {
		t.Fatal(err)
	}
	updatedResult, ok := updated.(*generated.EstablishedResult)
	if !ok || updatedResult.AggregateVersion != 2 {
		t.Fatalf("update response = %#v, want typed version 2 result", updated)
	}

	conflictRequest := testRoleRequest(
		generated.IamManageRolesCommandDataActionUpdate,
		generated.NewOptUUID(createdResult.AggregateId),
		"Scoped role conflict", "entity-1",
	)
	conflictRequest.ExpectedVersion = generated.NewOptInt(1)
	conflictRequest.Data.Approval.SubjectVersion = 2
	conflict, err := handler.IamManageRoles(ctx, conflictRequest, generated.IamManageRolesParams{
		IdempotencyKey: "role-http-conflict", IfMatch: generated.NewOptString("\"2\""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if response, ok := conflict.(*generated.IamManageRolesConflict); !ok || response.Code != "VERSION_CONFLICT" {
		t.Fatalf("version mismatch response = %#v, want typed conflict", conflict)
	}

	deniedService, deniedActor := testRoleHTTPService(t, false)
	deniedContext, err := identity.WithActor(context.Background(), deniedActor)
	if err != nil {
		t.Fatal(err)
	}
	denied, err := (IdentityHandler{RoleService: deniedService}).IamManageRoles(deniedContext, createRequest, generated.IamManageRolesParams{IdempotencyKey: "role-http-denied"})
	if err != nil {
		t.Fatal(err)
	}
	if response, ok := denied.(*generated.IamManageRolesForbidden); !ok || response.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denial response = %#v, want typed forbidden", denied)
	}

	invalidRequest := testRoleRequest(generated.IamManageRolesCommandDataActionCreate, generated.OptUUID{}, "Duplicate scope role", "entity-1")
	invalidRequest.Data.Grants[0].ScopeIds = []string{"entity-1", "entity-1"}
	invalid, err := handler.IamManageRoles(ctx, invalidRequest, generated.IamManageRolesParams{IdempotencyKey: "role-http-invalid"})
	if err != nil {
		t.Fatal(err)
	}
	if response, ok := invalid.(*generated.IamManageRolesUnprocessableEntity); !ok || response.Code != "VALIDATION_FAILED" {
		t.Fatalf("validation response = %#v, want typed validation failure", invalid)
	}
}

type testEmergencyAuthorizer struct {
	allowed bool
}

func (authorizer testEmergencyAuthorizer) AuthorizeEmergencyAccess(_ context.Context, _ identity.ApplicationActor, command identity.EmergencyAccessGrantCommand, _ *identity.EmergencyAccessGrant) (identity.AuthorizationDecision, error) {
	permission := identity.EmergencyAccessRevokePermission
	if command.Action == identity.EmergencyAccessActionGrant {
		permission = identity.EmergencyAccessGrantPermission
	}
	return identity.AuthorizationDecision{Allowed: authorizer.allowed, Permission: permission, Outcome: identity.AuthorizationAllowed, PolicyVersion: "emergency-policy-v1", PolicyReference: "emergency-policy", DecisionReference: uuid.New(), ApprovedScopeIDs: []string{"*"}}, nil
}

type testEmergencyApproval struct{}

func (testEmergencyApproval) ValidateEmergencyAccessApproval(context.Context, identity.ApplicationActor, identity.EmergencyAccessGrantCommand, *identity.EmergencyAccessGrant, string) error {
	return nil
}

func testEmergencyHTTPService(t *testing.T, allowed bool) (*identity.EmergencyAccessService, identity.ApplicationActor) {
	t.Helper()
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	repository := identity.NewMemoryEmergencyAccessRepository()
	service, err := identity.NewEmergencyAccessService(repository, testEmergencyAuthorizer{allowed: allowed}, testEmergencyApproval{}, &identity.MemoryEmergencyAccessAuditRecorder{}, identity.WeekdayEmergencyAccessCalendar{}, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	actor := identity.ApplicationActor{UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "emergency-admin-oid", TID: "tenant", Sub: "emergency-admin-sub", Assurance: identity.AuthenticationAssurance{AuthenticatedAt: now.Add(-time.Minute), AssuranceLevel: "high", Methods: "mfa"}}}
	return service, actor
}

func TestIdentityHandlerManagesEmergencyAccessAndReview(t *testing.T) {
	service, actor := testEmergencyHTTPService(t, true)
	handler := IdentityHandler{EmergencyAccessService: service}
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	grantRequest := &generated.IamGrantEmergencyAccessCommandRequest{
		CommandId: generated.UUID(uuid.New()),
		Data: generated.IamGrantEmergencyAccessCommandData{
			TargetActorId: generated.UUID(uuid.New()), Permissions: []string{"finance.gl.submit.posting.request"}, ScopeIds: []string{"entity-1"},
			ReasonCode: "break-fix", StartsAt: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC),
			Approval: generated.IamApprovalDecisionReference{ApprovalRequestId: generated.UUID(uuid.New()), DecisionId: generated.UUID(uuid.New()), PolicyVersion: "emergency-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "test", ApproverUserId: generated.UUID(uuid.New())},
		},
	}
	created, err := handler.IamGrantEmergencyAccess(ctx, grantRequest, generated.IamGrantEmergencyAccessParams{IdempotencyKey: "emergency-grant-http"})
	if err != nil {
		t.Fatal(err)
	}
	createdResult, ok := created.(*generated.EstablishedResult)
	if !ok || createdResult.AggregateVersion != 1 {
		t.Fatalf("grant result = %#v, want established version 1", created)
	}

	revokeRequest := &generated.IamRevokeEmergencyAccessCommandRequest{CommandId: generated.UUID(uuid.New()), ExpectedVersion: 1, Data: generated.IamRevokeEmergencyAccessCommandData{GrantId: createdResult.AggregateId, ReasonCode: "incident-contained", ReviewStatus: generated.NewOptIamRevokeEmergencyAccessCommandDataReviewStatus(generated.IamRevokeEmergencyAccessCommandDataReviewStatusPending)}}
	revoked, err := handler.IamRevokeEmergencyAccess(ctx, revokeRequest, generated.IamRevokeEmergencyAccessParams{IdempotencyKey: "emergency-revoke-http", IfMatch: generated.NewOptString("\"1\"")})
	if err != nil {
		t.Fatal(err)
	}
	revokedResult, ok := revoked.(*generated.EstablishedResult)
	if !ok || revokedResult.AggregateVersion != 2 || string(revokedResult.Data["status"]) != `"revoked"` {
		t.Fatalf("revoke result = %#v, want revoked version 2", revoked)
	}

	completeRequest := &generated.IamRevokeEmergencyAccessCommandRequest{CommandId: generated.UUID(uuid.New()), ExpectedVersion: 2, Data: generated.IamRevokeEmergencyAccessCommandData{GrantId: createdResult.AggregateId, ReasonCode: "post-use-review", ReviewStatus: generated.NewOptIamRevokeEmergencyAccessCommandDataReviewStatus(generated.IamRevokeEmergencyAccessCommandDataReviewStatusCompleted), ReviewOutcomeCode: generated.NewOptString("review-code-001"), ReviewReference: generated.NewOptString("review-http-1")}}
	completed, err := handler.IamRevokeEmergencyAccess(ctx, completeRequest, generated.IamRevokeEmergencyAccessParams{IdempotencyKey: "emergency-review-http", IfMatch: generated.NewOptString("\"2\"")})
	if err != nil {
		t.Fatal(err)
	}
	if result, ok := completed.(*generated.EstablishedResult); !ok || result.AggregateVersion != 3 || string(result.Data["reviewStatus"]) != `"completed"` {
		t.Fatalf("review result = %#v, want completed version 3", completed)
	}
}

func TestIdentityHandlerMapsEmergencyStepUpAndDenial(t *testing.T) {
	service, actor := testEmergencyHTTPService(t, false)
	actor.Subject.Assurance.AuthenticatedAt = time.Date(2026, 9, 25, 7, 0, 0, 0, time.UTC)
	ctx, err := identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	request := &generated.IamGrantEmergencyAccessCommandRequest{CommandId: generated.UUID(uuid.New()), Data: generated.IamGrantEmergencyAccessCommandData{TargetActorId: generated.UUID(uuid.New()), Permissions: []string{"finance.gl.submit.posting.request"}, ScopeIds: []string{"entity-1"}, ReasonCode: "break-fix", StartsAt: time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC), ExpiresAt: time.Date(2026, 9, 25, 10, 0, 0, 0, time.UTC), Approval: generated.IamApprovalDecisionReference{ApprovalRequestId: generated.UUID(uuid.New()), DecisionId: generated.UUID(uuid.New()), PolicyVersion: "emergency-policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "test", ApproverUserId: generated.UUID(uuid.New())}}}
	response, err := (IdentityHandler{EmergencyAccessService: service}).IamGrantEmergencyAccess(ctx, request, generated.IamGrantEmergencyAccessParams{IdempotencyKey: "emergency-step-up"})
	if err != nil {
		t.Fatal(err)
	}
	if forbidden, ok := response.(*generated.IamGrantEmergencyAccessForbidden); !ok || forbidden.Code != "STEP_UP_REQUIRED" {
		t.Fatalf("step-up response = %#v, want typed STEP_UP_REQUIRED", response)
	}

	actor.Subject.Assurance.AuthenticatedAt = time.Date(2026, 9, 25, 7, 59, 0, 0, time.UTC)
	ctx, err = identity.WithActor(context.Background(), actor)
	if err != nil {
		t.Fatal(err)
	}
	response, err = (IdentityHandler{EmergencyAccessService: service}).IamGrantEmergencyAccess(ctx, request, generated.IamGrantEmergencyAccessParams{IdempotencyKey: "emergency-denied"})
	if err != nil {
		t.Fatal(err)
	}
	if forbidden, ok := response.(*generated.IamGrantEmergencyAccessForbidden); !ok || forbidden.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denial response = %#v, want typed AUTHORIZATION_DENIED", response)
	}
}
