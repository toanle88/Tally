---
description: Review the complete current branch diff without modifying files
agent: plan
subtask: true
---

Load and follow the `review-current-branch-diff` skill before beginning.

Review the complete current branch diff.

Requirements:

- Do not modify, format, stage, commit, restore, reset, clean, stash, merge,
  rebase, switch branches, or delete files.
- Include committed branch changes, staged changes, unstaged changes, and
  relevant untracked files.
- Determine the comparison base safely from the remote default branch.
- Read `AGENTS.md` before reviewing.
- Apply all TALLY architecture, finance-integrity, security, testing, and
  repository-hygiene rules defined by the skill.
- Run only safe, relevant verification commands.
- Never claim a test or build passed unless it was executed successfully.
- Report findings in severity order with file and line references.
- Finish with a merge recommendation:
  - Ready
  - Ready after listed fixes
  - Not ready

Additional review focus supplied by the user:

$ARGUMENTS