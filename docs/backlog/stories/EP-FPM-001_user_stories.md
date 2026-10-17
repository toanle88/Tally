# EP-FPM-001 — Fiscal Period Management User Stories

| Field | Value |
|---|---|
| Epic | EP-FPM-001 |
| Capability | Fiscal period management |
| Milestone | M3 in the roadmap and delivery plan; the delivery plan also identifies an M2 open-period slice |
| Status | Parent marked done by explicit user-requested status override on 2026-10-17; focused local evidence exists for US2–US6, while child delivery items and epic quality gates remain open |
| Owning module | internal/fiscalperiod |
| Owning schema | fiscal_period |
| Primary users | Finance Manager, Controller, Close Coordinator |
| Dependencies | EP-GL-001 and EP-WFA-001 |
| Source of truth | finance_domain_model_ddd.md, PRD, UX workflow specification, NFR, system design, technical specifications, finance delivery plan, ROADMAP.md |

This document is a planning and delivery-traceability artifact. It decomposes EP-FPM-001 into implementable user stories and records focused implementation evidence for US2–US6. The parent status override is a planning/status decision; it does not claim that the child delivery items, US1 coverage, integrations, UI qualification, persistence integration, or quality gates are complete.

The current repository contains the approved fiscal-period action contracts and generated client artifacts, GL posting-gate implementation owned by the GL module, and focused US2–US6 FPM runtime/persistence/API/UI slices. US1 delivery and the remaining FPM qualification evidence remain open. Existing user changes in the worktree are preserved.

## 1. Outcome

Finance users can control a fiscal period through soft close, hard close, approved reopen, and mandatory reclose while preserving one authoritative posting-gate owner, immutable control history, approval evidence, auditability, and recoverability after dependency or process interruption.

The completed epic must make the period-control state visible and explainable at every material transition:

- the fiscal period state and scoped accounting ownership;
- the active control process, control epoch, gate mode, gate version, and responsible owner;
- close tasks, exceptions, approvals, admitted posting summaries, ledger watermarks, and seal status;
- whether a failure is a domain rejection, authorization failure, version/concurrency conflict, dependency unavailability, or internal failure;
- whether a reopen completed with no admitted posting or requires a mandatory reclose.

FPM owns period-control state and lifecycle decisions. GL owns posting-gate and journal authority. Workflow owns approval decisions. Source contexts own correction postings. FPM must consume authoritative results from those contexts rather than treating caller assertions or eventual projections as proof.

## 2. Learning objective

Implementers should learn how a modular-monolith aggregate boundary coordinates a long-running, approval-bearing financial control process without moving ownership across modules:

- model FiscalPeriod, SoftCloseRun, CloseRun, and ReopenRequest with immutable lineage and aggregate versioning;
- make state-changing commands idempotent and safe under duplicate, retry, stale-version, and concurrent delivery;
- coordinate FPM with the GL posting-gate contract and Workflow approval evidence through explicit ports;
- recover from interruption using authoritative status reconciliation;
- expose safe, accessible progress and exception states without weakening a restrictive gate;
- verify domain, persistence, API, integration, authorization, audit, and operational evidence together.

## 3. Scope

This epic plans the following delivery surface:

- FPM domain aggregates, commands, events, policies, invariants, and state transitions;
- FPM application handlers and ports for GL gate, Workflow decisions, IAM/authorization, audit, close-task dependencies, reporting, and seal/reference operations;
- fiscal_period migrations, repositories, optimistic concurrency, idempotency records, and immutable lineage;
- the approved FPM HTTP command routes and versioned integration command/event handling;
- transactional outbox and inbox/deduplication behavior for FPM-owned outcomes;
- period-control dashboard and close, reopen, recovery, and exception-review surfaces described by the UX specification;
- progress, aging, dependency, exception, audit, and recovery observability;
- focused tests and evidence for normal, boundary, duplicate, concurrency, dependency-failure, interruption, and recovery paths.

The implementation must remain within the approved modular-monolith architecture. Changes to an FPM aggregate and its control record commit inside the FPM owning module. Cross-context effects use approved ports, commands, events, or reference operations; FPM does not read another module's adapter or database schema directly.

## 4. Explicit exclusions

The stories do not authorize:

- GL posting-gate, journal, barrier, ledger-watermark, or operational-reopen implementation; those remain GL-owned dependencies;
- Workflow approval policy, approval decision, delegation, or SoD implementation; FPM consumes immutable Workflow decisions;
- FX, fixed-asset, revenue, intercompany, reporting, audit-integrity, or source-context posting implementations;
- a second accounting producer or any journal-writing behavior in FPM;
- new command names, event names, routes, schemas, or architecture decisions not present in the approved specifications;
- direct access to another module's tables, repositories, or adapters;
- treating an unavailable dependency as a successful approval, open period, released gate, or completed close;
- marking any child delivery item or quality gate complete; parent epic status is controlled separately by an explicit roadmap/status decision;
- production, release, performance, or qualification claims without command/test evidence.

## 5. Ownership and domain rules

### 5.1 Aggregate and module ownership

| Area | FPM responsibility | Other owner |
|---|---|---|
| FiscalPeriod | Scoped period identity, status, expected versions, lifecycle transitions | — |
| SoftCloseRun | Soft-close policy, control epoch, handoff and exit lineage | GL owns the active posting gate |
| CloseRun | Initial hard-close/reclose process, checklist state, barrier evidence, finalization state, seal reference | GL owns barrier/ledger evidence; Audit owns seal processing |
| ReopenRequest | Reason, mode, scope, correction package, expiry, approval reference, admission outcome, reclose path | Workflow owns decisions; GL owns reopen gate and admitted-posting summary |
| Posting gate | Request and apply authoritative GL results | GL |
| Approval | Submit/consume decision references and revalidate current state | Workflow |
| Journal/posting | Never originate or edit journals | GL and source contexts |
| Audit | Emit auditable facts and reference audit/seal outcomes | Audit |

