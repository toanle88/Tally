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
			Action: generated.IamManageUsersCommandDataActionUpdate,
			UserId: generated.NewOptUUID(generated.UUID(userID)),
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
			Action: generated.IamManageUsersCommandDataActionUpdate,
			UserId: generated.NewOptUUID(createdResult.AggregateId),
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
