package httpapi

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/invoicing"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
)

var (
	errCommandScopeConflict   = errors.New("accounting scope mismatch")
	errCommandVersionConflict = errors.New("command body and header versions do not agree")
)

func (handler IdentityHandler) InvConfigureInvoiceTemplates(ctx context.Context, request *generated.InvConfigureInvoiceTemplatesCommandRequest, params generated.InvConfigureInvoiceTemplatesParams) (generated.InvConfigureInvoiceTemplatesRes, error) {
	correlationID := invoiceTemplateConfigureCorrelation(params)
	if handler.InvoiceTemplateService == nil {
		return invoiceTemplateConfigureProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return invoiceTemplateConfigureProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return invoiceTemplateConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The invoice-template command envelope is invalid.", correlationID), nil
	}
	scopeID, err := commandScopeID(request.AccountingScopeId, params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errCommandScopeConflict) {
			return invoiceTemplateConfigureProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return invoiceTemplateConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}
	expectedVersion, err := commandExpectedVersion(request.ExpectedVersion, params.IfMatch)
	if err != nil {
		if errors.Is(err, errCommandVersionConflict) {
			return invoiceTemplateConfigureProblem(http.StatusConflict, "VERSION_CONFLICT", "The expected and If-Match versions do not agree.", correlationID), nil
		}
		return invoiceTemplateConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected or If-Match version is invalid.", correlationID), nil
	}
	data := request.Data
	templateID := uuid.Nil
	if data.TemplateId.Set {
		templateID = uuid.UUID(data.TemplateId.Value)
	}
	lineItems := make([]invoicing.TemplateLineItem, 0, len(data.LineItems))
	for _, item := range data.LineItems {
		chargeRule := item.ChargeRule
		rule := invoicing.ChargeRule{Kind: invoicing.ChargeRuleKind(string(chargeRule.Kind))}
		if chargeRule.Amount.Set {
			rule.Amount = string(chargeRule.Amount.Value)
		}
		if chargeRule.Currency.Set {
			rule.Currency = string(chargeRule.Currency.Value)
		}
		if chargeRule.Rate.Set {
			rule.Rate = string(chargeRule.Rate.Value)
		}
		lineItems = append(lineItems, invoicing.TemplateLineItem{
			ID: uuid.UUID(item.ID), Code: item.Code, Description: item.Description, ChargeRule: rule,
			TaxCategory: invoicing.TaxCategory{Code: item.TaxCategory},
		})
	}
	applicability := invoicing.Applicability{Mode: invoicing.ApplicabilityMode(string(data.Applicability.Mode))}
	if data.Applicability.Dimensions.Set {
		applicability.Dimensions = make(map[string]string, len(data.Applicability.Dimensions.Value))
		for key, value := range data.Applicability.Dimensions.Value {
			applicability.Dimensions[key] = value
		}
	}
	payment := invoicing.PaymentMethod{Method: invoicing.PaymentMethodKind(string(data.PaymentMethod.Method)), TermsDays: data.PaymentMethod.TermsDays}
	if data.PaymentMethod.PaymentReference.Set {
		payment.PaymentReference = data.PaymentMethod.PaymentReference.Value
	}
	command := invoicing.InvoiceTemplateCommand{
		Action: string(data.Action), TemplateID: templateID, AccountingScopeID: scopeID,
		Code: data.Code, Name: data.Name, BillingFrequency: invoicing.BillingFrequency(string(data.BillingFrequency)),
		LineItems: lineItems, PaymentMethod: payment, Applicability: applicability,
		EffectiveDateFrom: data.EffectiveFrom, EffectiveDateTo: commandEffectiveDateTo(data.EffectiveTo),
		ExpectedVersion: expectedVersion, IdempotencyKey: params.IdempotencyKey,
		CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.InvoiceTemplateService.Execute(ctx, invoicing.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapInvoiceTemplateConfigureError(err, correlationID), nil
	}
	return establishedInvoiceTemplateResult(result, correlationID), nil
}

func (handler IdentityHandler) InvListInvoiceTemplates(ctx context.Context, params generated.InvListInvoiceTemplatesParams) (generated.InvListInvoiceTemplatesRes, error) {
	correlationID := invoiceTemplateListCorrelation(params)
	if handler.InvoiceTemplateService == nil {
		return invoiceTemplateListProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return invoiceTemplateListProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	values, err := handler.InvoiceTemplateService.ListSafe(ctx, invoicing.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, optionalScopeID(params.XAccountingScopeID))
	if err != nil {
		return mapInvoiceTemplateListError(err, correlationID), nil
	}
	// The cursor is the last UUID returned by this handler, so the page order
	// must use that same stable key regardless of the repository's display sort.
	sort.Slice(values, func(i, j int) bool { return bytes.Compare(values[i].ID[:], values[j].ID[:]) < 0 })
	if params.PageAfter.Set {
		after, parseErr := uuid.Parse(strings.TrimSpace(params.PageAfter.Value))
		if parseErr != nil {
			return invoiceTemplateListProblem(http.StatusBadRequest, "INVALID_PAGE", "The page cursor is invalid.", correlationID), nil
		}
		filtered := values[:0]
		for _, value := range values {
			if bytes.Compare(value.ID[:], after[:]) > 0 {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	pageSize := 50
	if params.PageSize.Set {
		pageSize = params.PageSize.Value
	}
	hasMore := len(values) > pageSize
	if hasMore {
		values = values[:pageSize]
	}
	page := map[string]any{"size": pageSize, "hasMore": hasMore}
	if hasMore && len(values) > 0 {
		page["after"] = values[len(values)-1].ID.String()
	}
	data := generated.EstablishedResultData{"templates": mustRaw(values), "page": mustRaw(page)}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(uuid.Nil), AggregateVersion: 0, CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/invoicing/reference/invoice-templates"}, Data: data}, nil
}

func (handler IdentityHandler) InvGetInvoiceTemplate(ctx context.Context, params generated.InvGetInvoiceTemplateParams) (generated.InvGetInvoiceTemplateRes, error) {
	correlationID := invoiceTemplateGetCorrelation(params)
	if handler.InvoiceTemplateService == nil {
		return invoiceTemplateGetProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return invoiceTemplateGetProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	value, err := handler.InvoiceTemplateService.GetSafe(ctx, invoicing.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, uuid.UUID(params.InvoiceTemplateId), optionalScopeID(params.XAccountingScopeID))
	if err != nil {
		return mapInvoiceTemplateGetError(err, correlationID), nil
	}
	data := generated.EstablishedResultData{"template": mustRaw(value)}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(value.ID), AggregateVersion: int(value.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/invoicing/reference/invoice-templates/" + value.ID.String()}, Data: data}, nil
}

func commandScopeID(requestScope generated.UUID, headerScope generated.OptUUID) (uuid.UUID, error) {
	scopeID := uuid.UUID(requestScope)
	if scopeID == uuid.Nil {
		return uuid.Nil, errors.New("accounting scope is required")
	}
	if headerScope.Set && uuid.UUID(headerScope.Value) != scopeID {
		return uuid.Nil, errCommandScopeConflict
	}
	return scopeID, nil
}

func commandExpectedVersion(body generated.OptInt, ifMatch generated.OptString) (*aggregateversion.AggregateVersion, error) {
	var result *aggregateversion.AggregateVersion
	if body.Set {
		value, err := aggregateversion.FromInt64(int64(body.Value))
		if err != nil {
			return nil, err
		}
		result = &value
	}
	if ifMatch.Set {
		value, err := parseIfMatchVersion(ifMatch.Value)
		if err != nil {
			return nil, err
		}
		if result != nil && result.Value() != value.Value() {
			return nil, errCommandVersionConflict
		}
		result = &value
	}
	return result, nil
}

func commandEffectiveDateTo(value generated.OptDate) *time.Time {
	if !value.Set {
		return nil
	}
	result := value.Value
	return &result
}

func invoiceTemplateConfigureCorrelation(params generated.InvConfigureInvoiceTemplatesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func invoiceTemplateListCorrelation(params generated.InvListInvoiceTemplatesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func invoiceTemplateGetCorrelation(params generated.InvGetInvoiceTemplateParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedInvoiceTemplateResult(result invoicing.InvoiceTemplateCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"template":          mustRaw(result.Template),
		"templateId":        mustRaw(result.Template.ID.String()),
		"aggregateVersion":  mustRaw(result.Template.Version.Value()),
		"validationOutcome": mustRaw(result.ValidationOutcome),
		"decisionReference": mustRaw(result.DecisionReference.String()),
		"policyReference":   mustRaw(result.PolicyReference),
		"replayed":          mustRaw(result.Replayed),
		"owner":             mustRaw("authorized billing operator (audit-backed)"),
		"nextAction":        mustRaw("review and publish when approved"),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.Template.ID), AggregateVersion: int(result.Template.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/invoicing/reference/invoice-templates/" + result.Template.ID.String()}, Data: data}
}

func mapInvoiceTemplateConfigureError(err error, correlationID uuid.UUID) generated.InvConfigureInvoiceTemplatesRes {
	switch {
	case errors.Is(err, invoicing.ErrInvoiceTemplateAuthorizationDenied):
		return invoiceTemplateConfigureProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested invoice-template scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateAuthorizationStale):
		return invoiceTemplateConfigureProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateAuthorizationUnavailable), errors.Is(err, invoicing.ErrInvoiceTemplateAuditUnavailable), errors.Is(err, invoicing.ErrInvoiceTemplateReferenceUnavailable):
		return invoiceTemplateConfigureProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template operation could not be completed.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateVersionConflict):
		return invoiceTemplateConfigureProblem(http.StatusConflict, "VERSION_CONFLICT", "The invoice template changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateIdempotencyConflict):
		return invoiceTemplateConfigureProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateCommandInProgress):
		return invoiceTemplateConfigureProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateNotFound):
		return invoiceTemplateConfigureProblem(http.StatusConflict, "INVOICE_TEMPLATE_NOT_FOUND", "The requested invoice template does not exist.", correlationID)
	case errors.Is(err, invoicing.ErrInvalidInvoiceTemplateCommand):
		return invoiceTemplateConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The invoice-template command is invalid.", correlationID)
	case errors.Is(err, invoicing.ErrInvalidInvoiceTemplate), errors.Is(err, invoicing.ErrInvoiceTemplateDuplicate), errors.Is(err, invoicing.ErrInvoiceTemplateReferenceInvalid), errors.Is(err, invoicing.ErrInvoiceTemplateDurableCommandFailed):
		return invoiceTemplateConfigureProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The invoice-template command violates a configuration rule.", correlationID)
	default:
		return invoiceTemplateConfigureProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template operation could not be completed.", correlationID)
	}
}

func invoiceTemplateConfigureProblem(status int, code, detail string, correlationID uuid.UUID) generated.InvConfigureInvoiceTemplatesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.InvConfigureInvoiceTemplatesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.InvConfigureInvoiceTemplatesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.InvConfigureInvoiceTemplatesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.InvConfigureInvoiceTemplatesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.InvConfigureInvoiceTemplatesServiceUnavailable(problem)
		return &value
	}
}

func mapInvoiceTemplateListError(err error, correlationID uuid.UUID) generated.InvListInvoiceTemplatesRes {
	if errors.Is(err, invoicing.ErrInvoiceTemplateAuthorizationDenied) {
		return invoiceTemplateListProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested invoice-template scope is outside the actor scope.", correlationID)
	}
	return invoiceTemplateListProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template list could not be completed.", correlationID)
}

func invoiceTemplateListProblem(status int, code, detail string, correlationID uuid.UUID) generated.InvListInvoiceTemplatesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.InvListInvoiceTemplatesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.InvListInvoiceTemplatesForbidden(problem)
		return &value
	default:
		value := generated.InvListInvoiceTemplatesServiceUnavailable(problem)
		return &value
	}
}

func mapInvoiceTemplateGetError(err error, correlationID uuid.UUID) generated.InvGetInvoiceTemplateRes {
	switch {
	case errors.Is(err, invoicing.ErrInvoiceTemplateNotFound):
		return invoiceTemplateGetProblem(http.StatusNotFound, "INVOICE_TEMPLATE_NOT_FOUND", "The requested invoice template does not exist.", correlationID)
	case errors.Is(err, invoicing.ErrInvoiceTemplateAuthorizationDenied):
		return invoiceTemplateGetProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested invoice-template scope is outside the actor scope.", correlationID)
	default:
		return invoiceTemplateGetProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The invoice-template read could not be completed.", correlationID)
	}
}

func invoiceTemplateGetProblem(status int, code, detail string, correlationID uuid.UUID) generated.InvGetInvoiceTemplateRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.InvGetInvoiceTemplateBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.InvGetInvoiceTemplateForbidden(problem)
		return &value
	case http.StatusNotFound:
		value := generated.InvGetInvoiceTemplateNotFound(problem)
		return &value
	default:
		value := generated.InvGetInvoiceTemplateServiceUnavailable(problem)
		return &value
	}
}

