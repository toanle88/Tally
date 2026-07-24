# Finance Platform Risks, Costs, Governance and Traceability

| Document-control field | Value |
|---|---|
| Version | 1.0 |
| Baseline date | 2026-07-24 |
| Status | Passed |
| Source baseline | Finance DDD v3.1; Functional PRD v1.5; UX v1.0; NFR v1.0; Solution/System Design v1.0; Technical Specifications v1.0 |
| Delivery profile | Solo, part-time learning project; local-first; low-cost Azure demonstrations |
| Owner | Learning Product and Delivery Owner |

> **Purpose:** Define delivery governance, conceptual ownership, risk and cost controls, exact source-to-delivery traceability and the final review checkpoint.
>
> **Planning rule:** Iteration counts and elapsed-time ranges are planning assumptions, not commitments. Reforecast after M2 using observed completion rate, defect rate and available learning time.

## 1. Governance model

One learner may perform several roles, but decisions and reviews are recorded using distinct conceptual responsibilities.

| Role | Responsibilities | May also implement? | Required independent perspective |
|---|---|---|---|
| Product owner | Scope, priority, milestone outcome and acceptance | Yes | Challenges whether work delivers user value |
| Domain owner | Accounting semantics, invariants and terminology | Yes | Reviews financial correctness separately from implementation convenience |
| Architecture owner | Boundaries, dependencies, ADRs and NFR fit | Yes | Challenges unnecessary complexity |
| Backend owner | Go, domain/application code, API and workers | Yes | Peer/self-review checklist |
| Frontend owner | React, Tailwind/daisyUI, UX behavior and accessibility | Yes | Manual user and accessibility perspective |
| Database owner | PostgreSQL schema, migrations, queries and recovery | Yes | Reconciliation and restore perspective |
| Security/privacy owner | Identity, authorization, SoD, secrets and sensitive data | Yes | Threat-based review |
| QA owner | Test design, acceptance, defects and release evidence | Yes | Attempts to falsify success criteria |
| Operations/cloud owner | Terraform, Azure, observability, backup and cost | Yes | Operational exercise evidence |

## 2. Decision and change control

- Architecture or technology changes require an ADR.
- Requirement, workflow, NFR or acceptance changes update their authoritative source before delivery mappings.
- Milestone scope changes record impact on dependencies, dates, cost, risk and tests.
- Deferrals identify the exact IDs, reason, consequence and next review point.
- A milestone cannot conceal unsatisfied scope under a generic “future enhancement” label.

## 3. Risk register

| Risk | Description | Probability | Impact | Mitigation | Owner |
|---|---|---|---|---|---|
| RISK-001 | Scope exceeds a solo learning project | High | High | Use milestone cut lines; pause after M2, M4 or M6 if learning objectives are met; do not begin a phase without its dependencies. | Product owner |
| RISK-002 | Architecture becomes a distributed system prematurely | Medium | High | Keep one deployable modular monolith and one PostgreSQL database; require an ADR before extracting a service. | Architecture owner |
| RISK-003 | Financial invariants are implemented inconsistently | Medium | Critical | Centralize value objects and invariant tests within owning modules; require QG-02 and QG-07. | Domain owner |
| RISK-004 | Cross-module database access bypasses ownership | Medium | High | Schema ownership, role grants, static dependency checks and repository boundary tests. | Database owner |
| RISK-005 | Azure costs exceed the learning budget | Medium | Medium | Local-first development, scale-to-zero, ephemeral demo environments, Terraform destroy plans, budget alerts and monthly review. | Cloud owner |
| RISK-006 | Authentication setup blocks domain learning | Medium | Medium | Provide local development identity mode with production-equivalent claims; integrate Entra incrementally before M1 exit. | Security owner |
| RISK-007 | UI breadth delays complete workflows | High | Medium | Build shared worklist/detail/action patterns; deliver only screens required for the current vertical slice. | UX owner |
| RISK-008 | Concurrency defects remain hidden in happy-path tests | Medium | Critical | Testcontainers concurrency suites, deterministic barriers, repeated race tests and QG-07 before milestone exit. | QA owner |
| RISK-009 | Reporting queries degrade transaction processing | Medium | High | Use projections, bounded queries, read replicas only when justified, and performance tests before M8. | Reporting owner |
| RISK-010 | Generated API, SQL or Terraform artifacts drift | Medium | Medium | CI regeneration checks, format/validate gates and committed generated contracts where appropriate. | Engineering owner |
| RISK-011 | NFR baseline is unrealistic for the learning deployment | High | Medium | Separate functional learning acceptance from production qualification; document unexecuted qualification as unsatisfied, not silently passed. | Architecture owner |
| RISK-012 | Seed/demo data does not represent financial edge cases | Medium | High | Versioned scenario datasets covering currencies, periods, duplicates, corrections, failures and concurrent changes. | QA owner |
| RISK-013 | Sensitive information appears in logs or demo data | Low | High | Synthetic data only, log redaction tests, secret scanning and QG-06. | Security owner |
| RISK-014 | Long-running workflows lack recoverability | Medium | High | Durable state, outbox/inbox, restart tests, reconciliation worklists and runbooks. | Operations owner |
| RISK-015 | Delivery dates are treated as commitments | Medium | Medium | Label all dates as planning assumptions; reforecast after M2 using measured throughput. | Product owner |
| RISK-016 | Technical specifications are too broad to implement safely in one change | High | High | One vertical slice per pull request chain; feature flags and small migrations; maximum reviewable change limits. | Engineering owner |
| RISK-017 | Accessibility is deferred until late | Medium | High | Shared accessible components in M0; automated checks every change; manual screen-reader gates at milestone demos. | UX owner |
| RISK-018 | Backups exist but restoration is unproven | Medium | Critical | Scheduled restore exercises and financial reconciliation after restore before M9. | Operations owner |

## 4. Cost-control plan

The thresholds below are governance choices, not Azure price guarantees. Review current Azure pricing before every material provisioning change.

| Control | Rule |
|---|---|
| COST-001 — Local-first default | Daily development uses Docker Compose and local PostgreSQL; Azure is not required for feature work. |
| COST-002 — Learning budget alert | Configure an Azure monthly budget alert at USD 20 or the learner-selected local-currency equivalent. |
| COST-003 — Hard review threshold | Do not approve a Terraform apply expected to make the active-month total exceed USD 50 without explicit review. |
| COST-004 — Ephemeral demo environment | Create the demo environment only for deployment/recovery exercises and destroy it after the exercise. |
| COST-005 — Scale-to-zero | Use minimum replicas zero for eligible learning workloads and one maximum replica until a qualification exercise requires more. |
| COST-006 — Database discipline | Use local PostgreSQL normally; provision the smallest suitable Azure PostgreSQL profile only for Azure-specific exercises. |
| COST-007 — No premium platform by default | Do not add AKS, Kafka, Redis, Elasticsearch, API Management, service mesh or paid UI grids without an approved ADR and cost impact. |
| COST-008 — Pre-apply review | Terraform plan must include a human-readable resource and cost-impact summary before apply. |
| COST-009 — Tagging | Every Azure resource carries project, environment, owner, expiry and cost-center tags. |
| COST-010 — Monthly reconciliation | Review actual Azure charges, orphaned resources and budget alerts at least monthly while Azure resources exist. |

## 5. Progress reporting

