variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "roles" { type = set(object({ role_definition_name = string, scope = string })) }
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
