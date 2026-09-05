terraform {
  required_version = ">= 1.8, < 2.0"
  required_providers {
    azuread = { source = "hashicorp/azuread", version = "~> 3.0" }
  }
}
