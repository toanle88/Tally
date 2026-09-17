variable "name" { type = string }
variable "location" { type = string }
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key) && trimspace(var.tags[key]) != ""]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform ownership and classification tags are mandatory."
  }
}

variable "ci_deployment_principal_id" {
  description = "Optional object ID of the environment-specific GitHub Actions deployment principal."
  type        = string
  default     = null

  validation {
    condition     = var.ci_deployment_principal_id == null || can(regex("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$", var.ci_deployment_principal_id))
    error_message = "ci_deployment_principal_id must be a UUID when supplied."
  }
}
