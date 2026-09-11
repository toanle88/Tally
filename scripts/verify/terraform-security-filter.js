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

function filterReport(report, exceptions, environment, scanRoot, repositoryRoot) {
  if (!report || !report.results || !Array.isArray(report.results.failed_checks)) {
    throw new Error("Checkov report is missing results.failed_checks");
  }
  const failed = report.results.failed_checks;
  const remaining = failed.filter((check) => !exceptions.some((exception) => {
    const reportPath = relativeReportPath(check.file_path ?? "", scanRoot, repositoryRoot);
    return exception.environment === environment && exception.rule_id === check.check_id && exception.path === reportPath;
  }));
  return { failed, remaining };
}

function selfTest() {
  const report = { results: { failed_checks: [
    { check_id: "TEST_ALLOWED", file_path: "main.tf" },
    { check_id: "TEST_BLOCKED", file_path: "main.tf" },
  ] } };
  const result = filterReport(report, [{ environment: "dev", rule_id: "TEST_ALLOWED", path: "infra/terraform/dev/main.tf" }], "dev", "infra/terraform/dev", ".");
  if (result.remaining.length !== 1 || result.remaining[0].check_id !== "TEST_BLOCKED") {
    throw new Error("security exception filtering self-test failed");
  }
  const observedRuleIds = ["CKV_AZURE_237", "CKV_AZURE_167", "CKV_AZURE_166", "CKV_AZURE_163", "CKV_AZURE_233", "CKV_AZURE_164", "CKV_AZURE_139", "CKV_AZURE_165", "CKV_AZURE_110", "CKV_AZURE_42", "CKV_AZURE_136", "CKV2_AZURE_32", "CKV2_AZURE_57"];
  const observedReport = { results: { failed_checks: observedRuleIds.map((check_id) => ({ check_id, file_path: check_id.startsWith("CKV_AZURE_23") || check_id === "CKV_AZURE_237" || check_id === "CKV_AZURE_167" || check_id === "CKV_AZURE_166" || check_id === "CKV_AZURE_163" || check_id === "CKV_AZURE_164" || check_id === "CKV_AZURE_139" || check_id === "CKV_AZURE_165" ? "/../../modules/container-registry/main.tf" : check_id === "CKV2_AZURE_57" ? "/../../modules/postgresql/main.tf" : "/../../modules/key-vault/main.tf" })) } };
  const observedResult = filterReport(observedReport, loadManifest(path.join(__dirname, "terraform-policy-exceptions.json")), "dev", path.join(process.cwd(), "infra/terraform/environments/dev"), process.cwd());
  if (observedResult.remaining.length !== 0) throw new Error("observed Checkov exception filtering self-test failed");
  const duplicateRuleResult = filterReport({ results: { failed_checks: [{ check_id: "CKV_AZURE_136", file_path: "/../../modules/postgresql/main.tf" }] } }, loadManifest(path.join(__dirname, "terraform-policy-exceptions.json")), "dev", path.join(process.cwd(), "infra/terraform/environments/dev"), process.cwd());
  if (duplicateRuleResult.remaining.length !== 0) throw new Error("path-specific duplicate rule exception self-test failed");
  const demoRuleIds = ["CKV_AZURE_237", "CKV_AZURE_167", "CKV_AZURE_166", "CKV_AZURE_163", "CKV_AZURE_233", "CKV_AZURE_164", "CKV_AZURE_139", "CKV_AZURE_165", "CKV_AZURE_110", "CKV_AZURE_42", "CKV_AZURE_136", "CKV2_AZURE_32", "CKV2_AZURE_57"];
  const demoReport = { results: { failed_checks: demoRuleIds.map((check_id) => ({ check_id, file_path: check_id === "CKV_AZURE_136" || check_id === "CKV2_AZURE_57" ? "/../../modules/postgresql/main.tf" : check_id === "CKV_AZURE_237" || check_id === "CKV_AZURE_167" || check_id === "CKV_AZURE_166" || check_id === "CKV_AZURE_163" || check_id === "CKV_AZURE_233" || check_id === "CKV_AZURE_164" || check_id === "CKV_AZURE_139" || check_id === "CKV_AZURE_165" ? "/../../modules/container-registry/main.tf" : "/../../modules/key-vault/main.tf" })) } };
  const demoResult = filterReport(demoReport, loadManifest(path.join(__dirname, "terraform-policy-exceptions.json")), "demo", path.join(process.cwd(), "infra/terraform/environments/demo"), process.cwd());
  if (demoResult.remaining.length !== 0) throw new Error("demo Checkov exception filtering self-test failed");
  console.log("Terraform security exception filtering self-test passed.");
}

if (process.argv.includes("--self-test")) {
  selfTest();
  process.exit(0);
}

const reportFile = process.argv[process.argv.indexOf("--report") + 1];
const manifestFile = process.argv[process.argv.indexOf("--manifest") + 1];
const environment = process.argv[process.argv.indexOf("--environment") + 1];
const scanRoot = process.argv[process.argv.indexOf("--scan-root") + 1];
const repositoryRoot = process.argv[process.argv.indexOf("--repository-root") + 1];
if (![reportFile, manifestFile, environment, scanRoot, repositoryRoot].every(Boolean)) {
  throw new Error("usage: terraform-security-filter.js --report <file> --manifest <file> --environment <name> --scan-root <dir> --repository-root <dir>");
}

const report = JSON.parse(fs.readFileSync(reportFile, "utf8"));
const { failed, remaining } = filterReport(report, loadManifest(manifestFile), environment, scanRoot, repositoryRoot);
if (remaining.length > 0) {
  console.error(`Checkov reported ${remaining.length} unapproved failure(s) (${failed.length - remaining.length} approved exception(s)).`);
  for (const check of remaining) console.error(`${check.check_id}: ${check.file_path} ${check.resource ?? ""}`);
  process.exit(1);
}
console.log(`Checkov passed with ${failed.length} approved exception(s).`);
