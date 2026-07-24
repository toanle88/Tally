# Finance Platform Functional Product Requirements Document

| Document-control field | Value |
|---|---|
| Version | 1.5 |
| Baseline date | 2026-07-24 |
| Status | Consistency-verified functional baseline |
| Source domain baseline | Finance Domain Model & Use Cases — DDD Baseline v3.1 |
| Source checkpoint | DDD-3.1-2026-07-24 |
| Source verified-content SHA-256 | a9d437d23656c36d340afb3a5a31c93a23e574f53db186483a9edfdf32d3e652 |
| Document owner | Finance Product Management |
| Companion documents | Functional Requirements Catalog v1.5; Functional Traceability and Acceptance v1.5 |
| Intended audience | Product, Finance SMEs, UX, QA, Architecture, Engineering, Security, and Operations |

> **Purpose:** Define what the finance product must enable users and connected business actors to accomplish. This document translates the approved DDD baseline into product behavior, workflows, controls, user-visible outcomes, and functional acceptance expectations.

> **Boundary:** This PRD specifies functional behavior only. It does not prescribe APIs, persistence, middleware, deployment, infrastructure, observability implementation, or solution architecture.

## 1. Executive Summary
The product shall provide a controlled finance platform spanning general ledger, subledgers, payments, period control, asset accounting, revenue recognition, currency, consolidation, tax, approvals, access control, bank reconciliation, and audit integrity. It shall preserve accounting ownership, immutable correction lineage, scoped authorization, and explainable recovery across all supported workflows.

## 2. Product Objectives
- Provide a coherent functional experience across the finance lifecycle while preserving bounded-context ownership.
- Enable finance users and approvers to complete normal work, resolve exceptions, and prove the resulting accounting outcome.
- Prevent duplicate, unauthorized, out-of-period, unbalanced, or ownership-conflicting financial effects.
- Make workflow state, blocked actions, approvals, reconciliation status, and correction lineage visible to authorized users.
- Support multi-entity, multi-book, multi-currency, effective-dated, and audit-sensitive finance operations.

## 3. Non-Goals
- Technical architecture, service decomposition, API/event schemas, database design, middleware, and deployment topology.
- UI visual design, field-level wireframes, and design-system rules; those belong in the UX specification.
- Capacity targets, SLOs, observability implementation, migration execution, runbooks, release planning, and sprint scope.
- Capabilities excluded by DDD §12 unless separately approved.

## 4. Users and Business Actors
| Capability | Primary users | Functional responsibility |
|---|---|---|
| Organization & Master Data | Master Data Steward; Finance Administrator | Maintain legal entities, registrations, ownership interests, parties, customer/vendor profiles, and fiscal calendars, preserving effective dates where the domain defines them. |
| General Ledger (GL) | Accountant; GL Manager; Controller; authorized subledger actor | Prepare, approve, post, reverse, and review journals and authoritative posting-gate outcomes. |
| Accounts Payable (AP) | AP Specialist; AP Manager; Invoice Approver | Register, match, approve, dispute, void, pay, and correct vendor liabilities. |
| Accounts Receivable (AR) | Billing Specialist; Cash Applications Specialist; Collections Specialist; AR Manager | Issue receivables, record and apply receipts, manage credits/refunds, and resolve customer-balance exceptions. |
| Payroll | Payroll Specialist; Payroll Manager; Payroll Approver | Calculate, approve, post, correct, and reconcile payroll and payroll-related obligations. |
| Invoicing | Billing Operations Specialist; Billing Manager | Configure billing, calculate charges, generate invoices, and finalize billing handoff. |
| Payments & Cash Management | Treasury Specialist; Payment Operator; Cash Manager; Payment Approver | Approve and execute outgoing payments, record incoming settlements and returns, and reconcile cash outcomes. |
| Financial Reporting | Financial Reporting Accountant; Consolidation Manager; Controller | Run consolidation, apply translation results, and publish controlled financial statements. |
| Multi-Entity / Intercompany | Intercompany Accountant; Counterparty Accountant; Residual Approver | Match, approve, settle, and eliminate intercompany activity. |
| Revenue Recognition | Revenue Accountant; Revenue Manager; Contract Approver | Assess contracts, approve schedules, publish accounting profiles, modify contracts, and run recognition. |
| Fixed Assets | Fixed Asset Accountant; Asset Manager; Disposal Approver | Capitalize, depreciate, impair, transfer, dispose, and reconcile asset settlement obligations. |
| Multi-Currency | Treasury Accountant; FX Administrator; Consolidation Accountant | Publish rate sets, run revaluation and translation, and retain conversion evidence. |
| Fiscal Period Management | Finance Manager; Controller; Close Coordinator | Control soft close, hard close, reopen, reclose, exceptions, and period-control recovery. |
| COA Segment Accounting | Chart-of-Accounts Administrator; Finance Approver | Maintain segment definitions and values, validate combinations, and govern effective-dated changes. |
| Bank Feeds & Reconciliation | Bank Reconciliation Specialist; Cash Accountant | Import statements, propose/confirm/reverse matches, and complete reconciliation. |
| Tax Filing | Tax Specialist; Tax Manager; Tax Approver | Determine tax, prepare/submit returns and amendments, post return-level adjustments, and settle tax obligations. |
| Workflow & Approvals | Approver; Delegator; Approval Administrator | Create, decide, delegate, and escalate approval requests while enforcing policy and segregation of duties. |
| Identity & Access | Security Administrator; Access Approver; Auditor | Administer users, roles, access policies, segregation rules, and controlled emergency access. |
| Audit Integrity | Internal Auditor; Compliance Officer; Integrity Administrator | Append audit evidence, create seals, verify proof, rotate verification credentials, and manage integrity incidents. |

External or connected actors include procurement snapshot providers, payment/bank providers, tax authorities, rate providers, and automated schedulers. They supply evidence or trigger work but do not displace the authoritative ownership defined by the DDD baseline.

