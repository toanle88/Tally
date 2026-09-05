variable "name" { type = string }
variable "repository" { type = string }
variable "environment" { type = string }
variable "subject" {
  type = string
  validation {
    condition     = length(var.subject) > 0 && !strcontains(var.subject, "*")
    error_message = "GitHub federation subjects must be explicit and non-wildcard."
  }
}
