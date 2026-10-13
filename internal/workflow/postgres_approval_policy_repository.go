package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

// PostgresApprovalPolicyAuditWriter is supplied by the audit owner. Workflow
// passes the open transaction and never reaches into another module's audit
// schema or adapter.
type PostgresApprovalPolicyAuditWriter func(context.Context, pgx.Tx, ApprovalPolicyAuditRecord) (uuid.UUID, error)

type PostgresApprovalPolicyRepository struct {
	pool               *pgxpool.Pool
	auditWriter        PostgresApprovalPolicyAuditWriter
	requestAuditWriter PostgresApprovalRequestAuditWriter
	requestEventWriter PostgresApprovalEventWriter
}

func NewPostgresApprovalPolicyRepository(pool *pgxpool.Pool, auditWriter PostgresApprovalPolicyAuditWriter) (*PostgresApprovalPolicyRepository, error) {
	return newPostgresApprovalPolicyRepository(pool, auditWriter, nil, nil)
}

func NewPostgresApprovalPolicyRepositoryWithWorkflow(pool *pgxpool.Pool, auditWriter PostgresApprovalPolicyAuditWriter, requestAuditWriter PostgresApprovalRequestAuditWriter, requestEventWriter PostgresApprovalEventWriter) (*PostgresApprovalPolicyRepository, error) {
	return newPostgresApprovalPolicyRepository(pool, auditWriter, requestAuditWriter, requestEventWriter)
}

func newPostgresApprovalPolicyRepository(pool *pgxpool.Pool, auditWriter PostgresApprovalPolicyAuditWriter, requestAuditWriter PostgresApprovalRequestAuditWriter, requestEventWriter PostgresApprovalEventWriter) (*PostgresApprovalPolicyRepository, error) {
	if pool == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return &PostgresApprovalPolicyRepository{pool: pool, auditWriter: auditWriter, requestAuditWriter: requestAuditWriter, requestEventWriter: requestEventWriter}, nil
}

func (repository *PostgresApprovalPolicyRepository) Get(ctx context.Context, id uuid.UUID) (ApprovalPolicy, error) {
	if repository == nil || repository.pool == nil {
		return ApprovalPolicy{}, ErrInvalidApprovalPolicyService
	}
	return repository.readPolicy(ctx, repository.pool, id)
}

func (repository *PostgresApprovalPolicyRepository) List(ctx context.Context) ([]ApprovalPolicy, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM workflow.approval_policy ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]ApprovalPolicy, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		policy, err := repository.readPolicy(ctx, repository.pool, id)
		if err != nil {
			return nil, err
		}
		result = append(result, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sortPolicies(result), nil
}

func (repository *PostgresApprovalPolicyRepository) ListSelectable(ctx context.Context) ([]ApprovalPolicy, error) {
	policies, err := repository.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]ApprovalPolicy, 0, len(policies))
	for _, policy := range policies {
		result = append(result, selectablePolicyVersions(policy)...)
	}
	return sortPolicies(result), nil
}

func (repository *PostgresApprovalPolicyRepository) Commit(ctx context.Context, mutation ApprovalPolicyMutation) error {
	_, err := repository.CommitWithResult(ctx, mutation)
	return err
}

func (repository *PostgresApprovalPolicyRepository) CommitWithResult(ctx context.Context, mutation ApprovalPolicyMutation) (uuid.UUID, error) {
	return repository.commit(ctx, mutation, nil)
}

func (repository *PostgresApprovalPolicyRepository) CommitWithIdempotency(ctx context.Context, mutation ApprovalPolicyMutation, durable DurableApprovalPolicyMutationCommit) (uuid.UUID, error) {
	return repository.commit(ctx, mutation, &durable)
}

