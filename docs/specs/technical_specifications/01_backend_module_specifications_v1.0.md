# Finance Platform Backend Module Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready technical baseline |
| Architecture source | Solution/System Design v1.0 |
| Runtime | Go 1.26.x modular monolith |

## 1. Purpose and authority

This specification defines the exact Go repository boundaries, application-layer conventions, aggregate implementation rules, error model, configuration pattern and module ownership. Domain semantics remain authoritative in DDD v3.1; this document selects implementation mechanisms without changing ownership or invariants.

## 2. Repository layout

```text
cmd/api/main.go
cmd/worker/main.go
internal/<module>/domain/
internal/<module>/application/
internal/<module>/ports/
internal/<module>/adapters/http/
internal/<module>/adapters/postgres/
internal/<module>/adapters/events/
internal/platform/authn/
internal/platform/authz/
internal/platform/config/
internal/platform/database/
internal/platform/httpx/
internal/platform/idempotency/
internal/platform/observability/
internal/platform/outbox/
contracts/openapi/
db/migrations/
web/
infra/terraform/
```

## 3. Dependency rules

1. `domain` imports only the Go standard library and approved exact-decimal/identifier packages.
2. `application` imports its own `domain` and declared port interfaces.
3. Adapters implement ports and may import platform packages; domain and application packages never import adapters.
4. One module may invoke another only through an exported application port or durable integration contract.
5. No module imports another module's PostgreSQL adapter or accesses another schema directly.
6. Cross-module dependency cycles fail the architecture test.
7. Shared code is limited to technical primitives; finance rules remain in their owning module.

## 4. Module ownership catalog

| Prefix | Bounded context | Go package | Schema | Aggregate roots | Functional scope |
|---|---|---|---|---|---|
| OMD | Organization & Master Data | `internal/organization` | `organization` | LegalEntity, Party, CustomerProfile, VendorProfile, FiscalCalendar | FR-OMD-001, FR-OMD-002, FR-OMD-003, FR-OMD-004, FR-OMD-005, FR-OMD-006 |
| GL | General Ledger | `internal/gl` | `gl` | Ledger, AccountingBook, ChartOfAccounts, Account, JournalEntry, PeriodPostingGate | FR-GL-001, FR-GL-002, FR-GL-003, FR-GL-004, FR-GL-005, FR-GL-006, FR-GL-007, FR-GL-008, FR-GL-009, FR-GL-010, FR-GL-011, FR-GL-012, FR-GL-013, FR-GL-014, FR-GL-015, FR-GL-016, FR-GL-017, FR-GL-018 |
| AP | Accounts Payable | `internal/ap` | `ap` | VendorInvoice, PaymentRequest | FR-AP-001, FR-AP-002, FR-AP-003, FR-AP-004, FR-AP-005, FR-AP-006, FR-AP-007, FR-AP-008, FR-AP-009, FR-AP-010 |
| AR | Accounts Receivable | `internal/ar` | `ar` | CustomerInvoice, ReceivableOpenItem, CustomerReceipt, CreditNote, CustomerRefundRequest | FR-AR-001, FR-AR-002, FR-AR-003, FR-AR-004, FR-AR-005, FR-AR-006, FR-AR-007, FR-AR-008, FR-AR-009, FR-AR-010, FR-AR-011, FR-AR-012, FR-AR-013, FR-AR-014, FR-AR-015, FR-AR-016 |
| PAYR | Payroll | `internal/payroll` | `payroll` | PayrollRun, EmployeePayrollProfile, PayrollTaxFiling | FR-PAYR-001, FR-PAYR-002, FR-PAYR-003, FR-PAYR-004, FR-PAYR-005, FR-PAYR-006, FR-PAYR-007 |
| INV | Invoicing | `internal/invoicing` | `invoicing` | InvoiceTemplate, BillingSchedule, GeneratedInvoice | FR-INV-001, FR-INV-002, FR-INV-003, FR-INV-004, FR-INV-005, FR-INV-006 |
| PCM | Payments & Cash Management | `internal/payments` | `payments` | BankAccount, PaymentBatch, PaymentInstruction, PaymentReturn, ExpectedIncomingSettlement, UnallocatedIncomingSettlement, SettlementReceipt | FR-PCM-001, FR-PCM-002, FR-PCM-003, FR-PCM-004, FR-PCM-005, FR-PCM-006, FR-PCM-007, FR-PCM-008, FR-PCM-009, FR-PCM-010, FR-PCM-011, FR-PCM-012, FR-PCM-013, FR-PCM-014, FR-PCM-015, FR-PCM-016, FR-PCM-017, FR-PCM-018, FR-PCM-019, FR-PCM-020, FR-PCM-021, FR-PCM-022, FR-PCM-023, FR-PCM-024, FR-PCM-025 |
| RPT | Financial Reporting | `internal/reporting` | `reporting` | ReportDefinition, ConsolidationRun, FinancialStatement | FR-RPT-001, FR-RPT-002, FR-RPT-003, FR-RPT-004, FR-RPT-005, FR-RPT-006 |
| IC | Multi-Entity / Intercompany | `internal/intercompany` | `intercompany` | IntercompanyAgreement, IntercompanyTransaction, SettlementRun, EliminationRun | FR-IC-001, FR-IC-002, FR-IC-003, FR-IC-004, FR-IC-005, FR-IC-006, FR-IC-007, FR-IC-008, FR-IC-009, FR-IC-010, FR-IC-011 |
| REV | Revenue Recognition | `internal/revenue` | `revenue` | RevenueContract, RevenueSchedule, ContractModification | FR-REV-001, FR-REV-002, FR-REV-003, FR-REV-004, FR-REV-005, FR-REV-006 |
| FA | Fixed Assets | `internal/fixedassets` | `fixed_assets` | FixedAsset, DepreciationRun, ImpairmentAssessment, AssetDisposal | FR-FA-001, FR-FA-002, FR-FA-003, FR-FA-004, FR-FA-005, FR-FA-006, FR-FA-007, FR-FA-008, FR-FA-009, FR-FA-010, FR-FA-011, FR-FA-012, FR-FA-013, FR-FA-014, FR-FA-015, FR-FA-016, FR-FA-017, FR-FA-018, FR-FA-019, FR-FA-020, FR-FA-021 |
| FX | Multi-Currency | `internal/multicurrency` | `multi_currency` | CurrencyRateSet, RevaluationRun, TranslationRun | FR-FX-001, FR-FX-002, FR-FX-003, FR-FX-004, FR-FX-005 |
| FPM | Fiscal Period Management | `internal/fiscalperiod` | `fiscal_period` | FiscalPeriod, SoftCloseRun, CloseRun, ReopenRequest | FR-FPM-001, FR-FPM-002, FR-FPM-003, FR-FPM-004, FR-FPM-005, FR-FPM-006, FR-FPM-007, FR-FPM-008, FR-FPM-009, FR-FPM-010, FR-FPM-011, FR-FPM-012, FR-FPM-013 |
| COA | COA Segment Accounting | `internal/coa` | `coa` | SegmentDefinition, SegmentCombination, SegmentChangeRequest | FR-COA-001, FR-COA-002, FR-COA-003, FR-COA-004, FR-COA-005 |
| BFR | Bank Feeds & Reconciliation | `internal/bankfeeds` | `bank_reconciliation` | BankFeedConnection, BankStatement, ReconciliationSession | FR-BFR-001, FR-BFR-002, FR-BFR-003, FR-BFR-004, FR-BFR-005, FR-BFR-006 |
| TAX | Tax Filing | `internal/tax` | `tax` | TaxConfiguration, TaxReturn, FilingSubmission, TaxAmendment, ReturnLevelTaxAdjustment, TaxPaymentObligation | FR-TAX-001, FR-TAX-002, FR-TAX-003, FR-TAX-004, FR-TAX-005, FR-TAX-006, FR-TAX-007, FR-TAX-008, FR-TAX-009, FR-TAX-010, FR-TAX-011, FR-TAX-012, FR-TAX-013, FR-TAX-014, FR-TAX-015, FR-TAX-016 |
| WFA | Workflow & Approvals | `internal/workflow` | `workflow` | ApprovalPolicy, ApprovalRequest, Delegation | FR-WFA-001, FR-WFA-002, FR-WFA-003, FR-WFA-004, FR-WFA-005 |
| IAM | Identity & Access | `internal/identity` | `identity` | User, Role, AccessPolicy, SegregationRule | FR-IAM-001, FR-IAM-002, FR-IAM-003, FR-IAM-004, FR-IAM-005, FR-IAM-006 |
| AUD | Audit Integrity | `internal/audit` | `audit` | AuditChain | FR-AUD-001, FR-AUD-002, FR-AUD-003, FR-AUD-004, FR-AUD-005 |

