package coa

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresSegmentChangeRequestRepository owns request persistence in the COA
// schema. The subject is locked and re-read in the same transaction so the
// captured source version cannot become stale between validation and insert.
type PostgresSegmentChangeRequestRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresSegmentDefinitionAuditWriter
}

func NewPostgresSegmentChangeRequestRepository(pool *pgxpool.Pool, auditWriter PostgresSegmentDefinitionAuditWriter) (*PostgresSegmentChangeRequestRepository, error) {
	if pool == nil {
		return nil, ErrInvalidSegmentChangeRequestService
	}
	return &PostgresSegmentChangeRequestRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresSegmentChangeRequestRepository) CommitSegmentChangeRequest(ctx context.Context, mutation SegmentChangeRequestMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresSegmentChangeRequestRepository) CommitSegmentChangeRequestWithIdempotency(ctx context.Context, mutation SegmentChangeRequestMutation, durable DurableSegmentChangeRequestCommit) error {
	return repository.commit(ctx, mutation, &durable)
}

func (repository *PostgresSegmentChangeRequestRepository) commit(ctx context.Context, mutation SegmentChangeRequestMutation, durable *DurableSegmentChangeRequestCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidSegmentChangeRequestService
	}
	if repository.auditWriter == nil {
		return ErrSegmentChangeRequestAuditUnavailable
	}
	if err := mutation.Request.Validate(); err != nil {
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

	current, err := lockAndReadSegmentChangeSubject(ctx, tx, mutation.Request.ChangeType, mutation.Request.SubjectID)
	if err != nil {
		return err
	}
	if current.ScopeID != mutation.Request.ScopeID || current.Version.Value() != mutation.Request.SubjectVersion.Value() || FingerprintSegmentChangeSubject(current) != mutation.Request.SubjectFingerprint {
		return ErrSegmentChangeRequestVersionConflict
	}

	proposal, err := json.Marshal(mutation.Request.ProposedChange.Canonical(mutation.Request.ChangeType))
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO coa.segment_change_request
		(segment_change_request_id, scope_id, change_type, subject_id, subject_version,
		 requested_effective_date, approval_request_id, approval_status, application_status,
		 validation_outcome, conflict_code, rejection_reason, next_action, proposed_change,
		 subject_fingerprint, proposed_fingerprint, aggregate_version, revision_number,
		 created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21)`,
		mutation.Request.ID, mutation.Request.ScopeID, mutation.Request.ChangeType, mutation.Request.SubjectID, mutation.Request.SubjectVersion.Value(),
		mutation.Request.RequestedEffectiveDate, mutation.Request.ApprovalRequestID, mutation.Request.ApprovalStatus, mutation.Request.ApplicationStatus,
		mutation.Request.ValidationOutcome, nullableText(mutation.Request.ConflictCode), nullableText(mutation.Request.RejectionReason), mutation.Request.NextAction,
		proposal, mutation.Request.SubjectFingerprint, mutation.Request.ProposedFingerprint, mutation.Request.Version.Value(), mutation.Request.RevisionNumber,
		mutation.Request.CreatedBy, mutation.Request.CreatedAt, mutation.Request.UpdatedAt)
	if err != nil {
		return mapSegmentChangeRequestPostgresError(err)
	}

	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrSegmentChangeRequestAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE coa.segment_change_request SET last_audit_reference=$1 WHERE segment_change_request_id=$2`, auditReference, mutation.Request.ID); err != nil {
		return mapSegmentChangeRequestPostgresError(err)
	}

	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidSegmentChangeRequestService
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

func lockAndReadSegmentChangeSubject(ctx context.Context, tx pgx.Tx, changeType string, subjectID uuid.UUID) (SegmentChangeRequestSubject, error) {
	if changeType != SegmentChangeTypeDefinition && changeType != SegmentChangeTypeValue {
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestUnsupportedSubject
	}
	definitionID := subjectID
	if changeType == SegmentChangeTypeValue {
		if err := tx.QueryRow(ctx, `SELECT segment_definition_id FROM coa.segment_value WHERE segment_value_id=$1 FOR UPDATE`, subjectID).Scan(&definitionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
			}
			return SegmentChangeRequestSubject{}, mapSegmentChangeRequestPostgresError(err)
		}
	}
	if _, err := tx.Exec(ctx, `SELECT segment_definition_id FROM coa.segment_definition WHERE segment_definition_id=$1 FOR UPDATE`, definitionID); err != nil {
		return SegmentChangeRequestSubject{}, mapSegmentChangeRequestPostgresError(err)
	}
	definition, err := readSegmentDefinition(ctx, tx, definitionID)
	if errors.Is(err, ErrSegmentDefinitionNotFound) {
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
	}
	if err != nil {
		return SegmentChangeRequestSubject{}, err
	}
	if changeType == SegmentChangeTypeDefinition {
		return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: definition.ID, ScopeID: definition.ScopeID, Version: definition.Version, RevisionNumber: definition.RevisionNumber, Definition: &definition}, nil
	}
	for index := range definition.Values {
		if definition.Values[index].ID == subjectID {
			value := cloneSegmentValue(definition.Values[index])
			return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: subjectID, ScopeID: definition.ScopeID, Version: definition.Version, RevisionNumber: definition.RevisionNumber, Definition: &definition, Value: &value}, nil
		}
	}
	return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
}

func nullableText(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func mapSegmentChangeRequestPostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrSegmentChangeRequestDuplicate
		case "23514":
			return ErrInvalidSegmentChangeRequest
		}
	}
	return err
}

var _ SegmentChangeRequestRepository = (*PostgresSegmentChangeRequestRepository)(nil)
var _ DurableSegmentChangeRequestRepository = (*PostgresSegmentChangeRequestRepository)(nil)
