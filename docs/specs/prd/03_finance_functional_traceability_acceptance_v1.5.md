# Finance Platform Functional Traceability and Acceptance

| Document-control field | Value |
|---|---|
| Version | 1.5 |
| Baseline date | 2026-07-24 |
| Status | Consistency-verified traceability and acceptance baseline |
| Source | Finance DDD v3.1 checkpoint DDD-3.1-2026-07-24 |
| Companions | Finance Functional PRD v1.5; Functional Requirements Catalog v1.5 |
| Source verified-content SHA-256 | a9d437d23656c36d340afb3a5a31c93a23e574f53db186483a9edfdf32d3e652 |

## 1. Traceability Rules
- `Direct requirement IDs` identify exact requirements for named DDD operations and explicit PRD functional actions that provide the stated workflow behavior.
- `Supporting families` identify cross-cutting or dependent capability requirements. A supporting family does not replace a missing direct requirement for an explicitly scoped functional behavior.
- Functional acceptance IDs use the prefix `FAC` to avoid being mistaken for source DDD acceptance identifiers.
- Acceptance wording is derived from DDD §14 and does not redefine ownership, equations, lifecycle states, or correction semantics.

## 2. Workflow-to-Requirement Traceability
| Workflow ID | Functional workflow | DDD source | Direct requirement IDs | Supporting families | Acceptance source |
|---|---|---|---|---|---|
| WF-6.1 | Period Close: Hard Close | DDD §6.1 | FR-FPM-003, FR-FPM-006, FR-FPM-007, FR-FPM-008, FR-GL-006, FR-GL-007, FR-GL-008, FR-GL-014, FR-RPT-006 | FR-FPM-*, FR-GL-*, FR-WFA-*, FR-FX-*, FR-FA-*, FR-REV-*, FR-IC-*, FR-RPT-*, FR-AUD-* | DDD acceptance §§14.3 and 14.8 |
| WF-6.2 | Fiscal Period Reopen and Reclose | DDD §6.2 | FR-FPM-006, FR-FPM-009, FR-FPM-010, FR-FPM-011, FR-GL-001, FR-GL-007, FR-GL-008, FR-GL-009, FR-GL-010, FR-GL-013, FR-GL-014 | FR-FPM-*, FR-GL-*, FR-WFA-*, FR-RPT-*, FR-AUD-* | DDD acceptance §§14.10 and 14.8 |
| WF-6.3 | Intercompany Reconciliation and Settlement | DDD §6.3 | FR-IC-001, FR-IC-002, FR-IC-003, FR-IC-004, FR-IC-005 | FR-IC-*, FR-PCM-*, FR-FX-*, FR-WFA-*, FR-GL-*, FR-RPT-* | DDD acceptance §14.11 |
| WF-6.4 | Fixed Asset Disposal with Gain or Loss Recognition | DDD §6.4 | FR-FA-005, FR-FA-007, FR-FA-008 | FR-FA-*, FR-WFA-*, FR-GL-*, FR-AP-*, FR-PCM-* | DDD acceptance §§14.2 and 14.13.7 |
| WF-6.5 | Revenue Recognition for a SaaS Contract | DDD §6.5 | FR-AR-001, FR-REV-001, FR-REV-002, FR-REV-003, FR-REV-004, FR-REV-005, FR-REV-006 | FR-REV-*, FR-WFA-*, FR-INV-*, FR-AR-*, FR-GL-*, FR-FX-*, FR-RPT-* | DDD acceptance §§14.12 and 14.13.8 |
| WF-6.6 | Journal Entry Posting and Reversal | DDD §6.6 | FR-GL-001, FR-GL-002, FR-GL-003 | FR-GL-*, FR-WFA-* | DDD acceptance §§14.1 and 14.9 |
| WF-6.7 | Customer Receipt Recording with Partial Application | DDD §6.7 | FR-AR-002, FR-AR-003, FR-AR-004, FR-AR-005 | FR-AR-*, FR-BFR-*, FR-GL-* | DDD acceptance §14.6 |
| WF-7.1 | Vendor Invoice Registration, Matching, Approval, Dispute, and Void | DDD §7.1 | FR-AP-001, FR-AP-006, FR-AP-008, FR-AP-009, FR-AP-010 | FR-AP-*, FR-WFA-*, FR-OMD-* | DDD acceptance §14.13.1 |
| WF-7.2 | Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation | DDD §7.2 | FR-PCM-003 | FR-PCM-*, FR-WFA-*, FR-AP-*, FR-PAYR-*, FR-TAX-*, FR-FA-*, FR-AR-* | DDD acceptance §14.13.2 |
| WF-7.3 | Customer Credit, Refund, Overpayment, Chargeback, and Write-Off | DDD §7.3 | FR-AR-006, FR-AR-007, FR-AR-008, FR-AR-009, FR-AR-010, FR-AR-011, FR-AR-012, FR-AR-013, FR-AR-014, FR-AR-015, FR-AR-016 | FR-AR-*, FR-PCM-*, FR-WFA-* | DDD acceptance §14.13.3 |
| WF-7.4 | Bank Statement Import, Matching, Unmatching, and Reconciliation | DDD §7.4 | FR-BFR-001, FR-BFR-002, FR-BFR-003, FR-BFR-004, FR-BFR-005 | FR-BFR-*, FR-AR-*, FR-AP-*, FR-PCM-* | DDD acceptance §14.13.4 |
| WF-7.5 | Foreign-Currency Invoice Settlement and Realized FX | DDD §7.5 | None named | FR-AP-*, FR-AR-*, FR-FX-*, FR-PCM-*, FR-GL-* | DDD acceptance §14.13.5 |
| WF-7.6 | Period-End Revaluation, Rerun, and Next-Period Reversal | DDD §7.6 | FR-FX-002, FR-FX-003, FR-FX-004 | FR-FX-*, FR-GL-*, FR-FPM-*, FR-WFA-* | DDD acceptance §14.13.6 |
| WF-7.7 | Full Fixed-Asset Lifecycle and Disposal Variants | DDD §7.7 | FR-FA-001, FR-FA-002, FR-FA-003, FR-FA-004, FR-FA-005, FR-FA-006, FR-FA-007, FR-FA-008, FR-FA-018, FR-FA-019, FR-FA-020, FR-FA-021 | FR-FA-*, FR-AP-*, FR-PCM-*, FR-WFA-*, FR-GL-* | DDD acceptance §14.13.7 |
| WF-7.8 | Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration | DDD §7.8 | FR-REV-004, FR-REV-005 | FR-REV-*, FR-AR-*, FR-INV-*, FR-WFA-*, FR-GL-* | DDD acceptance §14.13.8 |
| WF-7.9 | Consolidation, Ownership Changes, Translation, Eliminations, and Rerun | DDD §7.9 | FR-RPT-001, FR-RPT-003, FR-RPT-004 | FR-RPT-*, FR-FX-*, FR-IC-*, FR-WFA-* | DDD acceptance §14.13.9 |
| WF-7.10 | Tax Return Submission, Rejection, Amendment, Payment, and Evidence | DDD §7.10 | FR-TAX-010 | FR-TAX-*, FR-WFA-*, FR-PCM-*, FR-GL-* | DDD acceptance §14.13.10 |
| WF-7.11 | Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment | DDD §7.11 | FR-PAYR-001, FR-PAYR-002, FR-PAYR-003, FR-PAYR-004, FR-PAYR-007 | FR-PAYR-*, FR-WFA-*, FR-PCM-*, FR-TAX-*, FR-GL-* | DDD acceptance §14.13.11 |
| WF-7.12 | Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen | DDD §7.12 | FR-FPM-001, FR-FPM-012, FR-FPM-013, FR-GL-011, FR-GL-012, FR-GL-013 | FR-FPM-*, FR-GL-*, FR-WFA-*, FR-AUD-* | DDD acceptance §14.13.12 |
| WF-7.13 | Cross-Context Event Interpretation, Ordering, and Replay | DDD §7.13 | GFR-006, GFR-007, GFR-008, GFR-009, GFR-012, GFR-013, GFR-014 | Each affected receiving-capability family | DDD acceptance §14.13.13 |
| WF-7.14 | Concurrent Aggregate and Domain-Process Modification Rules | DDD §7.14 | GFR-006, GFR-007, GFR-008, GFR-009, GFR-012, GFR-013, GFR-014 | Each affected owning-capability family | DDD acceptance §14.13.14 |
| WF-7.15 | Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation | DDD §7.15 | FR-AUD-001, FR-AUD-002, FR-AUD-003, FR-AUD-004, FR-AUD-005 | FR-AUD-*, FR-IAM-* | DDD acceptance §14.13.15 |

