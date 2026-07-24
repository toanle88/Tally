package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/toanle88/Tally/internal/platform/httpx"
)

func TestLiveness(t *testing.T) {
	t.Parallel()

	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	recorder := httptest.NewRecorder()

	httpx.Liveness(recorder, request)

	response := recorder.Result()
	defer response.Body.Close()

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d",
			http.StatusOK, response.StatusCode)
	}

	const expectedContentType = "application/json"
	if actual := response.Header.Get("Content-Type"); actual != expectedContentType {
		t.Fatalf("expected Content-Type %q, got %q",
			expectedContentType, actual)
	}

	var body map[string]any

	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}

	if actual, ok := body["status"]; !ok || actual != "ok" {
		t.Fatalf("expected body status to be 'ok', got %v", actual)
	}
}
