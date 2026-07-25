#!/usr/bin/env bash
set -euo pipefail

readonly ENV_FILE=".env"

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "Database verification failed: ${ENV_FILE} does not exist." >&2
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

readonly SEED_NAME="tally-local-platform"
readonly SEED_VERSION="1"
readonly SEED_FILE="db/seeds/local/v1.sql"

if [[ ! -f "${SEED_FILE}" ]]; then
  echo "Seed file not found: ${SEED_FILE}" >&2
  exit 1
fi

expected_checksum="$(sha256sum "${SEED_FILE}" | awk '{print $1}')"

actual_checksum="$(
  docker compose exec -T postgres sh -c '
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

if [[ -z "${actual_checksum}" ]]; then
  echo "Seed verification failed: manifest entry is missing" >&2
  exit 1
fi

if [[ "${actual_checksum}" != "${expected_checksum}" ]]; then
  echo "Seed verification failed: checksum mismatch" >&2
  echo "Expected: ${expected_checksum}" >&2
  echo "Actual:   ${actual_checksum}" >&2
  exit 1
fi

go tool goose \
  -dir db/migrations/bootstrap \
  -table "goose_bootstrap_db_version" \
  postgres \
  "${DATABASE_URL}" \
  status

go tool goose \
  -dir db/migrations/platform \
  -table "platform.goose_db_version" \
  postgres \
  "${DATABASE_URL}" \
  status

echo "Database migration and seed verification passed"