### 2.1 Capability Coverage Completion

The Version 1.5 review reconciled authoritative records and explicit scenario behavior that were present in the DDD scope and master capability map but lacked standalone functional requirements.

| Coverage area | Explicit requirement IDs | DDD basis |
|---|---|---|
| GL configuration: ledgers, accounting books, charts, and accounts | FR-GL-015–FR-GL-018 | §§1.1, 2.2, 3.1, and 10 |
| AR overpayment, chargeback, and write-off behavior | FR-AR-014–FR-AR-016 | §§1.3, 2.4, 3.2, 7.3, and 14.13.3 |
| Payroll profiles and payroll-source filing records | FR-PAYR-006–FR-PAYR-007 | §§2.5, 7.11, and 11 |
| Payments-owned bank-account records | FR-PCM-025 | §§2.7, 8, 10, and 11.3 |
| Report definitions and ledger/statutory financial statements | FR-RPT-005–FR-RPT-006 | §§1.2, 2.8, 10, and 11 |
| Intercompany agreements and source transactions | FR-IC-010–FR-IC-011 | §§2.9, 3.4, 6.3, 7.9, and 10 |
| Impairment assessment, transfer, split, and posted-disposal correction | FR-FA-018–FR-FA-021 | §§1.3, 2.11, 3.6, 6.4, 7.7, and 14.13.7 |
| Bank-feed connection records | FR-BFR-006 | §§1.1, 2.15, and 11.3 |
| Tax configurations | FR-TAX-016 | §§2.16 and 10 |
| Approval policies | FR-WFA-005 | §§2.17, 8, and 10 |

## 3. Role and Segregation-of-Duties Matrix
| Role / duty | Permitted responsibility | Prohibited combination |
|---|---|---|
| Payment preparer | Prepare and maintain payment batches | Approve the same batch unless an explicitly approved emergency policy permits it |
| Fiscal reopen requester | Request a scoped or operational reopen | Approve the same reopen request |
| Vendor-master administrator | Maintain vendor payment details | Release a payment to newly changed details during the cooling-off period |
| Journal preparer | Create manual journal entries | Approve own journal above the configured threshold |
| Payroll-detail user | Access employee-level payroll data | Receive payroll-detail access solely through summary-ledger permissions |
| Policy administrator | Propose administrative policy changes | Apply unapproved policy changes |
| Emergency-access user | Perform specifically authorized emergency actions | Retain access beyond expiry or use it outside the recorded reason |

## 4. Functional Acceptance Scenarios
### Source DDD §14.1 — GL Posting
- **FAC-14-1-01:** Given a valid open period and balanced lines, when a posting request is submitted, then GL posts exactly one journal entry and returns its reference.
- **FAC-14-1-02:** Given the same accounting scope, idempotency key, and request fingerprint are submitted again, when GL handles the request, then it returns the prior in-progress or terminal result without a duplicate entry.
- **FAC-14-1-03:** Given the same accounting scope and idempotency key are reused with a different request fingerprint, when GL handles the request, then it returns IdempotencyConflict and creates no journal entry.
- **FAC-14-1-04:** Given the period is hard closed, when an ordinary posting request is submitted, then GL rejects it with the authoritative period status.
- **FAC-14-1-05:** Given debit and credit totals differ after currency rounding, when a request is validated, then no journal entry is posted.
- **FAC-14-1-06:** Given a posted entry is reversed, when GL records the equal-and-opposite entry, then the original entry's recorded status and lines remain unchanged, the reversal carries ReversalOfJournalEntryId, and query projections may display the original as reversed.
- **FAC-14-1-07:** Given a journal requires human approval, when the request is accepted, then GL stores it as PendingApproval, emits PostingPendingApproval, and creates no posted ledger effect.
- **FAC-14-1-08:** Given a journal is pending approval and the period, posting gate, account configuration, or authorization changes before approval, when approval is processed, then GL revalidates current conditions, rejects posting when they are no longer valid, and creates no posted entry.
- **FAC-14-1-09:** Given the same approval command is delivered more than once, when GL processes the duplicates, then at most one journal entry is posted and every duplicate receives the existing result.

