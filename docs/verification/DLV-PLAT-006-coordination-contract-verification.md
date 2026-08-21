# DLV-PLAT-006 Coordination-Contract Verification

This record verifies the contract-only coordination and commit-boundary test
scope. It does not claim PostgreSQL persistence, cross-process coordination,
owning-capability transaction integration, or exactly-once financial/business
effects.

## Evidence scope

- Concurrent first acquisition has one execution owner.
- Same-fingerprint retries return `in_progress`, `established`, or `failed`.
- Changed fingerprints return `ErrIdempotencyConflict` without mutation.
- Test-only staged effects and result bookkeeping are published by the harness
  commit operation.
- Rollback leaves the coordinator in progress and publishes no staged effect.
- Callback failure and panic do not create false terminal success.
- Identity, fingerprint, defensive-copy, ownership, and OpenAPI-preservation checks remain covered.

The transaction harness is test-only bookkeeping. Its commit operation invokes
the coordinator finalizer and then records the simulated effect; it does not
provide rollback of arbitrary callbacks and does not model PostgreSQL locking,
durability, connection loss, failover, or cross-process visibility.

## Verification commands

```bash
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 make idempotency-check
git diff --check
```

## Evidence basis

- `internal/platform/idempotency/coordinator.go`
- `internal/platform/idempotency/coordinator_test.go`
- `internal/platform/idempotency/transaction_harness_test.go`
- `scripts/verify/idempotency.sh`

## Deferred acceptance

User Story 5 and DLV-PLAT-006 remain open for PostgreSQL persistence,
owning-business-transaction integration, recovery after database failure, and
finance-level exactly-once business-effect proof.
