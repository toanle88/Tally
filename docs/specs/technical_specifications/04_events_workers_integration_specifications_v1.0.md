# Finance Platform Events, Workers and Integration Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready integration baseline |
| Initial transport | PostgreSQL outbox/inbox |

## 1. Integration principles

- Domain outcomes are recorded in the owning module before integration delivery.
- The outbox record and local business effect commit in the same transaction.
- The consumer inbox record, local effect and resulting outbox records commit in the same transaction.
- At-least-once delivery is assumed; effects are exactly-once by business identity and local constraints.
- Delayed and out-of-order messages remain explicit states; no consumer fabricates missing prerequisites.

## 2. Event envelope

```json
{
  "messageId": "0195a91b-20ab-7c15-8aa8-4e111a8bd622",
  "eventType": "JournalEntryPosted",
  "eventVersion": 1,
  "occurredAt": "2026-07-24T08:00:00Z",
  "sourceContext": "gl",
  "aggregateId": "0195a91b-20ab-7c15-8aa8-4e111a8bd620",
  "aggregateVersion": 8,
  "accountingScopeId": "0195a91b-20ab-7c15-8aa8-4e111a8bd619",
  "correlationId": "0195a91b-20ab-7c15-8aa8-4e111a8bd621",
  "causationId": "0195a91b-20ab-7c15-8aa8-4e111a8bd618",
  "dataClassification": "internal",
  "payloadFingerprint": "sha256:...",
  "data": {}
}
```

## 3. Command inventory

