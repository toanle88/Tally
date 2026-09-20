# DLV-OPS-001 User Story 2 — Redacted structured logs

## Scope

This evidence covers structured JSON logging and deterministic redaction for
the current API and worker composition roots. It does not claim completion of
trace instrumentation, metrics, exporters, dashboards, authorization policy,
finance modules, telemetry persistence, or production monitoring.

## Implementation

- `internal/platform/telemetry` owns the strict `slog` JSON handler, stable
  event fields, telemetry-context fields, typed diagnostic references, and
  safe HTTP request logging.
- `cmd/api` and `cmd/worker` use stable lifecycle messages and error codes
  instead of `log.Printf` and never serialize raw errors.
- Unknown/grouped attributes and sensitive values are rejected by the handler;
  request URLs, headers, bodies, SQL, events, responses, and credentials are
  not logged.
- No finance bounded-context package, public API schema, database migration,
  event envelope, or telemetry persistence was changed.

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Engineering foundation |
| `EP-OPS-001` | Parent observability epic |
| `DLV-OPS-001` | Structured telemetry foundation |
| `ARC-OBS-001` | Technical correlation across work boundaries |
| `NFR-OBS-003` | Stable, sanitized support diagnostics |
| `NFR-OBS-004` | No unmasked sensitive telemetry values |
| `NFR-OBS-008` | Distinguishable operational outcomes |
| `NFR-SEC-010` | Secrets and credentials excluded from logs |
| `NFR-MNT-001`, `NFR-TST-003` | Traceability and verification evidence |
| `QG-08` | Observability and operational readiness |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `go.exe test ./internal/platform/telemetry ./cmd/api ./cmd/worker` | Passed | Focused packages passed. |
| `go.exe vet ./internal/platform/telemetry ./cmd/api ./cmd/worker` | Passed | No diagnostics. |
| `go.exe test ./...` | Passed | All repository Go packages passed. |
| `bash -n scripts/verify/structured-logs.sh` | Passed | Verification script parses successfully. |
| `GO_BIN=".../go.exe" bash scripts/verify/structured-logs.sh` | Blocked | Focused tests passed; race stage requires CGO. |
| `go.exe test -race ./internal/platform/telemetry ./cmd/api ./cmd/worker` | Blocked | `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. |
| `make structured-logs-check` | Blocked | Repository shell has no `go` executable. |
| `git diff --check` | Passed | No whitespace errors. |

## Strict review

The current branch `feat/dlv-ops-001-us2-redacted-structured-logs-main` was
reviewed against `origin/main` using the repository's strict review criteria.
Review status is `APPROVE` at `91/100`, with no blocker, high, or medium
findings. Non-blocking suggestions are to preserve optional
`http.ResponseWriter` capabilities in the request wrapper and add an
integration-style API lifecycle-failure test.

The review confirms that no public API schema, database migration, finance
module, or roadmap completion status was changed by User Story 2.

No production data, credentials, raw telemetry payloads, or secrets were used.
