# EP-INV-001 — Invoicing User Stories

| Field | Value |
|---|---|
| Epic | `EP-INV-001` |
| Capability | Invoicing |
| Milestone | M4 in the roadmap and delivery plan |
| Status | US1 and US2 local implementation slices are present; cross-context reference integration and release qualification remain open |
| Owning module | `internal/invoicing` |
| Owning schema | `invoicing` |
| Primary users | Billing Operations Specialist; Billing Manager |
| Delivery dependencies | EP-OMD-001, EP-IAM-001, EP-UX-001 |
| Downstream handoff | Accounts Receivable through the approved customer-invoice handoff boundary |
| Source of truth | DDD, PRD, UX, NFR, system design, technical specifications, delivery plan, and `ROADMAP.md` |

This document is a planning and delivery-traceability artifact. It decomposes
EP-INV-001 into six functional user stories. US1 and US2 now have local domain,
API, persistence, and UI implementation slices with focused evidence. The
remaining stories, cross-context reference adapters, full persistence
qualification, and release evidence remain open.

The repository contains the approved Invoicing OpenAPI paths and generated
client/server artifacts. US1 established the initial template slice; US2 adds
the schedule aggregate, owner persistence boundary, typed command/read
contract, handlers, and capability UI while retaining the approved modular
ownership boundaries.

## 1. Outcome

Billing users can maintain versioned invoice templates and customer billing
schedules, generate deterministic commercial invoice versions from approved
billing inputs, review validation and calculation evidence, recalculate or
cancel only unfinalized invoices, and finalize eligible invoices for an
idempotent handoff to Accounts Receivable.

The completed epic must make the following visible and explainable:

- the accounting scope, customer and contract references, and responsible
  billing user;
- the template, schedule, source, tax, payment, and calculation versions used;
- the generated invoice identity, version, lines, amounts, currencies, and
  validation outcome;
- whether an invoice is still unfinalized, finalized, cancelled, pending an
  external outcome, or blocked by validation or dependency failure;
- the immutable finalization version and AR handoff outcome;
- the source-version differences and lineage for every permitted recalculation;
- the reason and evidence for every permitted cancellation; and
- the support, authorization, correlation, and audit evidence needed to
  diagnose a material outcome without exposing sensitive data.

## 2. Learning objective

Implementers should learn how a modular-monolith bounded context owns
commercial billing documents while coordinating with master data, tax,
revenue, and Accounts Receivable without moving ownership across modules:

- model `InvoiceTemplate`, `BillingSchedule`, and `GeneratedInvoice` as
  independently owned aggregates;
- apply effective-dated configuration and retain the exact source versions
  used by historical invoice versions;
- calculate money with exact decimal and currency-policy semantics;
- make generation, finalization, recalculation, cancellation, and cross-context
  handoff safe under retry, duplicate delivery, stale versions, and concurrent
  changes;
- preserve the boundary that Invoicing creates the commercial document while
  AR owns the customer invoice, receivable, and billing posting; and
- deliver domain, persistence, API, authorization, UI, audit, observability,
  integration, and recovery evidence as one vertical capability slice.

## 3. Scope

This epic includes:

- Invoicing domain aggregates, value objects, commands, validation policies,
  lifecycle transitions, and immutable version lineage;
- configuration of invoice templates, including template line items, billing
  frequency, charge rules, tax categories, and payment-method settings;
- configuration and maintenance of customer billing schedules, scheduled
  charges, accounting scope, customer and contract references, next billing
  date, and schedule status;
- generation of `GeneratedInvoice` records from approved billing inputs with
  source evidence, calculation results, validation exceptions, and stable
  generation identity;
- finalization of eligible generated invoices and the approved AR handoff
  boundary;
- recalculation of unfinalized invoices using an authoritative new source
  version;
- cancellation of unfinalized invoices with an immutable reason and lineage;
- the approved `/api/v1/invoicing` command and query surfaces, common command,
  established-result, problem, correlation, idempotency, and version contracts;
- the `invoicing` schema, owner-only repositories, aggregate versions,
  idempotency records, audit references, and transactional outbox/inbox
  coordination required by the approved platform contracts;
- the billing operations worklist, template editor, schedule editor,
  generation run, generated-invoice preview, and finalization/AR-handoff
  surfaces defined by the UX specification;
- authorization, accounting-scope filtering, audit evidence, safe telemetry,
  support references, and accessible success, pending, blocked, conflict,
  unavailable, and exception states; and
- focused domain, persistence, API, integration, frontend, accessibility,
  workflow, concurrency, idempotency, and recovery evidence.

## 4. Explicit exclusions

These stories do not authorize:

- ownership or implementation of the AR `CustomerInvoice`,
  `ReceivableOpenItem`, receipt, credit, refund, chargeback, write-off, or
  customer-adjustment aggregates;
- any customer-invoice, receivable, or billing journal posting in Invoicing;
  DDD §5.2 makes AR the sole owner of those accounting effects;
- direct reads or writes of the AR, GL, OMD, Tax Filing, Revenue Recognition,
  Payments, or any other module's adapter, repository, or database schema;
- vendor invoice, procurement matching, AP liability, payment, receipt,
  collection, or bank-settlement behavior;
- Tax Filing configuration or tax determination ownership; Invoicing consumes
  an authoritative tax result or remains blocked when the required result is
  missing or unresolved;
- Revenue Recognition contract assessment, schedule approval, or accounting
  profile publication; Invoicing may retain approved source references needed
  by the published contract;
- GL journal submission, posting-gate ownership, or ledger-entry correction;
  Invoicing must not call `SubmitPostingRequest` for customer billing;
- editing, deleting, or overwriting a finalized invoice or any other
  established financial fact; corrections belong to the approved AR or other
  owning-context correction flow;
- an unapproved scheduler, provider, tax engine, event name, topic, route,
  request field, response field, permission, state, or persistence table;
- treating an accepted local finalization as successful AR issuance when the
  downstream handoff is pending, rejected, unavailable, or ambiguous; or
- production, performance, disaster-recovery, or release-qualification claims
  without the corresponding evidence.

## 5. Ownership and domain rules

### 5.1 Owning component