## 5. Functional Scope and Capability Map
| Capability | Classification | Functional responsibility | Authoritative business records |
|---|---|---|---|
| Organization & Master Data | Supporting | Legal entities, parties, fiscal calendars, registrations, organization hierarchy | LegalEntity, Party, CustomerProfile, VendorProfile, FiscalCalendar |
| General Ledger (GL) | Core | Ledger and accounting-book configuration, chart of accounts, journal validation, posting, and authoritative posting admission | Ledger, AccountingBook, ChartOfAccounts, Account, JournalEntry, PeriodPostingGate |
| Accounts Payable (AP) | Core | Vendor invoices, matching against procurement snapshots, liabilities, payment requests | VendorInvoice, PaymentRequest |
| Accounts Receivable (AR) | Core | Customer invoices, receivable open items, receipts, credit notes, refunds, and collections balances | CustomerInvoice, ReceivableOpenItem, CustomerReceipt, CreditNote, CustomerRefundRequest |
| Payroll | Supporting | Payroll calculation, liabilities, deductions, payroll posting requests | PayrollRun, EmployeePayrollProfile, PayrollTaxFiling |
| Invoicing | Core | Billing schedules, usage and charge calculation, invoice generation | InvoiceTemplate, BillingSchedule, GeneratedInvoice |
| Payments & Cash Management | Supporting | Payment batches, bank accounts, outgoing payment execution and returns, expected non-customer incoming settlements, observed settlement receipts, unallocated incoming cash, cash posting, and reconciliation status | BankAccount, PaymentBatch, PaymentInstruction, PaymentReturn, ExpectedIncomingSettlement, SettlementReceipt, UnallocatedIncomingSettlement |
| Financial Reporting | Core | Financial statements, consolidation, report definitions and mappings | ReportDefinition, ConsolidationRun, FinancialStatement |
| Multi-Entity / Intercompany | Core | Intercompany agreements, reconciliation, netting, settlement and elimination instructions | IntercompanyAgreement, IntercompanyTransaction, SettlementRun, EliminationRun |
| Revenue Recognition | Core | ASC 606 or IFRS 15 assessment, allocation, contract balances and recognition schedules | RevenueContract, RevenueSchedule, ContractModification |
| Fixed Assets | Core | Asset acquisition, capitalization, depreciation, impairment, transfer and disposal | FixedAsset, DepreciationRun, ImpairmentAssessment, AssetDisposal |
| Multi-Currency | Core | FX-rate publication, translation, revaluation and realized or unrealized gain/loss calculation | CurrencyRateSet, RevaluationRun, TranslationRun |
| Fiscal Period Management | Core | Fiscal-period state authority, close orchestration, and scoped or operational reopen controls | FiscalPeriod, SoftCloseRun, CloseRun, ReopenRequest |
| COA Segment Accounting | Supporting | Segment definitions, combinations, assignments and effective-dated controls | SegmentDefinition, SegmentCombination, SegmentChangeRequest |
| Bank Feeds & Reconciliation | Supporting | Provider connections, statement ingestion, transaction matching and reconciliation | BankFeedConnection, BankStatement, ReconciliationSession |
| Tax Filing | Supporting | Tax determination inputs, returns, submissions, amendments, return-level adjustments, tax-payment obligations, and filing status | TaxConfiguration, TaxReturn, FilingSubmission, TaxAmendment, ReturnLevelTaxAdjustment, TaxPaymentObligation |
| Workflow & Approvals | Generic | Configurable approval policies, steps, decisions, delegation and escalation | ApprovalPolicy, ApprovalRequest, Delegation |
| Identity & Access | Generic | Users, roles, permissions, access scopes and segregation-of-duties controls | User, Role, AccessPolicy, SegregationRule |
| Audit Integrity | Supporting | Append-only audit evidence, integrity sealing, proof generation, verification, and incident lineage | AuditChain |

## 6. Product-Wide Functional Principles
- **GFR-001:** The product shall use the DDD v3.1 ubiquitous language and bounded-context ownership as the authoritative functional vocabulary.
- **GFR-002:** Every user action shall be evaluated against the access dimensions applicable to that action, including legal entity, business unit or segment, account or account class, transaction type, amount threshold, currency, fiscal period, data sensitivity, and requested action.
- **GFR-003:** The product shall prevent prohibited segregation-of-duties combinations and shall expose the reason for a denied action.
- **GFR-004:** Approval-bearing actions shall route through the applicable approval policy and revalidate current business state when the decision is applied.
- **GFR-005:** Posted, accepted, or otherwise established financial facts shall not be edited destructively; correction shall use reversal, adjustment, amendment, return, unapplication, replacement, or compensation.
- **GFR-006:** Repeated submission of the same business identity and fingerprint shall return the established result without repeating the business effect.
- **GFR-007:** Reuse of the same business identity with changed functional content shall be rejected as an idempotency conflict.
- **GFR-008:** Concurrent changes shall be checked against expected versions and shall never silently overwrite an established business outcome.
- **GFR-009:** Each user-visible workflow shall expose current state, allowed actions, blocked actions, blocking reason, responsible owner, and correction or recovery path.
- **GFR-010:** Financial amounts shall display transaction, functional, and presentation currency where applicable, including the rate-set and conversion evidence used.
- **GFR-011:** Every accounting effect shall have one owning producer, and the product shall prevent duplicate accounting ownership across capabilities.
- **GFR-012:** Cross-context workflows shall expose intermediate, exception, reconciliation, and terminal states rather than presenting later outcomes as immediate success.
- **GFR-013:** The product shall preserve immutable lineage among source facts, approvals, postings, reversals, returns, amendments, replacements, and compensations.
- **GFR-014:** All material actions and decisions shall be auditable with actor, time, scope, source, action, authorization, correlation, and the applicable before/after or event fingerprints.
- **GFR-015:** Sensitive payroll, tax, bank, and personal information shall be shown only to authorized users and shall be minimized in shared business evidence and views.
- **GFR-016:** Search, worklists, and reports shall support the filters applicable to their records, including accounting scope, state, owner, date, amount, currency, exception type, and approval status where present.
- **GFR-017:** User interfaces shall distinguish dependency unavailability from domain rejection and shall show the next permitted resolution action.
- **GFR-018:** Business records subject to legal hold shall remain available and immutable until the hold is formally released, even when the ordinary retention period would otherwise permit destruction.
- **GFR-019:** Effective-dated configurations shall preserve the rule and version used by historical transactions.
- **GFR-020:** The product shall provide exportable, access-controlled evidence for approvals, postings, reconciliations, close/reopen activity, and audit-integrity verification.
- **GFR-021:** PRD-defined functional actions shall not be represented as DDD commands or domain events unless the DDD baseline is explicitly changed.
- **GFR-022:** Every functional workflow shall trace to exact requirement IDs for named DDD operations and for explicit PRD functional actions that implement stated DDD behavior. Supporting capability families identify cross-cutting dependencies and shall not substitute for missing direct functional coverage.

## 7. End-to-End Functional Workflows
### WF-6.1 — Period Close: Hard Close
- **DDD source:** §6.1
- **Primary actor:** Finance Manager
- **Supporting actors:** Workflow & Approvals, close approvers, accountants, automated close scheduler
- **Owning capability:** Fiscal Period Management
- **Primary named operations:** StartHardClose
- **Functional objective:** Finance Manager initiates hard close.
- **Expected functional behavior:** Fiscal Period Management creates one candidate CloseRun in non-owning Initiating state while the period remains SoftClosed; the existing soft-close process remains the sole gate owner. The close orchestrator issues AcquirePostingBarrier(softCloseRunId, softCloseControlEpoch, closeRunId) with the accounting scope, fiscal period, expected gate version, policy version, both process identifiers, and command fingerprint. As one domain consistency outcome, GL verifies the active soft-close owner and epoch, freezes and stores that epoch's admission summary, records the run and epoch as the prior process, clears ActiveSoftCloseRunId and ActiveSoftCloseControlEpoch, sets ActiveCloseRunId, initializes a zero admission summary for the close run, changes the gate to CloseOnly, increments postingGateVersion, records the barrier ledger position, and returns PostingBarrierAcquired with the frozen soft-close summary. There is no gate version at which both processes own admission.
- **Direct requirement IDs:** FR-FPM-003, FR-FPM-006, FR-FPM-007, FR-FPM-008, FR-GL-006, FR-GL-007, FR-GL-008, FR-GL-014, FR-RPT-006
- **Supporting requirement families:** FR-FPM-*, FR-GL-*, FR-WFA-*, FR-FX-*, FR-FA-*, FR-REV-*, FR-IC-*, FR-RPT-*, FR-AUD-*

