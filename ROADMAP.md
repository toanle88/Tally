# TALLY Roadmap

> Flat epic checklist. Every `DLV-*` item from the delivery plan is listed under an owning roadmap epic.
> `[x]` means complete or intentionally closed; `[ ]` means open, planned, or awaiting qualification evidence.
> Workflow and NFR delivery tables do not define an Epic column, so those items are grouped under their primary roadmap epic for navigation.

**Last updated:** 2026-09-22

## [ ] EP-PLAT-001 — Engineering foundation (M0)

Repository, Go and React applications, PostgreSQL, migrations, sqlc, OpenAPI, shared finance primitives, idempotency, integration workers, and pull-request quality checks.

- [x] `DLV-PLAT-001` — Create monorepo with Go API, React application and shared commands.
- [x] `DLV-PLAT-002` — Create Docker Compose PostgreSQL development environment.
- [x] `DLV-PLAT-003` — Establish Goose migrations, pgx and sqlc workflow.
- [x] `DLV-PLAT-004` — Establish OpenAPI-first REST workflow.
- [x] `DLV-PLAT-005` — Implement shared money, currency, accounting-scope, identity and version primitives.
- [x] `DLV-PLAT-006` — Implement request fingerprint and idempotency foundation. (focused evidence passes; environment-dependent persistence qualification remains)
- [x] `DLV-PLAT-007` — Implement PostgreSQL outbox/inbox and worker foundation.
- [ ] `DLV-CI-001` — Create pull-request CI quality pipeline. (hosted CI qualification remains)
- [ ] `DLV-GFR-001` — GFR-001 — EP-PLAT-001
- [ ] `DLV-GFR-006` — GFR-006 — EP-PLAT-001
- [ ] `DLV-GFR-007` — GFR-007 — EP-PLAT-001
- [ ] `DLV-GFR-008` — GFR-008 — EP-PLAT-001
- [ ] `DLV-GFR-013` — GFR-013 — EP-PLAT-001
- [ ] `DLV-GFR-021` — GFR-021 — EP-PLAT-001
- [ ] `DLV-GFR-022` — GFR-022 — EP-PLAT-001
- [ ] `DLV-WF-7.13` — WF-7.13 — Cross-Context Event Interpretation, Ordering, and Replay
- [ ] `DLV-WF-7.14` — WF-7.14 — Concurrent Aggregate and Domain-Process Modification Rules

## [ ] EP-UX-001 — Shared UX and design system (M0)

Tailwind and daisyUI application shell, routing, shared finance interaction components, forms, worklists, status surfaces, and accessibility support.

- [x] `DLV-UX-001` — Implement Tailwind/daisyUI application shell and component abstractions.
- [x] `DLV-UX-002` — Implement accessibility test harness. (M0 scope complete; witnessed screen-reader and actual 400% browser-zoom reviews remain release qualification work)
- [ ] `DLV-GFR-009` — GFR-009 — EP-UX-001
- [ ] `DLV-GFR-016` — GFR-016 — EP-UX-001
- [ ] `DLV-GFR-017` — GFR-017 — EP-UX-001

## [x] EP-IAC-001 — Terraform and Azure learning environment (M0)

Closed by owner decision on 2026-09-20. Remaining authenticated Azure exercises and external qualification evidence are intentionally deferred.

- [ ] `DLV-IAC-001` — Create Terraform modules and local state bootstrap.
- [ ] `DLV-IAC-002` — Create optional Azure dev/demo deployment.

## [x] EP-OPS-001 — Observability and operational foundation (M0)

Structured logs, correlation, OpenTelemetry traces, bounded metrics, operational dashboards, alerts, runbooks, and operational evidence.

Locally complete and intentionally closed on 2026-09-22. Environment-dependent
race/repository-native Go checks, Azure/production qualification, paging,
retention, and future capability qualification remain deferred.

- [x] `DLV-OPS-001` — Implement structured logging, correlation and OpenTelemetry. (local implementation and evidence complete; race/repository-native Go and production qualification remain deferred)
- [x] `DLV-OPS-002` — Create baseline operational dashboard and runbook template. (local contracts and readiness evidence complete; external qualification remains deferred)
- [ ] `DLV-GFR-012` — GFR-012 — EP-OPS-001 (capability-level requirement remains open; not closed by the operational foundation)

## [ ] EP-IAM-001 — Identity and access (M1)

Entra authentication, application permissions, accounting-scope authorization, segregation-of-duties controls, and emergency access.

- [ ] `DLV-GFR-002` — GFR-002 — EP-IAM-001
- [ ] `DLV-GFR-003` — GFR-003 — EP-IAM-001
- [ ] `DLV-GFR-015` — GFR-015 — EP-IAM-001
- [ ] `DLV-FR-IAM-001` — FR-IAM-001 — `Manage users`
- [ ] `DLV-FR-IAM-002` — FR-IAM-002 — `Manage roles`
- [ ] `DLV-FR-IAM-003` — FR-IAM-003 — `Manage access policies`
- [ ] `DLV-FR-IAM-004` — FR-IAM-004 — `Manage segregation rules`
- [ ] `DLV-FR-IAM-005` — FR-IAM-005 — `Grant emergency access`
- [ ] `DLV-FR-IAM-006` — FR-IAM-006 — `Revoke emergency access`

