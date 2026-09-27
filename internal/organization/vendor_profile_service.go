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

type VendorProfileCommand struct {
	Action               string
	VendorProfileID      uuid.UUID
	ScopeID              uuid.UUID
	PartyID              uuid.UUID
	ExpectedPartyVersion *aggregateversion.AggregateVersion
	PaymentTerms         string
	WithholdingTreatment string
	RemittancePreference string
	EffectiveFrom        time.Time
	EffectiveTo          *time.Time
	Approval             *ApprovalDecisionReference
	ExpectedVersion      *aggregateversion.AggregateVersion
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command VendorProfileCommand) Validate() error {
	switch command.Action {
	case VendorProfileActionCreate:
		if command.VendorProfileID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include profile id or expected version", ErrInvalidVendorProfileCommand)
		}
		if command.EffectiveTo != nil {
			return fmt.Errorf("%w: create cannot include an end date; maintain the profile to end-date it", ErrInvalidVendorProfileCommand)
		}
	case VendorProfileActionMaintain:
		if command.VendorProfileID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: maintain profile id and expected version are required", ErrInvalidVendorProfileCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action", ErrInvalidVendorProfileCommand)
	}
	if command.ScopeID == uuid.Nil || command.PartyID == uuid.Nil || command.ExpectedPartyVersion == nil {
		return fmt.Errorf("%w: accounting scope, party, and expected party version are required", ErrInvalidVendorProfileCommand)
	}
	if err := validateVendorProfileCode(command.PaymentTerms, "payment terms"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidVendorProfileCommand, err)
	}
	if err := validateVendorProfileCode(command.WithholdingTreatment, "withholding treatment"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidVendorProfileCommand, err)
	}
	if err := validateVendorProfileCode(command.RemittancePreference, "remittance preference"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidVendorProfileCommand, err)
	}
	if command.EffectiveFrom.IsZero() || (command.EffectiveTo != nil && !command.EffectiveTo.After(command.EffectiveFrom)) {
		return fmt.Errorf("%w: effective interval is invalid", ErrInvalidVendorProfileCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidVendorProfileCommand)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidVendorProfileCommand)
		}
	}
	return nil
}

type VendorProfileAuthorizer interface {
	AuthorizeVendorProfile(context.Context, Actor, VendorProfileCommand, *VendorProfile) (AuthorizationDecision, error)
}

type VendorProfileFieldAuthorizer interface {
	AuthorizeVendorProfileFields(context.Context, Actor, VendorProfileCommand, *VendorProfile) (VendorProfileFieldAuthorization, error)
}

type VendorProfileApprovalValidator interface {
	ValidateVendorProfileApproval(context.Context, VendorProfileCommand, VendorProfile) error
}

