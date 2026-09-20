package integration

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/events"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

// withEventContext bridges the already-validated platform envelope identity
// into the runtime context used by database, inbox, outbox, and handler
// callbacks. Trace metadata remains runtime-only by design.
func withEventContext(ctx context.Context, event events.Envelope) (context.Context, error) {
	correlationID, err := uuid.Parse(event.CorrelationID())
	if err != nil {
		return nil, fmt.Errorf("invalid event correlation id: %w", err)
	}
	causationID, err := uuid.Parse(event.CausationID())
	if err != nil {
		return nil, fmt.Errorf("invalid event causation id: %w", err)
	}
	return telemetry.WithMessage(ctx, correlationID, causationID)
}