### WF-6.2 — Fiscal Period Reopen and Reclose
- **DDD source:** §6.2
- **Primary actor:** Controller
- **Supporting actors:** Independent approvers, affected subledger owners, close domain process, General Ledger (GL), Audit Integrity, Financial Reporting
- **Owning capability:** Fiscal Period Management owns ReopenRequest, the period transitions, and reclose orchestration. GL owns the posting gate and every journal-entry admission decision. Workflow owns approval decisions.
- **Primary named operations:** RequestReopen, ApplyReopenApprovalDecision, OpenScopedReopenGate, SubmitPostingRequest, CloseScopedReopenGate, StartReclose, and BeginRecloseGate
- **Functional objective:** Fiscal Period Reopen and Reclose
- **Expected functional behavior:** The Controller submits RequestReopen with accounting scope, reason, affected accounts or transaction classes, proposed corrections, impact analysis, authorization expiry, and expected period and gate versions. Fiscal Period Management creates ReopenRequest in PendingApproval, records the immutable request fingerprint, and emits ReopenRequested through its domain events. Workflow evaluates segregation of duties and records an immutable approval decision. Fiscal Period Management consumes it through ApplyReopenApprovalDecision, revalidates the request version and policy applicability, and moves the request to Approved or Rejected without changing the period or GL gate.
- **Direct requirement IDs:** FR-FPM-006, FR-FPM-009, FR-FPM-010, FR-FPM-011, FR-GL-001, FR-GL-007, FR-GL-008, FR-GL-009, FR-GL-010, FR-GL-013, FR-GL-014
- **Supporting requirement families:** FR-FPM-*, FR-GL-*, FR-WFA-*, FR-RPT-*, FR-AUD-*

### WF-6.3 — Intercompany Reconciliation and Settlement
- **DDD source:** §6.3
- **Primary actor:** Intercompany Accountant
- **Supporting actors:** Counterparty accountants, residual approver, Workflow & Approvals, Multi-Currency, Payments & Cash Management, GL, Financial Reporting
- **Owning capability:** Multi-Entity / Intercompany owns agreements, reciprocal matching, netting, residual treatment, and settlement-run state. Payments owns bank execution. Financial Reporting owns consolidation elimination records.
- **Primary named operations:** StartSettlement, MatchIntercompanyItems, ApplyResidualApprovalDecision, CreateSettlementInstructions, and CompleteSettlementRun
- **Functional objective:** Intercompany Reconciliation and Settlement
- **Expected functional behavior:** The accountant starts a settlement run with participant scopes, agreement version, cutoff, rate policy, and expected open-item versions. Intercompany snapshots and reserves eligible items, rejecting items already reserved, settled, disputed, or changed after the cutoff. Multi-Entity / Intercompany matches reciprocal documents using agreement identifiers, source references, dates, currencies, amounts, and tolerance rules. One-sided or ambiguous items move to ExceptionsPending and remain open.
- **Direct requirement IDs:** FR-IC-001, FR-IC-002, FR-IC-003, FR-IC-004, FR-IC-005
- **Supporting requirement families:** FR-IC-*, FR-PCM-*, FR-FX-*, FR-WFA-*, FR-GL-*, FR-RPT-*

### WF-6.4 — Fixed Asset Disposal with Gain or Loss Recognition
- **DDD source:** §6.4
- **Primary actor:** Fixed Asset Accountant
- **Supporting actors:** Workflow & Approvals, General Ledger (GL), Accounts Payable (AP), Payments & Cash Management, and the disposal-recovery domain service
- **Owning capability:** Fixed Assets
- **Primary named operations:** DisposeAsset
- **Functional objective:** Accountant submits an approved disposal intent for sale, scrap, or partial disposal.
- **Expected functional behavior:** Fixed Assets retrieves cost, accumulated depreciation, impairment, components, carrying amount, prior disposal history, and the current asset version. Fixed Assets loads the approved disposal date, gross proceeds, disposal costs, disposed quantity or component, and reason from the immutable approval subject. Any material change invalidates the approval and returns the request to the proposal-and-approval flow in Section 7.7. Fixed Assets calculates depreciation through the disposal date if required.
- **Direct requirement IDs:** FR-FA-005, FR-FA-007, FR-FA-008
- **Supporting requirement families:** FR-FA-*, FR-WFA-*, FR-GL-*, FR-AP-*, FR-PCM-*

### WF-6.5 — Revenue Recognition for a SaaS Contract
- **DDD source:** §6.5
- **Primary actor:** Revenue Accountant
- **Supporting actors:** Contract approver, Workflow & Approvals, Invoicing, AR, GL, Multi-Currency, Financial Reporting
- **Owning capability:** Revenue Recognition owns accounting assessment, performance obligations, contract balances, schedule versions, and published accounting profiles. Invoicing owns commercial invoice generation. AR owns invoices, receivables, credits, refunds, and billing postings.
- **Primary named operations:** AssessContract, ApplyRevenueScheduleApprovalDecision, PublishRevenueAccountingProfile, RunRecognition, ModifyContract, and ApplyContractModificationApprovalDecision
- **Functional objective:** Revenue Recognition for a SaaS Contract
- **Expected functional behavior:** The accountant submits AssessContract with the contract source version and expected RevenueContract version. Revenue Recognition validates enforceability, collectibility, combination rules, promised goods and services, and contract term. Revenue Recognition identifies performance obligations and assesses distinctness. Non-distinct setup or implementation activity is combined with the related service obligation.
- **Direct requirement IDs:** FR-AR-001, FR-REV-001, FR-REV-002, FR-REV-003, FR-REV-004, FR-REV-005, FR-REV-006
- **Supporting requirement families:** FR-REV-*, FR-WFA-*, FR-INV-*, FR-AR-*, FR-GL-*, FR-FX-*, FR-RPT-*

