package coa

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	SegmentDefinitionManagementPermission = "finance.coa.maintain.segment.definitions"
	SegmentValueManagementPermission      = "finance.coa.maintain.segment.values"
)

const (
	SegmentDefinitionActionCreate = "create"
	SegmentDefinitionActionUpdate = "update"
	SegmentValueActionCreate      = "create"
	SegmentValueActionUpdate      = "update"

	SegmentStatusDraft     = "draft"
	SegmentStatusActive    = "active"
	SegmentStatusSuspended = "suspended"
	SegmentStatusRetired   = "retired"
)

var (
	ErrInvalidSegmentDefinition                  = errors.New("invalid segment definition")
	ErrInvalidSegmentDefinitionCommand           = errors.New("invalid segment definition command")
	ErrSegmentDefinitionNotFound                 = errors.New("segment definition not found")
	ErrSegmentDefinitionVersionConflict          = errors.New("segment definition version conflict")
	ErrSegmentDefinitionAuthorizationDenied      = errors.New("segment definition authorization denied")
	ErrSegmentDefinitionAuthorizationUnavailable = errors.New("segment definition authorization unavailable")
	ErrSegmentDefinitionAuthorizationStale       = errors.New("segment definition authorization stale")
	ErrSegmentDefinitionIdempotencyConflict      = errors.New("segment definition idempotency conflict")
	ErrSegmentDefinitionCommandInProgress        = errors.New("segment definition command is already in progress")
	ErrSegmentDefinitionAuditUnavailable         = errors.New("segment definition audit unavailable")
	ErrSegmentDefinitionDuplicate                = errors.New("duplicate segment definition")
	ErrInvalidSegmentDefinitionService           = errors.New("invalid segment definition service")
	ErrSegmentDefinitionDurableCommandFailed     = errors.New("segment definition command previously failed")
	ErrInvalidSegmentValue                       = errors.New("invalid segment value")
	ErrInvalidSegmentValueCommand                = errors.New("invalid segment value command")
	ErrSegmentValueNotFound                      = errors.New("segment value not found")
	ErrSegmentValueVersionConflict               = errors.New("segment value version conflict")
	ErrSegmentValueAuthorizationDenied           = errors.New("segment value authorization denied")
	ErrSegmentValueAuthorizationUnavailable      = errors.New("segment value authorization unavailable")
	ErrSegmentValueAuthorizationStale            = errors.New("segment value authorization stale")
	ErrSegmentValueIdempotencyConflict           = errors.New("segment value idempotency conflict")
	ErrSegmentValueCommandInProgress             = errors.New("segment value command is already in progress")
	ErrSegmentValueAuditUnavailable              = errors.New("segment value audit unavailable")
	ErrSegmentValueDuplicate                     = errors.New("duplicate segment value")
	ErrInvalidSegmentValueService                = errors.New("invalid segment value service")
	ErrSegmentValueDurableCommandFailed          = errors.New("segment value command previously failed")
)

type SegmentStatus string

func (status SegmentStatus) String() string { return string(status) }

func canonicalSegmentStatus(value string) SegmentStatus {
	return SegmentStatus(strings.ToLower(strings.TrimSpace(value)))
}

func validSegmentStatus(status SegmentStatus) bool {
	switch status {
	case SegmentStatusDraft, SegmentStatusActive, SegmentStatusSuspended, SegmentStatusRetired:
		return true
	default:
		return false
	}
}

func validSegmentStatusTransition(from, to SegmentStatus) bool {
	if from == to {
		return true
	}
	switch from {
	case SegmentStatusDraft:
		return to == SegmentStatusActive || to == SegmentStatusRetired
	case SegmentStatusActive:
		return to == SegmentStatusSuspended || to == SegmentStatusRetired
	case SegmentStatusSuspended:
		return to == SegmentStatusActive || to == SegmentStatusRetired
	case SegmentStatusRetired:
		return false
	default:
		return false
	}
}

