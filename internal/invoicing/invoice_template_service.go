package invoicing

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

type InvoiceTemplateAuthorizer interface {
	AuthorizeInvoiceTemplate(context.Context, Actor, InvoiceTemplateCommand, *InvoiceTemplate) (AuthorizationDecision, error)
}

type InvoiceTemplateReferenceValidator interface {
	ValidateInvoiceTemplateReferences(context.Context, Actor, InvoiceTemplateCommand) error
}

type InvoiceTemplateAuditRecorder interface {
	RecordInvoiceTemplateMutation(context.Context, InvoiceTemplateAuditRecord) error
}

type InvoiceTemplateRepository interface {
	GetInvoiceTemplate(context.Context, uuid.UUID) (InvoiceTemplate, error)
	ListInvoiceTemplates(context.Context, *uuid.UUID) ([]InvoiceTemplate, error)
	CommitInvoiceTemplateMutation(context.Context, InvoiceTemplateMutation) error
}

type DurableInvoiceTemplateRepository interface {
	CommitInvoiceTemplateMutationWithIdempotency(context.Context, InvoiceTemplateMutation, DurableInvoiceTemplateMutationCommit) error
}

type DurableInvoiceTemplateMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableInvoiceTemplateServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type InvoiceTemplateCommandResult struct {
	Template          SafeInvoiceTemplate `json:"template"`
	DecisionReference uuid.UUID           `json:"decisionReference"`
	PolicyReference   string              `json:"policyReference"`
	ValidationOutcome string              `json:"validationOutcome"`
	Replayed          bool                `json:"replayed,omitempty"`
}

type InvoiceTemplateService struct {
	repository  InvoiceTemplateRepository
	authorizer  InvoiceTemplateAuthorizer
	references  InvoiceTemplateReferenceValidator
	audit       InvoiceTemplateAuditRecorder
	clock       func() time.Time
	durable     *DurableInvoiceTemplateServiceConfig
	mu          sync.Mutex
	idempotency map[string]storedInvoiceTemplateCommand
}

type storedInvoiceTemplateCommand struct {
	fingerprint string
	result      InvoiceTemplateCommandResult
}

func NewInvoiceTemplateService(repository InvoiceTemplateRepository, authorizer InvoiceTemplateAuthorizer, references InvoiceTemplateReferenceValidator, audit InvoiceTemplateAuditRecorder, clock func() time.Time) (*InvoiceTemplateService, error) {
	return newInvoiceTemplateService(repository, authorizer, references, audit, clock, nil)
}

func NewInvoiceTemplateServiceWithDurableIdempotency(repository InvoiceTemplateRepository, authorizer InvoiceTemplateAuthorizer, references InvoiceTemplateReferenceValidator, audit InvoiceTemplateAuditRecorder, clock func() time.Time, durable DurableInvoiceTemplateServiceConfig) (*InvoiceTemplateService, error) {
	if durable.Database == nil || durable.Coordinator == nil || durable.Policy.RecordTTL <= 0 || durable.Policy.LeaseTTL <= 0 || durable.Policy.LeaseTTL >= durable.Policy.RecordTTL || strings.TrimSpace(durable.OperationID) == "" {
		return nil, ErrInvalidInvoiceTemplateService
	}
	return newInvoiceTemplateService(repository, authorizer, references, audit, clock, &durable)
}

func newInvoiceTemplateService(repository InvoiceTemplateRepository, authorizer InvoiceTemplateAuthorizer, references InvoiceTemplateReferenceValidator, audit InvoiceTemplateAuditRecorder, clock func() time.Time, durable *DurableInvoiceTemplateServiceConfig) (*InvoiceTemplateService, error) {
	if repository == nil || authorizer == nil || references == nil || audit == nil || clock == nil {
		if references == nil {
			return nil, ErrInvalidInvoiceTemplateReferenceValidator
		}
		if audit == nil {
			return nil, ErrInvalidInvoiceTemplateAuditRecorder
		}
		return nil, ErrInvalidInvoiceTemplateService
	}
	if binder, ok := repository.(interface {
		BindInvoiceTemplateAuditRecorder(InvoiceTemplateAuditRecorder)
	}); ok {
		binder.BindInvoiceTemplateAuditRecorder(audit)
	}
	return &InvoiceTemplateService{
		repository: repository, authorizer: authorizer, references: references, audit: audit,
		clock: clock, durable: durable, idempotency: make(map[string]storedInvoiceTemplateCommand),
	}, nil
}

