# Finance Platform Work Breakdown and Backlog

| Document-control field | Value |
|---|---|
| Version | 1.0 |
| Baseline date | 2026-07-24 |
| Status | Passed |
| Source baseline | Finance DDD v3.1; Functional PRD v1.5; UX v1.0; NFR v1.0; Solution/System Design v1.0; Technical Specifications v1.0 |
| Delivery profile | Solo, part-time learning project; local-first; low-cost Azure demonstrations |
| Owner | Learning Product and Delivery Owner |

> **Purpose:** Define epics, requirement delivery items, workflow demonstrations, cross-cutting work and readiness/done criteria with stable delivery identifiers.
>
> **Planning rule:** Iteration counts and elapsed-time ranges are planning assumptions, not commitments. Reforecast after M2 using observed completion rate, defect rate and available learning time.

## 1. Backlog conventions

| Item type | Identifier | Meaning |
|---|---|---|
| Epic | `EP-*` | Coherent capability or platform outcome |
| Functional delivery item | `DLV-FR-*` | One exact `FR-*` requirement delivered through all layers |
| Global delivery item | `DLV-GFR-*` | One cross-cutting `GFR-*` control |
| Workflow demonstration | `DLV-WF-*` | End-to-end workflow and mapped acceptance evidence |
| Quality qualification item | `DLV-NFR-*` | One exact NFR verification obligation |
| Platform item | `DLV-PLAT-*` | Engineering or environment foundation not represented as product behavior |

Priority meanings: `P0` blocks the current milestone; `P1` is required for the declared capability; `P2` belongs to the full-domain path and may be deferred at an approved stopping point.

## 2. Epic catalog

| Epic | Outcome | Milestone | Scope |
|---|---|---|---|
| EP-PLAT-001 | Engineering foundation | M0 | Repository, Go/React applications, Docker Compose, migrations, sqlc, OpenAPI, testing and conventions. |
| EP-UX-001 | Shared UX and design system | M0 | Tailwind, daisyUI abstractions, routing, forms, tables, accessibility and shared operational surfaces. |
| EP-IAC-001 | Terraform and Azure learning environment | M0 | Terraform state bootstrap, low-cost Azure modules, budget controls and ephemeral demo deployment. |
| EP-OPS-001 | Observability and operational foundation | M0 | Structured logs, traces, metrics, dashboards, runbooks and operational evidence. |
| EP-IAM-001 | Identity and access | M1 | Entra authentication, application permissions, accounting-scope authorization and emergency access. |
| EP-OMD-001 | Organization and master data | M1 | Legal entities, parties, profiles and fiscal calendars. |
| EP-COA-001 | COA segment configuration | M1 | Segment definitions, values, combinations and approved segment changes. |
| EP-GL-001 | General Ledger | M2 | Journal validation, approval, posting, reversal, gates and ledger inquiry. |
| EP-WFA-001 | Workflow and approvals | M3 | Policies, requests, decisions, delegation, escalation and decision application. |
| EP-FPM-001 | Fiscal period management | M3 | Soft close, hard close, reopen, reclose and control recovery. |
| EP-INV-001 | Invoicing | M4 | Templates, schedules, generated invoices and AR handoff. |
| EP-AR-001 | Accounts Receivable | M4 | Invoices, open items, receipts, applications, credits, refunds and adjustments. |
| EP-AP-001 | Accounts Payable | M5 | Vendor invoices, matching, approval, liabilities and payment requests. |
| EP-PCM-001 | Payments and cash management | M5 | Batches, instructions, settlements, returns, exceptions and expected incoming settlement. |
| EP-BFR-001 | Bank feeds and reconciliation | M6 | Connections, imports, matching, unmatching and reconciliation. |
| EP-FA-001 | Fixed Assets | M7 | Capitalization, depreciation, impairment, transfer, split and disposal. |
| EP-REV-001 | Revenue Recognition | M7 | Contracts, obligations, profiles, schedules and modifications. |
| EP-FX-001 | Multi-Currency | M8 | Rates, realized FX, revaluation and translation. |
| EP-IC-001 | Intercompany | M8 | Agreements, transactions, matching, settlement and eliminations. |
| EP-RPT-001 | Financial Reporting | M8 | Definitions, statements, consolidation, lineage and publication. |
| EP-TAX-001 | Tax Filing | M9 | Configurations, returns, submissions, amendments, adjustments and payments. |
| EP-PAYR-001 | Payroll | M9 | Profiles, runs, corrections, off-cycle processing, failed payments and filing amendments. |
| EP-AUD-001 | Audit Integrity | M9 | Evidence ingestion, verification, credential rotation, incidents and proof access. |
| EP-QUAL-001 | Full-system qualification | M9 | Security, privacy, accessibility, capacity, performance, recovery and release evidence. |

## 3. Platform foundation backlog

| Delivery ID | Epic | Milestone | Deliverable | Exit evidence |
|---|---|---|---|---|
| DLV-PLAT-001 | EP-PLAT-001 | M0 | Create monorepo with Go API, React application and shared commands. | Clean clone builds and tests locally. |
| DLV-PLAT-002 | EP-PLAT-001 | M0 | Create Docker Compose PostgreSQL development environment. | Database starts, migrates, seeds and resets reproducibly. |
| DLV-PLAT-003 | EP-PLAT-001 | M0 | Establish Goose migrations, pgx and sqlc workflow. | CI detects migration and generated-code drift. |
| DLV-PLAT-004 | EP-PLAT-001 | M0 | Establish OpenAPI-first REST workflow. | OpenAPI validates and generated clients compile. |
| DLV-PLAT-005 | EP-PLAT-001 | M0 | Implement shared money, currency, accounting-scope, identity and version primitives. | Unit and serialization tests pass without floating-point money. |
| DLV-PLAT-006 | EP-PLAT-001 | M0 | Implement request fingerprint and idempotency foundation. | Same identity/same fingerprint and changed-fingerprint tests pass. |
| DLV-PLAT-007 | EP-PLAT-001 | M0 | Implement PostgreSQL outbox/inbox and worker foundation. | Crash-before/after-commit and duplicate-delivery tests pass. |
| DLV-UX-001 | EP-UX-001 | M0 | Implement Tailwind/daisyUI application shell and component abstractions. | Shared worklist, detail, action, status, form and dialog examples pass. |
| DLV-UX-002 | EP-UX-001 | M0 | Implement accessibility test harness. | axe, keyboard and manual screen-reader check procedure is repeatable. |
| DLV-OPS-001 | EP-OPS-001 | M0 | Implement structured logging, correlation and OpenTelemetry. | Trace spans and redacted logs link UI, API, database and worker activity. |
| DLV-OPS-002 | EP-OPS-001 | M0 | Create baseline operational dashboard and runbook template. | Health, error, backlog and business-result panels are visible. |
| DLV-IAC-001 | EP-IAC-001 | M0 | Create Terraform modules and local state bootstrap. | fmt, validate and plan pass with no credentials committed. |
| DLV-IAC-002 | EP-IAC-001 | M0 | Create optional Azure dev/demo deployment. | Static Web App, Container App and PostgreSQL exercise deploy and destroy reproducibly. |
| DLV-CI-001 | EP-PLAT-001 | M0 | Create pull-request CI quality pipeline. | Go, frontend, OpenAPI, SQL, Terraform, security and documentation checks gate merge. |

## 4. Global functional-control backlog

| Delivery ID | Source | Epic | Milestone | Deliverable |
|---|---|---|---|---|
| DLV-GFR-001 | GFR-001 | EP-PLAT-001 | M0 | The product shall use the DDD v3.1 ubiquitous language and bounded-context ownership as the authoritative functional vocabulary. |
| DLV-GFR-002 | GFR-002 | EP-IAM-001 | M1 | Every user action shall be evaluated against the access dimensions applicable to that action, including legal entity, business unit or segment, accou… |
| DLV-GFR-003 | GFR-003 | EP-IAM-001 | M1 | The product shall prevent prohibited segregation-of-duties combinations and shall expose the reason for a denied action. |
| DLV-GFR-004 | GFR-004 | EP-WFA-001 | M2 | Approval-bearing actions shall route through the applicable approval policy and revalidate current business state when the decision is applied. |
| DLV-GFR-005 | GFR-005 | EP-GL-001 | M2 | Posted, accepted, or otherwise established financial facts shall not be edited destructively; correction shall use reversal, adjustment, amendment, r… |
| DLV-GFR-006 | GFR-006 | EP-PLAT-001 | M0 | Repeated submission of the same business identity and fingerprint shall return the established result without repeating the business effect. |
| DLV-GFR-007 | GFR-007 | EP-PLAT-001 | M0 | Reuse of the same business identity with changed functional content shall be rejected as an idempotency conflict. |
| DLV-GFR-008 | GFR-008 | EP-PLAT-001 | M0 | Concurrent changes shall be checked against expected versions and shall never silently overwrite an established business outcome. |
| DLV-GFR-009 | GFR-009 | EP-UX-001 | M0 | Each user-visible workflow shall expose current state, allowed actions, blocked actions, blocking reason, responsible owner, and correction or recove… |
| DLV-GFR-010 | GFR-010 | EP-FX-001 | M8 | Financial amounts shall display transaction, functional, and presentation currency where applicable, including the rate-set and conversion evidence u… |
| DLV-GFR-011 | GFR-011 | EP-GL-001 | M2 | Every accounting effect shall have one owning producer, and the product shall prevent duplicate accounting ownership across capabilities. |
| DLV-GFR-012 | GFR-012 | EP-OPS-001 | M0 | Cross-context workflows shall expose intermediate, exception, reconciliation, and terminal states rather than presenting later outcomes as immediate… |
| DLV-GFR-013 | GFR-013 | EP-PLAT-001 | M0 | The product shall preserve immutable lineage among source facts, approvals, postings, reversals, returns, amendments, replacements, and compensations. |
| DLV-GFR-014 | GFR-014 | EP-AUD-001 | M1 | All material actions and decisions shall be auditable with actor, time, scope, source, action, authorization, correlation, and the applicable before/… |
| DLV-GFR-015 | GFR-015 | EP-IAM-001 | M1 | Sensitive payroll, tax, bank, and personal information shall be shown only to authorized users and shall be minimized in shared business evidence and… |
| DLV-GFR-016 | GFR-016 | EP-UX-001 | M0 | Search, worklists, and reports shall support the filters applicable to their records, including accounting scope, state, owner, date, amount, currenc… |
| DLV-GFR-017 | GFR-017 | EP-UX-001 | M0 | User interfaces shall distinguish dependency unavailability from domain rejection and shall show the next permitted resolution action. |
| DLV-GFR-018 | GFR-018 | EP-AUD-001 | M9 | Business records subject to legal hold shall remain available and immutable until the hold is formally released, even when the ordinary retention per… |
| DLV-GFR-019 | GFR-019 | EP-OMD-001 | M1 | Effective-dated configurations shall preserve the rule and version used by historical transactions. |
| DLV-GFR-020 | GFR-020 | EP-AUD-001 | M9 | The product shall provide exportable, access-controlled evidence for approvals, postings, reconciliations, close/reopen activity, and audit-integrity… |
| DLV-GFR-021 | GFR-021 | EP-PLAT-001 | M0 | PRD-defined functional actions shall not be represented as DDD commands or domain events unless the DDD baseline is explicitly changed. |
| DLV-GFR-022 | GFR-022 | EP-PLAT-001 | M0 | Every functional workflow shall trace to exact requirement IDs for named DDD operations and for explicit PRD functional actions that implement stated… |

