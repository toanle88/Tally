# DLV-UX-001 User Story 5 verification

Branch: `feat/dlv-ux-001-us5-form-approval-posting-validation-conflict`

Story 5 adds the shared form, validation, approval, posting, confirmation,
typed-outcome, and concurrency-conflict presentation boundary. All data is
synthetic; no finance capability, API mutation, persistence, authentication,
authorization policy, or authoritative financial state was added.

Implemented evidence is in `web/src/app/workflow-example.tsx`,
`web/src/app/workflow-fixtures.ts`, `web/src/components/workflow-context/`,
and the four Story 5 component directories.

## Verification

| Command | Result |
|---|---|
| `pnpm -C web test` | Passed |
| `pnpm -C web build` | Passed |
| `git diff --check` | Passed |

The focused tests cover canonical locale decimal conversion, linked validation
issues, approval decision/application separation, deliberate posting retry, and
conflict evidence/recovery controls. No provider request is automatically retried;
identity-content conflicts require a new business identity.
