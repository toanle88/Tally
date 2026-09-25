package identity

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserLifecycleStartsInactiveAndTerminatesImmutably(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	subject := AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}
	user, err := NewUser(uuid.New(), subject, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if user.Status != UserStatusInactive {
		t.Fatalf("status = %q, want inactive", user.Status)
	}

	if err := user.Activate(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := user.Suspend(now.Add(2 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := user.Activate(now.Add(3 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := user.Terminate(now.Add(4 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(user.Terminate(now.Add(5*time.Minute)), ErrUserTerminated) {
		t.Fatalf("re-termination error = %v, want ErrUserTerminated", err)
	}
	if !errors.Is(user.Activate(now.Add(6*time.Minute)), ErrUserTerminated) {
		t.Fatalf("activation after termination error = %v, want ErrUserTerminated", err)
	}
}

func TestUserRejectsDuplicateAssignmentsAndPreservesSubject(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	roleID := uuid.New()
	user, err := NewUser(uuid.New(), AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	assignment := RoleAssignment{RoleID: roleID, Scopes: []EntityAccessScope{{ScopeID: "entity-1"}}}
	if err := user.ReplaceAssignments([]RoleAssignment{assignment, assignment}, now.Add(time.Minute)); !errors.Is(err, ErrDuplicateAssignment) {
		t.Fatalf("duplicate assignment error = %v, want ErrDuplicateAssignment", err)
	}
	if err := user.ReplaceAssignments([]RoleAssignment{assignment}, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if user.AuthenticationSubject != (AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}) {
		t.Fatalf("subject changed: %#v", user.AuthenticationSubject)
	}
}

func TestUserValidateRejectsInvalidAssignments(t *testing.T) {
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	user, err := NewUser(uuid.New(), AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}, []RoleAssignment{{RoleID: uuid.New()}}, now)
	if !errors.Is(err, ErrInvalidAssignment) {
		t.Fatalf("constructor error = %v, want ErrInvalidAssignment", err)
	}
	if user.ID != uuid.Nil {
		t.Fatalf("invalid user was returned: %#v", user)
	}
}
