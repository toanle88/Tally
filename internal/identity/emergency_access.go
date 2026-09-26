package identity

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

const (
	EmergencyAccessActionGrant  = "grant"
	EmergencyAccessActionRevoke = "revoke"

	EmergencyAccessGrantStatusActive  = "active"
	EmergencyAccessGrantStatusRevoked = "revoked"
	EmergencyAccessGrantStatusExpired = "expired"

	EmergencyAccessReviewPending   = "pending"
	EmergencyAccessReviewCompleted = "completed"
	EmergencyAccessReviewOverdue   = "overdue"

	EmergencyAccessGrantPermission  = "finance.iam.grant.emergency.access"
	EmergencyAccessRevokePermission = "finance.iam.revoke.emergency.access"

	EmergencyAccessDefaultMaxDuration = 4 * time.Hour
	EmergencyAccessAssuranceMaxAge    = 5 * time.Minute
)

var (
	ErrInvalidEmergencyAccessCommand       = errors.New("invalid emergency access command")
	ErrInvalidEmergencyAccessGrant         = errors.New("invalid emergency access grant")
	ErrEmergencyAccessNotFound             = errors.New("emergency access grant not found")
	ErrEmergencyAccessAlreadyRevoked       = errors.New("emergency access grant is already revoked")
	ErrEmergencyAccessReviewCompleted      = errors.New("emergency access review is already completed")
	ErrEmergencyAccessReviewInvalid        = errors.New("invalid emergency access review transition")
	ErrEmergencyAccessAuthorizationDenied  = errors.New("emergency access authorization denied")
	ErrEmergencyAccessAuthorizationStale   = errors.New("emergency access authorization is stale")
	ErrEmergencyAccessAuthorizationExpired = errors.New("emergency access authorization has expired")
	ErrEmergencyAccessAuthorizationUnavail = errors.New("emergency access authorization is unavailable")
	ErrEmergencyAccessApprovalUnavailable  = errors.New("emergency access approval is unavailable")
	ErrEmergencyAccessApprovalRejected     = errors.New("emergency access approval was rejected")
	ErrEmergencyAccessStepUpRequired       = errors.New("emergency access authentication assurance requires step-up")
	ErrEmergencyAccessVersionConflict      = errors.New("emergency access grant version conflict")
	ErrEmergencyAccessIdempotencyConflict  = errors.New("emergency access command idempotency conflict")
	ErrEmergencyAccessCommandInProgress    = errors.New("emergency access command is already in progress")
	ErrEmergencyAccessDurableCommandFailed = errors.New("emergency access command previously failed")
	ErrEmergencyAccessAuditUnavailable     = errors.New("emergency access audit recorder unavailable")
	ErrInvalidEmergencyAccessService       = errors.New("invalid emergency access service")
	ErrEmergencyAccessPolicyUnavailable    = errors.New("emergency access policy unavailable")
	ErrEmergencyAccessCalendarUnavailable  = errors.New("emergency access business calendar unavailable")
)