| Area | Invoicing responsibility | Other owner |
|---|---|---|
| Invoice template | Template identity, line items, billing frequency, charge rules, tax categories, payment-method settings, applicability, and version lineage | OMD and other reference-data owners provide authoritative references |
| Billing schedule | Customer/contract schedule, scheduled charges, accounting scope, next billing date, status, and source-version references | OMD owns customer identity; source-contract owner owns contract facts |
| Generated invoice | Generated invoice identity/version, lines, money, calculation/source evidence, validation, unfinalized lifecycle, finalization, cancellation, and handoff intent | AR owns customer-invoice issuance and receivable accounting |
| Tax | Tax inputs and finalization readiness reference | Tax Filing owns authoritative tax determination and tax rules |
| Revenue classification | Source reference needed by downstream invoice accounting where applicable | Revenue Recognition publishes the versioned accounting profile; AR retains the profile used for accounting |
| Journal posting | None for customer billing | AR owns the billing and receivable posting; GL owns journal authority |
| Audit | Emit material-action evidence and retain references | Audit Integrity owns the audit chain and proof/seal processing |

### 5.2 Aggregates and invariants

The three aggregate roots and their source-defined members are:

- `InvoiceTemplate`, with `TemplateLineItem` entities and the value objects
  `TemplateId`, `BillingFrequency`, `ChargeRule`, `TaxCategory`, and
  `PaymentMethod`;
- `BillingSchedule`, with `ScheduledCharge` entities and the value objects
  `BillingScheduleId`, `AccountingScope`, `CustomerId`, `ContractReference`,
  `NextBillingDate`, and `ScheduleStatus`; and
- `GeneratedInvoice`, with `GeneratedInvoiceLine` entities and the value
  objects `GeneratedInvoiceId`, `AccountingScope`, `CustomerId`,
  `InvoiceVersion`, `Money`, `GenerationSource`, and `IdempotencyKey`.

Every story must preserve these invariants:

1. Every ledger-bound command carries explicit accounting scope. Scope is
   never inferred solely from tenant, browser route, or ambient session state.
2. Template and schedule configuration is versioned and effective-dated where
   applicable. A generated invoice retains the exact configuration/source
   versions used for its calculation.
3. Customer, contract, tax, payment, and other cross-context references are
   validated through approved ports or published contracts. Invoicing does not
   become the source of truth for another context's record.
4. Monetary calculations use exact decimal semantics, currency-specific scale,
   configured rounding rules, and explicit transaction/functional/presentation
   currency meaning where more than one currency is possible.
5. Generation is idempotent by business scope, identity, fingerprint, and
   source version. The same identity and fingerprint return the established
   result; changed content returns an idempotency conflict with no second
   invoice effect.
6. Concurrent configuration, generation, recalculation, finalization, or
   cancellation changes use expected versions and establish one valid outcome
   or a typed conflict; no command silently overwrites another outcome.
7. A generated invoice cannot be finalized until its required billing inputs,
   calculations, customer/tax references, and finalization-readiness controls
   are authoritative and complete.
8. Finalization establishes one immutable invoice version and one correlated
   AR handoff intent. A retry or duplicate handoff cannot create a second
   customer invoice or billing effect.
9. Recalculation is allowed only for an unfinalized invoice and must use an
   authoritative new source version. It never changes a finalized version or
   erases the prior source/calculation lineage.
10. Cancellation is allowed only for an unfinalized invoice. It retains the
    cancelled version, reason, actor, scope, and audit lineage and cannot alter
    a finalized downstream invoice.
11. Invoicing creates commercial billing documents but never submits the
    customer billing journal. AR is the sole producer of the customer invoice,
    receivable, and billing accounting effect.
12. Material configuration and lifecycle transitions retain authorization,
    correlation, causation, source version, expected version, idempotency,
    and audit evidence.

### 5.3 Cross-context boundaries

- **Organization & Master Data:** Invoicing consumes published customer,
  legal-entity, calendar, and other applicable reference identities. It does
  not mutate OMD records or read OMD tables directly.
- **Accounts Receivable:** A finalized generated invoice is handed to AR
  through the approved `InvoiceFinalized` / `IssueCustomerInvoice` boundary.
  AR validates and owns the customer invoice, receivable open item, and billing
  posting. Invoicing tracks the downstream outcome without becoming a second
  authoritative AR record.
- **Tax Filing:** Finalization is blocked when required tax determination is
  missing, expired, unsupported, or inconsistent. Invoicing does not invent a
  tax result or silently choose a default tax rule.
- **Revenue Recognition:** Where invoice accounting requires revenue
  classification, downstream AR must retain the exact approved profile and
  version used. Invoicing does not publish or alter the profile.
- **General Ledger:** Invoicing supplies no journal entry. Any billing posting
  request is created by AR and validated/admitted by GL.
- **Workflow & Approvals:** The six approved INV functional actions do not
  introduce a new approval command. If a future policy makes an INV action
  approval-bearing, it must use the existing Workflow contract and an approved
  requirement change.

### 5.4 State vocabulary

The plan uses only the states named or implied by the approved PRD and DDD
scope: configured or versioned template/schedule records; generated,
unfinalized, finalized, and cancelled invoice outcomes; validation,
pending/dependency, conflict, and exception outcomes; and AR handoff outcomes.
Implementation must not add a new public lifecycle state merely to simplify a
handler or UI.

## 6. Traceability

### 6.1 Story coverage of the delivery plan

| Story | Direct delivery item | Functional requirement | Approved operation | Primary aggregate |
|---|---|---|---|---|
| US1 | `DLV-FR-INV-001` | `FR-INV-001` | `invConfigureInvoiceTemplates` | `InvoiceTemplate` |
| US2 | `DLV-FR-INV-002` | `FR-INV-002` | `invConfigureBillingSchedules` | `BillingSchedule` |
| US3 | `DLV-FR-INV-003` | `FR-INV-003` | `invGenerateInvoices` | `GeneratedInvoice` |
| US4 | `DLV-FR-INV-004` | `FR-INV-004` | `invFinalizeGeneratedInvoices` | `GeneratedInvoice` and AR handoff intent |
| US5 | `DLV-FR-INV-005` | `FR-INV-005` | `invRecalculateUnfinalizedInvoices` | `GeneratedInvoice` |
| US6 | `DLV-FR-INV-006` | `FR-INV-006` | `invCancelUnfinalizedInvoices` | `GeneratedInvoice` |

