package httpapi

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

func (handler IdentityHandler) GlMaintainChartsOfAccounts(ctx context.Context, request *generated.GlMaintainChartsOfAccountsCommandRequest, params generated.GlMaintainChartsOfAccountsParams) (generated.GlMaintainChartsOfAccountsRes, error) {
	correlationID := glChartCorrelation(params)
	if handler.ChartOfAccountsService == nil {
		return glChartProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The chart-of-accounts service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glChartProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return glChartProblem(http.StatusBadRequest, "INVALID_REQUEST", "The chart-of-accounts command envelope is invalid.", correlationID), nil
	}
	scopeID, err := glScopeID(request.AccountingScopeId, params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glChartProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return glChartProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}
	expectedVersion, err := glExpectedVersion(request.ExpectedVersion, params.IfMatch)
	if err != nil {
		if errors.Is(err, errGLVersionConflict) {
			return glChartProblem(http.StatusConflict, "VERSION_CONFLICT", "The expected and If-Match versions do not agree.", correlationID), nil
		}
		return glChartProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected or If-Match version is invalid.", correlationID), nil
	}
	data := request.Data
	chartID := uuid.Nil
	if data.ChartOfAccountsId.Set {
		chartID = uuid.UUID(data.ChartOfAccountsId.Value)
	}
	command := gl.ChartOfAccountsCommand{Action: string(data.Action), ChartOfAccountsID: chartID, AccountingScopeID: scopeID, LedgerID: uuid.UUID(data.LedgerId), AccountCodePolicy: data.AccountCodePolicy, LifecycleStatus: string(data.LifecycleStatus), EffectiveDateFrom: data.EffectiveDateFrom, EffectiveDateTo: glEffectiveDateTo(data.EffectiveDateTo), Approval: glApproval(data.Approval), ExpectedVersion: expectedVersion, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String()}
	result, err := handler.ChartOfAccountsService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlChartError(err, correlationID), nil
	}
	return establishedGlChartResult(result, correlationID), nil
}

func (handler IdentityHandler) GlMaintainAccountsAndReportingMappings(ctx context.Context, request *generated.GlMaintainAccountsAndReportingMappingsCommandRequest, params generated.GlMaintainAccountsAndReportingMappingsParams) (generated.GlMaintainAccountsAndReportingMappingsRes, error) {
	correlationID := glAccountCorrelation(params)
	if handler.AccountService == nil {
		return glAccountProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The account service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return glAccountProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return glAccountProblem(http.StatusBadRequest, "INVALID_REQUEST", "The account command envelope is invalid.", correlationID), nil
	}
	scopeID, err := glScopeID(request.AccountingScopeId, params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errGLScopeConflict) {
			return glAccountProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return glAccountProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}
	expectedVersion, err := glExpectedVersion(request.ExpectedVersion, params.IfMatch)
	if err != nil {
		if errors.Is(err, errGLVersionConflict) {
			return glAccountProblem(http.StatusConflict, "VERSION_CONFLICT", "The expected and If-Match versions do not agree.", correlationID), nil
		}
		return glAccountProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected or If-Match version is invalid.", correlationID), nil
	}
	data := request.Data
	accountID := uuid.Nil
	if data.AccountId.Set {
		accountID = uuid.UUID(data.AccountId.Value)
	}
	command := gl.AccountCommand{Action: string(data.Action), AccountID: accountID, AccountingScopeID: scopeID, ChartOfAccountsID: uuid.UUID(data.ChartOfAccountsId), AccountCode: data.AccountCode, AccountName: data.AccountName, AccountType: data.AccountType, NormalBalance: string(data.NormalBalance), LifecycleStatus: string(data.LifecycleStatus), Restrictions: glAccountRestrictions(data.Restrictions), CurrencyPolicy: data.CurrencyPolicy, ReportingMappings: glReportingMappings(data.ReportingMappings), EffectiveDateFrom: data.EffectiveDateFrom, EffectiveDateTo: glEffectiveDateTo(data.EffectiveDateTo), Approval: glApproval(data.Approval), ExpectedVersion: expectedVersion, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String()}
	result, err := handler.AccountService.Execute(ctx, gl.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapGlAccountError(err, correlationID), nil
	}
	return establishedGlAccountResult(result, correlationID), nil
}

