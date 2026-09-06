# DLV-IAC-001 User Story 4 — Environment Profile Verification

Status: implemented locally on `feat/dlv-iac-001-us4-environment-profiles`; live Azure plans remain an external prerequisite.

## Scope and source basis

This record covers the three environment roots required by User Story 4:

- `infra/terraform/environments/dev`
- `infra/terraform/environments/demo`
- `infra/terraform/environments/prod-reference`

The profile composition follows TS07 sections 3–7 and the deployment controls in
the system design. It does not claim production qualification, Azure apply, or
the Story 5 budget/policy/drift gates.

## Profile controls

| Profile | Workloads | PostgreSQL | Network/deletion control |
|---|---|---|---|
| dev | API and worker `0/1`, `0.5` vCPU, `1GiB` | v18, `B_Standard_B1ms`, 32GiB, seven-day backup | Public access requires explicit single-host rules; no expiry or deletion lock |
| demo | Same low-cost profile, synthetic data only | Same disposable PostgreSQL profile | Expiry tag is mandatory; cleanup is restricted to the demo state key |
| prod-reference | API minimum two, independent worker minimum one | Measured SKU/storage, HA, 35-day geo-redundant backup | Private PostgreSQL, private Container Apps subnet, deny-by-default Key Vault, `CanNotDelete` lock with `prevent_destroy` |

All roots use names based on `<organization>-fin-<environment>-<region>-<resource>-<nn>`.
Azure resource-name limits require the `prod-reference` physical-name abbreviation
`pr`; the full profile name remains in tags, state keys, outputs, and checks.

All roots use immutable image digests, managed identities, diagnostics, required
ownership/classification tags, and a repository/token-free Static Web App. The
PostgreSQL administrator password is a sensitive write-only external input.
The profile roots do not configure Static Web App application deployment or a
private Static Web App-to-Container Apps edge path; `prod-reference` therefore
remains a non-deployable topology reference until that integration is delivered
in the later deployment scope.

Learning firewall rules are validated as explicit single-host IPv4 entries. The
worker Prometheus scaler contract requires `serverAddress`, `query`, and
`threshold` metadata; live values remain external inputs. The production
PostgreSQL firewall set is empty and its network mode is private.

## Verification commands

```text
node scripts/verify/terraform-environments-contract.js
make terraform-environments-check
make terraform-check
make terraform-modules-check
```

The environment contract checks module inventory, profile constants, outputs,
learning firewall restrictions, demo expiry, token-free Static Web Apps, worker
scaling inputs, and the production deletion lock. The existing Terraform gate
continues to verify provider constraints, lockfiles, state keys, bootstrap
boundaries, secret hygiene, and forbidden artifacts.

Provider-backed initialization, validation, and Terraform tests require a
networked Terraform Registry/provider environment. A real `terraform plan` for
each root also requires the bootstrapped remote state and externally supplied
non-secret profile inputs plus the sensitive write-only password. Until those
prerequisites are available, this record must not claim live plan completion.

## Demo cleanup procedure

1. Run from the repository root and select only `infra/terraform/environments/demo`.
2. Confirm the backend key is `demo/terraform.tfstate`.
3. Confirm the expiry tag and synthetic-data-only status.
4. Export or back up any disposable demo database data required for the exercise.
5. Run and review `terraform plan -destroy` for the demo root.
6. Require explicit confirmation, then run `terraform destroy` for the demo root only.
7. Verify the demo resource group and resources are gone.
8. Do not reuse the command or backend selection for `dev` or `prod-reference`.