### 5.2 Invariants that every story must preserve

1. A fiscal period state is unique within its scoped accounting scope and calendar period.
2. At most one period-control process owns the posting gate. A candidate close or reclose may exist without admission authority, but there must be no dual owner.
3. Hard close requires mandatory steps, resolved or approved exceptions, required approvals, authoritative posting-gate evidence, and a final ledger watermark.
4. FPM obtains the authoritative GL barrier before transitioning a period to Closing.
5. Earlier gate versions are rejected after a barrier; pre-barrier and close postings must be terminal or reconciled before finalization.
6. GL remains authoritative for owner, gate epoch, admitted counters, positions, and posting-gate status.
7. Soft-close, handoff, barrier, release/finalization, reopen, reclose, and status-reconciliation commands are idempotent by the specified scope/process identity. Conflicting parameters produce a typed conflict and no state change.
8. An initial hard-close barrier can be released only when no close posting was admitted. A reclose barrier is never releasable.
9. If GL finalizes before FPM persists the result, FPM reconciles with GetPostingGateStatus and completes the transition idempotently; it never releases a finalized gate.
10. Reopen admission is determined by the GL gate and the approved request, not by caller assertion. Expiry rejects new postings while retaining control ownership.
11. A reopen with no admitted posting completes with no change. Any admitted correction requires a mandatory reclose with a new close-run identity and preserved prior lineage.
12. Material transitions retain authorization, approval, reason, expected-version, correlation, causation, and audit evidence.

### 5.3 State and evidence vocabulary

The stories use the source-defined states and evidence:

- FiscalPeriod: Open, SoftClosed, Closing, HardClosed, Reopening, and nonterminal ExpiredPendingClosure;
- SoftCloseRun: active control epochs, HandoffPending, Active, Completed, or Superseded;
- CloseRun: candidate/initiating, barrier-acquired, in progress, completed, aborted, and seal states NotRequested, SealPending, Sealed, or SealFailed;
- ReopenRequest: Draft, PendingApproval, Approved, GateOpening, Active, ExpiredPendingClosure, GateClosing, CompletedNoChange, RecloseHandoff, RecloseInProgress, Completed, Rejected, Cancelled, or Failed;
- authoritative gate evidence: gate version, owner/epoch, frozen admission summary, barrier ledger position, final watermark, and GL status reference.

## 6. Traceability

### 6.1 Story coverage of the FPM delivery plan

Each direct FPM delivery item is assigned to one story. Cross-cutting use of ApplyPostingGateResult is listed as supporting in later stories but is implemented and directly traced in US2.

| Story | Direct delivery items | Functional requirements | Primary action |
|---|---|---|---|
| US1 | DLV-FR-FPM-001, DLV-FR-FPM-002 | FR-FPM-001, FR-FPM-002 | StartSoftClose, EndSoftClose |
| US2 | DLV-FR-FPM-003, DLV-FR-FPM-006 | FR-FPM-003, FR-FPM-006 | StartHardClose, ApplyPostingGateResult |
| US3 | DLV-FR-FPM-004, DLV-FR-FPM-005, DLV-FR-FPM-007, DLV-FR-FPM-008 | FR-FPM-004, FR-FPM-005, FR-FPM-007, FR-FPM-008 | ResumeCloseRun, AbortCloseRun, ApplyCloseExceptionApprovalDecision, ApplyCloseApprovalDecision |
| US4 | DLV-FR-FPM-009, DLV-FR-FPM-010 | FR-FPM-009, FR-FPM-010 | RequestReopen, ApplyReopenApprovalDecision |
| US5 | DLV-FR-FPM-011 | FR-FPM-011 | StartReclose |
| US6 | DLV-FR-FPM-012, DLV-FR-FPM-013 | FR-FPM-012, FR-FPM-013 | TakeOverPeriodControl, ExtendCloseException |

### 6.2 Workflow and UX traceability

| Story | Workflow/source requirements | UX surfaces and acceptance anchors |
|---|---|---|
| US1 | WF-7.12; FR-FPM-001, FR-FPM-002 | FPM-WS-01, FPM-SCR-01; UXA-7-12-01, UXA-7-12-02 |
| US2 | WF-6.1; FR-FPM-003, FR-FPM-006 | FPM-WS-01, FPM-SCR-02, GL-SCR-03; UXA-6-1-01, UXA-6-1-02 |
| US3 | WF-6.1; FR-FPM-004, FR-FPM-005, FR-FPM-007, FR-FPM-008 | FPM-SCR-02, FPM-SCR-05, WFA-SCR-02; UXA-6-1-03, UXA-6-1-04 |
| US4 | WF-6.2; FR-FPM-009, FR-FPM-010 | FPM-WS-01, FPM-SCR-03, GL-SCR-03, WFA-SCR-02; UXA-6-2-01, UXA-6-2-04 |
| US5 | WF-6.2; FR-FPM-006, FR-FPM-011 | FPM-SCR-02, FPM-SCR-03, GL-SCR-03; UXA-6-2-02, UXA-6-2-03 |
| US6 | WF-7.12; FR-FPM-001, FR-FPM-012, FR-FPM-013 | FPM-WS-01, FPM-SCR-04, FPM-SCR-05, FPM-SCR-03; UXA-7-12-01 through UXA-7-12-04 |

The workflow dependencies are supporting requirements owned by the relevant modules. FPM must expose the state and evidence needed by those workflows, but must not claim completion of another context's delivery item.

### 6.3 Supporting cross-cutting requirements

The epic contributes evidence to, but does not complete, the cross-cutting delivery items owned elsewhere:

- GFR-002 and GFR-003: default-deny authorization, high-risk authentication, and separation of duties;
- GFR-004: approval-bearing actions routed through Workflow;
- GFR-006, GFR-007, and GFR-008: idempotency, conflict detection, and optimistic concurrency;
- GFR-009: visible blocked, pending, and recovery states;
- GFR-012 and GFR-013: cross-context state and immutable lineage;
- GFR-014: audit evidence;
- GFR-017: dependency unavailable versus business rejection.

