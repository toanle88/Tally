# Operations & delivery

TALLY treats observability, recovery, and delivery controls as part of the
finance platform rather than as an afterthought. The current implementation is
a local and CI learning baseline; it does not claim a production operating
service.

## Signals and evidence

The platform carries trace, span, correlation, and causation references across
HTTP requests, commands, database transactions, outbox messages, and external
calls. Structured `slog` records use bounded fields such as service, module,
operation, actor/scope references, aggregate version, result, error code,
retryability, and data classification.

Logs, traces, and metrics exclude tokens, credentials, complete bank details,
payroll values, unrestricted tax identifiers, and unrestricted request/event
bodies. They describe operations; they do not replace append-only audit
evidence.

## Operational surfaces

The design and current shared UI cover:

- API/latency, database, worker, outbox/inbox, and dependency health;
- journal/posting, approval, settlement/reconciliation, period-close, and
  integrity signals when those capability modules exist;
- backlog age, retry/poison states, business exceptions, alert ownership, and
  runbook links;
- cross-context event exception and concurrency-conflict surfaces that route
  resolution to the owning capability instead of becoming a second mutation
  surface.

The current web app includes synthetic operational examples and explicit
placeholder states for capability routes. It does not pretend that a dashboard
panel is a live finance record.

## Durable coordination

The Go worker foundation demonstrates:

1. transactionally persisted outbox and inbox identity;
2. lease-safe claim, renewal, owner fencing, and rescheduling;
3. typed retry delays and managed-exception retention;
4. bounded worker admission and graceful lifecycle/shutdown behavior;
5. duplicate delivery, ordering/gap, crash/restart, and generation-scoped
   replay behavior.

The initial worker composition fails safely when no capability consumer registry
exists. It does not claim business work that has no registered owner.

## Runbooks and alert posture

The approved runbook set covers failed migrations, outbox backlog/poison items,
database restore and reconciliation, provider outage, period-control
interruption, audit mismatch, Entra/authorization outage, Terraform state
recovery, credential rotation, and capacity saturation. Alerts are bounded by
severity:

| Severity | Example | Expected posture |
| --- | --- | --- |
| P1 | Possible duplicate financial effect, audit mismatch, lost posting evidence, broad Class A outage | Immediate page; block affected writes when integrity is uncertain |
| P2 | Outbox age or close/payment backlog breach, database saturation | Urgent investigation and controlled degradation |
| P3 | Single provider failure, report delay, elevated conflict rate | Business-hours response with visible status |
| P4 | Capacity forecast or noncritical warning | Planned remediation |

## Terraform and CI/CD

Terraform is organized around repository boundaries, reusable low-cost modules,
separate dev/demo/prod-reference profiles, protected remote state, budgets,
plan/policy/drift/cost/security checks, and safe destroy wrappers. GitHub
Actions uses OIDC federation in the approved design; long-lived cloud secrets
are not committed.

The Azure learning profile is deliberately disposable: scale-to-zero/small
resources reduce cost, and live deployment is optional. The production
reference profile requires multiple replicas, private/HA database posture,
backup/restore, load, and operational qualification that are not satisfied by
the learning profile.

## Verification boundary

Representative evidence is available for [telemetry context and redaction](https://github.com/toanle88/Tally/tree/main/docs/verification),
[traces and metrics](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-OPS-001-us3-traces-bounded-platform-metrics.md),
[operational readiness](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-OPS-002-us4-operational-readiness-evidence.md),
[worker lifecycle and replay](https://github.com/toanle88/Tally/blob/main/docs/verification/DLV-PLAT-007-worker-lifecycle-replay.md),
and [Terraform/Azure controls](https://github.com/toanle88/Tally/tree/main/docs/verification).

Local checks may be blocked by Docker, network, Node, Azure credentials, or
host process integration. Each evidence record reports the limitation instead
of upgrading an unavailable check to “passed.”
