package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
)

type postingQueryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func (repository *PostgresPostingRepository) LoadPendingPosting(ctx context.Context, journalID, scopeID uuid.UUID) (PostingApprovalSubject, error) {
	if repository == nil || repository.pool == nil {
		return PostingApprovalSubject{}, ErrInvalidPostingApprovalDecisionService
	}
	subject, err := loadPostingApprovalSubject(ctx, repository.pool, journalID, scopeID, false)
	if err != nil {
		return PostingApprovalSubject{}, err
	}
	if subject.Journal.Status != PostingStatusPendingApproval {
		return PostingApprovalSubject{}, ErrPostingApprovalDecisionAlreadyApplied
	}
	return subject, nil
}

func (repository *PostgresPostingRepository) ResolvePostingApprovalApplication(ctx context.Context, command ApplyJournalApprovalDecisionCommand) (PostingApprovalDecisionResult, bool, error) {
	if repository == nil || repository.pool == nil {
		return PostingApprovalDecisionResult{}, false, ErrInvalidPostingApprovalDecisionService
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return PostingApprovalDecisionResult{}, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	var scopeID uuid.UUID
	var applicationKey, applicationFingerprint string
	var decisionID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT accounting_scope_id, COALESCE(approval_application_key, ''), COALESCE(approval_application_fingerprint, ''), approval_decision_id
		FROM gl.journal_entry
		WHERE journal_entry_id = $1
		  AND (approval_application_key = $2 OR approval_decision_id = $3)
		FOR UPDATE`, command.JournalID, command.IdempotencyKey, command.Decision.Reference.DecisionID).Scan(&scopeID, &applicationKey, &applicationFingerprint, &decisionID)
	if errors.Is(err, pgx.ErrNoRows) {
		if err := tx.Commit(ctx); err != nil {
			return PostingApprovalDecisionResult{}, false, err
		}
		committed = true
		return PostingApprovalDecisionResult{}, false, nil
	}
	if err != nil {
		return PostingApprovalDecisionResult{}, false, mapPostingPostgresError(err)
	}
	if scopeID != command.AccountingScopeID {
		return PostingApprovalDecisionResult{}, false, ErrPostingApprovalDecisionScopeConflict
	}
	if applicationFingerprint != command.ApplicationFingerprint {
		return PostingApprovalDecisionResult{}, false, ErrPostingApprovalDecisionIdempotencyConflict
	}
	subject, err := loadPostingApprovalSubject(ctx, tx, command.JournalID, command.AccountingScopeID, true)
	if err != nil {
		return PostingApprovalDecisionResult{}, false, err
	}
	applicationStatus := PostingApprovalApplicationUnchanged
	if applicationKey == command.IdempotencyKey {
		applicationStatus = PostingApprovalApplicationApplied
	}
	result := postingApprovalDecisionResultFromSubject(subject, applicationStatus)
	result.Replayed = true
	if decisionID == nil {
		return PostingApprovalDecisionResult{}, false, ErrPostingApprovalDecisionUnavailable
	}
	if err := tx.Commit(ctx); err != nil {
		return PostingApprovalDecisionResult{}, false, err
	}
	committed = true
	return result, true, nil
}

func (repository *PostgresPostingRepository) CommitPostingApprovalDecision(ctx context.Context, commit PostingApprovalDecisionCommit) (PostingApprovalDecisionResult, error) {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil {
		return PostingApprovalDecisionResult{}, ErrInvalidPostingApprovalDecisionService
	}
	_, _, err := postingApprovalTraceUUIDs(commit.Command)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	subject, err := loadPostingApprovalSubject(ctx, tx, commit.Command.JournalID, commit.Command.AccountingScopeID, false)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	gate, err := repository.lockGate(ctx, tx, subject.Request)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	subject, err = loadPostingApprovalSubject(ctx, tx, commit.Command.JournalID, commit.Command.AccountingScopeID, true)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}

	var existingFingerprint string
	var existingDecisionID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT COALESCE(approval_application_fingerprint, ''), approval_decision_id FROM gl.journal_entry WHERE journal_entry_id = $1 FOR UPDATE`, commit.Command.JournalID).Scan(&existingFingerprint, &existingDecisionID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return PostingApprovalDecisionResult{}, mapPostingPostgresError(err)
	}
	if existingFingerprint != "" {
		if existingFingerprint != commit.Command.ApplicationFingerprint {
			return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionIdempotencyConflict
		}
		result := postingApprovalDecisionResultFromSubject(subject, PostingApprovalApplicationApplied)
		result.Replayed = true
		if existingDecisionID != nil && *existingDecisionID == commit.Decision.Decision.Reference.DecisionID {
			result.ApplicationStatus = PostingApprovalApplicationUnchanged
		}
		if err := tx.Commit(ctx); err != nil {
			return PostingApprovalDecisionResult{}, err
		}
		committed = true
		return result, nil
	}
	if subject.Journal.Status != PostingStatusPendingApproval || subject.Journal.Approval == nil || !subject.Journal.Approval.Required {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionAlreadyApplied
	}
	if subject.Journal.Version != commit.Command.ExpectedJournalVersion {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionVersionConflict
	}
	if subject.Journal.Approval.ApprovalRequestID != commit.Decision.Decision.Reference.ApprovalRequestID || subject.Journal.Approval.PolicyVersion != commit.Decision.Decision.Reference.PolicyVersion || subject.Journal.RequestFingerprint != commit.Decision.Decision.Reference.CandidateFingerprint || subject.Journal.Version != commit.Decision.Decision.Reference.SubjectVersion {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionReferenceInvalid
	}
	if err := repository.validateReferences(ctx, tx, subject.Request); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if err := validatePostingApprovalGateEvidence(subject.Request, gate.Evidence, commit.Decision.Decision.Outcome); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if err := validatePostingGateAdmissionOwner(subject.Request, gate); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	commit.GateEvidence = gate.Evidence
	approval := *subject.Journal.Approval
	decisionReference := commit.Decision.Decision.Reference
	approval.Decision = &decisionReference
	approval.DecisionOutcome = commit.Decision.Decision.Outcome
	approval.DecisionReason = commit.Decision.Decision.Reason
	approval.AuthorizationReference = commit.Authorization.DecisionReference
	appliedAt := commit.Now.UTC()
	approval.AppliedAt = &appliedAt
	approvalJSON, err := json.Marshal(approval)
	if err != nil {
		return PostingApprovalDecisionResult{}, err
	}

	status := PostingStatusCancelled
	outcome := PostingOutcomePostingRejected
	approvalStatus := PostingApprovalOutcomeRejected
	nextAction := "Review the rejection and submit a new posting request if correction is required"
	journalNumber := subject.Journal.Number
	var ledgerPosition any
	if commit.Decision.Decision.Outcome == PostingApprovalOutcomeApproved {
		status = PostingStatusPosted
		outcome = PostingOutcomeJournalEntryPosted
		approvalStatus = PostingApprovalOutcomeApproved
		nextAction = "No action required"
		position := gate.LedgerPosition
		ledgerPosition = position
		journalNumber = fmt.Sprintf("JE-%08d", position)
		if err := repository.advancePostingGateAdmission(ctx, tx, subject.Request, gate, position, commit.Now.UTC()); err != nil {
			return PostingApprovalDecisionResult{}, err
		}
	}

	attemptID := uuid.New()
	auditReference, err := repository.auditWriter(ctx, tx, postingApprovalAuditRecord(commit, attemptID, outcome))
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return PostingApprovalDecisionResult{}, err
		}
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionAuditUnavailable
	}
	commandTag, err := tx.Exec(ctx, `
		UPDATE gl.journal_entry
		SET lifecycle_status = $2, journal_number = $3, approval_reference = $4, aggregate_version = aggregate_version + 1,
		    ledger_position = $5, audit_reference = $6, approval_application_key = $7,
		    approval_application_fingerprint = $8, approval_decision_id = $9, approval_applied_at = $10
		WHERE journal_entry_id = $1 AND accounting_scope_id = $11 AND lifecycle_status = $12 AND aggregate_version = $13`,
		commit.Command.JournalID, status, journalNumber, approvalJSON, ledgerPosition, auditReference, commit.Command.IdempotencyKey,
		commit.Command.ApplicationFingerprint, decisionReference.DecisionID, appliedAt, commit.Command.AccountingScopeID, PostingStatusPendingApproval, commit.Command.ExpectedJournalVersion)
	if err != nil {
		return PostingApprovalDecisionResult{}, mapPostingPostgresError(err)
	}
	if commandTag.RowsAffected() != 1 {
		return PostingApprovalDecisionResult{}, ErrPostingApprovalDecisionVersionConflict
	}

	request := subject.Request
	request.CorrelationID = commit.Command.CorrelationID
	request.CausationID = commit.Command.CausationID
	if err := repository.insertApprovalAttempt(ctx, tx, commit, attemptID, subject.Journal.ID, outcome, commit.Now.UTC()); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	resultJournal := subject.Journal
	resultJournal.Status = status
	resultJournal.Number = journalNumber
	resultJournal.Version++
	resultJournal.LedgerPosition = valueOrAnyInt64(ledgerPosition)
	resultJournal.Approval = &approval
	resultJournal.GateEvidence = gate.Evidence
	resultJournal.AuditReference = auditReference
	result := PostingApprovalDecisionResult{
		Outcome: outcome, ApplicationStatus: PostingApprovalApplicationApplied, Journal: &resultJournal, Approval: decisionReference,
		ApprovalRequestID: decisionReference.ApprovalRequestID, DecisionReference: commit.Authorization.DecisionReference,
		PolicyReference: approval.PolicyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus,
		SourceReference: resultJournal.SourceReference, GateEvidence: gate.Evidence, AuditReference: auditReference, NextAction: nextAction,
	}
	if err := repository.writeEvent(ctx, tx, PostingCommit{Request: request}, postingResultFromApprovalDecision(result), commit.Now.UTC()); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PostingApprovalDecisionResult{}, err
	}
	committed = true
	return result, nil
}

