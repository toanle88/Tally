#!/usr/bin/env bash

set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${project_root}"

command="${1:-up}"

migration_dir_for_schema() {
	local schema="$1"

	case "${schema}" in
		bootstrap)
			printf '%s\n' "db/migrations/bootstrap"
			;;
		platform)
			printf '%s\n' "db/migrations/platform"
			;;
		*)
			echo "Unknown migration schema: ${schema}" >&2
			echo "Supported schemas: bootstrap, platform" >&2
			exit 2
			;;
	esac
}

create_migration() {
	local schema="${1:-}"
	local name="${2:-}"
	local migration_dir

	if [[ -z "${schema}" || -z "${name}" ]]; then
		echo "Usage: $0 create <bootstrap|platform> <migration_name>" >&2
		exit 2
	fi

	migration_dir="$(migration_dir_for_schema "${schema}")"

	go tool goose \
		-dir "${migration_dir}" \
		create "${name}" sql
}

check_migration_inventory() {
	local inventory_file="db/migrations/checksums.sha256"
	local generated_file

	if [[ ! -f "${inventory_file}" ]]; then
		echo "Migration checksum inventory is missing: ${inventory_file}" >&2
		echo "Create it with: make db-migrate-inventory" >&2
		exit 1
	fi

	generated_file="$(mktemp)"
	trap 'rm -f "${generated_file}"' RETURN

	find db/migrations \
		-type f \
		-name '*.sql' \
		-print0 |
		sort -z |
		xargs -0 sha256sum >"${generated_file}"

	if ! diff --unified "${inventory_file}" "${generated_file}"; then
		echo "Migration checksum inventory is out of date." >&2
		echo "Do not modify an existing migration after it is committed." >&2
		exit 1
	fi

	echo "Migration checksum inventory is valid."
}

validate_migrations() {
	echo "Validating db/migrations/bootstrap..."
	go tool goose \
		-dir db/migrations/bootstrap \
		validate

	echo "Validating db/migrations/platform..."
	go tool goose \
		-dir db/migrations/platform \
		validate

	echo "Migration validation passed."
}

# These commands are filesystem-only and require no database credentials.
case "${command}" in
	create)
		create_migration "${2:-}" "${3:-}"
		exit 0
		;;
	check)
		check_migration_inventory
		exit 0
		;;
	validate)
		validate_migrations
		exit 0
		;;
esac

# Only commands that connect to PostgreSQL load database configuration.
if [[ -f .env ]]; then
	set -a
	# shellcheck disable=SC1091
	source .env
	set +a
fi

: "${TALLY_DB_NAME:?TALLY_DB_NAME is required}"
: "${TALLY_DB_USER:?TALLY_DB_USER is required}"
: "${TALLY_DB_PASSWORD:?TALLY_DB_PASSWORD is required}"

TALLY_DB_PORT="${TALLY_DB_PORT:-5432}"

DATABASE_URL="$(
	printf 'postgres://%s:%s@localhost:%s/%s?sslmode=disable' \
		"${TALLY_DB_USER}" \
		"${TALLY_DB_PASSWORD}" \
		"${TALLY_DB_PORT}" \
		"${TALLY_DB_NAME}"
)"

run_goose() {
	local migration_dir="$1"
	local version_table="$2"
	local goose_command="$3"

	go tool goose \
		-dir "${migration_dir}" \
		-table "${version_table}" \
		postgres "${DATABASE_URL}" \
		"${goose_command}"
}

run_for_all_migration_sets() {
	local goose_command="$1"

	echo "Running ${goose_command} for db/migrations/bootstrap..."
	run_goose \
		"db/migrations/bootstrap" \
		"goose_bootstrap_db_version" \
		"${goose_command}"

	echo "Running ${goose_command} for db/migrations/platform..."
	run_goose \
		"db/migrations/platform" \
		"platform.goose_db_version" \
		"${goose_command}"
}

case "${command}" in
	up)
		run_for_all_migration_sets up
		echo "Database migrations completed successfully."
		;;
	status)
		run_for_all_migration_sets status
		;;
	*)
		echo "Unknown migration command: ${command}" >&2
		echo "Usage: $0 {up|status|create|validate|check}" >&2
		exit 2
		;;
esac
