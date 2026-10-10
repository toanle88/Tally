//go:build integration

package database

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
	"github.com/toanle88/Tally/internal/platform/money"
)

func TestGLPostingRepositoryPostsAtomicallyAndSerializesGateAdmission(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	request, references := integrationPostingRequest()
	if err := insertPostingReferenceFixture(ctx, fixture.pool, references); err != nil {
		t.Fatal(err)
	}

	var auditCalls int32
	auditReference := uuid.New()
	repository, err := gl.NewPostgresPostingRepository(fixture.pool, func(context.Context, pgx.Tx, gl.PostingAuditRecord) (uuid.UUID, error) {
		atomic.AddInt32(&auditCalls, 1)
		return auditReference, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	service := integrationPostingService(t, repository)
	actor := gl.Actor{UserID: uuid.New(), SubjectReference: "posting-integration-actor"}

	first, err := service.Execute(ctx, actor, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Outcome != gl.PostingOutcomeJournalEntryPosted || first.LifecycleStatus() != gl.PostingStatusPosted || first.Journal == nil || first.Journal.LedgerPosition != 1 {
		t.Fatalf("first result = %#v", first)
	}
	assertPostingCounts(t, ctx, fixture, 1, 2, 1, 1, 2)
	if atomic.LoadInt32(&auditCalls) != 1 {
		t.Fatalf("audit calls = %d, want 1", auditCalls)
	}

	replay, err := service.Execute(ctx, actor, request)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Replayed || replay.AggregateID() != first.AggregateID() || replay.GateEvidence.GateMode != gl.PostingGateOpen {
		t.Fatalf("replay = %#v", replay)
	}
	assertPostingCounts(t, ctx, fixture, 1, 2, 1, 1, 2)

	changed := request
	changed.Description = "changed business content"
	if _, err := service.Execute(ctx, actor, changed); !errors.Is(err, gl.ErrPostingIdempotencyConflict) {
		t.Fatalf("changed retry error = %v, want idempotency conflict", err)
	}
	duplicateSource := request
	duplicateSource.IdempotencyKey = "posting-source-duplicate"
	duplicateSource.Description = "different source content"
	if _, err := service.Execute(ctx, actor, duplicateSource); !errors.Is(err, gl.ErrPostingSourceDuplicate) {
		t.Fatalf("duplicate source error = %v, want source duplicate", err)
	}
	assertPostingCounts(t, ctx, fixture, 1, 2, 3, 3, 2)

	concurrentOne := request
	concurrentOne.RequestID = uuid.New()
	concurrentOne.SourceAggregateID = uuid.New()
	concurrentOne.IdempotencyKey = "posting-concurrent-1"
	concurrentTwo := request
	concurrentTwo.RequestID = uuid.New()
	concurrentTwo.SourceAggregateID = uuid.New()
	concurrentTwo.IdempotencyKey = "posting-concurrent-2"
	errs := make(chan error, 2)
	var waitGroup sync.WaitGroup
	for _, candidate := range []gl.PostingRequest{concurrentOne, concurrentTwo} {
		candidate := candidate
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			_, executeErr := service.Execute(ctx, actor, candidate)
			errs <- executeErr
		}()
	}
	waitGroup.Wait()
	close(errs)
	for executeErr := range errs {
		if executeErr != nil {
			t.Fatal(executeErr)
		}
	}
	assertPostingCounts(t, ctx, fixture, 3, 6, 5, 5, 4)

	if _, err := fixture.pool.Exec(ctx, `UPDATE gl.journal_entry SET description = 'tampered' WHERE journal_entry_id = $1`, first.AggregateID()); err == nil {
		t.Fatal("posted journal mutation succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, `UPDATE gl.journal_entry SET functional_currency = 'EUR' WHERE journal_entry_id = $1`, first.AggregateID()); err == nil {
		t.Fatal("posted journal scope mutation succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, `UPDATE gl.journal_entry_line SET line_reference = 'tampered' WHERE journal_entry_id = $1 AND line_number = 1`, first.AggregateID()); err == nil {
		t.Fatal("posted journal-line mutation succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, `INSERT INTO gl.journal_entry_line (journal_entry_id, accounting_scope_id, line_number, account_id, debit_or_credit, line_currency_mode, transaction_amount, functional_amount, segment_combination_id, line_reference) VALUES ($1,$2,99,$3,'debit','TransactionAndFunctional',0,0,$4,'tampered')`, first.AggregateID(), references.scopeID, references.accountIDs[0], uuid.New()); err == nil {
		t.Fatal("posted journal-line insertion succeeded")
	}
	if _, err := fixture.pool.Exec(ctx, `DELETE FROM gl.journal_entry WHERE journal_entry_id = $1`, first.AggregateID()); err == nil {
		t.Fatal("posted journal deletion succeeded")
	}

	failingRepository, err := gl.NewPostgresPostingRepository(fixture.pool, func(context.Context, pgx.Tx, gl.PostingAuditRecord) (uuid.UUID, error) {
		return uuid.Nil, errors.New("audit write failed")
	})
	if err != nil {
		t.Fatal(err)
	}
	failingService := integrationPostingService(t, failingRepository)
	failingRequest := request
	failingRequest.RequestID = uuid.New()
	failingRequest.SourceAggregateID = uuid.New()
	failingRequest.IdempotencyKey = "posting-audit-failure"
	if _, err := failingService.Execute(ctx, actor, failingRequest); err == nil || err.Error() != "audit write failed" {
		t.Fatalf("audit failure = %v, want audit write failure", err)
	}
	assertPostingCounts(t, ctx, fixture, 3, 6, 5, 5, 4)
}

type integrationPostingReferences struct {
	scopeID          uuid.UUID
	tenantID         uuid.UUID
	legalEntityID    uuid.UUID
	ledgerID         uuid.UUID
	accountingBookID uuid.UUID
	chartID          uuid.UUID
	accountIDs       [2]uuid.UUID
	periodID         uuid.UUID
}

func integrationPostingRequest() (gl.PostingRequest, integrationPostingReferences) {
	references := integrationPostingReferences{
		scopeID: uuid.New(), tenantID: uuid.New(), legalEntityID: uuid.New(), ledgerID: uuid.New(), accountingBookID: uuid.New(), chartID: uuid.New(), accountIDs: [2]uuid.UUID{uuid.New(), uuid.New()}, periodID: uuid.New(),
	}
	scope, _ := accountingscope.New(references.tenantID, references.legalEntityID, references.ledgerID, references.accountingBookID, "USD")
	return gl.PostingRequest{
		ContractVersion: gl.PostingContractVersion, RequestID: uuid.New(), SourceContext: "accounts-payable", SourceAggregateType: "vendor-invoice", SourceAggregateID: uuid.New(), SourceVersion: 1,
		AccountingScope: scope, AccountingScopeID: references.scopeID, PostingDate: time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC), FiscalPeriodID: references.periodID, PeriodStateVersion: 1, PostingGateVersion: 1, PostingPurpose: gl.PostingPurposeOrdinary,
		TransactionCurrency: "USD", IdempotencyKey: "posting-integration-1", CorrelationID: uuid.NewString(), CausationID: uuid.NewString(), Description: "Integration posting",
		Lines: []gl.PostingLine{
			{AccountID: references.accountIDs[0], DebitOrCredit: gl.PostingDebit, LineCurrencyMode: gl.PostingLineTransactionAndFunctional, TransactionAmount: "100.00", FunctionalAmount: "100.00", SegmentCombinationID: uuid.New()},
			{AccountID: references.accountIDs[1], DebitOrCredit: gl.PostingCredit, LineCurrencyMode: gl.PostingLineTransactionAndFunctional, TransactionAmount: "100.00", FunctionalAmount: "100.00", SegmentCombinationID: uuid.New()},
		},
	}, references
}

func insertPostingReferenceFixture(ctx context.Context, pool *pgxpool.Pool, references integrationPostingReferences) error {
	now := time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC)
	effectiveFrom := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	fiscalCalendarID := uuid.New()
	if _, err := pool.Exec(ctx, `INSERT INTO gl.ledger (ledger_id, accounting_scope_id, legal_entity_id, ledger_type, functional_currency, fiscal_calendar_id, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference) VALUES ($1,$2,$3,'primary','USD',$4,'active',$5,NULL,NULL,1,1,$6,$6,NULL)`, references.ledgerID, references.scopeID, references.legalEntityID, fiscalCalendarID, effectiveFrom, now); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `INSERT INTO gl.accounting_book (accounting_book_id, accounting_scope_id, ledger_id, book_type, accounting_basis, posting_policy_version, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference) VALUES ($1,$2,$3,'statutory','accrual','posting-v1','active',$4,NULL,NULL,1,1,$5,$5,NULL)`, references.accountingBookID, references.scopeID, references.ledgerID, effectiveFrom, now); err != nil {
		return err
	}
	if _, err := pool.Exec(ctx, `INSERT INTO gl.chart_of_accounts (chart_of_accounts_id, accounting_scope_id, ledger_id, account_code_policy, lifecycle_status, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference) VALUES ($1,$2,$3,'NNNN','active',$4,NULL,NULL,1,1,$5,$5,NULL)`, references.chartID, references.scopeID, references.ledgerID, effectiveFrom, now); err != nil {
		return err
	}
	for index, accountID := range references.accountIDs {
		accountCode, accountType, normalBalance := "1000", "asset", "debit"
		if index == 1 {
			accountCode, accountType, normalBalance = "4000", "revenue", "credit"
		}
		if _, err := pool.Exec(ctx, `INSERT INTO gl.account (account_id, accounting_scope_id, chart_of_accounts_id, account_code, account_name, account_type, normal_balance, lifecycle_status, restrictions, currency_policy, reporting_mappings, effective_from, effective_to, approval_reference, aggregate_version, revision_number, created_at, updated_at, last_audit_reference) VALUES ($1,$2,$3,$4,$5,$6,$7,'active','[]'::jsonb,'functional-or-transaction','[]'::jsonb,$8,NULL,NULL,1,1,$9,$9,NULL)`, accountID, references.scopeID, references.chartID, accountCode, "Posting account "+accountCode, accountType, normalBalance, effectiveFrom, now); err != nil {
			return err
		}
	}
	_, err := pool.Exec(ctx, `INSERT INTO gl.period_posting_gate (accounting_scope_id, fiscal_period_id, gate_mode, period_state_version, gate_version, next_ledger_position, created_at, updated_at) VALUES ($1,$2,'Open',1,1,1,$3,$3)`, references.scopeID, references.periodID, now)
	return err
}

func integrationPostingService(t *testing.T, repository gl.PostingRepository) *gl.PostingService {
	t.Helper()
	registry, err := money.NewCurrencyRegistry([]money.CurrencyMetadata{{Code: "USD", Scale: 2}})
	if err != nil {
		t.Fatal(err)
	}
	service, err := gl.NewPostingService(
		repository,
		gl.MemoryPostingAuthorizer{Decision: gl.PostingAuthorizationDecision{Allowed: true, PolicyReference: "integration-policy", PolicyVersion: "integration-v1"}},
		gl.AllowAllPostingReferenceValidator{}, gl.MemoryPostingGateValidator{}, gl.AllowAllPostingApprovalValidator{}, &gl.MemoryPostingAuditRecorder{}, registry, func() time.Time { return time.Date(2026, 8, 15, 8, 0, 0, 0, time.UTC) },
	)
	if err != nil {
		t.Fatal(err)
	}
	return service
}

func assertPostingCounts(t *testing.T, ctx context.Context, fixture *integrationFixture, journals, lines, attempts, outbox, nextPosition int) {
	t.Helper()
	var journalCount, lineCount, attemptCount, outboxCount, nextLedgerPosition int
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM gl.journal_entry`).Scan(&journalCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM gl.journal_entry_line`).Scan(&lineCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM gl.posting_attempt`).Scan(&attemptCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT COUNT(*) FROM integration.outbox`).Scan(&outboxCount); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `SELECT next_ledger_position FROM gl.period_posting_gate LIMIT 1`).Scan(&nextLedgerPosition); err != nil {
		t.Fatal(err)
	}
	if journalCount != journals || lineCount != lines || attemptCount != attempts || outboxCount != outbox || nextLedgerPosition != nextPosition {
		t.Fatalf("posting counts journals=%d lines=%d attempts=%d outbox=%d next=%d, want %d/%d/%d/%d/%d", journalCount, lineCount, attemptCount, outboxCount, nextLedgerPosition, journals, lines, attempts, outbox, nextPosition)
	}
}
