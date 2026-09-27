package organization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type PostgresAuditWriter func(context.Context, pgx.Tx, AuditRecord) (uuid.UUID, error)

type PostgresLegalEntityRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresAuditWriter
}

func NewPostgresLegalEntityRepository(pool *pgxpool.Pool, auditWriter PostgresAuditWriter) (*PostgresLegalEntityRepository, error) {
	if pool == nil {
		return nil, ErrInvalidLegalEntityService
	}
	return &PostgresLegalEntityRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresLegalEntityRepository) Get(ctx context.Context, id uuid.UUID) (LegalEntity, error) {
	if repository == nil || repository.pool == nil {
		return LegalEntity{}, ErrInvalidLegalEntityService
	}
	return readLegalEntity(ctx, repository.pool, id)
}
func (repository *PostgresLegalEntityRepository) List(ctx context.Context, scopeID *uuid.UUID) ([]LegalEntity, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidLegalEntityService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM organization.legal_entity WHERE ($1::uuid IS NULL OR scope_id = $1) ORDER BY id`, nullableUUID(scopeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]LegalEntity, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		entity, err := readLegalEntity(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, entity)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (repository *PostgresLegalEntityRepository) CommitLegalEntityMutation(ctx context.Context, mutation LegalEntityMutation) error {
	return repository.commit(ctx, mutation, nil)
}
func (repository *PostgresLegalEntityRepository) CommitLegalEntityMutationWithIdempotency(ctx context.Context, mutation LegalEntityMutation, commit DurableLegalEntityMutationCommit) error {
	return repository.commit(ctx, mutation, &commit)
}

func (repository *PostgresLegalEntityRepository) commit(ctx context.Context, mutation LegalEntityMutation, durable *DurableLegalEntityMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidLegalEntityService
	}
	if repository.auditWriter == nil {
		return ErrLegalEntityAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrLegalEntityVersionConflict
		}
	} else if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrLegalEntityVersionConflict
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
	approval, err := json.Marshal(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.After.Approval == nil {
		approval = nil
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO organization.legal_entity (id, scope_id, legal_name, functional_currency, presentation_currency, tax_registration_id, status, effective_from, effective_to, aggregate_version, revision_number, approval_reference, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`, mutation.After.ID, mutation.After.ScopeID, mutation.After.LegalName, mutation.After.FunctionalCurrency, mutation.After.PresentationCurrency, nullableText(mutation.After.TaxRegistrationID), mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.Version.Value(), mutation.After.RevisionNumber, approval, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE organization.legal_entity SET legal_name=$1, functional_currency=$2, presentation_currency=$3, tax_registration_id=$4, status=$5, effective_from=$6, effective_to=$7, aggregate_version=$8, revision_number=$9, approval_reference=$10, updated_at=$11 WHERE id=$12 AND aggregate_version=$13`, mutation.After.LegalName, mutation.After.FunctionalCurrency, mutation.After.PresentationCurrency, nullableText(mutation.After.TaxRegistrationID), mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.Version.Value(), mutation.After.RevisionNumber, approval, mutation.After.UpdatedAt, mutation.After.ID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrLegalEntityVersionConflict
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM organization.legal_entity_registration WHERE legal_entity_id=$1`, mutation.After.ID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM organization.legal_entity_address WHERE legal_entity_id=$1`, mutation.After.ID)
		}
		if err == nil {
			_, err = tx.Exec(ctx, `DELETE FROM organization.legal_entity_ownership_interest WHERE legal_entity_id=$1`, mutation.After.ID)
		}
	}
	if err != nil {
		return mapOrganizationPostgresError(err)
	}
	for _, value := range mutation.After.Registrations {
		_, err = tx.Exec(ctx, `INSERT INTO organization.legal_entity_registration (id, legal_entity_id, registration_type, identifier, jurisdiction, effective_from, effective_to) VALUES ($1,$2,$3,$4,$5,$6,$7)`, value.ID, mutation.After.ID, value.Type, value.Identifier, value.Jurisdiction, value.EffectiveFrom, value.EffectiveTo)
		if err != nil {
			return mapOrganizationPostgresError(err)
		}
	}
	for _, value := range mutation.After.Addresses {
		_, err = tx.Exec(ctx, `INSERT INTO organization.legal_entity_address (id, legal_entity_id, address_type, line1, line2, locality, region, postal_code, country_code, effective_from, effective_to) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, value.ID, mutation.After.ID, value.Type, value.Line1, nullableText(value.Line2), value.Locality, nullableText(value.Region), value.PostalCode, value.CountryCode, value.EffectiveFrom, value.EffectiveTo)
		if err != nil {
			return mapOrganizationPostgresError(err)
		}
	}
	for _, value := range mutation.After.OwnershipInterests {
		_, err = tx.Exec(ctx, `INSERT INTO organization.legal_entity_ownership_interest (id, legal_entity_id, owner_reference, percentage, effective_from, effective_to) VALUES ($1,$2,$3,$4,$5,$6)`, value.ID, mutation.After.ID, value.OwnerReference, value.Percentage, value.EffectiveFrom, value.EffectiveTo)
		if err != nil {
			return mapOrganizationPostgresError(err)
		}
	}
	snapshot, err := json.Marshal(mutation.After.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO organization.legal_entity_revision (legal_entity_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, mutation.After.ID, mutation.After.RevisionNumber, mutation.After.Version.Value(), snapshot, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.UpdatedAt)
	if err != nil {
		return mapOrganizationPostgresError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrLegalEntityAuditUnavailable
	}
	_, err = tx.Exec(ctx, `UPDATE organization.legal_entity SET last_audit_reference=$1 WHERE id=$2`, auditReference, mutation.After.ID)
	if err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidLegalEntityService
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

type pgxQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readLegalEntity(ctx context.Context, queryer pgxQueryer, id uuid.UUID) (LegalEntity, error) {
	var entity LegalEntity
	var tax *string
	var effectiveTo *time.Time
	var approvalJSON []byte
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT id, scope_id, legal_name, functional_currency, presentation_currency, tax_registration_id, status, effective_from, effective_to, aggregate_version, revision_number, approval_reference, created_at, updated_at FROM organization.legal_entity WHERE id=$1`, id).Scan(&entity.ID, &entity.ScopeID, &entity.LegalName, &entity.FunctionalCurrency, &entity.PresentationCurrency, &tax, &entity.Status, &entity.EffectiveFrom, &effectiveTo, &version, &revision, &approvalJSON, &entity.CreatedAt, &entity.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return LegalEntity{}, ErrLegalEntityNotFound
	}
	if err != nil {
		return LegalEntity{}, err
	}
	parsedVersion, err := aggregateversion.FromInt64(version)
	if err != nil {
		return LegalEntity{}, err
	}
	entity.Version, entity.RevisionNumber, entity.EffectiveFrom = parsedVersion, revision, dateOnly(entity.EffectiveFrom)
	if effectiveTo != nil {
		entity.EffectiveTo = ptrDate(*effectiveTo)
	}
	if tax != nil {
		entity.TaxRegistrationID = *tax
	}
	if len(approvalJSON) > 0 {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalJSON, &approval); err != nil {
			return LegalEntity{}, err
		}
		entity.Approval = &approval
	}
	registrations, err := queryRegistrations(ctx, queryer, id)
	if err != nil {
		return LegalEntity{}, err
	}
	addresses, err := queryAddresses(ctx, queryer, id)
	if err != nil {
		return LegalEntity{}, err
	}
	ownership, err := queryOwnership(ctx, queryer, id)
	if err != nil {
		return LegalEntity{}, err
	}
	entity.Registrations, entity.Addresses, entity.OwnershipInterests = registrations, addresses, ownership
	revisions, err := queryRevisions(ctx, queryer, id)
	if err != nil {
		return LegalEntity{}, err
	}
	entity.Revisions = revisions
	if err := entity.Validate(); err != nil {
		return LegalEntity{}, err
	}
	return entity, nil
}

func queryRegistrations(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]LegalEntityRegistration, error) {
	rows, err := queryer.Query(ctx, `SELECT id, registration_type, identifier, jurisdiction, effective_from, effective_to FROM organization.legal_entity_registration WHERE legal_entity_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]LegalEntityRegistration, 0)
	for rows.Next() {
		var value LegalEntityRegistration
		if err := rows.Scan(&value.ID, &value.Type, &value.Identifier, &value.Jurisdiction, &value.EffectiveFrom, &value.EffectiveTo); err != nil {
			return nil, err
		}
		value.EffectiveFrom = dateOnly(value.EffectiveFrom)
		if value.EffectiveTo != nil {
			value.EffectiveTo = ptrDate(*value.EffectiveTo)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func queryAddresses(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]LegalEntityAddress, error) {
	rows, err := queryer.Query(ctx, `SELECT id, address_type, line1, line2, locality, region, postal_code, country_code, effective_from, effective_to FROM organization.legal_entity_address WHERE legal_entity_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]LegalEntityAddress, 0)
	for rows.Next() {
		var value LegalEntityAddress
		var line2, region *string
		if err := rows.Scan(&value.ID, &value.Type, &value.Line1, &line2, &value.Locality, &region, &value.PostalCode, &value.CountryCode, &value.EffectiveFrom, &value.EffectiveTo); err != nil {
			return nil, err
		}
		if line2 != nil {
			value.Line2 = *line2
		}
		if region != nil {
			value.Region = *region
		}
		value.EffectiveFrom = dateOnly(value.EffectiveFrom)
		if value.EffectiveTo != nil {
			value.EffectiveTo = ptrDate(*value.EffectiveTo)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func queryOwnership(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]LegalEntityOwnershipInterest, error) {
	rows, err := queryer.Query(ctx, `SELECT id, owner_reference, percentage::text, effective_from, effective_to FROM organization.legal_entity_ownership_interest WHERE legal_entity_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]LegalEntityOwnershipInterest, 0)
	for rows.Next() {
		var value LegalEntityOwnershipInterest
		if err := rows.Scan(&value.ID, &value.OwnerReference, &value.Percentage, &value.EffectiveFrom, &value.EffectiveTo); err != nil {
			return nil, err
		}
		value.EffectiveFrom = dateOnly(value.EffectiveFrom)
		if value.EffectiveTo != nil {
			value.EffectiveTo = ptrDate(*value.EffectiveTo)
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
func queryRevisions(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]LegalEntityRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM organization.legal_entity_revision WHERE legal_entity_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]LegalEntityRevision, 0)
	for rows.Next() {
		var value LegalEntityRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&value.RevisionNumber, &version, &snapshot, &value.CreatedAt); err != nil {
			return nil, err
		}
		value.Version, err = aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &value.Snapshot); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func mapOrganizationPostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrLegalEntityDuplicate
	}
	return err
}
func nullableText(value string) *string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return &value
}
func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

var _ LegalEntityRepository = (*PostgresLegalEntityRepository)(nil)
var _ DurableLegalEntityRepository = (*PostgresLegalEntityRepository)(nil)
var _ platformidempotency.TxBeginner = (*pgxpool.Pool)(nil)
