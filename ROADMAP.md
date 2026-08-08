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
    - [ ] **DLV-PLAT-005 — Shared finance primitives**
      - [ ] **User Story 1 — Implement exact-decimal money and currency primitives**
      - [ ] **User Story 2 — Implement explicit accounting-scope identity**
      - [ ] **User Story 3 — Implement stable identity primitives**
      - [ ] **User Story 4 — Implement aggregate version primitives**
      - [ ] **User Story 5 — Prove serialization and boundary behavior**

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
