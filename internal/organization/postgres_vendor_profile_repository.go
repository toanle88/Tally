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

type PostgresVendorProfileAuditWriter func(context.Context, pgx.Tx, VendorProfileAuditRecord) (uuid.UUID, error)

type PostgresVendorProfileRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresVendorProfileAuditWriter
}

func NewPostgresVendorProfileRepository(pool *pgxpool.Pool, auditWriter PostgresVendorProfileAuditWriter) (*PostgresVendorProfileRepository, error) {
	if pool == nil {
		return nil, ErrInvalidVendorProfileService
	}
	return &PostgresVendorProfileRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresVendorProfileRepository) Get(ctx context.Context, id uuid.UUID) (VendorProfile, error) {
	if repository == nil || repository.pool == nil {
		return VendorProfile{}, ErrInvalidVendorProfileService
	}
	return readVendorProfile(ctx, repository.pool, id)
}

func (repository *PostgresVendorProfileRepository) List(ctx context.Context, scopeID *uuid.UUID) ([]VendorProfile, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidVendorProfileService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM organization.vendor_profile WHERE ($1::uuid IS NULL OR scope_id = $1) ORDER BY id`, nullableUUID(scopeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]VendorProfile, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		profile, err := readVendorProfile(ctx, repository.pool, id)
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

func (repository *PostgresVendorProfileRepository) CommitVendorProfileMutation(ctx context.Context, mutation VendorProfileMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresVendorProfileRepository) CommitVendorProfileMutationWithIdempotency(ctx context.Context, mutation VendorProfileMutation, durable DurableVendorProfileMutationCommit) error {
	return repository.commit(ctx, mutation, &durable)
}

func (repository *PostgresVendorProfileRepository) commit(ctx context.Context, mutation VendorProfileMutation, durable *DurableVendorProfileMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidVendorProfileService
	}
	if repository.auditWriter == nil {
		return ErrVendorProfileAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.ExpectedPartyVersion == nil {
		return ErrVendorProfilePartyVersionConflict
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrVendorProfileVersionConflict
		}
	} else if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrVendorProfileVersionConflict
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
		return ErrVendorProfilePartyNotFound
	}
	if err != nil {
		return err
	}
	partyVersion, err := aggregateversion.FromInt64(partyVersionValue)
	if err != nil {
		return err
	}
	if partyScope != mutation.After.ScopeID || strings.ToLower(partyType) != "vendor" {
		return ErrVendorProfilePartyInvalid
	}
	if !partyVersion.Matches(*mutation.ExpectedPartyVersion) || !mutation.After.PartyVersion.Matches(*mutation.ExpectedPartyVersion) {
		return ErrVendorProfilePartyVersionConflict
	}

	approval, err := vendorProfileApprovalJSON(mutation.After.Approval)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO organization.vendor_profile (id, scope_id, party_id, party_version, payment_terms, withholding_treatment, remittance_preference, status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)`, mutation.After.ID, mutation.After.ScopeID, mutation.After.PartyID, mutation.After.PartyVersion.Value(), mutation.After.PaymentTerms, mutation.After.WithholdingTreatment, mutation.After.RemittancePreference, mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var currentVersion int64
		var currentScope, currentParty uuid.UUID
		err = tx.QueryRow(ctx, `SELECT aggregate_version, scope_id, party_id FROM organization.vendor_profile WHERE id=$1 FOR UPDATE`, mutation.After.ID).Scan(&currentVersion, &currentScope, &currentParty)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrVendorProfileNotFound
		}
		if err != nil {
			return err
		}
		if currentScope != mutation.After.ScopeID || currentParty != mutation.After.PartyID || currentVersion != mutation.ExpectedVersion.Value() {
			return ErrVendorProfileVersionConflict
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE organization.vendor_profile SET party_version=$1, payment_terms=$2, withholding_treatment=$3, remittance_preference=$4, status=$5, effective_from=$6, effective_to=$7, approval_reference=$8, aggregate_version=$9, revision_number=$10, updated_at=$11 WHERE id=$12 AND aggregate_version=$13`, mutation.After.PartyVersion.Value(), mutation.After.PaymentTerms, mutation.After.WithholdingTreatment, mutation.After.RemittancePreference, mutation.After.Status, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrVendorProfileVersionConflict
		}
	}
	if err != nil {
		return mapVendorProfilePostgresError(err)
	}
	snapshot, err := json.Marshal(mutation.After.Snapshot())
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO organization.vendor_profile_revision (vendor_profile_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, mutation.After.ID, mutation.After.RevisionNumber, mutation.After.Version.Value(), snapshot, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.UpdatedAt); err != nil {
		return mapVendorProfilePostgresError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrVendorProfileAuditUnavailable
	}
	if _, err = tx.Exec(ctx, `UPDATE organization.vendor_profile SET last_audit_reference=$1 WHERE id=$2`, auditReference, mutation.After.ID); err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidVendorProfileService
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

func vendorProfileApprovalJSON(value *ApprovalDecisionReference) (any, error) {
	if value == nil {
		return nil, nil
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func readVendorProfile(ctx context.Context, queryer pgxQueryer, id uuid.UUID) (VendorProfile, error) {
	var profile VendorProfile
	var effectiveTo *time.Time
	var approvalJSON []byte
	var partyVersion, version, revision int64
	err := queryer.QueryRow(ctx, `SELECT id, scope_id, party_id, party_version, payment_terms, withholding_treatment, remittance_preference, status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at FROM organization.vendor_profile WHERE id=$1`, id).Scan(&profile.ID, &profile.ScopeID, &profile.PartyID, &partyVersion, &profile.PaymentTerms, &profile.WithholdingTreatment, &profile.RemittancePreference, &profile.Status, &profile.EffectiveFrom, &effectiveTo, &approvalJSON, &version, &revision, &profile.CreatedAt, &profile.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return VendorProfile{}, ErrVendorProfileNotFound
	}
	if err != nil {
		return VendorProfile{}, err
	}
	profile.PartyVersion, err = aggregateversion.FromInt64(partyVersion)
	if err != nil {
		return VendorProfile{}, err
	}
	profile.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return VendorProfile{}, err
	}
	profile.RevisionNumber = revision
	profile.EffectiveFrom = dateOnly(profile.EffectiveFrom)
	if effectiveTo != nil {
		profile.EffectiveTo = ptrDate(*effectiveTo)
	}
	if len(approvalJSON) > 0 && string(approvalJSON) != "null" {
		var approval ApprovalDecisionReference
		if err := json.Unmarshal(approvalJSON, &approval); err != nil {
			return VendorProfile{}, err
		}
		profile.Approval = &approval
	}
	profile.Revisions, err = queryVendorProfileRevisions(ctx, queryer, id)
	if err != nil {
		return VendorProfile{}, err
	}
	if err := profile.Validate(); err != nil {
		return VendorProfile{}, err
	}
	return profile, nil
}

func queryVendorProfileRevisions(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]VendorProfileRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM organization.vendor_profile_revision WHERE vendor_profile_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]VendorProfileRevision, 0)
	for rows.Next() {
		var revision VendorProfileRevision
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

func mapVendorProfilePostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrVendorProfileDuplicate
		case "23503":
			return ErrVendorProfilePartyNotFound
		case "23514":
			return ErrInvalidVendorProfile
		}
	}
	return err
}

var _ VendorProfileRepository = (*PostgresVendorProfileRepository)(nil)
var _ DurableVendorProfileRepository = (*PostgresVendorProfileRepository)(nil)
