# EP-IAC-001 — Terraform and Azure Learning Environment User Stories

## 1. Outcome

Provide a low-cost, repeatable Terraform and Azure learning environment for
TALLY. Local development remains the primary environment; Azure `dev` and
`demo` environments are optional exercises, and `prod-reference` is a design
reference rather than a production deployment claim.

## 2. Learning objective

Learn remote Terraform state, reusable Azure modules, workload identity,
managed services, cost controls, plan review, ephemeral deployment, and safe
destruction without introducing production infrastructure or credentials into
the repository.

## 3. Scope and ownership

`EP-IAC-001` belongs to M0 and contains:

- `DLV-IAC-001` — Terraform modules and local state bootstrap.
- `DLV-IAC-002` — optional Azure development/demo deployment.

Infrastructure owns resource provisioning and deployment configuration. It does
not own application behavior, finance data, authorization decisions, database
migrations, observability semantics, or release qualification.

## 4. Explicit exclusions

- No production deployment or production availability claim.
- No AKS, Redis, Kafka, API Management, premium data grids, or unnecessary
  persistent services.
- No real finance, payroll, tax, bank, credential, or personal data.
- No client secrets committed to source or Terraform state.
- No portal-only configuration that bypasses Terraform ownership.
- No replacement of application, database, identity, observability, or CI
  delivery items owned elsewhere.

## 5. Traceability

| Field | Value |
|---|---|
| Milestone | M0 |
| Parent epic | `EP-IAC-001` |
| Delivery items | `DLV-IAC-001`, `DLV-IAC-002` |
| Functional requirements | None; platform/environment scope |
| Global requirements | `GFR-001`, `GFR-012`, `GFR-014`, `GFR-015` as applicable |
| NFR areas | `NFR-SEC-*`, `NFR-PRV-*`, `NFR-OBS-*`, `NFR-REC-*`, `NFR-CMP-*`, `NFR-MNT-*` |
| Quality gates | `QG-01`, `QG-03`, `QG-04`, `QG-06`, `QG-08`, `QG-10` as applicable |

## 6. Dependencies

- `DLV-PLAT-001` supplies the repository and root command conventions.
- `DLV-PLAT-003` supplies persistence conventions; this epic provisions no
  finance schema or migration.
- `DLV-PLAT-004` supplies API contract artifacts; deployment must not invent
  routes or runtime behavior.
- `DLV-OPS-001` and `DLV-OPS-002` own application telemetry, dashboards, and
  runbooks; infrastructure only exposes the required platform integration.

## 7. Delivery item: DLV-IAC-001 — Terraform modules and local state bootstrap

### User Story 1 — Establish the Terraform repository boundary

**As the project owner, I want a predictable Terraform layout and pinned
provider/tooling contract, so that infrastructure changes are reviewable and
reproducible.**

- [ ] The repository contains the approved `infra/terraform/bootstrap`,
  `modules`, and `environments/dev`, `demo`, and `prod-reference` boundaries.
- [ ] Terraform requires `>= 1.8, < 2.0` and the approved AzureRM, AzureAD, and
  Random provider major versions.
- [ ] `terraform fmt -check` and `terraform validate` run for every applicable
  root module.
- [ ] Provider and module versions are constrained and lock files are reviewed.
- [ ] The layout does not contain credentials, generated build output, or
  application/domain code.

### User Story 2 — Bootstrap protected remote state

**As the project owner, I want remote state bootstrapped separately from
workload resources, so that state is locked, encrypted, access-controlled, and
isolated per environment.**

- [ ] Bootstrap creates the minimum Azure Blob state resources with encryption,
  restricted RBAC, and state locking.
- [ ] Bootstrap state is separate from workload state and can be initialized
  before the environment roots.
- [ ] Each environment uses a separate state key and identity; workspaces are
  not the sole isolation mechanism.
- [ ] Bootstrap instructions document prerequisites, plan review, recovery, and
  ownership without embedding secrets.
