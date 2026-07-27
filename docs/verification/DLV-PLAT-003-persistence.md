# DLV-PLAT-003 Persistence Workflow Verification

> This record proves the focused persistence workflow only. It does not prove
> `DLV-CI-001`, full `QG-03`, production recovery, or milestone `M0`.

## Environment

- Commit: `ad9a1bd`
- Verification date: 2026-07-27
- Operating system: WSL on `DESKTOP-FKJFA38`
- Go version: not included in pasted transcript
- Docker version: daemon available; exact version not included in pasted transcript
- PostgreSQL image/version: `postgres:18.4-bookworm`
- Goose version: `v3.27.1`
- sqlc version: `v1.31.1`
- pgx module version: `github.com/jackc/pgx/v5 v5.10.0`
- Testcontainers module version: `github.com/testcontainers/testcontainers-go v0.43.0`

## Persistence Contract

- Initialized migration schemas: `bootstrap`, `platform`
- Migration history tables: `goose_bootstrap_db_version`,
  `platform.goose_db_version`
- Migration inventory/checksum method: `db/migrations/checksums.sha256`
- sqlc configuration path: `sqlc.yaml`
- Query source paths: `db/queries/platform/`
- Generated output paths: `internal/platform/database/platformdb/`
- Root persistence-check command: `make persistence-check`
- CI workflow: `.github/workflows/persistence.yml`
- CI job: `Persistence drift`

## Positive Verification

Commands executed from `/mnt/d/Dev/Projects/Tally`:

1. `make persistence-tools-version`
2. `make sqlc-compile`
3. `make sqlc-check`
4. `make persistence-check`
5. `make check`
6. `make verify-database`

Results:

- Pinned-tool verification: passed; Goose `v3.27.1`, sqlc `v1.31.1`
- Migration validation: passed for `db/migrations/bootstrap` and
  `db/migrations/platform`
- Migration inventory/checksum verification: passed
- sqlc compile: passed
- Clean sqlc drift check: passed
- Go compilation and unit tests: passed with `go test ./...`
- Clean persistence integration: passed for
  `github.com/toanle88/Tally/internal/platform/database` in 4.271s
- Clean migration application: passed
- Second migration application: passed; Goose reported no pending migrations
  and current version `1` for both initialized migration sets
- Migration status: bootstrap migration
  `00001_create_platform_schema.sql` and platform migration
  `00001_create_local_seed_manifest.sql` applied
- Generated query integration: passed through the platform database integration
  test package
- Transaction commit: passed through the platform database integration test
  package
- Transaction rollback: passed through the platform database integration test
  package
- PostgreSQL cleanup: passed; no cleanup failure was reported
- Working tree unchanged after successful check: passed by
  `make persistence-check`
- Real credentials required: No
- Azure or external services required: No

## Controlled Negative Verification

The pasted transcript does not include the temporary mutation outputs for:

- Historical migration drift
- Stale sqlc output
- Manual generated-file edit/delete
- Invalid migration or generated-code compile error
- CI migration-drift failure
- CI generated-code-drift failure

Record those outputs here when they are captured.
