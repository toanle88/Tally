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

var (
	ErrInvalidUserCommand             = errors.New("invalid identity user command")
	ErrIdempotencyConflict            = errors.New("identity command idempotency conflict")
	ErrCommandInProgress              = errors.New("identity command is already in progress")
	ErrDuplicateAuthenticationSubject = errors.New("duplicate authentication subject")
	ErrAuditUnavailable               = errors.New("identity audit recorder unavailable")
	ErrInvalidUserService             = errors.New("invalid identity user service")
	ErrDurableCommandFailed           = errors.New("identity command previously failed")
)

const (
	UserActionCreate         = "create"
	UserActionUpdate         = "update"
	UserActionActivate       = "activate"
	UserActionSuspend        = "suspend"
	UserActionTerminate      = "terminate"
	UserManagementPermission = "finance.iam.manage.users"
)

type UserCommand struct {
	Action                string
	UserID                uuid.UUID
	AuthenticationSubject *AuthenticationSubject
	Assignments           []RoleAssignment
	ExpectedVersion       *aggregateversion.AggregateVersion
	IdempotencyKey        string
	CorrelationID         string
	CausationID           string
}

func (command UserCommand) Validate() error {
	switch command.Action {
	case UserActionCreate:
		if command.UserID != uuid.Nil {
			return fmt.Errorf("%w: create must not provide user id", ErrInvalidUserCommand)
		}
		if command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create must not provide expected version", ErrInvalidUserCommand)
		}
		if command.AuthenticationSubject == nil {
			return fmt.Errorf("%w: authentication subject is required", ErrInvalidUserCommand)
		}
		if err := command.AuthenticationSubject.Validate(); err != nil {
			return err
		}
	case UserActionUpdate:
		if command.UserID == uuid.Nil {
			return fmt.Errorf("%w: update user id is required", ErrInvalidUserCommand)
		}
		if command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: update expected version is required", ErrInvalidUserCommand)
		}
		if command.AuthenticationSubject != nil {
			return fmt.Errorf("%w: update cannot replace authentication subject", ErrAuthenticationImmutable)
		}
		if command.Assignments == nil {
			return fmt.Errorf("%w: update requires the complete assignment set", ErrInvalidUserCommand)
		}
	case UserActionActivate, UserActionSuspend, UserActionTerminate:
		if command.UserID == uuid.Nil {
			return fmt.Errorf("%w: user id is required", ErrInvalidUserCommand)
		}
		if command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: expected version is required", ErrInvalidUserCommand)
		}
		if command.AuthenticationSubject != nil || command.Assignments != nil {
			return fmt.Errorf("%w: lifecycle command cannot include identity or assignments", ErrInvalidUserCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidUserCommand, command.Action)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidUserCommand)
	}
	if command.Action == UserActionCreate || command.Action == UserActionUpdate {
		if _, err := normalizeAssignments(command.Assignments); err != nil {
			return err
		}
	}
	return nil
}

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

type UserAuthorizer interface {
	AuthorizeUserManagement(context.Context, ApplicationActor, UserCommand, *User) (AuthorizationDecision, error)
}

// RolePolicyLookup keeps role and policy administration in their owning
// stories while identity validates that referenced identities are approved.
type RolePolicyLookup interface {
	ValidateRoleAssignments(context.Context, []RoleAssignment) error
}

type AllowAllRolePolicyLookup struct{}

func (AllowAllRolePolicyLookup) ValidateRoleAssignments(context.Context, []RoleAssignment) error {
	return nil
}

type AuditRecord struct {
	UserID                        uuid.UUID
	ActorUserID                   uuid.UUID
	ActorAuthenticationSubjectRef string
	Action                        string
	ScopeIDs                      []string
	Permission                    string
	PolicyReference               string
	PolicyVersion                 string
	DecisionReference             uuid.UUID
	ApprovalRequestID             uuid.UUID
	ApprovalDecisionID            uuid.UUID
	ApproverUserID                uuid.UUID
	RevisionVersion               int64
	BeforeFingerprint             string
	AfterFingerprint              string
	CorrelationID                 string
	CausationID                   string
}

