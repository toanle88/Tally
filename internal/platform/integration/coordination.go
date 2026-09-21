package integration

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/toanle88/Tally/internal/platform/database/platformdb"
	"github.com/toanle88/Tally/internal/platform/events"
	"github.com/toanle88/Tally/internal/platform/telemetry"
	"go.opentelemetry.io/otel/trace"
)

var (
	ErrInvalidCoordinator      = errors.New("invalid integration coordinator")
	ErrInvalidDelivery         = errors.New("invalid integration delivery")
	ErrInvalidPublication      = errors.New("invalid integration publication")
	ErrCommitAmbiguous         = errors.New("integration transaction commit is ambiguous")
	ErrInboxNotFound           = errors.New("integration inbox record not found")
	ErrInboxStateChanged       = errors.New("integration inbox state changed before establishment")
	ErrEffectPanic             = errors.New("integration effect panicked")
	ErrIdentityContentConflict = errors.New("integration identity content conflict")
)

const failureRecordTimeout = 5 * time.Second

// TxBeginner is the narrow database boundary required by Coordinator.
type TxBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// ResultReference identifies the owning-context result established by a
// receiving effect. The coordinator treats it as opaque JSON bytes.
type ResultReference []byte

// Publication is an event that must be written to the outbox in the current
// transaction. AvailableAt is required so callers choose scheduling explicitly.
type Publication struct {
	Event       events.Envelope
	AvailableAt time.Time
}

// Delivery identifies one receiving-context observation of an event.
type Delivery struct {
	ConsumerName string
	Event        events.Envelope
}

// SourceEffect applies the owning source-context business effect using the
// transaction that will also receive the outbox record.
type SourceEffect func(context.Context, pgx.Tx) error

// ConsumerEffect applies the receiving-context local effect and returns the
// result reference and any integration publications created by that effect.
type ConsumerEffect func(context.Context, pgx.Tx) (ResultReference, []Publication, error)

// LocalResultLookup checks an owning-context result while holding the inbox
// row transaction lock. It must not create a business effect.
type LocalResultLookup func(context.Context, pgx.Tx) (ResultReference, bool, error)

type OutcomeState string

const (
	OutcomeEstablished   OutcomeState = "established"
	OutcomeProcessing    OutcomeState = "processing"
	OutcomeFailed        OutcomeState = "failed"
	OutcomeRetryEligible OutcomeState = "retry_eligible"
)

// Outcome is the durable coordination result. ResultReference is copied on
// construction and read so callers cannot mutate stored evidence accidentally.
type Outcome struct {
	State           OutcomeState
	resultReference ResultReference
}

func (o Outcome) Result() ResultReference {
	return cloneResultReference(o.resultReference)
}

func NewOutcome(state OutcomeState, result ResultReference) Outcome {
	return Outcome{State: state, resultReference: cloneResultReference(result)}
}

// IdentityContentConflict reports a changed fingerprint for an existing
// consumer/message identity. Audit and integrity policy remain caller-owned.
type IdentityContentConflict struct {
	ConsumerName        string
	MessageID           string
	StoredFingerprint   string
	ReceivedFingerprint string
}

func (e IdentityContentConflict) Error() string {
	return fmt.Sprintf(
		"%v: consumer=%s message_id=%s stored_fingerprint=%s received_fingerprint=%s",
		ErrIdentityContentConflict,
		e.ConsumerName,
		e.MessageID,
		e.StoredFingerprint,
		e.ReceivedFingerprint,
	)
}

func (e IdentityContentConflict) Unwrap() error { return ErrIdentityContentConflict }

type Coordinator struct {
	db              TxBeginner
	instrumentation *telemetry.Instrumentation
}

func NewCoordinator(db TxBeginner) *Coordinator {
	return NewCoordinatorWithInstrumentation(db, nil)
}

func NewCoordinatorWithInstrumentation(db TxBeginner, instrumentation *telemetry.Instrumentation) *Coordinator {
	return &Coordinator{db: db, instrumentation: instrumentation}
}

