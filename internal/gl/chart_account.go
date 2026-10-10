package gl

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

const (
	ChartOfAccountsManagementPermission = "finance.gl.maintain.charts.of.accounts"
	AccountManagementPermission         = "finance.gl.maintain.accounts.and.reporting.mappings"

	ChartOfAccountsActionCreate = "create"
	ChartOfAccountsActionUpdate = "update"
	AccountActionCreate         = "create"
	AccountActionUpdate         = "update"
)

var (
	ErrInvalidChartOfAccounts                  = errors.New("invalid chart of accounts")
	ErrInvalidAccount                          = errors.New("invalid account")
	ErrInvalidChartOfAccountsCommand           = errors.New("invalid chart-of-accounts command")
	ErrInvalidAccountCommand                   = errors.New("invalid account command")
	ErrChartOfAccountsNotFound                 = errors.New("chart of accounts not found")
	ErrAccountNotFound                         = errors.New("account not found")
	ErrChartOfAccountsVersionConflict          = errors.New("chart-of-accounts version conflict")
	ErrAccountVersionConflict                  = errors.New("account version conflict")
	ErrChartOfAccountsDuplicate                = errors.New("duplicate chart-of-accounts effective identity")
	ErrAccountDuplicate                        = errors.New("duplicate account effective identity")
	ErrChartOfAccountsAuthorizationDenied      = errors.New("chart-of-accounts authorization denied")
	ErrAccountAuthorizationDenied              = errors.New("account authorization denied")
	ErrChartOfAccountsAuthorizationUnavailable = errors.New("chart-of-accounts authorization unavailable")
	ErrAccountAuthorizationUnavailable         = errors.New("account authorization unavailable")
	ErrChartOfAccountsAuthorizationStale       = errors.New("chart-of-accounts authorization policy is stale")
	ErrAccountAuthorizationStale               = errors.New("account authorization policy is stale")
	ErrChartOfAccountsAuditUnavailable         = errors.New("chart-of-accounts audit unavailable")
	ErrAccountAuditUnavailable                 = errors.New("account audit unavailable")
	ErrChartOfAccountsReferenceInvalid         = errors.New("chart-of-accounts reference is invalid")
	ErrAccountReferenceInvalid                 = errors.New("account reference is invalid")
	ErrChartOfAccountsReferenceUnavailable     = errors.New("chart-of-accounts reference unavailable")
	ErrAccountReferenceUnavailable             = errors.New("account reference unavailable")
	ErrChartOfAccountsApprovalRequired         = errors.New("chart-of-accounts approval required")
	ErrAccountApprovalRequired                 = errors.New("account approval required")
	ErrChartOfAccountsApprovalInvalid          = errors.New("chart-of-accounts approval is invalid")
	ErrAccountApprovalInvalid                  = errors.New("account approval is invalid")
	ErrChartOfAccountsApprovalUnavailable      = errors.New("chart-of-accounts approval validation unavailable")
	ErrAccountApprovalUnavailable              = errors.New("account approval validation unavailable")
	ErrChartOfAccountsIdempotencyConflict      = errors.New("chart-of-accounts idempotency conflict")
	ErrAccountIdempotencyConflict              = errors.New("account idempotency conflict")
	ErrChartOfAccountsCommandInProgress        = errors.New("chart-of-accounts command is already in progress")
	ErrAccountCommandInProgress                = errors.New("account command is already in progress")
	ErrChartOfAccountsDurableCommandFailed     = errors.New("durable chart-of-accounts command failed")
	ErrAccountDurableCommandFailed             = errors.New("durable account command failed")
	ErrInvalidChartOfAccountsService           = errors.New("invalid chart-of-accounts service")
	ErrInvalidAccountService                   = errors.New("invalid account service")
)

// AccountRestriction is owned by GL. Restriction codes may reference a
// published COA policy, but GL never reads the COA schema directly.
type AccountRestriction struct {
	RestrictionCode string `json:"restrictionCode"`
	Description     string `json:"description,omitempty"`
}

func (restriction AccountRestriction) canonical() AccountRestriction {
	restriction.RestrictionCode = strings.TrimSpace(restriction.RestrictionCode)
	restriction.Description = strings.TrimSpace(restriction.Description)
	return restriction
}

