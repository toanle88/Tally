package idempotency

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var (
	ErrInvalidIdempotencyPolicy = errors.New("invalid idempotency policy")
	ErrLeaseExpired             = errors.New("idempotency lease expired")
	ErrLeaseOwned               = errors.New("idempotency lease is owned by another execution")
)

// IdempotencyPolicy controls reservation ownership and retention metadata.
// The database clock is used for persisted lease and expiry comparisons.
type IdempotencyPolicy struct {
	RecordTTL time.Duration
	LeaseTTL  time.Duration
}

func (p IdempotencyPolicy) validate() error {
	if p.RecordTTL <= 0 || p.LeaseTTL <= 0 || p.LeaseTTL >= p.RecordTTL {
		return ErrInvalidIdempotencyPolicy
	}
	return nil
}

// TxBeginner is the transaction boundary used by durable acquisition.
type TxBeginner interface {
	BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
}

// DurableAcquisition is a database-backed ownership decision.
type DurableAcquisition struct {
	decision    AcquisitionDecision
	ownerToken  uuid.UUID
	result      CommandResultMetadata
	identity    IdempotencyIdentity
	fingerprint Fingerprint
	operationID string
}

func (a DurableAcquisition) Decision() AcquisitionDecision { return a.decision }
func (a DurableAcquisition) OwnerToken() uuid.UUID         { return a.ownerToken }
func (a DurableAcquisition) Result() CommandResultMetadata { return cloneMetadata(a.result) }

// DurableCoordinator reserves an identity in its own short transaction.
// The owning business transaction is supplied separately to Finalize.
type DurableCoordinator interface {
	Acquire(context.Context, TxBeginner, IdempotencyIdentity, Fingerprint, string, IdempotencyPolicy) (DurableAcquisition, error)
	Finalize(context.Context, pgx.Tx, DurableAcquisition, CommandResultMetadata) error
}

type PostgresCoordinator struct{}

func NewPostgresCoordinator() *PostgresCoordinator { return &PostgresCoordinator{} }

