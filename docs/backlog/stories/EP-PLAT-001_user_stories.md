# EP-PLAT-001 — Engineering Foundation User Stories

| Field | Value |
|---|---|
| Epic | `EP-PLAT-001` — Engineering foundation |
| Status | Locally complete and intentionally closed on 2026-09-22; hosted CI, branch protection, and requirement-level qualification remain deferred |
| Milestone | `M0` — Engineering foundation |
| Delivery items | `DLV-CI-001`, `DLV-PLAT-001` through `DLV-PLAT-007` |
| Source of detail | The consolidated child sections below retain the delivery-item acceptance criteria and evidence. |

## 1. Outcome

Provide the repository, application shells, local database workflow, API contract
workflow, shared finance primitives, idempotency foundation, integration
transport, workers, and pull-request quality checks required by later finance
capabilities.

## Current closure status

The local M0 engineering-foundation implementation and evidence scope is
complete and intentionally closed in the roadmap. `DLV-PLAT-001` through
`DLV-PLAT-007` have local implementation evidence, and `DLV-CI-001` has its
repository workflow and contract evidence.

Hosted CI positive/negative/drift/skipped-job qualification, external GitHub
branch-protection activation, and broader GFR/workflow qualification remain
explicitly deferred. Those deferrals do not represent missing local platform
implementation and do not close the separate requirement rows under this
epic.

## 2. Learning objective

Learn how to build a local-first modular-monolith foundation while preserving
bounded-context ownership, exact monetary values, immutable financial facts,
safe retries, durable integration, generated-contract consistency, and
reproducible verification.

## 3. Scope

- Go API and React application shells with shared root commands.
- Docker Compose PostgreSQL, Goose migrations, pgx, and sqlc.
- OpenAPI-first REST contracts and generated Go/TypeScript artifacts.
- Exact-decimal money, currency, accounting-scope, identity, and version
  primitives.
- Request fingerprinting, idempotency coordination, PostgreSQL outbox/inbox,
  and worker lifecycle behavior.
- Pull-request checks for API, frontend, contracts, persistence, Terraform,
  security, and documentation quality.

## 4. Explicit exclusions

- Finance-domain aggregates, commands, workflows, or authoritative financial
  records owned by later capability epics.
- Production cloud deployment or production qualification.
- Replacing bounded-context ownership with shared business repositories or
  cross-schema writes.
- Treating generated artifacts, telemetry, or integration delivery as a
  substitute for domain acceptance evidence.

## 5. Delivery-item map

### `DLV-CI-001` — Pull-request CI quality pipeline

**Status:** Locally complete and intentionally closed — aggregate workflow and
repository contract implemented; hosted CI qualification pending.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Establish the pull-request pipeline contract
- User Story 2 — Gate API, worker, and frontend quality
- User Story 3 — Gate contract, persistence, and generated artifacts
- User Story 4 — Gate infrastructure, security, and documentation quality
- User Story 5 — Publish safe evidence and enforce the merge gate

### `DLV-PLAT-001` — Monorepo foundation

**Status:** Complete.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Establish the monorepo structure
- User Story 2 — Provide a minimal Go API shell
- User Story 3 — Provide a minimal React application shell
- User Story 4 — Provide shared root commands
- User Story 5 — Document and prove clean-clone reproducibility

### `DLV-PLAT-002` — Docker Compose PostgreSQL development environment

**Status:** Complete.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Define safe local database configuration
- User Story 2 — Start and health-check PostgreSQL
- User Story 3 — Provide root database lifecycle commands
- User Story 4 — Orchestrate migration and deterministic seeding
- User Story 5 — Reset and prove reproducibility

### `DLV-PLAT-003` — Goose migrations, pgx, and sqlc workflow

**Status:** Complete.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Establish the Goose migration contract
- User Story 2 — Establish the pgx database foundation
- User Story 3 — Establish the sqlc generation workflow
- User Story 4 — Prove migrations, pgx, and sqlc together
- User Story 5 — Detect persistence drift in CI

### `DLV-PLAT-004` — OpenAPI-first REST workflow and generated clients

**Status:** Complete.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Establish the contract layout and common schemas
- User Story 2 — Validate and bundle the OpenAPI contract
- User Story 3 — Generate and verify Go API artifacts
- User Story 4 — Generate and verify the TypeScript client
- User Story 5 — Detect contract and generated-artifact drift

### `DLV-PLAT-005` — Shared finance primitives

**Status:** Complete.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Implement exact-decimal money and currency primitives
- User Story 2 — Implement explicit accounting-scope identity
- User Story 3 — Implement stable identity primitives
- User Story 4 — Implement aggregate version primitives
- User Story 5 — Prove serialization and boundary behavior

### `DLV-PLAT-006` — Request fingerprint and idempotency foundation

**Status:** Complete for the accepted platform scope; environment-dependent
persistence qualification remains.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Define canonical request fingerprinting
- User Story 2 — Define scoped idempotency identity and stored command-result metadata
- User Story 3 — Coordinate established results for identical retries
- User Story 4 — Reject changed content at the platform boundary
- User Story 5 — Prove transactional, concurrent, and boundary behavior

### `DLV-PLAT-007` — PostgreSQL outbox/inbox and worker foundation

**Status:** Complete for the platform foundation; semantic payload safety is a
separate deferred follow-up.

Detailed story content is consolidated in Section 9 below.

- User Story 1 — Define the versioned event envelope and safe payload contract
- User Story 2 — Persist durable PostgreSQL outbox and inbox records
- User Story 3 — Coordinate transactional publication and consumption
- User Story 4 — Dispatch due outbox work with leases and typed retries
- User Story 5 — Prove worker lifecycle, crash recovery, duplicate delivery, and replay

## 6. Dependencies and handoffs

- `DLV-PLAT-001` supplies the repository and root command foundation.
- `DLV-PLAT-002` and `DLV-PLAT-003` supply the local persistence workflow.
- `DLV-PLAT-004` supplies the API contract and generated-artifact workflow.
- `DLV-PLAT-005` supplies shared value and identity primitives.
- `DLV-PLAT-006` and `DLV-PLAT-007` supply retry, integration, and worker
  boundaries for later bounded contexts.
- `DLV-CI-001` turns the focused checks into the repository merge gate.

## 7. Definition of done

- Every child delivery item has its required acceptance criteria and evidence
  reviewed.
- The local foundation is reproducible from a clean checkout.
- Generated contracts and persistence artifacts have drift detection.
- Retry, concurrency, integration, and worker failure behavior is explicit.
- No finance capability is marked complete by foundation work alone.

## 8. Source references

- [Finance Platform Delivery Plan](../../specs/finance_delivery_plan_v1.0.md)
- [Solution Architecture Overview](../../specs/system_design/01_solution_architecture_overview_v1.0.md)
- [Backend Module Technical Specifications](../../specs/technical_specifications/01_backend_module_specifications_v1.0.md)
- [API and OpenAPI Technical Specifications](../../specs/technical_specifications/02_api_openapi_specifications_v1.0.md)
- [Database and Persistence Technical Specifications](../../specs/technical_specifications/03_database_persistence_specifications_v1.0.md)
- [Events, Workers, and Integration Technical Specifications](../../specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md)

## 9. Consolidated detailed delivery-item stories

### DLV-CI-001 — Pull-request CI Quality Pipeline User Stories


| Field | Value |
|---|---|
| Status | Implemented — hosted CI qualification pending |
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Delivery item | `DLV-CI-001` |
| Deliverable | Create a pull-request CI quality pipeline. |
| Exit evidence | Go, frontend, OpenAPI, SQL, Terraform, security, and documentation checks gate merge. |

#### 1. Outcome

Provide one repository-owned pull-request quality pipeline that runs the
existing focused verification commands, fails closed on quality regressions,
and exposes stable, reviewable evidence without requiring production data,
Azure credentials, or live infrastructure.

The pipeline orchestrates checks owned by their existing delivery items. It
does not move application, database, Terraform, security, or documentation
rules into a new shared module.

#### 2. Learning objective

Learn how to compose a safe CI quality gate from repository-controlled commands,
dependency lockfiles, service containers, generated-artifact checks, and
credential-free infrastructure verification. Preserve a clear boundary between
pull-request validation, protected deployment, smoke testing, and later release
qualification.

#### 3. Scope and ownership

`DLV-CI-001` owns pull-request orchestration, job dependencies, failure
propagation, safe evidence publication, and the documented required-check
contract. The delivery item does not own the rules tested by each focused gate.

Current repository evidence includes focused workflows for OpenAPI, persistence,
Terraform, and documentation, plus root commands for Go, frontend, generated
artifacts, database, infrastructure, and security checks. The missing scope is
the complete pull-request pipeline that composes those checks consistently.

##### Delivery-item boundaries

| Concern | Owner | CI responsibility |
|---|---|---|
| Go API, worker, and platform tests | `DLV-PLAT-001` through `DLV-PLAT-007` | Invoke the approved root commands and preserve their failure status. |
| Frontend tests and build | `DLV-PLAT-001`, `DLV-UX-001`, `DLV-UX-002` | Install from the committed lockfile and run the approved test/build commands. |
| OpenAPI and generated artifacts | `DLV-PLAT-004` | Invoke `make api-check`; do not duplicate generator logic in workflow YAML. |
| Migrations, sqlc, and persistence | `DLV-PLAT-002`, `DLV-PLAT-003` | Provide the PostgreSQL runner/service and invoke the approved persistence gate. |
| Terraform and infrastructure safety | `DLV-IAC-001`, `DLV-IAC-002` repository checks | Run credential-free checks only; do not apply, destroy, or qualify Azure. |
| Security and secret hygiene | Applicable security/NFR owners | Run the repository-approved scanners and prevent secret leakage in logs/artifacts. |
| Documentation | Repository documentation owners | Run the documentation build/check and publish only safe failure/success summaries. |

#### 4. Explicit exclusions

- No finance capability, domain aggregate, command, query, API route, database
  table, migration, event, or frontend screen.
- No replacement of the focused checks owned by `DLV-PLAT-*`, `DLV-UX-*`, or
  `DLV-IAC-*`.
- No Azure login, remote-state access, Terraform apply, Terraform destroy,
  smoke test, or live Azure qualification in pull-request CI.
- No production credentials, personal data, finance data, Terraform state,
  plans, `.tfvars`, connection strings, or raw provider error payloads in CI
  output or artifacts.
- No claim that configured GitHub branch protection is active unless the
  external repository setting is separately verified.
- No production release, deployment, rollback, backup/restore, disaster
  recovery, performance, or full-system qualification evidence.
- No new CI framework or package manager; pnpm remains the sole frontend
  package manager.

#### 5. Owning bounded context & architecture

- **Owning Bounded Context / Module:** Repository/platform delivery tooling; no
  finance bounded context owns CI orchestration.
- **Domain Aggregates & Invariants:** None. CI must preserve existing module
  ownership, immutable financial facts, exact-decimal money, idempotency,
  transactional outbox, and authorization/audit boundaries by not bypassing
  the commands and tests that enforce them.
- **Workflow ownership:** Pull-request validation belongs to `DLV-CI-001`.
  Protected apply belongs to the existing deployment workflow and remains
  separate from this delivery item.

#### 6. Contract & impact analysis

- **Application Commands / Queries:** None added. CI invokes existing commands,
  Make targets, package scripts, and verification wrappers.
- **API Impact (Routes / Schemas):** None. OpenAPI validation is consumed as a
  gate only.
- **Database Impact (Schemas / Tables):** None. A disposable PostgreSQL service
  or approved Testcontainers path may be used for verification; no committed
  schema or data change belongs here.
- **Event or Worker Impact:** None. Existing outbox/inbox and worker tests run
  through their owning commands.
- **Authorization Impact (Permissions / Scope):** CI workflow permissions must
  be least-privilege and must not expose deployment or cloud credentials to
  untrusted pull-request code.
- **Frontend Impact (Routes / Components):** None. Frontend tests and builds
  validate the existing React application.
- **Observability Impact (Logs / Traces / Metrics):** CI summaries must identify
  the job, commit, command, result, and safe failure context without secrets or
  unrestricted diagnostic payloads.

#### 7. Failure, concurrency & idempotency rules

- **Expected Failure Cases:** Dependency installation failure, tool-version
  mismatch, compilation/test failure, generated-artifact drift, migration or
  sqlc drift, invalid OpenAPI contract, Terraform validation/policy/security
  failure, documentation build failure, missing service dependency, and secret
  or forbidden-artifact detection failure.
- **Failure propagation:** A failed required command returns non-zero and fails
  its job. Wrapper output must not convert a failed check into a successful
  status or continue past a required failure without recording it.
- **Concurrency:** A pull-request run may be superseded by a newer run according
  to the reviewed workflow policy, but cancellation must not leave required
  checks falsely successful or mutate shared finance/cloud state.
- **Idempotency:** Re-running the same commit must be safe and produce the same
  repository verification result. CI may use disposable service state, but it
  must not depend on an already-mutated shared database or Azure environment.
- **Secret safety:** Pull-request jobs use no Azure or production credentials;
  logs and artifacts are checked for secrets, state, plans, connection strings,
  and raw sensitive payloads.
- **External failures:** Missing Docker or unavailable external services must be
  reported as an explicit blocked/environment result during local diagnosis;
  the hosted required pipeline must provision the dependencies it claims to
  require.

#### 8. Traceability identifiers

| Traceability field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-CI-001` |
| Direct functional requirement IDs | None; this is delivery tooling, not finance behavior. |
| Direct global requirement IDs | None; the pipeline verifies controls owned by other delivery items. |
| NFR contributions | `NFR-MNT-001`, `NFR-MNT-010`, `NFR-SEC-010`, `NFR-OBS-004`, `NFR-TST-003`, `NFR-TST-006`, `NFR-TST-009` as applicable. |
| Quality-gate contribution | `QG-01`, `QG-03`, `QG-04`, `QG-05`, and `QG-10`; security checks support `QG-06` but do not complete M1 security qualification. |
| Workflow IDs | None; pull-request orchestration is a delivery control, not a finance workflow. |

#### 9. Source references

- `docs/specs/finance_delivery_plan_v1.0.md` — `DLV-CI-001` deliverable,
  exit evidence, quality gates, test layers, and release evidence.
- `docs/specs/system_design/04_security_deployment_operations_v1.0.md` — CI/CD
  pipeline, protected apply, secret, deployment, and observability boundaries.
- `docs/specs/finance_nonfunctional_requirements_v1.0.md` — traceability,
  secret safety, operational diagnostics, testing, and release evidence.
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md`
  — verification layers and evidence expectations.
- `docs/backlog/stories/EP-PLAT-001_user_stories.md` — repository and root
  command boundary.
- `docs/backlog/stories/EP-PLAT-001_user_stories.md` — local PostgreSQL and
  persistence-environment boundary.
- `docs/backlog/stories/EP-PLAT-001_user_stories.md` — focused persistence CI
  boundary and explicit ownership by `DLV-CI-001`.
- `docs/backlog/stories/DLV-IAC-001_user_stories.md` and
  `docs/backlog/stories/EP-IAC-001_user_stories.md` — credential-free Terraform
  checks and closed/deferred Azure qualification boundary.
- `Makefile`, `package.json`, `web/package.json`, and `.github/workflows/` —
  current commands and focused workflow evidence.

#### 10. Dependencies

- `DLV-PLAT-001` root commands, Go/React shell, and clean-clone conventions.
- `DLV-PLAT-002` PostgreSQL lifecycle and disposable database verification.
- `DLV-PLAT-003` migration, pgx, sqlc, checksum, and persistence gate.
- `DLV-PLAT-004` OpenAPI and generated-artifact gate.
- `DLV-PLAT-005` through `DLV-PLAT-007` focused platform verification commands.
- `DLV-UX-001` and `DLV-UX-002` frontend/component/accessibility checks.
- Credential-free repository checks from `DLV-IAC-001`; EP-IAC-001 closure means
  live Azure exercises are not a dependency.
- Hosted CI support for the declared Go, Node.js, pnpm, Docker/PostgreSQL,
  Terraform, Python, TFLint, Checkov, and Infracost tool paths.

#### 11. User stories

##### User Story 1 — Establish the pull-request pipeline contract

**As a repository maintainer, I want a single documented pull-request CI
contract, so that every change receives the same required quality checks.**

- [ ] The workflow is clearly identified as pull-request quality validation and
  does not include protected deployment or live Azure operations.
- [ ] Trigger, permissions, job names, dependency order, cancellation behavior,
  and required status-check names are documented.
- [ ] Tool versions come from repository-controlled manifests or reviewed setup
  actions; dependency installation uses the committed lockfiles.
- [ ] The workflow has no secret or cloud-credential requirement for ordinary
  pull requests, including pull requests from untrusted branches.
- [ ] Existing focused workflows are reused or intentionally superseded without
  running the same expensive gate ambiguously or silently dropping coverage.