### WF-6.6 — Journal Entry Posting and Reversal
- **DDD source:** §6.6
- **Primary actor:** Accountant or authorized subledger
- **Supporting actors:** As defined by DDD
- **Owning capability:** General Ledger
- **Primary named operations:** SubmitPostingRequest, ApplyJournalApprovalDecision, and ReverseJournalEntry
- **Functional objective:** Journal Entry Posting and Reversal
- **Expected functional behavior:** Actor or subledger submits a posting request with an idempotency key, request fingerprint, source reference, expected period-state version, and expected posting-gate version. GL checks duplicate processing and returns the existing in-progress or terminal result when already handled. GL validates the request structure, source and accounting scope, debit-credit equality, ledger and accounting-book relationships, account restrictions, segment combinations, currency precision, posting purpose, authorization, and approval policy.
- **Direct requirement IDs:** FR-GL-001, FR-GL-002, FR-GL-003
- **Supporting requirement families:** FR-GL-*, FR-WFA-*

### WF-6.7 — Customer Receipt Recording with Partial Application
- **DDD source:** §6.7
- **Primary actor:** Cash Applications Specialist
- **Supporting actors:** Bank Feeds & Reconciliation, collections specialist, AR accounting-recovery domain service
- **Owning capability:** Accounts Receivable
- **Primary named operations:** RecordReceipt, ApplyReceipt, UnapplyReceipt, and RollbackUnpostedApplicationBatch
- **Functional objective:** The specialist records a receipt. When allocations are supplied or later confirmed, the specialist applies some or all of the unapplied amount to receivable open items.
- **Expected functional behavior:** The specialist submits RecordReceipt linked to a normalized bank transaction or approved manual source using a source fingerprint, expected source state, idempotency key, and command fingerprint. As one RecordReceipt domain outcome, AR creates CustomerReceipt with AppliedAmount = 0, UnappliedAmount = ReceiptAmount, and ReceiptAccountingStatus = PostingPending, and records ReceiptRecorded plus the posting domain event. AR submits the receipt-recording posting to GL: debit cash or bank clearing and credit unapplied cash. An identical repeated posting command returns the established result and never recreates the receipt. AR records the returned journal reference and changes the receipt-accounting status to Posted; a terminal posting failure changes it to PostingFailed and establishes a domain exception visible to reconciliation and close controls.
- **Direct requirement IDs:** FR-AR-002, FR-AR-003, FR-AR-004, FR-AR-005
- **Supporting requirement families:** FR-AR-*, FR-BFR-*, FR-GL-*

## 8. Additional Functional Scenarios
### WF-7.1 — Vendor Invoice Registration, Matching, Approval, Dispute, and Void
- **DDD source:** §7.1
- **Actors and ownership:** AP Specialist, AP approver, and procurement-data provider. AP owns RegisterVendorInvoice, ValidateVendorInvoice, ApplyVendorInvoiceApprovalDecision, DisputeVendorInvoice, and VoidVendorInvoice; Workflow owns the underlying approval decision; VendorInvoice is the consistency boundary and AP is the sole liability-posting producer.
- **Functional behavior:** Received -> Validated -> PendingApproval -> Approved -> PartiallyPaid -> Paid, with DuplicateSuspected, Disputed, Rejected, and Voided alternatives. Events include VendorInvoiceRegistered, VendorInvoiceMatched, VendorInvoiceApproved, VendorInvoiceDisputed, and VendorInvoiceVoided.
- **Controls:** Vendor and accounting scope are active; immutable purchase-order and receipt snapshots are versioned; duplicate fingerprint is unique in the configured vendor, entity, invoice-number, date, and amount scope; matching tolerance and tax policy are versioned; an approved or posted invoice is not edited in place.
- **Failure and recovery:** Posting failure leaves the invoice approved with visible PostingPending or PostingFailed; AP retries the same posting identifier. An uncertain GL outcome is reconciled by the stable posting identity and returns the existing result. Procurement snapshot changes require explicit rematch and a new invoice version.
- **Direct requirement IDs:** FR-AP-001, FR-AP-006, FR-AP-008, FR-AP-009, FR-AP-010
- **Supporting requirement families:** FR-AP-*, FR-WFA-*, FR-OMD-*

### WF-7.2 — Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation
- **DDD source:** §7.2
- **Actors and ownership:** Payment preparer, independent approver, treasury operator, provider, reconciler, and owning obligation contexts. Payments owns batch and instruction execution, expectation registration and resolution, observed receipts, evidence-backed no-journal cancellation, owner acknowledgement, receipt reversal, and linked outgoing payment returns; Workflow owns approval decisions. Each PaymentInstruction, PaymentReturn, ExpectedIncomingSettlement, and SettlementReceipt is independently versioned.
- **Functional behavior:** A batch reaches Completed with FullySettled, FullyCancelled, PartiallySettledCancelled, or CompletedWithExceptions; whole-batch Cancelled applies only when CancelPaymentBatch succeeds before any instruction is provider-submitted. An instruction can reach Settled, Cancelled, PartiallySettledCancelled, or owner-decided terminal ExceptionResolved; the latter is required before CompletedWithExceptions. A provider return first reserves money, then moves it to posted return only after the cash correction, waits for owner application, and finally reconciles; a terminally unposted return releases only its reservation through evidence-backed CancelledNoJournal. Typed return-exception outcomes cover corrected application, approved reclassification, accepted exception, or rejection with a required linked reversal. Original return-posting failure and posted-return reversal use separate lifecycle states. An expectation supports typed exception resolution, expiry, reconciliation, cancellation, closure, and reopening after receipt reversal when still collectible. Excess bank allocations are posted to unallocated incoming cash clearing and then resolved independently. A receipt has independent validation, posting, owner-application, and reconciliation states, including reversal from an owner-rejected exception.
- **Controls:** Funding and beneficiary references are valid; control totals balance by currency; preparer and approver are segregated; instruction authorized amount equals settled plus cancelled plus remaining; net settlement equals gross settlement minus posted returns plus linked return reversals; cumulative active returns net of reversal cannot exceed gross settlement; an expectation owns nonnegative cumulative allocation, reconciliation and rollback references under expected equals received plus remaining; excess bank allocations remain separate; named Payments consistency rules coordinate instruction with return and expectation with receipt.
- **Failure and recovery:** A permanently unposted receipt or a return in PostingFailed can enter its CancellingNoJournal path only after authoritative posting-cancellation evidence; a posted return instead follows ReversalPending and can never use no-journal cancellation. GL no-journal proof permits one all-or-nothing expectation-allocation rollback or reserved-return rollback; the original bank evidence remains immutable. Nonterminal return, expectation or validation exceptions require explicit resolution commands; instruction exceptions require an owning-context decision. Failed partial payment preserves cumulative settlement. Unallocated excess posting failures retry the original posting identifier. Lost outcomes and provider returns recover by authoritative provider or bank evidence; reserved plus posted returns minus authoritative linked reversals cannot exceed instruction gross settlement.
- **Direct requirement IDs:** FR-PCM-003
- **Supporting requirement families:** FR-PCM-*, FR-WFA-*, FR-AP-*, FR-PAYR-*, FR-TAX-*, FR-FA-*, FR-AR-*

