# Finance Platform API and OpenAPI Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready API baseline |
| Contract format | OpenAPI 3.1, REST/JSON |

## 1. API principles

- Base URL is `/api/v1`.
- Command endpoints are idempotent business operations, not CRUD substitutes for established financial facts.
- Query endpoints never mutate state.
- The API is authoritative for authorization; the SPA's visible actions are advisory.
- Breaking changes require `/api/v2`; additive optional fields and new enum values follow compatibility review.

## 2. Required headers

| Header | Required | Purpose |
|---|---|---|
| `Authorization: Bearer <token>` | Yes outside local | Entra access token |
| `X-Correlation-Id` | Optional | Client correlation UUID; server creates when absent |
| `Idempotency-Key` | All state-changing financial operations | Stable business retry identity |
| `If-Match` | Existing mutable aggregate commands | Quoted aggregate version, for example `"7"` |
| `X-Accounting-Scope-Id` | Ledger-bound operations | Explicit accounting scope |
| `Accept-Language` | Optional | User-facing message localization |

## 3. Common command request

```json
{
  "commandId": "0195a91b-20ab-7c15-8aa8-4e111a8bd618",
  "expectedVersion": 7,
  "accountingScopeId": "0195a91b-20ab-7c15-8aa8-4e111a8bd619",
  "businessDate": "2026-07-24",
  "data": {}
}
```

The server computes the canonical fingerprint from semantic request fields. Actor, authentication subject and authorization evidence are server-derived.

## 4. Common established result

```json
{
  "status": "established",
  "aggregateId": "0195a91b-20ab-7c15-8aa8-4e111a8bd620",
  "aggregateVersion": 8,
  "processId": null,
  "correlationId": "0195a91b-20ab-7c15-8aa8-4e111a8bd621",
  "links": {"self": "/api/v1/..."},
  "data": {}
}
```

## 5. Error schema

```json
{
  "type": "https://finance.example/errors/version-conflict",
  "title": "Version conflict",
  "status": 409,
  "code": "VERSION_CONFLICT",
  "detail": "The record changed after it was reviewed.",
  "correlationId": "0195a91b-20ab-7c15-8aa8-4e111a8bd621",
  "currentVersion": 9,
  "fieldErrors": []
}
```

The contract follows RFC 9457 problem details with stable application `code` values.

## 6. Query surface

