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
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const MasterDataPublicationPermission = "finance.omd.publish.approved.master.data.changes"

const (
	MasterDataAggregateLegalEntity     MasterDataAggregateType = "legal_entity"
	MasterDataAggregateParty           MasterDataAggregateType = "party"
	MasterDataAggregateCustomerProfile MasterDataAggregateType = "customer_profile"
	MasterDataAggregateVendorProfile   MasterDataAggregateType = "vendor_profile"
	MasterDataAggregateFiscalCalendar  MasterDataAggregateType = "fiscal_calendar"

	MasterDataPublicationAction = "publish-approved-master-data-change"
	MasterDataPublicationStatus = "published"
	MasterDataDependentPending  = "pending"

	LegalEntityPublishedEvent     = "LegalEntityPublished"
	PartyPublishedEvent           = "PartyPublished"
	CustomerProfilePublishedEvent = "CustomerProfilePublished"
	VendorProfilePublishedEvent   = "VendorProfilePublished"
	FiscalCalendarPublishedEvent  = "FiscalCalendarPublished"
)

var (
	ErrInvalidMasterDataPublicationCommand           = errors.New("invalid master-data publication command")
	ErrMasterDataPublicationNotFound                 = errors.New("master-data publication source not found")
	ErrMasterDataPublicationVersionConflict          = errors.New("master-data publication source version conflict")
	ErrMasterDataPublicationSourceChanged            = errors.New("master-data publication source changed during publication")
	ErrMasterDataPublicationAuthorizationDenied      = errors.New("master-data publication authorization denied")
	ErrMasterDataPublicationAuthorizationUnavailable = errors.New("master-data publication authorization unavailable")
	ErrMasterDataPublicationAuthorizationStale       = errors.New("master-data publication authorization stale")
	ErrMasterDataPublicationApprovalRequired         = errors.New("master-data publication approval required")
	ErrMasterDataPublicationApprovalStale            = errors.New("master-data publication approval is stale")
	ErrMasterDataPublicationIdempotencyConflict      = errors.New("master-data publication idempotency conflict")
	ErrMasterDataPublicationCommandInProgress        = errors.New("master-data publication command is already in progress")
	ErrMasterDataPublicationAuditUnavailable         = errors.New("master-data publication audit unavailable")
	ErrMasterDataPublicationAlreadyPublished         = errors.New("master-data aggregate version is already published")
	ErrInvalidMasterDataPublicationService           = errors.New("invalid master-data publication service")
	ErrMasterDataPublicationDurableCommandFailed     = errors.New("master-data publication command previously failed")
)

type MasterDataAggregateType string

func (aggregateType MasterDataAggregateType) valid() bool {
	switch aggregateType {
	case MasterDataAggregateLegalEntity, MasterDataAggregateParty, MasterDataAggregateCustomerProfile, MasterDataAggregateVendorProfile, MasterDataAggregateFiscalCalendar:
		return true
	default:
		return false
	}
}

func (aggregateType MasterDataAggregateType) EventType() string {
	switch aggregateType {
	case MasterDataAggregateLegalEntity:
		return LegalEntityPublishedEvent
	case MasterDataAggregateParty:
		return PartyPublishedEvent
	case MasterDataAggregateCustomerProfile:
		return CustomerProfilePublishedEvent
	case MasterDataAggregateVendorProfile:
		return VendorProfilePublishedEvent
	case MasterDataAggregateFiscalCalendar:
		return FiscalCalendarPublishedEvent
	default:
		return ""
	}
}

type MasterDataPublicationCommand struct {
	AggregateType   MasterDataAggregateType
	AggregateID     uuid.UUID
	ScopeID         uuid.UUID
	ExpectedVersion *aggregateversion.AggregateVersion
	IdempotencyKey  string
	CorrelationID   string
	CausationID     string
}

