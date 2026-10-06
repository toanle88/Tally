package coa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const SegmentCombinationValidationPermission = "finance.coa.validate.segment.combinations"

const (
	SegmentCombinationValidationStatusValid   = "valid"
	SegmentCombinationValidationStatusInvalid = "invalid"
	SegmentCombinationEffective               = "effective"
	SegmentCombinationNotEffective            = "not-effective"
)

var (
	ErrInvalidSegmentCombinationValidationService           = errors.New("invalid segment-combination validation service")
	ErrInvalidSegmentCombinationValidationCommand           = errors.New("invalid segment-combination validation command")
	ErrSegmentCombinationValidationAuthorizationDenied      = errors.New("segment-combination validation authorization denied")
	ErrSegmentCombinationValidationAuthorizationUnavailable = errors.New("segment-combination validation authorization unavailable")
	ErrSegmentCombinationValidationAuthorizationStale       = errors.New("segment-combination validation authorization stale")
	ErrSegmentCombinationValidationIdempotencyConflict      = errors.New("segment-combination validation idempotency conflict")
	ErrSegmentCombinationValidationInProgress               = errors.New("segment-combination validation is already in progress")
	ErrSegmentCombinationValidationPreviouslyFailed         = errors.New("segment-combination validation previously failed")
)

type SegmentCombinationValueReference struct {
	SegmentDefinitionID uuid.UUID `json:"segmentDefinitionId"`
	SegmentValueID      uuid.UUID `json:"segmentValueId"`
}

type SegmentCombinationValidationCommand struct {
	ScopeID        uuid.UUID
	BusinessDate   time.Time
	SegmentValues  []SegmentCombinationValueReference
	IdempotencyKey string
	CorrelationID  string
	CausationID    string
}

func (command SegmentCombinationValidationCommand) Canonical() SegmentCombinationValidationCommand {
	command.BusinessDate = dateOnly(command.BusinessDate)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.SegmentValues = append([]SegmentCombinationValueReference(nil), command.SegmentValues...)
	sort.Slice(command.SegmentValues, func(left, right int) bool {
		if command.SegmentValues[left].SegmentDefinitionID != command.SegmentValues[right].SegmentDefinitionID {
			return command.SegmentValues[left].SegmentDefinitionID.String() < command.SegmentValues[right].SegmentDefinitionID.String()
		}
		return command.SegmentValues[left].SegmentValueID.String() < command.SegmentValues[right].SegmentValueID.String()
	})
	return command
}

func (command SegmentCombinationValidationCommand) Validate() error {
	command = command.Canonical()
	if command.ScopeID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope is required", ErrInvalidSegmentCombinationValidationCommand)
	}
	if command.BusinessDate.IsZero() {
		return fmt.Errorf("%w: business date is required", ErrInvalidSegmentCombinationValidationCommand)
	}
	if len(command.SegmentValues) == 0 {
		return fmt.Errorf("%w: at least one segment value is required", ErrInvalidSegmentCombinationValidationCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidSegmentCombinationValidationCommand)
	}
	for index, reference := range command.SegmentValues {
		if reference.SegmentDefinitionID == uuid.Nil || reference.SegmentValueID == uuid.Nil {
			return fmt.Errorf("%w: segment value reference %d is incomplete", ErrInvalidSegmentCombinationValidationCommand, index)
		}
	}
	return nil
}

type SegmentCombinationValidationIssue struct {
	SegmentDefinitionID *uuid.UUID `json:"segmentDefinitionId,omitempty"`
	SegmentValueID      *uuid.UUID `json:"segmentValueId,omitempty"`
	Reason              string     `json:"reason"`
}

type SegmentCombinationSourceVersion struct {
	SegmentDefinitionID       uuid.UUID `json:"segmentDefinitionId"`
	SegmentValueID            uuid.UUID `json:"segmentValueId"`
	SegmentDefinitionVersion  int64     `json:"segmentDefinitionVersion"`
	SegmentDefinitionRevision int64     `json:"segmentDefinitionRevision"`
}

type SegmentCombinationValidationResult struct {
	ValidationStatus    string                              `json:"validationStatus"`
	EffectiveDateResult string                              `json:"effectiveDateResult"`
	SourceVersions      []SegmentCombinationSourceVersion   `json:"sourceVersions"`
	InvalidValues       []SegmentCombinationValidationIssue `json:"invalidValues"`
	Restrictions        []string                            `json:"restrictions"`
	RejectionReasons    []SegmentCombinationValidationIssue `json:"rejectionReasons"`
	NextAction          string                              `json:"nextAction"`
	Replayed            bool                                `json:"replayed,omitempty"`
}

type SegmentCombinationValidationSnapshot struct {
	Definitions map[uuid.UUID]SegmentDefinition
}

