package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	DelegateApprovalPermission = "finance.wfa.delegate.approval"
	ApprovalDelegatedEventType = "approval_delegated"
)

var (
	ErrInvalidDelegation                  = errors.New("invalid workflow delegation")
	ErrInvalidDelegationCommand           = errors.New("invalid workflow delegation command")
	ErrDelegationNotFound                 = errors.New("workflow delegation not found")
	ErrDelegationAuthorizationDenied      = errors.New("workflow delegation authorization denied")
	ErrDelegationAuthorizationUnavailable = errors.New("workflow delegation authorization unavailable")
	ErrDelegationAuthorizationStale       = errors.New("workflow delegation authorization is stale")
	ErrDelegationExpired                  = errors.New("workflow delegation is expired")
	ErrDelegationPolicyConflict           = errors.New("workflow delegation policy conflict")
	ErrDelegationScopeConflict            = errors.New("workflow delegation scope conflict")
	ErrDelegationPolicyIncompatible       = errors.New("workflow policy does not permit delegation")
	ErrDelegationDuplicate                = errors.New("duplicate workflow delegation identity")
	ErrDelegationIdempotencyConflict      = errors.New("workflow delegation idempotency conflict")
	ErrDelegationCommandInProgress        = errors.New("workflow delegation command is already in progress")
	ErrDelegationAuditUnavailable         = errors.New("workflow delegation audit unavailable")
	ErrDelegationDurableCommandFailed     = errors.New("workflow delegation command previously failed")
)

// Delegation is an immutable, policy-versioned authority record. Its active
// state is derived from the effective interval; this slice intentionally has
// no mutation or revoke operation.
type Delegation struct {
	ID              uuid.UUID                         `json:"id"`
	DelegatorUserID uuid.UUID                         `json:"delegatorUserId"`
	DelegateUserID  uuid.UUID                         `json:"delegateUserId"`
	PolicyID        uuid.UUID                         `json:"policyId"`
	PolicyVersion   string                            `json:"policyVersion"`
	Scope           ApprovalPolicyScope               `json:"scope"`
	EffectiveFrom   time.Time                         `json:"effectiveFrom"`
	EffectiveTo     time.Time                         `json:"effectiveTo"`
	Version         aggregateversion.AggregateVersion `json:"aggregateVersion"`
	AuditReference  uuid.UUID                         `json:"auditReference,omitempty"`
	CorrelationID   string                            `json:"correlationId,omitempty"`
	CausationID     string                            `json:"causationId,omitempty"`
	CreatedAt       time.Time                         `json:"createdAt"`
	UpdatedAt       time.Time                         `json:"updatedAt"`
}

func (delegation Delegation) Validate() error {
	if delegation.ID == uuid.Nil || delegation.DelegatorUserID == uuid.Nil || delegation.DelegateUserID == uuid.Nil || delegation.PolicyID == uuid.Nil {
		return fmt.Errorf("%w: delegation identities are required", ErrInvalidDelegation)
	}
	if delegation.DelegatorUserID == delegation.DelegateUserID {
		return fmt.Errorf("%w: delegator and delegate must be different actors", ErrInvalidDelegation)
	}
	if strings.TrimSpace(delegation.PolicyVersion) == "" {
		return fmt.Errorf("%w: policy version is required", ErrInvalidDelegation)
	}
	if err := delegation.Scope.Validate(); err != nil {
		return err
	}
	if delegation.EffectiveFrom.IsZero() || delegation.EffectiveTo.IsZero() || !delegation.EffectiveTo.After(delegation.EffectiveFrom) {
		return fmt.Errorf("%w: effective interval must be finite and ordered", ErrInvalidDelegation)
	}
	if delegation.Version.Value() < 1 {
		return fmt.Errorf("%w: aggregate version must be positive", ErrInvalidDelegation)
	}
	if !delegation.CreatedAt.IsZero() && !delegation.UpdatedAt.IsZero() && delegation.UpdatedAt.Before(delegation.CreatedAt) {
		return fmt.Errorf("%w: timestamps are not ordered", ErrInvalidDelegation)
	}
	return nil
}

