# DLV-PLAT-003 — Goose Migrations, pgx and sqlc Workflow User Stories

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

## 1. Purpose and scope

Break `DLV-PLAT-003` into small, reviewable stories that establish the approved PostgreSQL persistence workflow without prematurely implementing finance capabilities.

The completed item provides:

- pinned, repository-controlled Goose and sqlc tooling;
- an ordered Goose migration layout with schema-owned history;
- a minimal pgx connection and transaction workflow under `internal/platform/database`;
- a sqlc source-to-generated-code workflow using an existing technical platform table;
- PostgreSQL 18 integration verification from a clean database; and
- local and CI checks that reject migration or generated-code drift.

This is a persistence-tooling foundation. It completes no `FR-*`, `GFR-*`, `WF-*`, or `NFR-*` delivery item and does not complete `M0` by itself.

## 2. Authoritative implementation baseline

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

## 3. Boundary resolution

### 3.1 Boundary with DLV-PLAT-002

| Concern | `DLV-PLAT-002` | `DLV-PLAT-003` |
|---|---|---|
| PostgreSQL service | Starts, health-checks, stops, resets, seeds, and verifies the local database. | Reuses that environment; does not replace Compose lifecycle ownership. |
| Migration invocation | Calls the current repository migration command and propagates failure. | Defines the Goose layout, history, authoring rules, commands, and verification behind that invocation. |
| Seed manifest | Creates and verifies deterministic synthetic local seed state. | May use the existing technical seed-manifest table as the first sqlc workflow target without changing its business meaning. |
| Reproducibility | Proves reset and preparation from a missing volume. | Proves migrations and generated code are reproducible and drift-free. |

The existing real migration must be preserved or deliberately converted into valid Goose form. Replacing it with a no-op migration does not satisfy this item.

### 3.2 Boundary with DLV-CI-001

The exact `DLV-PLAT-003` exit evidence requires CI detection, while `DLV-CI-001` owns the complete pull-request quality pipeline.

| Concern | `DLV-PLAT-003` | `DLV-CI-001` |
|---|---|---|
| Persistence checks | Owns executable Goose/sqlc drift checks and their minimal CI invocation. | Incorporates or orchestrates those checks in the full repository quality pipeline. |
| CI breadth | Only migration, sqlc generation, compilation, and minimum persistence verification required by this item. | Go, frontend, OpenAPI, SQL, Terraform, security, documentation, branch gating, and complete PR quality policy. |
| Completion claim | May claim the exact persistence drift exit evidence. | Remains incomplete until the complete PR pipeline and all declared checks gate merge. |

A focused persistence CI job or workflow is permitted because it is required by the authoritative exit evidence. It must not be represented as completing `DLV-CI-001`.

### 3.3 Boundary with capability delivery

This item establishes the workflow, not the finance data model.

- Do not create all 19 bounded-context schemas merely to imitate the final topology.
- Initialize only schemas and technical tables already justified by delivered work.
- Do not create finance aggregate tables, finance seed records, business repositories, business commands, API operations, events, or screens.
- Later capability items add their owned migrations and sqlc queries through this workflow.

## 4. Required implementation inputs

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

## 5. Cross-story constraints

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

## 6. User Story 1 — Establish the Goose migration contract

**As the TALLY developer, I want a pinned and schema-aware Goose migration workflow, so that database changes are ordered, reviewable, repeatable, and owned by the correct schema.**

### Value

Turns the minimum migration path used by `DLV-PLAT-002` into the approved long-term migration workflow without changing finance-domain scope.

### Required command intents

Names are recommended aliases, not architecture contracts.

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Validate migrations | `db-migrate-validate` | Validate migration ordering and Goose syntax for each initialized schema. |
| Show status | `db-migrate-status` | Show applied and pending migrations for each initialized schema. |
| Apply migrations | Existing `db-migrate` | Apply pending migrations through pinned Goose tooling. |
| Create migration | `db-migrate-create` | Create a correctly located migration skeleton for an explicitly selected schema. |
| Verify migration source | `db-migrate-check` | Verify migration inventory/checksums or the selected equivalent drift contract. |

