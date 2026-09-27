package organization

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
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

type PartyCommand struct {
	Action               string
	PartyID              uuid.UUID
	ScopeID              uuid.UUID
	Name                 string
	PartyType            PartyType
	Status               PartyStatus
	TaxIdentifier        *string
	ContactMethods       []PartyContactMethod
	Addresses            []PartyAddress
	Classifications      []PartyClassification
	BankDetailReferences []PartyBankDetailReference
	Approval             *ApprovalDecisionReference
	ExpectedVersion      *aggregateversion.AggregateVersion
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command PartyCommand) Validate() error {
	switch command.Action {
	case PartyActionCreate:
		if command.PartyID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include id or expected version", ErrInvalidPartyCommand)
		}
	case PartyActionMaintain:
		if command.PartyID == uuid.Nil || command.ExpectedVersion == nil {
			return fmt.Errorf("%w: maintain id and expected version are required", ErrInvalidPartyCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action", ErrInvalidPartyCommand)
	}
	if command.ScopeID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope is required", ErrInvalidPartyCommand)
	}
	if canonicalPartyText(command.Name) == "" || canonicalPartyText(command.Name) != command.Name {
		return fmt.Errorf("%w: party name is required and must be canonical", ErrInvalidPartyCommand)
	}
	if err := validatePartyCode(string(command.PartyType), "party type"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPartyCommand, err)
	}
	if err := validatePartyCode(string(command.Status), "party status"); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPartyCommand, err)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidPartyCommand)
	}
	if command.TaxIdentifier != nil && strings.TrimSpace(*command.TaxIdentifier) != *command.TaxIdentifier {
		return fmt.Errorf("%w: restricted tax identifier is not canonical", ErrInvalidPartyCommand)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidPartyCommand)
		}
	}
	return nil
}

type PartyFieldAuthorization struct {
	Identity                bool
	RestrictedTaxIdentifier bool
	PersonalData            bool
	Classifications         bool
	BankDetailReferences    bool
}

func (authorization PartyFieldAuthorization) all() bool {
	return authorization.Identity && authorization.RestrictedTaxIdentifier && authorization.PersonalData && authorization.Classifications && authorization.BankDetailReferences
}

type PartyAuthorizer interface {
	AuthorizeParty(context.Context, Actor, PartyCommand, *Party) (AuthorizationDecision, error)
}

type PartyFieldAuthorizer interface {
	AuthorizePartyFields(context.Context, Actor, PartyCommand, *Party) (PartyFieldAuthorization, error)
}

type PartyBankReferenceValidationRequest struct {
	ScopeID    uuid.UUID
	PartyID    uuid.UUID
	PartyType  PartyType
	References []PartyBankDetailReference
}

type PartyBankReferenceValidator interface {
	ValidateAndCanonicalizePartyBankReferences(context.Context, PartyBankReferenceValidationRequest) ([]PartyBankDetailReference, error)
}

type PartyBankControlRequest struct {
	ScopeID    uuid.UUID
	PartyID    uuid.UUID
	PartyType  PartyType
	References []PartyBankDetailReference
	Approval   *ApprovalDecisionReference
}

type PartyBankControlEvaluator interface {
	EvaluatePartyBankControl(context.Context, PartyBankControlRequest) (BankDetailControlState, error)
}

