resource "azurerm_container_app_environment" "this" {
  name                       = var.name
  resource_group_name        = var.resource_group_name
  location                   = var.location
  log_analytics_workspace_id = var.log_analytics_workspace_id
  infrastructure_subnet_id   = var.network_mode == "private" ? var.infrastructure_subnet_id : null
  zone_redundancy_enabled    = var.network_mode == "private" ? var.zone_redundancy_enabled : null
  tags                       = var.tags
}
check "demo_expiry" {
  assert {
    condition     = var.tags["environment"] != "demo" || (contains(keys(var.tags), "expires_on") && can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.tags["expires_on"])))
    error_message = "Demo resources require an ISO expires_on tag."
  }
}
