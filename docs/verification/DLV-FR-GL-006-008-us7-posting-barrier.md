# DLV-FR-GL-006 / DLV-FR-GL-007 / DLV-FR-GL-008 — User Story 7 verification evidence

Status: implementation added on branch `feat/gl-us7-posting-barrier`.
The GL-owned posting-barrier acquisition, release, and finalization transitions,
transactional PostgreSQL adapter, generic action handlers, action-result-driven
`GL-SCR-03` workflow, and focused local tests are implemented. PostgreSQL
runtime qualification, browser/accessibility qualification, and the public
`GetPostingGateStatus` operation remain explicit boundaries.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 7 — Acquire, release, and finalize the posting barrier
- Delivery: DLV-FR-GL-006 / FR-GL-006; DLV-FR-GL-007 / FR-GL-007;
  DLV-FR-GL-008 / FR-GL-008
- Existing operations: `glAcquirePostingBarrier`, `glReleasePostingBarrier`,
  `glFinalizePostingGate`
- Permissions: `finance.gl.acquire.posting.barrier`,
  `finance.gl.release.posting.barrier`, `finance.gl.finalize.posting.gate`
- Primary screen: `GL-SCR-03`
- Branch: `feat/gl-us7-posting-barrier`

## Implementation evidence

- `internal/gl/posting_barrier.go` adds typed acquire, release, and finalize
  commands, canonical fingerprints, the GL-owned application-port contract,
  internal authoritative status seam, exclusive-owner transitions, frozen
  admission summaries, barrier position, watermark, close type, and immutable
  finalization evidence.
- `internal/gl/soft_close_gate.go` extends the existing GL-owned gate aggregate,
  authorization mapping, durable idempotency operation identities, event
  identity mapping, memory adapter, and established result evidence without
  introducing a parallel aggregate.
- `internal/gl/postgres_soft_close_gate_repository.go` locks and persists all
  barrier fields, summaries, audit evidence, idempotency metadata, and
  transactional outbox publications in one PostgreSQL transaction.
- `internal/gl/postgres_posting_repository.go`,
  `internal/gl/postgres_posting_approval_repository.go`, and
  `internal/gl/postgres_journal_reversal_repository.go` record admitted close
  or reopen postings and advance the ledger position with the gate counters in
  the same transaction as the established journal outcome.
- `db/migrations/gl/00007_add_posting_barrier_state.sql` adds barrier ownership,
  prior-process metadata, close type, barrier position, final watermark,
  finalized close-run identity, constraints, and the active-owner index.
  SQLC output and migration checksums are synchronized.
- `internal/platform/httpapi/gl_posting_barrier_handler.go` maps the existing
  generic action envelopes to the three typed commands and returns established
  or typed problem results through the existing generated routes.
- `web/src/app/gl-posting-gate-workspace.tsx` extends `GL-SCR-03` with acquire,
  release, and finalize actions, stable retry identities, barrier/watermark and
  frozen-summary evidence, blocked ordinary-posting guidance, and no public
  status read.
- `internal/platform/database/gl_soft_close_gate_integration_test.go` contains
  opt-in PostgreSQL coverage for transfer, release, finalization, persistence,
  outbox publication, and durable replay. The test source compiles with the
  integration build tag; runtime execution is not claimed below.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Acquire verifies soft-close owner/epoch and policy, freezes outgoing counters, records prior ownership and barrier position, transfers to `CloseOnly`, and initializes zero close counters | `PeriodPostingGate.acquirePostingBarrier`; `TestPostingBarrierAcquireAndReleaseRestoresNextSoftCloseEpoch`; HTTP handler evidence test | Implemented and focused locally tested |
| Stale epoch, competing owner, invalid close type, scope, and expected version reject without partial transfer | Typed command validation; domain stale-epoch test; HTTP invalid-close-type test; memory repository rollback path | Implemented and focused locally tested |
| Initial-hard-close release requires zero admissions, restores the prior owner/mode, increments the soft-close epoch, and clears close ownership | `PeriodPostingGate.releasePostingBarrier`; domain and HTTP acquire/release tests | Implemented and focused locally tested |
| Reclose or admitted-close release is rejected and evidence remains authoritative | Reclose and admitted-counter domain test; posting admission owner/counter enforcement | Implemented; PostgreSQL runtime qualification pending |
| Finalization freezes the close summary, retains watermark/finalized owner/fingerprint, changes to `HardClosed`, and clears active ownership only after the summary exists | `PeriodPostingGate.finalizePostingGate`; finalization/replay domain test; HTTP finalization test | Implemented and focused locally tested |
| Interrupted finalization is recoverable through authoritative status and a finalized gate cannot be released | Internal `GetPostingGateStatus` seam and immutable finalized state; public status operation is reserved for Story 10 | Internal seam implemented; interruption/runtime qualification remains open |
| `GL-SCR-03` shows owner, close type, barrier position, counters, watermark, summaries, blocked actions, and next action | `gl-posting-gate-workspace.tsx`; focused component tests | Implemented and locally tested; browser/accessibility qualification remains open |

## Verification executed

| Command | Result |
| --- | --- |
| `make db-migrate-inventory` | Pass; checksum inventory refreshed |
| `make db-migrate-check` | Pass; migration checksum inventory valid |
| `make db-sqlc-check` | Pass; generated persistence model synchronized and Go suite passed inside the check |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/gl ./internal/platform/httpapi ./cmd/api` | Pass |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^$'` | Pass; integration-tagged tests compile, no integration tests executed |
| `./node_modules/.bin/tsc --noEmit` from `web` | Pass |
| `./node_modules/.bin/vitest run src/app/gl-posting-gate-workspace.test.tsx --pool=forks --maxWorkers=2 --fileParallelism=false` from `web` | Pass; 3 tests |

## Qualification limits and source notes

- PostgreSQL/Testcontainers runtime, lock/concurrency behavior, transaction
  rollback under injected audit/outbox failure, browser keyboard/focus/reflow,
  assistive-technology review, live authorization, live FPM handoff, audit-chain
  verification, performance, and release qualification were not run. The
  repository instructions make persistence integration checks opt-in.
- The public `GetPostingGateStatus` API remains Story 10 scope. Story 7 exposes
  an internal GL-owned recovery seam and keeps the monitor action-result driven.
- No FPM aggregate, schema, or runtime adapter was added. The typed
  `PostingGateApplicationPort` is GL-owned and carries commands/results only.
- The delivery plan assigns FR-GL-006–008 to M3 while the EP-GL-001 roadmap
  header says M2. Existing backlog traceability retains this source conflict;
  it is not reclassified here.

## Source references

- `docs/specs/finance_domain_model_ddd.md` — §§5.3, 6.1, 14.3, especially the
  `PeriodPostingGate` barrier, release, finalization, and recovery contract.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` —
  FR-GL-006, FR-GL-007, and FR-GL-008.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — `GL-SCR-03`,
  `FPM-SCR-02`, and WF-6.1/WF-6.2 close and recovery behavior.
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md`,
  `04_events_workers_integration_specifications_v1.0.md`,
  `05_frontend_ui_technical_specifications_v1.0.md`,
  `09_testing_performance_recovery_specifications_v1.0.md`, and
  `10_technical_traceability_decisions_v1.0.md`.
- `contracts/openapi/paths/general-ledger.yaml` — existing generic action
  routes, headers, permissions, and established/problem result contracts.