func (delegation Delegation) ActiveAt(at time.Time) bool {
	at = at.UTC()
	return !at.Before(delegation.EffectiveFrom.UTC()) && at.Before(delegation.EffectiveTo.UTC())
}

type DelegateApprovalCommand struct {
	DelegationID   uuid.UUID
	DelegateUserID uuid.UUID
	PolicyID       uuid.UUID
	PolicyVersion  string
	Scope          ApprovalPolicyScope
	EffectiveFrom  time.Time
	EffectiveTo    time.Time
	IdempotencyKey string
	CorrelationID  string
	CausationID    string
}

func (command DelegateApprovalCommand) Validate() error {
	if command.DelegationID == uuid.Nil || command.DelegateUserID == uuid.Nil || command.PolicyID == uuid.Nil || strings.TrimSpace(command.PolicyVersion) == "" || strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: delegation identity, policy, delegate, and idempotency key are required", ErrInvalidDelegationCommand)
	}
	if err := command.Scope.Validate(); err != nil {
		return err
	}
	if command.EffectiveFrom.IsZero() || command.EffectiveTo.IsZero() || !command.EffectiveTo.After(command.EffectiveFrom) {
		return fmt.Errorf("%w: a finite effective interval is required", ErrInvalidDelegationCommand)
	}
	return nil
}

type DelegationAuditRecord struct {
	DelegationID                  uuid.UUID
	ActorUserID                   uuid.UUID
	ActorAuthenticationSubjectRef string
	DelegateUserID                uuid.UUID
	PolicyID                      uuid.UUID
	PolicyVersion                 string
	ScopeIDs                      []string
	Permission                    string
	DecisionReference             uuid.UUID
	EffectiveFrom                 time.Time
	EffectiveTo                   time.Time
	AfterFingerprint              string
	CorrelationID                 string
	CausationID                   string
}

type DelegationAuditRecorder interface {
	RecordDelegationMutation(context.Context, DelegationAuditRecord) (uuid.UUID, error)
}

type DelegationEvent struct {
	Type               string
	DelegationID       uuid.UUID
	AggregateVersion   int64
	DelegatorUserID    uuid.UUID
	DelegateUserID     uuid.UUID
	PolicyID           uuid.UUID
	PolicyVersion      string
	ScopeIDs           []string
	EffectiveFrom      time.Time
	EffectiveTo        time.Time
	DataClassification string
	OccurredAt         time.Time
	CorrelationID      string
	CausationID        string
}

type DelegationEventRecorder interface {
	RecordDelegationEvent(context.Context, DelegationEvent) error
}

type DelegationRepository interface {
	Get(context.Context, uuid.UUID) (Delegation, error)
	List(context.Context) ([]Delegation, error)
	Save(context.Context, Delegation, DelegationAuditRecord, DelegationEvent) error
}

type DurableDelegationServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableDelegationCommit struct {
	Coordinator   platformidempotency.DurableCoordinator
	Acquisition   platformidempotency.DurableAcquisition
	Result        platformidempotency.CommandResultMetadata
	ResultBuilder func(uuid.UUID) (platformidempotency.CommandResultMetadata, error)
}

type DurableDelegationRepository interface {
	SaveWithIdempotency(context.Context, Delegation, DelegationAuditRecord, DelegationEvent, DurableDelegationCommit) error
}

type DelegationCommitResultRepository interface {
	SaveWithResult(context.Context, Delegation, DelegationAuditRecord, DelegationEvent) (uuid.UUID, error)
}

type DelegationAuthorizer interface {
	AuthorizeApprovalDelegation(context.Context, Actor, DelegateApprovalCommand, ApprovalPolicy) (AuthorizationDecision, error)
}

