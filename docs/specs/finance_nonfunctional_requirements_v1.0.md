# Finance Platform Nonfunctional Requirements Specification
| Document-control field | Value |
|---|---|
| Version | 1.0 |
| Baseline date | 2026-07-24 |
| Status | Consistency-verified NFR baseline |
| Source domain baseline | Finance Domain Model & Use Cases — DDD Baseline v3.1 |
| Source functional baseline | Finance Functional PRD and Requirements Pack v1.5 |
| Source UX baseline | Finance UX and Workflow Specification v1.0 |
| Document owner | Finance Product, Architecture, Security, and Operations |
| Intended audience | Product, Finance SMEs, UX, QA, Architecture, Engineering, Security, Privacy, Compliance, Operations, Support, and Audit |
> **Purpose:** Define measurable quality, control, resilience, security, privacy, accessibility, interoperability, and operational expectations for the finance platform without prescribing application architecture or infrastructure products.
>
> **Target status:** Numerical targets in this specification are the baseline qualification and acceptance targets. A contract, jurisdiction, or approved service agreement may impose stricter targets. A weaker target requires an explicit change to this baseline with accountable approval and impact analysis.
>
> **Source authority:** DDD v3.1 remains authoritative for domain semantics and invariants; Functional PRD v1.5 remains authoritative for required product behavior; UX v1.0 remains authoritative for user interaction. This document defines how well those behaviors must operate and must not add new domain facts or functional actions.
<a id="table-of-contents"></a>
## Table of Contents

