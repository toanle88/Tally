# DLV-UX-002 User Story 5 — Accessibility qualification evidence and defect decisions

Status: implemented on branch `feat/dlv-ux-002-us5-qualification-evidence`.

This record defines the evidence and defect-decision boundary for the existing
accessibility harness. It qualifies implemented shared UI and synthetic fixture
coverage only. It does not claim that future finance capabilities, production
exports, or all 22 critical workflows are accessible or release-qualified.

## Qualification runner

The root command supports a complete run and explicit focused reruns:

```text
pnpm test:a11y:qualification
pnpm test:a11y:qualification -- --target <target>
pnpm test:a11y:qualification -- --list-targets
```

`all` is the default. Focused selection is explicit: use `all` when a shared
component or interaction change can affect more than one accessibility mode.
The runner does not infer target ownership from arbitrary Git diffs.

| Target | Playwright selection or command | Affected scope | Affected evidence |
|---|---|---|---|
| `shell` | `axe: routed application shell` | Routed application shell at `/` | [US1](./DLV-UX-002-us1-automated-accessibility-harness.md), this record |
| `examples` | `axe: synthetic integrated examples` | Synthetic integrated examples at `/development/examples` | [US1](./DLV-UX-002-us1-automated-accessibility-harness.md), this record |
| `keyboard` | `keyboard: shared synthetic examples` | Keyboard and focus behavior for `/development/examples` | [US2](./DLV-UX-002-us2-keyboard-focus-behavior.md), this record |
| `semantic` | `semantic and screen-reader review proxy coverage` | Semantic and screen-reader proxy coverage for `/development/examples` | [US3](./DLV-UX-002-us3-screen-reader-semantic-review.md), this record |
| `visual` | `visual adaptability and motion preferences` | Visual adaptability and motion preferences for `/development/examples` | [US4](./DLV-UX-002-us4-visual-adaptability-motion.md), this record |
| `negative` | `pnpm test:a11y:negative` | Controlled unlabeled-button violation sentinel | [US1](./DLV-UX-002-us1-automated-accessibility-harness.md), this record |
| `all` | Full `test:a11y`, followed by the negative wrapper | All implemented accessibility checks | `docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md`; `docs/verification/DLV-UX-002-us2-keyboard-focus-behavior.md`; `docs/verification/DLV-UX-002-us3-screen-reader-semantic-review.md`; `docs/verification/DLV-UX-002-us4-visual-adaptability-motion.md`; this record |

The runner prints the selected target, affected scope, exact rerun command,
and evidence paths. Invalid targets, no-match selections, and failed checks
return non-zero. The negative target is expected to observe the controlled
`button-name` failure and then return success through its wrapper.

## Result vocabulary

Each evidence row has exactly one result. Only `Pass` may count as a passing
check, and a scoped Pass never implies release qualification for an untested
future workflow.

| Result | Meaning |
|---|---|
| `Pass` | The named target and method passed in the recorded environment. |
| `Fail` | The named target ran and did not satisfy its expected result. |
| `Blocked` | The check could not complete because of a documented environment, access, or dependency blocker. |
| `Not Applicable` | The behavior does not exist in the named scope, with rationale recorded. |
| `Not Yet Implemented` | The target or capability is not present in this delivery scope. |
| `Not Run` | An applicable check exists but has not yet been executed. |

## Verification report template

Copy one row for every check or witnessed review:

| Check | Target | Requirement or UX criterion | Verification method | Environment | Data profile | Owner | Result | Evidence reference | Scope/notes |
|---|---|---|---|---|---|---|---|---|---|
|  |  |  |  |  | Synthetic only / other approved profile | Frontend owner or QA owner | Pass / Fail / Blocked / Not Applicable / Not Yet Implemented / Not Run |  |  |

For every `Fail` or `Blocked` finding, add the defect record below before any
release or milestone claim.

| Finding | Severity | Impact | Affected workflow/class | Owner | Corrective action | Due date or exception expiry | Compensating control | Retest command/result | Closure decision |
|---|---|---|---|---|---|---|---|---|---|
|  | Critical / High / Medium / Low |  |  | Frontend owner / QA owner |  |  |  |  | Open / Closed / Approved exception |

## Synthetic baseline report

The following results are scoped to the routed shell and synthetic integrated
examples. They are not production or full critical-workflow qualification.

