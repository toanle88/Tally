package workflow

import (
	"context"
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
	ApprovalRequestedEventType        = "approval_requested"
	ApprovalDecisionRecordedEventType = "approval_decision_recorded"
)

const (
	ApprovalRequestStatusPending  = "Pending"
	ApprovalRequestStatusApproved = "Approved"
	ApprovalRequestStatusRejected = "Rejected"
)

const (
	ApprovalDecisionApproved = "approved"
	ApprovalDecisionRejected = "rejected"
)

type AuthorizationOutcome string

const (
	AuthorizationAllowed     AuthorizationOutcome = "allowed"
	AuthorizationDenied      AuthorizationOutcome = "denied"
	AuthorizationUnavailable AuthorizationOutcome = "unavailable"
	AuthorizationStale       AuthorizationOutcome = "stale"
	AuthorizationExpired     AuthorizationOutcome = "expired"
)

type AuthorizationDecision struct {
	Allowed              bool
	Outcome              AuthorizationOutcome
	Permission           string
	ApprovedScopeIDs     []string
	PolicyReference      string
	PolicyVersion        string
	DecisionReference    uuid.UUID
	ReasonCode           string
	Reason               string
	ApplicableDimensions []string
}

type ApprovalRequestStep struct {
	ID               uuid.UUID         `json:"id"`
	Sequence         int               `json:"sequence"`
	RequiredRoles    []RoleRequirement `json:"requiredRoles"`
	MinimumApprovals int               `json:"minimumApprovals"`
	Status           string            `json:"status"`
}

type ApprovalDecision struct {
	ID                   uuid.UUID `json:"id"`
	DecisionVersion      int64     `json:"decisionVersion"`
	Outcome              string    `json:"outcome"`
	Approver             Actor     `json:"approver"`
	Reason               string    `json:"reason"`
	PolicyVersion        string    `json:"policyVersion"`
	SubjectVersion       int64     `json:"subjectVersion"`
	CandidateFingerprint string    `json:"candidateFingerprint"`
	RecordedAt           time.Time `json:"recordedAt"`
}

type ApprovalRequest struct {
	ID                 uuid.UUID                         `json:"id"`
	SubjectType        string                            `json:"subjectType"`
	SubjectID          string                            `json:"subjectId"`
	SubjectVersion     int64                             `json:"subjectVersion"`
	SubjectFingerprint string                            `json:"subjectFingerprint"`
	PolicyID           uuid.UUID                         `json:"policyId"`
	PolicyVersion      string                            `json:"policyVersion"`
	Scope              ApprovalPolicyScope               `json:"scope"`
	Requester          Actor                             `json:"requester"`
	Steps              []ApprovalRequestStep             `json:"steps"`
	Decisions          []ApprovalDecision                `json:"decisions"`
	Status             string                            `json:"status"`
	Version            aggregateversion.AggregateVersion `json:"version"`
	CorrelationID      string                            `json:"correlationId"`
	CausationID        string                            `json:"causationId"`
	CreatedAt          time.Time                         `json:"createdAt"`
	UpdatedAt          time.Time                         `json:"updatedAt"`
}

func (request ApprovalRequest) Validate() error {
	if request.ID == uuid.Nil || strings.TrimSpace(request.SubjectType) == "" || strings.TrimSpace(request.SubjectID) == "" || request.PolicyID == uuid.Nil || strings.TrimSpace(request.PolicyVersion) == "" {
		return fmt.Errorf("%w: approval request identity is incomplete", ErrInvalidApprovalPolicyCommand)
	}
	if request.SubjectVersion < 1 || strings.TrimSpace(request.SubjectFingerprint) == "" || request.Version.Value() < 1 {
		return fmt.Errorf("%w: approval request subject evidence is incomplete", ErrInvalidApprovalPolicyCommand)
	}
	if err := request.Scope.Validate(); err != nil {
		return err
	}
	if err := request.Requester.Validate(); err != nil {
		return err
	}
	switch request.Status {
	case ApprovalRequestStatusPending, ApprovalRequestStatusApproved, ApprovalRequestStatusRejected:
	default:
		return fmt.Errorf("%w: unsupported approval request status", ErrInvalidApprovalPolicyCommand)
	}
	if len(request.Steps) == 0 {
		return fmt.Errorf("%w: approval request requires at least one step", ErrInvalidApprovalPolicyCommand)
	}
	for _, step := range request.Steps {
		if step.ID == uuid.Nil || step.Sequence < 1 || step.MinimumApprovals < 1 {
			return fmt.Errorf("%w: approval request step is invalid", ErrInvalidApprovalPolicyCommand)
		}
	}
	decisionIDs := make(map[uuid.UUID]struct{}, len(request.Decisions))
	for _, decision := range request.Decisions {
		if decision.ID == uuid.Nil || decision.DecisionVersion < 1 || decision.PolicyVersion != request.PolicyVersion || decision.SubjectVersion != request.SubjectVersion || decision.CandidateFingerprint != request.SubjectFingerprint {
			return fmt.Errorf("%w: approval decision evidence is invalid", ErrInvalidApprovalPolicyCommand)
		}
		if decision.Outcome != ApprovalDecisionApproved && decision.Outcome != ApprovalDecisionRejected {
			return fmt.Errorf("%w: approval decision outcome is invalid", ErrInvalidApprovalPolicyCommand)
		}
		if err := decision.Approver.Validate(); err != nil {
			return err
		}
		if _, exists := decisionIDs[decision.ID]; exists {
			return fmt.Errorf("%w: duplicate approval decision", ErrInvalidApprovalPolicyCommand)
		}
		decisionIDs[decision.ID] = struct{}{}
	}
	return nil
}

