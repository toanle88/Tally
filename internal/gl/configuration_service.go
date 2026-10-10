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

type LedgerAuthorizer interface {
	AuthorizeLedger(context.Context, Actor, LedgerCommand, *Ledger) (AuthorizationDecision, error)
}

type AccountingBookAuthorizer interface {
	AuthorizeAccountingBook(context.Context, Actor, AccountingBookCommand, *AccountingBook) (AuthorizationDecision, error)
}

type ReferenceValidator interface {
	ValidateLedgerReferences(context.Context, Actor, LedgerCommand) error
	ValidateAccountingBookReferences(context.Context, Actor, AccountingBookCommand) error
}

type ApprovalValidator interface {
	ValidateLedgerApproval(context.Context, Actor, LedgerCommand, Ledger) error
	ValidateAccountingBookApproval(context.Context, Actor, AccountingBookCommand, AccountingBook) error
}

type LedgerAuditRecord struct {
	LedgerID              uuid.UUID
	ApprovalRequestID     uuid.UUID
	ApprovalDecisionID    uuid.UUID
	ApproverUserID        uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	AccountingScopeID     uuid.UUID
	LegalEntityID         uuid.UUID
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

type AccountingBookAuditRecord struct {
	AccountingBookID      uuid.UUID
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

type LedgerAuditRecorder interface {
	RecordLedgerMutation(context.Context, LedgerAuditRecord) error
}

type AccountingBookAuditRecorder interface {
	RecordAccountingBookMutation(context.Context, AccountingBookAuditRecord) error
}

type LedgerMutation struct {
	Before          Ledger
	After           Ledger
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           LedgerAuditRecord
}

type AccountingBookMutation struct {
	Before          AccountingBook
	After           AccountingBook
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           AccountingBookAuditRecord
}

type LedgerRepository interface {
	GetLedger(context.Context, uuid.UUID) (Ledger, error)
	ListLedgers(context.Context, *uuid.UUID) ([]Ledger, error)
	CommitLedgerMutation(context.Context, LedgerMutation) error
}

type AccountingBookRepository interface {
	GetAccountingBook(context.Context, uuid.UUID) (AccountingBook, error)
	ListAccountingBooks(context.Context, *uuid.UUID) ([]AccountingBook, error)
	CommitAccountingBookMutation(context.Context, AccountingBookMutation) error
}

type DurableLedgerRepository interface {
	CommitLedgerMutationWithIdempotency(context.Context, LedgerMutation, DurableLedgerMutationCommit) error
}

type DurableAccountingBookRepository interface {
	CommitAccountingBookMutationWithIdempotency(context.Context, AccountingBookMutation, DurableAccountingBookMutationCommit) error
}

type DurableLedgerMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableAccountingBookMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableLedgerServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableAccountingBookServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type LedgerCommandResult struct {
	Ledger            SafeLedger `json:"ledger"`
	DecisionReference uuid.UUID  `json:"decisionReference"`
	PolicyReference   string     `json:"policyReference"`
	ValidationOutcome string     `json:"validationOutcome"`
	ApprovalStatus    string     `json:"approvalStatus"`
	Replayed          bool       `json:"replayed,omitempty"`
}

type AccountingBookCommandResult struct {
	AccountingBook    SafeAccountingBook `json:"accountingBook"`
	DecisionReference uuid.UUID          `json:"decisionReference"`
	PolicyReference   string             `json:"policyReference"`
	ValidationOutcome string             `json:"validationOutcome"`
	ApprovalStatus    string             `json:"approvalStatus"`
	Replayed          bool               `json:"replayed,omitempty"`
}

type LedgerService struct {
	repository  LedgerRepository
	authorizer  LedgerAuthorizer
	references  ReferenceValidator
	approval    ApprovalValidator
	audit       LedgerAuditRecorder
	clock       func() time.Time
	durable     *DurableLedgerServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedLedgerCommand
}

type AccountingBookService struct {
	repository  AccountingBookRepository
	authorizer  AccountingBookAuthorizer
	references  ReferenceValidator
	approval    ApprovalValidator
	audit       AccountingBookAuditRecorder
	clock       func() time.Time
	durable     *DurableAccountingBookServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedAccountingBookCommand
}

type storedLedgerCommand struct {
	fingerprint string
	result      LedgerCommandResult
}

type storedAccountingBookCommand struct {
	fingerprint string
	result      AccountingBookCommandResult
}

func NewLedgerService(repository LedgerRepository, authorizer LedgerAuthorizer, references ReferenceValidator, approval ApprovalValidator, audit LedgerAuditRecorder, clock func() time.Time) (*LedgerService, error) {
	return newLedgerService(repository, authorizer, references, approval, audit, clock, nil)
}

func NewLedgerServiceWithDurableIdempotency(repository LedgerRepository, authorizer LedgerAuthorizer, references ReferenceValidator, approval ApprovalValidator, audit LedgerAuditRecorder, clock func() time.Time, durable DurableLedgerServiceConfig) (*LedgerService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidLedgerService
	}
	return newLedgerService(repository, authorizer, references, approval, audit, clock, &durable)
}

func newLedgerService(repository LedgerRepository, authorizer LedgerAuthorizer, references ReferenceValidator, approval ApprovalValidator, audit LedgerAuditRecorder, clock func() time.Time, durable *DurableLedgerServiceConfig) (*LedgerService, error) {
	if repository == nil || authorizer == nil || references == nil || approval == nil || audit == nil || clock == nil {
		if references == nil {
			return nil, ErrInvalidReferenceValidator
		}
		return nil, ErrInvalidLedgerService
	}
	if binder, ok := repository.(interface{ BindLedgerAuditRecorder(LedgerAuditRecorder) }); ok {
		binder.BindLedgerAuditRecorder(audit)
	}
	return &LedgerService{repository: repository, authorizer: authorizer, references: references, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedLedgerCommand)}, nil
}