##### User Story 2 — Gate API, worker, and frontend quality

**As a contributor, I want application and frontend checks to run on every
pull request, so that compile, test, and build regressions fail before merge.**

- [ ] Go API, worker, and platform tests run through the approved root command
  and return a non-zero result on failure.
- [ ] The API and frontend production builds run from repository-controlled
  commands and return a non-zero result on failure.
- [ ] Frontend dependencies install with the committed `web/pnpm-lock.yaml`;
  no alternate frontend package manager is introduced.
- [ ] The check covers the existing accessibility and component test commands
  at the scope owned by `DLV-UX-001` and `DLV-UX-002` without claiming manual
  screen-reader or release qualification.
- [ ] A clean reviewed commit produces a passing result with no generated,
  dependency, build, test, or local-environment artifacts left in the tree.

##### User Story 3 — Gate contract, persistence, and generated artifacts

**As a platform maintainer, I want contract and persistence drift checks in the
same pull-request quality path, so that source, generated output, and database
verification cannot diverge silently.**

- [ ] OpenAPI validation, Go generation, TypeScript generation, artifact
  inventory, and generated-output drift are checked through `make api-check`.
- [ ] Migration validation/checksum and sqlc drift checks run through the
  approved persistence commands.
- [ ] The PostgreSQL 18 persistence gate runs with a disposable service or the
  approved Testcontainers path and fails on migration, compile, integration, or
  drift failure.
- [ ] Focused persistence, OpenAPI, and generated-artifact workflows remain
  traceable to their owning delivery items.
- [ ] Temporary negative evidence proves at least one representative contract,
  migration, or generated-artifact mutation fails the relevant gate and is
  restored before the final verification.

##### User Story 4 — Gate infrastructure, security, and documentation quality

**As a reviewer, I want credential-free infrastructure, security, and
documentation checks in pull-request CI, so that unsafe repository changes are
blocked without touching Azure or production data.**

- [ ] Terraform formatting, validation, environment/module contracts, policy,
  security, lock/tool checks, and repository CI contracts run through the
  approved credential-free commands.
- [ ] Terraform checks initialize without remote-state access and never run
  Azure apply, destroy, smoke, or authenticated drift operations in PR CI.
- [ ] Secret/forbidden-artifact checks fail when credentials, state, plans,
  `.tfvars`, raw sensitive output, or generated local artifacts are introduced.
- [ ] Documentation builds through the repository's approved `pnpm docs:check`
  or equivalent documented command and fails on broken documentation output.
- [ ] Security and infrastructure output is redacted and does not expose
  provider credentials, connection strings, raw state, or unbounded error data.

##### User Story 5 — Publish safe evidence and enforce the merge gate

**As a reviewer and repository owner, I want stable, safe CI evidence and
required status checks, so that a green pull request means the declared gates
actually passed.**

- [ ] Job summaries identify commit, tool/runtime versions, executed command,
  result, and a safe failure location without including secrets or raw state.
- [ ] Required checks fail closed when a dependency job fails, a command is
  skipped unexpectedly, or a required evidence artifact is missing.
- [ ] The workflow does not report success merely because a focused job was
  skipped due to path filters, missing services, or an unavailable tool.
- [ ] The documented status-check contract is suitable for repository branch
  protection; any external branch-protection setting is recorded separately
  rather than assumed.
- [ ] A verification record links the workflow revision, representative passing
  run, representative negative run, and known deferred qualification scope.

#### 12. Definition of Ready

- [x] The milestone, parent epic, delivery item, deliverable, and exit evidence
  are identified in the approved delivery plan.
- [x] Existing focused workflows and root verification commands have been
  inventoried.
- [x] Boundaries with platform, UX, infrastructure, protected apply, and later
  release qualification work are documented.
- [x] The repository uses pnpm as the sole frontend package manager and commits
  the relevant lockfiles.
- [x] The required-check name and hosted runner/service strategy are selected;
  external branch-protection configuration remains separately unverified.
- [x] The final workflow fan-in and duplicate-work policy are approved.
- [x] The representative clean and controlled-negative evidence location is
  `docs/verification/DLV-CI-001-pull-request-quality-pipeline.md`.

#### 13. Definition of Done

- [ ] All five user stories pass their acceptance criteria with evidence.
- [ ] The pull-request workflow runs the declared Go, frontend, OpenAPI, SQL,
  Terraform, security, and documentation checks.
- [ ] Required commands fail the relevant job and the aggregate required check
  fails closed.
- [ ] Tool setup and dependency installation use repository-controlled versions
  and committed lockfiles.
- [ ] CI uses no production or Azure credentials for pull-request validation
  and performs no live infrastructure mutation.
- [ ] Logs, summaries, and artifacts contain no secrets, raw Terraform state,
  plans, connection strings, or sensitive application data.
- [ ] Focused workflows remain owned by their delivery items and no command or
  quality gate is silently duplicated or omitted.
- [ ] A clean reviewed run, representative failure run, and generated-output or
  migration drift run are recorded.
- [ ] Documentation identifies what the pipeline verifies and what it does not
  qualify, including live Azure, production, recovery, and full-system claims.
- [ ] The implementation diff preserves modular-monolith ownership and does not
  mark `EP-PLAT-001`, `EP-OPS-001`, `M0`, or any release quality gate complete
  without its own evidence.
- [ ] No critical or high unresolved CI, security, or repository-integrity defect
  remains.

#### 14. Suggested implementation steps

1. Confirm the required-check contract, runner/tool versions, service-container
   strategy, and branch-protection boundary.
2. Map existing focused workflows and root commands into a minimal job graph;
   remove or preserve duplicate workflows deliberately.
3. Add the aggregate pull-request workflow with least-privilege permissions,
   lockfile installation, dependency fan-in, and fail-closed behavior.
4. Add the application, contract, persistence, infrastructure, security, and
   documentation gates using existing commands rather than reimplementing them
   in YAML.
5. Add safe summaries and controlled negative checks for command failure,
   generated drift, secret/forbidden artifacts, and skipped-job behavior.
6. Run the clean positive and negative verification, record evidence, and update
   the roadmap/status only for criteria actually verified.

#### 15. Required test evidence

- [ ] Workflow syntax and action/setup configuration validation.
- [ ] Clean pull-request run on a reviewed commit.
- [ ] Go test/build failure propagation.
- [ ] Frontend frozen-install, test, and build failure propagation.
- [ ] OpenAPI/generated-artifact drift failure and restoration.
- [ ] Migration/sqlc/persistence drift failure and restoration.
- [ ] Terraform policy/security/forbidden-artifact failure and restoration.
- [ ] Documentation build failure propagation.
- [ ] Secret/state/plan/raw-error redaction or forbidden-artifact negative checks.
- [ ] Evidence that an unexpectedly skipped required job cannot produce an
  aggregate success.

#### 16. Likely files / packages involved

- `.github/workflows/` — aggregate pull-request workflow and focused workflow
  coordination.
- `Makefile` — existing database, API, Terraform, and verification entry points.
- `package.json` and `web/package.json` — root/frontend install, test, build,
  and documentation commands.
- `pnpm-lock.yaml` and `web/pnpm-lock.yaml` — frozen dependency inputs.
- `scripts/verify/`, `scripts/openapi/`, `scripts/db/`, and `scripts/deploy/` —
  existing verification wrappers and their self-tests.
- `infra/terraform/` — credential-free Terraform roots, modules, tests, and
  provider lockfiles.
- `docs/verification/` — final DLV-CI-001 evidence record.

#### 17. Risks or open questions

- The delivery plan defines `DLV-CI-001`; the roadmap and this story record the
  repository implementation while keeping hosted qualification evidence
  explicitly pending.
- The repository has focused workflows already. Combining them may create
  duplicate runs or inconsistent required status names unless the fan-in policy
  is decided first.
- Persistence checks require Docker/PostgreSQL support; the hosted runner
  strategy must be explicit and reproducible.
- Branch-protection settings are external repository state and require separate
  verification; workflow configuration alone cannot prove that merges are
  blocked.
- The closed EP-IAC-001 does not remove the need for credential-free Terraform
  contract checks, but it does remove live Azure deployment and qualification
  from this delivery item's dependency path.
- The aggregate workflow and repository-owned CI contract now exist on the
  implementation branch, but no acceptance criterion is marked complete until
  hosted positive, negative, drift, and skipped-job evidence is recorded.

#### 18. Implementation status

Repository implementation complete on branch
`codex/dlv-ci-001-pr-quality-pipeline`:

- `.github/workflows/pull-request-quality.yml` provides six quality jobs and
  the fail-closed `Pull-request quality / required` fan-in job.
- Focused OpenAPI, persistence, Terraform, and documentation workflows retain
  push/manual verification without duplicate pull-request triggers.
- `make ci-check`, `make repository-integrity-check`, and
  `make terraform-pr-check` expose the repository-owned CI contracts.
- `scripts/verify/ci-contract.js` and `scripts/verify/repository-integrity.sh`
  provide workflow, static-safety, and forbidden-artifact checks; Terraform
  provider-backed planning remains outside credential-free PR scope.
- Implementation evidence is recorded in
  `docs/verification/DLV-CI-001-pull-request-quality-pipeline.md`.

Hosted CI evidence remains required for the clean run, command failure
propagation, generated/migration drift, scanner execution, documentation
failure propagation, and aggregate skipped-job behavior. Authenticated
Terraform provider planning is deferred to protected Azure qualification.
External GitHub branch-protection activation remains unverified.

### DLV-PLAT-001 — Monorepo Foundation User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-001` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Artifact version | 1.2 |
| Review status | Passed — three review passes completed; pnpm revision consistency-reviewed |
| Delivery profile | Solo, part-time, local-first learning project |
| Dependency position | No predecessor is identified for this item; its parent epic has no epic dependency. |
| Authoritative deliverable | Create monorepo with Go API, React application and shared commands. |
| Authoritative exit evidence | Clean clone builds and tests locally. |

#### 1. Purpose and scope

Break `DLV-PLAT-001` into small, reviewable stories without absorbing work assigned to adjacent M0 delivery items.

The completed delivery item provides:

- one repository containing the initial Go API and React application;
- a minimal Go API shell aligned with the selected HTTP baseline;
- a minimal React, TypeScript, and Vite application shell;
- one documented root command surface for both applications; and
- repeatable clean-clone build and test evidence.

This is an engineering foundation item, not product behavior. It completes no `FR-*`, `GFR-*`, `WF-*`, or `NFR-*` delivery item and does not complete M0 by itself.

#### 2. Authoritative implementation baseline

| Area | Baseline used by these stories |
|---|---|
| Architecture | One-repository modular monolith; bounded-context boundaries remain explicit. |
| Backend | Go `1.26.x`; `net/http` with `chi`; composition root at `cmd/api`. |
| Frontend | Node.js `24 LTS`; React `19.2`; TypeScript; Vite `8`. |
| Frontend package manager | pnpm only; commit `web/pnpm-lock.yaml`, pin the selected exact pnpm version through the `packageManager` field in `web/package.json`, and use frozen-lockfile installation for reproducibility. |
| Frontend tests | Vitest and Testing Library for the application-shell test. MSW is not required until an API interaction needs it. |
| Repository boundary | Reusable technical facilities belong under `internal/platform`; no finance bounded-context implementation is created by this item. |

Patch versions and lockfile-resolved dependency versions are implementation evidence. The pnpm selection is authorized by `ADR-021`; the exact pnpm version remains an implementation pin recorded in `web/package.json`.

#### 3. Required implementation inputs

The item is not ready to start until these local implementation inputs are resolved:

- canonical repository URL and Go module path;
- root command mechanism, using the simplest option that does not add an unnecessary runtime or broad task-runner dependency;
- exact pnpm version to pin in `web/package.json`; and
- frontend development-port behavior, using checked-in configuration when the Vite default is changed. The API uses the approved `HTTP_ADDR` default `:8080`.

Do not invent a placeholder Go module path such as `example.com/...`.

#### 4. Cross-story constraints

1. Preserve the approved modular-monolith direction and one-repository structure.
2. `cmd/api` is a composition root; reusable HTTP behavior belongs under `internal/platform`.
3. Shared platform code contains technical behavior only and owns no finance business facts.
4. Do not create bounded-context implementations such as `internal/organization`, `internal/gl`, `internal/ap`, or `internal/ar`.
5. Do not add database schemas, tables, migrations, SQL, generated SQL, outbox, inbox, or worker behavior.
6. Do not add OpenAPI contracts, authentication, authorization, idempotency, observability instrumentation, Terraform, Azure, or CI implementation.
7. Do not add a general ORM, external message broker, cache, cloud SDK, or global frontend state framework.
8. Do not add placeholder money types or floating-point monetary examples.
9. Do not represent Tailwind, daisyUI abstractions, routing, accessibility qualification, or finance workflows as completed.
10. Use synthetic, non-sensitive text only and commit no credentials or local environment secrets.
11. Do not create empty future directories merely to imitate the final repository tree.

---

#### 5. User Story 1 — Establish the monorepo structure

**As the TALLY developer, I want a stable repository structure for the Go API and React application, so that later delivery items can extend the platform without reorganizing or violating module boundaries.**

##### Value

Creates the physical foundation for the modular monolith and frontend without prematurely implementing business modules or adjacent platform capabilities.

##### Acceptance criteria

- [x] The repository root contains `cmd/`, `internal/`, and `web/`.
- [x] The Go API entry point is located at `cmd/api/main.go`.
- [x] Reusable API-shell HTTP behavior is located under `internal/platform/httpx/` or an equivalently narrow technical package under `internal/platform/`.
- [x] The frontend application is located directly under `/web`; it is not nested inside a backend-only parent directory.
- [x] The root contains `README.md`, `.gitignore`, `go.mod`, and the selected root command manifest.
- [x] `go.mod` uses the confirmed canonical module path and a Go directive compatible with the approved Go `1.26.x` baseline.
- [x] `go.sum` is committed when the selected Go dependencies generate it.
- [x] pnpm is the sole frontend package manager and `web/pnpm-lock.yaml` is committed.
- [x] `web/package.json` pins the selected exact pnpm version through its `packageManager` field.
- [x] No npm, Yarn, or other frontend lockfile is committed.
- [x] Local environment files, dependency directories, frontend build output, test output, and compiled binaries are ignored.
- [x] No finance bounded-context package, database artifact, event contract, generated API contract, or cloud resource is introduced.
- [x] Future repository areas remain absent until their owning delivery items require them.

##### Expected minimum shape

```text
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   └── platform/
│       └── httpx/
├── web/
│   ├── package.json
│   └── pnpm-lock.yaml
├── .gitignore
├── go.mod
├── README.md
└── <root command manifest>
```

##### Evidence

- Repository tree from the implementation commit.
- Confirmed repository URL and Go module path.
- Output of `go env GOMOD` from the repository root.
- Diff review confirming that excluded capability and persistence code was not added.

---

#### 6. User Story 2 — Provide a minimal Go API shell

**As the TALLY developer, I want a small, testable Go HTTP application, so that the backend process can be built and exercised before finance behavior is introduced.**

##### Value

Establishes the API process boundary, selected router, composition root, and first deterministic backend feedback loop.

##### Acceptance criteria

- [x] `go build ./cmd/api` succeeds.
- [x] The API uses `net/http` with `chi`, without introducing a broader web framework.
- [x] `GET /health/live` returns HTTP `200 OK` and valid JSON.
- [x] The liveness response checks process liveness only and does not claim database or dependency readiness.
- [x] The liveness response exposes no secrets, environment contents, stack traces, or internal topology.
- [x] The liveness handler is implemented as reusable technical behavior under `internal/platform/`.
- [x] A deterministic Go test verifies the liveness handler status, content type, and response shape.
- [x] Invalid startup configuration and listen failures produce an explicit non-successful process outcome.
- [x] The API handles operating-system termination signals and attempts graceful shutdown with a bounded timeout.
- [x] The API starts and its tests pass without PostgreSQL, Docker, Azure, authentication, or external services.
- [x] `go test ./...` succeeds.

`/health/ready` is not required by this item because database and migration compatibility do not yet exist. A later item must implement readiness without changing the approved liveness semantics.

##### Likely files

```text
cmd/api/main.go
internal/platform/httpx/health.go
internal/platform/httpx/health_test.go
```

Exact file names may vary, but the composition-root and reusable-platform boundary must remain clear.

##### Evidence

- Output from `go build ./cmd/api`.
- Output from `go test ./...`.
- Local request and response for `GET /health/live`.
- Shutdown verification note identifying the signal and observed bounded shutdown outcome.

---

#### 7. User Story 3 — Provide a minimal React application shell

**As the TALLY developer, I want a minimal React, TypeScript, and Vite application under `/web`, so that frontend work begins from the approved stack and can later grow by capability without restructuring.**

##### Value

Creates the frontend runtime and first deterministic component test while leaving shared finance UX and capability behavior to their owning delivery items.