| Owner | Command | Contract identity | Implementation |
|---|---|---|---|
| GL | SubmitPostingRequest | `submit-posting-request.v1` | Owning module application handler |
| GL | ApplyJournalApprovalDecision | `apply-journal-approval-decision.v1` | Owning module application handler |
| GL | ReverseJournalEntry | `reverse-journal-entry.v1` | Owning module application handler |
| GL | EnterSoftCloseGate | `enter-soft-close-gate.v1` | Owning module application handler |
| GL | ExitSoftCloseGate | `exit-soft-close-gate.v1` | Owning module application handler |
| GL | AcquirePostingBarrier | `acquire-posting-barrier.v1` | Owning module application handler |
| GL | ReleasePostingBarrier | `release-posting-barrier.v1` | Owning module application handler |
| GL | FinalizePostingGate | `finalize-posting-gate.v1` | Owning module application handler |
| GL | OpenScopedReopenGate | `open-scoped-reopen-gate.v1` | Owning module application handler |
| GL | CloseScopedReopenGate | `close-scoped-reopen-gate.v1` | Owning module application handler |
| GL | OpenOperationalReopenGate | `open-operational-reopen-gate.v1` | Owning module application handler |
| GL | CloseOperationalReopenGate | `close-operational-reopen-gate.v1` | Owning module application handler |
| GL | BeginRecloseGate | `begin-reclose-gate.v1` | Owning module application handler |
| AP | RegisterVendorInvoice | `register-vendor-invoice.v1` | Owning module application handler |
| AP | ApplyAssetClearingClassification | `apply-asset-clearing-classification.v1` | Owning module application handler |
| AP | ApplyIncomingSettlement | `apply-incoming-settlement.v1` | Owning module application handler |
| AP | ReverseIncomingSettlementApplication | `reverse-incoming-settlement-application.v1` | Owning module application handler |
| AP | ApplyPaymentReturn | `apply-payment-return.v1` | Owning module application handler |
| AP | ApplyVendorInvoiceApprovalDecision | `apply-vendor-invoice-approval-decision.v1` | Owning module application handler |
| AP | RequestPayment | `request-payment.v1` | Owning module application handler |
| AP | ValidateVendorInvoice | `validate-vendor-invoice.v1` | Owning module application handler |
| AP | DisputeVendorInvoice | `dispute-vendor-invoice.v1` | Owning module application handler |
| AP | VoidVendorInvoice | `void-vendor-invoice.v1` | Owning module application handler |
| AR | IssueCustomerInvoice | `issue-customer-invoice.v1` | Owning module application handler |
| AR | RecordReceipt | `record-receipt.v1` | Owning module application handler |
| AR | ApplyReceipt | `apply-receipt.v1` | Owning module application handler |
| AR | UnapplyReceipt | `unapply-receipt.v1` | Owning module application handler |
| AR | RollbackUnpostedApplicationBatch | `rollback-unposted-application-batch.v1` | Owning module application handler |
| AR | IssueCreditNote | `issue-credit-note.v1` | Owning module application handler |
| AR | CreateCustomerRefundRequest | `create-customer-refund-request.v1` | Owning module application handler |
| AR | CancelCustomerRefundRequest | `cancel-customer-refund-request.v1` | Owning module application handler |
| AR | ApplyCustomerRefundApprovalDecision | `apply-customer-refund-approval-decision.v1` | Owning module application handler |
| AR | RequestCustomerRefundPayment | `request-customer-refund-payment.v1` | Owning module application handler |
| AR | CancelCustomerRefundPayment | `cancel-customer-refund-payment.v1` | Owning module application handler |
| AR | ApplyCustomerRefundPaymentResult | `apply-customer-refund-payment-result.v1` | Owning module application handler |
| AR | ApplyPaymentReturn | `apply-payment-return.v1` | Owning module application handler |
| PAYR | CalculatePayrollRun | `calculate-payroll-run.v1` | Owning module application handler |
| PAYR | ApplyPayrollRunApprovalDecision | `apply-payroll-run-approval-decision.v1` | Owning module application handler |
| PAYR | PostPayrollRun | `post-payroll-run.v1` | Owning module application handler |
| PAYR | CreatePayrollCorrection | `create-payroll-correction.v1` | Owning module application handler |
| PAYR | ApplyPaymentReturn | `apply-payment-return.v1` | Owning module application handler |
| PCM | PreparePaymentBatch | `prepare-payment-batch.v1` | Owning module application handler |
| PCM | ApplyPaymentBatchApprovalDecision | `apply-payment-batch-approval-decision.v1` | Owning module application handler |
| PCM | CancelPaymentBatch | `cancel-payment-batch.v1` | Owning module application handler |
| PCM | RegisterExpectedIncomingSettlement | `register-expected-incoming-settlement.v1` | Owning module application handler |
| PCM | ResolveExpectedIncomingSettlementException | `resolve-expected-incoming-settlement-exception.v1` | Owning module application handler |
| PCM | CancelExpectedIncomingSettlement | `cancel-expected-incoming-settlement.v1` | Owning module application handler |
| PCM | CloseExpectedIncomingSettlement | `close-expected-incoming-settlement.v1` | Owning module application handler |
| PCM | CreatePaymentInstructionFromObligation | `create-payment-instruction-from-obligation.v1` | Owning module application handler |
| PCM | SubmitPaymentInstruction | `submit-payment-instruction.v1` | Owning module application handler |
| PCM | RetryPaymentInstruction | `retry-payment-instruction.v1` | Owning module application handler |
| PCM | CancelPaymentInstruction | `cancel-payment-instruction.v1` | Owning module application handler |
| PCM | ApplyPaymentInstructionExceptionDecision | `apply-payment-instruction-exception-decision.v1` | Owning module application handler |
| PCM | RecordPaymentReturn | `record-payment-return.v1` | Owning module application handler |
| PCM | CancelUnpostedPaymentReturn | `cancel-unposted-payment-return.v1` | Owning module application handler |
| PCM | AcknowledgePaymentReturn | `acknowledge-payment-return.v1` | Owning module application handler |
| PCM | ResolvePaymentReturnException | `resolve-payment-return-exception.v1` | Owning module application handler |
| PCM | RecordUnallocatedIncomingSettlement | `record-unallocated-incoming-settlement.v1` | Owning module application handler |
| PCM | ResolveUnallocatedIncomingSettlement | `resolve-unallocated-incoming-settlement.v1` | Owning module application handler |
| PCM | RecordIncomingSettlement | `record-incoming-settlement.v1` | Owning module application handler |
| PCM | ResolveSettlementReceiptValidationException | `resolve-settlement-receipt-validation-exception.v1` | Owning module application handler |
| PCM | ResolveIncomingSettlementOwnerException | `resolve-incoming-settlement-owner-exception.v1` | Owning module application handler |
| PCM | CancelUnpostedSettlementReceipt | `cancel-unposted-settlement-receipt.v1` | Owning module application handler |
| PCM | AcknowledgeIncomingSettlement | `acknowledge-incoming-settlement.v1` | Owning module application handler |
| PCM | ReverseIncomingSettlement | `reverse-incoming-settlement.v1` | Owning module application handler |
| RPT | RunConsolidation | `run-consolidation.v1` | Owning module application handler |
| RPT | ApplyTranslationResult | `apply-translation-result.v1` | Owning module application handler |
| RPT | ApplyConsolidationApprovalDecision | `apply-consolidation-approval-decision.v1` | Owning module application handler |
| RPT | PublishConsolidatedStatement | `publish-consolidated-statement.v1` | Owning module application handler |
| IC | StartSettlement | `start-settlement.v1` | Owning module application handler |
| IC | MatchIntercompanyItems | `match-intercompany-items.v1` | Owning module application handler |
| IC | ApplyResidualApprovalDecision | `apply-residual-approval-decision.v1` | Owning module application handler |
| IC | CreateSettlementInstructions | `create-settlement-instructions.v1` | Owning module application handler |
| IC | CompleteSettlementRun | `complete-settlement-run.v1` | Owning module application handler |
| IC | ApplyIncomingSettlement | `apply-incoming-settlement.v1` | Owning module application handler |
| IC | ReverseIncomingSettlementApplication | `reverse-incoming-settlement-application.v1` | Owning module application handler |
| IC | ApplyPaymentReturn | `apply-payment-return.v1` | Owning module application handler |
| IC | RunElimination | `run-elimination.v1` | Owning module application handler |
| REV | AssessContract | `assess-contract.v1` | Owning module application handler |
| REV | ApplyRevenueScheduleApprovalDecision | `apply-revenue-schedule-approval-decision.v1` | Owning module application handler |
| REV | PublishRevenueAccountingProfile | `publish-revenue-accounting-profile.v1` | Owning module application handler |
| REV | ModifyContract | `modify-contract.v1` | Owning module application handler |
| REV | ApplyContractModificationApprovalDecision | `apply-contract-modification-approval-decision.v1` | Owning module application handler |
| REV | RunRecognition | `run-recognition.v1` | Owning module application handler |
| FA | CapitalizeAsset | `capitalize-asset.v1` | Owning module application handler |
| FA | CreateAssetAcquisitionClearing | `create-asset-acquisition-clearing.v1` | Owning module application handler |
| FA | RunDepreciation | `run-depreciation.v1` | Owning module application handler |
| FA | ApplyImpairmentApprovalDecision | `apply-impairment-approval-decision.v1` | Owning module application handler |
| FA | DisposeAsset | `dispose-asset.v1` | Owning module application handler |
| FA | ApplyAssetDisposalApprovalDecision | `apply-asset-disposal-approval-decision.v1` | Owning module application handler |
| FA | CancelUnpostedAssetDisposal | `cancel-unposted-asset-disposal.v1` | Owning module application handler |
| FA | CompensateFailedDisposalPosting | `compensate-failed-disposal-posting.v1` | Owning module application handler |
| FA | CreateDisposalSettlementClearing | `create-disposal-settlement-clearing.v1` | Owning module application handler |
| FA | ApplyAssetSupplierLiabilityResult | `apply-asset-supplier-liability-result.v1` | Owning module application handler |
| FA | ApplyIncomingSettlement | `apply-incoming-settlement.v1` | Owning module application handler |
| FA | ReverseIncomingSettlementApplication | `reverse-incoming-settlement-application.v1` | Owning module application handler |
| FA | ApplyPaymentReturn | `apply-payment-return.v1` | Owning module application handler |
| FA | ApplyAssetSettlementResult | `apply-asset-settlement-result.v1` | Owning module application handler |
| FA | ReclassifyDisposalCostForPayment | `reclassify-disposal-cost-for-payment.v1` | Owning module application handler |
| FA | RequestDisposalCostPayment | `request-disposal-cost-payment.v1` | Owning module application handler |
| FA | RequestDisposalCostPaymentReplacement | `request-disposal-cost-payment-replacement.v1` | Owning module application handler |
| FX | PublishRateSet | `publish-rate-set.v1` | Owning module application handler |
| FX | RunRevaluation | `run-revaluation.v1` | Owning module application handler |
| FX | ApplyRevaluationApprovalDecision | `apply-revaluation-approval-decision.v1` | Owning module application handler |
| FX | PostRevaluationRun | `post-revaluation-run.v1` | Owning module application handler |
| FX | RunTranslation | `run-translation.v1` | Owning module application handler |
| FPM | StartSoftClose | `start-soft-close.v1` | Owning module application handler |
| FPM | EndSoftClose | `end-soft-close.v1` | Owning module application handler |
| FPM | StartHardClose | `start-hard-close.v1` | Owning module application handler |
| FPM | ResumeCloseRun | `resume-close-run.v1` | Owning module application handler |
| FPM | AbortCloseRun | `abort-close-run.v1` | Owning module application handler |
| FPM | ApplyPostingGateResult | `apply-posting-gate-result.v1` | Owning module application handler |
| FPM | ApplyCloseExceptionApprovalDecision | `apply-close-exception-approval-decision.v1` | Owning module application handler |
| FPM | ApplyCloseApprovalDecision | `apply-close-approval-decision.v1` | Owning module application handler |
| FPM | RequestReopen | `request-reopen.v1` | Owning module application handler |
| FPM | ApplyReopenApprovalDecision | `apply-reopen-approval-decision.v1` | Owning module application handler |
| FPM | StartReclose | `start-reclose.v1` | Owning module application handler |
| FPM | TakeOverPeriodControl | `take-over-period-control.v1` | Owning module application handler |
| FPM | ExtendCloseException | `extend-close-exception.v1` | Owning module application handler |
| COA | ApplySegmentChangeApprovalDecision | `apply-segment-change-approval-decision.v1` | Owning module application handler |
| BFR | ImportStatement | `import-statement.v1` | Owning module application handler |
| BFR | ProposeMatch | `propose-match.v1` | Owning module application handler |
| BFR | ConfirmMatch | `confirm-match.v1` | Owning module application handler |
| BFR | Unmatch | `unmatch.v1` | Owning module application handler |
| BFR | CompleteReconciliation | `complete-reconciliation.v1` | Owning module application handler |
| TAX | DetermineTax | `determine-tax.v1` | Owning module application handler |
| TAX | PrepareTaxReturn | `prepare-tax-return.v1` | Owning module application handler |
| TAX | ApplyTaxReturnApprovalDecision | `apply-tax-return-approval-decision.v1` | Owning module application handler |
| TAX | SubmitTaxReturn | `submit-tax-return.v1` | Owning module application handler |
| TAX | CreateTaxAmendment | `create-tax-amendment.v1` | Owning module application handler |
| TAX | ApplyTaxAmendmentApprovalDecision | `apply-tax-amendment-approval-decision.v1` | Owning module application handler |
| TAX | SubmitTaxAmendment | `submit-tax-amendment.v1` | Owning module application handler |
| TAX | CreateReturnLevelTaxAdjustment | `create-return-level-tax-adjustment.v1` | Owning module application handler |
| TAX | ApplyReturnLevelTaxAdjustmentApprovalDecision | `apply-return-level-tax-adjustment-approval-decision.v1` | Owning module application handler |
| TAX | PostReturnLevelTaxAdjustment | `post-return-level-tax-adjustment.v1` | Owning module application handler |
| TAX | RequestTaxPayment | `request-tax-payment.v1` | Owning module application handler |
| TAX | RecordTaxPaymentSettlement | `record-tax-payment-settlement.v1` | Owning module application handler |
| TAX | ApplyIncomingSettlement | `apply-incoming-settlement.v1` | Owning module application handler |
| TAX | ReverseIncomingSettlementApplication | `reverse-incoming-settlement-application.v1` | Owning module application handler |
| TAX | ApplyPaymentReturn | `apply-payment-return.v1` | Owning module application handler |
| WFA | CreateApprovalRequest | `create-approval-request.v1` | Owning module application handler |
| WFA | DecideApprovalRequest | `decide-approval-request.v1` | Owning module application handler |
| WFA | DelegateApproval | `delegate-approval.v1` | Owning module application handler |
| WFA | EscalateApproval | `escalate-approval.v1` | Owning module application handler |
| AUD | AppendAuditableEvent | `append-auditable-event.v1` | Owning module application handler |
| AUD | CreateAuditSeal | `create-audit-seal.v1` | Owning module application handler |
| AUD | RotateVerificationCredential | `rotate-verification-credential.v1` | Owning module application handler |
| AUD | EscalateIntegrityIncident | `escalate-integrity-incident.v1` | Owning module application handler |

