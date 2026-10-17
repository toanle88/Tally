---
name: commit-and-create-pr
description: Commit explicitly authorized current changes and create and verify a GitHub pull request. Do not use for review-only, commit-only, or implementation work.
metadata:
  workflow: commit-and-create-pr
---

# Commit and Create Pull Request

Use this skill only when the user explicitly requests both a commit and a pull
request. This skill performs the GitHub workflow only: it does not implement,
repair, refactor, review, or test the code.

## Before changing history

- Read repository instructions that affect commits, branches, or pull requests,
  such as `AGENTS.md`, `CONTRIBUTING.md`, and PR templates.
- Inspect the worktree, branch, remote, and configured default branch:

  ```text
  git status --short
  git branch --show-current
  git symbolic-ref --quiet refs/remotes/origin/HEAD
  git remote -v
  ```

- Do not operate from a detached HEAD, the default branch, or a protected
  branch.
- Use the current worktree changes when the user has not specified paths. If
  clearly unrelated changes make the scope ambiguous, list them and ask which
  files belong in the commit.
- Stage only the explicit intended paths. Never use `git add .`, `git add -A`,
  `git commit -a`, reset, clean, restore, or force-push.
- Do not block this workflow because tests or code review were not run or
  because the implementation may need follow-up. Do not make those changes.

## Commit

- Derive a concise conventional commit subject from the actual changes and
  repository conventions, unless the user supplied a message.
- Inspect the staged diff and stat, then commit the intended paths:

  ```text
  git add -- <explicit paths>
  git diff --cached --check
  git diff --cached --stat
  git commit -m "<subject>"
  ```

- If staging reveals unrelated files, unstage only those files and stop if the
  intended scope cannot be separated safely.

## Push and create the pull request

- Confirm GitHub CLI authentication and repository detection with `gh auth
  status -h github.com` and `gh repo view`. Run both commands in the same
  process environment that will perform the GitHub operations.
- Prefer an already-exported `GH_TOKEN` or `GITHUB_TOKEN` when the environment
  provides one. Check only whether the variable is present; never print its
  value, put it in a command argument or remote URL, or log it. A token set in
  `~/.bashrc` may be absent from a login or non-interactive shell when the
  startup chain does not source that file. Use the authenticated shell context
  or ask the user to export the token in the command environment; do not
  silently edit shell startup files.
- Distinguish the environment token from a stored credential in
  `~/.config/gh/hosts.yml`: `gh auth status` may report both. If the active
  `GH_TOKEN`/`GITHUB_TOKEN` is absent or GitHub rejects it, stop and report the
  exact condition. Do not launch `gh auth login`/`gh auth refresh` or replace
  credentials as an implicit part of this workflow.
- Push the current branch without force:

  ```text
  git push -u origin HEAD
  ```

- Resolve the pull-request base from the configured default branch. Check for
  an existing open pull request for the branch before creating one.
- Create a non-draft pull request unless the user, branch convention, or
  repository instructions request a draft. Include a concise summary and
  state that tests/review were not run by this workflow when applicable.
- Verify the result:

  ```text
  gh pr view --json title,url,state,headRefName,baseRefName
  ```

Never claim a commit, push, or pull request exists until its command succeeds;
never claim the pull request exists until `gh pr view` confirms it.

## Stop conditions

Stop and report the exact condition when authorization, scope, branch validity,
staging, commit, push, GitHub authentication, pull-request creation, or
verification fails. This includes a missing credential in the command
environment and an invalid `GH_TOKEN`/`GITHUB_TOKEN`; do not expose the token
while diagnosing either condition. Preserve the worktree. A code-review
finding or missing test is not a stop condition for this skill unless the user
or repository instructions explicitly make it one.