### 6.2 UX traceability

| Story | UX surfaces | Required interaction evidence |
|---|---|---|
| US1 | `INV-WS-01`, `INV-SCR-01` | Template identity/version, fields, applicability, validation, authorization, conflict, and recovery states |
| US2 | `INV-WS-01`, `INV-SCR-02` | Scope, customer/contract reference, charges, next billing date, schedule status, validation, and conflict states |
| US3 | `INV-WS-01`, `INV-SCR-03`, `INV-SCR-04` | Source inputs, generated records, warnings, failures, rerun eligibility, lines, money, taxes, and source version |
| US4 | `INV-WS-01`, `INV-SCR-04`, `INV-SCR-05` | Finalization blockers, immutable version, AR handoff status, pending/rejected outcome, and safe retry/reconciliation |
| US5 | `INV-WS-01`, `INV-SCR-03`, `INV-SCR-04` | New source version, changed calculations, prior lineage, validation result, and no-finalized-fact overwrite |
| US6 | `INV-WS-01`, `INV-SCR-04` | Reasoned confirmation, eligibility, cancellation result, immutable lineage, and finalized-state rejection |

### 6.3 Cross-cutting requirements and quality gates

The epic contributes evidence to these inherited requirements; it does not
close the global requirement families by itself:

- `GFR-001` through `GFR-022`, especially ownership and vocabulary (`GFR-001`),
  access and segregation (`GFR-002`–`GFR-004`), immutable facts (`GFR-005`),
  idempotency and conflict behavior (`GFR-006`–`GFR-008`), visible state and
  recovery (`GFR-009`), currency evidence (`GFR-010`), single accounting
  ownership (`GFR-011`), cross-context state and lineage (`GFR-012`–`GFR-013`),
  audit (`GFR-014`), privacy (`GFR-015`), search/filter behavior (`GFR-016`),
  dependency distinction (`GFR-017`), legal hold (`GFR-018`), effective-dated
  configuration (`GFR-019`), evidence export (`GFR-020`), and traceability
  controls (`GFR-021`–`GFR-022`);
- Invoicing's NFR capability baseline: primary class B, critical class B, with
  principal categories `PERF`, `CAP`, `AVL`, `REL`, `SEC`, `PRV`, `AUD`,
  `OBS`, `ACC`, `LOC`, `INT`, `MNT`, and `TST`;
- focused NFR evidence for durability, duplicate prevention, conflicts,
  exact decimal money, immutable lineage, cross-context ownership, audit,
  default-deny access, privacy minimization, operational state, accessibility,
  localization, integration compatibility, and recovery; and
- M4 minimum quality gates: `QG-01`, `QG-02`, `QG-03`, `QG-04`, `QG-05`,
  `QG-06`, `QG-07`, `QG-08`, and `QG-10`.

### 6.4 Approved API surface

All six operations use the approved base URL `/api/v1`, common
`CommandRequest`, common established/problem responses, server-derived actor
and authorization evidence, and the operation-specific headers in the
OpenAPI contract. Configuration operations use `PUT`; action operations use
`POST`.

| Requirement | Method and route | Operation ID | Permission |
|---|---|---|---|
| `FR-INV-001` | `PUT /api/v1/invoicing/configuration/configure-invoice-templates` | `invConfigureInvoiceTemplates` | `finance.inv.configure.invoice.templates` |
| `FR-INV-002` | `PUT /api/v1/invoicing/configuration/configure-billing-schedules` | `invConfigureBillingSchedules` | `finance.inv.configure.billing.schedules` |
| `FR-INV-003` | `POST /api/v1/invoicing/actions/generate-invoices` | `invGenerateInvoices` | `finance.inv.generate.invoices` |
| `FR-INV-004` | `POST /api/v1/invoicing/actions/finalize-generated-invoices` | `invFinalizeGeneratedInvoices` | `finance.inv.finalize.generated.invoices` |
| `FR-INV-005` | `POST /api/v1/invoicing/actions/recalculate-unfinalized-invoices` | `invRecalculateUnfinalizedInvoices` | `finance.inv.recalculate.unfinalized.invoices` |
| `FR-INV-006` | `POST /api/v1/invoicing/actions/cancel-unfinalized-invoices` | `invCancelUnfinalizedInvoices` | `finance.inv.cancel.unfinalized.invoices` |

No route, operation ID, permission, request field, response field, or error
code may be added to this plan unless an authoritative specification is
updated first.

### 6.5 Integration boundary and source gap

The DDD context map names `InvoiceFinalized` and `IssueCustomerInvoice` as the
Invoicing-to-AR published relationship. The technical command inventory
currently defines AR's `issue-customer-invoice.v1`, but does not define an
Invoicing-owned command topic or a complete `InvoiceFinalized` event payload.
US4 therefore carries an explicit readiness gate: implementation must use the
approved contract once confirmed, and must not invent an event name, topic,
payload, or delivery outcome to fill the gap.

Cross-context messages must preserve stable business identity, source identity,
scope, aggregate/source version, amount and currency where applicable,
occurred time, correlation, causation, authoritative owner, contract version,
and safe data classification. The local aggregate and outbox record commit in
the Invoicing module; AR establishes its own authoritative outcome.

## 7. User stories

### 7.1 User Story 1 — Configure versioned invoice templates

As a Billing Operations Specialist or Billing Manager, I want to configure an
invoice template so that generated invoices use approved line-item, charge,
tax, frequency, and payment settings.

Direct traceability: `DLV-FR-INV-001`, `FR-INV-001`,
`invConfigureInvoiceTemplates`, `INV-WS-01`, and `INV-SCR-01`.

Acceptance criteria:

- [ ] An authorized actor can maintain an `InvoiceTemplate` within an explicit
  accounting scope using the approved operation, permission, command contract,
  and required idempotency metadata.
- [ ] The template accepts only the DDD/PRD-defined billing frequency, template
  line items, charge rules, tax categories, and payment-method settings.
- [ ] Required fields, unsupported combinations, effective-date/applicability
  conflicts, invalid reference values, and incomplete calculation inputs return
  typed validation outcomes without establishing a partial template.
