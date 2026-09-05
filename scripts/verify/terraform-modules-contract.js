const fs = require("fs");
const path = require("path");
const root = path.resolve(__dirname, "../../infra/terraform/modules");
const contract = JSON.parse(fs.readFileSync(path.join(__dirname, "terraform-modules-contract.json"), "utf8"));
const fail = (message) => { throw new Error(message); };
for (const name of contract.modules) {
  const dir = path.join(root, name);
  if (!fs.existsSync(dir)) fail(`missing module: ${name}`);
  for (const file of contract.required_files) if (!fs.existsSync(path.join(dir, file))) fail(`${name}: missing ${file}`);
  const main = fs.readFileSync(path.join(dir, "main.tf"), "utf8");
  const variables = fs.readFileSync(path.join(dir, "variables.tf"), "utf8");
  const versions = fs.readFileSync(path.join(dir, "versions.tf"), "utf8");
  if (/provider\s+"|backend\s+"|module\s+"/.test(main + versions)) fail(`${name}: modules may not own providers, backends, or child modules`);
  if (!["budget", "github-federation"].includes(name) && !variables.includes('variable "tags"')) fail(`${name}: tags input is required`);
  if (!["resource-group", "github-federation"].includes(name) && !variables.includes("validation")) fail(`${name}: boundary validations are required`);
  if (name === "container-registry" && !/admin_enabled\s*=\s*false/.test(main)) fail("ACR admin must be disabled");
  if (name === "container-app" && (!main.includes("@${var.image.digest}") || !main.includes("identity = var.identity_id"))) fail("Container App must use digest and managed identity");
  if (name === "container-app" && (!main.includes('dynamic "ingress"') || !main.includes('dynamic "secret"'))) fail("Container App ingress and Key Vault secret references are required");
  if (name === "container-app" && (!main.includes('dynamic "liveness_probe"') || !main.includes('dynamic "readiness_probe"') || !main.includes('port      = 8080'))) fail("Container App health probes must use the approved API port and paths");
  if (name === "container-app" && !main.includes('dynamic "authentication"')) fail("Container App backlog scaling authentication must be supported");
  if (name === "container-app" && !variables.includes('sensitive')) fail("Container App secret reference inputs must be marked sensitive");
  if (name === "postgresql" && (!/version\s*=\s*var\.postgres_version/.test(main) || !/value\s*=\s*"ON"/.test(main) || !/value\s*=\s*"TLSv1.2"/.test(main))) fail("PostgreSQL baseline is incomplete");
  if (name === "key-vault" && (!main.includes("default_action") || !main.includes("azurerm_role_assignment"))) fail("Key Vault network or role contract is incomplete");
  if (name === "postgresql" && (!main.includes('dynamic "high_availability"') || !main.includes("primary_availability_zone"))) fail("PostgreSQL HA zones must be wired");
  if (name === "postgresql" && !variables.includes('sensitive')) fail("PostgreSQL password input must be marked sensitive");
  if (name === "container-app-environment" && !variables.includes("network_mode")) fail("Container Apps network mode must be explicit");
  if (name === "github-federation" && (!main.includes("token.actions.githubusercontent.com") || main.includes("subject = " + "\"*\""))) fail("GitHub federation must use fixed issuer and non-wildcard subject");
  if (/password\s*=|secret_value\s*=|repository_token\s*=\s*"/.test(main)) fail(`${name}: secret values must not be defined in module code`);
}
for (const name of contract.test_modules) {
  if (!fs.existsSync(path.join(root, name, "tests", "contract.tftest.hcl"))) fail(`${name}: missing focused Terraform contract test`);
}
console.log(`Terraform module contract passed for ${contract.modules.length} modules.`);
