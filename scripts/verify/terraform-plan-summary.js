const fs = require("fs");

const [planPath, environment, commit] = process.argv.slice(2);
if (!planPath || !environment || !commit) {
  console.error("usage: terraform-plan-summary.js <plan-json> <environment> <commit>");
  process.exit(2);
}

const plan = JSON.parse(fs.readFileSync(planPath, "utf8"));
const changes = Array.isArray(plan.resource_changes) ? plan.resource_changes : [];
const actions = changes.map((change) => ({
  address: change.address,
  actions: Array.isArray(change.change?.actions) ? change.change.actions : [],
}));

const counts = { create: 0, update: 0, delete: 0, replace: 0, read: 0 };
for (const change of actions) {
  const set = new Set(change.actions);
  if (set.has("create") && set.has("delete")) counts.replace += 1;
  else if (set.has("create")) counts.create += 1;
  else if (set.has("update")) counts.update += 1;
  else if (set.has("delete")) counts.delete += 1;
  else if (set.has("read")) counts.read += 1;
}

process.stdout.write(`${JSON.stringify({
  environment,
  commit,
  terraform_version: plan.terraform_version ?? "unknown",
  resource_change_counts: counts,
  resource_actions: actions,
  sensitive_values_included: false,
}, null, 2)}\n`);
