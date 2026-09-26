# EP-OMD-001 — Organization and Master Data User Stories

| Field | Value |
|---|---|
| Epic | `EP-OMD-001` — Organization and master data |
| Status | Planned — story decomposition created on 2026-09-26; implementation evidence is not yet available |
| Milestone | `M1` |
| Dependencies | `EP-PLAT-001`, `EP-IAM-001` |
| Delivery items | `DLV-GFR-019`, `DLV-FR-OMD-001` through `DLV-FR-OMD-006` |
| Owning bounded context | Organization & Master Data — `internal/organization`, `organization` schema |
| Primary users | Master Data Steward; Finance Administrator |
| Exit evidence | All six stories pass the acceptance and verification evidence in this document; `EP-OMD-001` remains open until then. |

## 1. Outcome

Deliver the Organization & Master Data slice as the authoritative owner for
legal entities, parties, customer profiles, vendor profiles, and fiscal
calendars. Authorized users can maintain validated, effective-dated master
data, review approval and publication state, and make approved versions
available to dependent capabilities without allowing downstream modules to
write OMD records directly.

Historical consumers retain the master-data identity and version applicable to
their facts. Sensitive party and bank-detail information is minimized,
masked, and kept out of errors, logs, notifications, exports, and published
payloads unless an approved contract explicitly requires a safe reference.

## 2. Learning objective

Learn how to implement a supporting bounded context with aggregate ownership,
effective dating, optimistic concurrency, idempotent commands, approval
handoff, transactional publication, and explicit authorization while keeping
downstream contexts dependent on published language and authoritative
reference contracts.

## 3. Scope

- Maintain `LegalEntity` identity, registrations, addresses, ownership
  interests, lifecycle state, and effective-date ranges.
- Maintain `Party` identity, type, status, contact methods, addresses,
  classifications, bank-detail references, and applicable cooling-off state.
- Maintain `CustomerProfile` terms, limits, billing preferences, and tax
  treatment.
- Maintain `VendorProfile` payment terms, withholding treatment, and remittance
  preferences.
- Define and maintain `FiscalCalendar` patterns and `CalendarPeriod`
  definitions, including dependent-scope impact review.
- Publish approved master-data changes while preserving record identity,
  applicable status, effective dates, approval evidence, and reference
  versions.
- Provide the existing master-data worklist and record screens as the UI
  entry points for search, review, maintenance, and publication work.
- Apply IAM authorization, material-action audit, safe error handling,
  correlation, idempotency, and bounded operational telemetry at each command
  boundary.

## 4. Explicit exclusions

- No ownership of `FiscalPeriod`; that authority remains with Fiscal Period
  Management.
- No ledger, accounting-book, chart-of-accounts, account, journal, posting,
  invoice, receivable, payable, liability, or payment state changes.
- No direct writes to another bounded context's adapter or database schema.
- No Workflow & Approvals implementation; OMD consumes an approved decision
  where the applicable requirement requires approval.
- No Audit Integrity chain implementation; OMD uses the approved audit port and
  does not become the authoritative audit-chain owner.
- No raw bank account data, credentials, provider tokens, secrets, or
  unrestricted tax/personal data in OMD records, API errors, telemetry,
  notifications, exports, or published payloads.
- No new public route, event name, database schema/table family, cross-schema
  query, or generated API contract. Existing master-data contracts are wired
  only where their approved ownership and semantics are implemented.
- No production-data qualification, live Entra qualification, finance-module
  enforcement, or full audit-chain qualification.

## 5. Ownership and domain rules

### 5.1 Owning component

OMD owns `LegalEntity`, `Party`, `CustomerProfile`, `VendorProfile`, and
`FiscalCalendar` aggregates in `internal/organization` and the `organization`
schema. The persistence contract names the primary tables
`legal_entity`, `party`, `customer_profile`, `vendor_profile`, and
`fiscal_calendar`, accessed through the owning module's repository interfaces.

Business contexts consume stable identifiers and only the immutable snapshots
needed for their own facts. The OMD module is the only writer of these
aggregates.

### 5.2 Domain objects and invariants

