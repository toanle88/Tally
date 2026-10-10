# DLV-FR-GL-015/016 — User Story 1 ledger and accounting-book maintenance

Status: implementation and local repository verification are present on
`feat/gl-us1-ledger-book-maintenance`. The story is not presented as fully
qualified: live cross-context OMD integration evidence, complete segregation-of-duties
approval integration, journal-history integration, and browser qualification
remain follow-up evidence.

## Scope delivered

The change delivers the GL-owned ledger and accounting-book configuration
boundary across the domain, application, PostgreSQL adapter, OpenAPI contract,
generated clients, HTTP runtime, and `GL-SCR-04`:

- `internal/gl/configuration.go` defines effective-dated `Ledger` and
  `AccountingBook` aggregates, revisions, lifecycle transitions, exact
  currency validation, optimistic versions, and safe projections.
- `internal/gl/configuration_service.go` defines command services, explicit
  authorization/reference/approval/audit ports, local and durable idempotency,
  duplicate effective-date checks, and atomic audit-before-commit behavior.
- `cmd/api/gl_reference_validator.go` wires the PostgreSQL runtime to the OMD
  application reference boundary for scoped legal-entity and fiscal-calendar
  existence checks. The memory profile keeps its fixture adapter; no GL code
  reads OMD repositories or tables directly.
- PostgreSQL GL wiring uses `gl.UnavailableApprovalValidator` until an approved
  Workflow decision adapter exists, so unverifiable approval references fail
  closed with a service-unavailable result rather than being accepted.
- `internal/gl/postgres_configuration_repository.go` owns GL persistence,
  transaction boundaries, scope advisory locking, optimistic updates,
  revision retention, audit linkage, overlap checks, and the ledger-to-book
  relationship. It does not read OMD tables directly.
- `db/migrations/bootstrap/00005_create_gl_schema.sql` and
  `db/migrations/gl/00001_create_ledger_and_accounting_book_schema.sql` add
  the GL-owned schema and revision tables. `sqlc.yaml`, GL queries, migration
  orchestration, and checksums include the new set.
- `contracts/openapi/components/common.yaml` and
  `contracts/openapi/paths/general-ledger.yaml` define typed command,
  established-result, projection, and problem contracts for
  `glMaintainLedgers` and `glMaintainAccountingBooks`.
- `internal/platform/httpapi/gl_handler.go` maps typed requests, scope and
  `If-Match` headers, actor context, correlation/causation, idempotency, safe
  established results, and typed errors. Runtime wiring exists for memory and
  PostgreSQL modes.
- `web/src/app/gl-ledger-book-workspace.tsx` adds the scoped `GL-SCR-04`
  configuration surface, local safe-read adapter, lifecycle/date/duplicate
  checks, typed mutation calls, safe-result refresh, and visible control status
  including owner, evidence, blocked action/reason, and next action.

## Contract decisions requiring follow-up

Two source-contract tensions were preserved as explicit decisions rather than
inventing new identifiers or routes:

1. `docs/specs/system_design/03_data_integration_architecture_v1.0.md`
   describes a general-ledger boundary as `/api/v1/gl`, while the approved
   PRD/technical/OpenAPI/backlog operation entries use
   `/api/v1/general-ledger/...`. This implementation retains the existing
   operation paths and IDs from the OpenAPI/backlog contract.
2. The DDD defines a full posting `AccountingScope` containing ledger and book
   identifiers, while a configuration command must create those identifiers.
   For Story 1, `accountingScopeId` is therefore treated as a required opaque
   configuration/authorization scope ID; legal-entity ownership and the
   ledger-to-book relationship remain explicit fields. Full posting-scope
   construction is deferred until the posting contract is implemented.

## Traceability