func NewAccountingBookService(repository AccountingBookRepository, authorizer AccountingBookAuthorizer, references ReferenceValidator, approval ApprovalValidator, audit AccountingBookAuditRecorder, clock func() time.Time) (*AccountingBookService, error) {
	return newAccountingBookService(repository, authorizer, references, approval, audit, clock, nil)
}

func NewAccountingBookServiceWithDurableIdempotency(repository AccountingBookRepository, authorizer AccountingBookAuthorizer, references ReferenceValidator, approval ApprovalValidator, audit AccountingBookAuditRecorder, clock func() time.Time, durable DurableAccountingBookServiceConfig) (*AccountingBookService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidAccountingBookService
	}
	return newAccountingBookService(repository, authorizer, references, approval, audit, clock, &durable)
}

func newAccountingBookService(repository AccountingBookRepository, authorizer AccountingBookAuthorizer, references ReferenceValidator, approval ApprovalValidator, audit AccountingBookAuditRecorder, clock func() time.Time, durable *DurableAccountingBookServiceConfig) (*AccountingBookService, error) {
	if repository == nil || authorizer == nil || references == nil || approval == nil || audit == nil || clock == nil {
		if references == nil {
			return nil, ErrInvalidReferenceValidator
		}
		return nil, ErrInvalidAccountingBookService
	}
	if binder, ok := repository.(interface {
		BindAccountingBookAuditRecorder(AccountingBookAuditRecorder)
	}); ok {
		binder.BindAccountingBookAuditRecorder(audit)
	}
	return &AccountingBookService{repository: repository, authorizer: authorizer, references: references, approval: approval, audit: audit, clock: clock, durable: durable, idempotency: make(map[string]storedAccountingBookCommand)}, nil
}

