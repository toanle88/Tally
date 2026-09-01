# DLV-UX-002 — Accessibility Test Harness User Stories

## 1. Outcome

Provide a repeatable accessibility verification boundary for the TALLY frontend. The harness shall combine automated checks, keyboard interaction checks, screen-reader procedures, zoom/reflow and reduced-motion checks, and documented evidence without claiming that future finance capabilities are already accessible or implemented.

## 2. Learning objective

Learn how to turn accessibility requirements into executable frontend checks and witnessed review evidence while keeping component ownership, workflow ownership, and release qualification boundaries explicit.

## 3. Scope and ownership

`DLV-UX-002` is owned by `EP-UX-001` and contributes to M0 quality gate `QG-05`.

In scope:

- An automated accessibility test entry point for shared components and the synthetic integrated example.
- Playwright accessibility checks that can be reused by future capability workflows.
- Keyboard-only and focus-management verification helpers and procedures.
- A repeatable manual screen-reader and browser-matrix procedure.
- Zoom, reflow, contrast, reduced-motion, status-announcement, and error-association checks.
- Accessibility evidence, defect severity, remediation ownership, and traceability conventions.

Out of scope:

- Implementing finance capabilities, API endpoints, persistence, authentication, authorization, or domain behavior.
- Replacing the component tests delivered by `DLV-UX-001`.
- Claiming that all 22 critical workflows pass before those workflows exist.
- Full-system accessibility qualification owned by the applicable release and NFR gates.
- Introducing a new frontend test framework when the approved Vitest, Testing Library, Playwright, and axe approach is sufficient.

## 4. Source requirements and boundaries

The stories are derived from:

- UX specification §10.1 and `UXA-COMMON-010`.
- NFR-ACC-001 through NFR-ACC-012.
- NFR-TST-002, NFR-TST-003, and NFR-TST-007.
- Frontend technical specification §§10–11.
- Testing, performance, and recovery specification §8 and release condition §10.

The harness verifies presentation and interaction semantics. It does not become authoritative for finance state, authorization decisions, or workflow outcomes. A failing check identifies evidence and a remediation owner; it does not silently change business behavior.

## 5. Shared test vocabulary

The harness uses the existing synthetic examples and shared component vocabulary:

- **Target:** a shared component, route, representative screen, or workflow journey under test.
- **Interaction mode:** automated axe, keyboard-only, screen reader, zoom/reflow, reduced motion, or manual review.
- **Finding:** rule/check identifier, target, observed result, severity, reproducibility, and owner.
- **Evidence:** command or procedure, environment, browser/assistive-technology combination, timestamp, result, and linked finding.
- **Workflow class:** representative shared interaction or future Class A critical workflow. The harness must not imply future workflow completion.

## 6. User Story 1 — Establish the automated accessibility harness

**As the TALLY developer, I want a deterministic automated accessibility check entry point, so that shared UI regressions are detected consistently.**

### Acceptance criteria

- [x] The frontend exposes a documented command for automated accessibility checks that runs in non-watch mode and returns a non-zero result for an accessibility violation.
- [x] The automated suite uses the approved frontend test/tooling boundary and scans the shared shell, representative shared components, and the synthetic integrated example with synthetic data.
- [x] The suite checks semantic regions, headings, labels, accessible names, status meaning, table structure, validation associations, dialog semantics, and live-region markup where applicable.
- [x] The suite does not suppress, broadly disable, or reclassify violations merely to make the baseline pass; any justified rule configuration is narrow, documented, and reviewed.
- [x] Test output identifies the target and failing rule sufficiently for a developer to reproduce the finding.
- [x] The harness does not log credentials, tokens, restricted values, or sensitive fixture data.

### Evidence

- Automated test configuration and documented command.
- Passing baseline output for shared targets.
- Controlled negative check proving a known violation fails the command.
- Rule-configuration review showing no unexplained global exclusions.

### User Story 1 implementation evidence

The implemented harness, target mapping, dependency boundary, passing output,
and controlled negative proof are recorded in
[`docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md`](../../verification/DLV-UX-002-us1-automated-accessibility-harness.md).

## 7. User Story 2 — Verify keyboard operation and focus behavior

**As a keyboard-only user, I want every shared interaction to have predictable focus and completion behavior, so that I can complete work without a pointing device.**

### Acceptance criteria

