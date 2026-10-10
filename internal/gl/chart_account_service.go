package gl

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

type ChartOfAccountsAuthorizer interface {
	AuthorizeChartOfAccounts(context.Context, Actor, ChartOfAccountsCommand, *ChartOfAccounts) (AuthorizationDecision, error)
}

type AccountAuthorizer interface {
	AuthorizeAccount(context.Context, Actor, AccountCommand, *Account) (AuthorizationDecision, error)
}

// ChartAccountReferenceValidator is the GL application boundary for parent,
// COA, OMD, and reporting references. Implementations must not read another
// module's adapter or schema directly.
type ChartAccountReferenceValidator interface {
	ValidateChartOfAccountsReferences(context.Context, Actor, ChartOfAccountsCommand) error
	ValidateAccountReferences(context.Context, Actor, AccountCommand) error
}

type ChartAccountApprovalValidator interface {
	ValidateChartOfAccountsApproval(context.Context, Actor, ChartOfAccountsCommand, ChartOfAccounts) error
	ValidateAccountApproval(context.Context, Actor, AccountCommand, Account) error
}

type ChartOfAccountsAuditRecord struct {
	ChartOfAccountsID     uuid.UUID
	ApprovalRequestID     uuid.UUID
	ApprovalDecisionID    uuid.UUID
	ApproverUserID        uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	AccountingScopeID     uuid.UUID
	LedgerID              uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	RevisionNumber        int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CorrelationID         string
	CausationID           string
}

type AccountAuditRecord struct {
	AccountID             uuid.UUID
	ApprovalRequestID     uuid.UUID
	ApprovalDecisionID    uuid.UUID
	ApproverUserID        uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	AccountingScopeID     uuid.UUID
	ChartOfAccountsID     uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	RevisionNumber        int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CorrelationID         string
	CausationID           string
}

type ChartOfAccountsAuditRecorder interface {
	RecordChartOfAccountsMutation(context.Context, ChartOfAccountsAuditRecord) error
}

type AccountAuditRecorder interface {
	RecordAccountMutation(context.Context, AccountAuditRecord) error
}

type ChartOfAccountsMutation struct {
	Before          ChartOfAccounts
	After           ChartOfAccounts
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           ChartOfAccountsAuditRecord
}

type AccountMutation struct {
	Before          Account
	After           Account
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           AccountAuditRecord
}

type ChartOfAccountsRepository interface {
	GetChartOfAccounts(context.Context, uuid.UUID) (ChartOfAccounts, error)
	ListChartsOfAccounts(context.Context, *uuid.UUID) ([]ChartOfAccounts, error)
	CommitChartOfAccountsMutation(context.Context, ChartOfAccountsMutation) error
}

type AccountRepository interface {
	GetAccount(context.Context, uuid.UUID) (Account, error)
	ListAccounts(context.Context, *uuid.UUID) ([]Account, error)
	CommitAccountMutation(context.Context, AccountMutation) error
}

type DurableChartOfAccountsRepository interface {
	CommitChartOfAccountsMutationWithIdempotency(context.Context, ChartOfAccountsMutation, DurableChartOfAccountsMutationCommit) error
}

type DurableAccountRepository interface {
	CommitAccountMutationWithIdempotency(context.Context, AccountMutation, DurableAccountMutationCommit) error
}

type DurableChartOfAccountsMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableAccountMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableChartOfAccountsServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableAccountServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type ChartOfAccountsCommandResult struct {
	ChartOfAccounts   SafeChartOfAccounts `json:"chartOfAccounts"`
	DecisionReference uuid.UUID           `json:"decisionReference"`
	PolicyReference   string              `json:"policyReference"`
	ValidationOutcome string              `json:"validationOutcome"`
	ApprovalStatus    string              `json:"approvalStatus"`
	Replayed          bool                `json:"replayed,omitempty"`
}

type AccountCommandResult struct {
	Account           SafeAccount `json:"account"`
	DecisionReference uuid.UUID   `json:"decisionReference"`
	PolicyReference   string      `json:"policyReference"`
	ValidationOutcome string      `json:"validationOutcome"`
	ApprovalStatus    string      `json:"approvalStatus"`
	Replayed          bool        `json:"replayed,omitempty"`
}

