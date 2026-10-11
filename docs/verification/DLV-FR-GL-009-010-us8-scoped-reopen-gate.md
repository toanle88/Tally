# DLV-FR-GL-009 / DLV-FR-GL-010 — User Story 8 verification evidence

Status: implementation added on branch `feat/gl-us8-scoped-reopen-gate`.
The GL-owned scoped-reopen transitions, immutable authorization evidence,
posting admission checks, generic action handlers, persistence migration, and
focused UI/API coverage are implemented. PostgreSQL runtime, browser, and
accessibility qualification remain explicit boundaries.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 8 — Open and close a scoped reopen gate
- Delivery: DLV-FR-GL-009 / FR-GL-009; DLV-FR-GL-010 / FR-GL-010
- Operations: `glOpenScopedReopenGate`, `glCloseScopedReopenGate`
- Command identities: `open-scoped-reopen-gate.v1`,
  `close-scoped-reopen-gate.v1`
- Event types: `scoped_reopen_gate_opened`, `scoped_reopen_gate_closed`
- Permissions: `finance.gl.open.scoped.reopen.gate`,
  `finance.gl.close.scoped.reopen.gate`
- Primary screen: `GL-SCR-03`; correction context: `GL-SCR-01` and
  `GL-SCR-02`
- Branch: `feat/gl-us8-scoped-reopen-gate`

## Implementation evidence

- `internal/gl/scoped_reopen_gate.go` adds canonical account and
  transaction-class scopes, expiry and control-authority checks, approval
  evidence, no-change closure, mandatory-reclose handoff, typed conflicts,
  command fingerprints, and GL-owned open/close application-port methods.
- `internal/gl/posting.go` and the PostgreSQL posting/reversal adapters admit
  only matching `ReopenCorrection` requests and advance the active owner’s
  counters in the same journal transaction. Expiry is checked against the
  commit clock, not a caller-provided timestamp.
- `internal/gl/journal_reversal.go` and the reversal action handler carry
  optional `reopenRequestId` and `controlAuthorityEpoch` evidence. When both
  are supplied, the existing linked-reversal workflow emits a
  `ReopenCorrection` journal through the scoped gate; ordinary reversals keep
  their existing `PolicyAdjustment` behavior.
- `db/migrations/gl/00008_add_scoped_reopen_gate_state.sql` persists active
  reopen ownership, scope, expiry, authority epoch, approval evidence, and
  no-change policy. It also persists correction scope/expiry on journal facts
  and protects that evidence with a posted-journal immutability trigger.
- `internal/gl/postgres_soft_close_gate_repository.go` locks and persists the
  complete gate state, audit evidence, durable idempotency result, and
  transactional outbox event in the existing GL transaction boundary.
- `internal/platform/httpapi/gl_scoped_reopen_gate_handler.go` maps the
  existing generic command envelope to typed GL commands and returns the
  established/problem result contracts. The posting result contract now
  exposes reopen request, scope, expiry, and authority epoch evidence.
- `web/src/app/gl-posting-gate-workspace.tsx` shows the approved scope,
  expiry, owner, counters, frozen summary, admission status, and mandatory
  reclose handoff. `web/src/app/gl-posting-workspace.tsx` carries correction
  request and authority evidence and displays returned gate evidence.
- No FPM or Workflow aggregate, schema, or adapter was added. The
  `PostingGateApplicationPort` remains a GL-owned command/result boundary.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Open only a hard-closed gate into `ScopedReopen` with canonical scope, expiry, request owner, authority epoch, approval, and zero counters | `PeriodPostingGate.openScopedReopenGate`; scoped gate domain test; scoped gate HTTP handler test | Implemented and focused locally tested |
