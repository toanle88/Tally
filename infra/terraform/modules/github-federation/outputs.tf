output "client_id" { value = azuread_application.this.client_id }
output "principal_id" { value = azuread_service_principal.this.object_id }
