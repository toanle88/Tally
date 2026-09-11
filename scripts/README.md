# Scripts

This directory contains repository automation used for local development, database lifecycle management, verification, and project tooling.

The **Makefile** is the supported developer interface. Most scripts are implementation details and should normally be invoked through their corresponding `make` targets.

---

# Directory Structure

```text
scripts/
├── db/
│   ├── migrate.sh      # Goose migration workflow
│   ├── seed.sh         # Local database seed
│   └── verify.sh       # Database verification
├── openapi/
│   ├── api-check.sh                    # Aggregate contract and artifact drift gate
│   ├── api-generate-check.sh          # Working-tree-safe Go generation check
│   ├── expected-go-artifacts.txt      # Generated Go artifact inventory
│   ├── expected-typescript-artifacts.txt # Generated TypeScript inventory
│   ├── go-generate.sh                 # Pinned ogen generation wrapper
│   ├── go-negative-check.sh           # Isolated negative generation checks
│   ├── typescript-generate.sh        # Bundled OpenAPI TypeScript generation
│   ├── typescript-client-check.sh    # TypeScript generation, drift, and compile check
│   └── typescript-negative-check.sh  # TypeScript negative generation checks
├── verify/
│   ├── database.sh      # End-to-end database verification
│   ├── terraform.sh     # Terraform boundary verification
│   ├── terraform-environments.sh # Environment profile verification
│   ├── terraform-modules.sh # Reusable module contract and validation gate
│   ├── terraform-tools.sh # Pinned Terraform verification tool check
│   ├── terraform-lint.sh # TFLint gate
│   ├── terraform-security.sh # Checkov gate and exception manifest validation
│   ├── terraform-plan-policy.js # Credential-free Terraform plan policy gate
│   ├── terraform-drift.sh # Authenticated refresh-only drift procedure
│   └── terraform-cost.sh # External-plan Infracost review gate
│   ├── outbox-inbox-persistence.sh # DLV-PLAT-007 User Story 2 persistence gate
│   ├── transactional-coordination.sh # DLV-PLAT-007 User Story 3 transaction gate
│   ├── outbox-dispatch.sh # DLV-PLAT-007 User Story 4 dispatch gate
│   ├── outbox-worker.sh # DLV-PLAT-007 User Story 5 worker/replay gate
│   ├── openapi-story1.sh # OpenAPI User Story 1 verification
│   ├── accessibility-negative.sh # Controlled axe failure proof
│   └── accessibility-qualification.sh # Focused/full accessibility qualification runner
└── README.md
```

---

# Database Scripts

## migrate.sh

Runs Goose database migrations for all migration sets.

Supported commands:

| Command | Description |
|----------|-------------|
| `up` | Apply pending migrations |
| `status` | Show migration status |
| `create <schema> <name>` | Create a SQL migration skeleton in an initialized schema migration directory |
| `validate` | Validate migration filenames and ordering without connecting to PostgreSQL |
| `check` | Verify migration checksum inventory |

Normally use the Make targets instead:

```bash
make db-migrate
make db-migrate-status
make db-migrate-validate
make db-migrate-create SCHEMA=platform NAME=add_example_table
make db-migrate-check
```

Direct usage:

```bash
./scripts/db/migrate.sh up
./scripts/db/migrate.sh status
./scripts/db/migrate.sh create platform add_example_table
./scripts/db/migrate.sh validate
./scripts/db/migrate.sh check
```

Supported migration creation schemas are `bootstrap` and `platform`. Do not create migration directories or history tables for future finance schemas until their owning delivery item introduces a real migration.

---

## seed.sh

Applies the local development seed.

Characteristics:

- idempotent
- records applied seed
- detects seed drift
- safe to execute multiple times

Recommended command:

```bash
make db-seed
```

---

## verify.sh

Verifies that the local database is correctly prepared.

Checks include:

- bootstrap migrations applied
- platform migrations applied
- no pending migrations
- seed manifest exists exactly once
- expected seed version is installed

Recommended command:

```bash
make db-verify
```

---

# Verification Scripts

## Terraform and protected remote state

The Terraform gate checks the four applicable roots, exact Terraform/provider
constraints, reviewed provider lockfiles, formatting, backend-disabled
initialization, validation, the protected bootstrap resource contract, distinct
state keys and identities, required security settings, and the absence of
credentials, state, plans, variables, generated output, and application/domain
code in workload roots:

```bash
make terraform-check
make terraform-modules-check
make terraform-environments-check
make terraform-tools-check
make terraform-lint-check
make terraform-security-check

# Authenticated, external-state operations; do not run without an approved environment.
ENVIRONMENT=dev make terraform-drift-check
ACTIVE_MONTH_COST=15 PLAN_JSON=/path/to/terraform-show.json make terraform-cost-check
```

The complete Story 5 evidence boundary and plan-review record are documented
in [DLV-IAC-001 User Story 5 verification](../docs/verification/DLV-IAC-001-us5-plan-policy-drift-cost.md).

## Accessibility qualification

The root `pnpm test:a11y:qualification` command runs all accessibility checks
and the controlled negative proof by default. Use an explicit target for a
focused rerun after an interaction change:

```bash
pnpm test:a11y:qualification -- --list-targets
pnpm test:a11y:qualification -- --target semantic
pnpm test:a11y:qualification -- --target all
```

