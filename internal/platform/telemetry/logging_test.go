package telemetry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

func TestLoggerEmitsStableJSONFieldsAndReferences(t *testing.T) {
	var output bytes.Buffer
	when := time.Date(2026, time.September, 20, 15, 4, 5, 123000000, time.FixedZone("ICT", 7*60*60))
	logger, err := NewLogger(LoggerConfig{
		Service: "tally-api",
		Writer:  &output,
		Now:     func() time.Time { return when },
	})
	if err != nil {
		t.Fatal(err)
	}

	traceID, err := trace.TraceIDFromHex("0102030405060708090a0b0c0d0e0f10")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("0102030405060708")
	if err != nil {
		t.Fatal(err)
	}
	correlationID := uuid.New()
	causationID := uuid.New()
	ctx, err := With(context.Background(), TelemetryContext{
		TraceID:       traceID,
		SpanID:        spanID,
		CorrelationID: correlationID,
		CausationID:   causationID,
	})
	if err != nil {
		t.Fatal(err)
	}

	actorID := uuid.New()
	scopeID := uuid.New()
	aggregate, err := NewAggregateReference("journal_entry", uuid.New(), 7)
	if err != nil {
		t.Fatal(err)
	}
	logger.Emit(ctx, slog.LevelInfo, Event{
		Message:            "journal_posting_completed",
		Module:             "general_ledger",
		Operation:          "post_journal",
		Result:             "success",
		Retryable:          false,
		DataClassification: "internal",
		ActorID:            &actorID,
		AccountingScopeID:  &scopeID,
		Aggregate:          &aggregate,
	})

	fields := decodeLog(t, output.Bytes())
	if fields["timestamp"] != "2026-09-20T08:04:05Z" {
		t.Fatalf("timestamp = %v, want RFC3339 UTC timestamp", fields["timestamp"])
	}
	for key, want := range map[string]any{
		"level":               "INFO",
		"message":             "journal_posting_completed",
		"service":             "tally-api",
		"module":              "general_ledger",
		"operation":           "post_journal",
		"result":              "success",
		"error_code":          "",
		"retryable":           false,
		"data_classification": "internal",
		"trace_id":            traceID.String(),
		"span_id":             spanID.String(),
		"correlation_id":      correlationID.String(),
		"causation_id":        causationID.String(),
		"actor_id":            actorID.String(),
		"accounting_scope_id": scopeID.String(),
		"aggregate_type":      "journal_entry",
		"aggregate_id":        aggregate.ID.String(),
		"aggregate_version":   float64(7),
	} {
		if fields[key] != want {
			t.Errorf("%s = %v, want %v", key, fields[key], want)
		}
	}
}

func TestLoggerDefaultsClassificationAndRejectsUnsafeMessage(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(LoggerConfig{Service: "tally-worker", Writer: &output})
	if err != nil {
		t.Fatal(err)
	}

	logger.Emit(context.Background(), slog.LevelError, Event{
		Message:            "password=do-not-log",
		Module:             "platform.worker",
		Operation:          "run",
		Result:             "failure",
		ErrorCode:          "worker_run_failed",
		DataClassification: "not-a-classification",
	})

	fields := decodeLog(t, output.Bytes())
	if fields["message"] != "telemetry_message_invalid" {
		t.Fatalf("message = %v, want safe fallback", fields["message"])
	}
	if fields["data_classification"] != string(defaultDataClassification) {
		t.Fatalf("classification = %v, want %q", fields["data_classification"], defaultDataClassification)
	}
	if strings.Contains(output.String(), "do-not-log") {
		t.Fatal("unsafe message value was emitted")
	}
}

