# Finance Platform Architecture Traceability, Decisions and Verification

| Field | Value |
|---|---|
| Version | 1.0 |
| Status | Passed — complete source and design traceability |
| Pack | Finance Solution/System Design v1.0 |

## 1. Source Baseline Manifest

| Source | Full-file SHA-256 |
|---|---|
| ddd | `0edb1702b5278e75d80efb98d2e909110e73dd34bd362e6149a11bb1e0554402` |
| prd | `50f7451b6f768feb2fbf13aa928acb3aefb590c8c36a990ecfef6988c0705335` |
| catalog | `bbd490b458597779f792157c5f146ff488a12f26c9b26cd0f1d31b391ed52f5f` |
| acceptance | `1f511e2d699dc5ae2c83abe720d20d29c5eaa1683bb69d3e7f545099dad63f4e` |
| ux | `84761c9805b7e639ae9dab8fb3800f6e02d80c843ec86aac9e9057241c3501d5` |
| nfr | `52099bfa577d7682493e95702d82c8e1f8b23c1e038a7ad4a935815b10df3dbd` |

## 2. Architecture Decision Records

| ADR | Decision | Status | Rationale |
|---|---|---|---|
| ADR-001 | Use a modular monolith | Accepted | Reduces operational cost and distributed-consistency complexity while preserving DDD boundaries and later extraction paths. |
| ADR-002 | Use Go for the backend | Accepted | Small runtime, strong concurrency support, static binaries and direct standard-library integration. |
| ADR-003 | Use React and TypeScript for the SPA | Accepted | Matches the approved UX model and broad ecosystem support. |
| ADR-004 | Use Tailwind CSS and daisyUI | Accepted | Low-cost component styling; application wrappers provide semantic behavior and accessibility. |
| ADR-005 | Use PostgreSQL as the authoritative store | Accepted | Strong transactions, constraints, exact numeric types, mature backup/recovery and local/cloud parity. |
| ADR-006 | Use one database with schema-per-context | Accepted | Minimizes cost and operational burden while making ownership visible and enforceable. |
| ADR-007 | Use pgx and sqlc instead of a general ORM | Accepted | Keeps SQL and financial constraints explicit and reviewable. |
| ADR-008 | Use REST/JSON with OpenAPI 3.1 | Accepted | Simple learning path, contract generation and broad interoperability. |
| ADR-009 | Use PostgreSQL outbox/inbox without a broker initially | Accepted | Provides durable asynchronous semantics without adding Kafka or Service Bus cost and operations. |
| ADR-010 | Use Microsoft Entra ID for authentication | Accepted | Azure-aligned OIDC and MFA/conditional-access integration; application retains finance authorization. |
| ADR-011 | Use Azure Static Web Apps for the SPA | Accepted | Managed static hosting and repository integration for React builds. |
| ADR-012 | Use Azure Container Apps for Go workloads | Accepted | Container deployment, managed ingress and scale-to-zero for the learning profile. |
| ADR-013 | Use Azure Database for PostgreSQL Flexible Server | Accepted | Managed PostgreSQL with stop/start and burstable options for learning, plus HA options for production qualification. |
| ADR-014 | Use Terraform for infrastructure as code | Accepted | User-selected portable IaC with AzureRM/AzureAD providers and remote state. |
| ADR-015 | Use GitHub Actions with OIDC federation | Accepted | Avoids stored Azure client secrets and keeps delivery close to the repository. |
| ADR-016 | Use OpenTelemetry and slog | Accepted | Vendor-neutral instrumentation and structured Go logging. |
| ADR-017 | Separate learning and production profiles | Accepted | Prevents low-cost settings from being misrepresented as satisfying production NFRs. |
| ADR-018 | Defer a message broker | Accepted | Add Azure Service Bus only when independent deployment, external fan-out or measured throughput requires it. |
| ADR-019 | Defer Redis and distributed caching | Accepted | Correctness and simplicity outweigh cache complexity until measured read load justifies it. |
| ADR-020 | Use reporting-owned projections | Accepted | Protects authoritative write workloads and preserves source watermarks and rebuildability. |

A change to an accepted ADR requires a superseding ADR, impact assessment and targeted/full review according to Section 10.

## 3. Global Functional Requirement Traceability

