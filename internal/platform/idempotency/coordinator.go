package idempotency

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
)

var (
	// ErrFingerprintMismatch is retained for compatibility with the original
	// coordination contract. ErrIdempotencyConflict is its stable semantic
	// alias for transport and application callers.
	ErrFingerprintMismatch  = errors.New("idempotency fingerprint mismatch")
	ErrIdempotencyConflict  = ErrFingerprintMismatch
	ErrInvalidAcquisition   = errors.New("invalid idempotency acquisition")
	ErrAlreadyFinalized     = errors.New("idempotency acquisition already finalized")
	ErrFinalizationProgress = errors.New("idempotency finalization already in progress")
	ErrInvalidTerminalState = errors.New("invalid idempotency terminal state")
	ErrCommitPanic          = errors.New("idempotency commit panicked")
)

type AcquisitionDecision string

const (
	DecisionExecute AcquisitionDecision = "execute"
	DecisionReturn  AcquisitionDecision = "return"
)

// Acquisition is the result of claiming or observing one scoped command.
// Its fields are private so callers can only finalize a coordinator-issued claim.
type Acquisition struct {
	decision    AcquisitionDecision
	token       string
	result      CommandResultMetadata
	identity    IdempotencyIdentity
	fingerprint Fingerprint
}

func (a Acquisition) Decision() AcquisitionDecision { return a.decision }
func (a Acquisition) Token() string                 { return a.token }
func (a Acquisition) Result() CommandResultMetadata { return cloneMetadata(a.result) }

// Coordinator coordinates one in-memory execution owner and established results.
// Production persistence and transaction ownership remain with the bounded context.
type Coordinator interface {
	Acquire(identity IdempotencyIdentity, fingerprint Fingerprint, operationID string) (Acquisition, error)
	Finalize(acquisition Acquisition, commit func(CommandResultMetadata) (CommandResultMetadata, error)) error
}

type memoryEntry struct {
	identity    IdempotencyIdentity
	fingerprint Fingerprint
	operationID string
	result      CommandResultMetadata
	token       string
	finalizing  bool
	terminal    bool
}

// MemoryCoordinator is a concurrency-safe coordination-contract test double.
type MemoryCoordinator struct {
	mu      sync.Mutex
	entries map[IdempotencyIdentity]*memoryEntry
}

func NewMemoryCoordinator() *MemoryCoordinator {
	return &MemoryCoordinator{entries: make(map[IdempotencyIdentity]*memoryEntry)}
}

func (c *MemoryCoordinator) Acquire(identity IdempotencyIdentity, fingerprint Fingerprint, operationID string) (Acquisition, error) {
	if c == nil {
		return Acquisition{}, ErrInvalidAcquisition
	}
	if _, err := NewCommandResultMetadata(identity, fingerprint, operationID, StateInProgress, nil, nil, nil, nil); err != nil {
		return Acquisition{}, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if entry, ok := c.entries[identity]; ok {
		if entry.fingerprint != fingerprint {
			return Acquisition{}, ErrFingerprintMismatch
		}
		return Acquisition{
			decision: DecisionReturn, result: cloneMetadata(entry.result), identity: identity, fingerprint: fingerprint,
		}, nil
	}

	result, err := NewCommandResultMetadata(identity, fingerprint, operationID, StateInProgress, nil, nil, nil, nil)
	if err != nil {
		return Acquisition{}, err
	}
	token := uuid.NewString()
	c.entries[identity] = &memoryEntry{
		identity: identity, fingerprint: fingerprint, operationID: operationID,
		result: result, token: token,
	}
	return Acquisition{
		decision: DecisionExecute, token: token, result: cloneMetadata(result), identity: identity, fingerprint: fingerprint,
	}, nil
}

func (c *MemoryCoordinator) Finalize(acquisition Acquisition, commit func(CommandResultMetadata) (CommandResultMetadata, error)) (err error) {
	if c == nil || acquisition.decision != DecisionExecute || acquisition.token == "" || commit == nil {
		return ErrInvalidAcquisition
	}

	c.mu.Lock()
	entry, ok := c.entries[acquisition.identity]
	if !ok || entry.token != acquisition.token || entry.fingerprint != acquisition.fingerprint {
		c.mu.Unlock()
		return ErrInvalidAcquisition
	}
	if entry.terminal {
		c.mu.Unlock()
		return ErrAlreadyFinalized
	}
	if entry.finalizing {
		c.mu.Unlock()
		return ErrFinalizationProgress
	}
	entry.finalizing = true
	result := cloneMetadata(entry.result)
	c.mu.Unlock()

	var callbackErr error
	var finalResult CommandResultMetadata
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				callbackErr = fmt.Errorf("%w: %v", ErrCommitPanic, recovered)
			}
		}()
		finalResult, callbackErr = commit(result)
	}()

	c.mu.Lock()
	defer c.mu.Unlock()
	if callbackErr != nil {
		entry.finalizing = false
		return callbackErr
	}
	if finalResult.State() != StateEstablished && finalResult.State() != StateFailed {
		entry.finalizing = false
		return ErrInvalidTerminalState
	}
	if !finalResult.Identity().Equal(entry.identity) || finalResult.Fingerprint() != entry.fingerprint || finalResult.OperationID() != entry.operationID {
		entry.finalizing = false
		return ErrInvalidTerminalState
	}
	entry.result = cloneMetadata(finalResult)
	entry.finalizing = false
	entry.terminal = true
	return nil
}

func cloneMetadata(value CommandResultMetadata) CommandResultMetadata {
	return CommandResultMetadata{
		identity: value.identity, fingerprint: value.fingerprint, operationID: value.operationID,
		state: value.state, resultStatus: cloneInt(value.resultStatus), resultBody: cloneBytes(value.resultBody),
		aggregateID: cloneUUID(value.aggregateID), processID: cloneUUID(value.processID),
	}
}