##### Acceptance criteria

- [x] The frontend uses the approved Node.js `24 LTS`, React `19.2`, TypeScript, and Vite `8` baselines.
- [x] `pnpm install --frozen-lockfile` succeeds inside `/web` using the committed lockfile.
- [x] The application renders a minimal shell that identifies the product as TALLY.
- [x] Implemented source areas include `web/src/app/` and `web/src/test/`.
- [x] The structure remains compatible with later `routes`, `components`, `capabilities`, and `lib` areas without adding empty or speculative finance implementations.
- [x] A deterministic Vitest and Testing Library test proves that the application shell mounts and displays the TALLY identity.
- [x] The frontend test runs once in non-watch mode and exits with the correct process status.
- [x] `pnpm build` succeeds with no TypeScript compilation errors.
- [x] Unused Vite sample assets and demonstration code are removed.
- [x] Tailwind, daisyUI wrappers, TanStack Query, React Hook Form, Zod, TanStack Table, routing, and business-capability screens are not represented as completed.

##### Likely files

```text
web/package.json
web/pnpm-lock.yaml
web/index.html
web/tsconfig.json
web/vite.config.ts
web/src/main.tsx
web/src/app/app.tsx
web/src/app/app.test.tsx
web/src/test/setup.ts
```

##### Evidence

- Output from `pnpm install --frozen-lockfile`.
- Output from the frontend non-watch test command.
- Output from `pnpm build`.
- Local verification note or screenshot showing the TALLY application shell.

---

#### 8. User Story 4 — Provide shared root commands

**As the TALLY developer, I want one documented command surface at the repository root, so that I can build, test, verify, and run both applications consistently.**

##### Value

Creates a simple engineering feedback loop and prevents one application from being silently omitted from local verification.

##### Required command intents

The command names are an implementation choice. The names below are recommended aliases, not new product contracts.

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Build all | `build` | Build the Go API and production frontend bundle. |
| Test all | `test` | Run all Go tests and frontend tests once. |
| Verify all | `check` | Run the complete local build-and-test verification. |
| Run API | `dev-api` | Start the local Go API. |
| Run frontend | `dev-web` | Start the Vite development server. |

##### Acceptance criteria

- [x] Every required command intent can be invoked from the repository root.
- [x] The build-all command fails when either the Go build or frontend build fails.
- [x] The test-all command fails when either the Go tests or frontend tests fail.
- [x] The verify-all command runs the complete build-and-test sequence and returns a non-zero status when any required step fails.
- [x] No command reports success when the Go or frontend portion was skipped.
- [x] The API and frontend development commands start their applications independently.
- [x] The command mechanism does not add an unnecessary runtime or broad task-runner dependency.
- [x] Command names, prerequisites, working directory, and expected behavior are documented in `README.md`.
- [x] A controlled negative check proves that the verify-all command propagates a child failure.

##### Evidence

- Output from the root build-all, test-all, and verify-all commands.
- Controlled negative-check output showing non-zero failure propagation.
- Local startup notes for the API and frontend commands.

---

#### 9. User Story 5 — Document and prove clean-clone reproducibility

**As the TALLY developer, I want clean-clone setup instructions and recorded verification, so that the repository foundation is reproducible rather than dependent on untracked local state.**

##### Value

Directly proves the authoritative exit evidence: a clean clone builds and tests locally.

##### Documentation acceptance criteria

- [x] `README.md` lists the required Go and Node.js major versions and the exact pinned pnpm version.
- [x] `README.md` documents clean-clone setup and frontend dependency installation.
- [x] The root command intents and their selected names are documented.
- [x] Local API and frontend startup are documented.
- [x] The API default `HTTP_ADDR` of `:8080` and the configured frontend development port behavior are documented.
- [x] `GET /health/live` and its limited liveness meaning are documented.
- [x] Commands for all Go and frontend tests are documented.
- [x] Story boundaries and explicit exclusions are documented.
- [x] PostgreSQL, Docker, migrations, OpenAPI, authentication, Azure, observability instrumentation, CI, and finance modules are not described as completed.

##### Clean-clone verification criteria

- [x] A new clone can install frontend dependencies using `pnpm install --frozen-lockfile`.
- [x] The selected root verify-all command succeeds from the new clone.
- [x] The Go API can be started and its liveness endpoint returns HTTP `200`.
- [x] The React application can be started and displays the TALLY shell.
- [x] Verification does not require uncommitted source, pre-existing `node_modules`, pre-existing compiled binaries, credentials, a database, or a cloud resource.
- [x] Build caches may accelerate verification but are not required for success.
- [x] The evidence record includes commit identifier, operating system, Go version, Node.js version, pnpm version, commands executed, and final result.

##### Evidence record template

```markdown
## DLV-PLAT-001 clean-clone verification

- Commit:
- Operating system:
- Go version:
- Node.js version:
- pnpm version:
- Root command mechanism:
- Commands executed:
  1.
  2.
  3.
- API liveness result:
- Frontend shell result:
- Overall result: PASS | FAIL
- Notes:
```

---

#### 10. Delivery-item acceptance summary

`DLV-PLAT-001` is complete only when all five stories are complete and every condition below passes:

- [x] The clean repository contains a minimal Go API and React application.
- [x] Both applications build successfully.
- [x] Both applications have at least one deterministic test and all tests pass.
- [x] Shared root commands build, test, verify, and run the applications without silent omissions.
- [x] `GET /health/live` returns a safe process-liveness response.
- [x] Clean-clone setup and verification evidence are recorded.
- [x] No scope owned by a later delivery item has been implemented or claimed as complete.

#### 11. Explicit exclusions and follow-on ownership

| Excluded work | Owning delivery item or epic |
|---|---|
| Docker Compose and PostgreSQL local environment | `DLV-PLAT-002` |
| Goose migrations, pgx, sqlc, and generated SQL | `DLV-PLAT-003` |
| OpenAPI-first workflow and generated clients | `DLV-PLAT-004` |
| Money, currency, accounting-scope, identity, and version primitives | `DLV-PLAT-005` |
| Request fingerprint and idempotency behavior | `DLV-PLAT-006` |
| PostgreSQL outbox/inbox and worker foundation | `DLV-PLAT-007` |
| Tailwind/daisyUI shell and shared UX abstractions | `DLV-UX-001` |
| Accessibility qualification harness | `DLV-UX-002` |
| Structured logging, correlation, metrics, traces, and OpenTelemetry | `DLV-OPS-001` |
| Baseline dashboard and runbooks | `DLV-OPS-002` |
| Terraform and Azure learning environment | `DLV-IAC-001`, `DLV-IAC-002` |
| Pull-request CI quality pipeline | `DLV-CI-001` |
| Entra authentication and finance authorization | `EP-IAM-001` delivery items |
| Any finance capability aggregate, handler, table, route, event, or screen | Later owning capability delivery items |

#### 12. Definition of Ready

This platform item is ready only when:

- [ ] The exact delivery item, parent epic, milestone, deliverable, and exit evidence are identified.
- [ ] Adjacent-item boundaries and explicit exclusions are understood.
- [ ] The canonical repository URL and Go module path are confirmed.
- [ ] The root command mechanism and frontend development-port behavior are selected; the API retains the approved `HTTP_ADDR` default `:8080`.
- [ ] The approved tool major versions are available or a documented setup path exists.
- [ ] Required build, test, startup, failure-propagation, and clean-clone evidence is identified.
- [ ] The five stories are small enough to implement as one or a short chain of reviewable changes.

Functional requirement IDs, authoritative finance records, finance UX states, authorization contracts, persistence contracts, correction paths, concurrency paths, and integration events are not applicable because this item establishes no product behavior or finance state.

#### 13. Definition of Done

This delivery item is done only when:

- [x] Every acceptance criterion in Sections 5–10 passes; none is silently deferred.
- [x] Required code, dependency manifests, lockfiles, tests, and documentation are complete.
- [x] Go build and tests pass.
- [x] Frontend dependency installation, tests, and production build pass.
- [x] The root verify-all command passes and its failure propagation is proven.
- [x] The clean-clone verification passes and evidence is attached or linked.
- [x] Documentation matches the versions, commands, ports, endpoint, and behavior actually implemented.
- [x] The implementation diff is reviewed against the modular-monolith and frontend capability boundaries.
- [x] No credentials, dependency directories, generated build output, test output, or local-only configuration are committed.
- [x] Traceability to the delivery item and applicable technical specifications is current.
- [x] No functional requirement, workflow, NFR qualification, adjacent delivery item, or M0 milestone is incorrectly marked complete.
- [x] No critical or high unresolved defect remains.

##### General Definition-of-Done applicability

| General control | Applicability to `DLV-PLAT-001` |
|---|---|
| Database migrations and repository integration tests | Not applicable; persistence belongs to `DLV-PLAT-002` and `DLV-PLAT-003`. |
| OpenAPI and generated contracts | Not applicable; owned by `DLV-PLAT-004`. |
| Authorization, sensitive finance data, and audit evidence | Not applicable; this item has no protected finance action or finance record. |
| Idempotency, concurrency, correction, and recovery paths | Not applicable to the repository shell; owned by later platform and capability items. Startup failure and command failure propagation are tested here. |
| Accessibility and localization qualification | Not completed here; owned by dedicated UX and NFR delivery items. The minimal shell must not block later adoption. |
| Structured logs, metrics, and traces | Not completed here; owned by `DLV-OPS-001`. Health output and ordinary startup failures must still avoid secrets. |
| Clean-environment demonstration | Applicable and mandatory through the clean-clone verification. |

#### 14. Traceability and quality-gate contribution

| Traceability field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-001` |
| Direct functional requirement IDs | None |
| Direct global requirement IDs completed | None; global controls have separate delivery items. |
| Workflow IDs | None |
| NFR qualification completed | None |
| Primary quality-gate contribution | `QG-01` through controlled source scope and traceability; `QG-10` through repeatable local evidence. |
| Gates not completed by this item | `QG-03`, `QG-04`, `QG-05`, and `QG-08` remain dependent on their owning M0 items. |
| Exit evidence | Clean clone builds and tests locally. |

#### 15. Consistency review record

##### Review pass 1 — Corrected

The first review identified and corrected:

- an unsupported `P0` assignment;
- a required `docs/` directory not present in the approved minimum repository baseline;
- an unconditional `go.sum` requirement even though it exists only when dependencies generate it;
- omission of the approved `chi` router from the Go API shell;
- exact root command names presented as source requirements rather than implementation aliases;
- a Definition of Done that allowed deferred acceptance criteria; and
- omission of the no-critical-or-high-defect completion rule.

##### Review pass 2 — Passed

The second review passed all 44 structural and semantic checks covering:

- exact delivery-item scope and exit evidence;
- selected Go, HTTP, React, TypeScript, Node.js, and Vite baselines;
- backend and frontend repository boundaries;
- liveness semantics and the approved API default address;
- adjacent M0 delivery-item ownership;
- Definition-of-Ready and Definition-of-Done applicability;
- M0 quality-gate contribution without false completion claims; and
- Markdown structure, evidence completeness, and internal consistency.

##### Review pass 3 — Passed

The pnpm revision review confirmed:

- `ADR-021` authorizes pnpm as the sole frontend package manager;
- the solution architecture, story criteria, clean-clone commands, evidence template, and lockfile expectations agree;
- `web/pnpm-lock.yaml` is committed and alternate frontend lockfiles are prohibited;
- the exact pnpm version is pinned through the `packageManager` field in `web/package.json`;
- frozen-lockfile installation is used for clean-clone and later CI reproducibility; and
- no finance-domain, module-boundary, runtime, framework, testing, deployment, or adjacent-delivery-item behavior changed.

**Final consistency result: PASS.**

#### 16. Source references

- `02_work_breakdown_backlog_v1.0.md`
  - Platform foundation backlog: exact `DLV-PLAT-001` deliverable and exit evidence.
  - Adjacent platform items: separation of Docker, persistence, OpenAPI, primitives, idempotency, outbox/inbox, UX, operations, infrastructure, and CI.
  - Definitions of Ready and Done.
- `01_backend_module_specifications_v1.0.md`
  - Repository layout and dependency rules.
- `01_solution_architecture_overview_v1.0.md`
  - Modular-monolith direction, selected Go/React technology baseline, and pnpm package-manager policy.
- `05_architecture_traceability_decisions_v1.0.md`
  - `ADR-021` authorizing pnpm as the sole frontend package manager.
- `05_frontend_ui_technical_specifications_v1.0.md`
  - Approved frontend structure and capability boundary rule.
- `08_observability_operations_specifications_v1.0.md`
  - `/health/live` semantics and safe health responses.
- `04_quality_testing_environment_plan_v1.0.md`
  - Continuous quality evidence, quality gates, and test-layer expectations.
- `03_dependencies_milestones_releases_v1.0.md`
  - `EP-PLAT-001` dependency position and M0 gate context.

### DLV-PLAT-002 — Docker Compose PostgreSQL Development Environment User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-002` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Artifact version | 1.0 |
| Review status | Passed — three consistency-review passes completed |
| Delivery profile | Solo, part-time, local-first learning project |
| Dependency position | `DLV-PLAT-001` supplies the practical repository and root-command foundation. No authoritative item-level predecessor is stated. Completion also requires a minimum executable migration and seed contract coordinated with `DLV-PLAT-003`, without claiming that item complete. |
| Authoritative deliverable | Create Docker Compose PostgreSQL development environment. |
| Authoritative exit evidence | Database starts, migrates, seeds and resets reproducibly. |

#### 1. Purpose and scope

Break `DLV-PLAT-002` into small, reviewable stories without absorbing work assigned to `DLV-PLAT-003`.

The completed item provides:

- one local PostgreSQL service managed through Docker Compose;
- safe local configuration with no committed real credentials;
- root command intents for database startup, health, shutdown, migration, seed, verification, and reset;
- deterministic invocation of the current repository migration and seed contracts; and
- evidence that the database can be destroyed and recreated from committed source.

This is an engineering-environment item. It completes no `FR-*`, `GFR-*`, `WF-*`, or `NFR-*` delivery item and does not complete `M0` by itself.

#### 2. Authoritative implementation baseline

| Area | Baseline |
|---|---|
| Architecture | Preserve the modular monolith and one PostgreSQL database per environment. |
| Local profile | Daily development is local-first; Docker Compose and local PostgreSQL require no Azure resource. |
| Database | PostgreSQL `18.x`; use a reviewed pinned image reference, not an unbounded `latest` tag. |
| Ownership | Future schema-per-context ownership remains intact; this item creates no finance schema, aggregate table, or business fact. |
| Migration boundary | `DLV-PLAT-002` owns local execution and reproducibility. `DLV-PLAT-003` owns Goose, pgx, sqlc, generated SQL, migration conventions, and drift detection. |
| Seed boundary | A versioned deterministic seed contract is required, using synthetic data without inventing finance records. |
| Security | Real credentials and developer-specific environment files are not committed. |
| Evidence | Only executed commands and recorded results can establish success. |

Patch versions, image digests, service names, ports, local database names, usernames, and command aliases are implementation evidence rather than architecture contracts.

#### 3. Boundary resolution with DLV-PLAT-003

The `DLV-PLAT-002` exit evidence says the database “migrates,” while `DLV-PLAT-003` owns the migration and database-access workflow. The boundary is:

| Concern | `DLV-PLAT-002` | `DLV-PLAT-003` |
|---|---|---|
| PostgreSQL service | Define, start, health-check, stop, and reset it. | No ownership. |
| Migration execution | Invoke the current approved migration command against local PostgreSQL and propagate failure. | Establish Goose layout and history, migration authoring, pgx, sqlc, generated SQL, and drift checks. |
| Seed execution | Invoke a versioned synthetic seed contract and verify its declared result. | Supply migration-created structures needed by that contract when persistence work owns them. |
| CI | No CI implementation or completion claim. | Migration/generated-code drift is later enforced through CI. |
| Completion | Requires a real migration and seed path. | Need not be fully complete merely to provide that minimum executable contract. |

A no-op migration, empty seed, or command that always exits successfully without proving database state does not satisfy this item.

#### 4. Required implementation inputs

Before closing the item, resolve from repository evidence:

- Docker Engine and Docker Compose v2 setup;
- a reviewed PostgreSQL `18.x` image reference;
- Compose project, service, volume, host binding, port, database, username, and local-only credential source;
- root command aliases using the mechanism established by `DLV-PLAT-001`;
- a minimum real migration command;
- a versioned synthetic seed command;
- a deterministic prepared-state verification method; and
- a reset scope limited to TALLY project-owned resources.

Do not invent a finance schema, table, identifier, or business record merely to create a seed target.

#### 5. Cross-story constraints

