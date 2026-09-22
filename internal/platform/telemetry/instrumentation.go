package telemetry

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

const (
	metricHTTPDuration      = "finance_http_request_duration_seconds"
	metricCommandTotal      = "finance_command_total"
	metricDBDuration        = "finance_db_transaction_duration_seconds"
	metricOutboxPending     = "finance_outbox_pending_total"
	metricOutboxOldestAge   = "finance_outbox_oldest_age_seconds"
	metricInboxFailureTotal = "finance_inbox_failure_total"
	instrumentationName     = "github.com/toanle88/Tally/internal/platform/telemetry"
	maxLabelValues          = 64
	maxLabelLength          = 128
	maxInt64Value           = int64(1<<63 - 1)
)

var ErrInvalidInstrumentationConfig = errors.New("invalid telemetry instrumentation configuration")

// InstrumentationConfig supplies providers to the technical instrumentation
// layer. When either provider is nil, the OpenTelemetry process default is
// used. The repository intentionally does not configure a remote exporter here;
// composition roots can install one in a later operations delivery item.
type InstrumentationConfig struct {
	Service        string
	TracerProvider trace.TracerProvider
	MeterProvider  metric.MeterProvider
}

// Instrumentation owns the allow-listed spans and bounded platform metrics.
// Callers cannot record arbitrary attributes through this type; every public
// recording method applies a finite label registry and rejects sensitive text.
type Instrumentation struct {
	service string
	tracer  trace.Tracer

	shutdowners  []providerShutdowner
	shutdownOnce sync.Once
	shutdownErr  error

	httpRequestDuration   metric.Float64Histogram
	commandTotal          metric.Int64Counter
	dbTransactionDuration metric.Float64Histogram
	outboxPending         metric.Int64Gauge
	outboxOldestAge       metric.Float64Gauge
	inboxFailureTotal     metric.Int64Counter
	moduleLabels          *labelRegistry
	operationLabels       *labelRegistry
	resultLabels          *labelRegistry
	errorCodeLabels       *labelRegistry
	routeLabels           *labelRegistry
	eventTypeLabels       *labelRegistry
	consumerLabels        *labelRegistry
	failureClassLabels    *labelRegistry
}

// providerShutdowner is implemented by OpenTelemetry SDK providers. The API
// provider interfaces intentionally do not expose lifecycle methods, so
// Instrumentation only owns shutdown for providers that explicitly support it.
type providerShutdowner interface {
	Shutdown(context.Context) error
}

// SpanAttributes is the complete safe attribute vocabulary for technical
// spans. It deliberately has no fields for payloads, IDs, SQL, URLs, or raw
// errors.
type SpanAttributes struct {
	Module             string
	Operation          string
	Result             string
	ErrorCode          string
	Retryable          bool
	Route              string
	Method             string
	StatusClass        string
	EventType          string
	Consumer           string
	FailureClass       string
	DataClassification string
}

// OutboxBacklogPoint is a read-only platform backlog observation. It contains
// no message identity or payload and is intentionally separate from database
// query models.
type OutboxBacklogPoint struct {
	EventType string
	Pending   int64
	OldestAge time.Duration
}

