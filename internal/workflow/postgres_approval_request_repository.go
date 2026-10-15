package workflow

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
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type PostgresApprovalRequestAuditWriter func(context.Context, pgx.Tx, ApprovalRequestAuditRecord) (uuid.UUID, error)
type PostgresApprovalEventWriter func(context.Context, pgx.Tx, ApprovalEvent) error

type PostgresApprovalRequestRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresApprovalRequestAuditWriter
	eventWriter PostgresApprovalEventWriter
}

func NewPostgresApprovalRequestRepository(pool *pgxpool.Pool, auditWriter PostgresApprovalRequestAuditWriter, eventWriter PostgresApprovalEventWriter) (*PostgresApprovalRequestRepository, error) {
	if pool == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return &PostgresApprovalRequestRepository{pool: pool, auditWriter: auditWriter, eventWriter: eventWriter}, nil
}

func (repository *PostgresApprovalRequestRepository) Get(ctx context.Context, id uuid.UUID) (ApprovalRequest, error) {
	if repository == nil || repository.pool == nil {
		return ApprovalRequest{}, ErrInvalidApprovalPolicyService
	}
	var request ApprovalRequest
	var requesterID uuid.UUID
	var correlationID, causationID *uuid.UUID
	var snapshot, dimensions, roles, scope, steps []byte
	var amountText *string
	var dueAt *time.Time
	var version int64
	var createdAt, updatedAt time.Time
	if err := repository.pool.QueryRow(ctx, `SELECT id, subject_type, subject_id, subject_version, subject_fingerprint, subject_snapshot, snapshot_fingerprint, source_context, operation, transaction_type, data_classification, dimensions, amount::text, currency, due_at, responsible_roles, policy_id, policy_version, scope, steps, status, aggregate_version, requester_user_id, requester_subject_reference, correlation_id, causation_id, created_at, updated_at FROM workflow.approval_request WHERE id=$1`, id).Scan(&request.ID, &request.SubjectType, &request.SubjectID, &request.SubjectVersion, &request.SubjectFingerprint, &snapshot, &request.SnapshotFingerprint, &request.SourceContext, &request.Operation, &request.TransactionType, &request.DataClassification, &dimensions, &amountText, &request.Currency, &dueAt, &roles, &request.PolicyID, &request.PolicyVersion, &scope, &steps, &request.Status, &version, &requesterID, &request.Requester.SubjectReference, &correlationID, &causationID, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApprovalRequest{}, ErrApprovalPolicyNotFound
		}
		return ApprovalRequest{}, err
	}
	request.Requester.UserID = requesterID
	request.Version = aggregateversion.AggregateVersion(version)
	if len(snapshot) > 0 {
		if err := json.Unmarshal(snapshot, &request.SubjectSnapshot); err != nil {
			return ApprovalRequest{}, err
		}
	}
	if len(dimensions) > 0 {
		if err := json.Unmarshal(dimensions, &request.Dimensions); err != nil {
			return ApprovalRequest{}, err
		}
	}
	if len(roles) > 0 {
		if err := json.Unmarshal(roles, &request.ResponsibleRoles); err != nil {
			return ApprovalRequest{}, err
		}
	}
	if amountText != nil && strings.TrimSpace(*amountText) != "" {
		amount, err := decimal.NewFromString(*amountText)
		if err != nil {
			return ApprovalRequest{}, err
		}
		request.Amount = &amount
	}
	request.DueAt = dueAt
	request.Scope = ApprovalPolicyScope{}
	if err := json.Unmarshal(scope, &request.Scope); err != nil {
		return ApprovalRequest{}, err
	}
	if err := json.Unmarshal(steps, &request.Steps); err != nil {
		return ApprovalRequest{}, err
	}
	request.CorrelationID = uuidString(correlationID)
	request.CausationID = uuidString(causationID)
	request.CreatedAt, request.UpdatedAt = createdAt, updatedAt
	decisionRows, err := repository.pool.Query(ctx, `SELECT id, decision_version, outcome, approver_user_id, approver_subject_reference, reason, policy_version, subject_version, candidate_fingerprint, step_id, delegation_reference, recorded_at FROM workflow.approval_decision WHERE approval_request_id=$1 ORDER BY decision_version`, id)
	if err != nil {
		return ApprovalRequest{}, err
	}
	defer decisionRows.Close()
	for decisionRows.Next() {
		var decision ApprovalDecision
		var stepID *uuid.UUID
		if err := decisionRows.Scan(&decision.ID, &decision.DecisionVersion, &decision.Outcome, &decision.Approver.UserID, &decision.Approver.SubjectReference, &decision.Reason, &decision.PolicyVersion, &decision.SubjectVersion, &decision.CandidateFingerprint, &stepID, &decision.DelegationReference, &decision.RecordedAt); err != nil {
			return ApprovalRequest{}, err
		}
		if stepID != nil {
			decision.StepID = *stepID
		}
		request.Decisions = append(request.Decisions, decision)
	}
	if err := decisionRows.Err(); err != nil {
		return ApprovalRequest{}, err
	}
	escalationRows, err := repository.pool.Query(ctx, `SELECT id, step_id, reason, blocker, responsible_roles, target_roles, actor_user_id, actor_subject_reference, process_reference, policy_id, policy_version, subject_version, candidate_fingerprint, due_at, occurred_at, audit_reference, correlation_id, causation_id FROM workflow.approval_request_escalation WHERE approval_request_id=$1 ORDER BY occurred_at, id`, id)
	if err != nil {
		return ApprovalRequest{}, err
	}
	defer escalationRows.Close()
	for escalationRows.Next() {
		var escalation ApprovalEscalation
		var responsibleRoles, targetRoles []byte
		var actorUserID *uuid.UUID
		var actorSubjectReference, processReference *string
		var dueAt *time.Time
		var correlationID, causationID *uuid.UUID
		if err := escalationRows.Scan(&escalation.ID, &escalation.StepID, &escalation.Reason, &escalation.Blocker, &responsibleRoles, &targetRoles, &actorUserID, &actorSubjectReference, &processReference, &escalation.PolicyID, &escalation.PolicyVersion, &escalation.SubjectVersion, &escalation.CandidateFingerprint, &dueAt, &escalation.OccurredAt, &escalation.AuditReference, &correlationID, &causationID); err != nil {
			return ApprovalRequest{}, err
		}
		if err := json.Unmarshal(responsibleRoles, &escalation.ResponsibleRoles); err != nil {
			return ApprovalRequest{}, err
		}
		if err := json.Unmarshal(targetRoles, &escalation.TargetRoles); err != nil {
			return ApprovalRequest{}, err
		}
		if actorUserID != nil {
			escalation.ActorUserID = *actorUserID
		}
		if actorSubjectReference != nil {
			escalation.ActorSubjectReference = *actorSubjectReference
		}
		if processReference != nil {
			escalation.ProcessReference = *processReference
		}
		escalation.DueAt = dueAt
		escalation.CorrelationID = uuidString(correlationID)
		escalation.CausationID = uuidString(causationID)
		request.Escalations = append(request.Escalations, escalation)
	}
	if err := escalationRows.Err(); err != nil {
		return ApprovalRequest{}, err
	}
	if err := request.Validate(); err != nil {
		return ApprovalRequest{}, err
	}
	return request, nil
}

