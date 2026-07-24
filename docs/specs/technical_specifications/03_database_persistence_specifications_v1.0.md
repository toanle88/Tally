# Finance Platform Database and Persistence Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready persistence baseline |
| Database | PostgreSQL 18.x |

## 1. Database topology

- One PostgreSQL server and database per environment.
- One owned schema per bounded context plus platform schemas `platform`, `integration` and `audit`.
- Cross-context foreign keys and cross-schema write-path joins are prohibited.
- One migration history table is retained per schema through Goose.
- Production qualification enables PostgreSQL HA, backup and point-in-time recovery according to the NFR profile.

## 2. Schema and role catalog

| Schema | Owner | Read/write role | Read-only role | Primary aggregate tables |
|---|---|---|---|---|
| organization | Organization & Master Data | `app_organization_rw` | `app_organization_ro` | legal_entity, party, customer_profile, vendor_profile, fiscal_calendar |
| gl | General Ledger | `app_gl_rw` | `app_gl_ro` | ledger, accounting_book, chart_of_accounts, account, journal_entry, period_posting_gate |
| ap | Accounts Payable | `app_ap_rw` | `app_ap_ro` | vendor_invoice, payment_request |
| ar | Accounts Receivable | `app_ar_rw` | `app_ar_ro` | customer_invoice, receivable_open_item, customer_receipt, credit_note, customer_refund_request |
| payroll | Payroll | `app_payroll_rw` | `app_payroll_ro` | payroll_run, employee_payroll_profile, payroll_tax_filing |
| invoicing | Invoicing | `app_invoicing_rw` | `app_invoicing_ro` | invoice_template, billing_schedule, generated_invoice |
| payments | Payments & Cash Management | `app_payments_rw` | `app_payments_ro` | bank_account, payment_batch, payment_instruction, payment_return, expected_incoming_settlement, unallocated_incoming_settlement, settlement_receipt |
| reporting | Financial Reporting | `app_reporting_rw` | `app_reporting_ro` | report_definition, consolidation_run, financial_statement |
| intercompany | Multi-Entity / Intercompany | `app_intercompany_rw` | `app_intercompany_ro` | intercompany_agreement, intercompany_transaction, settlement_run, elimination_run |
| revenue | Revenue Recognition | `app_revenue_rw` | `app_revenue_ro` | revenue_contract, revenue_schedule, contract_modification |
| fixed_assets | Fixed Assets | `app_fixed_assets_rw` | `app_fixed_assets_ro` | fixed_asset, depreciation_run, impairment_assessment, asset_disposal |
| multi_currency | Multi-Currency | `app_multi_currency_rw` | `app_multi_currency_ro` | currency_rate_set, revaluation_run, translation_run |
| fiscal_period | Fiscal Period Management | `app_fiscal_period_rw` | `app_fiscal_period_ro` | fiscal_period, soft_close_run, close_run, reopen_request |
| coa | COA Segment Accounting | `app_coa_rw` | `app_coa_ro` | segment_definition, segment_combination, segment_change_request |
| bank_reconciliation | Bank Feeds & Reconciliation | `app_bank_reconciliation_rw` | `app_bank_reconciliation_ro` | bank_feed_connection, bank_statement, reconciliation_session |
| tax | Tax Filing | `app_tax_rw` | `app_tax_ro` | tax_configuration, tax_return, filing_submission, tax_amendment, return_level_tax_adjustment, tax_payment_obligation |
| workflow | Workflow & Approvals | `app_workflow_rw` | `app_workflow_ro` | approval_policy, approval_request, delegation |
| identity | Identity & Access | `app_identity_rw` | `app_identity_ro` | user, role, access_policy, segregation_rule |
| audit | Audit Integrity | `app_audit_rw` | `app_audit_ro` | audit_chain |

## 3. Common column standards