type EmergencyAccessGrant struct {
	ID                uuid.UUID
	TargetActorID     uuid.UUID
	Status            string
	Permissions       []string
	ScopeIDs          []string
	ReasonCode        string
	GrantingActorID   uuid.UUID
	Approval          ApprovalDecisionReference
	PolicyVersion     string
	StartAt           time.Time
	ExpiresAt         time.Time
	ReviewStatus      string
	ReviewOutcomeCode string
	ReviewReference   string
	ReviewDueAt       time.Time
	RevokedAt         *time.Time
	RevokedBy         *uuid.UUID
	RevocationReason  string
	Version           aggregateversion.AggregateVersion
	AuditReference    uuid.UUID
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func (grant EmergencyAccessGrant) Validate() error {
	if grant.ID == uuid.Nil || grant.TargetActorID == uuid.Nil || grant.GrantingActorID == uuid.Nil {
		return fmt.Errorf("%w: grant and actor identities are required", ErrInvalidEmergencyAccessGrant)
	}
	if grant.Status != EmergencyAccessGrantStatusActive && grant.Status != EmergencyAccessGrantStatusRevoked {
		return fmt.Errorf("%w: unsupported status", ErrInvalidEmergencyAccessGrant)
	}
	if grant.Version.Value() < 1 {
		return fmt.Errorf("%w: version is required", ErrInvalidEmergencyAccessGrant)
	}
	if len(grant.Permissions) == 0 || len(grant.ScopeIDs) == 0 {
		return fmt.Errorf("%w: permissions and scopes are required", ErrInvalidEmergencyAccessGrant)
	}
	if _, err := normalizeEmergencyAccessValues(grant.Permissions, true); err != nil {
		return err
	}
	if _, err := normalizeEmergencyAccessValues(grant.ScopeIDs, false); err != nil {
		return err
	}
	if strings.TrimSpace(grant.ReasonCode) == "" || grant.StartAt.IsZero() || grant.ExpiresAt.IsZero() || !grant.ExpiresAt.After(grant.StartAt) {
		return fmt.Errorf("%w: reason and valid time interval are required", ErrInvalidEmergencyAccessGrant)
	}
	if grant.ExpiresAt.Sub(grant.StartAt) > EmergencyAccessDefaultMaxDuration {
		return fmt.Errorf("%w: duration exceeds four-hour default maximum", ErrInvalidEmergencyAccessGrant)
	}
	if err := grant.Approval.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(grant.PolicyVersion) == "" {
		return fmt.Errorf("%w: policy version is required", ErrInvalidEmergencyAccessGrant)
	}
	if grant.ReviewStatus != EmergencyAccessReviewPending && grant.ReviewStatus != EmergencyAccessReviewCompleted {
		return fmt.Errorf("%w: unsupported review status", ErrInvalidEmergencyAccessGrant)
	}
	if grant.ReviewStatus == EmergencyAccessReviewCompleted && strings.TrimSpace(grant.ReviewOutcomeCode) == "" {
		return fmt.Errorf("%w: completed review requires an outcome code", ErrInvalidEmergencyAccessGrant)
	}
	if grant.ReviewDueAt.IsZero() {
		return fmt.Errorf("%w: review deadline is required", ErrInvalidEmergencyAccessGrant)
	}
	if grant.Status == EmergencyAccessGrantStatusRevoked {
		if grant.RevokedAt == nil || grant.RevokedBy == nil || *grant.RevokedBy == uuid.Nil || strings.TrimSpace(grant.RevocationReason) == "" {
			return fmt.Errorf("%w: revoked grant requires revocation evidence", ErrInvalidEmergencyAccessGrant)
		}
	}
	return nil
}

func (grant EmergencyAccessGrant) EffectiveStatus(now time.Time) string {
	if grant.Status == EmergencyAccessGrantStatusRevoked {
		return EmergencyAccessGrantStatusRevoked
	}
	if !now.UTC().Before(grant.ExpiresAt.UTC()) {
		return EmergencyAccessGrantStatusExpired
	}
	return EmergencyAccessGrantStatusActive
}

func (grant EmergencyAccessGrant) EffectiveReviewStatus(now time.Time) string {
	if grant.ReviewStatus == EmergencyAccessReviewPending && !now.UTC().Before(grant.ReviewDueAt.UTC()) {
		return EmergencyAccessReviewOverdue
	}
	return grant.ReviewStatus
}

func (grant EmergencyAccessGrant) Allows(now time.Time, actorID uuid.UUID, permission, scopeID string) bool {
	if grant.EffectiveStatus(now) != EmergencyAccessGrantStatusActive || actorID != grant.TargetActorID || now.UTC().Before(grant.StartAt.UTC()) {
		return false
	}
	for _, value := range grant.Permissions {
		if value == strings.TrimSpace(permission) {
			for _, scope := range grant.ScopeIDs {
				if scope == strings.TrimSpace(scopeID) || scope == "*" {
					return true
				}
			}
		}
	}
	return false
}

type EmergencyAccessGrantCommand struct {
	Action            string
	GrantID           uuid.UUID
	TargetActorID     uuid.UUID
	Permissions       []string
	ScopeIDs          []string
	ReasonCode        string
	StartsAt          time.Time
	ExpiresAt         time.Time
	Approval          ApprovalDecisionReference
	ReviewStatus      string
	ReviewOutcomeCode string
	ReviewReference   string
	ExpectedVersion   *aggregateversion.AggregateVersion
	IdempotencyKey    string
	CorrelationID     string
	CausationID       string
}

func (command EmergencyAccessGrantCommand) Validate() error {
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidEmergencyAccessCommand)
	}
	switch command.Action {
	case EmergencyAccessActionGrant:
		if command.GrantID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: grant must not provide grant id or expected version", ErrInvalidEmergencyAccessCommand)
		}
		if command.TargetActorID == uuid.Nil || strings.TrimSpace(command.ReasonCode) == "" {
			return fmt.Errorf("%w: target actor and reason code are required", ErrInvalidEmergencyAccessCommand)
		}
		if command.StartsAt.IsZero() || command.ExpiresAt.IsZero() || !command.ExpiresAt.After(command.StartsAt) {
			return fmt.Errorf("%w: valid start and expiry are required", ErrInvalidEmergencyAccessCommand)
		}
		if command.ExpiresAt.Sub(command.StartsAt) > EmergencyAccessDefaultMaxDuration {
			return fmt.Errorf("%w: duration exceeds four-hour default maximum", ErrInvalidEmergencyAccessCommand)
		}
		if _, err := normalizeEmergencyAccessValues(command.Permissions, true); err != nil {
			return err
		}
		if _, err := normalizeEmergencyAccessValues(command.ScopeIDs, false); err != nil {
			return err
		}
		if err := command.Approval.Validate(); err != nil {
			return err
		}
	case EmergencyAccessActionRevoke:
		if command.GrantID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: grant id and expected version are required", ErrInvalidEmergencyAccessCommand)
		}
		if strings.TrimSpace(command.ReasonCode) == "" {
			return fmt.Errorf("%w: revocation reason is required", ErrInvalidEmergencyAccessCommand)
		}
		if command.ReviewStatus == "" {
			command.ReviewStatus = EmergencyAccessReviewPending
		}
		if command.ReviewStatus != EmergencyAccessReviewPending && command.ReviewStatus != EmergencyAccessReviewCompleted {
			return fmt.Errorf("%w: review status must be pending or completed", ErrInvalidEmergencyAccessCommand)
		}
		if command.ReviewStatus == EmergencyAccessReviewCompleted && (strings.TrimSpace(command.ReviewOutcomeCode) == "" || strings.TrimSpace(command.ReviewReference) == "") {
			return fmt.Errorf("%w: completed review requires outcome and reference", ErrInvalidEmergencyAccessCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidEmergencyAccessCommand, command.Action)
	}
	return nil
}

