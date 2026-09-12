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
  test_module_root="$test_root/$test_module"
  mkdir -p "$test_module_root"
  cp -R "$modules_root/$test_module/." "$test_module_root/"

  # Ignore only generated working data in the temporary copy. In particular,
  # do not copy an ignored .terraform directory or its provider symlinks.
  find "$test_module_root" -type d -name .terraform -prune -exec rm -rf -- {} +
  find "$test_module_root" -type l -name .terraform -delete
  find "$test_module_root" -type f \( -name '*.tfstate' -o -name '*.tfstate.*' -o -name '*.tfplan' -o -name '*.tfvars' -o -name '*.tfvars.json' \) -delete

  terraform -chdir="$test_module_root" init -backend=false -input=false -upgrade=false
  terraform -chdir="$test_module_root" validate
  terraform -chdir="$test_module_root" test
done
if find "$modules_root" -type f \( -name '*.tfstate' -o -name '*.tfstate.*' -o -name '*.tfplan' -o -name '*.tfvars' -o -name '*.tfvars.json' \) -print -quit | grep -q .; then
  echo "Terraform module artifacts must not be committed" >&2
  exit 1
fi
echo "Terraform module verification passed."
