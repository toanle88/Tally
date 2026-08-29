package integration

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
	platformworker "github.com/toanle88/Tally/internal/platform/worker"
)

func TestDefaultRetryPolicyMapsAttempts(t *testing.T) {
	policy := DefaultRetryPolicy()
	want := []time.Duration{5 * time.Second, 30 * time.Second, 2 * time.Minute, 10 * time.Minute, 30 * time.Minute, 30 * time.Minute}
	for attempt, expected := range want {
		if got := policy.delayForAttempt(int32(attempt + 1)); got != expected {
			t.Fatalf("attempt %d delay = %s, want %s", attempt+1, got, expected)
		}
	}
}

func TestNewDispatcherValidatesLeaseAgainstHandlerP99(t *testing.T) {
	store := &dispatcherStore{}
	registration := HandlerRegistration{
		Key:     HandlerKey{EventType: "TestEvent", EventVersion: 1},
		Handler: func(context.Context, events.Envelope) error { return nil },
	}
	if _, err := NewDispatcher(store, []HandlerRegistration{registration}, DispatcherConfig{
		LeaseDuration:      time.Second,
		HandlerP99Duration: time.Second,
	}); !errors.Is(err, ErrInvalidDispatcher) {
		t.Fatalf("invalid lease error = %v, want ErrInvalidDispatcher", err)
	}
}

func TestDispatcherReschedulesTypedTransientFailure(t *testing.T) {
	now := time.Date(2026, 8, 29, 10, 0, 0, 0, time.UTC)
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}, establishRows: 1}
	dispatcher := newTestDispatcher(t, store, now, HandlerFailure{
		Class: TransientDependency,
		Code:  "provider_unavailable",
	})

	report, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Rescheduled != 1 || report.ManagedExceptions != 0 {
		t.Fatalf("report = %#v, want one reschedule", report)
	}
	if store.rescheduled == nil {
		t.Fatal("transient failure was not rescheduled")
	}
	wantAt := now.Add(5 * time.Second)
	if !store.rescheduled.AvailableAt.Time.Equal(wantAt) {
		t.Fatalf("available_at = %s, want %s", store.rescheduled.AvailableAt.Time, wantAt)
	}
	if store.rescheduled.LastErrorCode.String != "provider_unavailable" {
		t.Fatalf("last error code = %q", store.rescheduled.LastErrorCode.String)
	}
}

func TestDispatcherReschedulesWrappedPointerTransientFailure(t *testing.T) {
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}}
	dispatcher := newTestDispatcher(t, store, time.Now().UTC(), fmt.Errorf("wrapped: %w", &HandlerFailure{
		Class: TransientDependency,
		Code:  "provider_unavailable",
	}))

	report, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.Rescheduled != 1 || store.managedCode != "" {
		t.Fatalf("report = %#v, managed code = %q, want one reschedule", report, store.managedCode)
	}
}

func TestLeaseRenewalOccursBeforeTwoThirds(t *testing.T) {
	lease := 30 * time.Second
	if got, limit := leaseRenewalInterval(lease), lease*2/3; got >= limit {
		t.Fatalf("renewal interval = %s, want less than %s", got, limit)
	}
}

func TestDispatcherManagesNonTransientFailure(t *testing.T) {
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}}
	dispatcher := newTestDispatcher(t, store, time.Now().UTC(), HandlerFailure{
		Class: DomainRejection,
		Code:  "posting_rejected",
	})

	report, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.ManagedExceptions != 1 || store.managedCode != "posting_rejected" {
		t.Fatalf("report = %#v, managed code = %q", report, store.managedCode)
	}
	if store.rescheduled != nil {
		t.Fatal("non-transient failure was rescheduled")
	}
}

func TestDispatcherDoesNotPersistUnsafeFailureCode(t *testing.T) {
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}}
	dispatcher := newTestDispatcher(t, store, time.Now().UTC(), HandlerFailure{
		Class: DomainRejection,
		Code:  "secret\nvalue",
	})

	if _, err := dispatcher.DispatchOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.managedCode != "invalid_failure_code" {
		t.Fatalf("managed code = %q, want sanitized code", store.managedCode)
	}
}

func TestDispatcherManagesTenthTransientAttempt(t *testing.T) {
	row := dispatcherOutboxRow(t, maxOutboxAttempts)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}}
	dispatcher := newTestDispatcher(t, store, time.Now().UTC(), HandlerFailure{
		Class: TransientDependency,
		Code:  "provider_timeout",
	})

	report, err := dispatcher.DispatchOnce(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if report.ManagedExceptions != 1 || store.managedCode != "provider_timeout" {
		t.Fatalf("report = %#v, managed code = %q", report, store.managedCode)
	}
	if store.rescheduled != nil {
		t.Fatal("tenth transient attempt was rescheduled")
	}
}

func TestDispatcherFencesLostEstablishment(t *testing.T) {
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}, establishRows: 0}
	dispatcher := newTestDispatcher(t, store, time.Now().UTC(), nil)

	report, err := dispatcher.DispatchOnce(context.Background())
	if !errors.Is(err, ErrLeaseLost) || report.LeaseLost != 1 {
		t.Fatalf("report = %#v, error = %v, want lease lost", report, err)
	}
}

