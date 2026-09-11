locals { notifications = { for threshold in var.thresholds : "threshold-${threshold}" => threshold } }
resource "azurerm_consumption_budget_subscription" "this" {
  count           = var.scope.kind == "subscription" ? 1 : 0
  name            = var.name
  subscription_id = var.scope.id
  amount          = var.amount
  time_grain      = "Monthly"
  time_period {
    start_date = var.start_date
    end_date   = var.end_date
  }
  dynamic "notification" {
    for_each = local.notifications
    content {
      enabled        = true
      threshold      = notification.value
      operator       = "GreaterThan"
      contact_emails = var.contacts.contact_emails
      contact_roles  = var.contacts.contact_roles
      contact_groups = var.contacts.contact_groups
    }
  }
}
resource "azurerm_consumption_budget_resource_group" "this" {
  count             = var.scope.kind == "resource_group" ? 1 : 0
  name              = var.name
  resource_group_id = var.scope.id
  amount            = var.amount
  time_grain        = "Monthly"
  time_period {
    start_date = var.start_date
    end_date   = var.end_date
  }
  dynamic "notification" {
    for_each = local.notifications
    content {
      enabled        = true
      threshold      = notification.value
      operator       = "GreaterThan"
      contact_emails = var.contacts.contact_emails
      contact_roles  = var.contacts.contact_roles
      contact_groups = var.contacts.contact_groups
    }
  }
}
