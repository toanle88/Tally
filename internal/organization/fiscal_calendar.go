package organization

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

const FiscalCalendarManagementPermission = "finance.omd.maintain.fiscal.calendars"

const (
	FiscalCalendarActionCreate   = "create"
	FiscalCalendarActionMaintain = "maintain"

	FiscalCalendarStatusDraft    = "draft"
	FiscalCalendarStatusActive   = "active"
	FiscalCalendarStatusEndDated = "end_dated"
)

var (
	ErrInvalidFiscalCalendar                  = errors.New("invalid fiscal calendar")
	ErrInvalidFiscalCalendarCommand           = errors.New("invalid fiscal calendar command")
	ErrFiscalCalendarNotFound                 = errors.New("fiscal calendar not found")
	ErrFiscalCalendarVersionConflict          = errors.New("fiscal calendar version conflict")
	ErrFiscalCalendarAuthorizationDenied      = errors.New("fiscal calendar authorization denied")
	ErrFiscalCalendarAuthorizationUnavailable = errors.New("fiscal calendar authorization unavailable")
	ErrFiscalCalendarAuthorizationStale       = errors.New("fiscal calendar authorization stale")
	ErrFiscalCalendarIdempotencyConflict      = errors.New("fiscal calendar idempotency conflict")
	ErrFiscalCalendarCommandInProgress        = errors.New("fiscal calendar command is already in progress")
	ErrFiscalCalendarAuditUnavailable         = errors.New("fiscal calendar audit unavailable")
	ErrFiscalCalendarApprovalRequired         = errors.New("fiscal calendar approval required")
	ErrFiscalCalendarApprovalRejected         = errors.New("fiscal calendar approval rejected")
	ErrFiscalCalendarApprovalUnavailable      = errors.New("fiscal calendar approval unavailable")
	ErrFiscalCalendarDuplicate                = errors.New("duplicate fiscal calendar value")
	ErrInvalidFiscalCalendarService           = errors.New("invalid fiscal calendar service")
	ErrFiscalCalendarDurableCommandFailed     = errors.New("fiscal calendar command previously failed")
	ErrFiscalCalendarImpactUnavailable        = errors.New("fiscal calendar impact is unavailable")
)

var fiscalCalendarCodePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

// CalendarPeriod is an explicitly supplied period definition. OMD stores the
// definition; it does not generate periods or create FPM-owned FiscalPeriod
// records.
type CalendarPeriod struct {
	ID        uuid.UUID `json:"id"`
	Reference string    `json:"reference"`
	Ordinal   int       `json:"ordinal"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type FiscalCalendar struct {
	ID             uuid.UUID                         `json:"id"`
	ScopeID        uuid.UUID                         `json:"scopeId"`
	CalendarType   string                            `json:"calendarType"`
	PeriodPattern  string                            `json:"periodPattern"`
	Status         string                            `json:"status"`
	EffectiveFrom  time.Time                         `json:"effectiveFrom"`
	EffectiveTo    *time.Time                        `json:"effectiveTo,omitempty"`
	Periods        []CalendarPeriod                  `json:"periods"`
	Approval       *ApprovalDecisionReference        `json:"approval,omitempty"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber int64                             `json:"revisionNumber"`
	CreatedAt      time.Time                         `json:"createdAt"`
	UpdatedAt      time.Time                         `json:"updatedAt"`
	Revisions      []FiscalCalendarRevision          `json:"revisions,omitempty"`
}

