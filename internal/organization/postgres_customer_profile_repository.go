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
)

type PostgresCustomerProfileAuditWriter func(context.Context, pgx.Tx, CustomerProfileAuditRecord) (uuid.UUID, error)

type PostgresCustomerProfileRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresCustomerProfileAuditWriter
}

func NewPostgresCustomerProfileRepository(pool *pgxpool.Pool, auditWriter PostgresCustomerProfileAuditWriter) (*PostgresCustomerProfileRepository, error) {
	if pool == nil {
		return nil, ErrInvalidCustomerProfileService
	}
	return &PostgresCustomerProfileRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresCustomerProfileRepository) Get(ctx context.Context, id uuid.UUID) (CustomerProfile, error) {
	if repository == nil || repository.pool == nil {
		return CustomerProfile{}, ErrInvalidCustomerProfileService
	}
	return readCustomerProfile(ctx, repository.pool, id)
}

func (repository *PostgresCustomerProfileRepository) List(ctx context.Context, scopeID *uuid.UUID) ([]CustomerProfile, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidCustomerProfileService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM organization.customer_profile WHERE ($1::uuid IS NULL OR scope_id = $1) ORDER BY id`, nullableUUID(scopeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]CustomerProfile, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		profile, err := readCustomerProfile(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (repository *PostgresCustomerProfileRepository) CommitCustomerProfileMutation(ctx context.Context, mutation CustomerProfileMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresCustomerProfileRepository) CommitCustomerProfileMutationWithIdempotency(ctx context.Context, mutation CustomerProfileMutation, durable DurableCustomerProfileMutationCommit) error {
	return repository.commit(ctx, mutation, &durable)
}

func (repository *PostgresCustomerProfileRepository) commit(ctx context.Context, mutation CustomerProfileMutation, durable *DurableCustomerProfileMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidCustomerProfileService
	}
	if repository.auditWriter == nil {
		return ErrCustomerProfileAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.ExpectedPartyVersion == nil {
		return ErrCustomerProfilePartyVersionConflict
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrCustomerProfileVersionConflict
		}
	} else if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrCustomerProfileVersionConflict
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

	var partyScope uuid.UUID
	var partyType string
	var partyVersionValue int64
	err = tx.QueryRow(ctx, `SELECT scope_id, party_type, aggregate_version FROM organization.party WHERE id=$1 FOR UPDATE`, mutation.After.PartyID).Scan(&partyScope, &partyType, &partyVersionValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrCustomerProfilePartyNotFound
	}
	if err != nil {
		return err
	}
	partyVersion, err := aggregateversion.FromInt64(partyVersionValue)
	if err != nil {
		return err
	}
	if partyScope != mutation.After.ScopeID || strings.ToLower(partyType) != "customer" {
		return ErrCustomerProfilePartyInvalid
	}
	if !partyVersion.Matches(*mutation.ExpectedPartyVersion) {
		return ErrCustomerProfilePartyVersionConflict
	}
	if !mutation.After.PartyVersion.Matches(*mutation.ExpectedPartyVersion) {
		return ErrCustomerProfilePartyVersionConflict
	}

	approval, err := customerProfileApprovalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO organization.customer_profile (id, scope_id, party_id, party_version, credit_terms, credit_limit_amount, credit_limit_currency, billing_preference, tax_treatment, status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, mutation.After.ID, mutation.After.ScopeID, mutation.After.PartyID, mutation.After.PartyVersion.Value(), mutation.After.CreditTerms, mutation.After.CreditLimit.Amount, mutation.After.CreditLimit.Currency, mutation.After.BillingPreference, mutation.After.TaxTreatment, mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var currentVersion int64
		var currentScope, currentParty uuid.UUID
		err = tx.QueryRow(ctx, `SELECT aggregate_version, scope_id, party_id FROM organization.customer_profile WHERE id=$1 FOR UPDATE`, mutation.After.ID).Scan(&currentVersion, &currentScope, &currentParty)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrCustomerProfileNotFound
		}
		if err != nil {
			return err
		}
		if currentScope != mutation.After.ScopeID || currentParty != mutation.After.PartyID || currentVersion != mutation.ExpectedVersion.Value() {
			return ErrCustomerProfileVersionConflict
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE organization.customer_profile SET party_version=$1, credit_terms=$2, credit_limit_amount=$3, credit_limit_currency=$4, billing_preference=$5, tax_treatment=$6, status=$7, effective_from=$8, effective_to=$9, approval_reference=$10, aggregate_version=$11, revision_number=$12, updated_at=$13 WHERE id=$14 AND aggregate_version=$15`, mutation.After.PartyVersion.Value(), mutation.After.CreditTerms, mutation.After.CreditLimit.Amount, mutation.After.CreditLimit.Currency, mutation.After.BillingPreference, mutation.After.TaxTreatment, mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrCustomerProfileVersionConflict
		}
	}
	if err != nil {
		return mapCustomerProfilePostgresError(err)
	}
	snapshot, err := json.Marshal(mutation.After.Snapshot())
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO organization.customer_profile_revision (customer_profile_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, mutation.After.ID, mutation.After.RevisionNumber, mutation.After.Version.Value(), snapshot, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.UpdatedAt); err != nil {
		return mapCustomerProfilePostgresError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrCustomerProfileAuditUnavailable
	}
	if _, err = tx.Exec(ctx, `UPDATE organization.customer_profile SET last_audit_reference=$1 WHERE id=$2`, auditReference, mutation.After.ID); err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidCustomerProfileService
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