## [ ] EP-OMD-001 — Organization and master data (M1)

Legal entities, parties, customer and vendor profiles, and fiscal calendars.

- [ ] `DLV-GFR-019` — GFR-019 — EP-OMD-001
- [ ] `DLV-FR-OMD-001` — FR-OMD-001 — `Maintain legal entities`
- [ ] `DLV-FR-OMD-002` — FR-OMD-002 — `Maintain parties`
- [ ] `DLV-FR-OMD-003` — FR-OMD-003 — `Maintain customer profiles`
- [ ] `DLV-FR-OMD-004` — FR-OMD-004 — `Maintain vendor profiles`
- [ ] `DLV-FR-OMD-005` — FR-OMD-005 — `Maintain fiscal calendars`
- [ ] `DLV-FR-OMD-006` — FR-OMD-006 — `Publish approved master-data changes`

## [ ] EP-COA-001 — COA segment configuration (M1)

Segment definitions, segment values, account combinations, and approved segment changes.

- [ ] `DLV-FR-COA-001` — FR-COA-001 — `Maintain segment definitions`
- [ ] `DLV-FR-COA-002` — FR-COA-002 — `Maintain segment values`
- [ ] `DLV-FR-COA-003` — FR-COA-003 — `Validate segment combinations`
- [ ] `DLV-FR-COA-004` — FR-COA-004 — `Request segment changes`
- [ ] `DLV-FR-COA-005` — FR-COA-005 — `ApplySegmentChangeApprovalDecision`

## [ ] EP-GL-001 — General Ledger (M2)

Journal validation, approval, posting, reversal, posting gates, ledgers, books, charts of accounts, accounts, and ledger inquiry.

- [ ] `DLV-GFR-005` — GFR-005 — EP-GL-001
- [ ] `DLV-GFR-011` — GFR-011 — EP-GL-001
- [ ] `DLV-FR-GL-001` — FR-GL-001 — `SubmitPostingRequest`
- [ ] `DLV-FR-GL-002` — FR-GL-002 — `ApplyJournalApprovalDecision`
- [ ] `DLV-FR-GL-003` — FR-GL-003 — `ReverseJournalEntry`
- [ ] `DLV-FR-GL-004` — FR-GL-004 — `EnterSoftCloseGate`
- [ ] `DLV-FR-GL-005` — FR-GL-005 — `ExitSoftCloseGate`
- [ ] `DLV-FR-GL-006` — FR-GL-006 — `AcquirePostingBarrier`
- [ ] `DLV-FR-GL-007` — FR-GL-007 — `ReleasePostingBarrier`
- [ ] `DLV-FR-GL-008` — FR-GL-008 — `FinalizePostingGate`
- [ ] `DLV-FR-GL-009` — FR-GL-009 — `OpenScopedReopenGate`
- [ ] `DLV-FR-GL-010` — FR-GL-010 — `CloseScopedReopenGate`
- [ ] `DLV-FR-GL-011` — FR-GL-011 — `OpenOperationalReopenGate`
- [ ] `DLV-FR-GL-012` — FR-GL-012 — `CloseOperationalReopenGate`
- [ ] `DLV-FR-GL-013` — FR-GL-013 — `BeginRecloseGate`
- [ ] `DLV-FR-GL-014` — FR-GL-014 — `GetPostingGateStatus`
- [ ] `DLV-FR-GL-015` — FR-GL-015 — `Maintain ledgers`
- [ ] `DLV-FR-GL-016` — FR-GL-016 — `Maintain accounting books`
- [ ] `DLV-FR-GL-017` — FR-GL-017 — `Maintain charts of accounts`
- [ ] `DLV-FR-GL-018` — FR-GL-018 — `Maintain accounts and reporting mappings`
- [ ] `DLV-WF-6.6` — WF-6.6 — Journal Entry Posting and Reversal

## [ ] EP-WFA-001 — Workflow and approvals (M3)

Approval policies, requests, decisions, delegation, escalation, and decision application.

- [ ] `DLV-GFR-004` — GFR-004 — EP-WFA-001
- [ ] `DLV-FR-WFA-001` — FR-WFA-001 — `CreateApprovalRequest`
- [ ] `DLV-FR-WFA-002` — FR-WFA-002 — `DecideApprovalRequest`
- [ ] `DLV-FR-WFA-003` — FR-WFA-003 — `DelegateApproval`
- [ ] `DLV-FR-WFA-004` — FR-WFA-004 — `EscalateApproval`
- [ ] `DLV-FR-WFA-005` — FR-WFA-005 — `Maintain approval policies`

## [ ] EP-FPM-001 — Fiscal period management (M3)

