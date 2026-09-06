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
  demo_expires_on                            = "2099-12-31"
}

run "profile_outputs" {
  command = plan

  assert {
    condition     = output.environment_profile == "demo"
    error_message = "demo must expose the demo profile name."
  }

  assert {
    condition     = output.production_deletion_lock_enabled == false
    error_message = "demo must not expose a production deletion lock."
  }
}

run "rejects_invalid_expiry" {
  command = plan

  variables {
    demo_expires_on = "not-a-date"
  }

  expect_failures = [var.demo_expires_on]
}