type AuditRecorder interface {
	RecordUserMutation(context.Context, AuditRecord) error
}

type UserRepository interface {
	Get(context.Context, uuid.UUID) (User, error)
	FindByAuthenticationSubject(context.Context, AuthenticationSubject) (User, error)
	List(context.Context) ([]User, error)
}

// DurableUserServiceConfig connects IAM commands to the shared platform
// reservation/finalization protocol. Identity supplies its own opaque module
// scope; the platform still owns durable idempotency storage.
type DurableUserServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type UserMutation struct {
	Before          User
	After           User
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           AuditRecord
}

type DurableUserMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

// AtomicUserMutationRepository commits the identity change and audit record
// as one local operation. The PostgreSQL adapter invokes the audit port while
// its identity transaction is still open.
type AtomicUserMutationRepository interface {
	CommitUserMutation(context.Context, UserMutation) error
}

// DurableUserMutationRepository additionally finalizes the shared platform
// idempotency record in the same business transaction.
type DurableUserMutationRepository interface {
	CommitUserMutationWithIdempotency(context.Context, UserMutation, DurableUserMutationCommit) error
}

type UserCommandResult struct {
	User              User
	DecisionReference uuid.UUID
	PolicyReference   string
	Replayed          bool
}

type UserService struct {
	repository UserRepository
	authorizer UserAuthorizer
	rolePolicy RolePolicyLookup
	audit      AuditRecorder
	clock      func() time.Time
	durable    *DurableUserServiceConfig

	idempotencyMu sync.Mutex
	idempotency   map[string]storedUserCommand
}

type storedUserCommand struct {
	fingerprint string
	result      UserCommandResult
}

func NewUserService(repository UserRepository, authorizer UserAuthorizer, audit AuditRecorder, clock func() time.Time, lookups ...RolePolicyLookup) (*UserService, error) {
	return newUserService(repository, authorizer, audit, clock, nil, lookups...)
}

func NewUserServiceWithDurableIdempotency(repository UserRepository, authorizer UserAuthorizer, audit AuditRecorder, clock func() time.Time, durable DurableUserServiceConfig, lookups ...RolePolicyLookup) (*UserService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" || len(lookups) == 0 || lookups[0] == nil {
		return nil, ErrInvalidUserService
	}
	return newUserService(repository, authorizer, audit, clock, &durable, lookups...)
}

func newUserService(repository UserRepository, authorizer UserAuthorizer, audit AuditRecorder, clock func() time.Time, durable *DurableUserServiceConfig, lookups ...RolePolicyLookup) (*UserService, error) {
	if repository == nil || authorizer == nil || audit == nil || clock == nil {
		return nil, ErrInvalidUserService
	}
	rolePolicy := RolePolicyLookup(AllowAllRolePolicyLookup{})
	if len(lookups) > 0 && lookups[0] != nil {
		rolePolicy = lookups[0]
	}
	if binder, ok := repository.(interface{ BindAuditRecorder(AuditRecorder) }); ok {
		binder.BindAuditRecorder(audit)
	}
	return &UserService{
		repository:  repository,
		authorizer:  authorizer,
		rolePolicy:  rolePolicy,
		audit:       audit,
		clock:       clock,
		durable:     durable,
		idempotency: make(map[string]storedUserCommand),
	}, nil
}