func (restriction AccountRestriction) Validate() error {
	if restriction.RestrictionCode == "" || len([]rune(restriction.RestrictionCode)) > 120 {
		return errors.New("restriction code is required")
	}
	if len([]rune(restriction.Description)) > 500 {
		return errors.New("restriction description is too long")
	}
	return nil
}

// AccountReportingMapping is a GL-owned reference to an approved reporting
// definition/line. Reporting definitions remain owned by the reporting
// bounded context and are validated through an application boundary.
type AccountReportingMapping struct {
	ReportingDefinitionID uuid.UUID  `json:"reportingDefinitionId"`
	ReportingLineCode     string     `json:"reportingLineCode"`
	Approved              bool       `json:"approved"`
	EffectiveDateFrom     time.Time  `json:"effectiveDateFrom"`
	EffectiveDateTo       *time.Time `json:"effectiveDateTo,omitempty"`
}

func (mapping AccountReportingMapping) canonical() AccountReportingMapping {
	mapping.ReportingLineCode = strings.TrimSpace(mapping.ReportingLineCode)
	mapping.EffectiveDateFrom = dateOnly(mapping.EffectiveDateFrom)
	mapping.EffectiveDateTo = cloneDate(mapping.EffectiveDateTo)
	return mapping
}

func (mapping AccountReportingMapping) Validate() error {
	if mapping.ReportingDefinitionID == uuid.Nil || mapping.ReportingLineCode == "" || len([]rune(mapping.ReportingLineCode)) > 120 {
		return errors.New("reporting definition and line code are required")
	}
	if !mapping.Approved {
		return errors.New("reporting mapping must be approved")
	}
	if mapping.EffectiveDateTo != nil && mapping.EffectiveDateFrom.IsZero() {
		return errors.New("reporting mapping effective start is required when an end is supplied")
	}
	if !mapping.EffectiveDateFrom.IsZero() && mapping.EffectiveDateTo != nil && mapping.EffectiveDateTo.Before(mapping.EffectiveDateFrom) {
		return errors.New("reporting mapping effective date interval is invalid")
	}
	return nil
}

type ChartOfAccountsCommand struct {
	Action            string
	ChartOfAccountsID uuid.UUID
	AccountingScopeID uuid.UUID
	LedgerID          uuid.UUID
	AccountCodePolicy string
	LifecycleStatus   string
	EffectiveDateFrom time.Time
	EffectiveDateTo   *time.Time
	Approval          *ApprovalDecisionReference
	ExpectedVersion   *aggregateversion.AggregateVersion
	IdempotencyKey    string
	CorrelationID     string
	CausationID       string
}

func (command ChartOfAccountsCommand) Canonical() ChartOfAccountsCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.AccountCodePolicy = strings.TrimSpace(command.AccountCodePolicy)
	command.LifecycleStatus = strings.ToLower(strings.TrimSpace(command.LifecycleStatus))
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval = cloneApproval(command.Approval)
	return command
}

func (command ChartOfAccountsCommand) Validate() error {
	if command.Action != ChartOfAccountsActionCreate && command.Action != ChartOfAccountsActionUpdate {
		return fmt.Errorf("%w: unsupported action", ErrInvalidChartOfAccountsCommand)
	}
	if command.AccountingScopeID == uuid.Nil || command.LedgerID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope and ledger are required", ErrInvalidChartOfAccountsCommand)
	}
	if command.AccountCodePolicy == "" || len([]rune(command.AccountCodePolicy)) > 120 {
		return fmt.Errorf("%w: account-code policy is required", ErrInvalidChartOfAccountsCommand)
	}
	if command.EffectiveDateFrom.IsZero() || command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidChartOfAccountsCommand)
	}
	if !validLifecycleStatus(command.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidChartOfAccountsCommand)
	}
	if command.Action == ChartOfAccountsActionCreate {
		if command.ChartOfAccountsID != uuid.Nil || command.ExpectedVersion != nil || command.LifecycleStatus == LedgerStatusRetired {
			return fmt.Errorf("%w: create cannot include an id, expected version, or retired status", ErrInvalidChartOfAccountsCommand)
		}
	} else if command.ChartOfAccountsID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
		return fmt.Errorf("%w: update requires an id and valid expected version", ErrInvalidChartOfAccountsCommand)
	}
	if command.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidChartOfAccountsCommand)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidChartOfAccountsCommand, err)
		}
	}
	return nil
}

