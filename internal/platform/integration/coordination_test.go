package integration

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/events"
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