- `LegalEntity` contains `Registration`, `EntityAddress`, and
  `OwnershipInterest`; its value objects include `LegalEntityId`, `LegalName`,
  `FunctionalCurrency`, `PresentationCurrency`, `TaxRegistrationId`, and
  `EffectiveDateRange`.
- `Party` contains `ContactMethod`, `Address`, and `BankDetailReference`; its
  value objects include `PartyId`, `PartyType`, `TaxIdentifier`, and
  `PartyStatus`.
- `CustomerProfile` contains `CustomerId`, `PartyId`, `CreditTerms`,
  `CreditLimit`, `BillingPreference`, and `TaxTreatment`.
- `VendorProfile` contains `VendorId`, `PartyId`, `PaymentTerms`,
  `WithholdingTreatment`, and `RemittancePreference`.
- `FiscalCalendar` contains `CalendarPeriod`; its value objects include
  `CalendarId`, `CalendarType`, and `PeriodPattern`.
- Scoped uniqueness, lifecycle status, effective dates, approvals where
  applicable, historical reference versions, and command-specific validation
  are enforced at the owning aggregate/application boundary.
- Effective-dated changes preserve the rule and version used by historical
  transactions. A later master-data change never rewrites an established
  downstream fact.
- Credentials and bank tokens are not domain values. A bank-detail reference
  may carry only the approved reference and consent/provider metadata required
  by the contract.
- Approval decisions remain owned by Workflow & Approvals. OMD applies an
  immutable approved decision reference only after revalidating the current
  aggregate version and business state.
- Every command carries explicit tenant/legal-entity or other required scope;
  scope is never inferred from ambient context alone.

## 6. Traceability

| Source | Coverage in this epic |
|---|---|
| `DLV-GFR-019` / `GFR-019` | Effective-dated configuration preserves the rule and version used by historical transactions. |
| `DLV-FR-OMD-001` / `FR-OMD-001` | User Story 1 — Maintain legal entities. |
| `DLV-FR-OMD-002` / `FR-OMD-002` | User Story 2 — Maintain parties. |
| `DLV-FR-OMD-003` / `FR-OMD-003` | User Story 3 — Maintain customer profiles. |
| `DLV-FR-OMD-004` / `FR-OMD-004` | User Story 4 — Maintain vendor profiles. |
| `DLV-FR-OMD-005` / `FR-OMD-005` | User Story 5 — Maintain fiscal calendars. |
| `DLV-FR-OMD-006` / `FR-OMD-006` | User Story 6 — Publish approved master-data changes. |
| DDD §§1.1, 1.2, 2.1, 9, 10, 11 | Aggregate ownership, context relationships, OMD model, concurrency, effective dating, audit, retention, and privacy. |
| PRD §3.1; UX §7.1 | OMD capability responsibilities, actors, screens, worklist behavior, and authoritative records. |
| `NFR-SEC-004`, `NFR-SEC-009`–`012`, `NFR-SEC-017` | Default-deny authorization, protection in transit/at rest, attributable changes, and permission-preserving exports/bulk actions. |
| `NFR-PRV-001`–`003`, `NFR-PRV-008`, `NFR-PRV-009` | Purpose limitation, minimization, field-level privacy, synthetic non-production data, and safe notifications. |
| `NFR-AUD-001`, `NFR-AUD-003` | Material actions and state transitions carry attributable, correlated audit evidence. |
| `NFR-REL-014`, `NFR-REL-015` | Historical source-version preservation and deterministic import/export or reconciliation evidence where applicable. |
| `NFR-PERF-006` | Authorized master-data search and source-reference lookup performance qualification. |
| `ARC-SEC-002`, `ARC-PRV-001`, `ARC-AUD-001`, `ARC-OBS-001`, `TADR-012` | Security, privacy, audit-boundary, observability, and operational-telemetry separation controls. |

## 7. User stories

### User Story 1 — Maintain legal entities

**Delivery item:** `DLV-FR-OMD-001` / `FR-OMD-001`
**Existing API operation:** `omdMaintainLegalEntities`
**Permission:** `finance.omd.maintain.legal.entities`
**Primary screen:** `OMD-SCR-01`; worklist entry `OMD-WS-01`

As a Master Data Steward, I want to maintain legal-entity identity and
effective-dated registrations, addresses, and ownership interests so that
dependent capabilities can use an authoritative organizational record.

