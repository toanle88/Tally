# Platform Operations Runbook Template

| Field | Value |
|---|---|
| Version | `v1` |
| Status | M0 local operational baseline |
| Owner | Platform Operations |
| Audience | Authorized operations users; authorization enforcement remains with `EP-IAM-001` |
| Data classification | Operational diagnostic data; not authoritative financial or audit evidence |
| Provider boundary | Provider-neutral; live monitoring, paging, and production qualification are deferred |

## Purpose

Use this template for a safe, repeatable response to a platform failure. A
runbook is an operational procedure and does not replace an owning domain
operation, audit evidence, the transactional outbox/inbox, or an approved
recovery decision.

Each runbook must identify what is known, what is uncertain, which actions are
safe, what evidence must be preserved, and who owns the next decision. Missing
or stale telemetry is an operational condition; it is never success, zero,
healthy, or authoritative financial evidence.

## Required runbook structure

Every runbook must contain the following sections.

## Runbook metadata

- **Runbook:** `<approved RUN-* identifier and title, or an explicitly named unnumbered platform scenario>`
- **Owner:** `<accountable team>`
- **Version:** `<version>`
- **Effective date:** `<date or pending approval>`
- **Status:** `<local baseline, approved, or deferred>`
- **Audience:** `<authorized operator audience>`
- **Data classification:** `<classification and evidence restrictions>`
- **Related dashboard:** `<dashboard contract or none>`
- **Related alerts:** `<alert conditions or none>`

## Prerequisites

List the authorization, service state, support reference, dependency access,
maintenance approval, synthetic-data requirement, and other conditions needed
before taking action. Do not request or record secrets, credentials, tokens,
raw payloads, or unrestricted financial identifiers.

## Detection

Describe the typed signal, support reference, dashboard state, health result,
or operator report that starts the procedure. Distinguish unavailable, stale,
degraded, pending, failed, and integrity-uncertain states.

## Decision points

State the stop conditions and branching decisions. If correctness, ownership,
or authoritative evidence is uncertain, contain the affected work and
escalate rather than guessing or retrying indefinitely.

## Safe commands

List only repository-supported, read-only, synthetic, or explicitly approved
forward-fix commands. Identify the environment and approval required for any
state-changing command. If the required operational capability does not exist,
the safe action is to preserve evidence and escalate; do not invent a CLI or
give direct database repair instructions.

## Evidence to preserve

Record the commit/build identifier, environment, time window, sanitized support
reference, typed state, bounded metric values, command result, owner, action,
and result. Exclude secrets, credentials, tokens, connection strings, raw
request/event/response payloads, SQL text, bank details, payroll values, tax
identifiers, and unrestricted record identifiers.

## Escalation

Identify the first responder, capability or dependency owner, Incident
Commander path, response target, and the condition that requires escalation.

## Recovery checks

Define the technical checks that show the failure is contained and the service
is safe to resume. Recovery is not complete merely because a process is up or
telemetry is green.

## Reconciliation requirements

Identify the authoritative records, pending work, source watermarks, control
totals, audit evidence, or external outcomes that must be reconciled. A
replayed, restored, or retried operation must not create a duplicate effect.

## Closure criteria

List the evidence, approvals, residual exceptions, owner assignments,
communication, and review needed before closing the incident. Unresolved
differences remain assigned exceptions; they are not hidden by closing an
alert.

## Deferred qualification

State which production, Azure, paging, quarterly, RTO/RPO, finance-domain, or
external-provider qualification is not provided by the local runbook. Never
present a local contract check as production evidence.

## Mandatory safety rules

Every runbook must explicitly preserve these boundaries:

- Direct destructive financial edits are prohibited; established facts are
  corrected through their owning domain operation.
- Unreviewed portal changes are prohibited; infrastructure ownership remains
  with the approved Terraform workflow.
- Secret disclosure is prohibited; diagnostics contain only bounded,
  sanitized references.
- Telemetry is diagnostic and is not authoritative financial, audit, posting,
  settlement, reconciliation, or recovery evidence.
- Retriable state-changing operations use their existing idempotency,
  establishment, lease, and reconciliation rules.