Every capability exposes these exact read patterns when relevant:

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/<capability>/work-items` | Filtered, cursor-paged worklist |
| GET | `/api/v1/<capability>/records/{recordType}/{recordId}` | Authoritative record detail/projection |
| GET | `/api/v1/<capability>/records/{recordType}/{recordId}/timeline` | Lifecycle and correction lineage |
| GET | `/api/v1/<capability>/results/{businessIdentity}` | Established-result lookup after duplicate or ambiguous submission |
| GET | `/api/v1/<capability>/evidence/{evidenceId}` | Authorized evidence metadata and expiring retrieval link |

Cursor pagination uses `page[size]` (default 50, maximum 200) and `page[after]`. Sorting uses a server allow-list. Unbounded list responses are prohibited.

## 7. Exact operation endpoint catalog

| Requirement | Method | Path | OpenAPI operationId | Permission | Provenance |
|---|---|---|---|---|---|
| FR-OMD-001 | PUT | `/api/v1/master-data/configuration/maintain-legal-entities` | `omdMaintainLegalEntities` | finance.omd.maintain.legal.entities | PRD functional action |
| FR-OMD-002 | PUT | `/api/v1/master-data/configuration/maintain-parties` | `omdMaintainParties` | finance.omd.maintain.parties | PRD functional action |
| FR-OMD-003 | PUT | `/api/v1/master-data/configuration/maintain-customer-profiles` | `omdMaintainCustomerProfiles` | finance.omd.maintain.customer.profiles | PRD functional action |
| FR-OMD-004 | PUT | `/api/v1/master-data/configuration/maintain-vendor-profiles` | `omdMaintainVendorProfiles` | finance.omd.maintain.vendor.profiles | PRD functional action |
| FR-OMD-005 | PUT | `/api/v1/master-data/configuration/maintain-fiscal-calendars` | `omdMaintainFiscalCalendars` | finance.omd.maintain.fiscal.calendars | PRD functional action |
| FR-OMD-006 | POST | `/api/v1/master-data/actions/publish-approved-master-data-changes` | `omdPublishApprovedMasterDataChanges` | finance.omd.publish.approved.master.data.changes | PRD functional action |
| FR-GL-001 | POST | `/api/v1/general-ledger/actions/submit-posting-request` | `glSubmitPostingRequest` | finance.gl.submit.posting.request | DDD representative command |
| FR-GL-002 | POST | `/api/v1/general-ledger/actions/apply-journal-approval-decision` | `glApplyJournalApprovalDecision` | finance.gl.apply.journal.approval.decision | DDD representative command |
| FR-GL-003 | POST | `/api/v1/general-ledger/actions/reverse-journal-entry` | `glReverseJournalEntry` | finance.gl.reverse.journal.entry | DDD representative command |
| FR-GL-004 | POST | `/api/v1/general-ledger/actions/enter-soft-close-gate` | `glEnterSoftCloseGate` | finance.gl.enter.soft.close.gate | DDD representative command |
| FR-GL-005 | POST | `/api/v1/general-ledger/actions/exit-soft-close-gate` | `glExitSoftCloseGate` | finance.gl.exit.soft.close.gate | DDD representative command |
| FR-GL-006 | POST | `/api/v1/general-ledger/actions/acquire-posting-barrier` | `glAcquirePostingBarrier` | finance.gl.acquire.posting.barrier | DDD representative command |
| FR-GL-007 | POST | `/api/v1/general-ledger/actions/release-posting-barrier` | `glReleasePostingBarrier` | finance.gl.release.posting.barrier | DDD representative command |
| FR-GL-008 | POST | `/api/v1/general-ledger/actions/finalize-posting-gate` | `glFinalizePostingGate` | finance.gl.finalize.posting.gate | DDD representative command |
| FR-GL-009 | POST | `/api/v1/general-ledger/actions/open-scoped-reopen-gate` | `glOpenScopedReopenGate` | finance.gl.open.scoped.reopen.gate | DDD representative command |
| FR-GL-010 | POST | `/api/v1/general-ledger/actions/close-scoped-reopen-gate` | `glCloseScopedReopenGate` | finance.gl.close.scoped.reopen.gate | DDD representative command |
| FR-GL-011 | POST | `/api/v1/general-ledger/actions/open-operational-reopen-gate` | `glOpenOperationalReopenGate` | finance.gl.open.operational.reopen.gate | DDD representative command |
| FR-GL-012 | POST | `/api/v1/general-ledger/actions/close-operational-reopen-gate` | `glCloseOperationalReopenGate` | finance.gl.close.operational.reopen.gate | DDD representative command |
| FR-GL-013 | POST | `/api/v1/general-ledger/actions/begin-reclose-gate` | `glBeginRecloseGate` | finance.gl.begin.reclose.gate | DDD representative command |
| FR-GL-014 | GET | `/api/v1/general-ledger/reference/get-posting-gate-status` | `glGetPostingGateStatus` | finance.gl.get.posting.gate.status | DDD reference operation |
| FR-GL-015 | PUT | `/api/v1/general-ledger/configuration/maintain-ledgers` | `glMaintainLedgers` | finance.gl.maintain.ledgers | PRD functional action |
| FR-GL-016 | PUT | `/api/v1/general-ledger/configuration/maintain-accounting-books` | `glMaintainAccountingBooks` | finance.gl.maintain.accounting.books | PRD functional action |
| FR-GL-017 | PUT | `/api/v1/general-ledger/configuration/maintain-charts-of-accounts` | `glMaintainChartsOfAccounts` | finance.gl.maintain.charts.of.accounts | PRD functional action |
| FR-GL-018 | PUT | `/api/v1/general-ledger/configuration/maintain-accounts-and-reporting-mappings` | `glMaintainAccountsAndReportingMappings` | finance.gl.maintain.accounts.and.reporting.mappings | PRD functional action |
| FR-AP-001 | POST | `/api/v1/accounts-payable/actions/register-vendor-invoice` | `apRegisterVendorInvoice` | finance.ap.register.vendor.invoice | DDD representative command |
| FR-AP-002 | POST | `/api/v1/accounts-payable/actions/apply-asset-clearing-classification` | `apApplyAssetClearingClassification` | finance.ap.apply.asset.clearing.classification | DDD representative command |
| FR-AP-003 | POST | `/api/v1/accounts-payable/actions/apply-incoming-settlement` | `apApplyIncomingSettlement` | finance.ap.apply.incoming.settlement | DDD representative command |
| FR-AP-004 | POST | `/api/v1/accounts-payable/actions/reverse-incoming-settlement-application` | `apReverseIncomingSettlementApplication` | finance.ap.reverse.incoming.settlement.application | DDD representative command |
| FR-AP-005 | POST | `/api/v1/accounts-payable/actions/apply-payment-return` | `apApplyPaymentReturn` | finance.ap.apply.payment.return | DDD representative command |
| FR-AP-006 | POST | `/api/v1/accounts-payable/actions/apply-vendor-invoice-approval-decision` | `apApplyVendorInvoiceApprovalDecision` | finance.ap.apply.vendor.invoice.approval.decision | DDD representative command |
| FR-AP-007 | POST | `/api/v1/accounts-payable/actions/request-payment` | `apRequestPayment` | finance.ap.request.payment | DDD representative command |
| FR-AP-008 | POST | `/api/v1/accounts-payable/actions/validate-vendor-invoice` | `apValidateVendorInvoice` | finance.ap.validate.vendor.invoice | DDD detailed command |
| FR-AP-009 | POST | `/api/v1/accounts-payable/actions/dispute-vendor-invoice` | `apDisputeVendorInvoice` | finance.ap.dispute.vendor.invoice | DDD detailed command |
| FR-AP-010 | POST | `/api/v1/accounts-payable/actions/void-vendor-invoice` | `apVoidVendorInvoice` | finance.ap.void.vendor.invoice | DDD detailed command |
| FR-AR-001 | POST | `/api/v1/accounts-receivable/actions/issue-customer-invoice` | `arIssueCustomerInvoice` | finance.ar.issue.customer.invoice | DDD representative command |
| FR-AR-002 | POST | `/api/v1/accounts-receivable/actions/record-receipt` | `arRecordReceipt` | finance.ar.record.receipt | DDD representative command |
| FR-AR-003 | POST | `/api/v1/accounts-receivable/actions/apply-receipt` | `arApplyReceipt` | finance.ar.apply.receipt | DDD representative command |
| FR-AR-004 | POST | `/api/v1/accounts-receivable/actions/unapply-receipt` | `arUnapplyReceipt` | finance.ar.unapply.receipt | DDD representative command |
| FR-AR-005 | POST | `/api/v1/accounts-receivable/actions/rollback-unposted-application-batch` | `arRollbackUnpostedApplicationBatch` | finance.ar.rollback.unposted.application.batch | DDD representative command |
| FR-AR-006 | POST | `/api/v1/accounts-receivable/actions/issue-credit-note` | `arIssueCreditNote` | finance.ar.issue.credit.note | DDD representative command |
| FR-AR-007 | POST | `/api/v1/accounts-receivable/actions/create-customer-refund-request` | `arCreateCustomerRefundRequest` | finance.ar.create.customer.refund.request | DDD representative command |
| FR-AR-008 | POST | `/api/v1/accounts-receivable/actions/cancel-customer-refund-request` | `arCancelCustomerRefundRequest` | finance.ar.cancel.customer.refund.request | DDD representative command |
| FR-AR-009 | POST | `/api/v1/accounts-receivable/actions/apply-customer-refund-approval-decision` | `arApplyCustomerRefundApprovalDecision` | finance.ar.apply.customer.refund.approval.decision | DDD representative command |
| FR-AR-010 | POST | `/api/v1/accounts-receivable/actions/request-customer-refund-payment` | `arRequestCustomerRefundPayment` | finance.ar.request.customer.refund.payment | DDD representative command |
| FR-AR-011 | POST | `/api/v1/accounts-receivable/actions/cancel-customer-refund-payment` | `arCancelCustomerRefundPayment` | finance.ar.cancel.customer.refund.payment | DDD representative command |
| FR-AR-012 | POST | `/api/v1/accounts-receivable/actions/apply-customer-refund-payment-result` | `arApplyCustomerRefundPaymentResult` | finance.ar.apply.customer.refund.payment.result | DDD representative command |
| FR-AR-013 | POST | `/api/v1/accounts-receivable/actions/apply-payment-return` | `arApplyPaymentReturn` | finance.ar.apply.payment.return | DDD representative command |
| FR-AR-014 | POST | `/api/v1/accounts-receivable/actions/resolve-customer-overpayments` | `arResolveCustomerOverpayments` | finance.ar.resolve.customer.overpayments | PRD functional action |
| FR-AR-015 | POST | `/api/v1/accounts-receivable/actions/record-customer-chargebacks` | `arRecordCustomerChargebacks` | finance.ar.record.customer.chargebacks | PRD functional action |
| FR-AR-016 | POST | `/api/v1/accounts-receivable/actions/record-receivable-write-offs` | `arRecordReceivableWriteOffs` | finance.ar.record.receivable.write.offs | PRD functional action |
| FR-PAYR-001 | POST | `/api/v1/payroll/actions/calculate-payroll-run` | `payrCalculatePayrollRun` | finance.payr.calculate.payroll.run | DDD representative command |
| FR-PAYR-002 | POST | `/api/v1/payroll/actions/apply-payroll-run-approval-decision` | `payrApplyPayrollRunApprovalDecision` | finance.payr.apply.payroll.run.approval.decision | DDD representative command |
| FR-PAYR-003 | POST | `/api/v1/payroll/actions/post-payroll-run` | `payrPostPayrollRun` | finance.payr.post.payroll.run | DDD representative command |
| FR-PAYR-004 | POST | `/api/v1/payroll/actions/create-payroll-correction` | `payrCreatePayrollCorrection` | finance.payr.create.payroll.correction | DDD representative command |
| FR-PAYR-005 | POST | `/api/v1/payroll/actions/apply-payment-return` | `payrApplyPaymentReturn` | finance.payr.apply.payment.return | DDD representative command |
| FR-PAYR-006 | PUT | `/api/v1/payroll/configuration/maintain-employee-payroll-profiles` | `payrMaintainEmployeePayrollProfiles` | finance.payr.maintain.employee.payroll.profiles | PRD functional action |
| FR-PAYR-007 | PUT | `/api/v1/payroll/configuration/maintain-payroll-tax-filing-records` | `payrMaintainPayrollTaxFilingRecords` | finance.payr.maintain.payroll.tax.filing.records | PRD functional action |
| FR-INV-001 | PUT | `/api/v1/invoicing/configuration/configure-invoice-templates` | `invConfigureInvoiceTemplates` | finance.inv.configure.invoice.templates | PRD functional action |
| FR-INV-002 | PUT | `/api/v1/invoicing/configuration/configure-billing-schedules` | `invConfigureBillingSchedules` | finance.inv.configure.billing.schedules | PRD functional action |
| FR-INV-003 | POST | `/api/v1/invoicing/actions/generate-invoices` | `invGenerateInvoices` | finance.inv.generate.invoices | PRD functional action |
| FR-INV-004 | POST | `/api/v1/invoicing/actions/finalize-generated-invoices` | `invFinalizeGeneratedInvoices` | finance.inv.finalize.generated.invoices | PRD functional action |
| FR-INV-005 | POST | `/api/v1/invoicing/actions/recalculate-unfinalized-invoices` | `invRecalculateUnfinalizedInvoices` | finance.inv.recalculate.unfinalized.invoices | PRD functional action |
| FR-INV-006 | POST | `/api/v1/invoicing/actions/cancel-unfinalized-invoices` | `invCancelUnfinalizedInvoices` | finance.inv.cancel.unfinalized.invoices | PRD functional action |
| FR-PCM-001 | POST | `/api/v1/payments/actions/prepare-payment-batch` | `pcmPreparePaymentBatch` | finance.pcm.prepare.payment.batch | DDD representative command |
| FR-PCM-002 | POST | `/api/v1/payments/actions/apply-payment-batch-approval-decision` | `pcmApplyPaymentBatchApprovalDecision` | finance.pcm.apply.payment.batch.approval.decision | DDD representative command |
| FR-PCM-003 | POST | `/api/v1/payments/actions/cancel-payment-batch` | `pcmCancelPaymentBatch` | finance.pcm.cancel.payment.batch | DDD representative command |
| FR-PCM-004 | POST | `/api/v1/payments/actions/register-expected-incoming-settlement` | `pcmRegisterExpectedIncomingSettlement` | finance.pcm.register.expected.incoming.settlement | DDD representative command |
| FR-PCM-005 | POST | `/api/v1/payments/actions/resolve-expected-incoming-settlement-exception` | `pcmResolveExpectedIncomingSettlementException` | finance.pcm.resolve.expected.incoming.settlement.exception | DDD representative command |
| FR-PCM-006 | POST | `/api/v1/payments/actions/cancel-expected-incoming-settlement` | `pcmCancelExpectedIncomingSettlement` | finance.pcm.cancel.expected.incoming.settlement | DDD representative command |
| FR-PCM-007 | POST | `/api/v1/payments/actions/close-expected-incoming-settlement` | `pcmCloseExpectedIncomingSettlement` | finance.pcm.close.expected.incoming.settlement | DDD representative command |
| FR-PCM-008 | POST | `/api/v1/payments/actions/create-payment-instruction-from-obligation` | `pcmCreatePaymentInstructionFromObligation` | finance.pcm.create.payment.instruction.from.obligation | DDD representative command |
| FR-PCM-009 | POST | `/api/v1/payments/actions/submit-payment-instruction` | `pcmSubmitPaymentInstruction` | finance.pcm.submit.payment.instruction | DDD representative command |
| FR-PCM-010 | POST | `/api/v1/payments/actions/retry-payment-instruction` | `pcmRetryPaymentInstruction` | finance.pcm.retry.payment.instruction | DDD representative command |
| FR-PCM-011 | POST | `/api/v1/payments/actions/cancel-payment-instruction` | `pcmCancelPaymentInstruction` | finance.pcm.cancel.payment.instruction | DDD representative command |
| FR-PCM-012 | POST | `/api/v1/payments/actions/apply-payment-instruction-exception-decision` | `pcmApplyPaymentInstructionExceptionDecision` | finance.pcm.apply.payment.instruction.exception.decision | DDD representative command |
| FR-PCM-013 | POST | `/api/v1/payments/actions/record-payment-return` | `pcmRecordPaymentReturn` | finance.pcm.record.payment.return | DDD representative command |
| FR-PCM-014 | POST | `/api/v1/payments/actions/cancel-unposted-payment-return` | `pcmCancelUnpostedPaymentReturn` | finance.pcm.cancel.unposted.payment.return | DDD representative command |
| FR-PCM-015 | POST | `/api/v1/payments/actions/acknowledge-payment-return` | `pcmAcknowledgePaymentReturn` | finance.pcm.acknowledge.payment.return | DDD representative command |
| FR-PCM-016 | POST | `/api/v1/payments/actions/resolve-payment-return-exception` | `pcmResolvePaymentReturnException` | finance.pcm.resolve.payment.return.exception | DDD representative command |
| FR-PCM-017 | POST | `/api/v1/payments/actions/record-unallocated-incoming-settlement` | `pcmRecordUnallocatedIncomingSettlement` | finance.pcm.record.unallocated.incoming.settlement | DDD representative command |
| FR-PCM-018 | POST | `/api/v1/payments/actions/resolve-unallocated-incoming-settlement` | `pcmResolveUnallocatedIncomingSettlement` | finance.pcm.resolve.unallocated.incoming.settlement | DDD representative command |
| FR-PCM-019 | POST | `/api/v1/payments/actions/record-incoming-settlement` | `pcmRecordIncomingSettlement` | finance.pcm.record.incoming.settlement | DDD representative command |
| FR-PCM-020 | POST | `/api/v1/payments/actions/resolve-settlement-receipt-validation-exception` | `pcmResolveSettlementReceiptValidationException` | finance.pcm.resolve.settlement.receipt.validation.exception | DDD representative command |
| FR-PCM-021 | POST | `/api/v1/payments/actions/resolve-incoming-settlement-owner-exception` | `pcmResolveIncomingSettlementOwnerException` | finance.pcm.resolve.incoming.settlement.owner.exception | DDD representative command |
| FR-PCM-022 | POST | `/api/v1/payments/actions/cancel-unposted-settlement-receipt` | `pcmCancelUnpostedSettlementReceipt` | finance.pcm.cancel.unposted.settlement.receipt | DDD representative command |
| FR-PCM-023 | POST | `/api/v1/payments/actions/acknowledge-incoming-settlement` | `pcmAcknowledgeIncomingSettlement` | finance.pcm.acknowledge.incoming.settlement | DDD representative command |
| FR-PCM-024 | POST | `/api/v1/payments/actions/reverse-incoming-settlement` | `pcmReverseIncomingSettlement` | finance.pcm.reverse.incoming.settlement | DDD representative command |
| FR-PCM-025 | PUT | `/api/v1/payments/configuration/maintain-bank-accounts` | `pcmMaintainBankAccounts` | finance.pcm.maintain.bank.accounts | PRD functional action |
| FR-RPT-001 | POST | `/api/v1/reporting/actions/run-consolidation` | `rptRunConsolidation` | finance.rpt.run.consolidation | DDD representative command |
| FR-RPT-002 | POST | `/api/v1/reporting/actions/apply-translation-result` | `rptApplyTranslationResult` | finance.rpt.apply.translation.result | DDD representative command |
| FR-RPT-003 | POST | `/api/v1/reporting/actions/apply-consolidation-approval-decision` | `rptApplyConsolidationApprovalDecision` | finance.rpt.apply.consolidation.approval.decision | DDD representative command |
| FR-RPT-004 | POST | `/api/v1/reporting/actions/publish-consolidated-statement` | `rptPublishConsolidatedStatement` | finance.rpt.publish.consolidated.statement | DDD representative command |
| FR-RPT-005 | PUT | `/api/v1/reporting/configuration/maintain-report-definitions` | `rptMaintainReportDefinitions` | finance.rpt.maintain.report.definitions | PRD functional action |
| FR-RPT-006 | POST | `/api/v1/reporting/actions/generate-and-publish-ledger-financial-statements` | `rptGenerateAndPublishLedgerFinancialStatements` | finance.rpt.generate.and.publish.ledger.financial.statements | PRD functional action |
| FR-IC-001 | POST | `/api/v1/intercompany/actions/start-settlement` | `icStartSettlement` | finance.ic.start.settlement | DDD representative command |
| FR-IC-002 | POST | `/api/v1/intercompany/actions/match-intercompany-items` | `icMatchIntercompanyItems` | finance.ic.match.intercompany.items | DDD representative command |
| FR-IC-003 | POST | `/api/v1/intercompany/actions/apply-residual-approval-decision` | `icApplyResidualApprovalDecision` | finance.ic.apply.residual.approval.decision | DDD representative command |
| FR-IC-004 | POST | `/api/v1/intercompany/actions/create-settlement-instructions` | `icCreateSettlementInstructions` | finance.ic.create.settlement.instructions | DDD representative command |
| FR-IC-005 | POST | `/api/v1/intercompany/actions/complete-settlement-run` | `icCompleteSettlementRun` | finance.ic.complete.settlement.run | DDD representative command |
| FR-IC-006 | POST | `/api/v1/intercompany/actions/apply-incoming-settlement` | `icApplyIncomingSettlement` | finance.ic.apply.incoming.settlement | DDD representative command |
| FR-IC-007 | POST | `/api/v1/intercompany/actions/reverse-incoming-settlement-application` | `icReverseIncomingSettlementApplication` | finance.ic.reverse.incoming.settlement.application | DDD representative command |
| FR-IC-008 | POST | `/api/v1/intercompany/actions/apply-payment-return` | `icApplyPaymentReturn` | finance.ic.apply.payment.return | DDD representative command |
| FR-IC-009 | POST | `/api/v1/intercompany/actions/run-elimination` | `icRunElimination` | finance.ic.run.elimination | DDD representative command |
| FR-IC-010 | PUT | `/api/v1/intercompany/configuration/maintain-intercompany-agreements` | `icMaintainIntercompanyAgreements` | finance.ic.maintain.intercompany.agreements | PRD functional action |
| FR-IC-011 | POST | `/api/v1/intercompany/actions/record-intercompany-transactions` | `icRecordIntercompanyTransactions` | finance.ic.record.intercompany.transactions | PRD functional action |
| FR-REV-001 | POST | `/api/v1/revenue-recognition/actions/assess-contract` | `revAssessContract` | finance.rev.assess.contract | DDD representative command |
| FR-REV-002 | POST | `/api/v1/revenue-recognition/actions/apply-revenue-schedule-approval-decision` | `revApplyRevenueScheduleApprovalDecision` | finance.rev.apply.revenue.schedule.approval.decision | DDD representative command |
| FR-REV-003 | POST | `/api/v1/revenue-recognition/actions/publish-revenue-accounting-profile` | `revPublishRevenueAccountingProfile` | finance.rev.publish.revenue.accounting.profile | DDD representative command |
| FR-REV-004 | POST | `/api/v1/revenue-recognition/actions/modify-contract` | `revModifyContract` | finance.rev.modify.contract | DDD representative command |
| FR-REV-005 | POST | `/api/v1/revenue-recognition/actions/apply-contract-modification-approval-decision` | `revApplyContractModificationApprovalDecision` | finance.rev.apply.contract.modification.approval.decision | DDD representative command |
| FR-REV-006 | POST | `/api/v1/revenue-recognition/actions/run-recognition` | `revRunRecognition` | finance.rev.run.recognition | DDD representative command |
| FR-FA-001 | POST | `/api/v1/fixed-assets/actions/capitalize-asset` | `faCapitalizeAsset` | finance.fa.capitalize.asset | DDD representative command |
| FR-FA-002 | POST | `/api/v1/fixed-assets/actions/create-asset-acquisition-clearing` | `faCreateAssetAcquisitionClearing` | finance.fa.create.asset.acquisition.clearing | DDD representative command |
| FR-FA-003 | POST | `/api/v1/fixed-assets/actions/run-depreciation` | `faRunDepreciation` | finance.fa.run.depreciation | DDD representative command |
| FR-FA-004 | POST | `/api/v1/fixed-assets/actions/apply-impairment-approval-decision` | `faApplyImpairmentApprovalDecision` | finance.fa.apply.impairment.approval.decision | DDD representative command |
| FR-FA-005 | POST | `/api/v1/fixed-assets/actions/dispose-asset` | `faDisposeAsset` | finance.fa.dispose.asset | DDD representative command |
| FR-FA-006 | POST | `/api/v1/fixed-assets/actions/apply-asset-disposal-approval-decision` | `faApplyAssetDisposalApprovalDecision` | finance.fa.apply.asset.disposal.approval.decision | DDD representative command |
| FR-FA-007 | POST | `/api/v1/fixed-assets/actions/cancel-unposted-asset-disposal` | `faCancelUnpostedAssetDisposal` | finance.fa.cancel.unposted.asset.disposal | DDD representative command |
| FR-FA-008 | POST | `/api/v1/fixed-assets/actions/compensate-failed-disposal-posting` | `faCompensateFailedDisposalPosting` | finance.fa.compensate.failed.disposal.posting | DDD representative command |
| FR-FA-009 | POST | `/api/v1/fixed-assets/actions/create-disposal-settlement-clearing` | `faCreateDisposalSettlementClearing` | finance.fa.create.disposal.settlement.clearing | DDD representative command |
| FR-FA-010 | POST | `/api/v1/fixed-assets/actions/apply-asset-supplier-liability-result` | `faApplyAssetSupplierLiabilityResult` | finance.fa.apply.asset.supplier.liability.result | DDD representative command |
| FR-FA-011 | POST | `/api/v1/fixed-assets/actions/apply-incoming-settlement` | `faApplyIncomingSettlement` | finance.fa.apply.incoming.settlement | DDD representative command |
| FR-FA-012 | POST | `/api/v1/fixed-assets/actions/reverse-incoming-settlement-application` | `faReverseIncomingSettlementApplication` | finance.fa.reverse.incoming.settlement.application | DDD representative command |
| FR-FA-013 | POST | `/api/v1/fixed-assets/actions/apply-payment-return` | `faApplyPaymentReturn` | finance.fa.apply.payment.return | DDD representative command |
| FR-FA-014 | POST | `/api/v1/fixed-assets/actions/apply-asset-settlement-result` | `faApplyAssetSettlementResult` | finance.fa.apply.asset.settlement.result | DDD representative command |
| FR-FA-015 | POST | `/api/v1/fixed-assets/actions/reclassify-disposal-cost-for-payment` | `faReclassifyDisposalCostForPayment` | finance.fa.reclassify.disposal.cost.for.payment | DDD representative command |
| FR-FA-016 | POST | `/api/v1/fixed-assets/actions/request-disposal-cost-payment` | `faRequestDisposalCostPayment` | finance.fa.request.disposal.cost.payment | DDD representative command |
| FR-FA-017 | POST | `/api/v1/fixed-assets/actions/request-disposal-cost-payment-replacement` | `faRequestDisposalCostPaymentReplacement` | finance.fa.request.disposal.cost.payment.replacement | DDD representative command |
| FR-FA-018 | POST | `/api/v1/fixed-assets/actions/record-impairment-assessments` | `faRecordImpairmentAssessments` | finance.fa.record.impairment.assessments | PRD functional action |
| FR-FA-019 | POST | `/api/v1/fixed-assets/actions/transfer-assets-or-components` | `faTransferAssetsOrComponents` | finance.fa.transfer.assets.or.components | PRD functional action |
| FR-FA-020 | POST | `/api/v1/fixed-assets/actions/split-assets-or-components` | `faSplitAssetsOrComponents` | finance.fa.split.assets.or.components | PRD functional action |
| FR-FA-021 | POST | `/api/v1/fixed-assets/actions/correct-posted-asset-disposals` | `faCorrectPostedAssetDisposals` | finance.fa.correct.posted.asset.disposals | PRD functional action |
| FR-FX-001 | POST | `/api/v1/multi-currency/actions/publish-rate-set` | `fxPublishRateSet` | finance.fx.publish.rate.set | DDD representative command |
| FR-FX-002 | POST | `/api/v1/multi-currency/actions/run-revaluation` | `fxRunRevaluation` | finance.fx.run.revaluation | DDD representative command |
| FR-FX-003 | POST | `/api/v1/multi-currency/actions/apply-revaluation-approval-decision` | `fxApplyRevaluationApprovalDecision` | finance.fx.apply.revaluation.approval.decision | DDD representative command |
| FR-FX-004 | POST | `/api/v1/multi-currency/actions/post-revaluation-run` | `fxPostRevaluationRun` | finance.fx.post.revaluation.run | DDD representative command |
| FR-FX-005 | POST | `/api/v1/multi-currency/actions/run-translation` | `fxRunTranslation` | finance.fx.run.translation | DDD representative command |
| FR-FPM-001 | POST | `/api/v1/fiscal-periods/actions/start-soft-close` | `fpmStartSoftClose` | finance.fpm.start.soft.close | DDD representative command |
| FR-FPM-002 | POST | `/api/v1/fiscal-periods/actions/end-soft-close` | `fpmEndSoftClose` | finance.fpm.end.soft.close | DDD representative command |
| FR-FPM-003 | POST | `/api/v1/fiscal-periods/actions/start-hard-close` | `fpmStartHardClose` | finance.fpm.start.hard.close | DDD representative command |
| FR-FPM-004 | POST | `/api/v1/fiscal-periods/actions/resume-close-run` | `fpmResumeCloseRun` | finance.fpm.resume.close.run | DDD representative command |
| FR-FPM-005 | POST | `/api/v1/fiscal-periods/actions/abort-close-run` | `fpmAbortCloseRun` | finance.fpm.abort.close.run | DDD representative command |
| FR-FPM-006 | POST | `/api/v1/fiscal-periods/actions/apply-posting-gate-result` | `fpmApplyPostingGateResult` | finance.fpm.apply.posting.gate.result | DDD representative command |
| FR-FPM-007 | POST | `/api/v1/fiscal-periods/actions/apply-close-exception-approval-decision` | `fpmApplyCloseExceptionApprovalDecision` | finance.fpm.apply.close.exception.approval.decision | DDD representative command |
| FR-FPM-008 | POST | `/api/v1/fiscal-periods/actions/apply-close-approval-decision` | `fpmApplyCloseApprovalDecision` | finance.fpm.apply.close.approval.decision | DDD representative command |
| FR-FPM-009 | POST | `/api/v1/fiscal-periods/actions/request-reopen` | `fpmRequestReopen` | finance.fpm.request.reopen | DDD representative command |
| FR-FPM-010 | POST | `/api/v1/fiscal-periods/actions/apply-reopen-approval-decision` | `fpmApplyReopenApprovalDecision` | finance.fpm.apply.reopen.approval.decision | DDD representative command |
| FR-FPM-011 | POST | `/api/v1/fiscal-periods/actions/start-reclose` | `fpmStartReclose` | finance.fpm.start.reclose | DDD representative command |
| FR-FPM-012 | POST | `/api/v1/fiscal-periods/actions/take-over-period-control` | `fpmTakeOverPeriodControl` | finance.fpm.take.over.period.control | DDD representative command |
| FR-FPM-013 | POST | `/api/v1/fiscal-periods/actions/extend-close-exception` | `fpmExtendCloseException` | finance.fpm.extend.close.exception | DDD representative command |
| FR-COA-001 | PUT | `/api/v1/coa-segments/configuration/maintain-segment-definitions` | `coaMaintainSegmentDefinitions` | finance.coa.maintain.segment.definitions | PRD functional action |
| FR-COA-002 | PUT | `/api/v1/coa-segments/configuration/maintain-segment-values` | `coaMaintainSegmentValues` | finance.coa.maintain.segment.values | PRD functional action |
| FR-COA-003 | POST | `/api/v1/coa-segments/actions/validate-segment-combinations` | `coaValidateSegmentCombinations` | finance.coa.validate.segment.combinations | PRD functional action |
| FR-COA-004 | POST | `/api/v1/coa-segments/actions/request-segment-changes` | `coaRequestSegmentChanges` | finance.coa.request.segment.changes | PRD functional action |
| FR-COA-005 | POST | `/api/v1/coa-segments/actions/apply-segment-change-approval-decision` | `coaApplySegmentChangeApprovalDecision` | finance.coa.apply.segment.change.approval.decision | DDD detailed command |
| FR-BFR-001 | POST | `/api/v1/bank-reconciliation/actions/import-statement` | `bfrImportStatement` | finance.bfr.import.statement | DDD representative command |
| FR-BFR-002 | POST | `/api/v1/bank-reconciliation/actions/propose-match` | `bfrProposeMatch` | finance.bfr.propose.match | DDD representative command |
| FR-BFR-003 | POST | `/api/v1/bank-reconciliation/actions/confirm-match` | `bfrConfirmMatch` | finance.bfr.confirm.match | DDD representative command |
| FR-BFR-004 | POST | `/api/v1/bank-reconciliation/actions/unmatch` | `bfrUnmatch` | finance.bfr.unmatch | DDD representative command |
| FR-BFR-005 | POST | `/api/v1/bank-reconciliation/actions/complete-reconciliation` | `bfrCompleteReconciliation` | finance.bfr.complete.reconciliation | DDD representative command |
| FR-BFR-006 | PUT | `/api/v1/bank-reconciliation/configuration/maintain-bank-feed-connections` | `bfrMaintainBankFeedConnections` | finance.bfr.maintain.bank.feed.connections | PRD functional action |
| FR-TAX-001 | POST | `/api/v1/tax/actions/determine-tax` | `taxDetermineTax` | finance.tax.determine.tax | DDD representative command |
| FR-TAX-002 | POST | `/api/v1/tax/actions/prepare-tax-return` | `taxPrepareTaxReturn` | finance.tax.prepare.tax.return | DDD representative command |
| FR-TAX-003 | POST | `/api/v1/tax/actions/apply-tax-return-approval-decision` | `taxApplyTaxReturnApprovalDecision` | finance.tax.apply.tax.return.approval.decision | DDD representative command |
| FR-TAX-004 | POST | `/api/v1/tax/actions/submit-tax-return` | `taxSubmitTaxReturn` | finance.tax.submit.tax.return | DDD representative command |
| FR-TAX-005 | POST | `/api/v1/tax/actions/create-tax-amendment` | `taxCreateTaxAmendment` | finance.tax.create.tax.amendment | DDD representative command |
| FR-TAX-006 | POST | `/api/v1/tax/actions/apply-tax-amendment-approval-decision` | `taxApplyTaxAmendmentApprovalDecision` | finance.tax.apply.tax.amendment.approval.decision | DDD representative command |
| FR-TAX-007 | POST | `/api/v1/tax/actions/submit-tax-amendment` | `taxSubmitTaxAmendment` | finance.tax.submit.tax.amendment | DDD representative command |
| FR-TAX-008 | POST | `/api/v1/tax/actions/create-return-level-tax-adjustment` | `taxCreateReturnLevelTaxAdjustment` | finance.tax.create.return.level.tax.adjustment | DDD representative command |
| FR-TAX-009 | POST | `/api/v1/tax/actions/apply-return-level-tax-adjustment-approval-decision` | `taxApplyReturnLevelTaxAdjustmentApprovalDecision` | finance.tax.apply.return.level.tax.adjustment.approval.decision | DDD representative command |
| FR-TAX-010 | POST | `/api/v1/tax/actions/post-return-level-tax-adjustment` | `taxPostReturnLevelTaxAdjustment` | finance.tax.post.return.level.tax.adjustment | DDD representative command |
| FR-TAX-011 | POST | `/api/v1/tax/actions/request-tax-payment` | `taxRequestTaxPayment` | finance.tax.request.tax.payment | DDD representative command |
| FR-TAX-012 | POST | `/api/v1/tax/actions/record-tax-payment-settlement` | `taxRecordTaxPaymentSettlement` | finance.tax.record.tax.payment.settlement | DDD representative command |
| FR-TAX-013 | POST | `/api/v1/tax/actions/apply-incoming-settlement` | `taxApplyIncomingSettlement` | finance.tax.apply.incoming.settlement | DDD representative command |
| FR-TAX-014 | POST | `/api/v1/tax/actions/reverse-incoming-settlement-application` | `taxReverseIncomingSettlementApplication` | finance.tax.reverse.incoming.settlement.application | DDD representative command |
| FR-TAX-015 | POST | `/api/v1/tax/actions/apply-payment-return` | `taxApplyPaymentReturn` | finance.tax.apply.payment.return | DDD representative command |
| FR-TAX-016 | PUT | `/api/v1/tax/configuration/maintain-tax-configurations` | `taxMaintainTaxConfigurations` | finance.tax.maintain.tax.configurations | PRD functional action |
| FR-WFA-001 | POST | `/api/v1/approvals/actions/create-approval-request` | `wfaCreateApprovalRequest` | finance.wfa.create.approval.request | DDD representative command |
| FR-WFA-002 | POST | `/api/v1/approvals/actions/decide-approval-request` | `wfaDecideApprovalRequest` | finance.wfa.decide.approval.request | DDD representative command |
| FR-WFA-003 | POST | `/api/v1/approvals/actions/delegate-approval` | `wfaDelegateApproval` | finance.wfa.delegate.approval | DDD representative command |
| FR-WFA-004 | POST | `/api/v1/approvals/actions/escalate-approval` | `wfaEscalateApproval` | finance.wfa.escalate.approval | DDD representative command |
| FR-WFA-005 | PUT | `/api/v1/approvals/configuration/maintain-approval-policies` | `wfaMaintainApprovalPolicies` | finance.wfa.maintain.approval.policies | PRD functional action |
| FR-IAM-001 | POST | `/api/v1/identity-access/actions/manage-users` | `iamManageUsers` | finance.iam.manage.users | PRD functional action |
| FR-IAM-002 | POST | `/api/v1/identity-access/actions/manage-roles` | `iamManageRoles` | finance.iam.manage.roles | PRD functional action |
| FR-IAM-003 | POST | `/api/v1/identity-access/actions/manage-access-policies` | `iamManageAccessPolicies` | finance.iam.manage.access.policies | PRD functional action |
| FR-IAM-004 | POST | `/api/v1/identity-access/actions/manage-segregation-rules` | `iamManageSegregationRules` | finance.iam.manage.segregation.rules | PRD functional action |
| FR-IAM-005 | POST | `/api/v1/identity-access/actions/grant-emergency-access` | `iamGrantEmergencyAccess` | finance.iam.grant.emergency.access | PRD functional action |
| FR-IAM-006 | POST | `/api/v1/identity-access/actions/revoke-emergency-access` | `iamRevokeEmergencyAccess` | finance.iam.revoke.emergency.access | PRD functional action |
| FR-AUD-001 | POST | `/api/v1/audit-integrity/actions/append-auditable-event` | `audAppendAuditableEvent` | finance.aud.append.auditable.event | DDD representative command |
| FR-AUD-002 | POST | `/api/v1/audit-integrity/actions/create-audit-seal` | `audCreateAuditSeal` | finance.aud.create.audit.seal | DDD representative command |
| FR-AUD-003 | POST | `/api/v1/audit-integrity/actions/rotate-verification-credential` | `audRotateVerificationCredential` | finance.aud.rotate.verification.credential | DDD representative command |
| FR-AUD-004 | POST | `/api/v1/audit-integrity/actions/escalate-integrity-incident` | `audEscalateIntegrityIncident` | finance.aud.escalate.integrity.incident | DDD representative command |
| FR-AUD-005 | GET | `/api/v1/audit-integrity/reference/verify-proof` | `audVerifyProof` | finance.aud.verify.proof | DDD reference operation |

## 8. HTTP result rules

| Situation | Status | Body |
|---|---|---|
| Command establishes a result | 200 | Established result |
| New top-level resource created by a PRD action | 201 | Established result plus `Location` |
| External or long-running process accepted with no terminal outcome | 202 | Process identity and status link |
| Safe duplicate | Original success status | Original established result |
| Validation failure | 400 | Problem details with field errors |
| Domain rejection | 422 | Problem details with rule code and current state |
| Authorization denial | 403 | Problem details without sensitive policy disclosure |
| Version or identity-content conflict | 409 | Current version or idempotency conflict evidence |
| Dependency unavailable | 503 | Retryability and result-lookup guidance |

## 9. OpenAPI file structure

```text
contracts/openapi/openapi.yaml
contracts/openapi/components/common.yaml
contracts/openapi/components/money.yaml
contracts/openapi/components/errors.yaml
contracts/openapi/paths/<capability>.yaml
contracts/openapi/examples/
```

CI shall lint and bundle OpenAPI, reject duplicate `operationId` values, generate TypeScript client types, and compare generated artifacts with source control.

## 10. Money and sensitive fields

- Monetary amount fields are JSON strings matching `^-?[0-9]+(\.[0-9]+)?$`.
- Currency is a three-letter uppercase code.
- Bank account numbers, tax identifiers and payroll details are never returned in general worklists.
- Sensitive fields use explicit response projections; there is no generic reflection-based serialization of domain aggregates.

## 11. Rate limits and request sizes

- Default interactive limit: 120 requests/minute/user and 600 requests/minute/tenant in the learning profile.
- Command body maximum: 2 MiB; approved statement/import endpoints use a separate upload flow and size policy.
- Rate limiting never substitutes for business admission controls and does not discard accepted work.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `9ee7272934fd3961c73273fcd1b4bbe20c4c486e657cf6c754066c37722fa797` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