Soft close, hard close, reopen, reclose, posting-gate recovery, and period control evidence.

- [ ] `DLV-FR-FPM-001` — FR-FPM-001 — `StartSoftClose`
- [ ] `DLV-FR-FPM-002` — FR-FPM-002 — `EndSoftClose`
- [ ] `DLV-FR-FPM-003` — FR-FPM-003 — `StartHardClose`
- [ ] `DLV-FR-FPM-004` — FR-FPM-004 — `ResumeCloseRun`
- [ ] `DLV-FR-FPM-005` — FR-FPM-005 — `AbortCloseRun`
- [ ] `DLV-FR-FPM-006` — FR-FPM-006 — `ApplyPostingGateResult`
- [ ] `DLV-FR-FPM-007` — FR-FPM-007 — `ApplyCloseExceptionApprovalDecision`
- [ ] `DLV-FR-FPM-008` — FR-FPM-008 — `ApplyCloseApprovalDecision`
- [ ] `DLV-FR-FPM-009` — FR-FPM-009 — `RequestReopen`
- [ ] `DLV-FR-FPM-010` — FR-FPM-010 — `ApplyReopenApprovalDecision`
- [ ] `DLV-FR-FPM-011` — FR-FPM-011 — `StartReclose`
- [ ] `DLV-FR-FPM-012` — FR-FPM-012 — `TakeOverPeriodControl`
- [ ] `DLV-FR-FPM-013` — FR-FPM-013 — `ExtendCloseException`
- [ ] `DLV-WF-6.1` — WF-6.1 — Period Close: Hard Close
- [ ] `DLV-WF-6.2` — WF-6.2 — Fiscal Period Reopen and Reclose
- [ ] `DLV-WF-7.12` — WF-7.12 — Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen

## [ ] EP-INV-001 — Invoicing (M4)

Invoice templates, billing schedules, generated invoices, finalization, and AR handoff.

- [ ] `DLV-FR-INV-001` — FR-INV-001 — `Configure invoice templates`
- [ ] `DLV-FR-INV-002` — FR-INV-002 — `Configure billing schedules`
- [ ] `DLV-FR-INV-003` — FR-INV-003 — `Generate invoices`
- [ ] `DLV-FR-INV-004` — FR-INV-004 — `Finalize generated invoices`
- [ ] `DLV-FR-INV-005` — FR-INV-005 — `Recalculate unfinalized invoices`
- [ ] `DLV-FR-INV-006` — FR-INV-006 — `Cancel unfinalized invoices`

## [ ] EP-AR-001 — Accounts Receivable (M4)

Invoices, open items, receipts, applications, unapplications, credits, refunds, and adjustments.

- [ ] `DLV-FR-AR-001` — FR-AR-001 — `IssueCustomerInvoice`
- [ ] `DLV-FR-AR-002` — FR-AR-002 — `RecordReceipt`
- [ ] `DLV-FR-AR-003` — FR-AR-003 — `ApplyReceipt`
- [ ] `DLV-FR-AR-004` — FR-AR-004 — `UnapplyReceipt`
- [ ] `DLV-FR-AR-005` — FR-AR-005 — `RollbackUnpostedApplicationBatch`
- [ ] `DLV-FR-AR-006` — FR-AR-006 — `IssueCreditNote`
- [ ] `DLV-FR-AR-007` — FR-AR-007 — `CreateCustomerRefundRequest`
- [ ] `DLV-FR-AR-008` — FR-AR-008 — `CancelCustomerRefundRequest`
- [ ] `DLV-FR-AR-009` — FR-AR-009 — `ApplyCustomerRefundApprovalDecision`
- [ ] `DLV-FR-AR-010` — FR-AR-010 — `RequestCustomerRefundPayment`
- [ ] `DLV-FR-AR-011` — FR-AR-011 — `CancelCustomerRefundPayment`
- [ ] `DLV-FR-AR-012` — FR-AR-012 — `ApplyCustomerRefundPaymentResult`
- [ ] `DLV-FR-AR-013` — FR-AR-013 — `ApplyPaymentReturn`
- [ ] `DLV-FR-AR-014` — FR-AR-014 — `Resolve customer overpayments`
- [ ] `DLV-FR-AR-015` — FR-AR-015 — `Record customer chargebacks`
- [ ] `DLV-FR-AR-016` — FR-AR-016 — `Record receivable write-offs`
- [ ] `DLV-WF-6.7` — WF-6.7 — Customer Receipt Recording with Partial Application
- [ ] `DLV-WF-7.3` — WF-7.3 — Customer Credit, Refund, Overpayment, Chargeback, and Write-Off

## [ ] EP-AP-001 — Accounts Payable (M5)

Vendor invoices, matching, approval, liabilities, payment requests, and settlement-related corrections.

