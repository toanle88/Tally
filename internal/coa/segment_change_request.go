package coa

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

const (
	SegmentChangeRequestPermission = "finance.coa.request.segment.changes"

	SegmentChangeTypeDefinition = "definition"
	SegmentChangeTypeValue      = "value"

	SegmentChangeRequestAction = "request"

	SegmentChangeRequestApprovalPending = "pending"
	SegmentChangeRequestApplicationOpen = "not-applied"
	SegmentChangeRequestNextAction      = "await-approval"
)

var (
	ErrInvalidSegmentChangeRequest                  = errors.New("invalid segment change request")
	ErrInvalidSegmentChangeRequestCommand           = errors.New("invalid segment change request command")
	ErrInvalidSegmentChangeRequestService           = errors.New("invalid segment change request service")
	ErrSegmentChangeRequestSubjectNotFound          = errors.New("segment change request subject not found")
	ErrSegmentChangeRequestUnsupportedSubject       = errors.New("unsupported segment change request subject")
	ErrSegmentChangeRequestVersionConflict          = errors.New("segment change request subject version conflict")
	ErrSegmentChangeRequestAuthorizationDenied      = errors.New("segment change request authorization denied")
	ErrSegmentChangeRequestAuthorizationUnavailable = errors.New("segment change request authorization unavailable")
	ErrSegmentChangeRequestAuthorizationStale       = errors.New("segment change request authorization stale")
	ErrSegmentChangeRequestIdempotencyConflict      = errors.New("segment change request idempotency conflict")
	ErrSegmentChangeRequestCommandInProgress        = errors.New("segment change request command is already in progress")
	ErrSegmentChangeRequestAuditUnavailable         = errors.New("segment change request audit unavailable")
	ErrSegmentChangeRequestDurableCommandFailed     = errors.New("segment change request command previously failed")
	ErrSegmentChangeRequestDuplicate                = errors.New("duplicate segment change request")
)

// SegmentChangeProposal reuses the maintenance fields owned by the existing
// definition and value commands. It is deliberately a proposal snapshot: it
// does not mutate either subject and is applied only by Story 5.
type SegmentChangeProposal struct {
	Action              string        `json:"action"`
	SegmentDefinitionID uuid.UUID     `json:"segmentDefinitionId,omitempty"`
	SegmentValueID      uuid.UUID     `json:"segmentValueId,omitempty"`
	SegmentType         string        `json:"segmentType,omitempty"`
	Code                string        `json:"code,omitempty"`
	Name                string        `json:"name,omitempty"`
	Value               string        `json:"value,omitempty"`
	Description         string        `json:"description,omitempty"`
	Status              SegmentStatus `json:"status,omitempty"`
	EffectiveDateFrom   time.Time     `json:"effectiveDateFrom"`
	EffectiveDateTo     *time.Time    `json:"effectiveDateTo,omitempty"`
}

func (proposal SegmentChangeProposal) Canonical(changeType string) SegmentChangeProposal {
	proposal.Action = strings.ToLower(strings.TrimSpace(proposal.Action))
	proposal.SegmentType = strings.TrimSpace(proposal.SegmentType)
	proposal.Code = strings.TrimSpace(proposal.Code)
	proposal.Name = strings.TrimSpace(proposal.Name)
	proposal.Value = normalizeSegmentValue(proposal.Value)
	proposal.Description = strings.TrimSpace(proposal.Description)
	proposal.Status = canonicalSegmentStatus(proposal.Status.String())
	proposal.EffectiveDateFrom = dateOnly(proposal.EffectiveDateFrom)
	proposal.EffectiveDateTo = cloneDate(proposal.EffectiveDateTo)
	if changeType == SegmentChangeTypeDefinition {
		proposal.SegmentValueID = uuid.Nil
	}
	if changeType == SegmentChangeTypeValue {
		proposal.SegmentType = ""
		proposal.Code = ""
		proposal.Name = ""
	}
	return proposal
}

type SegmentChangeRequestCommand struct {
	ChangeType             string
	SubjectID              uuid.UUID
	SubjectVersion         aggregateversion.AggregateVersion
	ScopeID                uuid.UUID
	RequestedEffectiveDate time.Time
	ApprovalRequestID      uuid.UUID
	ProposedChange         SegmentChangeProposal
	IdempotencyKey         string
	CorrelationID          string
	CausationID            string
}

