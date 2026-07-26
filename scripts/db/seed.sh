#!/usr/bin/env bash
set -euo pipefail

readonly SEED_NAME="tally-local-platform"
readonly SEED_VERSION="1"
readonly SEED_FILE="db/seeds/local/v1.sql"

if [[ ! -f "${SEED_FILE}" ]]; then
  echo "Seed file not found: ${SEED_FILE}" >&2
  exit 1
fi

seed_checksum="$(sha256sum "${SEED_FILE}" | awk '{print $1}')"

docker compose exec -T postgres sh -c '
  psql \
    --username "$POSTGRES_USER" \
    --dbname "$POSTGRES_DB" \
    --set ON_ERROR_STOP=1
' <<SQL
DO \$\$
DECLARE
    existing_checksum text;
BEGIN
    SELECT checksum
      INTO existing_checksum
      FROM platform.local_seed_manifest
     WHERE seed_name = '${SEED_NAME}'
       AND seed_version = ${SEED_VERSION};

    IF existing_checksum IS NOT NULL
       AND existing_checksum <> '${seed_checksum}' THEN
        RAISE EXCEPTION
            'Seed % version % was changed after application',
            '${SEED_NAME}',
            ${SEED_VERSION};
    END IF;
END
\$\$;
SQL

docker compose exec -T postgres sh -c "
  psql \
    --username \"\$POSTGRES_USER\" \
    --dbname \"\$POSTGRES_DB\" \
    --set ON_ERROR_STOP=1 \
    -v seed_name='${SEED_NAME}' \
    -v seed_version='${SEED_VERSION}' \
    -v checksum='${seed_checksum}'
" < "${SEED_FILE}"

echo "Seed applied: ${SEED_NAME} v${SEED_VERSION}"