func (service *LedgerService) Execute(ctx context.Context, actor Actor, command LedgerCommand) (LedgerCommandResult, error) {
	if service == nil {
		return LedgerCommandResult{}, ErrInvalidLedgerService
	}
	if err := actor.Validate(); err != nil {
		return LedgerCommandResult{}, err
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return LedgerCommandResult{}, err
	}
	fingerprint, err := ledgerCommandFingerprint(command)
	if err != nil {
		return LedgerCommandResult{}, err
	}

	key := actor.UserID.String() + ":" + command.AccountingScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		durableIdentity, err = newDurableIdentity("ledger", actor, command.AccountingScopeID, command.IdempotencyKey)
		if err != nil {
			return LedgerCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return LedgerCommandResult{}, ErrLedgerIdempotencyConflict
		}
		if err != nil {
			return LedgerCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return LedgerCommandResult{}, ErrLedgerCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return LedgerCommandResult{}, ErrLedgerDurableCommandFailed
			}
			result, decodeErr := decodeLedgerResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return LedgerCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return LedgerCommandResult{}, ErrLedgerIdempotencyConflict
			}
			result := cloneLedgerCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *Ledger
	if command.Action == LedgerActionUpdate {
		loaded, getErr := service.repository.GetLedger(ctx, command.LedgerID)
		if getErr != nil {
			service.finalizeLedgerFailure(ctx, acquisition, getErr)
			return LedgerCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeLedgerFailure(ctx, acquisition, ErrLedgerVersionConflict)
			return LedgerCommandResult{}, ErrLedgerVersionConflict
		}
		if current.AccountingScopeID != command.AccountingScopeID {
			service.finalizeLedgerFailure(ctx, acquisition, ErrLedgerAuthorizationDenied)
			return LedgerCommandResult{}, ErrLedgerAuthorizationDenied
		}
	}

	decision, err := service.authorizer.AuthorizeLedger(ctx, actor, command, current)
	if err != nil {
		service.finalizeLedgerFailure(ctx, acquisition, err)
		return LedgerCommandResult{}, err
	}
	if err := authorizeLedgerDecision(decision, command.AccountingScopeID); err != nil {
		service.finalizeLedgerFailure(ctx, acquisition, err)
		return LedgerCommandResult{}, err
	}
	if err := normalizeLedgerReferenceError(service.references.ValidateLedgerReferences(ctx, actor, command)); err != nil {
		service.finalizeLedgerFailure(ctx, acquisition, err)
		return LedgerCommandResult{}, err
	}

	now := service.clock().UTC()
	var before, after Ledger
	switch command.Action {
	case LedgerActionCreate:
		before = Ledger{}
		after, err = NewLedger(uuid.New(), command.AccountingScopeID, command.LegalEntityID, command.LedgerType, command.FunctionalCurrency, command.FiscalCalendarID, command.LifecycleStatus, command.EffectiveDateFrom, command.EffectiveDateTo, command.Approval, now)
	case LedgerActionUpdate:
		before = cloneLedger(*current)
		after = cloneLedger(*current)
		err = after.Replace(*current, command, now)
	}
	if err == nil && decision.ApprovalRequired && command.Approval == nil {
		err = ErrLedgerApprovalRequired
	} else if err == nil && command.Approval != nil {
		err = validateLedgerApproval(service.approval.ValidateLedgerApproval(ctx, actor, command, after), decision, command.Approval)
	}
	if err != nil {
		service.finalizeLedgerFailure(ctx, acquisition, err)
		return LedgerCommandResult{}, err
	}

	record := LedgerAuditRecord{LedgerID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, AccountingScopeID: after.AccountingScopeID, LegalEntityID: after.LegalEntityID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintLedger(before), AfterFingerprint: FingerprintLedger(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	if command.Approval != nil {
		record.ApprovalRequestID = command.Approval.ApprovalRequestID
		record.ApprovalDecisionID = command.Approval.DecisionID
		record.ApproverUserID = command.Approval.ApproverUserID
	}
	result := LedgerCommandResult{Ledger: after.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus(command.Approval)}
	mutation := LedgerMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableLedgerRepository)
		if !ok {
			service.finalizeLedgerFailure(ctx, acquisition, ErrInvalidLedgerService)
			return LedgerCommandResult{}, ErrInvalidLedgerService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeLedgerFailure(ctx, acquisition, marshalErr)
			return LedgerCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeLedgerFailure(ctx, acquisition, metadataErr)
			return LedgerCommandResult{}, metadataErr
		}
		err = committer.CommitLedgerMutationWithIdempotency(ctx, mutation, DurableLedgerMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitLedgerMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeLedgerFailure(ctx, acquisition, err)
		return LedgerCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedLedgerCommand{fingerprint: fingerprint, result: cloneLedgerCommandResult(result)}
	}
	return result, nil
}

