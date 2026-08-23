package events

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/idempotency"
)

type Classification string

const (
	Public           Classification = "public"
	Internal         Classification = "internal"
	Confidential     Classification = "confidential"
	HighlyRestricted Classification = "highly_restricted"
)

type ValidationCode string

const (
	CodeValid                   ValidationCode = "valid"
	CodeMalformedIdentity       ValidationCode = "malformed_identity"
	CodeInvalidAggregateVersion ValidationCode = "invalid_aggregate_version"
	CodeInvalidScopeIdentifier  ValidationCode = "invalid_scope_identifier"
	CodeInvalidClassification   ValidationCode = "invalid_classification"
	CodeUnsupportedEventVersion ValidationCode = "unsupported_event_version"
	CodeInvalidPayload          ValidationCode = "invalid_payload"
	CodeFingerprintMismatch     ValidationCode = "fingerprint_mismatch"
	CodeInvalidOccurredAt       ValidationCode = "invalid_occurred_at"
)

var ErrFingerprintMismatch = errors.New("event payload fingerprint mismatch")

type EnvelopeInput struct {
	MessageID, EventType                          string
	EventVersion                                  int
	OccurredAt                                    time.Time
	SourceContext, AggregateID                    string
	AggregateVersion                              int64
	AccountingScopeID, CorrelationID, CausationID string
	DataClassification                            Classification
	PayloadFingerprint                            string
	Data                                          []byte
}

type Envelope struct{ input EnvelopeInput }

func NewEnvelope(in EnvelopeInput) (Envelope, error) {
	in.OccurredAt = in.OccurredAt.UTC()
	canonical, err := CanonicalizePayload(in.Data)
	if err != nil {
		return Envelope{}, err
	}
	in.Data = canonical
	e := Envelope{input: in}
	if r := e.Validate(); !r.Valid() {
		return Envelope{}, r.Err()
	}
	return e, nil
}

func (e Envelope) MessageID() string                  { return e.input.MessageID }
func (e Envelope) EventType() string                  { return e.input.EventType }
func (e Envelope) EventVersion() int                  { return e.input.EventVersion }
func (e Envelope) OccurredAt() time.Time              { return e.input.OccurredAt }
func (e Envelope) SourceContext() string              { return e.input.SourceContext }
func (e Envelope) AggregateID() string                { return e.input.AggregateID }
func (e Envelope) AggregateVersion() int64            { return e.input.AggregateVersion }
func (e Envelope) AccountingScopeID() string          { return e.input.AccountingScopeID }
func (e Envelope) CorrelationID() string              { return e.input.CorrelationID }
func (e Envelope) CausationID() string                { return e.input.CausationID }
func (e Envelope) DataClassification() Classification { return e.input.DataClassification }
func (e Envelope) PayloadFingerprint() string         { return e.input.PayloadFingerprint }
func (e Envelope) Data() []byte                       { return bytes.Clone(e.input.Data) }

type ValidationResult struct {
	code ValidationCode
	err  error
}

func (r ValidationResult) Valid() bool                     { return r.code == CodeValid }
func (r ValidationResult) Code() ValidationCode            { return r.code }
func (r ValidationResult) Err() error                      { return r.err }
func valid() ValidationResult                              { return ValidationResult{code: CodeValid} }
func invalid(c ValidationCode, err error) ValidationResult { return ValidationResult{c, err} }