type CreateApprovalRequestCommand struct {
	RequestID          uuid.UUID
	SubjectType        string
	SubjectID          string
	SubjectVersion     int64
	SubjectFingerprint string
	PolicyID           uuid.UUID
	PolicyVersion      string
	Scope              ApprovalPolicyScope
	Steps              []ApprovalStepDefinition
	IdempotencyKey     string
	CorrelationID      string
	CausationID        string
}

func (command CreateApprovalRequestCommand) Validate() error {
	if strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.SubjectType) == "" || strings.TrimSpace(command.SubjectID) == "" || command.SubjectVersion < 1 || strings.TrimSpace(command.SubjectFingerprint) == "" || command.PolicyID == uuid.Nil || strings.TrimSpace(command.PolicyVersion) == "" {
		return fmt.Errorf("%w: approval request command identity and idempotency evidence are required", ErrInvalidApprovalPolicyCommand)
	}
	if err := command.Scope.Validate(); err != nil {
		return err
	}
	if len(command.Steps) == 0 {
		return fmt.Errorf("%w: approval request requires steps", ErrInvalidApprovalPolicyCommand)
	}
	for _, step := range command.Steps {
		if err := step.Validate(); err != nil {
			return err
		}
	}
	return nil
}

type DecideApprovalRequestCommand struct {
	RequestID            uuid.UUID
	ExpectedVersion      *aggregateversion.AggregateVersion
	DecisionID           uuid.UUID
	Outcome              string
	Reason               string
	PolicyVersion        string
	SubjectVersion       int64
	CandidateFingerprint string
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command DecideApprovalRequestCommand) Validate() error {
	if command.RequestID == uuid.Nil || strings.TrimSpace(command.IdempotencyKey) == "" || (command.Outcome != ApprovalDecisionApproved && command.Outcome != ApprovalDecisionRejected) {
		return fmt.Errorf("%w: approval decision identity, outcome, and idempotency key are required", ErrInvalidApprovalPolicyCommand)
	}
	if command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 || command.SubjectVersion < 1 || strings.TrimSpace(command.PolicyVersion) == "" || strings.TrimSpace(command.CandidateFingerprint) == "" {
		return fmt.Errorf("%w: approval decision evidence is incomplete", ErrInvalidApprovalPolicyCommand)
	}
	if command.Outcome == ApprovalDecisionRejected && strings.TrimSpace(command.Reason) == "" {
		return fmt.Errorf("%w: a rejected approval requires a reason", ErrInvalidApprovalPolicyCommand)
	}
	return nil
}

type ApprovalRequestAuditRecord struct {
	ApprovalRequestID             uuid.UUID
	ApprovalDecisionID            uuid.UUID
	ActorUserID                   uuid.UUID
	ActorAuthenticationSubjectRef string
	ApproverUserID                uuid.UUID
	Action                        string
	ScopeIDs                      []string
	Reason                        string
	Permission                    string
	PolicyReference               string
	DecisionReference             uuid.UUID
	PolicyVersion                 string
	RevisionVersion               int64
	SubjectVersion                int64
	BeforeFingerprint             string
	AfterFingerprint              string
	CorrelationID                 string
	CausationID                   string
}

type ApprovalRequestAuditRecorder interface {
	RecordApprovalRequestMutation(context.Context, ApprovalRequestAuditRecord) (uuid.UUID, error)
}

type ApprovalEvent struct {
	Type                 string
	RequestID            uuid.UUID
	DecisionID           uuid.UUID
	DecisionVersion      int64
	Outcome              string
	ApproverUserID       uuid.UUID
	AggregateVersion     int64
	PolicyID             uuid.UUID
	PolicyVersion        string
	SubjectID            string
	SubjectVersion       int64
	CandidateFingerprint string
	ScopeIDs             []string
	OccurredAt           time.Time
	CorrelationID        string
	CausationID          string
}