func loadPostingApprovalSubject(ctx context.Context, queryer postingQueryer, journalID, scopeID uuid.UUID, forUpdate bool) (PostingApprovalSubject, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var (
		journal                                               PostingJournal
		actualScopeID, tenantID                               uuid.UUID
		legalEntityID, ledgerID                               uuid.UUID
		accountingBookID, fiscalPeriodID                      uuid.UUID
		sourceAggregateID, requestID                          uuid.UUID
		postingAuthorizationID                                *uuid.UUID
		closeRunID, reopenRequestID                           *uuid.UUID
		operationalReopenRequestID                            *uuid.UUID
		controlAuthorityEpoch                                 *int64
		functionalCurrency, sourceContext                     string
		sourceAggregateType, idempotencyKey                   string
		requestFingerprint, transactionCurrency               string
		postingPurpose, description, status                   string
		postingDate                                           time.Time
		periodStateVersion, postingGateVersion, sourceVersion int64
		adjustmentPeriodIndicator                             bool
		conversionJSON, approvalJSON                          []byte
		correlationID, causationID                            uuid.UUID
		submittedByUserID                                     *uuid.UUID
	)
	err := queryer.QueryRow(ctx, `
		SELECT journal_entry_id, journal_number, accounting_scope_id, tenant_id, legal_entity_id, ledger_id, accounting_book_id,
		       functional_currency, source_context, source_aggregate_type, source_aggregate_id, source_version, request_id,
		       idempotency_key, request_fingerprint, posting_date, fiscal_period_id, period_state_version, posting_gate_version,
		       posting_purpose, adjustment_period_indicator, posting_authorization_id, close_run_id, reopen_request_id,
		       operational_reopen_request_id, control_authority_epoch, transaction_currency, conversion_evidence, description,
		       lifecycle_status, approval_reference, aggregate_version, COALESCE(ledger_position, 0), audit_reference,
		       correlation_id, causation_id, submitted_by_user_id
		FROM gl.journal_entry WHERE journal_entry_id = $1`+lock, journalID).Scan(
		&journal.ID, &journal.Number, &actualScopeID, &tenantID, &legalEntityID, &ledgerID, &accountingBookID,
		&functionalCurrency, &sourceContext, &sourceAggregateType, &sourceAggregateID, &sourceVersion, &requestID,
		&idempotencyKey, &requestFingerprint, &postingDate, &fiscalPeriodID, &periodStateVersion, &postingGateVersion,
		&postingPurpose, &adjustmentPeriodIndicator, &postingAuthorizationID, &closeRunID, &reopenRequestID,
		&operationalReopenRequestID, &controlAuthorityEpoch, &transactionCurrency, &conversionJSON, &description,
		&status, &approvalJSON, &journal.Version, &journal.LedgerPosition, &journal.AuditReference, &correlationID, &causationID, &submittedByUserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PostingApprovalSubject{}, ErrPostingApprovalDecisionNotFound
	}
	if err != nil {
		return PostingApprovalSubject{}, mapPostingPostgresError(err)
	}
	if actualScopeID != scopeID {
		return PostingApprovalSubject{}, ErrPostingApprovalDecisionScopeConflict
	}
	scope, err := accountingscope.New(tenantID, legalEntityID, ledgerID, accountingBookID, functionalCurrency)
	if err != nil {
		return PostingApprovalSubject{}, ErrPostingApprovalDecisionReferenceInvalid
	}
	request := PostingRequest{
		ContractVersion: PostingContractVersion, RequestID: requestID, SourceContext: sourceContext, SourceAggregateType: sourceAggregateType,
		SourceAggregateID: sourceAggregateID, SourceVersion: sourceVersion, AccountingScope: scope, AccountingScopeID: actualScopeID,
		PostingDate: postingDate, FiscalPeriodID: fiscalPeriodID, PeriodStateVersion: periodStateVersion, PostingGateVersion: postingGateVersion,
		PostingPurpose: postingPurpose, AdjustmentPeriodIndicator: adjustmentPeriodIndicator, PostingAuthorizationID: postingAuthorizationID,
		CloseRunID: closeRunID, ReopenRequestID: reopenRequestID, OperationalReopenRequestID: operationalReopenRequestID, ControlAuthorityEpoch: controlAuthorityEpoch,
		TransactionCurrency: transactionCurrency, Description: description, IdempotencyKey: idempotencyKey, RequestFingerprint: requestFingerprint,
		CorrelationID: correlationID.String(), CausationID: causationID.String(),
	}
	if len(conversionJSON) > 0 && string(conversionJSON) != "null" {
		var evidence PostingConversionEvidence
		if err := json.Unmarshal(conversionJSON, &evidence); err != nil {
			return PostingApprovalSubject{}, err
		}
		request.ConversionEvidence = &evidence
	}
	if len(approvalJSON) > 0 && string(approvalJSON) != "null" {
		var approval PostingApproval
		if err := json.Unmarshal(approvalJSON, &approval); err != nil {
			return PostingApprovalSubject{}, err
		}
		journal.Approval = &approval
	}
	journal.Status = status
	journal.SourceReference = sourceReference(request)
	journal.RequestFingerprint = requestFingerprint
	journal.GateEvidence = PostingGateEvidence{FiscalPeriodID: fiscalPeriodID, PeriodStateVersion: periodStateVersion, PostingGateVersion: postingGateVersion, GateMode: PostingGateOpen}
	if submittedByUserID != nil {
		journal.SubmittedByUserID = *submittedByUserID
	}
	rows, err := queryer.Query(ctx, `SELECT account_id, debit_or_credit, line_currency_mode, transaction_amount::text, functional_amount::text, segment_combination_id, line_reference FROM gl.journal_entry_line WHERE journal_entry_id = $1 ORDER BY line_number`, journalID)
	if err != nil {
		return PostingApprovalSubject{}, mapPostingPostgresError(err)
	}
	defer rows.Close()
	for rows.Next() {
		var line PostingLine
		if err := rows.Scan(&line.AccountID, &line.DebitOrCredit, &line.LineCurrencyMode, &line.TransactionAmount, &line.FunctionalAmount, &line.SegmentCombinationID, &line.LineReference); err != nil {
			return PostingApprovalSubject{}, mapPostingPostgresError(err)
		}
		request.Lines = append(request.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return PostingApprovalSubject{}, mapPostingPostgresError(err)
	}
	return PostingApprovalSubject{Journal: journal, Request: request}, nil
}

func postingApprovalDecisionResultFromSubject(subject PostingApprovalSubject, applicationStatus string) PostingApprovalDecisionResult {
	outcome := PostingOutcomePostingPendingApproval
	approvalStatus := "pending"
	nextAction := "Approval decision required"
	var decision ApprovalDecisionReference
	authorizationReference := uuid.Nil
	if subject.Journal.Approval != nil {
		if subject.Journal.Approval.Decision != nil {
			decision = *subject.Journal.Approval.Decision
		}
		authorizationReference = subject.Journal.Approval.AuthorizationReference
		if subject.Journal.Approval.DecisionOutcome == PostingApprovalOutcomeApproved {
			outcome = PostingOutcomeJournalEntryPosted
			approvalStatus = PostingApprovalOutcomeApproved
			nextAction = "No action required"
		} else if subject.Journal.Approval.DecisionOutcome == PostingApprovalOutcomeRejected || subject.Journal.Status == PostingStatusCancelled {
			outcome = PostingOutcomePostingRejected
			approvalStatus = PostingApprovalOutcomeRejected
			nextAction = "Review the rejection and submit a new posting request if correction is required"
		}
	}
	journal := clonePostingJournal(subject.Journal)
	policyReference := ""
	if subject.Journal.Approval != nil {
		policyReference = subject.Journal.Approval.PolicyReference
	}
	if authorizationReference == uuid.Nil {
		authorizationReference = decision.DecisionID
	}
	return PostingApprovalDecisionResult{Outcome: outcome, ApplicationStatus: applicationStatus, Journal: &journal, Approval: decision, ApprovalRequestID: decision.ApprovalRequestID, DecisionReference: authorizationReference, PolicyReference: policyReference, ValidationOutcome: "valid", ApprovalStatus: approvalStatus, SourceReference: subject.Journal.SourceReference, GateEvidence: subject.Journal.GateEvidence, AuditReference: subject.Journal.AuditReference, NextAction: nextAction}
}

func postingResultFromApprovalDecision(result PostingApprovalDecisionResult) PostingResult {
	return PostingResult{Outcome: result.Outcome, Journal: result.Journal, ApprovalRequestID: result.ApprovalRequestID, NextAction: result.NextAction, ValidationOutcome: result.ValidationOutcome, ApprovalStatus: result.ApprovalStatus, SourceReference: result.SourceReference, GateEvidence: result.GateEvidence, AuditReference: result.AuditReference, Issues: result.Issues}
}

func (repository *PostgresPostingRepository) insertApprovalAttempt(ctx context.Context, tx pgx.Tx, commit PostingApprovalDecisionCommit, attemptID, journalID uuid.UUID, outcome string, now time.Time) error {
	correlationID, causationID, err := postingApprovalTraceUUIDs(commit.Command)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO gl.posting_attempt (posting_attempt_id, request_id, journal_entry_id, accounting_scope_id, source_context, source_aggregate_type, source_aggregate_id, source_version, idempotency_key, request_fingerprint, outcome, operation, issues, actor_user_id, actor_subject_reference, correlation_id, causation_id, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,'[]'::jsonb,$13,$14,$15,$16,$17)`, attemptID, commit.Subject.Request.RequestID, journalID, commit.Subject.Request.AccountingScopeID, commit.Subject.Request.SourceContext, commit.Subject.Request.SourceAggregateType, commit.Subject.Request.SourceAggregateID, commit.Subject.Request.SourceVersion, commit.Command.IdempotencyKey, commit.Command.ApplicationFingerprint, outcome, PostingApprovalAction, commit.Actor.UserID, commit.Actor.SubjectReference, correlationID, causationID, now)
	return mapPostingPostgresError(err)
}

func postingApprovalAuditRecord(commit PostingApprovalDecisionCommit, attemptID uuid.UUID, outcome string) PostingAuditRecord {
	decision := commit.Decision.Decision.Reference
	return PostingAuditRecord{AttemptID: attemptID, JournalID: commit.Subject.Journal.ID, ActorUserID: commit.Actor.UserID, ActorSubjectReference: commit.Actor.SubjectReference,
		Outcome: outcome, AccountingScope: commit.Subject.Request.AccountingScope, AccountingScopeID: commit.Subject.Request.AccountingScopeID,
		SourceContext: commit.Subject.Request.SourceContext, SourceAggregateType: commit.Subject.Request.SourceAggregateType, SourceAggregateID: commit.Subject.Request.SourceAggregateID, SourceVersion: commit.Subject.Request.SourceVersion,
		RequestID: commit.Subject.Request.RequestID, IdempotencyKey: commit.Command.IdempotencyKey, RequestFingerprint: commit.Subject.Request.RequestFingerprint,
		Permission: PostingApprovalPermission, PolicyReference: commit.Decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: commit.Authorization.DecisionReference,
		GateEvidence:      commit.GateEvidence,
		ApprovalRequestID: decision.ApprovalRequestID, ApprovalDecisionID: decision.DecisionID, ApprovalDecisionVersion: decision.DecisionVersion, ApprovalSubjectVersion: decision.SubjectVersion,
		ApprovalCandidateFingerprint: decision.CandidateFingerprint, ApprovalApproverUserID: decision.ApproverUserID, ApprovalOutcome: commit.Decision.Decision.Outcome,
		ApprovalDecisionReason: commit.Decision.Decision.Reason, ApplicationFingerprint: commit.Command.ApplicationFingerprint, CorrelationID: commit.Command.CorrelationID, CausationID: commit.Command.CausationID}
}

func postingApprovalTraceUUIDs(command ApplyJournalApprovalDecisionCommand) (uuid.UUID, uuid.UUID, error) {
	correlationID, err := uuid.Parse(command.CorrelationID)
	if err != nil || correlationID == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrInvalidPostingApprovalDecisionCommand
	}
	causationID, err := uuid.Parse(command.CausationID)
	if err != nil || causationID == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrInvalidPostingApprovalDecisionCommand
	}
	return correlationID, causationID, nil
}

func valueOrAnyInt64(value any) int64 {
	if value == nil {
		return 0
	}
	return value.(int64)
}

var _ PostingApprovalDecisionRepository = (*PostgresPostingRepository)(nil)
