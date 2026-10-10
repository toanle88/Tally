# EP-GL-001 — General Ledger User Stories

| Field | Value |
|---|---|
| Epic | EP-GL-001 — General Ledger |
| Status | User Story 1 implementation is present on `feat/gl-us1-ledger-book-maintenance`; see the scoped verification record for open boundaries and unrun qualification checks |
| Milestone | M2 — General ledger and posting control |
| Delivery items | DLV-GFR-005, DLV-GFR-011, DLV-FR-GL-001 through DLV-FR-GL-018, DLV-WF-6.6 |
| Owning bounded context | General Ledger — `internal/gl`, `gl` schema |
| Primary users | Accountant; GL Manager; Controller; authorized subledger actor |
| Authoritative records | Ledger, AccountingBook, ChartOfAccounts, Account, JournalEntry, PeriodPostingGate |
| UX surfaces | `GL-WS-01`, `GL-SCR-01` through `GL-SCR-05`; supporting approval view `WFA-SCR-02` |
| Exit evidence | Domain, persistence, API, authorization, audit, idempotency, concurrency, UI, accessibility, integration, and recovery evidence for the scoped stories |

This document is the planning artifact for EP-GL-001. The roadmap epic and its
delivery items remain unchecked until implementation and verification evidence
exist. The stories below are intentionally grouped into coherent vertical
slices while preserving every source requirement and operation identifier.

## 1. Outcome

Deliver General Ledger as the authoritative owner of ledger configuration,
accounting books, charts of accounts, accounts, journal entries, posting
admission, posting gates, reversals, and ledger evidence.

Authorized users and approved domain processes can configure the accounting
structure, submit and validate balanced posting requests, apply immutable
approval decisions, post journals, correct established facts through linked
reversals, and control admission during close and reopen activity. Every
established accounting effect remains attributable to one owning producer and
retains the versions, approvals, scope, gate evidence, and correction lineage
needed to explain it later.

## 2. Learning objective

Learn how to deliver a high-integrity bounded context whose state-changing
commands combine exact monetary validation, optimistic concurrency,
idempotency, authorization, immutable correction, transactional outbox
publication, and user-visible recovery states in one owning-module boundary.

## 3. Scope

- Maintain effective-dated ledgers and accounting books.
- Maintain versioned charts of accounts, accounts, posting restrictions,
  currency policies, normal balances, and reporting mappings.
- Accept the standard version-2 posting request contract and validate its
  accounting scope, source identity, period, gate, currency, account, segment,
  and balancing controls.
- Establish journals directly when approval is not required, or establish
  `PendingApproval` without a ledger effect when approval is required.
- Apply an immutable Workflow approval decision after revalidating current GL
  state and posting conditions.
- Reverse a posted journal by appending a separate linked correction journal.
- Own the GL posting-gate transitions for soft close, hard-close barrier,
  scoped reopen, operational reopen, reclose handoff, and authoritative status
  inquiry.
- Expose the approved GL workbench, journal detail/result, gate monitor, and
  configuration screens with actionable states, blocking reasons, owners,
  evidence, and safe recovery paths.
- Apply explicit accounting scope, IAM authorization, segregation-of-duties
  checks, material-action audit evidence, idempotency, optimistic concurrency,
  safe telemetry, and transactional outbox behavior at applicable boundaries.

## 4. Explicit exclusions

- Fiscal Period Management owns `FiscalPeriod`, `SoftCloseRun`, `CloseRun`,
  `ReopenRequest`, period lifecycle, close orchestration, and reclose
  orchestration. GL owns the posting gate and journal admission decisions;
  it does not become the FPM owner.
- Workflow & Approvals owns approval policies, approval requests, approval
  decisions, delegation, and escalation. GL consumes an immutable decision
  reference and applies it to its own journal boundary.
- COA Segment Accounting owns segment definitions, values, combinations, and
  governed segment changes. GL validates and stores approved references through
  the published boundary; it does not write the `coa` schema.
- Subledgers own their commercial or operational source facts and remain the
  sole producer of their accounting effects. GL owns the authoritative journal
  record and rejects duplicate accounting ownership.
- Multi-Currency owns rate sets, revaluation, and translation. This epic uses
  the approved conversion-evidence contract where functional amounts are
  required; it does not deliver the FX capability.
- Financial Reporting, tax, consolidation, settlement, and downstream
  projections are not GL-owned records or mutation surfaces.
- No direct access to another bounded context's adapter, repository, or
  database schema.
- No destructive edit of a posted, accepted, or otherwise established
  financial fact.
- No new public route, request shape, response type, permission identifier,
  event, or persistence ownership outside the approved specifications.
- No production, live Entra, production audit-chain, full performance/capacity,
  disaster-recovery, or release qualification claim from local story work
  alone.

## 5. Ownership and domain rules

### 5.1 Owning component

General Ledger owns the following aggregate roots and the `gl` PostgreSQL
schema:

- `Ledger`
- `AccountingBook`
- `ChartOfAccounts`
- `Account`
- `JournalEntry`
- `PeriodPostingGate`

Other capabilities call GL application ports or consume published events. They
do not query or mutate GL tables directly. GL validates final journal effects,
but the context that owns a source business fact remains the producer of that
fact's posting request.

### 5.2 Journal and posting invariants

- A posting request identifies one tenant/accounting scope, legal entity,
  ledger, accounting book, functional currency, source aggregate/version,
  posting date, fiscal period, expected period/gate versions, purpose, and
  idempotency fingerprint.
- Version-2 posting requests contain exactly one transaction currency. A
  `TransactionAndFunctional` line uses the request currency; a
  `FunctionalOnlyAdjustment` line has zero transaction amount and a permitted,
  evidenced functional amount. Other mixed-currency representations are
  rejected or split into separately correlated requests.
- A journal has at least two lines and must balance both transaction and
  functional totals. Currency scale, rounding, account eligibility, segment
  combination, effective-date, ledger, book, legal-entity, and fiscal-calendar
  rules are validated before authoritative posting.
- GL validates the local period gate, expected gate version, period-state
  reference, posting purpose, authorization evidence, and any close/reopen
  scope in the same domain decision that admits the journal.
- Approval-required requests establish `PendingApproval` with no ledger effect.
  Approval application revalidates current conditions; it does not replay a
  stale validation result.
- A repeated accounting identity and fingerprint returns the established
  in-progress or terminal result. Reuse with changed content returns an
  idempotency conflict and creates no journal.
- Rejected posting attempts remain distinguishable from established journal
  facts. They do not create a journal entry.