func (service *UserService) Execute(ctx context.Context, actor ApplicationActor, command UserCommand) (UserCommandResult, error) {
	if service == nil {
		return UserCommandResult{}, ErrInvalidUserService
	}
	if err := actor.Validate(); err != nil {
		return UserCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return UserCommandResult{}, err
	}

	fingerprint, err := fingerprintCommand(command)
	if err != nil {
		return UserCommandResult{}, err
	}
	idempotencyKey := actor.UserID.String() + ":" + command.IdempotencyKey
	var durableAcquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scopeKey, scopeErr := json.Marshal(struct {
			Module  string    `json:"module"`
			ActorID uuid.UUID `json:"actorId"`
		}{Module: "identity", ActorID: actor.UserID})
		if scopeErr != nil {
			return UserCommandResult{}, scopeErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scopeKey), command.IdempotencyKey)
		if err != nil {
			return UserCommandResult{}, err
		}
		durableAcquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return UserCommandResult{}, ErrIdempotencyConflict
		}
		if err != nil {
			return UserCommandResult{}, err
		}
		if durableAcquisition.Decision() == platformidempotency.DecisionReturn {
			switch durableAcquisition.Result().State() {
			case platformidempotency.StateFailed:
				return UserCommandResult{}, durableFailureError(durableAcquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return UserCommandResult{}, ErrCommandInProgress
			case platformidempotency.StateEstablished:
				result, decodeErr := decodeUserCommandResult(durableAcquisition.Result().ResultBody())
				if decodeErr != nil {
					return UserCommandResult{}, decodeErr
				}
				result.Replayed = true
				return result, nil
			}
		}
	} else {
		service.idempotencyMu.Lock()
		defer service.idempotencyMu.Unlock()

		if previous, ok := service.idempotency[idempotencyKey]; ok {
			if previous.fingerprint != fingerprint {
				return UserCommandResult{}, ErrIdempotencyConflict
			}
			result := cloneCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *User
	if command.Action != UserActionCreate {
		user, getErr := service.repository.Get(ctx, command.UserID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, getErr)
			return UserCommandResult{}, getErr
		}
		current = &user
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, durableAcquisition, ErrVersionConflict)
			return UserCommandResult{}, ErrVersionConflict
		}
	}

	decision, err := service.authorizer.AuthorizeUserManagement(ctx, actor, command, cloneUserPtr(current))
	if err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return UserCommandResult{}, err
	}
	if err := authorizeDecision(command, decision, current); err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return UserCommandResult{}, err
	}

	assignments := command.Assignments
	if command.Action != UserActionCreate && command.Action != UserActionUpdate && current != nil {
		assignments = current.Assignments
	}
	if err := service.rolePolicy.ValidateRoleAssignments(ctx, assignments); err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return UserCommandResult{}, err
	}

	now := service.clock().UTC()
	var before User
	var after User
	switch command.Action {
	case UserActionCreate:
		before = User{}
		after, err = NewUser(uuid.New(), *command.AuthenticationSubject, command.Assignments, now)
	case UserActionUpdate:
		before = cloneUser(*current)
		after = cloneUser(*current)
		if err == nil {
			err = after.ReplaceAssignments(command.Assignments, now)
		}
	case UserActionActivate, UserActionSuspend, UserActionTerminate:
		before = cloneUser(*current)
		after = cloneUser(*current)
		switch command.Action {
		case UserActionActivate:
			err = after.Activate(now)
		case UserActionSuspend:
			err = after.Suspend(now)
		case UserActionTerminate:
			err = after.Terminate(now)
		}
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return UserCommandResult{}, err
	}
	if command.Action != UserActionCreate {
		nextVersion, advanceErr := after.Version.Advance()
		if advanceErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, advanceErr)
			return UserCommandResult{}, advanceErr
		}
		after.Version = nextVersion
	}

	record := AuditRecord{
		UserID:                        after.ID,
		ActorUserID:                   actor.UserID,
		ActorAuthenticationSubjectRef: fingerprintSubject(actor.Subject),
		Action:                        command.Action,
		ScopeIDs:                      assignedScopeIDs(after.Assignments),
		Permission:                    decision.Permission,
		PolicyReference:               decision.PolicyReference,
		PolicyVersion:                 decision.PolicyVersion,
		DecisionReference:             decision.DecisionReference,
		BeforeFingerprint:             fingerprintUser(before),
		AfterFingerprint:              fingerprintUser(after),
		CorrelationID:                 command.CorrelationID,
		CausationID:                   command.CausationID,
	}
	result := UserCommandResult{
		User:              cloneUser(after),
		DecisionReference: decision.DecisionReference,
		PolicyReference:   decision.PolicyReference,
	}
	mutation := UserMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableUserMutationRepository)
		if !ok {
			return UserCommandResult{}, ErrInvalidUserService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			return UserCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			return UserCommandResult{}, metadataErr
		}
		err = committer.CommitUserMutationWithIdempotency(ctx, mutation, DurableUserMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata})
	} else if committer, ok := service.repository.(AtomicUserMutationRepository); ok {
		err = committer.CommitUserMutation(ctx, mutation)
	} else {
		return UserCommandResult{}, ErrInvalidUserService
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return UserCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[idempotencyKey] = storedUserCommand{fingerprint: fingerprint, result: cloneCommandResult(result)}
	}
	return result, nil
}