Acceptance criteria:

- [ ] An authorized command can create or maintain the legal-entity fields
  defined by `FR-OMD-001`, including end-dating a no-longer-applicable entity.
- [ ] Scoped uniqueness, lifecycle status, effective-date, approval-where-
  applicable, and historical-version validation outcomes are explicit and
  typed; invalid input does not partially change the aggregate.
- [ ] A successful result identifies the accepted aggregate/version and
  applicable effective interval without exposing restricted values.
- [ ] `If-Match`, idempotency, correlation, audit, and material-action
  evidence follow the existing platform contracts; stale or replayed commands
  cannot silently overwrite a newer version.
- [ ] `OMD-SCR-01` and `OMD-WS-01` expose identity, state, effective dates,
  validation, approval, and next action without creating a second mutation
  surface in another capability.

Suggested implementation steps:

1. Define the LegalEntity aggregate rules and versioned application command in
   `internal/organization`.
2. Add owning-schema migration, repository, and transaction coordination
   using the existing database and idempotency primitives.
3. Wire the existing OpenAPI operation and permission through the HTTP adapter;
   preserve typed common results and problem details.
4. Build the record and worklist projections in the master-data UI.
5. Add domain, persistence, API, authorization, concurrency, accessibility,
   and negative-disclosure evidence.

Required test evidence:

- [ ] Domain tests for identity, child records, lifecycle, effective dates,
  validation, and aggregate-version conflicts.
- [ ] Transactional API tests for authorization, idempotent retry, stale
  version, typed problem details, audit failure, and correlation propagation.
- [ ] Persistence tests proving organization-schema ownership and historical
  version preservation.
- [ ] UI and Playwright tests for `OMD-SCR-01`, masked sensitive values,
  keyboard behavior, and safe status/error announcements.

### User Story 2 — Maintain parties

**Delivery item:** `DLV-FR-OMD-002` / `FR-OMD-002`
**Existing API operation:** `omdMaintainParties`
**Permission:** `finance.omd.maintain.parties`
**Primary screen:** `OMD-SCR-02`; worklist entry `OMD-WS-01`

As a Master Data Steward, I want to maintain party identity, status, contact
methods, addresses, classifications, and bank-detail references so that
customer and vendor capabilities can reference safe, current party data.

Acceptance criteria:

- [ ] An authorized command maintains the Party fields named by `FR-OMD-002`
  and returns the accepted identity, `PartyStatus`, version, and validation
  outcome.
- [ ] Bank details are represented only by approved references; raw account
  values, credentials, and provider tokens never enter records, responses,
  errors, logs, exports, notifications, or published payloads.
- [ ] Vendor-related bank-detail changes expose applicable approval and
  cooling-off state without allowing OMD to bypass IAM, Workflow, or payment
  controls.
- [ ] Field-level authorization is independent: a user may see allowed party
  fields while restricted bank/tax/personal values remain masked or absent.
- [ ] Conflicts, invalid references, stale versions, unavailable audit or
  dependency checks, and unauthorized attempts fail closed with safe typed
  outcomes.

Suggested implementation steps:

1. Implement Party and its contact, address, classification, and
   `BankDetailReference` rules in the owning module.
2. Connect safe reference validation and applicable approval/cooling-off
   status through existing ports rather than importing another schema.
3. Persist and expose only the approved field projection through the existing
   operation and common problem contract.
4. Implement `OMD-SCR-02` masking, accessible names, reveal/audit boundaries,
   and safe filtering.
5. Prove negative disclosure behavior across API, UI, telemetry, and export
   projections.

Required test evidence:

- [ ] Unit tests for PartyStatus, field-level independence, reference-only
  bank details, approval/cooling-off outcomes, and version conflicts.
- [ ] API tests proving no raw secret or sensitive value appears in any
  response or problem detail.
- [ ] Audit and telemetry tests proving actor, scope, purpose, classification,
  outcome, correlation, and support references are bounded and safe.
- [ ] Component and Playwright tests for masking, accessible-name safety,
  keyboard reveal behavior, and denied-field filters.

### User Story 3 — Maintain customer profiles