- `Posted` journal entries are immutable. Corrections append a linked reversal,
  adjustment, replacement, or other approved correction owned by the relevant
  context. A journal cannot be reversed more than once without the controlled
  counter-reversal semantics defined by the DDD.

### 5.3 Configuration invariants

- A ledger has legal-entity ownership, ledger type, functional currency,
  fiscal calendar, lifecycle status, and effective dates.
- An accounting book belongs to an existing ledger and preserves its
  accounting basis, book type, posting-policy version, lifecycle, and effective
  dates.
- Charts of accounts are versioned and effective-dated. Account-code policy
  changes do not rewrite the chart or account version used by historical
  journals.
- Account codes are unique in the effective chart scope. Account type, normal
  balance, posting restrictions, currency policy, effective dates, and
  reporting mappings must be valid. Historical journal facts retain the
  applicable account/chart versions.

### 5.4 Posting-gate invariants

- `PeriodPostingGate` is unique by accounting scope and fiscal period. Gate
  version, active admission counters, ownership, and frozen summaries commit
  atomically with the gate decision or journal admission.
- At most one soft-close, close, scoped-reopen, operational-reopen, reclose,
  or takeover process owns a gate at a time. Process identifiers, expected
  versions, fingerprints, scope, epoch, expiry, and watermark are conflict
  checked.
- Soft close records the active process and control epoch. Exiting soft close
  freezes the epoch-qualified admission summary and restores the eligible
  mode only for the matching owner and epoch.
- Acquiring a hard-close barrier transfers ownership from the active soft-close
  process to the initial close run, records the barrier position, and changes
  the gate to `CloseOnly`. Releasing is permitted only for an initial hard
  close with zero close admissions. A reclose barrier is never releasable.
- Finalization freezes the active counters, retains the final ledger watermark,
  changes the gate to `HardClosed`, and clears active ownership only after the
  frozen summary is authoritative.
- Scoped and operational reopen gates require approved scope/authority,
  expiry, owner, and expected gate version. Expiry rejects new admissions but
  retains ownership until closure is authoritative.
- A reopen close with no admitted posting may restore `HardClosed` only under
  the approved no-change policy. Any admitted reopen posting requires a
  reclose handoff; `BeginRecloseGate` transfers ownership and establishes a
  non-releasable reclose barrier.
- `GetPostingGateStatus` is a read-only authoritative reference operation and
  never establishes a new financial fact.

## 6. Traceability

| Source requirement | Story coverage |
|---|---|
| `DLV-GFR-005` / GFR-005 | Stories 3–9: established facts are immutable and corrections use linked domain records. |
| `DLV-GFR-011` / GFR-011 | Stories 3–5: GL validates one authoritative accounting owner and preserves source identity. |
| `DLV-FR-GL-001` / FR-GL-001 | Story 3 — Submit and validate posting requests. |
| `DLV-FR-GL-002` / FR-GL-002 | Story 4 — Apply journal approval decisions. |
| `DLV-FR-GL-003` / FR-GL-003 | Story 5 — Reverse posted journal entries. |
| `DLV-FR-GL-004` / FR-GL-004 | Story 6 — Enter soft-close gates. |
| `DLV-FR-GL-005` / FR-GL-005 | Story 6 — Exit soft-close gates. |
| `DLV-FR-GL-006` / FR-GL-006 | Story 7 — Acquire posting barriers. |
| `DLV-FR-GL-007` / FR-GL-007 | Story 7 — Release initial posting barriers. |
| `DLV-FR-GL-008` / FR-GL-008 | Story 7 — Finalize posting gates. |
| `DLV-FR-GL-009` / FR-GL-009 | Story 8 — Open scoped reopen gates. |
| `DLV-FR-GL-010` / FR-GL-010 | Story 8 — Close scoped reopen gates. |
| `DLV-FR-GL-011` / FR-GL-011 | Story 9 — Open operational reopen gates. |
| `DLV-FR-GL-012` / FR-GL-012 | Story 9 — Close operational reopen gates. |
| `DLV-FR-GL-013` / FR-GL-013 | Story 9 — Begin reclose gate handoff. |
| `DLV-FR-GL-014` / FR-GL-014 | Story 10 — Read authoritative posting-gate status. |
| `DLV-FR-GL-015` / FR-GL-015 | Story 1 — Maintain ledgers. |
| `DLV-FR-GL-016` / FR-GL-016 | Story 1 — Maintain accounting books. |
| `DLV-FR-GL-017` / FR-GL-017 | Story 2 — Maintain charts of accounts. |
| `DLV-FR-GL-018` / FR-GL-018 | Story 2 — Maintain accounts and reporting mappings. |
| `DLV-WF-6.6` / WF-6.6 | Stories 3–5 — Journal entry posting and reversal. |

The delivery plan assigns FR-GL-015–018 to M1, FR-GL-001–005 to M2, and
FR-GL-006–014 to M3, while the epic itself is listed as M2. This document
retains those source milestone assignments and uses the dependency order below;
it does not resolve or reclassify the planning baseline.

## 7. User stories

### User Story 1 — Maintain ledgers and accounting books

**Delivery items:** `DLV-FR-GL-015`, `FR-GL-015`; `DLV-FR-GL-016`, `FR-GL-016`

**Existing API operations:**

- `glMaintainLedgers` — `PUT /api/v1/general-ledger/configuration/maintain-ledgers` — `finance.gl.maintain.ledgers`
- `glMaintainAccountingBooks` — `PUT /api/v1/general-ledger/configuration/maintain-accounting-books` — `finance.gl.maintain.accounting.books`

**Primary screen:** `GL-SCR-04`; worklist entry `GL-WS-01` where applicable.

As a GL Manager, I want to maintain effective-dated ledgers and accounting
books so that journal validation uses an authoritative accounting structure and
historical entries retain the configuration versions under which they were
established.

Acceptance criteria:

- [x] An authorized command can create or maintain a ledger with explicit
  legal-entity ownership, ledger type, functional currency, fiscal calendar,
  lifecycle status, and effective-date range.
- [x] An authorized command can create or maintain an accounting book for an
  existing ledger with accounting basis, book type, posting-policy version,
  lifecycle status, and effective dates.
- [ ] Missing or invalid legal entities, calendars, ledger relationships,
  currencies, effective-date intervals, or scoped identities are rejected
  atomically with typed reasons; PostgreSQL runtime OMD references use the
  owning application boundary and live cross-context qualification remains
  tracked in the verification record.
- [x] Accepted results identify the authoritative aggregate identity, version,
  ownership, currency/calendar or book relationship, effective dates, lifecycle
  state, approval evidence where applicable, and validation outcome.
