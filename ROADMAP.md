# TALLY Roadmap

> Checkboxes show the accepted delivery status. Parent items remain open until all required child scope is complete.

## Status legend

- `[x]` — complete.
- `[ ]` — open or in progress.

## Current delivery checkpoint

- [ ] **M0 — Engineering foundation**
  - [ ] **EP-PLAT-001 — Engineering foundation**
    - [x] **DLV-PLAT-001 — Monorepo foundation**
      - [x] **User Story 1 — Establish the monorepo structure**
      - [x] **User Story 2 — Provide a minimal Go API shell**
      - [x] **User Story 3 — Provide a minimal React application shell**
      - [x] **User Story 4 — Provide shared root commands**
      - [x] **User Story 5 — Document and prove clean-clone reproducibility**
    - [x] **DLV-PLAT-002 — Docker Compose PostgreSQL development environment**
      - [x] **User Story 1 — Define safe local database configuration**
      - [x] **User Story 2 — Start and health-check PostgreSQL**
      - [x] **User Story 3 — Provide root database lifecycle commands**
      - [x] **User Story 4 — Orchestrate migration and deterministic seeding**
      - [x] **User Story 5 — Reset and prove reproducibility**
    - [x] **DLV-PLAT-003 — Goose migrations, pgx and sqlc workflow**
      - [x] **User Story 1 — Establish the Goose migration contract**
      - [x] **User Story 2 — Establish the pgx database foundation**
      - [x] **User Story 3 — Establish the sqlc generation workflow**
      - [x] **User Story 4 — Prove migrations, pgx, and sqlc together**
      - [x] **User Story 5 — Detect persistence drift in CI**
    - [x] **DLV-PLAT-004 — OpenAPI-first REST workflow and generated clients**
      - [x] **User Story 1 — Establish the contract layout and common schemas**
      - [x] **User Story 2 — Validate and bundle the OpenAPI contract**
      - [x] **User Story 3 — Generate and verify Go API artifacts**
      - [x] **User Story 4 — Generate and verify the TypeScript client**
      - [x] **User Story 5 — Detect contract and generated-artifact drift**
    - [x] **DLV-PLAT-005 — Shared finance primitives**
      - [x] **User Story 1 — Implement exact-decimal money and currency primitives**
      - [x] **User Story 2 — Implement explicit accounting-scope identity**
      - [x] **User Story 3 — Implement stable identity primitives**
      - [x] **User Story 4 — Implement aggregate version primitives**
      - [x] **User Story 5 — Prove serialization and boundary behavior**
    - [x] **DLV-PLAT-006 — Request fingerprint and idempotency foundation**
      - [x] **User Story 1 — Define canonical request fingerprinting**
      - [x] **User Story 2 — Define scoped idempotency identity and stored command-result metadata**
      - [x] **User Story 3 — Coordinate established results for identical retries**
      - [x] **User Story 4 — Reject changed content at the platform boundary**
      - [x] **User Story 5 — Prove platform transactional, concurrent, and boundary behavior**
        - Owning finance transaction, authorization/audit integration, cross-process recovery, and finance-level exactly-once proof are follow-up capability scope.
        - Focused evidence passes; Docker-backed integration and pinned SQLC drift verification remain environment-dependent release gates.
    - [x] **DLV-PLAT-007 — PostgreSQL outbox/inbox and worker foundation**
      - [x] **User Story 1 envelope foundation — versioned structural envelope and deterministic payload fingerprinting**
      - [x] **User Story 2 — Persist durable PostgreSQL outbox and inbox records**
      - [x] **User Story 3 — Coordinate transactional publication and consumption**
      - [x] **User Story 4 — Dispatch due outbox work with leases and typed retries**
      - [x] **User Story 5 — Prove worker lifecycle, crash recovery, duplicate delivery, and replay**
      - Semantic payload safety is a separate deferred follow-up; DLV-PLAT-007 is verified by `make outbox-worker-check`.
  - [ ] **EP-UX-001 — Shared UX and design system**
    - [x] **DLV-UX-001 — Shared UX and design system application shell and component abstractions**
      - [x] **User Story 1 — Establish the design-system foundation and application shell**
      - [x] **User Story 2 — Establish routed navigation and accounting-scope context**
      - [x] **User Story 3 — Build record context, lifecycle, money, evidence, and privacy components**
      - [x] **User Story 4 — Build worklist, action, settlement, exception, progress, and result components**
      - [x] **User Story 5 — Build forms, approval/posting panels, validation, confirmation, and conflict interactions**
    - [x] **DLV-UX-002 — Accessibility test harness** (M0 owner-approved scope; witnessed screen-reader and actual 400% browser-zoom reviews deferred to release qualification)
      - [x] **User Story 1 — Establish the automated accessibility harness**
      - [x] **User Story 2 — Verify keyboard operation and focus behavior**
      - [x] **User Story 3 — Establish screen-reader and semantic review coverage** (repository semantic coverage and review procedure complete; witnessed SR-01/SR-02 review deferred)
      - [x] **User Story 4 — Verify visual adaptability and motion preferences** (automated baseline complete; actual 400% browser-zoom review deferred)
      - [x] **User Story 5 — Produce accessibility qualification evidence and defect decisions**
  - [ ] **EP-IAC-001 — Terraform and Azure learning environment**
    - [ ] **DLV-IAC-001 — Terraform modules and local state bootstrap**
      - [x] **User Story 1 — Establish the Terraform repository boundary**
      - [ ] **User Story 2 — Bootstrap protected remote state** (local contract verified; Azure plan and per-environment identity authentication pending)
      - [ ] **User Story 3 — Implement reusable low-cost modules**
      - [ ] **User Story 4 — Define environment profiles safely**
      - [ ] **User Story 5 — Verify plans, policy, drift, and cost controls**
    - [ ] **DLV-IAC-002 — Optional Azure dev/demo deployment**
      - [ ] **User Story 6 — Deploy the optional learning environment**
      - [ ] **User Story 7 — Federate CI/CD without long-lived secrets**
      - [ ] **User Story 8 — Smoke-test deployment readiness**
      - [ ] **User Story 9 — Destroy disposable resources safely**
      - [ ] **User Story 10 — Recover state and document operations**

