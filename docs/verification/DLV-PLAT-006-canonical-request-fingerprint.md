# DLV-PLAT-006 User Story 1 Verification

This record verifies canonical functional request projection serialization and
fingerprinting only. It does not claim scoped identity, result metadata,
transaction, retry, conflict, concurrency, recovery, outbox/inbox, or finance
workflow behavior.

## Contract

- Functional projections are UTF-8 JSON objects with compact canonical output.
- Object keys are decoded and sorted by UTF-8 bytes; arrays preserve order.
- Duplicate keys, unsupported numbers, malformed strings, lone surrogates, and
  nesting beyond 256 levels are rejected.
- Hash input is `tally-request-fingerprint:v1`, one NUL byte, then canonical
  JSON; the result is lowercase SHA-256 with a `sha256:` prefix.

## Verification commands

```bash
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./internal/platform/idempotency
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go vet ./internal/platform/idempotency
make request-fingerprint-check
make api-check
GOCACHE=/tmp/tally-go-cache-dlv-plat-006 go test ./...
git diff --check
```

## Evidence

Tests cover whitespace and object-order equivalence, escaped keys and ordinary
strings, UTF-8 byte ordering, surrogate pairs, duplicate keys, invalid numbers,
material changes, golden canonical/hash bytes, and accepted/rejected depth
boundaries. The package ownership audit rejects finance-context and
infrastructure imports; OpenAPI drift checks confirm this foundation does not
alter API artifacts.

Verification basis: `HEAD` `4eee428` plus the staged working-tree changes.
No commit was created because repository instructions prohibit committing
unless explicitly requested.

Command evidence:

- Focused tests: passed (`ok`, package `internal/platform/idempotency`).
- Focused vet: passed (exit status 0).
- `make request-fingerprint-check`: passed.
- `git diff --check`: passed; Git emitted only an unrelated existing CRLF
  normalization warning for `DLV-PLAT-003_user_stories.md`.
- `make api-check`: not completed in the restricted environment.
- `go test ./...`: previously failed because sandbox networking forbids local
  TCP listeners in database tests; not claimed as passed.
