#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
work_dir="$(mktemp -d)"
trap 'rm -rf "${work_dir}"' EXIT

fail() {
  echo "API negative check failed: $1" >&2
  exit 1
}

expect_check_failure() {
  local name="$1"
  local candidate="$2"
  local output="${work_dir}/${name}.out"
  if bash "${root}/scripts/openapi/api-generate-check.sh" "${candidate}" >"${output}" 2>&1; then
    fail "${name} unexpectedly passed"
  fi
  [[ -s "${output}" ]] || fail "${name} produced no diagnostic output"
  echo "${name}: rejected as expected"
}

baseline="${work_dir}/baseline"
bash "${root}/scripts/openapi/go-generate.sh" "${baseline}" "${root}/contracts/openapi/openapi.yaml"

cp -a "${baseline}" "${work_dir}/manual"
echo '// intentional manual edit' >> "${work_dir}/manual/oas_server_gen.go"
expect_check_failure manual-edit "${work_dir}/manual"

cp -a "${baseline}" "${work_dir}/missing"
rm "${work_dir}/missing/oas_server_gen.go"
expect_check_failure missing-file "${work_dir}/missing"

cp -a "${baseline}" "${work_dir}/extra"
touch "${work_dir}/extra/unexpected.go"
expect_check_failure extra-file "${work_dir}/extra"

invalid="${work_dir}/invalid.yaml"
printf 'openapi: [\n' >"${invalid}"
if bash "${root}/scripts/openapi/go-generate.sh" "${work_dir}/invalid-output" "${invalid}" >"${work_dir}/invalid.out" 2>&1; then
  fail "invalid-input unexpectedly passed"
fi
[[ -s "${work_dir}/invalid.out" ]] || fail "invalid-input produced no diagnostic output"
echo 'invalid-input: rejected as expected (status=1)'

echo 'API negative checks passed.'