- [ ] A successful change returns the template identity, established version,
  applicable scope, and safe result links; a rejected change exposes a
  field-level or aggregate-level reason suitable for correction.
- [ ] A repeated command with the same identity and fingerprint returns the
  established result. Reusing the identity with changed content returns an
  idempotency conflict and changes no template.
- [ ] A stale expected version or concurrent update returns a typed conflict;
  the handler never silently overwrites another template version.
- [ ] A template version used by an invoice remains retrievable and is not
  changed by later template maintenance.
- [ ] Authorization uses default-deny, applicable accounting-scope dimensions,
  and any configured segregation rule; the client cannot grant itself access.
- [ ] The editor and worklist show identity, version, applicability, settings,
  validation, owner, scope, and next action without relying on color alone.
- [ ] Material configuration changes create audit evidence with actor,
  authentication subject, scope, action, correlation, authorization, and
  before/after or event fingerprints.

Suggested implementation steps:

1. Confirm the approved template command data and reference-validation
   boundaries; do not add fields not defined by the source contracts.
2. Implement `InvoiceTemplate` validation, versioning, and idempotent command
   handling inside `internal/invoicing`.
3. Add the owner migration, repository, aggregate-version update, idempotency,
   audit, and transactional-outbox records in the `invoicing` schema.
4. Wire the existing OpenAPI operation and permission through the module
   handler without changing generated contract artifacts unnecessarily.
5. Build `INV-SCR-01` and its worklist states using shared UX components,
   explicit scope, accessible validation, conflict, and unavailable outcomes.
6. Add focused domain, persistence, API, authorization, UI, accessibility,
   audit, and idempotency evidence.

Required test evidence:

- [ ] Template invariant, field-validation, effective-date, scope, and exact
  version-lineage unit tests.
- [ ] Persistence tests for owner-only schema access, aggregate version,
  idempotency, audit evidence, and rollback/no-partial-commit behavior.
- [ ] API tests for success, authentication, authorization/scope denial,
  validation, stale version, changed duplicate, unavailable dependency, and
  common problem/result contracts.
- [ ] Component and route tests for the editor, worklist filters, keyboard
  operation, accessible errors, status meaning, and sensitive-value masking.

### 7.2 User Story 2 — Configure customer billing schedules

As a Billing Operations Specialist, I want to configure a customer billing
schedule so that the system knows which approved charges are eligible for
invoice generation and when they should be evaluated.

Direct traceability: `DLV-FR-INV-002`, `FR-INV-002`,
`invConfigureBillingSchedules`, `INV-WS-01`, and `INV-SCR-02`.

Acceptance criteria:

- [x] An authorized actor can maintain a `BillingSchedule` with explicit
  accounting scope, customer reference, contract reference, scheduled charges,
  next billing date, and schedule status as defined by the approved contract.
- [ ] Customer, contract, template/billing, currency, calendar, and effective
  date inputs are validated through approved reference boundaries; Invoicing
  does not read another module's tables directly. The current Postgres runtime
  intentionally returns dependency-unavailable until the owning reference
  ports are wired.
- [x] An invalid schedule, unsupported date/time-zone combination, missing
  required billing input, or ineligible customer/contract reference is rejected
  without creating a partially configured schedule.
- [x] The result exposes schedule identity, scope, customer and contract
  references, next billing date, status, version, and validation outcome to an
  authorized user.
- [x] Repeating the same command identity and fingerprint returns the existing
  result; changed content under the same identity returns an idempotency
  conflict with no second schedule effect. Postgres runtime retries use the
  durable platform idempotency coordinator; memory tests cover the same
  command semantics.
- [x] Stale expected versions and concurrent schedule changes produce typed
  conflicts and preserve the winning version and its audit lineage.
- [x] A schedule version used for generation is retained as source evidence and
  later schedule maintenance does not rewrite the generated invoice's history.
- [x] Configuration actions enforce default-deny authorization, accounting
  scope, applicable amount/currency/period dimensions, and configured
  segregation rules.
- [ ] `INV-SCR-02` shows schedule identity, customer/contract reference,
  charges, next billing date, status, validation, blocked reason, and next
  action with keyboard-complete form behavior. The local editor/worklist covers
  the core identity, charge, date, status, validation, scope, and conflict
  states; richer blocked/filter/keyboard qualification remains open.
- [ ] Schedule changes are audited and emit safe operational telemetry with a
  stable support reference and no unmasked sensitive values. Audit persistence
  is wired; end-to-end telemetry/release evidence remains open.

Suggested implementation steps:

1. Confirm the approved schedule input/reference contract and date/time-zone
   rules before choosing persistence columns or UI fields. **Completed for the
   typed local slice; owning reference ports remain open.**
2. Implement `BillingSchedule` and `ScheduledCharge` validation, versioning,
   and idempotent configuration handling in `internal/invoicing`. **Completed
   locally, including durable Postgres idempotency wiring.**
3. Add schema, repository, owner-role, audit, idempotency, and outbox support.
   **Schedule schema/repository/audit/idempotency are present; no schedule
   event/outbox contract was invented because the approved event is not yet
   defined.**
4. Wire `invConfigureBillingSchedules` and its exact permission. **Completed,
   including authoritative list/detail reads.**
5. Build `INV-SCR-02` and worklist filters for scope, customer, status, date,
   exception, and next action. **Editor/worklist and scope/error/conflict
   states are present; richer server-side filters remain open.**
6. Add domain, persistence, API, authorization, UI, accessibility, and
   concurrency evidence. **Focused domain/API/UI/concurrency evidence is
   present; database integration and release/accessibility qualification remain
   open.**

Required test evidence:

- [x] Schedule identity, scope, customer/contract reference, date, status,
  charge, version, and validation tests.
- [ ] Persistence tests for versioned source history, idempotency, concurrent
  update conflict, audit, and no-partial-commit behavior. Focused repository
  behavior is implemented; PostgreSQL integration qualification remains open.
- [ ] API tests for all declared status classes and authorization/scope
  outcomes. Focused tests cover establishment, replay, changed duplicate,
  reads, authorization denial, and stale-version conflict; the complete status
  matrix remains open.
