# DLV-CI-001 — Pull-request CI Quality Pipeline User Stories

| Field | Value |
|---|---|
| Status | Implemented — hosted CI qualification pending |
| Milestone | `M0` |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Delivery item | `DLV-CI-001` |
| Deliverable | Create a pull-request CI quality pipeline. |
| Exit evidence | Go, frontend, OpenAPI, SQL, Terraform, security, and documentation checks gate merge. |

## 1. Outcome

Provide one repository-owned pull-request quality pipeline that runs the
existing focused verification commands, fails closed on quality regressions,
and exposes stable, reviewable evidence without requiring production data,
Azure credentials, or live infrastructure.

The pipeline orchestrates checks owned by their existing delivery items. It
does not move application, database, Terraform, security, or documentation
rules into a new shared module.

## 2. Learning objective

Learn how to compose a safe CI quality gate from repository-controlled commands,
dependency lockfiles, service containers, generated-artifact checks, and
credential-free infrastructure verification. Preserve a clear boundary between
pull-request validation, protected deployment, smoke testing, and later release
qualification.

## 3. Scope and ownership

`DLV-CI-001` owns pull-request orchestration, job dependencies, failure
propagation, safe evidence publication, and the documented required-check
contract. The delivery item does not own the rules tested by each focused gate.

Current repository evidence includes focused workflows for OpenAPI, persistence,
Terraform, and documentation, plus root commands for Go, frontend, generated
artifacts, database, infrastructure, and security checks. The missing scope is
the complete pull-request pipeline that composes those checks consistently.

### Delivery-item boundaries

| Concern | Owner | CI responsibility |
|---|---|---|
| Go API, worker, and platform tests | `DLV-PLAT-001` through `DLV-PLAT-007` | Invoke the approved root commands and preserve their failure status. |
| Frontend tests and build | `DLV-PLAT-001`, `DLV-UX-001`, `DLV-UX-002` | Install from the committed lockfile and run the approved test/build commands. |
| OpenAPI and generated artifacts | `DLV-PLAT-004` | Invoke `make api-check`; do not duplicate generator logic in workflow YAML. |
| Migrations, sqlc, and persistence | `DLV-PLAT-002`, `DLV-PLAT-003` | Provide the PostgreSQL runner/service and invoke the approved persistence gate. |
| Terraform and infrastructure safety | `DLV-IAC-001`, `DLV-IAC-002` repository checks | Run credential-free checks only; do not apply, destroy, or qualify Azure. |
| Security and secret hygiene | Applicable security/NFR owners | Run the repository-approved scanners and prevent secret leakage in logs/artifacts. |
| Documentation | Repository documentation owners | Run the documentation build/check and publish only safe failure/success summaries. |

## 4. Explicit exclusions

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

## 5. Owning bounded context & architecture

- **Owning Bounded Context / Module:** Repository/platform delivery tooling; no
  finance bounded context owns CI orchestration.
- **Domain Aggregates & Invariants:** None. CI must preserve existing module
  ownership, immutable financial facts, exact-decimal money, idempotency,
  transactional outbox, and authorization/audit boundaries by not bypassing
  the commands and tests that enforce them.
- **Workflow ownership:** Pull-request validation belongs to `DLV-CI-001`.
  Protected apply belongs to the existing deployment workflow and remains
  separate from this delivery item.

## 6. Contract & impact analysis

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

## 7. Failure, concurrency & idempotency rules

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

## 8. Traceability identifiers

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

## 9. Source references

- `docs/specs/finance_delivery_plan_v1.0.md` — `DLV-CI-001` deliverable,
  exit evidence, quality gates, test layers, and release evidence.
- `docs/specs/system_design/04_security_deployment_operations_v1.0.md` — CI/CD
  pipeline, protected apply, secret, deployment, and observability boundaries.
- `docs/specs/finance_nonfunctional_requirements_v1.0.md` — traceability,
  secret safety, operational diagnostics, testing, and release evidence.
- `docs/specs/technical_specifications/09_testing_performance_recovery_specifications_v1.0.md`
  — verification layers and evidence expectations.
- `docs/backlog/stories/DLV-PLAT-001_user_stories.md` — repository and root
  command boundary.
- `docs/backlog/stories/DLV-PLAT-002_user_stories.md` — local PostgreSQL and
  persistence-environment boundary.
- `docs/backlog/stories/DLV-PLAT-003_user_stories.md` — focused persistence CI
  boundary and explicit ownership by `DLV-CI-001`.
- `docs/backlog/stories/DLV-IAC-001_user_stories.md` and
  `docs/backlog/stories/EP-IAC-001_user_stories.md` — credential-free Terraform
  checks and closed/deferred Azure qualification boundary.
- `Makefile`, `package.json`, `web/package.json`, and `.github/workflows/` —
  current commands and focused workflow evidence.

## 10. Dependencies

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

## 11. User stories

### User Story 1 — Establish the pull-request pipeline contract

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

### User Story 2 — Gate API, worker, and frontend quality

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

### User Story 3 — Gate contract, persistence, and generated artifacts

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

### User Story 4 — Gate infrastructure, security, and documentation quality

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

### User Story 5 — Publish safe evidence and enforce the merge gate

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

## 12. Definition of Ready

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

## 13. Definition of Done

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

## 14. Suggested implementation steps

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

## 15. Required test evidence

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

## 16. Likely files / packages involved

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

## 17. Risks or open questions

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

## 18. Implementation status

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
