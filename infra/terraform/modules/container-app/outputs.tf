output "id" { value = azurerm_container_app.this.id }
output "url" { value = try("https://${azurerm_container_app.this.ingress[0].fqdn}", null) }
output "identity_id" { value = var.identity_id }
