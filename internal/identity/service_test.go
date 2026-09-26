package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestUserServiceCreateIsScopedIdempotentAndAuditedOnce(t *testing.T) {
	service, repository, audit := testUserService(t, []string{"entity-1"})
	actor := testActor(t)
	roleID := uuid.New()
	command := UserCommand{
		Action:                UserActionCreate,
		AuthenticationSubject: &AuthenticationSubject{OID: "new-oid", TID: "tenant", Sub: "new-sub"},
		Assignments:           []RoleAssignment{{RoleID: roleID, Scopes: []EntityAccessScope{{ScopeID: "entity-1"}}}},
		IdempotencyKey:        "create-1",
		CorrelationID:         "corr-1",
		CausationID:           "cause-1",
	}

	first, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.User.Status != UserStatusInactive {
		t.Fatalf("created status = %q, want inactive", first.User.Status)
	}
	if first.User.Version.Value() != 1 {
		t.Fatalf("created version = %d, want 1", first.User.Version.Value())
	}
	replay, err := service.Execute(context.Background(), actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed {
		t.Fatal("repeat command was not marked as replayed")
	}
	if len(audit.Records) != 1 {
		t.Fatalf("audit records = %d, want 1", len(audit.Records))
	}
	users, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 1 {
		t.Fatalf("users = %d, want 1", len(users))
	}

	command.Assignments[0].Scopes[0].ScopeID = "entity-2"
	if !errors.Is(func() error {
		_, err := service.Execute(context.Background(), actor, command)
		return err
	}(), ErrIdempotencyConflict) {
		t.Fatal("changed retry did not return ErrIdempotencyConflict")
	}
}

func TestUserServiceDeniesOutsideScopeWithoutSideEffects(t *testing.T) {
	service, repository, audit := testUserService(t, []string{"entity-1"})
	_, err := service.Execute(context.Background(), testActor(t), UserCommand{
		Action:                UserActionCreate,
		AuthenticationSubject: &AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"},
		Assignments:           []RoleAssignment{{RoleID: uuid.New(), Scopes: []EntityAccessScope{{ScopeID: "entity-2"}}}},
		IdempotencyKey:        "denied-1",
	})
	if !errors.Is(err, ErrAuthorizationDenied) {
		t.Fatalf("error = %v, want ErrAuthorizationDenied", err)
	}
	users, listErr := repository.List(context.Background())
	if listErr != nil {
		t.Fatal(listErr)
	}
	if len(users) != 0 {
		t.Fatalf("users = %d after denial, want 0", len(users))
	}
	if len(audit.Records) != 0 {
		t.Fatalf("audit records = %d after denial, want 0", len(audit.Records))
	}
}

func TestUserServiceRejectsStaleVersionAndReplacesAssignmentsAtomically(t *testing.T) {
	service, _, _ := testUserService(t, []string{"entity-1", "entity-2"})
	actor := testActor(t)
	created, err := service.Execute(context.Background(), actor, UserCommand{
		Action:                UserActionCreate,
		AuthenticationSubject: &AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"},
		Assignments:           []RoleAssignment{{RoleID: uuid.New(), Scopes: []EntityAccessScope{{ScopeID: "entity-1"}}}},
		IdempotencyKey:        "create-2",
	})
	if err != nil {
		t.Fatal(err)
	}
	newRole := uuid.New()
	updated, err := service.Execute(context.Background(), actor, UserCommand{
		Action:          UserActionUpdate,
		UserID:          created.User.ID,
		ExpectedVersion: versionForTest(1),
		Assignments:     []RoleAssignment{{RoleID: newRole, Scopes: []EntityAccessScope{{ScopeID: "entity-2"}}}},
		IdempotencyKey:  "update-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.User.Version.Value() != 2 || len(updated.User.Assignments) != 1 || updated.User.Assignments[0].RoleID != newRole {
		t.Fatalf("replacement result = %#v", updated.User)
	}
	_, err = service.Execute(context.Background(), actor, UserCommand{
		Action:          UserActionUpdate,
		UserID:          created.User.ID,
		ExpectedVersion: versionForTest(1),
		Assignments:     []RoleAssignment{{RoleID: uuid.New(), Scopes: []EntityAccessScope{{ScopeID: "entity-1"}}}},
		IdempotencyKey:  "update-stale",
	})
	if !errors.Is(err, ErrVersionConflict) {
		t.Fatalf("stale update error = %v, want ErrVersionConflict", err)
	}
}

func TestUserServiceLifecycleRevokesSuspendedAndTerminatedUsers(t *testing.T) {
	service, repository, _ := testUserService(t, []string{"entity-1"})
	actor := testActor(t)
	created, err := service.Execute(context.Background(), actor, UserCommand{
		Action:                UserActionCreate,
		AuthenticationSubject: &AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"},
		Assignments:           []RoleAssignment{{RoleID: uuid.New(), Scopes: []EntityAccessScope{{ScopeID: "entity-1"}}}},
		IdempotencyKey:        "create-3",
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := service.Execute(context.Background(), actor, UserCommand{
		Action: UserActionActivate, UserID: created.User.ID, ExpectedVersion: versionForTest(1), IdempotencyKey: "activate-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.User.Status != UserStatusActive {
		t.Fatalf("status after activation = %q", active.User.Status)
	}
	suspended, err := service.Execute(context.Background(), actor, UserCommand{
		Action: UserActionSuspend, UserID: created.User.ID, ExpectedVersion: versionForTest(2), IdempotencyKey: "suspend-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if suspended.User.Status != UserStatusSuspended {
		t.Fatalf("status after suspension = %q", suspended.User.Status)
	}
	terminated, err := service.Execute(context.Background(), actor, UserCommand{
		Action: UserActionTerminate, UserID: created.User.ID, ExpectedVersion: versionForTest(3), IdempotencyKey: "terminate-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if terminated.User.Status != UserStatusTerminated {
		t.Fatalf("status after termination = %q", terminated.User.Status)
	}
	stored, err := repository.Get(context.Background(), created.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != UserStatusTerminated {
		t.Fatalf("stored status = %q, want terminated", stored.Status)
	}
}

func TestUserServiceKeepsAuthenticationSubjectImmutable(t *testing.T) {
	service, _, _ := testUserService(t, []string{"entity-1"})
	created, err := service.Execute(context.Background(), testActor(t), UserCommand{
		Action:                UserActionCreate,
		AuthenticationSubject: &AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"},
		IdempotencyKey:        "create-4",
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Execute(context.Background(), testActor(t), UserCommand{
		Action:                UserActionUpdate,
		UserID:                created.User.ID,
		ExpectedVersion:       versionForTest(1),
		AuthenticationSubject: &AuthenticationSubject{OID: "other", TID: "tenant", Sub: "other"},
		IdempotencyKey:        "update-subject",
	})
	if !errors.Is(err, ErrAuthenticationImmutable) {
		t.Fatalf("subject update error = %v, want ErrAuthenticationImmutable", err)
	}
}

func TestUserServiceAuditFailureHasNoSideEffects(t *testing.T) {
	repository := NewMemoryUserRepository()
	audit := &MemoryAuditRecorder{Err: errors.New("audit unavailable")}
	service, err := NewUserService(repository, MemoryUserAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: UserManagementPermission, ApprovedScopeIDs: []string{"*"},
		PolicyReference: "policy-v1", DecisionReference: uuid.New(),
	}}, audit, time.Now)
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.Execute(context.Background(), testActor(t), UserCommand{
		Action:                UserActionCreate,
		AuthenticationSubject: &AuthenticationSubject{OID: "audit-failure-oid", TID: "tenant", Sub: "audit-failure-sub"},
		IdempotencyKey:        "audit-failure-1",
	})
	if err == nil {
		t.Fatal("audit failure returned nil error")
	}
	users, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != 0 {
		t.Fatalf("users after audit failure = %d, want 0", len(users))
	}
}

func testUserService(t *testing.T, scopes []string) (*UserService, *MemoryUserRepository, *MemoryAuditRecorder) {
	t.Helper()
	repository := NewMemoryUserRepository()
	audit := &MemoryAuditRecorder{}
	authorizer := MemoryUserAuthorizer{Decision: AuthorizationDecision{
		Allowed: true, Permission: UserManagementPermission, ApprovedScopeIDs: scopes,
		PolicyReference: "policy-v1", DecisionReference: uuid.New(),
	}}
	service, err := NewUserService(repository, authorizer, audit, func() time.Time {
		return time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	})
	if err != nil {
		t.Fatal(err)
	}
	return service, repository, audit
}

func TestMemoryRepositoryReplaceRejectsAuthenticationSubjectChange(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	subject := AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}
	user, err := NewUser(uuid.New(), subject, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	repository := NewMemoryUserRepository()
	if err := repository.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	changed := user
	changed.AuthenticationSubject = AuthenticationSubject{OID: "other", TID: "tenant", Sub: "other"}
	changed.Version = aggregateversion.AggregateVersion(2)
	if err := repository.Replace(context.Background(), changed, aggregateversion.Initial()); !errors.Is(err, ErrAuthenticationImmutable) {
		t.Fatalf("subject replacement error = %v, want ErrAuthenticationImmutable", err)
	}
	audit := &MemoryAuditRecorder{}
	repository.BindAuditRecorder(audit)
	expected := aggregateversion.Initial()
	if err := repository.CommitUserMutation(context.Background(), UserMutation{Before: user, After: changed, ExpectedVersion: &expected, Audit: AuditRecord{UserID: user.ID, Action: UserActionUpdate, DecisionReference: uuid.New()}}); !errors.Is(err, ErrAuthenticationImmutable) {
		t.Fatalf("atomic subject replacement error = %v, want ErrAuthenticationImmutable", err)
	}
}

func testActor(t *testing.T) ApplicationActor {
	t.Helper()
	return ApplicationActor{
		UserID:  uuid.New(),
		Subject: AuthenticationSubject{OID: "admin-oid", TID: "tenant", Sub: "admin-sub"},
	}
}

func versionForTest(value int64) *aggregateversion.AggregateVersion {
	version, err := aggregateversion.FromInt64(value)
	if err != nil {
		panic(err)
	}
	return &version
}