Relevant NFR evidence includes NFR-REL-001, NFR-REL-002, NFR-REL-003, NFR-REL-004, NFR-REL-008, NFR-SEC-002, NFR-SEC-003, NFR-SEC-004, NFR-SEC-005, NFR-SEC-012, NFR-REC-001, NFR-REC-005, NFR-REC-007, NFR-REC-010, NFR-REC-012, NFR-OBS-001, NFR-OBS-002, NFR-OBS-003, NFR-OBS-006, NFR-OBS-007, NFR-OBS-008, NFR-INT-001 through NFR-INT-005, NFR-ACC-002, NFR-ACC-004, NFR-ACC-006, NFR-ACC-007, and NFR-ACC-008.

### 6.4 Approved API and integration surface

The technical specifications define these POST routes and operation IDs. Every state-changing request uses the approved CommandRequest shape and idempotency header. Responses use the common established/problem contract and preserve typed distinctions for authorization, domain, version, idempotency, and dependency failures.

| Action | HTTP route | Operation ID | Story |
|---|---|---|---|
| StartSoftClose | /api/v1/fiscal-periods/actions/start-soft-close | fpmStartSoftClose | US1 |
| EndSoftClose | /api/v1/fiscal-periods/actions/end-soft-close | fpmEndSoftClose | US1 |
| StartHardClose | /api/v1/fiscal-periods/actions/start-hard-close | fpmStartHardClose | US2 |
| ResumeCloseRun | /api/v1/fiscal-periods/actions/resume-close-run | fpmResumeCloseRun | US3 |
| AbortCloseRun | /api/v1/fiscal-periods/actions/abort-close-run | fpmAbortCloseRun | US3 |
| ApplyPostingGateResult | /api/v1/fiscal-periods/actions/apply-posting-gate-result | fpmApplyPostingGateResult | US2 |
| ApplyCloseExceptionApprovalDecision | /api/v1/fiscal-periods/actions/apply-close-exception-approval-decision | fpmApplyCloseExceptionApprovalDecision | US3 |
| ApplyCloseApprovalDecision | /api/v1/fiscal-periods/actions/apply-close-approval-decision | fpmApplyCloseApprovalDecision | US3 |
| RequestReopen | /api/v1/fiscal-periods/actions/request-reopen | fpmRequestReopen | US4 |
| ApplyReopenApprovalDecision | /api/v1/fiscal-periods/actions/apply-reopen-approval-decision | fpmApplyReopenApprovalDecision | US4 |
| StartReclose | /api/v1/fiscal-periods/actions/start-reclose | fpmStartReclose | US5 |
| TakeOverPeriodControl | /api/v1/fiscal-periods/actions/take-over-period-control | fpmTakeOverPeriodControl | US6 |
| ExtendCloseException | /api/v1/fiscal-periods/actions/extend-close-exception | fpmExtendCloseException | US6 |

The approved command topics mirror these actions with version v1. FPM-owned outcomes use the source-defined event names: SoftCloseStarted, SoftCloseEnded, SoftCloseHandoffStarted, SoftCloseResumed, SoftCloseSuperseded, GateAdmissionSummaryRecorded, PeriodStateChanged, CloseStepCompleted, CloseRunResumed, CloseRunAborted, ReopenRequested, PeriodReopened, ReopenCompletedNoChange, RecloseHandoffStarted, OperationalReopenActivated, OperationalReopenRequestExpired, and PeriodReclosed. GL-owned gate events remain GL-owned.

## 7. User stories

### US1 — Start and end a scoped soft close

As a Finance Manager or Close Coordinator, I want to start and end a soft close for one scoped fiscal period so that posting is controlled during close preparation without changing another accounting scope or leaving an orphaned gate.

Direct traceability: DLV-FR-FPM-001, DLV-FR-FPM-002, FR-FPM-001, FR-FPM-002, StartSoftClose, EndSoftClose.

Supporting traceability: GL EnterSoftCloseGate and ExitSoftCloseGate, WF-7.12, FPM-WS-01, FPM-SCR-01, GFR-002, GFR-003, GFR-006, GFR-007, GFR-008, GFR-009, GFR-014, GFR-017.

Acceptance criteria:

- [ ] A permitted actor can start soft close only for an Open period in the requested accounting scope, with a valid policy version, expected period version, and idempotency key.
- [ ] StartSoftClose creates the scoped SoftCloseRun and control epoch 1, changes the period to SoftClosed, and requests GL to acquire the SoftClosePolicy gate with no close or reopen admission authority.
- [ ] The committed result records the GL gate version, policy version, control epoch, owner identity, correlation/causation, and audit reference. The period-control view shows the owner and restrictive state.
- [ ] A second scope can be controlled independently. A run or gate in one scope cannot mutate the period state of another scope.
- [ ] EndSoftClose is accepted only for the active matching run and epoch. It freezes the authoritative admission summary, records the handoff/exit evidence, asks GL to exit the gate, and returns the period to Open only after a successful authoritative result.
- [ ] Repeating the same command identity and fingerprint returns the same established outcome. Reusing an identity with different content returns an idempotency conflict and changes nothing.
- [ ] Stale expected versions, another active owner, an invalid policy, or a gate-version mismatch return typed conflicts or domain rejection without relaxing the current gate.
- [ ] An unavailable GL or audit dependency leaves the period in a restrictive, explainable state with a retry/reconciliation path; the UI does not imply that the period is open.
- [ ] The dashboard and soft-close screen expose status, gate mode/version, owner, control epoch, admission summary, pending dependency, and actionable error text with keyboard-accessible status and error announcements.

Expected implementation evidence:

- FPM aggregate and handler tests for normal start/end, scope isolation, duplicate, changed duplicate, stale version, owner conflict, and dependency-unavailable paths.
- GL port contract tests showing exclusive entry/exit and immutable epoch summary.
- Persistence tests for FiscalPeriod, SoftCloseRun, aggregate version, idempotency, outbox, and audit reference.
- API and UI tests for the approved route, permission, typed problem, and visible recovery state.

### US2 — Start hard close and acquire the authoritative posting barrier

As a Finance Manager, I want to start hard close and acquire the GL posting barrier so that the close process has exclusive admission control and a reliable ledger starting point.

Direct traceability: DLV-FR-FPM-003, DLV-FR-FPM-006, FR-FPM-003, FR-FPM-006, StartHardClose, ApplyPostingGateResult.

Supporting traceability: WF-6.1, GL AcquirePostingBarrier and GetPostingGateStatus, FPM-WS-01, FPM-SCR-02, GL-SCR-03, SoftCloseHandoffStarted, GateAdmissionSummaryRecorded, PeriodStateChanged, GFR-004, GFR-006, GFR-007, GFR-008, GFR-012, GFR-013, GFR-014, GFR-017.

Acceptance criteria:

- [ ] StartHardClose is accepted only for an eligible SoftClosed period with the expected active soft-close run, policy, gate version, authorized requester, and no competing period-control owner.
- [ ] FPM creates a candidate CloseRun with a new process identity in a non-owning Initiating state and sends GL an AcquirePostingBarrier request containing scope, period, expected gate, policy/process identities, and request fingerprint.
- [ ] GL atomically transfers gate ownership to the close run, changes the mode to CloseOnly, freezes the soft-close epoch, and records the barrier ledger position. FPM never treats the candidate as owning before the authoritative result.
- [ ] Applying the matching successful gate result records the immutable admission summary and barrier evidence, marks the soft-close handoff pending, marks the CloseRun barrier acquired, and changes the period to Closing.
- [ ] No state permits both the soft-close and close run to own the posting gate. A stale or earlier gate version rejects ordinary posting and cannot be applied to a different process.
- [ ] Duplicate delivery of the same gate result is harmless. A result with a changed fingerprint, wrong scope, wrong process, or incompatible expected version produces a typed conflict or integration exception with no partial state change.
- [ ] If barrier acquisition fails, the period remains SoftClosed with its existing soft-close owner. If GL succeeded but FPM did not persist, status reconciliation completes the same transition without issuing a release.
- [ ] The hard-close screen shows the barrier state, owner, gate version, frozen summary, ledger position, and blocked reason. It distinguishes dependency unavailability from a rejected business precondition.

Expected implementation evidence:

- Domain tests for candidate creation, barrier result application, owner exclusivity, and authoritative status recovery.
- GL port/integration contract tests for atomic barrier acquisition and immutable admission evidence.
- Persistence and outbox tests for CloseRun, control epoch handoff, idempotency, version conflicts, and event ordering.
- API, authorization, audit, and UI tests for StartHardClose and ApplyPostingGateResult.

US2 implementation evidence delivered on branch `feat/dlv-fr-fpm-001-us2-start-hard-close`:

- `internal/fiscalperiod/hard_close.go` models the eligible `SoftClosed` handoff, creates a non-owning `Initiating` `CloseRun`, invokes the GL-owned `AcquirePostingBarrier` port, applies only a matching authoritative `CloseOnly` result, retains frozen admission and barrier evidence, and records `GateAdmissionSummaryRecorded`, `SoftCloseHandoffStarted`, and `PeriodStateChanged` events.
- `internal/fiscalperiod/postgres_repository.go` persists the FPM control records with row locks, optimistic versions, audit participation, and transactional outbox publications without reading GL tables.
- `internal/platform/httpapi/fiscal_period_handler.go` implements the approved `fpmStartHardClose` and `fpmApplyPostingGateResult` command routes with authorization, typed conflicts, dependency-unavailable responses, and established results.
- `db/migrations/bootstrap/00007_create_fiscal_period_schema.sql` and `db/migrations/fiscal_period/00001_create_fiscal_period_tables.sql` establish the FPM-owned persistence boundary; migration validation and checksum inventory are updated.
- `web/src/app/fiscal-period-hard-close-workspace.tsx` and the FPM route entries expose pending, restrictive, blocked, and authoritative barrier evidence states for `FPM-WS-01` and `FPM-SCR-02`.
- Focused domain, handler, route, and component tests cover candidate creation, owner exclusivity, replay/conflict behavior, dependency failure, authoritative result application, authorization boundary, and visible recovery state.

The documented local verification commands for this slice have passed. PostgreSQL/Testcontainers runtime, browser/accessibility qualification, and the remaining FPM stories are explicitly deferred.

### US3 — Execute, approve, resume, or abort a hard-close run

As a Close Coordinator, I want to execute close tasks, resolve exceptions, apply approval decisions, and recover or abort safely so that a hard close cannot finalize until its evidence is complete.

Direct traceability: DLV-FR-FPM-004, DLV-FR-FPM-005, DLV-FR-FPM-007, DLV-FR-FPM-008, FR-FPM-004, FR-FPM-005, FR-FPM-007, FR-FPM-008, ResumeCloseRun, AbortCloseRun, ApplyCloseExceptionApprovalDecision, ApplyCloseApprovalDecision.

Supporting traceability: WF-6.1, close-task ports for FX/fixed assets/revenue/intercompany/reporting/audit, GL FinalizePostingGate and ReleasePostingBarrier, FPM-SCR-02, FPM-SCR-05, WFA-SCR-02, CloseStepCompleted, CloseRunResumed, CloseRunAborted, GFR-002, GFR-003, GFR-004, GFR-008, GFR-009, GFR-014, GFR-017.

Acceptance criteria:

