variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "roles" {
  type = set(object({ role_definition_name = string, scope = string }))
  validation {
    condition     = length(distinct([for role in var.roles : role.role_definition_name])) == length(var.roles)
    error_message = "Managed identity role_definition_name values must be unique."
  }
}
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