- [ ] `If-Match`, explicit accounting scope, idempotency, correlation,
  authorization, segregation-of-duties, and material-action audit follow the
  approved command contract. Stale updates return a version conflict; a safe
  replay returns the established result.
- [ ] A policy-version or effective-date change preserves the configuration
  version referenced by established journals and does not rewrite financial
  history.
- [x] `GL-SCR-04` shows current state, effective dates, version, owner,
  approval/validation status, blocked action, blocking reason, and recovery or
  next-action guidance.

Suggested implementation steps:

1. Define the `Ledger` and `AccountingBook` aggregates, value objects, and
   effective-date rules in `internal/gl`.
2. Add GL-owned persistence, scoped uniqueness, version checks, and migration
   coverage without reading OMD tables directly; use the approved OMD
   application/reference boundary for legal entities and fiscal calendars.
3. Wire the existing OpenAPI operations, IAM permissions, common command/result
   and problem contracts, idempotency, audit port, and safe response mapping.
4. Add the configuration views and worklist states to `GL-SCR-04` and
   `GL-WS-01`.
5. Verify acceptance, rejection, replay, stale-version, authorization,
   audit-failure, and dependency-unavailable paths.

Required test evidence:

- [x] Domain tests for field/value validation, relationship rules,
  effective-date boundaries, lifecycle transitions, and expected-version
  conflicts.
- [x] Persistence tests for GL schema ownership, constraints, rollback, and
  historical configuration-version retention.
- [x] API handler tests cover scope/permission enforcement, idempotency,
  `If-Match`, typed problems, correlation, audit failure, and dependency
  failure; live runtime and Workflow/SOD qualification remains tracked in the
  verification record.
- [x] Component tests for configuration forms, validation summaries, and safe
  status updates.
- [ ] Playwright tests for configuration forms, keyboard/focus behavior,
  zoom/reflow, and safe status updates.

Implementation evidence and remaining boundaries are recorded in
[`docs/verification/DLV-FR-GL-015-016-us1-ledger-book-maintenance.md`](../../verification/DLV-FR-GL-015-016-us1-ledger-book-maintenance.md).

### User Story 2 — Maintain charts of accounts and account/reporting mappings

**Delivery items:** `DLV-FR-GL-017`, `FR-GL-017`; `DLV-FR-GL-018`, `FR-GL-018`

**Existing API operations:**

- `glMaintainChartsOfAccounts` — `PUT /api/v1/general-ledger/configuration/maintain-charts-of-accounts` — `finance.gl.maintain.charts.of.accounts`
- `glMaintainAccountsAndReportingMappings` — `PUT /api/v1/general-ledger/configuration/maintain-accounts-and-reporting-mappings` — `finance.gl.maintain.accounts.and.reporting.mappings`

**Primary screen:** `GL-SCR-05`; worklist entry `GL-WS-01` where applicable.

As a GL Manager, I want to maintain versioned charts, accounts, restrictions,
currency policies, and reporting mappings so that only valid account and
segment combinations can reach the journal and historical entries remain
reproducible.

Acceptance criteria:

- [x] An authorized command can create or maintain a versioned chart of
  accounts and account-code policy for an existing ledger with a valid
  effective scope.
- [x] An authorized command can create or maintain accounts with account code,
  name, type, normal balance, status, posting restrictions, currency policy,
  effective dates, and approved reporting mappings.
- [x] The command validates account-code uniqueness in the effective chart,
  account type/normal balance, restrictions, currency policy, mapping
  references, chart version, and effective-date rules before commit.
- [x] COA segment definitions and combinations are referenced through the
  published COA boundary; GL does not read or write the `coa` schema.
- [x] Accepted results identify chart/account identity and version, ledger and
  chart relationship, restrictions, mappings, effective dates, approval
  evidence, and validation outcome.
- [ ] Configuration changes preserve the chart/account version used by
  established journals and never rewrite historical journal facts.
- [x] Scope, authorization, expected-version, idempotency, correlation, audit,
  and safe conflict behavior match the approved platform contract.
- [x] `GL-SCR-05` shows current state, dependent impact, effective dates,
  approval/validation status, blocked actions, and the next permitted action.

Suggested implementation steps:

1. Define `ChartOfAccounts`, `Account`, restriction, currency-policy, and
   reporting-mapping value objects inside `internal/gl`.
2. Add versioned GL persistence and indexes for effective chart/account
   lookups; retain source-version references needed by journal validation.
3. Implement the approved configuration operations with IAM, audit,
   idempotency, optimistic concurrency, and published COA/OMD reference ports.
4. Build `GL-SCR-05` and its worklist projections without creating a second
   account mutation surface in reporting or COA.
5. Test invalid mappings, overlapping effective dates, stale versions,
   duplicate replay, authorization denial, and safe dependency failures.

Required test evidence:

- [x] Domain tests for chart/account lifecycle, uniqueness, effective dating,
  restrictions, currency policy, normal balances, and mapping validation.
- [x] Persistence tests cover version retention, parent constraints, and
  atomic rollback; journal references to the correct historical configuration
  remain open until the JournalEntry aggregate is delivered.
- [x] API, authorization, audit, idempotency, and optimistic-concurrency tests.
- [ ] Component tests for chart/account configuration, validation,
  dependent-impact messaging, accessibility, and safe errors.
- [ ] Playwright tests for chart/account configuration, keyboard/focus,
  zoom/reflow, and safe status updates.

#### User Story 2 implementation evidence — 2026-10-10

The implementation is on branch `feat/gl-us2-chart-account-mappings`.
Evidence and remaining qualification boundaries are recorded in
[docs/verification/DLV-FR-GL-017-018-us2-chart-account-mappings.md](../../verification/DLV-FR-GL-017-018-us2-chart-account-mappings.md).

### User Story 3 — Submit and validate a posting request

**Delivery item:** `DLV-FR-GL-001` / `FR-GL-001`

**Existing API operation:** `glSubmitPostingRequest` — `POST /api/v1/general-ledger/actions/submit-posting-request` — `finance.gl.submit.posting.request`

**Primary screens:** `GL-WS-01`, `GL-SCR-01`, `GL-SCR-02`.

As an Accountant or authorized subledger actor, I want to submit a posting
request and see the authoritative validation result so that only a valid,
balanced, authorized accounting effect can be posted once.

Acceptance criteria:

- [ ] The command accepts the version-2 posting contract: source context and
  aggregate/version, accounting scope, posting date and period, expected
  period/gate versions, posting purpose, transaction currency, conversion
  evidence where applicable, lines, correlation/causation, and idempotency
  identity/fingerprint.
- [ ] GL validates ledger, book, legal entity, functional currency, fiscal
  calendar, fiscal period, gate mode/version, posting purpose, account and
  segment eligibility/effective dates, authorization, and source ownership.