type PartyAuditRecord struct {
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

type PartyAuditRecorder interface {
	RecordPartyMutation(context.Context, PartyAuditRecord) error
}

type PartyMutation struct {
	Before          Party
	After           Party
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           PartyAuditRecord
}

type PartyRepository interface {
	Get(context.Context, uuid.UUID) (Party, error)
	CommitPartyMutation(context.Context, PartyMutation) error
}

type DurablePartyRepository interface {
	CommitPartyMutationWithIdempotency(context.Context, PartyMutation, DurablePartyMutationCommit) error
}

type DurablePartyMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurablePartyServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type PartyCommandResult struct {
	Party             SafeParty              `json:"party"`
	DecisionReference uuid.UUID              `json:"decisionReference"`
	PolicyReference   string                 `json:"policyReference"`
	ValidationOutcome string                 `json:"validationOutcome"`
	BankControl       BankDetailControlState `json:"bankControl"`
	Replayed          bool                   `json:"replayed,omitempty"`
}

type PartyService struct {
	repository      PartyRepository
	authorizer      PartyAuthorizer
	fieldAuthorizer PartyFieldAuthorizer
	bankReferences  PartyBankReferenceValidator
	bankControl     PartyBankControlEvaluator
	audit           PartyAuditRecorder
	clock           func() time.Time
	durable         *DurablePartyServiceConfig
	mu              sync.Mutex
	idempotency     map[string]storedPartyCommand
}

type storedPartyCommand struct {
	fingerprint string
	result      PartyCommandResult
}

func NewPartyService(repository PartyRepository, authorizer PartyAuthorizer, fieldAuthorizer PartyFieldAuthorizer, bankReferences PartyBankReferenceValidator, bankControl PartyBankControlEvaluator, audit PartyAuditRecorder, clock func() time.Time) (*PartyService, error) {
	return newPartyService(repository, authorizer, fieldAuthorizer, bankReferences, bankControl, audit, clock, nil)
}

func NewPartyServiceWithDurableIdempotency(repository PartyRepository, authorizer PartyAuthorizer, fieldAuthorizer PartyFieldAuthorizer, bankReferences PartyBankReferenceValidator, bankControl PartyBankControlEvaluator, audit PartyAuditRecorder, clock func() time.Time, durable DurablePartyServiceConfig) (*PartyService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidPartyService
	}
	return newPartyService(repository, authorizer, fieldAuthorizer, bankReferences, bankControl, audit, clock, &durable)
}

func newPartyService(repository PartyRepository, authorizer PartyAuthorizer, fieldAuthorizer PartyFieldAuthorizer, bankReferences PartyBankReferenceValidator, bankControl PartyBankControlEvaluator, audit PartyAuditRecorder, clock func() time.Time, durable *DurablePartyServiceConfig) (*PartyService, error) {
	if repository == nil || authorizer == nil || fieldAuthorizer == nil || bankReferences == nil || bankControl == nil || audit == nil || clock == nil {
		return nil, ErrInvalidPartyService
	}
	if binder, ok := repository.(interface{ BindAuditRecorder(PartyAuditRecorder) }); ok {
		binder.BindAuditRecorder(audit)
	}
	return &PartyService{repository: repository, authorizer: authorizer, fieldAuthorizer: fieldAuthorizer, bankReferences: bankReferences, bankControl: bankControl, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedPartyCommand)}, nil
}