func (command SegmentChangeRequestCommand) Canonical() SegmentChangeRequestCommand {
	command.ChangeType = strings.ToLower(strings.TrimSpace(command.ChangeType))
	command.RequestedEffectiveDate = dateOnly(command.RequestedEffectiveDate)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.ProposedChange = command.ProposedChange.Canonical(command.ChangeType)
	return command
}

func (command SegmentChangeRequestCommand) Validate() error {
	command = command.Canonical()
	if command.ChangeType != SegmentChangeTypeDefinition && command.ChangeType != SegmentChangeTypeValue {
		return fmt.Errorf("%w: change type %q is not supported by this COA slice", ErrSegmentChangeRequestUnsupportedSubject, command.ChangeType)
	}
	if command.ScopeID == uuid.Nil || command.SubjectID == uuid.Nil || command.SubjectVersion.Value() < 1 {
		return fmt.Errorf("%w: scope, subject, and subject version are required", ErrInvalidSegmentChangeRequestCommand)
	}
	if command.RequestedEffectiveDate.IsZero() {
		return fmt.Errorf("%w: requested effective date is required", ErrInvalidSegmentChangeRequestCommand)
	}
	if command.ApprovalRequestID == uuid.Nil {
		return fmt.Errorf("%w: a Workflow approval request reference is required", ErrInvalidSegmentChangeRequestCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidSegmentChangeRequestCommand)
	}
	proposal := command.ProposedChange
	if proposal.Action != SegmentChangeRequestAction {
		return fmt.Errorf("%w: only update proposals can be requested", ErrInvalidSegmentChangeRequestCommand)
	}
	if proposal.EffectiveDateFrom.IsZero() || proposal.EffectiveDateTo != nil && proposal.EffectiveDateTo.Before(proposal.EffectiveDateFrom) {
		return fmt.Errorf("%w: proposed effective date range is invalid", ErrInvalidSegmentChangeRequestCommand)
	}
	if !validSegmentStatus(proposal.Status) {
		return fmt.Errorf("%w: proposed lifecycle status is invalid", ErrInvalidSegmentChangeRequestCommand)
	}
	switch command.ChangeType {
	case SegmentChangeTypeDefinition:
		if proposal.SegmentDefinitionID != command.SubjectID || strings.TrimSpace(proposal.SegmentType) == "" || strings.TrimSpace(proposal.Code) == "" || strings.TrimSpace(proposal.Name) == "" {
			return fmt.Errorf("%w: the definition proposal does not identify a complete subject", ErrInvalidSegmentChangeRequestCommand)
		}
	case SegmentChangeTypeValue:
		if proposal.SegmentValueID != command.SubjectID || proposal.SegmentDefinitionID == uuid.Nil || proposal.Value == "" {
			return fmt.Errorf("%w: the value proposal does not identify a complete subject", ErrInvalidSegmentChangeRequestCommand)
		}
	}
	return nil
}

type SegmentChangeRequestSubject struct {
	ChangeType     string
	SubjectID      uuid.UUID
	ScopeID        uuid.UUID
	Version        aggregateversion.AggregateVersion
	RevisionNumber int64
	Definition     *SegmentDefinition
	Value          *SegmentValue
}

type SegmentChangeRequestSubjectReader interface {
	GetSegmentChangeSubject(context.Context, string, uuid.UUID) (SegmentChangeRequestSubject, error)
}

type SegmentChangeRequestAuthorizer interface {
	AuthorizeSegmentChangeRequest(context.Context, Actor, SegmentChangeRequestCommand, *SegmentChangeRequestSubject) (AuthorizationDecision, error)
}

type SegmentChangeRequestAuditRecorder interface {
	RecordSegmentChangeRequestMutation(context.Context, AuditRecord) error
}