| Metric | Definition | Anti-pattern avoided |
|---|---|---|
| Milestone health | Passed gates, unresolved blockers and forecast | Percent-complete optimism |
| Delivered requirements | Exact `FR-*` and `GFR-*` items meeting DoD | Counting started stories |
| Workflow evidence | Workflows passing mapped acceptance packs | UI-only demos |
| Defect escape | Defects found after milestone demo | Hiding rework inside velocity |
| Rework ratio | Time spent correcting previously accepted work | Inflated throughput |
| NFR qualification | Passed, failed or not-yet-qualified exact NFRs | Treating documents as execution evidence |
| Cloud spend | Actual charges, forecast and orphaned resources | Unbounded learning cost |
| Learning outcome | Decisions, patterns and failure modes understood | Code volume as learning measure |

## 6. Exact functional requirement traceability

| Requirement | Delivery item | Epic | Milestone |
|---|---|---|---|
| FR-AP-001 | DLV-FR-AP-001 | EP-AP-001 | M5 |
| FR-AP-002 | DLV-FR-AP-002 | EP-AP-001 | M7 |
| FR-AP-003 | DLV-FR-AP-003 | EP-AP-001 | M6 |
| FR-AP-004 | DLV-FR-AP-004 | EP-AP-001 | M6 |
| FR-AP-005 | DLV-FR-AP-005 | EP-AP-001 | M5 |
| FR-AP-006 | DLV-FR-AP-006 | EP-AP-001 | M5 |
| FR-AP-007 | DLV-FR-AP-007 | EP-AP-001 | M5 |
| FR-AP-008 | DLV-FR-AP-008 | EP-AP-001 | M5 |
| FR-AP-009 | DLV-FR-AP-009 | EP-AP-001 | M5 |
| FR-AP-010 | DLV-FR-AP-010 | EP-AP-001 | M5 |
| FR-AR-001 | DLV-FR-AR-001 | EP-AR-001 | M4 |
| FR-AR-002 | DLV-FR-AR-002 | EP-AR-001 | M4 |
| FR-AR-003 | DLV-FR-AR-003 | EP-AR-001 | M4 |
| FR-AR-004 | DLV-FR-AR-004 | EP-AR-001 | M4 |
| FR-AR-005 | DLV-FR-AR-005 | EP-AR-001 | M4 |
| FR-AR-006 | DLV-FR-AR-006 | EP-AR-001 | M4 |
| FR-AR-007 | DLV-FR-AR-007 | EP-AR-001 | M4 |
| FR-AR-008 | DLV-FR-AR-008 | EP-AR-001 | M4 |
| FR-AR-009 | DLV-FR-AR-009 | EP-AR-001 | M4 |
| FR-AR-010 | DLV-FR-AR-010 | EP-AR-001 | M5 |
| FR-AR-011 | DLV-FR-AR-011 | EP-AR-001 | M5 |
| FR-AR-012 | DLV-FR-AR-012 | EP-AR-001 | M5 |
| FR-AR-013 | DLV-FR-AR-013 | EP-AR-001 | M5 |
| FR-AR-014 | DLV-FR-AR-014 | EP-AR-001 | M5 |
| FR-AR-015 | DLV-FR-AR-015 | EP-AR-001 | M6 |
| FR-AR-016 | DLV-FR-AR-016 | EP-AR-001 | M4 |
| FR-AUD-001 | DLV-FR-AUD-001 | EP-AUD-001 | M1 |
| FR-AUD-002 | DLV-FR-AUD-002 | EP-AUD-001 | M9 |
| FR-AUD-003 | DLV-FR-AUD-003 | EP-AUD-001 | M9 |
| FR-AUD-004 | DLV-FR-AUD-004 | EP-AUD-001 | M9 |
| FR-AUD-005 | DLV-FR-AUD-005 | EP-AUD-001 | M9 |
| FR-BFR-001 | DLV-FR-BFR-001 | EP-BFR-001 | M6 |
| FR-BFR-002 | DLV-FR-BFR-002 | EP-BFR-001 | M6 |
| FR-BFR-003 | DLV-FR-BFR-003 | EP-BFR-001 | M6 |
| FR-BFR-004 | DLV-FR-BFR-004 | EP-BFR-001 | M6 |
| FR-BFR-005 | DLV-FR-BFR-005 | EP-BFR-001 | M6 |
| FR-BFR-006 | DLV-FR-BFR-006 | EP-BFR-001 | M6 |
| FR-COA-001 | DLV-FR-COA-001 | EP-COA-001 | M1 |
| FR-COA-002 | DLV-FR-COA-002 | EP-COA-001 | M1 |
| FR-COA-003 | DLV-FR-COA-003 | EP-COA-001 | M1 |
| FR-COA-004 | DLV-FR-COA-004 | EP-COA-001 | M1 |
| FR-COA-005 | DLV-FR-COA-005 | EP-COA-001 | M1 |
| FR-FA-001 | DLV-FR-FA-001 | EP-FA-001 | M7 |
| FR-FA-002 | DLV-FR-FA-002 | EP-FA-001 | M7 |
| FR-FA-003 | DLV-FR-FA-003 | EP-FA-001 | M7 |
| FR-FA-004 | DLV-FR-FA-004 | EP-FA-001 | M7 |
| FR-FA-005 | DLV-FR-FA-005 | EP-FA-001 | M7 |
| FR-FA-006 | DLV-FR-FA-006 | EP-FA-001 | M7 |
| FR-FA-007 | DLV-FR-FA-007 | EP-FA-001 | M7 |
| FR-FA-008 | DLV-FR-FA-008 | EP-FA-001 | M7 |
| FR-FA-009 | DLV-FR-FA-009 | EP-FA-001 | M7 |
| FR-FA-010 | DLV-FR-FA-010 | EP-FA-001 | M7 |
| FR-FA-011 | DLV-FR-FA-011 | EP-FA-001 | M7 |
| FR-FA-012 | DLV-FR-FA-012 | EP-FA-001 | M7 |
| FR-FA-013 | DLV-FR-FA-013 | EP-FA-001 | M7 |
| FR-FA-014 | DLV-FR-FA-014 | EP-FA-001 | M7 |
| FR-FA-015 | DLV-FR-FA-015 | EP-FA-001 | M7 |
| FR-FA-016 | DLV-FR-FA-016 | EP-FA-001 | M7 |
| FR-FA-017 | DLV-FR-FA-017 | EP-FA-001 | M7 |
| FR-FA-018 | DLV-FR-FA-018 | EP-FA-001 | M7 |
| FR-FA-019 | DLV-FR-FA-019 | EP-FA-001 | M7 |
| FR-FA-020 | DLV-FR-FA-020 | EP-FA-001 | M7 |
| FR-FA-021 | DLV-FR-FA-021 | EP-FA-001 | M7 |
| FR-FPM-001 | DLV-FR-FPM-001 | EP-FPM-001 | M3 |
| FR-FPM-002 | DLV-FR-FPM-002 | EP-FPM-001 | M3 |
| FR-FPM-003 | DLV-FR-FPM-003 | EP-FPM-001 | M3 |
| FR-FPM-004 | DLV-FR-FPM-004 | EP-FPM-001 | M3 |
| FR-FPM-005 | DLV-FR-FPM-005 | EP-FPM-001 | M3 |
| FR-FPM-006 | DLV-FR-FPM-006 | EP-FPM-001 | M3 |
| FR-FPM-007 | DLV-FR-FPM-007 | EP-FPM-001 | M3 |
| FR-FPM-008 | DLV-FR-FPM-008 | EP-FPM-001 | M3 |
| FR-FPM-009 | DLV-FR-FPM-009 | EP-FPM-001 | M3 |
| FR-FPM-010 | DLV-FR-FPM-010 | EP-FPM-001 | M3 |
| FR-FPM-011 | DLV-FR-FPM-011 | EP-FPM-001 | M3 |
| FR-FPM-012 | DLV-FR-FPM-012 | EP-FPM-001 | M3 |
| FR-FPM-013 | DLV-FR-FPM-013 | EP-FPM-001 | M3 |
| FR-FX-001 | DLV-FR-FX-001 | EP-FX-001 | M8 |
| FR-FX-002 | DLV-FR-FX-002 | EP-FX-001 | M8 |
| FR-FX-003 | DLV-FR-FX-003 | EP-FX-001 | M8 |
| FR-FX-004 | DLV-FR-FX-004 | EP-FX-001 | M8 |
| FR-FX-005 | DLV-FR-FX-005 | EP-FX-001 | M8 |
| FR-GL-001 | DLV-FR-GL-001 | EP-GL-001 | M2 |
| FR-GL-002 | DLV-FR-GL-002 | EP-GL-001 | M2 |
| FR-GL-003 | DLV-FR-GL-003 | EP-GL-001 | M2 |
| FR-GL-004 | DLV-FR-GL-004 | EP-GL-001 | M2 |
| FR-GL-005 | DLV-FR-GL-005 | EP-GL-001 | M2 |
| FR-GL-006 | DLV-FR-GL-006 | EP-GL-001 | M3 |
| FR-GL-007 | DLV-FR-GL-007 | EP-GL-001 | M3 |
| FR-GL-008 | DLV-FR-GL-008 | EP-GL-001 | M3 |
| FR-GL-009 | DLV-FR-GL-009 | EP-GL-001 | M3 |
| FR-GL-010 | DLV-FR-GL-010 | EP-GL-001 | M3 |
| FR-GL-011 | DLV-FR-GL-011 | EP-GL-001 | M3 |
| FR-GL-012 | DLV-FR-GL-012 | EP-GL-001 | M3 |
| FR-GL-013 | DLV-FR-GL-013 | EP-GL-001 | M3 |
| FR-GL-014 | DLV-FR-GL-014 | EP-GL-001 | M3 |
| FR-GL-015 | DLV-FR-GL-015 | EP-GL-001 | M1 |
| FR-GL-016 | DLV-FR-GL-016 | EP-GL-001 | M1 |
| FR-GL-017 | DLV-FR-GL-017 | EP-GL-001 | M1 |
| FR-GL-018 | DLV-FR-GL-018 | EP-GL-001 | M1 |
| FR-IAM-001 | DLV-FR-IAM-001 | EP-IAM-001 | M1 |
| FR-IAM-002 | DLV-FR-IAM-002 | EP-IAM-001 | M1 |
| FR-IAM-003 | DLV-FR-IAM-003 | EP-IAM-001 | M1 |
| FR-IAM-004 | DLV-FR-IAM-004 | EP-IAM-001 | M1 |
| FR-IAM-005 | DLV-FR-IAM-005 | EP-IAM-001 | M1 |
| FR-IAM-006 | DLV-FR-IAM-006 | EP-IAM-001 | M1 |
| FR-IC-001 | DLV-FR-IC-001 | EP-IC-001 | M8 |
| FR-IC-002 | DLV-FR-IC-002 | EP-IC-001 | M8 |
| FR-IC-003 | DLV-FR-IC-003 | EP-IC-001 | M8 |
| FR-IC-004 | DLV-FR-IC-004 | EP-IC-001 | M8 |
| FR-IC-005 | DLV-FR-IC-005 | EP-IC-001 | M8 |
| FR-IC-006 | DLV-FR-IC-006 | EP-IC-001 | M8 |
| FR-IC-007 | DLV-FR-IC-007 | EP-IC-001 | M8 |
| FR-IC-008 | DLV-FR-IC-008 | EP-IC-001 | M8 |
| FR-IC-009 | DLV-FR-IC-009 | EP-IC-001 | M8 |
| FR-IC-010 | DLV-FR-IC-010 | EP-IC-001 | M8 |
| FR-IC-011 | DLV-FR-IC-011 | EP-IC-001 | M8 |
| FR-INV-001 | DLV-FR-INV-001 | EP-INV-001 | M4 |
| FR-INV-002 | DLV-FR-INV-002 | EP-INV-001 | M4 |
| FR-INV-003 | DLV-FR-INV-003 | EP-INV-001 | M4 |
| FR-INV-004 | DLV-FR-INV-004 | EP-INV-001 | M4 |
| FR-INV-005 | DLV-FR-INV-005 | EP-INV-001 | M4 |
| FR-INV-006 | DLV-FR-INV-006 | EP-INV-001 | M4 |
| FR-OMD-001 | DLV-FR-OMD-001 | EP-OMD-001 | M1 |
| FR-OMD-002 | DLV-FR-OMD-002 | EP-OMD-001 | M1 |
| FR-OMD-003 | DLV-FR-OMD-003 | EP-OMD-001 | M1 |
| FR-OMD-004 | DLV-FR-OMD-004 | EP-OMD-001 | M1 |
| FR-OMD-005 | DLV-FR-OMD-005 | EP-OMD-001 | M1 |
| FR-OMD-006 | DLV-FR-OMD-006 | EP-OMD-001 | M1 |
| FR-PAYR-001 | DLV-FR-PAYR-001 | EP-PAYR-001 | M9 |
| FR-PAYR-002 | DLV-FR-PAYR-002 | EP-PAYR-001 | M9 |
| FR-PAYR-003 | DLV-FR-PAYR-003 | EP-PAYR-001 | M9 |
| FR-PAYR-004 | DLV-FR-PAYR-004 | EP-PAYR-001 | M9 |
| FR-PAYR-005 | DLV-FR-PAYR-005 | EP-PAYR-001 | M9 |
| FR-PAYR-006 | DLV-FR-PAYR-006 | EP-PAYR-001 | M9 |
| FR-PAYR-007 | DLV-FR-PAYR-007 | EP-PAYR-001 | M9 |
| FR-PCM-001 | DLV-FR-PCM-001 | EP-PCM-001 | M5 |
| FR-PCM-002 | DLV-FR-PCM-002 | EP-PCM-001 | M5 |
| FR-PCM-003 | DLV-FR-PCM-003 | EP-PCM-001 | M5 |
| FR-PCM-004 | DLV-FR-PCM-004 | EP-PCM-001 | M5 |
| FR-PCM-005 | DLV-FR-PCM-005 | EP-PCM-001 | M5 |
| FR-PCM-006 | DLV-FR-PCM-006 | EP-PCM-001 | M5 |
| FR-PCM-007 | DLV-FR-PCM-007 | EP-PCM-001 | M5 |
| FR-PCM-008 | DLV-FR-PCM-008 | EP-PCM-001 | M5 |
| FR-PCM-009 | DLV-FR-PCM-009 | EP-PCM-001 | M5 |
| FR-PCM-010 | DLV-FR-PCM-010 | EP-PCM-001 | M5 |
| FR-PCM-011 | DLV-FR-PCM-011 | EP-PCM-001 | M5 |
| FR-PCM-012 | DLV-FR-PCM-012 | EP-PCM-001 | M5 |
| FR-PCM-013 | DLV-FR-PCM-013 | EP-PCM-001 | M5 |
| FR-PCM-014 | DLV-FR-PCM-014 | EP-PCM-001 | M5 |
| FR-PCM-015 | DLV-FR-PCM-015 | EP-PCM-001 | M5 |
| FR-PCM-016 | DLV-FR-PCM-016 | EP-PCM-001 | M5 |
| FR-PCM-017 | DLV-FR-PCM-017 | EP-PCM-001 | M6 |
| FR-PCM-018 | DLV-FR-PCM-018 | EP-PCM-001 | M6 |
| FR-PCM-019 | DLV-FR-PCM-019 | EP-PCM-001 | M6 |
| FR-PCM-020 | DLV-FR-PCM-020 | EP-PCM-001 | M6 |
| FR-PCM-021 | DLV-FR-PCM-021 | EP-PCM-001 | M6 |
| FR-PCM-022 | DLV-FR-PCM-022 | EP-PCM-001 | M6 |
| FR-PCM-023 | DLV-FR-PCM-023 | EP-PCM-001 | M6 |
| FR-PCM-024 | DLV-FR-PCM-024 | EP-PCM-001 | M6 |
| FR-PCM-025 | DLV-FR-PCM-025 | EP-PCM-001 | M5 |
| FR-REV-001 | DLV-FR-REV-001 | EP-REV-001 | M7 |
| FR-REV-002 | DLV-FR-REV-002 | EP-REV-001 | M7 |
| FR-REV-003 | DLV-FR-REV-003 | EP-REV-001 | M7 |
| FR-REV-004 | DLV-FR-REV-004 | EP-REV-001 | M7 |
| FR-REV-005 | DLV-FR-REV-005 | EP-REV-001 | M7 |
| FR-REV-006 | DLV-FR-REV-006 | EP-REV-001 | M7 |
| FR-RPT-001 | DLV-FR-RPT-001 | EP-RPT-001 | M8 |
| FR-RPT-002 | DLV-FR-RPT-002 | EP-RPT-001 | M8 |
| FR-RPT-003 | DLV-FR-RPT-003 | EP-RPT-001 | M8 |
| FR-RPT-004 | DLV-FR-RPT-004 | EP-RPT-001 | M8 |
| FR-RPT-005 | DLV-FR-RPT-005 | EP-RPT-001 | M8 |
| FR-RPT-006 | DLV-FR-RPT-006 | EP-RPT-001 | M8 |
| FR-TAX-001 | DLV-FR-TAX-001 | EP-TAX-001 | M9 |
| FR-TAX-002 | DLV-FR-TAX-002 | EP-TAX-001 | M9 |
| FR-TAX-003 | DLV-FR-TAX-003 | EP-TAX-001 | M9 |
| FR-TAX-004 | DLV-FR-TAX-004 | EP-TAX-001 | M9 |
| FR-TAX-005 | DLV-FR-TAX-005 | EP-TAX-001 | M9 |
| FR-TAX-006 | DLV-FR-TAX-006 | EP-TAX-001 | M9 |
| FR-TAX-007 | DLV-FR-TAX-007 | EP-TAX-001 | M9 |
| FR-TAX-008 | DLV-FR-TAX-008 | EP-TAX-001 | M9 |
| FR-TAX-009 | DLV-FR-TAX-009 | EP-TAX-001 | M9 |
| FR-TAX-010 | DLV-FR-TAX-010 | EP-TAX-001 | M9 |
| FR-TAX-011 | DLV-FR-TAX-011 | EP-TAX-001 | M9 |
| FR-TAX-012 | DLV-FR-TAX-012 | EP-TAX-001 | M9 |
| FR-TAX-013 | DLV-FR-TAX-013 | EP-TAX-001 | M9 |
| FR-TAX-014 | DLV-FR-TAX-014 | EP-TAX-001 | M9 |
| FR-TAX-015 | DLV-FR-TAX-015 | EP-TAX-001 | M9 |
| FR-TAX-016 | DLV-FR-TAX-016 | EP-TAX-001 | M9 |
| FR-WFA-001 | DLV-FR-WFA-001 | EP-WFA-001 | M3 |
| FR-WFA-002 | DLV-FR-WFA-002 | EP-WFA-001 | M3 |
| FR-WFA-003 | DLV-FR-WFA-003 | EP-WFA-001 | M3 |
| FR-WFA-004 | DLV-FR-WFA-004 | EP-WFA-001 | M3 |
| FR-WFA-005 | DLV-FR-WFA-005 | EP-WFA-001 | M3 |

