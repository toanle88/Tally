# Tally

A modern double-entry accounting application — currently in early engineering foundation phase.

---

## Prerequisites

- [Go](https://go.dev/dl/) 1.26.3 or later
- [Node.js](https://nodejs.org/) 24 LTS or later
- [pnpm](https://pnpm.io/installation) 11.9.0 (`corepack enable pnpm` or `npm install -g pnpm@11.9.0`)
- [Docker Engine](https://docs.docker.com/engine/install/) and [Docker Compose v2](https://docs.docker.com/compose/)

---

## Getting Started

```bash
# Install frontend dependencies
pnpm --dir web install --frozen-lockfile

# Start PostgreSQL (background, waits for health check)
make db-up

# Start the API server (listens on :8080 or $HTTP_ADDR)
pnpm dev-api

# Start the frontend dev server (separate terminal)
pnpm dev-web

# Run all checks (tests + build)
pnpm check
```

Local PostgreSQL configuration is managed through `.env`:

```bash
cp .env.example .env
```

For a new local environment:
```bash
make verify-database-clean
```

---

## Project Commands

### Root package.json scripts

All commands must be run from the repository root.

| Command | Description |
|---|---|
| `pnpm build` | Build both API and web |
| `pnpm build:api` | `go build ./cmd/api` |
| `pnpm build:web` | `tsc -b && vite build` (via `pnpm -C web run build`) |
| `pnpm test` | Run all tests (API + web) |
| `pnpm test:api` | `go test ./...` |
| `pnpm test:web` | `vitest run` (via `pnpm -C web run test`) |
| `pnpm check` | `pnpm run test && pnpm run build` |
| `pnpm dev-api` | `go run ./cmd/api` |
| `pnpm dev-web` | `vite` dev server (via `pnpm -C web run dev`) |

### Makefile targets — PostgreSQL lifecycle

| Target | Description |
|---|---|
| `make db-up` | Start PostgreSQL and wait for healthy status |
| `make db-wait` | Poll Docker health check up to 50 attempts, 1 s interval |
| `make db-status` | Show container status (`docker compose ps postgres`) |
| `make db-version` | Query `SHOW server_version` inside the container |
| `make db-down` | Stop the PostgreSQL container |
| `make db-migrate` | Apply Goose migrations (bootstrap schema, platform tables) |
| `make db-seed` | Apply the committed synthetic seed data |
| `make db-verify` | Verify migration status and seed checksum |
| `make db-prepare` | Run db-migrate → db-seed → db-verify in order, stop on failure |
| `make db-reset` | Destroy and recreate the PostgreSQL volume from scratch |
| `make db-migrate-status` | Show applied and pending Goose migrations per schema |
| `make db-migrate-validate` | Validate migration ordering and Goose syntax without connecting |
| `make db-migrate-create` | Create a migration skeleton: `make db-migrate-create SCHEMA=platform NAME=desc` |
| `make db-migrate-check` | Verify migration checksum inventory matches committed migrations |
| `make db-migrate-inventory` | Regenerate `db/migrations/checksums.sha256` |
| `make check` | Run migration validation, checksum check, and `go test ./...` |
| `make verify-database` | Run end-to-end database verification from current state |
| `make verify-database-clean` | Delete volume, recreate, and run full verification from scratch |

---

## Local PostgreSQL

A single PostgreSQL 18.4 (bookworm) development container is defined in
[`compose.yaml`](./compose.yaml) with a database-aware health check.

- Bound to `127.0.0.1:5432` (localhost only).
- Data persists across restarts via a named volume.
- Configured through environment variables in `.env`.
- For local learning and development only — no production availability,
  backup, recovery, or security qualification claim.

See `.env.example` for the full set of configurable variables.

### Database lifecycle commands

See [`scripts/README.md`](scripts/README.md) for script behavior, direct usage, and destructive-command warnings.

Run all database commands from the repository root.

| Command | Behavior |
|---|---|
| `make db-config` | Validates the effective Docker Compose configuration. |
| `make db-up` | Starts PostgreSQL in detached mode and waits until it is healthy. |
| `make db-wait` | Waits for PostgreSQL to become healthy and fails after a bounded timeout. |
| `make db-status` | Shows the PostgreSQL container and health state. |
| `make db-logs` | Shows the latest PostgreSQL logs. |
| `make db-shell` | Opens `psql` against the configured local database. |
| `make db-version` | Prints the running PostgreSQL server version. |
| `make db-down` | Stops the Compose environment while preserving the database volume. |
| `make db-migrate` | Applies Goose migrations from `db/migrations/bootstrap` and `db/migrations/platform`. |
| `make db-seed` | Applies the committed synthetic seed from `db/seeds/local/v1.sql` and records its checksum. |
| `make db-verify` | Checks the recorded seed checksum and runs `goose status` on both migration directories. |
| `make db-prepare` | Orchestrates wait → migrate → seed → verify in order; stops on first failure. |
| `make db-reset`   | Destroys the PostgreSQL volume (`docker compose down --volumes`), then runs `db-up` and `db-prepare`. Shows a warning before the destructive step. |

`make db-down` is non-destructive. It stops and removes the Compose
container and network but retains the named PostgreSQL data volume.

The commands use the checked-in `compose.yaml` definition and the local
configuration described above. Docker Engine and Docker Compose v2 must be
available.

Failures from Docker Compose, PostgreSQL health checks, `psql`, Goose
migrations, and seed verification are returned as non-zero command results.
These commands configure only the local learning environment; they do not
establish production readiness, finance schemas, Azure PostgreSQL, backup, or
disaster recovery.

---

## Architecture

Tally uses a **modular-monolith** architecture: a single deployable Go
application with enforced bounded-context module boundaries under
`internal/`, and a React single-page application frontend. The architecture
is specified in [`docs/specs/system_design/`](./docs/specs/system_design/).

The technology baseline (from the approved solution architecture):

| Area | Current | Planned |
|---|---|---|
| Backend | Go 1.26 + chi/v5 5.3.1 | — |
| Frontend | React 19, TypeScript, Vite | — |
| Package manager | pnpm 11.9.0 | — |
| Database | PostgreSQL 18 Docker Compose dev service, Goose migrations, seed/verify scripts, migration checksum inventory | pgx + sqlc |
| Styling | — | Tailwind CSS + daisyUI |
| Client state | — | TanStack Query |
| API contract | Single GET /health/live endpoint | REST/JSON + OpenAPI 3.1 |
| Infrastructure | — | Terraform + Azure |
| CI/CD | — | GitHub Actions |

---

## Current Repository Structure

```
.
├── cmd/
│   └── api/
│       └── main.go                    # Go API entry point, graceful shutdown
├── db/
│   ├── migrations/
│   │   ├── bootstrap/
│   │   │   └── 00001_create_platform_schema.sql
│   │   ├── platform/
│   │   │   └── 00001_create_local_seed_manifest.sql
│   │   └── checksums.sha256              # Migration integrity checksums
│   └── seeds/
│       └── local/
│           └── v1.sql                 # Synthetic seed data
├── internal/
│   ├── .gitkeep
│   └── platform/
│       └── httpx/
│           ├── health.go              # GET /health/live handler
│           └── health_test.go         # Liveness test
├── scripts/
│   ├── README.md                    # Script documentation
│   ├── db/
│   │   ├── migrate.sh                 # Goose migration runner
│   │   ├── seed.sh                    # Seed applier with checksum guard
│   │   └── verify.sh                  # Migration status + seed checksum verification
│   └── verify/
│       └── database.sh                # End-to-end database verification workflow
├── web/
│   ├── src/
│   │   ├── app/
│   │   │   ├── app.tsx                # App shell (<h1>TALLY</h1>)
│   │   │   ├── app.css
│   │   │   └── app.test.tsx           # Shell render test
│   │   ├── main.tsx                   # React entry point
│   │   ├── index.css
│   │   └── test/
│   │       └── setup.ts               # jest-dom matchers
│   ├── public/
│   │   └── favicon.svg
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig.json
│   ├── tsconfig.app.json
│   ├── tsconfig.node.json
│   ├── eslint.config.js
│   ├── package.json
│   ├── .gitignore
│   └── README.md                      # Vite scaffold notice (unused)
├── docs/
│   ├── backlog/                       # User stories (DLV-PLAT-001, DLV-PLAT-002, DLV-PLAT-003)
│   │   ├── epic-template.md
│   │   ├── milestone-template.md
│   │   ├── story-template.md
│   │   └── stories/
│   │       ├── DLV-PLAT-001_user_stories.md
│   │       └── DLV-PLAT-002_user_stories.md
│   ├── specs/                         # PRD, domain model, UX, NFR,
│   │                                  # system design, technical specs
│   └── verification/                  # Clean-clone and reproducibility evidence
│       ├── DLV-PLAT-001_clean_clone_evidence.md
│       └── DLV-PLAT-002_local-db-reproducibility.md
├── .agents/
│   ├── commands/
│   │   ├── review-branch-diff.md
│   │   └── update-readme.md
│   └── skills/
│       ├── create-pr/
│       │   └── SKILL.md
│       ├── review-current-branch-diff/
│       │   └── SKILL.md
│       └── update-readme/
│           └── SKILL.md
├── .env.example                       # Developer PostgreSQL config template
├── .gitattributes
├── compose.yaml                       # PostgreSQL 18.4 dev service + health check
├── Makefile                           # PostgreSQL lifecycle + migration targets
├── package.json                       # Root scripts (pnpm@11.9.0)
├── pnpm-lock.yaml
├── go.mod                             # Go 1.26.3, chi/v5 5.3.1, goose/v3 3.27.1
├── go.sum
├── AGENTS.md
├── ROADMAP.md
├── LICENSE                            # MIT
└── README.md
```

**What exists today:**
- Go API binary with a single endpoint: `GET /health/live` returns
  `{"status":"ok"}` with graceful shutdown.
- React + Vite app shell that renders the TALLY heading.
- Test infrastructure: Go tests via `go test`, frontend tests via Vitest +
  Testing Library.
- Supporting config: TypeScript, ESLint, Vite, Go modules, path alias (`@/`).
- Docker Compose PostgreSQL 18.4 development service with health check and
  `make` lifecycle commands (`db-up`, `db-wait`, `db-status`, `db-version`,
  `db-down`).
- Goose v3.27.1 database migrations for the `platform` schema and
  `local_seed_manifest` table, run via `make db-migrate`.
  `bootstrap` schema migration creates the `platform` schema.
- Migration validation (`make db-migrate-validate`), status (`make db-migrate-status`),
  and creation (`make db-migrate-create`) commands.
- Migration checksum inventory (`db/migrations/checksums.sha256`) with drift
  detection (`make db-migrate-check`).
- Versioned synthetic seed (`db/seeds/local/v1.sql`) with checksum-based
  drift detection, run via `make db-seed`.
- Migration and seed verification via `make db-verify`.
- Orchestrated `make db-prepare` that runs wait → migrate → seed → verify
  and stops on first failure.
- Destructive database reset (`make db-reset`) that warns, removes the volume,
  and recreates everything via `db-up` and `db-prepare`.
- Verification evidence: clean-clone reproducibility
  (`docs/verification/DLV-PLAT-001_clean_clone_evidence.md`) and local-db
  reproducibility (`docs/verification/DLV-PLAT-002_local-db-reproducibility.md`).
- Delivery planning artifacts: user stories for DLV-PLAT-001 through DLV-PLAT-003,
  backlog templates, architecture and technical specifications.

**What is specified but not yet implemented:**
- All 19 finance domain modules (GL, AP, AR, COA, etc.) — see
  [`docs/specs/system_design/02_application_module_design_v1.0.md`](./docs/specs/system_design/02_application_module_design_v1.0.md).
- Platform database foundation (`internal/platform/database` with pgx pool).
- sqlc generation workflow (sqlc configuration, queries, generated code).
- End-to-end persistence integration test (migrations + pgx + sqlc).
- CI drift detection workflow for migrations and generated code.
- OpenAPI specification, workers, authentication, authorization,
  observability, Terraform, Azure deployment, CI/CD.

---

## Planned Target Architecture

The approved architecture defines 19 bounded-context modules under
`internal/`, each following a consistent structure (planned):

```
internal/<module>/
├── domain/          # aggregates, value objects, policies, domain events
├── application/     # commands, queries, coordinators, transaction scripts
├── ports/           # repository, clock, authorization interfaces
└── adapters/        # postgres, HTTP mapping, integration-event mapping
```

The planned modules are: `organization`, `gl`, `ap`, `ar`, `payroll`,
`invoicing`, `payments`, `reporting`, `intercompany`, `revenue`,
`fixedassets`, `multicurrency`, `fiscalperiod`, `coa`, `bankfeeds`, `tax`,
`workflow`, `identity`, `audit`. None of these packages exist yet.

Cross-module rules (from the approved design):
- Domain packages import only the Go standard library.
- No module imports another module's repository or adapter.
- Cross-module calls use a published application interface or integration
  event.
- Each module owns its PostgreSQL schema.

---

## Finance Rules

TALLY enforces these design rules across all modules:

- Never use binary floating point for money.
- Posted or established financial records are immutable.
- Correct established facts through reversal, adjustment, amendment, return,
  unapplication, replacement, or compensation.
- Retriable state-changing operations require idempotency.
- Integration events use the transactional outbox.
- Material state changes require authorization and audit evidence.
- Modules must not directly access another module's adapter or owned schema.
- Shared technical packages must not contain capability-specific finance
  rules.
- Avoid unnecessary abstractions and dependencies.

---

## Roadmap

See [ROADMAP.md](./ROADMAP.md) for the full delivery plan spanning M0
(engineering foundation) through M9 (full-system qualification).

---

## License

[MIT](./LICENSE)
