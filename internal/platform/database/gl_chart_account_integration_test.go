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

func TestGLChartAccountRepositoryPreservesRevisionsAndRollsBackAuditFailures(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Minute)
	defer cancel()

	fixture := startIntegrationFixture(t, ctx)
	applyAndVerifyMigrations(t, ctx, fixture.databaseURL, fixture.migrationSets)
	fixture.openPool(t, ctx)

	scopeID := uuid.New()
	ledgerID := uuid.New()
	ledger, err := gl.NewLedger(
		ledgerID,
		scopeID,
		uuid.New(),
		"primary",
		"USD",
		uuid.New(),
		gl.LedgerStatusActive,
		integrationDate(2026, 1, 1),
		nil,
		nil,
		integrationDate(2026, 1, 2),
	)
	if err != nil {
		t.Fatal(err)
	}

	ledgerAuditReference := uuid.New()
	configurationRepository, err := gl.NewPostgresConfigurationRepository(
		fixture.pool,
		func(context.Context, pgx.Tx, gl.LedgerAuditRecord) (uuid.UUID, error) {
			return ledgerAuditReference, nil
		},
		func(context.Context, pgx.Tx, gl.AccountingBookAuditRecord) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := configurationRepository.CommitLedgerMutation(ctx, gl.LedgerMutation{
		After: ledger,
		Audit: gl.LedgerAuditRecord{
			LedgerID:              ledger.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "chart-account-integration",
			Action:                gl.LedgerActionCreate,
			AccountingScopeID:     scopeID,
			LegalEntityID:         ledger.LegalEntityID,
			Permission:            gl.LedgerManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        ledger.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}

	chartAuditReference := uuid.New()
	accountAuditReference := uuid.New()
	repository, err := gl.NewPostgresChartAccountRepository(
		fixture.pool,
		func(context.Context, pgx.Tx, gl.ChartOfAccountsAuditRecord) (uuid.UUID, error) {
			return chartAuditReference, nil
		},
		func(context.Context, pgx.Tx, gl.AccountAuditRecord) (uuid.UUID, error) {
			return accountAuditReference, nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	chart, err := gl.NewChartOfAccounts(
		uuid.New(),
		scopeID,
		ledger.ID,
		"NNNN-NN",
		gl.LedgerStatusDraft,
		integrationDate(2026, 1, 1),
		nil,
		nil,
		integrationDate(2026, 1, 2),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitChartOfAccountsMutation(ctx, gl.ChartOfAccountsMutation{
		After: chart,
		Audit: gl.ChartOfAccountsAuditRecord{
			ChartOfAccountsID:     chart.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "chart-account-integration",
			Action:                gl.ChartOfAccountsActionCreate,
			AccountingScopeID:     scopeID,
			LedgerID:              ledger.ID,
			Permission:            gl.ChartOfAccountsManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        chart.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}

	storedChart, err := repository.GetChartOfAccounts(ctx, chart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedChart.Version.Value() != 1 || len(storedChart.Revisions) != 1 || storedChart.LastAuditReference != chartAuditReference {
		t.Fatalf("stored chart = %#v", storedChart)
	}

	updatedChart := storedChart
	if err := updatedChart.Replace(storedChart, gl.ChartOfAccountsCommand{
		Action:            gl.ChartOfAccountsActionUpdate,
		ChartOfAccountsID: storedChart.ID,
		AccountingScopeID: scopeID,
		LedgerID:          ledger.ID,
		AccountCodePolicy: "NNNN-NN-N",
		LifecycleStatus:   gl.LedgerStatusActive,
		EffectiveDateFrom: integrationDate(2026, 1, 1),
	}, integrationDate(2026, 2, 1)); err != nil {
		t.Fatal(err)
	}
	expectedChartVersion := storedChart.Version
	if err := repository.CommitChartOfAccountsMutation(ctx, gl.ChartOfAccountsMutation{
		Before:          storedChart,
		After:           updatedChart,
		ExpectedVersion: &expectedChartVersion,
		Audit: gl.ChartOfAccountsAuditRecord{
			ChartOfAccountsID:     updatedChart.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "chart-account-integration",
			Action:                gl.ChartOfAccountsActionUpdate,
			AccountingScopeID:     scopeID,
			LedgerID:              ledger.ID,
			Permission:            gl.ChartOfAccountsManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        updatedChart.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}

	reloadedChart, err := repository.GetChartOfAccounts(ctx, chart.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedChart.Version.Value() != 2 || reloadedChart.LifecycleStatus != gl.LedgerStatusActive || len(reloadedChart.Revisions) != 2 || reloadedChart.Revisions[0].Snapshot.AccountCodePolicy != "NNNN-NN" {
		t.Fatalf("reloaded chart = %#v", reloadedChart)
	}

	account, err := gl.NewAccount(
		uuid.New(),
		scopeID,
		chart.ID,
		"1111",
		"Cash and bank",
		"asset",
		"debit",
		gl.LedgerStatusActive,
		[]gl.AccountRestriction{{RestrictionCode: "postable"}},
		"functional-or-transaction",
		[]gl.AccountReportingMapping{{
			ReportingDefinitionID: uuid.New(),
			ReportingLineCode:     "cash",
			Approved:              true,
		}},
		integrationDate(2026, 1, 1),
		nil,
		nil,
		integrationDate(2026, 1, 2),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitAccountMutation(ctx, gl.AccountMutation{
		After: account,
		Audit: gl.AccountAuditRecord{
			AccountID:             account.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "chart-account-integration",
			Action:                gl.AccountActionCreate,
			AccountingScopeID:     scopeID,
			ChartOfAccountsID:     chart.ID,
			Permission:            gl.AccountManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        account.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}

	storedAccount, err := repository.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedAccount.Version.Value() != 1 || len(storedAccount.Revisions) != 1 || storedAccount.LastAuditReference != accountAuditReference || len(storedAccount.ReportingMappings) != 1 {
		t.Fatalf("stored account = %#v", storedAccount)
	}

	updatedAccount := storedAccount
	if err := updatedAccount.Replace(storedAccount, gl.AccountCommand{
		Action:            gl.AccountActionUpdate,
		AccountID:         storedAccount.ID,
		AccountingScopeID: scopeID,
		ChartOfAccountsID: chart.ID,
		AccountCode:       storedAccount.AccountCode,
		AccountName:       "Cash and cash equivalents",
		AccountType:       "asset",
		NormalBalance:     "debit",
		LifecycleStatus:   gl.LedgerStatusActive,
		Restrictions:      storedAccount.Restrictions,
		CurrencyPolicy:    storedAccount.CurrencyPolicy,
		ReportingMappings: storedAccount.ReportingMappings,
		EffectiveDateFrom: integrationDate(2026, 1, 1),
	}, integrationDate(2026, 2, 1)); err != nil {
		t.Fatal(err)
	}
	expectedAccountVersion := storedAccount.Version
	if err := repository.CommitAccountMutation(ctx, gl.AccountMutation{
		Before:          storedAccount,
		After:           updatedAccount,
		ExpectedVersion: &expectedAccountVersion,
		Audit: gl.AccountAuditRecord{
			AccountID:             updatedAccount.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "chart-account-integration",
			Action:                gl.AccountActionUpdate,
			AccountingScopeID:     scopeID,
			ChartOfAccountsID:     chart.ID,
			Permission:            gl.AccountManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        updatedAccount.RevisionNumber,
		},
	}); err != nil {
		t.Fatal(err)
	}

	reloadedAccount, err := repository.GetAccount(ctx, account.ID)
	if err != nil {
		t.Fatal(err)
	}
	if reloadedAccount.Version.Value() != 2 || reloadedAccount.AccountName != "Cash and cash equivalents" || len(reloadedAccount.Revisions) != 2 || reloadedAccount.Revisions[0].Snapshot.AccountName != "Cash and bank" {
		t.Fatalf("reloaded account = %#v", reloadedAccount)
	}

	invalidParent, err := gl.NewAccount(
		uuid.New(),
		scopeID,
		uuid.New(),
		"9999",
		"Invalid parent",
		"asset",
		"debit",
		gl.LedgerStatusDraft,
		nil,
		"functional",
		nil,
		integrationDate(2026, 1, 1),
		nil,
		nil,
		integrationDate(2026, 1, 2),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.CommitAccountMutation(ctx, gl.AccountMutation{
		After: invalidParent,
		Audit: gl.AccountAuditRecord{
			AccountID:             invalidParent.ID,
			ActorUserID:           uuid.New(),
			ActorSubjectReference: "chart-account-integration",
			Action:                gl.AccountActionCreate,
			AccountingScopeID:     scopeID,
			ChartOfAccountsID:     invalidParent.ChartOfAccountsID,
			Permission:            gl.AccountManagementPermission,
			DecisionReference:     uuid.New(),
			RevisionNumber:        invalidParent.RevisionNumber,
		},
	}); !errors.Is(err, gl.ErrAccountReferenceInvalid) {
		t.Fatalf("invalid account parent error = %v", err)
	}

	auditFailureRepository, err := gl.NewPostgresChartAccountRepository(
		fixture.pool,
		func(context.Context, pgx.Tx, gl.ChartOfAccountsAuditRecord) (uuid.UUID, error) {
			return uuid.Nil, errors.New("audit unavailable")
		},
		func(context.Context, pgx.Tx, gl.AccountAuditRecord) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	failedChart, err := gl.NewChartOfAccounts(
		uuid.New(),
		scopeID,
		ledger.ID,
		"FAIL",
		gl.LedgerStatusDraft,
		integrationDate(2027, 1, 1),
		nil,
		nil,
		integrationDate(2027, 1, 2),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := auditFailureRepository.CommitChartOfAccountsMutation(ctx, gl.ChartOfAccountsMutation{
		After: failedChart,
		Audit: gl.ChartOfAccountsAuditRecord{
			ChartOfAccountsID: failedChart.ID,
			ActorUserID:       uuid.New(),
			Action:            gl.ChartOfAccountsActionCreate,
			AccountingScopeID: scopeID,
			LedgerID:          ledger.ID,
			Permission:        gl.ChartOfAccountsManagementPermission,
			DecisionReference: uuid.New(),
			RevisionNumber:    failedChart.RevisionNumber,
		},
	}); err == nil {
		t.Fatal("audit failure commit returned nil")
	}
	var failedCount int
	if err := fixture.pool.QueryRow(ctx, `SELECT count(*) FROM gl.chart_of_accounts WHERE chart_of_accounts_id = $1`, failedChart.ID).Scan(&failedCount); err != nil {
		t.Fatal(err)
	}
	if failedCount != 0 {
		t.Fatalf("audit failure left %d chart rows", failedCount)
	}
}

func integrationDate(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
