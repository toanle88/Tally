# DLV-PLAT-006 Durable Coordination Verification

This record covers the PostgreSQL-backed idempotency reservation and
finalization layer. It does not claim completion of the owning GL posting
effect, production authorization policy, or the full Audit Integrity
capability.

## Evidence scope

- Durable identities are unique by scope and idempotency key.
- Same-fingerprint terminal results are returned without a second owner.
- Changed fingerprints return `ErrIdempotencyConflict` without mutation.
- Owner tokens and leases support safe takeover after an expired reservation.
- Terminal result finalization is guarded by the owner token.
- Reservation and finalization use explicit PostgreSQL transaction boundaries.
- Independent PostgreSQL connections establish one reservation owner.
- Migration checksum and SQLC schema checks are part of the persistence gate.

## Verification commands

```bash
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 make idempotency-check
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 make idempotency-persistence-check
```

The persistence command requires Docker for PostgreSQL Testcontainers.

The current restricted environment has not completed the Docker-backed
integration gate: database tests requiring local TCP listeners are blocked by
the environment, and the repository SQLC check requires the pinned generator
toolchain. These are release-gate limitations, not evidence of a passing run.

## Evidence basis

- `internal/platform/idempotency/durable.go`
- `internal/platform/idempotency/durable_test.go`
- `internal/platform/database/idempotency_integration_test.go`
- `db/migrations/platform/00002_create_idempotency_record.sql`
- `scripts/verify/idempotency-persistence.sh`

## Follow-up capability scope

The following are intentionally outside DLV-PLAT-006 and must be verified by
the owning capability delivery items:

- A real owning GL posting transaction and immutable journal effect.
- Transaction-participating authorization and Audit Integrity implementations.
- Public handler-level `IDEMPOTENCY_CONFLICT` response verification.
- Commit ambiguity and cross-process recovery evidence.
