#!/usr/bin/env bash

set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${project_root}"

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

verify_migration_set() {
	local migration_name="$1"
	local migration_dir="$2"
	local version_table="$3"
	local status_output

	echo "Verifying ${migration_name} migrations..."

	status_output="$(
		go tool goose \
			-dir "${migration_dir}" \
			-table "${version_table}" \
			postgres "${DATABASE_URL}" \
			status 2>&1
	)"

	printf '%s\n' "${status_output}"

	if grep -qE '^[[:space:]]*Pending[[:space:]]' <<<"${status_output}"; then
		echo "${migration_name} migrations contain pending files." >&2
		exit 1
	fi
}

verify_seed_manifest() {
	local actual_checksum
	local expected_checksum
	local seed_file="db/seeds/local/v1.sql"

	echo "Verifying local seed manifest..."

	if [[ ! -f "${seed_file}" ]]; then
		echo "Seed file not found: ${seed_file}" >&2
		exit 1
	fi

	expected_checksum="$(sha256sum "${seed_file}" | awk '{print $1}')"

	actual_checksum="$(
		docker compose exec -T postgres \
			sh -c '
				psql \
					--username "$POSTGRES_USER" \
					--dbname "$POSTGRES_DB" \
					--tuples-only \
					--no-align \
					--set ON_ERROR_STOP=1 \
					--command "
						SELECT checksum
						FROM platform.local_seed_manifest
						WHERE seed_name = '\''tally-local-platform'\''
						  AND seed_version = 1;
					"
			'
	)"

	actual_checksum="$(tr -d '[:space:]' <<<"${actual_checksum}")"

	if [[ -z "${actual_checksum}" ]]; then
		echo "Seed verification failed: manifest entry is missing." >&2
		exit 1
	fi

	if [[ "${actual_checksum}" != "${expected_checksum}" ]]; then
		echo "Seed verification failed: checksum mismatch." >&2
		echo "Expected: ${expected_checksum}" >&2
		echo "Actual:   ${actual_checksum}" >&2
		exit 1
	fi

	echo "Local seed manifest is valid."
}

verify_migration_set \
	"bootstrap" \
	"db/migrations/bootstrap" \
	"goose_bootstrap_db_version"

verify_migration_set \
	"platform" \
	"db/migrations/platform" \
	"platform.goose_db_version"

verify_seed_manifest

echo "Database migration and seed verification passed."
