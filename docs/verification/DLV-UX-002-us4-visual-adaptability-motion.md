# DLV-UX-002 User Story 4 — Visual adaptability and motion preferences

Status: automated baseline verified 2026-09-02. User Story 4 remains open until manual browser-zoom confirmation is recorded.

## Scope and evidence

The visual-adaptability suite is [`web/tests/a11y/visual-adaptability.spec.ts`](../../web/tests/a11y/visual-adaptability.spec.ts). It uses synthetic fixture data only; no screenshots, secrets, or sensitive finance values are produced or committed.

The implementation provides:

- theme-specific contrast assertions for `finance-light` and `finance-dark`, including status, disabled, and restricted states;
- reflow assertions for the routed development examples at 200% text size and a 320px-wide 400%-browser-zoom-equivalent viewport;
- document overflow, visible-control containment, semantic table headers, and keyboard-focusable, caption-labelled table scroll regions;
- a real CSS animation probe verifying that `prefers-reduced-motion: reduce` shortens nonessential animation and disables smooth scrolling while preserving process/status content.

## Repeatable verification matrix

| Target | Viewport | Text/zoom setting | Motion | Browser | Result | Finding |
|---|---:|---|---|---|---|---|
| Light and dark theme contrast | 1280×720 | 100% | no preference | Playwright Chromium 1.62.1 / Chrome for Testing 151.0.7922.34 | PASS | None |
| Dense tables, forms, dialogs, status/process surfaces | 1280×720 | 200% root text (`32px`) | no preference | Playwright Chromium 1.62.1 / Chrome for Testing 151.0.7922.34 | PASS | None |
| Same representative surfaces | 320×900 | 400% browser-zoom equivalent; narrow viewport proxy, not an OS/browser zoom claim | no preference | Playwright Chromium 1.62.1 / Chrome for Testing 151.0.7922.34 | PASS | None |
| Process/status meaning and motion probe | 1280×720 | 100% | `prefers-reduced-motion: reduce` | Playwright Chromium 1.62.1 / Chrome for Testing 151.0.7922.34 | PASS | None |

The automated checks reject page-level horizontal overflow, clipped visible interactive controls, unlabelled table overflow, and missing table row/header semantics. Intentional table overflow is contained in a labelled, focusable region that preserves the table caption and headers.

## Manual release checklist

Run the same representative routes in a supported desktop browser with actual browser zoom at 400% and text-only enlargement at 200%, then repeat with the operating-system/browser reduced-motion preference enabled. Record viewport, browser/version, setting, target, result, and any finding reference here or in the release evidence. Before capturing any evidence, confirm that the screen contains only synthetic fixture values.

## Commands

```text
pnpm -C web test
pnpm -C web run build
pnpm test:a11y
pnpm test:a11y:negative
pnpm docs:check
```
