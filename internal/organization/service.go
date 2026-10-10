package organization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type Actor struct {
	UserID           uuid.UUID
	SubjectReference string
}

func (actor Actor) Validate() error {
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return ErrLegalEntityAuthorizationDenied
	}
	return nil
}

type LegalEntityCommand struct {
	Action               string
	LegalEntityID        uuid.UUID
	ScopeID              uuid.UUID
	LegalName            string
	FunctionalCurrency   string
	PresentationCurrency string
	TaxRegistrationID    string
	EffectiveFrom        time.Time
	EffectiveTo          *time.Time
	Registrations        []LegalEntityRegistration
	Addresses            []LegalEntityAddress
	OwnershipInterests   []LegalEntityOwnershipInterest
	Approval             *ApprovalDecisionReference
	ExpectedVersion      *aggregateversion.AggregateVersion
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command LegalEntityCommand) Validate() error {
	switch command.Action {
	case LegalEntityActionCreate:
		if command.LegalEntityID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include id or expected version", ErrInvalidLegalEntityCommand)
		}
		if command.ScopeID == uuid.Nil {
			return fmt.Errorf("%w: accounting scope is required", ErrInvalidLegalEntityCommand)
		}
		if err := validateCommandPayload(command); err != nil {
			return err
		}
	case LegalEntityActionMaintain:
		if command.LegalEntityID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: maintain id and expected version are required", ErrInvalidLegalEntityCommand)
		}
		if command.ScopeID == uuid.Nil {
			return fmt.Errorf("%w: accounting scope is required", ErrInvalidLegalEntityCommand)
		}
		if err := validateCommandPayload(command); err != nil {
			return err
		}
	case LegalEntityActionEndDate:
		if command.LegalEntityID == uuid.Nil || command.ExpectedVersion == nil || command.EffectiveTo == nil {
			return fmt.Errorf("%w: end-date id, expected version, and effectiveTo are required", ErrInvalidLegalEntityCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action", ErrInvalidLegalEntityCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidLegalEntityCommand)
	}
	return nil
}

func validateCommandPayload(command LegalEntityCommand) error {
	if command.EffectiveFrom.IsZero() || strings.TrimSpace(command.LegalName) == "" || len(command.Registrations) == 0 || len(command.Addresses) == 0 {
		return fmt.Errorf("%w: complete legal-entity payload is required", ErrInvalidLegalEntityCommand)
	}
	return nil
}

type AuthorizationDecision struct {
	Allowed           bool
	Outcome           string
	Permission        string
	ApprovedScopeIDs  []uuid.UUID
	PolicyReference   string
	PolicyVersion     string
	DecisionReference uuid.UUID
	ApprovalRequired  bool
	Reason            string
}

type Authorizer interface {
	AuthorizeLegalEntity(context.Context, Actor, LegalEntityCommand, *LegalEntity) (AuthorizationDecision, error)
	AuthorizeLegalEntityRead(context.Context, Actor, uuid.UUID) (AuthorizationDecision, error)
}
type ApprovalValidator interface {
	ValidateLegalEntityApproval(context.Context, LegalEntityCommand, LegalEntity) error
}
type AuditRecord struct {
	LegalEntityID         uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	ScopeID               uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	Approval              *ApprovalDecisionReference
	RevisionNumber        int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CorrelationID         string
	CausationID           string
}
type AuditRecorder interface {
	RecordLegalEntityMutation(context.Context, AuditRecord) error
}

type LegalEntityMutation struct {
	Before          LegalEntity
	After           LegalEntity
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           AuditRecord
}
type LegalEntityRepository interface {
	Get(context.Context, uuid.UUID) (LegalEntity, error)
	List(context.Context, *uuid.UUID) ([]LegalEntity, error)
	CommitLegalEntityMutation(context.Context, LegalEntityMutation) error
}
type DurableLegalEntityRepository interface {
	CommitLegalEntityMutationWithIdempotency(context.Context, LegalEntityMutation, DurableLegalEntityMutationCommit) error
}
type DurableLegalEntityMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}
type DurableLegalEntityServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type LegalEntityCommandResult struct {
	LegalEntity       SafeLegalEntity `json:"legalEntity"`
	DecisionReference uuid.UUID       `json:"decisionReference"`
	PolicyReference   string          `json:"policyReference"`
	Replayed          bool            `json:"replayed,omitempty"`
}

type LegalEntityService struct {
	repository  LegalEntityRepository
	authorizer  Authorizer
	approval    ApprovalValidator
	audit       AuditRecorder
	clock       func() time.Time
	durable     *DurableLegalEntityServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedLegalEntityCommand
}
type storedLegalEntityCommand struct {
	fingerprint string
	result      LegalEntityCommandResult
}

