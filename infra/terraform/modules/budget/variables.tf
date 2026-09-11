variable "name" { type = string }
variable "scope" {
  type = object({ kind = string, id = string })
  validation {
    condition     = contains(["subscription", "resource_group"], var.scope.kind)
    error_message = "Budget scope must be subscription or resource_group."
  }
}
variable "amount" {
  type = number
  validation {
    condition     = var.amount > 0
    error_message = "Budget amount must be positive."
  }
}
variable "thresholds" {
  type = set(number)
  validation {
    condition     = length(var.thresholds) > 0 && alltrue([for threshold in var.thresholds : threshold > 0])
    error_message = "Budget thresholds must be positive."
  }
}
variable "contacts" {
  type = object({ contact_emails = set(string), contact_roles = set(string), contact_groups = set(string) })

  validation {
    condition     = length(var.contacts.contact_emails) + length(var.contacts.contact_roles) + length(var.contacts.contact_groups) > 0
    error_message = "contacts must contain at least one email, role, or action-group recipient."
  }
}

variable "start_date" {
  description = "Inclusive UTC start date for the recurring budget period."
  type        = string
  default     = "2026-01-01T00:00:00Z"

  validation {
    condition     = can(formatdate("YYYY-MM-DD", var.start_date))
    error_message = "start_date must be an RFC3339 timestamp accepted by Terraform."
  }
}

variable "end_date" {
  description = "Exclusive UTC end date for the recurring budget period."
  type        = string
  default     = "2030-01-01T00:00:00Z"

  validation {
    condition     = can(formatdate("YYYY-MM-DD", var.end_date))
    error_message = "end_date must be an RFC3339 timestamp accepted by Terraform."
  }
}
