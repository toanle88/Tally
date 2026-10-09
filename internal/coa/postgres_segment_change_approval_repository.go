package coa

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

const segmentChangeRequestSelect = `
	SELECT segment_change_request_id, scope_id, change_type, subject_id, subject_version,
	       requested_effective_date, approval_request_id, approval_decision_id,
	       approval_policy_version, approval_decision_version, approval_subject_version,
	       approval_candidate_fingerprint, approval_approver_user_id, approval_decided_at,
	       approval_applied_at, applied_subject_version, resulting_subject_version,
	       decision_fingerprint, approval_status, application_status,
	       validation_outcome, conflict_code, rejection_reason, next_action, proposed_change,
	       subject_fingerprint, proposed_fingerprint, aggregate_version, revision_number,
	       created_by, created_at, updated_at, last_audit_reference
	FROM coa.segment_change_request
`

func (repository *PostgresSegmentChangeRequestRepository) Get(ctx context.Context, id uuid.UUID) (SegmentChangeRequest, error) {
	if repository == nil || repository.pool == nil {
		return SegmentChangeRequest{}, ErrInvalidSegmentChangeRequestService
	}
	return readSegmentChangeRequest(ctx, repository.pool, id)
}

func (repository *PostgresSegmentChangeRequestRepository) GetSegmentChangeSubject(ctx context.Context, changeType string, subjectID uuid.UUID) (SegmentChangeRequestSubject, error) {
	if repository == nil || repository.pool == nil {
		return SegmentChangeRequestSubject{}, ErrInvalidSegmentChangeRequestService
	}
	if changeType != SegmentChangeTypeDefinition && changeType != SegmentChangeTypeValue {
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestUnsupportedSubject
	}
	definitionID := subjectID
	if changeType == SegmentChangeTypeValue {
		if err := repository.pool.QueryRow(ctx, `SELECT segment_definition_id FROM coa.segment_value WHERE segment_value_id=$1`, subjectID).Scan(&definitionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
			}
			return SegmentChangeRequestSubject{}, mapSegmentValuePostgresError(err)
		}
	}
	definition, err := readSegmentDefinition(ctx, repository.pool, definitionID)
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

func readSegmentChangeRequest(ctx context.Context, queryer coaQueryer, id uuid.UUID) (SegmentChangeRequest, error) {
	var request SegmentChangeRequest
	var (
		subjectVersion, aggregateVersion, revisionNumber                         int64
		approvalDecisionVersion                                                  *int64
		approvalSubjectVersion, appliedSubjectVersion, resultingSubjectVersion   *int64
		approvalDecisionID, approvalApproverUserID, lastAuditReference           *uuid.UUID
		approvalPolicyVersion, approvalCandidateFingerprint, decisionFingerprint *string
		approvalDecidedAt, approvalAppliedAt                                     *time.Time
		proposal                                                                 []byte
	)
	err := queryer.QueryRow(ctx, segmentChangeRequestSelect+` WHERE segment_change_request_id=$1`, id).Scan(
		&request.ID, &request.ScopeID, &request.ChangeType, &request.SubjectID, &subjectVersion,
		&request.RequestedEffectiveDate, &request.ApprovalRequestID, &approvalDecisionID,
		&approvalPolicyVersion, &approvalDecisionVersion, &approvalSubjectVersion,
		&approvalCandidateFingerprint, &approvalApproverUserID, &approvalDecidedAt,
		&approvalAppliedAt, &appliedSubjectVersion, &resultingSubjectVersion,
		&decisionFingerprint, &request.ApprovalStatus, &request.ApplicationStatus,
		&request.ValidationOutcome, &request.ConflictCode, &request.RejectionReason, &request.NextAction, &proposal,
		&request.SubjectFingerprint, &request.ProposedFingerprint, &aggregateVersion, &revisionNumber,
		&request.CreatedBy, &request.CreatedAt, &request.UpdatedAt, &lastAuditReference,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return SegmentChangeRequest{}, ErrSegmentChangeRequestSubjectNotFound
	}
	if err != nil {
		return SegmentChangeRequest{}, mapSegmentChangeRequestPostgresError(err)
	}
	if err := json.Unmarshal(proposal, &request.ProposedChange); err != nil {
		return SegmentChangeRequest{}, err
	}
	request.SubjectVersion, err = aggregateversionFromInt64(subjectVersion)
	if err != nil {
		return SegmentChangeRequest{}, err
	}
	request.Version, err = aggregateversionFromInt64(aggregateVersion)
	if err != nil {
		return SegmentChangeRequest{}, err
	}
	request.RevisionNumber = revisionNumber
	if approvalDecisionID != nil {
		request.ApprovalDecisionID = *approvalDecisionID
	}
	if approvalPolicyVersion != nil {
		request.ApprovalPolicyVersion = *approvalPolicyVersion
	}
	if approvalDecisionVersion != nil {
		request.ApprovalDecisionVersion = *approvalDecisionVersion
	}
	if approvalSubjectVersion != nil {
		request.ApprovalSubjectVersion, err = aggregateversionFromInt64(*approvalSubjectVersion)
		if err != nil {
			return SegmentChangeRequest{}, err
		}
	}
	if approvalCandidateFingerprint != nil {
		request.ApprovalCandidateFingerprint = *approvalCandidateFingerprint
	}
	if approvalApproverUserID != nil {
		request.ApprovalApproverUserID = *approvalApproverUserID
	}
	if approvalDecidedAt != nil {
		request.ApprovalDecidedAt = approvalDecidedAt.UTC()
	}
	if approvalAppliedAt != nil {
		request.ApprovalAppliedAt = approvalAppliedAt.UTC()
	}
	if appliedSubjectVersion != nil {
		request.AppliedSubjectVersion, err = aggregateversionFromInt64(*appliedSubjectVersion)
		if err != nil {
			return SegmentChangeRequest{}, err
		}
	}
	if resultingSubjectVersion != nil {
		request.ResultingSubjectVersion, err = aggregateversionFromInt64(*resultingSubjectVersion)
		if err != nil {
			return SegmentChangeRequest{}, err
		}
	}
	if decisionFingerprint != nil {
		request.DecisionFingerprint = *decisionFingerprint
	}
	if lastAuditReference != nil {
		// The COA domain currently exposes audit identity through the audit port;
		// retain the database value only for persistence round-tripping.
		_ = lastAuditReference
	}
	request.RequestedEffectiveDate = dateOnly(request.RequestedEffectiveDate)
	request.CreatedAt = request.CreatedAt.UTC()
	request.UpdatedAt = request.UpdatedAt.UTC()
	return request, nil
}