| Story requirement | Implementation/evidence | Current result |
| --- | --- | --- |
| Maintain effective-dated ledgers | `LedgerCommand`, `Ledger`, revisions, lifecycle/date validation, typed PUT handler | Implemented locally |
| Maintain books for an existing ledger | `AccountingBookCommand`, in-memory parent check, PostgreSQL FK, typed PUT handler | Implemented locally |
| Reject invalid references and intervals atomically | OMD application reference adapter, nil/reference shape checks, date/currency/lifecycle rules, overlap checks, transactional rollback | Runtime path implemented; live cross-context integration evidence remains open |
| Return authoritative identity/version/relationship/evidence | Generated typed established results and safe projections | Implemented locally |
| Scope, authorization, idempotency, correlation, audit, optimistic concurrency | Service ports, evaluator authorizer, durable coordinator, audit transaction, `If-Match`/expected-version mapping | Local paths covered; full SoD approval integration remains open; approval now fails closed when Workflow is absent |
| Preserve configuration history | Append-only ledger/book revision rows and in-memory/PostgreSQL reloading | Implemented locally; no journal consumer exists yet |
| GL-SCR-04 state and recovery guidance | Scoped tables, editors, control-status panel, typed result refresh | Component-covered; Playwright not run |

## Verification executed

The following checks were executed in the current working tree. No commit was
created for this evidence record.

| Command | Result |
| --- | --- |
| `pnpm contract:lint` | Pass |
| `pnpm contract:check` | Pass; deterministic bundle SHA-256 `cc13a4ae6756283d459411c16f222a686df58a4bb81d25f23b61c4a12b3cc73a` |
| `GOCACHE=/tmp/tally-go-cache make api-generate-check` | Pass; 15 generated Go artifacts verified |
| `GOCACHE=/tmp/tally-go-cache make db-sqlc-check` | Pass |
| `make db-migrate-validate` | Pass for bootstrap, platform, identity, organization, COA, and GL sets |
| `make db-migrate-check` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./...` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test ./cmd/api ./internal/organization ./internal/gl ./internal/platform/httpapi` | Pass; OMD reference adapter, fail-closed approval, typed replay, and dependency mapping coverage |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^TestGLConfigurationRepositoryPreservesRevisionsAndRollsBackFailures$' -count=1` | Pass; PostgreSQL 18 Testcontainers repository, revision, same-scope FK, and rollback coverage |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^TestPersistenceWorkflow$' -count=1` | Pass; clean migration apply/history, generated-query commit/rollback, and reapplication |
| `pnpm -C web exec vitest run --pool=forks --maxWorkers=1 --fileParallelism=false` | Pass; 16 files and 87 tests |
| `pnpm -C web exec vitest run src/lib/scope/scope-context.test.tsx --pool=forks --maxWorkers=1 --fileParallelism=false` | Pass; 1 file and 3 tests |
| `pnpm -C web run test -- --runInBand` | Default worker configuration timed out after 15 files and 84 tests; superseded by the native single-worker pass above |
| `pnpm -C web run build` | Pass; Vite production build completed |
| `git diff --check` | Pass; only the repository's existing Makefile CRLF normalization warning |

The full frontend check used Vitest's native single-fork, single-worker
configuration because the default worker-start configuration previously timed
out in this environment. It completed with 16 files and 87 passing tests,
including the changed workspace and route coverage. The focused scope-context
check remains recorded separately above.

## Open qualification items

- Live PostgreSQL cross-context tests have not yet exercised the OMD reference
  adapter against created LegalEntity and FiscalCalendar aggregates. The
  runtime adapter is present and covered with application-port doubles plus an
  in-memory OMD application-service composition test; the live test remains a
  qualification item.
- A complete Workflow/SOD decision path is not part of this delivery.
  PostgreSQL approval-bearing GL commands fail closed through
  `gl.UnavailableApprovalValidator` until that adapter is supplied. The
  memory profile and domain unit tests retain permissive doubles for local
  fixture coverage only.
- No journal aggregate exists in this scope, so preservation of a version
  referenced by an established journal is represented by configuration
  revision retention but cannot yet be demonstrated end to end.
- No approved GL read endpoint exists yet. `GL-SCR-04` uses a clearly labelled
  local safe adapter and refreshes it only from the typed mutation result.
- Build-tagged PostgreSQL integration tests require a PostgreSQL 18
  Testcontainers runtime. The GL integration coverage is in
  `internal/platform/database/gl_configuration_integration_test.go`; the
  targeted repository and shared persistence workflow both passed in the
  recorded environment.
- Playwright, manual keyboard/focus/zoom, and assistive-technology review were
  not run for this story.

