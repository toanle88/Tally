package workflow

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/shopspring/decimal"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	ApprovalRequestedEventType        = "approval_requested"
	ApprovalDecisionRecordedEventType = "approval_decision_recorded"
	ApprovalEscalatedEventType        = "approval_escalated"
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

const ApprovalRequestStepStatusEscalated = "Escalated"

const ApprovalDecisionApplicationStatusNotReported = "not-reported"

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
	ID               uuid.UUID               `json:"id"`
	Sequence         int                     `json:"sequence"`
	Name             string                  `json:"name,omitempty"`
	RequiredRoles    []RoleRequirement       `json:"requiredRoles"`
	MinimumApprovals int                     `json:"minimumApprovals"`
	Delegation       DelegationConfiguration `json:"delegation,omitempty"`
	Escalation       EscalationConfiguration `json:"escalation,omitempty"`
	Status           string                  `json:"status"`
}

type ApprovalEscalation struct {
	ID                    uuid.UUID  `json:"id"`
	StepID                uuid.UUID  `json:"stepId"`
	Reason                string     `json:"reason"`
	Blocker               string     `json:"blocker,omitempty"`
	ResponsibleRoles      []string   `json:"responsibleRoles"`
	TargetRoles           []string   `json:"targetRoles"`
	ActorUserID           uuid.UUID  `json:"actorUserId,omitempty"`
	ActorSubjectReference string     `json:"actorSubjectReference,omitempty"`
	ProcessReference      string     `json:"processReference,omitempty"`
	PolicyID              uuid.UUID  `json:"policyId"`
	PolicyVersion         string     `json:"policyVersion"`
	SubjectVersion        int64      `json:"subjectVersion"`
	CandidateFingerprint  string     `json:"candidateFingerprint"`
	DueAt                 *time.Time `json:"dueAt,omitempty"`
	OccurredAt            time.Time  `json:"occurredAt"`
	AuditReference        uuid.UUID  `json:"auditReference"`
	CorrelationID         string     `json:"correlationId"`
	CausationID           string     `json:"causationId"`
}

type ApprovalDecision struct {
	ID                   uuid.UUID `json:"id"`
	DecisionVersion      int64     `json:"decisionVersion"`
	Outcome              string    `json:"outcome"`
	Approver             Actor     `json:"approver"`
	Reason               string    `json:"reason"`
	StepID               uuid.UUID `json:"stepId,omitempty"`
	DelegationReference  string    `json:"delegationReference,omitempty"`
	PolicyVersion        string    `json:"policyVersion"`
	SubjectVersion       int64     `json:"subjectVersion"`
	CandidateFingerprint string    `json:"candidateFingerprint"`
	RecordedAt           time.Time `json:"recordedAt"`
}

type ApprovalRequest struct {
	ID                  uuid.UUID                         `json:"id"`
	SubjectType         string                            `json:"subjectType"`
	SubjectID           string                            `json:"subjectId"`
	SubjectVersion      int64                             `json:"subjectVersion"`
	SubjectFingerprint  string                            `json:"subjectFingerprint"`
	SubjectSnapshot     ApprovalSubjectSnapshot           `json:"subjectSnapshot,omitempty"`
	SnapshotFingerprint string                            `json:"snapshotFingerprint,omitempty"`
	SourceContext       string                            `json:"sourceContext,omitempty"`
	Operation           string                            `json:"operation,omitempty"`
	TransactionType     string                            `json:"transactionType,omitempty"`
	DataClassification  string                            `json:"dataClassification,omitempty"`
	Dimensions          map[string]string                 `json:"dimensions,omitempty"`
	Amount              *decimal.Decimal                  `json:"amount,omitempty"`
	Currency            string                            `json:"currency,omitempty"`
	DueAt               *time.Time                        `json:"dueAt,omitempty"`
	ResponsibleRoles    []string                          `json:"responsibleRoles,omitempty"`
	PolicyID            uuid.UUID                         `json:"policyId"`
	PolicyVersion       string                            `json:"policyVersion"`
	Scope               ApprovalPolicyScope               `json:"scope"`
	Requester           Actor                             `json:"requester"`
	Steps               []ApprovalRequestStep             `json:"steps"`
	Decisions           []ApprovalDecision                `json:"decisions"`
	Escalations         []ApprovalEscalation              `json:"escalations,omitempty"`
	ActiveDelegation    *Delegation                       `json:"activeDelegation,omitempty"`
	Status              string                            `json:"status"`
	Version             aggregateversion.AggregateVersion `json:"version"`
	CorrelationID       string                            `json:"correlationId"`
	CausationID         string                            `json:"causationId"`
	CreatedAt           time.Time                         `json:"createdAt"`
	UpdatedAt           time.Time                         `json:"updatedAt"`
}