## 5. Capability requirement backlog

Every row is a complete vertical delivery item, not a backend-only task. It includes UX, API, domain, persistence, authorization, tests, observability and documentation required by the cited specifications.

| Delivery ID | Requirement | Capability | Action | Epic | Milestone | Priority | Short outcome |
|---|---|---|---|---|---|---|---|
| DLV-FR-AP-001 | FR-AP-001 | Accounts Payable | `RegisterVendorInvoice` | EP-AP-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AP-002 | FR-AP-002 | Accounts Payable | `ApplyAssetClearingClassification` | EP-AP-001 | M7 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-AP-003 | FR-AP-003 | Accounts Payable | `ApplyIncomingSettlement` | EP-AP-001 | M6 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-AP-004 | FR-AP-004 | Accounts Payable | `ReverseIncomingSettlementApplication` | EP-AP-001 | M6 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-AP-005 | FR-AP-005 | Accounts Payable | `ApplyPaymentReturn` | EP-AP-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-AP-006 | FR-AP-006 | Accounts Payable | `ApplyVendorInvoiceApprovalDecision` | EP-AP-001 | M5 | P1 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-AP-007 | FR-AP-007 | Accounts Payable | `RequestPayment` | EP-AP-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AP-008 | FR-AP-008 | Accounts Payable | `ValidateVendorInvoice` | EP-AP-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AP-009 | FR-AP-009 | Accounts Payable | `DisputeVendorInvoice` | EP-AP-001 | M5 | P1 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-AP-010 | FR-AP-010 | Accounts Payable | `VoidVendorInvoice` | EP-AP-001 | M5 | P1 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-AR-001 | FR-AR-001 | Accounts Receivable | `IssueCustomerInvoice` | EP-AR-001 | M4 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AR-002 | FR-AR-002 | Accounts Receivable | `RecordReceipt` | EP-AR-001 | M4 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AR-003 | FR-AR-003 | Accounts Receivable | `ApplyReceipt` | EP-AR-001 | M4 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-AR-004 | FR-AR-004 | Accounts Receivable | `UnapplyReceipt` | EP-AR-001 | M4 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-AR-005 | FR-AR-005 | Accounts Receivable | `RollbackUnpostedApplicationBatch` | EP-AR-001 | M4 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-AR-006 | FR-AR-006 | Accounts Receivable | `IssueCreditNote` | EP-AR-001 | M4 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AR-007 | FR-AR-007 | Accounts Receivable | `CreateCustomerRefundRequest` | EP-AR-001 | M4 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AR-008 | FR-AR-008 | Accounts Receivable | `CancelCustomerRefundRequest` | EP-AR-001 | M4 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-AR-009 | FR-AR-009 | Accounts Receivable | `ApplyCustomerRefundApprovalDecision` | EP-AR-001 | M4 | P1 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-AR-010 | FR-AR-010 | Accounts Receivable | `RequestCustomerRefundPayment` | EP-AR-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AR-011 | FR-AR-011 | Accounts Receivable | `CancelCustomerRefundPayment` | EP-AR-001 | M5 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-AR-012 | FR-AR-012 | Accounts Receivable | `ApplyCustomerRefundPaymentResult` | EP-AR-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-AR-013 | FR-AR-013 | Accounts Receivable | `ApplyPaymentReturn` | EP-AR-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-AR-014 | FR-AR-014 | Accounts Receivable | `Resolve customer overpayments` | EP-AR-001 | M5 | P1 | The overpayment amount, remaining unapplied balance, selected resolution, linked applications, refund or reclassif… |
| DLV-FR-AR-015 | FR-AR-015 | Accounts Receivable | `Record customer chargebacks` | EP-AR-001 | M6 | P1 | The chargeback record, restored or reclassified balances, linked accounting result, source evidence, correction li… |
| DLV-FR-AR-016 | FR-AR-016 | Accounts Receivable | `Record receivable write-offs` | EP-AR-001 | M4 | P1 | The write-off amount, remaining open balance, approval evidence, posting status and reference, adjustment lineage,… |
| DLV-FR-AUD-001 | FR-AUD-001 | Audit Integrity | `AppendAuditableEvent` | EP-AUD-001 | M1 | P1 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-AUD-002 | FR-AUD-002 | Audit Integrity | `CreateAuditSeal` | EP-AUD-001 | M9 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AUD-003 | FR-AUD-003 | Audit Integrity | `RotateVerificationCredential` | EP-AUD-001 | M9 | P1 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-AUD-004 | FR-AUD-004 | Audit Integrity | `EscalateIntegrityIncident` | EP-AUD-001 | M9 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-AUD-005 | FR-AUD-005 | Audit Integrity | `VerifyProof` | EP-AUD-001 | M9 | P1 | The authoritative verification outcome, audit scope, evidence range, proof reference, verification-credential refe… |
| DLV-FR-BFR-001 | FR-BFR-001 | Bank Feeds & Reconciliation | `ImportStatement` | EP-BFR-001 | M6 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-BFR-002 | FR-BFR-002 | Bank Feeds & Reconciliation | `ProposeMatch` | EP-BFR-001 | M6 | P1 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-BFR-003 | FR-BFR-003 | Bank Feeds & Reconciliation | `ConfirmMatch` | EP-BFR-001 | M6 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-BFR-004 | FR-BFR-004 | Bank Feeds & Reconciliation | `Unmatch` | EP-BFR-001 | M6 | P1 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-BFR-005 | FR-BFR-005 | Bank Feeds & Reconciliation | `CompleteReconciliation` | EP-BFR-001 | M6 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-BFR-006 | FR-BFR-006 | Bank Feeds & Reconciliation | `Maintain bank-feed connections` | EP-BFR-001 | M6 | P1 | The connection identity, provider, consent status, token-expiry metadata, synchronization cursor and status, last-… |
| DLV-FR-COA-001 | FR-COA-001 | COA Segment Accounting | `Maintain segment definitions` | EP-COA-001 | M1 | P1 | The segment-definition version, effective date, lifecycle status, approval status, and any validation rejection ar… |
| DLV-FR-COA-002 | FR-COA-002 | COA Segment Accounting | `Maintain segment values` | EP-COA-001 | M1 | P1 | The segment-value version, effective date, lifecycle status, approval status, and any validation rejection are vis… |
| DLV-FR-COA-003 | FR-COA-003 | COA Segment Accounting | `Validate segment combinations` | EP-COA-001 | M1 | P1 | The combination-validation result, applicable rule versions, effective-date result, and rejection reasons are disp… |
| DLV-FR-COA-004 | FR-COA-004 | COA Segment Accounting | `Request segment changes` | EP-COA-001 | M1 | P1 | The segment-change request reference, requested effective date, subject version, approval status, and any conflict… |
| DLV-FR-COA-005 | FR-COA-005 | COA Segment Accounting | `ApplySegmentChangeApprovalDecision` | EP-COA-001 | M1 | P1 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FA-001 | FR-FA-001 | Fixed Assets | `CapitalizeAsset` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-002 | FR-FA-002 | Fixed Assets | `CreateAssetAcquisitionClearing` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-003 | FR-FA-003 | Fixed Assets | `RunDepreciation` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-004 | FR-FA-004 | Fixed Assets | `ApplyImpairmentApprovalDecision` | EP-FA-001 | M7 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FA-005 | FR-FA-005 | Fixed Assets | `DisposeAsset` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-006 | FR-FA-006 | Fixed Assets | `ApplyAssetDisposalApprovalDecision` | EP-FA-001 | M7 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FA-007 | FR-FA-007 | Fixed Assets | `CancelUnpostedAssetDisposal` | EP-FA-001 | M7 | P2 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-FA-008 | FR-FA-008 | Fixed Assets | `CompensateFailedDisposalPosting` | EP-FA-001 | M7 | P2 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-FA-009 | FR-FA-009 | Fixed Assets | `CreateDisposalSettlementClearing` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-010 | FR-FA-010 | Fixed Assets | `ApplyAssetSupplierLiabilityResult` | EP-FA-001 | M7 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-FA-011 | FR-FA-011 | Fixed Assets | `ApplyIncomingSettlement` | EP-FA-001 | M7 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-FA-012 | FR-FA-012 | Fixed Assets | `ReverseIncomingSettlementApplication` | EP-FA-001 | M7 | P2 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-FA-013 | FR-FA-013 | Fixed Assets | `ApplyPaymentReturn` | EP-FA-001 | M7 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-FA-014 | FR-FA-014 | Fixed Assets | `ApplyAssetSettlementResult` | EP-FA-001 | M7 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-FA-015 | FR-FA-015 | Fixed Assets | `ReclassifyDisposalCostForPayment` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-016 | FR-FA-016 | Fixed Assets | `RequestDisposalCostPayment` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-017 | FR-FA-017 | Fixed Assets | `RequestDisposalCostPaymentReplacement` | EP-FA-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FA-018 | FR-FA-018 | Fixed Assets | `Record impairment assessments` | EP-FA-001 | M7 | P2 | The impairment-assessment identity, asset scope, recoverable and impairment amounts, assessment date, evidence, ap… |
| DLV-FR-FA-019 | FR-FA-019 | Fixed Assets | `Transfer assets or components` | EP-FA-001 | M7 | P2 | The transfer reference, source and destination classifications, effective date, preserved monetary balances, appro… |
| DLV-FR-FA-020 | FR-FA-020 | Fixed Assets | `Split assets or components` | EP-FA-001 | M7 | P2 | The split reference, source and successor portions, allocation basis, preserved monetary totals, approval and post… |
| DLV-FR-FA-021 | FR-FA-021 | Fixed Assets | `Correct posted asset disposals` | EP-FA-001 | M7 | P2 | The correction request, linked reversals and replacements, preserved original disposal, resulting asset and dispos… |
| DLV-FR-FPM-001 | FR-FPM-001 | Fiscal Period Management | `StartSoftClose` | EP-FPM-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FPM-002 | FR-FPM-002 | Fiscal Period Management | `EndSoftClose` | EP-FPM-001 | M3 | P0 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-FPM-003 | FR-FPM-003 | Fiscal Period Management | `StartHardClose` | EP-FPM-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FPM-004 | FR-FPM-004 | Fiscal Period Management | `ResumeCloseRun` | EP-FPM-001 | M3 | P0 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-FPM-005 | FR-FPM-005 | Fiscal Period Management | `AbortCloseRun` | EP-FPM-001 | M3 | P0 | The authoritative result and resulting business state are visible to authorized users. |
| DLV-FR-FPM-006 | FR-FPM-006 | Fiscal Period Management | `ApplyPostingGateResult` | EP-FPM-001 | M3 | P0 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-FPM-007 | FR-FPM-007 | Fiscal Period Management | `ApplyCloseExceptionApprovalDecision` | EP-FPM-001 | M3 | P0 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FPM-008 | FR-FPM-008 | Fiscal Period Management | `ApplyCloseApprovalDecision` | EP-FPM-001 | M3 | P0 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FPM-009 | FR-FPM-009 | Fiscal Period Management | `RequestReopen` | EP-FPM-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FPM-010 | FR-FPM-010 | Fiscal Period Management | `ApplyReopenApprovalDecision` | EP-FPM-001 | M3 | P0 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FPM-011 | FR-FPM-011 | Fiscal Period Management | `StartReclose` | EP-FPM-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FPM-012 | FR-FPM-012 | Fiscal Period Management | `TakeOverPeriodControl` | EP-FPM-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FPM-013 | FR-FPM-013 | Fiscal Period Management | `ExtendCloseException` | EP-FPM-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FX-001 | FR-FX-001 | Multi-Currency | `PublishRateSet` | EP-FX-001 | M8 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FX-002 | FR-FX-002 | Multi-Currency | `RunRevaluation` | EP-FX-001 | M8 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FX-003 | FR-FX-003 | Multi-Currency | `ApplyRevaluationApprovalDecision` | EP-FX-001 | M8 | P1 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-FX-004 | FR-FX-004 | Multi-Currency | `PostRevaluationRun` | EP-FX-001 | M8 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-FX-005 | FR-FX-005 | Multi-Currency | `RunTranslation` | EP-FX-001 | M8 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-001 | FR-GL-001 | General Ledger | `SubmitPostingRequest` | EP-GL-001 | M2 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-002 | FR-GL-002 | General Ledger | `ApplyJournalApprovalDecision` | EP-GL-001 | M2 | P0 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-GL-003 | FR-GL-003 | General Ledger | `ReverseJournalEntry` | EP-GL-001 | M2 | P0 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-GL-004 | FR-GL-004 | General Ledger | `EnterSoftCloseGate` | EP-GL-001 | M2 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-005 | FR-GL-005 | General Ledger | `ExitSoftCloseGate` | EP-GL-001 | M2 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-006 | FR-GL-006 | General Ledger | `AcquirePostingBarrier` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-007 | FR-GL-007 | General Ledger | `ReleasePostingBarrier` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-008 | FR-GL-008 | General Ledger | `FinalizePostingGate` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-009 | FR-GL-009 | General Ledger | `OpenScopedReopenGate` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-010 | FR-GL-010 | General Ledger | `CloseScopedReopenGate` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-011 | FR-GL-011 | General Ledger | `OpenOperationalReopenGate` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-012 | FR-GL-012 | General Ledger | `CloseOperationalReopenGate` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-013 | FR-GL-013 | General Ledger | `BeginRecloseGate` | EP-GL-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-GL-014 | FR-GL-014 | General Ledger | `GetPostingGateStatus` | EP-GL-001 | M3 | P0 | The current authoritative reference result, scope, version, status, owner, and relevant evidence references are di… |
| DLV-FR-GL-015 | FR-GL-015 | General Ledger | `Maintain ledgers` | EP-GL-001 | M1 | P0 | The accepted ledger version, ownership, currency, calendar, effective-date range, lifecycle status, approval evide… |
| DLV-FR-GL-016 | FR-GL-016 | General Ledger | `Maintain accounting books` | EP-GL-001 | M1 | P0 | The accepted accounting-book version, ledger relationship, accounting basis, policy version, effective dates, life… |
| DLV-FR-GL-017 | FR-GL-017 | General Ledger | `Maintain charts of accounts` | EP-GL-001 | M1 | P0 | The accepted chart-of-accounts version, ledger, account-code policy, effective dates, dependent-account impact, ap… |
| DLV-FR-GL-018 | FR-GL-018 | General Ledger | `Maintain accounts and reporting mappings` | EP-GL-001 | M1 | P0 | The accepted account version, chart relationship, status, restrictions, currency policy, reporting mappings, effec… |
| DLV-FR-IAM-001 | FR-IAM-001 | Identity & Access | `Manage users` | EP-IAM-001 | M1 | P0 | The user identity, UserStatus, authentication-subject reference, assigned role/access scopes, approval evidence wh… |
| DLV-FR-IAM-002 | FR-IAM-002 | Identity & Access | `Manage roles` | EP-IAM-001 | M1 | P0 | The role identity, permission grants, scopes, approval evidence where required, and any segregation conflict are v… |
| DLV-FR-IAM-003 | FR-IAM-003 | Identity & Access | `Manage access policies` | EP-IAM-001 | M1 | P0 | The access-policy version, applicable scopes and actions, effective date, approval evidence, and any validation co… |
| DLV-FR-IAM-004 | FR-IAM-004 | Identity & Access | `Manage segregation rules` | EP-IAM-001 | M1 | P0 | The segregation-rule identity, conflicting permission set, enforcement mode, approval evidence where required, and… |
| DLV-FR-IAM-005 | FR-IAM-005 | Identity & Access | `Grant emergency access` | EP-IAM-001 | M1 | P0 | The emergency-access grant reference, approved scope, reason, approver, start, expiry, and review requirement are… |
| DLV-FR-IAM-006 | FR-IAM-006 | Identity & Access | `Revoke emergency access` | EP-IAM-001 | M1 | P0 | The emergency-access revocation time, reason, actor, affected scope, and post-use review status are visible. |
| DLV-FR-IC-001 | FR-IC-001 | Multi-Entity / Intercompany | `StartSettlement` | EP-IC-001 | M8 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-IC-002 | FR-IC-002 | Multi-Entity / Intercompany | `MatchIntercompanyItems` | EP-IC-001 | M8 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-IC-003 | FR-IC-003 | Multi-Entity / Intercompany | `ApplyResidualApprovalDecision` | EP-IC-001 | M8 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-IC-004 | FR-IC-004 | Multi-Entity / Intercompany | `CreateSettlementInstructions` | EP-IC-001 | M8 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-IC-005 | FR-IC-005 | Multi-Entity / Intercompany | `CompleteSettlementRun` | EP-IC-001 | M8 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-IC-006 | FR-IC-006 | Multi-Entity / Intercompany | `ApplyIncomingSettlement` | EP-IC-001 | M8 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-IC-007 | FR-IC-007 | Multi-Entity / Intercompany | `ReverseIncomingSettlementApplication` | EP-IC-001 | M8 | P2 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-IC-008 | FR-IC-008 | Multi-Entity / Intercompany | `ApplyPaymentReturn` | EP-IC-001 | M8 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-IC-009 | FR-IC-009 | Multi-Entity / Intercompany | `RunElimination` | EP-IC-001 | M8 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-IC-010 | FR-IC-010 | Multi-Entity / Intercompany | `Maintain intercompany agreements` | EP-IC-001 | M8 | P2 | The accepted agreement version, participant scopes, settlement currency, rate policy, tolerance, effective dates,… |
| DLV-FR-IC-011 | FR-IC-011 | Multi-Entity / Intercompany | `Record intercompany transactions` | EP-IC-001 | M8 | P2 | The transaction version, participant scopes, counterparty references, amount, currency, lifecycle status, agreemen… |
| DLV-FR-INV-001 | FR-INV-001 | Invoicing | `Configure invoice templates` | EP-INV-001 | M4 | P1 | The invoice-template identity, configured line items and rules, tax and payment settings, usage applicability, and… |
| DLV-FR-INV-002 | FR-INV-002 | Invoicing | `Configure billing schedules` | EP-INV-001 | M4 | P1 | The billing-schedule identity, accounting scope, customer, source contract reference, next billing date, schedule… |
| DLV-FR-INV-003 | FR-INV-003 | Invoicing | `Generate invoices` | EP-INV-001 | M4 | P1 | The generated-invoice reference and version, calculated totals, source evidence, generation outcome, and validatio… |
| DLV-FR-INV-004 | FR-INV-004 | Invoicing | `Finalize generated invoices` | EP-INV-001 | M4 | P1 | The finalized invoice version, finalization outcome, downstream handoff status, and any blocking validation or tax… |
| DLV-FR-INV-005 | FR-INV-005 | Invoicing | `Recalculate unfinalized invoices` | EP-INV-001 | M4 | P1 | The recalculated invoice version, changed calculations, source-version differences, and resulting validation statu… |
| DLV-FR-INV-006 | FR-INV-006 | Invoicing | `Cancel unfinalized invoices` | EP-INV-001 | M4 | P1 | The cancellation result, reason, affected unfinalized version, and confirmation that no finalized downstream invoi… |
| DLV-FR-OMD-001 | FR-OMD-001 | Organization & Master Data | `Maintain legal entities` | EP-OMD-001 | M1 | P1 | The accepted legal-entity change, registrations, addresses, ownership interests, effective-date range, approval ev… |
| DLV-FR-OMD-002 | FR-OMD-002 | Organization & Master Data | `Maintain parties` | EP-OMD-001 | M1 | P1 | The accepted party identity, PartyStatus, contact and address changes, bank-detail references, cooling-off status… |
| DLV-FR-OMD-003 | FR-OMD-003 | Organization & Master Data | `Maintain customer profiles` | EP-OMD-001 | M1 | P1 | The accepted customer terms, limits, billing preferences, tax treatment, and any validation rejection are visible. |
| DLV-FR-OMD-004 | FR-OMD-004 | Organization & Master Data | `Maintain vendor profiles` | EP-OMD-001 | M1 | P1 | The accepted vendor payment terms, withholding treatment, remittance preferences, and any validation rejection are… |
| DLV-FR-OMD-005 | FR-OMD-005 | Organization & Master Data | `Maintain fiscal calendars` | EP-OMD-001 | M1 | P1 | The accepted fiscal-calendar pattern and calendar-period definitions, dependent-scope impact, and any validation r… |
| DLV-FR-OMD-006 | FR-OMD-006 | Organization & Master Data | `Publish approved master-data changes` | EP-OMD-001 | M1 | P1 | The published record identity, applicable status or effective date, approval evidence where required, dependent-ca… |
| DLV-FR-PAYR-001 | FR-PAYR-001 | Payroll | `CalculatePayrollRun` | EP-PAYR-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PAYR-002 | FR-PAYR-002 | Payroll | `ApplyPayrollRunApprovalDecision` | EP-PAYR-001 | M9 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-PAYR-003 | FR-PAYR-003 | Payroll | `PostPayrollRun` | EP-PAYR-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PAYR-004 | FR-PAYR-004 | Payroll | `CreatePayrollCorrection` | EP-PAYR-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PAYR-005 | FR-PAYR-005 | Payroll | `ApplyPaymentReturn` | EP-PAYR-001 | M9 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PAYR-006 | FR-PAYR-006 | Payroll | `Maintain employee payroll profiles` | EP-PAYR-001 | M9 | P2 | The accepted payroll-profile version, pay group, policy references, access restrictions, dependent-run impact, and… |
| DLV-FR-PAYR-007 | FR-PAYR-007 | Payroll | `Maintain payroll tax-filing records` | EP-PAYR-001 | M9 | P2 | The payroll tax-filing record, period, status, evidence references, related payroll runs, Tax Filing handoff statu… |
| DLV-FR-PCM-001 | FR-PCM-001 | Payments & Cash Management | `PreparePaymentBatch` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-002 | FR-PCM-002 | Payments & Cash Management | `ApplyPaymentBatchApprovalDecision` | EP-PCM-001 | M5 | P1 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-PCM-003 | FR-PCM-003 | Payments & Cash Management | `CancelPaymentBatch` | EP-PCM-001 | M5 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-PCM-004 | FR-PCM-004 | Payments & Cash Management | `RegisterExpectedIncomingSettlement` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-005 | FR-PCM-005 | Payments & Cash Management | `ResolveExpectedIncomingSettlementException` | EP-PCM-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-006 | FR-PCM-006 | Payments & Cash Management | `CancelExpectedIncomingSettlement` | EP-PCM-001 | M5 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-PCM-007 | FR-PCM-007 | Payments & Cash Management | `CloseExpectedIncomingSettlement` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-008 | FR-PCM-008 | Payments & Cash Management | `CreatePaymentInstructionFromObligation` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-009 | FR-PCM-009 | Payments & Cash Management | `SubmitPaymentInstruction` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-010 | FR-PCM-010 | Payments & Cash Management | `RetryPaymentInstruction` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-011 | FR-PCM-011 | Payments & Cash Management | `CancelPaymentInstruction` | EP-PCM-001 | M5 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-PCM-012 | FR-PCM-012 | Payments & Cash Management | `ApplyPaymentInstructionExceptionDecision` | EP-PCM-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-013 | FR-PCM-013 | Payments & Cash Management | `RecordPaymentReturn` | EP-PCM-001 | M5 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-014 | FR-PCM-014 | Payments & Cash Management | `CancelUnpostedPaymentReturn` | EP-PCM-001 | M5 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-PCM-015 | FR-PCM-015 | Payments & Cash Management | `AcknowledgePaymentReturn` | EP-PCM-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-016 | FR-PCM-016 | Payments & Cash Management | `ResolvePaymentReturnException` | EP-PCM-001 | M5 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-017 | FR-PCM-017 | Payments & Cash Management | `RecordUnallocatedIncomingSettlement` | EP-PCM-001 | M6 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-018 | FR-PCM-018 | Payments & Cash Management | `ResolveUnallocatedIncomingSettlement` | EP-PCM-001 | M6 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-019 | FR-PCM-019 | Payments & Cash Management | `RecordIncomingSettlement` | EP-PCM-001 | M6 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-PCM-020 | FR-PCM-020 | Payments & Cash Management | `ResolveSettlementReceiptValidationException` | EP-PCM-001 | M6 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-021 | FR-PCM-021 | Payments & Cash Management | `ResolveIncomingSettlementOwnerException` | EP-PCM-001 | M6 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-022 | FR-PCM-022 | Payments & Cash Management | `CancelUnpostedSettlementReceipt` | EP-PCM-001 | M6 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-PCM-023 | FR-PCM-023 | Payments & Cash Management | `AcknowledgeIncomingSettlement` | EP-PCM-001 | M6 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-PCM-024 | FR-PCM-024 | Payments & Cash Management | `ReverseIncomingSettlement` | EP-PCM-001 | M6 | P1 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-PCM-025 | FR-PCM-025 | Payments & Cash Management | `Maintain bank accounts` | EP-PCM-001 | M5 | P1 | The accepted bank-account identity, legal-entity ownership, masked identity, currency, status, authorization evide… |
| DLV-FR-REV-001 | FR-REV-001 | Revenue Recognition | `AssessContract` | EP-REV-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-REV-002 | FR-REV-002 | Revenue Recognition | `ApplyRevenueScheduleApprovalDecision` | EP-REV-001 | M7 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-REV-003 | FR-REV-003 | Revenue Recognition | `PublishRevenueAccountingProfile` | EP-REV-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-REV-004 | FR-REV-004 | Revenue Recognition | `ModifyContract` | EP-REV-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-REV-005 | FR-REV-005 | Revenue Recognition | `ApplyContractModificationApprovalDecision` | EP-REV-001 | M7 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-REV-006 | FR-REV-006 | Revenue Recognition | `RunRecognition` | EP-REV-001 | M7 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-RPT-001 | FR-RPT-001 | Financial Reporting | `RunConsolidation` | EP-RPT-001 | M8 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-RPT-002 | FR-RPT-002 | Financial Reporting | `ApplyTranslationResult` | EP-RPT-001 | M8 | P1 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-RPT-003 | FR-RPT-003 | Financial Reporting | `ApplyConsolidationApprovalDecision` | EP-RPT-001 | M8 | P1 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-RPT-004 | FR-RPT-004 | Financial Reporting | `PublishConsolidatedStatement` | EP-RPT-001 | M8 | P1 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-RPT-005 | FR-RPT-005 | Financial Reporting | `Maintain report definitions` | EP-RPT-001 | M8 | P1 | The report-definition version, report type, mappings, calculations, validation results, approval evidence where re… |
| DLV-FR-RPT-006 | FR-RPT-006 | Financial Reporting | `Generate and publish ledger financial statements` | EP-RPT-001 | M8 | P1 | The statement version, type, reporting scope, period, presentation currency, report-definition version, source wat… |
| DLV-FR-TAX-001 | FR-TAX-001 | Tax Filing | `DetermineTax` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-002 | FR-TAX-002 | Tax Filing | `PrepareTaxReturn` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-003 | FR-TAX-003 | Tax Filing | `ApplyTaxReturnApprovalDecision` | EP-TAX-001 | M9 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-TAX-004 | FR-TAX-004 | Tax Filing | `SubmitTaxReturn` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-005 | FR-TAX-005 | Tax Filing | `CreateTaxAmendment` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-006 | FR-TAX-006 | Tax Filing | `ApplyTaxAmendmentApprovalDecision` | EP-TAX-001 | M9 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-TAX-007 | FR-TAX-007 | Tax Filing | `SubmitTaxAmendment` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-008 | FR-TAX-008 | Tax Filing | `CreateReturnLevelTaxAdjustment` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-009 | FR-TAX-009 | Tax Filing | `ApplyReturnLevelTaxAdjustmentApprovalDecision` | EP-TAX-001 | M9 | P2 | The applied decision reference and resulting approved, rejected, unchanged, or conflict state are visible; the own… |
| DLV-FR-TAX-010 | FR-TAX-010 | Tax Filing | `PostReturnLevelTaxAdjustment` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-011 | FR-TAX-011 | Tax Filing | `RequestTaxPayment` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-012 | FR-TAX-012 | Tax Filing | `RecordTaxPaymentSettlement` | EP-TAX-001 | M9 | P2 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-TAX-013 | FR-TAX-013 | Tax Filing | `ApplyIncomingSettlement` | EP-TAX-001 | M9 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-TAX-014 | FR-TAX-014 | Tax Filing | `ReverseIncomingSettlementApplication` | EP-TAX-001 | M9 | P2 | The linked correction/cancellation outcome, preserved original fact, resulting balances or lifecycle state, and an… |
| DLV-FR-TAX-015 | FR-TAX-015 | Tax Filing | `ApplyPaymentReturn` | EP-TAX-001 | M9 | P2 | The accepted, rejected, exception, reconciled, or conflict outcome is visible with the authoritative source refere… |
| DLV-FR-TAX-016 | FR-TAX-016 | Tax Filing | `Maintain tax configurations` | EP-TAX-001 | M9 | P2 | The tax-configuration version, jurisdiction, category, effective dates, rates and rules, approval status, dependen… |
| DLV-FR-WFA-001 | FR-WFA-001 | Workflow & Approvals | `CreateApprovalRequest` | EP-WFA-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-WFA-002 | FR-WFA-002 | Workflow & Approvals | `DecideApprovalRequest` | EP-WFA-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-WFA-003 | FR-WFA-003 | Workflow & Approvals | `DelegateApproval` | EP-WFA-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-WFA-004 | FR-WFA-004 | Workflow & Approvals | `EscalateApproval` | EP-WFA-001 | M3 | P0 | The operation result, current lifecycle state, validation/approval status, responsible owner, and any exception or… |
| DLV-FR-WFA-005 | FR-WFA-005 | Workflow & Approvals | `Maintain approval policies` | EP-WFA-001 | M3 | P0 | The policy version, scope, thresholds, role requirements, effective dates, approval status, activation outcome, de… |