## 7. Exact global requirement traceability

| Requirement | Delivery item | Epic | Milestone |
|---|---|---|---|
| GFR-001 | DLV-GFR-001 | EP-PLAT-001 | M0 |
| GFR-002 | DLV-GFR-002 | EP-IAM-001 | M1 |
| GFR-003 | DLV-GFR-003 | EP-IAM-001 | M1 |
| GFR-004 | DLV-GFR-004 | EP-WFA-001 | M2 |
| GFR-005 | DLV-GFR-005 | EP-GL-001 | M2 |
| GFR-006 | DLV-GFR-006 | EP-PLAT-001 | M0 |
| GFR-007 | DLV-GFR-007 | EP-PLAT-001 | M0 |
| GFR-008 | DLV-GFR-008 | EP-PLAT-001 | M0 |
| GFR-009 | DLV-GFR-009 | EP-UX-001 | M0 |
| GFR-010 | DLV-GFR-010 | EP-FX-001 | M8 |
| GFR-011 | DLV-GFR-011 | EP-GL-001 | M2 |
| GFR-012 | DLV-GFR-012 | EP-OPS-001 | M0 |
| GFR-013 | DLV-GFR-013 | EP-PLAT-001 | M0 |
| GFR-014 | DLV-GFR-014 | EP-AUD-001 | M1 |
| GFR-015 | DLV-GFR-015 | EP-IAM-001 | M1 |
| GFR-016 | DLV-GFR-016 | EP-UX-001 | M0 |
| GFR-017 | DLV-GFR-017 | EP-UX-001 | M0 |
| GFR-018 | DLV-GFR-018 | EP-AUD-001 | M9 |
| GFR-019 | DLV-GFR-019 | EP-OMD-001 | M1 |
| GFR-020 | DLV-GFR-020 | EP-AUD-001 | M9 |
| GFR-021 | DLV-GFR-021 | EP-PLAT-001 | M0 |
| GFR-022 | DLV-GFR-022 | EP-PLAT-001 | M0 |

