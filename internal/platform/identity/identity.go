package identity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/google/uuid"
)

var (
	ErrNilID       = errors.New("identity is nil")
	ErrMalformedID = errors.New("identity is malformed")
	ErrInvalidJSON = errors.New("identity JSON is invalid")
)

type AggregateID struct{ value uuid.UUID }
type CorrelationID struct{ value uuid.UUID }
type CausationID struct{ value uuid.UUID }

func NewAggregateID() (AggregateID, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return AggregateID{}, err
	}
	return AggregateID{value: id}, nil
}

func FromAggregateUUID(value uuid.UUID) (AggregateID, error) {
	if value == uuid.Nil {
		return AggregateID{}, ErrNilID
	}
	return AggregateID{value: value}, nil
}

func FromCorrelationUUID(value uuid.UUID) (CorrelationID, error) {
	if value == uuid.Nil {
		return CorrelationID{}, ErrNilID
	}
	return CorrelationID{value: value}, nil
}

func FromCausationUUID(value uuid.UUID) (CausationID, error) {
	if value == uuid.Nil {
		return CausationID{}, ErrNilID
	}
	return CausationID{value: value}, nil
}

func ParseAggregateID(value string) (AggregateID, error) {
	id, err := parse(value)
	if err != nil {
		return AggregateID{}, err
	}
	return AggregateID{value: id}, nil
}

func ParseCorrelationID(value string) (CorrelationID, error) {
	id, err := parse(value)
	if err != nil {
		return CorrelationID{}, err
	}
	return CorrelationID{value: id}, nil
}

func ParseCausationID(value string) (CausationID, error) {
	id, err := parse(value)
	if err != nil {
		return CausationID{}, err
	}
	return CausationID{value: id}, nil
}

func (id AggregateID) String() string   { return id.value.String() }
func (id CorrelationID) String() string { return id.value.String() }
func (id CausationID) String() string   { return id.value.String() }

func (id AggregateID) UUID() uuid.UUID   { return id.value }
func (id CorrelationID) UUID() uuid.UUID { return id.value }
func (id CausationID) UUID() uuid.UUID   { return id.value }

func (id AggregateID) Equal(other AggregateID) bool     { return id.value == other.value }
func (id CorrelationID) Equal(other CorrelationID) bool { return id.value == other.value }
func (id CausationID) Equal(other CausationID) bool     { return id.value == other.value }

func (id AggregateID) MarshalJSON() ([]byte, error)   { return marshal(id.value) }
func (id CorrelationID) MarshalJSON() ([]byte, error) { return marshal(id.value) }
func (id CausationID) MarshalJSON() ([]byte, error)   { return marshal(id.value) }

func (id *AggregateID) UnmarshalJSON(data []byte) error {
	value, err := unmarshal(data)
	if err != nil {
		return err
	}
	*id = AggregateID{value: value}
	return nil
}

func (id *CorrelationID) UnmarshalJSON(data []byte) error {
	value, err := unmarshal(data)
	if err != nil {
		return err
	}
	*id = CorrelationID{value: value}
	return nil
}

func (id *CausationID) UnmarshalJSON(data []byte) error {
	value, err := unmarshal(data)
	if err != nil {
		return err
	}
	*id = CausationID{value: value}
	return nil
}

func parse(value string) (uuid.UUID, error) {
	if value == "" {
		return uuid.Nil, ErrMalformedID
	}
	id, err := uuid.Parse(value)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", ErrMalformedID, err)
	}
	if id == uuid.Nil {
		return uuid.Nil, ErrNilID
	}
	return id, nil
}

func marshal(value uuid.UUID) ([]byte, error) {
	if value == uuid.Nil {
		return nil, ErrNilID
	}
	return json.Marshal(value.String())
}

func unmarshal(data []byte) (uuid.UUID, error) {
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return uuid.Nil, ErrInvalidJSON
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var value string
	if err := decoder.Decode(&value); err != nil {
		return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return uuid.Nil, fmt.Errorf("%w: trailing JSON", ErrInvalidJSON)
		}
		return uuid.Nil, fmt.Errorf("%w: %v", ErrInvalidJSON, err)
	}
	return parse(value)
}
