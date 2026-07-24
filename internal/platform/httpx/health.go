package httpx

import (
	"encoding/json"
	"net/http"
)

// LivenessResponse represents the response structure for the liveness endpoint.
type LivenessResponse struct {
	Status string `json:"status"`
}

// Liveness handles GET /health/live
//
// The router is responsible for restricting this handler to GET requests.
func Liveness(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	_ = json.NewEncoder(w).Encode(LivenessResponse{Status: "ok"})
}
