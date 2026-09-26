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

type RoleCommand struct {
	Action          string
	RoleID          uuid.UUID
	Name            string
	Grants          []PermissionGrant
	Approval        ApprovalDecisionReference
	ExpectedVersion *aggregateversion.AggregateVersion
	IdempotencyKey  string
	CorrelationID   string
	CausationID     string
}

func (command RoleCommand) Validate() error {
	switch command.Action {
	case RoleActionCreate:
		if command.RoleID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create must not provide role id or expected version", ErrInvalidRoleCommand)
		}
		if strings.TrimSpace(command.Name) == "" {
			return fmt.Errorf("%w: role name is required", ErrInvalidRoleCommand)
		}
	case RoleActionUpdate:
		if command.RoleID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: update role id and expected version are required", ErrInvalidRoleCommand)
		}
		if strings.TrimSpace(command.Name) == "" {
			return fmt.Errorf("%w: role name is required", ErrInvalidRoleCommand)
		}
	case RoleActionRetire:
		if command.RoleID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: retire role id and expected version are required", ErrInvalidRoleCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidRoleCommand, command.Action)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidRoleCommand)
	}
	if err := command.Approval.Validate(); err != nil {
		return err
	}
	if command.Action != RoleActionRetire {
		if _, err := normalizePermissionGrants(command.Grants); err != nil {
			return err
		}
	}
	return nil
}

type RoleAuthorizer interface {
	AuthorizeRoleManagement(context.Context, ApplicationActor, RoleCommand, *Role) (AuthorizationDecision, error)
}

type RoleApprovalPort interface {
	ValidateRoleApproval(context.Context, ApplicationActor, RoleCommand, *Role, string) error
}

type RoleSegregationPort interface {
	ValidateRoleGrants(context.Context, []PermissionGrant) error
}

type AllowAllRoleApprovalPort struct{}

func (AllowAllRoleApprovalPort) ValidateRoleApproval(_ context.Context, actor ApplicationActor, command RoleCommand, current *Role, fingerprint string) error {
	expectedVersion := int64(1)
	if current != nil {
		expectedVersion = current.Version.Value() + 1
	}
	if command.Approval.SubjectVersion != expectedVersion {
		return fmt.Errorf("%w: subject version does not match candidate revision", ErrApprovalRejected)
	}
	if command.Approval.CandidateFingerprint != fingerprint {
		return fmt.Errorf("%w: candidate fingerprint does not match", ErrApprovalRejected)
	}
	if command.Approval.ApproverUserID == actor.UserID {
		return fmt.Errorf("%w: requesting actor cannot approve the same role change", ErrApprovalRejected)
	}
	return nil
}

type AllowAllRoleSegregationPort struct{}

func (AllowAllRoleSegregationPort) ValidateRoleGrants(context.Context, []PermissionGrant) error {
	return nil
}

type RoleAuditRecord struct {
	RoleID                 uuid.UUID
	ActorUserID            uuid.UUID
	ActorAuthenticationRef string
	Action                 string
	ScopeIDs               []string
	Permission             string
	PolicyReference        string
	DecisionReference      uuid.UUID
	ApprovalRequestID      uuid.UUID
	ApprovalDecisionID     uuid.UUID
	ApproverUserID         uuid.UUID
	RoleVersion            int64
	BeforeFingerprint      string
	AfterFingerprint       string
	CorrelationID          string
	CausationID            string
}

type RoleAuditRecorder interface {
	RecordRoleMutation(context.Context, RoleAuditRecord) error
}

type RoleRepository interface {
	Get(context.Context, uuid.UUID) (Role, error)
	List(context.Context) ([]Role, error)
	ValidateRoleAssignments(context.Context, []RoleAssignment) error
}

type RoleMutation struct {
	Before          Role
	After           Role
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           RoleAuditRecord
}

type AtomicRoleMutationRepository interface {
	CommitRoleMutation(context.Context, RoleMutation) error
}