type AccountCommand struct {
	Action            string
	AccountID         uuid.UUID
	AccountingScopeID uuid.UUID
	ChartOfAccountsID uuid.UUID
	AccountCode       string
	AccountName       string
	AccountType       string
	NormalBalance     string
	LifecycleStatus   string
	Restrictions      []AccountRestriction
	CurrencyPolicy    string
	ReportingMappings []AccountReportingMapping
	EffectiveDateFrom time.Time
	EffectiveDateTo   *time.Time
	Approval          *ApprovalDecisionReference
	ExpectedVersion   *aggregateversion.AggregateVersion
	IdempotencyKey    string
	CorrelationID     string
	CausationID       string
}

func (command AccountCommand) Canonical() AccountCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.AccountCode = strings.TrimSpace(command.AccountCode)
	command.AccountName = strings.TrimSpace(command.AccountName)
	command.AccountType = strings.ToLower(strings.TrimSpace(command.AccountType))
	command.NormalBalance = strings.ToLower(strings.TrimSpace(command.NormalBalance))
	command.LifecycleStatus = strings.ToLower(strings.TrimSpace(command.LifecycleStatus))
	command.CurrencyPolicy = strings.TrimSpace(command.CurrencyPolicy)
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	command.Restrictions = cloneAccountRestrictions(command.Restrictions)
	command.ReportingMappings = cloneAccountReportingMappings(command.ReportingMappings)
	for index := range command.Restrictions {
		command.Restrictions[index] = command.Restrictions[index].canonical()
	}
	for index := range command.ReportingMappings {
		command.ReportingMappings[index] = command.ReportingMappings[index].canonical()
	}
	command.IdempotencyKey = strings.TrimSpace(command.IdempotencyKey)
	command.CorrelationID = strings.TrimSpace(command.CorrelationID)
	command.CausationID = strings.TrimSpace(command.CausationID)
	command.Approval = cloneApproval(command.Approval)
	return command
}

func (command AccountCommand) Validate() error {
	if command.Action != AccountActionCreate && command.Action != AccountActionUpdate {
		return fmt.Errorf("%w: unsupported action", ErrInvalidAccountCommand)
	}
	if command.AccountingScopeID == uuid.Nil || command.ChartOfAccountsID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope and chart of accounts are required", ErrInvalidAccountCommand)
	}
	if command.AccountCode == "" || len([]rune(command.AccountCode)) > 80 || command.AccountName == "" || len([]rune(command.AccountName)) > 240 || command.AccountType == "" || len([]rune(command.AccountType)) > 80 {
		return fmt.Errorf("%w: account code, name, and type are required", ErrInvalidAccountCommand)
	}
	if command.NormalBalance != "debit" && command.NormalBalance != "credit" {
		return fmt.Errorf("%w: normal balance must be debit or credit", ErrInvalidAccountCommand)
	}
	if expected, known := normalBalanceForAccountType(command.AccountType); known && expected != command.NormalBalance {
		return fmt.Errorf("%w: account type and normal balance are incompatible", ErrInvalidAccountCommand)
	}
	if command.CurrencyPolicy == "" || len([]rune(command.CurrencyPolicy)) > 120 {
		return fmt.Errorf("%w: currency policy is required", ErrInvalidAccountCommand)
	}
	if command.EffectiveDateFrom.IsZero() || command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidAccountCommand)
	}
	if !validLifecycleStatus(command.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidAccountCommand)
	}
	if command.Action == AccountActionCreate {
		if command.AccountID != uuid.Nil || command.ExpectedVersion != nil || command.LifecycleStatus == LedgerStatusRetired {
			return fmt.Errorf("%w: create cannot include an id, expected version, or retired status", ErrInvalidAccountCommand)
		}
	} else if command.AccountID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
		return fmt.Errorf("%w: update requires an id and valid expected version", ErrInvalidAccountCommand)
	}
	if command.IdempotencyKey == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidAccountCommand)
	}
	seenRestrictions := make(map[string]struct{}, len(command.Restrictions))
	for _, restriction := range command.Restrictions {
		if err := restriction.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccountCommand, err)
		}
		key := strings.ToLower(restriction.RestrictionCode)
		if _, exists := seenRestrictions[key]; exists {
			return fmt.Errorf("%w: duplicate restriction code", ErrInvalidAccountCommand)
		}
		seenRestrictions[key] = struct{}{}
	}
	seenMappings := make([]AccountReportingMapping, 0, len(command.ReportingMappings))
	for _, mapping := range command.ReportingMappings {
		if err := mapping.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccountCommand, err)
		}
		for _, existing := range seenMappings {
			if existing.ReportingDefinitionID == mapping.ReportingDefinitionID && existing.ReportingLineCode == mapping.ReportingLineCode && rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mapping.EffectiveDateFrom, mapping.EffectiveDateTo) {
				return fmt.Errorf("%w: overlapping reporting mapping", ErrInvalidAccountCommand)
			}
		}
		seenMappings = append(seenMappings, mapping)
	}
	if command.Approval != nil {
		if err := command.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccountCommand, err)
		}
	}
	return nil
}

