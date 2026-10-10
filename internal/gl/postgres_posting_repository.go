package gl

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	platformevents "github.com/toanle88/Tally/internal/platform/events"
	platformintegration "github.com/toanle88/Tally/internal/platform/integration"
)

type PostgresPostingAuditWriter func(context.Context, pgx.Tx, PostingAuditRecord) (uuid.UUID, error)

type PostgresPostingRepository struct {
	pool        *pgxpool.Pool
	auditWriter PostgresPostingAuditWriter
}

func NewPostgresPostingRepository(pool *pgxpool.Pool, auditWriter PostgresPostingAuditWriter) (*PostgresPostingRepository, error) {
	if pool == nil || auditWriter == nil {
		return nil, ErrInvalidPostingService
	}
	return &PostgresPostingRepository{pool: pool, auditWriter: auditWriter}, nil
}

func (repository *PostgresPostingRepository) CommitPosting(ctx context.Context, commit PostingCommit) (PostingResult, error) {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil {
		return PostingResult{}, ErrInvalidPostingService
	}
	correlationID, causationID, err := postingTraceUUIDs(commit.Request)
	if err != nil {
		return PostingResult{}, err
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return PostingResult{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	gate, err := repository.lockGate(ctx, tx, commit.Request)
	if err != nil {
		return PostingResult{}, err
	}
	if err := validatePostingGateEvidence(commit.Request, gate.Evidence); err != nil {
		return PostingResult{}, err
	}
	if existing, found, err := repository.findJournalByIdempotency(ctx, tx, commit.Request); err != nil {
		return PostingResult{}, err
	} else if found {
		if existing.Journal == nil || existing.Journal.RequestFingerprint != commit.Request.RequestFingerprint {
			return PostingResult{}, ErrPostingIdempotencyConflict
		}
		existing = withCurrentPostingGate(existing, gate.Evidence)
		existing.Replayed = true
		return existing, nil
	}
	if existing, found, err := repository.findJournalBySource(ctx, tx, commit.Request); err != nil {
		return PostingResult{}, err
	} else if found {
		if existing.Journal == nil || existing.Journal.RequestFingerprint != commit.Request.RequestFingerprint {
			return PostingResult{}, ErrPostingSourceDuplicate
		}
		existing = withCurrentPostingGate(existing, gate.Evidence)
		existing.Replayed = true
		return existing, nil
	}
	if err := repository.validateReferences(ctx, tx, commit.Request); err != nil {
		return PostingResult{}, err
	}

	now := commit.Now.UTC()
	journalID := uuid.New()
	status := PostingStatusPosted
	outcome := PostingOutcomeJournalEntryPosted
	approvalStatus := "not_required"
	nextAction := "No action required"
	var approvalJSON []byte
	if commit.Approval.Required {
		status = PostingStatusPendingApproval
		outcome = PostingOutcomePostingPendingApproval
		approvalStatus = "pending"
		nextAction = "Approval decision required"
		approvalJSON, err = json.Marshal(commit.Approval)
		if err != nil {
			return PostingResult{}, err
		}
	}
	conversionJSON, err := json.Marshal(commit.Request.ConversionEvidence)
	if err != nil {
		return PostingResult{}, err
	}
	var ledgerPosition *int64
	journalNumber := "PJE-" + journalID.String()[:8]
	if status == PostingStatusPosted {
		position := gate.LedgerPosition
		ledgerPosition = &position
		journalNumber = fmt.Sprintf("JE-%08d", position)
		if _, err := tx.Exec(ctx, `UPDATE gl.period_posting_gate SET next_ledger_position = next_ledger_position + 1, updated_at = $3 WHERE accounting_scope_id = $1 AND fiscal_period_id = $2`, commit.Request.AccountingScopeID, commit.Request.FiscalPeriodID, now); err != nil {
			return PostingResult{}, mapPostingPostgresError(err)
		}
	}
	storedStatus := status
	if status == PostingStatusPosted {
		// The journal line immutability trigger must reject direct writes to an
		// established journal, while still allowing this transaction to build
		// the complete journal before its final lifecycle transition.
		storedStatus = PostingStatusPendingApproval
	}
	attemptID := uuid.New()
	auditReference, err := repository.auditWriter(ctx, tx, postingAuditRecord(commit, attemptID, journalID, outcome, nil))
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return PostingResult{}, err
		}
		return PostingResult{}, ErrPostingAuditUnavailable
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO gl.journal_entry (
			journal_entry_id, journal_number, accounting_scope_id, tenant_id, legal_entity_id, ledger_id, accounting_book_id,
			functional_currency, source_context, source_aggregate_type, source_aggregate_id, source_version, request_id,
			idempotency_key, request_fingerprint, posting_date, fiscal_period_id, period_state_version, posting_gate_version,
			posting_purpose, adjustment_period_indicator, posting_authorization_id, close_run_id, reopen_request_id, operational_reopen_request_id, control_authority_epoch,
			transaction_currency, conversion_evidence, description, lifecycle_status, approval_reference, aggregate_version,
			ledger_position, audit_reference, correlation_id, causation_id, created_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23,$24,$25,$26,$27,$28,$29,$30,$31,$32,$33,$34,$35,$36,$37)`,
		journalID, journalNumber, commit.Request.AccountingScopeID, commit.Request.AccountingScope.TenantID(), commit.Request.AccountingScope.LegalEntityID(), commit.Request.AccountingScope.LedgerID(), commit.Request.AccountingScope.AccountingBookID(),
		commit.Request.AccountingScope.FunctionalCurrency(), commit.Request.SourceContext, commit.Request.SourceAggregateType, commit.Request.SourceAggregateID, commit.Request.SourceVersion, commit.Request.RequestID,
		commit.Request.IdempotencyKey, commit.Request.RequestFingerprint, commit.Request.PostingDate, commit.Request.FiscalPeriodID, commit.Request.PeriodStateVersion, commit.Request.PostingGateVersion,
		commit.Request.PostingPurpose, commit.Request.AdjustmentPeriodIndicator, nullablePostingUUID(commit.Request.PostingAuthorizationID), nullablePostingUUID(commit.Request.CloseRunID), nullablePostingUUID(commit.Request.ReopenRequestID), nullablePostingUUID(commit.Request.OperationalReopenRequestID), nullablePostingInt64(commit.Request.ControlAuthorityEpoch),
		commit.Request.TransactionCurrency, nullablePostingJSON(conversionJSON), commit.Request.Description, storedStatus, nullablePostingJSON(approvalJSON), 1,
		ledgerPosition, auditReference, correlationID, causationID, now)
	if err != nil {
		return PostingResult{}, mapPostingPostgresError(err)
	}
	for index, line := range commit.Request.Lines {
		_, err = tx.Exec(ctx, `INSERT INTO gl.journal_entry_line (journal_entry_id, accounting_scope_id, line_number, account_id, debit_or_credit, line_currency_mode, transaction_amount, functional_amount, segment_combination_id, line_reference) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`, journalID, commit.Request.AccountingScopeID, index+1, line.AccountID, line.DebitOrCredit, line.LineCurrencyMode, line.TransactionAmount, line.FunctionalAmount, line.SegmentCombinationID, line.LineReference)
		if err != nil {
			return PostingResult{}, mapPostingPostgresError(err)
		}
	}
	if status == PostingStatusPosted {
		if _, err := tx.Exec(ctx, `UPDATE gl.journal_entry SET lifecycle_status = $2 WHERE journal_entry_id = $1`, journalID, status); err != nil {
			return PostingResult{}, mapPostingPostgresError(err)
		}
	}
	result := PostingResult{
		Outcome: outcome, ApprovalRequestID: commit.Approval.ApprovalRequestID, NextAction: nextAction,
		ValidationOutcome: "valid", ApprovalStatus: approvalStatus, SourceReference: sourceReference(commit.Request),
		GateEvidence: gate.Evidence, AuditReference: auditReference,
		Journal: &PostingJournal{ID: journalID, Number: journalNumber, Version: 1, LedgerPosition: valueOrZero(ledgerPosition), Status: status, SourceReference: sourceReference(commit.Request), RequestFingerprint: commit.Request.RequestFingerprint, GateEvidence: gate.Evidence, AuditReference: auditReference},
	}
	if commit.Approval.Required {
		approval := commit.Approval
		result.Journal.Approval = &approval
	}
	if err := repository.insertAttempt(ctx, tx, commit, attemptID, journalID, outcome, nil, now); err != nil {
		return PostingResult{}, err
	}
	if err := repository.writeEvent(ctx, tx, commit, result, now); err != nil {
		return PostingResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return PostingResult{}, err
	}
	committed = true
	return result, nil
}

func (repository *PostgresPostingRepository) RecordPostingAttempt(ctx context.Context, request PostingRequest, actor Actor, outcome string, issues []PostingIssue) error {
	if repository == nil || repository.pool == nil || repository.auditWriter == nil {
		return ErrInvalidPostingService
	}
	if _, _, err := postingTraceUUIDs(request); err != nil {
		return err
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
	now := time.Now().UTC()
	attemptID := uuid.New()
	if request.RequestFingerprint == "" {
		request.RequestFingerprint, _ = postingRequestFingerprint(request)
	}
	commit := PostingCommit{Request: request, Actor: actor, Now: now}
	auditReference, err := repository.auditWriter(ctx, tx, postingAuditRecord(commit, attemptID, uuid.Nil, outcome, issues))
	if err != nil || auditReference == uuid.Nil {
		if err != nil {
			return err
		}
		return ErrPostingAuditUnavailable
	}
	if err := repository.insertAttempt(ctx, tx, commit, attemptID, uuid.Nil, outcome, issues, now); err != nil {
		return err
	}
	result := PostingResult{Outcome: outcome, Issues: append([]PostingIssue(nil), issues...), AuditReference: auditReference}
	if err := repository.writeEvent(ctx, tx, commit, result, now); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	committed = true
	return nil
}

func (repository *PostgresPostingRepository) ResolveExistingPosting(ctx context.Context, request PostingRequest) (PostingResult, bool, error) {
	if repository == nil || repository.pool == nil {
		return PostingResult{}, false, ErrInvalidPostingService
	}
	tx, err := repository.pool.Begin(ctx)
	if err != nil {
		return PostingResult{}, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	existing, found, err := repository.findJournalByIdempotency(ctx, tx, request)
	foundByIdempotency := found
	if err != nil {
		return PostingResult{}, false, err
	}
	if !found {
		existing, found, err = repository.findJournalBySource(ctx, tx, request)
		if err != nil {
			return PostingResult{}, false, err
		}
	}
	if !found {
		if err := tx.Commit(ctx); err != nil {
			return PostingResult{}, false, err
		}
		committed = true
		return PostingResult{}, false, nil
	}
	if existing.Journal == nil || existing.Journal.RequestFingerprint != request.RequestFingerprint {
		if foundByIdempotency {
			return PostingResult{}, false, ErrPostingIdempotencyConflict
		}
		return PostingResult{}, false, ErrPostingSourceDuplicate
	}
	if gate, gateErr := repository.lockGate(ctx, tx, request); gateErr == nil {
		existing = withCurrentPostingGate(existing, gate.Evidence)
	} else if !errors.Is(gateErr, ErrPostingReferenceUnavailable) {
		return PostingResult{}, false, gateErr
	}
	if err := tx.Commit(ctx); err != nil {
		return PostingResult{}, false, err
	}
	committed = true
	existing.Replayed = true
	return existing, true, nil
}

type postgresPostingGate struct {
	Evidence       PostingGateEvidence
	LedgerPosition int64
}

func (repository *PostgresPostingRepository) lockGate(ctx context.Context, tx pgx.Tx, request PostingRequest) (postgresPostingGate, error) {
	var gate postgresPostingGate
	err := tx.QueryRow(ctx, `SELECT gate_mode, period_state_version, gate_version, next_ledger_position FROM gl.period_posting_gate WHERE accounting_scope_id = $1 AND fiscal_period_id = $2 FOR UPDATE`, request.AccountingScopeID, request.FiscalPeriodID).Scan(&gate.Evidence.GateMode, &gate.Evidence.PeriodStateVersion, &gate.Evidence.PostingGateVersion, &gate.LedgerPosition)
	if errors.Is(err, pgx.ErrNoRows) {
		return postgresPostingGate{}, ErrPostingReferenceUnavailable
	}
	if err != nil {
		return postgresPostingGate{}, mapPostingPostgresError(err)
	}
	gate.Evidence.FiscalPeriodID = request.FiscalPeriodID
	return gate, nil
}

func (repository *PostgresPostingRepository) findJournalByIdempotency(ctx context.Context, tx pgx.Tx, request PostingRequest) (PostingResult, bool, error) {
	return repository.findJournal(ctx, tx, `accounting_scope_id = $1 AND idempotency_key = $2`, request.AccountingScopeID, request.IdempotencyKey)
}

func (repository *PostgresPostingRepository) findJournalBySource(ctx context.Context, tx pgx.Tx, request PostingRequest) (PostingResult, bool, error) {
	return repository.findJournal(ctx, tx, `accounting_scope_id = $1 AND source_context = $2 AND source_aggregate_type = $3 AND source_aggregate_id = $4 AND source_version = $5`, request.AccountingScopeID, request.SourceContext, request.SourceAggregateType, request.SourceAggregateID, request.SourceVersion)
}

func (repository *PostgresPostingRepository) findJournal(ctx context.Context, tx pgx.Tx, predicate string, args ...any) (PostingResult, bool, error) {
	var (
		journalID, periodID, auditReference, sourceAggregateID                 uuid.UUID
		journalNumber, fingerprint, sourceContext, sourceAggregateType, status string
		version, position, periodVersion, gateVersion, sourceVersion           int64
		approvalJSON                                                           []byte
	)
	query := `SELECT journal_entry_id, journal_number, request_fingerprint, source_context, source_aggregate_type, source_aggregate_id, source_version, fiscal_period_id, period_state_version, posting_gate_version, lifecycle_status, approval_reference, aggregate_version, COALESCE(ledger_position, 0), audit_reference FROM gl.journal_entry WHERE ` + predicate + ` FOR UPDATE`
	err := tx.QueryRow(ctx, query, args...).Scan(&journalID, &journalNumber, &fingerprint, &sourceContext, &sourceAggregateType, &sourceAggregateID, &sourceVersion, &periodID, &periodVersion, &gateVersion, &status, &approvalJSON, &version, &position, &auditReference)
	if errors.Is(err, pgx.ErrNoRows) {
		return PostingResult{}, false, nil
	}
	if err != nil {
		return PostingResult{}, false, mapPostingPostgresError(err)
	}
	var approval PostingApproval
	if len(approvalJSON) > 0 && string(approvalJSON) != "null" {
		if err := json.Unmarshal(approvalJSON, &approval); err != nil {
			return PostingResult{}, false, err
		}
	}
	outcome := PostingOutcomeJournalEntryPosted
	nextAction := "No action required"
	approvalStatus := "not_required"
	if status == PostingStatusPendingApproval {
		outcome = PostingOutcomePostingPendingApproval
		nextAction = "Approval decision required"
		approvalStatus = "pending"
	}
	result := PostingResult{Outcome: outcome, ApprovalRequestID: approval.ApprovalRequestID, NextAction: nextAction, ValidationOutcome: "valid", ApprovalStatus: approvalStatus, SourceReference: fmt.Sprintf("%s:%s:%s:%d", sourceContext, sourceAggregateType, sourceAggregateID, sourceVersion), GateEvidence: PostingGateEvidence{FiscalPeriodID: periodID, PeriodStateVersion: periodVersion, PostingGateVersion: gateVersion, GateMode: "Open"}, AuditReference: auditReference}
	result.Journal = &PostingJournal{ID: journalID, Number: journalNumber, Version: version, LedgerPosition: position, Status: status, Approval: &approval, SourceReference: result.SourceReference, RequestFingerprint: fingerprint, GateEvidence: result.GateEvidence, AuditReference: auditReference}
	if approval.ApprovalRequestID == uuid.Nil {
		result.Journal.Approval = nil
	}
	return result, true, nil
}

func (repository *PostgresPostingRepository) validateReferences(ctx context.Context, tx pgx.Tx, request PostingRequest) error {
	var functionalCurrency string
	err := tx.QueryRow(ctx, `SELECT functional_currency FROM gl.ledger WHERE ledger_id = $1 AND accounting_scope_id = $2 AND legal_entity_id = $3 AND lifecycle_status = 'active' AND effective_from <= $4 AND (effective_to IS NULL OR effective_to >= $4) FOR SHARE`, request.AccountingScope.LedgerID(), request.AccountingScopeID, request.AccountingScope.LegalEntityID(), request.PostingDate).Scan(&functionalCurrency)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPostingReferenceInvalid
	}
	if err != nil {
		return mapPostingPostgresError(err)
	}
	if functionalCurrency != request.AccountingScope.FunctionalCurrency() {
		return ErrPostingReferenceInvalid
	}
	var bookID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT accounting_book_id FROM gl.accounting_book WHERE accounting_book_id = $1 AND accounting_scope_id = $2 AND ledger_id = $3 AND lifecycle_status = 'active' AND effective_from <= $4 AND (effective_to IS NULL OR effective_to >= $4) FOR SHARE`, request.AccountingScope.AccountingBookID(), request.AccountingScopeID, request.AccountingScope.LedgerID(), request.PostingDate).Scan(&bookID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrPostingReferenceInvalid
	}
	if err != nil {
		return mapPostingPostgresError(err)
	}
	for _, line := range request.Lines {
		var accountID uuid.UUID
		err = tx.QueryRow(ctx, `SELECT account.account_id FROM gl.account AS account JOIN gl.chart_of_accounts AS chart ON chart.chart_of_accounts_id = account.chart_of_accounts_id AND chart.accounting_scope_id = account.accounting_scope_id WHERE account.account_id = $1 AND account.accounting_scope_id = $2 AND chart.ledger_id = $3 AND account.lifecycle_status = 'active' AND chart.lifecycle_status = 'active' AND account.effective_from <= $4 AND (account.effective_to IS NULL OR account.effective_to >= $4) AND chart.effective_from <= $4 AND (chart.effective_to IS NULL OR chart.effective_to >= $4) FOR SHARE`, line.AccountID, request.AccountingScopeID, request.AccountingScope.LedgerID(), request.PostingDate).Scan(&accountID)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrPostingReferenceInvalid
		}
		if err != nil {
			return mapPostingPostgresError(err)
		}
	}
	return nil
}

