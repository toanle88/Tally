//go:build integration

package database

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

func TestTransactionalCoordinationSourcePublicationIsAtomic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"source"}`))
	publication := transactionalPublication(event)

	if err := coordinator.Publish(ctx, publication, func(ctx context.Context, tx pgx.Tx) error {
		_, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.source_effect (effect_id) VALUES ($1)`, schema), uuid.MustParse(event.AggregateID()))
		return err
	}); err != nil {
		t.Fatal(err)
	}

	assertSyntheticEffectCount(t, ctx, fixture, schema, "source_effect", 1)
	row, err := fixture.queries().GetOutboxBySourceIdentity(ctx, outboxSourceParams(event))
	if err != nil {
		t.Fatal(err)
	}
	if row.EventType != event.EventType() || row.PayloadFingerprint != event.PayloadFingerprint() {
		t.Fatalf("outbox row = %#v, want event metadata", row)
	}
	if !row.OccurredAt.Valid || !row.OccurredAt.Time.Equal(event.OccurredAt()) {
		t.Fatalf("outbox occurred_at = %#v, want %s", row.OccurredAt, event.OccurredAt())
	}
	if row.DataClassification != string(event.DataClassification()) {
		t.Fatalf("outbox data_classification = %q, want %q", row.DataClassification, event.DataClassification())
	}
}

func TestTransactionalCoordinationSourceFailureRollsBack(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"source-failure"}`))
	publication := transactionalPublication(event)
	cause := errors.New("synthetic source failure")

	if err := coordinator.Publish(ctx, publication, func(ctx context.Context, tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.source_effect (effect_id) VALUES ($1)`, schema), uuid.MustParse(event.AggregateID())); err != nil {
			return err
		}
		return cause
	}); !errors.Is(err, cause) {
		t.Fatalf("source error = %v, want %v", err, cause)
	}

	assertSyntheticEffectCount(t, ctx, fixture, schema, "source_effect", 0)
	if _, err := fixture.queries().GetOutboxBySourceIdentity(ctx, outboxSourceParams(event)); !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("rolled-back outbox lookup error = %v, want pgx.ErrNoRows", err)
	}
}

func TestTransactionalCoordinationConsumerPublicationIsAtomic(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"consumer"}`))
	resultReference := platformintegration.ResultReference(`{"reference":"consumer-result"}`)
	resultEvent := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"result"}`))
	resultEvent = transactionalEventWithSource(t, resultEvent, "synthetic-consumer")

	outcome, err := coordinator.Consume(ctx, platformintegration.Delivery{
		ConsumerName: "synthetic-projection",
		Event:        event,
	}, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(event.MessageID()), resultReference); err != nil {
			return nil, nil, err
		}
		return resultReference, []platformintegration.Publication{transactionalPublication(resultEvent)}, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != platformintegration.OutcomeEstablished || !jsonBytesEqual(outcome.Result(), resultReference) {
		t.Fatalf("outcome = %#v, want established result", outcome)
	}

	assertSyntheticEffectCount(t, ctx, fixture, schema, "consumer_effect", 1)
	inbox, err := fixture.queries().GetInboxByIdentity(ctx, inboxParams("synthetic-projection", event))
	if err != nil {
		t.Fatal(err)
	}
	if inbox.State != string(platformintegration.OutcomeEstablished) {
		t.Fatalf("inbox state = %q, want established", inbox.State)
	}
	if _, err := fixture.queries().GetOutboxBySourceIdentity(ctx, outboxSourceParams(resultEvent)); err != nil {
		t.Fatalf("resulting outbox lookup = %v", err)
	}
}

