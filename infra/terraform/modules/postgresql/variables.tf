variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "postgres_version" {
  type = string
  validation {
    condition     = var.postgres_version == "18"
    error_message = "PostgreSQL major version 18 is required."
  }
}
variable "sku_name" { type = string }
variable "storage_mb" {
  type = number
  validation {
    condition     = var.storage_mb >= 32768
    error_message = "PostgreSQL storage must be at least 32768 MB."
  }
}
variable "administrator_login" { type = string }
variable "administrator_password_wo" {
  type      = string
  sensitive = true
}
variable "administrator_password_wo_version" {
  type = number
  validation {
    condition     = var.administrator_password_wo_version >= 1
    error_message = "Write-only password version must be positive."
  }
}
variable "database_name" { type = string }
variable "backup" {
  type = object({ retention_days = number, geo_redundant_backup_enabled = bool })
  validation {
    condition     = var.backup.retention_days >= 7 && var.backup.retention_days <= 35
    error_message = "Backup retention must be between 7 and 35 days."
  }
}
variable "high_availability" { type = object({ enabled = bool, mode = optional(string), primary_availability_zone = optional(string), standby_availability_zone = optional(string) }) }
variable "network" {
  type = object({ mode = string, delegated_subnet_id = optional(string), private_dns_zone_id = optional(string), firewall_rules = set(object({ name = string, start_ip_address = string, end_ip_address = string })) })
  validation {
    condition     = (var.network.mode == "public" && var.network.delegated_subnet_id == null && var.network.private_dns_zone_id == null && length(var.network.firewall_rules) > 0) || (var.network.mode == "private" && var.network.delegated_subnet_id != null && var.network.private_dns_zone_id != null && length(var.network.firewall_rules) == 0)
    error_message = "PostgreSQL network must be explicit public-with-firewall or private-with-subnet-and-DNS."
  }
}
check "high_availability_configuration" {
  assert {
    condition     = !var.high_availability.enabled || (try(var.high_availability.mode, "ZoneRedundant") == "ZoneRedundant" && try(var.high_availability.standby_availability_zone, null) != null)
    error_message = "Enabled PostgreSQL HA requires ZoneRedundant mode and a standby availability zone."
  }
}
variable "diagnostics_workspace_id" { type = string }
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform"
    error_message = "Required Terraform tags are mandatory."
  }
}
