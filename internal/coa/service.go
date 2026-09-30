package coa

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type SegmentDefinitionService struct {
	repository  SegmentDefinitionRepository
	authorizer  Authorizer
	audit       AuditRecorder
	clock       func() time.Time
	durable     *DurableSegmentDefinitionServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSegmentDefinitionCommand
}

type storedSegmentDefinitionCommand struct {
	fingerprint string
	result      SegmentDefinitionCommandResult
}

func NewSegmentDefinitionService(repository SegmentDefinitionRepository, authorizer Authorizer, audit AuditRecorder, clock func() time.Time) (*SegmentDefinitionService, error) {
	return newSegmentDefinitionService(repository, authorizer, audit, clock, nil)
}

func NewSegmentDefinitionServiceWithDurableIdempotency(repository SegmentDefinitionRepository, authorizer Authorizer, audit AuditRecorder, clock func() time.Time, durable DurableSegmentDefinitionServiceConfig) (*SegmentDefinitionService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidSegmentDefinitionService
	}
	return newSegmentDefinitionService(repository, authorizer, audit, clock, &durable)
}

func newSegmentDefinitionService(repository SegmentDefinitionRepository, authorizer Authorizer, audit AuditRecorder, clock func() time.Time, durable *DurableSegmentDefinitionServiceConfig) (*SegmentDefinitionService, error) {
	if repository == nil || authorizer == nil || audit == nil || clock == nil {
		return nil, ErrInvalidSegmentDefinitionService
	}
	if binder, ok := repository.(interface{ BindAuditRecorder(AuditRecorder) }); ok {
		binder.BindAuditRecorder(audit)
	}
	return &SegmentDefinitionService{
		repository:  repository,
		authorizer:  authorizer,
		audit:       audit,
		clock:       clock,
		durable:     durable,
		idempotency: make(map[string]storedSegmentDefinitionCommand),
	}, nil
}

func (service *SegmentDefinitionService) Execute(ctx context.Context, actor Actor, command SegmentDefinitionCommand) (SegmentDefinitionCommandResult, error) {
	if service == nil {
		return SegmentDefinitionCommandResult{}, ErrInvalidSegmentDefinitionService
	}
	if err := actor.Validate(); err != nil {
		return SegmentDefinitionCommandResult{}, err
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SegmentDefinitionCommandResult{}, err
	}
	fingerprint, err := segmentDefinitionCommandFingerprint(command)
	if err != nil {
		return SegmentDefinitionCommandResult{}, err
	}

	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scope, marshalErr := json.Marshal(struct {
			Module string    `json:"module"`
			Actor  uuid.UUID `json:"actorId"`
			Scope  uuid.UUID `json:"scopeId"`
		}{"coa", actor.UserID, command.ScopeID})
		if marshalErr != nil {
			return SegmentDefinitionCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return SegmentDefinitionCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionIdempotencyConflict
		}
		if err != nil {
			return SegmentDefinitionCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionDurableCommandFailed
			}
			result, decodeErr := decodeDurableSegmentDefinitionResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return SegmentDefinitionCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	var current *SegmentDefinition
	if command.Action == SegmentDefinitionActionUpdate {
		loaded, getErr := service.repository.Get(ctx, command.SegmentDefinitionID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return SegmentDefinitionCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrSegmentDefinitionVersionConflict)
			return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionVersionConflict
		}
		if current.ScopeID != command.ScopeID {
			service.finalizeDurableFailure(ctx, acquisition, ErrSegmentDefinitionAuthorizationDenied)
			return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionAuthorizationDenied
		}
	}

	decision, err := service.authorizer.AuthorizeSegmentDefinition(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentDefinitionCommandResult{}, err
	}
	if err := authorizeSegmentDefinitionDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentDefinitionCommandResult{}, err
	}

	now := service.clock().UTC()
	var before, after SegmentDefinition
	switch command.Action {
	case SegmentDefinitionActionCreate:
		before = SegmentDefinition{}
		after, err = NewSegmentDefinition(uuid.New(), command.ScopeID, command.SegmentType, command.Code, command.Name, command.Status, command.EffectiveDateFrom, command.EffectiveDateTo, now)
	case SegmentDefinitionActionUpdate:
		before = cloneSegmentDefinition(*current)
		after = cloneSegmentDefinition(*current)
		err = after.Replace(*current, command.SegmentType, command.Code, command.Name, command.Status, command.EffectiveDateFrom, command.EffectiveDateTo, now)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentDefinitionCommandResult{}, err
	}

	record := AuditRecord{
		SegmentDefinitionID:   after.ID,
		ActorUserID:           actor.UserID,
		ActorSubjectReference: actor.SubjectReference,
		Action:                command.Action,
		ScopeID:               after.ScopeID,
		Permission:            decision.Permission,
		PolicyReference:       decision.PolicyReference,
		PolicyVersion:         decision.PolicyVersion,
		DecisionReference:     decision.DecisionReference,
		RevisionNumber:        after.RevisionNumber,
		BeforeFingerprint:     FingerprintSegmentDefinition(before),
		AfterFingerprint:      FingerprintSegmentDefinition(after),
		CorrelationID:         command.CorrelationID,
		CausationID:           command.CausationID,
	}
	result := SegmentDefinitionCommandResult{
		SegmentDefinition: after.SafeProjection(),
		DecisionReference: decision.DecisionReference,
		PolicyReference:   decision.PolicyReference,
		ValidationOutcome: "valid",
	}
	mutation := SegmentDefinitionMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}

	if service.durable != nil {
		committer, ok := service.repository.(DurableSegmentDefinitionRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidSegmentDefinitionService)
			return SegmentDefinitionCommandResult{}, ErrInvalidSegmentDefinitionService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return SegmentDefinitionCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return SegmentDefinitionCommandResult{}, metadataErr
		}
		err = committer.CommitSegmentDefinitionMutationWithIdempotency(ctx, mutation, DurableSegmentDefinitionMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitSegmentDefinitionMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentDefinitionCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedSegmentDefinitionCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func (service *SegmentDefinitionService) ListSafe(ctx context.Context, scopeID *uuid.UUID) ([]SafeSegmentDefinition, error) {
	if service == nil {
		return nil, ErrInvalidSegmentDefinitionService
	}
	values, err := service.repository.List(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeSegmentDefinition, 0, len(values))
	for _, value := range values {
		result = append(result, value.SafeProjection())
	}
	return result, nil
}

func (service *SegmentDefinitionService) GetSafe(ctx context.Context, id uuid.UUID, scopeID *uuid.UUID) (SafeSegmentDefinition, error) {
	if service == nil {
		return SafeSegmentDefinition{}, ErrInvalidSegmentDefinitionService
	}
	value, err := service.repository.Get(ctx, id)
	if err != nil {
		return SafeSegmentDefinition{}, err
	}
	if scopeID != nil && value.ScopeID != *scopeID {
		return SafeSegmentDefinition{}, ErrSegmentDefinitionAuthorizationDenied
	}
	return value.SafeProjection(), nil
}

func authorizeSegmentDefinitionDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != SegmentDefinitionManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrSegmentDefinitionAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrSegmentDefinitionAuthorizationDenied
}

