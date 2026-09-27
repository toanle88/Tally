package organization

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
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

type CustomerProfileCommand struct {
	Action               string
	CustomerProfileID    uuid.UUID
	ScopeID              uuid.UUID
	PartyID              uuid.UUID
	ExpectedPartyVersion *aggregateversion.AggregateVersion
	CreditTerms          string
	CreditLimit          CreditLimit
	BillingPreference    string
	TaxTreatment         string
	EffectiveFrom        time.Time
	EffectiveTo          *time.Time
	Approval             *ApprovalDecisionReference
	ExpectedVersion      *aggregateversion.AggregateVersion
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command CustomerProfileCommand) Validate() error {
	switch command.Action {
	case CustomerProfileActionCreate:
		if command.CustomerProfileID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include profile id or expected version", ErrInvalidCustomerProfileCommand)
		}
		if command.EffectiveTo != nil {
			return fmt.Errorf("%w: create cannot include an end date; maintain the profile to end-date it", ErrInvalidCustomerProfileCommand)
		}
	case CustomerProfileActionMaintain:
		if command.CustomerProfileID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: maintain profile id and expected version are required", ErrInvalidCustomerProfileCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action", ErrInvalidCustomerProfileCommand)
	}
	if command.ScopeID == uuid.Nil || command.PartyID == uuid.Nil || command.ExpectedPartyVersion == nil {
		return fmt.Errorf("%w: accounting scope, party, and expected party version are required", ErrInvalidCustomerProfileCommand)
	}
	if err := validateCustomerProfileCode(command.CreditTerms, "credit terms"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCustomerProfileCommand, err)
	}
	if _, err := command.CreditLimit.Canonicalize(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCustomerProfileCommand, err)
	}
	if err := validateCustomerProfileCode(command.BillingPreference, "billing preference"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCustomerProfileCommand, err)
	}
	if err := validateCustomerProfileCode(command.TaxTreatment, "tax treatment"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidCustomerProfileCommand, err)
	}
	if command.EffectiveFrom.IsZero() || (command.EffectiveTo != nil && !command.EffectiveTo.After(command.EffectiveFrom)) {
		return fmt.Errorf("%w: effective interval is invalid", ErrInvalidCustomerProfileCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidCustomerProfileCommand)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidCustomerProfileCommand)
		}
	}
	return nil
}

type CustomerProfileAuthorizer interface {
	AuthorizeCustomerProfile(context.Context, Actor, CustomerProfileCommand, *CustomerProfile) (AuthorizationDecision, error)
}

type CustomerProfileFieldAuthorizer interface {
	AuthorizeCustomerProfileFields(context.Context, Actor, CustomerProfileCommand, *CustomerProfile) (CustomerProfileFieldAuthorization, error)
}

type CustomerProfileApprovalValidator interface {
	ValidateCustomerProfileApproval(context.Context, CustomerProfileCommand, CustomerProfile) error
}

