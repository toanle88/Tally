# DLV-PLAT-002 — Docker Compose PostgreSQL Development Environment User Stories

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

## 1. Purpose and scope

Break `DLV-PLAT-002` into small, reviewable stories without absorbing work assigned to `DLV-PLAT-003`.

The completed item provides:

- one local PostgreSQL service managed through Docker Compose;
- safe local configuration with no committed real credentials;
- root command intents for database startup, health, shutdown, migration, seed, verification, and reset;
- deterministic invocation of the current repository migration and seed contracts; and
- evidence that the database can be destroyed and recreated from committed source.

This is an engineering-environment item. It completes no `FR-*`, `GFR-*`, `WF-*`, or `NFR-*` delivery item and does not complete `M0` by itself.

## 2. Authoritative implementation baseline

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

## 3. Boundary resolution with DLV-PLAT-003

The `DLV-PLAT-002` exit evidence says the database “migrates,” while `DLV-PLAT-003` owns the migration and database-access workflow. The boundary is:

| Concern | `DLV-PLAT-002` | `DLV-PLAT-003` |
|---|---|---|
| PostgreSQL service | Define, start, health-check, stop, and reset it. | No ownership. |
| Migration execution | Invoke the current approved migration command against local PostgreSQL and propagate failure. | Establish Goose layout and history, migration authoring, pgx, sqlc, generated SQL, and drift checks. |
| Seed execution | Invoke a versioned synthetic seed contract and verify its declared result. | Supply migration-created structures needed by that contract when persistence work owns them. |
| CI | No CI implementation or completion claim. | Migration/generated-code drift is later enforced through CI. |
| Completion | Requires a real migration and seed path. | Need not be fully complete merely to provide that minimum executable contract. |

A no-op migration, empty seed, or command that always exits successfully without proving database state does not satisfy this item.

## 4. Required implementation inputs

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

## 5. Cross-story constraints

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

## 6. User Story 1 — Define safe local database configuration

**As the TALLY developer, I want a checked-in Docker Compose definition and safe local configuration contract, so that each workstation begins from the same PostgreSQL baseline without committing real secrets.**

### Value

Establishes version, configuration, storage, and exposure rules before lifecycle automation.

### Acceptance criteria

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

### Likely files

```text
compose.yaml
.env.example
.gitignore
README.md
```

Exact names may differ.

### Evidence

- Docker and Compose version output.
- `docker compose config` output.
- Recorded image reference and resolved identifier where available.
- Secret and scope review of the diff.

---

## 7. User Story 2 — Start and health-check PostgreSQL

**As the TALLY developer, I want PostgreSQL to start through Docker Compose and become observably healthy, so that later commands do not race startup or report success too early.**

### Value

Creates a deterministic prerequisite for migration, seed, test, and development work.

### Acceptance criteria

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

### Evidence

- Startup and `docker compose ps` output showing healthy state.
- PostgreSQL version query.
- Stop/start persistence result.
- Controlled health-failure result with non-zero status.

---

## 8. User Story 3 — Provide root database lifecycle commands

**As the TALLY developer, I want a small root database command surface, so that I can operate local PostgreSQL consistently without memorizing Compose details.**

### Value

Aligns database operation with the root command approach established by `DLV-PLAT-001`.

### Required command intents

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

### Acceptance criteria

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

### Evidence

- Output from each selected lifecycle command.
- A controlled child-command failure proving propagation.
- Stop/start evidence showing volume preservation.

---

## 9. User Story 4 — Orchestrate migration and deterministic seeding

**As the TALLY developer, I want explicit migration, seed, and verification commands against local PostgreSQL, so that preparation is repeatable and independent of hidden first-start behavior.**

### Value

Proves “migrates and seeds” while preserving `DLV-PLAT-003` ownership.

### Required command intents

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Migrate | `db-migrate` | Invoke the current approved repository migration contract. |
| Seed | `db-seed` | Establish the declared synthetic seed state without duplicates. |
| Verify | `db-verify` | Prove migration state and seed version/fingerprint or equivalent deterministic result. |
| Prepare | `db-prepare` | Wait, migrate, seed, and verify in order; stop on failure. |

### Acceptance criteria

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

### Evidence

- Applied/current migration output.
- Seed version or fingerprint output.
- Prepared-state verification output.
- A second seed/verify run proving deterministic repetition.
- Controlled preparation failure with non-zero status.

---

## 10. User Story 5 — Reset and prove reproducibility

**As the TALLY developer, I want a deliberately destructive, project-scoped reset and recorded clean-environment proof, so that the database can be rebuilt without relying on an old volume.**

### Value

Directly proves the complete exit evidence.

### Reset acceptance criteria

- [ ] A clearly named reset command runs from repository root.
- [ ] Documentation labels it destructive to the TALLY local database.
- [ ] It removes only the TALLY Compose project’s declared disposable resources and database volume.
- [ ] It does not run broad Docker pruning or delete unrelated containers, networks, images, or volumes.
- [ ] It recreates PostgreSQL, waits for health, migrates, seeds, and verifies.
- [ ] Any failed phase returns non-zero.
- [ ] Success leaves PostgreSQL healthy in the declared migration and seed state.
- [ ] Two complete reset runs produce the same declared verification result.
- [ ] At least one verification begins with the prior TALLY database volume absent.
- [ ] Verification requires no uncommitted file, prior volume, real credential, Azure, or external provider.
- [ ] Image caches may improve speed but are not required for logical correctness.

### Documentation acceptance criteria