func (service *InvoiceTemplateService) Execute(ctx context.Context, actor Actor, command InvoiceTemplateCommand) (InvoiceTemplateCommandResult, error) {
	if service == nil {
		return InvoiceTemplateCommandResult{}, ErrInvalidInvoiceTemplateService
	}
	if err := actor.Validate(); err != nil {
		return InvoiceTemplateCommandResult{}, err
	}
	if err := command.Validate(); err != nil {
		return InvoiceTemplateCommandResult{}, err
	}
	command = command.Canonical()
	fingerprint, err := invoiceTemplateCommandFingerprint(command)
	if err != nil {
		return InvoiceTemplateCommandResult{}, err
	}

	key := actor.UserID.String() + ":" + command.AccountingScopeID.String() + ":" + command.IdempotencyKey
	var acquisition platformidempotency.DurableAcquisition
	var durableIdentity platformidempotency.IdempotencyIdentity
	if service.durable != nil {
		durableIdentity, err = newInvoiceTemplateDurableIdentity(actor, command.AccountingScopeID, command.IdempotencyKey)
		if err != nil {
			return InvoiceTemplateCommandResult{}, err
		}
		acquisition, err = service.durable.Coordinator.Acquire(ctx, service.durable.Database, durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, service.durable.Policy)
		if errors.Is(err, platformidempotency.ErrIdempotencyConflict) {
			return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateIdempotencyConflict
		}
		if err != nil {
			return InvoiceTemplateCommandResult{}, err
		}
		if acquisition.Decision() == platformidempotency.DecisionReturn {
			if acquisition.Result().State() == platformidempotency.StateInProgress {
				return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateCommandInProgress
			}
			if acquisition.Result().State() == platformidempotency.StateFailed {
				return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateDurableCommandFailed
			}
			result, decodeErr := decodeInvoiceTemplateResult(acquisition.Result().ResultBody())
			if decodeErr != nil {
				return InvoiceTemplateCommandResult{}, decodeErr
			}
			result.Replayed = true
			return result, nil
		}
	} else {
		service.mu.Lock()
		defer service.mu.Unlock()
		if previous, ok := service.idempotency[key]; ok {
			if previous.fingerprint != fingerprint {
				return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateIdempotencyConflict
			}
			result := cloneInvoiceTemplateCommandResult(previous.result)
			result.Replayed = true
			return result, nil
		}
	}

	var current *InvoiceTemplate
	if command.Action == InvoiceTemplateActionUpdate {
		loaded, getErr := service.repository.GetInvoiceTemplate(ctx, command.TemplateID)
		if getErr != nil {
			service.finalizeFailure(ctx, acquisition, getErr)
			return InvoiceTemplateCommandResult{}, getErr
		}
		current = &loaded
		if current.AccountingScopeID != command.AccountingScopeID {
			service.finalizeFailure(ctx, acquisition, ErrInvoiceTemplateAuthorizationDenied)
			return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateAuthorizationDenied
		}
		if command.ExpectedVersion == nil || !current.Version.Matches(*command.ExpectedVersion) {
			service.finalizeFailure(ctx, acquisition, ErrInvoiceTemplateVersionConflict)
			return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateVersionConflict
		}
	}

	decision, err := service.authorizer.AuthorizeInvoiceTemplate(ctx, actor, command, current)
	if err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return InvoiceTemplateCommandResult{}, err
	}
	if err := authorizeInvoiceTemplateDecision(decision, command.AccountingScopeID); err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return InvoiceTemplateCommandResult{}, err
	}
	if err := normalizeInvoiceTemplateReferenceError(service.references.ValidateInvoiceTemplateReferences(ctx, actor, command)); err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return InvoiceTemplateCommandResult{}, err
	}

	now := service.clock().UTC()
	var before, after InvoiceTemplate
	switch command.Action {
	case InvoiceTemplateActionCreate:
		before = InvoiceTemplate{}
		var templateID uuid.UUID
		templateID, err = uuid.NewV7()
		if err == nil {
			after, err = NewInvoiceTemplate(templateID, command.AccountingScopeID, command, now)
		}
	case InvoiceTemplateActionUpdate:
		before = cloneInvoiceTemplate(*current)
		after = cloneInvoiceTemplate(*current)
		err = after.Replace(command, now)
	}
	if err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return InvoiceTemplateCommandResult{}, err
	}

	audit := InvoiceTemplateAuditRecord{
		TemplateID: after.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference,
		Action: command.Action, AccountingScopeID: after.AccountingScopeID, Permission: decision.Permission,
		PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion,
		DecisionReference: decision.DecisionReference, ExpectedVersion: cloneVersion(command.ExpectedVersion),
		RevisionNumber: after.RevisionNumber, BeforeFingerprint: FingerprintInvoiceTemplate(before),
		AfterFingerprint: FingerprintInvoiceTemplate(after), CorrelationID: command.CorrelationID, CausationID: command.CausationID,
	}
	result := InvoiceTemplateCommandResult{
		Template: after.SafeProjection(), DecisionReference: decision.DecisionReference,
		PolicyReference: decision.PolicyReference, ValidationOutcome: "valid",
	}
	mutation := InvoiceTemplateMutation{Before: before, After: after, ExpectedVersion: cloneVersion(command.ExpectedVersion), Audit: audit}
	if service.durable != nil {
		committer, ok := service.repository.(DurableInvoiceTemplateRepository)
		if !ok {
			service.finalizeFailure(ctx, acquisition, ErrInvalidInvoiceTemplateService)
			return InvoiceTemplateCommandResult{}, ErrInvalidInvoiceTemplateService
		}
		body, marshalErr := json.Marshal(result)
		if marshalErr != nil {
			service.finalizeFailure(ctx, acquisition, marshalErr)
			return InvoiceTemplateCommandResult{}, marshalErr
		}
		status := 200
		aggregateID := after.ID
		metadata, metadataErr := platformidempotency.NewCommandResultMetadata(durableIdentity, platformidempotency.Fingerprint(fingerprint), service.durable.OperationID, platformidempotency.StateEstablished, &status, body, &aggregateID, nil)
		if metadataErr != nil {
			service.finalizeFailure(ctx, acquisition, metadataErr)
			return InvoiceTemplateCommandResult{}, metadataErr
		}
		err = committer.CommitInvoiceTemplateMutationWithIdempotency(ctx, mutation, DurableInvoiceTemplateMutationCommit{Coordinator: service.durable.Coordinator, Acquisition: acquisition, Result: metadata})
	} else {
		err = service.repository.CommitInvoiceTemplateMutation(ctx, mutation)
	}
	if err != nil {
		service.finalizeFailure(ctx, acquisition, err)
		return InvoiceTemplateCommandResult{}, err
	}
	if service.durable == nil {
		service.idempotency[key] = storedInvoiceTemplateCommand{fingerprint: fingerprint, result: cloneInvoiceTemplateCommandResult(result)}
	}
	return result, nil
}