- [x] Keyboard checks cover navigation, scope selection, worklist filtering and selection, tables, row actions, forms, validation summary links, confirmation surfaces, approval/posting panels, and conflict dialogs in the synthetic example.
- [x] Checks verify logical tab order, visible focus, escape/cancel behavior, dialog focus trapping, focus restoration, and a safe exit path.
- [x] Checks verify that failed submission focuses the validation summary or documented target and preserves valid input.
- [x] Checks verify that disabled or blocked actions remain understandable and are not the only way to discover why an action is unavailable.
- [x] Async updates do not unexpectedly move focus; material status changes use a documented announcement target.
- [x] A keyboard procedure is available for future critical workflows and records the target, steps, expected result, and observed result.

### Evidence

- Playwright or equivalent interaction tests for the shared example.
- Focus-order and dialog behavior results.
- Manual keyboard checklist template for future workflow journeys.

### User Story 2 implementation evidence

The keyboard interaction tests, shared focus behavior, procedure, and
verification record are documented in
[`docs/verification/DLV-UX-002-us2-keyboard-focus-behavior.md`](../../verification/DLV-UX-002-us2-keyboard-focus-behavior.md).

## 8. User Story 3 — Establish screen-reader and semantic review coverage

**As a screen-reader user, I want structure, state, errors, and progress exposed meaningfully, so that visual layout is not required to understand or complete an action.**

### Acceptance criteria

- [ ] The repository documents at least two approved screen-reader/browser combinations for the accessibility review matrix.
- [ ] The manual procedure covers landmarks, heading hierarchy, labels, descriptions, table headers and row identity, selection state, status/severity meaning, validation errors, dialogs, progress, and dynamic announcements.
- [ ] The procedure covers restricted and masked values without requiring protected content to be exposed in accessible names, errors, or empty states.
- [ ] Representative checks confirm that state, approval, reconciliation, exception, gain/loss, and debit/credit meaning is available as text or an equivalent accessible representation and does not rely on color alone.
- [ ] Manual findings record assistive technology, browser, operating system, target, steps, expected result, observed result, severity, owner, and evidence location.
- [ ] The procedure distinguishes an untested future capability workflow from a tested shared interaction and never records an unexecuted journey as passed.

### Evidence

- Versioned screen-reader/browser matrix and manual procedure.
- Witnessed review record for the shared shell and integrated example.
- Sample finding and remediation-trace record using synthetic data.

## 9. User Story 4 — Verify visual adaptability and motion preferences

**As a user who needs larger text, high contrast, or reduced motion, I want the interface to remain usable, so that accessibility preferences do not remove information or actions.**

### Acceptance criteria

- [ ] Checks cover applicable text and interactive-component contrast requirements for light and dark themes, including status and disabled/restricted states.
- [ ] Representative dense tables, forms, dialogs, and status/process surfaces remain usable at 200 percent text size and 400 percent browser zoom without loss of information or action.
- [ ] Reflow checks identify horizontal clipping, obscured controls, lost table meaning, and inaccessible overflow; any intentional responsive transformation remains semantically understandable.
- [ ] Reduced-motion checks confirm that nonessential animation can be reduced or disabled and that no critical meaning depends on motion.
- [ ] The test procedure records viewport, zoom/text-size setting, motion preference, browser, target, result, and finding reference.
- [ ] Screenshots or other visual evidence contain no secrets or sensitive finance values.

### Evidence

- Automated or scripted checks where practical, plus manual checklist.
- Representative light/dark, zoom/reflow, and reduced-motion evidence.
- Documented contrast or responsive exceptions with owner and remediation status.

## 10. User Story 5 — Produce accessibility qualification evidence and defect decisions

**As the delivery owner, I want accessibility results tied to requirements and severity decisions, so that defects are actionable and release evidence is trustworthy.**

### Acceptance criteria

- [ ] A result template maps each check to its target, requirement or UX acceptance criterion, verification method, environment, owner, result, and evidence reference.
- [ ] Evidence distinguishes pass, fail, blocked, not applicable, and not yet implemented; blocked or not-yet-implemented future workflows are not counted as passes.
- [ ] Findings blocking a Class A workflow are classified as release-blocking critical defects in accordance with NFR-ACC-011; other WCAG AA failures receive an approved remediation decision before release claims.
- [ ] The process records severity, impact, affected workflow/class, owner, corrective action, due date or exception expiry, compensating control where applicable, and retest result.
- [ ] The harness supports rerunning checks after interaction changes and identifies the changed target and affected accessibility evidence.
- [ ] The delivery evidence explicitly states that full review of all 22 critical workflows remains a release qualification obligation when those workflows are implemented.