func (service *AccountingBookService) Execute(ctx context.Context, actor Actor, command AccountingBookCommand) (AccountingBookCommandResult, error) {
	if service == nil {
		return AccountingBookCommandResult{}, ErrInvalidAccountingBookService
	}
	if err := actor.Validate(); err != nil {
		return AccountingBookCommandResult{}, ErrAccountingBookAuthorizationDenied
	}
	command = command.Canonical()
	if err := command.Validate(); err != nil {
		return AccountingBookCommandResult{}, err
	}
	fingerprint, err := accountingBookCommandFingerprint(command)
	if err != nil {
		return AccountingBookCommandResult{}, err
	}

	key := actor.UserID.String() + ":" + command.AccountingScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		durableIdentity, err = newDurableIdentity("accounting-book", actor, command.AccountingScopeID, command.IdempotencyKey)
		if err != nil {
			return AccountingBookCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return AccountingBookCommandResult{}, ErrAccountingBookIdempotencyConflict
		}
		if err != nil {
			return AccountingBookCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return AccountingBookCommandResult{}, ErrAccountingBookCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return AccountingBookCommandResult{}, ErrAccountingBookDurableCommandFailed
			}
			result, decodeErr := decodeAccountingBookResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return AccountingBookCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return AccountingBookCommandResult{}, ErrAccountingBookIdempotencyConflict
			}
			result := cloneAccountingBookCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *AccountingBook
	if command.Action == LedgerActionUpdate {
		loaded, getErr := service.repository.GetAccountingBook(ctx, command.AccountingBookID)
		if getErr != nil {
			service.finalizeAccountingBookFailure(ctx, acquisition, getErr)
			return AccountingBookCommandResult{}, getErr
		}
		current = &loaded
		if !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeAccountingBookFailure(ctx, acquisition, ErrAccountingBookVersionConflict)
			return AccountingBookCommandResult{}, ErrAccountingBookVersionConflict
		}
		if current.AccountingScopeID != command.AccountingScopeID {
			service.finalizeAccountingBookFailure(ctx, acquisition, ErrAccountingBookAuthorizationDenied)
			return AccountingBookCommandResult{}, ErrAccountingBookAuthorizationDenied
		}
	}

	decision, err := service.authorizer.AuthorizeAccountingBook(ctx, actor, command, current)
	if err != nil {
		service.finalizeAccountingBookFailure(ctx, acquisition, err)
		return AccountingBookCommandResult{}, err
	}
	if err := authorizeAccountingBookDecision(decision, command.AccountingScopeID); err != nil {
		service.finalizeAccountingBookFailure(ctx, acquisition, err)
		return AccountingBookCommandResult{}, err
	}
	if err := normalizeAccountingBookReferenceError(service.references.ValidateAccountingBookReferences(ctx, actor, command)); err != nil {
		service.finalizeAccountingBookFailure(ctx, acquisition, err)
		return AccountingBookCommandResult{}, err
	}

	now := service.clock().UTC()
	var before, after AccountingBook
	switch command.Action {
	case LedgerActionCreate:
		before = AccountingBook{}
		after, err = NewAccountingBook(uuid.New(), command.AccountingScopeID, command.LedgerID, command.BookType, command.AccountingBasis, command.PostingPolicyVersion, command.LifecycleStatus, command.EffectiveDateFrom, command.EffectiveDateTo, command.Approval, now)
	case LedgerActionUpdate:
		before = cloneAccountingBook(*current)
		after = cloneAccountingBook(*current)
		err = after.Replace(*current, command, now)
	}
	if err == nil && decision.ApprovalRequired && command.Approval == nil {
		err = ErrAccountingBookApprovalRequired
	} else if err == nil && command.Approval != nil {
		err = validateAccountingBookApproval(service.approval.ValidateAccountingBookApproval(ctx, actor, command, after), decision, command.Approval)
	}
	if err != nil {
		service.finalizeAccountingBookFailure(ctx, acquisition, err)
		return AccountingBookCommandResult{}, err
	}

	record := AccountingBookAuditRecord{AccountingBookID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: command.Action, AccountingScopeID: after.AccountingScopeID, LedgerID: after.LedgerID, Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintAccountingBook(before), AfterFingerprint: FingerprintAccountingBook(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID}
	if command.Approval != nil {
		record.ApprovalRequestID = command.Approval.ApprovalRequestID
		record.ApprovalDecisionID = command.Approval.DecisionID
		record.ApproverUserID = command.Approval.ApproverUserID
	}
	result := AccountingBookCommandResult{AccountingBook: after.SafeProjection(), DecisionReference: decision.DecisionReference, PolicyReference: decision.PolicyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus(command.Approval)}
	mutation := AccountingBookMutation{Before: before, After: after, ExpectedVersion: command.ExpectedVersion, Audit: record}
	if service.durable != nil {
		committer, ok := service.repository.(DurableAccountingBookRepository)
		if !ok {
			service.finalizeAccountingBookFailure(ctx, acquisition, ErrInvalidAccountingBookService)
			return AccountingBookCommandResult{}, ErrInvalidAccountingBookService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeAccountingBookFailure(ctx, acquisition, marshalErr)
			return AccountingBookCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeAccountingBookFailure(ctx, acquisition, metadataErr)
			return AccountingBookCommandResult{}, metadataErr
		}
		err = committer.CommitAccountingBookMutationWithIdempotency(ctx, mutation, DurableAccountingBookMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitAccountingBookMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeAccountingBookFailure(ctx, acquisition, err)
		return AccountingBookCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedAccountingBookCommand{fingerprint: fingerprint, result: cloneAccountingBookCommandResult(result)}
	}
	return result, nil
}

func (service *LedgerService) ListSafe(ctx context.Context, scopeID *uuid.UUID) ([]SafeLedger, error) {
	if service == nil {
		return nil, ErrInvalidLedgerService
	}
	ledgers, err := service.repository.ListLedgers(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeLedger, 0, len(ledgers))
	for _, ledger := range ledgers {
		result = append(result, ledger.SafeProjection())
	}
	return result, nil
}

func (service *AccountingBookService) ListSafe(ctx context.Context, scopeID *uuid.UUID) ([]SafeAccountingBook, error) {
	if service == nil {
		return nil, ErrInvalidAccountingBookService
	}
	books, err := service.repository.ListAccountingBooks(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeAccountingBook, 0, len(books))
	for _, book := range books {
		result = append(result, book.SafeProjection())
	}
	return result, nil
}

func authorizeLedgerDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	if !decision.Allowed || decision.Permission != LedgerManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrLedgerAuthorizationDenied
	}
	for _, allowedScopeID := range decision.ApprovedScopeIDs {
		if allowedScopeID == uuid.Nil || allowedScopeID == scopeID {
			return nil
		}
	}
	return ErrLedgerAuthorizationDenied
}

func authorizeAccountingBookDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	if !decision.Allowed || decision.Permission != AccountingBookManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrAccountingBookAuthorizationDenied
	}
	for _, allowedScopeID := range decision.ApprovedScopeIDs {
		if allowedScopeID == uuid.Nil || allowedScopeID == scopeID {
			return nil
		}
	}
	return ErrAccountingBookAuthorizationDenied
}

func normalizeLedgerReferenceError(err error) error {
	if err == nil || errors.Is(err, ErrLedgerReferenceInvalid) || errors.Is(err, ErrLedgerReferenceUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrLedgerReferenceInvalid, err)
}

func normalizeAccountingBookReferenceError(err error) error {
	if err == nil || errors.Is(err, ErrAccountingBookReferenceInvalid) || errors.Is(err, ErrAccountingBookReferenceUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrAccountingBookReferenceInvalid, err)
}

func validateLedgerApproval(validationErr error, decision AuthorizationDecision, approval *ApprovalDecisionReference) error {
	if decision.ApprovalRequired && approval == nil {
		return ErrLedgerApprovalRequired
	}
	if approval == nil {
		return nil
	}
	if validationErr != nil {
		if errors.Is(validationErr, ErrLedgerApprovalUnavailable) {
			return validationErr
		}
		return fmt.Errorf("%w: %v", ErrLedgerApprovalInvalid, validationErr)
	}
	return nil
}

func validateAccountingBookApproval(validationErr error, decision AuthorizationDecision, approval *ApprovalDecisionReference) error {
	if decision.ApprovalRequired && approval == nil {
		return ErrAccountingBookApprovalRequired
	}
	if approval == nil {
		return nil
	}
	if validationErr != nil {
		if errors.Is(validationErr, ErrAccountingBookApprovalUnavailable) {
			return validationErr
		}
		return fmt.Errorf("%w: %v", ErrAccountingBookApprovalInvalid, validationErr)
	}
	return nil
}

func approvalStatus(approval *ApprovalDecisionReference) string {
	if approval == nil {
		return "not-required"
	}
	return "approved"
}

func newDurableIdentity(aggregate string, actor Actor, scopeID uuid.UUID, key string) (platformidempotency.IdempotencyIdentity, error) {
	scope, err := json.Marshal(struct {
		Module    string    `json:"module"`
		Aggregate string    `json:"aggregate"`
		ActorID   uuid.UUID `json:"actorId"`
		ScopeID   uuid.UUID `json:"scopeId"`
	}{"general-ledger", aggregate, actor.UserID, scopeID})
	if err != nil {
		return platformidempotency.IdempotencyIdentity{}, err
	}
	return platformidempotency.NewOpaqueIdentity(string(scope), key)
}

