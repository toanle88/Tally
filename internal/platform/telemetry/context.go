// Package telemetry contains technical-only context propagation primitives.
//
// It intentionally carries identifiers rather than business payloads. Domain
// modules remain responsible for authorization, audit evidence, and financial
// facts.
package telemetry

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrInvalidContext       = errors.New("invalid telemetry context")
	ErrMissingContext       = errors.New("telemetry context is missing")
	ErrInvalidCorrelationID = errors.New("invalid correlation id")
	ErrInvalidCausationID   = errors.New("invalid causation id")
)

// TelemetryContext is the identifier-only context shared by technical
// boundaries. Trace and span identifiers are optional until a tracer creates
// a span. Correlation is required for every request or unit of work; causation
// becomes required when a command or message identity exists.
type TelemetryContext struct {
	TraceID    trace.TraceID
	SpanID     trace.SpanID
	TraceFlags trace.TraceFlags
	TraceState trace.TraceState

	CorrelationID uuid.UUID
	CausationID   uuid.UUID
}

// NewRoot returns a valid root context. A nil correlation ID is replaced with
// a new UUID so callers cannot accidentally create an uncorrelated request.
func NewRoot(correlationID uuid.UUID) TelemetryContext {
	if correlationID == uuid.Nil {
		correlationID = uuid.New()
	}
	return TelemetryContext{CorrelationID: correlationID}
}

// Validate checks the structural context invariants. Trace and span IDs must
// be present together because a W3C trace context cannot contain only one.
func (value TelemetryContext) Validate() error {
	if value.CorrelationID == uuid.Nil {
		return fmt.Errorf("%w: correlation id is required", ErrInvalidContext)
	}
	if value.TraceID.IsValid() != value.SpanID.IsValid() {
		return fmt.Errorf("%w: trace and span ids must be present together", ErrInvalidContext)
	}
	return nil
}

// With stores a validated copy of value in ctx.
func With(ctx context.Context, value TelemetryContext) (context.Context, error) {
	if ctx == nil {
		return nil, fmt.Errorf("%w: parent context is required", ErrInvalidContext)
	}
	if err := value.Validate(); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, contextKey{}, value), nil
}

// FromContext returns a copy of the technical context when one is present.
func FromContext(ctx context.Context) (TelemetryContext, bool) {
	if ctx == nil {
		return TelemetryContext{}, false
	}
	value, ok := ctx.Value(contextKey{}).(TelemetryContext)
	return value, ok
}

// Detach preserves telemetry values while removing cancellation and deadline
// state. It is intended for bounded failure-recording work that must outlive
// the failed operation without losing its diagnostic identity.
func Detach(ctx context.Context) context.Context {
	value, ok := FromContext(ctx)
	if !ok {
		return context.Background()
	}
	detached, err := With(context.Background(), value)
	if err != nil {
		return context.Background()
	}
	return detached
}

// WithCausation attaches the command or message identity that directly
// caused the current operation.
func WithCausation(ctx context.Context, causationID uuid.UUID) (context.Context, error) {
	if causationID == uuid.Nil {
		return nil, ErrInvalidCausationID
	}
	value, ok := FromContext(ctx)
	if !ok {
		return nil, ErrMissingContext
	}
	value.CausationID = causationID
	return With(ctx, value)
}

// WithMessage replaces the correlation and causation identifiers from an
// existing platform message while preserving any runtime trace context.
// Existing event-envelope validation remains the source of truth for message
// structure; this helper only bridges the validated IDs into context.Context.
func WithMessage(ctx context.Context, correlationID, causationID uuid.UUID) (context.Context, error) {
	if correlationID == uuid.Nil {
		return nil, ErrInvalidCorrelationID
	}
	if causationID == uuid.Nil {
		return nil, ErrInvalidCausationID
	}
	value, ok := FromContext(ctx)
	if !ok {
		value = NewRoot(correlationID)
	} else {
		value.CorrelationID = correlationID
	}
	value.CausationID = causationID
	return With(ctx, value)
}

// SpanContext converts the runtime identifiers to an OpenTelemetry span
// context for propagation or later instrumentation.
func (value TelemetryContext) SpanContext() trace.SpanContext {
	if !value.TraceID.IsValid() || !value.SpanID.IsValid() {
		return trace.SpanContext{}
	}
	return trace.NewSpanContext(trace.SpanContextConfig{
		TraceID:    value.TraceID,
		SpanID:     value.SpanID,
		TraceState: value.TraceState,
		TraceFlags: value.TraceFlags,
	})
}

type contextKey struct{}