type DelegationCommandResult struct {
	Delegation Delegation `json:"delegation"`
	Replayed   bool       `json:"replayed"`
}

type DelegationService struct {
	repository DelegationRepository
	policyRepo ApprovalPolicyRepository
	authorizer DelegationAuthorizer
	clock      func() time.Time
	durable    *DurableDelegationServiceConfig

	mu          sync.Mutex
	idempotency map[string]storedDelegationResult
}

type storedDelegationResult struct {
	fingerprint string
	result      DelegationCommandResult
}

func NewDelegationService(repository DelegationRepository, policyRepo ApprovalPolicyRepository, authorizer DelegationAuthorizer, clock func() time.Time) (*DelegationService, error) {
	return newDelegationService(repository, policyRepo, authorizer, clock, nil)
}

func NewDelegationServiceWithDurableIdempotency(repository DelegationRepository, policyRepo ApprovalPolicyRepository, authorizer DelegationAuthorizer, clock func() time.Time, durable DurableDelegationServiceConfig) (*DelegationService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidDelegationCommand
	}
	return newDelegationService(repository, policyRepo, authorizer, clock, &durable)
}

func newDelegationService(repository DelegationRepository, policyRepo ApprovalPolicyRepository, authorizer DelegationAuthorizer, clock func() time.Time, durable *DurableDelegationServiceConfig) (*DelegationService, error) {
	if repository == nil || policyRepo == nil || authorizer == nil || clock == nil {
		return nil, ErrInvalidDelegationCommand
	}
	return &DelegationService{repository: repository, policyRepo: policyRepo, authorizer: authorizer, clock: clock, durable: durable, idempotency: make(map[string]storedDelegationResult)}, nil
}