type SegmentDefinition struct {
	ID                uuid.UUID                         `json:"id"`
	ScopeID           uuid.UUID                         `json:"scopeId"`
	SegmentType       string                            `json:"segmentType"`
	Code              string                            `json:"code"`
	Name              string                            `json:"name"`
	Status            SegmentStatus                     `json:"status"`
	EffectiveDateFrom time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time                        `json:"effectiveDateTo,omitempty"`
	Version           aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber    int64                             `json:"revisionNumber"`
	CreatedAt         time.Time                         `json:"createdAt"`
	UpdatedAt         time.Time                         `json:"updatedAt"`
	Values            []SegmentValue                    `json:"values,omitempty"`
	Revisions         []SegmentDefinitionRevision       `json:"revisions,omitempty"`
}

type SegmentDefinitionSnapshot struct {
	SegmentType       string                 `json:"segmentType"`
	Code              string                 `json:"code"`
	Name              string                 `json:"name"`
	Status            SegmentStatus          `json:"status"`
	EffectiveDateFrom time.Time              `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time             `json:"effectiveDateTo,omitempty"`
	Values            []SegmentValueSnapshot `json:"values,omitempty"`
}

type SegmentValue struct {
	ID                  uuid.UUID     `json:"id"`
	SegmentDefinitionID uuid.UUID     `json:"segmentDefinitionId"`
	Value               string        `json:"value"`
	Description         string        `json:"description"`
	Status              SegmentStatus `json:"status"`
	EffectiveDateFrom   time.Time     `json:"effectiveDateFrom"`
	EffectiveDateTo     *time.Time    `json:"effectiveDateTo,omitempty"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
}

type SegmentValueSnapshot struct {
	ID                  uuid.UUID     `json:"id"`
	SegmentDefinitionID uuid.UUID     `json:"segmentDefinitionId"`
	Value               string        `json:"value"`
	Description         string        `json:"description"`
	Status              SegmentStatus `json:"status"`
	EffectiveDateFrom   time.Time     `json:"effectiveDateFrom"`
	EffectiveDateTo     *time.Time    `json:"effectiveDateTo,omitempty"`
	CreatedAt           time.Time     `json:"createdAt"`
	UpdatedAt           time.Time     `json:"updatedAt"`
}

type SegmentDefinitionRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	Snapshot       SegmentDefinitionSnapshot         `json:"snapshot"`
	CreatedAt      time.Time                         `json:"createdAt"`
}

func (definition SegmentDefinition) Snapshot() SegmentDefinitionSnapshot {
	values := make([]SegmentValueSnapshot, len(definition.Values))
	for index, value := range definition.Values {
		values[index] = value.Snapshot()
	}
	return SegmentDefinitionSnapshot{
		SegmentType:       definition.SegmentType,
		Code:              definition.Code,
		Name:              definition.Name,
		Status:            definition.Status,
		EffectiveDateFrom: definition.EffectiveDateFrom,
		EffectiveDateTo:   cloneDate(definition.EffectiveDateTo),
		Values:            values,
	}
}

func (value SegmentValue) Snapshot() SegmentValueSnapshot {
	return SegmentValueSnapshot{
		ID:                  value.ID,
		SegmentDefinitionID: value.SegmentDefinitionID,
		Value:               value.Value,
		Description:         value.Description,
		Status:              value.Status,
		EffectiveDateFrom:   value.EffectiveDateFrom,
		EffectiveDateTo:     cloneDate(value.EffectiveDateTo),
		CreatedAt:           value.CreatedAt,
		UpdatedAt:           value.UpdatedAt,
	}
}

