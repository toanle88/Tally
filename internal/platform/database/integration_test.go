package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresIntegrationImage  = "postgres:18.4-bookworm"
	integrationDatabase       = "tally_test"
	integrationUsername       = "tally_test"
	integrationMaxConnections = int32(4)
)

func TestPostgres18ConnectionCommitRollbackAndCleanup(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	container := startPostgres18Container(t, ctx)
	pool := openIntegrationPool(t, ctx, container)

	assertPoolMaxConnections(t, pool, integrationMaxConnections)
	assertPostgresMajorVersion(t, ctx, pool, 18)

	runTransactionProofs(t, ctx, pool)
	assertNoAcquiredConnections(t, pool)
}

func startPostgres18Container(
	t *testing.T,
	ctx context.Context,
) *postgres.PostgresContainer {
	t.Helper()

	container, err := postgres.Run(
		ctx,
		postgresIntegrationImage,
		postgres.WithDatabase(integrationDatabase),
		postgres.WithUsername(integrationUsername),
		postgres.WithPassword(generateIntegrationPassword(t)),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL 18 integration container: %v", err)
	}

	// Registered after successful startup. Since pool cleanup is registered
	// later, testing cleanup's LIFO order closes the pool first.
	testcontainers.CleanupContainer(t, container)

	return container
}

func generateIntegrationPassword(t *testing.T) string {
	t.Helper()

	var value [24]byte
	if _, err := rand.Read(value[:]); err != nil {
		t.Fatalf("generate PostgreSQL integration password: %v", err)
	}

	return hex.EncodeToString(value[:])
}

func openIntegrationPool(
	t *testing.T,
	ctx context.Context,
	container *postgres.PostgresContainer,
) *pgxpool.Pool {
	t.Helper()

	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve PostgreSQL integration endpoint: %v", err)
	}

	pool, err := Open(ctx, Config{
		DatabaseURL:    databaseURL,
		MaxConnections: integrationMaxConnections,
		ConnectTimeout: 10 * time.Second,
	})
	if err != nil {
		t.Fatalf("open PostgreSQL integration pool: %v", err)
	}

	t.Cleanup(pool.Close)

	return pool
}

func assertPoolMaxConnections(
	t *testing.T,
	pool *pgxpool.Pool,
	want int32,
) {
	t.Helper()

	if got := pool.Config().MaxConns; got != want {
		t.Fatalf("pool maximum connections = %d, want %d", got, want)
	}
}

func assertPostgresMajorVersion(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
	want int,
) {
	t.Helper()

	var versionText string
	if err := pool.QueryRow(ctx, "SHOW server_version_num").Scan(&versionText); err != nil {
		t.Fatalf("query PostgreSQL server version: %v", err)
	}

	version, err := strconv.Atoi(versionText)
	if err != nil {
		t.Fatalf("parse PostgreSQL server version %q: %v", versionText, err)
	}

	if major := version / 10000; major != want {
		t.Fatalf("PostgreSQL major version = %d, want %d", major, want)
	}
}

func runTransactionProofs(
	t *testing.T,
	ctx context.Context,
	pool *pgxpool.Pool,
) {
	t.Helper()

	conn, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatalf("acquire PostgreSQL connection: %v", err)
	}
	defer conn.Release()

	createTransactionProbeTable(t, ctx, conn)
	proveCommittedTransaction(t, ctx, conn)
	proveRolledBackTransaction(t, ctx, conn)
}

func createTransactionProbeTable(
	t *testing.T,
	ctx context.Context,
	conn *pgxpool.Conn,
) {
	t.Helper()

	_, err := conn.Exec(ctx, `
		CREATE TEMP TABLE pgx_transaction_probe (
			probe_value text PRIMARY KEY
		) ON COMMIT PRESERVE ROWS
	`)
	if err != nil {
		t.Fatalf("create transaction probe table: %v", err)
	}
}

func proveCommittedTransaction(
	t *testing.T,
	ctx context.Context,
	conn *pgxpool.Conn,
) {
	t.Helper()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin commit transaction: %v", err)
	}

	// Safe after commit: pgx returns a closed-transaction error, which is
	// intentionally ignored. It protects failure paths before commit.
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(
		ctx,
		"INSERT INTO pgx_transaction_probe (probe_value) VALUES ($1)",
		"committed",
	)
	if err != nil {
		t.Fatalf("insert commit probe: %v", err)
	}

	if err := tx.Commit(ctx); err != nil {
		t.Fatalf("commit probe transaction: %v", err)
	}

	assertProbeCount(t, ctx, conn, "committed", 1)
}

func proveRolledBackTransaction(
	t *testing.T,
	ctx context.Context,
	conn *pgxpool.Conn,
) {
	t.Helper()

	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin rollback transaction: %v", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	_, err = tx.Exec(
		ctx,
		"INSERT INTO pgx_transaction_probe (probe_value) VALUES ($1)",
		"rolled-back",
	)
	if err != nil {
		t.Fatalf("insert rollback probe: %v", err)
	}

	if _, err := tx.Exec(ctx, "SELECT 1 / 0"); err == nil {
		t.Fatal("intentional transaction failure unexpectedly succeeded")
	}

	if err := tx.Rollback(ctx); err != nil {
		t.Fatalf("rollback failed transaction: %v", err)
	}

	assertProbeCount(t, ctx, conn, "rolled-back", 0)
}

func assertProbeCount(
	t *testing.T,
	ctx context.Context,
	conn *pgxpool.Conn,
	probeValue string,
	want int,
) {
	t.Helper()

	var got int
	err := conn.QueryRow(
		ctx,
		`
			SELECT COUNT(*)
			FROM pgx_transaction_probe
			WHERE probe_value = $1
		`,
		probeValue,
	).Scan(&got)
	if err != nil {
		t.Fatalf("query probe value %q: %v", probeValue, err)
	}

	if got != want {
		t.Fatalf("probe count for %q = %d, want %d", probeValue, got, want)
	}
}

func assertNoAcquiredConnections(
	t *testing.T,
	pool *pgxpool.Pool,
) {
	t.Helper()

	if acquired := pool.Stat().AcquiredConns(); acquired != 0 {
		t.Fatalf(
			"acquired pool connections after release = %d, want 0",
			acquired,
		)
	}
}
