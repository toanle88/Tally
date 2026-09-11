#!/usr/bin/env bash

set -Eeuo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
args=()
for argument in "$@"; do
  case "${argument}" in
    "${root}"/*) args+=("/workspace/${argument#"${root}"/}") ;;
    *) args+=("${argument}") ;;
  esac
done

exec docker compose run --rm --no-deps checkov "${args[@]}"
