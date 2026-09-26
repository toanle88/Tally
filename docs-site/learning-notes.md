# Learning notes and design decisions

TALLY is intentionally built as a learning project. The useful unit of
progress is a repeatable vertical slice: domain behavior, persistence, API,
interface, authorization, tests, observability, recovery evidence, and clear
limits on what was not verified.

## Lessons so far

- A modular monolith keeps ownership visible while avoiding premature
  distributed-systems cost. A shared deployment does not justify shared table
  writes.
- Exact money, explicit accounting scope, immutable corrections, and
  optimistic concurrency are domain decisions—not late infrastructure polish.
- Authentication answers “who is this subject?” Authorization answers “may
  this operation happen in this scope, with this sensitivity and history?”
  TALLY keeps those boundaries separate and fails closed when a decision cannot
  be established.
- Durable outbox/inbox, idempotency, fingerprints, and result lookup are one
  coordination problem. Retries must return evidence of the established result
  rather than guessing whether a financial effect happened.
- Sensitive-access evidence should carry bounded metadata and references, not
  raw tokens, policy payloads, credentials, or protected values. Operational
  telemetry and authoritative audit evidence have different owners and
  retention rules.
- Shared abstractions should follow repeated concrete use. Foundational
  identity, money, scope, idempotency, evidence, and accessible interaction
  patterns are the deliberate exceptions.
- A local Azure learning profile is useful for infrastructure practice, but
  scale-to-zero and disposable resources do not satisfy production availability
  or recovery targets.

## Decision trail

The main architectural decisions are: modular monolith; PostgreSQL schema
ownership; OpenAPI-first REST; exact-decimal finance values; transactionally
coordinated outbox/inbox; Entra authentication with application-owned
authorization; and local-first, low-cost deployment profiles.

Architecture decisions and their rationale remain in the [system design
documents](https://github.com/toanle88/Tally/tree/main/docs/specs/system_design).
