package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
)

const replayTransactionSetting = "tally.replay"

const (
	maxReplayEvents = 10_000
	replaySeparator = ".replay."
)

var (
	ErrInvalidReplay       = errors.New("invalid integration replay")
	ErrReplayRangeTooLarge = errors.New("integration replay range exceeds maximum")
	ErrReplayIntegrity     = errors.New("integration replay integrity failure")
)

type ReplayRequest struct {
	SourceContext string
	From          time.Time
	To            time.Time
	ConsumerName  string
	Generation    string
	MaxEvents     int
}

type ReplayEffect func(context.Context, pgx.Tx, events.Envelope) (ResultReference, error)

type ReplayRegistration struct {
	Key    HandlerKey
	Effect ReplayEffect
}

type ReplayReport struct {
	Selected          int
	Established       int
	Deduplicated      int
	Failed            int
	IntegrityFailures int
}

type Replayer struct {
	db       TxBeginner
	handlers map[HandlerKey]ReplayEffect
}

func NewReplayer(db TxBeginner, registrations []ReplayRegistration) (*Replayer, error) {
	if db == nil || len(registrations) == 0 {
		return nil, ErrInvalidReplay
	}
	handlers := make(map[HandlerKey]ReplayEffect, len(registrations))
	for _, registration := range registrations {
		if registration.Key.EventType == "" || registration.Key.EventVersion < 1 || registration.Effect == nil {
			return nil, ErrInvalidReplay
		}
		if _, exists := handlers[registration.Key]; exists {
			return nil, fmt.Errorf("%w: duplicate replay registration", ErrInvalidReplay)
		}
		handlers[registration.Key] = registration.Effect
	}
	return &Replayer{db: db, handlers: handlers}, nil
}

func (r *Replayer) Replay(ctx context.Context, request ReplayRequest) (ReplayReport, error) {
	if r == nil || r.db == nil || ctx == nil {
		return ReplayReport{}, ErrInvalidReplay
	}
	request.From = request.From.UTC()
	request.To = request.To.UTC()
	if !validReplayRequest(request) {
		return ReplayReport{}, ErrInvalidReplay
	}
	consumerName := request.ConsumerName + replaySeparator + request.Generation

	selectionTx, err := r.db.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return ReplayReport{}, err
	}
	rows, err := platformdb.New(selectionTx).ListOutboxForReplay(ctx, platformdb.ListOutboxForReplayParams{
		SourceContext: request.SourceContext,
		FromTime:      timestamptzValue(request.From),
		ToTime:        timestamptzValue(request.To),
		MaxEvents:     int32(request.MaxEvents + 1),
	})
	if err != nil {
		_ = selectionTx.Rollback(ctx)
		return ReplayReport{}, err
	}
	if err := selectionTx.Commit(ctx); err != nil {
		return ReplayReport{}, fmt.Errorf("%w: %v", ErrCommitAmbiguous, err)
	}
	if len(rows) > request.MaxEvents {
		return ReplayReport{}, ErrReplayRangeTooLarge
	}

	report := ReplayReport{Selected: len(rows)}
	var joined error
	coordinator := NewCoordinator(r.db)
	for _, row := range rows {
		event, err := envelopeFromOutbox(row)
		if err != nil {
			report.IntegrityFailures++
			joined = errors.Join(joined, fmt.Errorf("%w: %v", ErrReplayIntegrity, err))
			continue
		}
		effect, ok := r.handlers[HandlerKey{EventType: event.EventType(), EventVersion: event.EventVersion()}]
		if !ok {
			report.IntegrityFailures++
			joined = errors.Join(joined, fmt.Errorf("%w: replay handler not registered for %s/%d", ErrReplayIntegrity, event.EventType(), event.EventVersion()))
			continue
		}
		outcome, existing, err := coordinator.consumeReplay(ctx, Delivery{ConsumerName: consumerName, Event: event}, effect)
		if err != nil {
			report.Failed++
			joined = errors.Join(joined, err)
			continue
		}
		if outcome.State == OutcomeEstablished {
			report.Established++
			if existing {
				report.Deduplicated++
			}
		}
	}
	return report, joined
}

func (c *Coordinator) consumeReplay(ctx context.Context, delivery Delivery, effect ReplayEffect) (Outcome, bool, error) {
	if err := c.validate(ctx); err != nil || effect == nil {
		return Outcome{}, false, ErrInvalidCoordinator
	}
	if err := validateDelivery(delivery); err != nil {
		return Outcome{}, false, err
	}
	tx, err := c.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return Outcome{}, false, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(ctx)
		}
	}()

	queries := platformdb.New(tx)
	if _, err := tx.Exec(ctx, "SELECT set_config($1, 'on', true)", replayTransactionSetting); err != nil {
		return Outcome{}, false, err
	}
	inbox, insertErr := queries.InsertInboxIfAbsent(ctx, inboxInsertParams(delivery))
	existing := false
	if errors.Is(insertErr, pgx.ErrNoRows) {
		existing = true
		inbox, err = queries.GetInboxForUpdate(ctx, inboxIdentityParams(delivery))
		if err != nil {
			return Outcome{}, false, err
		}
		if err := validateInboxFingerprint(delivery, inbox); err != nil {
			return Outcome{}, false, err
		}
		if inbox.State == string(OutcomeEstablished) {
			outcome, err := outcomeForInbox(inbox)
			return outcome, true, err
		}
	} else if insertErr != nil {
		return Outcome{}, false, insertErr
	}

	result, effectErr := effect(ctx, tx, delivery.Event)
	if effectErr != nil {
		_ = rollbackTransaction(tx, ctx)
		return Outcome{State: OutcomeFailed}, existing, c.recordFailure(ctx, delivery, effectErr)
	}
	if err := establishInbox(ctx, tx, delivery, result); err != nil {
		_ = rollbackTransaction(tx, ctx)
		return Outcome{State: OutcomeFailed}, existing, c.recordFailure(ctx, delivery, err)
	}
	if err := tx.Commit(ctx); err != nil {
		return Outcome{}, existing, fmt.Errorf("%w: %v", ErrCommitAmbiguous, err)
	}
	committed = true
	return NewOutcome(OutcomeEstablished, result), false, nil
}

func validReplayRequest(request ReplayRequest) bool {
	if request.SourceContext == "" || request.ConsumerName == "" || request.Generation == "" || request.MaxEvents < 1 || request.MaxEvents > maxReplayEvents || !request.From.Before(request.To) {
		return false
	}
	if !validIdentifier(request.SourceContext) || !validIdentifier(request.ConsumerName) || !validIdentifier(request.Generation) {
		return false
	}
	return !strings.Contains(request.ConsumerName, replaySeparator) && !strings.Contains(request.Generation, replaySeparator)
}