func TestTransactionalCoordinationConsumerFailureRetainsFailedEvidence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"consumer-failure"}`))
	cause := errors.New("synthetic consumer failure")

	outcome, err := coordinator.Consume(ctx, platformintegration.Delivery{
		ConsumerName: "synthetic-projection",
		Event:        event,
	}, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(event.MessageID()), []byte(`{"partial":true}`)); err != nil {
			return nil, nil, err
		}
		return nil, nil, cause
	})
	if !errors.Is(err, cause) {
		t.Fatalf("consumer error = %v, want %v", err, cause)
	}
	if outcome.State != platformintegration.OutcomeFailed {
		t.Fatalf("outcome = %#v, want failed", outcome)
	}

	assertSyntheticEffectCount(t, ctx, fixture, schema, "consumer_effect", 0)
	inbox, err := fixture.queries().GetInboxByIdentity(ctx, inboxParams("synthetic-projection", event))
	if err != nil {
		t.Fatal(err)
	}
	if inbox.State != string(platformintegration.OutcomeFailed) {
		t.Fatalf("inbox state = %q, want failed", inbox.State)
	}
}

func TestTransactionalCoordinationCancelledConsumerRetainsFailedEvidence(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"consumer-cancelled"}`))
	consumerCtx, cancelConsumer := context.WithCancel(ctx)
	defer cancelConsumer()

	outcome, err := coordinator.Consume(consumerCtx, platformintegration.Delivery{
		ConsumerName: "synthetic-projection",
		Event:        event,
	}, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(event.MessageID()), []byte(`{"partial":true}`)); err != nil {
			return nil, nil, err
		}
		cancelConsumer()
		return nil, nil, context.Canceled
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("consumer error = %v, want context.Canceled", err)
	}
	if outcome.State != platformintegration.OutcomeFailed {
		t.Fatalf("outcome = %#v, want failed", outcome)
	}

	assertSyntheticEffectCount(t, ctx, fixture, schema, "consumer_effect", 0)
	inbox, err := fixture.queries().GetInboxByIdentity(ctx, inboxParams("synthetic-projection", event))
	if err != nil {
		t.Fatal(err)
	}
	if inbox.State != string(platformintegration.OutcomeFailed) {
		t.Fatalf("inbox state = %q, want failed", inbox.State)
	}
}