### Source DDD §14.2 — Fixed Asset Disposal
- **FAC-14-2-01:** Given cost is 100, accumulated depreciation is 60, gross proceeds are 50, and disposal costs are 5 under NoSupplierNetResult, when Fixed Assets posts the required derecognition leg, then proceeds clearing is debited 50, accumulated depreciation is debited 60, asset cost is credited 100, the narrowly scoped disposal-cost accrual is credited 5, gain is credited 5, and Fixed Assets posts neither bank cash nor generic accounts payable.
- **FAC-14-2-02:** Given treatment is WithheldFromProceedsNetResult, when the same disposal posts, then net proceeds clearing is debited 45, accumulated depreciation is debited 60, asset cost is credited 100, net gain is credited 5, and no accrual or separate expense leg exists.
- **FAC-14-2-03:** Given treatment is NoSupplierSeparateExpense, when accounting completes, then the derecognition leg records gross asset-side gain or loss and a second required leg records disposal expense against the narrowly scoped accrual. AccountingStatus becomes Posted only after both leg identifiers have authoritative journal results.
- **FAC-14-2-04:** Given the derecognition leg posts but the required separate-expense leg fails, when Fixed Assets reconciles results, then the disposal is PartiallyPosted, the asset portion remains protected, no AP or Payments downstream intent is published, and recovery retries only the failed leg.
- **FAC-14-2-05:** Given treatment is SupplierInvoiceSeparateExpense, when accounting completes, then Fixed Assets records gross asset-side gain or loss, AP alone records expense and supplier liability, and SupplierLiabilityPosted is the terminal Fixed Assets handoff state.
- **FAC-14-2-06:** Given asset accounting has reached Posted and expected proceeds are observed, when Payments allocates and posts each receipt, then it debits cash and credits proceeds clearing exactly once, Fixed Assets applies the amount, Payments records reconciliation, and the disposed asset is unchanged.
- **FAC-14-2-07:** Given disposal proceeds arrive in multiple bank transactions, when each receipt is allocated, then expectation and AssetDisposal cumulative received and outstanding amounts update once without exceeding authorized proceeds.
- **FAC-14-2-08:** Given net proceeds are below carrying amount, when disposal posts, then the signed difference is debited to loss on disposal.
- **FAC-14-2-09:** Given an asset is fully depreciated and scrapped for zero proceeds, when disposal posts, then cost and accumulated depreciation are derecognized with no gain or loss except costs recognized by the selected treatment.
- **FAC-14-2-10:** Given GL posts one or more required legs but Fixed Assets is interrupted before recording completion, when recovery queries each stable leg identifier and request fingerprint, then existing results are retained, nonterminal legs alone retry, and no duplicate entry is created.
- **FAC-14-2-11:** Given an asset portion is DisposalPending, when depreciation, transfer, impairment, or another disposal targets it, then the command is rejected until the accounting posting set reaches an authorized terminal state.
- **FAC-14-2-12:** Given two disposal commands concurrently target the same asset component or quantity, when both validate the same starting version, then the unique active-disposal constraint allows at most one to succeed.
- **FAC-14-2-13:** Given accounting is PendingPosting, PartiallyPosted, or PostingFailed, when downstream processing runs, then no proceeds expectation, supplier classification, or cost-payment request is emitted.
- **FAC-14-2-14:** Given a posted disposal has a failed receipt or payment instruction, when recovery executes, then only orthogonal settlement status and linked correction records change; posted asset accounting remains immutable.
- **FAC-14-2-15:** Given settled disposal proceeds are later reversed, when the canonical IncomingSettlementReversed event classified as asset proceeds is applied, then net proceeds settlement decreases, outstanding proceeds increase, and status returns to Expected or PartiallySettled without changing asset derecognition.
- **FAC-14-2-16:** Given a settled no-supplier disposal-cost payment is returned, when the canonical PaymentInstructionReturned event classified as disposal cost and the owner acknowledgement are applied, then net cost settlement decreases, outstanding cost increases, status returns to PaymentRequested or PartiallySettled, and any replacement request uses a new instruction reference.
- **FAC-14-2-17:** Given a proceeds or disposal-cost settlement is Failed, when the original retry, replacement, or authoritative reconciliation succeeds, then Fixed Assets derives Expected, PaymentRequested, PartiallySettled, or Settled from the current net-settled and outstanding amounts without changing disposal accounting.
- **FAC-14-2-18:** Given the same canonical correction event is observed more than once, when Fixed Assets applies its event-identity rule, then the monetary reversal or return is applied exactly once.
- **FAC-14-2-19:** Given no disposal posting leg has been admitted and cancellation is approved, when CancelUnpostedAssetDisposal establishes that no posting leg can still establish a journal and GL confirms no journal for every request identifier and request fingerprint, then Fixed Assets restores the asset portion exactly once, records CancelledNoJournal, and emits no AP or Payments obligation.
- **FAC-14-2-20:** Given one disposal leg posted and another leg is irrecoverably rejected, when CompensateFailedDisposalPosting completes, then every successful leg has one linked reversal, the asset portion is restored only after all reversals post, the disposal reaches CompensatedFailed, and no downstream settlement obligation exists.

### Source DDD §14.3 — Hard Close
- **FAC-14-3-01:** Given all mandatory steps and approvals are complete, when hard close finishes, then the period is hard closed, the final ledger watermark is recorded, and subsequent ordinary postings are rejected.
- **FAC-14-3-02:** Given any gate ownership-acquisition command succeeds, when the ownership transition succeeds, then GL initializes ControlOwnerType, ControlOwnerId, and the applicable ControlOwnerEpoch, sets both admitted flags false, sets count to zero, and clears first and last admitted positions.
- **FAC-14-3-03:** Given a gate closes, releases, hands off, or finalizes, when GL accepts the command, then it freezes and returns the outgoing owner's immutable FrozenGateAdmissionSummary, and Fiscal Period Management records that summary exactly once on the controlling aggregate.
- **FAC-14-3-04:** Given an Open gate and an eligible soft-close policy, when EnterSoftCloseGate succeeds, then GL records ActiveSoftCloseRunId, applies the policy at journal append, increments the gate version, and Fiscal Period Management records SoftClosed.
- **FAC-14-3-05:** Given the same soft-close command is delivered again with the same policy version and fingerprint, when GL handles it, then it returns the existing gate result without another transition; conflicting policy content returns a domain conflict.
- **FAC-14-3-06:** Given an active soft-close run and epoch with no hard-close handoff, when ExitSoftCloseGate(softCloseRunId, softCloseControlEpoch) succeeds, then GL freezes an epoch-qualified summary, restores Open, clears the matching owner and epoch, and an ambiguous response is recoverable through GetPostingGateStatus.
- **FAC-14-3-07:** Given an active soft-close gate and a candidate hard-close run, when AcquirePostingBarrier(softCloseRunId, closeRunId) succeeds, then GL clears the soft-close owner and sets the close owner as one exclusive domain consistency outcome, records the prior process and barrier position, and exposes no gate version with two owners; Fiscal Period Management records HandoffPending, BarrierAcquired, and Closing as one named consistency outcome.
- **FAC-14-3-08:** Given ExitSoftCloseGate and AcquirePostingBarrier concurrently target the same gate version, when GL evaluates them, then exactly one wins: exit restores Open, or handoff establishes the hard-close owner; the losing command receives the authoritative version and creates no local lifecycle change.
- **FAC-14-3-09:** Given the handoff outcome acknowledgement is missing or ambiguous, when recovery queries the gate, then it either observes the original soft-close owner or the authoritative hard-close owner and completes the matching local Active or HandoffPending lifecycle exactly once.
- **FAC-14-3-10:** Given an ordinary posting was validated against the prior posting-gate version but has not established a journal when GL acquires the close barrier, when GL attempts to append it, then authoritative gate-version validation at posting admission rejects it and no journal entry is created.
- **FAC-14-3-11:** Given a required close posting fails, when the close run stops, then the period does not become hard closed and the failed step is resumable after in-flight postings are reconciled.
- **FAC-14-3-12:** Given GL finalized the posting gate but Fiscal Period Management failed before recording HardClosed, when the domain process queries the gate, then it receives the authoritative close-run identifier, finalized gate version, and ledger watermark and completes the period transition idempotently.
- **FAC-14-3-13:** Given a barrier was acquired and the authoritative gate record has ClosePostingAdmitted = false with zero admissions for the active close run, when an approved close abort calls ReleasePostingBarrier, then GL restores the prior gate mode and records the release.
- **FAC-14-3-14:** Given a soft-close run is handed off, the close barrier is released, and a second hard-close attempt begins, when the second AcquirePostingBarrier succeeds, then Fiscal Period Management preserves the first frozen SoftCloseControlEpoch, freezes a distinct second epoch, and no prior summary is overwritten.
- **FAC-14-3-15:** Given any close-authorized posting was appended and GL recorded the process identifier as part of the same journal-admission outcome, positive admitted count, and ledger position, when release is requested, then GL rejects the release and the close must resume or follow an approved recovery process.
- **FAC-14-3-16:** Given BeginRecloseGate established CloseType = Reclose, when any barrier release is requested, then GL rejects it regardless of whether the reclose itself has posted an additional close adjustment, because reopen postings already made finalization mandatory.
- **FAC-14-3-17:** Given all close postings have completed but the audit-seal outcome is not yet available, when hard close completes, then the period remains hard closed, an immutable seal request exists, SealStatus is SealPending or SealFailed, and repeated seal evaluation remains idempotent.
- **FAC-14-3-18:** Given the close run is completed while sealing is pending, when a reporting or control projection evaluates the close result, then it receives the final ledger watermark and the non-success seal status rather than an assertion that proof already exists.
- **FAC-14-3-19:** Given a duplicate seal result is delivered, when Fiscal Period Management records it, then the same proof reference is retained idempotently and a conflicting proof for the same request is rejected.
- **FAC-14-3-20:** Given a hard-closed period has an active scoped reopen, when an ordinary posting is submitted, then GL rejects it even though correction postings may be admitted.
- **FAC-14-3-21:** Given a hard-closed period is reopened and corrected, when it is reclosed, then the period returns through Reopening and Closing to HardClosed and both the original and revised seals remain verifiable.