func (command MasterDataPublicationCommand) Validate() error {
	command.AggregateType = MasterDataAggregateType(strings.ToLower(strings.TrimSpace(string(command.AggregateType))))
	if !command.AggregateType.valid() || command.AggregateID == uuid.Nil || command.ScopeID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
		return fmt.Errorf("%w: aggregate type, aggregate id, scope id, and expected version are required", ErrInvalidMasterDataPublicationCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidMasterDataPublicationCommand)
	}
	return nil
}

type MasterDataPublicationCandidate struct {
	AggregateType  MasterDataAggregateType
	AggregateID    uuid.UUID
	ScopeID        uuid.UUID
	Status         string
	EffectiveFrom  *time.Time
	EffectiveTo    *time.Time
	Version        aggregateversion.AggregateVersion
	RevisionNumber int64
	Approval       *ApprovalDecisionReference
	Fingerprint    string
	Payload        json.RawMessage
}

func (candidate MasterDataPublicationCandidate) validate() error {
	if !candidate.AggregateType.valid() || candidate.AggregateID == uuid.Nil || candidate.ScopeID == uuid.Nil || candidate.Version.Value() < 1 || candidate.RevisionNumber < 1 || (candidate.AggregateType != MasterDataAggregateParty && (candidate.EffectiveFrom == nil || candidate.EffectiveFrom.IsZero())) || strings.TrimSpace(candidate.Fingerprint) == "" || len(candidate.Payload) == 0 || !json.Valid(candidate.Payload) {
		return ErrMasterDataPublicationNotFound
	}
	return nil
}

type MasterDataPublicationResult struct {
	PublicationID         uuid.UUID                         `json:"publicationId"`
	MessageID             uuid.UUID                         `json:"messageId"`
	AggregateType         MasterDataAggregateType           `json:"aggregateType"`
	AggregateID           uuid.UUID                         `json:"aggregateId"`
	ScopeID               uuid.UUID                         `json:"scopeId"`
	AggregateVersion      aggregateversion.AggregateVersion `json:"aggregateVersion"`
	RevisionNumber        int64                             `json:"revisionNumber"`
	Status                string                            `json:"status"`
	EffectiveFrom         *time.Time                        `json:"effectiveFrom,omitempty"`
	EffectiveTo           *time.Time                        `json:"effectiveTo,omitempty"`
	DependentAvailability string                            `json:"dependentAvailability"`
	EventType             string                            `json:"eventType"`
	Replayed              bool                              `json:"replayed,omitempty"`
}

type MasterDataPublicationAuthorizer interface {
	AuthorizeMasterDataPublication(context.Context, Actor, MasterDataPublicationCommand, MasterDataPublicationCandidate) (AuthorizationDecision, error)
}

type MasterDataPublicationAuditRecord struct {
	PublicationID         uuid.UUID
	AggregateType         MasterDataAggregateType
	AggregateID           uuid.UUID
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

type MasterDataPublicationAuditRecorder interface {
	RecordMasterDataPublication(context.Context, MasterDataPublicationAuditRecord) error
}

type MasterDataPublicationRepository interface {
	Load(context.Context, MasterDataPublicationCommand) (MasterDataPublicationCandidate, error)
	Commit(context.Context, MasterDataPublicationCommand, MasterDataPublicationCandidate, platformevents.Envelope, MasterDataPublicationResult, MasterDataPublicationAuditRecord, *DurableMasterDataPublicationCommit) error
}

type DurableMasterDataPublicationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableMasterDataPublicationServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type MasterDataPublicationService struct {
	repository  MasterDataPublicationRepository
	authorizer  MasterDataPublicationAuthorizer
	audit       MasterDataPublicationAuditRecorder
	clock       func() time.Time
	durable     *DurableMasterDataPublicationServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedMasterDataPublication
}

type storedMasterDataPublication struct {
	fingerprint string
	result      MasterDataPublicationResult
}

func NewMasterDataPublicationService(repository MasterDataPublicationRepository, authorizer MasterDataPublicationAuthorizer, audit MasterDataPublicationAuditRecorder, clock func() time.Time) (*MasterDataPublicationService, error) {
	return newMasterDataPublicationService(repository, authorizer, audit, clock, nil)
}

func NewMasterDataPublicationServiceWithDurableIdempotency(repository MasterDataPublicationRepository, authorizer MasterDataPublicationAuthorizer, audit MasterDataPublicationAuditRecorder, clock func() time.Time, durable DurableMasterDataPublicationServiceConfig) (*MasterDataPublicationService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidMasterDataPublicationService
	}
	return newMasterDataPublicationService(repository, authorizer, audit, clock, &durable)
}

