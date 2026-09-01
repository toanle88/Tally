# DLV-UX-002 User Story 2 — Keyboard operation and focus behavior

Status: implemented on branch `feat/dlv-ux-002-us2-keyboard-focus-behavior`.

## Scope

The accessibility harness now verifies keyboard-only operation and focus
behavior for the synthetic integrated examples. The checks do not claim that
future finance capabilities or all 22 critical workflows are implemented or
qualified.

Covered synthetic interactions include:

- global navigation, accounting-scope selection, and unsaved-work confirmation;
- form tab order, failed submission, validation-summary navigation, and draft preservation;
- worklist filtering, sorting, selection, bulk action, pagination, and row links;
- approval and posting presentation, blocked-action explanations, and status updates; and
- confirmation cancellation, conflict-dialog focus trapping, Escape, safe exit, and focus restoration.

## Focus behavior

- Dynamic confirmation surfaces opt into initial focus on their safe cancel action and restore the triggering control when dismissed.
- Conflict dialogs focus their close action on mount, trap keyboard focus, close on Escape, restore the opener, and do not refocus when an asynchronous status update rerenders the dialog.
- Validation failures focus the validation summary, preserve valid form values, and let summary links move focus to the affected field.
- Blocked actions expose a visible reason and next action; unavailable controls are not the only explanation of the blocked state.
- Shared and raw table controls expose visible keyboard focus indicators.

## Reusable keyboard procedure

Use this checklist for future critical workflows. Record the exact browser,
viewport, target, observed result, and evidence reference for each run.

| Target | Keyboard steps | Expected result | Observed result | Result / evidence |
|---|---|---|---|---|
| Navigation and scope | Tab through navigation; activate a route; move through the scope selector with Arrow keys | Logical order, visible focus, correct route/scope, no pointer required |  |  |
| Worklist and table | Tab to filters, change a filter, sort a header, select an eligible row, activate pagination and a row link | Controls retain names, selection/sorting/pagination work, row identity remains clear |  |  |
| Form and validation | Tab through fields; submit invalid input; follow the first summary link | Summary receives focus, valid input remains, linked field receives focus |  |  |
| Confirmation surface | Open with Enter; cancel with Enter; reopen and confirm | Safe cancel receives focus and dismissal restores the trigger |  |  |
| Approval/posting and blocked action | Tab to available retry/action controls; inspect blocked controls and surrounding reason | Available action completes by keyboard; blocked reason and next action are discoverable |  |  |
| Conflict dialog | Open with Enter; use Tab and Shift+Tab at both boundaries; refresh; press Escape or activate Close | Focus remains trapped, async refresh does not move focus, safe exit restores the opener |  |  |

This procedure is a repeatable keyboard baseline for shared UI only. Screen
reader, zoom/reflow, contrast, reduced-motion, and full critical-workflow
qualification remain owned by the later DLV-UX-002 stories and release/NFR
qualification.

## Verification record

Verified locally on 2026-09-01:

- `pnpm test:a11y` — passed: 11 tests;
- `pnpm test:a11y:negative` — expected failure observed and wrapper passed;
- `pnpm -C web test` — passed: 47 tests in 9 files;
- `pnpm -C web run build` — passed; and
- `git diff --check` — passed.

No generated dependencies, secrets, finance state, API endpoints, persistence,
authentication, or authorization behavior were added.
