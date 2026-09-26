# Roadmap and progress

The delivery plan has ten milestones, M0 through M9. Progress is evidence,
not a percentage: a milestone needs its required behavior, controls,
qualification gates, and documentation, or it remains open. The roadmap also
distinguishes a local learning baseline from production qualification.

## Current checkpoint

This checkout is still at the M0/M1 boundary:

- **EP-PLAT-001 — Engineering foundation:** locally complete and intentionally
  closed. Hosted CI and external qualification remain deferred.
- **EP-OPS-001 — Observability and operational foundation:** locally complete
  and intentionally closed. Production telemetry, paging, retention, and
  environment-dependent checks remain deferred.
- **EP-IAC-001 — Terraform and Azure learning environment:** closed by owner
  decision; authenticated Azure exercises and external evidence are deferred.
- **EP-UX-001 — Shared UX and design system:** shared shell, components, and
  accessibility harness are implemented; manual qualification and future
  workflow coverage remain open.
- **EP-IAM-001 — Identity and access:** locally complete and intentionally
  closed by owner decision on 2026-09-26. Finance-action wiring, live Entra
  qualification, audit-chain ownership, and production security qualification
  remain deferred and are not claimed.

The repository’s [live roadmap](https://github.com/toanle88/Tally/blob/main/ROADMAP.md)
is the status authority. Its last recorded update is 2026-09-22; this site
also describes the current checkout’s IAM work as of 2026-09-26.

## Milestones

| Milestone | Outcome | Current meaning |
| --- | --- | --- |
| M0 | Engineering foundation | Platform, UX foundation, local environment, CI, and operational controls. |
| M1 | Identity and accounting configuration | IAM, scope, master data, COA, and ledger configuration. IAM is the active slice; the accounting modules remain future work. |
| M2 | First posted journal vertical slice | Journal validation, approval, posting, query, and reversal end to end. Recommended portfolio stopping point. |
| M3 | Approval and period controls | Approval policies, close, reopen, reclose, and posting-gate recovery. |
| M4 | Receivables and billing | Invoicing, receipts, applications, credits, write-offs, and refund obligations. |
| M5 | Payables and payment execution | Vendor liabilities, payment instructions, settlement, returns, and exceptions. |
| M6 | Bank and cash reconciliation | Statement import, matching, unmatching, and reconciliation. |
| M7 | Assets and revenue | Fixed-asset lifecycle and revenue recognition. |
| M8 | Currency, intercompany, and reporting | FX, revaluation, translation, consolidation, and statements. |
| M9 | Tax, payroll, audit, and qualification | Remaining domain capabilities and full security, accessibility, recovery, and performance qualification. |

## Recommended stopping points

| Stop point | Learning outcome |
| --- | --- |
| M2 | A strong portfolio-sized finance vertical slice with exact money, posting, reversal, persistence, API, UI, authorization, tests, and operations. |
| M4 | Subledger-to-ledger integration and multi-aggregate consistency. |
| M6 | External evidence, payment execution, returns, and reconciliation. |
| M8 | Period-end, currency, intercompany, consolidation, and reporting architecture. |
| M9 | The full declared domain and qualification evidence. |

See the [delivery plan](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_delivery_plan_v1.0.md)
for scope, dependencies, exit evidence, and reforecast rules.
