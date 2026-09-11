#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
tflint_bin="${TFLINT_BIN:-tflint}"
command -v "${tflint_bin}" >/dev/null 2>&1 || { echo "TFLint is required; run terraform-tools-check first" >&2; exit 1; }

for terraform_root in bootstrap environments/dev environments/demo environments/prod-reference modules; do
  echo "== TFLint ${terraform_root} =="
  "${tflint_bin}" --chdir="${root}/infra/terraform/${terraform_root}" --config="${root}/.tflint.hcl"
done