type ApprovalEventRecorder interface {
	RecordApprovalEvent(context.Context, ApprovalEvent) error
}

type ApprovalRequestRepository interface {
	Get(context.Context, uuid.UUID) (ApprovalRequest, error)
	Save(context.Context, ApprovalRequest, *ApprovalRequest, ApprovalRequestAuditRecord, ApprovalEvent) error
}

type DurableApprovalRequestServiceConfig struct {
	Database          platformidempotency.TxBeginner
	Coordinator       platformidempotency.DurableCoordinator
	Policy            platformidempotency.IdempotencyPolicy
	CreateOperationID string
	DecideOperationID string
}

type DurableApprovalRequestCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableApprovalRequestRepository interface {
	SaveWithIdempotency(context.Context, ApprovalRequest, *ApprovalRequest, ApprovalRequestAuditRecord, ApprovalEvent, DurableApprovalRequestCommit) error
}

type ApprovalRequestAuthorizer interface {
	AuthorizeApprovalDecision(context.Context, Actor, ApprovalRequest) (AuthorizationDecision, error)
}

type ApprovalRequestCommandResult struct {
	Request   ApprovalRequest
	Reference ApprovalDecisionReference
	Replayed  bool
}

// PreparedApprovalRequest contains the complete generic WFA request effect
// before persistence. Policy maintenance uses it to commit the request,
// audit record, and ApprovalRequested outbox message in the same transaction
// as the policy revision when the PostgreSQL repository supports that boundary.
type PreparedApprovalRequest struct {
	Request ApprovalRequest
	Audit   ApprovalRequestAuditRecord
	Event   ApprovalEvent
	Result  ApprovalRequestCommandResult
}

type ApprovalRequestService struct {
	repository ApprovalRequestRepository
	authorizer ApprovalRequestAuthorizer
	audit      ApprovalRequestAuditRecorder
	events     ApprovalEventRecorder
	clock      func() time.Time
	durable    *DurableApprovalRequestServiceConfig

	mu          sync.Mutex
	idempotency map[string]storedApprovalRequestResult
}

type storedApprovalRequestResult struct {
	fingerprint string
	result      ApprovalRequestCommandResult
}

func NewApprovalRequestService(repository ApprovalRequestRepository, authorizer ApprovalRequestAuthorizer, audit ApprovalRequestAuditRecorder, events ApprovalEventRecorder, clock func() time.Time) (*ApprovalRequestService, error) {
	return newApprovalRequestService(repository, authorizer, audit, events, clock, nil)
}

func NewApprovalRequestServiceWithDurableIdempotency(repository ApprovalRequestRepository, authorizer ApprovalRequestAuthorizer, audit ApprovalRequestAuditRecorder, events ApprovalEventRecorder, clock func() time.Time, durable DurableApprovalRequestServiceConfig) (*ApprovalRequestService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.CreateOperationID) == "" || strings.TrimSpace(durable.DecideOperationID) == "" {
		return nil, ErrInvalidApprovalPolicyService
	}
	return newApprovalRequestService(repository, authorizer, audit, events, clock, &durable)
}

func newApprovalRequestService(repository ApprovalRequestRepository, authorizer ApprovalRequestAuthorizer, audit ApprovalRequestAuditRecorder, events ApprovalEventRecorder, clock func() time.Time, durable *DurableApprovalRequestServiceConfig) (*ApprovalRequestService, error) {
	if repository == nil || authorizer == nil || audit == nil || events == nil || clock == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return &ApprovalRequestService{repository: repository, authorizer: authorizer, audit: audit, events: events, clock: clock, durable: durable, idempotency: make(map[string]storedApprovalRequestResult)}, nil
}