type RoleCommandResult struct {
	Role              Role
	DecisionReference uuid.UUID
	PolicyReference   string
	Approval          ApprovalDecisionReference
	Replayed          bool
}

type DurableRoleServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableRoleMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableRoleMutationRepository interface {
	CommitRoleMutationWithIdempotency(context.Context, RoleMutation, DurableRoleMutationCommit) error
}

type RoleService struct {
	repository  RoleRepository
	authorizer  RoleAuthorizer
	approval    RoleApprovalPort
	segregation RoleSegregationPort
	audit       RoleAuditRecorder
	clock       func() time.Time
	durable     *DurableRoleServiceConfig

	idempotencyMu sync.Mutex
	idempotency   map[string]storedRoleCommand
}

type storedRoleCommand struct {
	fingerprint string
	result      RoleCommandResult
}

func NewRoleService(repository RoleRepository, authorizer RoleAuthorizer, approval RoleApprovalPort, segregation RoleSegregationPort, audit RoleAuditRecorder, clock func() time.Time) (*RoleService, error) {
	return newRoleService(repository, authorizer, approval, segregation, audit, clock, nil)
}

func NewRoleServiceWithDurableIdempotency(repository RoleRepository, authorizer RoleAuthorizer, approval RoleApprovalPort, segregation RoleSegregationPort, audit RoleAuditRecorder, clock func() time.Time, durable DurableRoleServiceConfig) (*RoleService, error) {
	if durable.Database == nil || durable.Coordinator == nil ||
		durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 ||
		durable.Policy.LeaseTTL >= durable.Policy.RecordTTL ||
		strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidRoleService
	}
	return newRoleService(repository, authorizer, approval, segregation, audit, clock, &durable)
}

func newRoleService(repository RoleRepository, authorizer RoleAuthorizer, approval RoleApprovalPort, segregation RoleSegregationPort, audit RoleAuditRecorder, clock func() time.Time, durable *DurableRoleServiceConfig) (*RoleService, error) {
	if repository == nil || authorizer == nil || approval == nil || segregation == nil || audit == nil || clock == nil {
		return nil, ErrInvalidRoleService
	}
	if binder, ok := repository.(interface{ BindAuditRecorder(RoleAuditRecorder) }); ok {
		binder.BindAuditRecorder(audit)
	}
	return &RoleService{
		repository: repository, authorizer: authorizer, approval: approval,
		segregation: segregation, audit: audit, clock: clock, durable: durable,
		idempotency: make(map[string]storedRoleCommand),
	}, nil
}