**Delivery item:** `DLV-FR-OMD-003` / `FR-OMD-003`
**Existing API operation:** `omdMaintainCustomerProfiles`
**Permission:** `finance.omd.maintain.customer.profiles`
**Primary screen:** `OMD-SCR-03`; worklist entry `OMD-WS-01`

As a Master Data Steward, I want to maintain customer terms, limits, billing
preferences, and tax treatment against an authoritative party so that AR and
invoicing can consume a validated customer reference.

Acceptance criteria:

- [ ] An authorized command maintains the `CustomerProfile` fields defined by
  `FR-OMD-003`, including its `PartyId`, and reports accepted or rejected
  validation without partial state.
- [ ] Customer profile changes preserve aggregate version, effective-date,
  approval-where-applicable, and historical-reference rules.
- [ ] A missing, unauthorized, stale, or invalid party reference is rejected
  safely; OMD does not create customer invoices, receivables, credit, or
  collection state.
- [ ] Customer tax, limit, and terms values are classified and projected only
  to authorized fields in records, filters, notifications, exports, and
  telemetry.
- [ ] `OMD-SCR-03` identifies the authoritative Party and profile version and
  shows the next permitted action without exposing unrelated parties.

Suggested implementation steps:

1. Implement CustomerProfile invariants and its Party reference in OMD.
2. Revalidate current Party and scope versions through approved interfaces.
3. Add the repository transaction, idempotent command handling, and existing
   OpenAPI operation.
4. Add customer profile projections to the shared master-data worklist and
   record screen.
5. Test stale references, field authorization, safe errors, and downstream
   snapshot/version handoff.

Required test evidence:

- [ ] Domain tests for terms, limits, billing preference, tax treatment,
  PartyId, lifecycle, effective dates, and conflicts.
- [ ] API tests for typed rejection, authorization, idempotency, and safe
  correlation/support references.
- [ ] Integration tests proving OMD publishes identifiers/versions without
  writing AR or invoicing state.
- [ ] UI tests for field-filtered details, counts, filters, and export-safe
  projections.

### User Story 4 — Maintain vendor profiles

**Delivery item:** `DLV-FR-OMD-004` / `FR-OMD-004`
**Existing API operation:** `omdMaintainVendorProfiles`
**Permission:** `finance.omd.maintain.vendor.profiles`
**Primary screen:** `OMD-SCR-03`; worklist entry `OMD-WS-01`

As a Master Data Steward, I want to maintain vendor payment terms,
withholding treatment, and remittance preferences against an authoritative
party so that AP can consume a validated vendor reference without OMD owning
the payable or payment lifecycle.

Acceptance criteria:

- [ ] An authorized command maintains the `VendorProfile` fields defined by
  `FR-OMD-004`, including its `PartyId`, and reports all validation outcomes
  without partial state.
- [ ] Vendor profile revisions preserve aggregate versions, effective dates,
  applicable approvals, and historical references.
- [ ] Bank-detail references and remittance preferences remain minimized and
  field-authorized; raw account values and provider secrets are never exposed.
- [ ] Invalid or unavailable Party, approval, authorization, audit, or
  dependency checks fail closed; OMD does not create vendor invoices,
  liabilities, payment requests, or payment instructions.
- [ ] The UI clearly separates profile maintenance from AP payment behavior
  and exposes only the next permitted action.

Suggested implementation steps:

1. Implement VendorProfile rules and Party linkage in OMD.
2. Integrate applicable approval, cooling-off, and field-authorization ports.
3. Add transactional persistence, idempotency, optimistic concurrency, and
   the existing OpenAPI operation.
4. Add vendor projections to `OMD-SCR-03`, including safe rejection and
   publication states.
5. Test downstream AP handoff as an immutable reference/snapshot boundary.

Required test evidence:

- [ ] Domain tests for payment terms, withholding treatment, remittance
  preference, PartyId, effective dates, and version conflicts.
- [ ] API tests for denied, stale, unavailable, and validation outcomes with
  no sensitive leakage.
- [ ] Integration tests proving no AP or payment schema writes occur.
- [ ] UI and Playwright tests for restricted remittance/bank fields, status
  announcements, and safe bulk/export projections.

### User Story 5 — Maintain fiscal calendars