### Source DDD §14.4 — Accounting Ownership
- **FAC-14-4-01:** Given a finalized customer invoice, when accounting is created, then AR submits exactly one receivable and billing posting and Revenue Recognition does not duplicate that posting.
- **FAC-14-4-02:** Given scheduled revenue becomes recognizable, when the recognition run executes, then Revenue Recognition submits the recognition or reclassification posting and AR does not duplicate it.
- **FAC-14-4-03:** Given a payment instruction settles at the bank, when settlement is confirmed, then Payments submits the authoritative cash-settlement posting and AP updates invoice state from the settlement event without posting the same cash effect again.
- **FAC-14-4-04:** Given a customer receipt is partially applied, when applications change, then each authoritative ReceivableOpenItem creates its owned immutable ReceiptApplication, CustomerReceipt changes its applied and unapplied balances and creates one ReceiptApplicationBatch as part of the same AR domain outcome, and customer-invoice balances are refreshed idempotently from open-item events.
- **FAC-14-4-05:** Given an asset acquisition or disposal creates a supplier liability or bank settlement, when accounting is produced, then Fixed Assets posts only asset-side and asset-specific clearing or permitted accrual effects, AP alone posts supplier liabilities, and Payments alone posts bank cash.
- **FAC-14-4-06:** Given AR authorizes a customer refund, when payment execution begins, then AR alone posts refund payable and payment settlement clearing, Payments alone posts external bank cash and any linked return, and AR updates the refund only from authoritative instruction outcomes.
- **FAC-14-4-07:** Given an intercompany settlement produces an approved residual and outgoing or incoming cash settlement, when accounting is created, then Intercompany posts residual and due-to or due-from settlement-clearing effects, Payments posts only the corresponding bank-cash legs, and every required PaymentInstruction and SettlementReceipt is reconciled without duplication.
- **FAC-14-4-08:** Given elimination instructions are created, when consolidation runs, then Financial Reporting owns the elimination records and statutory GL remains unchanged unless a separately approved consolidation-ledger contract applies.

### Source DDD §14.5 — Scope and Published Contracts
- **FAC-14-5-01:** Given an accounting scope references a ledger and accounting book, when a posting is validated, then GL verifies that the book belongs to the ledger, the ledger belongs to the legal entity, and functional currency and effective dates are consistent.
- **FAC-14-5-02:** Given a close, depreciation, revaluation, intercompany, or consolidation run is created, when it executes, then its accounting, participant, or consolidation scope is recorded explicitly and is not inferred from ambient context.
- **FAC-14-5-03:** Given AR accounts for an invoice line, when it selects revenue classification, then it records the immutable RevenueAccountingProfileId and profile version used.
- **FAC-14-5-04:** Given the required revenue profile is missing, expired, or inconsistent, when AR attempts invoice accounting, then finalization or posting is blocked and no default account is silently selected.

### Source DDD §14.6 — Receipt Application Concurrency
- **FAC-14-6-01:** Given two receipts concurrently target the final available amount of the same receivable open item, when both commands execute, then at most one succeeds and the other receives a version or insufficient-balance conflict.
- **FAC-14-6-02:** Given one receipt is allocated across multiple open items, when any open item has a stale version or insufficient balance, then no allocation from that command is established.
- **FAC-14-6-03:** Given the same application command is delivered twice with the same command fingerprint, when AR handles the duplicate, then it returns the prior result and neither receipt nor open-item balances change again.
- **FAC-14-6-04:** Given an application command reuses its idempotency key with different allocations, when AR handles it, then it returns an idempotency conflict and changes no aggregate.
- **FAC-14-6-05:** Given two commands coordinate overlapping receipts and open items, when they execute, then the all-or-nothing consistency rule permits at most one valid outcome and no partial application.
- **FAC-14-6-06:** Given a new receipt is recorded without confirmed allocations, when the record-receipt flow completes, then the full amount remains unapplied and no application posting is created.
- **FAC-14-6-07:** Given a new receipt is recorded, when AR creates accounting, then it debits cash or bank clearing and credits unapplied cash exactly once.
- **FAC-14-6-08:** Given a posted receipt is partially applied, when AR creates application accounting, then it debits unapplied cash and credits accounts receivable for the applied amount without debiting cash again.
- **FAC-14-6-09:** Given application balances are established but the application posting outcome is uncertain, when AR retries with the same posting identifier and command fingerprint, then GL returns the existing result or posts once and neither the receipt nor open-item balances change again.
- **FAC-14-6-10:** Given all allocations in a posted application batch are unapplied, when UnapplyReceipt completes, then AR restores the affected balances as one domain consistency outcome and submits exactly one linked reversal of the batch posting.
- **FAC-14-6-11:** Given only part of a posted application batch is unapplied, when UnapplyReceipt completes, then AR restores only the selected balances and submits one linked compensating posting for the exact amount without reversing unrelated allocations.
- **FAC-14-6-12:** Given an application batch is PostingPending, PostingFailed, or CancellingNoJournal, when UnapplyReceipt executes, then AR rejects it without changing receipt or open-item balances.
- **FAC-14-6-13:** Given rollback begins while an application posting attempt is in progress, when AR records a new cancellation boundary, then no later posting attempt may use the superseded attempt identity; AR waits for the prior posting attempt to resolve before deciding whether a journal exists.
- **FAC-14-6-14:** Given GL admits the application posting before cancellation evidence becomes authoritative, when rollback reconciliation checks the authoritative GL result, then AR records the journal as Posted and requires normal UnapplyReceipt with one reversal or compensating entry.
- **FAC-14-6-15:** Given retry policy declares an application posting terminal, all prior posting attempts are resolved, and GL confirms no journal was admitted, when RollbackUnpostedApplicationBatch succeeds, then AR restores receipt and open-item balances exactly once, appends immutable ReceiptApplicationRollback facts, marks the batch CancelledNoJournal, and creates no reversal.
- **FAC-14-6-16:** Given a required receipt or application posting remains failed at period close, when close controls evaluate outstanding accounting, then the affected step is blocked or proceeds only through an explicitly approved and audited exception.
- **FAC-14-6-17:** Given receipt-recording accounting is pending or failed, when ApplyReceipt is submitted, then AR rejects the application and creates no allocation facts or application batch.
- **FAC-14-6-18:** Given one ApplyReceipt command contains several allocations, when it succeeds, then all allocations belong to one ApplicationBatchId, exactly one application-accounting request is emitted for the batch total, and line references retain allocation traceability.
- **FAC-14-6-19:** Given the same UnapplyReceipt command is delivered twice with the same command fingerprint, when AR handles the duplicate, then it returns the prior result and neither balances nor adjustment records change again.
- **FAC-14-6-20:** Given an unapplication command reuses its idempotency key or UnapplicationBatchId with different applications, amounts, or expected versions, when AR handles it, then it returns an idempotency conflict and changes no aggregate.
- **FAC-14-6-21:** Given two unapplication commands concurrently target the remaining adjustable amount of the same application, when both execute, then at most one succeeds and cumulative ReceiptApplicationAdjustment amounts never exceed the original application amount.
- **FAC-14-6-22:** Given an unapplication succeeds, when its records are inspected, then the original ReceiptApplication and ReceiptApplicationBatch remain unchanged and the adjustment and unapplication batch retain complete linkage to the original facts and accounting result.