- [ ] `DLV-FR-AP-001` — FR-AP-001 — `RegisterVendorInvoice`
- [ ] `DLV-FR-AP-002` — FR-AP-002 — `ApplyAssetClearingClassification`
- [ ] `DLV-FR-AP-003` — FR-AP-003 — `ApplyIncomingSettlement`
- [ ] `DLV-FR-AP-004` — FR-AP-004 — `ReverseIncomingSettlementApplication`
- [ ] `DLV-FR-AP-005` — FR-AP-005 — `ApplyPaymentReturn`
- [ ] `DLV-FR-AP-006` — FR-AP-006 — `ApplyVendorInvoiceApprovalDecision`
- [ ] `DLV-FR-AP-007` — FR-AP-007 — `RequestPayment`
- [ ] `DLV-FR-AP-008` — FR-AP-008 — `ValidateVendorInvoice`
- [ ] `DLV-FR-AP-009` — FR-AP-009 — `DisputeVendorInvoice`
- [ ] `DLV-FR-AP-010` — FR-AP-010 — `VoidVendorInvoice`
- [ ] `DLV-WF-7.1` — WF-7.1 — Vendor Invoice Registration, Matching, Approval, Dispute, and Void

## [ ] EP-PCM-001 — Payments and cash management (M5)

Payment batches, instructions, settlements, returns, exceptions, and expected incoming settlement.

- [ ] `DLV-FR-PCM-001` — FR-PCM-001 — `PreparePaymentBatch`
- [ ] `DLV-FR-PCM-002` — FR-PCM-002 — `ApplyPaymentBatchApprovalDecision`
- [ ] `DLV-FR-PCM-003` — FR-PCM-003 — `CancelPaymentBatch`
- [ ] `DLV-FR-PCM-004` — FR-PCM-004 — `RegisterExpectedIncomingSettlement`
- [ ] `DLV-FR-PCM-005` — FR-PCM-005 — `ResolveExpectedIncomingSettlementException`
- [ ] `DLV-FR-PCM-006` — FR-PCM-006 — `CancelExpectedIncomingSettlement`
- [ ] `DLV-FR-PCM-007` — FR-PCM-007 — `CloseExpectedIncomingSettlement`
- [ ] `DLV-FR-PCM-008` — FR-PCM-008 — `CreatePaymentInstructionFromObligation`
- [ ] `DLV-FR-PCM-009` — FR-PCM-009 — `SubmitPaymentInstruction`
- [ ] `DLV-FR-PCM-010` — FR-PCM-010 — `RetryPaymentInstruction`
- [ ] `DLV-FR-PCM-011` — FR-PCM-011 — `CancelPaymentInstruction`
- [ ] `DLV-FR-PCM-012` — FR-PCM-012 — `ApplyPaymentInstructionExceptionDecision`
- [ ] `DLV-FR-PCM-013` — FR-PCM-013 — `RecordPaymentReturn`
- [ ] `DLV-FR-PCM-014` — FR-PCM-014 — `CancelUnpostedPaymentReturn`
- [ ] `DLV-FR-PCM-015` — FR-PCM-015 — `AcknowledgePaymentReturn`
- [ ] `DLV-FR-PCM-016` — FR-PCM-016 — `ResolvePaymentReturnException`
- [ ] `DLV-FR-PCM-017` — FR-PCM-017 — `RecordUnallocatedIncomingSettlement`
- [ ] `DLV-FR-PCM-018` — FR-PCM-018 — `ResolveUnallocatedIncomingSettlement`
- [ ] `DLV-FR-PCM-019` — FR-PCM-019 — `RecordIncomingSettlement`
- [ ] `DLV-FR-PCM-020` — FR-PCM-020 — `ResolveSettlementReceiptValidationException`
- [ ] `DLV-FR-PCM-021` — FR-PCM-021 — `ResolveIncomingSettlementOwnerException`
- [ ] `DLV-FR-PCM-022` — FR-PCM-022 — `CancelUnpostedSettlementReceipt`
- [ ] `DLV-FR-PCM-023` — FR-PCM-023 — `AcknowledgeIncomingSettlement`
- [ ] `DLV-FR-PCM-024` — FR-PCM-024 — `ReverseIncomingSettlement`
- [ ] `DLV-FR-PCM-025` — FR-PCM-025 — `Maintain bank accounts`
- [ ] `DLV-WF-7.2` — WF-7.2 — Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation

## [ ] EP-BFR-001 — Bank feeds and reconciliation (M6)

Bank connections, statement imports, matching, unmatching, reconciliation, excess cash, and settlement corrections.

- [ ] `DLV-FR-BFR-001` — FR-BFR-001 — `ImportStatement`
- [ ] `DLV-FR-BFR-002` — FR-BFR-002 — `ProposeMatch`
- [ ] `DLV-FR-BFR-003` — FR-BFR-003 — `ConfirmMatch`
- [ ] `DLV-FR-BFR-004` — FR-BFR-004 — `Unmatch`
- [ ] `DLV-FR-BFR-005` — FR-BFR-005 — `CompleteReconciliation`
- [ ] `DLV-FR-BFR-006` — FR-BFR-006 — `Maintain bank-feed connections`
- [ ] `DLV-WF-7.4` — WF-7.4 — Bank Statement Import, Matching, Unmatching, and Reconciliation

