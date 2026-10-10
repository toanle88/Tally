package gl

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

type MemoryChartOfAccountsRepository struct {
	mu      sync.RWMutex
	charts  map[uuid.UUID]ChartOfAccounts
	ledgers LedgerRepository
	audit   ChartOfAccountsAuditRecorder
}

func NewMemoryChartOfAccountsRepository(ledgers ...LedgerRepository) *MemoryChartOfAccountsRepository {
	var ledgerRepository LedgerRepository
	if len(ledgers) > 0 {
		ledgerRepository = ledgers[0]
	}
	return &MemoryChartOfAccountsRepository{charts: make(map[uuid.UUID]ChartOfAccounts), ledgers: ledgerRepository}
}

func (repository *MemoryChartOfAccountsRepository) BindChartOfAccountsAuditRecorder(audit ChartOfAccountsAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryChartOfAccountsRepository) GetChartOfAccounts(_ context.Context, id uuid.UUID) (ChartOfAccounts, error) {
	if repository == nil {
		return ChartOfAccounts{}, ErrInvalidChartOfAccountsService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	chart, ok := repository.charts[id]
	if !ok {
		return ChartOfAccounts{}, ErrChartOfAccountsNotFound
	}
	return cloneChartOfAccounts(chart), nil
}

func (repository *MemoryChartOfAccountsRepository) ListChartsOfAccounts(_ context.Context, scopeID *uuid.UUID) ([]ChartOfAccounts, error) {
	if repository == nil {
		return nil, ErrInvalidChartOfAccountsService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]ChartOfAccounts, 0, len(repository.charts))
	for _, chart := range repository.charts {
		if scopeID != nil && chart.AccountingScopeID != *scopeID {
			continue
		}
		result = append(result, cloneChartOfAccounts(chart))
	}
	sortChartsOfAccounts(result)
	return result, nil
}

func (repository *MemoryChartOfAccountsRepository) CommitChartOfAccountsMutation(ctx context.Context, mutation ChartOfAccountsMutation) error {
	if repository == nil {
		return ErrInvalidChartOfAccountsService
	}
	if err := validateChartOfAccountsMutation(mutation); err != nil {
		return err
	}
	if repository.ledgers == nil {
		return ErrChartOfAccountsReferenceUnavailable
	}
	ledger, err := repository.ledgers.GetLedger(ctx, mutation.After.LedgerID)
	if err != nil || ledger.AccountingScopeID != mutation.After.AccountingScopeID {
		return ErrChartOfAccountsReferenceInvalid
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.charts[mutation.After.ID]; exists {
			return ErrChartOfAccountsDuplicate
		}
	} else {
		current, ok := repository.charts[mutation.After.ID]
		if !ok {
			return ErrChartOfAccountsNotFound
		}
		if current.AccountingScopeID != mutation.After.AccountingScopeID || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrChartOfAccountsVersionConflict
		}
	}
	for _, existing := range repository.charts {
		if existing.ID == mutation.After.ID || !sameChartOfAccountsIdentity(existing, mutation.After) {
			continue
		}
		if rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo) {
			return ErrChartOfAccountsDuplicate
		}
	}
	if repository.audit == nil {
		return ErrChartOfAccountsAuditUnavailable
	}
	if err := repository.audit.RecordChartOfAccountsMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.charts[mutation.After.ID] = cloneChartOfAccounts(mutation.After)
	return nil
}

type MemoryAccountRepository struct {
	mu       sync.RWMutex
	accounts map[uuid.UUID]Account
	charts   ChartOfAccountsRepository
	audit    AccountAuditRecorder
}

func NewMemoryAccountRepository(charts ...ChartOfAccountsRepository) *MemoryAccountRepository {
	var chartRepository ChartOfAccountsRepository
	if len(charts) > 0 {
		chartRepository = charts[0]
	}
	return &MemoryAccountRepository{accounts: make(map[uuid.UUID]Account), charts: chartRepository}
}