type SegmentCombinationValidationRepository interface {
	ReadSegmentCombinationValidationSnapshot(context.Context, uuid.UUID, []SegmentCombinationValueReference) (SegmentCombinationValidationSnapshot, error)
}

type SegmentCombinationValidationAuthorizer interface {
	AuthorizeSegmentCombinationValidation(context.Context, Actor, SegmentCombinationValidationCommand) (AuthorizationDecision, error)
}

type DurableSegmentCombinationValidationServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type SegmentCombinationValidationService struct {
	repository  SegmentCombinationValidationRepository
	authorizer  SegmentCombinationValidationAuthorizer
	durable     *DurableSegmentCombinationValidationServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSegmentCombinationValidation
}

type storedSegmentCombinationValidation struct {
	fingerprint string
	result      SegmentCombinationValidationResult
}

func NewSegmentCombinationValidationService(repository SegmentCombinationValidationRepository, authorizer SegmentCombinationValidationAuthorizer) (*SegmentCombinationValidationService, error) {
	return newSegmentCombinationValidationService(repository, authorizer, nil)
}

func NewSegmentCombinationValidationServiceWithDurableIdempotency(repository SegmentCombinationValidationRepository, authorizer SegmentCombinationValidationAuthorizer, durable DurableSegmentCombinationValidationServiceConfig) (*SegmentCombinationValidationService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidSegmentCombinationValidationService
	}
	return newSegmentCombinationValidationService(repository, authorizer, &durable)
}

func newSegmentCombinationValidationService(repository SegmentCombinationValidationRepository, authorizer SegmentCombinationValidationAuthorizer, durable *DurableSegmentCombinationValidationServiceConfig) (*SegmentCombinationValidationService, error) {
	if repository == nil || authorizer == nil {
		return nil, ErrInvalidSegmentCombinationValidationService
	}
	return &SegmentCombinationValidationService{
		repository:  repository,
		authorizer:  authorizer,
		durable:     durable,
		idempotency: make(map[string]storedSegmentCombinationValidation),
	}, nil
}