func (definition SegmentDefinition) Validate() error {
	if definition.ID == uuid.Nil || definition.ScopeID == uuid.Nil || definition.Version.Value() < 1 || definition.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidSegmentDefinition)
	}
	if strings.TrimSpace(definition.SegmentType) == "" || strings.TrimSpace(definition.Code) == "" || strings.TrimSpace(definition.Name) == "" {
		return fmt.Errorf("%w: segment type, code, and name are required", ErrInvalidSegmentDefinition)
	}
	if !validSegmentStatus(definition.Status) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidSegmentDefinition)
	}
	if definition.EffectiveDateFrom.IsZero() {
		return fmt.Errorf("%w: effective date from is required", ErrInvalidSegmentDefinition)
	}
	if definition.EffectiveDateTo != nil && definition.EffectiveDateTo.Before(definition.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date range is invalid", ErrInvalidSegmentDefinition)
	}
	if definition.CreatedAt.IsZero() || definition.UpdatedAt.IsZero() || definition.UpdatedAt.Before(definition.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidSegmentDefinition)
	}
	for index, value := range definition.Values {
		if err := value.validate(definition); err != nil {
			return fmt.Errorf("%w: value %d: %w", ErrInvalidSegmentDefinition, index, err)
		}
		for previousIndex := 0; previousIndex < index; previousIndex++ {
			previous := definition.Values[previousIndex]
			if normalizeSegmentValue(previous.Value) == normalizeSegmentValue(value.Value) && rangesOverlap(previous.EffectiveDateFrom, previous.EffectiveDateTo, value.EffectiveDateFrom, value.EffectiveDateTo) {
				return fmt.Errorf("%w: value %q has an overlapping effective range", ErrSegmentValueDuplicate, value.Value)
			}
		}
	}
	return nil
}

func (value SegmentValue) validate(parent SegmentDefinition) error {
	if value.ID == uuid.Nil || value.SegmentDefinitionID == uuid.Nil || value.SegmentDefinitionID != parent.ID {
		return fmt.Errorf("%w: identity is required", ErrInvalidSegmentValue)
	}
	if normalizeSegmentValue(value.Value) == "" {
		return fmt.Errorf("%w: value is required", ErrInvalidSegmentValue)
	}
	if value.Value != normalizeSegmentValue(value.Value) {
		return fmt.Errorf("%w: value must be canonicalized", ErrInvalidSegmentValue)
	}
	if !validSegmentStatus(value.Status) {
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidSegmentValue)
	}
	if value.EffectiveDateFrom.IsZero() || value.EffectiveDateTo != nil && value.EffectiveDateTo.Before(value.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date range is invalid", ErrInvalidSegmentValue)
	}
	if value.EffectiveDateFrom.Before(parent.EffectiveDateFrom) || parent.EffectiveDateTo != nil && (value.EffectiveDateTo == nil || value.EffectiveDateTo.After(*parent.EffectiveDateTo)) {
		return fmt.Errorf("%w: effective range must be contained by the segment definition", ErrInvalidSegmentValue)
	}
	if value.CreatedAt.IsZero() || value.UpdatedAt.IsZero() || value.UpdatedAt.Before(value.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidSegmentValue)
	}
	return nil
}

func normalizeSegmentValue(value string) string { return strings.TrimSpace(value) }

func NewSegmentValue(parent SegmentDefinition, id uuid.UUID, value, description string, status SegmentStatus, from time.Time, to *time.Time, now time.Time) (SegmentValue, error) {
	status = canonicalSegmentStatus(status.String())
	if status == SegmentStatusRetired {
		return SegmentValue{}, fmt.Errorf("%w: a new value cannot start retired", ErrInvalidSegmentValue)
	}
	now = now.UTC()
	result := SegmentValue{
		ID:                  id,
		SegmentDefinitionID: parent.ID,
		Value:               normalizeSegmentValue(value),
		Description:         strings.TrimSpace(description),
		Status:              status,
		EffectiveDateFrom:   dateOnly(from),
		EffectiveDateTo:     cloneDate(to),
		CreatedAt:           now,
		UpdatedAt:           now,
	}
	if err := result.validate(parent); err != nil {
		return SegmentValue{}, err
	}
	return result, nil
}