**Delivery item:** `DLV-FR-OMD-005` / `FR-OMD-005`
**Existing API operation:** `omdMaintainFiscalCalendars`
**Permission:** `finance.omd.maintain.fiscal.calendars`
**Primary screen:** `OMD-SCR-04`; worklist entry `OMD-WS-01`

As a Finance Administrator, I want to define and maintain fiscal-calendar
patterns and periods so that dependent capabilities can validate applicable
dates and review scope impact without OMD taking ownership of fiscal-period
state.

Acceptance criteria:

- [ ] An authorized command creates or maintains the `FiscalCalendar` pattern
  and `CalendarPeriod` definitions required by `FR-OMD-005`.
- [ ] Calendar validation rejects invalid, conflicting, or incomplete period
  definitions as a typed result before persistence; no partial calendar is
  published.
- [ ] Effective dates, lifecycle state, aggregate version, historical
  references, and dependent-scope impact are visible and preserved.
- [ ] OMD does not modify `FiscalPeriod` state, posting gates, close state, or
  any ledger-bound record owned by another context.
- [ ] `OMD-SCR-04` supports accessible period editing and impact review while
  `OMD-WS-01` links to the authoritative calendar record.

Suggested implementation steps:

1. Implement FiscalCalendar and CalendarPeriod validation in OMD.
2. Define the approved dependent-scope reference/impact query boundary
   without cross-schema writes or invented period authority.
3. Add transactional persistence, idempotent retry, optimistic concurrency,
   and the existing OpenAPI operation.
4. Build the calendar editor and impact review projection.
5. Test version preservation and downstream consumption by reference.

Required test evidence:

- [ ] Domain tests for calendar type, pattern, periods, effective dates,
  lifecycle, invalid overlaps/order, and version conflicts according to the
  approved domain rules.
- [ ] API and persistence tests for typed validation, authorization, retry,
  and organization-schema ownership.
- [ ] Integration tests proving fiscal-period and GL state remain outside OMD.
- [ ] Accessibility and semantic tests for period editing, impact review,
  keyboard operation, and safe validation announcements.

### User Story 6 — Publish approved master-data changes

**Delivery item:** `DLV-FR-OMD-006` / `FR-OMD-006`
**Existing API operation:** `omdPublishApprovedMasterDataChanges`
**Permission:** `finance.omd.publish.approved.master.data.changes`
**Primary screen:** `OMD-SCR-05`; worklist entry `OMD-WS-01`

As a Finance Administrator, I want to publish approved master-data versions
to dependent capabilities so that consumers can use the correct record
identity, status, effective date, and source version without bypassing
approval or changing historical facts.

Acceptance criteria:

- [ ] Publication accepts only an approved, current, applicable aggregate
  version and records the approval evidence required by the applicable rule.
- [ ] The result identifies the published record/version, applicable status or
  effective date, dependent-capability availability, and any safe publication
  rejection.
- [ ] Publication is idempotent and replay-safe; retries cannot duplicate a
  state transition or silently publish a superseded version.
- [ ] Aggregate state and any approved integration publication are coordinated
  transactionally through the existing outbox/integration boundary. Raw
  policy payloads, secrets, and restricted values are not published.
- [ ] `OMD-SCR-05` distinguishes pending approval, approved, published,
  rejected, stale, and unavailable states and provides only the next permitted
  action.
- [ ] Consumers receive identifiers and required immutable snapshots through
  the existing published-language/reference contract; they do not gain write
  access to OMD storage.

Suggested implementation steps:

1. Define the application-level publication decision and revalidation rules
   using the existing approval and audit ports.
2. Coordinate OMD state, outbox intent, idempotency result, and audit evidence
   in one transaction where the existing platform supports it.
3. Wire the existing publication operation without adding a new public route
   or event name.
4. Build publication review and dependent-availability projections.
5. Test duplicate delivery, crash/retry, stale approval, dependency
   unavailability, and safe payload projection.

Required test evidence:

- [ ] Domain/application tests for approved, rejected, stale, superseded,
  duplicate, and unavailable publication outcomes.
- [ ] Transactional API and outbox tests for idempotency, audit dependency
  failure, correlation/causation, and no partial publication.
- [ ] Contract tests for consumer identifiers, versions, effective dates, and
  restricted-field omission.
