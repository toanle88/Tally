#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

fail() {
  echo "TypeScript API negative check failed: $1" >&2
  exit 1
}

expect_check_failure() {
  local name="$1"
  local candidate="$2"
  local output="${work_dir}/${name}.out"
  if bash "${root}/scripts/openapi/typescript-client-check.sh" "${candidate}" >"${output}" 2>&1; then
    fail "${name} unexpectedly passed"
  fi
  [[ -s "${output}" ]] || fail "${name} produced no diagnostic output"
  echo "${name}: rejected as expected"
}

baseline="${work_dir}/baseline"
bash "${root}/scripts/openapi/typescript-generate.sh" "${baseline}"

cp -a "${baseline}" "${work_dir}/manual"
echo '// intentional manual edit' >> "${work_dir}/manual/types.gen.ts"
expect_check_failure manual-edit "${work_dir}/manual"

cp -a "${baseline}" "${work_dir}/missing"
rm "${work_dir}/missing/types.gen.ts"
expect_check_failure missing-file "${work_dir}/missing"

cp -a "${baseline}" "${work_dir}/extra"
touch "${work_dir}/extra/unexpected.ts"
expect_check_failure extra-file "${work_dir}/extra"

bash "${root}/scripts/openapi/typescript-client-check.sh" "${baseline}" >/dev/null
echo 'regenerated-output: accepted as expected'
echo 'TypeScript API negative checks passed.'
