# DLV-OPS-001 User Story 1 — Telemetry context and propagation

## Scope

This evidence covers the runtime-only telemetry context and propagation
foundation. It carries identifiers only and does not add telemetry persistence,
structured logging, span creation/export, metrics, or dashboard behavior.

The approved async-boundary decision is runtime context first: the existing
event envelope and outbox schema continue to persist `correlationId` and
`causationId`; durable `trace_id`/`span_id` metadata remains deferred to later
trace instrumentation work.

## Implementation

- `internal/platform/telemetry` defines validated identifier context, HTTP
  correlation handling, and W3C trace-context extraction/injection.
- `cmd/api` installs the middleware at the composition root.
- `internal/platform/integration` hydrates transaction, inbox, replay, and
  dispatcher handler contexts from the existing event envelope.
- No finance bounded-context package, event envelope, generated API artifact,
  migration, or database table was changed.

## Traceability

| Identifier | Relationship |
|---|---|
| `M0` | Engineering foundation |
| `EP-OPS-001` | Parent observability epic |
| `DLV-OPS-001` | Structured telemetry foundation |
| `ARC-OBS-001` | Technical correlation across work boundaries |
| `ARC-API-001` | Existing HTTP correlation convention |
| `ARC-EVT-001`–`ARC-EVT-003` | Outbox, inbox, ordering, and replay boundaries |
| `NFR-OBS-003`–`NFR-OBS-004` | Support references and safe diagnostics |
| `NFR-OBS-008` | Distinguishable technical outcome context |
| `NFR-MNT-001`, `NFR-TST-003` | Traceability and verification evidence |
| `QG-01`, `QG-08`, `QG-10` | Source integrity, operations, and release evidence |

## Verification results

| Command | Result | Evidence |
|---|---|---|
| `make telemetry-context-check` | Blocked | Repository shell has no `go` executable. |
| `go.exe test ./internal/platform/telemetry ./internal/platform/httpx ./internal/platform/integration ./internal/platform/worker` | Passed | All focused packages passed. |
| `go.exe vet ./internal/platform/telemetry ./internal/platform/httpx ./internal/platform/integration ./internal/platform/worker` | Passed | No diagnostics. |
| `go.exe test ./...` | Passed | All repository Go packages passed. |
| `go.exe test -race ./internal/platform/telemetry ./internal/platform/httpx ./internal/platform/integration ./internal/platform/worker` | Blocked | Toolchain reports `-race requires cgo`; `CGO_ENABLED=0`. |
| `git diff --check` | Passed | No whitespace errors. |
| `bash -n scripts/verify/telemetry-context.sh` | Passed | Verification script parses successfully. |

The focused gate is committed and covers package tests, race tests, `go vet`,
sensitive-field negative scanning, and whitespace validation. Race and the
repository-native Make gate remain open until a Linux Go toolchain or a CGO-
enabled Go toolchain is available.

## Current verification status

Implementation is present on branch
`feat/dlv-ops-001-us1-telemetry-context`. The repository-native `go` command is
not available, but the installed Windows Go toolchain provided the passing
non-race test and vet evidence above:

```text
/bin/bash: go: command not found
```

No production data, credentials, raw telemetry payloads, or secrets were used.