func (service *RoleService) Execute(ctx context.Context, actor ApplicationActor, command RoleCommand) (RoleCommandResult, error) {
	if service == nil {
		return RoleCommandResult{}, ErrInvalidRoleService
	}
	if err := actor.Validate(); err != nil {
		return RoleCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return RoleCommandResult{}, err
	}
	fingerprint, err := fingerprintRoleCommand(command)
	if err != nil {
		return RoleCommandResult{}, err
	}
	idempotencyKey := actor.UserID.String() + ":" + command.IdempotencyKey
	var durableAcquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		scopeKey, scopeErr := json.Marshal(struct {
			Module  string    `json:"module"`
			ActorID uuid.UUID `json:"actorId"`
		}{Module: "identity.role", ActorID: actor.UserID})
		if scopeErr != nil {
			return RoleCommandResult{}, scopeErr
		}
		durableIdentity, err = platformidempotency.NewOpaqueIdentity(string(scopeKey), command.IdempotencyKey)
		if err != nil {
			return RoleCommandResult{}, err
		}
		durableAcquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return RoleCommandResult{}, ErrRoleIdempotencyConflict
		}
		if err != nil {
			return RoleCommandResult{}, err
		}
		if durableAcquisition.Decision() == platformidempotency.DecisionReturn {
			switch durableAcquisition.Result().State() {
			case platformidempotency.StateFailed:
				return RoleCommandResult{}, roleDurableFailureError(durableAcquisition.Result().ResultBody())
			case platformidempotency.StateInProgress:
				return RoleCommandResult{}, ErrRoleCommandInProgress
			case platformidempotency.StateEstablished:
				result, decodeErr := decodeRoleCommandResult(durableAcquisition.Result().ResultBody())
				if decodeErr != nil {
					return RoleCommandResult{}, decodeErr
				}
				result.Replayed = true
				return result, nil
			}
		}
	} else {
		service.idempotencyMu.Lock()
		defer service.idempotencyMu.Unlock()
		if previous, exists := service.idempotency[idempotencyKey]; exists {
			if previous.fingerprint != fingerprint {
				return RoleCommandResult{}, ErrRoleIdempotencyConflict
			}
			result := cloneRoleCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *Role
	if command.Action != RoleActionCreate {
		role, getErr := service.repository.Get(ctx, command.RoleID)
		if getErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, getErr)
			return RoleCommandResult{}, getErr
		}
		current = &role
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeDurableFailure(ctx, durableAcquisition, ErrVersionConflict)
			return RoleCommandResult{}, ErrVersionConflict
		}
	}
	decision, err := service.authorizer.AuthorizeRoleManagement(ctx, actor, command, cloneRolePtr(current))
	if err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return RoleCommandResult{}, err
	}
	if err := authorizeRoleDecision(command, decision, current); err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return RoleCommandResult{}, err
	}

	now := service.clock().UTC()
	var before Role
	var after Role
	switch command.Action {
	case RoleActionCreate:
		after, err = NewRole(uuid.New(), command.Name, command.Grants, command.Approval, now)
	case RoleActionUpdate:
		before = cloneRole(*current)
		after = cloneRole(*current)
		err = after.Replace(command.Name, command.Grants, command.Approval, now)
	case RoleActionRetire:
		before = cloneRole(*current)
		after = cloneRole(*current)
		err = after.Retire(command.Approval, now)
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return RoleCommandResult{}, err
	}
	if command.Action != RoleActionCreate {
		nextVersion, advanceErr := after.Version.Advance()
		if advanceErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, advanceErr)
			return RoleCommandResult{}, advanceErr
		}
		after.Version = nextVersion
	}
	candidateFingerprint := candidateRoleFingerprint(command, after)
	if err := service.approval.ValidateRoleApproval(ctx, actor, command, current, candidateFingerprint); err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return RoleCommandResult{}, err
	}
	if err := service.segregation.ValidateRoleGrants(ctx, after.Grants); err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return RoleCommandResult{}, err
	}

	record := RoleAuditRecord{
		RoleID: after.ID, ActorUserID: actor.UserID,
		ActorAuthenticationRef: fingerprintSubject(actor.Subject), Action: command.Action,
		ScopeIDs: roleScopeIDs(after), Permission: decision.Permission,
		PolicyReference: decision.PolicyReference, DecisionReference: decision.DecisionReference,
		ApprovalRequestID:  command.Approval.ApprovalRequestID,
		ApprovalDecisionID: command.Approval.DecisionID, ApproverUserID: command.Approval.ApproverUserID,
		RoleVersion:       after.Version.Value(),
		BeforeFingerprint: fingerprintRole(before), AfterFingerprint: candidateFingerprint,
		CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	result := RoleCommandResult{
		Role: cloneRole(after), DecisionReference: decision.DecisionReference,
		PolicyReference: decision.PolicyReference, Approval: command.Approval,
	}
	mutation := RoleMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableRoleMutationRepository)
		if !ok {
			service.finalizeDurableFailure(ctx, durableAcquisition, ErrInvalidRoleService)
			return RoleCommandResult{}, ErrInvalidRoleService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, marshalErr)
			return RoleCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeDurableFailure(ctx, durableAcquisition, metadataErr)
			return RoleCommandResult{}, metadataErr
		}
		err = committer.CommitRoleMutationWithIdempotency(ctx, mutation, DurableRoleMutationCommit{
			Coordinator: service.durable.Coordinator, Acquisition: durableAcquisition, Result: metadata,
		})
	} else if committer, ok := service.repository.(AtomicRoleMutationRepository); ok {
		err = committer.CommitRoleMutation(ctx, mutation)
	} else {
		return RoleCommandResult{}, ErrInvalidRoleService
	}
	if err != nil {
		service.finalizeDurableFailure(ctx, durableAcquisition, err)
		return RoleCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[idempotencyKey] = storedRoleCommand{fingerprint: fingerprint, result: cloneRoleCommandResult(result)}
	}
	return result, nil
}