### Source DDD §14.7 — Posting Currency Semantics
- **FAC-14-7-01:** Given a posting request declares one transaction currency and every ordinary line uses TransactionAndFunctional amounts in that currency, when debit and credit totals balance in transaction and functional currency, then GL may post the entry.
- **FAC-14-7-02:** Given an authorized settlement request includes a FunctionalOnlyAdjustment line with zero transaction amount, a nonzero functional amount, immutable rate evidence, and balanced request totals in both currency views, when GL validates it, then GL posts it without creating a source-currency quantity or transaction-currency balance.
- **FAC-14-7-03:** Given a line has zero transaction amount but is not an authorized functional-only or statistical line, when GL validates it, then GL rejects the request.
- **FAC-14-7-04:** Given a functional-only line has a nonzero transaction amount, missing functional amount, unsupported account policy, or insufficient calculation evidence, when GL validates it, then GL rejects the request.
- **FAC-14-7-05:** Given a posting request contains a line denominated in a currency different from the request header, when GL validates it, then GL rejects the request and creates no journal entry.
- **FAC-14-7-06:** Given one business event requires accounting in two transaction currencies, when the producer submits accounting, then it sends separate correlated posting requests with an explicit settlement or clearing reference.

### Source DDD §14.8 — Fiscal-Period Scope and Exclusivity
- **FAC-14-8-01:** Given two accounting books share the same fiscal calendar period, when one book is hard closed, then the other book's scoped fiscal-period state is unchanged.
- **FAC-14-8-02:** Given any close or reopen process owns the posting gate for a scoped fiscal period, when another period-control process is started, then the second request is rejected or returns the existing active process.
- **FAC-14-8-03:** Given a scoped fiscal-period identifier is used with a different accounting scope, when a command is validated, then it is rejected without changing the period or posting gate.

### Source DDD §14.9 — Idempotent Event Handling and Duplicate Delivery
- **FAC-14-9-01:** Given a receiving bounded context observes a valid domain event but does not establish a local outcome, when the event is redelivered, then the event can be processed normally and no partial business effect is visible.
- **FAC-14-9-02:** Given a receiving bounded context has already established the event's local domain outcome, when the same event identity is redelivered, then the receiving bounded context returns the prior outcome and repeats no business side effect.
- **FAC-14-9-03:** Given an event outcome is initially ambiguous, when reconciliation is performed, then the receiving bounded context determines whether the domain effect exists before retrying and never treats the event as complete before the effect is established.

### Source DDD §14.10 — Fiscal Reopen and Reclose
- **FAC-14-10-01:** Given a hard-closed scoped period and an independently approved request, when OpenScopedReopenGate succeeds, then only authorized ReopenCorrection postings for the request and unexpired scope are admitted.
- **FAC-14-10-02:** Given authorization expires between caller validation and journal append, when GL performs its local gate check, then it rejects the posting and creates no journal.
- **FAC-14-10-03:** Given Fiscal Period Management is interrupted after GL opens or closes the gate, when recovery uses GetPostingGateStatus, then it completes the corresponding local transition exactly once.
- **FAC-14-10-04:** Given the same reversal or replacement command is delivered twice with the same fingerprint, when the producer and GL handle it, then only one correction journal exists.
- **FAC-14-10-05:** Given the authoritative gate record identifies the active request with ReopenPostingAdmitted = false and zero admissions, when CloseScopedReopenGate completes under no-change policy, then GL restores HardClosed, Fiscal Period Management records CompletedNoChange as one named consistency outcome, the prior watermark and financial close seal remain authoritative, and immutable no-admission evidence is retained.
- **FAC-14-10-06:** Given authorization expires, when Fiscal Period Management records ExpiredPendingClosure, then new postings remain rejected while the request continues to gate closure. If authoritative retained evidence shows any correction posted, reclose is mandatory; otherwise no-change closure is allowed.
- **FAC-14-10-07:** Given a candidate reclose is created and the gate contains a positive admission summary for that reopen request, when BeginRecloseGate succeeds, then Fiscal Period Management changes ReopenRequest to RecloseInProgress, CloseRun to BarrierAcquired, and FiscalPeriod to Closing as one named consistency outcome; the reclose barrier cannot be released.
- **FAC-14-10-08:** Given reclose completes, when reports and seals are inspected, then prior and revised watermarks, close runs, and proofs are retained and linked.

### Source DDD §14.11 — Intercompany Settlement
- **FAC-14-11-01:** Given reciprocal eligible items within tolerance, when matching completes, then each item is reserved once and included in one approved settlement snapshot.
- **FAC-14-11-02:** Given a residual equals the auto-approval boundary, when policy is evaluated, then the configured inclusive or exclusive rule is applied using decimal currency precision.
- **FAC-14-11-03:** Given two runs concurrently reserve the same open item, when both attempt the transition, then at most one succeeds and the other receives a reservation or version conflict.
- **FAC-14-11-04:** Given outgoing payment, incoming settlement-receipt, and reporting acknowledgements arrive out of order or twice, when Intercompany consumes them, then PaymentInstruction, SettlementReceipt, and settlement-run statuses advance monotonically and no clearing, cash, or settlement effect repeats.
- **FAC-14-11-05:** Given one outgoing payment instruction, expected incoming settlement, or observed receipt is missing, rejected, in exception, or posting-failed while others settle, when the run is evaluated, then it remains PartiallySettled, retains item-level states and evidence, and cannot complete until each required instruction and expectation is reconciled, or each receipt is Reconciled or resolved by an approved terminal exception.
- **FAC-14-11-06:** Given a completed run is corrected, when a new run is created, then prior postings, instructions, eliminations, and approvals remain immutable and linked.

### Source DDD §14.12 — Revenue Recognition
- **FAC-14-12-01:** Given a 12,000 USD annual subscription billed in advance, when AR bills and twelve monthly recognition runs complete, then receivable, contract liability, and revenue reconcile to zero remaining liability and 12,000 recognized revenue.
- **FAC-14-12-02:** Given service is recognized before billing, when three monthly runs and the quarterly invoice complete, then the contract asset is cleared by AR without duplicate revenue.
- **FAC-14-12-03:** Given the same schedule period is run twice, when the second command has the same fingerprint, then Revenue Recognition returns the prior result and recognized-to-date does not change again.
- **FAC-14-12-04:** Given a modification and recognition run race on the same schedule version, when expected versions are checked, then only one succeeds and the loser recalculates from the authoritative recognized-to-date amount.
- **FAC-14-12-05:** Given AR lacks a valid profile version, when invoice accounting is attempted, then finalization is blocked and no default account is selected.
- **FAC-14-12-06:** Given a recognition-posting acknowledgement is missing or ambiguous, when the same posting identifier is repeated, then GL returns the authoritative result and the schedule period advances once.

### Source DDD §14.13 — Additional Scenario Acceptance Criteria
- This source heading groups the detailed scenario-specific criteria that follow.

#### Source DDD §14.13.1 — Vendor Invoice
- **FAC-14-13-1-01:** Normal: A valid invoice with immutable matching snapshots and an applied Workflow approval decision posts one liability and retains its evidence versions.
- **FAC-14-13-1-02:** Boundary: An invoice at the configured tolerance boundary follows the policy's inclusive or exclusive rule; a conflicting duplicate fingerprint is quarantined.
- **FAC-14-13-1-03:** Concurrency: Concurrent matching, dispute, approval-application, or void commands produce one versioned transition and no lost update.
- **FAC-14-13-1-04:** Duplicate delivery: An exact duplicate registration or posting event returns the existing invoice or posting result without another liability.
- **FAC-14-13-1-05:** Failure and recovery: A lost GL response is reconciled by the original posting identifier; void after settlement is rejected in favor of an explicit correction flow.