func (service *DelegationService) DelegateApproval(ctx context.Context, actor Actor, command DelegateApprovalCommand) (result DelegationCommandResult, returnErr error) {
	var acquisition platformidempotency.DurableAcquisition
	defer func() {
		if returnErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, returnErr)
		}
	}()
	if service == nil {
		return DelegationCommandResult{}, ErrInvalidDelegationCommand
	}
	if err := actor.Validate(); err != nil {
		return DelegationCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return DelegationCommandResult{}, err
	}
	commandFP := delegationCommandFingerprint(command)
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		identity, err := delegationDurableIdentity(actor, command.IdempotencyKey)
		if err != nil {
			return DelegationCommandResult{}, err
		}
		durableIdentity = identity
		var acquireErr error
		acquisition, acquireErr = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.OperationID, service.durable.Policy)
		if errors.Is(acquireErr, platformidempotency.ErrIdempotencyConflict) {
			return DelegationCommandResult{}, ErrDelegationIdempotencyConflict
		}
		if acquireErr != nil {
			return DelegationCommandResult{}, acquireErr
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			switch acquisition.Result().State() {
			case platformidempotency.StateFailed:
				return DelegationCommandResult{}, durableDelegationFailureError(acquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return DelegationCommandResult{}, ErrDelegationCommandInProgress
			case platformidempotency.StateEstablished:
				stored, decodeErr := decodeDelegationCommandResult(acquisition.Result().ResultBody())
				if decodeErr != nil {
					return DelegationCommandResult{}, decodeErr
				}
				stored.Replayed = true
				return stored, nil
			}
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != commandFP {
				return DelegationCommandResult{}, ErrDelegationIdempotencyConflict
			}
			result := cloneDelegationResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	policy, err := service.policyRepo.Get(ctx, command.PolicyID)
	if err != nil {
		if errors.Is(err, ErrApprovalPolicyNotFound) {
			return DelegationCommandResult{}, ErrDelegationPolicyConflict
		}
		return DelegationCommandResult{}, fmt.Errorf("%w: %v", ErrDelegationAuthorizationUnavailable, err)
	}
	selected, err := delegationPolicyVersion(policy, command.PolicyVersion)
	if err != nil {
		return DelegationCommandResult{}, err
	}
	now := service.clock().UTC()
	if err := validateDelegationAgainstPolicy(command, selected, now); err != nil {
		return DelegationCommandResult{}, err
	}
	authorization, err := service.authorizer.AuthorizeApprovalDelegation(ctx, actor, command, selected)
	if err != nil {
		return DelegationCommandResult{}, ErrDelegationAuthorizationUnavailable
	}
	if err := validateAuthorization(authorization, DelegateApprovalPermission, command.Scope.CanonicalScopeIDs()); err != nil {
		switch {
		case errors.Is(err, ErrApprovalPolicyAuthorizationUnavailable):
			return DelegationCommandResult{}, ErrDelegationAuthorizationUnavailable
		case errors.Is(err, ErrApprovalPolicyAuthorizationStale):
			return DelegationCommandResult{}, ErrDelegationAuthorizationStale
		default:
			return DelegationCommandResult{}, ErrDelegationAuthorizationDenied
		}
	}
	delegation := Delegation{ID: command.DelegationID, DelegatorUserID: actor.UserID, DelegateUserID: command.DelegateUserID, PolicyID: command.PolicyID, PolicyVersion: strings.TrimSpace(command.PolicyVersion), Scope: canonicalScope(command.Scope), EffectiveFrom: command.EffectiveFrom.UTC(), EffectiveTo: command.EffectiveTo.UTC(), Version: aggregateversion.Initial(), CorrelationID: command.CorrelationID, CausationID: command.CausationID, CreatedAt: now, UpdatedAt: now}
	if err := delegation.Validate(); err != nil {
		return DelegationCommandResult{}, err
	}
	audit := DelegationAuditRecord{DelegationID: delegation.ID, ActorUserID: actor.UserID, ActorAuthenticationSubjectRef: actor.SubjectReference, DelegateUserID: delegation.DelegateUserID, PolicyID: delegation.PolicyID, PolicyVersion: delegation.PolicyVersion, ScopeIDs: delegation.Scope.CanonicalScopeIDs(), Permission: authorization.Permission, DecisionReference: authorization.DecisionReference, EffectiveFrom: delegation.EffectiveFrom, EffectiveTo: delegation.EffectiveTo, AfterFingerprint: fingerprint(delegation), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	event := DelegationEvent{Type: ApprovalDelegatedEventType, DelegationID: delegation.ID, AggregateVersion: delegation.Version.Value(), DelegatorUserID: delegation.DelegatorUserID, DelegateUserID: delegation.DelegateUserID, PolicyID: delegation.PolicyID, PolicyVersion: delegation.PolicyVersion, ScopeIDs: delegation.Scope.CanonicalScopeIDs(), EffectiveFrom: delegation.EffectiveFrom, EffectiveTo: delegation.EffectiveTo, DataClassification: "internal", OccurredAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	result = DelegationCommandResult{Delegation: delegation}
	var auditReference uuid.UUID
	if service.durable != nil {
		committer, ok := service.repository.(DurableDelegationRepository)
		if !ok {
			return DelegationCommandResult{}, ErrInvalidDelegationCommand
		}
		aggregateID := delegation.ID
		resultForDurable := result
		resultBuilder := func(reference uuid.UUID) (platformidempotency.CommandResultMetadata, error) {
			resultForDurable.Delegation.AuditReference = reference
			body, marshalErr := json.Marshal(resultForDurable)
			if marshalErr != nil {
				return platformidempotency.CommandResultMetadata{}, marshalErr
			}
			status := 200
			return platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		}
		metadata, metadataErr := resultBuilder(uuid.Nil)
		if metadataErr != nil {
			return DelegationCommandResult{}, metadataErr
		}
		if err := committer.SaveWithIdempotency(ctx, delegation, audit, event, DurableDelegationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata, ResultBuilder: resultBuilder}); err != nil {
			return DelegationCommandResult{}, err
		}
		auditReference = resultForDurable.Delegation.AuditReference
	} else if committer, ok := service.repository.(DelegationCommitResultRepository); ok {
		auditReference, err = committer.SaveWithResult(ctx, delegation, audit, event)
		if err != nil {
			return DelegationCommandResult{}, err
		}
	} else if err := service.repository.Save(ctx, delegation, audit, event); err != nil {
		return DelegationCommandResult{}, err
	}
	result.Delegation.AuditReference = auditReference
	if service.durable == nil {
		service.idempotency[key] = storedDelegationResult{fingerprint: commandFP, result: cloneDelegationResult(result)}
	}
	return result, nil
}

func delegationPolicyVersion(policy ApprovalPolicy, version string) (ApprovalPolicy, error) {
	if policy.PolicyVersion == strings.TrimSpace(version) {
		return cloneApprovalPolicy(policy), nil
	}
	for _, revision := range policy.History {
		if revision.Snapshot.PolicyVersion == strings.TrimSpace(version) {
			return snapshotToPolicy(revision), nil
		}
	}
	return ApprovalPolicy{}, ErrDelegationPolicyConflict
}

func validateDelegationAgainstPolicy(command DelegateApprovalCommand, policy ApprovalPolicy, now time.Time) error {
	if policy.ID != command.PolicyID || policy.PolicyVersion != strings.TrimSpace(command.PolicyVersion) || policy.Status != ApprovalPolicyStatusActive {
		return ErrDelegationPolicyConflict
	}
	if now.UTC().Equal(command.EffectiveTo.UTC()) || now.UTC().After(command.EffectiveTo.UTC()) {
		return ErrDelegationExpired
	}
	if command.EffectiveFrom.UTC().Before(policy.EffectiveFrom.UTC()) || command.EffectiveTo.UTC().After(policy.EffectiveToOr(command.EffectiveTo)) {
		return ErrDelegationPolicyConflict
	}
	if !policy.Scope.Contains(command.Scope.CanonicalScopeIDs()) {
		return ErrDelegationScopeConflict
	}
	for _, step := range policy.Steps {
		if !step.Delegation.Enabled {
			continue
		}
		for _, requirement := range step.RequiredRoles {
			if requirement.AllowDelegation && len(step.Delegation.DelegateToRoles) > 0 {
				return nil
			}
		}
	}
	return ErrDelegationPolicyIncompatible
}

func (policy ApprovalPolicy) EffectiveToOr(fallback time.Time) time.Time {
	if policy.EffectiveTo == nil {
		return fallback
	}
	return policy.EffectiveTo.UTC()
}

func delegationCommandFingerprint(command DelegateApprovalCommand) string {
	command.IdempotencyKey = ""
	command.CorrelationID = ""
	command.CausationID = ""
	return commandFingerprint(command)
}

func delegationDurableIdentity(actor Actor, key string) (platformidempotency.IdempotencyIdentity, error) {
	value, err := json.Marshal(struct {
		Module    string    `json:"module"`
		ActorID   uuid.UUID `json:"actorId"`
		Operation string    `json:"operation"`
	}{Module: "workflow", ActorID: actor.UserID, Operation: "delegate-approval"})
	if err != nil {
		return platformidempotency.IdempotencyIdentity{}, err
	}
	return platformidempotency.NewOpaqueIdentity(string(value), key)
}

func (service *DelegationService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := durableDelegationFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{Code: code})
	if err != nil {
		return
	}
	metadata, err := platformidempotency.NewCommandResultMetadata(acquisition.Result().Identity(), acquisition.Result().Fingerprint(), acquisition.Result().OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	tx, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, tx, acquisition, metadata); err != nil {
		return
	}
	if err := tx.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func durableDelegationFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrDelegationAuthorizationDenied):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrDelegationAuthorizationUnavailable):
		return "AUTHORIZATION_UNAVAILABLE", true
	case errors.Is(err, ErrDelegationAuthorizationStale):
		return "AUTHORIZATION_STALE", true
	case errors.Is(err, ErrDelegationPolicyConflict):
		return "POLICY_CONFLICT", true
	case errors.Is(err, ErrDelegationScopeConflict):
		return "SCOPE_CONFLICT", true
	case errors.Is(err, ErrDelegationPolicyIncompatible):
		return "POLICY_INCOMPATIBLE", true
	case errors.Is(err, ErrDelegationExpired):
		return "EXPIRED", true
	case errors.Is(err, ErrDelegationDuplicate):
		return "DUPLICATE", true
	case errors.Is(err, ErrDelegationAuditUnavailable):
		return "AUDIT_UNAVAILABLE", true
	case errors.Is(err, ErrInvalidDelegation), errors.Is(err, ErrInvalidDelegationCommand):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func durableDelegationFailureError(body []byte) error {
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrDelegationCommandInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrDelegationAuthorizationDenied
	case "AUTHORIZATION_UNAVAILABLE":
		return ErrDelegationAuthorizationUnavailable
	case "AUTHORIZATION_STALE":
		return ErrDelegationAuthorizationStale
	case "POLICY_CONFLICT":
		return ErrDelegationPolicyConflict
	case "SCOPE_CONFLICT":
		return ErrDelegationScopeConflict
	case "POLICY_INCOMPATIBLE":
		return ErrDelegationPolicyIncompatible
	case "EXPIRED":
		return ErrDelegationExpired
	case "DUPLICATE":
		return ErrDelegationDuplicate
	case "AUDIT_UNAVAILABLE":
		return ErrDelegationAuditUnavailable
	case "VALIDATION_FAILED":
		return ErrDelegationDurableCommandFailed
	default:
		return ErrDelegationCommandInProgress
	}
}

