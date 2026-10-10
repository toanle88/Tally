-- name: GetChartOfAccountsRecord :one
SELECT chart_of_accounts_id, accounting_scope_id, ledger_id, account_code_policy,
       lifecycle_status, effective_from, effective_to, approval_reference,
       aggregate_version, revision_number, created_at, updated_at, last_audit_reference
FROM gl.chart_of_accounts
WHERE chart_of_accounts_id = $1;

-- name: ListChartOfAccountsIDs :many
SELECT chart_of_accounts_id
FROM gl.chart_of_accounts
WHERE ($1::uuid IS NULL OR accounting_scope_id = $1)
ORDER BY chart_of_accounts_id;

-- name: GetAccountRecord :one
SELECT account_id, accounting_scope_id, chart_of_accounts_id, account_code,
       account_name, account_type, normal_balance, lifecycle_status,
       restrictions, currency_policy, reporting_mappings, effective_from,
       effective_to, approval_reference, aggregate_version, revision_number,
       created_at, updated_at, last_audit_reference
FROM gl.account
WHERE account_id = $1;

-- name: ListAccountIDs :many
SELECT account_id
FROM gl.account
WHERE ($1::uuid IS NULL OR accounting_scope_id = $1)
ORDER BY account_id;