1. Use one PostgreSQL server/database for the local environment; do not introduce one database per module.
2. Preserve future schema-per-context ownership and prohibit cross-context write shortcuts.
3. Use PostgreSQL `18.x`; do not use an unbounded `latest` image tag.
4. Require no Azure, Terraform, Entra ID, cloud credential, or external provider.
5. Commit no real credential, production connection string, key, certificate, or developer-specific `.env`.
6. Use synthetic, non-sensitive seed data only.
7. Add no finance aggregate, business command, API operation, event, outbox/inbox behavior, or capability UI.
8. Add no pgx repository, sqlc query, generated SQL, or general ORM.
9. Do not claim `DLV-PLAT-003`, full `QG-03`, CI drift detection, production readiness, or `M0` complete.
10. Do not rely solely on first-container-start initialization; migration and seed must be explicitly rerunnable.
11. Normal stop preserves the database volume; only a clearly named reset may destroy it.
12. Reset may remove only TALLY Compose project resources and must not delete unrelated Docker resources.
13. Root commands propagate Docker, health, migration, seed, and verification failures as non-zero outcomes.
14. Do not implement `/health/ready`; API database readiness requires a later approved database-integration change.
15. Do not claim production availability, recovery, performance, security, or other NFR qualification.

---

#### 6. User Story 1 — Define safe local database configuration

**As the TALLY developer, I want a checked-in Docker Compose definition and safe local configuration contract, so that each workstation begins from the same PostgreSQL baseline without committing real secrets.**

##### Value

Establishes version, configuration, storage, and exposure rules before lifecycle automation.

##### Acceptance criteria

- [x] A Docker Compose file is committed at a documented repository path.
- [x] It defines one PostgreSQL development service and no unrelated infrastructure.
- [x] It uses PostgreSQL `18.x` through a reviewed pinned tag or immutable digest, not `latest`.
- [x] It declares one project-owned persistent volume.
- [x] Host exposure is limited to the developer machine by default; selected port behavior is documented.
- [x] Database name, username, port, and credential inputs are configurable without editing the Compose file.
- [x] A checked-in example configuration contains clearly non-production placeholders or local-only values.
- [x] Developer-specific configuration is ignored by Git.
- [x] No real credential, production hostname, Azure reference, or sensitive finance data is committed.
- [x] `docker compose config` succeeds with the documented setup.
- [x] No finance schema, table, migration, business record, outbox, or inbox is introduced.
- [x] README wording distinguishes this learning environment from production/reference architecture.

##### Likely files

```text
compose.yaml
.env.example
.gitignore
README.md
```

Exact names may differ.

##### Evidence

- Docker and Compose version output.
- `docker compose config` output.
- Recorded image reference and resolved identifier where available.
- Secret and scope review of the diff.

---

#### 7. User Story 2 — Start and health-check PostgreSQL

**As the TALLY developer, I want PostgreSQL to start through Docker Compose and become observably healthy, so that later commands do not race startup or report success too early.**

##### Value

Creates a deterministic prerequisite for migration, seed, test, and development work.

##### Acceptance criteria

- [x] The service starts in detached mode through a documented command.
- [x] It defines a database-aware health check using the declared local database and user.
- [x] Health behavior has bounded interval, timeout, retries, and start period.
- [x] A wait command exits successfully only after health is established.
- [x] The wait path exits non-zero with useful diagnostics when health cannot be established within its bound.
- [x] Container startup failure is not reported as successful startup.
- [x] The server reports PostgreSQL `18.x`.
- [x] Stop/start without reset preserves named-volume state.
- [x] Startup requires no Azure, application process, authentication, or external provider.
- [x] Migration and seed are not hidden solely inside first-boot initialization.
- [x] API liveness semantics remain unchanged and `/health/ready` is not added.

##### Evidence

- Startup and `docker compose ps` output showing healthy state.
- PostgreSQL version query.
- Stop/start persistence result.
- Controlled health-failure result with non-zero status.

---

#### 8. User Story 3 — Provide root database lifecycle commands

**As the TALLY developer, I want a small root database command surface, so that I can operate local PostgreSQL consistently without memorizing Compose details.**

##### Value

Aligns database operation with the root command approach established by `DLV-PLAT-001`.

##### Required command intents

Names are recommended aliases, not contracts.

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Validate | `db-config` | Validate effective Compose configuration. |
| Start | `db-up` | Start PostgreSQL and propagate startup failure. |
| Wait | `db-wait` | Wait until healthy or fail within a bound. |
| Status | `db-status` | Show service and health state. |
| Logs | `db-logs` | Show database logs without printing credentials. |
| Shell | `db-shell` | Connect to the declared local database. |
| Stop | `db-down` | Stop services while preserving the volume. |

##### Acceptance criteria

- [x] Every lifecycle intent runs from repository root.
- [x] Commands use the checked-in Compose definition and documented local configuration.
- [x] Start includes or clearly directs the bounded health-wait step.
- [x] Stop preserves the volume.
- [x] Status distinguishes available Compose states where supported.
- [x] Shell uses local configuration and does not hard-code a production credential.
- [x] Logs do not echo full connection strings or credentials.
- [x] Missing Docker, invalid config, startup failure, health timeout, and shell failure return non-zero.
- [x] Repeating safe commands produces understandable outcomes.
- [x] No unnecessary task-runner dependency is added.
- [x] Prerequisites, names, working directory, normal behavior, and failure behavior are documented.

##### Evidence

- Output from each selected lifecycle command.
- A controlled child-command failure proving propagation.
- Stop/start evidence showing volume preservation.

---

#### 9. User Story 4 — Orchestrate migration and deterministic seeding

**As the TALLY developer, I want explicit migration, seed, and verification commands against local PostgreSQL, so that preparation is repeatable and independent of hidden first-start behavior.**

##### Value

Proves “migrates and seeds” while preserving `DLV-PLAT-003` ownership.

##### Required command intents

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Migrate | `db-migrate` | Invoke the current approved repository migration contract. |
| Seed | `db-seed` | Establish the declared synthetic seed state without duplicates. |
| Verify | `db-verify` | Prove migration state and seed version/fingerprint or equivalent deterministic result. |
| Prepare | `db-prepare` | Wait, migrate, seed, and verify in order; stop on failure. |

##### Acceptance criteria

- [x] All four intents run from repository root.
- [x] Migration refuses to report success when no real migration contract executed.
- [x] It uses the approved repository migration contract rather than a competing framework.
- [x] Seed input is committed, versioned, synthetic, and non-sensitive.
- [x] Seed input contains no real customer, vendor, employee, bank, payroll, tax, credential, or production data.
- [x] Seed input does not invent finance entities or facts before their owning delivery items.
- [x] Re-running seed establishes the same declared state without duplicate rows or unexplained differences.
- [x] Verification proves migration and seed state, not merely connectivity.
- [x] Prepare runs wait, migration, seed, and verification in that order.
- [x] Any migration, seed, or verification failure stops prepare and returns non-zero.
- [x] The commands work after normal stop/start and after reset.
- [x] No pgx repository, sqlc query, generated SQL, ORM, CI workflow, or drift-completion claim is added.
- [x] Traceability separates any same-chain `DLV-PLAT-003` work from this item.

##### Evidence

- Applied/current migration output.
- Seed version or fingerprint output.
- Prepared-state verification output.
- A second seed/verify run proving deterministic repetition.
- Controlled preparation failure with non-zero status.

---

#### 10. User Story 5 — Reset and prove reproducibility

**As the TALLY developer, I want a deliberately destructive, project-scoped reset and recorded clean-environment proof, so that the database can be rebuilt without relying on an old volume.**

##### Value

Directly proves the complete exit evidence.

##### Reset acceptance criteria

- [x] A clearly named reset command runs from repository root.
- [x] Documentation labels it destructive to the TALLY local database.
- [x] It removes only the TALLY Compose project’s declared disposable resources and database volume.
- [x] It does not run broad Docker pruning or delete unrelated containers, networks, images, or volumes.
- [x] It recreates PostgreSQL, waits for health, migrates, seeds, and verifies.
- [x] Any failed phase returns non-zero.
- [x] Success leaves PostgreSQL healthy in the declared migration and seed state.
- [x] Two complete reset runs produce the same declared verification result.
- [x] At least one verification begins with the prior TALLY database volume absent.
- [x] Verification requires no uncommitted file, prior volume, real credential, Azure, or external provider.
- [x] Image caches may improve speed but are not required for logical correctness.

##### Documentation acceptance criteria

- [x] README documents Docker and Compose prerequisites.
- [x] README documents safe local configuration setup.
- [x] README documents all selected lifecycle, migration, seed, verify, and reset commands.
- [x] README documents host binding and port behavior.
- [x] README distinguishes ordinary stop from destructive reset.
- [x] README explains the `DLV-PLAT-002`/`DLV-PLAT-003` boundary.
- [x] README does not claim Azure PostgreSQL, HA, backup, DR, production NFRs, CI, authentication, finance schemas, or business modules complete.
- [x] A verification record is committed or linked.

##### Evidence record template

```markdown
## DLV-PLAT-002 local database reproducibility verification

- Commit:
- Verification date:
- Operating system:
- Docker version:
- Docker Compose version:
- PostgreSQL image reference:
- Resolved PostgreSQL version or image identifier:
- Compose project/service:
- Local configuration source:
- Commands executed:
  1.
  2.
  3.
- Initial volume state:
- Configuration validation result:
- Startup and health result:
- Migration result:
- Seed version or fingerprint:
- Prepared-state verification result:
- First reset result:
- Second reset result:
- Unrelated Docker resources affected: No
- Real credentials required: No
- Azure or external services required: No
- Overall result: PASS | FAIL
- Notes:
```

---

#### 11. Delivery-item acceptance summary

`DLV-PLAT-002` is complete only when all five stories and all conditions below pass:

- [x] One checked-in PostgreSQL `18.x` Compose service exists.
- [x] Effective configuration validates and contains no real committed secret.
- [x] PostgreSQL starts and becomes healthy within a bound.
- [x] Root lifecycle commands operate it and propagate failures.
- [x] A real migration executes against a fresh database.
- [x] A versioned synthetic seed executes deterministically.
- [x] Prepared-state verification proves more than connectivity.
- [x] Stop/start preserves ordinary local state.
- [x] Reset destroys only project-owned state.
- [x] Two reset-and-prepare runs produce the same declared result.
- [x] Clean-environment evidence is recorded.
- [x] No adjacent delivery item is falsely marked complete.

#### 12. Explicit exclusions and follow-on ownership

| Excluded work | Owner |
|---|---|
| Goose layout/history, migration authoring, pgx, sqlc, generated SQL, and drift checks | `DLV-PLAT-003` |
| Pull-request CI enforcement | `DLV-CI-001` |
| OpenAPI-first workflow | `DLV-PLAT-004` |
| Shared money, currency, scope, identity, and version primitives | `DLV-PLAT-005` |
| Business idempotency foundation | `DLV-PLAT-006` |
| Outbox/inbox and worker foundation | `DLV-PLAT-007` |
| Finance schemas, aggregate tables, business migrations, and finance scenario data | Owning capability items through the approved persistence workflow |
| API database pool, repositories, and `/health/ready` | Later approved database-integration work using `DLV-PLAT-003` |
| Full Testcontainers persistence suites | Persistence and capability testing work |
| Structured telemetry, dashboards, and runbooks | `DLV-OPS-001`, `DLV-OPS-002` |
| Terraform, Azure PostgreSQL, networking, Key Vault, and cloud deployment | `DLV-IAC-001`, `DLV-IAC-002` |
| Entra and finance authorization | `EP-IAM-001` delivery items |
| Production HA, backup/restore, PITR, RTO/RPO, and NFR qualification | Later recovery and qualification items |

#### 13. Definition of Ready

- [x] Exact item, epic, milestone, deliverable, and exit evidence are identified.
- [x] The `DLV-PLAT-001` repository and root command surface are available or stable.
- [x] Docker Engine and Compose v2 have a setup path.
- [x] A reviewed PostgreSQL `18.x` image can be selected.
- [x] Local project/service/volume/binding/port/database/user/config choices can be recorded.
- [x] The `DLV-PLAT-002`/`DLV-PLAT-003` boundary is understood.
- [x] Minimum real migration, synthetic seed, and deterministic verification paths are available or scheduled in the same short chain.
- [x] Reset scope is limited to TALLY project resources.
- [x] Positive, negative, stop/start, rerun, reset, and clean-environment evidence is identified.
- [x] Five stories are small enough for one or a short chain of reviewable changes.

Finance requirements, authoritative finance records, UX states, authorization decisions, correction paths, business concurrency, and integration events are not applicable.

#### 14. Definition of Done

- [x] Every criterion in Sections 6–11 passes; none is silently deferred.
- [x] Compose config, root commands, safe example config, verification logic, checks, and documentation are complete.
- [x] `docker compose config` passes.
- [x] PostgreSQL `18.x` starts and becomes healthy within the declared bound.
- [x] Normal stop/start preserves the volume.
- [x] Migration, seed, and verification pass on a fresh database.
- [x] Repeated seed establishes the same state without duplicates.
- [x] Reset passes twice with the same verification result.
- [x] Controlled startup/health and preparation failures propagate non-zero.
- [x] Clean-environment evidence is attached or linked.
- [x] Documentation matches actual image, commands, host/port behavior, reset scope, and verification method.
- [x] No real credential, production data, unrelated Docker resource, build output, or developer-specific config is committed.
- [x] The diff is reviewed against data ownership and adjacent-item boundaries.
- [x] Traceability is current.
- [x] No FR, GFR, workflow, NFR qualification, `DLV-PLAT-003`, adjacent item, or `M0` is falsely marked complete.
- [x] No critical or high unresolved defect remains.

##### General Definition-of-Done applicability

| Control | Applicability |
|---|---|
| Domain/application behavior | Not applicable. |
| Database migrations | Local execution is mandatory; the full migration/access workflow remains `DLV-PLAT-003`. |
| Repository integration tests | Not completed unless needed for the environment contract. |
| OpenAPI | Not applicable. |
| Authorization/audit | No finance action exists; secret hygiene and synthetic seed data are mandatory. |
| Business idempotency/concurrency/corrections | Not applicable; deterministic technical seeding does not complete business controls. |
| Accessibility/localization | Not applicable. |
| Telemetry | Not completed; Docker/command output must avoid credentials. |
| Recovery/DR | Local recreation is proved; backup restore, PITR, RTO/RPO, and reconciliation are not. |
| Clean-environment demonstration | Mandatory. |

#### 15. Traceability and quality-gate contribution

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-002` |
| Direct FR IDs | None |
| Direct GFR IDs completed | None |
| Workflow IDs | None |
| NFR qualification | None |
| Primary contribution | `QG-01` for controlled scope; `QG-10` for repeatable clean-environment evidence. |
| Partial contribution | `QG-03` only through repeatable local migration execution; this item does not complete `QG-03`. |
| Exit evidence | Database starts, migrates, seeds and resets reproducibly. |

#### 16. Consistency review record

##### Review pass 1 — Corrected

The first semantic review corrected:

- overlap between “migrates” in this item and Goose/pgx/sqlc ownership in `DLV-PLAT-003`;
- possible false success from a no-op migration or empty seed;
- first-start-only initialization that would not prove rerun behavior;
- premature API database readiness work;
- invented finance schemas or records used merely as seed targets;
- unsafe broad Docker cleanup;
- unbounded image selection through `latest`; and
- false completion claims for `QG-03`, production readiness, `DLV-PLAT-003`, or `M0`.

The corrected boundary assigns local orchestration to this item and retains migration authoring/database-access workflow ownership in `DLV-PLAT-003`.

##### Review pass 2 — Passed

Structural and semantic checks passed for:

- exact scope and exit evidence;
- five small stories with value, criteria, and evidence;
- PostgreSQL `18.x` and local-first alignment;
- secret hygiene;
- bounded health and failure propagation;
- stop versus reset semantics;
- deterministic migration/seed/verify behavior;
- project-scoped deletion;
- adjacent-item exclusions;
- Ready/Done applicability;
- quality-gate contribution without false completion; and
- Markdown structure and internal references.

##### Review pass 3 — Passed

The final review confirmed:

- no finance requirement, ID, API, schema, table, event, or business seed fact was invented;
- no exact local port, credential, database name, service name, or alias is treated as an architecture contract;
- real migration and seed results are required, not mere connectivity;
- `DLV-PLAT-003` retains Goose, pgx, sqlc, generated SQL, and drift ownership;
- reset cannot affect unrelated Docker resources;
- no Azure, CI, authentication, observability, production, backup/restore, or NFR completion is claimed; and
- the acceptance summary proves “Database starts, migrates, seeds and resets reproducibly.”

**Final consistency result: PASS.**

#### 17. Source references

- `02_work_breakdown_backlog_v1.0.md` — exact deliverable, exit evidence, adjacent items, Ready, and Done.
- `01_solution_architecture_overview_v1.0.md` — modular monolith, PostgreSQL `18.x`, local Docker Compose profile, and version policy.
- `03_data_integration_architecture_v1.0.md` — PostgreSQL authority, schema ownership, no cross-context writes, and migration discipline.
- `03_database_persistence_specifications_v1.0.md` — one database per environment, PostgreSQL `18.x`, schema ownership, and migration responsibilities.
- `09_testing_performance_recovery_specifications_v1.0.md` — PostgreSQL-equivalent testing and recovery boundaries.
- `04_quality_testing_environment_plan_v1.0.md` — `QG-01`, `QG-03`, `QG-10`, environment progression, Ready, and Done.
- `03_dependencies_milestones_releases_v1.0.md` — local-primary policy, clean-environment evidence, and change control.
- `05_risks_costs_governance_traceability_v1.0.md` — local-first cost control, synthetic-data control, and scope governance.

### DLV-PLAT-003 — Goose Migrations, pgx and sqlc Workflow User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-003` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Artifact version | 1.0 |
| Review status | Passed — three consistency-review passes completed |
| Delivery profile | Solo, part-time, local-first learning project |
| Dependency position | `DLV-PLAT-001` supplies the repository and root command surface. `DLV-PLAT-002` supplies the reproducible PostgreSQL 18 local environment and the current executable migration contract. No authoritative item-level predecessor is stated, but both foundations are practical prerequisites. |
| Authoritative deliverable | Establish Goose migrations, pgx and sqlc workflow. |
| Authoritative exit evidence | CI detects migration and generated-code drift. |