### WF-7.3 — Customer Credit, Refund, Overpayment, Chargeback, and Write-Off
- **DDD source:** §7.3
- **Actors and ownership:** AR Specialist, collections manager, Workflow approver, Payments, provider, and reconciler. AR owns credit notes, receivable adjustments, overpayment and unapplied-cash decisions, CustomerRefundRequest, chargebacks, refunds, and write-offs; Payments owns external refund execution and bank cash. AR is the sole producer of receivable, refund-payable, and refund-clearing accounting.
- **Functional behavior:** Credit, chargeback, and write-off commands create immutable adjustment records. Approval rejection moves a refund request to Rejected, while CancelCustomerRefundRequest moves a draft, pending-approval, or approved request with no payment request to Cancelled and records CustomerRefundRequestCancelled. An approved refund creates CustomerRefundRequest; AR posts a refund-payable-to-payment-clearing leg and publishes CustomerRefundPaymentRequested. Payments creates the correlated instruction and publishes partial, settled, failure, remainder-cancelled, or returned outcomes. AR applies settlement outcomes idempotently. AR publishes CustomerRefundPaymentCancellationRequested when cancellation of an outstanding payment is requested. The authoritative Payments remainder-cancelled outcome moves a zero-gross-settlement refund to Cancelled, moves a partially settled refund to PartiallySettledCancelled, and restores the unpaid refund obligation or produces CustomerRefundPaymentReplacementRequested. A return moves the refund to ReturnCorrectionPending; correction posting failure becomes ReturnCorrectionPostingFailed and retries the same identifier, while an approved irrecoverable case becomes ReturnCorrectionException. Only after AR posts the linked clearing-to-refund-payable correction do returned and remaining amounts increase, net settlement decrease, PaymentReturnApplied publish, and a new clearing leg and replacement instruction become eligible.
- **Controls:** Original invoice, credit, or receipt exists; adjustment reason and authorization are valid; cumulative credits and write-offs do not exceed authoritative open balance; a refund does not exceed refundable unapplied cash or approved credit; AuthorizedMoney = NetSettledMoney + CancelledMoney + RemainingMoney and NetSettledMoney = GrossSettledMoney - ReturnedMoney; every clearing leg, instruction and return remains immutable; original invoice and receipt facts remain immutable.
- **Failure and recovery:** Failed accounting retains the established adjustment or refund intent with visible pending state and retries the same identifier. Provider refund failure does not reverse AR authorization automatically. A partially settled cancellation exposes and restores the unpaid remainder; a returned payment uses linked correction records; lost outcomes are reconciled from the instruction or return identifier.
- **Direct requirement IDs:** FR-AR-006, FR-AR-007, FR-AR-008, FR-AR-009, FR-AR-010, FR-AR-011, FR-AR-012, FR-AR-013, FR-AR-014, FR-AR-015, FR-AR-016
- **Supporting requirement families:** FR-AR-*, FR-PCM-*, FR-WFA-*

### WF-7.4 — Bank Statement Import, Matching, Unmatching, and Reconciliation
- **DDD source:** §7.4
- **Actors and ownership:** Reconciliation Specialist, bank-feed provider, AR, AP, and Payments. Bank Feeds & Reconciliation owns ImportStatement, ProposeMatch, ConfirmMatch, Unmatch, and CompleteReconciliation; it owns matching records but not subledger business facts or cash-settlement postings.
- **Functional behavior:** Statement Imported -> Validated -> Matching -> Reconciled, with Rejected or Exception paths. Session matches may be one-to-one, split, aggregate, or manual. MatchConfirmed, MatchReversed, and ReconciliationCompleted retain evidence versions.
- **Controls:** Bank account is active; statement opening balance equals the prior accepted closing balance or an approved exception exists; import fingerprint is unique; confirmed match allocations equal the statement-line amount within configured tolerance.
- **Failure and recovery:** Partial imports remain unaccepted. An interrupted confirmed match is reconciled by event identity and returns the prior result. Unmatch creates a compensating match record and may trigger owning-context correction; it never deletes audit history.
- **Direct requirement IDs:** FR-BFR-001, FR-BFR-002, FR-BFR-003, FR-BFR-004, FR-BFR-005
- **Supporting requirement families:** FR-BFR-*, FR-AR-*, FR-AP-*, FR-PCM-*

### WF-7.5 — Foreign-Currency Invoice Settlement and Realized FX
- **DDD source:** §7.5
- **Actors and ownership:** AR accountant for customer receipts, AP accountant for vendor invoices, Payments for AP bank execution, and Multi-Currency as immutable rate-evidence publisher. AR owns customer-receipt cash, receivable clearing, and customer-settlement realized FX. AP owns vendor-invoice clearing and vendor-settlement realized FX. Payments owns only the bank-cash leg of AP payment instructions.
- **Functional behavior:** The scenario follows the DDD-defined lifecycle and outcomes.
- **Controls:** Invoice currency, settlement currency, functional carrying amount, settlement amount, rate-set version, and authoritative receipt or payment evidence are available. Currency scale and arithmetic are valid, settlement cannot exceed the open item, and every cross-context leg uses a correlated clearing reference. Every PostingRequest has one declared transaction currency. A FunctionalOnlyAdjustment line carries zero transaction amount and a nonzero functional amount and is excluded from source-currency quantity reconciliation.
- **Failure and recovery:** If one leg posts before the others, the transaction-currency and functional-currency clearing positions remain visible and either block period-close controls or require an explicitly approved exception. Retry uses the original posting identifiers. Settlement reversal creates linked corrections and preserves the original transaction amounts, functional amounts, line modes, and rate evidence.
- **Direct requirement IDs:** None named
- **Supporting requirement families:** FR-AP-*, FR-AR-*, FR-FX-*, FR-PCM-*, FR-GL-*

### WF-7.6 — Period-End Revaluation, Rerun, and Next-Period Reversal
- **DDD source:** §7.6
- **Actors and ownership:** Treasury or GL accountant, Workflow approver, Multi-Currency, and GL. Multi-Currency owns RunRevaluation, ApplyRevaluationApprovalDecision, PostRevaluationRun, calculation results, reruns, and reversal instructions; Workflow owns the approval decision, and Multi-Currency is the sole producer of unrealized FX adjustments.
- **Functional behavior:** Draft -> Calculating -> PendingApproval -> Approved -> Posting -> Completed, with Rejected, Failed, Superseded, and Reversed states. Workflow records the immutable decision; Multi-Currency applies it through ApplyRevaluationApprovalDecision after revalidating the run version, source watermark, rate set, and policy. That command records Approved but creates no posting. PostRevaluationRun then changes the run to Posting as one domain consistency outcome, assigns immutable posting identifiers, and establishes the posting domain events. Events carry the approval reference, source watermark, rate-set version, result totals, journal references, and reversal references.
- **Controls:** Eligible monetary balances and period-end rate set are frozen by ledger watermark; policy version defines accounts, grouping, and reversal date; a run covers one accounting scope, period, and rate set.
- **Failure and recovery:** Calculation failure creates no posting. Partial GL success is reconciled per posting identifier before retry. Reversal failure remains visible and blocks later revaluation completion when policy requires a clean reversal.
- **Direct requirement IDs:** FR-FX-002, FR-FX-003, FR-FX-004
- **Supporting requirement families:** FR-FX-*, FR-GL-*, FR-FPM-*, FR-WFA-*

