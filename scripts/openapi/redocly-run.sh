#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
timeout_seconds="${OPENAPI_CHECK_TIMEOUT_SECONDS:-30}"

redocly="$root/node_modules/.bin/redocly"
if [[ ! -x "$redocly" ]]; then
  echo "Redocly CLI is not installed. Run pnpm install --frozen-lockfile first." >&2
  exit 1
fi

export CI=true
export REDOCLY_TELEMETRY=off
exec timeout --signal=TERM --kill-after=5s "${timeout_seconds}s" \
  "$redocly" "$@"
