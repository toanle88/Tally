# DLV-OPS-001 User Story 4 — Telemetry failure and sensitive-data boundaries

## Scope

This evidence covers `EP-OPS-001` / `DLV-OPS-001` User Story 4 for the
platform telemetry seams that exist in the repository as of 2026-09-21. It
does not claim live Azure export, remote monitoring, production alerting,
telemetry persistence, finance-domain commands, authorization policy, or
completion of `DLV-OPS-002`.

## Source state

- Branch: `codex/dlv-ops-001-us4-telemetry-failure-sensitive-data`
- Baseline commit: `5a89092` (`main` / User Story 3 telemetry baseline)
- Implementation state: working-tree changes; no commit, staging, or push was
  performed.
- Toolchain: repository Go target `1.26.3`; checks used the installed Windows
  Go toolchain through Git Bash because the Linux shell has no `go` binary.

## Implementation

- `internal/platform/telemetry` recognizes injected OpenTelemetry SDK provider
  shutdown capabilities and performs an at-most-once, caller-deadline-bounded
  shutdown. Recording methods remain non-authoritative and do not return
  exporter errors to business callers.
- API and worker composition roots use bounded shutdown contexts and emit only
  the stable `telemetry_shutdown_failed` error code when provider shutdown
  fails. Existing application results are not replaced by telemetry errors.
- Synthetic trace and metric exporters exercise exporter failure, blocking
  timeout, shutdown failure, repeated shutdown, and bounded invocation cases.
- Captured spans and metrics are checked for context continuity, bounded
  labels, and absence of secrets, credentials, bank/payroll/tax values,
  payloads, SQL, raw errors, and unrestricted identifiers.
- No remote exporter, Azure credential, database migration, telemetry table,
  public API schema, event envelope, or finance bounded-context code was added.

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Engineering foundation milestone. |
| `EP-OPS-001` / `DLV-OPS-001` | Parent observability epic and structured telemetry delivery item. |
| `ARC-OBS-001` | Trace, correlation, and causation continuity across technical boundaries. |
| `ARC-OBS-002` | Platform telemetry remains separate from authoritative business metrics. |
| `ARC-PRV-001` / `NFR-SEC-010` | Data minimization and secret/credential exclusion. |
| `NFR-OBS-003`, `NFR-OBS-004`, `NFR-OBS-008` | Safe support references, no sensitive telemetry, and typed outcomes. |
| `NFR-MNT-001`, `NFR-TST-003` | Traceability and verification evidence. |
| `QG-01`, `QG-08`, `QG-10` | Traceability, observability readiness, and release evidence gates. |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `go.exe test ./internal/platform/telemetry ./cmd/api ./cmd/worker ./internal/platform/idempotency ./internal/platform/integration ./internal/platform/worker` | Passed | Focused telemetry, composition-root, integration, idempotency, and worker tests passed. |
| `go.exe test -race ./internal/platform/telemetry ./cmd/api ./cmd/worker ./internal/platform/idempotency ./internal/platform/integration ./internal/platform/worker` | Passed | Focused race tests passed with the installed Windows Go toolchain. |
| `go.exe test ./...` | Passed | Full repository Go test suite passed. |
| `go.exe vet ./...` | Passed | No diagnostics. |
| `bash -n scripts/verify/telemetry-failure-sensitive-data.sh` | Passed | Verification script parses successfully. |
| `GO_BIN="/c/Program Files/Go/bin/go.exe" bash scripts/verify/telemetry-failure-sensitive-data.sh` | Passed | Focused tests, race tests, vet, and diff check passed through Git Bash and the installed Windows Go toolchain. |
| `make GO_BIN="/c/Program Files/Go/bin/go.exe" telemetry-failure-sensitive-data-check` | Passed | Repository-native Make target passed. |
| `gofmt -l` on changed Go files | Passed | Installed Windows formatter reported no unformatted files. |
| `git diff --check` | Passed | No whitespace errors; Git emitted only existing CRLF normalization warnings for repository text files. |

No production data, credentials, raw telemetry payloads, or secrets were used.

## Deferred qualification

Remote exporter availability, Azure Monitor delivery, production retry policy,
retention, alerting, and operational dashboard qualification remain deferred
to the approved operations delivery scope. Local failure injection is the
complete evidence for this story’s no-remote-service boundary.