func (request ApprovalRequest) Validate() error {
	if request.ID == uuid.Nil || strings.TrimSpace(request.SubjectType) == "" || strings.TrimSpace(request.SubjectID) == "" || request.PolicyID == uuid.Nil || strings.TrimSpace(request.PolicyVersion) == "" {
		return fmt.Errorf("%w: approval request identity is incomplete", ErrInvalidApprovalPolicyCommand)
	}
	if request.SubjectVersion < 1 || strings.TrimSpace(request.SubjectFingerprint) == "" || request.Version.Value() < 1 {
		return fmt.Errorf("%w: approval request subject evidence is incomplete", ErrInvalidApprovalPolicyCommand)
	}
	if request.SnapshotFingerprint != "" {
		if err := request.SubjectSnapshot.Validate(); err != nil {
			return err
		}
		if request.SnapshotFingerprint != request.SubjectSnapshot.Fingerprint() {
			return fmt.Errorf("%w: approval request snapshot fingerprint is invalid", ErrInvalidApprovalPolicyCommand)
		}
	}
	if request.Amount != nil && request.Amount.IsNegative() {
		return fmt.Errorf("%w: approval request amount cannot be negative", ErrInvalidApprovalPolicyCommand)
	}
	if request.Currency != "" && !currencyCodePattern.MatchString(request.Currency) {
		return fmt.Errorf("%w: approval request currency is invalid", ErrInvalidApprovalPolicyCommand)
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
		switch step.Status {
		case ApprovalRequestStatusPending, ApprovalRequestStepStatusEscalated, ApprovalRequestStatusApproved, ApprovalRequestStatusRejected:
		default:
			return fmt.Errorf("%w: approval request step status is invalid", ErrInvalidApprovalPolicyCommand)
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
		if len(decision.DelegationReference) > 240 {
			return fmt.Errorf("%w: approval decision delegation reference is too long", ErrInvalidApprovalPolicyCommand)
		}
		if err := decision.Approver.Validate(); err != nil {
			return err
		}
		if decision.StepID != uuid.Nil {
			knownStep := false
			for _, step := range request.Steps {
				if step.ID == decision.StepID {
					knownStep = true
					break
				}
			}
			if !knownStep {
				return fmt.Errorf("%w: approval decision step is unknown", ErrInvalidApprovalPolicyCommand)
			}
		}
		if _, exists := decisionIDs[decision.ID]; exists {
			return fmt.Errorf("%w: duplicate approval decision", ErrInvalidApprovalPolicyCommand)
		}
		decisionIDs[decision.ID] = struct{}{}
	}
	escalationIDs := make(map[uuid.UUID]struct{}, len(request.Escalations))
	escalatedSteps := make(map[uuid.UUID]struct{}, len(request.Escalations))
	for _, escalation := range request.Escalations {
		if escalation.ID == uuid.Nil || escalation.StepID == uuid.Nil || escalation.PolicyID != request.PolicyID || escalation.PolicyVersion != request.PolicyVersion || escalation.SubjectVersion != request.SubjectVersion || escalation.CandidateFingerprint != request.SubjectFingerprint || escalation.OccurredAt.IsZero() {
			return fmt.Errorf("%w: approval escalation evidence is invalid", ErrInvalidApprovalPolicyCommand)
		}
		if escalation.Reason != ApprovalEscalationReasonOverdue && escalation.Reason != ApprovalEscalationReasonBlocked {
			return fmt.Errorf("%w: approval escalation reason is invalid", ErrInvalidApprovalPolicyCommand)
		}
		if escalation.Reason == ApprovalEscalationReasonBlocked && strings.TrimSpace(escalation.Blocker) == "" {
			return fmt.Errorf("%w: blocked approval escalation requires a blocker", ErrInvalidApprovalPolicyCommand)
		}
		if len(escalation.Blocker) > approvalEscalationBlockerMaxLength || len(escalation.ProcessReference) > approvalEscalationProcessReferenceMaxLength {
			return fmt.Errorf("%w: approval escalation evidence is too long", ErrInvalidApprovalPolicyCommand)
		}
		if escalation.ActorUserID == uuid.Nil && strings.TrimSpace(escalation.ProcessReference) == "" {
			return fmt.Errorf("%w: approval escalation actor or process is required", ErrInvalidApprovalPolicyCommand)
		}
		if escalation.ActorUserID != uuid.Nil && strings.TrimSpace(escalation.ProcessReference) != "" {
			return fmt.Errorf("%w: approval escalation cannot contain both actor and process evidence", ErrInvalidApprovalPolicyCommand)
		}
		if escalation.ActorUserID == uuid.Nil && strings.TrimSpace(escalation.ActorSubjectReference) != "" {
			return fmt.Errorf("%w: process escalation cannot contain actor subject evidence", ErrInvalidApprovalPolicyCommand)
		}
		if escalation.ActorUserID != uuid.Nil && strings.TrimSpace(escalation.ActorSubjectReference) == "" {
			return fmt.Errorf("%w: approval escalation actor subject reference is required", ErrInvalidApprovalPolicyCommand)
		}
		knownStep := false
		for _, step := range request.Steps {
			if step.ID == escalation.StepID {
				knownStep = true
				break
			}
		}
		if !knownStep {
			return fmt.Errorf("%w: approval escalation step is unknown", ErrInvalidApprovalPolicyCommand)
		}
		if len(escalation.TargetRoles) == 0 {
			return fmt.Errorf("%w: approval escalation target roles are required", ErrInvalidApprovalPolicyCommand)
		}
		if _, exists := escalationIDs[escalation.ID]; exists {
			return fmt.Errorf("%w: duplicate approval escalation", ErrInvalidApprovalPolicyCommand)
		}
		if _, exists := escalatedSteps[escalation.StepID]; exists {
			return fmt.Errorf("%w: duplicate approval escalation step", ErrInvalidApprovalPolicyCommand)
		}
		escalationIDs[escalation.ID] = struct{}{}
		escalatedSteps[escalation.StepID] = struct{}{}
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
	ResolvePolicy      bool
	Applicability      ApprovalApplicability
	TransactionType    string
	DataClassification string
	Dimensions         map[string]string
	Amount             *decimal.Decimal
	Currency           string
	SubjectSnapshot    ApprovalSubjectSnapshot
	IdempotencyKey     string
	CorrelationID      string
	CausationID        string
}

func (command CreateApprovalRequestCommand) Validate() error {
	if strings.TrimSpace(command.IdempotencyKey) == "" || strings.TrimSpace(command.SubjectType) == "" || strings.TrimSpace(command.SubjectID) == "" || command.SubjectVersion < 1 || strings.TrimSpace(command.SubjectFingerprint) == "" {
		return fmt.Errorf("%w: approval request command identity and idempotency evidence are required", ErrInvalidApprovalPolicyCommand)
	}
	if err := command.Scope.Validate(); err != nil {
		return err
	}
	if command.ResolvePolicy {
		if command.PolicyID != uuid.Nil || strings.TrimSpace(command.PolicyVersion) != "" || len(command.Steps) != 0 {
			return fmt.Errorf("%w: resolved approval request must not provide a policy snapshot", ErrInvalidApprovalPolicyCommand)
		}
		if err := command.Applicability.Validate(); err != nil {
			return err
		}
		if err := command.SubjectSnapshot.Validate(); err != nil {
			return err
		}
		if command.Amount == nil {
			return fmt.Errorf("%w: resolved approval request amount is required", ErrInvalidApprovalPolicyCommand)
		}
		if command.Currency == "" || !currencyCodePattern.MatchString(command.Currency) {
			return fmt.Errorf("%w: resolved approval request currency is invalid", ErrInvalidApprovalPolicyCommand)
		}
		if command.SubjectSnapshot.Currency != "" && command.SubjectSnapshot.Currency != command.Currency {
			return fmt.Errorf("%w: approval subject snapshot currency must match the request currency", ErrInvalidApprovalPolicyCommand)
		}
		if command.SubjectSnapshot.Amount != nil && !command.SubjectSnapshot.Amount.Equal(*command.Amount) {
			return fmt.Errorf("%w: approval subject snapshot amount must match the request amount", ErrInvalidApprovalPolicyCommand)
		}
		return nil
	}
	if command.PolicyID == uuid.Nil || strings.TrimSpace(command.PolicyVersion) == "" {
		return fmt.Errorf("%w: approval request policy identity is required", ErrInvalidApprovalPolicyCommand)
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
	StepID               uuid.UUID
	Outcome              string
	Reason               string
	DelegationID         uuid.UUID
	DelegationReference  string
	PolicyVersion        string
	SubjectVersion       int64
	CandidateFingerprint string
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

type EscalateApprovalCommand struct {
	RequestID            uuid.UUID
	ExpectedVersion      *aggregateversion.AggregateVersion
	EscalationID         uuid.UUID
	StepID               uuid.UUID
	Reason               string
	Blocker              string
	PolicyVersion        string
	SubjectVersion       int64
	CandidateFingerprint string
	IdempotencyKey       string
	CorrelationID        string
	CausationID          string
}

func (command EscalateApprovalCommand) Validate() error {
	if command.RequestID == uuid.Nil || command.EscalationID == uuid.Nil || command.StepID == uuid.Nil || strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: approval escalation identity and idempotency key are required", ErrInvalidApprovalPolicyCommand)
	}
	if command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 || command.SubjectVersion < 1 || strings.TrimSpace(command.PolicyVersion) == "" || strings.TrimSpace(command.CandidateFingerprint) == "" || len(strings.TrimSpace(command.CandidateFingerprint)) > approvalEscalationFingerprintMaxLength {
		return fmt.Errorf("%w: approval escalation evidence is incomplete", ErrInvalidApprovalPolicyCommand)
	}
	command.Reason = strings.ToLower(strings.TrimSpace(command.Reason))
	command.Blocker = strings.TrimSpace(command.Blocker)
	if command.Reason != ApprovalEscalationReasonOverdue && command.Reason != ApprovalEscalationReasonBlocked {
		return fmt.Errorf("%w: approval escalation reason is invalid", ErrInvalidApprovalPolicyCommand)
	}
	if command.Reason == ApprovalEscalationReasonBlocked && command.Blocker == "" {
		return fmt.Errorf("%w: blocked approval escalation requires a blocker", ErrInvalidApprovalPolicyCommand)
	}
	if len(command.Blocker) > approvalEscalationBlockerMaxLength {
		return fmt.Errorf("%w: approval escalation blocker is too long", ErrInvalidApprovalPolicyCommand)
	}
	return nil
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
	if len(strings.TrimSpace(command.DelegationReference)) > 240 {
		return fmt.Errorf("%w: approval decision delegation reference is too long", ErrInvalidApprovalPolicyCommand)
	}
	return nil
}

type ApprovalRequestAuditRecord struct {
	ApprovalRequestID             uuid.UUID
	ApprovalDecisionID            uuid.UUID
	ApprovalEscalationID          uuid.UUID
	ActorUserID                   uuid.UUID
	ActorAuthenticationSubjectRef string
	ApproverUserID                uuid.UUID
	Action                        string
	ScopeIDs                      []string
	Reason                        string
	StepID                        uuid.UUID
	DelegationReference           string
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
	Type                  string
	RequestID             uuid.UUID
	DecisionID            uuid.UUID
	EscalationID          uuid.UUID
	DecisionVersion       int64
	Outcome               string
	EscalationReason      string
	EscalationBlocker     string
	EscalationTargetRoles []string
	ApproverUserID        uuid.UUID
	StepID                uuid.UUID
	DelegationReference   string
	AggregateVersion      int64
	PolicyID              uuid.UUID
	PolicyVersion         string
	SubjectID             string
	SubjectVersion        int64
	CandidateFingerprint  string
	ScopeIDs              []string
	DataClassification    string
	OccurredAt            time.Time
	DecidedAt             time.Time
	EscalatedAt           time.Time
	CorrelationID         string
	CausationID           string
}

type ApprovalEventRecorder interface {
	RecordApprovalEvent(context.Context, ApprovalEvent) error
}

type ApprovalRequestRepository interface {
	Get(context.Context, uuid.UUID) (ApprovalRequest, error)
	Save(context.Context, ApprovalRequest, *ApprovalRequest, ApprovalRequestAuditRecord, ApprovalEvent) (uuid.UUID, error)
}

type ApprovalEscalationCandidate struct {
	RequestID            uuid.UUID
	ExpectedVersion      aggregateversion.AggregateVersion
	StepID               uuid.UUID
	PolicyVersion        string
	SubjectVersion       int64
	CandidateFingerprint string
	DueAt                time.Time
}

type ApprovalEscalationCandidateReader interface {
	ListDueApprovalEscalationCandidates(context.Context, time.Time, int) ([]ApprovalEscalationCandidate, error)
}

// ApprovalRequestReader is the read-only boundary used by the WFA worklist
// and detail projections. It is separate from the mutation repository so
// future projection storage can be introduced without changing command code.
type ApprovalRequestReader interface {
	Get(context.Context, uuid.UUID) (ApprovalRequest, error)
	List(context.Context) ([]ApprovalRequest, error)
}

// DelegationReader is the read-only boundary used by approval decisions and
// WFA projections. It keeps delegation persistence inside Workflow while
// allowing decision authorization to revalidate the live effective interval.
type DelegationReader interface {
	Get(context.Context, uuid.UUID) (Delegation, error)
	List(context.Context) ([]Delegation, error)
}

type DurableApprovalRequestServiceConfig struct {
	Database            platformidempotency.TxBeginner
	Coordinator         platformidempotency.DurableCoordinator
	Policy              platformidempotency.IdempotencyPolicy
	CreateOperationID   string
	DecideOperationID   string
	EscalateOperationID string
}

type DurableApprovalRequestCommit struct {
	Coordinator   platformidempotency.DurableCoordinator
	Acquisition   platformidempotency.DurableAcquisition
	Result        platformidempotency.CommandResultMetadata
	ResultBuilder func(uuid.UUID) (platformidempotency.CommandResultMetadata, error)
}

type DurableApprovalRequestRepository interface {
	SaveWithIdempotency(context.Context, ApprovalRequest, *ApprovalRequest, ApprovalRequestAuditRecord, ApprovalEvent, DurableApprovalRequestCommit) (uuid.UUID, error)
}

type ApprovalRequestAuthorizer interface {
	AuthorizeApprovalDecision(context.Context, Actor, ApprovalRequest) (AuthorizationDecision, error)
}

type ApprovalRequestEscalationAuthorizer interface {
	AuthorizeApprovalEscalation(context.Context, Actor, ApprovalRequest, ApprovalRequestStep) (AuthorizationDecision, error)
}

type ApprovalRequestEscalationProcessAuthorizer interface {
	AuthorizeApprovalEscalationProcess(context.Context, string, ApprovalRequest, ApprovalRequestStep) (AuthorizationDecision, error)
}

// ApprovalRequestStepAuthorizer is an optional extension used by the runtime
// adapter to evaluate the current step's role and delegation eligibility. The
// generic authorizer remains the compatibility boundary for existing callers.
type ApprovalRequestStepAuthorizer interface {
	AuthorizeApprovalDecisionForStep(context.Context, Actor, ApprovalRequest, ApprovalRequestStep) (AuthorizationDecision, error)
}

// ApprovalRequestDecisionAuthorizer carries the command's delegation evidence
// into the runtime authorization boundary. It prevents a configured delegate
// role from being treated as eligible without a recorded delegation reference.
type ApprovalRequestDecisionAuthorizer interface {
	AuthorizeApprovalRequestDecision(context.Context, Actor, ApprovalRequest, ApprovalRequestStep, string) (AuthorizationDecision, error)
}

// ApprovalRequestDelegationAuthorizer is the stronger decision boundary used
// when a delegation aggregate has been resolved. The legacy interface remains
// supported for direct approver callers and older focused tests.
type ApprovalRequestDelegationAuthorizer interface {
	AuthorizeApprovalRequestDecisionWithDelegation(context.Context, Actor, ApprovalRequest, ApprovalRequestStep, string, *Delegation) (AuthorizationDecision, error)
}

type ApprovalRequestReadAuthorizer interface {
	AuthorizeApprovalRequestRead(context.Context, Actor, ApprovalRequest) (AuthorizationDecision, error)
}

type ApprovalRequestCommandResult struct {
	Request    ApprovalRequest
	Reference  ApprovalDecisionReference
	Escalation *ApprovalEscalation
	Replayed   bool
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
	repository       ApprovalRequestRepository
	authorizer       ApprovalRequestAuthorizer
	audit            ApprovalRequestAuditRecorder
	events           ApprovalEventRecorder
	clock            func() time.Time
	durable          *DurableApprovalRequestServiceConfig
	policyResolver   ApprovalPolicyResolver
	policyRepository ApprovalPolicyRepository
	delegationReader DelegationReader

	mu          sync.Mutex
	idempotency map[string]storedApprovalRequestResult
}

// BindPolicyResolver enables the ordinary approval-bearing request path while
// keeping the explicit policy-maintenance request path independent.
func (service *ApprovalRequestService) BindPolicyResolver(resolver ApprovalPolicyResolver) error {
	if service == nil || resolver == nil {
		return ErrInvalidApprovalPolicyService
	}
	service.policyResolver = resolver
	return nil
}

// BindPolicyRepository enables decision-time validation of the request's
// immutable policy version and effective interval.
func (service *ApprovalRequestService) BindPolicyRepository(repository ApprovalPolicyRepository) error {
	if service == nil || repository == nil {
		return ErrInvalidApprovalPolicyService
	}
	service.policyRepository = repository
	return nil
}

// BindDelegationReader enables fail-closed decision-time delegation checks and
// read projections without coupling approval requests to a delegation adapter.
func (service *ApprovalRequestService) BindDelegationReader(reader DelegationReader) error {
	if service == nil || reader == nil {
		return ErrInvalidApprovalPolicyService
	}
	service.delegationReader = reader
	return nil
}

type storedApprovalRequestResult struct {
	fingerprint string
	result      ApprovalRequestCommandResult
}

func NewApprovalRequestService(repository ApprovalRequestRepository, authorizer ApprovalRequestAuthorizer, audit ApprovalRequestAuditRecorder, events ApprovalEventRecorder, clock func() time.Time) (*ApprovalRequestService, error) {
	return newApprovalRequestService(repository, authorizer, audit, events, clock, nil)
}

func NewApprovalRequestServiceWithDurableIdempotency(repository ApprovalRequestRepository, authorizer ApprovalRequestAuthorizer, audit ApprovalRequestAuditRecorder, events ApprovalEventRecorder, clock func() time.Time, durable DurableApprovalRequestServiceConfig) (*ApprovalRequestService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.CreateOperationID) == "" || strings.TrimSpace(durable.DecideOperationID) == "" || strings.TrimSpace(durable.EscalateOperationID) == "" {
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
	if command.ResolvePolicy {
		if service.policyResolver == nil {
			return ApprovalRequestCommandResult{}, ErrApprovalPolicySubjectUnavailable
		}
		resolved, resolveErr := service.policyResolver.Resolve(ctx, ApprovalPolicySelection{
			Applicability: command.Applicability,
			Scope:         command.Scope,
			Amount:        command.Amount,
			Currency:      command.Currency,
			AsOf:          service.clock().UTC(),
		})
		if resolveErr != nil {
			return ApprovalRequestCommandResult{}, resolveErr
		}
		command.PolicyID = resolved.ID
		command.PolicyVersion = resolved.PolicyVersion
		command.Steps = cloneSteps(resolved.Steps)
		command.ResolvePolicy = false
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
		if _, err := committer.SaveWithIdempotency(ctx, request, nil, auditRecord, event, DurableApprovalRequestCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata}); err != nil {
			return ApprovalRequestCommandResult{}, err
		}
	} else if _, err := service.repository.Save(ctx, request, nil, auditRecord, event); err != nil {
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
		steps[index] = ApprovalRequestStep{ID: step.ID, Sequence: step.Sequence, Name: step.Name, RequiredRoles: append([]RoleRequirement(nil), step.RequiredRoles...), MinimumApprovals: step.MinimumApprovals, Delegation: step.Delegation, Escalation: step.Escalation, Status: ApprovalRequestStatusPending}
	}
	request := ApprovalRequest{ID: requestID, SubjectType: strings.TrimSpace(command.SubjectType), SubjectID: strings.TrimSpace(command.SubjectID), SubjectVersion: command.SubjectVersion, SubjectFingerprint: command.SubjectFingerprint, PolicyID: command.PolicyID, PolicyVersion: command.PolicyVersion, SubjectSnapshot: cloneSubjectSnapshot(command.SubjectSnapshot), SourceContext: strings.TrimSpace(command.Applicability.SourceContext), Operation: strings.TrimSpace(command.Applicability.Operation), TransactionType: strings.TrimSpace(command.TransactionType), DataClassification: strings.TrimSpace(command.DataClassification), Dimensions: cloneStringMap(command.Dimensions), Amount: cloneDecimal(command.Amount), Currency: strings.TrimSpace(command.Currency), SnapshotFingerprint: command.SubjectSnapshot.Fingerprint(), Scope: cloneScope(command.Scope), Requester: actor, Steps: steps, ResponsibleRoles: approvalRequestResponsibleRoles(steps), Status: ApprovalRequestStatusPending, Version: aggregateversion.Initial(), CorrelationID: command.CorrelationID, CausationID: command.CausationID, CreatedAt: now, UpdatedAt: now}
	if len(steps) > 0 && steps[0].Escalation.Enabled {
		dueAt := now.Add(time.Duration(steps[0].Escalation.AfterSeconds) * time.Second)
		request.DueAt = &dueAt
	}
	if err := request.Validate(); err != nil {
		return PreparedApprovalRequest{}, err
	}
	decision, err := service.authorizer.AuthorizeApprovalDecision(ctx, actor, request)
	if err != nil {
		return PreparedApprovalRequest{}, ErrApprovalPolicyApprovalUnavailable
	}
	if err := validateAuthorization(decision, CreateApprovalRequestsPermission, request.Scope.CanonicalScopeIDs()); err != nil {
		return PreparedApprovalRequest{}, err
	}
	auditRecord := ApprovalRequestAuditRecord{ApprovalRequestID: request.ID, ActorUserID: actor.UserID, ActorAuthenticationSubjectRef: actor.SubjectReference, Action: "CreateApprovalRequest", ScopeIDs: request.Scope.CanonicalScopeIDs(), Permission: decision.Permission, PolicyReference: decision.PolicyReference, DecisionReference: decision.DecisionReference, PolicyVersion: request.PolicyVersion, RevisionVersion: request.Version.Value(), SubjectVersion: request.SubjectVersion, AfterFingerprint: request.SubjectFingerprint, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	event := ApprovalEvent{Type: ApprovalRequestedEventType, RequestID: request.ID, AggregateVersion: request.Version.Value(), PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion, SubjectID: request.SubjectID, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, ScopeIDs: request.Scope.CanonicalScopeIDs(), DataClassification: request.DataClassification, OccurredAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
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
	if request.PolicyVersion != command.PolicyVersion || request.SubjectVersion != command.SubjectVersion || request.SubjectFingerprint != command.CandidateFingerprint {
		return ApprovalRequestCommandResult{}, fmt.Errorf("%w: decision evidence does not match the immutable request", ErrApprovalPolicyApprovalRejected)
	}
	if err := service.validateDecisionPolicy(ctx, request); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	step, err := actionableApprovalRequestStep(request, command.StepID)
	if err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	delegationReference := strings.TrimSpace(command.DelegationReference)
	var delegation *Delegation
	if command.DelegationID != uuid.Nil || (service.delegationReader != nil && delegationReference != "") {
		if service.delegationReader == nil {
			return ApprovalRequestCommandResult{}, ErrDelegationAuthorizationUnavailable
		}
		delegationID := command.DelegationID
		if delegationID == uuid.Nil {
			delegationID, err = uuid.Parse(delegationReference)
			if err != nil || delegationID == uuid.Nil {
				return ApprovalRequestCommandResult{}, ErrDelegationAuthorizationDenied
			}
		}
		resolved, resolveErr := service.delegationReader.Get(ctx, delegationID)
		if resolveErr != nil {
			if errors.Is(resolveErr, ErrDelegationNotFound) {
				return ApprovalRequestCommandResult{}, ErrDelegationAuthorizationDenied
			}
			return ApprovalRequestCommandResult{}, ErrDelegationAuthorizationUnavailable
		}
		if !resolved.ActiveAt(service.clock().UTC()) {
			return ApprovalRequestCommandResult{}, ErrDelegationExpired
		}
		if resolved.PolicyID != request.PolicyID || resolved.PolicyVersion != request.PolicyVersion {
			return ApprovalRequestCommandResult{}, ErrDelegationPolicyConflict
		}
		if !resolved.Scope.Contains(request.Scope.CanonicalScopeIDs()) {
			return ApprovalRequestCommandResult{}, ErrDelegationScopeConflict
		}
		delegation = &resolved
		delegationReference = resolved.ID.String()
	}
	decision, err := authorizeApprovalRequestDecision(ctx, service.authorizer, actor, request, step, delegationReference, delegation)
	if err != nil {
		return ApprovalRequestCommandResult{}, ErrApprovalPolicyApprovalUnavailable
	}
	if err := validateAuthorization(decision, DecideApprovalRequestsPermission, request.Scope.CanonicalScopeIDs()); err != nil {
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
	request.Decisions = append(request.Decisions, ApprovalDecision{ID: decisionID, DecisionVersion: request.Version.Value(), Outcome: command.Outcome, Approver: actor, Reason: strings.TrimSpace(command.Reason), StepID: step.ID, DelegationReference: delegationReference, PolicyVersion: command.PolicyVersion, SubjectVersion: command.SubjectVersion, CandidateFingerprint: command.CandidateFingerprint, RecordedAt: now})
	for index := range request.Steps {
		request.Steps[index].Status = request.Status
	}
	if err := request.Validate(); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	reference := ApprovalDecisionReference{ApprovalRequestID: request.ID, DecisionID: decisionID, PolicyVersion: command.PolicyVersion, DecisionVersion: request.Version.Value(), SubjectVersion: command.SubjectVersion, CandidateFingerprint: command.CandidateFingerprint, ApproverUserID: actor.UserID, DecidedAt: now}
	auditRecord := ApprovalRequestAuditRecord{ApprovalRequestID: request.ID, ApprovalDecisionID: decisionID, ActorUserID: actor.UserID, ActorAuthenticationSubjectRef: actor.SubjectReference, ApproverUserID: actor.UserID, Action: "DecideApprovalRequest", ScopeIDs: request.Scope.CanonicalScopeIDs(), Reason: command.Reason, StepID: step.ID, DelegationReference: delegationReference, Permission: decision.Permission, PolicyReference: decision.PolicyReference, DecisionReference: decision.DecisionReference, PolicyVersion: command.PolicyVersion, RevisionVersion: request.Version.Value(), SubjectVersion: command.SubjectVersion, BeforeFingerprint: fingerprint(requestBefore), AfterFingerprint: fingerprint(request), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	event := ApprovalEvent{Type: ApprovalDecisionRecordedEventType, RequestID: request.ID, DecisionID: decisionID, DecisionVersion: request.Version.Value(), Outcome: command.Outcome, ApproverUserID: actor.UserID, StepID: step.ID, DelegationReference: delegationReference, AggregateVersion: request.Version.Value(), PolicyID: request.PolicyID, PolicyVersion: request.PolicyVersion, SubjectID: request.SubjectID, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, ScopeIDs: request.Scope.CanonicalScopeIDs(), DataClassification: request.DataClassification, OccurredAt: now, DecidedAt: now, CorrelationID: command.CorrelationID, CausationID: command.CausationID}
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
		if _, err := committer.SaveWithIdempotency(ctx, request, &requestBefore, auditRecord, event, DurableApprovalRequestCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata}); err != nil {
			return ApprovalRequestCommandResult{}, err
		}
	} else if _, err := service.repository.Save(ctx, request, &requestBefore, auditRecord, event); err != nil {
		return ApprovalRequestCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedApprovalRequestResult{fingerprint: commandFP, result: cloneApprovalRequestResult(result)}
	}
	return result, nil
}

func authorizeApprovalDecisionForStep(ctx context.Context, authorizer ApprovalRequestAuthorizer, actor Actor, request ApprovalRequest, step ApprovalRequestStep) (AuthorizationDecision, error) {
	return authorizeApprovalRequestDecision(ctx, authorizer, actor, request, step, "", nil)
}

func authorizeApprovalRequestDecision(ctx context.Context, authorizer ApprovalRequestAuthorizer, actor Actor, request ApprovalRequest, step ApprovalRequestStep, delegationReference string, delegation *Delegation) (AuthorizationDecision, error) {
	if delegationAuthorizer, ok := authorizer.(ApprovalRequestDelegationAuthorizer); ok {
		return delegationAuthorizer.AuthorizeApprovalRequestDecisionWithDelegation(ctx, actor, request, step, delegationReference, delegation)
	}
	if decisionAuthorizer, ok := authorizer.(ApprovalRequestDecisionAuthorizer); ok {
		return decisionAuthorizer.AuthorizeApprovalRequestDecision(ctx, actor, request, step, delegationReference)
	}
	if stepAuthorizer, ok := authorizer.(ApprovalRequestStepAuthorizer); ok {
		return stepAuthorizer.AuthorizeApprovalDecisionForStep(ctx, actor, request, step)
	}
	return authorizer.AuthorizeApprovalDecision(ctx, actor, request)
}

func actionableApprovalRequestStep(request ApprovalRequest, requestedStepID uuid.UUID) (ApprovalRequestStep, error) {
	var actionable *ApprovalRequestStep
	for index := range request.Steps {
		step := &request.Steps[index]
		if step.Status != ApprovalRequestStatusPending && step.Status != ApprovalRequestStepStatusEscalated {
			continue
		}
		if actionable == nil || step.Sequence < actionable.Sequence {
			actionable = step
		}
	}
	if actionable == nil || (requestedStepID != uuid.Nil && actionable.ID != requestedStepID) {
		return ApprovalRequestStep{}, ErrApprovalPolicyStepConflict
	}
	return *actionable, nil
}

func (service *ApprovalRequestService) validateDecisionPolicy(ctx context.Context, request ApprovalRequest) error {
	if service == nil || service.policyRepository == nil {
		return nil
	}
	policy, err := service.policyRepository.Get(ctx, request.PolicyID)
	if err != nil {
		if errors.Is(err, ErrApprovalPolicyNotFound) {
			return ErrApprovalPolicyVersionConflict
		}
		return fmt.Errorf("%w: %v", ErrApprovalPolicyResolverUnavailable, err)
	}
	var selected *ApprovalPolicy
	if policy.PolicyVersion == request.PolicyVersion {
		selected = &policy
	} else {
		for index := range policy.History {
			if policy.History[index].Snapshot.PolicyVersion == request.PolicyVersion {
				historical := snapshotToPolicy(policy.History[index])
				selected = &historical
				break
			}
		}
	}
	if selected == nil {
		return ErrApprovalPolicyVersionConflict
	}
	now := service.clock().UTC()
	if selected.Status != ApprovalPolicyStatusActive || now.Before(selected.EffectiveFrom.UTC()) || (selected.EffectiveTo != nil && now.After(selected.EffectiveTo.UTC())) {
		return ErrApprovalPolicyExpired
	}
	return nil
}

func validateAuthorization(decision AuthorizationDecision, permission string, requestedScopeIDs []string) error {
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
	approved := make(map[string]struct{}, len(decision.ApprovedScopeIDs))
	for _, scopeID := range decision.ApprovedScopeIDs {
		approved[strings.TrimSpace(scopeID)] = struct{}{}
	}
	for _, scopeID := range requestedScopeIDs {
		if _, ok := approved["*"]; ok {
			break
		}
		if _, ok := approved[strings.TrimSpace(scopeID)]; !ok {
			return fmt.Errorf("%w: scope %q is outside the requesting actor scope", ErrApprovalPolicyAuthorizationDenied, scopeID)
		}
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
	case errors.Is(err, ErrApprovalPolicyResolverUnavailable):
		return "POLICY_RESOLUTION_UNAVAILABLE", true
	case errors.Is(err, ErrApprovalPolicyNotApplicable):
		return "POLICY_NOT_APPLICABLE", true
	case errors.Is(err, ErrApprovalPolicyAmbiguous):
		return "POLICY_AMBIGUOUS", true
	case errors.Is(err, ErrApprovalPolicySubjectUnavailable):
		return "SUBJECT_UNAVAILABLE", true
	case errors.Is(err, ErrApprovalPolicySubjectConflict):
		return "SUBJECT_CONFLICT", true
	case errors.Is(err, ErrApprovalPolicyVersionConflict):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrApprovalPolicyExpired):
		return "APPROVAL_EXPIRED", true
	case errors.Is(err, ErrApprovalPolicyStepConflict):
		return "STEP_CONFLICT", true
	case errors.Is(err, ErrApprovalPolicyEscalationNotEligible):
		return "ESCALATION_NOT_ELIGIBLE", true
	case errors.Is(err, ErrApprovalPolicyAlreadyEscalated):
		return "ALREADY_ESCALATED", true
	case errors.Is(err, ErrApprovalPolicyDuplicate):
		return "DUPLICATE", true
	case errors.Is(err, ErrApprovalPolicyApprovalUnavailable):
		return "APPROVAL_UNAVAILABLE", true
	case errors.Is(err, ErrApprovalPolicyAuditUnavailable):
		return "AUDIT_UNAVAILABLE", true
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
	case "POLICY_RESOLUTION_UNAVAILABLE":
		return ErrApprovalPolicyResolverUnavailable
	case "POLICY_NOT_APPLICABLE":
		return ErrApprovalPolicyNotApplicable
	case "POLICY_AMBIGUOUS":
		return ErrApprovalPolicyAmbiguous
	case "SUBJECT_UNAVAILABLE":
		return ErrApprovalPolicySubjectUnavailable
	case "SUBJECT_CONFLICT":
		return ErrApprovalPolicySubjectConflict
	case "VERSION_CONFLICT":
		return ErrApprovalPolicyVersionConflict
	case "APPROVAL_EXPIRED":
		return ErrApprovalPolicyExpired
	case "STEP_CONFLICT":
		return ErrApprovalPolicyStepConflict
	case "ESCALATION_NOT_ELIGIBLE":
		return ErrApprovalPolicyEscalationNotEligible
	case "ALREADY_ESCALATED":
		return ErrApprovalPolicyAlreadyEscalated
	case "DUPLICATE":
		return ErrApprovalPolicyDuplicate
	case "APPROVAL_UNAVAILABLE":
		return ErrApprovalPolicyApprovalUnavailable
	case "AUDIT_UNAVAILABLE":
		return ErrApprovalPolicyAuditUnavailable
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

func (repository *MemoryApprovalRequestRepository) List(_ context.Context) ([]ApprovalRequest, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	requests := make([]ApprovalRequest, 0, len(repository.requests))
	for _, request := range repository.requests {
		requests = append(requests, cloneApprovalRequest(request))
	}
	return requests, nil
}

func (repository *MemoryApprovalRequestRepository) ListDueApprovalEscalationCandidates(_ context.Context, now time.Time, limit int) ([]ApprovalEscalationCandidate, error) {
	if repository == nil || limit < 1 || limit > 100 {
		return nil, ErrInvalidApprovalPolicyCommand
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	candidates := make([]ApprovalEscalationCandidate, 0, limit)
	for _, request := range repository.requests {
		if request.Status != ApprovalRequestStatusPending || request.DueAt == nil || request.DueAt.After(now.UTC()) {
			continue
		}
		step, err := actionableApprovalRequestStep(request, uuid.Nil)
		if err != nil || step.Status != ApprovalRequestStatusPending || !step.Escalation.Enabled || step.Escalation.AfterSeconds < 1 || len(step.Escalation.EscalateToRoles) == 0 {
			continue
		}
		alreadyEscalated := false
		for _, escalation := range request.Escalations {
			if escalation.StepID == step.ID {
				alreadyEscalated = true
				break
			}
		}
		if alreadyEscalated {
			continue
		}
		candidates = append(candidates, ApprovalEscalationCandidate{RequestID: request.ID, ExpectedVersion: request.Version, StepID: step.ID, PolicyVersion: request.PolicyVersion, SubjectVersion: request.SubjectVersion, CandidateFingerprint: request.SubjectFingerprint, DueAt: request.DueAt.UTC()})
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].DueAt.Equal(candidates[right].DueAt) {
			return candidates[left].RequestID.String() < candidates[right].RequestID.String()
		}
		return candidates[left].DueAt.Before(candidates[right].DueAt)
	})
	if len(candidates) > limit {
		candidates = candidates[:limit]
	}
	return candidates, nil
}

func (repository *MemoryApprovalRequestRepository) Save(ctx context.Context, request ApprovalRequest, before *ApprovalRequest, auditRecord ApprovalRequestAuditRecord, event ApprovalEvent) (uuid.UUID, error) {
	if repository == nil {
		return uuid.Nil, ErrInvalidApprovalPolicyService
	}
	if err := request.Validate(); err != nil {
		return uuid.Nil, err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, exists := repository.requests[request.ID]
	if before == nil {
		if exists {
			return uuid.Nil, ErrApprovalPolicyDuplicate
		}
	} else if !exists || current.Version.Value() != before.Version.Value() {
		return uuid.Nil, ErrApprovalPolicyVersionConflict
	}
	if repository.audit == nil || repository.events == nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	auditReference, err := repository.audit.RecordApprovalRequestMutation(ctx, auditRecord)
	if err != nil {
		return uuid.Nil, err
	}
	if auditReference == uuid.Nil {
		return uuid.Nil, ErrApprovalPolicyAuditUnavailable
	}
	if err := repository.events.RecordApprovalEvent(ctx, event); err != nil {
		return uuid.Nil, err
	}
	stored := cloneApprovalRequest(request)
	if auditRecord.ApprovalEscalationID != uuid.Nil {
		for index := range stored.Escalations {
			if stored.Escalations[index].ID == auditRecord.ApprovalEscalationID {
				stored.Escalations[index].AuditReference = auditReference
				break
			}
		}
	}
	repository.requests[request.ID] = stored
	return auditReference, nil
}

type MemoryApprovalRequestAuthorizer struct {
	Decision         AuthorizationDecision
	DecideDecision   *AuthorizationDecision
	EscalateDecision *AuthorizationDecision
	ReadDecision     *AuthorizationDecision
	Err              error
}

func (authorizer MemoryApprovalRequestAuthorizer) AuthorizeApprovalDecisionForStep(ctx context.Context, actor Actor, request ApprovalRequest, _ ApprovalRequestStep) (AuthorizationDecision, error) {
	return authorizer.AuthorizeApprovalDecision(ctx, actor, request)
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

func (authorizer MemoryApprovalRequestAuthorizer) AuthorizeApprovalEscalation(_ context.Context, _ Actor, _ ApprovalRequest, _ ApprovalRequestStep) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	decision := authorizer.Decision
	if authorizer.EscalateDecision != nil {
		decision = *authorizer.EscalateDecision
	}
	if decision.Permission == "" {
		decision.Permission = EscalateApprovalPermission
	}
	if decision.DecisionReference == uuid.Nil {
		decision.DecisionReference = uuid.New()
	}
	decision.ApprovedScopeIDs = append([]string(nil), decision.ApprovedScopeIDs...)
	return decision, nil
}

func (authorizer MemoryApprovalRequestAuthorizer) AuthorizeApprovalEscalationProcess(_ context.Context, processReference string, _ ApprovalRequest, _ ApprovalRequestStep) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	if strings.TrimSpace(processReference) == "" {
		return AuthorizationDecision{}, ErrApprovalPolicyAuthorizationUnavailable
	}
	decision := authorizer.Decision
	if authorizer.EscalateDecision != nil {
		decision = *authorizer.EscalateDecision
	}
	if decision.Permission == "" {
		decision.Permission = EscalateApprovalPermission
	}
	if decision.DecisionReference == uuid.Nil {
		decision.DecisionReference = uuid.New()
	}
	decision.ApprovedScopeIDs = append([]string(nil), decision.ApprovedScopeIDs...)
	return decision, nil
}

func (authorizer MemoryApprovalRequestAuthorizer) AuthorizeApprovalRequestRead(_ context.Context, _ Actor, _ ApprovalRequest) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	if authorizer.ReadDecision != nil {
		decision := *authorizer.ReadDecision
		decision.ApprovedScopeIDs = append([]string(nil), decision.ApprovedScopeIDs...)
		return decision, nil
	}
	decision := authorizer.Decision
	decision.Permission = ReadApprovalRequestsPermission
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
	decide := AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: "finance.wfa.decide.approval.request", ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()}
	return newMemoryApprovalWorkflow(MemoryApprovalRequestAuthorizer{Decision: AuthorizationDecision{Allowed: true, Outcome: AuthorizationAllowed, Permission: "finance.wfa.create.approval.request", ApprovedScopeIDs: []string{"*"}, DecisionReference: uuid.New()}, DecideDecision: &decide})
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
	request.SubjectSnapshot = cloneSubjectSnapshot(request.SubjectSnapshot)
	request.Dimensions = cloneStringMap(request.Dimensions)
	request.Amount = cloneDecimal(request.Amount)
	request.DueAt = cloneTime(request.DueAt)
	request.ResponsibleRoles = append([]string(nil), request.ResponsibleRoles...)
	request.Steps = append([]ApprovalRequestStep(nil), request.Steps...)
	for index := range request.Steps {
		request.Steps[index].RequiredRoles = append([]RoleRequirement(nil), request.Steps[index].RequiredRoles...)
		request.Steps[index].Delegation.DelegateToRoles = append([]string(nil), request.Steps[index].Delegation.DelegateToRoles...)
		request.Steps[index].Escalation.EscalateToRoles = append([]string(nil), request.Steps[index].Escalation.EscalateToRoles...)
	}
	request.Decisions = append([]ApprovalDecision(nil), request.Decisions...)
	request.Escalations = append([]ApprovalEscalation(nil), request.Escalations...)
	for index := range request.Escalations {
		request.Escalations[index] = cloneApprovalEscalation(request.Escalations[index])
	}
	if request.ActiveDelegation != nil {
		delegation := *request.ActiveDelegation
		delegation.Scope = cloneScope(request.ActiveDelegation.Scope)
		request.ActiveDelegation = &delegation
	}
	return request
}

func approvalRequestResponsibleRoles(steps []ApprovalRequestStep) []string {
	if len(steps) == 0 {
		return nil
	}
	seen := make(map[string]struct{})
	roles := make([]string, 0)
	for _, step := range steps {
		for _, requirement := range step.RequiredRoles {
			if _, exists := seen[requirement.Role]; exists {
				continue
			}
			seen[requirement.Role] = struct{}{}
			roles = append(roles, requirement.Role)
		}
	}
	return roles
}

func cloneApprovalRequestResult(result ApprovalRequestCommandResult) ApprovalRequestCommandResult {
	result.Request = cloneApprovalRequest(result.Request)
	if result.Escalation != nil {
		escalation := cloneApprovalEscalation(*result.Escalation)
		result.Escalation = &escalation
	}
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