// Publish commits the source effect and its outbox record together.
func (c *Coordinator) Publish(ctx context.Context, publication Publication, effect SourceEffect) (err error) {
	if err := c.validate(ctx); err != nil {
		return err
	}
	if err := validatePublication(publication); err != nil {
		return err
	}
	if effect == nil {
		return ErrInvalidCoordinator
	}
	workCtx, operationSpan, operationStarted := c.startSpan(ctx, "outbox.publication", telemetry.SpanAttributes{
		Module:    "platform.integration",
		Operation: "outbox_publication",
	})
	defer func() {
		c.finishSpan(operationSpan, operationStarted, "platform.integration", "outbox_publication", err)
	}()
	workCtx, err = withEventContext(workCtx, publication.Event)
	if err != nil {
		return fmt.Errorf("establish publication context: %w", err)
	}

	txContext, transactionSpan, transactionStarted := c.startSpan(workCtx, "postgres.transaction", telemetry.SpanAttributes{
		Module:    "platform.database",
		Operation: "postgres_transaction",
	})
	defer func() {
		c.finishTransactionSpan(transactionSpan, transactionStarted, txContext, "platform.integration", "outbox_publication", err)
	}()
	tx, err := c.db.BeginTx(txContext, pgx.TxOptions{})
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(txContext)
		}
	}()

	if err := callSourceEffect(txContext, tx, effect); err != nil {
		return err
	}
	repositoryContext, repositorySpan, repositoryStarted := c.startSpan(txContext, "repository.operation", telemetry.SpanAttributes{
		Module:    "platform.integration",
		Operation: "outbox_insert",
	})
	insertErr := insertPublication(repositoryContext, tx, publication)
	c.finishSpan(repositorySpan, repositoryStarted, "platform.integration", "outbox_insert", insertErr)
	if insertErr != nil {
		return insertErr
	}
	if err := tx.Commit(txContext); err != nil {
		return fmt.Errorf("%w: %v", ErrCommitAmbiguous, err)
	}
	committed = true
	return nil
}