func ledgerCommandFingerprint(command LedgerCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func accountingBookCommandFingerprint(command AccountingBookCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func FingerprintLedger(ledger Ledger) string {
	data, _ := json.Marshal(ledger.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func FingerprintAccountingBook(book AccountingBook) string {
	data, _ := json.Marshal(book.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodeLedgerResult(body []byte) (LedgerCommandResult, error) {
	var result LedgerCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return LedgerCommandResult{}, ErrLedgerCommandInProgress
	}
	return result, nil
}

func decodeAccountingBookResult(body []byte) (AccountingBookCommandResult, error) {
	var result AccountingBookCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return AccountingBookCommandResult{}, ErrAccountingBookCommandInProgress
	}
	return result, nil
}

func (service *LedgerService) finalizeLedgerFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{ledgerFailureCode(commandErr)})
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

func (service *AccountingBookService) finalizeAccountingBookFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{accountingBookFailureCode(commandErr)})
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

func ledgerFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrLedgerAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrLedgerVersionConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrLedgerNotFound):
		return "LEDGER_NOT_FOUND"
	case errors.Is(err, ErrLedgerDuplicate):
		return "DUPLICATE_LEDGER"
	default:
		return "VALIDATION_FAILED"
	}
}

func accountingBookFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrAccountingBookAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrAccountingBookVersionConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrAccountingBookNotFound):
		return "ACCOUNTING_BOOK_NOT_FOUND"
	case errors.Is(err, ErrAccountingBookDuplicate):
		return "DUPLICATE_ACCOUNTING_BOOK"
	default:
		return "VALIDATION_FAILED"
	}
}

func cloneLedger(ledger Ledger) Ledger {
	ledger.EffectiveDateFrom = dateOnly(ledger.EffectiveDateFrom)
	ledger.EffectiveDateTo = cloneDate(ledger.EffectiveDateTo)
	ledger.Approval = cloneApproval(ledger.Approval)
	ledger.Revisions = cloneLedgerRevisions(ledger.Revisions)
	return ledger
}

func cloneAccountingBook(book AccountingBook) AccountingBook {
	book.EffectiveDateFrom = dateOnly(book.EffectiveDateFrom)
	book.EffectiveDateTo = cloneDate(book.EffectiveDateTo)
	book.Approval = cloneApproval(book.Approval)
	book.Revisions = cloneAccountingBookRevisions(book.Revisions)
	return book
}

func cloneLedgerCommandResult(result LedgerCommandResult) LedgerCommandResult {
	result.Ledger.EffectiveDateTo = cloneDate(result.Ledger.EffectiveDateTo)
	return result
}

func cloneAccountingBookCommandResult(result AccountingBookCommandResult) AccountingBookCommandResult {
	result.AccountingBook.EffectiveDateTo = cloneDate(result.AccountingBook.EffectiveDateTo)
	return result
}

func sortLedgers(values []Ledger) {
	sort.Slice(values, func(i, j int) bool { return values[i].ID.String() < values[j].ID.String() })
}

func sortAccountingBooks(values []AccountingBook) {
	sort.Slice(values, func(i, j int) bool { return values[i].ID.String() < values[j].ID.String() })
}

type MemoryLedgerRepository struct {
	mu      sync.RWMutex
	ledgers map[uuid.UUID]Ledger
	audit   LedgerAuditRecorder
}

func NewMemoryLedgerRepository() *MemoryLedgerRepository {
	return &MemoryLedgerRepository{ledgers: make(map[uuid.UUID]Ledger)}
}

func (repository *MemoryLedgerRepository) BindLedgerAuditRecorder(audit LedgerAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryLedgerRepository) GetLedger(_ context.Context, id uuid.UUID) (Ledger, error) {
	if repository == nil {
		return Ledger{}, ErrInvalidLedgerService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	ledger, ok := repository.ledgers[id]
	if !ok {
		return Ledger{}, ErrLedgerNotFound
	}
	return cloneLedger(ledger), nil
}

func (repository *MemoryLedgerRepository) ListLedgers(_ context.Context, scopeID *uuid.UUID) ([]Ledger, error) {
	if repository == nil {
		return nil, ErrInvalidLedgerService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]Ledger, 0, len(repository.ledgers))
	for _, ledger := range repository.ledgers {
		if scopeID != nil && ledger.AccountingScopeID != *scopeID {
			continue
		}
		result = append(result, cloneLedger(ledger))
	}
	sortLedgers(result)
	return result, nil
}

