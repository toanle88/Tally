# EP-UX-001 — Shared UX and Design System User Stories

| Field | Value |
|---|---|
| Epic | `EP-UX-001` — Shared UX and design system |
| Status | Open — M0 delivery scope is implemented; capability and release qualification remain pending |
| Milestone | `M0` — Engineering foundation |
| Delivery items | `DLV-UX-001` and `DLV-UX-002` |
| Source of detail | The consolidated child sections below retain the delivery-item acceptance criteria and evidence. |

## 1. Outcome

Provide the shared frontend shell, navigation, accounting-scope context,
reusable finance interaction components, and accessibility verification
harness needed by later capability epics.

## 2. Learning objective

Learn how to build accessible, state-aware finance interfaces that preserve
domain ownership, explain blocked and recoverable outcomes, protect restricted
values, and remain reusable without prematurely implementing finance
capabilities.

## 3. Scope

- Tailwind and daisyUI application shell and routed navigation.
- Accounting-scope context, record context, lifecycle, money, evidence, and
  privacy components.
- Worklists, actions, settlement and exception states, progress, results,
  forms, approval/posting panels, validation, confirmation, and conflict
  interactions.
- Automated accessibility checks, keyboard coverage, semantic review
  procedures, visual-adaptation checks, and qualification evidence templates.

## 4. Explicit exclusions

- Finance capability pages, domain commands, API routes, persistence, and
  authorization policy implementation.
- Claims that future finance workflows are implemented or fully accessible.
- Witnessed screen-reader sessions and actual browser-zoom sessions removed from
  the M0 harness scope; they remain release qualification work.
- Capability-specific interaction behavior owned by later epics.

## 5. Delivery-item map

### `DLV-UX-001` — Shared UX and design-system application shell and components

**Status:** Complete for the shared M0 component scope; capability and
qualification work remains excluded.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Establish the design-system foundation and application shell
- User Story 2 — Establish routed navigation and accounting-scope context
- User Story 3 — Build record context, lifecycle, money, evidence, and privacy components
- User Story 4 — Build worklist, action, settlement, exception, progress, and result components
- User Story 5 — Build forms, approval/posting panels, validation, confirmation, and conflict interactions

### `DLV-UX-002` — Accessibility test harness

**Status:** Complete for the owner-approved M0 scope; witnessed screen-reader
and actual browser-zoom qualification remain deferred.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Establish the automated accessibility harness
- User Story 2 — Verify keyboard operation and focus behavior
- User Story 3 — Establish screen-reader and semantic review coverage
- User Story 4 — Verify visual adaptability and motion preferences
- User Story 5 — Produce accessibility qualification evidence and defect decisions

## 6. Dependencies and handoffs

- `DLV-PLAT-001` supplies the application shell and root frontend commands.
- `DLV-PLAT-004` supplies the generated TypeScript API contract boundary.
- `EP-IAM-001` owns authentication, authorization, accounting-scope policy, and
  emergency access.
- Later capability epics own their finance workflow pages, commands, states,
  evidence, and capability-specific accessibility qualification.

## 7. Definition of done

- Both child delivery items have their applicable M0 acceptance criteria and
  evidence reviewed.
- Shared components expose state, ownership, blocked actions, and safe recovery
  meaning without claiming capability behavior.
- Accessibility checks are reproducible and distinguish baseline harness
  evidence from deferred full-workflow qualification.
- No finance capability or authorization rule is marked complete by shared UX
  work alone.

## 8. Source references

- [Finance Platform Delivery Plan](../../specs/finance_delivery_plan_v1.0.md)
- [UX and Workflow Specification](../../specs/finance_ux_workflow_specification_v1.0.md)
- [Frontend UI Technical Specifications](../../specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md)
- [Testing, Performance, and Recovery Technical Specifications](../../specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md)
- Accessibility harness details are consolidated in Section 9 below.

## 9. Consolidated detailed delivery-item stories

### DLV-UX-001 — Shared UX and Design System User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-UX-001` |
| Item type | UX and design-system foundation |
| Parent epic | `EP-UX-001` — Shared UX and design system |
| Milestone | `M0` — Engineering foundation |
| Artifact version | 1.0 |
| Review status | Implementation and verification complete for the five-story UX delivery |
| Implementation status | User Stories 1–5 complete; capability and qualification work remains excluded |
| Delivery profile | Solo, part-time, local-first learning project |
| Dependency position | `DLV-PLAT-001` is complete; `DLV-UX-002` remains a separate follow-up delivery item. |
| Authoritative deliverable | Implement the Tailwind/daisyUI application shell and shared UX component abstractions. |
| Authoritative exit evidence | Shared worklist, detail, action, status, form, and dialog examples pass. |

#### 1. Purpose and scope

Break `DLV-UX-001` into five reviewable stories that establish the shared frontend foundation without implementing finance capabilities or accessibility qualification as a separate delivery item.

This delivery item provides:

