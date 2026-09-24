package authentication

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEnvironmentMiddlewareFailsClosedWhenProviderIsUnconfigured(t *testing.T) {
	middleware := NewMiddlewareFromEnvironment(func(string) string { return "" }, nil)
	recorder := httptest.NewRecorder()
	middleware(http.NotFoundHandler()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/future", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}

func TestFixtureModeIsUnavailableOutsideLocalEnvironment(t *testing.T) {
	values := map[string]string{
		"APP_ENV":            "production",
		"TALLY_AUTH_MODE":    "fixture",
		"ENTRA_TENANT_ID":    "tenant",
		"ENTRA_API_AUDIENCE": "api://tally",
		"ENTRA_METADATA_URL": "https://issuer.example/.well-known/openid-configuration",
	}
	middleware := NewMiddlewareFromEnvironment(func(key string) string { return values[key] }, nil)
	recorder := httptest.NewRecorder()
	middleware(http.NotFoundHandler()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/future", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}

func TestFixtureModeRequiresExplicitLocalEnvironment(t *testing.T) {
	values := map[string]string{
		"TALLY_AUTH_MODE":    "fixture",
		"ENTRA_TENANT_ID":    "tenant",
		"ENTRA_API_AUDIENCE": "api://tally",
	}
	middleware := NewMiddlewareFromEnvironment(func(key string) string { return values[key] }, nil)
	recorder := httptest.NewRecorder()
	middleware(http.NotFoundHandler()).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/future", nil))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", recorder.Code)
	}
}
