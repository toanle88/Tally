locals {
  profile = "dev"
  prefix  = "${var.organization}-fin-${local.profile}-${var.region_code}"
  tags = {
    application         = "tally"
    environment         = local.profile
    owner               = var.owner
    cost_center         = var.cost_center
    managed_by          = "terraform"
    data_classification = "synthetic"
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
  retention_days            = 30
  tags                      = local.tags
}

module "container_registry" {
  source              = "../../modules/container-registry"
  name                = replace("${local.prefix}-acr-01", "-", "")
  resource_group_name = module.resource_group.name
  location            = var.location
  sku                 = "Basic"
  tags                = local.tags
}

module "container_app_environment" {
  source                     = "../../modules/container-app-environment"
  name                       = "${local.prefix}-cae-01"
  resource_group_name        = module.resource_group.name
  location                   = var.location
  log_analytics_workspace_id = module.log_analytics.workspace_id
  infrastructure_subnet_id   = null
  network_mode               = "public"
  zone_redundancy_enabled    = false
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
    min_replicas = 0
    max_replicas = 1
    http_rule = {
      name                = "http-concurrency"
      concurrent_requests = 20
    }
  }
  ingress = {
    external_enabled           = true
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
    min_replicas = 0
    max_replicas = 1
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
  sku_tier            = "Free"
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
  sku_name                          = "B_Standard_B1ms"
  storage_mb                        = 32768
  administrator_login               = var.postgres_administrator_login
  administrator_password_wo         = var.postgres_administrator_password_wo
  administrator_password_wo_version = var.postgres_administrator_password_wo_version
  database_name                     = var.postgres_database_name
  backup                            = { retention_days = 7, geo_redundant_backup_enabled = false }
  high_availability                 = { enabled = false }
  network = {
    mode                = "public"
    delegated_subnet_id = null
    private_dns_zone_id = null
    firewall_rules      = var.learning_firewall_rules
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
  ip_rules                   = var.learning_key_vault_ip_rules
  virtual_network_subnet_ids = toset([])
  soft_delete_retention_days = 7
  purge_protection_enabled   = false
  tags                       = local.tags
}

check "profile_contract" {
  assert {
    condition     = local.profile == "dev" && var.learning_firewall_rules != null
    error_message = "The dev root must remain an explicit learning profile."
  }
}