func normalizeEmergencyAccessValues(values []string, permissions bool) ([]string, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("%w: at least one value is required", ErrInvalidEmergencyAccessGrant)
	}
	result := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return nil, fmt.Errorf("%w: blank value is not allowed", ErrInvalidEmergencyAccessGrant)
		}
		if permissions && !isKnownPermission(value) {
			return nil, fmt.Errorf("%w: permission %q is not in the approved catalogue", ErrInvalidEmergencyAccessGrant, value)
		}
		if _, exists := seen[value]; exists {
			return nil, fmt.Errorf("%w: duplicate value %q", ErrInvalidEmergencyAccessGrant, value)
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Strings(result)
	return result, nil
}

type EmergencyAccessAuthorizer interface {
	AuthorizeEmergencyAccess(context.Context, ApplicationActor, EmergencyAccessGrantCommand, *EmergencyAccessGrant) (AuthorizationDecision, error)
}

type EmergencyAccessApprovalPort interface {
	ValidateEmergencyAccessApproval(context.Context, ApplicationActor, EmergencyAccessGrantCommand, *EmergencyAccessGrant, string) error
}

type AllowAllEmergencyAccessApprovalPort struct{}

func (AllowAllEmergencyAccessApprovalPort) ValidateEmergencyAccessApproval(_ context.Context, actor ApplicationActor, command EmergencyAccessGrantCommand, current *EmergencyAccessGrant, fingerprint string) error {
	expectedVersion := int64(1)
	if current != nil {
		expectedVersion = current.Version.Value() + 1
	}
	if command.Approval.SubjectVersion != expectedVersion || command.Approval.CandidateFingerprint != fingerprint {
		return ErrEmergencyAccessApprovalRejected
	}
	if current == nil && strings.TrimSpace(command.Approval.PolicyVersion) == "" {
		return ErrEmergencyAccessApprovalRejected
	}
	if command.Approval.ApproverUserID == actor.UserID {
		return fmt.Errorf("%w: requesting actor cannot approve the same grant", ErrEmergencyAccessApprovalRejected)
	}
	return nil
}

type EmergencyAccessCalendar interface {
	AddBusinessDays(time.Time, int) time.Time
}

type WeekdayEmergencyAccessCalendar struct{}

func (WeekdayEmergencyAccessCalendar) AddBusinessDays(value time.Time, days int) time.Time {
	value = value.UTC()
	step := 1
	if days < 0 {
		step = -1
	}
	for days != 0 {
		value = value.AddDate(0, 0, step)
		if value.Weekday() != time.Saturday && value.Weekday() != time.Sunday {
			days -= step
		}
	}
	return value
}

type EmergencyAccessAuditRecord struct {
	GrantID                uuid.UUID
	TargetActorID          uuid.UUID
	ActorUserID            uuid.UUID
	ActorAuthenticationRef string
	Action                 string
	ScopeIDs               []string
	Permission             string
	PolicyReference        string
	PolicyVersion          string
	DecisionReference      uuid.UUID
	ApprovalRequestID      uuid.UUID
	ApprovalDecisionID     uuid.UUID
	ApproverUserID         uuid.UUID
	GrantVersion           int64
	ReviewStatus           string
	BeforeFingerprint      string
	AfterFingerprint       string
	CorrelationID          string
	CausationID            string
}

type EmergencyAccessAuditRecorder interface {
	RecordEmergencyAccessMutation(context.Context, EmergencyAccessAuditRecord) error
}

type EmergencyAccessRepository interface {
	Get(context.Context, uuid.UUID) (EmergencyAccessGrant, error)
	List(context.Context) ([]EmergencyAccessGrant, error)
}

type EmergencyAccessMutation struct {
	Before          EmergencyAccessGrant
	After           EmergencyAccessGrant
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           EmergencyAccessAuditRecord
}

type AtomicEmergencyAccessMutationRepository interface {
	CommitEmergencyAccessMutation(context.Context, EmergencyAccessMutation) error
}

type DurableEmergencyAccessServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableEmergencyAccessMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableEmergencyAccessMutationRepository interface {
	CommitEmergencyAccessMutationWithIdempotency(context.Context, EmergencyAccessMutation, DurableEmergencyAccessMutationCommit) error
}