## 8. Exact workflow traceability

| Workflow | Delivery item | Milestone | Acceptance source |
|---|---|---|---|
| WF-6.1 | DLV-WF-6.1 | M3 | DDD acceptance §§14.3 and 14.8 |
| WF-6.2 | DLV-WF-6.2 | M3 | DDD acceptance §§14.10 and 14.8 |
| WF-6.3 | DLV-WF-6.3 | M8 | DDD acceptance §14.11 |
| WF-6.4 | DLV-WF-6.4 | M7 | DDD acceptance §§14.2 and 14.13.7 |
| WF-6.5 | DLV-WF-6.5 | M7 | DDD acceptance §§14.12 and 14.13.8 |
| WF-6.6 | DLV-WF-6.6 | M2 | DDD acceptance §§14.1 and 14.9 |
| WF-6.7 | DLV-WF-6.7 | M4 | DDD acceptance §14.6 |
| WF-7.1 | DLV-WF-7.1 | M5 | DDD acceptance §14.13.1 |
| WF-7.2 | DLV-WF-7.2 | M5 | DDD acceptance §14.13.2 |
| WF-7.3 | DLV-WF-7.3 | M6 | DDD acceptance §14.13.3 |
| WF-7.4 | DLV-WF-7.4 | M6 | DDD acceptance §14.13.4 |
| WF-7.5 | DLV-WF-7.5 | M8 | DDD acceptance §14.13.5 |
| WF-7.6 | DLV-WF-7.6 | M8 | DDD acceptance §14.13.6 |
| WF-7.7 | DLV-WF-7.7 | M7 | DDD acceptance §14.13.7 |
| WF-7.8 | DLV-WF-7.8 | M7 | DDD acceptance §14.13.8 |
| WF-7.9 | DLV-WF-7.9 | M8 | DDD acceptance §14.13.9 |
| WF-7.10 | DLV-WF-7.10 | M9 | DDD acceptance §14.13.10 |
| WF-7.11 | DLV-WF-7.11 | M9 | DDD acceptance §14.13.11 |
| WF-7.12 | DLV-WF-7.12 | M3 | DDD acceptance §14.13.12 |
| WF-7.13 | DLV-WF-7.13 | M9 | DDD acceptance §14.13.13 |
| WF-7.14 | DLV-WF-7.14 | M9 | DDD acceptance §14.13.14 |
| WF-7.15 | DLV-WF-7.15 | M9 | DDD acceptance §14.13.15 |

