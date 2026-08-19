# DLV-PLAT-006 User Story 2 Verification

This record verifies scoped idempotency identity and stored command-result
metadata only. It does not claim retry execution, conflict handling,
transaction atomicity, concurrency, recovery, outbox/inbox, or finance
workflow behavior.

## Contract

- Identity equality is `(AccountingScope, IdempotencyKey)`.
- `ScopeKey` is the compact JSON representation of the validated accounting
  scope.
- Keys and operation IDs are valid UTF-8, non-empty, unpadded, non-control
  text with a maximum size of 255 UTF-8 bytes.
- Result states are `in_progress`, `established`, and `failed`.
- Result metadata retains the fingerprint, operation ID, result state, result
  status/body metadata, and optional aggregate/process references without
  owning finance state.
- No new JSON wire contract or persistence migration is introduced.

## Verification commands

```bash
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./internal/platform/idempotency
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go vet ./internal/platform/idempotency
make request-fingerprint-check
make shared-primitives-check
make api-check
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./...
git diff --check
awk '/^```/{count++} END { exit count % 2 != 0 }' docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md
```

## Evidence

Focused tests cover scope/key equality, cross-scope separation, invalid and
boundary key values, Unicode preservation, canonical scope-key output,
lifecycle values, required metadata, valid/invalid result bodies, optional
references, defensive copying, and ownership boundaries.

Verification basis: branch `feat/dlv-plat-006-scoped-idempotency-metadata`,
`HEAD` `e2f818f198bd2ed6d470750b7d972f01264515b3`. Working-tree changes are:
`ROADMAP.md`, `docs/backlog/stories/DLV-PLAT-006_user_stories.md`,
`docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md`,
`docs/verification/DLV-PLAT-006-scoped-idempotency-metadata.md`,
`internal/platform/idempotency/metadata.go`, and
`internal/platform/idempotency/metadata_test.go`.

Command evidence:

- Focused tests: passed (`ok`, package `internal/platform/idempotency`).
- Focused vet: passed (exit status 0).
- `GOCACHE=/tmp/tally-go-cache-dlv-plat-006 make request-fingerprint-check`:
  passed.
- `make shared-primitives-check`: passed.
- `make api-check`: passed; no OpenAPI or generated-artifact changes were
  present in the working-tree diff.
- `GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./...`: blocked by the
  restricted environment denying local TCP listeners in
  `internal/platform/database` tests (`operation not permitted`). The
  idempotency and other non-network packages passed.
- `git diff --check`: passed.
- Persistence-specification Markdown fence balance: passed (8 fences).

An initial unscoped `make request-fingerprint-check` attempted to use the
environment's read-only default Go cache and failed; the prescribed
task-specific `GOCACHE` rerun passed.