type VendorProfileAuditRecord struct {
	VendorProfileID       uuid.UUID
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

type VendorProfileAuditRecorder interface {
	RecordVendorProfileMutation(context.Context, VendorProfileAuditRecord) error
}

type VendorProfileMutation struct {
	Before               VendorProfile
	After                VendorProfile
	ExpectedVersion      *aggregateversion.AggregateVersion
	ExpectedPartyVersion *aggregateversion.AggregateVersion
	Audit                VendorProfileAuditRecord
}

type VendorProfileRepository interface {
	Get(context.Context, uuid.UUID) (VendorProfile, error)
	List(context.Context, *uuid.UUID) ([]VendorProfile, error)
	CommitVendorProfileMutation(context.Context, VendorProfileMutation) error
}

type DurableVendorProfileRepository interface {
	CommitVendorProfileMutationWithIdempotency(context.Context, VendorProfileMutation, DurableVendorProfileMutationCommit) error
}

type DurableVendorProfileMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableVendorProfileServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type VendorProfileCommandResult struct {
	VendorProfile     SafeVendorProfile `json:"vendorProfile"`
	DecisionReference uuid.UUID         `json:"decisionReference"`
	PolicyReference   string            `json:"policyReference"`
	ValidationOutcome string            `json:"validationOutcome"`
	Replayed          bool              `json:"replayed,omitempty"`
}

type VendorProfileService struct {
	repository      VendorProfileRepository
	partyRepository PartyRepository
	authorizer      VendorProfileAuthorizer
	fieldAuthorizer VendorProfileFieldAuthorizer
	approval        VendorProfileApprovalValidator
	audit           VendorProfileAuditRecorder
	clock           func() time.Time
	durable         *DurableVendorProfileServiceConfig
	mu              sync.Mutex
	idempotency     map[string]storedVendorProfileCommand
}

type storedVendorProfileCommand struct {
	fingerprint string
	result      VendorProfileCommandResult
}

func NewVendorProfileService(repository VendorProfileRepository, partyRepository PartyRepository, authorizer VendorProfileAuthorizer, fieldAuthorizer VendorProfileFieldAuthorizer, approval VendorProfileApprovalValidator, audit VendorProfileAuditRecorder, clock func() time.Time) (*VendorProfileService, error) {
	return newVendorProfileService(repository, partyRepository, authorizer, fieldAuthorizer, approval, audit, clock, nil)
}

func NewVendorProfileServiceWithDurableIdempotency(repository VendorProfileRepository, partyRepository PartyRepository, authorizer VendorProfileAuthorizer, fieldAuthorizer VendorProfileFieldAuthorizer, approval VendorProfileApprovalValidator, audit VendorProfileAuditRecorder, clock func() time.Time, durable DurableVendorProfileServiceConfig) (*VendorProfileService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidVendorProfileService
	}
	return newVendorProfileService(repository, partyRepository, authorizer, fieldAuthorizer, approval, audit, clock, &durable)
}

func newVendorProfileService(repository VendorProfileRepository, partyRepository PartyRepository, authorizer VendorProfileAuthorizer, fieldAuthorizer VendorProfileFieldAuthorizer, approval VendorProfileApprovalValidator, audit VendorProfileAuditRecorder, clock func() time.Time, durable *DurableVendorProfileServiceConfig) (*VendorProfileService, error) {
	if repository == nil || partyRepository == nil || authorizer == nil || fieldAuthorizer == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidVendorProfileService
	}
	if binder, ok := repository.(interface {
		BindVendorProfileAuditRecorder(VendorProfileAuditRecorder)
	}); ok {
		binder.BindVendorProfileAuditRecorder(audit)
	}
	return &VendorProfileService{repository: repository, partyRepository: partyRepository, authorizer: authorizer, fieldAuthorizer: fieldAuthorizer, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedVendorProfileCommand)}, nil
}