func (service *PartyService) Execute(ctx context.Context, actor Actor, command PartyCommand) (PartyCommandResult, error) {
	if service == nil {
		return PartyCommandResult{}, ErrInvalidPartyService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return PartyCommandResult{}, ErrPartyAuthorizationDenied
	}
	if err := command.Validate(); err != nil {
		return PartyCommandResult{}, err
	}
	fingerprint, err := partyCommandFingerprint(command)
	if err != nil {
		return PartyCommandResult{}, err
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
			return PartyCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return PartyCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return PartyCommandResult{}, ErrPartyIdempotencyConflict
		}
		if err != nil {
			return PartyCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return PartyCommandResult{}, ErrPartyCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return PartyCommandResult{}, ErrPartyDurableCommandFailed
			}
			result, decodeErr := decodeDurablePartyResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return PartyCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return PartyCommandResult{}, ErrPartyIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	var current *Party
	if command.Action == PartyActionMaintain {
		loaded, getErr := service.repository.Get(ctx, command.PartyID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return PartyCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrPartyVersionConflict)
			return PartyCommandResult{}, ErrPartyVersionConflict
		}
		if current.ScopeID != command.ScopeID {
			service.finalizeDurableFailure(ctx, acquisition, ErrPartyAuthorizationDenied)
			return PartyCommandResult{}, ErrPartyAuthorizationDenied
		}
	}
	decision, err := service.authorizer.AuthorizeParty(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return PartyCommandResult{}, err
	}
	if err := authorizePartyDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return PartyCommandResult{}, err
	}
	fieldAccess, err := service.fieldAuthorizer.AuthorizePartyFields(ctx, actor, command, current)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, ErrPartyFieldAuthorizationUnavailable)
		return PartyCommandResult{}, ErrPartyFieldAuthorizationUnavailable
	}

	now := service.clock().UTC()
	var before, after Party
	if command.Action == PartyActionCreate {
		before = Party{}
		bankControl := BankDetailControlState{Status: PartyBankControlNotRequired}
		if len(command.BankDetailReferences) > 0 {
			bankControl = BankDetailControlState{}
		}
		bankReferences, referenceErr := service.canonicalizeBankReferences(ctx, command, uuid.Nil, command.BankDetailReferences, fieldAccess)
		if referenceErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, referenceErr)
			return PartyCommandResult{}, referenceErr
		}
		if command.BankDetailReferences != nil {
			command.BankDetailReferences = bankReferences
		}
		if len(command.BankDetailReferences) > 0 {
			bankControl, err = service.evaluateBankControl(ctx, command, uuid.Nil, bankReferences)
			if err != nil {
				service.finalizeDurableFailure(ctx, acquisition, err)
				return PartyCommandResult{}, err
			}
		}
		taxIdentifier := ""
		if command.TaxIdentifier != nil {
			taxIdentifier = *command.TaxIdentifier
		}
		if !fieldAccess.Identity || (command.TaxIdentifier != nil && !fieldAccess.RestrictedTaxIdentifier) || ((command.ContactMethods != nil || command.Addresses != nil) && !fieldAccess.PersonalData) || (command.Classifications != nil && !fieldAccess.Classifications) || (command.BankDetailReferences != nil && !fieldAccess.BankDetailReferences) {
			err = ErrPartyFieldAuthorizationDenied
		} else {
			after, err = NewParty(uuid.New(), command.ScopeID, command.Name, command.PartyType, command.Status, taxIdentifier, command.ContactMethods, command.Addresses, command.Classifications, command.BankDetailReferences, bankControl, now)
		}
	} else {
		before = cloneParty(*current)
		candidate := cloneParty(*current)
		candidate.Name, candidate.PartyType, candidate.Status = canonicalPartyText(command.Name), PartyType(canonicalPartyCode(string(command.PartyType))), PartyStatus(canonicalPartyCode(string(command.Status)))
		if command.TaxIdentifier != nil {
			candidate.TaxIdentifier = strings.TrimSpace(*command.TaxIdentifier)
		}
		if command.ContactMethods != nil {
			candidate.ContactMethods = newPartyContactIDs(command.ContactMethods)
		}
		if command.Addresses != nil {
			candidate.Addresses = newPartyAddressIDs(command.Addresses)
		}
		if command.Classifications != nil {
			candidate.Classifications = newPartyClassificationIDs(command.Classifications)
		}
		if command.BankDetailReferences != nil {
			candidate.BankDetailReferences = newPartyBankReferenceIDs(command.BankDetailReferences)
		}
		changed := partyChangedFields(before, candidate)
		if !fieldAccess.Identity && changed["identity"] {
			err = ErrPartyFieldAuthorizationDenied
		} else if !fieldAccess.RestrictedTaxIdentifier && changed["tax"] {
			err = ErrPartyFieldAuthorizationDenied
		} else if !fieldAccess.PersonalData && changed["personal"] {
			err = ErrPartyFieldAuthorizationDenied
		} else if !fieldAccess.Classifications && changed["classifications"] {
			err = ErrPartyFieldAuthorizationDenied
		} else if !fieldAccess.BankDetailReferences && changed["bank_references"] {
			err = ErrPartyFieldAuthorizationDenied
		} else {
			if command.BankDetailReferences != nil {
				candidate.BankDetailReferences, err = service.canonicalizeBankReferences(ctx, command, current.ID, candidate.BankDetailReferences, fieldAccess)
			}
			if err == nil && command.BankDetailReferences != nil && !equalPartyBankReferences(before.BankDetailReferences, candidate.BankDetailReferences) {
				candidate.BankControl, err = service.evaluateBankControl(ctx, command, current.ID, candidate.BankDetailReferences)
			}
			if err == nil {
				after = cloneParty(*current)
				err = after.Replace(*current, candidate.Name, candidate.PartyType, candidate.Status, candidate.TaxIdentifier, candidate.ContactMethods, candidate.Addresses, candidate.Classifications, candidate.BankDetailReferences, candidate.BankControl, now)
			}
		}
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return PartyCommandResult{}, err
	}
	changedFields := partyChangedFields(before, after)
	result := PartyCommandResult{Party: after.SafeProjectionWithAccess(fieldAccess), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "accepted", BankControl: cloneBankControl(after.BankControl)}
	record := PartyAuditRecord{PartyID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, ScopeID: after.ScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Approval: cloneApproval(command.Approval), RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintParty(before), AfterFingerprint: FingerprintParty(after), ChangedFields: sortedPartyChangedFields(changedFields), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	mutation := PartyMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurablePartyRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidPartyService)
			return PartyCommandResult{}, ErrInvalidPartyService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return PartyCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return PartyCommandResult{}, metadataErr
		}
		err = committer.CommitPartyMutationWithIdempotency(ctx, mutation, DurablePartyMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitPartyMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return PartyCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedPartyCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func (service *PartyService) canonicalizeBankReferences(ctx context.Context, command PartyCommand, partyID uuid.UUID, references []PartyBankDetailReference, access PartyFieldAuthorization) ([]PartyBankDetailReference, error) {
	if len(references) == 0 && command.Action == PartyActionCreate {
		return nil, nil
	}
	if !access.BankDetailReferences && len(references) > 0 {
		return nil, ErrPartyFieldAuthorizationDenied
	}
	result, err := service.bankReferences.ValidateAndCanonicalizePartyBankReferences(ctx, PartyBankReferenceValidationRequest{ScopeID: command.ScopeID, PartyID: partyID, PartyType: command.PartyType, References: clonePartyBankReferences(references)})
	if err != nil {
		if errors.Is(err, ErrPartyBankReferenceUnavailable) {
			return nil, ErrPartyBankReferenceUnavailable
		}
		return nil, ErrPartyBankReferenceInvalid
	}
	if err := validatePartyBankReferences(newPartyBankReferenceIDs(result)); err != nil {
		return nil, err
	}
	return newPartyBankReferenceIDs(result), nil
}

func (service *PartyService) evaluateBankControl(ctx context.Context, command PartyCommand, partyID uuid.UUID, references []PartyBankDetailReference) (BankDetailControlState, error) {
	state, err := service.bankControl.EvaluatePartyBankControl(ctx, PartyBankControlRequest{ScopeID: command.ScopeID, PartyID: partyID, PartyType: command.PartyType, References: clonePartyBankReferences(references), Approval: cloneApproval(command.Approval)})
	if err != nil {
		return BankDetailControlState{}, ErrPartyBankControlUnavailable
	}
	if err := state.Validate(); err != nil {
		return BankDetailControlState{}, ErrPartyBankControlUnavailable
	}
	switch state.Status {
	case PartyBankControlRejected:
		return BankDetailControlState{}, ErrPartyBankApprovalRejected
	case PartyBankControlStale:
		return BankDetailControlState{}, ErrPartyBankApprovalStale
	case PartyBankControlUnavailable:
		return BankDetailControlState{}, ErrPartyBankControlUnavailable
	}
	return cloneBankControl(state), nil
}

func authorizePartyDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	switch strings.ToLower(strings.TrimSpace(decision.Outcome)) {
	case "stale":
		return ErrPartyAuthorizationStale
	case "unavailable":
		return ErrPartyAuthorizationUnavailable
	}
	if !decision.Allowed || decision.Permission != PartyManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrPartyAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrPartyAuthorizationDenied
}