### WF-7.7 — Full Fixed-Asset Lifecycle and Disposal Variants
- **DDD source:** §7.7
- **Actors and ownership:** Fixed Asset Accountant, project accountant, Workflow approver, GL, AP, and Payments. Fixed Assets owns acquisition, construction in progress, capitalization, transfer, split, depreciation, impairment, disposal proposal, approval application, treatment selection, required posting-leg orchestration, cancellation, asset-specific clearing, and posted-disposal correction; AP alone owns supplier liabilities and Payments alone owns bank cash.
- **Functional behavior:** Acquisition or CIP progresses to Capitalized, then Active, with transfer, split, impairment, and disposal subflows. Disposal accounting can pass through PendingPosting, PostingFailed, PartiallyPosted, CancellingNoJournal, CancelledNoJournal, Compensating, CompensatedFailed, and Posted; Posted requires every treatment-defined leg to have an authoritative GL result. After Posted, proceeds settlement and no-supplier cost settlement progress independently. Receipt reversal returns proceeds to Expected or PartiallySettled; a disposal-cost payment return returns the no-supplier path to PaymentRequested or PartiallySettled and may create a replacement instruction. The supplier path terminates locally at SupplierLiabilityPosted.
- **Controls:** Asset cost, components, useful life, residual value, ownership scope, and source evidence are valid; carrying amount and component quantities cannot become negative; depreciation does not precede capitalization or continue after disposal; one active protected operation exists per asset portion; only a declared DisposalAccountingTreatment can determine the required posting legs.
- **Failure and recovery:** Protected asset state remains explicit across PendingPosting, PartiallyPosted, PostingFailed, CancellingNoJournal, and Compensating. When no leg posted, evidence-backed cancellation and GL no-journal proof restore the asset and reach CancelledNoJournal. When mixed success is irrecoverable, linked reversals for every successful leg must post before the asset is restored and the disposal reaches CompensatedFailed; reversal failure remains visible and protected. Settlement expectations and supplier classifications are emitted only after all required legs post. AP and Payments failures, incoming receipt reversals, and outgoing payment returns recover by clearing, instruction, return, expectation, or receipt reference; they update only orthogonal settlement balances and never mutate posted asset accounting.
- **Direct requirement IDs:** FR-FA-001, FR-FA-002, FR-FA-003, FR-FA-004, FR-FA-005, FR-FA-006, FR-FA-007, FR-FA-008, FR-FA-018, FR-FA-019, FR-FA-020, FR-FA-021
- **Supporting requirement families:** FR-FA-*, FR-AP-*, FR-PCM-*, FR-WFA-*, FR-GL-*

### WF-7.8 — Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration
- **DDD source:** §7.8
- **Actors and ownership:** Revenue Accountant, Workflow approver, Invoicing, and AR. Revenue Recognition owns modification assessment, ApplyContractModificationApprovalDecision, and schedule/profile versions; Workflow owns approval decisions and AR owns billing credits and refunds.
- **Functional behavior:** ModifyContract classifies separate-contract, prospective, or cumulative-catch-up treatment and enters PendingApproval when policy requires it. Workflow records the immutable decision; Revenue Recognition applies it through ApplyContractModificationApprovalDecision after revalidating the active contract and schedule versions. Renewal creates a linked term and schedule; cancellation or termination supersedes future recognition; variable consideration changes produce new allocation and schedule versions. Profile events identify effective invoice-line classification.
- **Controls:** Original contract, recognized-to-date, billed-to-date, remaining obligations, modification terms, and policy version are available; cumulative recognition cannot exceed constrained allocated consideration.
- **Failure and recovery:** Failed catch-up or reclassification remains pending and blocks final modification completion. A posted invoice correction is never replaced by RevenueRecognition; AR executes credit or replacement flows.
- **Direct requirement IDs:** FR-REV-004, FR-REV-005
- **Supporting requirement families:** FR-REV-*, FR-AR-*, FR-INV-*, FR-WFA-*, FR-GL-*

### WF-7.9 — Consolidation, Ownership Changes, Translation, Eliminations, and Rerun
- **DDD source:** §7.9
- **Actors and ownership:** Consolidation Accountant, Workflow approver, Multi-Currency, Intercompany, and Financial Reporting. Multi-Currency owns rate selection and versioned TranslationRun calculations. Financial Reporting owns RunConsolidation, ApplyConsolidationApprovalDecision, translated balances, CTA records, ownership calculations, elimination records, and published consolidated statements. Workflow owns the publication approval decision, and Intercompany supplies versioned elimination instructions.
- **Functional behavior:** Draft -> Collecting -> Translating -> Eliminating -> PendingApproval -> Approved -> Published, with Rejected, Failed, and Superseded paths. Reporting requests or consumes immutable translation results, records translated balances and CTA in the consolidation workspace, and applies elimination instructions. Workflow records the immutable publication decision; Reporting applies it through ApplyConsolidationApprovalDecision after revalidating all frozen input versions and moves the run to Approved. PublishConsolidatedStatement then publishes the statement version idempotently. Rerun creates a new version linked to the superseded run and statement.
- **Controls:** Participant scopes, ownership percentages and effective dates, source ledger watermarks, rate policy, TranslationResult versions, mapping versions, and elimination instructions are frozen for the run. Ownership totals and noncontrolling-interest calculations follow configured policy.
- **Failure and recovery:** Missing entity data, missing or conflicting translation results, elimination imbalance, or translation-validation failure blocks publication. A restart resumes from recorded domain-process results. Published corrections create a revised statement and retain the prior publication.
- **Direct requirement IDs:** FR-RPT-001, FR-RPT-003, FR-RPT-004
- **Supporting requirement families:** FR-RPT-*, FR-FX-*, FR-IC-*, FR-WFA-*

### WF-7.10 — Tax Return Submission, Rejection, Amendment, Payment, and Evidence
- **DDD source:** §7.10
- **Actors and ownership:** Tax Accountant, Workflow approver, tax-authority connector, Payments, and GL. Tax Filing owns returns, filing submissions, amendment lineages, return-level adjustment aggregates, evidence, TaxPaymentObligation, and filing or obligation status. Source subledgers own transaction-level tax. Workflow owns approval decisions. Payments owns payment instructions and authoritative bank-cash settlement.
- **Functional behavior:** The scenario follows the DDD-defined lifecycle and outcomes.
- **Controls:** Jurisdiction, period, configuration version, source totals, due date, credential reference, and authorization are valid. A TaxAmendment references an accepted original return and version; an accepted TaxReturn is never changed to an amended state. Submitted filing-content versions are immutable. Payment status cannot change filing acceptance status. A return-level adjustment cannot post without a complete Workflow decision reference and a stable source return or amendment version.
- **Failure and recovery:** An uncertain tax-authority outcome is reconciled against authoritative filing status before another submission. Rejection creates a corrected submission attempt or a new amendment version; accepted records are not overwritten. A failed return-level adjustment remains PostingFailed and retries the original posting identifier after reconciling with GL. Payment failure leaves the filing accepted and the obligation outstanding; retry or replacement uses explicit linked records.
- **Direct requirement IDs:** FR-TAX-010
- **Supporting requirement families:** FR-TAX-*, FR-WFA-*, FR-PCM-*, FR-GL-*

