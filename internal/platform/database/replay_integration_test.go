//go:build integration

package database

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

func TestReplayPreservesIdentityEvidenceAndAvoidsDuplicateEffects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	schema := "us5_replay_" + uuid.NewString()[:8]
	if _, err := fixture.pool.Exec(ctx, fmt.Sprintf(`
		CREATE SCHEMA %s;
		CREATE TABLE %s.consumer_effect (
			id bigserial primary key,
			event_id uuid not null
		);`, schema, schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = fixture.pool.Exec(cleanupCtx, fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	})

	coordinator := platformintegration.NewCoordinator(fixture.pool)
	eventsToReplay := []events.Envelope{
		transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"one"}`)),
		transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"two"}`)),
	}
	for _, event := range eventsToReplay {
		if err := coordinator.Publish(ctx, platformintegration.Publication{Event: event, AvailableAt: time.Now().UTC()}, func(context.Context, pgx.Tx) error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := fixture.queries().InsertInbox(ctx, platformdb.InsertInboxParams{
		ConsumerName:       "live-projection",
		MessageID:          uuidValue(uuid.MustParse(eventsToReplay[0].MessageID())),
		MessageFingerprint: eventsToReplay[0].PayloadFingerprint(),
		State:              string(platformintegration.OutcomeEstablished),
	}); err != nil {
		t.Fatal(err)
	}

	beforeOutbox := countRows(t, ctx, fixture, "integration", "outbox")
	replayer, err := platformintegration.NewReplayer(fixture.pool, []platformintegration.ReplayRegistration{{
		Key: platformintegration.HandlerKey{EventType: "SyntheticEvent", EventVersion: 1},
		Effect: func(ctx context.Context, tx pgx.Tx, event events.Envelope) (platformintegration.ResultReference, error) {
			if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s.consumer_effect (event_id) VALUES ($1)", schema), uuid.MustParse(event.MessageID())); err != nil {
				return nil, err
			}
			return platformintegration.ResultReference(`{"replayed":true}`), nil
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	request := platformintegration.ReplayRequest{
		SourceContext: "synthetic",
		From:          time.Now().UTC().Add(-time.Minute),
		To:            time.Now().UTC().Add(time.Minute),
		ConsumerName:  "projection",
		Generation:    "v1",
		MaxEvents:     2,
	}
	report, err := replayer.Replay(ctx, request)
	if err != nil || report.Selected != 2 || report.Established != 2 || report.Deduplicated != 0 {
		t.Fatalf("first replay report = %#v, error = %v", report, err)
	}
	if got := countRows(t, ctx, fixture, schema, "consumer_effect"); got != 2 {
		t.Fatalf("first replay effects = %d, want 2", got)
	}

	report, err = replayer.Replay(ctx, request)
	if err != nil || report.Deduplicated != 2 {
		t.Fatalf("duplicate replay report = %#v, error = %v", report, err)
	}
	if got := countRows(t, ctx, fixture, schema, "consumer_effect"); got != 2 {
		t.Fatalf("duplicate replay effects = %d, want 2", got)
	}

	request.Generation = "v2"
	report, err = replayer.Replay(ctx, request)
	if err != nil || report.Established != 2 || report.Deduplicated != 0 {
		t.Fatalf("new generation report = %#v, error = %v", report, err)
	}
	if got := countRows(t, ctx, fixture, schema, "consumer_effect"); got != 4 {
		t.Fatalf("new generation effects = %d, want 4", got)
	}
	if got := countRows(t, ctx, fixture, "integration", "outbox"); got != beforeOutbox {
		t.Fatalf("replay changed outbox count from %d to %d", beforeOutbox, got)
	}

	var inboxCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM integration.inbox WHERE consumer_name LIKE 'projection.replay.%'`).Scan(&inboxCount); err != nil {
		t.Fatal(err)
	}
	if inboxCount != 4 {
		t.Fatalf("replay inbox rows = %d, want 4", inboxCount)
	}
	if _, err := fixture.queries().GetInboxByIdentity(ctx, platformdb.GetInboxByIdentityParams{
		ConsumerName: "live-projection",
		MessageID:    uuidValue(uuid.MustParse(eventsToReplay[0].MessageID())),
	}); err != nil {
		t.Fatalf("live inbox evidence was removed: %v", err)
	}
}

func TestOutboxEventFactsAreImmutableButOperationalFieldsChange(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"immutable"}`))
	if err := platformintegration.NewCoordinator(fixture.pool).Publish(ctx, platformintegration.Publication{Event: event, AvailableAt: time.Now().UTC()}, func(context.Context, pgx.Tx) error { return nil }); err != nil {
		t.Fatal(err)
	}
	_, err := fixture.pool.Exec(ctx, `UPDATE integration.outbox SET payload = '{"changed":true}'::jsonb WHERE outbox_id = $1`, uuid.MustParse(event.MessageID()))
	if err == nil {
		t.Fatal("immutable payload update unexpectedly succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, `UPDATE integration.outbox SET available_at = clock_timestamp() WHERE outbox_id = $1`, uuid.MustParse(event.MessageID())); err != nil {
		t.Fatalf("operational update failed: %v", err)
	}
}

func TestReplayCannotPublishOutboxEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"event":"replay-guard"}`))
	if err := platformintegration.NewCoordinator(fixture.pool).Publish(ctx, platformintegration.Publication{Event: event, AvailableAt: time.Now().UTC()}, func(context.Context, pgx.Tx) error { return nil }); err != nil {
		t.Fatal(err)
	}
	before := countRows(t, ctx, fixture, "integration", "outbox")
	replayer, err := platformintegration.NewReplayer(fixture.pool, []platformintegration.ReplayRegistration{{
		Key: platformintegration.HandlerKey{EventType: "SyntheticEvent", EventVersion: 1},
		Effect: func(ctx context.Context, tx pgx.Tx, event events.Envelope) (platformintegration.ResultReference, error) {
			_, err := tx.Exec(ctx, `INSERT INTO integration.outbox (outbox_id, event_type, event_version, source_context, aggregate_id, aggregate_version, correlation_id, causation_id, payload, payload_fingerprint, available_at) VALUES ($1, 'ReplayPublication', 1, 'synthetic', $1, 1, $1, $1, '{}'::jsonb, 'sha256:replay', clock_timestamp())`, uuid.MustParse(event.MessageID()))
			return nil, err
		},
	}})
	if err != nil {
		t.Fatal(err)
	}
	report, err := replayer.Replay(ctx, platformintegration.ReplayRequest{
		SourceContext: "synthetic", From: time.Now().UTC().Add(-time.Minute), To: time.Now().UTC().Add(time.Minute),
		ConsumerName: "projection", Generation: "guard", MaxEvents: 1,
	})
	if err == nil || report.Failed != 1 {
		t.Fatalf("guard replay report = %#v, error = %v", report, err)
	}
	if got := countRows(t, ctx, fixture, "integration", "outbox"); got != before {
		t.Fatalf("guard replay changed outbox count from %d to %d", before, got)
	}
}

func countRows(t *testing.T, ctx context.Context, fixture *integrationFixture, schema, table string) int {
	t.Helper()
	var count int
	if err := fixture.pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s.%s", schema, table)).Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}
