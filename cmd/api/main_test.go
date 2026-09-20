package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/toanle88/Tally/internal/platform/telemetry"
)

func TestAPIErrorLogUsesStableCodeWithoutRawError(t *testing.T) {
	var output bytes.Buffer
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{
		Service: "test-api",
		Writer:  &output,
	})
	if err != nil {
		t.Fatal(err)
	}

	logAPIStoppedWithError(logger, errors.New("secret provider credential"))

	var fields map[string]any
	if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["error_code"] != "api_stopped_with_error" {
		t.Fatalf("error_code = %v, want api_stopped_with_error", fields["error_code"])
	}
	if bytes.Contains(output.Bytes(), []byte("secret provider credential")) {
		t.Fatal("raw API error appeared in structured log")
	}
}
