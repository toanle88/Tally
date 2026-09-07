#!/usr/bin/env bash
set -euo pipefail
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
modules_root="$repo_root/infra/terraform/modules"
test_root="$(mktemp -d)"
trap 'rm -rf "$test_root"' EXIT
terraform version >/dev/null
node "$repo_root/scripts/verify/terraform-modules-contract.js"
terraform -chdir="$modules_root" fmt -check -recursive
for test_module in budget container-app postgresql; do
  cp -R "$modules_root/$test_module" "$test_root/$test_module"
  terraform -chdir="$test_root/$test_module" init -backend=false -input=false -upgrade=false
  terraform -chdir="$test_root/$test_module" validate
  terraform -chdir="$test_root/$test_module" test
done
if find "$modules_root" -type f \( -name '*.tfstate' -o -name '*.tfplan' -o -name '*.tfvars' \) -print -quit | grep -q .; then
  echo "Terraform module artifacts must not be committed" >&2
  exit 1
fi
echo "Terraform module verification passed."
