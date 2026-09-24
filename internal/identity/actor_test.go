package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
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