// Consume establishes a new delivery, or returns the durable state of an
// existing delivery without invoking the effect for processing/failed rows.
func (c *Coordinator) Consume(ctx context.Context, delivery Delivery, effect ConsumerEffect) (outcome Outcome, err error) {
	if err := c.validate(ctx); err != nil {
		return Outcome{}, err
	}
	if err := validateDelivery(delivery); err != nil {
		return Outcome{}, err
	}
	if effect == nil {
		return Outcome{}, ErrInvalidCoordinator
	}
	workCtx, operationSpan, operationStarted := c.startSpan(ctx, "inbox.handling", telemetry.SpanAttributes{
		Module:    "platform.integration",
		Operation: "inbox_handling",
		Consumer:  delivery.ConsumerName,
	})
	defer func() { c.finishSpan(operationSpan, operationStarted, "platform.integration", "inbox_handling", err) }()
	workCtx, err = withEventContext(workCtx, delivery.Event)
	if err != nil {
		return Outcome{}, fmt.Errorf("establish delivery context: %w", err)
	}

	txContext, transactionSpan, transactionStarted := c.startSpan(workCtx, "postgres.transaction", telemetry.SpanAttributes{
		Module:    "platform.database",
		Operation: "postgres_transaction",
		Consumer:  delivery.ConsumerName,
	})
	defer func() {
		c.finishTransactionSpan(transactionSpan, transactionStarted, txContext, "platform.integration", "inbox_handling", err)
	}()
	tx, err := c.db.BeginTx(txContext, pgx.TxOptions{})
	if err != nil {
		return Outcome{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(txContext)
		}
	}()

	queries := platformdb.New(tx)
	repositoryContext, repositorySpan, repositoryStarted := c.startSpan(txContext, "repository.operation", telemetry.SpanAttributes{
		Module:    "platform.integration",
		Operation: "inbox_insert",
		Consumer:  delivery.ConsumerName,
	})
	inbox, insertErr := queries.InsertInboxIfAbsent(repositoryContext, inboxInsertParams(delivery))
	c.finishSpan(repositorySpan, repositoryStarted, "platform.integration", "inbox_insert", insertErr)
	if errors.Is(insertErr, pgx.ErrNoRows) {
		repositoryContext, repositorySpan, repositoryStarted = c.startSpan(txContext, "repository.operation", telemetry.SpanAttributes{
			Module:    "platform.integration",
			Operation: "inbox_lookup",
			Consumer:  delivery.ConsumerName,
		})
		inbox, err = queries.GetInboxForUpdate(repositoryContext, inboxIdentityParams(delivery))
		c.finishSpan(repositorySpan, repositoryStarted, "platform.integration", "inbox_lookup", err)
		if err != nil {
			return Outcome{}, err
		}
		if err := validateInboxFingerprint(delivery, inbox); err != nil {
			return Outcome{}, err
		}
		return outcomeForInbox(inbox)
	}
	if insertErr != nil {
		return Outcome{}, insertErr
	}

	result, publications, effectErr := callConsumerEffect(txContext, tx, effect)
	if effectErr != nil {
		_ = rollbackTransaction(tx, txContext)
		return Outcome{State: OutcomeFailed}, c.recordFailure(txContext, delivery, effectErr)
	}
	if err := insertPublications(txContext, tx, publications); err != nil {
		_ = rollbackTransaction(tx, txContext)
		return Outcome{State: OutcomeFailed}, c.recordFailure(txContext, delivery, err)
	}
	if err := establishInbox(txContext, tx, delivery, result); err != nil {
		_ = rollbackTransaction(tx, txContext)
		return Outcome{State: OutcomeFailed}, c.recordFailure(txContext, delivery, err)
	}
	if err := tx.Commit(txContext); err != nil {
		return Outcome{}, fmt.Errorf("%w: %v", ErrCommitAmbiguous, err)
	}
	committed = true
	return NewOutcome(OutcomeEstablished, result), nil
}