#### Source DDD §14.13.2 — Payment Execution
- **FAC-14-13-2-01:** Normal: An independently approved batch snapshots instruction versions and control totals; each instruction maintains authorized, settled, cancelled, and remaining amounts. The batch finishes as Completed with an explicit outcome. Incoming cash uses one expectation, one receipt per bank allocation, the named expectation-and-receipt consistency rule, posted cash, owning-context application, and Payments reconciliation.
- **FAC-14-13-2-02:** Boundary: AuthorizedMoney = SettledMoney + CancelledMoney + RemainingMoney, NetSettledMoney = SettledMoney - PostedReturnMoney + ReversedReturnMoney, ReservedReturnMoney + PostedReturnMoney - ReversedReturnMoney <= SettledMoney, and ReversedReturnMoney + ReconciledReturnMoney <= PostedReturnMoney; partial cancellation reaches PartiallySettledCancelled; whole-batch cancellation is allowed only before provider submission; CompletedWithExceptions requires resolved instruction exceptions; expectation expiry preserves partial receipts; ExpectedMoney = ReceivedMoney + RemainingMoney; excess bank money remains separately unallocated; and expectation cancellation requires zero received money.
- **FAC-14-13-2-03:** Concurrency: Concurrent partial outcomes update instruction balances once. Return reservation, posting, reversal, and acknowledgement coordinate PaymentInstruction before PaymentReturn, enforce a unique provider-return key and reversal-safe cumulative ceilings, and move amounts only through the declared reserved, posted, reversed, and reconciled balances. Allocation, acknowledgement, rollback, exception resolution, and reversal coordinate ExpectedIncomingSettlement before SettlementReceipt; expected versions, unique keys, authoritative posting-cancellation evidence, and typed conflicts ensure one valid outcome wins.
- **FAC-14-13-2-04:** Duplicate delivery: Repeated submission, expectation, bank observation, posting, owner application, acknowledgement, cancellation, exception resolution, rollback, provider return, or batch completion identifiers return prior results without another obligation, receipt, allocation, cash journal, clearing application, balance restoration, or return.
- **FAC-14-13-2-05:** Failure and recovery: Instruction, return, validation and expectation exceptions have typed resolution commands. Owner application rejection after posting can proceed to corrected application, approved reclassification, accepted exception, or a linked return reversal. A terminally unposted receipt or a return in PostingFailed uses CancellingNoJournal, GL no-journal proof, and one immutable allocation or return-reservation rollback; a posted return uses ReversalPending or ReversalFailed until an authoritative reversal reaches Reversed. Failed partial payment preserves settled amount and exposes the unpaid or cancelled remainder; provider returns reserve without reducing net settlement, move to posted exactly once, require owner acknowledgement or typed exception resolution, and remain within the reversal-safe gross-settlement ceiling.
- **FAC-14-13-2-06:** Given two provider-return records concurrently target the same remaining gross-settled amount, when the named instruction-and-return consistency rule is applied, then at most one reserves the amount and cumulative active returns never exceed gross settlement.
- **FAC-14-13-2-07:** Given a validated return reservation never produces a GL journal, when CancelUnpostedPaymentReturn establishes that prior posting attempts cannot still succeed and GL proves no journal, then the reservation is released once, the instruction reserved-return amount decreases, and the immutable return reaches CancelledNoJournal.
- **FAC-14-13-2-08:** Given a return cash correction is posted and the obligation owner applies it, when AcknowledgePaymentReturn runs, then ReconciledReturnMoney advances once and the return becomes Reconciled; owner rejection leaves a visible exception.
- **FAC-14-13-2-09:** Given a bank allocation exceeds an expectation remaining amount, when Payments records it, then no expectation balance becomes negative, the allowed amount is separately allocated, and the excess posts once to unallocated incoming cash clearing before an explicit allocation, refund, or reclassification resolution.
- **FAC-14-13-2-10:** Given a return is reserved but its cash-correction journal is still pending, when the instruction state is evaluated, then NetSettledMoney remains based on posted returns only and no owner return event is published.
- **FAC-14-13-2-11:** Given an obligation owner rejects a posted return application, when ResolvePaymentReturnException is applied, then exactly one typed outcome and immutable evidence determine whether the return resumes acknowledgement, reconciles by approved reclassification, reaches accepted exception, or enters ReversalPending and reaches Reversed only after the linked reversal is authoritative.
- **FAC-14-13-2-12:** Given ReturnRejectedWithReversal is selected for an unreconciled posted return, when the linked reversal becomes authoritative, then ReversedReturnMoney increases once, NetSettledMoney is restored by the reversed amount, the return reaches Reversed, and no no-journal cancellation path is permitted.
- **FAC-14-13-2-13:** Given a payment instruction fails terminally, when resolution is requested, then Payments cannot resolve it until the owning context supplies PaymentInstructionExceptionDecisionRecorded for the exact amount and owner version.

#### Source DDD §14.13.3 — Customer Adjustments
- **FAC-14-13-3-01:** Normal: Credit, refund authorization, overpayment treatment, chargeback, and write-off produce separately owned accounting effects. An approved CustomerRefundRequest posts refund payable to payment clearing, publishes one refund-payment obligation, and reaches settlement only from authoritative Payments outcomes.
- **FAC-14-13-3-02:** Boundary: Cumulative credit or write-off cannot exceed open receivable; refund cannot exceed refundable unapplied cash or approved credit; approval rejection reaches Rejected; requester withdrawal or pre-payment cancellation reaches Cancelled; a remainder-cancelled result reaches Cancelled when gross settlement is zero and PartiallySettledCancelled when gross settlement is positive; AuthorizedMoney = NetSettledMoney + CancelledMoney + RemainingMoney; NetSettledMoney = GrossSettledMoney - ReturnedMoney; and a replacement preserves prior clearing, instruction and return references.
- **FAC-14-13-3-03:** Concurrency: Overlapping adjustments coordinate receipt and open items deterministically. Concurrent refund settlement, cancellation, replacement, or return outcomes use expected refund and instruction versions so amounts advance once.
- **FAC-14-13-3-04:** Duplicate delivery: The same adjustment, refund request, payment obligation, settlement-outcome, replacement, or return identifier and fingerprint returns the prior result without changing balances again.
- **FAC-14-13-3-05:** Failure and recovery: Failed accounting or payment retains visible pending state and recovers using the original identifier. A cancelled remainder restores the unpaid refund obligation or creates a linked replacement request. A provider return enters ReturnCorrectionPending; a failed correction becomes ReturnCorrectionPostingFailed and retries the same posting identifier and request fingerprint, while an approved irrecoverable case becomes ReturnCorrectionException. Only after the AR clearing-to-refund-payable correction posts do returned and remaining amounts increase, net settlement decrease, owner acknowledgement publish, and replacement processing begin.

#### Source DDD §14.13.4 — Bank Reconciliation
- **FAC-14-13-4-01:** Normal: A validated statement can be matched one-to-one, split, aggregate, or manually and completes only when balances and approved differences reconcile.
- **FAC-14-13-4-02:** Boundary: Split allocations equal the statement-line amount within the configured precision and tolerance; opening-balance exceptions require approval.
- **FAC-14-13-4-03:** Concurrency: Two sessions cannot consume the same unmatched amount, and expected line versions prevent overlapping confirmation.
- **FAC-14-13-4-04:** Duplicate delivery: A duplicate statement fingerprint or match-confirm event returns or quarantines the existing result without another match effect.
- **FAC-14-13-4-05:** Failure and recovery: Partial import remains unaccepted; unmatch appends compensating history and triggers the owning-context correction rather than deleting the original match.