func (value *SegmentValue) Replace(current SegmentValue, parent SegmentDefinition, nextValue, description string, status SegmentStatus, from time.Time, to *time.Time, now time.Time) error {
	if value == nil || current.ID == uuid.Nil || value.ID != current.ID {
		return ErrInvalidSegmentValue
	}
	if current.Status == SegmentStatusRetired {
		return fmt.Errorf("%w: a retired segment value cannot be maintained", ErrInvalidSegmentValue)
	}
	status = canonicalSegmentStatus(status.String())
	if !validSegmentStatusTransition(current.Status, status) {
		return fmt.Errorf("%w: lifecycle transition from %s to %s is not allowed", ErrInvalidSegmentValue, current.Status, status)
	}
	value.ID = current.ID
	value.SegmentDefinitionID = current.SegmentDefinitionID
	value.Value = normalizeSegmentValue(nextValue)
	value.Description = strings.TrimSpace(description)
	value.Status = status
	value.EffectiveDateFrom = dateOnly(from)
	value.EffectiveDateTo = cloneDate(to)
	value.CreatedAt = current.CreatedAt
	value.UpdatedAt = now.UTC()
	return value.validate(parent)
}

func NewSegmentDefinition(id, scopeID uuid.UUID, segmentType, code, name string, status SegmentStatus, from time.Time, to *time.Time, now time.Time) (SegmentDefinition, error) {
	status = canonicalSegmentStatus(status.String())
	if status == SegmentStatusRetired {
		return SegmentDefinition{}, fmt.Errorf("%w: a new definition cannot start retired", ErrInvalidSegmentDefinition)
	}
	now = now.UTC()
	definition := SegmentDefinition{
		ID:                id,
		ScopeID:           scopeID,
		SegmentType:       strings.TrimSpace(segmentType),
		Code:              strings.TrimSpace(code),
		Name:              strings.TrimSpace(name),
		Status:            status,
		EffectiveDateFrom: dateOnly(from),
		EffectiveDateTo:   cloneDate(to),
		Version:           aggregateversion.Initial(),
		RevisionNumber:    1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if err := definition.Validate(); err != nil {
		return SegmentDefinition{}, err
	}
	definition.Revisions = []SegmentDefinitionRevision{{RevisionNumber: 1, Version: definition.Version, Snapshot: definition.Snapshot(), CreatedAt: now}}
	return definition, nil
}

func (definition *SegmentDefinition) Replace(current SegmentDefinition, segmentType, code, name string, status SegmentStatus, from time.Time, to *time.Time, now time.Time) error {
	if definition == nil || current.ID == uuid.Nil || definition.ID != current.ID {
		return ErrInvalidSegmentDefinition
	}
	if current.Status == SegmentStatusRetired {
		return fmt.Errorf("%w: a retired segment definition cannot be maintained", ErrInvalidSegmentDefinition)
	}
	status = canonicalSegmentStatus(status.String())
	if !validSegmentStatusTransition(current.Status, status) {
		return fmt.Errorf("%w: lifecycle transition from %s to %s is not allowed", ErrInvalidSegmentDefinition, current.Status, status)
	}
	nextVersion, err := current.Version.Advance()
	if err != nil {
		return err
	}
	definition.ID = current.ID
	definition.ScopeID = current.ScopeID
	definition.SegmentType = strings.TrimSpace(segmentType)
	definition.Code = strings.TrimSpace(code)
	definition.Name = strings.TrimSpace(name)
	definition.Status = status
	definition.EffectiveDateFrom = dateOnly(from)
	definition.EffectiveDateTo = cloneDate(to)
	definition.Version = nextVersion
	definition.RevisionNumber = current.RevisionNumber + 1
	definition.CreatedAt = current.CreatedAt
	definition.UpdatedAt = now.UTC()
	if err := definition.Validate(); err != nil {
		return err
	}
	definition.Revisions = append(cloneRevisions(current.Revisions), SegmentDefinitionRevision{
		RevisionNumber: definition.RevisionNumber,
		Version:        definition.Version,
		Snapshot:       definition.Snapshot(),
		CreatedAt:      definition.UpdatedAt,
	})
	return nil
}

type SafeSegmentDefinition struct {
	ID                uuid.UUID                         `json:"id"`
	ScopeID           uuid.UUID                         `json:"scopeId"`
	SegmentType       string                            `json:"segmentType"`
	Code              string                            `json:"code"`
	Name              string                            `json:"name"`
	Status            SegmentStatus                     `json:"status"`
	EffectiveDateFrom time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo   *time.Time                        `json:"effectiveDateTo,omitempty"`
	ApprovalStatus    string                            `json:"approvalStatus"`
	ValidationOutcome string                            `json:"validationOutcome"`
	NextAction        string                            `json:"nextAction"`
	Version           aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber    int64                             `json:"revisionNumber"`
}

func (definition SegmentDefinition) SafeProjection() SafeSegmentDefinition {
	nextAction := "maintain"
	if definition.Status == SegmentStatusRetired {
		nextAction = "view history"
	}
	return SafeSegmentDefinition{
		ID:                definition.ID,
		ScopeID:           definition.ScopeID,
		SegmentType:       definition.SegmentType,
		Code:              definition.Code,
		Name:              definition.Name,
		Status:            definition.Status,
		EffectiveDateFrom: definition.EffectiveDateFrom,
		EffectiveDateTo:   cloneDate(definition.EffectiveDateTo),
		ApprovalStatus:    "not-required",
		ValidationOutcome: "valid",
		NextAction:        nextAction,
		Version:           definition.Version,
		RevisionNumber:    definition.RevisionNumber,
	}
}

type SafeSegmentValue struct {
	ID                  uuid.UUID                         `json:"id"`
	SegmentDefinitionID uuid.UUID                         `json:"segmentDefinitionId"`
	ScopeID             uuid.UUID                         `json:"scopeId"`
	Value               string                            `json:"value"`
	Description         string                            `json:"description"`
	Status              SegmentStatus                     `json:"status"`
	EffectiveDateFrom   time.Time                         `json:"effectiveDateFrom"`
	EffectiveDateTo     *time.Time                        `json:"effectiveDateTo,omitempty"`
	ApprovalStatus      string                            `json:"approvalStatus"`
	ValidationOutcome   string                            `json:"validationOutcome"`
	NextAction          string                            `json:"nextAction"`
	Version             aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber      int64                             `json:"revisionNumber"`
}

func (value SegmentValue) SafeProjection(parent SegmentDefinition) SafeSegmentValue {
	nextAction := "maintain"
	if value.Status == SegmentStatusRetired {
		nextAction = "view history"
	}
	return SafeSegmentValue{
		ID:                  value.ID,
		SegmentDefinitionID: value.SegmentDefinitionID,
		ScopeID:             parent.ScopeID,
		Value:               value.Value,
		Description:         value.Description,
		Status:              value.Status,
		EffectiveDateFrom:   value.EffectiveDateFrom,
		EffectiveDateTo:     cloneDate(value.EffectiveDateTo),
		ApprovalStatus:      "not-required",
		ValidationOutcome:   "valid",
		NextAction:          nextAction,
		Version:             parent.Version,
		RevisionNumber:      parent.RevisionNumber,
	}
}

type SegmentDefinitionCommand struct {
	Action              string
	SegmentDefinitionID uuid.UUID
	ScopeID             uuid.UUID
	SegmentType         string
	Code                string
	Name                string
	Status              SegmentStatus
	EffectiveDateFrom   time.Time
	EffectiveDateTo     *time.Time
	ExpectedVersion     *aggregateversion.AggregateVersion
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command SegmentDefinitionCommand) Canonical() SegmentDefinitionCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.SegmentType = strings.TrimSpace(command.SegmentType)
	command.Code = strings.TrimSpace(command.Code)
	command.Name = strings.TrimSpace(command.Name)
	command.Status = canonicalSegmentStatus(command.Status.String())
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	return command
}

