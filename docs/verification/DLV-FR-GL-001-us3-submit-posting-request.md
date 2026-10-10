# DLV-FR-GL-001 — User Story 3 verification evidence

Status: implementation added on branch `feat/gl-us3-submit-posting-request`.
The typed domain, HTTP, frontend, migration, and PostgreSQL repository slices
are implemented and covered by local tests. Production PostgreSQL composition
still fails closed until the approved Fiscal/Period Management, Workflow/SOD,
and other reference adapters are available; Playwright qualification was not
run because this repository does not yet contain the referenced
`e2e/wf-6-6.spec.ts`.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 3 — Submit and validate a posting request
- Delivery: DLV-FR-GL-001 / FR-GL-001
- Existing operation: `glSubmitPostingRequest`
- Primary screens: `GL-WS-01`, `GL-SCR-01`, `GL-SCR-02`
- Branch: `feat/gl-us3-submit-posting-request`

## Implementation evidence

- `internal/gl/posting.go` defines the version-2 posting command, exact
  decimal validation, ownership/source checks, gate and period checks,
  authorization, approval outcomes, typed rejection issues, deterministic
  idempotency fingerprints, and fail-closed rejection auditing.
- `internal/gl/posting_memory.go` provides a test/runtime composition that
  preserves replay, changed-fingerprint conflict, source uniqueness, distinct
  rejected/idempotency-conflict attempts, pending-approval-without-ledger-
  effect, audit, and outbox intent behavior.
- `internal/gl/postgres_posting_repository.go` performs gate locking,
  reference checks, idempotency/source checks, journal/line/attempt/audit/
  outbox writes, and gate-position advancement in one PostgreSQL transaction.
  Pending approval does not advance the ledger position. The migration also
  blocks direct updates/deletes of posted journals and posted journal lines.
- `db/migrations/gl/00003_create_posting_schema.sql` adds the GL-owned gate,
  journal, journal-line, and posting-attempt schema, exact numeric amounts,
  scope/source/idempotency uniqueness, composite reference keys, and posted
  journal immutability checks. `internal/gl/gldb/models.go` is regenerated
  from the approved SQL queries.
- `contracts/openapi/paths/general-ledger.yaml` and
  `contracts/openapi/components/common.yaml` define the typed v2 command and
  established result. Go and TypeScript clients were regenerated.
- `internal/platform/httpapi/gl_posting_handler.go` maps scope, actor,
  permission, common headers, typed results, validation problems, conflicts,
  and unavailable dependencies without exposing another module's adapter.
- `web/src/app/gl-posting-workspace.tsx` implements the three scoped posting
  views with exact-decimal client validation, line add/remove controls,
  transaction/functional amounts, conversion and purpose evidence, safe
  outcome/recovery messaging, and result navigation.
- `web/src/routes/route-registry.ts` and `web/src/routes/router.tsx` register
  `GL-WS-01`, `GL-SCR-01`, and `GL-SCR-02` at the approved routes.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Version-2 command contract | Typed OpenAPI schemas, generated clients, handler mapping, and v2 request test | Implemented locally |
| Ledger/book/scope/period/gate/purpose/currency validation | Domain validator, gate repository checks, typed dependency ports, and rejection tests | Implemented locally; live FPM/reference adapters remain a qualification boundary |
| Exact decimal, line-count, balance, scale, currency, and conversion rules | `money.Money` plus decimal totals in `internal/gl/posting.go`; domain and component tests | Implemented and tested |
| Posted journal result | Memory service, HTTP handler, and PostgreSQL integration test verify identity, number, position, gate, source, audit, and outbox evidence | Implemented and tested |
| Pending approval has no ledger effect | Domain/memory service test asserts `PendingApproval`, `PJE-*`, and zero ledger position | Implemented locally; production Workflow/SOD adapter is fail-closed until supplied |
| Rejected, posted, pending, and idempotency-conflict outcomes | Domain, handler, and memory repository tests; typed result/problem mapping | Implemented locally |
| Replay and changed-content conflict | Replay is resolved before current gate validation; memory and PostgreSQL tests verify one effect, replay, retained source duplicate/idempotency-conflict attempts, and fingerprint conflict | Implemented and tested |
| Workbench/request/result recovery visibility | Component and router tests plus scoped workspace panels | Implemented locally; browser and assistive-technology qualification remains open |

## Verification executed

| Command | Result |
| --- | --- |
| `pnpm contract:lint` | Pass |
| `pnpm contract:check` | Pass; deterministic bundle and four invalid fixtures |
| `GOCACHE=/tmp/tally-go-cache make api-generate-check` | Pass |
| `make api-ts-check` | Pass |
| `make db-migrate-validate` | Pass |
| `make db-migrate-inventory` | Pass; checksum inventory refreshed |
| `make db-migrate-check` | Pass |
| `GOCACHE=/tmp/tally-go-cache make db-sqlc-check` | Pass with approved dependency-download escalation |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/gl ./internal/platform/httpapi ./cmd/api` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^$'` | Pass; integration package compiles |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^TestGLPostingRepositoryPostsAtomicallyAndSerializesGateAdmission$' -count=1 -v` | Pass with PostgreSQL 18 Testcontainers and approved Docker escalation |
| `pnpm -C web exec vitest run src/app/gl-posting-workspace.test.tsx` | Pass; 3 tests, including stable identity and conversion-evidence retry |
| `pnpm -C web exec vitest run src/routes/router.test.tsx -t 'GL posting'` | Pass; 2 tests |
| `pnpm -C web exec vitest run --pool=forks --maxWorkers=2 --fileParallelism=false --reporter=verbose` | Pass; 17 files and 92 tests |
| `pnpm -C web run build` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./...` | Pass |
| `git diff --check` | Pass |

## Qualification limits

- `cmd/api` keeps the PostgreSQL posting service unset until approved FPM,
  Workflow/SOD, audit, and cross-context reference adapters are composed; the
  HTTP handler therefore returns typed `503` dependency-unavailable results
  instead of silently using permissive doubles in production.
- GL does not read the COA, identity, organization, or Workflow schemas
  directly. Account/segment eligibility beyond the GL-owned references remains
  dependent on the published adapter boundary.
- PostgreSQL uniqueness and gate locking provide durable idempotency/source
  protection for this slice, including retained idempotency-conflict attempt
  outcomes; a separate platform idempotency coordinator is not yet composed
  in the runtime.
- No approved GL read endpoint exists. The frontend keeps safe scoped fixture
  context and the accepted mutation result; it does not invent a journal-read
  authority.
- Playwright, manual keyboard/focus/zoom/reflow, assistive-technology review,
  and live production adapter qualification were not run.
