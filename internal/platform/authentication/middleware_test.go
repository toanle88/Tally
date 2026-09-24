package authentication

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

func TestMiddlewareRequiresBearerTokenAndDoesNotLeakDiagnostics(t *testing.T) {
	privateKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC), &privateKey.PublicKey)
	var logs bytes.Buffer
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{Service: "tally-api", Writer: &logs})
	if err != nil {
		t.Fatal(err)
	}
	middleware := NewMiddleware(MiddlewareConfig{
		Validator: validator,
		Resolver:  mustFixtureResolver(t),
		Logger:    logger,
	})

	recorder := httptest.NewRecorder()
	middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("protected handler was called without credentials")
	})).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", recorder.Code)
	}
	if recorder.Header().Get("WWW-Authenticate") != "Bearer" {
		t.Fatalf("WWW-Authenticate = %q", recorder.Header().Get("WWW-Authenticate"))
	}
	if strings.Contains(recorder.Body.String(), "Authorization") || strings.Contains(logs.String(), "Authorization") {
		t.Fatalf("sensitive header name leaked in output: body=%q logs=%q", recorder.Body.String(), logs.String())
	}
}

func TestMiddlewareMapsValidSubjectToActorContext(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, now, &privateKey.PublicKey)
	userID := uuid.New()
	resolver, err := identity.NewFixtureResolver([]identity.FixtureIdentity{{
		Subject: identity.AuthenticationSubject{OID: "oid-fixture", TID: fixtureTenant, Sub: "subject-fixture"},
		UserID:  userID,
	}})
	if err != nil {
		t.Fatal(err)
	}
	raw := signedFixtureToken(t, privateKey, fixtureTokenOptions{Exp: now.Add(time.Minute)})

	called := false
	handler := NewMiddleware(MiddlewareConfig{Validator: validator, Resolver: resolver})(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		actor, ok := identity.ActorFromContext(request.Context())
		if !ok || actor.UserID != userID {
			t.Fatalf("actor = %#v, present = %v", actor, ok)
		}
		writer.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	handler.ServeHTTP(recorder, request)

	if !called || recorder.Code != http.StatusNoContent {
		t.Fatalf("called = %v, status = %d", called, recorder.Code)
	}
}

func TestMiddlewareDeniesUnknownActorWithoutRevealingMapping(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, now, &privateKey.PublicKey)
	raw := signedFixtureToken(t, privateKey, fixtureTokenOptions{OID: "unknown-oid", Exp: now.Add(time.Minute)})
	handler := NewMiddleware(MiddlewareConfig{Validator: validator, Resolver: mustFixtureResolver(t)})(http.NotFoundHandler())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", recorder.Code)
	}
	body := recorder.Body.String()
	if strings.Contains(body, "unknown-oid") || strings.Contains(body, "subject") || strings.Contains(body, "user") {
		t.Fatalf("mapping details leaked: %q", body)
	}
}

func TestMiddlewareReturnsDependencyUnavailableForResolverOutage(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, now, &privateKey.PublicKey)
	raw := signedFixtureToken(t, privateKey, fixtureTokenOptions{Exp: now.Add(time.Minute)})
	handler := NewMiddleware(MiddlewareConfig{Validator: validator, Resolver: testUnavailableResolver{}})(http.NotFoundHandler())
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	request.Header.Set("Authorization", "Bearer "+raw)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable || !strings.Contains(recorder.Body.String(), "DEPENDENCY_UNAVAILABLE") {
		t.Fatalf("status = %d, body = %q", recorder.Code, recorder.Body.String())
	}
}

func mustFixtureResolver(t *testing.T) *identity.FixtureResolver {
	t.Helper()
	resolver, err := identity.NewFixtureResolver([]identity.FixtureIdentity{{
		Subject: identity.AuthenticationSubject{OID: "oid-fixture", TID: fixtureTenant, Sub: "subject-fixture"},
		UserID:  uuid.New(),
	}})
	if err != nil {
		t.Fatal(err)
	}
	return resolver
}

type testUnavailableResolver struct{}

func (testUnavailableResolver) ResolveApplicationActor(context.Context, identity.AuthenticationSubject) (identity.ApplicationActor, error) {
	return identity.ApplicationActor{}, identity.ErrActorResolverUnavailable
}
