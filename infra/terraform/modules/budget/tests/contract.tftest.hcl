mock_provider "azurerm" {}

run "rejects_empty_thresholds" {
  command = plan

  variables {
    name       = "tally-learning-monthly"
    scope      = { kind = "subscription", id = "/subscriptions/test" }
    amount     = 20
    thresholds = []
    contacts   = { contact_emails = ["owner@example.test"], contact_roles = [], contact_groups = [] }
  }

  expect_failures = [var.thresholds]
}

run "rejects_missing_contacts" {
  command = plan

  variables {
    name       = "tally-learning-monthly"
    scope      = { kind = "subscription", id = "/subscriptions/test" }
    amount     = 20
    thresholds = [50, 80, 100]
    contacts   = { contact_emails = [], contact_roles = [], contact_groups = [] }
  }

  expect_failures = [var.contacts]
}
