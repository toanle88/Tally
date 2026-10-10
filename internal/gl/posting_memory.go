package gl

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type PostingAttempt struct {
	ID                 uuid.UUID
	RequestID          uuid.UUID
	JournalID          uuid.UUID
	Outcome            string
	AccountingScopeKey string
	SourceContext      string
	SourceAggregateID  uuid.UUID
	SourceVersion      int64
	IdempotencyKey     string
	RequestFingerprint string
	Issues             []PostingIssue
	CreatedAt          time.Time
}

type PostingEvent struct {
	ID         uuid.UUID
	Type       string
	OccurredAt time.Time
	Result     PostingResult
}

type MemoryPostingRepository struct {
	mu         sync.RWMutex
	results    map[string]PostingResult
	sources    map[string]PostingResult
	journals   []PostingJournal
	attempts   []PostingAttempt
	events     []PostingEvent
	audit      PostingAuditRecorder
	nextNumber int64
}

func NewMemoryPostingRepository() *MemoryPostingRepository {
	return &MemoryPostingRepository{
		results: make(map[string]PostingResult),
		sources: make(map[string]PostingResult),
	}
}

func (repository *MemoryPostingRepository) BindPostingAuditRecorder(audit PostingAuditRecorder) {
	if repository == nil {
		return
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	repository.audit = audit
}

func (repository *MemoryPostingRepository) ResolveExistingPosting(_ context.Context, request PostingRequest) (PostingResult, bool, error) {
	if repository == nil {
		return PostingResult{}, false, ErrInvalidPostingService
	}
	scopeKey, err := postingScopeKey(request)
	if err != nil {
		return PostingResult{}, false, err
	}
	idempotencyKey := scopeKey + "|" + request.IdempotencyKey
	sourceKey := fmt.Sprintf("%s|%s|%s|%s|%d", scopeKey, request.SourceContext, request.SourceAggregateType, request.SourceAggregateID, request.SourceVersion)

	repository.mu.RLock()
	defer repository.mu.RUnlock()
	if existing, ok := repository.results[idempotencyKey]; ok {
		if existingFingerprint(existing) != request.RequestFingerprint {
			return PostingResult{}, false, ErrPostingIdempotencyConflict
		}
		return clonePostingResult(existing), true, nil
	}
	if existing, ok := repository.sources[sourceKey]; ok {
		if existingFingerprint(existing) != request.RequestFingerprint {
			return PostingResult{}, false, ErrPostingSourceDuplicate
		}
		return clonePostingResult(existing), true, nil
	}
	return PostingResult{}, false, nil
}

func (repository *MemoryPostingRepository) CommitPosting(_ context.Context, commit PostingCommit) (PostingResult, error) {
	if repository == nil {
		return PostingResult{}, ErrInvalidPostingService
	}
	scopeKey, err := postingScopeKey(commit.Request)
	if err != nil {
		return PostingResult{}, err
	}
	idempotencyKey := scopeKey + "|" + commit.Request.IdempotencyKey
	sourceKey := fmt.Sprintf("%s|%s|%s|%s|%d", scopeKey, commit.Request.SourceContext, commit.Request.SourceAggregateType, commit.Request.SourceAggregateID, commit.Request.SourceVersion)

	repository.mu.Lock()
	defer repository.mu.Unlock()
	if existing, ok := repository.results[idempotencyKey]; ok {
		if existingFingerprint(existing) != commit.Request.RequestFingerprint {
			return PostingResult{}, ErrPostingIdempotencyConflict
		}
		existing.Replayed = true
		return clonePostingResult(existing), nil
	}
	if existing, ok := repository.sources[sourceKey]; ok {
		if existingFingerprint(existing) != commit.Request.RequestFingerprint {
			return PostingResult{}, ErrPostingSourceDuplicate
		}
		existing.Replayed = true
		repository.results[idempotencyKey] = clonePostingResult(existing)
		return clonePostingResult(existing), nil
	}

	journalID := uuid.New()
	status := PostingStatusPosted
	outcome := PostingOutcomeJournalEntryPosted
	nextAction := "No action required"
	approvalStatus := "not_required"
	journalNumber := ""
	ledgerPosition := int64(0)
	var approval *PostingApproval
	if commit.Approval.Required {
		status = PostingStatusPendingApproval
		outcome = PostingOutcomePostingPendingApproval
		nextAction = "Approval decision required"
		approvalStatus = "pending"
		journalNumber = fmt.Sprintf("PJE-%s", journalID.String()[:8])
		copy := commit.Approval
		approval = &copy
	} else {
		repository.nextNumber++
		ledgerPosition = repository.nextNumber
		journalNumber = fmt.Sprintf("JE-%08d", ledgerPosition)
	}

	journal := PostingJournal{
		ID: journalID, Number: journalNumber, Version: 1, LedgerPosition: ledgerPosition,
		Status: status, Approval: approval, SourceReference: sourceReference(commit.Request), GateEvidence: commit.GateEvidence,
		RequestFingerprint: commit.Request.RequestFingerprint,
	}
	auditReference, err := repository.recordAuditLocked(commit, uuid.New(), journalID, outcome, nil)
	if err != nil {
		return PostingResult{}, err
	}
	journal.AuditReference = auditReference
	result := PostingResult{
		Outcome: outcome, Journal: &journal, ApprovalRequestID: commit.Approval.ApprovalRequestID,
		NextAction: nextAction, ValidationOutcome: "valid", ApprovalStatus: approvalStatus,
		SourceReference: sourceReference(commit.Request), GateEvidence: commit.GateEvidence, AuditReference: auditReference,
	}
	resultCopy := clonePostingResult(result)
	repository.results[idempotencyKey] = resultCopy
	repository.sources[sourceKey] = clonePostingResult(result)
	repository.journals = append(repository.journals, journal)
	repository.attempts = append(repository.attempts, PostingAttempt{
		ID: uuid.New(), RequestID: commit.Request.RequestID, JournalID: journalID, Outcome: outcome,
		AccountingScopeKey: scopeKey, SourceContext: commit.Request.SourceContext, SourceAggregateID: commit.Request.SourceAggregateID,
		SourceVersion: commit.Request.SourceVersion, IdempotencyKey: commit.Request.IdempotencyKey,
		RequestFingerprint: commit.Request.RequestFingerprint, CreatedAt: commit.Now,
	})
	repository.events = append(repository.events, PostingEvent{ID: uuid.New(), Type: outcome, OccurredAt: commit.Now, Result: clonePostingResult(result)})
	return result, nil
}

func (repository *MemoryPostingRepository) RecordPostingAttempt(_ context.Context, request PostingRequest, actor Actor, outcome string, issues []PostingIssue) error {
	if repository == nil {
		return ErrInvalidPostingService
	}
	scopeKey, err := postingScopeKey(request)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	repository.mu.Lock()
	defer repository.mu.Unlock()
	attemptID := uuid.New()
	if err := repository.recordAuditOnlyLocked(request, actor, attemptID, outcome, issues); err != nil {
		return err
	}
	repository.attempts = append(repository.attempts, PostingAttempt{
		ID: attemptID, RequestID: request.RequestID, Outcome: outcome,
		AccountingScopeKey: scopeKey, SourceContext: request.SourceContext, SourceAggregateID: request.SourceAggregateID,
		SourceVersion: request.SourceVersion, IdempotencyKey: request.IdempotencyKey,
		RequestFingerprint: request.RequestFingerprint, Issues: append([]PostingIssue(nil), issues...), CreatedAt: now,
	})
	repository.events = append(repository.events, PostingEvent{ID: uuid.New(), Type: outcome, OccurredAt: now, Result: PostingResult{Outcome: outcome, Issues: append([]PostingIssue(nil), issues...)}})
	return nil
}

func (repository *MemoryPostingRepository) Journals() []PostingJournal {
	if repository == nil {
		return nil
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := append([]PostingJournal(nil), repository.journals...)
	for i := range result {
		if result[i].Approval != nil {
			approval := *result[i].Approval
			result[i].Approval = &approval
		}
	}
	return result
}

func (repository *MemoryPostingRepository) Attempts() []PostingAttempt {
	if repository == nil {
		return nil
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := append([]PostingAttempt(nil), repository.attempts...)
	for i := range result {
		result[i].Issues = append([]PostingIssue(nil), result[i].Issues...)
	}
	return result
}

func (repository *MemoryPostingRepository) Events() []PostingEvent {
	if repository == nil {
		return nil
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	result := append([]PostingEvent(nil), repository.events...)
	for i := range result {
		result[i].Result = clonePostingResult(result[i].Result)
	}
	return result
}

type MemoryPostingAuditRecorder struct {
	mu      sync.Mutex
	Records []PostingAuditRecord
	Err     error
}

func (recorder *MemoryPostingAuditRecorder) RecordPostingMutation(_ context.Context, record PostingAuditRecord) (uuid.UUID, error) {
	if recorder == nil {
		return uuid.Nil, ErrPostingAuditUnavailable
	}
	recorder.mu.Lock()
	defer recorder.mu.Unlock()
	if recorder.Err != nil {
		return uuid.Nil, recorder.Err
	}
	record.Issues = append([]PostingIssue(nil), record.Issues...)
	recorder.Records = append(recorder.Records, record)
	return uuid.New(), nil
}

func (repository *MemoryPostingRepository) recordAuditLocked(commit PostingCommit, attemptID, journalID uuid.UUID, outcome string, issues []PostingIssue) (uuid.UUID, error) {
	if repository.audit == nil {
		return uuid.Nil, ErrPostingAuditUnavailable
	}
	return repository.audit.RecordPostingMutation(context.Background(), PostingAuditRecord{
		AttemptID: attemptID, JournalID: journalID, ActorUserID: commit.Actor.UserID, ActorSubjectReference: commit.Actor.SubjectReference,
		Outcome: outcome, AccountingScope: commit.Request.AccountingScope, SourceContext: commit.Request.SourceContext,
		AccountingScopeID:   commit.Request.AccountingScopeID,
		SourceAggregateType: commit.Request.SourceAggregateType, SourceAggregateID: commit.Request.SourceAggregateID, SourceVersion: commit.Request.SourceVersion,
		RequestID: commit.Request.RequestID, IdempotencyKey: commit.Request.IdempotencyKey, RequestFingerprint: commit.Request.RequestFingerprint,
		Permission: PostingPermission, PolicyReference: commit.Authorization.PolicyReference, PolicyVersion: commit.Authorization.PolicyVersion,
		DecisionReference: commit.Authorization.DecisionReference, CorrelationID: commit.Request.CorrelationID, CausationID: commit.Request.CausationID,
		Issues: issues,
	})
}

func (repository *MemoryPostingRepository) recordAuditOnlyLocked(request PostingRequest, actor Actor, attemptID uuid.UUID, outcome string, issues []PostingIssue) error {
	if repository.audit == nil {
		return ErrPostingAuditUnavailable
	}
	_, err := repository.audit.RecordPostingMutation(context.Background(), PostingAuditRecord{
		AttemptID: attemptID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference,
		Outcome: outcome, AccountingScope: request.AccountingScope, SourceContext: request.SourceContext,
		AccountingScopeID:   request.AccountingScopeID,
		SourceAggregateType: request.SourceAggregateType, SourceAggregateID: request.SourceAggregateID, SourceVersion: request.SourceVersion,
		RequestID: request.RequestID, IdempotencyKey: request.IdempotencyKey, RequestFingerprint: request.RequestFingerprint,
		Permission: PostingPermission, CorrelationID: request.CorrelationID, CausationID: request.CausationID, Issues: issues,
	})
	return err
}

func postingScopeKey(request PostingRequest) (string, error) {
	data, err := request.AccountingScope.MarshalJSON()
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func sourceReference(request PostingRequest) string {
	return fmt.Sprintf("%s:%s:%s:%d", request.SourceContext, request.SourceAggregateType, request.SourceAggregateID, request.SourceVersion)
}

func existingFingerprint(result PostingResult) string {
	if result.Journal == nil {
		return ""
	}
	return result.Journal.RequestFingerprint
}

func clonePostingResult(result PostingResult) PostingResult {
	result.Issues = append([]PostingIssue(nil), result.Issues...)
	if result.Journal != nil {
		journal := *result.Journal
		if result.Journal.Approval != nil {
			approval := *result.Journal.Approval
			journal.Approval = &approval
		}
		result.Journal = &journal
	}
	return result
}

func sortPostingAttempts(attempts []PostingAttempt) {
	sort.SliceStable(attempts, func(i, j int) bool { return attempts[i].CreatedAt.Before(attempts[j].CreatedAt) })
}

var _ PostingRepository = (*MemoryPostingRepository)(nil)
var _ PostingAuditRecorder = (*MemoryPostingAuditRecorder)(nil)