- the approved Tailwind CSS and daisyUI styling foundation;
- a routed React application shell with global navigation and accounting-scope context;
- typed, semantic wrappers for all 18 shared UX components in the frontend technical specification;
- fixture-backed examples for common worklist, detail, action, status, form, and dialog behavior; and
- component-level tests that preserve the UX state, privacy, evidence, and correction semantics required by later capability work.

This delivery item does not establish financial facts, domain state transitions, accounting effects, authorization decisions, API endpoints, database records, or completed finance workflows.

#### 2. Authoritative implementation baseline

| Area | Baseline used by these stories |
|---|---|
| Frontend runtime | Node.js `24 LTS`, React `19.2`, TypeScript, and Vite `8`. |
| Styling | Tailwind CSS `4.x` with CSS-first configuration and daisyUI `5.x`. |
| Routing | React Router for the SPA route tree and browser navigation. |
| Server state | TanStack Query `5`; server state remains authoritative and query identity includes capability, accounting scope, record identity, and relevant filter/version. |
| Forms | React Hook Form for form state and Zod for client-side validation; the server remains authoritative. |
| Tables | TanStack Table `8` for sorting, filtering, pagination, column visibility, and eligible-row selection. |
| Frontend tests | Vitest `4`, React Testing Library, and fixture-driven API/problem states. MSW is reserved for actual API interaction tests. |
| Package manager | pnpm only, with the committed lockfile and frozen-lockfile installation. |
| Component boundary | Shared components wrap daisyUI classes and expose typed props. Feature code does not compose raw modal, alert, badge, table, input, or status classes where an approved wrapper exists. |
| Data boundary | Components consume typed presentation view models and synthetic fixtures. They do not calculate or establish financial facts. |

#### 3. Cross-story constraints

1. Preserve the approved modular-monolith architecture and keep frontend capability code separate from shared components.
2. Shared components contain technical presentation and interaction behavior only; finance rules remain in their owning bounded contexts.
3. Monetary values are represented as canonical decimal strings. No monetary component accepts binary floating-point values or performs financial calculation.
4. Established facts are displayed as immutable records with linked correction or replacement lineage; the UI never offers destructive editing for an established fact.
5. State, severity, approval, reconciliation, restriction, and gain/loss meaning never rely on color alone.
6. Scope, state, version, owner, permitted actions, blocked actions, blocking reason, and recovery/correction path remain visible wherever the applicable data exists.
7. Authentication and authorization seams may be represented by typed fixtures or adapters, but Entra authentication and finance authorization are owned by `EP-IAM-001`.
8. API problem handling may be demonstrated with typed fixtures, but this item does not add endpoints, change generated API artifacts, or replace server validation.
9. No database schema, finance capability package, command, event, accounting effect, or workflow implementation is introduced.
10. No credential, token, sensitive payroll/tax/bank/personal value, local environment file, dependency directory, or build output is committed.
11. The dedicated axe, screen-reader, browser-matrix, and accessibility qualification harness belongs to `DLV-UX-002`; these stories still require semantic HTML and deterministic component interaction behavior.
12. Detailed brand, color palette, typography, final page layouts, localization policy, and production performance qualification remain outside this delivery item unless already required by the approved baseline.

#### 4. Shared fixture and interface conventions

The five stories use one synthetic fixture vocabulary so the examples and tests exercise the same semantics:

- `RecordIdentity`: record type, business identifier, authoritative capability, accounting/business scope, lifecycle state, version, source, owner, and sensitivity classification.
- `MoneyAmount`: decimal-string value, currency code, and explicit transaction, functional, or presentation role; optional rate-set and conversion evidence are displayed, not calculated.
- `ActionState`: action label, permitted/blocked state, blocking reason, required confirmation, and safe next action.
- `ProcessState`: stage owner, current state, age, exception, evidence link, and authoritative record link.
- `ApiProblem`: the approved problem shape with type, title, status, code, detail, correlation ID, optional current version, and field errors.

Shared component props are typed and exported through the component implementation paths defined by the frontend technical specification. The exact component names and paths are implementation artifacts, not new API or domain contracts.

#### 5. User Story 1 — Establish the design-system foundation and application shell

**As the TALLY developer, I want an approved Tailwind/daisyUI foundation and semantic base primitives, so that shared finance interactions have a consistent visual and structural starting point.**

##### Value

Creates the styling, theme, status, button, input, panel, and layout foundation required by all later shared components without introducing capability-specific design or business behavior.

##### Acceptance criteria

