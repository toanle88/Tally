# DLV-PLAT-007 User Story 3 — Transactional Coordination Verification

## Verification status

Verified on 2026-08-24 from branch
`feat/dlv-plat-007-us3-transactional-coordination`.

## Evidence scope

- Source effects and their outbox publications commit atomically.
- Receiving inbox establishment, local effects, and resulting publications
  commit atomically.
- Same-fingerprint established deliveries return the stored result without
  invoking the local effect again.
- Changed fingerprints return a typed identity-content conflict and preserve
  the established evidence.
- Effect failures roll back local writes and retain failed inbox evidence.
- Failed inbox evidence is retained even when the consumer context is cancelled.
- Outbox publications preserve the event occurrence time and data classification
  alongside the payload and identity metadata.
- Reconciliation establishes an already-existing local result without replaying
  the effect, and retries only when no established local result exists.
- The platform package contains no finance bounded-context ownership, accounting,
  authorization, or audit policy.

## Verification command

```bash
make transactional-coordination-check
```

The focused gate runs unit tests, race tests, vet, package ownership checks,
sqlc compilation and drift checks, Docker-backed PostgreSQL integration tests,
OpenAPI preservation checks, and `git diff --check`.

## Evidence basis

- `internal/platform/integration/coordination.go`
- `internal/platform/integration/coordination_test.go`
- `internal/platform/database/transactional_coordination_integration_test.go`
- `db/queries/platform/integration_inbox.sql`
- `db/migrations/platform/00004_add_outbox_envelope_metadata.sql`
- `db/queries/platform/integration_outbox.sql`
- `internal/platform/database/platformdb/integration_inbox.sql.go`
- `scripts/verify/transactional-coordination.sh`

## Non-claims

This verification does not claim semantic payload minimization, outbox leases,
typed retries, worker lifecycle, poison-work management, ordering, replay, or
finance-level exactly-once business effects. Those remain User Stories 1
follow-up scope, 4, 5, or owning bounded-context scope.
