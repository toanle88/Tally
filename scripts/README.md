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
├── verify/
│   ├── database.sh      # End-to-end database verification
│   └── openapi-story1.sh # OpenAPI User Story 1 verification
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

## openapi-story1.sh

Verifies the OpenAPI User Story 1 structure and authoritative operation catalog:

```bash
./scripts/verify/openapi-story1.sh
```

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