- [x] Tailwind CSS `4.x` and daisyUI `5.x` are installed through pnpm and the committed lockfile supports `pnpm install --frozen-lockfile`.
- [x] The Vite application loads Tailwind through the approved CSS-first configuration and daisyUI through its approved plugin configuration.
- [x] The semantic theme baseline includes `finance-light` and `finance-dark` themes and the semantic states `success`, `warning`, `error`, `info`, `pending`, `reconciled`, `restricted`, and `disabled`.
- [x] Shared low-level primitives provide typed, semantic wrappers for buttons, links, headings, fields, panels, status badges, tables, and confirmation surfaces.
- [x] State and severity examples include text labels and accessible names in addition to visual styling.
- [x] The application shell has a stable responsive layout that can contain global navigation, scope context, page content, and status feedback.
- [x] The shell does not claim authentication, authorization, database readiness, or finance capability availability.
- [x] A deterministic component test proves that semantic state tokens render their text meaning and that the shell renders its main regions.
- [x] The frontend production build succeeds without TypeScript errors or generated build output being committed.

##### Evidence

- `web/package.json` and the committed pnpm lockfile showing the approved styling dependencies.
- Rendered shell example showing light/dark theme and semantic status variants.
- Vitest/Testing Library output for shell and primitive tests.
- Successful frontend build output.

##### User Story 1 implementation evidence

- `pnpm install --frozen-lockfile --ignore-workspace` passed from `web/` using `web/pnpm-lock.yaml`.
- `pnpm -C web test` passed: 2 test files and 10 tests.
- `pnpm -C web build` passed with Tailwind/daisyUI CSS emitted and no TypeScript errors.

#### 6. User Story 2 — Establish routed navigation and accounting-scope context

**As an authorized TALLY user, I want predictable navigation and an explicit accounting/business scope context, so that I always know where I am and which scope applies before reviewing or initiating work.**

##### Value

Establishes the route and scope boundaries that later capability workspaces can extend without duplicating navigation, scope, or protected-route behavior.

##### Component in scope

- `CMP-001` — Accounting Scope Selector

##### Acceptance criteria

- [x] React Router provides a typed route tree with stable screen identifiers and paths aligned to the approved UX and frontend screen catalog.
- [x] The shell exposes the approved global navigation areas: Home, Work, Records, Approvals, Exceptions, Reports, Administration, and Audit.
- [x] The route foundation includes the shared operational route contracts for `XCT-WS-01`, `XCT-SCR-01`, and `CON-SCR-01` without implementing their finance resolution logic.
- [x] A reusable development/example entry renders the shared component examples needed for the delivery-item exit evidence.
- [x] The persistent scope context can display tenant, legal entity, ledger, accounting book, functional currency, and period when supplied by the fixture.
- [x] A scope change refreshes the displayed scope-bound context, prevents submission against a stale scope, and exposes an explicit discard/return path when unsaved changes exist.
- [x] Scope changes provide a single integration point for canceling in-flight work and clearing scope-bound query state; no authoritative financial record is stored in a global client store.
- [x] Protected-route behavior is represented by an authentication/scope adapter seam and fixtures only; token acquisition, Entra integration, and permission evaluation remain excluded.
- [x] Navigation tests cover active navigation, direct route access, unknown-route handling, browser back/forward behavior, and stale-scope prevention.

##### Evidence

- Typed route registry and React Router configuration.
- Shell navigation and scope-context example.
- Vitest/Testing Library route and scope tests.
- Boundary review confirming that capability-specific pages remain owned by later delivery items.

Implementation evidence for User Story 2 is recorded in
`docs/verification/DLV-UX-001-us2-routed-navigation-scope-context.md`.

#### 7. User Story 3 — Build record context, lifecycle, money, evidence, and privacy components

**As a finance user, I want record identity, state history, amount meaning, evidence, and sensitivity information presented together, so that I can understand the authoritative fact without losing lineage or exposing restricted data.**

##### Components in scope

- `CMP-002` — Record Identity Header
- `CMP-004` — Lifecycle Timeline
- `CMP-007` — Money and Currency Panel
- `CMP-009` — Correction Lineage Panel
- `CMP-013` — Evidence Drawer
- `CMP-015` — Sensitive Data Guard
- `CMP-016` — Legal Hold Indicator

##### Acceptance criteria

- [x] `CMP-002` displays record identity, authoritative capability, scope, lifecycle state, version, source, owner, and sensitivity label.
- [x] `CMP-004` displays established state transitions with actor, timestamp, decision, and linked correction/replacement references in chronological order.
- [x] `CMP-007` keeps transaction, functional, and presentation amounts distinct, displays currency and rounding labels, and shows rate evidence when supplied.
- [x] `CMP-007` rejects or does not accept numeric monetary props that could silently introduce binary floating-point values.
- [x] `CMP-009` preserves the original established fact and links reversal, amendment, return, unapplication, replacement, compensation, or supersession records without offering destructive edit behavior.
- [x] `CMP-013` presents access-controlled source, approval, posting, provider/authority, reconciliation, close, statement, and audit evidence links supplied by the fixture.
- [x] `CMP-015` masks restricted values by default, labels the restricted section, prevents unauthorized export actions in the presentation layer, and does not reveal protected values in errors or empty states.
- [x] `CMP-016` displays legal hold independently from business lifecycle state and indicates that retention destruction is blocked while business correction remains a separate concern.
- [x] Component tests cover established facts, correction lineage, multi-currency labels, missing evidence, masked values, restricted access, and legal hold.

