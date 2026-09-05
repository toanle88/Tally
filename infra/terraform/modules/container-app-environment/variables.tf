variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "log_analytics_workspace_id" { type = string }
variable "infrastructure_subnet_id" { type = string }
variable "network_mode" {
  type = string
  validation {
    condition     = contains(["public", "private"], var.network_mode)
    error_message = "Container Apps network mode must be public or private."
  }
}
variable "zone_redundancy_enabled" { type = bool }
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
check "private_network" {
  assert {
    condition     = var.network_mode == "private" ? var.infrastructure_subnet_id != null : var.infrastructure_subnet_id == null
    error_message = "Private Container Apps environments require a subnet; public environments must not set one."
  }
}