- [ ] GL rejects fewer than two lines, unbalanced transaction or functional
  totals, invalid currency modes or scale, unsupported mixed currencies,
  missing conversion evidence, invalid account restrictions, stale gate or
  period versions, and duplicate accounting ownership before establishing a
  journal.
- [ ] A non-approval request establishes one `Posted` journal and returns its
  journal identity/number, version, ledger position, gate evidence, source
  reference, and audit reference in the established result.
- [ ] An approval-required request establishes `PendingApproval` with no
  ledger effect and exposes the approval reference and next action.
- [ ] Outcomes are distinguishable as `JournalEntryPosted`, `PostingRejected`,
  `PostingPendingApproval`, or `IdempotencyConflict`; rejected attempts do not
  become journal entries.
- [ ] Repeating the same accounting identity and fingerprint returns the
  existing result. Reusing it with changed business content returns a typed
  conflict and creates no second effect.
- [ ] `GL-WS-01`, `GL-SCR-01`, and `GL-SCR-02` identify the affected line or
  field, blocking rule, state, owner, evidence, and correction/recovery path.

Suggested implementation steps:

1. Define the `PostingRequest` input model, journal lifecycle, exact-decimal
   balancing rules, and domain errors in `internal/gl`.
2. Add the journal, journal-line, posting-attempt, idempotency, audit, and
   outbox persistence work needed for one local GL transaction.
3. Implement deterministic locking/version checks for the journal source,
   posting gate, account/configuration references, and idempotency record.
4. Wire the approved operation and common problem/result contract, including
   typed domain rejection versus dependency unavailability.
5. Build the journal workbench/detail/result states and run the WF-6.6 primary,
   exception, recovery, authorization, and duplicate paths.

Required test evidence:

- [ ] Domain tests for all posting invariants, lifecycle outcomes, exact
  currency semantics, source identity, gate/period checks, and duplicate or
  fingerprint-conflict behavior.
- [ ] PostgreSQL integration tests for balanced atomic posting, rollback,
  source uniqueness, idempotency, audit/outbox atomicity, and concurrent gate
  admission.
- [ ] API tests for IAM scope, permissions, common headers, typed 4xx/5xx
  results, and safe error projections.
- [ ] Playwright `e2e/wf-6-6.spec.ts` coverage for primary, exception,
  recovery, authorization, and accessibility paths.

#### User Story 3 implementation evidence — 2026-10-10

The implementation is on branch `feat/gl-us3-submit-posting-request`.
Evidence, acceptance traceability, verification commands, and remaining
production/browser qualification boundaries are recorded in
[`docs/verification/DLV-FR-GL-001-us3-submit-posting-request.md`](../../verification/DLV-FR-GL-001-us3-submit-posting-request.md).

### User Story 4 — Apply a journal approval decision

**Delivery item:** `DLV-FR-GL-002` / `FR-GL-002`

**Existing API operation:** `glApplyJournalApprovalDecision` — `POST /api/v1/general-ledger/actions/apply-journal-approval-decision` — `finance.gl.apply.journal.approval.decision`

**Primary screens:** `WFA-SCR-02`, `GL-SCR-02`, `GL-SCR-03` when gate state blocks admission.

As a GL Manager or authorized domain process, I want to apply an immutable
Workflow decision to a pending journal so that approval can establish a journal
only when the current GL state still permits it.

Acceptance criteria:

- [ ] GL accepts an immutable Workflow decision reference and verifies that it
  applies to the expected journal/request, source version, scope, and approval
  policy context. Workflow remains the owner of the decision.
- [ ] An approval decision revalidates journal version, accounting scope,
  ledger/book/account/chart/segment configuration, period state, gate mode and
  version, currency/balance rules, authorization, and any correction/reopen
  scope at application time.
- [ ] An approved decision establishes the posted journal atomically with its
  approval reference, gate evidence, audit record, and outbox intent.
- [ ] A rejected decision creates no ledger effect and returns the approved
  rejected/unchanged lifecycle result required by the current policy; a stale,
  mismatched, or already-applied decision returns a typed conflict.
- [ ] Repeated application with the same identity and fingerprint returns the
  established result; changed content cannot apply a second decision.
- [ ] The UI shows decision reference, subject/version, current state, applied
  result, revalidation failures, responsible owner, and next action without
  allowing approval policy to be edited in GL.

Suggested implementation steps:

1. Define the approval-decision application port and journal transition rules;
   do not create a local approval policy or duplicate Workflow aggregate.
2. Lock and revalidate the pending journal, gate, expected versions, and
   immutable decision reference in the documented order.
3. Commit the journal transition, decision application evidence, idempotency
   result, audit envelope, and outbox record together.
4. Wire the approved API operation and the approval/result views.
5. Test approval, rejection, invalidated decision, stale journal, gate change,
   duplicate delivery, authorization denial, and dependency failure.

Required test evidence:

- [ ] Domain tests prove no ledger effect before approval, current-state
  revalidation, decision/reference matching, and typed conflict outcomes.
- [ ] Integration tests prove Workflow decision consumption, atomic posting,
  audit/outbox behavior, and recovery after interruption at each commit
  boundary.
- [ ] API and UI tests prove permission/scope/segregation enforcement,
  decision-reference display, safe rejection, and accessible next actions.

### User Story 5 — Reverse a posted journal entry

**Delivery item:** `DLV-FR-GL-003` / `FR-GL-003`

**Existing API operation:** `glReverseJournalEntry` — `POST /api/v1/general-ledger/actions/reverse-journal-entry` — `finance.gl.reverse.journal.entry`

**Primary screens:** `GL-SCR-01`, `GL-SCR-02`, `GL-WS-01`.

As an Accountant or authorized correction owner, I want to reverse a posted
journal with a reason and evidence so that an established financial fact is
corrected without editing or deleting the original entry.

Acceptance criteria:

- [ ] The command requires the original journal reference, correction reason,
  accounting scope, effective/posting date, authorization evidence, expected
  version where applicable, and a new idempotency identity.
- [ ] Only an eligible posted journal can be reversed. The original entry,
  lines, source reference, approval, and audit history remain immutable.
- [ ] GL appends a separate reversal journal with equal-and-opposite lines,
  `ReversalOfJournalEntryId`, current gate/period evidence, and its own source
  and idempotency identity.
- [ ] The command rejects a duplicate reversal or stale/closed/restricted gate
  with a typed result; replay of the same command returns the established
  reversal result.
- [ ] The result and UI show the original, linked reversal, reason/evidence,
  resulting balances/lifecycle, ledger position, owner, and any blocking or
  recovery path.
