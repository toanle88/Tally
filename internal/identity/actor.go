// Package identity owns the application actor contract used by protected
// finance operations. Authentication adapters validate external subjects;
// this package maps those subjects to application-owned actor identities.
package identity

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidAuthenticationSubject = errors.New("invalid authentication subject")
	ErrApplicationActorUnavailable  = errors.New("application actor unavailable")
	ErrActorResolverUnavailable     = errors.New("actor resolver unavailable")
)

// AuthenticationSubject contains the stable identity claims accepted from the
// enterprise identity provider. Token scopes and roles are deliberately not
// part of this value; finance authorization remains application-owned.
type AuthenticationSubject struct {
	OID       string
	TID       string
	Sub       string
	Assurance AuthenticationAssurance
}

// AuthenticationAssurance contains non-sensitive assurance evidence from the
// validated authentication boundary. Raw tokens and unapproved claims never
// cross into the identity domain.
type AuthenticationAssurance struct {
	AuthenticatedAt time.Time
	AssuranceLevel  string
	Methods         string
	StepUpReference string
}

func (assurance AuthenticationAssurance) Satisfies(now time.Time, maxAge time.Duration) bool {
	if strings.TrimSpace(assurance.StepUpReference) != "" {
		return true
	}
	if assurance.AuthenticatedAt.IsZero() || maxAge <= 0 {
		return false
	}
	now = now.UTC()
	authenticatedAt := assurance.AuthenticatedAt.UTC()
	return !authenticatedAt.After(now) && now.Sub(authenticatedAt) <= maxAge
}

func NewAuthenticationSubject(oid, tid, sub string) (AuthenticationSubject, error) {
	value := AuthenticationSubject{
		OID: strings.TrimSpace(oid),
		TID: strings.TrimSpace(tid),
		Sub: strings.TrimSpace(sub),
	}
	if value.OID == "" || value.TID == "" || value.Sub == "" {
		return AuthenticationSubject{}, ErrInvalidAuthenticationSubject
	}
	return value, nil
}

func (subject AuthenticationSubject) Validate() error {
	if strings.TrimSpace(subject.OID) == "" ||
		strings.TrimSpace(subject.TID) == "" ||
		strings.TrimSpace(subject.Sub) == "" {
		return ErrInvalidAuthenticationSubject
	}
	return nil
}

// ApplicationActor is the identity used by finance and audit application
// boundaries after external authentication has succeeded.
type ApplicationActor struct {
	UserID  uuid.UUID
	Subject AuthenticationSubject
}

func (actor ApplicationActor) Validate() error {
	if actor.UserID == uuid.Nil {
		return fmt.Errorf("%w: application user id is required", ErrApplicationActorUnavailable)
	}
	return actor.Subject.Validate()
}

// UserResolver maps a validated external subject to an application actor.
// User lifecycle and persistence are intentionally outside this story.
type UserResolver interface {
	ResolveApplicationActor(context.Context, AuthenticationSubject) (ApplicationActor, error)
}

// RevalidatingUserResolver combines external subject resolution with the current
// application-owned user state. A non-active user cannot obtain an application actor.
type RevalidatingUserResolver struct {
	fallback   UserResolver
	repository UserRepository
}

func NewRevalidatingUserResolver(fallback UserResolver, repository UserRepository) (*RevalidatingUserResolver, error) {
	if fallback == nil || repository == nil {
		return nil, ErrActorResolverUnavailable
	}
	return &RevalidatingUserResolver{fallback: fallback, repository: repository}, nil
}

func (resolver *RevalidatingUserResolver) ResolveApplicationActor(ctx context.Context, subject AuthenticationSubject) (ApplicationActor, error) {
	if resolver == nil {
		return ApplicationActor{}, ErrActorResolverUnavailable
	}
	actor, err := resolver.fallback.ResolveApplicationActor(ctx, subject)
	if err != nil {
		return ApplicationActor{}, err
	}
	user, err := resolver.repository.FindByAuthenticationSubject(ctx, subject)
	if errors.Is(err, ErrUserNotFound) {
		// Only the explicit local fixture resolver may stand in for a not-yet-
		// provisioned user. Production authentication must have an active
		// application-owned user record before it can produce an actor.
		if _, fixture := resolver.fallback.(*FixtureResolver); fixture {
			return actor, nil
		}
		return ApplicationActor{}, ErrApplicationActorUnavailable
	}
	if err != nil {
		return ApplicationActor{}, err
	}
	if user.ID != actor.UserID || user.Status != UserStatusActive {
		return ApplicationActor{}, ErrApplicationActorUnavailable
	}
	return ApplicationActor{UserID: user.ID, Subject: subject}, nil
}

// FixtureIdentity describes a clearly marked local-only identity mapping.
type FixtureIdentity struct {
	Subject   AuthenticationSubject
	UserID    uuid.UUID
	Assurance AuthenticationAssurance
}

type subjectKey struct {
	oid string
	tid string
	sub string
}

// FixtureResolver is a deterministic resolver for local tests and fixture
// development. It must never be used as a production user directory.
type FixtureResolver struct {
	actors map[subjectKey]ApplicationActor
}

func NewFixtureResolver(entries []FixtureIdentity) (*FixtureResolver, error) {
	actors := make(map[subjectKey]ApplicationActor, len(entries))
	for _, entry := range entries {
		if err := entry.Subject.Validate(); err != nil {
			return nil, err
		}
		subject := entry.Subject
		if entry.Assurance != (AuthenticationAssurance{}) {
			subject.Assurance = entry.Assurance
		}
		actor := ApplicationActor{UserID: entry.UserID, Subject: subject}
		if err := actor.Validate(); err != nil {
			return nil, err
		}
		key := keyFor(entry.Subject)
		if _, exists := actors[key]; exists {
			return nil, fmt.Errorf("%w: duplicate fixture subject", ErrApplicationActorUnavailable)
		}
		actors[key] = actor
	}
	return &FixtureResolver{actors: actors}, nil
}

func (resolver *FixtureResolver) ResolveApplicationActor(_ context.Context, subject AuthenticationSubject) (ApplicationActor, error) {
	if resolver == nil {
		return ApplicationActor{}, ErrActorResolverUnavailable
	}
	if err := subject.Validate(); err != nil {
		return ApplicationActor{}, err
	}
	actor, ok := resolver.actors[keyFor(subject)]
	if !ok {
		return ApplicationActor{}, ErrApplicationActorUnavailable
	}
	return actor, nil
}

func keyFor(subject AuthenticationSubject) subjectKey {
	return subjectKey{oid: subject.OID, tid: subject.TID, sub: subject.Sub}
}

type actorContextKey struct{}

func WithActor(ctx context.Context, actor ApplicationActor) (context.Context, error) {
	if ctx == nil {
		return nil, ErrApplicationActorUnavailable
	}
	if err := actor.Validate(); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, actorContextKey{}, actor), nil
}

func ActorFromContext(ctx context.Context) (ApplicationActor, bool) {
	if ctx == nil {
		return ApplicationActor{}, false
	}
	actor, ok := ctx.Value(actorContextKey{}).(ApplicationActor)
	return actor, ok
}
