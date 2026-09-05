variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "sku_tier" {
  type = string
  validation {
    condition     = contains(["Free", "Standard"], var.sku_tier)
    error_message = "Static Web App SKU must be Free or Standard."
  }
}
variable "repository" { type = object({ url = string, branch = string }) }
variable "repository_token" {
  type      = string
  sensitive = true
}
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
