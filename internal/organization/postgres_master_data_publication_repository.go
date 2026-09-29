package organization

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

type PostgresMasterDataPublicationAuditWriter func(context.Context, pgx.Tx, MasterDataPublicationAuditRecord) (uuid.UUID, error)

type PostgresMasterDataPublicationRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresMasterDataPublicationAuditWriter
}

func NewPostgresMasterDataPublicationRepository(pool *pgxpool.Pool, auditWriter PostgresMasterDataPublicationAuditWriter) (*PostgresMasterDataPublicationRepository, error) {
	if pool == nil {
		return nil, ErrInvalidMasterDataPublicationService
	}
	return &PostgresMasterDataPublicationRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresMasterDataPublicationRepository) Load(ctx context.Context, command MasterDataPublicationCommand) (MasterDataPublicationCandidate, error) {
	if repository == nil || repository.pool == nil {
		return MasterDataPublicationCandidate{}, ErrInvalidMasterDataPublicationService
	}
	return loadMasterDataPublicationCandidate(ctx, repository.pool, command)
}

func (repository *PostgresMasterDataPublicationRepository) Commit(ctx context.Context, command MasterDataPublicationCommand, expected MasterDataPublicationCandidate, event platformevents.Envelope, result MasterDataPublicationResult, audit MasterDataPublicationAuditRecord, durable *DurableMasterDataPublicationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidMasterDataPublicationService
	}
	if repository.auditWriter == nil {
		return ErrMasterDataPublicationAuditUnavailable
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	if err := lockMasterDataAggregate(ctx, tx, command); err != nil {
		return err
	}
	current, err := loadMasterDataPublicationCandidate(ctx, tx, command)
	if err != nil {
		return err
	}
	if current.ScopeID != command.ScopeID || !current.Version.Matches(*command.ExpectedVersion) || current.Version.Value() != expected.Version.Value() || current.Fingerprint != expected.Fingerprint {
		return ErrMasterDataPublicationSourceChanged
	}
	var existingPublication uuid.UUID
	err = tx.QueryRow(ctx, `SELECT publication_id FROM organization.master_data_publication WHERE aggregate_type=$1 AND aggregate_id=$2 AND aggregate_version=$3 FOR UPDATE`, string(command.AggregateType), command.AggregateID, current.Version.Value()).Scan(&existingPublication)
	if err == nil {
		return ErrMasterDataPublicationAlreadyPublished
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	approval, err := json.Marshal(current.Approval)
	if err != nil {
		return err
	}
	if current.Approval == nil {
		approval = nil
	}
	auditReference, err := repository.auditWriter(ctx, tx, audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrMasterDataPublicationAuditUnavailable
	}
	_, err = tx.Exec(ctx, `INSERT INTO organization.master_data_publication (publication_id, aggregate_type, aggregate_id, scope_id, aggregate_version, revision_number, status, dependent_availability, event_type, event_version, message_id, effective_from, effective_to, approval_reference, source_fingerprint, audit_reference, published_at, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$17,$17)`, result.PublicationID, string(result.AggregateType), result.AggregateID, result.ScopeID, result.AggregateVersion.Value(), result.RevisionNumber, result.Status, result.DependentAvailability, event.EventType(), event.EventVersion(), result.MessageID, result.EffectiveFrom, result.EffectiveTo, approval, current.Fingerprint, auditReference, event.OccurredAt())
	if err != nil {
		return err
	}
	if err := platformintegration.WritePublication(ctx, tx, platformintegration.Publication{Event: event, AvailableAt: event.OccurredAt()}); err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidMasterDataPublicationService
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

func lockMasterDataAggregate(ctx context.Context, tx pgx.Tx, command MasterDataPublicationCommand) error {
	var id uuid.UUID
	var err error
	switch command.AggregateType {
	case MasterDataAggregateLegalEntity:
		err = tx.QueryRow(ctx, `SELECT id FROM organization.legal_entity WHERE id=$1 FOR UPDATE`, command.AggregateID).Scan(&id)
	case MasterDataAggregateParty:
		err = tx.QueryRow(ctx, `SELECT id FROM organization.party WHERE id=$1 FOR UPDATE`, command.AggregateID).Scan(&id)
	case MasterDataAggregateCustomerProfile:
		err = tx.QueryRow(ctx, `SELECT id FROM organization.customer_profile WHERE id=$1 FOR UPDATE`, command.AggregateID).Scan(&id)
	case MasterDataAggregateVendorProfile:
		err = tx.QueryRow(ctx, `SELECT id FROM organization.vendor_profile WHERE id=$1 FOR UPDATE`, command.AggregateID).Scan(&id)
	case MasterDataAggregateFiscalCalendar:
		err = tx.QueryRow(ctx, `SELECT id FROM organization.fiscal_calendar WHERE id=$1 FOR UPDATE`, command.AggregateID).Scan(&id)
	default:
		return ErrInvalidMasterDataPublicationCommand
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrMasterDataPublicationNotFound
	}
	return err
}

func loadMasterDataPublicationCandidate(ctx context.Context, queryer pgxQueryer, command MasterDataPublicationCommand) (MasterDataPublicationCandidate, error) {
	var candidate MasterDataPublicationCandidate
	switch command.AggregateType {
	case MasterDataAggregateLegalEntity:
		entity, err := readLegalEntity(ctx, queryer, command.AggregateID)
		if err != nil {
			if errors.Is(err, ErrLegalEntityNotFound) {
				return candidate, ErrMasterDataPublicationNotFound
			}
			return candidate, err
		}
		payload, marshalErr := json.Marshal(entity.SafeProjection())
		if marshalErr != nil {
			return candidate, marshalErr
		}
		candidate = MasterDataPublicationCandidate{AggregateType: MasterDataAggregateLegalEntity, AggregateID: entity.ID, ScopeID: entity.ScopeID, Status: entity.Status, EffectiveFrom: &entity.EffectiveFrom, EffectiveTo: cloneTime(entity.EffectiveTo), Version: entity.Version, RevisionNumber: entity.RevisionNumber, Approval: cloneApproval(entity.Approval), Fingerprint: Fingerprint(entity), Payload: payload}
	case MasterDataAggregateParty:
		party, err := readParty(ctx, queryer, command.AggregateID)
		if err != nil {
			if errors.Is(err, ErrPartyNotFound) {
				return candidate, ErrMasterDataPublicationNotFound
			}
			return candidate, err
		}
		payload, marshalErr := json.Marshal(party.SafeProjection())
		if marshalErr != nil {
			return candidate, marshalErr
		}
		candidate = MasterDataPublicationCandidate{AggregateType: MasterDataAggregateParty, AggregateID: party.ID, ScopeID: party.ScopeID, Status: string(party.Status), Version: party.Version, RevisionNumber: party.RevisionNumber, Approval: nil, Fingerprint: FingerprintParty(party), Payload: payload}
	case MasterDataAggregateCustomerProfile:
		profile, err := readCustomerProfile(ctx, queryer, command.AggregateID)
		if err != nil {
			if errors.Is(err, ErrCustomerProfileNotFound) {
				return candidate, ErrMasterDataPublicationNotFound
			}
			return candidate, err
		}
		payload, marshalErr := json.Marshal(profile.SafeProjection())
		if marshalErr != nil {
			return candidate, marshalErr
		}
		candidate = MasterDataPublicationCandidate{AggregateType: MasterDataAggregateCustomerProfile, AggregateID: profile.ID, ScopeID: profile.ScopeID, Status: profile.Status, EffectiveFrom: &profile.EffectiveFrom, EffectiveTo: cloneTime(profile.EffectiveTo), Version: profile.Version, RevisionNumber: profile.RevisionNumber, Approval: cloneApproval(profile.Approval), Fingerprint: FingerprintCustomerProfile(profile), Payload: payload}
	case MasterDataAggregateVendorProfile:
		profile, err := readVendorProfile(ctx, queryer, command.AggregateID)
		if err != nil {
			if errors.Is(err, ErrVendorProfileNotFound) {
				return candidate, ErrMasterDataPublicationNotFound
			}
			return candidate, err
		}
		payload, marshalErr := json.Marshal(profile.SafeProjection())
		if marshalErr != nil {
			return candidate, marshalErr
		}
		candidate = MasterDataPublicationCandidate{AggregateType: MasterDataAggregateVendorProfile, AggregateID: profile.ID, ScopeID: profile.ScopeID, Status: profile.Status, EffectiveFrom: &profile.EffectiveFrom, EffectiveTo: cloneTime(profile.EffectiveTo), Version: profile.Version, RevisionNumber: profile.RevisionNumber, Approval: cloneApproval(profile.Approval), Fingerprint: FingerprintVendorProfile(profile), Payload: payload}
	case MasterDataAggregateFiscalCalendar:
		calendar, err := readFiscalCalendar(ctx, queryer, command.AggregateID)
		if err != nil {
			if errors.Is(err, ErrFiscalCalendarNotFound) {
				return candidate, ErrMasterDataPublicationNotFound
			}
			return candidate, err
		}
		payload, marshalErr := json.Marshal(calendar.SafeProjection())
		if marshalErr != nil {
			return candidate, marshalErr
		}
		candidate = MasterDataPublicationCandidate{AggregateType: MasterDataAggregateFiscalCalendar, AggregateID: calendar.ID, ScopeID: calendar.ScopeID, Status: calendar.Status, EffectiveFrom: &calendar.EffectiveFrom, EffectiveTo: cloneTime(calendar.EffectiveTo), Version: calendar.Version, RevisionNumber: calendar.RevisionNumber, Approval: cloneApproval(calendar.Approval), Fingerprint: FingerprintFiscalCalendar(calendar), Payload: payload}
	default:
		return candidate, ErrInvalidMasterDataPublicationCommand
	}
	return candidate, nil
}
