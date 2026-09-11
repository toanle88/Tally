#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
source "${root}/scripts/verify/terraform-tools.env"

fail() {
  echo "Terraform tool verification failed: $*" >&2
  exit 1
}

require_command() {
  local name="$1"
  command -v "${!name}" >/dev/null 2>&1 || fail "${!name} is required"
}

TERRAFORM_BIN="${TERRAFORM_BIN:-terraform}"
TFLINT_BIN="${TFLINT_BIN:-tflint}"
CHECKOV_BIN="${CHECKOV_BIN:-checkov}"
INFRACOST_BIN="${INFRACOST_BIN:-infracost}"

require_command TERRAFORM_BIN
require_command TFLINT_BIN
require_command CHECKOV_BIN
require_command INFRACOST_BIN

terraform_actual="$("${TERRAFORM_BIN}" version -json | sed -n 's/.*"terraform_version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
tflint_actual="$("${TFLINT_BIN}" --version | tr '[:upper:]' '[:lower:]' | sed -n 's/.*tflint[^0-9]*\([0-9][0-9.]*\).*/\1/p' | head -1)"
checkov_actual="$("${CHECKOV_BIN}" --version 2>&1 | sed -n 's/.*\([0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\).*/\1/p' | head -1)"
infracost_actual="$("${INFRACOST_BIN}" --version 2>&1 | sed -n 's/.*\([0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\).*/\1/p' | head -1)"

[[ "${terraform_actual}" == "${TERRAFORM_VERSION}" ]] || fail "Terraform ${terraform_actual:-unknown}; expected ${TERRAFORM_VERSION}"
[[ "${tflint_actual}" == "${TFLINT_VERSION}" ]] || fail "TFLint ${tflint_actual:-unknown}; expected ${TFLINT_VERSION}"
[[ "${checkov_actual}" == "${CHECKOV_VERSION}" ]] || fail "Checkov ${checkov_actual:-unknown}; expected ${CHECKOV_VERSION}"
[[ "${infracost_actual}" == "${INFRACOST_VERSION}" ]] || fail "Infracost ${infracost_actual:-unknown}; expected ${INFRACOST_VERSION}"

echo "Terraform verification tools match ${root}/scripts/verify/terraform-tools.env"
