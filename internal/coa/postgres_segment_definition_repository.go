package coa

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

type PostgresSegmentDefinitionAuditWriter func(context.Context, pgx.Tx, AuditRecord) (uuid.UUID, error)

type PostgresSegmentDefinitionRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresSegmentDefinitionAuditWriter
}

func NewPostgresSegmentDefinitionRepository(pool *pgxpool.Pool, auditWriter PostgresSegmentDefinitionAuditWriter) (*PostgresSegmentDefinitionRepository, error) {
	if pool == nil {
		return nil, ErrInvalidSegmentDefinitionService
	}
	return &PostgresSegmentDefinitionRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresSegmentDefinitionRepository) Get(ctx context.Context, id uuid.UUID) (SegmentDefinition, error) {
	if repository == nil || repository.pool == nil {
		return SegmentDefinition{}, ErrInvalidSegmentDefinitionService
	}
	return readSegmentDefinition(ctx, repository.pool, id)
}

func (repository *PostgresSegmentDefinitionRepository) List(ctx context.Context, scopeID *uuid.UUID) ([]SegmentDefinition, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidSegmentDefinitionService
	}
	rows, err := repository.pool.Query(ctx, `
		SELECT segment_definition_id
		FROM coa.segment_definition
		WHERE ($1::uuid IS NULL OR scope_id = $1)
		ORDER BY segment_definition_id`, nullableUUID(scopeID))
	if err != nil {
		return nil, mapSegmentDefinitionPostgresError(err)
	}
	defer rows.Close()
	result := make([]SegmentDefinition, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, mapSegmentDefinitionPostgresError(err)
		}
		definition, err := readSegmentDefinition(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, definition)
	}
	if err := rows.Err(); err != nil {
		return nil, mapSegmentDefinitionPostgresError(err)
	}
	sortSegmentDefinitions(result)
	return result, nil
}

func (repository *PostgresSegmentDefinitionRepository) CommitSegmentDefinitionMutation(ctx context.Context, mutation SegmentDefinitionMutation) error {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresSegmentDefinitionRepository) CommitSegmentDefinitionMutationWithIdempotency(ctx context.Context, mutation SegmentDefinitionMutation, commit DurableSegmentDefinitionMutationCommit) error {
	return repository.commit(ctx, mutation, &commit)
}