func (service *InvoiceTemplateService) ListSafe(ctx context.Context, actor Actor, scopeID *uuid.UUID) ([]SafeInvoiceTemplate, error) {
	if service == nil {
		return nil, ErrInvalidInvoiceTemplateService
	}
	command := InvoiceTemplateCommand{Action: "read"}
	if scopeID != nil {
		command.AccountingScopeID = *scopeID
	}
	decision, err := service.authorizer.AuthorizeInvoiceTemplate(ctx, actor, command, nil)
	if err != nil {
		return nil, err
	}
	if scopeID != nil {
		if err := authorizeInvoiceTemplateDecision(decision, *scopeID); err != nil {
			return nil, err
		}
	} else if !decision.Allowed || decision.Permission != InvoiceTemplateManagementPermission || decision.DecisionReference == uuid.Nil || !containsWildcardScope(decision.ApprovedScopeIDs) {
		return nil, ErrInvoiceTemplateAuthorizationDenied
	}
	templates, err := service.repository.ListInvoiceTemplates(ctx, scopeID)
	if err != nil {
		return nil, err
	}
	result := make([]SafeInvoiceTemplate, 0, len(templates))
	for _, template := range templates {
		result = append(result, template.SafeProjection())
	}
	return result, nil
}

func (service *InvoiceTemplateService) GetSafe(ctx context.Context, actor Actor, id uuid.UUID, scopeID *uuid.UUID) (SafeInvoiceTemplate, error) {
	if service == nil {
		return SafeInvoiceTemplate{}, ErrInvalidInvoiceTemplateService
	}
	template, err := service.repository.GetInvoiceTemplate(ctx, id)
	if err != nil {
		return SafeInvoiceTemplate{}, err
	}
	if scopeID != nil && *scopeID != template.AccountingScopeID {
		return SafeInvoiceTemplate{}, ErrInvoiceTemplateAuthorizationDenied
	}
	decision, err := service.authorizer.AuthorizeInvoiceTemplate(ctx, actor, InvoiceTemplateCommand{Action: "read", AccountingScopeID: template.AccountingScopeID}, &template)
	if err != nil {
		return SafeInvoiceTemplate{}, err
	}
	if err := authorizeInvoiceTemplateDecision(decision, template.AccountingScopeID); err != nil {
		return SafeInvoiceTemplate{}, err
	}
	return template.SafeProjection(), nil
}