type durableFailurePayload struct {
	Code string `json:"code"`
}

func (service *UserService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := durableFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(durableFailurePayload{Code: code})
	if err != nil {
		return
	}
	initial := acquisition.Result()
	metadata, err := platformidempotency.NewCommandResultMetadata(
		initial.Identity(), initial.Fingerprint(), initial.OperationID(),
		platformidempotency.StateFailed, nil, body, nil, nil,
	)
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

func durableFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrAuthorizationDenied):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrAuthorizationUnavailable):
		return "AUTHORIZATION_UNAVAILABLE", true
	case errors.Is(err, ErrAuthorizationExpired):
		return "AUTHORIZATION_EXPIRED", true
	case errors.Is(err, ErrAuthorizationStale):
		return "AUTHORIZATION_STALE", true
	case errors.Is(err, ErrVersionConflict):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrUserNotFound):
		return "USER_NOT_FOUND", true
	case errors.Is(err, ErrDuplicateAuthenticationSubject):
		return "DUPLICATE_USER", true
	case errors.Is(err, ErrInvalidUserCommand), errors.Is(err, ErrInvalidAssignment), errors.Is(err, ErrDuplicateAssignment), errors.Is(err, ErrInvalidAuthenticationSubject), errors.Is(err, ErrInvalidUserTransition), errors.Is(err, ErrUserTerminated), errors.Is(err, ErrAuthenticationImmutable):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func durableFailureError(body []byte) error {
	var payload durableFailurePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrCommandInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrAuthorizationDenied
	case "AUTHORIZATION_UNAVAILABLE":
		return ErrAuthorizationUnavailable
	case "AUTHORIZATION_EXPIRED":
		return ErrAuthorizationExpired
	case "AUTHORIZATION_STALE":
		return ErrAuthorizationStale
	case "VERSION_CONFLICT":
		return ErrVersionConflict
	case "USER_NOT_FOUND":
		return ErrUserNotFound
	case "DUPLICATE_USER":
		return ErrDuplicateAuthenticationSubject
	case "VALIDATION_FAILED":
		return fmt.Errorf("%w: %s", ErrDurableCommandFailed, payload.Code)
	default:
		return ErrCommandInProgress
	}
}

