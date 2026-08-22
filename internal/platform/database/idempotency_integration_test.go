//go:build integration

package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
	"github.com/toanle88/Tally/internal/platform/idempotency"
)

func TestDurableIdempotencyReservationAndFinalization(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scope, err := accountingscope.New(uuid.New(), uuid.New(), uuid.New(), uuid.New(), "USD")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := idempotency.NewIdentity(scope, "durable-payment-1")
	if err != nil {
		t.Fatal(err)
	}
	coordinator := idempotency.NewPostgresCoordinator()
	policy := idempotency.IdempotencyPolicy{RecordTTL: time.Hour, LeaseTTL: time.Minute}
	first, err := coordinator.Acquire(ctx, fixture.pool, identity, "sha256:durable-payment", "payments.submit", policy)
	if err != nil || first.Decision() != idempotency.DecisionExecute {
		t.Fatalf("first acquisition = %#v, %v", first, err)
	}

	status := 200
	result, err := idempotency.NewCommandResultMetadata(identity, "sha256:durable-payment", "payments.submit", idempotency.StateEstablished, &status, []byte(`{"reference":"durable-payment-1"}`), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := fixture.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Finalize(ctx, tx, first, result); err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}

	retry, err := coordinator.Acquire(ctx, fixture.pool, identity, "sha256:durable-payment", "payments.submit", policy)
	if err != nil {
		t.Fatal(err)
	}
	if retry.Decision() != idempotency.DecisionReturn || retry.Result().State() != idempotency.StateEstablished {
		t.Fatalf("retry = %#v, want established result", retry)
	}
	if _, err := coordinator.Acquire(ctx, fixture.pool, identity, "sha256:changed", "payments.submit", policy); !errors.Is(err, idempotency.ErrIdempotencyConflict) {
		t.Fatalf("changed fingerprint error = %v", err)
	}
	if _, err := coordinator.Acquire(ctx, fixture.pool, identity, "sha256:durable-payment", "payments.refund", policy); !errors.Is(err, idempotency.ErrIdempotencyConflict) {
		t.Fatalf("changed operation error = %v", err)
	}
}

func TestDurableIdempotencyConcurrentReservationsHaveOneOwner(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scope, err := accountingscope.New(uuid.New(), uuid.New(), uuid.New(), uuid.New(), "USD")
	if err != nil {
		t.Fatal(err)
	}
	identity, err := idempotency.NewIdentity(scope, "durable-concurrent-1")
	if err != nil {
		t.Fatal(err)
	}
	coordinator := idempotency.NewPostgresCoordinator()
	policy := idempotency.IdempotencyPolicy{RecordTTL: time.Hour, LeaseTTL: time.Minute}
	const callers = 16
	decisions := make(chan idempotency.AcquisitionDecision, callers)
	errorsCh := make(chan error, callers)
	var wg sync.WaitGroup
	for range callers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			acquisition, acquireErr := coordinator.Acquire(ctx, fixture.pool, identity, "sha256:durable-concurrent", "payments.submit", policy)
			if acquireErr != nil {
				errorsCh <- acquireErr
				return
			}
			decisions <- acquisition.Decision()
		}()
	}
	wg.Wait()
	close(decisions)
	close(errorsCh)
	for err := range errorsCh {
		t.Fatal(err)
	}
	owners := 0
	for decision := range decisions {
		if decision == idempotency.DecisionExecute {
			owners++
		}
	}
	if owners != 1 {
		t.Fatalf("execution owners = %d, want 1", owners)
	}
}