// NewInstrumentation creates the platform instrumentation contract. OTel
// provider calls and metric recording are designed to be non-authoritative:
// provider implementations may be no-op and recording methods never return
// errors to business callers.
func NewInstrumentation(config InstrumentationConfig) (*Instrumentation, error) {
	if !validStableValue(config.Service, maxStableValueLength) {
		return nil, ErrInvalidInstrumentationConfig
	}
	if config.TracerProvider == nil {
		config.TracerProvider = otel.GetTracerProvider()
	}
	if config.MeterProvider == nil {
		config.MeterProvider = otel.GetMeterProvider()
	}

	meter := config.MeterProvider.Meter(instrumentationName)
	httpRequestDuration, err := meter.Float64Histogram(
		metricHTTPDuration,
		metric.WithDescription("HTTP request duration by bounded route, method, and status class"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	commandTotal, err := meter.Int64Counter(
		metricCommandTotal,
		metric.WithDescription("Technical command outcomes by bounded module, operation, and result"),
		metric.WithUnit("{command}"),
	)
	if err != nil {
		return nil, err
	}
	dbTransactionDuration, err := meter.Float64Histogram(
		metricDBDuration,
		metric.WithDescription("PostgreSQL transaction duration by bounded module, operation, and result"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	outboxPending, err := meter.Int64Gauge(
		metricOutboxPending,
		metric.WithDescription("Pending outbox items by bounded event type"),
		metric.WithUnit("{item}"),
	)
	if err != nil {
		return nil, err
	}
	outboxOldestAge, err := meter.Float64Gauge(
		metricOutboxOldestAge,
		metric.WithDescription("Age of the oldest pending outbox item by bounded event type"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}
	inboxFailureTotal, err := meter.Int64Counter(
		metricInboxFailureTotal,
		metric.WithDescription("Inbox failure outcomes by bounded consumer and error code"),
		metric.WithUnit("{failure}"),
	)
	if err != nil {
		return nil, err
	}

	shutdowners := make([]providerShutdowner, 0, 2)
	if shutdowner, ok := config.TracerProvider.(providerShutdowner); ok {
		shutdowners = append(shutdowners, shutdowner)
	}
	if shutdowner, ok := config.MeterProvider.(providerShutdowner); ok {
		shutdowners = append(shutdowners, shutdowner)
	}

	return &Instrumentation{
		service:               config.Service,
		tracer:                config.TracerProvider.Tracer(instrumentationName),
		shutdowners:           shutdowners,
		httpRequestDuration:   httpRequestDuration,
		commandTotal:          commandTotal,
		dbTransactionDuration: dbTransactionDuration,
		outboxPending:         outboxPending,
		outboxOldestAge:       outboxOldestAge,
		inboxFailureTotal:     inboxFailureTotal,
		moduleLabels:          newLabelRegistry(maxLabelValues),
		operationLabels:       newLabelRegistry(maxLabelValues),
		resultLabels:          newLabelRegistry(maxLabelValues),
		errorCodeLabels:       newLabelRegistry(maxLabelValues),
		routeLabels:           newLabelRegistry(maxLabelValues),
		eventTypeLabels:       newLabelRegistry(maxLabelValues),
		consumerLabels:        newLabelRegistry(maxLabelValues),
		failureClassLabels:    newLabelRegistry(maxLabelValues),
	}, nil
}

// Shutdown flushes and shuts down injected SDK providers at most once. The
// provider implementations own exporter behavior and receive the caller's
// deadline. A shutdown error is diagnostic only; recording methods never
// return telemetry errors to an authoritative operation.
func (i *Instrumentation) Shutdown(ctx context.Context) error {
	if i == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	i.shutdownOnce.Do(func() {
		shutdownErrors := make([]error, 0, len(i.shutdowners))
		for _, shutdowner := range i.shutdowners {
			if err := shutdowner.Shutdown(ctx); err != nil {
				shutdownErrors = append(shutdownErrors, err)
			}
		}
		i.shutdownErr = errors.Join(shutdownErrors...)
	})
	return i.shutdownErr
}

// StartSpan creates a child technical span and mirrors its current trace/span
// IDs into the repository telemetry context used by structured logs.
func (i *Instrumentation) StartSpan(ctx context.Context, name string, fields SpanAttributes) (context.Context, trace.Span) {
	if ctx == nil {
		ctx = context.Background()
	}
	if i == nil || i.tracer == nil {
		spanContext, span := trace.NewNoopTracerProvider().Tracer(instrumentationName).Start(ctx, safeSpanName(name))
		return spanContext, span
	}
	ctx, span := startSpan(ctx, i.tracer, safeSpanName(name))
	span.SetAttributes(i.spanAttributes(fields)...)
	return ctx, span
}

// SetSpanAttributes adds only the allow-listed technical fields to an
// existing span. It is useful when an outcome is known at the end of a
// bounded operation.
func (i *Instrumentation) SetSpanAttributes(span trace.Span, fields SpanAttributes) {
	if i == nil || span == nil {
		return
	}
	span.SetAttributes(i.spanAttributes(fields)...)
}

func (i *Instrumentation) spanAttributes(fields SpanAttributes) []attribute.KeyValue {
	if i == nil {
		return nil
	}
	attributes := make([]attribute.KeyValue, 0, 12)
	if value := i.moduleLabels.value(fields.Module, "other"); value != "" {
		attributes = append(attributes, attribute.String("module", value))
	}
	if value := i.operationLabels.value(fields.Operation, "other"); value != "" {
		attributes = append(attributes, attribute.String("operation", value))
	}
	if value := i.resultLabels.value(fields.Result, "other"); value != "" {
		attributes = append(attributes, attribute.String("result", value))
	}
	if value := i.errorCodeLabels.value(fields.ErrorCode, ""); value != "" {
		attributes = append(attributes, attribute.String("error_code", value))
	}
	if fields.Retryable {
		attributes = append(attributes, attribute.Bool("retryable", true))
	}
	if value := i.routeLabels.value(fields.Route, ""); value != "" {
		attributes = append(attributes, attribute.String("route", value))
	}
	if value := normalizeMethod(fields.Method); value != "" {
		attributes = append(attributes, attribute.String("method", value))
	}
	if value := normalizeStatusClass(fields.StatusClass); value != "" {
		attributes = append(attributes, attribute.String("status_class", value))
	}
	if value := i.eventTypeLabels.value(fields.EventType, ""); value != "" {
		attributes = append(attributes, attribute.String("event_type", value))
	}
	if value := i.consumerLabels.value(fields.Consumer, ""); value != "" {
		attributes = append(attributes, attribute.String("consumer", value))
	}
	if value := i.failureClassLabels.value(fields.FailureClass, ""); value != "" {
		attributes = append(attributes, attribute.String("failure_class", value))
	}
	if value := normalizeClassification(fields.DataClassification); value != "" {
		attributes = append(attributes, attribute.String("data_classification", value))
	}
	return attributes
}

func (i *Instrumentation) ObserveHTTPRequest(ctx context.Context, duration time.Duration, route, method, statusClass string) {
	if i == nil || i.httpRequestDuration == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	i.httpRequestDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("route", i.routeLabels.value(route, "unmatched")),
		attribute.String("method", normalizeMethodOrOther(method)),
		attribute.String("status_class", normalizeStatusClassOrOther(statusClass)),
	))
}

func (i *Instrumentation) AddCommand(ctx context.Context, amount int64, module, operation, result string) {
	if i == nil || i.commandTotal == nil || amount <= 0 {
		return
	}
	i.commandTotal.Add(ctx, amount, metric.WithAttributes(
		attribute.String("module", i.moduleLabels.value(module, "other")),
		attribute.String("operation", i.operationLabels.value(operation, "other")),
		attribute.String("result", i.resultLabels.value(result, "other")),
	))
}

func (i *Instrumentation) ObserveDBTransaction(ctx context.Context, duration time.Duration, module, operation, result string) {
	if i == nil || i.dbTransactionDuration == nil {
		return
	}
	if duration < 0 {
		duration = 0
	}
	i.dbTransactionDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("module", i.moduleLabels.value(module, "other")),
		attribute.String("operation", i.operationLabels.value(operation, "other")),
		attribute.String("result", i.resultLabels.value(result, "other")),
	))
}

func (i *Instrumentation) SetOutboxBacklog(ctx context.Context, eventType string, pending int64, oldestAge time.Duration) {
	if i == nil || i.outboxPending == nil || i.outboxOldestAge == nil {
		return
	}
	i.setOutboxBacklog(ctx, i.eventTypeLabels.value(eventType, "other"), pending, oldestAge)
}

// SetOutboxBacklogSnapshot records current values and clears previously seen
// bounded event-type series that are absent from the current snapshot.
func (i *Instrumentation) SetOutboxBacklogSnapshot(ctx context.Context, points []OutboxBacklogPoint) {
	if i == nil || i.outboxPending == nil || i.outboxOldestAge == nil {
		return
	}
	type backlogValue struct {
		pending   int64
		oldestAge time.Duration
	}
	values := make(map[string]backlogValue, len(points))
	for _, point := range points {
		label := i.eventTypeLabels.value(point.EventType, "other")
		value := values[label]
		if point.Pending > 0 {
			if value.pending > maxInt64Value-point.Pending {
				value.pending = maxInt64Value
			} else {
				value.pending += point.Pending
			}
		}
		if point.OldestAge > value.oldestAge {
			value.oldestAge = point.OldestAge
		}
		values[label] = value
	}
	for label, value := range values {
		i.setOutboxBacklog(ctx, label, value.pending, value.oldestAge)
	}
	for _, label := range i.eventTypeLabels.valuesSnapshot() {
		if _, ok := values[label]; !ok {
			i.setOutboxBacklog(ctx, label, 0, 0)
		}
	}
}

func (i *Instrumentation) setOutboxBacklog(ctx context.Context, label string, pending int64, oldestAge time.Duration) {
	if pending < 0 {
		pending = 0
	}
	if oldestAge < 0 {
		oldestAge = 0
	}
	attributes := metric.WithAttributes(attribute.String("event_type", label))
	i.outboxPending.Record(ctx, pending, attributes)
	i.outboxOldestAge.Record(ctx, oldestAge.Seconds(), attributes)
}

func (i *Instrumentation) AddInboxFailure(ctx context.Context, amount int64, consumer, errorCode string) {
	if i == nil || i.inboxFailureTotal == nil || amount <= 0 {
		return
	}
	i.inboxFailureTotal.Add(ctx, amount, metric.WithAttributes(
		attribute.String("consumer", i.consumerLabels.value(consumer, "other")),
		attribute.String("error_code", i.errorCodeLabels.value(errorCode, "other")),
	))
}

func safeSpanName(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > maxLabelLength || !utf8.ValidString(value) || containsSensitiveWord(value) {
		return "platform.unknown"
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') &&
			character != '_' && character != '-' && character != '.' {
			return "platform.unknown"
		}
	}
	return value
}

func normalizeLabel(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || len(value) > maxLabelLength || !utf8.ValidString(value) || containsSensitiveWord(value) {
		return ""
	}
	if _, err := uuid.Parse(value); err == nil {
		return ""
	}
	for _, character := range value {
		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') &&
			character != '_' && character != '-' && character != '.' &&
			character != '/' && character != '{' && character != '}' &&
			character != '*' {
			return ""
		}
	}
	return value
}

func normalizeMethod(value string) string {
	value = strings.ToUpper(strings.TrimSpace(value))
	switch value {
	case "GET", "HEAD", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "CONNECT", "TRACE":
		return value
	default:
		return ""
	}
}

func normalizeMethodOrOther(value string) string {
	if value = normalizeMethod(value); value != "" {
		return value
	}
	return "other"
}

func normalizeStatusClass(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1xx", "2xx", "3xx", "4xx", "5xx":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

func normalizeStatusClassOrOther(value string) string {
	if value = normalizeStatusClass(value); value != "" {
		return value
	}
	return "other"
}

func normalizeClassification(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "public", "internal", "confidential", "highly_restricted":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return ""
	}
}

type labelRegistry struct {
	mu     sync.Mutex
	values map[string]struct{}
	max    int
}

func (r *labelRegistry) valuesSnapshot() []string {
	if r == nil {
		return nil
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	values := make([]string, 0, len(r.values))
	for value := range r.values {
		values = append(values, value)
	}
	return values
}

func newLabelRegistry(max int) *labelRegistry {
	return &labelRegistry{values: make(map[string]struct{}, max), max: max}
}

func (r *labelRegistry) value(raw, fallback string) string {
	if r == nil {
		return fallback
	}
	value := normalizeLabel(raw)
	if value == "" {
		return fallback
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.values[value]; ok {
		return value
	}
	if len(r.values) >= r.max {
		return fallback
	}
	if len(r.values) == r.max-1 && fallback != "" {
		r.values[fallback] = struct{}{}
		return fallback
	}
	r.values[value] = struct{}{}
	return value
}
