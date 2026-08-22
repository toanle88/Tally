# Architecture

TALLY uses a modular monolith: one Go deployable with explicit internal bounded-context boundaries, a React SPA, and PostgreSQL schemas owned by their corresponding modules.

```text
React SPA → HTTP/API adapters → application modules → owned PostgreSQL schemas
                                      ↘ outbox and integration workers
```

The design is local-first and cost-conscious for learning. Azure has a separate demonstration/production-qualification path; the learning deployment is not represented as production-qualified.

## Ownership boundaries

Each finance fact has one owning bounded context. Modules communicate through application contracts and events. They do not reach into another module’s adapters or database schema. Shared packages contain technical primitives only.

See the [solution architecture overview](https://github.com/toanle88/Tally/blob/main/docs/specs/system_design/01_solution_architecture_overview_v1.0.md) for the canonical architecture.