func newMasterDataPublicationService(repository MasterDataPublicationRepository, authorizer MasterDataPublicationAuthorizer, audit MasterDataPublicationAuditRecorder, clock func() time.Time, durable *DurableMasterDataPublicationServiceConfig) (*MasterDataPublicationService, error) {
	if repository == nil || authorizer == nil || audit == nil || clock == nil {
		return nil, ErrInvalidMasterDataPublicationService
	}
	if binder, ok := repository.(interface {
		BindMasterDataPublicationAuditRecorder(MasterDataPublicationAuditRecorder)
	}); ok {
		binder.BindMasterDataPublicationAuditRecorder(audit)
	}
	return &MasterDataPublicationService{repository: repository, authorizer: authorizer, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedMasterDataPublication)}, nil
}

func (service *MasterDataPublicationService) Execute(ctx context.Context, actor Actor, command MasterDataPublicationCommand) (MasterDataPublicationResult, error) {
	if service == nil {
		return MasterDataPublicationResult{}, ErrInvalidMasterDataPublicationService
	}
	if err := actor.Validate(); err != nil {
		return MasterDataPublicationResult{}, ErrMasterDataPublicationAuthorizationDenied
	}
	command.AggregateType = MasterDataAggregateType(strings.ToLower(strings.TrimSpace(string(command.AggregateType))))
	if err := command.Validate(); err != nil {
		return MasterDataPublicationResult{}, err
	}
	fingerprint, err := masterDataPublicationCommandFingerprint(command)
	if err != nil {
		return MasterDataPublicationResult{}, err
	}
	key := actor.UserID.String() + ":" + command.ScopeID.String() + ":" + command.IdempotencyKey

	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scope, marshalErr := json.Marshal(struct {
			Module        string                  `json:"module"`
			Actor         uuid.UUID               `json:"actorId"`
			Scope         uuid.UUID               `json:"scopeId"`
			AggregateType MasterDataAggregateType `json:"aggregateType"`
			AggregateID   uuid.UUID               `json:"aggregateId"`
		}{"organization", actor.UserID, command.ScopeID, command.AggregateType, command.AggregateID})
		if marshalErr != nil {
			return MasterDataPublicationResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return MasterDataPublicationResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return MasterDataPublicationResult{}, ErrMasterDataPublicationIdempotencyConflict
		}
		if err != nil {
			return MasterDataPublicationResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return MasterDataPublicationResult{}, ErrMasterDataPublicationCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return MasterDataPublicationResult{}, ErrMasterDataPublicationDurableCommandFailed
			}
			result, decodeErr := decodeDurableMasterDataPublicationResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return MasterDataPublicationResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return MasterDataPublicationResult{}, ErrMasterDataPublicationIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	candidate, err := service.repository.Load(ctx, command)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	if err := candidate.validate(); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	if candidate.AggregateType != command.AggregateType || candidate.AggregateID != command.AggregateID || candidate.ScopeID != command.ScopeID {
		service.finalizeDurableFailure(ctx, acquisition, ErrMasterDataPublicationAuthorizationDenied)
		return MasterDataPublicationResult{}, ErrMasterDataPublicationAuthorizationDenied
	}
	if !candidate.Version.Matches(*command.ExpectedVersion) {
		service.finalizeDurableFailure(ctx, acquisition, ErrMasterDataPublicationVersionConflict)
		return MasterDataPublicationResult{}, ErrMasterDataPublicationVersionConflict
	}
	decision, err := service.authorizer.AuthorizeMasterDataPublication(ctx, actor, command, candidate)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	if err := authorizeMasterDataPublicationDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	if err := validateMasterDataPublicationApproval(candidate, decision); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}

	now := service.clock().UTC()
	publicationID, messageID := uuid.New(), uuid.New()
	correlationID := validOrNewUUID(command.CorrelationID)
	causationID := validOrNewUUID(command.CausationID)
	payload, err := masterDataPublicationPayload(candidate)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	payloadFingerprint, err := platformevents.ComputePayloadFingerprint(payload)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	event, err := platformevents.NewEnvelope(platformevents.EnvelopeInput{
		MessageID: messageID.String(), EventType: command.AggregateType.EventType(), EventVersion: 1, OccurredAt: now,
		SourceContext: "organization", AggregateID: candidate.AggregateID.String(), AggregateVersion: candidate.Version.Value(),
		AccountingScopeID: candidate.ScopeID.String(), CorrelationID: correlationID, CausationID: causationID,
		DataClassification: platformevents.Internal, PayloadFingerprint: payloadFingerprint, Data: payload,
	})
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	result := MasterDataPublicationResult{PublicationID: publicationID, MessageID: messageID, AggregateType: candidate.AggregateType, AggregateID: candidate.AggregateID, ScopeID: candidate.ScopeID, AggregateVersion: candidate.Version, RevisionNumber: candidate.RevisionNumber, Status: MasterDataPublicationStatus, EffectiveFrom: candidate.EffectiveFrom, EffectiveTo: cloneTime(candidate.EffectiveTo), DependentAvailability: MasterDataDependentPending, EventType: event.EventType()}
	audit := MasterDataPublicationAuditRecord{PublicationID: publicationID, AggregateType: candidate.AggregateType, AggregateID: candidate.AggregateID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: MasterDataPublicationAction, ScopeID: candidate.ScopeID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Approval: cloneApproval(candidate.Approval), RevisionNumber: candidate.RevisionNumber, BeforeFingerprint: candidate.Fingerprint, AfterFingerprint: candidate.Fingerprint, ChangedFields: []string{"publication_status", "integration_outbox"}, CorrelationID: correlationID, CausationID: causationID}

	var durableCommit *DurableMasterDataPublicationCommit
	if service.durable != nil {
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return MasterDataPublicationResult{}, marshalErr
		}
		status := 200
		aggregateID := candidate.AggregateID
		processID := publicationID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, &processID)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return MasterDataPublicationResult{}, metadataErr
		}
		durableCommit = &DurableMasterDataPublicationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata}
	}
	if err := service.repository.Commit(ctx, command, candidate, event, result, audit, durableCommit); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return MasterDataPublicationResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedMasterDataPublication{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func authorizeMasterDataPublicationDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	switch strings.ToLower(strings.TrimSpace(decision.Outcome)) {
	case "stale":
		return ErrMasterDataPublicationAuthorizationStale
	case "unavailable":
		return ErrMasterDataPublicationAuthorizationUnavailable
	}
	if !decision.Allowed || decision.Permission != MasterDataPublicationPermission || decision.DecisionReference == uuid.Nil {
		return ErrMasterDataPublicationAuthorizationDenied
	}
	for _, approved := range decision.ApprovedScopeIDs {
		if approved == uuid.Nil || approved == scopeID {
			return nil
		}
	}
	return ErrMasterDataPublicationAuthorizationDenied
}

