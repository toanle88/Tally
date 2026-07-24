# Finance Platform Frontend and UI Technical Specifications

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Implementation-ready frontend baseline |
| Stack | React 19.2, TypeScript, Vite 8, Tailwind CSS 4, daisyUI 5 |

## 1. Frontend structure

```text
web/src/app/
web/src/routes/
web/src/components/
web/src/capabilities/<capability>/api/
web/src/capabilities/<capability>/components/
web/src/capabilities/<capability>/pages/
web/src/capabilities/<capability>/schemas/
web/src/lib/auth/
web/src/lib/http/
web/src/lib/i18n/
web/src/test/
```

Capability code cannot import another capability's private components or API layer. Shared components contain no capability-specific finance rules.

## 2. Application state

- TanStack Query owns server state; query keys include capability, accounting scope, record identity and relevant filter/version.
- React Hook Form owns form state; Zod provides client validation generated or reconciled with OpenAPI.
- The server remains authoritative for domain, authorization, version and idempotency validation.
- No global client store duplicates authoritative financial records.
- Mutations invalidate exact affected keys and then use the server-established result.

## 3. Shared component implementation catalog

| ID | Component | Required behavior | Implementation path |
|---|---|---|---|
| CMP-001 | Accounting Scope Selector | Shows current accounting/business scope and prevents submission against a stale or unintended scope. | `src/components/accounting-scope-selector/accounting-scope-selector.tsx` |
| CMP-002 | Record Identity Header | Shows record ID/type, authoritative capability, scope, state, version, source, owner, and sensitivity label. | `src/components/record-identity-header/record-identity-header.tsx` |
| CMP-003 | State-Aware Action Bar | Shows permitted and blocked actions with reason; actions refresh after every material outcome. | `src/components/state-aware-action-bar/state-aware-action-bar.tsx` |
| CMP-004 | Lifecycle Timeline | Shows established state transitions, actors, timestamps, decisions, and linked correction/replacement paths. | `src/components/lifecycle-timeline/lifecycle-timeline.tsx` |
| CMP-005 | Approval Panel | Shows request, policy/version, subject snapshot, steps, decisions, delegation/escalation, and decision-application state. | `src/components/approval-panel/approval-panel.tsx` |
| CMP-006 | Posting Panel | Shows posting identity, purpose, request/result, journal references, period/gate evidence, failure, retry, reversal, and reconciliation. | `src/components/posting-panel/posting-panel.tsx` |
| CMP-007 | Money and Currency Panel | Shows labeled transaction/functional/presentation amounts, rounding, rate evidence, and signed gain/loss convention. | `src/components/money-and-currency-panel/money-and-currency-panel.tsx` |
| CMP-008 | Settlement and Reconciliation Panel | Shows gross, returned/reversed/cancelled/remaining/net balances, owner acknowledgement, reconciliation, and exceptions. | `src/components/settlement-and-reconciliation-panel/settlement-and-reconciliation-panel.tsx` |
| CMP-009 | Correction Lineage Panel | Links original fact, reversal, amendment, return, unapplication, replacement, compensation, and supersession. | `src/components/correction-lineage-panel/correction-lineage-panel.tsx` |
| CMP-010 | Validation Summary | Groups field, line, business-rule, authorization, dependency, and conflict results with navigation to the cause. | `src/components/validation-summary/validation-summary.tsx` |
| CMP-011 | Exception Resolution Panel | Shows exception type, amount/scope, owner, evidence, aging, authorized outcomes, and resulting state. | `src/components/exception-resolution-panel/exception-resolution-panel.tsx` |
| CMP-012 | Version Conflict Dialog | Shows expected/current version and state, changed fields, current owner, established-result lookup, and safe retry status. | `src/components/version-conflict-dialog/version-conflict-dialog.tsx` |
| CMP-013 | Evidence Drawer | Shows access-controlled source, approval, posting, provider/authority, reconciliation, close, statement, and audit evidence. | `src/components/evidence-drawer/evidence-drawer.tsx` |
| CMP-014 | Worklist and Saved Filters | Supports applicable filters, columns, assignment, aging, bulk selection where allowed, exports, and saved views. | `src/components/worklist-and-saved-filters/worklist-and-saved-filters.tsx` |
| CMP-015 | Sensitive Data Guard | Masks restricted values, labels restricted sections, prevents unauthorized export, and records denied access. | `src/components/sensitive-data-guard/sensitive-data-guard.tsx` |
| CMP-016 | Legal Hold Indicator | Shows hold status and prevents destructive retention actions without changing business lifecycle actions. | `src/components/legal-hold-indicator/legal-hold-indicator.tsx` |
| CMP-017 | Result Lookup | Finds the established in-progress or terminal result by business identity/fingerprint after duplicate or ambiguous submission. | `src/components/result-lookup/result-lookup.tsx` |
| CMP-018 | Process Progress Panel | Shows cross-capability steps, responsible owner, current/pending/failed/reconciled states, and links to authoritative records. | `src/components/process-progress-panel/process-progress-panel.tsx` |

