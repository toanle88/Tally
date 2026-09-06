locals {
  profile = "prod-reference"
  # Azure name limits require the physical-name abbreviation; tags and state
  # retain the full prod-reference profile name.
  prefix = "${var.organization}-fin-pr-${var.region_code}"
  tags = {
    application         = "tally"
    environment         = local.profile
    owner               = var.owner
    cost_center         = var.cost_center
    managed_by          = "terraform"
    data_classification = "confidential"
  }
  image_api = {
    registry_id           = module.container_registry.id
    registry_login_server = module.container_registry.login_server
    repository            = var.api_image_repository
    digest                = var.api_image_digest
  }
  image_worker = {
    registry_id           = module.container_registry.id
    registry_login_server = module.container_registry.login_server
    repository            = var.worker_image_repository
    digest                = var.worker_image_digest
  }
}

module "resource_group" {
  source   = "../../modules/resource-group"
  name     = "${local.prefix}-rg-01"
  location = var.location
  tags     = local.tags
}

module "log_analytics" {
  source                    = "../../modules/log-analytics"
  workspace_name            = "${local.prefix}-law-01"
  application_insights_name = "${local.prefix}-appi-01"
  resource_group_name       = module.resource_group.name
  location                  = var.location
  retention_days            = 90
  tags                      = local.tags
}

module "container_registry" {
  source              = "../../modules/container-registry"
  name                = replace("${local.prefix}-acr-01", "-", "")
  resource_group_name = module.resource_group.name
  location            = var.location
  sku                 = "Standard"
  tags                = local.tags
}

module "container_app_environment" {
  source                     = "../../modules/container-app-environment"
  name                       = "${local.prefix}-cae-01"
  resource_group_name        = module.resource_group.name
  location                   = var.location
  log_analytics_workspace_id = module.log_analytics.workspace_id
  infrastructure_subnet_id   = var.prod_container_apps_subnet_id
  network_mode               = "private"
  zone_redundancy_enabled    = true
  tags                       = local.tags
}

module "api_identity" {
  source              = "../../modules/managed-identity"
  name                = "${local.prefix}-api-id-01"
  resource_group_name = module.resource_group.name
  location            = var.location
  roles = toset([
    { role_definition_name = "AcrPull", scope = module.container_registry.id },
    { role_definition_name = "Key Vault Secrets User", scope = module.key_vault.id }
  ])
  tags = local.tags
}

module "worker_identity" {
  source              = "../../modules/managed-identity"
  name                = "${local.prefix}-worker-id-01"
  resource_group_name = module.resource_group.name
  location            = var.location
  roles = toset([
    { role_definition_name = "AcrPull", scope = module.container_registry.id },
    { role_definition_name = "Key Vault Secrets User", scope = module.key_vault.id }
  ])
  tags = local.tags
}

module "api" {
  source                       = "../../modules/container-app"
  name                         = "${local.prefix}-api-01"
  resource_group_name          = module.resource_group.name
  location                     = var.location
  container_app_environment_id = module.container_app_environment.id
  image                        = local.image_api
  identity_id                  = module.api_identity.id
  workload_type                = "api"
  cpu                          = 0.5
  memory                       = "1Gi"
  scaling = {
    min_replicas = 2
    max_replicas = var.prod_api_max_replicas
    http_rule = {
      name                = "http-concurrency"
      concurrent_requests = 20
    }
  }
  ingress = {
    external_enabled           = false
    target_port                = 8080
    transport                  = "http"
    allow_insecure_connections = false
  }
  health_paths = { live = "/health/live", ready = "/health/ready" }
  secret_refs  = []
  tags         = local.tags
}

module "worker" {
  source                       = "../../modules/container-app"
  name                         = "${local.prefix}-worker-01"
  resource_group_name          = module.resource_group.name
  location                     = var.location
  container_app_environment_id = module.container_app_environment.id
  image                        = local.image_worker
  identity_id                  = module.worker_identity.id
  workload_type                = "worker"
  cpu                          = 0.5
  memory                       = "1Gi"
  scaling = {
    min_replicas = 1
    max_replicas = var.prod_worker_max_replicas
    backlog_rule = var.worker_backlog_rule
  }
  ingress      = null
  health_paths = null
  secret_refs  = []
  tags         = local.tags
}

module "static_web_app" {
  source              = "../../modules/static-web-app"
  name                = "${local.prefix}-swa-01"
  resource_group_name = module.resource_group.name
  location            = var.location
  sku_tier            = "Standard"
  repository          = null
  repository_token    = null
  tags                = local.tags
}

module "postgresql" {
  source                            = "../../modules/postgresql"
  name                              = "${local.prefix}-pg-01"
  resource_group_name               = module.resource_group.name
  location                          = var.location
  postgres_version                  = "18"
  sku_name                          = var.prod_postgres_sku_name
  storage_mb                        = var.prod_postgres_storage_mb
  administrator_login               = var.postgres_administrator_login
  administrator_password_wo         = var.postgres_administrator_password_wo
  administrator_password_wo_version = var.postgres_administrator_password_wo_version
  database_name                     = var.postgres_database_name
  backup                            = { retention_days = 35, geo_redundant_backup_enabled = true }
  high_availability = {
    enabled                   = true
    mode                      = "ZoneRedundant"
    primary_availability_zone = var.prod_primary_availability_zone
    standby_availability_zone = var.prod_standby_availability_zone
  }
  network = {
    mode                = "private"
    delegated_subnet_id = var.prod_postgres_subnet_id
    private_dns_zone_id = var.prod_postgres_private_dns_zone_id
    firewall_rules      = toset([])
  }
  diagnostics_workspace_id = module.log_analytics.workspace_id
  tags                     = local.tags
}

module "key_vault" {
  source                     = "../../modules/key-vault"
  name                       = "${local.prefix}-kv-01"
  resource_group_name        = module.resource_group.name
  location                   = var.location
  tenant_id                  = var.tenant_id
  roles                      = toset([])
  bypass                     = "AzureServices"
  default_action             = "Deny"
  ip_rules                   = toset([])
  virtual_network_subnet_ids = var.prod_key_vault_subnet_ids
  soft_delete_retention_days = 90
  purge_protection_enabled   = true
  tags                       = local.tags
}

resource "azurerm_management_lock" "environment" {
  name       = "${local.prefix}-deletion-guard"
  scope      = module.resource_group.id
  lock_level = "CanNotDelete"
  notes      = "Production-reference resources require an explicit reviewed lock-removal change."

  lifecycle {
    prevent_destroy = true
  }
}

check "production_private_network" {
  assert {
    condition = (var.prod_container_apps_subnet_id != null &&
      var.prod_postgres_subnet_id != null &&
      var.prod_postgres_private_dns_zone_id != null &&
    length(var.prod_key_vault_subnet_ids) > 0)
    error_message = "Production-reference networking must use approved private subnet and DNS inputs."
  }
}

check "production_deletion_guard" {
  assert {
    condition     = azurerm_management_lock.environment.lock_level == "CanNotDelete"
    error_message = "Production-reference must retain a CanNotDelete management lock."
  }
}