func (command SegmentDefinitionCommand) Validate() error {
	command = command.Canonical()
	switch command.Action {
	case SegmentDefinitionActionCreate:
		if command.SegmentDefinitionID != uuid.Nil || command.ExpectedVersion != nil {
			return fmt.Errorf("%w: create cannot include id or expected version", ErrInvalidSegmentDefinitionCommand)
		}
	case SegmentDefinitionActionUpdate:
		if command.SegmentDefinitionID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: update id and expected version are required", ErrInvalidSegmentDefinitionCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidSegmentDefinitionCommand, command.Action)
	}
	if command.ScopeID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope is required", ErrInvalidSegmentDefinitionCommand)
	}
	if strings.TrimSpace(command.SegmentType) == "" || strings.TrimSpace(command.Code) == "" || strings.TrimSpace(command.Name) == "" {
		return fmt.Errorf("%w: segment type, code, and name are required", ErrInvalidSegmentDefinitionCommand)
	}
	if !validSegmentStatus(command.Status) || (command.Action == SegmentDefinitionActionCreate && command.Status == SegmentStatusRetired) {
		return fmt.Errorf("%w: lifecycle status is invalid for this action", ErrInvalidSegmentDefinitionCommand)
	}
	if command.EffectiveDateFrom.IsZero() || (command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom)) {
		return fmt.Errorf("%w: effective date range is invalid", ErrInvalidSegmentDefinitionCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidSegmentDefinitionCommand)
	}
	return nil
}