### Acceptance criteria

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

### Likely files

```text
go.mod
go.sum
db/migrations/<initialized-schema>/*.sql
db/migrations/<migration-inventory-or-checksum-file>
<root command manifest>
README.md
```

Exact names may differ, but schema ownership and pinned tooling must remain explicit.

### Evidence

- Pinned Goose version output.
- Migration validation output.
- Clean-database apply and status output.
- Second apply showing no pending change.
- Migration history location for every initialized schema.
- Controlled invalid-migration result with non-zero status.
- Controlled migration-drift result with non-zero status.

---

## 7. User Story 2 — Establish the pgx database foundation

**As the TALLY developer, I want a minimal pgx connection and transaction foundation, so that future repositories can use PostgreSQL explicitly without introducing an ORM or bypassing module boundaries.**

### Value

Provides the approved low-level database access mechanism while keeping business repositories in their future owning modules.

### Acceptance criteria

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

### Likely files

```text
internal/platform/database/
  pool.go
  pool_test.go
  integration_test.go
```

Exact files may differ. Keep the package small and technical.

### Evidence

- Go module diff showing pgx pinning.
- Successful PostgreSQL 18 connection test.
- Commit and rollback integration-test output.
- Controlled invalid-configuration or unavailable-database failure.
- Review confirming no ORM, generic repository, finance package, or credential logging was added.

---

## 8. User Story 3 — Establish the sqlc generation workflow

**As the TALLY developer, I want schema-owned SQL queries to generate typed Go code through sqlc, so that SQL remains explicit while routine row and parameter mapping is reproducible.**

### Value

Implements the approved pgx/sqlc choice and creates the source-to-generated-code feedback loop required for later capability repositories.

### Acceptance criteria

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

### Likely files

```text
sqlc.yaml
db/queries/platform/*.sql
internal/platform/database/<generated-package>/*.go
<root command manifest>
README.md
```

Exact paths may differ, but ownership and source/generated separation must remain clear.

### Evidence

- Pinned sqlc version output.
- sqlc generation output.
- Generated file list.
- Go compilation/test output.
- Integration result executing at least one generated query.
- Review confirming generated code was not hand-authored and no cross-schema query was introduced.

---

## 9. User Story 4 — Prove migrations, pgx, and sqlc together

**As the TALLY developer, I want a clean PostgreSQL integration test that applies migrations and executes generated queries through pgx, so that the persistence workflow is proven as one coherent path rather than as disconnected tools.**

### Value

Provides executable evidence that the selected migration, connection, generation, and query mechanisms work together against PostgreSQL 18.

### Acceptance criteria

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

### Evidence

- Integration-test command and output.
- Resolved PostgreSQL image/version.
- Applied migration versions and history table locations.
- Generated-query execution result.
- Commit and rollback assertions.
- A repeated run showing deterministic success.
- Controlled failure proving non-zero propagation.

---

## 10. User Story 5 — Detect persistence drift in CI

**As the TALLY developer, I want one local and CI persistence verification command, so that changed migrations or stale generated SQL code cannot be merged unnoticed.**

### Value

Directly proves the authoritative exit evidence: CI detects migration and generated-code drift.

### Required verification sequence

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

### Local acceptance criteria

- [ ] One documented root command runs the complete persistence verification.
- [ ] It uses repository-pinned Goose and sqlc versions.
- [ ] It does not silently skip migration validation, generation, compilation, or integration verification.
- [ ] It fails when a committed historical migration changes without the required migration-inventory update.
- [ ] It fails when a migration is deleted, duplicated, invalid, or out of order.
- [ ] It fails when sqlc source changes but generated code is stale.
- [ ] It fails when a generated file is manually changed or deleted.
- [ ] It fails when generated code does not compile.
- [ ] It fails when clean migration application or the generated query integration test fails.
- [ ] It succeeds from a clean checkout with a clean PostgreSQL test dependency.
- [ ] A successful run leaves the Git working tree unchanged.
- [ ] Failure propagation is non-zero and identifies the failed stage.