func (e Envelope) Validate() ValidationResult {
	if e.input.OccurredAt.IsZero() {
		return invalid(CodeInvalidOccurredAt, errors.New("occurredAt is required"))
	}
	for _, v := range []string{e.input.MessageID, e.input.AggregateID, e.input.CorrelationID, e.input.CausationID} {
		if _, err := uuid.Parse(v); err != nil {
			return invalid(CodeMalformedIdentity, err)
		}
	}
	for _, v := range []string{e.input.EventType, e.input.SourceContext} {
		if !identifier(v) {
			return invalid(CodeMalformedIdentity, errors.New("invalid identifier text"))
		}
	}
	if e.input.EventVersion != 1 {
		return invalid(CodeUnsupportedEventVersion, errors.New("unsupported event version"))
	}
	if e.input.AggregateVersion < 1 {
		return invalid(CodeInvalidAggregateVersion, errors.New("aggregate version must be positive"))
	}
	if e.input.AccountingScopeID != "" {
		if _, err := uuid.Parse(e.input.AccountingScopeID); err != nil {
			return invalid(CodeInvalidScopeIdentifier, err)
		}
	}
	switch e.input.DataClassification {
	case Public, Internal, Confidential, HighlyRestricted:
	default:
		return invalid(CodeInvalidClassification, errors.New("invalid data classification"))
	}
	canonical, err := CanonicalizePayload(e.input.Data)
	if err != nil {
		return invalid(CodeInvalidPayload, err)
	}
	fp, err := ComputePayloadFingerprint(canonical)
	if err != nil {
		return invalid(CodeInvalidPayload, err)
	}
	if e.input.PayloadFingerprint != fp {
		return invalid(CodeFingerprintMismatch, ErrFingerprintMismatch)
	}
	return valid()
}

func identifier(s string) bool {
	return s != "" && utf8.ValidString(s) && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}
func CanonicalizePayload(b []byte) ([]byte, error) { return idempotency.Canonicalize(b) }
func ComputePayloadFingerprint(b []byte) (string, error) {
	c, err := CanonicalizePayload(b)
	if err != nil {
		return "", err
	}
	h := sha256.Sum256(c)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

func (e Envelope) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		MessageID          string          `json:"messageId"`
		EventType          string          `json:"eventType"`
		EventVersion       int             `json:"eventVersion"`
		OccurredAt         string          `json:"occurredAt"`
		SourceContext      string          `json:"sourceContext"`
		AggregateID        string          `json:"aggregateId"`
		AggregateVersion   int64           `json:"aggregateVersion"`
		AccountingScopeID  string          `json:"accountingScopeId,omitempty"`
		CorrelationID      string          `json:"correlationId"`
		CausationID        string          `json:"causationId"`
		DataClassification Classification  `json:"dataClassification"`
		PayloadFingerprint string          `json:"payloadFingerprint"`
		Data               json.RawMessage `json:"data"`
	}{e.MessageID(), e.EventType(), e.EventVersion(), e.OccurredAt().Format(time.RFC3339Nano), e.SourceContext(), e.AggregateID(), e.AggregateVersion(), e.AccountingScopeID(), e.CorrelationID(), e.CausationID(), e.DataClassification(), e.PayloadFingerprint(), e.Data()})
}
func (e *Envelope) UnmarshalJSON(b []byte) error {
	var x struct {
		MessageID          string          `json:"messageId"`
		EventType          string          `json:"eventType"`
		EventVersion       int             `json:"eventVersion"`
		OccurredAt         string          `json:"occurredAt"`
		SourceContext      string          `json:"sourceContext"`
		AggregateID        string          `json:"aggregateId"`
		AggregateVersion   int64           `json:"aggregateVersion"`
		AccountingScopeID  string          `json:"accountingScopeId"`
		CorrelationID      string          `json:"correlationId"`
		CausationID        string          `json:"causationId"`
		DataClassification Classification  `json:"dataClassification"`
		PayloadFingerprint string          `json:"payloadFingerprint"`
		Data               json.RawMessage `json:"data"`
	}
	d := json.NewDecoder(bytes.NewReader(b))
	d.DisallowUnknownFields()
	if err := d.Decode(&x); err != nil {
		return err
	}
	var extra any
	if err := d.Decode(&extra); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON")
		}
		return fmt.Errorf("trailing JSON: %w", err)
	}
	t, err := time.Parse(time.RFC3339Nano, x.OccurredAt)
	if err != nil {
		return err
	}
	n, err := NewEnvelope(EnvelopeInput{x.MessageID, x.EventType, x.EventVersion, t, x.SourceContext, x.AggregateID, x.AggregateVersion, x.AccountingScopeID, x.CorrelationID, x.CausationID, x.DataClassification, x.PayloadFingerprint, bytes.Clone(x.Data)})
	if err != nil {
		return fmt.Errorf("invalid envelope: %w", err)
	}
	*e = n
	return nil
}