func validateMasterDataPublicationApproval(candidate MasterDataPublicationCandidate, decision AuthorizationDecision) error {
	if candidate.Approval == nil {
		if decision.ApprovalRequired || candidate.Status == LegalEntityStatusDraft || candidate.Status == "draft" || candidate.Status == CustomerProfileStatusDraft || candidate.Status == VendorProfileStatusDraft || candidate.Status == FiscalCalendarStatusDraft {
			return ErrMasterDataPublicationApprovalRequired
		}
		return nil
	}
	if err := candidate.Approval.Validate(); err != nil || candidate.Approval.SubjectVersion != candidate.Version.Value() || candidate.Approval.CandidateFingerprint != candidate.Fingerprint {
		return ErrMasterDataPublicationApprovalStale
	}
	return nil
}

func masterDataPublicationPayload(candidate MasterDataPublicationCandidate) ([]byte, error) {
	var snapshot json.RawMessage = append(json.RawMessage(nil), candidate.Payload...)
	payload := struct {
		AggregateType    MasterDataAggregateType           `json:"aggregateType"`
		AggregateID      uuid.UUID                         `json:"aggregateId"`
		ScopeID          uuid.UUID                         `json:"scopeId"`
		AggregateVersion aggregateversion.AggregateVersion `json:"aggregateVersion"`
		RevisionNumber   int64                             `json:"revisionNumber"`
		Status           string                            `json:"status"`
		EffectiveFrom    *time.Time                        `json:"effectiveFrom,omitempty"`
		EffectiveTo      *time.Time                        `json:"effectiveTo,omitempty"`
		Approval         *ApprovalDecisionReference        `json:"approval,omitempty"`
		Snapshot         json.RawMessage                   `json:"snapshot"`
	}{candidate.AggregateType, candidate.AggregateID, candidate.ScopeID, candidate.Version, candidate.RevisionNumber, candidate.Status, cloneTime(candidate.EffectiveFrom), cloneTime(candidate.EffectiveTo), cloneApproval(candidate.Approval), snapshot}
	return json.Marshal(payload)
}

