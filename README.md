# Tally

A modern double-entry accounting application for managing ledgers, vouchers, invoices, inventory, taxes, financial reports, and business transactions.

**Status:** Early development — monorepo foundation (M0) in progress. See [ROADMAP.md](./ROADMAP.md).

---

## Shared root commands

The shared commands below must be run from the repository root.

### Prerequisites

- Go 1.26.x
- Node.js 24 LTS
- pnpm using the exact version pinned in `web/package.json`
- Frontend dependencies installed with:

  ```bash
  pnpm --dir web install --frozen-lockfile
  ```

---
  
## Architecture

Tally uses a **modular-monolith** architecture: a single deployable Go application with enforced bounded-context module boundaries under `internal/`, and a React single-page application frontend. The architecture is specified in [`docs/specs/system_design/`](./docs/specs/system_design/).

The target technology baseline (from the approved solution architecture):

| Area | Selection |
|---|---|
| Backend | Go 1.26 + chi router |
| Frontend | React 19, TypeScript, Vite |
| Package manager | pnpm (pinned `pnpm@11.9.0`) |
| Database | PostgreSQL (planned) |
| Database access | pgx + sqlc (planned) |
| Migrations | Goose (planned) |
| Styling | Tailwind CSS + daisyUI (planned) |
| Client state | TanStack Query (planned) |
| API contract | REST/JSON + OpenAPI 3.1 (planned) |
| Infrastructure | Terraform + Azure (planned) |
| CI/CD | GitHub Actions (planned) |

---

## Current Repository Structure

```
.
├── cmd/
│   └── api/
│       └── main.go                    # Go API entry point
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
│   ├── backlog/                       # User stories (DLV-PLAT-001), templates
│   └── specs/                         # PRD, domain model, UX, NFR,
│                                      # system design, technical specs
├── .opencode/
│   ├── commands/                      # review-branch-diff, update-readme
│   └── skills/                        # review-current-branch-diff, update-readme
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
- Go API binary with a single endpoint: `GET /health/live` returns `{"status":"ok"}` with graceful shutdown.
- React + Vite app shell that renders the TALLY heading (moved to `web/src/app/`).
- Test infrastructure: Go tests via `go test`, frontend tests via Vitest + Testing Library.
- Supporting config: TypeScript, ESLint, Vite, Go modules, path alias (`@/`).

**What is specified but not yet implemented:**
- All 19 finance domain modules (GL, AP, AR, COA, etc.) — see [`docs/specs/system_design/02_application_module_design_v1.0.md`](./docs/specs/system_design/02_application_module_design_v1.0.md).
- Database, migrations, OpenAPI, workers, authentication, observability, Terraform, Azure deployment, CI/CD.

---

## Planned Target Architecture

The approved architecture defines 19 bounded-context modules under `internal/`, each following a consistent structure (planned):

```
internal/<module>/
├── domain/          # aggregates, value objects, policies, domain events
├── application/     # commands, queries, coordinators, transaction scripts
├── ports/           # repository, clock, authorization interfaces
└── adapters/        # postgres, HTTP mapping, integration-event mapping
```

The planned modules are: `organization`, `gl`, `ap`, `ar`, `payroll`, `invoicing`, `payments`, `reporting`, `intercompany`, `revenue`, `fixedassets`, `multicurrency`, `fiscalperiod`, `coa`, `bankfeeds`, `tax`, `workflow`, `identity`, `audit`. None of these packages exist yet.

Cross-module rules (from the approved design):
- Domain packages import only the Go standard library.
- No module imports another module's repository or adapter.
- Cross-module calls use a published application interface or integration event.
- Each module owns its PostgreSQL schema.

---

## Finance Rules

TALLY enforces these design rules across all modules:

- Never use binary floating point for money.
- Posted or established financial records are immutable.
- Correct established facts through reversal, adjustment, amendment, return, unapplication, replacement, or compensation.
- Retriable state-changing operations require idempotency.
- Integration events use the transactional outbox.
- Material state changes require authorization and audit evidence.
- Modules must not directly access another module's adapter or owned schema.
- Shared technical packages must not contain capability-specific finance rules.
- Avoid unnecessary abstractions and dependencies.

---

## Prerequisites

- [Go](https://go.dev/dl/) 1.26.3 or later
- [Node.js](https://nodejs.org/) 24 LTS or later
- [pnpm](https://pnpm.io/installation) 11.9.0 (`corepack enable pnpm` or `npm install -g pnpm@11.9.0`)

---

## Getting Started

```bash
# Clone the repository
git clone https://github.com/toanle88/Tally.git
cd Tally

# Install frontend dependencies
pnpm install

# Run all checks (tests + build)
pnpm check

# Start the API server (listens on :8080 or $HTTP_ADDR)
pnpm dev-api

# Start the frontend dev server (in another terminal)
pnpm dev-web
```

---

## Project Commands

All commands are defined in the root [`package.json`](./package.json).

| Command | Description |
|---|---|
| `pnpm build` | Build both API and web |
| `pnpm build:api` | `go build ./cmd/api` |
| `pnpm build:web` | `pnpm -C web run build` (tsc + vite) |
| `pnpm test` | Run all tests (API + web) |
| `pnpm test:api` | `go test ./...` |
| `pnpm test:web` | `pnpm -C web run test` (vitest) |
| `pnpm check` | `pnpm run test && pnpm run build` |
| `pnpm dev-api` | `go run ./cmd/api` |
| `pnpm dev-web` | `pnpm -C web run dev` (vite dev server) |

---

## Roadmap

See [ROADMAP.md](./ROADMAP.md) for the full delivery plan spanning M0 (engineering foundation) through M9 (full-system qualification). The current checkpoint is **DLV-PLAT-001** — monorepo foundation. All five User Stories under DLV-PLAT-001 are complete.

---

## License

[MIT](./LICENSE)