//go:build integration

package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type migrationSet struct {
	name          string
	directory     string
	historySchema string
	historyTable  string
}

func (s migrationSet) qualifiedHistoryTable() string {
	return s.historySchema + "." + s.historyTable
}

type migrationState struct {
	currentVersion int64
	targetVersion  int64
	appliedNow     int
	statusCount    int
	hasPending     bool
}

type migrationHistory struct {
	appliedRows   int64
	latestVersion int64
}

func applyAndVerifyMigrations(
	t *testing.T,
	ctx context.Context,
	databaseURL string,
	sets []migrationSet,
) map[string]migrationState {
	t.Helper()

	states := make(map[string]migrationState, len(sets))

	for _, set := range sets {
		state, err := applyMigrationSet(ctx, databaseURL, set)
		if err != nil {
			failPhase(t, "apply "+set.name+" migrations", err)
		}

		assertMigrationState(t, set, state)

		states[set.name] = state
	}

	return states
}

func applyMigrationSet(
	ctx context.Context,
	databaseURL string,
	set migrationSet,
) (migrationState, error) {
	var state migrationState

	if _, err := os.Stat(set.directory); err != nil {
		return state, fmt.Errorf(
			"inspect migration directory %q: %w",
			set.name,
			err,
		)
	}

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return state, fmt.Errorf(
			"open Goose connection for %q: %w",
			set.name,
			err,
		)
	}

	provider, err := goose.NewProvider(
		goose.DialectPostgres,
		db,
		os.DirFS(set.directory),
		goose.WithTableName(set.qualifiedHistoryTable()),
	)
	if err != nil {
		_ = db.Close()

		return state, fmt.Errorf(
			"create Goose provider for %q: %w",
			set.name,
			err,
		)
	}

	defer func() {
		_ = provider.Close()
	}()

	if err := provider.Ping(ctx); err != nil {
		return state, fmt.Errorf(
			"ping migration database for %q: %w",
			set.name,
			err,
		)
	}

	results, err := provider.Up(ctx)
	if err != nil {
		return state, fmt.Errorf(
			"apply migrations for %q: %w",
			set.name,
			err,
		)
	}
	state.appliedNow = len(results)

	current, target, err := provider.GetVersions(ctx)
	if err != nil {
		return state, fmt.Errorf(
			"get migration versions for %q: %w",
			set.name,
			err,
		)
	}
	state.currentVersion = current
	state.targetVersion = target

	state.hasPending, err = provider.HasPending(ctx)
	if err != nil {
		return state, fmt.Errorf(
			"check pending migrations for %q: %w",
			set.name,
			err,
		)
	}

	statuses, err := provider.Status(ctx)
	if err != nil {
		return state, fmt.Errorf(
			"read migration status for %q: %w",
			set.name,
			err,
		)
	}
	state.statusCount = len(statuses)

	for _, status := range statuses {
		if status.State != goose.StateApplied {
			return state, fmt.Errorf(
				"migration %q in set %q has state %s",
				status.Source.Path,
				set.name,
				status.State,
			)
		}
	}

	return state, nil
}

func assertMigrationState(
	t *testing.T,
	set migrationSet,
	state migrationState,
) {
	t.Helper()

	if state.currentVersion <= 0 {
		t.Fatalf(
			"%s current migration version = %d, want positive",
			set.name,
			state.currentVersion,
		)
	}

	if state.currentVersion != state.targetVersion {
		t.Fatalf(
			"%s current migration version = %d, target = %d",
			set.name,
			state.currentVersion,
			state.targetVersion,
		)
	}

	if state.statusCount == 0 {
		t.Fatalf("%s migration set has no migration sources", set.name)
	}

	if state.hasPending {
		t.Fatalf("%s migration set still has pending work", set.name)
	}
}

func assertMigrationReapplicationIsHarmless(
	t *testing.T,
	first map[string]migrationState,
	second map[string]migrationState,
) {
	t.Helper()

	for name, firstState := range first {
		secondState, ok := second[name]
		if !ok {
			t.Fatalf("second application omitted migration set %q", name)
		}

		if secondState.appliedNow != 0 {
			t.Fatalf(
				"%s second application applied %d migrations, want 0",
				name,
				secondState.appliedNow,
			)
		}

		if secondState.hasPending {
			t.Fatalf(
				"%s second application still has pending migrations",
				name,
			)
		}

		if secondState.currentVersion != firstState.currentVersion {
			t.Fatalf(
				"%s version changed on second application: first=%d second=%d",
				name,
				firstState.currentVersion,
				secondState.currentVersion,
			)
		}
	}
}

func verifyMigrationHistory(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	sets []migrationSet,
	states map[string]migrationState,
) {
	t.Helper()

	for _, set := range sets {
		state, ok := states[set.name]
		if !ok {
			t.Fatalf("missing migration state for set %q", set.name)
		}

		history, err := readMigrationHistory(ctx, pool, set)
		if err != nil {
			failPhase(t, "verify "+set.name+" migration history", err)
		}

		if history.latestVersion != state.currentVersion {
			t.Fatalf(
				"%s history latest version = %d, want %d",
				set.name,
				history.latestVersion,
				state.currentVersion,
			)
		}

		if history.appliedRows < int64(state.statusCount) {
			t.Fatalf(
				"%s history applied rows = %d, want at least %d",
				set.name,
				history.appliedRows,
				state.statusCount,
			)
		}
	}
}

func readMigrationHistory(
	ctx context.Context,
	pool *pgxpool.Pool,
	set migrationSet,
) (migrationHistory, error) {
	var history migrationHistory

	table := pgx.Identifier{
		set.historySchema,
		set.historyTable,
	}.Sanitize()

	query := fmt.Sprintf(`
		SELECT
			count(*) FILTER (WHERE is_applied),
			max(version_id) FILTER (WHERE is_applied)
		FROM %s
	`, table)

	var latest pgtype.Int8

	if err := pool.QueryRow(ctx, query).Scan(
		&history.appliedRows,
		&latest,
	); err != nil {
		return history, fmt.Errorf(
			"read history table %s: %w",
			set.qualifiedHistoryTable(),
			err,
		)
	}

	if !latest.Valid {
		return history, fmt.Errorf(
			"history table %s has no applied version",
			set.qualifiedHistoryTable(),
		)
	}

	history.latestVersion = latest.Int64

	return history, nil
}
