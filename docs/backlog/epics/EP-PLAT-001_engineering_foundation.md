# [EPIC] EP-PLAT-001: Engineering Foundation and Development Workspace

## Outcome
Establish a fully functional, local-first development workspace and initial project repository structure. This foundation enables continuous integration, automated builds, database migration tooling, and localized execution across all platform layers (Go backend, React SPA frontend, PostgreSQL persistence) without introducing premature business domain logic.

## Learning Objective
Understand how to set up a clean, maintainable monorepo layout for a Go modular monolith, configure local containerized dependencies (PostgreSQL), establish reproducible schema migration pipelines with `sqlc` and `pgx`, set up a Vite-based React shell with Tailwind CSS and daisyUI, and build automated quality verification workflows using GitHub Actions.

## Scope
- Root directory layout conforming to TALLY technical specifications.
- Containerized local environment using Docker Compose (`PostgreSQL 18`).
- Go backend workspace initialization with `cmd/api`, `cmd/worker`, and `internal/` module layout.
- Database migration tool configuration and initial schema baseline.
- React/TypeScript frontend SPA shell initialization with Vite, Tailwind CSS, daisyUI, and MSW for local API simulation.
- Basic GitHub Actions CI workflow for linting, testing, and contract verification (`QG-01`).

## Explicit Exclusions
- Production deployment infrastructure or Azure provisioning (handled in deployment epics).
- Domain aggregate logic, business commands, or financial event handling (reserved for milestone capability epics).
- Full Entra ID OAuth integration (mock/local fallback used for engineering setup).

## Source References
- `01_delivery_strategy_roadmap_v1.0.md` (§3 "Phase 1: Foundation and Configuration")
- `03_dependencies_milestones_releases_v1.0.md` (§2 "Epic dependency matrix")
- `01_backend_module_specifications_v1.0.md` (§2 "Repository layout", §3 "Dependency rules")
- `05_frontend_ui_technical_specifications_v1.0.md` (§1 "SPA Architecture", §11 "Frontend test baseline")

## Traceability Identifiers
- **Milestone:** M0 — Engineering Foundation
- **Requirement IDs (FR / GFR):** GFR-001, GFR-022
- **Workflow IDs (WF):** N/A
- **NFR IDs:** NFR-MNT-001, NFR-MNT-002, NFR-TST-001
- **Quality Gate:** QG-01

## Dependencies
- None (First Epic in critical path).

## Acceptance Criteria
- [ ] Docker Compose starts a clean local PostgreSQL instance accessible on port 5432.
- [ ] Go backend workspace compiles cleanly (`go build ./...`) and executes basic health checks (`/health/live`, `/health/ready`).
- [ ] Database migration scripts execute forward and backward cleanly without error.
- [ ] React SPA shell boots locally via Vite (`npm run dev`) and renders the layout shell using daisyUI themes.
- [ ] GitHub Actions CI workflow runs automatically on PRs to check code formatting, run unit tests, and validate build integrity.

## Suggested Implementation Breakdown (Child Features / Stories)
- [ ] `DLV-PLAT-001`: Establish Repository Structure, Docker Compose, Go Monolith Skeleton, React SPA Shell, Migration Pipeline, and CI Baseline.

## Required Test Evidence
- [ ] Automated CI pipeline execution log (Green build).
- [ ] Successful database migration run output log.
- [ ] Passing frontend and backend smoke tests.

## Definition of Done
- [ ] All child stories completed and passed review.
- [ ] No critical/high open defects.
- [ ] Architectural boundaries and domain invariants preserved.

## Risks or Open Questions
- Ensure local toolchain versions (Go 1.26+, Node 20+, Docker) match project prerequisites across workstations.