func aggregateversionFromInt64(value int64) (aggregateversion.AggregateVersion, error) {
	return aggregateversion.FromInt64(value)
}

func (repository *PostgresSegmentChangeRequestRepository) CommitSegmentChangeApproval(ctx context.Context, mutation SegmentChangeApprovalMutation) error {
	return repository.commitSegmentChangeApproval(ctx, mutation, nil)
}

func (repository *PostgresSegmentChangeRequestRepository) CommitSegmentChangeApprovalWithIdempotency(ctx context.Context, mutation SegmentChangeApprovalMutation, durable DurableSegmentChangeApprovalCommit) error {
	return repository.commitSegmentChangeApproval(ctx, mutation, &durable)
}

func (repository *PostgresSegmentChangeRequestRepository) commitSegmentChangeApproval(ctx context.Context, mutation SegmentChangeApprovalMutation, durable *DurableSegmentChangeApprovalCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidSegmentChangeRequestService
	}
	if repository.auditWriter == nil {
		return ErrSegmentChangeApprovalDecisionAuditUnavailable
	}
	if !mutation.ReplayOnly {
		if err := mutation.Request.Validate(); err != nil {
			return err
		}
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

	stored, err := readSegmentChangeRequest(ctx, tx, mutation.Request.ID)
	if err != nil {
		return err
	}
	if mutation.ReplayOnly {
		if stored.DecisionFingerprint == "" || stored.DecisionFingerprint != mutation.Request.DecisionFingerprint {
			return ErrSegmentChangeApprovalDecisionDuplicate
		}
		if err := finalizeSegmentChangeApprovalIdempotency(ctx, tx, durable); err != nil {
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			return err
		}
		committed = true
		return nil
	}
	if stored.ApprovalStatus != SegmentChangeRequestApprovalPending || stored.ApplicationStatus != SegmentChangeRequestApplicationOpen {
		return ErrSegmentChangeApprovalDecisionDuplicate
	}
	if stored.ScopeID != mutation.Request.ScopeID || stored.ID != mutation.Request.ID {
		return ErrSegmentChangeApprovalDecisionAuthorizationDenied
	}
	// Serialize inclusive effective-date uniqueness checks with the existing
	// segment-definition mutation path for the accounting scope.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1, 0))`, stored.ScopeID.String()); err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}

	current, err := lockAndReadSegmentChangeSubject(ctx, tx, stored.ChangeType, stored.SubjectID)
	if err != nil {
		return err
	}
	if mutation.Request.ApplicationStatus != SegmentChangeRequestApplicationConflict {
		if current.ScopeID != mutation.Request.ScopeID || current.Version.Value() != mutation.BeforeSubject.Version.Value() || FingerprintSegmentChangeSubject(current) != FingerprintSegmentChangeSubject(mutation.BeforeSubject) {
			return ErrSegmentChangeApprovalDecisionSubjectConflict
		}
	}
	if mutation.Request.ApplicationStatus == SegmentChangeRequestApplicationApplied {
		if err := applyPostgresSegmentChangeSubject(ctx, tx, mutation); err != nil {
			return err
		}
	}
	proposal, err := json.Marshal(mutation.Request.ProposedChange.Canonical(mutation.Request.ChangeType))
	if err != nil {
		return err
	}
	var tag pgconn.CommandTag
	tag, err = tx.Exec(ctx, `
		UPDATE coa.segment_change_request
		SET approval_decision_id=$1, approval_policy_version=$2, approval_decision_version=$3,
		    approval_subject_version=$4, approval_candidate_fingerprint=$5, approval_approver_user_id=$6,
		    approval_decided_at=$7, approval_applied_at=$8, applied_subject_version=$9,
		    resulting_subject_version=$10, decision_fingerprint=$11, approval_status=$12,
		    application_status=$13, validation_outcome=$14, conflict_code=$15, rejection_reason=$16,
		    next_action=$17, proposed_change=$18, aggregate_version=$19, revision_number=$20, updated_at=$21
		WHERE segment_change_request_id=$22 AND approval_status='pending' AND application_status='not-applied'`,
		mutation.Request.ApprovalDecisionID, nullableText(mutation.Request.ApprovalPolicyVersion), mutation.Request.ApprovalDecisionVersion,
		mutation.Request.ApprovalSubjectVersion.Value(), mutation.Request.ApprovalCandidateFingerprint, mutation.Request.ApprovalApproverUserID,
		mutation.Request.ApprovalDecidedAt, mutation.Request.ApprovalAppliedAt, mutation.Request.AppliedSubjectVersion.Value(),
		mutation.Request.ResultingSubjectVersion.Value(), mutation.Request.DecisionFingerprint, mutation.Request.ApprovalStatus,
		mutation.Request.ApplicationStatus, mutation.Request.ValidationOutcome, nullableText(mutation.Request.ConflictCode), nullableText(mutation.Request.RejectionReason),
		mutation.Request.NextAction, proposal, mutation.Request.Version.Value(), mutation.Request.RevisionNumber, mutation.Request.UpdatedAt, mutation.Request.ID)
	if err != nil {
		return mapSegmentChangeRequestPostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSegmentChangeApprovalDecisionDuplicate
	}
	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrSegmentChangeApprovalDecisionAuditUnavailable
	}
	if _, err := tx.Exec(ctx, `UPDATE coa.segment_change_request SET last_audit_reference=$1 WHERE segment_change_request_id=$2`, auditReference, mutation.Request.ID); err != nil {
		return mapSegmentChangeRequestPostgresError(err)
	}
	if mutation.Request.ApplicationStatus == SegmentChangeRequestApplicationApplied {
		if _, err := tx.Exec(ctx, `UPDATE coa.segment_definition SET last_audit_reference=$1 WHERE segment_definition_id=$2`, auditReference, mutation.AfterSubject.Definition.ID); err != nil {
			return mapSegmentDefinitionPostgresError(err)
		}
	}
	if err := finalizeSegmentChangeApprovalIdempotency(ctx, tx, durable); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func finalizeSegmentChangeApprovalIdempotency(ctx context.Context, tx pgx.Tx, durable *DurableSegmentChangeApprovalCommit) error {
	if durable == nil {
		return nil
	}
	if durable.Coordinator == nil {
		return ErrInvalidSegmentChangeRequestService
	}
	return durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, durable.Result)
}