type SegmentValueCommand struct {
	Action              string
	SegmentDefinitionID uuid.UUID
	SegmentValueID      uuid.UUID
	ScopeID             uuid.UUID
	Value               string
	Description         string
	Status              SegmentStatus
	EffectiveDateFrom   time.Time
	EffectiveDateTo     *time.Time
	ExpectedVersion     *aggregateversion.AggregateVersion
	IdempotencyKey      string
	CorrelationID       string
	CausationID         string
}

func (command SegmentValueCommand) Canonical() SegmentValueCommand {
	command.Action = strings.ToLower(strings.TrimSpace(command.Action))
	command.Value = normalizeSegmentValue(command.Value)
	command.Description = strings.TrimSpace(command.Description)
	command.Status = canonicalSegmentStatus(command.Status.String())
	command.EffectiveDateFrom = dateOnly(command.EffectiveDateFrom)
	command.EffectiveDateTo = cloneDate(command.EffectiveDateTo)
	return command
}

func (command SegmentValueCommand) Validate() error {
	command = command.Canonical()
	switch command.Action {
	case SegmentValueActionCreate:
		if command.SegmentDefinitionID == uuid.Nil || command.SegmentValueID != uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: create requires parent id and expected parent version", ErrInvalidSegmentValueCommand)
		}
	case SegmentValueActionUpdate:
		if command.SegmentDefinitionID == uuid.Nil || command.SegmentValueID == uuid.Nil || command.ExpectedVersion == nil || command.ExpectedVersion.Value() < 1 {
			return fmt.Errorf("%w: update id and expected parent version are required", ErrInvalidSegmentValueCommand)
		}
	default:
		return fmt.Errorf("%w: unsupported action %q", ErrInvalidSegmentValueCommand, command.Action)
	}
	if command.ScopeID == uuid.Nil {
		return fmt.Errorf("%w: accounting scope is required", ErrInvalidSegmentValueCommand)
	}
	if command.Value == "" {
		return fmt.Errorf("%w: value is required", ErrInvalidSegmentValueCommand)
	}
	if !validSegmentStatus(command.Status) || (command.Action == SegmentValueActionCreate && command.Status == SegmentStatusRetired) {
		return fmt.Errorf("%w: lifecycle status is invalid for this action", ErrInvalidSegmentValueCommand)
	}
	if command.EffectiveDateFrom.IsZero() || command.EffectiveDateTo != nil && command.EffectiveDateTo.Before(command.EffectiveDateFrom) {
		return fmt.Errorf("%w: effective date range is invalid", ErrInvalidSegmentValueCommand)
	}
	if strings.TrimSpace(command.IdempotencyKey) == "" {
		return fmt.Errorf("%w: idempotency key is required", ErrInvalidSegmentValueCommand)
	}
	return nil
}