## Milestone and epic roadmap

### - [ ] M0 — Engineering foundation

**Planning window:** Iterations 1–3 (6 weeks)

**Exit evidence:** Repository, local environment, CI, shared UI, database migration, API and observability foundations are demonstrable.

- [ ] **EP-PLAT-001 — Engineering foundation**
  - Scope: Repository, Go/React applications, Docker Compose, migrations, sqlc, OpenAPI, testing and conventions.
- [ ] **EP-UX-001 — Shared UX and design system**
  - Scope: Tailwind, daisyUI abstractions, routing, forms, tables, accessibility and shared operational surfaces.
- [ ] **EP-IAC-001 — Terraform and Azure learning environment**
  - Scope: Terraform state bootstrap, low-cost Azure modules, budget controls and ephemeral demo deployment.
- [ ] **EP-OPS-001 — Observability and operational foundation**
  - Scope: Structured logs, traces, metrics, dashboards, runbooks and operational evidence.

### - [ ] M1 — Identity and accounting configuration

**Planning window:** Iterations 4–7 (8 weeks)

**Exit evidence:** Authentication, authorization, accounting scope, master data, ledgers, books, chart and accounts are usable.

- [ ] **EP-IAM-001 — Identity and access**
  - Scope: Entra authentication, application permissions, accounting-scope authorization and emergency access.
- [ ] **EP-OMD-001 — Organization and master data**
  - Scope: Legal entities, parties, profiles and fiscal calendars.
- [ ] **EP-COA-001 — COA segment configuration**
  - Scope: Segment definitions, values, combinations and approved segment changes.

### - [ ] M2 — First posted journal vertical slice

**Planning window:** Iterations 8–11 (8 weeks)

**Exit evidence:** A journal can be created, validated, approved when required, posted, queried and reversed end to end.

- [ ] **EP-GL-001 — General Ledger**
  - Scope: Journal validation, approval, posting, reversal, gates and ledger inquiry.

### - [ ] M3 — Approval and period controls

**Planning window:** Iterations 12–15 (8 weeks)