func (repository *PostgresSegmentDefinitionRepository) commit(ctx context.Context, mutation SegmentDefinitionMutation, durable *DurableSegmentDefinitionMutationCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidSegmentDefinitionService
	}
	if repository.auditWriter == nil {
		return ErrSegmentDefinitionAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 {
			return ErrSegmentDefinitionVersionConflict
		}
	} else if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrSegmentDefinitionVersionConflict
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

	// Scope-level advisory serialization makes the application-level inclusive
	// date-range uniqueness rule safe when two administrators submit commands at
	// the same time. The database remains the sole owner of the check.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, mutation.After.ScopeID.String()); err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}

	if err := repository.ensureNoOverlap(ctx, tx, mutation.After); err != nil {
		return err
	}

	if mutation.Before.ID == uuid.Nil {
		_, err = tx.Exec(ctx, `
			INSERT INTO coa.segment_definition
			(segment_definition_id, scope_id, segment_type, code, name, status,
			 effective_from, effective_to, aggregate_version, revision_number,
			 created_at, updated_at)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
			mutation.After.ID, mutation.After.ScopeID, mutation.After.SegmentType, mutation.After.Code, mutation.After.Name, mutation.After.Status,
			mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.Version.Value(), mutation.After.RevisionNumber,
			mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `
			UPDATE coa.segment_definition
			SET segment_type=$1, code=$2, name=$3, status=$4, effective_from=$5,
			    effective_to=$6, aggregate_version=$7, revision_number=$8, updated_at=$9
			WHERE segment_definition_id=$10 AND aggregate_version=$11`,
			mutation.After.SegmentType, mutation.After.Code, mutation.After.Name, mutation.After.Status,
			mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.Version.Value(), mutation.After.RevisionNumber,
			mutation.After.UpdatedAt, mutation.After.ID, mutation.ExpectedVersion.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrSegmentDefinitionVersionConflict
		}
	}
	if err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}

	snapshot, err := json.Marshal(mutation.After.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO coa.segment_definition_revision
		(segment_definition_id, revision_number, aggregate_version, snapshot,
		 effective_from, effective_to, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		mutation.After.ID, mutation.After.RevisionNumber, mutation.After.Version.Value(), snapshot,
		mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo, mutation.After.UpdatedAt)
	if err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}

	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrSegmentDefinitionAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE coa.segment_definition SET last_audit_reference=$1 WHERE segment_definition_id=$2`, auditReference, mutation.After.ID); err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}

	if durable != nil {
		if durable.Coordinator == nil {
			return ErrInvalidSegmentDefinitionService
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

func (repository *PostgresSegmentDefinitionRepository) ensureNoOverlap(ctx context.Context, tx pgx.Tx, candidate SegmentDefinition) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT segment_definition_id
		FROM coa.segment_definition
		WHERE scope_id=$1
		  AND segment_type=$2
		  AND code=$3
		  AND segment_definition_id <> $4
		  AND effective_from <= COALESCE($5::date, DATE '9999-12-31')
		  AND COALESCE(effective_to, DATE '9999-12-31') >= $6::date
		LIMIT 1
		FOR UPDATE`, candidate.ScopeID, candidate.SegmentType, candidate.Code, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}
	return ErrSegmentDefinitionDuplicate
}

type coaQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func readSegmentDefinition(ctx context.Context, queryer coaQueryer, id uuid.UUID) (SegmentDefinition, error) {
	var definition SegmentDefinition
	var effectiveTo *time.Time
	var lastAuditReference *uuid.UUID
	var version, revision int64
	err := queryer.QueryRow(ctx, `
		SELECT segment_definition_id, scope_id, segment_type, code, name, status,
		       effective_from, effective_to, aggregate_version, revision_number,
		       created_at, updated_at, last_audit_reference
		FROM coa.segment_definition
		WHERE segment_definition_id=$1`, id).Scan(
		&definition.ID, &definition.ScopeID, &definition.SegmentType, &definition.Code, &definition.Name, &definition.Status,
		&definition.EffectiveDateFrom, &effectiveTo, &version, &revision, &definition.CreatedAt, &definition.UpdatedAt, &lastAuditReference)
	if errors.Is(err, pgx.ErrNoRows) {
		return SegmentDefinition{}, ErrSegmentDefinitionNotFound
	}
	if err != nil {
		return SegmentDefinition{}, mapSegmentDefinitionPostgresError(err)
	}
	parsedVersion, err := aggregateversion.FromInt64(version)
	if err != nil {
		return SegmentDefinition{}, err
	}
	definition.Version, definition.RevisionNumber = parsedVersion, revision
	definition.EffectiveDateFrom = dateOnly(definition.EffectiveDateFrom)
	definition.EffectiveDateTo = cloneDate(effectiveTo)
	revisions, err := querySegmentDefinitionRevisions(ctx, queryer, id)
	if err != nil {
		return SegmentDefinition{}, err
	}
	definition.Revisions = revisions
	if err := definition.Validate(); err != nil {
		return SegmentDefinition{}, err
	}
	return definition, nil
}

func querySegmentDefinitionRevisions(ctx context.Context, queryer coaQueryer, id uuid.UUID) ([]SegmentDefinitionRevision, error) {
	rows, err := queryer.Query(ctx, `
		SELECT revision_number, aggregate_version, snapshot, created_at
		FROM coa.segment_definition_revision
		WHERE segment_definition_id=$1
		ORDER BY revision_number`, id)
	if err != nil {
		return nil, mapSegmentDefinitionPostgresError(err)
	}
	defer rows.Close()
	result := make([]SegmentDefinitionRevision, 0)
	for rows.Next() {
		var revision SegmentDefinitionRevision
		var version int64
		var snapshot []byte
		if err := rows.Scan(&revision.RevisionNumber, &version, &snapshot, &revision.CreatedAt); err != nil {
			return nil, mapSegmentDefinitionPostgresError(err)
		}
		parsedVersion, err := aggregateversion.FromInt64(version)
		if err != nil {
			return nil, err
		}
		revision.Version = parsedVersion
		if err := json.Unmarshal(snapshot, &revision.Snapshot); err != nil {
			return nil, err
		}
		revision.Snapshot.EffectiveDateFrom = dateOnly(revision.Snapshot.EffectiveDateFrom)
		revision.Snapshot.EffectiveDateTo = cloneDate(revision.Snapshot.EffectiveDateTo)
		result = append(result, revision)
	}
	if err := rows.Err(); err != nil {
		return nil, mapSegmentDefinitionPostgresError(err)
	}
	return result, nil
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func mapSegmentDefinitionPostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrSegmentDefinitionDuplicate
		case "23514":
			return ErrInvalidSegmentDefinition
		}
	}
	return err
}

var _ SegmentDefinitionRepository = (*PostgresSegmentDefinitionRepository)(nil)
var _ DurableSegmentDefinitionRepository = (*PostgresSegmentDefinitionRepository)(nil)
