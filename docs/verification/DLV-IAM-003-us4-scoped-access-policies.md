# DLV-IAM-003 — User Story 4 verification evidence

Status: local implementation complete on branch `codex/iam-us4-scoped-access-policies`.
This delivery evaluates durable identity-owned policy revisions; full policy
administration UI/API remains a follow-up and is not marked complete here.

## Traceability

- Epic: EP-IAM-001
- Story: User Story 4 — Evaluate scoped access policies
- Delivery: DLV-FR-IAM-003, DLV-GFR-002
- Security/observability: NFR-SEC-004, NFR-SEC-006, NFR-SEC-011, NFR-OBS-008
- Domain: DDD §§2.18 and 8.1
- UX: IAM-SCR-05, CMP-015

## Implemented surfaces

- `internal/identity/authorization.go` defines the typed `DecisionInput`, immutable `AccessPolicy` revisions, conjunctive allow-only `AccessRule` matching, exact-decimal amount bounds, effective-date handling, default deny, safe decision/policy references, stale/unavailable/expired outcomes, and independent field reveal/export decisions. `MemoryAccessPolicyStore` supplies deterministic fixtures and rejects duplicate immutable revisions.
- `internal/identity/postgres_access_policy_repository.go`, `db/migrations/identity/00003_create_access_policy_schema.sql`, `db/queries/identity/access_policy.sql`, and generated SQLC output read the current identity-owned revision without cross-schema joins. The root current-version reference supports revision rotation while a database trigger rejects historical revision mutation; effective intervals, nonblank versions, permissions, and current-version indexes are enforced.
- Existing user and role mutation services re-evaluate through evaluator-backed adapters at their existing authoritative mutation boundary. Existing memory authorizers remain compatible for earlier stories. Evaluated policy version evidence is carried through the existing audit port; identity does not write the audit schema directly.
- Existing IAM HTTP mappings safely distinguish `403` expired/denied, `409` stale, and `503` unavailable outcomes. No public decision endpoint or policy-management UI/API was added.
- `IAM-SCR-05` synthetic UI state shows outcome, policy version, decision reference, matched dimension category, and safe next action for allowed, denied, expired, unavailable, and stale outcomes. `SensitiveDataGuard` remains independent from record access for reveal/export.
- `internal/platform/telemetry` adds bounded authorization result classification only; policy payloads, token claims, identifiers, raw errors, and sensitive values are not recorded.

## Verification log

| Command/check | Result |
|---|---|
| `go test ./internal/identity ./internal/platform/httpapi ./cmd/api` | Passed. |
| `go test ./...` | Passed. |
| `go test -tags=integration ./internal/platform/database` | Passed with PostgreSQL 18.4/Testcontainers, including current-version lookup, stale detection, immutable revision rejection, interval constraints, and existing identity persistence suites. |
| `make db-migrate-validate` | Passed for bootstrap, platform, and identity migrations. |
| `make db-migrate-check` | Passed; the new identity policy migration is included in `db/migrations/checksums.sha256`. |
| `make sqlc-check` | Passed; generated identity SQLC output is current. |
| Direct Windows Node Vitest run for `web/src/app/identity-access-workspace.test.tsx` | Passed: 1 file, 7 tests. |
| Direct Windows Node Vitest run for the web suite | Passed: 14 files, 63 tests. |
| Direct TypeScript build (`tsc -b`) and Vite production build | Passed. |
| Direct Playwright a11y specs (`accessibility`, `keyboard`, `semantic`, `visual-adaptability`) | Passed: 18 tests. |
| `git diff --check` | Passed. |
| Planned `pnpm -C web test` / `pnpm test:a11y` wrappers | Environment-limited: the WSL pnpm launcher cannot find Linux `node`, and Windows pnpm cannot execute the repository’s POSIX Vitest shim. Equivalent direct Windows Node Vitest/Playwright commands passed. |

## Boundary notes

No finance bounded-context behavior, accounting effect, outbox event,
cross-schema write, segregation-rule lifecycle, emergency-access lifecycle,
production Entra/MFA/revocation qualification, penetration testing, or release
qualification is claimed. Policy administration remains planned follow-up
scope. Synthetic policy references and sensitive values are not production
credentials or business records.
