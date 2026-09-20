package telemetry

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

func TestNewRootGeneratesCorrelationID(t *testing.T) {
	first := NewRoot(uuid.Nil)
	second := NewRoot(uuid.Nil)
	if first.CorrelationID == uuid.Nil || second.CorrelationID == uuid.Nil {
		t.Fatal("root context did not generate correlation IDs")
	}
	if first.CorrelationID == second.CorrelationID {
		t.Fatal("root contexts reused a correlation ID")
	}
	if err := first.Validate(); err != nil {
		t.Fatalf("generated root is invalid: %v", err)
	}
}

func TestContextPropagationAndCausation(t *testing.T) {
	correlationID := uuid.New()
	commandID := uuid.New()
	root := NewRoot(correlationID)
	ctx, err := With(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	ctx, err = WithCausation(ctx, commandID)
	if err != nil {
		t.Fatal(err)
	}

	got, ok := FromContext(ctx)
	if !ok || got.CorrelationID != correlationID || got.CausationID != commandID {
		t.Fatalf("context = %#v, present = %v", got, ok)
	}
	if _, err := WithCausation(ctx, uuid.Nil); !errors.Is(err, ErrInvalidCausationID) {
		t.Fatalf("nil causation error = %v", err)
	}
}

func TestWithRejectsInvalidValuesAndDoesNotMutateParent(t *testing.T) {
	if _, err := With(context.Background(), TelemetryContext{}); !errors.Is(err, ErrInvalidContext) {
		t.Fatalf("invalid context error = %v", err)
	}
	value := NewRoot(uuid.New())
	ctx, err := With(context.Background(), value)
	if err != nil {
		t.Fatal(err)
	}
	child, err := WithCausation(ctx, uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	parent, _ := FromContext(ctx)
	derived, _ := FromContext(child)
	if parent.CausationID != uuid.Nil || derived.CausationID == uuid.Nil {
		t.Fatalf("parent = %#v, derived = %#v", parent, derived)
	}
}

func TestDetachPreservesTelemetryWithoutCancellation(t *testing.T) {
	root := NewRoot(uuid.New())
	ctx, err := With(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	detached := Detach(ctx)
	got, ok := FromContext(detached)
	if !ok || got.CorrelationID != root.CorrelationID {
		t.Fatalf("detached context = %#v, present = %v", got, ok)
	}
	if detached.Done() != nil {
		t.Fatal("detached context unexpectedly retained cancellation")
	}
}

func TestHTTPMiddlewareGeneratesAndReturnsCorrelationID(t *testing.T) {
	var got TelemetryContext
	handler := Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got, _ = FromContext(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d", recorder.Code)
	}
	if got.CorrelationID == uuid.Nil || recorder.Header().Get(CorrelationHeader) != got.CorrelationID.String() {
		t.Fatalf("context/header correlation mismatch: %#v / %q", got, recorder.Header().Get(CorrelationHeader))
	}
}

func TestHTTPMiddlewarePreservesValidCorrelationID(t *testing.T) {
	want := uuid.New()
	var got TelemetryContext
	handler := Middleware(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		got, _ = FromContext(request.Context())
		writer.WriteHeader(http.StatusNoContent)
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header.Set(CorrelationHeader, want.String())
	handler.ServeHTTP(recorder, request)

	if got.CorrelationID != want || recorder.Header().Get(CorrelationHeader) != want.String() {
		t.Fatalf("context/header correlation = %s / %q, want %s", got.CorrelationID, recorder.Header().Get(CorrelationHeader), want)
	}
}

func TestHTTPMiddlewareRejectsMalformedCorrelationID(t *testing.T) {
	handler := Middleware(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		t.Fatal("handler was called for malformed correlation ID")
	}))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	request.Header.Set(CorrelationHeader, "not-a-uuid")
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusBadRequest)
	}
}

func TestHTTPTraceContextRoundTrip(t *testing.T) {
	traceID, err := trace.TraceIDFromHex("4bf92f3577b34da6a3ce929d0e0e4736")
	if err != nil {
		t.Fatal(err)
	}
	spanID, err := trace.SpanIDFromHex("00f067aa0ba902b7")
	if err != nil {
		t.Fatal(err)
	}
	root := NewRoot(uuid.New())
	root.TraceID = traceID
	root.SpanID = spanID
	root.TraceFlags = trace.FlagsSampled
	ctx, err := With(context.Background(), root)
	if err != nil {
		t.Fatal(err)
	}
	header := make(http.Header)
	if err := InjectHTTP(ctx, header); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header = header
	got, err := FromRequest(request)
	if err != nil {
		t.Fatal(err)
	}
	if got.TraceID != traceID || got.SpanID != spanID || got.TraceFlags != trace.FlagsSampled {
		t.Fatalf("trace context = %#v", got)
	}
}

func TestInjectHTTPRequiresContext(t *testing.T) {
	if err := InjectHTTP(context.Background(), make(http.Header)); !errors.Is(err, ErrMissingContext) {
		t.Fatalf("missing context error = %v", err)
	}
}
