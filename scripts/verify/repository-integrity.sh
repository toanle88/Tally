#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
workflow="${root}/.github/workflows/pull-request-quality.yml"

fail() {
	printf 'Repository integrity verification failed: %s\n' "$1" >&2
	exit 1
}

check_forbidden_path() {
	local path="$1"

	case "${path}" in
		.env|.env.*|*/.env|*/.env.*)
			[[ "${path}" == ".env.example" || "${path}" == */.env.example ]] || return 1
			;;
		*.tfstate|*.tfstate.*|*.tfplan|*.tfvars|*.tfvars.json|*/crash.log|*/crash.*.log)
			return 1
			;;
		*/.terraform/*|node_modules/*|*/node_modules/*|web/dist/*|docs-site/.vitepress/dist/*|coverage/*|*/coverage/*|test-results/*|*/test-results/*|playwright-report/*|*/playwright-report/*|.pnpm-store/*|web/.pnpm-store/*)
			return 1
			;;
	esac

	return 0
}

check_repository_paths() {
	local path

	while IFS= read -r path; do
		[[ -n "${path}" ]] || continue
		if ! check_forbidden_path "${path}"; then
			fail "forbidden repository artifact: ${path}"
		fi
	done < <(git -C "${root}" ls-files --cached --others --exclude-standard)
}

check_workflow_safety() {
	[[ -s "${workflow}" ]] || fail "aggregate pull-request workflow is missing"

	if rg -n -i 'pull_request_target|id-token:[[:space:]]*write|azure/login|AZURE_(CLIENT|TENANT|SUBSCRIPTION)|ARM_CLIENT_SECRET|terraform[[:space:]]+(apply|destroy)|azure-learning-(apply|destroy|smoke)|terraform-(drift|cost)-check' "${workflow}"; then
		fail "aggregate pull-request workflow contains privileged or live-infrastructure behavior"
	fi

	if rg -n -i 'upload-artifact' "${workflow}"; then
		fail "aggregate pull-request workflow attempts to publish artifacts"
	fi
}

self_test() {
	local allowed forbidden

	for allowed in .env.example nested/.env.example fixture.tfvars.example; do
		if check_forbidden_path "${allowed}" 2>/dev/null; then
			:
		else
			fail "self-test rejected an allowed path: ${allowed}"
		fi
	done

	for forbidden in .env.local nested/.env.local fixture.tfstate fixture.tfplan fixture.tfvars node_modules/package.json; do
		if check_forbidden_path "${forbidden}"; then
			fail "self-test accepted a forbidden path: ${forbidden}"
		fi
	done

	printf 'Repository integrity negative checks passed.\n'
}

if [[ "${1-}" == "--self-test" ]]; then
	self_test
	exit 0
fi

check_repository_paths
check_workflow_safety
printf 'Repository integrity verification passed.\n'
