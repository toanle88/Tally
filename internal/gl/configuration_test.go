package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestLedgerServiceCreateUpdateReplayAndHistory(t *testing.T) {
	clock := func() time.Time { return time.Date(2026, 1, 2, 15, 4, 5, 0, time.FixedZone("ICT", 7*60*60)) }
	scopeID := uuid.New()
	authorizer := MemoryLedgerAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "policy/gl", PolicyVersion: "7"}}
	audit := &MemoryLedgerAuditRecorder{}
	repository := NewMemoryLedgerRepository()
	service, err := NewLedgerService(repository, authorizer, AllowAllReferenceValidator{}, AllowAllApprovalValidator{}, audit, clock)
	if err != nil {
		t.Fatal(err)
	}
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject-1"}
	create := LedgerCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: uuid.New(), LedgerType: "primary", FunctionalCurrency: "usd", FiscalCalendarID: uuid.New(), LifecycleStatus: LedgerStatusDraft, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "ledger-create-1", CorrelationID: "corr-1", CausationID: "command-1"}
	created, err := service.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if created.Ledger.Version.Value() != 1 || created.Ledger.RevisionNumber != 1 || created.Ledger.FunctionalCurrency != "USD" {
		t.Fatalf("created projection = %#v", created.Ledger)
	}
	replayed, err := service.Execute(context.Background(), actor, create)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.Ledger.ID != created.Ledger.ID {
		t.Fatalf("replay = %#v", replayed)
	}

	updated, err := service.Execute(context.Background(), actor, LedgerCommand{Action: LedgerActionUpdate, LedgerID: created.Ledger.ID, AccountingScopeID: scopeID, LegalEntityID: created.Ledger.LegalEntityID, LedgerType: "primary", FunctionalCurrency: "USD", FiscalCalendarID: created.Ledger.FiscalCalendarID, LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "ledger-update-1", ExpectedVersion: version(1)})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Ledger.Version.Value() != 2 || updated.Ledger.LifecycleStatus != LedgerStatusActive {
		t.Fatalf("updated projection = %#v", updated.Ledger)
	}
	stored, err := repository.GetLedger(context.Background(), created.Ledger.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(stored.Revisions) != 2 || len(audit.Records) != 2 {
		t.Fatalf("stored revisions=%d audit records=%d, want 2 each", len(stored.Revisions), len(audit.Records))
	}
	if _, err := service.Execute(context.Background(), actor, LedgerCommand{Action: LedgerActionUpdate, LedgerID: created.Ledger.ID, AccountingScopeID: scopeID, LegalEntityID: created.Ledger.LegalEntityID, LedgerType: "primary", FunctionalCurrency: "USD", FiscalCalendarID: created.Ledger.FiscalCalendarID, LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "ledger-stale-1", ExpectedVersion: version(1)}); !errors.Is(err, ErrLedgerVersionConflict) {
		t.Fatalf("stale update error = %v, want version conflict", err)
	}
}

func TestLedgerServiceRejectsDuplicateAndAuthorizationScope(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject-1"}
	newService := func(decision AuthorizationDecision) *LedgerService {
		service, err := NewLedgerService(NewMemoryLedgerRepository(), MemoryLedgerAuthorizer{Decision: decision}, AllowAllReferenceValidator{}, AllowAllApprovalValidator{}, &MemoryLedgerAuditRecorder{}, time.Now)
		if err != nil {
			t.Fatal(err)
		}
		return service
	}
	command := LedgerCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: uuid.New(), LedgerType: "primary", FunctionalCurrency: "USD", FiscalCalendarID: uuid.New(), LifecycleStatus: LedgerStatusDraft, EffectiveDateFrom: date(2026, 2, 1), IdempotencyKey: "dup-1"}
	service := newService(AuthorizationDecision{Allowed: true, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}})
	if _, err := service.Execute(context.Background(), actor, command); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(context.Background(), actor, LedgerCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: command.LegalEntityID, LedgerType: command.LedgerType, FunctionalCurrency: command.FunctionalCurrency, FiscalCalendarID: command.FiscalCalendarID, LifecycleStatus: LedgerStatusDraft, EffectiveDateFrom: command.EffectiveDateFrom, IdempotencyKey: "dup-2"}); !errors.Is(err, ErrLedgerDuplicate) {
		t.Fatalf("duplicate error = %v, want duplicate", err)
	}

	denied := newService(AuthorizationDecision{Allowed: true, Permission: "finance.other.permission", DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}})
	if _, err := denied.Execute(context.Background(), actor, command); !errors.Is(err, ErrLedgerAuthorizationDenied) {
		t.Fatalf("denied error = %v, want authorization denied", err)
	}
}

