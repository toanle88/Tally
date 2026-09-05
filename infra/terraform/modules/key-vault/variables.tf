variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "tenant_id" { type = string }
variable "roles" { type = set(object({ principal_id = string, role_definition_name = string })) }
variable "bypass" { type = string }
variable "default_action" {
  type = string
  validation {
    condition     = var.default_action == "Deny"
    error_message = "Key Vault must default to deny."
  }
}
variable "ip_rules" { type = set(string) }
variable "virtual_network_subnet_ids" { type = set(string) }
variable "soft_delete_retention_days" {
  type = number
  validation {
    condition     = var.soft_delete_retention_days >= 7 && var.soft_delete_retention_days <= 90
    error_message = "Key Vault retention must be between 7 and 90 days."
  }
}
variable "purge_protection_enabled" { type = bool }
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