- [ ] ResumeCloseRun restarts an interrupted eligible run using the existing process identity, current aggregate version, idempotency identity, and authoritative gate status; it does not create a second owner.
- [ ] The run records each mandatory close step, dependency outcome, progress stage, count/time/owner/blocker information, and immutable evidence reference. Long-running work is accepted quickly and reports progress.
- [ ] ApplyCloseExceptionApprovalDecision accepts only a decision for the current close run, exception, policy/version, and approval request. It records the immutable Workflow decision reference and rejects stale, duplicate-with-changed-content, or wrong-run decisions.
- [ ] ApplyCloseApprovalDecision requires the current close state, required approval references, actor authorization, and SoD evidence. It cannot bypass unresolved mandatory tasks, unapproved exceptions, posting reconciliation, or a changed gate.
- [ ] The finalization request to GL includes the final admitted-posting evidence and ledger watermark. A successful authoritative finalization transitions the period to HardClosed and the CloseRun to Completed with SealPending; the gate is never released.
- [ ] If initial close has no admitted close posting, an authorized abort can release the initial barrier through the GL contract and open the next soft-close epoch. An abort after admitted posting, or any reclose abort, leaves the gate restrictive and requires reconciliation.
- [ ] If GL finalizes before FPM persists the result, ResumeCloseRun or status reconciliation applies the result idempotently. A seal failure leaves accounting close HardClosed with SealFailed and a retryable seal state.
- [ ] The UI distinguishes task failure, unresolved/expired exception, invalidated approval, dependency unavailability, finalization pending, and seal pending/failed. It never presents a failed seal as an open period.
- [ ] Audit evidence links the run, approvals, exceptions, gate evidence, final watermark, actor, and outcome without exposing sensitive values in logs or operational metrics.

Expected implementation evidence:

- Application tests for task orchestration, progress, approval application, finalization, resume, initial-abort, and seal failure.
- Contract tests for all close-task and GL ports, including dependency rejection versus unavailability.
- Concurrency/idempotency tests for simultaneous approval, resume, abort, and finalization.
- API, UI accessibility, audit, and observability tests for blocked and recovery states.

US3 implementation evidence delivered on branch `feat/dlv-fr-fpm-001-us3-close-run`:

- `internal/fiscalperiod/close_run.go` implements task execution/progress, current Workflow decision application, final approval and SoD checks, GL finalization, initial-barrier abort, authoritative release/finalization recovery, and separate audit-seal state.
- Close-run persistence retains task/exception/approval/finalization/release/seal evidence. PostgreSQL preserves an aborted run row when a later initial hard-close attempt creates a new close-run identity and selects active/retryable runs for recovery.
- `internal/platform/httpapi/fiscal_period_handler.go` exposes `ResumeCloseRun`, `AbortCloseRun`, `ApplyCloseExceptionApprovalDecision`, and `ApplyCloseApprovalDecision` through the approved routes with typed conflict, authorization, dependency, and approval errors.
- `web/src/app/fiscal-period-hard-close-workspace.tsx` exposes execution progress, exception and approval evidence, abort/resume actions, finalization-pending state, and seal-pending/failed state without presenting a failed seal as an open period.
- Focused domain tests cover barrier retry identity, task failure/retry, immutable approval evidence, finalization recovery, initial abort/release recovery, new identity after abort, and stable seal retry identity. The verification record is `docs/verification/DLV-FR-FPM-004-005-007-008-us3-close-run.md`.

The documented US3 local verification commands have passed. PostgreSQL/Testcontainers runtime, browser/accessibility qualification, slow/release checks, US4/US5, and epic completion remain explicitly deferred.

### US4 — Request and authorize a scoped or operational reopen

As a Controller, I want to request and approve a bounded reopen so that an established period can admit only the authorized correction work for a limited time.

Direct traceability: DLV-FR-FPM-009, DLV-FR-FPM-010, FR-FPM-009, FR-FPM-010, RequestReopen, ApplyReopenApprovalDecision.

Supporting traceability: WF-6.2, FPM-WS-01, FPM-SCR-03, GL-SCR-03, WFA-SCR-02, ReopenRequested, PeriodReopened, OperationalReopenActivated, GFR-002, GFR-003, GFR-004, GFR-006, GFR-007, GFR-008, GFR-012, GFR-013, GFR-014.

Acceptance criteria:

- [ ] RequestReopen is accepted only for an eligible HardClosed period and records mode, reason, correction scope, affected classes/accounts, expected reversals/replacements, impact, maximum duration, expected gate version, control authority epoch, requester, and correlation.
- [ ] The request enters PendingApproval without changing the fiscal-period state or opening a GL gate. A pending request cannot admit postings.
- [ ] ApplyReopenApprovalDecision consumes an immutable Workflow decision, revalidates current period/gate/version/SoD/authorization data, and records approval or rejection evidence. Rejection or invalidation leaves the period HardClosed and does not open a gate.
- [ ] An approved ScopedCorrection request asks GL to open only the approved correction gate. An approved Operational request validates bounded classes, actors, authority epoch, expiry, and approval before asking GL to open the operational gate. FPM does not implement the GL gate.
- [ ] A successful authoritative gate result changes the period to Reopening and records the active reopen request, gate version, expiry, and admission restrictions. The UI shows exactly what is permitted and when it expires.
- [ ] An expired request rejects new postings while retaining control ownership and transitions to ExpiredPendingClosure. Expiry cannot silently return the period to an open state.
- [ ] Duplicate same-fingerprint requests/decisions are idempotent. Changed content, wrong approval request, stale version, wrong scope, or unauthorized actor returns a typed conflict/rejection with no gate change.
- [ ] Reopen request and approval evidence is immutable, auditable, linked to the original close, and sufficient to explain the later no-change or reclose outcome.

Expected implementation evidence:

- Domain tests for request lifecycle, approval revalidation, scoped versus operational constraints, expiry, and rejection.
- Workflow and GL port contract tests proving no gate opens before an approved current-state decision.
- Persistence, idempotency, concurrency, authorization/SoD, audit, API, and UI tests.

US4 implementation evidence delivered on branch `feat/dlv-fr-fpm-009-010-us4-reopen`:

- `internal/fiscalperiod/reopen.go` owns the immutable reopen request package,
  canonical scoped/operational constraints, pending approval, current Workflow
  decision revalidation, SoD checks, authoritative GL gate opening, active
  `Reopening` state, and the internal expiry reconciliation seam that records
  `ExpiredPendingClosure` without releasing the GL owner.
- `internal/fiscalperiod/postgres_repository.go` and
  `db/migrations/fiscal_period/00003_add_reopen_request_lifecycle.sql` retain
  the request, correction/impact evidence, approval evidence, gate result,
  expiry, control epoch, close lineage, optimistic versions, audit, and
  transactional outbox boundary inside the FPM module.
- `cmd/api/fiscal_period_reopen_workflow.go` is the composition-root adapter to
  Workflow's query service. FPM receives only the narrow validation port and
  does not read Workflow storage or tables directly.
- `internal/platform/httpapi/fiscal_period_reopen_handler.go` maps the
  approved generic command envelope to `fpmRequestReopen` and
  `fpmApplyReopenApprovalDecision`; `fiscal_period_handler.go` exposes the
  established request, approval, scope, expiry, gate, and next-action evidence.
- `web/src/app/fiscal-period-reopen-workspace.tsx` and the `fpmReopen` route
  implement `FPM-SCR-03`, keeping pending/rejected/stale/unavailable evidence
  visibly restrictive and showing the exact approved scope and expiry before
  active work.
- Focused domain, API-handler, route, and component tests cover pending/no-gate,
  rejection, scoped versus operational gate selection, Workflow/SoD and
  changed-idempotency rejection, expiry ownership retention, and visible UI
  state. The verification record is
  `docs/verification/DLV-FR-FPM-009-010-us4-reopen.md`.

Focused local verification passed. The repository fast gate's migration checks
and changed packages passed, while unrelated listener-based authentication and
database tests could not bind local TCP sockets in the restricted sandbox.
PostgreSQL/Testcontainers runtime, browser/accessibility qualification, and
epic completion remain explicitly deferred.

### US5 — Close an approved reopen and reclose safely

As a Controller, I want the approved reopen to close deterministically and start a mandatory reclose when a correction was admitted so that a hard-closed period cannot remain partially controlled.

Direct traceability: DLV-FR-FPM-011, FR-FPM-011, StartReclose.

Supporting traceability: ApplyPostingGateResult from US2, GL CloseScopedReopenGate, CloseOperationalReopenGate, BeginRecloseGate, WF-6.2, FPM-SCR-02, FPM-SCR-03, ReopenCompletedNoChange, RecloseHandoffStarted, PeriodReclosed, GateAdmissionSummaryRecorded, PeriodStateChanged, GFR-006, GFR-007, GFR-008, GFR-012, GFR-013, GFR-014, GFR-017.

Acceptance criteria:

- [ ] FPM applies the authoritative GL close-gate result to the matching active reopen request and records the final admission summary, close-gate version, and cutoff evidence.
- [ ] If no posting was admitted, FPM completes the request as CompletedNoChange, restores HardClosed, retains the prior final watermark and seal lineage, and does not create a reclose run.
- [ ] If any approved correction was admitted, FPM creates a new Reclose CloseRun with a new identity and preserved prior close/reopen lineage, requests GL BeginRecloseGate, and moves the period through RecloseHandoff/RecloseInProgress.
- [ ] A reclose barrier transfers exclusive gate ownership to the reclose run and is never releasable. Finalization requires mandatory reclose tasks, approvals, resolved exceptions, authoritative admitted-posting evidence, and a new ledger watermark.
- [ ] Successful reclose returns the period to HardClosed with the new watermark and seal state while retaining the original close, reopen request, admission summary, and correction lineage.
- [ ] Expiry or dependency failure leaves the current owner and restrictive state visible. It cannot be treated as no-change without an authoritative zero-admission summary.
- [ ] Duplicate start/reclose/result messages are idempotent; changed fingerprints, stale expected versions, wrong reopen identity, and concurrent competing runs produce typed conflicts without a second reclose owner.
- [ ] The UI explicitly explains the decision path: no admitted posting means no-change; admitted posting means mandatory reclose. Users cannot select a less restrictive outcome.

Expected implementation evidence:

- Domain and application tests for zero-admission no-change, positive-admission reclose, expiry, dependency failure, duplicate, and concurrency paths.
- GL handoff/finalization contract tests and status-reconciliation tests.
- Persistence tests proving new reclose identity and immutable prior-lineage retention.
- Workflow, API, UI, audit, and outbox tests for the complete reopen/reclose journey.

US5 implementation evidence delivered on branch `feat/dlv-fr-fpm-011-us5-reclose`:

- `internal/fiscalperiod/reclose.go` implements `StartReclose` as the FPM-owned
  close/reclose orchestration boundary. It closes the approved scoped or
  operational GL gate, applies the authoritative frozen admission summary,
  completes zero-admission requests as `CompletedNoChange`, and preserves the
  prior close-run watermark and seal without creating a second run.
- Positive admission creates a new `Reclose` `CloseRun`, records the prior
  reopen request and admission summary as lineage, invokes `BeginRecloseGate`,
  and moves the request and period through `RecloseHandoff`/
  `RecloseInProgress` and `Closing`. Reclose barriers remain outside the
  initial-close abort/release path.
- `internal/fiscalperiod/close_run.go` finalizes a reclose with the existing
  mandatory task/approval/watermark/seal controls, retains the original close
  and reopen evidence, and emits `PeriodReclosed`.
- `internal/gl/operational_reopen_gate.go` and the scoped GL contract preserve
  the typed reopen owner and prior gate mode during reclose handoff. The new
  migration `db/migrations/fiscal_period/00004_allow_reclose_close_runs.sql`
  permits the persisted `Reclose` close-run type without widening unrelated
  status constraints.