type ChartOfAccounts struct {
	ID                 uuid.UUID
	AccountingScopeID  uuid.UUID
	LedgerID           uuid.UUID
	AccountCodePolicy  string
	LifecycleStatus    string
	EffectiveDateFrom  time.Time
	EffectiveDateTo    *time.Time
	Approval           *ApprovalDecisionReference
	Version            aggregateversion.AggregateVersion
	RevisionNumber     int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastAuditReference uuid.UUID
	Revisions          []ChartOfAccountsRevision
}

type ChartOfAccountsRevision struct {
	RevisionNumber int64
	Version        aggregateversion.AggregateVersion
	Snapshot       ChartOfAccountsSnapshot
	CreatedAt      time.Time
}

type ChartOfAccountsSnapshot struct {
	AccountingScopeID uuid.UUID                  `json:"accountingScopeId"`
	LedgerID          uuid.UUID                  `json:"ledgerId"`
	AccountCodePolicy string                     `json:"accountCodePolicy"`
	LifecycleStatus   string                     `json:"lifecycleStatus"`
	EffectiveDateFrom time.Time                  `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time                 `json:"effectiveDateTo,omitempty"`
	Approval          *ApprovalDecisionReference `json:"approval,omitempty"`
}

func (chart ChartOfAccounts) Snapshot() ChartOfAccountsSnapshot {
	return ChartOfAccountsSnapshot{AccountingScopeID: chart.AccountingScopeID, LedgerID: chart.LedgerID, AccountCodePolicy: chart.AccountCodePolicy, LifecycleStatus: chart.LifecycleStatus, EffectiveDateFrom: chart.EffectiveDateFrom, EffectiveDateTo: cloneDate(chart.EffectiveDateTo), Approval: cloneApproval(chart.Approval)}
}

func (chart ChartOfAccounts) Validate() error {
	if chart.ID == uuid.Nil || chart.AccountingScopeID == uuid.Nil || chart.LedgerID == uuid.Nil || chart.Version.Value() < 1 || chart.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidChartOfAccounts)
	}
	if chart.AccountCodePolicy == "" || len([]rune(chart.AccountCodePolicy)) > 120 {
		return fmt.Errorf("%w: account-code policy is required", ErrInvalidChartOfAccounts)
	}
	if !validLifecycleStatus(chart.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidChartOfAccounts)
	}
	if chart.EffectiveDateFrom.IsZero() || chart.EffectiveDateTo != nil && chart.EffectiveDateTo.Before(chart.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidChartOfAccounts)
	}
	if chart.CreatedAt.IsZero() || chart.UpdatedAt.IsZero() || chart.UpdatedAt.Before(chart.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidChartOfAccounts)
	}
	if chart.Approval != nil {
		if err := chart.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidChartOfAccounts, err)
		}
	}
	return nil
}

func NewChartOfAccounts(id, scopeID, ledgerID uuid.UUID, accountCodePolicy, status string, from time.Time, to *time.Time, approval *ApprovalDecisionReference, now time.Time) (ChartOfAccounts, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == LedgerStatusRetired {
		return ChartOfAccounts{}, fmt.Errorf("%w: a new chart cannot start retired", ErrInvalidChartOfAccounts)
	}
	now = now.UTC()
	chart := ChartOfAccounts{ID: id, AccountingScopeID: scopeID, LedgerID: ledgerID, AccountCodePolicy: strings.TrimSpace(accountCodePolicy), LifecycleStatus: status, EffectiveDateFrom: dateOnly(from), EffectiveDateTo: cloneDate(to), Approval: cloneApproval(approval), Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now, UpdatedAt: now}
	if err := chart.Validate(); err != nil {
		return ChartOfAccounts{}, err
	}
	chart.Revisions = []ChartOfAccountsRevision{{RevisionNumber: 1, Version: chart.Version, Snapshot: chart.Snapshot(), CreatedAt: now}}
	return chart, nil
}

func (chart *ChartOfAccounts) Replace(current ChartOfAccounts, command ChartOfAccountsCommand, now time.Time) error {
	if chart == nil || current.ID == uuid.Nil || chart.ID != current.ID {
		return ErrInvalidChartOfAccounts
	}
	if current.LifecycleStatus == LedgerStatusRetired {
		return fmt.Errorf("%w: a retired chart cannot be maintained", ErrInvalidChartOfAccounts)
	}
	if current.LedgerID != command.LedgerID || current.AccountingScopeID != command.AccountingScopeID {
		return ErrChartOfAccountsReferenceInvalid
	}
	if !validLifecycleTransition(current.LifecycleStatus, command.LifecycleStatus) {
		return fmt.Errorf("%w: lifecycle transition from %s to %s is not allowed", ErrInvalidChartOfAccounts, current.LifecycleStatus, command.LifecycleStatus)
	}
	next, err := current.Version.Advance()
	if err != nil {
		return err
	}
	*chart = ChartOfAccounts{ID: current.ID, AccountingScopeID: current.AccountingScopeID, LedgerID: current.LedgerID, AccountCodePolicy: strings.TrimSpace(command.AccountCodePolicy), LifecycleStatus: command.LifecycleStatus, EffectiveDateFrom: dateOnly(command.EffectiveDateFrom), EffectiveDateTo: cloneDate(command.EffectiveDateTo), Approval: cloneApproval(command.Approval), Version: next, RevisionNumber: current.RevisionNumber + 1, CreatedAt: current.CreatedAt, UpdatedAt: now.UTC(), LastAuditReference: current.LastAuditReference}
	if err := chart.Validate(); err != nil {
		return err
	}
	chart.Revisions = append(cloneChartRevisions(current.Revisions), ChartOfAccountsRevision{RevisionNumber: chart.RevisionNumber, Version: chart.Version, Snapshot: chart.Snapshot(), CreatedAt: chart.UpdatedAt})
	return nil
}

type SafeChartOfAccounts struct {
	ID                uuid.UUID                         `json:"id"`
	AccountingScopeID uuid.UUID                         `json:"accountingScopeId"`
	LedgerID          uuid.UUID                         `json:"ledgerId"`
	AccountCodePolicy string                            `json:"accountCodePolicy"`
	LifecycleStatus   string                            `json:"lifecycleStatus"`
	EffectiveDateFrom time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time                        `json:"effectiveDateTo,omitempty"`
	ApprovalStatus    string                            `json:"approvalStatus"`
	ValidationOutcome string                            `json:"validationOutcome"`
	NextAction        string                            `json:"nextAction"`
	Version           aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber    int64                             `json:"revisionNumber"`
}

func (chart ChartOfAccounts) SafeProjection() SafeChartOfAccounts {
	nextAction := "maintain"
	if chart.LifecycleStatus == LedgerStatusRetired {
		nextAction = "view history"
	}
	return SafeChartOfAccounts{ID: chart.ID, AccountingScopeID: chart.AccountingScopeID, LedgerID: chart.LedgerID, AccountCodePolicy: chart.AccountCodePolicy, LifecycleStatus: chart.LifecycleStatus, EffectiveDateFrom: chart.EffectiveDateFrom, EffectiveDateTo: cloneDate(chart.EffectiveDateTo), ApprovalStatus: approvalStatus(chart.Approval), ValidationOutcome: "valid", NextAction: nextAction, Version: chart.Version, RevisionNumber: chart.RevisionNumber}
}

type Account struct {
	ID                 uuid.UUID
	AccountingScopeID  uuid.UUID
	ChartOfAccountsID  uuid.UUID
	AccountCode        string
	AccountName        string
	AccountType        string
	NormalBalance      string
	LifecycleStatus    string
	Restrictions       []AccountRestriction
	CurrencyPolicy     string
	ReportingMappings  []AccountReportingMapping
	EffectiveDateFrom  time.Time
	EffectiveDateTo    *time.Time
	Approval           *ApprovalDecisionReference
	Version            aggregateversion.AggregateVersion
	RevisionNumber     int64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	LastAuditReference uuid.UUID
	Revisions          []AccountRevision
}

type AccountRevision struct {
	RevisionNumber int64
	Version        aggregateversion.AggregateVersion
	Snapshot       AccountSnapshot
	CreatedAt      time.Time
}

type AccountSnapshot struct {
	AccountingScopeID uuid.UUID                  `json:"accountingScopeId"`
	ChartOfAccountsID uuid.UUID                  `json:"chartOfAccountsId"`
	AccountCode       string                     `json:"accountCode"`
	AccountName       string                     `json:"accountName"`
	AccountType       string                     `json:"accountType"`
	NormalBalance     string                     `json:"normalBalance"`
	LifecycleStatus   string                     `json:"lifecycleStatus"`
	Restrictions      []AccountRestriction       `json:"restrictions,omitempty"`
	CurrencyPolicy    string                     `json:"currencyPolicy"`
	ReportingMappings []AccountReportingMapping  `json:"reportingMappings,omitempty"`
	EffectiveDateFrom time.Time                  `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time                 `json:"effectiveDateTo,omitempty"`
	Approval          *ApprovalDecisionReference `json:"approval,omitempty"`
}

