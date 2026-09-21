package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"log/slog"
)

const (
	defaultDataClassification = "internal"
	maxStableValueLength      = 128
	maxAggregateTypeLength    = 64
)

var (
	ErrInvalidLoggerConfig = errors.New("invalid telemetry logger configuration")
	ErrInvalidAggregateRef = errors.New("invalid aggregate reference")
)

// LoggerConfig configures the technical JSON logger. The writer is normally
// os.Stderr at a composition root; tests may provide an in-memory writer.
type LoggerConfig struct {
	Service string
	Writer  io.Writer
	Now     func() time.Time
}

// AggregateReference is the bounded diagnostic identity of an aggregate. It
// never contains an aggregate payload.
type AggregateReference struct {
	Type    string
	ID      uuid.UUID
	Version int64
}

// NewAggregateReference validates a diagnostic aggregate reference.
func NewAggregateReference(aggregateType string, aggregateID uuid.UUID, version int64) (AggregateReference, error) {
	if !validStableValue(aggregateType, maxAggregateTypeLength) ||
		aggregateID == uuid.Nil || version < 1 {
		return AggregateReference{}, ErrInvalidAggregateRef
	}
	return AggregateReference{Type: aggregateType, ID: aggregateID, Version: version}, nil
}

// Event contains only the stable, redacted fields permitted in an operational
// log record. Message is a stable code, not a free-form error or payload.
type Event struct {
	Message            string
	Module             string
	Operation          string
	Result             string
	ErrorCode          string
	Retryable          bool
	DataClassification string

	ActorID           *uuid.UUID
	AccountingScopeID *uuid.UUID
	Aggregate         *AggregateReference
}

// Logger emits technical JSON logs. Logging errors are intentionally not
// returned by Emit: diagnostic output must never change an authoritative
// operation's result.
type Logger struct {
	handler *jsonHandler
}

// NewLogger creates a JSON logger with a fixed service field and a strict
// allow-list for all fields written to the output.
func NewLogger(config LoggerConfig) (*Logger, error) {
	if !validStableValue(config.Service, maxStableValueLength) || config.Writer == nil {
		return nil, ErrInvalidLoggerConfig
	}
	if config.Now == nil {
		config.Now = time.Now
	}
	return &Logger{handler: &jsonHandler{
		mu:      &sync.Mutex{},
		service: config.Service,
		writer:  config.Writer,
		now:     config.Now,
	}}, nil
}

// Emit writes one structured log record. Context identifiers are copied from
// the existing technical context when available.
func (l *Logger) Emit(ctx context.Context, level slog.Level, event Event) {
	if l == nil || l.handler == nil {
		return
	}
	record := slog.NewRecord(l.handler.now().UTC(), level, safeMessage(event.Message), 0)
	record.AddAttrs(
		slog.String("module", stableOrEmpty(event.Module)),
		slog.String("operation", stableOrEmpty(event.Operation)),
		slog.String("result", stableOrEmpty(event.Result)),
		slog.String("error_code", stableOrEmpty(event.ErrorCode)),
		slog.Bool("retryable", event.Retryable),
		slog.String("data_classification", validClassification(event.DataClassification)),
	)
	if event.ActorID != nil && *event.ActorID != uuid.Nil {
		record.AddAttrs(slog.String("actor_id", event.ActorID.String()))
	}
	if event.AccountingScopeID != nil && *event.AccountingScopeID != uuid.Nil {
		record.AddAttrs(slog.String("accounting_scope_id", event.AccountingScopeID.String()))
	}
	if event.Aggregate != nil {
		if reference, err := NewAggregateReference(event.Aggregate.Type, event.Aggregate.ID, event.Aggregate.Version); err == nil {
			record.AddAttrs(
				slog.String("aggregate_type", reference.Type),
				slog.String("aggregate_id", reference.ID.String()),
				slog.Int64("aggregate_version", reference.Version),
			)
		}
	}
	_ = l.handler.Handle(ctx, record)
}

// RequestLoggingMiddleware emits one safe completion record per HTTP request.
// It intentionally excludes the URL, headers, request body and response body.
func RequestLoggingMiddleware(logger *Logger) func(http.Handler) http.Handler {
	if logger == nil {
		panic("telemetry: nil logger")
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			panic("telemetry: nil HTTP handler")
		}
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			observed := &statusWriter{ResponseWriter: writer, status: http.StatusOK}
			next.ServeHTTP(observed, request)

			level := slog.LevelInfo
			result := "success"
			errorCode := ""
			if observed.status >= http.StatusInternalServerError {
				level = slog.LevelError
				result = "failure"
				errorCode = "http_server_error"
			} else if observed.status >= http.StatusBadRequest {
				level = slog.LevelWarn
				result = "rejected"
				errorCode = "http_client_error"
			}
			logger.Emit(request.Context(), level, Event{
				Message:   "http_request_completed",
				Module:    "platform.http",
				Operation: "http_request",
				Result:    result,
				ErrorCode: errorCode,
			})
		})
	}
}

type statusWriter struct {
	http.ResponseWriter
	status      int
	wroteHeader bool
}