func customerProfileApprovalJSON(value *ApprovalDecisionReference) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readCustomerProfile(ctx context.Context, queryer pgxQueryer, id uuid.UUID) (CustomerProfile, error) {
	var profile CustomerProfile
	var effectiveTo *time.Time
	var approvalJSON []byte
	var amount, currency string
	var partyVersion, version, revision int64
	err := queryer.QueryRow(ctx, `SELECT id, scope_id, party_id, party_version, credit_terms, credit_limit_amount::text, credit_limit_currency, billing_preference, tax_treatment, status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at FROM organization.customer_profile WHERE id=$1`, id).Scan(&profile.ID, &profile.ScopeID, &profile.PartyID, &partyVersion, &profile.CreditTerms, &amount, &currency, &profile.BillingPreference, &profile.TaxTreatment, &profile.Status, &profile.EffectiveFrom, &effectiveTo, &approvalJSON, &version, &revision, &profile.CreatedAt, &profile.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return CustomerProfile{}, ErrCustomerProfileNotFound
	}
	if err != nil {
		return CustomerProfile{}, err
	}
	profile.PartyVersion, err = aggregateversion.FromInt64(partyVersion)
	if err != nil {
		return CustomerProfile{}, err
	}
	profile.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return CustomerProfile{}, err
	}
	profile.RevisionNumber = revision
	profile.EffectiveFrom = dateOnly(profile.EffectiveFrom)
	if effectiveTo != nil {
		profile.EffectiveTo = ptrDate(*effectiveTo)
	}
	profile.CreditLimit, err = (CreditLimit{Amount: amount, Currency: currency}).Canonicalize()
	if err != nil {
		return CustomerProfile{}, err
	}
	if len(approvalJSON) > 0 && string(approvalJSON) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalJSON, &approval); err != nil {
			return CustomerProfile{}, err
		}
		profile.Approval = &approval
	}
	profile.Revisions, err = queryCustomerProfileRevisions(ctx, queryer, id)
	if err != nil {
		return CustomerProfile{}, err
	}
	if err := profile.Validate(); err != nil {
		return CustomerProfile{}, err
	}
	return profile, nil
}

func queryCustomerProfileRevisions(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]CustomerProfileRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM organization.customer_profile_revision WHERE customer_profile_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]CustomerProfileRevision, 0)
	for rows.Next() {
		var revision CustomerProfileRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, err
		}
		revision.Version, err = aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		result = append(result, revision)
	}
	return result, rows.Err()
}

func mapCustomerProfilePostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrCustomerProfileDuplicate
		case "23503":
			return ErrCustomerProfilePartyNotFound
		case "23514":
			return ErrInvalidCustomerProfile
		}
	}
	return err
}

var _ CustomerProfileRepository = (*PostgresCustomerProfileRepository)(nil)
var _ DurableCustomerProfileRepository = (*PostgresCustomerProfileRepository)(nil)