func (service *SegmentDefinitionService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrSegmentDefinitionAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrSegmentDefinitionVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrSegmentDefinitionNotFound):
		code = "SEGMENT_DEFINITION_NOT_FOUND"
	case errors.Is(commandErr, ErrSegmentDefinitionDuplicate):
		code = "DUPLICATE_SEGMENT_DEFINITION"
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

func decodeDurableSegmentDefinitionResult(body []byte) (SegmentDefinitionCommandResult, error) {
	var result SegmentDefinitionCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SegmentDefinitionCommandResult{}, ErrSegmentDefinitionCommandInProgress
	}
	return result, nil
}

type MemorySegmentDefinitionRepository struct {
	mu          sync.RWMutex
	definitions map[uuid.UUID]SegmentDefinition
	audit       AuditRecorder
}

func NewMemorySegmentDefinitionRepository() *MemorySegmentDefinitionRepository {
	return &MemorySegmentDefinitionRepository{definitions: make(map[uuid.UUID]SegmentDefinition)}
}

func (repository *MemorySegmentDefinitionRepository) BindAuditRecorder(audit AuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemorySegmentDefinitionRepository) Get(_ context.Context, id uuid.UUID) (SegmentDefinition, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	definition, ok := repository.definitions[id]
	if !ok {
		return SegmentDefinition{}, ErrSegmentDefinitionNotFound
	}
	return cloneSegmentDefinition(definition), nil
}

func (repository *MemorySegmentDefinitionRepository) List(_ context.Context, scopeID *uuid.UUID) ([]SegmentDefinition, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]SegmentDefinition, 0, len(repository.definitions))
	for _, definition := range repository.definitions {
		if scopeID != nil && definition.ScopeID != *scopeID {
			continue
		}
		result = append(result, cloneSegmentDefinition(definition))
	}
	sortSegmentDefinitions(result)
	return result, nil
}

func (repository *MemorySegmentDefinitionRepository) CommitSegmentDefinitionMutation(ctx context.Context, mutation SegmentDefinitionMutation) error {
	if repository == nil {
		return ErrInvalidSegmentDefinitionService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 {
			return ErrSegmentDefinitionVersionConflict
		}
	} else {
		if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
			return ErrSegmentDefinitionVersionConflict
		}
		current, ok := repository.definitions[mutation.After.ID]
		if !ok {
			return ErrSegmentDefinitionNotFound
		}
		if !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrSegmentDefinitionVersionConflict
		}
	}
	for id, current := range repository.definitions {
		if id == mutation.After.ID || current.ScopeID != mutation.After.ScopeID || current.SegmentType != mutation.After.SegmentType || current.Code != mutation.After.Code {
			continue
		}
		if rangesOverlap(current.EffectiveDateFrom, current.EffectiveDateTo, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo) {
			return ErrSegmentDefinitionDuplicate
		}
	}
	if repository.audit == nil {
		return ErrSegmentDefinitionAuditUnavailable
	}
	if err := repository.audit.RecordSegmentDefinitionMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.definitions[mutation.After.ID] = cloneSegmentDefinition(mutation.After)
	return nil
}

type MemoryAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryAuthorizer) AuthorizeSegmentDefinition(context.Context, Actor, SegmentDefinitionCommand, *SegmentDefinition) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryAuditRecorder struct {
	mu      sync.Mutex
	Records []AuditRecord
	Err     error
}

func (recorder *MemoryAuditRecorder) RecordSegmentDefinitionMutation(_ context.Context, record AuditRecord) error {
	if recorder == nil {
		return ErrSegmentDefinitionAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func cloneSegmentDefinition(definition SegmentDefinition) SegmentDefinition {
	definition.EffectiveDateFrom = dateOnly(definition.EffectiveDateFrom)
	definition.EffectiveDateTo = cloneDate(definition.EffectiveDateTo)
	definition.Revisions = cloneRevisions(definition.Revisions)
	return definition
}
