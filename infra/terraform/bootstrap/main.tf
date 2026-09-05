locals {
  state_containers = toset([
    "bootstrap",
    "dev",
    "demo",
    "prod-reference",
  ])

  workload_environments = toset([
    "dev",
    "demo",
    "prod-reference",
  ])

  shared_tags = merge(var.tags, {
    application         = "tally"
    environment         = "shared"
    owner               = var.owner
    cost_center         = var.cost_center
    managed_by          = "terraform"
    data_classification = var.data_classification
  })

  identity_tags = {
    for environment in local.workload_environments : environment => merge(var.tags, {
      application         = "tally"
      environment         = environment
      owner               = var.owner
      cost_center         = var.cost_center
      managed_by          = "terraform"
      data_classification = var.data_classification
      }, environment == "demo" ? {
      expires_on = var.demo_expires_on
    } : {})
  }

  state_keys = {
    for container in local.state_containers : container => "${container}/terraform.tfstate"
  }
}

resource "azurerm_resource_group" "state" {
  name     = var.resource_group_name
  location = var.location
  tags     = local.shared_tags
}

resource "azurerm_storage_account" "state" {
  name                              = var.storage_account_name
  resource_group_name               = azurerm_resource_group.state.name
  location                          = azurerm_resource_group.state.location
  account_kind                      = "StorageV2"
  account_tier                      = "Standard"
  account_replication_type          = "LRS"
  https_traffic_only_enabled        = true
  min_tls_version                   = "TLS1_2"
  allow_nested_items_to_be_public   = false
  shared_access_key_enabled         = false
  default_to_oauth_authentication   = true
  infrastructure_encryption_enabled = true
  public_network_access_enabled     = true
  tags                              = local.shared_tags

  blob_properties {
    versioning_enabled = true

    delete_retention_policy {
      days = 7
    }
  }
}

resource "azurerm_storage_container" "state" {
  for_each = local.state_containers

  name                  = each.key
  storage_account_id    = azurerm_storage_account.state.id
  container_access_type = "private"
}

resource "azurerm_user_assigned_identity" "state" {
  for_each = local.workload_environments

  name                = "tally-fin-${each.key}-${var.region_code}-tfstate-01"
  resource_group_name = azurerm_resource_group.state.name
  location            = azurerm_resource_group.state.location
  tags                = local.identity_tags[each.key]
}

resource "azurerm_role_assignment" "bootstrap_operator" {
  scope                = azurerm_storage_container.state["bootstrap"].id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = var.bootstrap_principal_object_id
}

resource "azurerm_role_assignment" "environment_state" {
  for_each = local.workload_environments

  scope                = azurerm_storage_container.state[each.key].id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.state[each.key].principal_id
}