## 9. Exact NFR traceability

| NFR | Delivery item | Quality gate | First required milestone |
|---|---|---|---|
| NFR-ACC-001 | DLV-NFR-ACC-001 | QG-05 | M0 |
| NFR-ACC-002 | DLV-NFR-ACC-002 | QG-05 | M0 |
| NFR-ACC-003 | DLV-NFR-ACC-003 | QG-05 | M0 |
| NFR-ACC-004 | DLV-NFR-ACC-004 | QG-05 | M0 |
| NFR-ACC-005 | DLV-NFR-ACC-005 | QG-05 | M0 |
| NFR-ACC-006 | DLV-NFR-ACC-006 | QG-05 | M0 |
| NFR-ACC-007 | DLV-NFR-ACC-007 | QG-05 | M0 |
| NFR-ACC-008 | DLV-NFR-ACC-008 | QG-05 | M0 |
| NFR-ACC-009 | DLV-NFR-ACC-009 | QG-05 | M0 |
| NFR-ACC-010 | DLV-NFR-ACC-010 | QG-05 | M0 |
| NFR-ACC-011 | DLV-NFR-ACC-011 | QG-05 | M0 |
| NFR-ACC-012 | DLV-NFR-ACC-012 | QG-05 | M0 |
| NFR-AUD-001 | DLV-NFR-AUD-001 | QG-08 | M9 |
| NFR-AUD-002 | DLV-NFR-AUD-002 | QG-08 | M9 |
| NFR-AUD-003 | DLV-NFR-AUD-003 | QG-08 | M9 |
| NFR-AUD-004 | DLV-NFR-AUD-004 | QG-08 | M9 |
| NFR-AUD-005 | DLV-NFR-AUD-005 | QG-08 | M9 |
| NFR-AUD-006 | DLV-NFR-AUD-006 | QG-08 | M9 |
| NFR-AUD-007 | DLV-NFR-AUD-007 | QG-08 | M9 |
| NFR-AUD-008 | DLV-NFR-AUD-008 | QG-08 | M9 |
| NFR-AUD-009 | DLV-NFR-AUD-009 | QG-08 | M9 |
| NFR-AUD-010 | DLV-NFR-AUD-010 | QG-08 | M9 |
| NFR-AUD-011 | DLV-NFR-AUD-011 | QG-08 | M9 |
| NFR-AUD-012 | DLV-NFR-AUD-012 | QG-08 | M9 |
| NFR-AVL-001 | DLV-NFR-AVL-001 | QG-09 | M9 |
| NFR-AVL-002 | DLV-NFR-AVL-002 | QG-09 | M9 |
| NFR-AVL-003 | DLV-NFR-AVL-003 | QG-09 | M9 |
| NFR-AVL-004 | DLV-NFR-AVL-004 | QG-09 | M9 |
| NFR-AVL-005 | DLV-NFR-AVL-005 | QG-09 | M9 |
| NFR-AVL-006 | DLV-NFR-AVL-006 | QG-09 | M9 |
| NFR-AVL-007 | DLV-NFR-AVL-007 | QG-09 | M9 |
| NFR-AVL-008 | DLV-NFR-AVL-008 | QG-09 | M9 |
| NFR-AVL-009 | DLV-NFR-AVL-009 | QG-09 | M9 |
| NFR-AVL-010 | DLV-NFR-AVL-010 | QG-09 | M9 |
| NFR-CAP-001 | DLV-NFR-CAP-001 | QG-09 | M9 |
| NFR-CAP-002 | DLV-NFR-CAP-002 | QG-09 | M9 |
| NFR-CAP-003 | DLV-NFR-CAP-003 | QG-09 | M9 |
| NFR-CAP-004 | DLV-NFR-CAP-004 | QG-09 | M9 |
| NFR-CAP-005 | DLV-NFR-CAP-005 | QG-09 | M9 |
| NFR-CAP-006 | DLV-NFR-CAP-006 | QG-09 | M9 |
| NFR-CAP-007 | DLV-NFR-CAP-007 | QG-09 | M9 |
| NFR-CAP-008 | DLV-NFR-CAP-008 | QG-09 | M9 |
| NFR-CAP-009 | DLV-NFR-CAP-009 | QG-09 | M9 |
| NFR-CAP-010 | DLV-NFR-CAP-010 | QG-09 | M9 |
| NFR-CMP-001 | DLV-NFR-CMP-001 | QG-05 | M0 |
| NFR-CMP-002 | DLV-NFR-CMP-002 | QG-05 | M0 |
| NFR-CMP-003 | DLV-NFR-CMP-003 | QG-05 | M0 |
| NFR-CMP-004 | DLV-NFR-CMP-004 | QG-05 | M0 |
| NFR-CMP-005 | DLV-NFR-CMP-005 | QG-05 | M0 |
| NFR-CMP-006 | DLV-NFR-CMP-006 | QG-05 | M0 |
| NFR-CMP-007 | DLV-NFR-CMP-007 | QG-05 | M0 |
| NFR-CMP-008 | DLV-NFR-CMP-008 | QG-05 | M0 |
| NFR-INT-001 | DLV-NFR-INT-001 | QG-04 | M2 |
| NFR-INT-002 | DLV-NFR-INT-002 | QG-04 | M2 |
| NFR-INT-003 | DLV-NFR-INT-003 | QG-04 | M2 |
| NFR-INT-004 | DLV-NFR-INT-004 | QG-04 | M2 |
| NFR-INT-005 | DLV-NFR-INT-005 | QG-04 | M2 |
| NFR-INT-006 | DLV-NFR-INT-006 | QG-04 | M2 |
| NFR-INT-007 | DLV-NFR-INT-007 | QG-04 | M2 |
| NFR-INT-008 | DLV-NFR-INT-008 | QG-04 | M2 |
| NFR-INT-009 | DLV-NFR-INT-009 | QG-04 | M2 |
| NFR-INT-010 | DLV-NFR-INT-010 | QG-04 | M2 |
| NFR-LOC-001 | DLV-NFR-LOC-001 | QG-05 | M0 |
| NFR-LOC-002 | DLV-NFR-LOC-002 | QG-05 | M0 |
| NFR-LOC-003 | DLV-NFR-LOC-003 | QG-05 | M0 |
| NFR-LOC-004 | DLV-NFR-LOC-004 | QG-05 | M0 |
| NFR-LOC-005 | DLV-NFR-LOC-005 | QG-05 | M0 |
| NFR-LOC-006 | DLV-NFR-LOC-006 | QG-05 | M0 |
| NFR-LOC-007 | DLV-NFR-LOC-007 | QG-05 | M0 |
| NFR-LOC-008 | DLV-NFR-LOC-008 | QG-05 | M0 |
| NFR-MNT-001 | DLV-NFR-MNT-001 | QG-01 | M0 |
| NFR-MNT-002 | DLV-NFR-MNT-002 | QG-01 | M0 |
| NFR-MNT-003 | DLV-NFR-MNT-003 | QG-01 | M0 |
| NFR-MNT-004 | DLV-NFR-MNT-004 | QG-01 | M0 |
| NFR-MNT-005 | DLV-NFR-MNT-005 | QG-01 | M0 |
| NFR-MNT-006 | DLV-NFR-MNT-006 | QG-01 | M0 |
| NFR-MNT-007 | DLV-NFR-MNT-007 | QG-01 | M0 |
| NFR-MNT-008 | DLV-NFR-MNT-008 | QG-01 | M0 |
| NFR-MNT-009 | DLV-NFR-MNT-009 | QG-01 | M0 |
| NFR-MNT-010 | DLV-NFR-MNT-010 | QG-01 | M0 |
| NFR-MNT-011 | DLV-NFR-MNT-011 | QG-01 | M0 |
| NFR-MNT-012 | DLV-NFR-MNT-012 | QG-01 | M0 |
| NFR-OBS-001 | DLV-NFR-OBS-001 | QG-08 | M0 |
| NFR-OBS-002 | DLV-NFR-OBS-002 | QG-08 | M0 |
| NFR-OBS-003 | DLV-NFR-OBS-003 | QG-08 | M0 |
| NFR-OBS-004 | DLV-NFR-OBS-004 | QG-08 | M0 |
| NFR-OBS-005 | DLV-NFR-OBS-005 | QG-08 | M0 |
| NFR-OBS-006 | DLV-NFR-OBS-006 | QG-08 | M0 |
| NFR-OBS-007 | DLV-NFR-OBS-007 | QG-08 | M0 |
| NFR-OBS-008 | DLV-NFR-OBS-008 | QG-08 | M0 |
| NFR-OBS-009 | DLV-NFR-OBS-009 | QG-08 | M0 |
| NFR-OBS-010 | DLV-NFR-OBS-010 | QG-08 | M0 |
| NFR-OBS-011 | DLV-NFR-OBS-011 | QG-08 | M0 |
| NFR-OBS-012 | DLV-NFR-OBS-012 | QG-08 | M0 |
| NFR-PERF-001 | DLV-NFR-PERF-001 | QG-09 | M9 |
| NFR-PERF-002 | DLV-NFR-PERF-002 | QG-09 | M9 |
| NFR-PERF-003 | DLV-NFR-PERF-003 | QG-09 | M9 |
| NFR-PERF-004 | DLV-NFR-PERF-004 | QG-09 | M9 |
| NFR-PERF-005 | DLV-NFR-PERF-005 | QG-09 | M9 |
| NFR-PERF-006 | DLV-NFR-PERF-006 | QG-09 | M9 |
| NFR-PERF-007 | DLV-NFR-PERF-007 | QG-09 | M9 |
| NFR-PERF-008 | DLV-NFR-PERF-008 | QG-09 | M9 |
| NFR-PERF-009 | DLV-NFR-PERF-009 | QG-09 | M9 |
| NFR-PERF-010 | DLV-NFR-PERF-010 | QG-09 | M9 |
| NFR-PERF-011 | DLV-NFR-PERF-011 | QG-09 | M9 |
| NFR-PERF-012 | DLV-NFR-PERF-012 | QG-09 | M9 |
| NFR-PERF-013 | DLV-NFR-PERF-013 | QG-09 | M9 |
| NFR-PERF-014 | DLV-NFR-PERF-014 | QG-09 | M9 |
| NFR-PRV-001 | DLV-NFR-PRV-001 | QG-06 | M1 |
| NFR-PRV-002 | DLV-NFR-PRV-002 | QG-06 | M1 |
| NFR-PRV-003 | DLV-NFR-PRV-003 | QG-06 | M1 |
| NFR-PRV-004 | DLV-NFR-PRV-004 | QG-06 | M1 |
| NFR-PRV-005 | DLV-NFR-PRV-005 | QG-06 | M1 |
| NFR-PRV-006 | DLV-NFR-PRV-006 | QG-06 | M1 |
| NFR-PRV-007 | DLV-NFR-PRV-007 | QG-06 | M1 |
| NFR-PRV-008 | DLV-NFR-PRV-008 | QG-06 | M1 |
| NFR-PRV-009 | DLV-NFR-PRV-009 | QG-06 | M1 |
| NFR-PRV-010 | DLV-NFR-PRV-010 | QG-06 | M1 |
| NFR-REC-001 | DLV-NFR-REC-001 | QG-09 | M9 |
| NFR-REC-002 | DLV-NFR-REC-002 | QG-09 | M9 |
| NFR-REC-003 | DLV-NFR-REC-003 | QG-09 | M9 |
| NFR-REC-004 | DLV-NFR-REC-004 | QG-09 | M9 |
| NFR-REC-005 | DLV-NFR-REC-005 | QG-09 | M9 |
| NFR-REC-006 | DLV-NFR-REC-006 | QG-09 | M9 |
| NFR-REC-007 | DLV-NFR-REC-007 | QG-09 | M9 |
| NFR-REC-008 | DLV-NFR-REC-008 | QG-09 | M9 |
| NFR-REC-009 | DLV-NFR-REC-009 | QG-09 | M9 |
| NFR-REC-010 | DLV-NFR-REC-010 | QG-09 | M9 |
| NFR-REC-011 | DLV-NFR-REC-011 | QG-09 | M9 |
| NFR-REC-012 | DLV-NFR-REC-012 | QG-09 | M9 |
| NFR-REL-001 | DLV-NFR-REL-001 | QG-07 | M2 |
| NFR-REL-002 | DLV-NFR-REL-002 | QG-07 | M2 |
| NFR-REL-003 | DLV-NFR-REL-003 | QG-07 | M2 |
| NFR-REL-004 | DLV-NFR-REL-004 | QG-07 | M2 |
| NFR-REL-005 | DLV-NFR-REL-005 | QG-07 | M2 |
| NFR-REL-006 | DLV-NFR-REL-006 | QG-07 | M2 |
| NFR-REL-007 | DLV-NFR-REL-007 | QG-07 | M2 |
| NFR-REL-008 | DLV-NFR-REL-008 | QG-07 | M2 |
| NFR-REL-009 | DLV-NFR-REL-009 | QG-07 | M2 |
| NFR-REL-010 | DLV-NFR-REL-010 | QG-07 | M2 |
| NFR-REL-011 | DLV-NFR-REL-011 | QG-07 | M2 |
| NFR-REL-012 | DLV-NFR-REL-012 | QG-07 | M2 |
| NFR-REL-013 | DLV-NFR-REL-013 | QG-07 | M2 |
| NFR-REL-014 | DLV-NFR-REL-014 | QG-07 | M2 |
| NFR-REL-015 | DLV-NFR-REL-015 | QG-07 | M2 |
| NFR-REL-016 | DLV-NFR-REL-016 | QG-07 | M2 |
| NFR-SEC-001 | DLV-NFR-SEC-001 | QG-06 | M1 |
| NFR-SEC-002 | DLV-NFR-SEC-002 | QG-06 | M1 |
| NFR-SEC-003 | DLV-NFR-SEC-003 | QG-06 | M1 |
| NFR-SEC-004 | DLV-NFR-SEC-004 | QG-06 | M1 |
| NFR-SEC-005 | DLV-NFR-SEC-005 | QG-06 | M1 |
| NFR-SEC-006 | DLV-NFR-SEC-006 | QG-06 | M1 |
| NFR-SEC-007 | DLV-NFR-SEC-007 | QG-06 | M1 |
| NFR-SEC-008 | DLV-NFR-SEC-008 | QG-06 | M1 |
| NFR-SEC-009 | DLV-NFR-SEC-009 | QG-06 | M1 |
| NFR-SEC-010 | DLV-NFR-SEC-010 | QG-06 | M1 |
| NFR-SEC-011 | DLV-NFR-SEC-011 | QG-06 | M1 |
| NFR-SEC-012 | DLV-NFR-SEC-012 | QG-06 | M1 |
| NFR-SEC-013 | DLV-NFR-SEC-013 | QG-06 | M1 |
| NFR-SEC-014 | DLV-NFR-SEC-014 | QG-06 | M1 |
| NFR-SEC-015 | DLV-NFR-SEC-015 | QG-06 | M1 |
| NFR-SEC-016 | DLV-NFR-SEC-016 | QG-06 | M1 |
| NFR-SEC-017 | DLV-NFR-SEC-017 | QG-06 | M1 |
| NFR-SEC-018 | DLV-NFR-SEC-018 | QG-06 | M1 |
| NFR-TST-001 | DLV-NFR-TST-001 | QG-10 | M9 |
| NFR-TST-002 | DLV-NFR-TST-002 | QG-10 | M9 |
| NFR-TST-003 | DLV-NFR-TST-003 | QG-10 | M9 |
| NFR-TST-004 | DLV-NFR-TST-004 | QG-10 | M9 |
| NFR-TST-005 | DLV-NFR-TST-005 | QG-10 | M9 |
| NFR-TST-006 | DLV-NFR-TST-006 | QG-10 | M9 |
| NFR-TST-007 | DLV-NFR-TST-007 | QG-10 | M9 |
| NFR-TST-008 | DLV-NFR-TST-008 | QG-10 | M9 |
| NFR-TST-009 | DLV-NFR-TST-009 | QG-10 | M9 |
| NFR-TST-010 | DLV-NFR-TST-010 | QG-10 | M9 |