func TestDispatcherBoundsClaimsToAdmissionCapacity(t *testing.T) {
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}, establishRows: 1}
	admission, err := platformworker.NewAdmission(1)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewDispatcher(store, []HandlerRegistration{{
		Key:     HandlerKey{EventType: row.EventType, EventVersion: int(row.EventVersion)},
		Handler: func(context.Context, events.Envelope) error { return nil },
	}}, DispatcherConfig{
		LeaseDuration:      30 * time.Second,
		HandlerP99Duration: time.Second,
		BatchSize:          10,
		Concurrency:        10,
		Admission:          admission,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dispatcher.DispatchOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.claimBatch != 1 {
		t.Fatalf("claim batch = %d, want admission capacity 1", store.claimBatch)
	}
}

func TestDispatcherBoundsClaimsToConcurrencyAdmissionCapacity(t *testing.T) {
	row := dispatcherOutboxRow(t, 1)
	store := &dispatcherStore{rows: []platformdb.IntegrationOutbox{row}, establishRows: 1}
	dbAdmission, err := platformworker.NewAdmission(10)
	if err != nil {
		t.Fatal(err)
	}
	concurrencyAdmission, err := platformworker.NewAdmission(1)
	if err != nil {
		t.Fatal(err)
	}
	dispatcher, err := NewDispatcher(store, []HandlerRegistration{{
		Key:     HandlerKey{EventType: row.EventType, EventVersion: int(row.EventVersion)},
		Handler: func(context.Context, events.Envelope) error { return nil },
	}}, DispatcherConfig{
		LeaseDuration:        30 * time.Second,
		HandlerP99Duration:   time.Second,
		BatchSize:            10,
		Concurrency:          10,
		Admission:            dbAdmission,
		ConcurrencyAdmission: concurrencyAdmission,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := dispatcher.DispatchOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.claimBatch != 1 {
		t.Fatalf("claim batch = %d, want concurrency admission capacity 1", store.claimBatch)
	}
}

type dispatcherStore struct {
	rows          []platformdb.IntegrationOutbox
	claimBatch    int32
	rescheduled   *platformdb.RescheduleOutboxParams
	managedCode   string
	establishRows int64
}

func (s *dispatcherStore) ClaimDueOutbox(_ context.Context, params platformdb.ClaimDueOutboxParams) ([]platformdb.IntegrationOutbox, error) {
	s.claimBatch = params.BatchSize
	return s.rows, nil
}

func (s *dispatcherStore) RenewOutboxLease(context.Context, platformdb.RenewOutboxLeaseParams) (int64, error) {
	return 1, nil
}

func (s *dispatcherStore) EstablishOutbox(context.Context, platformdb.EstablishOutboxParams) (int64, error) {
	if s.establishRows == 0 {
		return 0, nil
	}
	return s.establishRows, nil
}

func (s *dispatcherStore) RescheduleOutbox(_ context.Context, params platformdb.RescheduleOutboxParams) (int64, error) {
	s.rescheduled = &params
	return 1, nil
}

func (s *dispatcherStore) MarkOutboxManagedException(_ context.Context, params platformdb.MarkOutboxManagedExceptionParams) (int64, error) {
	s.managedCode = params.LastErrorCode.String
	return 1, nil
}

func newTestDispatcher(t *testing.T, store OutboxStore, now time.Time, handlerError error) *Dispatcher {
	t.Helper()
	row := dispatcherOutboxRow(t, 1)
	dispatcher, err := NewDispatcher(store, []HandlerRegistration{{
		Key: HandlerKey{EventType: row.EventType, EventVersion: int(row.EventVersion)},
		Handler: func(context.Context, events.Envelope) error {
			return handlerError
		},
	}}, DispatcherConfig{
		LeaseDuration:      30 * time.Second,
		HandlerP99Duration: time.Second,
		BatchSize:          1,
		Concurrency:        1,
		Clock:              fixedDispatcherClock{now: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dispatcher
}

func dispatcherOutboxRow(t *testing.T, attempt int32) platformdb.IntegrationOutbox {
	t.Helper()
	data := []byte(`{"value":1}`)
	fingerprint, err := events.ComputePayloadFingerprint(data)
	if err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	return platformdb.IntegrationOutbox{
		OutboxID:           uuidValue(id),
		EventType:          "TestEvent",
		EventVersion:       1,
		SourceContext:      "platform",
		AggregateID:        uuidValue(uuid.New()),
		AggregateVersion:   1,
		CorrelationID:      uuidValue(uuid.New()),
		CausationID:        uuidValue(uuid.New()),
		Payload:            data,
		PayloadFingerprint: fingerprint,
		AttemptCount:       attempt,
		OccurredAt:         timestamptzValue(time.Now().UTC()),
		DataClassification: string(events.Internal),
	}
}

type fixedDispatcherClock struct{ now time.Time }

func (c fixedDispatcherClock) Now() time.Time { return c.now }

func (fixedDispatcherClock) Sleep(ctx context.Context, _ time.Duration) error {
	<-ctx.Done()
	return ctx.Err()
}

var _ OutboxStore = (*dispatcherStore)(nil)
var _ Clock = fixedDispatcherClock{}
