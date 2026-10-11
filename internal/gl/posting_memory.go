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
	Operation          string
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
	mu           sync.RWMutex
	results      map[string]PostingResult
	sources      map[string]PostingResult
	requests     map[uuid.UUID]PostingRequest
	applications map[string]storedPostingApprovalApplication
	decisions    map[string]storedPostingApprovalApplication
	journals     []PostingJournal
	attempts     []PostingAttempt
	events       []PostingEvent
	audit        PostingAuditRecorder
	nextNumber   int64
}

type storedPostingApprovalApplication struct {
	fingerprint string
	result      PostingApprovalDecisionResult
}

func NewMemoryPostingRepository() *MemoryPostingRepository {
	return &MemoryPostingRepository{
		results:      make(map[string]PostingResult),
		sources:      make(map[string]PostingResult),
		requests:     make(map[uuid.UUID]PostingRequest),
		applications: make(map[string]storedPostingApprovalApplication),
		decisions:    make(map[string]storedPostingApprovalApplication),
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
		RequestFingerprint: commit.Request.RequestFingerprint, SubmittedByUserID: commit.Actor.UserID,
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
	repository.requests[journalID] = clonePostingRequest(commit.Request)
	repository.journals = append(repository.journals, journal)
	repository.attempts = append(repository.attempts, PostingAttempt{
		ID: uuid.New(), RequestID: commit.Request.RequestID, JournalID: journalID, Outcome: outcome,
		AccountingScopeKey: scopeKey, SourceContext: commit.Request.SourceContext, SourceAggregateID: commit.Request.SourceAggregateID,
		SourceVersion: commit.Request.SourceVersion, IdempotencyKey: commit.Request.IdempotencyKey,
		RequestFingerprint: commit.Request.RequestFingerprint, Operation: "submit-posting-request", CreatedAt: commit.Now,
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
		RequestFingerprint: request.RequestFingerprint, Operation: "submit-posting-request", Issues: append([]PostingIssue(nil), issues...), CreatedAt: now,
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
	result := make([]PostingJournal, len(repository.journals))
	for i := range repository.journals {
		result[i] = clonePostingJournal(repository.journals[i])
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
		DecisionReference: commit.Authorization.DecisionReference, GateEvidence: commit.GateEvidence, CorrelationID: commit.Request.CorrelationID, CausationID: commit.Request.CausationID,
		Issues: issues,
	})
}

func (repository *MemoryPostingRepository) LoadPendingPosting(_ context.Context, journalID, scopeID uuid.UUID) (PostingApprovalSubject, error) {
	if repository == nil {
		return PostingApprovalSubject{}, ErrInvalidPostingApprovalDecisionService
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	request, ok := repository.requests[journalID]
	if !ok {
		return PostingApprovalSubject{}, ErrPostingApprovalDecisionNotFound
	}
	for _, journal := range repository.journals {
		if journal.ID != journalID {
			continue
		}
		if request.AccountingScopeID != scopeID {
			return PostingApprovalSubject{}, ErrPostingApprovalDecisionScopeConflict
		}
		if journal.Status != PostingStatusPendingApproval {
			return PostingApprovalSubject{}, ErrPostingApprovalDecisionAlreadyApplied
		}
		return PostingApprovalSubject{Journal: clonePostingJournal(journal), Request: clonePostingRequest(request)}, nil
	}
	return PostingApprovalSubject{}, ErrPostingApprovalDecisionNotFound
}

func (repository *MemoryPostingRepository) ResolvePostingApprovalApplication(_ context.Context, command ApplyJournalApprovalDecisionCommand) (PostingApprovalDecisionResult, bool, error) {
	if repository == nil {
		return PostingApprovalDecisionResult{}, false, ErrInvalidPostingApprovalDecisionService
	}
	scopeKey := command.AccountingScopeID.String()
	applicationKey := scopeKey + "|" + command.JournalID.String() + "|" + command.IdempotencyKey
	decisionKey := scopeKey + "|" + command.JournalID.String() + "|" + command.Decision.Reference.DecisionID.String()
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	if existing, ok := repository.applications[applicationKey]; ok {
		if existing.fingerprint != command.ApplicationFingerprint {
			return PostingApprovalDecisionResult{}, false, ErrPostingApprovalDecisionIdempotencyConflict
		}
		return clonePostingApprovalDecisionResult(existing.result), true, nil
	}
	if existing, ok := repository.decisions[decisionKey]; ok {
		if existing.fingerprint != command.ApplicationFingerprint {
			return PostingApprovalDecisionResult{}, false, ErrPostingApprovalDecisionIdempotencyConflict
		}
		result := clonePostingApprovalDecisionResult(existing.result)
		result.ApplicationStatus = PostingApprovalApplicationUnchanged
		return result, true, nil
	}
	return PostingApprovalDecisionResult{}, false, nil
}

func (repository *MemoryPostingRepository) CommitPostingApprovalDecision(_ context.Context, commit PostingApprovalDecisionCommit) (PostingApprovalDecisionResult, error) {
	if repository == nil {
		return PostingApprovalDecisionResult{}, ErrInvalidPostingApprovalDecisionService
	}
	scopeKey := commit.Command.AccountingScopeID.String()
	applicationKey := scopeKey + "|" + commit.Command.JournalID.String() + "|" + commit.Command.IdempotencyKey
	decisionKey := scopeKey + "|" + commit.Command.JournalID.String() + "|" + commit.Command.Decision.Reference.DecisionID.String()
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if existing, ok := repository.applications[applicationKey]; ok {
		if existing.fingerprint != commit.Command.ApplicationFingerprint {
			return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionIdempotencyConflict
		}
		result := clonePostingApprovalDecisionResult(existing.result)
		result.Replayed = true
		return result, nil
	}
	if existing, ok := repository.decisions[decisionKey]; ok {
		if existing.fingerprint != commit.Command.ApplicationFingerprint {
			return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionIdempotencyConflict
		}
		result := clonePostingApprovalDecisionResult(existing.result)
		result.ApplicationStatus = PostingApprovalApplicationUnchanged
		return result, nil
	}
	request, requestFound := repository.requests[commit.Command.JournalID]
	if !requestFound {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionNotFound
	}
	scope, scopeErr := postingScopeKey(request)
	if scopeErr != nil {
		return PostingApprovalDecisionResult{}, scopeErr
	}

	journalIndex := -1
	for index := range repository.journals {
		if repository.journals[index].ID == commit.Command.JournalID {
			journalIndex = index
			break
		}
	}
	if journalIndex < 0 {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionNotFound
	}
	current := repository.journals[journalIndex]
	if current.Status != PostingStatusPendingApproval {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionAlreadyApplied
	}
	if current.Version != commit.Command.ExpectedJournalVersion {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionVersionConflict
	}
	if current.Approval == nil || current.Approval.ApprovalRequestID != commit.Decision.Decision.Reference.ApprovalRequestID || current.Approval.PolicyVersion != commit.Decision.Decision.Reference.PolicyVersion || current.RequestFingerprint != commit.Decision.Decision.Reference.CandidateFingerprint || current.Version != commit.Decision.Decision.Reference.SubjectVersion {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionReferenceInvalid
	}

	approval := *current.Approval
	decisionReference := commit.Decision.Decision.Reference
	approval.Decision = &decisionReference
	approval.DecisionOutcome = commit.Decision.Decision.Outcome
	approval.DecisionReason = commit.Decision.Decision.Reason
	approval.AuthorizationReference = commit.Authorization.DecisionReference
	appliedAt := commit.Now.UTC()
	approval.AppliedAt = &appliedAt
	status := PostingStatusCancelled
	outcome := PostingOutcomePostingRejected
	approvalStatus := PostingApprovalOutcomeRejected
	nextAction := "Review the rejection and submit a new posting request if correction is required"
	ledgerPosition := int64(0)
	journalNumber := current.Number
	if commit.Decision.Decision.Outcome == PostingApprovalOutcomeApproved {
		status = PostingStatusPosted
		outcome = PostingOutcomeJournalEntryPosted
		approvalStatus = PostingApprovalOutcomeApproved
		nextAction = "No action required"
		repository.nextNumber++
		ledgerPosition = repository.nextNumber
		journalNumber = fmt.Sprintf("JE-%08d", ledgerPosition)
	}
	auditReference, err := repository.recordApprovalAuditLocked(commit, uuid.New(), outcome)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	current.Status = status
	current.Number = journalNumber
	current.LedgerPosition = ledgerPosition
	current.Version++
	current.Approval = &approval
	current.AuditReference = auditReference
	current.GateEvidence = commit.GateEvidence
	repository.journals[journalIndex] = current
	resultJournal := clonePostingJournal(current)
	result := PostingApprovalDecisionResult{Outcome: outcome, ApplicationStatus: PostingApprovalApplicationApplied, Journal: &resultJournal, Approval: decisionReference, ApprovalRequestID: decisionReference.ApprovalRequestID, DecisionReference: commit.Authorization.DecisionReference, PolicyReference: approval.PolicyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus, SourceReference: current.SourceReference, GateEvidence: commit.GateEvidence, AuditReference: auditReference, NextAction: nextAction}
	requestJournal := clonePostingJournal(current)
	requestResult := PostingResult{Outcome: outcome, Journal: &requestJournal, ApprovalRequestID: decisionReference.ApprovalRequestID, NextAction: nextAction, ValidationOutcome: "valid", ApprovalStatus: approvalStatus, SourceReference: current.SourceReference, GateEvidence: commit.GateEvidence, AuditReference: auditReference}
	idempotencyKey := scope + "|" + request.IdempotencyKey
	sourceKey := fmt.Sprintf("%s|%s|%s|%s|%d", scope, request.SourceContext, request.SourceAggregateType, request.SourceAggregateID, request.SourceVersion)
	repository.results[idempotencyKey] = clonePostingResult(requestResult)
	repository.sources[sourceKey] = clonePostingResult(requestResult)
	repository.applications[applicationKey] = storedPostingApprovalApplication{fingerprint: commit.Command.ApplicationFingerprint, result: clonePostingApprovalDecisionResult(result)}
	repository.decisions[decisionKey] = storedPostingApprovalApplication{fingerprint: commit.Command.ApplicationFingerprint, result: clonePostingApprovalDecisionResult(result)}
	repository.attempts = append(repository.attempts, PostingAttempt{ID: uuid.New(), RequestID: request.RequestID, JournalID: current.ID, Outcome: outcome, AccountingScopeKey: scope, SourceContext: request.SourceContext, SourceAggregateID: request.SourceAggregateID, SourceVersion: request.SourceVersion, IdempotencyKey: commit.Command.IdempotencyKey, RequestFingerprint: commit.Command.ApplicationFingerprint, Operation: PostingApprovalAction, CreatedAt: commit.Now.UTC()})
	repository.events = append(repository.events, PostingEvent{ID: uuid.New(), Type: outcome, OccurredAt: commit.Now.UTC(), Result: requestResult})
	return result, nil
}

func (repository *MemoryPostingRepository) recordApprovalAuditLocked(commit PostingApprovalDecisionCommit, attemptID uuid.UUID, outcome string) (uuid.UUID, error) {
	if repository.audit == nil {
		return uuid.Nil, ErrPostingApprovalDecisionAuditUnavailable
	}
	request := commit.Subject.Request
	decision := commit.Decision.Decision.Reference
	return repository.audit.RecordPostingMutation(context.Background(), PostingAuditRecord{
		AttemptID: attemptID, JournalID: commit.Subject.Journal.ID, ActorUserID: commit.Actor.UserID, ActorSubjectReference: commit.Actor.SubjectReference,
		Outcome: outcome, AccountingScope: request.AccountingScope, AccountingScopeID: request.AccountingScopeID, SourceContext: request.SourceContext,
		SourceAggregateType: request.SourceAggregateType, SourceAggregateID: request.SourceAggregateID, SourceVersion: request.SourceVersion,
		RequestID: request.RequestID, IdempotencyKey: commit.Command.IdempotencyKey, RequestFingerprint: request.RequestFingerprint,
		Permission: PostingApprovalPermission, PolicyReference: commit.Decision.PolicyReference, PolicyVersion: decision.PolicyVersion, GateEvidence: commit.GateEvidence,
		DecisionReference: commit.Authorization.DecisionReference, ApprovalRequestID: decision.ApprovalRequestID, ApprovalDecisionID: decision.DecisionID,
		ApprovalDecisionVersion: decision.DecisionVersion, ApprovalSubjectVersion: decision.SubjectVersion, ApprovalCandidateFingerprint: decision.CandidateFingerprint,
		ApprovalApproverUserID: decision.ApproverUserID, ApprovalOutcome: commit.Decision.Decision.Outcome, ApprovalDecisionReason: commit.Decision.Decision.Reason, ApplicationFingerprint: commit.Command.ApplicationFingerprint,
		CorrelationID: commit.Command.CorrelationID, CausationID: commit.Command.CausationID,
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
		journal := clonePostingJournal(*result.Journal)
		result.Journal = &journal
	}
	return result
}

func clonePostingApproval(value *PostingApproval) *PostingApproval {
	if value == nil {
		return nil
	}
	cloned := *value
	if value.Decision != nil {
		decision := *value.Decision
		cloned.Decision = &decision
	}
	if value.AppliedAt != nil {
		appliedAt := *value.AppliedAt
		cloned.AppliedAt = &appliedAt
	}
	return &cloned
}

func clonePostingJournal(value PostingJournal) PostingJournal {
	value.Approval = clonePostingApproval(value.Approval)
	return value
}

func clonePostingRequest(value PostingRequest) PostingRequest {
	cloned := value.Canonical()
	cloneUUID := func(value *uuid.UUID) *uuid.UUID {
		if value == nil {
			return nil
		}
		copy := *value
		return &copy
	}
	cloneInt64 := func(value *int64) *int64 {
		if value == nil {
			return nil
		}
		copy := *value
		return &copy
	}
	cloned.PostingAuthorizationID = cloneUUID(value.PostingAuthorizationID)
	cloned.CloseRunID = cloneUUID(value.CloseRunID)
	cloned.ReopenRequestID = cloneUUID(value.ReopenRequestID)
	cloned.OperationalReopenRequestID = cloneUUID(value.OperationalReopenRequestID)
	cloned.ControlAuthorityEpoch = cloneInt64(value.ControlAuthorityEpoch)
	return cloned
}

func clonePostingApprovalDecisionResult(result PostingApprovalDecisionResult) PostingApprovalDecisionResult {
	result.Issues = append([]PostingIssue(nil), result.Issues...)
	if result.Journal != nil {
		journal := clonePostingJournal(*result.Journal)
		result.Journal = &journal
	}
	return result
}

func sortPostingAttempts(attempts []PostingAttempt) {
	sort.SliceStable(attempts, func(i, j int) bool { return attempts[i].CreatedAt.Before(attempts[j].CreatedAt) })
}

var _ PostingRepository = (*MemoryPostingRepository)(nil)
var _ PostingAuditRecorder = (*MemoryPostingAuditRecorder)(nil)
