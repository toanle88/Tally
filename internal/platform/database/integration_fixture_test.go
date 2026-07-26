//go:build integration

package database

import (
	"context"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/toanle88/Tally/internal/platform/database/platformdb"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const (
	postgresIntegrationImage = "postgres:18.4-bookworm"

	integrationDatabase = "tally_test"
	integrationUsername = "tally_test"
	integrationPassword = "local-integration-only"

	connectionTimeout = 10 * time.Second
)

type integrationFixture struct {
	databaseURL   string
	migrationSets []migrationSet
	pool          *pgxpool.Pool
}

func startIntegrationFixture(
	t *testing.T,
	ctx context.Context,
) *integrationFixture {
	t.Helper()

	container, err := postgres.Run(
		ctx,
		postgresIntegrationImage,
		postgres.WithDatabase(integrationDatabase),
		postgres.WithUsername(integrationUsername),
		postgres.WithPassword(integrationPassword),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		failPhase(t, "start PostgreSQL 18 container", err)
	}

	testcontainers.CleanupContainer(t, container)

	databaseURL, err := container.ConnectionString(
		ctx,
		"sslmode=disable",
	)
	if err != nil {
		failPhase(t, "resolve PostgreSQL container endpoint", err)
	}

	root := repositoryRoot(t)

	fixture := &integrationFixture{
		databaseURL: databaseURL,
		migrationSets: []migrationSet{
			{
				name:          "bootstrap",
				directory:     filepath.Join(root, "db", "migrations", "bootstrap"),
				historySchema: "public",
				historyTable:  "goose_bootstrap_db_version",
			},
			{
				name:          "platform",
				directory:     filepath.Join(root, "db", "migrations", "platform"),
				historySchema: "platform",
				historyTable:  "goose_db_version",
			},
		},
	}

	t.Cleanup(func() {
		if fixture.pool != nil {
			fixture.pool.Close()
		}
	})

	return fixture
}

func (f *integrationFixture) openPool(
	t *testing.T,
	ctx context.Context,
) {
	t.Helper()

	if f.pool != nil {
		t.Fatal("platform pgx pool is already open")
	}

	pool, err := Open(ctx, Config{
		DatabaseURL:    f.databaseURL,
		MaxConnections: 4,
		ConnectTimeout: connectionTimeout,
	})
	if err != nil {
		failPhase(t, "open platform pgx pool", err)
	}

	f.pool = pool
}

func (f *integrationFixture) queries() *platformdb.Queries {
	if f.pool == nil {
		panic("integration fixture pool is not open")
	}

	return platformdb.New(f.pool)
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
		failPhase(t, "query PostgreSQL server version", err)
	}

	version, err := strconv.Atoi(versionText)
	if err != nil {
		failPhase(t, "parse PostgreSQL server version", err)
	}

	if major := version / 10000; major != want {
		t.Fatalf("PostgreSQL major version = %d, want %d", major, want)
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

func repositoryRoot(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test source location")
	}

	return filepath.Clean(
		filepath.Join(filepath.Dir(filename), "..", "..", ".."),
	)
}

func failPhase(t *testing.T, phase string, err error) {
	t.Helper()

	if err == nil {
		t.Fatalf("%s failed without an error", phase)
	}

	message := redactIntegrationSecrets(err.Error())

	t.Fatalf("%s failed: %s", phase, message)
}

func redactIntegrationSecrets(message string) string {
	secrets := []string{
		integrationPassword,
		integrationUsername,
	}

	for _, secret := range secrets {
		if strings.TrimSpace(secret) == "" {
			continue
		}

		message = strings.ReplaceAll(
			message,
			secret,
			"[redacted]",
		)
	}

	return message
}
