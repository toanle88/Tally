# DLV-IAC-001 User Story 1 — Terraform Repository Boundary Verification

## Scope

This record verifies the repository boundary and provider contract for the four
Terraform roots introduced by User Story 1:

- `infra/terraform/bootstrap`
- `infra/terraform/environments/dev`
- `infra/terraform/environments/demo`
- `infra/terraform/environments/prod-reference`

The slice intentionally does not create Azure resources, leaf modules, remote
state storage, environment wiring, plans, state files, or credentials.

## Verification evidence

Executed on branch `feat/dlv-iac-001-us1-terraform-repository-boundary`:

| Command | Result |
| --- | --- |
| `terraform version` | Terraform `1.15.2` |
| `bash -n scripts/verify/terraform.sh` | Passed |
| `make terraform-check` | Passed for all four roots |
| `make check` | Passed: migrations and Go tests |
| `pnpm docs:check` | Passed: VitePress build |
| `git diff --check` | Passed |

The Terraform gate also verified the required provider sources and constraints,
the `azurerm` backend declaration, root lockfiles, valid configuration, no
resource/data/module blocks, no credential assignments, and no forbidden
Terraform artifacts.

Locked provider versions are `hashicorp/azurerm` `4.81.0`,
`hashicorp/azuread` `3.9.0`, and `hashicorp/random` `3.9.0` in each root.

## Deferred items

Remote-state bootstrap and leaf-module implementation remain deferred to the
later User Stories in DLV-IAC-001. The approved system-design and technical
specification documents currently use different leaf-module names, so this
slice does not invent a module naming resolution.
