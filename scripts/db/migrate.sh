#!/usr/bin/env bash
set -euo pipefail

readonly ENV_FILE=".env"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Database migration failed: ${ENV_FILE} does not exist." >&2
  echo "Copy .env.example to .env first." >&2
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${ENV_FILE}"
set +a

: "${TALLY_DB_NAME:?TALLY_DB_NAME is required in .env}"
: "${TALLY_DB_USER:?TALLY_DB_USER is required in .env}"
: "${TALLY_DB_PASSWORD:?TALLY_DB_PASSWORD is required in .env}"
: "${TALLY_DB_PORT:?TALLY_DB_PORT is required in .env}"

readonly DATABASE_URL="postgres://${TALLY_DB_USER}:${TALLY_DB_PASSWORD}@127.0.0.1:${TALLY_DB_PORT}/${TALLY_DB_NAME}?sslmode=disable"

run_migrations() {
  local migration_dir="$1"
  local history_table="$2"

  if [[ ! -d "${migration_dir}" ]]; then
    echo "Migration directory not found: ${migration_dir}" >&2
    exit 1
  fi

  echo "Applying migrations from ${migration_dir}..."

  go tool goose \
    -dir "${migration_dir}" \
    -table "${history_table}" \
    postgres \
    "${DATABASE_URL}" \
    up
}

run_migrations \
  "db/migrations/bootstrap" \
  "goose_bootstrap_db_version"

run_migrations \
  "db/migrations/platform" \
  "platform.goose_db_version"

echo "Database migrations completed successfully."