# DLV-FR-GL-011 / DLV-FR-GL-012 / DLV-FR-GL-013 — User Story 9 verification evidence

Status: implementation added on branch `feat/gl-us9-operational-reopen-reclose`.
The GL-owned operational-reopen transitions, actor/class admission boundary,
expiry handling, reclose ownership transfer, generic action handlers, and
focused API/domain/UI coverage are implemented. PostgreSQL runtime,
cross-context handoff, browser, accessibility, and release qualification remain
explicit boundaries.

## Traceability

- Epic: EP-GL-001 — General Ledger
- Story: User Story 9 — Open and close operational reopen, then begin reclose
- Delivery: DLV-FR-GL-011 / FR-GL-011; DLV-FR-GL-012 / FR-GL-012;
  DLV-FR-GL-013 / FR-GL-013
- Operations: `glOpenOperationalReopenGate`,
  `glCloseOperationalReopenGate`, `glBeginRecloseGate`
- Command identities: `open-operational-reopen-gate.v1`,
  `close-operational-reopen-gate.v1`, `begin-reclose-gate.v1`
- Event types: `operational_reopen_gate_opened`,
  `operational_reopen_gate_expired`, `operational_reopen_gate_closed`,
  `reclose_gate_begun`
- Permissions: `finance.gl.open.operational.reopen.gate`,
  `finance.gl.close.operational.reopen.gate`,
  `finance.gl.begin.reclose.gate`
- Primary screen: `GL-SCR-03`; supporting posting context: `GL-SCR-01` and
  `GL-SCR-02`
- Branch: `feat/gl-us9-operational-reopen-reclose`

## Implementation evidence

- `internal/gl/operational_reopen_gate.go` adds canonical actor and
  transaction-class scope, approval/epoch/expiry validation, operational
  open/close transitions, frozen-summary checks, command fingerprints, and
  the reclose handoff. Both scope dimensions are required and matched
  conjunctively at admission.
- `internal/gl/posting.go` and the PostgreSQL posting, approval, and reversal
  paths validate the operational request, authority epoch, expiry, actor, and
  transaction class at the GL admission boundary. PostgreSQL admission also
  verifies the active owner and counter epoch while advancing counters in the
  journal transaction.
- Expiry rejects new operational-reopen admissions while retaining the
  request owner. The first authoritative close observing expiry emits
  `operational_reopen_gate_expired` before the close event in the same commit.
- `BeginRecloseGate` requires a positive operational frozen summary, preserves
  that prior summary, records the prior reopen owner and epoch, transfers
  ownership to the `CloseRun`, records `NextLedgerPosition - 1` as the barrier
  position, initializes zero reclose counters, and sets `CloseType = Reclose`.
  Existing barrier release rules reject release of that reclose barrier.
- `internal/gl/soft_close_gate.go` and
  `internal/gl/postgres_soft_close_gate_repository.go` carry the new durable
  operation identifiers, authorization mappings, idempotency fingerprints,
  audit records, and transactional outbox events.
- `internal/platform/httpapi/gl_operational_reopen_gate_handler.go` maps the
  existing command envelope to the GL application port and returns established
  or typed problem results. The common OpenAPI scope now exposes `actorIds`.
- `web/src/app/gl-posting-gate-workspace.tsx` displays operational scope,
  authority epoch, expiry, counters, frozen evidence, ownership transfer, and
  the mandatory reclose path while suppressing release for `CloseType = Reclose`.
- No FPM or Workflow aggregate, schema, or adapter was added. The GL-owned
  `PostingGateApplicationPort` exposes the command/result handoff boundary.

## Acceptance traceability

