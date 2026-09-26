# DLV-IAM-002 — User Story 3 verification evidence

Status: local implementation complete on branch codex/feat/iam-manage-roles.
Production Workflow approval/segregation adapters, live Entra qualification,
and release qualification remain outside this story.

## Traceability

- Epic: EP-IAM-001
- Story: User Story 3 — Manage roles and permission grants
- Delivery: DLV-FR-IAM-002
- Security/maintainability: NFR-SEC-004, NFR-SEC-012, NFR-MNT-003, NFR-MNT-006
- Quality gates: QG-01, QG-02, QG-03, QG-04, QG-06

## Implemented surfaces

- internal/identity/role.go and role_service.go define versioned role revisions, explicit approved-catalogue grants, opaque scope references, effective-date validation, duplicate detection, actor-scope containment, default-deny authorization, independent approval evidence, segregation evaluation, retirement, audit lineage, idempotency, and expected-version concurrency.
- internal/identity/postgres_role_repository.go, the identity migration/query sources, and generated SQLC output persist role roots, immutable revisions, grant rows, approver evidence, audit references, role assignment ownership, and durable idempotency atomically.
- Runtime composition uses the role repository for user-assignment lookup; unknown and retired roles cannot be newly assigned. Environment role authorization, approval, and segregation dependencies fail closed when not configured.
- The typed manage-roles OpenAPI route accepts create, complete replacement/update, and retirement commands with If-Match consistency, idempotency metadata, approval decision references, and safe typed error outcomes. Generated Go and TypeScript artifacts are checked in.
- IAM-WS-01 and IAM-SCR-02 use existing accessible workspace components for synthetic role worklist/detail, grants, scopes, effective dates, approval/validation states, retirement, masked evidence, duplicate handling, and version-conflict recovery.

## Verification log

| Command/check | Result |
|---|---|
| /usr/local/go/bin/go test ./internal/identity ./internal/platform/httpapi | Passed, including role domain/security and typed HTTP tests. |
| /usr/local/go/bin/go test ./... | Passed. |
| /usr/local/go/bin/go test -tags=integration ./internal/platform/database | Passed after role fixtures and role persistence coverage were added; clean migration, role revisions/grants, atomic updates, audit linkage, retirement, assignment lookup, and durable replay are covered. |
| PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH make sqlc-check | Passed. |
| make db-migrate-inventory && make db-migrate-check | Passed; role migration is included in db/migrations/checksums.sha256. |
| PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH make db-migrate-validate | Passed for bootstrap, platform, and identity migrations. |
| Direct Redocly lint | Passed for contracts/openapi/openapi.yaml. |
| Direct ogen generation | Passed; generated Go artifacts are current for the typed role contract. |
| Direct TypeScript OpenAPI generation | Passed with the installed Windows Node toolchain; generated client types include the typed manage-roles request and If-Match header. |
| Windows Node Vitest run identity-access-workspace | Passed: 1 file and 6 tests. |
| Windows Node TypeScript build (tsc -b) | Passed. |
| pnpm check | Partial: Go tests passed; web step stopped because Windows pnpm could not execute the POSIX vitest shim. The direct Vitest and TypeScript checks passed. |
| make api-check | Wrapper-limited: Windows Redocly received the WSL path as D:\mnt\d\...; direct lint, bundling, generation, and generated-artifact checks were run separately. |
| git diff --check | Passed. |

## Synthetic-data and boundary notes

Approval and segregation are consumed through explicit IAM ports. The local tests use synthetic doubles that validate candidate approval independence and segregation outcomes; the Workflow bounded context and production approval API are not implemented here. No finance posting, access-policy, segregation-rule management, emergency access, or live Entra behavior is claimed.
