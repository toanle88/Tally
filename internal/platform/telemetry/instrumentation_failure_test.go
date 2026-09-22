package telemetry

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var (
	errExporterFailed = errors.New("synthetic telemetry exporter failed")
	errShutdownFailed = errors.New("synthetic telemetry shutdown failed")
)

type controlledSpanExporter struct {
	exportStarted chan struct{}
	release       chan struct{}
	exportErr     error
	shutdownErr   error
	exportCalls   atomic.Int32
	shutdownCalls atomic.Int32
	startOnce     sync.Once
}

func (e *controlledSpanExporter) ExportSpans(ctx context.Context, _ []sdktrace.ReadOnlySpan) error {
	e.exportCalls.Add(1)
	if e.exportStarted != nil {
		e.startOnce.Do(func() { close(e.exportStarted) })
	}
	if e.release == nil {
		return e.exportErr
	}
	select {
	case <-e.release:
		return e.exportErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *controlledSpanExporter) Shutdown(context.Context) error {
	e.shutdownCalls.Add(1)
	return e.shutdownErr
}

type controlledMetricExporter struct {
	exportStarted chan struct{}
	release       chan struct{}
	exportErr     error
	shutdownErr   error
	exportCalls   atomic.Int32
	shutdownCalls atomic.Int32
	startOnce     sync.Once
}

func (e *controlledMetricExporter) Export(ctx context.Context, _ *metricdata.ResourceMetrics) error {
	e.exportCalls.Add(1)
	if e.exportStarted != nil {
		e.startOnce.Do(func() { close(e.exportStarted) })
	}
	if e.release == nil {
		return e.exportErr
	}
	select {
	case <-e.release:
		return e.exportErr
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (e *controlledMetricExporter) Temporality(sdkmetric.InstrumentKind) metricdata.Temporality {
	return metricdata.CumulativeTemporality
}

func (e *controlledMetricExporter) Aggregation(kind sdkmetric.InstrumentKind) sdkmetric.Aggregation {
	return sdkmetric.DefaultAggregationSelector(kind)
}

func (e *controlledMetricExporter) ForceFlush(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return e.exportErr
}

func (e *controlledMetricExporter) Shutdown(context.Context) error {
	e.shutdownCalls.Add(1)
	return e.shutdownErr
}

func newTestInstrumentation(t *testing.T, tracerProvider *sdktrace.TracerProvider, meterProvider *sdkmetric.MeterProvider) *Instrumentation {
	t.Helper()
	if tracerProvider == nil {
		tracerProvider = sdktrace.NewTracerProvider()
	}
	if meterProvider == nil {
		meterProvider = sdkmetric.NewMeterProvider()
	}
	instrumentation, err := NewInstrumentation(InstrumentationConfig{
		Service:        "tally-test",
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	})
	if err != nil {
		t.Fatal(err)
	}
	return instrumentation
}

func TestInstrumentationExporterFailureDoesNotChangeBusinessOutcomeOrRepeatOperation(t *testing.T) {
	exporter := &controlledSpanExporter{exportErr: errExporterFailed}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)))
	instrumentation := newTestInstrumentation(t, provider, nil)

	var authoritativeCalls atomic.Int32
	businessFailure := errors.New("synthetic business rejection")
	operation := func() error {
		authoritativeCalls.Add(1)
		_, span := instrumentation.StartSpan(context.Background(), "command.execute", SpanAttributes{
			Module:    "platform.synthetic",
			Operation: "synthetic_command",
		})
		span.End()
		return businessFailure
	}

	if err := operation(); !errors.Is(err, businessFailure) {
		t.Fatalf("business result = %v, want original business error", err)
	}
	if got := authoritativeCalls.Load(); got != 1 {
		t.Fatalf("authoritative operation calls = %d, want 1", got)
	}
	if got := exporter.exportCalls.Load(); got != 1 {
		t.Fatalf("export calls = %d, want 1", got)
	}
	if err := instrumentation.Shutdown(context.Background()); err != nil {
		t.Fatalf("shutdown error = %v, want exporter failure to remain non-authoritative", err)
	}
}