#### Source DDD §14.13.5 — FX Settlement
- **FAC-14-13-5-01:** Normal: AR clears customer receipt, receivable, unapplied cash, and realized FX to zero; for AP settlement, AP clears payable and realized FX through payment clearing while Payments owns the bank-cash leg. Every request identifies one transaction currency and separately records permitted functional amounts.
- **FAC-14-13-5-02:** Boundary: Settlement cannot exceed the open item; functional-only lines require zero transaction amount, nonzero functional amount, authorized account policy, and immutable rate evidence; transaction and functional totals both balance, source-currency open items reach zero, and no residual transaction-currency balance is introduced.
- **FAC-14-13-5-03:** Concurrency: Concurrent AR applications or AP settlement allocations use expected open-item versions so at most one consumes the remaining amount.
- **FAC-14-13-5-04:** Duplicate delivery: Duplicate receipt, payment-settlement, or realized-FX evidence returns the prior calculation and posting results.
- **FAC-14-13-5-05:** Failure and recovery: Unresolved clearing remains visible to close controls; ambiguous outcomes are reconciled by stable identifiers, and settlement reversal creates linked corrections.

#### Source DDD §14.13.6 — Revaluation
- **FAC-14-13-6-01:** Normal: Workflow records one immutable decision, Multi-Currency applies it to move the current run to Approved, and PostRevaluationRun separately assigns stable posting identifiers and moves it to Posting before one approved run values each eligible monetary balance exactly once and schedules any required reversal.
- **FAC-14-13-6-02:** Boundary: Zero differences create no posting unless policy requires statistical records; rounding residuals follow the configured account and precision.
- **FAC-14-13-6-03:** Concurrency: Only one active run exists per scope, period, policy, and source watermark; changed inputs create a new version.
- **FAC-14-13-6-04:** Duplicate delivery: The same approval, run, posting-start, and journal identifiers return existing decision-application, calculation, posting-intent, and journal results.
- **FAC-14-13-6-05:** Failure and recovery: Partial posting and next-period reversal failures remain visible and resume by stable identifiers; rerun uses either reversal or delta treatment, never both.

#### Source DDD §14.13.7 — Fixed-Asset Lifecycle
- **FAC-14-13-7-01:** Normal: Acquisition, capitalization, depreciation, transfer, split, impairment, disposal, cancellation, and correction preserve cost and carrying-amount equations by component. A valid treatment determines a fixed required-leg set, all legs post before downstream settlement obligations are published, AP alone owns supplier liabilities, and Payments alone owns bank cash.
- **FAC-14-13-7-02:** Boundary: NoCost requires zero disposal cost; the other four treatments require compatible evidence. Fully depreciated, zero-proceeds, partial-component, withheld-cost, no-supplier net or separate-expense, supplier classification, partial proceeds, partial no-supplier payment, reversed proceeds, and returned cost-payment cases preserve explicit asset-side, gross, reversal/return, net-settlement and outstanding equations.
- **FAC-14-13-7-03:** Concurrency: Protected operations on the same asset portion result in one winner. Concurrent posting-leg results update the same disposal version without allowing Posted until every required leg is authoritative.
- **FAC-14-13-7-04:** Duplicate delivery: Repeated lifecycle commands, posting-leg requests, expectations, receipts, owner acknowledgements, supplier-liability results, and payment outcomes return existing outcomes without duplicate asset, allocation, clearing, cash, liability, expense, gain/loss, or GL effects.
- **FAC-14-13-7-05:** Failure and recovery: If no leg posts, evidence-backed no-journal cancellation may restore the asset and reach CancelledNoJournal. If one leg posts and another is irrecoverable, linked compensation reverses every successful leg before asset restoration and CompensatedFailed; failed reversals keep the asset protected. Ordinary retry retains successful legs and retries only nonterminal ones. Posted-disposal correction reverses all posted asset-side legs without duplicating separately owned settlement effects. Receipt reversal and payment return reopen only the corresponding proceeds or no-supplier cost obligation through typed events and new replacement references.

#### Source DDD §14.13.8 — Revenue Modifications
- **FAC-14-13-8-01:** Normal: Separate-contract, prospective, and cumulative-catch-up conclusions create distinct versioned schedules and profiles while preserving prior recognition.
- **FAC-14-13-8-02:** Boundary: Recognition never exceeds constrained allocated consideration, and effective-date boundaries select exactly one applicable profile version.
- **FAC-14-13-8-03:** Concurrency: Concurrent invoice, recognition, and modification work cannot establish incompatible schedule, profile, or recognized-to-date versions.
- **FAC-14-13-8-04:** Duplicate delivery: Repeated modification, profile-publication, recognition, or invoice-consumption commands return prior results.
- **FAC-14-13-8-05:** Failure and recovery: Failed catch-up remains pending; cancellation or refund coordinates separately owned Revenue Recognition and AR corrections without overwriting history.

#### Source DDD §14.13.9 — Consolidation
- **FAC-14-13-9-01:** Normal: Multi-Currency publishes versioned translation results, Workflow records the publication decision, Financial Reporting applies it to move the frozen consolidation run to Approved, and PublishConsolidatedStatement records the active published statement exactly once.
- **FAC-14-13-9-02:** Boundary: Ownership effective dates, rate boundaries, elimination tolerances, and noncontrolling-interest calculations follow frozen policy versions.
- **FAC-14-13-9-03:** Concurrency: Only one active publication version exists per scope and period; changed watermarks, rates, translation results, or mappings create a new run.
- **FAC-14-13-9-04:** Duplicate delivery: Identical frozen inputs and approval-decision delivery return the existing run and applied decision and do not duplicate CTA, elimination, or publication records.
- **FAC-14-13-9-05:** Failure and recovery: Missing participant data, translation result, or balanced elimination blocks publication; restart resumes recorded domain-process states and published correction creates a new statement version.

#### Source DDD §14.13.10 — Tax Filing
- **FAC-14-13-10-01:** Normal: Workflow-approved returns and amendments are submitted as separate lineage aggregates whose accepted versions are immutable; source subledgers own transaction-tax corrections, applying the Workflow decision moves ReturnLevelTaxAdjustment to Approved, a separate posting command creates one authoritative PostingPending result, and Payments owns only the tax bank-cash leg.
- **FAC-14-13-10-02:** Boundary: An amendment requires an accepted original and never changes that original's authoritative state; a return-level adjustment requires an approved source version; payment cannot exceed the outstanding obligation; payment failure does not regress filing acceptance.
- **FAC-14-13-10-03:** Concurrency: Concurrent amendment, return-level adjustment, and payment commands validate accepted-return, amendment, adjustment, and obligation versions and cannot establish incompatible source versions.
- **FAC-14-13-10-04:** Duplicate delivery: Identical approval, filing, amendment, adjustment-posting, or payment keys and fingerprints return existing results; conflicting reuse is rejected.
- **FAC-14-13-10-05:** Failure and recovery: An uncertain authority outcome is reconciled before another submission; rejection creates a corrected attempt or amendment version; an uncertain GL outcome is reconciled by the adjustment's stable posting identifier; failed payment leaves an outstanding obligation and uses explicit retry or replacement.

#### Source DDD §14.13.11 — Payroll Corrections
- **FAC-14-13-11-01:** Normal: Workflow records one immutable payroll decision, Payroll applies it to the current calculated run version, and regular, off-cycle, and correction runs balance gross, deductions, liabilities, and net pay before producing separately owned payment and filing effects.
- **FAC-14-13-11-02:** Boundary: Zero-net, negative correction, statutory limit, and rounding cases follow policy without violating gross-minus-deductions-equals-net.
- **FAC-14-13-11-03:** Concurrency: Employee lines validate profile and prior-result versions so overlapping corrections cannot overwrite one another.
- **FAC-14-13-11-04:** Duplicate delivery: Repeated approval, run, posting, or employee-payment outcomes create one applied decision, calculation, liability, and settlement effect.
- **FAC-14-13-11-05:** Failure and recovery: Failed employee payment leaves the obligation outstanding and does not reverse payroll expense; correction creates a linked run and preserves restricted history.