func (service *ApprovalRequestService) CreateApprovalRequest(ctx context.Context, actor Actor, command CreateApprovalRequestCommand) (result ApprovalRequestCommandResult, returnErr error) {
	var durableAcquisition platformidempotency.DurableAcquisition
	defer func() {
		if service != nil && returnErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, returnErr)
		}
	}()
	if service == nil {
		return ApprovalRequestCommandResult{}, ErrInvalidApprovalPolicyService
	}
	if err := actor.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	commandFP := approvalRequestCreateCommandFingerprint(command)
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		identity, err := approvalRequestDurableIdentity(actor, command.IdempotencyKey, "create")
		if err != nil {
			return ApprovalRequestCommandResult{}, err
		}
		durableIdentity = identity
		var acquisitionErr error
		durableAcquisition, acquisitionErr = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.CreateOperationID, service.durable.Policy)
		if errors.Is(acquisitionErr, platformidempotency.ErrIdempotencyConflict) {
			return ApprovalRequestCommandResult{}, ErrApprovalPolicyIdempotencyConflict
		}
		if acquisitionErr != nil {
			return ApprovalRequestCommandResult{}, acquisitionErr
		}
		if durableAcquisition.Decision() == platformidempotency.DecisionReturn {
			switch durableAcquisition.Result().State() {
			case platformidempotency.StateFailed:
				return ApprovalRequestCommandResult{}, durableApprovalRequestFailureError(durableAcquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return ApprovalRequestCommandResult{}, ErrApprovalPolicyCommandInProgress
			case platformidempotency.StateEstablished:
				stored, decodeErr := decodeApprovalRequestCommandResult(durableAcquisition.Result().ResultBody())
				if decodeErr != nil {
					return ApprovalRequestCommandResult{}, decodeErr
				}
				stored.Replayed = true
				return stored, nil
			}
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != commandFP {
				return ApprovalRequestCommandResult{}, ErrApprovalPolicyIdempotencyConflict
			}
			result := cloneApprovalRequestResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}
	prepared, err := service.prepareCreateApprovalRequest(ctx, actor, command)
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	request := prepared.Request
	auditRecord := prepared.Audit
	event := prepared.Event
	result = prepared.Result
	if service.durable != nil {
		committer, ok := service.repository.(DurableApprovalRequestRepository)
		if !ok {
			return ApprovalRequestCommandResult{}, ErrInvalidApprovalPolicyService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return ApprovalRequestCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := request.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.CreateOperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			return ApprovalRequestCommandResult{}, metadataErr
		}
		if err := committer.SaveWithIdempotency(ctx, request, nil, auditRecord, event, DurableApprovalRequestCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata}); err != nil {
			return ApprovalRequestCommandResult{}, err
		}
	} else if err := service.repository.Save(ctx, request, nil, auditRecord, event); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedApprovalRequestResult{fingerprint: commandFP, result: cloneApprovalRequestResult(result)}
	}
	return result, nil
}