| Column | Type | Rule |
|---|---|---|
| `<aggregate>_id` | `uuid` | Primary identity; UUID v7 for application-created records |
| `accounting_scope_id` | `uuid` | Required on ledger-bound records |
| `aggregate_version` | `bigint` | Starts at 1 and increments once per aggregate transition |
| `status` | `text` | Check-constrained to DDD lifecycle values |
| `created_at` / `updated_at` | `timestamptz` | UTC; database supplied |
| `created_by` / `updated_by` | `uuid` | Application actor identity |
| `data_classification` | `text` | `internal`, `confidential`, or `highly_restricted` |
| `correlation_id` / `causation_id` | `uuid` | Required for material transitions |

## 4. Aggregate table catalog

| Module | Aggregate | Primary table | Primary key | Storage intent |
|---|---|---|---|---|
| OMD | LegalEntity | `organization.legal_entity` | `legal_entity_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| OMD | Party | `organization.party` | `party_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| OMD | CustomerProfile | `organization.customer_profile` | `customer_profile_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| OMD | VendorProfile | `organization.vendor_profile` | `vendor_profile_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| OMD | FiscalCalendar | `organization.fiscal_calendar` | `fiscal_calendar_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| GL | Ledger | `gl.ledger` | `ledger_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| GL | AccountingBook | `gl.accounting_book` | `accounting_book_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| GL | ChartOfAccounts | `gl.chart_of_accounts` | `chart_of_accounts_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| GL | Account | `gl.account` | `account_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| GL | JournalEntry | `gl.journal_entry` | `journal_entry_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| GL | PeriodPostingGate | `gl.period_posting_gate` | `period_posting_gate_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AP | VendorInvoice | `ap.vendor_invoice` | `vendor_invoice_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AP | PaymentRequest | `ap.payment_request` | `payment_request_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AR | CustomerInvoice | `ar.customer_invoice` | `customer_invoice_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AR | ReceivableOpenItem | `ar.receivable_open_item` | `receivable_open_item_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AR | CustomerReceipt | `ar.customer_receipt` | `customer_receipt_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AR | CreditNote | `ar.credit_note` | `credit_note_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AR | CustomerRefundRequest | `ar.customer_refund_request` | `customer_refund_request_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PAYR | PayrollRun | `payroll.payroll_run` | `payroll_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PAYR | EmployeePayrollProfile | `payroll.employee_payroll_profile` | `employee_payroll_profile_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PAYR | PayrollTaxFiling | `payroll.payroll_tax_filing` | `payroll_tax_filing_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| INV | InvoiceTemplate | `invoicing.invoice_template` | `invoice_template_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| INV | BillingSchedule | `invoicing.billing_schedule` | `billing_schedule_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| INV | GeneratedInvoice | `invoicing.generated_invoice` | `generated_invoice_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | BankAccount | `payments.bank_account` | `bank_account_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | PaymentBatch | `payments.payment_batch` | `payment_batch_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | PaymentInstruction | `payments.payment_instruction` | `payment_instruction_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | PaymentReturn | `payments.payment_return` | `payment_return_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | ExpectedIncomingSettlement | `payments.expected_incoming_settlement` | `expected_incoming_settlement_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | UnallocatedIncomingSettlement | `payments.unallocated_incoming_settlement` | `unallocated_incoming_settlement_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| PCM | SettlementReceipt | `payments.settlement_receipt` | `settlement_receipt_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| RPT | ReportDefinition | `reporting.report_definition` | `report_definition_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| RPT | ConsolidationRun | `reporting.consolidation_run` | `consolidation_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| RPT | FinancialStatement | `reporting.financial_statement` | `financial_statement_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IC | IntercompanyAgreement | `intercompany.intercompany_agreement` | `intercompany_agreement_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IC | IntercompanyTransaction | `intercompany.intercompany_transaction` | `intercompany_transaction_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IC | SettlementRun | `intercompany.settlement_run` | `settlement_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IC | EliminationRun | `intercompany.elimination_run` | `elimination_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| REV | RevenueContract | `revenue.revenue_contract` | `revenue_contract_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| REV | RevenueSchedule | `revenue.revenue_schedule` | `revenue_schedule_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| REV | ContractModification | `revenue.contract_modification` | `contract_modification_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FA | FixedAsset | `fixed_assets.fixed_asset` | `fixed_asset_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FA | DepreciationRun | `fixed_assets.depreciation_run` | `depreciation_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FA | ImpairmentAssessment | `fixed_assets.impairment_assessment` | `impairment_assessment_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FA | AssetDisposal | `fixed_assets.asset_disposal` | `asset_disposal_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FX | CurrencyRateSet | `multi_currency.currency_rate_set` | `currency_rate_set_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FX | RevaluationRun | `multi_currency.revaluation_run` | `revaluation_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FX | TranslationRun | `multi_currency.translation_run` | `translation_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FPM | FiscalPeriod | `fiscal_period.fiscal_period` | `fiscal_period_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FPM | SoftCloseRun | `fiscal_period.soft_close_run` | `soft_close_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FPM | CloseRun | `fiscal_period.close_run` | `close_run_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| FPM | ReopenRequest | `fiscal_period.reopen_request` | `reopen_request_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| COA | SegmentDefinition | `coa.segment_definition` | `segment_definition_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| COA | SegmentCombination | `coa.segment_combination` | `segment_combination_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| COA | SegmentChangeRequest | `coa.segment_change_request` | `segment_change_request_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| BFR | BankFeedConnection | `bank_reconciliation.bank_feed_connection` | `bank_feed_connection_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| BFR | BankStatement | `bank_reconciliation.bank_statement` | `bank_statement_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| BFR | ReconciliationSession | `bank_reconciliation.reconciliation_session` | `reconciliation_session_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| TAX | TaxConfiguration | `tax.tax_configuration` | `tax_configuration_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| TAX | TaxReturn | `tax.tax_return` | `tax_return_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| TAX | FilingSubmission | `tax.filing_submission` | `filing_submission_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| TAX | TaxAmendment | `tax.tax_amendment` | `tax_amendment_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| TAX | ReturnLevelTaxAdjustment | `tax.return_level_tax_adjustment` | `return_level_tax_adjustment_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| TAX | TaxPaymentObligation | `tax.tax_payment_obligation` | `tax_payment_obligation_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| WFA | ApprovalPolicy | `workflow.approval_policy` | `approval_policy_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| WFA | ApprovalRequest | `workflow.approval_request` | `approval_request_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| WFA | Delegation | `workflow.delegation` | `delegation_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IAM | User | `identity.user` | `user_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IAM | Role | `identity.role` | `role_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IAM | AccessPolicy | `identity.access_policy` | `access_policy_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| IAM | SegregationRule | `identity.segregation_rule` | `segregation_rule_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |
| AUD | AuditChain | `audit.audit_chain` | `audit_chain_id uuid` | Mutable aggregate row or append-only root according to DDD lifecycle |

## 5. Monetary representation

```sql
create domain platform.currency_code as text
  check (value ~ '^[A-Z]3$');

