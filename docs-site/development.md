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
