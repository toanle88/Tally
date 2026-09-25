package idempotency

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/accountingscope"
)

var (
	ErrInvalidIdempotencyKey = errors.New("invalid idempotency key")
	ErrInvalidOperationID    = errors.New("invalid operation ID")
	ErrInvalidScopeKey       = errors.New("invalid idempotency scope key")
	ErrInvalidState          = errors.New("invalid idempotency state")
	ErrInvalidResultBody     = errors.New("invalid result body")
	ErrInvalidResultMetadata = errors.New("invalid result metadata")
)

const maxIdentityTextBytes = 255

// IdempotencyIdentity identifies one business command within an accounting scope.
// Operation identity, aggregate versions, and event/correlation identities are separate.
type IdempotencyIdentity struct {
	scope    accountingscope.AccountingScope
	scopeKey string
	key      string
}

func NewIdentity(scope accountingscope.AccountingScope, key string) (IdempotencyIdentity, error) {
	if _, err := scope.MarshalJSON(); err != nil {
		return IdempotencyIdentity{}, err
	}
	if err := validateIdentityText(key); err != nil {
		return IdempotencyIdentity{}, err
	}
	scopeKey, err := scope.MarshalJSON()
	if err != nil {
		return IdempotencyIdentity{}, err
	}
	return IdempotencyIdentity{scope: scope, scopeKey: string(scopeKey), key: key}, nil
}

// NewOpaqueIdentity creates an idempotency identity for a bounded context
// whose commands are not scoped by an accounting scope. The scope key remains
// platform-owned JSON so all bounded contexts use the same durable table and
// reservation/finalization protocol.
func NewOpaqueIdentity(scopeKey, key string) (IdempotencyIdentity, error) {
	if err := validateScopeKey(scopeKey); err != nil {
		return IdempotencyIdentity{}, err
	}
	if err := validateIdentityText(key); err != nil {
		return IdempotencyIdentity{}, err
	}
	return IdempotencyIdentity{scopeKey: scopeKey, key: key}, nil
}

func (i IdempotencyIdentity) Scope() accountingscope.AccountingScope { return i.scope }
func (i IdempotencyIdentity) Key() string                            { return i.key }

func (i IdempotencyIdentity) Equal(other IdempotencyIdentity) bool {
	return i.scopeKey == other.scopeKey && i.key == other.key
}

// ScopeKey is the compact, validated JSON representation used by persistence adapters.
func (i IdempotencyIdentity) ScopeKey() (string, error) {
	if err := validateScopeKey(i.scopeKey); err != nil {
		return "", err
	}
	return i.scopeKey, nil
}

func validateScopeKey(value string) error {
	if value == "" || strings.TrimSpace(value) != value || !json.Valid([]byte(value)) {
		return ErrInvalidScopeKey
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal([]byte(value), &object); err != nil || object == nil {
		return ErrInvalidScopeKey
	}
	return nil
}

type ResultState string

const (
	StateInProgress  ResultState = "in_progress"
	StateEstablished ResultState = "established"
	StateFailed      ResultState = "failed"
)

func ParseResultState(value string) (ResultState, error) {
	state := ResultState(value)
	switch state {
	case StateInProgress, StateEstablished, StateFailed:
		return state, nil
	default:
		return "", ErrInvalidState
	}
}

// CommandResultMetadata contains coordination metadata and never owns finance state.
type CommandResultMetadata struct {
	identity     IdempotencyIdentity
	fingerprint  Fingerprint
	operationID  string
	state        ResultState
	resultStatus *int
	resultBody   []byte
	aggregateID  *uuid.UUID
	processID    *uuid.UUID
}

func NewCommandResultMetadata(identity IdempotencyIdentity, fingerprint Fingerprint, operationID string, state ResultState, resultStatus *int, resultBody []byte, aggregateID, processID *uuid.UUID) (CommandResultMetadata, error) {
	if _, err := identity.ScopeKey(); err != nil || identity.key == "" {
		return CommandResultMetadata{}, ErrInvalidResultMetadata
	}
	if fingerprint == "" {
		return CommandResultMetadata{}, ErrInvalidResultMetadata
	}
	if err := validateIdentityText(operationID); err != nil {
		return CommandResultMetadata{}, ErrInvalidOperationID
	}
	parsedState, err := ParseResultState(string(state))
	if err != nil {
		return CommandResultMetadata{}, err
	}
	if resultBody != nil && !json.Valid(resultBody) {
		return CommandResultMetadata{}, fmt.Errorf("%w: %w", ErrInvalidResultBody, ErrInvalidJSON)
	}
	if err := validateOptionalUUID(aggregateID); err != nil {
		return CommandResultMetadata{}, ErrInvalidResultMetadata
	}
	if err := validateOptionalUUID(processID); err != nil {
		return CommandResultMetadata{}, ErrInvalidResultMetadata
	}
	return CommandResultMetadata{
		identity: identity, fingerprint: fingerprint, operationID: operationID,
		state: parsedState, resultStatus: cloneInt(resultStatus), resultBody: cloneBytes(resultBody),
		aggregateID: cloneUUID(aggregateID), processID: cloneUUID(processID),
	}, nil
}

func (m CommandResultMetadata) Identity() IdempotencyIdentity { return m.identity }
func (m CommandResultMetadata) Fingerprint() Fingerprint      { return m.fingerprint }
func (m CommandResultMetadata) OperationID() string           { return m.operationID }
func (m CommandResultMetadata) State() ResultState            { return m.state }
func (m CommandResultMetadata) ResultStatus() *int            { return cloneInt(m.resultStatus) }
func (m CommandResultMetadata) ResultBody() []byte            { return cloneBytes(m.resultBody) }
func (m CommandResultMetadata) AggregateID() *uuid.UUID       { return cloneUUID(m.aggregateID) }
func (m CommandResultMetadata) ProcessID() *uuid.UUID         { return cloneUUID(m.processID) }

func validateIdentityText(value string) error {
	if value == "" || !utf8.ValidString(value) || len(value) > maxIdentityTextBytes {
		return ErrInvalidIdempotencyKey
	}
	if strings.TrimSpace(value) != value || strings.TrimSpace(value) == "" {
		return ErrInvalidIdempotencyKey
	}
	for _, r := range value {
		if unicode.IsControl(r) {
			return ErrInvalidIdempotencyKey
		}
	}
	return nil
}

func validateOptionalUUID(value *uuid.UUID) error {
	if value != nil && *value == uuid.Nil {
		return ErrInvalidResultMetadata
	}
	return nil
}

func cloneBytes(value []byte) []byte {
	if value == nil {
		return nil
	}
	return append([]byte(nil), value...)
}

func cloneInt(value *int) *int {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneUUID(value *uuid.UUID) *uuid.UUID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