- [ ] UI and Playwright tests for publication states, status announcements,
  keyboard actions, and safe error/support references.

## 8. Cross-cutting delivery contract

### 8.1 Existing API surface

The repository already contains the following OpenAPI operations. Story
implementation must preserve their paths, operation IDs, common
`CommandRequest`/`EstablishedResult`/problem-details types, correlation and
idempotency headers, and declared status classes. No additional public API is
part of this epic's story decomposition.

| Operation | Method and path | Permission |
|---|---|---|
| `omdMaintainLegalEntities` | `PUT /master-data/configuration/maintain-legal-entities` | `finance.omd.maintain.legal.entities` |
| `omdMaintainParties` | `PUT /master-data/configuration/maintain-parties` | `finance.omd.maintain.parties` |
| `omdMaintainCustomerProfiles` | `PUT /master-data/configuration/maintain-customer-profiles` | `finance.omd.maintain.customer.profiles` |
| `omdMaintainVendorProfiles` | `PUT /master-data/configuration/maintain-vendor-profiles` | `finance.omd.maintain.vendor.profiles` |
| `omdMaintainFiscalCalendars` | `PUT /master-data/configuration/maintain-fiscal-calendars` | `finance.omd.maintain.fiscal.calendars` |
| `omdPublishApprovedMasterDataChanges` | `POST /master-data/actions/publish-approved-master-data-changes` | `finance.omd.publish.approved.master.data.changes` |

### 8.2 Authorization, audit, and privacy

- IAM remains the authorization decision point. Every command checks the
  exact permission, explicit scope, field authorization, and approval state
  required for the operation.
- Material changes, publication attempts, denied attempts, and sensitive
  access attempts use the existing audit boundary with actor, authentication
  subject, scope, purpose, aggregate/action, authorization/approval outcome,
  correlation, causation, and before/after fingerprints. Raw values are not
  audit fields.
- If a required audit dependency is unavailable, a material or privileged
  action fails closed and does not commit a state change.
- Operational logs, traces, and metrics contain bounded classification,
  outcome, record type, version, correlation, and support references only.
  They are not authoritative audit evidence.
- Worklist filters, counts, bulk actions, and export projections apply row and
  field permissions before computing or rendering results. Restricted values
  must not leak through counts, empty-state messages, validation errors,
  accessible names, notifications, or generated data.

### 8.3 Consistency, concurrency, and integration

- Each aggregate update uses optimistic concurrency and the existing
  idempotency/fingerprint contract.
- Validation and persistence are atomic within the OMD consistency boundary.
  Cross-context publication is durable and replay-safe through the existing
  transactional outbox/integration mechanism.
- Downstream consumers receive published language and authoritative references;
  propagation may be eventual, while critical commands perform immediate
  authoritative validation.
- A failed downstream delivery does not cause a second OMD mutation or erase
  the accepted source version. Recovery evidence identifies the source record,
  version, correlation, and next action without exposing restricted data.

### 8.4 UI and observability

- `OMD-WS-01` groups work by actionable state and exposes scope, owner, age,
  approval, exception, and next action where authorized.
- `OMD-SCR-01` through `OMD-SCR-05` use the shared identity, state, action,
  lineage, evidence, and sensitivity patterns. Search results link to the
  authoritative record and do not create an alternate mutation surface.
- All status, validation, denied, stale, unavailable, and publication
  announcements are typed, localized through the existing UI contract, and
  safe for assistive technology.
- Telemetry is bounded and separate from authoritative audit evidence; support
  references remain stable without embedding payloads or secrets.

## 9. Dependencies and sequencing

1. Confirm platform command, persistence, idempotency, outbox, OpenAPI, and
   generated-artifact foundations from `EP-PLAT-001`.
2. Confirm IAM permissions, explicit scope, field authorization, and audit
   ports from `EP-IAM-001`.
3. Implement the shared OMD aggregate/application/persistence foundation.
4. Deliver User Stories 1–5 in aggregate order, beginning with Party and Legal
   Entity references needed by customer/vendor profiles and consumers.
5. Deliver User Story 6 after approval and publication integration boundaries
   are available.
6. Qualify downstream handoffs with COA, GL, AP, AR, and Fiscal Period
   Management only through their approved contracts; downstream epics remain
   separate.

