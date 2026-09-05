variable "location" {
  description = "Azure region for the remote-state resource group and identities."
  type        = string
}

variable "resource_group_name" {
  description = "Dedicated resource group name for Terraform state resources."
  type        = string
}

variable "storage_account_name" {
  description = "Globally unique Azure Storage account name."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9]{3,24}$", var.storage_account_name))
    error_message = "storage_account_name must contain only lowercase letters and digits and be 3-24 characters long."
  }
}

variable "region_code" {
  description = "Short region code used in managed identity names."
  type        = string

  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.region_code))
    error_message = "region_code must contain only lowercase letters, digits, and hyphens."
  }
}

variable "owner" {
  description = "Accountable owner recorded on all bootstrap resources."
  type        = string
}

variable "cost_center" {
  description = "Cost center recorded on all bootstrap resources."
  type        = string
}

variable "data_classification" {
  description = "Classification for state resources and identities."
  type        = string
  default     = "confidential"
}

variable "bootstrap_principal_object_id" {
  description = "Object ID of the Entra principal applying the bootstrap configuration."
  type        = string

  validation {
    condition     = can(regex("^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$", var.bootstrap_principal_object_id))
    error_message = "bootstrap_principal_object_id must be a UUID object ID."
  }
}

variable "demo_expires_on" {
  description = "Expiry date tag for the disposable demo state identity, in YYYY-MM-DD format."
  type        = string

  validation {
    condition     = can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.demo_expires_on))
    error_message = "demo_expires_on must use YYYY-MM-DD format."
  }
}

variable "tags" {
  description = "Additional non-sensitive tags merged with the required baseline tags."
  type        = map(string)
  default     = {}
}
