# Finance Platform Testing, Performance and Recovery Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Release-gate technical baseline |

## 1. Test layers

| Layer | Tool | Purpose |
|---|---|---|
| Domain unit | Go `testing` | Invariants, lifecycle, arithmetic, decisions |
| Application | Go `testing` with fakes | Authorization, idempotency, orchestration and error mapping |
| Repository integration | Testcontainers + PostgreSQL 18 | DDL, queries, locks, constraints and migrations |
| API contract | OpenAPI validators | Request/response/error compatibility |
| Frontend unit/component | Vitest + Testing Library + MSW | UI semantics and API states |
| End-to-end | Playwright | User workflows, permissions, recovery and accessibility |
| Performance | k6 or Go load harness | NFR latency, throughput and capacity |
| Failure injection | Testcontainers/proxy controls | Dependency failure, worker crash, lock contention and ambiguous outcomes |
| Recovery/DR | Terraform + restore scripts | Backup restore, reconciliation, RTO/RPO evidence |

## 2. Capability coverage

| Prefix | Capability | FR count | Backend tests | Frontend tests | Minimum layers |
|---|---|---|---|---|---|
| OMD | Organization & Master Data | 6 | `internal/organization/...` | `web/src/capabilities/master-data/...` | Domain + application + DB + API + UI |
| GL | General Ledger | 18 | `internal/gl/...` | `web/src/capabilities/general-ledger/...` | Domain + application + DB + API + UI |
| AP | Accounts Payable | 10 | `internal/ap/...` | `web/src/capabilities/accounts-payable/...` | Domain + application + DB + API + UI |
| AR | Accounts Receivable | 16 | `internal/ar/...` | `web/src/capabilities/accounts-receivable/...` | Domain + application + DB + API + UI |
| PAYR | Payroll | 7 | `internal/payroll/...` | `web/src/capabilities/payroll/...` | Domain + application + DB + API + UI |
| INV | Invoicing | 6 | `internal/invoicing/...` | `web/src/capabilities/invoicing/...` | Domain + application + DB + API + UI |
| PCM | Payments & Cash Management | 25 | `internal/payments/...` | `web/src/capabilities/payments/...` | Domain + application + DB + API + UI |
| RPT | Financial Reporting | 6 | `internal/reporting/...` | `web/src/capabilities/reporting/...` | Domain + application + DB + API + UI |
| IC | Multi-Entity / Intercompany | 11 | `internal/intercompany/...` | `web/src/capabilities/intercompany/...` | Domain + application + DB + API + UI |
| REV | Revenue Recognition | 6 | `internal/revenue/...` | `web/src/capabilities/revenue-recognition/...` | Domain + application + DB + API + UI |
| FA | Fixed Assets | 21 | `internal/fixedassets/...` | `web/src/capabilities/fixed-assets/...` | Domain + application + DB + API + UI |
| FX | Multi-Currency | 5 | `internal/multicurrency/...` | `web/src/capabilities/multi-currency/...` | Domain + application + DB + API + UI |
| FPM | Fiscal Period Management | 13 | `internal/fiscalperiod/...` | `web/src/capabilities/fiscal-periods/...` | Domain + application + DB + API + UI |
| COA | COA Segment Accounting | 5 | `internal/coa/...` | `web/src/capabilities/coa-segments/...` | Domain + application + DB + API + UI |
| BFR | Bank Feeds & Reconciliation | 6 | `internal/bankfeeds/...` | `web/src/capabilities/bank-reconciliation/...` | Domain + application + DB + API + UI |
| TAX | Tax Filing | 16 | `internal/tax/...` | `web/src/capabilities/tax/...` | Domain + application + DB + API + UI |
| WFA | Workflow & Approvals | 5 | `internal/workflow/...` | `web/src/capabilities/approvals/...` | Domain + application + DB + API + UI |
| IAM | Identity & Access | 6 | `internal/identity/...` | `web/src/capabilities/identity-access/...` | Domain + application + DB + API + UI |
| AUD | Audit Integrity | 5 | `internal/audit/...` | `web/src/capabilities/audit-integrity/...` | Domain + application + DB + API + UI |

## 3. Workflow end-to-end catalog