// ReconcileAndRetry checks for an already-established local result before
// retrying processing/failed inbox work. A found local result establishes the
// inbox without replaying the effect.
func (c *Coordinator) ReconcileAndRetry(
	ctx context.Context,
	delivery Delivery,
	lookup LocalResultLookup,
	effect ConsumerEffect,
) (outcome Outcome, err error) {
	if err := c.validate(ctx); err != nil {
		return Outcome{}, err
	}
	if err := validateDelivery(delivery); err != nil {
		return Outcome{}, err
	}
	if lookup == nil || effect == nil {
		return Outcome{}, ErrInvalidCoordinator
	}
	workCtx, operationSpan, operationStarted := c.startSpan(ctx, "recovery.action", telemetry.SpanAttributes{
		Module:    "platform.integration",
		Operation: "inbox_reconcile",
		Consumer:  delivery.ConsumerName,
	})
	defer func() { c.finishSpan(operationSpan, operationStarted, "platform.integration", "inbox_reconcile", err) }()
	workCtx, err = withEventContext(workCtx, delivery.Event)
	if err != nil {
		return Outcome{}, fmt.Errorf("establish delivery context: %w", err)
	}

	txContext, transactionSpan, transactionStarted := c.startSpan(workCtx, "postgres.transaction", telemetry.SpanAttributes{
		Module:    "platform.database",
		Operation: "postgres_transaction",
		Consumer:  delivery.ConsumerName,
	})
	defer func() {
		c.finishTransactionSpan(transactionSpan, transactionStarted, txContext, "platform.integration", "inbox_reconcile", err)
	}()
	tx, err := c.db.BeginTx(txContext, pgx.TxOptions{})
	if err != nil {
		return Outcome{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback(txContext)
		}
	}()

	queries := platformdb.New(tx)
	repositoryContext, repositorySpan, repositoryStarted := c.startSpan(txContext, "repository.operation", telemetry.SpanAttributes{
		Module:    "platform.integration",
		Operation: "inbox_lookup",
		Consumer:  delivery.ConsumerName,
	})
	inbox, err := queries.GetInboxForUpdate(repositoryContext, inboxIdentityParams(delivery))
	c.finishSpan(repositorySpan, repositoryStarted, "platform.integration", "inbox_lookup", err)
	if errors.Is(err, pgx.ErrNoRows) {
		return Outcome{}, ErrInboxNotFound
	}
	if err != nil {
		return Outcome{}, err
	}
	if err := validateInboxFingerprint(delivery, inbox); err != nil {
		return Outcome{}, err
	}
	if inbox.State == string(OutcomeEstablished) {
		return outcomeForInbox(inbox)
	}
	if inbox.State != string(OutcomeProcessing) && inbox.State != string(OutcomeFailed) {
		return Outcome{}, fmt.Errorf("%w: state=%s", ErrInvalidCoordinator, inbox.State)
	}

	result, found, err := lookup(txContext, tx)
	if err != nil {
		return Outcome{}, err
	}
	if found {
		if err := establishInbox(txContext, tx, delivery, result); err != nil {
			return Outcome{}, err
		}
		if err := tx.Commit(txContext); err != nil {
			return Outcome{}, fmt.Errorf("%w: %v", ErrCommitAmbiguous, err)
		}
		committed = true
		return NewOutcome(OutcomeEstablished, result), nil
	}

	result, publications, effectErr := callConsumerEffect(txContext, tx, effect)
	if effectErr != nil {
		_ = rollbackTransaction(tx, txContext)
		return Outcome{State: OutcomeFailed}, c.recordFailure(txContext, delivery, effectErr)
	}
	if err := insertPublications(txContext, tx, publications); err != nil {
		_ = rollbackTransaction(tx, txContext)
		return Outcome{State: OutcomeFailed}, c.recordFailure(txContext, delivery, err)
	}
	if err := establishInbox(txContext, tx, delivery, result); err != nil {
		_ = rollbackTransaction(tx, txContext)
		return Outcome{State: OutcomeFailed}, c.recordFailure(txContext, delivery, err)
	}
	if err := tx.Commit(txContext); err != nil {
		return Outcome{}, fmt.Errorf("%w: %v", ErrCommitAmbiguous, err)
	}
	committed = true
	return NewOutcome(OutcomeEstablished, result), nil
}

func (c *Coordinator) validate(ctx context.Context) error {
	if c == nil || c.db == nil || ctx == nil {
		return ErrInvalidCoordinator
	}
	return nil
}

func (c *Coordinator) startSpan(ctx context.Context, name string, fields telemetry.SpanAttributes) (context.Context, trace.Span, time.Time) {
	if c == nil || c.instrumentation == nil {
		return ctx, nil, time.Time{}
	}
	workCtx, span := c.instrumentation.StartSpan(ctx, name, fields)
	return workCtx, span, time.Now()
}

func (c *Coordinator) finishSpan(span trace.Span, started time.Time, module, operation string, err error) {
	if c == nil || c.instrumentation == nil || span == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "failure"
	}
	c.instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{
		Module:    module,
		Operation: operation,
		Result:    result,
	})
	span.End()
}

func (c *Coordinator) finishTransactionSpan(span trace.Span, started time.Time, ctx context.Context, operationModule, operation string, err error) {
	if c == nil || c.instrumentation == nil || span == nil {
		return
	}
	result := "success"
	if err != nil {
		result = "failure"
	}
	c.instrumentation.SetSpanAttributes(span, telemetry.SpanAttributes{
		Module:    "platform.database",
		Operation: "postgres_transaction",
		Result:    result,
	})
	c.instrumentation.ObserveDBTransaction(ctx, time.Since(started), operationModule, operation, result)
	span.End()
}

