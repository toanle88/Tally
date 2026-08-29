//go:build integration

package database

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

func TestOutboxDispatchPersistenceTransitionsAndFencing(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	queries := fixture.queries()

	dispatched := insertDispatchOutbox(t, ctx, queries, uuid.New(), "DispatcherEvent", time.Now().UTC().Add(-time.Minute))
	dispatcher, err := platformintegration.NewDispatcher(queries, []platformintegration.HandlerRegistration{{
		Key:     platformintegration.HandlerKey{EventType: dispatched.EventType, EventVersion: int(dispatched.EventVersion)},
		Handler: func(context.Context, events.Envelope) error { return nil },
	}}, platformintegration.DispatcherConfig{
		LeaseDuration:      100 * time.Millisecond,
		HandlerP99Duration: time.Millisecond,
		BatchSize:          1,
		Concurrency:        1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if report, err := dispatcher.DispatchOnce(ctx); err != nil || report.Established != 1 {
		t.Fatalf("dispatcher report = %#v, error = %v", report, err)
	}
	dispatchedRow, err := queries.GetOutboxBySourceIdentity(ctx, platformdb.GetOutboxBySourceIdentityParams{
		SourceContext: dispatched.SourceContext, AggregateID: dispatched.AggregateID,
		AggregateVersion: dispatched.AggregateVersion, EventType: dispatched.EventType,
	})
	if err != nil || !dispatchedRow.EstablishedAt.Valid || dispatchedRow.ClaimOwner.Valid {
		t.Fatalf("dispatched row = %#v, error = %v", dispatchedRow, err)
	}

	managedID := uuid.New()
	managed := insertDispatchOutbox(t, ctx, queries, managedID, "ManagedEvent", time.Now().UTC().Add(-time.Minute))
	claimed, err := queries.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
		LeaseDuration: pgtype.Interval{Microseconds: 100_000, Valid: true},
		ClaimOwner:    pgtype.Text{String: "worker-managed", Valid: true},
		BatchSize:     1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 1 || claimed[0].OutboxID != managed.OutboxID || claimed[0].AttemptCount != 1 {
		t.Fatalf("claimed managed candidate = %#v", claimed)
	}

	if rows, err := queries.RenewOutboxLease(ctx, platformdb.RenewOutboxLeaseParams{
		LeaseDuration: pgtype.Interval{Microseconds: 100_000, Valid: true},
		OutboxID:      managed.OutboxID,
		ClaimOwner:    pgtype.Text{String: "worker-managed", Valid: true},
	}); err != nil || rows != 1 {
		t.Fatalf("renew rows = %d, error = %v", rows, err)
	}
	if rows, err := queries.MarkOutboxManagedException(ctx, platformdb.MarkOutboxManagedExceptionParams{
		LastErrorCode: pgtype.Text{String: "handler_not_registered", Valid: true},
		OutboxID:      managed.OutboxID,
		ClaimOwner:    pgtype.Text{String: "worker-managed", Valid: true},
	}); err != nil || rows != 1 {
		t.Fatalf("managed rows = %d, error = %v", rows, err)
	}

	managedRow, err := queries.GetOutboxBySourceIdentity(ctx, platformdb.GetOutboxBySourceIdentityParams{
		SourceContext:    managed.SourceContext,
		AggregateID:      managed.AggregateID,
		AggregateVersion: managed.AggregateVersion,
		EventType:        managed.EventType,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !managedRow.ManagedExceptionAt.Valid || managedRow.LastErrorCode.String != "handler_not_registered" || managedRow.AttemptCount != 1 {
		t.Fatalf("managed row = %#v", managedRow)
	}
	claimed, err = queries.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
		LeaseDuration: pgtype.Interval{Microseconds: 100_000, Valid: true},
		ClaimOwner:    pgtype.Text{String: "worker-after-managed", Valid: true},
		BatchSize:     10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 0 {
		t.Fatalf("managed exception was reclaimable: %#v", claimed)
	}

	reschedule := insertDispatchOutbox(t, ctx, queries, uuid.New(), "RetryEvent", time.Now().UTC().Add(-time.Minute))
	claimed, err = queries.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
		LeaseDuration: pgtype.Interval{Microseconds: 100_000, Valid: true},
		ClaimOwner:    pgtype.Text{String: "worker-retry", Valid: true},
		BatchSize:     1,
	})
	if err != nil || len(claimed) != 1 || claimed[0].OutboxID != reschedule.OutboxID {
		t.Fatalf("retry claim = %#v, error = %v", claimed, err)
	}
	retryAt := time.Now().UTC().Add(5 * time.Second)
	if rows, err := queries.RescheduleOutbox(ctx, platformdb.RescheduleOutboxParams{
		AvailableAt:   timestamptzValue(retryAt),
		LastErrorCode: pgtype.Text{String: "provider_unavailable", Valid: true},
		OutboxID:      reschedule.OutboxID,
		ClaimOwner:    pgtype.Text{String: "worker-retry", Valid: true},
	}); err != nil || rows != 1 {
		t.Fatalf("reschedule rows = %d, error = %v", rows, err)
	}
	rescheduledRow, err := queries.GetOutboxBySourceIdentity(ctx, platformdb.GetOutboxBySourceIdentityParams{
		SourceContext:    reschedule.SourceContext,
		AggregateID:      reschedule.AggregateID,
		AggregateVersion: reschedule.AggregateVersion,
		EventType:        reschedule.EventType,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !rescheduledRow.AvailableAt.Valid || !rescheduledRow.AvailableAt.Time.After(time.Now().UTC()) || rescheduledRow.ClaimOwner.Valid {
		t.Fatalf("rescheduled row = %#v", rescheduledRow)
	}

	staleID := uuid.New()
	stale := insertDispatchOutbox(t, ctx, queries, staleID, "StaleEvent", time.Now().UTC().Add(-time.Minute))
	firstClaim, err := queries.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
		LeaseDuration: pgtype.Interval{Microseconds: 50_000, Valid: true},
		ClaimOwner:    pgtype.Text{String: "worker-stale", Valid: true},
		BatchSize:     1,
	})
	if err != nil || len(firstClaim) != 1 || firstClaim[0].OutboxID != stale.OutboxID {
		t.Fatalf("first stale claim = %#v, error = %v", firstClaim, err)
	}
	claimDeadline := time.Now().Add(time.Second)
	var secondClaim []platformdb.IntegrationOutbox
	for time.Now().Before(claimDeadline) {
		secondClaim, err = queries.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
			LeaseDuration: pgtype.Interval{Microseconds: 100_000, Valid: true},
			ClaimOwner:    pgtype.Text{String: "worker-current", Valid: true},
			BatchSize:     1,
		})
		if err != nil || len(secondClaim) != 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if err != nil || len(secondClaim) != 1 || secondClaim[0].OutboxID != stale.OutboxID {
		t.Fatalf("second stale claim = %#v, error = %v", secondClaim, err)
	}
	if rows, err := queries.EstablishOutbox(ctx, platformdb.EstablishOutboxParams{
		OutboxID:   stale.OutboxID,
		ClaimOwner: pgtype.Text{String: "worker-stale", Valid: true},
	}); err != nil || rows != 0 {
		t.Fatalf("stale establish rows = %d, error = %v", rows, err)
	}
	if rows, err := queries.EstablishOutbox(ctx, platformdb.EstablishOutboxParams{
		OutboxID:   stale.OutboxID,
		ClaimOwner: pgtype.Text{String: "worker-current", Valid: true},
	}); err != nil || rows != 1 {
		t.Fatalf("current establish rows = %d, error = %v", rows, err)
	}
}

func insertDispatchOutbox(t *testing.T, ctx context.Context, queries *platformdb.Queries, id uuid.UUID, eventType string, availableAt time.Time) platformdb.IntegrationOutbox {
	t.Helper()
	payload := []byte(`{"event":"dispatch"}`)
	fingerprint, err := events.ComputePayloadFingerprint(payload)
	if err != nil {
		t.Fatal(err)
	}
	row, err := queries.InsertOutbox(ctx, platformdb.InsertOutboxParams{
		OutboxID:           uuidValue(id),
		EventType:          eventType,
		EventVersion:       1,
		OccurredAt:         timestamptzValue(availableAt),
		SourceContext:      "platform-dispatch-test",
		AggregateID:        uuidValue(uuid.New()),
		AggregateVersion:   1,
		CorrelationID:      uuidValue(uuid.New()),
		CausationID:        uuidValue(uuid.New()),
		Payload:            payload,
		PayloadFingerprint: fingerprint,
		DataClassification: string(events.Internal),
		AvailableAt:        timestamptzValue(availableAt),
	})
	if err != nil {
		t.Fatal(err)
	}
	return row
}