## 10. Exact acceptance traceability

| Acceptance scenario | Test pack | Primary milestone |
|---|---|---|
| FAC-14-1-01 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-02 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-03 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-04 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-05 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-06 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-07 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-08 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-1-09 | `tests/acceptance/fac-14-1` | M2 |
| FAC-14-2-01 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-02 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-03 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-04 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-05 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-06 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-07 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-08 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-09 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-10 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-11 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-12 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-13 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-14 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-15 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-16 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-17 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-18 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-19 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-2-20 | `tests/acceptance/fac-14-2` | M7 |
| FAC-14-3-01 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-02 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-03 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-04 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-05 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-06 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-07 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-08 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-09 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-10 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-11 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-12 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-13 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-14 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-15 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-16 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-17 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-18 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-19 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-20 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-3-21 | `tests/acceptance/fac-14-3` | M3 |
| FAC-14-4-01 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-02 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-03 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-04 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-05 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-06 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-07 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-4-08 | `tests/acceptance/fac-14-4` | M9 |
| FAC-14-5-01 | `tests/acceptance/fac-14-5` | M9 |
| FAC-14-5-02 | `tests/acceptance/fac-14-5` | M9 |
| FAC-14-5-03 | `tests/acceptance/fac-14-5` | M9 |
| FAC-14-5-04 | `tests/acceptance/fac-14-5` | M9 |
| FAC-14-6-01 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-02 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-03 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-04 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-05 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-06 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-07 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-08 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-09 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-10 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-11 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-12 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-13 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-14 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-15 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-16 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-17 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-18 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-19 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-20 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-21 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-6-22 | `tests/acceptance/fac-14-6` | M4 |
| FAC-14-7-01 | `tests/acceptance/fac-14-7` | M8 |
| FAC-14-7-02 | `tests/acceptance/fac-14-7` | M8 |
| FAC-14-7-03 | `tests/acceptance/fac-14-7` | M8 |
| FAC-14-7-04 | `tests/acceptance/fac-14-7` | M8 |
| FAC-14-7-05 | `tests/acceptance/fac-14-7` | M8 |
| FAC-14-7-06 | `tests/acceptance/fac-14-7` | M8 |
| FAC-14-8-01 | `tests/acceptance/fac-14-8` | M3 |
| FAC-14-8-02 | `tests/acceptance/fac-14-8` | M3 |
| FAC-14-8-03 | `tests/acceptance/fac-14-8` | M3 |
| FAC-14-9-01 | `tests/acceptance/fac-14-9` | M9 |
| FAC-14-9-02 | `tests/acceptance/fac-14-9` | M9 |
| FAC-14-9-03 | `tests/acceptance/fac-14-9` | M9 |
| FAC-14-10-01 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-02 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-03 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-04 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-05 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-06 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-07 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-10-08 | `tests/acceptance/fac-14-10` | M3 |
| FAC-14-11-01 | `tests/acceptance/fac-14-11` | M8 |
| FAC-14-11-02 | `tests/acceptance/fac-14-11` | M8 |
| FAC-14-11-03 | `tests/acceptance/fac-14-11` | M8 |
| FAC-14-11-04 | `tests/acceptance/fac-14-11` | M8 |
| FAC-14-11-05 | `tests/acceptance/fac-14-11` | M8 |
| FAC-14-11-06 | `tests/acceptance/fac-14-11` | M8 |
| FAC-14-12-01 | `tests/acceptance/fac-14-12` | M7 |
| FAC-14-12-02 | `tests/acceptance/fac-14-12` | M7 |
| FAC-14-12-03 | `tests/acceptance/fac-14-12` | M7 |
| FAC-14-12-04 | `tests/acceptance/fac-14-12` | M7 |
| FAC-14-12-05 | `tests/acceptance/fac-14-12` | M7 |
| FAC-14-12-06 | `tests/acceptance/fac-14-12` | M7 |
| FAC-14-13-1-01 | `tests/acceptance/fac-14-13-1` | M5 |
| FAC-14-13-1-02 | `tests/acceptance/fac-14-13-1` | M5 |
| FAC-14-13-1-03 | `tests/acceptance/fac-14-13-1` | M5 |
| FAC-14-13-1-04 | `tests/acceptance/fac-14-13-1` | M5 |
| FAC-14-13-1-05 | `tests/acceptance/fac-14-13-1` | M5 |
| FAC-14-13-2-01 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-02 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-03 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-04 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-05 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-06 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-07 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-08 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-09 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-10 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-11 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-12 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-2-13 | `tests/acceptance/fac-14-13-2` | M5 |
| FAC-14-13-3-01 | `tests/acceptance/fac-14-13-3` | M4 |
| FAC-14-13-3-02 | `tests/acceptance/fac-14-13-3` | M4 |
| FAC-14-13-3-03 | `tests/acceptance/fac-14-13-3` | M4 |
| FAC-14-13-3-04 | `tests/acceptance/fac-14-13-3` | M4 |
| FAC-14-13-3-05 | `tests/acceptance/fac-14-13-3` | M4 |
| FAC-14-13-4-01 | `tests/acceptance/fac-14-13-4` | M6 |
| FAC-14-13-4-02 | `tests/acceptance/fac-14-13-4` | M6 |
| FAC-14-13-4-03 | `tests/acceptance/fac-14-13-4` | M6 |
| FAC-14-13-4-04 | `tests/acceptance/fac-14-13-4` | M6 |
| FAC-14-13-4-05 | `tests/acceptance/fac-14-13-4` | M6 |
| FAC-14-13-5-01 | `tests/acceptance/fac-14-13-5` | M8 |
| FAC-14-13-5-02 | `tests/acceptance/fac-14-13-5` | M8 |
| FAC-14-13-5-03 | `tests/acceptance/fac-14-13-5` | M8 |
| FAC-14-13-5-04 | `tests/acceptance/fac-14-13-5` | M8 |
| FAC-14-13-5-05 | `tests/acceptance/fac-14-13-5` | M8 |
| FAC-14-13-6-01 | `tests/acceptance/fac-14-13-6` | M8 |
| FAC-14-13-6-02 | `tests/acceptance/fac-14-13-6` | M8 |
| FAC-14-13-6-03 | `tests/acceptance/fac-14-13-6` | M8 |
| FAC-14-13-6-04 | `tests/acceptance/fac-14-13-6` | M8 |
| FAC-14-13-6-05 | `tests/acceptance/fac-14-13-6` | M8 |
| FAC-14-13-7-01 | `tests/acceptance/fac-14-13-7` | M7 |
| FAC-14-13-7-02 | `tests/acceptance/fac-14-13-7` | M7 |
| FAC-14-13-7-03 | `tests/acceptance/fac-14-13-7` | M7 |
| FAC-14-13-7-04 | `tests/acceptance/fac-14-13-7` | M7 |
| FAC-14-13-7-05 | `tests/acceptance/fac-14-13-7` | M7 |
| FAC-14-13-8-01 | `tests/acceptance/fac-14-13-8` | M7 |
| FAC-14-13-8-02 | `tests/acceptance/fac-14-13-8` | M7 |
| FAC-14-13-8-03 | `tests/acceptance/fac-14-13-8` | M7 |
| FAC-14-13-8-04 | `tests/acceptance/fac-14-13-8` | M7 |
| FAC-14-13-8-05 | `tests/acceptance/fac-14-13-8` | M7 |
| FAC-14-13-9-01 | `tests/acceptance/fac-14-13-9` | M8 |
| FAC-14-13-9-02 | `tests/acceptance/fac-14-13-9` | M8 |
| FAC-14-13-9-03 | `tests/acceptance/fac-14-13-9` | M8 |
| FAC-14-13-9-04 | `tests/acceptance/fac-14-13-9` | M8 |
| FAC-14-13-9-05 | `tests/acceptance/fac-14-13-9` | M8 |
| FAC-14-13-10-01 | `tests/acceptance/fac-14-13-10` | M9 |
| FAC-14-13-10-02 | `tests/acceptance/fac-14-13-10` | M9 |
| FAC-14-13-10-03 | `tests/acceptance/fac-14-13-10` | M9 |
| FAC-14-13-10-04 | `tests/acceptance/fac-14-13-10` | M9 |
| FAC-14-13-10-05 | `tests/acceptance/fac-14-13-10` | M9 |
| FAC-14-13-11-01 | `tests/acceptance/fac-14-13-11` | M9 |
| FAC-14-13-11-02 | `tests/acceptance/fac-14-13-11` | M9 |
| FAC-14-13-11-03 | `tests/acceptance/fac-14-13-11` | M9 |
| FAC-14-13-11-04 | `tests/acceptance/fac-14-13-11` | M9 |
| FAC-14-13-11-05 | `tests/acceptance/fac-14-13-11` | M9 |
| FAC-14-13-12-01 | `tests/acceptance/fac-14-13-12` | M3 |
| FAC-14-13-12-02 | `tests/acceptance/fac-14-13-12` | M3 |
| FAC-14-13-12-03 | `tests/acceptance/fac-14-13-12` | M3 |
| FAC-14-13-12-04 | `tests/acceptance/fac-14-13-12` | M3 |
| FAC-14-13-12-05 | `tests/acceptance/fac-14-13-12` | M3 |
| FAC-14-13-13-01 | `tests/acceptance/fac-14-13-13` | M9 |
| FAC-14-13-13-02 | `tests/acceptance/fac-14-13-13` | M9 |
| FAC-14-13-13-03 | `tests/acceptance/fac-14-13-13` | M9 |
| FAC-14-13-13-04 | `tests/acceptance/fac-14-13-13` | M9 |
| FAC-14-13-13-05 | `tests/acceptance/fac-14-13-13` | M9 |
| FAC-14-13-14-01 | `tests/acceptance/fac-14-13-14` | M9 |
| FAC-14-13-14-02 | `tests/acceptance/fac-14-13-14` | M9 |
| FAC-14-13-14-03 | `tests/acceptance/fac-14-13-14` | M9 |
| FAC-14-13-14-04 | `tests/acceptance/fac-14-13-14` | M9 |
| FAC-14-13-14-05 | `tests/acceptance/fac-14-13-14` | M9 |
| FAC-14-13-15-01 | `tests/acceptance/fac-14-13-15` | M9 |
| FAC-14-13-15-02 | `tests/acceptance/fac-14-13-15` | M9 |
| FAC-14-13-15-03 | `tests/acceptance/fac-14-13-15` | M9 |
| FAC-14-13-15-04 | `tests/acceptance/fac-14-13-15` | M9 |
| FAC-14-13-15-05 | `tests/acceptance/fac-14-13-15` | M9 |

