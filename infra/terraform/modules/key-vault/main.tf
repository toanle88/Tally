resource "azurerm_key_vault" "this" {
  name                       = var.name
  resource_group_name        = var.resource_group_name
  location                   = var.location
  tenant_id                  = var.tenant_id
  sku_name                   = "standard"
  purge_protection_enabled   = var.purge_protection_enabled
  soft_delete_retention_days = var.soft_delete_retention_days
  network_acls {
    bypass                     = var.bypass
    default_action             = var.default_action
    ip_rules                   = var.ip_rules
    virtual_network_subnet_ids = var.virtual_network_subnet_ids
  }
  tags = var.tags
}
check "demo_expiry" {
  assert {
    condition     = var.tags["environment"] != "demo" || (contains(keys(var.tags), "expires_on") && can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.tags["expires_on"])))
    error_message = "Demo resources require an ISO expires_on tag."
  }
}
resource "azurerm_role_assignment" "this" {
  for_each             = var.roles
  scope                = azurerm_key_vault.this.id
  role_definition_name = each.value.role_definition_name
  principal_id         = each.value.principal_id
}
