# Finance Platform Technical Traceability, Decisions and Verification

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Consistency-verified technical specification baseline |

## 1. Source manifest

| Source key | File | Full-file SHA-256 |
|---|---|---|
| ddd | finance_domain_model_ddd_v3.1.md | 0edb1702b5278e75d80efb98d2e909110e73dd34bd362e6149a11bb1e0554402 |
| prd | 01_finance_functional_prd_v1.5.md | 50f7451b6f768feb2fbf13aa928acb3aefb590c8c36a990ecfef6988c0705335 |
| catalog | 02_finance_functional_requirements_catalog_v1.5.md | bbd490b458597779f792157c5f146ff488a12f26c9b26cd0f1d31b391ed52f5f |
| acceptance | 03_finance_functional_traceability_acceptance_v1.5.md | 1f511e2d699dc5ae2c83abe720d20d29c5eaa1683bb69d3e7f545099dad63f4e |
| ux | finance_ux_workflow_specification_v1.0.md | 84761c9805b7e639ae9dab8fb3800f6e02d80c843ec86aac9e9057241c3501d5 |
| nfr | finance_nonfunctional_requirements_v1.0.md | 52099bfa577d7682493e95702d82c8e1f8b23c1e038a7ad4a935815b10df3dbd |
| sol1 | 01_solution_architecture_overview_v1.0.md | eccf07c9a5a17d631277a0ad4b49767c440999386cc1f49b019208cd54437a08 |
| sol2 | 02_application_module_design_v1.0.md | ddfda9ae43daf2212161fd340c6330cb06b2e964a780d81866e8513d6a213735 |
| sol3 | 03_data_integration_architecture_v1.0.md | 06478bbb5c0f5e336c3fcd5d8b2398e17ac43fcaebc1e3b2245c02beb9b79719 |
| sol4 | 04_security_deployment_operations_v1.0.md | 8f817895f37a129458c2632a72ad00b5ef1dbddbc9ef8ee25862236ec26b47ce |
| sol5 | 05_architecture_traceability_decisions_v1.0.md | fbbdd612ee00498e16caac4d8fe543e8f763f443ca1ab2ed68ca79495892f8e3 |

## 2. Technical document map

| Document | Purpose |
|---|---|
| 01_backend_module_specifications_v1.0.md | Backend modules and command implementation |
| 02_api_openapi_specifications_v1.0.md | REST/OpenAPI contracts |
| 03_database_persistence_specifications_v1.0.md | PostgreSQL persistence |
| 04_events_workers_integration_specifications_v1.0.md | Events/workers/integrations |
| 05_frontend_ui_technical_specifications_v1.0.md | React/Tailwind/daisyUI implementation |
| 06_security_identity_authorization_specifications_v1.0.md | Entra and finance authorization |
| 07_terraform_azure_deployment_specifications_v1.0.md | Terraform/Azure deployment |
| 08_observability_operations_specifications_v1.0.md | Telemetry and operations |
| 09_testing_performance_recovery_specifications_v1.0.md | Testing, performance and recovery |

## 3. Global functional traceability

| Requirement | Technical documents | Control |
|---|---|---|
| GFR-001 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-002 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-003 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-004 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-005 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-006 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-007 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-008 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-009 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-010 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-011 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-012 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-013 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-014 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-015 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-016 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-017 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-018 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-019 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-020 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-021 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |
| GFR-022 | 01, 02, 03, 04, 05, 06, 08, 09 | Cross-cutting implementation controls |

## 4. Capability functional traceability