The runner prints the affected target and verification-document paths. Its
result template, requirement traceability, defect decisions, and retest
example are recorded in
`docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md`.

## openapi-story1.sh

Verifies the OpenAPI User Story 1 structure and authoritative operation catalog:

```bash
./scripts/verify/openapi-story1.sh
```

## Go API generation

The OpenAPI Go workflow uses the repository-pinned `ogen v1.23.0` tool. The
wrapper accepts an optional output directory and input contract, then bundles
the contract with Redocly before generation:

```bash
make api-generate
make api-generate-check
make api-negative-check
make api-check
```

Generated output is written to `internal/platform/httpapi/generated/` and is
not a manual authoring surface. The expected file inventory is maintained in
`scripts/openapi/expected-go-artifacts.txt`. `make api-check` compares fresh
generation with the committed output and is the focused CI drift gate.

## TypeScript client generation

The TypeScript workflow bundles `contracts/openapi/openapi.yaml` with the
pinned Redocly CLI, then runs `@hey-api/openapi-ts 0.99.0` with its bundled
Fetch client. Generated output is written to `web/src/generated/api/` and is
not a manual authoring surface. The expected file inventory is maintained in
`scripts/openapi/expected-typescript-artifacts.txt`.

Use the wrapper as the supported entry point because it supplies a temporary
bundled OpenAPI input. Direct config execution expects the ignored fallback
bundle at `contracts/openapi/dist/openapi.bundle.yaml`.

Run:

```bash
make api-ts-generate
make api-ts-check
```

`api-ts-check` validates OpenAPI linting, deterministic temporary generation,
the generated marker, bundled Fetch client output, exact-decimal and
nullable/optional type assertions, and frontend TypeScript compilation. Clean
installation evidence is recorded separately. It also compares the complete
committed output directory with fresh generation, rejecting changed, missing,
deleted, or extra artifacts. `make api-negative-check` proves these failures
for both Go and TypeScript output.

The focused workflow at `.github/workflows/openapi.yml` installs the root and
frontend lockfiles with `--frozen-lockfile` and invokes `make api-check`.

## database.sh

Runs the complete database verification workflow.

Default mode:

```bash
make verify-database
```

or

```bash
./scripts/verify/database.sh
```

This verifies:

- shell syntax
- Docker Compose configuration
- PostgreSQL availability
- migrations
- migration idempotency
- seed idempotency
- database verification
- migration validation
- migration checksum inventory
- repository checks

## outbox-inbox-persistence.sh

Runs the focused DLV-PLAT-007 User Story 2 persistence gate:

```bash
make outbox-inbox-persistence-check
```

It validates migration checksums and syntax, sqlc source and generated-output
drift, PostgreSQL 18 outbox/inbox durability and constraint integration tests,
concurrent due-row claims, and `git diff --check`.

## outbox-dispatch.sh

Runs the focused DLV-PLAT-007 User Story 4 dispatcher gate:

```bash
make outbox-dispatch-check
```

It validates dispatcher unit, race, vet, package ownership, migration, SQLC,
and PostgreSQL lease, retry, fencing, and managed-exception integration tests.

## outbox-worker.sh

Runs the focused DLV-PLAT-007 User Story 5 worker and replay gate:

```bash
make outbox-worker-check
```

It validates worker lifecycle and admission quotas, dispatcher polling,
crash/restart evidence, duplicate and ordering behavior, replay identity and
immutability, migration rollback/upgrade behavior, SQLC drift, package
ownership, and PostgreSQL integration tests.

---

## Clean Verification

To prove a completely reproducible environment:

```bash
make verify-database-clean
```

or

```bash
./scripts/verify/database.sh --clean
```

This performs:

1. Delete PostgreSQL volume
2. Recreate database
3. Apply migrations
4. Apply seeds
5. Verify database
6. Run repository validation

> **Warning**
>
> This permanently deletes the local PostgreSQL Docker volume.

---

# Database Lifecycle

Typical local workflow:

```bash
make db-up
make db-migrate
make db-seed
make db-verify
```

Complete reproducibility verification:

```bash
make verify-database-clean
```

---

# Docker Usage

Database operations execute inside the Docker Compose PostgreSQL container.

A local installation of:

- PostgreSQL
- psql

is **not required**.

---

# Conventions

Scripts follow these conventions:

- Bash strict mode
- Execute from repository root
- Fail immediately on errors
- Produce deterministic output
- Safe to rerun unless explicitly documented otherwise

---

# Destructive Commands

The following commands remove the local PostgreSQL Docker volume:

```bash
make db-reset
```

```bash
make verify-database-clean
```

Use them only when a clean local database is required.

---

# Prerequisites

- Docker
- Docker Compose
- Bash
- GNU Make
- Go

---

# Development Guidelines

- Prefer Make targets over invoking scripts directly.
- Keep scripts platform-independent where practical.
- Database migrations are immutable after being committed.
- Update `db/migrations/checksums.sha256` whenever a migration is added.
- Migration descriptions should be reviewable before merge. When a migration reaches business data, document lock risk, expected duration, forward-fix or rollback approach, backup need, and a verification query in the review evidence.
- Seed drift must fail verification rather than overwrite existing metadata.
- Verification scripts should remain deterministic and repeatable.