func (handler IdentityHandler) InvConfigureBillingSchedules(ctx context.Context, request *generated.InvConfigureBillingSchedulesCommandRequest, params generated.InvConfigureBillingSchedulesParams) (generated.InvConfigureBillingSchedulesRes, error) {
	correlationID := billingScheduleConfigureCorrelation(params)
	if handler.BillingScheduleService == nil {
		return billingScheduleConfigureProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return billingScheduleConfigureProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return billingScheduleConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The billing-schedule command envelope is invalid.", correlationID), nil
	}
	scopeID, err := commandScopeID(request.AccountingScopeId, params.XAccountingScopeID)
	if err != nil {
		if errors.Is(err, errCommandScopeConflict) {
			return billingScheduleConfigureProblem(http.StatusConflict, "SCOPE_CONFLICT", "The accounting scope in the request and header do not agree.", correlationID), nil
		}
		return billingScheduleConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The accounting scope in the request is invalid.", correlationID), nil
	}
	expectedVersion, err := commandExpectedVersion(request.ExpectedVersion, params.IfMatch)
	if err != nil {
		if errors.Is(err, errCommandVersionConflict) {
			return billingScheduleConfigureProblem(http.StatusConflict, "VERSION_CONFLICT", "The expected and If-Match versions do not agree.", correlationID), nil
		}
		return billingScheduleConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The expected or If-Match versions are invalid.", correlationID), nil
	}
	data := request.Data
	scheduleID := uuid.Nil
	if data.BillingScheduleId.Set {
		scheduleID = uuid.UUID(data.BillingScheduleId.Value)
	}
	charges := make([]invoicing.ScheduledCharge, 0, len(data.Charges))
	for _, charge := range data.Charges {
		charges = append(charges, invoicing.ScheduledCharge{
			ID: uuid.UUID(charge.ID), TemplateID: uuid.UUID(charge.TemplateId), TemplateVersion: int64(charge.TemplateVersion),
			TemplateLineItemID: uuid.UUID(charge.TemplateLineItemId), Quantity: string(charge.Quantity), Amount: string(charge.Amount), Currency: string(charge.Currency),
		})
	}
	command := invoicing.BillingScheduleCommand{
		Action: string(data.Action), BillingScheduleID: scheduleID, AccountingScopeID: scopeID, CustomerID: uuid.UUID(data.CustomerId),
		ContractReference: data.ContractReference, Charges: charges, NextBillingDate: data.NextBillingDate, TimeZone: data.TimeZone,
		Status: invoicing.ScheduleStatus(string(data.Status)), EffectiveDateFrom: data.EffectiveFrom, EffectiveDateTo: commandEffectiveDateTo(data.EffectiveTo),
		ExpectedVersion: expectedVersion, IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(),
	}
	result, err := handler.BillingScheduleService.Execute(ctx, invoicing.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, command)
	if err != nil {
		return mapBillingScheduleConfigureError(err, correlationID), nil
	}
	return establishedBillingScheduleResult(result, correlationID), nil
}

