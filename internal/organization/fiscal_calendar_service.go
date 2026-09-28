package organization

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
	platformidentity "github.com/toanle88/Tally/internal/platform/identity"
)

type FiscalCalendarCommand struct {
	Action           string
	FiscalCalendarID uuid.UUID
	ScopeID          uuid.UUID
	CalendarType     string
	PeriodPattern    string
	EffectiveFrom    time.Time
	EffectiveTo      *time.Time
	Periods          []CalendarPeriod
	Approval         *ApprovalDecisionReference
	ExpectedVersion  *aggregateversion.AggregateVersion
	IdempotencyKey   string
	CorrelationID    string
	CausationID      string
}

func (command FiscalCalendarCommand) Validate() error {
	switch command.Action {
	case FiscalCalendarActionCreate:
		if command.FiscalCalendarID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include calendar id or expected version", ErrInvalidFiscalCalendarCommand)
		}
		if command.EffectiveTo != nil {
			return fmt.Errorf("%w: create cannot include an end date; maintain the calendar to end-date it", ErrInvalidFiscalCalendarCommand)
		}
	case FiscalCalendarActionMaintain:
		if command.FiscalCalendarID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: maintain calendar id and expected version are required", ErrInvalidFiscalCalendarCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action", ErrInvalidFiscalCalendarCommand)
	}
	if command.ScopeID == uuid.Nil || command.EffectiveFrom.IsZero() {
		return fmt.Errorf("%w: accounting scope and effective date are required", ErrInvalidFiscalCalendarCommand)
	}
	if command.EffectiveTo != nil && !command.EffectiveTo.After(command.EffectiveFrom) {
		return fmt.Errorf("%w: effective interval is invalid", ErrInvalidFiscalCalendarCommand)
	}
	if command.CalendarType != canonicalFiscalCalendarCode(command.CalendarType) || !fiscalCalendarCodePattern.MatchString(command.CalendarType) {
		return fmt.Errorf("%w: calendar type must be a canonical code", ErrInvalidFiscalCalendarCommand)
	}
	if command.PeriodPattern != canonicalFiscalCalendarCode(command.PeriodPattern) || !fiscalCalendarCodePattern.MatchString(command.PeriodPattern) {
		return fmt.Errorf("%w: period pattern must be canonical metadata", ErrInvalidFiscalCalendarCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidFiscalCalendarCommand)
	}
	status := FiscalCalendarStatusActive
	if command.EffectiveTo != nil {
		status = FiscalCalendarStatusEndDated
	}
	if _, err := newFiscalCalendar(uuid.New(), command.ScopeID, command.CalendarType, command.PeriodPattern, command.EffectiveFrom, command.EffectiveTo, command.Periods, command.Approval, status, time.Unix(1, 0).UTC()); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFiscalCalendarCommand, err)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidFiscalCalendarCommand)
		}
	}
	return nil
}

type FiscalCalendarAuthorizer interface {
	AuthorizeFiscalCalendar(context.Context, Actor, FiscalCalendarCommand, *FiscalCalendar) (AuthorizationDecision, error)
}

type FiscalCalendarApprovalValidator interface {
	ValidateFiscalCalendarApproval(context.Context, FiscalCalendarCommand, FiscalCalendar) error
}

type FiscalCalendarAuditRecord struct {
	FiscalCalendarID      uuid.UUID
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
	ChangedFields         []string
	CorrelationID         string
	CausationID           string
}

type FiscalCalendarAuditRecorder interface {
	RecordFiscalCalendarMutation(context.Context, FiscalCalendarAuditRecord) error
}

type FiscalCalendarImpactReader interface {
	ReadFiscalCalendarImpact(context.Context, Actor, FiscalCalendar) (FiscalCalendarImpact, error)
}

type UnavailableFiscalCalendarImpactReader struct{}

func (UnavailableFiscalCalendarImpactReader) ReadFiscalCalendarImpact(context.Context, Actor, FiscalCalendar) (FiscalCalendarImpact, error) {
	return FiscalCalendarImpact{Availability: "unavailable", Reason: "no approved downstream fiscal-calendar reader is configured"}, ErrFiscalCalendarImpactUnavailable
}

type FiscalCalendarMutation struct {
	Before          FiscalCalendar
	After           FiscalCalendar
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           FiscalCalendarAuditRecord
}

type FiscalCalendarRepository interface {
	Get(context.Context, uuid.UUID) (FiscalCalendar, error)
	List(context.Context, *uuid.UUID) ([]FiscalCalendar, error)
	CommitFiscalCalendarMutation(context.Context, FiscalCalendarMutation) error
}

type DurableFiscalCalendarRepository interface {
	CommitFiscalCalendarMutationWithIdempotency(context.Context, FiscalCalendarMutation, DurableFiscalCalendarMutationCommit) error
}

type DurableFiscalCalendarMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableFiscalCalendarServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type FiscalCalendarCommandResult struct {
	FiscalCalendar    SafeFiscalCalendar   `json:"fiscalCalendar"`
	Impact            FiscalCalendarImpact `json:"impact"`
	DecisionReference uuid.UUID            `json:"decisionReference"`
	PolicyReference   string               `json:"policyReference"`
	ValidationOutcome string               `json:"validationOutcome"`
	Replayed          bool                 `json:"replayed,omitempty"`
}

type FiscalCalendarService struct {
	repository   FiscalCalendarRepository
	authorizer   FiscalCalendarAuthorizer
	approval     FiscalCalendarApprovalValidator
	audit        FiscalCalendarAuditRecorder
	impactReader FiscalCalendarImpactReader
	clock        func() time.Time
	durable      *DurableFiscalCalendarServiceConfig
	mu           sync.Mutex
	idempotency  map[string]storedFiscalCalendarCommand
}

type storedFiscalCalendarCommand struct {
	fingerprint string
	result      FiscalCalendarCommandResult
}

func NewFiscalCalendarService(repository FiscalCalendarRepository, authorizer FiscalCalendarAuthorizer, approval FiscalCalendarApprovalValidator, audit FiscalCalendarAuditRecorder, impactReader FiscalCalendarImpactReader, clock func() time.Time) (*FiscalCalendarService, error) {
	return newFiscalCalendarService(repository, authorizer, approval, audit, impactReader, clock, nil)
}

func NewFiscalCalendarServiceWithDurableIdempotency(repository FiscalCalendarRepository, authorizer FiscalCalendarAuthorizer, approval FiscalCalendarApprovalValidator, audit FiscalCalendarAuditRecorder, impactReader FiscalCalendarImpactReader, clock func() time.Time, durable DurableFiscalCalendarServiceConfig) (*FiscalCalendarService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidFiscalCalendarService
	}
	return newFiscalCalendarService(repository, authorizer, approval, audit, impactReader, clock, &durable)
}

func newFiscalCalendarService(repository FiscalCalendarRepository, authorizer FiscalCalendarAuthorizer, approval FiscalCalendarApprovalValidator, audit FiscalCalendarAuditRecorder, impactReader FiscalCalendarImpactReader, clock func() time.Time, durable *DurableFiscalCalendarServiceConfig) (*FiscalCalendarService, error) {
	if repository == nil || authorizer == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidFiscalCalendarService
	}
	if impactReader == nil {
		impactReader = UnavailableFiscalCalendarImpactReader{}
	}
	if binder, ok := repository.(interface {
		BindFiscalCalendarAuditRecorder(FiscalCalendarAuditRecorder)
	}); ok {
		binder.BindFiscalCalendarAuditRecorder(audit)
	}
	return &FiscalCalendarService{repository: repository, authorizer: authorizer, approval: approval, audit: audit, impactReader: impactReader, clock: clock, durable: durable, idempotency: make(map[string]storedFiscalCalendarCommand)}, nil
}