- [ ] The reversal is attributable, audited, and published through the
  transactional outbox without assigning the original source fact to GL when
  another bounded context owns the correction.

Suggested implementation steps:

1. Define reversal eligibility, line inversion, linkage, and duplicate
   protection in the `JournalEntry` domain.
2. Implement the append-only reversal persistence and atomic original lookup,
   gate validation, audit, idempotency, and outbox behavior.
3. Wire the approved operation and reversal preview/result interaction.
4. Verify ordinary, close/reopen correction, duplicate, stale-version,
   authorization, audit failure, and recovery paths.

Required test evidence:

- [ ] Domain tests for immutable original, equal-and-opposite lines,
  single-reversal rule, correction purpose, and exact money behavior.
- [ ] Persistence/concurrency tests for atomic append, duplicate prevention,
  gate admission, outbox/audit durability, and interruption recovery.
- [ ] API/UI tests for reason/evidence requirements, permissions, linked
  lineage, blocking reasons, keyboard operation, and accessible confirmation.
- [ ] WF-6.6 reversal scenarios pass without presenting the original as
  editable.

### User Story 6 — Enter and exit a soft-close gate

**Delivery items:** `DLV-FR-GL-004` / `FR-GL-004`; `DLV-FR-GL-005` / `FR-GL-005`

**Existing API operations:**

- `glEnterSoftCloseGate` — `POST /api/v1/general-ledger/actions/enter-soft-close-gate` — `finance.gl.enter.soft.close.gate`
- `glExitSoftCloseGate` — `POST /api/v1/general-ledger/actions/exit-soft-close-gate` — `finance.gl.exit.soft.close.gate`

**Primary screen:** `GL-SCR-03`.

As a Controller or Fiscal Period Management process, I want to enter and exit
soft close through the GL-owned gate so that ordinary postings follow the
approved policy and the control epoch has durable admission evidence.

Acceptance criteria:

- [ ] `EnterSoftCloseGate` changes an eligible `Open` gate to
  `SoftClosePolicy`, records restrictions, owner, control epoch, policy
  version, expected gate version, and zero active counters.
- [ ] Only the matching soft-close process and epoch can exit the gate.
  `ExitSoftCloseGate` freezes and returns the epoch-qualified admission
  summary, restores the eligible mode, and clears ownership atomically.
- [ ] Conflicting process identifiers, policy versions, scopes, fingerprints,
  expected versions, or epochs return typed conflicts; identical retries return
  the established result.
- [ ] Gate transitions, counters, frozen summaries, idempotency result, audit
  evidence, and outbox intent commit as one GL-owned decision.
- [ ] The gate monitor shows mode, owner, epoch, version, restrictions,
  counters, frozen summary, blocked ordinary actions, and recovery guidance.

Suggested implementation steps:

1. Implement the `PeriodPostingGate` soft-close transitions and counter/frozen
   summary value objects in `internal/gl`.
2. Add deterministic gate locking, expected-version checks, command
   fingerprint history, audit, idempotency, and outbox persistence.
3. Wire the two action operations and the `GL-SCR-03` monitor to the approved
   FPM handoff port.
4. Test entry, exit, duplicate, concurrent owner, stale epoch/version,
   interrupted handoff, and authorization paths.

Required test evidence:

- [ ] Domain and PostgreSQL tests prove exclusive ownership, epoch checks,
  counter freezing, atomic mode/version transitions, and replay safety.
- [ ] API tests prove action permissions, explicit scope, typed conflicts,
  correlation, audit failure, and dependency-unavailable behavior.
- [ ] UI/accessibility tests prove current state, owner, blocking reason,
  allowed action, and non-disruptive status updates.

### User Story 7 — Acquire, release, and finalize the posting barrier

**Delivery items:** `DLV-FR-GL-006` / `FR-GL-006`; `DLV-FR-GL-007` / `FR-GL-007`; `DLV-FR-GL-008` / `FR-GL-008`

**Existing API operations:**

- `glAcquirePostingBarrier` — `POST /api/v1/general-ledger/actions/acquire-posting-barrier` — `finance.gl.acquire.posting.barrier`
- `glReleasePostingBarrier` — `POST /api/v1/general-ledger/actions/release-posting-barrier` — `finance.gl.release.posting.barrier`
- `glFinalizePostingGate` — `POST /api/v1/general-ledger/actions/finalize-posting-gate` — `finance.gl.finalize.posting.gate`

**Primary screen:** `GL-SCR-03`.

As a Controller or close-run process, I want the GL gate to acquire and
finalize a posting barrier safely so that a hard close has an auditable cutoff,
recovery point, and final ledger watermark.

Acceptance criteria:

- [ ] `AcquirePostingBarrier` verifies the active soft-close epoch, freezes its
  counters, records prior ownership and barrier position, transfers ownership
  to the initial close run, changes the gate to `CloseOnly`, and initializes
  close counters atomically.
- [ ] A stale soft-close epoch, active competing owner, invalid close type,
  scope, or expected gate version is rejected without partial transfer.
- [ ] `ReleasePostingBarrier` is valid only for an initial hard close with zero
  close admissions. It restores the prior mode/owner, increments the next soft
  close epoch, returns the new epoch, and clears close ownership.
- [ ] A reclose barrier cannot be released. Any admitted close posting or
  conflicting owner prevents release and leaves authoritative evidence.
- [ ] `FinalizePostingGate` freezes counters, records the final ledger
  watermark, changes the gate to `HardClosed`, clears active ownership only
  after the summary is authoritative, and retains immutable command results.
- [ ] Interrupted finalization is recoverable through `GetPostingGateStatus`;
  a finalized gate is never converted into an abort or released state.
- [ ] The UI shows barrier position, owner, counters, watermark, current mode,
  finalization state, blocked actions, and recovery next action.

Suggested implementation steps:

1. Implement initial-close barrier transfer, releasability checks, and final
   hard-close transition in the GL gate aggregate.
2. Persist barrier position, prior owner, counters, frozen summaries,
   watermarks, idempotency results, audit evidence, and outbox messages in one
   transaction per command.
3. Define the FPM application-port result contract without moving close-run
   ownership into GL.
4. Add recovery queries and tests for process interruption before and after
   ownership transfer and finalization.

Required test evidence:

- [ ] Domain tests for barrier ownership, zero-admission release, non-releasable
  reclose, final watermark, stale version, and duplicate command behavior.
- [ ] PostgreSQL concurrency/failure tests for atomic transfer, journal
  admission at the cutoff, frozen summaries, outbox/audit durability, and
  recovery by status query.
