# UX foundation

The UX is designed around authorized finance work rather than generic CRUD.
Every record detail should make scope, ownership, state, permitted actions,
lineage, evidence, and recovery visible. The current repository implements the
shared shell and synthetic workflow surfaces that make those semantics testable
before the finance bounded contexts arrive.

## Information architecture

The approved navigation areas are Home, Work, Records, Approvals, Exceptions,
Reports, Administration, and Audit. A persistent accounting/business scope
context shows the tenant, legal entity, ledger, book, functional currency, and
period when applicable. Changing scope refreshes data before an action can be
submitted.

## Shared interaction model

The component layer covers:

- accounting scope selection and record identity headers;
- state-aware action bars, lifecycle timelines, approval/posting/settlement
  panels, and correction lineage;
- validation summaries, exception resolution, result lookup, and safe
  concurrency-conflict dialogs;
- evidence drawers, worklists/saved filters, process progress, legal-hold
  indicators, money/currency evidence, and sensitive-data guards.

The interaction rules are intentionally explicit:

- authorization and scope are rechecked before submission;
- established facts expose correction actions instead of destructive editing;
- duplicates open the established result, while changed-content reuse is a
  conflict;
- stale versions show expected/current state and whether retry is safe;
- dependency-unavailable, authorization-denied, validation-rejected,
  pending, reconciled, reversed, and terminal states are distinct;
- sensitive fields remain masked unless both authorization and auditable access
  evidence are present.

## Current screens

The SPA currently demonstrates:

- the shared route and scope shell;
- IAM-WS-01 user/access administration and IAM-SCR-01 through IAM-SCR-05
  states, including roles, segregation, emergency access, decision
  explanation, masking, export denial, and version conflicts;
- cross-context event exception and concurrency-conflict operational examples;
- reusable record, workflow, operational, and development examples.

The broad Home, Work, Records, Approvals, Exceptions, Reports, Administration,
and Audit routes are navigation scaffolds unless explicitly backed by one of
those examples. No placeholder route is an authoritative finance workspace.

## Accessibility and privacy

The component tests and Playwright harness cover semantic labeling, axe rules,
keyboard operation, focus behavior, live status announcements, validation
association, visual reflow, reduced motion, and sensitive-data presentation.
The UI does not rely on color alone, and masked values are not revealed by
client-side filtering or export controls.

The current M0 evidence remains narrower than the full NFR qualification:
witnessed screen-reader review, actual 400% browser-zoom review, complete
critical finance workflows, supported-browser qualification, and production
data/privacy review remain open.

See the [UX/workflow specification](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_ux_workflow_specification_v1.0.md),
[frontend technical specification](https://github.com/toanle88/Tally/blob/main/docs/specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md),
and [UX/accessibility verification records](https://github.com/toanle88/Tally/tree/main/docs/verification).