##### Evidence

- Typed component props and synthetic record/evidence fixtures.
- Detail example combining identity, lifecycle, money, lineage, evidence, sensitivity, and legal-hold surfaces.
- Component test output covering normal, restricted, corrected, and incomplete-evidence states.

Implementation evidence for User Story 3 is recorded in
`docs/verification/DLV-UX-001-us3-record-context-lifecycle-money-evidence-privacy.md`.

#### 8. User Story 4 — Build worklist, action, settlement, exception, progress, and result components

**As an operational finance user, I want worklists and process surfaces to explain what can happen next and who owns each outcome, so that I can resolve work without confusing pending, rejected, unavailable, reconciled, or duplicate results.**

##### Components in scope

- `CMP-003` — State-Aware Action Bar
- `CMP-008` — Settlement and Reconciliation Panel
- `CMP-011` — Exception Resolution Panel
- `CMP-014` — Worklist and Saved Filters
- `CMP-017` — Result Lookup
- `CMP-018` — Process Progress Panel

##### Acceptance criteria

- [x] `CMP-014` supports applicable scope, state, owner, date, amount, currency, exception, and approval filters, with sorting, pagination, column visibility, saved fixture-backed views, and eligible-row selection.
- [x] Worklist examples show the responsible owner, age, amount/currency where applicable, next action, and authoritative record link.
- [x] Bulk actions are available only when every selected fixture row is eligible; the component does not imply that client eligibility replaces server validation.
- [x] `CMP-003` displays permitted actions and discoverable blocked actions with a reason and safe next action, refreshing after a material result.
- [x] `CMP-008` distinguishes gross, returned, reversed, canceled, remaining, net, owner acknowledgement, reconciliation, and exception states.
- [x] `CMP-011` displays exception type, scope/amount, owner, evidence, age, authorized resolutions, and resulting state.
- [x] `CMP-017` distinguishes safe duplicate, ambiguous outcome, and identity-content conflict and links to the established result or the required new business identity.
- [x] `CMP-018` identifies each cross-capability step owner and distinguishes current, pending, failed, partially completed, reconciled, and terminal states without presenting downstream outcomes as synchronous success.
- [x] Worklist and process tests cover empty, loading, pending, unavailable, rejected, partial, reconciled, duplicate, conflict, and exception states.
- [x] The worklist example supports a result link to the authoritative detail surface and does not become a second mutation surface for another capability.

##### Evidence

- Worklist, action, settlement, exception, result, and process fixtures.
- Worklist/detail example showing filtering, assignment, state-aware actions, and process ownership.
- Component test output covering all required result categories and selection rules.

Implementation evidence for User Story 4 is recorded in
`docs/verification/DLV-UX-001-us4-worklist-process-components.md`.

#### 9. User Story 5 — Build forms, approval/posting panels, validation, confirmation, and conflict interactions

**As a finance user submitting or reviewing a material action, I want clear validation, approval, posting, confirmation, and conflict behavior, so that I can correct input deliberately and never unknowingly repeat or overwrite a business outcome.**

##### Components in scope

- `CMP-005` — Approval Panel
- `CMP-006` — Posting Panel
- `CMP-010` — Validation Summary
- `CMP-012` — Version Conflict Dialog

##### Acceptance criteria

- [x] Form examples use React Hook Form for local form state and Zod for immediate client validation while preserving the server-authoritative boundary.
- [x] Monetary form fields display locale-aware values but submit canonical decimal strings in the fixture submission model.
- [x] `CMP-010` groups field, line, business-rule, authorization, dependency, and conflict errors; each error links to its cause and preserves valid input.
- [x] Failed submission gives the validation summary a predictable focus target and associates field/line errors with their controls.
- [x] `CMP-005` distinguishes approval requested, pending, delegated, escalated, decided, applied, rejected, expired, and invalidated states; a recorded decision is not shown as applied before revalidation.
- [x] `CMP-006` distinguishes posting request, pending, established result, rejection, failure, retry eligibility, reversal, and reconciliation, including period/gate evidence where supplied.
- [x] Irreversible or high-risk confirmation summarizes record, version, scope, amount/currency, intended state change, accounting owner, approvals, and lineage effect.
- [x] `CMP-012` shows expected versus current version/state, changed values, current owner, established-result lookup, and whether a deliberate retry is permitted.
- [x] Version conflicts never automatically resubmit a mutation. Identity-content conflicts require a new business identity.
- [x] Typed problem fixtures cover success, domain rejection, authorization denial, version conflict, idempotency conflict, dependency unavailability, ambiguous outcome, and unexpected failure with correlation reference.
- [x] Form and dialog tests cover keyboard completion, focus restoration, validation association, confirmation cancellation, safe retry, no blind retry, and preserved draft input where permitted.
- [x] The integrated example demonstrates worklist-to-detail navigation and renders representative worklist, detail, action, status, form, and dialog states using synthetic data.

