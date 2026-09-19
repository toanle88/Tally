---
name: code-review-strict
description: Strictly review a branch or supplied code diff and return an evidence-based APPROVE or REJECT with a 0-100 score, code-quality assessment, issues, suggestions, and verification limits.
metadata:
  workflow: code-review
---

# Strict Code Review

## Purpose and boundary

Use this skill when the user asks for a strict code review, branch review, pull-request review, or approval decision. Review the requested scope; when no scope is supplied, review the complete current branch change set.

This skill is read-only. Do not edit, format, stage, commit, push, reset, restore, clean, stash, rebase, merge, switch branches, delete files, or change external systems. Running relevant tests and static checks is allowed when they are safe and available, but report exactly what was or was not run.

Do not reject unchanged, unrelated pre-existing code merely because it could be improved. Review pre-existing code only when the change depends on it, changes its contract, or makes its defect relevant to the requested scope.

## Review authority

Read repository-level instructions first, such as `AGENTS.md`, `CONTRIBUTING.md`, and project-specific instructions. Use sources in this order:

1. Approved requirements, acceptance criteria, architecture, design, security, and operational specifications.
2. The implementation, tests, manifests, migrations, generated artifacts, and configuration in the repository.
3. Successful command and test output from the current review.
4. README, roadmap, backlog, and status claims.

Inspect the relevant source documents before judging a change. Narrow the read to the sections and files that govern the changed capability. If sources conflict, report the conflict as an issue or open question. Never silently invent a resolution, requirement, identifier, API, schema, command, event, or architecture decision.

Apply the repository's own rules first. Also check generally applicable concerns: correctness, security, privacy, data integrity, compatibility, reliability, observability, accessibility where relevant, and maintainability. Apply domain-specific rules only when the project defines them; for example, use the project's own rules for money, regulated data, immutable records, or audit evidence rather than assuming a universal policy.

## Establish the review scope

Before judging code, record:

- `git status --short` and the current branch name.
- The comparison base. Prefer the repository's configured remote default branch and verify that it exists. If it cannot be resolved, use an available local base such as `main` only when appropriate, and state the assumption.
- Commits unique to the current branch, staged changes, unstaged changes, and relevant untracked files.
- Changed documentation, manifests, lockfiles, migrations, generated files, workflows, scripts, and configuration.
- Any user-supplied file, commit, pull request, or narrower review scope.

For an untracked file, inspect its complete contents; ordinary `git diff` does not show it. For a changed file, inspect both the diff and enough surrounding implementation to understand callers, ownership, lifecycle, error handling, and tests. Include deletions and renamed files in the review.

Keep the scope inventory separate from findings. A file being changed is not itself an issue.

## Review procedure

### 1. Understand the intended behavior

Trace each meaningful change to its requirement, acceptance criterion, design decision, invariant, or explicitly stated user goal. Check normal, boundary, invalid, duplicate, concurrent, partial-failure, retry, recovery, authorization, privacy, migration, compatibility, and operational paths when applicable.

### 2. Inspect implementation quality

Review the changed code for:

- Correct behavior and complete error handling.
- Clear ownership and dependency direction.
- Safe handling of inputs, outputs, dates, identifiers, state transitions, and domain-specific data.
- Authentication and authorization at the correct boundary, secret handling, sensitive-data exposure, and safe errors.
- Transaction boundaries, idempotency, retries, concurrency control, durable event delivery, and external-outcome reconciliation where applicable.
- Useful abstractions with no avoidable duplication, dead code, unreachable branches, brittle coupling, or unnecessary dependencies.
- Migration safety, generated-artifact freshness, API/schema compatibility, and repository hygiene.
- Documentation and status claims that match repository reality.

### 3. Verify proportionately

Run the smallest relevant safe checks, prioritizing existing project commands and verification commands named by the repository documentation. Examples include formatting/lint checks, unit or component tests, integration tests, contract checks, migration/drift checks, architecture checks, secret/dependency scans, and focused scripts. Do not turn a review into an unrelated full-suite exercise without a reason.

Record each check as `passed`, `failed`, `not run`, or `blocked`, including the command and meaningful output. A failed check is evidence, not a hypothesis. A check that was not run is not a pass.

### 4. Write findings

Report only actionable findings supported by evidence. Each issue must include:

- Severity: `BLOCKER`, `HIGH`, `MEDIUM`, or `LOW`.
- Location: absolute or repository-relative file path and line number when available.
- Evidence: the observed behavior, code, command output, or authoritative requirement.
- Impact: the concrete correctness, security, integrity, operational, maintenance, or delivery risk.
- Required action: the smallest change or evidence needed to resolve it.

Do not hide a finding in suggestions. Suggestions are non-blocking improvements only.

Use these severity meanings:

- `BLOCKER`: data loss or corruption, duplicate or missing critical effect, broken security boundary, secret exposure, destructive mutation of an established record, a failed build/test that prevents the changed scope from working, or a direct violation of an approved project control.
- `HIGH`: a material correctness, authorization, audit, retry, concurrency, recovery, privacy, compatibility, or release-gate gap that could affect users or important data but is not an immediate blocker in every execution path.
- `MEDIUM`: a meaningful defect or missing test/evidence with limited scope, or a maintainability/operational weakness that should be fixed before the next relevant increment.
- `LOW`: a localized improvement that does not materially threaten correctness, security, integrity, or delivery.

## Score and decision

Produce one integer score from 0 through 100. Score the changed scope, not the repository as a whole. Do not award points for unverified claims.

| Dimension | Maximum | What earns the points |
|---|---:|---|
| Correctness and behavior | 25 | Requirements and lifecycle behavior are correct across normal, edge, error, retry, concurrency, and recovery paths. |
| Architecture and design integrity | 20 | Ownership, boundaries, dependency direction, contracts, and intended scope are preserved. |
| Security, data integrity, and operational controls | 20 | Authentication, authorization, privacy, secrets, data integrity, audit, reliability, and domain-specific controls are correct where applicable. |
| Tests and verification evidence | 15 | Changed behavior has appropriate passing tests/checks and claims are backed by actual output. |
| Code quality and maintainability | 10 | Clear, focused, idiomatic code with justified abstractions, useful errors, and manageable complexity. |
| Scope, compatibility, and repository hygiene | 10 | No unapproved scope, unsafe dependency, drift, generated-file mismatch, secret, or misleading documentation claim. |

Use the following decision rule exactly:

- `APPROVE` when the final score is at least 80 and there is no unresolved `BLOCKER`.
- `REJECT` when the final score is below 80 or there is any unresolved `BLOCKER`.
- Never use `PASS`, `FAIL`, `CONDITIONAL APPROVE`, or a second overall status.
- If a `BLOCKER` exists, cap the final score at 79 even if the raw dimension total would be higher.
- If required verification is unavailable, score the evidence dimension according to what is actually proven and state the limitation. Do not treat unavailable evidence as a pass.

The dimensions must sum to the displayed score after any blocker cap. Keep the score defensible: a material issue must reduce the affected dimension, and do not double-count the same defect across dimensions. A review can be approved with low-severity suggestions, but an approval must still disclose all unresolved issues.

## Required response format

Return this structure and keep the labels exact:

```text
Status: APPROVE | REJECT
Score: <0-100>/100

Summary:
<one concise decision paragraph>

Code quality:
- Correctness and behavior: <points>/25 — <evidence-based assessment>
- Architecture and design integrity: <points>/20 — <evidence-based assessment>
- Security, data integrity, and operational controls: <points>/20 — <evidence-based assessment>
- Tests and verification evidence: <points>/15 — <evidence-based assessment>
- Code quality and maintainability: <points>/10 — <evidence-based assessment>
- Scope, compatibility, and repository hygiene: <points>/10 — <evidence-based assessment>

Issues:
- [SEVERITY] <title>
  Location: <file:line or scope>
  Evidence: <specific evidence>
  Impact: <concrete impact>
  Required action: <smallest required fix or evidence>

Suggestions:
- <non-blocking improvement>
  Rationale: <why it helps>

Verification:
- <command>: <passed|failed|not run|blocked> — <brief result>

Open questions or limitations:
- <only unresolved item affecting confidence>
```

Use `Issues: None`, `Suggestions: None`, and `Open questions or limitations: None` when applicable. If the status is `REJECT`, the summary must state the blocking conditions. If the status is `APPROVE`, the summary must state why the score clears 80 and disclose any remaining low/medium issues.

The final response must not claim that a command passed when it was not run, must not present suggestions as resolved issues, and must not omit a relevant verification limitation.