func (service *ApprovalRequestService) prepareCreateApprovalRequest(ctx context.Context, actor Actor, command CreateApprovalRequestCommand) (PreparedApprovalRequest, error) {
	if service == nil {
		return PreparedApprovalRequest{}, ErrInvalidApprovalPolicyService
	}
	if err := actor.Validate(); err != nil {
		return PreparedApprovalRequest{}, err
	}
	if err := command.Validate(); err != nil {
		return PreparedApprovalRequest{}, err
	}
	now := service.clock().UTC()
	requestID := command.RequestID
	if requestID == uuid.Nil {
		requestID = uuid.New()
	}
	steps := make([]ApprovalRequestStep, len(command.Steps))
	for index, step := range command.Steps {
		steps[index] = ApprovalRequestStep{ID: step.ID, Sequence: step.Sequence, RequiredRoles: append([]RoleRequirement(nil), step.RequiredRoles...), MinimumApprovals: step.MinimumApprovals, Status: ApprovalRequestStatusPending}
	}
	request := ApprovalRequest{ID: requestID, SubjectType: strings.TrimSpace(command.SubjectType), SubjectID: strings.TrimSpace(command.SubjectID), SubjectVersion: command.SubjectVersion, SubjectFingerprint: command.SubjectFingerprint, PolicyID: command.PolicyID, PolicyVersion: command.PolicyVersion, Scope: cloneScope(command.Scope), Requester: actor, Steps: steps, Status: ApprovalRequestStatusPending, Version: aggregateversion.Initial(), CorrelationID: command.CorrelationID, CausationID: command.CausationID, CreatedAt: now, UpdatedAt: now}
	if err := request.Validate(); err != nil {
		return PreparedApprovalRequest{}, err
	}
	decision, err := service.authorizer.AuthorizeApprovalDecision(ctx, actor, request)
	if err != nil {
		return PreparedApprovalRequest{}, ErrApprovalPolicyApprovalUnavailable
	}
	if err := validateAuthorization(decision, "finance.wfa.create.approval.request"); err != nil {
		return PreparedApprovalRequest{}, err
	}
	auditRecord := ApprovalRequestAuditRecord{ApprovalRequestID: request.ID, ActorUserID: actor.UserID, ActorAuthenticationSubjectRef: actor.SubjectReference, Action: "CreateApprovalRequest", ScopeIDs: request.Scope.CanonicalScopeIDs(), Permission: decision.Permission, PolicyReference: decision.PolicyReference, DecisionReference: decision.DecisionReference, PolicyVersion: request.PolicyVersion, RevisionVersion: request.Version.Value(), SubjectVersion: request.SubjectVersion, AfterFingerprint: request.SubjectFingerprint, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	event := ApprovalEvent{Type: ApprovalRequestedEventType, RequestID: request.ID, AggregateVersion: request.Version.Value(), PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion, SubjectID: request.SubjectID, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, ScopeIDs: request.Scope.CanonicalScopeIDs(), OccurredAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	return PreparedApprovalRequest{Request: request, Audit: auditRecord, Event: event, Result: ApprovalRequestCommandResult{Request: cloneApprovalRequest(request)}}, nil
}

func (service *ApprovalRequestService) DecideApprovalRequest(ctx context.Context, actor Actor, command DecideApprovalRequestCommand) (result ApprovalRequestCommandResult, returnErr error) {
	var durableAcquisition platformidempotency.DurableAcquisition
	defer func() {
		if service != nil && returnErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, returnErr)
		}
	}()
	if service == nil {
		return ApprovalRequestCommandResult{}, ErrInvalidApprovalPolicyService
	}
	if err := actor.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	commandFP := approvalRequestDecisionCommandFingerprint(command)
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		identity, err := approvalRequestDurableIdentity(actor, command.IdempotencyKey, "decide")
		if err != nil {
			return ApprovalRequestCommandResult{}, err
		}
		durableIdentity = identity
		var acquisitionErr error
		durableAcquisition, acquisitionErr = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.DecideOperationID, service.durable.Policy)
		if errors.Is(acquisitionErr, platformidempotency.ErrIdempotencyConflict) {
			return ApprovalRequestCommandResult{}, ErrApprovalPolicyIdempotencyConflict
		}
		if acquisitionErr != nil {
			return ApprovalRequestCommandResult{}, acquisitionErr
		}
		if durableAcquisition.Decision() == platformidempotency.DecisionReturn {
			switch durableAcquisition.Result().State() {
			case platformidempotency.StateFailed:
				return ApprovalRequestCommandResult{}, durableApprovalRequestFailureError(durableAcquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return ApprovalRequestCommandResult{}, ErrApprovalPolicyCommandInProgress
			case platformidempotency.StateEstablished:
				stored, decodeErr := decodeApprovalRequestCommandResult(durableAcquisition.Result().ResultBody())
				if decodeErr != nil {
					return ApprovalRequestCommandResult{}, decodeErr
				}
				stored.Replayed = true
				return stored, nil
			}
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != commandFP {
				return ApprovalRequestCommandResult{}, ErrApprovalPolicyIdempotencyConflict
			}
			result := cloneApprovalRequestResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}
	request, err := service.repository.Get(ctx, command.RequestID)
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if !request.Version.Matches(*command.ExpectedVersion) {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyVersionConflict
	}
	if request.Status != ApprovalRequestStatusPending {
		return ApprovalRequestCommandResult{}, fmt.Errorf("%w: approval request is already decided", ErrApprovalPolicyApprovalRejected)
	}
	if request.Requester.UserID == actor.UserID {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicySelfApproval
	}
	if request.PolicyVersion != command.PolicyVersion || request.SubjectVersion != command.SubjectVersion || request.SubjectFingerprint != command.CandidateFingerprint {
		return ApprovalRequestCommandResult{}, fmt.Errorf("%w: decision evidence does not match the immutable request", ErrApprovalPolicyApprovalRejected)
	}
	decision, err := service.authorizer.AuthorizeApprovalDecision(ctx, actor, request)
	if err != nil {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyApprovalUnavailable
	}
	if err := validateAuthorization(decision, "finance.wfa.decide.approval.request"); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	nextVersion, err := request.Version.Advance()
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	decisionID := command.DecisionID
	if decisionID == uuid.Nil {
		decisionID = uuid.New()
	}
	now := service.clock().UTC()
	requestBefore := cloneApprovalRequest(request)
	request.Version = nextVersion
	if command.Outcome == ApprovalDecisionApproved {
		request.Status = ApprovalRequestStatusApproved
	} else {
		request.Status = ApprovalRequestStatusRejected
	}
	request.UpdatedAt = now
	request.Decisions = append(request.Decisions, ApprovalDecision{ID: decisionID, DecisionVersion: request.Version.Value(), Outcome: command.Outcome, Approver: actor, Reason: strings.TrimSpace(command.Reason), PolicyVersion: command.PolicyVersion, SubjectVersion: command.SubjectVersion, CandidateFingerprint: command.CandidateFingerprint, RecordedAt: now})
	for index := range request.Steps {
		request.Steps[index].Status = request.Status
	}
	if err := request.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	reference := ApprovalDecisionReference{ApprovalRequestID: request.ID, DecisionID: decisionID, PolicyVersion: command.PolicyVersion, DecisionVersion: request.Version.Value(), SubjectVersion: command.SubjectVersion, CandidateFingerprint: command.CandidateFingerprint, ApproverUserID: actor.UserID}
	auditRecord := ApprovalRequestAuditRecord{ApprovalRequestID: request.ID, ApprovalDecisionID: decisionID, ActorUserID: actor.UserID, ActorAuthenticationSubjectRef: actor.SubjectReference, ApproverUserID: actor.UserID, Action: "DecideApprovalRequest", ScopeIDs: request.Scope.CanonicalScopeIDs(), Reason: command.Reason, Permission: decision.Permission, PolicyReference: decision.PolicyReference, DecisionReference: decision.DecisionReference, PolicyVersion: command.PolicyVersion, RevisionVersion: request.Version.Value(), SubjectVersion: command.SubjectVersion, BeforeFingerprint: fingerprint(requestBefore), AfterFingerprint: fingerprint(request), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	event := ApprovalEvent{Type: ApprovalDecisionRecordedEventType, RequestID: request.ID, DecisionID: decisionID, DecisionVersion: request.Version.Value(), Outcome: command.Outcome, ApproverUserID: actor.UserID, AggregateVersion: request.Version.Value(), PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion, SubjectID: request.SubjectID, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, ScopeIDs: request.Scope.CanonicalScopeIDs(), OccurredAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	result = ApprovalRequestCommandResult{Request: cloneApprovalRequest(request), Reference: reference}
	if service.durable != nil {
		committer, ok := service.repository.(DurableApprovalRequestRepository)
		if !ok {
			return ApprovalRequestCommandResult{}, ErrInvalidApprovalPolicyService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return ApprovalRequestCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := request.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(commandFP), service.durable.DecideOperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			return ApprovalRequestCommandResult{}, metadataErr
		}
		if err := committer.SaveWithIdempotency(ctx, request, &requestBefore, auditRecord, event, DurableApprovalRequestCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata}); err != nil {
			return ApprovalRequestCommandResult{}, err
		}
	} else if err := service.repository.Save(ctx, request, &requestBefore, auditRecord, event); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedApprovalRequestResult{fingerprint: commandFP, result: cloneApprovalRequestResult(result)}
	}
	return result, nil
}