func (service *RoleService) finalizeDurableFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	code, ok := roleDurableFailureCode(commandErr)
	if !ok {
		return
	}
	body, err := json.Marshal(durableFailurePayload{Code: code})
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

func roleDurableFailureCode(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrRoleAuthorizationDenied):
		return "AUTHORIZATION_DENIED", true
	case errors.Is(err, ErrVersionConflict):
		return "VERSION_CONFLICT", true
	case errors.Is(err, ErrRoleNotFound):
		return "ROLE_NOT_FOUND", true
	case errors.Is(err, ErrApprovalRejected), errors.Is(err, ErrApprovalRequired), errors.Is(err, ErrSegregationConflict),
		errors.Is(err, ErrInvalidRoleCommand), errors.Is(err, ErrInvalidPermissionGrant), errors.Is(err, ErrDuplicatePermissionGrant),
		errors.Is(err, ErrRoleRetired):
		return "VALIDATION_FAILED", true
	default:
		return "", false
	}
}

func roleDurableFailureError(body []byte) error {
	var payload durableFailurePayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return ErrRoleCommandInProgress
	}
	switch payload.Code {
	case "AUTHORIZATION_DENIED":
		return ErrRoleAuthorizationDenied
	case "VERSION_CONFLICT":
		return ErrVersionConflict
	case "ROLE_NOT_FOUND":
		return ErrRoleNotFound
	case "VALIDATION_FAILED":
		return fmt.Errorf("%w: %s", ErrRoleDurableCommandFailed, payload.Code)
	default:
		return ErrRoleCommandInProgress
	}
}

func decodeRoleCommandResult(body []byte) (RoleCommandResult, error) {
	var result RoleCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return RoleCommandResult{}, err
	}
	return result, nil
}

func authorizeRoleDecision(command RoleCommand, decision AuthorizationDecision, current *Role) error {
	if !decision.Allowed || decision.Permission != RoleManagementPermission || decision.DecisionReference == uuid.Nil {
		return fmt.Errorf("%w: %s", ErrRoleAuthorizationDenied, decision.Reason)
	}
	grants := command.Grants
	if command.Action == RoleActionRetire && current != nil {
		grants = current.Grants
	}
	approved := make(map[string]struct{}, len(decision.ApprovedScopeIDs))
	for _, scopeID := range decision.ApprovedScopeIDs {
		approved[strings.TrimSpace(scopeID)] = struct{}{}
	}
	for _, grant := range grants {
		for _, scopeID := range grant.ScopeIDs {
			if _, ok := approved["*"]; ok {
				continue
			}
			if _, ok := approved[strings.TrimSpace(scopeID)]; !ok {
				return fmt.Errorf("%w: scope %q is outside the administering actor scope", ErrRoleAuthorizationDenied, scopeID)
			}
		}
	}
	return nil
}

type MemoryRoleRepository struct {
	mu    sync.RWMutex
	roles map[uuid.UUID]Role
	audit RoleAuditRecorder
}

func NewMemoryRoleRepository() *MemoryRoleRepository {
	return &MemoryRoleRepository{roles: make(map[uuid.UUID]Role)}
}

