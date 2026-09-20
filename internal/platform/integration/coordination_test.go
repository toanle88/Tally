package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/events"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

func testEnvelope(t *testing.T, messageID string, payload []byte) events.Envelope {
	t.Helper()
	fingerprint, err := events.ComputePayloadFingerprint(payload)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := events.NewEnvelope(events.EnvelopeInput{
		MessageID:          messageID,
		EventType:          "SyntheticEvent",
		EventVersion:       1,
		OccurredAt:         time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC),
		SourceContext:      "synthetic",
		AggregateID:        uuid.NewString(),
		AggregateVersion:   1,
		CorrelationID:      uuid.NewString(),
		CausationID:        uuid.NewString(),
		DataClassification: events.Internal,
		PayloadFingerprint: fingerprint,
		Data:               payload,
	})
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func TestOutcomeResultIsDefensivelyCopied(t *testing.T) {
	result := ResultReference(`{"reference":"established"}`)
	outcome := NewOutcome(OutcomeEstablished, result)
	result[0] = 'X'

	got := outcome.Result()
	got[0] = 'Y'
	if string(outcome.Result()) != `{"reference":"established"}` {
		t.Fatalf("stored result reference changed after caller mutation: %q", outcome.Result())
	}
}

func TestIdentityContentConflictIsTyped(t *testing.T) {
	conflict := IdentityContentConflict{
		ConsumerName:        "projection",
		MessageID:           uuid.NewString(),
		StoredFingerprint:   "sha256:stored",
		ReceivedFingerprint: "sha256:received",
	}
	if !errors.Is(conflict, ErrIdentityContentConflict) {
		t.Fatalf("conflict does not unwrap to ErrIdentityContentConflict: %v", conflict)
	}
	if conflict.Error() == "" {
		t.Fatal("conflict error is empty")
	}
}

func TestValidationRejectsInvalidDeliveryAndPublication(t *testing.T) {
	envelope := testEnvelope(t, uuid.NewString(), []byte(`{"event":"test"}`))
	if err := validateDelivery(Delivery{ConsumerName: "", Event: envelope}); !errors.Is(err, ErrInvalidDelivery) {
		t.Fatalf("invalid delivery error = %v", err)
	}
	if err := validatePublication(Publication{Event: envelope}); !errors.Is(err, ErrInvalidPublication) {
		t.Fatalf("invalid publication error = %v", err)
	}
}

func TestPublishPropagatesEnvelopeContextToTransactionBoundary(t *testing.T) {
	envelope := testEnvelope(t, uuid.NewString(), []byte(`{"event":"test"}`))
	beginErr := errors.New("begin failed")
	beginner := &capturingTxBeginner{err: beginErr}

	err := NewCoordinator(beginner).Publish(context.Background(), Publication{
		Event:       envelope,
		AvailableAt: time.Now().UTC(),
	}, func(context.Context, pgx.Tx) error {
		t.Fatal("source effect was called after transaction begin failed")
		return nil
	})
	if !errors.Is(err, beginErr) {
		t.Fatalf("publish error = %v, want %v", err, beginErr)
	}
	value, ok := telemetry.FromContext(beginner.ctx)
	if !ok {
		t.Fatal("transaction boundary did not receive telemetry context")
	}
	correlationID, _ := uuid.Parse(envelope.CorrelationID())
	causationID, _ := uuid.Parse(envelope.CausationID())
	if value.CorrelationID != correlationID || value.CausationID != causationID {
		t.Fatalf("transaction context = %#v, want correlation=%s causation=%s", value, correlationID, causationID)
	}
}

func TestWithEventContextBridgesEnvelopeIdentity(t *testing.T) {
	envelope := testEnvelope(t, uuid.NewString(), []byte(`{"event":"test"}`))
	ctx, err := withEventContext(context.Background(), envelope)
	if err != nil {
		t.Fatal(err)
	}
	value, ok := telemetry.FromContext(ctx)
	if !ok {
		t.Fatal("event context is missing")
	}
	correlationID, _ := uuid.Parse(envelope.CorrelationID())
	causationID, _ := uuid.Parse(envelope.CausationID())
	if value.CorrelationID != correlationID || value.CausationID != causationID {
		t.Fatalf("event context = %#v, want correlation=%s causation=%s", value, correlationID, causationID)
	}
}

type capturingTxBeginner struct {
	ctx context.Context
	err error
}

func (b *capturingTxBeginner) BeginTx(ctx context.Context, _ pgx.TxOptions) (pgx.Tx, error) {
	b.ctx = ctx
	return nil, b.err
}
