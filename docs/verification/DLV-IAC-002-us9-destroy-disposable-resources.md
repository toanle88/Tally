# DLV-IAC-002 User Story 9 — Destroy disposable resources safely

## Scope and evidence boundary

This record covers the repository-side implementation for EP-IAC-001 User
Story 9. It provides a guarded Terraform destroy workflow for an explicitly
selected Azure dev or demo environment. The implementation is credential-free
in repository tests; it does not claim that an authenticated Azure demo
deployment, backup, destroy, or post-destroy verification has been witnessed.
That live evidence remains an operator acceptance activity and must use
synthetic data.

Traceability:

M0 → EP-IAC-001 → DLV-IAC-002 → User Story 9

Controls: ARC-IAC-001, ARC-CICD-001, GFR-014, GFR-015, NFR-SEC-010,
NFR-PRV-010, NFR-MNT-007, NFR-MNT-008, NFR-OBS-004, QG-01, QG-06, QG-08,
QG-10.

## Repository implementation

The supported operator interface is:

~~~text
CONFIRM_DESTROY=demo ENVIRONMENT=demo make azure-learning-destroy
make azure-learning-destroy-check
~~~

The destroy wrapper:

- accepts only dev or demo and derives the Terraform root and Azure Blob state
  container from that single selected environment;
- requires CONFIRM_DESTROY to match before Azure subscription or remote-state
  operations;
- verifies the Azure CLI subscription, Terraform environment_profile,
  resource-group ID/name, and PostgreSQL server ID/name when present;
- creates a temporary terraform plan -destroy, inspects its actions, and shows
  only a value-free resource-address summary before a second interactive
  environment-name confirmation;
- checks the installed Azure CLI help for backup create --name and fails
  closed if that option is unavailable;
- creates and verifies a uniquely named Azure-managed PostgreSQL on-demand
  backup before applying the exact temporary destroy plan when the selected
  state still contains the PostgreSQL server;
- applies only the generated plan file, then verifies resource-group absence,
  empty Terraform state, and a follow-up Terraform operation that acquires and
  releases the remote state lock;
- removes temporary plans, logs, state listings, and post-destroy plan files
  on exit and writes only a redacted status report under
  artifacts/deployment-destroy/<environment>.json.

The wrapper does not use az group delete, terraform force-unlock, manual Blob
lock deletion, Docker, bootstrap state, or another environment root. A demo
destroy cannot implicitly destroy dev.

## Azure backup command discrepancy

The current Azure CLI reference documents the flexible-server backup create and
show commands with the --name form:
[Azure CLI reference](https://learn.microsoft.com/en-us/cli/azure/postgres/flexible-server/backup?view=azure-cli-latest).
Microsoft's backup guidance currently shows --backup-name:
[backup guidance](https://learn.microsoft.com/en-us/azure/postgresql/backup-restore/how-to-perform-backups).
The wrapper does not guess between these forms. It checks the installed CLI
help for --name and refuses to continue when the contract is not exposed.
The operator must resolve any CLI version mismatch explicitly before a live
destroy.

## Credential-free repository evidence

Run:

~~~text
bash scripts/deploy/azure-learning.sh --self-test
bash scripts/deploy/azure-learning-destroy.test.sh
make azure-learning-destroy-check
make terraform-check terraform-environments-check terraform-modules-check terraform-ci-check
terraform fmt -check -recursive infra/terraform
git diff --check
~~~

The destroy contract test uses fake Terraform and Azure CLI executables. It
covers invalid environment, missing confirmation, subscription mismatch,
unrelated resource-group identity, backup failure, unsafe destroy-plan action,
partial rerun without a remaining PostgreSQL state entry, dev/demo scope
isolation, secret-like output checks, resource-group absence, empty state, and
lock-release verification. It does not contact Azure or prove remote-state
permissions.

## Live acceptance evidence template

Complete this section only after an authenticated disposable demo run. Keep
live evidence separate from the repository contract-test result. Do not paste
credentials, passwords, tokens, connection strings, raw Terraform state,
Terraform plans, or raw Azure error payloads.

~~~text
Story: User Story 9 / DLV-IAC-002
Commit:
Environment: demo
Operator:
UTC start/end:
Synthetic-data deployment reference:
Backup result: pass/fail/not-applicable
Backup identifier (sanitized):
Destroy result: pass/fail
Resource-group absence: pass/fail
Terraform state empty: pass/fail
Remote state lock acquire/release: pass/fail
Temporary artifact cleanup: pass/fail
Unresolved exceptions:
Evidence locations:
~~~

## Acceptance status

Repository implementation and credential-free evidence are complete. The
story remains open until one authenticated disposable demo deployment is
backed up, destroyed, and verified with preserved redacted Azure evidence.