func (service *SegmentCombinationValidationService) Execute(ctx context.Context, actor Actor, command SegmentCombinationValidationCommand) (SegmentCombinationValidationResult, error) {
	if service == nil {
		return SegmentCombinationValidationResult{}, ErrInvalidSegmentCombinationValidationService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return SegmentCombinationValidationResult{}, ErrSegmentCombinationValidationAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SegmentCombinationValidationResult{}, err
	}
	fingerprint, err := segmentCombinationValidationFingerprint(command)
	if err != nil {
		return SegmentCombinationValidationResult{}, err
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
			return SegmentCombinationValidationResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return SegmentCombinationValidationResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return SegmentCombinationValidationResult{}, ErrSegmentCombinationValidationIdempotencyConflict
		}
		if err != nil {
			return SegmentCombinationValidationResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return SegmentCombinationValidationResult{}, ErrSegmentCombinationValidationInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return SegmentCombinationValidationResult{}, ErrSegmentCombinationValidationPreviouslyFailed
			}
			result, decodeErr := decodeDurableSegmentCombinationValidationResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return SegmentCombinationValidationResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return SegmentCombinationValidationResult{}, ErrSegmentCombinationValidationIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	decision, err := service.authorizer.AuthorizeSegmentCombinationValidation(ctx, actor, command)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentCombinationValidationResult{}, err
	}
	if err := authorizeSegmentCombinationValidationDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentCombinationValidationResult{}, err
	}

	snapshot, err := service.repository.ReadSegmentCombinationValidationSnapshot(ctx, command.ScopeID, command.SegmentValues)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentCombinationValidationResult{}, err
	}
	result := validateSegmentCombination(command, snapshot)
	if service.durable != nil {
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return SegmentCombinationValidationResult{}, marshalErr
		}
		status := 200
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, nil, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return SegmentCombinationValidationResult{}, metadataErr
		}
		if err := service.finalizeDurableResult(ctx, acquisition, metadata); err != nil {
			return SegmentCombinationValidationResult{}, err
		}
	} else {
		service.idempotency[key] = storedSegmentCombinationValidation{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func authorizeSegmentCombinationValidationDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != SegmentCombinationValidationPermission || decision.DecisionReference == uuid.Nil {
		return ErrSegmentCombinationValidationAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrSegmentCombinationValidationAuthorizationDenied
}

func validateSegmentCombination(command SegmentCombinationValidationCommand, snapshot SegmentCombinationValidationSnapshot) SegmentCombinationValidationResult {
	result := SegmentCombinationValidationResult{
		ValidationStatus:    SegmentCombinationValidationStatusValid,
		EffectiveDateResult: SegmentCombinationEffective,
		SourceVersions:      make([]SegmentCombinationSourceVersion, 0, len(command.SegmentValues)),
		InvalidValues:       make([]SegmentCombinationValidationIssue, 0),
		Restrictions:        make([]string, 0),
		RejectionReasons:    make([]SegmentCombinationValidationIssue, 0),
		NextAction:          "proceed",
	}
	restrictions := make(map[string]struct{})
	seenDefinitions := make(map[uuid.UUID]struct{})
	dateEffective := true
	addIssue := func(definitionID, valueID uuid.UUID, reason, restriction string, invalidValue bool) {
		issue := SegmentCombinationValidationIssue{Reason: reason}
		if definitionID != uuid.Nil {
			id := definitionID
			issue.SegmentDefinitionID = &id
		}
		if valueID != uuid.Nil {
			id := valueID
			issue.SegmentValueID = &id
		}
		result.RejectionReasons = append(result.RejectionReasons, issue)
		if invalidValue {
			result.InvalidValues = append(result.InvalidValues, issue)
		}
		if restriction != "" {
			restrictions[restriction] = struct{}{}
		}
	}

	for _, reference := range command.SegmentValues {
		definition, found := snapshot.Definitions[reference.SegmentDefinitionID]
		if _, duplicate := seenDefinitions[reference.SegmentDefinitionID]; duplicate {
			addIssue(reference.SegmentDefinitionID, reference.SegmentValueID, "a segment definition is selected more than once", "duplicate", true)
		} else {
			seenDefinitions[reference.SegmentDefinitionID] = struct{}{}
		}
		if !found {
			addIssue(reference.SegmentDefinitionID, reference.SegmentValueID, "the segment definition was not found in the selected scope", "missing-value", true)
			dateEffective = false
			continue
		}
		if definition.ScopeID != command.ScopeID {
			addIssue(definition.ID, reference.SegmentValueID, "the segment definition is outside the requested accounting scope", "scope", true)
			dateEffective = false
			continue
		}
		result.SourceVersions = append(result.SourceVersions, SegmentCombinationSourceVersion{
			SegmentDefinitionID:       definition.ID,
			SegmentValueID:            reference.SegmentValueID,
			SegmentDefinitionVersion:  definition.Version.Value(),
			SegmentDefinitionRevision: definition.RevisionNumber,
		})
		if definition.Status != SegmentStatusActive {
			addIssue(definition.ID, reference.SegmentValueID, "the segment definition is not active for validation", "lifecycle", true)
		}
		if !dateWithinRange(command.BusinessDate, definition.EffectiveDateFrom, definition.EffectiveDateTo) {
			addIssue(definition.ID, reference.SegmentValueID, "the segment definition is not effective on the requested date", "effective-date", true)
			dateEffective = false
		}

		var value *SegmentValue
		for index := range definition.Values {
			if definition.Values[index].ID == reference.SegmentValueID {
				candidate := definition.Values[index]
				value = &candidate
				break
			}
		}
		if value == nil {
			addIssue(definition.ID, reference.SegmentValueID, "the segment value was not found in the selected definition", "missing-value", true)
			dateEffective = false
			continue
		}
		if value.Status != SegmentStatusActive {
			addIssue(definition.ID, value.ID, "the segment value is not active for validation", "lifecycle", true)
		}
		if !dateWithinRange(command.BusinessDate, value.EffectiveDateFrom, value.EffectiveDateTo) {
			addIssue(definition.ID, value.ID, "the segment value is not effective on the requested date", "effective-date", true)
			dateEffective = false
		}
	}

	result.ValidationStatus = SegmentCombinationValidationStatusInvalid
	if len(result.RejectionReasons) == 0 {
		result.ValidationStatus = SegmentCombinationValidationStatusValid
	}
	if !dateEffective {
		result.EffectiveDateResult = SegmentCombinationNotEffective
	}
	for restriction := range restrictions {
		result.Restrictions = append(result.Restrictions, restriction)
	}
	sort.Strings(result.Restrictions)
	sortSegmentCombinationValidationIssues(result.RejectionReasons)
	sortSegmentCombinationValidationIssues(result.InvalidValues)
	if result.ValidationStatus != SegmentCombinationValidationStatusValid {
		result.NextAction = "correct-and-revalidate"
	}
	return result
}

func dateWithinRange(value, from time.Time, to *time.Time) bool {
	value = dateOnly(value)
	from = dateOnly(from)
	if value.Before(from) {
		return false
	}
	return to == nil || !value.After(dateOnly(*to))
}

func sortSegmentCombinationValidationIssues(issues []SegmentCombinationValidationIssue) {
	sort.Slice(issues, func(left, right int) bool {
		leftDefinition, rightDefinition := uuid.Nil, uuid.Nil
		if issues[left].SegmentDefinitionID != nil {
			leftDefinition = *issues[left].SegmentDefinitionID
		}
		if issues[right].SegmentDefinitionID != nil {
			rightDefinition = *issues[right].SegmentDefinitionID
		}
		if leftDefinition != rightDefinition {
			return leftDefinition.String() < rightDefinition.String()
		}
		leftValue, rightValue := uuid.Nil, uuid.Nil
		if issues[left].SegmentValueID != nil {
			leftValue = *issues[left].SegmentValueID
		}
		if issues[right].SegmentValueID != nil {
			rightValue = *issues[right].SegmentValueID
		}
		if leftValue != rightValue {
			return leftValue.String() < rightValue.String()
		}
		return issues[left].Reason < issues[right].Reason
	})
}

func segmentCombinationValidationFingerprint(command SegmentCombinationValidationCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (service *SegmentCombinationValidationService) finalizeDurableResult(ctx context.Context, acquisition platformidempotency.DurableAcquisition, result platformidempotency.CommandResultMetadata) error {
	tx, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, tx, acquisition, result); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (service *SegmentCombinationValidationService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{validationFailureCode(commandErr)})
	if err != nil {
		return
	}
	initial := acquisition.Result()
	metadata, err := platformidempotency.NewCommandResultMetadata(initial.Identity(), initial.Fingerprint(), initial.OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	_ = service.finalizeDurableResult(ctx, acquisition, metadata)
}

func validationFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrSegmentCombinationValidationAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrSegmentCombinationValidationAuthorizationStale):
		return "POLICY_STALE"
	case errors.Is(err, ErrSegmentCombinationValidationIdempotencyConflict):
		return "IDEMPOTENCY_CONFLICT"
	case errors.Is(err, ErrInvalidSegmentCombinationValidationCommand):
		return "INVALID_REQUEST"
	default:
		return "SERVICE_UNAVAILABLE"
	}
}