func decodeDelegationCommandResult(body []byte) (DelegationCommandResult, error) {
	var result DelegationCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return DelegationCommandResult{}, err
	}
	return result, nil
}

func cloneDelegation(value Delegation) Delegation {
	value.Scope = cloneScope(value.Scope)
	return value
}

func cloneDelegationResult(result DelegationCommandResult) DelegationCommandResult {
	result.Delegation = cloneDelegation(result.Delegation)
	return result
}

// MemoryDelegationRepository is used by the in-memory API fixture and focused
// tests. Side effects are performed before the new value is published.
type MemoryDelegationRepository struct {
	mu          sync.RWMutex
	delegations map[uuid.UUID]Delegation
	audit       DelegationAuditRecorder
	events      DelegationEventRecorder
}

func NewMemoryDelegationRepository() *MemoryDelegationRepository {
	return &MemoryDelegationRepository{delegations: make(map[uuid.UUID]Delegation)}
}

func (repository *MemoryDelegationRepository) BindPorts(audit DelegationAuditRecorder, events DelegationEventRecorder) {
	repository.audit, repository.events = audit, events
}

func (repository *MemoryDelegationRepository) Get(_ context.Context, id uuid.UUID) (Delegation, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	delegation, ok := repository.delegations[id]
	if !ok {
		return Delegation{}, ErrDelegationNotFound
	}
	return cloneDelegation(delegation), nil
}