**Exit evidence:** Approval policies, soft/hard close, reopen/reclose and posting-gate recovery are demonstrated.

- [ ] **EP-WFA-001 — Workflow and approvals**
  - Scope: Policies, requests, decisions, delegation, escalation and decision application.
- [ ] **EP-FPM-001 — Fiscal period management**
  - Scope: Soft close, hard close, reopen, reclose and control recovery.

### - [ ] M4 — Receivables and billing

**Planning window:** Iterations 16–21 (12 weeks)

**Exit evidence:** Invoice issue, receipt recording, application, unapplication, credits, write-offs and refund obligations are demonstrated; external refund settlement follows in M5.

- [ ] **EP-INV-001 — Invoicing**
  - Scope: Templates, schedules, generated invoices and AR handoff.
- [ ] **EP-AR-001 — Accounts Receivable**
  - Scope: Invoices, open items, receipts, applications, credits, refunds and adjustments.

### - [ ] M5 — Payables and payment execution

**Planning window:** Iterations 22–29 (16 weeks)

**Exit evidence:** Vendor invoice through payment instruction, settlement, cancellation, return and exception resolution is demonstrable.

- [ ] **EP-AP-001 — Accounts Payable**
  - Scope: Vendor invoices, matching, approval, liabilities and payment requests.
- [ ] **EP-PCM-001 — Payments and cash management**
  - Scope: Batches, instructions, settlements, returns, exceptions and expected incoming settlement.

### - [ ] M6 — Bank and cash reconciliation

**Planning window:** Iterations 30–34 (10 weeks)

**Exit evidence:** Statement import, matching, incoming settlement, excess cash, supplier-refund application and customer chargeback correction are complete.

- [ ] **EP-BFR-001 — Bank feeds and reconciliation**
  - Scope: Connections, imports, matching, unmatching and reconciliation.

### - [ ] M7 — Assets and revenue

**Planning window:** Iterations 35–41 (14 weeks)

**Exit evidence:** Fixed-asset lifecycle/disposal and revenue-contract recognition/modification workflows are complete.

- [ ] **EP-FA-001 — Fixed Assets**
  - Scope: Capitalization, depreciation, impairment, transfer, split and disposal.
- [ ] **EP-REV-001 — Revenue Recognition**
  - Scope: Contracts, obligations, profiles, schedules and modifications.

### - [ ] M8 — Currency, intercompany and reporting

**Planning window:** Iterations 42–48 (14 weeks)

**Exit evidence:** FX, revaluation, translation, intercompany settlement, consolidation and statements are demonstrated.

- [ ] **EP-FX-001 — Multi-Currency**
  - Scope: Rates, realized FX, revaluation and translation.
- [ ] **EP-IC-001 — Intercompany**
  - Scope: Agreements, transactions, matching, settlement and eliminations.
- [ ] **EP-RPT-001 — Financial Reporting**
  - Scope: Definitions, statements, consolidation, lineage and publication.

### - [ ] M9 — Tax, payroll, audit and qualification

**Planning window:** Iterations 49–54 (12 weeks)

**Exit evidence:** Tax/payroll correction flows, audit verification and full security, accessibility, recovery and performance qualification pass.

- [ ] **EP-TAX-001 — Tax Filing**
  - Scope: Configurations, returns, submissions, amendments, adjustments and payments.
- [ ] **EP-PAYR-001 — Payroll**
  - Scope: Profiles, runs, corrections, off-cycle processing, failed payments and filing amendments.
- [ ] **EP-AUD-001 — Audit Integrity**
  - Scope: Evidence ingestion, verification, credential rotation, incidents and proof access.
- [ ] **EP-QUAL-001 — Full-system qualification**
  - Scope: Security, privacy, accessibility, capacity, performance, recovery and release evidence.

## Completion rules

1. A user story is complete only when every acceptance criterion and required evidence passes.
2. A delivery item is complete only when all of its user stories and exit evidence pass.
3. An epic is complete only when all required delivery items and applicable quality gates pass.
4. A milestone is complete only when every required epic, workflow demonstration, and minimum gate passes.
5. Partial percentage completion never marks a milestone complete.
