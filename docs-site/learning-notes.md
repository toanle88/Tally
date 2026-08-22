# Learning notes and design decisions

TALLY is intentionally built as a learning project. The useful unit of progress is a repeatable vertical slice: domain behavior, persistence, API, interface, authorization, tests, observability, and recovery evidence.

Selected lessons:

- A modular monolith keeps ownership visible while avoiding premature distributed-systems cost.
- Exact money and immutable corrections are foundational decisions, not late hardening.
- Shared abstractions should follow repeated concrete use; foundational identity, money, scope, idempotency, and evidence primitives are the exception.
- Local development is the default, while Azure exercises deployment and recovery concerns without implying production qualification.

Architecture decisions and their rationale remain in the [system design documents](https://github.com/toanle88/Tally/tree/main/docs/specs/system_design).