func (repository *MemoryDelegationRepository) List(_ context.Context) ([]Delegation, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]Delegation, 0, len(repository.delegations))
	for _, delegation := range repository.delegations {
		result = append(result, cloneDelegation(delegation))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].CreatedAt.After(result[right].CreatedAt) })
	return result, nil
}

func (repository *MemoryDelegationRepository) Save(ctx context.Context, delegation Delegation, audit DelegationAuditRecord, event DelegationEvent) error {
	_, err := repository.SaveWithResult(ctx, delegation, audit, event)
	return err
}

func (repository *MemoryDelegationRepository) SaveWithResult(ctx context.Context, delegation Delegation, audit DelegationAuditRecord, event DelegationEvent) (uuid.UUID, error) {
	if repository == nil || repository.audit == nil || repository.events == nil {
		return uuid.Nil, ErrDelegationAuditUnavailable
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.delegations[delegation.ID]; exists {
		return uuid.Nil, ErrDelegationDuplicate
	}
	auditReference, err := repository.audit.RecordDelegationMutation(ctx, audit)
	if err != nil {
		return uuid.Nil, err
	}
	if auditReference == uuid.Nil {
		return uuid.Nil, ErrDelegationAuditUnavailable
	}
	if err := repository.events.RecordDelegationEvent(ctx, event); err != nil {
		return uuid.Nil, err
	}
	delegation.AuditReference = auditReference
	repository.delegations[delegation.ID] = cloneDelegation(delegation)
	return auditReference, nil
}

type MemoryDelegationAuditRecorder struct {
	mu      sync.Mutex
	Records []DelegationAuditRecord
}

func (recorder *MemoryDelegationAuditRecorder) RecordDelegationMutation(_ context.Context, record DelegationAuditRecord) (uuid.UUID, error) {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	reference := uuid.New()
	recorder.Records = append(recorder.Records, record)
	return reference, nil
}

type MemoryDelegationEventRecorder struct {
	mu     sync.Mutex
	Events []DelegationEvent
}

func (recorder *MemoryDelegationEventRecorder) RecordDelegationEvent(_ context.Context, event DelegationEvent) error {
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	recorder.Events = append(recorder.Events, event)
	return nil
}

// PostgresDelegationRepository owns the workflow.delegation table and writes
// its audit and integration publication in the same local transaction.
type PostgresDelegationAuditWriter func(context.Context, pgx.Tx, DelegationAuditRecord) (uuid.UUID, error)
type PostgresDelegationEventWriter func(context.Context, pgx.Tx, DelegationEvent) error

type PostgresDelegationRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresDelegationAuditWriter
	eventWriter PostgresDelegationEventWriter
}

