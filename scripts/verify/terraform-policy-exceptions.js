const fs = require("fs");
const path = require("path");

const file = path.join(__dirname, "terraform-policy-exceptions.json");
const document = JSON.parse(fs.readFileSync(file, "utf8"));
if (document.schema_version !== 1 || !Array.isArray(document.exceptions)) {
  throw new Error("policy exception manifest must use schema_version 1 and an exceptions array");
}

for (const exception of document.exceptions) {
  for (const field of ["rule_id", "path", "rationale", "owner", "expires_on"]) {
    if (typeof exception[field] !== "string" || exception[field].length === 0) {
      throw new Error(`policy exception is missing ${field}`);
    }
  }
  if (exception.path.includes("*") || exception.path.includes("..")) {
    throw new Error(`policy exception path must be a concrete repository path: ${exception.path}`);
  }
  if (exception.path.startsWith("infra/terraform/bootstrap") || exception.path.startsWith("infra/terraform/environments/prod-reference")) {
    throw new Error(`policy exceptions are not permitted for bootstrap or prod-reference: ${exception.path}`);
  }
  if (!/^\d{4}-\d{2}-\d{2}$/.test(exception.expires_on)) {
    throw new Error(`policy exception expiry must use YYYY-MM-DD: ${exception.expires_on}`);
  }
  if (exception.expires_on < new Date().toISOString().slice(0, 10)) {
    throw new Error(`policy exception is expired: ${exception.rule_id} at ${exception.path}`);
  }
}

console.log(`Terraform policy exception manifest passed (${document.exceptions.length} entries).`);