func (account Account) Snapshot() AccountSnapshot {
	return AccountSnapshot{AccountingScopeID: account.AccountingScopeID, ChartOfAccountsID: account.ChartOfAccountsID, AccountCode: account.AccountCode, AccountName: account.AccountName, AccountType: account.AccountType, NormalBalance: account.NormalBalance, LifecycleStatus: account.LifecycleStatus, Restrictions: cloneAccountRestrictions(account.Restrictions), CurrencyPolicy: account.CurrencyPolicy, ReportingMappings: cloneAccountReportingMappings(account.ReportingMappings), EffectiveDateFrom: account.EffectiveDateFrom, EffectiveDateTo: cloneDate(account.EffectiveDateTo), Approval: cloneApproval(account.Approval)}
}

func (account Account) Validate() error {
	if account.ID == uuid.Nil || account.AccountingScopeID == uuid.Nil || account.ChartOfAccountsID == uuid.Nil || account.Version.Value() < 1 || account.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidAccount)
	}
	if account.AccountCode == "" || len([]rune(account.AccountCode)) > 80 || account.AccountName == "" || len([]rune(account.AccountName)) > 240 || account.AccountType == "" || len([]rune(account.AccountType)) > 80 {
		return fmt.Errorf("%w: account code, name, and type are required", ErrInvalidAccount)
	}
	if account.NormalBalance != "debit" && account.NormalBalance != "credit" {
		return fmt.Errorf("%w: normal balance must be debit or credit", ErrInvalidAccount)
	}
	if expected, known := normalBalanceForAccountType(account.AccountType); known && expected != account.NormalBalance {
		return fmt.Errorf("%w: account type and normal balance are incompatible", ErrInvalidAccount)
	}
	if account.CurrencyPolicy == "" || len([]rune(account.CurrencyPolicy)) > 120 {
		return fmt.Errorf("%w: currency policy is required", ErrInvalidAccount)
	}
	if !validLifecycleStatus(account.LifecycleStatus) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidAccount)
	}
	if account.EffectiveDateFrom.IsZero() || account.EffectiveDateTo != nil && account.EffectiveDateTo.Before(account.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date interval is invalid", ErrInvalidAccount)
	}
	if account.CreatedAt.IsZero() || account.UpdatedAt.IsZero() || account.UpdatedAt.Before(account.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidAccount)
	}
	seenRestrictions := make(map[string]struct{}, len(account.Restrictions))
	for _, restriction := range account.Restrictions {
		if err := restriction.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccount, err)
		}
		key := strings.ToLower(restriction.RestrictionCode)
		if _, exists := seenRestrictions[key]; exists {
			return fmt.Errorf("%w: duplicate restriction code", ErrInvalidAccount)
		}
		seenRestrictions[key] = struct{}{}
	}
	seenMappings := make([]AccountReportingMapping, 0, len(account.ReportingMappings))
	for _, mapping := range account.ReportingMappings {
		if err := mapping.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccount, err)
		}
		for _, existing := range seenMappings {
			if existing.ReportingDefinitionID == mapping.ReportingDefinitionID && existing.ReportingLineCode == mapping.ReportingLineCode && rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mapping.EffectiveDateFrom, mapping.EffectiveDateTo) {
				return fmt.Errorf("%w: overlapping reporting mapping", ErrInvalidAccount)
			}
		}
		seenMappings = append(seenMappings, mapping)
	}
	if account.Approval != nil {
		if err := account.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: %v", ErrInvalidAccount, err)
		}
	}
	return nil
}

