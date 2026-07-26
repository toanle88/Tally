#!/usr/bin/env bash

set -Eeuo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${project_root}"

clean_database=false

usage() {
	cat <<'EOF'
Usage:
  ./scripts/verify/database.sh
  ./scripts/verify/database.sh --clean

Options:
  --clean  Delete and recreate the local PostgreSQL volume before verification.
  --help   Show this help message.

The default verification does not delete the database volume.
EOF
}

while (($# > 0)); do
	case "$1" in
		--clean)
			clean_database=true
			;;
		--help | -h)
			usage
			exit 0
			;;
		*)
			echo "Unknown argument: $1" >&2
			usage >&2
			exit 2
			;;
	esac

	shift
done

current_step="initialization"

on_error() {
	local exit_code=$?
	local line_number="${1:-unknown}"

	echo >&2
	echo "Database verification failed." >&2
	echo "Step: ${current_step}" >&2
	echo "Line: ${line_number}" >&2
	echo "Exit code: ${exit_code}" >&2

	exit "${exit_code}"
}

trap 'on_error "${LINENO}"' ERR

run() {
	local description="$1"
	shift

	current_step="${description}"

	echo
	echo "== ${description} =="
	"$@"
}

run "Validate migrate.sh syntax" \
	bash -n scripts/db/migrate.sh

run "Validate seed.sh syntax" \
	bash -n scripts/db/seed.sh

run "Validate verify.sh syntax" \
	bash -n scripts/db/verify.sh

run "Validate Docker Compose configuration" \
	make db-config

if [[ "${clean_database}" == true ]]; then
	echo
	echo "WARNING: --clean permanently deletes the TALLY local PostgreSQL volume."

	run "Recreate PostgreSQL from a clean volume" \
		make db-reset
else
	run "Start PostgreSQL" \
		make db-up

	run "Wait for PostgreSQL health" \
		make db-wait
fi

run "Apply database migrations" \
	make db-migrate

run "Reapply database migrations to prove idempotency" \
	make db-migrate

run "Show database migration status" \
	make db-migrate-status

run "Apply local database seed" \
	make db-seed

run "Reapply local database seed to prove idempotency" \
	make db-seed

run "Verify migration and seed state" \
	make db-verify

run "Validate migration files" \
	make db-migrate-validate

run "Verify migration checksum inventory" \
	make db-migrate-check

run "Run repository checks" \
	make check

current_step="complete"

echo
echo "Database verification passed."