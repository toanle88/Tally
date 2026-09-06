mock_provider "azurerm" {}
mock_provider "azuread" {}
mock_provider "random" {}

variables {
  organization                               = "tally"
  location                                   = "eastus"
  region_code                                = "eus"
  owner                                      = "learning-owner"
  cost_center                                = "learning"
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
  learning_firewall_rules                    = toset([{ name = "test-host", start_ip_address = "203.0.113.10", end_ip_address = "203.0.113.10" }])
  learning_key_vault_ip_rules                = toset(["203.0.113.10"])
}

run "profile_outputs" {
  command = plan

  assert {
    condition     = output.environment_profile == "dev"
    error_message = "dev must expose the dev profile name."
  }

  assert {
    condition     = output.production_deletion_lock_enabled == false
    error_message = "dev must not expose a production deletion lock."
  }
}

run "rejects_broad_learning_firewall" {
  command = plan

  variables {
    learning_firewall_rules = toset([{ name = "open", start_ip_address = "0.0.0.0", end_ip_address = "255.255.255.255" }])
  }

  expect_failures = [var.learning_firewall_rules]
}

run "rejects_incomplete_prometheus_rule" {
  command = plan

  variables {
    worker_backlog_rule = { name = "outbox-backlog", custom_rule_type = "prometheus", metadata = { threshold = "10" } }
  }

  expect_failures = [var.worker_backlog_rule]
}

run "rejects_region_code_that_exceeds_azure_names" {
  command = plan

  variables {
    region_code = "eustu"
  }

  expect_failures = [var.region_code]
}