## 5. Aggregate persistence contract

| Module | Aggregate root | Primary table | Concurrency field | Access rule |
|---|---|---|---|---|
| OMD | LegalEntity | `organization.legal_entity` | `aggregate_version bigint` | Repository interface in owning module only |
| OMD | Party | `organization.party` | `aggregate_version bigint` | Repository interface in owning module only |
| OMD | CustomerProfile | `organization.customer_profile` | `aggregate_version bigint` | Repository interface in owning module only |
| OMD | VendorProfile | `organization.vendor_profile` | `aggregate_version bigint` | Repository interface in owning module only |
| OMD | FiscalCalendar | `organization.fiscal_calendar` | `aggregate_version bigint` | Repository interface in owning module only |
| GL | Ledger | `gl.ledger` | `aggregate_version bigint` | Repository interface in owning module only |
| GL | AccountingBook | `gl.accounting_book` | `aggregate_version bigint` | Repository interface in owning module only |
| GL | ChartOfAccounts | `gl.chart_of_accounts` | `aggregate_version bigint` | Repository interface in owning module only |
| GL | Account | `gl.account` | `aggregate_version bigint` | Repository interface in owning module only |
| GL | JournalEntry | `gl.journal_entry` | `aggregate_version bigint` | Repository interface in owning module only |
| GL | PeriodPostingGate | `gl.period_posting_gate` | `aggregate_version bigint` | Repository interface in owning module only |
| AP | VendorInvoice | `ap.vendor_invoice` | `aggregate_version bigint` | Repository interface in owning module only |
| AP | PaymentRequest | `ap.payment_request` | `aggregate_version bigint` | Repository interface in owning module only |
| AR | CustomerInvoice | `ar.customer_invoice` | `aggregate_version bigint` | Repository interface in owning module only |
| AR | ReceivableOpenItem | `ar.receivable_open_item` | `aggregate_version bigint` | Repository interface in owning module only |
| AR | CustomerReceipt | `ar.customer_receipt` | `aggregate_version bigint` | Repository interface in owning module only |
| AR | CreditNote | `ar.credit_note` | `aggregate_version bigint` | Repository interface in owning module only |
| AR | CustomerRefundRequest | `ar.customer_refund_request` | `aggregate_version bigint` | Repository interface in owning module only |
| PAYR | PayrollRun | `payroll.payroll_run` | `aggregate_version bigint` | Repository interface in owning module only |
| PAYR | EmployeePayrollProfile | `payroll.employee_payroll_profile` | `aggregate_version bigint` | Repository interface in owning module only |
| PAYR | PayrollTaxFiling | `payroll.payroll_tax_filing` | `aggregate_version bigint` | Repository interface in owning module only |
| INV | InvoiceTemplate | `invoicing.invoice_template` | `aggregate_version bigint` | Repository interface in owning module only |
| INV | BillingSchedule | `invoicing.billing_schedule` | `aggregate_version bigint` | Repository interface in owning module only |
| INV | GeneratedInvoice | `invoicing.generated_invoice` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | BankAccount | `payments.bank_account` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | PaymentBatch | `payments.payment_batch` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | PaymentInstruction | `payments.payment_instruction` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | PaymentReturn | `payments.payment_return` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | ExpectedIncomingSettlement | `payments.expected_incoming_settlement` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | UnallocatedIncomingSettlement | `payments.unallocated_incoming_settlement` | `aggregate_version bigint` | Repository interface in owning module only |
| PCM | SettlementReceipt | `payments.settlement_receipt` | `aggregate_version bigint` | Repository interface in owning module only |
| RPT | ReportDefinition | `reporting.report_definition` | `aggregate_version bigint` | Repository interface in owning module only |
| RPT | ConsolidationRun | `reporting.consolidation_run` | `aggregate_version bigint` | Repository interface in owning module only |
| RPT | FinancialStatement | `reporting.financial_statement` | `aggregate_version bigint` | Repository interface in owning module only |
| IC | IntercompanyAgreement | `intercompany.intercompany_agreement` | `aggregate_version bigint` | Repository interface in owning module only |
| IC | IntercompanyTransaction | `intercompany.intercompany_transaction` | `aggregate_version bigint` | Repository interface in owning module only |
| IC | SettlementRun | `intercompany.settlement_run` | `aggregate_version bigint` | Repository interface in owning module only |
| IC | EliminationRun | `intercompany.elimination_run` | `aggregate_version bigint` | Repository interface in owning module only |
| REV | RevenueContract | `revenue.revenue_contract` | `aggregate_version bigint` | Repository interface in owning module only |
| REV | RevenueSchedule | `revenue.revenue_schedule` | `aggregate_version bigint` | Repository interface in owning module only |
| REV | ContractModification | `revenue.contract_modification` | `aggregate_version bigint` | Repository interface in owning module only |
| FA | FixedAsset | `fixed_assets.fixed_asset` | `aggregate_version bigint` | Repository interface in owning module only |
| FA | DepreciationRun | `fixed_assets.depreciation_run` | `aggregate_version bigint` | Repository interface in owning module only |
| FA | ImpairmentAssessment | `fixed_assets.impairment_assessment` | `aggregate_version bigint` | Repository interface in owning module only |
| FA | AssetDisposal | `fixed_assets.asset_disposal` | `aggregate_version bigint` | Repository interface in owning module only |
| FX | CurrencyRateSet | `multi_currency.currency_rate_set` | `aggregate_version bigint` | Repository interface in owning module only |
| FX | RevaluationRun | `multi_currency.revaluation_run` | `aggregate_version bigint` | Repository interface in owning module only |
| FX | TranslationRun | `multi_currency.translation_run` | `aggregate_version bigint` | Repository interface in owning module only |
| FPM | FiscalPeriod | `fiscal_period.fiscal_period` | `aggregate_version bigint` | Repository interface in owning module only |
| FPM | SoftCloseRun | `fiscal_period.soft_close_run` | `aggregate_version bigint` | Repository interface in owning module only |
| FPM | CloseRun | `fiscal_period.close_run` | `aggregate_version bigint` | Repository interface in owning module only |
| FPM | ReopenRequest | `fiscal_period.reopen_request` | `aggregate_version bigint` | Repository interface in owning module only |
| COA | SegmentDefinition | `coa.segment_definition` | `aggregate_version bigint` | Repository interface in owning module only |
| COA | SegmentCombination | `coa.segment_combination` | `aggregate_version bigint` | Repository interface in owning module only |
| COA | SegmentChangeRequest | `coa.segment_change_request` | `aggregate_version bigint` | Repository interface in owning module only |
| BFR | BankFeedConnection | `bank_reconciliation.bank_feed_connection` | `aggregate_version bigint` | Repository interface in owning module only |
| BFR | BankStatement | `bank_reconciliation.bank_statement` | `aggregate_version bigint` | Repository interface in owning module only |
| BFR | ReconciliationSession | `bank_reconciliation.reconciliation_session` | `aggregate_version bigint` | Repository interface in owning module only |
| TAX | TaxConfiguration | `tax.tax_configuration` | `aggregate_version bigint` | Repository interface in owning module only |
| TAX | TaxReturn | `tax.tax_return` | `aggregate_version bigint` | Repository interface in owning module only |
| TAX | FilingSubmission | `tax.filing_submission` | `aggregate_version bigint` | Repository interface in owning module only |
| TAX | TaxAmendment | `tax.tax_amendment` | `aggregate_version bigint` | Repository interface in owning module only |
| TAX | ReturnLevelTaxAdjustment | `tax.return_level_tax_adjustment` | `aggregate_version bigint` | Repository interface in owning module only |
| TAX | TaxPaymentObligation | `tax.tax_payment_obligation` | `aggregate_version bigint` | Repository interface in owning module only |
| WFA | ApprovalPolicy | `workflow.approval_policy` | `aggregate_version bigint` | Repository interface in owning module only |
| WFA | ApprovalRequest | `workflow.approval_request` | `aggregate_version bigint` | Repository interface in owning module only |
| WFA | Delegation | `workflow.delegation` | `aggregate_version bigint` | Repository interface in owning module only |
| IAM | User | `identity.user` | `aggregate_version bigint` | Repository interface in owning module only |
| IAM | Role | `identity.role` | `aggregate_version bigint` | Repository interface in owning module only |
| IAM | AccessPolicy | `identity.access_policy` | `aggregate_version bigint` | Repository interface in owning module only |
| IAM | SegregationRule | `identity.segregation_rule` | `aggregate_version bigint` | Repository interface in owning module only |
| AUD | AuditChain | `audit.audit_chain` | `aggregate_version bigint` | Repository interface in owning module only |

