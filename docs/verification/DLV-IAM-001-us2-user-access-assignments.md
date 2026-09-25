# DLV-IAM-001 — User Story 2 verification evidence

Status: implementation added on branch codex/iam-us2-user-access-assignments; local Go, PostgreSQL, API, and frontend checks pass through the available host toolchains, with wrapper and regeneration limitations recorded below.

## Traceability

- Epic: EP-IAM-001
- Story: User Story 2 — Manage users and access assignments
- Delivery: DLV-FR-IAM-001, DLV-GFR-015
- Security/privacy: NFR-SEC-006, NFR-SEC-012, NFR-PRV-001–NFR-PRV-003
- Quality gates: QG-01, QG-02, QG-03, QG-04, QG-05, QG-06, QG-08, QG-10

## Implemented surfaces

- internal/identity contains explicit inactive/active/suspended/terminated lifecycle transitions, immutable authentication subjects, normalized role assignments, duplicate-assignment rejection, and atomic complete-set replacement semantics.
- The identity application service defines repository, authorization-decision, and audit ports; enforces permission finance.iam.manage.users, scope containment, idempotency fingerprints, expected versions, safe error categories, and audit fingerprints.
- Identity-owned migration/query sources define identity.user_account, assignment joins, subject uniqueness, status/version checks, and optimistic updates; generated SQLC output and a transaction-scoped PostgreSQL repository adapter are included. Integration fixtures include the identity migration set.
- The approved manage-users route has a typed action envelope for create, update, activate, suspend, and terminate, plus an If-Match contract parameter and masked command example.
- The generated transport boundary is composed under the existing authentication middleware; ogen v1.23.0 regenerated the typed Go request, handlers, and validators. The runtime selector shares the authoritative PostgreSQL repository with authentication revalidation when DATABASE_URL is configured and fails closed instead of silently selecting memory when the audit-owner or role-policy port is missing.
- The frontend includes IAM-WS-01 and IAM-SCR-01 worklist/detail behavior with masked subject data, lifecycle state, assignment replacement, review evidence, validation, safe duplicate/denial states, and version-conflict recovery.

## Verification log

| Command | Result |
|---|---|
| `go.exe test ./...` | Passed; all Go packages, including identity and generated identity SQLC package. |
| `go.exe test -race ./internal/identity` | Blocked by host configuration: `-race requires cgo; enable cgo by setting CGO_ENABLED=1`. |
| `go.exe run sqlc@v1.31.1 compile -f sqlc.yaml` | Passed. |
| `go.exe run sqlc@v1.31.1 diff -f sqlc.yaml` | Passed; generated SQLC output is current. |
| Direct Redocly lint | Passed: `contracts/openapi/openapi.yaml` is valid. |
| Direct ogen v1.23.0 generation | Passed; regenerated checked-in Go artifacts from the typed IAM contract. |
| Direct web Vitest | Passed: 14 files and 59 tests. |
| Direct web TypeScript build | Passed: `tsc -b`. |
| Direct web production build | Passed: Vite build completed; emitted only the existing chunk-size warning. |
| Direct Playwright accessibility/keyboard suite | Passed: 18 tests. |
| `go.exe test ./internal/platform/httpapi` | Passed; generated-server and handler tests cover required idempotency headers, typed actions, masked subjects, denial, lifecycle activation, and version conflicts. |
| `go.exe test ./cmd/api` | Passed; API composition keeps health anonymous, protects the API, shares the persistent identity repository with authentication, and fails closed without the audit-owner or role-policy dependency. |
| `git diff --check` | Passed. |
| `bash -n scripts/db/migrate.sh scripts/db/verify.sh scripts/openapi/api-generate-check.sh scripts/openapi/redocly-run.sh` | Passed. |
| `bash scripts/db/migrate.sh check` | Passed: migration checksum inventory is valid. |
| `go.exe test -tags integration ./internal/platform/database` | Passed against PostgreSQL in Docker; the complete database integration suite completed successfully, including identity schema ownership, subject uniqueness, atomic assignment replacement, concurrent optimistic-lock coverage, transactional audit references, and durable idempotency replay. The same integration coverage also proves deterministic authorization failures are finalized as failed and replayed without re-executing the command. |
| `pnpm` wrapper commands | Blocked before task execution because pnpm attempted to reconcile/remove the existing modules directory without a TTY. |
| Direct TypeScript OpenAPI generation | Blocked by installed dependency mismatch: `open` requests `canAccessPowerShell` from `wsl-utils`, which is not exported. |
| `make api-check` | Blocked in the wrapper before generation because Windows Node mapped the WSL workspace path as `D:\mnt\d\...`; direct lint, ogen generation, Go tests, and SQLC checks passed separately. |

## Remaining evidence

The following evidence remains:

- Race checks require a CGO-enabled host configuration.
- TypeScript OpenAPI regeneration remains blocked by the installed open/wsl-utils mismatch.
- The full make api-check wrapper remains blocked by the WSL-to-Windows path mapping; its direct lint, ogen generation, Go, SQLC, and frontend checks are recorded above.
- Identity persistence invokes the approved transactional audit-writer port while the identity transaction is open, stores the returned audit reference in `identity.user_account`, and finalizes the shared PostgreSQL idempotency record in that same transaction. The full append-only audit-chain owner and role-policy production adapters remain outside this story and must be injected at composition time.

Role CRUD, policy CRUD, segregation rules, emergency access, full audit-chain persistence, production revocation-latency qualification, and live Entra qualification remain outside User Story 2.
