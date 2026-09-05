output "server_fqdn" { value = azurerm_postgresql_flexible_server.this.fqdn }
output "database_id" { value = azurerm_postgresql_flexible_server_database.this.id }
