-- name: GetLedgerRoot :one
SELECT ledger_id, accounting_scope_id, legal_entity_id, ledger_type,
       functional_currency, fiscal_calendar_id, lifecycle_status,
       effective_from, effective_to, approval_reference,
       aggregate_version, revision_number, created_at, updated_at,
       last_audit_reference
FROM gl.ledger
WHERE ledger_id = sqlc.arg(ledger_id);

-- name: ListLedgerRoots :many
SELECT ledger_id, accounting_scope_id, legal_entity_id, ledger_type,
       functional_currency, fiscal_calendar_id, lifecycle_status,
       effective_from, effective_to, approval_reference,
       aggregate_version, revision_number, created_at, updated_at,
       last_audit_reference
FROM gl.ledger
WHERE (sqlc.narg(accounting_scope_id)::uuid IS NULL OR accounting_scope_id = sqlc.narg(accounting_scope_id)::uuid)
ORDER BY ledger_id;

-- name: GetAccountingBookRoot :one
SELECT accounting_book_id, accounting_scope_id, ledger_id, book_type,
       accounting_basis, posting_policy_version, lifecycle_status,
       effective_from, effective_to, approval_reference,
       aggregate_version, revision_number, created_at, updated_at,
       last_audit_reference
FROM gl.accounting_book
WHERE accounting_book_id = sqlc.arg(accounting_book_id);

-- name: ListAccountingBookRoots :many
SELECT accounting_book_id, accounting_scope_id, ledger_id, book_type,
       accounting_basis, posting_policy_version, lifecycle_status,
       effective_from, effective_to, approval_reference,
       aggregate_version, revision_number, created_at, updated_at,
       last_audit_reference
FROM gl.accounting_book
WHERE (sqlc.narg(accounting_scope_id)::uuid IS NULL OR accounting_scope_id = sqlc.narg(accounting_scope_id)::uuid)
ORDER BY accounting_book_id;
