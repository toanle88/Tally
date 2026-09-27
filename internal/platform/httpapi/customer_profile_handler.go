package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

// customerProfileCommandData is intentionally private. The generated
// OpenAPI contract remains the generic command envelope; this is the v1
// semantic payload owned by the OMD implementation.
type customerProfileCommandData struct {
	Action            string                                  `json:"action"`
	CustomerProfileID string                                  `json:"customerProfileId"`
	ScopeID           string                                  `json:"scopeId"`
	PartyID           string                                  `json:"partyId"`
	PartyVersion      *int64                                  `json:"partyVersion"`
	CreditTerms       string                                  `json:"creditTerms"`
	CreditLimit       *customerProfileCreditLimitData         `json:"creditLimit"`
	BillingPreference string                                  `json:"billingPreference"`
	TaxTreatment      string                                  `json:"taxTreatment"`
	EffectiveFrom     string                                  `json:"effectiveFrom"`
	EffectiveTo       *string                                 `json:"effectiveTo"`
	Approval          *organization.ApprovalDecisionReference `json:"approval"`
}

type customerProfileCreditLimitData struct {
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

var (
	errCustomerProfileTransportConflict = errors.New("customer profile transport conflict")
	errCustomerProfileTransportInvalid  = errors.New("customer profile transport invalid")
)

func (handler IdentityHandler) OmdMaintainCustomerProfiles(ctx context.Context, request *generated.CommandRequest, params generated.OmdMaintainCustomerProfilesParams) (generated.OmdMaintainCustomerProfilesRes, error) {
	correlationID := correlationFromOmdCustomerProfilesParams(params)
	ctx, finish := beginCustomerProfileTelemetry(handler.Instrumentation, ctx)
	outcome := "failed"
	defer func() { finish(outcome) }()
	if handler.CustomerProfileService == nil {
		return omdCustomerProfileProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The customer-profile service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdCustomerProfileProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return omdCustomerProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The customer-profile command data is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil || containsForbiddenCustomerProfileKey(data) {
		return omdCustomerProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The customer-profile command data is invalid.", correlationID), nil
	}
	payload, err := decodeCustomerProfileCommandData(data)
	if err != nil {
		return omdCustomerProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The customer-profile command data is invalid.", correlationID), nil
	}
	command, err := customerProfileCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		if errors.Is(err, errCustomerProfileTransportConflict) {
			return omdCustomerProfileProblem(http.StatusConflict, "VERSION_CONFLICT", "The customer-profile version, Party version, or accounting scope does not agree with the command envelope.", correlationID), nil
		}
		return omdCustomerProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The customer-profile command data is invalid.", correlationID), nil
	}
	if _, present := telemetry.FromContext(ctx); present {
		if enriched, causationErr := telemetry.WithCausation(ctx, uuid.UUID(request.CommandId)); causationErr == nil {
			ctx = enriched
		}
	}
	result, err := handler.CustomerProfileService.Execute(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapOmdCustomerProfileError(err, correlationID), nil
	}
	outcome = "established"
	if result.Replayed {
		outcome = "replayed"
	}
	return establishedCustomerProfileResult(result, correlationID), nil
}

func decodeCustomerProfileCommandData(data []byte) (customerProfileCommandData, error) {
	var payload customerProfileCommandData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return customerProfileCommandData{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return customerProfileCommandData{}, errCustomerProfileTransportInvalid
	}
	return payload, nil
}

func customerProfileCommandFromTransport(request *generated.CommandRequest, params generated.OmdMaintainCustomerProfilesParams, payload customerProfileCommandData, correlationID uuid.UUID) (organization.CustomerProfileCommand, error) {
	command := organization.CustomerProfileCommand{
		Action: strings.ToLower(strings.TrimSpace(payload.Action)), ScopeID: uuid.Nil, PartyID: uuid.Nil,
		CreditTerms: strings.ToLower(strings.TrimSpace(payload.CreditTerms)), BillingPreference: strings.ToLower(strings.TrimSpace(payload.BillingPreference)), TaxTreatment: strings.ToLower(strings.TrimSpace(payload.TaxTreatment)),
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(), Approval: payload.Approval,
	}
	if !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil {
		return command, errCustomerProfileTransportInvalid
	}
	command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	if payload.ScopeID != "" {
		payloadScope, err := uuid.Parse(strings.TrimSpace(payload.ScopeID))
		if err != nil {
			return command, errCustomerProfileTransportInvalid
		}
		if payloadScope != command.ScopeID {
			return command, errCustomerProfileTransportConflict
		}
	}
	if payload.PartyID == "" {
		return command, errCustomerProfileTransportInvalid
	}
	partyID, err := uuid.Parse(strings.TrimSpace(payload.PartyID))
	if err != nil {
		return command, errCustomerProfileTransportInvalid
	}
	command.PartyID = partyID
	if payload.CustomerProfileID != "" {
		profileID, err := uuid.Parse(strings.TrimSpace(payload.CustomerProfileID))
		if err != nil {
			return command, errCustomerProfileTransportInvalid
		}
		command.CustomerProfileID = profileID
	}
	if payload.PartyVersion == nil || *payload.PartyVersion < 1 {
		return command, errCustomerProfileTransportInvalid
	}
	partyVersion, err := aggregateversion.FromInt64(*payload.PartyVersion)
	if err != nil {
		return command, errCustomerProfileTransportInvalid
	}
	command.ExpectedPartyVersion = &partyVersion
	if payload.CreditLimit == nil {
		return command, errCustomerProfileTransportInvalid
	}
	command.CreditLimit = organization.CreditLimit{Amount: payload.CreditLimit.Amount, Currency: strings.ToUpper(strings.TrimSpace(payload.CreditLimit.Currency))}
	command.EffectiveFrom, err = parseCustomerProfileDate(payload.EffectiveFrom)
	if err != nil {
		return command, errCustomerProfileTransportInvalid
	}
	if payload.EffectiveTo != nil && strings.TrimSpace(*payload.EffectiveTo) != "" {
		value, parseErr := parseCustomerProfileDate(*payload.EffectiveTo)
		if parseErr != nil {
			return command, errCustomerProfileTransportInvalid
		}
		command.EffectiveTo = &value
	}
	var bodyVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, errCustomerProfileTransportInvalid
		}
		bodyVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, errCustomerProfileTransportInvalid
		}
		if bodyVersion != nil && bodyVersion.Value() != headerVersion.Value() {
			return command, errCustomerProfileTransportConflict
		}
		command.ExpectedVersion = &headerVersion
	} else {
		command.ExpectedVersion = bodyVersion
	}
	if command.Action == organization.CustomerProfileActionCreate && command.ExpectedVersion != nil {
		return command, errCustomerProfileTransportInvalid
	}
	if command.Action == organization.CustomerProfileActionMaintain && !params.IfMatch.Set {
		return command, errCustomerProfileTransportInvalid
	}
	if err := command.Validate(); err != nil {
		return command, errCustomerProfileTransportInvalid
	}
	return command, nil
}

func parseCustomerProfileDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func containsForbiddenCustomerProfileKey(data []byte) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return true
	}
	return forbiddenCustomerProfileValue(value)
}

func forbiddenCustomerProfileValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
			switch normalized {
			case "account", "accountnumber", "iban", "routingnumber", "swift", "credential", "credentials", "password", "token", "secret", "rawaccount", "rawbankdetails", "taxidentifier", "rawtaxidentifier":
				return true
			}
			if forbiddenCustomerProfileValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if forbiddenCustomerProfileValue(child) {
				return true
			}
		}
	}
	return false
}

func establishedCustomerProfileResult(result organization.CustomerProfileCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"customerProfile":   mustRaw(result.CustomerProfile),
		"partyId":           mustRaw(result.CustomerProfile.PartyID.String()),
		"partyVersion":      mustRaw(result.CustomerProfile.PartyVersion.Value()),
		"status":            mustRaw(result.CustomerProfile.Status),
		"aggregateVersion":  mustRaw(result.CustomerProfile.Version.Value()),
		"validationOutcome": mustRaw(result.ValidationOutcome),
		"decisionReference": mustRaw(result.DecisionReference.String()),
		"policyReference":   mustRaw(result.PolicyReference),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.CustomerProfile.ID), AggregateVersion: int(result.CustomerProfile.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/configuration/maintain-customer-profiles"}, Data: data}
}