type EmergencyAccessCommandResult struct {
	Grant             EmergencyAccessGrant
	DecisionReference uuid.UUID
	PolicyReference   string
	Approval          ApprovalDecisionReference
	Replayed          bool
}

type EmergencyAccessService struct {
	repository EmergencyAccessRepository
	authorizer EmergencyAccessAuthorizer
	approval   EmergencyAccessApprovalPort
	audit      EmergencyAccessAuditRecorder
	calendar   EmergencyAccessCalendar
	clock      func() time.Time
	durable    *DurableEmergencyAccessServiceConfig

	idempotencyMu sync.Mutex
	idempotency   map[string]storedEmergencyAccessCommand
}

type storedEmergencyAccessCommand struct {
	fingerprint string
	result      EmergencyAccessCommandResult
}

func NewEmergencyAccessService(repository EmergencyAccessRepository, authorizer EmergencyAccessAuthorizer, approval EmergencyAccessApprovalPort, audit EmergencyAccessAuditRecorder, calendar EmergencyAccessCalendar, clock func() time.Time) (*EmergencyAccessService, error) {
	return newEmergencyAccessService(repository, authorizer, approval, audit, calendar, clock, nil)
}

func NewEmergencyAccessServiceWithDurableIdempotency(repository EmergencyAccessRepository, authorizer EmergencyAccessAuthorizer, approval EmergencyAccessApprovalPort, audit EmergencyAccessAuditRecorder, calendar EmergencyAccessCalendar, clock func() time.Time, durable DurableEmergencyAccessServiceConfig) (*EmergencyAccessService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidEmergencyAccessService
	}
	return newEmergencyAccessService(repository, authorizer, approval, audit, calendar, clock, &durable)
}

func newEmergencyAccessService(repository EmergencyAccessRepository, authorizer EmergencyAccessAuthorizer, approval EmergencyAccessApprovalPort, audit EmergencyAccessAuditRecorder, calendar EmergencyAccessCalendar, clock func() time.Time, durable *DurableEmergencyAccessServiceConfig) (*EmergencyAccessService, error) {
	if repository == nil || authorizer == nil || approval == nil || audit == nil || calendar == nil || clock == nil {
		return nil, ErrInvalidEmergencyAccessService
	}
	service := &EmergencyAccessService{repository: repository, authorizer: authorizer, approval: approval, audit: audit, calendar: calendar, clock: clock, durable: durable, idempotency: make(map[string]storedEmergencyAccessCommand)}
	if binder, ok := repository.(interface {
		BindEmergencyAccessAuditRecorder(EmergencyAccessAuditRecorder)
	}); ok {
		binder.BindEmergencyAccessAuditRecorder(audit)
	}
	return service, nil
}