create domain platform.money_amount as numeric(38,12);
```

- Domain policies enforce the allowed scale and rounding mode for each operation.
- `numeric` values are never converted through binary floating point.
- A money value is stored as amount plus currency; functional and presentation amounts use separate labeled columns.

## 6. Platform tables

```sql
create table platform.idempotency_record (
  scope_key text not null,
  idempotency_key text not null,
  canonical_fingerprint text not null,
  operation_id text not null,
  state text not null check (state in ('in_progress','established','failed')),
  result_status integer,
  result_body jsonb,
  aggregate_id uuid,
  process_id uuid,
  created_at timestamptz not null default clock_timestamp(),
  expires_at timestamptz not null,
  primary key (scope_key, idempotency_key)
);

create table integration.outbox (
  outbox_id uuid primary key,
  event_type text not null,
  event_version integer not null,
  source_context text not null,
  aggregate_id uuid not null,
  aggregate_version bigint not null,
  accounting_scope_id uuid,
  correlation_id uuid not null,
  causation_id uuid not null,
  payload jsonb not null,
  payload_fingerprint text not null,
  available_at timestamptz not null default clock_timestamp(),
  claimed_until timestamptz,
  claim_owner text,
  attempt_count integer not null default 0,
  established_at timestamptz,
  last_error_code text,
  created_at timestamptz not null default clock_timestamp(),
  unique (source_context, aggregate_id, aggregate_version, event_type)
);