Application components wrap daisyUI classes and expose typed props. Feature code must not compose raw modal, alert, badge or input classes when an approved wrapper exists.

## 4. Screen and route catalog

| Screen ID | Screen | Route | Technical responsibility |
|---|---|---|---|
| XCT-WS-01 | Cross-context event exception worklist | `/operations/xct-ws-01` | A shared operational view filtered to the receiving capability; it shows event outcomes and links resolution to the authoritative receiving record without transferring ownership. |
| XCT-SCR-01 | Cross-context event outcome detail | `/operations/xct-scr-01` | Shows event identity, contract version, scope, source version, receiving outcome, prerequisites, local effect, evidence, and authorized resolution in the receiving capability. |
| CON-SCR-01 | Concurrency conflict and safe-retry view | `/operations/con-scr-01` | Shows attempted action, expected/current version and state, changed values, active owner/process epoch, established-result lookup, and whether deliberate retry is permitted. |
| OMD-WS-01 | Master-data worklist | `/master-data/omd-ws-01` | Search, review, and route legal-entity, party, profile, calendar, and publication work. |
| OMD-SCR-01 | Legal-entity record | `/master-data/omd-scr-01` | Maintain identity, registrations, addresses, ownership interests, and effective dates. |
| OMD-SCR-02 | Party record | `/master-data/omd-scr-02` | Maintain party identity, status, contacts, addresses, classifications, and bank-detail references. |
| OMD-SCR-03 | Customer and vendor profiles | `/master-data/omd-scr-03` | Maintain customer terms and vendor payment/remittance attributes. |
| OMD-SCR-04 | Fiscal-calendar editor | `/master-data/omd-scr-04` | Define calendar patterns and periods and review dependent-scope impact. |
| OMD-SCR-05 | Publication review | `/master-data/omd-scr-05` | Review approvals, effective dates where defined, validation results, and dependent-capability availability. |
| GL-WS-01 | Journal workbench | `/general-ledger/gl-ws-01` | Create, import, validate, approve, post, search, and reverse journals. |
| GL-SCR-01 | Journal detail | `/general-ledger/gl-scr-01` | Show scope, source, lines, balancing, approval, posting state, and correction lineage. |
| GL-SCR-02 | Posting result and conflict view | `/general-ledger/gl-scr-02` | Show established results, rejections, approval pending, idempotency conflicts, and safe next actions. |
| GL-SCR-03 | Posting-gate monitor | `/general-ledger/gl-scr-03` | Show gate mode, owner, version, admission summary, barrier position, expiry, and allowed posting purposes. |
| GL-SCR-04 | Ledger and accounting-book configuration | `/general-ledger/gl-scr-04` | Maintain ledgers, books, accounting basis, currencies, calendars, and posting policies. |
| GL-SCR-05 | Chart and account configuration | `/general-ledger/gl-scr-05` | Maintain charts, accounts, restrictions, currency policy, and reporting mappings. |
| AP-WS-01 | Vendor-invoice worklist | `/accounts-payable/ap-ws-01` | Manage received, duplicate-suspected, unmatched, approval, dispute, posting, and payment states. |
| AP-SCR-01 | Vendor-invoice detail | `/accounts-payable/ap-scr-01` | Show invoice, lines, tax, snapshots, duplicate evidence, approval, posting, balance, and correction lineage. |
| AP-SCR-02 | Match review | `/accounts-payable/ap-scr-02` | Compare invoice lines with immutable purchase-order and receipt snapshots and tolerance results. |
| AP-SCR-03 | Dispute and void panel | `/accounts-payable/ap-scr-03` | Record reason, evidence, affected amount, allowed state transition, and correction outcome. |
| AP-SCR-04 | Payment-request builder | `/accounts-payable/ap-scr-04` | Allocate approved invoice obligations to a payment request and validate totals/currency. |
| AP-SCR-05 | Settlement and return view | `/accounts-payable/ap-scr-05` | Show incoming applications, payment results, returns, remaining liability, and reconciliation. |
| AR-WS-01 | Receivables worklist | `/accounts-receivable/ar-ws-01` | Manage invoices, open items, receipts, unapplied cash, credits, refunds, chargebacks, and write-offs. |
| AR-SCR-01 | Customer invoice and open-item detail | `/accounts-receivable/ar-scr-01` | Show invoice, open amount, applications, adjustments, aging, and lineage. |
| AR-SCR-02 | Receipt workbench | `/accounts-receivable/ar-scr-02` | Record receipts and show receipt-accounting, applied/unapplied balances, and bank evidence. |
| AR-SCR-03 | Receipt allocation workspace | `/accounts-receivable/ar-scr-03` | Allocate one receipt to one or more open items with real-time availability and validation. |
| AR-SCR-04 | Application and unapplication detail | `/accounts-receivable/ar-scr-04` | Show immutable application facts, batch accounting, adjustments, rollback evidence, and journal results. |
| AR-SCR-05 | Credit and customer-adjustment workspace | `/accounts-receivable/ar-scr-05` | Create credit notes, overpayment resolutions, chargebacks, and write-offs. |
| AR-SCR-06 | Customer-refund workspace | `/accounts-receivable/ar-scr-06` | Manage approval, clearing legs, payment instructions, settlement, cancellation, return correction, and replacement. |
| PAYR-WS-01 | Payroll-run worklist | `/payroll/payr-ws-01` | Manage regular, off-cycle, correction, approval, posting, payment, and settlement states. |
| PAYR-SCR-01 | Payroll-run detail | `/payroll/payr-scr-01` | Show pay-group totals, employee-level restricted results, approval, posting, and payment status. |
| PAYR-SCR-02 | Payroll variance and validation review | `/payroll/payr-scr-02` | Review gross, deductions, net, tax, exceptions, and changes from prior runs. |
| PAYR-SCR-03 | Payroll correction workspace | `/payroll/payr-scr-03` | Create linked correction or off-cycle runs without overwriting finalized facts. |
| PAYR-SCR-04 | Employee payroll profile | `/payroll/payr-scr-04` | Maintain pay group, tax profile reference, and payment method reference under restricted access. |
| PAYR-SCR-05 | Payroll tax-filing record | `/payroll/payr-scr-05` | Maintain payroll-source filing records and links to statutory amendments. |
| INV-WS-01 | Billing operations worklist | `/invoicing/inv-ws-01` | Manage templates, schedules, generated invoices, recalculation, finalization, and cancellation. |
| INV-SCR-01 | Invoice-template editor | `/invoicing/inv-scr-01` | Maintain billing-frequency, charge-rule, tax-category, and payment-method definitions. |
| INV-SCR-02 | Billing-schedule editor | `/invoicing/inv-scr-02` | Maintain customer/contract schedule, charges, next billing date, and schedule status. |
| INV-SCR-03 | Invoice-generation run | `/invoicing/inv-scr-03` | Show source inputs, generated records, warnings, failures, and rerun eligibility. |
| INV-SCR-04 | Generated-invoice preview | `/invoicing/inv-scr-04` | Review lines, money, taxes, source version, and finalization blockers. |
| INV-SCR-05 | Finalization and AR handoff | `/invoicing/inv-scr-05` | Confirm finalization, show immutable invoice version, and track AR acceptance or rejection. |
| PCM-WS-01 | Treasury worklist | `/payments/pcm-ws-01` | Manage payment batches, instructions, returns, incoming expectations, receipts, exceptions, and unallocated cash. |
| PCM-SCR-01 | Payment-batch detail | `/payments/pcm-scr-01` | Show control totals, instructions, approvals, submission state, outcome, and cancellation eligibility. |
| PCM-SCR-02 | Payment-instruction detail | `/payments/pcm-scr-02` | Show obligation, beneficiary, attempts, settlement balances, cancellation, exception decision, and returns. |
| PCM-SCR-03 | Payment-return detail | `/payments/pcm-scr-03` | Show observation, reservation, posting, owner acknowledgement, exception resolution, reversal, and reconciliation. |
| PCM-SCR-04 | Incoming-settlement expectation | `/payments/pcm-scr-04` | Show expected, received, reconciled, remaining, expiry, receipt allocations, and resolutions. |
| PCM-SCR-05 | Settlement-receipt detail | `/payments/pcm-scr-05` | Show bank allocation, validation, cash posting, owner application, reconciliation, reversal, and evidence. |
| PCM-SCR-06 | Unallocated incoming cash | `/payments/pcm-scr-06` | Show suspense posting, candidate expectation, exception reason, resolution, and authoritative posting result. |
| PCM-SCR-07 | Bank-account record | `/payments/pcm-scr-07` | Maintain masked identity, currency, bank identifier, and account status under scoped authorization. |
| RPT-WS-01 | Reporting and consolidation worklist | `/reporting/rpt-ws-01` | Manage report definitions, consolidation runs, statement generation, review, and publication. |
| RPT-SCR-01 | Report-definition editor | `/reporting/rpt-scr-01` | Maintain statement structure, mappings, calculations, period, and presentation currency. |
| RPT-SCR-02 | Consolidation run | `/reporting/rpt-scr-02` | Show participant scopes, source watermarks, translation, eliminations, exceptions, approval, and rerun lineage. |
| RPT-SCR-03 | Financial statement review | `/reporting/rpt-scr-03` | Show definition version, source watermarks, statement lines, validation, publication state, and supersession. |
| RPT-SCR-04 | Consolidation exception workspace | `/reporting/rpt-scr-04` | Resolve missing data, ownership, translation, elimination, and approval blockers. |
| IC-WS-01 | Intercompany worklist | `/intercompany/ic-ws-01` | Manage agreements, transactions, matching, residuals, settlement, returns, and eliminations. |
| IC-SCR-01 | Intercompany agreement | `/intercompany/ic-scr-01` | Maintain participant scopes, currency, rate policy, tolerance, and effective period. |
| IC-SCR-02 | Intercompany transaction | `/intercompany/ic-scr-02` | Record reciprocal references, amount, currency, counterparty, and status. |
| IC-SCR-03 | Settlement-run workspace | `/intercompany/ic-scr-03` | Select items, reserve, match, net, approve residuals, create instructions, and complete. |
| IC-SCR-04 | Match and residual review | `/intercompany/ic-scr-04` | Compare reciprocal items, show differences, exceptions, and required approval. |
| IC-SCR-05 | Elimination run | `/intercompany/ic-scr-05` | Review versioned elimination instructions, exceptions, approvals, and reporting result. |
| REV-WS-01 | Revenue-accounting worklist | `/revenue-recognition/rev-ws-01` | Manage contract assessments, schedules, profiles, modifications, recognition, and exceptions. |
| REV-SCR-01 | Revenue-contract assessment | `/revenue-recognition/rev-scr-01` | Review source contract, collectibility, combination, promises, obligations, and transaction price. |
| REV-SCR-02 | Performance-obligation and allocation workspace | `/revenue-recognition/rev-scr-02` | Define obligations, allocation evidence, contract balances, and policy version. |
| REV-SCR-03 | Revenue-schedule review | `/revenue-recognition/rev-scr-03` | Review schedule versions, milestones, recognition dates, amounts, and approval state. |
| REV-SCR-04 | Revenue accounting profile | `/revenue-recognition/rev-scr-04` | Review immutable published classification used by AR and Invoicing. |
| REV-SCR-05 | Contract modification workspace | `/revenue-recognition/rev-scr-05` | Classify and approve modification, renewal, cancellation, refund, variable consideration, and catch-up. |
| REV-SCR-06 | Recognition run | `/revenue-recognition/rev-scr-06` | Preview scheduled recognition, validate period/rates, post, reconcile, and rerun where allowed. |
| FA-WS-01 | Asset register and worklist | `/fixed-assets/fa-ws-01` | Manage capitalization, depreciation, impairment, transfer, split, disposal, settlement, and correction. |
| FA-SCR-01 | Asset detail | `/fixed-assets/fa-scr-01` | Show cost, components, depreciation, impairment, carrying amount, location, status, and lineage. |
| FA-SCR-02 | Capitalization and acquisition clearing | `/fixed-assets/fa-scr-02` | Create asset cost and reconcile supplier-liability clearing without taking AP ownership. |
| FA-SCR-03 | Depreciation run | `/fixed-assets/fa-scr-03` | Review policy, period, asset calculations, exceptions, approval, and posting. |
| FA-SCR-04 | Impairment assessment | `/fixed-assets/fa-scr-04` | Record recoverable amount, impairment, evidence, approval, and posting result. |
| FA-SCR-05 | Transfer and split workspace | `/fixed-assets/fa-scr-05` | Select assets/components, effective date, destination/classification, allocation, and validation. |
| FA-SCR-06 | Asset-disposal workspace | `/fixed-assets/fa-scr-06` | Review carrying amount, treatment, proceeds, costs, required posting legs, approval, and settlement. |
| FA-SCR-07 | Disposal settlement and correction | `/fixed-assets/fa-scr-07` | Track proceeds, costs, returns/reversals, failures, replacement, compensation, and posted-disposal correction. |
| FX-WS-01 | Currency operations worklist | `/multi-currency/fx-ws-01` | Manage rate sets, revaluation, translation, approvals, postings, reruns, and reversals. |
| FX-SCR-01 | Rate-set publication | `/multi-currency/fx-scr-01` | Review provider, rate type/date, currency pairs, rates, version, and publication status. |
| FX-SCR-02 | Revaluation run | `/multi-currency/fx-scr-02` | Select scope/period/rates, preview exposures and gain/loss, approve, post, rerun, and reverse. |
| FX-SCR-03 | Translation run | `/multi-currency/fx-scr-03` | Select participant scopes, presentation currency, rate policy, watermarks, and publish translation result. |
| FX-SCR-04 | Conversion evidence panel | `/multi-currency/fx-scr-04` | Show transaction/functional amounts, rate set, rate type, date/time, and calculation lineage. |
| FPM-WS-01 | Period-control dashboard | `/fiscal-periods/fpm-ws-01` | Show period status, gate mode, active owner, close/reopen processes, tasks, exceptions, and cutoffs. |
| FPM-SCR-01 | Soft-close run | `/fiscal-periods/fpm-scr-01` | Start/end soft close, show policy, control epoch, admissions, exceptions, and handoff status. |
| FPM-SCR-02 | Hard-close/reclose run | `/fiscal-periods/fpm-scr-02` | Show checklist, barrier, close tasks, approvals, admissions, watermark, seal, and finalization. |
| FPM-SCR-03 | Reopen request | `/fiscal-periods/fpm-scr-03` | Capture reason, mode, scope, corrections, impact, approval, expiry, admissions, and completion path. |
| FPM-SCR-04 | Period-control recovery | `/fiscal-periods/fpm-scr-04` | Show authoritative gate status, takeover eligibility, control authority, cutoff, expiry, and recovery actions. |
| FPM-SCR-05 | Close exception review | `/fiscal-periods/fpm-scr-05` | Record exception, owner, due date, impact, approval/extension, resolution, and blocker status. |
| COA-WS-01 | Segment administration worklist | `/coa-segments/coa-ws-01` | Manage definitions, values, combinations, changes, approvals, and validation exceptions. |
| COA-SCR-01 | Segment definition | `/coa-segments/coa-scr-01` | Maintain segment type, code, name, status, and effective date range. |
| COA-SCR-02 | Segment values | `/coa-segments/coa-scr-02` | Maintain values, descriptions, statuses, and effective date ranges. |
| COA-SCR-03 | Combination validator | `/coa-segments/coa-scr-03` | Validate combinations and show invalid values, restrictions, and effective-date reasons. |
| COA-SCR-04 | Segment change request | `/coa-segments/coa-scr-04` | Review requested change, effective date, approval, impacted records, and applied decision. |
| BFR-WS-01 | Bank reconciliation worklist | `/bank-reconciliation/bfr-ws-01` | Manage connections, imports, statements, matching, unmatching, differences, and completion. |
| BFR-SCR-01 | Bank-feed connection | `/bank-reconciliation/bfr-scr-01` | Maintain provider, credential-reference status, consent, expiry, synchronization position, and connection status without exposing credential material. |
| BFR-SCR-02 | Statement import | `/bank-reconciliation/bfr-scr-02` | Review statement identity, period, balances, fingerprint, lines, duplicates, and validation. |
| BFR-SCR-03 | Matching workspace | `/bank-reconciliation/bfr-scr-03` | Compare bank lines with candidate business records; propose, confirm, reject, or split matches. |
| BFR-SCR-04 | Reconciliation session | `/bank-reconciliation/bfr-scr-04` | Show opening/closing balance, matched/unmatched totals, differences, exceptions, and completion evidence. |
| BFR-SCR-05 | Match detail and unmatch | `/bank-reconciliation/bfr-scr-05` | Show match evidence, rule version, user confirmation, downstream effects, and reversal eligibility. |
| TAX-WS-01 | Tax operations worklist | `/tax/tax-ws-01` | Manage configuration, determinations, returns, submissions, amendments, adjustments, payments, and evidence. |
| TAX-SCR-01 | Tax configuration | `/tax/tax-scr-01` | Maintain jurisdictions, rules, rates, categories, and effective-date ranges. |
| TAX-SCR-02 | Tax determination review | `/tax/tax-scr-02` | Review source facts, rule version, jurisdiction, calculation, exceptions, and finalization blocker. |
| TAX-SCR-03 | Tax return and submission | `/tax/tax-scr-03` | Prepare, approve, submit, reconcile authority outcome, and preserve immutable submitted versions. |
| TAX-SCR-04 | Tax amendment | `/tax/tax-scr-04` | Link accepted original return/version, reason, approval, submission, and accepted lineage. |
| TAX-SCR-05 | Return-level adjustment | `/tax/tax-scr-05` | Create, approve, post, retry, and reconcile tax adjustments independently of return/amendment records. |
| TAX-SCR-06 | Tax payment obligation | `/tax/tax-scr-06` | Request payment, track instruction and settlement, handle return/failure, and preserve filing status. |
| WFA-WS-01 | Approval inbox and worklist | `/approvals/wfa-ws-01` | Show assigned, delegated, escalated, due, blocked, and completed approvals. |
| WFA-SCR-01 | Approval request detail | `/approvals/wfa-scr-01` | Show subject snapshot, policy/version, steps, decisions, delegation, escalation, and application status. |
| WFA-SCR-02 | Approval decision workspace | `/approvals/wfa-scr-02` | Approve or reject with required evidence and show segregation and current-state revalidation. |
| WFA-SCR-03 | Approval policy editor | `/approvals/wfa-scr-03` | Maintain applicability, thresholds, steps, approvers, delegations, escalations, and effective dates. |
| WFA-SCR-04 | Delegation and escalation | `/approvals/wfa-scr-04` | Create/revoke delegation and escalate overdue or blocked decisions with audit evidence. |
| IAM-WS-01 | Access administration worklist | `/identity-access/iam-ws-01` | Manage users, roles, policies, segregation rules, emergency access, reviews, and exceptions. |
| IAM-SCR-01 | User access record | `/identity-access/iam-scr-01` | Show identity, authentication subject, roles, scopes, status, and access-review evidence. |
| IAM-SCR-02 | Role and access policy | `/identity-access/iam-scr-02` | Maintain permissions across scope and action dimensions. |
| IAM-SCR-03 | Segregation rule | `/identity-access/iam-scr-03` | Maintain prohibited combinations, thresholds, exceptions, and approval policy. |
| IAM-SCR-04 | Emergency-access request | `/identity-access/iam-scr-04` | Capture reason, scope, expiry, approval, permitted actions, activity, and revocation/review. |
| IAM-SCR-05 | Access decision explanation | `/identity-access/iam-scr-05` | Show allowed/denied result, applicable dimensions, conflicting duty, and next permitted action. |
| AUD-WS-01 | Audit-integrity worklist | `/audit-integrity/aud-ws-01` | Manage appended evidence, seals, proof verification, credential rotation, and incidents. |
| AUD-SCR-01 | Audit-chain scope | `/audit-integrity/aud-scr-01` | Show sequence, events, fingerprints, gaps, seals, legal hold, and source references. |
| AUD-SCR-02 | Seal detail | `/audit-integrity/aud-scr-02` | Show covered range, seal status, verification credential, supersession, and proof reference. |
| AUD-SCR-03 | Proof verification | `/audit-integrity/aud-scr-03` | Select scope/range/proof and show Valid, MissingEvent, ProofMismatch, InvalidProof, or UnsupportedVersion. |
| AUD-SCR-04 | Integrity incident | `/audit-integrity/aud-scr-04` | Show affected range, evidence, severity, owner, containment, recovery, corrective seals, and closure. |
| AUD-SCR-05 | Verification-credential rotation | `/audit-integrity/aud-scr-05` | Show current/previous credential intervals, reason, approvals, impact, and replacement-seal requirements. |