#### 1. Purpose and scope

Break `DLV-PLAT-003` into small, reviewable stories that establish the approved PostgreSQL persistence workflow without prematurely implementing finance capabilities.

The completed item provides:

- pinned, repository-controlled Goose and sqlc tooling;
- an ordered Goose migration layout with schema-owned history;
- a minimal pgx connection and transaction workflow under `internal/platform/database`;
- a sqlc source-to-generated-code workflow using an existing technical platform table;
- PostgreSQL 18 integration verification from a clean database; and
- local and CI checks that reject migration or generated-code drift.

This is a persistence-tooling foundation. It completes no `FR-*`, `GFR-*`, `WF-*`, or `NFR-*` delivery item and does not complete `M0` by itself.

#### 2. Authoritative implementation baseline

| Area | Baseline |
|---|---|
| Architecture | Preserve the modular monolith and explicit module/schema ownership. |
| Database | PostgreSQL `18.x`; one server and database per environment. |
| Migrations | Goose SQL migrations; ordered and reviewable; expand/migrate/contract discipline. |
| Migration history | One Goose migration-history table per initialized schema. |
| Database access | `pgx` and `sqlc`; no general ORM. |
| Repository layout | Shared database facilities belong under `internal/platform/database`; migrations belong under `db/migrations/`. |
| Query ownership | A query may reference only its owning schema and approved platform schemas. Cross-context write-path joins are prohibited. |
| Testing | PostgreSQL integration verification uses the production-equivalent PostgreSQL major version; Testcontainers is the approved integration-test mechanism. |
| Shared-environment correction | Migration failure is handled by forward-fix by default. Destructive rollback requires explicit evidence that no established financial fact is lost. |
| Exit evidence | CI must detect both migration drift and stale or manually changed sqlc output. |

Exact Goose, pgx, sqlc, Testcontainers, and transitive dependency versions are implementation pins recorded in repository manifests. They are not new architecture decisions.

#### 3. Boundary resolution

##### 3.1 Boundary with DLV-PLAT-002

| Concern | `DLV-PLAT-002` | `DLV-PLAT-003` |
|---|---|---|
| PostgreSQL service | Starts, health-checks, stops, resets, seeds, and verifies the local database. | Reuses that environment; does not replace Compose lifecycle ownership. |
| Migration invocation | Calls the current repository migration command and propagates failure. | Defines the Goose layout, history, authoring rules, commands, and verification behind that invocation. |
| Seed manifest | Creates and verifies deterministic synthetic local seed state. | May use the existing technical seed-manifest table as the first sqlc workflow target without changing its business meaning. |
| Reproducibility | Proves reset and preparation from a missing volume. | Proves migrations and generated code are reproducible and drift-free. |

The existing real migration must be preserved or deliberately converted into valid Goose form. Replacing it with a no-op migration does not satisfy this item.

##### 3.2 Boundary with DLV-CI-001

The exact `DLV-PLAT-003` exit evidence requires CI detection, while `DLV-CI-001` owns the complete pull-request quality pipeline.

| Concern | `DLV-PLAT-003` | `DLV-CI-001` |
|---|---|---|
| Persistence checks | Owns executable Goose/sqlc drift checks and their minimal CI invocation. | Incorporates or orchestrates those checks in the full repository quality pipeline. |
| CI breadth | Only migration, sqlc generation, compilation, and minimum persistence verification required by this item. | Go, frontend, OpenAPI, SQL, Terraform, security, documentation, branch gating, and complete PR quality policy. |
| Completion claim | May claim the exact persistence drift exit evidence. | Remains incomplete until the complete PR pipeline and all declared checks gate merge. |

A focused persistence CI job or workflow is permitted because it is required by the authoritative exit evidence. It must not be represented as completing `DLV-CI-001`.

##### 3.3 Boundary with capability delivery

This item establishes the workflow, not the finance data model.

- Do not create all 19 bounded-context schemas merely to imitate the final topology.
- Initialize only schemas and technical tables already justified by delivered work.
- Do not create finance aggregate tables, finance seed records, business repositories, business commands, API operations, events, or screens.
- Later capability items add their owned migrations and sqlc queries through this workflow.

#### 4. Required implementation inputs

Before closing the item, resolve from repository evidence:

- the current root command mechanism and existing database aliases;
- the current migration file and local seed-manifest structure created during `DLV-PLAT-002`;
- exact pinned Goose, pgx, sqlc, and Testcontainers versions;
- the selected migration directory convention for schema-owned migration sets;
- the selected generated-code path under `internal/platform/database`;
- the minimal platform query used to prove sqlc generation;
- the migration checksum or equivalent immutable-history verification method;
- the CI runner strategy for PostgreSQL 18; and
- whether the repository uses a focused persistence workflow now or a reusable job intended for later inclusion in `DLV-CI-001`.

Do not invent a new finance table or query merely to give sqlc something to generate. Use the current technical platform persistence contract.

#### 5. Cross-story constraints

1. Preserve one PostgreSQL server/database per environment and schema-per-context ownership.
2. Use Goose, pgx, and sqlc; do not add a general ORM or competing migration framework.
3. Pin tool versions in repository-controlled manifests; do not depend on unspecified globally installed versions.
4. Keep migrations under `db/migrations/` and shared database code under `internal/platform/database/`.
5. Retain one Goose migration-history table per initialized schema.
6. Do not create empty future schemas, migration histories, roles, repositories, or query packages merely to mirror the final architecture.
7. Do not add cross-context foreign keys, cross-schema write-path joins, or direct access to another module's future schema.
8. Do not create or modify finance aggregate tables, business records, commands, events, API contracts, or UI behavior.
9. Do not implement money, currency, accounting-scope, identity, version, idempotency, outbox, inbox, or worker foundations owned by `DLV-PLAT-005` through `DLV-PLAT-007`.
10. Do not use floating point for any numeric persistence example.
11. Do not wire finance repositories or claim application-level database readiness before an approved consumer exists.
12. Do not expose `DATABASE_URL`, credentials, or complete connection strings in logs, CI output, test failure messages, or generated files.
13. Treat committed historical migrations as immutable; changes require a new migration or an explicitly reviewed pre-release correction with updated evidence.
14. Shared-environment migration recovery is forward-fix by default; local disposable reset is not evidence that destructive production rollback is safe.
15. Generated sqlc files are machine-produced, committed when selected by the repository workflow, and never manually edited.
16. Root and CI commands must return non-zero when migration validation, clean application, generation, compilation, integration verification, or drift checks fail.
17. Do not claim full `QG-03`, `DLV-CI-001`, production readiness, recovery qualification, or `M0` complete.

---

#### 6. User Story 1 — Establish the Goose migration contract

**As the TALLY developer, I want a pinned and schema-aware Goose migration workflow, so that database changes are ordered, reviewable, repeatable, and owned by the correct schema.**

##### Value

Turns the minimum migration path used by `DLV-PLAT-002` into the approved long-term migration workflow without changing finance-domain scope.

##### Required command intents

Names are recommended aliases, not architecture contracts.

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Validate migrations | `db-migrate-validate` | Validate migration ordering and Goose syntax for each initialized schema. |
| Show status | `db-migrate-status` | Show applied and pending migrations for each initialized schema. |
| Apply migrations | Existing `db-migrate` | Apply pending migrations through pinned Goose tooling. |
| Create migration | `db-migrate-create` | Create a correctly located migration skeleton for an explicitly selected schema. |
| Verify migration source | `db-migrate-check` | Verify migration inventory/checksums or the selected equivalent drift contract. |

##### Acceptance criteria

- [x] Goose is the sole migration framework.
- [x] The selected exact Goose version is pinned through a repository-controlled Go tool or equivalent tooling manifest.
- [x] Migration commands use the pinned version and do not require a separately installed unversioned Goose binary.
- [x] The current real platform migration is valid Goose SQL and preserves its existing technical behavior.
- [x] Migrations are grouped by owning schema under `db/migrations/` or an equivalently explicit schema-owned layout.
- [x] Each initialized schema uses its own Goose migration-history table.
- [x] No migration-history table is created for a schema that has no delivered migration.
- [x] Migration filenames are ordered, unique, and documented.
- [x] Migration files contain the required Goose annotations and have deterministic statement boundaries.
- [x] The existing root `db-migrate` command delegates to the Goose workflow rather than a competing raw runner.
- [x] Migration commands use documented local configuration without hard-coding credentials or production endpoints.
- [x] Applying migrations to a clean PostgreSQL 18 database succeeds.
- [x] A second apply reports no pending migration and makes no duplicate schema change.
- [x] Invalid ordering, duplicate versions, invalid SQL, or an unavailable database returns non-zero.
- [x] A deterministic migration inventory, checksum manifest, or equivalent reviewed mechanism detects changed, deleted, or unrecorded committed migrations.
- [x] Shared-environment guidance states forward-fix by default; a local `down` command, if retained, is documented as disposable-development behavior only.
- [x] Migration authoring guidance requires lock-risk, expected duration, forward-fix/rollback approach, backup need, and verification query when the change reaches business data.
- [x] No finance schema or aggregate table is introduced.

##### Likely files

```text
go.mod
go.sum
db/migrations/<initialized-schema>/*.sql
db/migrations/<migration-inventory-or-checksum-file>
<root command manifest>
README.md
```

Exact names may differ, but schema ownership and pinned tooling must remain explicit.

##### Evidence

- Pinned Goose version output.
- Migration validation output.
- Clean-database apply and status output.
- Second apply showing no pending change.
- Migration history location for every initialized schema.
- Controlled invalid-migration result with non-zero status.
- Controlled migration-drift result with non-zero status.

---

#### 7. User Story 2 — Establish the pgx database foundation

**As the TALLY developer, I want a minimal pgx connection and transaction foundation, so that future repositories can use PostgreSQL explicitly without introducing an ORM or bypassing module boundaries.**

##### Value

Provides the approved low-level database access mechanism while keeping business repositories in their future owning modules.

##### Acceptance criteria

- [x] The repository uses the approved pgx major line and pins the selected exact dependency version through Go modules.
- [x] Shared pgx setup is located under `internal/platform/database/`.
- [x] The package opens a PostgreSQL pool from validated configuration rather than embedding a connection string.
- [x] Pool creation accepts a context and uses a bounded connectivity check.
- [x] Pool shutdown is explicit and safe to call from a composition root or test cleanup.
- [x] The connection string and credentials are never logged or included in returned errors.
- [x] The implementation remains compatible with the approved `DATABASE_URL` and bounded pool-size configuration contract.
- [x] The package exposes only the minimal facilities needed by sqlc and future repositories; it does not introduce a generic repository, unit-of-work framework, service locator, or ORM abstraction.
- [x] Transactions use pgx transaction semantics and sqlc's transaction-compatible query binding rather than a custom finance transaction model.
- [x] A committed integration test proves successful connection to PostgreSQL 18.
- [x] A committed integration test proves transaction commit behavior using technical synthetic data.
- [x] A committed integration test proves rollback on an intentional error or cancellation.
- [x] Test cleanup leaves no unexplained persistent row or open connection.
- [x] Connection refusal, invalid configuration, context expiry, and transaction failure produce explicit non-success outcomes.
- [x] The API is not required to gain finance repositories, protected actions, or `/health/ready` merely to complete this story.
- [x] No business module imports or uses the platform database package before its own approved delivery item.

##### Likely files

```text
internal/platform/database/
  pool.go
  pool_test.go
  integration_test.go
```

Exact files may differ. Keep the package small and technical.

##### Evidence

- Go module diff showing pgx pinning.
- Successful PostgreSQL 18 connection test.
- Commit and rollback integration-test output.
- Controlled invalid-configuration or unavailable-database failure.
- Review confirming no ORM, generic repository, finance package, or credential logging was added.

---

#### 8. User Story 3 — Establish the sqlc generation workflow

**As the TALLY developer, I want schema-owned SQL queries to generate typed Go code through sqlc, so that SQL remains explicit while routine row and parameter mapping is reproducible.**

##### Value

Implements the approved pgx/sqlc choice and creates the source-to-generated-code feedback loop required for later capability repositories.

##### Acceptance criteria

- [x] The selected exact sqlc version is pinned through a repository-controlled tool or equivalent tooling manifest.
- [x] A checked-in sqlc configuration identifies schema source, query source, Go package, output location, and pgx target.
- [x] Query source is grouped by owning schema or module.
- [x] The initial query targets the existing delivered technical platform persistence contract; it does not require a new finance table.
- [x] The initial query has a real use in verification or integration testing and is not an unused demonstration query.
- [x] Query names are stable and descriptive.
- [x] Generated Go code is placed under `internal/platform/database/` in a clearly generated package or directory.
- [x] Generated files include the tool-generated marker and are not manually edited.
- [x] Generated code is committed if the selected workflow uses committed generation artifacts.
- [x] sqlc uses pgx-compatible generated code and does not introduce `database/sql` merely as a second access path.
- [x] Generated code compiles with `go test ./...` or the repository's complete Go verification command.
- [x] A transaction can bind the generated query set to a pgx transaction.
- [x] The initial query references only the `platform` schema and any PostgreSQL system metadata explicitly needed by the tooling.
- [x] No query references a future finance schema or performs a cross-context join.
- [x] No generated type introduces floating-point representation for exact numeric database values.
- [x] Changing query or schema source and running generation produces a deterministic, reviewable diff.
- [x] Deleting or manually editing generated output is detected by the drift check.
- [x] README documents where source SQL lives, where generated code lives, how to regenerate, and that source SQL—not generated Go—is the authoring surface.

##### Likely files

```text
sqlc.yaml
db/queries/platform/*.sql
internal/platform/database/<generated-package>/*.go
<root command manifest>
README.md
```

Exact paths may differ, but ownership and source/generated separation must remain clear.

##### Evidence

- Pinned sqlc version output.
- sqlc generation output.
- Generated file list.
- Go compilation/test output.
- Integration result executing at least one generated query.
- Review confirming generated code was not hand-authored and no cross-schema query was introduced.

---

#### 9. User Story 4 — Prove migrations, pgx, and sqlc together

**As the TALLY developer, I want a clean PostgreSQL integration test that applies migrations and executes generated queries through pgx, so that the persistence workflow is proven as one coherent path rather than as disconnected tools.**

##### Value

Provides executable evidence that the selected migration, connection, generation, and query mechanisms work together against PostgreSQL 18.

##### Acceptance criteria

- [x] The integration test uses Testcontainers with PostgreSQL `18.x` or the exact approved production-equivalent major version.
- [x] The test begins from a new database instance or isolated database state.
- [x] The test applies all initialized schema migrations through the same Goose command or library contract used by normal development.
- [x] The test verifies the expected per-schema migration history and latest applied version.
- [x] The test opens pgx using the platform database foundation.
- [x] The test executes at least one committed sqlc-generated query against the delivered technical platform table.
- [x] The test proves a successful transaction commit.
- [x] The test proves rollback or cancellation leaves no established test effect.
- [x] A second migration application is harmless and reports no pending work.
- [x] The test does not depend on a developer's existing Compose volume, host PostgreSQL installation, Azure, or production credential.
- [x] Test data is synthetic, deterministic, and non-sensitive.
- [x] Tests are isolated and can run repeatedly without duplicate or unexplained state.
- [x] Container startup, migration, query, assertion, and cleanup failures propagate non-zero.
- [x] Failure output identifies the phase without printing secrets.
- [x] The integration test is bounded by explicit context or test timeout.
- [x] The test does not claim finance repository behavior, business idempotency, concurrency correctness, recovery qualification, or full `QG-03` completion.

