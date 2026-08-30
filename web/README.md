# TALLY frontend

The frontend is a React 19, TypeScript, and Vite application. Tailwind CSS 4
is the primary styling and layout system. daisyUI 5 provides the CSS-first
theme foundation and is used behind shared UI wrappers.

## Local commands

Run these commands from the repository root:

```bash
pnpm --dir web install --frozen-lockfile
pnpm -C web test
pnpm -C web build
pnpm dev-web
```

The frontend lockfile is maintained at `web/pnpm-lock.yaml`. pnpm is the only
supported frontend package manager.

## Current foundation

User Story 1 provides:

- `finance-light` and `finance-dark` daisyUI themes;
- semantic state tokens for success, warning, error, info, pending,
  reconciled, restricted, and disabled;
- typed wrappers for buttons, links, headings, fields, panels, status badges,
  tables, and inline confirmation surfaces; and
- a responsive presentational shell with navigation, scope, content, and
  status-feedback regions.

User Story 2 adds:

- stable React Router paths for the eight global navigation areas;
- contract-only routes for `XCT-WS-01`, `XCT-SCR-01`, and `CON-SCR-01`;
- an in-memory accounting-scope context with tenant, legal entity, ledger,
  accounting book, currency, and period display;
- fixture-only authentication/scope protection and explicit unsaved-work
  discard/return handling; and
- stale-scope guards plus the integration point for canceling work and clearing
  scope-bound query state.

User Story 3 adds:

- typed record identity, lifecycle, money, correction-lineage, evidence,
  sensitive-data, and legal-hold component wrappers;
- a synthetic established-record detail example at `/development/examples`;
- decimal-string money props with distinct transaction, functional, and
  presentation roles plus supplied rate and rounding evidence; and
- default masking for restricted values with fixture-controlled authorized
  reveal/export actions.

User Story 4 adds:

- typed operational fixtures and wrappers for worklists, saved filters,
  state-aware actions, settlement/reconciliation, exception resolution, result
  lookup, and process progress;
- a TanStack Table-backed worklist with exact decimal-string filtering,
  pagination, sorting, column visibility, fixture views, and eligibility-aware
  selection; and
- explicit pending, unavailable, rejected, partial, reconciled, duplicate,
  conflict, exception, owner, and next-action presentation without API
  mutation or authoritative finance state.

The application still does not provide Entra authentication, authorization
policy evaluation, finance capabilities, API mutations, database readiness, or
authoritative financial records. The development examples and fixtures are not
production data or a global financial-record store.

Shared primitives are exported from `src/components/ui/`. Capability code and
the routed application shell will be added by later delivery stories.
