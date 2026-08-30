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

The application currently renders synthetic UI examples only. It does not
provide authentication, authorization, accounting-scope selection, finance
capabilities, API mutations, database readiness, or authoritative financial
records.

Shared primitives are exported from `src/components/ui/`. Capability code and
the routed application shell will be added by later delivery stories.
