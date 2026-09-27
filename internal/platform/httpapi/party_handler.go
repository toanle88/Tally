package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

// partyCommandData is intentionally private. The generated OpenAPI contract
// remains the generic command envelope; this is the v1 semantic payload owned
// by the OMD implementation.
type partyCommandData struct {
	Action               string                                  `json:"action"`
	PartyID              string                                  `json:"partyId"`
	ScopeID              string                                  `json:"scopeId"`
	Name                 string                                  `json:"name"`
	PartyType            string                                  `json:"partyType"`
	Status               string                                  `json:"status"`
	TaxIdentifier        *string                                 `json:"taxIdentifier"`
	ContactMethods       []partyContactMethodData                `json:"contactMethods"`
	Addresses            []partyAddressData                      `json:"addresses"`
	Classifications      []partyClassificationData               `json:"classifications"`
	BankDetailReferences []partyBankReferenceData                `json:"bankDetailReferences"`
	Approval             *organization.ApprovalDecisionReference `json:"approval"`
}

type partyContactMethodData struct {
	Type  string `json:"type"`
	Value string `json:"value"`
	Label string `json:"label"`
}

type partyAddressData struct {
	Type        string `json:"type"`
	Line1       string `json:"line1"`
	Line2       string `json:"line2"`
	Locality    string `json:"locality"`
	Region      string `json:"region"`
	PostalCode  string `json:"postalCode"`
	CountryCode string `json:"countryCode"`
}

type partyClassificationData struct {
	Code  string `json:"code"`
	Value string `json:"value"`
}

type partyBankReferenceData struct {
	Reference        string `json:"reference"`
	ProviderCode     string `json:"providerCode"`
	ConsentReference string `json:"consentReference"`
}

var (
	errPartyTransportConflict = errors.New("party transport conflict")
	errPartyTransportInvalid  = errors.New("party transport invalid")
)

func (handler IdentityHandler) OmdMaintainParties(ctx context.Context, request *generated.CommandRequest, params generated.OmdMaintainPartiesParams) (generated.OmdMaintainPartiesRes, error) {
	correlationID := correlationFromOmdPartiesParams(params)
	ctx, finish := beginPartyTelemetry(handler.Instrumentation, ctx)
	outcome := "failed"
	defer func() { finish(outcome) }()
	if handler.PartyService == nil {
		return omdPartyProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The party service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdPartyProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return omdPartyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The party command data is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil || containsForbiddenPartyKey(data) {
		return omdPartyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The party command data is invalid.", correlationID), nil
	}
	payload, err := decodePartyCommandData(data)
	if err != nil {
		return omdPartyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The party command data is invalid.", correlationID), nil
	}
	command, err := partyCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		if errors.Is(err, errPartyTransportConflict) {
			return omdPartyProblem(http.StatusConflict, "VERSION_CONFLICT", "The party version or accounting scope does not agree with the command envelope.", correlationID), nil
		}
		return omdPartyProblem(http.StatusBadRequest, "INVALID_REQUEST", "The party command data is invalid.", correlationID), nil
	}
	if _, present := telemetry.FromContext(ctx); present {
		if enriched, causationErr := telemetry.WithCausation(ctx, uuid.UUID(request.CommandId)); causationErr == nil {
			ctx = enriched
		}
	}
	result, err := handler.PartyService.Execute(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapOmdPartyError(err, correlationID), nil
	}
	outcome = "established"
	if result.Replayed {
		outcome = "replayed"
	}
	return establishedPartyResult(result, correlationID), nil
}

func decodePartyCommandData(data []byte) (partyCommandData, error) {
	var payload partyCommandData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return partyCommandData{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return partyCommandData{}, errPartyTransportInvalid
	}
	return payload, nil
}

func partyCommandFromTransport(request *generated.CommandRequest, params generated.OmdMaintainPartiesParams, payload partyCommandData, correlationID uuid.UUID) (organization.PartyCommand, error) {
	command := organization.PartyCommand{
		Action:         strings.ToLower(strings.TrimSpace(payload.Action)),
		ScopeID:        uuid.Nil,
		Name:           strings.TrimSpace(payload.Name),
		PartyType:      organization.PartyType(strings.ToLower(strings.TrimSpace(payload.PartyType))),
		Status:         organization.PartyStatus(strings.ToLower(strings.TrimSpace(payload.Status))),
		TaxIdentifier:  payload.TaxIdentifier,
		IdempotencyKey: params.IdempotencyKey,
		CorrelationID:  correlationID.String(),
		CausationID:    uuid.UUID(request.CommandId).String(),
		Approval:       payload.Approval,
	}
	if !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil {
		return command, errPartyTransportInvalid
	}
	command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	if payload.ScopeID != "" {
		payloadScope, err := uuid.Parse(strings.TrimSpace(payload.ScopeID))
		if err != nil {
			return command, errPartyTransportInvalid
		}
		if payloadScope != command.ScopeID {
			return command, errPartyTransportConflict
		}
	}
	if payload.PartyID != "" {
		partyID, err := uuid.Parse(strings.TrimSpace(payload.PartyID))
		if err != nil {
			return command, errPartyTransportInvalid
		}
		command.PartyID = partyID
	}
	var bodyVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, errPartyTransportInvalid
		}
		bodyVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, errPartyTransportInvalid
		}
		if bodyVersion != nil && bodyVersion.Value() != headerVersion.Value() {
			return command, errPartyTransportConflict
		}
		command.ExpectedVersion = &headerVersion
	} else {
		command.ExpectedVersion = bodyVersion
	}
	if command.Action == organization.PartyActionCreate && command.ExpectedVersion != nil {
		return command, errPartyTransportInvalid
	}
	if command.Action == organization.PartyActionMaintain && !params.IfMatch.Set {
		return command, errPartyTransportInvalid
	}
	for _, value := range payload.ContactMethods {
		command.ContactMethods = append(command.ContactMethods, organization.PartyContactMethod{Type: value.Type, Value: value.Value, Label: value.Label})
	}
	for _, value := range payload.Addresses {
		command.Addresses = append(command.Addresses, organization.PartyAddress{Type: value.Type, Line1: value.Line1, Line2: value.Line2, Locality: value.Locality, Region: value.Region, PostalCode: value.PostalCode, CountryCode: value.CountryCode})
	}
	for _, value := range payload.Classifications {
		command.Classifications = append(command.Classifications, organization.PartyClassification{Code: value.Code, Value: value.Value})
	}
	for _, value := range payload.BankDetailReferences {
		command.BankDetailReferences = append(command.BankDetailReferences, organization.PartyBankDetailReference{Reference: value.Reference, ProviderCode: value.ProviderCode, ConsentReference: value.ConsentReference})
	}
	if err := command.Validate(); err != nil {
		return command, errPartyTransportInvalid
	}
	return command, nil
}