func (w *statusWriter) WriteHeader(status int) {
	if w.wroteHeader {
		return
	}
	w.status = status
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(status)
}

func (w *statusWriter) Write(body []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}
	return w.ResponseWriter.Write(body)
}

// Unwrap preserves compatibility with http.ResponseController and wrapped
// response-writer capabilities.
func (w *statusWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }

type jsonHandler struct {
	mu      *sync.Mutex
	service string
	writer  io.Writer
	now     func() time.Time
	attrs   []slog.Attr
	grouped bool
}

func (h *jsonHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *jsonHandler) Handle(ctx context.Context, record slog.Record) error {
	fields := map[string]any{
		"timestamp":           timestamp(record.Time, h.now),
		"level":               record.Level.String(),
		"message":             safeMessage(record.Message),
		"service":             h.service,
		"module":              "",
		"operation":           "",
		"result":              "",
		"error_code":          "",
		"retryable":           false,
		"data_classification": defaultDataClassification,
	}

	if !h.grouped {
		for _, attr := range h.attrs {
			addAllowedField(fields, attr)
		}
		record.Attrs(func(attr slog.Attr) bool {
			addAllowedField(fields, attr)
			return true
		})
	}
	addContextFields(fields, ctx)

	h.mu.Lock()
	defer h.mu.Unlock()
	encoder := json.NewEncoder(h.writer)
	return encoder.Encode(fields)
}

func (h *jsonHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	clone := *h
	clone.attrs = append(append([]slog.Attr(nil), h.attrs...), attrs...)
	return &clone
}

func (h *jsonHandler) WithGroup(name string) slog.Handler {
	clone := *h
	if strings.TrimSpace(name) != "" {
		clone.grouped = true
	}
	return &clone
}

func addContextFields(fields map[string]any, ctx context.Context) {
	value, ok := FromContext(ctx)
	if !ok {
		return
	}
	if value.TraceID.IsValid() {
		fields["trace_id"] = value.TraceID.String()
	}
	if value.SpanID.IsValid() {
		fields["span_id"] = value.SpanID.String()
	}
	if value.CorrelationID != uuid.Nil {
		fields["correlation_id"] = value.CorrelationID.String()
	}
	if value.CausationID != uuid.Nil {
		fields["causation_id"] = value.CausationID.String()
	}
}

func addAllowedField(fields map[string]any, attr slog.Attr) {
	if attr.Key == "" {
		return
	}
	attr.Value = attr.Value.Resolve()
	switch attr.Key {
	case "module", "operation", "result", "error_code":
		if attr.Value.Kind() == slog.KindString && validStableValue(attr.Value.String(), maxStableValueLength) {
			value := attr.Value.String()
			fields[attr.Key] = value
		}
	case "retryable":
		if attr.Value.Kind() == slog.KindBool {
			fields[attr.Key] = attr.Value.Bool()
		}
	case "data_classification":
		if attr.Value.Kind() == slog.KindString {
			fields[attr.Key] = validClassification(attr.Value.String())
		}
	case "actor_id", "accounting_scope_id", "aggregate_id", "trace_id", "span_id", "correlation_id", "causation_id":
		if attr.Value.Kind() == slog.KindString {
			if parsed, err := uuid.Parse(attr.Value.String()); err == nil && parsed != uuid.Nil {
				fields[attr.Key] = parsed.String()
			}
		}
	case "aggregate_type":
		if attr.Value.Kind() == slog.KindString && validStableValue(attr.Value.String(), maxAggregateTypeLength) {
			value := attr.Value.String()
			fields[attr.Key] = value
		}
	case "aggregate_version":
		if attr.Value.Kind() == slog.KindInt64 && attr.Value.Int64() >= 1 {
			fields[attr.Key] = attr.Value.Int64()
		}
	}
}

func timestamp(value time.Time, now func() time.Time) string {
	if value.IsZero() {
		value = now()
	}
	return value.UTC().Format(time.RFC3339)
}

func safeMessage(value string) string {
	if !validStableValue(value, maxStableValueLength) || containsSensitiveWord(value) {
		return "telemetry_message_invalid"
	}
	return value
}

func stableOrEmpty(value string) string {
	if validStableValue(value, maxStableValueLength) && !containsSensitiveWord(value) {
		return value
	}
	return ""
}

func validStableValue(value string, maxLength int) bool {
	if value == "" || len(value) > maxLength || strings.TrimSpace(value) != value {
		return false
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') &&
			character != '_' && character != '-' && character != '.' {
			return false
		}
	}
	return true
}

func containsSensitiveWord(value string) bool {
	value = strings.ToLower(value)
	for _, word := range []string{"token", "credential", "password", "secret", "bank", "payroll", "tax", "verification"} {
		if strings.Contains(value, word) {
			return true
		}
	}
	return false
}

func validClassification(value string) string {
	switch value {
	case "public", "internal", "confidential", "highly_restricted":
		return value
	default:
		return defaultDataClassification
	}
}

var _ slog.Handler = (*jsonHandler)(nil)