func authorizeDecision(command UserCommand, decision AuthorizationDecision, current *User) error {
	switch decision.Outcome {
	case AuthorizationExpired:
		return fmt.Errorf("%w: %s", ErrAuthorizationExpired, decision.ReasonCode)
	case AuthorizationUnavailable:
		return ErrAuthorizationUnavailable
	case AuthorizationStale:
		return ErrAuthorizationStale
	}
	if !decision.Allowed || decision.Permission != UserManagementPermission {
		return fmt.Errorf("%w: %s", ErrAuthorizationDenied, decision.Reason)
	}
	if decision.DecisionReference == uuid.Nil {
		return fmt.Errorf("%w: authorization decision reference is required", ErrAuthorizationDenied)
	}
	var assignments []RoleAssignment
	if command.Action == UserActionCreate || command.Action == UserActionUpdate {
		assignments = command.Assignments
	} else if current != nil {
		assignments = current.Assignments
	}
	approved := make(map[string]struct{}, len(decision.ApprovedScopeIDs))
	for _, scopeID := range decision.ApprovedScopeIDs {
		approved[strings.TrimSpace(scopeID)] = struct{}{}
	}
	for _, assignment := range assignments {
		for _, scope := range assignment.Scopes {
			scopeID := strings.TrimSpace(scope.ScopeID)
			if _, ok := approved["*"]; ok {
				continue
			}
			if _, ok := approved[scopeID]; !ok {
				return fmt.Errorf("%w: scope %q is outside the administering actor scope", ErrAuthorizationDenied, scopeID)
			}
		}
	}
	return nil
}

type MemoryUserRepository struct {
	mu       sync.RWMutex
	users    map[uuid.UUID]User
	subjects map[subjectKey]uuid.UUID
	audit    AuditRecorder
}

func NewMemoryUserRepository() *MemoryUserRepository {
	return &MemoryUserRepository{users: make(map[uuid.UUID]User), subjects: make(map[subjectKey]uuid.UUID)}
}

func (repository *MemoryUserRepository) BindAuditRecorder(audit AuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryUserRepository) CommitUserMutation(ctx context.Context, mutation UserMutation) error {
	if repository == nil {
		return ErrInvalidUserService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.After.ID == uuid.Nil {
		return ErrInvalidUser
	}
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.users[mutation.After.ID]; exists {
			return fmt.Errorf("%w: user id", ErrDuplicateAuthenticationSubject)
		}
		key := keyFor(mutation.After.AuthenticationSubject)
		if _, exists := repository.subjects[key]; exists {
			return ErrDuplicateAuthenticationSubject
		}
	} else {
		current, exists := repository.users[mutation.After.ID]
		if !exists {
			return ErrUserNotFound
		}
		if mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
			return ErrVersionConflict
		}
		if mutation.After.AuthenticationSubject != current.AuthenticationSubject {
			return ErrAuthenticationImmutable
		}
	}
	if repository.audit == nil {
		return ErrAuditUnavailable
	}
	if err := repository.audit.RecordUserMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.users[mutation.After.ID] = cloneUser(mutation.After)
	if mutation.Before.ID == uuid.Nil {
		repository.subjects[keyFor(mutation.After.AuthenticationSubject)] = mutation.After.ID
	}
	return nil
}

func (repository *MemoryUserRepository) Create(_ context.Context, user User) error {
	if repository == nil {
		return ErrInvalidUserService
	}
	if err := user.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if _, exists := repository.users[user.ID]; exists {
		return fmt.Errorf("%w: user id", ErrDuplicateAuthenticationSubject)
	}
	key := keyFor(user.AuthenticationSubject)
	if _, exists := repository.subjects[key]; exists {
		return ErrDuplicateAuthenticationSubject
	}
	repository.users[user.ID] = cloneUser(user)
	repository.subjects[key] = user.ID
	return nil
}

func (repository *MemoryUserRepository) Get(_ context.Context, id uuid.UUID) (User, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	user, ok := repository.users[id]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return cloneUser(user), nil
}

func (repository *MemoryUserRepository) FindByAuthenticationSubject(_ context.Context, subject AuthenticationSubject) (User, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	userID, ok := repository.subjects[keyFor(subject)]
	if !ok {
		return User{}, ErrUserNotFound
	}
	return cloneUser(repository.users[userID]), nil
}

