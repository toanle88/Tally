//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
)

func TestIdentityAccessPolicyStoreCurrentVersionAndImmutableRevisions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	policyID := uuid.New()
	actorID := uuid.New()
	now := time.Now().UTC().Truncate(time.Microsecond)
	transaction, err := fixture.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = transaction.Exec(ctx, `
        INSERT INTO identity.access_policy (id, current_version, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $4)`, policyID, "v1", identity.AccessPolicyStatusActive, now)
	if err != nil {
		_ = transaction.Rollback(ctx)
		t.Fatal(err)
	}
	_, err = transaction.Exec(ctx, `
        INSERT INTO identity.access_policy_revision
            (policy_id, policy_version, status, effective_from, subject_actor_ids, subject_role_ids, permissions, rules, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		policyID, "v1", identity.AccessPolicyStatusActive, now.Add(-time.Hour),
		[]uuid.UUID{actorID}, []uuid.UUID{}, []string{"finance.test"}, []byte(`[{}]`), now)
	if err != nil {
		_ = transaction.Rollback(ctx)
		t.Fatal(err)
	}
	if err := transaction.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	insertRevision := func(version string, effectiveFrom time.Time) {
		t.Helper()
		_, err := fixture.pool.Exec(ctx, `
            INSERT INTO identity.access_policy_revision
                (policy_id, policy_version, status, effective_from, subject_actor_ids, subject_role_ids, permissions, rules, created_at)
            VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
			policyID, version, identity.AccessPolicyStatusActive, effectiveFrom,
			[]uuid.UUID{actorID}, []uuid.UUID{}, []string{"finance.test"}, []byte(`[{}]`), now)
		if err != nil {
			t.Fatal(err)
		}
	}
	store, err := identity.NewPostgresAccessPolicyStore(fixture.pool)
	if err != nil {
		t.Fatal(err)
	}
	evaluator, err := identity.NewPolicyEvaluator(store, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	decision, err := evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actorID, Permission: "finance.test"})
	if err != nil || decision.Outcome != identity.AuthorizationAllowed || decision.PolicyVersion != "v1" {
		t.Fatalf("v1 decision = %#v, err=%v", decision, err)
	}

	insertRevision("v2", now.Add(-time.Hour))
	if _, err := fixture.pool.Exec(ctx, `UPDATE identity.access_policy SET current_version = $2, updated_at = $3 WHERE id = $1`, policyID, "v2", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	decision, err = evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actorID, Permission: "finance.test", ExpectedPolicyVersion: "v1"})
	if err != nil || decision.Outcome != identity.AuthorizationStale {
		t.Fatalf("stale decision after current-version change = %#v, err=%v", decision, err)
	}

	if _, err := fixture.pool.Exec(ctx, `UPDATE identity.access_policy_revision SET rules = $3 WHERE policy_id = $1 AND policy_version = $2`, policyID, "v2", []byte(`[{}]`)); err == nil {
		t.Fatal("historical/current policy revision update succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, `
        INSERT INTO identity.access_policy_revision
            (policy_id, policy_version, status, effective_from, effective_to, subject_actor_ids, subject_role_ids, permissions, rules, created_at)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`,
		policyID, "invalid", identity.AccessPolicyStatusActive, now, now.Add(-time.Minute), []uuid.UUID{}, []uuid.UUID{}, []string{"finance.test"}, []byte(`[{}]`), now); err == nil {
		t.Fatal("invalid effective interval succeeded")
	} else if errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}
