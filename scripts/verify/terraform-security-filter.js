#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

function loadManifest(file) {
  const document = JSON.parse(fs.readFileSync(file, "utf8"));
  return document.exceptions ?? [];
}

function relativeReportPath(filePath, scanRoot, repositoryRoot) {
  const normalized = filePath.replaceAll("\\", "/").replace(/^\/+/, "");
  return path.posix.relative(repositoryRoot, path.resolve(scanRoot, normalized)).replaceAll("\\", "/");
}

function filterReport(report, exceptions, scanRoot, repositoryRoot) {
  if (!report || !report.results || !Array.isArray(report.results.failed_checks)) {
    throw new Error("Checkov report is missing results.failed_checks");
  }
  const failed = report.results.failed_checks;
  const remaining = failed.filter((check) => !exceptions.some((exception) => {
    const reportPath = relativeReportPath(check.file_path ?? "", scanRoot, repositoryRoot);
    return exception.rule_id === check.check_id && exception.path === reportPath;
  }));
  return { failed, remaining };
}

function selfTest() {
  const report = { results: { failed_checks: [
    { check_id: "TEST_ALLOWED", file_path: "main.tf" },
    { check_id: "TEST_BLOCKED", file_path: "main.tf" },
  ] } };
  const result = filterReport(report, [{ rule_id: "TEST_ALLOWED", path: "infra/terraform/dev/main.tf" }], "infra/terraform/dev", ".");
  if (result.remaining.length !== 1 || result.remaining[0].check_id !== "TEST_BLOCKED") {
    throw new Error("security exception filtering self-test failed");
  }
  console.log("Terraform security exception filtering self-test passed.");
}

if (process.argv.includes("--self-test")) {
  selfTest();
  process.exit(0);
}

const reportFile = process.argv[process.argv.indexOf("--report") + 1];
const manifestFile = process.argv[process.argv.indexOf("--manifest") + 1];
const scanRoot = process.argv[process.argv.indexOf("--scan-root") + 1];
const repositoryRoot = process.argv[process.argv.indexOf("--repository-root") + 1];
if (![reportFile, manifestFile, scanRoot, repositoryRoot].every(Boolean)) {
  throw new Error("usage: terraform-security-filter.js --report <file> --manifest <file> --scan-root <dir> --repository-root <dir>");
}

const report = JSON.parse(fs.readFileSync(reportFile, "utf8"));
const { failed, remaining } = filterReport(report, loadManifest(manifestFile), scanRoot, repositoryRoot);
if (remaining.length > 0) {
  console.error(`Checkov reported ${remaining.length} unapproved failure(s) (${failed.length - remaining.length} approved exception(s)).`);
  for (const check of remaining) console.error(`${check.check_id}: ${check.file_path} ${check.resource ?? ""}`);
  process.exit(1);
}
console.log(`Checkov passed with ${failed.length} approved exception(s).`);