- [ ] README documents Docker and Compose prerequisites.
- [ ] README documents safe local configuration setup.
- [ ] README documents all selected lifecycle, migration, seed, verify, and reset commands.
- [ ] README documents host binding and port behavior.
- [ ] README distinguishes ordinary stop from destructive reset.
- [ ] README explains the `DLV-PLAT-002`/`DLV-PLAT-003` boundary.
- [ ] README does not claim Azure PostgreSQL, HA, backup, DR, production NFRs, CI, authentication, finance schemas, or business modules complete.
- [ ] A verification record is committed or linked.

### Evidence record template

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

## 11. Delivery-item acceptance summary

`DLV-PLAT-002` is complete only when all five stories and all conditions below pass:

- [ ] One checked-in PostgreSQL `18.x` Compose service exists.
- [ ] Effective configuration validates and contains no real committed secret.
- [ ] PostgreSQL starts and becomes healthy within a bound.
- [ ] Root lifecycle commands operate it and propagate failures.
- [ ] A real migration executes against a fresh database.
- [ ] A versioned synthetic seed executes deterministically.
- [ ] Prepared-state verification proves more than connectivity.
- [ ] Stop/start preserves ordinary local state.
- [ ] Reset destroys only project-owned state.
- [ ] Two reset-and-prepare runs produce the same declared result.
- [ ] Clean-environment evidence is recorded.
- [ ] No adjacent delivery item is falsely marked complete.

## 12. Explicit exclusions and follow-on ownership

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

## 13. Definition of Ready

- [ ] Exact item, epic, milestone, deliverable, and exit evidence are identified.
- [ ] The `DLV-PLAT-001` repository and root command surface are available or stable.
- [ ] Docker Engine and Compose v2 have a setup path.
- [ ] A reviewed PostgreSQL `18.x` image can be selected.
- [ ] Local project/service/volume/binding/port/database/user/config choices can be recorded.
- [ ] The `DLV-PLAT-002`/`DLV-PLAT-003` boundary is understood.
- [ ] Minimum real migration, synthetic seed, and deterministic verification paths are available or scheduled in the same short chain.
- [ ] Reset scope is limited to TALLY project resources.
- [ ] Positive, negative, stop/start, rerun, reset, and clean-environment evidence is identified.
- [ ] Five stories are small enough for one or a short chain of reviewable changes.

Finance requirements, authoritative finance records, UX states, authorization decisions, correction paths, business concurrency, and integration events are not applicable.

## 14. Definition of Done

- [ ] Every criterion in Sections 6–11 passes; none is silently deferred.
- [ ] Compose config, root commands, safe example config, verification logic, checks, and documentation are complete.
- [ ] `docker compose config` passes.
- [ ] PostgreSQL `18.x` starts and becomes healthy within the declared bound.
- [ ] Normal stop/start preserves the volume.
- [ ] Migration, seed, and verification pass on a fresh database.
- [ ] Repeated seed establishes the same state without duplicates.
- [ ] Reset passes twice with the same verification result.
- [ ] Controlled startup/health and preparation failures propagate non-zero.
- [ ] Clean-environment evidence is attached or linked.
- [ ] Documentation matches actual image, commands, host/port behavior, reset scope, and verification method.
- [ ] No real credential, production data, unrelated Docker resource, build output, or developer-specific config is committed.
- [ ] The diff is reviewed against data ownership and adjacent-item boundaries.
- [ ] Traceability is current.
- [ ] No FR, GFR, workflow, NFR qualification, `DLV-PLAT-003`, adjacent item, or `M0` is falsely marked complete.
- [ ] No critical or high unresolved defect remains.

### General Definition-of-Done applicability

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

## 15. Traceability and quality-gate contribution

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

## 16. Consistency review record

### Review pass 1 — Corrected

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

### Review pass 2 — Passed

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

### Review pass 3 — Passed

The final review confirmed:

- no finance requirement, ID, API, schema, table, event, or business seed fact was invented;
- no exact local port, credential, database name, service name, or alias is treated as an architecture contract;
- real migration and seed results are required, not mere connectivity;
- `DLV-PLAT-003` retains Goose, pgx, sqlc, generated SQL, and drift ownership;
- reset cannot affect unrelated Docker resources;
- no Azure, CI, authentication, observability, production, backup/restore, or NFR completion is claimed; and
- the acceptance summary proves “Database starts, migrates, seeds and resets reproducibly.”

**Final consistency result: PASS.**

## 17. Source references

- `02_work_breakdown_backlog_v1.0.md` — exact deliverable, exit evidence, adjacent items, Ready, and Done.
- `01_solution_architecture_overview_v1.0.md` — modular monolith, PostgreSQL `18.x`, local Docker Compose profile, and version policy.
- `03_data_integration_architecture_v1.0.md` — PostgreSQL authority, schema ownership, no cross-context writes, and migration discipline.
- `03_database_persistence_specifications_v1.0.md` — one database per environment, PostgreSQL `18.x`, schema ownership, and migration responsibilities.
- `09_testing_performance_recovery_specifications_v1.0.md` — PostgreSQL-equivalent testing and recovery boundaries.
- `04_quality_testing_environment_plan_v1.0.md` — `QG-01`, `QG-03`, `QG-10`, environment progression, Ready, and Done.
- `03_dependencies_milestones_releases_v1.0.md` — local-primary policy, clean-environment evidence, and change control.
- `05_risks_costs_governance_traceability_v1.0.md` — local-first cost control, synthetic-data control, and scope governance.