func (repository *PostgresApprovalPolicyRepository) commit(ctx context.Context, mutation ApprovalPolicyMutation, durable *DurableApprovalPolicyMutationCommit) (uuid.UUID, error) {
	if repository == nil || repository.pool == nil {
		return uuid.Nil, ErrInvalidApprovalPolicyService
	}
	if repository.auditWriter == nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	if err := mutation.After.Validate(); err != nil {
		return uuid.Nil, err
	}
	if mutation.Before == nil {
		if mutation.After.Version.Value() != aggregateversion.Initial().Value() {
			return uuid.Nil, ErrApprovalPolicyVersionConflict
		}
	} else if mutation.Before.ID != mutation.After.ID || mutation.After.Version.Value() != mutation.Before.Version.Value()+1 {
		return uuid.Nil, ErrApprovalPolicyVersionConflict
	}

	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return uuid.Nil, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// All policy-family writes use the same transaction lock. The overlap
	// query below therefore remains valid under concurrent activations.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended('workflow.approval_policy.selectable', 0))`); err != nil {
		return uuid.Nil, err
	}
	if mutation.After.Status == ApprovalPolicyStatusActive {
		selector := policySelectorFingerprint(mutation.After)
		var conflict bool
		err = tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1
				FROM workflow.approval_policy_version_identity AS v
				JOIN workflow.approval_policy AS p
				  ON p.id = v.policy_id
				JOIN workflow.approval_policy_revision AS r
				  ON r.policy_id = v.policy_id
				 AND r.policy_version = v.policy_version
				WHERE p.status <> 'Superseded'
				  AND r.status = 'Active'
				  AND v.policy_id <> $1
			  AND v.selector_fingerprint = $2
			  AND v.effective_from <= COALESCE($3::timestamptz, 'infinity'::timestamptz)
			  AND COALESCE(v.effective_to, 'infinity'::timestamptz) >= $4
			)`, mutation.After.ID, selector, mutation.After.EffectiveTo, mutation.After.EffectiveFrom).Scan(&conflict)
		if err != nil {
			return uuid.Nil, err
		}
		if conflict {
			return uuid.Nil, ErrApprovalPolicyOverlap
		}
	}

	if mutation.Before == nil {
		_, err = tx.Exec(ctx, `INSERT INTO workflow.approval_policy (id, code, current_policy_version, status, aggregate_version, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7)`, mutation.After.ID, mutation.After.Code, mutation.After.PolicyVersion, mutation.After.Status, mutation.After.Version.Value(), mutation.After.CreatedAt, mutation.After.UpdatedAt)
	} else {
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE workflow.approval_policy SET code=$1, current_policy_version=$2, status=$3, aggregate_version=$4, updated_at=$5 WHERE id=$6 AND aggregate_version=$7`, mutation.After.Code, mutation.After.PolicyVersion, mutation.After.Status, mutation.After.Version.Value(), mutation.After.UpdatedAt, mutation.After.ID, mutation.Before.Version.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return uuid.Nil, ErrApprovalPolicyVersionConflict
		}
	}
	if err != nil {
		return uuid.Nil, mapPostgresApprovalPolicyError(err)
	}
	if mutation.ApprovalRequest != nil {
		if mutation.ApprovalRequest.Request.ID != mutation.After.ApprovalRequestID || repository.requestAuditWriter == nil || repository.requestEventWriter == nil {
			return uuid.Nil, ErrApprovalPolicyApprovalUnavailable
		}
		if err := repository.insertPreparedApprovalRequest(ctx, tx, *mutation.ApprovalRequest); err != nil {
			return uuid.Nil, err
		}
	}

	auditReference, err := repository.auditWriter(ctx, tx, mutation.Audit)
	if err != nil {
		return uuid.Nil, err
	}
	if auditReference == uuid.Nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}

	applicability, err := json.Marshal(mutation.After.Applicability)
	if err != nil {
		return uuid.Nil, err
	}
	scope, err := json.Marshal(mutation.After.Scope)
	if err != nil {
		return uuid.Nil, err
	}
	rules, err := json.Marshal(mutation.After.Rules)
	if err != nil {
		return uuid.Nil, err
	}
	steps, err := json.Marshal(mutation.After.Steps)
	if err != nil {
		return uuid.Nil, err
	}
	var approval []byte
	if mutation.After.Approval != nil {
		approval, err = json.Marshal(mutation.After.Approval)
		if err != nil {
			return uuid.Nil, err
		}
	}
	selector := policySelectorFingerprint(mutation.After)
	candidateFingerprint := mutation.After.CandidateFingerprint()
	versionIdentityTag, err := tx.Exec(ctx, `
		INSERT INTO workflow.approval_policy_version_identity (policy_id, policy_version, code, candidate_fingerprint, selector_fingerprint, effective_from, effective_to, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (policy_id, policy_version) DO NOTHING`, mutation.After.ID, mutation.After.PolicyVersion, mutation.After.Code, candidateFingerprint, selector, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, mutation.After.UpdatedAt)
	if err != nil {
		return uuid.Nil, mapPostgresApprovalPolicyError(err)
	}
	if versionIdentityTag.RowsAffected() == 0 {
		var existingFingerprint string
		if err := tx.QueryRow(ctx, `SELECT candidate_fingerprint FROM workflow.approval_policy_version_identity WHERE policy_id=$1 AND policy_version=$2`, mutation.After.ID, mutation.After.PolicyVersion).Scan(&existingFingerprint); err != nil {
			return uuid.Nil, mapPostgresApprovalPolicyError(err)
		}
		if existingFingerprint != candidateFingerprint {
			return uuid.Nil, ErrApprovalPolicyVersionConflict
		}
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO workflow.approval_policy_revision (policy_id, revision_version, policy_version, code, name, status, applicability, scope, rules, steps, effective_from, effective_to, approval, approval_request_id, activation_outcome, dependent_workflow_impact, audit_reference, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7::jsonb,$8::jsonb,$9::jsonb,$10::jsonb,$11,$12,$13::jsonb,$14,$15,$16,$17,$18)`, mutation.After.ID, mutation.After.Version.Value(), mutation.After.PolicyVersion, mutation.After.Code, mutation.After.Name, mutation.After.Status, applicability, scope, rules, steps, mutation.After.EffectiveFrom, mutation.After.EffectiveTo, approval, nullableUUID(mutation.After.ApprovalRequestID), mutation.After.ActivationOutcome, mutation.After.DependentWorkflowImpact, auditReference, mutation.After.UpdatedAt)
	if err != nil {
		return uuid.Nil, mapPostgresApprovalPolicyError(err)
	}
	for _, rule := range mutation.After.Rules {
		normalized := rule.normalized()
		requiredRoles, marshalErr := json.Marshal(normalized.RequiredRoles)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		delegation, marshalErr := json.Marshal(normalized.Delegation)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		escalation, marshalErr := json.Marshal(normalized.Escalation)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		var minimum, maximum any
		if normalized.Threshold.Minimum != nil {
			minimum = normalized.Threshold.Minimum.String()
		}
		if normalized.Threshold.Maximum != nil {
			maximum = normalized.Threshold.Maximum.String()
		}
		if _, err := tx.Exec(ctx, `INSERT INTO workflow.approval_policy_rule (policy_id, revision_version, rule_id, name, minimum_amount, maximum_amount, currency, required_roles, delegation, escalation) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10::jsonb)`, mutation.After.ID, mutation.After.Version.Value(), normalized.ID, normalized.Name, minimum, maximum, normalized.Threshold.Currency, requiredRoles, delegation, escalation); err != nil {
			return uuid.Nil, mapPostgresApprovalPolicyError(err)
		}
	}
	for _, step := range mutation.After.Steps {
		requiredRoles, marshalErr := json.Marshal(step.RequiredRoles)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		delegation, marshalErr := json.Marshal(step.Delegation)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		escalation, marshalErr := json.Marshal(step.Escalation)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		if _, err := tx.Exec(ctx, `INSERT INTO workflow.approval_policy_step_definition (policy_id, revision_version, step_id, sequence, name, required_roles, minimum_approvals, delegation, escalation) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8::jsonb,$9::jsonb)`, mutation.After.ID, mutation.After.Version.Value(), step.ID, step.Sequence, step.Name, requiredRoles, step.MinimumApprovals, delegation, escalation); err != nil {
			return uuid.Nil, mapPostgresApprovalPolicyError(err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE workflow.approval_policy SET last_audit_reference=$1 WHERE id=$2`, auditReference, mutation.After.ID); err != nil {
		return uuid.Nil, err
	}
	if durable != nil {
		if durable.Coordinator == nil || durable.Acquisition.Decision() != platformidempotency.DecisionExecute {
			return uuid.Nil, platformidempotency.ErrInvalidAcquisition
		}
		result := durable.Result
		if durable.ResultBuilder != nil {
			var err error
			result, err = durable.ResultBuilder(auditReference)
			if err != nil {
				return uuid.Nil, err
			}
		}
		if err := durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, result); err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	committed = true
	return auditReference, nil
}