Every mutable aggregate shall expose:

```go
type AggregateMeta struct {
    ID        uuid.UUID
    Version   int64
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

- Commands carry `ExpectedVersion` when they modify an existing aggregate.
- The repository update uses `WHERE aggregate_id = $1 AND aggregate_version = $2` and increments exactly once.
- Zero affected rows become `VERSION_CONFLICT`; handlers do not silently reload and overwrite.
- Established financial facts use append-only child/fact tables and linked correction records.

## 6. Application command contract

```go
type CommandMeta struct {
    ActorID             uuid.UUID
    AuthenticationID    string
    AccountingScopeID   uuid.UUID
    IdempotencyKey      string
    CanonicalFingerprint string
    ExpectedVersion     *int64
    CorrelationID       uuid.UUID
    CausationID         uuid.UUID
}

type CommandResult[T any] struct {
    Status          string
    AggregateID     uuid.UUID
    AggregateVersion int64
    ProcessID       *uuid.UUID
    Data            T
}
```

Command-handler order is normative:

1. Validate transport schema and canonicalize business content.
2. Authenticate and resolve the actor.
3. Authorize the action and accounting scope; evaluate segregation rules.
4. Acquire or read the idempotency record.
5. Start the local PostgreSQL transaction.
6. Load and lock required aggregates in the documented order.
7. Revalidate versions, domain state, effective rules and approvals.
8. Apply the domain operation and collect domain outcomes.
9. Persist aggregate/fact changes, idempotency result, audit envelope and outbox records in the same transaction.
10. Commit; only then return the established result.

## 7. Domain error contract

| Code | HTTP | Meaning | Required behavior |
|---|---|---|---|
| VALIDATION_FAILED | 400 | Input shape or field validation failed | No state change |
| DOMAIN_RULE_REJECTED | 422 | DDD invariant or lifecycle rule rejected the action | No state change |
| AUTHENTICATION_REQUIRED | 401 | Token absent or invalid | No state change |
| AUTHORIZATION_DENIED | 403 | Scope, permission or segregation rule denied the action | No state change |
| NOT_FOUND | 404 | Authoritative record not found in authorized scope | No state change |
| VERSION_CONFLICT | 409 | Expected aggregate/process version differs | Return current version and safe-retry guidance |
| IDEMPOTENCY_CONFLICT | 409 | Identity reused with a different canonical fingerprint | No repeated effect |
| DEPENDENCY_UNAVAILABLE | 503 | Required external dependency unavailable | Persist no false success; expose retry/reconciliation |
| AMBIGUOUS_OUTCOME | 202 | External outcome not yet authoritative | Return process/result identity |
| RATE_LIMITED | 429 | Declared workload/admission limit reached | Retry only after supplied delay |

## 8. Money, dates and identifiers

- JSON money amounts are decimal strings; Go domain money uses an exact-decimal type and ISO 4217 currency code.
- PostgreSQL uses `numeric(38,12)` for generalized monetary/rate quantities; business-specific scales are enforced by domain policy.
- Binary floating point is prohibited for accounting quantities.
- Business dates use `date`; instants use UTC `timestamptz`; the applicable business time zone is explicit.
- Aggregate and command identities use UUID v7 when created by this application; externally supplied stable identities retain their source format in a typed value object.

## 9. Module command-handler inventory


### 9.1 Organization & Master Data

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-OMD-001 | Maintain legal entities | PRD functional action | `omdMaintainLegalEntitiesHandler` | finance.omd.maintain.legal.entities |
| FR-OMD-002 | Maintain parties | PRD functional action | `omdMaintainPartiesHandler` | finance.omd.maintain.parties |
| FR-OMD-003 | Maintain customer profiles | PRD functional action | `omdMaintainCustomerProfilesHandler` | finance.omd.maintain.customer.profiles |
| FR-OMD-004 | Maintain vendor profiles | PRD functional action | `omdMaintainVendorProfilesHandler` | finance.omd.maintain.vendor.profiles |
| FR-OMD-005 | Maintain fiscal calendars | PRD functional action | `omdMaintainFiscalCalendarsHandler` | finance.omd.maintain.fiscal.calendars |
| FR-OMD-006 | Publish approved master-data changes | PRD functional action | `omdPublishApprovedMasterDataChangesHandler` | finance.omd.publish.approved.master.data.changes |

### 9.2 General Ledger

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-GL-001 | SubmitPostingRequest | DDD representative command | `glSubmitPostingRequestHandler` | finance.gl.submit.posting.request |
| FR-GL-002 | ApplyJournalApprovalDecision | DDD representative command | `glApplyJournalApprovalDecisionHandler` | finance.gl.apply.journal.approval.decision |
| FR-GL-003 | ReverseJournalEntry | DDD representative command | `glReverseJournalEntryHandler` | finance.gl.reverse.journal.entry |
| FR-GL-004 | EnterSoftCloseGate | DDD representative command | `glEnterSoftCloseGateHandler` | finance.gl.enter.soft.close.gate |
| FR-GL-005 | ExitSoftCloseGate | DDD representative command | `glExitSoftCloseGateHandler` | finance.gl.exit.soft.close.gate |
| FR-GL-006 | AcquirePostingBarrier | DDD representative command | `glAcquirePostingBarrierHandler` | finance.gl.acquire.posting.barrier |
| FR-GL-007 | ReleasePostingBarrier | DDD representative command | `glReleasePostingBarrierHandler` | finance.gl.release.posting.barrier |
| FR-GL-008 | FinalizePostingGate | DDD representative command | `glFinalizePostingGateHandler` | finance.gl.finalize.posting.gate |
| FR-GL-009 | OpenScopedReopenGate | DDD representative command | `glOpenScopedReopenGateHandler` | finance.gl.open.scoped.reopen.gate |
| FR-GL-010 | CloseScopedReopenGate | DDD representative command | `glCloseScopedReopenGateHandler` | finance.gl.close.scoped.reopen.gate |
| FR-GL-011 | OpenOperationalReopenGate | DDD representative command | `glOpenOperationalReopenGateHandler` | finance.gl.open.operational.reopen.gate |
| FR-GL-012 | CloseOperationalReopenGate | DDD representative command | `glCloseOperationalReopenGateHandler` | finance.gl.close.operational.reopen.gate |
| FR-GL-013 | BeginRecloseGate | DDD representative command | `glBeginRecloseGateHandler` | finance.gl.begin.reclose.gate |
| FR-GL-014 | GetPostingGateStatus | DDD reference operation | `glGetPostingGateStatusHandler` | finance.gl.get.posting.gate.status |
| FR-GL-015 | Maintain ledgers | PRD functional action | `glMaintainLedgersHandler` | finance.gl.maintain.ledgers |
| FR-GL-016 | Maintain accounting books | PRD functional action | `glMaintainAccountingBooksHandler` | finance.gl.maintain.accounting.books |
| FR-GL-017 | Maintain charts of accounts | PRD functional action | `glMaintainChartsOfAccountsHandler` | finance.gl.maintain.charts.of.accounts |
| FR-GL-018 | Maintain accounts and reporting mappings | PRD functional action | `glMaintainAccountsAndReportingMappingsHandler` | finance.gl.maintain.accounts.and.reporting.mappings |

### 9.3 Accounts Payable

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-AP-001 | RegisterVendorInvoice | DDD representative command | `apRegisterVendorInvoiceHandler` | finance.ap.register.vendor.invoice |
| FR-AP-002 | ApplyAssetClearingClassification | DDD representative command | `apApplyAssetClearingClassificationHandler` | finance.ap.apply.asset.clearing.classification |
| FR-AP-003 | ApplyIncomingSettlement | DDD representative command | `apApplyIncomingSettlementHandler` | finance.ap.apply.incoming.settlement |
| FR-AP-004 | ReverseIncomingSettlementApplication | DDD representative command | `apReverseIncomingSettlementApplicationHandler` | finance.ap.reverse.incoming.settlement.application |
| FR-AP-005 | ApplyPaymentReturn | DDD representative command | `apApplyPaymentReturnHandler` | finance.ap.apply.payment.return |
| FR-AP-006 | ApplyVendorInvoiceApprovalDecision | DDD representative command | `apApplyVendorInvoiceApprovalDecisionHandler` | finance.ap.apply.vendor.invoice.approval.decision |
| FR-AP-007 | RequestPayment | DDD representative command | `apRequestPaymentHandler` | finance.ap.request.payment |
| FR-AP-008 | ValidateVendorInvoice | DDD detailed command | `apValidateVendorInvoiceHandler` | finance.ap.validate.vendor.invoice |
| FR-AP-009 | DisputeVendorInvoice | DDD detailed command | `apDisputeVendorInvoiceHandler` | finance.ap.dispute.vendor.invoice |
| FR-AP-010 | VoidVendorInvoice | DDD detailed command | `apVoidVendorInvoiceHandler` | finance.ap.void.vendor.invoice |

### 9.4 Accounts Receivable

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-AR-001 | IssueCustomerInvoice | DDD representative command | `arIssueCustomerInvoiceHandler` | finance.ar.issue.customer.invoice |
| FR-AR-002 | RecordReceipt | DDD representative command | `arRecordReceiptHandler` | finance.ar.record.receipt |
| FR-AR-003 | ApplyReceipt | DDD representative command | `arApplyReceiptHandler` | finance.ar.apply.receipt |
| FR-AR-004 | UnapplyReceipt | DDD representative command | `arUnapplyReceiptHandler` | finance.ar.unapply.receipt |
| FR-AR-005 | RollbackUnpostedApplicationBatch | DDD representative command | `arRollbackUnpostedApplicationBatchHandler` | finance.ar.rollback.unposted.application.batch |
| FR-AR-006 | IssueCreditNote | DDD representative command | `arIssueCreditNoteHandler` | finance.ar.issue.credit.note |
| FR-AR-007 | CreateCustomerRefundRequest | DDD representative command | `arCreateCustomerRefundRequestHandler` | finance.ar.create.customer.refund.request |
| FR-AR-008 | CancelCustomerRefundRequest | DDD representative command | `arCancelCustomerRefundRequestHandler` | finance.ar.cancel.customer.refund.request |
| FR-AR-009 | ApplyCustomerRefundApprovalDecision | DDD representative command | `arApplyCustomerRefundApprovalDecisionHandler` | finance.ar.apply.customer.refund.approval.decision |
| FR-AR-010 | RequestCustomerRefundPayment | DDD representative command | `arRequestCustomerRefundPaymentHandler` | finance.ar.request.customer.refund.payment |
| FR-AR-011 | CancelCustomerRefundPayment | DDD representative command | `arCancelCustomerRefundPaymentHandler` | finance.ar.cancel.customer.refund.payment |
| FR-AR-012 | ApplyCustomerRefundPaymentResult | DDD representative command | `arApplyCustomerRefundPaymentResultHandler` | finance.ar.apply.customer.refund.payment.result |
| FR-AR-013 | ApplyPaymentReturn | DDD representative command | `arApplyPaymentReturnHandler` | finance.ar.apply.payment.return |
| FR-AR-014 | Resolve customer overpayments | PRD functional action | `arResolveCustomerOverpaymentsHandler` | finance.ar.resolve.customer.overpayments |
| FR-AR-015 | Record customer chargebacks | PRD functional action | `arRecordCustomerChargebacksHandler` | finance.ar.record.customer.chargebacks |
| FR-AR-016 | Record receivable write-offs | PRD functional action | `arRecordReceivableWriteOffsHandler` | finance.ar.record.receivable.write.offs |

### 9.5 Payroll

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-PAYR-001 | CalculatePayrollRun | DDD representative command | `payrCalculatePayrollRunHandler` | finance.payr.calculate.payroll.run |
| FR-PAYR-002 | ApplyPayrollRunApprovalDecision | DDD representative command | `payrApplyPayrollRunApprovalDecisionHandler` | finance.payr.apply.payroll.run.approval.decision |
| FR-PAYR-003 | PostPayrollRun | DDD representative command | `payrPostPayrollRunHandler` | finance.payr.post.payroll.run |
| FR-PAYR-004 | CreatePayrollCorrection | DDD representative command | `payrCreatePayrollCorrectionHandler` | finance.payr.create.payroll.correction |
| FR-PAYR-005 | ApplyPaymentReturn | DDD representative command | `payrApplyPaymentReturnHandler` | finance.payr.apply.payment.return |
| FR-PAYR-006 | Maintain employee payroll profiles | PRD functional action | `payrMaintainEmployeePayrollProfilesHandler` | finance.payr.maintain.employee.payroll.profiles |
| FR-PAYR-007 | Maintain payroll tax-filing records | PRD functional action | `payrMaintainPayrollTaxFilingRecordsHandler` | finance.payr.maintain.payroll.tax.filing.records |

### 9.6 Invoicing

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-INV-001 | Configure invoice templates | PRD functional action | `invConfigureInvoiceTemplatesHandler` | finance.inv.configure.invoice.templates |
| FR-INV-002 | Configure billing schedules | PRD functional action | `invConfigureBillingSchedulesHandler` | finance.inv.configure.billing.schedules |
| FR-INV-003 | Generate invoices | PRD functional action | `invGenerateInvoicesHandler` | finance.inv.generate.invoices |
| FR-INV-004 | Finalize generated invoices | PRD functional action | `invFinalizeGeneratedInvoicesHandler` | finance.inv.finalize.generated.invoices |
| FR-INV-005 | Recalculate unfinalized invoices | PRD functional action | `invRecalculateUnfinalizedInvoicesHandler` | finance.inv.recalculate.unfinalized.invoices |
| FR-INV-006 | Cancel unfinalized invoices | PRD functional action | `invCancelUnfinalizedInvoicesHandler` | finance.inv.cancel.unfinalized.invoices |

### 9.7 Payments & Cash Management

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-PCM-001 | PreparePaymentBatch | DDD representative command | `pcmPreparePaymentBatchHandler` | finance.pcm.prepare.payment.batch |
| FR-PCM-002 | ApplyPaymentBatchApprovalDecision | DDD representative command | `pcmApplyPaymentBatchApprovalDecisionHandler` | finance.pcm.apply.payment.batch.approval.decision |
| FR-PCM-003 | CancelPaymentBatch | DDD representative command | `pcmCancelPaymentBatchHandler` | finance.pcm.cancel.payment.batch |
| FR-PCM-004 | RegisterExpectedIncomingSettlement | DDD representative command | `pcmRegisterExpectedIncomingSettlementHandler` | finance.pcm.register.expected.incoming.settlement |
| FR-PCM-005 | ResolveExpectedIncomingSettlementException | DDD representative command | `pcmResolveExpectedIncomingSettlementExceptionHandler` | finance.pcm.resolve.expected.incoming.settlement.exception |
| FR-PCM-006 | CancelExpectedIncomingSettlement | DDD representative command | `pcmCancelExpectedIncomingSettlementHandler` | finance.pcm.cancel.expected.incoming.settlement |
| FR-PCM-007 | CloseExpectedIncomingSettlement | DDD representative command | `pcmCloseExpectedIncomingSettlementHandler` | finance.pcm.close.expected.incoming.settlement |
| FR-PCM-008 | CreatePaymentInstructionFromObligation | DDD representative command | `pcmCreatePaymentInstructionFromObligationHandler` | finance.pcm.create.payment.instruction.from.obligation |
| FR-PCM-009 | SubmitPaymentInstruction | DDD representative command | `pcmSubmitPaymentInstructionHandler` | finance.pcm.submit.payment.instruction |
| FR-PCM-010 | RetryPaymentInstruction | DDD representative command | `pcmRetryPaymentInstructionHandler` | finance.pcm.retry.payment.instruction |
| FR-PCM-011 | CancelPaymentInstruction | DDD representative command | `pcmCancelPaymentInstructionHandler` | finance.pcm.cancel.payment.instruction |
| FR-PCM-012 | ApplyPaymentInstructionExceptionDecision | DDD representative command | `pcmApplyPaymentInstructionExceptionDecisionHandler` | finance.pcm.apply.payment.instruction.exception.decision |
| FR-PCM-013 | RecordPaymentReturn | DDD representative command | `pcmRecordPaymentReturnHandler` | finance.pcm.record.payment.return |
| FR-PCM-014 | CancelUnpostedPaymentReturn | DDD representative command | `pcmCancelUnpostedPaymentReturnHandler` | finance.pcm.cancel.unposted.payment.return |
| FR-PCM-015 | AcknowledgePaymentReturn | DDD representative command | `pcmAcknowledgePaymentReturnHandler` | finance.pcm.acknowledge.payment.return |
| FR-PCM-016 | ResolvePaymentReturnException | DDD representative command | `pcmResolvePaymentReturnExceptionHandler` | finance.pcm.resolve.payment.return.exception |
| FR-PCM-017 | RecordUnallocatedIncomingSettlement | DDD representative command | `pcmRecordUnallocatedIncomingSettlementHandler` | finance.pcm.record.unallocated.incoming.settlement |
| FR-PCM-018 | ResolveUnallocatedIncomingSettlement | DDD representative command | `pcmResolveUnallocatedIncomingSettlementHandler` | finance.pcm.resolve.unallocated.incoming.settlement |
| FR-PCM-019 | RecordIncomingSettlement | DDD representative command | `pcmRecordIncomingSettlementHandler` | finance.pcm.record.incoming.settlement |
| FR-PCM-020 | ResolveSettlementReceiptValidationException | DDD representative command | `pcmResolveSettlementReceiptValidationExceptionHandler` | finance.pcm.resolve.settlement.receipt.validation.exception |
| FR-PCM-021 | ResolveIncomingSettlementOwnerException | DDD representative command | `pcmResolveIncomingSettlementOwnerExceptionHandler` | finance.pcm.resolve.incoming.settlement.owner.exception |
| FR-PCM-022 | CancelUnpostedSettlementReceipt | DDD representative command | `pcmCancelUnpostedSettlementReceiptHandler` | finance.pcm.cancel.unposted.settlement.receipt |
| FR-PCM-023 | AcknowledgeIncomingSettlement | DDD representative command | `pcmAcknowledgeIncomingSettlementHandler` | finance.pcm.acknowledge.incoming.settlement |
| FR-PCM-024 | ReverseIncomingSettlement | DDD representative command | `pcmReverseIncomingSettlementHandler` | finance.pcm.reverse.incoming.settlement |
| FR-PCM-025 | Maintain bank accounts | PRD functional action | `pcmMaintainBankAccountsHandler` | finance.pcm.maintain.bank.accounts |

### 9.8 Financial Reporting

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-RPT-001 | RunConsolidation | DDD representative command | `rptRunConsolidationHandler` | finance.rpt.run.consolidation |
| FR-RPT-002 | ApplyTranslationResult | DDD representative command | `rptApplyTranslationResultHandler` | finance.rpt.apply.translation.result |
| FR-RPT-003 | ApplyConsolidationApprovalDecision | DDD representative command | `rptApplyConsolidationApprovalDecisionHandler` | finance.rpt.apply.consolidation.approval.decision |
| FR-RPT-004 | PublishConsolidatedStatement | DDD representative command | `rptPublishConsolidatedStatementHandler` | finance.rpt.publish.consolidated.statement |
| FR-RPT-005 | Maintain report definitions | PRD functional action | `rptMaintainReportDefinitionsHandler` | finance.rpt.maintain.report.definitions |
| FR-RPT-006 | Generate and publish ledger financial statements | PRD functional action | `rptGenerateAndPublishLedgerFinancialStatementsHandler` | finance.rpt.generate.and.publish.ledger.financial.statements |

### 9.9 Multi-Entity / Intercompany

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-IC-001 | StartSettlement | DDD representative command | `icStartSettlementHandler` | finance.ic.start.settlement |
| FR-IC-002 | MatchIntercompanyItems | DDD representative command | `icMatchIntercompanyItemsHandler` | finance.ic.match.intercompany.items |
| FR-IC-003 | ApplyResidualApprovalDecision | DDD representative command | `icApplyResidualApprovalDecisionHandler` | finance.ic.apply.residual.approval.decision |
| FR-IC-004 | CreateSettlementInstructions | DDD representative command | `icCreateSettlementInstructionsHandler` | finance.ic.create.settlement.instructions |
| FR-IC-005 | CompleteSettlementRun | DDD representative command | `icCompleteSettlementRunHandler` | finance.ic.complete.settlement.run |
| FR-IC-006 | ApplyIncomingSettlement | DDD representative command | `icApplyIncomingSettlementHandler` | finance.ic.apply.incoming.settlement |
| FR-IC-007 | ReverseIncomingSettlementApplication | DDD representative command | `icReverseIncomingSettlementApplicationHandler` | finance.ic.reverse.incoming.settlement.application |
| FR-IC-008 | ApplyPaymentReturn | DDD representative command | `icApplyPaymentReturnHandler` | finance.ic.apply.payment.return |
| FR-IC-009 | RunElimination | DDD representative command | `icRunEliminationHandler` | finance.ic.run.elimination |
| FR-IC-010 | Maintain intercompany agreements | PRD functional action | `icMaintainIntercompanyAgreementsHandler` | finance.ic.maintain.intercompany.agreements |
| FR-IC-011 | Record intercompany transactions | PRD functional action | `icRecordIntercompanyTransactionsHandler` | finance.ic.record.intercompany.transactions |

### 9.10 Revenue Recognition

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-REV-001 | AssessContract | DDD representative command | `revAssessContractHandler` | finance.rev.assess.contract |
| FR-REV-002 | ApplyRevenueScheduleApprovalDecision | DDD representative command | `revApplyRevenueScheduleApprovalDecisionHandler` | finance.rev.apply.revenue.schedule.approval.decision |
| FR-REV-003 | PublishRevenueAccountingProfile | DDD representative command | `revPublishRevenueAccountingProfileHandler` | finance.rev.publish.revenue.accounting.profile |
| FR-REV-004 | ModifyContract | DDD representative command | `revModifyContractHandler` | finance.rev.modify.contract |
| FR-REV-005 | ApplyContractModificationApprovalDecision | DDD representative command | `revApplyContractModificationApprovalDecisionHandler` | finance.rev.apply.contract.modification.approval.decision |
| FR-REV-006 | RunRecognition | DDD representative command | `revRunRecognitionHandler` | finance.rev.run.recognition |

### 9.11 Fixed Assets

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-FA-001 | CapitalizeAsset | DDD representative command | `faCapitalizeAssetHandler` | finance.fa.capitalize.asset |
| FR-FA-002 | CreateAssetAcquisitionClearing | DDD representative command | `faCreateAssetAcquisitionClearingHandler` | finance.fa.create.asset.acquisition.clearing |
| FR-FA-003 | RunDepreciation | DDD representative command | `faRunDepreciationHandler` | finance.fa.run.depreciation |
| FR-FA-004 | ApplyImpairmentApprovalDecision | DDD representative command | `faApplyImpairmentApprovalDecisionHandler` | finance.fa.apply.impairment.approval.decision |
| FR-FA-005 | DisposeAsset | DDD representative command | `faDisposeAssetHandler` | finance.fa.dispose.asset |
| FR-FA-006 | ApplyAssetDisposalApprovalDecision | DDD representative command | `faApplyAssetDisposalApprovalDecisionHandler` | finance.fa.apply.asset.disposal.approval.decision |
| FR-FA-007 | CancelUnpostedAssetDisposal | DDD representative command | `faCancelUnpostedAssetDisposalHandler` | finance.fa.cancel.unposted.asset.disposal |
| FR-FA-008 | CompensateFailedDisposalPosting | DDD representative command | `faCompensateFailedDisposalPostingHandler` | finance.fa.compensate.failed.disposal.posting |
| FR-FA-009 | CreateDisposalSettlementClearing | DDD representative command | `faCreateDisposalSettlementClearingHandler` | finance.fa.create.disposal.settlement.clearing |
| FR-FA-010 | ApplyAssetSupplierLiabilityResult | DDD representative command | `faApplyAssetSupplierLiabilityResultHandler` | finance.fa.apply.asset.supplier.liability.result |
| FR-FA-011 | ApplyIncomingSettlement | DDD representative command | `faApplyIncomingSettlementHandler` | finance.fa.apply.incoming.settlement |
| FR-FA-012 | ReverseIncomingSettlementApplication | DDD representative command | `faReverseIncomingSettlementApplicationHandler` | finance.fa.reverse.incoming.settlement.application |
| FR-FA-013 | ApplyPaymentReturn | DDD representative command | `faApplyPaymentReturnHandler` | finance.fa.apply.payment.return |
| FR-FA-014 | ApplyAssetSettlementResult | DDD representative command | `faApplyAssetSettlementResultHandler` | finance.fa.apply.asset.settlement.result |
| FR-FA-015 | ReclassifyDisposalCostForPayment | DDD representative command | `faReclassifyDisposalCostForPaymentHandler` | finance.fa.reclassify.disposal.cost.for.payment |
| FR-FA-016 | RequestDisposalCostPayment | DDD representative command | `faRequestDisposalCostPaymentHandler` | finance.fa.request.disposal.cost.payment |
| FR-FA-017 | RequestDisposalCostPaymentReplacement | DDD representative command | `faRequestDisposalCostPaymentReplacementHandler` | finance.fa.request.disposal.cost.payment.replacement |
| FR-FA-018 | Record impairment assessments | PRD functional action | `faRecordImpairmentAssessmentsHandler` | finance.fa.record.impairment.assessments |
| FR-FA-019 | Transfer assets or components | PRD functional action | `faTransferAssetsOrComponentsHandler` | finance.fa.transfer.assets.or.components |
| FR-FA-020 | Split assets or components | PRD functional action | `faSplitAssetsOrComponentsHandler` | finance.fa.split.assets.or.components |
| FR-FA-021 | Correct posted asset disposals | PRD functional action | `faCorrectPostedAssetDisposalsHandler` | finance.fa.correct.posted.asset.disposals |

### 9.12 Multi-Currency

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-FX-001 | PublishRateSet | DDD representative command | `fxPublishRateSetHandler` | finance.fx.publish.rate.set |
| FR-FX-002 | RunRevaluation | DDD representative command | `fxRunRevaluationHandler` | finance.fx.run.revaluation |
| FR-FX-003 | ApplyRevaluationApprovalDecision | DDD representative command | `fxApplyRevaluationApprovalDecisionHandler` | finance.fx.apply.revaluation.approval.decision |
| FR-FX-004 | PostRevaluationRun | DDD representative command | `fxPostRevaluationRunHandler` | finance.fx.post.revaluation.run |
| FR-FX-005 | RunTranslation | DDD representative command | `fxRunTranslationHandler` | finance.fx.run.translation |

### 9.13 Fiscal Period Management

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-FPM-001 | StartSoftClose | DDD representative command | `fpmStartSoftCloseHandler` | finance.fpm.start.soft.close |
| FR-FPM-002 | EndSoftClose | DDD representative command | `fpmEndSoftCloseHandler` | finance.fpm.end.soft.close |
| FR-FPM-003 | StartHardClose | DDD representative command | `fpmStartHardCloseHandler` | finance.fpm.start.hard.close |
| FR-FPM-004 | ResumeCloseRun | DDD representative command | `fpmResumeCloseRunHandler` | finance.fpm.resume.close.run |
| FR-FPM-005 | AbortCloseRun | DDD representative command | `fpmAbortCloseRunHandler` | finance.fpm.abort.close.run |
| FR-FPM-006 | ApplyPostingGateResult | DDD representative command | `fpmApplyPostingGateResultHandler` | finance.fpm.apply.posting.gate.result |
| FR-FPM-007 | ApplyCloseExceptionApprovalDecision | DDD representative command | `fpmApplyCloseExceptionApprovalDecisionHandler` | finance.fpm.apply.close.exception.approval.decision |
| FR-FPM-008 | ApplyCloseApprovalDecision | DDD representative command | `fpmApplyCloseApprovalDecisionHandler` | finance.fpm.apply.close.approval.decision |
| FR-FPM-009 | RequestReopen | DDD representative command | `fpmRequestReopenHandler` | finance.fpm.request.reopen |
| FR-FPM-010 | ApplyReopenApprovalDecision | DDD representative command | `fpmApplyReopenApprovalDecisionHandler` | finance.fpm.apply.reopen.approval.decision |
| FR-FPM-011 | StartReclose | DDD representative command | `fpmStartRecloseHandler` | finance.fpm.start.reclose |
| FR-FPM-012 | TakeOverPeriodControl | DDD representative command | `fpmTakeOverPeriodControlHandler` | finance.fpm.take.over.period.control |
| FR-FPM-013 | ExtendCloseException | DDD representative command | `fpmExtendCloseExceptionHandler` | finance.fpm.extend.close.exception |

### 9.14 COA Segment Accounting

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-COA-001 | Maintain segment definitions | PRD functional action | `coaMaintainSegmentDefinitionsHandler` | finance.coa.maintain.segment.definitions |
| FR-COA-002 | Maintain segment values | PRD functional action | `coaMaintainSegmentValuesHandler` | finance.coa.maintain.segment.values |
| FR-COA-003 | Validate segment combinations | PRD functional action | `coaValidateSegmentCombinationsHandler` | finance.coa.validate.segment.combinations |
| FR-COA-004 | Request segment changes | PRD functional action | `coaRequestSegmentChangesHandler` | finance.coa.request.segment.changes |
| FR-COA-005 | ApplySegmentChangeApprovalDecision | DDD detailed command | `coaApplySegmentChangeApprovalDecisionHandler` | finance.coa.apply.segment.change.approval.decision |

### 9.15 Bank Feeds & Reconciliation

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-BFR-001 | ImportStatement | DDD representative command | `bfrImportStatementHandler` | finance.bfr.import.statement |
| FR-BFR-002 | ProposeMatch | DDD representative command | `bfrProposeMatchHandler` | finance.bfr.propose.match |
| FR-BFR-003 | ConfirmMatch | DDD representative command | `bfrConfirmMatchHandler` | finance.bfr.confirm.match |
| FR-BFR-004 | Unmatch | DDD representative command | `bfrUnmatchHandler` | finance.bfr.unmatch |
| FR-BFR-005 | CompleteReconciliation | DDD representative command | `bfrCompleteReconciliationHandler` | finance.bfr.complete.reconciliation |
| FR-BFR-006 | Maintain bank-feed connections | PRD functional action | `bfrMaintainBankFeedConnectionsHandler` | finance.bfr.maintain.bank.feed.connections |

### 9.16 Tax Filing

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-TAX-001 | DetermineTax | DDD representative command | `taxDetermineTaxHandler` | finance.tax.determine.tax |
| FR-TAX-002 | PrepareTaxReturn | DDD representative command | `taxPrepareTaxReturnHandler` | finance.tax.prepare.tax.return |
| FR-TAX-003 | ApplyTaxReturnApprovalDecision | DDD representative command | `taxApplyTaxReturnApprovalDecisionHandler` | finance.tax.apply.tax.return.approval.decision |
| FR-TAX-004 | SubmitTaxReturn | DDD representative command | `taxSubmitTaxReturnHandler` | finance.tax.submit.tax.return |
| FR-TAX-005 | CreateTaxAmendment | DDD representative command | `taxCreateTaxAmendmentHandler` | finance.tax.create.tax.amendment |
| FR-TAX-006 | ApplyTaxAmendmentApprovalDecision | DDD representative command | `taxApplyTaxAmendmentApprovalDecisionHandler` | finance.tax.apply.tax.amendment.approval.decision |
| FR-TAX-007 | SubmitTaxAmendment | DDD representative command | `taxSubmitTaxAmendmentHandler` | finance.tax.submit.tax.amendment |
| FR-TAX-008 | CreateReturnLevelTaxAdjustment | DDD representative command | `taxCreateReturnLevelTaxAdjustmentHandler` | finance.tax.create.return.level.tax.adjustment |
| FR-TAX-009 | ApplyReturnLevelTaxAdjustmentApprovalDecision | DDD representative command | `taxApplyReturnLevelTaxAdjustmentApprovalDecisionHandler` | finance.tax.apply.return.level.tax.adjustment.approval.decision |
| FR-TAX-010 | PostReturnLevelTaxAdjustment | DDD representative command | `taxPostReturnLevelTaxAdjustmentHandler` | finance.tax.post.return.level.tax.adjustment |
| FR-TAX-011 | RequestTaxPayment | DDD representative command | `taxRequestTaxPaymentHandler` | finance.tax.request.tax.payment |
| FR-TAX-012 | RecordTaxPaymentSettlement | DDD representative command | `taxRecordTaxPaymentSettlementHandler` | finance.tax.record.tax.payment.settlement |
| FR-TAX-013 | ApplyIncomingSettlement | DDD representative command | `taxApplyIncomingSettlementHandler` | finance.tax.apply.incoming.settlement |
| FR-TAX-014 | ReverseIncomingSettlementApplication | DDD representative command | `taxReverseIncomingSettlementApplicationHandler` | finance.tax.reverse.incoming.settlement.application |
| FR-TAX-015 | ApplyPaymentReturn | DDD representative command | `taxApplyPaymentReturnHandler` | finance.tax.apply.payment.return |
| FR-TAX-016 | Maintain tax configurations | PRD functional action | `taxMaintainTaxConfigurationsHandler` | finance.tax.maintain.tax.configurations |

### 9.17 Workflow & Approvals

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-WFA-001 | CreateApprovalRequest | DDD representative command | `wfaCreateApprovalRequestHandler` | finance.wfa.create.approval.request |
| FR-WFA-002 | DecideApprovalRequest | DDD representative command | `wfaDecideApprovalRequestHandler` | finance.wfa.decide.approval.request |
| FR-WFA-003 | DelegateApproval | DDD representative command | `wfaDelegateApprovalHandler` | finance.wfa.delegate.approval |
| FR-WFA-004 | EscalateApproval | DDD representative command | `wfaEscalateApprovalHandler` | finance.wfa.escalate.approval |
| FR-WFA-005 | Maintain approval policies | PRD functional action | `wfaMaintainApprovalPoliciesHandler` | finance.wfa.maintain.approval.policies |

### 9.18 Identity & Access

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-IAM-001 | Manage users | PRD functional action | `iamManageUsersHandler` | finance.iam.manage.users |
| FR-IAM-002 | Manage roles | PRD functional action | `iamManageRolesHandler` | finance.iam.manage.roles |
| FR-IAM-003 | Manage access policies | PRD functional action | `iamManageAccessPoliciesHandler` | finance.iam.manage.access.policies |
| FR-IAM-004 | Manage segregation rules | PRD functional action | `iamManageSegregationRulesHandler` | finance.iam.manage.segregation.rules |
| FR-IAM-005 | Grant emergency access | PRD functional action | `iamGrantEmergencyAccessHandler` | finance.iam.grant.emergency.access |
| FR-IAM-006 | Revoke emergency access | PRD functional action | `iamRevokeEmergencyAccessHandler` | finance.iam.revoke.emergency.access |

### 9.19 Audit Integrity

| Requirement | Operation | Provenance | Handler | Permission |
|---|---|---|---|---|
| FR-AUD-001 | AppendAuditableEvent | DDD representative command | `audAppendAuditableEventHandler` | finance.aud.append.auditable.event |
| FR-AUD-002 | CreateAuditSeal | DDD representative command | `audCreateAuditSealHandler` | finance.aud.create.audit.seal |
| FR-AUD-003 | RotateVerificationCredential | DDD representative command | `audRotateVerificationCredentialHandler` | finance.aud.rotate.verification.credential |
| FR-AUD-004 | EscalateIntegrityIncident | DDD representative command | `audEscalateIntegrityIncidentHandler` | finance.aud.escalate.integrity.incident |
| FR-AUD-005 | VerifyProof | DDD reference operation | `audVerifyProofHandler` | finance.aud.verify.proof |

## 10. Configuration loading

Configuration is loaded once at startup from environment variables and secret references. Unknown fields, missing required fields or invalid ranges fail startup. Runtime-mutated finance policy belongs to authoritative domain configuration, not environment variables.

| Variable | Type | Default | Secret | Validation |
|---|---|---|---|---|
| `APP_ENV` | enum | `local` | No | `local`, `dev`, `demo`, `prod-reference` |
| `HTTP_ADDR` | string | `:8080` | No | valid listen address |
| `DATABASE_URL` | URI | none | Yes | PostgreSQL TLS outside local |
| `DB_MAX_CONNS` | integer | `20` | No | 2–200 |
| `HTTP_READ_TIMEOUT` | duration | `15s` | No | 1s–60s |
| `HTTP_WRITE_TIMEOUT` | duration | `30s` | No | 1s–120s |
| `OUTBOX_BATCH_SIZE` | integer | `100` | No | 1–1000 |
| `OUTBOX_POLL_INTERVAL` | duration | `1s` | No | 100ms–60s |
| `ENTRA_TENANT_ID` | UUID/string | none | No | required outside local |
| `ENTRA_API_AUDIENCE` | string | none | No | required outside local |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | URI | empty | No | valid URI when set |

## 11. Architecture tests

- Import graph test rejects cross-module adapter/repository imports.
- SQL ownership test rejects queries referencing a schema other than the owning module or approved platform schemas.
- Domain packages fail if they import HTTP, SQL, Azure or Terraform libraries.
- Every DDD-backed command has exactly one owning handler.
- Every handler has at least one authorization test, one domain rejection test, one idempotency test and one version-conflict test where mutable state is involved.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `b05cdd2c172a57233c874a919e06ce03adeb849fdb8fb8257d2c7b4e56f12a76` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