func applyPostgresSegmentChangeSubject(ctx context.Context, tx pgx.Tx, mutation SegmentChangeApprovalMutation) error {
	if mutation.AfterSubject.Definition == nil {
		return ErrSegmentChangeApprovalDecisionSubjectConflict
	}
	after := *mutation.AfterSubject.Definition
	if err := after.Validate(); err != nil {
		return err
	}
	if mutation.Request.ChangeType == SegmentChangeTypeDefinition {
		if err := ensurePostgresDefinitionNoOverlap(ctx, tx, after); err != nil {
			return err
		}
		var tag pgconn.CommandTag
		var err error
		tag, err = tx.Exec(ctx, `
			UPDATE coa.segment_definition
			SET segment_type=$1, code=$2, name=$3, status=$4, effective_from=$5, effective_to=$6,
			    aggregate_version=$7, revision_number=$8, updated_at=$9
			WHERE segment_definition_id=$10 AND aggregate_version=$11`,
			after.SegmentType, after.Code, after.Name, after.Status, after.EffectiveDateFrom, after.EffectiveDateTo,
			after.Version.Value(), after.RevisionNumber, after.UpdatedAt, after.ID, mutation.BeforeSubject.Version.Value())
		if err != nil {
			return mapSegmentDefinitionPostgresError(err)
		}
		if tag.RowsAffected() != 1 {
			return ErrSegmentChangeApprovalDecisionSubjectConflict
		}
		return insertPostgresDefinitionRevision(ctx, tx, after)
	}
	if mutation.Request.ChangeType != SegmentChangeTypeValue || mutation.AfterSubject.Value == nil {
		return ErrSegmentChangeRequestUnsupportedSubject
	}
	if err := ensurePostgresValueNoOverlap(ctx, tx, after.ID, *mutation.AfterSubject.Value); err != nil {
		return err
	}
	value := mutation.AfterSubject.Value
	var tag pgconn.CommandTag
	var err error
	tag, err = tx.Exec(ctx, `
		UPDATE coa.segment_value
		SET value=$1, description=$2, status=$3, effective_from=$4, effective_to=$5, updated_at=$6
		WHERE segment_value_id=$7 AND segment_definition_id=$8`,
		value.Value, value.Description, value.Status, value.EffectiveDateFrom, value.EffectiveDateTo, value.UpdatedAt, value.ID, after.ID)
	if err != nil {
		return mapSegmentValuePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSegmentValueNotFound
	}
	tag, err = tx.Exec(ctx, `
		UPDATE coa.segment_definition
		SET aggregate_version=$1, revision_number=$2, updated_at=$3
		WHERE segment_definition_id=$4 AND aggregate_version=$5`,
		after.Version.Value(), after.RevisionNumber, after.UpdatedAt, after.ID, mutation.BeforeSubject.Version.Value())
	if err != nil {
		return mapSegmentValuePostgresError(err)
	}
	if tag.RowsAffected() != 1 {
		return ErrSegmentChangeApprovalDecisionSubjectConflict
	}
	return insertPostgresDefinitionRevision(ctx, tx, after)
}