type Actor struct {
	UserID           uuid.UUID
	SubjectReference string
}

func (actor Actor) Validate() error {
	if actor.UserID == uuid.Nil || strings.TrimSpace(actor.SubjectReference) == "" {
		return ErrSegmentDefinitionAuthorizationDenied
	}
	return nil
}

type AuthorizationDecision struct {
	Allowed           bool
	Outcome           string
	Permission        string
	ApprovedScopeIDs  []uuid.UUID
	PolicyReference   string
	PolicyVersion     string
	DecisionReference uuid.UUID
	Reason            string
}

type Authorizer interface {
	AuthorizeSegmentDefinition(context.Context, Actor, SegmentDefinitionCommand, *SegmentDefinition) (AuthorizationDecision, error)
}

type SegmentValueAuthorizer interface {
	AuthorizeSegmentValue(context.Context, Actor, SegmentValueCommand, *SegmentDefinition, *SegmentValue) (AuthorizationDecision, error)
}

type AuditRecord struct {
	SegmentDefinitionID   uuid.UUID
	SegmentValueID        uuid.UUID
	ActorUserID           uuid.UUID
	ActorSubjectReference string
	Action                string
	ScopeID               uuid.UUID
	Permission            string
	PolicyReference       string
	PolicyVersion         string
	DecisionReference     uuid.UUID
	RevisionNumber        int64
	BeforeFingerprint     string
	AfterFingerprint      string
	CorrelationID         string
	CausationID           string
}

type AuditRecorder interface {
	RecordSegmentDefinitionMutation(context.Context, AuditRecord) error
}

type SegmentValueAuditRecorder interface {
	RecordSegmentValueMutation(context.Context, AuditRecord) error
}

type SegmentDefinitionMutation struct {
	Before          SegmentDefinition
	After           SegmentDefinition
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           AuditRecord
}

type SegmentDefinitionRepository interface {
	Get(context.Context, uuid.UUID) (SegmentDefinition, error)
	List(context.Context, *uuid.UUID) ([]SegmentDefinition, error)
	CommitSegmentDefinitionMutation(context.Context, SegmentDefinitionMutation) error
}

type SegmentValueMutation struct {
	Before          SegmentDefinition
	After           SegmentDefinition
	ValueBefore     *SegmentValue
	ValueAfter      SegmentValue
	ExpectedVersion *aggregateversion.AggregateVersion
	Audit           AuditRecord
}

type SegmentValueRepository interface {
	Get(context.Context, uuid.UUID) (SegmentDefinition, error)
	CommitSegmentValueMutation(context.Context, SegmentValueMutation) error
}

type DurableSegmentDefinitionRepository interface {
	CommitSegmentDefinitionMutationWithIdempotency(context.Context, SegmentDefinitionMutation, DurableSegmentDefinitionMutationCommit) error
}

type DurableSegmentValueRepository interface {
	CommitSegmentValueMutationWithIdempotency(context.Context, SegmentValueMutation, DurableSegmentValueMutationCommit) error
}

type DurableSegmentDefinitionMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableSegmentValueMutationCommit struct {
	Coordinator platformidempotency.DurableCoordinator
	Acquisition platformidempotency.DurableAcquisition
	Result      platformidempotency.CommandResultMetadata
}

type DurableSegmentDefinitionServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type DurableSegmentValueServiceConfig struct {
	Database    platformidempotency.TxBeginner
	Coordinator platformidempotency.DurableCoordinator
	Policy      platformidempotency.IdempotencyPolicy
	OperationID string
}

type SegmentDefinitionCommandResult struct {
	SegmentDefinition SafeSegmentDefinition `json:"segmentDefinition"`
	DecisionReference uuid.UUID             `json:"decisionReference"`
	PolicyReference   string                `json:"policyReference"`
	ValidationOutcome string                `json:"validationOutcome"`
	Replayed          bool                  `json:"replayed,omitempty"`
}

type SegmentValueCommandResult struct {
	SegmentValue      SafeSegmentValue `json:"segmentValue"`
	DecisionReference uuid.UUID        `json:"decisionReference"`
	PolicyReference   string           `json:"policyReference"`
	ValidationOutcome string           `json:"validationOutcome"`
	Replayed          bool             `json:"replayed,omitempty"`
}

func FingerprintSegmentDefinition(definition SegmentDefinition) string {
	data, _ := json.Marshal(definition.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func segmentDefinitionCommandFingerprint(command SegmentDefinitionCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func FingerprintSegmentValue(value SegmentValue) string {
	data, _ := json.Marshal(value.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func segmentValueCommandFingerprint(command SegmentValueCommand) (string, error) {
	command = command.Canonical()
	command.IdempotencyKey, command.CorrelationID, command.CausationID = "", "", ""
	data, err := json.Marshal(command)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func cloneDate(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	result := dateOnly(*value)
	return &result
}

func dateOnly(value time.Time) time.Time {
	if value.IsZero() {
		return time.Time{}
	}
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
}

func cloneRevisions(values []SegmentDefinitionRevision) []SegmentDefinitionRevision {
	result := make([]SegmentDefinitionRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.EffectiveDateTo = cloneDate(result[index].Snapshot.EffectiveDateTo)
		result[index].Snapshot.Values = cloneValueSnapshots(result[index].Snapshot.Values)
	}
	return result
}

func cloneValueSnapshots(values []SegmentValueSnapshot) []SegmentValueSnapshot {
	result := make([]SegmentValueSnapshot, len(values))
	copy(result, values)
	for index := range result {
		result[index].EffectiveDateFrom = dateOnly(result[index].EffectiveDateFrom)
		result[index].EffectiveDateTo = cloneDate(result[index].EffectiveDateTo)
	}
	return result
}

func rangesOverlap(fromA time.Time, toA *time.Time, fromB time.Time, toB *time.Time) bool {
	endA, endB := time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC), time.Date(9999, 12, 31, 0, 0, 0, 0, time.UTC)
	if toA != nil {
		endA = dateOnly(*toA)
	}
	if toB != nil {
		endB = dateOnly(*toB)
	}
	return !dateOnly(fromA).After(endB) && !dateOnly(fromB).After(endA)
}

func sortSegmentDefinitions(values []SegmentDefinition) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].ScopeID != values[j].ScopeID {
			return values[i].ScopeID.String() < values[j].ScopeID.String()
		}
		if values[i].SegmentType != values[j].SegmentType {
			return values[i].SegmentType < values[j].SegmentType
		}
		if values[i].Code != values[j].Code {
			return values[i].Code < values[j].Code
		}
		return values[i].EffectiveDateFrom.Before(values[j].EffectiveDateFrom)
	})
}

func sortSegmentValues(values []SegmentValue) {
	sort.Slice(values, func(i, j int) bool {
		if normalizeSegmentValue(values[i].Value) != normalizeSegmentValue(values[j].Value) {
			return normalizeSegmentValue(values[i].Value) < normalizeSegmentValue(values[j].Value)
		}
		return values[i].EffectiveDateFrom.Before(values[j].EffectiveDateFrom)
	})
}
