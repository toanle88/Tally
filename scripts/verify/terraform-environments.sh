#!/usr/bin/env bash
set -Eeuo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"

command -v "${terraform_bin}" >/dev/null 2>&1 || {
  echo "Terraform is required" >&2
  exit 1
}

node "${repo_root}/scripts/verify/terraform-environments-contract.js"

for environment in dev demo prod-reference; do
  environment_root="${repo_root}/infra/terraform/environments/${environment}"
  "${terraform_bin}" -chdir="${environment_root}" fmt -check -no-color
  "${terraform_bin}" -chdir="${environment_root}" init -backend=false -input=false -lockfile=readonly -no-color
  "${terraform_bin}" -chdir="${environment_root}" validate -no-color
  "${terraform_bin}" -chdir="${environment_root}" test -no-color
done

echo "Terraform environment verification passed."