func NewAccount(id, scopeID, chartID uuid.UUID, code, name, accountType, normalBalance, status string, restrictions []AccountRestriction, currencyPolicy string, mappings []AccountReportingMapping, from time.Time, to *time.Time, approval *ApprovalDecisionReference, now time.Time) (Account, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status == LedgerStatusRetired {
		return Account{}, fmt.Errorf("%w: a new account cannot start retired", ErrInvalidAccount)
	}
	now = now.UTC()
	from = dateOnly(from)
	to = cloneDate(to)
	account := Account{ID: id, AccountingScopeID: scopeID, ChartOfAccountsID: chartID, AccountCode: strings.TrimSpace(code), AccountName: strings.TrimSpace(name), AccountType: strings.ToLower(strings.TrimSpace(accountType)), NormalBalance: strings.ToLower(strings.TrimSpace(normalBalance)), LifecycleStatus: status, Restrictions: cloneAccountRestrictions(restrictions), CurrencyPolicy: strings.TrimSpace(currencyPolicy), ReportingMappings: normalizeAccountReportingMappings(mappings, from, to), EffectiveDateFrom: from, EffectiveDateTo: to, Approval: cloneApproval(approval), Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now, UpdatedAt: now}
	for index := range account.Restrictions {
		account.Restrictions[index] = account.Restrictions[index].canonical()
	}
	if err := account.Validate(); err != nil {
		return Account{}, err
	}
	account.Revisions = []AccountRevision{{RevisionNumber: 1, Version: account.Version, Snapshot: account.Snapshot(), CreatedAt: now}}
	return account, nil
}

