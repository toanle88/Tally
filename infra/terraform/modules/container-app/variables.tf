variable "name" { type = string }
variable "resource_group_name" { type = string }
variable "location" { type = string }
variable "container_app_environment_id" { type = string }
variable "image" {
  type = object({ registry_id = string, registry_login_server = string, repository = string, digest = string })
  validation {
    condition     = can(regex("^sha256:[0-9a-f]{64}$", var.image.digest))
    error_message = "Container images must use a lowercase immutable sha256 digest."
  }
}
variable "identity_id" { type = string }
variable "workload_type" {
  type = string
  validation {
    condition     = contains(["api", "worker"], var.workload_type)
    error_message = "Workload type must be api or worker."
  }
}
variable "cpu" { type = number }
variable "memory" { type = string }
variable "scaling" {
  type = object({ min_replicas = number, max_replicas = number, http_rule = optional(object({ name = string, concurrent_requests = number })), backlog_rule = optional(object({ name = string, custom_rule_type = string, metadata = map(string), authentication = optional(object({ secret_name = string, trigger_parameter = string })) })) })
  validation {
    condition     = var.scaling.min_replicas >= 0 && var.scaling.max_replicas >= var.scaling.min_replicas
    error_message = "Scaling replica bounds are invalid."
  }
}
variable "ingress" { type = object({ external_enabled = bool, target_port = number, transport = string, allow_insecure_connections = bool }) }
variable "health_paths" {
  type    = object({ live = string, ready = string })
  default = null
}
variable "secret_refs" {
  type      = list(object({ name = string, key_vault_secret_id = string, environment_variable = string }))
  sensitive = true
}
variable "tags" {
  type = map(string)
  validation {
    condition     = alltrue([for key in ["application", "environment", "owner", "cost_center", "managed_by", "data_classification"] : contains(keys(var.tags), key)]) && var.tags["managed_by"] == "terraform" && (var.tags["environment"] != "demo" || (contains(keys(var.tags), "expires_on") && can(regex("^[0-9]{4}-[0-9]{2}-[0-9]{2}$", var.tags["expires_on"]))))
    error_message = "Required Terraform tags are mandatory."
  }
}

check "workload_scaling" {
  assert {
    condition     = var.workload_type == "api" ? var.ingress != null && try(var.scaling.http_rule, null) != null : var.ingress == null && try(var.scaling.backlog_rule, null) != null
    error_message = "API workloads require ingress and HTTP scaling; worker workloads require backlog scaling and no ingress."
  }
}

check "api_ingress" {
  assert {
    condition     = var.workload_type != "api" || (var.ingress != null && var.health_paths != null && var.ingress.target_port == 8080 && var.ingress.transport == "http" && !var.ingress.allow_insecure_connections && var.health_paths.live == "/health/live" && var.health_paths.ready == "/health/ready")
    error_message = "API workloads must use secure HTTP ingress on port 8080 and the approved health paths."
  }
}
