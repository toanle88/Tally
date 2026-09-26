# Authoritative sources

This public site presents selected summaries. The repository remains the
authority for requirements, domain rules, architecture, implementation detail,
delivery decisions, and verification evidence.

## Product and domain baseline

- [Finance DDD baseline](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_domain_model_ddd.md)
- [Functional PRD](https://github.com/toanle88/Tally/blob/main/docs/specs/prd/01_finance_functional_prd_v1.5.md)
- [Functional requirements catalog](https://github.com/toanle88/Tally/blob/main/docs/specs/prd/02_finance_functional_requirements_catalog_v1.5.md)
- [Functional traceability and acceptance](https://github.com/toanle88/Tally/blob/main/docs/specs/prd/03_finance_functional_traceability_acceptance_v1.5.md)
- [UX and workflow specification](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_ux_workflow_specification_v1.0.md)
- [Non-functional requirements](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_nonfunctional_requirements_v1.0.md)

## Design and delivery

- [System design pack](https://github.com/toanle88/Tally/tree/main/docs/specs/system_design)
- [Technical specifications](https://github.com/toanle88/Tally/tree/main/docs/specs/technical_specifications)
- [Finance delivery plan](https://github.com/toanle88/Tally/blob/main/docs/specs/finance_delivery_plan_v1.0.md)
- [Backlog and user stories](https://github.com/toanle88/Tally/tree/main/docs/backlog)
- [Live roadmap](https://github.com/toanle88/Tally/blob/main/ROADMAP.md)

## Verification evidence

The [verification directory](https://github.com/toanle88/Tally/tree/main/docs/verification)
contains command-level evidence and explicit limitations. Representative
groups are:

- [Platform primitives and persistence](https://github.com/toanle88/Tally/tree/main/docs/verification)
- [Outbox, inbox, dispatch, and worker lifecycle](https://github.com/toanle88/Tally/tree/main/docs/verification)
- [Identity and access](https://github.com/toanle88/Tally/tree/main/docs/verification)
- [UX and accessibility](https://github.com/toanle88/Tally/tree/main/docs/verification)
- [Operations and Terraform](https://github.com/toanle88/Tally/tree/main/docs/verification)

An evidence record is not automatically a production qualification record. It
is tied to its branch, date, environment, command output, and stated boundary.

## Reading status correctly

The live roadmap’s `[x]` means a delivery item is locally complete or
intentionally closed; it does not mean that Azure, production, security,
availability, recovery, or full-domain qualification has passed. The OpenAPI
catalog and UX specifications describe the approved target surface; implemented
runtime behavior is established by repository code and verification output.