## 11. Technical specification applicability

| Technical specification | Delivery use |
|---|---|
| 01 Backend Module Specifications | Module packages, command handlers, domain implementation and dependency rules |
| 02 API and OpenAPI Specifications | Exact operation routes, schemas, errors and idempotency contracts |
| 03 Database and Persistence Specifications | PostgreSQL schemas, constraints, indexes, locks and migrations |
| 04 Events, Workers and Integration Specifications | Outbox/inbox, workers, event contracts and external integrations |
| 05 Frontend and UI Technical Specifications | Routes, screens, components, forms and accessibility |
| 06 Security, Identity and Authorization Specifications | Entra, permissions, scope, SoD and sensitive data |
| 07 Terraform and Azure Deployment Specifications | Infrastructure modules, environments, state and deployment |
| 08 Observability and Operations Specifications | Logs, metrics, traces, alerts and runbooks |
| 09 Testing, Performance and Recovery Specifications | Test suites, load profiles, backup, restore and DR |
| 10 Technical Traceability and Decisions | Technical ADRs and source-to-contract verification |

## 12. Unscheduled scope

The baseline does not schedule mobile clients, advanced procurement, inventory, manufacturing, expense management, advanced treasury, production multi-region topology, Kafka, Redis, Elasticsearch, AKS, service mesh or a dedicated data lake. Such work requires new approved scope and delivery analysis.
## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `f85868c1abb31e39a996de6a377a642c23937758f93c94b3421b47d8ec6e7488` |
| Review status | Passed |
| Reuse rule | Re-run structural checks when this body hash and all source hashes remain unchanged. Run targeted semantic review for localized backlog, estimate, dependency, milestone, gate, risk or cost changes. Run the full suite for requirement, workflow, acceptance, architecture, technical-specification or source-hash changes. |

### Checks recorded

- All 193 FRs, 22 GFRs, 22 workflows, 174 NFRs and 199 acceptance scenarios are mapped exactly.
- 18 delivery risks and 10 cost controls are recorded.
- Governance, conceptual ownership, progress metrics and change control are defined.
- Technical-specification applicability and unscheduled scope are explicit.
