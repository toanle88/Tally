---
name: review-current-branch-diff
description: Review the complete current TALLY branch diff without editing files, focusing on correctness, architecture boundaries, financial integrity, security, tests, repository hygiene, and unsupported claims.
compatibility: opencode
metadata:
  project: tally
  workflow: code-review
---

# Review Current TALLY Branch Diff

## Purpose

Review every relevant change on the current Git branch without modifying the
repository.

The review must cover:

- Commits made on the current branch
- Staged changes
- Unstaged changes
- Relevant untracked files
- Documentation and dependency changes
- Verification evidence

Do not edit, format, stage, commit, reset, restore, clean, stash, rebase, merge,
switch branches, or delete files.

## Project authority

Read `AGENTS.md` before reviewing.

When present, also inspect:

- `README.md`
- `ROADMAP.md`
- Root `package.json`
- `web/package.json`
- `go.mod`
- `.gitignore`
- Relevant TALLY specifications referenced by the changed files

Use this authority order:

1. Approved TALLY architecture and domain specifications
2. Repository implementation and manifests
3. Actual test and command evidence
4. README and roadmap status claims

When sources conflict, report the conflict. Do not invent a resolution.

## TALLY rules

Flag any change that violates these rules:

- Preserve bounded-context and module boundaries.
- Do not use binary floating point for money.
- Posted or otherwise established financial records are immutable.
- Corrections use reversal, adjustment, amendment, return, unapplication,
  replacement, or compensation.
- Retriable state-changing commands require idempotency.
- Integration events use the transactional outbox.
- Consumers use durable inbox identities where applicable.
- State changes require authorization and audit evidence.
- One module must not access another module's PostgreSQL adapter or owned
  schema directly.
- Domain and application packages must not import adapters.
- Shared technical packages must not contain capability-specific finance rules.
- Cross-module dependency cycles are prohibited.
- Avoid unnecessary abstractions and new dependencies.
- Do not invent requirements, identifiers, APIs, schemas, commands, events, or
  architecture decisions.
- Never claim code works without command or test evidence.

## Determine the comparison base

First inspect:

```text
git status --short
git branch --show-current
git remote -v
git symbolic-ref --quiet refs/remotes/origin/HEAD