- [ ] State outputs and plans are treated as confidential and are not committed.

### User Story 3 — Implement reusable low-cost modules

**As an infrastructure learner, I want small modules with explicit inputs and
outputs, so that the Azure environment can be composed without hidden portal
configuration.**

- [ ] Modules cover resource group, monitoring/log analytics, container
  registry, Container Apps environment/app, Static Web Apps, PostgreSQL,
  Key Vault, managed identity, budget, and GitHub federation as applicable.
- [ ] Module interfaces match the approved technical specification and expose
  resource identifiers rather than secret values.
- [ ] Required naming and tags include application, environment, owner,
  cost-center, `managed_by=terraform`, data classification, and demo expiry.
- [ ] PostgreSQL is version 18, TLS is required, and environment-specific
  backup/network settings are explicit.
- [ ] Container images use immutable digests; ACR admin access is disabled and
  workload identity is used for pulls and secret references.

### User Story 4 — Define environment profiles safely

**As the project owner, I want dev, demo, and production-reference profiles to
be visibly different, so that a learning environment cannot be mistaken for a
production authorization.**

- [ ] `dev` uses the smallest suitable burstable and scale-to-zero learning
  profile.
- [ ] `demo` is disposable, uses synthetic data, and carries an expiry tag and
  destroy procedure.
- [ ] `prod-reference` prohibits casual destruction, uses private database
  access, and represents the approved reference topology without claiming
  qualification.
- [ ] Environment variables and sensitive inputs are supplied externally and
  validated without being written to source.
- [ ] Public database access is absent from the production-reference profile;
  learning-profile exceptions are explicit and restricted.

### User Story 5 — Verify plans, policy, drift, and cost controls

**As the project owner, I want infrastructure safety checks before apply, so
that drift, accidental exposure, and runaway learning cost are visible.**

- [ ] Formatting, validation, provider-lock, lint, security scanning, and
  policy checks are documented and executable.
- [ ] Plan checks detect public production-reference PostgreSQL, missing managed
  identity, missing diagnostics, invalid replica settings, and missing tags.
- [ ] Budgets alert at 50%, 80%, and 100% of the monthly learning budget.
- [ ] Drift detection is scheduled or documented at least daily for shared
  environments.
- [ ] A plan review records environment, commit, resource changes, estimated
  cost, data classification, and approval decision.

## 8. Delivery item: DLV-IAC-002 — Optional Azure dev/demo deployment

### User Story 6 — Deploy the optional learning environment

**As a learner, I want to deploy the application stack to Azure on demand, so
that I can practice cloud deployment without making Azure a local-development
dependency.**

- [ ] A documented command initializes, plans, and applies the selected Azure
  environment after explicit confirmation.
- [ ] The stack follows the approved learning topology: Static Web Apps, a
  Container Apps API, PostgreSQL Flexible Server, Key Vault, managed identity,
  ACR, and monitoring as applicable.
- [ ] Application endpoints and health checks use existing repository behavior;
  no finance capability is invented by the deployment.
- [ ] Synthetic, non-sensitive data is the only permitted deployment data.
- [ ] Apply output identifies created resources and safe next steps without
  printing secrets.

### User Story 7 — Federate CI/CD without long-lived secrets

**As the project owner, I want GitHub Actions to authenticate through OIDC,
so that deployment does not depend on stored Azure client secrets.**

- [ ] Pull requests run Terraform format, validation, security checks, and plan.
- [ ] Protected environments have separate federated identity subjects.
- [ ] Apply is restricted to protected branches and required approval.
- [ ] State and plan artifacts are access-controlled and retained according to
  the approved policy.
- [ ] Federation subjects are least-privilege and repository/environment
  specific.

### User Story 8 — Smoke-test deployment readiness

**As a release learner, I want a deployment smoke test, so that a successful
Terraform apply is not confused with a usable environment.**

