# Finance Platform Security, Identity and Authorization Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready security baseline |
| Identity provider | Microsoft Entra ID |

## 1. Authentication flow

- SPA uses authorization-code flow with PKCE.
- API validates issuer, audience, signature, algorithm, expiry and not-before using Entra discovery/JWKS.
- Accepted clock skew is 120 seconds.
- Local development uses a clearly marked local issuer and fixture identities; it is disabled in Azure environments.
- Service workloads use managed identity or GitHub OIDC federation; long-lived Azure client secrets are prohibited.

## 2. Required token and actor fields

| Field | Source | Use |
|---|---|---|
| `oid` | Entra token | Stable authentication subject |
| `tid` | Entra token | Tenant validation |
| `sub` | Entra token | Protocol subject |
| `scp` / `roles` | Entra token | API audience/scope admission only |
| Application user ID | Identity module | Finance actor and audit identity |
| Available accounting scopes | Identity module | Candidate scopes; operation authorization still evaluates policy |

## 3. Authorization decision

```go
type DecisionInput struct {
    ActorID uuid.UUID
    Permission string
    AccountingScopeID *uuid.UUID
    LegalEntityID *uuid.UUID
    SegmentIDs []uuid.UUID
    Amount *decimal.Decimal
    Currency *string
    FiscalPeriodID *uuid.UUID
    DataClassification string
    SubjectActorIDs []uuid.UUID
}
```

The Identity & Access module evaluates permissions, access policies, amount/segment/scope constraints, segregation rules and emergency grants. Deny is the default. The decision reference and policy version are stored with material actions.

## 4. Permission catalog

