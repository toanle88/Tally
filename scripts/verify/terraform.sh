#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
terraform_bin="${TERRAFORM_BIN:-terraform}"
terraform_root="${root}/infra/terraform"

roots=(
	"${terraform_root}/bootstrap"
	"${terraform_root}/environments/dev"
	"${terraform_root}/environments/demo"
	"${terraform_root}/environments/prod-reference"
)

fail() {
	echo "terraform verification failed: $*" >&2
	exit 1
}

command -v "${terraform_bin}" >/dev/null 2>&1 || fail "Terraform is required"

terraform_version="$(${terraform_bin} version -json | sed -n 's/.*"terraform_version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')"
[[ -n "${terraform_version}" ]] || fail "could not determine Terraform version"
terraform_major="${terraform_version%%.*}"
terraform_minor="${terraform_version#*.}"
terraform_minor="${terraform_minor%%.*}"
(( terraform_major == 1 && terraform_minor >= 8 )) || fail "Terraform ${terraform_version} is outside >= 1.8, < 2.0"

test -d "${terraform_root}/modules" || fail "missing modules boundary"

require_contract() {
	local root_path="$1"
	local contract="${root_path}/versions.tf"
	test -f "${contract}" || fail "missing root contract: ${contract}"

	grep -Fq 'required_version = ">= 1.8, < 2.0"' "${contract}" || fail "invalid Terraform constraint: ${contract}"
	grep -Fq 'source  = "hashicorp/azurerm"' "${contract}" || fail "missing AzureRM source: ${contract}"
	grep -Fq 'version = "~> 4.0"' "${contract}" || fail "invalid AzureRM constraint: ${contract}"
	grep -Fq 'source  = "hashicorp/azuread"' "${contract}" || fail "missing AzureAD source: ${contract}"
	grep -Fq 'version = "~> 3.0"' "${contract}" || fail "invalid AzureAD constraint: ${contract}"
	grep -Fq 'source  = "hashicorp/random"' "${contract}" || fail "missing Random source: ${contract}"
	grep -Fq 'version = "~> 3.6"' "${contract}" || fail "invalid Random constraint: ${contract}"
	grep -Fq 'backend "azurerm" {}' "${contract}" || fail "missing AzureRM backend declaration: ${contract}"
}

check_lockfile() {
	local root_path="$1"
	local lockfile="${root_path}/.terraform.lock.hcl"
	test -s "${lockfile}" || fail "missing provider lockfile: ${lockfile}"

	for provider in azurerm azuread random; do
		grep -Fq "provider \"registry.terraform.io/hashicorp/${provider}\"" "${lockfile}" || fail "missing ${provider} lock entry: ${lockfile}"
	done
	grep -Fq 'constraints = "~> 4.0"' "${lockfile}" || fail "missing AzureRM lock constraint: ${lockfile}"
	[[ "$(grep -Ec '^[[:space:]]*constraints[[:space:]]*=' "${lockfile}")" -eq 3 ]] || fail "lockfile does not constrain all providers: ${lockfile}"
	[[ "$(grep -Ec '^[[:space:]]*version[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"' "${lockfile}")" -eq 3 ]] || fail "lockfile does not resolve all providers: ${lockfile}"
	[[ "$(grep -Ec '^[[:space:]]*hashes[[:space:]]*=[[:space:]]*\[' "${lockfile}")" -eq 3 ]] || fail "lockfile does not contain checksums for all providers: ${lockfile}"
}

echo "== Terraform root contract verification =="
for root_path in "${roots[@]}"; do
	test -d "${root_path}" || fail "missing Terraform root: ${root_path}"
	require_contract "${root_path}"

	echo "-- ${root_path#${root}/}"
	"${terraform_bin}" -chdir="${root_path}" fmt -check
	"${terraform_bin}" -chdir="${root_path}" init -backend=false -input=false -lockfile=readonly -no-color
	"${terraform_bin}" -chdir="${root_path}" validate -no-color
	check_lockfile "${root_path}"
done

echo "== Boundary and artifact verification =="
source_files="$(rg --files --hidden -g '!.terraform/**' "${terraform_root}" || true)"
while IFS= read -r file; do
	[[ -z "${file}" ]] && continue
	case "${file}" in
		*.tf|*.terraform.lock.hcl|*.gitkeep) ;;
		*) fail "unexpected infrastructure artifact: ${file}" ;;
	esac
done <<<"${source_files}"

if rg -n '^[[:space:]]*(resource|data|module)[[:space:]]+"' "${terraform_root}" --glob '*.tf' --glob '!.terraform/**'; then
	fail "User Story 1 must not declare Terraform resources, data sources, or modules"
fi

if rg -n -i '(client_secret|access_key|secret_value|password)[[:space:]]*=[[:space:]]*"[^$]' "${terraform_root}" --glob '*.tf' --glob '!.terraform/**'; then
	fail "literal credential material found in Terraform source"
fi

files="$(git ls-files --cached --others --exclude-standard -- "${terraform_root}" || true)"
while IFS= read -r file; do
	[[ -z "${file}" ]] && continue
	case "${file}" in
		*.tfstate|*.tfstate.*|*.tfplan|*.tfvars|*.tfvars.json|*/crash.log|*/crash.*.log)
			fail "forbidden tracked or unignored Terraform artifact: ${file}"
			;;
		*/internal/*|*/cmd/*|*/web/*)
			fail "application/domain path crossed into infrastructure: ${file}"
			;;
	esac
done <<<"${files}"

echo "Terraform repository boundary verification passed"
