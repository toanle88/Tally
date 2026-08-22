# Local development

The repository is a Go API and React application managed from the root. Node.js 24 and pnpm 11.9.0 are the current JavaScript toolchain.

```bash
pnpm install --frozen-lockfile
pnpm test
pnpm build
```

For the public documentation site:

```bash
pnpm docs:dev
pnpm docs:check
pnpm docs:preview
```

Keep changes small, add tests for changed behavior, and do not commit build output, secrets, local environment files, or alternate frontend lockfiles.