- [ ] UI tests for schedule editing, validation association, keyboard focus,
  conflict recovery, worklist filters, and accessible status updates. Focused
  tests cover authoritative loading, validation, exact-decimal submission, and
  conflict recovery; exhaustive keyboard/filter/accessibility evidence remains
  open.

Implementation note — 2026-10-17: the local US2 slice adds
`internal/invoicing/billing_schedule.go`, the owner migration
`db/migrations/invoicing/00002_create_billing_schedule_schema.sql`, Postgres
root/revision persistence, durable idempotency, typed OpenAPI request/read
surfaces (`invListBillingSchedules`, `invGetBillingSchedule`), and the
`INV-SCR-02` editor/worklist. The production wiring uses an explicit
unavailable reference validator until approved Customer, Contract,
InvoiceTemplate, and calendar ports are available; this prevents the schedule
module from reading another bounded context's adapter or schema directly.

### 7.3 User Story 3 — Generate invoices from approved billing inputs

As a Billing Operations Specialist, I want to generate invoices from approved
templates, schedules, and billing inputs so that the system produces a
reviewable invoice version with deterministic calculations and source evidence.

Direct traceability: `DLV-FR-INV-003`, `FR-INV-003`, `invGenerateInvoices`,
`INV-WS-01`, `INV-SCR-03`, and `INV-SCR-04`.

Acceptance criteria:

- [ ] An authorized actor can invoke invoice generation using the approved
  action route, permission, accounting scope, and idempotency contract.
- [ ] Generation validates that the selected template, schedule, customer,
  contract, billing inputs, tax configuration, and applicable source versions
  are present, compatible, and eligible.
- [ ] Charge calculations use exact decimal money, currency scale, configured
  rounding, and the applicable transaction/functional/presentation currency
  semantics; binary floating point is not used for authoritative amounts.
- [ ] Each established `GeneratedInvoice` exposes its identity, invoice
  version, source evidence, lines, amounts, currency, generation outcome, and
  validation exceptions to authorized users.
- [ ] A repeated generation identity and fingerprint returns the existing
  in-progress or terminal result without creating a second invoice or repeated
  downstream effect.
- [ ] Reuse of the identity with different functional content returns an
  idempotency conflict and establishes no additional invoice effect.
- [ ] Concurrent generation against the same eligible source is resolved by
  expected version and business identity; no duplicate invoice version or
  silently lost source update is accepted.
- [ ] A missing, stale, invalid, or unavailable dependency is represented as a
  typed blocked, pending, or exception outcome; generation does not fabricate
  a successful source result.
- [ ] Generation does not finalize an invoice, create an AR customer invoice,
  or submit a GL billing posting.
- [ ] `INV-SCR-03` shows source inputs, generated records, warnings, failures,
  age, support reference, and rerun eligibility. `INV-SCR-04` shows lines,
  money, tax/source evidence, version, and finalization blockers.
- [ ] Generation attempts, outcomes, source versions, and material calculation
  results are audited and observable without logging full sensitive payloads.

Suggested implementation steps:

1. Define the generation application boundary and source-reference ports using
   only approved contracts; record unresolved contract fields as open decisions.
2. Implement deterministic calculation, rounding, validation, and
   `GeneratedInvoice` version creation inside `internal/invoicing`.
3. Add durable generation identity/fingerprint coordination, aggregate
   versions, outbox records, and recovery/result lookup.
4. Wire `invGenerateInvoices` to the existing OpenAPI contract and exact
   permission.
5. Build the generation-run and invoice-preview surfaces with explicit
   partial, pending, blocked, failed, and rerun states.
6. Add fault, duplicate, concurrency, calculation, persistence, API, UI,
   accessibility, audit, telemetry, and recovery tests.

Required test evidence:

- [ ] Domain tests for deterministic calculation, currency scale, rounding,
  source eligibility, validation, versioning, and no-finalization behavior.
- [ ] Persistence and integration tests for durable idempotency, outbox/inbox,
  duplicate delivery, restart/ambiguous result recovery, and no duplicate
  generated invoice.
- [ ] API tests for success, validation, conflict, authorization, dependency
  unavailability, established-result lookup, and safe error classification.
- [ ] Frontend/component tests for generation progress, warnings, failures,
  rerun eligibility, invoice preview, keyboard operation, and announcements.
- [ ] Synthetic end-to-end evidence covering normal, boundary, duplicate,
  concurrency, dependency-failure, and recovery paths.

### 7.4 User Story 4 — Finalize generated invoices and hand off to AR

As a Billing Manager, I want to finalize an eligible generated invoice and see
the AR handoff outcome so that commercial billing becomes an immutable version
without duplicating receivable or billing accounting.

Direct traceability: `DLV-FR-INV-004`, `FR-INV-004`,
`invFinalizeGeneratedInvoices`, `INV-WS-01`, `INV-SCR-04`, and `INV-SCR-05`.

Readiness constraint: the DDD names the `InvoiceFinalized` / `IssueCustomerInvoice`
relationship, but the technical integration command inventory currently lists
only AR's `issue-customer-invoice.v1`. The exact Invoicing-owned publication
shape and delivery semantics must be confirmed before implementation. This
story must not create an unapproved event or topic.

Acceptance criteria:

- [ ] An authorized actor can request finalization for an eligible generated,
  unfinalized invoice using the approved action route, permission, and
  idempotency contract.
- [ ] Finalization revalidates the current generated-invoice version, source
  versions, calculation completeness, customer/tax configuration, applicable
  currency and period rules, and all finalization blockers at the authoritative
  boundary.
- [ ] Missing, unresolved, expired, unsupported, or inconsistent required tax
  results block finalization with an explainable typed outcome and no AR
  customer invoice effect.
- [ ] Successful finalization creates one immutable finalized invoice version,
  retains its source and calculation evidence, and prevents in-place edits.
- [ ] Finalization records one correlated AR handoff intent using the confirmed
  `InvoiceFinalized` / `IssueCustomerInvoice` contract and the stable invoice
  identity/version. Invoicing does not create the AR aggregate or billing
  journal.
- [ ] A duplicate finalization or duplicate handoff delivery returns the
  established result and cannot create a second customer invoice, receivable,
  or billing posting.
- [ ] An ambiguous or unavailable AR outcome remains visibly pending or
  exceptional until an authoritative result is reconciled; it is never shown
  as accepted merely because the local request was sent.
