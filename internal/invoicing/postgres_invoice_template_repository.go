package invoicing

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type PostgresInvoiceTemplateAuditWriter func(context.Context, pgx.Tx, InvoiceTemplateAuditRecord) (uuid.UUID, error)

type PostgresInvoiceTemplateRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresInvoiceTemplateAuditWriter
}

func NewPostgresInvoiceTemplateRepository(pool *pgxpool.Pool, auditWriter PostgresInvoiceTemplateAuditWriter) (*PostgresInvoiceTemplateRepository, error) {
	if pool == nil {
		return nil, ErrInvalidInvoiceTemplateService
	}
	return &PostgresInvoiceTemplateRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresInvoiceTemplateRepository) GetInvoiceTemplate(ctx context.Context, id uuid.UUID) (InvoiceTemplate, error) {
	if repository == nil || repository.pool == nil {
		return InvoiceTemplate{}, ErrInvalidInvoiceTemplateService
	}
	return readInvoiceTemplate(ctx, repository.pool, id)
}

func (repository *PostgresInvoiceTemplateRepository) ListInvoiceTemplates(ctx context.Context, scopeID *uuid.UUID) ([]InvoiceTemplate, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidInvoiceTemplateService
	}
	rows, err := repository.pool.Query(ctx, `SELECT invoice_template_id FROM invoicing.invoice_template WHERE ($1::uuid IS NULL OR accounting_scope_id = $1) ORDER BY code, invoice_template_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapInvoiceTemplatePostgresError(err)
	}
	defer rows.Close()
	result := make([]InvoiceTemplate, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapInvoiceTemplatePostgresError(err)
		}
		template, err := readInvoiceTemplate(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, template)
	}
	if err := rows.Err(); err != nil {
		return nil, mapInvoiceTemplatePostgresError(err)
	}
	sortInvoiceTemplates(result)
	return result, nil
}

func (repository *PostgresInvoiceTemplateRepository) GetInvoiceTemplateRevision(ctx context.Context, id uuid.UUID, version aggregateversion.AggregateVersion) (InvoiceTemplateRevision, error) {
	template, err := repository.GetInvoiceTemplate(ctx, id)
	if err != nil {
		return InvoiceTemplateRevision{}, err
	}
	for _, revision := range template.Revisions {
		if revision.Snapshot.Version == version {
			return revision, nil
		}
	}
	return InvoiceTemplateRevision{}, ErrInvoiceTemplateRevisionNotFound
}

func (repository *PostgresInvoiceTemplateRepository) ListInvoiceTemplateRevisions(ctx context.Context, id uuid.UUID) ([]InvoiceTemplateRevision, error) {
	template, err := repository.GetInvoiceTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	return append([]InvoiceTemplateRevision(nil), template.Revisions...), nil
}

func (repository *PostgresInvoiceTemplateRepository) CommitInvoiceTemplateMutation(ctx context.Context, mutation InvoiceTemplateMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresInvoiceTemplateRepository) CommitInvoiceTemplateMutationWithIdempotency(ctx context.Context, mutation InvoiceTemplateMutation, durable DurableInvoiceTemplateMutationCommit) error {
	return repository.commit(ctx, mutation, &durable)
}

func (repository *PostgresInvoiceTemplateRepository) commit(ctx context.Context, mutation InvoiceTemplateMutation, durable *DurableInvoiceTemplateMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidInvoiceTemplateService
	}
	if repository.auditWriter == nil {
		return ErrInvoiceTemplateAuditUnavailable
	}
	if err := validateInvoiceTemplateMutation(mutation); err != nil {
		return err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, mutation.After.AccountingScopeID.String()); err != nil {
		return mapInvoiceTemplatePostgresError(err)
	}
	if err := repository.ensureNoOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}
	paymentMethod, err := json.Marshal(mutation.After.PaymentMethod)
	if err != nil {
		return err
	}
	applicability, err := json.Marshal(mutation.After.Applicability)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO invoicing.invoice_template (invoice_template_id, accounting_scope_id, code, name, billing_frequency, payment_method, applicability, effective_from, effective_to, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, mutation.After.ID, mutation.After.AccountingScopeID, mutation.After.Code, mutation.After.Name, mutation.After.BillingFrequency, paymentMethod, applicability, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE invoicing.invoice_template SET code=$1, name=$2, billing_frequency=$3, payment_method=$4, applicability=$5, effective_from=$6, effective_to=$7, aggregate_version=$8, revision_number=$9, updated_at=$10 WHERE invoice_template_id=$11 AND accounting_scope_id=$12 AND aggregate_version=$13`, mutation.After.Code, mutation.After.Name, mutation.After.BillingFrequency, paymentMethod, applicability, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.After.AccountingScopeID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrInvoiceTemplateVersionConflict
		}
	}
	if err != nil {
		return mapInvoiceTemplatePostgresError(err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM invoicing.invoice_template_line_item WHERE invoice_template_id=$1`, mutation.After.ID); err != nil {
		return mapInvoiceTemplatePostgresError(err)
	}
	for index, item := range mutation.After.LineItems {
		chargeRule, marshalErr := json.Marshal(item.ChargeRule)
		if marshalErr != nil {
			return marshalErr
		}
		taxCategory, marshalErr := json.Marshal(item.TaxCategory)
		if marshalErr != nil {
			return marshalErr
		}
		if _, err := tx.Exec(ctx, `INSERT INTO invoicing.invoice_template_line_item (invoice_template_id, line_item_id, line_order, code, description, charge_rule, tax_category) VALUES ($1,$2,$3,$4,$5,$6,$7)`, mutation.After.ID, item.ID, index, item.Code, item.Description, chargeRule, taxCategory); err != nil {
			return mapInvoiceTemplatePostgresError(err)
		}
	}
	if err := insertInvoiceTemplateRevision(ctx, tx, mutation.After); err != nil {
		return err
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrInvoiceTemplateAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE invoicing.invoice_template SET last_audit_reference=$1 WHERE invoice_template_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapInvoiceTemplatePostgresError(err)
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidInvoiceTemplateService
		}
		if err := durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, durable.Result); err != nil {
			return err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

type invoiceTemplateQueryer interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func readInvoiceTemplate(ctx context.Context, queryer invoiceTemplateQueryer, id uuid.UUID) (InvoiceTemplate, error) {
	var template InvoiceTemplate
	var effectiveTo *time.Time
	var paymentBytes, applicabilityBytes []byte
	var lastAudit *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT invoice_template_id, accounting_scope_id, code, name, billing_frequency, payment_method, applicability, effective_from, effective_to, aggregate_version, revision_number, created_at, updated_at, last_audit_reference FROM invoicing.invoice_template WHERE invoice_template_id=$1`, id).Scan(&template.ID, &template.AccountingScopeID, &template.Code, &template.Name, &template.BillingFrequency, &paymentBytes, &applicabilityBytes, &template.EffectiveDateFrom, &effectiveTo, &version, &revision, &template.CreatedAt, &template.UpdatedAt, &lastAudit)
	if errors.Is(err, pgx.ErrNoRows) {
		return InvoiceTemplate{}, ErrInvoiceTemplateNotFound
	}
	if err != nil {
		return InvoiceTemplate{}, mapInvoiceTemplatePostgresError(err)
	}
	template.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return InvoiceTemplate{}, err
	}
	template.RevisionNumber = revision
	template.EffectiveDateFrom = dateOnly(template.EffectiveDateFrom)
	template.EffectiveDateTo = cloneDate(effectiveTo)
	if lastAudit != nil {
		template.LastAuditReference = *lastAudit
	}
	if err := json.Unmarshal(paymentBytes, &template.PaymentMethod); err != nil {
		return InvoiceTemplate{}, err
	}
	if err := json.Unmarshal(applicabilityBytes, &template.Applicability); err != nil {
		return InvoiceTemplate{}, err
	}
	rows, err := queryer.Query(ctx, `SELECT line_item_id, code, description, charge_rule, tax_category FROM invoicing.invoice_template_line_item WHERE invoice_template_id=$1 ORDER BY line_order`, id)
	if err != nil {
		return InvoiceTemplate{}, mapInvoiceTemplatePostgresError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var item TemplateLineItem
		var chargeRule, taxCategory []byte
		if err := rows.Scan(&item.ID, &item.Code, &item.Description, &chargeRule, &taxCategory); err != nil {
			return InvoiceTemplate{}, mapInvoiceTemplatePostgresError(err)
		}
		if err := json.Unmarshal(chargeRule, &item.ChargeRule); err != nil {
			return InvoiceTemplate{}, err
		}
		if err := json.Unmarshal(taxCategory, &item.TaxCategory); err != nil {
			return InvoiceTemplate{}, err
		}
		template.LineItems = append(template.LineItems, item)
	}
	if err := rows.Err(); err != nil {
		return InvoiceTemplate{}, mapInvoiceTemplatePostgresError(err)
	}
	template.Revisions, err = queryInvoiceTemplateRevisions(ctx, queryer, id)
	if err != nil {
		return InvoiceTemplate{}, err
	}
	return template, nil
}