func (service *VendorProfileService) Execute(ctx context.Context, actor Actor, command VendorProfileCommand) (VendorProfileCommandResult, error) {
	if service == nil {
		return VendorProfileCommandResult{}, ErrInvalidVendorProfileService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return VendorProfileCommandResult{}, ErrVendorProfileAuthorizationDenied
	}
	if err := command.Validate(); err != nil {
		return VendorProfileCommandResult{}, err
	}
	fingerprint, err := vendorProfileCommandFingerprint(command)
	if err != nil {
		return VendorProfileCommandResult{}, err
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
			return VendorProfileCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return VendorProfileCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return VendorProfileCommandResult{}, ErrVendorProfileIdempotencyConflict
		}
		if err != nil {
			return VendorProfileCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return VendorProfileCommandResult{}, ErrVendorProfileCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return VendorProfileCommandResult{}, ErrVendorProfileDurableCommandFailed
			}
			result, decodeErr := decodeDurableVendorProfileResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return VendorProfileCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return VendorProfileCommandResult{}, ErrVendorProfileIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	var current *VendorProfile
	if command.Action == VendorProfileActionMaintain {
		loaded, getErr := service.repository.Get(ctx, command.VendorProfileID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return VendorProfileCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfileVersionConflict)
			return VendorProfileCommandResult{}, ErrVendorProfileVersionConflict
		}
		if current.ScopeID != command.ScopeID {
			service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfileAuthorizationDenied)
			return VendorProfileCommandResult{}, ErrVendorProfileAuthorizationDenied
		}
		if current.PartyID != command.PartyID {
			service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfilePartyMismatch)
			return VendorProfileCommandResult{}, ErrVendorProfilePartyMismatch
		}
	}
	party, err := service.partyRepository.Get(ctx, command.PartyID)
	if errors.Is(err, ErrPartyNotFound) {
		err = ErrVendorProfilePartyNotFound
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return VendorProfileCommandResult{}, err
	}
	if party.ScopeID != command.ScopeID || strings.ToLower(string(party.PartyType)) != "vendor" {
		service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfilePartyInvalid)
		return VendorProfileCommandResult{}, ErrVendorProfilePartyInvalid
	}
	if !party.Version.Matches(*command.ExpectedPartyVersion) {
		service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfilePartyVersionConflict)
		return VendorProfileCommandResult{}, ErrVendorProfilePartyVersionConflict
	}
	decision, err := service.authorizer.AuthorizeVendorProfile(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return VendorProfileCommandResult{}, err
	}
	if err := authorizeVendorProfileDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return VendorProfileCommandResult{}, err
	}
	fieldAccess, err := service.fieldAuthorizer.AuthorizeVendorProfileFields(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfileFieldAuthorizationUnavailable)
		return VendorProfileCommandResult{}, ErrVendorProfileFieldAuthorizationUnavailable
	}
	if !fieldAccess.all() {
		service.finalizeDurableFailure(ctx, acquisition, ErrVendorProfileFieldAuthorizationDenied)
		return VendorProfileCommandResult{}, ErrVendorProfileFieldAuthorizationDenied
	}

	now := service.clock().UTC()
	var before, after VendorProfile
	if command.Action == VendorProfileActionCreate {
		before = VendorProfile{}
		status := VendorProfileStatusActive
		if decision.ApprovalRequired {
			status = VendorProfileStatusDraft
		}
		newID, idErr := platformidentity.NewAggregateID()
		if idErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, idErr)
			return VendorProfileCommandResult{}, idErr
		}
		after, err = NewVendorProfile(newID.UUID(), command.ScopeID, command.PartyID, party.Version, command.PaymentTerms, command.WithholdingTreatment, command.RemittancePreference, command.EffectiveFrom, command.Approval, status, now)
	} else {
		before = cloneVendorProfile(*current)
		after = cloneVendorProfile(*current)
		err = after.Replace(*current, party.Version, command.PaymentTerms, command.WithholdingTreatment, command.RemittancePreference, command.EffectiveFrom, command.EffectiveTo, command.Approval, now)
	}
	if err == nil && decision.ApprovalRequired {
		if command.Approval == nil {
			err = ErrVendorProfileApprovalRequired
		} else {
			err = service.approval.ValidateVendorProfileApproval(ctx, command, after)
		}
	} else if err == nil && command.Approval != nil {
		err = service.approval.ValidateVendorProfileApproval(ctx, command, after)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return VendorProfileCommandResult{}, err
	}
	changedFields := vendorProfileChangedFields(before, after)
	result := VendorProfileCommandResult{VendorProfile: after.SafeProjectionWithAccess(fieldAccess), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "accepted"}
	record := VendorProfileAuditRecord{VendorProfileID: after.ID, PartyID: after.PartyID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, ScopeID: after.ScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Approval: cloneApproval(command.Approval), RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintVendorProfile(before), AfterFingerprint: FingerprintVendorProfile(after), ChangedFields: sortedVendorProfileChangedFields(changedFields), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	mutation := VendorProfileMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, ExpectedPartyVersion: command.ExpectedPartyVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableVendorProfileRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidVendorProfileService)
			return VendorProfileCommandResult{}, ErrInvalidVendorProfileService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return VendorProfileCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return VendorProfileCommandResult{}, metadataErr
		}
		err = committer.CommitVendorProfileMutationWithIdempotency(ctx, mutation, DurableVendorProfileMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitVendorProfileMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return VendorProfileCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedVendorProfileCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func authorizeVendorProfileDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	switch strings.ToLower(strings.TrimSpace(decision.Outcome)) {
	case "stale":
		return ErrVendorProfileAuthorizationStale
	case "unavailable":
		return ErrVendorProfileAuthorizationUnavailable
	}
	if !decision.Allowed || decision.Permission != VendorProfileManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrVendorProfileAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrVendorProfileAuthorizationDenied
}

