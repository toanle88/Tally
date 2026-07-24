[STORY] DLV-PLAT-001: Establish Modular Monolith Repository Structure, Local Docker Compose Environment, Tooling, and CI Baseline

Outcome

A clean, reproducible, and fully integrated repository structure that starts a local PostgreSQL 18 database via Docker Compose, enforces Go 1.26+ and React/Vite/TypeScript compilation, runs `sqlc` code generation, lints OpenAPI contracts, and passes initial architecture boundary linter checks in CI.

Learning Objective

Learn how to structure a Go modular monolith preserving strict context isolation, configure local multi-container development workflows, set up code-generation pipelines (`sqlc`, OpenAPI), and enforce architectural boundaries using static analysis in GitHub Actions.

Scope

- Root directory layout for backend (`/internal`, `/cmd/server`) and frontend (`/web`).
- Go 1.26+ module initialization (`go.mod`, `go.sum`).
- Docker Compose setup (`docker-compose.yml`) running PostgreSQL 18 with health checks.
- Database migration setup using Goose and initial schema files for platform tables (`outbox`, `inbox`).
- `sqlc` configuration (`sqlc.yaml`) and initial query generation pipeline.
- OpenAPI linting setup using Redocly CLI and TypeScript type generation.
- React + TypeScript + Vite frontend shell under `/web` with Tailwind CSS and daisyUI dependencies configured.
- Architecture static linter rules enforcing forbidden cross-module Go package imports.
- GitHub Actions CI workflow (`.github/workflows/ci.yml`) running build, linting, and unit test execution.

Explicit Exclusions

- Functional domain aggregates, HTTP handlers, or REST controllers for business capabilities (e.g., General Ledger, Accounts Payable).
- Microsoft Entra ID or live Azure cloud deployment (local-first mock/placeholder configuration only).
- Advanced messaging brokers (Kafka, RabbitMQ, Azure Service Bus) or external caches (Redis).

Owning Bounded Context & Architecture

- Owning Bounded Context / Module: Platform / Cross-Cutting Infrastructure (`internal/platform`)
- Domain Aggregates & Invariants: None (Foundation setup; enforces architectural boundary invariants that domain code must not violate).

## Contract & Impact Analysis

- Application Commands / Queries: None
- API Impact (Routes / Schemas): Root health probe `/healthz` and OpenAPI contract validation.
- Database Impact (Schemas / Tables): PostgreSQL 18 database initialization with `platform` schema for outbox/inbox/migration metadata.
- Event or Worker Impact: Transactional outbox table schema created in `platform` schema.
- Authorization Impact (Permissions / Scope): None.
- Frontend Impact (Routes / Components): Base React Single Page Application (SPA) shell mounted with Tailwind CSS / daisyUI styling baseline.
- Observability Impact (Logs / Traces / Metrics): Go `slog` structured logger and OpenTelemetry SDK initialization skeleton.

## Failure, Concurrency & Idempotency Rules

- Expected Failure Cases: Database container connection timeout during local startup; cross-module import violation detected by architecture linter.
- Concurrency & Idempotency Considerations: Migration scripts must be strictly sequential and idempotent; Goose migration lock table prevents concurrent migration execution.

## Traceability Identifiers

- Milestone: M0 — Engineering Foundation
- Parent Epic ID: EP-PLAT-001
- Requirement IDs (FR / GFR): GFR-001, GFR-021, GFR-022
- Workflow IDs (WF): None (Platform foundation)
- NFR IDs: NFR-MNT-001, NFR-MNT-002, NFR-TST-001
- Quality Gate: QG-01

## Source References

- 01_solution_architecture_overview_v1.0.md (§2 "Architecture Baseline")
- 01_backend_module_specifications_v1.0.md (§2 "Repository Layout", §11 "Architecture Tests")
- 03_database_persistence_specifications_v1.0.md (§2 "Schema Catalog", §9 "Migration Policy")
- 05_frontend_ui_technical_specifications_v1.0.md (§1 "Frontend Technology Stack")
- 03_dependencies_milestones_releases_v1.0.md (§3 "Milestone Definition Catalog — M0")

Dependencies

- None (First delivery item in the delivery roadmap).

Acceptance Criteria

- [ ] `docker-compose up` cleanly provisions PostgreSQL 18 with accessible health status on port 5432.
- [ ] Go backend compiles without errors and passes `go test ./...`.
- [ ] `sqlc generate` executes without error against SQL query definitions.
- [ ] Frontend application compiles via `npm run build` inside `/web`.
- [ ] Architecture linter rejects any hypothetical Go import from `internal/<module_a>/...` into `internal/<module_b>/...`.
- [ ] GitHub Actions CI workflow completes successfully on PR/push.

Suggested Implementation Steps

1. Initialize Go root module `go.mod` and directory structure (`/cmd/server`, `/internal/platform`, `/internal/<bounded_context>`).
2. Create `/web` frontend directory with Vite, React, TypeScript, Tailwind CSS, and daisyUI.
3. Configure `docker-compose.yml` with PostgreSQL 18 service, environment variables, and volume mounts.
4. Set up Goose migration folder under `/migrations` and create baseline migration for outbox/inbox tables.
5. Add `sqlc.yaml` and Redocly OpenAPI linter scripts to package configs.
6. Implement architecture boundary linter test in Go verifying internal package isolation.
7. Create `.github/workflows/ci.yml` script to automate build, lint, and test steps.

Required Test Evidence

- [ ] Unit tests (Go package import boundary static tests).
- [ ] Integration / API tests (Database connection health check unit/integration test).
- [ ] Frontend / Component tests (Vite build and React shell mounting unit test).
- [ ] End-to-end / Workflow tests (CI pipeline execution log proving clean execution).

Likely Files / Packages Involved

- `go.mod`, `go.sum`
- `docker-compose.yml`
- `sqlc.yaml`
- `cmd/server/main.go`
- `internal/platform/architecture_test.go`
- `migrations/00001_initial_platform_schema.sql`
- `web/package.json`, `web/vite.config.ts`
- `.github/workflows/ci.yml`

Definition of Done

- [ ] Code implemented and reviewed.
- [ ] Required unit, integration, or end-to-end test evidence attached.
- [ ] Architectural rules (module boundaries, monetary types, immutability) preserved.

Risks or Open Questions

- Ensure local workstation has Docker Engine/Desktop running with sufficient resources before initiating `docker-compose up`.