- [ ] API/UI/accessibility tests for authorization, blocked actions, evidence,
  and safe recovery guidance.

### User Story 8 — Open and close a scoped reopen gate

**Delivery items:** `DLV-FR-GL-009` / `FR-GL-009`; `DLV-FR-GL-010` / `FR-GL-010`

**Existing API operations:**

- `glOpenScopedReopenGate` — `POST /api/v1/general-ledger/actions/open-scoped-reopen-gate` — `finance.gl.open.scoped.reopen.gate`
- `glCloseScopedReopenGate` — `POST /api/v1/general-ledger/actions/close-scoped-reopen-gate` — `finance.gl.close.scoped.reopen.gate`

**Primary screen:** `GL-SCR-03`; supporting correction context in `GL-SCR-01` and `GL-SCR-02`.

As a Controller or approved reopen process, I want to open and close a
scope-limited reopen gate so that only authorized corrections can enter a
hard-closed period and the result determines whether reclose is required.

Acceptance criteria:

- [ ] `OpenScopedReopenGate` changes `HardClosed` to `ScopedReopen`, records
  approved accounting scope, expiry, reopen request owner, expected gate
  version, and zero reopen counters.
- [ ] Reopen correction postings are admitted only when their request,
  account/transaction scope, authorization, epoch/expiry, period, and gate
  version match the active gate.
- [ ] `CloseScopedReopenGate` freezes and returns the reopen summary. With zero
  admissions and approved no-change policy it restores `HardClosed`; with any
  admission it changes to `CloseOnly` and retains the owner and summary for
  reclose handoff.
- [ ] Expiry rejects new postings but does not silently clear the owner or
  discard the summary. Duplicate and changed-fingerprint commands return the
  established result or typed conflict.
- [ ] GL never edits the original posted journal; correction subledgers or an
  authorized manual-journal producer submit linked reversals/replacements.
- [ ] The monitor and journal screens show approved scope, expiry, owner,
  counters, frozen summary, permitted posting purpose, and whether reclose is
  mandatory.

Suggested implementation steps:

1. Implement scoped-reopen gate transitions, scope/expiry checks, and frozen
   summary handling in GL.
2. Connect FPM's approved reopen request/result boundary and preserve Workflow
   decision references without implementing FPM or Workflow aggregates.
3. Add posting-purpose checks to the standard posting command and recovery via
   authoritative gate status.
4. Test no-change closure, admitted correction, expiry, duplicate delivery,
   stale versions, competing owners, and interrupted handoff.

Required test evidence:

- [ ] Domain and persistence tests for hard-closed admission, scoped limits,
  expiry, counters, no-change closure, and mandatory reclose handoff.
- [ ] Cross-context tests prove FPM receives the exact frozen summary and GL
  remains the sole journal/gate authority.
- [ ] API/UI/accessibility tests cover permissions, safe scope display,
  rejection reasons, and correction/recovery next actions.

### User Story 9 — Open and close operational reopen, then begin reclose

**Delivery items:** `DLV-FR-GL-011` / `FR-GL-011`; `DLV-FR-GL-012` / `FR-GL-012`; `DLV-FR-GL-013` / `FR-GL-013`

**Existing API operations:**

- `glOpenOperationalReopenGate` — `POST /api/v1/general-ledger/actions/open-operational-reopen-gate` — `finance.gl.open.operational.reopen.gate`
- `glCloseOperationalReopenGate` — `POST /api/v1/general-ledger/actions/close-operational-reopen-gate` — `finance.gl.close.operational.reopen.gate`
- `glBeginRecloseGate` — `POST /api/v1/general-ledger/actions/begin-reclose-gate` — `finance.gl.begin.reclose.gate`

**Primary screen:** `GL-SCR-03`; correction details in `GL-SCR-01` and
`GL-SCR-02`.

As a Controller or authorized operational-reopen process, I want to govern a
bounded operational reopen and transfer admitted work to a reclose barrier so
that outage recovery cannot broaden posting authority or bypass a new close.

Acceptance criteria:

- [ ] `OpenOperationalReopenGate` changes `HardClosed` to
  `OperationalReopen`, recording permitted classes, actor scope, authority
  epoch, expiry, request owner, expected gate version, and zero counters.
- [ ] Operational-reopen postings require the active request identifier,
  authority epoch, permitted class/actor, `OperationalReopen` purpose, valid
  scope, and unexpired authorization at the GL admission boundary.
- [ ] `CloseOperationalReopenGate` freezes and returns counters. A permitted
  no-change result restores `HardClosed`; any admission changes the gate to
  `CloseOnly` and retains the owner and summary.
- [ ] Expiry stops new operational-reopen admissions while retaining ownership
  until an authoritative close result. Conflicting actor scope, epoch, expiry,
  process, or fingerprint is a typed conflict.
- [ ] `BeginRecloseGate` requires a positive retained reopen summary, transfers
  ownership from the reopen request to the reclose run, records the reclose
  barrier position, initializes reclose counters, and returns the prior frozen
  summary. The reclose barrier cannot be released.
- [ ] Recovery after interruption uses the authoritative gate status and
  command fingerprint; a later frozen summary cannot be mistaken for the
  earlier command result.
- [ ] The UI distinguishes operational reopen authority from ordinary or
  scoped reopen authority and shows the mandatory reclose path.

Suggested implementation steps:

1. Implement operational-reopen scope/class/actor/epoch validation and the
   close/reclose gate transitions in GL.
2. Define the FPM operational-reopen and reclose handoff ports using the
   approved process identifiers and frozen-summary contract.
3. Add status-based recovery for open, close, and begin-reclose commands.
4. Test allowed and denied classes/actors, expiry, no-change closure, positive
   admission, ownership transfer, duplicate delivery, and crash recovery.

Required test evidence:

- [ ] Domain tests for operational authority, epoch, class, actor, expiry,
  frozen counters, reclose ownership, and non-releasable barrier rules.
- [ ] Integration/failure tests for FPM handoff, outbox/inbox replay,
  interruption before and after transfer, and exact summary matching.
- [ ] API/UI/accessibility tests for least privilege, safe evidence display,
  blocking reasons, and recovery instructions.

### User Story 10 — Read authoritative posting-gate status

**Delivery item:** `DLV-FR-GL-014` / `FR-GL-014`

**Existing API operation:** `glGetPostingGateStatus` — `GET /api/v1/general-ledger/reference/get-posting-gate-status` — `finance.gl.get.posting.gate.status`

**Primary screen:** `GL-SCR-03`; used by close/reopen recovery and operational support.

As a Controller, close-process owner, or authorized support user, I want to read
the authoritative posting-gate status so that I can determine the current
owner, allowed action, admission evidence, and safe recovery step without
creating a new business fact.