- [1. Purpose, Scope, and Non-Goals](#section-1)
- [2. Requirement Conventions and Measurement](#section-2)
- [3. Service Classes and Qualification Profile](#section-3)
- [4. Nonfunctional Requirements](#section-4)
  - [4.1 Performance and Responsiveness](#nfr-perf)
  - [4.2 Capacity and Scalability](#nfr-cap)
  - [4.3 Availability and Service Continuity](#nfr-avl)
  - [4.4 Reliability, Data Integrity, and Consistency](#nfr-rel)
  - [4.5 Security, Identity, and Access Control](#nfr-sec)
  - [4.6 Privacy, Retention, and Legal Hold](#nfr-prv)
  - [4.7 Auditability, Evidence, and Nonrepudiation](#nfr-aud)
  - [4.8 Resilience, Backup, and Disaster Recovery](#nfr-rec)
  - [4.9 Observability, Operations, and Supportability](#nfr-obs)
  - [4.10 Accessibility and Inclusive Use](#nfr-acc)
  - [4.11 Compatibility and Client Quality](#nfr-cmp)
  - [4.12 Localization, Currency, Date, and Language Quality](#nfr-loc)
  - [4.13 Interoperability and External Dependency Quality](#nfr-int)
  - [4.14 Maintainability, Change Safety, and Operability](#nfr-mnt)
  - [4.15 Verification, Testing, and Release Quality](#nfr-tst)
- [5. Capability Applicability Matrix](#section-5)
- [6. Workflow Criticality and Recovery Matrix](#section-6)
- [7. Verification and Acceptance Strategy](#section-7)
- [8. Assumptions, Exceptions, and Change Control](#section-8)
- [9. Verification Checkpoint](#section-9)
<a id="section-1"></a>
## 1. Purpose, Scope, and Non-Goals

### 1.1 In scope

This specification covers measurable expectations for:
- Interactive and long-running performance.
- Capacity, growth, and controlled overload.
- Availability, resilience, recovery, and continuity.
- Financial correctness, consistency, precision, and evidence.
- Identity, authorization, segregation of duties, security, privacy, retention, and legal hold.
- Audit integrity, observability, supportability, accessibility, compatibility, localization, interoperability, maintainability, and testability.
- All 19 capabilities, 22 functional workflows, 193 capability requirements, 22 global requirements, 199 functional acceptance scenarios, and 99 UX acceptance criteria in the verified source baselines.

### 1.2 Non-goals

This document does not choose:
- Application, service, storage, messaging, analytics, deployment, or network architecture.
- Cloud, database, queue, identity, monitoring, backup, cryptographic, or vendor products.
- API or event schemas, database indexes, topology, implementation language, or framework.
- Staffing model, commercial support tier, release plan, migration design, or detailed operational runbook.

Those decisions must demonstrate compliance with this specification in later solution and delivery artifacts.
<a id="section-2"></a>
## 2. Requirement Conventions and Measurement

- `Shall` means mandatory.
- Percentiles use completed eligible observations in the declared measurement window; p95 and p99 are not averages.
- External-provider elapsed time is excluded only where a requirement explicitly says so. Local acceptance, pending-state visibility, timeout, and exception handling remain measured.
- Availability is measured from representative user or business outcomes, not process uptime alone.
- The baseline service window is 24x7 in the applicable business time zones unless an approved contract defines a narrower window; critical finance windows may define stricter objectives.
- An acknowledged authoritative state is a business outcome that the product has presented as established with an authoritative identity or reference.
- A requirement may be stricter for a capability or jurisdiction; the strictest applicable requirement governs.
- Every exception requires owner, rationale, affected scope, risk, compensating control, approval, expiry, and retest date.
- Measurement evidence must preserve environment, data profile, workload, versions, time range, exclusions, and result distribution.

### 2.1 Verification methods

| Method | Meaning |
|---|---|
| Analysis | Review of design, configuration, policy, traceability, or calculated capacity. |
| Inspection | Direct examination of product behavior, evidence, records, or documentation. |
| Test | Controlled functional or nonfunctional execution with expected measurable result. |
| Exercise | Coordinated operational, security, recovery, or incident simulation. |
| Monitoring | Production measurement over a declared window. |
| Reconciliation | Comparison of authoritative totals, identities, lineage, or evidence with zero unexplained difference. |
<a id="section-3"></a>
## 3. Service Classes and Qualification Profile

### 3.1 Service classes

| Class | Definition | Examples | Baseline availability | Baseline RTO |
|---|---|---|---:|---:|
| A | Authoritative financial-control or high-risk state-changing function. | Posting, approval application, payment, receipt application, filing, close/reopen, correction, audit proof. | 99.95% monthly | 2 hours |
| B | Interactive operational maintenance, review, search, and exception resolution. | Worklists, record detail, master data, invoice review, operational reporting. | 99.90% monthly | 4 hours |
| C | Analytical, import, export, scheduled, or large-volume processing with visible progress. | Consolidation, statements, revaluation, bank import, depreciation, evidence package. | 99.50% monthly | 8 hours |
| D | Administration and reference configuration. | Access, policies, calendars, charts, mappings, connections, templates. | 99.90% monthly | 8 hours |

One capability may contain more than one service class. The stricter class applies to a workflow step that establishes or controls authoritative financial state.

### 3.2 Baseline qualification profile

| Dimension | Baseline |
|---|---:|
| Concurrent active users | 1,000 |
| Sustained authoritative state changes | 250 per second for 15 minutes |
| Burst authoritative state changes | 500 per second for 60 seconds |
| Sustained read/search/worklist requests | 2,500 per second for 30 minutes |
| Journal lines per business day | 10,000,000 |
| Lines in one journal set | 1,000,000 |
| Payment instructions in one batch | 100,000 |
| Bank-statement lines in one import | 1,000,000 |
| Open AP or AR items per accounting scope | 10,000,000 |
| Historical AP or AR items per accounting scope | 50,000,000 |
| Open approval tasks per accounting scope | 100,000 |
| Annual capacity growth allowance | 20% for three years |

The profile is a qualification floor, not a forecast or hard product limit. Capacity planning must replace it with higher approved values when forecast or contract requires them.
<a id="section-4"></a>
## 4. Nonfunctional Requirements
<a id="nfr-perf"></a>
### 4.1 Performance and Responsiveness

Defines measurable response, completion, freshness, and progress-reporting expectations under the baseline qualification profile.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-PERF-001` | Class A interactive validation shall complete within 2 seconds at p95 and 5 seconds at p99, excluding elapsed time controlled by an unavailable external authority or provider. | Measure from action submission to presentation of validated, rejected, conflict, or dependency-unavailable outcome over rolling 30-day production windows and qualification tests. | GFR-009, GFR-017; UX §§5, 9; WF-6.1–WF-7.15 |
| `NFR-PERF-002` | Class A authoritative state-change acknowledgement shall complete within 3 seconds at p95 and 8 seconds at p99 when no external outcome is required. | Measure from accepted submission to authoritative result reference and lifecycle state becoming visible to the initiating user. | GFR-005–GFR-009, GFR-012; DDD §§3, 4, 9 |
| `NFR-PERF-003` | When a Class A action depends on an external outcome, local acceptance or rejection shall be visible within 3 seconds at p95, and the external-pending state shall become visible within 5 seconds. | Measure separately from external-provider completion time; no external dependency time may be reported as local processing time. | GFR-012, GFR-017; FR-PCM-*, FR-BFR-*, FR-TAX-*, FR-PAYR-* |
| `NFR-PERF-004` | Standard record-detail views shall render primary facts, state, ownership, permitted actions, and first-page lineage within 2.5 seconds at p95 and 6 seconds at p99. | Test records with up to 10,000 lineage items, using bounded first-page retrieval and visible continuation. | GFR-009, GFR-013, GFR-014; UX §4.3 |
| `NFR-PERF-005` | Standard filtered worklists shall display the first page within 3 seconds at p95 and 7 seconds at p99 for a result population of at least 10 million records. | Measure with representative filters for scope, state, owner, date, amount, currency, exception, and approval status. | GFR-016; UX §§4.1, 7 |
| `NFR-PERF-006` | Cross-capability and global record search shall return the first page within 4 seconds at p95 and 10 seconds at p99 for the baseline online-data profile. | Qualification shall include exact identifier, source reference, party, amount/date range, and authorized full-text searches. | GFR-016; FR-OMD-*, FR-GL-*, FR-AP-*, FR-AR-*, FR-AUD-* |
| `NFR-PERF-007` | Approval inbox, approval subject, and decision views shall render within 2 seconds at p95 and 5 seconds at p99. | Measure with at least 100,000 open approval tasks and all applicable segregation checks enabled. | GFR-002–GFR-004; FR-WFA-*, UX §5.1 |
| `NFR-PERF-008` | Bulk or long-running actions shall acknowledge accepted work within 5 seconds and refresh visible progress at least once every 60 seconds while work remains active. | Verify that progress includes stage, completed/total counts where meaningful, last update, owner, blockers, and cancellation eligibility. | GFR-009, GFR-012, GFR-017; WF-6.1, WF-7.2, WF-7.4, WF-7.9 |
| `NFR-PERF-009` | A standard operational report over one accounting scope and period shall complete within 30 seconds at p95; a standard ledger or statutory financial statement shall complete within 5 minutes at p95. | Qualification uses the baseline reporting profile and records the definition version and source watermarks. | FR-RPT-005, FR-RPT-006; GFR-010, GFR-020 |
| `NFR-PERF-010` | A consolidation, period-end revaluation, depreciation run, or hard-close calculation over the baseline period-end profile shall complete within 4 hours, excluding unresolved business approvals or externally unavailable evidence. | Measure end-to-end processing time by stage and separately report blocked business time. | WF-6.1, WF-7.6, WF-7.7, WF-7.9; FR-FPM-*, FR-FX-*, FR-FA-*, FR-RPT-* |
| `NFR-PERF-011` | A payment batch containing 100,000 instructions shall complete internal validation within 15 minutes and reach an externally pending or internally terminal state within 60 minutes, excluding provider settlement time. | Verify validation exceptions are itemized and successful instructions are not delayed by unrelated invalid instructions unless policy requires whole-batch rejection. | WF-7.2; FR-PCM-001–FR-PCM-009 |
| `NFR-PERF-012` | Import of a bank statement containing 1,000,000 lines shall complete validation and make accepted lines available for matching within 2 hours. | Measure rejected-line reporting, duplicate detection, balance checks, and progress visibility under the same target. | WF-7.4; FR-BFR-001–FR-BFR-006 |
| `NFR-PERF-013` | User-visible notifications for assigned work, approval, rejection, conflict, material exception, completion, and integrity incident shall be issued within 2 minutes of the authoritative triggering outcome. | Measure from authoritative outcome time to notification availability; external delivery-channel delay is reported separately. | GFR-009, GFR-014, GFR-017; UX §9.1 |
| `NFR-PERF-014` | Dependent workspaces shall reflect a published cross-capability business outcome within 30 seconds at p95 and 5 minutes at p99. | Measure from authoritative publication to first authorized dependent view; exceptions beyond p99 shall be visible and alertable. | GFR-012–GFR-014; WF-7.13 |
<a id="nfr-cap"></a>
### 4.2 Capacity and Scalability

Defines the minimum qualification volume, growth tolerance, and degradation behavior without prescribing deployment topology.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-CAP-001` | The production service shall support at least 1,000 concurrently active authenticated users while meeting Class A and Class B performance targets. | Qualification shall use representative role, scope, search, approval, posting, reporting, and exception workloads rather than idle sessions. | All capabilities; UX §3 |
| `NFR-CAP-002` | The service shall sustain at least 250 authoritative state-changing operations per second for 15 minutes and a burst of 500 per second for 60 seconds without duplicate financial effects or invariant violations. | Qualification mixes journal, approval, receipt, payment, reconciliation, correction, and administrative changes. | GFR-005–GFR-009, GFR-011–GFR-014 |
| `NFR-CAP-003` | The service shall sustain at least 2,500 read, search, worklist, evidence, and reporting requests per second for 30 minutes while meeting the applicable performance targets. | Qualification includes permission filtering and realistic data distribution. | GFR-015, GFR-016, GFR-020 |
| `NFR-CAP-004` | The service shall process at least 10 million journal lines per business day and retain exact posting, source, approval, currency, period, and correction lineage. | Daily qualification totals shall reconcile to authoritative ledger counts and values. | FR-GL-*; DDD §§2.2, 3.1, 5.1 |
| `NFR-CAP-005` | A single journal import or generated posting set shall support at least 1,000,000 lines, with validation errors returned at line and aggregate level without truncation. | Qualification shall include balanced, unbalanced, mixed-currency, invalid-account, and period-gate cases. | FR-GL-001, FR-GL-002; GFR-010, GFR-011 |
| `NFR-CAP-006` | Each accounting scope shall support at least 10 million open AP and AR items and 50 million historical items while meeting worklist and direct-identifier lookup targets. | Qualification shall include aged, partially settled, disputed, corrected, and closed items. | FR-AP-*, FR-AR-*; GFR-016 |
| `NFR-CAP-007` | The service shall support at least 100,000 open approval tasks and 1 million completed decisions per accounting scope without violating decision uniqueness or application revalidation. | Qualification shall include delegation, escalation, expiry, and segregation checks. | FR-WFA-*; GFR-003, GFR-004 |
| `NFR-CAP-008` | The baseline capacity shall permit 20 percent annual growth in users, records, evidence, and daily operations for three years without changing domain semantics or reducing agreed service objectives. | Capacity reviews shall be performed at least quarterly and before forecast utilization exceeds 70 percent of any approved limit. | All capabilities; GFR-001, GFR-013 |
| `NFR-CAP-009` | When a declared capacity limit is approached, the product shall preserve Class A correctness and expose controlled throttling, backlog, or deferred completion rather than silent loss, partial financial effects, or misleading success. | Qualification shall exceed each baseline limit by at least 25 percent and verify typed outcomes and recovery. | GFR-009, GFR-012, GFR-017; DDD §9 |
| `NFR-CAP-010` | Capacity limits, current utilization, forecast exhaustion date, and affected service class shall be visible to authorized operations and product owners. | Verify monthly capacity reporting and alerts at 70, 80, and 90 percent of approved limits. | NFR-OBS-*; all capabilities |
<a id="nfr-avl"></a>
### 4.3 Availability and Service Continuity

Defines service objectives, maintenance constraints, and transparent degraded operation.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-AVL-001` | Class A financial-control functions shall achieve 99.95 percent monthly availability during the declared service window. | Availability is measured by successful completion of representative Class A synthetic and real-user transactions; planned maintenance counts unless it meets NFR-AVL-005. | WF-6.1–WF-7.15; FR-GL-*, FR-PCM-*, FR-FPM-*, FR-WFA-* |
| `NFR-AVL-002` | Class B interactive operational functions shall achieve 99.90 percent monthly availability during the declared service window. | Measure record, worklist, search, maintenance, and exception-resolution paths by capability and aggregate. | All operational capabilities |
| `NFR-AVL-003` | Class C analytical and scheduled functions shall achieve 99.50 percent monthly availability and meet declared completion windows. | Measure report, consolidation, revaluation, depreciation, import, export, and verification functions separately from interactive availability. | FR-RPT-*, FR-FX-*, FR-FA-*, FR-BFR-*, FR-AUD-* |
| `NFR-AVL-004` | Class D administration and reference functions shall achieve 99.90 percent monthly availability, with no loss of previously established business-state access when administration is unavailable. | Verify that active business records remain usable under established configuration where policy permits. | FR-OMD-*, FR-COA-*, FR-IAM-*, FR-WFA-* |
| `NFR-AVL-005` | Planned maintenance shall be excluded from availability only when communicated at least 7 calendar days in advance, limited to 4 hours per month, and accompanied by affected functions, time zone, recovery plan, and user-visible status. | Audit maintenance notices and actual windows; overruns count as unavailable time. | GFR-009, GFR-014, GFR-017 |
| `NFR-AVL-006` | A dependency failure shall not cause unrelated capabilities or accounting scopes to become unavailable when their domain rules do not require that dependency. | Test provider, authority, reporting, notification, and reference-data failures independently. | GFR-012, GFR-017; DDD §§1.2, 9.4 |
| `NFR-AVL-007` | Degraded, read-only, pending, or unavailable states shall be identified to affected users within 60 seconds and shall never be presented as business success or rejection. | Verify status messaging, blocked actions, owner, and recovery path for each service class. | GFR-009, GFR-012, GFR-017; UX §9.2 |
| `NFR-AVL-008` | Availability shall be measured per capability, workflow, accounting scope, and service class so that a healthy aggregate metric cannot hide a failed critical workflow. | Monthly service reporting shall include numerator, denominator, exclusions, and top failure causes. | All 19 capabilities and 22 workflows |
| `NFR-AVL-009` | Close, payment, payroll, tax, and filing critical windows shall support temporary elevated service objectives approved at least 10 business days before the window. | The elevated window shall define staffing, monitoring, change restrictions, and recovery expectations without changing domain rules. | WF-6.1, WF-7.2, WF-7.10, WF-7.11 |
| `NFR-AVL-010` | Service-status history and material availability incidents shall remain accessible to authorized stakeholders for at least 24 months. | Verify incident start/end, affected functions, impact, root cause category, and corrective action are retained. | NFR-OBS-*; GFR-014 |
<a id="nfr-rel"></a>
### 4.4 Reliability, Data Integrity, and Consistency

Defines correctness guarantees for financial facts, lifecycle transitions, cross-capability outcomes, precision, and reconciliation.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-REL-001` | Every acknowledged authoritative financial state change shall be durable and shall not be lost after process, host, zone, or regional recovery. | Verify by fault injection immediately before, during, and after acknowledgement. | GFR-005, GFR-013; DDD §§3, 9 |
| `NFR-REL-002` | Repeated observation or submission of an identical business identity and fingerprint shall produce exactly one business effect and the established result. | Execute duplicate tests at least 100 times per critical command and across recovery boundaries. | GFR-006; DDD §§3.8, 9.3, 14.9, 14.13.13 |
| `NFR-REL-003` | Reuse of an identity with changed business content shall produce an explicit conflict and no additional financial effect. | Verify conflict evidence includes established identity, fingerprint/result reference, and required new-action path. | GFR-007; DDD §9.3 |
| `NFR-REL-004` | Concurrent noncommutative changes shall never silently overwrite one another and shall establish either one valid outcome or a typed conflict with no partial effect. | Run race tests for every multi-aggregate and ownership-transfer rule defined by DDD §9.1. | GFR-008, GFR-009; DDD §§7.14, 9.1, 14.13.14 |
| `NFR-REL-005` | One hundred percent of posted journals shall satisfy debit-credit balance, accounting-scope, period-gate, currency, account, and segment invariants before becoming authoritative. | Reconcile journal headers, lines, gate evidence, and source references continuously and during qualification. | FR-GL-*; DDD §§3.1, 5.1–5.3 |
| `NFR-REL-006` | Monetary calculations shall use exact decimal semantics at the currency and policy precision defined by the domain, with no silent truncation or binary-floating approximation in authoritative amounts. | Test minor-unit, high-precision rate, rounding boundary, negative correction, and aggregate-total cases. | GFR-010; DDD §10 |
| `NFR-REL-007` | All cumulative payment, return, refund, receipt, expectation, asset-settlement, and reconciliation equations defined in DDD §§2–3 shall hold after every authoritative transition. | Execute invariant checks during qualification, reconciliation, and controlled production monitoring. | FR-PCM-*, FR-AR-*, FR-FA-*; DDD §§2.4, 2.7, 2.11, 3.2, 3.6, 3.7 |
| `NFR-REL-008` | Established source facts and their reversal, return, amendment, unapplication, replacement, or compensation lineage shall remain independently retrievable for the full retention period. | Verify bidirectional navigation and totals before and after correction. | GFR-005, GFR-013; DDD §§4, 9.4, 11 |
| `NFR-REL-009` | A cross-capability outcome shall identify exactly one authoritative owner, and dependent projections shall never create a second authoritative accounting or settlement fact. | Reconcile published outcome counts and values by source identity and owner. | GFR-011, GFR-012; DDD §§1.1–1.3 |
| `NFR-REL-010` | Partial success shall be represented at item, leg, or obligation level and shall not be collapsed into whole-process success or failure. | Verify payment, disposal, close, receipt, consolidation, and filing workflows with mixed outcomes. | GFR-009, GFR-012; WF-6.1, WF-6.4, WF-7.2, WF-7.7, WF-7.9, WF-7.10 |
| `NFR-REL-011` | Dependent views shall preserve source identity, source version, accounting scope, amount, currency, and authoritative outcome reference without reinterpretation. | Compare authoritative and dependent records for 100 percent of sampled cross-capability outcomes. | GFR-012–GFR-014; WF-7.13 |
| `NFR-REL-012` | Financial statements and reports shall retain the exact definition version, source watermarks, accounting or consolidation scope, currency basis, and generation time used. | Verify that regeneration from unchanged inputs reproduces the same authoritative values. | FR-RPT-005, FR-RPT-006; DDD §2.8 |
| `NFR-REL-013` | Business timestamps shall preserve an unambiguous UTC instant and the applicable business time zone or local date meaning; daylight-saving changes shall not duplicate or omit business periods. | Test time-zone boundaries, daylight-saving transitions, period cutoff, and legal-date rules. | DDD §10; all date-sensitive workflows |
| `NFR-REL-014` | Reference-data changes shall not alter the rule, rate, account, mapping, policy, or configuration version retained by historical transactions. | Verify before/after reporting and correction behavior after configuration changes. | GFR-019; DDD §§2, 10 |
| `NFR-REL-015` | Every import, export, reconciliation, and migration-equivalent data movement shall provide record counts, accepted/rejected counts, control totals where applicable, and a deterministic reconciliation result. | Qualification shall prove no silent record omission or duplication. | GFR-014, GFR-020; FR-BFR-*, FR-GL-*, FR-RPT-* |
| `NFR-REL-016` | The daily integrity control shall report zero unexplained differences among authoritative records, dependent outcomes, audit evidence, and declared control totals; any difference shall create an owned exception within 15 minutes. | Verify exception includes scope, record identities, amount/value difference, first occurrence, and resolution status. | GFR-009, GFR-012, GFR-014; NFR-OBS-* |
<a id="nfr-sec"></a>
### 4.5 Security, Identity, and Access Control

Defines identity assurance, least privilege, segregation, session protection, information protection, and vulnerability expectations.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-SEC-001` | All workforce users shall authenticate through an approved enterprise identity provider; local password authentication shall be disabled except for controlled emergency recovery identities. | Verify identity source, authentication assurance, and emergency-account inventory quarterly. | FR-IAM-*; GFR-002, GFR-003 |
| `NFR-SEC-002` | Multi-factor authentication shall be required for all users, and phishing-resistant authentication shall be required for privileged administration, payment release, period reopen, high-risk approval, and audit-integrity administration. | Verify at authentication and step-up points through positive and negative tests. | FR-IAM-*, FR-WFA-*, FR-PCM-*, FR-FPM-*, FR-AUD-* |
| `NFR-SEC-003` | High-risk actions shall require authentication assurance no older than 5 minutes or an explicit step-up challenge. | High-risk actions include payment release, bank-detail change, manual journal approval above threshold, period reopen, emergency access, sensitive export, and verification-credential rotation. | DDD §8; GFR-002–GFR-004 |
| `NFR-SEC-004` | Authorization shall use default-deny and least-privilege rules across the access dimensions applicable to each action. | Test every role against allowed and denied legal entity, segment, account, transaction, amount, currency, period, sensitivity, and action scopes. | GFR-002; DDD §8.1 |
| `NFR-SEC-005` | Prohibited segregation-of-duties combinations shall be prevented before the business action is established and shall expose a non-sensitive reason and permitted resolution path. | Test all minimum segregation rules from DDD §8.2 and configured extensions. | GFR-003; FR-IAM-004, FR-WFA-* |
| `NFR-SEC-006` | Access revocation, role removal, and emergency-access expiry shall take effect across interactive and noninteractive access within 15 minutes. | Measure from authoritative access-policy change to denied access in every affected capability. | FR-IAM-001–FR-IAM-006; DDD §8.3 |
| `NFR-SEC-007` | General interactive sessions shall expire after 30 minutes of inactivity and privileged or sensitive sessions after 15 minutes; users shall receive an accessible warning at least 2 minutes before expiry. | Verify unsaved work handling does not establish unauthorized actions after expiry. | UX §10.1; FR-IAM-* |
| `NFR-SEC-008` | Concurrent session and location anomalies shall be risk-evaluated and shall trigger step-up, termination, or security review according to approved policy. | Verify no sensitive action proceeds when the required assurance cannot be established. | FR-IAM-*, GFR-014 |
| `NFR-SEC-009` | Sensitive data shall be protected in transit and at rest using organization-approved cryptographic controls and key-management policy. | Verify coverage for business records, evidence, exports, backups, and administrative channels. | GFR-015, GFR-020; DDD §11.3 |
| `NFR-SEC-010` | Secrets, provider credentials, signing material, and bank tokens shall never appear in source records, user-visible errors, notifications, logs, exports, or support diagnostics. | Run automated secret scanning and manual negative tests before every release. | GFR-015; DDD §§1.1, 11.3 |
| `NFR-SEC-011` | Sensitive values shall be masked by default and revealed only through an explicit authorized action that is separately audited. | Test payroll, tax, bank, personal, and security-sensitive fields in views, comparisons, exports, notifications, and errors. | GFR-015; UX §10.2 |
| `NFR-SEC-012` | Every privileged, emergency, access-policy, segregation-rule, approval-policy, bank-detail, and verification-credential change shall be independently attributable and auditable. | Verify actor, approver where required, reason, scope, before/after fingerprints, and effective interval. | GFR-014; FR-IAM-*, FR-WFA-*, FR-OMD-002, FR-AUD-* |
| `NFR-SEC-013` | Emergency access shall be time-bound to no more than 4 hours per grant unless a stricter policy applies, reason-coded, approved where possible, and reviewed within 1 business day. | Verify automatic expiry and complete action review. | DDD §8.3; FR-IAM-005, FR-IAM-006 |
| `NFR-SEC-014` | Production data shall not be used in development or test environments unless irreversibly deidentified and explicitly approved. | Verify environment data inventories and sampling at least quarterly. | GFR-015, GFR-018 |
| `NFR-SEC-015` | Critical security vulnerabilities shall be remediated or formally risk-accepted within 7 calendar days and high-severity vulnerabilities within 30 calendar days. | Measure from validated finding time; exceptions require accountable owner, expiry, compensating controls, and approval. | NFR-MNT-*; all capabilities |
| `NFR-SEC-016` | Security testing shall include threat modeling, automated scanning, dependency review, penetration testing, and authorization-abuse testing before production release and at least annually thereafter. | Verify that no unresolved critical finding remains at release. | All capabilities; DDD §§8, 11 |
| `NFR-SEC-017` | Evidence exports and bulk downloads shall enforce the same row and field permissions as interactive views and shall record actor, scope, filters, purpose, and sensitivity classification. | Verify exports cannot infer or include unauthorized values. | GFR-015, GFR-020; UX §10.2 |
| `NFR-SEC-018` | Security events indicating credential compromise, privilege escalation, mass export, segregation bypass, or audit-integrity risk shall create a security incident alert within 5 minutes. | Verify event-to-alert latency and incident ownership through quarterly exercises. | FR-IAM-*, FR-AUD-*; NFR-OBS-* |
<a id="nfr-prv"></a>
### 4.6 Privacy, Retention, and Legal Hold

Defines data minimization, purpose limitation, retention, legal hold, residency, and safe handling of sensitive finance data.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-PRV-001` | Every stored and displayed field shall have an approved business purpose, data classification, authoritative owner, and retention category. | Review data inventory before release and after material field additions. | GFR-001, GFR-015, GFR-018; DDD §11 |
| `NFR-PRV-002` | Payroll, tax, bank, personal, and security-sensitive data shall be minimized in shared events, worklists, notifications, exports, and audit evidence. | Verify that identifiers or masked references replace full sensitive values wherever the business purpose permits. | GFR-015; DDD §11.3 |
| `NFR-PRV-003` | Users shall see only the minimum sensitive detail necessary for their authorized task, with field-level restrictions enforced independently from record-level access. | Test role combinations and indirect disclosure through filters, counts, comparisons, and errors. | GFR-002, GFR-015 |
| `NFR-PRV-004` | Retention shall be configurable by jurisdiction, record type, accounting scope, legal entity, and legal-hold status, and shall preserve all linked correction and audit lineage. | Verify retention decisions against declared policy and source relationships. | GFR-018; DDD §11.2 |
| `NFR-PRV-005` | A legal hold shall prevent destruction within 1 hour of authoritative hold application and shall remain effective until formal release. | Verify hold propagation to source records, evidence, exports, backups subject to policy, and scheduled destruction. | GFR-018; UX §10.3 |
| `NFR-PRV-006` | Privacy-access, correction, restriction, or deletion requests shall be resolved according to jurisdiction without altering legally required financial facts or audit evidence. | Verify that denied deletion cites the lawful retention basis and that permitted corrections use linked lineage. | GFR-005, GFR-013, GFR-018 |
| `NFR-PRV-007` | Data residency and cross-border transfer restrictions shall be enforceable by tenant, legal entity, jurisdiction, and data classification where contract or law requires them. | Qualification shall verify storage, processing, export, support, and recovery locations against approved policy. | GFR-002, GFR-015 |
| `NFR-PRV-008` | Sensitive data in nonproduction diagnostics, demonstrations, training, and support material shall be synthetic or irreversibly deidentified. | Audit samples quarterly and after incident data capture. | NFR-SEC-014; GFR-015 |
| `NFR-PRV-009` | Notifications shall contain no full bank account, tax identifier, payroll detail, personal address, secret, or verification credential. | Automated tests shall scan notification content and templates before release. | GFR-015; UX §9.1 |
| `NFR-PRV-010` | Approved destruction shall be irreversible, scoped, evidenced, and reconciled to policy, while preserving required destruction evidence without retaining the destroyed content. | Verify counts, scope, approval, hold checks, completion time, and exception reporting. | DDD §11.2; GFR-014, GFR-018 |
<a id="nfr-aud"></a>
### 4.7 Auditability, Evidence, and Nonrepudiation

Defines completeness, timeliness, integrity, searchability, proof, and export expectations for business and audit evidence.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-AUD-001` | One hundred percent of material business actions, decisions, state transitions, corrections, access changes, exports, and administrative overrides shall create audit evidence. | Reconcile authoritative action counts to audit-evidence counts daily with zero unexplained omissions. | GFR-014; DDD §11.1 |
| `NFR-AUD-002` | Audit evidence shall become queryable within 5 seconds of the authoritative business outcome at p95 and within 30 seconds at p99. | Measure by event class, capability, and accounting scope. | GFR-014; FR-AUD-001 |
| `NFR-AUD-003` | Each audit record shall include event identity, occurred and recorded times, actor and authentication subject, source context and aggregate reference, action type, correlation and causation, authorization or approval reference, classification, and integrity evidence required by DDD §11.1. | Verify 100 percent schema conformance for required fields. | DDD §11.1; FR-AUD-001 |
| `NFR-AUD-004` | Audit history shall be append-only from the user and business perspective; corrections shall create linked evidence and never replace prior evidence. | Verify through authorized and unauthorized modification attempts. | GFR-005, GFR-013, GFR-014; DDD §§2.19, 11 |
| `NFR-AUD-005` | Audit-integrity verification shall return one typed result and shall detect missing sequence, mismatched proof, unsupported version, invalid proof, and credential-interval boundary cases. | Execute all DDD §14.13.15 normal, boundary, concurrency, duplicate, and recovery cases. | FR-AUD-002–FR-AUD-005; WF-7.15 |
| `NFR-AUD-006` | Clock difference among authoritative business and audit sources shall not exceed 1 second, and any violation shall create an operations alert within 5 minutes. | Monitor continuously and test recovery from time-source loss. | DDD §11.1; NFR-OBS-* |
| `NFR-AUD-007` | Authorized users shall retrieve audit evidence by business identifier, actor, action, scope, date range, correlation, approval, posting, settlement, correction, or incident reference within 5 seconds at p95 for the baseline online period. | Verify permission filtering and masked fields. | GFR-016, GFR-020; FR-AUD-* |
| `NFR-AUD-008` | A standard evidence package for one workflow instance shall be generated within 5 minutes and shall identify source facts, approvals, authoritative effects, corrections, scope, versions, timestamps, filters, and sensitivity classification. | Test every WF-6.x and WF-7.x workflow. | GFR-013, GFR-020; UX §10.3 |
| `NFR-AUD-009` | Evidence exports shall be integrity-verifiable and shall disclose whether any requested evidence was excluded because of access, retention, legal hold, or unsupported scope. | Verify export and subsequent verification under unchanged and altered-content cases. | GFR-015, GFR-020; FR-AUD-* |
| `NFR-AUD-010` | Audit and evidence access shall itself be audited, including search, view, reveal, export, verification, incident action, and credential rotation. | Reconcile privileged audit-access activity monthly. | GFR-014, GFR-015; FR-AUD-*, FR-IAM-* |
| `NFR-AUD-011` | Audit-integrity incidents shall preserve affected evidence, typed verification result, severity, containment, owner, business impact, and recovery history for the applicable retention period. | Verify incident closure requires approved resolution and post-incident review. | WF-7.15; FR-AUD-004 |
| `NFR-AUD-012` | Legal hold and retention operations shall produce independently searchable evidence of policy, scope, approval, application, release, destruction, and exceptions. | Verify evidence remains after permitted content destruction. | GFR-018; NFR-PRV-004–NFR-PRV-010 |
<a id="nfr-rec"></a>
### 4.8 Resilience, Backup, and Disaster Recovery

Defines recovery objectives, fault behavior, restoration testing, backlog recovery, and business continuity.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-REC-001` | The recovery point objective for acknowledged authoritative financial state and its audit evidence shall be zero data loss. | Fault-injection and disaster-recovery tests shall confirm every acknowledged outcome is present exactly once after recovery. | GFR-005–GFR-008, GFR-013, GFR-014 |
| `NFR-REC-002` | Draft or unsubmitted user work shall be recoverable to a point no older than 60 seconds where the workflow permits drafts. | Verify autosave or equivalent user-visible recovery without establishing unauthorized business effects. | UX §§4.3, 5; applicable FR families |
| `NFR-REC-003` | Class A functions shall have a recovery time objective of 2 hours after declaration of a service disaster. | Measure time to verified business operation, reconciliation, and user access, not only technical process start. | WF-6.1–WF-7.15 |
| `NFR-REC-004` | Class B functions shall have an RTO of 4 hours, and Class C and D functions shall have an RTO of 8 hours. | Test each service class independently and report unmet dependencies. | All capabilities |
| `NFR-REC-005` | Recovery from process, host, zone, or regional interruption shall not duplicate postings, approvals, payments, receipts, filings, corrections, or audit evidence. | Execute interruption at each documented state boundary and compare authoritative outcomes. | DDD §§9.3, 9.4; GFR-006 |
| `NFR-REC-006` | Backup restoration shall be tested at least quarterly, and a full disaster-recovery exercise shall be completed at least annually. | Tests shall include financial reconciliation, audit proof, access controls, legal holds, and source/dependent consistency. | All capabilities; NFR-REL-* |
| `NFR-REC-007` | Recovery procedures shall preserve accounting scope, lifecycle state, versions, idempotency identities, approvals, source watermarks, and correction lineage. | Verify restored records against pre-failure control totals and hashes. | GFR-006–GFR-014 |
| `NFR-REC-008` | After a dependency or service outage of up to 1 hour, the product shall process the accumulated Class A backlog within 2 hours while continuing to accept new work within approved capacity. | Measure backlog age, catch-up rate, duplicate prevention, and exception volume. | GFR-012, GFR-017; WF-7.13 |
| `NFR-REC-009` | A failed or uncertain external submission shall remain visibly pending or exceptional until reconciled; recovery shall not assume success or automatically create a second business obligation. | Test payment, tax, bank, filing, and provider-return uncertainty. | FR-PCM-*, FR-TAX-*, FR-BFR-*; DDD §9.4 |
| `NFR-REC-010` | Manual recovery, override, reconciliation, and data repair actions shall require authorization, reason, before/after evidence, and independent review for high-risk records. | Verify no direct destructive edit bypasses domain correction semantics. | GFR-003–GFR-005, GFR-014 |
| `NFR-REC-011` | Disaster recovery shall preserve declared data residency, encryption, access, privacy, retention, and legal-hold obligations. | Verify recovery locations and restored policy enforcement. | NFR-SEC-*, NFR-PRV-* |
| `NFR-REC-012` | Recovery completion shall require reconciliation of authoritative records, dependent outcomes, control totals, pending work, and audit evidence, with all residual differences assigned to owned exceptions. | Verify sign-off by business, operations, and security owners for Class A recovery. | NFR-REL-016; NFR-AUD-001 |
<a id="nfr-obs"></a>
### 4.9 Observability, Operations, and Supportability

Defines business-aware health, detection, alerting, diagnostics, incident management, and service reporting.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-OBS-001` | Authorized operations users shall have current health views for every capability and service class, including availability, latency, errors, pending work, aging, exceptions, and capacity. | Data freshness shall be no older than 2 minutes for Class A and 5 minutes for other classes. | All capabilities; GFR-009, GFR-012 |
| `NFR-OBS-002` | Critical financial-control failure, invariant violation, unexplained reconciliation difference, security incident, or audit-integrity failure shall be detected within 5 minutes and alert an accountable responder within 10 minutes. | Verify through quarterly alert exercises and production incident records. | NFR-REL-016, NFR-SEC-018, NFR-AUD-011 |
| `NFR-OBS-003` | Every user-visible error or exception shall provide a stable support reference that links authorized support staff to scope, workflow, stage, time, correlation, and sanitized diagnostic evidence. | Verify support can diagnose without requesting secrets or unrestricted screenshots. | GFR-009, GFR-014, GFR-015 |
| `NFR-OBS-004` | Logs, traces, metrics, and support diagnostics shall contain no unmasked sensitive values, secrets, full bank details, payroll details, tax identifiers, or verification credentials. | Verify automated sensitive-data scans run continuously and before release. | GFR-015; NFR-SEC-010 |
| `NFR-OBS-005` | Service-level objectives and error budgets shall be reported monthly by capability, workflow, service class, and accounting scope where material. | Reports shall include availability, p95/p99 latency, completion windows, data freshness, incident impact, and exclusions. | NFR-PERF-*, NFR-AVL-* |
| `NFR-OBS-006` | Pending approvals, postings, payments, receipts, returns, reconciliations, filings, close steps, and audit incidents shall expose age distributions and oldest-item details. | Verify alert thresholds are configurable by workflow and business calendar. | GFR-009, GFR-012, GFR-016 |
| `NFR-OBS-007` | Every alert shall have severity, owner, response target, runbook or resolution guide, suppression policy, and escalation path. | Verify no production critical alert remains unowned for more than 15 minutes. | All Class A workflows |
| `NFR-OBS-008` | Operational telemetry shall distinguish domain rejection, authorization denial, concurrency conflict, dependency unavailability, capacity control, and internal failure. | Verify service reporting does not combine these outcomes into one generic error rate. | GFR-007–GFR-009, GFR-017; UX §9.2 |
| `NFR-OBS-009` | Incident records shall include affected capabilities, workflows, scopes, users, business records, duration, financial impact, evidence impact, containment, recovery, and follow-up actions. | High-severity incidents require review within 5 business days. | GFR-014; NFR-AUD-011 |
| `NFR-OBS-010` | Monitoring and incident evidence shall be retained for at least 13 months, and material incident records for at least 7 years or the longer applicable policy. | Verify retention classification and legal-hold compatibility. | NFR-PRV-004, NFR-PRV-005 |
| `NFR-OBS-011` | The product shall provide user-visible service status for material degradation and planned maintenance, with affected functions, start time, current status, workaround, and next update. | Verify updates occur at least every 30 minutes during a material incident. | NFR-AVL-005, NFR-AVL-007 |
| `NFR-OBS-012` | Operational changes that alter alert thresholds, service objectives, capacity limits, or business-health definitions shall be authorized, versioned, and auditable. | Verify before/after values, owner, approval, effective time, and rollback path. | GFR-014; NFR-MNT-* |
<a id="nfr-acc"></a>
### 4.10 Accessibility and Inclusive Use

Defines measurable inclusive interaction requirements for all user-facing workflows and evidence.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-ACC-001` | All user-facing workflows shall conform to WCAG 2.2 Level AA. | Verify through automated checks and manual testing of representative pages and all 22 critical workflows before release. | UX §10.1; all FR families |
| `NFR-ACC-002` | Every action, form, dialog, worklist, table, timeline, and navigation element shall be operable with keyboard alone using a logical focus order. | Test without pointing-device input, including conflict, approval, correction, and recovery paths. | UX §§4–10 |
| `NFR-ACC-003` | Interactive controls, headings, regions, table headers, state changes, validation errors, and progress updates shall expose correct semantic information to assistive technology. | Test with at least two approved screen-reader and browser combinations. | UX §§4.3, 5, 10.1 |
| `NFR-ACC-004` | State, severity, approval, reconciliation, exception, gain/loss, and debit/credit meaning shall not rely on color alone. | Verify text labels, icons with accessible names, and patterns where visual distinction is required. | UX §10.1 |
| `NFR-ACC-005` | Text and interactive components shall meet applicable contrast requirements, and content shall remain usable at 200 percent text size and 400 percent browser zoom without loss of information or action. | Test representative dense financial tables, forms, dialogs, and reports. | UX §10.1 |
| `NFR-ACC-006` | Validation errors shall be announced, summarized, associated with the affected field or line, and provide a specific correction action without clearing valid user input. | Test single-field, multi-line, aggregate, authorization, and conflict errors. | GFR-009, GFR-017; UX §5.1 |
| `NFR-ACC-007` | Dynamic updates shall not unexpectedly move focus; material status changes shall be announced in a non-disruptive way. | Test long-running progress, approval outcome, external outcome, and exception creation. | UX §§5, 9 |
| `NFR-ACC-008` | Time limits shall be avoidable, extendable, or announced with sufficient time except where the business or security rule requires a fixed expiry; fixed expiry shall preserve safe recovery. | Test session, approval, authorization, reopen, and provider-window expiries. | NFR-SEC-007; DDD §§4, 8 |
| `NFR-ACC-009` | Users shall be able to reduce or disable nonessential animation and motion; no critical meaning shall depend on animation. | Test operating-system reduced-motion preference. | UX §10.1 |
| `NFR-ACC-010` | Evidence exports and printable reports shall preserve reading order, headings, table associations, labels, and textual status meaning. | Verify at least one human-readable export for each report and evidence class. | GFR-020; UX §10.3 |
| `NFR-ACC-011` | Accessibility defects that block a Class A workflow shall be treated as release-blocking critical defects; other WCAG AA failures shall have approved remediation before release. | Verify defect classification and release evidence. | NFR-TST-*; all workflows |
| `NFR-ACC-012` | Accessibility acceptance shall include users with disabilities or qualified accessibility specialists for high-risk workflows at least annually and after major interaction redesign. | Record findings, decisions, and remediation traceability. | UX WF-6.1–WF-7.15 |
<a id="nfr-cmp"></a>
### 4.11 Compatibility and Client Quality

Defines supported client environments, viewport behavior, document quality, and graceful capability detection.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-CMP-001` | The product shall support the current and previous two major stable releases of approved Chromium-based browsers and Firefox, plus the current and previous major Safari release where the client platform permits it. | Verify compatibility for each production release. | All user-facing capabilities |
| `NFR-CMP-002` | Core workflows shall remain usable at a minimum viewport of 1280 by 720 CSS pixels and at common 1920 by 1080 desktop resolution without horizontal scrolling of the entire page. | Test that dense tables scroll within a labeled region while retaining row identity and headers. | UX §§4–8 |
| `NFR-CMP-003` | Approval, worklist, record review, exception review, and evidence review shall remain usable at a 1024-pixel-wide tablet viewport. | Test that intentionally unavailable actions are identified before user input is lost. | UX §§4–9 |
| `NFR-CMP-004` | Client capability, browser, network, or storage limitations shall produce a supported-state message and safe recovery rather than partial submission or silent degradation. | Test disabled storage, interrupted network, unsupported browser, and expired session cases. | GFR-009, GFR-017 |
| `NFR-CMP-005` | Human-readable reports and evidence documents shall render consistently for screen, print, and approved document export, with no clipped amounts, hidden status, or lost lineage references. | Test representative long identifiers, multi-currency values, and multi-page tables. | GFR-020; FR-RPT-*, FR-AUD-* |
| `NFR-CMP-006` | User-visible behavior shall be independent of client locale for authoritative identifiers, monetary values, dates, and workflow outcomes. | Compare the same record under at least three supported locales. | GFR-001, GFR-010; NFR-LOC-* |
| `NFR-CMP-007` | Client-side interruption, refresh, back navigation, or duplicate action shall not repeat an authoritative effect and shall return the established outcome or safe resume path. | Test every Class A submit and approval action. | GFR-006–GFR-009 |
| `NFR-CMP-008` | Supported client and assistive-technology matrices, known limitations, and end-of-support dates shall be published and reviewed at least quarterly. | Verify notification at least 90 days before planned support removal. | NFR-MNT-*; NFR-ACC-* |
<a id="nfr-loc"></a>
### 4.12 Localization, Currency, Date, and Language Quality

Defines locale-safe presentation while preserving canonical finance semantics.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-LOC-001` | Dates, times, numbers, and monetary values shall use the user-selected locale for presentation while preserving canonical stored values and explicit accounting/business scope. | Test at least three locales with different date, decimal, grouping, and negative-number conventions. | GFR-010; DDD §10 |
| `NFR-LOC-002` | No user-visible date shall be ambiguous; numeric dates shall include a locale-defined format or an unambiguous written month, and evidence exports shall include the applicable time zone. | Test cutoff, due date, posting date, occurred time, and recorded time. | DDD §§10, 11.1 |
| `NFR-LOC-003` | Currency codes shall be displayed whenever more than one currency is possible, and transaction, functional, and presentation amounts shall be labeled separately. | Test cross-currency invoice, settlement, revaluation, translation, and reporting workflows. | GFR-010; WF-7.5, WF-7.6, WF-7.9 |
| `NFR-LOC-004` | Currency minor units, accounting precision, rate precision, rounding rule, and sign convention shall follow the effective policy retained by the business record. | Verify historical records after policy change. | GFR-010, GFR-019; DDD §10 |
| `NFR-LOC-005` | Business time-zone and daylight-saving rules shall be applied consistently to cutoff, period, execution, due, filing, and authorization-expiry decisions. | Test skipped, repeated, and offset-changing local times. | DDD §§4, 10 |
| `NFR-LOC-006` | The interface and generated documents shall support Unicode names, addresses, references, and descriptions without data loss or identifier ambiguity. | Test combining characters, non-Latin scripts, and right-to-left content values even when the interface language remains left-to-right. | FR-OMD-*, GFR-015 |
| `NFR-LOC-007` | Translatable interface text shall allow at least 30 percent expansion without clipping critical labels, amounts, or actions. | Test longest supported translations and responsive layouts. | NFR-CMP-002–NFR-CMP-005 |
| `NFR-LOC-008` | Unsupported locale, currency, calendar, or jurisdiction-specific behavior shall be blocked with an explicit scope message and approved-extension path rather than silently applying a default. | Verify no unsupported statutory interpretation is implied. | DDD §12; GFR-009, GFR-017 |
<a id="nfr-int"></a>
### 4.13 Interoperability and External Dependency Quality

Defines semantic compatibility, duplicate and ordering behavior, evidence preservation, imports, exports, and provider isolation.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-INT-001` | Every inbound and outbound business exchange shall preserve stable business identity, source identity, scope, version, amount, currency, occurred time, correlation, and authoritative owner where applicable. | Verify round-trip evidence for each cross-capability and external workflow. | GFR-001, GFR-012–GFR-014; DDD §1.2 |
| `NFR-INT-002` | Published exchange contracts shall be versioned; unsupported versions shall be rejected or quarantined with no partial financial effect and a visible resolution path. | Test current, prior-supported, unknown, and malformed versions. | WF-7.13; DDD §§3.8, 7.13 |
| `NFR-INT-003` | Duplicate and out-of-order inbound observations shall establish at most one valid local effect and shall preserve the original event or provider identity. | Execute duplicate, replay, reorder, and delayed-delivery tests across recovery. | GFR-006–GFR-008; WF-7.13 |
| `NFR-INT-004` | A sequence gap, missing reference, invalid scope, amount mismatch, or unsupported semantic transformation shall create a typed owned exception and no silent partial effect. | Verify exception evidence, owner, age, retry eligibility, and resolution. | GFR-009, GFR-012, GFR-017; DDD §7.13 |
| `NFR-INT-005` | External-provider or authority unavailability shall be isolated from unrelated product functions and shall preserve accepted local intent and reconciliation state. | Test bank, payment, tax, rate, identity, and notification dependency outages. | NFR-AVL-006; GFR-017 |
| `NFR-INT-006` | Import processing shall identify accepted, rejected, duplicate, deferred, and warning records individually and shall reconcile their counts and control totals to the source. | Test full acceptance, full rejection, mixed result, restart, and corrected re-import. | NFR-REL-015; FR-BFR-*, FR-GL-*, FR-OMD-* |
| `NFR-INT-007` | Export processing shall identify source scope, filter, record count, generation time, version, and sensitivity classification and shall be reproducible from unchanged authoritative inputs. | Verify authorized and restricted export cases. | GFR-020; NFR-AUD-008, NFR-AUD-009 |
| `NFR-INT-008` | External uncertainty shall remain a distinct pending or exception state until authoritative evidence establishes success, rejection, cancellation, return, or timeout disposition. | Test that automated retry cannot create a second business obligation. | GFR-012, GFR-017; FR-PCM-*, FR-TAX-*, FR-BFR-* |
| `NFR-INT-009` | Changes to an external or cross-capability contract shall preserve at least one previously supported version for an approved transition period or provide an approved coordinated cutover with reconciliation. | Verify backward-compatibility and rollback evidence before release. | NFR-MNT-004; DDD §5 |
| `NFR-INT-010` | Interoperability health shall report exchange volume, success, rejection, exception, age, duplicate, out-of-order, and reconciliation metrics by source and contract version. | Data freshness shall meet NFR-OBS-001. | NFR-OBS-*; WF-7.13 |
<a id="nfr-mnt"></a>
### 4.14 Maintainability, Change Safety, and Operability

Defines traceability, controlled change, compatibility, release safety, documentation, and sustainable operation.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-MNT-001` | Every delivered behavior shall trace to an approved DDD, FR, GFR, UX, or NFR requirement, and every requirement shall trace to verification evidence or an approved deferral. | Verify 100 percent traceability completeness at release. | DDD v3.1; FPRD v1.5; UX v1.0 |
| `NFR-MNT-002` | Changes to domain terminology, ownership, aggregate state, command, event, invariant, correction, or acceptance meaning shall update the DDD baseline before dependent functional, UX, NFR, and solution artifacts. | Verify source hashes and change-impact review. | GFR-001, GFR-021, GFR-022 |
| `NFR-MNT-003` | Business rules, accounting policies, approval policies, access policies, mappings, rates, and report definitions shall be versioned and historical records shall retain the applied version. | Test rule change, rollback where permitted, and historical reproduction. | GFR-004, GFR-019; DDD §§2, 10 |
| `NFR-MNT-004` | Published interfaces and exchange contracts shall remain backward compatible during the approved support window; breaking change requires explicit versioning, migration, reconciliation, and rollback plan. | Verify compatibility tests before production release. | NFR-INT-002, NFR-INT-009 |
| `NFR-MNT-005` | A production release shall be reversible or safely forward-correctable within 30 minutes when it causes a critical regression, without loss or duplication of acknowledged financial state. | Exercise rollback or forward-fix procedure before high-risk releases. | NFR-REC-001, NFR-REC-005 |
| `NFR-MNT-006` | Configuration and administrative changes shall require authorization, validation, audit evidence, effective-time control where applicable, and a documented recovery path. | Verify changes for OMD, GL, COA, tax, workflow, identity, reporting, rates, and bank connections. | FR-OMD-*, FR-GL-015–018, FR-COA-*, FR-TAX-016, FR-WFA-005, FR-IAM-* |
| `NFR-MNT-007` | Operational runbooks shall exist for every Class A workflow, critical alert, recovery procedure, security incident, and audit-integrity incident. | Runbooks shall identify owner, prerequisites, decision points, evidence, escalation, and completion checks and be reviewed at least quarterly. | NFR-OBS-007; NFR-REC-* |
| `NFR-MNT-008` | User, support, operations, security, and audit documentation shall be updated in the same release as changed behavior and shall identify the applicable version and effective date. | Verify documentation-to-product consistency by sampling before release. | All capabilities |
| `NFR-MNT-009` | Performance regression for an unchanged qualification profile shall not exceed 10 percent without approved capacity or product-owner acceptance, and no Class A target may be breached. | Compare release candidate to the current production baseline. | NFR-PERF-*, NFR-CAP-* |
| `NFR-MNT-010` | No production release shall contain an unresolved critical defect in financial integrity, security, privacy, availability, recovery, accessibility, or audit evidence. | Verify every high-severity exception has documented owner, impact, expiry, compensating control, and approval. | NFR-TST-*; all categories |
| `NFR-MNT-011` | Dependencies and supported client platforms shall have documented ownership, support status, end-of-life date, upgrade plan, and risk classification. | Review at least quarterly and alert 180 days before end of support. | NFR-CMP-008; NFR-SEC-015 |
| `NFR-MNT-012` | The NFR baseline, source hashes, requirement counts, open exceptions, and validation results shall be checkpointed; unchanged hashes permit reuse of prior review evidence. | Any changed requirement or source hash shall trigger the affected-category and traceability review. | Verification checkpoint in this document |
<a id="nfr-tst"></a>
### 4.15 Verification, Testing, and Release Quality

Defines mandatory verification coverage and pass conditions for functional, UX, NFR, security, recovery, and audit quality.
| ID | Requirement | Acceptance and measurement | Traceability |
|---|---|---|---|
| `NFR-TST-001` | All 199 Functional PRD acceptance scenarios shall have executable or formally witnessed verification evidence before production release. | Coverage shall preserve normal, boundary, concurrency, duplicate, and failure/recovery meanings. | Functional Traceability and Acceptance v1.5 |
| `NFR-TST-002` | All 99 UX acceptance criteria shall have verification evidence across the supported client and accessibility matrix. | Critical workflow criteria shall be exercised end to end. | UX v1.0 §11 |
| `NFR-TST-003` | Every NFR in this document shall have an identified verification method, environment, data profile, owner, result, and evidence reference or an approved time-bound exception. | Coverage completeness shall be 100 percent before release. | All NFR categories |
| `NFR-TST-004` | Class A workflows shall pass end-to-end fault, duplicate, concurrency, restart, and recovery tests at every material authoritative boundary. | Verify zero unexplained financial differences and zero duplicate effects. | WF-6.1–WF-7.15; DDD §§9, 14 |
| `NFR-TST-005` | Performance and capacity qualification shall use the declared baseline profile, production-like data distribution, authorization filtering, and representative cross-capability concurrency. | Results shall include p50, p95, p99, throughput, saturation, error classes, and recovery after overload. | NFR-PERF-*, NFR-CAP-* |
| `NFR-TST-006` | Security verification shall include authentication, authorization, segregation, step-up, session, export, secret, privacy, vulnerability, and incident-alert tests. | Verify that no critical security finding remains unresolved. | NFR-SEC-*, NFR-PRV-* |
| `NFR-TST-007` | Accessibility verification shall combine automated scanning, keyboard testing, screen-reader testing, zoom/reflow testing, and manual review of all critical workflows. | Verify release severity is assigned according to NFR-ACC-011. | NFR-ACC-* |
| `NFR-TST-008` | Backup, restore, zone failure, regional recovery, dependency outage, backlog catch-up, and disaster-recovery tests shall reconcile authoritative data and audit evidence. | Required exercises shall meet NFR-REC objectives. | NFR-REC-* |
| `NFR-TST-009` | Release evidence shall include requirement traceability, test results, unresolved exceptions, capacity headroom, service-objective readiness, runbooks, support readiness, security approval, and business sign-off. | Verify missing mandatory evidence blocks release. | NFR-MNT-001, NFR-MNT-007, NFR-MNT-010 |
| `NFR-TST-010` | Production verification after release shall confirm representative Class A and Class B workflows, cross-capability visibility, monitoring, alerts, audit evidence, and rollback readiness within the approved release window. | Verify any critical failure triggers rollback or forward-correction under NFR-MNT-005. | NFR-OBS-*, NFR-AUD-*, NFR-MNT-005 |
<a id="section-5"></a>
## 5. Capability Applicability Matrix

`Primary class` is the usual interactive or processing class. `Critical class` is the class applied when the capability controls or establishes authoritative financial state. Category codes refer to §4 and do not reduce globally applicable security, privacy, audit, accessibility, or change-control obligations.

| Capability | Family | Primary class | Critical class | Principal NFR categories |
|---|---|---|---|---|
| Organization & Master Data | `FR-OMD-*` | D | B | SEC, PRV, AUD, ACC, CMP, LOC, MNT, TST |
| General Ledger | `FR-GL-*` | A | A | PERF, CAP, AVL, REL, SEC, AUD, REC, OBS, ACC, INT, MNT, TST |
| Accounts Payable | `FR-AP-*` | A | A | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Accounts Receivable | `FR-AR-*` | A | A | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Payroll | `FR-PAYR-*` | A | A | PERF, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Invoicing | `FR-INV-*` | B | B | PERF, CAP, AVL, REL, SEC, PRV, AUD, OBS, ACC, LOC, INT, MNT, TST |
| Payments & Cash Management | `FR-PCM-*` | A | A | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Financial Reporting | `FR-RPT-*` | C | B | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, CMP, LOC, MNT, TST |
| Multi-Entity / Intercompany | `FR-IC-*` | A | A | PERF, AVL, REL, SEC, AUD, REC, OBS, ACC, LOC, INT, MNT, TST |
| Revenue Recognition | `FR-REV-*` | A | A | PERF, AVL, REL, SEC, AUD, REC, OBS, ACC, LOC, INT, MNT, TST |
| Fixed Assets | `FR-FA-*` | A | A | PERF, AVL, REL, SEC, AUD, REC, OBS, ACC, INT, MNT, TST |
| Multi-Currency | `FR-FX-*` | C | A | PERF, CAP, AVL, REL, SEC, AUD, REC, OBS, ACC, LOC, INT, MNT, TST |
| Fiscal Period Management | `FR-FPM-*` | A | A | PERF, AVL, REL, SEC, AUD, REC, OBS, ACC, INT, MNT, TST |
| COA Segment Accounting | `FR-COA-*` | D | B | PERF, AVL, REL, SEC, AUD, OBS, ACC, LOC, MNT, TST |
| Bank Feeds & Reconciliation | `FR-BFR-*` | C | A | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Tax Filing | `FR-TAX-*` | A | A | PERF, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Workflow & Approvals | `FR-WFA-*` | A | A | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Identity & Access | `FR-IAM-*` | D | A | PERF, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, INT, MNT, TST |
| Audit Integrity | `FR-AUD-*` | A | A | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, ACC, CMP, INT, MNT, TST |

<a id="section-6"></a>
## 6. Workflow Criticality and Recovery Matrix

| Workflow | Name | Service class | Completion or response target | RTO | RPO | Principal categories |
|---|---|---|---|---|---|---|
| `WF-6.1` | Period Close: Hard Close | A | 4 hours | 2 hours | 0 acknowledged-state loss | PERF, AVL, REL, SEC, AUD, REC, OBS, TST |
| `WF-6.2` | Fiscal Period Reopen and Reclose | A | Class A interactive plus close window | 2 hours | 0 acknowledged-state loss | PERF, AVL, REL, SEC, AUD, REC, OBS, TST |
| `WF-6.3` | Intercompany Reconciliation and Settlement | A | 4 hours for baseline settlement run | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, INT, TST |
| `WF-6.4` | Fixed Asset Disposal | A | Class A interactive; 60 minutes internal settlement handoff | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, INT, TST |
| `WF-6.5` | SaaS Revenue Recognition | A | 4 hours for baseline period run | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, LOC, TST |
| `WF-6.6` | Journal Posting and Reversal | A | NFR-PERF-001 and NFR-PERF-002 | 2 hours | 0 acknowledged-state loss | PERF, CAP, REL, SEC, AUD, REC, TST |
| `WF-6.7` | Receipt Recording and Application | A | NFR-PERF-001 and NFR-PERF-002 | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, TST |
| `WF-7.1` | Vendor Invoice Lifecycle | A | Class A interactive | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, PRV, AUD, REC, TST |
| `WF-7.2` | Payment Execution | A | NFR-PERF-011 | 2 hours | 0 acknowledged-state loss | PERF, CAP, AVL, REL, SEC, PRV, AUD, REC, OBS, INT, TST |
| `WF-7.3` | Customer Adjustments | A | Class A interactive | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, PRV, AUD, REC, TST |
| `WF-7.4` | Bank Reconciliation | C/A | NFR-PERF-012 and Class A confirmation | 4 hours processing; 2 hours authoritative control | 0 acknowledged-state loss | PERF, CAP, REL, SEC, AUD, REC, INT, TST |
| `WF-7.5` | Foreign-Currency Settlement | A | Class A interactive | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, LOC, INT, TST |
| `WF-7.6` | Period-End Revaluation | C/A | 4 hours | 4 hours processing; 2 hours authoritative control | 0 acknowledged-state loss | PERF, CAP, REL, SEC, AUD, REC, LOC, TST |
| `WF-7.7` | Fixed-Asset Lifecycle | A | 4 hours for baseline period run | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, INT, TST |
| `WF-7.8` | Revenue Modifications | A | Class A interactive plus 4-hour rerun | 2 hours | 0 acknowledged-state loss | PERF, REL, SEC, AUD, REC, LOC, TST |
| `WF-7.9` | Consolidation | C/A | 4 hours | 8 hours processing; 2 hours authoritative publication control | 0 acknowledged-state loss | PERF, CAP, REL, SEC, AUD, REC, OBS, LOC, TST |
| `WF-7.10` | Tax Filing | A | Class A interactive; external time excluded | 2 hours | 0 acknowledged-state loss | PERF, AVL, REL, SEC, PRV, AUD, REC, INT, TST |
| `WF-7.11` | Payroll Corrections | A | Class A interactive and approved payroll window | 2 hours | 0 acknowledged-state loss | PERF, AVL, REL, SEC, PRV, AUD, REC, INT, TST |
| `WF-7.12` | Period-Control Recovery | A | Class A status within 60 seconds | 2 hours | 0 acknowledged-state loss | AVL, REL, SEC, AUD, REC, OBS, TST |
| `WF-7.13` | Cross-Context Event Handling | A | NFR-PERF-014 | 2 hours | 0 acknowledged-state loss | PERF, REL, AUD, REC, OBS, INT, TST |
| `WF-7.14` | Concurrency Rules | A | Typed conflict within Class A target | 2 hours | 0 acknowledged-state loss | PERF, REL, AUD, REC, TST |
| `WF-7.15` | Audit Integrity | A | Verification within 5 seconds p95 for standard proof | 2 hours | 0 acknowledged-state loss | PERF, AVL, REL, SEC, PRV, AUD, REC, OBS, TST |

<a id="section-7"></a>
## 7. Verification and Acceptance Strategy

### 7.1 Required evidence by release stage

| Stage | Minimum evidence |
|---|---|
| Product baseline | Approved targets, source traceability, capability/workflow applicability, assumptions, and exceptions. |
| Solution design | Design decisions mapped to every applicable NFR, capacity model, failure model, security/privacy controls, and verification approach. |
| Build completion | Automated functional, integrity, security, accessibility, compatibility, and component-level NFR results. |
| Preproduction qualification | Production-like performance, capacity, resilience, recovery, interoperability, audit, and operational readiness evidence. |
| Production release | Completed traceability, no blocking defects, approved exceptions, runbooks, monitoring, support readiness, rollback/forward-fix readiness, and business/security/operations sign-off. |
| Post-release | Production verification, SLO monitoring, reconciliation, incident review, capacity review, and corrective-action tracking. |

### 7.2 Release pass conditions

A release passes this NFR baseline only when:
1. Every applicable NFR has passing evidence or an approved unexpired exception.
2. All Class A integrity, duplicate, concurrency, security, privacy, audit, recovery, and accessibility gates pass without a critical defect.
3. Performance and capacity meet the qualification profile with at least 30 percent headroom at expected peak, unless a higher approved profile applies.
4. Recovery reconciles authoritative records, dependent outcomes, and audit evidence with zero unexplained differences.
5. All required runbooks, alerts, ownership, support, and status communication are operational.
6. Source DDD, Functional PRD, and UX checkpoint hashes match this document's declared source versions.

### 7.3 Production review cadence

| Review | Cadence |
|---|---|
| Availability, latency, errors, backlog, and SLOs | Monthly, with continuous monitoring |
| Capacity and growth forecast | Quarterly and before 70% forecast utilization |
| Access, segregation, emergency access, and privileged activity | Quarterly; emergency access within 1 business day |
| Vulnerability and dependency lifecycle | Continuous, summarized monthly and reviewed quarterly |
| Backup restoration | Quarterly |
| Disaster recovery | At least annually |
| Accessibility critical workflows | Every material interaction change and at least annually |
| NFR baseline and exceptions | Every material source or target change and at least annually |
<a id="section-8"></a>
## 8. Assumptions, Exceptions, and Change Control

### 8.1 Assumptions

- The DDD v3.1, Functional PRD v1.5, and UX v1.0 checkpoints are unchanged.
- The service is intended for multi-scope finance operation and may be used across time zones and jurisdictions.
- External provider completion times are not controlled by the product, but acceptance, pending-state visibility, timeout, exception, reconciliation, and recovery remain controlled.
- Jurisdiction, contract, or approved business calendar may impose stricter availability, retention, residency, security, or recovery expectations.
- Numerical qualification values are minimum acceptance targets and must be increased when forecast, contract, or observed demand requires it.

### 8.2 Exception record

Every NFR exception shall contain:
- Requirement ID and affected capability, workflow, environment, tenant, accounting scope, or jurisdiction.
- Measured result and target.
- Business, financial, security, privacy, audit, accessibility, and operational impact.
- Root cause and compensating control.
- Accountable owner and independent approver.
- Start date, expiry date, retest date, and closure evidence.

### 8.3 Change triggers

A targeted NFR review is required when any source hash changes or when a change affects service class, workload, latency, capacity, availability, financial integrity, security, privacy, retention, audit, recovery, accessibility, compatibility, localization, interoperability, support, or verification meaning. A full review is required when a new capability or workflow is added, a Class A boundary changes, or a source baseline changes domain ownership or authoritative financial behavior.

<a id="section-9"></a>
## 9. Verification Checkpoint

| Checkpoint field | Value |
|---|---|
| Checkpoint ID | NFR-1.0-2026-07-24 |
| Verified body SHA-256 | `e135e0c6198cfc5899b08777efc4bf9338ab46d5244a70c6f3dbd555ca5de5a9` |
| Hash boundary | UTF-8 bytes from the title through the blank line immediately preceding §9; §9 excluded |
| Source DDD checkpoint | DDD-3.1-2026-07-24 — `a9d437d23656c36d340afb3a5a31c93a23e574f53db186483a9edfdf32d3e652` |
| Source Functional PRD checkpoint | FPRD-1.5-2026-07-24 — `f5b7be0973f532851abf06cf8c408caf110dad538735aa21490c9ff8b84a2b8f` |
| Source Requirements Catalog checkpoint | FREQ-1.5-2026-07-24 — `76bb4ec1ecb155a08df478ea9d4101d40643c6eb372c34ae4cd85f9d82a2c69a` |
| Source Traceability checkpoint | FTRACE-1.5-2026-07-24 — `6489ac5a5d00562b15e600294d72175addde25d55aaaee3568d02537a8354f4a` |
| Source UX checkpoint | UXWF-1.0-2026-07-24 — `7bf6e021b60c4a441d32c7fb1e0b14f501000168b544fb108afaff87a92a8891` |
| NFR categories | 15 |
| NFR requirements | 174 |
| Capabilities covered | 19 |
| Workflows covered | 22 |
| Review result | Passed |
| Open consistency defects | None known within the declared NFR scope |
| Review reuse rule | When this body hash and all source hashes remain unchanged, rerun structural and hash checks only. Re-run affected categories and matrices after a source, target, capability, workflow, or requirement change. |

### 9.1 Requirement counts

| Category | Count |
|---|---:|
| PERF | 14 |
| CAP | 10 |
| AVL | 10 |
| REL | 16 |
| SEC | 18 |
| PRV | 10 |
| AUD | 12 |
| REC | 12 |
| OBS | 12 |
| ACC | 12 |
| CMP | 8 |
| LOC | 8 |
| INT | 10 |
| MNT | 12 |
| TST | 10 |
| **Total** | **174** |

### 9.2 Validation gates

| Gate | Result |
|---|---|
| Source checkpoint hashes match | Passed |
| Requirement IDs are unique and category-consistent | Passed |
| Every requirement uses mandatory measurable language | Passed |
| All 19 capabilities are represented | Passed |
| All 22 workflows are represented | Passed |
| All 22 GFR identifiers are referenced by the NFR baseline | Passed |
| DDD-only and functional/UX source authority remain unchanged | Passed |
| Service classes, availability, RTO, and RPO are internally consistent | Passed |
| Qualification profile is referenced by measurable targets | Passed |
| Security, privacy, audit, recovery, accessibility, and release gates are represented | Passed |
| Internal links and anchors resolve | Passed |
| Markdown tables are structurally valid | Passed |
| No unresolved placeholders or unbalanced code fences remain | Passed |

### 9.3 Independent semantic review

The final pass verified that:
- NFRs define quality or operating outcomes and do not introduce new domain commands, events, aggregates, lifecycle states, or accounting ownership.
- External elapsed time is separated from product-controlled acceptance, status, exception, reconciliation, and recovery time.
- Acknowledged authoritative financial state consistently uses zero-data-loss recovery expectations.
- Class A availability and RTO values are consistent across service, capability, workflow, recovery, and release sections.
- Security, privacy, retention, legal hold, audit, and accessibility requirements do not weaken the source baselines.
- Capacity and performance values are explicitly qualification floors and not undocumented forecasts or product hard limits.
- Every exception is time-bound, owned, approved, and retested.