## 10. Risks and open decisions

- The repository has approved OMD OpenAPI paths and generated adapters, but no
  `internal/organization` or `web/src/capabilities/master-data` implementation
  yet. Implementation must reconcile generated stubs with the owning module
  without changing the contract casually.
- The exact approval applicability for each aggregate is domain/workflow
  policy, not an OMD-only decision. Do not encode a new approval rule in this
  backlog without an approved source.
- The exact published master-data event names and payload versions must come
  from the existing event/integration contract. This document intentionally
  does not invent event names.
- Dependent-scope impact for fiscal calendars must use an approved read/query
  boundary; it must not become a cross-schema write or an accidental owner of
  fiscal-period state.
- Production qualification, live identity-provider qualification, and full
  audit-chain qualification remain deferred until the relevant epics and
  environments are available.

## 11. Required verification evidence

The following evidence is required when implementation begins; no item below
is claimed as complete by this planning document.

- [ ] Focused OMD domain/application/persistence/API tests for all six stories.
- [ ] HTTP contract tests for permissions, typed problem details, correlation,
  idempotency, optimistic concurrency, safe errors, and no sensitive leakage.
- [ ] Outbox/integration tests for publication transactionality, replay,
  duplicate delivery, and bounded payload projection.
- [ ] UI component tests and Playwright accessibility/semantic tests for
  `OMD-WS-01` and `OMD-SCR-01` through `OMD-SCR-05`.
- [ ] Negative scans proving secrets, tokens, raw bank/tax/personal values,
  unrestricted errors, and raw policy payloads do not appear in logs, traces,
  responses, notifications, or evidence projections.
- [ ] OpenAPI lint, bundle, generated Go/TypeScript drift, migration, and sqlc
  checks pass without unapproved contract changes.
- [ ] `go vet ./...`, supported race checks, repository Go/frontend checks,
  `pnpm run build`, and `git diff --check` pass.
- [ ] Synthetic or deidentified data is used for non-production tests; no live
  Entra, production audit-chain, or production security qualification claim is
  made.

## 12. Definition of done

- [ ] All six child stories satisfy their acceptance criteria and have reviewable
  implementation evidence.
- [ ] The `organization` schema and `internal/organization` module preserve
  bounded-context ownership and approved repository boundaries.
- [ ] Existing OpenAPI routes and generated artifacts remain synchronized; no
  unapproved route, event, schema, or cross-module dependency is introduced.
- [ ] Effective dates, aggregate versions, approval references, idempotency,
  retries, failure behavior, and historical source versions are proven.
- [ ] Authorization, field-level privacy, material-action audit, safe
  observability, accessibility, and export/bulk filtering evidence passes.
- [ ] Downstream handoffs use stable identifiers, versions, and immutable
  snapshots without allowing downstream mutation of OMD records.
- [ ] Roadmap and epic status are changed to complete only after the evidence
  above is successful; this story document does not close the epic by itself.

## 13. Source references

- [Finance Domain Model & Use Cases — DDD Baseline](../../specs/finance_domain_model_ddd.md)
- [Finance Functional PRD](../../specs/prd/01_finance_functional_prd_v1.5.md)
- [Finance Functional Requirements Catalog](../../specs/prd/02_finance_functional_requirements_catalog_v1.5.md)
- [Finance UX Workflow Specification](../../specs/finance_ux_workflow_specification_v1.0.md)
- [Finance Nonfunctional Requirements](../../specs/finance_nonfunctional_requirements_v1.0.md)
- [Finance Delivery Plan](../../specs/finance_delivery_plan_v1.0.md)
- [Solution Architecture Overview](../../specs/system_design/01_solution_architecture_overview_v1.0.md)
- [Backend Module Technical Specifications](../../specs/technical_specifications/01_backend_module_specifications_v1.0.md)
- [Database and Persistence Technical Specifications](../../specs/technical_specifications/03_database_persistence_specifications_v1.0.md)
- [Events, Workers, and Integration Technical Specifications](../../specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md)
- [Testing, Performance, and Recovery Specifications](../../specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md)
- [Existing master-data OpenAPI paths](../../../contracts/openapi/paths/master-data.yaml)