### Evidence

- Accessibility verification report template and one completed synthetic baseline report.
- Requirement-to-check traceability for NFR-ACC-001–012 and NFR-TST-007 as applicable to this item.
- Defect classification and retest example.
- Clear list of deferred/full-release qualification work.

## 11. Delivery-item acceptance summary

`DLV-UX-002` is complete only when all five stories are complete and:

- [ ] The automated accessibility command, keyboard procedure, screen-reader procedure, visual-adaptation checks, and evidence template are reproducible from a clean checkout.
- [ ] Shared shell, representative components, and the synthetic integrated example have passing baseline evidence or explicitly recorded findings.
- [ ] A controlled negative check proves that the automated gate fails on a known violation.
- [ ] Keyboard, focus, semantic, announcement, zoom/reflow, contrast, and reduced-motion expectations are represented in executable checks or witnessed procedures.
- [ ] Accessibility findings have severity, ownership, remediation or exception decisions, and retest traceability.
- [ ] No future finance workflow, capability, authorization rule, or production qualification result is incorrectly marked complete.
- [ ] No credentials, sensitive fixtures, dependency directories, build output, or local-only configuration is committed.

## 12. Definition of Ready

- [ ] `DLV-UX-001` shared shell, component, and synthetic-example prerequisites remain complete.
- [ ] The approved UX, NFR, frontend technical, and testing specifications remain unchanged or impact is recorded.
- [ ] The supported browser and screen-reader combinations are selected and available for the manual procedure.
- [ ] The automated test runner can mount the shared targets without requiring finance APIs, persistence, authentication, or external services.
- [ ] Accessibility evidence storage and defect ownership conventions are agreed before baseline results are recorded.

## 13. Definition of Done

- [ ] Every acceptance criterion in Sections 6–10 passes or has an explicitly recorded, approved exception within scope.
- [ ] Automated, keyboard, screen-reader, visual-adaptation, and evidence checks are documented and repeatable.
- [ ] The baseline report links results to the applicable UX and NFR requirements.
- [ ] The harness fails safely and visibly when a required automated check cannot run or a known violation is introduced.
- [ ] Manual evidence identifies the exact browser and assistive-technology combination used.
- [ ] Full critical-workflow qualification remains assigned to release/NFR qualification and is not silently deferred inside this item.
- [ ] No critical or high unresolved defect remains within this delivery item without an approved exception.

## 14. Traceability

| Source | Applicability |
|---|---|
| UX §10.1 | Keyboard, semantic state, table, error, dynamic-update, and dialog behavior |
| UXA-COMMON-010 | Cross-cutting keyboard, focus, status, table, validation, and dialog acceptance |
| NFR-ACC-001–012 | WCAG target, keyboard, semantics, non-color meaning, contrast/zoom, errors, updates, time limits, motion, exports, severity, and specialist review |
| NFR-TST-002 | UX acceptance evidence across the supported client/accessibility matrix |
| NFR-TST-003 | Verification method, environment, owner, result, and evidence completeness |
| NFR-TST-007 | Combined automated, keyboard, screen-reader, zoom/reflow, and manual verification |
| Frontend technical spec §§10–11 | Approved accessibility target, tooling boundary, and test layers |
| Testing specification §8 and release §10 | WCAG checks, manual critical-journey review, and accessibility release evidence |

## 15. Explicit follow-on ownership

| Follow-on work | Owner |
|---|---|
| Fixing a shared component defect | `DLV-UX-001` or the component-owning delivery item |
| Accessibility behavior of a finance capability screen | Owning capability delivery item |
| Full review of all 22 critical workflows | Applicable workflow and release qualification items |
| Annual/high-risk specialist review | NFR-ACC-012 / full-system qualification |
| Production accessibility monitoring and support process | `EP-OPS-001` and applicable quality gates |

## 16. Source references

- `docs/specs/finance_ux_workflow_specification_v1.0.md`
- `docs/specs/finance_nonfunctional_requirements_v1.0.md`
- `docs/specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md`
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md`
- `docs/specs/finance_delivery_plan_v1.0.md`