func (c *PostgresCoordinator) Acquire(
	ctx context.Context,
	db TxBeginner,
	identity IdempotencyIdentity,
	fingerprint Fingerprint,
	operationID string,
	policy IdempotencyPolicy,
) (DurableAcquisition, error) {
	if c == nil || db == nil || ctx == nil {
		return DurableAcquisition{}, ErrInvalidAcquisition
	}
	if err := policy.validate(); err != nil {
		return DurableAcquisition{}, err
	}
	if _, err := NewCommandResultMetadata(identity, fingerprint, operationID, StateInProgress, nil, nil, nil, nil); err != nil {
		return DurableAcquisition{}, err
	}
	scopeKey, err := identity.ScopeKey()
	if err != nil {
		return DurableAcquisition{}, err
	}
	tx, err := db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return DurableAcquisition{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	ownerToken := uuid.New()
	var insertedToken uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO platform.idempotency_record (
			scope_key, idempotency_key, canonical_fingerprint, operation_id,
			state, owner_token, lease_until, expires_at
		)
		VALUES ($1, $2, $3, $4, 'in_progress', $5,
			clock_timestamp() + $6::interval,
			clock_timestamp() + $7::interval)
		ON CONFLICT (scope_key, idempotency_key) DO NOTHING
		RETURNING owner_token
	`, scopeKey, identity.Key(), string(fingerprint), operationID, ownerToken,
		policy.LeaseTTL.String(), policy.RecordTTL.String()).Scan(&insertedToken)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return DurableAcquisition{}, err
		}
		return DurableAcquisition{
			decision: DecisionExecute, ownerToken: insertedToken,
			identity: identity, fingerprint: fingerprint, operationID: operationID,
		}, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return DurableAcquisition{}, err
	}

	var (
		storedFingerprint string
		storedOperationID string
		storedState       string
		resultStatus      *int
		resultBody        []byte
		aggregateID       *uuid.UUID
		processID         *uuid.UUID
		storedOwner       uuid.UUID
		leaseUntil        time.Time
		leaseExpired      bool
	)
	err = tx.QueryRow(ctx, `
		SELECT canonical_fingerprint, operation_id, state, result_status,
			result_body, aggregate_id, process_id, owner_token, lease_until,
			(clock_timestamp() >= lease_until)
		FROM platform.idempotency_record
		WHERE scope_key = $1 AND idempotency_key = $2
		FOR UPDATE
	`, scopeKey, identity.Key()).Scan(
		&storedFingerprint, &storedOperationID, &storedState, &resultStatus,
		&resultBody, &aggregateID, &processID, &storedOwner, &leaseUntil, &leaseExpired)
	if err != nil {
		return DurableAcquisition{}, err
	}
	if Fingerprint(storedFingerprint) != fingerprint {
		return DurableAcquisition{}, ErrIdempotencyConflict
	}
	if storedOperationID != operationID {
		return DurableAcquisition{}, ErrIdempotencyConflict
	}
	state, err := ParseResultState(storedState)
	if err != nil {
		return DurableAcquisition{}, err
	}
	result, err := NewCommandResultMetadata(identity, fingerprint, storedOperationID, state, resultStatus, resultBody, aggregateID, processID)
	if err != nil {
		return DurableAcquisition{}, err
	}
	if state == StateInProgress && leaseExpired {
		commandTag, err := tx.Exec(ctx, `
			UPDATE platform.idempotency_record
			SET owner_token = $3,
				lease_until = clock_timestamp() + $4::interval
			WHERE scope_key = $1 AND idempotency_key = $2
			  AND state = 'in_progress' AND lease_until <= clock_timestamp()
		`, scopeKey, identity.Key(), ownerToken, policy.LeaseTTL.String())
		if err != nil {
			return DurableAcquisition{}, err
		}
		if commandTag.RowsAffected() != 1 {
			return DurableAcquisition{}, ErrLeaseExpired
		}
		if err := tx.Commit(ctx); err != nil {
			return DurableAcquisition{}, err
		}
		return DurableAcquisition{
			decision: DecisionExecute, ownerToken: ownerToken,
			identity: identity, fingerprint: fingerprint, operationID: operationID,
		}, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return DurableAcquisition{}, err
	}
	return DurableAcquisition{
		decision: DecisionReturn, result: result,
		identity: identity, fingerprint: fingerprint, operationID: storedOperationID,
	}, nil
}

func (c *PostgresCoordinator) Finalize(ctx context.Context, tx pgx.Tx, acquisition DurableAcquisition, result CommandResultMetadata) error {
	if c == nil || tx == nil || ctx == nil || acquisition.decision != DecisionExecute || acquisition.ownerToken == uuid.Nil {
		return ErrInvalidAcquisition
	}
	if result.Identity().Equal(acquisition.identity) == false || result.Fingerprint() != acquisition.fingerprint || result.OperationID() != acquisition.operationID {
		return ErrInvalidTerminalState
	}
	if result.State() != StateEstablished && result.State() != StateFailed {
		return ErrInvalidTerminalState
	}
	scopeKey, err := acquisition.identity.ScopeKey()
	if err != nil {
		return err
	}
	commandTag, err := tx.Exec(ctx, `
		UPDATE platform.idempotency_record
		SET state = $4, result_status = $5, result_body = $6,
			aggregate_id = $7, process_id = $8
		WHERE scope_key = $1 AND idempotency_key = $2
		  AND owner_token = $3 AND state = 'in_progress'
	`, scopeKey, acquisition.identity.Key(), acquisition.ownerToken, result.State(),
		result.ResultStatus(), result.ResultBody(), result.AggregateID(), result.ProcessID())
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return fmt.Errorf("%w: owner token no longer owns record", ErrLeaseOwned)
	}
	return nil
}