func (account *Account) Replace(current Account, command AccountCommand, now time.Time) error {
	if account == nil || current.ID == uuid.Nil || account.ID != current.ID {
		return ErrInvalidAccount
	}
	if current.LifecycleStatus == LedgerStatusRetired {
		return fmt.Errorf("%w: a retired account cannot be maintained", ErrInvalidAccount)
	}
	if current.ChartOfAccountsID != command.ChartOfAccountsID || current.AccountingScopeID != command.AccountingScopeID {
		return ErrAccountReferenceInvalid
	}
	if !validLifecycleTransition(current.LifecycleStatus, command.LifecycleStatus) {
		return fmt.Errorf("%w: lifecycle transition from %s to %s is not allowed", ErrInvalidAccount, current.LifecycleStatus, command.LifecycleStatus)
	}
	next, err := current.Version.Advance()
	if err != nil {
		return err
	}
	*account = Account{ID: current.ID, AccountingScopeID: current.AccountingScopeID, ChartOfAccountsID: current.ChartOfAccountsID, AccountCode: strings.TrimSpace(command.AccountCode), AccountName: strings.TrimSpace(command.AccountName), AccountType: strings.ToLower(strings.TrimSpace(command.AccountType)), NormalBalance: strings.ToLower(strings.TrimSpace(command.NormalBalance)), LifecycleStatus: command.LifecycleStatus, Restrictions: cloneAccountRestrictions(command.Restrictions), CurrencyPolicy: strings.TrimSpace(command.CurrencyPolicy), ReportingMappings: normalizeAccountReportingMappings(command.ReportingMappings, dateOnly(command.EffectiveDateFrom), command.EffectiveDateTo), EffectiveDateFrom: dateOnly(command.EffectiveDateFrom), EffectiveDateTo: cloneDate(command.EffectiveDateTo), Approval: cloneApproval(command.Approval), Version: next, RevisionNumber: current.RevisionNumber + 1, CreatedAt: current.CreatedAt, UpdatedAt: now.UTC(), LastAuditReference: current.LastAuditReference}
	for index := range account.Restrictions {
		account.Restrictions[index] = account.Restrictions[index].canonical()
	}
	if err := account.Validate(); err != nil {
		return err
	}
	account.Revisions = append(cloneAccountRevisions(current.Revisions), AccountRevision{RevisionNumber: account.RevisionNumber, Version: account.Version, Snapshot: account.Snapshot(), CreatedAt: account.UpdatedAt})
	return nil
}