## 4. Event catalog

| Producer | Event | Version | Source schema | Payload schema name |
|---|---|---|---|---|
| GL | JournalEntryPosted | 1 | gl | `journal_entry_posted` |
| GL | PostingRejected | 1 | gl | `posting_rejected` |
| GL | PostingPendingApproval | 1 | gl | `posting_pending_approval` |
| GL | IdempotencyConflict | 1 | gl | `idempotency_conflict` |
| GL | JournalEntryReversed | 1 | gl | `journal_entry_reversed` |
| GL | PostingAdmissionRecorded | 1 | gl | `posting_admission_recorded` |
| GL | SoftCloseGateEntered | 1 | gl | `soft_close_gate_entered` |
| GL | SoftCloseGateExited | 1 | gl | `soft_close_gate_exited` |
| GL | PostingBarrierAcquired | 1 | gl | `posting_barrier_acquired` |
| GL | PostingBarrierReleased | 1 | gl | `posting_barrier_released` |
| GL | PostingGateFinalized | 1 | gl | `posting_gate_finalized` |
| GL | ScopedReopenGateOpened | 1 | gl | `scoped_reopen_gate_opened` |
| GL | ScopedReopenGateClosed | 1 | gl | `scoped_reopen_gate_closed` |
| GL | OperationalReopenGateOpened | 1 | gl | `operational_reopen_gate_opened` |
| GL | OperationalReopenGateClosed | 1 | gl | `operational_reopen_gate_closed` |
| GL | OperationalReopenGateExpired | 1 | gl | `operational_reopen_gate_expired` |
| GL | RecloseGateBegun | 1 | gl | `reclose_gate_begun` |
| AP | VendorInvoiceApprovalApplied | 1 | ap | `vendor_invoice_approval_applied` |
| AP | VendorInvoiceApproved | 1 | ap | `vendor_invoice_approved` |
| AP | AssetSupplierLiabilityPosted | 1 | ap | `asset_supplier_liability_posted` |
| AP | AssetSupplierLiabilityReversed | 1 | ap | `asset_supplier_liability_reversed` |
| AP | IncomingSettlementApplied | 1 | ap | `incoming_settlement_applied` |
| AP | IncomingSettlementApplicationRejected | 1 | ap | `incoming_settlement_application_rejected` |
| AP | IncomingSettlementApplicationReversed | 1 | ap | `incoming_settlement_application_reversed` |
| AP | PaymentReturnApplied | 1 | ap | `payment_return_applied` |
| AP | PaymentReturnApplicationRejected | 1 | ap | `payment_return_application_rejected` |
| AP | PaymentInstructionExceptionDecisionRecorded | 1 | ap | `payment_instruction_exception_decision_recorded` |
| AP | PaymentRequested | 1 | ap | `payment_requested` |
| AP | VendorInvoicePaid | 1 | ap | `vendor_invoice_paid` |
| AR | CustomerInvoiceIssued | 1 | ar | `customer_invoice_issued` |
| AR | ReceivableOpenItemCreated | 1 | ar | `receivable_open_item_created` |
| AR | ReceiptRecorded | 1 | ar | `receipt_recorded` |
| AR | ReceiptAccountingPosted | 1 | ar | `receipt_accounting_posted` |
| AR | ReceiptAccountingFailed | 1 | ar | `receipt_accounting_failed` |
| AR | ReceiptApplied | 1 | ar | `receipt_applied` |
| AR | ReceiptApplicationAccountingPosted | 1 | ar | `receipt_application_accounting_posted` |
| AR | ReceiptApplicationAccountingFailed | 1 | ar | `receipt_application_accounting_failed` |
| AR | ReceiptApplicationCancellationStarted | 1 | ar | `receipt_application_cancellation_started` |
| AR | ReceiptApplicationCancelledNoJournal | 1 | ar | `receipt_application_cancelled_no_journal` |
| AR | ReceiptUnapplied | 1 | ar | `receipt_unapplied` |
| AR | ReceiptUnapplicationAccountingPosted | 1 | ar | `receipt_unapplication_accounting_posted` |
| AR | ReceiptUnapplicationAccountingFailed | 1 | ar | `receipt_unapplication_accounting_failed` |
| AR | ReceivableOpenItemBalanceChanged | 1 | ar | `receivable_open_item_balance_changed` |
| AR | CreditNoteIssued | 1 | ar | `credit_note_issued` |
| AR | CustomerRefundRequestCreated | 1 | ar | `customer_refund_request_created` |
| AR | CustomerRefundRequestCancelled | 1 | ar | `customer_refund_request_cancelled` |
| AR | CustomerRefundApprovalApplied | 1 | ar | `customer_refund_approval_applied` |
| AR | CustomerRefundPaymentRequested | 1 | ar | `customer_refund_payment_requested` |
| AR | CustomerRefundPaymentCancellationRequested | 1 | ar | `customer_refund_payment_cancellation_requested` |
| AR | CustomerRefundPaymentReplacementRequested | 1 | ar | `customer_refund_payment_replacement_requested` |
| AR | CustomerRefundStatusUpdated | 1 | ar | `customer_refund_status_updated` |
| AR | PaymentReturnApplied | 1 | ar | `payment_return_applied` |
| AR | PaymentReturnApplicationRejected | 1 | ar | `payment_return_application_rejected` |
| AR | PaymentInstructionExceptionDecisionRecorded | 1 | ar | `payment_instruction_exception_decision_recorded` |
| PAYR | PayrollRunCalculated | 1 | payroll | `payroll_run_calculated` |
| PAYR | PaymentReturnApplied | 1 | payroll | `payment_return_applied` |
| PAYR | PaymentReturnApplicationRejected | 1 | payroll | `payment_return_application_rejected` |
| PAYR | PaymentInstructionExceptionDecisionRecorded | 1 | payroll | `payment_instruction_exception_decision_recorded` |
| PAYR | PayrollRunApprovalApplied | 1 | payroll | `payroll_run_approval_applied` |
| PAYR | PayrollRunPosted | 1 | payroll | `payroll_run_posted` |
| PAYR | PayrollCorrectionCreated | 1 | payroll | `payroll_correction_created` |
| PCM | PaymentBatchApprovalApplied | 1 | payments | `payment_batch_approval_applied` |
| PCM | PaymentBatchCancelled | 1 | payments | `payment_batch_cancelled` |
| PCM | PaymentBatchCompleted | 1 | payments | `payment_batch_completed` |
| PCM | ExpectedIncomingSettlementRegistered | 1 | payments | `expected_incoming_settlement_registered` |
| PCM | ExpectedIncomingSettlementExceptionResolved | 1 | payments | `expected_incoming_settlement_exception_resolved` |
| PCM | ExpectedIncomingSettlementCancelled | 1 | payments | `expected_incoming_settlement_cancelled` |
| PCM | ExpectedIncomingSettlementClosed | 1 | payments | `expected_incoming_settlement_closed` |
| PCM | PaymentInstructionSubmitted | 1 | payments | `payment_instruction_submitted` |
| PCM | PaymentInstructionPartiallySettled | 1 | payments | `payment_instruction_partially_settled` |
| PCM | PaymentInstructionCancelled | 1 | payments | `payment_instruction_cancelled` |
| PCM | PaymentInstructionRemainderCancelled | 1 | payments | `payment_instruction_remainder_cancelled` |
| PCM | PaymentInstructionUnpaidAmountRestored | 1 | payments | `payment_instruction_unpaid_amount_restored` |
| PCM | PaymentInstructionPartiallySettledCancelled | 1 | payments | `payment_instruction_partially_settled_cancelled` |
| PCM | PaymentInstructionFailed | 1 | payments | `payment_instruction_failed` |
| PCM | PaymentInstructionExceptionPending | 1 | payments | `payment_instruction_exception_pending` |
| PCM | PaymentInstructionExceptionDecisionRequired | 1 | payments | `payment_instruction_exception_decision_required` |
| PCM | PaymentInstructionExceptionResolved | 1 | payments | `payment_instruction_exception_resolved` |
| PCM | PaymentInstructionSettled | 1 | payments | `payment_instruction_settled` |
| PCM | PaymentReturnRecorded | 1 | payments | `payment_return_recorded` |
| PCM | PaymentReturnValidationRejected | 1 | payments | `payment_return_validation_rejected` |
| PCM | PaymentReturnPosted | 1 | payments | `payment_return_posted` |
| PCM | PaymentReturnAwaitingOwnerAcknowledgement | 1 | payments | `payment_return_awaiting_owner_acknowledgement` |
| PCM | PaymentReturnPostingFailed | 1 | payments | `payment_return_posting_failed` |
| PCM | PaymentReturnCancellationStarted | 1 | payments | `payment_return_cancellation_started` |
| PCM | PaymentReturnCancelledNoJournal | 1 | payments | `payment_return_cancelled_no_journal` |
| PCM | PaymentReturnExceptionDecisionApplied | 1 | payments | `payment_return_exception_decision_applied` |
| PCM | PaymentReturnReversalPending | 1 | payments | `payment_return_reversal_pending` |
| PCM | PaymentReturnReversalFailed | 1 | payments | `payment_return_reversal_failed` |
| PCM | PaymentReturnReversed | 1 | payments | `payment_return_reversed` |
| PCM | PaymentReturnReconciled | 1 | payments | `payment_return_reconciled` |
| PCM | PaymentInstructionReturned | 1 | payments | `payment_instruction_returned` |
| PCM | UnallocatedIncomingSettlementRecorded | 1 | payments | `unallocated_incoming_settlement_recorded` |
| PCM | UnallocatedIncomingSettlementPostingPending | 1 | payments | `unallocated_incoming_settlement_posting_pending` |
| PCM | UnallocatedIncomingSettlementPosted | 1 | payments | `unallocated_incoming_settlement_posted` |
| PCM | UnallocatedIncomingSettlementPostingFailed | 1 | payments | `unallocated_incoming_settlement_posting_failed` |
| PCM | UnallocatedIncomingSettlementResolved | 1 | payments | `unallocated_incoming_settlement_resolved` |
| PCM | CustomerRefundPartiallySettled | 1 | payments | `customer_refund_partially_settled` |
| PCM | CustomerRefundSettled | 1 | payments | `customer_refund_settled` |
| PCM | CustomerRefundFailed | 1 | payments | `customer_refund_failed` |
| PCM | CustomerRefundRemainderCancelled | 1 | payments | `customer_refund_remainder_cancelled` |
| PCM | IncomingSettlementRecorded | 1 | payments | `incoming_settlement_recorded` |
| PCM | IncomingSettlementValidationRejected | 1 | payments | `incoming_settlement_validation_rejected` |
| PCM | IncomingSettlementValidationExceptionResolved | 1 | payments | `incoming_settlement_validation_exception_resolved` |
| PCM | IncomingSettlementPosted | 1 | payments | `incoming_settlement_posted` |
| PCM | IncomingSettlementAwaitingOwnerAcknowledgement | 1 | payments | `incoming_settlement_awaiting_owner_acknowledgement` |
| PCM | IncomingSettlementOwnerApplicationRejected | 1 | payments | `incoming_settlement_owner_application_rejected` |
| PCM | IncomingSettlementOwnerExceptionResolved | 1 | payments | `incoming_settlement_owner_exception_resolved` |
| PCM | IncomingSettlementCancellationStarted | 1 | payments | `incoming_settlement_cancellation_started` |
| PCM | IncomingSettlementCancelledNoJournal | 1 | payments | `incoming_settlement_cancelled_no_journal` |
| PCM | IncomingSettlementFailed | 1 | payments | `incoming_settlement_failed` |
| PCM | IncomingSettlementReconciled | 1 | payments | `incoming_settlement_reconciled` |
| PCM | IncomingSettlementReversed | 1 | payments | `incoming_settlement_reversed` |
| PCM | DisposalCostPaymentPartiallySettled | 1 | payments | `disposal_cost_payment_partially_settled` |
| PCM | DisposalCostPaymentSettled | 1 | payments | `disposal_cost_payment_settled` |
| PCM | AssetSettlementFailed | 1 | payments | `asset_settlement_failed` |
| RPT | ConsolidationApprovalApplied | 1 | reporting | `consolidation_approval_applied` |
| RPT | ConsolidationPublished | 1 | reporting | `consolidation_published` |
| RPT | ConsolidationFailed | 1 | reporting | `consolidation_failed` |
| RPT | ConsolidatedStatementPublished | 1 | reporting | `consolidated_statement_published` |
| IC | IncomingSettlementApplied | 1 | intercompany | `incoming_settlement_applied` |
| IC | IncomingSettlementApplicationRejected | 1 | intercompany | `incoming_settlement_application_rejected` |
| IC | IncomingSettlementApplicationReversed | 1 | intercompany | `incoming_settlement_application_reversed` |
| IC | PaymentReturnApplied | 1 | intercompany | `payment_return_applied` |
| IC | PaymentReturnApplicationRejected | 1 | intercompany | `payment_return_application_rejected` |
| IC | PaymentInstructionExceptionDecisionRecorded | 1 | intercompany | `payment_instruction_exception_decision_recorded` |
| IC | SettlementCompleted | 1 | intercompany | `settlement_completed` |
| IC | SettlementFailed | 1 | intercompany | `settlement_failed` |
| IC | EliminationInstructionsCreated | 1 | intercompany | `elimination_instructions_created` |
| REV | RevenueScheduleApprovalApplied | 1 | revenue | `revenue_schedule_approval_applied` |
| REV | RevenueAccountingProfilePublished | 1 | revenue | `revenue_accounting_profile_published` |
| REV | RevenueScheduleActivated | 1 | revenue | `revenue_schedule_activated` |
| REV | ContractModificationApprovalApplied | 1 | revenue | `contract_modification_approval_applied` |
| REV | ContractModified | 1 | revenue | `contract_modified` |
| REV | RevenueRecognized | 1 | revenue | `revenue_recognized` |
| FA | AssetCapitalized | 1 | fixed_assets | `asset_capitalized` |
| FA | AssetAcquisitionClearingPublished | 1 | fixed_assets | `asset_acquisition_clearing_published` |
| FA | DepreciationCalculated | 1 | fixed_assets | `depreciation_calculated` |
| FA | ImpairmentApprovalApplied | 1 | fixed_assets | `impairment_approval_applied` |
| FA | AssetDisposalApprovalApplied | 1 | fixed_assets | `asset_disposal_approval_applied` |
| FA | AssetDisposalApproved | 1 | fixed_assets | `asset_disposal_approved` |
| FA | DisposalSettlementClearingCreated | 1 | fixed_assets | `disposal_settlement_clearing_created` |
| FA | DisposalSupplierCostClassificationPublished | 1 | fixed_assets | `disposal_supplier_cost_classification_published` |
| FA | ExpectedAssetProceedsSettlementCreated | 1 | fixed_assets | `expected_asset_proceeds_settlement_created` |
| FA | AssetIncomingSettlementApplied | 1 | fixed_assets | `asset_incoming_settlement_applied` |
| FA | AssetIncomingSettlementApplicationRejected | 1 | fixed_assets | `asset_incoming_settlement_application_rejected` |
| FA | AssetIncomingSettlementApplicationReversed | 1 | fixed_assets | `asset_incoming_settlement_application_reversed` |
| FA | DisposalCostReclassifiedForPayment | 1 | fixed_assets | `disposal_cost_reclassified_for_payment` |
| FA | DisposalCostPaymentRequested | 1 | fixed_assets | `disposal_cost_payment_requested` |
| FA | DisposalCostPaymentReplacementRequested | 1 | fixed_assets | `disposal_cost_payment_replacement_requested` |
| FA | PaymentReturnApplied | 1 | fixed_assets | `payment_return_applied` |
| FA | PaymentReturnApplicationRejected | 1 | fixed_assets | `payment_return_application_rejected` |
| FA | PaymentInstructionExceptionDecisionRecorded | 1 | fixed_assets | `payment_instruction_exception_decision_recorded` |
| FA | AssetDisposalCancellationStarted | 1 | fixed_assets | `asset_disposal_cancellation_started` |
| FA | AssetDisposalCancelledNoJournal | 1 | fixed_assets | `asset_disposal_cancelled_no_journal` |
| FA | AssetDisposalCompensationStarted | 1 | fixed_assets | `asset_disposal_compensation_started` |
| FA | AssetDisposalCompensatedFailed | 1 | fixed_assets | `asset_disposal_compensated_failed` |
| FA | AssetDisposalCompensationFailed | 1 | fixed_assets | `asset_disposal_compensation_failed` |
| FA | AssetDisposed | 1 | fixed_assets | `asset_disposed` |
| FX | RateSetPublished | 1 | multi_currency | `rate_set_published` |
| FX | RevaluationApprovalApplied | 1 | multi_currency | `revaluation_approval_applied` |
| FX | RevaluationPostingStarted | 1 | multi_currency | `revaluation_posting_started` |
| FX | RevaluationCompleted | 1 | multi_currency | `revaluation_completed` |
| FX | RevaluationFailed | 1 | multi_currency | `revaluation_failed` |
| FX | TranslationResultPublished | 1 | multi_currency | `translation_result_published` |
| FX | TranslationRunFailed | 1 | multi_currency | `translation_run_failed` |
| FPM | SoftCloseStarted | 1 | fiscal_period | `soft_close_started` |
| FPM | SoftCloseEnded | 1 | fiscal_period | `soft_close_ended` |
| FPM | SoftCloseHandoffStarted | 1 | fiscal_period | `soft_close_handoff_started` |
| FPM | SoftCloseResumed | 1 | fiscal_period | `soft_close_resumed` |
| FPM | SoftCloseSuperseded | 1 | fiscal_period | `soft_close_superseded` |
| FPM | GateAdmissionSummaryRecorded | 1 | fiscal_period | `gate_admission_summary_recorded` |
| FPM | PeriodStateChanged | 1 | fiscal_period | `period_state_changed` |
| FPM | CloseStepCompleted | 1 | fiscal_period | `close_step_completed` |
| FPM | CloseRunResumed | 1 | fiscal_period | `close_run_resumed` |
| FPM | CloseRunAborted | 1 | fiscal_period | `close_run_aborted` |
| FPM | ReopenRequested | 1 | fiscal_period | `reopen_requested` |
| FPM | PeriodReopened | 1 | fiscal_period | `period_reopened` |
| FPM | ReopenCompletedNoChange | 1 | fiscal_period | `reopen_completed_no_change` |
| FPM | RecloseHandoffStarted | 1 | fiscal_period | `reclose_handoff_started` |
| FPM | OperationalReopenActivated | 1 | fiscal_period | `operational_reopen_activated` |
| FPM | OperationalReopenRequestExpired | 1 | fiscal_period | `operational_reopen_request_expired` |
| FPM | PeriodReclosed | 1 | fiscal_period | `period_reclosed` |
| BFR | StatementImported | 1 | bank_reconciliation | `statement_imported` |
| BFR | MatchConfirmed | 1 | bank_reconciliation | `match_confirmed` |
| BFR | MatchReversed | 1 | bank_reconciliation | `match_reversed` |
| BFR | ReconciliationCompleted | 1 | bank_reconciliation | `reconciliation_completed` |
| TAX | TaxDetermined | 1 | tax | `tax_determined` |
| TAX | TaxReturnApprovalApplied | 1 | tax | `tax_return_approval_applied` |
| TAX | TaxReturnSubmitted | 1 | tax | `tax_return_submitted` |
| TAX | TaxSubmissionRejected | 1 | tax | `tax_submission_rejected` |
| TAX | TaxAmendmentCreated | 1 | tax | `tax_amendment_created` |
| TAX | TaxAmendmentApprovalApplied | 1 | tax | `tax_amendment_approval_applied` |
| TAX | TaxAmendmentSubmitted | 1 | tax | `tax_amendment_submitted` |
| TAX | TaxAmendmentAccepted | 1 | tax | `tax_amendment_accepted` |
| TAX | TaxAmendmentRejected | 1 | tax | `tax_amendment_rejected` |
| TAX | ReturnLevelTaxAdjustmentCreated | 1 | tax | `return_level_tax_adjustment_created` |
| TAX | ReturnLevelTaxAdjustmentApprovalApplied | 1 | tax | `return_level_tax_adjustment_approval_applied` |
| TAX | ReturnLevelTaxAdjustmentPosted | 1 | tax | `return_level_tax_adjustment_posted` |
| TAX | ReturnLevelTaxAdjustmentFailed | 1 | tax | `return_level_tax_adjustment_failed` |
| TAX | TaxPaymentRequested | 1 | tax | `tax_payment_requested` |
| TAX | TaxPaymentSettled | 1 | tax | `tax_payment_settled` |
| TAX | TaxPaymentFailed | 1 | tax | `tax_payment_failed` |
| TAX | IncomingSettlementApplied | 1 | tax | `incoming_settlement_applied` |
| TAX | IncomingSettlementApplicationRejected | 1 | tax | `incoming_settlement_application_rejected` |
| TAX | IncomingSettlementApplicationReversed | 1 | tax | `incoming_settlement_application_reversed` |
| TAX | PaymentReturnApplied | 1 | tax | `payment_return_applied` |
| TAX | PaymentReturnApplicationRejected | 1 | tax | `payment_return_application_rejected` |
| TAX | PaymentInstructionExceptionDecisionRecorded | 1 | tax | `payment_instruction_exception_decision_recorded` |
| WFA | ApprovalRequested | 1 | workflow | `approval_requested` |
| WFA | ApprovalDecisionRecorded | 1 | workflow | `approval_decision_recorded` |
| WFA | ApprovalDelegated | 1 | workflow | `approval_delegated` |
| WFA | ApprovalEscalated | 1 | workflow | `approval_escalated` |
| AUD | AuditableEventAppended | 1 | audit | `auditable_event_appended` |
| AUD | AuditSealCreated | 1 | audit | `audit_seal_created` |
| AUD | VerificationCredentialRotated | 1 | audit | `verification_credential_rotated` |
| AUD | IntegrityIncidentEscalated | 1 | audit | `integrity_incident_escalated` |