func (service *EmergencyAccessService) Execute(ctx context.Context, actor ApplicationActor, command EmergencyAccessGrantCommand) (EmergencyAccessCommandResult, error) {
	if service == nil {
		return EmergencyAccessCommandResult{}, ErrInvalidEmergencyAccessService
	}
	if err := actor.Validate(); err != nil {
		return EmergencyAccessCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return EmergencyAccessCommandResult{}, err
	}
	now := service.clock().UTC()
	if !actor.Subject.Assurance.Satisfies(now, EmergencyAccessAssuranceMaxAge) {
		return EmergencyAccessCommandResult{}, ErrEmergencyAccessStepUpRequired
	}
	fingerprint, err := fingerprintEmergencyAccessCommand(command)
	if err != nil {
		return EmergencyAccessCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scopeKey, marshalErr := json.Marshal(struct {
			Module  string    `json:"module"`
			ActorID uuid.UUID `json:"actorId"`
		}{Module: "identity.emergency-access", ActorID: actor.UserID})
		if marshalErr != nil {
			return EmergencyAccessCommandResult{}, marshalErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scopeKey), command.IdempotencyKey)
		if err != nil {
			return EmergencyAccessCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return EmergencyAccessCommandResult{}, ErrEmergencyAccessIdempotencyConflict
		}
		if err != nil {
			return EmergencyAccessCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			switch acquisition.Result().State() {
			case platformidempotency.StateFailed:
				return EmergencyAccessCommandResult{}, emergencyAccessDurableFailureError(acquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return EmergencyAccessCommandResult{}, ErrEmergencyAccessCommandInProgress
			case platformidempotency.StateEstablished:
				result, decodeErr := decodeEmergencyAccessCommandResult(acquisition.Result().ResultBody())
				if decodeErr != nil {
					return EmergencyAccessCommandResult{}, decodeErr
				}
				result.Replayed = true
				return result, nil
			}
		}
	} else {
		service.idempotencyMu.Lock()
		defer service.idempotencyMu.Unlock()
		if previous, exists := service.idempotency[key]; exists {
			if previous.fingerprint != fingerprint {
				return EmergencyAccessCommandResult{}, ErrEmergencyAccessIdempotencyConflict
			}
			result := cloneEmergencyAccessCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *EmergencyAccessGrant
	if command.Action == EmergencyAccessActionRevoke {
		grant, getErr := service.repository.Get(ctx, command.GrantID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, getErr)
			return EmergencyAccessCommandResult{}, getErr
		}
		if !grant.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrEmergencyAccessVersionConflict)
			return EmergencyAccessCommandResult{}, ErrEmergencyAccessVersionConflict
		}
		current = &grant
	}
	decision, err := service.authorizer.AuthorizeEmergencyAccess(ctx, actor, command, cloneEmergencyAccessGrantPtr(current))
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return EmergencyAccessCommandResult{}, err
	}
	if err := authorizeEmergencyAccessDecision(command, decision); err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return EmergencyAccessCommandResult{}, err
	}

	var before EmergencyAccessGrant
	var after EmergencyAccessGrant
	if command.Action == EmergencyAccessActionGrant {
		policyVersion := decision.PolicyVersion
		if strings.TrimSpace(policyVersion) == "" {
			policyVersion = command.Approval.PolicyVersion
		}
		permissions, normalizeErr := normalizeEmergencyAccessValues(command.Permissions, true)
		if normalizeErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, normalizeErr)
			return EmergencyAccessCommandResult{}, normalizeErr
		}
		scopes, normalizeErr := normalizeEmergencyAccessValues(command.ScopeIDs, false)
		if normalizeErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, normalizeErr)
			return EmergencyAccessCommandResult{}, normalizeErr
		}
		dueAt := service.calendar.AddBusinessDays(now, 1).UTC()
		after = EmergencyAccessGrant{
			ID: uuid.New(), TargetActorID: command.TargetActorID, Status: EmergencyAccessGrantStatusActive,
			Permissions: permissions, ScopeIDs: scopes,
			ReasonCode: strings.TrimSpace(command.ReasonCode), GrantingActorID: actor.UserID, Approval: command.Approval,
			PolicyVersion: policyVersion, StartAt: command.StartsAt.UTC(), ExpiresAt: command.ExpiresAt.UTC(),
			ReviewStatus: EmergencyAccessReviewPending, ReviewDueAt: dueAt, Version: aggregateversion.Initial(),
			CreatedAt: now, UpdatedAt: now,
		}
		if err := after.Validate(); err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return EmergencyAccessCommandResult{}, err
		}
		if strings.TrimSpace(command.Approval.PolicyVersion) != strings.TrimSpace(after.PolicyVersion) {
			service.finalizeDurableFailure(ctx, acquisition, ErrEmergencyAccessApprovalRejected)
			return EmergencyAccessCommandResult{}, ErrEmergencyAccessApprovalRejected
		}
		candidateFingerprint := candidateEmergencyAccessFingerprint(command, after)
		if err := service.approval.ValidateEmergencyAccessApproval(ctx, actor, command, nil, candidateFingerprint); err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return EmergencyAccessCommandResult{}, err
		}
	} else {
		before = cloneEmergencyAccessGrant(*current)
		after = cloneEmergencyAccessGrant(*current)
		if after.Status == EmergencyAccessGrantStatusRevoked {
			if after.ReviewStatus == EmergencyAccessReviewCompleted {
				service.finalizeDurableFailure(ctx, acquisition, ErrEmergencyAccessReviewCompleted)
				return EmergencyAccessCommandResult{}, ErrEmergencyAccessReviewCompleted
			}
			if command.ReviewStatus != EmergencyAccessReviewCompleted {
				service.finalizeDurableFailure(ctx, acquisition, ErrEmergencyAccessAlreadyRevoked)
				return EmergencyAccessCommandResult{}, ErrEmergencyAccessAlreadyRevoked
			}
			after.applyReview(command)
		} else {
			if command.ReviewStatus == EmergencyAccessReviewCompleted {
				service.finalizeDurableFailure(ctx, acquisition, ErrEmergencyAccessReviewInvalid)
				return EmergencyAccessCommandResult{}, ErrEmergencyAccessReviewInvalid
			}
			after.Status = EmergencyAccessGrantStatusRevoked
			revokedAt := now
			revokedBy := actor.UserID
			after.RevokedAt = &revokedAt
			after.RevokedBy = &revokedBy
			after.RevocationReason = strings.TrimSpace(command.ReasonCode)
			after.applyReview(command)
		}
		nextVersion, advanceErr := after.Version.Advance()
		if advanceErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, advanceErr)
			return EmergencyAccessCommandResult{}, advanceErr
		}
		after.Version = nextVersion
		if err := after.Validate(); err != nil {
			service.finalizeDurableFailure(ctx, acquisition, err)
			return EmergencyAccessCommandResult{}, err
		}
	}

	if command.Action == EmergencyAccessActionGrant {
		after.Version = aggregateversion.Initial()
	}
	candidateFingerprint := candidateEmergencyAccessFingerprint(command, after)
	audit := EmergencyAccessAuditRecord{
		GrantID: after.ID, TargetActorID: after.TargetActorID, ActorUserID: actor.UserID,
		ActorAuthenticationRef: fingerprintSubject(actor.Subject), Action: command.Action,
		ScopeIDs: append([]string(nil), after.ScopeIDs...), Permission: decision.Permission,
		PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion,
		DecisionReference: decision.DecisionReference, ApprovalRequestID: after.Approval.ApprovalRequestID,
		ApprovalDecisionID: after.Approval.DecisionID, ApproverUserID: after.Approval.ApproverUserID,
		GrantVersion: after.Version.Value(), ReviewStatus: after.EffectiveReviewStatus(now),
		BeforeFingerprint: fingerprintEmergencyAccessGrant(before), AfterFingerprint: candidateFingerprint,
		CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	result := EmergencyAccessCommandResult{Grant: cloneEmergencyAccessGrant(after), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, Approval: after.Approval}
	mutation := EmergencyAccessMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: audit}
	if service.durable != nil {
		committer, ok := service.repository.(DurableEmergencyAccessMutationRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, acquisition, ErrInvalidEmergencyAccessService)
			return EmergencyAccessCommandResult{}, ErrInvalidEmergencyAccessService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, marshalErr)
			return EmergencyAccessCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, acquisition, metadataErr)
			return EmergencyAccessCommandResult{}, metadataErr
		}
		err = committer.CommitEmergencyAccessMutationWithIdempotency(ctx, mutation, DurableEmergencyAccessMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else if committer, ok := service.repository.(AtomicEmergencyAccessMutationRepository); ok {
		err = committer.CommitEmergencyAccessMutation(ctx, mutation)
	} else {
		err = ErrInvalidEmergencyAccessService
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, acquisition, err)
		return EmergencyAccessCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedEmergencyAccessCommand{fingerprint: fingerprint, result: cloneEmergencyAccessCommandResult(result)}
	}
	return result, nil
}