| Requirement | Architecture controls |
|---|---|
| GFR-001 | ARC-DOM-001, ARC-MOD-001 |
| GFR-002 | ARC-SEC-002 |
| GFR-003 | ARC-SEC-002 |
| GFR-004 | ARC-TXN-002, ARC-SEC-002 |
| GFR-005 | ARC-DOM-003, ARC-DATA-003 |
| GFR-006 | ARC-IDEM-001 |
| GFR-007 | ARC-IDEM-001 |
| GFR-008 | ARC-CON-001 |
| GFR-009 | ARC-API-001, ARC-OBS-002 |
| GFR-010 | ARC-DATA-002, ARC-DATA-004 |
| GFR-011 | ARC-DOM-002 |
| GFR-012 | ARC-TXN-002, ARC-EVT-001, ARC-EVT-002 |
| GFR-013 | ARC-DOM-003, ARC-DATA-003 |
| GFR-014 | ARC-AUD-001, ARC-OBS-001 |
| GFR-015 | ARC-PRV-001, ARC-SEC-002 |
| GFR-016 | ARC-DATA-004, ARC-PERF-001 |
| GFR-017 | ARC-API-001, ARC-OBS-002 |
| GFR-018 | ARC-DATA-003, ARC-REC-001 |
| GFR-019 | ARC-DATA-003 |
| GFR-020 | ARC-AUD-001, ARC-PRV-001 |
| GFR-021 | ARC-DOM-001 |
| GFR-022 | ARC-TST-001 |

## 4. Capability Requirement Traceability

Every Functional PRD capability requirement is assigned to one owner module and schema. Supporting modules may participate through published ports or events but do not acquire ownership.