func (service *FiscalCalendarService) Execute(ctx context.Context, actor Actor, command FiscalCalendarCommand) (FiscalCalendarCommandResult, error) {
	if service == nil {
		return FiscalCalendarCommandResult{}, ErrInvalidFiscalCalendarService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return FiscalCalendarCommandResult{}, ErrFiscalCalendarAuthorizationDenied
	}
	if err := command.Validate(); err != nil {
		return FiscalCalendarCommandResult{}, err
	}
	fingerprint, err := fiscalCalendarCommandFingerprint(command)
	if err != nil {
		return FiscalCalendarCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.ScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scope, marshalErr := json.Marshal(struct {
			Module string    `json:"module"`
			Actor  uuid.UUID `json:"actorId"`
			Scope  uuid.UUID `json:"scopeId"`
		}{"organization", actor.UserID, command.ScopeID})
		if marshalErr != nil {
			return FiscalCalendarCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return FiscalCalendarCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return FiscalCalendarCommandResult{}, ErrFiscalCalendarIdempotencyConflict
		}
		if err != nil {
			return FiscalCalendarCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return FiscalCalendarCommandResult{}, ErrFiscalCalendarCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return FiscalCalendarCommandResult{}, ErrFiscalCalendarDurableCommandFailed
			}
			result, decodeErr := decodeDurableFiscalCalendarResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return FiscalCalendarCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return FiscalCalendarCommandResult{}, ErrFiscalCalendarIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	var current *FiscalCalendar
	if command.Action == FiscalCalendarActionMaintain {
		loaded, getErr := service.repository.Get(ctx, command.FiscalCalendarID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return FiscalCalendarCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrFiscalCalendarVersionConflict)
			return FiscalCalendarCommandResult{}, ErrFiscalCalendarVersionConflict
		}
		if current.ScopeID != command.ScopeID {
			service.finalizeDurableFailure(ctx, acquisition, ErrFiscalCalendarAuthorizationDenied)
			return FiscalCalendarCommandResult{}, ErrFiscalCalendarAuthorizationDenied
		}
	}
	decision, err := service.authorizer.AuthorizeFiscalCalendar(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return FiscalCalendarCommandResult{}, err
	}
	if err := authorizeFiscalCalendarDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return FiscalCalendarCommandResult{}, err
	}
	now := service.clock().UTC()
	var before, after FiscalCalendar
	switch command.Action {
	case FiscalCalendarActionCreate:
		before = FiscalCalendar{}
		status := FiscalCalendarStatusActive
		if decision.ApprovalRequired {
			status = FiscalCalendarStatusDraft
		}
		newID, idErr := platformidentity.NewAggregateID()
		if idErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, idErr)
			return FiscalCalendarCommandResult{}, idErr
		}
		after, err = NewFiscalCalendar(newID.UUID(), command.ScopeID, command.CalendarType, command.PeriodPattern, command.EffectiveFrom, command.Periods, command.Approval, status, now)
	case FiscalCalendarActionMaintain:
		before = cloneFiscalCalendar(*current)
		after = cloneFiscalCalendar(*current)
		err = after.Replace(*current, command.CalendarType, command.PeriodPattern, command.EffectiveFrom, command.EffectiveTo, command.Periods, command.Approval, now)
	}
	if err == nil && decision.ApprovalRequired {
		if command.Approval == nil {
			err = ErrFiscalCalendarApprovalRequired
		} else {
			err = service.approval.ValidateFiscalCalendarApproval(ctx, command, after)
		}
	} else if err == nil && command.Approval != nil {
		err = service.approval.ValidateFiscalCalendarApproval(ctx, command, after)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return FiscalCalendarCommandResult{}, err
	}
	impact, impactErr := service.impactReader.ReadFiscalCalendarImpact(ctx, actor, after)
	if impactErr != nil || strings.TrimSpace(impact.Availability) == "" {
		impact = FiscalCalendarImpact{Availability: "unavailable", Reason: "downstream fiscal-period impact is not available from an approved read-only boundary"}
	}
	result := FiscalCalendarCommandResult{FiscalCalendar: after.SafeProjection(), Impact: impact, DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "accepted"}
	record := FiscalCalendarAuditRecord{FiscalCalendarID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, ScopeID: after.ScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Approval: cloneApproval(command.Approval), RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintFiscalCalendar(before), AfterFingerprint: FingerprintFiscalCalendar(after), ChangedFields: fiscalCalendarChangedFields(before, after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	mutation := FiscalCalendarMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableFiscalCalendarRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidFiscalCalendarService)
			return FiscalCalendarCommandResult{}, ErrInvalidFiscalCalendarService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return FiscalCalendarCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return FiscalCalendarCommandResult{}, metadataErr
		}
		err = committer.CommitFiscalCalendarMutationWithIdempotency(ctx, mutation, DurableFiscalCalendarMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitFiscalCalendarMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return FiscalCalendarCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedFiscalCalendarCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func authorizeFiscalCalendarDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	switch strings.ToLower(strings.TrimSpace(decision.Outcome)) {
	case "stale":
		return ErrFiscalCalendarAuthorizationStale
	case "unavailable":
		return ErrFiscalCalendarAuthorizationUnavailable
	}
	if !decision.Allowed || decision.Permission != FiscalCalendarManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrFiscalCalendarAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrFiscalCalendarAuthorizationDenied
}

func fiscalCalendarCommandFingerprint(command FiscalCalendarCommand) (string, error) {
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:]), nil
}

func fiscalCalendarChangedFields(before, after FiscalCalendar) []string {
	changed := make(map[string]bool)
	if before.ID == uuid.Nil || before.CalendarType != after.CalendarType || before.PeriodPattern != after.PeriodPattern || !reflect.DeepEqual(before.Periods, after.Periods) {
		changed["calendar_definition"] = true
	}
	if before.ID == uuid.Nil || before.Status != after.Status || !before.EffectiveFrom.Equal(after.EffectiveFrom) || !sameFiscalCalendarTime(before.EffectiveTo, after.EffectiveTo) {
		changed["lifecycle"] = true
	}
	if before.ID == uuid.Nil || !sameApproval(before.Approval, after.Approval) {
		changed["approval"] = true
	}
	result := make([]string, 0, len(changed))
	for field, value := range changed {
		if value {
			result = append(result, field)
		}
	}
	sort.Strings(result)
	return result
}

func sameFiscalCalendarTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func (service *FiscalCalendarService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrFiscalCalendarAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrFiscalCalendarVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrFiscalCalendarNotFound):
		code = "CALENDAR_NOT_FOUND"
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

