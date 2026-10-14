---
name: update-docs
description: Update the relevant TALLY documentation from current repository evidence while preserving the approved modular-monolith architecture and clearly separating implemented behavior from planned architecture.
metadata:
  project: tally
  workflow: documentation
---

# Update TALLY Documentation

## Purpose

Update the relevant TALLY documentation so it accurately represents:

1. What currently exists in the repository.
2. What commands have been verified.
3. The approved TALLY modular-monolith architecture.
4. The distinction between current implementation and planned target structure.
5. Consistent status and links across the affected documentation pages.

Select the smallest affected documentation set. This normally includes some
combination of:

- `README.md` for the repository overview, setup, architecture, and verified
  command snapshot.
- `docs-site/` pages for current status, architecture, development, operations,
  UX, or roadmap guidance.
- `ROADMAP.md` for epic and delivery status.
- `docs/backlog/` for story scope, acceptance evidence, and deferred work.
- `docs/verification/` for command-level or delivery-level evidence.

Do not update every document by default. Do not change approved specifications,
`AGENTS.md`, source code, generated build output, or unrelated documentation
unless the user explicitly requests it.

## Authority order

Use sources in this order:

1. `AGENTS.md` for stable TALLY architecture and engineering rules.
2. Relevant approved specifications for intended behavior and terminology.
3. Actual tracked repository files for current implementation.
4. Actual command output for build and test claims.
5. `ROADMAP.md` and backlog/verification records for delivery status.
6. Existing documentation as editable content.

When sources conflict, preserve repository reality for implementation claims,
preserve approved specifications for intended architecture and behavior, and
report the conflict. Never silently resolve a requirements conflict.

Never convert a roadmap item or planned architecture into an implementation claim.

## Required inspection

Before editing, read:

- `AGENTS.md`
- `README.md`
- `ROADMAP.md`
- The relevant approved specification files named by `AGENTS.md`.
- The relevant `docs-site/`, `docs/backlog/`, and `docs/verification/` files.
- `package.json`, `web/package.json`, `go.mod`, and `.gitignore` when the
  documentation includes commands, dependencies, or repository boundaries.

Inspect:

```text
git status --short
git ls-files
git diff -- <affected documentation files>
```

## Workflow

1. Identify the implementation or delivery area being documented and locate
   existing references to it with `rg`.
2. Read the affected documentation and the relevant repository/specification
   sources before editing.
3. Update only the affected documentation files, keeping current behavior,
   verified evidence, planned scope, and deferred qualification explicit.
4. Keep links, navigation labels, dates, command names, and status statements
   consistent across the changed documents.
5. Do not claim a test, build, migration, persistence check, hosted workflow,
   or qualification result without successful command output.

## Verification

- Run `git diff --check` for the changed documentation.
- Run `pnpm docs:check` when `docs-site/` content or its configuration changes.
- Use focused repository checks only when a documentation claim depends on
  current command output; do not run opt-in release or persistence gates merely
  because documentation was edited.
- Confirm the final diff contains documentation changes only and do not stage,
  commit, reset, clean, or revert repository files.