type ChartOfAccountsService struct {
	repository  ChartOfAccountsRepository
	authorizer  ChartOfAccountsAuthorizer
	references  ChartAccountReferenceValidator
	approval    ChartAccountApprovalValidator
	audit       ChartOfAccountsAuditRecorder
	clock       func() time.Time
	durable     *DurableChartOfAccountsServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedChartOfAccountsCommand
}

type AccountService struct {
	repository  AccountRepository
	authorizer  AccountAuthorizer
	references  ChartAccountReferenceValidator
	approval    ChartAccountApprovalValidator
	audit       AccountAuditRecorder
	clock       func() time.Time
	durable     *DurableAccountServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedAccountCommand
}

type storedChartOfAccountsCommand struct {
	fingerprint string
	result      ChartOfAccountsCommandResult
}

type storedAccountCommand struct {
	fingerprint string
	result      AccountCommandResult
}

func NewChartOfAccountsService(repository ChartOfAccountsRepository, authorizer ChartOfAccountsAuthorizer, references ChartAccountReferenceValidator, approval ChartAccountApprovalValidator, audit ChartOfAccountsAuditRecorder, clock func() time.Time) (*ChartOfAccountsService, error) {
	return newChartOfAccountsService(repository, authorizer, references, approval, audit, clock, nil)
}

func NewChartOfAccountsServiceWithDurableIdempotency(repository ChartOfAccountsRepository, authorizer ChartOfAccountsAuthorizer, references ChartAccountReferenceValidator, approval ChartAccountApprovalValidator, audit ChartOfAccountsAuditRecorder, clock func() time.Time, durable DurableChartOfAccountsServiceConfig) (*ChartOfAccountsService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidChartOfAccountsService
	}
	return newChartOfAccountsService(repository, authorizer, references, approval, audit, clock, &durable)
}

func newChartOfAccountsService(repository ChartOfAccountsRepository, authorizer ChartOfAccountsAuthorizer, references ChartAccountReferenceValidator, approval ChartAccountApprovalValidator, audit ChartOfAccountsAuditRecorder, clock func() time.Time, durable *DurableChartOfAccountsServiceConfig) (*ChartOfAccountsService, error) {
	if repository == nil || authorizer == nil || references == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidChartOfAccountsService
	}
	if binder, ok := repository.(interface {
		BindChartOfAccountsAuditRecorder(ChartOfAccountsAuditRecorder)
	}); ok {
		binder.BindChartOfAccountsAuditRecorder(audit)
	}
	return &ChartOfAccountsService{repository: repository, authorizer: authorizer, references: references, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedChartOfAccountsCommand)}, nil
}

func NewAccountService(repository AccountRepository, authorizer AccountAuthorizer, references ChartAccountReferenceValidator, approval ChartAccountApprovalValidator, audit AccountAuditRecorder, clock func() time.Time) (*AccountService, error) {
	return newAccountService(repository, authorizer, references, approval, audit, clock, nil)
}

func NewAccountServiceWithDurableIdempotency(repository AccountRepository, authorizer AccountAuthorizer, references ChartAccountReferenceValidator, approval ChartAccountApprovalValidator, audit AccountAuditRecorder, clock func() time.Time, durable DurableAccountServiceConfig) (*AccountService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidAccountService
	}
	return newAccountService(repository, authorizer, references, approval, audit, clock, &durable)
}

func newAccountService(repository AccountRepository, authorizer AccountAuthorizer, references ChartAccountReferenceValidator, approval ChartAccountApprovalValidator, audit AccountAuditRecorder, clock func() time.Time, durable *DurableAccountServiceConfig) (*AccountService, error) {
	if repository == nil || authorizer == nil || references == nil || approval == nil || audit == nil || clock == nil {
		return nil, ErrInvalidAccountService
	}
	if binder, ok := repository.(interface{ BindAccountAuditRecorder(AccountAuditRecorder) }); ok {
		binder.BindAccountAuditRecorder(audit)
	}
	return &AccountService{repository: repository, authorizer: authorizer, references: references, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedAccountCommand)}, nil
}

