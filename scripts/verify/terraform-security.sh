#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
checkov_bin="${CHECKOV_BIN:-checkov}"
command -v "${checkov_bin}" >/dev/null 2>&1 || { echo "Checkov is required; run terraform-tools-check first" >&2; exit 1; }
manifest="${root}/scripts/verify/terraform-policy-exceptions.json"
node "${root}/scripts/verify/terraform-policy-exceptions.js"
node "${root}/scripts/verify/terraform-security-filter.js" --self-test

for terraform_root in bootstrap environments/dev environments/demo environments/prod-reference; do
  echo "== Checkov ${terraform_root} =="
  report_dir="$(mktemp -d "${TMPDIR:-/tmp}/tally-checkov.XXXXXX")"
  trap 'rm -rf "${report_dir}"' EXIT
  report_file="${report_dir}/results_json.json"
  scan_root="${root}/infra/terraform/${terraform_root}"
  set +e
  "${checkov_bin}" --directory "${scan_root}" --framework terraform --config-file "${root}/.checkov.yaml" --output json --output-file-path "${report_dir}" --quiet
  checkov_status=$?
  set -e
  case "${checkov_status}" in
    0|1) ;;
    *) echo "Checkov failed operationally for ${terraform_root} with exit code ${checkov_status}" >&2; exit 1 ;;
  esac
  [[ -s "${report_file}" ]] || { echo "Checkov did not produce a report for ${terraform_root} (exit ${checkov_status})" >&2; exit 1; }
  node "${root}/scripts/verify/terraform-security-filter.js" \
    --report "${report_file}" \
    --manifest "${manifest}" \
    --scan-root "${scan_root}" \
    --repository-root "${root}"
done

echo "Terraform security scanning passed."
