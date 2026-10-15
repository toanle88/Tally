# Current status & evidence

This page answers the question the older site did not answer clearly: what is
actually implemented in the current repository, what is a contract or synthetic
fixture, and what remains unverified.

**Checkpoint:** 2026-10-15, branch
`feat/dlv-fr-wfa-003-us4-delegate-approval`.

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
| Runtime | Go API, optional Go worker, React/TypeScript SPA, Vite, Docker Compose PostgreSQL, and implemented GL/workflow runtime slices | No complete end-to-end finance suite is claimed |
| Persistence | Goose migrations, `pgx`, `sqlc`, owned platform, identity, organization, COA, GL, and workflow schemas, seed and checksum checks | PostgreSQL-backed evidence depends on Docker/database availability |
| API | OpenAPI 3.1 catalog, generated Go server artifacts, generated TypeScript client, typed problem outcomes, bearer security contract, and runtime identity/organization/COA/GL/workflow routes | The contract still catalogs future capability routes; it is not proof every route is runtime-backed |
| Finance primitives | Exact-decimal money/currency, accounting scope, aggregate version, serialization and boundary checks | Broader subledger and finance aggregates remain future scope |
| Idempotency | Canonical request fingerprint, scoped metadata, durable coordinator, duplicate/result/conflict behavior used by implemented commands | Future capability commands must continue adopting the shared contracts |
| Integration | Event envelope, transactional outbox/inbox, atomic effects, lease/fencing, typed retry, worker lifecycle, crash recovery, ordering, and replay foundations | No external broker is required; semantic capability consumers are future work |
| UX foundation | Routed shell, scope selector, record/workflow context, worklists, lifecycle, evidence, conflict, result, money, operational components, and authorized GL/workflow workspaces | Many future capability routes remain placeholders or synthetic examples |
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

## Implemented General Ledger and workflow slices

The current checkout extends beyond the IAM foundation:

| Slice | What is present | What is not claimed |
| --- | --- | --- |
| General Ledger | Ledgers, books, charts of accounts, accounts, posting, journal approval, reversal, posting barriers, soft close, scoped reopen, and operational reopen/reclose controls | Complete finance-suite coverage, full cross-context qualification, or production release qualification |
| Workflow policy | Effective-dated policy maintenance, applicability and overlap validation, policy-version evidence, independent approval boundaries, and authorized policy UI/API states | Escalation and full workflow qualification |
| Approval requests and decisions | Effective-policy resolution, immutable subject evidence, copied approval steps, durable idempotency, audit/outbox integration, authorized worklist/detail reads, decision-time authorization, and immutable decision evidence | Consuming-context application acknowledgement, escalation, and full workflow qualification |
| Approval delegation | Immutable finite, scoped policy-version delegation, IAM authorization/SoD boundary, audit/outbox evidence, durable replay/conflict handling, delegated worklist/detail/decision projections, and WFA-SCR-04 | Revoke/end contract, live Entra/production qualification, PostgreSQL/Testcontainers release gates, and full accessibility witnessing |

The [WFA story evidence](https://github.com/toanle88/Tally/blob/main/docs/backlog/stories/EP-WFA-001_user_stories.md)
lists the exact routes, implementation areas, accepted behavior, and deferred
qualification evidence for these slices.

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
and full qualification. The current checkout has organization/master-data, COA,
GL, and initial workflow policy/request slices; most remaining domain
aggregates and later workflow behavior are still represented by specifications
and OpenAPI contracts rather than complete runtime implementations.

The next learning boundaries are hardening and qualifying the delivered GL and
workflow slices, then extending the same vertical-slice discipline to the
remaining finance contexts.

## Known documentation/status caveat

The live `ROADMAP.md` may record a parent epic as locally complete or closed by
owner decision. That is a learning-scope status, not proof of live Entra
behavior, complete finance-action integration, audit-chain ownership, or
production security and release qualification. Check the repository evidence
and the exact command output before treating a capability as qualified.
