# Local development

TALLY is managed from the repository root. The supported baseline is Go 1.26.3
or later, Node.js 24 LTS or later, pnpm 11.9.0, Docker Compose v2, and
Terraform 1.8–1.x.
The repository uses pnpm as its only frontend package manager.

## First run

```bash
# Install root tools such as VitePress with the committed lockfile
pnpm install --frozen-lockfile

# Install the web app with its committed lockfile
pnpm --dir web --ignore-workspace install --frozen-lockfile

# Configure local API/database settings
cp .env.example .env

# Start PostgreSQL, migrate, seed, and verify
make db-prepare

# Run the API and web app in separate terminals
pnpm dev-api
pnpm dev-web
```

For browser authentication, copy `web/.env.example` to
`web/.env.local`. Use `VITE_AUTH_MODE=fixture` only for an explicitly local
development fixture run, or configure `VITE_AUTH_MODE=msal` and the Entra
placeholders for an interactive run. The API fails closed for protected
`/api/v1` requests when its authentication configuration is missing;
`/health/live` remains anonymous.

## Checks that matter

```bash
pnpm test              # Go and frontend tests
pnpm build             # API and frontend build
pnpm check             # test + build
pnpm docs:check        # build this VitePress site
make shared-primitives-check
make idempotency-check
make outbox-worker-check
make iam-sensitive-evidence-check
make operational-readiness-check
```

The focused `make` targets are useful while learning a slice. Database-backed
checks require Docker and a configured local database; live Azure, Entra,
provider, load, recovery, and production-qualification checks require their
declared external environment. Verification records state when a command was
blocked by the host rather than treating it as a pass.

## Documentation site

```bash
pnpm docs:dev
pnpm docs:check
pnpm docs:preview
```

VitePress serves `docs-site/` with a `/Tally/` base path for GitHub Pages.
Keep generated output, secrets, local environment files, binaries, and
alternate lockfiles out of commits. Add tests or focused verification when
changing behavior, and do not claim a check passed without command output.

## Local authentication

The API fails closed for protected `/api/v1` requests until its Entra tenant,
audience, and OIDC discovery URL are supplied. `/health/live`
remains anonymous. The browser defaults to an unauthenticated state; it does
not create a development identity automatically.

Copy the API/database template from the repository root to `.env`. Copy
`web/.env.example` to `web/.env.local` for browser authentication; Vite loads
frontend variables from the `web` project directory.

For an explicitly local-only signed-fixture run, set `VITE_AUTH_MODE=fixture`
in `web/.env.local` while using a development build. Fixture mode is not
available in production builds. For an interactive Entra run, set
`VITE_AUTH_MODE=msal` and provide the `VITE_ENTRA_*` placeholders from
`web/.env.example` through a local, uncommitted environment file. Configure
`VITE_ENTRA_AUTHORITY` with the authority for the selected Entra tenant rather
than relying on a provider-specific default. For External ID, the API
discovery URL follows
`https://<tenant-name>.ciamlogin.com/<tenant-id>/v2.0/.well-known/openid-configuration`.
No passwords, production credentials, raw access tokens, or
tenant identifiers belong in the repository.
