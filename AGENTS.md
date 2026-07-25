# TALLY Agent Instructions

## Purpose

Help build TALLY as a solo, part-time learning project. Prioritize understanding, correctness, simplicity, and consistency over speed.

## Source authority

Read the relevant documents before changing code or documentation:

1. `docs/specs/finance_domain_model_ddd.md` — domain language, ownership, invariants, lifecycle, commands, and events.
2. `docs/specs/prd/` — functional requirements and acceptance behavior.
3. `docs/specs/finance_ux_workflow_specification_v1.0.md` — user interaction and workflow behavior.
4. `docs/specs/finance_nonfunctional_requirements_v1.0.md` — quality and operational targets.
5. `docs/specs/system_design/` — approved solution architecture and ADRs.
6. `docs/specs/technical_specifications/` — implementation details.
7. `docs/specs/finance_delivery_plan_v1.0.md`, `docs/backlog/`, and `ROADMAP.md` — delivery scope and status.

When sources conflict, report the conflict. Do not invent a resolution, requirement, identifier, API, schema, command, event, or architecture decision.

Repository files show what currently exists. Successful command output shows what has actually been verified.

## Architecture rules

- Preserve the approved modular-monolith architecture and bounded-context ownership.
- Keep finance rules inside their owning module.
- Do not access another module's adapter or database schema directly.
- Shared packages contain technical primitives only.
- Avoid unnecessary abstractions and new dependencies.
- Do not implement future delivery scope early.

## Finance rules

- Never use binary floating point for money.
- Posted or established financial facts are immutable.
- Correct established facts through reversal, adjustment, amendment, return, unapplication, replacement, or compensation.
- Retriable state-changing operations require idempotency.
- Integration events use the transactional outbox.
- Material state changes require authorization and audit evidence.

## Working rules

- Inspect the current branch and relevant specifications before editing.
- Make the smallest change that satisfies the active scope.
- Add or update tests for changed behavior.
- Never claim code works without successful command or test evidence.
- Do not commit generated dependencies, build output, binaries, secrets, local environment files, or alternate frontend lockfiles.
- pnpm is the sole frontend package manager.
- Do not stage, commit, reset, clean, or revert unless explicitly requested.