func (repository *MemoryAccountRepository) BindAccountAuditRecorder(audit AccountAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryAccountRepository) GetAccount(_ context.Context, id uuid.UUID) (Account, error) {
	if repository == nil {
		return Account{}, ErrInvalidAccountService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	account, ok := repository.accounts[id]
	if !ok {
		return Account{}, ErrAccountNotFound
	}
	return cloneAccount(account), nil
}

func (repository *MemoryAccountRepository) ListAccounts(_ context.Context, scopeID *uuid.UUID) ([]Account, error) {
	if repository == nil {
		return nil, ErrInvalidAccountService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]Account, 0, len(repository.accounts))
	for _, account := range repository.accounts {
		if scopeID != nil && account.AccountingScopeID != *scopeID {
			continue
		}
		result = append(result, cloneAccount(account))
	}
	sortAccounts(result)
	return result, nil
}

func (repository *MemoryAccountRepository) CommitAccountMutation(ctx context.Context, mutation AccountMutation) error {
	if repository == nil {
		return ErrInvalidAccountService
	}
	if err := validateAccountMutation(mutation); err != nil {
		return err
	}
	if repository.charts == nil {
		return ErrAccountReferenceUnavailable
	}
	chart, err := repository.charts.GetChartOfAccounts(ctx, mutation.After.ChartOfAccountsID)
	if err != nil || chart.AccountingScopeID != mutation.After.AccountingScopeID {
		return ErrAccountReferenceInvalid
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.accounts[mutation.After.ID]; exists {
			return ErrAccountDuplicate
		}
	} else {
		current, ok := repository.accounts[mutation.After.ID]
		if !ok {
			return ErrAccountNotFound
		}
		if current.AccountingScopeID != mutation.After.AccountingScopeID || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrAccountVersionConflict
		}
	}
	for _, existing := range repository.accounts {
		if existing.ID == mutation.After.ID || !sameAccountIdentity(existing, mutation.After) {
			continue
		}
		if rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo) {
			return ErrAccountDuplicate
		}
	}
	if repository.audit == nil {
		return ErrAccountAuditUnavailable
	}
	if err := repository.audit.RecordAccountMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.accounts[mutation.After.ID] = cloneAccount(mutation.After)
	return nil
}

func validateChartOfAccountsMutation(mutation ChartOfAccountsMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 {
			return ErrChartOfAccountsVersionConflict
		}
		return nil
	}
	if mutation.ExpectedVersion == nil || mutation.Before.ID != mutation.After.ID || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrChartOfAccountsVersionConflict
	}
	return nil
}

func validateAccountMutation(mutation AccountMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 {
			return ErrAccountVersionConflict
		}
		return nil
	}
	if mutation.ExpectedVersion == nil || mutation.Before.ID != mutation.After.ID || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrAccountVersionConflict
	}
	return nil
}

func sameChartOfAccountsIdentity(left, right ChartOfAccounts) bool {
	return left.AccountingScopeID == right.AccountingScopeID && left.LedgerID == right.LedgerID && left.AccountCodePolicy == right.AccountCodePolicy
}

func sameAccountIdentity(left, right Account) bool {
	return left.AccountingScopeID == right.AccountingScopeID && left.ChartOfAccountsID == right.ChartOfAccountsID && left.AccountCode == right.AccountCode
}

type MemoryChartOfAccountsAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryChartOfAccountsAuthorizer) AuthorizeChartOfAccounts(context.Context, Actor, ChartOfAccountsCommand, *ChartOfAccounts) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryAccountAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryAccountAuthorizer) AuthorizeAccount(context.Context, Actor, AccountCommand, *Account) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllChartAccountReferenceValidator struct{}

func (AllowAllChartAccountReferenceValidator) ValidateChartOfAccountsReferences(context.Context, Actor, ChartOfAccountsCommand) error {
	return nil
}

