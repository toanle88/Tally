# DLV-UX-001 User Story 2 Verification

Branch: `feat/dlv-ux-001-us2-routed-navigation-scope-context`

User Story 2 establishes the frontend route and accounting-scope boundaries.
It does not implement authentication, authorization policy evaluation, finance
capabilities, API mutations, persistence, or authoritative financial records.

Implemented evidence:

- `web/src/routes/route-registry.ts` defines stable global and operational
  route identifiers and paths.
- `web/src/routes/router.tsx` provides active navigation, protected-route
  fixture seams, direct routes, and unknown-route handling.
- `web/src/lib/scope/scope-context.tsx` provides in-memory scope state,
  revision-based stale protection, and the single scope-change integration
  point.
- `web/src/components/accounting-scope-selector/` renders the persistent scope
  context and unsaved-work confirmation.
- `web/src/lib/auth/auth-scope-adapter.ts` contains the fixture-only adapter
  contract.

Verification commands:

```text
pnpm -C web test
pnpm -C web build
```

Observed result: `5` test files and `21` tests passed; the production TypeScript
and Vite build completed successfully.
