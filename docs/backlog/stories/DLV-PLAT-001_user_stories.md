# DLV-PLAT-001 — Monorepo Foundation User Stories

| Field | Value |
|---|---|
| Delivery item | `DLV-PLAT-001` |
| Item type | Platform foundation item |
| Parent epic | `EP-PLAT-001` — Engineering foundation |
| Milestone | `M0` — Engineering foundation |
| Artifact version | 1.1 |
| Review status | Passed — two review passes completed |
| Delivery profile | Solo, part-time, local-first learning project |
| Dependency position | No predecessor is identified for this item; its parent epic has no epic dependency. |
| Authoritative deliverable | Create monorepo with Go API, React application and shared commands. |
| Authoritative exit evidence | Clean clone builds and tests locally. |

## 1. Purpose and scope

Break `DLV-PLAT-001` into small, reviewable stories without absorbing work assigned to adjacent M0 delivery items.

The completed delivery item provides:

- one repository containing the initial Go API and React application;
- a minimal Go API shell aligned with the selected HTTP baseline;
- a minimal React, TypeScript, and Vite application shell;
- one documented root command surface for both applications; and
- repeatable clean-clone build and test evidence.

This is an engineering foundation item, not product behavior. It completes no `FR-*`, `GFR-*`, `WF-*`, or `NFR-*` delivery item and does not complete M0 by itself.

## 2. Authoritative implementation baseline

| Area | Baseline used by these stories |
|---|---|
| Architecture | One-repository modular monolith; bounded-context boundaries remain explicit. |
| Backend | Go `1.26.x`; `net/http` with `chi`; composition root at `cmd/api`. |
| Frontend | Node.js `24 LTS`; React `19.2`; TypeScript; Vite `8`. |
| Frontend tests | Vitest and Testing Library for the application-shell test. MSW is not required until an API interaction needs it. |
| Repository boundary | Reusable technical facilities belong under `internal/platform`; no finance bounded-context implementation is created by this item. |

Patch versions and lockfile-resolved dependency versions are implementation evidence, not new architecture decisions.

## 3. Required implementation inputs

The item is not ready to start until these local implementation inputs are resolved:

- canonical repository URL and Go module path;
- root command mechanism, using the simplest option that does not add an unnecessary runtime or broad task-runner dependency; and
- frontend development-port behavior, using checked-in configuration when the Vite default is changed. The API uses the approved `HTTP_ADDR` default `:8080`.

Do not invent a placeholder Go module path such as `example.com/...`.

## 4. Cross-story constraints

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

## 5. User Story 1 — Establish the monorepo structure

**As the TALLY developer, I want a stable repository structure for the Go API and React application, so that later delivery items can extend the platform without reorganizing or violating module boundaries.**

### Value

Creates the physical foundation for the modular monolith and frontend without prematurely implementing business modules or adjacent platform capabilities.

### Acceptance criteria

- [ ] The repository root contains `cmd/`, `internal/`, and `web/`.
- [ ] The Go API entry point is located at `cmd/api/main.go`.
- [ ] Reusable API-shell HTTP behavior is located under `internal/platform/httpx/` or an equivalently narrow technical package under `internal/platform/`.
- [ ] The frontend application is located directly under `/web`; it is not nested inside a backend-only parent directory.
- [ ] The root contains `README.md`, `.gitignore`, `go.mod`, and the selected root command manifest.
- [ ] `go.mod` uses the confirmed canonical module path and a Go directive compatible with the approved Go `1.26.x` baseline.
- [ ] `go.sum` is committed when the selected Go dependencies generate it.
- [ ] npm is used for the frontend and `web/package-lock.json` is committed.
- [ ] Local environment files, dependency directories, frontend build output, test output, and compiled binaries are ignored.
- [ ] No finance bounded-context package, database artifact, event contract, generated API contract, or cloud resource is introduced.
- [ ] Future repository areas remain absent until their owning delivery items require them.

### Expected minimum shape

```text
.
├── cmd/
│   └── api/
│       └── main.go
├── internal/
│   └── platform/
│       └── httpx/
├── web/
├── .gitignore
├── go.mod
├── README.md
└── <root command manifest>
```

### Evidence

