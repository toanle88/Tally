package authentication

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

// NewMiddlewareFromEnvironment composes the authentication boundary without
// making provider configuration part of domain code. Missing configuration is
// deliberately a request-time dependency failure so anonymous health checks
// remain available while protected API routes fail closed.
func NewMiddlewareFromEnvironment(getenv func(string) string, logger *telemetry.Logger) func(http.Handler) http.Handler {
	if getenv == nil {
		getenv = os.Getenv
	}
	appEnvironment := strings.ToLower(strings.TrimSpace(getenv("APP_ENV")))
	if appEnvironment == "" {
		appEnvironment = "production"
	}
	authMode := strings.ToLower(strings.TrimSpace(getenv("TALLY_AUTH_MODE")))
	if authMode == "fixture" && appEnvironment != "local" {
		return UnavailableMiddleware(logger)
	}

	tenantID := strings.TrimSpace(getenv("ENTRA_TENANT_ID"))
	audience := strings.TrimSpace(getenv("ENTRA_API_AUDIENCE"))
	if tenantID == "" || audience == "" {
		return UnavailableMiddleware(logger)
	}
	metadataURL := strings.TrimSpace(getenv("ENTRA_METADATA_URL"))
	if metadataURL == "" {
		return UnavailableMiddleware(logger)
	}
	keys, err := NewOIDCKeySource(OIDCDiscoveryConfig{URL: metadataURL})
	if err != nil {
		return UnavailableMiddleware(logger)
	}
	validator, err := NewValidator(Config{
		IssuerSource: keys,
		TenantID:     tenantID,
		Audience:     audience,
	}, keys)
	if err != nil {
		return UnavailableMiddleware(logger)
	}

	var resolver identity.UserResolver = unavailableResolver{}
	if authMode == "fixture" {
		fixtureResolver, err := newEnvironmentFixtureResolver(getenv, tenantID)
		if err != nil {
			return UnavailableMiddleware(logger)
		}
		resolver = fixtureResolver
	}
	return NewMiddleware(MiddlewareConfig{Validator: validator, Resolver: resolver, Logger: logger})
}

// UnavailableMiddleware is used when the authentication provider or the
// application actor directory is not configured. It never passes a protected
// request through accidentally.
func UnavailableMiddleware(logger *telemetry.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("authentication: nil HTTP handler")
		}
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			writeProblem(writer, request.Context(), http.StatusServiceUnavailable, "DEPENDENCY_UNAVAILABLE", "Authentication is temporarily unavailable.")
			logFailure(logger, request.Context(), "authentication_dependency_unavailable", true)
		})
	}
}

func newEnvironmentFixtureResolver(getenv func(string) string, tenantID string) (*identity.FixtureResolver, error) {
	oid := strings.TrimSpace(getenv("AUTH_FIXTURE_OID"))
	if oid == "" {
		oid = "local-fixture-oid"
	}
	sub := strings.TrimSpace(getenv("AUTH_FIXTURE_SUB"))
	if sub == "" {
		sub = "local-fixture-subject"
	}
	fixtureTenant := strings.TrimSpace(getenv("AUTH_FIXTURE_TID"))
	if fixtureTenant == "" {
		fixtureTenant = tenantID
	}
	userID := strings.TrimSpace(getenv("AUTH_FIXTURE_USER_ID"))
	if userID == "" {
		userID = "00000000-0000-0000-0000-000000000001"
	}
	parsedUserID, err := uuid.Parse(userID)
	if err != nil {
		return nil, err
	}
	return identity.NewFixtureResolver([]identity.FixtureIdentity{{
		Subject: identity.AuthenticationSubject{OID: oid, TID: fixtureTenant, Sub: sub},
		UserID:  parsedUserID,
	}})
}

type unavailableResolver struct{}

func (unavailableResolver) ResolveApplicationActor(context.Context, identity.AuthenticationSubject) (identity.ApplicationActor, error) {
	return identity.ApplicationActor{}, identity.ErrActorResolverUnavailable
}