func (grant *EmergencyAccessGrant) applyReview(command EmergencyAccessGrantCommand) {
	status := command.ReviewStatus
	if status == "" {
		status = EmergencyAccessReviewPending
	}
	grant.ReviewStatus = status
	grant.ReviewOutcomeCode = strings.TrimSpace(command.ReviewOutcomeCode)
	grant.ReviewReference = strings.TrimSpace(command.ReviewReference)
}

func authorizeEmergencyAccessDecision(command EmergencyAccessGrantCommand, decision AuthorizationDecision) error {
	switch decision.Outcome {
	case AuthorizationExpired:
		return ErrEmergencyAccessAuthorizationExpired
	case AuthorizationUnavailable:
		return ErrEmergencyAccessAuthorizationUnavail
	case AuthorizationStale:
		return ErrEmergencyAccessAuthorizationStale
	}
	wanted := EmergencyAccessRevokePermission
	if command.Action == EmergencyAccessActionGrant {
		wanted = EmergencyAccessGrantPermission
	}
	if !decision.Allowed || decision.Permission != wanted || decision.DecisionReference == uuid.Nil {
		return ErrEmergencyAccessAuthorizationDenied
	}
	return nil
}

type EmergencyAccessEvaluator struct {
	repository EmergencyAccessRepository
	clock      func() time.Time
}

type EmergencyAccessDecision struct {
	Allowed           bool
	GrantReference    uuid.UUID
	TargetActorID     uuid.UUID
	Permission        string
	ScopeID           string
	Outcome           AuthorizationOutcome
	ReasonCode        string
	DecisionReference uuid.UUID
}

func NewEmergencyAccessEvaluator(repository EmergencyAccessRepository, clock func() time.Time) (*EmergencyAccessEvaluator, error) {
	if repository == nil {
		return nil, ErrEmergencyAccessPolicyUnavailable
	}
	if clock == nil {
		clock = time.Now
	}
	return &EmergencyAccessEvaluator{repository: repository, clock: clock}, nil
}

func (evaluator *EmergencyAccessEvaluator) Evaluate(ctx context.Context, actorID uuid.UUID, permission, scopeID, grantReference string) (EmergencyAccessDecision, error) {
	decision := EmergencyAccessDecision{Allowed: false, TargetActorID: actorID, Permission: strings.TrimSpace(permission), ScopeID: strings.TrimSpace(scopeID), Outcome: AuthorizationDenied, ReasonCode: AuthorizationReasonDefaultDeny, DecisionReference: uuid.New()}
	grantID, err := uuid.Parse(strings.TrimSpace(grantReference))
	if err != nil || grantID == uuid.Nil {
		return decision, nil
	}
	grant, err := evaluator.repository.Get(ctx, grantID)
	if err != nil {
		if errors.Is(err, ErrEmergencyAccessNotFound) {
			return decision, nil
		}
		decision.Outcome = AuthorizationUnavailable
		decision.ReasonCode = AuthorizationReasonPolicyUnavailable
		return decision, err
	}
	decision.GrantReference = grant.ID
	if grant.Allows(evaluator.clock().UTC(), actorID, permission, scopeID) {
		decision.Allowed = true
		decision.Outcome = AuthorizationAllowed
		decision.ReasonCode = AuthorizationReasonAllowed
	}
	return decision, nil
}

type MemoryEmergencyAccessRepository struct {
	mu     sync.RWMutex
	grants map[uuid.UUID]EmergencyAccessGrant
	audit  EmergencyAccessAuditRecorder
}

