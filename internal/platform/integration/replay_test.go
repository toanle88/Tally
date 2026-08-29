package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestReplayRequestValidation(t *testing.T) {
	base := ReplayRequest{
		SourceContext: "synthetic",
		From:          time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		To:            time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		ConsumerName:  "projection",
		Generation:    "v1",
		MaxEvents:     maxReplayEvents,
	}
	if !validReplayRequest(base) {
		t.Fatal("valid replay request was rejected")
	}
	for name, request := range map[string]ReplayRequest{
		"reversed range":    func() ReplayRequest { value := base; value.To = value.From; return value }(),
		"too many events":   func() ReplayRequest { value := base; value.MaxEvents = maxReplayEvents + 1; return value }(),
		"reserved consumer": func() ReplayRequest { value := base; value.ConsumerName = "projection.replay.live"; return value }(),
		"empty generation":  func() ReplayRequest { value := base; value.Generation = ""; return value }(),
	} {
		if validReplayRequest(request) {
			t.Errorf("%s was accepted", name)
		}
	}
}

func TestNewReplayerRejectsInvalidRegistrations(t *testing.T) {
	if _, err := NewReplayer(nil, nil); !errors.Is(err, ErrInvalidReplay) {
		t.Fatalf("invalid replayer error = %v", err)
	}
	if _, err := NewReplayer(structTxBeginner{}, []ReplayRegistration{{Key: HandlerKey{EventType: "SyntheticEvent", EventVersion: 1}}}); !errors.Is(err, ErrInvalidReplay) {
		t.Fatalf("nil replay effect error = %v", err)
	}
}

type structTxBeginner struct{}

func (structTxBeginner) BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error) {
	return nil, nil
}