| Requirement | Owner module | Owned schema | Primary architecture controls |
|---|---|---|---|
| FR-AP-001 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-002 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-003 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-004 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-005 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-006 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-007 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-008 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-009 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AP-010 | `internal/ap` | `ap` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-001 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-002 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-003 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-004 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-005 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-006 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-007 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-008 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-009 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-010 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-011 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-012 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-013 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-014 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-015 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AR-016 | `internal/ar` | `ar` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-AUD-001 | `internal/audit` | `audit` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-003 |
| FR-AUD-002 | `internal/audit` | `audit` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-003 |
| FR-AUD-003 | `internal/audit` | `audit` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-003 |
| FR-AUD-004 | `internal/audit` | `audit` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-003 |
| FR-AUD-005 | `internal/audit` | `audit` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-003 |
| FR-BFR-001 | `internal/bankfeeds` | `bank_reconciliation` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-BFR-002 | `internal/bankfeeds` | `bank_reconciliation` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-BFR-003 | `internal/bankfeeds` | `bank_reconciliation` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-BFR-004 | `internal/bankfeeds` | `bank_reconciliation` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-BFR-005 | `internal/bankfeeds` | `bank_reconciliation` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-BFR-006 | `internal/bankfeeds` | `bank_reconciliation` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-COA-001 | `internal/coa` | `coa` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-COA-002 | `internal/coa` | `coa` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-COA-003 | `internal/coa` | `coa` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-COA-004 | `internal/coa` | `coa` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-COA-005 | `internal/coa` | `coa` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FA-001 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-002 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-003 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-004 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-005 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-006 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-007 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-008 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-009 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-010 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-011 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-012 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-013 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-014 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-015 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-016 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-017 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-018 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-019 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-020 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FA-021 | `internal/fixedassets` | `fixed_assets` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FPM-001 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-002 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-003 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-004 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-005 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-006 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-007 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-008 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-009 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-010 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-011 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-012 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FPM-013 | `internal/fiscalperiod` | `fiscal_period` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-FX-001 | `internal/multicurrency` | `multi_currency` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FX-002 | `internal/multicurrency` | `multi_currency` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FX-003 | `internal/multicurrency` | `multi_currency` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FX-004 | `internal/multicurrency` | `multi_currency` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-FX-005 | `internal/multicurrency` | `multi_currency` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-GL-001 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-002 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-003 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-004 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-005 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-006 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-007 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-008 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-009 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-010 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-011 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-012 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-013 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-014 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-015 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-016 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-017 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-GL-018 | `internal/gl` | `gl` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-TXN-001 |
| FR-IAM-001 | `internal/identity` | `identity` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-SEC-001 |
| FR-IAM-002 | `internal/identity` | `identity` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-SEC-001 |
| FR-IAM-003 | `internal/identity` | `identity` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-SEC-001 |
| FR-IAM-004 | `internal/identity` | `identity` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-SEC-001 |
| FR-IAM-005 | `internal/identity` | `identity` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-SEC-001 |
| FR-IAM-006 | `internal/identity` | `identity` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-SEC-001 |
| FR-IC-001 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-002 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-003 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-004 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-005 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-006 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-007 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-008 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-009 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-010 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-IC-011 | `internal/intercompany` | `intercompany` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-INV-001 | `internal/invoicing` | `invoicing` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-INV-002 | `internal/invoicing` | `invoicing` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-INV-003 | `internal/invoicing` | `invoicing` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-INV-004 | `internal/invoicing` | `invoicing` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-INV-005 | `internal/invoicing` | `invoicing` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-INV-006 | `internal/invoicing` | `invoicing` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-OMD-001 | `internal/organization` | `organization` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-OMD-002 | `internal/organization` | `organization` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-OMD-003 | `internal/organization` | `organization` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-OMD-004 | `internal/organization` | `organization` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-OMD-005 | `internal/organization` | `organization` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-OMD-006 | `internal/organization` | `organization` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-PAYR-001 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PAYR-002 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PAYR-003 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PAYR-004 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PAYR-005 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PAYR-006 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PAYR-007 | `internal/payroll` | `payroll` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-001 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-002 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-003 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-004 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-005 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-006 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-007 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-008 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-009 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-010 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-011 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-012 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-013 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-014 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-015 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-016 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-017 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-018 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-019 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-020 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-021 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-022 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-023 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-024 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-PCM-025 | `internal/payments` | `payments` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-REV-001 | `internal/revenue` | `revenue` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-REV-002 | `internal/revenue` | `revenue` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-REV-003 | `internal/revenue` | `revenue` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-REV-004 | `internal/revenue` | `revenue` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-REV-005 | `internal/revenue` | `revenue` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-REV-006 | `internal/revenue` | `revenue` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-RPT-001 | `internal/reporting` | `reporting` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-004, ARC-PERF-001 |
| FR-RPT-002 | `internal/reporting` | `reporting` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-004, ARC-PERF-001 |
| FR-RPT-003 | `internal/reporting` | `reporting` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-004, ARC-PERF-001 |
| FR-RPT-004 | `internal/reporting` | `reporting` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-004, ARC-PERF-001 |
| FR-RPT-005 | `internal/reporting` | `reporting` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-004, ARC-PERF-001 |
| FR-RPT-006 | `internal/reporting` | `reporting` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DATA-004, ARC-PERF-001 |
| FR-TAX-001 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-002 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-003 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-004 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-005 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-006 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-007 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-008 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-009 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-010 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-011 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-012 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-013 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-014 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-015 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-TAX-016 | `internal/tax` | `tax` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001, ARC-DOM-002, ARC-DOM-003, ARC-EVT-001, ARC-EVT-002 |
| FR-WFA-001 | `internal/workflow` | `workflow` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-WFA-002 | `internal/workflow` | `workflow` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-WFA-003 | `internal/workflow` | `workflow` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-WFA-004 | `internal/workflow` | `workflow` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |
| FR-WFA-005 | `internal/workflow` | `workflow` | ARC-DOM-001, ARC-MOD-001, ARC-API-001, ARC-DATA-001, ARC-SEC-002, ARC-AUD-001 |

## 5. Workflow Traceability