func NewMemoryEmergencyAccessRepository() *MemoryEmergencyAccessRepository {
	return &MemoryEmergencyAccessRepository{grants: make(map[uuid.UUID]EmergencyAccessGrant)}
}

func (repository *MemoryEmergencyAccessRepository) BindEmergencyAccessAuditRecorder(audit EmergencyAccessAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryEmergencyAccessRepository) CommitEmergencyAccessMutation(ctx context.Context, mutation EmergencyAccessMutation) error {
	if repository == nil {
		return ErrInvalidEmergencyAccessService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.grants[mutation.After.ID]; exists {
			return ErrEmergencyAccessIdempotencyConflict
		}
	} else {
		current, exists := repository.grants[mutation.After.ID]
		if !exists {
			return ErrEmergencyAccessNotFound
		}
		if mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
			return ErrEmergencyAccessVersionConflict
		}
	}
	if repository.audit == nil {
		return ErrEmergencyAccessAuditUnavailable
	}
	if err := repository.audit.RecordEmergencyAccessMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.grants[mutation.After.ID] = cloneEmergencyAccessGrant(mutation.After)
	return nil
}

func (repository *MemoryEmergencyAccessRepository) Get(_ context.Context, id uuid.UUID) (EmergencyAccessGrant, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	grant, ok := repository.grants[id]
	if !ok {
		return EmergencyAccessGrant{}, ErrEmergencyAccessNotFound
	}
	return cloneEmergencyAccessGrant(grant), nil
}

func (repository *MemoryEmergencyAccessRepository) List(_ context.Context) ([]EmergencyAccessGrant, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]EmergencyAccessGrant, 0, len(repository.grants))
	for _, grant := range repository.grants {
		result = append(result, cloneEmergencyAccessGrant(grant))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

type MemoryEmergencyAccessAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryEmergencyAccessAuthorizer) AuthorizeEmergencyAccess(context.Context, ApplicationActor, EmergencyAccessGrantCommand, *EmergencyAccessGrant) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return cloneAuthorizationDecision(authorizer.Decision), nil
}

type MemoryEmergencyAccessAuditRecorder struct {
	mu      sync.Mutex
	Records []EmergencyAccessAuditRecord
	Err     error
}

