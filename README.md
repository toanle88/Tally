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
| `make db-wait` | Poll Docker health check up to 30 s |
| `make db-status` | Show container status (`docker compose ps postgres`) |
| `make db-version` | Query `SHOW server_version` inside the container |
| `make db-down` | Stop the PostgreSQL container |

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

`make db-down` is non-destructive. It stops and removes the Compose
container and network but retains the named PostgreSQL data volume.

The commands use the checked-in `compose.yaml` definition and the local
configuration described above. Docker Engine and Docker Compose v2 must be
available.

Failures from Docker Compose, PostgreSQL health checks, and `psql` are
returned as non-zero command results. These commands configure only the local
learning environment; they do not establish production readiness, database
migrations, finance schemas, Azure PostgreSQL, backup, or disaster recovery.

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
| Database | PostgreSQL 18 Docker Compose dev service | pgx + sqlc, migrations (Goose) |
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
├── internal/
│   ├── .gitkeep
│   └── platform/
│       └── httpx/
│           ├── health.go              # GET /health/live handler
│           └── health_test.go         # Liveness test
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
│   ├── backlog/                       # User stories (DLV-PLAT-001, DLV-PLAT-002)
│   │   ├── epic-template.md
│   │   ├── milestone-template.md
│   │   ├── story-template.md
│   │   └── stories/
│   │       ├── DLV-PLAT-001_clean_clone_evidence.md
│   │       ├── DLV-PLAT-001_user_stories.md
│   │       └── DLV-PLAT-002_user_stories.md
│   └── specs/                         # PRD, domain model, UX, NFR,
│                                      # system design, technical specs
├── .opencode/
│   ├── commands/
│   │   ├── review-branch-diff.md
│   │   └── update-readme.md
│   └── skills/
│       ├── review-current-branch-diff/
│       │   └── SKILL.md
│       └── update-readme/
│           └── SKILL.md
├── .env.example                       # Developer PostgreSQL config template
├── compose.yaml                       # PostgreSQL 18.4 dev service + health check
├── Makefile                           # PostgreSQL lifecycle targets
├── package.json                       # Root scripts (pnpm@11.9.0)
├── pnpm-lock.yaml
├── go.mod                             # Go 1.26.3, chi/v5 5.3.1
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
- Delivery planning artifacts: user stories, backlog templates, architecture
  and technical specifications.

**What is specified but not yet implemented:**
- All 19 finance domain modules (GL, AP, AR, COA, etc.) — see
  [`docs/specs/system_design/02_application_module_design_v1.0.md`](./docs/specs/system_design/02_application_module_design_v1.0.md).
- App-level database access (pgx, sqlc), migrations (Goose).
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