# DLV-PLAT-007 User Story 2 Verification

This record covers the durable PostgreSQL outbox and inbox persistence
foundation. It does not claim transactional source/consumer coordination,
workers, retries, stale-worker fencing, replay, or crash/database-recovery
fault injection from later DLV-PLAT-007 stories.

Transactional source/consumer coordination is verified separately in
`DLV-PLAT-007-transactional-coordination.md`.

## Evidence scope

- `integration.outbox` and `integration.inbox` are created by the platform
  Goose migration set.
- Named primary-key, uniqueness, state, and dispatch-index constraints are
  present.
- sqlc-generated models and queries support insertion, source-identity lookup,
  due claiming, expired-lease lookup, inbox lookup, and reconciliation lookup.
- PostgreSQL integration tests cover durable commit/reopen behavior, rollback,
  duplicate constraints, state validation, ordering, and concurrent claims.

## Verification command

```bash
make outbox-inbox-persistence-check
```

The focused command verifies migration checksums and syntax, sqlc compilation
and generated-output drift, the PostgreSQL 18 Testcontainers integration test,
and whitespace errors.

The complete repository persistence gate remains:

```bash
make persistence-check
```

## Positive verification

Verified on 2026-08-23 from branch
`feat/dlv-plat-007-us2-outbox-inbox-persistence`:

- `make outbox-inbox-persistence-check` passed.
- `make persistence-check` passed.
- Goose `v3.27.1` and sqlc `v1.31.1` checks passed.
- Migration validation and checksum verification passed.
- sqlc compilation and generated-output drift checks passed.
- `go test ./...` passed.
- PostgreSQL 18 Testcontainers persistence integration tests passed.
- Concurrent outbox claims returned distinct rows.
- Committed outbox/inbox rows remained readable after closing and reopening the
  PostgreSQL connection pool.
- Persistence verification left the Git working tree unchanged during its
  reproducibility check.

## Evidence basis

- `db/migrations/platform/00003_create_integration_outbox_inbox.sql`
- `db/queries/platform/integration_outbox.sql`
- `db/queries/platform/integration_inbox.sql`
- `internal/platform/database/platformdb/`
- `internal/platform/database/outbox_inbox_integration_test.go`
- `scripts/verify/outbox-inbox-persistence.sh`
