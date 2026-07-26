//go:build integration

package database

import (
	"context"
	"testing"
	"time"
)

func TestPersistenceWorkflow(t *testing.T) {
	ctx, cancel := context.WithTimeout(
		context.Background(),
		4*time.Minute,
	)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)

	var firstMigrationStates map[string]migrationState

	if !t.Run("apply migrations to clean PostgreSQL", func(t *testing.T) {
		firstMigrationStates = applyAndVerifyMigrations(
			t,
			ctx,
			fixture.databaseURL,
			fixture.migrationSets,
		)
	}) {
		return
	}

	if !t.Run("open the platform pgx pool", func(t *testing.T) {
		fixture.openPool(t, ctx)

		assertPostgresMajorVersion(
			t,
			ctx,
			fixture.pool,
			18,
		)
	}) {
		return
	}

	if !t.Run("verify schema-owned migration history", func(t *testing.T) {
		verifyMigrationHistory(
			t,
			ctx,
			fixture.pool,
			fixture.migrationSets,
			firstMigrationStates,
		)
	}) {
		return
	}

	if !t.Run("execute generated query and commit", func(t *testing.T) {
		proveGeneratedQueryCommit(
			t,
			ctx,
			fixture.pool,
			fixture.queries(),
		)
	}) {
		return
	}

	if !t.Run("execute generated query and roll back", func(t *testing.T) {
		proveGeneratedQueryRollback(
			t,
			ctx,
			fixture.pool,
			fixture.queries(),
		)
	}) {
		return
	}

	if !t.Run("reapply migrations without pending work", func(t *testing.T) {
		secondMigrationStates := applyAndVerifyMigrations(
			t,
			ctx,
			fixture.databaseURL,
			fixture.migrationSets,
		)

		assertMigrationReapplicationIsHarmless(
			t,
			firstMigrationStates,
			secondMigrationStates,
		)
	}) {
		return
	}

	t.Run("verify final persistence state", func(t *testing.T) {
		assertCommittedRowExists(
			t,
			ctx,
			fixture.queries(),
			commitSeedVersion,
		)

		assertRolledBackRowDoesNotExist(
			t,
			ctx,
			fixture.queries(),
			rollbackSeedVersion,
		)

		assertNoAcquiredConnections(t, fixture.pool)
	})
}