func masterDataPublicationCommandFingerprint(command MasterDataPublicationCommand) (string, error) {
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func validOrNewUUID(value string) string {
	if parsed, err := uuid.Parse(strings.TrimSpace(value)); err == nil && parsed != uuid.Nil {
		return parsed.String()
	}
	return uuid.NewString()
}

func decodeDurableMasterDataPublicationResult(body []byte) (MasterDataPublicationResult, error) {
	var result MasterDataPublicationResult
	if err := json.Unmarshal(body, &result); err != nil {
		return MasterDataPublicationResult{}, ErrMasterDataPublicationCommandInProgress
	}
	return result, nil
}

func (service *MasterDataPublicationService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrMasterDataPublicationAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrMasterDataPublicationAuthorizationStale):
		code = "POLICY_STALE"
	case errors.Is(commandErr, ErrMasterDataPublicationVersionConflict), errors.Is(commandErr, ErrMasterDataPublicationSourceChanged):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrMasterDataPublicationNotFound):
		code = "MASTER_DATA_NOT_FOUND"
	case errors.Is(commandErr, ErrMasterDataPublicationApprovalRequired), errors.Is(commandErr, ErrMasterDataPublicationApprovalStale):
		code = "APPROVAL_REQUIRED"
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

// MemoryMasterDataPublicationRepository is used by unit tests and the
// in-memory API composition. It stores immutable candidate snapshots and
// records the published envelope, while the Postgres implementation provides
// the durable transaction and outbox behavior.
type MemoryMasterDataPublicationRepository struct {
	mu           sync.Mutex
	candidates   map[string]MasterDataPublicationCandidate
	publications map[string]MasterDataPublicationResult
	events       []platformevents.Envelope
	audit        MasterDataPublicationAuditRecorder
}

func NewMemoryMasterDataPublicationRepository(candidates ...MasterDataPublicationCandidate) *MemoryMasterDataPublicationRepository {
	repository := &MemoryMasterDataPublicationRepository{candidates: make(map[string]MasterDataPublicationCandidate), publications: make(map[string]MasterDataPublicationResult)}
	for _, candidate := range candidates {
		_ = repository.Put(candidate)
	}
	return repository
}