func TestTransactionalCoordinationDuplicateReturnsEstablishedResult(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"duplicate"}`))
	resultReference := platformintegration.ResultReference(`{"reference":"duplicate-result"}`)
	delivery := platformintegration.Delivery{ConsumerName: "synthetic-projection", Event: event}
	effects := 0
	effect := func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		effects++
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(event.MessageID()), resultReference); err != nil {
			return nil, nil, err
		}
		return resultReference, nil, nil
	}

	if _, err := coordinator.Consume(ctx, delivery, effect); err != nil {
		t.Fatal(err)
	}
	outcome, err := coordinator.Consume(ctx, delivery, func(context.Context, pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		return nil, nil, errors.New("duplicate effect must not execute")
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != platformintegration.OutcomeEstablished || !jsonBytesEqual(outcome.Result(), resultReference) {
		t.Fatalf("duplicate outcome = %#v, want established result", outcome)
	}
	if effects != 1 {
		t.Fatalf("effect calls = %d, want 1", effects)
	}
	assertSyntheticEffectCount(t, ctx, fixture, schema, "consumer_effect", 1)
}

func TestTransactionalCoordinationChangedFingerprintConflicts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	messageID := uuid.NewString()
	first := transactionalTestEnvelope(t, messageID, []byte(`{"event":"original"}`))
	changed := transactionalTestEnvelope(t, messageID, []byte(`{"event":"changed"}`))
	delivery := platformintegration.Delivery{ConsumerName: "synthetic-projection", Event: first}
	if _, err := coordinator.Consume(ctx, delivery, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		if _, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(first.MessageID()), []byte(`{"reference":"original"}`)); err != nil {
			return nil, nil, err
		}
		return platformintegration.ResultReference(`{"reference":"original"}`), nil, nil
	}); err != nil {
		t.Fatal(err)
	}

	_, err := coordinator.Consume(ctx, platformintegration.Delivery{ConsumerName: delivery.ConsumerName, Event: changed}, func(context.Context, pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		return nil, nil, errors.New("conflicting effect must not execute")
	})
	var conflict platformintegration.IdentityContentConflict
	if !errors.As(err, &conflict) {
		t.Fatalf("conflict error = %v, want IdentityContentConflict", err)
	}
	if !errors.Is(err, platformintegration.ErrIdentityContentConflict) {
		t.Fatalf("conflict error = %v, want identity-content sentinel", err)
	}
	inbox, err := fixture.queries().GetInboxByIdentity(ctx, inboxParams(delivery.ConsumerName, first))
	if err != nil {
		t.Fatal(err)
	}
	if inbox.MessageFingerprint != first.PayloadFingerprint() || inbox.State != string(platformintegration.OutcomeEstablished) {
		t.Fatalf("conflicting delivery changed evidence: %#v", inbox)
	}
}

func TestTransactionalCoordinationReconciliationPreventsDuplicateEffect(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture, coordinator, schema := startTransactionalCoordinationFixture(t, ctx)
	foundEvent := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"reconcile-found"}`))
	foundReference := []byte(`{"reference":"already-established"}`)
	insertInboxRecord(t, ctx, fixture, "synthetic-projection", foundEvent, string(platformintegration.OutcomeFailed))
	insertSyntheticEffect(t, ctx, fixture, schema, foundEvent.MessageID(), foundReference)

	lookup := func(ctx context.Context, tx pgx.Tx, event events.Envelope) (platformintegration.ResultReference, bool, error) {
		var reference []byte
		err := tx.QueryRow(ctx, fmt.Sprintf(`SELECT result_reference FROM %s.consumer_effect WHERE effect_id = $1`, schema), uuid.MustParse(event.MessageID())).Scan(&reference)
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, false, nil
		}
		return reference, err == nil, err
	}
	delivery := platformintegration.Delivery{ConsumerName: "synthetic-projection", Event: foundEvent}
	outcome, err := coordinator.ReconcileAndRetry(ctx, delivery, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, bool, error) {
		return lookup(ctx, tx, foundEvent)
	}, func(context.Context, pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		return nil, nil, errors.New("established local result must prevent retry")
	})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.State != platformintegration.OutcomeEstablished || !jsonBytesEqual(outcome.Result(), foundReference) {
		t.Fatalf("reconciled outcome = %#v, want established result", outcome)
	}

	retryEvent := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"reconcile-retry"}`))
	insertInboxRecord(t, ctx, fixture, "synthetic-projection", retryEvent, string(platformintegration.OutcomeProcessing))
	retryDelivery := platformintegration.Delivery{ConsumerName: "synthetic-projection", Event: retryEvent}
	retryReference := platformintegration.ResultReference(`{"reference":"retried"}`)
	retryOutcome, err := coordinator.ReconcileAndRetry(ctx, retryDelivery, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, bool, error) {
		return lookup(ctx, tx, retryEvent)
	}, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
		if err := insertSyntheticEffectTx(ctx, tx, schema, retryEvent.MessageID(), retryReference); err != nil {
			return nil, nil, err
		}
		return retryReference, nil, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if retryOutcome.State != platformintegration.OutcomeEstablished || string(retryOutcome.Result()) != string(retryReference) {
		t.Fatalf("retry outcome = %#v, want established result", retryOutcome)
	}
	assertSyntheticEffectCount(t, ctx, fixture, schema, "consumer_effect", 2)
}

func startTransactionalCoordinationFixture(t *testing.T, ctx context.Context) (*integrationFixture, *platformintegration.Coordinator, string) {
	t.Helper()
	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	schema := "us3_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := fixture.pool.Exec(ctx, fmt.Sprintf(`
		CREATE SCHEMA %s;
		CREATE TABLE %s.source_effect (effect_id uuid primary key);
		CREATE TABLE %s.consumer_effect (
			effect_id uuid primary key,
			result_reference jsonb not null
		);`, schema, schema, schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = fixture.pool.Exec(cleanupCtx, fmt.Sprintf(`DROP SCHEMA %s CASCADE`, schema))
	})
	return fixture, platformintegration.NewCoordinator(fixture.pool), schema
}

func transactionalTestEnvelope(t *testing.T, messageID string, payload []byte) events.Envelope {
	t.Helper()
	fingerprint, err := events.ComputePayloadFingerprint(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := events.NewEnvelope(events.EnvelopeInput{
		MessageID:          messageID,
		EventType:          "SyntheticEvent",
		EventVersion:       1,
		OccurredAt:         time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC),
		SourceContext:      "synthetic",
		AggregateID:        uuid.NewString(),
		AggregateVersion:   1,
		CorrelationID:      uuid.NewString(),
		CausationID:        uuid.NewString(),
		DataClassification: events.Internal,
		PayloadFingerprint: fingerprint,
		Data:               payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func transactionalEventWithSource(t *testing.T, event events.Envelope, source string) events.Envelope {
	t.Helper()
	payload := event.Data()
	fingerprint, err := events.ComputePayloadFingerprint(payload)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := events.NewEnvelope(events.EnvelopeInput{
		MessageID:          event.MessageID(),
		EventType:          event.EventType(),
		EventVersion:       event.EventVersion(),
		OccurredAt:         event.OccurredAt(),
		SourceContext:      source,
		AggregateID:        event.AggregateID(),
		AggregateVersion:   event.AggregateVersion(),
		CorrelationID:      event.CorrelationID(),
		CausationID:        event.CausationID(),
		DataClassification: event.DataClassification(),
		PayloadFingerprint: fingerprint,
		Data:               payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	return updated
}

func transactionalPublication(event events.Envelope) platformintegration.Publication {
	return platformintegration.Publication{Event: event, AvailableAt: time.Now().UTC()}
}

func outboxSourceParams(event events.Envelope) platformdb.GetOutboxBySourceIdentityParams {
	return platformdb.GetOutboxBySourceIdentityParams{
		SourceContext:    event.SourceContext(),
		AggregateID:      uuidValue(uuid.MustParse(event.AggregateID())),
		AggregateVersion: event.AggregateVersion(),
		EventType:        event.EventType(),
	}
}

func inboxParams(consumer string, event events.Envelope) platformdb.GetInboxByIdentityParams {
	return platformdb.GetInboxByIdentityParams{ConsumerName: consumer, MessageID: uuidValue(uuid.MustParse(event.MessageID()))}
}

func insertInboxRecord(t *testing.T, ctx context.Context, fixture *integrationFixture, consumer string, event events.Envelope, state string) {
	t.Helper()
	if _, err := fixture.queries().InsertInbox(ctx, platformdb.InsertInboxParams{
		ConsumerName:       consumer,
		MessageID:          uuidValue(uuid.MustParse(event.MessageID())),
		MessageFingerprint: event.PayloadFingerprint(),
		State:              state,
	}); err != nil {
		t.Fatal(err)
	}
}

func insertSyntheticEffect(t *testing.T, ctx context.Context, fixture *integrationFixture, schema, effectID string, reference []byte) {
	t.Helper()
	if _, err := fixture.pool.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(effectID), reference); err != nil {
		t.Fatal(err)
	}
}

func insertSyntheticEffectTx(ctx context.Context, tx pgx.Tx, schema, effectID string, reference []byte) error {
	_, err := tx.Exec(ctx, fmt.Sprintf(`INSERT INTO %s.consumer_effect (effect_id, result_reference) VALUES ($1, $2)`, schema), uuid.MustParse(effectID), reference)
	return err
}

func assertSyntheticEffectCount(t *testing.T, ctx context.Context, fixture *integrationFixture, schema, table string, want int) {
	t.Helper()
	var got int
	if err := fixture.pool.QueryRow(ctx, fmt.Sprintf(`SELECT count(*) FROM %s.%s`, schema, table)).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s.%s rows = %d, want %d", schema, table, got, want)
	}
}

func jsonBytesEqual(left, right []byte) bool {
	leftCanonical, leftErr := events.CanonicalizePayload(left)
	rightCanonical, rightErr := events.CanonicalizePayload(right)
	return leftErr == nil && rightErr == nil && bytes.Equal(leftCanonical, rightCanonical)
}
