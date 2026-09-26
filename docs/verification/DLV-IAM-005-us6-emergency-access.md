# DLV-IAM-005 User Story 6 — Emergency access verification

## Scope and evidence boundary

This record covers the IAM-owned emergency-access grant lifecycle for
`EP-IAM-001` User Story 6. It proves the domain lifecycle, external approval
and authentication-assurance boundaries, immutable identity persistence,
typed grant/revoke operations, authorization denial after expiry or
revocation, review deadlines, audit metadata, and the IAM-SCR-04 synthetic
workflow.

Workflow Approval remains an external dependency. The implementation consumes
the existing versioned approval-decision reference shape through an IAM port;
it does not add a Workflow Approval implementation, a public review operation,
or an unapproved integration event.

## Source state

| Field | Value |
|---|---|
| Branch | `codex/iam-us6-emergency-access` |
| Baseline | Existing repository branch before Story 6 changes |
| Implementation state | Working-tree changes; no staging, commit, push, reset, clean, or revert performed |
| Verification date | 2026-09-26 |
| Verification profile | IAM domain, HTTP, authentication, API generation, SQLC generation, and frontend source checks |
| Remote dependencies | Go tests requiring a missing OpenTelemetry module and frontend execution were limited by the WSL runtime environment |

## Implementation evidence

- `internal/identity/emergency_access.go` defines the
  `EmergencyAccessGrant` aggregate, lifecycle and review transitions,
  four-hour policy cap, five-minute assurance rule, approval port, business
  calendar port, authorization evaluator, audit metadata, idempotency, and
  memory runtime doubles.
- `internal/identity/postgres_emergency_access_repository.go`,
  `db/migrations/identity/00005_create_emergency_access_schema.sql`, and
  `db/queries/identity/emergency_access_grant.sql` implement IAM-owned root,
  immutable revisions, normalized permissions/scopes, current-version
  selection, and optimistic concurrency.
- `internal/platform/httpapi/identity_handler.go` and the generated OpenAPI
  artifacts implement the existing `iamGrantEmergencyAccess` and
  `iamRevokeEmergencyAccess` operations with typed request data and typed
  authorization, approval, assurance, conflict, duplicate, unavailable, and
  validation failures.
- `internal/identity/actor.go` and
  `internal/platform/authentication/validator.go` preserve authentication
  assurance age, assurance level, methods, and explicit step-up reference.
- `cmd/api/identity_runtime.go` wires memory and PostgreSQL runtimes, the
  environment-gated approval double, IAM authorization, audit linkage, and
  the weekday business-calendar implementation.
- `web/src/app/identity-access-workspace.tsx` and the route registry implement
  IAM-SCR-04 at `/identity-access/iam-scr-04` with masked synthetic grant,
  revoke, expiry, review, denial, duplicate, conflict, and step-up states.

## Traceability

| Identifier | Evidence relationship |
|---|---|
| `DLV-FR-IAM-005` | Grant command, approval, assurance, expiry, authorization, persistence, audit, and UI workflow |
| `DLV-FR-IAM-006` | Revoke command, default-deny propagation, review update, persistence, audit, and UI workflow |
| `NFR-SEC-003` | High-risk assurance age or explicit step-up is required by service authorization |
| `NFR-SEC-006` | Independent approval reference and policy version are validated before grant |
| `NFR-SEC-013` | Expired and revoked grants are denied even when cleanup is delayed |
| `NFR-AUD-001` | Grant, revoke, and review transitions carry audit metadata and grant reference |
| `NFR-AUD-003` | Audit references are linked through the IAM audit port without raw sensitive payloads |
| `IAM-SCR-04` | Synthetic grant/revoke/review workflow and negative state fixtures |

## Verification commands and results

| Command or check | Result | Evidence or limitation |
|---|---|---|
| `go test ./internal/identity` | Passed | Domain lifecycle, expiry, revocation, review, approval, assurance, evaluator, and idempotency tests passed. |
| `go test ./internal/platform/httpapi` | Passed | Typed grant/revoke API, lifecycle, step-up, denial, and problem mapping tests passed. |
| `go test ./internal/platform/authentication -run 'TestValidator|TestEnvironment|TestFixtureMode|TestMiddleware'` | Passed | Token assurance extraction, fixture assurance, middleware, and fail-closed authentication tests passed. |
| `go test ./cmd/api` | Passed | Memory/PostgreSQL runtime wiring and API package tests passed from the available cache. |
| `go test ./...` | Blocked by environment dependency access | The suite reached unchanged telemetry tests but could not download `go.opentelemetry.io/otel/sdk/metric@v1.44.0` because the sandbox denied DNS/network access. |
| `go test ./internal/platform/authentication` | Blocked by environment socket access | The unchanged JWKS test could not bind the `httptest` IPv6 listener in this restricted WSL session (`listen tcp6 [::1]:0: socket: operation not permitted`). |
| `make db-sqlc-generate` | Passed | SQLC generated the emergency-access queries and identity models. |
| OpenAPI bundle and Ogen generation | Passed | `contracts/openapi/dist/openapi.bundle.yaml` and generated Go HTTP artifacts were regenerated from the updated contract. |
| TypeScript OpenAPI generation | Passed | `web/src/generated/api` was regenerated from the bundled contract using the repository’s configured generator. |
| Frontend test/build execution | Blocked by environment runtime | The repository UI tests and route assertions are present, but the WSL Node bridge failed before test startup with `UtilBindVsockAnyPort`. |
| `git diff --check` | Passed | No whitespace errors. |

## Safety and limitations

- Approval and business-calendar production integrations remain explicit ports;
  local runtimes use only explicit synthetic doubles.
- No PostgreSQL-backed migration execution is claimed here because Docker or a
  live PostgreSQL service was not available in this environment.
- The UI is intentionally synthetic and does not expose actor secrets,
  token claims, approval payloads, or sensitive action data.
- Revocation propagation is represented by synchronous evaluator denial and
  audit linkage; no new event name was invented for a future outbox contract.
