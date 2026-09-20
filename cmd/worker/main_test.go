package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"testing"

	"github.com/toanle88/Tally/internal/platform/telemetry"
)

func TestRunFailsBeforeDatabaseSetupWithoutConsumers(t *testing.T) {
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{
		Service: "test-worker",
		Writer:  io.Discard,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := run(context.Background(), logger); !errors.Is(err, ErrNoRegisteredConsumers) {
		t.Fatalf("run error = %v, want ErrNoRegisteredConsumers", err)
	}
}

func TestWorkerFailureLogUsesStableCodeWithoutRawError(t *testing.T) {
	var output bytes.Buffer
	logger, err := telemetry.NewLogger(telemetry.LoggerConfig{
		Service: "test-worker",
		Writer:  &output,
	})
	if err != nil {
		t.Fatal(err)
	}

	logWorkerStoppedWithError(logger, context.Background(), errors.New("secret provider credential"))

	var fields map[string]any
	if err := json.Unmarshal(output.Bytes(), &fields); err != nil {
		t.Fatal(err)
	}
	if fields["error_code"] != "worker_run_failed" {
		t.Fatalf("error_code = %v, want worker_run_failed", fields["error_code"])
	}
	if bytes.Contains(output.Bytes(), []byte("secret provider credential")) {
		t.Fatal("raw worker error appeared in structured log")
	}
}
