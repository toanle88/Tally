//go:build integration

package database

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
)

func TestOutboxInboxPersistence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	assertIntegrationSchemaObjects(t, ctx, fixture.pool)

	firstOutboxID := uuid.New()
	firstAggregateID := uuid.New()
	firstOutbox := insertIntegrationOutbox(
		t,
		ctx,
		fixture.queries(),
		firstOutboxID,
		firstAggregateID,
		"JournalEntryPosted",
		time.Now().UTC().Add(-3*time.Minute),
	)
	secondOutboxID := uuid.New()
	insertIntegrationOutbox(
		t,
		ctx,
		fixture.queries(),
		secondOutboxID,
		uuid.New(),
		"JournalEntryApproved",
		time.Now().UTC().Add(-2*time.Minute),
	)
	insertIntegrationOutbox(
		t,
		ctx,
		fixture.queries(),
		uuid.New(),
		uuid.New(),
		"BusyEvent",
		time.Now().UTC().Add(-time.Minute),
	)
	expiredOutboxID := uuid.New()
	insertIntegrationOutbox(
		t,
		ctx,
		fixture.queries(),
		expiredOutboxID,
		uuid.New(),
		"ExpiredEvent",
		time.Now().UTC().Add(-time.Minute),
	)
	establishedOutboxID := uuid.New()
	insertIntegrationOutbox(
		t,
		ctx,
		fixture.queries(),
		establishedOutboxID,
		uuid.New(),
		"EstablishedEvent",
		time.Now().UTC().Add(-30*time.Second),
	)

	if _, err := fixture.pool.Exec(
		ctx,
		`UPDATE integration.outbox
		 SET claimed_until = clock_timestamp() + interval '1 hour', claim_owner = 'busy-worker'
		 WHERE event_type = 'BusyEvent'`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(
		ctx,
		`UPDATE integration.outbox
		 SET claimed_until = clock_timestamp() - interval '1 minute', claim_owner = 'expired-worker'
		 WHERE event_type = 'ExpiredEvent'`,
	); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.pool.Exec(
		ctx,
		`UPDATE integration.outbox
		 SET established_at = clock_timestamp()
		 WHERE event_type = 'EstablishedEvent'`,
	); err != nil {
		t.Fatal(err)
	}

	gotBySource, err := fixture.queries().GetOutboxBySourceIdentity(
		ctx,
		platformdb.GetOutboxBySourceIdentityParams{
			SourceContext:    firstOutbox.SourceContext,
			AggregateID:      firstOutbox.AggregateID,
			AggregateVersion: firstOutbox.AggregateVersion,
			EventType:        firstOutbox.EventType,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotBySource.OutboxID != firstOutbox.OutboxID || string(gotBySource.Payload) != string(firstOutbox.Payload) {
		t.Fatalf("source lookup = %#v, want %#v", gotBySource, firstOutbox)
	}

	expired, err := fixture.queries().ListExpiredOutbox(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(expired) != 1 || expired[0].EventType != "ExpiredEvent" {
		t.Fatalf("expired outbox = %#v, want expired-event row", expired)
	}

	claimed, err := fixture.queries().ClaimDueOutbox(
		ctx,
		platformdb.ClaimDueOutboxParams{
			LeaseDuration: pgtype.Interval{
				Microseconds: int64((30 * time.Second).Microseconds()),
				Valid:        true,
			},
			ClaimOwner: pgtype.Text{String: "worker-a", Valid: true},
			BatchSize:  3,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(claimed) != 3 {
		t.Fatalf("claimed rows = %d, want 3", len(claimed))
	}
	if claimed[0].OutboxID.Bytes != [16]byte(firstOutboxID) ||
		claimed[1].OutboxID.Bytes != [16]byte(secondOutboxID) ||
		claimed[2].OutboxID.Bytes != [16]byte(expiredOutboxID) {
		t.Fatalf("claimed order = %#v, want first, second, and expired rows", claimed)
	}
	for _, row := range claimed {
		if row.OutboxID.Bytes == [16]byte(establishedOutboxID) {
			t.Fatal("established outbox row was claimed")
		}
	}
	for _, row := range claimed {
		if row.ClaimOwner.String != "worker-a" || !row.ClaimOwner.Valid || row.AttemptCount != 1 {
			t.Fatalf("claimed metadata = %#v, want worker-a and attempt 1", row)
		}
	}

	if _, err := fixture.queries().InsertOutbox(
		ctx,
		platformdb.InsertOutboxParams{
			OutboxID:           uuidValue(uuid.New()),
			EventType:          firstOutbox.EventType,
			EventVersion:       firstOutbox.EventVersion,
			OccurredAt:         firstOutbox.OccurredAt,
			SourceContext:      firstOutbox.SourceContext,
			AggregateID:        firstOutbox.AggregateID,
			AggregateVersion:   firstOutbox.AggregateVersion,
			CorrelationID:      firstOutbox.CorrelationID,
			CausationID:        firstOutbox.CausationID,
			Payload:            firstOutbox.Payload,
			PayloadFingerprint: firstOutbox.PayloadFingerprint,
			DataClassification: firstOutbox.DataClassification,
			AvailableAt:        firstOutbox.AvailableAt,
		},
	); !isPostgresConstraintError(err, "23505") {
		t.Fatalf("duplicate outbox error = %v, want unique violation", err)
	}

	testConcurrentOutboxClaims(t, ctx, fixture.pool)

	processingID := uuid.New()
	failedID := uuid.New()
	establishedInboxID := uuid.New()
	insertIntegrationInbox(t, ctx, fixture.queries(), "projection", processingID, "fp-processing", "processing", nil, pgtype.Timestamptz{})
	insertIntegrationInbox(t, ctx, fixture.queries(), "projection", failedID, "fp-failed", "failed", []byte(`{"reference":"failed"}`), pgtype.Timestamptz{})
	insertIntegrationInbox(t, ctx, fixture.queries(), "projection", establishedInboxID, "fp-established", "established", []byte(`{"reference":"established"}`), timestamptzValue(time.Now().UTC()))

	if _, err := fixture.pool.Exec(
		ctx,
		`UPDATE integration.inbox
		 SET first_received_at = CASE message_id
		     WHEN $1 THEN clock_timestamp() - interval '3 minutes'
		     WHEN $2 THEN clock_timestamp() - interval '2 minutes'
		     WHEN $3 THEN clock_timestamp() - interval '1 minute'
		     END
		 WHERE message_id IN ($1, $2, $3)`,
		failedID,
		processingID,
		establishedInboxID,
	); err != nil {
		t.Fatal(err)
	}

	gotInbox, err := fixture.queries().GetInboxByIdentity(
		ctx,
		platformdb.GetInboxByIdentityParams{
			ConsumerName: "projection",
			MessageID:    uuidValue(processingID),
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if gotInbox.State != "processing" || gotInbox.MessageFingerprint != "fp-processing" {
		t.Fatalf("inbox lookup = %#v, want processing record", gotInbox)
	}

	reconciliation, err := fixture.queries().ListInboxForReconciliation(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(reconciliation) != 2 || reconciliation[0].MessageID.Bytes != [16]byte(failedID) || reconciliation[1].MessageID.Bytes != [16]byte(processingID) {
		t.Fatalf("reconciliation rows = %#v, want failed then processing", reconciliation)
	}

	if _, err := fixture.queries().InsertInbox(
		ctx,
		platformdb.InsertInboxParams{
			ConsumerName:       "projection",
			MessageID:          uuidValue(processingID),
			MessageFingerprint: "changed-fingerprint",
			State:              "processing",
		},
	); !isPostgresConstraintError(err, "23505") {
		t.Fatalf("duplicate inbox error = %v, want unique violation", err)
	}
	if _, err := fixture.queries().InsertInbox(
		ctx,
		platformdb.InsertInboxParams{
			ConsumerName:       "projection",
			MessageID:          uuidValue(uuid.New()),
			MessageFingerprint: "fp-invalid",
			State:              "invalid",
		},
	); !isPostgresConstraintError(err, "23514") {
		t.Fatalf("invalid inbox state error = %v, want check violation", err)
	}

	rollbackID := uuid.New()
	tx, err := fixture.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = fixture.queries().WithTx(tx).InsertOutbox(
		ctx,
		newOutboxParams(rollbackID, uuid.New(), "RolledBackEvent", time.Now().UTC()),
	)
	if err != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(err)
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.queries().GetOutboxBySourceIdentity(ctx, platformdb.GetOutboxBySourceIdentityParams{
		SourceContext:    "platform-test",
		AggregateID:      uuidValue(rollbackID),
		AggregateVersion: 1,
		EventType:        "RolledBackEvent",
	}); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("rolled-back outbox lookup error = %v, want pgx.ErrNoRows", err)
	}

	fixture.pool.Close()
	fixture.pool = nil
	fixture.openPool(t, ctx)

	durableOutbox, err := fixture.queries().GetOutboxBySourceIdentity(ctx, platformdb.GetOutboxBySourceIdentityParams{
		SourceContext:    firstOutbox.SourceContext,
		AggregateID:      firstOutbox.AggregateID,
		AggregateVersion: firstOutbox.AggregateVersion,
		EventType:        firstOutbox.EventType,
	})
	if err != nil {
		t.Fatal(err)
	}
	if durableOutbox.OutboxID != firstOutbox.OutboxID || durableOutbox.AttemptCount != 1 {
		t.Fatalf("durable outbox after reopen = %#v, want claimed row", durableOutbox)
	}
	durableInbox, err := fixture.queries().GetInboxByIdentity(ctx, platformdb.GetInboxByIdentityParams{
		ConsumerName: "projection",
		MessageID:    uuidValue(processingID),
	})
	if err != nil {
		t.Fatal(err)
	}
	if durableInbox.State != "processing" {
		t.Fatalf("durable inbox after reopen = %#v, want processing", durableInbox)
	}
}

func assertIntegrationSchemaObjects(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	for _, table := range []string{"outbox", "inbox"} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM information_schema.tables
				WHERE table_schema = 'integration' AND table_name = $1
			)`, table).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("integration.%s table does not exist", table)
		}
	}

	for _, constraint := range []string{
		"integration_outbox_pk",
		"integration_outbox_source_event_unique",
		"integration_outbox_established_managed_check",
		"integration_inbox_pk",
		"integration_inbox_state_check",
	} {
		var exists bool
		if err := pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM pg_constraint WHERE conname = $1
			)`, constraint).Scan(&exists); err != nil {
			t.Fatal(err)
		}
		if !exists {
			t.Fatalf("constraint %s does not exist", constraint)
		}
	}

	var indexExists bool
	if err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM pg_indexes
			WHERE schemaname = 'integration' AND indexname = 'integration_outbox_due_idx'
		)`).Scan(&indexExists); err != nil {
		t.Fatal(err)
	}
	if !indexExists {
		t.Fatal("integration_outbox_due_idx does not exist")
	}
}

func insertIntegrationOutbox(
	t *testing.T,
	ctx context.Context,
	queries *platformdb.Queries,
	outboxID uuid.UUID,
	aggregateID uuid.UUID,
	eventType string,
	availableAt time.Time,
) platformdb.IntegrationOutbox {
	t.Helper()

	row, err := queries.InsertOutbox(ctx, newOutboxParams(outboxID, aggregateID, eventType, availableAt))
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func newOutboxParams(
	outboxID uuid.UUID,
	aggregateID uuid.UUID,
	eventType string,
	availableAt time.Time,
) platformdb.InsertOutboxParams {
	return platformdb.InsertOutboxParams{
		OutboxID:           uuidValue(outboxID),
		EventType:          eventType,
		EventVersion:       1,
		OccurredAt:         timestamptzValue(availableAt),
		SourceContext:      "platform-test",
		AggregateID:        uuidValue(aggregateID),
		AggregateVersion:   1,
		CorrelationID:      uuidValue(uuid.New()),
		CausationID:        uuidValue(uuid.New()),
		Payload:            []byte(`{"event":"test"}`),
		PayloadFingerprint: "sha256:integration-test",
		DataClassification: "internal",
		AvailableAt:        timestamptzValue(availableAt),
	}
}

func insertIntegrationInbox(
	t *testing.T,
	ctx context.Context,
	queries *platformdb.Queries,
	consumerName string,
	messageID uuid.UUID,
	fingerprint string,
	state string,
	resultReference []byte,
	establishedAt pgtype.Timestamptz,
) platformdb.IntegrationInbox {
	t.Helper()

	row, err := queries.InsertInbox(ctx, platformdb.InsertInboxParams{
		ConsumerName:       consumerName,
		MessageID:          uuidValue(messageID),
		MessageFingerprint: fingerprint,
		State:              state,
		ResultReference:    resultReference,
		EstablishedAt:      establishedAt,
	})
	if err != nil {
		t.Fatal(err)
	}
	return row
}

func testConcurrentOutboxClaims(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()

	queries := platformdb.New(pool)
	insertIntegrationOutbox(t, ctx, queries, uuid.New(), uuid.New(), "ConcurrentEventA", time.Now().UTC().Add(-time.Minute))
	insertIntegrationOutbox(t, ctx, queries, uuid.New(), uuid.New(), "ConcurrentEventB", time.Now().UTC().Add(-time.Minute))

	start := make(chan struct{})
	results := make(chan []platformdb.IntegrationOutbox, 2)
	errorsCh := make(chan error, 2)
	var wg sync.WaitGroup
	for _, owner := range []string{"concurrent-worker-a", "concurrent-worker-b"} {
		wg.Add(1)
		go func(owner string) {
			defer wg.Done()
			<-start
			claimed, err := queries.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
				LeaseDuration: pgtype.Interval{
					Microseconds: int64((30 * time.Second).Microseconds()),
					Valid:        true,
				},
				ClaimOwner: pgtype.Text{String: owner, Valid: true},
				BatchSize:  1,
			})
			if err != nil {
				errorsCh <- err
				return
			}
			results <- claimed
		}(owner)
	}
	close(start)
	wg.Wait()
	close(results)
	close(errorsCh)

	for err := range errorsCh {
		t.Fatal(err)
	}
	claimedIDs := make(map[[16]byte]struct{})
	claimedRows := 0
	for rows := range results {
		claimedRows += len(rows)
		for _, row := range rows {
			claimedIDs[row.OutboxID.Bytes] = struct{}{}
		}
	}
	if claimedRows != 2 || len(claimedIDs) != 2 {
		t.Fatalf("concurrent claims = rows %d, unique IDs %d; want two distinct rows", claimedRows, len(claimedIDs))
	}
}

func uuidValue(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(value), Valid: true}
}

func timestamptzValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

func isPostgresConstraintError(err error, code string) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == code
}
