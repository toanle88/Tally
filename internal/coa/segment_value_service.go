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

type SegmentValueService struct {
	repository  SegmentValueRepository
	authorizer  SegmentValueAuthorizer
	audit       SegmentValueAuditRecorder
	clock       func() time.Time
	durable     *DurableSegmentValueServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSegmentValueCommand
}

type storedSegmentValueCommand struct {
	fingerprint string
	result      SegmentValueCommandResult
}

func NewSegmentValueService(repository SegmentValueRepository, authorizer SegmentValueAuthorizer, audit SegmentValueAuditRecorder, clock func() time.Time) (*SegmentValueService, error) {
	return newSegmentValueService(repository, authorizer, audit, clock, nil)
}

func NewSegmentValueServiceWithDurableIdempotency(repository SegmentValueRepository, authorizer SegmentValueAuthorizer, audit SegmentValueAuditRecorder, clock func() time.Time, durable DurableSegmentValueServiceConfig) (*SegmentValueService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidSegmentValueService
	}
	return newSegmentValueService(repository, authorizer, audit, clock, &durable)
}

func newSegmentValueService(repository SegmentValueRepository, authorizer SegmentValueAuthorizer, audit SegmentValueAuditRecorder, clock func() time.Time, durable *DurableSegmentValueServiceConfig) (*SegmentValueService, error) {
	if repository == nil || authorizer == nil || audit == nil || clock == nil {
		return nil, ErrInvalidSegmentValueService
	}
	if binder, ok := repository.(interface {
		BindSegmentValueAuditRecorder(SegmentValueAuditRecorder)
	}); ok {
		binder.BindSegmentValueAuditRecorder(audit)
	}
	return &SegmentValueService{
		repository:  repository,
		authorizer:  authorizer,
		audit:       audit,
		clock:       clock,
		durable:     durable,
		idempotency: make(map[string]storedSegmentValueCommand),
	}, nil
}

