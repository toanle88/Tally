# OpenAPI contract source

`openapi.yaml` is the OpenAPI 3.1 root document. Shared transport components are
in `components/common.yaml`; the approved contract-only operation catalog is in
`paths/catalog.yaml`; deterministic, synthetic examples are in `examples/`.

Each catalog operation declares `x-owner-capability`. The value must be one of
the approved capability boundaries:

`master-data`, `general-ledger`, `accounts-payable`, `accounts-receivable`,
`payroll`, `invoicing`, `payments`, `reporting`, `intercompany`,
`revenue-recognition`, `fixed-assets`, `multi-currency`, `fiscal-periods`,
`coa-segments`, `bank-reconciliation`, `tax`, `approvals`, `identity-access`,
or `audit-integrity`.

The contract is validated and bundled with the root commands `pnpm
contract:lint`, `pnpm contract:bundle`, and `pnpm contract:check`. Redocly CLI
is pinned in the root package manifest and configured by `redocly.yaml`.

The bundle at `contracts/openapi/dist/openapi.bundle.yaml` is derived output and
is ignored by Git. The pinned `ogen v1.23.0` Go generation wrapper creates a
temporary bundle before generating server interfaces and transport types under
`internal/platform/httpapi/generated/`. The pinned `@hey-api/openapi-ts 0.99.0`
TypeScript wrapper uses the same bundled contract and writes client/types under
`web/src/generated/api/`. Invalid-contract fixtures used by the focused checks
live under `contracts/openapi/fixtures/invalid/`.

Generated Go and TypeScript files are machine-produced and must not be edited
manually. This delivery item defines transport contracts and generation
workflows only. The focused `make api-check` gate and
`.github/workflows/openapi.yml` enforce contract and generated-artifact drift;
they do not implement handlers, finance behavior, authorization, persistence,
eventing, or the full `DLV-CI-001` quality pipeline.