### WF-7.11 — Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment
- **DDD source:** §7.11
- **Actors and ownership:** Payroll Administrator, Workflow approver, employee-payment operator, Tax Filing, and Payments. Payroll owns calculations, ApplyPayrollRunApprovalDecision, corrections, payroll liabilities, and payroll posting requests; Workflow owns the approval decision, Payments owns cash execution, and Tax Filing owns statutory filing and amendment status.
- **Functional behavior:** Regular or off-cycle run progresses Draft -> Calculated -> PendingApproval -> Approved -> Posted -> PaymentPending -> Settled, with Rejected, employee-level payment failure, and linked correction-run alternatives. Workflow records the immutable decision; Payroll applies it through ApplyPayrollRunApprovalDecision after revalidating the run, employee-result, and policy versions.
- **Controls:** Pay group, period, employee profile and tax references, prior run, correction reason, and authorization are valid; gross minus deductions equals net; confidential access is least privilege; a finalized run is corrected through a linked run, not overwritten.
- **Failure and recovery:** A failed employee payment remains outstanding without reversing payroll expense. Retry or alternate payment uses the same business obligation. Posted calculation correction creates a linked off-cycle or correction run.
- **Direct requirement IDs:** FR-PAYR-001, FR-PAYR-002, FR-PAYR-003, FR-PAYR-004, FR-PAYR-007
- **Supporting requirement families:** FR-PAYR-*, FR-WFA-*, FR-PCM-*, FR-TAX-*, FR-GL-*

### WF-7.12 — Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen
- **DDD source:** §7.12
- **Actors and ownership:** Controller, close operator, recovery operator, independent approver, Fiscal Period Management, Workflow, and GL. Fiscal Period Management owns process state and ReopenRequest; Workflow owns approval decisions; GL owns PeriodPostingGate. TakeOverPeriodControl, ExtendCloseException, OpenOperationalReopenGate, CloseOperationalReopenGate, and BeginRecloseGate supplement Sections 6.1 and 6.2.
- **Functional behavior:** StartSoftClose creates control epoch 1. A prolonged outage preserves the restrictive gate. After control authority expires and takeover is approved, takeover changes the process owner while retaining the process identity and history. Late adjustments require explicit cutoff classification. Exception expiry blocks finalization until renewed or resolved. A released initial close opens the next immutable soft-close control epoch before another handoff. A full operational reopen uses ReopenMode = Operational; GL changes the gate from HardClosed to OperationalReopen, admits only approved OperationalReopen postings for the bounded business transaction classes and actors, and records whether any posting was admitted. CloseOperationalReopenGate either restores HardClosed for an audited no-change outcome or retains the reopen request as owner in CloseOnly; BeginRecloseGate then transfers ownership to the candidate reclose run as one exclusive domain consistency outcome before reclose.
- **Controls:** Existing process state, control-owner status or authority expiry, gate status, independent approval, cutoff policy, permitted posting classes, actor or authorized-subject scope, expiry, and expected versions are verifiable. Only one process owns a scoped period. Takeover and operational reopen never change GL state without the expected gate version and control authority epoch.
- **Failure and recovery:** Process recovery uses authoritative gate status and resumes recorded domain-process states. An unreachable dependency keeps the gate restrictive. A finalized gate is never released. Operational-reopen expiry rejects new postings, moves the request to ExpiredPendingClosure, and does not remove process ownership. If the authoritative retained gate-admission summary records zero postings, the GL gate may close directly to HardClosed under audited policy and Fiscal Period Management idempotently records CompletedNoChange. If the summary records any posting, BeginRecloseGate must transfer ownership to a reclose run as one exclusive domain consistency outcome before finalization; that reclose barrier cannot be released back to the reopen request, and recovery resumes the mandatory reclose across the handoff.
- **Direct requirement IDs:** FR-FPM-001, FR-FPM-012, FR-FPM-013, FR-GL-011, FR-GL-012, FR-GL-013
- **Supporting requirement families:** FR-FPM-*, FR-GL-*, FR-WFA-*, FR-AUD-*

### WF-7.13 — Cross-Context Event Interpretation, Ordering, and Replay
- **DDD source:** §7.13
- **Actors and ownership:** Every receiving bounded context owns the interpretation of a published event and the resulting local domain effect. A domain steward owns any policy for authorized deferral, rejection, correction, or replay.
- **Functional behavior:** A receiving context validates the event and then applies it, defers it pending prerequisites, rejects it, or records an exception for authorized resolution. Out-of-order events are deferred, rejected by version, or applied only when the receiving context defines a commutative domain rule.
- **Controls:** Events carry event identity, semantic contract version, accounting or business scope, correlation reference, causation reference, and source aggregate version. Unknown contracts and invalid scopes change no domain state. One event identity can produce at most one local business effect per receiving context.
- **Failure and recovery:** Invalid or unprocessable events retain their identity, fingerprint, reason, and authorized resolution. Reconstruction begins from a known domain position, observes ordering rules, and verifies the expected domain outcomes.
- **Direct requirement IDs:** GFR-006, GFR-007, GFR-008, GFR-009, GFR-012, GFR-013, GFR-014
- **Supporting requirement families:** Each affected receiving-capability family

### WF-7.14 — Concurrent Aggregate and Domain-Process Modification Rules
- **DDD source:** §7.14
- **Actors and ownership:** Business user, automated actor, and the owning bounded context for invoices, payment instructions, payment batches, close runs, settlement runs, and revenue schedules.
- **Functional behavior:** The owner validates authorization, expected version, active lifecycle state, and protected-operation flags; it establishes one transition and its domain events or returns a typed conflict with the current version and safe retry guidance.
- **Controls:** Every command carries the expected aggregate version and immutable command fingerprint. Operations declare whether they commute, conflict, or require a named multi-aggregate consistency boundary.
- **Failure and recovery:** An ambiguous outcome is resolved by querying the command result through its idempotency identity. Concurrency-conflict retry preserves the command fingerprint. A superseded actor or process epoch cannot establish a later transition.
- **Direct requirement IDs:** GFR-006, GFR-007, GFR-008, GFR-009, GFR-012, GFR-013, GFR-014
- **Supporting requirement families:** Each affected owning-capability family

