#!/usr/bin/env bash

set -Eeuo pipefail

readonly script_dir="$(
	cd -- "$(dirname -- "${BASH_SOURCE[0]}")" >/dev/null 2>&1
	pwd
)"
readonly repository_root="$(
	cd -- "${script_dir}/../.." >/dev/null 2>&1
	pwd
)"

cd "${repository_root}"

current_stage="initialization"
status_before_file="$(mktemp)"
status_after_file="$(mktemp)"

cleanup() {
	rm -f "${status_before_file}" "${status_after_file}"
}

report_failure() {
	local exit_code=$?

	printf '\nPersistence verification failed during stage: %s\n' \
		"${current_stage}" >&2

	exit "${exit_code}"
}

trap cleanup EXIT
trap report_failure ERR

run_stage() {
	local stage_name="$1"
	shift

	current_stage="${stage_name}"

	printf '\n==> %s\n' "${stage_name}"
	"$@"
}

require_file() {
	local path="$1"

	if [[ ! -f "${path}" ]]; then
		printf 'Required file does not exist: %s\n' "${path}" >&2
		return 1
	fi
}

require_command() {
	local command_name="$1"

	if ! command -v "${command_name}" >/dev/null 2>&1; then
		printf 'Required command is unavailable: %s\n' "${command_name}" >&2
		return 1
	fi
}

verify_repository() {
	local discovered_root

	discovered_root="$(git rev-parse --show-toplevel)"

	if [[ "${discovered_root}" != "${repository_root}" ]]; then
		printf 'Expected repository root %s, found %s\n' \
			"${repository_root}" \
			"${discovered_root}" >&2
		return 1
	fi

	require_file "go.mod"
	require_file "sqlc.yaml"
	require_file "Makefile"
}

verify_docker() {
	if ! docker info >/dev/null 2>&1; then
		printf '%s\n' \
			"Docker is required for the PostgreSQL 18 Testcontainers verification." \
			"Docker is installed but its daemon is unavailable." >&2
		return 1
	fi
}

capture_worktree_status() {
	local destination="$1"

	git status \
		--porcelain=v1 \
		--untracked-files=all \
		> "${destination}"
}

verify_worktree_unchanged() {
	capture_worktree_status "${status_after_file}"

	if cmp -s "${status_before_file}" "${status_after_file}"; then
		return 0
	fi

	printf '%s\n' \
		"Persistence verification changed the Git working tree." \
		"Generated files or another checked artifact are not reproducible." >&2

	diff \
		--unified \
		--label "working tree before persistence verification" \
		--label "working tree after persistence verification" \
		"${status_before_file}" \
		"${status_after_file}" >&2 || true

	return 1
}

run_stage \
	"verify repository prerequisites" \
	verify_repository

run_stage \
	"verify required commands" \
	require_command git

run_stage \
	"verify Go command" \
	require_command go

run_stage \
	"verify Make command" \
	require_command make

run_stage \
	"verify Docker command" \
	require_command docker

run_stage \
	"verify Docker daemon" \
	verify_docker

run_stage \
	"capture initial Git working-tree state" \
	capture_worktree_status "${status_before_file}"

run_stage \
	"verify pinned Goose and sqlc tools" \
	make --no-print-directory persistence-tools-version

run_stage \
	"validate Goose migration sets" \
	make --no-print-directory db-migrate-validate

run_stage \
	"verify migration inventory and checksums" \
	make --no-print-directory db-migrate-check

run_stage \
	"compile sqlc schema and query source" \
	make --no-print-directory sqlc-compile

run_stage \
	"detect stale or manually changed sqlc output" \
	make --no-print-directory sqlc-check

run_stage \
	"compile and test Go packages" \
	go test ./...

run_stage \
	"run clean PostgreSQL 18 persistence integration tests" \
	make --no-print-directory persistence-integration-test

run_stage \
	"verify persistence commands left the working tree unchanged" \
	verify_worktree_unchanged

printf '\nPersistence verification passed.\n'