func (repository *MemoryMasterDataPublicationRepository) BindMasterDataPublicationAuditRecorder(audit MasterDataPublicationAuditRecorder) {
	if repository != nil {
		repository.audit = audit
	}
}

func (repository *MemoryMasterDataPublicationRepository) Put(candidate MasterDataPublicationCandidate) error {
	if repository == nil {
		return ErrInvalidMasterDataPublicationService
	}
	if err := candidate.validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.candidates[masterDataPublicationKey(candidate.AggregateType, candidate.AggregateID)] = cloneMasterDataPublicationCandidate(candidate)
	return nil
}

func (repository *MemoryMasterDataPublicationRepository) Load(_ context.Context, command MasterDataPublicationCommand) (MasterDataPublicationCandidate, error) {
	if repository == nil {
		return MasterDataPublicationCandidate{}, ErrInvalidMasterDataPublicationService
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	candidate, ok := repository.candidates[masterDataPublicationKey(command.AggregateType, command.AggregateID)]
	if !ok {
		return MasterDataPublicationCandidate{}, ErrMasterDataPublicationNotFound
	}
	return cloneMasterDataPublicationCandidate(candidate), nil
}

func (repository *MemoryMasterDataPublicationRepository) Commit(ctx context.Context, command MasterDataPublicationCommand, candidate MasterDataPublicationCandidate, event platformevents.Envelope, result MasterDataPublicationResult, audit MasterDataPublicationAuditRecord, _ *DurableMasterDataPublicationCommit) error {
	if repository == nil {
		return ErrInvalidMasterDataPublicationService
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, ok := repository.candidates[masterDataPublicationKey(command.AggregateType, command.AggregateID)]
	if !ok {
		return ErrMasterDataPublicationNotFound
	}
	if current.Version.Value() != candidate.Version.Value() || current.Fingerprint != candidate.Fingerprint {
		return ErrMasterDataPublicationSourceChanged
	}
	publicationKey := masterDataPublicationVersionKey(candidate)
	if _, exists := repository.publications[publicationKey]; exists {
		return ErrMasterDataPublicationAlreadyPublished
	}
	if repository.audit == nil {
		return ErrMasterDataPublicationAuditUnavailable
	}
	if err := repository.audit.RecordMasterDataPublication(ctx, audit); err != nil {
		return err
	}
	repository.publications[publicationKey] = result
	repository.events = append(repository.events, event)
	return nil
}

func (repository *MemoryMasterDataPublicationRepository) PublishedEvents() []platformevents.Envelope {
	if repository == nil {
		return nil
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	result := make([]platformevents.Envelope, len(repository.events))
	copy(result, repository.events)
	return result
}

type MemoryMasterDataPublicationAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryMasterDataPublicationAuthorizer) AuthorizeMasterDataPublication(context.Context, Actor, MasterDataPublicationCommand, MasterDataPublicationCandidate) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryMasterDataPublicationAuditRecorder struct {
	mu      sync.Mutex
	Records []MasterDataPublicationAuditRecord
	Err     error
}

func (recorder *MemoryMasterDataPublicationAuditRecorder) RecordMasterDataPublication(_ context.Context, record MasterDataPublicationAuditRecord) error {
	if recorder == nil {
		return ErrMasterDataPublicationAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func masterDataPublicationKey(aggregateType MasterDataAggregateType, aggregateID uuid.UUID) string {
	return string(aggregateType) + ":" + aggregateID.String()
}

func masterDataPublicationVersionKey(candidate MasterDataPublicationCandidate) string {
	return fmt.Sprintf("%s:%s:%d", candidate.AggregateType, candidate.AggregateID, candidate.Version.Value())
}

func cloneMasterDataPublicationCandidate(candidate MasterDataPublicationCandidate) MasterDataPublicationCandidate {
	candidate.EffectiveTo = cloneTime(candidate.EffectiveTo)
	candidate.Approval = cloneApproval(candidate.Approval)
	candidate.Payload = append(json.RawMessage(nil), candidate.Payload...)
	return candidate
}