| Check | Target | Requirement mapping | Method/environment | Owner | Result | Evidence |
|---|---|---|---|---|---|---|
| Routed shell axe scan | `shell` | NFR-ACC-001, NFR-ACC-003, NFR-ACC-004 | `pnpm test:a11y:qualification -- --target shell`; Playwright Chromium | Frontend owner | Pass | [US1](./DLV-UX-002-us1-automated-accessibility-harness.md) |
| Synthetic axe states and dialogs | `examples` | NFR-ACC-001, NFR-ACC-003, NFR-ACC-004, NFR-ACC-006 | `pnpm test:a11y:qualification -- --target examples`; Playwright Chromium | Frontend owner | Pass | [US1](./DLV-UX-002-us1-automated-accessibility-harness.md) |
| Keyboard and focus behavior | `keyboard` | NFR-ACC-002, NFR-ACC-006, NFR-ACC-007; UXA-COMMON-010 | Focused qualification runner; Playwright Chromium | Frontend owner | Pass | [US2](./DLV-UX-002-us2-keyboard-focus-behavior.md) |
| Semantic and screen-reader proxy checks | `semantic` | NFR-ACC-003, NFR-ACC-004, NFR-ACC-006, NFR-ACC-007 | Focused qualification runner; Playwright Chromium proxy | Frontend owner | Pass | [US3](./DLV-UX-002-us3-screen-reader-semantic-review.md) |
| Contrast, 200% text, and narrow reflow proxy | `visual` | NFR-ACC-005 | Focused qualification runner; Playwright Chromium | Frontend owner | Pass | [US4](./DLV-UX-002-us4-visual-adaptability-motion.md) |
| Reduced-motion behavior | `visual` | NFR-ACC-009 | Focused qualification runner; Playwright Chromium reduced-motion emulation | Frontend owner | Pass | [US4](./DLV-UX-002-us4-visual-adaptability-motion.md) |
| Controlled axe negative proof | `negative` | NFR-ACC-011; NFR-TST-007 | `pnpm test:a11y:negative`; isolated unlabeled-button fixture | QA owner | Pass | [US1](./DLV-UX-002-us1-automated-accessibility-harness.md) |
| NVDA / Firefox / Windows 11 review | SR-01 | NFR-ACC-003, NFR-ACC-012; NFR-TST-007 | Manual assistive-technology review | QA owner / accessibility reviewer | Not Run | [US3](./DLV-UX-002-us3-screen-reader-semantic-review.md) |
| VoiceOver / Safari / macOS review | SR-02 | NFR-ACC-003, NFR-ACC-012; NFR-TST-007 | Manual assistive-technology review | QA owner / accessibility reviewer | Not Run | [US3](./DLV-UX-002-us3-screen-reader-semantic-review.md) |
| Actual 400% browser zoom | `visual` | NFR-ACC-005; NFR-TST-007 | Manual browser-zoom review | QA owner / accessibility reviewer | Not Run | [US4](./DLV-UX-002-us4-visual-adaptability-motion.md) |
| Evidence export and printable-report accessibility | Future export surfaces | NFR-ACC-010 | Human-readable export review | Owning capability / QA owner | Not Yet Implemented | Future release evidence |
| All 22 critical workflows | `WF-6.1`–`WF-7.15` | NFR-ACC-001, NFR-TST-002, NFR-TST-007 | Full workflow qualification | Owning capability / QA owner | Not Yet Implemented | Future release evidence |

### Requirement-level traceability decisions

The scoped result and the requirement/release state are intentionally separate.