### CI acceptance criteria

- [ ] A focused GitHub Actions job or workflow invokes the same repository root persistence command used locally.
- [ ] The workflow uses PostgreSQL `18.x` through a service container or the approved Testcontainers path.
- [ ] Tool versions come from repository-controlled pins rather than floating CI installation commands.
- [ ] The workflow requires no Azure credential, production database, or committed secret.
- [ ] The workflow does not print `DATABASE_URL`, database passwords, or complete secret values.
- [ ] The job fails on migration drift.
- [ ] The job fails on stale or manually changed sqlc output.
- [ ] The job fails on migration, compile, or integration-test failure.
- [ ] The job succeeds on the reviewed clean branch.
- [ ] The job is named clearly enough to be reused or included by the later `DLV-CI-001` full PR pipeline.
- [ ] Documentation explicitly states that this focused job does not complete `DLV-CI-001`.

### Controlled negative evidence

Record evidence for at least these temporary, uncommitted mutations, restoring the clean tree after each check:

1. change a committed migration without updating its inventory/checksum;
2. change a sqlc query without regenerating;
3. manually change or delete generated output; and
4. introduce an invalid migration or generated-code compile error.

Each mutation must make the local check fail. At least migration drift and generated-code drift must also be demonstrated as CI failures through a temporary test commit, workflow-dispatch branch, or equivalent reviewable evidence that is not merged in the failing state.

### Documentation acceptance criteria

- [ ] README documents pinned-tool setup and verification.
- [ ] README documents migration creation, validation, status, application, and drift checking.
- [ ] README documents sqlc source, generation, generated output, and no-manual-edit rule.
- [ ] README documents the persistence integration-test command and PostgreSQL/Docker prerequisites.
- [ ] README explains forward-fix versus disposable local reset.
- [ ] README explains the `DLV-PLAT-003`/`DLV-CI-001` boundary.
- [ ] README does not claim finance schemas, finance repositories, OpenAPI, idempotency, outbox/inbox, production recovery, full `QG-03`, full CI, or `M0` complete.
- [ ] A verification record is committed or linked.

### Evidence record template

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

## 11. Delivery-item acceptance summary

`DLV-PLAT-003` is complete only when all five stories and every condition below pass:

- [ ] Goose and sqlc tool versions are pinned in repository-controlled manifests.
- [ ] The current real platform migration is valid under the Goose workflow.
- [ ] Migration sets are schema-owned and every initialized schema has its own history table.
- [ ] Clean migration and repeated migration application pass against PostgreSQL 18.
- [ ] A deterministic migration inventory/checksum or equivalent mechanism detects drift.
- [ ] A minimal pgx database foundation exists under `internal/platform/database`.
- [ ] Connection, transaction commit, and rollback behavior pass integration tests.
- [ ] sqlc configuration, query source, and generated Go output are committed according to the selected workflow.
- [ ] Generated code uses pgx and compiles.
- [ ] At least one real generated query executes against the delivered technical platform persistence contract.
- [ ] Persistence integration verification begins from clean PostgreSQL state and is repeatable.
- [ ] One root command validates migrations, regenerates sqlc, checks the working tree, compiles, migrates, and tests.
- [ ] Local negative checks prove migration and generated-code drift detection.
- [ ] CI invokes the same persistence command and rejects both migration drift and generated-code drift.
- [ ] Documentation and verification evidence are current.
- [ ] No finance schema, aggregate table, business repository, API operation, event, or capability behavior is introduced.
- [ ] No adjacent delivery item or milestone is falsely marked complete.

## 12. Explicit exclusions and follow-on ownership

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

## 13. Definition of Ready

