---
name: user-story-delivery
description: Turn a user story into a traceable implementation plan and, when implementation is requested, create a branch, update relevant documentation, and verify the change.
metadata:
  workflow: user-story-delivery
---

# User Story Delivery

Use this skill when the user supplies a user story and wants it taken through
planning or implementation. The user story is the only required input. Infer
routine metadata from the repository; ask a question only when an unresolved
ambiguity can change scope, ownership, correctness, or a destructive action.

## Project authority

Read repository-level agent instructions first, including `AGENTS.md` when
present. Then locate and inspect the project's authoritative requirements,
architecture, design, delivery, backlog, and implementation documentation.
Prefer sources in this order when they conflict:

1. Approved requirements and domain/product specifications
2. Approved architecture and technical specifications
3. Existing implementation and repository conventions
4. Backlog, roadmap, and delivery documentation
5. Successful command and test output

Use `rg` to locate requirement, acceptance, milestone, module, route, and
command identifiers. Do not invent identifiers or resolve conflicts silently;
record conflicts and assumptions in the plan.

## Workflow

### 1 — Determine the mode

A story, “plan”, or planning language starts **Plan** mode. “Implement”,
“build”, “ship”, or explicit approval starts **Implement** mode. If ambiguous,
produce the plan first and wait for implementation approval.

### 2 — Create the plan

Inspect the repository and produce a plan containing:

- outcome, learning objective, scope, explicit exclusions, and definition of done;
- owning component/module, domain objects, invariants, and dependency boundaries;
- mapped requirement, acceptance, milestone, quality-gate, and parent-item IDs
  when those identifiers exist;
- application behavior, API, database, integration, authorization, UI, and
  observability impact as applicable;
- failure, concurrency, retry, data-integrity, security, and compatibility behavior;
- ordered implementation steps, likely files/packages, dependencies, risks,
  acceptance criteria, and required test evidence.

Create or update the smallest appropriate backlog or planning document using an
existing repository template when available. Do not mark roadmap or delivery
items complete during planning.

### 3 — Suggest and create the branch

In Plan mode, suggest a branch but do not create it. Follow the repository's
documented naming convention; otherwise use an appropriate prefix such as
`feat/`, `fix/`, `refactor/`, or `docs/`, followed by a short kebab-case slug.

At the start of Implement mode:

1. Inspect `git status --short`, `git branch --show-current`, and the proposed
   branch name.
2. Do not switch away from or overwrite a branch containing unrelated user
   changes. If the worktree contains unrelated changes, report the exact blocker.
3. Create the suggested branch with `git switch -c <branch>` only after the
   implementation request is explicit. Never stage, commit, push, reset, clean,
   or delete branches as part of this skill.

### 4 — Implement and update documentation

Implement only the approved story scope and follow the plan. Preserve the
repository's architecture, ownership boundaries, security controls, data
integrity rules, and established patterns. Add tests for changed behavior.

Update documentation as part of the story only when the change warrants it and
repository evidence supports it:

- update the story's acceptance/evidence and status;
- update backlog, roadmap, or traceability documents when delivery status or
  mappings changed;
- update technical or user documentation when behavior or commands changed;
- preserve the distinction between planned, implemented, and verified behavior.

Use an available plan-review skill before implementation when the plan is
non-trivial or crosses components. Use an available branch-diff review skill
before declaring implementation complete. Treat blocking findings as gates:
fix them or stop with the findings documented.

### 5 — Verify and hand off

Run the narrowest relevant checks first, then the repository checks needed by
the affected layers. Discover commands from repository manifests and
documentation; do not claim irrelevant or unrun checks as evidence. Include
the exact commands and outcomes.

The final report must state:

- the plan file or backlog story changed;
- the branch created, or the suggested branch in Plan mode;
- implementation and documentation changes;
- verification commands and pass/fail results;
- unresolved risks, deferred scope, and any source conflicts.

Never claim completion from inspection alone. Successful command output is the
evidence for verification, and an unverified acceptance criterion remains open.
