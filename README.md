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
|---|---|---|
| `make db-config` | Validate Docker Compose configuration |
| `make db-up` | Start PostgreSQL and wait for healthy status |
| `make db-wait` | Poll Docker health check up to 50 attempts, 1 s interval |
| `make db-status` | Show container status (`docker compose ps postgres`) |
| `make db-logs` | Show the latest PostgreSQL logs |
| `make db-shell` | Open `psql` against the configured local database |
| `make db-version` | Query `SHOW server_version` inside the container |
| `make db-down` | Stop the PostgreSQL container (preserves data volume) |
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
| `make db-sqlc-version` | Show the pinned sqlc version |
| `make db-sqlc-generate` | Regenerate typed Go query code from committed SQL source |
| `make db-sqlc-check` | Regenerate sqlc output, detect drift, and run Go tests |
| `make db-sqlc-integration` | Execute the sqlc platform seed-manifest integration test |
| `make persistence-tools-version` | Show pinned Goose and sqlc versions used by the release gate |
| `make sqlc-compile` | Parse and type-check committed sqlc schema and query source |
| `make sqlc-check` | Compare committed sqlc output with deterministic generated output |
| `make persistence-integration-test` | Run PostgreSQL 18 Testcontainers persistence integration tests |
| `make persistence-check` | Run the focused persistence drift gate used by CI |
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

---

## Persistence Verification

The focused persistence release gate is:

```bash
make persistence-check
```

It verifies the repository-pinned Goose and sqlc tools, validates Goose
migration sets, checks `db/migrations/checksums.sha256`, compiles sqlc source,
detects stale or manually edited generated sqlc output, runs Go tests, and then
runs the PostgreSQL 18 Testcontainers persistence integration tests.

Author SQL in `db/queries/` and regenerate committed output with
`make db-sqlc-generate`. Generated files under
`internal/platform/database/platformdb/` are machine-produced and must not be
manually edited. Author migrations under the owning schema directory in
`db/migrations/`, update the checksum inventory with
`make db-migrate-inventory`, and verify drift with `make db-migrate-check`.

Shared-environment migration correction is forward-fix by default. Destructive
reset commands such as `make db-reset` and `make verify-database-clean` are for
disposable local development state only.

The GitHub Actions workflow at `.github/workflows/persistence.yml` invokes the
same `make persistence-check` command for focused DLV-PLAT-003 persistence
drift detection. It does not complete the broader DLV-CI-001 pull-request
quality pipeline.

---

## Architecture

Tally uses a **modular-monolith** architecture: a single deployable Go
application with enforced bounded-context module boundaries under
`internal/`, and a React single-page application frontend. The architecture
is specified in [`docs/specs/system_design/`](./docs/specs/system_design/).

The technology baseline (from the approved solution architecture):

| Area | Current | Planned |
|---|---|---|
| Backend | Go 1.26.3 + chi/v5 5.3.1 | — |
| Frontend | React 19, TypeScript, Vite | — |
| Package manager | pnpm 11.9.0 | — |
| Database | PostgreSQL 18 Docker Compose dev service, Goose migrations, pgx/v5 connection pool, sqlc-generated platform queries, seed/verify scripts, migration checksum inventory, focused persistence drift command and CI workflow | Broader PR quality-pipeline integration |
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
│   ├── queries/
│   │   └── platform/
│   │       └── local_seed_manifest.sql    # sqlc source query
│   └── seeds/
│       └── local/
│           └── v1.sql                 # Synthetic seed data
├── internal/
│   ├── .gitkeep
│   └── platform/
│       ├── database/
│       │   ├── pool.go                 # pgx connection pool with config validation
│       │   ├── pool_test.go            # Pool validation and security unit tests
│       │   ├── integration_fixture_test.go    # testcontainers fixture setup
│       │   ├── integration_test.go     # PostgreSQL 18 integration tests
│       │   ├── migration_integration_test.go  # migration apply/verify helpers
│       │   ├── transaction_integration_test.go # generated query commit/rollback proof
│       │   └── platformdb/             # sqlc-generated platform query package
│       │       ├── db.go
│       │       ├── local_seed_manifest.sql.go
│       │       └── models.go
│       └── httpx/
│           ├── health.go              # GET /health/live handler
│           └── health_test.go         # Liveness test
├── scripts/
│   ├── README.md                    # Script documentation
│   ├── db/
│   │   ├── migrate.sh                 # Goose migration runner
│   │   ├── sqlc-check.sh              # sqlc generation drift check
│   │   ├── seed.sh                    # Seed applier with checksum guard
│   │   └── verify.sh                  # Migration status + seed checksum verification
│   ├── tools/
│   │   └── sqlc.sh                    # Pinned sqlc launcher
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
│   ├── pnpm-lock.yaml
│   ├── .gitignore
│   └── README.md                      # Vite scaffold notice (unused)
├── docs/
│   ├── backlog/                       # User stories (DLV-PLAT-001, DLV-PLAT-002, DLV-PLAT-003)
│   │   ├── epic-template.md
│   │   ├── milestone-template.md
│   │   ├── story-template.md
│   │   └── stories/
│   │       ├── DLV-PLAT-001_user_stories.md
│   │       ├── DLV-PLAT-002_user_stories.md
│   │       └── DLV-PLAT-003_user_stories.md
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
├── .gitignore
├── compose.yaml                       # PostgreSQL 18.4 dev service + health check
├── Makefile                           # PostgreSQL lifecycle + migration targets
├── package.json                       # Root scripts (pnpm@11.9.0)
├── pnpm-lock.yaml
├── sqlc.yaml                            # sqlc code-generation configuration
├── go.mod                             # Go 1.26.3, chi/v5 5.3.1, goose/v3 3.27.1
├── go.sum
├── AGENTS.md
├── ROADMAP.md
├── LICENSE                            # MIT
└── README.md
```

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
