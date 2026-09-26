# Identity & access

TALLY separates enterprise authentication from application-owned finance
authorization. Microsoft Entra ID can establish the external subject; the
Identity & Access bounded context decides whether a particular operation is
allowed in a particular accounting/business scope and records the decision
metadata needed by material actions.

## Authentication boundary

The approved flow is authorization-code with PKCE in the SPA and bearer JWT
validation at the API. The API validates trusted OIDC discovery metadata,
issuer, tenant, audience, signature, algorithm, expiry, and not-before time.
It maps the validated `oid`, `tid`, and `sub` claims to an application user and
actor. TALLY does not store passwords or replace Entra MFA and conditional
access.

Local fixture authentication is explicitly development-only. Missing or
invalid protected-route configuration fails closed; `/health/live` is the
anonymous liveness exception.

## Authorization boundary

The identity module owns `User`, `Role`, `AccessPolicy`, `SegregationRule`, and
`EmergencyAccessGrant`. A decision evaluates only the dimensions applicable to
the operation, which can include:

- legal entity, ledger, accounting book, business unit, or segment;
- account or account class, transaction type, amount, currency, or fiscal
  period;
- data sensitivity, requested action, actor history, and emergency grant;
- role/permission version and segregation-of-duties rules.

Deny is the default. A decision carries a safe outcome, permission/policy
version, decision reference, and permitted next action. The SPA may hide an
action for usability, but the API re-evaluates authorization at the mutation
boundary.

## IAM delivery slices

| Slice | Current behavior |
| --- | --- |
| Users | Create, activate, suspend, terminate, and replace complete assignment sets with expected-version, idempotency, containment, masking, and audit references. |
| Roles | Versioned explicit permission grants and opaque scope constraints; retirement is non-destructive and historical revisions remain available. |
| Policies | Durable policy reads and operation-specific scoped evaluation with distinct allowed, denied, expired, stale, and unavailable outcomes. |
| Segregation | Versioned IAM-owned rules, minimum v1 conflicts, fail-closed stale/unavailable/history checks, and safe explanation states. |
| Emergency access | Independent approval and assurance ports, reason/scope/expiry, four-hour default cap, expiry/revocation denial, and post-use review metadata. |
| Sensitive evidence | Metadata-only access observations, safe decision projections, masked restricted values, and audited, permission-filtered export context. |

These are IAM-owned boundaries and synthetic/local evidence. They do not create
finance-domain aggregates or bypass their ownership.

## Emergency access is temporary, not a back door

An emergency grant is a separate lifecycle with actor, permissions/scopes,
reason, start, expiry, approval reference, authentication assurance, grant
reference, and review state. Expired or revoked grants are denied even if
cleanup is delayed. A default four-hour cap and post-use review keep urgent
access attributable and bounded.

## Sensitive data and evidence

Sensitive values are minimized in views, errors, events, logs, and exports.
The current evidence contract permits stable references, classifications,
decision/policy/grant references, correlation/causation, and fingerprints; it
rejects raw subjects, tokens, credentials, policy payloads, and protected
values. A reveal or export attempt requires safe authorization metadata and an
audit callback. If that callback fails, the value remains masked and the
export does not run.

Operational telemetry remains separate from authoritative audit evidence. IAM
uses ports to the future Audit Integrity owner rather than writing another
module’s audit schema directly.

## Verification boundary

Focused local evidence exists for [authenticated application identity](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-IAM-001-us1-authenticated-application-identity.md),
[users and assignments](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-IAM-001-us2-user-access-assignments.md),
[roles](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-IAM-002-us3-role-permission-grants.md),
[scoped policies](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-IAM-003-us4-scoped-access-policies.md),
[segregation of duties](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-IAM-004-us5-segregation-of-duties.md),
[emergency access](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-IAM-005-us6-emergency-access.md),
and [sensitive-access evidence](https://github.com/toanle88/Tally/blob/codex/iam-us7-sensitive-access-evidence/docs/verification/DLV-IAM-006-us7-sensitive-access-evidence.md).

The records explicitly retain limitations: some frontend/database wrappers
were blocked by host tooling, live Entra and production controls are not
qualified, and finance-action enforcement waits for the owning bounded
contexts.
