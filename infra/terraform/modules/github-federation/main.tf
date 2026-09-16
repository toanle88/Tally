resource "azuread_application" "this" { display_name = var.name }
resource "azuread_service_principal" "this" { client_id = azuread_application.this.client_id }
#checkov:skip=CKV_AZURE_249:Subject is a validated module input; bootstrap supplies exact repository/environment subjects and the CI contract verifies them.
resource "azuread_application_federated_identity_credential" "github" {
  application_id = azuread_application.this.id
  display_name   = "github-${var.environment}"
  description    = "GitHub Actions OIDC for ${var.repository}"
  issuer         = "https://token.actions.githubusercontent.com"
  subject        = var.subject
  audiences      = ["api://AzureADTokenExchange"]
}