func authorizeInvoiceTemplateDecision(decision AuthorizationDecision, scopeID uuid.UUID) error {
	if !decision.Allowed || decision.Permission != InvoiceTemplateManagementPermission || decision.DecisionReference == uuid.Nil {
		return ErrInvoiceTemplateAuthorizationDenied
	}
	if scopeID == uuid.Nil {
		if containsWildcardScope(decision.ApprovedScopeIDs) {
			return nil
		}
		return ErrInvoiceTemplateAuthorizationDenied
	}
	for _, approved := range decision.ApprovedScopeIDs {
		if approved == uuid.Nil || approved == scopeID {
			return nil
		}
	}
	return ErrInvoiceTemplateAuthorizationDenied
}

func containsWildcardScope(scopes []uuid.UUID) bool {
	for _, scope := range scopes {
		if scope == uuid.Nil {
			return true
		}
	}
	return false
}

func normalizeInvoiceTemplateReferenceError(err error) error {
	if err == nil || errors.Is(err, ErrInvoiceTemplateReferenceInvalid) || errors.Is(err, ErrInvoiceTemplateReferenceUnavailable) {
		return err
	}
	return fmt.Errorf("%w: %v", ErrInvoiceTemplateReferenceInvalid, err)
}

func newInvoiceTemplateDurableIdentity(actor Actor, scopeID uuid.UUID, key string) (platformidempotency.IdempotencyIdentity, error) {
	identity, err := json.Marshal(struct {
		Module    string    `json:"module"`
		Aggregate string    `json:"aggregate"`
		ActorID   uuid.UUID `json:"actorId"`
		ScopeID   uuid.UUID `json:"scopeId"`
	}{"invoicing", "invoice-template", actor.UserID, scopeID})
	if err != nil {
		return platformidempotency.IdempotencyIdentity{}, err
	}
	return platformidempotency.NewOpaqueIdentity(string(identity), key)
}

func decodeInvoiceTemplateResult(body []byte) (InvoiceTemplateCommandResult, error) {
	var result InvoiceTemplateCommandResult
	if err := json.Unmarshal(body, &result); err != nil {
		return InvoiceTemplateCommandResult{}, ErrInvoiceTemplateCommandInProgress
	}
	return result, nil
}

