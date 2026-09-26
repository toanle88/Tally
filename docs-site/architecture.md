# Architecture

TALLY is a modular monolith: one Go API deployable, an optional Go worker,
one React SPA, and one PostgreSQL database whose schemas are owned by their
bounded contexts.

```text
Browser → React SPA → HTTP/API adapters → application modules → owned schemas
                                             ↘ coordinators and platform ports
                                                ↘ PostgreSQL outbox/inbox → worker
```

The architecture keeps domain ownership visible while avoiding the cost and
operational burden of premature service extraction. Local development is the
default. Azure has a separate disposable learning profile and a production
reference profile; neither should be confused with production qualification.

## Ownership is the primary constraint

Each business fact has one authoritative owner. A module owns its domain
model, application ports, repositories, migrations, and PostgreSQL schema.
Other modules exchange identifiers, published contracts, and events; they do
not query or mutate another module’s tables directly. Shared packages contain
technical primitives such as money, scope, versions, idempotency, transport,
and telemetry—not finance rules.

The domain baseline defines 19 bounded contexts:

| Group | Contexts |
| --- | --- |
| Core accounting | General Ledger; Accounts Payable; Accounts Receivable; Invoicing; Financial Reporting; Multi-Entity / Intercompany; Revenue Recognition; Fixed Assets; Multi-Currency; Fiscal Period Management |
| Supporting business | Organization & Master Data; Payroll; Payments & Cash Management; COA Segment Accounting; Bank Feeds & Reconciliation; Tax Filing; Audit Integrity |
| Generic/control | Workflow & Approvals; Identity & Access |

The table describes the approved domain target. The current repository has
implemented platform foundations and a growing Identity & Access slice; it has
not implemented all 19 business modules.

## Request and data flow

1. The SPA establishes an authenticated application actor and selected
   accounting scope.
2. API middleware validates the external identity, correlation context, and
   protected-route requirements.
3. The owning application module evaluates authorization, idempotency,
   expected version, domain rules, and its local transaction boundary.
4. The owner persists authoritative state and, when needed, an outbox event in
   the same PostgreSQL transaction.
5. The worker claims and dispatches durable outbox work. Consumers use inbox
   identity and local effects to make retries safe.
6. Telemetry records bounded operational signals; audit evidence is written
   through an audit port and is not replaced by logs or traces.

## Persistence and integration

PostgreSQL is the system of record. The design uses Goose migrations, explicit
SQL, generated `sqlc` access, local foreign keys and constraints, optimistic
aggregate versions, exact `NUMERIC` monetary storage, and forward-compatible
migration discipline.

The learning baseline does not require Kafka or Azure Service Bus. Integration
intent is stored in PostgreSQL, then dispatched after commit. Event envelopes
carry source context, aggregate identity/version, accounting scope, correlation
and causation references, classification, and a payload fingerprint. Duplicate
delivery returns the established inbox result; gaps and changed content under
one identity are treated as integrity problems.

## Deployment profiles

| Profile | Intended use | Shape | Qualification status |
| --- | --- | --- | --- |
| Local | Daily development and tests | Vite + Go processes, Docker Compose PostgreSQL | Developer environment only |
| Azure learning | Disposable demonstrations and Azure practice | Static Web Apps, scale-to-zero Container Apps, small PostgreSQL Flexible Server, Key Vault/Monitor | Low-cost learning profile; not production-qualified |
| Production reference | Future formal qualification | Multiple API replicas, separately scalable worker, HA/private PostgreSQL, backups and restore evidence | Design target, not a claimed deployment |

## Current implementation boundary

The runtime currently demonstrates health, authenticated API boundaries,
Identity & Access operations, and platform coordination. The OpenAPI contract
catalogues the wider future API surface, but contract entries are not proof
that every capability route is implemented. The web application similarly
contains shared and synthetic operational screens; its placeholder capability
routes do not own finance records.

See the [solution architecture overview](https://github.com/toanle88/Tally/blob/main/docs/specs/system_design/01_solution_architecture_overview_v1.0.md),
[application/module design](https://github.com/toanle88/Tally/blob/main/docs/specs/system_design/02_application_module_design_v1.0.md),
and [data/integration architecture](https://github.com/toanle88/Tally/blob/main/docs/specs/system_design/03_data_integration_architecture_v1.0.md)
for the canonical design.
