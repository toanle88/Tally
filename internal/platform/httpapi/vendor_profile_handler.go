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

// vendorProfileCommandData is the semantic v1 payload for the existing
// generic command envelope. Bank details are Party-owned and are not accepted.
type vendorProfileCommandData struct {
	Action               string                                  `json:"action"`
	VendorProfileID      string                                  `json:"vendorProfileId"`
	ScopeID              string                                  `json:"scopeId"`
	PartyID              string                                  `json:"partyId"`
	PartyVersion         *int64                                  `json:"partyVersion"`
	PaymentTerms         string                                  `json:"paymentTerms"`
	WithholdingTreatment string                                  `json:"withholdingTreatment"`
	RemittancePreference string                                  `json:"remittancePreference"`
	EffectiveFrom        string                                  `json:"effectiveFrom"`
	EffectiveTo          *string                                 `json:"effectiveTo"`
	Approval             *organization.ApprovalDecisionReference `json:"approval"`
}

var (
	errVendorProfileTransportConflict = errors.New("vendor profile transport conflict")
	errVendorProfileTransportInvalid  = errors.New("vendor profile transport invalid")
)

func (handler IdentityHandler) OmdMaintainVendorProfiles(ctx context.Context, request *generated.CommandRequest, params generated.OmdMaintainVendorProfilesParams) (generated.OmdMaintainVendorProfilesRes, error) {
	correlationID := correlationFromOmdVendorProfilesParams(params)
	ctx, finish := beginVendorProfileTelemetry(handler.Instrumentation, ctx)
	outcome := "failed"
	defer func() { finish(outcome) }()
	if handler.VendorProfileService == nil {
		return omdVendorProfileProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The vendor-profile service is unavailable.", correlationID), nil
	}
	actor, ok := identity.ActorFromContext(ctx)
	if !ok {
		return omdVendorProfileProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The administering actor is not authorized.", correlationID), nil
	}
	if request == nil || uuid.UUID(request.CommandId) == uuid.Nil {
		return omdVendorProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The vendor-profile command data is invalid.", correlationID), nil
	}
	data, err := json.Marshal(request.Data)
	if err != nil || containsForbiddenVendorProfileKey(data) {
		return omdVendorProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The vendor-profile command data is invalid.", correlationID), nil
	}
	payload, err := decodeVendorProfileCommandData(data)
	if err != nil {
		return omdVendorProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The vendor-profile command data is invalid.", correlationID), nil
	}
	command, err := vendorProfileCommandFromTransport(request, params, payload, correlationID)
	if err != nil {
		if errors.Is(err, errVendorProfileTransportConflict) {
			return omdVendorProfileProblem(http.StatusConflict, "VERSION_CONFLICT", "The vendor-profile version, Party version, or accounting scope does not agree with the command envelope.", correlationID), nil
		}
		return omdVendorProfileProblem(http.StatusBadRequest, "INVALID_REQUEST", "The vendor-profile command data is invalid.", correlationID), nil
	}
	if _, present := telemetry.FromContext(ctx); present {
		if enriched, causationErr := telemetry.WithCausation(ctx, uuid.UUID(request.CommandId)); causationErr == nil {
			ctx = enriched
		}
	}
	result, err := handler.VendorProfileService.Execute(ctx, organization.Actor{UserID: actor.UserID, SubjectReference: actor.UserID.String()}, command)
	if err != nil {
		return mapOmdVendorProfileError(err, correlationID), nil
	}
	outcome = "established"
	if result.Replayed {
		outcome = "replayed"
	}
	return establishedVendorProfileResult(result, correlationID), nil
}

func decodeVendorProfileCommandData(data []byte) (vendorProfileCommandData, error) {
	var payload vendorProfileCommandData
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return vendorProfileCommandData{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return vendorProfileCommandData{}, errVendorProfileTransportInvalid
	}
	return payload, nil
}