- `internal/platform/httpapi/fiscal_period_handler.go` exposes the established
  `fpmStartReclose` route and supports reopen-close results through the shared
  `fpmApplyPostingGateResult` boundary. The FPM workspace explains that GL,
  rather than the controller, determines no-change versus mandatory reclose.
- Focused domain, GL, and component tests cover zero-admission no-change,
  positive-admission handoff/finalization, scoped lineage, and the UI decision
  path. The verification record is
  `docs/verification/DLV-FR-FPM-011-us5-reclose.md`.

Focused local verification passed. PostgreSQL/Testcontainers, listener-based
authentication/database tests, browser/accessibility qualification, and
release/slow checks remain deferred; see the verification record for exact
commands and results.

### US6 — Recover period control and extend close exceptions

As a recovery operator or Close Coordinator, I want to take over an eligible interrupted period-control process and extend a justified close exception so that outages and aging work remain controlled without changing the process history or bypassing approval.

Direct traceability: DLV-FR-FPM-012, DLV-FR-FPM-013, FR-FPM-012, FR-FPM-013, TakeOverPeriodControl, ExtendCloseException.

Supporting traceability: WF-7.12, FPM-WS-01, FPM-SCR-04, FPM-SCR-05, AUD-SCR-04, SoftCloseResumed, CloseRunResumed, OperationalReopenRequestExpired, PeriodStateChanged, GFR-002, GFR-003, GFR-006, GFR-007, GFR-008, GFR-009, GFR-014, GFR-017.

Acceptance criteria:

- [ ] TakeOverPeriodControl is permitted only for an expired or otherwise eligible authority epoch with the required actor authorization, reason, evidence, approval, expected version, and current authoritative gate status.
- [ ] A takeover changes the responsible controller/owner while preserving the period-control process identity, control epoch lineage, gate identity, prior owner, timestamps, and audit history.
- [ ] Takeover does not open, release, or weaken a posting gate. If the dependency is unavailable, the period remains restrictive and the UI shows a pending reconciliation state.
- [ ] ExtendCloseException applies only to an identified current exception and close run, records the justification, new due/expiry time, actor, authorization, approval/evidence reference, and updated version, and never bypasses the final approval or gate requirements.
- [ ] Expired exceptions block finalization and remain visible as blockers. An extension does not retroactively approve an already rejected or finalized decision.
- [ ] Recovery can reconcile an interrupted GL result using GetPostingGateStatus and complete the matching FPM transition idempotently. It cannot infer a successful result from a timeout.
- [ ] Duplicate, stale, changed-content, wrong-process, unauthorized, and concurrent takeover/extension requests return typed outcomes without losing the existing owner or exception evidence.
- [ ] The recovery and exception screens show owner, authority/cutoff/expiry, gate state, pending actions, blocker age, next safe action, and audit evidence. Status changes are announced accessibly and are not color-only.

Expected implementation evidence:

- Domain, authorization, concurrency, idempotency, and recovery tests for takeover and exception extension.
- Status-reconciliation and dependency-unavailable tests proving restrictive behavior.
- Persistence/audit/outbox tests for preserved process identity and exception lineage.
- API/UI/observability tests for aging, cutoff, expiry, and operator guidance.

Current local evidence: [DLV-FR-FPM-012/013 US6 verification record](../../verification/DLV-FR-FPM-012-013-us6-period-control-recovery.md) and [RUN-005 period-control interruption runbook](../../operations/runbooks/run-005-period-control-interruption.md). Roadmap and acceptance completion markers remain unchanged until the deferred qualification evidence is available.

## 8. Cross-story definition of done

The explicit parent-status override does not change this evidence-based delivery definition of done.

The epic is ready for delivery review only when all of the following are evidenced:

- [ ] All 13 direct FR-FPM delivery items are implemented and linked to tests, API behavior, persistence, and user-visible outcomes.
- [ ] FiscalPeriod, SoftCloseRun, CloseRun, and ReopenRequest are implemented in internal/fiscalperiod with the fiscal_period schema and no cross-module table access.
- [ ] Aggregate versioning, idempotency, duplicate detection, typed conflicts, transactional outbox, and inbox/deduplication behavior cover every state-changing command.
- [ ] GL gate coordination preserves exclusive ownership, expected gate versions, immutable admission summaries, barrier positions, final watermarks, no-change/reclose branching, and status reconciliation.
- [ ] Workflow approval references are immutable, current-state revalidated, authorization/SoD checked, and auditable. FPM never originates or edits a Workflow decision.
- [ ] The approved API routes, operation IDs, permissions, schemas, generated artifacts, and problem responses are deterministic and backward compatible.
- [ ] The period-control dashboard and FPM-SCR-01 through FPM-SCR-05 expose state, owner, gate, progress, approval, exception, cutoff, expiry, recovery, and seal evidence with keyboard and screen-reader support.
- [ ] Logs, metrics, traces, audit records, and alerts distinguish rejection, authorization, concurrency, dependency, capacity, and internal failure without sensitive financial values.
- [ ] Normal, boundary, duplicate, changed duplicate, concurrency, stale version, dependency unavailable, interrupted process, partial completion, expiry, takeover, no-change, reclose, and seal-failure tests pass.
- [ ] Traceability is updated with actual implementation evidence, and no item is marked complete merely because its contract or generated artifact exists.
- [ ] M3 quality-gate evidence is available for QG-01, QG-02, QG-03, QG-04, QG-05, QG-06, QG-07, QG-08, and QG-10. QG-09 performance/capacity/recovery qualification remains subject to the delivery-plan milestone unless separately scheduled.

## 9. Ordered implementation plan