func containsForbiddenPartyKey(data []byte) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return true
	}
	return forbiddenPartyValue(value)
}

func forbiddenPartyValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
			switch normalized {
			case "account", "accountnumber", "iban", "routingnumber", "swift", "credential", "credentials", "password", "token", "secret", "rawaccount", "rawbankdetails":
				return true
			}
			if forbiddenPartyValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if forbiddenPartyValue(child) {
				return true
			}
		}
	}
	return false
}

func establishedPartyResult(result organization.PartyCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"party":             mustRaw(result.Party),
		"status":            mustRaw(result.Party.Status),
		"aggregateVersion":  mustRaw(result.Party.Version.Value()),
		"validationOutcome": mustRaw(result.ValidationOutcome),
		"bankControl":       mustRaw(result.BankControl),
		"decisionReference": mustRaw(result.DecisionReference.String()),
		"policyReference":   mustRaw(result.PolicyReference),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.Party.ID), AggregateVersion: int(result.Party.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/configuration/maintain-parties"}, Data: data}
}

func mapOmdPartyError(err error, correlationID uuid.UUID) generated.OmdMaintainPartiesRes {
	switch {
	case errors.Is(err, organization.ErrPartyAuthorizationDenied), errors.Is(err, organization.ErrPartyFieldAuthorizationDenied):
		return omdPartyProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested party fields or scope are outside the administering actor authority.", correlationID)
	case errors.Is(err, organization.ErrPartyAuthorizationStale):
		return omdPartyProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, organization.ErrPartyVersionConflict):
		return omdPartyProblem(http.StatusConflict, "VERSION_CONFLICT", "The party changed after it was loaded. Refresh and retry with the current version.", correlationID)
	case errors.Is(err, organization.ErrPartyIdempotencyConflict):
		return omdPartyProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, organization.ErrPartyCommandInProgress):
		return omdPartyProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, organization.ErrPartyNotFound):
		return omdPartyProblem(http.StatusConflict, "PARTY_NOT_FOUND", "The requested party does not exist.", correlationID)
	case errors.Is(err, organization.ErrPartyBankApprovalStale):
		return omdPartyProblem(http.StatusConflict, "BANK_CONTROL_STALE", "The bank-control decision is stale. Refresh and retry through the approval flow.", correlationID)
	case errors.Is(err, organization.ErrPartyAuthorizationUnavailable), errors.Is(err, organization.ErrPartyFieldAuthorizationUnavailable), errors.Is(err, organization.ErrPartyAuditUnavailable), errors.Is(err, organization.ErrPartyBankReferenceUnavailable), errors.Is(err, organization.ErrPartyBankControlUnavailable), errors.Is(err, organization.ErrPartyDurableCommandFailed), errors.Is(err, organization.ErrInvalidPartyService):
		return omdPartyProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The party operation could not be completed.", correlationID)
	case errors.Is(err, organization.ErrPartyBankApprovalRejected), errors.Is(err, organization.ErrPartyBankReferenceInvalid), errors.Is(err, organization.ErrInvalidParty), errors.Is(err, organization.ErrInvalidPartyCommand), errors.Is(err, organization.ErrPartyDuplicate):
		return omdPartyProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The party command violates an organization rule.", correlationID)
	default:
		return omdPartyProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The party operation could not be completed.", correlationID)
	}
}

func omdPartyProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdMaintainPartiesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdMaintainPartiesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdMaintainPartiesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.OmdMaintainPartiesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.OmdMaintainPartiesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.OmdMaintainPartiesServiceUnavailable(problem)
		return &value
	}
}

func correlationFromOmdPartiesParams(params generated.OmdMaintainPartiesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func beginPartyTelemetry(instrumentation *telemetry.Instrumentation, ctx context.Context) (context.Context, func(string)) {
	if instrumentation == nil {
		return ctx, func(string) {}
	}
	ctx, span := instrumentation.StartSpan(ctx, "organization.omd_maintain_parties", telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainParties", DataClassification: "highly_restricted"})
	finished := false
	return ctx, func(result string) {
		if finished {
			return
		}
		finished = true
		instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainParties", Result: result, DataClassification: "highly_restricted"})
		instrumentation.AddCommand(ctx, 1, "organization", "omdMaintainParties", result)
		span.End()
	}
}
