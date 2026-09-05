variable "name" { type = string }
variable "location" { type = string }
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key) && trimspace(var.tags[key]) != ""]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform ownership and classification tags are mandatory."
  }
}
