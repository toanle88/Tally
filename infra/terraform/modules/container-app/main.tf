locals { image = "${var.image.registry_login_server}/${var.image.repository}@${var.image.digest}" }
resource "azurerm_container_app" "this" {
  name                         = var.name
  resource_group_name          = var.resource_group_name
  container_app_environment_id = var.container_app_environment_id
  revision_mode                = "Single"
  tags                         = var.tags
  identity {
    type         = "UserAssigned"
    identity_ids = [var.identity_id]
  }
  registry {
    server   = var.image.registry_login_server
    identity = var.identity_id
  }
  template {
    min_replicas = var.scaling.min_replicas
    max_replicas = var.scaling.max_replicas
    container {
      name   = var.name
      image  = local.image
      cpu    = var.cpu
      memory = var.memory
      dynamic "env" {
        for_each = var.secret_refs
        content {
          name        = env.value.environment_variable
          secret_name = env.value.name
        }
      }
      dynamic "liveness_probe" {
        for_each = var.workload_type == "api" ? [var.health_paths] : []
        content {
          transport = "HTTP"
          port      = 8080
          path      = liveness_probe.value.live
        }
      }
      dynamic "readiness_probe" {
        for_each = var.workload_type == "api" ? [var.health_paths] : []
        content {
          transport = "HTTP"
          port      = 8080
          path      = readiness_probe.value.ready
        }
      }
    }
    dynamic "http_scale_rule" {
      for_each = try(var.scaling.http_rule, null) == null ? [] : [var.scaling.http_rule]
      content {
        name                = http_scale_rule.value.name
        concurrent_requests = http_scale_rule.value.concurrent_requests
      }
    }
    dynamic "custom_scale_rule" {
      for_each = try(var.scaling.backlog_rule, null) == null ? [] : [var.scaling.backlog_rule]
      content {
        name             = custom_scale_rule.value.name
        custom_rule_type = custom_scale_rule.value.custom_rule_type
        metadata         = custom_scale_rule.value.metadata
        dynamic "authentication" {
          for_each = try(custom_scale_rule.value.authentication, null) == null ? [] : [custom_scale_rule.value.authentication]
          content {
            secret_name       = authentication.value.secret_name
            trigger_parameter = authentication.value.trigger_parameter
          }
        }
      }
    }
  }
  dynamic "secret" {
    for_each = var.secret_refs
    content {
      name                = secret.value.name
      key_vault_secret_id = secret.value.key_vault_secret_id
      identity            = var.identity_id
    }
  }
  dynamic "ingress" {
    for_each = var.ingress == null ? [] : [var.ingress]
    content {
      external_enabled           = ingress.value.external_enabled
      target_port                = ingress.value.target_port
      transport                  = ingress.value.transport
      allow_insecure_connections = ingress.value.allow_insecure_connections
      traffic_weight {
        percentage      = 100
        latest_revision = true
      }
    }
  }
}