func (repository *MemoryRoleRepository) BindAuditRecorder(audit RoleAuditRecorder) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryRoleRepository) CommitRoleMutation(ctx context.Context, mutation RoleMutation) error {
	if repository == nil {
		return ErrInvalidRoleService
	}
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.roles[mutation.After.ID]; exists {
			return ErrRoleIdempotencyConflict
		}
	} else {
		current, exists := repository.roles[mutation.After.ID]
		if !exists {
			return ErrRoleNotFound
		}
		if mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) ||
			mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 {
			return ErrVersionConflict
		}
	}
	if repository.audit == nil {
		return ErrRoleAuditUnavailable
	}
	if err := repository.audit.RecordRoleMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.roles[mutation.After.ID] = cloneRole(mutation.After)
	return nil
}

func (repository *MemoryRoleRepository) Get(_ context.Context, id uuid.UUID) (Role, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	role, ok := repository.roles[id]
	if !ok {
		return Role{}, ErrRoleNotFound
	}
	return cloneRole(role), nil
}

func (repository *MemoryRoleRepository) List(_ context.Context) ([]Role, error) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]Role, 0, len(repository.roles))
	for _, role := range repository.roles {
		result = append(result, cloneRole(role))
	}
	sort.Slice(result, func(left, right int) bool { return result[left].ID.String() < result[right].ID.String() })
	return result, nil
}

func (repository *MemoryRoleRepository) ValidateRoleAssignments(_ context.Context, assignments []RoleAssignment) error {
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	for _, assignment := range assignments {
		role, ok := repository.roles[assignment.RoleID]
		if !ok {
			return ErrRoleNotFound
		}
		if role.Status != RoleStatusActive {
			return ErrRoleRetired
		}
	}
	return nil
}

type MemoryRoleAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryRoleAuthorizer) AuthorizeRoleManagement(context.Context, ApplicationActor, RoleCommand, *Role) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return cloneAuthorizationDecision(authorizer.Decision), nil
}

type MemoryRoleAuditRecorder struct {
	mu      sync.Mutex
	Records []RoleAuditRecord
	Err     error
}

func (recorder *MemoryRoleAuditRecorder) RecordRoleMutation(_ context.Context, record RoleAuditRecord) error {
	if recorder == nil {
		return ErrRoleAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func cloneRolePtr(role *Role) *Role {
	if role == nil {
		return nil
	}
	copy := cloneRole(*role)
	return &copy
}

func cloneRoleCommandResult(result RoleCommandResult) RoleCommandResult {
	result.Role = cloneRole(result.Role)
	return result
}

func roleScopeIDs(role Role) []string {
	seen := map[string]struct{}{}
	for _, grant := range role.Grants {
		for _, scopeID := range grant.ScopeIDs {
			seen[scopeID] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for scopeID := range seen {
		result = append(result, scopeID)
	}
	sort.Strings(result)
	return result
}

func fingerprintRoleCommand(command RoleCommand) (string, error) {
	payload := struct {
		Action          string
		RoleID          string
		Name            string
		Grants          []PermissionGrant
		Approval        ApprovalDecisionReference
		ExpectedVersion int64
	}{Action: command.Action, RoleID: command.RoleID.String(), Name: strings.TrimSpace(command.Name),
		Grants: command.Grants, Approval: command.Approval}
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

func candidateRoleFingerprint(command RoleCommand, role Role) string {
	grants, err := normalizePermissionGrants(role.Grants)
	if err != nil {
		return ""
	}
	roleID := uuid.Nil
	if command.Action != RoleActionCreate {
		roleID = role.ID
	}
	data, _ := json.Marshal(struct {
		Action  string
		RoleID  uuid.UUID
		Name    string
		Grants  []PermissionGrant
		Version int64
	}{command.Action, roleID, role.Name, grants, role.Version.Value()})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprintRole(role Role) string {
	if role.ID == uuid.Nil {
		return ""
	}
	data, _ := json.Marshal(struct {
		ID      uuid.UUID
		Name    string
		Status  string
		Grants  []PermissionGrant
		Version int64
	}{role.ID, role.Name, role.Status, role.Grants, role.Version.Value()})
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