func queryInvoiceTemplateRevisions(ctx context.Context, queryer invoiceTemplateQueryer, id uuid.UUID) ([]InvoiceTemplateRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, snapshot, snapshot_fingerprint, created_at FROM invoicing.invoice_template_revision WHERE invoice_template_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapInvoiceTemplatePostgresError(err)
	}
	defer rows.Close()
	result := make([]InvoiceTemplateRevision, 0)
	for rows.Next() {
		var revision InvoiceTemplateRevision
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &snapshot, &revision.Fingerprint, &revision.CapturedAt); err != nil {
			return nil, mapInvoiceTemplatePostgresError(err)
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, mapInvoiceTemplatePostgresError(err)
	}
	return result, nil
}

func insertInvoiceTemplateRevision(ctx context.Context, tx pgx.Tx, template InvoiceTemplate) error {
	snapshot, err := json.Marshal(template.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO invoicing.invoice_template_revision (invoice_template_id, revision_number, aggregate_version, snapshot, snapshot_fingerprint, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`, template.ID, template.RevisionNumber, template.Version.Value(), snapshot, FingerprintInvoiceTemplate(template), template.EffectiveDateFrom, template.EffectiveDateTo, template.UpdatedAt)
	return mapInvoiceTemplatePostgresError(err)
}

func (repository *PostgresInvoiceTemplateRepository) ensureNoOverlap(ctx context.Context, tx pgx.Tx, candidate InvoiceTemplate) error {
	rows, err := tx.Query(ctx, `SELECT invoice_template_id, applicability FROM invoicing.invoice_template WHERE accounting_scope_id=$1 AND code=$2 AND invoice_template_id<>$3 AND effective_from <= COALESCE($4::date, DATE '9999-12-31') AND COALESCE(effective_to, DATE '9999-12-31') >= $5::date FOR UPDATE`, candidate.AccountingScopeID, candidate.Code, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom)
	if err != nil {
		return mapInvoiceTemplatePostgresError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var existingID uuid.UUID
		var applicabilityBytes []byte
		if err := rows.Scan(&existingID, &applicabilityBytes); err != nil {
			return mapInvoiceTemplatePostgresError(err)
		}
		var existingApplicability Applicability
		if err := json.Unmarshal(applicabilityBytes, &existingApplicability); err != nil {
			return err
		}
		if applicabilityOverlaps(existingApplicability, candidate.Applicability) {
			return ErrInvoiceTemplateDuplicate
		}
	}
	if err := rows.Err(); err != nil {
		return mapInvoiceTemplatePostgresError(err)
	}
	return nil
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapInvoiceTemplatePostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "invoice_template_effective_check", "invoice_template_frequency_check", "invoice_template_code_check", "invoice_template_name_check", "invoice_template_payment_method_object_check", "invoice_template_applicability_object_check":
			return ErrInvalidInvoiceTemplate
		case "invoice_template_line_item_fk", "invoice_template_line_item_code_check", "invoice_template_line_item_description_check", "invoice_template_line_item_charge_rule_object_check", "invoice_template_line_item_tax_category_object_check":
			return ErrInvalidInvoiceTemplate
		case "invoice_template_revision_effective_check", "invoice_template_revision_fingerprint_check":
			return ErrInvalidInvoiceTemplate
		}
	}
	return err
}

var _ InvoiceTemplateRepository = (*PostgresInvoiceTemplateRepository)(nil)
var _ DurableInvoiceTemplateRepository = (*PostgresInvoiceTemplateRepository)(nil)