func (repository *PostgresPostingRepository) insertAttempt(ctx context.Context, tx pgx.Tx, commit PostingCommit, attemptID, journalID uuid.UUID, outcome string, issues []PostingIssue, now time.Time) error {
	issuesJSON := []byte("[]")
	if len(issues) > 0 {
		var err error
		issuesJSON, err = json.Marshal(issues)
		if err != nil {
			return err
		}
	}
	correlationID, causationID, err := postingTraceUUIDs(commit.Request)
	if err != nil {
		return err
	}
	var journalValue any
	if journalID != uuid.Nil {
		journalValue = journalID
	}
	_, err = tx.Exec(ctx, `INSERT INTO gl.posting_attempt (posting_attempt_id, request_id, journal_entry_id, accounting_scope_id, source_context, source_aggregate_type, source_aggregate_id, source_version, idempotency_key, request_fingerprint, outcome, issues, actor_user_id, actor_subject_reference, correlation_id, causation_id, created_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, attemptID, commit.Request.RequestID, journalValue, commit.Request.AccountingScopeID, commit.Request.SourceContext, commit.Request.SourceAggregateType, commit.Request.SourceAggregateID, commit.Request.SourceVersion, commit.Request.IdempotencyKey, nullablePostingFingerprint(commit.Request.RequestFingerprint), outcome, issuesJSON, commit.Actor.UserID, commit.Actor.SubjectReference, correlationID, causationID, now)
	return mapPostingPostgresError(err)
}

func (repository *PostgresPostingRepository) writeEvent(ctx context.Context, tx pgx.Tx, commit PostingCommit, result PostingResult, now time.Time) error {
	aggregateID := result.AggregateID()
	if aggregateID == uuid.Nil {
		aggregateID = commit.Request.RequestID
	}
	aggregateVersion := result.AggregateVersion()
	if aggregateVersion < 1 {
		aggregateVersion = 1
	}
	payload, err := json.Marshal(struct {
		Outcome         string              `json:"outcome"`
		JournalID       uuid.UUID           `json:"journalId"`
		JournalNumber   string              `json:"journalNumber,omitempty"`
		LifecycleStatus string              `json:"lifecycleStatus"`
		SourceReference string              `json:"sourceReference"`
		GateEvidence    PostingGateEvidence `json:"gateEvidence"`
		Issues          []PostingIssue      `json:"issues,omitempty"`
	}{Outcome: result.Outcome, JournalID: result.AggregateID(), JournalNumber: journalNumber(result), LifecycleStatus: result.LifecycleStatus(), SourceReference: result.SourceReference, GateEvidence: result.GateEvidence, Issues: result.Issues})
	if err != nil {
		return err
	}
	payloadFingerprint, err := platformevents.ComputePayloadFingerprint(payload)
	if err != nil {
		return err
	}
	event, err := platformevents.NewEnvelope(platformevents.EnvelopeInput{MessageID: uuid.NewString(), EventType: result.Outcome, EventVersion: 1, OccurredAt: now, SourceContext: "general-ledger", AggregateID: aggregateID.String(), AggregateVersion: aggregateVersion, AccountingScopeID: commit.Request.AccountingScopeID.String(), CorrelationID: commit.Request.CorrelationID, CausationID: commit.Request.CausationID, DataClassification: platformevents.Internal, PayloadFingerprint: payloadFingerprint, Data: payload})
	if err != nil {
		return err
	}
	return platformintegration.WritePublication(ctx, tx, platformintegration.Publication{Event: event, AvailableAt: now})
}

func postingAuditRecord(commit PostingCommit, attemptID, journalID uuid.UUID, outcome string, issues []PostingIssue) PostingAuditRecord {
	return PostingAuditRecord{AttemptID: attemptID, JournalID: journalID, ActorUserID: commit.Actor.UserID, ActorSubjectReference: commit.Actor.SubjectReference, Outcome: outcome, AccountingScope: commit.Request.AccountingScope, AccountingScopeID: commit.Request.AccountingScopeID, SourceContext: commit.Request.SourceContext, SourceAggregateType: commit.Request.SourceAggregateType, SourceAggregateID: commit.Request.SourceAggregateID, SourceVersion: commit.Request.SourceVersion, RequestID: commit.Request.RequestID, IdempotencyKey: commit.Request.IdempotencyKey, RequestFingerprint: commit.Request.RequestFingerprint, Permission: PostingPermission, PolicyReference: commit.Authorization.PolicyReference, PolicyVersion: commit.Authorization.PolicyVersion, DecisionReference: commit.Authorization.DecisionReference, CorrelationID: commit.Request.CorrelationID, CausationID: commit.Request.CausationID, Issues: issues}
}

func nullablePostingJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nullablePostingFingerprint(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func nullablePostingUUID(value *uuid.UUID) any {
	if value == nil || *value == uuid.Nil {
		return nil
	}
	return *value
}

func nullablePostingInt64(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}

func journalNumber(result PostingResult) string {
	if result.Journal == nil {
		return ""
	}
	return result.Journal.Number
}

func withCurrentPostingGate(result PostingResult, evidence PostingGateEvidence) PostingResult {
	result.GateEvidence = evidence
	if result.Journal != nil {
		result.Journal.GateEvidence = evidence
	}
	return result
}

func postingTraceUUIDs(request PostingRequest) (uuid.UUID, uuid.UUID, error) {
	correlationID, err := uuid.Parse(request.CorrelationID)
	if err != nil || correlationID == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrInvalidPostingRequest
	}
	causationID, err := uuid.Parse(request.CausationID)
	if err != nil || causationID == uuid.Nil {
		return uuid.Nil, uuid.Nil, ErrInvalidPostingRequest
	}
	return correlationID, causationID, nil
}

func mapPostingPostgresError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.ConstraintName {
		case "journal_entry_idempotency_unique":
			return ErrPostingIdempotencyConflict
		case "journal_entry_source_unique":
			return ErrPostingSourceDuplicate
		case "journal_entry_ledger_fk", "journal_entry_book_fk", "journal_entry_line_account_fk":
			return ErrPostingReferenceInvalid
		case "journal_entry_line_amount_check", "journal_entry_line_functional_only_check":
			return ErrPostingValidation
		}
	}
	return err
}

var _ PostingRepository = (*PostgresPostingRepository)(nil)
