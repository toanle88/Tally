package organization

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

type PostgresPartyAuditWriter func(context.Context, pgx.Tx, PartyAuditRecord) (uuid.UUID, error)

type PostgresPartyRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresPartyAuditWriter
}

func NewPostgresPartyRepository(pool *pgxpool.Pool, auditWriter PostgresPartyAuditWriter) (*PostgresPartyRepository, error) {
	if pool == nil {
		return nil, ErrInvalidPartyService
	}
	return &PostgresPartyRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresPartyRepository) Get(ctx context.Context, id uuid.UUID) (Party, error) {
	if repository == nil || repository.pool == nil {
		return Party{}, ErrInvalidPartyService
	}
	return readParty(ctx, repository.pool, id)
}

func (repository *PostgresPartyRepository) List(ctx context.Context, scopeID *uuid.UUID) ([]Party, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidPartyService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM organization.party WHERE ($1::uuid IS NULL OR scope_id = $1) ORDER BY id`, nullableUUID(scopeID))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Party, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		party, err := readParty(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, party)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sortParties(result)
	return result, nil
}

func (repository *PostgresPartyRepository) CommitPartyMutation(ctx context.Context, mutation PartyMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresPartyRepository) CommitPartyMutationWithIdempotency(ctx context.Context, mutation PartyMutation, commit DurablePartyMutationCommit) error {
	return repository.commit(ctx, mutation, &commit)
}

func (repository *PostgresPartyRepository) commit(ctx context.Context, mutation PartyMutation, durable *DurablePartyMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidPartyService
	}
	if repository.auditWriter == nil {
		return ErrPartyAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrPartyVersionConflict
		}
	} else if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrPartyVersionConflict
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

	bankControl, err := json.Marshal(mutation.After.BankControl)
	if err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `INSERT INTO organization.party (id, scope_id, name, party_type, status, tax_identifier, bank_control, aggregate_version, revision_number, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, mutation.After.ID, mutation.After.ScopeID, mutation.After.Name, string(mutation.After.PartyType), string(mutation.After.Status), nullableText(mutation.After.TaxIdentifier), bankControl, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var currentVersion int64
		var currentScope uuid.UUID
		err = tx.QueryRow(ctx, `SELECT aggregate_version, scope_id FROM organization.party WHERE id=$1 FOR UPDATE`, mutation.After.ID).Scan(&currentVersion, &currentScope)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPartyNotFound
		}
		if err != nil {
			return err
		}
		if currentScope != mutation.After.ScopeID || currentVersion != mutation.ExpectedVersion.Value() {
			return ErrPartyVersionConflict
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE organization.party SET name=$1, party_type=$2, status=$3, tax_identifier=$4, bank_control=$5, aggregate_version=$6, revision_number=$7, updated_at=$8 WHERE id=$9 AND aggregate_version=$10`, mutation.After.Name, string(mutation.After.PartyType), string(mutation.After.Status), nullableText(mutation.After.TaxIdentifier), bankControl, mutation.After.Version.Value(), mutation.After.RevisionNumber, mutation.After.UpdatedAt, mutation.After.ID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrPartyVersionConflict
		}
		if err == nil {
			for _, query := range []string{
				`DELETE FROM organization.party_contact_method WHERE party_id=$1`,
				`DELETE FROM organization.party_address WHERE party_id=$1`,
				`DELETE FROM organization.party_classification WHERE party_id=$1`,
				`DELETE FROM organization.party_bank_detail_reference WHERE party_id=$1`,
			} {
				if _, err = tx.Exec(ctx, query, mutation.After.ID); err != nil {
					return mapPartyPostgresError(err)
				}
			}
		}
	}
	if err != nil {
		return mapPartyPostgresError(err)
	}
	for _, value := range mutation.After.ContactMethods {
		_, err = tx.Exec(ctx, `INSERT INTO organization.party_contact_method (id, party_id, contact_type, contact_value, label) VALUES ($1,$2,$3,$4,$5)`, value.ID, mutation.After.ID, value.Type, value.Value, nullableText(value.Label))
		if err != nil {
			return mapPartyPostgresError(err)
		}
	}
	for _, value := range mutation.After.Addresses {
		_, err = tx.Exec(ctx, `INSERT INTO organization.party_address (id, party_id, address_type, line1, line2, locality, region, postal_code, country_code) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, value.ID, mutation.After.ID, value.Type, value.Line1, nullableText(value.Line2), value.Locality, nullableText(value.Region), value.PostalCode, value.CountryCode)
		if err != nil {
			return mapPartyPostgresError(err)
		}
	}
	for _, value := range mutation.After.Classifications {
		_, err = tx.Exec(ctx, `INSERT INTO organization.party_classification (id, party_id, classification_code, classification_value) VALUES ($1,$2,$3,$4)`, value.ID, mutation.After.ID, value.Code, value.Value)
		if err != nil {
			return mapPartyPostgresError(err)
		}
	}
	for _, value := range mutation.After.BankDetailReferences {
		_, err = tx.Exec(ctx, `INSERT INTO organization.party_bank_detail_reference (id, party_id, reference, provider_code, consent_reference) VALUES ($1,$2,$3,$4,$5)`, value.ID, mutation.After.ID, value.Reference, nullableText(value.ProviderCode), nullableText(value.ConsentReference))
		if err != nil {
			return mapPartyPostgresError(err)
		}
	}
	snapshot, err := json.Marshal(mutation.After.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO organization.party_revision (party_id, revision_number, aggregate_version, snapshot, created_at) VALUES ($1,$2,$3,$4,$5)`, mutation.After.ID, mutation.After.RevisionNumber, mutation.After.Version.Value(), snapshot, mutation.After.UpdatedAt)
	if err != nil {
		return mapPartyPostgresError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrPartyAuditUnavailable
	}
	if _, err = tx.Exec(ctx, `UPDATE organization.party SET last_audit_reference=$1 WHERE id=$2`, auditReference, mutation.After.ID); err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidPartyService
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

func readParty(ctx context.Context, queryer pgxQueryer, id uuid.UUID) (Party, error) {
	var party Party
	var tax *string
	var partyType, status string
	var bankControlJSON []byte
	var version, revision int64
	err := queryer.QueryRow(ctx, `SELECT id, scope_id, name, party_type, status, tax_identifier, bank_control, aggregate_version, revision_number, created_at, updated_at FROM organization.party WHERE id=$1`, id).Scan(&party.ID, &party.ScopeID, &party.Name, &partyType, &status, &tax, &bankControlJSON, &version, &revision, &party.CreatedAt, &party.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Party{}, ErrPartyNotFound
	}
	if err != nil {
		return Party{}, err
	}
	party.PartyType, party.Status = PartyType(partyType), PartyStatus(status)
	party.Version, err = aggregateversion.FromInt64(version)
	if err != nil {
		return Party{}, err
	}
	party.RevisionNumber = revision
	if tax != nil {
		party.TaxIdentifier = *tax
	}
	if len(bankControlJSON) > 0 {
		if err := json.Unmarshal(bankControlJSON, &party.BankControl); err != nil {
			return Party{}, err
		}
	}
	if party.BankControl.Status == "" {
		party.BankControl.Status = PartyBankControlNotRequired
	}
	party.ContactMethods, err = queryPartyContacts(ctx, queryer, id)
	if err != nil {
		return Party{}, err
	}
	party.Addresses, err = queryPartyAddresses(ctx, queryer, id)
	if err != nil {
		return Party{}, err
	}
	party.Classifications, err = queryPartyClassifications(ctx, queryer, id)
	if err != nil {
		return Party{}, err
	}
	party.BankDetailReferences, err = queryPartyBankReferences(ctx, queryer, id)
	if err != nil {
		return Party{}, err
	}
	party.Revisions, err = queryPartyRevisions(ctx, queryer, id)
	if err != nil {
		return Party{}, err
	}
	if err := party.Validate(); err != nil {
		return Party{}, err
	}
	return party, nil
}

func queryPartyContacts(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]PartyContactMethod, error) {
	rows, err := queryer.Query(ctx, `SELECT id, contact_type, contact_value, label FROM organization.party_contact_method WHERE party_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PartyContactMethod, 0)
	for rows.Next() {
		var value PartyContactMethod
		var label *string
		if err := rows.Scan(&value.ID, &value.Type, &value.Value, &label); err != nil {
			return nil, err
		}
		if label != nil {
			value.Label = *label
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func queryPartyAddresses(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]PartyAddress, error) {
	rows, err := queryer.Query(ctx, `SELECT id, address_type, line1, line2, locality, region, postal_code, country_code FROM organization.party_address WHERE party_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PartyAddress, 0)
	for rows.Next() {
		var value PartyAddress
		var line2, region *string
		if err := rows.Scan(&value.ID, &value.Type, &value.Line1, &line2, &value.Locality, &region, &value.PostalCode, &value.CountryCode); err != nil {
			return nil, err
		}
		if line2 != nil {
			value.Line2 = *line2
		}
		if region != nil {
			value.Region = *region
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func queryPartyClassifications(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]PartyClassification, error) {
	rows, err := queryer.Query(ctx, `SELECT id, classification_code, classification_value FROM organization.party_classification WHERE party_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PartyClassification, 0)
	for rows.Next() {
		var value PartyClassification
		if err := rows.Scan(&value.ID, &value.Code, &value.Value); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func queryPartyBankReferences(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]PartyBankDetailReference, error) {
	rows, err := queryer.Query(ctx, `SELECT id, reference, provider_code, consent_reference FROM organization.party_bank_detail_reference WHERE party_id=$1 ORDER BY id`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PartyBankDetailReference, 0)
	for rows.Next() {
		var value PartyBankDetailReference
		var provider, consent *string
		if err := rows.Scan(&value.ID, &value.Reference, &provider, &consent); err != nil {
			return nil, err
		}
		if provider != nil {
			value.ProviderCode = *provider
		}
		if consent != nil {
			value.ConsentReference = *consent
		}
		result = append(result, value)
	}
	return result, rows.Err()
}

func queryPartyRevisions(ctx context.Context, queryer pgxQueryer, id uuid.UUID) ([]PartyRevision, error) {
	rows, err := queryer.Query(ctx, `SELECT revision_number, aggregate_version, snapshot, created_at FROM organization.party_revision WHERE party_id=$1 ORDER BY revision_number`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]PartyRevision, 0)
	for rows.Next() {
		var value PartyRevision
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

func mapPartyPostgresError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrPartyDuplicate
		case "23514", "23503":
			return ErrInvalidParty
		}
	}
	if strings.Contains(strings.ToLower(err.Error()), "party") && strings.Contains(strings.ToLower(err.Error()), "constraint") {
		return ErrInvalidParty
	}
	return err
}

var _ PartyRepository = (*PostgresPartyRepository)(nil)
var _ DurablePartyRepository = (*PostgresPartyRepository)(nil)