## [ ] EP-FA-001 — Fixed Assets (M7)

Capitalization, depreciation, impairment, transfer, split, disposal, and disposal correction.

- [ ] `DLV-FR-FA-001` — FR-FA-001 — `CapitalizeAsset`
- [ ] `DLV-FR-FA-002` — FR-FA-002 — `CreateAssetAcquisitionClearing`
- [ ] `DLV-FR-FA-003` — FR-FA-003 — `RunDepreciation`
- [ ] `DLV-FR-FA-004` — FR-FA-004 — `ApplyImpairmentApprovalDecision`
- [ ] `DLV-FR-FA-005` — FR-FA-005 — `DisposeAsset`
- [ ] `DLV-FR-FA-006` — FR-FA-006 — `ApplyAssetDisposalApprovalDecision`
- [ ] `DLV-FR-FA-007` — FR-FA-007 — `CancelUnpostedAssetDisposal`
- [ ] `DLV-FR-FA-008` — FR-FA-008 — `CompensateFailedDisposalPosting`
- [ ] `DLV-FR-FA-009` — FR-FA-009 — `CreateDisposalSettlementClearing`
- [ ] `DLV-FR-FA-010` — FR-FA-010 — `ApplyAssetSupplierLiabilityResult`
- [ ] `DLV-FR-FA-011` — FR-FA-011 — `ApplyIncomingSettlement`
- [ ] `DLV-FR-FA-012` — FR-FA-012 — `ReverseIncomingSettlementApplication`
- [ ] `DLV-FR-FA-013` — FR-FA-013 — `ApplyPaymentReturn`
- [ ] `DLV-FR-FA-014` — FR-FA-014 — `ApplyAssetSettlementResult`
- [ ] `DLV-FR-FA-015` — FR-FA-015 — `ReclassifyDisposalCostForPayment`
- [ ] `DLV-FR-FA-016` — FR-FA-016 — `RequestDisposalCostPayment`
- [ ] `DLV-FR-FA-017` — FR-FA-017 — `RequestDisposalCostPaymentReplacement`
- [ ] `DLV-FR-FA-018` — FR-FA-018 — `Record impairment assessments`
- [ ] `DLV-FR-FA-019` — FR-FA-019 — `Transfer assets or components`
- [ ] `DLV-FR-FA-020` — FR-FA-020 — `Split assets or components`
- [ ] `DLV-FR-FA-021` — FR-FA-021 — `Correct posted asset disposals`
- [ ] `DLV-WF-6.4` — WF-6.4 — Fixed Asset Disposal with Gain or Loss Recognition
- [ ] `DLV-WF-7.7` — WF-7.7 — Full Fixed-Asset Lifecycle and Disposal Variants

## [ ] EP-REV-001 — Revenue Recognition (M7)

Revenue contracts, performance obligations, recognition profiles, schedules, and contract modifications.

- [ ] `DLV-FR-REV-001` — FR-REV-001 — `AssessContract`
- [ ] `DLV-FR-REV-002` — FR-REV-002 — `ApplyRevenueScheduleApprovalDecision`
- [ ] `DLV-FR-REV-003` — FR-REV-003 — `PublishRevenueAccountingProfile`
- [ ] `DLV-FR-REV-004` — FR-REV-004 — `ModifyContract`
- [ ] `DLV-FR-REV-005` — FR-REV-005 — `ApplyContractModificationApprovalDecision`
- [ ] `DLV-FR-REV-006` — FR-REV-006 — `RunRecognition`
- [ ] `DLV-WF-6.5` — WF-6.5 — Revenue Recognition for a SaaS Contract
- [ ] `DLV-WF-7.8` — WF-7.8 — Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration

## [ ] EP-FX-001 — Multi-Currency (M8)

Exchange rates, realized foreign exchange, revaluation, and translation.

- [ ] `DLV-GFR-010` — GFR-010 — EP-FX-001
- [ ] `DLV-FR-FX-001` — FR-FX-001 — `PublishRateSet`
- [ ] `DLV-FR-FX-002` — FR-FX-002 — `RunRevaluation`
- [ ] `DLV-FR-FX-003` — FR-FX-003 — `ApplyRevaluationApprovalDecision`
- [ ] `DLV-FR-FX-004` — FR-FX-004 — `PostRevaluationRun`
- [ ] `DLV-FR-FX-005` — FR-FX-005 — `RunTranslation`
- [ ] `DLV-WF-7.5` — WF-7.5 — Foreign-Currency Invoice Settlement and Realized FX
- [ ] `DLV-WF-7.6` — WF-7.6 — Period-End Revaluation, Rerun, and Next-Period Reversal

## [ ] EP-IC-001 — Intercompany (M8)

Intercompany agreements, transactions, matching, settlement, incoming settlement, returns, and eliminations.

