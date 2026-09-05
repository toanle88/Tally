output "id" { value = var.scope.kind == "subscription" ? azurerm_consumption_budget_subscription.this[0].id : azurerm_consumption_budget_resource_group.this[0].id }
