#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
environment="${ENVIRONMENT:-}"
case "${environment}" in
  dev|demo|prod-reference) ;;
  *) echo "ENVIRONMENT must be dev, demo, or prod-reference" >&2; exit 1 ;;
esac

terraform_bin="${TERRAFORM_BIN:-terraform}"
terraform_root="${root}/infra/terraform/environments/${environment}"
plan_file="$(mktemp "${TMPDIR:-/tmp}/tally-${environment}-drift.XXXXXX.tfplan)"
trap 'rm -f "${plan_file}"' EXIT

"${terraform_bin}" -chdir="${terraform_root}" init -input=false
set +e
"${terraform_bin}" -chdir="${terraform_root}" plan -refresh-only -input=false -lock-timeout=5m -detailed-exitcode -out="${plan_file}" "$@"
status=$?
set -e
case "${status}" in
  0) echo "Terraform drift check passed: ${environment} is in sync." ;;
  2) echo "Terraform drift detected for ${environment}; review the refresh-only plan." >&2; exit 2 ;;
  *) echo "Terraform drift check failed for ${environment} with exit code ${status}." >&2; exit 1 ;;
esac
