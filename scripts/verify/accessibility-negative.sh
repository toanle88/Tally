#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
temp_dir="$(mktemp -d)"
log_file="$temp_dir/accessibility-negative.log"
trap 'rm -r -- "$temp_dir"' EXIT

set +e
"$root/web/node_modules/.bin/playwright" test --config="$root/web/playwright.negative.config.ts" >"$log_file" 2>&1
status="$?"
set -e

cat "$log_file"

if [[ "$status" -eq 0 ]]; then
  echo 'Accessibility negative probe unexpectedly passed.' >&2
  exit 1
fi

rg -q 'A11Y_NEGATIVE_SENTINEL' "$log_file" || {
  echo 'Accessibility negative probe failed without the expected sentinel.' >&2
  exit 1
}

rg -q 'button-name' "$log_file" || {
  echo 'Accessibility negative probe failed without the expected button-name rule.' >&2
  exit 1
}

echo 'Accessibility negative probe verified: the known button-name violation failed the Playwright command.'