type CustomerProfileAuditRecord struct {
	CustomerProfileID     uuid.UUID
	PartyID               uuid.UUID
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

type CustomerProfileAuditRecorder interface {
	RecordCustomerProfileMutation(context.Context, CustomerProfileAuditRecord) error
}

type CustomerProfileMutation struct {
	Before               CustomerProfile
	After                CustomerProfile
	ExpectedVersion      *aggregateversion.AggregateVersion
	ExpectedPartyVersion *aggregateversion.AggregateVersion
	Audit                CustomerProfileAuditRecord
}

type CustomerProfileRepository interface {
	Get(context.Context, uuid.UUID) (CustomerProfile, error)
	List(context.Context, *uuid.UUID) ([]CustomerProfile, error)
	CommitCustomerProfileMutation(context.Context, CustomerProfileMutation) error
}

type DurableCustomerProfileRepository interface {
	CommitCustomerProfileMutationWithIdempotency(context.Context, CustomerProfileMutation, DurableCustomerProfileMutationCommit) error
}

type DurableCustomerProfileMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableCustomerProfileServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type CustomerProfileCommandResult struct {
	CustomerProfile   SafeCustomerProfile `json:"customerProfile"`
	DecisionReference uuid.UUID           `json:"decisionReference"`
	PolicyReference   string              `json:"policyReference"`
	ValidationOutcome string              `json:"validationOutcome"`
	Replayed          bool                `json:"replayed,omitempty"`
}

type CustomerProfileService struct {
	repository      CustomerProfileRepository
	partyRepository PartyRepository
	authorizer      CustomerProfileAuthorizer
	fieldAuthorizer CustomerProfileFieldAuthorizer
	approval        CustomerProfileApprovalValidator
	audit           CustomerProfileAuditRecorder
	clock           func() time.Time
	durable         *DurableCustomerProfileServiceConfig
	mu              sync.Mutex
	idempotency     map[string]storedCustomerProfileCommand
}

type storedCustomerProfileCommand struct {
	fingerprint string
	result      CustomerProfileCommandResult
}

func NewCustomerProfileService(repository CustomerProfileRepository, partyRepository PartyRepository, authorizer CustomerProfileAuthorizer, fieldAuthorizer CustomerProfileFieldAuthorizer, approval CustomerProfileApprovalValidator, audit CustomerProfileAuditRecorder, clock func() time.Time) (*CustomerProfileService, error) {
	return newCustomerProfileService(repository, partyRepository, authorizer, fieldAuthorizer, approval, audit, clock, nil)
}

func NewCustomerProfileServiceWithDurableIdempotency(repository CustomerProfileRepository, partyRepository PartyRepository, authorizer CustomerProfileAuthorizer, fieldAuthorizer CustomerProfileFieldAuthorizer, approval CustomerProfileApprovalValidator, audit CustomerProfileAuditRecorder, clock func() time.Time, durable DurableCustomerProfileServiceConfig) (*CustomerProfileService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidCustomerProfileService
	}
	return newCustomerProfileService(repository, partyRepository, authorizer, fieldAuthorizer, approval, audit, clock, &durable)
}

func newCustomerProfileService(repository CustomerProfileRepository, partyRepository PartyRepository, authorizer CustomerProfileAuthorizer, fieldAuthorizer CustomerProfileFieldAuthorizer, approval CustomerProfileApprovalValidator, audit CustomerProfileAuditRecorder, clock func() time.Time, durable *DurableCustomerProfileServiceConfig) (*CustomerProfileService, error) {
	if repository == nil || partyRepository == nil || authorizer == nil || fieldAuthorizer == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidCustomerProfileService
	}
	if binder, ok := repository.(interface {
		BindCustomerProfileAuditRecorder(CustomerProfileAuditRecorder)
	}); ok {
		binder.BindCustomerProfileAuditRecorder(audit)
	}
	return &CustomerProfileService{repository: repository, partyRepository: partyRepository, authorizer: authorizer, fieldAuthorizer: fieldAuthorizer, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedCustomerProfileCommand)}, nil
}

