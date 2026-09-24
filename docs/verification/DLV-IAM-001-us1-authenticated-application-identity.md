# DLV-IAM-001 — User Story 1 authenticated application identity

## Status

Local User Story 1 verification is recorded on branch
`codex/iam-us1-authenticated-application-identity`.

The evidence covers the technical authentication boundary only: signed local
fixtures, bearer validation, actor mapping port, fail-closed API middleware,
OpenAPI bearer metadata/401 responses, and the browser auth provider seam.
Identity schema, user lifecycle, roles, policies, segregation, emergency
access, and finance authorization remain planned for User Stories 2–7.

## Verified checks

| Check | Result | Evidence boundary |
|---|---|---|
| `go test ./...` | PASS | Identity, JWT/JWKS, middleware, API routing, telemetry, and existing Go packages. |
| `pnpm -C web test` | PASS | 13 files, 56 tests, including MSAL, step-up, session deadlines, fixture mode, and one-refresh retry. |
| `pnpm -C web run build` | PASS | Strict TypeScript build and Vite production bundle. |
| `make check` | PASS | Migration validation/checksum inventory and full Go suite. |
| `make api-check` | PASS | Contract lint, deterministic Go/TypeScript generation, committed artifact drift, and type-check fixture. |
| `make api-negative-check` | PASS | Invalid contract and generated-artifact mutations rejected without secret-like diagnostics. |
| `pnpm build` | PASS | API compilation and frontend production build. |
| `pnpm contract:lint` | PASS | OpenAPI security scheme, shared 401 response, and all 193 operation responses lint successfully. |
| Pinned ogen generation from the verified contract bundle | PASS | Committed Go artifacts include the generated authentication response/security surface. |
| Pinned TypeScript generation from the verified contract bundle | PASS | Committed generated Fetch client artifacts are deterministic. |

## Security behavior covered

- Issuer, tenant, audience, signature, algorithm, expiry, `nbf`, and the
  approved 120-second clock-skew rule are validated.
- Unknown keys and discovery/JWKS failures fail closed with safe dependency
  diagnostics. Missing, malformed, invalid, and unmapped subjects do not
  reveal claims, tokens, policy details, or provider errors.
- Authentication telemetry contains bounded outcome/classification fields and
  excludes Authorization headers and sensitive claim payloads.
- `/health/live` is anonymous; future `/api/v1` operations are behind the
  authentication boundary. No `/me` or login API endpoint was added.

## Limitations

The local signed-fixture verification passed. A live Entra tenant, real
production security controls, and production qualification remain unverified.
The External ID discovery URL is documented as a placeholder only; no tenant
