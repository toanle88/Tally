# DLV-IAM-004 User Story 5 — Segregation-of-duties verification

## Scope and evidence boundary

This record covers the IAM-owned portion of `EP-IAM-001` User Story 5. It
proves immutable versioned segregation-rule administration, rule-backed role
validation, the published IAM decision boundary, safe API outcomes, and the
IAM administration/explanation surfaces using synthetic data.

Payment, fiscal-period, vendor, payroll, and GL action handlers are not present
in this checkout. Their final action-time integrations remain deferred to the
bounded contexts that own those actions; this story supplies the IAM decision
port and synthetic contract coverage for them.

## Source state

| Field | Value |
|---|---|
| Branch | `codex/iam-us5-segregation-of-duties` |
| Baseline | `main` at `21661cc56bf2cea1e2134fe6b8834a6c1eb2ea68` |
| Implementation state | Working-tree changes; no staging, commit, push, reset, or cleanup performed |
| Verification date | 2026-09-26 |
| Verification profile | Local synthetic unit, API, generated-contract, frontend, and integration-package compile checks |
| Remote dependencies | None; Docker was unavailable, so PostgreSQL execution was not run |

## Implementation evidence

- `internal/identity/segregation.go` defines immutable rule revisions, the six
  minimum v1 rules, scope/history/threshold/cooling-off/exception evaluation,
  fail-closed stale and unavailable outcomes, the published
  `SegregationDecisionPort`, and an outcome-only telemetry observer.
- `internal/identity/segregation_durable.go` connects rule mutations to the
  platform idempotency coordinator for durable replay and conservative failure
  recovery.
- `db/migrations/identity/00004_create_segregation_rule_schema.sql` and
  `db/queries/identity/segregation_rule.sql` define the identity-owned schema,
  seeded baseline rules, immutable revisions, current-version selection, and
  optimistic concurrency queries.
- `internal/identity/postgres_segregation_rule_repository.go` performs rule
  persistence and audit linkage through the audit port without writing the
  audit schema directly.
- `internal/platform/httpapi/segregation_rule_handler.go` and the typed OpenAPI
  contract implement `manage-segregation-rules`, required idempotency, typed
  expected-version/If-Match handling, independent approval, and safe denial
  mappings.
- `web/src/app/identity-access-workspace.tsx` adds IAM-SCR-03 rule
  administration and IAM-SCR-05 non-sensitive decision explanation states;
  finance-domain screens were not added.

## Traceability

| Identifier | Relationship |
|---|---|
| `M1` | Milestone for this delivery |
| `EP-IAM-001` / `DLV-FR-IAM-004` | Parent epic and delivery requirement |
| `DLV-GFR-003` / `FR-IAM-004` | Segregation and IAM functional requirements |
| DDD §8.2 | IAM ownership and segregation domain boundary |
| `NFR-SEC-005` / `NFR-SEC-012` | Authorization, separation of duties, and fail-closed integrity controls |
| `IAM-WS-01` / `IAM-SCR-03` / `IAM-SCR-05` | Administration and explanation workflow surfaces |
| `QG-01`–`QG-06`, `QG-08` | Source, test, contract, persistence, security, and verification gates |

## Verification commands and results

| Command | Result | Evidence |
|---|---|---|
| `go test ./internal/identity` | Passed | Domain rules, fail-closed decisions, exceptions, thresholds, role validation, and service replay tests passed. |
| `go test ./cmd/api ./internal/identity` | Passed | Runtime wiring and IAM domain packages passed. |
| `go test ./internal/platform/httpapi` | Passed | Typed rule API, headers, conflict, stale, denial, approval, and idempotency tests passed. |
| `go test ./internal/platform/database -tags integration -run '^$'` | Passed | PostgreSQL integration test package compiled without running Docker-backed tests. |
| `go test ./...` | Failed in unchanged environment wrappers | All changed packages passed, but `internal/platform/aggregateversion` and `internal/platform/identity` compile-check tests reported `compile failed for an unexpected reason` with empty nested compiler output. Their `compilecheck_test.go` files are unchanged; Docker/WSL Windows process resolution prevented reliable nested `go` diagnostics. |
| `tsc --noEmit -p web/tsconfig.json` | Passed | Frontend type-check completed with the installed Node/TypeScript toolchain. |
| `vitest run` | Passed | 14 test files and 64 tests passed, including the new segregation administration/explanation test. |
| `vite build` | Passed | Production build completed; Vite emitted only the existing chunk-size warning. |
| `redocly lint contracts/openapi/openapi.yaml` | Passed | OpenAPI description validated; the pinned CLI also reported that a newer CLI is available and that `npm` is unavailable, neither changed the exit status. |
| `go run github.com/sqlc-dev/sqlc/cmd/sqlc@v1.31.1 generate -f sqlc.yaml` | Passed | SQLC generated the identity rule queries and models successfully. |
| `bash scripts/db/migrate.sh check` | Passed | The migration checksum inventory, including identity migration `00004`, is valid. |
| `bash scripts/db/migrate.sh validate` | Blocked in the WSL shell | The filesystem validator reached Goose but the shell has no `go` executable; the tagged database package still compiled with the installed Windows Go toolchain. |
| `git diff --check` | Passed | No whitespace errors. |
| Docker-backed PostgreSQL integration test | Not run | Docker socket access was unavailable (`permission denied`); the integration source covers migration ownership, immutable revisions, current selection, optimistic concurrency, audit linkage, and durable replay. |

## Safety and limitations

- Rule revisions and established audit references are immutable; updates create
  a new version and use expected-version concurrency checks.
- A stale rule/policy version, unavailable rule store, missing required history,
  expired exception, or unsafe approval state fails closed without a protected
  action being established.
- API and UI explanations expose safe reason codes, non-sensitive reasons,
  permitted resolution guidance, and decision/version references only.
- The implementation does not claim production database execution, live
  finance-action wiring, production authorization policy configuration, or
  production telemetry qualification.
