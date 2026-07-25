---
description: Update README.md from current repository evidence and TALLY architecture
agent: build
---

Load and follow the `update-readme` skill using the skill tool before doing
anything else.

Update `README.md` using the current repository as the source of truth.

Requirements:

- Read `AGENTS.md` and the `update-readme` skill first.
- Inspect the existing `README.md`.
- Inspect `ROADMAP.md`, but do not modify it.
- Inspect the root `package.json`, `web/package.json`, `go.mod`, `.gitignore`,
  `go.sum`, and `web/pnpm-lock.yaml` when present.
- Inspect the tracked repository structure with `git ls-files`.
- Inspect `git status --short` before editing.
- Preserve unrelated working-tree changes.
- Modify only `README.md`.
- Do not stage or commit anything.

README content must:

- Describe the current repository accurately.
- Explain that TALLY uses a modular-monolith architecture.
- Preserve bounded-context and module boundaries.
- Clearly separate the current repository structure from the planned target
  structure.
- Include only current paths proven by repository evidence in the current
  structure tree.
- Label future folders and functionality as planned.
- Document only commands that exist in the repository manifests.
- Use pnpm as the sole frontend package manager.
- Use exact versions from repository files when available.
- Avoid invented paths, commands, ports, endpoints, environment variables,
  APIs, schemas, events, identifiers, modules, or cloud resources.
- Avoid claiming tests, builds, or clean-clone verification pass without actual
  evidence.
- Avoid presenting PostgreSQL, migrations, OpenAPI, workers, authentication,
  authorization, observability, Terraform, Azure, CI, or finance modules as
  implemented unless repository evidence proves they exist.
- Keep the README practical and avoid duplicating the full architecture
  specification.

Preserve these TALLY rules:

- No binary floating point for money.
- Posted or established financial records are immutable.
- Corrections use reversal, adjustment, amendment, return, unapplication,
  replacement, or compensation.
- Retriable state-changing commands require idempotency.
- Integration events use the transactional outbox.
- State changes require authorization and audit evidence.
- Modules must not directly access another module's adapter or owned schema.
- Shared technical packages must not contain capability-specific finance rules.
- Avoid unnecessary abstractions and dependencies.

After editing, verify:

1. Every path shown in the current repository tree exists in `git ls-files`.
2. Every documented command exists in the relevant package manifest.
3. pnpm is the only documented frontend package manager.
4. Planned functionality is clearly labeled planned.
5. No success claim lacks test or command evidence.
6. No TALLY architecture rule has been weakened.
7. Only `README.md` was modified by this command.
8. The Markdown diff has no whitespace errors.

Finish by showing:

```text
git diff --check -- README.md
git diff -- README.md
git status --short