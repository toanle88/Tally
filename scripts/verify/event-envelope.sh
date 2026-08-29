#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "${root}"
export GOCACHE="${GOCACHE:-/tmp/tally-go-cache-dlv-plat-007}"

go test ./internal/platform/events ./internal/platform/idempotency
go test -race ./internal/platform/events
go vet ./internal/platform/events ./internal/platform/idempotency

echo "== Golden fixture verification =="
test -s internal/platform/events/testdata/envelope_vectors.json

echo "== Markdown fence verification =="
for file in docs/verification/DLV-PLAT-007-event-envelope.md docs/specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md docs/specs/system_design/03_data_integration_architecture_v1.0.md; do
  count="$(awk '/^```/{n++} END{print n+0}' "$file")"
  test "$((count % 2))" -eq 0 || { echo "unbalanced Markdown fences: $file" >&2; exit 1; }
done

echo "== Specification checkpoint verification =="
python3 - <<'PY'
from hashlib import sha256
from pathlib import Path
import re
paths = {
    Path("docs/specs/technical_specifications/04_events_workers_integration_specifications_v1.0.md"): "9d90cbc0bdf6d119c9b0ccb5db13271dfe78a9dacd4ebe0163e9360814adfe83",
    Path("docs/specs/system_design/03_data_integration_architecture_v1.0.md"): "e2580eb71d5da6564b3a9f2b9a557e2a40e1043851be6f269a573935e47fce73",
}
for path, expected in paths.items():
    text = path.read_text()
    actual = sha256(text.split("## Verification Checkpoint", 1)[0].encode()).hexdigest()
    recorded = re.search(r"Verified body SHA-256 \| `([^`]+)`", text).group(1)
    if actual != expected or recorded != actual:
        raise SystemExit(f"checkpoint mismatch: {path}")
PY

if rg -n 'internal/(ap|ar|audit|bankfeeds|coa|database|fixedassets|fiscalperiod|gl|httpapi|intercompany|invoicing|multicurrency|organization|payments|payroll|reporting|revenue|tax|workflow)' internal/platform/events --glob '*.go'; then
  echo "events must remain a platform technical package" >&2
  exit 1
fi

if ! git diff --quiet -- contracts/openapi internal/platform/httpapi/generated || ! git diff --cached --quiet -- contracts/openapi internal/platform/httpapi/generated; then
  echo "event envelope verification changed API artifacts" >&2
  exit 1
fi

git diff --check
echo "event envelope verification passed"
