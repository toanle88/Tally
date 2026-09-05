resource "azurerm_postgresql_flexible_server" "this" {
  name                              = var.name
  resource_group_name               = var.resource_group_name
  location                          = var.location
  version                           = var.postgres_version
  sku_name                          = var.sku_name
  storage_mb                        = var.storage_mb
  administrator_login               = var.administrator_login
  administrator_password_wo         = var.administrator_password_wo
  administrator_password_wo_version = var.administrator_password_wo_version
  backup_retention_days             = var.backup.retention_days
  geo_redundant_backup_enabled      = var.backup.geo_redundant_backup_enabled
  delegated_subnet_id               = var.network.mode == "private" ? var.network.delegated_subnet_id : null
  private_dns_zone_id               = var.network.mode == "private" ? var.network.private_dns_zone_id : null
  public_network_access_enabled     = var.network.mode == "public"
  tags                              = var.tags
  zone                              = try(var.high_availability.primary_availability_zone, null)
  dynamic "high_availability" {
    for_each = var.high_availability.enabled ? [var.high_availability] : []
    content {
      mode                      = try(high_availability.value.mode, "ZoneRedundant")
      standby_availability_zone = try(high_availability.value.standby_availability_zone, null)
    }
  }
}
check "demo_expiry" {
  assert {
    condition     = var.tags["environment"] != "demo" || (contains(keys(var.tags), "expires_on") && can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.tags["expires_on"])))
    error_message = "Demo resources require an ISO expires_on tag."
  }
}
resource "azurerm_postgresql_flexible_server_database" "this" {
  name      = var.database_name
  server_id = azurerm_postgresql_flexible_server.this.id
  charset   = "UTF8"
  collation = "en_US.utf8"
}
resource "azurerm_postgresql_flexible_server_configuration" "secure_transport" {
  name      = "require_secure_transport"
  server_id = azurerm_postgresql_flexible_server.this.id
  value     = "ON"
}
resource "azurerm_postgresql_flexible_server_configuration" "tls" {
  name      = "ssl_min_protocol_version"
  server_id = azurerm_postgresql_flexible_server.this.id
  value     = "TLSv1.2"
}
resource "azurerm_postgresql_flexible_server_firewall_rule" "this" {
  for_each         = var.network.mode == "public" ? { for rule in var.network.firewall_rules : rule.name => rule } : {}
  name             = each.value.name
  server_id        = azurerm_postgresql_flexible_server.this.id
  start_ip_address = each.value.start_ip_address
  end_ip_address   = each.value.end_ip_address
}
resource "azurerm_monitor_diagnostic_setting" "this" {
  name                       = "${var.name}-diagnostics"
  target_resource_id         = azurerm_postgresql_flexible_server.this.id
  log_analytics_workspace_id = var.diagnostics_workspace_id
  enabled_log { category = "PostgreSQLLogs" }
  enabled_metric {
    category = "AllMetrics"
  }
}
