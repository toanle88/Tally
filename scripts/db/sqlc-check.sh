#!/usr/bin/env bash
set -euo pipefail

generated_dirs=(
  "internal/platform/database/platformdb"
  "internal/identity/identitydb"
  "internal/organization/organizationdb"
  "internal/coa/coadb"
)

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

if [[ ! -f scripts/tools/sqlc.sh ]]; then
  fail "scripts/tools/sqlc.sh is missing"
fi

for generated_dir in "${generated_dirs[@]}"; do
  if [[ ! -d "${generated_dir}" ]]; then
    fail "generated directory is missing: ${generated_dir}; run make db-sqlc-generate and commit the output first"
  fi

  if [[ -z "$(git ls-files -- "${generated_dir}")" ]]; then
    fail "generated output is not committed: ${generated_dir}"
  fi

  # Check before generation so a manual edit cannot be overwritten and hidden.
  if ! git diff --quiet -- "${generated_dir}"; then
    fail "generated output has unstaged changes: ${generated_dir}"
  fi

  if ! git diff --cached --quiet -- "${generated_dir}"; then
    fail "generated output has staged changes: ${generated_dir}"
  fi

  if [[ -n "$(git ls-files --others --exclude-standard -- "${generated_dir}")" ]]; then
    fail "generated output contains untracked files: ${generated_dir}"
  fi
done

bash ./scripts/tools/sqlc.sh generate -f sqlc.yaml

# Source/schema drift is detected when regeneration changes committed output.
for generated_dir in "${generated_dirs[@]}"; do
  if ! git diff --quiet -- "${generated_dir}"; then
    git --no-pager diff -- "${generated_dir}" >&2
    fail "generated output is stale: ${generated_dir}; run make db-sqlc-generate and commit the result"
  fi

  if ! git diff --cached --quiet -- "${generated_dir}"; then
    fail "generation left staged changes in generated output: ${generated_dir}"
  fi

  if [[ -n "$(git ls-files --others --exclude-standard -- "${generated_dir}")" ]]; then
    git ls-files --others --exclude-standard -- "${generated_dir}" >&2
    fail "generation created untracked output: ${generated_dir}"
  fi
done

go test ./...