func (recorder *MemoryEmergencyAccessAuditRecorder) RecordEmergencyAccessMutation(_ context.Context, record EmergencyAccessAuditRecord) error {
	if recorder == nil {
		return ErrEmergencyAccessAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func fingerprintEmergencyAccessCommand(command EmergencyAccessGrantCommand) (string, error) {
	var permissions, scopes []string
	var err error
	if command.Action == EmergencyAccessActionGrant {
		permissions, err = normalizeEmergencyAccessValues(command.Permissions, true)
		if err != nil {
			return "", err
		}
		scopes, err = normalizeEmergencyAccessValues(command.ScopeIDs, false)
		if err != nil {
			return "", err
		}
	}
	payload := struct {
		Action            string
		GrantID           uuid.UUID
		TargetActorID     uuid.UUID
		Permissions       []string
		ScopeIDs          []string
		ReasonCode        string
		StartsAt          time.Time
		ExpiresAt         time.Time
		Approval          ApprovalDecisionReference
		ReviewStatus      string
		ReviewOutcomeCode string
		ReviewReference   string
		ExpectedVersion   int64
	}{Action: command.Action, GrantID: command.GrantID, TargetActorID: command.TargetActorID, Permissions: permissions, ScopeIDs: scopes, ReasonCode: strings.TrimSpace(command.ReasonCode), StartsAt: command.StartsAt.UTC(), ExpiresAt: command.ExpiresAt.UTC(), Approval: command.Approval, ReviewStatus: command.ReviewStatus, ReviewOutcomeCode: command.ReviewOutcomeCode, ReviewReference: command.ReviewReference}
	if command.ExpectedVersion != nil {
		payload.ExpectedVersion = command.ExpectedVersion.Value()
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func candidateEmergencyAccessFingerprint(command EmergencyAccessGrantCommand, grant EmergencyAccessGrant) string {
	grantID := command.GrantID
	if command.Action == EmergencyAccessActionGrant {
		// The approval is created before the aggregate identifier is assigned,
		// matching the existing IAM role approval fingerprint contract.
		grantID = uuid.Nil
	}
	data, _ := json.Marshal(struct {
		Action            string
		GrantID           uuid.UUID
		TargetActorID     uuid.UUID
		Permissions       []string
		ScopeIDs          []string
		ReasonCode        string
		StartAt           time.Time
		ExpiresAt         time.Time
		Version           int64
		PolicyVersion     string
		ReviewStatus      string
		ReviewOutcomeCode string
	}{command.Action, grantID, grant.TargetActorID, grant.Permissions, grant.ScopeIDs, grant.ReasonCode, grant.StartAt, grant.ExpiresAt, grant.Version.Value(), grant.PolicyVersion, grant.ReviewStatus, grant.ReviewOutcomeCode})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprintEmergencyAccessGrant(grant EmergencyAccessGrant) string {
	if grant.ID == uuid.Nil {
		return ""
	}
	data, _ := json.Marshal(struct {
		ID                uuid.UUID
		TargetActorID     uuid.UUID
		Status            string
		Permissions       []string
		ScopeIDs          []string
		StartAt           time.Time
		ExpiresAt         time.Time
		Version           int64
		ReviewStatus      string
		ReviewOutcomeCode string
	}{grant.ID, grant.TargetActorID, grant.Status, grant.Permissions, grant.ScopeIDs, grant.StartAt, grant.ExpiresAt, grant.Version.Value(), grant.ReviewStatus, grant.ReviewOutcomeCode})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneEmergencyAccessGrant(grant EmergencyAccessGrant) EmergencyAccessGrant {
	grant.Permissions = append([]string(nil), grant.Permissions...)
	grant.ScopeIDs = append([]string(nil), grant.ScopeIDs...)
	if grant.RevokedAt != nil {
		value := *grant.RevokedAt
		grant.RevokedAt = &value
	}
	if grant.RevokedBy != nil {
		value := *grant.RevokedBy
		grant.RevokedBy = &value
	}
	return grant
}

func cloneEmergencyAccessGrantPtr(grant *EmergencyAccessGrant) *EmergencyAccessGrant {
	if grant == nil {
		return nil
	}
	copy := cloneEmergencyAccessGrant(*grant)
	return &copy
}

func cloneEmergencyAccessCommandResult(result EmergencyAccessCommandResult) EmergencyAccessCommandResult {
	result.Grant = cloneEmergencyAccessGrant(result.Grant)
	return result
}

type emergencyAccessDurableFailurePayload struct {
	Code string `json:"code"`
}

func (service *EmergencyAccessService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := emergencyAccessDurableFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(emergencyAccessDurableFailurePayload{Code: code})
	if err != nil {
		return
	}
	initial := acquisition.Result()
	metadata, err := platformidempotency.NewCommandResultMetadata(initial.Identity(), initial.Fingerprint(), initial.OperationID(), platformidempotency.StateFailed, nil, body, nil, nil)
	if err != nil {
		return
	}
	transaction, err := service.durable.Database.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return
	}
	committed := false
	defer func() {
		if !committed {
			_ = transaction.Rollback(ctx)
		}
	}()
	if err := service.durable.Coordinator.Finalize(ctx, transaction, acquisition, metadata); err != nil {
		return
	}
	if err := transaction.Commit(ctx); err != nil {
		return
	}
	committed = true
}

func emergencyAccessDurableFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrEmergencyAccessAuthorizationDenied):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrEmergencyAccessAuthorizationUnavail), errors.Is(err, ErrEmergencyAccessApprovalUnavailable), errors.Is(err, ErrEmergencyAccessAuditUnavailable):
		return "DEPENDENCY_UNAVAILABLE", true
	case errors.Is(err, ErrEmergencyAccessAuthorizationExpired):
		return "AUTHORIZATION_EXPIRED", true
	case errors.Is(err, ErrEmergencyAccessAuthorizationStale):
		return "AUTHORIZATION_STALE", true
	case errors.Is(err, ErrEmergencyAccessVersionConflict):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrEmergencyAccessAlreadyRevoked):
		return "ALREADY_REVOKED", true
	case errors.Is(err, ErrEmergencyAccessNotFound):
		return "GRANT_NOT_FOUND", true
	case errors.Is(err, ErrInvalidEmergencyAccessCommand), errors.Is(err, ErrInvalidEmergencyAccessGrant), errors.Is(err, ErrEmergencyAccessApprovalRejected), errors.Is(err, ErrEmergencyAccessStepUpRequired), errors.Is(err, ErrEmergencyAccessReviewCompleted), errors.Is(err, ErrEmergencyAccessReviewInvalid):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func emergencyAccessDurableFailureError(body []byte) error {
	var payload emergencyAccessDurableFailurePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrEmergencyAccessCommandInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrEmergencyAccessAuthorizationDenied
	case "DEPENDENCY_UNAVAILABLE":
		return ErrEmergencyAccessAuthorizationUnavail
	case "AUTHORIZATION_EXPIRED":
		return ErrEmergencyAccessAuthorizationExpired
	case "AUTHORIZATION_STALE":
		return ErrEmergencyAccessAuthorizationStale
	case "VERSION_CONFLICT":
		return ErrEmergencyAccessVersionConflict
	case "ALREADY_REVOKED":
		return ErrEmergencyAccessAlreadyRevoked
	case "GRANT_NOT_FOUND":
		return ErrEmergencyAccessNotFound
	case "VALIDATION_FAILED":
		return fmt.Errorf("%w: %s", ErrEmergencyAccessDurableCommandFailed, payload.Code)
	default:
		return ErrEmergencyAccessCommandInProgress
	}
}

func decodeEmergencyAccessCommandResult(body []byte) (EmergencyAccessCommandResult, error) {
	var result EmergencyAccessCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return EmergencyAccessCommandResult{}, err
	}
	return result, nil
}