type SegmentChangeRequest struct {
	ID                     uuid.UUID
	ScopeID                uuid.UUID
	ChangeType             string
	SubjectID              uuid.UUID
	SubjectVersion         aggregateversion.AggregateVersion
	RequestedEffectiveDate time.Time
	ApprovalRequestID      uuid.UUID
	ApprovalStatus         string
	ApplicationStatus      string
	ValidationOutcome      string
	ConflictCode           string
	RejectionReason        string
	NextAction             string
	ProposedChange         SegmentChangeProposal
	SubjectFingerprint     string
	ProposedFingerprint    string
	Version                aggregateversion.AggregateVersion
	RevisionNumber         int64
	CreatedBy              uuid.UUID
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (request SegmentChangeRequest) Validate() error {
	if request.ID == uuid.Nil || request.ScopeID == uuid.Nil || request.SubjectID == uuid.Nil || request.SubjectVersion.Value() < 1 || request.ApprovalRequestID == uuid.Nil {
		return fmt.Errorf("%w: identity and references are required", ErrInvalidSegmentChangeRequest)
	}
	if request.ChangeType != SegmentChangeTypeDefinition && request.ChangeType != SegmentChangeTypeValue {
		return fmt.Errorf("%w: unsupported change type", ErrInvalidSegmentChangeRequest)
	}
	if request.RequestedEffectiveDate.IsZero() || request.Version.Value() < 1 || request.RevisionNumber < 1 {
		return fmt.Errorf("%w: lifecycle and version state are invalid", ErrInvalidSegmentChangeRequest)
	}
	if request.ApprovalStatus != SegmentChangeRequestApprovalPending || request.ApplicationStatus != SegmentChangeRequestApplicationOpen {
		return fmt.Errorf("%w: initial request lifecycle state is invalid", ErrInvalidSegmentChangeRequest)
	}
	if request.ValidationOutcome != "valid" || request.NextAction != SegmentChangeRequestNextAction {
		return fmt.Errorf("%w: validation and next-action state are invalid", ErrInvalidSegmentChangeRequest)
	}
	if strings.TrimSpace(request.SubjectFingerprint) == "" || strings.TrimSpace(request.ProposedFingerprint) == "" || request.CreatedBy == uuid.Nil || request.CreatedAt.IsZero() || request.UpdatedAt.Before(request.CreatedAt) {
		return fmt.Errorf("%w: provenance is required", ErrInvalidSegmentChangeRequest)
	}
	if err := request.ProposedChange.Canonical(request.ChangeType).validateFor(request.ChangeType, request.SubjectID); err != nil {
		return err
	}
	return nil
}

func (proposal SegmentChangeProposal) validateFor(changeType string, subjectID uuid.UUID) error {
	if proposal.Action != SegmentChangeRequestAction || proposal.EffectiveDateFrom.IsZero() || proposal.EffectiveDateTo != nil && proposal.EffectiveDateTo.Before(proposal.EffectiveDateFrom) || !validSegmentStatus(proposal.Status) {
		return fmt.Errorf("%w: proposal lifecycle is invalid", ErrInvalidSegmentChangeRequest)
	}
	switch changeType {
	case SegmentChangeTypeDefinition:
		if proposal.SegmentDefinitionID != subjectID || proposal.SegmentType == "" || proposal.Code == "" || proposal.Name == "" {
			return fmt.Errorf("%w: definition proposal is incomplete", ErrInvalidSegmentChangeRequest)
		}
	case SegmentChangeTypeValue:
		if proposal.SegmentValueID != subjectID || proposal.SegmentDefinitionID == uuid.Nil || proposal.Value == "" {
			return fmt.Errorf("%w: value proposal is incomplete", ErrInvalidSegmentChangeRequest)
		}
	default:
		return fmt.Errorf("%w: unsupported change type", ErrInvalidSegmentChangeRequest)
	}
	return nil
}

type SafeSegmentChangeRequest struct {
	ID                     uuid.UUID                         `json:"id"`
	ScopeID                uuid.UUID                         `json:"scopeId"`
	ChangeType             string                            `json:"changeType"`
	SubjectID              uuid.UUID                         `json:"subjectId"`
	SubjectVersion         int64                             `json:"subjectVersion"`
	RequestedEffectiveDate time.Time                         `json:"requestedEffectiveDate"`
	ApprovalRequestID      uuid.UUID                         `json:"approvalRequestId"`
	ApprovalStatus         string                            `json:"approvalStatus"`
	ApplicationStatus      string                            `json:"applicationStatus"`
	ValidationOutcome      string                            `json:"validationOutcome"`
	ConflictCode           string                            `json:"conflictCode,omitempty"`
	RejectionReason        string                            `json:"rejectionReason,omitempty"`
	NextAction             string                            `json:"nextAction"`
	ProposedChange         SegmentChangeProposal             `json:"proposedChange"`
	Version                aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber         int64                             `json:"revisionNumber"`
}

func (request SegmentChangeRequest) SafeProjection() SafeSegmentChangeRequest {
	return SafeSegmentChangeRequest{
		ID: request.ID, ScopeID: request.ScopeID, ChangeType: request.ChangeType, SubjectID: request.SubjectID,
		SubjectVersion: request.SubjectVersion.Value(), RequestedEffectiveDate: dateOnly(request.RequestedEffectiveDate), ApprovalRequestID: request.ApprovalRequestID,
		ApprovalStatus: request.ApprovalStatus, ApplicationStatus: request.ApplicationStatus, ValidationOutcome: request.ValidationOutcome,
		ConflictCode: request.ConflictCode, RejectionReason: request.RejectionReason, NextAction: request.NextAction,
		ProposedChange: request.ProposedChange.Canonical(request.ChangeType), Version: request.Version, RevisionNumber: request.RevisionNumber,
	}
}

type SegmentChangeRequestCommandResult struct {
	SegmentChangeRequest SafeSegmentChangeRequest `json:"segmentChangeRequest"`
	DecisionReference    uuid.UUID                `json:"decisionReference"`
	PolicyReference      string                   `json:"policyReference"`
	ValidationOutcome    string                   `json:"validationOutcome"`
	Replayed             bool                     `json:"replayed,omitempty"`
}

type SegmentChangeRequestMutation struct {
	Request SegmentChangeRequest
	Subject SegmentChangeRequestSubject
	Audit   AuditRecord
}

type SegmentChangeRequestRepository interface {
	CommitSegmentChangeRequest(context.Context, SegmentChangeRequestMutation) error
}

type DurableSegmentChangeRequestRepository interface {
	CommitSegmentChangeRequestWithIdempotency(context.Context, SegmentChangeRequestMutation, DurableSegmentChangeRequestCommit) error
}

type DurableSegmentChangeRequestCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableSegmentChangeRequestServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type SegmentChangeRequestService struct {
	repository  SegmentChangeRequestRepository
	subjects    SegmentChangeRequestSubjectReader
	authorizer  SegmentChangeRequestAuthorizer
	audit       SegmentChangeRequestAuditRecorder
	clock       func() time.Time
	durable     *DurableSegmentChangeRequestServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedSegmentChangeRequestCommand
}

type storedSegmentChangeRequestCommand struct {
	fingerprint string
	result      SegmentChangeRequestCommandResult
}

func NewSegmentChangeRequestService(repository SegmentChangeRequestRepository, subjects SegmentChangeRequestSubjectReader, authorizer SegmentChangeRequestAuthorizer, audit SegmentChangeRequestAuditRecorder, clock func() time.Time) (*SegmentChangeRequestService, error) {
	return newSegmentChangeRequestService(repository, subjects, authorizer, audit, clock, nil)
}

func NewSegmentChangeRequestServiceWithDurableIdempotency(repository SegmentChangeRequestRepository, subjects SegmentChangeRequestSubjectReader, authorizer SegmentChangeRequestAuthorizer, audit SegmentChangeRequestAuditRecorder, clock func() time.Time, durable DurableSegmentChangeRequestServiceConfig) (*SegmentChangeRequestService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidSegmentChangeRequestService
	}
	return newSegmentChangeRequestService(repository, subjects, authorizer, audit, clock, &durable)
}