| Requirement | Operation | Technical documents | Handler / endpoint | Permission |
|---|---|---|---|---|
| FR-OMD-001 | Maintain legal entities | 01, 02, 03, 05, 06, 09 | `omdMaintainLegalEntities` / `/api/v1/master-data/configuration/maintain-legal-entities` | finance.omd.maintain.legal.entities |
| FR-OMD-002 | Maintain parties | 01, 02, 03, 05, 06, 09 | `omdMaintainParties` / `/api/v1/master-data/configuration/maintain-parties` | finance.omd.maintain.parties |
| FR-OMD-003 | Maintain customer profiles | 01, 02, 03, 05, 06, 09 | `omdMaintainCustomerProfiles` / `/api/v1/master-data/configuration/maintain-customer-profiles` | finance.omd.maintain.customer.profiles |
| FR-OMD-004 | Maintain vendor profiles | 01, 02, 03, 05, 06, 09 | `omdMaintainVendorProfiles` / `/api/v1/master-data/configuration/maintain-vendor-profiles` | finance.omd.maintain.vendor.profiles |
| FR-OMD-005 | Maintain fiscal calendars | 01, 02, 03, 05, 06, 09 | `omdMaintainFiscalCalendars` / `/api/v1/master-data/configuration/maintain-fiscal-calendars` | finance.omd.maintain.fiscal.calendars |
| FR-OMD-006 | Publish approved master-data changes | 01, 02, 03, 05, 06, 09 | `omdPublishApprovedMasterDataChanges` / `/api/v1/master-data/actions/publish-approved-master-data-changes` | finance.omd.publish.approved.master.data.changes |
| FR-GL-001 | SubmitPostingRequest | 01, 02, 03, 05, 06, 09 | `glSubmitPostingRequest` / `/api/v1/general-ledger/actions/submit-posting-request` | finance.gl.submit.posting.request |
| FR-GL-002 | ApplyJournalApprovalDecision | 01, 02, 03, 05, 06, 09 | `glApplyJournalApprovalDecision` / `/api/v1/general-ledger/actions/apply-journal-approval-decision` | finance.gl.apply.journal.approval.decision |
| FR-GL-003 | ReverseJournalEntry | 01, 02, 03, 05, 06, 09 | `glReverseJournalEntry` / `/api/v1/general-ledger/actions/reverse-journal-entry` | finance.gl.reverse.journal.entry |
| FR-GL-004 | EnterSoftCloseGate | 01, 02, 03, 05, 06, 09 | `glEnterSoftCloseGate` / `/api/v1/general-ledger/actions/enter-soft-close-gate` | finance.gl.enter.soft.close.gate |
| FR-GL-005 | ExitSoftCloseGate | 01, 02, 03, 05, 06, 09 | `glExitSoftCloseGate` / `/api/v1/general-ledger/actions/exit-soft-close-gate` | finance.gl.exit.soft.close.gate |
| FR-GL-006 | AcquirePostingBarrier | 01, 02, 03, 05, 06, 09 | `glAcquirePostingBarrier` / `/api/v1/general-ledger/actions/acquire-posting-barrier` | finance.gl.acquire.posting.barrier |
| FR-GL-007 | ReleasePostingBarrier | 01, 02, 03, 05, 06, 09 | `glReleasePostingBarrier` / `/api/v1/general-ledger/actions/release-posting-barrier` | finance.gl.release.posting.barrier |
| FR-GL-008 | FinalizePostingGate | 01, 02, 03, 05, 06, 09 | `glFinalizePostingGate` / `/api/v1/general-ledger/actions/finalize-posting-gate` | finance.gl.finalize.posting.gate |
| FR-GL-009 | OpenScopedReopenGate | 01, 02, 03, 05, 06, 09 | `glOpenScopedReopenGate` / `/api/v1/general-ledger/actions/open-scoped-reopen-gate` | finance.gl.open.scoped.reopen.gate |
| FR-GL-010 | CloseScopedReopenGate | 01, 02, 03, 05, 06, 09 | `glCloseScopedReopenGate` / `/api/v1/general-ledger/actions/close-scoped-reopen-gate` | finance.gl.close.scoped.reopen.gate |
| FR-GL-011 | OpenOperationalReopenGate | 01, 02, 03, 05, 06, 09 | `glOpenOperationalReopenGate` / `/api/v1/general-ledger/actions/open-operational-reopen-gate` | finance.gl.open.operational.reopen.gate |
| FR-GL-012 | CloseOperationalReopenGate | 01, 02, 03, 05, 06, 09 | `glCloseOperationalReopenGate` / `/api/v1/general-ledger/actions/close-operational-reopen-gate` | finance.gl.close.operational.reopen.gate |
| FR-GL-013 | BeginRecloseGate | 01, 02, 03, 05, 06, 09 | `glBeginRecloseGate` / `/api/v1/general-ledger/actions/begin-reclose-gate` | finance.gl.begin.reclose.gate |
| FR-GL-014 | GetPostingGateStatus | 01, 02, 03, 05, 06, 09 | `glGetPostingGateStatus` / `/api/v1/general-ledger/reference/get-posting-gate-status` | finance.gl.get.posting.gate.status |
| FR-GL-015 | Maintain ledgers | 01, 02, 03, 05, 06, 09 | `glMaintainLedgers` / `/api/v1/general-ledger/configuration/maintain-ledgers` | finance.gl.maintain.ledgers |
| FR-GL-016 | Maintain accounting books | 01, 02, 03, 05, 06, 09 | `glMaintainAccountingBooks` / `/api/v1/general-ledger/configuration/maintain-accounting-books` | finance.gl.maintain.accounting.books |
| FR-GL-017 | Maintain charts of accounts | 01, 02, 03, 05, 06, 09 | `glMaintainChartsOfAccounts` / `/api/v1/general-ledger/configuration/maintain-charts-of-accounts` | finance.gl.maintain.charts.of.accounts |
| FR-GL-018 | Maintain accounts and reporting mappings | 01, 02, 03, 05, 06, 09 | `glMaintainAccountsAndReportingMappings` / `/api/v1/general-ledger/configuration/maintain-accounts-and-reporting-mappings` | finance.gl.maintain.accounts.and.reporting.mappings |
| FR-AP-001 | RegisterVendorInvoice | 01, 02, 03, 05, 06, 09 | `apRegisterVendorInvoice` / `/api/v1/accounts-payable/actions/register-vendor-invoice` | finance.ap.register.vendor.invoice |
| FR-AP-002 | ApplyAssetClearingClassification | 01, 02, 03, 05, 06, 09 | `apApplyAssetClearingClassification` / `/api/v1/accounts-payable/actions/apply-asset-clearing-classification` | finance.ap.apply.asset.clearing.classification |
| FR-AP-003 | ApplyIncomingSettlement | 01, 02, 03, 05, 06, 09 | `apApplyIncomingSettlement` / `/api/v1/accounts-payable/actions/apply-incoming-settlement` | finance.ap.apply.incoming.settlement |
| FR-AP-004 | ReverseIncomingSettlementApplication | 01, 02, 03, 05, 06, 09 | `apReverseIncomingSettlementApplication` / `/api/v1/accounts-payable/actions/reverse-incoming-settlement-application` | finance.ap.reverse.incoming.settlement.application |
| FR-AP-005 | ApplyPaymentReturn | 01, 02, 03, 05, 06, 09 | `apApplyPaymentReturn` / `/api/v1/accounts-payable/actions/apply-payment-return` | finance.ap.apply.payment.return |
| FR-AP-006 | ApplyVendorInvoiceApprovalDecision | 01, 02, 03, 05, 06, 09 | `apApplyVendorInvoiceApprovalDecision` / `/api/v1/accounts-payable/actions/apply-vendor-invoice-approval-decision` | finance.ap.apply.vendor.invoice.approval.decision |
| FR-AP-007 | RequestPayment | 01, 02, 03, 05, 06, 09 | `apRequestPayment` / `/api/v1/accounts-payable/actions/request-payment` | finance.ap.request.payment |
| FR-AP-008 | ValidateVendorInvoice | 01, 02, 03, 05, 06, 09 | `apValidateVendorInvoice` / `/api/v1/accounts-payable/actions/validate-vendor-invoice` | finance.ap.validate.vendor.invoice |
| FR-AP-009 | DisputeVendorInvoice | 01, 02, 03, 05, 06, 09 | `apDisputeVendorInvoice` / `/api/v1/accounts-payable/actions/dispute-vendor-invoice` | finance.ap.dispute.vendor.invoice |
| FR-AP-010 | VoidVendorInvoice | 01, 02, 03, 05, 06, 09 | `apVoidVendorInvoice` / `/api/v1/accounts-payable/actions/void-vendor-invoice` | finance.ap.void.vendor.invoice |
| FR-AR-001 | IssueCustomerInvoice | 01, 02, 03, 05, 06, 09 | `arIssueCustomerInvoice` / `/api/v1/accounts-receivable/actions/issue-customer-invoice` | finance.ar.issue.customer.invoice |
| FR-AR-002 | RecordReceipt | 01, 02, 03, 05, 06, 09 | `arRecordReceipt` / `/api/v1/accounts-receivable/actions/record-receipt` | finance.ar.record.receipt |
| FR-AR-003 | ApplyReceipt | 01, 02, 03, 05, 06, 09 | `arApplyReceipt` / `/api/v1/accounts-receivable/actions/apply-receipt` | finance.ar.apply.receipt |
| FR-AR-004 | UnapplyReceipt | 01, 02, 03, 05, 06, 09 | `arUnapplyReceipt` / `/api/v1/accounts-receivable/actions/unapply-receipt` | finance.ar.unapply.receipt |
| FR-AR-005 | RollbackUnpostedApplicationBatch | 01, 02, 03, 05, 06, 09 | `arRollbackUnpostedApplicationBatch` / `/api/v1/accounts-receivable/actions/rollback-unposted-application-batch` | finance.ar.rollback.unposted.application.batch |
| FR-AR-006 | IssueCreditNote | 01, 02, 03, 05, 06, 09 | `arIssueCreditNote` / `/api/v1/accounts-receivable/actions/issue-credit-note` | finance.ar.issue.credit.note |
| FR-AR-007 | CreateCustomerRefundRequest | 01, 02, 03, 05, 06, 09 | `arCreateCustomerRefundRequest` / `/api/v1/accounts-receivable/actions/create-customer-refund-request` | finance.ar.create.customer.refund.request |
| FR-AR-008 | CancelCustomerRefundRequest | 01, 02, 03, 05, 06, 09 | `arCancelCustomerRefundRequest` / `/api/v1/accounts-receivable/actions/cancel-customer-refund-request` | finance.ar.cancel.customer.refund.request |
| FR-AR-009 | ApplyCustomerRefundApprovalDecision | 01, 02, 03, 05, 06, 09 | `arApplyCustomerRefundApprovalDecision` / `/api/v1/accounts-receivable/actions/apply-customer-refund-approval-decision` | finance.ar.apply.customer.refund.approval.decision |
| FR-AR-010 | RequestCustomerRefundPayment | 01, 02, 03, 05, 06, 09 | `arRequestCustomerRefundPayment` / `/api/v1/accounts-receivable/actions/request-customer-refund-payment` | finance.ar.request.customer.refund.payment |
| FR-AR-011 | CancelCustomerRefundPayment | 01, 02, 03, 05, 06, 09 | `arCancelCustomerRefundPayment` / `/api/v1/accounts-receivable/actions/cancel-customer-refund-payment` | finance.ar.cancel.customer.refund.payment |
| FR-AR-012 | ApplyCustomerRefundPaymentResult | 01, 02, 03, 05, 06, 09 | `arApplyCustomerRefundPaymentResult` / `/api/v1/accounts-receivable/actions/apply-customer-refund-payment-result` | finance.ar.apply.customer.refund.payment.result |
| FR-AR-013 | ApplyPaymentReturn | 01, 02, 03, 05, 06, 09 | `arApplyPaymentReturn` / `/api/v1/accounts-receivable/actions/apply-payment-return` | finance.ar.apply.payment.return |
| FR-AR-014 | Resolve customer overpayments | 01, 02, 03, 05, 06, 09 | `arResolveCustomerOverpayments` / `/api/v1/accounts-receivable/actions/resolve-customer-overpayments` | finance.ar.resolve.customer.overpayments |
| FR-AR-015 | Record customer chargebacks | 01, 02, 03, 05, 06, 09 | `arRecordCustomerChargebacks` / `/api/v1/accounts-receivable/actions/record-customer-chargebacks` | finance.ar.record.customer.chargebacks |
| FR-AR-016 | Record receivable write-offs | 01, 02, 03, 05, 06, 09 | `arRecordReceivableWriteOffs` / `/api/v1/accounts-receivable/actions/record-receivable-write-offs` | finance.ar.record.receivable.write.offs |
| FR-PAYR-001 | CalculatePayrollRun | 01, 02, 03, 05, 06, 09 | `payrCalculatePayrollRun` / `/api/v1/payroll/actions/calculate-payroll-run` | finance.payr.calculate.payroll.run |
| FR-PAYR-002 | ApplyPayrollRunApprovalDecision | 01, 02, 03, 05, 06, 09 | `payrApplyPayrollRunApprovalDecision` / `/api/v1/payroll/actions/apply-payroll-run-approval-decision` | finance.payr.apply.payroll.run.approval.decision |
| FR-PAYR-003 | PostPayrollRun | 01, 02, 03, 05, 06, 09 | `payrPostPayrollRun` / `/api/v1/payroll/actions/post-payroll-run` | finance.payr.post.payroll.run |
| FR-PAYR-004 | CreatePayrollCorrection | 01, 02, 03, 05, 06, 09 | `payrCreatePayrollCorrection` / `/api/v1/payroll/actions/create-payroll-correction` | finance.payr.create.payroll.correction |
| FR-PAYR-005 | ApplyPaymentReturn | 01, 02, 03, 05, 06, 09 | `payrApplyPaymentReturn` / `/api/v1/payroll/actions/apply-payment-return` | finance.payr.apply.payment.return |
| FR-PAYR-006 | Maintain employee payroll profiles | 01, 02, 03, 05, 06, 09 | `payrMaintainEmployeePayrollProfiles` / `/api/v1/payroll/configuration/maintain-employee-payroll-profiles` | finance.payr.maintain.employee.payroll.profiles |
| FR-PAYR-007 | Maintain payroll tax-filing records | 01, 02, 03, 05, 06, 09 | `payrMaintainPayrollTaxFilingRecords` / `/api/v1/payroll/configuration/maintain-payroll-tax-filing-records` | finance.payr.maintain.payroll.tax.filing.records |
| FR-INV-001 | Configure invoice templates | 01, 02, 03, 05, 06, 09 | `invConfigureInvoiceTemplates` / `/api/v1/invoicing/configuration/configure-invoice-templates` | finance.inv.configure.invoice.templates |
| FR-INV-002 | Configure billing schedules | 01, 02, 03, 05, 06, 09 | `invConfigureBillingSchedules` / `/api/v1/invoicing/configuration/configure-billing-schedules` | finance.inv.configure.billing.schedules |
| FR-INV-003 | Generate invoices | 01, 02, 03, 05, 06, 09 | `invGenerateInvoices` / `/api/v1/invoicing/actions/generate-invoices` | finance.inv.generate.invoices |
| FR-INV-004 | Finalize generated invoices | 01, 02, 03, 05, 06, 09 | `invFinalizeGeneratedInvoices` / `/api/v1/invoicing/actions/finalize-generated-invoices` | finance.inv.finalize.generated.invoices |
| FR-INV-005 | Recalculate unfinalized invoices | 01, 02, 03, 05, 06, 09 | `invRecalculateUnfinalizedInvoices` / `/api/v1/invoicing/actions/recalculate-unfinalized-invoices` | finance.inv.recalculate.unfinalized.invoices |
| FR-INV-006 | Cancel unfinalized invoices | 01, 02, 03, 05, 06, 09 | `invCancelUnfinalizedInvoices` / `/api/v1/invoicing/actions/cancel-unfinalized-invoices` | finance.inv.cancel.unfinalized.invoices |
| FR-PCM-001 | PreparePaymentBatch | 01, 02, 03, 05, 06, 09 | `pcmPreparePaymentBatch` / `/api/v1/payments/actions/prepare-payment-batch` | finance.pcm.prepare.payment.batch |
| FR-PCM-002 | ApplyPaymentBatchApprovalDecision | 01, 02, 03, 05, 06, 09 | `pcmApplyPaymentBatchApprovalDecision` / `/api/v1/payments/actions/apply-payment-batch-approval-decision` | finance.pcm.apply.payment.batch.approval.decision |
| FR-PCM-003 | CancelPaymentBatch | 01, 02, 03, 05, 06, 09 | `pcmCancelPaymentBatch` / `/api/v1/payments/actions/cancel-payment-batch` | finance.pcm.cancel.payment.batch |
| FR-PCM-004 | RegisterExpectedIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmRegisterExpectedIncomingSettlement` / `/api/v1/payments/actions/register-expected-incoming-settlement` | finance.pcm.register.expected.incoming.settlement |
| FR-PCM-005 | ResolveExpectedIncomingSettlementException | 01, 02, 03, 05, 06, 09 | `pcmResolveExpectedIncomingSettlementException` / `/api/v1/payments/actions/resolve-expected-incoming-settlement-exception` | finance.pcm.resolve.expected.incoming.settlement.exception |
| FR-PCM-006 | CancelExpectedIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmCancelExpectedIncomingSettlement` / `/api/v1/payments/actions/cancel-expected-incoming-settlement` | finance.pcm.cancel.expected.incoming.settlement |
| FR-PCM-007 | CloseExpectedIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmCloseExpectedIncomingSettlement` / `/api/v1/payments/actions/close-expected-incoming-settlement` | finance.pcm.close.expected.incoming.settlement |
| FR-PCM-008 | CreatePaymentInstructionFromObligation | 01, 02, 03, 05, 06, 09 | `pcmCreatePaymentInstructionFromObligation` / `/api/v1/payments/actions/create-payment-instruction-from-obligation` | finance.pcm.create.payment.instruction.from.obligation |
| FR-PCM-009 | SubmitPaymentInstruction | 01, 02, 03, 05, 06, 09 | `pcmSubmitPaymentInstruction` / `/api/v1/payments/actions/submit-payment-instruction` | finance.pcm.submit.payment.instruction |
| FR-PCM-010 | RetryPaymentInstruction | 01, 02, 03, 05, 06, 09 | `pcmRetryPaymentInstruction` / `/api/v1/payments/actions/retry-payment-instruction` | finance.pcm.retry.payment.instruction |
| FR-PCM-011 | CancelPaymentInstruction | 01, 02, 03, 05, 06, 09 | `pcmCancelPaymentInstruction` / `/api/v1/payments/actions/cancel-payment-instruction` | finance.pcm.cancel.payment.instruction |
| FR-PCM-012 | ApplyPaymentInstructionExceptionDecision | 01, 02, 03, 05, 06, 09 | `pcmApplyPaymentInstructionExceptionDecision` / `/api/v1/payments/actions/apply-payment-instruction-exception-decision` | finance.pcm.apply.payment.instruction.exception.decision |
| FR-PCM-013 | RecordPaymentReturn | 01, 02, 03, 05, 06, 09 | `pcmRecordPaymentReturn` / `/api/v1/payments/actions/record-payment-return` | finance.pcm.record.payment.return |
| FR-PCM-014 | CancelUnpostedPaymentReturn | 01, 02, 03, 05, 06, 09 | `pcmCancelUnpostedPaymentReturn` / `/api/v1/payments/actions/cancel-unposted-payment-return` | finance.pcm.cancel.unposted.payment.return |
| FR-PCM-015 | AcknowledgePaymentReturn | 01, 02, 03, 05, 06, 09 | `pcmAcknowledgePaymentReturn` / `/api/v1/payments/actions/acknowledge-payment-return` | finance.pcm.acknowledge.payment.return |
| FR-PCM-016 | ResolvePaymentReturnException | 01, 02, 03, 05, 06, 09 | `pcmResolvePaymentReturnException` / `/api/v1/payments/actions/resolve-payment-return-exception` | finance.pcm.resolve.payment.return.exception |
| FR-PCM-017 | RecordUnallocatedIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmRecordUnallocatedIncomingSettlement` / `/api/v1/payments/actions/record-unallocated-incoming-settlement` | finance.pcm.record.unallocated.incoming.settlement |
| FR-PCM-018 | ResolveUnallocatedIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmResolveUnallocatedIncomingSettlement` / `/api/v1/payments/actions/resolve-unallocated-incoming-settlement` | finance.pcm.resolve.unallocated.incoming.settlement |
| FR-PCM-019 | RecordIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmRecordIncomingSettlement` / `/api/v1/payments/actions/record-incoming-settlement` | finance.pcm.record.incoming.settlement |
| FR-PCM-020 | ResolveSettlementReceiptValidationException | 01, 02, 03, 05, 06, 09 | `pcmResolveSettlementReceiptValidationException` / `/api/v1/payments/actions/resolve-settlement-receipt-validation-exception` | finance.pcm.resolve.settlement.receipt.validation.exception |
| FR-PCM-021 | ResolveIncomingSettlementOwnerException | 01, 02, 03, 05, 06, 09 | `pcmResolveIncomingSettlementOwnerException` / `/api/v1/payments/actions/resolve-incoming-settlement-owner-exception` | finance.pcm.resolve.incoming.settlement.owner.exception |
| FR-PCM-022 | CancelUnpostedSettlementReceipt | 01, 02, 03, 05, 06, 09 | `pcmCancelUnpostedSettlementReceipt` / `/api/v1/payments/actions/cancel-unposted-settlement-receipt` | finance.pcm.cancel.unposted.settlement.receipt |
| FR-PCM-023 | AcknowledgeIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmAcknowledgeIncomingSettlement` / `/api/v1/payments/actions/acknowledge-incoming-settlement` | finance.pcm.acknowledge.incoming.settlement |
| FR-PCM-024 | ReverseIncomingSettlement | 01, 02, 03, 05, 06, 09 | `pcmReverseIncomingSettlement` / `/api/v1/payments/actions/reverse-incoming-settlement` | finance.pcm.reverse.incoming.settlement |
| FR-PCM-025 | Maintain bank accounts | 01, 02, 03, 05, 06, 09 | `pcmMaintainBankAccounts` / `/api/v1/payments/configuration/maintain-bank-accounts` | finance.pcm.maintain.bank.accounts |
| FR-RPT-001 | RunConsolidation | 01, 02, 03, 05, 06, 09 | `rptRunConsolidation` / `/api/v1/reporting/actions/run-consolidation` | finance.rpt.run.consolidation |
| FR-RPT-002 | ApplyTranslationResult | 01, 02, 03, 05, 06, 09 | `rptApplyTranslationResult` / `/api/v1/reporting/actions/apply-translation-result` | finance.rpt.apply.translation.result |
| FR-RPT-003 | ApplyConsolidationApprovalDecision | 01, 02, 03, 05, 06, 09 | `rptApplyConsolidationApprovalDecision` / `/api/v1/reporting/actions/apply-consolidation-approval-decision` | finance.rpt.apply.consolidation.approval.decision |
| FR-RPT-004 | PublishConsolidatedStatement | 01, 02, 03, 05, 06, 09 | `rptPublishConsolidatedStatement` / `/api/v1/reporting/actions/publish-consolidated-statement` | finance.rpt.publish.consolidated.statement |
| FR-RPT-005 | Maintain report definitions | 01, 02, 03, 05, 06, 09 | `rptMaintainReportDefinitions` / `/api/v1/reporting/configuration/maintain-report-definitions` | finance.rpt.maintain.report.definitions |
| FR-RPT-006 | Generate and publish ledger financial statements | 01, 02, 03, 05, 06, 09 | `rptGenerateAndPublishLedgerFinancialStatements` / `/api/v1/reporting/actions/generate-and-publish-ledger-financial-statements` | finance.rpt.generate.and.publish.ledger.financial.statements |
| FR-IC-001 | StartSettlement | 01, 02, 03, 05, 06, 09 | `icStartSettlement` / `/api/v1/intercompany/actions/start-settlement` | finance.ic.start.settlement |
| FR-IC-002 | MatchIntercompanyItems | 01, 02, 03, 05, 06, 09 | `icMatchIntercompanyItems` / `/api/v1/intercompany/actions/match-intercompany-items` | finance.ic.match.intercompany.items |
| FR-IC-003 | ApplyResidualApprovalDecision | 01, 02, 03, 05, 06, 09 | `icApplyResidualApprovalDecision` / `/api/v1/intercompany/actions/apply-residual-approval-decision` | finance.ic.apply.residual.approval.decision |
| FR-IC-004 | CreateSettlementInstructions | 01, 02, 03, 05, 06, 09 | `icCreateSettlementInstructions` / `/api/v1/intercompany/actions/create-settlement-instructions` | finance.ic.create.settlement.instructions |
| FR-IC-005 | CompleteSettlementRun | 01, 02, 03, 05, 06, 09 | `icCompleteSettlementRun` / `/api/v1/intercompany/actions/complete-settlement-run` | finance.ic.complete.settlement.run |
| FR-IC-006 | ApplyIncomingSettlement | 01, 02, 03, 05, 06, 09 | `icApplyIncomingSettlement` / `/api/v1/intercompany/actions/apply-incoming-settlement` | finance.ic.apply.incoming.settlement |
| FR-IC-007 | ReverseIncomingSettlementApplication | 01, 02, 03, 05, 06, 09 | `icReverseIncomingSettlementApplication` / `/api/v1/intercompany/actions/reverse-incoming-settlement-application` | finance.ic.reverse.incoming.settlement.application |
| FR-IC-008 | ApplyPaymentReturn | 01, 02, 03, 05, 06, 09 | `icApplyPaymentReturn` / `/api/v1/intercompany/actions/apply-payment-return` | finance.ic.apply.payment.return |
| FR-IC-009 | RunElimination | 01, 02, 03, 05, 06, 09 | `icRunElimination` / `/api/v1/intercompany/actions/run-elimination` | finance.ic.run.elimination |
| FR-IC-010 | Maintain intercompany agreements | 01, 02, 03, 05, 06, 09 | `icMaintainIntercompanyAgreements` / `/api/v1/intercompany/configuration/maintain-intercompany-agreements` | finance.ic.maintain.intercompany.agreements |
| FR-IC-011 | Record intercompany transactions | 01, 02, 03, 05, 06, 09 | `icRecordIntercompanyTransactions` / `/api/v1/intercompany/actions/record-intercompany-transactions` | finance.ic.record.intercompany.transactions |
| FR-REV-001 | AssessContract | 01, 02, 03, 05, 06, 09 | `revAssessContract` / `/api/v1/revenue-recognition/actions/assess-contract` | finance.rev.assess.contract |
| FR-REV-002 | ApplyRevenueScheduleApprovalDecision | 01, 02, 03, 05, 06, 09 | `revApplyRevenueScheduleApprovalDecision` / `/api/v1/revenue-recognition/actions/apply-revenue-schedule-approval-decision` | finance.rev.apply.revenue.schedule.approval.decision |
| FR-REV-003 | PublishRevenueAccountingProfile | 01, 02, 03, 05, 06, 09 | `revPublishRevenueAccountingProfile` / `/api/v1/revenue-recognition/actions/publish-revenue-accounting-profile` | finance.rev.publish.revenue.accounting.profile |
| FR-REV-004 | ModifyContract | 01, 02, 03, 05, 06, 09 | `revModifyContract` / `/api/v1/revenue-recognition/actions/modify-contract` | finance.rev.modify.contract |
| FR-REV-005 | ApplyContractModificationApprovalDecision | 01, 02, 03, 05, 06, 09 | `revApplyContractModificationApprovalDecision` / `/api/v1/revenue-recognition/actions/apply-contract-modification-approval-decision` | finance.rev.apply.contract.modification.approval.decision |
| FR-REV-006 | RunRecognition | 01, 02, 03, 05, 06, 09 | `revRunRecognition` / `/api/v1/revenue-recognition/actions/run-recognition` | finance.rev.run.recognition |
| FR-FA-001 | CapitalizeAsset | 01, 02, 03, 05, 06, 09 | `faCapitalizeAsset` / `/api/v1/fixed-assets/actions/capitalize-asset` | finance.fa.capitalize.asset |
| FR-FA-002 | CreateAssetAcquisitionClearing | 01, 02, 03, 05, 06, 09 | `faCreateAssetAcquisitionClearing` / `/api/v1/fixed-assets/actions/create-asset-acquisition-clearing` | finance.fa.create.asset.acquisition.clearing |
| FR-FA-003 | RunDepreciation | 01, 02, 03, 05, 06, 09 | `faRunDepreciation` / `/api/v1/fixed-assets/actions/run-depreciation` | finance.fa.run.depreciation |
| FR-FA-004 | ApplyImpairmentApprovalDecision | 01, 02, 03, 05, 06, 09 | `faApplyImpairmentApprovalDecision` / `/api/v1/fixed-assets/actions/apply-impairment-approval-decision` | finance.fa.apply.impairment.approval.decision |
| FR-FA-005 | DisposeAsset | 01, 02, 03, 05, 06, 09 | `faDisposeAsset` / `/api/v1/fixed-assets/actions/dispose-asset` | finance.fa.dispose.asset |
| FR-FA-006 | ApplyAssetDisposalApprovalDecision | 01, 02, 03, 05, 06, 09 | `faApplyAssetDisposalApprovalDecision` / `/api/v1/fixed-assets/actions/apply-asset-disposal-approval-decision` | finance.fa.apply.asset.disposal.approval.decision |
| FR-FA-007 | CancelUnpostedAssetDisposal | 01, 02, 03, 05, 06, 09 | `faCancelUnpostedAssetDisposal` / `/api/v1/fixed-assets/actions/cancel-unposted-asset-disposal` | finance.fa.cancel.unposted.asset.disposal |
| FR-FA-008 | CompensateFailedDisposalPosting | 01, 02, 03, 05, 06, 09 | `faCompensateFailedDisposalPosting` / `/api/v1/fixed-assets/actions/compensate-failed-disposal-posting` | finance.fa.compensate.failed.disposal.posting |
| FR-FA-009 | CreateDisposalSettlementClearing | 01, 02, 03, 05, 06, 09 | `faCreateDisposalSettlementClearing` / `/api/v1/fixed-assets/actions/create-disposal-settlement-clearing` | finance.fa.create.disposal.settlement.clearing |
| FR-FA-010 | ApplyAssetSupplierLiabilityResult | 01, 02, 03, 05, 06, 09 | `faApplyAssetSupplierLiabilityResult` / `/api/v1/fixed-assets/actions/apply-asset-supplier-liability-result` | finance.fa.apply.asset.supplier.liability.result |
| FR-FA-011 | ApplyIncomingSettlement | 01, 02, 03, 05, 06, 09 | `faApplyIncomingSettlement` / `/api/v1/fixed-assets/actions/apply-incoming-settlement` | finance.fa.apply.incoming.settlement |
| FR-FA-012 | ReverseIncomingSettlementApplication | 01, 02, 03, 05, 06, 09 | `faReverseIncomingSettlementApplication` / `/api/v1/fixed-assets/actions/reverse-incoming-settlement-application` | finance.fa.reverse.incoming.settlement.application |
| FR-FA-013 | ApplyPaymentReturn | 01, 02, 03, 05, 06, 09 | `faApplyPaymentReturn` / `/api/v1/fixed-assets/actions/apply-payment-return` | finance.fa.apply.payment.return |
| FR-FA-014 | ApplyAssetSettlementResult | 01, 02, 03, 05, 06, 09 | `faApplyAssetSettlementResult` / `/api/v1/fixed-assets/actions/apply-asset-settlement-result` | finance.fa.apply.asset.settlement.result |
| FR-FA-015 | ReclassifyDisposalCostForPayment | 01, 02, 03, 05, 06, 09 | `faReclassifyDisposalCostForPayment` / `/api/v1/fixed-assets/actions/reclassify-disposal-cost-for-payment` | finance.fa.reclassify.disposal.cost.for.payment |
| FR-FA-016 | RequestDisposalCostPayment | 01, 02, 03, 05, 06, 09 | `faRequestDisposalCostPayment` / `/api/v1/fixed-assets/actions/request-disposal-cost-payment` | finance.fa.request.disposal.cost.payment |
| FR-FA-017 | RequestDisposalCostPaymentReplacement | 01, 02, 03, 05, 06, 09 | `faRequestDisposalCostPaymentReplacement` / `/api/v1/fixed-assets/actions/request-disposal-cost-payment-replacement` | finance.fa.request.disposal.cost.payment.replacement |
| FR-FA-018 | Record impairment assessments | 01, 02, 03, 05, 06, 09 | `faRecordImpairmentAssessments` / `/api/v1/fixed-assets/actions/record-impairment-assessments` | finance.fa.record.impairment.assessments |
| FR-FA-019 | Transfer assets or components | 01, 02, 03, 05, 06, 09 | `faTransferAssetsOrComponents` / `/api/v1/fixed-assets/actions/transfer-assets-or-components` | finance.fa.transfer.assets.or.components |
| FR-FA-020 | Split assets or components | 01, 02, 03, 05, 06, 09 | `faSplitAssetsOrComponents` / `/api/v1/fixed-assets/actions/split-assets-or-components` | finance.fa.split.assets.or.components |
| FR-FA-021 | Correct posted asset disposals | 01, 02, 03, 05, 06, 09 | `faCorrectPostedAssetDisposals` / `/api/v1/fixed-assets/actions/correct-posted-asset-disposals` | finance.fa.correct.posted.asset.disposals |
| FR-FX-001 | PublishRateSet | 01, 02, 03, 05, 06, 09 | `fxPublishRateSet` / `/api/v1/multi-currency/actions/publish-rate-set` | finance.fx.publish.rate.set |
| FR-FX-002 | RunRevaluation | 01, 02, 03, 05, 06, 09 | `fxRunRevaluation` / `/api/v1/multi-currency/actions/run-revaluation` | finance.fx.run.revaluation |
| FR-FX-003 | ApplyRevaluationApprovalDecision | 01, 02, 03, 05, 06, 09 | `fxApplyRevaluationApprovalDecision` / `/api/v1/multi-currency/actions/apply-revaluation-approval-decision` | finance.fx.apply.revaluation.approval.decision |
| FR-FX-004 | PostRevaluationRun | 01, 02, 03, 05, 06, 09 | `fxPostRevaluationRun` / `/api/v1/multi-currency/actions/post-revaluation-run` | finance.fx.post.revaluation.run |
| FR-FX-005 | RunTranslation | 01, 02, 03, 05, 06, 09 | `fxRunTranslation` / `/api/v1/multi-currency/actions/run-translation` | finance.fx.run.translation |
| FR-FPM-001 | StartSoftClose | 01, 02, 03, 05, 06, 09 | `fpmStartSoftClose` / `/api/v1/fiscal-periods/actions/start-soft-close` | finance.fpm.start.soft.close |
| FR-FPM-002 | EndSoftClose | 01, 02, 03, 05, 06, 09 | `fpmEndSoftClose` / `/api/v1/fiscal-periods/actions/end-soft-close` | finance.fpm.end.soft.close |
| FR-FPM-003 | StartHardClose | 01, 02, 03, 05, 06, 09 | `fpmStartHardClose` / `/api/v1/fiscal-periods/actions/start-hard-close` | finance.fpm.start.hard.close |
| FR-FPM-004 | ResumeCloseRun | 01, 02, 03, 05, 06, 09 | `fpmResumeCloseRun` / `/api/v1/fiscal-periods/actions/resume-close-run` | finance.fpm.resume.close.run |
| FR-FPM-005 | AbortCloseRun | 01, 02, 03, 05, 06, 09 | `fpmAbortCloseRun` / `/api/v1/fiscal-periods/actions/abort-close-run` | finance.fpm.abort.close.run |
| FR-FPM-006 | ApplyPostingGateResult | 01, 02, 03, 05, 06, 09 | `fpmApplyPostingGateResult` / `/api/v1/fiscal-periods/actions/apply-posting-gate-result` | finance.fpm.apply.posting.gate.result |
| FR-FPM-007 | ApplyCloseExceptionApprovalDecision | 01, 02, 03, 05, 06, 09 | `fpmApplyCloseExceptionApprovalDecision` / `/api/v1/fiscal-periods/actions/apply-close-exception-approval-decision` | finance.fpm.apply.close.exception.approval.decision |
| FR-FPM-008 | ApplyCloseApprovalDecision | 01, 02, 03, 05, 06, 09 | `fpmApplyCloseApprovalDecision` / `/api/v1/fiscal-periods/actions/apply-close-approval-decision` | finance.fpm.apply.close.approval.decision |
| FR-FPM-009 | RequestReopen | 01, 02, 03, 05, 06, 09 | `fpmRequestReopen` / `/api/v1/fiscal-periods/actions/request-reopen` | finance.fpm.request.reopen |
| FR-FPM-010 | ApplyReopenApprovalDecision | 01, 02, 03, 05, 06, 09 | `fpmApplyReopenApprovalDecision` / `/api/v1/fiscal-periods/actions/apply-reopen-approval-decision` | finance.fpm.apply.reopen.approval.decision |
| FR-FPM-011 | StartReclose | 01, 02, 03, 05, 06, 09 | `fpmStartReclose` / `/api/v1/fiscal-periods/actions/start-reclose` | finance.fpm.start.reclose |
| FR-FPM-012 | TakeOverPeriodControl | 01, 02, 03, 05, 06, 09 | `fpmTakeOverPeriodControl` / `/api/v1/fiscal-periods/actions/take-over-period-control` | finance.fpm.take.over.period.control |
| FR-FPM-013 | ExtendCloseException | 01, 02, 03, 05, 06, 09 | `fpmExtendCloseException` / `/api/v1/fiscal-periods/actions/extend-close-exception` | finance.fpm.extend.close.exception |
| FR-COA-001 | Maintain segment definitions | 01, 02, 03, 05, 06, 09 | `coaMaintainSegmentDefinitions` / `/api/v1/coa-segments/configuration/maintain-segment-definitions` | finance.coa.maintain.segment.definitions |
| FR-COA-002 | Maintain segment values | 01, 02, 03, 05, 06, 09 | `coaMaintainSegmentValues` / `/api/v1/coa-segments/configuration/maintain-segment-values` | finance.coa.maintain.segment.values |
| FR-COA-003 | Validate segment combinations | 01, 02, 03, 05, 06, 09 | `coaValidateSegmentCombinations` / `/api/v1/coa-segments/actions/validate-segment-combinations` | finance.coa.validate.segment.combinations |
| FR-COA-004 | Request segment changes | 01, 02, 03, 05, 06, 09 | `coaRequestSegmentChanges` / `/api/v1/coa-segments/actions/request-segment-changes` | finance.coa.request.segment.changes |
| FR-COA-005 | ApplySegmentChangeApprovalDecision | 01, 02, 03, 05, 06, 09 | `coaApplySegmentChangeApprovalDecision` / `/api/v1/coa-segments/actions/apply-segment-change-approval-decision` | finance.coa.apply.segment.change.approval.decision |
| FR-BFR-001 | ImportStatement | 01, 02, 03, 05, 06, 09 | `bfrImportStatement` / `/api/v1/bank-reconciliation/actions/import-statement` | finance.bfr.import.statement |
| FR-BFR-002 | ProposeMatch | 01, 02, 03, 05, 06, 09 | `bfrProposeMatch` / `/api/v1/bank-reconciliation/actions/propose-match` | finance.bfr.propose.match |
| FR-BFR-003 | ConfirmMatch | 01, 02, 03, 05, 06, 09 | `bfrConfirmMatch` / `/api/v1/bank-reconciliation/actions/confirm-match` | finance.bfr.confirm.match |
| FR-BFR-004 | Unmatch | 01, 02, 03, 05, 06, 09 | `bfrUnmatch` / `/api/v1/bank-reconciliation/actions/unmatch` | finance.bfr.unmatch |
| FR-BFR-005 | CompleteReconciliation | 01, 02, 03, 05, 06, 09 | `bfrCompleteReconciliation` / `/api/v1/bank-reconciliation/actions/complete-reconciliation` | finance.bfr.complete.reconciliation |
| FR-BFR-006 | Maintain bank-feed connections | 01, 02, 03, 05, 06, 09 | `bfrMaintainBankFeedConnections` / `/api/v1/bank-reconciliation/configuration/maintain-bank-feed-connections` | finance.bfr.maintain.bank.feed.connections |
| FR-TAX-001 | DetermineTax | 01, 02, 03, 05, 06, 09 | `taxDetermineTax` / `/api/v1/tax/actions/determine-tax` | finance.tax.determine.tax |
| FR-TAX-002 | PrepareTaxReturn | 01, 02, 03, 05, 06, 09 | `taxPrepareTaxReturn` / `/api/v1/tax/actions/prepare-tax-return` | finance.tax.prepare.tax.return |
| FR-TAX-003 | ApplyTaxReturnApprovalDecision | 01, 02, 03, 05, 06, 09 | `taxApplyTaxReturnApprovalDecision` / `/api/v1/tax/actions/apply-tax-return-approval-decision` | finance.tax.apply.tax.return.approval.decision |
| FR-TAX-004 | SubmitTaxReturn | 01, 02, 03, 05, 06, 09 | `taxSubmitTaxReturn` / `/api/v1/tax/actions/submit-tax-return` | finance.tax.submit.tax.return |
| FR-TAX-005 | CreateTaxAmendment | 01, 02, 03, 05, 06, 09 | `taxCreateTaxAmendment` / `/api/v1/tax/actions/create-tax-amendment` | finance.tax.create.tax.amendment |
| FR-TAX-006 | ApplyTaxAmendmentApprovalDecision | 01, 02, 03, 05, 06, 09 | `taxApplyTaxAmendmentApprovalDecision` / `/api/v1/tax/actions/apply-tax-amendment-approval-decision` | finance.tax.apply.tax.amendment.approval.decision |
| FR-TAX-007 | SubmitTaxAmendment | 01, 02, 03, 05, 06, 09 | `taxSubmitTaxAmendment` / `/api/v1/tax/actions/submit-tax-amendment` | finance.tax.submit.tax.amendment |
| FR-TAX-008 | CreateReturnLevelTaxAdjustment | 01, 02, 03, 05, 06, 09 | `taxCreateReturnLevelTaxAdjustment` / `/api/v1/tax/actions/create-return-level-tax-adjustment` | finance.tax.create.return.level.tax.adjustment |
| FR-TAX-009 | ApplyReturnLevelTaxAdjustmentApprovalDecision | 01, 02, 03, 05, 06, 09 | `taxApplyReturnLevelTaxAdjustmentApprovalDecision` / `/api/v1/tax/actions/apply-return-level-tax-adjustment-approval-decision` | finance.tax.apply.return.level.tax.adjustment.approval.decision |
| FR-TAX-010 | PostReturnLevelTaxAdjustment | 01, 02, 03, 05, 06, 09 | `taxPostReturnLevelTaxAdjustment` / `/api/v1/tax/actions/post-return-level-tax-adjustment` | finance.tax.post.return.level.tax.adjustment |
| FR-TAX-011 | RequestTaxPayment | 01, 02, 03, 05, 06, 09 | `taxRequestTaxPayment` / `/api/v1/tax/actions/request-tax-payment` | finance.tax.request.tax.payment |
| FR-TAX-012 | RecordTaxPaymentSettlement | 01, 02, 03, 05, 06, 09 | `taxRecordTaxPaymentSettlement` / `/api/v1/tax/actions/record-tax-payment-settlement` | finance.tax.record.tax.payment.settlement |
| FR-TAX-013 | ApplyIncomingSettlement | 01, 02, 03, 05, 06, 09 | `taxApplyIncomingSettlement` / `/api/v1/tax/actions/apply-incoming-settlement` | finance.tax.apply.incoming.settlement |
| FR-TAX-014 | ReverseIncomingSettlementApplication | 01, 02, 03, 05, 06, 09 | `taxReverseIncomingSettlementApplication` / `/api/v1/tax/actions/reverse-incoming-settlement-application` | finance.tax.reverse.incoming.settlement.application |
| FR-TAX-015 | ApplyPaymentReturn | 01, 02, 03, 05, 06, 09 | `taxApplyPaymentReturn` / `/api/v1/tax/actions/apply-payment-return` | finance.tax.apply.payment.return |
| FR-TAX-016 | Maintain tax configurations | 01, 02, 03, 05, 06, 09 | `taxMaintainTaxConfigurations` / `/api/v1/tax/configuration/maintain-tax-configurations` | finance.tax.maintain.tax.configurations |
| FR-WFA-001 | CreateApprovalRequest | 01, 02, 03, 05, 06, 09 | `wfaCreateApprovalRequest` / `/api/v1/approvals/actions/create-approval-request` | finance.wfa.create.approval.request |
| FR-WFA-002 | DecideApprovalRequest | 01, 02, 03, 05, 06, 09 | `wfaDecideApprovalRequest` / `/api/v1/approvals/actions/decide-approval-request` | finance.wfa.decide.approval.request |
| FR-WFA-003 | DelegateApproval | 01, 02, 03, 05, 06, 09 | `wfaDelegateApproval` / `/api/v1/approvals/actions/delegate-approval` | finance.wfa.delegate.approval |
| FR-WFA-004 | EscalateApproval | 01, 02, 03, 05, 06, 09 | `wfaEscalateApproval` / `/api/v1/approvals/actions/escalate-approval` | finance.wfa.escalate.approval |
| FR-WFA-005 | Maintain approval policies | 01, 02, 03, 05, 06, 09 | `wfaMaintainApprovalPolicies` / `/api/v1/approvals/configuration/maintain-approval-policies` | finance.wfa.maintain.approval.policies |
| FR-IAM-001 | Manage users | 01, 02, 03, 05, 06, 09 | `iamManageUsers` / `/api/v1/identity-access/actions/manage-users` | finance.iam.manage.users |
| FR-IAM-002 | Manage roles | 01, 02, 03, 05, 06, 09 | `iamManageRoles` / `/api/v1/identity-access/actions/manage-roles` | finance.iam.manage.roles |
| FR-IAM-003 | Manage access policies | 01, 02, 03, 05, 06, 09 | `iamManageAccessPolicies` / `/api/v1/identity-access/actions/manage-access-policies` | finance.iam.manage.access.policies |
| FR-IAM-004 | Manage segregation rules | 01, 02, 03, 05, 06, 09 | `iamManageSegregationRules` / `/api/v1/identity-access/actions/manage-segregation-rules` | finance.iam.manage.segregation.rules |
| FR-IAM-005 | Grant emergency access | 01, 02, 03, 05, 06, 09 | `iamGrantEmergencyAccess` / `/api/v1/identity-access/actions/grant-emergency-access` | finance.iam.grant.emergency.access |
| FR-IAM-006 | Revoke emergency access | 01, 02, 03, 05, 06, 09 | `iamRevokeEmergencyAccess` / `/api/v1/identity-access/actions/revoke-emergency-access` | finance.iam.revoke.emergency.access |
| FR-AUD-001 | AppendAuditableEvent | 01, 02, 03, 05, 06, 09 | `audAppendAuditableEvent` / `/api/v1/audit-integrity/actions/append-auditable-event` | finance.aud.append.auditable.event |
| FR-AUD-002 | CreateAuditSeal | 01, 02, 03, 05, 06, 09 | `audCreateAuditSeal` / `/api/v1/audit-integrity/actions/create-audit-seal` | finance.aud.create.audit.seal |
| FR-AUD-003 | RotateVerificationCredential | 01, 02, 03, 05, 06, 09 | `audRotateVerificationCredential` / `/api/v1/audit-integrity/actions/rotate-verification-credential` | finance.aud.rotate.verification.credential |
| FR-AUD-004 | EscalateIntegrityIncident | 01, 02, 03, 05, 06, 09 | `audEscalateIntegrityIncident` / `/api/v1/audit-integrity/actions/escalate-integrity-incident` | finance.aud.escalate.integrity.incident |
| FR-AUD-005 | VerifyProof | 01, 02, 03, 05, 06, 09 | `audVerifyProof` / `/api/v1/audit-integrity/reference/verify-proof` | finance.aud.verify.proof |

## 5. Workflow traceability

| Workflow | Title | Technical documents | Verification |
|---|---|---|---|
| WF-6.1 | Period Close: Hard Close | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-6.2 | Fiscal Period Reopen and Reclose | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-6.3 | Intercompany Reconciliation and Settlement | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-6.4 | Fixed Asset Disposal with Gain or Loss Recognition | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-6.5 | Revenue Recognition for a SaaS Contract | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-6.6 | Journal Entry Posting and Reversal | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-6.7 | Customer Receipt Recording with Partial Application | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.1 | Vendor Invoice Registration, Matching, Approval, Dispute, and Void | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.2 | Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.3 | Customer Credit, Refund, Overpayment, Chargeback, and Write-Off | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.4 | Bank Statement Import, Matching, Unmatching, and Reconciliation | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.5 | Foreign-Currency Invoice Settlement and Realized FX | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.6 | Period-End Revaluation, Rerun, and Next-Period Reversal | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.7 | Full Fixed-Asset Lifecycle and Disposal Variants | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.8 | Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.9 | Consolidation, Ownership Changes, Translation, Eliminations, and Rerun | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.10 | Tax Return Submission, Rejection, Amendment, Payment, and Evidence | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.11 | Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.12 | Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.13 | Cross-Context Event Interpretation, Ordering, and Replay | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.14 | Concurrent Aggregate and Domain-Process Modification Rules | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |
| WF-7.15 | Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation | 01, 02, 03, 04, 05, 06, 09 | Playwright + module/API/database/integration tests |

## 6. Nonfunctional traceability

| Requirement | Technical documents | Verification |
|---|---|---|
| NFR-ACC-001 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-002 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-003 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-004 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-005 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-006 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-007 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-008 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-009 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-010 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-011 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-ACC-012 | 05, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-001 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-002 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-003 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-004 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-005 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-006 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-007 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-008 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-009 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-010 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-011 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AUD-012 | 03, 04, 06, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-001 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-002 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-003 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-004 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-005 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-006 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-007 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-008 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-009 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-AVL-010 | 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-001 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-002 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-003 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-004 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-005 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-006 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-007 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-008 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-009 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CAP-010 | 03, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-001 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-002 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-003 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-004 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-005 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-006 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-007 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-CMP-008 | 02, 05, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-001 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-002 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-003 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-004 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-005 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-006 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-007 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-008 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-009 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-INT-010 | 02, 04, 09 | Verified by the NFR-specific tests and controls |
| NFR-LOC-001 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-002 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-003 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-004 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-005 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-006 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-007 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-LOC-008 | 02, 05 | Verified by the NFR-specific tests and controls |
| NFR-MNT-001 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-002 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-003 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-004 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-005 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-006 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-007 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-008 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-009 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-010 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-011 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-MNT-012 | 01, 03, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-001 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-002 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-003 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-004 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-005 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-006 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-007 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-008 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-009 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-010 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-011 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-OBS-012 | 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-001 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-002 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-003 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-004 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-005 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-006 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-007 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-008 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-009 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-010 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-011 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-012 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-013 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PERF-014 | 03, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-PRV-001 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-002 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-003 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-004 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-005 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-006 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-007 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-008 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-009 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-PRV-010 | 05, 06, 08 | Verified by the NFR-specific tests and controls |
| NFR-REC-001 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-002 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-003 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-004 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-005 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-006 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-007 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-008 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-009 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-010 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-011 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REC-012 | 03, 04, 07, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-001 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-002 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-003 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-004 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-005 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-006 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-007 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-008 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-009 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-010 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-011 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-012 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-013 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-014 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-015 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-REL-016 | 01, 03, 04, 08, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-001 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-002 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-003 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-004 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-005 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-006 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-007 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-008 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-009 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-010 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-011 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-012 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-013 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-014 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-015 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-016 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-017 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-SEC-018 | 06, 07, 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-001 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-002 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-003 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-004 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-005 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-006 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-007 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-008 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-009 | 09 | Verified by the NFR-specific tests and controls |
| NFR-TST-010 | 09 | Verified by the NFR-specific tests and controls |

## 7. Functional acceptance traceability

| Acceptance ID | Technical document | Treatment |
|---|---|---|
| FAC-14-1-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-1-09 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-10-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-11-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-11-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-11-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-11-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-11-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-11-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-12-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-12-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-12-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-12-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-12-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-12-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-1-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-1-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-1-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-1-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-1-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-10-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-10-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-10-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-10-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-10-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-11-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-11-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-11-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-11-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-11-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-12-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-12-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-12-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-12-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-12-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-13-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-13-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-13-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-13-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-13-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-14-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-14-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-14-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-14-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-14-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-15-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-15-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-15-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-15-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-15-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-09 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-10 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-11 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-12 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-2-13 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-3-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-3-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-3-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-3-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-3-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-4-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-4-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-4-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-4-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-4-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-5-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-5-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-5-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-5-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-5-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-6-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-6-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-6-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-6-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-6-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-7-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-7-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-7-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-7-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-7-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-8-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-8-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-8-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-8-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-8-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-9-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-9-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-9-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-9-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-13-9-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-09 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-10 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-11 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-12 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-13 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-14 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-15 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-16 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-17 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-18 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-19 | 09 | Functional acceptance retained as source test intent |
| FAC-14-2-20 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-09 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-10 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-11 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-12 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-13 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-14 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-15 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-16 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-17 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-18 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-19 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-20 | 09 | Functional acceptance retained as source test intent |
| FAC-14-3-21 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-4-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-5-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-5-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-5-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-5-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-07 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-08 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-09 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-10 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-11 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-12 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-13 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-14 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-15 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-16 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-17 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-18 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-19 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-20 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-21 | 09 | Functional acceptance retained as source test intent |
| FAC-14-6-22 | 09 | Functional acceptance retained as source test intent |
| FAC-14-7-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-7-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-7-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-7-04 | 09 | Functional acceptance retained as source test intent |
| FAC-14-7-05 | 09 | Functional acceptance retained as source test intent |
| FAC-14-7-06 | 09 | Functional acceptance retained as source test intent |
| FAC-14-8-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-8-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-8-03 | 09 | Functional acceptance retained as source test intent |
| FAC-14-9-01 | 09 | Functional acceptance retained as source test intent |
| FAC-14-9-02 | 09 | Functional acceptance retained as source test intent |
| FAC-14-9-03 | 09 | Functional acceptance retained as source test intent |

## 8. Architecture-control and ADR inheritance

All solution controls `ARC-ACC-001, ARC-API-001, ARC-AUD-001, ARC-CAP-001, ARC-CICD-001, ARC-CON-001, ARC-DATA-001, ARC-DATA-002, ARC-DATA-003, ARC-DATA-004, ARC-DOM-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003, ARC-IAC-001, ARC-IDEM-001, ARC-MOD-001, ARC-OBS-001, ARC-OBS-002, ARC-PERF-001, ARC-PRV-001, ARC-REC-001, ARC-REL-001, ARC-REL-002, ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-TST-001, ARC-TXN-001, ARC-TXN-002` and solution decisions `ADR-001, ADR-002, ADR-003, ADR-004, ADR-005, ADR-006, ADR-007, ADR-008, ADR-009, ADR-010, ADR-011, ADR-012, ADR-013, ADR-014, ADR-015, ADR-016, ADR-017, ADR-018, ADR-019, ADR-020` remain binding. A technical decision may refine but may not contradict them without updating Solution/System Design v1.0.

## 9. Technical ADRs

| ID | Decision | Status | Rationale |
|---|---|---|---|
| TADR-001 | Use UUID v7 for application-created identities | Accepted | Time-sortable stable identifiers without database sequences across modules. |
| TADR-002 | Use numeric(38,12) plus exact-decimal Go abstraction | Accepted | Prevents binary floating-point accounting errors; policy constrains scale. |
| TADR-003 | Use command-oriented action endpoints | Accepted | Maps explicit DDD/PRD operations without destructive CRUD semantics. |
| TADR-004 | Use RFC 9457 problem details | Accepted | Stable machine/user error contract. |
| TADR-005 | Use PostgreSQL row version plus deterministic locks | Accepted | Supports optimistic conflicts and high-integrity multi-aggregate rules. |
| TADR-006 | Use PostgreSQL outbox/inbox and lease fencing | Accepted | Durable asynchronous semantics without a broker initially. |
| TADR-007 | Use schema-per-context roles | Accepted | Enforces modular-monolith data ownership. |
| TADR-008 | Use generated OpenAPI TypeScript types | Accepted | Reduces client/server contract drift. |
| TADR-009 | Wrap daisyUI in semantic application components | Accepted | Centralizes accessibility and finance interaction behavior. |
| TADR-010 | Use Azure Blob remote state and GitHub OIDC | Accepted | Secures Terraform state and avoids long-lived deployment credentials. |
| TADR-011 | Use separate API, worker and reporting pools | Accepted | Prevents background/read workloads from starving financial writes. |
| TADR-012 | Keep audit evidence separate from operational logs | Accepted | Preserves audit authority and retention semantics. |

## 10. Change-control rules

- An API path/schema change updates OpenAPI, generated types, tests and traceability in one change.
- A table/constraint/index change updates migration, sqlc queries, recovery/performance evidence and traceability.
- A domain command/event/aggregate/state change starts in DDD, then propagates through PRD, UX, NFR impact review, Solution Design and these specifications.
- A technology major-version change requires compatibility, security, migration and rollback evidence.
- No implementation divergence is accepted as an undocumented exception.

## 11. Verification summary

| Check | Expected |
|---|---|
| Modules | 19 modules and schemas |
| Aggregate roots | 72 mapped |
| Functional requirements | 193 unique and traced |
| Global requirements | 22 unique and traced |
| Workflows | 22 unique and traced |
| NFRs | 174 unique and traced |
| Acceptance scenarios | 199 retained by reference |
| Representative commands | 148 cataloged |
| Representative events | 217 cataloged |
| Open defects | None at baseline publication |

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `85e012afff6a16b69be98002d54129fa50164e4f05a98f01ac881fdbb1630c85` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