| Requirement | Permission | Module | Operation | Scope dimensions |
|---|---|---|---|---|
| FR-OMD-001 | finance.omd.maintain.legal.entities | OMD | Maintain legal entities | Accounting scope plus operation-specific dimensions |
| FR-OMD-002 | finance.omd.maintain.parties | OMD | Maintain parties | Accounting scope plus operation-specific dimensions |
| FR-OMD-003 | finance.omd.maintain.customer.profiles | OMD | Maintain customer profiles | Accounting scope plus operation-specific dimensions |
| FR-OMD-004 | finance.omd.maintain.vendor.profiles | OMD | Maintain vendor profiles | Accounting scope plus operation-specific dimensions |
| FR-OMD-005 | finance.omd.maintain.fiscal.calendars | OMD | Maintain fiscal calendars | Accounting scope plus operation-specific dimensions |
| FR-OMD-006 | finance.omd.publish.approved.master.data.changes | OMD | Publish approved master-data changes | Accounting scope plus operation-specific dimensions |
| FR-GL-001 | finance.gl.submit.posting.request | GL | SubmitPostingRequest | Accounting scope plus operation-specific dimensions |
| FR-GL-002 | finance.gl.apply.journal.approval.decision | GL | ApplyJournalApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-GL-003 | finance.gl.reverse.journal.entry | GL | ReverseJournalEntry | Accounting scope plus operation-specific dimensions |
| FR-GL-004 | finance.gl.enter.soft.close.gate | GL | EnterSoftCloseGate | Accounting scope plus operation-specific dimensions |
| FR-GL-005 | finance.gl.exit.soft.close.gate | GL | ExitSoftCloseGate | Accounting scope plus operation-specific dimensions |
| FR-GL-006 | finance.gl.acquire.posting.barrier | GL | AcquirePostingBarrier | Accounting scope plus operation-specific dimensions |
| FR-GL-007 | finance.gl.release.posting.barrier | GL | ReleasePostingBarrier | Accounting scope plus operation-specific dimensions |
| FR-GL-008 | finance.gl.finalize.posting.gate | GL | FinalizePostingGate | Accounting scope plus operation-specific dimensions |
| FR-GL-009 | finance.gl.open.scoped.reopen.gate | GL | OpenScopedReopenGate | Accounting scope plus operation-specific dimensions |
| FR-GL-010 | finance.gl.close.scoped.reopen.gate | GL | CloseScopedReopenGate | Accounting scope plus operation-specific dimensions |
| FR-GL-011 | finance.gl.open.operational.reopen.gate | GL | OpenOperationalReopenGate | Accounting scope plus operation-specific dimensions |
| FR-GL-012 | finance.gl.close.operational.reopen.gate | GL | CloseOperationalReopenGate | Accounting scope plus operation-specific dimensions |
| FR-GL-013 | finance.gl.begin.reclose.gate | GL | BeginRecloseGate | Accounting scope plus operation-specific dimensions |
| FR-GL-014 | finance.gl.get.posting.gate.status | GL | GetPostingGateStatus | Accounting scope plus operation-specific dimensions |
| FR-GL-015 | finance.gl.maintain.ledgers | GL | Maintain ledgers | Accounting scope plus operation-specific dimensions |
| FR-GL-016 | finance.gl.maintain.accounting.books | GL | Maintain accounting books | Accounting scope plus operation-specific dimensions |
| FR-GL-017 | finance.gl.maintain.charts.of.accounts | GL | Maintain charts of accounts | Accounting scope plus operation-specific dimensions |
| FR-GL-018 | finance.gl.maintain.accounts.and.reporting.mappings | GL | Maintain accounts and reporting mappings | Accounting scope plus operation-specific dimensions |
| FR-AP-001 | finance.ap.register.vendor.invoice | AP | RegisterVendorInvoice | Accounting scope plus operation-specific dimensions |
| FR-AP-002 | finance.ap.apply.asset.clearing.classification | AP | ApplyAssetClearingClassification | Accounting scope plus operation-specific dimensions |
| FR-AP-003 | finance.ap.apply.incoming.settlement | AP | ApplyIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-AP-004 | finance.ap.reverse.incoming.settlement.application | AP | ReverseIncomingSettlementApplication | Accounting scope plus operation-specific dimensions |
| FR-AP-005 | finance.ap.apply.payment.return | AP | ApplyPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-AP-006 | finance.ap.apply.vendor.invoice.approval.decision | AP | ApplyVendorInvoiceApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-AP-007 | finance.ap.request.payment | AP | RequestPayment | Accounting scope plus operation-specific dimensions |
| FR-AP-008 | finance.ap.validate.vendor.invoice | AP | ValidateVendorInvoice | Accounting scope plus operation-specific dimensions |
| FR-AP-009 | finance.ap.dispute.vendor.invoice | AP | DisputeVendorInvoice | Accounting scope plus operation-specific dimensions |
| FR-AP-010 | finance.ap.void.vendor.invoice | AP | VoidVendorInvoice | Accounting scope plus operation-specific dimensions |
| FR-AR-001 | finance.ar.issue.customer.invoice | AR | IssueCustomerInvoice | Accounting scope plus operation-specific dimensions |
| FR-AR-002 | finance.ar.record.receipt | AR | RecordReceipt | Accounting scope plus operation-specific dimensions |
| FR-AR-003 | finance.ar.apply.receipt | AR | ApplyReceipt | Accounting scope plus operation-specific dimensions |
| FR-AR-004 | finance.ar.unapply.receipt | AR | UnapplyReceipt | Accounting scope plus operation-specific dimensions |
| FR-AR-005 | finance.ar.rollback.unposted.application.batch | AR | RollbackUnpostedApplicationBatch | Accounting scope plus operation-specific dimensions |
| FR-AR-006 | finance.ar.issue.credit.note | AR | IssueCreditNote | Accounting scope plus operation-specific dimensions |
| FR-AR-007 | finance.ar.create.customer.refund.request | AR | CreateCustomerRefundRequest | Accounting scope plus operation-specific dimensions |
| FR-AR-008 | finance.ar.cancel.customer.refund.request | AR | CancelCustomerRefundRequest | Accounting scope plus operation-specific dimensions |
| FR-AR-009 | finance.ar.apply.customer.refund.approval.decision | AR | ApplyCustomerRefundApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-AR-010 | finance.ar.request.customer.refund.payment | AR | RequestCustomerRefundPayment | Accounting scope plus operation-specific dimensions |
| FR-AR-011 | finance.ar.cancel.customer.refund.payment | AR | CancelCustomerRefundPayment | Accounting scope plus operation-specific dimensions |
| FR-AR-012 | finance.ar.apply.customer.refund.payment.result | AR | ApplyCustomerRefundPaymentResult | Accounting scope plus operation-specific dimensions |
| FR-AR-013 | finance.ar.apply.payment.return | AR | ApplyPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-AR-014 | finance.ar.resolve.customer.overpayments | AR | Resolve customer overpayments | Accounting scope plus operation-specific dimensions |
| FR-AR-015 | finance.ar.record.customer.chargebacks | AR | Record customer chargebacks | Accounting scope plus operation-specific dimensions |
| FR-AR-016 | finance.ar.record.receivable.write.offs | AR | Record receivable write-offs | Accounting scope plus operation-specific dimensions |
| FR-PAYR-001 | finance.payr.calculate.payroll.run | PAYR | CalculatePayrollRun | Accounting scope plus operation-specific dimensions |
| FR-PAYR-002 | finance.payr.apply.payroll.run.approval.decision | PAYR | ApplyPayrollRunApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-PAYR-003 | finance.payr.post.payroll.run | PAYR | PostPayrollRun | Accounting scope plus operation-specific dimensions |
| FR-PAYR-004 | finance.payr.create.payroll.correction | PAYR | CreatePayrollCorrection | Accounting scope plus operation-specific dimensions |
| FR-PAYR-005 | finance.payr.apply.payment.return | PAYR | ApplyPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-PAYR-006 | finance.payr.maintain.employee.payroll.profiles | PAYR | Maintain employee payroll profiles | Accounting scope plus operation-specific dimensions |
| FR-PAYR-007 | finance.payr.maintain.payroll.tax.filing.records | PAYR | Maintain payroll tax-filing records | Accounting scope plus operation-specific dimensions |
| FR-INV-001 | finance.inv.configure.invoice.templates | INV | Configure invoice templates | Accounting scope plus operation-specific dimensions |
| FR-INV-002 | finance.inv.configure.billing.schedules | INV | Configure billing schedules | Accounting scope plus operation-specific dimensions |
| FR-INV-003 | finance.inv.generate.invoices | INV | Generate invoices | Accounting scope plus operation-specific dimensions |
| FR-INV-004 | finance.inv.finalize.generated.invoices | INV | Finalize generated invoices | Accounting scope plus operation-specific dimensions |
| FR-INV-005 | finance.inv.recalculate.unfinalized.invoices | INV | Recalculate unfinalized invoices | Accounting scope plus operation-specific dimensions |
| FR-INV-006 | finance.inv.cancel.unfinalized.invoices | INV | Cancel unfinalized invoices | Accounting scope plus operation-specific dimensions |
| FR-PCM-001 | finance.pcm.prepare.payment.batch | PCM | PreparePaymentBatch | Accounting scope plus operation-specific dimensions |
| FR-PCM-002 | finance.pcm.apply.payment.batch.approval.decision | PCM | ApplyPaymentBatchApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-PCM-003 | finance.pcm.cancel.payment.batch | PCM | CancelPaymentBatch | Accounting scope plus operation-specific dimensions |
| FR-PCM-004 | finance.pcm.register.expected.incoming.settlement | PCM | RegisterExpectedIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-005 | finance.pcm.resolve.expected.incoming.settlement.exception | PCM | ResolveExpectedIncomingSettlementException | Accounting scope plus operation-specific dimensions |
| FR-PCM-006 | finance.pcm.cancel.expected.incoming.settlement | PCM | CancelExpectedIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-007 | finance.pcm.close.expected.incoming.settlement | PCM | CloseExpectedIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-008 | finance.pcm.create.payment.instruction.from.obligation | PCM | CreatePaymentInstructionFromObligation | Accounting scope plus operation-specific dimensions |
| FR-PCM-009 | finance.pcm.submit.payment.instruction | PCM | SubmitPaymentInstruction | Accounting scope plus operation-specific dimensions |
| FR-PCM-010 | finance.pcm.retry.payment.instruction | PCM | RetryPaymentInstruction | Accounting scope plus operation-specific dimensions |
| FR-PCM-011 | finance.pcm.cancel.payment.instruction | PCM | CancelPaymentInstruction | Accounting scope plus operation-specific dimensions |
| FR-PCM-012 | finance.pcm.apply.payment.instruction.exception.decision | PCM | ApplyPaymentInstructionExceptionDecision | Accounting scope plus operation-specific dimensions |
| FR-PCM-013 | finance.pcm.record.payment.return | PCM | RecordPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-PCM-014 | finance.pcm.cancel.unposted.payment.return | PCM | CancelUnpostedPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-PCM-015 | finance.pcm.acknowledge.payment.return | PCM | AcknowledgePaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-PCM-016 | finance.pcm.resolve.payment.return.exception | PCM | ResolvePaymentReturnException | Accounting scope plus operation-specific dimensions |
| FR-PCM-017 | finance.pcm.record.unallocated.incoming.settlement | PCM | RecordUnallocatedIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-018 | finance.pcm.resolve.unallocated.incoming.settlement | PCM | ResolveUnallocatedIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-019 | finance.pcm.record.incoming.settlement | PCM | RecordIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-020 | finance.pcm.resolve.settlement.receipt.validation.exception | PCM | ResolveSettlementReceiptValidationException | Accounting scope plus operation-specific dimensions |
| FR-PCM-021 | finance.pcm.resolve.incoming.settlement.owner.exception | PCM | ResolveIncomingSettlementOwnerException | Accounting scope plus operation-specific dimensions |
| FR-PCM-022 | finance.pcm.cancel.unposted.settlement.receipt | PCM | CancelUnpostedSettlementReceipt | Accounting scope plus operation-specific dimensions |
| FR-PCM-023 | finance.pcm.acknowledge.incoming.settlement | PCM | AcknowledgeIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-024 | finance.pcm.reverse.incoming.settlement | PCM | ReverseIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-PCM-025 | finance.pcm.maintain.bank.accounts | PCM | Maintain bank accounts | Accounting scope plus operation-specific dimensions |
| FR-RPT-001 | finance.rpt.run.consolidation | RPT | RunConsolidation | Accounting scope plus operation-specific dimensions |
| FR-RPT-002 | finance.rpt.apply.translation.result | RPT | ApplyTranslationResult | Accounting scope plus operation-specific dimensions |
| FR-RPT-003 | finance.rpt.apply.consolidation.approval.decision | RPT | ApplyConsolidationApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-RPT-004 | finance.rpt.publish.consolidated.statement | RPT | PublishConsolidatedStatement | Accounting scope plus operation-specific dimensions |
| FR-RPT-005 | finance.rpt.maintain.report.definitions | RPT | Maintain report definitions | Accounting scope plus operation-specific dimensions |
| FR-RPT-006 | finance.rpt.generate.and.publish.ledger.financial.statements | RPT | Generate and publish ledger financial statements | Accounting scope plus operation-specific dimensions |
| FR-IC-001 | finance.ic.start.settlement | IC | StartSettlement | Accounting scope plus operation-specific dimensions |
| FR-IC-002 | finance.ic.match.intercompany.items | IC | MatchIntercompanyItems | Accounting scope plus operation-specific dimensions |
| FR-IC-003 | finance.ic.apply.residual.approval.decision | IC | ApplyResidualApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-IC-004 | finance.ic.create.settlement.instructions | IC | CreateSettlementInstructions | Accounting scope plus operation-specific dimensions |
| FR-IC-005 | finance.ic.complete.settlement.run | IC | CompleteSettlementRun | Accounting scope plus operation-specific dimensions |
| FR-IC-006 | finance.ic.apply.incoming.settlement | IC | ApplyIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-IC-007 | finance.ic.reverse.incoming.settlement.application | IC | ReverseIncomingSettlementApplication | Accounting scope plus operation-specific dimensions |
| FR-IC-008 | finance.ic.apply.payment.return | IC | ApplyPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-IC-009 | finance.ic.run.elimination | IC | RunElimination | Accounting scope plus operation-specific dimensions |
| FR-IC-010 | finance.ic.maintain.intercompany.agreements | IC | Maintain intercompany agreements | Accounting scope plus operation-specific dimensions |
| FR-IC-011 | finance.ic.record.intercompany.transactions | IC | Record intercompany transactions | Accounting scope plus operation-specific dimensions |
| FR-REV-001 | finance.rev.assess.contract | REV | AssessContract | Accounting scope plus operation-specific dimensions |
| FR-REV-002 | finance.rev.apply.revenue.schedule.approval.decision | REV | ApplyRevenueScheduleApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-REV-003 | finance.rev.publish.revenue.accounting.profile | REV | PublishRevenueAccountingProfile | Accounting scope plus operation-specific dimensions |
| FR-REV-004 | finance.rev.modify.contract | REV | ModifyContract | Accounting scope plus operation-specific dimensions |
| FR-REV-005 | finance.rev.apply.contract.modification.approval.decision | REV | ApplyContractModificationApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-REV-006 | finance.rev.run.recognition | REV | RunRecognition | Accounting scope plus operation-specific dimensions |
| FR-FA-001 | finance.fa.capitalize.asset | FA | CapitalizeAsset | Accounting scope plus operation-specific dimensions |
| FR-FA-002 | finance.fa.create.asset.acquisition.clearing | FA | CreateAssetAcquisitionClearing | Accounting scope plus operation-specific dimensions |
| FR-FA-003 | finance.fa.run.depreciation | FA | RunDepreciation | Accounting scope plus operation-specific dimensions |
| FR-FA-004 | finance.fa.apply.impairment.approval.decision | FA | ApplyImpairmentApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-FA-005 | finance.fa.dispose.asset | FA | DisposeAsset | Accounting scope plus operation-specific dimensions |
| FR-FA-006 | finance.fa.apply.asset.disposal.approval.decision | FA | ApplyAssetDisposalApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-FA-007 | finance.fa.cancel.unposted.asset.disposal | FA | CancelUnpostedAssetDisposal | Accounting scope plus operation-specific dimensions |
| FR-FA-008 | finance.fa.compensate.failed.disposal.posting | FA | CompensateFailedDisposalPosting | Accounting scope plus operation-specific dimensions |
| FR-FA-009 | finance.fa.create.disposal.settlement.clearing | FA | CreateDisposalSettlementClearing | Accounting scope plus operation-specific dimensions |
| FR-FA-010 | finance.fa.apply.asset.supplier.liability.result | FA | ApplyAssetSupplierLiabilityResult | Accounting scope plus operation-specific dimensions |
| FR-FA-011 | finance.fa.apply.incoming.settlement | FA | ApplyIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-FA-012 | finance.fa.reverse.incoming.settlement.application | FA | ReverseIncomingSettlementApplication | Accounting scope plus operation-specific dimensions |
| FR-FA-013 | finance.fa.apply.payment.return | FA | ApplyPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-FA-014 | finance.fa.apply.asset.settlement.result | FA | ApplyAssetSettlementResult | Accounting scope plus operation-specific dimensions |
| FR-FA-015 | finance.fa.reclassify.disposal.cost.for.payment | FA | ReclassifyDisposalCostForPayment | Accounting scope plus operation-specific dimensions |
| FR-FA-016 | finance.fa.request.disposal.cost.payment | FA | RequestDisposalCostPayment | Accounting scope plus operation-specific dimensions |
| FR-FA-017 | finance.fa.request.disposal.cost.payment.replacement | FA | RequestDisposalCostPaymentReplacement | Accounting scope plus operation-specific dimensions |
| FR-FA-018 | finance.fa.record.impairment.assessments | FA | Record impairment assessments | Accounting scope plus operation-specific dimensions |
| FR-FA-019 | finance.fa.transfer.assets.or.components | FA | Transfer assets or components | Accounting scope plus operation-specific dimensions |
| FR-FA-020 | finance.fa.split.assets.or.components | FA | Split assets or components | Accounting scope plus operation-specific dimensions |
| FR-FA-021 | finance.fa.correct.posted.asset.disposals | FA | Correct posted asset disposals | Accounting scope plus operation-specific dimensions |
| FR-FX-001 | finance.fx.publish.rate.set | FX | PublishRateSet | Accounting scope plus operation-specific dimensions |
| FR-FX-002 | finance.fx.run.revaluation | FX | RunRevaluation | Accounting scope plus operation-specific dimensions |
| FR-FX-003 | finance.fx.apply.revaluation.approval.decision | FX | ApplyRevaluationApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-FX-004 | finance.fx.post.revaluation.run | FX | PostRevaluationRun | Accounting scope plus operation-specific dimensions |
| FR-FX-005 | finance.fx.run.translation | FX | RunTranslation | Accounting scope plus operation-specific dimensions |
| FR-FPM-001 | finance.fpm.start.soft.close | FPM | StartSoftClose | Accounting scope plus operation-specific dimensions |
| FR-FPM-002 | finance.fpm.end.soft.close | FPM | EndSoftClose | Accounting scope plus operation-specific dimensions |
| FR-FPM-003 | finance.fpm.start.hard.close | FPM | StartHardClose | Accounting scope plus operation-specific dimensions |
| FR-FPM-004 | finance.fpm.resume.close.run | FPM | ResumeCloseRun | Accounting scope plus operation-specific dimensions |
| FR-FPM-005 | finance.fpm.abort.close.run | FPM | AbortCloseRun | Accounting scope plus operation-specific dimensions |
| FR-FPM-006 | finance.fpm.apply.posting.gate.result | FPM | ApplyPostingGateResult | Accounting scope plus operation-specific dimensions |
| FR-FPM-007 | finance.fpm.apply.close.exception.approval.decision | FPM | ApplyCloseExceptionApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-FPM-008 | finance.fpm.apply.close.approval.decision | FPM | ApplyCloseApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-FPM-009 | finance.fpm.request.reopen | FPM | RequestReopen | Accounting scope plus operation-specific dimensions |
| FR-FPM-010 | finance.fpm.apply.reopen.approval.decision | FPM | ApplyReopenApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-FPM-011 | finance.fpm.start.reclose | FPM | StartReclose | Accounting scope plus operation-specific dimensions |
| FR-FPM-012 | finance.fpm.take.over.period.control | FPM | TakeOverPeriodControl | Accounting scope plus operation-specific dimensions |
| FR-FPM-013 | finance.fpm.extend.close.exception | FPM | ExtendCloseException | Accounting scope plus operation-specific dimensions |
| FR-COA-001 | finance.coa.maintain.segment.definitions | COA | Maintain segment definitions | Accounting scope plus operation-specific dimensions |
| FR-COA-002 | finance.coa.maintain.segment.values | COA | Maintain segment values | Accounting scope plus operation-specific dimensions |
| FR-COA-003 | finance.coa.validate.segment.combinations | COA | Validate segment combinations | Accounting scope plus operation-specific dimensions |
| FR-COA-004 | finance.coa.request.segment.changes | COA | Request segment changes | Accounting scope plus operation-specific dimensions |
| FR-COA-005 | finance.coa.apply.segment.change.approval.decision | COA | ApplySegmentChangeApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-BFR-001 | finance.bfr.import.statement | BFR | ImportStatement | Accounting scope plus operation-specific dimensions |
| FR-BFR-002 | finance.bfr.propose.match | BFR | ProposeMatch | Accounting scope plus operation-specific dimensions |
| FR-BFR-003 | finance.bfr.confirm.match | BFR | ConfirmMatch | Accounting scope plus operation-specific dimensions |
| FR-BFR-004 | finance.bfr.unmatch | BFR | Unmatch | Accounting scope plus operation-specific dimensions |
| FR-BFR-005 | finance.bfr.complete.reconciliation | BFR | CompleteReconciliation | Accounting scope plus operation-specific dimensions |
| FR-BFR-006 | finance.bfr.maintain.bank.feed.connections | BFR | Maintain bank-feed connections | Accounting scope plus operation-specific dimensions |
| FR-TAX-001 | finance.tax.determine.tax | TAX | DetermineTax | Accounting scope plus operation-specific dimensions |
| FR-TAX-002 | finance.tax.prepare.tax.return | TAX | PrepareTaxReturn | Accounting scope plus operation-specific dimensions |
| FR-TAX-003 | finance.tax.apply.tax.return.approval.decision | TAX | ApplyTaxReturnApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-TAX-004 | finance.tax.submit.tax.return | TAX | SubmitTaxReturn | Accounting scope plus operation-specific dimensions |
| FR-TAX-005 | finance.tax.create.tax.amendment | TAX | CreateTaxAmendment | Accounting scope plus operation-specific dimensions |
| FR-TAX-006 | finance.tax.apply.tax.amendment.approval.decision | TAX | ApplyTaxAmendmentApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-TAX-007 | finance.tax.submit.tax.amendment | TAX | SubmitTaxAmendment | Accounting scope plus operation-specific dimensions |
| FR-TAX-008 | finance.tax.create.return.level.tax.adjustment | TAX | CreateReturnLevelTaxAdjustment | Accounting scope plus operation-specific dimensions |
| FR-TAX-009 | finance.tax.apply.return.level.tax.adjustment.approval.decision | TAX | ApplyReturnLevelTaxAdjustmentApprovalDecision | Accounting scope plus operation-specific dimensions |
| FR-TAX-010 | finance.tax.post.return.level.tax.adjustment | TAX | PostReturnLevelTaxAdjustment | Accounting scope plus operation-specific dimensions |
| FR-TAX-011 | finance.tax.request.tax.payment | TAX | RequestTaxPayment | Accounting scope plus operation-specific dimensions |
| FR-TAX-012 | finance.tax.record.tax.payment.settlement | TAX | RecordTaxPaymentSettlement | Accounting scope plus operation-specific dimensions |
| FR-TAX-013 | finance.tax.apply.incoming.settlement | TAX | ApplyIncomingSettlement | Accounting scope plus operation-specific dimensions |
| FR-TAX-014 | finance.tax.reverse.incoming.settlement.application | TAX | ReverseIncomingSettlementApplication | Accounting scope plus operation-specific dimensions |
| FR-TAX-015 | finance.tax.apply.payment.return | TAX | ApplyPaymentReturn | Accounting scope plus operation-specific dimensions |
| FR-TAX-016 | finance.tax.maintain.tax.configurations | TAX | Maintain tax configurations | Accounting scope plus operation-specific dimensions |
| FR-WFA-001 | finance.wfa.create.approval.request | WFA | CreateApprovalRequest | Accounting scope plus operation-specific dimensions |
| FR-WFA-002 | finance.wfa.decide.approval.request | WFA | DecideApprovalRequest | Accounting scope plus operation-specific dimensions |
| FR-WFA-003 | finance.wfa.delegate.approval | WFA | DelegateApproval | Accounting scope plus operation-specific dimensions |
| FR-WFA-004 | finance.wfa.escalate.approval | WFA | EscalateApproval | Accounting scope plus operation-specific dimensions |
| FR-WFA-005 | finance.wfa.maintain.approval.policies | WFA | Maintain approval policies | Accounting scope plus operation-specific dimensions |
| FR-IAM-001 | finance.iam.manage.users | IAM | Manage users | Accounting scope plus operation-specific dimensions |
| FR-IAM-002 | finance.iam.manage.roles | IAM | Manage roles | Accounting scope plus operation-specific dimensions |
| FR-IAM-003 | finance.iam.manage.access.policies | IAM | Manage access policies | Accounting scope plus operation-specific dimensions |
| FR-IAM-004 | finance.iam.manage.segregation.rules | IAM | Manage segregation rules | Accounting scope plus operation-specific dimensions |
| FR-IAM-005 | finance.iam.grant.emergency.access | IAM | Grant emergency access | Accounting scope plus operation-specific dimensions |
| FR-IAM-006 | finance.iam.revoke.emergency.access | IAM | Revoke emergency access | Accounting scope plus operation-specific dimensions |
| FR-AUD-001 | finance.aud.append.auditable.event | AUD | AppendAuditableEvent | Accounting scope plus operation-specific dimensions |
| FR-AUD-002 | finance.aud.create.audit.seal | AUD | CreateAuditSeal | Accounting scope plus operation-specific dimensions |
| FR-AUD-003 | finance.aud.rotate.verification.credential | AUD | RotateVerificationCredential | Accounting scope plus operation-specific dimensions |
| FR-AUD-004 | finance.aud.escalate.integrity.incident | AUD | EscalateIntegrityIncident | Accounting scope plus operation-specific dimensions |
| FR-AUD-005 | finance.aud.verify.proof | AUD | VerifyProof | Accounting scope plus operation-specific dimensions |