- Repository tree from the implementation commit.
- Confirmed repository URL and Go module path.
- Output of `go env GOMOD` from the repository root.
- Diff review confirming that excluded capability and persistence code was not added.

---

## 6. User Story 2 — Provide a minimal Go API shell

**As the TALLY developer, I want a small, testable Go HTTP application, so that the backend process can be built and exercised before finance behavior is introduced.**

### Value

Establishes the API process boundary, selected router, composition root, and first deterministic backend feedback loop.

### Acceptance criteria

- [ ] `go build ./cmd/api` succeeds.
- [ ] The API uses `net/http` with `chi`, without introducing a broader web framework.
- [ ] `GET /health/live` returns HTTP `200 OK` and valid JSON.
- [ ] The liveness response checks process liveness only and does not claim database or dependency readiness.
- [ ] The liveness response exposes no secrets, environment contents, stack traces, or internal topology.
- [ ] The liveness handler is implemented as reusable technical behavior under `internal/platform/`.
- [ ] A deterministic Go test verifies the liveness handler status, content type, and response shape.
- [ ] Invalid startup configuration and listen failures produce an explicit non-successful process outcome.
- [ ] The API handles operating-system termination signals and attempts graceful shutdown with a bounded timeout.
- [ ] The API starts and its tests pass without PostgreSQL, Docker, Azure, authentication, or external services.
- [ ] `go test ./...` succeeds.

`/health/ready` is not required by this item because database and migration compatibility do not yet exist. A later item must implement readiness without changing the approved liveness semantics.

### Likely files

```text
cmd/api/main.go
internal/platform/httpx/health.go
internal/platform/httpx/health_test.go
```

Exact file names may vary, but the composition-root and reusable-platform boundary must remain clear.

### Evidence

- Output from `go build ./cmd/api`.
- Output from `go test ./...`.
- Local request and response for `GET /health/live`.
- Shutdown verification note identifying the signal and observed bounded shutdown outcome.

---

## 7. User Story 3 — Provide a minimal React application shell

**As the TALLY developer, I want a minimal React, TypeScript, and Vite application under `/web`, so that frontend work begins from the approved stack and can later grow by capability without restructuring.**

### Value

Creates the frontend runtime and first deterministic component test while leaving shared finance UX and capability behavior to their owning delivery items.

### Acceptance criteria

- [ ] The frontend uses the approved Node.js `24 LTS`, React `19.2`, TypeScript, and Vite `8` baselines.
- [ ] `npm ci` succeeds inside `/web` using the committed lockfile.
- [ ] The application renders a minimal shell that identifies the product as TALLY.
- [ ] Implemented source areas include `web/src/app/` and `web/src/test/`.
- [ ] The structure remains compatible with later `routes`, `components`, `capabilities`, and `lib` areas without adding empty or speculative finance implementations.
- [ ] A deterministic Vitest and Testing Library test proves that the application shell mounts and displays the TALLY identity.
- [ ] The frontend test runs once in non-watch mode and exits with the correct process status.
- [ ] `npm run build` succeeds with no TypeScript compilation errors.
- [ ] Unused Vite sample assets and demonstration code are removed.
- [ ] Tailwind, daisyUI wrappers, TanStack Query, React Hook Form, Zod, TanStack Table, routing, and business-capability screens are not represented as completed.

### Likely files

```text
web/package.json
web/package-lock.json
web/index.html
web/tsconfig.json
web/vite.config.ts
web/src/main.tsx
web/src/app/app.tsx
web/src/app/app.test.tsx
web/src/test/setup.ts
```

### Evidence

- Output from `npm ci`.
- Output from the frontend non-watch test command.
- Output from `npm run build`.
- Local verification note or screenshot showing the TALLY application shell.

---

## 8. User Story 4 — Provide shared root commands

**As the TALLY developer, I want one documented command surface at the repository root, so that I can build, test, verify, and run both applications consistently.**

### Value

Creates a simple engineering feedback loop and prevents one application from being silently omitted from local verification.

### Required command intents

The command names are an implementation choice. The names below are recommended aliases, not new product contracts.

| Intent | Recommended alias | Required behavior |
|---|---|---|
| Build all | `build` | Build the Go API and production frontend bundle. |
| Test all | `test` | Run all Go tests and frontend tests once. |
| Verify all | `check` | Run the complete local build-and-test verification. |
| Run API | `dev-api` | Start the local Go API. |
| Run frontend | `dev-web` | Start the Vite development server. |