func NewPostgresDelegationRepository(pool *pgxpool.Pool, auditWriter PostgresDelegationAuditWriter, eventWriter PostgresDelegationEventWriter) (*PostgresDelegationRepository, error) {
	if pool == nil || auditWriter == nil || eventWriter == nil {
		return nil, ErrInvalidDelegationCommand
	}
	return &PostgresDelegationRepository{pool: pool, auditWriter: auditWriter, eventWriter: eventWriter}, nil
}

func (repository *PostgresDelegationRepository) Get(ctx context.Context, id uuid.UUID) (Delegation, error) {
	if repository == nil || repository.pool == nil || id == uuid.Nil {
		return Delegation{}, ErrInvalidDelegationCommand
	}
	var delegation Delegation
	var scope []byte
	var correlationID, causationID *uuid.UUID
	var version int64
	if err := repository.pool.QueryRow(ctx, `SELECT delegation_id, delegator_user_id, delegate_user_id, policy_id, policy_version, scope, effective_from, effective_to, aggregate_version, audit_reference, correlation_id, causation_id, created_at, updated_at FROM workflow.delegation WHERE delegation_id=$1`, id).Scan(&delegation.ID, &delegation.DelegatorUserID, &delegation.DelegateUserID, &delegation.PolicyID, &delegation.PolicyVersion, &scope, &delegation.EffectiveFrom, &delegation.EffectiveTo, &version, &delegation.AuditReference, &correlationID, &causationID, &delegation.CreatedAt, &delegation.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return Delegation{}, ErrDelegationNotFound
		}
		return Delegation{}, err
	}
	if err := json.Unmarshal(scope, &delegation.Scope); err != nil {
		return Delegation{}, err
	}
	delegation.Version = aggregateversion.AggregateVersion(version)
	delegation.CorrelationID = uuidString(correlationID)
	delegation.CausationID = uuidString(causationID)
	if err := delegation.Validate(); err != nil {
		return Delegation{}, err
	}
	return delegation, nil
}