func (repository *PostgresApprovalRequestRepository) List(ctx context.Context) ([]ApprovalRequest, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM workflow.approval_request ORDER BY created_at DESC, id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	requests := make([]ApprovalRequest, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		request, err := repository.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		requests = append(requests, request)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return requests, nil
}

func (repository *PostgresApprovalRequestRepository) ListDueApprovalEscalationCandidates(ctx context.Context, now time.Time, limit int) ([]ApprovalEscalationCandidate, error) {
	if repository == nil || repository.pool == nil || limit < 1 || limit > 100 {
		return nil, ErrInvalidApprovalPolicyCommand
	}
	rows, err := repository.pool.Query(ctx, `SELECT id FROM workflow.approval_request WHERE status=$1 AND due_at IS NOT NULL AND due_at <= $2 ORDER BY due_at, id LIMIT $3`, ApprovalRequestStatusPending, now.UTC(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	candidates := make([]ApprovalEscalationCandidate, 0, limit)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		request, err := repository.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		step, err := actionableApprovalRequestStep(request, uuid.Nil)
		if err != nil || step.Status != ApprovalRequestStatusPending || !step.Escalation.Enabled || step.Escalation.AfterSeconds < 1 || len(step.Escalation.EscalateToRoles) == 0 {
			continue
		}
		alreadyEscalated := false
		for _, escalation := range request.Escalations {
			if escalation.StepID == step.ID {
				alreadyEscalated = true
				break
			}
		}
		if alreadyEscalated {
			continue
		}
		candidates = append(candidates, ApprovalEscalationCandidate{RequestID: request.ID, ExpectedVersion: request.Version, StepID: step.ID, PolicyVersion: request.PolicyVersion, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, DueAt: request.DueAt.UTC()})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return candidates, nil
}

func (repository *PostgresApprovalRequestRepository) Save(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent) (uuid.UUID, error) {
	return repository.save(ctx, request, before, auditRecord, event, nil)
}

func (repository *PostgresApprovalRequestRepository) SaveWithIdempotency(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent, durable DurableApprovalRequestCommit) (uuid.UUID, error) {
	return repository.save(ctx, request, before, auditRecord, event, &durable)
}

func (repository *PostgresApprovalRequestRepository) save(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent, durable *DurableApprovalRequestCommit) (uuid.UUID, error) {
	if repository == nil || repository.pool == nil {
		return uuid.Nil, ErrInvalidApprovalPolicyService
	}
	if repository.auditWriter == nil || repository.eventWriter == nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	if err := request.Validate(); err != nil {
		return uuid.Nil, err
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
	if before == nil {
		scope, marshalErr := json.Marshal(request.Scope)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		steps, marshalErr := json.Marshal(request.Steps)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		snapshot := []byte(`{}`)
		if request.SnapshotFingerprint != "" {
			snapshot, marshalErr = json.Marshal(request.SubjectSnapshot)
			if marshalErr != nil {
				return uuid.Nil, marshalErr
			}
		}
		dimensionValues := request.Dimensions
		if dimensionValues == nil {
			dimensionValues = map[string]string{}
		}
		dimensions, marshalErr := json.Marshal(dimensionValues)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		roleValues := request.ResponsibleRoles
		if roleValues == nil {
			roleValues = []string{}
		}
		roles, marshalErr := json.Marshal(roleValues)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		var amount any
		if request.Amount != nil {
			amount = request.Amount.String()
		}
		_, err = tx.Exec(ctx, `INSERT INTO workflow.approval_request (id, subject_type, subject_id, subject_version, subject_fingerprint, subject_snapshot, snapshot_fingerprint, source_context, operation, transaction_type, data_classification, dimensions, amount, currency, due_at, responsible_roles, policy_id, policy_version, scope, steps, status, aggregate_version, requester_user_id, requester_subject_reference, correlation_id, causation_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12::jsonb,$13,$14,$15,$16::jsonb,$17,$18,$19::jsonb,$20::jsonb,$21,$22,$23,$24,$25,$26,$27,$28)`, request.ID, request.SubjectType, request.SubjectID, request.SubjectVersion, request.SubjectFingerprint, snapshot, request.SnapshotFingerprint, request.SourceContext, request.Operation, request.TransactionType, request.DataClassification, dimensions, amount, request.Currency, request.DueAt, roles, request.PolicyID, request.PolicyVersion, scope, steps, request.Status, request.Version.Value(), request.Requester.UserID, request.Requester.SubjectReference, nullableUUIDString(request.CorrelationID), nullableUUIDString(request.CausationID), request.CreatedAt, request.UpdatedAt)
	} else {
		steps, marshalErr := json.Marshal(request.Steps)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE workflow.approval_request SET steps=$1::jsonb, status=$2, aggregate_version=$3, updated_at=$4 WHERE id=$5 AND aggregate_version=$6`, steps, request.Status, request.Version.Value(), request.UpdatedAt, request.ID, before.Version.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return uuid.Nil, ErrApprovalPolicyVersionConflict
		}
		if err == nil && len(request.Decisions) > len(before.Decisions) {
			decision := request.Decisions[len(request.Decisions)-1]
			_, err = tx.Exec(ctx, `INSERT INTO workflow.approval_decision (id, approval_request_id, decision_version, outcome, approver_user_id, approver_subject_reference, reason, policy_version, subject_version, candidate_fingerprint, step_id, delegation_reference, recorded_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`, decision.ID, request.ID, decision.DecisionVersion, decision.Outcome, decision.Approver.UserID, decision.Approver.SubjectReference, decision.Reason, decision.PolicyVersion, decision.SubjectVersion, decision.CandidateFingerprint, nullableUUID(decision.StepID), decision.DelegationReference, decision.RecordedAt)
		}
	}
	if err != nil {
		return uuid.Nil, mapPostgresApprovalRequestError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, auditRecord)
	if err != nil {
		return uuid.Nil, err
	}
	if auditReference == uuid.Nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	if event.EscalationID != uuid.Nil {
		if before == nil || len(request.Escalations) <= len(before.Escalations) {
			return uuid.Nil, ErrInvalidApprovalPolicyCommand
		}
		escalation := request.Escalations[len(request.Escalations)-1]
		if escalation.ID != event.EscalationID {
			return uuid.Nil, ErrInvalidApprovalPolicyCommand
		}
		responsibleRoles, marshalErr := json.Marshal(escalation.ResponsibleRoles)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		targetRoles, marshalErr := json.Marshal(escalation.TargetRoles)
		if marshalErr != nil {
			return uuid.Nil, marshalErr
		}
		var actorUserID any
		if escalation.ActorUserID != uuid.Nil {
			actorUserID = escalation.ActorUserID
		}
		_, err = tx.Exec(ctx, `INSERT INTO workflow.approval_request_escalation (id, approval_request_id, step_id, reason, blocker, responsible_roles, target_roles, actor_user_id, actor_subject_reference, process_reference, policy_id, policy_version, subject_version, candidate_fingerprint, due_at, occurred_at, audit_reference, correlation_id, causation_id) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7::jsonb,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`, escalation.ID, request.ID, escalation.StepID, escalation.Reason, escalation.Blocker, responsibleRoles, targetRoles, actorUserID, nullableText(escalation.ActorSubjectReference), nullableText(escalation.ProcessReference), escalation.PolicyID, escalation.PolicyVersion, escalation.SubjectVersion, escalation.CandidateFingerprint, escalation.DueAt, escalation.OccurredAt, auditReference, nullableUUIDString(escalation.CorrelationID), nullableUUIDString(escalation.CausationID))
		if err != nil {
			return uuid.Nil, mapPostgresApprovalRequestError(err)
		}
	}
	if err := repository.eventWriter(ctx, tx, event); err != nil {
		return uuid.Nil, err
	}
	if durable != nil && durable.ResultBuilder != nil {
		result, buildErr := durable.ResultBuilder(auditReference)
		if buildErr != nil {
			return uuid.Nil, buildErr
		}
		durable.Result = result
	}
	if durable != nil {
		if durable.Coordinator == nil || durable.Acquisition.Decision() != platformidempotency.DecisionExecute {
			return uuid.Nil, platformidempotency.ErrInvalidAcquisition
		}
		if err := durable.Coordinator.Finalize(ctx, tx, durable.Acquisition, durable.Result); err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	committed = true
	return auditReference, nil
}

type PostgresApprovalWorkflow struct {
	Requests *PostgresApprovalRequestRepository
	Service  *ApprovalRequestService
}

func (workflow *PostgresApprovalWorkflow) CreatePolicyApproval(ctx context.Context, actor Actor, policy ApprovalPolicy, command ApprovalPolicyCommand) (ApprovalRequest, error) {
	if workflow == nil || workflow.Service == nil {
		return ApprovalRequest{}, ErrApprovalPolicyApprovalUnavailable
	}
	result, err := workflow.Service.CreateApprovalRequest(ctx, actor, CreateApprovalRequestCommand{RequestID: policy.ApprovalRequestID, SubjectType: "approval-policy", SubjectID: policy.ID.String(), SubjectVersion: policy.Version.Value(), SubjectFingerprint: policy.CandidateFingerprint(), PolicyID: policy.ID, PolicyVersion: policy.PolicyVersion, Scope: policy.Scope, Steps: policy.Steps, IdempotencyKey: command.IdempotencyKey + ":approval-request", CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	if err != nil {
		return ApprovalRequest{}, err
	}
	return result.Request, nil
}

func (workflow *PostgresApprovalWorkflow) PreparePolicyApproval(ctx context.Context, actor Actor, policy ApprovalPolicy, command ApprovalPolicyCommand) (PreparedApprovalRequest, error) {
	if workflow == nil || workflow.Service == nil {
		return PreparedApprovalRequest{}, ErrApprovalPolicyApprovalUnavailable
	}
	return workflow.Service.prepareCreateApprovalRequest(ctx, actor, CreateApprovalRequestCommand{
		RequestID:          policy.ApprovalRequestID,
		SubjectType:        "approval-policy",
		SubjectID:          policy.ID.String(),
		SubjectVersion:     policy.Version.Value(),
		SubjectFingerprint: policy.CandidateFingerprint(),
		PolicyID:           policy.ID,
		PolicyVersion:      policy.PolicyVersion,
		Scope:              policy.Scope,
		Steps:              policy.Steps,
		IdempotencyKey:     command.IdempotencyKey + ":approval-request",
		CorrelationID:      command.CorrelationID,
		CausationID:        command.CausationID,
	})
}

func (workflow *PostgresApprovalWorkflow) ValidatePolicyApproval(ctx context.Context, actor Actor, policy ApprovalPolicy, reference ApprovalDecisionReference) error {
	if workflow == nil || workflow.Requests == nil {
		return ErrApprovalPolicyApprovalUnavailable
	}
	request, err := workflow.Requests.Get(ctx, reference.ApprovalRequestID)
	if err != nil {
		return err
	}
	if request.Status != ApprovalRequestStatusApproved || request.PolicyID != policy.ID || request.PolicyVersion != policy.PolicyVersion || request.SubjectFingerprint != policy.CandidateFingerprint() {
		return ErrApprovalPolicyApprovalRejected
	}
	if request.Requester.UserID == reference.ApproverUserID {
		return ErrApprovalPolicySelfApproval
	}
	for _, decision := range request.Decisions {
		if decision.ID == reference.DecisionID && decision.DecisionVersion == reference.DecisionVersion && decision.PolicyVersion == reference.PolicyVersion && decision.Outcome == ApprovalDecisionApproved && decision.Approver.UserID == reference.ApproverUserID && decision.CandidateFingerprint == reference.CandidateFingerprint && decision.SubjectVersion == reference.SubjectVersion {
			return nil
		}
	}
	return ErrApprovalPolicyApprovalRejected
}

func NewPostgresApprovalWorkflow(pool *pgxpool.Pool, auditWriter PostgresApprovalRequestAuditWriter, eventWriter PostgresApprovalEventWriter, authorizer ApprovalRequestAuthorizer, clock func() time.Time) (*PostgresApprovalWorkflow, error) {
	return newPostgresApprovalWorkflow(pool, auditWriter, eventWriter, authorizer, clock, nil)
}

func NewPostgresApprovalWorkflowWithDurableIdempotency(pool *pgxpool.Pool, auditWriter PostgresApprovalRequestAuditWriter, eventWriter PostgresApprovalEventWriter, authorizer ApprovalRequestAuthorizer, clock func() time.Time, durable DurableApprovalRequestServiceConfig) (*PostgresApprovalWorkflow, error) {
	if durable.Database == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return newPostgresApprovalWorkflow(pool, auditWriter, eventWriter, authorizer, clock, &durable)
}

func newPostgresApprovalWorkflow(pool *pgxpool.Pool, auditWriter PostgresApprovalRequestAuditWriter, eventWriter PostgresApprovalEventWriter, authorizer ApprovalRequestAuthorizer, clock func() time.Time, durable *DurableApprovalRequestServiceConfig) (*PostgresApprovalWorkflow, error) {
	if authorizer == nil || clock == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	repository, err := NewPostgresApprovalRequestRepository(pool, auditWriter, eventWriter)
	if err != nil {
		return nil, err
	}
	// The repository invokes the transaction-bound audit and event writers;
	// the service arguments are retained as non-nil constructor dependencies so
	// the same service can also be used with the memory repository.
	audit := &MemoryApprovalRequestAuditRecorder{}
	events := &MemoryApprovalEventRecorder{}
	var service *ApprovalRequestService
	if durable != nil {
		service, err = NewApprovalRequestServiceWithDurableIdempotency(repository, authorizer, audit, events, clock, *durable)
	} else {
		service, err = NewApprovalRequestService(repository, authorizer, audit, events, clock)
	}
	if err != nil {
		return nil, err
	}
	return &PostgresApprovalWorkflow{Requests: repository, Service: service}, nil
}

func mapPostgresApprovalRequestError(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505":
			return ErrApprovalPolicyDuplicate
		case "23503":
			return ErrApprovalPolicyVersionConflict
		}
	}
	return err
}

func uuidString(value *uuid.UUID) string {
	if value == nil || *value == uuid.Nil {
		return ""
	}
	return value.String()
}

func nullableUUIDString(value string) *uuid.UUID {
	if value == "" {
		return nil
	}
	parsed, err := uuid.Parse(value)
	if err != nil || parsed == uuid.Nil {
		return nil
	}
	return &parsed
}

func nullableText(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

var _ ApprovalRequestRepository = (*PostgresApprovalRequestRepository)(nil)
var _ ApprovalEscalationCandidateReader = (*PostgresApprovalRequestRepository)(nil)
var _ ApprovalWorkflowPort = (*PostgresApprovalWorkflow)(nil)
var _ ApprovalWorkflowPreparationPort = (*PostgresApprovalWorkflow)(nil)
