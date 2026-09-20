const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "../..");
const workflowDirectory = path.join(root, ".github/workflows");
const read = (relativePath) => fs.readFileSync(path.join(root, relativePath), "utf8");
const fail = (message) => { throw new Error(`CI contract verification failed: ${message}`); };
const requireText = (text, pattern, description) => {
  if (!pattern.test(text)) fail(`missing ${description}`);
};
const forbidText = (text, pattern, description) => {
  if (pattern.test(text)) fail(`forbidden ${description}`);
};

const aggregate = read(".github/workflows/pull-request-quality.yml");
const focusedWorkflows = ["openapi.yml", "persistence.yml", "terraform.yml", "docs-site.yml"];
const requiredJobs = {
  application: "Go and frontend quality",
  contracts: "OpenAPI and generated-artifact drift",
  persistence: "PostgreSQL and SQL drift",
  infrastructure: "Terraform credential-free plan and policy",
  security: "Security and repository integrity",
  documentation: "Documentation quality",
  required: "required",
};

requireText(aggregate, /^name: Pull-request quality$/m, "aggregate workflow name");
requireText(aggregate, /^  pull_request:\s*$/m, "unfiltered pull-request trigger");
requireText(aggregate, /^permissions:\n  contents: read\n  pull-requests: read$/m, "read-only workflow permissions");
requireText(aggregate, /^  required:\n(?:.*\n)*?    if: \$\{\{ always\(\) \}\}/m, "always-running aggregate job");
requireText(aggregate, /needs:\n(?:\s+- (?:application|contracts|persistence|infrastructure|security|documentation)\n){6}/m, "aggregate job dependencies");
requireText(aggregate, /APPLICATION_RESULT: \$\{\{ needs\.application\.result \}\}/, "application result fan-in check");
requireText(aggregate, /DOCUMENTATION_RESULT: \$\{\{ needs\.documentation\.result \}\}/, "documentation result fan-in check");
requireText(aggregate, /Pull-request quality \/ required/, "stable aggregate status-check name");
requireText(aggregate, /GITHUB_STEP_SUMMARY/, "safe aggregate summary");
requireText(aggregate, /Declared command\/tooling/, "aggregate command and tool evidence");
requireText(aggregate, /actions\/dependency-review-action@a1d282b36b6f3519aa1f3fc636f609c47dddb294/, "immutable Dependency Review action pin");
requireText(aggregate, /gitleaks\/gitleaks-action@e0c47f4f8be36e29cdc102c57e68cb5cbf0e8d1e/, "immutable Gitleaks action pin");
requireText(aggregate, /GITLEAKS_ENABLE_UPLOAD_ARTIFACT: "false"/, "Gitleaks artifact suppression");
requireText(aggregate, /GITLEAKS_ENABLE_COMMENTS: "false"/, "Gitleaks comment suppression");

const checkoutCount = (aggregate.match(/uses: actions\/checkout@/g) || []).length;
const nonPersistentCheckoutCount = (aggregate.match(/persist-credentials: false/g) || []).length;
if (checkoutCount !== nonPersistentCheckoutCount) fail("every aggregate checkout must disable credential persistence");

for (const [jobId, jobName] of Object.entries(requiredJobs)) {
  requireText(aggregate, new RegExp(`^  ${jobId}:\\n[\\s\\S]*?^    name: ${jobName.replace(/[.*+?^${}()|[\\]\\]/g, "\\$&")}$`, "m"), `${jobId} job name`);
}

for (const workflowName of focusedWorkflows) {
  const workflow = read(path.posix.join(".github/workflows", workflowName));
  forbidText(workflow, /^  pull_request:/m, `${workflowName} pull-request trigger`);
}

const terraformWorkflow = read(".github/workflows/terraform.yml");
requireText(terraformWorkflow, /^  push:\n    branches: \[main\]\n    paths:\n/m, "Terraform focused push path filter");

forbidText(aggregate, /pull_request_target|id-token:\s*write|azure\/login|AZURE_(CLIENT|TENANT|SUBSCRIPTION)|ARM_CLIENT_SECRET/i, "privileged pull-request credential pattern");
forbidText(aggregate, /terraform\s+(apply|destroy)|azure-learning-(apply|destroy|smoke)|terraform-(drift|cost)-check/i, "live infrastructure operation");
forbidText(aggregate, /upload-artifact/i, "artifact publication from the pull-request workflow");

const workflowFiles = fs.readdirSync(workflowDirectory).filter((file) => file.endsWith(".yml") || file.endsWith(".yaml"));
for (const file of workflowFiles) {
  const text = fs.readFileSync(path.join(workflowDirectory, file), "utf8");
  if (/\t/.test(text)) fail(`workflow contains a tab character: ${file}`);
}

console.log("CI contract verification passed.");
