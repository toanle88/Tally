package integration

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
	platformworker "github.com/toanle88/Tally/internal/platform/worker"
)

const (
	defaultOutboxLease             = 30 * time.Second
	defaultOutboxBatchSize         = 100
	defaultOutboxConcurrency       = 100
	maxOutboxAttempts        int32 = 10
)

var (
	ErrInvalidDispatcher = errors.New("invalid outbox dispatcher")
	ErrLeaseLost         = errors.New("outbox lease lost")
)

// OutboxStore is the platform-owned persistence boundary used by Dispatcher.
type OutboxStore interface {
	ClaimDueOutbox(context.Context, platformdb.ClaimDueOutboxParams) ([]platformdb.IntegrationOutbox, error)
	RenewOutboxLease(context.Context, platformdb.RenewOutboxLeaseParams) (int64, error)
	EstablishOutbox(context.Context, platformdb.EstablishOutboxParams) (int64, error)
	RescheduleOutbox(context.Context, platformdb.RescheduleOutboxParams) (int64, error)
	MarkOutboxManagedException(context.Context, platformdb.MarkOutboxManagedExceptionParams) (int64, error)
}

// Handler processes one valid event envelope. Business effects remain owned by
// the receiving bounded context and must provide their own idempotency rules.
type Handler func(context.Context, events.Envelope) error

type FailureClass string

const (
	TransientDependency   FailureClass = "transient_dependency"
	DomainRejection       FailureClass = "domain_rejection"
	AuthorizationDenied   FailureClass = "authorization_denied"
	IdempotencyConflict   FailureClass = "idempotency_conflict"
	DataIntegrityMismatch FailureClass = "data_integrity_mismatch"
)

// HandlerFailure is the safe, typed failure contract used by the dispatcher.
// Only TransientDependency is automatically retried.
type HandlerFailure struct {
	Class FailureClass
	Code  string
	Cause error
}

func (e HandlerFailure) Error() string {
	if e.Code == "" {
		return string(e.Class)
	}
	return e.Code
}

func (e HandlerFailure) Unwrap() error { return e.Cause }

type HandlerKey struct {
	EventType    string
	EventVersion int
}

type RetryPolicy struct {
	Delays []time.Duration
}

func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{Delays: []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute}}
}

func (p RetryPolicy) delayForAttempt(attempt int32) time.Duration {
	if len(p.Delays) == 0 {
		return 0
	}
	index := int(attempt - 1)
	if index < 0 {
		index = 0
	}
	if index >= len(p.Delays) {
		index = len(p.Delays) - 1
	}
	return p.Delays[index]
}

type HandlerRegistration struct {
	Key         HandlerKey
	Handler     Handler
	RetryPolicy RetryPolicy
}