Each payload schema contains only the minimum facts required by approved consumers. Full aggregates, secrets, bank numbers, payroll details and unrestricted remittance text are prohibited.

## 5. Outbox claiming algorithm

```sql
with claim as (
  select outbox_id
  from integration.outbox
  where established_at is null
    and available_at <= clock_timestamp()
    and (claimed_until is null or claimed_until < clock_timestamp())
  order by available_at, outbox_id
  for update skip locked
  limit $1
)
update integration.outbox o
set claimed_until = clock_timestamp() + $2::interval,
    claim_owner = $3,
    attempt_count = attempt_count + 1
from claim
where o.outbox_id = claim.outbox_id
returning o.*;
```

- Claim duration is 30 seconds by default and must exceed the measured 99th-percentile handler duration.
- A dispatcher renews before two-thirds of the lease is consumed.
- Establishment or reschedule checks `claim_owner`; stale workers cannot overwrite a newer claim.
- Poison work becomes a managed exception; it is never silently deleted.

## 6. Inbox algorithm

1. Insert `(consumer_name, message_id, fingerprint, processing)`.
2. On primary-key conflict, compare fingerprints.
3. Same fingerprint and established state returns the stored result.
4. Different fingerprint returns an identity-content conflict and raises an integrity alert.
5. Load prerequisites and expected versions.
6. Apply the receiving module's domain operation.
7. Commit inbox result, local effects and new outbox records together.