## 5. Route protection

1. Resolve Entra authentication before protected routes render.
2. Load actor capability grants and available accounting scopes.
3. Require explicit scope selection for scope-bound screens.
4. Hide unavailable actions for usability, but handle API 403 as authoritative.
5. On scope change, cancel in-flight requests and clear scope-bound query caches.

## 6. API client contract

```ts
export interface ApiProblem {
  type: string;
  title: string;
  status: number;
  code: string;
  detail: string;
  correlationId: string;
  currentVersion?: number;
  fieldErrors?: Array<{ field: string; code: string; message: string }>;
}
```

- Generated OpenAPI types are committed and checked for drift.
- HTTP wrapper adds correlation, scope, idempotency and `If-Match` headers.
- Decimal money remains a string in the client model.
- Retry is disabled by default for mutations; user-controlled safe retry follows typed API guidance.

## 7. Form rules

- Monetary entry uses locale-aware display but canonical decimal-string submission.
- Validation summary links to each field/line and receives focus on failed submission.
- Irreversible or high-risk actions require a typed confirmation that shows record, scope, amount and effect.
- A version conflict opens `CMP-012` with reviewed versus current state; it never automatically resubmits.
- Draft persistence is allowed only for DDD/PRD states that permit drafts.

## 8. Table rules