func vendorProfileCommandFromTransport(request *generated.CommandRequest, params generated.OmdMaintainVendorProfilesParams, payload vendorProfileCommandData, correlationID uuid.UUID) (organization.VendorProfileCommand, error) {
	command := organization.VendorProfileCommand{
		Action: strings.ToLower(strings.TrimSpace(payload.Action)), ScopeID: uuid.Nil, PartyID: uuid.Nil,
		PaymentTerms: strings.ToLower(strings.TrimSpace(payload.PaymentTerms)), WithholdingTreatment: strings.ToLower(strings.TrimSpace(payload.WithholdingTreatment)), RemittancePreference: strings.ToLower(strings.TrimSpace(payload.RemittancePreference)),
		IdempotencyKey: params.IdempotencyKey, CorrelationID: correlationID.String(), CausationID: uuid.UUID(request.CommandId).String(), Approval: payload.Approval,
	}
	if !request.AccountingScopeId.Set || uuid.UUID(request.AccountingScopeId.Value) == uuid.Nil {
		return command, errVendorProfileTransportInvalid
	}
	command.ScopeID = uuid.UUID(request.AccountingScopeId.Value)
	if payload.ScopeID != "" {
		payloadScope, err := uuid.Parse(strings.TrimSpace(payload.ScopeID))
		if err != nil {
			return command, errVendorProfileTransportInvalid
		}
		if payloadScope != command.ScopeID {
			return command, errVendorProfileTransportConflict
		}
	}
	if payload.PartyID == "" {
		return command, errVendorProfileTransportInvalid
	}
	partyID, err := uuid.Parse(strings.TrimSpace(payload.PartyID))
	if err != nil {
		return command, errVendorProfileTransportInvalid
	}
	command.PartyID = partyID
	if payload.VendorProfileID != "" {
		profileID, err := uuid.Parse(strings.TrimSpace(payload.VendorProfileID))
		if err != nil {
			return command, errVendorProfileTransportInvalid
		}
		command.VendorProfileID = profileID
	}
	if payload.PartyVersion == nil || *payload.PartyVersion < 1 {
		return command, errVendorProfileTransportInvalid
	}
	partyVersion, err := aggregateversion.FromInt64(*payload.PartyVersion)
	if err != nil {
		return command, errVendorProfileTransportInvalid
	}
	command.ExpectedPartyVersion = &partyVersion
	command.EffectiveFrom, err = parseVendorProfileDate(payload.EffectiveFrom)
	if err != nil {
		return command, errVendorProfileTransportInvalid
	}
	if payload.EffectiveTo != nil && strings.TrimSpace(*payload.EffectiveTo) != "" {
		value, parseErr := parseVendorProfileDate(*payload.EffectiveTo)
		if parseErr != nil {
			return command, errVendorProfileTransportInvalid
		}
		command.EffectiveTo = &value
	}
	var bodyVersion *aggregateversion.AggregateVersion
	if request.ExpectedVersion.Set {
		version, err := aggregateversion.FromInt64(int64(request.ExpectedVersion.Value))
		if err != nil {
			return command, errVendorProfileTransportInvalid
		}
		bodyVersion = &version
	}
	if params.IfMatch.Set {
		headerVersion, err := parseIfMatchVersion(params.IfMatch.Value)
		if err != nil {
			return command, errVendorProfileTransportInvalid
		}
		if bodyVersion != nil && bodyVersion.Value() != headerVersion.Value() {
			return command, errVendorProfileTransportConflict
		}
		command.ExpectedVersion = &headerVersion
	} else {
		command.ExpectedVersion = bodyVersion
	}
	if command.Action == organization.VendorProfileActionCreate && command.ExpectedVersion != nil {
		return command, errVendorProfileTransportInvalid
	}
	if command.Action == organization.VendorProfileActionMaintain && !params.IfMatch.Set {
		return command, errVendorProfileTransportInvalid
	}
	if err := command.Validate(); err != nil {
		return command, errVendorProfileTransportInvalid
	}
	return command, nil
}

func parseVendorProfileDate(value string) (time.Time, error) {
	return time.Parse("2006-01-02", strings.TrimSpace(value))
}

func containsForbiddenVendorProfileKey(data []byte) bool {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return true
	}
	return forbiddenVendorProfileValue(value)
}

func forbiddenVendorProfileValue(value any) bool {
	switch typed := value.(type) {
	case map[string]any:
		for key, child := range typed {
			normalized := strings.NewReplacer("_", "", "-", "").Replace(strings.ToLower(key))
			switch normalized {
			case "account", "accountnumber", "iban", "routingnumber", "swift", "credential", "credentials", "password", "token", "secret", "rawaccount", "rawbankdetails", "bankdetails", "bankaccount", "providercredential", "taxidentifier", "rawtaxidentifier":
				return true
			}
			if forbiddenVendorProfileValue(child) {
				return true
			}
		}
	case []any:
		for _, child := range typed {
			if forbiddenVendorProfileValue(child) {
				return true
			}
		}
	}
	return false
}

func establishedVendorProfileResult(result organization.VendorProfileCommandResult, correlationID uuid.UUID) *generated.EstablishedResult {
	data := generated.EstablishedResultData{
		"vendorProfile":     mustRaw(result.VendorProfile),
		"partyId":           mustRaw(result.VendorProfile.PartyID.String()),
		"partyVersion":      mustRaw(result.VendorProfile.PartyVersion.Value()),
		"status":            mustRaw(result.VendorProfile.Status),
		"aggregateVersion":  mustRaw(result.VendorProfile.Version.Value()),
		"validationOutcome": mustRaw(result.ValidationOutcome),
		"decisionReference": mustRaw(result.DecisionReference.String()),
		"policyReference":   mustRaw(result.PolicyReference),
	}
	return &generated.EstablishedResult{Status: "established", AggregateId: generated.UUID(result.VendorProfile.ID), AggregateVersion: int(result.VendorProfile.Version.Value()), CorrelationId: generated.UUID(correlationID), Links: generated.Links{Self: "/api/v1/master-data/configuration/maintain-vendor-profiles"}, Data: data}
}

