resource "azurerm_static_web_app" "this" {
  name                = var.name
  resource_group_name = var.resource_group_name
  location            = var.location
  sku_tier            = var.sku_tier
  repository_url      = var.repository == null ? null : var.repository.url
  repository_branch   = var.repository == null ? null : var.repository.branch
  repository_token    = var.repository_token
  tags                = var.tags
}
check "demo_expiry" {
  assert {
    condition     = var.tags["environment"] != "demo" || (contains(keys(var.tags), "expires_on") && can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.tags["expires_on"])))
    error_message = "Demo resources require an ISO expires_on tag."
  }
}
check "repository_configuration" {
  assert {
    condition     = (var.repository == null && var.repository_token == null) || (var.repository != null && var.repository_token != null)
    error_message = "Repository URL, branch, and token must be supplied together."
  }
}