func (service *ChartOfAccountsService) Execute(ctx context.Context, actor Actor, command ChartOfAccountsCommand) (ChartOfAccountsCommandResult, error) {
	if service == nil {
		return ChartOfAccountsCommandResult{}, ErrInvalidChartOfAccountsService
	}
	if err := actor.Validate(); err != nil {
		return ChartOfAccountsCommandResult{}, ErrChartOfAccountsAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return ChartOfAccountsCommandResult{}, err
	}
	fingerprint, err := chartOfAccountsCommandFingerprint(command)
	if err != nil {
		return ChartOfAccountsCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.AccountingScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		durableIdentity, err = newDurableIdentity("chart-of-accounts", actor, command.AccountingScopeID, command.IdempotencyKey)
		if err != nil {
			return ChartOfAccountsCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return ChartOfAccountsCommandResult{}, ErrChartOfAccountsIdempotencyConflict
		}
		if err != nil {
			return ChartOfAccountsCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return ChartOfAccountsCommandResult{}, ErrChartOfAccountsCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return ChartOfAccountsCommandResult{}, ErrChartOfAccountsDurableCommandFailed
			}
			result, decodeErr := decodeChartOfAccountsResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return ChartOfAccountsCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return ChartOfAccountsCommandResult{}, ErrChartOfAccountsIdempotencyConflict
			}
			result := cloneChartOfAccountsCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *ChartOfAccounts
	if command.Action == ChartOfAccountsActionUpdate {
		loaded, getErr := service.repository.GetChartOfAccounts(ctx, command.ChartOfAccountsID)
		if getErr != nil {
			service.finalizeChartOfAccountsFailure(ctx, acquisition, getErr)
			return ChartOfAccountsCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeChartOfAccountsFailure(ctx, acquisition, ErrChartOfAccountsVersionConflict)
			return ChartOfAccountsCommandResult{}, ErrChartOfAccountsVersionConflict
		}
		if current.AccountingScopeID != command.AccountingScopeID {
			service.finalizeChartOfAccountsFailure(ctx, acquisition, ErrChartOfAccountsAuthorizationDenied)
			return ChartOfAccountsCommandResult{}, ErrChartOfAccountsAuthorizationDenied
		}
	}
	decision, err := service.authorizer.AuthorizeChartOfAccounts(ctx, actor, command, current)
	if err != nil {
		service.finalizeChartOfAccountsFailure(ctx, acquisition, err)
		return ChartOfAccountsCommandResult{}, err
	}
	if err := authorizeChartOfAccountsDecision(decision, command.AccountingScopeID); err != nil {
		service.finalizeChartOfAccountsFailure(ctx, acquisition, err)
		return ChartOfAccountsCommandResult{}, err
	}
	if err := normalizeChartReferenceError(service.references.ValidateChartOfAccountsReferences(ctx, actor, command)); err != nil {
		service.finalizeChartOfAccountsFailure(ctx, acquisition, err)
		return ChartOfAccountsCommandResult{}, err
	}
	now := service.clock().UTC()
	var before, after ChartOfAccounts
	switch command.Action {
	case ChartOfAccountsActionCreate:
		before = ChartOfAccounts{}
		after, err = NewChartOfAccounts(uuid.New(), command.AccountingScopeID, command.LedgerID, command.AccountCodePolicy, command.LifecycleStatus, command.EffectiveDateFrom, command.EffectiveDateTo, command.Approval, now)
	case ChartOfAccountsActionUpdate:
		before = cloneChartOfAccounts(*current)
		after = cloneChartOfAccounts(*current)
		err = after.Replace(*current, command, now)
	}
	if err == nil && decision.ApprovalRequired && command.Approval == nil {
		err = ErrChartOfAccountsApprovalRequired
	} else if err == nil && command.Approval != nil {
		err = validateChartOfAccountsApproval(service.approval.ValidateChartOfAccountsApproval(ctx, actor, command, after), decision, command.Approval)
	}
	if err != nil {
		service.finalizeChartOfAccountsFailure(ctx, acquisition, err)
		return ChartOfAccountsCommandResult{}, err
	}
	record := ChartOfAccountsAuditRecord{ChartOfAccountsID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, AccountingScopeID: after.AccountingScopeID, LedgerID: after.LedgerID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintChartOfAccounts(before), AfterFingerprint: FingerprintChartOfAccounts(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	if command.Approval != nil {
		record.ApprovalRequestID = command.Approval.ApprovalRequestID
		record.ApprovalDecisionID = command.Approval.DecisionID
		record.ApproverUserID = command.Approval.ApproverUserID
	}
	result := ChartOfAccountsCommandResult{ChartOfAccounts: after.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus(command.Approval)}
	mutation := ChartOfAccountsMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableChartOfAccountsRepository)
		if !ok {
			service.finalizeChartOfAccountsFailure(ctx, acquisition, ErrInvalidChartOfAccountsService)
			return ChartOfAccountsCommandResult{}, ErrInvalidChartOfAccountsService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeChartOfAccountsFailure(ctx, acquisition, marshalErr)
			return ChartOfAccountsCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeChartOfAccountsFailure(ctx, acquisition, metadataErr)
			return ChartOfAccountsCommandResult{}, metadataErr
		}
		err = committer.CommitChartOfAccountsMutationWithIdempotency(ctx, mutation, DurableChartOfAccountsMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitChartOfAccountsMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeChartOfAccountsFailure(ctx, acquisition, err)
		return ChartOfAccountsCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedChartOfAccountsCommand{fingerprint: fingerprint, result: cloneChartOfAccountsCommandResult(result)}
	}
	return result, nil
}

func (service *AccountService) Execute(ctx context.Context, actor Actor, command AccountCommand) (AccountCommandResult, error) {
	if service == nil {
		return AccountCommandResult{}, ErrInvalidAccountService
	}
	if err := actor.Validate(); err != nil {
		return AccountCommandResult{}, ErrAccountAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return AccountCommandResult{}, err
	}
	fingerprint, err := accountCommandFingerprint(command)
	if err != nil {
		return AccountCommandResult{}, err
	}
	key := actor.UserID.String() + ":" + command.AccountingScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		durableIdentity, err = newDurableIdentity("account", actor, command.AccountingScopeID, command.IdempotencyKey)
		if err != nil {
			return AccountCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return AccountCommandResult{}, ErrAccountIdempotencyConflict
		}
		if err != nil {
			return AccountCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return AccountCommandResult{}, ErrAccountCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return AccountCommandResult{}, ErrAccountDurableCommandFailed
			}
			result, decodeErr := decodeAccountResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return AccountCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return AccountCommandResult{}, ErrAccountIdempotencyConflict
			}
			result := cloneAccountCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *Account
	if command.Action == AccountActionUpdate {
		loaded, getErr := service.repository.GetAccount(ctx, command.AccountID)
		if getErr != nil {
			service.finalizeAccountFailure(ctx, acquisition, getErr)
			return AccountCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeAccountFailure(ctx, acquisition, ErrAccountVersionConflict)
			return AccountCommandResult{}, ErrAccountVersionConflict
		}
		if current.AccountingScopeID != command.AccountingScopeID {
			service.finalizeAccountFailure(ctx, acquisition, ErrAccountAuthorizationDenied)
			return AccountCommandResult{}, ErrAccountAuthorizationDenied
		}
	}
	decision, err := service.authorizer.AuthorizeAccount(ctx, actor, command, current)
	if err != nil {
		service.finalizeAccountFailure(ctx, acquisition, err)
		return AccountCommandResult{}, err
	}
	if err := authorizeAccountDecision(decision, command.AccountingScopeID); err != nil {
		service.finalizeAccountFailure(ctx, acquisition, err)
		return AccountCommandResult{}, err
	}
	if err := normalizeAccountReferenceError(service.references.ValidateAccountReferences(ctx, actor, command)); err != nil {
		service.finalizeAccountFailure(ctx, acquisition, err)
		return AccountCommandResult{}, err
	}
	now := service.clock().UTC()
	var before, after Account
	switch command.Action {
	case AccountActionCreate:
		before = Account{}
		after, err = NewAccount(uuid.New(), command.AccountingScopeID, command.ChartOfAccountsID, command.AccountCode, command.AccountName, command.AccountType, command.NormalBalance, command.LifecycleStatus, command.Restrictions, command.CurrencyPolicy, command.ReportingMappings, command.EffectiveDateFrom, command.EffectiveDateTo, command.Approval, now)
	case AccountActionUpdate:
		before = cloneAccount(*current)
		after = cloneAccount(*current)
		err = after.Replace(*current, command, now)
	}
	if err == nil && decision.ApprovalRequired && command.Approval == nil {
		err = ErrAccountApprovalRequired
	} else if err == nil && command.Approval != nil {
		err = validateAccountApproval(service.approval.ValidateAccountApproval(ctx, actor, command, after), decision, command.Approval)
	}
	if err != nil {
		service.finalizeAccountFailure(ctx, acquisition, err)
		return AccountCommandResult{}, err
	}
	record := AccountAuditRecord{AccountID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, AccountingScopeID: after.AccountingScopeID, ChartOfAccountsID: after.ChartOfAccountsID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintAccount(before), AfterFingerprint: FingerprintAccount(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	if command.Approval != nil {
		record.ApprovalRequestID = command.Approval.ApprovalRequestID
		record.ApprovalDecisionID = command.Approval.DecisionID
		record.ApproverUserID = command.Approval.ApproverUserID
	}
	result := AccountCommandResult{Account: after.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus(command.Approval)}
	mutation := AccountMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableAccountRepository)
		if !ok {
			service.finalizeAccountFailure(ctx, acquisition, ErrInvalidAccountService)
			return AccountCommandResult{}, ErrInvalidAccountService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeAccountFailure(ctx, acquisition, marshalErr)
			return AccountCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeAccountFailure(ctx, acquisition, metadataErr)
			return AccountCommandResult{}, metadataErr
		}
		err = committer.CommitAccountMutationWithIdempotency(ctx, mutation, DurableAccountMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitAccountMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeAccountFailure(ctx, acquisition, err)
		return AccountCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedAccountCommand{fingerprint: fingerprint, result: cloneAccountCommandResult(result)}
	}
	return result, nil
}

func (service *ChartOfAccountsService) ListSafe(ctx context.Context, scopeID *uuid.UUID) ([]SafeChartOfAccounts, error) {
	if service == nil {
		return nil, ErrInvalidChartOfAccountsService
	}
	values, err := service.repository.ListChartsOfAccounts(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeChartOfAccounts, 0, len(values))
	for _, value := range values {
		result = append(result, value.SafeProjection())
	}
	return result, nil
}

func (service *AccountService) ListSafe(ctx context.Context, scopeID *uuid.UUID) ([]SafeAccount, error) {
	if service == nil {
		return nil, ErrInvalidAccountService
	}
	values, err := service.repository.ListAccounts(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeAccount, 0, len(values))
	for _, value := range values {
		result = append(result, value.SafeProjection())
	}
	return result, nil
}

func authorizeChartOfAccountsDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	if !decision.Allowed || decision.Permission != ChartOfAccountsManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrChartOfAccountsAuthorizationDenied
	}
	for _, allowedScopeID := range decision.ApprovedScopeIDs {
		if allowedScopeID == uuid.Nil || allowedScopeID == scopeID {
			return nil
		}
	}
	return ErrChartOfAccountsAuthorizationDenied
}

func authorizeAccountDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	if !decision.Allowed || decision.Permission != AccountManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrAccountAuthorizationDenied
	}
	for _, allowedScopeID := range decision.ApprovedScopeIDs {
		if allowedScopeID == uuid.Nil || allowedScopeID == scopeID {
			return nil
		}
	}
	return ErrAccountAuthorizationDenied
}

func normalizeChartReferenceError(err error) error {
	if err == nil || errors.Is(err, ErrChartOfAccountsReferenceInvalid) || errors.Is(err, ErrChartOfAccountsReferenceUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrChartOfAccountsReferenceInvalid, err)
}

func normalizeAccountReferenceError(err error) error {
	if err == nil || errors.Is(err, ErrAccountReferenceInvalid) || errors.Is(err, ErrAccountReferenceUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrAccountReferenceInvalid, err)
}

func validateChartOfAccountsApproval(validationErr error, decision AuthorizationDecision, approval *ApprovalDecisionReference) error {
	if decision.ApprovalRequired && approval == nil {
		return ErrChartOfAccountsApprovalRequired
	}
	if approval == nil {
		return nil
	}
	if validationErr != nil {
		if errors.Is(validationErr, ErrChartOfAccountsApprovalUnavailable) {
			return validationErr
		}
		return fmt.Errorf("%w: %v", ErrChartOfAccountsApprovalInvalid, validationErr)
	}
	return nil
}