func decodeDurableFiscalCalendarResult(body []byte) (FiscalCalendarCommandResult, error) {
	var result FiscalCalendarCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return FiscalCalendarCommandResult{}, ErrFiscalCalendarCommandInProgress
	}
	return result, nil
}

type MemoryFiscalCalendarRepository struct {
	mu        sync.RWMutex
	calendars map[uuid.UUID]FiscalCalendar
	audit     FiscalCalendarAuditRecorder
}

func NewMemoryFiscalCalendarRepository() *MemoryFiscalCalendarRepository {
	return &MemoryFiscalCalendarRepository{calendars: make(map[uuid.UUID]FiscalCalendar)}
}

func (repository *MemoryFiscalCalendarRepository) BindFiscalCalendarAuditRecorder(audit FiscalCalendarAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryFiscalCalendarRepository) Get(_ context.Context, id uuid.UUID) (FiscalCalendar, error) {
	if repository == nil {
		return FiscalCalendar{}, ErrInvalidFiscalCalendarService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	calendar, ok := repository.calendars[id]
	if !ok {
		return FiscalCalendar{}, ErrFiscalCalendarNotFound
	}
	return cloneFiscalCalendar(calendar), nil
}

func (repository *MemoryFiscalCalendarRepository) List(_ context.Context, scopeID *uuid.UUID) ([]FiscalCalendar, error) {
	if repository == nil {
		return nil, ErrInvalidFiscalCalendarService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]FiscalCalendar, 0, len(repository.calendars))
	for _, calendar := range repository.calendars {
		if scopeID != nil && calendar.ScopeID != *scopeID {
			continue
		}
		result = append(result, cloneFiscalCalendar(calendar))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

func (repository *MemoryFiscalCalendarRepository) CommitFiscalCalendarMutation(ctx context.Context, mutation FiscalCalendarMutation) error {
	if repository == nil {
		return ErrInvalidFiscalCalendarService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrFiscalCalendarVersionConflict
		}
		for _, current := range repository.calendars {
			if current.ScopeID == mutation.After.ScopeID && current.CalendarType == mutation.After.CalendarType && current.Status != FiscalCalendarStatusEndDated {
				return ErrFiscalCalendarDuplicate
			}
		}
	} else {
		if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
			return ErrFiscalCalendarVersionConflict
		}
		current, ok := repository.calendars[mutation.After.ID]
		if !ok {
			return ErrFiscalCalendarNotFound
		}
		if !current.Version.Matches(*mutation.ExpectedVersion) || current.ScopeID != mutation.After.ScopeID {
			return ErrFiscalCalendarVersionConflict
		}
		if mutation.After.Status != FiscalCalendarStatusEndDated {
			for id, other := range repository.calendars {
				if id != mutation.After.ID && other.ScopeID == mutation.After.ScopeID && other.CalendarType == mutation.After.CalendarType && other.Status != FiscalCalendarStatusEndDated {
					return ErrFiscalCalendarDuplicate
				}
			}
		}
	}
	if repository.audit == nil {
		return ErrFiscalCalendarAuditUnavailable
	}
	if err := repository.audit.RecordFiscalCalendarMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.calendars[mutation.After.ID] = cloneFiscalCalendar(mutation.After)
	return nil
}

type MemoryFiscalCalendarAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryFiscalCalendarAuthorizer) AuthorizeFiscalCalendar(context.Context, Actor, FiscalCalendarCommand, *FiscalCalendar) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllFiscalCalendarApprovalValidator struct{}

func (AllowAllFiscalCalendarApprovalValidator) ValidateFiscalCalendarApproval(context.Context, FiscalCalendarCommand, FiscalCalendar) error {
	return nil
}

type MemoryFiscalCalendarAuditRecorder struct {
	mu      sync.Mutex
	Records []FiscalCalendarAuditRecord
	Err     error
}

func (recorder *MemoryFiscalCalendarAuditRecorder) RecordFiscalCalendarMutation(_ context.Context, record FiscalCalendarAuditRecord) error {
	if recorder == nil {
		return ErrFiscalCalendarAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	record.ChangedFields = append([]string(nil), record.ChangedFields...)
	record.Approval = cloneApproval(record.Approval)
	recorder.Records = append(recorder.Records, record)
	return nil
}

var _ FiscalCalendarRepository = (*MemoryFiscalCalendarRepository)(nil)
var _ FiscalCalendarAuthorizer = MemoryFiscalCalendarAuthorizer{}
var _ FiscalCalendarApprovalValidator = AllowAllFiscalCalendarApprovalValidator{}
var _ FiscalCalendarAuditRecorder = (*MemoryFiscalCalendarAuditRecorder)(nil)
var _ FiscalCalendarImpactReader = UnavailableFiscalCalendarImpactReader{}