- [ ] A rejected downstream handoff retains the finalized commercial fact and
  exposes the typed rejection and permitted recovery path; destructive edit or
  silent retry with a new business identity is not allowed.
- [ ] `INV-SCR-04` shows finalization blockers and immutable version evidence;
  `INV-SCR-05` shows AR acceptance, rejection, pending, exception, correlation,
  and reconciliation status.
- [ ] Finalization and handoff actions retain authorization, actor, scope,
  source/version, correlation/causation, contract version, audit evidence, and
  support reference.

Suggested implementation steps:

1. Resolve and record the exact AR handoff contract, event/command ownership,
   version, payload allowlist, retry identity, and result-reconciliation rule.
2. Implement finalization readiness and immutable version transition in
   `GeneratedInvoice` without adding a new lifecycle identifier.
3. Implement the approved AR handoff port, transactional outbox, inbox/dedupe,
   result lookup, and typed pending/rejected/exception outcomes.
4. Wire `invFinalizeGeneratedInvoices` and the exact permission.
5. Build the finalization and AR-handoff screens with confirmation, blocker,
   pending, rejection, and reconciliation behavior.
6. Add cross-context duplicate, out-of-order, unavailable, recovery, audit,
   and single-accounting-owner evidence.

Required test evidence:

- [ ] Domain tests for finalization readiness, tax/source validation,
  immutability, version conflicts, and no-finalization on blocked inputs.
- [ ] Integration tests for the confirmed AR handoff, transactional outbox,
  inbox deduplication, duplicate/out-of-order delivery, ambiguous outcome
  reconciliation, and no duplicate AR/GL effects.
- [ ] API tests for authorization, validation, conflict, pending/unavailable,
  established-result, and downstream rejection outcomes.
- [ ] UI tests for blocker review, typed confirmation, AR outcome states,
  keyboard focus, live announcements, and immutable-version display.
- [ ] End-to-end evidence proving finalization produces one AR handoff and that
  AR—not Invoicing—owns customer-invoice and billing accounting.

### 7.5 User Story 5 — Recalculate an unfinalized invoice

As a Billing Operations Specialist, I want to recalculate an unfinalized
invoice from an authoritative new source version so that corrected billing
inputs can be reviewed without changing finalized facts.

Direct traceability: `DLV-FR-INV-005`, `FR-INV-005`,
`invRecalculateUnfinalizedInvoices`, `INV-WS-01`, `INV-SCR-03`, and `INV-SCR-04`.

Acceptance criteria:

- [ ] An authorized actor can request recalculation only for an unfinalized
  generated invoice using the approved action route and idempotency contract.
- [ ] The command requires and validates an authoritative new source version;
  stale, missing, or incompatible source evidence is rejected without a
  partial calculation effect.
- [ ] Recalculation records the new invoice version, changed calculations,
  source-version differences, validation status, and lineage to the prior
  unfinalized version.
- [ ] No finalized invoice version, AR customer invoice, receivable, or billing
  posting is edited or overwritten by recalculation.
- [ ] The recalculated result uses exact decimal and currency-policy semantics
  and re-evaluates applicable tax, payment, customer, contract, and
  finalization-readiness controls.
- [ ] An exact duplicate returns the established recalculation result; changed
  content under the same identity returns an idempotency conflict; concurrent
  source changes return a typed version conflict.
- [ ] Recalculation does not implicitly finalize, cancel, or hand off the
  invoice. The next permitted action is visible to the authorized user.
- [ ] Dependency unavailability, calculation failure, and business validation
  rejection remain distinct outcomes with stable support references.
- [ ] The UI shows prior and current source versions, changed calculations,
  validation status, and the reason a version is or is not eligible for
  finalization.
- [ ] Recalculation is audited, observable, retry-safe, and privacy-safe.

Suggested implementation steps:

1. Define the approved source-version comparison and invoice-version lineage
   semantics without inventing a new public state.
2. Implement unfinalized-only recalculation and exact calculation reuse in the
   `GeneratedInvoice` aggregate/application handler.
3. Add version, idempotency, persistence, audit, outbox, and recovery support.
4. Wire `invRecalculateUnfinalizedInvoices` and its exact permission.
5. Add comparison and validation views to the generation/preview surfaces.
6. Add domain, concurrency, API, persistence, UI, accessibility, and recovery
   evidence.

Required test evidence:

- [ ] Domain tests for eligibility, source-version changes, exact arithmetic,
  lineage, finalization protection, and cancellation protection.
- [ ] Persistence tests for version history, expected-version conflicts,
  idempotency, restart, and no mutation of finalized facts.
- [ ] API tests for validation, conflict, duplicate, authorization, and
  dependency-unavailable outcomes.
- [ ] UI tests for source-version comparison, changed-calculation display,
  validation association, keyboard operation, and status announcements.

### 7.6 User Story 6 — Cancel an unfinalized invoice

As a Billing Operations Specialist or Billing Manager, I want to cancel an
unfinalized invoice with a recorded reason so that an invalid or no-longer-
required billing result cannot be finalized or handed to AR.

Direct traceability: `DLV-FR-INV-006`, `FR-INV-006`,
`invCancelUnfinalizedInvoices`, `INV-WS-01`, and `INV-SCR-04`.

Acceptance criteria:

- [ ] An authorized actor can request cancellation only for an unfinalized
  generated invoice using the approved action route, permission, reason, and
  idempotency metadata.
- [ ] The command revalidates the current invoice version and cancellation
  eligibility; a finalized invoice or established AR handoff is rejected with
  a typed outcome.
- [ ] A successful cancellation retains the affected invoice version, reason,
  actor, scope, correlation, source evidence, and cancellation timestamp in
  immutable lineage.
- [ ] Cancellation cannot alter a finalized downstream invoice, receivable,
  billing posting, or other established financial fact.
- [ ] A repeated cancellation with the same identity and fingerprint returns
  the established result. Changed content under the same identity returns an
  idempotency conflict and changes nothing.
- [ ] Concurrent finalization, recalculation, and cancellation operations
  resolve through expected versions so that only one valid lifecycle outcome
  wins and no partial outcome is hidden.
- [ ] The cancellation confirmation identifies the record, scope, invoice
  version, reason, and effect, and requires an explicit user action for the
  material transition.