### WF-7.15 — Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation
- **DDD source:** §7.15
- **Actors and ownership:** Auditor, security operator, incident commander, and Audit Integrity. Audit Integrity owns the state-changing commands AppendAuditableEvent, CreateAuditSeal, RotateVerificationCredential, and EscalateIntegrityIncident. VerifyProof is an authoritative domain reference operation that evaluates evidence without changing aggregate state.
- **Functional behavior:** Chain state progresses through appended events and periodic seals. Verification returns Valid, MissingEvent, ProofMismatch, InvalidProof, or UnsupportedVersion. Verification-credential rotation closes the old credential interval and begins a new one without rewriting prior seals. A corrective seal supersedes only the prior proof result, not source events.
- **Controls:** Audit sequence is contiguous within an AuditScope; each event retains its integrity-format version, event fingerprint, prior-event fingerprint, and recorded time; seals and verification-credential references are immutable; secret credential material is not part of domain events.
- **Failure and recovery:** A missing event or proof mismatch suspends proof status for the affected range, preserves evidence, and opens an incident. Recovery re-establishes missing audit evidence from authoritative source events; it never fabricates or edits an event. Verification-credential compromise triggers rotation, impact analysis, and replacement seals where policy permits.
- **Direct requirement IDs:** FR-AUD-001, FR-AUD-002, FR-AUD-003, FR-AUD-004, FR-AUD-005
- **Supporting requirement families:** FR-AUD-*, FR-IAM-*

## 9. Functional Workspaces and User Experience Expectations
- Role-based home pages shall show actionable worklists, approvals, exceptions, reconciliation items, close tasks, and overdue obligations.
- Record views shall show business identity, accounting scope, current state, version, amount/currency, owner, approval, posting, settlement, correction lineage, and audit history.
- Available actions shall be state-aware and permission-aware; disabled actions shall show the blocking rule.
- Multi-step processes shall show completed, current, pending, failed, compensated, and reconciled steps with the responsible capability.
- Search and list views shall support saved filters, export, and evidence links while respecting data-sensitivity rules.
- Financial users shall be able to distinguish business rejection, approval rejection, posting failure, provider failure, reconciliation exception, and authorization denial.

## 10. Reporting and Evidence
- Provide operational status reports for invoices, receipts, applications, refunds, payments, returns, incoming settlements, bank reconciliation, assets, revenue schedules, tax filings, payroll corrections, and period control.
- Provide financial statements and consolidation outputs tied to report-definition versions and source ledger watermarks.
- Provide exception-aging, unresolved-approval, failed-posting, unreconciled-settlement, and close-blocker reports.
- Provide user-visible lineage from source business fact to approval, posting, settlement, reversal, amendment, replacement, and audit evidence.

## 11. Dependencies and Assumptions
- The DDD v3.1 baseline remains authoritative for vocabulary, ownership, invariants, commands/events, lifecycle semantics, and correction behavior.
- Procurement supplies immutable purchase-order and receipt snapshots and is outside this product scope.
- External providers may supply bank, payment, tax, payroll, and rate evidence, but the product remains authoritative for the business state assigned by the DDD baseline.
- Country-specific statutory, localization, and regulatory details require separately approved functional extensions.

## 12. Functional Acceptance and Sign-Off
- Every functional requirement shall trace to one or more DDD sections and applicable functional acceptance scenarios.
- Product, Finance SME, QA, Security, UX, and Architecture reviewers shall approve scope and behavior before solution design is baselined.
- A requirement shall not change accounting ownership, lifecycle semantics, equations, or correction behavior without a corresponding approved DDD change.

## 13. Consistency Review Corrections through Version 1.5
1. Workflow headings use `WF-6.x` and `WF-7.x`, preventing DDD source numbering from being mistaken for this document’s local section hierarchy.
2. PRD-defined functional actions are explicitly distinguished from DDD commands and domain events.
3. Incorrect event outcomes produced by automated name matching were removed; user-visible results are stated functionally and DDD outcomes remain authoritative in the source sections.
4. Detailed DDD commands omitted from the representative command table are included without renumbering Version 1.0 requirements.
5. Workflow traceability includes exact requirement IDs where named operations are present, plus supporting capability families where behavior is broader than a named command.
6. Authorization now applies the dimensions relevant to each action rather than implying that every dimension is mandatory for every action.
7. Legal-hold behavior now overrides ordinary retention-based destruction until the hold is formally released.
8. Commands defined outside DDD §5.5 are labeled `DDD detailed command`; the COA requirement rows are ordered by stable requirement ID.
9. PRD-only functional actions use action-specific user-visible outcomes instead of a generic effective-date result that did not fit validation, invoice generation, or emergency-access actions.

10. Capability control baselines are explicitly conditional; only controls relevant to the operation are mandatory.
11. DDD-backed requirement rows now include exact operation source locators in addition to their capability-model sources.
12. Unsupported `LegalEntity`, customer-profile, and invoice-template lifecycle assumptions were removed from PRD-only actions.
13. Cross-cutting scenario mappings no longer use the ambiguous supporting-family value `ALL`, and §14.13 acceptance subgroups use the correct heading hierarchy in the companion document.

14. Added explicit functional requirements for DDD-scoped authoritative records and scenario behavior that previously appeared only in the capability map or workflow prose: GL configuration, AR overpayment/chargeback/write-off, payroll profiles and payroll-source filings, bank accounts, report definitions and ledger statements, intercompany agreements and source transactions, fixed-asset impairment/transfer/split/posted-disposal correction, bank-feed connections, tax configurations, and approval policies.
15. Workflow traceability now uses supporting families only for cross-cutting dependencies; explicit workflow behavior has direct functional requirement IDs.

16. Corrected PRD-only outcomes that asserted effective dates, versions, or direct bank-account linkage not defined by the DDD record model; effective dates and versions are now stated only where the source defines or requires them.
17. Reclassified configuration and source-data maintenance as supporting dependencies rather than direct workflow actions, and mapped cross-context event and concurrency scenarios directly to their governing global requirements.
18. Corrected two operation-source locators and converted the functional review checklist from an unexecuted template to a completed review record.

## 14. Verification Checkpoint
| Checkpoint field | Value |
|---|---|
| Verified body SHA-256 | `f5b7be0973f532851abf06cf8c408caf110dad538735aa21490c9ff8b84a2b8f` |
| Hash boundary | UTF-8 bytes from title through the blank line immediately preceding ## 14. Verification Checkpoint; checkpoint section excluded |
| Checkpoint ID | FPRD-1.5-2026-07-24 |
| Source baseline | DDD v3.1 / DDD-3.1-2026-07-24 |
| Source verified-content SHA-256 | `a9d437d23656c36d340afb3a5a31c93a23e574f53db186483a9edfdf32d3e652` |
| Global requirements | 22 |
| Capability requirements | 193 |
| Core workflows | 7 |
| Additional scenarios | 15 |
| Review result | Passed after Version 1.5 semantic, source-traceability, workflow-classification, and checklist corrections |
| Review rule | When this hash and the companion hashes remain unchanged, repeat only structural validation. Re-run affected capability/workflow/acceptance review when requirement meaning or the DDD source hash changes. |
