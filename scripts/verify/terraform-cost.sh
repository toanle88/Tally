#!/usr/bin/env bash

set -Eeuo pipefail

plan_json="${PLAN_JSON:-}"
infracost_bin="${INFRACOST_BIN:-infracost}"
active_month_cost="${ACTIVE_MONTH_COST:-}"
active_month_limit="${ACTIVE_MONTH_COST_LIMIT:-50}"
[[ -n "${plan_json}" && -f "${plan_json}" ]] || { echo "PLAN_JSON must point to terraform show -json output" >&2; exit 1; }
[[ "${active_month_cost}" =~ ^[0-9]+([.][0-9]+)?$ ]] || { echo "ACTIVE_MONTH_COST must be the current active-month cost as a non-negative number" >&2; exit 1; }
[[ "${active_month_limit}" =~ ^[0-9]+([.][0-9]+)?$ ]] || { echo "ACTIVE_MONTH_COST_LIMIT must be a non-negative number" >&2; exit 1; }
command -v "${infracost_bin}" >/dev/null 2>&1 || { echo "Infracost is required; run terraform-tools-check first" >&2; exit 1; }

output_dir="$(mktemp -d "${TMPDIR:-/tmp}/tally-infracost.XXXXXX")"
trap 'rm -rf "${output_dir}"' EXIT
output_file="${output_dir}/breakdown.json"
"${infracost_bin}" breakdown --path "${plan_json}" --format json --out-file "${output_file}"

node - "${output_file}" "${active_month_cost}" "${active_month_limit}" <<'NODE'
const fs = require("fs");
const report = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
const current = Number(process.argv[3]);
const limit = Number(process.argv[4]);
const projectTotals = (report.projects ?? []).map((project) => Number(project.breakdown?.totalMonthlyCost)).filter(Number.isFinite);
const delta = projectTotals.length > 0 ? projectTotals.reduce((sum, value) => sum + value, 0) : Number(report.totalMonthlyCost ?? report.breakdown?.totalMonthlyCost ?? 0);
const activeMonthTotal = current + delta;
console.log(`Current active-month cost: USD ${current.toFixed(2)}`);
console.log(`Estimated plan delta: USD ${delta.toFixed(2)}`);
console.log(`Estimated active-month total: USD ${activeMonthTotal.toFixed(2)}`);
if (activeMonthTotal > limit && process.env.COST_APPROVAL !== "approved") {
  console.error(`Estimated active-month total exceeds USD ${limit.toFixed(2)}; set COST_APPROVAL=approved only after recorded review.`);
  process.exit(2);
}
NODE
