# Finance Platform UX and Workflow Specification

| Document-control field | Value |
|---|---|
| Version | 1.0 |
| Baseline date | 2026-07-24 |
| Status | Consistency-verified UX/workflow baseline |
| Source functional baseline | Finance Platform Functional PRD v1.5 |
| Source requirements baseline | Functional Requirements Catalog v1.5 |
| Source acceptance baseline | Functional Traceability and Acceptance v1.5 |
| Source domain baseline | Finance Domain Model & Use Cases — DDD Baseline v3.1 |
| Source DDD checkpoint | DDD-3.1-2026-07-24 |
| Source DDD verified-content SHA-256 | `a9d437d23656c36d340afb3a5a31c93a23e574f53db186483a9edfdf32d3e652` |
| Document owner | Finance Product Design |
| Intended audience | Product, Finance SMEs, UX, Accessibility, QA, Architecture, Engineering, Security, and Operations |

> **Purpose:** Define how authorized users navigate, understand, and complete the functional behavior established by the Finance Functional PRD v1.5, while preserving DDD ownership, lifecycle, accounting, correction, and evidence semantics.

> **Boundary:** This specification defines information architecture, workspaces, screens, interaction rules, user journeys, messages, and UX acceptance criteria. It does not prescribe visual styling, APIs, persistence, service design, middleware, infrastructure, deployment, performance targets, or implementation technology.