func TestInstrumentationShutdownReturnsExporterFailureOnce(t *testing.T) {
	exporter := &controlledSpanExporter{shutdownErr: errShutdownFailed}
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(sdktrace.NewSimpleSpanProcessor(exporter)))
	instrumentation := newTestInstrumentation(t, provider, nil)

	firstErr := instrumentation.Shutdown(context.Background())
	secondErr := instrumentation.Shutdown(context.Background())
	if !errors.Is(firstErr, errShutdownFailed) || !errors.Is(secondErr, errShutdownFailed) {
		t.Fatalf("shutdown errors = %v / %v, want %v", firstErr, secondErr, errShutdownFailed)
	}
	if got := exporter.shutdownCalls.Load(); got != 1 {
		t.Fatalf("exporter shutdown calls = %d, want 1", got)
	}
}

func TestInstrumentationShutdownBoundsBlockingExporter(t *testing.T) {
	exporter := &controlledSpanExporter{
		exportStarted: make(chan struct{}),
		release:       make(chan struct{}),
	}
	processor := sdktrace.NewBatchSpanProcessor(
		exporter,
		sdktrace.WithBatchTimeout(time.Hour),
		sdktrace.WithMaxExportBatchSize(1),
	)
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(processor))
	instrumentation := newTestInstrumentation(t, provider, nil)
	_, span := instrumentation.StartSpan(context.Background(), "command.execute", SpanAttributes{
		Module:    "platform.synthetic",
		Operation: "synthetic_command",
	})
	span.End()

	shutdownContext, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	shutdownResult := make(chan error, 1)
	go func() { shutdownResult <- instrumentation.Shutdown(shutdownContext) }()

	select {
	case <-exporter.exportStarted:
	case <-time.After(time.Second):
		t.Fatal("blocking exporter was not invoked")
	}

	select {
	case err := <-shutdownResult:
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("shutdown error = %v, want context deadline", err)
		}
	case <-time.After(time.Second):
		t.Fatal("telemetry shutdown exceeded its context deadline")
	}
	if got := exporter.exportCalls.Load(); got != 1 {
		t.Fatalf("export calls = %d, want one bounded attempt", got)
	}
}

func TestInstrumentationMetricExporterFailureIsDiagnosticOnly(t *testing.T) {
	exporter := &controlledMetricExporter{exportErr: errExporterFailed}
	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(time.Hour))
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	instrumentation := newTestInstrumentation(t, nil, provider)
	instrumentation.ObserveHTTPRequest(context.Background(), time.Millisecond, "/health/live", "GET", "2xx")

	if err := instrumentation.Shutdown(context.Background()); !errors.Is(err, errExporterFailed) {
		t.Fatalf("shutdown error = %v, want metric exporter failure", err)
	}
	if got := exporter.exportCalls.Load(); got != 1 {
		t.Fatalf("metric export calls = %d, want one bounded attempt", got)
	}
}

func TestInstrumentationMetricAttributesExcludeSensitiveValues(t *testing.T) {
	reader := sdkmetric.NewManualReader()
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader))
	instrumentation := newTestInstrumentation(t, nil, provider)
	defer func() { _ = instrumentation.Shutdown(context.Background()) }()

	secretPayload := "password=secret bank=bank-value tax=tax-value"
	instrumentation.ObserveHTTPRequest(context.Background(), time.Millisecond, "/accounts/"+secretPayload, "GET", "2xx")
	instrumentation.AddCommand(context.Background(), 1, "platform.synthetic", secretPayload, "success")
	instrumentation.AddInboxFailure(context.Background(), 1, secretPayload, secretPayload)

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &resourceMetrics); err != nil {
		t.Fatal(err)
	}
	serialized := strings.ToLower(string(mustJSONMarshal(t, resourceMetrics)))
	for _, forbidden := range []string{"password", "secret", "bank-value", "tax-value", "accounts/"} {
		if strings.Contains(serialized, forbidden) {
			t.Fatalf("forbidden telemetry value %q appeared in metrics: %s", forbidden, serialized)
		}
	}
}

func mustJSONMarshal(t *testing.T, value any) []byte {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