- [ ] The worklist and invoice preview show cancellation result, reason,
  affected version, owner, and the fact that no finalized downstream invoice
  was altered.
- [ ] Cancellation audit evidence and telemetry include a support reference,
  authorization evidence, and safe data classification.

Suggested implementation steps:

1. Confirm the approved cancellation reason representation and eligibility
   rules; do not introduce a reason-code identifier absent from the source
   contract.
2. Implement the unfinalized-only cancellation transition and immutable
   lineage in `GeneratedInvoice`.
3. Add persistence, idempotency, audit, and transactional coordination.
4. Wire `invCancelUnfinalizedInvoices` and its exact permission.
5. Add the shared confirmation, worklist state, and finalized-state conflict
   behavior.
6. Add concurrency, API, persistence, UI, accessibility, audit, and recovery
   evidence.

Required test evidence:

- [ ] Domain tests for eligibility, reason validation, immutability, and
  finalization/handoff protection.
- [ ] Persistence tests for duplicate cancellation, changed duplicate,
  expected-version race, audit evidence, and no-partial-commit behavior.
- [ ] API tests for authorization, validation, finalized-state rejection,
  conflict, established result, and unavailable outcomes.
- [ ] UI tests for typed confirmation, reason display, focus restoration,
  status meaning, and accessible error/recovery behavior.

## 8. Cross-cutting delivery contract

### 8.1 Command and result behavior

- State-changing operations use the approved command request, stable business
  idempotency identity, canonical request fingerprint, explicit scope where
  applicable, correlation, and source/expected version information.
- The same identity and fingerprint return the established in-progress or
  terminal result. Changed content under the same identity returns a typed
  idempotency conflict with no second effect.
- Expected-version conflicts, domain validation rejection, authorization
  denial, dependency unavailability, capacity control, and internal failure
  remain distinct outcomes.
- Ambiguous external or cross-context outcomes remain pending or exceptional
  until an authoritative lookup/reconciliation establishes the result.
- Retry uses the original business identity and fingerprint. A corrected
  business request uses a new source version and identity.

### 8.2 Persistence and transaction boundaries

- The `invoicing` module owns `invoice_template`, `billing_schedule`, and
  `generated_invoice` and their repositories. Each aggregate retains
  `aggregate_version` and immutable lineage needed by its lifecycle.
- A local aggregate change, idempotency reservation/result, audit reference,
  and outbox record commit within the owning module transaction.
- Cross-context effects use approved ports, commands, events, inbox
  deduplication, and transactional outbox behavior. No module reads another
  module's schema or adapter directly.
- Migrations, SQLC or generated persistence artifacts, and OpenAPI artifacts
  must be reproducible and drift-checked when implementation begins.

### 8.3 Authorization, audit, privacy, and telemetry

- IAM is the authorization decision point. API enforcement is authoritative;
  hidden UI actions are only a usability aid.
- Each operation uses its exact `finance.inv.*` permission and applicable
  accounting-scope, legal-entity, customer, amount, currency, period,
  sensitivity, and action dimensions.
- Material actions retain actor, authentication subject, authorization or
  approval reference when applicable, scope, source aggregate, action,
  correlation/causation, expected/source version, and before/after or event
  fingerprints.
- Logs, traces, metrics, errors, exports, and support views exclude secrets,
  unrestricted personal/tax data, and full sensitive payloads. User-visible
  outcomes retain a stable support reference.
- Operational telemetry distinguishes validation rejection, authorization
  denial, version/idempotency conflict, dependency unavailability, capacity
  control, and internal failure, and exposes pending/exception age where
  applicable.

### 8.4 UI and accessibility

- Use the approved routes `/invoicing/inv-ws-01`, `/invoicing/inv-scr-01`,
  `/invoicing/inv-scr-02`, `/invoicing/inv-scr-03`,
  `/invoicing/inv-scr-04`, and `/invoicing/inv-scr-05`.
- Worklists expose applicable scope, owner, state, amount/currency, age,
  exception, approval or authorization status, and next action.
- Forms use canonical decimal-string submission, locale-aware presentation,
  linked validation summaries, typed confirmation for material transitions,
  and explicit conflict recovery; mutations are not automatically resubmitted.
- Status, severity, validation, pending, exception, and handoff meaning are
  conveyed with text and accessible semantics, not color alone.
- Keyboard focus, live-region updates, screen-reader labels, table structure,
  responsive layout, contrast, 200% text size, 400% browser zoom, and reduced
  motion are part of each story's evidence.

## 9. Dependencies and sequencing

1. Confirm the six approved OpenAPI paths, generated artifacts, common command
   contract, permission catalog, and current implementation baseline.
2. Resolve the Invoicing-to-AR publication/handoff contract before US4 leaves
   planning. Do not fill the technical event-inventory gap with an invented
   identifier.
3. Implement US1 and US2 so templates and schedules provide versioned inputs
   for generation. They may be developed in parallel after the shared module
   boundary is established.
4. Implement US3 as the first commercial-document vertical slice, including
   exact calculation, persistence, API, UI, authorization, audit, and recovery.
5. Implement US5 and US6 against the unfinalized `GeneratedInvoice` lifecycle;
   both must protect finalization and downstream facts.
6. Implement US4 after the AR handoff contract is confirmed, then qualify the
   finalization-to-AR path with duplicate, delayed, rejected, and unavailable
   outcomes.
7. Coordinate the resulting handoff with EP-AR-001. AR owns customer-invoice
   issuance, receivable state, and billing accounting; Invoicing supplies only
   its commercial document and authoritative source evidence.
8. Run the M4 workflow and quality evidence with synthetic data before making
   any production or release claim.

## 10. Definition of Ready

- [ ] The approved DDD, PRD, UX, NFR, system-design, API, persistence, and
  integration references have been reviewed for the story being implemented.
- [ ] The story's exact FR/DLV/API/permission/UX identifiers are recorded and
  no new domain identifier is needed to begin.
- [ ] Required source-reference and cross-context contracts are available;
  unresolved source gaps are recorded as implementation gates.
- [ ] Synthetic fixtures cover at least one valid, boundary, duplicate,
  conflict, dependency-failure, and recovery case.