| Workflow | Title | Modules/components | Representative sequence |
|---|---|---|---|
| WF-6.1 | Period Close: Hard Close | `fiscalperiod`, `gl`, `workflow`, `multicurrency`, `fixedassets`, `revenue`, `intercompany`, `reporting`, `audit` | SEQ-FPM-001 |
| WF-6.2 | Fiscal Period Reopen and Reclose | `fiscalperiod`, `gl`, `workflow`, `reporting`, `audit` | SEQ-FPM-001 |
| WF-6.3 | Intercompany Reconciliation and Settlement | `intercompany`, `payments`, `gl`, `workflow`, `reporting` | — |
| WF-6.4 | Fixed Asset Disposal with Gain or Loss Recognition | `fixedassets`, `ap`, `payments`, `gl`, `workflow` | SEQ-FA-001 |
| WF-6.5 | Revenue Recognition for a SaaS Contract | `revenue`, `invoicing`, `ar`, `gl`, `workflow` | — |
| WF-6.6 | Journal Entry Posting and Reversal | `gl`, `workflow`, `audit` | SEQ-GL-001 |
| WF-6.7 | Customer Receipt Recording with Partial Application | `ar`, `gl`, `bankfeeds` | SEQ-AR-001 |
| WF-7.1 | Vendor Invoice Registration, Matching, Approval, Dispute, and Void | `ap`, `organization`, `workflow`, `gl` | — |
| WF-7.2 | Payment Batch Approval, Submission, Retry, Partial Settlement, and Cancellation | `payments`, `ap`, `workflow`, `gl` | SEQ-PCM-001 |
| WF-7.3 | Customer Credit, Refund, Overpayment, Chargeback, and Write-Off | `ar`, `payments`, `workflow`, `gl` | — |
| WF-7.4 | Bank Statement Import, Matching, Unmatching, and Reconciliation | `bankfeeds`, `payments`, `ar`, `ap` | — |
| WF-7.5 | Foreign-Currency Invoice Settlement and Realized FX | `ar`, `ap`, `payments`, `multicurrency`, `gl` | — |
| WF-7.6 | Period-End Revaluation, Rerun, and Next-Period Reversal | `multicurrency`, `gl`, `fiscalperiod`, `workflow` | — |
| WF-7.7 | Full Fixed-Asset Lifecycle and Disposal Variants | `fixedassets`, `ap`, `payments`, `gl`, `workflow` | SEQ-FA-001 |
| WF-7.8 | Revenue Modification, Renewal, Cancellation, Refund, and Variable Consideration | `revenue`, `ar`, `invoicing`, `gl`, `workflow` | — |
| WF-7.9 | Consolidation, Ownership Changes, Translation, Eliminations, and Rerun | `reporting`, `intercompany`, `multicurrency`, `workflow` | — |
| WF-7.10 | Tax Return Submission, Rejection, Amendment, Payment, and Evidence | `tax`, `payments`, `workflow`, `gl`, `audit` | — |
| WF-7.11 | Payroll Correction, Off-Cycle Run, Failed Payment, and Tax Amendment | `payroll`, `payments`, `tax`, `workflow`, `gl` | — |
| WF-7.12 | Period-Control Outage, Takeover, Cutoff, Exception Expiry, and Full Operational Reopen | `fiscalperiod`, `gl`, `workflow`, `audit` | SEQ-FPM-001 |
| WF-7.13 | Cross-Context Event Interpretation, Ordering, and Replay | `integration`, `audit`, `organization`, `gl`, `ap`, `ar`, `payroll`, `invoicing`, `payments`, `reporting`, `intercompany`, `revenue`, `fixedassets`, `multicurrency`, `fiscalperiod`, `coa`, `bankfeeds`, `tax`, `workflow`, `identity`, `audit` | — |
| WF-7.14 | Concurrent Aggregate and Domain-Process Modification Rules | `platform`, `organization`, `gl`, `ap`, `ar`, `payroll`, `invoicing`, `payments`, `reporting`, `intercompany`, `revenue`, `fixedassets`, `multicurrency`, `fiscalperiod`, `coa`, `bankfeeds`, `tax`, `workflow`, `identity`, `audit` | — |
| WF-7.15 | Audit Integrity Verification, Missing Evidence, Proof Mismatch, Verification-Credential Rotation, and Incident Escalation | `audit`, `identity`, `workflow` | SEQ-AUD-001 |

## 6. Nonfunctional Requirement Traceability

The learning profile is not the production qualification environment. An NFR may be architecturally addressed while its numerical target remains subject to production-profile verification.