func insertPostgresDefinitionRevision(ctx context.Context, tx pgx.Tx, definition SegmentDefinition) error {
	snapshot, err := json.Marshal(definition.Snapshot())
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO coa.segment_definition_revision
		(segment_definition_id, revision_number, aggregate_version, snapshot, effective_from, effective_to, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`, definition.ID, definition.RevisionNumber, definition.Version.Value(), snapshot,
		definition.EffectiveDateFrom, definition.EffectiveDateTo, definition.UpdatedAt)
	return mapSegmentDefinitionPostgresError(err)
}

func ensurePostgresDefinitionNoOverlap(ctx context.Context, tx pgx.Tx, candidate SegmentDefinition) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT segment_definition_id
		FROM coa.segment_definition
		WHERE scope_id=$1 AND segment_type=$2 AND code=$3 AND segment_definition_id<>$4
		  AND effective_from <= COALESCE($5::date, DATE '9999-12-31')
		  AND COALESCE(effective_to, DATE '9999-12-31') >= $6::date
		LIMIT 1 FOR UPDATE`, candidate.ScopeID, candidate.SegmentType, candidate.Code, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapSegmentDefinitionPostgresError(err)
	}
	return ErrSegmentDefinitionDuplicate
}

func ensurePostgresValueNoOverlap(ctx context.Context, tx pgx.Tx, definitionID uuid.UUID, candidate SegmentValue) error {
	var existingID uuid.UUID
	err := tx.QueryRow(ctx, `
		SELECT segment_value_id
		FROM coa.segment_value
		WHERE segment_definition_id=$1 AND value=$2 AND segment_value_id<>$3
		  AND effective_from <= COALESCE($4::date, DATE '9999-12-31')
		  AND COALESCE(effective_to, DATE '9999-12-31') >= $5::date
		LIMIT 1 FOR UPDATE`, definitionID, candidate.Value, candidate.ID, candidate.EffectiveDateTo, candidate.EffectiveDateFrom).Scan(&existingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return mapSegmentValuePostgresError(err)
	}
	return ErrSegmentValueDuplicate
}

var _ DurableSegmentChangeApprovalRepository = (*PostgresSegmentChangeRequestRepository)(nil)

var _ SegmentChangeRequestRepository = (*PostgresSegmentChangeRequestRepository)(nil)
