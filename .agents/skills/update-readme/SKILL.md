---
name: update-readme
description: Update TALLY README.md from current repository evidence while preserving the approved modular-monolith architecture and clearly separating implemented behavior from planned architecture.
compatibility: opencode
metadata:
  project: tally
  workflow: documentation
---

# Update TALLY README

## Purpose

Update `README.md` so it accurately represents:

1. What currently exists in the repository.
2. What commands have been verified.
3. The approved TALLY modular-monolith architecture.
4. The distinction between current implementation and planned target structure.

Modify only `README.md`.

## Authority order

Use sources in this order:

1. `AGENTS.md` for stable TALLY architecture and engineering rules.
2. Actual tracked repository files for current implementation.
3. Actual command output for build and test claims.
4. `ROADMAP.md` for delivery status.
5. Existing `README.md` as editable documentation.

When sources conflict, preserve repository reality and report the conflict.

Never convert a roadmap item or planned architecture into an implementation claim.

## Required inspection

Before editing, read:

- `AGENTS.md`
- `README.md`
- `ROADMAP.md`
- `package.json`
- `web/package.json`
- `go.mod`
- `.gitignore`

Inspect:

```text
git status --short
git ls-files
git diff -- README.md