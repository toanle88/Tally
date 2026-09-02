# DLV-UX-002 User Story 1 — Automated accessibility harness

Status: implemented on branch `feat/dlv-ux-002-us1-automated-accessibility-harness`.

## Scope

The frontend now has a deterministic, non-watch Playwright Test entry point
using `@axe-core/playwright`. It runs one Chromium project with trace,
screenshot, and video output disabled. The suite uses only synthetic fixture
data and does not log credentials, tokens, restricted values, or sensitive
fixture data.

The covered targets are:

- `/` — routed application shell and landmark structure;
- `/development/examples` — shared regions, headings, labels, accessible
  names, status meaning, tables, and live-region markup;
- the confirmation state — material-action confirmation semantics;
- the validation-error state — field associations, invalid state, and error
  output; and
- the conflict-dialog state — modal semantics, accessible name, and safe exit.

## Commands

From the repository root:

```bash
pnpm --dir web exec playwright install chromium
pnpm test:a11y
pnpm test:a11y:negative
```

`pnpm test:a11y` type-checks the Playwright configuration/tests and runs the
five Chromium scenarios. `pnpm test:a11y:negative` runs an isolated fixture
with an unlabeled button and succeeds only when the Playwright command fails
with the `button-name` axe rule and the `A11Y_NEGATIVE_SENTINEL` marker.

## Verification record

Verified locally on 2026-09-01:

- `pnpm test:a11y` — passed: 5 tests;
- `pnpm test:a11y:negative` — expected failure observed and wrapper passed;
- From `web/`, `./node_modules/.bin/vitest run` — passed;
- From `web/`, `./node_modules/.bin/tsc -b` — passed;
- From `web/`, `./node_modules/.bin/vite build` — passed; and
- `git diff --check` — passed.

The approved dependency boundary is limited to the frontend lockfile:
`@playwright/test` 1.62.1 and `@axe-core/playwright` 4.13.0. No axe rules are
globally disabled or reclassified.

## Deferred scope

Screen-reader review and visual adaptability manual qualification remain the
separately tracked DLV-UX-002 User Stories 3–4. Qualification evidence and
defect decisions are recorded in
[`DLV-UX-002-us5-accessibility-qualification-evidence.md`](./DLV-UX-002-us5-accessibility-qualification-evidence.md).
