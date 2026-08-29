# DLV-PLAT-007 User Story 5 — Worker Lifecycle, Recovery, and Replay

## Scope

This checkpoint covers the platform worker host, PostgreSQL dispatcher polling,
crash-equivalent transaction boundaries, duplicate delivery, lease recovery,
and generation-scoped replay. It does not claim finance capability handlers,
semantic payload minimization, external broker behavior, or finance-level
exactly-once effects.

## Verification matrix

| Acceptance criterion | Evidence |
|---|---|
| Worker lifecycle, limits, budgets, namespaces, and bounded shutdown | `internal/platform/worker/host_test.go`, worker gate unit/race/vet checks |
| Source/consumer crash recovery | `TestCrashRecoveryCommitBoundaries` plus existing transactional coordination integration tests |
| Concurrent claims, leases, fencing, restart, retry exhaustion, poison work | Existing outbox/inbox and dispatch integration tests |
| Generic duplicate, invalid, and changed-fingerprint boundaries | Existing dispatcher/coordinator integration tests; capability-owned delayed, out-of-order, and prerequisite behavior remains deferred |
| Replay range, identity, generation isolation, retained inbox evidence, and no duplicate effects | `TestReplayPreservesIdentityEvidenceAndAvoidsDuplicateEffects` |
| Immutable outbox event facts | `TestOutboxEventFactsAreImmutableButOperationalFieldsChange` and migration guard |
| Package and generated-artifact consistency | `make outbox-worker-check` |

## Commands

```bash
make outbox-worker-check
GOCACHE=/tmp/tally-go-cache-dlv-plat-007-us5 go test ./...
GOCACHE=/tmp/tally-go-cache-dlv-plat-007-us5 go vet ./...
```

PostgreSQL integration tests require Docker/Testcontainers and local network
binding. The verification command must report the command output and preserve
the migration checksum and SQLC drift checks.
