//go:build integration

package database

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/toanle88/Tally/internal/platform/database/platformdb"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	commitSeedName      = "integration-commit"
	commitSeedVersion   = int64(101)
	rollbackSeedName    = "integration-rollback"
	rollbackSeedVersion = int64(102)
)

func proveGeneratedQueryCommit(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *platformdb.Queries,
) {
	t.Helper()

	tx, err := pool.Begin(ctx)
	if err != nil {
		failPhase(t, "begin commit transaction", err)
	}
	defer rollbackForCleanup(tx)

	checksum := testChecksum(commitSeedVersion)

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO platform.local_seed_manifest (
				seed_name,
				seed_version,
				checksum
			)
			VALUES ($1, $2, $3)
		`,
		commitSeedName,
		commitSeedVersion,
		checksum,
	)
	if err != nil {
		failPhase(t, "insert committed technical manifest", err)
	}

	gotChecksum, err := queries.WithTx(tx).
		GetLocalSeedManifestChecksum(
			ctx,
			platformdb.GetLocalSeedManifestChecksumParams{
				SeedName:    commitSeedName,
				SeedVersion: commitSeedVersion,
			},
		)
	if err != nil {
		failPhase(t, "query manifest inside commit transaction", err)
	}

	assertManifest(
		t,
		commitSeedVersion,
		gotChecksum,
		commitSeedVersion,
		checksum,
	)

	if err := tx.Commit(ctx); err != nil {
		failPhase(t, "commit generated-query transaction", err)
	}

	establishedChecksum, err := queries.GetLocalSeedManifestChecksum(
		ctx,
		platformdb.GetLocalSeedManifestChecksumParams{
			SeedName:    commitSeedName,
			SeedVersion: commitSeedVersion,
		},
	)
	if err != nil {
		failPhase(t, "query committed technical manifest", err)
	}

	assertManifest(
		t,
		commitSeedVersion,
		establishedChecksum,
		commitSeedVersion,
		checksum,
	)
}

func proveGeneratedQueryRollback(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	queries *platformdb.Queries,
) {
	t.Helper()

	tx, err := pool.Begin(ctx)
	if err != nil {
		failPhase(t, "begin rollback transaction", err)
	}
	defer rollbackForCleanup(tx)

	checksum := testChecksum(rollbackSeedVersion)

	_, err = tx.Exec(
		ctx,
		`
			INSERT INTO platform.local_seed_manifest (
				seed_name,
				seed_version,
				checksum
			)
			VALUES ($1, $2, $3)
		`,
		rollbackSeedName,
		rollbackSeedVersion,
		checksum,
	)
	if err != nil {
		failPhase(t, "insert rollback technical manifest", err)
	}

	gotChecksum, err := queries.WithTx(tx).
		GetLocalSeedManifestChecksum(
			ctx,
			platformdb.GetLocalSeedManifestChecksumParams{
				SeedName:    rollbackSeedName,
				SeedVersion: rollbackSeedVersion,
			},
		)
	if err != nil {
		failPhase(t, "query manifest inside rollback transaction", err)
	}

	assertManifest(
		t,
		rollbackSeedVersion,
		gotChecksum,
		rollbackSeedVersion,
		checksum,
	)

	if err := tx.Rollback(ctx); err != nil {
		failPhase(t, "roll back generated-query transaction", err)
	}

	_, err = queries.GetLocalSeedManifestChecksum(
		ctx,
		platformdb.GetLocalSeedManifestChecksumParams{
			SeedName:    rollbackSeedName,
			SeedVersion: rollbackSeedVersion,
		},
	)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"query rolled-back manifest error = %v, want pgx.ErrNoRows",
			err,
		)
	}
}

func assertCommittedRowExists(
	t *testing.T,
	ctx context.Context,
	queries *platformdb.Queries,
	seedVersion int64,
) {
	t.Helper()

	checksum, err := queries.GetLocalSeedManifestChecksum(
		ctx,
		platformdb.GetLocalSeedManifestChecksumParams{
			SeedName:    commitSeedName,
			SeedVersion: seedVersion,
		},
	)
	if err != nil {
		failPhase(t, "verify committed technical manifest", err)
	}

	assertManifest(
		t,
		seedVersion,
		checksum,
		seedVersion,
		testChecksum(seedVersion),
	)
}

func assertRolledBackRowDoesNotExist(
	t *testing.T,
	ctx context.Context,
	queries *platformdb.Queries,
	seedVersion int64,
) {
	t.Helper()

	_, err := queries.GetLocalSeedManifestChecksum(
		ctx,
		platformdb.GetLocalSeedManifestChecksumParams{
			SeedName:    rollbackSeedName,
			SeedVersion: seedVersion,
		},
	)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf(
			"rolled-back manifest lookup error = %v, want pgx.ErrNoRows",
			err,
		)
	}
}

func rollbackForCleanup(tx pgx.Tx) {
	cleanupCtx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	_ = tx.Rollback(cleanupCtx)
}

func testChecksum(value int64) string {
	sum := sha256.Sum256([]byte(strconv.FormatInt(value, 10)))
	return hex.EncodeToString(sum[:])
}

func assertManifest(
	t *testing.T,
	gotVersion int64,
	gotChecksum string,
	wantVersion int64,
	wantChecksum string,
) {
	t.Helper()

	if gotVersion != wantVersion {
		t.Fatalf(
			"manifest seed version = %d, want %d",
			gotVersion,
			wantVersion,
		)
	}

	if gotChecksum != wantChecksum {
		t.Fatalf(
			"manifest checksum = %q, want %q",
			gotChecksum,
			wantChecksum,
		)
	}
}
