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
variable "contacts" { type = object({ contact_emails = set(string), contact_roles = set(string), contact_groups = set(string) }) }