| Requirement | Scoped evidence result | Requirement/release qualification state | Basis |
|---|---|---|---|
| NFR-ACC-001 | Pass for shared shell/examples | Not Yet Implemented | All 22 critical workflows are not implemented. |
| NFR-ACC-002 | Pass for synthetic keyboard coverage | Not Yet Implemented | Future workflow interactions remain untested. |
| NFR-ACC-003 | Pass for semantic proxy; SR-01/SR-02 Not Run | Not Run | The approved screen-reader matrix is pending. |
| NFR-ACC-004 | Pass for synthetic state and text meaning | Not Yet Implemented | Future capability state coverage remains pending. |
| NFR-ACC-005 | Pass for automated contrast, 200% text, and narrow reflow proxy | Not Run | Actual 400% browser zoom remains pending. |
| NFR-ACC-006 | Pass for synthetic validation flows | Not Yet Implemented | Future validation paths remain pending. |
| NFR-ACC-007 | Pass for synthetic status/progress behavior | Not Yet Implemented | Future workflow outcomes remain pending. |
| NFR-ACC-008 | Not Applicable | Not Applicable | No time-limit behavior exists in the current synthetic target; re-evaluate with time-limited workflows. |
| NFR-ACC-009 | Pass for synthetic reduced-motion coverage | Not Yet Implemented | Future capability motion remains pending. |
| NFR-ACC-010 | Not Yet Implemented | Not Yet Implemented | No export or printable-report surface exists in this scope. |
| NFR-ACC-011 | Pass for the documented process and sample retest | Pass for process only | This does not approve a production release. |
| NFR-ACC-012 | Not Yet Implemented | Not Yet Implemented | No high-risk workflow specialist review exists in this scope. |
| NFR-TST-003 | Pass for the template’s evidence fields | Not Yet Implemented | This story does not qualify every NFR in the document. |
| NFR-TST-007 | Pass for implemented automated, keyboard, proxy, and visual layers | Not Yet Implemented | Full manual and all-22 workflow qualification remains pending. |
| UXA-COMMON-010 | Pass for implemented shared interactions | Not Yet Implemented | Future workflows remain outside this delivery scope. |

## Defect decisions and retest example

`SR-SAMPLE-001` is an illustrative closed trace, not a live baseline finding:

| Field | Decision |
|---|---|
| Finding / initial result | `SR-SAMPLE-001`; Fail before remediation because table row identity was not announced. |
| Severity | Medium; the synthetic shared-table proxy had a workaround and did not block a Class A workflow. |
| Impact / affected class | Assistive-technology users could not identify table rows; shared table proxy, not a future Class A workflow. |
| Owner | Frontend owner, with QA owner recording the disposition. |
| Corrective action | Add row-header metadata and `<th scope="row">`. |
| Due date / exception expiry | Due before baseline acceptance; no exception. |
| Compensating control | None required for the illustrative synthetic example. |
| Retest | `pnpm test:a11y:qualification -- --target semantic` and `pnpm -C web test`; both must pass. |
| Retest result / closure | Pass after remediation; Closed. No live defect remains. |

For an actual Class A blocker, the decision is Critical and release-blocking
under NFR-ACC-011. Other WCAG AA failures must have a recorded severity,
owner, corrective action, and approved remediation or time-bound exception
before release claims. High findings block the affected milestone; Medium and
Low findings may follow the delivery-plan deferral policy only with their
required disposition.

## Deferred and follow-on qualification

This User Story does not mark the following work complete:

- witnessed NVDA/Firefox/Windows and VoiceOver/Safari/macOS reviews;
- actual 400% browser-zoom review;
- accessibility of exports and printable reports;
- time-limit, authorization, provider-window, and other future workflow paths;
- specialist review for high-risk workflows; or
- full review of all 22 critical workflows `WF-6.1` through `WF-7.15`.

Those checks remain release/NFR qualification obligations when the applicable
capabilities and workflows are implemented. `DLV-UX-002` remains open until
its other user stories and delivery-item evidence are complete.

## Verification record

Verified locally on 2026-09-02 from base revision `9fbe14d` with the working
tree changes on `feat/dlv-ux-002-us5-qualification-evidence`.

Environment: pnpm 11.9.0, Playwright 1.62.1, Chromium/Chrome for Testing
151.0.7922.34, synthetic fixture data only.

- `pnpm test:a11y:qualification -- --list-targets` — passed;
- invalid target check — exited with code 2 as expected;
- focused `shell` target — passed: 1 test;
- focused `examples` target — passed: 4 tests;
- focused `keyboard` target — passed: 6 tests;
- focused `semantic` target — passed: 3 tests;
- focused `visual` target — passed: 4 tests;
- focused `negative` target — expected axe failure observed and wrapper passed;
- focused `all` target — passed: 18 positive tests and the expected negative proof;
- `pnpm test` — passed: Go tests and 47 frontend tests in 9 files;
- `pnpm build` — passed: API and frontend builds (the restricted sandbox emitted
  a non-fatal Go module stat-cache warning);
- `pnpm docs:check` — passed; and
- `git diff --check` — passed after the final documentation update.

Documentation-only status is never used as evidence of a passing check.
