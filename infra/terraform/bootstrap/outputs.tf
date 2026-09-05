output "state_resource_group_id" {
  description = "Resource ID of the dedicated Terraform state resource group."
  value       = azurerm_resource_group.state.id
}

output "state_storage_account_id" {
  description = "Resource ID of the Terraform state storage account."
  value       = azurerm_storage_account.state.id
}

output "state_storage_account_name" {
  description = "Name of the Terraform state storage account."
  value       = azurerm_storage_account.state.name
}

output "state_container_ids" {
  description = "Resource IDs of the isolated state containers."
  value = {
    for name, container in azurerm_storage_container.state : name => container.id
  }
}

output "state_container_names" {
  description = "Names of the isolated state containers."
  value       = sort([for container in azurerm_storage_container.state : container.name])
}

output "state_keys" {
  description = "Explicit state key for each bootstrap and workload root."
  value       = local.state_keys
}

output "state_identity_client_ids" {
  description = "Client IDs of the per-environment state identities."
  value = {
    for environment, identity in azurerm_user_assigned_identity.state : environment => identity.client_id
  }
}

output "state_identity_principal_ids" {
  description = "Principal IDs of the per-environment state identities."
  value = {
    for environment, identity in azurerm_user_assigned_identity.state : environment => identity.principal_id
  }
}
