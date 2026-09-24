package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toanle88/Tally/internal/platform/telemetry"
)

func TestNewRouterKeepsHealthAnonymousAndProtectsAPI(t *testing.T) {
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{Service: "tally-api", Writer: &bytes.Buffer{}})
	if err != nil {
		t.Fatal(err)
	}
	instrumentation, err := telemetry.NewInstrumentation(telemetry.InstrumentationConfig{Service: "tally-api"})
	if err != nil {
		t.Fatal(err)
	}
	defer instrumentation.Shutdown(t.Context())

	router := newRouter(instrumentation, logger, func(string) string { return "" })

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/health/live", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want 200", health.Code)
	}

	protected := httptest.NewRecorder()
	router.ServeHTTP(protected, httptest.NewRequest(http.MethodGet, "/api/v1/future-operation", nil))
	if protected.Code != http.StatusServiceUnavailable {
		t.Fatalf("protected status = %d, want 503", protected.Code)
	}
}