func TestJSONHandlerDropsUnknownAndSensitiveAttributes(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(LoggerConfig{Service: "tally-api", Writer: &output})
	if err != nil {
		t.Fatal(err)
	}

	record := slog.NewRecord(time.Date(2026, time.September, 20, 0, 0, 0, 0, time.UTC), slog.LevelInfo, "safe_event", 0)
	record.AddAttrs(
		slog.String("module", "platform.http"),
		slog.String("operation", "http_request"),
		slog.String("result", "success"),
		slog.String("password", "password-value"),
		slog.String("token", "token-value"),
		slog.String("bank_details", "bank-value"),
		slog.String("payroll", "payroll-value"),
		slog.String("tax_identifier", "tax-value"),
		slog.String("verification_credential", "verification-value"),
		slog.String("sql", "SELECT secret FROM accounts"),
		slog.String("request_body", `{"password":"password-value"}`),
		slog.String("response_body", `{"token":"token-value"}`),
		slog.String("event_body", `{"bank":"bank-value"}`),
		slog.String("error", "raw internal error with secret-value"),
		slog.Group("nested", slog.String("secret", "nested-secret-value")),
		slog.String("unknown", "unknown-value"),
	)
	if err := logger.handler.Handle(context.Background(), record); err != nil {
		t.Fatal(err)
	}

	fields := decodeLog(t, output.Bytes())
	for _, forbidden := range []string{
		"password-value",
		"token-value",
		"bank-value",
		"payroll-value",
		"tax-value",
		"verification-value",
		"SELECT secret FROM accounts",
		"password-value",
		"nested-secret-value",
		"unknown-value",
		"raw internal error",
	} {
		if strings.Contains(output.String(), forbidden) {
			t.Fatalf("forbidden value %q appeared in log: %s", forbidden, output.String())
		}
	}
	for _, forbiddenKey := range []string{
		"password",
		"token",
		"bank_details",
		"payroll",
		"tax_identifier",
		"verification_credential",
		"sql",
		"request_body",
		"response_body",
		"event_body",
		"error",
		"nested",
		"unknown",
	} {
		if _, present := fields[forbiddenKey]; present {
			t.Errorf("forbidden key %q was emitted", forbiddenKey)
		}
	}
}

func TestRequestLoggingMiddlewarePreservesCorrelationAndOmitsPayloads(t *testing.T) {
	var output bytes.Buffer
	logger, err := NewLogger(LoggerConfig{Service: "tally-api", Writer: &output})
	if err != nil {
		t.Fatal(err)
	}

	correlationID := uuid.New()
	handler := Middleware(RequestLoggingMiddleware(logger)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusCreated)
		if _, err := io.WriteString(writer, `{"password":"do-not-log"}`); err != nil {
			t.Fatal(err)
		}
	})))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/accounts?token=do-not-log", nil)
	request.Header.Set(CorrelationHeader, correlationID.String())
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusCreated)
	}
	fields := decodeLog(t, output.Bytes())
	if fields["correlation_id"] != correlationID.String() {
		t.Fatalf("correlation_id = %v, want %s", fields["correlation_id"], correlationID)
	}
	if fields["operation"] != "http_request" || fields["result"] != "success" {
		t.Fatalf("request fields = %v, want safe success record", fields)
	}
	if strings.Contains(output.String(), "do-not-log") {
		t.Fatal("request or response payload appeared in log")
	}
}

func TestNewLoggerAndAggregateReferenceValidateConfiguration(t *testing.T) {
	if _, err := NewLogger(LoggerConfig{Service: "", Writer: io.Discard}); err == nil {
		t.Fatal("empty service was accepted")
	}
	if _, err := NewLogger(LoggerConfig{Service: "tally-api"}); err == nil {
		t.Fatal("nil writer was accepted")
	}
	if _, err := NewAggregateReference("journal_entry", uuid.Nil, 1); err == nil {
		t.Fatal("nil aggregate ID was accepted")
	}
	if _, err := NewAggregateReference("journal_entry", uuid.New(), 0); err == nil {
		t.Fatal("non-positive aggregate version was accepted")
	}
}

func decodeLog(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var fields map[string]any
	if err := json.Unmarshal(data, &fields); err != nil {
		t.Fatalf("decode log JSON: %v; output=%s", err, data)
	}
	return fields
}