func partyCommandFingerprint(command PartyCommand) (string, error) {
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func partyChangedFields(before, after Party) map[string]bool {
	changed := make(map[string]bool)
	if before.ID == uuid.Nil || before.Name != after.Name || before.PartyType != after.PartyType || before.Status != after.Status {
		changed["identity"] = true
	}
	if before.ID != uuid.Nil && before.TaxIdentifier != after.TaxIdentifier {
		changed["tax"] = true
	}
	if before.ID == uuid.Nil || !equalPartyContacts(before.ContactMethods, after.ContactMethods) || !equalPartyAddresses(before.Addresses, after.Addresses) {
		changed["personal"] = true
	}
	if before.ID == uuid.Nil || !equalPartyClassifications(before.Classifications, after.Classifications) {
		changed["classifications"] = true
	}
	if before.ID == uuid.Nil || !equalPartyBankReferences(before.BankDetailReferences, after.BankDetailReferences) {
		changed["bank_references"] = true
	}
	return changed
}

func sortedPartyChangedFields(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for key, changed := range values {
		if changed {
			result = append(result, key)
		}
	}
	sort.Strings(result)
	return result
}

func equalPartyContacts(left, right []PartyContactMethod) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Type != right[index].Type || left[index].Value != right[index].Value || left[index].Label != right[index].Label {
			return false
		}
	}
	return true
}
func equalPartyAddresses(left, right []PartyAddress) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Type != right[index].Type || left[index].Line1 != right[index].Line1 || left[index].Line2 != right[index].Line2 || left[index].Locality != right[index].Locality || left[index].Region != right[index].Region || left[index].PostalCode != right[index].PostalCode || left[index].CountryCode != right[index].CountryCode {
			return false
		}
	}
	return true
}
func equalPartyClassifications(left, right []PartyClassification) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Code != right[index].Code || left[index].Value != right[index].Value {
			return false
		}
	}
	return true
}
func equalPartyBankReferences(left, right []PartyBankDetailReference) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].Reference != right[index].Reference || left[index].ProviderCode != right[index].ProviderCode || left[index].ConsentReference != right[index].ConsentReference {
			return false
		}
	}
	return true
}