func (c *Coordinator) recordFailure(ctx context.Context, delivery Delivery, cause error) (returnErr error) {
	recoveryCtx, cancel := context.WithTimeout(telemetry.Detach(ctx), failureRecordTimeout)
	defer cancel()
	recoveryCtx, recoverySpan, recoveryStarted := c.startSpan(recoveryCtx, "recovery.action", telemetry.SpanAttributes{
		Module:       "platform.integration",
		Operation:    "inbox_failure_record",
		Consumer:     delivery.ConsumerName,
		FailureClass: "inbox_processing_failed",
	})
	defer func() {
		c.finishSpan(recoverySpan, recoveryStarted, "platform.integration", "inbox_failure_record", returnErr)
	}()

	tx, err := c.db.BeginTx(recoveryCtx, pgx.TxOptions{})
	if err != nil {
		return errors.Join(cause, err)
	}
	rows, err := platformdb.New(tx).RecordInboxFailure(recoveryCtx, failureParams(delivery))
	if err != nil {
		_ = tx.Rollback(recoveryCtx)
		return errors.Join(cause, err)
	}
	if err := tx.Commit(recoveryCtx); err != nil {
		return errors.Join(cause, fmt.Errorf("%w: %v", ErrCommitAmbiguous, err))
	}
	if rows == 1 && c.instrumentation != nil {
		c.instrumentation.AddInboxFailure(recoveryCtx, 1, delivery.ConsumerName, "inbox_processing_failed")
	}
	return cause
}

func rollbackTransaction(tx pgx.Tx, ctx context.Context) error {
	if ctx.Err() == nil {
		return tx.Rollback(ctx)
	}
	rollbackCtx, cancel := context.WithTimeout(context.Background(), failureRecordTimeout)
	defer cancel()
	return tx.Rollback(rollbackCtx)
}

func callSourceEffect(ctx context.Context, tx pgx.Tx, effect SourceEffect) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: %v", ErrEffectPanic, recovered)
		}
	}()
	return effect(ctx, tx)
}

func callConsumerEffect(ctx context.Context, tx pgx.Tx, effect ConsumerEffect) (result ResultReference, publications []Publication, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("%w: %v", ErrEffectPanic, recovered)
		}
	}()
	return effect(ctx, tx)
}

func insertPublications(ctx context.Context, tx pgx.Tx, publications []Publication) error {
	for _, publication := range publications {
		if err := validatePublication(publication); err != nil {
			return err
		}
		if err := insertPublication(ctx, tx, publication); err != nil {
			return err
		}
	}
	return nil
}

func insertPublication(ctx context.Context, tx pgx.Tx, publication Publication) error {
	event := publication.Event
	messageID, _ := uuid.Parse(event.MessageID())
	aggregateID, _ := uuid.Parse(event.AggregateID())
	correlationID, _ := uuid.Parse(event.CorrelationID())
	causationID, _ := uuid.Parse(event.CausationID())
	accountingScopeID := pgtype.UUID{}
	if event.AccountingScopeID() != "" {
		scopeID, _ := uuid.Parse(event.AccountingScopeID())
		accountingScopeID = uuidValue(scopeID)
	}
	_, err := platformdb.New(tx).InsertOutbox(ctx, platformdb.InsertOutboxParams{
		OutboxID:           uuidValue(messageID),
		EventType:          event.EventType(),
		EventVersion:       int32(event.EventVersion()),
		OccurredAt:         timestamptzValue(event.OccurredAt()),
		SourceContext:      event.SourceContext(),
		AggregateID:        uuidValue(aggregateID),
		AggregateVersion:   event.AggregateVersion(),
		AccountingScopeID:  accountingScopeID,
		CorrelationID:      uuidValue(correlationID),
		CausationID:        uuidValue(causationID),
		Payload:            event.Data(),
		PayloadFingerprint: event.PayloadFingerprint(),
		DataClassification: string(event.DataClassification()),
		AvailableAt:        timestamptzValue(publication.AvailableAt),
	})
	return err
}

