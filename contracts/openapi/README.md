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

This story defines transport contracts only. It does not implement handlers,
finance behavior, authorization, persistence, eventing, generation, bundling,
or CI drift checks.