func (service *PartyService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrPartyAuthorizationDenied), errors.Is(commandErr, ErrPartyFieldAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrPartyVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrPartyNotFound):
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

func decodeDurablePartyResult(body []byte) (PartyCommandResult, error) {
	var result PartyCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return PartyCommandResult{}, ErrPartyCommandInProgress
	}
	return result, nil
}

type MemoryPartyRepository struct {
	mu      sync.RWMutex
	parties map[uuid.UUID]Party
	audit   PartyAuditRecorder
}

func NewMemoryPartyRepository() *MemoryPartyRepository {
	return &MemoryPartyRepository{parties: make(map[uuid.UUID]Party)}
}

func (repository *MemoryPartyRepository) BindAuditRecorder(audit PartyAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryPartyRepository) Get(_ context.Context, id uuid.UUID) (Party, error) {
	if repository == nil {
		return Party{}, ErrInvalidPartyService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	party, ok := repository.parties[id]
	if !ok {
		return Party{}, ErrPartyNotFound
	}
	return cloneParty(party), nil
}

func (repository *MemoryPartyRepository) List(_ context.Context, scopeID *uuid.UUID) ([]Party, error) {
	if repository == nil {
		return nil, ErrInvalidPartyService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]Party, 0, len(repository.parties))
	for _, party := range repository.parties {
		if scopeID != nil && party.ScopeID != *scopeID {
			continue
		}
		result = append(result, cloneParty(party))
	}
	sortParties(result)
	return result, nil
}

