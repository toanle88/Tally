//go:build integration

package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/idempotency"
)

func TestIdentityUserRepositoryPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	auditWriter := identity.PostgresAuditWriter(func(context.Context, pgx.Tx, identity.AuditRecord) (uuid.UUID, error) {
		return uuid.New(), nil
	})
	repository, err := identity.NewPostgresUserRepositoryWithAudit(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	commitMutation := func(before identity.User, after identity.User, expected *aggregateversion.AggregateVersion) error {
		return repository.CommitUserMutation(ctx, identity.UserMutation{
			Before: before, After: after, ExpectedVersion: expected,
			Audit: identity.AuditRecord{UserID: after.ID, Action: "test.identity-mutation", DecisionReference: uuid.New()},
		})
	}

	t.Run("identity schema owns user state", func(t *testing.T) {
		var identityUserTable bool
		if err := fixture.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'identity' AND table_name = 'user_account'
			)
		`).Scan(&identityUserTable); err != nil {
			t.Fatal(err)
		}
		if !identityUserTable {
			t.Fatal("identity.user_account was not created")
		}

		var platformUserTable bool
		if err := fixture.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'platform' AND table_name = 'user_account'
			)
		`).Scan(&platformUserTable); err != nil {
			t.Fatal(err)
		}
		if platformUserTable {
			t.Fatal("user state leaked into the platform schema")
		}
	})

	roleID := uuid.New()
	user, err := identity.NewUser(
		uuid.New(),
		identity.AuthenticationSubject{OID: "oid-1", TID: "tenant-1", Sub: "subject-1"},
		[]identity.RoleAssignment{{RoleID: roleID, Scopes: []identity.EntityAccessScope{{ScopeID: "entity-1"}}}},
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := commitMutation(identity.User{}, user, nil); err != nil {
		t.Fatal(err)
	}

	t.Run("subject uniqueness is enforced", func(t *testing.T) {
		duplicate := user
		duplicate.ID = uuid.New()
		if err := commitMutation(identity.User{}, duplicate, nil); !errors.Is(err, identity.ErrDuplicateAuthenticationSubject) {
			t.Fatalf("duplicate create error = %v, want ErrDuplicateAuthenticationSubject", err)
		}
	})

	t.Run("assignment replacement is atomic", func(t *testing.T) {
		stored, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		before := stored
		stored.Assignments = []identity.RoleAssignment{{RoleID: uuid.New(), Scopes: []identity.EntityAccessScope{{ScopeID: "entity-2"}}}}
		stored.UpdatedAt = stored.UpdatedAt.Add(time.Minute)
		nextVersion, err := stored.Version.Advance()
		if err != nil {
			t.Fatal(err)
		}
		stored.Version = nextVersion
		if err := commitMutation(before, stored, func() *aggregateversion.AggregateVersion { expected := before.Version; return &expected }()); err != nil {
			t.Fatal(err)
		}

		reloaded, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if reloaded.Version.Value() != 2 || len(reloaded.Assignments) != 1 || reloaded.Assignments[0].Scopes[0].ScopeID != "entity-2" {
			t.Fatalf("reloaded replacement = %#v, want one entity-2 assignment at version 2", reloaded)
		}

		invalid := reloaded
		invalid.Assignments = []identity.RoleAssignment{{RoleID: uuid.Nil, Scopes: []identity.EntityAccessScope{{ScopeID: "entity-3"}}}}
		invalid.Version = aggregateversion.AggregateVersion(3)
		if err := commitMutation(reloaded, invalid, func() *aggregateversion.AggregateVersion { expected := reloaded.Version; return &expected }()); !errors.Is(err, identity.ErrInvalidAssignment) {
			t.Fatalf("invalid replacement error = %v, want ErrInvalidAssignment", err)
		}
		unchanged, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if unchanged.Version.Value() != 2 || unchanged.Assignments[0].Scopes[0].ScopeID != "entity-2" {
			t.Fatalf("invalid replacement changed state: %#v", unchanged)
		}
	})

	t.Run("optimistic locking permits one concurrent update", func(t *testing.T) {
		base, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		left := base
		left.Status = identity.UserStatusActive
		left.UpdatedAt = left.UpdatedAt.Add(time.Minute)
		left.Version = aggregateversion.AggregateVersion(3)
		right := base
		right.Status = identity.UserStatusSuspended
		right.UpdatedAt = right.UpdatedAt.Add(2 * time.Minute)
		right.Version = aggregateversion.AggregateVersion(3)

		start := make(chan struct{})
		results := make(chan error, 2)
		var wait sync.WaitGroup
		for _, candidate := range []identity.User{left, right} {
			candidate := candidate
			wait.Add(1)
			go func() {
				defer wait.Done()
				<-start
				expected := base.Version
				results <- commitMutation(base, candidate, &expected)
			}()
		}
		close(start)
		wait.Wait()
		close(results)

		successes := 0
		conflicts := 0
		for result := range results {
			switch {
			case result == nil:
				successes++
			case errors.Is(result, identity.ErrVersionConflict):
				conflicts++
			default:
				t.Fatalf("concurrent replacement error = %v", result)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("concurrent updates: successes=%d conflicts=%d, want 1 each", successes, conflicts)
		}
		final, err := repository.Get(ctx, user.ID)
		if err != nil {
			t.Fatal(err)
		}
		if final.Version.Value() != 3 {
			t.Fatalf("final version = %d, want 3", final.Version.Value())
		}
		if final.Status != identity.UserStatusActive && final.Status != identity.UserStatusSuspended {
			t.Fatalf("final status = %q, want one successful concurrent update", final.Status)
		}
	})

	assertNoAcquiredConnections(t, fixture.pool)
}

func TestIdentityUserServiceDurableIdempotencyReplaysAfterRestart(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	auditReference := uuid.New()
	auditWriter := identity.PostgresAuditWriter(func(ctx context.Context, tx pgx.Tx, _ identity.AuditRecord) (uuid.UUID, error) {
		if _, err := tx.Exec(ctx, "SELECT 1"); err != nil {
			return uuid.Nil, err
		}
		return auditReference, nil
	})
	repository, err := identity.NewPostgresUserRepositoryWithAudit(fixture.pool, auditWriter)
	if err != nil {
		t.Fatal(err)
	}
	policy := idempotency.IdempotencyPolicy{RecordTTL: time.Hour, LeaseTTL: time.Minute}
	actor := identity.ApplicationActor{
		UserID:  uuid.New(),
		Subject: identity.AuthenticationSubject{OID: "durable-admin", TID: "tenant", Sub: "durable-admin"},
	}
	authorizer := identity.MemoryUserAuthorizer{Decision: identity.AuthorizationDecision{
		Allowed: true, Permission: identity.UserManagementPermission,
		ApprovedScopeIDs: []string{"*"}, PolicyReference: "policy-v1", DecisionReference: uuid.New(),
	}}
	command := identity.UserCommand{
		Action:                identity.UserActionCreate,
		AuthenticationSubject: &identity.AuthenticationSubject{OID: "durable-oid", TID: "tenant", Sub: "durable-sub"},
		Assignments:           []identity.RoleAssignment{{RoleID: uuid.New(), Scopes: []identity.EntityAccessScope{{ScopeID: "entity-1"}}}},
		IdempotencyKey:        "durable-create-1",
	}

	newService := func() *identity.UserService {
		service, err := identity.NewUserServiceWithDurableIdempotency(
			repository, authorizer, &identity.MemoryAuditRecorder{}, time.Now,
			identity.DurableUserServiceConfig{
				Database: fixture.pool, Coordinator: idempotency.NewPostgresCoordinator(),
				Policy: policy, OperationID: "identity.manage-users.v1",
			},
			identity.AllowAllRolePolicyLookup{},
		)
		if err != nil {
			t.Fatal(err)
		}
		return service
	}

	first, err := newService().Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if first.Replayed {
		t.Fatal("first durable command was replayed")
	}
	var (
		storedState string
		storedBody  []byte
		storedAudit uuid.UUID
	)
	scopeKey := fmt.Sprintf("{\"module\":\"identity\",\"actorId\":\"%s\"}", actor.UserID)
	if err := fixture.pool.QueryRow(ctx,
		"SELECT state, result_body FROM platform.idempotency_record WHERE scope_key = $1 AND idempotency_key = $2",
		scopeKey, command.IdempotencyKey,
	).Scan(&storedState, &storedBody); err != nil {
		t.Fatal(err)
	}
	if storedState != string(idempotency.StateEstablished) || len(storedBody) == 0 {
		t.Fatalf("durable result state=%q body=%d, want established result", storedState, len(storedBody))
	}
	if err := fixture.pool.QueryRow(ctx,
		"SELECT last_audit_reference FROM identity.user_account WHERE id = $1",
		first.User.ID,
	).Scan(&storedAudit); err != nil {
		t.Fatal(err)
	}
	if storedAudit != auditReference {
		t.Fatalf("last audit reference = %s, want %s", storedAudit, auditReference)
	}

	replayed, err := newService().Execute(ctx, actor, command)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.User.ID != first.User.ID {
		t.Fatalf("replay = %#v, want same user with Replayed=true", replayed)
	}

	command.Assignments[0].Scopes[0].ScopeID = "entity-2"
	if _, err := newService().Execute(ctx, actor, command); !errors.Is(err, identity.ErrIdempotencyConflict) {
		t.Fatalf("changed durable retry error = %v, want ErrIdempotencyConflict", err)
	}

	deniedService, err := identity.NewUserServiceWithDurableIdempotency(
		repository, identity.MemoryUserAuthorizer{Decision: identity.AuthorizationDecision{
			Allowed: false, Permission: identity.UserManagementPermission, DecisionReference: uuid.New(),
		}}, &identity.MemoryAuditRecorder{}, time.Now,
		identity.DurableUserServiceConfig{
			Database: fixture.pool, Coordinator: idempotency.NewPostgresCoordinator(),
			Policy: policy, OperationID: "identity.manage-users.v1",
		},
		identity.AllowAllRolePolicyLookup{},
	)
	if err != nil {
		t.Fatal(err)
	}
	deniedCommand := command
	deniedCommand.AuthenticationSubject = &identity.AuthenticationSubject{OID: "durable-denied-oid", TID: "tenant", Sub: "durable-denied-sub"}
	deniedCommand.IdempotencyKey = "durable-denied-1"
	if _, err := deniedService.Execute(ctx, actor, deniedCommand); !errors.Is(err, identity.ErrAuthorizationDenied) {
		t.Fatalf("denied durable command error = %v, want ErrAuthorizationDenied", err)
	}
	if err := fixture.pool.QueryRow(ctx,
		"SELECT state FROM platform.idempotency_record WHERE scope_key = $1 AND idempotency_key = $2",
		scopeKey, deniedCommand.IdempotencyKey,
	).Scan(&storedState); err != nil {
		t.Fatal(err)
	}
	if storedState != string(idempotency.StateFailed) {
		t.Fatalf("denied durable state = %q, want failed", storedState)
	}
	if _, err := deniedService.Execute(ctx, actor, deniedCommand); !errors.Is(err, identity.ErrAuthorizationDenied) {
		t.Fatalf("denied durable retry error = %v, want ErrAuthorizationDenied", err)
	}
}
