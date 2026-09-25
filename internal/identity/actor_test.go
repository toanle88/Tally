package identity

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

func TestFixtureResolverMapsValidatedSubjectToApplicationActor(t *testing.T) {
	subject, err := NewAuthenticationSubject("oid-1", "tenant-1", "sub-1")
	if err != nil {
		t.Fatal(err)
	}
	userID := uuid.MustParse("0195a91b-20ab-7c15-8aa8-4e111a8bd618")
	resolver, err := NewFixtureResolver([]FixtureIdentity{{Subject: subject, UserID: userID}})
	if err != nil {
		t.Fatal(err)
	}

	actor, err := resolver.ResolveApplicationActor(context.Background(), subject)
	if err != nil {
		t.Fatal(err)
	}
	if actor.UserID != userID || actor.Subject != subject {
		t.Fatalf("actor = %#v, want user %s and subject %#v", actor, userID, subject)
	}
}

func TestFixtureResolverDoesNotResolveUnknownSubject(t *testing.T) {
	known, err := NewAuthenticationSubject("oid-1", "tenant-1", "sub-1")
	if err != nil {
		t.Fatal(err)
	}
	unknown, err := NewAuthenticationSubject("oid-2", "tenant-1", "sub-2")
	if err != nil {
		t.Fatal(err)
	}
	resolver, err := NewFixtureResolver([]FixtureIdentity{{Subject: known, UserID: uuid.New()}})
	if err != nil {
		t.Fatal(err)
	}

	_, err = resolver.ResolveApplicationActor(context.Background(), unknown)
	if !errors.Is(err, ErrApplicationActorUnavailable) {
		t.Fatalf("error = %v, want ErrApplicationActorUnavailable", err)
	}
}

func TestActorContextRejectsInvalidActor(t *testing.T) {
	_, err := WithActor(context.Background(), ApplicationActor{})
	if !errors.Is(err, ErrApplicationActorUnavailable) {
		t.Fatalf("error = %v, want ErrApplicationActorUnavailable", err)
	}
}

func TestAuthenticationSubjectRequiresAllStableClaims(t *testing.T) {
	if _, err := NewAuthenticationSubject("oid", "", "sub"); !errors.Is(err, ErrInvalidAuthenticationSubject) {
		t.Fatalf("error = %v, want ErrInvalidAuthenticationSubject", err)
	}
}

type staticUserResolver struct {
	actor ApplicationActor
}

func (resolver staticUserResolver) ResolveApplicationActor(context.Context, AuthenticationSubject) (ApplicationActor, error) {
	return resolver.actor, nil
}

func TestRevalidatingResolverRejectsUnknownProductionUser(t *testing.T) {
	subject := AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}
	resolver, err := NewRevalidatingUserResolver(staticUserResolver{
		actor: ApplicationActor{UserID: uuid.New(), Subject: subject},
	}, NewMemoryUserRepository())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveApplicationActor(context.Background(), subject); !errors.Is(err, ErrApplicationActorUnavailable) {
		t.Fatalf("unknown production user error = %v, want ErrApplicationActorUnavailable", err)
	}
}

func TestRevalidatingResolverRechecksLifecycleStatus(t *testing.T) {
	subject := AuthenticationSubject{OID: "oid", TID: "tenant", Sub: "subject"}
	userID := uuid.New()
	fallback, err := NewFixtureResolver([]FixtureIdentity{{Subject: subject, UserID: userID}})
	if err != nil {
		t.Fatal(err)
	}
	repository := NewMemoryUserRepository()
	now := time.Date(2026, 9, 25, 8, 0, 0, 0, time.UTC)
	user, err := NewUser(userID, subject, nil, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), user); err != nil {
		t.Fatal(err)
	}
	resolver, err := NewRevalidatingUserResolver(fallback, repository)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveApplicationActor(context.Background(), subject); !errors.Is(err, ErrApplicationActorUnavailable) {
		t.Fatalf("inactive resolution error = %v, want ErrApplicationActorUnavailable", err)
	}
	if err := user.Activate(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	nextVersion, err := user.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	user.Version = nextVersion
	if err := repository.Replace(context.Background(), user, aggregateversion.Initial()); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveApplicationActor(context.Background(), subject); err != nil {
		t.Fatal(err)
	}
	if err := user.Suspend(now.Add(2 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	nextVersion, err = user.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	user.Version = nextVersion
	if err := repository.Replace(context.Background(), user, aggregateversion.AggregateVersion(2)); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveApplicationActor(context.Background(), subject); !errors.Is(err, ErrApplicationActorUnavailable) {
		t.Fatalf("suspended resolution error = %v, want ErrApplicationActorUnavailable", err)
	}
	if err := user.Terminate(now.Add(3 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	nextVersion, err = user.Version.Advance()
	if err != nil {
		t.Fatal(err)
	}
	user.Version = nextVersion
	if err := repository.Replace(context.Background(), user, aggregateversion.AggregateVersion(3)); err != nil {
		t.Fatal(err)
	}
	if _, err := resolver.ResolveApplicationActor(context.Background(), subject); !errors.Is(err, ErrApplicationActorUnavailable) {
		t.Fatalf("terminated resolution error = %v, want ErrApplicationActorUnavailable", err)
	}
}