func (repository *MemoryPartyRepository) CommitPartyMutation(ctx context.Context, mutation PartyMutation) error {
	if repository == nil {
		return ErrInvalidPartyService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != aggregateversion.Initial().Value() || mutation.After.RevisionNumber != 1 {
			return ErrPartyVersionConflict
		}
	} else {
		if mutation.After.ID != mutation.Before.ID || mutation.ExpectedVersion == nil || mutation.Before.Version.Value() != mutation.ExpectedVersion.Value() || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
			return ErrPartyVersionConflict
		}
		current, ok := repository.parties[mutation.After.ID]
		if !ok {
			return ErrPartyNotFound
		}
		if !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrPartyVersionConflict
		}
	}
	for id, current := range repository.parties {
		if id != mutation.After.ID && current.ScopeID == mutation.After.ScopeID && strings.EqualFold(current.Name, mutation.After.Name) {
			return ErrPartyDuplicate
		}
	}
	if repository.audit == nil {
		return ErrPartyAuditUnavailable
	}
	if err := repository.audit.RecordPartyMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.parties[mutation.After.ID] = cloneParty(mutation.After)
	return nil
}

type MemoryPartyAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryPartyAuthorizer) AuthorizeParty(context.Context, Actor, PartyCommand, *Party) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryPartyFieldAuthorizer struct {
	Decision PartyFieldAuthorization
	Err      error
}

func (authorizer MemoryPartyFieldAuthorizer) AuthorizePartyFields(context.Context, Actor, PartyCommand, *Party) (PartyFieldAuthorization, error) {
	if authorizer.Err != nil {
		return PartyFieldAuthorization{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllPartyFieldAuthorizer struct{}

func (AllowAllPartyFieldAuthorizer) AuthorizePartyFields(context.Context, Actor, PartyCommand, *Party) (PartyFieldAuthorization, error) {
	return PartyFieldAuthorization{Identity: true, RestrictedTaxIdentifier: true, PersonalData: true, Classifications: true, BankDetailReferences: true}, nil
}

type AllowAllPartyBankReferenceValidator struct{}

func (AllowAllPartyBankReferenceValidator) ValidateAndCanonicalizePartyBankReferences(_ context.Context, request PartyBankReferenceValidationRequest) ([]PartyBankDetailReference, error) {
	values := newPartyBankReferenceIDs(request.References)
	if err := validatePartyBankReferences(values); err != nil {
		return nil, err
	}
	return values, nil
}

type AllowAllPartyBankControlEvaluator struct{}

func (AllowAllPartyBankControlEvaluator) EvaluatePartyBankControl(context.Context, PartyBankControlRequest) (BankDetailControlState, error) {
	return BankDetailControlState{Status: PartyBankControlApproved}, nil
}

type MemoryPartyBankControlEvaluator struct {
	State BankDetailControlState
	Err   error
}

func (evaluator MemoryPartyBankControlEvaluator) EvaluatePartyBankControl(context.Context, PartyBankControlRequest) (BankDetailControlState, error) {
	if evaluator.Err != nil {
		return BankDetailControlState{}, evaluator.Err
	}
	return evaluator.State, nil
}

type MemoryPartyAuditRecorder struct {
	mu      sync.Mutex
	Records []PartyAuditRecord
	Err     error
}

func (recorder *MemoryPartyAuditRecorder) RecordPartyMutation(_ context.Context, record PartyAuditRecord) error {
	if recorder == nil {
		return ErrPartyAuditUnavailable
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

var _ PartyRepository = (*MemoryPartyRepository)(nil)
var _ PartyAuthorizer = MemoryPartyAuthorizer{}
var _ PartyFieldAuthorizer = MemoryPartyFieldAuthorizer{}
var _ PartyBankReferenceValidator = AllowAllPartyBankReferenceValidator{}
var _ PartyBankControlEvaluator = AllowAllPartyBankControlEvaluator{}
var _ PartyAuditRecorder = (*MemoryPartyAuditRecorder)(nil)