func validateAuthorization(decision AuthorizationDecision, permission string) error {
	switch decision.Outcome {
	case AuthorizationUnavailable:
		return ErrApprovalPolicyAuthorizationUnavailable
	case AuthorizationStale:
		return ErrApprovalPolicyAuthorizationStale
	case AuthorizationExpired:
		return ErrApprovalPolicyAuthorizationDenied
	}
	if !decision.Allowed || decision.Permission != permission {
		return ErrApprovalPolicyAuthorizationDenied
	}
	if decision.DecisionReference == uuid.Nil {
		return fmt.Errorf("%w: authorization decision reference is required", ErrApprovalPolicyAuthorizationDenied)
	}
	return nil
}

func approvalRequestDurableIdentity(actor Actor, key, operation string) (platformidempotency.IdempotencyIdentity, error) {
	scopeKey, err := json.Marshal(struct {
		Module    string    `json:"module"`
		ActorID   uuid.UUID `json:"actorId"`
		Operation string    `json:"operation"`
	}{Module: "workflow", ActorID: actor.UserID, Operation: operation})
	if err != nil {
		return platformidempotency.IdempotencyIdentity{}, err
	}
	return platformidempotency.NewOpaqueIdentity(string(scopeKey), key)
}

func (service *ApprovalRequestService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := durableApprovalRequestFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{code})
	if err != nil {
		return
	}
	metadata, err := platformidempotency.NewCommandResultMetadata(acquisition.Result().Identity(), acquisition.Result().Fingerprint(), acquisition.Result().OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
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

func durableApprovalRequestFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrApprovalPolicyAuthorizationDenied):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrApprovalPolicyAuthorizationUnavailable):
		return "AUTHORIZATION_UNAVAILABLE", true
	case errors.Is(err, ErrApprovalPolicyAuthorizationStale):
		return "AUTHORIZATION_STALE", true
	case errors.Is(err, ErrApprovalPolicyVersionConflict):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrApprovalPolicySelfApproval):
		return "SELF_APPROVAL_DENIED", true
	case errors.Is(err, ErrApprovalPolicyApprovalRejected):
		return "APPROVAL_REJECTED", true
	case errors.Is(err, ErrInvalidApprovalPolicyCommand):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func durableApprovalRequestFailureError(body []byte) error {
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrApprovalPolicyCommandInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrApprovalPolicyAuthorizationDenied
	case "AUTHORIZATION_UNAVAILABLE":
		return ErrApprovalPolicyAuthorizationUnavailable
	case "AUTHORIZATION_STALE":
		return ErrApprovalPolicyAuthorizationStale
	case "VERSION_CONFLICT":
		return ErrApprovalPolicyVersionConflict
	case "SELF_APPROVAL_DENIED":
		return ErrApprovalPolicySelfApproval
	case "APPROVAL_REJECTED":
		return ErrApprovalPolicyApprovalRejected
	case "VALIDATION_FAILED":
		return ErrApprovalPolicyDurableCommandFailed
	default:
		return ErrApprovalPolicyCommandInProgress
	}
}

