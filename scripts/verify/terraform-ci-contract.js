const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "../..");
const read = (file) => fs.readFileSync(path.join(root, file), "utf8");
const fail = (message) => { throw new Error(message); };

const bootstrap = read("infra/terraform/bootstrap/main.tf");
const terraformWorkflow = read(".github/workflows/terraform.yml");
const applyWorkflow = read(".github/workflows/terraform-apply.yml");
const applyScript = read("scripts/deploy/terraform-apply.sh");
const summary = read("scripts/verify/terraform-plan-summary.js");

for (const environment of ["dev", "demo", "prod-reference"]) {
  const subject = `repo:toanle88/Tally:environment:${environment}`;
  if (!bootstrap.includes(subject)) fail(`missing exact OIDC subject: ${subject}`);
}
if (bootstrap.includes("subject = \"*\"") || /client_secret\s*=|access_key\s*=/i.test(bootstrap)) {
  fail("bootstrap contains a wildcard subject or long-lived credential pattern");
}
if (!/make terraform-check[^\r\n]*terraform-environments-check/.test(terraformWorkflow)) fail("PR workflow must run Terraform plan tests");
if (/azure\/login|id-token:\s*write|pull_request_target|AZURE_CLIENT_SECRET|creds:/i.test(terraformWorkflow)) {
  fail("credential-free PR workflow contains Azure credentials or privileged OIDC access");
}
for (const required of [
  "workflow_dispatch",
  "if: github.ref == 'refs/heads/main'",
  "id-token: write",
  "azure/login@v3",
  "AZURE_CORE_OUTPUT: none",
  "terraform-apply.sh",
  "retention-days: 7",
  "environment: ${{ inputs.environment }}",
]) {
  if (!applyWorkflow.includes(required)) fail(`apply workflow missing ${required}`);
}
if (/pull_request_target|AZURE_CLIENT_SECRET|AZURE_CREDENTIALS|creds:/i.test(applyWorkflow)) {
  fail("apply workflow contains a forbidden long-lived credential pattern");
}
for (const environment of ["dev", "demo", "prod-reference"]) {
  if (!applyScript.includes(`\"${environment}\"`)) fail(`apply script does not allow ${environment}`);
}
if (!applyScript.includes("RUNNER_TEMP") || !applyScript.includes("show -json") || !/\bplan\b/.test(applyScript) || !/\bapply\b/.test(applyScript)) {
  fail("apply script must use temporary exact-plan execution");
}
if (!summary.includes("sensitive_values_included: false") || summary.includes("change.before") || summary.includes("change.after")) {
  fail("plan summary must exclude resource values");
}

console.log("Terraform CI/OIDC contract passed.");