func mapOmdVendorProfileError(err error, correlationID uuid.UUID) generated.OmdMaintainVendorProfilesRes {
	switch {
	case errors.Is(err, organization.ErrVendorProfileAuthorizationDenied), errors.Is(err, organization.ErrVendorProfileFieldAuthorizationDenied):
		return omdVendorProfileProblem(http.StatusForbidden, "AUTHORIZATION_DENIED", "The requested vendor-profile fields or scope are outside the administering actor authority.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileAuthorizationStale):
		return omdVendorProfileProblem(http.StatusConflict, "POLICY_STALE", "The authorization policy changed. Refresh and retry.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileVersionConflict), errors.Is(err, organization.ErrVendorProfilePartyVersionConflict):
		return omdVendorProfileProblem(http.StatusConflict, "VERSION_CONFLICT", "The vendor profile or authoritative Party changed after it was loaded. Refresh and retry with current versions.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileIdempotencyConflict):
		return omdVendorProfileProblem(http.StatusConflict, "IDEMPOTENCY_CONFLICT", "The idempotency key was already used for different command data.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileCommandInProgress):
		return omdVendorProfileProblem(http.StatusConflict, "COMMAND_IN_PROGRESS", "The command is still being finalized. Retry with the same idempotency key.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileNotFound):
		return omdVendorProfileProblem(http.StatusConflict, "PROFILE_NOT_FOUND", "The requested vendor profile does not exist.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileDuplicate):
		return omdVendorProfileProblem(http.StatusConflict, "DUPLICATE_PROFILE", "A vendor profile already exists for this Party and accounting scope.", correlationID)
	case errors.Is(err, organization.ErrVendorProfilePartyNotFound), errors.Is(err, organization.ErrVendorProfilePartyInvalid), errors.Is(err, organization.ErrVendorProfilePartyMismatch):
		return omdVendorProfileProblem(http.StatusUnprocessableEntity, "INVALID_PARTY_REFERENCE", "The referenced Party is missing, outside the accounting scope, or is not a vendor Party.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileAuthorizationUnavailable), errors.Is(err, organization.ErrVendorProfileFieldAuthorizationUnavailable), errors.Is(err, organization.ErrVendorProfileAuditUnavailable), errors.Is(err, organization.ErrVendorProfileDurableCommandFailed), errors.Is(err, organization.ErrInvalidVendorProfileService):
		return omdVendorProfileProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The vendor-profile operation could not be completed.", correlationID)
	case errors.Is(err, organization.ErrVendorProfileApprovalRequired), errors.Is(err, organization.ErrVendorProfileApprovalRejected), errors.Is(err, organization.ErrInvalidVendorProfile), errors.Is(err, organization.ErrInvalidVendorProfileCommand):
		return omdVendorProfileProblem(http.StatusUnprocessableEntity, "VALIDATION_FAILED", "The vendor-profile command violates an organization rule.", correlationID)
	default:
		return omdVendorProfileProblem(http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "The vendor-profile operation could not be completed.", correlationID)
	}
}

func omdVendorProfileProblem(status int, code, detail string, correlationID uuid.UUID) generated.OmdMaintainVendorProfilesRes {
	problem := generated.ProblemDetails{Type: "https://tally.local/problems/" + code, Title: http.StatusText(status), Status: status, Code: code, Detail: detail, CorrelationId: generated.UUID(correlationID)}
	switch status {
	case http.StatusBadRequest:
		value := generated.OmdMaintainVendorProfilesBadRequest(problem)
		return &value
	case http.StatusForbidden:
		value := generated.OmdMaintainVendorProfilesForbidden(problem)
		return &value
	case http.StatusConflict:
		value := generated.OmdMaintainVendorProfilesConflict(problem)
		return &value
	case http.StatusUnprocessableEntity:
		value := generated.OmdMaintainVendorProfilesUnprocessableEntity(problem)
		return &value
	default:
		value := generated.OmdMaintainVendorProfilesServiceUnavailable(problem)
		return &value
	}
}

func correlationFromOmdVendorProfilesParams(params generated.OmdMaintainVendorProfilesParams) uuid.UUID {
	if params.XCorrelationID.Set {
		return uuid.UUID(params.XCorrelationID.Value)
	}
	return uuid.New()
}

func beginVendorProfileTelemetry(instrumentation *telemetry.Instrumentation, ctx context.Context) (context.Context, func(string)) {
	if instrumentation == nil {
		return ctx, func(string) {}
	}
	ctx, span := instrumentation.StartSpan(ctx, "organization.omd_maintain_vendor_profiles", telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainVendorProfiles", DataClassification: "highly_restricted"})
	finished := false
	return ctx, func(result string) {
		if finished {
			return
		}
		finished = true
		instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{Module: "organization", Operation: "omdMaintainVendorProfiles", Result: result, DataClassification: "highly_restricted"})
		instrumentation.AddCommand(ctx, 1, "organization", "omdMaintainVendorProfiles", result)
		span.End()
	}
}