func (repository *PostgresDelegationRepository) List(ctx context.Context) ([]Delegation, error) {
	if repository == nil || repository.pool == nil {
		return nil, ErrInvalidDelegationCommand
	}
	rows, err := repository.pool.Query(ctx, `SELECT delegation_id FROM workflow.delegation ORDER BY created_at DESC, delegation_id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Delegation, 0)
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		delegation, err := repository.Get(ctx, id)
		if err != nil {
			return nil, err
		}
		result = append(result, delegation)
	}
	return result, rows.Err()
}

func (repository *PostgresDelegationRepository) Save(ctx context.Context, delegation Delegation, audit DelegationAuditRecord, event DelegationEvent) error {
	_, err := repository.save(ctx, delegation, audit, event, nil)
	return err
}

func (repository *PostgresDelegationRepository) SaveWithResult(ctx context.Context, delegation Delegation, audit DelegationAuditRecord, event DelegationEvent) (uuid.UUID, error) {
	return repository.save(ctx, delegation, audit, event, nil)
}

func (repository *PostgresDelegationRepository) SaveWithIdempotency(ctx context.Context, delegation Delegation, audit DelegationAuditRecord, event DelegationEvent, durable DurableDelegationCommit) error {
	_, err := repository.save(ctx, delegation, audit, event, &durable)
	return err
}

func (repository *PostgresDelegationRepository) save(ctx context.Context, delegation Delegation, audit DelegationAuditRecord, event DelegationEvent, durable *DurableDelegationCommit) (uuid.UUID, error) {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil || repository.eventWriter == nil {
		return uuid.Nil, ErrDelegationAuditUnavailable
	}
	if err := delegation.Validate(); err != nil {
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
	auditReference, err := repository.auditWriter(ctx, tx, audit)
	if err != nil {
		return uuid.Nil, err
	}
	if auditReference == uuid.Nil {
		return uuid.Nil, ErrDelegationAuditUnavailable
	}
	scope, err := json.Marshal(delegation.Scope)
	if err != nil {
		return uuid.Nil, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO workflow.delegation (delegation_id, delegator_user_id, delegate_user_id, policy_id, policy_version, scope, effective_from, effective_to, aggregate_version, audit_reference, correlation_id, causation_id, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6::jsonb,$7,$8,$9,$10,$11,$12,$13,$14)`, delegation.ID, delegation.DelegatorUserID, delegation.DelegateUserID, delegation.PolicyID, delegation.PolicyVersion, scope, delegation.EffectiveFrom, delegation.EffectiveTo, delegation.Version.Value(), auditReference, nullableUUIDString(delegation.CorrelationID), nullableUUIDString(delegation.CausationID), delegation.CreatedAt, delegation.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return uuid.Nil, ErrDelegationDuplicate
		}
		return uuid.Nil, err
	}
	if err := repository.eventWriter(ctx, tx, event); err != nil {
		return uuid.Nil, err
	}
	if durable != nil {
		if durable.Coordinator == nil || durable.Acquisition.Decision() != platformidempotency.DecisionExecute {
			return uuid.Nil, platformidempotency.ErrInvalidAcquisition
		}
		result := durable.Result
		if durable.ResultBuilder != nil {
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

var _ DelegationRepository = (*MemoryDelegationRepository)(nil)
var _ DelegationRepository = (*PostgresDelegationRepository)(nil)
var _ DelegationCommitResultRepository = (*MemoryDelegationRepository)(nil)
var _ DelegationCommitResultRepository = (*PostgresDelegationRepository)(nil)
var _ DurableDelegationRepository = (*PostgresDelegationRepository)(nil)
