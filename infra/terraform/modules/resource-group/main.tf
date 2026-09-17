resource "azurerm_resource_group" "this" {
  name     = var.name
  location = var.location
  tags     = var.tags
}

resource "azurerm_role_assignment" "ci_deployer" {
  count                = var.ci_deployment_principal_id == null ? 0 : 1
  scope                = azurerm_resource_group.this.id
  role_definition_name = "Contributor"
  principal_id         = var.ci_deployment_principal_id
}
check "demo_expiry" {
  assert {
    condition     = var.tags["environment"] != "demo" || (contains(keys(var.tags), "expires_on") && can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.tags["expires_on"])))
    error_message = "Demo resources require an ISO expires_on tag."
  }
}
