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
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

const zeroUUIDLiteral = "00000000-0000-0000-0000-000000000000"

func (repository *PostgresPostingRepository) LoadJournalForReversal(ctx context.Context, scopeID, journalID uuid.UUID) (JournalEntrySnapshot, error) {
	if repository == nil || repository.pool == nil {
		return JournalEntrySnapshot{}, ErrInvalidPostingService
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return JournalEntrySnapshot{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	snapshot, err := repository.loadJournalSnapshot(ctx, tx, scopeID, journalID, false)
	if err != nil {
		return JournalEntrySnapshot{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return JournalEntrySnapshot{}, err
	}
	committed = true
	return snapshot, nil
}

func (repository *PostgresPostingRepository) ResolveExistingJournalReversal(ctx context.Context, command ReverseJournalEntryCommand, fingerprint string) (JournalReversalResult, bool, error) {
	if repository == nil || repository.pool == nil {
		return JournalReversalResult{}, false, ErrInvalidPostingService
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return JournalReversalResult{}, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	reversalID, existingFingerprint, found, err := repository.findReversalIdentity(ctx, tx, command)
	if err != nil {
		return JournalReversalResult{}, false, err
	}
	if !found {
		if _, _, found, err := repository.findExistingReversalForOriginal(ctx, tx, command); err != nil {
			return JournalReversalResult{}, false, err
		} else if found {
			return JournalReversalResult{}, false, ErrJournalEntryAlreadyReversed
		}
		if err := tx.Commit(ctx); err != nil {
			return JournalReversalResult{}, false, err
		}
		committed = true
		return JournalReversalResult{}, false, nil
	}
	if existingFingerprint != fingerprint {
		return JournalReversalResult{}, false, ErrPostingIdempotencyConflict
	}
	reversal, err := repository.loadJournalSnapshot(ctx, tx, command.AccountingScopeID, reversalID, true)
	if err != nil {
		return JournalReversalResult{}, false, err
	}
	if reversal.Journal.ReversalOfJournalEntryID == nil {
		return JournalReversalResult{}, false, ErrJournalEntryNotFound
	}
	original, err := repository.loadJournalSnapshot(ctx, tx, command.AccountingScopeID, *reversal.Journal.ReversalOfJournalEntryID, true)
	if err != nil {
		return JournalReversalResult{}, false, err
	}
	if err := tx.Commit(ctx); err != nil {
		return JournalReversalResult{}, false, err
	}
	committed = true
	return journalReversalResultFromSnapshots(original, reversal, reversal.Journal.CorrectionReason, uuidValue(reversal.Request.PostingAuthorizationID), reversal.Journal.GateEvidence, reversal.Journal.AuditReference, fingerprint), true, nil
}

func (repository *PostgresPostingRepository) CommitJournalReversal(ctx context.Context, commit JournalReversalCommit) (JournalReversalResult, error) {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil {
		return JournalReversalResult{}, ErrInvalidPostingService
	}
	correlationID, causationID, err := postingTraceUUIDs(commit.Request)
	if err != nil {
		return JournalReversalResult{}, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return JournalReversalResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	// Replay and identity conflicts are resolved before gate admission so a
	// safe retry remains replayable after the period has advanced or closed.
	reversalID, existingFingerprint, found, err := repository.findReversalIdentity(ctx, tx, commit.Command)
	if err != nil {
		return JournalReversalResult{}, err
	}
	if found {
		if existingFingerprint != commit.RequestFingerprint {
			return JournalReversalResult{}, ErrPostingIdempotencyConflict
		}
		reversal, loadErr := repository.loadJournalSnapshot(ctx, tx, commit.Command.AccountingScopeID, reversalID, true)
		if loadErr != nil {
			return JournalReversalResult{}, loadErr
		}
		if reversal.Journal.ReversalOfJournalEntryID == nil {
			return JournalReversalResult{}, ErrJournalEntryNotFound
		}
		original, loadErr := repository.loadJournalSnapshot(ctx, tx, commit.Command.AccountingScopeID, *reversal.Journal.ReversalOfJournalEntryID, true)
		if loadErr != nil {
			return JournalReversalResult{}, loadErr
		}
		if err := tx.Commit(ctx); err != nil {
			return JournalReversalResult{}, err
		}
		committed = true
		result := journalReversalResultFromSnapshots(original, reversal, reversal.Journal.CorrectionReason, uuidValue(reversal.Request.PostingAuthorizationID), reversal.Journal.GateEvidence, reversal.Journal.AuditReference, commit.RequestFingerprint)
		result.Replayed = true
		return result, nil
	}
	if _, _, found, err := repository.findExistingReversalForOriginal(ctx, tx, commit.Command); err != nil {
		return JournalReversalResult{}, err
	} else if found {
		return JournalReversalResult{}, ErrJournalEntryAlreadyReversed
	}

	gate, err := repository.lockGate(ctx, tx, commit.Request)
	if err != nil {
		return JournalReversalResult{}, err
	}
	if err := validatePostingGateEvidenceAt(commit.Request, gate.Evidence, commit.Now.UTC()); err != nil {
		return JournalReversalResult{}, err
	}
	if err := validatePostingGateAdmissionOwner(commit.Request, gate); err != nil {
		return JournalReversalResult{}, err
	}
	if _, _, found, err := repository.findExistingReversalForOriginal(ctx, tx, commit.Command); err != nil {
		return JournalReversalResult{}, err
	} else if found {
		return JournalReversalResult{}, ErrJournalEntryAlreadyReversed
	}
	original, err := repository.loadJournalSnapshot(ctx, tx, commit.Command.AccountingScopeID, commit.Command.OriginalJournalID, true)
	if err != nil {
		return JournalReversalResult{}, err
	}
	if err := validateReversalOriginal(commit.Command, original); err != nil {
		return JournalReversalResult{}, err
	}
	if err := repository.validateReferences(ctx, tx, commit.Request); err != nil {
		return JournalReversalResult{}, err
	}

	now := commit.Now.UTC()
	reversalID = uuid.New()
	journalNumber := fmt.Sprintf("JE-%08d", gate.LedgerPosition)
	if err := repository.advancePostingGateAdmission(ctx, tx, commit.Request, gate, gate.LedgerPosition, now); err != nil {
		return JournalReversalResult{}, err
	}
	attemptID := uuid.New()
	auditReference, err := repository.auditWriter(ctx, tx, journalReversalAuditRecord(commit, attemptID, reversalID, PostingOutcomeJournalEntryReversed, nil))
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return JournalReversalResult{}, err
		}
		return JournalReversalResult{}, ErrPostingAuditUnavailable
	}

	conversionJSON, err := json.Marshal(commit.Request.ConversionEvidence)
	if err != nil {
		return JournalReversalResult{}, err
	}
	storedStatus := PostingStatusPendingApproval
	_, err = tx.Exec(ctx, `
		INSERT INTO gl.journal_entry (
			journal_entry_id, journal_number, accounting_scope_id, tenant_id, legal_entity_id, ledger_id, accounting_book_id,
			functional_currency, source_context, source_aggregate_type, source_aggregate_id, source_version, request_id,
			idempotency_key, request_fingerprint, posting_date, fiscal_period_id, period_state_version, posting_gate_version,
			posting_gate_mode, posting_purpose, adjustment_period_indicator, posting_authorization_id, close_run_id, reopen_request_id, operational_reopen_request_id, control_authority_epoch, reopen_authorization_scope, reopen_authorization_expiry,
			transaction_currency, conversion_evidence, description, lifecycle_status, approval_reference, aggregate_version,
			ledger_position, audit_reference, correlation_id, causation_id, submitted_by_user_id, reversal_of_journal_entry_id, correction_reason, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37,$38,$39,$40,$41,$42,$43)`,
		reversalID, journalNumber, commit.Request.AccountingScopeID, commit.Request.AccountingScope.TenantID(), commit.Request.AccountingScope.LegalEntityID(), commit.Request.AccountingScope.LedgerID(), commit.Request.AccountingScope.AccountingBookID(),
		commit.Request.AccountingScope.FunctionalCurrency(), commit.Request.SourceContext, commit.Request.SourceAggregateType, commit.Request.SourceAggregateID, commit.Request.SourceVersion, commit.Request.RequestID,
		commit.Request.IdempotencyKey, commit.Request.RequestFingerprint, commit.Request.PostingDate, commit.Request.FiscalPeriodID, commit.Request.PeriodStateVersion, commit.Request.PostingGateVersion,
		gate.Evidence.GateMode, commit.Request.PostingPurpose, commit.Request.AdjustmentPeriodIndicator, nullablePostingUUID(commit.Request.PostingAuthorizationID), nullablePostingUUID(commit.Request.CloseRunID), nullablePostingUUID(commit.Request.ReopenRequestID), nullablePostingUUID(commit.Request.OperationalReopenRequestID), nullablePostingInt64(commit.Request.ControlAuthorityEpoch), nonNullPostingJSON(gate.Evidence.AuthorizationScope), nullablePostingTime(gate.Evidence.AuthorizationExpiry),
		commit.Request.TransactionCurrency, nullablePostingJSON(conversionJSON), commit.Request.Description, storedStatus, nil, 1,
		gate.LedgerPosition, auditReference, correlationID, causationID, commit.Actor.UserID, commit.Command.OriginalJournalID, commit.Command.CorrectionReason, now)
	if err != nil {
		return JournalReversalResult{}, mapPostingPostgresError(err)
	}
	for index, line := range commit.Request.Lines {
		if _, err := tx.Exec(ctx, `INSERT INTO gl.journal_entry_line (journal_entry_id, accounting_scope_id, line_number, account_id, debit_or_credit, line_currency_mode, transaction_amount, functional_amount, segment_combination_id, line_reference) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, reversalID, commit.Request.AccountingScopeID, index+1, line.AccountID, line.DebitOrCredit, line.LineCurrencyMode, line.TransactionAmount, line.FunctionalAmount, line.SegmentCombinationID, line.LineReference); err != nil {
			return JournalReversalResult{}, mapPostingPostgresError(err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE gl.journal_entry SET lifecycle_status = $2 WHERE journal_entry_id = $1`, reversalID, PostingStatusPosted); err != nil {
		return JournalReversalResult{}, mapPostingPostgresError(err)
	}
	if err := repository.insertReversalAttempt(ctx, tx, commit, attemptID, reversalID, PostingOutcomeJournalEntryReversed, nil, now); err != nil {
		return JournalReversalResult{}, err
	}
	reversal := JournalEntrySnapshot{
		Journal: PostingJournal{ID: reversalID, Number: journalNumber, Version: 1, LedgerPosition: gate.LedgerPosition, Status: PostingStatusPosted, SourceReference: sourceReference(commit.Request), RequestFingerprint: commit.Request.RequestFingerprint, GateEvidence: gate.Evidence, AuditReference: auditReference, SubmittedByUserID: commit.Actor.UserID, PostingPurpose: commit.Request.PostingPurpose, PostingAuthorizationID: cloneUUIDValue(commit.Request.PostingAuthorizationID), ReversalOfJournalEntryID: cloneUUIDValue(&commit.Command.OriginalJournalID), CorrectionReason: commit.Command.CorrectionReason, Lines: append([]PostingLine(nil), commit.Request.Lines...)},
		Request: clonePostingRequest(commit.Request), Lines: append([]PostingLine(nil), commit.Request.Lines...),
	}
	if err := repository.writeJournalReversalEvent(ctx, tx, commit, original, reversal, auditReference, now); err != nil {
		return JournalReversalResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return JournalReversalResult{}, err
	}
	committed = true
	return journalReversalResultFromSnapshots(original, reversal, commit.Command.CorrectionReason, commit.Command.PostingAuthorizationID, gate.Evidence, auditReference, commit.RequestFingerprint), nil
}

func (repository *PostgresPostingRepository) RecordJournalReversalAttempt(ctx context.Context, command ReverseJournalEntryCommand, actor Actor, fingerprint, outcome string, issues []PostingIssue) error {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil {
		return ErrInvalidPostingService
	}
	correlationID, err := uuid.Parse(command.CorrelationID)
	if err != nil || correlationID == uuid.Nil {
		return ErrInvalidJournalReversalCommand
	}
	causationID, err := uuid.Parse(command.CausationID)
	if err != nil || causationID == uuid.Nil {
		return ErrInvalidJournalReversalCommand
	}
	if fingerprint == "" {
		fingerprint, _ = journalReversalFingerprint(command)
	}
	storedOutcome := PostingOutcomePostingRejected
	if outcome == "conflict" && hasReversalIssue(issues, "idempotency_conflict") {
		storedOutcome = PostingOutcomeIdempotencyConflict
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()
	attemptID := uuid.New()
	auditReference, err := repository.auditWriter(ctx, tx, PostingAuditRecord{
		AttemptID: attemptID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Outcome: storedOutcome,
		Operation: JournalReversalOperation, OriginalJournalID: command.OriginalJournalID, CorrectionReason: command.CorrectionReason,
		AccountingScopeID: command.AccountingScopeID, SourceContext: command.SourceContext, SourceAggregateType: command.SourceAggregateType,
		SourceAggregateID: command.SourceAggregateID, SourceVersion: command.SourceVersion, RequestID: command.CommandID,
		IdempotencyKey: command.IdempotencyKey, RequestFingerprint: fingerprint, Permission: JournalReversalPermission,
		CorrelationID: command.CorrelationID, CausationID: command.CausationID, Issues: issues,
	})
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrPostingAuditUnavailable
	}
	if err := repository.insertReversalAttempt(ctx, tx, JournalReversalCommit{Command: command, Actor: actor, RequestFingerprint: fingerprint}, attemptID, uuid.Nil, storedOutcome, issues, time.Now().UTC()); err != nil {
		return err
	}
	if err := repository.writeJournalReversalAttemptEvent(ctx, tx, command, storedOutcome, auditReference, correlationID, causationID, issues, time.Now().UTC()); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (repository *PostgresPostingRepository) findReversalIdentity(ctx context.Context, tx pgx.Tx, command ReverseJournalEntryCommand) (uuid.UUID, string, bool, error) {
	var reversalID uuid.UUID
	var fingerprint string
	err := tx.QueryRow(ctx, `SELECT journal_entry_id, request_fingerprint FROM gl.journal_entry WHERE accounting_scope_id = $1 AND idempotency_key = $2 FOR UPDATE`, command.AccountingScopeID, command.IdempotencyKey).Scan(&reversalID, &fingerprint)
	if err == nil {
		return reversalID, fingerprint, true, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, mapPostingPostgresError(err)
	}
	err = tx.QueryRow(ctx, `SELECT journal_entry_id, request_fingerprint FROM gl.journal_entry WHERE accounting_scope_id = $1 AND reversal_of_journal_entry_id IS NOT NULL AND source_context = $2 AND source_aggregate_type = $3 AND source_aggregate_id = $4 AND source_version = $5 FOR UPDATE`, command.AccountingScopeID, command.SourceContext, command.SourceAggregateType, command.SourceAggregateID, command.SourceVersion).Scan(&reversalID, &fingerprint)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, mapPostingPostgresError(err)
	}
	return reversalID, fingerprint, true, nil
}

func (repository *PostgresPostingRepository) findExistingReversalForOriginal(ctx context.Context, tx pgx.Tx, command ReverseJournalEntryCommand) (uuid.UUID, string, bool, error) {
	var reversalID uuid.UUID
	var fingerprint string
	err := tx.QueryRow(ctx, `SELECT journal_entry_id, request_fingerprint FROM gl.journal_entry WHERE accounting_scope_id = $1 AND reversal_of_journal_entry_id = $2 FOR UPDATE`, command.AccountingScopeID, command.OriginalJournalID).Scan(&reversalID, &fingerprint)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, "", false, nil
	}
	if err != nil {
		return uuid.Nil, "", false, mapPostingPostgresError(err)
	}
	return reversalID, fingerprint, true, nil
}

func (repository *PostgresPostingRepository) loadJournalSnapshot(ctx context.Context, tx pgx.Tx, scopeID, journalID uuid.UUID, forUpdate bool) (JournalEntrySnapshot, error) {
	lock := ""
	if forUpdate {
		lock = " FOR UPDATE"
	}
	var (
		id, storedScopeID, tenantID, legalEntityID, ledgerID, bookID, sourceID, requestID, periodID, authID, closeRunID, reopenID, operationalReopenID, auditID, submittedBy, reversalOf uuid.UUID
		journalNumber, functionalCurrency, sourceContext, sourceType, idempotencyKey, fingerprint, postingPurpose, transactionCurrency, description, status, gateMode, correctionReason  string
		sourceVersion, periodVersion, postingGateVersion, controlEpoch, aggregateVersion, ledgerPosition                                                                                 int64
		postingDate                                                                                                                                                                      time.Time
		adjustmentPeriod                                                                                                                                                                 bool
		conversionJSON, approvalJSON, reopenScope                                                                                                                                        []byte
		reopenExpiry                                                                                                                                                                     *time.Time
		correlationID, causationID                                                                                                                                                       uuid.UUID
	)
	query := `SELECT journal_entry_id, journal_number, accounting_scope_id, tenant_id, legal_entity_id, ledger_id, accounting_book_id, functional_currency, source_context, source_aggregate_type, source_aggregate_id, source_version, request_id, idempotency_key, request_fingerprint, posting_date, fiscal_period_id, period_state_version, posting_gate_version, posting_gate_mode, posting_purpose, adjustment_period_indicator, COALESCE(posting_authorization_id, '` + zeroUUIDLiteral + `'), COALESCE(close_run_id, '` + zeroUUIDLiteral + `'), COALESCE(reopen_request_id, '` + zeroUUIDLiteral + `'), COALESCE(operational_reopen_request_id, '` + zeroUUIDLiteral + `'), COALESCE(control_authority_epoch, 0), reopen_authorization_scope, reopen_authorization_expiry, transaction_currency, conversion_evidence, description, lifecycle_status, approval_reference, aggregate_version, COALESCE(ledger_position, 0), audit_reference, correlation_id, causation_id, COALESCE(submitted_by_user_id, '` + zeroUUIDLiteral + `'), COALESCE(reversal_of_journal_entry_id, '` + zeroUUIDLiteral + `'), COALESCE(correction_reason, '') FROM gl.journal_entry WHERE journal_entry_id = $1 AND accounting_scope_id = $2` + lock
	err := tx.QueryRow(ctx, query, journalID, scopeID).Scan(
		&id, &journalNumber, &storedScopeID, &tenantID, &legalEntityID, &ledgerID, &bookID, &functionalCurrency, &sourceContext, &sourceType, &sourceID, &sourceVersion, &requestID, &idempotencyKey, &fingerprint, &postingDate, &periodID, &periodVersion, &postingGateVersion, &gateMode, &postingPurpose, &adjustmentPeriod, &authID, &closeRunID, &reopenID, &operationalReopenID, &controlEpoch, &reopenScope, &reopenExpiry, &transactionCurrency, &conversionJSON, &description, &status, &approvalJSON, &aggregateVersion, &ledgerPosition, &auditID, &correlationID, &causationID, &submittedBy, &reversalOf, &correctionReason,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return JournalEntrySnapshot{}, ErrJournalEntryNotFound
	}
	if err != nil {
		return JournalEntrySnapshot{}, mapPostingPostgresError(err)
	}
	scope, err := accountingscope.New(tenantID, legalEntityID, ledgerID, bookID, functionalCurrency)
	if err != nil {
		return JournalEntrySnapshot{}, ErrPostingReferenceInvalid
	}
	request := PostingRequest{
		ContractVersion: PostingContractVersion, RequestID: requestID, SourceContext: sourceContext, SourceAggregateType: sourceType, SourceAggregateID: sourceID, SourceVersion: sourceVersion,
		AccountingScope: scope, AccountingScopeID: storedScopeID, PostingDate: postingDate, FiscalPeriodID: periodID, PeriodStateVersion: periodVersion, PostingGateVersion: postingGateVersion,
		PostingPurpose: postingPurpose, AdjustmentPeriodIndicator: adjustmentPeriod, TransactionCurrency: transactionCurrency, Description: description,
		IdempotencyKey: idempotencyKey, RequestFingerprint: fingerprint, CorrelationID: correlationID.String(), CausationID: causationID.String(),
	}
	if authID != uuid.Nil {
		request.PostingAuthorizationID = &authID
	}
	if closeRunID != uuid.Nil {
		request.CloseRunID = &closeRunID
	}
	if reopenID != uuid.Nil {
		request.ReopenRequestID = &reopenID
	}
	if operationalReopenID != uuid.Nil {
		request.OperationalReopenRequestID = &operationalReopenID
	}
	if controlEpoch > 0 {
		request.ControlAuthorityEpoch = &controlEpoch
	}
	if len(conversionJSON) > 0 && string(conversionJSON) != "null" {
		var evidence PostingConversionEvidence
		if err := json.Unmarshal(conversionJSON, &evidence); err != nil {
			return JournalEntrySnapshot{}, err
		}
		request.ConversionEvidence = &evidence
	}
	var approval *PostingApproval
	if len(approvalJSON) > 0 && string(approvalJSON) != "null" {
		var value PostingApproval
		if err := json.Unmarshal(approvalJSON, &value); err != nil {
			return JournalEntrySnapshot{}, err
		}
		approval = &value
	}
	var reversalPointer *uuid.UUID
	if reversalOf != uuid.Nil {
		reversalPointer = &reversalOf
		request.ReversalOfJournalEntryID = &reversalOf
	}
	lines, err := repository.loadJournalLines(ctx, tx, scopeID, id)
	if err != nil {
		return JournalEntrySnapshot{}, err
	}
	journal := PostingJournal{
		ID: id, Number: journalNumber, Version: aggregateVersion, LedgerPosition: ledgerPosition, Status: status, Approval: approval,
		SourceReference: sourceReference(request), RequestFingerprint: fingerprint,
		GateEvidence:   PostingGateEvidence{FiscalPeriodID: periodID, PeriodStateVersion: periodVersion, PostingGateVersion: postingGateVersion, GateMode: gateMode, ReopenRequestID: cloneUUIDValue(request.ReopenRequestID), AuthorizationScope: append(json.RawMessage(nil), reopenScope...), AuthorizationExpiry: reopenExpiry, ControlAuthorityEpoch: cloneInt64Pointer(request.ControlAuthorityEpoch)},
		AuditReference: auditID, SubmittedByUserID: submittedBy, PostingPurpose: postingPurpose, PostingAuthorizationID: cloneUUIDValue(request.PostingAuthorizationID),
		ReversalOfJournalEntryID: reversalPointer, CorrectionReason: correctionReason, Lines: append([]PostingLine(nil), lines...),
	}
	request.Lines = append([]PostingLine(nil), lines...)
	return JournalEntrySnapshot{Journal: journal, Request: request, Lines: lines}, nil
}

func (repository *PostgresPostingRepository) loadJournalLines(ctx context.Context, tx pgx.Tx, scopeID, journalID uuid.UUID) ([]PostingLine, error) {
	rows, err := tx.Query(ctx, `SELECT account_id, debit_or_credit, line_currency_mode, transaction_amount::text, functional_amount::text, segment_combination_id, line_reference FROM gl.journal_entry_line WHERE journal_entry_id = $1 AND accounting_scope_id = $2 ORDER BY line_number`, journalID, scopeID)
	if err != nil {
		return nil, mapPostingPostgresError(err)
	}
	defer rows.Close()
	var lines []PostingLine
	for rows.Next() {
		var line PostingLine
		if err := rows.Scan(&line.AccountID, &line.DebitOrCredit, &line.LineCurrencyMode, &line.TransactionAmount, &line.FunctionalAmount, &line.SegmentCombinationID, &line.LineReference); err != nil {
			return nil, err
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, mapPostingPostgresError(err)
	}
	return lines, nil
}

func (repository *PostgresPostingRepository) insertReversalAttempt(ctx context.Context, tx pgx.Tx, commit JournalReversalCommit, attemptID, journalID uuid.UUID, outcome string, issues []PostingIssue, now time.Time) error {
	issuesJSON := []byte("[]")
	if len(issues) > 0 {
		var err error
		issuesJSON, err = json.Marshal(issues)
		if err != nil {
			return err
		}
	}
	var journalValue any
	if journalID != uuid.Nil {
		journalValue = journalID
	}
	_, err := tx.Exec(ctx, `INSERT INTO gl.posting_attempt (posting_attempt_id, request_id, journal_entry_id, accounting_scope_id, source_context, source_aggregate_type, source_aggregate_id, source_version, idempotency_key, request_fingerprint, outcome, issues, actor_user_id, actor_subject_reference, correlation_id, causation_id, operation, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)`, attemptID, commit.Command.CommandID, journalValue, commit.Command.AccountingScopeID, commit.Command.SourceContext, commit.Command.SourceAggregateType, commit.Command.SourceAggregateID, commit.Command.SourceVersion, commit.Command.IdempotencyKey, commit.RequestFingerprint, outcome, issuesJSON, commit.Actor.UserID, commit.Actor.SubjectReference, commit.Command.CorrelationID, commit.Command.CausationID, JournalReversalOperation, now)
	return mapPostingPostgresError(err)
}

func journalReversalAuditRecord(commit JournalReversalCommit, attemptID, journalID uuid.UUID, outcome string, issues []PostingIssue) PostingAuditRecord {
	return PostingAuditRecord{
		AttemptID: attemptID, JournalID: journalID, ActorUserID: commit.Actor.UserID, ActorSubjectReference: commit.Actor.SubjectReference,
		Outcome: outcome, Operation: JournalReversalOperation, OriginalJournalID: commit.Command.OriginalJournalID, CorrectionReason: commit.Command.CorrectionReason,
		AccountingScope: commit.Original.Request.AccountingScope, AccountingScopeID: commit.Command.AccountingScopeID,
		SourceContext: commit.Command.SourceContext, SourceAggregateType: commit.Command.SourceAggregateType, SourceAggregateID: commit.Command.SourceAggregateID,
		SourceVersion: commit.Command.SourceVersion, RequestID: commit.Command.CommandID, IdempotencyKey: commit.Command.IdempotencyKey,
		RequestFingerprint: commit.RequestFingerprint, Permission: JournalReversalPermission, PolicyReference: commit.Authorization.PolicyReference,
		PolicyVersion: commit.Authorization.PolicyVersion, DecisionReference: commit.Authorization.DecisionReference, GateEvidence: commit.GateEvidence,
		CorrelationID: commit.Command.CorrelationID, CausationID: commit.Command.CausationID, Issues: issues,
	}
}

func (repository *PostgresPostingRepository) writeJournalReversalEvent(ctx context.Context, tx pgx.Tx, commit JournalReversalCommit, original, reversal JournalEntrySnapshot, auditReference uuid.UUID, now time.Time) error {
	payload, err := json.Marshal(struct {
		Outcome                  string              `json:"outcome"`
		OriginalJournalID        uuid.UUID           `json:"originalJournalId"`
		ReversalJournalID        uuid.UUID           `json:"reversalJournalId"`
		JournalNumber            string              `json:"journalNumber"`
		LifecycleStatus          string              `json:"lifecycleStatus"`
		ReversalOfJournalEntryID uuid.UUID           `json:"reversalOfJournalEntryId"`
		SourceReference          string              `json:"sourceReference"`
		CorrectionReason         string              `json:"correctionReason"`
		GateEvidence             PostingGateEvidence `json:"gateEvidence"`
		AuditReference           uuid.UUID           `json:"auditReference"`
	}{
		Outcome: PostingOutcomeJournalEntryReversed, OriginalJournalID: original.Journal.ID, ReversalJournalID: reversal.Journal.ID,
		JournalNumber: reversal.Journal.Number, LifecycleStatus: reversal.Journal.Status, ReversalOfJournalEntryID: original.Journal.ID,
		SourceReference: reversal.Journal.SourceReference, CorrectionReason: commit.Command.CorrectionReason, GateEvidence: commit.GateEvidence, AuditReference: auditReference,
	})
	if err != nil {
		return err
	}
	return repository.writeReversalEventEnvelope(ctx, tx, commit.Command, reversal.Journal.ID, reversal.Journal.Version, PostingOutcomeJournalEntryReversed, payload, now)
}

func (repository *PostgresPostingRepository) writeJournalReversalAttemptEvent(ctx context.Context, tx pgx.Tx, command ReverseJournalEntryCommand, outcome string, auditReference, correlationID, causationID uuid.UUID, issues []PostingIssue, now time.Time) error {
	payload, err := json.Marshal(struct {
		Outcome           string         `json:"outcome"`
		OriginalJournalID uuid.UUID      `json:"originalJournalId"`
		CorrectionReason  string         `json:"correctionReason,omitempty"`
		AuditReference    uuid.UUID      `json:"auditReference"`
		Issues            []PostingIssue `json:"issues,omitempty"`
	}{Outcome: outcome, OriginalJournalID: command.OriginalJournalID, CorrectionReason: command.CorrectionReason, AuditReference: auditReference, Issues: issues})
	if err != nil {
		return err
	}
	return repository.writeReversalEventEnvelope(ctx, tx, command, command.OriginalJournalID, 1, outcome, payload, now, correlationID, causationID)
}

func (repository *PostgresPostingRepository) writeReversalEventEnvelope(ctx context.Context, tx pgx.Tx, command ReverseJournalEntryCommand, aggregateID uuid.UUID, aggregateVersion int64, eventType string, payload []byte, now time.Time, trace ...uuid.UUID) error {
	correlationID, causationID := uuid.Nil, uuid.Nil
	if len(trace) == 2 {
		correlationID, causationID = trace[0], trace[1]
	} else {
		var err error
		correlationID, causationID, err = postingTraceUUIDs(PostingRequest{CorrelationID: command.CorrelationID, CausationID: command.CausationID})
		if err != nil {
			return err
		}
	}
	payloadFingerprint, err := platformevents.ComputePayloadFingerprint(payload)
	if err != nil {
		return err
	}
	event, err := platformevents.NewEnvelope(platformevents.EnvelopeInput{
		MessageID: uuid.NewString(), EventType: eventType, EventVersion: 1, OccurredAt: now, SourceContext: "general-ledger",
		AggregateID: aggregateID.String(), AggregateVersion: aggregateVersion, AccountingScopeID: command.AccountingScopeID.String(),
		CorrelationID: correlationID.String(), CausationID: causationID.String(), DataClassification: platformevents.Internal,
		PayloadFingerprint: payloadFingerprint, Data: payload,
	})
	if err != nil {
		return err
	}
	return platformintegration.WritePublication(ctx, tx, platformintegration.Publication{Event: event, AvailableAt: now})
}

func journalReversalResultFromSnapshots(original, reversal JournalEntrySnapshot, reason string, authorizationReference uuid.UUID, gateEvidence PostingGateEvidence, auditReference uuid.UUID, fingerprint string) JournalReversalResult {
	return JournalReversalResult{
		Outcome: PostingOutcomeJournalEntryReversed, Original: cloneJournalEntrySnapshot(original), Reversal: cloneJournalEntrySnapshot(reversal),
		CorrectionReason: reason, AuthorizationReference: authorizationReference, GateEvidence: gateEvidence, AuditReference: auditReference,
		NextAction: "No action required", RequestFingerprint: fingerprint,
	}
}

var _ JournalReversalRepository = (*PostgresPostingRepository)(nil)
