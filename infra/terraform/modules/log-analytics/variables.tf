variable "workspace_name" { type = string }
variable "application_insights_name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "retention_days" {
  type = number
  validation {
    condition     = var.retention_days >= 30 && var.retention_days <= 730
    error_message = "Retention must be between 30 and 730 days."
  }
}
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