func NewLegalEntityService(repository LegalEntityRepository, authorizer Authorizer, approval ApprovalValidator, audit AuditRecorder, clock func() time.Time) (*LegalEntityService, error) {
	return newLegalEntityService(repository, authorizer, approval, audit, clock, nil)
}
func NewLegalEntityServiceWithDurableIdempotency(repository LegalEntityRepository, authorizer Authorizer, approval ApprovalValidator, audit AuditRecorder, clock func() time.Time, durable DurableLegalEntityServiceConfig) (*LegalEntityService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidLegalEntityService
	}
	return newLegalEntityService(repository, authorizer, approval, audit, clock, &durable)
}
func newLegalEntityService(repository LegalEntityRepository, authorizer Authorizer, approval ApprovalValidator, audit AuditRecorder, clock func() time.Time, durable *DurableLegalEntityServiceConfig) (*LegalEntityService, error) {
	if repository == nil || authorizer == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidLegalEntityService
	}
	if binder, ok := repository.(interface{ BindAuditRecorder(AuditRecorder) }); ok {
		binder.BindAuditRecorder(audit)
	}
	return &LegalEntityService{repository: repository, authorizer: authorizer, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedLegalEntityCommand)}, nil
}

func (service *LegalEntityService) Execute(ctx context.Context, actor Actor, command LegalEntityCommand) (LegalEntityCommandResult, error) {
	if service == nil {
		return LegalEntityCommandResult{}, ErrInvalidLegalEntityService
	}
	if err := actor.Validate(); err != nil {
		return LegalEntityCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return LegalEntityCommandResult{}, err
	}
	fingerprint, err := legalEntityCommandFingerprint(command)
	if err != nil {
		return LegalEntityCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scope, marshalErr := json.Marshal(struct {
			Module string    `json:"module"`
			Actor  uuid.UUID `json:"actorId"`
		}{"organization", actor.UserID})
		if marshalErr != nil {
			return LegalEntityCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return LegalEntityCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return LegalEntityCommandResult{}, ErrLegalEntityIdempotencyConflict
		}
		if err != nil {
			return LegalEntityCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return LegalEntityCommandResult{}, ErrLegalEntityCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return LegalEntityCommandResult{}, ErrLegalEntityDurableCommandFailed
			}
			result, decodeErr := decodeDurableLegalEntityResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return LegalEntityCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return LegalEntityCommandResult{}, ErrLegalEntityIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	var current *LegalEntity
	if command.Action != LegalEntityActionCreate {
		loaded, getErr := service.repository.Get(ctx, command.LegalEntityID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return LegalEntityCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrLegalEntityVersionConflict)
			return LegalEntityCommandResult{}, ErrLegalEntityVersionConflict
		}
		if command.ScopeID != uuid.Nil && current.ScopeID != command.ScopeID {
			service.finalizeDurableFailure(ctx, acquisition, ErrLegalEntityAuthorizationDenied)
			return LegalEntityCommandResult{}, ErrLegalEntityAuthorizationDenied
		}
	}
	decision, err := service.authorizer.AuthorizeLegalEntity(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return LegalEntityCommandResult{}, err
	}
	if err := authorizeLegalEntityDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return LegalEntityCommandResult{}, err
	}
	now := service.clock().UTC()
	var before, after LegalEntity
	switch command.Action {
	case LegalEntityActionCreate:
		before = LegalEntity{}
		status := LegalEntityStatusActive
		if decision.ApprovalRequired {
			status = LegalEntityStatusDraft
		}
		after, err = NewLegalEntity(uuid.New(), command.ScopeID, command.LegalName, command.FunctionalCurrency, command.PresentationCurrency, command.TaxRegistrationID, command.EffectiveFrom, newChildIDs(command.Registrations), newAddressIDs(command.Addresses), newOwnershipIDs(command.OwnershipInterests), command.Approval, status, now)
	case LegalEntityActionMaintain:
		before = cloneLegalEntity(*current)
		after = cloneLegalEntity(*current)
		err = after.Replace(*current, command.LegalName, command.FunctionalCurrency, command.PresentationCurrency, command.TaxRegistrationID, command.EffectiveFrom, command.EffectiveTo, newChildIDs(command.Registrations), newAddressIDs(command.Addresses), newOwnershipIDs(command.OwnershipInterests), command.Approval, now)
	case LegalEntityActionEndDate:
		before = cloneLegalEntity(*current)
		after = cloneLegalEntity(*current)
		err = after.EndDate(*current, *command.EffectiveTo, now)
	}
	if err == nil && decision.ApprovalRequired {
		if command.Approval == nil {
			err = ErrLegalEntityApprovalRequired
		} else {
			err = service.approval.ValidateLegalEntityApproval(ctx, command, after)
		}
	} else if err == nil && command.Approval != nil {
		err = service.approval.ValidateLegalEntityApproval(ctx, command, after)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return LegalEntityCommandResult{}, err
	}
	record := AuditRecord{LegalEntityID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, ScopeID: after.ScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Approval: cloneApproval(command.Approval), RevisionNumber: after.RevisionNumber, BeforeFingerprint: Fingerprint(before), AfterFingerprint: Fingerprint(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	result := LegalEntityCommandResult{LegalEntity: after.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference}
	mutation := LegalEntityMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableLegalEntityRepository)
		if !ok {
			return LegalEntityCommandResult{}, ErrInvalidLegalEntityService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return LegalEntityCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			return LegalEntityCommandResult{}, metadataErr
		}
		err = committer.CommitLegalEntityMutationWithIdempotency(ctx, mutation, DurableLegalEntityMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitLegalEntityMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return LegalEntityCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedLegalEntityCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func (service *LegalEntityService) ListSafe(ctx context.Context, actor Actor, scopeID *uuid.UUID) ([]SafeLegalEntity, error) {
	if service == nil {
		return nil, ErrInvalidLegalEntityService
	}
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	entities, err := service.repository.List(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeLegalEntity, 0, len(entities))
	for _, entity := range entities {
		decision, decisionErr := service.authorizer.AuthorizeLegalEntityRead(ctx, actor, entity.ScopeID)
		if decisionErr != nil {
			return nil, decisionErr
		}
		if err := authorizeLegalEntityReadDecision(decision, entity.ScopeID); err != nil {
			continue
		}
		result = append(result, entity.SafeProjection())
	}
	return result, nil
}

func (service *LegalEntityService) GetSafe(ctx context.Context, actor Actor, id uuid.UUID, scopeID *uuid.UUID) (SafeLegalEntity, error) {
	if service == nil {
		return SafeLegalEntity{}, ErrInvalidLegalEntityService
	}
	if err := actor.Validate(); err != nil {
		return SafeLegalEntity{}, err
	}
	entity, err := service.repository.Get(ctx, id)
	if err != nil {
		return SafeLegalEntity{}, err
	}
	if scopeID != nil && entity.ScopeID != *scopeID {
		return SafeLegalEntity{}, ErrLegalEntityAuthorizationDenied
	}
	decision, err := service.authorizer.AuthorizeLegalEntityRead(ctx, actor, entity.ScopeID)
	if err != nil {
		return SafeLegalEntity{}, err
	}
	if err := authorizeLegalEntityReadDecision(decision, entity.ScopeID); err != nil {
		return SafeLegalEntity{}, err
	}
	return entity.SafeProjection(), nil
}

// GetReference is the OMD application boundary for downstream authoritative
// identity checks. The downstream command must establish actor authorization;
// this method only returns the scoped identity and never exposes the full
// LegalEntity aggregate or reads the OMD schema outside its owning module.
func (service *LegalEntityService) GetReference(ctx context.Context, id, scopeID uuid.UUID) (LegalEntityReference, error) {
	if service == nil {
		return LegalEntityReference{}, ErrInvalidLegalEntityService
	}
	if id == uuid.Nil || scopeID == uuid.Nil {
		return LegalEntityReference{}, ErrLegalEntityNotFound
	}
	entity, err := service.repository.Get(ctx, id)
	if err != nil {
		return LegalEntityReference{}, err
	}
	if entity.ScopeID != scopeID {
		return LegalEntityReference{}, ErrLegalEntityNotFound
	}
	return LegalEntityReference{ID: entity.ID, ScopeID: entity.ScopeID}, nil
}

func authorizeLegalEntityDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != LegalEntityManagementPermission {
		return ErrLegalEntityAuthorizationDenied
	}
	if decision.DecisionReference == uuid.Nil {
		return ErrLegalEntityAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrLegalEntityAuthorizationDenied
}
func authorizeLegalEntityReadDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != LegalEntityReadPermission || decision.DecisionReference == uuid.Nil {
		return ErrLegalEntityAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrLegalEntityAuthorizationDenied
}
func legalEntityCommandFingerprint(command LegalEntityCommand) (string, error) {
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
func newChildIDs(values []LegalEntityRegistration) []LegalEntityRegistration {
	result := append([]LegalEntityRegistration(nil), values...)
	for index := range result {
		if result[index].ID == uuid.Nil {
			result[index].ID = uuid.New()
		}
	}
	return result
}
func newAddressIDs(values []LegalEntityAddress) []LegalEntityAddress {
	result := append([]LegalEntityAddress(nil), values...)
	for index := range result {
		if result[index].ID == uuid.Nil {
			result[index].ID = uuid.New()
		}
	}
	return result
}
func newOwnershipIDs(values []LegalEntityOwnershipInterest) []LegalEntityOwnershipInterest {
	result := append([]LegalEntityOwnershipInterest(nil), values...)
	for index := range result {
		if result[index].ID == uuid.Nil {
			result[index].ID = uuid.New()
		}
	}
	return result
}

func (service *LegalEntityService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrLegalEntityAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrLegalEntityVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrLegalEntityNotFound):
		code = "LEGAL_ENTITY_NOT_FOUND"
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{code})
	if err != nil {
		return
	}
	initial := acquisition.Result()
	metadata, err := platformidempotency.NewCommandResultMetadata(initial.Identity(), initial.Fingerprint(), initial.OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
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
func decodeDurableLegalEntityResult(body []byte) (LegalEntityCommandResult, error) {
	var result LegalEntityCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return LegalEntityCommandResult{}, ErrLegalEntityCommandInProgress
	}
	return result, nil
}

type MemoryLegalEntityRepository struct {
	mu       sync.RWMutex
	entities map[uuid.UUID]LegalEntity
	audit    AuditRecorder
}

func NewMemoryLegalEntityRepository() *MemoryLegalEntityRepository {
	return &MemoryLegalEntityRepository{entities: make(map[uuid.UUID]LegalEntity)}
}
func (repository *MemoryLegalEntityRepository) BindAuditRecorder(audit AuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}
func (repository *MemoryLegalEntityRepository) Get(_ context.Context, id uuid.UUID) (LegalEntity, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	entity, ok := repository.entities[id]
	if !ok {
		return LegalEntity{}, ErrLegalEntityNotFound
	}
	return cloneLegalEntity(entity), nil
}
func (repository *MemoryLegalEntityRepository) List(_ context.Context, scopeID *uuid.UUID) ([]LegalEntity, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]LegalEntity, 0, len(repository.entities))
	for _, entity := range repository.entities {
		if scopeID != nil && entity.ScopeID != *scopeID {
			continue
		}
		result = append(result, cloneLegalEntity(entity))
	}
	sortLegalEntities(result)
	return result, nil
}
func (repository *MemoryLegalEntityRepository) CommitLegalEntityMutation(ctx context.Context, mutation LegalEntityMutation) error {
	if repository == nil {
		return ErrInvalidLegalEntityService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrLegalEntityVersionConflict
		}
		for _, current := range repository.entities {
			if current.ScopeID == mutation.After.ScopeID && strings.EqualFold(current.LegalName, mutation.After.LegalName) && current.Status != LegalEntityStatusEndDated {
				return ErrLegalEntityDuplicate
			}
		}
	} else {
		if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
			return ErrLegalEntityVersionConflict
		}
		current, ok := repository.entities[mutation.After.ID]
		if !ok {
			return ErrLegalEntityNotFound
		}
		if mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrLegalEntityVersionConflict
		}
	}
	if repository.audit == nil {
		return ErrLegalEntityAuditUnavailable
	}
	if err := repository.audit.RecordLegalEntityMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.entities[mutation.After.ID] = cloneLegalEntity(mutation.After)
	return nil
}

type MemoryAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryAuthorizer) AuthorizeLegalEntity(context.Context, Actor, LegalEntityCommand, *LegalEntity) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}
func (authorizer MemoryAuthorizer) AuthorizeLegalEntityRead(context.Context, Actor, uuid.UUID) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	decision := authorizer.Decision
	if decision.Permission == LegalEntityManagementPermission {
		decision.Permission = LegalEntityReadPermission
	}
	return decision, nil
}

type AllowAllApprovalValidator struct{}

func (AllowAllApprovalValidator) ValidateLegalEntityApproval(context.Context, LegalEntityCommand, LegalEntity) error {
	return nil
}

type MemoryAuditRecorder struct {
	mu      sync.Mutex
	Records []AuditRecord
	Err     error
}

func (recorder *MemoryAuditRecorder) RecordLegalEntityMutation(_ context.Context, record AuditRecord) error {
	if recorder == nil {
		return ErrLegalEntityAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}