func vendorProfileCommandFingerprint(command VendorProfileCommand) (string, error) {
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("sha256:%x", sum[:]), nil
}

func vendorProfileChangedFields(before, after VendorProfile) map[string]bool {
	changed := make(map[string]bool)
	if before.ID == uuid.Nil || before.PartyID != after.PartyID || before.PartyVersion != after.PartyVersion {
		changed["party_reference"] = true
	}
	if before.ID == uuid.Nil || before.PaymentTerms != after.PaymentTerms {
		changed["payment_terms"] = true
	}
	if before.ID == uuid.Nil || before.WithholdingTreatment != after.WithholdingTreatment {
		changed["withholding_treatment"] = true
	}
	if before.ID == uuid.Nil || before.RemittancePreference != after.RemittancePreference {
		changed["remittance_preference"] = true
	}
	if before.ID == uuid.Nil || before.Status != after.Status || !before.EffectiveFrom.Equal(after.EffectiveFrom) || !sameVendorProfileTime(before.EffectiveTo, after.EffectiveTo) {
		changed["lifecycle"] = true
	}
	if before.ID == uuid.Nil || !sameApproval(before.Approval, after.Approval) {
		changed["approval"] = true
	}
	return changed
}

func sameVendorProfileTime(left, right *time.Time) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func sortedVendorProfileChangedFields(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for key, changed := range values {
		if changed {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func (service *VendorProfileService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrVendorProfileAuthorizationDenied), errors.Is(commandErr, ErrVendorProfileFieldAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrVendorProfileVersionConflict), errors.Is(commandErr, ErrVendorProfilePartyVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrVendorProfilePartyNotFound):
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

func decodeDurableVendorProfileResult(body []byte) (VendorProfileCommandResult, error) {
	var result VendorProfileCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return VendorProfileCommandResult{}, ErrVendorProfileCommandInProgress
	}
	return result, nil
}

type MemoryVendorProfileRepository struct {
	mu              sync.RWMutex
	profiles        map[uuid.UUID]VendorProfile
	partyRepository PartyRepository
	audit           VendorProfileAuditRecorder
}

func NewMemoryVendorProfileRepository(partyRepository ...PartyRepository) *MemoryVendorProfileRepository {
	var parties PartyRepository
	if len(partyRepository) > 0 {
		parties = partyRepository[0]
	}
	return &MemoryVendorProfileRepository{profiles: make(map[uuid.UUID]VendorProfile), partyRepository: parties}
}

func (repository *MemoryVendorProfileRepository) BindVendorProfileAuditRecorder(audit VendorProfileAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryVendorProfileRepository) Get(_ context.Context, id uuid.UUID) (VendorProfile, error) {
	if repository == nil {
		return VendorProfile{}, ErrInvalidVendorProfileService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	profile, ok := repository.profiles[id]
	if !ok {
		return VendorProfile{}, ErrVendorProfileNotFound
	}
	return cloneVendorProfile(profile), nil
}

func (repository *MemoryVendorProfileRepository) List(_ context.Context, scopeID *uuid.UUID) ([]VendorProfile, error) {
	if repository == nil {
		return nil, ErrInvalidVendorProfileService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]VendorProfile, 0, len(repository.profiles))
	for _, profile := range repository.profiles {
		if scopeID != nil && profile.ScopeID != *scopeID {
			continue
		}
		result = append(result, cloneVendorProfile(profile))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

func (repository *MemoryVendorProfileRepository) CommitVendorProfileMutation(ctx context.Context, mutation VendorProfileMutation) error {
	if repository == nil {
		return ErrInvalidVendorProfileService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if repository.partyRepository == nil {
		return ErrVendorProfilePartyNotFound
	}
	party, err := repository.partyRepository.Get(ctx, mutation.After.PartyID)
	if errors.Is(err, ErrPartyNotFound) {
		return ErrVendorProfilePartyNotFound
	}
	if err != nil {
		return err
	}
	if party.ScopeID != mutation.After.ScopeID || strings.ToLower(string(party.PartyType)) != "vendor" {
		return ErrVendorProfilePartyInvalid
	}
	if mutation.ExpectedPartyVersion == nil || !party.Version.Matches(*mutation.ExpectedPartyVersion) || !mutation.After.PartyVersion.Matches(*mutation.ExpectedPartyVersion) {
		return ErrVendorProfilePartyVersionConflict
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrVendorProfileVersionConflict
		}
		for _, current := range repository.profiles {
			if current.ScopeID == mutation.After.ScopeID && current.PartyID == mutation.After.PartyID {
				return ErrVendorProfileDuplicate
			}
		}
	} else {
		if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
			return ErrVendorProfileVersionConflict
		}
		current, ok := repository.profiles[mutation.After.ID]
		if !ok {
			return ErrVendorProfileNotFound
		}
		if !current.Version.Matches(*mutation.ExpectedVersion) || current.PartyID != mutation.After.PartyID {
			return ErrVendorProfileVersionConflict
		}
	}
	if repository.audit == nil {
		return ErrVendorProfileAuditUnavailable
	}
	if err := repository.audit.RecordVendorProfileMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.profiles[mutation.After.ID] = cloneVendorProfile(mutation.After)
	return nil
}

type MemoryVendorProfileAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryVendorProfileAuthorizer) AuthorizeVendorProfile(context.Context, Actor, VendorProfileCommand, *VendorProfile) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryVendorProfileFieldAuthorizer struct {
	Decision VendorProfileFieldAuthorization
	Err      error
}

func (authorizer MemoryVendorProfileFieldAuthorizer) AuthorizeVendorProfileFields(context.Context, Actor, VendorProfileCommand, *VendorProfile) (VendorProfileFieldAuthorization, error) {
	if authorizer.Err != nil {
		return VendorProfileFieldAuthorization{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllVendorProfileApprovalValidator struct{}

func (AllowAllVendorProfileApprovalValidator) ValidateVendorProfileApproval(context.Context, VendorProfileCommand, VendorProfile) error {
	return nil
}

type MemoryVendorProfileAuditRecorder struct {
	mu      sync.Mutex
	Records []VendorProfileAuditRecord
	Err     error
}

func (recorder *MemoryVendorProfileAuditRecorder) RecordVendorProfileMutation(_ context.Context, record VendorProfileAuditRecord) error {
	if recorder == nil {
		return ErrVendorProfileAuditUnavailable
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

var _ VendorProfileRepository = (*MemoryVendorProfileRepository)(nil)
var _ VendorProfileAuthorizer = MemoryVendorProfileAuthorizer{}
var _ VendorProfileFieldAuthorizer = MemoryVendorProfileFieldAuthorizer{}
var _ VendorProfileApprovalValidator = AllowAllVendorProfileApprovalValidator{}
var _ VendorProfileAuditRecorder = (*MemoryVendorProfileAuditRecorder)(nil)
