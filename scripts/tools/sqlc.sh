#!/usr/bin/env bash
set -euo pipefail

readonly SQLC_MODULE="github.com/sqlc-dev/sqlc/cmd/sqlc"
readonly SQLC_VERSION="v1.31.1"

exec go run "${SQLC_MODULE}@${SQLC_VERSION}" "$@"