## 7. Worker catalog

| Worker | Source | Cadence | Batch | Claim/selection | Failure handling |
|---|---|---|---|---|---|
| outbox-dispatcher | integration.outbox | 1s | 100 | Lease with `FOR UPDATE SKIP LOCKED` | Exponential backoff; managed failure after 10 attempts |
| inbox-reconciler | integration.inbox | 30s | 100 | Find processing/failed items | Reconcile established local result before retry |
| approval-escalation | workflow.approval_request | 1m | 100 | Due/expired approval steps | Emit escalation or expiry outcome once |
| period-control-expiry | fiscal_period.reopen_request | 1m | 100 | Authority expiry reached | Close admission first; never infer closure from time alone |
| payment-retry | payments.payment_instruction | 30s | 50 | Retryable provider outcomes | Provider idempotency and result lookup required |
| report-projection | integration.outbox | 5s | 500 | Reporting consumer inbox | Rebuildable from authoritative watermark |
| audit-sealer | audit.audit_chain | 1m | 1000 events | Canonical ordered range | Same seal request identity on retry |
| retention-review | platform.retention_policy | 24h | 100 | Eligible records without hold | Evidence and approval before destruction |

The `cmd/worker` process hosts all workers initially. Each worker has an independent concurrency limit, database pool budget, shutdown deadline and metrics namespace.