TanStack Table provides sorting, filtering, pagination, column visibility and selection. Server-side operations are used when result sets may exceed 500 rows. Bulk actions are shown only when every selected row is eligible; the API still validates each business identity.

## 9. Tailwind and daisyUI tokens

```css
@import "tailwindcss";
@plugin "daisyui" {
  themes: finance-light --default, finance-dark --prefersdark;
}
```

Semantic tokens define `success`, `warning`, `error`, `info`, `pending`, `reconciled`, `restricted` and `disabled`. State is never conveyed by color alone.

## 10. Accessibility

- WCAG 2.2 AA is the release target.
- All critical workflows are keyboard complete.
- Dialogs trap and restore focus; async changes use appropriate live regions.
- Table headers, row actions, validation and status badges have accessible names.
- Automated axe checks run in component and Playwright suites; critical workflows receive manual screen-reader review.

## 11. Frontend test baseline

| Layer | Tool | Required coverage |
|---|---|---|
| Utility/schema | Vitest | Decimal/date formatting, Zod schemas, permission display helpers |
| Component | Vitest + Testing Library | Semantics, keyboard, validation, states, sensitive masking |
| API simulation | MSW | Success, domain rejection, 403, 409, 503, ambiguous outcome |
| Workflow | Playwright | All 22 `WF-*` journeys, critical exception/recovery paths |
| Accessibility | axe + manual | Shared components and critical journeys |
| Visual | Playwright screenshots | Stable shared shells/components, not pixel-perfect business data |

## 12. Security and privacy

- Tokens are kept in memory through the authentication library; no access token in local storage.
- Rendering untrusted text uses React escaping; approved rich content is sanitized.
- Sensitive values are masked by default and excluded from client logs, analytics and error telemetry.
- Export/download URLs are short-lived and never persisted in browser storage.

## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `18bb5e611fb5498f60994a4fd68ac3fdcc80058175deda5f8722e3a68d4fc726` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; run the full suite for API, database, event, security, deployment, recovery, or technology-baseline changes. |