1. Freeze the story traceability and resolve the planning decisions below before coding. Confirm the exact GL, Workflow, audit, close-task, and read/status port contracts that the slice will consume.
2. Implement US1 first as the scoped open-period/soft-close slice: FPM aggregates, persistence, idempotency, GL entry/exit ports, handlers, events/outbox, permissions, dashboard state, and focused tests.
3. Implement US2: candidate hard-close run, atomic GL barrier handoff, ApplyPostingGateResult, authoritative admission evidence, optimistic concurrency, status reconciliation, and barrier UI.
4. Implement US3: close-task orchestration ports, progress/checklist state, exception and final approval application, resume/abort rules, finalization, watermark, seal-pending/failed handling, and recovery tests.
5. Implement US4: reopen request package, approval reference application, scoped/operational validation, GL gate-opening port, expiry, and approval/revalidation UI.
6. Implement US5: close-gate result application, no-admission completion, reclose identity/lineage, non-releasable reclose barrier, finalization, and mandatory-reclose tests.
7. Implement US6: authority epoch/takeover, exception extension, cutoff/expiry handling, status reconciliation, operator guidance, and recovery/observability tests.
8. For each slice, wire only the approved HTTP route, command topic, event, permission, audit action, and generated contract needed by that slice. Generate artifacts with the repository's existing tooling; do not hand-edit generated output.
9. Run focused package/API/UI tests and the repository fast check. Record exact commands and results. Defer opt-in slow, persistence-integration, and API release checks unless explicitly requested.
10. Update the delivery traceability and relevant documentation from verified repository evidence. Leave child delivery and qualification status unchanged until the corresponding implementation and qualification evidence exists; parent status is updated separately by an explicit status decision.

Likely implementation areas are internal/fiscalperiod, db/migrations/fiscal_period, existing API/worker wiring, the approved OpenAPI path and generated-client locations, web FPM screens, outbox/inbox infrastructure, audit/authorization ports, and runbook/observability documentation. Exact filenames and generated paths should be confirmed during implementation rather than invented in this planning artifact.

## 10. Risks and decisions required before implementation

1. Milestone wording differs by source. The delivery plan names an M2 EP-FPM-001 open-period slice and an M3 full FPM epic; ROADMAP.md maps EP-FPM-001 and its FR-FPM items to M3. This plan retains both facts and starts with US1 as the open-period/soft-close slice. The delivery owner must confirm whether US1 is the intended M2 boundary.
2. The action catalog defines the 13 FPM command routes but does not define a dedicated FPM read/query route for the period-control dashboard. Before UI implementation, agree the authorized read projection/status contract. Use GL GetPostingGateStatus as its approved reference operation; do not invent a new route in this plan.
3. Operational reopen appears in the DDD and WF-7.12, but the direct FPM actions are FR-FPM-001, FR-FPM-012, and FR-FPM-013 while GL owns the operational gate commands FR-GL-011 through FR-GL-013. Confirm the cross-context port and evidence contract without creating replacement FPM commands.
4. Hard-close orchestration depends on future or separately owned FX, fixed-asset, revenue, intercompany, reporting, and audit capabilities. FPM implementation should use explicit ports, contract fixtures, or scheduled dependencies and must not implement those epics early.
5. ApplyPostingGateResult is reused by soft-close handoff, hard-close finalization, reopen closure, and reclose. It is directly assigned to US2 and supporting in US5; the implementation must keep one handler contract with explicit result types and state guards rather than create action-specific duplicates.
6. The DDD describes SoftCloseRun control epochs separately from FiscalPeriod state and the GL gate. Persistence and UI must preserve those identities; a single mutable status field must not replace the epoch or authoritative gate evidence.
7. Accounting close and audit seal are separate outcomes. A seal pending or failed state must not reopen a HardClosed period, and retry behavior must preserve the established close fact.

## 11. Dependencies and readiness

| Dependency | Needed for | Readiness condition |
|---|---|---|
| EP-GL-001 | Soft-close gate, barrier, reopen/reclose gate, admission summary, final watermark, status reconciliation | GL ports/contracts and authoritative result semantics are available or explicitly scheduled |
| EP-WFA-001 | Close exception, close approval, reopen approval, SoD evidence | Workflow decision identity, immutable reference, current-state validation, and decision outcome contract are available |
| Close-task contexts | US3 and US5 task/checklist completion | Each dependency has a versioned status/evidence port; unavailable is distinguishable from rejection |
| Audit | All material transitions and seal references | Audit event/evidence contract and no-sensitive-value logging rules are available |
| IAM/authentication | High-risk close/reopen/takeover/extension actions | Permission names, MFA/assurance, actor identity, and SoD evaluation contract are available |
| Reporting and UI projections | Dashboard, completion visibility, statement/audit links | Authorized read/status projection contract is agreed |

Definition of Ready for each story is the delivery-plan definition: exact source IDs, owning module/record, UX state/errors, API/persistence/event/auth contracts, dependencies, and identified normal/correction/duplicate/concurrency/failure tests.

## 12. Planning verification

- [x] Read the repository AGENTS.md instructions and preserved existing worktree changes.
- [x] Read the FPM domain model, PRD/catalog/traceability, UX workflow specification, NFR, system design, technical specifications, delivery plan, and roadmap sections relevant to this epic.
- [x] Covered all 13 direct FR-FPM delivery items exactly once across six ordered stories.
- [x] Included scope, exclusions, ownership, invariants, dependencies, API/integration/auth/UI/observability surfaces, acceptance criteria, tests, risks, and quality-gate expectations.
- [x] Kept GL gate/journal ownership, Workflow decisions, and future context implementations outside FPM scope.
- [x] Recorded source ambiguities instead of silently inventing a resolution.
- [ ] No implementation, generated artifact change, migration, branch creation, child completion claim, or qualification claim has been made by this planning artifact.
- [x] The parent-status override is recorded separately from child delivery and quality-gate completion.
- [ ] Before implementation, create a feature branch from the current main branch. Suggested starting branch: feat/dlv-fr-fpm-001-us1-start-soft-close.

## 13. Parent status closure

EP-FPM-001 is marked done in `ROADMAP.md` by explicit user-requested parent-status override on 2026-10-17. The child delivery items and the cross-story definition of done remain evidence-based and open; the status override does not waive implementation, integration, authorization, audit, persistence, accessibility, recovery, production, or release qualification requirements.