func (repository *PostgresApprovalPolicyRepository) insertPreparedApprovalRequest(ctx context.Context, tx pgx.Tx, prepared PreparedApprovalRequest) error {
	request := prepared.Request
	if err := request.Validate(); err != nil {
		return err
	}
	scope, err := json.Marshal(request.Scope)
	if err != nil {
		return err
	}
	steps, err := json.Marshal(request.Steps)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO workflow.approval_request (id, subject_type, subject_id, subject_version, subject_fingerprint, policy_id, policy_version, scope, steps, status, aggregate_version, requester_user_id, requester_subject_reference, correlation_id, causation_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10,$11,$12,$13,$14,$15,$16,$17)`, request.ID, request.SubjectType, request.SubjectID, request.SubjectVersion, request.SubjectFingerprint, request.PolicyID, request.PolicyVersion, scope, steps, request.Status, request.Version.Value(), request.Requester.UserID, request.Requester.SubjectReference, nullableUUIDString(request.CorrelationID), nullableUUIDString(request.CausationID), request.CreatedAt, request.UpdatedAt); err != nil {
		return mapPostgresApprovalRequestError(err)
	}
	if auditReference, err := repository.requestAuditWriter(ctx, tx, prepared.Audit); err != nil {
		return err
	} else if auditReference == uuid.Nil {
		return ErrApprovalPolicyAuditUnavailable
	}
	if err := repository.requestEventWriter(ctx, tx, prepared.Event); err != nil {
		return err
	}
	return nil
}

type workflowQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (repository *PostgresApprovalPolicyRepository) readPolicy(ctx context.Context, queryer workflowQueryer, id uuid.UUID) (ApprovalPolicy, error) {
	var rootID uuid.UUID
	var code, currentPolicyVersion, status string
	var aggregateVersion int64
	var createdAt, updatedAt time.Time
	if err := queryer.QueryRow(ctx, `SELECT id, code, current_policy_version, status, aggregate_version, created_at, updated_at FROM workflow.approval_policy WHERE id=$1`, id).Scan(&rootID, &code, &currentPolicyVersion, &status, &aggregateVersion, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApprovalPolicy{}, ErrApprovalPolicyNotFound
		}
		return ApprovalPolicy{}, err
	}
	rows, err := queryer.Query(ctx, `SELECT policy_id, revision_version, policy_version, code, name, status, applicability, scope, rules, steps, effective_from, effective_to, approval, approval_request_id, activation_outcome, dependent_workflow_impact, created_at FROM workflow.approval_policy_revision WHERE policy_id=$1 ORDER BY revision_version`, id)
	if err != nil {
		return ApprovalPolicy{}, err
	}
	defer rows.Close()
	var current ApprovalPolicy
	var history []ApprovalPolicyRevision
	for rows.Next() {
		var revision ApprovalPolicyRevision
		var revisionID uuid.UUID
		var revisionVersion int64
		var policyVersion, revisionCode, name, revisionStatus, activationOutcome, dependentWorkflowImpact string
		var applicability, scope, rules, steps, approval []byte
		var requestID *uuid.UUID
		var effectiveFrom time.Time
		var effectiveTo *time.Time
		var revisionCreatedAt time.Time
		if err := rows.Scan(&revisionID, &revisionVersion, &policyVersion, &revisionCode, &name, &revisionStatus, &applicability, &scope, &rules, &steps, &effectiveFrom, &effectiveTo, &approval, &requestID, &activationOutcome, &dependentWorkflowImpact, &revisionCreatedAt); err != nil {
			return ApprovalPolicy{}, err
		}
		var decodedApplicability ApprovalApplicability
		var decodedScope ApprovalPolicyScope
		var decodedRules []ApprovalRule
		var decodedSteps []ApprovalStepDefinition
		if err := json.Unmarshal(applicability, &decodedApplicability); err != nil {
			return ApprovalPolicy{}, err
		}
		if err := json.Unmarshal(scope, &decodedScope); err != nil {
			return ApprovalPolicy{}, err
		}
		if err := json.Unmarshal(rules, &decodedRules); err != nil {
			return ApprovalPolicy{}, err
		}
		if err := json.Unmarshal(steps, &decodedSteps); err != nil {
			return ApprovalPolicy{}, err
		}
		var decodedApproval *ApprovalDecisionReference
		if len(approval) > 0 {
			decodedApproval = &ApprovalDecisionReference{}
			if err := json.Unmarshal(approval, decodedApproval); err != nil {
				return ApprovalPolicy{}, err
			}
		}
		snapshot := ApprovalPolicySnapshot{ID: revisionID, Code: revisionCode, Name: name, PolicyVersion: policyVersion, Status: revisionStatus, Applicability: decodedApplicability, Scope: decodedScope, Rules: decodedRules, Steps: decodedSteps, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, Approval: decodedApproval, ActivationOutcome: activationOutcome, DependentWorkflowImpact: dependentWorkflowImpact}
		if requestID != nil {
			snapshot.ApprovalRequestID = *requestID
		}
		revision = ApprovalPolicyRevision{PolicyVersion: policyVersion, Version: aggregateversion.AggregateVersion(revisionVersion), Snapshot: snapshot, CreatedAt: revisionCreatedAt}
		if revisionVersion == aggregateVersion {
			current = ApprovalPolicy{ID: rootID, Code: code, Name: name, PolicyVersion: currentPolicyVersion, Status: status, Applicability: decodedApplicability, Scope: decodedScope, Rules: decodedRules, Steps: decodedSteps, EffectiveFrom: effectiveFrom, EffectiveTo: effectiveTo, Approval: cloneApproval(decodedApproval), Version: aggregateversion.AggregateVersion(aggregateVersion), CreatedAt: createdAt, UpdatedAt: updatedAt, ApprovalRequestID: snapshot.ApprovalRequestID, ActivationOutcome: activationOutcome, DependentWorkflowImpact: dependentWorkflowImpact}
		} else {
			history = append(history, revision)
		}
	}
	if err := rows.Err(); err != nil {
		return ApprovalPolicy{}, err
	}
	if current.ID == uuid.Nil {
		return ApprovalPolicy{}, fmt.Errorf("%w: current revision is missing", ErrApprovalPolicyNotFound)
	}
	current.History = history
	return current, nil
}

func policySelectorFingerprint(policy ApprovalPolicy) string {
	return selectorFingerprint(policy.Applicability, policy.Scope)
}

func nullableUUID(value uuid.UUID) *uuid.UUID {
	if value == uuid.Nil {
		return nil
	}
	return &value
}

func mapPostgresApprovalPolicyError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			if strings.Contains(pgErr.ConstraintName, "effective") || strings.Contains(pgErr.ConstraintName, "selector") {
				return ErrApprovalPolicyOverlap
			}
			return ErrApprovalPolicyDuplicate
		case "23503":
			return ErrApprovalPolicyVersionConflict
		}
	}
	return err
}

var _ ApprovalPolicyRepository = (*PostgresApprovalPolicyRepository)(nil)
