# Current status & evidence

This page answers the question the older site did not answer clearly: what is
actually implemented in the current repository, what is a contract or synthetic
fixture, and what remains unverified.

**Checkpoint:** 2026-09-26, branch
`codex/iam-us7-sensitive-access-evidence`.

## How to read the status

There are three different kinds of truth in this project:

1. **Approved target:** the DDD, PRD, UX, NFR, system-design, and technical
   specification documents define the intended finance platform.
2. **Repository reality:** source files, migrations, generated artifacts, and
   route wiring show what this checkout contains.
3. **Verification evidence:** each evidence record states the exact command,
   environment, date, result, and limitation. A passing local check is not a
   production qualification claim.

## Implemented foundation

| Area | Current repository evidence | Boundary |
| --- | --- | --- |
| Runtime | Go API, optional Go worker, React/TypeScript SPA, Vite, Docker Compose PostgreSQL | No complete finance transaction is wired end to end yet |
| Persistence | Goose migrations, `pgx`, `sqlc`, owned platform/identity schemas, seed and checksum checks | PostgreSQL-backed evidence depends on Docker/database availability |
| API | OpenAPI 3.1 catalog, generated Go server artifacts, generated TypeScript client, typed problem outcomes, bearer security contract | The contract catalogs future capability routes; it is not proof every route is runtime-backed |
| Finance primitives | Exact-decimal money/currency, accounting scope, aggregate version, serialization and boundary checks | No GL or subledger aggregate has been delivered |
| Idempotency | Canonical request fingerprint, scoped metadata, durable coordinator, duplicate/result/conflict behavior | Capability commands must still adopt the shared contracts as they are built |
| Integration | Event envelope, transactional outbox/inbox, atomic effects, lease/fencing, typed retry, worker lifecycle, crash recovery, ordering, and replay foundations | No external broker is required; semantic capability consumers are future work |
| UX foundation | Routed shell, scope selector, record/workflow context, worklists, lifecycle, evidence, conflict, result, money, and operational components | Many routes are intentionally placeholders or synthetic examples |
| Accessibility | Automated axe, keyboard/focus, semantic, visual-adaptability, and reduced-motion coverage | Witnessed screen-reader and real 400% browser-zoom qualification remain open |

Representative evidence: [platform verification](https://github.com/toanle88/Tally/tree/main/docs/verification),
[OpenAPI artifacts](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-PLAT-004-contract-drift.md),
[UX verification](https://github.com/toanle88/Tally/tree/main/docs/verification).

## Implemented IAM slice

The current branch contains IAM work through User Story 7:

| Slice | What is present | What is not claimed |
| --- | --- | --- |
| Authenticated application identity | Entra/OIDC discovery and JWT validation boundary, actor construction, PKCE browser client states, fixture mode, fail-closed protected API, anonymous liveness | Live tenant behavior, production MFA/conditional access, penetration testing, production qualification |
| User assignments | User lifecycle, immutable authentication subject, role/scope assignment replacement, idempotency, optimistic concurrency, masking, API/UI evidence | Complete revocation-propagation qualification and all wrapper environments |
| Roles and grants | Versioned roles, explicit grants/scopes, retirement, authorization containment, approval/audit ports, API/UI states | Production workflow and segregation adapters |
| Scoped policies | Operation-specific dimensions, default-deny decisions, policy/version references, safe outcomes, durable policy reads | Full policy administration UI/API and all finance-action wiring |
| Segregation of duties | Versioned IAM-owned rules, fail-closed evaluator, typed administration and explanation states | Enforcement inside future payment, close, vendor, payroll, and GL actions |
| Emergency access | Time-bound grant/revoke lifecycle, four-hour cap, approval and assurance ports, expiry, review metadata, typed API/UI states | Production approval integration and live tenant/security qualification |
| Sensitive access evidence | Metadata-only observation contract, safe decision projections, masked reveal/export guards, permission-filtered export context | Audit bounded-context implementation and public evidence-export API |

Start with the [Identity & Access page](/identity-access) and the individual
[IAM verification records](https://github.com/toanle88/Tally/tree/main/docs/verification).

## Operations and infrastructure

The repository also contains locally verified operational foundations:

- Structured JSON logging with sensitive-field redaction, request/command/
  database/outbox correlation, OpenTelemetry traces, and bounded platform and
  business metrics.
- Health, backlog, error, business-result, alert, runbook, and operational
  readiness contracts.
- Terraform repository boundaries, reusable low-cost modules, environment
  profiles, budget/plan/drift/security checks, GitHub OIDC federation
  contracts, disposable-resource safeguards, and deployment smoke wrappers.

These are learning and release foundations. Azure credentials, live resource
provisioning, production paging, retention, backup/restore, load testing, and
formal recovery qualification are not implied.

See [operations and delivery](/operations) and the [operations evidence
records](https://github.com/toanle88/Tally/tree/main/docs/verification).

## What is still future scope

The approved domain target includes organization/master data, COA, GL, workflow
and period controls, invoicing, AR, AP, payments, bank reconciliation, fixed
assets, revenue, FX, intercompany, reporting, tax, payroll, audit integrity,
and full qualification. Those contexts are represented in the source
specifications and OpenAPI catalog, but their authoritative aggregates and
workflow implementations are not in this checkout.

The recommended next portfolio boundary is M2: a complete journal posting and
reversal slice across domain, persistence, API, UI, authorization, tests,
observability, and recovery evidence.

## Known documentation/status caveat

The live `ROADMAP.md` records EP-IAM-001 as locally complete and intentionally
closed by owner decision on 2026-09-26. This is a local learning-scope closure:
the evidence records and repository code do not claim live Entra behavior,
finance-action integration, audit-chain ownership, or production security and
release qualification.
