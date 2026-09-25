package identity

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

var (
	ErrInvalidUser             = errors.New("invalid identity user")
	ErrInvalidUserTransition   = errors.New("invalid identity user transition")
	ErrUserTerminated          = errors.New("identity user is terminated")
	ErrDuplicateAssignment     = errors.New("duplicate role assignment")
	ErrInvalidAssignment       = errors.New("invalid role assignment")
	ErrAuthenticationImmutable = errors.New("authentication subject is immutable")
	ErrVersionConflict         = errors.New("identity user version conflict")
	ErrAuthorizationDenied     = errors.New("identity user authorization denied")
	ErrUserNotFound            = errors.New("identity user not found")
)

// UserStatus is deliberately explicit so suspended and terminated identities
// cannot be confused with an absent application user.
type UserStatus string

const (
	UserStatusInactive   UserStatus = "inactive"
	UserStatusActive     UserStatus = "active"
	UserStatusSuspended  UserStatus = "suspended"
	UserStatusTerminated UserStatus = "terminated"
)

func (status UserStatus) Validate() error {
	switch status {
	case UserStatusInactive, UserStatusActive, UserStatusSuspended, UserStatusTerminated:
		return nil
	default:
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidUser, status)
	}
}

// EntityAccessScope is an opaque reference to scope data owned by another
// bounded context. Identity stores the reference and never copies that
// context's business record.
type EntityAccessScope struct {
	ScopeID string
}

func (scope EntityAccessScope) Validate() error {
	if strings.TrimSpace(scope.ScopeID) == "" {
		return fmt.Errorf("%w: scope id is required", ErrInvalidAssignment)
	}
	return nil
}

type RoleAssignment struct {
	RoleID uuid.UUID
	Scopes []EntityAccessScope
}

func (assignment RoleAssignment) Validate() error {
	if assignment.RoleID == uuid.Nil {
		return fmt.Errorf("%w: role id is required", ErrInvalidAssignment)
	}
	if len(assignment.Scopes) == 0 {
		return fmt.Errorf("%w: at least one scope is required", ErrInvalidAssignment)
	}
	seen := make(map[string]struct{}, len(assignment.Scopes))
	for _, scope := range assignment.Scopes {
		if err := scope.Validate(); err != nil {
			return err
		}
		key := strings.TrimSpace(scope.ScopeID)
		if _, exists := seen[key]; exists {
			return fmt.Errorf("%w: role %s scope %q", ErrDuplicateAssignment, assignment.RoleID, key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func normalizeAssignments(assignments []RoleAssignment) ([]RoleAssignment, error) {
	result := make([]RoleAssignment, len(assignments))
	seen := make(map[string]struct{}, len(assignments))
	for i, assignment := range assignments {
		if err := assignment.Validate(); err != nil {
			return nil, err
		}
		copyAssignment := RoleAssignment{RoleID: assignment.RoleID, Scopes: make([]EntityAccessScope, len(assignment.Scopes))}
		for scopeIndex, scope := range assignment.Scopes {
			copyAssignment.Scopes[scopeIndex] = EntityAccessScope{ScopeID: strings.TrimSpace(scope.ScopeID)}
		}
		sort.Slice(copyAssignment.Scopes, func(left, right int) bool {
			return copyAssignment.Scopes[left].ScopeID < copyAssignment.Scopes[right].ScopeID
		})
		key := copyAssignment.RoleID.String()
		if _, exists := seen[key]; exists {
			return nil, fmt.Errorf("%w: role %s", ErrDuplicateAssignment, copyAssignment.RoleID)
		}
		seen[key] = struct{}{}
		result[i] = copyAssignment
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].RoleID.String() < result[right].RoleID.String()
	})
	return result, nil
}

type User struct {
	ID                    uuid.UUID
	AuthenticationSubject AuthenticationSubject
	Status                UserStatus
	Assignments           []RoleAssignment
	Version               aggregateversion.AggregateVersion
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

func NewUser(id uuid.UUID, subject AuthenticationSubject, assignments []RoleAssignment, now time.Time) (User, error) {
	if id == uuid.Nil {
		return User{}, fmt.Errorf("%w: id is required", ErrInvalidUser)
	}
	normalizedSubject, err := NewAuthenticationSubject(subject.OID, subject.TID, subject.Sub)
	if err != nil {
		return User{}, err
	}
	normalized, err := normalizeAssignments(assignments)
	if err != nil {
		return User{}, err
	}
	version := aggregateversion.Initial()
	return User{
		ID:                    id,
		AuthenticationSubject: normalizedSubject,
		Status:                UserStatusInactive,
		Assignments:           normalized,
		Version:               version,
		CreatedAt:             now,
		UpdatedAt:             now,
	}, nil
}

func (user User) Validate() error {
	if user.ID == uuid.Nil {
		return fmt.Errorf("%w: id is required", ErrInvalidUser)
	}
	if err := user.AuthenticationSubject.Validate(); err != nil {
		return err
	}
	if err := user.Status.Validate(); err != nil {
		return err
	}
	if _, err := normalizeAssignments(user.Assignments); err != nil {
		return err
	}
	if user.Version.Value() < 1 {
		return fmt.Errorf("%w: version must be positive", ErrInvalidUser)
	}
	if user.CreatedAt.IsZero() || user.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: timestamps are required", ErrInvalidUser)
	}
	return nil
}

func (user *User) ReplaceAssignments(assignments []RoleAssignment, now time.Time) error {
	if user == nil {
		return ErrInvalidUser
	}
	if user.Status == UserStatusTerminated {
		return ErrUserTerminated
	}
	normalized, err := normalizeAssignments(assignments)
	if err != nil {
		return err
	}
	user.Assignments = normalized
	user.UpdatedAt = now
	return nil
}

func (user *User) Activate(now time.Time) error {
	if user == nil {
		return ErrInvalidUser
	}
	switch user.Status {
	case UserStatusInactive, UserStatusSuspended:
		user.Status = UserStatusActive
		user.UpdatedAt = now
		return nil
	case UserStatusTerminated:
		return ErrUserTerminated
	default:
		return fmt.Errorf("%w: cannot activate %s user", ErrInvalidUserTransition, user.Status)
	}
}

func (user *User) Suspend(now time.Time) error {
	if user == nil {
		return ErrInvalidUser
	}
	switch user.Status {
	case UserStatusInactive, UserStatusActive:
		user.Status = UserStatusSuspended
		user.UpdatedAt = now
		return nil
	case UserStatusTerminated:
		return ErrUserTerminated
	default:
		return fmt.Errorf("%w: cannot suspend %s user", ErrInvalidUserTransition, user.Status)
	}
}

func (user *User) Terminate(now time.Time) error {
	if user == nil {
		return ErrInvalidUser
	}
	if user.Status == UserStatusTerminated {
		return ErrUserTerminated
	}
	user.Status = UserStatusTerminated
	user.UpdatedAt = now
	return nil
}