## 8. External adapter contracts

| Adapter | Required behavior |
|---|---|
| Bank/payment provider | Provider idempotency key, signed/verified callback evidence, result lookup before retry, normalized failure taxonomy |
| Procurement snapshot | Immutable PO/receipt version, source fingerprint, effective source identity |
| Tax authority | Submission identity, status polling, rejection evidence, amendment lineage |
| Payroll/filing provider | Submission identity, restricted payload projection, settlement and return evidence |
| Evidence/document service | Metadata and expiring access link only; business record stores immutable reference |
| Entra ID | OIDC discovery and JWKS validation; no finance authorization ownership |

## 9. Retry policy

- Retry only typed transient dependency failures.
- Default delays: 5s, 30s, 2m, 10m, 30m; context adapters may override through approved configuration.
- Domain rejection, authorization denial, idempotency conflict and data-integrity mismatch are not automatically retried.
- Before retrying an ambiguous external submission, query the provider using the original stable identity.

## 10. Broker adoption trigger

Azure Service Bus may replace or supplement the database dispatcher only after an ADR demonstrates independent deployment/fan-out, measured throughput, or external durability needs. Event identity, inbox semantics, payload schemas and domain ownership remain unchanged.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `00ad16ef174204066ce96252668fb10762487fad23847d9c6e0b9211f1b6e260` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