##### Evidence

- Typed form, problem, approval, posting, and conflict fixtures.
- Integrated example covering the delivery-item exit evidence.
- Vitest/Testing Library output for form, validation, approval, posting, confirmation, and conflict behavior.
- Frontend build and test output.

Implementation evidence for User Story 5 is recorded in
`docs/verification/DLV-UX-001-us5-form-approval-posting-validation-conflict.md`.

#### 10. Delivery-item acceptance summary

`DLV-UX-001` is complete only when all five stories are complete and every condition below passes:

- [x] Tailwind/daisyUI styling and semantic state tokens are installed and reproducible through pnpm.
- [x] The routed application shell exposes global navigation, scope context, and shared operational route seams.
- [x] All 18 shared components from the frontend technical specification are implemented as typed semantic wrappers and are covered by component tests.
- [x] Synthetic examples demonstrate worklist, detail, action, status, form, and dialog behavior.
- [x] State, scope, version, ownership, correction, evidence, privacy, currency, conflict, and recovery semantics are represented without introducing finance business logic.
- [x] Frontend tests and build pass, and no generated output, secrets, or alternate lockfile is committed.
- [x] Basic semantic and keyboard behavior is tested, while the dedicated accessibility harness remains assigned to `DLV-UX-002`.
- [x] No finance capability, API endpoint, persistence behavior, authentication implementation, authorization policy, workflow demonstration, or M0 milestone is incorrectly marked complete.

#### 11. Explicit exclusions and follow-on ownership

| Excluded work | Owning delivery item or epic |
|---|---|
| Accessibility axe harness, screen-reader procedure, browser matrix, and qualification evidence | `DLV-UX-002`; applicable NFR qualification items |
| Entra authentication, permissions, accounting-scope authorization, and segregation-of-duties decisions | `EP-IAM-001` |
| Finance capability routes and pages | Owning capability delivery items |
| Finance records, aggregates, commands, events, accounting effects, and correction rules | Owning bounded-context delivery items |
| OpenAPI endpoints and generated API contract changes | `DLV-PLAT-004` and owning capability delivery items |
| Database schemas, migrations, queries, and authoritative persistence | `DLV-PLAT-002`, `DLV-PLAT-003`, and owning capability delivery items |
| Structured logs, metrics, traces, dashboards, and runbooks | `DLV-OPS-001`, `DLV-OPS-002` |
| Full workflow E2E coverage for `WF-6.1`–`WF-7.15` | Workflow delivery items and later quality gates |
| Final brand system, localization policy, production performance, and visual qualification | Applicable UX/NFR qualification scope |

#### 12. Definition of Ready

This delivery item is ready only when:

- [ ] `DLV-PLAT-001` and its frontend shell/build/test prerequisites remain complete.
- [ ] The UX v1.0, frontend technical specification, system-design baseline, and delivery-plan checkpoints remain unchanged or their impact is recorded.
- [ ] The React Router approach, five-story split, complete 18-component scope, and fixture-driven examples are confirmed.
- [ ] The shared fixture vocabulary and component boundary are agreed before capability-specific view models are introduced.
- [ ] Tailwind/daisyUI, React Router, TanStack Query, React Hook Form, Zod, and TanStack Table dependency versions are available through the approved pnpm workflow.
- [ ] Required shell, route, scope, component, form, conflict, and example evidence is identified.
- [ ] No API, persistence, authentication, or finance capability dependency is silently required to demonstrate this item.

#### 13. Definition of Done

This delivery item is done only when:

- [ ] Every acceptance criterion in Sections 5–9 passes; none is silently deferred.
- [ ] All 18 shared components are present under the shared component boundary and expose typed props.
- [ ] Component behavior is covered by deterministic Vitest/Testing Library tests using synthetic, non-sensitive fixtures.
- [ ] Frontend dependency installation, tests, and production build pass with pnpm and the committed lockfile.
- [ ] The integrated examples demonstrate the authoritative UX state and recovery semantics required by the delivery item.
- [ ] Semantic HTML, labels, visible focus, keyboard operation, error association, and non-color status meaning are tested at the component level.
- [ ] `DLV-UX-002` remains the owner of the dedicated accessibility harness and full qualification evidence.
- [ ] Documentation and traceability identify the implemented component groups, test evidence, and explicit exclusions.
- [ ] No credential, sensitive fixture value, dependency directory, build output, alternate frontend lockfile, or local-only configuration is committed.
- [ ] No critical or high unresolved defect remains within this delivery item.

#### 14. Traceability and quality-gate contribution