#### Source DDD §14.13.12 — Period-Control Recovery
- **FAC-14-13-12-01:** Normal: Soft close enters from Open under one policy owner; hard-close handoff transfers that owner to the close run as one exclusive domain consistency outcome. A policy-approved operational reopen records bounded posting classes, actor scope, authority epoch, and expiry; when postings occur, BeginRecloseGate transfers the reopen owner to the reclose run as one exclusive domain consistency outcome before finalization.
- **FAC-14-13-12-02:** Boundary: Conflicting soft-close policy versions are rejected; soft close can exit to Open before handoff; reopen expiry moves the request to ExpiredPendingClosure and rejects new postings in the GL journal-admission boundary; authoritative zero-admission evidence and positive-admission evidence select distinct no-change and reclose outcomes.
- **FAC-14-13-12-03:** Concurrency: Concurrent soft-close exit versus hard-close handoff, takeover, close, scoped-reopen, operational-reopen, or reclose attempts produce one authoritative gate owner at each version. Candidate successor process records have no admission authority before the exclusive handoff outcome.
- **FAC-14-13-12-04:** Duplicate delivery: Repeated soft-close entry or exit, ownership handoff, takeover, gate-open, gate-close, BeginRecloseGate, and extension commands return the prior result when identifiers and fingerprints match.
- **FAC-14-13-12-05:** Failure and recovery: Ambiguous soft-close or ownership-transfer responses and dependency outages keep or reveal exactly one authoritative owner and the authoritative admitted-process count and ledger positions through GetPostingGateStatus. A handoff, barrier release, soft-close resume, and second handoff preserve separate immutable SoftCloseControlEpoch summaries. Expired operational reopen retains ownership until direct no-change closure or reconciled reclose handoff, and a transferred mandatory reclose resumes rather than releasing its barrier.

#### Source DDD §14.13.13 — Cross-Context Event Handling
- **FAC-14-13-13-01:** Normal: A valid published event identity establishes at most one local domain effect and any resulting domain events in each receiving context.
- **FAC-14-13-13-02:** Boundary: Unknown contract version, invalid scope, sequence gap, or unsupported semantic transformation is rejected, deferred, or recorded as an exception under the receiving context's domain rule.
- **FAC-14-13-13-03:** Concurrency: Concurrent observations of the same event establish one local effect; replay preserves the original event identity.
- **FAC-14-13-13-04:** Duplicate delivery: Re-observation after an outcome exists returns the prior result and repeats no business effect.
- **FAC-14-13-13-05:** Failure and recovery: An event with no established outcome leaves no partial domain effect; unprocessable events retain evidence and an authorized resolution; reconstruction from a known domain position reproduces expected projections.

#### Source DDD §14.13.14 — Concurrency Rules
- **FAC-14-13-14-01:** Normal: A command with the current version and valid lifecycle state establishes one transition and its domain events.
- **FAC-14-13-14-02:** Boundary: A stale version, expired authorization epoch, invalid protected-operation state, or noncommutative overlap returns a typed conflict with no side effect.
- **FAC-14-13-14-03:** Concurrency: Approved multi-aggregate commands coordinate named participants in the deterministic order defined by Section 9.1 and establish no partial result when an invariant fails.
- **FAC-14-13-14-04:** Duplicate delivery: A command with an ambiguous outcome or repeated identity is resolved by its idempotency identity and returns the existing in-progress or terminal result.
- **FAC-14-13-14-05:** Failure and recovery: Concurrency-conflict retry preserves the command fingerprint, and an ownership or process-epoch change prevents a superseded actor from establishing a transition.

#### Source DDD §14.13.15 — Audit Integrity
- **FAC-14-13-15-01:** Normal: Contiguous events append to the scoped chain, seal creation covers the declared range, and proof verification returns Valid.
- **FAC-14-13-15-02:** Boundary: Missing sequence, unsupported integrity format, verification-credential interval edge, and seal-range boundary produce explicit deterministic outcomes.
- **FAC-14-13-15-03:** Concurrency: Expected audit-sequence validation allows one next chain position and prevents competing seals from publishing conflicting evidence.
- **FAC-14-13-15-04:** Duplicate delivery: Duplicate event or seal identifiers return existing positions when fingerprints match and create integrity conflicts when they differ.
- **FAC-14-13-15-05:** Failure and recovery: Missing event, proof mismatch, invalid proof, or verification-credential compromise suspends the affected proof status, opens an incident, preserves evidence, and never edits source events.

## 5. Functional Review Checklist
- [x] Scope and non-goals agree with the DDD baseline.
- [x] PRD functional actions are not mislabeled as DDD commands or events.
- [x] Every capability has named users, functional actions, controls, results, and DDD references.
- [x] Every authoritative record and explicit DDD scenario behavior is covered by a direct requirement or identified as a derived outcome of a named operation.
- [x] Every core/additional workflow has direct requirement IDs where named operations exist and supporting families for broader behavior.
- [x] Approval and segregation rules are represented in workflow behavior.
- [x] Correction, reversal, amendment, return, unapplication, replacement, and compensation preserve immutable lineage.
- [x] User-visible states distinguish pending, approved, posted, failed, exception, reconciled, reversed, and cancelled outcomes as applicable.
- [x] Acceptance includes normal, boundary, authorization, concurrency, duplicate, failure, recovery, and audit cases.
- [x] No requirement prescribes implementation architecture.

## 6. Version 1.5 Review Notes
- Preserved all 22 workflow IDs and all 199 functional acceptance IDs and meanings.
- Preserved explicit coverage for GL reporting, intercompany source data, payment bank accounts, AR adjustments, bank-feed connections, fixed-asset lifecycle, tax configuration, and payroll profile/filing records; configuration and source-record maintenance remain supporting dependencies unless executed by the workflow itself.
- Added a capability-coverage completion table showing the DDD basis for every new requirement family.
- Clarified that supporting requirement families are cross-cutting dependencies and do not replace direct functional coverage.
- Reconfirmed every acceptance scenario against the unchanged DDD v3.1 §14 source.
- Reclassified configuration and source-record maintenance as supporting dependencies rather than direct workflow actions.
- Added direct GFR mappings for the event-handling and concurrency workflows and completed the functional review checklist.

## 7. Verification Checkpoint
| Checkpoint field | Value |
|---|---|
| Verified body SHA-256 | `6489ac5a5d00562b15e600294d72175addde25d55aaaee3568d02537a8354f4a` |
| Hash boundary | UTF-8 bytes from title through the blank line immediately preceding ## 7. Verification Checkpoint; checkpoint section excluded |
| Checkpoint ID | FTRACE-1.5-2026-07-24 |
| Source DDD checkpoint | DDD-3.1-2026-07-24 |
| Workflows | 22 |
| Functional acceptance scenarios | 199 |
| Requirement references valid | Yes |
| Acceptance IDs unique | Yes |
| Acceptance semantics equal to DDD §14 | Yes |
| Review result | Passed: direct-versus-supporting workflow mappings, cross-cutting GFR traceability, IDs, completed checklist, and DDD acceptance equivalence reconciled |
| Review rule | When this hash, the catalog hash, and DDD source hash remain unchanged, repeat only structural validation. |
