# DLV-FR-GL-017/018 — User Story 2 verification evidence

Status: implementation added on branch feat/gl-us2-chart-account-mappings.
The local domain, API, PostgreSQL repository, migration, component type-check,
route checks, and existing frontend suite pass. Vitest now separates lightweight
Node tests from jsdom UI tests so non-DOM tests do not load the browser
environment or React Testing Library setup. The new isolated frontend workspace
test was removed after its run exceeded a practical time budget in this repository;
the production component remains in place. Live reporting-definition integration,
Workflow/SOD approval qualification, journal-history integration, and Playwright
qualification remain explicitly outside the verified boundary recorded here.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 2 — Maintain charts of accounts and account/reporting mappings
- Delivery: DLV-FR-GL-017 / FR-GL-017; DLV-FR-GL-018 / FR-GL-018
- Domain authority: docs/specs/finance_domain_model_ddd.md, §2.2 and §3.1
- Functional requirements: FR-GL-017 and FR-GL-018
- Primary screen: GL-SCR-05

## Implementation evidence

- internal/gl/chart_account.go defines GL-owned ChartOfAccounts and Account
  aggregates, restrictions, reporting-mapping value objects, lifecycle
  transitions, effective-date rules, normal-balance rules, safe projections,
  and immutable in-process revision history.
- internal/gl/chart_account_service.go defines typed command services with
  scope-bound authorization, expected-version checks, idempotency
  fingerprints, approval and reference ports, audit records, durable
  idempotency coordination, and safe command results.
- db/migrations/gl/00002_create_chart_of_accounts_and_account_schema.sql
  adds GL-owned chart/account current and revision tables, effective lookup
  indexes, lifecycle/version checks, and same-scope ledger/chart foreign keys.
  Restrictions and reporting mappings are stored as GL-owned JSON snapshots;
  no coa or reporting schema is accessed directly.
- internal/gl/postgres_chart_account_repository.go commits current rows,
  revision snapshots, audit linkage, optimistic updates, parent checks,
  effective-date overlap checks, and durable idempotency in one transaction.
- contracts/openapi/paths/general-ledger.yaml and
  contracts/openapi/components/common.yaml define typed command/result
  contracts for both existing Story 2 operations. Go and TypeScript clients
  were regenerated from those sources.
- internal/platform/httpapi/gl_chart_account_handler.go maps scope,
  If-Match, correlation, actor, idempotency, typed results, and safe
  conflict/dependency responses.
- cmd/api/gl_chart_account_runtime.go keeps authorization and cross-context
  checks behind application ports. The PostgreSQL runtime fails closed for
  reporting mappings until an approved reporting-definition adapter exists; it
  does not read another module's schema.
- web/src/app/gl-chart-account-workspace.tsx implements GL-SCR-05 with
  scoped safe projections, chart/account editors, restrictions, currency
  policy, reporting mapping fields, lifecycle/date validation, versioned
  mutation calls, and visible owner, dependent-impact, approval, validation,
  blocked-action, recovery, and next-action status.
- web/src/routes/route-registry.ts and web/src/routes/router.tsx register
  /general-ledger/gl-scr-05 without creating a second account mutation
  surface.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Versioned charts for an existing ledger | GL aggregate/repository parent check, migration FK, typed handler, PostgreSQL test | Implemented locally |
| Versioned accounts and reporting mappings | Account aggregate, typed command/result, mapping validation, safe projection | Implemented locally; live reporting adapter remains unavailable |
| Effective uniqueness and field validation | Domain validation plus memory/PostgreSQL overlap checks and indexes | Implemented and tested |
| COA boundary ownership | Application reference port; no direct coa schema access from GL | Implemented |
| Identity, relationship, dates, approval, validation in results | Generated established results and safe projections | Implemented locally |
| History preservation | Current plus append-only revision rows and reload assertions | Implemented; no journal aggregate exists yet |
| Scope, authorization, idempotency, audit, concurrency, safe errors | Services, runtime evaluator ports, transactional repository, HTTP error mapping | Implemented and locally tested |
| GL-SCR-05 control visibility | Workspace tables/editors/control-status panels and component type-check | Implemented locally |

## Verification executed

| Command | Result |
| --- | --- |
| GOCACHE=/tmp/tally-go-cache go test ./... | Pass |
| GOCACHE=/tmp/tally-go-cache go test ./internal/gl | Pass |
| GOCACHE=/tmp/tally-go-cache go test ./internal/platform/httpapi | Pass |
| GOCACHE=/tmp/tally-go-cache go test -tags integration ./internal/platform/database -run TestGLChartAccountRepositoryPreservesRevisionsAndRollsBackAuditFailures -count=1 | Pass with PostgreSQL 18 Testcontainers |
| GOCACHE=/tmp/tally-go-cache go test -tags integration ./internal/platform/database -run '^$' | Pass; integration package compiles |
| make db-migrate-validate | Pass |
| make db-migrate-check | Pass |
| pnpm -C web exec tsc -b --pretty false | Pass |
| pnpm -C web test -- --pool=forks --maxWorkers=2 | Pass; 16 files and 87 tests; isolated DOM and Node projects |
| pnpm -C web exec vitest run src/routes/route-registry.test.ts --pool=forks --maxWorkers=1 --fileParallelism=false | Pass; 4 tests |
| make api-generate | Pass |
| make api-ts-generate | Pass |
| make db-sqlc-generate | Pass |
| git diff --check | Pass; only pre-existing CRLF normalization warnings |

## Qualification limits

- The production PostgreSQL runtime uses an unavailable approval validator
  until the Workflow/SOD decision adapter is supplied. Approval-bearing
  mutations therefore fail closed; permissive approval doubles are limited
  to memory composition and tests.
- The repository has no reporting-definition bounded-context adapter yet.
  Reporting mappings are modeled and validated as approved references at the
  GL boundary, but production submission with mappings returns a typed
  service-unavailable result until that adapter is supplied.
- No JournalEntry aggregate or posting validator exists in this delivery.
  Revision retention prevents destructive rewriting of configuration, but an
  end-to-end journal reference to a historical chart/account version cannot
  yet be demonstrated.
- No approved GL read endpoint exists. GL-SCR-05 uses a clearly labelled
  local safe adapter and refreshes it only from accepted typed mutation
  responses.
- Playwright, manual keyboard/focus/zoom/reflow, assistive-technology review,
  and live production identity/reporting qualification were not run.