func newSegmentChangeRequestService(repository SegmentChangeRequestRepository, subjects SegmentChangeRequestSubjectReader, authorizer SegmentChangeRequestAuthorizer, audit SegmentChangeRequestAuditRecorder, clock func() time.Time, durable *DurableSegmentChangeRequestServiceConfig) (*SegmentChangeRequestService, error) {
	if repository == nil || subjects == nil || authorizer == nil || audit == nil || clock == nil {
		return nil, ErrInvalidSegmentChangeRequestService
	}
	if binder, ok := repository.(interface {
		BindSegmentChangeRequestAuditRecorder(SegmentChangeRequestAuditRecorder)
	}); ok {
		binder.BindSegmentChangeRequestAuditRecorder(audit)
	}
	return &SegmentChangeRequestService{repository: repository, subjects: subjects, authorizer: authorizer, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedSegmentChangeRequestCommand)}, nil
}

func (service *SegmentChangeRequestService) Execute(ctx context.Context, actor Actor, command SegmentChangeRequestCommand) (SegmentChangeRequestCommandResult, error) {
	if service == nil {
		return SegmentChangeRequestCommandResult{}, ErrInvalidSegmentChangeRequestService
	}
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return SegmentChangeRequestCommandResult{}, ErrSegmentChangeRequestAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return SegmentChangeRequestCommandResult{}, err
	}
	fingerprint, err := segmentChangeRequestCommandFingerprint(command)
	if err != nil {
		return SegmentChangeRequestCommandResult{}, err
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
			return SegmentChangeRequestCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scope), command.IdempotencyKey)
		if err != nil {
			return SegmentChangeRequestCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return SegmentChangeRequestCommandResult{}, ErrSegmentChangeRequestIdempotencyConflict
		}
		if err != nil {
			return SegmentChangeRequestCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return SegmentChangeRequestCommandResult{}, ErrSegmentChangeRequestCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return SegmentChangeRequestCommandResult{}, ErrSegmentChangeRequestDurableCommandFailed
			}
			result, decodeErr := decodeDurableSegmentChangeRequestResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return SegmentChangeRequestCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return SegmentChangeRequestCommandResult{}, ErrSegmentChangeRequestIdempotencyConflict
			}
			result := previous.result
			result.Replayed = true
			return result, nil
		}
	}

	subject, err := service.subjects.GetSegmentChangeSubject(ctx, command.ChangeType, command.SubjectID)
	if err != nil {
		err = mapSegmentChangeRequestSubjectError(err)
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	if subject.ScopeID != command.ScopeID {
		err = ErrSegmentChangeRequestAuthorizationDenied
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	if subject.Version.Value() != command.SubjectVersion.Value() {
		err = ErrSegmentChangeRequestVersionConflict
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	if err := validateSegmentChangeProposal(command, subject); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	decision, err := service.authorizer.AuthorizeSegmentChangeRequest(ctx, actor, command, &subject)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	if err := authorizeSegmentChangeRequestDecision(decision, command.ScopeID); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}

	now := service.clock().UTC()
	proposedFingerprint, err := FingerprintSegmentChangeProposal(command.ProposedChange)
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	request := SegmentChangeRequest{
		ID: uuid.New(), ScopeID: command.ScopeID, ChangeType: command.ChangeType, SubjectID: command.SubjectID, SubjectVersion: command.SubjectVersion,
		RequestedEffectiveDate: command.RequestedEffectiveDate, ApprovalRequestID: command.ApprovalRequestID,
		ApprovalStatus: SegmentChangeRequestApprovalPending, ApplicationStatus: SegmentChangeRequestApplicationOpen, ValidationOutcome: "valid", NextAction: SegmentChangeRequestNextAction,
		ProposedChange: command.ProposedChange, SubjectFingerprint: FingerprintSegmentChangeSubject(subject), ProposedFingerprint: proposedFingerprint,
		Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedBy: actor.UserID, CreatedAt: now, UpdatedAt: now,
	}
	record := AuditRecord{
		SegmentChangeRequestID: request.ID, SegmentDefinitionID: subjectDefinitionID(subject), SegmentValueID: subjectValueID(subject), ApprovalRequestID: command.ApprovalRequestID,
		ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: SegmentChangeRequestAction, ScopeID: command.ScopeID,
		Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference,
		RevisionNumber: request.RevisionNumber, BeforeFingerprint: request.SubjectFingerprint, AfterFingerprint: request.ProposedFingerprint, CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	if err := request.Validate(); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	result := SegmentChangeRequestCommandResult{SegmentChangeRequest: request.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: request.ValidationOutcome}
	mutation := SegmentChangeRequestMutation{Request: request, Subject: subject, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableSegmentChangeRequestRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidSegmentChangeRequestService)
			return SegmentChangeRequestCommandResult{}, ErrInvalidSegmentChangeRequestService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return SegmentChangeRequestCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := request.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return SegmentChangeRequestCommandResult{}, metadataErr
		}
		err = committer.CommitSegmentChangeRequestWithIdempotency(ctx, mutation, DurableSegmentChangeRequestCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitSegmentChangeRequest(ctx, mutation)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return SegmentChangeRequestCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedSegmentChangeRequestCommand{fingerprint: fingerprint, result: result}
	}
	return result, nil
}

func validateSegmentChangeProposal(command SegmentChangeRequestCommand, subject SegmentChangeRequestSubject) error {
	if subject.ChangeType != command.ChangeType || subject.SubjectID != command.SubjectID || subject.Version.Value() != command.SubjectVersion.Value() {
		return ErrSegmentChangeRequestVersionConflict
	}
	if !dateOnly(command.RequestedEffectiveDate).Equal(dateOnly(command.ProposedChange.EffectiveDateFrom)) {
		return fmt.Errorf("%w: requested and proposed effective dates must agree", ErrInvalidSegmentChangeRequestCommand)
	}
	switch command.ChangeType {
	case SegmentChangeTypeDefinition:
		if subject.Definition == nil {
			return ErrSegmentChangeRequestSubjectNotFound
		}
		candidate := cloneSegmentDefinition(*subject.Definition)
		if err := candidate.Replace(*subject.Definition, command.ProposedChange.SegmentType, command.ProposedChange.Code, command.ProposedChange.Name, command.ProposedChange.Status, command.ProposedChange.EffectiveDateFrom, command.ProposedChange.EffectiveDateTo, subject.Definition.UpdatedAt.Add(time.Nanosecond)); err != nil {
			return err
		}
	case SegmentChangeTypeValue:
		if subject.Definition == nil || subject.Value == nil {
			return ErrSegmentChangeRequestSubjectNotFound
		}
		candidate := cloneSegmentValue(*subject.Value)
		if err := candidate.Replace(*subject.Value, *subject.Definition, command.ProposedChange.Value, command.ProposedChange.Description, command.ProposedChange.Status, command.ProposedChange.EffectiveDateFrom, command.ProposedChange.EffectiveDateTo, subject.Value.UpdatedAt.Add(time.Nanosecond)); err != nil {
			return err
		}
	default:
		return ErrSegmentChangeRequestUnsupportedSubject
	}
	return nil
}

func authorizeSegmentChangeRequestDecision(decision AuthorizationDecision, scope uuid.UUID) error {
	if !decision.Allowed || decision.Permission != SegmentChangeRequestPermission || decision.DecisionReference == uuid.Nil {
		return ErrSegmentChangeRequestAuthorizationDenied
	}
	for _, allowed := range decision.ApprovedScopeIDs {
		if allowed == uuid.Nil || allowed == scope {
			return nil
		}
	}
	return ErrSegmentChangeRequestAuthorizationDenied
}

func mapSegmentChangeRequestSubjectError(err error) error {
	switch {
	case errors.Is(err, ErrSegmentChangeRequestUnsupportedSubject):
		return err
	case errors.Is(err, ErrSegmentDefinitionNotFound), errors.Is(err, ErrSegmentValueNotFound), errors.Is(err, ErrSegmentChangeRequestSubjectNotFound):
		return ErrSegmentChangeRequestSubjectNotFound
	default:
		return err
	}
}

func subjectDefinitionID(subject SegmentChangeRequestSubject) uuid.UUID {
	if subject.ChangeType == SegmentChangeTypeDefinition {
		return subject.SubjectID
	}
	if subject.Definition != nil {
		return subject.Definition.ID
	}
	return uuid.Nil
}

func subjectValueID(subject SegmentChangeRequestSubject) uuid.UUID {
	if subject.ChangeType == SegmentChangeTypeValue {
		return subject.SubjectID
	}
	return uuid.Nil
}

func FingerprintSegmentChangeSubject(subject SegmentChangeRequestSubject) string {
	var value any
	switch subject.ChangeType {
	case SegmentChangeTypeDefinition:
		if subject.Definition != nil {
			value = struct {
				Type       string                    `json:"type"`
				Definition SegmentDefinitionSnapshot `json:"definition"`
			}{subject.ChangeType, subject.Definition.Snapshot()}
		}
	case SegmentChangeTypeValue:
		if subject.Value != nil {
			var definitionFingerprint string
			if subject.Definition != nil {
				definitionFingerprint = FingerprintSegmentDefinition(*subject.Definition)
			}
			value = struct {
				Type                  string               `json:"type"`
				DefinitionFingerprint string               `json:"definitionFingerprint"`
				Value                 SegmentValueSnapshot `json:"value"`
			}{subject.ChangeType, definitionFingerprint, subject.Value.Snapshot()}
		}
	}
	data, _ := json.Marshal(value)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func FingerprintSegmentChangeProposal(proposal SegmentChangeProposal) (string, error) {
	data, err := json.Marshal(proposal)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func segmentChangeRequestCommandFingerprint(command SegmentChangeRequestCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func (service *SegmentChangeRequestService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code := "VALIDATION_FAILED"
	switch {
	case errors.Is(commandErr, ErrSegmentChangeRequestAuthorizationDenied):
		code = "AUTHORIZATION_DENIED"
	case errors.Is(commandErr, ErrSegmentChangeRequestVersionConflict):
		code = "VERSION_CONFLICT"
	case errors.Is(commandErr, ErrSegmentChangeRequestSubjectNotFound):
		code = "SUBJECT_NOT_FOUND"
	case errors.Is(commandErr, ErrSegmentChangeRequestUnsupportedSubject):
		code = "UNSUPPORTED_SUBJECT"
	case errors.Is(commandErr, ErrSegmentChangeRequestDurableCommandFailed):
		code = "COMMAND_FAILED"
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

func decodeDurableSegmentChangeRequestResult(body []byte) (SegmentChangeRequestCommandResult, error) {
	var result SegmentChangeRequestCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return SegmentChangeRequestCommandResult{}, ErrSegmentChangeRequestCommandInProgress
	}
	return result, nil
}

type MemorySegmentChangeRequestRepository struct {
	mu       sync.RWMutex
	requests map[uuid.UUID]SegmentChangeRequest
	subjects SegmentChangeRequestSubjectReader
	audit    SegmentChangeRequestAuditRecorder
}

func NewMemorySegmentChangeRequestRepository(subjects SegmentChangeRequestSubjectReader) *MemorySegmentChangeRequestRepository {
	return &MemorySegmentChangeRequestRepository{requests: make(map[uuid.UUID]SegmentChangeRequest), subjects: subjects}
}

func (repository *MemorySegmentChangeRequestRepository) BindSegmentChangeRequestAuditRecorder(audit SegmentChangeRequestAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemorySegmentChangeRequestRepository) Get(_ context.Context, id uuid.UUID) (SegmentChangeRequest, error) {
	if repository == nil {
		return SegmentChangeRequest{}, ErrInvalidSegmentChangeRequestService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	request, ok := repository.requests[id]
	if !ok {
		return SegmentChangeRequest{}, ErrSegmentChangeRequestSubjectNotFound
	}
	return cloneSegmentChangeRequest(request), nil
}

func (repository *MemorySegmentChangeRequestRepository) CommitSegmentChangeRequest(ctx context.Context, mutation SegmentChangeRequestMutation) error {
	return repository.commit(ctx, mutation)
}

func (repository *MemorySegmentChangeRequestRepository) CommitSegmentChangeRequestWithIdempotency(ctx context.Context, mutation SegmentChangeRequestMutation, _ DurableSegmentChangeRequestCommit) error {
	return repository.commit(ctx, mutation)
}

func (repository *MemorySegmentChangeRequestRepository) commit(ctx context.Context, mutation SegmentChangeRequestMutation) error {
	if repository == nil || repository.subjects == nil {
		return ErrInvalidSegmentChangeRequestService
	}
	if err := mutation.Request.Validate(); err != nil {
		return err
	}
	current, err := repository.subjects.GetSegmentChangeSubject(ctx, mutation.Request.ChangeType, mutation.Request.SubjectID)
	if err != nil {
		return mapSegmentChangeRequestSubjectError(err)
	}
	if current.ScopeID != mutation.Request.ScopeID || current.Version.Value() != mutation.Request.SubjectVersion.Value() || FingerprintSegmentChangeSubject(current) != mutation.Request.SubjectFingerprint {
		return ErrSegmentChangeRequestVersionConflict
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.requests[mutation.Request.ID]; exists {
		return ErrSegmentChangeRequestDuplicate
	}
	for _, existing := range repository.requests {
		if existing.ScopeID == mutation.Request.ScopeID && existing.ChangeType == mutation.Request.ChangeType && existing.SubjectID == mutation.Request.SubjectID && existing.SubjectVersion.Value() == mutation.Request.SubjectVersion.Value() && existing.ApprovalStatus == SegmentChangeRequestApprovalPending && existing.ApplicationStatus == SegmentChangeRequestApplicationOpen && existing.ProposedFingerprint == mutation.Request.ProposedFingerprint {
			return ErrSegmentChangeRequestDuplicate
		}
	}
	if repository.audit == nil {
		return ErrSegmentChangeRequestAuditUnavailable
	}
	if err := repository.audit.RecordSegmentChangeRequestMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.requests[mutation.Request.ID] = cloneSegmentChangeRequest(mutation.Request)
	return nil
}

func (recorder *MemoryAuditRecorder) RecordSegmentChangeRequestMutation(_ context.Context, record AuditRecord) error {
	if recorder == nil {
		return ErrSegmentChangeRequestAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func (authorizer MemoryAuthorizer) AuthorizeSegmentChangeRequest(context.Context, Actor, SegmentChangeRequestCommand, *SegmentChangeRequestSubject) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

func cloneSegmentChangeRequest(request SegmentChangeRequest) SegmentChangeRequest {
	request.RequestedEffectiveDate = dateOnly(request.RequestedEffectiveDate)
	request.ProposedChange = request.ProposedChange.Canonical(request.ChangeType)
	request.ProposedChange.EffectiveDateTo = cloneDate(request.ProposedChange.EffectiveDateTo)
	return request
}

func (repository *MemorySegmentDefinitionRepository) GetSegmentChangeSubject(ctx context.Context, changeType string, subjectID uuid.UUID) (SegmentChangeRequestSubject, error) {
	if repository == nil {
		return SegmentChangeRequestSubject{}, ErrInvalidSegmentChangeRequestService
	}
	switch changeType {
	case SegmentChangeTypeDefinition:
		definition, err := repository.Get(ctx, subjectID)
		if errors.Is(err, ErrSegmentDefinitionNotFound) {
			return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
		}
		if err != nil {
			return SegmentChangeRequestSubject{}, err
		}
		return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: definition.ID, ScopeID: definition.ScopeID, Version: definition.Version, RevisionNumber: definition.RevisionNumber, Definition: &definition}, nil
	case SegmentChangeTypeValue:
		repository.mu.RLock()
		defer repository.mu.RUnlock()
		for _, definition := range repository.definitions {
			for _, value := range definition.Values {
				if value.ID == subjectID {
					parent := cloneSegmentDefinition(definition)
					candidate := cloneSegmentValue(value)
					return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: value.ID, ScopeID: parent.ScopeID, Version: parent.Version, RevisionNumber: parent.RevisionNumber, Definition: &parent, Value: &candidate}, nil
				}
			}
		}
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
	default:
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestUnsupportedSubject
	}
}

func (repository *PostgresSegmentDefinitionRepository) GetSegmentChangeSubject(ctx context.Context, changeType string, subjectID uuid.UUID) (SegmentChangeRequestSubject, error) {
	if repository == nil || repository.pool == nil {
		return SegmentChangeRequestSubject{}, ErrInvalidSegmentChangeRequestService
	}
	definitionID := subjectID
	if changeType == SegmentChangeTypeValue {
		if err := repository.pool.QueryRow(ctx, `SELECT segment_definition_id FROM coa.segment_value WHERE segment_value_id=$1`, subjectID).Scan(&definitionID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
			}
			return SegmentChangeRequestSubject{}, mapSegmentValuePostgresError(err)
		}
	}
	if changeType != SegmentChangeTypeDefinition && changeType != SegmentChangeTypeValue {
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestUnsupportedSubject
	}
	definition, err := repository.Get(ctx, definitionID)
	if errors.Is(err, ErrSegmentDefinitionNotFound) {
		return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
	}
	if err != nil {
		return SegmentChangeRequestSubject{}, err
	}
	if changeType == SegmentChangeTypeDefinition {
		return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: definition.ID, ScopeID: definition.ScopeID, Version: definition.Version, RevisionNumber: definition.RevisionNumber, Definition: &definition}, nil
	}
	for index := range definition.Values {
		if definition.Values[index].ID == subjectID {
			value := cloneSegmentValue(definition.Values[index])
			return SegmentChangeRequestSubject{ChangeType: changeType, SubjectID: subjectID, ScopeID: definition.ScopeID, Version: definition.Version, RevisionNumber: definition.RevisionNumber, Definition: &definition, Value: &value}, nil
		}
	}
	return SegmentChangeRequestSubject{}, ErrSegmentChangeRequestSubjectNotFound
}

var _ SegmentChangeRequestRepository = (*MemorySegmentChangeRequestRepository)(nil)
var _ DurableSegmentChangeRequestRepository = (*MemorySegmentChangeRequestRepository)(nil)
var _ SegmentChangeRequestSubjectReader = (*MemorySegmentDefinitionRepository)(nil)
var _ SegmentChangeRequestSubjectReader = (*PostgresSegmentDefinitionRepository)(nil)
var _ SegmentChangeRequestAuditRecorder = (*MemoryAuditRecorder)(nil)