func (repository *MemoryLedgerRepository) CommitLedgerMutation(ctx context.Context, mutation LedgerMutation) error {
	if repository == nil {
		return ErrInvalidLedgerService
	}
	if err := validateLedgerMutation(mutation); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.ledgers[mutation.After.ID]; exists {
			return ErrLedgerDuplicate
		}
	} else {
		current, ok := repository.ledgers[mutation.After.ID]
		if !ok {
			return ErrLedgerNotFound
		}
		if current.AccountingScopeID != mutation.After.AccountingScopeID || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrLedgerVersionConflict
		}
	}
	for _, existing := range repository.ledgers {
		if existing.ID == mutation.After.ID || !sameLedgerIdentity(existing, mutation.After) {
			continue
		}
		if rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo) {
			return ErrLedgerDuplicate
		}
	}
	if repository.audit == nil {
		return ErrLedgerAuditUnavailable
	}
	if err := repository.audit.RecordLedgerMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.ledgers[mutation.After.ID] = cloneLedger(mutation.After)
	return nil
}

type MemoryAccountingBookRepository struct {
	mu      sync.RWMutex
	books   map[uuid.UUID]AccountingBook
	ledgers LedgerRepository
	audit   AccountingBookAuditRecorder
}

func NewMemoryAccountingBookRepository(ledgers LedgerRepository) *MemoryAccountingBookRepository {
	return &MemoryAccountingBookRepository{books: make(map[uuid.UUID]AccountingBook), ledgers: ledgers}
}

func (repository *MemoryAccountingBookRepository) BindAccountingBookAuditRecorder(audit AccountingBookAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryAccountingBookRepository) GetAccountingBook(_ context.Context, id uuid.UUID) (AccountingBook, error) {
	if repository == nil {
		return AccountingBook{}, ErrInvalidAccountingBookService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	book, ok := repository.books[id]
	if !ok {
		return AccountingBook{}, ErrAccountingBookNotFound
	}
	return cloneAccountingBook(book), nil
}

func (repository *MemoryAccountingBookRepository) ListAccountingBooks(_ context.Context, scopeID *uuid.UUID) ([]AccountingBook, error) {
	if repository == nil {
		return nil, ErrInvalidAccountingBookService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]AccountingBook, 0, len(repository.books))
	for _, book := range repository.books {
		if scopeID != nil && book.AccountingScopeID != *scopeID {
			continue
		}
		result = append(result, cloneAccountingBook(book))
	}
	sortAccountingBooks(result)
	return result, nil
}

func (repository *MemoryAccountingBookRepository) CommitAccountingBookMutation(ctx context.Context, mutation AccountingBookMutation) error {
	if repository == nil {
		return ErrInvalidAccountingBookService
	}
	if err := validateAccountingBookMutation(mutation); err != nil {
		return err
	}
	if repository.ledgers == nil {
		return ErrAccountingBookReferenceUnavailable
	}
	ledger, err := repository.ledgers.GetLedger(ctx, mutation.After.LedgerID)
	if err != nil {
		return ErrAccountingBookReferenceInvalid
	}
	if ledger.AccountingScopeID != mutation.After.AccountingScopeID {
		return ErrAccountingBookReferenceInvalid
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.books[mutation.After.ID]; exists {
			return ErrAccountingBookDuplicate
		}
	} else {
		current, ok := repository.books[mutation.After.ID]
		if !ok {
			return ErrAccountingBookNotFound
		}
		if current.AccountingScopeID != mutation.After.AccountingScopeID || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrAccountingBookVersionConflict
		}
	}
	for _, existing := range repository.books {
		if existing.ID == mutation.After.ID || !sameAccountingBookIdentity(existing, mutation.After) {
			continue
		}
		if rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo) {
			return ErrAccountingBookDuplicate
		}
	}
	if repository.audit == nil {
		return ErrAccountingBookAuditUnavailable
	}
	if err := repository.audit.RecordAccountingBookMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.books[mutation.After.ID] = cloneAccountingBook(mutation.After)
	return nil
}

func validateLedgerMutation(mutation LedgerMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 {
			return ErrLedgerVersionConflict
		}
		return nil
	}
	if mutation.ExpectedVersion == nil || mutation.Before.ID != mutation.After.ID || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrLedgerVersionConflict
	}
	return nil
}

