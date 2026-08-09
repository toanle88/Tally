package aggregateversion

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"math"
	"strconv"
)

var (
	ErrInvalidVersion  = errors.New("invalid aggregate version")
	ErrVersionOverflow = errors.New("aggregate version overflow")
	ErrInvalidJSON     = errors.New("aggregate version JSON is invalid")
)

// AggregateVersion is the validated version of a mutable aggregate.
// Versions are one-based and advance monotonically.
type AggregateVersion int64

func Initial() AggregateVersion { return AggregateVersion(1) }

func FromInt64(value int64) (AggregateVersion, error) {
	if value < 1 {
		return 0, ErrInvalidVersion
	}
	return AggregateVersion(value), nil
}

func (v AggregateVersion) Value() int64 { return int64(v) }

func (v AggregateVersion) Matches(expected AggregateVersion) bool {
	return v.valid() && expected.valid() && v == expected
}

func (v AggregateVersion) Advance() (AggregateVersion, error) {
	if !v.valid() {
		return 0, ErrInvalidVersion
	}
	if v == AggregateVersion(math.MaxInt64) {
		return 0, ErrVersionOverflow
	}
	return v + 1, nil
}

func (v AggregateVersion) MarshalJSON() ([]byte, error) {
	if !v.valid() {
		return nil, ErrInvalidVersion
	}
	return []byte(strconv.FormatInt(int64(v), 10)), nil
}

func (v *AggregateVersion) UnmarshalJSON(data []byte) error {
	if v == nil {
		return ErrInvalidJSON
	}
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		return ErrInvalidJSON
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	var value int64
	if err := decoder.Decode(&value); err != nil {
		return ErrInvalidJSON
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return ErrInvalidJSON
	}
	parsed, err := FromInt64(value)
	if err != nil {
		return ErrInvalidJSON
	}
	*v = parsed
	return nil
}

func (v AggregateVersion) valid() bool { return v >= 1 }