##### Evidence

- Integration-test command and output.
- Resolved PostgreSQL image/version.
- Applied migration versions and history table locations.
- Generated-query execution result.
- Commit and rollback assertions.
- A repeated run showing deterministic success.
- Controlled failure proving non-zero propagation.

---

#### 10. User Story 5 — Detect persistence drift in CI

**As the TALLY developer, I want one local and CI persistence verification command, so that changed migrations or stale generated SQL code cannot be merged unnoticed.**

##### Value

Directly proves the authoritative exit evidence: CI detects migration and generated-code drift.

##### Required verification sequence

The exact command name is an implementation choice. A recommended alias is `db-check` or `persistence-check`.

The check shall perform, in a deterministic order:

1. verify pinned tool availability and versions;
2. validate the migration inventory and Goose migration sets;
3. verify committed migration checksums or the selected equivalent history contract;
4. regenerate sqlc output from committed schema and query source;
5. fail when generation changes, adds, deletes, or leaves untracked generated files;
6. compile and test the generated Go code;
7. apply migrations to clean PostgreSQL 18 state; and
8. run the minimum persistence integration verification.

##### Local acceptance criteria

- [x] One documented root command runs the complete persistence verification.
- [x] It uses repository-pinned Goose and sqlc versions.
- [x] It does not silently skip migration validation, generation, compilation, or integration verification.
- [x] It fails when a committed historical migration changes without the required migration-inventory update.
- [x] It fails when a migration is deleted, duplicated, invalid, or out of order.
- [x] It fails when sqlc source changes but generated code is stale.
- [x] It fails when a generated file is manually changed or deleted.
- [x] It fails when generated code does not compile.
- [x] It fails when clean migration application or the generated query integration test fails.
- [x] It succeeds from a clean checkout with a clean PostgreSQL test dependency.
- [x] A successful run leaves the Git working tree unchanged.
- [x] Failure propagation is non-zero and identifies the failed stage.

##### CI acceptance criteria

- [x] A focused GitHub Actions job or workflow invokes the same repository root persistence command used locally.
- [x] The workflow uses PostgreSQL `18.x` through a service container or the approved Testcontainers path.
- [x] Tool versions come from repository-controlled pins rather than floating CI installation commands.
- [x] The workflow requires no Azure credential, production database, or committed secret.
- [x] The workflow does not print `DATABASE_URL`, database passwords, or complete secret values.
- [x] The job fails on migration drift.
- [x] The job fails on stale or manually changed sqlc output.
- [x] The job fails on migration, compile, or integration-test failure.
- [x] The job succeeds on the reviewed clean branch.
- [x] The job is named clearly enough to be reused or included by the later `DLV-CI-001` full PR pipeline.
- [x] Documentation explicitly states that this focused job does not complete `DLV-CI-001`.

##### Controlled negative evidence

Record evidence for at least these temporary, uncommitted mutations, restoring the clean tree after each check:

1. change a committed migration without updating its inventory/checksum;
2. change a sqlc query without regenerating;
3. manually change or delete generated output; and
4. introduce an invalid migration or generated-code compile error.

Each mutation must make the local check fail. At least migration drift and generated-code drift must also be demonstrated as CI failures through a temporary test commit, workflow-dispatch branch, or equivalent reviewable evidence that is not merged in the failing state.

##### Documentation acceptance criteria

- [x] README documents pinned-tool setup and verification.
- [x] README documents migration creation, validation, status, application, and drift checking.
- [x] README documents sqlc source, generation, generated output, and no-manual-edit rule.
- [x] README documents the persistence integration-test command and PostgreSQL/Docker prerequisites.
- [x] README explains forward-fix versus disposable local reset.
- [x] README explains the `DLV-PLAT-003`/`DLV-CI-001` boundary.
- [x] README does not claim finance schemas, finance repositories, OpenAPI, idempotency, outbox/inbox, production recovery, full `QG-03`, full CI, or `M0` complete.
- [x] A verification record is committed or linked.

##### Evidence record template

```markdown
## DLV-PLAT-003 persistence workflow verification

- Commit:
- Verification date:
- Operating system:
- Go version:
- Docker version:
- PostgreSQL image/version:
- Goose version:
- sqlc version:
- pgx module version:
- Testcontainers module version:
- Initialized migration schemas:
- Migration history tables:
- Migration inventory/checksum method:
- sqlc configuration path:
- Query source paths:
- Generated output paths:
- Root persistence-check command:
- CI workflow/job:
- Commands executed:
  1.
  2.
  3.
- Clean migration result:
- Second migration result:
- Generated query integration result:
- Transaction commit result:
- Transaction rollback result:
- Clean generated-code drift result:
- Migration drift negative test:
- Stale sqlc output negative test:
- Manual generated-file edit/delete negative test:
- Invalid migration/compile negative test:
- CI migration-drift result:
- CI generated-code-drift result:
- Working tree clean after successful check: Yes | No
- Real credentials required: No
- Azure or external services required: No
- Overall result: PASS | FAIL
- Notes:
```

---

#### 11. Delivery-item acceptance summary

`DLV-PLAT-003` is complete only when all five stories and every condition below pass:

- [x] Goose and sqlc tool versions are pinned in repository-controlled manifests.
- [x] The current real platform migration is valid under the Goose workflow.
- [x] Migration sets are schema-owned and every initialized schema has its own history table.
- [x] Clean migration and repeated migration application pass against PostgreSQL 18.
- [x] A deterministic migration inventory/checksum or equivalent mechanism detects drift.
- [x] A minimal pgx database foundation exists under `internal/platform/database`.
- [x] Connection, transaction commit, and rollback behavior pass integration tests.
- [x] sqlc configuration, query source, and generated Go output are committed according to the selected workflow.
- [x] Generated code uses pgx and compiles.
- [x] At least one real generated query executes against the delivered technical platform persistence contract.
- [x] Persistence integration verification begins from clean PostgreSQL state and is repeatable.
- [x] One root command validates migrations, regenerates sqlc, checks the working tree, compiles, migrates, and tests.
- [x] Local negative checks prove migration and generated-code drift detection.
- [x] CI invokes the same persistence command and rejects both migration drift and generated-code drift.
- [x] Documentation and verification evidence are current.
- [x] No finance schema, aggregate table, business repository, API operation, event, or capability behavior is introduced.
- [x] No adjacent delivery item or milestone is falsely marked complete.

#### 12. Explicit exclusions and follow-on ownership

| Excluded work | Owner |
|---|---|
| Full pull-request CI pipeline, branch-gate policy, all-stack quality orchestration | `DLV-CI-001` |
| OpenAPI contract, validation, and generated TypeScript clients | `DLV-PLAT-004` |
| Shared money, currency, accounting-scope, identity, and aggregate-version primitives | `DLV-PLAT-005` |
| Request fingerprint and business idempotency foundation | `DLV-PLAT-006` |
| Outbox, inbox, worker, delivery, retry, and replay foundations | `DLV-PLAT-007` |
| Finance schemas, aggregate tables, constraints, indexes, repositories, and queries | Owning capability delivery items using this workflow |
| Complete schema-role enforcement for all future bounded contexts | Owning schema migrations and later security/database qualification work |
| API database wiring and route-specific readiness behavior | Later approved application integration using the platform database package |
| Domain/application handler tests and finance repository tests | Owning functional delivery items |
| Full architecture import/SQL-ownership test suite | `DLV-CI-001` and applicable capability/database work |
| Structured telemetry, dashboards, and runbooks | `DLV-OPS-001`, `DLV-OPS-002` |
| Terraform, Azure PostgreSQL, networking, Key Vault, and deployment | `DLV-IAC-001`, `DLV-IAC-002` |
| Backup, PITR, restore, RTO/RPO, and production NFR qualification | Later recovery and qualification items |

#### 13. Definition of Ready

- [x] Exact item, epic, milestone, deliverable, and exit evidence are identified.
- [x] `DLV-PLAT-001` root commands and repository layout are stable.
- [x] `DLV-PLAT-002` PostgreSQL 18 lifecycle and reproducible reset pass.
- [x] The existing real migration and technical seed-manifest persistence contract are identified from the repository.
- [x] Exact Goose, pgx, sqlc, and Testcontainers versions can be selected and pinned.
- [x] Schema-owned migration and generated-code paths can be chosen without inventing finance modules.
- [x] The migration drift method is selected.
- [x] The focused CI boundary with `DLV-CI-001` is understood.
- [x] Positive, negative, clean-database, repeated-run, and CI evidence is identified.
- [x] Five stories are small enough for one or a short chain of reviewable changes.

Finance requirements, authoritative finance records, UX states, authorization decisions, correction paths, business concurrency, integration events, and production recovery are not applicable.

#### 14. Definition of Done

- [x] Every criterion in Sections 6–11 passes; none is silently deferred.
- [x] Tool pins, migrations, migration inventory/checksums, pgx foundation, sqlc source/configuration, generated code, tests, root commands, focused CI, and documentation are complete.
- [x] Go dependency manifests are tidy and committed.
- [x] Goose migration validation and clean application pass.
- [x] Repeated migration application is harmless.
- [x] Migration history is schema-owned for every initialized schema.
- [x] pgx connection, commit, rollback, and cleanup tests pass.
- [x] sqlc generation is deterministic and generated code compiles.
- [x] PostgreSQL 18 integration verification passes from clean state and on repetition.
- [x] The root persistence check passes and leaves a clean working tree.
- [x] Controlled local migration-drift and generated-code-drift checks fail as expected.
- [x] Focused CI rejects migration drift and generated-code drift and passes on the clean branch.
- [x] Documentation matches actual versions, commands, file locations, migration behavior, generated-code policy, and CI job.
- [x] No real credential, production data, build output, test artifact, or developer-specific config is committed.
- [x] The diff is reviewed against modular-monolith, schema ownership, query ownership, and adjacent-item boundaries.
- [x] Traceability is current.
- [x] No functional requirement, workflow, NFR qualification, full `QG-03`, `DLV-CI-001`, adjacent delivery item, or `M0` milestone is incorrectly marked complete.
- [x] No critical or high unresolved defect remains.

##### General Definition-of-Done applicability

| Control | Applicability |
|---|---|
| Domain/application behavior | Not applicable; no finance operation is implemented. |
| Database migrations | Mandatory for the initialized technical schema and migration workflow. |
| Repository integration tests | A minimal platform persistence integration test is mandatory; complete capability repository suites remain later work. |
| OpenAPI | Not applicable; owned by `DLV-PLAT-004`. |
| Authorization/audit | No protected finance action exists; secret hygiene is mandatory. |
| Business idempotency/concurrency/corrections | Not completed; transaction commit/rollback is technical workflow proof only. |
| Accessibility/localization | Not applicable. |
| Telemetry | Not completed; command, test, and CI output must avoid credentials. |
| Recovery/DR | Forward-fix policy is documented; backup/restore and DR are not completed. |
| Clean-environment demonstration | Mandatory through clean PostgreSQL integration and focused CI evidence. |

#### 15. Traceability and quality-gate contribution

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-003` |
| Direct FR IDs | None |
| Direct GFR IDs completed | None |
| Workflow IDs | None |
| NFR qualification | None |
| Primary contribution | `QG-03` through the persistence toolchain, migration verification, sqlc generation, and minimum integration test; `QG-10` through clean local and CI evidence. |
| Partial-only warning | This item does not complete full `QG-03` for all future schemas, constraints, locks, indexes, repositories, backfills, or recovery paths. |
| Exit evidence | CI detects migration and generated-code drift. |

#### 16. Consistency review record

##### Review pass 1 — Corrected

The first semantic review corrected:

- a possible conflict between `DLV-PLAT-003`'s CI exit evidence and `DLV-CI-001` ownership by limiting this item to a focused persistence CI invocation;
- premature creation of all 19 schemas and migration histories before their owning capability work;
- invented finance tables or demo queries merely to exercise sqlc;
- migration rollback wording that could imply destructive shared-environment rollback is normal;
- a generic repository or unit-of-work abstraction not required by the approved pgx/sqlc design;
- possible API startup/readiness wiring that would broaden the story beyond the persistence workflow;
- dependence on globally installed unpinned Goose or sqlc versions;
- drift checks that only regenerated code but did not fail on changed/deleted/untracked output; and
- a false claim that this item completes full `QG-03`, full CI, production persistence, or `M0`.

##### Review pass 2 — Passed

Structural and semantic checks passed for:

- exact deliverable and exit evidence;
- five small stories with value, criteria, likely files, and evidence;
- approved PostgreSQL 18, Goose, pgx, sqlc, Go, and Testcontainers baselines;
- one database and schema-per-context ownership;
- per-initialized-schema Goose history without speculative future schemas;
- current technical seed-manifest reuse without invented finance facts;
- pinned and deterministic tooling;
- clean migration, repeat migration, generation, compilation, and integration proof;
- local and CI negative drift evidence;
- `DLV-PLAT-002`, `DLV-CI-001`, and future capability boundaries;
- Definition-of-Ready and Definition-of-Done applicability; and
- quality-gate contribution without false completion.

##### Review pass 3 — Passed

The final review confirmed:

- no unapproved requirement, identifier, API, event, finance schema, aggregate table, or business record was invented;
- no floating-point monetary representation is introduced;
- historical migrations are protected by a deterministic drift contract;
- generated sqlc code is reproducible, compiled, and not manually edited;
- pgx remains the sole low-level access path and no ORM is introduced;
- queries remain schema-owned and cannot establish cross-context write shortcuts;
- shared-environment recovery remains forward-fix by default;
- secrets are excluded from source, logs, tests, and CI output;
- the focused persistence CI check proves the exact delivery-item exit evidence while leaving the full PR pipeline to `DLV-CI-001`; and
- completion means the user-story specification is internally consistent, not that implementation code has already passed.

**Final consistency result: PASS.**

#### 17. Source references

- `02_work_breakdown_backlog_v1.0.md` / `finance_delivery_plan_v1.0.md` — exact `DLV-PLAT-003` deliverable and exit evidence, quality gates, Definition of Done, and CI/local environment policy.
- `01_solution_architecture_overview_v1.0.md` — modular monolith, PostgreSQL 18, Goose, pgx, sqlc, Testcontainers, and local-first technology baseline.
- `05_architecture_traceability_decisions_v1.0.md` — ADR-005 through ADR-007 for PostgreSQL, one database/schema-per-context, and pgx/sqlc instead of an ORM.
- `01_backend_module_specifications_v1.0.md` — `internal/platform/database`, `db/migrations`, module dependency rules, transaction contract, configuration, and SQL ownership tests.
- `03_data_integration_architecture_v1.0.md` — PostgreSQL authority, schema ownership, exact-decimal requirement, migration discipline, and no cross-context writes.
- `03_database_persistence_specifications_v1.0.md` — PostgreSQL topology, one Goose history table per schema, migration policy, constraints, and role-verification expectations.
- `09_testing_performance_recovery_specifications_v1.0.md` — PostgreSQL 18/Testcontainers repository integration tests and release requirement that sqlc output and migrations have no drift.
- `10_technical_traceability_decisions_v1.0.md` — table/constraint/index changes update migrations, sqlc queries, evidence, and traceability.
- `EP-PLAT-001_user_stories.md` — established boundary assigning Goose layout/history, pgx, sqlc, generated SQL, and drift checks to `DLV-PLAT-003`.

### DLV-PLAT-004 — OpenAPI-First REST Workflow User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-004` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | Complete |
| Dependency position | Builds on `DLV-PLAT-001` and `DLV-PLAT-003`; no finance capability implementation is required. |
| Exit evidence | OpenAPI 3.1 validates and bundles; generated Go and TypeScript artifacts compile; focused CI detects contract or generated-artifact drift. |

#### 1. Purpose and scope

**As the TALLY developer, I want one version-controlled OpenAPI contract to drive the HTTP boundary and generated client artifacts, so that API consumers and future finance modules share a reviewable, compatible contract.**

This item establishes the OpenAPI-first workflow. It may use a minimal technical health or shell operation to prove generation, but must not implement finance commands, repositories, authorization policy, business idempotency, or domain events.

#### 2. Approved boundaries

- The contract format is OpenAPI 3.1 and REST/JSON.
- Contract sources live under `contracts/openapi/`, with shared components, capability paths, and examples separated according to the API specification.
- The contract owns transport schemas, operation IDs, required headers, problem-details errors, pagination conventions, and HTTP result rules.
- Generated Go server/types and TypeScript client/types are derived artifacts and are not hand-edited.
- Money uses the API specification's exact-decimal representation; examples and logs contain no secrets or real financial data.

#### 3. Explicit exclusions

- Finance endpoint behavior, domain commands, aggregates, repositories, database wiring, or business events.
- Authentication, authorization, accounting-scope enforcement, audit evidence, and business idempotency (`DLV-PLAT-006`).
- Integration events, outbox/inbox, workers, retries, and replay (`DLV-PLAT-007`).
- Full pull-request quality orchestration (`DLV-CI-001`) beyond the focused contract drift check.
- Frontend routing, forms, design-system components, or capability screens (`EP-UX-001`).

