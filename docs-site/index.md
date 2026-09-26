# TALLY

## A finance platform built to learn by doing

TALLY is a local-first, double-entry finance platform and solo, part-time
learning project. It is a working engineering foundation with an expanding
identity and access slice—not a finished accounting product and not a
production-qualified service.

The project uses a deliberately demanding target domain: journals, ledgers,
subledgers, payments, close, reporting, tax, payroll, audit integrity, and
cross-context coordination. The implementation is delivered as vertical
slices, with the domain rules, persistence, API, UX, authorization,
observability, recovery, and evidence considered together.

## What exists today

The current checkout contains:

- A Go API and worker, React/TypeScript SPA, PostgreSQL migrations, `pgx`,
  `sqlc`, and an OpenAPI-first contract workflow.
- Exact-decimal money and currency primitives, explicit accounting scope,
  aggregate versions, canonical request fingerprints, and scoped idempotency.
- Durable outbox/inbox coordination, transactional publication, lease-safe
  dispatch, typed retry, worker lifecycle controls, crash recovery, duplicate
  handling, ordering checks, and controlled replay foundations.
- Entra/OIDC authentication boundaries, application-owned IAM users, roles,
  access policies, segregation rules, emergency access, and metadata-only
  sensitive-access evidence contracts.
- A shared finance UX shell with scope context, worklists, record details,
  lifecycle/evidence surfaces, conflict recovery, masked data guards, and
  accessibility checks.
- Structured logs, OpenTelemetry context/traces/metrics, operational health
  views, alert/runbook contracts, Terraform checks, and optional low-cost Azure
  learning infrastructure.

These are foundations and synthetic workflow surfaces. The finance bounded
contexts that own journals, invoices, payments, tax, payroll, and reporting
remain future delivery scope. See [current status and evidence](/current-status)
for the boundary between implemented code, local verification, and planned
capability.

## Explore TALLY

- [Current status & evidence](/current-status): what is implemented, what is
  only a contract or fixture, and where verification is limited.
- [Architecture](/architecture): the modular-monolith shape, ownership rules,
  persistence, integration, and deployment profiles.
- [Finance integrity](/finance-integrity): exact money, immutable facts,
  correction lineage, idempotency, and audit boundaries.
- [Identity & access](/identity-access): authentication, authorization,
  segregation, emergency access, privacy, and evidence.
- [Operations & delivery](/operations): telemetry, workers, runbooks,
  Terraform, CI, and recovery boundaries.
- [UX foundation](/ux): the shared interaction model and currently available
  screens and components.
- [Roadmap and progress](/roadmap): milestones, stopping points, and delivery
  status from the repository roadmap.
- [Local development](/development): reproducible commands for the API, web
  app, database, and docs site.

The repository specifications remain authoritative; this site is a maintained
public guide to the decisions and evidence.
