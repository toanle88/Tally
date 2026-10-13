package workflow

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
	var scope, steps []byte
	var version int64
	var createdAt, updatedAt time.Time
	if err := repository.pool.QueryRow(ctx, `SELECT id, subject_type, subject_id, subject_version, subject_fingerprint, policy_id, policy_version, scope, steps, status, aggregate_version, requester_user_id, requester_subject_reference, correlation_id, causation_id, created_at, updated_at FROM workflow.approval_request WHERE id=$1`, id).Scan(&request.ID, &request.SubjectType, &request.SubjectID, &request.SubjectVersion, &request.SubjectFingerprint, &request.PolicyID, &request.PolicyVersion, &scope, &steps, &request.Status, &version, &requesterID, &request.Requester.SubjectReference, &correlationID, &causationID, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApprovalRequest{}, ErrApprovalPolicyNotFound
		}
		return ApprovalRequest{}, err
	}
	request.Requester.UserID = requesterID
	request.Version = aggregateversion.AggregateVersion(version)
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
	decisionRows, err := repository.pool.Query(ctx, `SELECT id, decision_version, outcome, approver_user_id, approver_subject_reference, reason, policy_version, subject_version, candidate_fingerprint, recorded_at FROM workflow.approval_decision WHERE approval_request_id=$1 ORDER BY decision_version`, id)
	if err != nil {
		return ApprovalRequest{}, err
	}
	defer decisionRows.Close()
	for decisionRows.Next() {
		var decision ApprovalDecision
		if err := decisionRows.Scan(&decision.ID, &decision.DecisionVersion, &decision.Outcome, &decision.Approver.UserID, &decision.Approver.SubjectReference, &decision.Reason, &decision.PolicyVersion, &decision.SubjectVersion, &decision.CandidateFingerprint, &decision.RecordedAt); err != nil {
			return ApprovalRequest{}, err
		}
		request.Decisions = append(request.Decisions, decision)
	}
	if err := decisionRows.Err(); err != nil {
		return ApprovalRequest{}, err
	}
	if err := request.Validate(); err != nil {
		return ApprovalRequest{}, err
	}
	return request, nil
}

func (repository *PostgresApprovalRequestRepository) Save(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent) error {
	return repository.save(ctx, request, before, auditRecord, event, nil)
}

func (repository *PostgresApprovalRequestRepository) SaveWithIdempotency(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent, durable DurableApprovalRequestCommit) error {
	return repository.save(ctx, request, before, auditRecord, event, &durable)
}

func (repository *PostgresApprovalRequestRepository) save(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent, durable *DurableApprovalRequestCommit) error {
	if repository == nil || repository.pool == nil {
		return ErrInvalidApprovalPolicyService
	}
	if repository.auditWriter == nil || repository.eventWriter == nil {
		return ErrApprovalPolicyAuditUnavailable
	}
	if err := request.Validate(); err != nil {
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
	if before == nil {
		scope, marshalErr := json.Marshal(request.Scope)
		if marshalErr != nil {
			return marshalErr
		}
		steps, marshalErr := json.Marshal(request.Steps)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = tx.Exec(ctx, `INSERT INTO workflow.approval_request (id, subject_type, subject_id, subject_version, subject_fingerprint, policy_id, policy_version, scope, steps, status, aggregate_version, requester_user_id, requester_subject_reference, correlation_id, causation_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8::jsonb,$9::jsonb,$10,$11,$12,$13,$14,$15,$16,$17)`, request.ID, request.SubjectType, request.SubjectID, request.SubjectVersion, request.SubjectFingerprint, request.PolicyID, request.PolicyVersion, scope, steps, request.Status, request.Version.Value(), request.Requester.UserID, request.Requester.SubjectReference, nullableUUIDString(request.CorrelationID), nullableUUIDString(request.CausationID), request.CreatedAt, request.UpdatedAt)
	} else {
		steps, marshalErr := json.Marshal(request.Steps)
		if marshalErr != nil {
			return marshalErr
		}
		var tag pgconn.CommandTag
		tag, err = tx.Exec(ctx, `UPDATE workflow.approval_request SET steps=$1::jsonb, status=$2, aggregate_version=$3, updated_at=$4 WHERE id=$5 AND aggregate_version=$6`, steps, request.Status, request.Version.Value(), request.UpdatedAt, request.ID, before.Version.Value())
		if err == nil && tag.RowsAffected() != 1 {
			return ErrApprovalPolicyVersionConflict
		}
		if err == nil && len(request.Decisions) > len(before.Decisions) {
			decision := request.Decisions[len(request.Decisions)-1]
			_, err = tx.Exec(ctx, `INSERT INTO workflow.approval_decision (id, approval_request_id, decision_version, outcome, approver_user_id, approver_subject_reference, reason, policy_version, subject_version, candidate_fingerprint, recorded_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, decision.ID, request.ID, decision.DecisionVersion, decision.Outcome, decision.Approver.UserID, decision.Approver.SubjectReference, decision.Reason, decision.PolicyVersion, decision.SubjectVersion, decision.CandidateFingerprint, decision.RecordedAt)
		}
	}
	if err != nil {
		return mapPostgresApprovalRequestError(err)
	}
	auditReference, err := repository.auditWriter(ctx, tx, auditRecord)
	if err != nil {
		return err
	}
	if auditReference == uuid.Nil {
		return ErrApprovalPolicyAuditUnavailable
	}
	if err := repository.eventWriter(ctx, tx, event); err != nil {
		return err
	}
	if durable != nil {
		if durable.Coordinator == nil || durable.Acquisition.Decision() != platformidempotency.DecisionExecute {
			return platformidempotency.ErrInvalidAcquisition
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

var _ ApprovalRequestRepository = (*PostgresApprovalRequestRepository)(nil)
var _ ApprovalWorkflowPort = (*PostgresApprovalWorkflow)(nil)
var _ ApprovalWorkflowPreparationPort = (*PostgresApprovalWorkflow)(nil)