- [ ] `DLV-FR-IC-001` — FR-IC-001 — `StartSettlement`
- [ ] `DLV-FR-IC-002` — FR-IC-002 — `MatchIntercompanyItems`
- [ ] `DLV-FR-IC-003` — FR-IC-003 — `ApplyResidualApprovalDecision`
- [ ] `DLV-FR-IC-004` — FR-IC-004 — `CreateSettlementInstructions`
- [ ] `DLV-FR-IC-005` — FR-IC-005 — `CompleteSettlementRun`
- [ ] `DLV-FR-IC-006` — FR-IC-006 — `ApplyIncomingSettlement`
- [ ] `DLV-FR-IC-007` — FR-IC-007 — `ReverseIncomingSettlementApplication`
- [ ] `DLV-FR-IC-008` — FR-IC-008 — `ApplyPaymentReturn`
- [ ] `DLV-FR-IC-009` — FR-IC-009 — `RunElimination`
- [ ] `DLV-FR-IC-010` — FR-IC-010 — `Maintain intercompany agreements`
- [ ] `DLV-FR-IC-011` — FR-IC-011 — `Record intercompany transactions`
- [ ] `DLV-WF-6.3` — WF-6.3 — Intercompany Reconciliation and Settlement

## [ ] EP-RPT-001 — Financial Reporting (M8)

Report definitions, statements, consolidation, lineage, and publication.

- [ ] `DLV-FR-RPT-001` — FR-RPT-001 — `RunConsolidation`
- [ ] `DLV-FR-RPT-002` — FR-RPT-002 — `ApplyTranslationResult`
- [ ] `DLV-FR-RPT-003` — FR-RPT-003 — `ApplyConsolidationApprovalDecision`
- [ ] `DLV-FR-RPT-004` — FR-RPT-004 — `PublishConsolidatedStatement`
- [ ] `DLV-FR-RPT-005` — FR-RPT-005 — `Maintain report definitions`
- [ ] `DLV-FR-RPT-006` — FR-RPT-006 — `Generate and publish ledger financial statements`
- [ ] `DLV-WF-7.9` — WF-7.9 — Consolidation, Ownership Changes, Translation, Eliminations, and Rerun

## [ ] EP-TAX-001 — Tax Filing (M9)

Tax configurations, returns, submissions, amendments, adjustments, and payments.

- [ ] `DLV-FR-TAX-001` — FR-TAX-001 — `DetermineTax`
- [ ] `DLV-FR-TAX-002` — FR-TAX-002 — `PrepareTaxReturn`
- [ ] `DLV-FR-TAX-003` — FR-TAX-003 — `ApplyTaxReturnApprovalDecision`
- [ ] `DLV-FR-TAX-004` — FR-TAX-004 — `SubmitTaxReturn`
- [ ] `DLV-FR-TAX-005` — FR-TAX-005 — `CreateTaxAmendment`
- [ ] `DLV-FR-TAX-006` — FR-TAX-006 — `ApplyTaxAmendmentApprovalDecision`
- [ ] `DLV-FR-TAX-007` — FR-TAX-007 — `SubmitTaxAmendment`
- [ ] `DLV-FR-TAX-008` — FR-TAX-008 — `CreateReturnLevelTaxAdjustment`
- [ ] `DLV-FR-TAX-009` — FR-TAX-009 — `ApplyReturnLevelTaxAdjustmentApprovalDecision`
- [ ] `DLV-FR-TAX-010` — FR-TAX-010 — `PostReturnLevelTaxAdjustment`
- [ ] `DLV-FR-TAX-011` — FR-TAX-011 — `RequestTaxPayment`
- [ ] `DLV-FR-TAX-012` — FR-TAX-012 — `RecordTaxPaymentSettlement`
- [ ] `DLV-FR-TAX-013` — FR-TAX-013 — `ApplyIncomingSettlement`
- [ ] `DLV-FR-TAX-014` — FR-TAX-014 — `ReverseIncomingSettlementApplication`
- [ ] `DLV-FR-TAX-015` — FR-TAX-015 — `ApplyPaymentReturn`
- [ ] `DLV-FR-TAX-016` — FR-TAX-016 — `Maintain tax configurations`
- [ ] `DLV-WF-7.10` — WF-7.10 — Tax Return Submission, Rejection, Amendment, Payment, and Evidence

## [ ] EP-PAYR-001 — Payroll (M9)

Payroll profiles, runs, corrections, off-cycle processing, failed payments, and filing amendments.

- [ ] `DLV-FR-PAYR-001` — FR-PAYR-001 — `CalculatePayrollRun`
- [ ] `DLV-FR-PAYR-002` — FR-PAYR-002 — `ApplyPayrollRunApprovalDecision`
- [ ] `DLV-FR-PAYR-003` — FR-PAYR-003 — `PostPayrollRun`
- [ ] `DLV-FR-PAYR-004` — FR-PAYR-004 — `CreatePayrollCorrection`
- [ ] `DLV-FR-PAYR-005` — FR-PAYR-005 — `ApplyPaymentReturn`
- [ ] `DLV-FR-PAYR-006` — FR-PAYR-006 — `Maintain employee payroll profiles`
- [ ] `DLV-FR-PAYR-007` — FR-PAYR-007 — `Maintain payroll tax-filing records`
- [ ] `DLV-WF-7.11` — WF-7.11 — Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment

