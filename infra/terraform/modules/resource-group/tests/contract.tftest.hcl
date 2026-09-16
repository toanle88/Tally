mock_provider "azurerm" {}

variables {
  name     = "tally-fin-dev-eus-rg-01"
  location = "eastus"
  tags = {
    application         = "tally"
    environment         = "dev"
    owner               = "learning-owner"
    cost_center         = "learning"
    managed_by          = "terraform"
    data_classification = "synthetic"
  }
}

run "without_ci_principal" {
  command = plan
}

run "with_ci_principal" {
  command = plan

  variables {
    ci_deployment_principal_id = "00000000-0000-0000-0000-000000000001"
  }
}