func (service *CustomerProfileService) Execute(ctx context.Context, actor Actor, command CustomerProfileCommand) (CustomerProfileCommandResult, error) {
	if service == nil {
		return CustomerProfileCommandResult{}, ErrInvalidCustomerProfileService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return CustomerProfileCommandResult{}, ErrCustomerProfileAuthorizationDenied
	}
	if err := command.Validate(); err != nil {
		return CustomerProfileCommandResult{}, err
	}
	fingerprint, err := customerProfileCommandFingerprint(command)
	if err != nil {
		return CustomerProfileCommandResult{}, err
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
			return CustomerProfileCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return CustomerProfileCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return CustomerProfileCommandResult{}, ErrCustomerProfileIdempotencyConflict
		}
		if err != nil {
			return CustomerProfileCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return CustomerProfileCommandResult{}, ErrCustomerProfileCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return CustomerProfileCommandResult{}, ErrCustomerProfileDurableCommandFailed
			}
			result, decodeErr := decodeDurableCustomerProfileResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return CustomerProfileCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return CustomerProfileCommandResult{}, ErrCustomerProfileIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	var current *CustomerProfile
	if command.Action == CustomerProfileActionMaintain {
		loaded, getErr := service.repository.Get(ctx, command.CustomerProfileID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return CustomerProfileCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfileVersionConflict)
			return CustomerProfileCommandResult{}, ErrCustomerProfileVersionConflict
		}
		if current.ScopeID != command.ScopeID {
			service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfileAuthorizationDenied)
			return CustomerProfileCommandResult{}, ErrCustomerProfileAuthorizationDenied
		}
		if current.PartyID != command.PartyID {
			service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfilePartyMismatch)
			return CustomerProfileCommandResult{}, ErrCustomerProfilePartyMismatch
		}
	}
	party, err := service.partyRepository.Get(ctx, command.PartyID)
	if errors.Is(err, ErrPartyNotFound) {
		err = ErrCustomerProfilePartyNotFound
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return CustomerProfileCommandResult{}, err
	}
	if party.ScopeID != command.ScopeID || strings.ToLower(string(party.PartyType)) != "customer" {
		service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfilePartyInvalid)
		return CustomerProfileCommandResult{}, ErrCustomerProfilePartyInvalid
	}
	if !party.Version.Matches(*command.ExpectedPartyVersion) {
		service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfilePartyVersionConflict)
		return CustomerProfileCommandResult{}, ErrCustomerProfilePartyVersionConflict
	}
	decision, err := service.authorizer.AuthorizeCustomerProfile(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return CustomerProfileCommandResult{}, err
	}
	if err := authorizeCustomerProfileDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return CustomerProfileCommandResult{}, err
	}
	fieldAccess, err := service.fieldAuthorizer.AuthorizeCustomerProfileFields(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfileFieldAuthorizationUnavailable)
		return CustomerProfileCommandResult{}, ErrCustomerProfileFieldAuthorizationUnavailable
	}
	if !fieldAccess.all() {
		service.finalizeDurableFailure(ctx, acquisition, ErrCustomerProfileFieldAuthorizationDenied)
		return CustomerProfileCommandResult{}, ErrCustomerProfileFieldAuthorizationDenied
	}

	now := service.clock().UTC()
	var before, after CustomerProfile
	if command.Action == CustomerProfileActionCreate {
		before = CustomerProfile{}
		status := CustomerProfileStatusActive
		if decision.ApprovalRequired {
			status = CustomerProfileStatusDraft
		}
		newID, idErr := platformidentity.NewAggregateID()
		if idErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, idErr)
			return CustomerProfileCommandResult{}, idErr
		}
		after, err = NewCustomerProfile(newID.UUID(), command.ScopeID, command.PartyID, party.Version, command.CreditTerms, command.CreditLimit, command.BillingPreference, command.TaxTreatment, command.EffectiveFrom, command.Approval, status, now)
	} else {
		before = cloneCustomerProfile(*current)
		after = cloneCustomerProfile(*current)
		err = after.Replace(*current, party.Version, command.CreditTerms, command.CreditLimit, command.BillingPreference, command.TaxTreatment, command.EffectiveFrom, command.EffectiveTo, command.Approval, now)
	}
	if err == nil && decision.ApprovalRequired {
		if command.Approval == nil {
			err = ErrCustomerProfileApprovalRequired
		} else {
			err = service.approval.ValidateCustomerProfileApproval(ctx, command, after)
		}
	} else if err == nil && command.Approval != nil {
		err = service.approval.ValidateCustomerProfileApproval(ctx, command, after)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return CustomerProfileCommandResult{}, err
	}
	changedFields := customerProfileChangedFields(before, after)
	result := CustomerProfileCommandResult{CustomerProfile: after.SafeProjectionWithAccess(fieldAccess), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "accepted"}
	record := CustomerProfileAuditRecord{CustomerProfileID: after.ID, PartyID: after.PartyID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, ScopeID: after.ScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Approval: cloneApproval(command.Approval), RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintCustomerProfile(before), AfterFingerprint: FingerprintCustomerProfile(after), ChangedFields: sortedCustomerProfileChangedFields(changedFields), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	mutation := CustomerProfileMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, ExpectedPartyVersion: command.ExpectedPartyVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableCustomerProfileRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidCustomerProfileService)
			return CustomerProfileCommandResult{}, ErrInvalidCustomerProfileService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return CustomerProfileCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return CustomerProfileCommandResult{}, metadataErr
		}
		err = committer.CommitCustomerProfileMutationWithIdempotency(ctx, mutation, DurableCustomerProfileMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitCustomerProfileMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return CustomerProfileCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedCustomerProfileCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func authorizeCustomerProfileDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	switch strings.ToLower(strings.TrimSpace(decision.Outcome)) {
	case "stale":
		return ErrCustomerProfileAuthorizationStale
	case "unavailable":
		return ErrCustomerProfileAuthorizationUnavailable
	}
	if !decision.Allowed || decision.Permission != CustomerProfileManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrCustomerProfileAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrCustomerProfileAuthorizationDenied
}