- [ ] Smoke tests verify expected resource outputs, API reachability, health
  endpoints, TLS, identity access, image pull, and required diagnostics.
- [ ] A failed smoke test returns a failure result and identifies the owning
  resource or dependency.
- [ ] Database migrations run through the approved controlled release path, not
  once per application replica.
- [ ] No smoke test establishes finance state or uses production data.
- [ ] Results record commit, environment, resource versions, test outcome, and
  cleanup status.

### User Story 9 — Destroy disposable resources safely

**As the project owner, I want demo resources to be easy to destroy, so that
  Azure cost and abandoned-resource risk remain bounded.**

- [ ] The demo environment has a documented, repeatable destroy command.
- [ ] Destruction requires an explicit environment and confirmation and cannot
  target local or unrelated resources.
- [ ] Nonlocal database destruction performs a final backup/export check first.
- [ ] Destroy verification confirms resources are gone and no secret or
  state-lock cleanup was skipped.
- [ ] The dev environment is not destroyed implicitly by demo cleanup.

### User Story 10 — Recover state and document operations

**As the project owner, I want state and deployment recovery instructions, so
that an infrastructure exercise can be repeated after failure.**

- [ ] Runbooks cover state-lock recovery, drift, failed apply, failed destroy,
  credential/identity rotation, and resource-group recovery.
- [ ] Recovery instructions preserve Terraform ownership and do not recommend
  unreviewed portal edits as the normal path.
- [ ] Backup/restore and recovery exercises identify data, access, retention,
  and reconciliation obligations.
- [ ] Operational evidence includes owner, timestamp, environment, command,
  result, and unresolved exception.
- [ ] The documentation distinguishes learning-profile limitations from
  production-reference requirements.

## 9. Epic acceptance summary

- [ ] Both delivery items and all ten stories pass their applicable criteria.
- [ ] `fmt`, `validate`, lint, security, policy, provider-lock, and plan checks
  pass without committed credentials.
- [ ] Dev and demo plans are reviewable, cost-bounded, tagged, and disposable.
- [ ] The production-reference plan has private database access, managed
  identity, diagnostics, and no unsupported availability claim.
- [ ] Optional Azure deployment, smoke test, and destroy are repeatable with
  synthetic data.
- [ ] State, plan, deployment, drift, cost, and recovery evidence is recorded.
- [ ] No application, finance-domain, database-migration, or authorization
  ownership boundary is bypassed.

## 10. Required evidence

- Terraform version and provider lock evidence.
- `terraform fmt -check`, `validate`, lint, security, and policy output.
- Reviewed plans for `dev`, `demo`, and `prod-reference`.
- Budget, tags, managed identity, network, diagnostics, and image-digest
  assertions.
- Optional apply/smoke/destroy evidence for the Azure demo environment.
- Drift and state-recovery evidence.
- Cost/resource inventory and cleanup confirmation.

## 11. Definition of Ready

- [ ] Repository and root command conventions are available.
- [ ] The approved Terraform technical specification and deployment architecture
  remain unchanged or their impact is recorded.
- [ ] An Azure subscription, region, budget, and owner are available for an
  optional exercise; local development does not depend on them.
- [ ] No production credentials or sensitive data are required.

## 12. Definition of Done

- [ ] All child stories pass review with evidence attached.
- [ ] No critical/high infrastructure or security defect remains unresolved.
- [ ] Azure cost, expiry, destroy, and state-recovery controls are documented.
- [ ] Terraform owns declared resources and drift is detectable.
- [ ] Architectural, security, privacy, and data-boundary rules are preserved.

## 13. Source references

- `docs/specs/finance_delivery_plan_v1.0.md`
- `docs/specs/system_design/04_security_deployment_operations_v1.0.md`
- `docs/specs/technical_specifications/07_terraform_azure_deployment_specifications_v1.0.md`
- `docs/specs/finance_nonfunctional_requirements_v1.0.md`
- `docs/backlog/epic-template.md`
- `docs/backlog/story-template.md`