| NFR | Architecture controls | Verification |
|---|---|---|
| NFR-ACC-001 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-002 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-003 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-004 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-005 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-006 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-007 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-008 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-009 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-010 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-011 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-ACC-012 | ARC-ACC-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-001 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-002 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-003 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-004 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-005 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-006 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-007 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-008 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-009 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-010 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-011 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AUD-012 | ARC-AUD-001, ARC-DATA-003, ARC-OBS-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-001 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-002 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-003 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-004 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-005 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-006 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-007 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-008 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-009 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-AVL-010 | ARC-REL-001, ARC-REL-002, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-001 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-002 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-003 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-004 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-005 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-006 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-007 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-008 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-009 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CAP-010 | ARC-CAP-001, ARC-PERF-001, ARC-REL-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-001 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-002 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-003 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-004 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-005 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-006 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-007 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-CMP-008 | ARC-API-001, ARC-ACC-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-001 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-002 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-003 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-004 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-005 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-006 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-007 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-008 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-009 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-INT-010 | ARC-API-001, ARC-EVT-001, ARC-EVT-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-001 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-002 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-003 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-004 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-005 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-006 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-007 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-LOC-008 | ARC-DATA-002, ARC-ACC-001, ARC-API-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-001 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-002 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-003 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-004 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-005 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-006 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-007 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-008 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-009 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-010 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-011 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-MNT-012 | ARC-MOD-001, ARC-IAC-001, ARC-CICD-001, ARC-TST-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-001 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-002 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-003 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-004 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-005 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-006 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-007 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-008 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-009 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-010 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-011 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-OBS-012 | ARC-OBS-001, ARC-OBS-002 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-001 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-002 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-003 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-004 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-005 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-006 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-007 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-008 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-009 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-010 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-011 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-012 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-013 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PERF-014 | ARC-PERF-001, ARC-OBS-001, ARC-CAP-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-001 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-002 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-003 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-004 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-005 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-006 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-007 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-008 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-009 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-PRV-010 | ARC-PRV-001, ARC-SEC-002, ARC-SEC-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-001 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-002 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-003 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-004 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-005 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-006 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-007 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-008 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-009 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-010 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-011 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REC-012 | ARC-REC-001, ARC-REL-002, ARC-EVT-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-001 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-002 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-003 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-004 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-005 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-006 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-007 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-008 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-009 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-010 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-011 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-012 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-013 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-014 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-015 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-REL-016 | ARC-TXN-001, ARC-TXN-002, ARC-IDEM-001, ARC-CON-001, ARC-EVT-001, ARC-EVT-002, ARC-DATA-003 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-001 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-002 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-003 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-004 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-005 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-006 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-007 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-008 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-009 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-010 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-011 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-012 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-013 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-014 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-015 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-016 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-017 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-SEC-018 | ARC-SEC-001, ARC-SEC-002, ARC-SEC-003, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-001 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-002 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-003 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-004 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-005 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-006 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-007 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-008 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-009 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |
| NFR-TST-010 | ARC-TST-001, ARC-CICD-001 | Architecture/control test plus requirement-specific measurement from the NFR specification. |

## 7. Functional Acceptance Coverage

All 199 `FAC-*` acceptance scenarios remain authoritative in the Functional PRD acceptance document. Architecture verification groups them as follows:

| Acceptance source | Architecture verification |
|---|---|
| GL posting, currency and period gate | API contract, database invariant, concurrency and journal reconciliation tests |
| Receipt application and customer adjustments | Deterministic-lock, exact-money, duplicate and recovery tests |
| Payment execution, returns and incoming settlement | Provider adapter, outbox/inbox, reconciliation and exception tests |
| Close, reopen and period-control recovery | Gate-ownership, crash-recovery, watermark and audit-evidence tests |
| Fixed assets and revenue | Multi-leg posting, compensation, schedule version and settlement tests |
| Intercompany, FX and consolidation | Participant-scope, rate evidence, elimination and rerun tests |
| Tax and payroll | Restricted-data, amendment/correction, submission and payment tests |
| Cross-context delivery and concurrency | Duplicate, out-of-order, poison, replay and expected-version tests |
| Audit integrity | Sequence, missing evidence, mismatch, credential rotation and incident tests |

## 8. Architecture Risks

