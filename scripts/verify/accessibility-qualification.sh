#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
target="all"

usage() {
  cat <<'EOF'
Usage: pnpm test:a11y:qualification [-- --target TARGET]
       pnpm test:a11y:qualification -- --list-targets

Targets:
  shell     Routed application shell axe checks
  examples  Synthetic integrated examples axe checks
  keyboard  Synthetic keyboard and focus checks
  semantic  Synthetic semantic and screen-reader proxy checks
  visual    Visual adaptability and motion checks
  negative  Controlled known-violation proof
  all       All positive checks followed by the negative proof (default)
EOF
}

while (($# > 0)); do
  case "$1" in
    --)
      shift
      ;;
    --target)
      (($# >= 2)) || { echo "--target requires a value." >&2; usage >&2; exit 2; }
      target="$2"
      shift 2
      ;;
    --list-targets)
      usage
      exit 0
      ;;
    --help|-h)
      usage
      exit 0
      ;;
    *)
      echo "Unknown argument: $1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

grep_pattern=""
scope=""
declare -a evidence=()

case "$target" in
  shell)
    grep_pattern='axe: routed application shell'
    scope="Routed application shell at /"
    evidence=(
      "docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  examples)
    grep_pattern='axe: synthetic integrated examples'
    scope="Synthetic integrated examples at /development/examples"
    evidence=(
      "docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  keyboard)
    grep_pattern='keyboard: shared synthetic examples'
    scope="Keyboard and focus behavior for /development/examples"
    evidence=(
      "docs/verification/DLV-UX-002-us2-keyboard-focus-behavior.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  semantic)
    grep_pattern='semantic and screen-reader review proxy coverage'
    scope="Semantic and screen-reader proxy coverage for /development/examples"
    evidence=(
      "docs/verification/DLV-UX-002-us3-screen-reader-semantic-review.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  visual)
    grep_pattern='visual adaptability and motion preferences'
    scope="Visual adaptability and motion preferences for /development/examples"
    evidence=(
      "docs/verification/DLV-UX-002-us4-visual-adaptability-motion.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  negative)
    scope="Controlled unlabeled-button violation sentinel"
    evidence=(
      "docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  all)
    scope="All implemented accessibility checks and the controlled negative proof"
    evidence=(
      "docs/verification/DLV-UX-002-us1-automated-accessibility-harness.md"
      "docs/verification/DLV-UX-002-us2-keyboard-focus-behavior.md"
      "docs/verification/DLV-UX-002-us3-screen-reader-semantic-review.md"
      "docs/verification/DLV-UX-002-us4-visual-adaptability-motion.md"
      "docs/verification/DLV-UX-002-us5-accessibility-qualification-evidence.md"
    )
    ;;
  *)
    echo "Unknown accessibility qualification target: $target" >&2
    usage >&2
    exit 2
    ;;
esac

echo "Accessibility qualification target: $target"
echo "Affected scope: $scope"
echo "Affected evidence:"
for evidence_path in "${evidence[@]}"; do
  echo "  - $evidence_path"
done

if [[ "$target" == "negative" ]]; then
  echo "Rerun command: pnpm test:a11y:negative"
  pnpm --dir "$root" test:a11y:negative
  exit 0
fi

if [[ "$target" == "all" ]]; then
  echo "Rerun command: pnpm test:a11y"
  pnpm --dir "$root" test:a11y
  echo "Rerun command: pnpm test:a11y:negative"
  pnpm --dir "$root" test:a11y:negative
  exit 0
fi

echo "Rerun commands:"
echo "  pnpm --dir web exec tsc -p tsconfig.playwright.json --noEmit"
echo "  pnpm --dir web exec playwright test --config=playwright.config.ts --grep $grep_pattern"
pnpm --dir "$root/web" exec tsc -p tsconfig.playwright.json --noEmit
pnpm --dir "$root/web" exec playwright test --config=playwright.config.ts --grep "$grep_pattern"
