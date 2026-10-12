package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/idempotency"
	"github.com/toanle88/Tally/internal/platform/money"
)

const (
	PostingApprovalOutcomeApproved = "approved"
	PostingApprovalOutcomeRejected = "rejected"

	PostingApprovalApplicationApplied   = "applied"
	PostingApprovalApplicationUnchanged = "unchanged"
	PostingApprovalApplicationConflict  = "conflict"

	PostingApprovalAction      = "apply-journal-approval-decision"
	PostingApprovalSubjectType = "journal-entry"
)

var (
	ErrInvalidPostingApprovalDecisionCommand       = errors.New("invalid posting approval decision command")
	ErrPostingApprovalDecisionAuthorizationDenied  = errors.New("posting approval decision authorization denied")
	ErrPostingApprovalDecisionAuthorizationMissing = errors.New("posting approval decision authorization unavailable")
	ErrPostingApprovalDecisionUnavailable          = errors.New("posting approval decision unavailable")
	ErrPostingApprovalDecisionReferenceInvalid     = errors.New("posting approval decision reference is invalid")
	ErrPostingApprovalDecisionNotFound             = errors.New("pending posting journal not found")
	ErrPostingApprovalDecisionAlreadyApplied       = errors.New("posting approval decision already applied")
	ErrPostingApprovalDecisionVersionConflict      = errors.New("pending posting journal version conflict")
	ErrPostingApprovalDecisionScopeConflict        = errors.New("pending posting journal scope conflict")
	ErrPostingApprovalDecisionIdempotencyConflict  = errors.New("posting approval decision idempotency conflict")
	ErrPostingApprovalDecisionCommandInProgress    = errors.New("posting approval decision command is already in progress")
	ErrPostingApprovalDecisionAuditUnavailable     = errors.New("posting approval decision audit unavailable")
	ErrPostingApprovalDecisionOutboxUnavailable    = errors.New("posting approval decision outbox unavailable")
	ErrPostingApprovalDecisionValidation           = errors.New("posting approval decision validation failed")
	ErrInvalidPostingApprovalDecisionService       = errors.New("invalid posting approval decision service")
)

// PostingApprovalDecision is the immutable decision evidence supplied by the
// Workflow boundary. GL validates its reference, but does not evaluate the
// approval policy or create a second approval aggregate.
type PostingApprovalDecision struct {
	Outcome   string
	Reference ApprovalDecisionReference
	Reason    string
}

func (decision PostingApprovalDecision) Canonical() PostingApprovalDecision {
	decision.Outcome = strings.ToLower(strings.TrimSpace(decision.Outcome))
	decision.Reason = strings.TrimSpace(decision.Reason)
	decision.Reference.PolicyVersion = strings.TrimSpace(decision.Reference.PolicyVersion)
	decision.Reference.CandidateFingerprint = strings.TrimSpace(decision.Reference.CandidateFingerprint)
	return decision
}

func (decision PostingApprovalDecision) Validate() error {
	decision = decision.Canonical()
	if decision.Outcome != PostingApprovalOutcomeApproved && decision.Outcome != PostingApprovalOutcomeRejected {
		return fmt.Errorf("%w: outcome must be approved or rejected", ErrInvalidPostingApprovalDecisionCommand)
	}
	if err := decision.Reference.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrPostingApprovalDecisionReferenceInvalid, err)
	}
	return nil
}

// ApplyJournalApprovalDecisionCommand is the GL-owned command. Request
// identity and source version are loaded from the pending journal; callers
// cannot replace the submitted financial facts while applying a decision.
type ApplyJournalApprovalDecisionCommand struct {
	CommandID              uuid.UUID
	JournalID              uuid.UUID
	AccountingScopeID      uuid.UUID
	ExpectedJournalVersion int64
	Decision               PostingApprovalDecision
	IdempotencyKey         string
	ApplicationFingerprint string
	CorrelationID          string
	CausationID            string
}

func (command ApplyJournalApprovalDecisionCommand) Canonical() ApplyJournalApprovalDecisionCommand {
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.ApplicationFingerprint = strings.TrimSpace(command.ApplicationFingerprint)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Decision = command.Decision.Canonical()
	return command
}