type SafeAccount struct {
	ID                uuid.UUID                         `json:"id"`
	AccountingScopeID uuid.UUID                         `json:"accountingScopeId"`
	ChartOfAccountsID uuid.UUID                         `json:"chartOfAccountsId"`
	AccountCode       string                            `json:"accountCode"`
	AccountName       string                            `json:"accountName"`
	AccountType       string                            `json:"accountType"`
	NormalBalance     string                            `json:"normalBalance"`
	LifecycleStatus   string                            `json:"lifecycleStatus"`
	Restrictions      []AccountRestriction              `json:"restrictions,omitempty"`
	CurrencyPolicy    string                            `json:"currencyPolicy"`
	ReportingMappings []AccountReportingMapping         `json:"reportingMappings,omitempty"`
	EffectiveDateFrom time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time                        `json:"effectiveDateTo,omitempty"`
	ApprovalStatus    string                            `json:"approvalStatus"`
	ValidationOutcome string                            `json:"validationOutcome"`
	NextAction        string                            `json:"nextAction"`
	Version           aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber    int64                             `json:"revisionNumber"`
}

func (account Account) SafeProjection() SafeAccount {
	nextAction := "maintain"
	if account.LifecycleStatus == LedgerStatusRetired {
		nextAction = "view history"
	}
	return SafeAccount{ID: account.ID, AccountingScopeID: account.AccountingScopeID, ChartOfAccountsID: account.ChartOfAccountsID, AccountCode: account.AccountCode, AccountName: account.AccountName, AccountType: account.AccountType, NormalBalance: account.NormalBalance, LifecycleStatus: account.LifecycleStatus, Restrictions: cloneAccountRestrictions(account.Restrictions), CurrencyPolicy: account.CurrencyPolicy, ReportingMappings: cloneAccountReportingMappings(account.ReportingMappings), EffectiveDateFrom: account.EffectiveDateFrom, EffectiveDateTo: cloneDate(account.EffectiveDateTo), ApprovalStatus: approvalStatus(account.Approval), ValidationOutcome: "valid", NextAction: nextAction, Version: account.Version, RevisionNumber: account.RevisionNumber}
}

func normalBalanceForAccountType(accountType string) (string, bool) {
	switch strings.NewReplacer("-", "", "_", "", " ", "").Replace(strings.ToLower(strings.TrimSpace(accountType))) {
	case "asset", "assets", "expense", "expenses":
		return "debit", true
	case "liability", "liabilities", "equity", "revenue", "revenues", "income":
		return "credit", true
	default:
		return "", false
	}
}

func normalizeAccountReportingMappings(values []AccountReportingMapping, from time.Time, to *time.Time) []AccountReportingMapping {
	result := cloneAccountReportingMappings(values)
	for index := range result {
		result[index] = result[index].canonical()
		if result[index].EffectiveDateFrom.IsZero() {
			result[index].EffectiveDateFrom = from
		}
		if result[index].EffectiveDateTo == nil && to != nil {
			result[index].EffectiveDateTo = cloneDate(to)
		}
	}
	return result
}

func cloneAccountRestrictions(values []AccountRestriction) []AccountRestriction {
	if values == nil {
		return nil
	}
	result := make([]AccountRestriction, len(values))
	copy(result, values)
	return result
}

func cloneAccountReportingMappings(values []AccountReportingMapping) []AccountReportingMapping {
	if values == nil {
		return nil
	}
	result := make([]AccountReportingMapping, len(values))
	for index, value := range values {
		result[index] = value.canonical()
	}
	return result
}

func cloneChartRevisions(values []ChartOfAccountsRevision) []ChartOfAccountsRevision {
	if values == nil {
		return nil
	}
	result := make([]ChartOfAccountsRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.EffectiveDateTo = cloneDate(result[index].Snapshot.EffectiveDateTo)
		result[index].Snapshot.Approval = cloneApproval(result[index].Snapshot.Approval)
	}
	return result
}

func cloneAccountRevisions(values []AccountRevision) []AccountRevision {
	if values == nil {
		return nil
	}
	result := make([]AccountRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.Restrictions = cloneAccountRestrictions(result[index].Snapshot.Restrictions)
		result[index].Snapshot.ReportingMappings = cloneAccountReportingMappings(result[index].Snapshot.ReportingMappings)
		result[index].Snapshot.EffectiveDateTo = cloneDate(result[index].Snapshot.EffectiveDateTo)
		result[index].Snapshot.Approval = cloneApproval(result[index].Snapshot.Approval)
	}
	return result
}