### Acceptance criteria

- [ ] Every required command intent can be invoked from the repository root.
- [ ] The build-all command fails when either the Go build or frontend build fails.
- [ ] The test-all command fails when either the Go tests or frontend tests fail.
- [ ] The verify-all command runs the complete build-and-test sequence and returns a non-zero status when any required step fails.
- [ ] No command reports success when the Go or frontend portion was skipped.
- [ ] The API and frontend development commands start their applications independently.
- [ ] The command mechanism does not add an unnecessary runtime or broad task-runner dependency.
- [ ] Command names, prerequisites, working directory, and expected behavior are documented in `README.md`.
- [ ] A controlled negative check proves that the verify-all command propagates a child failure.

### Evidence

- Output from the root build-all, test-all, and verify-all commands.
- Controlled negative-check output showing non-zero failure propagation.
- Local startup notes for the API and frontend commands.

---

## 9. User Story 5 — Document and prove clean-clone reproducibility

**As the TALLY developer, I want clean-clone setup instructions and recorded verification, so that the repository foundation is reproducible rather than dependent on untracked local state.**

### Value

Directly proves the authoritative exit evidence: a clean clone builds and tests locally.

### Documentation acceptance criteria

- [ ] `README.md` lists the required Go, Node.js, and npm major versions.
- [ ] `README.md` documents clean-clone setup and frontend dependency installation.
- [ ] The root command intents and their selected names are documented.
- [ ] Local API and frontend startup are documented.
- [ ] The API default `HTTP_ADDR` of `:8080` and the configured frontend development port behavior are documented.
- [ ] `GET /health/live` and its limited liveness meaning are documented.
- [ ] Commands for all Go and frontend tests are documented.
- [ ] Story boundaries and explicit exclusions are documented.
- [ ] PostgreSQL, Docker, migrations, OpenAPI, authentication, Azure, observability instrumentation, CI, and finance modules are not described as completed.

### Clean-clone verification criteria

- [ ] A new clone can install frontend dependencies using `npm ci`.
- [ ] The selected root verify-all command succeeds from the new clone.
- [ ] The Go API can be started and its liveness endpoint returns HTTP `200`.
- [ ] The React application can be started and displays the TALLY shell.
- [ ] Verification does not require uncommitted source, pre-existing `node_modules`, pre-existing compiled binaries, credentials, a database, or a cloud resource.
- [ ] Build caches may accelerate verification but are not required for success.
- [ ] The evidence record includes commit identifier, operating system, Go version, Node.js version, npm version, commands executed, and final result.

### Evidence record template