func establishInbox(ctx context.Context, tx pgx.Tx, delivery Delivery, result ResultReference) error {
	rows, err := platformdb.New(tx).EstablishInbox(ctx, platformdb.EstablishInboxParams{
		ConsumerName:    delivery.ConsumerName,
		MessageID:       uuidValue(mustUUID(delivery.Event.MessageID())),
		ResultReference: cloneResultReference(result),
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return ErrInboxStateChanged
	}
	return nil
}

func validatePublication(publication Publication) error {
	if publication.AvailableAt.IsZero() {
		return fmt.Errorf("%w: available_at is required", ErrInvalidPublication)
	}
	if result := publication.Event.Validate(); !result.Valid() {
		return fmt.Errorf("%w: %v", ErrInvalidPublication, result.Err())
	}
	return nil
}

func validateDelivery(delivery Delivery) error {
	if !validIdentifier(delivery.ConsumerName) {
		return fmt.Errorf("%w: consumer name is invalid", ErrInvalidDelivery)
	}
	if result := delivery.Event.Validate(); !result.Valid() {
		return fmt.Errorf("%w: %v", ErrInvalidDelivery, result.Err())
	}
	return nil
}

func validateInboxFingerprint(delivery Delivery, inbox platformdb.IntegrationInbox) error {
	if inbox.MessageFingerprint != delivery.Event.PayloadFingerprint() {
		return IdentityContentConflict{
			ConsumerName:        delivery.ConsumerName,
			MessageID:           delivery.Event.MessageID(),
			StoredFingerprint:   inbox.MessageFingerprint,
			ReceivedFingerprint: delivery.Event.PayloadFingerprint(),
		}
	}
	return nil
}

func outcomeForInbox(inbox platformdb.IntegrationInbox) (Outcome, error) {
	switch OutcomeState(inbox.State) {
	case OutcomeEstablished:
		return NewOutcome(OutcomeEstablished, ResultReference(inbox.ResultReference)), nil
	case OutcomeProcessing:
		return Outcome{State: OutcomeProcessing}, nil
	case OutcomeFailed:
		return Outcome{State: OutcomeFailed}, nil
	default:
		return Outcome{}, fmt.Errorf("%w: state=%s", ErrInvalidCoordinator, inbox.State)
	}
}

func inboxInsertParams(delivery Delivery) platformdb.InsertInboxIfAbsentParams {
	return platformdb.InsertInboxIfAbsentParams{
		ConsumerName:       delivery.ConsumerName,
		MessageID:          uuidValue(mustUUID(delivery.Event.MessageID())),
		MessageFingerprint: delivery.Event.PayloadFingerprint(),
	}
}

func inboxIdentityParams(delivery Delivery) platformdb.GetInboxForUpdateParams {
	return platformdb.GetInboxForUpdateParams{
		ConsumerName: delivery.ConsumerName,
		MessageID:    uuidValue(mustUUID(delivery.Event.MessageID())),
	}
}

func failureParams(delivery Delivery) platformdb.RecordInboxFailureParams {
	return platformdb.RecordInboxFailureParams{
		ConsumerName:       delivery.ConsumerName,
		MessageID:          uuidValue(mustUUID(delivery.Event.MessageID())),
		MessageFingerprint: delivery.Event.PayloadFingerprint(),
	}
}

func validIdentifier(value string) bool {
	return value != "" && utf8.ValidString(value) && strings.TrimSpace(value) == value && !strings.ContainsFunc(value, unicode.IsControl)
}

func mustUUID(value string) uuid.UUID {
	parsed, _ := uuid.Parse(value)
	return parsed
}

func uuidValue(value uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: [16]byte(value), Valid: true}
}

func timestamptzValue(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value.UTC(), Valid: true}
}

func cloneResultReference(value ResultReference) ResultReference {
	return ResultReference(append([]byte(nil), value...))
}