func glAccountRestrictions(values []generated.GlAccountRestriction) []gl.AccountRestriction {
	result := make([]gl.AccountRestriction, 0, len(values))
	for _, value := range values {
		restriction := gl.AccountRestriction{RestrictionCode: value.RestrictionCode}
		if value.Description.Set {
			restriction.Description = value.Description.Value
		}
		result = append(result, restriction)
	}
	return result
}

func glReportingMappings(values []generated.GlAccountReportingMapping) []gl.AccountReportingMapping {
	result := make([]gl.AccountReportingMapping, 0, len(values))
	for _, value := range values {
		mapping := gl.AccountReportingMapping{ReportingDefinitionID: uuid.UUID(value.ReportingDefinitionId), ReportingLineCode: value.ReportingLineCode, Approved: value.Approved}
		if value.EffectiveDateFrom.Set {
			mapping.EffectiveDateFrom = value.EffectiveDateFrom.Value
		}
		mapping.EffectiveDateTo = glEffectiveDateTo(value.EffectiveDateTo)
		result = append(result, mapping)
	}
	return result
}

func establishedGlChartResult(result gl.ChartOfAccountsCommandResult, correlationID uuid.UUID) *generated.GlMaintainChartsOfAccountsEstablishedResult {
	data := generated.GlMaintainChartsOfAccountsResultData{ChartOfAccounts: generated.GlChartOfAccountsProjection{ID: generated.UUID(result.ChartOfAccounts.ID), AccountingScopeId: generated.UUID(result.ChartOfAccounts.AccountingScopeID), LedgerId: generated.UUID(result.ChartOfAccounts.LedgerID), AccountCodePolicy: result.ChartOfAccounts.AccountCodePolicy, LifecycleStatus: result.ChartOfAccounts.LifecycleStatus, EffectiveDateFrom: result.ChartOfAccounts.EffectiveDateFrom, ApprovalStatus: result.ChartOfAccounts.ApprovalStatus, ValidationOutcome: result.ChartOfAccounts.ValidationOutcome, NextAction: result.ChartOfAccounts.NextAction, Version: int(result.ChartOfAccounts.Version.Value()), RevisionNumber: int(result.ChartOfAccounts.RevisionNumber)}, ValidationOutcome: result.ValidationOutcome, ApprovalStatus: result.ApprovalStatus}
	data.ChartOfAccounts.EffectiveDateTo = generatedGlDate(result.ChartOfAccounts.EffectiveDateTo)
	if result.DecisionReference != uuid.Nil {
		data.DecisionReference = generated.NewOptUUID(generated.UUID(result.DecisionReference))
	}
	if result.PolicyReference != "" {
		data.PolicyReference = generated.NewOptString(result.PolicyReference)
	}
	if result.Replayed {
		data.Replayed = generated.NewOptBool(true)
	}
	return &generated.GlMaintainChartsOfAccountsEstablishedResult{Status: "established", AggregateId: generated.UUID(result.ChartOfAccounts.ID), AggregateVersion: int(result.ChartOfAccounts.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/general-ledger/configuration/maintain-charts-of-accounts"}, Data: data}
}

func establishedGlAccountResult(result gl.AccountCommandResult, correlationID uuid.UUID) *generated.GlMaintainAccountsAndReportingMappingsEstablishedResult {
	projection := generated.GlAccountProjection{ID: generated.UUID(result.Account.ID), AccountingScopeId: generated.UUID(result.Account.AccountingScopeID), ChartOfAccountsId: generated.UUID(result.Account.ChartOfAccountsID), AccountCode: result.Account.AccountCode, AccountName: result.Account.AccountName, AccountType: result.Account.AccountType, NormalBalance: generated.GlAccountProjectionNormalBalance(result.Account.NormalBalance), LifecycleStatus: result.Account.LifecycleStatus, Restrictions: make([]generated.GlAccountRestriction, 0, len(result.Account.Restrictions)), CurrencyPolicy: result.Account.CurrencyPolicy, ReportingMappings: make([]generated.GlAccountReportingMapping, 0, len(result.Account.ReportingMappings)), EffectiveDateFrom: result.Account.EffectiveDateFrom, ApprovalStatus: result.Account.ApprovalStatus, ValidationOutcome: result.Account.ValidationOutcome, NextAction: result.Account.NextAction, Version: int(result.Account.Version.Value()), RevisionNumber: int(result.Account.RevisionNumber)}
	projection.EffectiveDateTo = generatedGlDate(result.Account.EffectiveDateTo)
	for _, restriction := range result.Account.Restrictions {
		value := generated.GlAccountRestriction{RestrictionCode: restriction.RestrictionCode}
		if restriction.Description != "" {
			value.Description = generated.NewOptString(restriction.Description)
		}
		projection.Restrictions = append(projection.Restrictions, value)
	}
	for _, mapping := range result.Account.ReportingMappings {
		projection.ReportingMappings = append(projection.ReportingMappings, generated.GlAccountReportingMapping{ReportingDefinitionId: generated.UUID(mapping.ReportingDefinitionID), ReportingLineCode: mapping.ReportingLineCode, Approved: mapping.Approved, EffectiveDateFrom: generated.NewOptDate(mapping.EffectiveDateFrom), EffectiveDateTo: generatedGlDate(mapping.EffectiveDateTo)})
	}
	data := generated.GlMaintainAccountsAndReportingMappingsResultData{Account: projection, ValidationOutcome: result.ValidationOutcome, ApprovalStatus: result.ApprovalStatus}
	if result.DecisionReference != uuid.Nil {
		data.DecisionReference = generated.NewOptUUID(generated.UUID(result.DecisionReference))
	}
	if result.PolicyReference != "" {
		data.PolicyReference = generated.NewOptString(result.PolicyReference)
	}
	if result.Replayed {
		data.Replayed = generated.NewOptBool(true)
	}
	return &generated.GlMaintainAccountsAndReportingMappingsEstablishedResult{Status: "established", AggregateId: generated.UUID(result.Account.ID), AggregateVersion: int(result.Account.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/general-ledger/configuration/maintain-accounts-and-reporting-mappings"}, Data: data}
}

func glChartCorrelation(params generated.GlMaintainChartsOfAccountsParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func glAccountCorrelation(params generated.GlMaintainAccountsAndReportingMappingsParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func mapGlChartError(err error, correlationID uuid.UUID) generated.GlMaintainChartsOfAccountsRes {
	switch {
	case errors.Is(err, gl.ErrChartOfAccountsAuthorizationUnavailable):
		return glChartProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsAuthorizationStale):
		return glChartProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsAuthorizationDenied):
		return glChartProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested chart-of-accounts scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsVersionConflict):
		return glChartProblem(http.StatusConflict, "VERSION_CONFLICT", "The chart of accounts changed after it was loaded. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsIdempotencyConflict):
		return glChartProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different chart data.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsCommandInProgress):
		return glChartProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The chart command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsNotFound):
		return glChartProblem(http.StatusConflict, "CHART_OF_ACCOUNTS_NOT_FOUND", "The requested chart of accounts does not exist.", correlationID)
	case errors.Is(err, gl.ErrInvalidChartOfAccountsCommand):
		return glChartProblem(http.StatusBadRequest, "INVALID_REQUEST", "The chart-of-accounts command is invalid.", correlationID)
	case errors.Is(err, gl.ErrInvalidChartOfAccounts), errors.Is(err, gl.ErrChartOfAccountsDuplicate), errors.Is(err, gl.ErrChartOfAccountsReferenceInvalid), errors.Is(err, gl.ErrChartOfAccountsApprovalRequired), errors.Is(err, gl.ErrChartOfAccountsApprovalInvalid), errors.Is(err, gl.ErrChartOfAccountsDurableCommandFailed):
		return glChartProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The chart-of-accounts command violates a general-ledger rule.", correlationID)
	case errors.Is(err, gl.ErrChartOfAccountsReferenceUnavailable), errors.Is(err, gl.ErrChartOfAccountsApprovalUnavailable), errors.Is(err, gl.ErrChartOfAccountsAuditUnavailable), errors.Is(err, gl.ErrInvalidChartOfAccountsService):
		return glChartProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The chart-of-accounts operation could not be completed.", correlationID)
	default:
		return glChartProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The chart-of-accounts operation could not be completed.", correlationID)
	}
}

func mapGlAccountError(err error, correlationID uuid.UUID) generated.GlMaintainAccountsAndReportingMappingsRes {
	switch {
	case errors.Is(err, gl.ErrAccountAuthorizationUnavailable):
		return glAccountProblem(http.StatusServiceUnavailable, "POLICY_UNAVAILABLE", "The authorization policy could not be evaluated. Retry later.", correlationID)
	case errors.Is(err, gl.ErrAccountAuthorizationStale):
		return glAccountProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrAccountAuthorizationDenied):
		return glAccountProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested account scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, gl.ErrAccountVersionConflict):
		return glAccountProblem(http.StatusConflict, "VERSION_CONFLICT", "The account changed after it was loaded. Refresh and retry.", correlationID)
	case errors.Is(err, gl.ErrAccountIdempotencyConflict):
		return glAccountProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different account data.", correlationID)
	case errors.Is(err, gl.ErrAccountCommandInProgress):
		return glAccountProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The account command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, gl.ErrAccountNotFound):
		return glAccountProblem(http.StatusConflict, "ACCOUNT_NOT_FOUND", "The requested account does not exist.", correlationID)
	case errors.Is(err, gl.ErrInvalidAccountCommand):
		return glAccountProblem(http.StatusBadRequest, "INVALID_REQUEST", "The account command is invalid.", correlationID)
	case errors.Is(err, gl.ErrInvalidAccount), errors.Is(err, gl.ErrAccountDuplicate), errors.Is(err, gl.ErrAccountReferenceInvalid), errors.Is(err, gl.ErrAccountApprovalRequired), errors.Is(err, gl.ErrAccountApprovalInvalid), errors.Is(err, gl.ErrAccountDurableCommandFailed):
		return glAccountProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The account command violates a general-ledger rule.", correlationID)
	case errors.Is(err, gl.ErrAccountReferenceUnavailable), errors.Is(err, gl.ErrAccountApprovalUnavailable), errors.Is(err, gl.ErrAccountAuditUnavailable), errors.Is(err, gl.ErrInvalidAccountService):
		return glAccountProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The account operation could not be completed.", correlationID)
	default:
		return glAccountProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The account operation could not be completed.", correlationID)
	}
}

func glChartProblem(status int, code, detail string, correlationID uuid.UUID) generated.GlMaintainChartsOfAccountsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlMaintainChartsOfAccountsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlMaintainChartsOfAccountsForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlMaintainChartsOfAccountsConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlMaintainChartsOfAccountsUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlMaintainChartsOfAccountsServiceUnavailable(problem)
		return &value
	}
}

func glAccountProblem(status int, code, detail string, correlationID uuid.UUID) generated.GlMaintainAccountsAndReportingMappingsRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.GlMaintainAccountsAndReportingMappingsBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.GlMaintainAccountsAndReportingMappingsForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.GlMaintainAccountsAndReportingMappingsConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.GlMaintainAccountsAndReportingMappingsUnprocessableEntity(problem)
		return &value
	default:
		value := generated.GlMaintainAccountsAndReportingMappingsServiceUnavailable(problem)
		return &value
	}
}