- [ ] The owning module/schema boundary and affected ports are agreed before
  persistence or adapter code is added.

## 11. Definition of Done

- [ ] All six child stories have implementation evidence for their applicable
  domain, persistence, API, authorization, UI, audit, observability, and
  integration behavior.
- [ ] The six FR requirements, their DLV items, applicable UX criteria, NFR
  controls, and M4 quality gates map to successful evidence or an approved,
  time-bound deferral.
- [ ] InvoiceTemplate, BillingSchedule, and GeneratedInvoice remain owned by
  `internal/invoicing`; no cross-module schema or adapter access exists.
- [ ] Finalized commercial invoice facts are immutable and the AR handoff does
  not create duplicate customer-invoice or billing accounting effects.
- [ ] Exact monetary calculations, currency evidence, source-version lineage,
  idempotency, optimistic concurrency, transactional outbox/inbox, and
  recovery behavior are verified.
- [ ] All user-visible INV screens meet the applicable accessibility and
  responsive interaction requirements.
- [ ] OpenAPI, generated Go/TypeScript artifacts, persistence code, migrations,
  tests, and documentation are synchronized.
- [ ] No critical or high finance-integrity, authorization, privacy, audit,
  accessibility, recovery, or compatibility defect remains without an
  approved exception.
- [ ] EP-AR-001 receives a documented, versioned, idempotent handoff contract
  and the ownership boundary is demonstrated end to end.

## 12. Required verification evidence

- [ ] Focused domain tests for all three aggregates and every acceptance rule.
- [ ] Persistence/migration tests for empty database, upgrade, owner-only
  access, aggregate versions, idempotency, audit references, and rollback.
- [ ] API/OpenAPI lint, contract, generated-artifact, authorization, error,
  correlation, idempotency, and expected-version tests.
- [ ] Integration tests for outbox/inbox, duplicate delivery, out-of-order or
  delayed handoff, dependency outage, result lookup, and reconciliation.
- [ ] Frontend unit/component tests for all five INV screens, route protection,
  forms, tables, status surfaces, validation, conflicts, and sensitive-value
  masking.
- [ ] Automated accessibility checks plus keyboard, screen-reader, zoom,
  contrast, reflow, and reduced-motion evidence for representative screens.
- [ ] Workflow evidence for normal, boundary, duplicate, concurrency,
  dependency-failure, restart, and recovery paths, including the confirmed AR
  handoff.
- [ ] Observability, privacy, audit, support-reference, and redaction checks.
- [ ] Focused local checks such as `pnpm test:fast` or `make check-fast`, plus
  narrower module/API/UI checks discovered when implementation begins. Slow,
  persistence-integration, and release checks remain opt-in unless requested.

## 13. Risks and open decisions

- **AR handoff contract gap:** DDD §1.2 names `InvoiceFinalized` and
  `IssueCustomerInvoice`, while the technical command inventory currently
  exposes AR's `issue-customer-invoice.v1` without a complete Invoicing-owned
  publication payload. Resolve ownership, topic/event identity, payload
  allowlist, retry identity, and reconciliation semantics before US4.
- **Generic GL-port wording:** System design §6 lists GL posting among generic
  Invoicing outbound ports, while DDD §5.2 explicitly says Invoicing never
  submits the billing journal. This plan follows the more specific DDD
  ownership rule: AR owns customer billing posting. Clarify the generic module
  wording if it is intended to authorize anything else.
- **Approved billing-input contract:** The PRD requires generation from
  approved billing inputs, but the current Invoicing technical inventory does
  not enumerate every upstream source contract. Confirm the minimum source
  versions and failure semantics before implementing generation.
- **Tax and revenue dependencies:** Finalization needs authoritative tax and,
  where applicable, revenue-classification evidence. Confirm the exact
  reference contract and effective-date behavior without moving Tax Filing or
  Revenue Recognition ownership into Invoicing.
- **Generation scheduling:** The PRD defines billing schedules and an invoice
  generation action but does not authorize a new scheduler contract in this
  epic. Treat generation as the approved action until an authoritative
  scheduling design is added.
- **No current capability implementation:** Generated API artifacts exist, but
  the owning module, persistence, handlers, and UI remain to be built. Do not
  treat generated stubs as functional evidence.

## 14. Source references

- [Finance Domain Model & Use Cases — DDD Baseline](../../specs/finance_domain_model_ddd.md)
- [Finance Functional PRD](../../specs/prd/01_finance_functional_prd_v1.5.md)
- [Finance Functional Requirements Catalog](../../specs/prd/02_finance_functional_requirements_catalog_v1.5.md)
- [Finance Functional Traceability and Acceptance](../../specs/prd/03_finance_functional_traceability_acceptance_v1.5.md)
- [Finance UX Workflow Specification](../../specs/finance_ux_workflow_specification_v1.0.md)
- [Finance Nonfunctional Requirements](../../specs/finance_nonfunctional_requirements_v1.0.md)
- [Solution Architecture Overview](../../specs/system_design/01_solution_architecture_overview_v1.0.md)
- [Application Module Design](../../specs/system_design/02_application_module_design_v1.0.md)
- [Data and Integration Architecture](../../specs/system_design/03_data_integration_architecture_v1.0.md)
- [Backend Module Technical Specifications](../../specs/technical_specifications/01_backend_module_specifications_v1.0.md)
- [API and OpenAPI Technical Specifications](../../specs/technical_specifications/02_api_openapi_specifications_v1.0.md)
- [Database Persistence Technical Specifications](../../specs/technical_specifications/03_database_persistence_specifications_v1.0.md)
- [Events, Workers, and Integration Technical Specifications](../../specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md)
- [Frontend UI Technical Specifications](../../specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md)
- [Observability and Operations Technical Specifications](../../specs/technical_specifications/08_observability_operations_specifications_v1.0.md)
- [Testing, Performance, and Recovery Technical Specifications](../../specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md)
- [Technical Traceability, Decisions, and Verification](../../specs/technical_specifications/10_technical_traceability_decisions_v1.0.md)
- [Finance Platform Delivery Plan](../../specs/finance_delivery_plan_v1.0.md)
- [Invoicing OpenAPI paths](../../../contracts/openapi/paths/invoicing.yaml)
- [TALLY Roadmap](../../../ROADMAP.md)