type FiscalCalendarRevision struct {
	RevisionNumber int64                             `json:"revisionNumber"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	Snapshot       FiscalCalendarSnapshot            `json:"snapshot"`
	CreatedAt      time.Time                         `json:"createdAt"`
}

type FiscalCalendarSnapshot struct {
	CalendarType  string                     `json:"calendarType"`
	PeriodPattern string                     `json:"periodPattern"`
	Status        string                     `json:"status"`
	EffectiveFrom time.Time                  `json:"effectiveFrom"`
	EffectiveTo   *time.Time                 `json:"effectiveTo,omitempty"`
	Periods       []CalendarPeriod           `json:"periods"`
	Approval      *ApprovalDecisionReference `json:"approval,omitempty"`
}

type SafeFiscalCalendar struct {
	ID             uuid.UUID                         `json:"id"`
	ScopeID        uuid.UUID                         `json:"scopeId"`
	CalendarType   string                            `json:"calendarType"`
	PeriodPattern  string                            `json:"periodPattern"`
	Status         string                            `json:"status"`
	EffectiveFrom  time.Time                         `json:"effectiveFrom"`
	EffectiveTo    *time.Time                        `json:"effectiveTo,omitempty"`
	Periods        []CalendarPeriod                  `json:"periods"`
	ApprovalStatus string                            `json:"approvalStatus"`
	Version        aggregateversion.AggregateVersion `json:"version"`
	RevisionNumber int64                             `json:"revisionNumber"`
	Revisions      []FiscalCalendarRevision          `json:"revisions,omitempty"`
	NextAction     string                            `json:"nextAction"`
}

// FiscalCalendarReference is the minimal OMD-owned identity contract used by
// downstream contexts for authoritative scope validation.
type FiscalCalendarReference struct {
	ID      uuid.UUID `json:"id"`
	ScopeID uuid.UUID `json:"scopeId"`
}

type FiscalCalendarImpactReference struct {
	Context   string `json:"context"`
	Reference string `json:"reference"`
	Version   int64  `json:"version,omitempty"`
	State     string `json:"state,omitempty"`
}

// FiscalCalendarImpact is deliberately a read-only result. An unavailable
// reader is a safe, explicit state and never blocks OMD persistence.
type FiscalCalendarImpact struct {
	Availability string                          `json:"availability"`
	Reason       string                          `json:"reason,omitempty"`
	References   []FiscalCalendarImpactReference `json:"references,omitempty"`
}

func (period CalendarPeriod) Validate() error {
	if period.ID == uuid.Nil || period.Reference != canonicalFiscalCalendarCode(period.Reference) || !fiscalCalendarCodePattern.MatchString(period.Reference) {
		return fmt.Errorf("%w: period identity and canonical reference are required", ErrInvalidFiscalCalendar)
	}
	if period.Ordinal < 1 || period.StartDate.IsZero() || period.EndDate.IsZero() || period.EndDate.Before(period.StartDate) {
		return fmt.Errorf("%w: period ordinal and date interval are invalid", ErrInvalidFiscalCalendar)
	}
	return nil
}

func (calendar FiscalCalendar) Snapshot() FiscalCalendarSnapshot {
	return FiscalCalendarSnapshot{
		CalendarType: calendar.CalendarType, PeriodPattern: calendar.PeriodPattern, Status: calendar.Status,
		EffectiveFrom: calendar.EffectiveFrom, EffectiveTo: cloneTime(calendar.EffectiveTo),
		Periods: cloneCalendarPeriods(calendar.Periods), Approval: cloneApproval(calendar.Approval),
	}
}

func (calendar FiscalCalendar) Validate() error {
	if calendar.ID == uuid.Nil || calendar.ScopeID == uuid.Nil || calendar.Version.Value() < 1 || calendar.RevisionNumber < 1 {
		return fmt.Errorf("%w: identity and version are required", ErrInvalidFiscalCalendar)
	}
	if calendar.CalendarType != canonicalFiscalCalendarCode(calendar.CalendarType) || !fiscalCalendarCodePattern.MatchString(calendar.CalendarType) {
		return fmt.Errorf("%w: calendar type must be a canonical code", ErrInvalidFiscalCalendar)
	}
	if calendar.PeriodPattern != canonicalFiscalCalendarCode(calendar.PeriodPattern) || !fiscalCalendarCodePattern.MatchString(calendar.PeriodPattern) {
		return fmt.Errorf("%w: period pattern must be canonical metadata", ErrInvalidFiscalCalendar)
	}
	switch calendar.Status {
	case FiscalCalendarStatusDraft, FiscalCalendarStatusActive:
		if calendar.EffectiveTo != nil {
			return fmt.Errorf("%w: non-ended calendar cannot have an end date", ErrInvalidFiscalCalendar)
		}
	case FiscalCalendarStatusEndDated:
		if calendar.EffectiveTo == nil {
			return fmt.Errorf("%w: ended calendar requires an end date", ErrInvalidFiscalCalendar)
		}
	default:
		return fmt.Errorf("%w: unsupported lifecycle status", ErrInvalidFiscalCalendar)
	}
	if calendar.EffectiveFrom.IsZero() || (calendar.EffectiveTo != nil && !calendar.EffectiveTo.After(calendar.EffectiveFrom)) {
		return fmt.Errorf("%w: effective interval is invalid", ErrInvalidFiscalCalendar)
	}
	if len(calendar.Periods) == 0 {
		return fmt.Errorf("%w: at least one explicit period is required", ErrInvalidFiscalCalendar)
	}
	seenIDs := make(map[uuid.UUID]struct{}, len(calendar.Periods))
	seenReferences := make(map[string]struct{}, len(calendar.Periods))
	for index, period := range calendar.Periods {
		if err := period.Validate(); err != nil {
			return err
		}
		if period.Ordinal != index+1 {
			return fmt.Errorf("%w: period ordinals must be contiguous and ordered", ErrInvalidFiscalCalendar)
		}
		if _, exists := seenIDs[period.ID]; exists {
			return fmt.Errorf("%w: period ids must be unique", ErrInvalidFiscalCalendar)
		}
		if _, exists := seenReferences[period.Reference]; exists {
			return fmt.Errorf("%w: period references must be unique", ErrInvalidFiscalCalendar)
		}
		seenIDs[period.ID] = struct{}{}
		seenReferences[period.Reference] = struct{}{}
		if period.StartDate.Before(calendar.EffectiveFrom) {
			return fmt.Errorf("%w: period starts before the calendar effective date", ErrInvalidFiscalCalendar)
		}
		if calendar.EffectiveTo != nil && period.EndDate.After(*calendar.EffectiveTo) {
			return fmt.Errorf("%w: period ends after the calendar effective date", ErrInvalidFiscalCalendar)
		}
		if index > 0 {
			previous := calendar.Periods[index-1]
			if !period.StartDate.After(previous.EndDate) {
				return fmt.Errorf("%w: period definitions overlap", ErrInvalidFiscalCalendar)
			}
		}
	}
	if calendar.Approval != nil {
		if err := calendar.Approval.Validate(); err != nil {
			return fmt.Errorf("%w: approval reference is invalid", ErrInvalidFiscalCalendar)
		}
	}
	if calendar.CreatedAt.IsZero() || calendar.UpdatedAt.IsZero() || calendar.UpdatedAt.Before(calendar.CreatedAt) {
		return fmt.Errorf("%w: timestamps are invalid", ErrInvalidFiscalCalendar)
	}
	return nil
}

func NewFiscalCalendar(id, scopeID uuid.UUID, calendarType, periodPattern string, effectiveFrom time.Time, periods []CalendarPeriod, approval *ApprovalDecisionReference, status string, now time.Time) (FiscalCalendar, error) {
	return newFiscalCalendar(id, scopeID, calendarType, periodPattern, effectiveFrom, nil, periods, approval, status, now)
}

func newFiscalCalendar(id, scopeID uuid.UUID, calendarType, periodPattern string, effectiveFrom time.Time, effectiveTo *time.Time, periods []CalendarPeriod, approval *ApprovalDecisionReference, status string, now time.Time) (FiscalCalendar, error) {
	calendar := FiscalCalendar{
		ID: id, ScopeID: scopeID, CalendarType: canonicalFiscalCalendarCode(calendarType), PeriodPattern: canonicalFiscalCalendarCode(periodPattern),
		Status: status, EffectiveFrom: dateOnly(effectiveFrom), EffectiveTo: cloneTime(effectiveTo), Periods: assignCalendarPeriodIDs(periods), Approval: cloneApproval(approval),
		Version: aggregateversion.Initial(), RevisionNumber: 1, CreatedAt: now.UTC(), UpdatedAt: now.UTC(),
	}
	if err := calendar.Validate(); err != nil {
		return FiscalCalendar{}, err
	}
	calendar.Revisions = []FiscalCalendarRevision{{RevisionNumber: calendar.RevisionNumber, Version: calendar.Version, Snapshot: calendar.Snapshot(), CreatedAt: calendar.UpdatedAt}}
	return calendar, nil
}

func (calendar *FiscalCalendar) Replace(current FiscalCalendar, calendarType, periodPattern string, effectiveFrom time.Time, effectiveTo *time.Time, periods []CalendarPeriod, approval *ApprovalDecisionReference, now time.Time) error {
	if calendar == nil || current.ID == uuid.Nil || calendar.ID != current.ID {
		return ErrInvalidFiscalCalendar
	}
	if current.Status == FiscalCalendarStatusEndDated {
		return fmt.Errorf("%w: an end-dated calendar cannot be maintained", ErrInvalidFiscalCalendar)
	}
	nextVersion, err := current.Version.Advance()
	if err != nil {
		return err
	}
	periods, err = preserveCalendarPeriodIDs(current.Periods, periods)
	if err != nil {
		return err
	}
	calendar.ScopeID = current.ScopeID
	calendar.CalendarType = canonicalFiscalCalendarCode(calendarType)
	calendar.PeriodPattern = canonicalFiscalCalendarCode(periodPattern)
	calendar.EffectiveFrom, calendar.EffectiveTo = dateOnly(effectiveFrom), cloneTime(effectiveTo)
	calendar.Periods = periods
	calendar.Approval, calendar.UpdatedAt = cloneApproval(approval), now.UTC()
	if calendar.EffectiveTo != nil {
		calendar.Status = FiscalCalendarStatusEndDated
	} else if current.Status == FiscalCalendarStatusDraft {
		calendar.Status = FiscalCalendarStatusActive
	} else {
		calendar.Status = current.Status
	}
	calendar.Version, calendar.RevisionNumber = nextVersion, current.RevisionNumber+1
	if err := calendar.Validate(); err != nil {
		return err
	}
	calendar.Revisions = append(cloneFiscalCalendarRevisions(current.Revisions), FiscalCalendarRevision{RevisionNumber: calendar.RevisionNumber, Version: calendar.Version, Snapshot: calendar.Snapshot(), CreatedAt: calendar.UpdatedAt})
	return nil
}

func (calendar FiscalCalendar) SafeProjection() SafeFiscalCalendar {
	approvalStatus, nextAction := "not-required", "maintain"
	if calendar.Approval != nil {
		approvalStatus, nextAction = "approved", "ready"
	}
	if calendar.Status == FiscalCalendarStatusDraft {
		approvalStatus, nextAction = "pending", "submit for approval"
	}
	if calendar.Status == FiscalCalendarStatusEndDated {
		nextAction = "view history"
	}
	return SafeFiscalCalendar{
		ID: calendar.ID, ScopeID: calendar.ScopeID, CalendarType: calendar.CalendarType, PeriodPattern: calendar.PeriodPattern,
		Status: calendar.Status, EffectiveFrom: calendar.EffectiveFrom, EffectiveTo: cloneTime(calendar.EffectiveTo),
		Periods: cloneCalendarPeriods(calendar.Periods), ApprovalStatus: approvalStatus, Version: calendar.Version,
		RevisionNumber: calendar.RevisionNumber, Revisions: cloneFiscalCalendarRevisions(calendar.Revisions), NextAction: nextAction,
	}
}

func FingerprintFiscalCalendar(calendar FiscalCalendar) string {
	data, _ := json.Marshal(calendar.Snapshot())
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func canonicalFiscalCalendarCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func assignCalendarPeriodIDs(periods []CalendarPeriod) []CalendarPeriod {
	result := cloneCalendarPeriods(periods)
	for index := range result {
		result[index].ID = valueOrNewUUID(result[index].ID)
		result[index].Reference = canonicalFiscalCalendarCode(result[index].Reference)
		result[index].StartDate = dateOnly(result[index].StartDate)
		result[index].EndDate = dateOnly(result[index].EndDate)
		if result[index].Ordinal == 0 {
			result[index].Ordinal = index + 1
		}
	}
	return result
}

func preserveCalendarPeriodIDs(current, periods []CalendarPeriod) ([]CalendarPeriod, error) {
	byReference := make(map[string]CalendarPeriod, len(current))
	for _, period := range current {
		byReference[period.Reference] = period
	}
	result := assignCalendarPeriodIDs(periods)
	for index := range result {
		if prior, ok := byReference[result[index].Reference]; ok {
			if periods[index].ID != uuid.Nil && periods[index].ID != prior.ID {
				return nil, fmt.Errorf("%w: an existing period reference cannot change identity", ErrInvalidFiscalCalendar)
			}
			result[index].ID = prior.ID
		}
	}
	return result, nil
}

func valueOrNewUUID(value uuid.UUID) uuid.UUID {
	if value == uuid.Nil {
		return uuid.New()
	}
	return value
}

func cloneCalendarPeriods(values []CalendarPeriod) []CalendarPeriod {
	result := make([]CalendarPeriod, len(values))
	copy(result, values)
	return result
}

func cloneFiscalCalendarRevisions(values []FiscalCalendarRevision) []FiscalCalendarRevision {
	result := make([]FiscalCalendarRevision, len(values))
	copy(result, values)
	for index := range result {
		result[index].Snapshot.Periods = cloneCalendarPeriods(values[index].Snapshot.Periods)
		result[index].Snapshot.EffectiveTo = cloneTime(values[index].Snapshot.EffectiveTo)
		result[index].Snapshot.Approval = cloneApproval(values[index].Snapshot.Approval)
	}
	return result
}

func cloneFiscalCalendar(value FiscalCalendar) FiscalCalendar {
	value.Periods = cloneCalendarPeriods(value.Periods)
	value.EffectiveTo = cloneTime(value.EffectiveTo)
	value.Approval = cloneApproval(value.Approval)
	value.Revisions = cloneFiscalCalendarRevisions(value.Revisions)
	return value
}
