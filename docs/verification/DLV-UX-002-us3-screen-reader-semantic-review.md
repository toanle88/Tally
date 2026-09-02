# DLV-UX-002 User Story 3 — Screen-reader and semantic review coverage

Status: repository coverage implemented; witnessed assistive-technology review is pending.

This document defines repeatable review coverage for the shared shell and
synthetic integrated examples. It does not claim that a real NVDA or VoiceOver
session has passed, and it does not qualify future finance workflows that are
not implemented.

## Approved review matrix

| Matrix ID | Assistive technology | Browser | Operating system | Version fields | Result |
| --- | --- | --- | --- | --- | --- |
| SR-01 | NVDA | Firefox | Windows 11 | Record exact NVDA, Firefox, and Windows versions at execution | Not Run |
| SR-02 | VoiceOver | Safari | macOS | Record exact VoiceOver, Safari, and macOS versions at execution | Not Run |

The matrix is approved for manual review. A reviewer must replace the version
placeholder with observed versions and record the execution date before marking
either row Pass, Fail, or Blocked.

## Review procedure

1. Start the application with the repository instructions and open the
   development examples route. Record the matrix ID, exact versions, date,
   viewport, target, and reviewer.
2. Navigate by landmarks and headings. Confirm that navigation, main content,
   titled regions, and heading hierarchy identify the current context.
3. Traverse controls and fields. Confirm every label, description, required
   state, validation error, and error-to-control association is announced.
4. Review tables. Confirm the caption, column headers, explicit row identity,
   selection control names and states, sorting state, pagination status, and
   empty state are understandable without visual styling.
5. Review representative state and outcome text: pending, approval,
   reconciliation, exception, process progress, gain/loss, and debit/credit
   meaning. Confirm severity and state are not conveyed by color alone.
6. Open the conflict dialog and confirm its action, record/context, expected
   and current values, safe exit, and available actions are announced.
7. Review restricted data. Confirm masked values remain masked and protected
   content does not appear in accessible names, errors, announcements, or
   empty states. Authorized reveal behavior must be tested separately and only
   with synthetic data.
8. Record each result as exactly one of `Pass`, `Fail`, `Blocked`, `Not Run`,
   or `Not Yet Implemented`. A future capability or unexecuted journey is
   `Not Yet Implemented` or `Not Run`, never Pass.

## Coverage and evidence record

| Target | Repository proxy | Manual result | Evidence |
| --- | --- | --- | --- |
| Shared shell, headings, landmarks, labels, descriptions | `web/tests/a11y/semantic.spec.ts` | Not Run | SR-01 / SR-02 pending |
| Tables, headers, row identity, selection, sorting, pagination, empty state | `web/tests/a11y/semantic.spec.ts`; `web/src/components/ui/ui.test.tsx` | Not Run | Semantic proxy test output |
| State, approval, reconciliation, exception, gain/loss, debit/credit text | `web/tests/a11y/semantic.spec.ts` | Not Run | Synthetic development examples |
| Validation, dialog, progress, and dynamic announcements | `web/tests/a11y/semantic.spec.ts` | Not Run | Synthetic workflow examples |
| Restricted and masked values | `web/tests/a11y/semantic.spec.ts` | Not Run | Synthetic restricted reference |
| Future finance capability journeys | No route or proxy claims execution | Not Yet Implemented | Excluded from this story |

### Sample finding and remediation trace

| Field | Synthetic sample |
| --- | --- |
| Finding ID / severity | SR-SAMPLE-001 / Moderate |
| AT, browser, OS | NVDA / Firefox / Windows 11 — sample only, not witnessed |
| Target and steps | Table row identity; navigate the `Foundation surfaces` table by rows |
| Expected / observed | Expected each row announces its identity from a row header; observed before remediation: data cells only |
| Owner | `EP-UX-001 / Learning Product and Delivery Owner` |
| Evidence | `web/src/components/ui/data-table.tsx`, semantic proxy test |
| Remediation / retest | Add `rowHeader` metadata and `<th scope="row">`; rerun component and Playwright checks |

This sample uses no real financial record, credential, or protected value.
