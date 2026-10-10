package httpapi

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func TestGLChartAccountHandlersUseTypedCommandsAndPreserveReplay(t *testing.T) {
	scopeID := uuid.New()
	ledgerRepository := gl.NewMemoryLedgerRepository()
	ledger, err := gl.NewLedger(
		uuid.New(),
		scopeID,
		uuid.New(),
		"primary",
		"USD",
		uuid.New(),
		gl.LedgerStatusActive,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		nil,
		nil,
		time.Now().UTC(),
	)
	if err != nil {
		t.Fatal(err)
	}
	ledgerRepository.BindLedgerAuditRecorder(&gl.MemoryLedgerAuditRecorder{})
	if err := ledgerRepository.CommitLedgerMutation(context.Background(), gl.LedgerMutation{
		After: ledger,
		Audit: gl.LedgerAuditRecord{
			LedgerID: ledger.ID, ActorUserID: uuid.New(), ActorSubjectReference: "http-test",
			Action: gl.LedgerActionCreate, AccountingScopeID: scopeID, LegalEntityID: ledger.LegalEntityID,
			Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), RevisionNumber: 1,
		},
	}); err != nil {
		t.Fatal(err)
	}

	chartAuthorizer := gl.MemoryChartOfAccountsAuthorizer{Decision: gl.AuthorizationDecision{
		Allowed: true, Permission: gl.ChartOfAccountsManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}}
	chartRepository := gl.NewMemoryChartOfAccountsRepository(ledgerRepository)
	chartService, err := gl.NewChartOfAccountsService(
		chartRepository,
		chartAuthorizer,
		gl.AllowAllChartAccountReferenceValidator{},
		gl.AllowAllChartAccountApprovalValidator{},
		&gl.MemoryChartOfAccountsAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}

	accountAuthorizer := gl.MemoryAccountAuthorizer{Decision: gl.AuthorizationDecision{
		Allowed: true, Permission: gl.AccountManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
	}}
	accountService, err := gl.NewAccountService(
		gl.NewMemoryAccountRepository(chartRepository),
		accountAuthorizer,
		gl.AllowAllChartAccountReferenceValidator{},
		gl.AllowAllChartAccountApprovalValidator{},
		&gl.MemoryAccountAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}

	handler := IdentityHandler{ChartOfAccountsService: chartService, AccountService: accountService}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{
		UserID:  uuid.New(),
		Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"},
	})
	if err != nil {
		t.Fatal(err)
	}

	chartRequest := &generated.GlMaintainChartsOfAccountsCommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.UUID(scopeID),
		Data: generated.GlMaintainChartsOfAccountsCommandData{
			Action:            generated.GlMaintainChartsOfAccountsCommandDataActionCreate,
			LedgerId:          generated.UUID(ledger.ID),
			AccountCodePolicy: "NNNN",
			LifecycleStatus:   generated.GlMaintainChartsOfAccountsCommandDataLifecycleStatusDraft,
			EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	chartParams := generated.GlMaintainChartsOfAccountsParams{IdempotencyKey: "http-chart-create"}
	response, err := handler.GlMaintainChartsOfAccounts(actorContext, chartRequest, chartParams)
	if err != nil {
		t.Fatal(err)
	}
	createdChart, ok := response.(*generated.GlMaintainChartsOfAccountsEstablishedResult)
	if !ok || createdChart.Status != "established" || createdChart.AggregateVersion != 1 || createdChart.Data.ChartOfAccounts.LedgerId != generated.UUID(ledger.ID) {
		t.Fatalf("chart response = %#v", response)
	}

	replayedResponse, err := handler.GlMaintainChartsOfAccounts(actorContext, chartRequest, chartParams)
	if err != nil {
		t.Fatal(err)
	}
	replayedChart, ok := replayedResponse.(*generated.GlMaintainChartsOfAccountsEstablishedResult)
	if !ok || !replayedChart.Data.Replayed.Set || !replayedChart.Data.Replayed.Value {
		t.Fatalf("replayed chart response = %#v", replayedResponse)
	}

	update := *chartRequest
	update.CommandId = generated.UUID(uuid.New())
	update.ExpectedVersion = generated.NewOptInt(1)
	update.Data.Action = generated.GlMaintainChartsOfAccountsCommandDataActionUpdate
	update.Data.ChartOfAccountsId = generated.NewOptUUID(createdChart.AggregateId)
	update.Data.LifecycleStatus = generated.GlMaintainChartsOfAccountsCommandDataLifecycleStatusActive
	updatedResponse, err := handler.GlMaintainChartsOfAccounts(actorContext, &update, generated.GlMaintainChartsOfAccountsParams{
		IdempotencyKey: "http-chart-update",
		IfMatch:        generated.NewOptString("1"),
	})
	if err != nil {
		t.Fatal(err)
	}
	updatedChart, ok := updatedResponse.(*generated.GlMaintainChartsOfAccountsEstablishedResult)
	if !ok || updatedChart.AggregateVersion != 2 || updatedChart.Data.ChartOfAccounts.LifecycleStatus != gl.LedgerStatusActive {
		t.Fatalf("updated chart response = %#v", updatedResponse)
	}

	accountRequest := &generated.GlMaintainAccountsAndReportingMappingsCommandRequest{
		CommandId:         generated.UUID(uuid.New()),
		AccountingScopeId: generated.UUID(scopeID),
		Data: generated.GlMaintainAccountsAndReportingMappingsCommandData{
			Action:            generated.GlMaintainAccountsAndReportingMappingsCommandDataActionCreate,
			ChartOfAccountsId: createdChart.AggregateId,
			AccountCode:       "1000",
			AccountName:       "Cash",
			AccountType:       "asset",
			NormalBalance:     generated.GlMaintainAccountsAndReportingMappingsCommandDataNormalBalanceDebit,
			LifecycleStatus:   generated.GlMaintainAccountsAndReportingMappingsCommandDataLifecycleStatusActive,
			Restrictions:      []generated.GlAccountRestriction{{RestrictionCode: "postable"}},
			CurrencyPolicy:    "functional",
			ReportingMappings: []generated.GlAccountReportingMapping{},
			EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	accountResponse, err := handler.GlMaintainAccountsAndReportingMappings(actorContext, accountRequest, generated.GlMaintainAccountsAndReportingMappingsParams{IdempotencyKey: "http-account-create"})
	if err != nil {
		t.Fatal(err)
	}
	createdAccount, ok := accountResponse.(*generated.GlMaintainAccountsAndReportingMappingsEstablishedResult)
	if !ok || createdAccount.Status != "established" || createdAccount.Data.Account.ChartOfAccountsId != createdChart.AggregateId || createdAccount.Data.Account.NormalBalance != generated.GlAccountProjectionNormalBalanceDebit {
		t.Fatalf("account response = %#v", accountResponse)
	}
}

func TestGLChartAccountHandlersMapAuthorizationAndReportingDependencyFailures(t *testing.T) {
	scopeID := uuid.New()
	deniedService, err := gl.NewChartOfAccountsService(
		gl.NewMemoryChartOfAccountsRepository(),
		gl.MemoryChartOfAccountsAuthorizer{Decision: gl.AuthorizationDecision{
			Allowed: false, Permission: gl.ChartOfAccountsManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
		}},
		gl.AllowAllChartAccountReferenceValidator{},
		gl.AllowAllChartAccountApprovalValidator{},
		&gl.MemoryChartOfAccountsAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actorContext, err := identity.WithActor(context.Background(), identity.ApplicationActor{
		UserID: uuid.New(), Subject: identity.AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"},
	})
	if err != nil {
		t.Fatal(err)
	}
	chartRequest := &generated.GlMaintainChartsOfAccountsCommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.UUID(scopeID),
		Data: generated.GlMaintainChartsOfAccountsCommandData{
			Action: generated.GlMaintainChartsOfAccountsCommandDataActionCreate, LedgerId: generated.UUID(uuid.New()),
			AccountCodePolicy: "NN", LifecycleStatus: generated.GlMaintainChartsOfAccountsCommandDataLifecycleStatusDraft,
			EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	deniedResponse, err := (IdentityHandler{ChartOfAccountsService: deniedService}).GlMaintainChartsOfAccounts(actorContext, chartRequest, generated.GlMaintainChartsOfAccountsParams{IdempotencyKey: "http-chart-denied"})
	if err != nil {
		t.Fatal(err)
	}
	denied, ok := deniedResponse.(*generated.GlMaintainChartsOfAccountsForbidden)
	if !ok || denied.Code != "AUTHORIZATION_DENIED" {
		t.Fatalf("denied response = %#v", deniedResponse)
	}

	accountService, err := gl.NewAccountService(
		gl.NewMemoryAccountRepository(),
		gl.MemoryAccountAuthorizer{Decision: gl.AuthorizationDecision{
			Allowed: true, Permission: gl.AccountManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{scopeID},
		}},
		gl.UnavailableChartAccountReferenceValidator{},
		gl.AllowAllChartAccountApprovalValidator{},
		&gl.MemoryAccountAuditRecorder{},
		time.Now,
	)
	if err != nil {
		t.Fatal(err)
	}
	accountRequest := &generated.GlMaintainAccountsAndReportingMappingsCommandRequest{
		CommandId: generated.UUID(uuid.New()), AccountingScopeId: generated.UUID(scopeID),
		Data: generated.GlMaintainAccountsAndReportingMappingsCommandData{
			Action:            generated.GlMaintainAccountsAndReportingMappingsCommandDataActionCreate,
			ChartOfAccountsId: generated.UUID(uuid.New()), AccountCode: "1000", AccountName: "Cash", AccountType: "asset",
			NormalBalance:   generated.GlMaintainAccountsAndReportingMappingsCommandDataNormalBalanceDebit,
			LifecycleStatus: generated.GlMaintainAccountsAndReportingMappingsCommandDataLifecycleStatusDraft,
			CurrencyPolicy:  "functional", EffectiveDateFrom: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
			ReportingMappings: []generated.GlAccountReportingMapping{{ReportingDefinitionId: generated.UUID(uuid.New()), ReportingLineCode: "cash", Approved: true}},
		},
	}
	unavailableResponse, err := (IdentityHandler{AccountService: accountService}).GlMaintainAccountsAndReportingMappings(actorContext, accountRequest, generated.GlMaintainAccountsAndReportingMappingsParams{IdempotencyKey: "http-account-reporting-unavailable"})
	if err != nil {
		t.Fatal(err)
	}
	unavailable, ok := unavailableResponse.(*generated.GlMaintainAccountsAndReportingMappingsServiceUnavailable)
	if !ok || unavailable.Code != "SERVICE_UNAVAILABLE" {
		t.Fatalf("unavailable response = %#v", unavailableResponse)
	}
}