```markdown
## DLV-PLAT-001 clean-clone verification

- Commit:
- Operating system:
- Go version:
- Node.js version:
- npm version:
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

## 10. Delivery-item acceptance summary

`DLV-PLAT-001` is complete only when all five stories are complete and every condition below passes:

- [ ] The clean repository contains a minimal Go API and React application.
- [ ] Both applications build successfully.
- [ ] Both applications have at least one deterministic test and all tests pass.
- [ ] Shared root commands build, test, verify, and run the applications without silent omissions.
- [ ] `GET /health/live` returns a safe process-liveness response.
- [ ] Clean-clone setup and verification evidence are recorded.
- [ ] No scope owned by a later delivery item has been implemented or claimed as complete.

## 11. Explicit exclusions and follow-on ownership

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

## 12. Definition of Ready

This platform item is ready only when:

- [ ] The exact delivery item, parent epic, milestone, deliverable, and exit evidence are identified.
- [ ] Adjacent-item boundaries and explicit exclusions are understood.
- [ ] The canonical repository URL and Go module path are confirmed.
- [ ] The root command mechanism and frontend development-port behavior are selected; the API retains the approved `HTTP_ADDR` default `:8080`.
- [ ] The approved tool major versions are available or a documented setup path exists.
- [ ] Required build, test, startup, failure-propagation, and clean-clone evidence is identified.
- [ ] The five stories are small enough to implement as one or a short chain of reviewable changes.

Functional requirement IDs, authoritative finance records, finance UX states, authorization contracts, persistence contracts, correction paths, concurrency paths, and integration events are not applicable because this item establishes no product behavior or finance state.

## 13. Definition of Done

This delivery item is done only when:

- [ ] Every acceptance criterion in Sections 5–10 passes; none is silently deferred.
- [ ] Required code, dependency manifests, lockfiles, tests, and documentation are complete.
- [ ] Go build and tests pass.
- [ ] Frontend dependency installation, tests, and production build pass.
- [ ] The root verify-all command passes and its failure propagation is proven.
- [ ] The clean-clone verification passes and evidence is attached or linked.
- [ ] Documentation matches the versions, commands, ports, endpoint, and behavior actually implemented.
- [ ] The implementation diff is reviewed against the modular-monolith and frontend capability boundaries.
- [ ] No credentials, dependency directories, generated build output, test output, or local-only configuration are committed.
- [ ] Traceability to the delivery item and applicable technical specifications is current.
- [ ] No functional requirement, workflow, NFR qualification, adjacent delivery item, or M0 milestone is incorrectly marked complete.
- [ ] No critical or high unresolved defect remains.

### General Definition-of-Done applicability

| General control | Applicability to `DLV-PLAT-001` |
|---|---|
| Database migrations and repository integration tests | Not applicable; persistence belongs to `DLV-PLAT-002` and `DLV-PLAT-003`. |
| OpenAPI and generated contracts | Not applicable; owned by `DLV-PLAT-004`. |
| Authorization, sensitive finance data, and audit evidence | Not applicable; this item has no protected finance action or finance record. |
| Idempotency, concurrency, correction, and recovery paths | Not applicable to the repository shell; owned by later platform and capability items. Startup failure and command failure propagation are tested here. |
| Accessibility and localization qualification | Not completed here; owned by dedicated UX and NFR delivery items. The minimal shell must not block later adoption. |
| Structured logs, metrics, and traces | Not completed here; owned by `DLV-OPS-001`. Health output and ordinary startup failures must still avoid secrets. |
| Clean-environment demonstration | Applicable and mandatory through the clean-clone verification. |

## 14. Traceability and quality-gate contribution

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

## 15. Consistency review record

### Review pass 1 — Corrected

The first review identified and corrected:

- an unsupported `P0` assignment;
- a required `docs/` directory not present in the approved minimum repository baseline;
- an unconditional `go.sum` requirement even though it exists only when dependencies generate it;
- omission of the approved `chi` router from the Go API shell;
- exact root command names presented as source requirements rather than implementation aliases;
- a Definition of Done that allowed deferred acceptance criteria; and
- omission of the no-critical-or-high-defect completion rule.

### Review pass 2 — Passed

The final review passed all 44 structural and semantic checks covering:

- exact delivery-item scope and exit evidence;
- selected Go, HTTP, React, TypeScript, Node.js, and Vite baselines;
- backend and frontend repository boundaries;
- liveness semantics and the approved API default address;
- adjacent M0 delivery-item ownership;
- Definition-of-Ready and Definition-of-Done applicability;
- M0 quality-gate contribution without false completion claims; and
- Markdown structure, evidence completeness, and internal consistency.

**Final consistency result: PASS.**

## 16. Source references

- `02_work_breakdown_backlog_v1.0.md`
  - Platform foundation backlog: exact `DLV-PLAT-001` deliverable and exit evidence.
  - Adjacent platform items: separation of Docker, persistence, OpenAPI, primitives, idempotency, outbox/inbox, UX, operations, infrastructure, and CI.
  - Definitions of Ready and Done.
- `01_backend_module_specifications_v1.0.md`
  - Repository layout and dependency rules.
- `01_solution_architecture_overview_v1.0.md`
  - Modular-monolith direction and selected Go/React technology baseline.
- `05_frontend_ui_technical_specifications_v1.0.md`
  - Approved frontend structure and capability boundary rule.
- `08_observability_operations_specifications_v1.0.md`
  - `/health/live` semantics and safe health responses.
- `04_quality_testing_environment_plan_v1.0.md`
  - Continuous quality evidence, quality gates, and test-layer expectations.
- `03_dependencies_milestones_releases_v1.0.md`
  - `EP-PLAT-001` dependency position and M0 gate context.