type Clock interface {
	Now() time.Time
	Sleep(context.Context, time.Duration) error
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

func (realClock) Sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type DispatcherConfig struct {
	LeaseDuration        time.Duration
	HandlerP99Duration   time.Duration
	BatchSize            int
	Concurrency          int
	Admission            platformworker.Admission
	ConcurrencyAdmission platformworker.Admission
	Clock                Clock
}

type DispatchReport struct {
	Claimed           int
	Established       int
	Rescheduled       int
	ManagedExceptions int
	LeaseLost         int
}

type Dispatcher struct {
	store                OutboxStore
	handlers             map[HandlerKey]HandlerRegistration
	config               DispatcherConfig
	admission            platformworker.Admission
	concurrencyAdmission platformworker.Admission
}

func NewDispatcher(store OutboxStore, registrations []HandlerRegistration, config DispatcherConfig) (*Dispatcher, error) {
	if store == nil || len(registrations) == 0 {
		return nil, ErrInvalidDispatcher
	}
	if config.LeaseDuration == 0 {
		config.LeaseDuration = defaultOutboxLease
	}
	if config.HandlerP99Duration <= 0 || config.LeaseDuration <= config.HandlerP99Duration {
		return nil, fmt.Errorf("%w: lease duration must exceed handler p99 duration", ErrInvalidDispatcher)
	}
	if config.BatchSize == 0 {
		config.BatchSize = defaultOutboxBatchSize
	}
	if config.Concurrency == 0 {
		config.Concurrency = defaultOutboxConcurrency
	}
	if config.BatchSize < 1 || config.Concurrency < 1 {
		return nil, fmt.Errorf("%w: batch size and concurrency must be positive", ErrInvalidDispatcher)
	}
	if config.Admission == nil {
		admission, err := platformworker.NewAdmission(config.Concurrency)
		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrInvalidDispatcher, err)
		}
		config.Admission = admission
	}
	if config.Admission.Capacity() < 1 {
		return nil, fmt.Errorf("%w: admission capacity must be positive", ErrInvalidDispatcher)
	}
	if config.ConcurrencyAdmission == nil {
		admission, err := platformworker.NewAdmission(config.Concurrency)
		if err != nil {
			return nil, fmt.Errorf("%w: concurrency admission must be positive", ErrInvalidDispatcher)
		}
		config.ConcurrencyAdmission = admission
	}
	if config.ConcurrencyAdmission.Capacity() < 1 {
		return nil, fmt.Errorf("%w: concurrency admission capacity must be positive", ErrInvalidDispatcher)
	}
	if config.Clock == nil {
		config.Clock = realClock{}
	}

	handlers := make(map[HandlerKey]HandlerRegistration, len(registrations))
	for _, registration := range registrations {
		if registration.Key.EventType == "" || registration.Key.EventVersion < 1 || registration.Handler == nil {
			return nil, ErrInvalidDispatcher
		}
		if len(registration.RetryPolicy.Delays) == 0 {
			registration.RetryPolicy = DefaultRetryPolicy()
		}
		for _, delay := range registration.RetryPolicy.Delays {
			if delay <= 0 {
				return nil, fmt.Errorf("%w: retry delays must be positive", ErrInvalidDispatcher)
			}
		}
		if _, exists := handlers[registration.Key]; exists {
			return nil, fmt.Errorf("%w: duplicate handler registration", ErrInvalidDispatcher)
		}
		handlers[registration.Key] = registration
	}

	return &Dispatcher{store: store, handlers: handlers, config: config, admission: config.Admission, concurrencyAdmission: config.ConcurrencyAdmission}, nil
}

// DispatchOnce claims and processes one bounded batch. Item failures are
// durably recorded; returned errors indicate persistence or lease failures.
func (d *Dispatcher) DispatchOnce(ctx context.Context) (DispatchReport, error) {
	if d == nil || d.store == nil || ctx == nil {
		return DispatchReport{}, ErrInvalidDispatcher
	}
	owner := uuid.NewString()
	batchSize := d.config.BatchSize
	if capacity := d.admission.Capacity(); batchSize > capacity {
		batchSize = capacity
	}
	if capacity := d.concurrencyAdmission.Capacity(); batchSize > capacity {
		batchSize = capacity
	}
	if batchSize > d.config.Concurrency {
		batchSize = d.config.Concurrency
	}
	claimed, err := d.store.ClaimDueOutbox(ctx, platformdb.ClaimDueOutboxParams{
		LeaseDuration: intervalValue(d.config.LeaseDuration),
		ClaimOwner:    pgtype.Text{String: owner, Valid: true},
		BatchSize:     int32(batchSize),
	})
	if err != nil {
		return DispatchReport{}, err
	}

	report := DispatchReport{Claimed: len(claimed)}
	results := make(chan dispatchResult, len(claimed))
	semaphore := make(chan struct{}, d.config.Concurrency)
	var wg sync.WaitGroup
	for _, row := range claimed {
		row := row
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := d.concurrencyAdmission.Acquire(ctx); err != nil {
				results <- dispatchResult{err: err}
				return
			}
			defer d.concurrencyAdmission.Release()
			if err := d.admission.Acquire(ctx); err != nil {
				results <- dispatchResult{err: err}
				return
			}
			defer d.admission.Release()
			select {
			case semaphore <- struct{}{}:
			case <-ctx.Done():
				results <- dispatchResult{err: ctx.Err()}
				return
			}
			defer func() { <-semaphore }()
			results <- d.process(ctx, row, owner)
		}()
	}
	wg.Wait()
	close(results)

	var joined error
	for result := range results {
		switch result.outcome {
		case dispatchEstablished:
			report.Established++
		case dispatchRescheduled:
			report.Rescheduled++
		case dispatchManaged:
			report.ManagedExceptions++
		case dispatchLeaseLost:
			report.LeaseLost++
		}
		joined = errors.Join(joined, result.err)
	}
	return report, joined
}