func (service *InvoiceTemplateService) finalizeFailure(ctx context.Context, acquisition platformidempotency.DurableAcquisition, commandErr error) {
	if service == nil || service.durable == nil || acquisition.Decision() != platformidempotency.DecisionExecute {
		return
	}
	body, err := json.Marshal(struct {
		Code string `json:"code"`
	}{invoiceTemplateFailureCode(commandErr)})
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

func invoiceTemplateFailureCode(err error) string {
	switch {
	case errors.Is(err, ErrInvoiceTemplateAuthorizationDenied):
		return "AUTHORIZATION_DENIED"
	case errors.Is(err, ErrInvoiceTemplateVersionConflict):
		return "VERSION_CONFLICT"
	case errors.Is(err, ErrInvoiceTemplateNotFound):
		return "INVOICE_TEMPLATE_NOT_FOUND"
	case errors.Is(err, ErrInvoiceTemplateDuplicate):
		return "DUPLICATE_INVOICE_TEMPLATE"
	case errors.Is(err, ErrInvoiceTemplateIdempotencyConflict):
		return "IDEMPOTENCY_CONFLICT"
	default:
		return "VALIDATION_FAILED"
	}
}

func cloneInvoiceTemplateCommandResult(result InvoiceTemplateCommandResult) InvoiceTemplateCommandResult {
	result.Template.LineItems = cloneLineItems(result.Template.LineItems)
	result.Template.Applicability = cloneApplicability(result.Template.Applicability)
	result.Template.EffectiveDateTo = cloneDate(result.Template.EffectiveDateTo)
	if result.Template.Revisions != nil {
		result.Template.Revisions = append([]SafeInvoiceTemplateRevision(nil), result.Template.Revisions...)
	}
	return result
}

func cloneVersion(value *aggregateversion.AggregateVersion) *aggregateversion.AggregateVersion {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

type MemoryInvoiceTemplateRepository struct {
	mu        sync.RWMutex
	templates map[uuid.UUID]InvoiceTemplate
	audit     InvoiceTemplateAuditRecorder
}

func NewMemoryInvoiceTemplateRepository() *MemoryInvoiceTemplateRepository {
	return &MemoryInvoiceTemplateRepository{templates: make(map[uuid.UUID]InvoiceTemplate)}
}

func (repository *MemoryInvoiceTemplateRepository) BindInvoiceTemplateAuditRecorder(audit InvoiceTemplateAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryInvoiceTemplateRepository) GetInvoiceTemplate(_ context.Context, id uuid.UUID) (InvoiceTemplate, error) {
	if repository == nil {
		return InvoiceTemplate{}, ErrInvalidInvoiceTemplateService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	template, ok := repository.templates[id]
	if !ok {
		return InvoiceTemplate{}, ErrInvoiceTemplateNotFound
	}
	return cloneInvoiceTemplate(template), nil
}

func (repository *MemoryInvoiceTemplateRepository) ListInvoiceTemplates(_ context.Context, scopeID *uuid.UUID) ([]InvoiceTemplate, error) {
	if repository == nil {
		return nil, ErrInvalidInvoiceTemplateService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := make([]InvoiceTemplate, 0, len(repository.templates))
	for _, template := range repository.templates {
		if scopeID != nil && template.AccountingScopeID != *scopeID {
			continue
		}
		result = append(result, cloneInvoiceTemplate(template))
	}
	sortInvoiceTemplates(result)
	return result, nil
}

func (repository *MemoryInvoiceTemplateRepository) GetInvoiceTemplateRevision(ctx context.Context, id uuid.UUID, version aggregateversion.AggregateVersion) (InvoiceTemplateRevision, error) {
	template, err := repository.GetInvoiceTemplate(ctx, id)
	if err != nil {
		return InvoiceTemplateRevision{}, err
	}
	for _, revision := range template.Revisions {
		if revision.Snapshot.Version == version {
			return revision, nil
		}
	}
	return InvoiceTemplateRevision{}, ErrInvoiceTemplateRevisionNotFound
}

func (repository *MemoryInvoiceTemplateRepository) ListInvoiceTemplateRevisions(ctx context.Context, id uuid.UUID) ([]InvoiceTemplateRevision, error) {
	template, err := repository.GetInvoiceTemplate(ctx, id)
	if err != nil {
		return nil, err
	}
	result := make([]InvoiceTemplateRevision, len(template.Revisions))
	copy(result, template.Revisions)
	return result, nil
}

func (repository *MemoryInvoiceTemplateRepository) CommitInvoiceTemplateMutation(ctx context.Context, mutation InvoiceTemplateMutation) error {
	if repository == nil {
		return ErrInvalidInvoiceTemplateService
	}
	if err := validateInvoiceTemplateMutation(mutation); err != nil {
		return err
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if mutation.Before.ID == uuid.Nil {
		if _, exists := repository.templates[mutation.After.ID]; exists {
			return ErrInvoiceTemplateDuplicate
		}
	} else {
		current, ok := repository.templates[mutation.After.ID]
		if !ok {
			return ErrInvoiceTemplateNotFound
		}
		if current.AccountingScopeID != mutation.After.AccountingScopeID || mutation.ExpectedVersion == nil || !current.Version.Matches(*mutation.ExpectedVersion) {
			return ErrInvoiceTemplateVersionConflict
		}
	}
	for _, existing := range repository.templates {
		if existing.ID == mutation.After.ID || existing.AccountingScopeID != mutation.After.AccountingScopeID || existing.Code != mutation.After.Code || !applicabilityOverlaps(existing.Applicability, mutation.After.Applicability) {
			continue
		}
		if rangesOverlap(existing.EffectiveDateFrom, existing.EffectiveDateTo, mutation.After.EffectiveDateFrom, mutation.After.EffectiveDateTo) {
			return ErrInvoiceTemplateDuplicate
		}
	}
	if repository.audit == nil {
		return ErrInvoiceTemplateAuditUnavailable
	}
	if err := repository.audit.RecordInvoiceTemplateMutation(ctx, mutation.Audit); err != nil {
		return err
	}
	repository.templates[mutation.After.ID] = cloneInvoiceTemplate(mutation.After)
	return nil
}

type MemoryInvoiceTemplateAuthorizer struct {
	Decision AuthorizationDecision
	Err      error
}

func (authorizer MemoryInvoiceTemplateAuthorizer) AuthorizeInvoiceTemplate(context.Context, Actor, InvoiceTemplateCommand, *InvoiceTemplate) (AuthorizationDecision, error) {
	if authorizer.Err != nil {
		return AuthorizationDecision{}, authorizer.Err
	}
	return authorizer.Decision, nil
}

type AllowAllInvoiceTemplateReferenceValidator struct{}

func (AllowAllInvoiceTemplateReferenceValidator) ValidateInvoiceTemplateReferences(context.Context, Actor, InvoiceTemplateCommand) error {
	return nil
}

// UnavailableInvoiceTemplateReferenceValidator is the production-safe
// default until the owning Tax/payment reference adapters are connected.
// Invoicing must not establish a template while those authoritative ports are
// unavailable.
type UnavailableInvoiceTemplateReferenceValidator struct{}

func (UnavailableInvoiceTemplateReferenceValidator) ValidateInvoiceTemplateReferences(context.Context, Actor, InvoiceTemplateCommand) error {
	return ErrInvoiceTemplateReferenceUnavailable
}

type MemoryInvoiceTemplateAuditRecorder struct {
	mu      sync.Mutex
	Records []InvoiceTemplateAuditRecord
	Err     error
}

func (recorder *MemoryInvoiceTemplateAuditRecorder) RecordInvoiceTemplateMutation(_ context.Context, record InvoiceTemplateAuditRecord) error {
	if recorder == nil {
		return ErrInvoiceTemplateAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return recorder.Err
	}
	recorder.Records = append(recorder.Records, record)
	return nil
}

func validateInvoiceTemplateMutation(mutation InvoiceTemplateMutation) error {
	if err := mutation.After.Validate(); err != nil {
		return err
	}
	if mutation.Before.ID == uuid.Nil {
		if mutation.After.Version.Value() != 1 || mutation.After.RevisionNumber != 1 || mutation.ExpectedVersion != nil {
			return ErrInvalidInvoiceTemplate
		}
		return nil
	}
	if mutation.Before.ID != mutation.After.ID || mutation.ExpectedVersion == nil || !mutation.Before.Version.Matches(*mutation.ExpectedVersion) {
		return ErrInvoiceTemplateVersionConflict
	}
	if mutation.After.Version.Value() != mutation.Before.Version.Value()+1 || mutation.After.RevisionNumber != mutation.Before.RevisionNumber+1 {
		return ErrInvoiceTemplateVersionConflict
	}
	return nil
}

var _ InvoiceTemplateRepository = (*MemoryInvoiceTemplateRepository)(nil)
var _ InvoiceTemplateAuthorizer = MemoryInvoiceTemplateAuthorizer{}
var _ InvoiceTemplateReferenceValidator = AllowAllInvoiceTemplateReferenceValidator{}
var _ InvoiceTemplateReferenceValidator = UnavailableInvoiceTemplateReferenceValidator{}
var _ InvoiceTemplateAuditRecorder = (*MemoryInvoiceTemplateAuditRecorder)(nil)