Acceptance criteria:

- [ ] The read is restricted by explicit accounting scope and permission and
  returns no state-changing effect, idempotency reservation, or domain event.
- [ ] The result includes authoritative mode, gate version, active/prior
  owners, barrier position, final ledger watermark, authorization scope,
  authority epoch, expiry, active admission counters, last frozen summary,
  command fingerprint/type, and relevant evidence references.
- [ ] The result distinguishes ordinary, close, scoped-reopen, operational-
  reopen, hard-closed, and recovery states without inferring a successful
  transition from a caller's local process state.
- [ ] Missing gate, invalid scope, unauthorized access, dependency
  unavailability, and stale recovery context are typed and safe to display.
- [ ] `GL-SCR-03` shows allowed/blocked actions, owner, blocking reason,
  evidence, and the next permitted recovery action. It never becomes a second
  mutation model for FPM or Workflow.

Suggested implementation steps:

1. Define the authoritative status projection from `PeriodPostingGate` and its
   immutable frozen-summary history.
2. Implement the approved read operation with scope authorization, safe
   projection, correlation, and bounded observability.
3. Use the reference operation in close/reopen recovery and UI refresh without
   a polling loop that establishes business state.
4. Test every gate mode, ownership boundary, missing/dependency failure, and
   permission case.

Required test evidence:

- [ ] Domain/query tests prove status is derived from authoritative gate state
  and does not mutate it.
- [ ] API tests prove scope filtering, permission denial, safe problem details,
  and correct status fields for every mode.
- [ ] UI/accessibility tests prove readable status, focused updates, keyboard
  access, and clear recovery actions.

## 8. Cross-cutting delivery contract

### 8.1 API and command boundary

- Preserve the approved OpenAPI operation IDs, paths, HTTP methods, permission
  identifiers, common `CommandRequest`/established-result/problem contracts,
  `X-Correlation-Id`, `X-Accounting-Scope-Id`, and required `Idempotency-Key`
  behavior for state-changing operations.
- Configuration updates use the approved `If-Match` header and expected-version
  semantics. Gate and journal commands carry their operation-specific expected
  versions and process identifiers in the approved command data.
- `GetPostingGateStatus` remains a read-only `GET`; it does not acquire an
  idempotency reservation or publish a business event.
- Domain rejection, authentication failure, authorization denial, version
  conflict, idempotency conflict, dependency unavailability, and internal
  failure remain distinguishable typed outcomes.

### 8.2 Persistence and transaction boundary

- GL owns `gl.ledger`, `gl.accounting_book`, `gl.chart_of_accounts`,
  `gl.account`, `gl.journal_entry`, `gl.journal_entry_line`, and
  `gl.period_posting_gate` plus any approved append-only attempt, correction,
  summary, audit-reference, and idempotency records.
- PostgreSQL stores money as exact `NUMERIC`; Go domain code uses the approved
  exact-decimal abstraction. Binary floating point is not permitted for
  financial amounts or balancing.
- A state change, its idempotency result, audit envelope, and outbox intent
  commit in the owning GL transaction. Aggregate updates use expected-version
  predicates; zero affected rows become typed version conflicts.
- Journal posting and gate admission lock/version the required GL records in a
  deterministic order and never partially establish a journal or counter.
- Cross-context effects use application ports, durable process checkpoints,
  transactional outbox, and inbox/deduplication rules. No cross-schema write is
  introduced.

### 8.3 Integration outcomes

The GL event catalog remains the approved source of event names and payload
semantics. Applicable GL outcomes include:

- `JournalEntryPosted`, `PostingRejected`, `PostingPendingApproval`,
  `IdempotencyConflict`, and `JournalEntryReversed`.
- `PostingAdmissionRecorded`, `SoftCloseGateEntered`, `SoftCloseGateExited`,
  `PostingBarrierAcquired`, `PostingBarrierReleased`, and
  `PostingGateFinalized`.
- `ScopedReopenGateOpened`, `ScopedReopenGateClosed`,
  `OperationalReopenGateOpened`, `OperationalReopenGateClosed`,
  `OperationalReopenGateExpired`, and `RecloseGateBegun`.

Each published outcome carries the approved immutable business identity,
aggregate version, accounting scope, correlation, causation, and safe data
classification. State and outbox intent commit together. Consumers use inbox
deduplication, version/order checks, and explicit pending or exception states;
they do not infer a successful accounting effect from an event that has not
been durably established.

### 8.4 Authorization, audit, privacy, and observability

- IAM is the authorization decision point for permission, accounting scope,
  segregation, actor, and emergency-access conditions. GL fails closed when a
  required decision cannot be established.
- Material configuration, posting, approval application, reversal, gate, and
  recovery actions record actor, authentication, scope, source, action,
  authorization, correlation/causation, expected version, before/after state
  references, and reason/evidence through the approved audit port.
- Manual recovery, override, reconciliation, or repair requires authorization,
  a reason, before/after evidence, and independent review for high-risk
  records. It never bypasses GL correction or gate semantics.
- Logs, traces, metrics, UI notifications, and exported evidence use safe
  identifiers, operation names, result/error classes, retryability, and data
  classification. They do not expose unrestricted journal lines, credentials,
  tokens, or sensitive source payloads.
- At minimum, operational telemetry covers command outcome, idempotency
  conflict, version conflict, gate owner/mode, posting rejection reason class,
  outbox lag, recovery age, and pending/exception worklist age.

### 8.5 UI and workflow contract

- `GL-WS-01` groups work by actionable state and exposes scope, owner, amount or
  currency where applicable, approval, exception, age, posting status, and next
  action.
- `GL-SCR-01` shows source, scope, lines, balancing, currency, segments,
  approval, gate evidence, lifecycle, and immutable correction lineage.
- `GL-SCR-02` shows established results, rejection, pending approval,
  idempotency conflict, version conflict, dependency failure, and safe next
  action.
- `GL-SCR-03` shows gate mode, owner, version, barrier position, counters,
  frozen summaries, expiry, authority scope/epoch, and permitted posting
  purposes.
- `GL-SCR-04` and `GL-SCR-05` show effective-dated configuration versions,
  validation, approval, dependent impact, blocked action, and recovery path.
- Critical workflows meet the approved keyboard, focus, validation-summary,
  zoom/reflow, contrast, screen-reader, and non-disruptive status-update
  requirements. Aggregated views do not become a second mutation surface for
  another capability's record.

## 9. Dependencies and recommended sequence

### 9.1 Dependencies

- EP-PLAT-001 provides the command/result, exact-money, database, idempotency,
  optimistic-concurrency, outbox, and architecture foundations.