| Requirement | Evidence | Result |
| --- | --- | --- |
| Open only a hard-closed gate into `OperationalReopen` with actor/class scope, expiry, authority epoch, request owner, approval, expected version, and zero counters | `PeriodPostingGate.openOperationalReopenGate`; operational gate domain and HTTP handler tests | Implemented and focused locally tested |
| Admit only matching request, epoch, actor, transaction class, purpose, gate version, and unexpired authorization | Actor-aware posting validation and PostgreSQL owner revalidation; allowed/denied/expiry tests | Implemented and focused locally tested; PostgreSQL runtime qualification pending |
| Close to `HardClosed` for permitted no-change, otherwise `CloseOnly` with retained owner and frozen summary | `closeOperationalReopenGate`; no-change and positive-admission tests | Implemented and focused locally tested |
| Expiry retains ownership and emits an expiry outcome before authoritative close; conflicting fingerprints are typed conflicts | Expiry admission test, ordered event test, memory idempotency replay/conflict test, durable operation mappings | Implemented and focused locally tested |
| Begin reclose requires positive operational evidence, preserves the prior summary, transfers ownership, records the barrier, initializes counters, and disallows release | `beginRecloseGate`; ownership/barrier/summary assertions and release-rejection test | Implemented and focused locally tested |
| Recovery uses authoritative GL state and stable command identity | Internal `GetPostingGateStatus` seam, durable idempotency configuration, command fingerprints, replay test | Implemented locally; interruption/outbox replay qualification pending |
| UI distinguishes operational authority and shows mandatory reclose recovery | `GL-SCR-03` implementation and operational open → close → begin-reclose UI test | Implemented and focused locally tested; browser/accessibility qualification pending |

## Verification executed

| Command | Result |
| --- | --- |
| `GOCACHE=/tmp/tally-go-cache go test ./internal/gl ./internal/platform/httpapi ./cmd/api` | Pass |
| `pnpm -C web exec vitest run src/app/gl-posting-gate-workspace.test.tsx` | Pass; 5 tests |
| `pnpm -C web run build` | Pass; TypeScript build and Vite production bundle |
| `GOCACHE=/tmp/tally-go-cache make api-generate-check` | Pass; OpenAPI validation, deterministic Go generation, and Go verification |
| `make api-ts-check` | Pass; deterministic TypeScript generation and contract fixture checks |
| `git diff --check` | Pass; no whitespace errors |

## Qualification limits and source notes

- PostgreSQL/Testcontainers runtime, lock/concurrency behavior, rollback under
  injected audit/outbox failure, outbox/inbox replay, interruption before and
  after ownership transfer, browser keyboard/focus/reflow, assistive
  technology review, live IAM authorization, live FPM handoff, audit-chain,
  performance, and release qualification were not run. Repository guidance
  makes persistence integration and release checks opt-in.
- The public `GetPostingGateStatus` operation remains Story 10 scope. This
  implementation uses the internal GL-owned recovery seam and action results.
- The approved operational scope is intentionally conjunctive: a posting must
  match both an approved actor and an approved transaction class. The posting
  `sourceAggregateType` is the transaction-class evidence; no new cross-context
  posting field was invented.
- The delivery plan and roadmap assign this work to M3 while the EP-GL-001
  header says M2. The conflict is recorded rather than silently reclassified.
- The DDD gate contract (§5.3) says a `ReopenRequest` active counter has a
  null `ControlOwnerEpoch`, while the Story 8 backlog and approved delivery
  plan require an authority epoch for reopen admission. This implementation
  follows the Story 8 backlog/plan by retaining the epoch in the active
  counter and dedicated reopen-authority field; the source conflict remains
  open for domain-owner resolution.

## Source references

- `docs/specs/finance_domain_model_ddd.md` — operational reopen, posting
  admission, immutable journal, frozen-summary, and reclose-barrier rules.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` —
  FR-GL-011 through FR-GL-013.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — `GL-SCR-03`,
  `GL-SCR-01`, `GL-SCR-02`, and reopen/reclose workflow behavior.
- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md`,
  `04_events_workers_integration_specifications_v1.0.md`,
  `05_frontend_ui_technical_specifications_v1.0.md`, and
  `10_technical_traceability_decisions_v1.0.md`.