func decodeApprovalRequestCommandResult(body []byte) (ApprovalRequestCommandResult, error) {
	var result ApprovalRequestCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	return result, nil
}

// MemoryApprovalRequestRepository keeps request state and all decision
// evidence in memory for local API fixtures and focused domain tests. Its
// Save method invokes audit and event ports before publishing the new value,
// so a failed side effect cannot leave a partially established request.
type MemoryApprovalRequestRepository struct {
	mu       sync.RWMutex
	requests map[uuid.UUID]ApprovalRequest
	audit    ApprovalRequestAuditRecorder
	events   ApprovalEventRecorder
}

func NewMemoryApprovalRequestRepository() *MemoryApprovalRequestRepository {
	return &MemoryApprovalRequestRepository{requests: make(map[uuid.UUID]ApprovalRequest)}
}

func (repository *MemoryApprovalRequestRepository) BindPorts(audit ApprovalRequestAuditRecorder, events ApprovalEventRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit, repository.events = audit, events
}

func (repository *MemoryApprovalRequestRepository) Get(_ context.Context, id uuid.UUID) (ApprovalRequest, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	request, ok := repository.requests[id]
	if !ok {
		return ApprovalRequest{}, ErrApprovalPolicyNotFound
	}
	return cloneApprovalRequest(request), nil
}

func (repository *MemoryApprovalRequestRepository) Save(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent) error {
	if repository == nil {
		return ErrInvalidApprovalPolicyService
	}
	if err := request.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, exists := repository.requests[request.ID]
	if before == nil {
		if exists {
			return ErrApprovalPolicyDuplicate
		}
	} else if !exists || current.Version.Value() != before.Version.Value() {
		return ErrApprovalPolicyVersionConflict
	}
	if repository.audit == nil || repository.events == nil {
		return ErrApprovalPolicyAuditUnavailable
	}
	if _, err := repository.audit.RecordApprovalRequestMutation(ctx, auditRecord); err != nil {
		return err
	}
	if err := repository.events.RecordApprovalEvent(ctx, event); err != nil {
		return err
	}
	repository.requests[request.ID] = cloneApprovalRequest(request)
	return nil
}

type MemoryApprovalRequestAuthorizer struct {
	Decision       AuthorizationDecision
	DecideDecision *AuthorizationDecision
	Err            error
}

func (authorizer MemoryApprovalRequestAuthorizer) AuthorizeApprovalDecision(_ context.Context, actor Actor, request ApprovalRequest) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	decision := authorizer.Decision
	if authorizer.DecideDecision != nil && actor.UserID != request.Requester.UserID {
		decision = *authorizer.DecideDecision
	}
	decision.ApprovedScopeIDs = append([]string(nil), decision.ApprovedScopeIDs...)
	return decision, nil
}

type MemoryApprovalRequestAuditRecorder struct {
	mu      sync.Mutex
	Records []ApprovalRequestAuditRecord
	Err     error
}

func (recorder *MemoryApprovalRequestAuditRecorder) RecordApprovalRequestMutation(_ context.Context, record ApprovalRequestAuditRecord) (uuid.UUID, error) {
	if recorder == nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return uuid.Nil, recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return uuid.New(), nil
}

type MemoryApprovalEventRecorder struct {
	mu     sync.Mutex
	Events []ApprovalEvent
	Err    error
}

func (recorder *MemoryApprovalEventRecorder) RecordApprovalEvent(_ context.Context, event ApprovalEvent) error {
	if recorder == nil {
		return ErrApprovalPolicyApprovalUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Events = append(recorder.Events, event)
	return nil
}

// ApprovalWorkflowPort is the smallest generic WFA boundary needed by policy
// activation. The policy module never creates a policy-specific command or
// event: it creates and validates the generic request/decision evidence here.
type ApprovalWorkflowPort interface {
	CreatePolicyApproval(context.Context, Actor, ApprovalPolicy, ApprovalPolicyCommand) (ApprovalRequest, error)
	ValidatePolicyApproval(context.Context, Actor, ApprovalPolicy, ApprovalDecisionReference) error
}

type ApprovalWorkflowPreparationPort interface {
	PreparePolicyApproval(context.Context, Actor, ApprovalPolicy, ApprovalPolicyCommand) (PreparedApprovalRequest, error)
}

type MemoryApprovalWorkflow struct {
	Requests *MemoryApprovalRequestRepository
	Service  *ApprovalRequestService
}

func NewMemoryApprovalWorkflow() (*MemoryApprovalWorkflow, error) {
	decide := AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: "finance.wfa.decide.approval.request", DecisionReference: uuid.New()}
	return newMemoryApprovalWorkflow(MemoryApprovalRequestAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: "finance.wfa.create.approval.request", DecisionReference: uuid.New()}, DecideDecision: &decide})
}

