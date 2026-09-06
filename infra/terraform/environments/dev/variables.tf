variable "organization" {
  type = string
  validation {
    condition     = can(regex("^[a-z0-9]{1,5}$", var.organization))
    error_message = "organization must contain 1-5 lowercase letters or digits so Azure names remain within limits."
  }
}

variable "location" { type = string }
variable "region_code" {
  type = string
  validation {
    condition     = can(regex("^[a-z0-9]{2,4}$", var.region_code))
    error_message = "region_code must contain 2-4 lowercase letters or digits so Azure names remain within limits."
  }
}
variable "owner" { type = string }
variable "cost_center" { type = string }
variable "tenant_id" { type = string }
variable "api_image_repository" { type = string }
variable "api_image_digest" {
  type = string
  validation {
    condition     = can(regex("^sha256:[0-9a-f]{64}$", var.api_image_digest))
    error_message = "API images must use lowercase immutable sha256 digests."
  }
}
variable "worker_image_repository" { type = string }
variable "worker_image_digest" {
  type = string
  validation {
    condition     = can(regex("^sha256:[0-9a-f]{64}$", var.worker_image_digest))
    error_message = "Worker images must use lowercase immutable sha256 digests."
  }
}
variable "worker_backlog_rule" {
  type = object({
    name             = string
    custom_rule_type = string
    metadata         = map(string)
    authentication   = optional(object({ secret_name = string, trigger_parameter = string }))
  })
  validation {
    condition = (trimspace(var.worker_backlog_rule.name) != "" &&
      trimspace(var.worker_backlog_rule.custom_rule_type) != "" &&
      length(var.worker_backlog_rule.metadata) > 0 &&
      (var.worker_backlog_rule.custom_rule_type != "prometheus" || alltrue([
        for key in ["serverAddress", "query", "threshold"] : contains(keys(var.worker_backlog_rule.metadata), key)
    ])))
    error_message = "Worker backlog scaling requires a named rule, type, metadata, and Prometheus serverAddress/query/threshold when applicable."
  }
}
variable "postgres_administrator_login" {
  type = string
  validation {
    condition     = can(regex("^[a-z][a-z0-9_]{2,31}$", var.postgres_administrator_login))
    error_message = "PostgreSQL administrator login must be 3-32 lowercase characters."
  }
}
variable "postgres_administrator_password_wo" {
  type      = string
  sensitive = true
}
variable "postgres_administrator_password_wo_version" {
  type = number
  validation {
    condition     = var.postgres_administrator_password_wo_version >= 1
    error_message = "PostgreSQL write-only password version must be positive."
  }
}
variable "postgres_database_name" { type = string }
variable "learning_firewall_rules" {
  type = set(object({ name = string, start_ip_address = string, end_ip_address = string }))
  validation {
    condition = length(var.learning_firewall_rules) > 0 && alltrue([
      for rule in var.learning_firewall_rules :
      rule.start_ip_address == rule.end_ip_address &&
      rule.start_ip_address != "0.0.0.0" &&
      rule.start_ip_address != "255.255.255.255" &&
      can(regex("^[0-9]{1,3}(\\.[0-9]{1,3}){3}$", rule.start_ip_address)) &&
      can(cidrhost("${rule.start_ip_address}/32", 0))
    ])
    error_message = "Learning PostgreSQL access must use explicit, valid single-host IPv4 firewall rules."
  }
}
variable "learning_key_vault_ip_rules" {
  type = set(string)
  validation {
    condition = length(var.learning_key_vault_ip_rules) > 0 && alltrue([
      for address in var.learning_key_vault_ip_rules :
      address != "0.0.0.0" && address != "255.255.255.255" &&
      can(regex("^[0-9]{1,3}(\\.[0-9]{1,3}){3}$", address)) &&
      can(cidrhost("${address}/32", 0))
    ])
    error_message = "Learning Key Vault access must use explicit, valid single-host IPv4 rules."
  }
}
