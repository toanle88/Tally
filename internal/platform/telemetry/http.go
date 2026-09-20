package telemetry

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

const CorrelationHeader = "X-Correlation-Id"

// Middleware establishes the request correlation context and preserves a
// valid W3C trace context when the caller supplies one. Invalid correlation
// headers are rejected rather than silently replaced.
func Middleware(next http.Handler) http.Handler {
	if next == nil {
		panic("telemetry: nil HTTP handler")
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		value, err := FromRequest(request)
		if err != nil {
			http.Error(writer, "invalid correlation id", http.StatusBadRequest)
			return
		}
		ctx, err := With(request.Context(), value)
		if err != nil {
			http.Error(writer, "invalid telemetry context", http.StatusInternalServerError)
			return
		}
		writer.Header().Set(CorrelationHeader, value.CorrelationID.String())
		next.ServeHTTP(writer, request.WithContext(ctx))
	})
}

// FromRequest creates a root context from an HTTP request. Correlation is
// application-owned and uses X-Correlation-Id; trace context is transport-
// owned and uses the standard W3C traceparent carrier.
func FromRequest(request *http.Request) (TelemetryContext, error) {
	if request == nil {
		return TelemetryContext{}, fmt.Errorf("%w: request is required", ErrInvalidContext)
	}

	correlationID := uuid.Nil
	if raw := request.Header.Get(CorrelationHeader); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			return TelemetryContext{}, fmt.Errorf("%w: %v", ErrInvalidCorrelationID, err)
		}
		correlationID = parsed
	}
	value := NewRoot(correlationID)

	propagated := propagation.TraceContext{}.Extract(request.Context(), propagation.HeaderCarrier(request.Header))
	spanContext := trace.SpanContextFromContext(propagated)
	if spanContext.IsValid() {
		value.TraceID = spanContext.TraceID()
		value.SpanID = spanContext.SpanID()
		value.TraceFlags = spanContext.TraceFlags()
		value.TraceState = spanContext.TraceState()
	}
	return value, nil
}

// InjectHTTP writes the current correlation and W3C trace context into an
// outbound HTTP header carrier. It does not create a span or an exporter.
func InjectHTTP(ctx context.Context, header http.Header) error {
	if header == nil {
		return fmt.Errorf("%w: header is required", ErrInvalidContext)
	}
	value, ok := FromContext(ctx)
	if !ok {
		return ErrMissingContext
	}
	header.Set(CorrelationHeader, value.CorrelationID.String())
	if value.TraceID.IsValid() && value.SpanID.IsValid() {
		carrierContext := trace.ContextWithSpanContext(context.Background(), value.SpanContext())
		propagation.TraceContext{}.Inject(carrierContext, propagation.HeaderCarrier(header))
	}
	return nil
}
