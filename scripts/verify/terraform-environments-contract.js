const fs = require("fs");
const path = require("path");

const root = path.resolve(__dirname, "../../infra/terraform/environments");
const contract = JSON.parse(fs.readFileSync(path.join(__dirname, "terraform-environments-contract.json"), "utf8"));
const fail = (message) => { throw new Error(message); };

const source = (profile) => {
  const dir = path.join(root, profile);
  const files = fs.readdirSync(dir).filter((file) => file.endsWith(".tf"));
  return files.map((file) => fs.readFileSync(path.join(dir, file), "utf8")).join("\n");
};

for (const [profile, rules] of Object.entries(contract.profiles)) {
  const dir = path.join(root, profile);
  if (!fs.existsSync(dir)) fail(`missing environment root: ${profile}`);
  const text = source(profile);

  if (!text.includes(`profile = "${profile}"`)) fail(`${profile}: profile constant is missing`);
  const moduleNames = [...text.matchAll(/module\s+"([^"]+)"/g)].map((match) => match[1]).sort();
  const expectedModules = [...contract.required_modules].sort();
  if (JSON.stringify(moduleNames) !== JSON.stringify(expectedModules)) {
    fail(`${profile}: environment module inventory is not exact`);
  }
  const resourceNames = [...text.matchAll(/resource\s+"([^"]+)"\s+"([^"]+)"/g)].map((match) => `${match[1]}.${match[2]}`).sort();
  const expectedResources = profile === "prod-reference" ? ["azurerm_management_lock.environment"] : [];
  if (JSON.stringify(resourceNames) !== JSON.stringify(expectedResources)) {
    fail(`${profile}: direct resource inventory is outside the approved root boundary`);
  }
  if (/data\s+"[^"]+"\s+"[^"]+"/.test(text)) {
    fail(`${profile}: environment roots must not add data sources outside the approved composition`);
  }
  for (const moduleName of contract.required_modules) {
    if (!new RegExp(`module\\s+"${moduleName}"`).test(text)) fail(`${profile}: missing module ${moduleName}`);
  }
  for (const forbidden of ["budget", "github_federation"]) {
    if (new RegExp(`module\\s+"${forbidden}"`).test(text)) fail(`${profile}: future module ${forbidden} is out of scope`);
  }
  for (const outputName of contract.required_outputs) {
    if (!new RegExp(`output\\s+"${outputName}"`).test(text)) fail(`${profile}: missing output ${outputName}`);
  }

  if (!text.includes(`network_mode               = "${rules.network_mode}"`) && !text.includes(`network_mode             = "${rules.network_mode}"`)) {
    fail(`${profile}: invalid Container Apps network mode`);
  }
  if (!text.includes(`mode                = "${rules.postgres_mode}"`)) fail(`${profile}: invalid PostgreSQL network mode`);
  if (!text.includes(`data_classification = "${rules.data_classification}"`)) fail(`${profile}: invalid data classification`);
  if (!text.includes("repository          = null") && !text.includes("repository            = null")) fail(`${profile}: Static Web App repository must be null`);
  if (!text.includes("repository_token    = null")) fail(`${profile}: Static Web App repository token must be null`);
  if (!text.includes("worker_backlog_rule")) fail(`${profile}: worker backlog input is missing`);
  if (!text.includes("administrator_password_wo         = var.postgres_administrator_password_wo")) fail(`${profile}: PostgreSQL password must remain an external write-only input`);
  if (!text.includes("secret_refs  = []")) fail(`${profile}: profile must not invent application secret values`);
  if (profile !== "prod-reference") {
    if (!text.includes("start_ip_address == rule.end_ip_address")) fail(`${profile}: learning firewall rules are not single-host restricted`);
    if (!text.includes('rule.start_ip_address != "0.0.0.0"')) fail(`${profile}: learning firewall rules do not reject all-address input`);
    if ((text.match(/min_replicas = 0/g) || []).length < 2) fail(`${profile}: API and worker must support scale-to-zero`);
    if ((text.match(/max_replicas = 1/g) || []).length < 2) fail(`${profile}: API and worker maximum replicas must remain one`);
    if (!text.includes('sku_name                          = "B_Standard_B1ms"')) fail(`${profile}: learning PostgreSQL SKU must remain B_Standard_B1ms`);
    if (!text.includes("storage_mb                        = 32768")) fail(`${profile}: learning PostgreSQL storage must remain 32768 MB`);
    if (!text.includes("retention_days = 7")) fail(`${profile}: learning PostgreSQL backup retention must remain seven days`);
  } else {
    if (!text.includes("min_replicas = 2")) fail(`${profile}: API must have at least two replicas`);
    if (!text.includes("min_replicas = 1")) fail(`${profile}: worker must have at least one replica`);
    if (!text.includes("max_replicas = var.prod_api_max_replicas")) fail(`${profile}: API capacity must be externally measured`);
    if (!text.includes("max_replicas = var.prod_worker_max_replicas")) fail(`${profile}: worker capacity must be externally measured`);
    if (!text.includes("enabled                   = true")) fail(`${profile}: PostgreSQL HA must be enabled`);
    if (!text.includes("retention_days = 35")) fail(`${profile}: production PostgreSQL backup retention must be 35 days`);
  }

  if (rules.requires_expiry) {
    if (!text.includes('variable "demo_expires_on"')) fail(`${profile}: expiry input is missing`);
    if (!text.includes("expires_on          = var.demo_expires_on")) fail(`${profile}: expiry tag is missing`);
  } else if (text.includes('variable "demo_expires_on"')) {
    fail(`${profile}: expiry input is only allowed for demo`);
  }

  if (rules.requires_lock) {
    if (!text.includes('resource "azurerm_management_lock" "environment"')) fail(`${profile}: production deletion lock is missing`);
    if (!text.includes('lock_level = "CanNotDelete"')) fail(`${profile}: production deletion lock level is invalid`);
    if (!text.includes("prevent_destroy = true")) fail(`${profile}: production deletion lock must be non-destroyable`);
    if (!text.includes('firewall_rules      = toset([])')) fail(`${profile}: production PostgreSQL firewall must be empty`);
  } else if (text.includes('resource "azurerm_management_lock"')) {
    fail(`${profile}: deletion lock is only allowed for prod-reference`);
  }
}

console.log("Terraform environment contract passed for dev, demo, and prod-reference.");