func decodeDurableSegmentCombinationValidationResult(body []byte) (SegmentCombinationValidationResult, error) {
	var result SegmentCombinationValidationResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SegmentCombinationValidationResult{}, ErrSegmentCombinationValidationInProgress
	}
	return result, nil
}

func (authorizer MemoryAuthorizer) AuthorizeSegmentCombinationValidation(context.Context, Actor, SegmentCombinationValidationCommand) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

func (repository *MemorySegmentDefinitionRepository) ReadSegmentCombinationValidationSnapshot(_ context.Context, _ uuid.UUID, references []SegmentCombinationValueReference) (SegmentCombinationValidationSnapshot, error) {
	if repository == nil {
		return SegmentCombinationValidationSnapshot{}, ErrInvalidSegmentCombinationValidationService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	definitions := make(map[uuid.UUID]SegmentDefinition)
	for _, reference := range references {
		if _, alreadyRead := definitions[reference.SegmentDefinitionID]; alreadyRead {
			continue
		}
		definition, found := repository.definitions[reference.SegmentDefinitionID]
		if found {
			definitions[reference.SegmentDefinitionID] = cloneSegmentDefinition(definition)
		}
	}
	return SegmentCombinationValidationSnapshot{Definitions: definitions}, nil
}

func (repository *PostgresSegmentDefinitionRepository) ReadSegmentCombinationValidationSnapshot(ctx context.Context, _ uuid.UUID, references []SegmentCombinationValueReference) (SegmentCombinationValidationSnapshot, error) {
	if repository == nil || repository.pool == nil {
		return SegmentCombinationValidationSnapshot{}, ErrInvalidSegmentCombinationValidationService
	}
	tx, err := repository.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return SegmentCombinationValidationSnapshot{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	definitions := make(map[uuid.UUID]SegmentDefinition)
	ids := make([]uuid.UUID, 0, len(references))
	seen := make(map[uuid.UUID]struct{})
	for _, reference := range references {
		if _, ok := seen[reference.SegmentDefinitionID]; ok {
			continue
		}
		seen[reference.SegmentDefinitionID] = struct{}{}
		ids = append(ids, reference.SegmentDefinitionID)
	}
	sort.Slice(ids, func(left, right int) bool { return ids[left].String() < ids[right].String() })
	for _, id := range ids {
		definition, readErr := readSegmentDefinition(ctx, tx, id)
		if errors.Is(readErr, ErrSegmentDefinitionNotFound) {
			continue
		}
		if readErr != nil {
			return SegmentCombinationValidationSnapshot{}, readErr
		}
		definitions[id] = definition
	}
	if err := tx.Commit(ctx); err != nil {
		return SegmentCombinationValidationSnapshot{}, err
	}
	committed = true
	return SegmentCombinationValidationSnapshot{Definitions: definitions}, nil
}

var _ SegmentCombinationValidationRepository = (*MemorySegmentDefinitionRepository)(nil)
var _ SegmentCombinationValidationRepository = (*PostgresSegmentDefinitionRepository)(nil)
var _ SegmentCombinationValidationAuthorizer = MemoryAuthorizer{}