func (repository *MemoryUserRepository) Replace(_ context.Context, user User, expected aggregateversion.AggregateVersion) error {
	if repository == nil {
		return ErrInvalidUserService
	}
	if err := user.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	current, ok := repository.users[user.ID]
	if !ok {
		return ErrUserNotFound
	}
	if !current.Version.Matches(expected) {
		return ErrVersionConflict
	}
	if user.AuthenticationSubject != current.AuthenticationSubject {
		return ErrAuthenticationImmutable
	}
	if user.Version.Value() != expected.Value()+1 {
		return ErrVersionConflict
	}
	repository.users[user.ID] = cloneUser(user)
	return nil
}

func (repository *MemoryUserRepository) List(_ context.Context) ([]User, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]User, 0, len(repository.users))
	for _, user := range repository.users {
		result = append(result, cloneUser(user))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

type MemoryUserAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

var _ AtomicUserMutationRepository = (*MemoryUserRepository)(nil)

func (authorizer MemoryUserAuthorizer) AuthorizeUserManagement(context.Context, ApplicationActor, UserCommand, *User) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return cloneAuthorizationDecision(authorizer.Decision), nil
}

type MemoryAuditRecorder struct {
	mu      sync.Mutex
	Records []AuditRecord
	Err     error
}

func (recorder *MemoryAuditRecorder) RecordUserMutation(_ context.Context, record AuditRecord) error {
	if recorder == nil {
		return ErrInvalidUserService
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func cloneAuthorizationDecision(decision AuthorizationDecision) AuthorizationDecision {
	decision.ApprovedScopeIDs = append([]string(nil), decision.ApprovedScopeIDs...)
	decision.ApplicableDimensions = append([]string(nil), decision.ApplicableDimensions...)
	return decision
}

func cloneCommandResult(result UserCommandResult) UserCommandResult {
	result.User = cloneUser(result.User)
	return result
}

func decodeUserCommandResult(data []byte) (UserCommandResult, error) {
	if len(data) == 0 {
		return UserCommandResult{}, ErrCommandInProgress
	}
	var result UserCommandResult
	if err := json.Unmarshal(data, &result); err != nil {
		return UserCommandResult{}, fmt.Errorf("decode established identity result: %w", err)
	}
	return result, nil
}

func cloneUserPtr(user *User) *User {
	if user == nil {
		return nil
	}
	copy := cloneUser(*user)
	return &copy
}

func cloneUser(user User) User {
	assignments := make([]RoleAssignment, len(user.Assignments))
	for index, assignment := range user.Assignments {
		assignments[index] = RoleAssignment{
			RoleID: assignment.RoleID,
			Scopes: append([]EntityAccessScope(nil), assignment.Scopes...),
		}
	}
	user.Assignments = assignments
	return user
}

func fingerprintCommand(command UserCommand) (string, error) {
	normalized, err := normalizeAssignments(command.Assignments)
	if err != nil {
		return "", err
	}
	payload := struct {
		Action          string
		UserID          string
		Subject         *AuthenticationSubject
		Assignments     []RoleAssignment
		ExpectedVersion int64
	}{
		Action: command.Action, UserID: command.UserID.String(), Subject: command.AuthenticationSubject,
		Assignments: normalized,
	}
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

func fingerprintSubject(subject AuthenticationSubject) string {
	data, _ := json.Marshal(subject)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprintUser(user User) string {
	if user.ID == uuid.Nil {
		return ""
	}
	data, _ := json.Marshal(struct {
		ID          uuid.UUID
		SubjectRef  string
		Status      UserStatus
		Assignments []RoleAssignment
		Version     int64
	}{
		ID: user.ID, SubjectRef: fingerprintSubject(user.AuthenticationSubject),
		Status: user.Status, Assignments: user.Assignments, Version: user.Version.Value(),
	})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func assignedScopeIDs(assignments []RoleAssignment) []string {
	seen := make(map[string]struct{})
	for _, assignment := range assignments {
		for _, scope := range assignment.Scopes {
			seen[scope.ScopeID] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for scopeID := range seen {
		result = append(result, scopeID)
	}
	sort.Strings(result)
	return result
}