func (AllowAllChartAccountReferenceValidator) ValidateAccountReferences(context.Context, Actor, AccountCommand) error {
	return nil
}

type MemoryChartAccountReferenceValidator struct {
	ChartError   error
	AccountError error
}

func (validator MemoryChartAccountReferenceValidator) ValidateChartOfAccountsReferences(context.Context, Actor, ChartOfAccountsCommand) error {
	return validator.ChartError
}

func (validator MemoryChartAccountReferenceValidator) ValidateAccountReferences(context.Context, Actor, AccountCommand) error {
	return validator.AccountError
}

type UnavailableChartAccountReferenceValidator struct{}

func (UnavailableChartAccountReferenceValidator) ValidateChartOfAccountsReferences(context.Context, Actor, ChartOfAccountsCommand) error {
	return ErrChartOfAccountsReferenceUnavailable
}

func (UnavailableChartAccountReferenceValidator) ValidateAccountReferences(context.Context, Actor, AccountCommand) error {
	return ErrAccountReferenceUnavailable
}

type MemoryChartAccountApprovalValidator struct {
	ChartError   error
	AccountError error
}

func (validator MemoryChartAccountApprovalValidator) ValidateChartOfAccountsApproval(context.Context, Actor, ChartOfAccountsCommand, ChartOfAccounts) error {
	return validator.ChartError
}

func (validator MemoryChartAccountApprovalValidator) ValidateAccountApproval(context.Context, Actor, AccountCommand, Account) error {
	return validator.AccountError
}

type AllowAllChartAccountApprovalValidator struct{}

func (AllowAllChartAccountApprovalValidator) ValidateChartOfAccountsApproval(context.Context, Actor, ChartOfAccountsCommand, ChartOfAccounts) error {
	return nil
}

func (AllowAllChartAccountApprovalValidator) ValidateAccountApproval(context.Context, Actor, AccountCommand, Account) error {
	return nil
}

type UnavailableChartAccountApprovalValidator struct{}

func (UnavailableChartAccountApprovalValidator) ValidateChartOfAccountsApproval(context.Context, Actor, ChartOfAccountsCommand, ChartOfAccounts) error {
	return ErrChartOfAccountsApprovalUnavailable
}

func (UnavailableChartAccountApprovalValidator) ValidateAccountApproval(context.Context, Actor, AccountCommand, Account) error {
	return ErrAccountApprovalUnavailable
}

type MemoryChartOfAccountsAuditRecorder struct {
	mu      sync.Mutex
	Records []ChartOfAccountsAuditRecord
	Err     error
}

func (recorder *MemoryChartOfAccountsAuditRecorder) RecordChartOfAccountsMutation(_ context.Context, record ChartOfAccountsAuditRecord) error {
	if recorder == nil {
		return ErrChartOfAccountsAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

type MemoryAccountAuditRecorder struct {
	mu      sync.Mutex
	Records []AccountAuditRecord
	Err     error
}

func (recorder *MemoryAccountAuditRecorder) RecordAccountMutation(_ context.Context, record AccountAuditRecord) error {
	if recorder == nil {
		return ErrAccountAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

var _ ChartOfAccountsRepository = (*MemoryChartOfAccountsRepository)(nil)
var _ AccountRepository = (*MemoryAccountRepository)(nil)
var _ ChartOfAccountsAuthorizer = MemoryChartOfAccountsAuthorizer{}
var _ AccountAuthorizer = MemoryAccountAuthorizer{}
var _ ChartAccountReferenceValidator = AllowAllChartAccountReferenceValidator{}
var _ ChartAccountApprovalValidator = AllowAllChartAccountApprovalValidator{}
var _ ChartOfAccountsAuditRecorder = (*MemoryChartOfAccountsAuditRecorder)(nil)
var _ AccountAuditRecorder = (*MemoryAccountAuditRecorder)(nil)