| Workflow | Title | Playwright specification | Required paths |
|---|---|---|---|
| WF-6.1 | Period Close: Hard Close | `e2e/wf-6-1.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-6.2 | Fiscal Period Reopen and Reclose | `e2e/wf-6-2.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-6.3 | Intercompany Reconciliation and Settlement | `e2e/wf-6-3.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-6.4 | Fixed Asset Disposal with Gain or Loss Recognition | `e2e/wf-6-4.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-6.5 | Revenue Recognition for a SaaS Contract | `e2e/wf-6-5.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-6.6 | Journal Entry Posting and Reversal | `e2e/wf-6-6.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-6.7 | Customer Receipt Recording with Partial Application | `e2e/wf-6-7.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.1 | Vendor Invoice Registration, Matching, Approval, Dispute, and Void | `e2e/wf-7-1.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.2 | Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation | `e2e/wf-7-2.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.3 | Customer Credit, Refund, Overpayment, Chargeback, and Write-Off | `e2e/wf-7-3.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.4 | Bank Statement Import, Matching, Unmatching, and Reconciliation | `e2e/wf-7-4.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.5 | Foreign-Currency Invoice Settlement and Realized FX | `e2e/wf-7-5.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.6 | Period-End Revaluation, Rerun, and Next-Period Reversal | `e2e/wf-7-6.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.7 | Full Fixed-Asset Lifecycle and Disposal Variants | `e2e/wf-7-7.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.8 | Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration | `e2e/wf-7-8.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.9 | Consolidation, Ownership Changes, Translation, Eliminations, and Rerun | `e2e/wf-7-9.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.10 | Tax Return Submission, Rejection, Amendment, Payment, and Evidence | `e2e/wf-7-10.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.11 | Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment | `e2e/wf-7-11.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.12 | Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen | `e2e/wf-7-12.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.13 | Cross-Context Event Interpretation, Ordering, and Replay | `e2e/wf-7-13.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.14 | Concurrent Aggregate and Domain-Process Modification Rules | `e2e/wf-7-14.spec.ts` | Primary, exception, recovery and authorization paths |
| WF-7.15 | Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation | `e2e/wf-7-15.spec.ts` | Primary, exception, recovery and authorization paths |

## 4. Mandatory financial-integrity tests

1. Journal transaction and functional totals balance before posting.
2. Posted journal and immutable facts cannot be destructively edited.
3. Same idempotency identity/fingerprint returns the original result; changed fingerprint conflicts.
4. Concurrent receipt applications cannot over-apply receipt or open item.
5. Payment-return reserved/posted/reversed/reconciled equations hold under concurrent delivery.
6. Period gate ownership is exclusive and stale gate versions reject admission.
7. Reclose is mandatory after admitted reopen posting.
8. Fixed-asset multi-leg partial success retries only missing legs and preserves successful references.
9. Inbox redelivery never repeats a local financial effect.
10. Restore/replay/reconciliation reproduces authoritative balances and audit sequence.

## 5. Performance qualification

The NFR specification is authoritative for thresholds. Test plans record dataset, concurrency, request mix, warm-up, duration, Azure topology, database size, result percentiles and resource saturation.

| Profile | Purpose | Minimum characteristics |
|---|---|---|
| Local developer | Fast feedback | Reduced dataset; no NFR claim |
| Azure learning | Deployment exercise | Scale-to-zero permitted; no NFR claim |
| Production qualification | Formal evidence | Min replicas, PostgreSQL HA/private topology, representative dataset and sustained load |

Performance tests fail if the target is met only by violating integrity, dropping accepted work, disabling audit, bypassing authorization or using unrealistic cached data.

## 6. Concurrency and failure matrix

| Scenario | Injection | Pass condition |
|---|---|---|
| Optimistic conflict | Two commands use same expected version | One establishes; one receives typed conflict |
| Deterministic locks | Reverse lock-request order in concurrent callers | Implementation still acquires documented order; no deadlock/over-allocation |
| Worker crash | Terminate after external call before establishment | Result lookup/reconciliation avoids duplicate effect |
| Outbox lease expiry | Pause claimant beyond lease | New claimant establishes; stale claimant is fenced |
| Database failover/connection loss | Drop connections during transaction | No partial local commit; safe retry/result lookup |
| Provider duplicate callback | Deliver identical message repeatedly | One inbox/local effect; established result returned |
| Out-of-order prerequisite | Deliver dependent event first | Managed pending/exception; no fabricated success |

## 7. Migration tests

- Empty-database migration to current version.
- Upgrade from every supported release baseline.
- Expand/migrate/contract compatibility with previous application version.
- Backfill interruption/resume and duplicate execution.
- Constraint validation and representative query plans.
- Production-sized migration timing in qualification environment.

## 8. Security and accessibility tests

- Token issuer/audience/signature/expiry failures.
- Permission, accounting-scope, amount, segment, data classification and segregation denials.
- Emergency grant expiry and post-use evidence.
- OWASP API/web controls, dependency and container scanning.
- WCAG 2.2 AA automated checks plus manual keyboard and screen-reader verification for critical workflows.

## 9. Recovery tests

At least quarterly for a production-qualified environment: restore PostgreSQL to an isolated environment, apply migrations, verify audit/evidence sequence, reconcile control totals and source watermarks, test external-outcome reconciliation, and record measured RTO/RPO.

## 10. Release pass conditions

- All changed requirement mappings have passing tests.
- No critical/high unresolved security finding without approved exception.
- OpenAPI, generated client, sqlc output and migrations have no drift.
- Financial-integrity, concurrency, idempotency and recovery suites pass.
- NFR qualification evidence passes for any production claim.
- Manual accessibility evidence exists for changed critical journeys.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `52f247411282c3762be09413945ce06266e4b567456f711896a894d851e6564f` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