#### 4. User stories

##### User Story 1 — Establish the contract layout and common schemas

**As the TALLY developer, I want a maintainable OpenAPI source layout and common schemas, so that future capabilities extend the API without duplicating transport rules.**

- [x] An OpenAPI 3.1 root document exists under `contracts/openapi/` and references committed component and path files.
- [x] Common headers, command requests, established results, pagination/query conventions, and RFC 9457-style problem details are represented consistently.
- [x] Examples are deterministic and contain no credentials or real personal/financial data.
- [x] Every operation has a unique, stable `operationId` and an identified owning capability boundary.

##### User Story 2 — Validate and bundle the OpenAPI contract

**As the TALLY developer, I want pinned, repeatable contract validation, so that malformed references and incompatible API changes fail before review.**

- [x] The repository pins the selected OpenAPI linter/bundler and exposes a root validation command.
- [x] Validation covers all referenced files and rejects broken references, duplicate operation IDs, invalid schemas, and malformed examples.
- [x] A negative fixture proves an invalid contract fails with a non-zero result and no secret output.
- [x] Bundling or equivalent verification output is deterministic.

##### User Story 3 — Generate and verify Go API artifacts

**As the Go API developer, I want generated server interfaces and transport types from the contract, so that future handlers conform to the reviewed HTTP boundary.**

- [x] A pinned generator configuration produces Go artifacts from the committed contract.
- [x] Generated Go code compiles under the normal Go verification command and has a generated-file marker.
- [x] Generation is deterministic and does not require a globally installed unpinned tool.
- [x] A minimal compile or shell integration references the generated boundary without introducing finance behavior.

##### User Story 4 — Generate and verify the TypeScript client

**As the React developer, I want generated TypeScript API types/client functions, so that the frontend cannot silently diverge from the backend contract.**

- [x] A pinned generator configuration produces TypeScript client/types from the same OpenAPI source.
- [x] Generated TypeScript output compiles with the pnpm-managed frontend verification command.
- [x] The client preserves operation IDs, response/error types, exact-decimal transport values, and nullable/optional distinctions.
- [x] Generated output is clearly marked and no frontend screen is required to complete this item.

##### User Story 5 — Detect contract and generated-artifact drift

**As the TALLY maintainer, I want local and focused CI checks to detect contract and generated-code drift, so that committed artifacts cannot become stale.**

- [x] One documented root command validates the contract, regenerates both language artifacts, checks compilation, and fails when regeneration changes tracked output.
- [x] Local negative verification proves the check fails for stale output and passes after regeneration.
- [x] Focused CI invokes the same command on a clean checkout.
- [x] The workflow clearly leaves full API compatibility, security, frontend, SQL, Terraform, and documentation gates to their owning items or `DLV-CI-001`.

#### 5. Definition of Ready

- [x] The OpenAPI 3.1 specification and exact endpoint catalog are available as the source of truth.
- [x] Contract directory, generated output locations, and toolchain versions are agreed and documented.
- [x] Boundaries with `DLV-PLAT-003`, `DLV-PLAT-005`–`007`, `EP-UX-001`, and `DLV-CI-001` are preserved.
- [x] Five stories are small enough for one or a short chain of reviewable changes.

#### 6. Definition of Done

- [x] All five stories and acceptance criteria pass.
- [x] OpenAPI validation and bundling are reproducible from a clean checkout.
- [x] Generated Go and TypeScript artifacts compile and remain synchronized with the contract.
- [x] Local and focused CI drift checks pass, including controlled negative checks.
- [x] Documentation identifies commands, tool pins, source locations, generated locations, and ownership boundaries.
- [x] No finance capability, protected action, business event, or adjacent delivery item is falsely marked complete.

#### 7. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-004` |
| Direct FR/GFR IDs | None; this item establishes transport tooling rather than finance behavior. |
| Quality contribution | API/integration-contract foundation; partial `QG-04` and clean-generation evidence for `QG-10`. |
| Exit evidence | OpenAPI validates; generated Go and TypeScript artifacts compile; focused CI rejects contract/generated-artifact drift. |

#### 8. Source references

- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md`
- `docs/specs/finance_delivery_plan_v1.0.md`
- `docs/specs/technical_specifications/10_technical_traceability_decisions_v1.0.md`
- `docs/backlog/stories/EP-PLAT-001_user_stories.md`

### DLV-PLAT-005 — Shared Finance Primitives User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-005` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | Complete |
| Dependency position | Builds on `DLV-PLAT-001` and `DLV-PLAT-003`; may use the transport conventions from `DLV-PLAT-004`. No finance capability implementation is required. |
| Exit evidence | Shared primitives have unit and serialization tests; money uses exact decimal arithmetic; invalid precision, scope, identity, and version values are rejected deterministically. |

#### 1. Purpose and scope

**As the TALLY developer, I want shared finance primitives with explicit invariants, so that finance modules represent money, currency, accounting scope, identity, and optimistic-concurrency versions consistently.**

This item establishes reusable domain-neutral primitives required by later finance capabilities. It does not implement a finance aggregate, command handler, repository, endpoint behavior, authorization policy, or integration workflow.

#### 2. Approved boundaries

- Money must not use binary floating point. Arithmetic and serialization preserve exact decimal meaning and currency scale rules.
- Currency is an explicit ISO currency identity with deterministic validation and supported precision metadata.
- `AccountingScope` explicitly contains tenant, legal entity, ledger, accounting book, and functional currency identifiers; scope is never inferred from ambient request context.
- Stable identifiers are opaque, validated values with type-safe construction where practical; they do not encode business state.
- Aggregate versions support expected-version checks and monotonic advancement for optimistic concurrency.
- Established financial facts remain immutable; these primitives do not provide destructive mutation helpers.
- Shared packages contain technical/domain primitives only and do not own finance module behavior or persistence schemas.

#### 3. Explicit exclusions

- Finance aggregates, domain commands, domain events, repositories, migrations, or capability-specific schemas.
- Request fingerprints and business idempotency (`DLV-PLAT-006`).
- Outbox, inbox, workers, retries, delivery, and replay (`DLV-PLAT-007`).
- Authentication, authorization, audit evidence, or accounting-scope policy decisions.
- Currency-rate mastering, revaluation, translation, or FX calculations owned by Multi-Currency.
- API endpoint behavior or frontend screens.

#### 4. User stories

##### User Story 1 — Implement exact-decimal money and currency primitives

**As a finance module developer, I want exact-decimal money and validated currency values, so that calculations cannot silently lose monetary precision.**

- [x] Money construction rejects malformed decimal values and values that violate the agreed currency precision policy.
- [x] Money arithmetic is deterministic, preserves currency identity, and rejects incompatible currencies or invalid results.
- [x] Currency values validate the agreed canonical representation and expose the precision policy selected in Definition of Ready.
- [x] Serialization uses the exact-decimal representation selected in Definition of Ready and never converts money through binary floating point.
- [x] Tests cover zero, permitted negative values, precision boundaries, rounding boundaries, and incompatible currencies.

##### User Story 2 — Implement explicit accounting-scope identity

**As a finance module developer, I want an explicit accounting scope, so that ledger-bound facts cannot be attributed to an inferred or incomplete scope.**

- [x] `AccountingScope` contains tenant, legal entity, ledger, accounting book, and functional currency identifiers.
- [x] Construction rejects missing or malformed components; cross-component relationship consistency remains with the owning contexts.
- [x] Equality and serialization include every scope component, including functional currency.
- [x] The primitive does not read ambient tenant, request, session, or authentication context.
- [x] Tests prove distinct ledgers or accounting books remain distinct even under the same legal entity.

##### User Story 3 — Implement stable identity primitives

**As a finance module developer, I want validated stable identifiers, so that records and references remain unambiguous across module boundaries.**

- [x] `AggregateID`, `CorrelationID`, and `CausationID` have validated UUID construction and canonical serialization in `internal/platform/identity`.
- [x] Empty, malformed, nil, and cross-type identifiers are rejected deterministically; command idempotency remains distinct and is deferred to `DLV-PLAT-006`.
- [x] Identifier equality is value-based and does not depend on display formatting.
- [x] Identifiers do not carry mutable lifecycle state or infer ownership outside their declared type.
- [x] Tests cover round-trip serialization, equality, invalid input, stable sentinel errors, UUID v7 generation, and compile-time type distinction.

##### User Story 4 — Implement aggregate version primitives

**As a finance module developer, I want explicit aggregate versions, so that concurrent changes can be detected without overwriting an established outcome.**

- [x] A new aggregate starts at `AggregateVersion(1)`; the typed domain range is `1..math.MaxInt64`, while persistence and transport remain explicit `int64`/`bigint` boundaries.
- [x] Version advancement is monotonic and rejects overflow or invalid transitions.
- [x] Expected-version comparison distinguishes a matching version from a stale version.
- [x] JSON serialization round-trips integer versions without string conversion; `FromInt64` and `Value` provide explicit boundary conversion.
- [x] Tests cover initial, matching, stale, advancement, invalid, maximum, overflow, JSON, and explicit-conversion boundary values.

##### User Story 5 — Prove serialization and boundary behavior

**As the TALLY maintainer, I want focused verification for the shared primitives, so that later modules can adopt them without duplicating incompatible rules.**

- [x] Unit tests cover all primitive invariants and negative cases.
- [x] Serialization tests cover the exact Go boundary and API representation selected in Definition of Ready, without changing the OpenAPI contract unnecessarily.
- [x] Tests prove no binary floating-point money representation is used in implementation or serialized output.
- [x] Package ownership and import checks show shared primitives do not depend on finance bounded-context adapters or schemas.
- [x] A documented root command runs the focused verification reproducibly.

#### 5. Definition of Ready

- [x] User Story 1 uses `shopspring/decimal`, caller-provided immutable currency metadata, configured currency scale from `0..12`, and the PostgreSQL-compatible `numeric(38,12)` domain ceiling.
- [x] User Story 1 is implemented in `internal/platform/money` with explicit constructors, accessors, arithmetic methods, stable errors, and canonical amount-text serialization. The API continues to represent amount and currency separately.
- [x] User Story 2 uses UUID components, a canonical uppercase three-letter functional-currency code, structural-only validation, and an exact JSON round-trip contract in `internal/platform/accountingscope`.
- [x] User Story 3 uses typed UUID identities in `internal/platform/identity`: application-created aggregate identities use UUID v7; correlation and causation identities validate existing UUID fields; JSON is canonical lowercase UUID text; `ErrNilID`, `ErrMalformedID`, and `ErrInvalidJSON` are stable error contracts.
- [x] User Story 4 uses `internal/platform/aggregateversion.AggregateVersion` with initial value `1`, valid range `1..math.MaxInt64`, stable validation/overflow/JSON errors, and explicit `FromInt64`/`Value` conversion at persistence and transport boundaries. The OpenAPI integer contract is unchanged.
- [x] Boundaries with `DLV-PLAT-004`, `DLV-PLAT-006`, `DLV-PLAT-007`, and finance capability items are preserved.
- [x] Five stories are small enough for one or a short chain of reviewable changes.

#### 6. Definition of Done

- [x] All five stories and acceptance criteria pass.
- [x] User Story 1 focused unit, serialization, and boundary tests pass.
- [x] User Story 1 money has no binary floating-point implementation or serialization path.
- [x] User Story 1 documentation identifies the primitives, invariants, command, and ownership boundary.
- [x] User Story 2 focused unit, serialization, boundary, and package-ownership tests pass.
- [x] User Story 2 documentation identifies the representation, structural-validation boundary, command, and ownership boundary.
- [x] User Story 3 focused unit, serialization, type-safety, and package-ownership tests pass.
- [x] User Story 3 documentation identifies the identity types, UUID-version policy, serialization/error contract, idempotency boundary, and ownership boundary.
- [x] User Story 4 focused unit, JSON, explicit-conversion, type-safety, and package-ownership tests pass.
- [x] User Story 4 documentation identifies the typed representation, valid range, initial value, conversion boundary, serialization/error contract, and ownership boundary.
- [x] User Story 5 focused verification covers all primitive suites, serialization boundaries, float-free money, package ownership, and unchanged OpenAPI artifacts.
- [x] User Story 5 documentation identifies the shared verification command, representations, boundary contracts, and ownership checks.
- [x] No finance capability, idempotency behavior, integration workflow, or adjacent delivery item was marked complete.

#### 7. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-005` |
| Direct FR/GFR IDs | Contributes to `GFR-001` and `GFR-008`; this item does not complete either control or `GFR-013`. |
| Quality contribution | Shared type-safety, precision, serialization, and optimistic-concurrency foundation. |
| Exit evidence | Unit and serialization tests pass without floating-point money; invalid primitive and boundary cases are covered. |

#### 8. Source references

- `docs/specs/finance_domain_model_ddd.md` — `AccountingScope`, money/currency precision, identity, and optimistic concurrency rules.
- `docs/specs/finance_delivery_plan_v1.0.md` — platform foundation backlog and DLV-PLAT-005 exit evidence.
- `docs/specs/technical_specifications/01_backend_module_specifications_v1.0.md`
- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md`
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md`
- `docs/specs/technical_specifications/06_security_identity_authorization_specifications_v1.0.md`
- `docs/specs/technical_specifications/10_technical_traceability_decisions_v1.0.md`
- `docs/backlog/stories/EP-PLAT-001_user_stories.md`

### DLV-PLAT-006 — Request Fingerprint and Idempotency Foundation User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-006` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | DLV-PLAT-006 platform foundation implemented; focused evidence passes, with the environment-dependent persistence gate pending |
| Dependency position | Builds on the shared identity and accounting-scope primitives from `DLV-PLAT-005`; provides a foundation for capability handlers and the outbox/inbox work in `DLV-PLAT-007`. |
| Exit evidence | Canonical fingerprint, durable reservation, same-content retry, changed-content conflict, transactional metadata finalization, concurrent ownership, boundary, and architecture evidence pass; Docker/SQLC persistence verification remains an environment-dependent gate. |

#### 1. Purpose and scope

**As the TALLY platform maintainer, I want a scoped request-fingerprint and idempotency foundation, so that retried commands return established outcomes and cannot repeat or change a business effect.**

This item defines reusable platform behavior and contracts. It does not own a
finance aggregate or capability workflow. Owning-capability integration proves
how a finance transaction uses this foundation and is tracked by the relevant
capability delivery item.

The coordination-contract and durable persistence implementations provide
platform ownership, established-result lookup, terminal-result finalization,
lease handling, and database-level concurrency behavior. They do not claim
exactly-once financial/business effects, authorization policy, audit policy,
or a specific owning-capability transaction.

#### 2. Approved boundaries

- The foundation package is `internal/platform/idempotency`.
- A canonical request fingerprint represents functional command content, using a deterministic canonicalization and stable cryptographic digest. Equivalent functional content produces the same fingerprint; material content changes produce a different fingerprint.
- The business idempotency identity is scoped by the applicable `AccountingScope` and command/business identity. It is distinct from aggregate version, event identity, correlation identity, and causation identity.
- For the same scope, identity, and fingerprint, the established in-progress or terminal command result is returned without repeating the business effect.
- Reusing an identity with a different canonical fingerprint returns `IDEMPOTENCY_CONFLICT` and produces no business effect.
- Idempotency metadata has an explicit durable reservation/finalization transaction boundary. An owning capability must later persist its business change and terminal result together.
- Stored metadata includes the scoped identity, canonical fingerprint, lifecycle/result status, and the stable result reference or response metadata needed to return the established result; it does not own finance state.
- Shared platform code provides technical coordination only. The owning finance bounded context remains responsible for authorization, audit, validation, aggregate invariants, and business effects.
- Outbox, inbox, workers, delivery retries, and replay orchestration remain deferred to `DLV-PLAT-007`.

#### 3. Explicit exclusions

- Finance aggregates, capability commands or handlers, authorization policies, audit policy, migrations, or finance-owned persistence schemas.
- Event identity deduplication or consumer inbox behavior.
- External provider idempotency, delivery scheduling, retry policy, or replay tooling.
- Frontend screens or changes to existing OpenAPI idempotency-header contracts.
- Owning finance transactions, journal effects, authorization policy, audit persistence, and finance-level exactly-once proof.

#### 4. User stories

##### User Story 1 — Define canonical request fingerprinting

**As a platform and finance module developer, I want functional request content canonicalized deterministically, so that semantically equivalent retries can be recognized.**

- [x] Canonicalization defines included functional fields, excluded transport/non-functional fields, ordering, normalization, and representation for supported values.
- [x] Equivalent functional content produces byte-identical canonical input and the same fingerprint.
- [x] Material changes to functional content produce a different fingerprint.
- [x] Fingerprint serialization is stable and suitable for persistence, comparison, logging, and audit references without exposing unnecessary sensitive request content.
- [x] Tests cover field ordering, omitted versus explicit values where applicable, Unicode/encoding, nested collections, nullability, and boundary-sized requests.

