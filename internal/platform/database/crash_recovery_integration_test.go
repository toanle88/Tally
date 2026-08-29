//go:build integration

package database

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

const (
	crashHelperEnv = "TALLY_CRASH_HELPER"
	crashExitCode  = 97
)

func TestCrashRecoveryCommitBoundaries(t *testing.T) {
	if os.Getenv(crashHelperEnv) == "1" {
		runCrashRecoveryHelper(t)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()
	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)
	schema := "us5_crash_" + uuid.NewString()[:8]
	if _, err := fixture.pool.Exec(ctx, fmt.Sprintf(`
		CREATE SCHEMA %s;
		CREATE TABLE %s.source_effect (effect_id uuid primary key);
		CREATE TABLE %s.consumer_effect (effect_id uuid primary key);`, schema, schema, schema)); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = fixture.pool.Exec(cleanupCtx, fmt.Sprintf("DROP SCHEMA %s CASCADE", schema))
	})

	t.Run("source before commit", func(t *testing.T) {
		event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"crash":"source-before"}`))
		runCrashHelper(t, fixture.databaseURL, schema, event, "source", "commit-before")
		assertTableCount(t, ctx, fixture, schema, "source_effect", 0)
		assertOutboxAbsent(t, ctx, fixture, event)
	})

	t.Run("source after commit", func(t *testing.T) {
		event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"crash":"source-after"}`))
		runCrashHelper(t, fixture.databaseURL, schema, event, "source", "commit-after")
		assertTableCount(t, ctx, fixture, schema, "source_effect", 1)
		if _, err := fixture.queries().GetOutboxBySourceIdentity(ctx, outboxSourceParams(event)); err != nil {
			t.Fatalf("committed outbox lookup = %v", err)
		}
	})

	for _, phase := range []string{"effect-before-establishment", "commit-before"} {
		t.Run("consumer "+phase, func(t *testing.T) {
			event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"crash":"consumer-before"}`))
			runCrashHelper(t, fixture.databaseURL, schema, event, "consumer", phase)
			assertTableCount(t, ctx, fixture, schema, "consumer_effect", 0)
			assertInboxAbsent(t, ctx, fixture, "crash-consumer", event)
		})
	}

	t.Run("consumer after commit", func(t *testing.T) {
		event := transactionalTestEnvelope(t, uuid.NewString(), []byte(`{"crash":"consumer-after"}`))
		runCrashHelper(t, fixture.databaseURL, schema, event, "consumer", "commit-after")
		assertTableCount(t, ctx, fixture, schema, "consumer_effect", 1)
		inbox, err := fixture.queries().GetInboxByIdentity(ctx, inboxParams("crash-consumer", event))
		if err != nil || inbox.State != string(platformintegration.OutcomeEstablished) {
			t.Fatalf("committed inbox = %#v, error = %v", inbox, err)
		}
	})
}

func runCrashHelper(t *testing.T, databaseURL, schema string, event events.Envelope, operation, phase string) {
	t.Helper()
	payload, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	command := exec.Command(os.Args[0], "-test.run=TestCrashRecoveryCommitBoundaries")
	command.Env = append(os.Environ(),
		crashHelperEnv+"=1",
		"TALLY_CRASH_DATABASE_URL="+databaseURL,
		"TALLY_CRASH_SCHEMA="+schema,
		"TALLY_CRASH_EVENT="+string(payload),
		"TALLY_CRASH_OPERATION="+operation,
		"TALLY_CRASH_PHASE="+phase,
	)
	err = command.Run()
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != crashExitCode {
		t.Fatalf("crash helper error = %v, want exit code %d", err, crashExitCode)
	}
}

func runCrashRecoveryHelper(t *testing.T) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, os.Getenv("TALLY_CRASH_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	var event events.Envelope
	if err := json.Unmarshal([]byte(os.Getenv("TALLY_CRASH_EVENT")), &event); err != nil {
		t.Fatal(err)
	}
	phase := os.Getenv("TALLY_CRASH_PHASE")
	coordinator := platformintegration.NewCoordinator(crashTxBeginner{db: pool, phase: phase})
	schema := os.Getenv("TALLY_CRASH_SCHEMA")
	if os.Getenv("TALLY_CRASH_OPERATION") == "source" {
		_ = coordinator.Publish(ctx, platformintegration.Publication{Event: event, AvailableAt: time.Now().UTC()}, func(ctx context.Context, tx pgx.Tx) error {
			_, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s.source_effect (effect_id) VALUES ($1)", schema), uuid.MustParse(event.MessageID()))
			return err
		})
	} else {
		_, _ = coordinator.Consume(ctx, platformintegration.Delivery{ConsumerName: "crash-consumer", Event: event}, func(ctx context.Context, tx pgx.Tx) (platformintegration.ResultReference, []platformintegration.Publication, error) {
			if _, err := tx.Exec(ctx, fmt.Sprintf("INSERT INTO %s.consumer_effect (effect_id) VALUES ($1)", schema), uuid.MustParse(event.MessageID())); err != nil {
				return nil, nil, err
			}
			if phase == "effect-before-establishment" {
				os.Exit(crashExitCode)
			}
			return platformintegration.ResultReference(`{"crash":true}`), nil, nil
		})
	}
	os.Exit(crashExitCode)
}

type crashTxBeginner struct {
	db    *pgxpool.Pool
	phase string
}

func (b crashTxBeginner) BeginTx(ctx context.Context, options pgx.TxOptions) (pgx.Tx, error) {
	tx, err := b.db.BeginTx(ctx, options)
	if err != nil {
		return nil, err
	}
	return crashTx{Tx: tx, phase: b.phase}, nil
}

type crashTx struct {
	pgx.Tx
	phase string
}

func (tx crashTx) Commit(ctx context.Context) error {
	if tx.phase == "commit-before" {
		os.Exit(crashExitCode)
	}
	err := tx.Tx.Commit(ctx)
	if tx.phase == "commit-after" {
		os.Exit(crashExitCode)
	}
	return err
}

func assertTableCount(t *testing.T, ctx context.Context, fixture *integrationFixture, schema, table string, want int) {
	t.Helper()
	var got int
	if err := fixture.pool.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s.%s", schema, table)).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("%s.%s count = %d, want %d", schema, table, got, want)
	}
}

func assertOutboxAbsent(t *testing.T, ctx context.Context, fixture *integrationFixture, event events.Envelope) {
	t.Helper()
	if _, err := fixture.queries().GetOutboxBySourceIdentity(ctx, outboxSourceParams(event)); err == nil {
		t.Fatal("outbox row unexpectedly exists")
	}
}

func assertInboxAbsent(t *testing.T, ctx context.Context, fixture *integrationFixture, consumer string, event events.Envelope) {
	t.Helper()
	if _, err := fixture.queries().GetInboxByIdentity(ctx, inboxParams(consumer, event)); err == nil {
		t.Fatal("inbox row unexpectedly exists")
	}
}

var _ platformintegration.TxBeginner = crashTxBeginner{}