func customerProfileCommandFingerprint(command CustomerProfileCommand) (string, error) {
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	return fingerprintBytes(data), nil
}

func fingerprintBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:])
}

func customerProfileChangedFields(before, after CustomerProfile) map[string]bool {
	changed := make(map[string]bool)
	if before.ID == uuid.Nil || before.PartyID != after.PartyID || before.PartyVersion != after.PartyVersion {
		changed["party_reference"] = true
	}
	if before.ID == uuid.Nil || before.CreditTerms != after.CreditTerms {
		changed["credit_terms"] = true
	}
	if before.ID == uuid.Nil || before.CreditLimit != after.CreditLimit {
		changed["credit_limit"] = true
	}
	if before.ID == uuid.Nil || before.BillingPreference != after.BillingPreference {
		changed["billing_preference"] = true
	}
	if before.ID == uuid.Nil || before.TaxTreatment != after.TaxTreatment {
		changed["tax_treatment"] = true
	}
	if before.ID == uuid.Nil || before.Status != after.Status || !before.EffectiveFrom.Equal(after.EffectiveFrom) || !sameCustomerProfileTime(before.EffectiveTo, after.EffectiveTo) {
		changed["lifecycle"] = true
	}
	if before.ID == uuid.Nil || !sameApproval(before.Approval, after.Approval) {
		changed["approval"] = true
	}
	return changed
}

func sameCustomerProfileTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sameApproval(left, right *ApprovalDecisionReference) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func sortedCustomerProfileChangedFields(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for key, changed := range values {
		if changed {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func (service *CustomerProfileService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrCustomerProfileAuthorizationDenied), errors.Is(commandErr, ErrCustomerProfileFieldAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrCustomerProfileVersionConflict), errors.Is(commandErr, ErrCustomerProfilePartyVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrCustomerProfilePartyNotFound):
		code = "PARTY_NOT_FOUND"
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

func decodeDurableCustomerProfileResult(body []byte) (CustomerProfileCommandResult, error) {
	var result CustomerProfileCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return CustomerProfileCommandResult{}, ErrCustomerProfileCommandInProgress
	}
	return result, nil
}

type MemoryCustomerProfileRepository struct {
	mu              sync.RWMutex
	profiles        map[uuid.UUID]CustomerProfile
	partyRepository PartyRepository
	audit           CustomerProfileAuditRecorder
}

func NewMemoryCustomerProfileRepository(partyRepository ...PartyRepository) *MemoryCustomerProfileRepository {
	var parties PartyRepository
	if len(partyRepository) > 0 {
		parties = partyRepository[0]
	}
	return &MemoryCustomerProfileRepository{profiles: make(map[uuid.UUID]CustomerProfile), partyRepository: parties}
}

func (repository *MemoryCustomerProfileRepository) BindCustomerProfileAuditRecorder(audit CustomerProfileAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryCustomerProfileRepository) Get(_ context.Context, id uuid.UUID) (CustomerProfile, error) {
	if repository == nil {
		return CustomerProfile{}, ErrInvalidCustomerProfileService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	profile, ok := repository.profiles[id]
	if !ok {
		return CustomerProfile{}, ErrCustomerProfileNotFound
	}
	return cloneCustomerProfile(profile), nil
}

func (repository *MemoryCustomerProfileRepository) List(_ context.Context, scopeID *uuid.UUID) ([]CustomerProfile, error) {
	if repository == nil {
		return nil, ErrInvalidCustomerProfileService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]CustomerProfile, 0, len(repository.profiles))
	for _, profile := range repository.profiles {
		if scopeID != nil && profile.ScopeID != *scopeID {
			continue
		}
		result = append(result, cloneCustomerProfile(profile))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

func (repository *MemoryCustomerProfileRepository) CommitCustomerProfileMutation(ctx context.Context, mutation CustomerProfileMutation) error {
	if repository == nil {
		return ErrInvalidCustomerProfileService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if repository.partyRepository == nil {
		return ErrCustomerProfilePartyNotFound
	}
	party, err := repository.partyRepository.Get(ctx, mutation.After.PartyID)
	if errors.Is(err, ErrPartyNotFound) {
		return ErrCustomerProfilePartyNotFound
	}
	if err != nil {
		return err
	}
	if party.ScopeID != mutation.After.ScopeID || strings.ToLower(string(party.PartyType)) != "customer" {
		return ErrCustomerProfilePartyInvalid
	}
	if mutation.ExpectedPartyVersion == nil || !party.Version.Matches(*mutation.ExpectedPartyVersion) {
		return ErrCustomerProfilePartyVersionConflict
	}
	if !mutation.After.PartyVersion.Matches(*mutation.ExpectedPartyVersion) {
		return ErrCustomerProfilePartyVersionConflict
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrCustomerProfileVersionConflict
		}
		for _, current := range repository.profiles {
			if current.ScopeID == mutation.After.ScopeID && current.PartyID == mutation.After.PartyID {
				return ErrCustomerProfileDuplicate
			}
		}
	} else {
		if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
			return ErrCustomerProfileVersionConflict
		}
		current, ok := repository.profiles[mutation.After.ID]
		if !ok {
			return ErrCustomerProfileNotFound
		}
		if !current.Version.Matches(*mutation.ExpectedVersion) || current.PartyID != mutation.After.PartyID {
			return ErrCustomerProfileVersionConflict
		}
	}
	if repository.audit == nil {
		return ErrCustomerProfileAuditUnavailable
	}
	if err := repository.audit.RecordCustomerProfileMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.profiles[mutation.After.ID] = cloneCustomerProfile(mutation.After)
	return nil
}

type MemoryCustomerProfileAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryCustomerProfileAuthorizer) AuthorizeCustomerProfile(context.Context, Actor, CustomerProfileCommand, *CustomerProfile) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryCustomerProfileFieldAuthorizer struct {
	Decision CustomerProfileFieldAuthorization
	Err      error
}

func (authorizer MemoryCustomerProfileFieldAuthorizer) AuthorizeCustomerProfileFields(context.Context, Actor, CustomerProfileCommand, *CustomerProfile) (CustomerProfileFieldAuthorization, error) {
	if authorizer.Err != nil {
		return CustomerProfileFieldAuthorization{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllCustomerProfileApprovalValidator struct{}

func (AllowAllCustomerProfileApprovalValidator) ValidateCustomerProfileApproval(context.Context, CustomerProfileCommand, CustomerProfile) error {
	return nil
}

type MemoryCustomerProfileAuditRecorder struct {
	mu      sync.Mutex
	Records []CustomerProfileAuditRecord
	Err     error
}

func (recorder *MemoryCustomerProfileAuditRecorder) RecordCustomerProfileMutation(_ context.Context, record CustomerProfileAuditRecord) error {
	if recorder == nil {
		return ErrCustomerProfileAuditUnavailable
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

var _ CustomerProfileRepository = (*MemoryCustomerProfileRepository)(nil)
var _ CustomerProfileAuthorizer = MemoryCustomerProfileAuthorizer{}
var _ CustomerProfileFieldAuthorizer = MemoryCustomerProfileFieldAuthorizer{}
var _ CustomerProfileApprovalValidator = AllowAllCustomerProfileApprovalValidator{}
var _ CustomerProfileAuditRecorder = (*MemoryCustomerProfileAuditRecorder)(nil)
