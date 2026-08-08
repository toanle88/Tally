---
name: plan-review
description: Review implementation and project plans for feasibility, completeness, alignment, risks, dependencies, and verification. Use when asked to review, approve, reject, or improve a plan, proposal, roadmap, design plan, migration plan, or execution outline. Return a clear PASS or REJECT verdict with prioritized issues and actionable suggestions.
---

# Plan Review

Review plans against the plan’s stated goal, scope, constraints, context, and acceptance criteria. Inspect relevant repository files, specifications, existing implementation, and project conventions when available. Do not invent requirements; identify missing information as an issue or assumption.

## Review procedure

1. Establish the review basis: goal, intended outcome, scope, constraints, assumptions, stakeholders, dependencies, and success criteria.
2. Trace the plan from objective to execution. Check that each step has an owner or responsible actor, inputs, outputs, sequencing, and a way to verify completion.
3. Check feasibility: technical approach, architecture boundaries, resources, timing, dependencies, rollout, rollback, and operational impact.
4. Check correctness and completeness: edge cases, failure handling, security, privacy, data integrity, compatibility, testing, observability, and documentation where relevant.
5. Separate blocking issues from non-blocking improvements. Treat unsupported claims, ambiguous scope, missing acceptance criteria, unsafe changes, and unverified dependencies as issues rather than silently assuming them away.
6. Decide the verdict using the rules below.

## Verdict rules

Return exactly one overall status:

- **PASS** — The plan is sufficiently clear, feasible, safe, and verifiable to proceed. Minor suggestions may remain, but none blocks execution.
- **REJECT** — One or more issues must be resolved before execution. Reject when the plan is materially incomplete, contradictory, infeasible, unsafe, out of scope, or not objectively verifiable.

Do not use “conditional pass” as the overall status. If conditions are required before execution, use **REJECT** and state the conditions.

## Required response format

```text
Status: PASS | REJECT

Summary:
<one concise paragraph explaining the decision>

Issues:
- [BLOCKER|HIGH|MEDIUM|LOW] <issue>
  Evidence: <plan detail or repository/source evidence>
  Impact: <why it matters>
  Required action: <what must change>

Suggestions:
- <optional improvement>
  Rationale: <why it improves the plan>

Open questions:
- <only unresolved questions that affect confidence or execution>
```

Use `Issues: None` and `Open questions: None` when applicable. Keep findings specific and actionable. Cite file paths, section names, identifiers, or line numbers when evidence is available. Distinguish facts from inferences and state assumptions explicitly.

## Review quality

Prefer a small number of well-supported findings over exhaustive speculation. Do not reject a plan merely because it could be more detailed; reject only when the missing detail can change feasibility, safety, scope, correctness, or verification. When the plan is underspecified, explain the smallest clarification or revision needed to make it reviewable.