func validateAccountingBookMutation(mutation AccountingBookMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.ExpectedVersion != nil || mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 {
			return ErrAccountingBookVersionConflict
		}
		return nil
	}
	if mutation.ExpectedVersion == nil || mutation.Before.ID != mutation.After.ID || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) || mutation.After.Version.Value() != mutation.ExpectedVersion.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrAccountingBookVersionConflict
	}
	return nil
}

func sameLedgerIdentity(left, right Ledger) bool {
	return left.AccountingScopeID == right.AccountingScopeID && left.LegalEntityID == right.LegalEntityID && left.LedgerType == right.LedgerType && left.FunctionalCurrency == right.FunctionalCurrency && left.FiscalCalendarID == right.FiscalCalendarID
}

func sameAccountingBookIdentity(left, right AccountingBook) bool {
	return left.AccountingScopeID == right.AccountingScopeID && left.LedgerID == right.LedgerID && left.BookType == right.BookType && left.AccountingBasis == right.AccountingBasis
}

type MemoryLedgerAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryLedgerAuthorizer) AuthorizeLedger(context.Context, Actor, LedgerCommand, *Ledger) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type MemoryAccountingBookAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryAccountingBookAuthorizer) AuthorizeAccountingBook(context.Context, Actor, AccountingBookCommand, *AccountingBook) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllReferenceValidator struct{}

func (AllowAllReferenceValidator) ValidateLedgerReferences(context.Context, Actor, LedgerCommand) error {
	return nil
}

func (AllowAllReferenceValidator) ValidateAccountingBookReferences(context.Context, Actor, AccountingBookCommand) error {
	return nil
}

type MemoryReferenceValidator struct {
	LedgerError         error
	AccountingBookError error
}

func (validator MemoryReferenceValidator) ValidateLedgerReferences(context.Context, Actor, LedgerCommand) error {
	return validator.LedgerError
}

func (validator MemoryReferenceValidator) ValidateAccountingBookReferences(context.Context, Actor, AccountingBookCommand) error {
	return validator.AccountingBookError
}

// UnavailableApprovalValidator is used by runtime profiles that do not have
// an approved Workflow decision adapter. It fails closed instead of accepting
// an unverifiable approval reference.
type UnavailableApprovalValidator struct{}

func (UnavailableApprovalValidator) ValidateLedgerApproval(context.Context, Actor, LedgerCommand, Ledger) error {
	return ErrLedgerApprovalUnavailable
}

func (UnavailableApprovalValidator) ValidateAccountingBookApproval(context.Context, Actor, AccountingBookCommand, AccountingBook) error {
	return ErrAccountingBookApprovalUnavailable
}

type AllowAllApprovalValidator struct{}

func (AllowAllApprovalValidator) ValidateLedgerApproval(context.Context, Actor, LedgerCommand, Ledger) error {
	return nil
}

func (AllowAllApprovalValidator) ValidateAccountingBookApproval(context.Context, Actor, AccountingBookCommand, AccountingBook) error {
	return nil
}

type MemoryLedgerAuditRecorder struct {
	mu      sync.Mutex
	Records []LedgerAuditRecord
	Err     error
}

func (recorder *MemoryLedgerAuditRecorder) RecordLedgerMutation(_ context.Context, record LedgerAuditRecord) error {
	if recorder == nil {
		return ErrLedgerAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

type MemoryAccountingBookAuditRecorder struct {
	mu      sync.Mutex
	Records []AccountingBookAuditRecord
	Err     error
}

func (recorder *MemoryAccountingBookAuditRecorder) RecordAccountingBookMutation(_ context.Context, record AccountingBookAuditRecord) error {
	if recorder == nil {
		return ErrAccountingBookAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

var _ LedgerRepository = (*MemoryLedgerRepository)(nil)
var _ AccountingBookRepository = (*MemoryAccountingBookRepository)(nil)
var _ LedgerAuthorizer = MemoryLedgerAuthorizer{}
var _ AccountingBookAuthorizer = MemoryAccountingBookAuthorizer{}
var _ ReferenceValidator = AllowAllReferenceValidator{}
var _ ApprovalValidator = AllowAllApprovalValidator{}
var _ ApprovalValidator = UnavailableApprovalValidator{}
var _ LedgerAuditRecorder = (*MemoryLedgerAuditRecorder)(nil)
var _ AccountingBookAuditRecorder = (*MemoryAccountingBookAuditRecorder)(nil)