- EP-IAM-001 provides authentication, permissions, accounting-scope
  authorization, segregation-of-duties, and emergency-access evaluation.
- EP-OMD-001 provides legal-entity and fiscal-calendar references.
- EP-COA-001 provides segment definitions, values, combinations, and approved
  change references.
- EP-UX-001 and EP-OPS-001 provide shared accessible surfaces and safe
  telemetry/operational conventions.
- WFA contracts are required for approval-bearing journal requests and
  decisions; WFA remains the owner of the decision.
- FPM contracts are required for soft close, hard close, reopen, reclose, and
  status handoffs; FPM remains the owner of period processes.
- Later subledger capabilities depend on the standard GL posting contract and
  use GL's authoritative journal result; GL must not duplicate their business
  facts.

### 9.2 Recommended implementation order

1. Freeze the GL command, event, permission, persistence, and cross-context
   port contracts against the approved sources.
2. Deliver Story 1 and Story 2 so ledger/book/chart/account configuration and
   historical versions exist before journal validation.
3. Deliver Story 3 for non-approval posting and Story 4 for Workflow decision
   application.
4. Deliver Story 5 for immutable reversal and WF-6.6 end-to-end evidence.
5. Deliver Story 10 early enough to make gate recovery authoritative.
6. Deliver Story 6, then Story 7 for soft close and initial hard-close
   barriers.
7. Deliver Story 8 and Story 9 only with approved FPM/WFA handoff contracts,
   then qualify reopen/reclose recovery and ownership transfer.
8. Run the complete GL vertical verification matrix and update this artifact
   with evidence; do not mark roadmap items complete from planning alone.

## 10. Risks and unresolved source decisions

- The delivery plan's milestone assignments span M1, M2, and M3 under the
  single EP-GL-001 epic. This story document records the difference but does
  not invent a new milestone decision.
- WFA and FPM are later epics than the GL operations that consume their
  decisions or process ownership. Their application-port and event contracts
  must be approved before implementation; GL must not create substitute
  approval or period aggregates.
- The technical persistence baseline gives the essential journal and gate
  constraints, while the DDD posting contract carries additional scope,
  purpose, currency, approval, authority, and summary fields. The migration
  design must reconcile those requirements before implementation without
  weakening the DDD invariants.
- Configuration items are assigned M1 priority in the delivery plan even
  though the epic is listed as M2. Their implementation may be sequenced as an
  earlier prerequisite, but the roadmap source remains authoritative until
  explicitly changed.
- Local test evidence cannot establish live Entra behavior, production audit
  integrity, NFR capacity, zero-loss recovery, or release readiness. Those
  remain qualification work and must be reported separately.

## 11. Required epic verification evidence

- [ ] Domain tests cover all 18 FR-GL operations, journal lifecycle,
  configuration effective dating, exact money, gate ownership, reopen/reclose,
  correction lineage, idempotency, and expected-version conflicts.
- [ ] PostgreSQL integration tests cover migrations, schema ownership,
  constraints, deterministic locking, rollback, source uniqueness, gate
  counters/frozen summaries, audit/outbox atomicity, and recovery.
- [ ] API contract, authorization, scope, segregation, typed error,
  idempotency, correlation, and safe-data tests pass for every mapped route.
- [ ] Component and Playwright tests cover `GL-WS-01`, `GL-SCR-01` through
  `GL-SCR-05`, supporting approval interaction, keyboard operation,
  accessibility, validation, conflict, dependency failure, and recovery UI.
- [ ] `e2e/wf-6-6.spec.ts` passes primary, exception, recovery, and
  authorization paths for posting and reversal.
- [ ] Gate close/reopen workflows prove one owner, stale-version rejection,
  no-change closure, mandatory reclose after admitted reopen posting, and
  status-based recovery after interruption.
- [ ] Integration tests prove transactional outbox, inbox deduplication,
  duplicate delivery safety, event ordering/gap handling, and single
  accounting ownership across a representative subledger.
- [ ] Performance, capacity, availability, recovery, audit, security,
  accessibility, and operational results are measured against the NFR baseline;
  no threshold is inferred from unit or local UI tests.

## 12. Definition of done

- [ ] Every delivery item in the traceability table has an implemented,
  source-backed vertical slice or an explicitly recorded approved deferral.
- [ ] GL remains the sole owner of its records and no module bypasses its
  application boundary or schema ownership.
- [ ] Established journals and configuration history are immutable in the
  required sense; corrections and gate transitions preserve lineage.
- [ ] Approval, authorization, idempotency, expected-version, audit, outbox,
  UI, accessibility, failure, and recovery evidence is attached to the changed
  story sections.
- [ ] Documentation, OpenAPI, migrations, generated artifacts, tests, and
  operational evidence are synchronized with the delivered behavior.
- [ ] The roadmap is updated only after the repository contains the required
  implementation and successful verification evidence; this planning document
  alone does not close EP-GL-001.

## 13. Source references

- `docs/specs/finance_domain_model_ddd.md` — §§2.2, 3.1, 3.3, 4.1, 5.1,
  5.3, 5.5, 6.1, 6.2, 6.6, 7.12, 14.1, 14.3, 14.8, 14.10, and 14.13.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` — §3.2,
  FR-GL-001 through FR-GL-018.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — §7.2, §8.6,
  `GL-WS-01`, `GL-SCR-01` through `GL-SCR-05`, and WF-6.6.
- `docs/specs/finance_nonfunctional_requirements_v1.0.md` — General Ledger and
  WF-6.6 quality class, performance, capacity, reliability, security, audit,
  recovery, accessibility, integration, maintenance, and testing baselines.
- `docs/specs/finance_delivery_plan_v1.0.md` — epic, global requirements,
  capability backlog, milestone assignments, and exit-evidence rules.
- `ROADMAP.md` — EP-GL-001 and its delivery-item inventory.
- `docs/specs/system_design/01_solution_architecture_overview_v1.0.md` and
  `docs/specs/system_design/02_application_module_design_v1.0.md` — bounded
  context, transaction, module, and GL sequence boundaries.
- `docs/specs/technical_specifications/01_backend_module_specifications_v1.0.md`,
  `03_database_persistence_specifications_v1.0.md`,
  `04_events_workers_integration_specifications_v1.0.md`,
  `09_testing_performance_recovery_specifications_v1.0.md`, and
  `10_technical_traceability_decisions_v1.0.md` — implementation, persistence,
  event, verification, and API/permission baselines.
- `contracts/openapi/paths/general-ledger.yaml` — approved route methods,
  operation IDs, permission metadata, common headers, and response contracts.