## 6. Workflow demonstration backlog

| Delivery ID | Workflow | Milestone | Demonstration | Direct requirements | Acceptance source |
|---|---|---|---|---|---|
| DLV-WF-6.1 | WF-6.1 — Period Close: Hard Close | M3 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-FPM-003, FR-FPM-006, FR-FPM-007, FR-FPM-008, FR-GL-006, FR-GL-007, FR-GL-008, FR-GL-014, FR-RPT-006 | DDD acceptance §§14.3 and 14.8 |
| DLV-WF-6.2 | WF-6.2 — Fiscal Period Reopen and Reclose | M3 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-FPM-006, FR-FPM-009, FR-FPM-010, FR-FPM-011, FR-GL-001, FR-GL-007, FR-GL-008, FR-GL-009, FR-GL-010, FR-GL-013, FR-GL-014 | DDD acceptance §§14.10 and 14.8 |
| DLV-WF-6.3 | WF-6.3 — Intercompany Reconciliation and Settlement | M8 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-IC-001, FR-IC-002, FR-IC-003, FR-IC-004, FR-IC-005 | DDD acceptance §14.11 |
| DLV-WF-6.4 | WF-6.4 — Fixed Asset Disposal with Gain or Loss Recognition | M7 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-FA-005, FR-FA-007, FR-FA-008 | DDD acceptance §§14.2 and 14.13.7 |
| DLV-WF-6.5 | WF-6.5 — Revenue Recognition for a SaaS Contract | M7 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-AR-001, FR-REV-001, FR-REV-002, FR-REV-003, FR-REV-004, FR-REV-005, FR-REV-006 | DDD acceptance §§14.12 and 14.13.8 |
| DLV-WF-6.6 | WF-6.6 — Journal Entry Posting and Reversal | M2 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-GL-001, FR-GL-002, FR-GL-003 | DDD acceptance §§14.1 and 14.9 |
| DLV-WF-6.7 | WF-6.7 — Customer Receipt Recording with Partial Application | M4 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-AR-002, FR-AR-003, FR-AR-004, FR-AR-005 | DDD acceptance §14.6 |
| DLV-WF-7.1 | WF-7.1 — Vendor Invoice Registration, Matching, Approval, Dispute, and Void | M5 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-AP-001, FR-AP-006, FR-AP-008, FR-AP-009, FR-AP-010 | DDD acceptance §14.13.1 |
| DLV-WF-7.2 | WF-7.2 — Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation | M5 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-PCM-003 | DDD acceptance §14.13.2 |
| DLV-WF-7.3 | WF-7.3 — Customer Credit, Refund, Overpayment, Chargeback, and Write-Off | M6 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-AR-006, FR-AR-007, FR-AR-008, FR-AR-009, FR-AR-010, FR-AR-011, FR-AR-012, FR-AR-013, FR-AR-014, FR-AR-015, FR-AR-016 | DDD acceptance §14.13.3 |
| DLV-WF-7.4 | WF-7.4 — Bank Statement Import, Matching, Unmatching, and Reconciliation | M6 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-BFR-001, FR-BFR-002, FR-BFR-003, FR-BFR-004, FR-BFR-005 | DDD acceptance §14.13.4 |
| DLV-WF-7.5 | WF-7.5 — Foreign-Currency Invoice Settlement and Realized FX | M8 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | None named | DDD acceptance §14.13.5 |
| DLV-WF-7.6 | WF-7.6 — Period-End Revaluation, Rerun, and Next-Period Reversal | M8 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-FX-002, FR-FX-003, FR-FX-004 | DDD acceptance §14.13.6 |
| DLV-WF-7.7 | WF-7.7 — Full Fixed-Asset Lifecycle and Disposal Variants | M7 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-FA-001, FR-FA-002, FR-FA-003, FR-FA-004, FR-FA-005, FR-FA-006, FR-FA-007, FR-FA-008, FR-FA-018, FR-FA-019, FR-FA-020, FR-FA-021 | DDD acceptance §14.13.7 |
| DLV-WF-7.8 | WF-7.8 — Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration | M7 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-REV-004, FR-REV-005 | DDD acceptance §14.13.8 |
| DLV-WF-7.9 | WF-7.9 — Consolidation, Ownership Changes, Translation, Eliminations, and Rerun | M8 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-RPT-001, FR-RPT-003, FR-RPT-004 | DDD acceptance §14.13.9 |
| DLV-WF-7.10 | WF-7.10 — Tax Return Submission, Rejection, Amendment, Payment, and Evidence | M9 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-TAX-010 | DDD acceptance §14.13.10 |
| DLV-WF-7.11 | WF-7.11 — Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment | M9 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-PAYR-001, FR-PAYR-002, FR-PAYR-003, FR-PAYR-004, FR-PAYR-007 | DDD acceptance §14.13.11 |
| DLV-WF-7.12 | WF-7.12 — Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen | M3 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-FPM-001, FR-FPM-012, FR-FPM-013, FR-GL-011, FR-GL-012, FR-GL-013 | DDD acceptance §14.13.12 |
| DLV-WF-7.13 | WF-7.13 — Cross-Context Event Interpretation, Ordering, and Replay | M9 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | GFR-006, GFR-007, GFR-008, GFR-009, GFR-012, GFR-013, GFR-014 | DDD acceptance §14.13.13 |
| DLV-WF-7.14 | WF-7.14 — Concurrent Aggregate and Domain-Process Modification Rules | M9 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | GFR-006, GFR-007, GFR-008, GFR-009, GFR-012, GFR-013, GFR-014 | DDD acceptance §14.13.14 |
| DLV-WF-7.15 | WF-7.15 — Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation | M9 | Run the normal, boundary, duplicate, concurrency, failure and recovery paths applicable to this workflow. | FR-AUD-001, FR-AUD-002, FR-AUD-003, FR-AUD-004, FR-AUD-005 | DDD acceptance §14.13.15 |

