# DLV-OPS-001 User Story 3 — Traces and bounded platform metrics

## Scope

This evidence covers technical tracing and the approved bounded platform metric
catalogue at seams that exist in the repository as of 2026-09-21. It does not
claim implementation of authorization policy, finance command handlers,
provider calls, report jobs, live exporters, dashboards, telemetry persistence,
or production retention policy.

## Implementation

- `internal/platform/telemetry` owns the injectable OpenTelemetry tracer/meter
  contract, safe span attributes, finite label registries, HTTP request spans,
  and the six approved platform metrics.
- HTTP tracing runs inside the chi router so canonical route templates are
  available; query strings and raw URLs are never recorded.
- Existing durable idempotency, integration publication/consumption/recovery,
  PostgreSQL transaction, repository, outbox claim/delivery, replay, backlog,
  inbox failure, and worker lifecycle seams are instrumented with stable span
  names and typed outcomes.
- Outbox backlog gauges use one read-only grouped query and generated sqlc
  bindings. No migration, telemetry table, event fact, or finance schema was
  added.
- Labels are normalized through finite registries. Invalid values, opaque UUID
  identifiers, SQL-like/free-form values, and sensitive words collapse to a
  safe bounded category or are omitted. No raw payload, SQL statement, message
  ID, customer ID, credential, token, bank, payroll, tax, or unrestricted error
  is attached to a span or metric.
- API and worker roots create service-scoped instrumentation without installing
  a remote exporter. Provider ownership remains injectable for later operations
  delivery work.

## Trace and metric inventory

| Area | Implemented contract |
|---|---|
| HTTP | `http.request`; `finance_http_request_duration_seconds` by canonical route, method, and status class. |
| Idempotency | `idempotency.lookup` and `idempotency.finalize`. |
| Integration | `outbox.publication`, `inbox.handling`, `inbox.replay`, `recovery.action`, and `repository.operation`. |
| Database | `postgres.transaction`; `finance_db_transaction_duration_seconds` by bounded module, operation, and result. |
| Outbox | `outbox.claim`, `outbox.delivery`, `outbox.backlog`; `finance_outbox_pending_total` and `finance_outbox_oldest_age_seconds` by bounded event type. |
| Inbox failures | `finance_inbox_failure_total` by bounded consumer and normalized error code. |
| Command helper | `finance_command_total` is exposed for current/future command seams; no finance command handler exists in this repository to call it. |
| Deferred components | Authorization, provider, report-job, and finance command-handler spans remain deferred until their owning components exist. |

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Engineering foundation. |
| `EP-OPS-001` / `DLV-OPS-001` | Parent observability epic and delivery item. |
| `ARC-OBS-001` / `ARC-OBS-002` | Technical context continuity and separation of platform/business metric ownership. |
| `ARC-PRV-001` / `NFR-SEC-010` | Telemetry minimization and secret/credential exclusion. |
| `NFR-OBS-003`, `NFR-OBS-004`, `NFR-OBS-008` | Safe diagnostics, no sensitive telemetry, and typed outcomes. |
| `NFR-MNT-001`, `NFR-TST-003` | Source-to-verification traceability and evidence. |
| `QG-01`, `QG-08`, `QG-10` | Traceability, observability readiness, and release evidence gates. |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `go.exe test ./internal/platform/telemetry ./internal/platform/idempotency ./internal/platform/integration ./cmd/api ./cmd/worker` | Passed | Focused seam tests passed after final instrumentation wiring. |
| `go.exe test ./...` | Passed | Full repository Go test suite passed. |
| `go.exe vet ./...` | Passed | No diagnostics. |
| `go.exe tool sqlc -f sqlc.yaml compile` | Passed | Updated query source compiles against the repository schema. |
| `go.exe tool sqlc -f sqlc.yaml diff` | Passed | Generated platform bindings are deterministic and current. |
| `bash -n scripts/verify/traces-metrics.sh` | Passed | Verification script parses successfully. |
| `GO_BIN="/mnt/c/Program Files/Go/bin/go.exe" bash scripts/verify/traces-metrics.sh` | Partially passed | Focused tests pass; the race stage stops with `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. |
| `go.exe test -race ./internal/platform/telemetry ./internal/platform/idempotency ./internal/platform/integration ./cmd/api ./cmd/worker` | Blocked | Available Windows toolchain reports `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. |
| `make traces-metrics-check` | Blocked | Repository shell reports `go: command not found`; the focused script can use `GO_BIN` with the available toolchain. |
| `git diff --check` | Passed | No whitespace errors; Git emitted only existing CRLF normalization warnings for repository text files. |

No production data, credentials, raw telemetry payloads, or secrets were used.