func (service *SegmentValueService) Execute(ctx context.Context, actor Actor, command SegmentValueCommand) (SegmentValueCommandResult, error) {
	if service == nil {
		return SegmentValueCommandResult{}, ErrInvalidSegmentValueService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return SegmentValueCommandResult{}, ErrSegmentValueAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SegmentValueCommandResult{}, err
	}
	fingerprint, err := segmentValueCommandFingerprint(command)
	if err != nil {
		return SegmentValueCommandResult{}, err
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
			return SegmentValueCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return SegmentValueCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return SegmentValueCommandResult{}, ErrSegmentValueIdempotencyConflict
		}
		if err != nil {
			return SegmentValueCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return SegmentValueCommandResult{}, ErrSegmentValueCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return SegmentValueCommandResult{}, ErrSegmentValueDurableCommandFailed
			}
			result, decodeErr := decodeDurableSegmentValueResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return SegmentValueCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return SegmentValueCommandResult{}, ErrSegmentValueIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	parent, err := service.repository.Get(ctx, command.SegmentDefinitionID)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	if parent.ScopeID != command.ScopeID || parent.Status == SegmentStatusRetired {
		err = ErrSegmentValueAuthorizationDenied
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	if command.ExpectedVersion == nil || !parent.Version.Matches(*command.ExpectedVersion) {
		err = ErrSegmentValueVersionConflict
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}

	var current *SegmentValue
	if command.Action == SegmentValueActionUpdate {
		for index := range parent.Values {
			if parent.Values[index].ID == command.SegmentValueID {
				value := parent.Values[index]
				current = &value
				break
			}
		}
		if current == nil {
			err = ErrSegmentValueNotFound
			service.finalizeDurableFailure(ctx, acquisition, err)
			return SegmentValueCommandResult{}, err
		}
	}

	decision, err := service.authorizer.AuthorizeSegmentValue(ctx, actor, command, &parent, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	if err := authorizeSegmentValueDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}

	now := service.clock().UTC()
	before := cloneSegmentDefinition(parent)
	after := cloneSegmentDefinition(parent)
	nextVersion, err := parent.Version.Advance()
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	after.Version = nextVersion
	after.RevisionNumber = parent.RevisionNumber + 1
	after.UpdatedAt = now

	var valueAfter SegmentValue
	var valueBefore *SegmentValue
	switch command.Action {
	case SegmentValueActionCreate:
		valueAfter, err = NewSegmentValue(parent, uuid.New(), command.Value, command.Description, command.Status, command.EffectiveDateFrom, command.EffectiveDateTo, now)
	case SegmentValueActionUpdate:
		beforeValue := cloneSegmentValue(*current)
		valueBefore = &beforeValue
		valueAfter = cloneSegmentValue(*current)
		err = valueAfter.Replace(*current, parent, command.Value, command.Description, command.Status, command.EffectiveDateFrom, command.EffectiveDateTo, now)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	if command.Action == SegmentValueActionCreate {
		after.Values = append(after.Values, valueAfter)
	} else {
		for index := range after.Values {
			if after.Values[index].ID == valueAfter.ID {
				after.Values[index] = valueAfter
				break
			}
		}
	}
	sortSegmentValues(after.Values)
	if err := after.Validate(); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	after.Revisions = append(cloneRevisions(parent.Revisions), SegmentDefinitionRevision{
		RevisionNumber: after.RevisionNumber,
		Version:        after.Version,
		Snapshot:       after.Snapshot(),
		CreatedAt:      after.UpdatedAt,
	})

	record := AuditRecord{
		SegmentDefinitionID:   after.ID,
		SegmentValueID:        valueAfter.ID,
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
	result := SegmentValueCommandResult{
		SegmentValue:      valueAfter.SafeProjection(after),
		DecisionReference: decision.DecisionReference,
		PolicyReference:   decision.PolicyReference,
		ValidationOutcome: "valid",
	}
	mutation := SegmentValueMutation{Before: before, After: after, ValueBefore: valueBefore, ValueAfter: valueAfter, ExpectedVersion: command.ExpectedVersion, Audit: record}

	if service.durable != nil {
		committer, ok := service.repository.(DurableSegmentValueRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidSegmentValueService)
			return SegmentValueCommandResult{}, ErrInvalidSegmentValueService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return SegmentValueCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return SegmentValueCommandResult{}, metadataErr
		}
		err = committer.CommitSegmentValueMutationWithIdempotency(ctx, mutation, DurableSegmentValueMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitSegmentValueMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentValueCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedSegmentValueCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func authorizeSegmentValueDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != SegmentValueManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrSegmentValueAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrSegmentValueAuthorizationDenied
}

func (service *SegmentValueService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrSegmentValueAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrSegmentValueVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrSegmentDefinitionNotFound):
		code = "SEGMENT_DEFINITION_NOT_FOUND"
	case errors.Is(commandErr, ErrSegmentValueNotFound):
		code = "SEGMENT_VALUE_NOT_FOUND"
	case errors.Is(commandErr, ErrSegmentValueDuplicate):
		code = "DUPLICATE_SEGMENT_VALUE"
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

func decodeDurableSegmentValueResult(body []byte) (SegmentValueCommandResult, error) {
	var result SegmentValueCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SegmentValueCommandResult{}, ErrSegmentValueCommandInProgress
	}
	return result, nil
}

func validateSegmentValueMutation(mutation SegmentValueMutation) error {
	if mutation.Before.ID == uuid.Nil || mutation.After.ID == uuid.Nil || mutation.Before.ID != mutation.After.ID || mutation.ExpectedVersion == nil {
		return ErrSegmentValueVersionConflict
	}
	if mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrSegmentValueVersionConflict
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if err := mutation.ValueAfter.validate(mutation.After); err != nil {
		return err
	}
	if mutation.ValueBefore != nil && mutation.ValueBefore.ID != mutation.ValueAfter.ID {
		return ErrSegmentValueVersionConflict
	}
	return nil
}

func (repository *MemorySegmentDefinitionRepository) BindSegmentValueAuditRecorder(audit SegmentValueAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.valueAudit = audit
}

func (repository *MemorySegmentDefinitionRepository) CommitSegmentValueMutation(ctx context.Context, mutation SegmentValueMutation) error {
	if repository == nil {
		return ErrInvalidSegmentValueService
	}
	if err := validateSegmentValueMutation(mutation); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, ok := repository.definitions[mutation.After.ID]
	if !ok {
		return ErrSegmentDefinitionNotFound
	}
	if !current.Version.Matches(*mutation.ExpectedVersion) {
		return ErrSegmentValueVersionConflict
	}
	if current.ScopeID != mutation.After.ScopeID || current.RevisionNumber != mutation.Before.RevisionNumber {
		return ErrSegmentValueVersionConflict
	}
	if mutation.ValueBefore == nil {
		for _, value := range current.Values {
			if value.ID == mutation.ValueAfter.ID {
				return ErrSegmentValueDuplicate
			}
		}
	} else {
		var currentValue *SegmentValue
		for _, value := range current.Values {
			if value.ID == mutation.ValueAfter.ID {
				candidate := cloneSegmentValue(value)
				currentValue = &candidate
				break
			}
		}
		if currentValue == nil {
			return ErrSegmentValueNotFound
		}
		if FingerprintSegmentValue(*currentValue) != FingerprintSegmentValue(*mutation.ValueBefore) {
			return ErrSegmentValueVersionConflict
		}
	}
	if repository.valueAudit == nil {
		return ErrSegmentValueAuditUnavailable
	}
	if err := repository.valueAudit.RecordSegmentValueMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.definitions[mutation.After.ID] = cloneSegmentDefinition(mutation.After)
	return nil
}

func (authorizer MemoryAuthorizer) AuthorizeSegmentValue(context.Context, Actor, SegmentValueCommand, *SegmentDefinition, *SegmentValue) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

func (recorder *MemoryAuditRecorder) RecordSegmentValueMutation(_ context.Context, record AuditRecord) error {
	if recorder == nil {
		return ErrSegmentValueAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func cloneSegmentValue(value SegmentValue) SegmentValue {
	value.EffectiveDateFrom = dateOnly(value.EffectiveDateFrom)
	value.EffectiveDateTo = cloneDate(value.EffectiveDateTo)
	return value
}