func (handler IdentityHandler) InvListBillingSchedules(ctx context.Context, params generated.InvListBillingSchedulesParams) (generated.InvListBillingSchedulesRes, error) {
	correlationID := billingScheduleListCorrelation(params)
	if handler.BillingScheduleService == nil {
		return billingScheduleListProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return billingScheduleListProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	values, err := handler.BillingScheduleService.ListSafe(ctx, invoicing.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, optionalScopeID(params.XAccountingScopeID))
	if err != nil {
		return mapBillingScheduleListError(err, correlationID), nil
	}
	sort.Slice(values, func(i, j int) bool { return bytes.Compare(values[i].ID[:], values[j].ID[:]) < 0 })
	if params.PageAfter.Set {
		after, parseErr := uuid.Parse(strings.TrimSpace(params.PageAfter.Value))
		if parseErr != nil {
			return billingScheduleListProblem(http.StatusBadRequest, "INVALID_PAGE", "The page cursor is invalid.", correlationID), nil
		}
		filtered := values[:0]
		for _, value := range values {
			if bytes.Compare(value.ID[:], after[:]) > 0 {
				filtered = append(filtered, value)
			}
		}
		values = filtered
	}
	pageSize := 50
	if params.PageSize.Set {
		pageSize = params.PageSize.Value
	}
	hasMore := len(values) > pageSize
	if hasMore {
		values = values[:pageSize]
	}
	page := map[string]any{"size": pageSize, "hasMore": hasMore}
	if hasMore && len(values) > 0 {
		page["after"] = values[len(values)-1].ID.String()
	}
	data := generated.EstablishedResultData{"billingSchedules": mustRaw(values), "page": mustRaw(page)}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(uuid.Nil), AggregateVersion: 0, CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/invoicing/reference/billing-schedules"}, Data: data}, nil
}

func (handler IdentityHandler) InvGetBillingSchedule(ctx context.Context, params generated.InvGetBillingScheduleParams) (generated.InvGetBillingScheduleRes, error) {
	correlationID := billingScheduleGetCorrelation(params)
	if handler.BillingScheduleService == nil {
		return billingScheduleGetProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule read service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return billingScheduleGetProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requesting actor is not authorized.", correlationID), nil
	}
	value, err := handler.BillingScheduleService.GetSafe(ctx, invoicing.Actor{UserID: actor.UserID, SubjectReference: glActorSubject(actor)}, uuid.UUID(params.BillingScheduleId), optionalScopeID(params.XAccountingScopeID))
	if err != nil {
		return mapBillingScheduleGetError(err, correlationID), nil
	}
	data := generated.EstablishedResultData{"billingSchedule": mustRaw(value)}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(value.ID), AggregateVersion: int(value.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/invoicing/reference/billing-schedules/" + value.ID.String()}, Data: data}, nil
}

func billingScheduleConfigureCorrelation(params generated.InvConfigureBillingSchedulesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func billingScheduleListCorrelation(params generated.InvListBillingSchedulesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func billingScheduleGetCorrelation(params generated.InvGetBillingScheduleParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func establishedBillingScheduleResult(result invoicing.BillingScheduleCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"billingSchedule": mustRaw(result.Schedule), "billingScheduleId": mustRaw(result.Schedule.ID.String()),
		"aggregateVersion": mustRaw(result.Schedule.Version.Value()), "validationOutcome": mustRaw(result.ValidationOutcome),
		"decisionReference": mustRaw(result.DecisionReference.String()), "policyReference": mustRaw(result.PolicyReference),
		"replayed": mustRaw(result.Replayed), "owner": mustRaw("authorized billing operator (audit-backed)"),
		"nextAction": mustRaw("review and maintain when approved"),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.Schedule.ID), AggregateVersion: int(result.Schedule.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/invoicing/reference/billing-schedules/" + result.Schedule.ID.String()}, Data: data}
}

func mapBillingScheduleConfigureError(err error, correlationID uuid.UUID) generated.InvConfigureBillingSchedulesRes {
	switch {
	case errors.Is(err, invoicing.ErrBillingScheduleAuthorizationDenied):
		return billingScheduleConfigureProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested billing-schedule scope is outside the administering actor scope.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleAuthorizationStale):
		return billingScheduleConfigureProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleAuthorizationUnavailable), errors.Is(err, invoicing.ErrBillingScheduleAuditUnavailable), errors.Is(err, invoicing.ErrBillingScheduleReferenceUnavailable):
		return billingScheduleConfigureProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule operation could not be completed.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleVersionConflict):
		return billingScheduleConfigureProblem(http.StatusConflict, "VERSION_CONFLICT", "The billing schedule changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleIdempotencyConflict):
		return billingScheduleConfigureProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleCommandInProgress):
		return billingScheduleConfigureProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleNotFound):
		return billingScheduleConfigureProblem(http.StatusConflict, "BILLING_SCHEDULE_NOT_FOUND", "The requested billing schedule does not exist.", correlationID)
	case errors.Is(err, invoicing.ErrInvalidBillingScheduleCommand):
		return billingScheduleConfigureProblem(http.StatusBadRequest, "INVALID_REQUEST", "The billing-schedule command is invalid.", correlationID)
	case errors.Is(err, invoicing.ErrInvalidBillingSchedule), errors.Is(err, invoicing.ErrBillingScheduleDuplicate), errors.Is(err, invoicing.ErrBillingScheduleReferenceInvalid), errors.Is(err, invoicing.ErrBillingScheduleDurableCommandFailed):
		return billingScheduleConfigureProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The billing-schedule command violates a configuration rule.", correlationID)
	default:
		return billingScheduleConfigureProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule operation could not be completed.", correlationID)
	}
}

func billingScheduleConfigureProblem(status int, code, detail string, correlationID uuid.UUID) generated.InvConfigureBillingSchedulesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.InvConfigureBillingSchedulesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.InvConfigureBillingSchedulesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.InvConfigureBillingSchedulesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.InvConfigureBillingSchedulesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.InvConfigureBillingSchedulesServiceUnavailable(problem)
		return &value
	}
}

