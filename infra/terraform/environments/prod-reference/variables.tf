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
variable "prod_container_apps_subnet_id" { type = string }
variable "prod_postgres_subnet_id" { type = string }
variable "prod_postgres_private_dns_zone_id" { type = string }
variable "prod_key_vault_subnet_ids" {
  type = set(string)
  validation {
    condition     = length(var.prod_key_vault_subnet_ids) > 0
    error_message = "Production Key Vault networking requires at least one approved subnet."
  }
}
variable "prod_postgres_sku_name" {
  type = string
  validation {
    condition     = trimspace(var.prod_postgres_sku_name) != ""
    error_message = "Production-reference PostgreSQL SKU must be supplied from measured capacity evidence."
  }
}
variable "prod_postgres_storage_mb" {
  type = number
  validation {
    condition     = var.prod_postgres_storage_mb >= 32768
    error_message = "Production-reference PostgreSQL storage must be at least 32768 MB."
  }
}
variable "prod_api_max_replicas" {
  type = number
  validation {
    condition     = var.prod_api_max_replicas >= 2
    error_message = "Production-reference API capacity must allow at least two replicas."
  }
}
variable "prod_worker_max_replicas" {
  type = number
  validation {
    condition     = var.prod_worker_max_replicas >= 1
    error_message = "Production-reference worker capacity must allow at least one replica."
  }
}
variable "prod_primary_availability_zone" { type = string }
variable "prod_standby_availability_zone" { type = string }