- [ ] Exact item, epic, milestone, deliverable, and exit evidence are identified.
- [ ] `DLV-PLAT-001` root commands and repository layout are stable.
- [ ] `DLV-PLAT-002` PostgreSQL 18 lifecycle and reproducible reset pass.
- [ ] The existing real migration and technical seed-manifest persistence contract are identified from the repository.
- [ ] Exact Goose, pgx, sqlc, and Testcontainers versions can be selected and pinned.
- [ ] Schema-owned migration and generated-code paths can be chosen without inventing finance modules.
- [ ] The migration drift method is selected.
- [ ] The focused CI boundary with `DLV-CI-001` is understood.
- [ ] Positive, negative, clean-database, repeated-run, and CI evidence is identified.
- [ ] Five stories are small enough for one or a short chain of reviewable changes.

Finance requirements, authoritative finance records, UX states, authorization decisions, correction paths, business concurrency, integration events, and production recovery are not applicable.

## 14. Definition of Done

- [ ] Every criterion in Sections 6–11 passes; none is silently deferred.
- [ ] Tool pins, migrations, migration inventory/checksums, pgx foundation, sqlc source/configuration, generated code, tests, root commands, focused CI, and documentation are complete.
- [ ] Go dependency manifests are tidy and committed.
- [ ] Goose migration validation and clean application pass.
- [ ] Repeated migration application is harmless.
- [ ] Migration history is schema-owned for every initialized schema.
- [ ] pgx connection, commit, rollback, and cleanup tests pass.
- [ ] sqlc generation is deterministic and generated code compiles.
- [ ] PostgreSQL 18 integration verification passes from clean state and on repetition.
- [ ] The root persistence check passes and leaves a clean working tree.
- [ ] Controlled local migration-drift and generated-code-drift checks fail as expected.
- [ ] Focused CI rejects migration drift and generated-code drift and passes on the clean branch.
- [ ] Documentation matches actual versions, commands, file locations, migration behavior, generated-code policy, and CI job.
- [ ] No real credential, production data, build output, test artifact, or developer-specific config is committed.
- [ ] The diff is reviewed against modular-monolith, schema ownership, query ownership, and adjacent-item boundaries.
- [ ] Traceability is current.
- [ ] No functional requirement, workflow, NFR qualification, full `QG-03`, `DLV-CI-001`, adjacent delivery item, or `M0` milestone is incorrectly marked complete.
- [ ] No critical or high unresolved defect remains.

### General Definition-of-Done applicability

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

## 15. Traceability and quality-gate contribution

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

## 16. Consistency review record

### Review pass 1 — Corrected

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

### Review pass 2 — Passed

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

### Review pass 3 — Passed

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

## 17. Source references

- `02_work_breakdown_backlog_v1.0.md` / `finance_delivery_plan_v1.0.md` — exact `DLV-PLAT-003` deliverable and exit evidence, quality gates, Definition of Done, and CI/local environment policy.
- `01_solution_architecture_overview_v1.0.md` — modular monolith, PostgreSQL 18, Goose, pgx, sqlc, Testcontainers, and local-first technology baseline.
- `05_architecture_traceability_decisions_v1.0.md` — ADR-005 through ADR-007 for PostgreSQL, one database/schema-per-context, and pgx/sqlc instead of an ORM.
- `01_backend_module_specifications_v1.0.md` — `internal/platform/database`, `db/migrations`, module dependency rules, transaction contract, configuration, and SQL ownership tests.
- `03_data_integration_architecture_v1.0.md` — PostgreSQL authority, schema ownership, exact-decimal requirement, migration discipline, and no cross-context writes.
- `03_database_persistence_specifications_v1.0.md` — PostgreSQL topology, one Goose history table per schema, migration policy, constraints, and role-verification expectations.
- `09_testing_performance_recovery_specifications_v1.0.md` — PostgreSQL 18/Testcontainers repository integration tests and release requirement that sqlc output and migrations have no drift.
- `10_technical_traceability_decisions_v1.0.md` — table/constraint/index changes update migrations, sqlc queries, evidence, and traceability.
- `DLV-PLAT-002_user_stories.md` — established boundary assigning Goose layout/history, pgx, sqlc, generated SQL, and drift checks to `DLV-PLAT-003`.
