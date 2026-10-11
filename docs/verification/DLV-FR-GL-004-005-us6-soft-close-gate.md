# DLV-FR-GL-004 / DLV-FR-GL-005 — User Story 6 verification evidence

Status: implementation added on branch `feat/gl-us6-soft-close-gate`.
The GL-owned soft-close entry and exit transitions, generic action handlers,
transactional PostgreSQL adapter, action-result-driven `GL-SCR-03` monitor,
and focused local tests are implemented. PostgreSQL integration execution,
browser/accessibility qualification, and production Fiscal Period Management
and soft-close policy adapter composition remain explicit qualification
boundaries.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 6 — Enter and exit a soft-close gate
- Delivery: DLV-FR-GL-004 / FR-GL-004; DLV-FR-GL-005 / FR-GL-005
- Existing operations: `glEnterSoftCloseGate`, `glExitSoftCloseGate`
- Permissions: `finance.gl.enter.soft.close.gate`,
  `finance.gl.exit.soft.close.gate`
- Primary screen: `GL-SCR-03`
- Branch: `feat/gl-us6-soft-close-gate`

## Implementation evidence

- `internal/gl/soft_close_gate.go` defines the GL-owned `PeriodPostingGate`,
  canonical opaque restrictions, active counters, immutable frozen summaries,
  entry/exit transitions, owner and epoch checks, expected-version checks,
  typed conflicts, policy-evaluator port, authorization port, and in-memory
  development adapter.
- `internal/gl/postgres_soft_close_gate_repository.go` locks the gate row,
  verifies the expected aggregate version, writes the gate state, audit record,
  exact v1 GL event, transactional outbox publication, and durable idempotency
  result in one PostgreSQL transaction.
- `db/migrations/gl/00006_add_soft_close_gate_state.sql` adds the gate identity,
  aggregate version, active owner/epoch, policy payload, active counters,
  frozen summary, retained summary history, constraints, and owner index.
  `internal/gl/gldb/models.go` and migration checksums are synchronized.
- `internal/platform/httpapi/gl_soft_close_gate_handler.go` maps the existing
  generic command envelope to both GL operations and returns established
  results containing mode, scope, owner, epoch, versions, opaque restrictions,
  counters, frozen summaries, audit evidence, ordinary-posting admission,
  replay state, and recovery next action.
- `internal/gl/posting.go` and `internal/gl/posting_approval.go` invoke the
  GL-owned soft-close policy port for ordinary posting admission and approval
  application, and fail closed when no evaluator is composed.
- `cmd/api/identity_runtime.go` composes the PostgreSQL repository with durable
  idempotency and identity authorization. The production policy evaluator is
  intentionally unavailable until the approved GL/FPM policy adapter exists,
  so entry fails closed rather than persisting an unevaluated policy.
- `web/src/app/gl-posting-gate-workspace.tsx` implements `GL-SCR-03` as an
  action-result-driven monitor. It keeps retry identity stable, uses the latest
  established result for the next command version, and displays the gate mode,
  owner, epoch, version, restrictions, ordinary-posting admission, counters,
  frozen summary, audit reference, and recovery guidance. It does not add the
  `GetPostingGateStatus` read reserved for Story 10.
- `internal/platform/database/gl_soft_close_gate_integration_test.go` adds
  opt-in PostgreSQL coverage for entry, exit, replay, changed-fingerprint
  conflict, outbox publication, and audit-failure rollback.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Enter changes eligible `Open` to `SoftClosePolicy` and records restrictions, owner, epoch, policy version, expected version, and zero counters | `PeriodPostingGate.enter`, `TestSoftCloseGateEnterAndExitPreserveBoundaryEvidence`, HTTP result test | Implemented and locally tested |
| Matching owner and epoch alone can exit; exit freezes the epoch summary, restores `Open`, and clears ownership | `PeriodPostingGate.exit`, domain test, handler test | Implemented and locally tested |
| Typed conflicts cover scope/version/mode/owner/epoch/policy and changed fingerprints; identical retries replay | Service idempotency paths, domain/HTTP tests, durable coordinator integration path | Implemented and locally tested; PostgreSQL execution is opt-in and pending |
| Gate state, counters, frozen summaries, audit, idempotency, and outbox commit as one GL decision | Memory rollback tests, PostgreSQL transaction adapter, opt-in integration test | Implemented; PostgreSQL integration test added but not run under the default focused verification |
| Monitor shows state, ownership, restrictions, blocked ordinary actions, and recovery guidance | `GL-SCR-03` workspace, route test, workspace component tests | Implemented and locally tested; browser/accessibility qualification remains open |

## Verification executed

| Command | Result |
| --- | --- |
| `make db-migrate-inventory` | Pass; checksum inventory refreshed |
| `make db-migrate-validate` | Pass |
| `make db-migrate-check` | Pass |
| `make db-sqlc-check` | Pass; generated persistence model synchronized and Go suite passed inside the check |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/gl ./internal/platform/httpapi ./cmd/api` | Pass |
| `./node_modules/.bin/tsc --noEmit` from `web` | Pass |
| `./node_modules/.bin/vitest run src/routes/route-registry.test.ts src/routes/router.test.tsx --pool=threads --maxWorkers=1 --fileParallelism=false` from `web` | Pass; 17 tests |
| `./node_modules/.bin/vitest run src/app/gl-posting-gate-workspace.test.tsx --pool=threads --maxWorkers=1 --fileParallelism=false` from `web` | Pass; 2 tests |
| `git diff --check` | Pass |

## Qualification limits

- `GetPostingGateStatus` is intentionally not implemented in this story; the
  monitor is action-result driven until Story 10 delivers the authoritative
  status operation.
- The production identity runtime uses `UnavailableSoftClosePolicyEvaluator`
  and therefore rejects entry with a typed service-unavailable result until an
  approved policy evaluator/FPM handoff adapter is composed. The in-memory
  allow-all evaluator is a development/test adapter only.
- The production runtime has no approved FPM/soft-close policy adapter
  composition in this repository. The posting and approval services now invoke
  the GL evaluator port, but production posting admission remains unqualified
  until that adapter is supplied; the in-memory allow-all evaluator is for
  development and tests only.
- The opt-in integration test was not run because repository instructions make
  persistence-integration checks release/CI checks requiring explicit user
  request. Its code remains available for the PostgreSQL qualification run.
- Playwright, manual keyboard/focus/zoom/reflow checks, assistive-technology
  review, live Entra authorization, live FPM handoff, production audit-chain,
  performance, recovery, and release qualification were not run.

## Source references

- `docs/specs/finance_domain_model_ddd.md` — §§2.2, 3.3, 5.3, 14.3,
  especially the `PeriodPostingGate` soft-close contract and journal-admission
  boundary.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` —
  FR-GL-004 and FR-GL-005.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — `GL-SCR-03` and
  period-control workflow behavior.
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md`,
  `04_events_workers_integration_specifications_v1.0.md`, `05_frontend_ui_technical_specifications_v1.0.md`,
  `09_testing_performance_recovery_specifications_v1.0.md`, and
  `10_technical_traceability_decisions_v1.0.md`.
- `contracts/openapi/paths/general-ledger.yaml` — existing generic action
  routes, permissions, headers, and established/problem result contracts.