func mapOmdCustomerProfileError(err error, correlationID uuid.UUID) generated.OmdMaintainCustomerProfilesRes {
	switch {
	case errors.Is(err, organization.ErrCustomerProfileAuthorizationDenied), errors.Is(err, organization.ErrCustomerProfileFieldAuthorizationDenied):
		return omdCustomerProfileProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested customer-profile fields or scope are outside the administering actor authority.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileAuthorizationStale):
		return omdCustomerProfileProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileVersionConflict), errors.Is(err, organization.ErrCustomerProfilePartyVersionConflict):
		return omdCustomerProfileProblem(http.StatusConflict, "VERSION_CONFLICT", "The customer profile or authoritative Party changed after it was loaded. Refresh and retry with current versions.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileIdempotencyConflict):
		return omdCustomerProfileProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileCommandInProgress):
		return omdCustomerProfileProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileNotFound):
		return omdCustomerProfileProblem(http.StatusConflict, "PROFILE_NOT_FOUND", "The requested customer profile does not exist.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileDuplicate):
		return omdCustomerProfileProblem(http.StatusConflict, "DUPLICATE_PROFILE", "A customer profile already exists for this Party and accounting scope.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfilePartyNotFound), errors.Is(err, organization.ErrCustomerProfilePartyInvalid), errors.Is(err, organization.ErrCustomerProfilePartyMismatch):
		return omdCustomerProfileProblem(http.StatusUnprocessableEntity, "INVALID_PARTY_REFERENCE", "The referenced Party is missing, outside the accounting scope, or is not a customer Party.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileAuthorizationUnavailable), errors.Is(err, organization.ErrCustomerProfileFieldAuthorizationUnavailable), errors.Is(err, organization.ErrCustomerProfileAuditUnavailable), errors.Is(err, organization.ErrCustomerProfileDurableCommandFailed), errors.Is(err, organization.ErrInvalidCustomerProfileService):
		return omdCustomerProfileProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The customer-profile operation could not be completed.", correlationID)
	case errors.Is(err, organization.ErrCustomerProfileApprovalRequired), errors.Is(err, organization.ErrCustomerProfileApprovalRejected), errors.Is(err, organization.ErrInvalidCustomerProfile), errors.Is(err, organization.ErrInvalidCustomerProfileCommand):
		return omdCustomerProfileProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The customer-profile command violates an organization rule.", correlationID)
	default:
		return omdCustomerProfileProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The customer-profile operation could not be completed.", correlationID)
	}
}

func omdCustomerProfileProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdMaintainCustomerProfilesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdMaintainCustomerProfilesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdMaintainCustomerProfilesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.OmdMaintainCustomerProfilesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.OmdMaintainCustomerProfilesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.OmdMaintainCustomerProfilesServiceUnavailable(problem)
		return &value
	}
}

func correlationFromOmdCustomerProfilesParams(params generated.OmdMaintainCustomerProfilesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func beginCustomerProfileTelemetry(instrumentation *telemetry.Instrumentation, ctx context.Context) (context.Context, func(string)) {
	if instrumentation == nil {
		return ctx, func(string) {}
	}
	ctx, span := instrumentation.StartSpan(ctx, "organization.omd_maintain_customer_profiles", telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainCustomerProfiles", DataClassification: "highly_restricted"})
	finished := false
	return ctx, func(result string) {
		if finished {
			return
		}
		finished = true
		instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainCustomerProfiles", Result: result, DataClassification: "highly_restricted"})
		instrumentation.AddCommand(ctx, 1, "organization", "omdMaintainCustomerProfiles", result)
		span.End()
	}
}