// Run polls the dispatcher until cancellation or a claim-level persistence
// failure. Lease loss for an individual item is expected under contention.
func (d *Dispatcher) Run(ctx context.Context, pollInterval time.Duration) error {
	if d == nil || ctx == nil || pollInterval <= 0 {
		return ErrInvalidDispatcher
	}
	for {
		_, err := d.DispatchOnce(ctx)
		if ctx.Err() != nil {
			return nil
		}
		if err != nil && !errors.Is(err, ErrLeaseLost) {
			return err
		}
		if err := d.config.Clock.Sleep(ctx, pollInterval); err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
				return nil
			}
			return err
		}
	}
}

type dispatchOutcome int

const (
	dispatchEstablished dispatchOutcome = iota + 1
	dispatchRescheduled
	dispatchManaged
	dispatchLeaseLost
)

type dispatchResult struct {
	outcome dispatchOutcome
	err     error
}

func (d *Dispatcher) process(ctx context.Context, row platformdb.IntegrationOutbox, owner string) dispatchResult {
	event, err := envelopeFromOutbox(row)
	if err != nil {
		return d.manage(ctx, row, owner, string(DataIntegrityMismatch), dispatchManaged)
	}
	registration, ok := d.handlers[HandlerKey{EventType: event.EventType(), EventVersion: event.EventVersion()}]
	if !ok {
		return d.manage(ctx, row, owner, "handler_not_registered", dispatchManaged)
	}

	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	handlerDone := make(chan error, 1)
	leaseLost := make(chan error, 1)
	go func() { handlerDone <- registration.Handler(handlerCtx, event) }()
	go d.renewLease(handlerCtx, row, owner, leaseLost)

	select {
	case err = <-handlerDone:
	case err = <-leaseLost:
		cancel()
		return dispatchResult{outcome: dispatchLeaseLost, err: err}
	}
	if err == nil {
		rows, establishErr := d.store.EstablishOutbox(ctx, platformdb.EstablishOutboxParams{OutboxID: row.OutboxID, ClaimOwner: pgtype.Text{String: owner, Valid: true}})
		if establishErr != nil {
			return dispatchResult{outcome: dispatchLeaseLost, err: establishErr}
		}
		if rows != 1 {
			return dispatchResult{outcome: dispatchLeaseLost, err: ErrLeaseLost}
		}
		return dispatchResult{outcome: dispatchEstablished}
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return dispatchResult{outcome: dispatchLeaseLost, err: err}
	}

	failure, typedFailure := asHandlerFailure(err)
	failureCode, validCode := "", false
	if typedFailure {
		failureCode, validCode = normalizeFailureCode(failure.Code)
	}
	if !typedFailure || failure.Class != TransientDependency || !validCode {
		code := "unexpected_handler_failure"
		if typedFailure && validCode {
			code = failureCode
		} else if typedFailure && failure.Code != "" {
			code = "invalid_failure_code"
		}
		return d.manage(ctx, row, owner, code, dispatchManaged)
	}
	if row.AttemptCount >= maxOutboxAttempts {
		return d.manage(ctx, row, owner, failureCode, dispatchManaged)
	}

	availableAt := d.config.Clock.Now().Add(registration.RetryPolicy.delayForAttempt(row.AttemptCount))
	rows, rescheduleErr := d.store.RescheduleOutbox(ctx, platformdb.RescheduleOutboxParams{
		AvailableAt:   timestamptzValue(availableAt),
		LastErrorCode: pgtype.Text{String: failureCode, Valid: true},
		OutboxID:      row.OutboxID,
		ClaimOwner:    pgtype.Text{String: owner, Valid: true},
	})
	if rescheduleErr != nil {
		return dispatchResult{outcome: dispatchLeaseLost, err: rescheduleErr}
	}
	if rows != 1 {
		return dispatchResult{outcome: dispatchLeaseLost, err: ErrLeaseLost}
	}
	return dispatchResult{outcome: dispatchRescheduled}
}