func validateAccountApproval(validationErr error, decision AuthorizationDecision, approval *ApprovalDecisionReference) error {
	if decision.ApprovalRequired && approval == nil {
		return ErrAccountApprovalRequired
	}
	if approval == nil {
		return nil
	}
	if validationErr != nil {
		if errors.Is(validationErr, ErrAccountApprovalUnavailable) {
			return validationErr
		}
		return fmt.Errorf("%w: %v", ErrAccountApprovalInvalid, validationErr)
	}
	return nil
}

func chartOfAccountsCommandFingerprint(command ChartOfAccountsCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func accountCommandFingerprint(command AccountCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func FingerprintChartOfAccounts(chart ChartOfAccounts) string {
	data, _ := json.Marshal(chart.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func FingerprintAccount(account Account) string {
	data, _ := json.Marshal(account.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodeChartOfAccountsResult(body []byte) (ChartOfAccountsCommandResult, error) {
	var result ChartOfAccountsCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return ChartOfAccountsCommandResult{}, ErrChartOfAccountsCommandInProgress
	}
	return result, nil
}

func decodeAccountResult(body []byte) (AccountCommandResult, error) {
	var result AccountCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return AccountCommandResult{}, ErrAccountCommandInProgress
	}
	return result, nil
}

func (service *ChartOfAccountsService) finalizeChartOfAccountsFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{chartOfAccountsFailureCode(commandErr)})
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

func (service *AccountService) finalizeAccountFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{accountFailureCode(commandErr)})
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

func chartOfAccountsFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrChartOfAccountsAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrChartOfAccountsVersionConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrChartOfAccountsNotFound):
		return "CHART_OF_ACCOUNTS_NOT_FOUND"
	case errors.Is(err, ErrChartOfAccountsDuplicate):
		return "DUPLICATE_CHART_OF_ACCOUNTS"
	default:
		return "VALIDATION_FAILED"
	}
}

func accountFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrAccountAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrAccountVersionConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrAccountNotFound):
		return "ACCOUNT_NOT_FOUND"
	case errors.Is(err, ErrAccountDuplicate):
		return "DUPLICATE_ACCOUNT"
	default:
		return "VALIDATION_FAILED"
	}
}

func cloneChartOfAccounts(value ChartOfAccounts) ChartOfAccounts {
	value.EffectiveDateFrom = dateOnly(value.EffectiveDateFrom)
	value.EffectiveDateTo = cloneDate(value.EffectiveDateTo)
	value.Approval = cloneApproval(value.Approval)
	value.Revisions = cloneChartRevisions(value.Revisions)
	return value
}

func cloneAccount(value Account) Account {
	value.EffectiveDateFrom = dateOnly(value.EffectiveDateFrom)
	value.EffectiveDateTo = cloneDate(value.EffectiveDateTo)
	value.Approval = cloneApproval(value.Approval)
	value.Restrictions = cloneAccountRestrictions(value.Restrictions)
	value.ReportingMappings = cloneAccountReportingMappings(value.ReportingMappings)
	value.Revisions = cloneAccountRevisions(value.Revisions)
	return value
}

func cloneChartOfAccountsCommandResult(value ChartOfAccountsCommandResult) ChartOfAccountsCommandResult {
	value.ChartOfAccounts.EffectiveDateTo = cloneDate(value.ChartOfAccounts.EffectiveDateTo)
	return value
}

func cloneAccountCommandResult(value AccountCommandResult) AccountCommandResult {
	value.Account.EffectiveDateTo = cloneDate(value.Account.EffectiveDateTo)
	value.Account.Restrictions = cloneAccountRestrictions(value.Account.Restrictions)
	value.Account.ReportingMappings = cloneAccountReportingMappings(value.Account.ReportingMappings)
	return value
}

func sortChartsOfAccounts(values []ChartOfAccounts) {
	sort.Slice(values, func(i, j int) bool { return values[i].ID.String() < values[j].ID.String() })
}

func sortAccounts(values []Account) {
	sort.Slice(values, func(i, j int) bool { return values[i].ID.String() < values[j].ID.String() })
}
