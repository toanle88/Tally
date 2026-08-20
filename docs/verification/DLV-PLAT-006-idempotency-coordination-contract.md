# DLV-PLAT-006 Idempotency Coordination-Contract Verification

This record verifies the coordination-contract prerequisite only. It does not
claim completion of User Story 3, PostgreSQL persistence, transaction
atomicity, rollback/recovery, cross-process coordination, or exactly-once
financial/business effects.

## Contract evidence

- A new scoped identity and fingerprint produce one execution owner.
- Same-fingerprint retries return the existing `in_progress`, `established`,
  or `failed` result without a second owner.
- Mismatched fingerprints return `ErrFingerprintMismatch` without mutation;
  stable HTTP/API conflict mapping remains User Story 4 scope.
- Finalization uses explicit `active`, `finalizing`, and `terminal` state
  transitions.
- Retries during finalization observe `in_progress`.
- Callback failure or recovered panic restores `in_progress`.
- Returned metadata and response bodies are defensively copied.

## Deferred acceptance

The following User Story 3 criteria remain open:

- Real coordination with an owning business transaction.
- Proof that an identical retry cannot commit a second business effect.

User Stories 4 and 5 remain open. DLV-PLAT-006 remains open.

## Verification commands

```bash
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./internal/platform/idempotency
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go vet ./internal/platform/idempotency
make request-fingerprint-check
make shared-primitives-check
make api-check
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./...
git diff --check
```

## Evidence basis

Focused tests are in:

- `internal/platform/idempotency/coordinator.go`
- `internal/platform/idempotency/coordinator_test.go`

The full repository suite must be reported separately. If database tests
cannot open local listeners in the execution environment, the full suite is
environment-blocked and must not be reported as passed.

## Command evidence

- Focused tests: passed.
- Focused race tests: passed.
- Focused vet: passed.
- `GOCACHE=/tmp/tally-go-cache-dlv-plat-006 make request-fingerprint-check`:
  passed.
- `make shared-primitives-check`: passed.
- `make api-check`: passed.
- `git diff --check`: passed.
- `GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./...`: blocked by the
  restricted environment denying local TCP listeners in
  `internal/platform/database` tests. Non-network packages passed.
- An initial unscoped request-fingerprint check used the read-only default Go
  cache; the task-specific `GOCACHE` rerun passed.
