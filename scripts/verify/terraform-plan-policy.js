#!/usr/bin/env node

const fs = require("fs");
const path = require("path");

const requiredTags = ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"];
const taggableTypes = new Set([
  "azurerm_application_insights",
  "azurerm_container_app",
  "azurerm_container_app_environment",
  "azurerm_container_registry",
  "azurerm_key_vault",
  "azurerm_log_analytics_workspace",
  "azurerm_postgresql_flexible_server",
  "azurerm_resource_group",
  "azurerm_static_web_app",
  "azurerm_storage_account",
  "azurerm_user_assigned_identity",
]);

function fail(message) { throw new Error(message); }

function collectResources(module, result = []) {
  for (const resource of module?.resources ?? []) result.push(resource);
  for (const child of module?.child_modules ?? []) collectResources(child, result);
  return result;
}

function numericValues(value, key, result = []) {
  if (!value || typeof value !== "object") return result;
  if (Object.prototype.hasOwnProperty.call(value, key) && typeof value[key] === "number") result.push(value[key]);
  for (const child of Object.values(value)) numericValues(child, key, result);
  return result;
}

function hasAssignedIdentity(value) {
  if (!value || typeof value !== "object") return false;
  if (value.type === "UserAssigned" && Array.isArray(value.user_assigned_identity_ids) && value.user_assigned_identity_ids.length > 0) return true;
  return Object.values(value).some(hasAssignedIdentity);
}

function budgetThresholds(value, result = []) {
  if (!value || typeof value !== "object") return result;
  if (typeof value.threshold === "number") result.push(value.threshold);
  for (const child of Object.values(value)) budgetThresholds(child, result);
  return result;
}

function checkPlan(plan, environment) {
  if (plan.format_version === undefined || !plan.planned_values?.root_module) fail("plan JSON must be produced by terraform show -json");
  const resources = collectResources(plan.planned_values.root_module);
  const problems = [];
  const add = (message) => problems.push(message);

  for (const resource of resources) {
    const values = resource.values ?? {};
    if (taggableTypes.has(resource.type)) {
      const tags = values.tags ?? values.tags_all ?? {};
      for (const tag of requiredTags) if (!Object.prototype.hasOwnProperty.call(tags, tag)) add(`${resource.address}: missing required tag ${tag}`);
      if (environment === "demo" && tags.environment === "demo" && !tags.expires_on) add(`${resource.address}: demo resource is missing expires_on`);
    }
  }

  const postgres = resources.filter((resource) => resource.type === "azurerm_postgresql_flexible_server");
  const diagnostics = resources.filter((resource) => resource.type === "azurerm_monitor_diagnostic_setting");
  if (environment === "prod-reference") {
    if (postgres.length === 0) add("prod-reference: PostgreSQL plan resource is missing");
    for (const resource of postgres) {
      if (resource.values?.public_network_access_enabled !== false) add(`${resource.address}: production PostgreSQL must disable public network access`);
    }
    for (const resource of postgres) {
      const modulePrefix = resource.address.split(".azurerm_postgresql_flexible_server.")[0];
      const postgresId = resource.values?.id;
      const matchingDiagnostics = diagnostics.filter((diagnostic) => {
        const targetResourceId = diagnostic.values?.target_resource_id;
        if (targetResourceId) return Boolean(postgresId) && targetResourceId === postgresId;
        return !postgresId && diagnostic.address.startsWith(`${modulePrefix}.azurerm_monitor_diagnostic_setting.`);
      });
      if (matchingDiagnostics.length === 0 || matchingDiagnostics.some((diagnostic) => !diagnostic.values?.log_analytics_workspace_id)) {
        add(`${resource.address}: PostgreSQL diagnostics must target this module and a Log Analytics workspace`);
      }
    }
    for (const role of ["api", "worker"]) {
      const apps = resources.filter((resource) => resource.type === "azurerm_container_app" && resource.address.includes(`module.${role}`));
      if (apps.length === 0) add(`prod-reference: missing ${role} Container App plan resource`);
      for (const app of apps) {
        if (!hasAssignedIdentity(app.values)) add(`${app.address}: managed identity is required`);
        const minimums = numericValues(app.values, "min_replicas");
        const maximums = numericValues(app.values, "max_replicas");
        const expected = role === "api" ? 2 : 1;
        if (minimums.length === 0 || minimums.some((value) => value < expected)) add(`${app.address}: min_replicas must be at least ${expected}`);
        if (maximums.length === 0 || maximums.some((value, index) => value < (minimums[index] ?? expected))) add(`${app.address}: max_replicas must not be below min_replicas`);
      }
    }
  }

  if (environment === "bootstrap") {
    const budgets = resources.filter((resource) => resource.type === "azurerm_consumption_budget_subscription");
    if (budgets.length !== 1) add("bootstrap: exactly one subscription budget is required");
    const thresholds = budgets.flatMap((resource) => budgetThresholds(resource.values));
    if (JSON.stringify([...new Set(thresholds)].sort((a, b) => a - b)) !== JSON.stringify([50, 80, 100])) add("bootstrap: budget notifications must be exactly 50, 80, and 100 percent");
  }

  if (problems.length > 0) {
    const error = new Error(`Terraform plan policy failed for ${environment}:\n- ${problems.join("\n- ")}`);
    error.problems = problems;
    throw error;
  }
  return `Terraform plan policy passed for ${environment} (${resources.length} resources).`;
}

function selfTest() {
  const fixtureRoot = path.join(__dirname, "fixtures", "terraform-plan-policy", "v1");
  const valid = JSON.parse(fs.readFileSync(path.join(fixtureRoot, "valid-prod-reference.json"), "utf8"));
  const invalid = JSON.parse(fs.readFileSync(path.join(fixtureRoot, "invalid-prod-reference.json"), "utf8"));
  const validBootstrap = JSON.parse(fs.readFileSync(path.join(fixtureRoot, "valid-bootstrap.json"), "utf8"));
  const invalidBootstrap = JSON.parse(fs.readFileSync(path.join(fixtureRoot, "invalid-bootstrap.json"), "utf8"));
  checkPlan(valid, "prod-reference");
  checkPlan(validBootstrap, "bootstrap");
  try {
    checkPlan(invalid, "prod-reference");
    fail("invalid fixture unexpectedly passed");
  } catch (error) {
    if (!error.problems) throw error;
    for (const expected of ["public network access", "managed identity", "diagnostics", "min_replicas", "missing required tag"]) {
      if (!error.message.includes(expected)) fail(`invalid production fixture did not exercise ${expected}`);
    }
  }
  try {
    checkPlan(invalidBootstrap, "bootstrap");
    fail("invalid bootstrap fixture unexpectedly passed");
  } catch (error) {
    if (!error.problems || !error.message.includes("50, 80, and 100")) throw error;
  }
  console.log("Terraform plan policy self-test passed.");
}

const args = process.argv.slice(2);
if (args.includes("--self-test")) {
  selfTest();
} else {
  const environmentIndex = args.indexOf("--environment");
  const planIndex = args.indexOf("--plan-json");
  if (environmentIndex < 0 || planIndex < 0 || !args[environmentIndex + 1] || !args[planIndex + 1]) fail("usage: terraform-plan-policy.js --environment <name> --plan-json <terraform-show-json>");
  const plan = JSON.parse(fs.readFileSync(args[planIndex + 1], "utf8"));
  console.log(checkPlan(plan, args[environmentIndex + 1]));
}