## 7. NFR qualification backlog

| Delivery ID | NFR | Category | Quality gate | First required milestone | Verification summary |
|---|---|---|---|---|---|
| DLV-NFR-ACC-001 | NFR-ACC-001 | Accessibility and Inclusive Use | QG-05 | M0 | Verify through automated checks and manual testing of representative pages and all 22 critical workflows before release. |
| DLV-NFR-ACC-002 | NFR-ACC-002 | Accessibility and Inclusive Use | QG-05 | M0 | Test without pointing-device input, including conflict, approval, correction, and recovery paths. |
| DLV-NFR-ACC-003 | NFR-ACC-003 | Accessibility and Inclusive Use | QG-05 | M0 | Test with at least two approved screen-reader and browser combinations. |
| DLV-NFR-ACC-004 | NFR-ACC-004 | Accessibility and Inclusive Use | QG-05 | M0 | Verify text labels, icons with accessible names, and patterns where visual distinction is required. |
| DLV-NFR-ACC-005 | NFR-ACC-005 | Accessibility and Inclusive Use | QG-05 | M0 | Test representative dense financial tables, forms, dialogs, and reports. |
| DLV-NFR-ACC-006 | NFR-ACC-006 | Accessibility and Inclusive Use | QG-05 | M0 | Test single-field, multi-line, aggregate, authorization, and conflict errors. |
| DLV-NFR-ACC-007 | NFR-ACC-007 | Accessibility and Inclusive Use | QG-05 | M0 | Test long-running progress, approval outcome, external outcome, and exception creation. |
| DLV-NFR-ACC-008 | NFR-ACC-008 | Accessibility and Inclusive Use | QG-05 | M0 | Test session, approval, authorization, reopen, and provider-window expiries. |
| DLV-NFR-ACC-009 | NFR-ACC-009 | Accessibility and Inclusive Use | QG-05 | M0 | Test operating-system reduced-motion preference. |
| DLV-NFR-ACC-010 | NFR-ACC-010 | Accessibility and Inclusive Use | QG-05 | M0 | Verify at least one human-readable export for each report and evidence class. |
| DLV-NFR-ACC-011 | NFR-ACC-011 | Accessibility and Inclusive Use | QG-05 | M0 | Verify defect classification and release evidence. |
| DLV-NFR-ACC-012 | NFR-ACC-012 | Accessibility and Inclusive Use | QG-05 | M0 | Record findings, decisions, and remediation traceability. |
| DLV-NFR-AUD-001 | NFR-AUD-001 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Reconcile authoritative action counts to audit-evidence counts daily with zero unexplained omissions. |
| DLV-NFR-AUD-002 | NFR-AUD-002 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Measure by event class, capability, and accounting scope. |
| DLV-NFR-AUD-003 | NFR-AUD-003 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Verify 100 percent schema conformance for required fields. |
| DLV-NFR-AUD-004 | NFR-AUD-004 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Verify through authorized and unauthorized modification attempts. |
| DLV-NFR-AUD-005 | NFR-AUD-005 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Execute all DDD §14.13.15 normal, boundary, concurrency, duplicate, and recovery cases. |
| DLV-NFR-AUD-006 | NFR-AUD-006 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Monitor continuously and test recovery from time-source loss. |
| DLV-NFR-AUD-007 | NFR-AUD-007 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Verify permission filtering and masked fields. |
| DLV-NFR-AUD-008 | NFR-AUD-008 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Test every WF-6.x and WF-7.x workflow. |
| DLV-NFR-AUD-009 | NFR-AUD-009 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Verify export and subsequent verification under unchanged and altered-content cases. |
| DLV-NFR-AUD-010 | NFR-AUD-010 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Reconcile privileged audit-access activity monthly. |
| DLV-NFR-AUD-011 | NFR-AUD-011 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Verify incident closure requires approved resolution and post-incident review. |
| DLV-NFR-AUD-012 | NFR-AUD-012 | Auditability, Evidence, and Nonrepudiation | QG-08 | M9 | Verify evidence remains after permitted content destruction. |
| DLV-NFR-AVL-001 | NFR-AVL-001 | Availability and Service Continuity | QG-09 | M9 | Availability is measured by successful completion of representative Class A synthetic and real-user transactions; planned maintenance… |
| DLV-NFR-AVL-002 | NFR-AVL-002 | Availability and Service Continuity | QG-09 | M9 | Measure record, worklist, search, maintenance, and exception-resolution paths by capability and aggregate. |
| DLV-NFR-AVL-003 | NFR-AVL-003 | Availability and Service Continuity | QG-09 | M9 | Measure report, consolidation, revaluation, depreciation, import, export, and verification functions separately from interactive avail… |
| DLV-NFR-AVL-004 | NFR-AVL-004 | Availability and Service Continuity | QG-09 | M9 | Verify that active business records remain usable under established configuration where policy permits. |
| DLV-NFR-AVL-005 | NFR-AVL-005 | Availability and Service Continuity | QG-09 | M9 | Audit maintenance notices and actual windows; overruns count as unavailable time. |
| DLV-NFR-AVL-006 | NFR-AVL-006 | Availability and Service Continuity | QG-09 | M9 | Test provider, authority, reporting, notification, and reference-data failures independently. |
| DLV-NFR-AVL-007 | NFR-AVL-007 | Availability and Service Continuity | QG-09 | M9 | Verify status messaging, blocked actions, owner, and recovery path for each service class. |
| DLV-NFR-AVL-008 | NFR-AVL-008 | Availability and Service Continuity | QG-09 | M9 | Monthly service reporting shall include numerator, denominator, exclusions, and top failure causes. |
| DLV-NFR-AVL-009 | NFR-AVL-009 | Availability and Service Continuity | QG-09 | M9 | The elevated window shall define staffing, monitoring, change restrictions, and recovery expectations without changing domain rules. |
| DLV-NFR-AVL-010 | NFR-AVL-010 | Availability and Service Continuity | QG-09 | M9 | Verify incident start/end, affected functions, impact, root cause category, and corrective action are retained. |
| DLV-NFR-CAP-001 | NFR-CAP-001 | Capacity and Scalability | QG-09 | M9 | Qualification shall use representative role, scope, search, approval, posting, reporting, and exception workloads rather than idle ses… |
| DLV-NFR-CAP-002 | NFR-CAP-002 | Capacity and Scalability | QG-09 | M9 | Qualification mixes journal, approval, receipt, payment, reconciliation, correction, and administrative changes. |
| DLV-NFR-CAP-003 | NFR-CAP-003 | Capacity and Scalability | QG-09 | M9 | Qualification includes permission filtering and realistic data distribution. |
| DLV-NFR-CAP-004 | NFR-CAP-004 | Capacity and Scalability | QG-09 | M9 | Daily qualification totals shall reconcile to authoritative ledger counts and values. |
| DLV-NFR-CAP-005 | NFR-CAP-005 | Capacity and Scalability | QG-09 | M9 | Qualification shall include balanced, unbalanced, mixed-currency, invalid-account, and period-gate cases. |
| DLV-NFR-CAP-006 | NFR-CAP-006 | Capacity and Scalability | QG-09 | M9 | Qualification shall include aged, partially settled, disputed, corrected, and closed items. |
| DLV-NFR-CAP-007 | NFR-CAP-007 | Capacity and Scalability | QG-09 | M9 | Qualification shall include delegation, escalation, expiry, and segregation checks. |
| DLV-NFR-CAP-008 | NFR-CAP-008 | Capacity and Scalability | QG-09 | M9 | Capacity reviews shall be performed at least quarterly and before forecast utilization exceeds 70 percent of any approved limit. |
| DLV-NFR-CAP-009 | NFR-CAP-009 | Capacity and Scalability | QG-09 | M9 | Qualification shall exceed each baseline limit by at least 25 percent and verify typed outcomes and recovery. |
| DLV-NFR-CAP-010 | NFR-CAP-010 | Capacity and Scalability | QG-09 | M9 | Verify monthly capacity reporting and alerts at 70, 80, and 90 percent of approved limits. |
| DLV-NFR-CMP-001 | NFR-CMP-001 | Compatibility and Client Quality | QG-05 | M0 | Verify compatibility for each production release. |
| DLV-NFR-CMP-002 | NFR-CMP-002 | Compatibility and Client Quality | QG-05 | M0 | Test that dense tables scroll within a labeled region while retaining row identity and headers. |
| DLV-NFR-CMP-003 | NFR-CMP-003 | Compatibility and Client Quality | QG-05 | M0 | Test that intentionally unavailable actions are identified before user input is lost. |
| DLV-NFR-CMP-004 | NFR-CMP-004 | Compatibility and Client Quality | QG-05 | M0 | Test disabled storage, interrupted network, unsupported browser, and expired session cases. |
| DLV-NFR-CMP-005 | NFR-CMP-005 | Compatibility and Client Quality | QG-05 | M0 | Test representative long identifiers, multi-currency values, and multi-page tables. |
| DLV-NFR-CMP-006 | NFR-CMP-006 | Compatibility and Client Quality | QG-05 | M0 | Compare the same record under at least three supported locales. |
| DLV-NFR-CMP-007 | NFR-CMP-007 | Compatibility and Client Quality | QG-05 | M0 | Test every Class A submit and approval action. |
| DLV-NFR-CMP-008 | NFR-CMP-008 | Compatibility and Client Quality | QG-05 | M0 | Verify notification at least 90 days before planned support removal. |
| DLV-NFR-INT-001 | NFR-INT-001 | Interoperability and External Dependency Quality | QG-04 | M2 | Verify round-trip evidence for each cross-capability and external workflow. |
| DLV-NFR-INT-002 | NFR-INT-002 | Interoperability and External Dependency Quality | QG-04 | M2 | Test current, prior-supported, unknown, and malformed versions. |
| DLV-NFR-INT-003 | NFR-INT-003 | Interoperability and External Dependency Quality | QG-04 | M2 | Execute duplicate, replay, reorder, and delayed-delivery tests across recovery. |
| DLV-NFR-INT-004 | NFR-INT-004 | Interoperability and External Dependency Quality | QG-04 | M2 | Verify exception evidence, owner, age, retry eligibility, and resolution. |
| DLV-NFR-INT-005 | NFR-INT-005 | Interoperability and External Dependency Quality | QG-04 | M2 | Test bank, payment, tax, rate, identity, and notification dependency outages. |
| DLV-NFR-INT-006 | NFR-INT-006 | Interoperability and External Dependency Quality | QG-04 | M2 | Test full acceptance, full rejection, mixed result, restart, and corrected re-import. |
| DLV-NFR-INT-007 | NFR-INT-007 | Interoperability and External Dependency Quality | QG-04 | M2 | Verify authorized and restricted export cases. |
| DLV-NFR-INT-008 | NFR-INT-008 | Interoperability and External Dependency Quality | QG-04 | M2 | Test that automated retry cannot create a second business obligation. |
| DLV-NFR-INT-009 | NFR-INT-009 | Interoperability and External Dependency Quality | QG-04 | M2 | Verify backward-compatibility and rollback evidence before release. |
| DLV-NFR-INT-010 | NFR-INT-010 | Interoperability and External Dependency Quality | QG-04 | M2 | Data freshness shall meet NFR-OBS-001. |
| DLV-NFR-LOC-001 | NFR-LOC-001 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Test at least three locales with different date, decimal, grouping, and negative-number conventions. |
| DLV-NFR-LOC-002 | NFR-LOC-002 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Test cutoff, due date, posting date, occurred time, and recorded time. |
| DLV-NFR-LOC-003 | NFR-LOC-003 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Test cross-currency invoice, settlement, revaluation, translation, and reporting workflows. |
| DLV-NFR-LOC-004 | NFR-LOC-004 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Verify historical records after policy change. |
| DLV-NFR-LOC-005 | NFR-LOC-005 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Test skipped, repeated, and offset-changing local times. |
| DLV-NFR-LOC-006 | NFR-LOC-006 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Test combining characters, non-Latin scripts, and right-to-left content values even when the interface language remains left-to-right. |
| DLV-NFR-LOC-007 | NFR-LOC-007 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Test longest supported translations and responsive layouts. |
| DLV-NFR-LOC-008 | NFR-LOC-008 | Localization, Currency, Date, and Language Quality | QG-05 | M0 | Verify no unsupported statutory interpretation is implied. |
| DLV-NFR-MNT-001 | NFR-MNT-001 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Verify 100 percent traceability completeness at release. |
| DLV-NFR-MNT-002 | NFR-MNT-002 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Verify source hashes and change-impact review. |
| DLV-NFR-MNT-003 | NFR-MNT-003 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Test rule change, rollback where permitted, and historical reproduction. |
| DLV-NFR-MNT-004 | NFR-MNT-004 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Verify compatibility tests before production release. |
| DLV-NFR-MNT-005 | NFR-MNT-005 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Exercise rollback or forward-fix procedure before high-risk releases. |
| DLV-NFR-MNT-006 | NFR-MNT-006 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Verify changes for OMD, GL, COA, tax, workflow, identity, reporting, rates, and bank connections. |
| DLV-NFR-MNT-007 | NFR-MNT-007 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Runbooks shall identify owner, prerequisites, decision points, evidence, escalation, and completion checks and be reviewed at least qu… |
| DLV-NFR-MNT-008 | NFR-MNT-008 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Verify documentation-to-product consistency by sampling before release. |
| DLV-NFR-MNT-009 | NFR-MNT-009 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Compare release candidate to the current production baseline. |
| DLV-NFR-MNT-010 | NFR-MNT-010 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Verify every high-severity exception has documented owner, impact, expiry, compensating control, and approval. |
| DLV-NFR-MNT-011 | NFR-MNT-011 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Review at least quarterly and alert 180 days before end of support. |
| DLV-NFR-MNT-012 | NFR-MNT-012 | Maintainability, Change Safety, and Operability | QG-01 | M0 | Any changed requirement or source hash shall trigger the affected-category and traceability review. |
| DLV-NFR-OBS-001 | NFR-OBS-001 | Observability, Operations, and Supportability | QG-08 | M0 | Data freshness shall be no older than 2 minutes for Class A and 5 minutes for other classes. |
| DLV-NFR-OBS-002 | NFR-OBS-002 | Observability, Operations, and Supportability | QG-08 | M0 | Verify through quarterly alert exercises and production incident records. |
| DLV-NFR-OBS-003 | NFR-OBS-003 | Observability, Operations, and Supportability | QG-08 | M0 | Verify support can diagnose without requesting secrets or unrestricted screenshots. |
| DLV-NFR-OBS-004 | NFR-OBS-004 | Observability, Operations, and Supportability | QG-08 | M0 | Verify automated sensitive-data scans run continuously and before release. |
| DLV-NFR-OBS-005 | NFR-OBS-005 | Observability, Operations, and Supportability | QG-08 | M0 | Reports shall include availability, p95/p99 latency, completion windows, data freshness, incident impact, and exclusions. |
| DLV-NFR-OBS-006 | NFR-OBS-006 | Observability, Operations, and Supportability | QG-08 | M0 | Verify alert thresholds are configurable by workflow and business calendar. |
| DLV-NFR-OBS-007 | NFR-OBS-007 | Observability, Operations, and Supportability | QG-08 | M0 | Verify no production critical alert remains unowned for more than 15 minutes. |
| DLV-NFR-OBS-008 | NFR-OBS-008 | Observability, Operations, and Supportability | QG-08 | M0 | Verify service reporting does not combine these outcomes into one generic error rate. |
| DLV-NFR-OBS-009 | NFR-OBS-009 | Observability, Operations, and Supportability | QG-08 | M0 | High-severity incidents require review within 5 business days. |
| DLV-NFR-OBS-010 | NFR-OBS-010 | Observability, Operations, and Supportability | QG-08 | M0 | Verify retention classification and legal-hold compatibility. |
| DLV-NFR-OBS-011 | NFR-OBS-011 | Observability, Operations, and Supportability | QG-08 | M0 | Verify updates occur at least every 30 minutes during a material incident. |
| DLV-NFR-OBS-012 | NFR-OBS-012 | Observability, Operations, and Supportability | QG-08 | M0 | Verify before/after values, owner, approval, effective time, and rollback path. |
| DLV-NFR-PERF-001 | NFR-PERF-001 | Performance and Responsiveness | QG-09 | M9 | Measure from action submission to presentation of validated, rejected, conflict, or dependency-unavailable outcome over rolling 30-day… |
| DLV-NFR-PERF-002 | NFR-PERF-002 | Performance and Responsiveness | QG-09 | M9 | Measure from accepted submission to authoritative result reference and lifecycle state becoming visible to the initiating user. |
| DLV-NFR-PERF-003 | NFR-PERF-003 | Performance and Responsiveness | QG-09 | M9 | Measure separately from external-provider completion time; no external dependency time may be reported as local processing time. |
| DLV-NFR-PERF-004 | NFR-PERF-004 | Performance and Responsiveness | QG-09 | M9 | Test records with up to 10,000 lineage items, using bounded first-page retrieval and visible continuation. |
| DLV-NFR-PERF-005 | NFR-PERF-005 | Performance and Responsiveness | QG-09 | M9 | Measure with representative filters for scope, state, owner, date, amount, currency, exception, and approval status. |
| DLV-NFR-PERF-006 | NFR-PERF-006 | Performance and Responsiveness | QG-09 | M9 | Qualification shall include exact identifier, source reference, party, amount/date range, and authorized full-text searches. |
| DLV-NFR-PERF-007 | NFR-PERF-007 | Performance and Responsiveness | QG-09 | M9 | Measure with at least 100,000 open approval tasks and all applicable segregation checks enabled. |
| DLV-NFR-PERF-008 | NFR-PERF-008 | Performance and Responsiveness | QG-09 | M9 | Verify that progress includes stage, completed/total counts where meaningful, last update, owner, blockers, and cancellation eligibili… |
| DLV-NFR-PERF-009 | NFR-PERF-009 | Performance and Responsiveness | QG-09 | M9 | Qualification uses the baseline reporting profile and records the definition version and source watermarks. |
| DLV-NFR-PERF-010 | NFR-PERF-010 | Performance and Responsiveness | QG-09 | M9 | Measure end-to-end processing time by stage and separately report blocked business time. |
| DLV-NFR-PERF-011 | NFR-PERF-011 | Performance and Responsiveness | QG-09 | M9 | Verify validation exceptions are itemized and successful instructions are not delayed by unrelated invalid instructions unless policy… |
| DLV-NFR-PERF-012 | NFR-PERF-012 | Performance and Responsiveness | QG-09 | M9 | Measure rejected-line reporting, duplicate detection, balance checks, and progress visibility under the same target. |
| DLV-NFR-PERF-013 | NFR-PERF-013 | Performance and Responsiveness | QG-09 | M9 | Measure from authoritative outcome time to notification availability; external delivery-channel delay is reported separately. |
| DLV-NFR-PERF-014 | NFR-PERF-014 | Performance and Responsiveness | QG-09 | M9 | Measure from authoritative publication to first authorized dependent view; exceptions beyond p99 shall be visible and alertable. |
| DLV-NFR-PRV-001 | NFR-PRV-001 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Review data inventory before release and after material field additions. |
| DLV-NFR-PRV-002 | NFR-PRV-002 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Verify that identifiers or masked references replace full sensitive values wherever the business purpose permits. |
| DLV-NFR-PRV-003 | NFR-PRV-003 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Test role combinations and indirect disclosure through filters, counts, comparisons, and errors. |
| DLV-NFR-PRV-004 | NFR-PRV-004 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Verify retention decisions against declared policy and source relationships. |
| DLV-NFR-PRV-005 | NFR-PRV-005 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Verify hold propagation to source records, evidence, exports, backups subject to policy, and scheduled destruction. |
| DLV-NFR-PRV-006 | NFR-PRV-006 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Verify that denied deletion cites the lawful retention basis and that permitted corrections use linked lineage. |
| DLV-NFR-PRV-007 | NFR-PRV-007 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Qualification shall verify storage, processing, export, support, and recovery locations against approved policy. |
| DLV-NFR-PRV-008 | NFR-PRV-008 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Audit samples quarterly and after incident data capture. |
| DLV-NFR-PRV-009 | NFR-PRV-009 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Automated tests shall scan notification content and templates before release. |
| DLV-NFR-PRV-010 | NFR-PRV-010 | Privacy, Retention, and Legal Hold | QG-06 | M1 | Verify counts, scope, approval, hold checks, completion time, and exception reporting. |
| DLV-NFR-REC-001 | NFR-REC-001 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Fault-injection and disaster-recovery tests shall confirm every acknowledged outcome is present exactly once after recovery. |
| DLV-NFR-REC-002 | NFR-REC-002 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Verify autosave or equivalent user-visible recovery without establishing unauthorized business effects. |
| DLV-NFR-REC-003 | NFR-REC-003 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Measure time to verified business operation, reconciliation, and user access, not only technical process start. |
| DLV-NFR-REC-004 | NFR-REC-004 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Test each service class independently and report unmet dependencies. |
| DLV-NFR-REC-005 | NFR-REC-005 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Execute interruption at each documented state boundary and compare authoritative outcomes. |
| DLV-NFR-REC-006 | NFR-REC-006 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Tests shall include financial reconciliation, audit proof, access controls, legal holds, and source/dependent consistency. |
| DLV-NFR-REC-007 | NFR-REC-007 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Verify restored records against pre-failure control totals and hashes. |
| DLV-NFR-REC-008 | NFR-REC-008 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Measure backlog age, catch-up rate, duplicate prevention, and exception volume. |
| DLV-NFR-REC-009 | NFR-REC-009 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Test payment, tax, bank, filing, and provider-return uncertainty. |
| DLV-NFR-REC-010 | NFR-REC-010 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Verify no direct destructive edit bypasses domain correction semantics. |
| DLV-NFR-REC-011 | NFR-REC-011 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Verify recovery locations and restored policy enforcement. |
| DLV-NFR-REC-012 | NFR-REC-012 | Resilience, Backup, and Disaster Recovery | QG-09 | M9 | Verify sign-off by business, operations, and security owners for Class A recovery. |
| DLV-NFR-REL-001 | NFR-REL-001 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify by fault injection immediately before, during, and after acknowledgement. |
| DLV-NFR-REL-002 | NFR-REL-002 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Execute duplicate tests at least 100 times per critical command and across recovery boundaries. |
| DLV-NFR-REL-003 | NFR-REL-003 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify conflict evidence includes established identity, fingerprint/result reference, and required new-action path. |
| DLV-NFR-REL-004 | NFR-REL-004 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Run race tests for every multi-aggregate and ownership-transfer rule defined by DDD §9.1. |
| DLV-NFR-REL-005 | NFR-REL-005 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Reconcile journal headers, lines, gate evidence, and source references continuously and during qualification. |
| DLV-NFR-REL-006 | NFR-REL-006 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Test minor-unit, high-precision rate, rounding boundary, negative correction, and aggregate-total cases. |
| DLV-NFR-REL-007 | NFR-REL-007 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Execute invariant checks during qualification, reconciliation, and controlled production monitoring. |
| DLV-NFR-REL-008 | NFR-REL-008 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify bidirectional navigation and totals before and after correction. |
| DLV-NFR-REL-009 | NFR-REL-009 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Reconcile published outcome counts and values by source identity and owner. |
| DLV-NFR-REL-010 | NFR-REL-010 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify payment, disposal, close, receipt, consolidation, and filing workflows with mixed outcomes. |
| DLV-NFR-REL-011 | NFR-REL-011 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Compare authoritative and dependent records for 100 percent of sampled cross-capability outcomes. |
| DLV-NFR-REL-012 | NFR-REL-012 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify that regeneration from unchanged inputs reproduces the same authoritative values. |
| DLV-NFR-REL-013 | NFR-REL-013 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Test time-zone boundaries, daylight-saving transitions, period cutoff, and legal-date rules. |
| DLV-NFR-REL-014 | NFR-REL-014 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify before/after reporting and correction behavior after configuration changes. |
| DLV-NFR-REL-015 | NFR-REL-015 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Qualification shall prove no silent record omission or duplication. |
| DLV-NFR-REL-016 | NFR-REL-016 | Reliability, Data Integrity, and Consistency | QG-07 | M2 | Verify exception includes scope, record identities, amount/value difference, first occurrence, and resolution status. |
| DLV-NFR-SEC-001 | NFR-SEC-001 | Security, Identity, and Access Control | QG-06 | M1 | Verify identity source, authentication assurance, and emergency-account inventory quarterly. |
| DLV-NFR-SEC-002 | NFR-SEC-002 | Security, Identity, and Access Control | QG-06 | M1 | Verify at authentication and step-up points through positive and negative tests. |
| DLV-NFR-SEC-003 | NFR-SEC-003 | Security, Identity, and Access Control | QG-06 | M1 | High-risk actions include payment release, bank-detail change, manual journal approval above threshold, period reopen, emergency acces… |
| DLV-NFR-SEC-004 | NFR-SEC-004 | Security, Identity, and Access Control | QG-06 | M1 | Test every role against allowed and denied legal entity, segment, account, transaction, amount, currency, period, sensitivity, and act… |
| DLV-NFR-SEC-005 | NFR-SEC-005 | Security, Identity, and Access Control | QG-06 | M1 | Test all minimum segregation rules from DDD §8.2 and configured extensions. |
| DLV-NFR-SEC-006 | NFR-SEC-006 | Security, Identity, and Access Control | QG-06 | M1 | Measure from authoritative access-policy change to denied access in every affected capability. |
| DLV-NFR-SEC-007 | NFR-SEC-007 | Security, Identity, and Access Control | QG-06 | M1 | Verify unsaved work handling does not establish unauthorized actions after expiry. |
| DLV-NFR-SEC-008 | NFR-SEC-008 | Security, Identity, and Access Control | QG-06 | M1 | Verify no sensitive action proceeds when the required assurance cannot be established. |
| DLV-NFR-SEC-009 | NFR-SEC-009 | Security, Identity, and Access Control | QG-06 | M1 | Verify coverage for business records, evidence, exports, backups, and administrative channels. |
| DLV-NFR-SEC-010 | NFR-SEC-010 | Security, Identity, and Access Control | QG-06 | M1 | Run automated secret scanning and manual negative tests before every release. |
| DLV-NFR-SEC-011 | NFR-SEC-011 | Security, Identity, and Access Control | QG-06 | M1 | Test payroll, tax, bank, personal, and security-sensitive fields in views, comparisons, exports, notifications, and errors. |
| DLV-NFR-SEC-012 | NFR-SEC-012 | Security, Identity, and Access Control | QG-06 | M1 | Verify actor, approver where required, reason, scope, before/after fingerprints, and effective interval. |
| DLV-NFR-SEC-013 | NFR-SEC-013 | Security, Identity, and Access Control | QG-06 | M1 | Verify automatic expiry and complete action review. |
| DLV-NFR-SEC-014 | NFR-SEC-014 | Security, Identity, and Access Control | QG-06 | M1 | Verify environment data inventories and sampling at least quarterly. |
| DLV-NFR-SEC-015 | NFR-SEC-015 | Security, Identity, and Access Control | QG-06 | M1 | Measure from validated finding time; exceptions require accountable owner, expiry, compensating controls, and approval. |
| DLV-NFR-SEC-016 | NFR-SEC-016 | Security, Identity, and Access Control | QG-06 | M1 | Verify that no unresolved critical finding remains at release. |
| DLV-NFR-SEC-017 | NFR-SEC-017 | Security, Identity, and Access Control | QG-06 | M1 | Verify exports cannot infer or include unauthorized values. |
| DLV-NFR-SEC-018 | NFR-SEC-018 | Security, Identity, and Access Control | QG-06 | M1 | Verify event-to-alert latency and incident ownership through quarterly exercises. |
| DLV-NFR-TST-001 | NFR-TST-001 | Verification, Testing, and Release Quality | QG-10 | M9 | Coverage shall preserve normal, boundary, concurrency, duplicate, and failure/recovery meanings. |
| DLV-NFR-TST-002 | NFR-TST-002 | Verification, Testing, and Release Quality | QG-10 | M9 | Critical workflow criteria shall be exercised end to end. |
| DLV-NFR-TST-003 | NFR-TST-003 | Verification, Testing, and Release Quality | QG-10 | M9 | Coverage completeness shall be 100 percent before release. |
| DLV-NFR-TST-004 | NFR-TST-004 | Verification, Testing, and Release Quality | QG-10 | M9 | Verify zero unexplained financial differences and zero duplicate effects. |
| DLV-NFR-TST-005 | NFR-TST-005 | Verification, Testing, and Release Quality | QG-10 | M9 | Results shall include p50, p95, p99, throughput, saturation, error classes, and recovery after overload. |
| DLV-NFR-TST-006 | NFR-TST-006 | Verification, Testing, and Release Quality | QG-10 | M9 | Verify that no critical security finding remains unresolved. |
| DLV-NFR-TST-007 | NFR-TST-007 | Verification, Testing, and Release Quality | QG-10 | M9 | Verify release severity is assigned according to NFR-ACC-011. |
| DLV-NFR-TST-008 | NFR-TST-008 | Verification, Testing, and Release Quality | QG-10 | M9 | Required exercises shall meet NFR-REC objectives. |
| DLV-NFR-TST-009 | NFR-TST-009 | Verification, Testing, and Release Quality | QG-10 | M9 | Verify missing mandatory evidence blocks release. |
| DLV-NFR-TST-010 | NFR-TST-010 | Verification, Testing, and Release Quality | QG-10 | M9 | Verify any critical failure triggers rollback or forward-correction under NFR-MNT-005. |