func NewMemoryApprovalWorkflowWithAuthorizer(authorizer ApprovalRequestAuthorizer) (*MemoryApprovalWorkflow, error) {
	if authorizer == nil {
		return nil, ErrInvalidApprovalPolicyService
	}
	return newMemoryApprovalWorkflow(authorizer)
}

func newMemoryApprovalWorkflow(authorizer ApprovalRequestAuthorizer) (*MemoryApprovalWorkflow, error) {
	repository := NewMemoryApprovalRequestRepository()
	audit := &MemoryApprovalRequestAuditRecorder{}
	events := &MemoryApprovalEventRecorder{}
	repository.BindPorts(audit, events)
	service, err := NewApprovalRequestService(repository, authorizer, audit, events, time.Now)
	if err != nil {
		return nil, err
	}
	return &MemoryApprovalWorkflow{Requests: repository, Service: service}, nil
}

func (workflow *MemoryApprovalWorkflow) CreatePolicyApproval(ctx context.Context, actor Actor, policy ApprovalPolicy, command ApprovalPolicyCommand) (ApprovalRequest, error) {
	if workflow == nil || workflow.Service == nil {
		return ApprovalRequest{}, ErrApprovalPolicyApprovalUnavailable
	}
	result, err := workflow.Service.CreateApprovalRequest(ctx, actor, CreateApprovalRequestCommand{RequestID: policy.ApprovalRequestID, SubjectType: "approval-policy", SubjectID: policy.ID.String(), SubjectVersion: policy.Version.Value(), SubjectFingerprint: policy.CandidateFingerprint(), PolicyID: policy.ID, PolicyVersion: policy.PolicyVersion, Scope: policy.Scope, Steps: policy.Steps, IdempotencyKey: command.IdempotencyKey + ":approval-request", CorrelationID: command.CorrelationID, CausationID: command.CausationID})
	if err != nil {
		return ApprovalRequest{}, err
	}
	return result.Request, nil
}

func (workflow *MemoryApprovalWorkflow) ValidatePolicyApproval(ctx context.Context, actor Actor, policy ApprovalPolicy, reference ApprovalDecisionReference) error {
	if workflow == nil || workflow.Requests == nil {
		return ErrApprovalPolicyApprovalUnavailable
	}
	request, err := workflow.Requests.Get(ctx, reference.ApprovalRequestID)
	if err != nil {
		return err
	}
	if request.Status != ApprovalRequestStatusApproved || request.PolicyID != policy.ID || request.PolicyVersion != policy.PolicyVersion || request.SubjectFingerprint != policy.CandidateFingerprint() {
		return ErrApprovalPolicyApprovalRejected
	}
	if request.Requester.UserID == reference.ApproverUserID {
		return ErrApprovalPolicySelfApproval
	}
	for _, decision := range request.Decisions {
		if decision.ID == reference.DecisionID && decision.DecisionVersion == reference.DecisionVersion && decision.PolicyVersion == reference.PolicyVersion && decision.Outcome == ApprovalDecisionApproved && decision.Approver.UserID == reference.ApproverUserID && decision.CandidateFingerprint == reference.CandidateFingerprint && decision.SubjectVersion == reference.SubjectVersion {
			return nil
		}
	}
	return ErrApprovalPolicyApprovalRejected
}

func cloneApprovalRequest(request ApprovalRequest) ApprovalRequest {
	request.Scope = cloneScope(request.Scope)
	request.Steps = append([]ApprovalRequestStep(nil), request.Steps...)
	for index := range request.Steps {
		request.Steps[index].RequiredRoles = append([]RoleRequirement(nil), request.Steps[index].RequiredRoles...)
	}
	request.Decisions = append([]ApprovalDecision(nil), request.Decisions...)
	return request
}

func cloneApprovalRequestResult(result ApprovalRequestCommandResult) ApprovalRequestCommandResult {
	result.Request = cloneApprovalRequest(result.Request)
	return result
}

func commandFingerprint(value any) string {
	data, _ := json.Marshal(value)
	return fingerprint(json.RawMessage(data))
}

func approvalRequestCreateCommandFingerprint(command CreateApprovalRequestCommand) string {
	command.IdempotencyKey = ""
	command.CorrelationID = ""
	command.CausationID = ""
	return commandFingerprint(command)
}

func approvalRequestDecisionCommandFingerprint(command DecideApprovalRequestCommand) string {
	command.IdempotencyKey = ""
	command.CorrelationID = ""
	command.CausationID = ""
	return commandFingerprint(command)
}

var _ ApprovalRequestRepository = (*MemoryApprovalRequestRepository)(nil)
var _ ApprovalWorkflowPort = (*MemoryApprovalWorkflow)(nil)