| Admit only matching scoped corrections with current period/gate version, request owner, authority epoch, unexpired authorization, account scope, and transaction class | `validatePostingGateEvidenceAt`; PostgreSQL lock/admission path; scoped gate domain test | Implemented and focused locally tested; PostgreSQL runtime qualification pending |
| Close to `HardClosed` for approved no-change, otherwise `CloseOnly` with retained owner and frozen summary | `PeriodPostingGate.closeScopedReopenGate`; no-change and admitted-correction domain/API tests | Implemented and focused locally tested |
| Preserve owner and summary after expiry; safely replay identical commands and reject changed fingerprints | expiry evidence test; durable/in-memory service fingerprint paths; replay conflict test | Implemented and focused locally tested |
| Keep established journals immutable and corrections append-only | existing reversal workflow, including scoped linked-reversal evidence; correction-purpose validation; journal evidence immutability migration trigger | Implemented; PostgreSQL runtime qualification pending |
| Show scope, expiry, owner, counters, frozen summary, admission purpose, and reclose guidance | `GL-SCR-03`, posting request/result UI, UI fixtures | Implemented; TypeScript passed, Vitest execution timed out in this environment |

## Verification executed

| Command | Result |
| --- | --- |
| `make db-migrate-check` | Pass; migration checksum inventory valid |
| `GOCACHE=/tmp/tally-go-cache make db-sqlc-check` | Pass; SQLC generation and Go suite passed |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/gl ./internal/platform/httpapi ./cmd/api` | Pass |
| `GOCACHE=/tmp/tally-go-cache make check-fast` | Pass; migration validation/checksum and full Go suite |
| `GOCACHE=/tmp/tally-go-cache go test -tags=integration ./internal/platform/database -run '^$'` | Pass; integration-tagged database tests compile, no runtime test executed |
| `GOCACHE=/tmp/tally-go-cache make api-generate-check` | Pass; OpenAPI lint, bundle, deterministic Go generation, and Go verification passed |
| `make api-ts-check` | Pass; deterministic TypeScript generation and contract fixture checks passed |
| `./node_modules/.bin/tsc --noEmit` from `web` | Pass |
| `git diff --check` | Pass |
| `timeout 60s ./node_modules/.bin/vitest run src/app/gl-posting-gate-workspace.test.tsx src/app/gl-posting-workspace.test.tsx --pool=forks --maxWorkers=2 --fileParallelism=false` | Timed out after Vitest printed `RUN`; no test result was produced |

## Qualification limits and source notes

- PostgreSQL/Testcontainers runtime, lock/concurrency behavior, rollback under
  injected audit/outbox failure, browser keyboard/focus/reflow, assistive
  technology review, live IAM authorization, live FPM handoff, audit-chain
  verification, performance, and release qualification were not run. The
  repository instructions make persistence integration checks opt-in.
- The public `GetPostingGateStatus` operation remains Story 10 scope. This
  implementation uses the internal GL-owned recovery seam and action results.
- The approved scope uses existing posting `sourceAggregateType` as the
  transaction-class evidence; no new cross-context posting field was invented.
- The delivery plan and roadmap assign this work to M3 while the EP-GL-001
  header says M2. The conflict is recorded rather than silently reclassified.
- The DDD gate contract (§5.3) says a `ReopenRequest` active counter has a
  null `ControlOwnerEpoch`, while the Story 8 backlog and approved delivery
  plan require an authority epoch for scoped-reopen admission. This
  implementation follows the Story 8 backlog/plan by retaining the epoch in
  both the active counter and the dedicated reopen-authority field; the source
  conflict remains open for domain-owner resolution.

## Source references

- `docs/specs/finance_domain_model_ddd.md` — scoped reopen, posting admission,
  immutable journal, and frozen-summary rules.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` —
  FR-GL-009 and FR-GL-010.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — `GL-SCR-03`,
  `GL-SCR-01`, `GL-SCR-02`, and reopen/reclose workflow behavior.
- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md`,
  `03_database_persistence_specifications_v1.0.md`,
  `04_events_workers_integration_specifications_v1.0.md`,
  `05_frontend_ui_technical_specifications_v1.0.md`, and
  `10_technical_traceability_decisions_v1.0.md`.
