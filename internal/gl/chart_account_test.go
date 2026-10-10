package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestChartAndAccountServicesPreserveRevisionsAndReplay(t *testing.T) {
	ctx := context.Background()
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "chart-account-actor"}
	decision := AuthorizationDecision{Allowed: true, Permission: ChartOfAccountsManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}, PolicyReference: "policy/gl", PolicyVersion: "1"}
	ledgerRepository := NewMemoryLedgerRepository()
	ledgerAudit := &MemoryLedgerAuditRecorder{}
	ledger, err := NewLedger(uuid.New(), scopeID, uuid.New(), "primary", "USD", uuid.New(), LedgerStatusActive, date(2026, 1, 1), nil, nil, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	ledgerRepository.BindLedgerAuditRecorder(ledgerAudit)
	if err := ledgerRepository.CommitLedgerMutation(ctx, LedgerMutation{After: ledger, Audit: LedgerAuditRecord{LedgerID: ledger.ID, ActorUserID: actor.UserID, ActorSubjectReference: actor.SubjectReference, Action: LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: ledger.LegalEntityID, Permission: LedgerManagementPermission, DecisionReference: uuid.New(), RevisionNumber: 1}}); err != nil {
		t.Fatal(err)
	}

	chartRepository := NewMemoryChartOfAccountsRepository(ledgerRepository)
	chartAudit := &MemoryChartOfAccountsAuditRecorder{}
	chartService, err := NewChartOfAccountsService(chartRepository, MemoryChartOfAccountsAuthorizer{Decision: decision}, AllowAllChartAccountReferenceValidator{}, AllowAllChartAccountApprovalValidator{}, chartAudit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	createCommand := ChartOfAccountsCommand{Action: ChartOfAccountsActionCreate, AccountingScopeID: scopeID, LedgerID: ledger.ID, AccountCodePolicy: "NNNN", LifecycleStatus: LedgerStatusDraft, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "chart-create-1"}
	created, err := chartService.Execute(ctx, actor, createCommand)
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := chartService.Execute(ctx, actor, createCommand)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.Replayed || replayed.ChartOfAccounts.ID != created.ChartOfAccounts.ID {
		t.Fatalf("replayed chart = %#v", replayed)
	}
	updated, err := chartService.Execute(ctx, actor, ChartOfAccountsCommand{Action: ChartOfAccountsActionUpdate, ChartOfAccountsID: created.ChartOfAccounts.ID, AccountingScopeID: scopeID, LedgerID: ledger.ID, AccountCodePolicy: "NNNN-NN", LifecycleStatus: LedgerStatusActive, EffectiveDateFrom: date(2026, 1, 1), ExpectedVersion: version(1), IdempotencyKey: "chart-update-1"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ChartOfAccounts.Version.Value() != 2 || len(chartAudit.Records) != 2 {
		t.Fatalf("updated chart=%#v audits=%d", updated.ChartOfAccounts, len(chartAudit.Records))
	}
	storedChart, err := chartRepository.GetChartOfAccounts(ctx, created.ChartOfAccounts.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storedChart.Revisions) != 2 || storedChart.Revisions[0].Snapshot.AccountCodePolicy != "NNNN" {
		t.Fatalf("chart revisions = %#v", storedChart.Revisions)
	}

	accountDecision := decision
	accountDecision.Permission = AccountManagementPermission
	accountRepository := NewMemoryAccountRepository(chartRepository)
	accountAudit := &MemoryAccountAuditRecorder{}
	accountService, err := NewAccountService(accountRepository, MemoryAccountAuthorizer{Decision: accountDecision}, AllowAllChartAccountReferenceValidator{}, AllowAllChartAccountApprovalValidator{}, accountAudit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	accountResult, err := accountService.Execute(ctx, actor, AccountCommand{Action: AccountActionCreate, AccountingScopeID: scopeID, ChartOfAccountsID: created.ChartOfAccounts.ID, AccountCode: "1000", AccountName: "Cash", AccountType: "asset", NormalBalance: "debit", LifecycleStatus: LedgerStatusActive, Restrictions: []AccountRestriction{{RestrictionCode: "postable"}}, CurrencyPolicy: "functional-or-transaction", ReportingMappings: []AccountReportingMapping{{ReportingDefinitionID: uuid.New(), ReportingLineCode: "cash", Approved: true}}, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "account-create-1"})
	if err != nil {
		t.Fatal(err)
	}
	if accountResult.Account.ReportingMappings[0].EffectiveDateFrom != date(2026, 1, 1) {
		t.Fatalf("mapping dates = %#v", accountResult.Account.ReportingMappings)
	}
	if _, err := accountService.Execute(ctx, actor, AccountCommand{Action: AccountActionUpdate, AccountID: accountResult.Account.ID, AccountingScopeID: scopeID, ChartOfAccountsID: created.ChartOfAccounts.ID, AccountCode: "1000", AccountName: "Cash and equivalents", AccountType: "asset", NormalBalance: "debit", LifecycleStatus: LedgerStatusActive, CurrencyPolicy: "functional-or-transaction", EffectiveDateFrom: date(2026, 1, 1), ExpectedVersion: version(1), IdempotencyKey: "account-update-1"}); err != nil {
		t.Fatal(err)
	}
	storedAccount, err := accountRepository.GetAccount(ctx, accountResult.Account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(storedAccount.Revisions) != 2 || storedAccount.Revisions[0].Snapshot.AccountName != "Cash" || len(accountAudit.Records) != 2 {
		t.Fatalf("account revisions=%#v audits=%d", storedAccount.Revisions, len(accountAudit.Records))
	}
	if _, err := accountService.Execute(ctx, actor, AccountCommand{Action: AccountActionUpdate, AccountID: accountResult.Account.ID, AccountingScopeID: scopeID, ChartOfAccountsID: created.ChartOfAccounts.ID, AccountCode: "1000", AccountName: "stale", AccountType: "asset", NormalBalance: "debit", LifecycleStatus: LedgerStatusActive, CurrencyPolicy: "functional-or-transaction", EffectiveDateFrom: date(2026, 1, 1), ExpectedVersion: version(1), IdempotencyKey: "account-stale-1"}); !errors.Is(err, ErrAccountVersionConflict) {
		t.Fatalf("stale account error=%v", err)
	}
}

func TestAccountValidationRejectsInvalidBalanceMappingAndOverlap(t *testing.T) {
	from := date(2026, 1, 1)
	if _, err := NewAccount(uuid.New(), uuid.New(), uuid.New(), "1000", "Cash", "asset", "credit", LedgerStatusDraft, nil, "functional", nil, from, nil, nil, time.Now().UTC()); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("normal-balance error=%v, want invalid account", err)
	}
	if _, err := NewAccount(uuid.New(), uuid.New(), uuid.New(), "1000", "Cash", "asset", "debit", LedgerStatusDraft, nil, "functional", []AccountReportingMapping{{ReportingDefinitionID: uuid.New(), ReportingLineCode: "cash", Approved: false}}, from, nil, nil, time.Now().UTC()); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("unapproved mapping error=%v, want invalid account", err)
	}
	if _, err := NewAccount(uuid.New(), uuid.New(), uuid.New(), "1000", "Cash", "asset", "debit", LedgerStatusDraft, []AccountRestriction{{RestrictionCode: "postable"}, {RestrictionCode: "postable"}}, "functional", nil, from, nil, nil, time.Now().UTC()); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("duplicate restriction error=%v, want invalid account", err)
	}
	mappingDefinitionID := uuid.New()
	if _, err := NewAccount(uuid.New(), uuid.New(), uuid.New(), "1000", "Cash", "asset", "debit", LedgerStatusDraft, nil, "functional", []AccountReportingMapping{{ReportingDefinitionID: mappingDefinitionID, ReportingLineCode: "cash", Approved: true}, {ReportingDefinitionID: mappingDefinitionID, ReportingLineCode: "cash", Approved: true}}, from, nil, nil, time.Now().UTC()); !errors.Is(err, ErrInvalidAccount) {
		t.Fatalf("overlapping mapping error=%v, want invalid account", err)
	}
}

func TestChartAndAccountServicesFailClosedForAuthorizationAndAudit(t *testing.T) {
	scopeID := uuid.New()
	actor := Actor{UserID: uuid.New(), SubjectReference: "actor"}
	command := ChartOfAccountsCommand{Action: ChartOfAccountsActionCreate, AccountingScopeID: scopeID, LedgerID: uuid.New(), AccountCodePolicy: "NN", LifecycleStatus: LedgerStatusDraft, EffectiveDateFrom: date(2026, 1, 1), IdempotencyKey: "chart-fail-1"}
	service, err := NewChartOfAccountsService(NewMemoryChartOfAccountsRepository(), MemoryChartOfAccountsAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: "finance.other", DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID}}}, AllowAllChartAccountReferenceValidator{}, AllowAllChartAccountApprovalValidator{}, &MemoryChartOfAccountsAuditRecorder{}, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Execute(context.Background(), actor, command); !errors.Is(err, ErrChartOfAccountsAuthorizationDenied) {
		t.Fatalf("authorization error=%v", err)
	}
	audit := &MemoryChartOfAccountsAuditRecorder{Err: errors.New("audit storage unavailable")}
	auditService, err := NewChartOfAccountsService(NewMemoryChartOfAccountsRepository(), MemoryChartOfAccountsAuthorizer{Decision: AuthorizationDecision{Allowed: true, Permission: ChartOfAccountsManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}}, AllowAllChartAccountReferenceValidator{}, AllowAllChartAccountApprovalValidator{}, audit, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := auditService.Execute(context.Background(), actor, command); err == nil {
		t.Fatal("expected audit failure")
	}
}