## 8. Definition of Ready

A delivery item is ready only when:

- source `FR-*`, `GFR-*`, `NFR-*`, workflow and acceptance IDs are known;
- owning module and authoritative record are identified;
- UX screen/state and user-visible errors are defined;
- API, persistence, event and authorization contracts exist or are included in the item;
- dependencies are complete or scheduled earlier in the same milestone;
- normal, correction, duplicate, concurrency and failure tests are identified where applicable; and
- the item is small enough to demonstrate within one or a short chain of reviewable changes.

## 9. Definition of Done

A delivery item is done only when:

- required code, migrations and generated contracts are complete;
- domain, repository, API, frontend and end-to-end tests pass as applicable;
- authorization, sensitive-data and audit behavior are tested;
- idempotency, concurrency, correction and recovery paths pass where applicable;
- accessibility and localization checks pass for user-facing behavior;
- logs, metrics and traces identify the business outcome without exposing secrets;
- requirement, technical-specification and test traceability are updated;
- the clean-environment demo passes; and
- no critical or high unresolved defect remains.
## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `79cfdab38cefcfa01c383324541241fd9791711020566c42a7371a9cfe63e4e1` |
| Review status | Passed |
| Reuse rule | Re-run structural checks when this body hash and all source hashes remain unchanged. Run targeted semantic review for localized backlog, estimate, dependency, milestone, gate, risk or cost changes. Run the full suite for requirement, workflow, acceptance, architecture, technical-specification or source-hash changes. |

### Checks recorded

- All 22 GFRs have delivery items.
- All 193 FRs have stable vertical delivery items.
- All 22 workflows have demonstration items.
- All 174 NFRs have qualification items.
- Platform, readiness and done criteria are defined.