## [ ] EP-AUD-001 — Audit Integrity (M9)

Evidence ingestion, verification, credential rotation, incidents, legal holds, and controlled proof access.

- [ ] `DLV-GFR-014` — GFR-014 — EP-AUD-001
- [ ] `DLV-GFR-018` — GFR-018 — EP-AUD-001
- [ ] `DLV-GFR-020` — GFR-020 — EP-AUD-001
- [ ] `DLV-FR-AUD-001` — FR-AUD-001 — `AppendAuditableEvent`
- [ ] `DLV-FR-AUD-002` — FR-AUD-002 — `CreateAuditSeal`
- [ ] `DLV-FR-AUD-003` — FR-AUD-003 — `RotateVerificationCredential`
- [ ] `DLV-FR-AUD-004` — FR-AUD-004 — `EscalateIntegrityIncident`
- [ ] `DLV-FR-AUD-005` — FR-AUD-005 — `VerifyProof`
- [ ] `DLV-WF-7.15` — WF-7.15 — Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation

## [ ] EP-QUAL-001 — Full-system qualification (M9)

Security, privacy, accessibility, capacity, performance, recovery, and release evidence.

- [ ] `DLV-NFR-ACC-001` — NFR-ACC-001 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-002` — NFR-ACC-002 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-003` — NFR-ACC-003 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-004` — NFR-ACC-004 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-005` — NFR-ACC-005 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-006` — NFR-ACC-006 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-007` — NFR-ACC-007 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-008` — NFR-ACC-008 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-009` — NFR-ACC-009 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-010` — NFR-ACC-010 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-011` — NFR-ACC-011 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-ACC-012` — NFR-ACC-012 — Accessibility and Inclusive Use (QG-05, M0)
- [ ] `DLV-NFR-AUD-001` — NFR-AUD-001 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-002` — NFR-AUD-002 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-003` — NFR-AUD-003 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-004` — NFR-AUD-004 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-005` — NFR-AUD-005 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-006` — NFR-AUD-006 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-007` — NFR-AUD-007 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-008` — NFR-AUD-008 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-009` — NFR-AUD-009 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-010` — NFR-AUD-010 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-011` — NFR-AUD-011 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AUD-012` — NFR-AUD-012 — Auditability, Evidence, and Nonrepudiation (QG-08, M9)
- [ ] `DLV-NFR-AVL-001` — NFR-AVL-001 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-002` — NFR-AVL-002 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-003` — NFR-AVL-003 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-004` — NFR-AVL-004 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-005` — NFR-AVL-005 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-006` — NFR-AVL-006 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-007` — NFR-AVL-007 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-008` — NFR-AVL-008 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-009` — NFR-AVL-009 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-AVL-010` — NFR-AVL-010 — Availability and Service Continuity (QG-09, M9)
- [ ] `DLV-NFR-CAP-001` — NFR-CAP-001 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-002` — NFR-CAP-002 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-003` — NFR-CAP-003 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-004` — NFR-CAP-004 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-005` — NFR-CAP-005 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-006` — NFR-CAP-006 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-007` — NFR-CAP-007 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-008` — NFR-CAP-008 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-009` — NFR-CAP-009 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CAP-010` — NFR-CAP-010 — Capacity and Scalability (QG-09, M9)
- [ ] `DLV-NFR-CMP-001` — NFR-CMP-001 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-002` — NFR-CMP-002 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-003` — NFR-CMP-003 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-004` — NFR-CMP-004 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-005` — NFR-CMP-005 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-006` — NFR-CMP-006 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-007` — NFR-CMP-007 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-CMP-008` — NFR-CMP-008 — Compatibility and Client Quality (QG-05, M0)
- [ ] `DLV-NFR-INT-001` — NFR-INT-001 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-002` — NFR-INT-002 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-003` — NFR-INT-003 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-004` — NFR-INT-004 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-005` — NFR-INT-005 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-006` — NFR-INT-006 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-007` — NFR-INT-007 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-008` — NFR-INT-008 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-009` — NFR-INT-009 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-INT-010` — NFR-INT-010 — Interoperability and External Dependency Quality (QG-04, M2)
- [ ] `DLV-NFR-LOC-001` — NFR-LOC-001 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-002` — NFR-LOC-002 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-003` — NFR-LOC-003 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-004` — NFR-LOC-004 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-005` — NFR-LOC-005 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-006` — NFR-LOC-006 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-007` — NFR-LOC-007 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-LOC-008` — NFR-LOC-008 — Localization, Currency, Date, and Language Quality (QG-05, M0)
- [ ] `DLV-NFR-MNT-001` — NFR-MNT-001 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-002` — NFR-MNT-002 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-003` — NFR-MNT-003 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-004` — NFR-MNT-004 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-005` — NFR-MNT-005 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-006` — NFR-MNT-006 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-007` — NFR-MNT-007 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-008` — NFR-MNT-008 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-009` — NFR-MNT-009 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-010` — NFR-MNT-010 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-011` — NFR-MNT-011 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-MNT-012` — NFR-MNT-012 — Maintainability, Change Safety, and Operability (QG-01, M0)
- [ ] `DLV-NFR-OBS-001` — NFR-OBS-001 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-002` — NFR-OBS-002 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-003` — NFR-OBS-003 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-004` — NFR-OBS-004 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-005` — NFR-OBS-005 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-006` — NFR-OBS-006 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-007` — NFR-OBS-007 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-008` — NFR-OBS-008 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-009` — NFR-OBS-009 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-010` — NFR-OBS-010 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-011` — NFR-OBS-011 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-OBS-012` — NFR-OBS-012 — Observability, Operations, and Supportability (QG-08, M0)
- [ ] `DLV-NFR-PERF-001` — NFR-PERF-001 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-002` — NFR-PERF-002 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-003` — NFR-PERF-003 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-004` — NFR-PERF-004 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-005` — NFR-PERF-005 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-006` — NFR-PERF-006 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-007` — NFR-PERF-007 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-008` — NFR-PERF-008 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-009` — NFR-PERF-009 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-010` — NFR-PERF-010 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-011` — NFR-PERF-011 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-012` — NFR-PERF-012 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-013` — NFR-PERF-013 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PERF-014` — NFR-PERF-014 — Performance and Responsiveness (QG-09, M9)
- [ ] `DLV-NFR-PRV-001` — NFR-PRV-001 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-002` — NFR-PRV-002 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-003` — NFR-PRV-003 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-004` — NFR-PRV-004 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-005` — NFR-PRV-005 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-006` — NFR-PRV-006 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-007` — NFR-PRV-007 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-008` — NFR-PRV-008 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-009` — NFR-PRV-009 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-PRV-010` — NFR-PRV-010 — Privacy, Retention, and Legal Hold (QG-06, M1)
- [ ] `DLV-NFR-REC-001` — NFR-REC-001 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-002` — NFR-REC-002 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-003` — NFR-REC-003 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-004` — NFR-REC-004 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-005` — NFR-REC-005 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-006` — NFR-REC-006 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-007` — NFR-REC-007 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-008` — NFR-REC-008 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-009` — NFR-REC-009 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-010` — NFR-REC-010 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-011` — NFR-REC-011 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REC-012` — NFR-REC-012 — Resilience, Backup, and Disaster Recovery (QG-09, M9)
- [ ] `DLV-NFR-REL-001` — NFR-REL-001 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-002` — NFR-REL-002 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-003` — NFR-REL-003 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-004` — NFR-REL-004 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-005` — NFR-REL-005 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-006` — NFR-REL-006 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-007` — NFR-REL-007 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-008` — NFR-REL-008 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-009` — NFR-REL-009 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-010` — NFR-REL-010 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-011` — NFR-REL-011 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-012` — NFR-REL-012 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-013` — NFR-REL-013 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-014` — NFR-REL-014 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-015` — NFR-REL-015 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-REL-016` — NFR-REL-016 — Reliability, Data Integrity, and Consistency (QG-07, M2)
- [ ] `DLV-NFR-SEC-001` — NFR-SEC-001 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-002` — NFR-SEC-002 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-003` — NFR-SEC-003 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-004` — NFR-SEC-004 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-005` — NFR-SEC-005 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-006` — NFR-SEC-006 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-007` — NFR-SEC-007 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-008` — NFR-SEC-008 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-009` — NFR-SEC-009 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-010` — NFR-SEC-010 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-011` — NFR-SEC-011 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-012` — NFR-SEC-012 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-013` — NFR-SEC-013 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-014` — NFR-SEC-014 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-015` — NFR-SEC-015 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-016` — NFR-SEC-016 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-017` — NFR-SEC-017 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-SEC-018` — NFR-SEC-018 — Security, Identity, and Access Control (QG-06, M1)
- [ ] `DLV-NFR-TST-001` — NFR-TST-001 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-002` — NFR-TST-002 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-003` — NFR-TST-003 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-004` — NFR-TST-004 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-005` — NFR-TST-005 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-006` — NFR-TST-006 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-007` — NFR-TST-007 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-008` — NFR-TST-008 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-009` — NFR-TST-009 — Verification, Testing, and Release Quality (QG-10, M9)
- [ ] `DLV-NFR-TST-010` — NFR-TST-010 — Verification, Testing, and Release Quality (QG-10, M9)

## Completion rule

An epic is checked only when its required delivery items and applicable quality gates pass, or when the epic has been explicitly closed with deferred scope recorded. Partial implementation does not count as complete.
