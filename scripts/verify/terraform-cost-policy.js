#!/usr/bin/env node

const fs = require("fs");

function fail(message) {
  throw new Error(message);
}

function normalize(value) {
  let { coefficient, scale } = value;
  while (scale > 0 && coefficient % 10n === 0n) {
    coefficient /= 10n;
    scale -= 1;
  }
  return { coefficient, scale };
}

function parseDecimal(value, name, { signed = false } = {}) {
  if (typeof value !== "string") fail(`${name} must be a decimal text value`);
  const pattern = signed ? /^-?[0-9]+(?:\.[0-9]+)?$/ : /^[0-9]+(?:\.[0-9]+)?$/;
  if (!pattern.test(value)) fail(`${name} must be a ${signed ? "signed " : ""}decimal value`);
  const negative = value.startsWith("-");
  const unsigned = negative ? value.slice(1) : value;
  const [whole, fraction = ""] = unsigned.split(".");
  let coefficient = BigInt(`${whole}${fraction}` || "0");
  if (negative) coefficient = -coefficient;
  return normalize({ coefficient, scale: fraction.length });
}

function align(left, right) {
  const scale = Math.max(left.scale, right.scale);
  return {
    left: left.coefficient * 10n ** BigInt(scale - left.scale),
    right: right.coefficient * 10n ** BigInt(scale - right.scale),
    scale,
  };
}

function add(left, right) {
  const values = align(left, right);
  return normalize({ coefficient: values.left + values.right, scale: values.scale });
}

function compare(left, right) {
  const values = align(left, right);
  return values.left < values.right ? -1 : values.left > values.right ? 1 : 0;
}

function absolute(value) {
  return { coefficient: value.coefficient < 0n ? -value.coefficient : value.coefficient, scale: value.scale };
}

function format(value, places = 2) {
  const negative = value.coefficient < 0n;
  const magnitude = absolute(value);
  let coefficient = magnitude.coefficient;
  if (magnitude.scale > places) {
    const divisor = 10n ** BigInt(magnitude.scale - places);
    const remainder = coefficient % divisor;
    coefficient /= divisor;
    if (remainder * 2n >= divisor) coefficient += 1n;
  } else {
    coefficient *= 10n ** BigInt(places - magnitude.scale);
  }
  const digits = coefficient.toString().padStart(places + 1, "0");
  const result = `${digits.slice(0, -places)}.${digits.slice(-places)}`;
  return negative && coefficient !== 0n ? `-${result}` : result;
}

function getReportDelta(report) {
  if (Object.prototype.hasOwnProperty.call(report, "diffTotalMonthlyCost")) {
    return parseDecimal(report.diffTotalMonthlyCost, "diffTotalMonthlyCost", { signed: true });
  }

  if (!Array.isArray(report.projects) || report.projects.length === 0) {
    fail("Infracost report is missing diffTotalMonthlyCost and project diffs");
  }

  return report.projects.reduce((total, project, index) => {
    const value = project?.diff?.totalMonthlyCost;
    return add(total, parseDecimal(value, `projects[${index}].diff.totalMonthlyCost`, { signed: true }));
  }, parseDecimal("0", "project diff baseline", { signed: true }));
}

function evaluate(report, currentText, limitText, approval) {
  if (report?.currency !== "USD") fail("Infracost report currency must be USD");
  const current = parseDecimal(currentText, "ACTIVE_MONTH_COST");
  const limit = parseDecimal(limitText, "ACTIVE_MONTH_COST_LIMIT");
  const planned = parseDecimal(report.totalMonthlyCost, "totalMonthlyCost");
  const delta = getReportDelta(report);
  const activeMonthTotal = add(current, delta);
  if (activeMonthTotal.coefficient < 0n) fail("estimated active-month total cannot be negative");
  return { current, limit, planned, delta, activeMonthTotal, requiresApproval: compare(activeMonthTotal, limit) > 0 && approval !== "approved" };
}

function printResult(result) {
  console.log(`Current active-month cost: USD ${format(result.current)}`);
  console.log(`Estimated planned monthly cost: USD ${format(result.planned)}`);
  console.log(`Estimated plan delta: USD ${format(result.delta)}`);
  console.log(`Estimated active-month total: USD ${format(result.activeMonthTotal)}`);
  if (result.requiresApproval) {
    console.error(`Estimated active-month total exceeds USD ${format(result.limit)}; set COST_APPROVAL=approved only after recorded review.`);
    process.exitCode = 2;
  }
}

function assert(condition, message) {
  if (!condition) fail(message);
}

function selfTest() {
  const exact = evaluate({ currency: "USD", totalMonthlyCost: "100", diffTotalMonthlyCost: "0.001" }, "49.999", "50");
  assert(compare(exact.activeMonthTotal, parseDecimal("50", "expected")) === 0, "exact boundary arithmetic failed");
  assert(!exact.requiresApproval, "exact threshold incorrectly required approval");

  const over = evaluate({ currency: "USD", totalMonthlyCost: "100", diffTotalMonthlyCost: "0.002" }, "49.999", "50");
  assert(over.requiresApproval, "over-threshold arithmetic did not require approval");
  assert(!evaluate({ currency: "USD", totalMonthlyCost: "100", diffTotalMonthlyCost: "0.002" }, "49.999", "50", "approved").requiresApproval, "approved over-threshold cost failed");

  const fallback = evaluate({ currency: "USD", totalMonthlyCost: "12", projects: [{ diff: { totalMonthlyCost: "0.10" } }, { diff: { totalMonthlyCost: "0.20" } }] }, "1", "50");
  assert(compare(fallback.delta, parseDecimal("0.30", "expected")) === 0, "project diff fallback failed");

  let failed = false;
  try { evaluate({ currency: "USD", totalMonthlyCost: "100", diffTotalMonthlyCost: 1 }, "1", "50"); } catch (_) { failed = true; }
  assert(failed, "numeric monetary values were accepted through binary floating point");

  failed = false;
  try { evaluate({ currency: "USD", totalMonthlyCost: "100", diffTotalMonthlyCost: "1" }, "1", "50"); } catch (_) { failed = true; }
  assert(!failed, "valid cost report was rejected");
  console.log("Terraform cost-policy self-test passed.");
}

if (process.argv.includes("--self-test")) {
  selfTest();
} else {
  const report = JSON.parse(fs.readFileSync(process.argv[2], "utf8"));
  const result = evaluate(report, process.argv[3], process.argv[4], process.env.COST_APPROVAL);
  printResult(result);
}
