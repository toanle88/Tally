mock_provider "azurerm" {}

run "rejects_mutable_image" {
  command = plan

  variables {
    name                         = "tally-fin-dev-eastus-api-01"
    resource_group_name          = "rg-tally"
    location                     = "eastus"
    container_app_environment_id = "/subscriptions/test/resourceGroups/rg/providers/Microsoft.App/managedEnvironments/test"
    image = {
      registry_id           = "/subscriptions/test/resourceGroups/rg/providers/Microsoft.ContainerRegistry/registries/test"
      registry_login_server = "test.azurecr.io"
      repository            = "api"
      digest                = "latest"
    }
    identity_id   = "/subscriptions/test/resourceGroups/rg/providers/Microsoft.ManagedIdentity/userAssignedIdentities/test"
    workload_type = "api"
    cpu           = 0.5
    memory        = "1Gi"
    scaling       = { min_replicas = 0, max_replicas = 1, http_rule = { name = "http", concurrent_requests = 10 } }
    ingress       = { external_enabled = true, target_port = 8080, transport = "http", allow_insecure_connections = false }
    health_paths  = { live = "/health/live", ready = "/health/ready" }
    secret_refs   = []
    tags          = { application = "tally", environment = "dev", owner = "team", cost_center = "learning", managed_by = "terraform", data_classification = "internal" }
  }

  expect_failures = [var.image]
}