create table integration.inbox (
  consumer_name text not null,
  message_id uuid not null,
  message_fingerprint text not null,
  state text not null check (state in ('processing','established','failed')),
  result_reference jsonb,
  first_received_at timestamptz not null default clock_timestamp(),
  established_at timestamptz,
  primary key (consumer_name, message_id)
);
```

## 7. High-integrity DDL baselines

### 7.1 GL journal

```sql
create table gl.journal_entry (
  journal_entry_id uuid primary key,
  accounting_scope_id uuid not null,
  journal_number text,
  posting_date date not null,
  fiscal_period_id uuid not null,
  status text not null check (status in ('Draft','PendingApproval','Posted','Cancelled')),
  transaction_currency platform.currency_code not null,
  source_context text not null,
  source_aggregate_id uuid not null,
  source_version bigint not null,
  idempotency_key text not null,
  canonical_fingerprint text not null,
  reversal_of_journal_entry_id uuid references gl.journal_entry(journal_entry_id),
  aggregate_version bigint not null,
  created_at timestamptz not null default clock_timestamp(),
  updated_at timestamptz not null default clock_timestamp(),
  unique (accounting_scope_id, idempotency_key),
  unique (accounting_scope_id, source_context, source_aggregate_id, source_version)
);

create table gl.journal_entry_line (
  journal_entry_id uuid not null references gl.journal_entry(journal_entry_id),
  line_number integer not null,
  account_id uuid not null,
  debit_or_credit text not null check (debit_or_credit in ('Debit','Credit')),
  transaction_amount platform.money_amount not null,
  functional_amount platform.money_amount not null,
  segment_combination_id uuid,
  line_reference text,
  primary key (journal_entry_id, line_number)
);
```

Posting additionally validates balanced transaction and functional totals inside the GL transaction before status becomes `Posted`.

### 7.2 AR receipt application

`ar.customer_receipt` is locked first. Affected `ar.receivable_open_item` rows are locked in ascending UUID order. Immutable `ar.receipt_application` rows and the receipt/open-item balances commit together. Unique constraints prevent an application identity from being established twice.

### 7.3 Payments return

`payments.payment_instruction` is locked before `payments.payment_return`. Check constraints enforce nonnegative balances; the handler enforces the DDD equations involving settled, reserved, posted, reversed and reconciled return amounts before commit.

### 7.4 Period gate

`gl.period_posting_gate` is unique by `(accounting_scope_id, fiscal_period_id)`. Gate transition, version increment, active admission counters and frozen summary history commit with the admission decision. Finalized command results are immutable and keyed by command fingerprint.

## 8. Index baseline

| Workload | Required index pattern |
|---|---|
| Aggregate lookup | Primary key plus `(accounting_scope_id, <id>)` where scope-bound |
| Worklists | Partial/composite index on scope, status, owner and business date |
| Idempotency | Unique scope/key; expiry index |
| Outbox dispatch | Partial index on `(available_at, outbox_id)` where `established_at is null` |
| Inbox dedupe | Primary consumer/message identity |
| Journal source lookup | Unique scope/source context/source aggregate/source version |
| Timeline | `(aggregate_id, occurred_at, sequence)` |
| Reporting projection refresh | Source watermark and projection state |

Every added index requires the target query, expected cardinality and `EXPLAIN (ANALYZE, BUFFERS)` evidence in a representative dataset.

## 9. Migration policy

1. Migrations are forward-only in shared environments; rollback uses a reviewed compensating migration or restored disposable environment.
2. Expand/migrate/contract separates incompatible changes across releases.
3. A migration never rewrites a large financial table in one unbounded transaction.
4. Backfills are resumable, idempotent, observable and reconciliation-tested.
5. Schema ownership roles are applied by migration and verified in CI.
6. Destructive contract steps require evidence that old application versions and data paths are retired.

## 10. Retention, legal hold and archival

- Retention policy is represented by record type, jurisdiction and accounting scope.
- Legal hold blocks destruction while preserving business correction behavior.
- Archive exports include checksums, schema version and lineage references.
- Audit and posted financial facts are never deleted solely because an operational projection expires.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `af929d5aed082a5424fcb1ddde20cb3bb75faffc9d55641ecd3e56bfc09a4158` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
