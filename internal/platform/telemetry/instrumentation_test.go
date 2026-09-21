package telemetry

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func TestInstrumentationUsesAllowListedSpansAndMirrorsTraceContext(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()

	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	defer func() { _ = meterProvider.Shutdown(context.Background()) }()

	instrumentation, err := NewInstrumentation(InstrumentationConfig{
		Service:        "tally-api",
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	})
	if err != nil {
		t.Fatal(err)
	}

	rawIdentifier := uuid.New().String()
	rootContext, err := With(context.Background(), NewRoot(uuid.New()))
	if err != nil {
		t.Fatal(err)
	}
	ctx, span := instrumentation.StartSpan(rootContext, "safe.operation", SpanAttributes{
		Module:    "platform.http",
		Operation: "http_request",
		EventType: rawIdentifier,
		Route:     "/accounts/:id?token=secret",
		Method:    "GET",
	})
	instrumentation.SetSpanAttributes(span, SpanAttributes{
		Result:    "success",
		ErrorCode: "sql_select_password",
	})
	span.End()

	value, ok := FromContext(ctx)
	if !ok || !value.TraceID.IsValid() || !value.SpanID.IsValid() {
		t.Fatalf("created span was not mirrored into telemetry context: %#v, present=%v", value, ok)
	}

	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	for _, attribute := range ended[0].Attributes() {
		if attribute.Key == "error_code" || strings.Contains(attribute.Value.AsString(), rawIdentifier) || strings.Contains(strings.ToLower(attribute.Value.AsString()), "password") || strings.Contains(strings.ToLower(attribute.Value.AsString()), "token") {
			t.Fatalf("sensitive span attribute was recorded: %s=%s", attribute.Key, attribute.Value.AsString())
		}
	}
}

func TestInstrumentationPublishesOnlyBoundedPlatformMetrics(t *testing.T) {
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	defer func() { _ = meterProvider.Shutdown(context.Background()) }()
	instrumentation, err := NewInstrumentation(InstrumentationConfig{
		Service:       "tally-worker",
		MeterProvider: meterProvider,
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	instrumentation.ObserveHTTPRequest(ctx, 25*time.Millisecond, "/health/live", "GET", "2xx")
	instrumentation.ObserveDBTransaction(ctx, 40*time.Millisecond, "platform.integration", "outbox_publication", "success")
	instrumentation.SetOutboxBacklogSnapshot(ctx, []OutboxBacklogPoint{{EventType: "SyntheticEvent", Pending: 3, OldestAge: 2 * time.Second}})
	instrumentation.AddInboxFailure(ctx, 1, "synthetic_consumer", "inbox_processing_failed")
	for index := 0; index < maxLabelValues*2; index++ {
		instrumentation.AddCommand(ctx, 1, "platform.synthetic", "operation-"+strconv.Itoa(index), "success")
	}

	var resourceMetrics metricdata.ResourceMetrics
	if err := reader.Collect(ctx, &resourceMetrics); err != nil {
		t.Fatal(err)
	}
	if len(resourceMetrics.ScopeMetrics) != 1 {
		t.Fatalf("scope metrics = %d, want 1", len(resourceMetrics.ScopeMetrics))
	}

	wanted := map[string]bool{
		metricHTTPDuration:      false,
		metricCommandTotal:      false,
		metricDBDuration:        false,
		metricOutboxPending:     false,
		metricOutboxOldestAge:   false,
		metricInboxFailureTotal: false,
	}
	for _, value := range resourceMetrics.ScopeMetrics[0].Metrics {
		if _, ok := wanted[value.Name]; !ok {
			t.Fatalf("unexpected metric %q", value.Name)
		}
		wanted[value.Name] = true
		if points := metricDataPointCount(value.Data); points > maxLabelValues {
			t.Fatalf("metric %q has %d points, want at most %d", value.Name, points, maxLabelValues)
		}
	}
	for name, present := range wanted {
		if !present {
			t.Errorf("metric %q was not recorded", name)
		}
	}
}

func TestRequestTracingMiddlewareUsesCanonicalRouteAndStatusClass(t *testing.T) {
	recorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))
	defer func() { _ = tracerProvider.Shutdown(context.Background()) }()
	reader := metric.NewManualReader()
	meterProvider := metric.NewMeterProvider(metric.WithReader(reader))
	defer func() { _ = meterProvider.Shutdown(context.Background()) }()
	instrumentation, err := NewInstrumentation(InstrumentationConfig{
		Service:        "tally-api",
		TracerProvider: tracerProvider,
		MeterProvider:  meterProvider,
	})
	if err != nil {
		t.Fatal(err)
	}

	router := chi.NewRouter()
	router.Use(RequestTracingMiddleware(instrumentation))
	router.Get("/health/live", func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	})
	recorderHTTP := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live?token=not-recorded", nil)
	router.ServeHTTP(recorderHTTP, request)
	if recorderHTTP.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d", recorderHTTP.Code, http.StatusNoContent)
	}
	ended := recorder.Ended()
	if len(ended) != 1 {
		t.Fatalf("ended spans = %d, want 1", len(ended))
	}
	attributes := make(map[string]string)
	for _, attribute := range ended[0].Attributes() {
		attributes[string(attribute.Key)] = attribute.Value.AsString()
	}
	if attributes["route"] != "/health/live" || attributes["method"] != "GET" || attributes["status_class"] != "2xx" {
		t.Fatalf("request span attributes = %#v", attributes)
	}
	if strings.Contains(attributes["route"], "token") {
		t.Fatal("query string was included in route attribute")
	}
}

func metricDataPointCount(data metricdata.Aggregation) int {
	switch value := data.(type) {
	case metricdata.Sum[int64]:
		return len(value.DataPoints)
	case metricdata.Sum[float64]:
		return len(value.DataPoints)
	case metricdata.Gauge[int64]:
		return len(value.DataPoints)
	case metricdata.Gauge[float64]:
		return len(value.DataPoints)
	case metricdata.Histogram[int64]:
		return len(value.DataPoints)
	case metricdata.Histogram[float64]:
		return len(value.DataPoints)
	default:
		return 0
	}
}