func (command ApplyJournalApprovalDecisionCommand) Validate() error {
	command = command.Canonical()
	if command.CommandID == uuid.Nil || command.JournalID == uuid.Nil || command.AccountingScopeID == uuid.Nil {
		return fmt.Errorf("%w: command, journal, and accounting-scope identifiers are required", ErrInvalidPostingApprovalDecisionCommand)
	}
	if command.ExpectedJournalVersion < 1 {
		return fmt.Errorf("%w: expected journal version must be positive", ErrInvalidPostingApprovalDecisionCommand)
	}
	if command.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidPostingApprovalDecisionCommand)
	}
	if command.CorrelationID == "" || command.CausationID == "" {
		return fmt.Errorf("%w: correlation and causation references are required", ErrInvalidPostingApprovalDecisionCommand)
	}
	return command.Decision.Validate()
}

// PostingApprovalDecisionEvidence is the verified, immutable Workflow result
// consumed by GL. Implementations may use a local test adapter or a future
// Workflow application boundary.
type PostingApprovalDecisionEvidence struct {
	Decision        PostingApprovalDecision
	SubjectType     string
	SubjectID       uuid.UUID
	SubjectVersion  int64
	ScopeID         uuid.UUID
	PolicyReference string
}

type PostingApprovalDecisionVerifier interface {
	VerifyPostingApprovalDecision(context.Context, Actor, PostingApprovalSubject, PostingApprovalDecision) (PostingApprovalDecisionEvidence, error)
}

type PostingApprovalDecisionAuthorization struct {
	Allowed           bool
	Permission        string
	PolicyReference   string
	PolicyVersion     string
	DecisionReference uuid.UUID
	Reason            string
}

type PostingApprovalDecisionAuthorizer interface {
	AuthorizePostingApprovalDecision(context.Context, Actor, ApplyJournalApprovalDecisionCommand, PostingApprovalSubject) (PostingApprovalDecisionAuthorization, error)
}

// PostingApprovalSubject is the immutable GL snapshot that the decision is
// allowed to affect. Request contains the original lines and posting facts;
// Journal contains the current aggregate identity/version.
type PostingApprovalSubject struct {
	Journal PostingJournal
	Request PostingRequest
}

type PostingApprovalDecisionResult struct {
	Outcome           string
	ApplicationStatus string
	Journal           *PostingJournal
	Approval          ApprovalDecisionReference
	ApprovalRequestID uuid.UUID
	DecisionReference uuid.UUID
	PolicyReference   string
	ValidationOutcome string
	ApprovalStatus    string
	SourceReference   string
	GateEvidence      PostingGateEvidence
	AuditReference    uuid.UUID
	NextAction        string
	Issues            []PostingIssue
	Replayed          bool
}

func (result PostingApprovalDecisionResult) AggregateID() uuid.UUID {
	if result.Journal == nil {
		return uuid.Nil
	}
	return result.Journal.ID
}

func (result PostingApprovalDecisionResult) AggregateVersion() int64 {
	if result.Journal == nil {
		return 0
	}
	return result.Journal.Version
}

func (result PostingApprovalDecisionResult) LifecycleStatus() string {
	if result.Journal == nil {
		return "Rejected"
	}
	return result.Journal.Status
}

type PostingApprovalDecisionCommit struct {
	Command       ApplyJournalApprovalDecisionCommand
	Actor         Actor
	Subject       PostingApprovalSubject
	Decision      PostingApprovalDecisionEvidence
	Authorization PostingApprovalDecisionAuthorization
	GateEvidence  PostingGateEvidence
	Now           time.Time
}

type PostingApprovalDecisionRepository interface {
	LoadPendingPosting(context.Context, uuid.UUID, uuid.UUID) (PostingApprovalSubject, error)
	ResolvePostingApprovalApplication(context.Context, ApplyJournalApprovalDecisionCommand) (PostingApprovalDecisionResult, bool, error)
	CommitPostingApprovalDecision(context.Context, PostingApprovalDecisionCommit) (PostingApprovalDecisionResult, error)
}

type PostingApprovalDecisionService struct {
	repository  PostingApprovalDecisionRepository
	verifier    PostingApprovalDecisionVerifier
	authorizer  PostingApprovalDecisionAuthorizer
	postingAuth PostingAuthorizer
	references  PostingReferenceValidator
	gate        PostingGateValidator
	policy      SoftClosePostingPolicyEvaluator
	currency    money.CurrencyRegistry
	clock       func() time.Time
}