func normalizeFailureCode(code string) (string, bool) {
	if len(code) == 0 || len(code) > 128 {
		return "", false
	}
	for _, character := range code {
		if (character < 'a' || character > 'z') &&
			(character < '0' || character > '9') &&
			character != '_' && character != '-' && character != '.' {
			return "", false
		}
	}
	return code, true
}

func asHandlerFailure(err error) (HandlerFailure, bool) {
	var value HandlerFailure
	if errors.As(err, &value) {
		return value, true
	}
	var pointer *HandlerFailure
	if errors.As(err, &pointer) && pointer != nil {
		return *pointer, true
	}
	return HandlerFailure{}, false
}

func (d *Dispatcher) renewLease(ctx context.Context, row platformdb.IntegrationOutbox, owner string, lost chan<- error) {
	for {
		if err := d.config.Clock.Sleep(ctx, leaseRenewalInterval(d.config.LeaseDuration)); err != nil {
			return
		}
		rows, err := d.store.RenewOutboxLease(ctx, platformdb.RenewOutboxLeaseParams{
			LeaseDuration: intervalValue(d.config.LeaseDuration),
			OutboxID:      row.OutboxID,
			ClaimOwner:    pgtype.Text{String: owner, Valid: true},
		})
		if err != nil {
			select {
			case lost <- err:
			default:
			}
			return
		}
		if rows != 1 {
			select {
			case lost <- ErrLeaseLost:
			default:
			}
			return
		}
	}
}

func leaseRenewalInterval(lease time.Duration) time.Duration {
	// Renew comfortably before two-thirds of the lease is consumed so scheduler
	// jitter cannot turn the renewal point into a lease-expiry race.
	return lease / 2
}

func (d *Dispatcher) manage(ctx context.Context, row platformdb.IntegrationOutbox, owner, code string, outcome dispatchOutcome) dispatchResult {
	rows, err := d.store.MarkOutboxManagedException(ctx, platformdb.MarkOutboxManagedExceptionParams{
		LastErrorCode: pgtype.Text{String: code, Valid: true},
		OutboxID:      row.OutboxID,
		ClaimOwner:    pgtype.Text{String: owner, Valid: true},
	})
	if err != nil {
		return dispatchResult{outcome: dispatchLeaseLost, err: err}
	}
	if rows != 1 {
		return dispatchResult{outcome: dispatchLeaseLost, err: ErrLeaseLost}
	}
	return dispatchResult{outcome: outcome}
}

func envelopeFromOutbox(row platformdb.IntegrationOutbox) (events.Envelope, error) {
	if !row.OutboxID.Valid || !row.AggregateID.Valid || !row.CorrelationID.Valid || !row.CausationID.Valid || !row.OccurredAt.Valid {
		return events.Envelope{}, ErrInvalidDelivery
	}
	scope := ""
	if row.AccountingScopeID.Valid {
		scope = uuid.UUID(row.AccountingScopeID.Bytes).String()
	}
	return events.NewEnvelope(events.EnvelopeInput{
		MessageID:          uuid.UUID(row.OutboxID.Bytes).String(),
		EventType:          row.EventType,
		EventVersion:       int(row.EventVersion),
		OccurredAt:         row.OccurredAt.Time,
		SourceContext:      row.SourceContext,
		AggregateID:        uuid.UUID(row.AggregateID.Bytes).String(),
		AggregateVersion:   row.AggregateVersion,
		AccountingScopeID:  scope,
		CorrelationID:      uuid.UUID(row.CorrelationID.Bytes).String(),
		CausationID:        uuid.UUID(row.CausationID.Bytes).String(),
		DataClassification: events.Classification(row.DataClassification),
		PayloadFingerprint: row.PayloadFingerprint,
		Data:               row.Payload,
	})
}

func intervalValue(duration time.Duration) pgtype.Interval {
	return pgtype.Interval{Microseconds: duration.Microseconds(), Valid: true}
}
