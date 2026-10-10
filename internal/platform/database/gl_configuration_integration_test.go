//go:build integration

package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/gl"
)

func TestGLConfigurationRepositoryPreservesRevisionsAndRollsBackFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	var glTable, publicTable bool
	if err := fixture.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'gl' AND table_name = 'ledger'
		)`).Scan(&glTable); err != nil {
		t.Fatal(err)
	}
	if err := fixture.pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables
			WHERE table_schema = 'public' AND table_name = 'ledger'
		)`).Scan(&publicTable); err != nil {
		t.Fatal(err)
	}
	if !glTable || publicTable {
		t.Fatalf("gl ledger=%t, public ledger=%t", glTable, publicTable)
	}

	scopeID := uuid.New()
	legalEntityID := uuid.New()
	fiscalCalendarID := uuid.New()
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	createdAt := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	ledger, err := gl.NewLedger(
		uuid.New(), scopeID, legalEntityID, "primary", "USD", fiscalCalendarID,
		gl.LedgerStatusDraft, from, nil, nil, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}

	ledgerAuditReference := uuid.New()
	bookAuditReference := uuid.New()
	repository, err := gl.NewPostgresConfigurationRepository(
		fixture.pool,
		func(context.Context, pgx.Tx, gl.LedgerAuditRecord) (uuid.UUID, error) {
			return ledgerAuditReference, nil
		},
		func(context.Context, pgx.Tx, gl.AccountingBookAuditRecord) (uuid.UUID, error) {
			return bookAuditReference, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	ledgerAudit := gl.LedgerAuditRecord{
		LedgerID:              ledger.ID,
		ActorUserID:           uuid.New(),
		ActorSubjectReference: "integration-actor",
		Action:                gl.LedgerActionCreate,
		AccountingScopeID:     scopeID,
		LegalEntityID:         legalEntityID,
		Permission:            gl.LedgerManagementPermission,
		DecisionReference:     uuid.New(),
		RevisionNumber:        ledger.RevisionNumber,
	}
	if err := repository.CommitLedgerMutation(ctx, gl.LedgerMutation{After: ledger, Audit: ledgerAudit}); err != nil {
		t.Fatal(err)
	}

	stored, err := repository.GetLedger(ctx, ledger.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Version.Value() != 1 || len(stored.Revisions) != 1 || stored.LastAuditReference != ledgerAuditReference {
		t.Fatalf("stored ledger = %#v, want version 1, one revision, and audit reference", stored)
	}

	updated := stored
	if err := updated.Replace(stored, gl.LedgerCommand{
		Action:             gl.LedgerActionUpdate,
		LedgerID:           stored.ID,
		AccountingScopeID:  scopeID,
		LegalEntityID:      legalEntityID,
		LedgerType:         "primary",
		FunctionalCurrency: "USD",
		FiscalCalendarID:   fiscalCalendarID,
		LifecycleStatus:    gl.LedgerStatusActive,
		EffectiveDateFrom:  from,
	}, time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	expectedVersion := stored.Version
	if err := repository.CommitLedgerMutation(ctx, gl.LedgerMutation{
		Before:          stored,
		After:           updated,
		ExpectedVersion: &expectedVersion,
		Audit: gl.LedgerAuditRecord{
			LedgerID:              updated.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "integration-actor",
			Action:                gl.LedgerActionUpdate,
			AccountingScopeID:     scopeID,
			LegalEntityID:         legalEntityID,
			Permission:            gl.LedgerManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        updated.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}
	reloaded, err := repository.GetLedger(ctx, ledger.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.Version.Value() != 2 || reloaded.LifecycleStatus != gl.LedgerStatusActive || len(reloaded.Revisions) != 2 || reloaded.Revisions[0].Snapshot.LifecycleStatus != gl.LedgerStatusDraft {
		t.Fatalf("reloaded ledger = %#v, want active version 2 with preserved draft revision", reloaded)
	}

	book, err := gl.NewAccountingBook(
		uuid.New(), scopeID, ledger.ID, "statutory", "accrual", "posting-v1",
		gl.LedgerStatusActive, from, nil, nil, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitAccountingBookMutation(ctx, gl.AccountingBookMutation{
		After: book,
		Audit: gl.AccountingBookAuditRecord{
			AccountingBookID:      book.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "integration-actor",
			Action:                gl.LedgerActionCreate,
			AccountingScopeID:     scopeID,
			LedgerID:              ledger.ID,
			Permission:            gl.AccountingBookManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        book.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}
	storedBook, err := repository.GetAccountingBook(ctx, book.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedBook.LedgerID != ledger.ID || storedBook.Version.Value() != 1 || len(storedBook.Revisions) != 1 || storedBook.LastAuditReference != bookAuditReference {
		t.Fatalf("stored accounting book = %#v", storedBook)
	}

	invalidParent, err := gl.NewAccountingBook(
		uuid.New(), scopeID, uuid.New(), "management", "accrual", "posting-v1",
		gl.LedgerStatusDraft, from, nil, nil, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitAccountingBookMutation(ctx, gl.AccountingBookMutation{
		After: invalidParent,
		Audit: gl.AccountingBookAuditRecord{AccountingBookID: invalidParent.ID, ActorUserID: uuid.New(), ActorSubjectReference: "integration-actor", Action: gl.LedgerActionCreate, AccountingScopeID: scopeID, LedgerID: invalidParent.LedgerID, Permission: gl.AccountingBookManagementPermission, DecisionReference: uuid.New(), RevisionNumber: invalidParent.RevisionNumber},
	}); !errors.Is(err, gl.ErrAccountingBookReferenceInvalid) {
		t.Fatalf("invalid parent error = %v, want accounting-book reference invalid", err)
	}

	otherScopeLedger, err := gl.NewLedger(
		uuid.New(), uuid.New(), uuid.New(), "primary", "USD", uuid.New(),
		gl.LedgerStatusDraft, from, nil, nil, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitLedgerMutation(ctx, gl.LedgerMutation{
		After: otherScopeLedger,
		Audit: gl.LedgerAuditRecord{LedgerID: otherScopeLedger.ID, ActorUserID: uuid.New(), ActorSubjectReference: "integration-actor", Action: gl.LedgerActionCreate, AccountingScopeID: otherScopeLedger.AccountingScopeID, LegalEntityID: otherScopeLedger.LegalEntityID, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), RevisionNumber: otherScopeLedger.RevisionNumber},
	}); err != nil {
		t.Fatal(err)
	}

	mismatchedScopeBook, err := gl.NewAccountingBook(
		uuid.New(), scopeID, otherScopeLedger.ID, "management", "accrual", "posting-v1",
		gl.LedgerStatusDraft, from, nil, nil, createdAt,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitAccountingBookMutation(ctx, gl.AccountingBookMutation{
		After: mismatchedScopeBook,
		Audit: gl.AccountingBookAuditRecord{AccountingBookID: mismatchedScopeBook.ID, ActorUserID: uuid.New(), ActorSubjectReference: "integration-actor", Action: gl.LedgerActionCreate, AccountingScopeID: scopeID, LedgerID: mismatchedScopeBook.LedgerID, Permission: gl.AccountingBookManagementPermission, DecisionReference: uuid.New(), RevisionNumber: mismatchedScopeBook.RevisionNumber},
	}); !errors.Is(err, gl.ErrAccountingBookReferenceInvalid) {
		t.Fatalf("mismatched parent scope error = %v, want accounting-book reference invalid", err)
	}

	failingRepository, err := gl.NewPostgresConfigurationRepository(
		fixture.pool,
		func(context.Context, pgx.Tx, gl.LedgerAuditRecord) (uuid.UUID, error) {
			return uuid.Nil, errors.New("audit unavailable")
		},
		func(context.Context, pgx.Tx, gl.AccountingBookAuditRecord) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := gl.NewLedger(uuid.New(), scopeID, uuid.New(), "secondary", "EUR", uuid.New(), gl.LedgerStatusDraft, from, nil, nil, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := failingRepository.CommitLedgerMutation(ctx, gl.LedgerMutation{After: failed, Audit: gl.LedgerAuditRecord{LedgerID: failed.ID, ActorUserID: uuid.New(), ActorSubjectReference: "integration-actor", Action: gl.LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: failed.LegalEntityID, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), RevisionNumber: failed.RevisionNumber}}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var count int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM gl.ledger WHERE ledger_id = $1`, failed.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("audit failure left %d ledger rows", count)
	}
}