func NewPostingApprovalDecisionService(
	repository PostingApprovalDecisionRepository,
	verifier PostingApprovalDecisionVerifier,
	authorizer PostingApprovalDecisionAuthorizer,
	postingAuth PostingAuthorizer,
	references PostingReferenceValidator,
	gate PostingGateValidator,
	currency money.CurrencyRegistry,
	clock func() time.Time,
	policy ...SoftClosePostingPolicyEvaluator,
) (*PostingApprovalDecisionService, error) {
	if repository == nil || verifier == nil || authorizer == nil || postingAuth == nil || references == nil || gate == nil || clock == nil {
		return nil, ErrInvalidPostingApprovalDecisionService
	}
	var softClosePolicy SoftClosePostingPolicyEvaluator
	if len(policy) > 0 {
		softClosePolicy = policy[0]
	}
	return &PostingApprovalDecisionService{repository: repository, verifier: verifier, authorizer: authorizer, postingAuth: postingAuth, references: references, gate: gate, policy: softClosePolicy, currency: currency, clock: clock}, nil
}

func (service *PostingApprovalDecisionService) Execute(ctx context.Context, actor Actor, command ApplyJournalApprovalDecisionCommand) (PostingApprovalDecisionResult, error) {
	if service == nil {
		return PostingApprovalDecisionResult{}, ErrInvalidPostingApprovalDecisionService
	}
	if err := actor.Validate(); err != nil {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	fingerprint, err := postingApprovalDecisionFingerprint(command)
	if err != nil {
		return PostingApprovalDecisionResult{}, fmt.Errorf("%w: fingerprint: %v", ErrInvalidPostingApprovalDecisionCommand, err)
	}
	command.ApplicationFingerprint = fingerprint

	if result, found, err := service.repository.ResolvePostingApprovalApplication(ctx, command); err != nil {
		return PostingApprovalDecisionResult{}, err
	} else if found {
		result.Replayed = true
		return result, nil
	}

	subject, err := service.repository.LoadPendingPosting(ctx, command.JournalID, command.AccountingScopeID)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if subject.Journal.Version != command.ExpectedJournalVersion {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionVersionConflict
	}
	if subject.Journal.Status != PostingStatusPendingApproval || subject.Journal.Approval == nil || !subject.Journal.Approval.Required {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionAlreadyApplied
	}
	if subject.Request.AccountingScopeID != command.AccountingScopeID {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionScopeConflict
	}

	decision, err := service.verifier.VerifyPostingApprovalDecision(ctx, actor, subject, command.Decision)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	decision.Decision = decision.Decision.Canonical()
	if decision.Decision.Outcome != command.Decision.Outcome || decision.Decision.Reference != command.Decision.Reference {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionReferenceInvalid
	}
	approval := subject.Journal.Approval
	if approval == nil || decision.SubjectType != PostingApprovalSubjectType || decision.SubjectID != subject.Journal.ID || decision.SubjectVersion != subject.Journal.Version || decision.ScopeID != command.AccountingScopeID || strings.TrimSpace(decision.PolicyReference) == "" || decision.PolicyReference != approval.PolicyReference {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionReferenceInvalid
	}

	authorization, err := service.authorizer.AuthorizePostingApprovalDecision(ctx, actor, command, subject)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if !authorization.Allowed || authorization.Permission != PostingApprovalPermission || authorization.DecisionReference == uuid.Nil {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionAuthorizationDenied
	}

	request := subject.Request.Canonical()
	request.RequestFingerprint = subject.Journal.RequestFingerprint
	if err := request.Validate(service.currency); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if err := service.references.ValidatePostingReferences(ctx, actor, request); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	currentAuthorization, err := service.postingAuth.AuthorizePosting(ctx, actor, request)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if !currentAuthorization.Allowed {
		return PostingApprovalDecisionResult{}, ErrPostingAuthorizationDenied
	}
	gateEvidence, err := service.gate.ValidatePostingGate(ctx, request)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if err := validatePostingApprovalGateEvidenceForActor(request, gateEvidence, command.Decision.Outcome, actor); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if command.Decision.Outcome == PostingApprovalOutcomeApproved {
		if err := evaluateSoftClosePostingPolicy(ctx, actor, request, gateEvidence, service.policy); err != nil {
			return PostingApprovalDecisionResult{}, err
		}
	}

	return service.repository.CommitPostingApprovalDecision(ctx, PostingApprovalDecisionCommit{
		Command: command, Actor: actor, Subject: subject, Decision: decision,
		Authorization: authorization, GateEvidence: gateEvidence, Now: service.clock().UTC(),
	})
}

func validatePostingApprovalGateEvidence(request PostingRequest, evidence PostingGateEvidence, outcome string) error {
	return validatePostingApprovalGateEvidenceForActorAt(request, evidence, outcome, Actor{}, time.Now().UTC())
}

func validatePostingApprovalGateEvidenceAt(request PostingRequest, evidence PostingGateEvidence, outcome string, now time.Time) error {
	return validatePostingApprovalGateEvidenceForActorAt(request, evidence, outcome, Actor{}, now)
}

func validatePostingApprovalGateEvidenceForActor(request PostingRequest, evidence PostingGateEvidence, outcome string, actor Actor) error {
	return validatePostingApprovalGateEvidenceForActorAt(request, evidence, outcome, actor, time.Now().UTC())
}

func validatePostingApprovalGateEvidenceForActorAt(request PostingRequest, evidence PostingGateEvidence, outcome string, actor Actor, now time.Time) error {
	if evidence.FiscalPeriodID != request.FiscalPeriodID {
		return fmt.Errorf("%w: fiscal period does not match", ErrPostingPeriodConflict)
	}
	if evidence.PeriodStateVersion != request.PeriodStateVersion {
		return fmt.Errorf("%w: expected %d, current %d", ErrPostingPeriodConflict, request.PeriodStateVersion, evidence.PeriodStateVersion)
	}
	if evidence.PostingGateVersion != request.PostingGateVersion {
		return fmt.Errorf("%w: expected %d, current %d", ErrPostingGateConflict, request.PostingGateVersion, evidence.PostingGateVersion)
	}
	if outcome == PostingApprovalOutcomeApproved {
		return validatePostingGateEvidenceForActorAt(request, evidence, actor, now)
	}
	return nil
}

func postingApprovalDecisionFingerprint(command ApplyJournalApprovalDecisionCommand) (string, error) {
	command = command.Canonical()
	command.CommandID = uuid.Nil
	command.IdempotencyKey = ""
	command.ApplicationFingerprint = ""
	command.CorrelationID = ""
	command.CausationID = ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	fingerprint, err := idempotency.ComputeFingerprint(data)
	return string(fingerprint), err
}

// MemoryPostingApprovalDecisionVerifier is an explicit in-process adapter for
// tests and the development runtime. A production Workflow adapter must verify
// the same immutable subject, policy, and decision evidence at this boundary.
type MemoryPostingApprovalDecisionVerifier struct {
	Err error
}

func (verifier MemoryPostingApprovalDecisionVerifier) VerifyPostingApprovalDecision(_ context.Context, _ Actor, subject PostingApprovalSubject, decision PostingApprovalDecision) (PostingApprovalDecisionEvidence, error) {
	if verifier.Err != nil {
		return PostingApprovalDecisionEvidence{}, verifier.Err
	}
	decision = decision.Canonical()
	if err := decision.Validate(); err != nil {
		return PostingApprovalDecisionEvidence{}, err
	}
	approval := subject.Journal.Approval
	if approval == nil || approval.ApprovalRequestID != decision.Reference.ApprovalRequestID || approval.PolicyVersion != decision.Reference.PolicyVersion || subject.Journal.RequestFingerprint != decision.Reference.CandidateFingerprint || subject.Journal.Version != decision.Reference.SubjectVersion {
		return PostingApprovalDecisionEvidence{}, ErrPostingApprovalDecisionReferenceInvalid
	}
	return PostingApprovalDecisionEvidence{Decision: decision, SubjectType: "journal-entry", SubjectID: subject.Journal.ID, SubjectVersion: subject.Journal.Version, ScopeID: subject.Request.AccountingScopeID, PolicyReference: approval.PolicyReference}, nil
}

type MemoryPostingApprovalDecisionAuthorizer struct {
	Decision PostingApprovalDecisionAuthorization
	Err      error
}

func (authorizer MemoryPostingApprovalDecisionAuthorizer) AuthorizePostingApprovalDecision(context.Context, Actor, ApplyJournalApprovalDecisionCommand, PostingApprovalSubject) (PostingApprovalDecisionAuthorization, error) {
	if authorizer.Err != nil {
		return PostingApprovalDecisionAuthorization{}, authorizer.Err
	}
	decision := authorizer.Decision
	if decision.Permission == "" {
		decision.Permission = PostingApprovalPermission
	}
	if decision.DecisionReference == uuid.Nil {
		decision.DecisionReference = uuid.New()
	}
	return decision, nil
}

var _ PostingApprovalDecisionRepository = (*MemoryPostingRepository)(nil)
var _ PostingApprovalDecisionVerifier = MemoryPostingApprovalDecisionVerifier{}
var _ PostingApprovalDecisionAuthorizer = MemoryPostingApprovalDecisionAuthorizer{}