func mapBillingScheduleListError(err error, correlationID uuid.UUID) generated.InvListBillingSchedulesRes {
	if errors.Is(err, invoicing.ErrBillingScheduleAuthorizationDenied) {
		return billingScheduleListProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested billing-schedule scope is outside the actor scope.", correlationID)
	}
	return billingScheduleListProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule list could not be completed.", correlationID)
}

func billingScheduleListProblem(status int, code, detail string, correlationID uuid.UUID) generated.InvListBillingSchedulesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.InvListBillingSchedulesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.InvListBillingSchedulesForbidden(problem)
		return &value
	default:
		value := generated.InvListBillingSchedulesServiceUnavailable(problem)
		return &value
	}
}

func mapBillingScheduleGetError(err error, correlationID uuid.UUID) generated.InvGetBillingScheduleRes {
	switch {
	case errors.Is(err, invoicing.ErrBillingScheduleNotFound):
		return billingScheduleGetProblem(http.StatusNotFound, "BILLING_SCHEDULE_NOT_FOUND", "The requested billing schedule does not exist.", correlationID)
	case errors.Is(err, invoicing.ErrBillingScheduleAuthorizationDenied):
		return billingScheduleGetProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested billing-schedule scope is outside the actor scope.", correlationID)
	default:
		return billingScheduleGetProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The billing-schedule read could not be completed.", correlationID)
	}
}

func billingScheduleGetProblem(status int, code, detail string, correlationID uuid.UUID) generated.InvGetBillingScheduleRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.InvGetBillingScheduleBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.InvGetBillingScheduleForbidden(problem)
		return &value
	case http.StatusNotFound:
		value := generated.InvGetBillingScheduleNotFound(problem)
		return &value
	default:
		value := generated.InvGetBillingScheduleServiceUnavailable(problem)
		return &value
	}
}
