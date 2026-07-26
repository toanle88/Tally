---
name: create-pr
description: Create a GitHub pull request for the current TALLY branch following project conventions. Validate branch naming, derive title from commits, build body with auto-linked issues, gate on review-current-branch-diff, then create and verify the PR.
compatibility: opencode
metadata:
  project: tally
  workflow: create-pr
---

# Create TALLY Pull Request

## Purpose

Create a GitHub pull request for the current branch that follows TALLY project
conventions. The skill validates the branch name, derives the title from commits,
builds the body with auto-linked issues, gates on the review skill, creates the
PR, and returns the URL.

Do not stage, commit, push, rewrite history, or delete branches.

## Project authority

Read `AGENTS.md` before proceeding.

When present, also inspect:

- `README.md`
- `ROADMAP.md`
- Root `package.json`
- `web/package.json`
- `go.mod`
- `.gitignore`

## Steps

### 1 — Validate the branch name

Inspect the current branch:

```text
git branch --show-current
```

The branch name must start with one of these prefixes:

- `feat/`
- `docs/`
- `fix/`
- `refactor/`

If it does not, report the branch name, explain that TALLY requires a standard
prefix, and refuse to create the PR.

Extract the delivery/epic code if present (e.g. `DLV-PLAT-002` from
`feat/dlv-plat-002-foo-bar`) for optional use in the PR body.

### 2 — Determine the comparison base

Run:

```text
git symbolic-ref --quiet refs/remotes/origin/HEAD
```

Expected output: `refs/remotes/origin/main`. Verify that the remote tracking
branch exists:

```text
git rev-parse --verify origin/main
```

If neither exists, use `main` as the base and report the assumption.

### 3 — Derive the PR title

Collect commits on the current branch that are not on the base:

```text
git log <base>..HEAD --reverse --format="%s"
```

Use the **first commit subject** as the default PR title. Log all commits for
manual review.

If the first commit subject starts with `feat:`, `docs:`, `fix:`, or `refactor:`,
preserve it as-is.

### 4 — Build the PR body

Collect the full commit log:

```text
git log <base>..HEAD --format="%s%n%b"
```

Build the body as a markdown unordered list of commit subjects on separate lines.

Then scan every commit subject and body for GitHub issue references (`#\d+`).
Collect unique issue numbers and append a links section at the end of the body:

```text
---

Closes #<N>
```

If multiple issues are referenced, list each on its own line.

If a commit body contains a longer description, include it as a sub-bullet under
that commit.

Also scan the branch name for an issue reference (`#\d+`) and add it if found.

### 5 — Gate on review skill

Load and run the `review-current-branch-diff` skill before creating the PR.

If the review skill reports violations that affect correctness or evidence
claims, flag them to the user and allow them to decide whether to proceed.

### 6 — Create the PR

Use `gh pr create` with the derived title and body:

```bash
gh pr create \
  --base <base> \
  --head "$(git branch --show-current)" \
  --title "<derived title>" \
  --body "<derived body>"
```

If the branch name contains `wip` or `draft` (case-insensitive), add
`--draft`.

### 7 — Verify

Confirm the PR was created:

```text
gh pr view --json title,url
```

Return the PR URL to the user.

## TALLY rules to enforce during review

Flag any PR that includes changes violating these rules. These are inherited
from the review skill and `AGENTS.md`:

- Preserve bounded-context and module boundaries.
- Do not use binary floating point for money.
- Posted or otherwise established financial records are immutable.
- Corrections use reversal, adjustment, amendment, return, unapplication,
  replacement, or compensation.
- Retriable state-changing commands require idempotency.
- Integration events use the transactional outbox.
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