#!/usr/bin/env bash
set -euo pipefail

generated_dir="internal/platform/database/platformdb"

fail() {
  printf 'sqlc check failed: %s\n' "$1" >&2
  exit 1
}

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  fail "run this command from a Git working tree"
fi

repo_root="$(git rev-parse --show-toplevel)"
cd "${repo_root}"

if [[ ! -f sqlc.yaml ]]; then
  fail "sqlc.yaml is missing"
fi

if [[ ! -x scripts/tools/sqlc.sh ]]; then
  fail "scripts/tools/sqlc.sh is missing or is not executable"
fi

if [[ ! -d "${generated_dir}" ]]; then
  fail "generated directory is missing; run make db-sqlc-generate and commit the output first"
fi

if [[ -z "$(git ls-files -- "${generated_dir}")" ]]; then
  fail "generated output is not committed"
fi

# Check before generation so a manual edit cannot be overwritten and hidden.
if ! git diff --quiet -- "${generated_dir}"; then
  fail "generated output has unstaged changes"
fi

if ! git diff --cached --quiet -- "${generated_dir}"; then
  fail "generated output has staged changes"
fi

if [[ -n "$(git ls-files --others --exclude-standard -- "${generated_dir}")" ]]; then
  fail "generated output contains untracked files"
fi

./scripts/tools/sqlc.sh generate -f sqlc.yaml

# Source/schema drift is detected when regeneration changes committed output.
if ! git diff --quiet -- "${generated_dir}"; then
  git --no-pager diff -- "${generated_dir}" >&2
  fail "generated output is stale; run make db-sqlc-generate and commit the result"
fi

if ! git diff --cached --quiet -- "${generated_dir}"; then
  fail "generation left staged changes in generated output"
fi

if [[ -n "$(git ls-files --others --exclude-standard -- "${generated_dir}")" ]]; then
  git ls-files --others --exclude-standard -- "${generated_dir}" >&2
  fail "generation created untracked output"
fi

go test ./...
