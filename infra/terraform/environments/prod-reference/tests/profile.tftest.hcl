mock_provider "azurerm" {}
mock_provider "azuread" {}
mock_provider "random" {}

variables {
  organization                               = "tally"
  location                                   = "eastus"
  region_code                                = "eus"
  owner                                      = "reference-owner"
  cost_center                                = "reference"
  tenant_id                                  = "00000000-0000-0000-0000-000000000001"
  api_image_repository                       = "tally-api"
  api_image_digest                           = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
  worker_image_repository                    = "tally-worker"
  worker_image_digest                        = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
  worker_backlog_rule                        = { name = "outbox-backlog", custom_rule_type = "prometheus", metadata = { serverAddress = "https://prometheus.invalid", query = "sum(tally_worker_backlog)", threshold = "10" } }
  postgres_administrator_login               = "tallyadmin"
  postgres_administrator_password_wo         = "test-only-write-only-input"
  postgres_administrator_password_wo_version = 1
  postgres_database_name                     = "tally"
  prod_container_apps_subnet_id              = "/subscriptions/test/resourceGroups/network/providers/Microsoft.Network/virtualNetworks/test/subnets/apps"
  prod_postgres_subnet_id                    = "/subscriptions/test/resourceGroups/network/providers/Microsoft.Network/virtualNetworks/test/subnets/postgres"
  prod_postgres_private_dns_zone_id          = "/subscriptions/test/resourceGroups/network/providers/Microsoft.Network/privateDnsZones/privatelink.postgres.database.azure.com"
  prod_key_vault_subnet_ids                  = toset(["/subscriptions/test/resourceGroups/network/providers/Microsoft.Network/virtualNetworks/test/subnets/apps"])
  prod_postgres_sku_name                     = "GP_Standard_D2s_v3"
  prod_postgres_storage_mb                   = 65536
  prod_api_max_replicas                      = 4
  prod_worker_max_replicas                   = 2
  prod_primary_availability_zone             = "1"
  prod_standby_availability_zone             = "2"
}

run "profile_outputs" {
  command = plan

  assert {
    condition     = output.environment_profile == "prod-reference"
    error_message = "prod-reference must expose the production-reference profile name."
  }

  assert {
    condition     = output.production_deletion_lock_enabled == true
    error_message = "prod-reference must expose its deletion guard."
  }
}