## 5. Segregation-of-duties controls

- Preparer cannot approve the same payment batch unless an approved emergency policy explicitly permits it.
- Reopen requester cannot approve the same reopen.
- Vendor bank-detail changer cannot release payment to those details during the cooling-off period.
- Manual-journal preparer cannot approve above the configured threshold.
- Payroll-detail permission is distinct from summary-ledger permission.
- Policy administration requires independent approval and audit evidence.

Segregation checks run both when a decision is recorded and when it is applied to the current business state.

## 6. Emergency access

Emergency grants contain actor, permissions/scopes, reason, approver, start, expiry and review state. Maximum default duration is four hours. Expired grants are unusable even if cleanup is delayed. All actions under a grant carry the grant reference and receive post-use review.

## 7. Service-to-service security

- Azure workloads use managed identity where supported.
- Internal HTTP calls, when introduced, require audience-specific tokens and private networking.
- PostgreSQL roles are distinct for API, worker, migrations and read-only reporting.
- Key Vault access uses least-privilege RBAC and managed identity.

## 8. Web security

| Control | Baseline |
|---|---|
| TLS | TLS 1.2 minimum; prefer TLS 1.3 where supported |
| CORS | Exact Static Web Apps origins; no wildcard with credentials |
| CSP | `default-src 'self'`; explicit Entra/API/connect/image allowances; no unsafe inline scripts |
| HSTS | Enabled outside local |
| CSRF | Bearer-token API with no ambient auth cookie; state-changing endpoints reject browser form content types |
| Content type | JSON only for API commands; strict parsing and unknown-field rejection |
| Uploads | Separate controlled path with MIME, size and malware controls |

## 9. Secrets and cryptographic material

- Key Vault stores provider credentials, certificates and verification/signing keys.
- Terraform state and CI logs must not contain secret values.
- Key identifiers, not secret material, appear in domain/audit records.
- Rotation retains overlap for verification, tests rollback and records the credential lineage.

## 10. Security verification

- Static analysis, dependency vulnerability scanning, secret scanning and container scanning gate pull requests/releases.
- Authorization tests cover every permission, deny-by-default, cross-scope access and all minimum segregation rules.
- Threat-model review is required for new external adapters, new sensitive projections or new upload/export paths.
- Penetration testing is required before any environment is represented as production-ready.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `6e88c05513a2d01dbf4aa39343be85c400c42d0b71289941d1c0e252d58a4cba` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