| Risk | Description | Impact | Mitigation |
|---|---|---|---|
| RISK-001 | Learning Azure profile does not satisfy production availability/capacity NFRs | High | Maintain explicit profiles; run NFR qualification only on the production reference topology. |
| RISK-002 | One PostgreSQL server is a shared failure and resource domain | High | Schema ownership, workload pools, backups, HA for production and extraction plan for proven hotspots. |
| RISK-003 | Long-running close/report/import jobs can starve interactive work | High | Separate worker process/pools, admission control, progress checkpoints and workload-specific limits. |
| RISK-004 | daisyUI visual classes do not guarantee accessible behavior | High | Typed semantic wrappers, automated accessibility checks and manual keyboard/screen-reader review. |
| RISK-005 | Finance authorization and segregation rules become inconsistent | High | Central policy service inside IAM module, decision evidence, rule tests and deny-by-default handling. |
| RISK-006 | Outbox backlog or poison work delays cross-context outcomes | Medium | Lease/fencing, attempts, typed failure, backlog alerts, repair UI and controlled replay. |
| RISK-007 | Reporting queries overload the authoritative database | High | Reporting projections, read-optimized indexes, separate connection pool and future replica/warehouse decision. |
| RISK-008 | Developers bypass module boundaries through shared database access | High | Database roles, repository packages, import tests, code review and no cross-schema joins in write paths. |
| RISK-009 | Azure PostgreSQL is the main recurring learning cost | Medium | Local-first development, stopped/disposable demo environments and Terraform destroy workflows. |
| RISK-010 | Audit-integrity cryptographic mechanism is not yet selected | Medium | Keep audit envelope and immutable sequence now; select sealing/verification algorithm in technical specification ADR. |
| RISK-011 | Exact decimal library behavior differs from PostgreSQL NUMERIC | High | Arithmetic conformance suite, explicit rounding policies and golden accounting examples before implementation approval. |
| RISK-012 | Scale-to-zero cold starts can violate interactive performance targets | Medium | Allow learning-profile exception; production profile uses minimum replicas and measured warm capacity. |

## 9. Detailed Technical Specification Backlog

| Specification | Required contents |
|---|---|
| TS-API | Exact OpenAPI paths, schemas, examples, errors, auth scopes, idempotency and pagination. |
| TS-DB | Tables, columns, types, constraints, indexes, partitions, isolation and migration DDL. |
| TS-EVENT | Event schemas, versions, outbox/inbox tables, dispatch rules, attempts, replay and repair. |
| TS-AUTH | Entra registrations, token claims, permission catalog, scope model and segregation algorithms. |
| TS-FRONTEND | Route tree, component APIs, query keys, form schemas, accessibility and testing fixtures. |
| TS-TERRAFORM | Provider/module versions, resources, variables, state bootstrap, policies and environment parameters. |
| TS-OBS | Log fields, metric names, trace attributes, dashboards, alerts and retention. |
| TS-RECOVERY | Backup settings, restore commands, RTO/RPO procedures and reconciliation queries. |
| TS-PERF | Workload models, pool sizes, batch limits, query budgets and performance test plans. |
| TS-AUDIT | Canonicalization, hash/signature algorithms, key rotation, proof format and verification. |

## 10. Verification Strategy

### 10.1 Automated gates

- Markdown structure, links, tables and code fences.
- Source SHA-256 manifest.
- Unique architecture controls, ADRs, risks and traceability IDs.
- Coverage of all 22 GFR IDs, all 193 FR IDs, all 22 workflows and all 174 NFR IDs.
- Module/schema ownership uses only the 19 approved bounded contexts.
- Selected technology baseline appears consistently across all documents.
- No unresolved draft or incomplete-content markers.
- Checkpoint hashes match each document body.
- ZIP contains Markdown files only.

### 10.2 Manual semantic gates

- No design control changes DDD ownership, commands, states or invariants.
- Low-cost learning settings are never claimed to meet production NFRs.
- Cross-context processes do not use direct schema writes or distributed transactions.
- Authentication and authorization responsibilities remain separate.
- UI component choice does not reduce accessibility obligations.
- Recovery prioritizes established financial correctness over rapid uncertain availability.

### 10.3 Review trigger policy

- **No re-review:** all source hashes and generated body hashes match.
- **Targeted review:** wording, diagram or non-authoritative reference changes with unchanged ownership, data, security and recovery decisions.
- **Full review:** source baseline, architecture style, technology stack, data ownership, transaction/event semantics, security boundary, production topology, RTO/RPO or an accepted ADR changes.


## Verification Checkpoint

| Field | Value |
|---|---|
| Verified body SHA-256 | `7cf912b8ffc5f625f126cf99cb699f1977e5dea8543f415a3d465bb90b80cef6` |
| Review status | Passed |
| Reuse rule | Re-run targeted checks when this hash or a source hash changes; re-run the full suite for architecture, data ownership, security, recovery, or technology-baseline changes. |
