mock_provider "azurerm" {}

run "rejects_non_v18" {
  command = plan

  variables {
    name                              = "tally-fin-dev-eastus-pg-01"
    resource_group_name               = "rg-tally"
    location                          = "eastus"
    postgres_version                  = "16"
    sku_name                          = "B_Standard_B1ms"
    storage_mb                        = 32768
    administrator_login               = "tallyadmin"
    administrator_password_wo         = "test-password"
    administrator_password_wo_version = 1
    database_name                     = "tally"
    backup                            = { retention_days = 7, geo_redundant_backup_enabled = false }
    high_availability                 = { enabled = false }
    network                           = { mode = "public", firewall_rules = [{ name = "learning", start_ip_address = "203.0.113.10", end_ip_address = "203.0.113.10" }] }
    diagnostics_workspace_id          = "/subscriptions/test/resourceGroups/rg/providers/Microsoft.OperationalInsights/workspaces/test"
    tags                              = { application = "tally", environment = "dev", owner = "team", cost_center = "learning", managed_by = "terraform", data_classification = "internal" }
  }

  expect_failures = [var.postgres_version]
}