##### User Story 2 — Define scoped idempotency identity and stored command-result metadata

**As a platform developer, I want a validated scoped idempotency identity and result record, so that distinct business scopes and command outcomes cannot be confused.**

- [x] Missing, malformed, empty, or otherwise invalid identities are rejected deterministically with stable validation behavior.
- [x] Identity equality includes the applicable accounting scope and business/command identity; identical text in different scopes remains distinct.
- [x] Documentation and types distinguish business identity/fingerprint from aggregate version, event identity, correlation ID, and causation ID.
- [x] Stored result metadata can represent in-progress and terminal outcomes and retains the canonical fingerprint and stable result reference/response metadata.
- [x] Result metadata does not mutate or replace an established financial fact and does not make the platform package the owner of a finance aggregate.

##### User Story 3 — Coordinate established results for identical retries

**As a platform maintainer, I want the foundation to coordinate identical retries, so owning command handlers can safely return established outcomes.**

- [x] The first accepted identity/fingerprint establishes one durable result record and one execution owner.
- [x] A repeat with the same scope, identity, and fingerprint returns the existing in-progress or terminal result.
- [x] The platform coordinator does not issue a second owner for an identical retry.
- [x] An ambiguous or in-progress result remains discoverable through the established identity and is never presented as a false success.
- [x] Tests cover retry before completion, retry after success, and retry after a terminal failure or rejection.

###### Coordination-contract prerequisite delivered

- [x] First execution ownership, same-fingerprint established-result lookup, terminal-result transition, and ambiguous/in-progress visibility are covered by `internal/platform/idempotency` tests.
- Owning-transaction integration and exactly-once business-effect protection remain follow-up capability scope.

##### User Story 4 — Reject changed content at the platform boundary

**As a platform maintainer, I want changed content under an existing identity rejected, so that one business identity cannot establish conflicting outcomes.**

- [x] A same-scope identity with a different canonical fingerprint returns `ErrIdempotencyConflict` using the stable platform error meaning.
- [x] The platform conflict contains no request content and does not disclose protected data.
- [x] A conflict does not update stored result metadata or create a second owner.
- [x] The platform contract directs callers to preserve the identity and fingerprint pairing rather than overwrite it.
- [x] Tests cover material content changes and verify no platform state change.

The owning transport adapter remains responsible for mapping this platform
error to the existing `IDEMPOTENCY_CONFLICT` HTTP 409 contract.

##### User Story 5 — Prove transactional, concurrent, and boundary behavior

> User Story 5 is complete for the platform foundation. Cross-process recovery,
> owning finance effects, authorization, audit integration, and finance-level
> exactly-once behavior remain capability follow-up scope.

**As the TALLY maintainer, I want focused verification of the idempotency foundation, so that owning capabilities have a durable and concurrency-safe platform contract.**

- [x] Concurrent first submissions for one scoped identity establish at most one platform owner; losers observe the established, in-progress, or deterministic conflict result.
- [x] Idempotency reservation and terminal metadata finalization have explicit transaction boundaries; rollback leaves no false terminal result and lease recovery is guarded by owner tokens.
- [x] Tests prove same-content retry, changed-content conflict, malformed identity, fingerprint boundaries, and terminal/in-progress result behavior.
- [x] Package and architecture checks show finance modules and adapters do not own shared idempotency behavior or access another module's schema directly.
- [x] Existing OpenAPI idempotency-header contracts remain unchanged and documented root verification commands run the focused checks reproducibly.

#### 5. Definition of Done

- [x] The platform slices of all five stories and their acceptance criteria pass.
- [x] Canonicalization and fingerprint tests prove equivalent content stability and material-change distinction.
- [x] Identity validation, scope separation, result metadata, retry, conflict, transaction, concurrency, lease, and boundary evidence pass.
- [x] The platform does not create a second owner for an identical retry, and changed content creates no second platform effect.
- [x] Package ownership, architecture, and unchanged OpenAPI checks pass.
- [x] Owning finance effects, authorization policy, audit persistence, frontend behavior, and outbox/inbox workflow remain outside this delivery item.

The Docker-backed integration and pinned SQLC drift checks are release-gate
verification and are not marked as passed until they run successfully.

#### 6. Traceability

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Delivery item | `DLV-PLAT-006` |
| Direct requirement IDs | Contributes to `GFR-006` and `GFR-007`; supports `GFR-008` and the immutable lineage expectations of `GFR-013`. |
| Workflow IDs | `WF-7.13`, `WF-7.14` |
| Quality contribution | Deterministic retry, conflict safety, transaction atomicity, and concurrency safety for later command handlers. |
| Exit evidence | Same-content, changed-content, transactional metadata, concurrent ownership, boundary, package-boundary, and contract-preservation checks pass. |

#### 7. Source references

- `docs/specs/finance_delivery_plan_v1.0.md` — platform foundation backlog and GFR mappings.
- `docs/specs/finance_domain_model_ddd.md` — command fingerprint, duplicate delivery, concurrency, recovery, and immutable-fact rules.
- `docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md` — `GFR-006`, `GFR-007`, and `GFR-008`.
- `docs/specs/prd/01_finance_functional_prd_v1.5.md` — `WF-7.13` and `WF-7.14`.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — safe repeat and identity-content conflict behavior.
- `docs/specs/technical_specifications/01_backend_module_specifications_v1.0.md` — `IDEMPOTENCY_CONFLICT` error contract and transaction boundary.
- `docs/specs/technical_specifications/02_api_openapi_specifications_v1.0.md` — existing idempotency-header contract.
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md` — local transaction and persistence conventions.
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md` — concurrency and recovery verification expectations.
- `docs/backlog/stories/EP-PLAT-001_user_stories.md` — shared identity and accounting-scope foundation boundary.

### DLV-PLAT-007 — PostgreSQL Outbox/Inbox and Worker Foundation User Stories


| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-007` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Status | Complete — User Stories 1–5 platform foundation criteria are verified; semantic payload safety is a separate deferred follow-up |
| Dependency position | Builds on `DLV-PLAT-003` persistence conventions and `DLV-PLAT-006` idempotency coordination; provides the integration-delivery foundation for later bounded contexts. |
| Exit evidence | Versioned event contracts, transactional outbox/inbox persistence, lease-safe dispatch, retry and poison-work handling, worker lifecycle behavior, crash recovery, duplicate-delivery, ordering, and replay tests pass. |

#### 1. Outcome

**As the TALLY platform maintainer, I want durable outbox/inbox and worker foundations, so that integration events survive process failure, are delivered at least once, and establish at most one local business effect per receiving context.**

This delivery item establishes the PostgreSQL-backed integration transport and
worker contracts used by later bounded contexts. The owning source and receiving
contexts remain responsible for their domain effects, authorization, audit
evidence, and business invariants.

#### 2. Learning objective

Learn how a modular monolith preserves transactional event publication,
consumer deduplication, lease ownership, retry safety, and replay evidence
without introducing an external broker.

#### 3. Scope

- Define the versioned integration-event envelope and minimal payload contract.
- Create the PostgreSQL `integration.outbox` and `integration.inbox` persistence foundation.
- Define atomic source-side publication and receiving-side consumption boundaries.
- Implement the initial outbox dispatcher and worker-host lifecycle contracts.
- Prove duplicate delivery, crash recovery, ordering, retry, lease fencing, and replay behavior.

#### 4. Explicit exclusions

- Finance aggregates, capability commands, receiving-context domain effects, or accounting ownership.
- Authentication, authorization policy, audit policy, or user-facing workflow screens.
- Provider-specific payment, bank, tax, payroll, or document workers.
- External brokers such as Azure Service Bus, Kafka, or RabbitMQ; adoption requires a separate ADR.
- Operational dashboards and runbooks owned by `DLV-OPS-001` and `DLV-OPS-002`.
- Full finance-level exactly-once proof, which belongs to the owning capability integrations.

#### 5. Owning bounded context & architecture

- **Owning bounded context / module:** Platform integration adapter; source and receiving bounded contexts own business interpretation and effects.
- **Persistence ownership:** `integration.outbox` and `integration.inbox` are platform integration schemas. No finance module may access another module's adapter or schema directly.
- **Initial worker host:** `cmd/worker` hosts all workers initially, with independent concurrency limits, database-pool budgets, shutdown deadlines, and metrics namespaces.
- **Transport:** PostgreSQL durable state is the initial transport. Registered in-process consumers and approved external adapters are invoked by the dispatcher.

#### 6. User stories

##### User Story 1 — Define the versioned event envelope and safe payload contract

**As a platform and bounded-context developer, I want a stable event envelope, so that every published event carries enough identity and lineage for validation, delivery, deduplication, and replay.**

- [x] The platform envelope defines the canonical identity, lineage, classification, fingerprint, and object-payload wire fields.
- [x] Event identity, semantic contract version, source aggregate version, scope, correlation, causation, and payload fingerprint are distinct concepts.
- [x] Unknown contract versions, invalid scope, malformed identity, invalid timestamps, invalid payloads, and fingerprint mismatches produce explicit structural validation outcomes without changing domain state.
- [x] Serialization and fingerprinting are deterministic and preserve the original event identity and stored fingerprint across round-trip/replay-shaped tests.

###### Deferred follow-up — event-specific payload safety

Semantic payload minimization, event-specific schemas, sensitive-data allowlists, and semantic-transformation validation remain deferred outside DLV-PLAT-007. They are not implemented or claimed by this platform foundation.

##### User Story 2 — Persist durable PostgreSQL outbox and inbox records

**As a platform developer, I want durable outbox and inbox records, so that event publication and consumer identity survive process restart and database recovery.**

- [x] A migration creates `integration.outbox` with event identity, source aggregate/version, scope, lineage references, payload, fingerprint, availability, claim, attempt, error, and establishment fields.
- [x] A migration creates `integration.inbox` with consumer/message identity, message fingerprint, processing state, result reference, receipt time, and establishment time.
- [x] Outbox uniqueness prevents duplicate publication for the same source context, aggregate, aggregate version, and event type.
- [x] Inbox primary-key identity is `(consumer_name, message_id)`; state values are limited to `processing`, `established`, and `failed`.
- [x] Queries support due-outbox selection, lease expiry, inbox reconciliation, and scoped duplicate lookup without bypassing schema ownership.

##### User Story 3 — Coordinate transactional publication and consumption

**As an owning-context developer, I want explicit transaction boundaries, so that acknowledged local effects cannot be separated from their integration evidence.**

- [x] The source business effect and its outbox record commit in one database transaction.
- [x] A receiving inbox record, receiving local effect, and any resulting outbox records commit in one database transaction.
- [x] A same-fingerprint duplicate delivery returns the established inbox result and repeats no local business effect.
- [x] A different fingerprint for the same consumer/message identity returns an identity-content conflict, preserves existing evidence, and raises an integrity outcome.
- [x] A processing or failed inbox item remains discoverable and does not fabricate success; reconciliation checks the established local result before retrying.
- [x] Platform integration code does not become the owner of finance aggregates, accounting effects, authorization, or audit policy.

##### User Story 4 — Dispatch due outbox work with leases and typed retries

**As a platform operator, I want lease-safe dispatch and controlled retries, so that multiple workers can process due events without stale workers overwriting newer outcomes.**

- [x] Claiming selects due, unestablished records using `FOR UPDATE SKIP LOCKED`, orders by availability and outbox identity, and assigns an owner token and lease.
- [x] The default claim duration is 30 seconds, renewal occurs before two-thirds of lease consumption, and the duration remains longer than the measured p99 handler duration.
- [x] Establishment and rescheduling require the current claim owner; an expired or superseded worker cannot overwrite a newer claim.
- [x] Only typed transient dependency failures are retried, using the default delays of 5 seconds, 30 seconds, 2 minutes, 10 minutes, and 30 minutes unless an approved adapter policy overrides them.
- [x] Domain rejection, authorization denial, idempotency conflict, and data-integrity mismatch are not automatically retried.
- [x] Work that reaches ten failed attempts becomes a managed exception with retained evidence and is never silently deleted.

##### User Story 5 — Prove worker lifecycle, crash recovery, duplicate delivery, and replay

**As the TALLY maintainer, I want focused integration verification, so that the platform can recover safely without duplicate effects or lost event evidence.**

- [x] The worker host starts and stops workers with independent limits, pool budgets, metrics namespaces, and a bounded shutdown deadline.
- [x] Tests cover crashes before source commit, after source commit, before consumer establishment, and after consumer local-effect processing.
- [x] Tests cover concurrent claims, lease expiry, stale-worker fencing, worker restart, retry exhaustion, and managed poison work.
- [ ] Capability-owned tests cover duplicate, out-of-order, delayed, unknown-version, invalid-scope, missing-prerequisite, and changed-fingerprint deliveries; this platform foundation proves only the generic duplicate, validation, and fingerprint boundaries.
- [x] Replay selects an immutable event range and consumer generation, preserves original event identities, retains existing inbox evidence, and reproduces expected projections without duplicate business effects.
- [x] Focused verification proves package boundaries, migration/generated-code consistency, and the crash-before/after-commit and duplicate-delivery exit evidence.

#### 7. Definition of Ready

- [ ] The event envelope and payload rules in the approved integration technical specification are treated as the contract baseline.
- [ ] PostgreSQL remains the initial transport; no broker design is introduced without an ADR.
- [ ] The transaction harness can inject failure at source commit, consumer commit, and worker establishment boundaries.
- [ ] Test fixtures can represent valid, duplicate, delayed, out-of-order, invalid, and poison events without creating a finance capability.
- [ ] The worker process and package ownership boundaries are agreed with `DLV-OPS-001`, `DLV-OPS-002`, and later capability delivery items.

#### 8. Definition of Done

- [x] All five platform foundation stories and their in-scope acceptance criteria pass.
- [x] Outbox and inbox migrations, constraints, indexes, and generated database artifacts are verified.
- [x] Source publication and receiving consumption transaction boundaries are demonstrated with failure injection.
- [x] Dispatcher leases, owner-token fencing, retry classification, backoff, and managed poison outcomes are tested.
- [x] Duplicate, ordering, replay, restart, and crash-recovery evidence passes without duplicate local effects or silently lost event evidence.
- [x] Worker lifecycle, package ownership, observability boundaries, and documentation are reviewed.
- [x] No finance capability, external broker, frontend behavior, or unrelated M0 delivery item is marked complete.

#### 9. Traceability identifiers

| Field | Value |
|---|---|
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` |
| Requirement IDs | Contributes to `GFR-006`, `GFR-007`, `GFR-008`, `GFR-009`, `GFR-012`, `GFR-013`, and `GFR-014`; does not complete capability-specific controls by itself. |
| Workflow IDs | `WF-7.13`, `WF-7.14` |
| NFR IDs | `NFR-REL-001`, `NFR-REL-002`, `NFR-INT-003`, `NFR-REC-001`, `NFR-REC-005`, `NFR-REC-007`, and applicable `NFR-TST-*` obligations. |
| Quality contribution | Durable integration evidence, at-least-once delivery, bounded-context inbox deduplication, lease-safe workers, and replay/recovery safety. |
| Exit evidence | Crash-before/after-commit and duplicate-delivery tests pass, with explicit outcomes for retry, ordering, invalid events, poison work, and replay. |

#### 10. Source references

- `docs/specs/finance_delivery_plan_v1.0.md` — DLV-PLAT-007 backlog and M0 exit evidence.
- `docs/specs/finance_domain_model_ddd.md` — cross-context event invariants, event interpretation, ordering, replay, concurrency, and recovery.
- `docs/specs/finance_ux_workflow_specification_v1.0.md` — `WF-7.13` cross-context event interpretation, ordering, and replay.
- `docs/specs/finance_nonfunctional_requirements_v1.0.md` — reliability, integration, recovery, and test obligations.
- `docs/specs/system_design/03_data_integration_architecture_v1.0.md` — PostgreSQL integration transport, ordering, replay, and recovery architecture.
- `docs/specs/technical_specifications/03_database_persistence_specifications_v1.0.md` — outbox/inbox tables, constraints, and indexes.
- `docs/specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md` — envelope, claiming, inbox, worker, retry, and broker-adoption contracts.
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md` — fault injection, duplicate, lease, recovery, and replay verification.

#### 11. Dependencies

- `DLV-PLAT-003` — Goose, pgx, sqlc, migration, and persistence verification workflow.
- `DLV-PLAT-006` — request fingerprint, scoped identity, established-result lookup, and owner-token coordination foundations.
- `DLV-OPS-001` and `DLV-OPS-002` — later structured telemetry, dashboards, and runbook completion.
- Future bounded-context delivery items — source effects, receiving effects, authorization, audit, and business-level exactly-once behavior.