func TestAccountingBookServiceRequiresParentAndApproval(t *testing.T) {
	ctx := context.Background()
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "subject-1"}
	ledgerRepository := NewMemoryLedgerRepository()
	ledgerAudit := &MemoryLedgerAuditRecorder{}
	ledgerService, err := NewLedgerService(ledgerRepository, MemoryLedgerAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllReferenceValidator{}, AllowAllApprovalValidator{}, ledgerAudit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	ledgerResult, err := ledgerService.Execute(ctx, actor, LedgerCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: uuid.New(), LedgerType: "primary", FunctionalCurrency: "USD", FiscalCalendarID: uuid.New(), LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "book-parent-ledger"})
	if err != nil {
		t.Fatal(err)
	}
	bookRepository := NewMemoryAccountingBookRepository(ledgerRepository)
	bookAudit := &MemoryAccountingBookAuditRecorder{}
	bookService, err := NewAccountingBookService(bookRepository, MemoryAccountingBookAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: AccountingBookManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, ApprovalRequired: true}}, AllowAllReferenceValidator{}, AllowAllApprovalValidator{}, bookAudit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	base := AccountingBookCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LedgerID: ledgerResult.Ledger.ID, BookType: "statutory", AccountingBasis: "accrual", PostingPolicyVersion: "posting-v1", LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "book-create-1"}
	if _, err := bookService.Execute(ctx, actor, base); !errors.Is(err, ErrAccountingBookApprovalRequired) {
		t.Fatalf("missing approval error = %v, want approval required", err)
	}
	if _, err := bookService.Execute(ctx, actor, AccountingBookCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LedgerID: uuid.New(), BookType: "statutory", AccountingBasis: "accrual", PostingPolicyVersion: "posting-v1", LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), Approval: validApproval(), IdempotencyKey: "book-missing-parent"}); !errors.Is(err, ErrAccountingBookReferenceInvalid) {
		t.Fatalf("missing parent error = %v, want reference invalid", err)
	}
	base.Approval = validApproval()
	base.IdempotencyKey = "book-create-2"
	created, err := bookService.Execute(ctx, actor, base)
	if err != nil {
		t.Fatal(err)
	}
	if created.AccountingBook.LedgerID != ledgerResult.Ledger.ID || len(bookAudit.Records) != 1 {
		t.Fatalf("book result=%#v audit=%d", created.AccountingBook, len(bookAudit.Records))
	}
}

func TestLedgerServiceDoesNotPublishWhenAuditFails(t *testing.T) {
	scopeID := uuid.New()
	audit := &MemoryLedgerAuditRecorder{Err: errors.New("audit storage unavailable")}
	repository := NewMemoryLedgerRepository()
	service, err := NewLedgerService(repository, MemoryLedgerAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllReferenceValidator{}, AllowAllApprovalValidator{}, audit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "subject-1"}, LedgerCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: uuid.New(), LedgerType: "primary", FunctionalCurrency: "USD", FiscalCalendarID: uuid.New(), LifecycleStatus: LedgerStatusDraft, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "audit-failure-1"})
	if err == nil {
		t.Fatal("expected audit failure")
	}
	if _, getErr := repository.ListLedgers(context.Background(), &scopeID); getErr != nil {
		t.Fatal(getErr)
	} else if len(repositoryListMust(t, repository, scopeID)) != 0 {
		t.Fatal("ledger was persisted despite audit failure")
	}
}

func TestGLServicesFailClosedWhenWorkflowApprovalIsUnavailable(t *testing.T) {
	scopeID := uuid.New()
	command := LedgerCommand{
		Action:             LedgerActionCreate,
		AccountingScopeID:  scopeID,
		LegalEntityID:      uuid.New(),
		LedgerType:         "primary",
		FunctionalCurrency: "USD",
		FiscalCalendarID:   uuid.New(),
		LifecycleStatus:    LedgerStatusActive,
		EffectiveDateFrom:  date(2026, 1, 1),
		Approval:           validApproval(),
		IdempotencyKey:     "approval-unavailable-1",
	}
	repository := NewMemoryLedgerRepository()
	service, err := NewLedgerService(
		repository,
		MemoryLedgerAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}},
		AllowAllReferenceValidator{},
		UnavailableApprovalValidator{},
		&MemoryLedgerAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, command); !errors.Is(err, ErrLedgerApprovalUnavailable) {
		t.Fatalf("approval error = %v, want approval unavailable", err)
	}
	if ledgers := repositoryListMust(t, repository, scopeID); len(ledgers) != 0 {
		t.Fatalf("ledger count = %d, want no persisted ledger", len(ledgers))
	}
}

func TestLedgerServiceReportsRequiredBeforeCallingUnavailableApprovalValidator(t *testing.T) {
	scopeID := uuid.New()
	service, err := NewLedgerService(
		NewMemoryLedgerRepository(),
		MemoryLedgerAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, ApprovalRequired: true}},
		AllowAllReferenceValidator{},
		UnavailableApprovalValidator{},
		&MemoryLedgerAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	command := LedgerCommand{Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: uuid.New(), LedgerType: "primary", FunctionalCurrency: "USD", FiscalCalendarID: uuid.New(), LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "approval-required-1"}
	if _, err := service.Execute(context.Background(), Actor{UserID: uuid.New(), SubjectReference: "actor"}, command); !errors.Is(err, ErrLedgerApprovalRequired) {
		t.Fatalf("approval error = %v, want approval required", err)
	}
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func version(value int64) *aggregateversion.AggregateVersion {
	result, err := aggregateversion.FromInt64(value)
	if err != nil {
		panic(err)
	}
	return &result
}

func validApproval() *ApprovalDecisionReference {
	return &ApprovalDecisionReference{ApprovalRequestID: uuid.New(), DecisionID: uuid.New(), PolicyVersion: "policy-v1", DecisionVersion: 1, SubjectVersion: 1, CandidateFingerprint: "sha256:candidate", ApproverUserID: uuid.New()}
}

func repositoryListMust(t *testing.T, repository *MemoryLedgerRepository, scopeID uuid.UUID) []Ledger {
	t.Helper()
	values, err := repository.ListLedgers(context.Background(), &scopeID)
	if err != nil {
		t.Fatal(err)
	}
	return values
}
