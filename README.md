# Tally

A modern double-entry accounting application for managing ledgers, vouchers, invoices, inventory, taxes, financial reports, and business transactions.

**Status:** Early development — monorepo foundation (M0) in progress. See [ROADMAP.md](./ROADMAP.md).

---

## Architecture

Tally is planned as a **modular monolith**: a single deployable Go application with enforced bounded-context module boundaries under `internal/`, and a React single-page application frontend. The architecture is specified in [`docs/specs/system_design/`](./docs/specs/system_design/).

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
│       └── main.go              # Go API entry point
├── internal/
│   ├── .gitkeep
│   └── platform/
│       └── httpx/
│           ├── health.go        # GET /health/live handler
│           └── health_test.go   # Liveness test
├── web/
│   ├── src/
│   │   ├── main.tsx             # React entry point
│   │   ├── App.tsx              # App shell (<h1>TALLY</h1>)
│   │   ├── App.test.tsx         # Shell render test
│   │   ├── App.css
│   │   ├── index.css
│   │   └── test/
│   │       └── setup.ts         # jest-dom matchers
│   ├── public/
│   │   ├── favicon.svg
│   │   └── icons.svg
│   ├── index.html
│   ├── vite.config.ts
│   ├── tsconfig*.json
│   ├── eslint.config.js
│   └── package.json
├── docs/
│   ├── backlog/                 # User stories (DLV-PLAT-001)
│   └── specs/                   # PRD, domain model, UX, NFR,
│                                # system design, technical specs
├── package.json                 # Root scripts
├── pnpm-lock.yaml
├── go.mod
├── go.sum
├── ROADMAP.md
├── LICENSE                      # MIT
└── README.md
```

**What exists today:**
- Go API binary with a single endpoint: `GET /health/live` returns `{"status":"ok"}` with graceful shutdown.
- React + Vite app shell that renders the TALLY heading.
- Test infrastructure: Go tests via `go test`, frontend tests via Vitest + Testing Library.
- Supporting config: TypeScript, ESLint, Vite, Go modules.

**What is specified but not yet implemented:**
- All 19 finance domain modules (GL, AP, AR, COA, etc.) — see [`docs/specs/system_design/02_application_module_design_v1.0.md`](./docs/specs/system_design/02_application_module_design_v1.0.md).
- Database, migrations, OpenAPI, workers, authentication, observability, Terraform, Azure deployment, CI/CD.

---

## Planned Target Architecture

The approved architecture defines 19 bounded-context modules under `internal/`, each following a consistent structure:

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

See [ROADMAP.md](./ROADMAP.md) for the full delivery plan spanning M0 (engineering foundation) through M9 (full-system qualification). The current checkpoint is **DLV-PLAT-001** — monorepo foundation — with User Story 1 (monorepo structure) and User Story 2 (minimal Go API shell) complete. Stories 3–5 remain in progress.

---

## License

[MIT](./LICENSE)