| Traceability field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-UX-001` |
| Delivery item | `DLV-UX-001` |
| Direct global controls | `GFR-009`, `GFR-016`, `GFR-017` |
| Supporting UX criteria | `UXA-COMMON-001`–`UXA-COMMON-010` at shared-component level |
| Supporting technical decisions | `ADR-004`, `TADR-009`, `TS-FRONTEND` |
| Supporting NFR areas | Accessibility and inclusive use, compatibility, privacy, testability, and maintainability; full qualification remains in the applicable NFR items. |
| Primary test layers | Frontend component tests and fixture-driven UI integration tests |
| Exit evidence | Shared worklist, detail, action, status, form, and dialog examples pass. |

The shared components contribute presentation and interaction controls for `GFR-010`, `GFR-011`, `GFR-013`, `GFR-014`, `GFR-015`, `GFR-018`, and `GFR-019`, but those global requirements remain owned by their delivery items and are not marked complete by `DLV-UX-001`.

#### 15. Source references

- `docs/specs/finance_domain_model_ddd.md` — domain ownership, immutable facts, correction, concurrency, and evidence semantics.
- `docs/specs/prd/` — functional baseline that shared UX must present without redefining.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — navigation, scope context, record-detail shell, common interaction model, component catalog, messages, privacy, evidence, and UX acceptance criteria.
- `docs/specs/finance_nonfunctional_requirements_v1.0.md` — accessibility, privacy, compatibility, testability, and release-quality expectations.
- `docs/specs/system_design/01_solution_architecture_overview_v1.0.md` — approved frontend technology baseline.
- `docs/specs/system_design/02_application_module_design_v1.0.md` — shared frontend architecture and error model.
- `docs/specs/system_design/05_architecture_traceability_decisions_v1.0.md` — `ADR-004`, `TADR-009`, and `TS-FRONTEND` traceability.
- `docs/specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md` — frontend structure, component paths, route catalog, form/table rules, API problem shape, styling, and test baseline.
- `docs/specs/finance_delivery_plan_v1.0.md` — `EP-UX-001`, `DLV-UX-001`, dependencies, quality gates, and definition-of-ready/done rules.

#### 16. Consistency review checklist

- [ ] Exactly five user stories are defined for `DLV-UX-001`.
- [ ] All 18 shared component IDs `CMP-001` through `CMP-018` are assigned exactly once.
- [ ] `DLV-UX-002` accessibility-harness ownership is explicit.
- [ ] Capability-specific finance behavior is excluded.
- [ ] No story claims completion of an FR, workflow, API, persistence, authentication, authorization, or M0 milestone.
- [ ] Direct global-control traceability is limited to the UX-owned controls, with supporting controls clearly identified as contributions only.
- [ ] All acceptance criteria and completion checkboxes remain unchecked until implementation evidence exists.

### DLV-UX-002 — Accessibility Test Harness User Stories


#### 1. Outcome

Provide a repeatable accessibility verification boundary for the TALLY frontend. The harness shall combine automated checks, keyboard interaction checks, screen-reader procedures, zoom/reflow and reduced-motion checks, and documented evidence without claiming that future finance capabilities are already accessible or implemented.

**M0 scope decision:** The owner has removed the witnessed screen-reader sessions
and actual browser-zoom session from this delivery item's completion scope. They
remain deferred qualification work and are not represented as passed evidence.

#### 2. Learning objective

Learn how to turn accessibility requirements into executable frontend checks and witnessed review evidence while keeping component ownership, workflow ownership, and release qualification boundaries explicit.

#### 3. Scope and ownership

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

#### 4. Source requirements and boundaries

The stories are derived from:

- UX specification §10.1 and `UXA-COMMON-010`.
- NFR-ACC-001 through NFR-ACC-012.
- NFR-TST-002, NFR-TST-003, and NFR-TST-007.
- Frontend technical specification §§10–11.
- Testing, performance, and recovery specification §8 and release condition §10.

The harness verifies presentation and interaction semantics. It does not become authoritative for finance state, authorization decisions, or workflow outcomes. A failing check identifies evidence and a remediation owner; it does not silently change business behavior.

#### 5. Shared test vocabulary

The harness uses the existing synthetic examples and shared component vocabulary:

- **Target:** a shared component, route, representative screen, or workflow journey under test.
- **Interaction mode:** automated axe, keyboard-only, screen reader, zoom/reflow, reduced motion, or manual review.
- **Finding:** rule/check identifier, target, observed result, severity, reproducibility, and owner.
- **Evidence:** command or procedure, environment, browser/assistive-technology combination, timestamp, result, and linked finding.
- **Workflow class:** representative shared interaction or future Class A critical workflow. The harness must not imply future workflow completion.

#### 6. User Story 1 — Establish the automated accessibility harness

**As the TALLY developer, I want a deterministic automated accessibility check entry point, so that shared UI regressions are detected consistently.**

##### Acceptance criteria

- [x] The frontend exposes a documented command for automated accessibility checks that runs in non-watch mode and returns a non-zero result for an accessibility violation.
- [x] The automated suite uses the approved frontend test/tooling boundary and scans the shared shell, representative shared components, and the synthetic integrated example with synthetic data.
- [x] The suite checks semantic regions, headings, labels, accessible names, status meaning, table structure, validation associations, dialog semantics, and live-region markup where applicable.
- [x] The suite does not suppress, broadly disable, or reclassify violations merely to make the baseline pass; any justified rule configuration is narrow, documented, and reviewed.
- [x] Test output identifies the target and failing rule sufficiently for a developer to reproduce the finding.
- [x] The harness does not log credentials, tokens, restricted values, or sensitive fixture data.

##### Evidence

- Automated test configuration and documented command.
- Passing baseline output for shared targets.
- Controlled negative check proving a known violation fails the command.
- Rule-configuration review showing no unexplained global exclusions.

##### User Story 1 implementation evidence

The implemented harness, target mapping, dependency boundary, passing output,
and controlled negative proof are recorded in
[`docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md`](../../verification/DLV-UX-002-us1-automated-accessibility-harness.md).

#### 7. User Story 2 — Verify keyboard operation and focus behavior

**As a keyboard-only user, I want every shared interaction to have predictable focus and completion behavior, so that I can complete work without a pointing device.**

##### Acceptance criteria

- [x] Keyboard checks cover navigation, scope selection, worklist filtering and selection, tables, row actions, forms, validation summary links, confirmation surfaces, approval/posting panels, and conflict dialogs in the synthetic example.
- [x] Checks verify logical tab order, visible focus, escape/cancel behavior, dialog focus trapping, focus restoration, and a safe exit path.
- [x] Checks verify that failed submission focuses the validation summary or documented target and preserves valid input.
- [x] Checks verify that disabled or blocked actions remain understandable and are not the only way to discover why an action is unavailable.
- [x] Async updates do not unexpectedly move focus; material status changes use a documented announcement target.
- [x] A keyboard procedure is available for future critical workflows and records the target, steps, expected result, and observed result.

##### Evidence

- Playwright or equivalent interaction tests for the shared example.
- Focus-order and dialog behavior results.
- Manual keyboard checklist template for future workflow journeys.

##### User Story 2 implementation evidence

The keyboard interaction tests, shared focus behavior, procedure, and
verification record are documented in
[`docs/verification/DLV-UX-002-us2-keyboard-focus-behavior.md`](../../verification/DLV-UX-002-us2-keyboard-focus-behavior.md).

#### 8. User Story 3 — Establish screen-reader and semantic review coverage

**As a screen-reader user, I want structure, state, errors, and progress exposed meaningfully, so that visual layout is not required to understand or complete an action.**

##### Acceptance criteria

- [x] The repository documents at least two approved screen-reader/browser combinations for the accessibility review matrix.
- [x] The manual procedure covers landmarks, heading hierarchy, labels, descriptions, table headers and row identity, selection state, status/severity meaning, validation errors, dialogs, progress, and dynamic announcements.
- [x] The procedure covers restricted and masked values without requiring protected content to be exposed in accessible names, errors, or empty states.
- [x] Representative checks confirm that state, approval, reconciliation, exception, gain/loss, and debit/credit meaning is available as text or an equivalent accessible representation and does not rely on color alone.
- [x] Manual findings record assistive technology, browser, operating system, target, steps, expected result, observed result, severity, owner, and evidence location.
- [x] The procedure distinguishes an untested future capability workflow from a tested shared interaction and never records an unexecuted journey as passed.

##### Evidence

- Versioned screen-reader/browser matrix and manual procedure.
- Witnessed review record for the shared shell and integrated example.
- Sample finding and remediation-trace record using synthetic data.

Repository matrix, procedure, and sample trace: [`docs/verification/DLV-UX-002-us3-screen-reader-semantic-review.md`](../../verification/DLV-UX-002-us3-screen-reader-semantic-review.md). Witnessed SR-01 and SR-02 execution is deferred to later qualification.

#### 9. User Story 4 — Verify visual adaptability and motion preferences

**As a user who needs larger text, high contrast, or reduced motion, I want the interface to remain usable, so that accessibility preferences do not remove information or actions.**

##### Acceptance criteria

- [x] Checks cover applicable text and interactive-component contrast requirements for light and dark themes, including status and disabled/restricted states.
- [x] Representative dense tables, forms, dialogs, and status/process surfaces pass the automated 200 percent text and 400 percent browser-zoom-equivalent checks. Actual browser-zoom review is deferred to later qualification.
- [x] Reflow checks identify horizontal clipping, obscured controls, lost table meaning, and inaccessible overflow; any intentional responsive transformation remains semantically understandable.
- [x] Reduced-motion checks confirm that nonessential animation can be reduced or disabled and that no critical meaning depends on motion.
- [x] The test procedure records viewport, zoom/text-size setting, motion preference, browser, target, result, and finding reference.
- [x] Screenshots or other visual evidence contain no secrets or sensitive finance values.

##### Evidence

- Automated or scripted checks where practical, plus manual checklist.
- Representative light/dark, zoom/reflow, and reduced-motion evidence.
- Documented contrast or responsive exceptions with owner and remediation status.

Implementation and automated verification record: [`docs/verification/DLV-UX-002-us4-visual-adaptability-motion.md`](../../verification/DLV-UX-002-us4-visual-adaptability-motion.md).

#### 10. User Story 5 — Produce accessibility qualification evidence and defect decisions

**As the delivery owner, I want accessibility results tied to requirements and severity decisions, so that defects are actionable and release evidence is trustworthy.**

##### Acceptance criteria

- [x] A result template maps each check to its target, requirement or UX acceptance criterion, verification method, environment, owner, result, and evidence reference.
- [x] Evidence distinguishes pass, fail, blocked, not applicable, and not yet implemented; blocked or not-yet-implemented future workflows are not counted as passes.
- [x] Findings blocking a Class A workflow are classified as release-blocking critical defects in accordance with NFR-ACC-011; other WCAG AA failures receive an approved remediation decision before release claims.
- [x] The process records severity, impact, affected workflow/class, owner, corrective action, due date or exception expiry, compensating control where applicable, and retest result.
- [x] The harness supports rerunning checks after interaction changes and identifies the changed target and affected accessibility evidence.
- [x] The delivery evidence explicitly states that full review of all 22 critical workflows remains a release qualification obligation when those workflows are implemented.

##### Evidence

- [x] Accessibility verification report template and one completed synthetic baseline report.
- [x] Requirement-to-check traceability for NFR-ACC-001–012 and NFR-TST-007 as applicable to this item.
- [x] Defect classification and retest example.
- [x] Clear list of deferred/full-release qualification work.

Implementation and verification evidence: [`docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md`](../../verification/DLV-UX-002-us5-accessibility-qualification-evidence.md).

#### 11. Delivery-item acceptance summary

`DLV-UX-002` is complete for its revised M0 scope when all five stories are complete and:

- [x] The automated accessibility command, keyboard procedure, screen-reader procedure, visual-adaptation checks, and evidence template are reproducible from a clean checkout.
- [x] Shared shell, representative components, and the synthetic integrated example have passing baseline evidence or explicitly recorded findings.
- [x] A controlled negative check proves that the automated gate fails on a known violation.
- [x] Keyboard, focus, semantic, announcement, zoom/reflow, contrast, and reduced-motion expectations are represented in executable checks or documented procedures. Witnessed screen-reader and actual browser-zoom execution is deferred.
- [x] Accessibility findings have severity, ownership, remediation or exception decisions, and retest traceability.
- [x] No future finance workflow, capability, authorization rule, or production qualification result is incorrectly marked complete.
- [x] No credentials, sensitive fixtures, dependency directories, build output, or local-only configuration is committed.

#### 12. Definition of Ready

- [ ] `DLV-UX-001` shared shell, component, and synthetic-example prerequisites remain complete.
- [ ] The approved UX, NFR, frontend technical, and testing specifications remain unchanged or impact is recorded.
- [ ] The supported browser and screen-reader combinations are selected and available for the manual procedure.
- [ ] The automated test runner can mount the shared targets without requiring finance APIs, persistence, authentication, or external services.
- [ ] Accessibility evidence storage and defect ownership conventions are agreed before baseline results are recorded.

#### 13. Definition of Done

- [x] Every acceptance criterion in Sections 6–10 passes or has an explicitly recorded M0 scope decision; deferred manual qualification remains assigned to later release qualification.
- [x] Automated, keyboard, screen-reader, visual-adaptation, and evidence checks are documented and repeatable.
- [x] The baseline report links results to the applicable UX and NFR requirements.
- [x] The harness fails safely and visibly when a required automated check cannot run or a known violation is introduced.
- [x] Manual evidence requirements identify the exact browser and assistive-technology combination to use when deferred qualification runs.
- [x] Full critical-workflow qualification remains assigned to release/NFR qualification and is explicitly deferred from this item.
- [x] No critical or high unresolved defect remains within this delivery item without an approved exception.

#### 14. Traceability

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

#### 15. Explicit follow-on ownership

| Follow-on work | Owner |
|---|---|
| Fixing a shared component defect | `DLV-UX-001` or the component-owning delivery item |
| Accessibility behavior of a finance capability screen | Owning capability delivery item |
| Full review of all 22 critical workflows | Applicable workflow and release qualification items |
| Annual/high-risk specialist review | NFR-ACC-012 / full-system qualification |
| Production accessibility monitoring and support process | `EP-OPS-001` and applicable quality gates |
| Deferred witnessed screen-reader and actual browser-zoom qualification | `EP-QUAL-001` / applicable release qualification gate |

#### 16. Source references

- `docs/specs/finance_ux_workflow_specification_v1.0.md`
- `docs/specs/finance_nonfunctional_requirements_v1.0.md`
- `docs/specs/technical_specifications/05_frontend_ui_technical_specifications_v1.0.md`
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md`
- `docs/specs/finance_delivery_plan_v1.0.md`