<a id="table-of-contents"></a>
## Table of Contents
- [1. Purpose, Scope, and Source Authority](#section-1)
- [2. UX Principles](#section-2)
- [3. Users and Role Contexts](#section-3)
- [4. Information Architecture](#section-4)
- [5. Common Interaction Model](#section-5)
- [6. Shared Screen and Component Catalog](#section-6)
- [7. Capability Workspace Specifications](#section-7)
- [8. Detailed Workflow Specifications](#section-8)
- [9. Notifications, Messages, and Recovery](#section-9)
- [10. Accessibility, Privacy, and Evidence](#section-10)
- [11. UX Acceptance and Traceability](#section-11)
- [12. Non-Goals and Dependencies](#section-12)
- [13. Verification Checkpoint](#section-13)

<a id="section-1"></a>
## 1. Purpose, Scope, and Source Authority

This document is the UX interpretation of the verified functional baseline. It does not redefine business ownership, invariants, state transitions, commands, events, or accounting equations. When wording conflicts, the authority order is: DDD v3.1, Functional PRD v1.5, Functional Requirements Catalog v1.5, Functional Traceability and Acceptance v1.5, then this UX specification.

The specification covers:
- Role-based navigation and workspaces for all 19 product capabilities.
- Shared interaction patterns for scope, state, approval, posting, settlement, correction, evidence, conflict, and recovery.
- A screen catalog and state-aware action model.
- Detailed UX journeys for all 22 functional workflows (`WF-6.1`–`WF-6.7` and `WF-7.1`–`WF-7.15`).
- UX acceptance criteria and exact `FR-*`/`GFR-*` traceability.

<a id="section-2"></a>
## 2. UX Principles

| Principle | UX requirement |
|---|---|
| Ownership before convenience | Every record and action identifies the authoritative capability. Cross-capability views may aggregate status, but they do not imply shared mutation authority. |
| State explains action | Every detail view shows current state, permitted actions, blocked actions, blocking reasons, owner, and recovery/correction path. |
| Established facts remain visible | Posted, accepted, settled, reconciled, or otherwise established facts are never visually replaced by their corrections; reversal, return, amendment, replacement, or compensation appears as linked lineage. |
| One financial meaning per amount | Transaction, functional, and presentation amounts are labeled distinctly; rate evidence and sign conventions are visible where applicable. |
| Intermediate outcomes are first-class | Approval pending, posting pending, partial settlement, owner acknowledgement, reconciliation, exception, and reversal states remain visible rather than collapsing into a generic success/failure label. |
| Safe repetition and conflict clarity | A safe repeat returns the established result. Changed content under the same identity and concurrent state changes produce typed conflicts with a deliberate next action. |
| Controls are explained | Authorization and segregation denials identify the applicable scope/action dimension without exposing restricted policy or sensitive data. |
| Evidence is navigable | Users can move from source fact to approval, posting, settlement, correction, statement, and audit evidence according to access rights. |
| Recovery is explicit | Dependency unavailability, domain rejection, validation failure, and business exception use distinct messages and resolution paths. |
| Sensitive detail is minimized | Payroll, tax, bank, and personal details are shown only where needed and only to authorized roles. |

<a id="section-3"></a>
## 3. Users and Role Contexts

A user may hold multiple roles, but each action is evaluated in the current accounting/business scope and against applicable segregation rules. Workspaces show only records and actions the user is permitted to access.

| Capability | Primary role contexts | Default UX focus |
|---|---|---|
| Organization & Master Data | Master Data Steward; Finance Administrator. | Maintain legal entities, registrations, ownership interests, parties, customer/vendor profiles, and fiscal calendars, preserving effective dates where the domain defines them. |
| General Ledger (GL) | Accountant; GL Manager; Controller; authorized subledger actor. | Prepare, approve, post, reverse, and review journals and authoritative posting-gate outcomes. |
| Accounts Payable (AP) | AP Specialist; AP Manager; Invoice Approver. | Register, match, approve, dispute, void, pay, and correct vendor liabilities. |
| Accounts Receivable (AR) | Billing Specialist; Cash Applications Specialist; Collections Specialist; AR Manager. | Issue receivables, record and apply receipts, manage credits/refunds, and resolve customer-balance exceptions. |
| Payroll | Payroll Specialist; Payroll Manager; Payroll Approver. | Calculate, approve, post, correct, and reconcile payroll and payroll-related obligations. |
| Invoicing | Billing Operations Specialist; Billing Manager. | Configure billing, calculate charges, generate invoices, and finalize billing handoff. |
| Payments & Cash Management | Treasury Specialist; Payment Operator; Cash Manager; Payment Approver. | Approve and execute outgoing payments, record incoming settlements and returns, and reconcile cash outcomes. |
| Financial Reporting | Financial Reporting Accountant; Consolidation Manager; Controller. | Run consolidation, apply translation results, and publish controlled financial statements. |
| Multi-Entity / Intercompany | Intercompany Accountant; Counterparty Accountant; Residual Approver. | Match, approve, settle, and eliminate intercompany activity. |
| Revenue Recognition | Revenue Accountant; Revenue Manager; Contract Approver. | Assess contracts, approve schedules, publish accounting profiles, modify contracts, and run recognition. |
| Fixed Assets | Fixed Asset Accountant; Asset Manager; Disposal Approver. | Capitalize, depreciate, impair, transfer, dispose, and reconcile asset settlement obligations. |
| Multi-Currency | Treasury Accountant; FX Administrator; Consolidation Accountant. | Publish rate sets, run revaluation and translation, and retain conversion evidence. |
| Fiscal Period Management | Finance Manager; Controller; Close Coordinator. | Control soft close, hard close, reopen, reclose, exceptions, and period-control recovery. |
| COA Segment Accounting | Chart-of-Accounts Administrator; Finance Approver. | Maintain segment definitions and values, validate combinations, and govern effective-dated changes. |
| Bank Feeds & Reconciliation | Bank Reconciliation Specialist; Cash Accountant. | Import statements, propose/confirm/reverse matches, and complete reconciliation. |
| Tax Filing | Tax Specialist; Tax Manager; Tax Approver. | Determine tax, prepare/submit returns and amendments, post return-level adjustments, and settle tax obligations. |
| Workflow & Approvals | Approver; Delegator; Approval Administrator. | Create, decide, delegate, and escalate approval requests while enforcing policy and segregation of duties. |
| Identity & Access | Security Administrator; Access Approver; Auditor. | Administer users, roles, access policies, segregation rules, and controlled emergency access. |
| Audit Integrity | Internal Auditor; Compliance Officer; Integrity Administrator. | Append audit evidence, create seals, verify proof, rotate verification credentials, and manage integrity incidents. |

External providers and schedulers appear as evidence sources or triggers. The interface identifies their evidence and status but never presents them as the authoritative owner of product business state.

<a id="section-4"></a>
## 4. Information Architecture

### 4.1 Global navigation

| Navigation area | Purpose | Visibility rule |
|---|---|---|
| Home | Role-based summary of assigned work, approvals, exceptions, close tasks, overdue obligations, and recent outcomes. | Visible to authenticated users; content is permission-filtered. |
| Work | Capability worklists and end-to-end process workspaces. | Shows only capabilities and scopes available to the user. |
| Records | Search and direct access to authoritative business records. | Filtered by record type, scope, sensitivity, and action rights. |
| Approvals | Assigned, delegated, escalated, and completed approval decisions. | Visible to approvers, delegators, administrators, and auditors as permitted. |
| Exceptions | Validation, posting, settlement, reconciliation, period-control, event, and integrity exceptions. | Visible to the owning/resolving role and authorized oversight roles. |
| Reports | Operational reports, financial statements, exception aging, and evidence exports. | Filtered by reporting and data-sensitivity permissions. |
| Administration | Master data, configuration, access, policies, connections, and calendars. | Visible only to the applicable administrator roles. |
| Audit | Audit history, proof verification, seals, incidents, and evidence search. | Visible only to authorized audit/compliance/integrity roles. |

### 4.2 Accounting and business scope context

- The persistent scope context shows tenant, legal entity, ledger, accounting book, functional currency, and period when applicable.
- Scope changes refresh the worklist before an action can be submitted; unsaved changes require explicit discard or return.
- Cross-entity/consolidation work uses participant or consolidation scope rather than implying one ambient legal entity.
- Every record detail shows its authoritative scope. Cross-scope actions show all participant scopes before confirmation.

### 4.3 Record-detail shell

Every authoritative record detail uses the following semantic regions:
1. **Identity and scope:** business identifier, record type, accounting/business scope, source reference, and current version.
2. **State and ownership:** lifecycle state, authoritative capability, responsible owner, and last material change.
3. **Primary facts:** amounts, currencies, dates, parties, configuration versions, and source evidence.
4. **Action bar:** permitted actions; blocked actions remain discoverable with the blocking reason where disclosure is allowed.
5. **Process panels:** approval, posting, payment/settlement, reconciliation, close/reopen, or filing panels as applicable.
6. **Lineage timeline:** source, decisions, established effects, corrections, replacements, and evidence.
7. **Audit/evidence:** actor, time, authorization, fingerprints, exports, and legal-hold status according to access.

<a id="section-5"></a>
## 5. Common Interaction Model

### 5.1 Common state and action rules

| Rule | Required UX behavior | Traceability |
|---|---|---|
| Scope and authorization | Before submission, show the selected scope and the action being authorized. On denial, identify the applicable dimension or prohibited duty and the next permitted action. | `GFR-001`, `GFR-002`, `GFR-003` |
| Approval | Show requested, pending, delegated, escalated, decided, applied, rejected, expired, and invalidated states separately. A recorded decision is not displayed as applied until the business owner revalidates and applies it. | `GFR-004`, `GFR-009`, `GFR-013` |
| Established facts and correction | Disable destructive editing after establishment. Offer only allowed reversal, amendment, return, unapplication, replacement, or compensation actions and preserve the original fact. | `GFR-005`, `GFR-013` |
| Safe repeat | When the same identity/fingerprint is repeated, show the established in-progress or terminal result and do not ask the user to recreate the action. | `GFR-006` |
| Identity-content conflict | When identity is reused with changed content, show a conflict, the established fingerprint/result reference, and a requirement to create a new business action. | `GFR-007` |
| Concurrency conflict | Show expected versus current version/state, current owner, whether retry is safe, and a refresh/compare/re-enter path. Never silently merge. | `GFR-008`, `GFR-009` |
| Intermediate states | Show cross-capability progress by owner and distinguish pending, partial, failed, acknowledged, reconciled, reversed, and terminal outcomes. | `GFR-009`, `GFR-012` |
| Currency | Label transaction, functional, and presentation amounts; expose rate-set/type/date evidence and functional-only adjustment lines where applicable. | `GFR-010` |
| Accounting ownership | Show which capability owns each accounting effect and prevent the same business effect from appearing as two user-submittable postings. | `GFR-011` |
| Audit/evidence | Provide access-controlled lineage and exports for material actions, decisions, postings, settlements, close/reopen, and integrity verification. | `GFR-014`, `GFR-020` |
| Privacy and legal hold | Mask/minimize sensitive data, preserve legal-held records, and explain restricted access without exposing the protected value. | `GFR-015`, `GFR-018` |
| Search and worklists | Provide applicable scope, state, owner, date, amount, currency, exception, and approval filters with saved views. | `GFR-016` |
| Dependency versus domain outcome | Use distinct messages and status labels for unavailable dependency, validation failure, authorization denial, business rejection, and established exception. | `GFR-017` |
| Effective-dated history | Show the configuration/rule version used by historical transactions and do not relabel history with the current rule. | `GFR-019` |
| PRD action naming | Do not label PRD-only functional actions as DDD commands/events unless the DDD baseline changes. | `GFR-021` |
| Traceability | Every detailed workflow and screen family links to exact requirement IDs and source workflow/acceptance groups. | `GFR-022` |

### 5.2 Confirmation rules

Confirmation is required only when an action establishes a material business outcome, starts an external obligation, changes a period-control owner/gate, publishes a statement/profile/rate, or creates a correction. The confirmation summarizes scope, record/version, amount/currency, intended state change, accounting owner, approvals, and irreversible lineage effects. Routine navigation, validation, filtering, and established-result lookup do not require confirmation.

### 5.3 Error and recovery presentation

| Category | Presentation | Allowed next actions |
|---|---|---|
| Field validation | Inline at the field plus a summary linked to each affected field/line. | Correct and revalidate. |
| Business-rule rejection | Page-level typed result with authoritative rule, current state/version, and evidence. | Return, correct source facts, request approval, or choose an allowed alternative. |
| Authorization/segregation denial | Typed denial with applicable dimension or duty conflict. | Change scope, request access/approval, delegate where permitted, or abandon. |
| Concurrency conflict | Comparison of reviewed snapshot and current state. | Refresh, deliberate safe retry, create a new action, or abandon. |
| Safe duplicate | Established result card. | Open result; no resubmission. |
| Identity-content conflict | Conflict card showing reused identity and changed fingerprint. | Create a new business identity/action. |
| Dependency unavailable | Availability notice without implying business rejection or success. | Retry status lookup, save draft where allowed, or return to worklist. |
| Managed exception | Persistent exception record with owner, aging, evidence, and allowed resolutions. | Apply one authorized resolution or escalate. |
| Ambiguous outcome | Result lookup state keyed by established business identity. | Wait for/perform authoritative result reconciliation before retry. |

<a id="section-6"></a>
## 6. Shared Screen and Component Catalog

| Component ID | Component | Required behavior |
|---|---|---|
| `CMP-001` | Accounting Scope Selector | Shows current accounting/business scope and prevents submission against a stale or unintended scope. |
| `CMP-002` | Record Identity Header | Shows record ID/type, authoritative capability, scope, state, version, source, owner, and sensitivity label. |
| `CMP-003` | State-Aware Action Bar | Shows permitted and blocked actions with reason; actions refresh after every material outcome. |
| `CMP-004` | Lifecycle Timeline | Shows established state transitions, actors, timestamps, decisions, and linked correction/replacement paths. |
| `CMP-005` | Approval Panel | Shows request, policy/version, subject snapshot, steps, decisions, delegation/escalation, and decision-application state. |
| `CMP-006` | Posting Panel | Shows posting identity, purpose, request/result, journal references, period/gate evidence, failure, retry, reversal, and reconciliation. |
| `CMP-007` | Money and Currency Panel | Shows labeled transaction/functional/presentation amounts, rounding, rate evidence, and signed gain/loss convention. |
| `CMP-008` | Settlement and Reconciliation Panel | Shows gross, returned/reversed/cancelled/remaining/net balances, owner acknowledgement, reconciliation, and exceptions. |
| `CMP-009` | Correction Lineage Panel | Links original fact, reversal, amendment, return, unapplication, replacement, compensation, and supersession. |
| `CMP-010` | Validation Summary | Groups field, line, business-rule, authorization, dependency, and conflict results with navigation to the cause. |
| `CMP-011` | Exception Resolution Panel | Shows exception type, amount/scope, owner, evidence, aging, authorized outcomes, and resulting state. |
| `CMP-012` | Version Conflict Dialog | Shows expected/current version and state, changed fields, current owner, established-result lookup, and safe retry status. |
| `CMP-013` | Evidence Drawer | Shows access-controlled source, approval, posting, provider/authority, reconciliation, close, statement, and audit evidence. |
| `CMP-014` | Worklist and Saved Filters | Supports applicable filters, columns, assignment, aging, bulk selection where allowed, exports, and saved views. |
| `CMP-015` | Sensitive Data Guard | Masks restricted values, labels restricted sections, prevents unauthorized export, and records denied access. |
| `CMP-016` | Legal Hold Indicator | Shows hold status and prevents destructive retention actions without changing business lifecycle actions. |
| `CMP-017` | Result Lookup | Finds the established in-progress or terminal result by business identity/fingerprint after duplicate or ambiguous submission. |
| `CMP-018` | Process Progress Panel | Shows cross-capability steps, responsible owner, current/pending/failed/reconciled states, and links to authoritative records. |

### 6.1 Shared operational surfaces

These surfaces do not own business records. They route the user to the authoritative receiving or owning capability.

| Screen ID | Surface | Required behavior |
|---|---|---|
| `XCT-WS-01` | Cross-context event exception worklist | A shared operational view filtered to the receiving capability; it shows event outcomes and links resolution to the authoritative receiving record without transferring ownership. |
| `XCT-SCR-01` | Cross-context event outcome detail | Shows event identity, contract version, scope, source version, receiving outcome, prerequisites, local effect, evidence, and authorized resolution in the receiving capability. |
| `CON-SCR-01` | Concurrency conflict and safe-retry view | Shows attempted action, expected/current version and state, changed values, active owner/process epoch, established-result lookup, and whether deliberate retry is permitted. |

<a id="section-7"></a>
## 7. Capability Workspace Specifications

<a id="capability-omd"></a>
### 7.1 Organization & Master Data

- **Primary users:** Master Data Steward; Finance Administrator.
- **Purpose:** Legal entities, parties, fiscal calendars, registrations, organization hierarchy.
- **Authoritative records:** LegalEntity, Party, CustomerProfile, VendorProfile, FiscalCalendar.
- **Functional requirement coverage:** `FR-OMD-001`, `FR-OMD-002`, `FR-OMD-003`, `FR-OMD-004`, `FR-OMD-005`, `FR-OMD-006`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `OMD-WS-01` | Master-data worklist | Search, review, and route legal-entity, party, profile, calendar, and publication work. |
| `OMD-SCR-01` | Legal-entity record | Maintain identity, registrations, addresses, ownership interests, and effective dates. |
| `OMD-SCR-02` | Party record | Maintain party identity, status, contacts, addresses, classifications, and bank-detail references. |
| `OMD-SCR-03` | Customer and vendor profiles | Maintain customer terms and vendor payment/remittance attributes. |
| `OMD-SCR-04` | Fiscal-calendar editor | Define calendar patterns and periods and review dependent-scope impact. |
| `OMD-SCR-05` | Publication review | Review approvals, effective dates where defined, validation results, and dependent-capability availability. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-gl"></a>
### 7.2 General Ledger (GL)

- **Primary users:** Accountant; GL Manager; Controller; authorized subledger actor.
- **Purpose:** Ledger and accounting-book configuration, chart of accounts, journal validation, posting, and authoritative posting admission.
- **Authoritative records:** Ledger, AccountingBook, ChartOfAccounts, Account, JournalEntry, PeriodPostingGate.
- **Functional requirement coverage:** `FR-GL-001`, `FR-GL-002`, `FR-GL-003`, `FR-GL-004`, `FR-GL-005`, `FR-GL-006`, `FR-GL-007`, `FR-GL-008`, `FR-GL-009`, `FR-GL-010`, `FR-GL-011`, `FR-GL-012`, `FR-GL-013`, `FR-GL-014`, `FR-GL-015`, `FR-GL-016`, `FR-GL-017`, `FR-GL-018`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `GL-WS-01` | Journal workbench | Create, import, validate, approve, post, search, and reverse journals. |
| `GL-SCR-01` | Journal detail | Show scope, source, lines, balancing, approval, posting state, and correction lineage. |
| `GL-SCR-02` | Posting result and conflict view | Show established results, rejections, approval pending, idempotency conflicts, and safe next actions. |
| `GL-SCR-03` | Posting-gate monitor | Show gate mode, owner, version, admission summary, barrier position, expiry, and allowed posting purposes. |
| `GL-SCR-04` | Ledger and accounting-book configuration | Maintain ledgers, books, accounting basis, currencies, calendars, and posting policies. |
| `GL-SCR-05` | Chart and account configuration | Maintain charts, accounts, restrictions, currency policy, and reporting mappings. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-ap"></a>
### 7.3 Accounts Payable (AP)

- **Primary users:** AP Specialist; AP Manager; Invoice Approver.
- **Purpose:** Vendor invoices, matching against procurement snapshots, liabilities, payment requests.
- **Authoritative records:** VendorInvoice, PaymentRequest.
- **Functional requirement coverage:** `FR-AP-001`, `FR-AP-002`, `FR-AP-003`, `FR-AP-004`, `FR-AP-005`, `FR-AP-006`, `FR-AP-007`, `FR-AP-008`, `FR-AP-009`, `FR-AP-010`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `AP-WS-01` | Vendor-invoice worklist | Manage received, duplicate-suspected, unmatched, approval, dispute, posting, and payment states. |
| `AP-SCR-01` | Vendor-invoice detail | Show invoice, lines, tax, snapshots, duplicate evidence, approval, posting, balance, and correction lineage. |
| `AP-SCR-02` | Match review | Compare invoice lines with immutable purchase-order and receipt snapshots and tolerance results. |
| `AP-SCR-03` | Dispute and void panel | Record reason, evidence, affected amount, allowed state transition, and correction outcome. |
| `AP-SCR-04` | Payment-request builder | Allocate approved invoice obligations to a payment request and validate totals/currency. |
| `AP-SCR-05` | Settlement and return view | Show incoming applications, payment results, returns, remaining liability, and reconciliation. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-ar"></a>
### 7.4 Accounts Receivable (AR)

- **Primary users:** Billing Specialist; Cash Applications Specialist; Collections Specialist; AR Manager.
- **Purpose:** Customer invoices, receivable open items, receipts, credit notes, refunds, and collections balances.
- **Authoritative records:** CustomerInvoice, ReceivableOpenItem, CustomerReceipt, CreditNote, CustomerRefundRequest.
- **Functional requirement coverage:** `FR-AR-001`, `FR-AR-002`, `FR-AR-003`, `FR-AR-004`, `FR-AR-005`, `FR-AR-006`, `FR-AR-007`, `FR-AR-008`, `FR-AR-009`, `FR-AR-010`, `FR-AR-011`, `FR-AR-012`, `FR-AR-013`, `FR-AR-014`, `FR-AR-015`, `FR-AR-016`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `AR-WS-01` | Receivables worklist | Manage invoices, open items, receipts, unapplied cash, credits, refunds, chargebacks, and write-offs. |
| `AR-SCR-01` | Customer invoice and open-item detail | Show invoice, open amount, applications, adjustments, aging, and lineage. |
| `AR-SCR-02` | Receipt workbench | Record receipts and show receipt-accounting, applied/unapplied balances, and bank evidence. |
| `AR-SCR-03` | Receipt allocation workspace | Allocate one receipt to one or more open items with real-time availability and validation. |
| `AR-SCR-04` | Application and unapplication detail | Show immutable application facts, batch accounting, adjustments, rollback evidence, and journal results. |
| `AR-SCR-05` | Credit and customer-adjustment workspace | Create credit notes, overpayment resolutions, chargebacks, and write-offs. |
| `AR-SCR-06` | Customer-refund workspace | Manage approval, clearing legs, payment instructions, settlement, cancellation, return correction, and replacement. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-payr"></a>
### 7.5 Payroll

- **Primary users:** Payroll Specialist; Payroll Manager; Payroll Approver.
- **Purpose:** Payroll calculation, liabilities, deductions, payroll posting requests.
- **Authoritative records:** PayrollRun, EmployeePayrollProfile, PayrollTaxFiling.
- **Functional requirement coverage:** `FR-PAYR-001`, `FR-PAYR-002`, `FR-PAYR-003`, `FR-PAYR-004`, `FR-PAYR-005`, `FR-PAYR-006`, `FR-PAYR-007`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `PAYR-WS-01` | Payroll-run worklist | Manage regular, off-cycle, correction, approval, posting, payment, and settlement states. |
| `PAYR-SCR-01` | Payroll-run detail | Show pay-group totals, employee-level restricted results, approval, posting, and payment status. |
| `PAYR-SCR-02` | Payroll variance and validation review | Review gross, deductions, net, tax, exceptions, and changes from prior runs. |
| `PAYR-SCR-03` | Payroll correction workspace | Create linked correction or off-cycle runs without overwriting finalized facts. |
| `PAYR-SCR-04` | Employee payroll profile | Maintain pay group, tax profile reference, and payment method reference under restricted access. |
| `PAYR-SCR-05` | Payroll tax-filing record | Maintain payroll-source filing records and links to statutory amendments. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-inv"></a>
### 7.6 Invoicing

- **Primary users:** Billing Operations Specialist; Billing Manager.
- **Purpose:** Billing schedules, usage and charge calculation, invoice generation.
- **Authoritative records:** InvoiceTemplate, BillingSchedule, GeneratedInvoice.
- **Functional requirement coverage:** `FR-INV-001`, `FR-INV-002`, `FR-INV-003`, `FR-INV-004`, `FR-INV-005`, `FR-INV-006`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `INV-WS-01` | Billing operations worklist | Manage templates, schedules, generated invoices, recalculation, finalization, and cancellation. |
| `INV-SCR-01` | Invoice-template editor | Maintain billing-frequency, charge-rule, tax-category, and payment-method definitions. |
| `INV-SCR-02` | Billing-schedule editor | Maintain customer/contract schedule, charges, next billing date, and schedule status. |
| `INV-SCR-03` | Invoice-generation run | Show source inputs, generated records, warnings, failures, and rerun eligibility. |
| `INV-SCR-04` | Generated-invoice preview | Review lines, money, taxes, source version, and finalization blockers. |
| `INV-SCR-05` | Finalization and AR handoff | Confirm finalization, show immutable invoice version, and track AR acceptance or rejection. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-pcm"></a>
### 7.7 Payments & Cash Management

- **Primary users:** Treasury Specialist; Payment Operator; Cash Manager; Payment Approver.
- **Purpose:** Payment batches, bank accounts, outgoing payment execution and returns, expected non-customer incoming settlements, observed settlement receipts, unallocated incoming cash, cash posting, and reconciliation status.
- **Authoritative records:** BankAccount, PaymentBatch, PaymentInstruction, PaymentReturn, ExpectedIncomingSettlement, SettlementReceipt, UnallocatedIncomingSettlement.
- **Functional requirement coverage:** `FR-PCM-001`, `FR-PCM-002`, `FR-PCM-003`, `FR-PCM-004`, `FR-PCM-005`, `FR-PCM-006`, `FR-PCM-007`, `FR-PCM-008`, `FR-PCM-009`, `FR-PCM-010`, `FR-PCM-011`, `FR-PCM-012`, `FR-PCM-013`, `FR-PCM-014`, `FR-PCM-015`, `FR-PCM-016`, `FR-PCM-017`, `FR-PCM-018`, `FR-PCM-019`, `FR-PCM-020`, `FR-PCM-021`, `FR-PCM-022`, `FR-PCM-023`, `FR-PCM-024`, `FR-PCM-025`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `PCM-WS-01` | Treasury worklist | Manage payment batches, instructions, returns, incoming expectations, receipts, exceptions, and unallocated cash. |
| `PCM-SCR-01` | Payment-batch detail | Show control totals, instructions, approvals, submission state, outcome, and cancellation eligibility. |
| `PCM-SCR-02` | Payment-instruction detail | Show obligation, beneficiary, attempts, settlement balances, cancellation, exception decision, and returns. |
| `PCM-SCR-03` | Payment-return detail | Show observation, reservation, posting, owner acknowledgement, exception resolution, reversal, and reconciliation. |
| `PCM-SCR-04` | Incoming-settlement expectation | Show expected, received, reconciled, remaining, expiry, receipt allocations, and resolutions. |
| `PCM-SCR-05` | Settlement-receipt detail | Show bank allocation, validation, cash posting, owner application, reconciliation, reversal, and evidence. |
| `PCM-SCR-06` | Unallocated incoming cash | Show suspense posting, candidate expectation, exception reason, resolution, and authoritative posting result. |
| `PCM-SCR-07` | Bank-account record | Maintain masked identity, currency, bank identifier, and account status under scoped authorization. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-rpt"></a>
### 7.8 Financial Reporting

- **Primary users:** Financial Reporting Accountant; Consolidation Manager; Controller.
- **Purpose:** Financial statements, consolidation, report definitions and mappings.
- **Authoritative records:** ReportDefinition, ConsolidationRun, FinancialStatement.
- **Functional requirement coverage:** `FR-RPT-001`, `FR-RPT-002`, `FR-RPT-003`, `FR-RPT-004`, `FR-RPT-005`, `FR-RPT-006`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `RPT-WS-01` | Reporting and consolidation worklist | Manage report definitions, consolidation runs, statement generation, review, and publication. |
| `RPT-SCR-01` | Report-definition editor | Maintain statement structure, mappings, calculations, period, and presentation currency. |
| `RPT-SCR-02` | Consolidation run | Show participant scopes, source watermarks, translation, eliminations, exceptions, approval, and rerun lineage. |
| `RPT-SCR-03` | Financial statement review | Show definition version, source watermarks, statement lines, validation, publication state, and supersession. |
| `RPT-SCR-04` | Consolidation exception workspace | Resolve missing data, ownership, translation, elimination, and approval blockers. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-ic"></a>
### 7.9 Multi-Entity / Intercompany

- **Primary users:** Intercompany Accountant; Counterparty Accountant; Residual Approver.
- **Purpose:** Intercompany agreements, reconciliation, netting, settlement and elimination instructions.
- **Authoritative records:** IntercompanyAgreement, IntercompanyTransaction, SettlementRun, EliminationRun.
- **Functional requirement coverage:** `FR-IC-001`, `FR-IC-002`, `FR-IC-003`, `FR-IC-004`, `FR-IC-005`, `FR-IC-006`, `FR-IC-007`, `FR-IC-008`, `FR-IC-009`, `FR-IC-010`, `FR-IC-011`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `IC-WS-01` | Intercompany worklist | Manage agreements, transactions, matching, residuals, settlement, returns, and eliminations. |
| `IC-SCR-01` | Intercompany agreement | Maintain participant scopes, currency, rate policy, tolerance, and effective period. |
| `IC-SCR-02` | Intercompany transaction | Record reciprocal references, amount, currency, counterparty, and status. |
| `IC-SCR-03` | Settlement-run workspace | Select items, reserve, match, net, approve residuals, create instructions, and complete. |
| `IC-SCR-04` | Match and residual review | Compare reciprocal items, show differences, exceptions, and required approval. |
| `IC-SCR-05` | Elimination run | Review versioned elimination instructions, exceptions, approvals, and reporting result. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-rev"></a>
### 7.10 Revenue Recognition

- **Primary users:** Revenue Accountant; Revenue Manager; Contract Approver.
- **Purpose:** ASC 606 or IFRS 15 assessment, allocation, contract balances and recognition schedules.
- **Authoritative records:** RevenueContract, RevenueSchedule, ContractModification.
- **Functional requirement coverage:** `FR-REV-001`, `FR-REV-002`, `FR-REV-003`, `FR-REV-004`, `FR-REV-005`, `FR-REV-006`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `REV-WS-01` | Revenue-accounting worklist | Manage contract assessments, schedules, profiles, modifications, recognition, and exceptions. |
| `REV-SCR-01` | Revenue-contract assessment | Review source contract, collectibility, combination, promises, obligations, and transaction price. |
| `REV-SCR-02` | Performance-obligation and allocation workspace | Define obligations, allocation evidence, contract balances, and policy version. |
| `REV-SCR-03` | Revenue-schedule review | Review schedule versions, milestones, recognition dates, amounts, and approval state. |
| `REV-SCR-04` | Revenue accounting profile | Review immutable published classification used by AR and Invoicing. |
| `REV-SCR-05` | Contract modification workspace | Classify and approve modification, renewal, cancellation, refund, variable consideration, and catch-up. |
| `REV-SCR-06` | Recognition run | Preview scheduled recognition, validate period/rates, post, reconcile, and rerun where allowed. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-fa"></a>
### 7.11 Fixed Assets

- **Primary users:** Fixed Asset Accountant; Asset Manager; Disposal Approver.
- **Purpose:** Asset acquisition, capitalization, depreciation, impairment, transfer and disposal.
- **Authoritative records:** FixedAsset, DepreciationRun, ImpairmentAssessment, AssetDisposal.
- **Functional requirement coverage:** `FR-FA-001`, `FR-FA-002`, `FR-FA-003`, `FR-FA-004`, `FR-FA-005`, `FR-FA-006`, `FR-FA-007`, `FR-FA-008`, `FR-FA-009`, `FR-FA-010`, `FR-FA-011`, `FR-FA-012`, `FR-FA-013`, `FR-FA-014`, `FR-FA-015`, `FR-FA-016`, `FR-FA-017`, `FR-FA-018`, `FR-FA-019`, `FR-FA-020`, `FR-FA-021`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `FA-WS-01` | Asset register and worklist | Manage capitalization, depreciation, impairment, transfer, split, disposal, settlement, and correction. |
| `FA-SCR-01` | Asset detail | Show cost, components, depreciation, impairment, carrying amount, location, status, and lineage. |
| `FA-SCR-02` | Capitalization and acquisition clearing | Create asset cost and reconcile supplier-liability clearing without taking AP ownership. |
| `FA-SCR-03` | Depreciation run | Review policy, period, asset calculations, exceptions, approval, and posting. |
| `FA-SCR-04` | Impairment assessment | Record recoverable amount, impairment, evidence, approval, and posting result. |
| `FA-SCR-05` | Transfer and split workspace | Select assets/components, effective date, destination/classification, allocation, and validation. |
| `FA-SCR-06` | Asset-disposal workspace | Review carrying amount, treatment, proceeds, costs, required posting legs, approval, and settlement. |
| `FA-SCR-07` | Disposal settlement and correction | Track proceeds, costs, returns/reversals, failures, replacement, compensation, and posted-disposal correction. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-fx"></a>
### 7.12 Multi-Currency

- **Primary users:** Treasury Accountant; FX Administrator; Consolidation Accountant.
- **Purpose:** FX-rate publication, translation, revaluation and realized or unrealized gain/loss calculation.
- **Authoritative records:** CurrencyRateSet, RevaluationRun, TranslationRun.
- **Functional requirement coverage:** `FR-FX-001`, `FR-FX-002`, `FR-FX-003`, `FR-FX-004`, `FR-FX-005`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `FX-WS-01` | Currency operations worklist | Manage rate sets, revaluation, translation, approvals, postings, reruns, and reversals. |
| `FX-SCR-01` | Rate-set publication | Review provider, rate type/date, currency pairs, rates, version, and publication status. |
| `FX-SCR-02` | Revaluation run | Select scope/period/rates, preview exposures and gain/loss, approve, post, rerun, and reverse. |
| `FX-SCR-03` | Translation run | Select participant scopes, presentation currency, rate policy, watermarks, and publish translation result. |
| `FX-SCR-04` | Conversion evidence panel | Show transaction/functional amounts, rate set, rate type, date/time, and calculation lineage. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-fpm"></a>
### 7.13 Fiscal Period Management

- **Primary users:** Finance Manager; Controller; Close Coordinator.
- **Purpose:** Fiscal-period state authority, close orchestration, and scoped or operational reopen controls.
- **Authoritative records:** FiscalPeriod, SoftCloseRun, CloseRun, ReopenRequest.
- **Functional requirement coverage:** `FR-FPM-001`, `FR-FPM-002`, `FR-FPM-003`, `FR-FPM-004`, `FR-FPM-005`, `FR-FPM-006`, `FR-FPM-007`, `FR-FPM-008`, `FR-FPM-009`, `FR-FPM-010`, `FR-FPM-011`, `FR-FPM-012`, `FR-FPM-013`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `FPM-WS-01` | Period-control dashboard | Show period status, gate mode, active owner, close/reopen processes, tasks, exceptions, and cutoffs. |
| `FPM-SCR-01` | Soft-close run | Start/end soft close, show policy, control epoch, admissions, exceptions, and handoff status. |
| `FPM-SCR-02` | Hard-close/reclose run | Show checklist, barrier, close tasks, approvals, admissions, watermark, seal, and finalization. |
| `FPM-SCR-03` | Reopen request | Capture reason, mode, scope, corrections, impact, approval, expiry, admissions, and completion path. |
| `FPM-SCR-04` | Period-control recovery | Show authoritative gate status, takeover eligibility, control authority, cutoff, expiry, and recovery actions. |
| `FPM-SCR-05` | Close exception review | Record exception, owner, due date, impact, approval/extension, resolution, and blocker status. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-coa"></a>
### 7.14 COA Segment Accounting

- **Primary users:** Chart-of-Accounts Administrator; Finance Approver.
- **Purpose:** Segment definitions, combinations, assignments and effective-dated controls.
- **Authoritative records:** SegmentDefinition, SegmentCombination, SegmentChangeRequest.
- **Functional requirement coverage:** `FR-COA-001`, `FR-COA-002`, `FR-COA-003`, `FR-COA-004`, `FR-COA-005`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `COA-WS-01` | Segment administration worklist | Manage definitions, values, combinations, changes, approvals, and validation exceptions. |
| `COA-SCR-01` | Segment definition | Maintain segment type, code, name, status, and effective date range. |
| `COA-SCR-02` | Segment values | Maintain values, descriptions, statuses, and effective date ranges. |
| `COA-SCR-03` | Combination validator | Validate combinations and show invalid values, restrictions, and effective-date reasons. |
| `COA-SCR-04` | Segment change request | Review requested change, effective date, approval, impacted records, and applied decision. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-bfr"></a>
### 7.15 Bank Feeds & Reconciliation

- **Primary users:** Bank Reconciliation Specialist; Cash Accountant.
- **Purpose:** Provider connections, statement ingestion, transaction matching and reconciliation.
- **Authoritative records:** BankFeedConnection, BankStatement, ReconciliationSession.
- **Functional requirement coverage:** `FR-BFR-001`, `FR-BFR-002`, `FR-BFR-003`, `FR-BFR-004`, `FR-BFR-005`, `FR-BFR-006`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `BFR-WS-01` | Bank reconciliation worklist | Manage connections, imports, statements, matching, unmatching, differences, and completion. |
| `BFR-SCR-01` | Bank-feed connection | Maintain provider, credential-reference status, consent, expiry, synchronization position, and connection status without exposing credential material. |
| `BFR-SCR-02` | Statement import | Review statement identity, period, balances, fingerprint, lines, duplicates, and validation. |
| `BFR-SCR-03` | Matching workspace | Compare bank lines with candidate business records; propose, confirm, reject, or split matches. |
| `BFR-SCR-04` | Reconciliation session | Show opening/closing balance, matched/unmatched totals, differences, exceptions, and completion evidence. |
| `BFR-SCR-05` | Match detail and unmatch | Show match evidence, rule version, user confirmation, downstream effects, and reversal eligibility. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-tax"></a>
### 7.16 Tax Filing

- **Primary users:** Tax Specialist; Tax Manager; Tax Approver.
- **Purpose:** Tax determination inputs, returns, submissions, amendments, return-level adjustments, tax-payment obligations, and filing status.
- **Authoritative records:** TaxConfiguration, TaxReturn, FilingSubmission, TaxAmendment, ReturnLevelTaxAdjustment, TaxPaymentObligation.
- **Functional requirement coverage:** `FR-TAX-001`, `FR-TAX-002`, `FR-TAX-003`, `FR-TAX-004`, `FR-TAX-005`, `FR-TAX-006`, `FR-TAX-007`, `FR-TAX-008`, `FR-TAX-009`, `FR-TAX-010`, `FR-TAX-011`, `FR-TAX-012`, `FR-TAX-013`, `FR-TAX-014`, `FR-TAX-015`, `FR-TAX-016`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `TAX-WS-01` | Tax operations worklist | Manage configuration, determinations, returns, submissions, amendments, adjustments, payments, and evidence. |
| `TAX-SCR-01` | Tax configuration | Maintain jurisdictions, rules, rates, categories, and effective-date ranges. |
| `TAX-SCR-02` | Tax determination review | Review source facts, rule version, jurisdiction, calculation, exceptions, and finalization blocker. |
| `TAX-SCR-03` | Tax return and submission | Prepare, approve, submit, reconcile authority outcome, and preserve immutable submitted versions. |
| `TAX-SCR-04` | Tax amendment | Link accepted original return/version, reason, approval, submission, and accepted lineage. |
| `TAX-SCR-05` | Return-level adjustment | Create, approve, post, retry, and reconcile tax adjustments independently of return/amendment records. |
| `TAX-SCR-06` | Tax payment obligation | Request payment, track instruction and settlement, handle return/failure, and preserve filing status. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-wfa"></a>
### 7.17 Workflow & Approvals

- **Primary users:** Approver; Delegator; Approval Administrator.
- **Purpose:** Configurable approval policies, steps, decisions, delegation and escalation.
- **Authoritative records:** ApprovalPolicy, ApprovalRequest, Delegation.
- **Functional requirement coverage:** `FR-WFA-001`, `FR-WFA-002`, `FR-WFA-003`, `FR-WFA-004`, `FR-WFA-005`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `WFA-WS-01` | Approval inbox and worklist | Show assigned, delegated, escalated, due, blocked, and completed approvals. |
| `WFA-SCR-01` | Approval request detail | Show subject snapshot, policy/version, steps, decisions, delegation, escalation, and application status. |
| `WFA-SCR-02` | Approval decision workspace | Approve or reject with required evidence and show segregation and current-state revalidation. |
| `WFA-SCR-03` | Approval policy editor | Maintain applicability, thresholds, steps, approvers, delegations, escalations, and effective dates. |
| `WFA-SCR-04` | Delegation and escalation | Create/revoke delegation and escalate overdue or blocked decisions with audit evidence. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-iam"></a>
### 7.18 Identity & Access

- **Primary users:** Security Administrator; Access Approver; Auditor.
- **Purpose:** Users, roles, permissions, access scopes and segregation-of-duties controls.
- **Authoritative records:** User, Role, AccessPolicy, SegregationRule.
- **Functional requirement coverage:** `FR-IAM-001`, `FR-IAM-002`, `FR-IAM-003`, `FR-IAM-004`, `FR-IAM-005`, `FR-IAM-006`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `IAM-WS-01` | Access administration worklist | Manage users, roles, policies, segregation rules, emergency access, reviews, and exceptions. |
| `IAM-SCR-01` | User access record | Show identity, authentication subject, roles, scopes, status, and access-review evidence. |
| `IAM-SCR-02` | Role and access policy | Maintain permissions across scope and action dimensions. |
| `IAM-SCR-03` | Segregation rule | Maintain prohibited combinations, thresholds, exceptions, and approval policy. |
| `IAM-SCR-04` | Emergency-access request | Capture reason, scope, expiry, approval, permitted actions, activity, and revocation/review. |
| `IAM-SCR-05` | Access decision explanation | Show allowed/denied result, applicable dimensions, conflicting duty, and next permitted action. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="capability-aud"></a>
### 7.19 Audit Integrity

- **Primary users:** Internal Auditor; Compliance Officer; Integrity Administrator.
- **Purpose:** Append-only audit evidence, integrity sealing, proof generation, verification, and incident lineage.
- **Authoritative records:** AuditChain.
- **Functional requirement coverage:** `FR-AUD-001`, `FR-AUD-002`, `FR-AUD-003`, `FR-AUD-004`, `FR-AUD-005`

| Screen ID | Workspace / screen | UX responsibility |
|---|---|---|
| `AUD-WS-01` | Audit-integrity worklist | Manage appended evidence, seals, proof verification, credential rotation, and incidents. |
| `AUD-SCR-01` | Audit-chain scope | Show sequence, events, fingerprints, gaps, seals, legal hold, and source references. |
| `AUD-SCR-02` | Seal detail | Show covered range, seal status, verification credential, supersession, and proof reference. |
| `AUD-SCR-03` | Proof verification | Select scope/range/proof and show Valid, MissingEvent, ProofMismatch, InvalidProof, or UnsupportedVersion. |
| `AUD-SCR-04` | Integrity incident | Show affected range, evidence, severity, owner, containment, recovery, corrective seals, and closure. |
| `AUD-SCR-05` | Verification-credential rotation | Show current/previous credential intervals, reason, approvals, impact, and replacement-seal requirements. |

**Workspace requirements**
- Default worklists group records by actionable state and expose scope, owner, amount/currency where applicable, age, approval, exception, and next action.
- Record details use the common identity, state, action, lineage, evidence, and sensitivity components applicable to the record.
- Search results link to the authoritative record; aggregated process views never become a second mutation surface for another capability’s record.

<a id="section-8"></a>
## 8. Detailed Workflow Specifications

Each workflow uses the source PRD workflow ID. `UXA-*` criteria in this document supplement, but do not replace, the functional acceptance scenarios in the source acceptance baseline.

<a id="workflow-wf-6-1"></a>
### 8.1 WF-6.1 — Period Close: Hard Close

- **User goal:** Finance Manager initiates hard close.
- **Primary actor/context:** Finance Manager
- **Supporting actors:** Workflow & Approvals, close approvers, accountants, automated close scheduler
- **Ownership:** Fiscal Period Management
- **Entry point:** From the period-control dashboard for a SoftClosed period, or from an assigned close task.
- **Primary screens:** `FPM-WS-01`, `FPM-SCR-02`, `GL-SCR-03`, `FPM-SCR-05`, `WFA-SCR-02`, `RPT-SCR-03`, `AUD-SCR-02`
- **Direct requirements:** `FR-FPM-003`, `FR-FPM-006`, `FR-FPM-007`, `FR-FPM-008`, `FR-GL-006`, `FR-GL-007`, `FR-GL-008`, `FR-GL-014`, `FR-RPT-006`
- **Supporting families:** FR-FPM-*, FR-GL-*, FR-WFA-*, FR-FX-*, FR-FA-*, FR-REV-*, FR-IC-*, FR-RPT-*, FR-AUD-*
- **DDD source:** §6.1
- **Functional acceptance source:** DDD acceptance §§14.3 and 14.8; this UX flow does not alter those scenarios.

#### User journey
1. Select the accounting scope and period; confirm `SoftClosed`, the active soft-close owner/epoch, and the current posting-gate version.
2. Start hard close and review the candidate close-run summary, checklist template, cutoff evidence, and unresolved blockers before submission.
3. Show the barrier-acquisition transition as a single protected step. While pending, keep the soft-close owner visible and disable finalization.
4. After barrier acquisition, show `CloseOnly`, the close-run owner, barrier ledger position, and frozen soft-close admission summary.
5. Present close tasks by responsible capability: revaluation, depreciation, revenue recognition, intercompany controls, journal exceptions, and reporting checks.
6. Route close exceptions and the final close decision through approval workspaces; show applied versus merely recorded approval decisions.
7. Before finalization, show a readiness summary covering checklist completion, admitted close postings, reconciliations, final watermark, and unresolved exceptions.
8. After finalization, show `HardClosed`, the final gate version and watermark, statement readiness, and audit-seal status without hiding a pending or failed seal.

#### Exception and recovery states
- Gate owner/version mismatch
- Close task or posting failure
- Unresolved or expired close exception
- Final approval invalidated by changed business state
- Seal pending or failed after accounting close

#### Completion condition
Period is HardClosed with a final watermark; the close run is completed; statement and audit evidence are linked and visible.

#### UX acceptance criteria
- **UXA-6-1-01:** The UI never shows both the soft-close run and close run as active gate owners.
- **UXA-6-1-02:** Ordinary posting actions are visibly unavailable after the barrier and explain the `CloseOnly` restriction.
- **UXA-6-1-03:** Finalization remains unavailable until required tasks, approvals, and gate evidence are satisfied.
- **UXA-6-1-04:** A completed accounting close remains distinguishable from audit-seal completion.

<a id="workflow-wf-6-2"></a>
### 8.2 WF-6.2 — Fiscal Period Reopen and Reclose

- **User goal:** Fiscal Period Reopen and Reclose
- **Primary actor/context:** Controller
- **Supporting actors:** Independent approvers, affected subledger owners, close domain process, General Ledger (GL), Audit Integrity, Financial Reporting
- **Ownership:** Fiscal Period Management owns ReopenRequest, the period transitions, and reclose orchestration. GL owns the posting gate and every journal-entry admission decision. Workflow owns approval decisions.
- **Entry point:** From a HardClosed period record or a correction item that requires a closed-period change.
- **Primary screens:** `FPM-WS-01`, `FPM-SCR-03`, `WFA-SCR-02`, `GL-SCR-03`, `GL-WS-01`, `FPM-SCR-02`, `RPT-SCR-03`, `AUD-SCR-02`
- **Direct requirements:** `FR-FPM-006`, `FR-FPM-009`, `FR-FPM-010`, `FR-FPM-011`, `FR-GL-001`, `FR-GL-007`, `FR-GL-008`, `FR-GL-009`, `FR-GL-010`, `FR-GL-013`, `FR-GL-014`
- **Supporting families:** FR-FPM-*, FR-GL-*, FR-WFA-*, FR-RPT-*, FR-AUD-*
- **DDD source:** §6.2
- **Functional acceptance source:** DDD acceptance §§14.10 and 14.8; this UX flow does not alter those scenarios.

#### User journey
1. Create a reopen request with mode, reason, impacted accounts or transaction classes, proposed corrections, impact analysis, authorization expiry, and expected versions.
2. Show the request in `PendingApproval`; the period and posting gate remain HardClosed until the decision is applied.
3. After approval revalidation, open the scoped or operational reopen gate and show allowed posting purposes, actors/classes, expiry, owner, and gate version.
4. Provide a correction worklist that shows admitted, rejected, pending, and failed postings and their relation to the reopen request.
5. On close request, show the retained gate-admission summary and determine the no-change versus reclose path from authoritative admissions.
6. For zero admitted postings, close to HardClosed and show `CompletedNoChange`, retaining the prior watermark and seal lineage.
7. For any admitted posting, require `StartReclose`/`BeginRecloseGate`, show exclusive ownership transfer, and run the reclose checklist.
8. Publish revised statements and seal lineage after reclose; preserve the original close and reopen evidence.

#### Exception and recovery states
- Approval rejection
- Authorization expiry
- Posting outside approved scope
- Stale period/gate version
- ExpiredPendingClosure
- Mandatory reclose interrupted during ownership handoff

#### Completion condition
Request is Rejected, CompletedNoChange, or Completed after reclose, with explicit statement and seal lineage.

#### UX acceptance criteria
- **UXA-6-2-01:** Opening a reopen gate is impossible before an approval decision is applied and current state is revalidated.
- **UXA-6-2-02:** The UI derives the no-change/reclose choice from the retained admission summary rather than user selection.
- **UXA-6-2-03:** An admitted correction makes reclose mandatory and disables direct restoration to HardClosed.
- **UXA-6-2-04:** Expired admission blocks new postings but does not imply the reopen owner has released control.

<a id="workflow-wf-6-3"></a>
### 8.3 WF-6.3 — Intercompany Reconciliation and Settlement

- **User goal:** Intercompany Reconciliation and Settlement
- **Primary actor/context:** Intercompany Accountant
- **Supporting actors:** Counterparty accountants, residual approver, Workflow & Approvals, Multi-Currency, Payments & Cash Management, GL, Financial Reporting
- **Ownership:** Multi-Entity / Intercompany owns agreements, reciprocal matching, netting, residual treatment, and settlement-run state. Payments owns bank execution. Financial Reporting owns consolidation elimination records.
- **Entry point:** From the intercompany worklist for a settlement period and agreement.
- **Primary screens:** `IC-WS-01`, `IC-SCR-01`, `IC-SCR-03`, `IC-SCR-04`, `WFA-SCR-02`, `PCM-SCR-02`, `PCM-SCR-04`, `RPT-SCR-02`, `IC-SCR-05`
- **Direct requirements:** `FR-IC-001`, `FR-IC-002`, `FR-IC-003`, `FR-IC-004`, `FR-IC-005`
- **Supporting families:** FR-IC-*, FR-PCM-*, FR-FX-*, FR-WFA-*, FR-GL-*, FR-RPT-*
- **DDD source:** §6.3
- **Functional acceptance source:** DDD acceptance §14.11; this UX flow does not alter those scenarios.

#### User journey
1. Select participant accounting scopes, agreement/version, cutoff, rate policy, settlement currency, and eligible open items.
2. Validate and reserve eligible items; display excluded, disputed, already reserved, settled, or changed items with reasons.
3. Run reciprocal matching and present matched pairs, one-sided items, ambiguous matches, currency differences, and tolerance results.
4. Resolve or approve residual differences; show policy evidence and prevent instruction creation while blocking exceptions remain.
5. Preview netting and settlement instructions by participant, direction, currency, and clearing obligation before creation.
6. Track outgoing instructions and incoming expectations/receipts independently while presenting one settlement-run overview.
7. Apply settlement and return outcomes to intercompany clearing without rewriting source transactions.
8. Complete the run only when required obligations and exceptions have explicit outcomes; then make elimination instructions available to reporting.

#### Exception and recovery states
- Reservation/version conflict
- Unmatched or ambiguous reciprocal item
- Residual outside tolerance or pending approval
- Partial/failed/returned settlement
- Incoming receipt owner-application exception

#### Completion condition
Settlement run has explicit completed or exception outcome; clearing, payment/receipt, and elimination references are linked.

#### UX acceptance criteria
- **UXA-6-3-01:** Source intercompany items remain visible and unchanged after netting or settlement.
- **UXA-6-3-02:** A residual outside tolerance cannot be silently absorbed.
- **UXA-6-3-03:** Outgoing and incoming cash legs remain distinguishable from intercompany clearing obligations.
- **UXA-6-3-04:** Run completion exposes unresolved exceptions rather than presenting partial settlement as full success.

<a id="workflow-wf-6-4"></a>
### 8.4 WF-6.4 — Fixed Asset Disposal with Gain or Loss Recognition

- **User goal:** Accountant submits an approved disposal intent for sale, scrap, or partial disposal.
- **Primary actor/context:** Fixed Asset Accountant
- **Supporting actors:** Workflow & Approvals, General Ledger (GL), Accounts Payable (AP), Payments & Cash Management, and the disposal-recovery domain service
- **Ownership:** Fixed Assets
- **Entry point:** From an eligible asset/component record or an approved disposal task.
- **Primary screens:** `FA-WS-01`, `FA-SCR-01`, `FA-SCR-06`, `WFA-SCR-02`, `FA-SCR-07`, `GL-SCR-02`, `PCM-SCR-04`, `PCM-SCR-02`, `AP-SCR-01`
- **Direct requirements:** `FR-FA-005`, `FR-FA-007`, `FR-FA-008`
- **Supporting families:** FR-FA-*, FR-WFA-*, FR-GL-*, FR-AP-*, FR-PCM-*
- **DDD source:** §6.4
- **Functional acceptance source:** DDD acceptance §§14.2 and 14.13.7; this UX flow does not alter those scenarios.

#### User journey
1. Select asset or components and load cost, accumulated depreciation, impairment, carrying amount, prior disposal history, and current version.
2. Enter disposal date, reason, gross proceeds, disposal costs, quantity/components, counterparty evidence, and proposed accounting treatment.
3. Preview depreciation through disposal date, carrying amount, signed gain/loss, required posting legs, and which capability owns each accounting effect.
4. Submit for approval and show that material source changes invalidate the approval subject.
5. Post required Fixed Assets legs and show each leg independently; do not enable settlement handoffs until required accounting is authoritative.
6. If one required leg fails, show `PartiallyPosted`, protect the successful leg, and offer retry or approved compensation only for the failed path.
7. Create proceeds expectations, supplier-liability classification, or disposal-cost payment requests according to the selected closed treatment.
8. Track receipts, payments, returns, reversals, failures, replacements, and posted-disposal corrections while preserving the original disposal.

#### Exception and recovery states
- Asset/component version conflict
- Invalid treatment combination
- Approval invalidated by changed carrying amount or inputs
- Partial multi-leg posting
- Proceeds reversal or disposal-cost payment return
- Irrecoverable mixed-leg failure requiring compensation

#### Completion condition
Accounting and each required settlement obligation have explicit states; asset status and correction lineage are authoritative.

#### UX acceptance criteria
- **UXA-6-4-01:** The treatment selector exposes only allowed closed policy combinations.
- **UXA-6-4-02:** The preview identifies Fixed Assets, AP, and Payments ownership without duplicate cash or liability entries.
- **UXA-6-4-03:** A successful posting leg is never resubmitted when another leg fails.
- **UXA-6-4-04:** Settlement actions remain unavailable until the required accounting state permits them.

<a id="workflow-wf-6-5"></a>
### 8.5 WF-6.5 — Revenue Recognition for a SaaS Contract

- **User goal:** Revenue Recognition for a SaaS Contract
- **Primary actor/context:** Revenue Accountant
- **Supporting actors:** Contract approver, Workflow & Approvals, Invoicing, AR, GL, Multi-Currency, Financial Reporting
- **Ownership:** Revenue Recognition owns accounting assessment, performance obligations, contract balances, schedule versions, and published accounting profiles. Invoicing owns commercial invoice generation. AR owns invoices, receivables, credits, refunds, and billing postings.
- **Entry point:** From a new or changed contract source record requiring revenue-accounting assessment.
- **Primary screens:** `REV-WS-01`, `REV-SCR-01`, `REV-SCR-02`, `REV-SCR-03`, `WFA-SCR-02`, `REV-SCR-04`, `INV-SCR-05`, `AR-SCR-01`, `REV-SCR-06`, `RPT-SCR-03`
- **Direct requirements:** `FR-AR-001`, `FR-REV-001`, `FR-REV-002`, `FR-REV-003`, `FR-REV-004`, `FR-REV-005`, `FR-REV-006`
- **Supporting families:** FR-REV-*, FR-WFA-*, FR-INV-*, FR-AR-*, FR-GL-*, FR-FX-*, FR-RPT-*
- **DDD source:** §6.5
- **Functional acceptance source:** DDD acceptance §§14.12 and 14.13.8; this UX flow does not alter those scenarios.

#### User journey
1. Open the contract assessment with source version, accounting scope, customer, terms, promises, pricing, and policy version.
2. Record enforceability, collectibility, combination, contract term, and distinctness conclusions with supporting evidence.
3. Define performance obligations, transaction-price allocation, contract balances, and recognition methods.
4. Generate and review the revenue schedule; route approval and show the exact schedule/business version to which the decision applies.
5. Publish an immutable revenue accounting profile and show its effective range and downstream availability.
6. Track Invoicing and AR use of the profile without presenting contract approval as automatic deferred revenue or receivable posting.
7. Run recognition by period; preview scheduled amounts, rates, posting state, failures, and rerun/correction lineage.
8. Expose contract, schedule, profile, invoice, recognition, and statement traceability in one cross-capability timeline.

#### Exception and recovery states
- Collectibility or enforceability unresolved
- Approval invalidated by changed contract/schedule
- Missing or stale accounting profile
- Posting failure
- Source modification requiring reassessment

#### Completion condition
Approved schedule and profile are published; recognition results and downstream billing/AR relationships are visible.

#### UX acceptance criteria
- **UXA-6-5-01:** The interface distinguishes commercial billing from revenue-recognition accounting.
- **UXA-6-5-02:** Every invoice line using a revenue profile shows the exact profile/version.
- **UXA-6-5-03:** A changed classification creates a new profile version rather than editing a published one.
- **UXA-6-5-04:** Recognition posting failure preserves the schedule and exposes a retry/correction path.

<a id="workflow-wf-6-6"></a>
### 8.6 WF-6.6 — Journal Entry Posting and Reversal

- **User goal:** Journal Entry Posting and Reversal
- **Primary actor/context:** Accountant or authorized subledger
- **Supporting actors:** Workflow & Approvals for approval-bearing journals; originating subledger or accountant; Fiscal Period Management and the GL posting gate for period admission.
- **Ownership:** General Ledger
- **Entry point:** From the journal workbench, a manual-journal task, or an authorized subledger posting result link.
- **Primary screens:** `GL-WS-01`, `GL-SCR-01`, `GL-SCR-02`, `WFA-SCR-02`, `GL-SCR-03`
- **Direct requirements:** `FR-GL-001`, `FR-GL-002`, `FR-GL-003`
- **Supporting families:** FR-GL-*, FR-WFA-*
- **DDD source:** §6.6
- **Functional acceptance source:** DDD acceptance §§14.1 and 14.9; this UX flow does not alter those scenarios.

#### User journey
1. Create or inspect the posting request with source identity, accounting scope, dates, posting purpose, transaction currency, conversion evidence, and lines.
2. Run validation and show balancing, account/segment, period/gate, currency, authorization, and approval-policy results grouped by blocking severity.
3. On repeat submission, show the established in-progress or terminal result instead of offering a duplicate action.
4. When approval is required, show `PendingApproval`, the approval request, and absence of ledger effect.
5. On decision application, revalidate the current period, gate, account configuration, and authorization before posting.
6. Show the posted journal number, ledger position, period/gate evidence, lines, source, and immutable audit history.
7. For reversal, require reason/date/scope, preview equal-and-opposite lines, and create a linked reversal journal without editing the original.

#### Exception and recovery states
- Unbalanced or invalid lines
- Hard-closed or restricted gate
- Approval rejection or invalidation
- Idempotency conflict
- Expected-version conflict
- Dependency unavailable versus domain rejection

#### Completion condition
Request has a typed rejection, pending approval, posted result, or linked reversal with complete evidence.

#### UX acceptance criteria
- **UXA-6-6-01:** Validation messages identify the affected line/field and the authoritative blocking rule.
- **UXA-6-6-02:** The original posted journal never becomes editable after posting or reversal.
- **UXA-6-6-03:** Idempotency conflict clearly distinguishes changed content from a safe repeat.
- **UXA-6-6-04:** Approval does not bypass current-state revalidation.

<a id="workflow-wf-6-7"></a>
### 8.7 WF-6.7 — Customer Receipt Recording with Partial Application

- **User goal:** The specialist records a receipt. When allocations are supplied or later confirmed, the specialist applies some or all of the unapplied amount to receivable open items.
- **Primary actor/context:** Cash Applications Specialist
- **Supporting actors:** Bank Feeds & Reconciliation, collections specialist, AR accounting-recovery domain service
- **Ownership:** Accounts Receivable
- **Entry point:** From a normalized bank transaction, AR receipt worklist, or approved manual receipt source.
- **Primary screens:** `AR-WS-01`, `AR-SCR-02`, `AR-SCR-03`, `AR-SCR-04`, `BFR-SCR-05`, `GL-SCR-02`
- **Direct requirements:** `FR-AR-002`, `FR-AR-003`, `FR-AR-004`, `FR-AR-005`
- **Supporting families:** FR-AR-*, FR-BFR-*, FR-GL-*
- **DDD source:** §6.7
- **Functional acceptance source:** DDD acceptance §14.6; this UX flow does not alter those scenarios.

#### User journey
1. Record the receipt with customer, date, amount/currency, bank/source reference, and source identity; show Applied = 0 and Unapplied = receipt amount.
2. Track receipt-recording accounting separately and block application until the cash/unapplied-cash posting is authoritative.
3. Open the allocation workspace showing receipt availability and candidate open items with current open amounts, currency, due date, and eligibility.
4. Enter one or more allocations; validate customer/scope/currency, total available, open-item balances, and expected versions before commit.
5. Show the committed immutable application batch and application-accounting status separately from receipt-recording accounting.
6. For unapplication, select posted application facts, enter reason/amount, and preview the adjustment without deleting original applications.
7. For a terminally unposted application batch, offer evidence-backed rollback—not unapplication—and show restored balances plus immutable rollback facts.
8. Expose receipt, application, unapplication/rollback, posting, bank match, and reconciliation lineage.

#### Exception and recovery states
- Receipt posting pending/failed
- Over-allocation
- Open-item version conflict
- Application posting failed
- Unapplication attempted before application posting
- No-journal proof unavailable

#### Completion condition
Receipt and each application/unapplication batch have authoritative balances, accounting states, and evidence.

#### UX acceptance criteria
- **UXA-6-7-01:** The UI never combines receipt recording and application into one indistinguishable accounting result.
- **UXA-6-7-02:** Allocation totals update before submission and cannot exceed receipt or open-item availability.
- **UXA-6-7-03:** Unapplication never deletes or edits the original application fact.
- **UXA-6-7-04:** Rollback is offered only for a terminally unposted application batch with authoritative no-journal evidence.

<a id="workflow-wf-7-1"></a>
### 8.8 WF-7.1 — Vendor Invoice Registration, Matching, Approval, Dispute, and Void

- **User goal:** Received -> Validated -> PendingApproval -> Approved -> PartiallyPaid -> Paid, with DuplicateSuspected, Disputed, Rejected, and Voided alternatives
- **Primary actor/context:** AP Specialist, AP approver, and procurement-data provider
- **Ownership:** AP Specialist, AP approver, and procurement-data provider. AP owns RegisterVendorInvoice, ValidateVendorInvoice, ApplyVendorInvoiceApprovalDecision, DisputeVendorInvoice, and VoidVendorInvoice; Workflow owns the underlying approval decision; VendorInvoice is the consistency boundary and AP is the sole liability-posting producer.
- **Entry point:** From invoice capture/import, a procurement-snapshot link, or the AP invoice worklist.
- **Primary screens:** `AP-WS-01`, `AP-SCR-01`, `AP-SCR-02`, `WFA-SCR-02`, `AP-SCR-03`, `GL-SCR-02`, `AP-SCR-04`
- **Direct requirements:** `FR-AP-001`, `FR-AP-006`, `FR-AP-008`, `FR-AP-009`, `FR-AP-010`
- **Supporting families:** FR-AP-*, FR-WFA-*, FR-OMD-*
- **DDD source:** §7.1
- **Functional acceptance source:** DDD acceptance §14.13.1; this UX flow does not alter those scenarios.

#### User journey
1. Register invoice identity, vendor, scope, number/date, amounts, tax, lines, and source evidence; immediately show duplicate-fingerprint results.
2. Validate vendor/scope, totals, tax, required fields, and snapshot versions; distinguish validation rejection from duplicate suspicion.
3. Open match review to compare invoice, purchase-order, and receipt snapshots and show tolerance/exception results per line.
4. Route approval with a frozen subject snapshot; after decision, revalidate the current invoice and snapshot versions.
5. Allow dispute from eligible states with reason, amount, owner, evidence, and expected resolution; show disputed balances separately.
6. After approval, show liability-posting state and journal result; preserve approval when posting retries are needed.
7. Create payment requests only from eligible outstanding allocations; show requested, paid, returned, and remaining liability.
8. Void only through the allowed pre/post-settlement correction path; never delete the established invoice.

#### Exception and recovery states
- Duplicate suspected/conflict
- Procurement snapshot changed
- Match outside tolerance
- Approval rejection/invalidation
- Posting failed
- Dispute unresolved
- Void after settlement requiring credit/refund/recovery

#### Completion condition
Invoice is rejected, disputed, voided, approved/posted with remaining balance, or paid with complete correction lineage.

#### UX acceptance criteria
- **UXA-7-1-01:** Duplicate suspicion is visible before approval and does not silently merge records.
- **UXA-7-1-02:** Match evidence identifies the exact snapshot versions used.
- **UXA-7-1-03:** A posted or settled invoice cannot be edited in place.
- **UXA-7-1-04:** Posting retry preserves the approved invoice and stable posting identity.

<a id="workflow-wf-7-2"></a>
### 8.9 WF-7.2 — Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation

- **User goal:** A batch reaches Completed with FullySettled, FullyCancelled, PartiallySettledCancelled, or CompletedWithExceptions; whole-batch Cancelled applies only when CancelPaymentBatch succeeds before any instruction is provider-submitted
- **Primary actor/context:** Payment preparer, independent approver, treasury operator, provider, reconciler, and owning obligation contexts
- **Ownership:** Payment preparer, independent approver, treasury operator, provider, reconciler, and owning obligation contexts. Payments owns batch and instruction execution, expectation registration and resolution, observed receipts, evidence-backed no-journal cancellation, owner acknowledgement, receipt reversal, and linked outgoing payment returns; Workflow owns approval decisions. Each PaymentInstruction, PaymentReturn, ExpectedIncomingSettlement, and SettlementReceipt is independently versioned.
- **Entry point:** From approved payment obligations, the treasury worklist, or an expected incoming-settlement handoff.
- **Primary screens:** `PCM-WS-01`, `PCM-SCR-01`, `WFA-SCR-02`, `PCM-SCR-02`, `PCM-SCR-03`, `PCM-SCR-04`, `PCM-SCR-05`, `PCM-SCR-06`, `GL-SCR-02`
- **Direct requirements:** `FR-PCM-003`
- **Supporting families:** FR-PCM-*, FR-WFA-*, FR-AP-*, FR-PAYR-*, FR-TAX-*, FR-FA-*, FR-AR-*
- **DDD source:** §7.2
- **Functional acceptance source:** DDD acceptance §14.13.2; this UX flow does not alter those scenarios.

#### User journey
1. Prepare a payment batch from eligible obligations; show control totals by currency, funding account, beneficiary validation, and preparer/approver segregation.
2. Route and apply approval; show the approved batch/instruction versions and prevent changed instructions from using stale approval.
3. Submit instructions and track each independently through prepared, submitted, acknowledged, partial, settled, failed, cancellation, or exception outcomes.
4. Allow whole-batch cancellation only before any instruction is provider-submitted; after submission, derive the batch outcome from terminal instruction mixtures.
5. For failed or unpaid remainder, show owner-controlled exception decisions: cancel remainder, replacement obligation, accepted unpaid exception, or policy write-off.
6. For provider returns, show reservation, cash correction posting, owner acknowledgement, reconciliation, no-journal cancellation eligibility, and posted-return reversal as distinct paths.
7. For incoming obligations, register expectations and allocate observed bank receipts without exceeding remaining amount; track validation, posting, owner application, and reconciliation separately.
8. Post excess bank allocation to unallocated incoming cash and require explicit allocation/refund/reclassification resolution.
9. Expose gross, cancelled, remaining, reserved return, posted return, reversed return, reconciled return, and net settlement balances with the source equations: `AuthorizedMoney = SettledMoney + CancelledMoney + RemainingMoney`, `NetSettledMoney = SettledMoney - PostedReturnMoney + ReversedReturnMoney`, `ReservedReturnMoney + PostedReturnMoney - ReversedReturnMoney <= SettledMoney`, and `ReversedReturnMoney + ReconciledReturnMoney <= PostedReturnMoney`.

#### Exception and recovery states
- Control-total or beneficiary validation failure
- Approval invalidated
- Provider uncertain/failed outcome
- Partial settlement and cancellation
- Owner decision pending
- Return posting failed versus posted-return reversal
- Receipt validation/owner rejection
- Unallocated cash posting or resolution failure

#### Completion condition
Batch and every instruction/return/expectation/receipt have explicit terminal or managed-exception states; no obligation or cash difference is hidden.

#### UX acceptance criteria
- **UXA-7-2-01:** Batch cancellation is unavailable after provider submission begins and explains the resulting completion-outcome model.
- **UXA-7-2-02:** Return no-journal cancellation is unavailable after a return cash posting exists.
- **UXA-7-2-03:** Incoming allocation cannot make expectation remaining amount negative.
- **UXA-7-2-04:** Excess cash remains a separate posted exception until an authorized resolution is authoritative.
- **UXA-7-2-05:** All settlement balance equations are visible and reconcile.

<a id="workflow-wf-7-3"></a>
### 8.10 WF-7.3 — Customer Credit, Refund, Overpayment, Chargeback, and Write-Off

- **User goal:** Credit, chargeback, and write-off commands create immutable adjustment records
- **Primary actor/context:** AR Specialist, collections manager, Workflow approver, Payments, provider, and reconciler
- **Ownership:** AR Specialist, collections manager, Workflow approver, Payments, provider, and reconciler. AR owns credit notes, receivable adjustments, overpayment and unapplied-cash decisions, CustomerRefundRequest, chargebacks, refunds, and write-offs; Payments owns external refund execution and bank cash. AR is the sole producer of receivable, refund-payable, and refund-clearing accounting.
- **Entry point:** From a customer account, receipt/open item, collections exception, or approved credit/refund task.
- **Primary screens:** `AR-WS-01`, `AR-SCR-05`, `AR-SCR-06`, `WFA-SCR-02`, `PCM-SCR-02`, `PCM-SCR-03`, `GL-SCR-02`
- **Direct requirements:** `FR-AR-006`, `FR-AR-007`, `FR-AR-008`, `FR-AR-009`, `FR-AR-010`, `FR-AR-011`, `FR-AR-012`, `FR-AR-013`, `FR-AR-014`, `FR-AR-015`, `FR-AR-016`
- **Supporting families:** FR-AR-*, FR-PCM-*, FR-WFA-*
- **DDD source:** §7.3
- **Functional acceptance source:** DDD acceptance §14.13.3; this UX flow does not alter those scenarios.

#### User journey
1. Choose credit note, overpayment resolution, chargeback, write-off, or customer refund and show the eligible source balances and policy limits.
2. Capture reason, amount/currency, source invoice/receipt/credit, evidence, expected version, and required approval.
3. Preview the AR-owned receivable/refund-payable/clearing accounting effect and prevent amounts above authoritative eligibility.
4. For refunds, distinguish approval rejection from user cancellation before payment request; preserve the request and decision lineage.
5. After approval, post the refund-payable-to-clearing leg, request payment, and track partial/settled/failed/remainder-cancelled outcomes.
6. When cancellation is requested, show that the provider/Payments outcome—not the request alone—determines zero-settlement `Cancelled` versus `PartiallySettledCancelled`.
7. For a returned refund, require the linked clearing-to-refund-payable correction before increasing remaining amount or enabling replacement payment.
8. Show original invoice/receipt, adjustments, refund legs, instructions, settlements, returns, correction postings, replacement lineage, and the source equations `AuthorizedMoney = NetSettledMoney + CancelledMoney + RemainingMoney` and `NetSettledMoney = GrossSettledMoney - ReturnedMoney`.

#### Exception and recovery states
- Adjustment exceeds open/refundable amount
- Approval rejection/invalidation
- Posting failed
- Provider failure
- Partial settlement cancellation
- Return correction posting failed or exception
- Concurrent customer-balance change

#### Completion condition
Each adjustment/refund has an explicit accounting, payment, return, cancellation, and remaining-obligation outcome.

#### UX acceptance criteria
- **UXA-7-3-01:** Credit, chargeback, and write-off create immutable adjustment records rather than editing the invoice.
- **UXA-7-3-02:** Refund approval rejection and request cancellation are distinct user-visible outcomes.
- **UXA-7-3-03:** A returned refund does not restore remaining amount until AR correction posting is authoritative.
- **UXA-7-3-04:** The original receipt and invoice facts remain unchanged.

<a id="workflow-wf-7-4"></a>
### 8.11 WF-7.4 — Bank Statement Import, Matching, Unmatching, and Reconciliation

- **User goal:** Statement Imported -> Validated -> Matching -> Reconciled, with Rejected or Exception paths
- **Primary actor/context:** Reconciliation Specialist, bank-feed provider, AR, AP, and Payments
- **Ownership:** Reconciliation Specialist, bank-feed provider, AR, AP, and Payments. Bank Feeds & Reconciliation owns ImportStatement, ProposeMatch, ConfirmMatch, Unmatch, and CompleteReconciliation; it owns matching records but not subledger business facts or cash-settlement postings.
- **Entry point:** From a bank-feed connection, uploaded statement, or reconciliation worklist.
- **Primary screens:** `BFR-WS-01`, `BFR-SCR-01`, `BFR-SCR-02`, `BFR-SCR-03`, `BFR-SCR-05`, `BFR-SCR-04`
- **Direct requirements:** `FR-BFR-001`, `FR-BFR-002`, `FR-BFR-003`, `FR-BFR-004`, `FR-BFR-005`
- **Supporting families:** FR-BFR-*, FR-AR-*, FR-AP-*, FR-PCM-*
- **DDD source:** §7.4
- **Functional acceptance source:** DDD acceptance §14.13.4; this UX flow does not alter those scenarios.

#### User journey
1. Select bank account and connection; import a statement and show fingerprint, period, opening/closing balances, lines, and duplicate result.
2. Validate statement continuity, currency/account, totals, and line identity; surface rejected lines without losing the statement evidence.
3. Show candidate matches by normalized bank line/allocation and owning business record, with rule version, confidence/reason, and amount coverage.
4. Allow propose, confirm, reject, split, or leave unmatched according to role and record state; confirmation shows downstream owner and effect.
5. For unmatch, show affected downstream acknowledgements/accounting and require the allowed reversal or correction path before completion.
6. Maintain reconciliation totals for matched, unmatched, timing, posted exceptions, and differences; explain each residual.
7. Complete only when required differences are resolved or explicitly accepted under policy; retain import and match history.

#### Exception and recovery states
- Duplicate statement
- Opening/closing balance discontinuity
- Candidate ambiguity
- Downstream record changed
- Unmatch blocked by irreversible downstream state
- Unresolved difference

#### Completion condition
Reconciliation session is completed with balanced totals or explicit approved differences and full match/unmatch lineage.

#### UX acceptance criteria
- **UXA-7-4-01:** A normalized bank allocation is uniquely identifiable and cannot be applied twice.
- **UXA-7-4-02:** Match confirmation shows the authoritative business owner before commitment.
- **UXA-7-4-03:** Unmatch does not silently erase downstream accounting or settlement effects.
- **UXA-7-4-04:** Completion explains every remaining difference.

<a id="workflow-wf-7-5"></a>
### 8.12 WF-7.5 — Foreign-Currency Invoice Settlement and Realized FX

- **User goal:** The scenario follows the DDD-defined lifecycle and outcomes
- **Primary actor/context:** AR accountant for customer receipts, AP accountant for vendor invoices, Payments for AP bank execution, and Multi-Currency as immutable rate-evidence publisher
- **Ownership:** AR accountant for customer receipts, AP accountant for vendor invoices, Payments for AP bank execution, and Multi-Currency as immutable rate-evidence publisher. AR owns customer-receipt cash, receivable clearing, and customer-settlement realized FX. AP owns vendor-invoice clearing and vendor-settlement realized FX. Payments owns only the bank-cash leg of AP payment instructions.
- **Entry point:** From an eligible foreign-currency customer or vendor invoice settlement.
- **Primary screens:** `AP-SCR-01`, `AR-SCR-01`, `FX-SCR-04`, `PCM-SCR-02`, `AR-SCR-02`, `GL-SCR-02`
- **Direct requirements:** None
- **Supporting families:** FR-AP-*, FR-AR-*, FR-FX-*, FR-PCM-*, FR-GL-*
- **DDD source:** §7.5
- **Functional acceptance source:** DDD acceptance §14.13.5; this UX flow does not alter those scenarios.

#### User journey
1. Show invoice transaction currency, functional currency, original recognition rate evidence, open transaction amount, and open functional amount.
2. Select or confirm settlement evidence and rate set/type/date permitted by policy.
3. Preview settlement allocation, functional cash/clearing amount, and realized FX gain/loss with ownership identified as AR or AP.
4. Validate that each posting request uses one transaction currency and that functional-only adjustment lines are clearly identified.
5. Submit settlement/accounting and track cash evidence, invoice closure/partial balance, realized FX posting, and journal references.
6. For return/reversal, show the linked correction and restored open amount without rewriting the original settlement.

#### Exception and recovery states
- Missing/stale rate evidence
- Currency mismatch
- Settlement exceeds open amount
- Period/gate rejection
- Cash settlement or posting failure
- Payment return/receipt reversal

#### Completion condition
Invoice and settlement balances reconcile in transaction and functional currencies with immutable rate evidence and realized FX result.

#### UX acceptance criteria
- **UXA-7-5-01:** Users can distinguish original recognition rate from settlement rate.
- **UXA-7-5-02:** The realized FX preview identifies the owning subledger.
- **UXA-7-5-03:** Transaction and functional amounts never appear as one ambiguous amount.
- **UXA-7-5-04:** Returns/reversals preserve original settlement evidence.

<a id="workflow-wf-7-6"></a>
### 8.13 WF-7.6 — Period-End Revaluation, Rerun, and Next-Period Reversal

- **User goal:** Draft -> Calculating -> PendingApproval -> Approved -> Posting -> Completed, with Rejected, Failed, Superseded, and Reversed states
- **Primary actor/context:** Treasury or GL accountant, Workflow approver, Multi-Currency, and GL
- **Ownership:** Treasury or GL accountant, Workflow approver, Multi-Currency, and GL. Multi-Currency owns RunRevaluation, ApplyRevaluationApprovalDecision, PostRevaluationRun, calculation results, reruns, and reversal instructions; Workflow owns the approval decision, and Multi-Currency is the sole producer of unrealized FX adjustments.
- **Entry point:** From period-close tasks or the currency operations worklist.
- **Primary screens:** `FX-WS-01`, `FX-SCR-02`, `FX-SCR-01`, `WFA-SCR-02`, `GL-SCR-02`, `FPM-WS-01`
- **Direct requirements:** `FR-FX-002`, `FR-FX-003`, `FR-FX-004`
- **Supporting families:** FR-FX-*, FR-GL-*, FR-FPM-*, FR-WFA-*
- **DDD source:** §7.6
- **Functional acceptance source:** DDD acceptance §14.13.6; this UX flow does not alter those scenarios.

#### User journey
1. Create a revaluation run for accounting scope/period/rate set and show eligible monetary balances and source versions.
2. Preview calculations by account/currency, functional adjustment, gain/loss, rounding, and excluded items with reasons.
3. Route approval and show the exact calculation/run version to which the decision applies.
4. Post the approved run and show each posting result; keep the run approved when retrying a failed posting.
5. For rerun, compare prior and new inputs/results, preserve prior run lineage, and prevent duplicate active effects.
6. Create or track the next-period reversal with the original run, date, purpose, and journal references.

#### Exception and recovery states
- Missing/unpublished rate
- Source balance changed
- Approval invalidated
- Posting failed/uncertain
- Rerun conflict
- Reversal period unavailable

#### Completion condition
Run is posted or explicitly failed/cancelled with rerun and reversal lineage visible.

#### UX acceptance criteria
- **UXA-7-6-01:** Preview totals reconcile to the displayed source population.
- **UXA-7-6-02:** Approval applies to an immutable run version and is revalidated before posting.
- **UXA-7-6-03:** Rerun never overwrites or duplicates the original run.
- **UXA-7-6-04:** Next-period reversal is linked and separately traceable.

<a id="workflow-wf-7-7"></a>
### 8.14 WF-7.7 — Full Fixed-Asset Lifecycle and Disposal Variants

- **User goal:** Acquisition or CIP progresses to Capitalized, then Active, with transfer, split, impairment, and disposal subflows
- **Primary actor/context:** Fixed Asset Accountant, project accountant, Workflow approver, GL, AP, and Payments
- **Ownership:** Fixed Asset Accountant, project accountant, Workflow approver, GL, AP, and Payments. Fixed Assets owns acquisition, construction in progress, capitalization, transfer, split, depreciation, impairment, disposal proposal, approval application, treatment selection, required posting-leg orchestration, cancellation, asset-specific clearing, and posted-disposal correction; AP alone owns supplier liabilities and Payments alone owns bank cash.
- **Entry point:** From the asset register, AP clearing handoff, period task, or asset exception.
- **Primary screens:** `FA-WS-01`, `FA-SCR-02`, `FA-SCR-03`, `FA-SCR-04`, `FA-SCR-05`, `FA-SCR-06`, `FA-SCR-07`
- **Direct requirements:** `FR-FA-001`, `FR-FA-002`, `FR-FA-003`, `FR-FA-004`, `FR-FA-005`, `FR-FA-006`, `FR-FA-007`, `FR-FA-008`, `FR-FA-018`, `FR-FA-019`, `FR-FA-020`, `FR-FA-021`
- **Supporting families:** FR-FA-*, FR-AP-*, FR-PCM-*, FR-WFA-*, FR-GL-*
- **DDD source:** §7.7
- **Functional acceptance source:** DDD acceptance §14.13.7; this UX flow does not alter those scenarios.

#### User journey
1. Capitalize assets/components from approved source evidence and reconcile acquisition clearing without taking supplier-liability ownership.
2. Run depreciation with policy/period preview, exceptions, approval where required, posting, and immutable records.
3. Record impairment assessment and apply approval before posting the impairment result.
4. Transfer or split assets/components with allocation preview, effective date, destination/classification, and carrying-amount reconciliation.
5. Execute disposal using the detailed disposal workflow and treatment-specific accounting/settlement paths.
6. Handle incoming proceeds, disposal-cost payment, returns/reversals, failures, replacement, and settlement reopening.
7. Correct a posted disposal through linked correction records and postings; retain the original asset/disposal history.

#### Exception and recovery states
- Capitalization clearing mismatch
- Depreciation exception
- Impairment approval invalidated
- Transfer/split allocation imbalance
- Disposal partial posting
- Settlement return/reversal
- Posted-disposal correction required

#### Completion condition
Every asset/component has reconciled cost, depreciation, impairment, location/classification, disposal, and correction lineage.

#### UX acceptance criteria
- **UXA-7-7-01:** Asset cost and component allocations reconcile after capitalization, transfer, and split.
- **UXA-7-7-02:** Finalized depreciation and impairment facts are corrected through linked records.
- **UXA-7-7-03:** Supplier liability and bank cash remain owned by AP and Payments.
- **UXA-7-7-04:** Posted disposal correction does not rewrite the original disposal.

<a id="workflow-wf-7-8"></a>
### 8.15 WF-7.8 — Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration

- **User goal:** ModifyContract classifies separate-contract, prospective, or cumulative-catch-up treatment and enters PendingApproval when policy requires it
- **Primary actor/context:** Revenue Accountant, Workflow approver, Invoicing, and AR
- **Ownership:** Revenue Accountant, Workflow approver, Invoicing, and AR. Revenue Recognition owns modification assessment, ApplyContractModificationApprovalDecision, and schedule/profile versions; Workflow owns approval decisions and AR owns billing credits and refunds.
- **Entry point:** From a changed contract, renewal, cancellation, refund, or variable-consideration event.
- **Primary screens:** `REV-SCR-01`, `REV-SCR-05`, `REV-SCR-02`, `REV-SCR-03`, `WFA-SCR-02`, `REV-SCR-04`, `INV-SCR-05`, `AR-SCR-05`, `REV-SCR-06`
- **Direct requirements:** `FR-REV-004`, `FR-REV-005`
- **Supporting families:** FR-REV-*, FR-AR-*, FR-INV-*, FR-WFA-*, FR-GL-*
- **DDD source:** §7.8
- **Functional acceptance source:** DDD acceptance §14.13.8; this UX flow does not alter those scenarios.

#### User journey
1. Compare the new source contract version with the current revenue contract and show changed terms, promises, pricing, dates, and cancellation/refund facts.
2. Classify the modification under policy and record whether it is separate, prospective, cumulative catch-up, renewal, cancellation, or variable consideration update.
3. Preview performance-obligation, allocation, contract-balance, schedule, profile, billing, credit/refund, and catch-up impacts.
4. Route approval against the exact modification and affected schedule versions; revalidate before application.
5. Create new immutable modification, schedule, and profile versions rather than editing published history.
6. Coordinate downstream invoice, credit/refund, and recognition actions while showing which capability owns each accounting effect.
7. Run catch-up or prospective recognition and expose posting/reconciliation results.

#### Exception and recovery states
- Source version changed
- Modification classification incomplete
- Approval invalidated
- Downstream invoice already finalized
- Refund/credit eligibility conflict
- Recognition posting failed

#### Completion condition
Modification is rejected or applied with revised schedules/profiles and explicit downstream/catch-up outcomes.

#### UX acceptance criteria
- **UXA-7-8-01:** The before/after comparison identifies every affected obligation and balance.
- **UXA-7-8-02:** Published schedule/profile versions remain immutable.
- **UXA-7-8-03:** AR credit/refund and Revenue Recognition effects are clearly separated.
- **UXA-7-8-04:** Catch-up amount and rationale are reviewable before posting.

<a id="workflow-wf-7-9"></a>
### 8.16 WF-7.9 — Consolidation, Ownership Changes, Translation, Eliminations, and Rerun

- **User goal:** Draft -> Collecting -> Translating -> Eliminating -> PendingApproval -> Approved -> Published, with Rejected, Failed, and Superseded paths
- **Primary actor/context:** Consolidation Accountant, Workflow approver, Multi-Currency, Intercompany, and Financial Reporting
- **Ownership:** Consolidation Accountant, Workflow approver, Multi-Currency, Intercompany, and Financial Reporting. Multi-Currency owns rate selection and versioned TranslationRun calculations. Financial Reporting owns RunConsolidation, ApplyConsolidationApprovalDecision, translated balances, CTA records, ownership calculations, elimination records, and published consolidated statements. Workflow owns the publication approval decision, and Intercompany supplies versioned elimination instructions.
- **Entry point:** From a consolidation calendar task or reporting worklist.
- **Primary screens:** `RPT-WS-01`, `RPT-SCR-02`, `FX-SCR-03`, `IC-SCR-05`, `RPT-SCR-04`, `WFA-SCR-02`, `RPT-SCR-03`
- **Direct requirements:** `FR-RPT-001`, `FR-RPT-003`, `FR-RPT-004`
- **Supporting families:** FR-RPT-*, FR-FX-*, FR-IC-*, FR-WFA-*
- **DDD source:** §7.9
- **Functional acceptance source:** DDD acceptance §14.13.9; this UX flow does not alter those scenarios.

#### User journey
1. Select consolidation scope/period, participant scopes, ownership model, report definition, presentation currency, and source watermarks.
2. Validate source completeness and ownership changes; show missing, stale, or inconsistent participant data.
3. Run or apply translation results and display rate policy, source watermarks, translated balances, and CTA evidence.
4. Import/review intercompany elimination instructions and create reporting-owned elimination records.
5. Resolve translation, ownership, mapping, elimination, and balance exceptions with assigned owner and evidence.
6. Route consolidation approval; revalidate source watermarks and unresolved exceptions when applying the decision.
7. For rerun, preserve prior result/statement lineage and show differences caused by changed source, rates, ownership, or rules.
8. Publish consolidated statements with definition version, source watermarks, translation result versions, elimination lineage, and publication status.

#### Exception and recovery states
- Missing/stale participant data
- Ownership model change
- Translation or CTA exception
- Elimination mismatch
- Approval invalidated
- Rerun after publication

#### Completion condition
Consolidation run and statement are approved/published or explicitly blocked, with complete source and supersession lineage.

#### UX acceptance criteria
- **UXA-7-9-01:** Statutory ledger data is not presented as modified by consolidation workspace adjustments.
- **UXA-7-9-02:** Every published statement identifies exact source watermarks and definition version.
- **UXA-7-9-03:** Rerun preserves the prior published result and clearly identifies supersession.
- **UXA-7-9-04:** Exceptions identify the responsible capability and next action.

<a id="workflow-wf-7-10"></a>
### 8.17 WF-7.10 — Tax Return Submission, Rejection, Amendment, Payment, and Evidence

- **User goal:** The scenario follows the DDD-defined lifecycle and outcomes
- **Primary actor/context:** Tax Accountant, Workflow approver, tax-authority connector, Payments, and GL
- **Ownership:** Tax Accountant, Workflow approver, tax-authority connector, Payments, and GL. Tax Filing owns returns, filing submissions, amendment lineages, return-level adjustment aggregates, evidence, TaxPaymentObligation, and filing or obligation status. Source subledgers own transaction-level tax. Workflow owns approval decisions. Payments owns payment instructions and authoritative bank-cash settlement.
- **Entry point:** From a tax calendar obligation, determination exception, or authority response.
- **Primary screens:** `TAX-WS-01`, `TAX-SCR-02`, `TAX-SCR-03`, `WFA-SCR-02`, `TAX-SCR-04`, `TAX-SCR-05`, `TAX-SCR-06`, `PCM-SCR-02`
- **Direct requirements:** `FR-TAX-010`
- **Supporting families:** FR-TAX-*, FR-WFA-*, FR-PCM-*, FR-GL-*
- **DDD source:** §7.10
- **Functional acceptance source:** DDD acceptance §14.13.10; this UX flow does not alter those scenarios.

#### User journey
1. Review tax configuration/rule version and source facts; complete tax determination or expose a finalization blocker.
2. Prepare the return with jurisdiction, period, lines, totals, evidence, due date, and source versions.
3. Route approval and show that the submitted filing content/version becomes immutable.
4. Submit and track attempts; distinguish submitted, accepted, rejected, uncertain, and corrected outcomes using authority evidence.
5. For rejection or post-acceptance change, create a corrected submission attempt or separate TaxAmendment linked to the accepted original/version.
6. Create, approve, post, retry, and reconcile return-level adjustments independently of return/amendment lifecycle.
7. Create and track tax payment obligations through Payments without changing filing acceptance status.
8. Provide exportable filing, approval, submission, amendment, adjustment, payment, and authority evidence.

#### Exception and recovery states
- Determination unresolved
- Approval invalidated
- Authority outcome uncertain
- Submission rejected
- Adjustment posting failed
- Payment failed/returned
- Sensitive evidence access denied

#### Completion condition
Return/amendment and payment obligations have explicit independent statuses and evidence.

#### UX acceptance criteria
- **UXA-7-10-01:** An accepted return is never edited to represent an amendment.
- **UXA-7-10-02:** Payment failure does not change filing acceptance status.
- **UXA-7-10-03:** Authority uncertainty is reconciled before another submission is created.
- **UXA-7-10-04:** Return-level adjustment posting remains independently retryable and traceable.

<a id="workflow-wf-7-11"></a>
### 8.18 WF-7.11 — Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment

- **User goal:** Regular or off-cycle run progresses Draft -> Calculated -> PendingApproval -> Approved -> Posted -> PaymentPending -> Settled, with Rejected, employee-level payment failure, and linked correction-run alternatives
- **Primary actor/context:** Payroll Administrator, Workflow approver, employee-payment operator, Tax Filing, and Payments
- **Ownership:** Payroll Administrator, Workflow approver, employee-payment operator, Tax Filing, and Payments. Payroll owns calculations, ApplyPayrollRunApprovalDecision, corrections, payroll liabilities, and payroll posting requests; Workflow owns the approval decision, Payments owns cash execution, and Tax Filing owns statutory filing and amendment status.
- **Entry point:** From the payroll calendar, off-cycle request, failed payment, or correction case.
- **Primary screens:** `PAYR-WS-01`, `PAYR-SCR-01`, `PAYR-SCR-02`, `WFA-SCR-02`, `GL-SCR-02`, `PCM-SCR-02`, `PAYR-SCR-03`, `PAYR-SCR-05`, `TAX-SCR-04`
- **Direct requirements:** `FR-PAYR-001`, `FR-PAYR-002`, `FR-PAYR-003`, `FR-PAYR-004`, `FR-PAYR-007`
- **Supporting families:** FR-PAYR-*, FR-WFA-*, FR-PCM-*, FR-TAX-*, FR-GL-*
- **DDD source:** §7.11
- **Functional acceptance source:** DDD acceptance §14.13.11; this UX flow does not alter those scenarios.

#### User journey
1. Create a regular, off-cycle, or correction run for pay group/period and load employee profile/tax references.
2. Calculate and review gross, deductions, net, employer tax, variances, exceptions, and confidential employee details under restricted access.
3. Route approval against exact run and employee-result versions; revalidate before application.
4. Post payroll expense/liabilities and show pending/posted/failed results without exposing restricted details in summary views.
5. Create payment obligations and track employee-level or aggregate settlement while preserving payroll calculation ownership.
6. For failed employee payment, keep the obligation outstanding and offer retry/alternate payment rather than reversing payroll expense.
7. For posted calculation correction, create a linked correction/off-cycle run and show impact on payment and tax filing.
8. Create or link the required payroll tax amendment without overwriting the original payroll run or filing.

#### Exception and recovery states
- Employee/profile/tax validation
- Approval invalidated
- Posting failure
- Employee payment failure/return
- Correction after finalization
- Tax amendment required

#### Completion condition
Run is rejected, posted and settled, or linked to explicit outstanding payments/corrections/amendments.

#### UX acceptance criteria
- **UXA-7-11-01:** Detailed employee data is hidden from users with only summary-ledger access.
- **UXA-7-11-02:** Gross minus deductions equals net and variances are visible before approval.
- **UXA-7-11-03:** Failed payment does not reverse the established payroll expense.
- **UXA-7-11-04:** Correction creates a linked run and preserves the finalized original.

<a id="workflow-wf-7-12"></a>
### 8.19 WF-7.12 — Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen

- **User goal:** StartSoftClose creates control epoch 1
- **Primary actor/context:** Controller, close operator, recovery operator, independent approver, Fiscal Period Management, Workflow, and GL
- **Ownership:** Controller, close operator, recovery operator, independent approver, Fiscal Period Management, Workflow, and GL. Fiscal Period Management owns process state and ReopenRequest; Workflow owns approval decisions; GL owns PeriodPostingGate. TakeOverPeriodControl, ExtendCloseException, OpenOperationalReopenGate, CloseOperationalReopenGate, and BeginRecloseGate supplement Sections 6.1 and 6.2.
- **Entry point:** From a period-control alert, expired authority, close/reopen outage, or controller recovery task.
- **Primary screens:** `FPM-WS-01`, `FPM-SCR-04`, `GL-SCR-03`, `WFA-SCR-02`, `FPM-SCR-05`, `FPM-SCR-03`, `FPM-SCR-02`, `AUD-SCR-04`
- **Direct requirements:** `FR-FPM-001`, `FR-FPM-012`, `FR-FPM-013`, `FR-GL-011`, `FR-GL-012`, `FR-GL-013`
- **Supporting families:** FR-FPM-*, FR-GL-*, FR-WFA-*, FR-AUD-*
- **DDD source:** §7.12
- **Functional acceptance source:** DDD acceptance §14.13.12; this UX flow does not alter those scenarios.

#### User journey
1. Open recovery view showing authoritative period/gate state, active owner/epoch, process state, expiry, admission summary, and last successful action.
2. If control authority expired, request takeover with reason/evidence and independent approval; preserve the process identity and prior owner history.
3. Classify late adjustments against cutoff policy and show whether they are allowed close, reopen, or future-period actions.
4. Resolve or extend close exceptions through approval; expired exceptions remain visible blockers.
5. For operational reopen, capture permitted transaction classes/actors and expiry, apply approval, and open `OperationalReopen` only after gate validation.
6. At closure, show the retained admission summary and automatically select `CompletedNoChange` or mandatory reclose.
7. For mandatory reclose, show exclusive ownership transfer and resume from the recorded process/gate state after interruption.
8. Escalate integrity or control anomalies with preserved evidence; never relax the gate merely because a dependency is unavailable.

#### Exception and recovery states
- Dependency unavailable
- Authority expired
- Takeover rejected
- Gate/process mismatch
- Cutoff violation
- Exception expired
- Operational reopen expired
- Interrupted mandatory reclose

#### Completion condition
Restrictive control is preserved and the process resumes to takeover, no-change closure, or completed reclose with evidence.

#### UX acceptance criteria
- **UXA-7-12-01:** Unavailable dependencies never cause the UI to imply the period is open.
- **UXA-7-12-02:** Takeover changes the controller but not the process identity/history.
- **UXA-7-12-03:** The admission summary—not user preference—determines no-change versus reclose.
- **UXA-7-12-04:** A finalized gate cannot be released.

<a id="workflow-wf-7-13"></a>
### 8.20 WF-7.13 — Cross-Context Event Interpretation, Ordering, and Replay

- **User goal:** A receiving context validates the event and then applies it, defers it pending prerequisites, rejects it, or records an exception for authorized resolution
- **Primary actor/context:** Every receiving bounded context owns the interpretation of a published event and the resulting local domain effect
- **Ownership:** Every receiving bounded context owns the interpretation of a published event and the resulting local domain effect. A domain steward owns any policy for authorized deferral, rejection, correction, or replay.
- **Entry point:** From a cross-context event exception, reconciliation discrepancy, or authorized reconstruction task.
- **Primary screens:** `XCT-WS-01`, `XCT-SCR-01`
- **Direct requirements:** `GFR-006`, `GFR-007`, `GFR-008`, `GFR-009`, `GFR-012`, `GFR-013`, `GFR-014`
- **Supporting families:** Each affected receiving-capability family
- **DDD source:** §7.13
- **Functional acceptance source:** DDD acceptance §14.13.13; this UX flow does not alter those scenarios.

#### User journey
1. Show the receiving capability, event identity/fingerprint, semantic contract version, scope, source aggregate/version, correlation, and causation.
2. Show the receiving outcome as Applied, Deferred, Rejected, or Exception with the authoritative reason and local effect reference.
3. For deferred events, list missing prerequisites and automatically refresh eligibility when those prerequisites are established.
4. For rejected or exception events, present only authorized resolution choices: corrected source, approved interpretation, reclassification, or no-effect closure.
5. For reconstruction, choose a known domain position/range, preview affected outcomes, apply ordering rules, and verify resulting business state.
6. Preserve every attempt and result; one event identity can show at most one established local business effect per receiving capability.

#### Exception and recovery states
- Unknown semantic contract
- Invalid scope
- Out-of-order source version
- Missing prerequisite
- Changed fingerprint under same identity
- Reconstruction mismatch

#### Completion condition
Each event has an established receiving outcome or authorized unresolved exception with evidence.

#### UX acceptance criteria
- **UXA-7-13-01:** The UI distinguishes deferred from rejected.
- **UXA-7-13-02:** A changed fingerprint under the same event identity is shown as conflict, not safe repeat.
- **UXA-7-13-03:** Reconstruction never hides or overwrites the original receiving outcome.
- **UXA-7-13-04:** One event identity cannot establish duplicate local business effects.

<a id="workflow-wf-7-14"></a>
### 8.21 WF-7.14 — Concurrent Aggregate and Domain-Process Modification Rules

- **User goal:** The owner validates authorization, expected version, active lifecycle state, and protected-operation flags; it establishes one transition and its domain events or returns a typed conflict with the current version and safe retry guidance
- **Primary actor/context:** Business user, automated actor, and the owning bounded context for invoices, payment instructions, payment batches, close runs, settlement runs, and revenue schedules
- **Ownership:** Business user, automated actor, and the owning bounded context for invoices, payment instructions, payment batches, close runs, settlement runs, and revenue schedules.
- **Entry point:** When a user action or domain process is rejected because the target version, state, or owner changed.
- **Primary screens:** `CON-SCR-01`
- **Direct requirements:** `GFR-006`, `GFR-007`, `GFR-008`, `GFR-009`, `GFR-012`, `GFR-013`, `GFR-014`
- **Supporting families:** Each affected owning-capability family
- **DDD source:** §7.14
- **Functional acceptance source:** DDD acceptance §14.13.14; this UX flow does not alter those scenarios.

#### User journey
1. Show a typed conflict containing the attempted action, expected version/state, current version/state, current owner/process epoch, and whether retry is safe.
2. Offer a comparison between the user-reviewed snapshot and current authoritative values, limited to fields the user may access.
3. For commutative actions, allow refresh and resubmit with a new expected version after explicit confirmation.
4. For conflicting actions, require the user to abandon, re-enter, or choose an allowed correction path; never silently merge.
5. For an ambiguous prior submission, look up the established result by business identity/fingerprint before enabling retry.
6. Show superseded-owner/epoch conflicts as terminal for that actor and link to the active process owner.

#### Exception and recovery states
- Version conflict
- State no longer permits action
- Protected operation active
- Ambiguous prior result
- Superseded process owner/epoch
- Authorization changed during review

#### Completion condition
User sees the established outcome or completes a deliberate safe retry against current state.

#### UX acceptance criteria
- **UXA-7-14-01:** No conflict path silently overwrites current state.
- **UXA-7-14-02:** Safe retry is offered only after established-result lookup.
- **UXA-7-14-03:** The comparison view respects data sensitivity.
- **UXA-7-14-04:** Superseded process actors cannot submit a later transition.

<a id="workflow-wf-7-15"></a>
### 8.22 WF-7.15 — Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation

- **User goal:** Chain state progresses through appended events and periodic seals
- **Primary actor/context:** Auditor, security operator, incident commander, and Audit Integrity
- **Ownership:** Auditor, security operator, incident commander, and Audit Integrity. Audit Integrity owns the state-changing commands AppendAuditableEvent, CreateAuditSeal, RotateVerificationCredential, and EscalateIntegrityIncident. VerifyProof is an authoritative domain reference operation that evaluates evidence without changing aggregate state.
- **Entry point:** From an audit request, scheduled integrity check, proof link, or integrity alert.
- **Primary screens:** `AUD-WS-01`, `AUD-SCR-01`, `AUD-SCR-02`, `AUD-SCR-03`, `AUD-SCR-04`, `AUD-SCR-05`
- **Direct requirements:** `FR-AUD-001`, `FR-AUD-002`, `FR-AUD-003`, `FR-AUD-004`, `FR-AUD-005`
- **Supporting families:** FR-AUD-*, FR-IAM-*
- **DDD source:** §7.15
- **Functional acceptance source:** DDD acceptance §14.13.15; this UX flow does not alter those scenarios.

#### User journey
1. Select AuditScope, sequence/time range, proof/seal, and verification purpose; show access and legal-hold constraints.
2. Run proof verification and show exactly one result: Valid, MissingEvent, ProofMismatch, InvalidProof, or UnsupportedVersion.
3. For non-valid results, show affected range, missing/mismatched evidence, verification credential, prior seals, and source references without exposing secret material.
4. Open an integrity incident with severity, owner, containment, affected business evidence, and required recovery actions.
5. Recover missing evidence only from authoritative source events and show validation before resealing.
6. Rotate verification credentials by closing the prior interval and beginning a new one; show impact analysis and replacement-seal requirements.
7. Create corrective seals only where policy permits and show that they supersede proof results, never source events.

#### Exception and recovery states
- Missing event
- Proof mismatch
- Invalid proof
- Unsupported integrity version
- Credential compromise
- Recovery evidence incomplete

#### Completion condition
Verification is Valid or an incident remains explicitly open with preserved evidence, containment, and recovery plan.

#### UX acceptance criteria
- **UXA-7-15-01:** Verification presents one unambiguous typed result.
- **UXA-7-15-02:** Secret credential material is never displayed or exported.
- **UXA-7-15-03:** Recovered evidence is linked to authoritative sources and is not fabricated.
- **UXA-7-15-04:** Credential rotation and corrective seals preserve prior history.

<a id="section-9"></a>
## 9. Notifications, Messages, and Recovery

### 9.1 Notification events

| Notification class | Recipients | Minimum content |
|---|---|---|
| Assigned work | Current owner or assigned role | Record identity, scope, task, due date where defined, priority, and direct link. |
| Approval requested/delegated/escalated | Assigned approver/delegate and oversight roles | Subject snapshot/version, policy, requester, amount/scope, due/escalation status, and decision link. |
| Business rejection or invalidation | Initiator and current owner | Typed reason, authoritative state/version, changed conditions, and allowed next action. |
| Posting/settlement/reconciliation exception | Owning operational role and exception owner | Amount/currency, affected record, current stage, evidence, aging, and resolution link. |
| Period-control blocker/expiry | Close/reopen owner, controller, and assigned resolver | Period/scope, gate/owner, blocker, expiry/cutoff, admissions, and required action. |
| Provider/authority outcome | Owning business role | Provider/authority reference, business record, outcome, date/time, and reconciliation/correction action. |
| Integrity incident | Integrity, security, audit, and designated business owners | Audit scope/range, typed verification result, severity, containment, and incident link. |
| Completion/publication | Initiator and subscribed authorized roles | Final state, key references, statement/evidence links, and any remaining nonblocking condition. |

Notifications never expose restricted payroll, tax, bank, personal, or credential details. A notification is a link to the authoritative record, not an alternate action surface unless the action is explicitly permitted from that context.

### 9.2 Standard message semantics

- **Completed:** The authoritative owner established the requested business outcome.
- **Pending approval:** No approval-bearing business transition has been applied yet.
- **Pending external or cross-capability outcome:** The local intent exists, but the owning downstream outcome is not authoritative yet.
- **Partially completed:** Some independent obligations/effects are established; remaining items are listed and retain their own states.
- **Rejected:** The authoritative business rule or applied approval decision rejected the action; no implied retry without changed conditions.
- **Unavailable:** A required dependency cannot currently provide evidence; no business rejection or success is implied.
- **Conflict:** The expected identity/content/version/state differs from authoritative state; a deliberate new action or safe retry is required.
- **Exception:** A durable business exception exists with an assigned owner and authorized resolution path.

<a id="section-10"></a>
## 10. Accessibility, Privacy, and Evidence

### 10.1 Accessibility requirements
- All workflows are operable with keyboard-only navigation and expose a logical focus order.
- State, severity, approval, and reconciliation distinctions do not rely on color alone; text labels and semantic status are always present.
- Tables expose headers, row identity, selection state, sorting, filtering, validation, and pagination semantics to assistive technology.
- Errors are announced, summarized, and linked to the affected field or line; focus moves predictably after submission.
- Dynamic process updates expose a readable status change without unexpectedly moving focus.
- Confirmation and conflict dialogs identify the action, record, scope, outcome, and safe exit path.

### 10.2 Privacy and sensitivity
- Payroll-detail, tax, bank, personal, and security-sensitive sections are separately permissioned and clearly labeled.
- Masked values remain masked in worklists, notifications, exports, and shared evidence unless the user has explicit detail access.
- Evidence exports apply the same row/field permissions as the interactive view and record the exporting actor and scope.
- Audit and integrity views display fingerprints, identifiers, and verification references without revealing secret credential material.

### 10.3 Evidence and legal hold
- Every material outcome provides a stable evidence link to the authoritative record and related source/decision/effect/correction records.
- Legal hold is displayed independently from business lifecycle state and prevents destruction while permitting authorized business corrections.
- Exported evidence identifies the source record/version, scope, generation time, actor, applied filters, and sensitivity classification.

<a id="section-11"></a>
## 11. UX Acceptance and Traceability

### 11.1 Cross-cutting UX acceptance

- **UXA-COMMON-001:** Every record detail shows authoritative capability, scope, lifecycle state, version, owner, allowed actions, blocked actions, and correction/recovery path.
- **UXA-COMMON-002:** Every approval-bearing action distinguishes decision recorded from decision applied and shows current-state revalidation.
- **UXA-COMMON-003:** Established financial facts remain visible and immutable; correction lineage is linked and navigable.
- **UXA-COMMON-004:** Safe repeat, identity-content conflict, concurrency conflict, dependency unavailable, domain rejection, and managed exception use distinct messages and next actions.
- **UXA-COMMON-005:** Transaction, functional, and presentation currency are never presented as one unlabeled amount where more than one applies.
- **UXA-COMMON-006:** Cross-capability process views identify the owner of each step and link to—not replace—the authoritative record.
- **UXA-COMMON-007:** Sensitive values remain masked and unauthorized users cannot infer them from errors, notifications, exports, or comparisons.
- **UXA-COMMON-008:** All worklists support the applicable scope, state, owner, date, amount, currency, exception, and approval filters.
- **UXA-COMMON-009:** Evidence exports preserve source/version/scope/filter context and are access controlled and auditable.
- **UXA-COMMON-010:** Keyboard, focus, status, table, validation, and dialog behavior meets the accessibility requirements in §10.1.

### 11.2 Workflow traceability matrix

| Workflow | UX section | Primary screens | Direct requirement IDs | Functional acceptance source |
|---|---|---|---|---|
| `WF-6.1` | [§8.1](#workflow-wf-6-1) | `FPM-WS-01`, `FPM-SCR-02`, `GL-SCR-03`, `FPM-SCR-05`, `WFA-SCR-02`, `RPT-SCR-03`, `AUD-SCR-02` | `FR-FPM-003`, `FR-FPM-006`, `FR-FPM-007`, `FR-FPM-008`, `FR-GL-006`, `FR-GL-007`, `FR-GL-008`, `FR-GL-014`, `FR-RPT-006` | DDD acceptance §§14.3 and 14.8 |
| `WF-6.2` | [§8.2](#workflow-wf-6-2) | `FPM-WS-01`, `FPM-SCR-03`, `WFA-SCR-02`, `GL-SCR-03`, `GL-WS-01`, `FPM-SCR-02`, `RPT-SCR-03`, `AUD-SCR-02` | `FR-FPM-006`, `FR-FPM-009`, `FR-FPM-010`, `FR-FPM-011`, `FR-GL-001`, `FR-GL-007`, `FR-GL-008`, `FR-GL-009`, `FR-GL-010`, `FR-GL-013`, `FR-GL-014` | DDD acceptance §§14.10 and 14.8 |
| `WF-6.3` | [§8.3](#workflow-wf-6-3) | `IC-WS-01`, `IC-SCR-01`, `IC-SCR-03`, `IC-SCR-04`, `WFA-SCR-02`, `PCM-SCR-02`, `PCM-SCR-04`, `RPT-SCR-02`, `IC-SCR-05` | `FR-IC-001`, `FR-IC-002`, `FR-IC-003`, `FR-IC-004`, `FR-IC-005` | DDD acceptance §14.11 |
| `WF-6.4` | [§8.4](#workflow-wf-6-4) | `FA-WS-01`, `FA-SCR-01`, `FA-SCR-06`, `WFA-SCR-02`, `FA-SCR-07`, `GL-SCR-02`, `PCM-SCR-04`, `PCM-SCR-02`, `AP-SCR-01` | `FR-FA-005`, `FR-FA-007`, `FR-FA-008` | DDD acceptance §§14.2 and 14.13.7 |
| `WF-6.5` | [§8.5](#workflow-wf-6-5) | `REV-WS-01`, `REV-SCR-01`, `REV-SCR-02`, `REV-SCR-03`, `WFA-SCR-02`, `REV-SCR-04`, `INV-SCR-05`, `AR-SCR-01`, `REV-SCR-06`, `RPT-SCR-03` | `FR-AR-001`, `FR-REV-001`, `FR-REV-002`, `FR-REV-003`, `FR-REV-004`, `FR-REV-005`, `FR-REV-006` | DDD acceptance §§14.12 and 14.13.8 |
| `WF-6.6` | [§8.6](#workflow-wf-6-6) | `GL-WS-01`, `GL-SCR-01`, `GL-SCR-02`, `WFA-SCR-02`, `GL-SCR-03` | `FR-GL-001`, `FR-GL-002`, `FR-GL-003` | DDD acceptance §§14.1 and 14.9 |
| `WF-6.7` | [§8.7](#workflow-wf-6-7) | `AR-WS-01`, `AR-SCR-02`, `AR-SCR-03`, `AR-SCR-04`, `BFR-SCR-05`, `GL-SCR-02` | `FR-AR-002`, `FR-AR-003`, `FR-AR-004`, `FR-AR-005` | DDD acceptance §14.6 |
| `WF-7.1` | [§8.8](#workflow-wf-7-1) | `AP-WS-01`, `AP-SCR-01`, `AP-SCR-02`, `WFA-SCR-02`, `AP-SCR-03`, `GL-SCR-02`, `AP-SCR-04` | `FR-AP-001`, `FR-AP-006`, `FR-AP-008`, `FR-AP-009`, `FR-AP-010` | DDD acceptance §14.13.1 |
| `WF-7.2` | [§8.9](#workflow-wf-7-2) | `PCM-WS-01`, `PCM-SCR-01`, `WFA-SCR-02`, `PCM-SCR-02`, `PCM-SCR-03`, `PCM-SCR-04`, `PCM-SCR-05`, `PCM-SCR-06`, `GL-SCR-02` | `FR-PCM-003` | DDD acceptance §14.13.2 |
| `WF-7.3` | [§8.10](#workflow-wf-7-3) | `AR-WS-01`, `AR-SCR-05`, `AR-SCR-06`, `WFA-SCR-02`, `PCM-SCR-02`, `PCM-SCR-03`, `GL-SCR-02` | `FR-AR-006`, `FR-AR-007`, `FR-AR-008`, `FR-AR-009`, `FR-AR-010`, `FR-AR-011`, `FR-AR-012`, `FR-AR-013`, `FR-AR-014`, `FR-AR-015`, `FR-AR-016` | DDD acceptance §14.13.3 |
| `WF-7.4` | [§8.11](#workflow-wf-7-4) | `BFR-WS-01`, `BFR-SCR-01`, `BFR-SCR-02`, `BFR-SCR-03`, `BFR-SCR-05`, `BFR-SCR-04` | `FR-BFR-001`, `FR-BFR-002`, `FR-BFR-003`, `FR-BFR-004`, `FR-BFR-005` | DDD acceptance §14.13.4 |
| `WF-7.5` | [§8.12](#workflow-wf-7-5) | `AP-SCR-01`, `AR-SCR-01`, `FX-SCR-04`, `PCM-SCR-02`, `AR-SCR-02`, `GL-SCR-02` | None | DDD acceptance §14.13.5 |
| `WF-7.6` | [§8.13](#workflow-wf-7-6) | `FX-WS-01`, `FX-SCR-02`, `FX-SCR-01`, `WFA-SCR-02`, `GL-SCR-02`, `FPM-WS-01` | `FR-FX-002`, `FR-FX-003`, `FR-FX-004` | DDD acceptance §14.13.6 |
| `WF-7.7` | [§8.14](#workflow-wf-7-7) | `FA-WS-01`, `FA-SCR-02`, `FA-SCR-03`, `FA-SCR-04`, `FA-SCR-05`, `FA-SCR-06`, `FA-SCR-07` | `FR-FA-001`, `FR-FA-002`, `FR-FA-003`, `FR-FA-004`, `FR-FA-005`, `FR-FA-006`, `FR-FA-007`, `FR-FA-008`, `FR-FA-018`, `FR-FA-019`, `FR-FA-020`, `FR-FA-021` | DDD acceptance §14.13.7 |
| `WF-7.8` | [§8.15](#workflow-wf-7-8) | `REV-SCR-01`, `REV-SCR-05`, `REV-SCR-02`, `REV-SCR-03`, `WFA-SCR-02`, `REV-SCR-04`, `INV-SCR-05`, `AR-SCR-05`, `REV-SCR-06` | `FR-REV-004`, `FR-REV-005` | DDD acceptance §14.13.8 |
| `WF-7.9` | [§8.16](#workflow-wf-7-9) | `RPT-WS-01`, `RPT-SCR-02`, `FX-SCR-03`, `IC-SCR-05`, `RPT-SCR-04`, `WFA-SCR-02`, `RPT-SCR-03` | `FR-RPT-001`, `FR-RPT-003`, `FR-RPT-004` | DDD acceptance §14.13.9 |
| `WF-7.10` | [§8.17](#workflow-wf-7-10) | `TAX-WS-01`, `TAX-SCR-02`, `TAX-SCR-03`, `WFA-SCR-02`, `TAX-SCR-04`, `TAX-SCR-05`, `TAX-SCR-06`, `PCM-SCR-02` | `FR-TAX-010` | DDD acceptance §14.13.10 |
| `WF-7.11` | [§8.18](#workflow-wf-7-11) | `PAYR-WS-01`, `PAYR-SCR-01`, `PAYR-SCR-02`, `WFA-SCR-02`, `GL-SCR-02`, `PCM-SCR-02`, `PAYR-SCR-03`, `PAYR-SCR-05`, `TAX-SCR-04` | `FR-PAYR-001`, `FR-PAYR-002`, `FR-PAYR-003`, `FR-PAYR-004`, `FR-PAYR-007` | DDD acceptance §14.13.11 |
| `WF-7.12` | [§8.19](#workflow-wf-7-12) | `FPM-WS-01`, `FPM-SCR-04`, `GL-SCR-03`, `WFA-SCR-02`, `FPM-SCR-05`, `FPM-SCR-03`, `FPM-SCR-02`, `AUD-SCR-04` | `FR-FPM-001`, `FR-FPM-012`, `FR-FPM-013`, `FR-GL-011`, `FR-GL-012`, `FR-GL-013` | DDD acceptance §14.13.12 |
| `WF-7.13` | [§8.20](#workflow-wf-7-13) | `XCT-WS-01`, `XCT-SCR-01` | `GFR-006`, `GFR-007`, `GFR-008`, `GFR-009`, `GFR-012`, `GFR-013`, `GFR-014` | DDD acceptance §14.13.13 |
| `WF-7.14` | [§8.21](#workflow-wf-7-14) | `CON-SCR-01` | `GFR-006`, `GFR-007`, `GFR-008`, `GFR-009`, `GFR-012`, `GFR-013`, `GFR-014` | DDD acceptance §14.13.14 |
| `WF-7.15` | [§8.22](#workflow-wf-7-15) | `AUD-WS-01`, `AUD-SCR-01`, `AUD-SCR-02`, `AUD-SCR-03`, `AUD-SCR-04`, `AUD-SCR-05` | `FR-AUD-001`, `FR-AUD-002`, `FR-AUD-003`, `FR-AUD-004`, `FR-AUD-005` | DDD acceptance §14.13.15 |

### 11.3 Capability traceability matrix

| Capability | UX section | Screen count | Requirement count | Requirement IDs |
|---|---|---:|---:|---|
| Organization & Master Data | [§7.1](#capability-omd) | 6 | 6 | `FR-OMD-001`, `FR-OMD-002`, `FR-OMD-003`, `FR-OMD-004`, `FR-OMD-005`, `FR-OMD-006` |
| General Ledger (GL) | [§7.2](#capability-gl) | 6 | 18 | `FR-GL-001`, `FR-GL-002`, `FR-GL-003`, `FR-GL-004`, `FR-GL-005`, `FR-GL-006`, `FR-GL-007`, `FR-GL-008`, `FR-GL-009`, `FR-GL-010`, `FR-GL-011`, `FR-GL-012`, `FR-GL-013`, `FR-GL-014`, `FR-GL-015`, `FR-GL-016`, `FR-GL-017`, `FR-GL-018` |
| Accounts Payable (AP) | [§7.3](#capability-ap) | 6 | 10 | `FR-AP-001`, `FR-AP-002`, `FR-AP-003`, `FR-AP-004`, `FR-AP-005`, `FR-AP-006`, `FR-AP-007`, `FR-AP-008`, `FR-AP-009`, `FR-AP-010` |
| Accounts Receivable (AR) | [§7.4](#capability-ar) | 7 | 16 | `FR-AR-001`, `FR-AR-002`, `FR-AR-003`, `FR-AR-004`, `FR-AR-005`, `FR-AR-006`, `FR-AR-007`, `FR-AR-008`, `FR-AR-009`, `FR-AR-010`, `FR-AR-011`, `FR-AR-012`, `FR-AR-013`, `FR-AR-014`, `FR-AR-015`, `FR-AR-016` |
| Payroll | [§7.5](#capability-payr) | 6 | 7 | `FR-PAYR-001`, `FR-PAYR-002`, `FR-PAYR-003`, `FR-PAYR-004`, `FR-PAYR-005`, `FR-PAYR-006`, `FR-PAYR-007` |
| Invoicing | [§7.6](#capability-inv) | 6 | 6 | `FR-INV-001`, `FR-INV-002`, `FR-INV-003`, `FR-INV-004`, `FR-INV-005`, `FR-INV-006` |
| Payments & Cash Management | [§7.7](#capability-pcm) | 8 | 25 | `FR-PCM-001`, `FR-PCM-002`, `FR-PCM-003`, `FR-PCM-004`, `FR-PCM-005`, `FR-PCM-006`, `FR-PCM-007`, `FR-PCM-008`, `FR-PCM-009`, `FR-PCM-010`, `FR-PCM-011`, `FR-PCM-012`, `FR-PCM-013`, `FR-PCM-014`, `FR-PCM-015`, `FR-PCM-016`, `FR-PCM-017`, `FR-PCM-018`, `FR-PCM-019`, `FR-PCM-020`, `FR-PCM-021`, `FR-PCM-022`, `FR-PCM-023`, `FR-PCM-024`, `FR-PCM-025` |
| Financial Reporting | [§7.8](#capability-rpt) | 5 | 6 | `FR-RPT-001`, `FR-RPT-002`, `FR-RPT-003`, `FR-RPT-004`, `FR-RPT-005`, `FR-RPT-006` |
| Multi-Entity / Intercompany | [§7.9](#capability-ic) | 6 | 11 | `FR-IC-001`, `FR-IC-002`, `FR-IC-003`, `FR-IC-004`, `FR-IC-005`, `FR-IC-006`, `FR-IC-007`, `FR-IC-008`, `FR-IC-009`, `FR-IC-010`, `FR-IC-011` |
| Revenue Recognition | [§7.10](#capability-rev) | 7 | 6 | `FR-REV-001`, `FR-REV-002`, `FR-REV-003`, `FR-REV-004`, `FR-REV-005`, `FR-REV-006` |
| Fixed Assets | [§7.11](#capability-fa) | 8 | 21 | `FR-FA-001`, `FR-FA-002`, `FR-FA-003`, `FR-FA-004`, `FR-FA-005`, `FR-FA-006`, `FR-FA-007`, `FR-FA-008`, `FR-FA-009`, `FR-FA-010`, `FR-FA-011`, `FR-FA-012`, `FR-FA-013`, `FR-FA-014`, `FR-FA-015`, `FR-FA-016`, `FR-FA-017`, `FR-FA-018`, `FR-FA-019`, `FR-FA-020`, `FR-FA-021` |
| Multi-Currency | [§7.12](#capability-fx) | 5 | 5 | `FR-FX-001`, `FR-FX-002`, `FR-FX-003`, `FR-FX-004`, `FR-FX-005` |
| Fiscal Period Management | [§7.13](#capability-fpm) | 6 | 13 | `FR-FPM-001`, `FR-FPM-002`, `FR-FPM-003`, `FR-FPM-004`, `FR-FPM-005`, `FR-FPM-006`, `FR-FPM-007`, `FR-FPM-008`, `FR-FPM-009`, `FR-FPM-010`, `FR-FPM-011`, `FR-FPM-012`, `FR-FPM-013` |
| COA Segment Accounting | [§7.14](#capability-coa) | 5 | 5 | `FR-COA-001`, `FR-COA-002`, `FR-COA-003`, `FR-COA-004`, `FR-COA-005` |
| Bank Feeds & Reconciliation | [§7.15](#capability-bfr) | 6 | 6 | `FR-BFR-001`, `FR-BFR-002`, `FR-BFR-003`, `FR-BFR-004`, `FR-BFR-005`, `FR-BFR-006` |
| Tax Filing | [§7.16](#capability-tax) | 7 | 16 | `FR-TAX-001`, `FR-TAX-002`, `FR-TAX-003`, `FR-TAX-004`, `FR-TAX-005`, `FR-TAX-006`, `FR-TAX-007`, `FR-TAX-008`, `FR-TAX-009`, `FR-TAX-010`, `FR-TAX-011`, `FR-TAX-012`, `FR-TAX-013`, `FR-TAX-014`, `FR-TAX-015`, `FR-TAX-016` |
| Workflow & Approvals | [§7.17](#capability-wfa) | 5 | 5 | `FR-WFA-001`, `FR-WFA-002`, `FR-WFA-003`, `FR-WFA-004`, `FR-WFA-005` |
| Identity & Access | [§7.18](#capability-iam) | 6 | 6 | `FR-IAM-001`, `FR-IAM-002`, `FR-IAM-003`, `FR-IAM-004`, `FR-IAM-005`, `FR-IAM-006` |
| Audit Integrity | [§7.19](#capability-aud) | 6 | 5 | `FR-AUD-001`, `FR-AUD-002`, `FR-AUD-003`, `FR-AUD-004`, `FR-AUD-005` |

<a id="section-12"></a>
## 12. Non-Goals and Dependencies

This document does not define:
- Brand, color, typography, visual design tokens, or final page layout.
- APIs, event payloads, databases, queues, middleware, service boundaries, infrastructure, deployment, or observability implementation.
- Performance targets, availability targets, migration execution, operational runbooks, or release planning.
- New business states, commands, events, accounting equations, or capability ownership beyond the verified DDD and Functional PRD baselines.

Dependencies and assumptions:
- The DDD v3.1 and Functional PRD v1.5 checkpoints remain unchanged.
- Procurement, external payment/bank providers, tax authorities, rate providers, and schedulers provide evidence or triggers as described by the source baselines.
- Country-specific localization and statutory detail require separately approved functional and UX extensions.
- Detailed wireframes and visual prototypes may refine layout and interaction presentation but must retain the screen responsibilities, state semantics, and traceability in this specification.

<a id="section-13"></a>
## 13. Verification Checkpoint

| Checkpoint field | Value |
|---|---|
| Verified body SHA-256 | `7bf6e021b60c4a441d32c7fb1e0b14f501000168b544fb108afaff87a92a8891` |
| Hash boundary | UTF-8 bytes from title through the blank line immediately preceding §13; checkpoint section excluded |
| Checkpoint ID | UXWF-1.0-2026-07-24 |
| Source DDD checkpoint | DDD-3.1-2026-07-24 |
| Source Functional PRD checkpoint | FPRD-1.5-2026-07-24 |
| Source Functional PRD verified body SHA-256 | `f5b7be0973f532851abf06cf8c408caf110dad538735aa21490c9ff8b84a2b8f` |
| Source Requirements Catalog verified body SHA-256 | `76bb4ec1ecb155a08df478ea9d4101d40643c6eb372c34ae4cd85f9d82a2c69a` |
| Source Traceability/Acceptance verified body SHA-256 | `6489ac5a5d00562b15e600294d72175addde25d55aaaee3568d02537a8354f4a` |
| Capabilities | 19 |
| Capability requirements covered | 193 |
| Global requirements covered | 22 |
| Capability screen definitions | 117 |
| Shared operational surfaces | 3 |
| Total screen definitions | 120 |
| Functional workflows | 22 |
| UX workflow acceptance criteria | 89 |
| Cross-cutting UX acceptance criteria | 10 |
| Total UX acceptance criteria | 99 |
| Independent second-pass validation | Passed |
| Review result | Passed |
| Review reuse rule | When this body hash and all source hashes remain unchanged, repeat structural/hash validation only. Re-run affected capability/workflow review when requirement meaning, screen responsibility, workflow semantics, or source hashes change. |

### 13.1 Validation gates

| Validation gate | Result |
|---|---|
| Source checkpoint hashes match | Passed |
| 19 capabilities represented | Passed |
| 22 workflows represented | Passed |
| All source direct requirement IDs exist | Passed |
| All requirement references are valid | Passed |
| All referenced screen IDs are defined | Passed |
| All internal links resolve | Passed |
| Screen IDs are unique | Passed |
| No unresolved placeholders | Passed |
| No solution-design vocabulary outside stated boundary | Passed |
| Exact functional acceptance sources mapped | Passed |
| No vague actor placeholders | Passed |
| Markdown code fences balanced | Passed |

### 13.2 Independent second-pass validation

| Independent validation gate | Result |
|---|---|
| Checkpoint body hash matches | Passed |
| All 193 capability requirement IDs are represented | Passed |
| All 22 global requirement IDs are represented | Passed |
| All source direct requirement IDs appear in their workflow sections | Passed |
| All 22 functional acceptance-source mappings match the source traceability document | Passed |
| 120 screen IDs are unique, defined, and referenced consistently | Passed |
| 18 shared component IDs are unique | Passed |
| 99 UX acceptance IDs are unique | Passed |
| Anchors, internal links, H1–H3 headings, and Markdown table columns are valid | Passed |
| Cross-context event resolution remains owned by the receiving capability | Passed |
| Concurrency resolution remains owned by the affected business owner, not Identity & Access | Passed |
| No generic acceptance fallback, vague actor placeholder, unresolved marker, or unbalanced code fence remains | Passed |
