package authentication

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	fixtureIssuer   = "https://tally.local.test/issuer"
	fixtureTenant   = "tenant-fixture"
	fixtureAudience = "api://tally"
	fixtureKID      = "fixture-key"
)

func TestValidatorAcceptsSignedFixtureAndReturnsStableSubject(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, now, &privateKey.PublicKey)

	raw := signedFixtureToken(t, privateKey, fixtureTokenOptions{
		OID: "oid-123",
		TID: fixtureTenant,
		Sub: "subject-123",
		Exp: now.Add(10 * time.Minute),
	})
	subject, err := validator.ValidateAccessToken(context.Background(), raw)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if subject.OID != "oid-123" || subject.TID != fixtureTenant || subject.Sub != "subject-123" {
		t.Fatalf("subject = %#v", subject)
	}
}

func TestValidatorRejectsInvalidTokenCategories(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	otherKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, now, &privateKey.PublicKey)

	tests := []struct {
		name  string
		raw   func() string
		want  error
		check string
	}{
		{
			name: "empty",
			raw:  func() string { return "" },
			want: ErrInvalidToken,
		},
		{
			name: "malformed",
			raw:  func() string { return "not-a-jwt" },
			want: ErrInvalidToken,
		},
		{
			name: "wrong issuer",
			raw: func() string {
				return signedFixtureToken(t, privateKey, fixtureTokenOptions{Issuer: "https://wrong.example", Exp: now.Add(time.Minute)})
			},
			want: ErrInvalidToken,
		},
		{
			name: "wrong audience",
			raw: func() string {
				return signedFixtureToken(t, privateKey, fixtureTokenOptions{Audience: "api://wrong", Exp: now.Add(time.Minute)})
			},
			want: ErrInvalidToken,
		},
		{
			name: "wrong tenant",
			raw: func() string {
				return signedFixtureToken(t, privateKey, fixtureTokenOptions{TID: "tenant-wrong", Exp: now.Add(time.Minute)})
			},
			want: ErrInvalidToken,
		},
		{
			name: "expired",
			raw: func() string {
				return signedFixtureToken(t, privateKey, fixtureTokenOptions{Exp: now.Add(-ApprovedClockSkew - time.Second)})
			},
			want: ErrInvalidToken,
		},
		{
			name: "not before",
			raw: func() string {
				return signedFixtureToken(t, privateKey, fixtureTokenOptions{Exp: now.Add(time.Minute), NBF: now.Add(ApprovedClockSkew + time.Second)})
			},
			want: ErrInvalidToken,
		},
		{
			name: "bad signature",
			raw: func() string {
				return signedFixtureToken(t, otherKey, fixtureTokenOptions{Exp: now.Add(time.Minute)})
			},
			want: ErrInvalidToken,
		},
		{
			name: "algorithm mismatch",
			raw: func() string {
				claims := fixtureClaims(fixtureTokenOptions{Exp: now.Add(time.Minute)})
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
				token.Header["kid"] = fixtureKID
				raw, err := token.SignedString([]byte("not-a-rsa-key"))
				if err != nil {
					t.Fatal(err)
				}
				return raw
			},
			want: ErrInvalidToken,
		},
		{
			name: "missing key id",
			raw: func() string {
				token := jwt.NewWithClaims(jwt.SigningMethodRS256, fixtureClaims(fixtureTokenOptions{Exp: now.Add(time.Minute)}))
				raw, err := token.SignedString(privateKey)
				if err != nil {
					t.Fatal(err)
				}
				return raw
			},
			want: ErrInvalidToken,
		},
		{
			name: "unknown key",
			raw: func() string {
				return signedFixtureToken(t, privateKey, fixtureTokenOptions{KID: "unknown-key", Exp: now.Add(time.Minute)})
			},
			want: ErrInvalidToken,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validator.ValidateAccessToken(context.Background(), test.raw())
			if !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
			if strings.Contains(err.Error(), "tenant-wrong") || strings.Contains(err.Error(), "not-a-jwt") {
				t.Fatalf("error leaked token data: %v", err)
			}
		})
	}
}

func TestValidatorAppliesApprovedClockSkewAtBoundaries(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	validator := newFixtureValidator(t, now, &privateKey.PublicKey)

	tests := []struct {
		name string
		opts fixtureTokenOptions
		want bool
	}{
		{name: "expiration within skew boundary", opts: fixtureTokenOptions{Exp: now.Add(-ApprovedClockSkew + time.Second)}, want: true},
		{name: "expiration beyond skew", opts: fixtureTokenOptions{Exp: now.Add(-ApprovedClockSkew - time.Second)}, want: false},
		{name: "not before within skew boundary", opts: fixtureTokenOptions{Exp: now.Add(time.Minute), NBF: now.Add(ApprovedClockSkew - time.Second)}, want: true},
		{name: "not before beyond skew", opts: fixtureTokenOptions{Exp: now.Add(time.Minute), NBF: now.Add(ApprovedClockSkew + time.Second)}, want: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := validator.ValidateAccessToken(context.Background(), signedFixtureToken(t, privateKey, test.opts))
			if (err == nil) != test.want {
				t.Fatalf("error = %v, want valid = %v", err, test.want)
			}
		})
	}
}

func TestValidatorFailsClosedWhenKeyDependencyIsUnavailable(t *testing.T) {
	now := time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC)
	privateKey := newTestRSAKey(t)
	validator, err := NewValidator(Config{
		Issuer:   fixtureIssuer,
		TenantID: fixtureTenant,
		Audience: fixtureAudience,
		Clock:    func() time.Time { return now },
	}, dependencyKeySource{})
	if err != nil {
		t.Fatal(err)
	}

	_, err = validator.ValidateAccessToken(context.Background(), signedFixtureToken(t, privateKey, fixtureTokenOptions{Exp: now.Add(time.Minute)}))
	if !errors.Is(err, ErrAuthenticationDependency) {
		t.Fatalf("error = %v, want dependency error", err)
	}
}

func newFixtureValidator(t *testing.T, now time.Time, publicKey *rsa.PublicKey) *Validator {
	t.Helper()
	validator, err := NewValidator(Config{
		Issuer:   fixtureIssuer,
		TenantID: fixtureTenant,
		Audience: fixtureAudience,
		Clock:    func() time.Time { return now },
	}, StaticKeySource{fixtureKID: publicKey})
	if err != nil {
		t.Fatal(err)
	}
	return validator
}

type fixtureTokenOptions struct {
	Issuer   string
	Audience string
	OID      string
	TID      string
	Sub      string
	KID      string
	Exp      time.Time
	NBF      time.Time
}

func fixtureClaims(options fixtureTokenOptions) accessTokenClaims {
	issuer := options.Issuer
	if issuer == "" {
		issuer = fixtureIssuer
	}
	audience := options.Audience
	if audience == "" {
		audience = fixtureAudience
	}
	oid := options.OID
	if oid == "" {
		oid = "oid-fixture"
	}
	tid := options.TID
	if tid == "" {
		tid = fixtureTenant
	}
	sub := options.Sub
	if sub == "" {
		sub = "subject-fixture"
	}
	return accessTokenClaims{
		OID: oid,
		TID: tid,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Audience:  jwt.ClaimStrings{audience},
			Subject:   sub,
			ExpiresAt: jwt.NewNumericDate(options.Exp),
			NotBefore: numericDate(options.NBF),
		},
	}
}

func numericDate(value time.Time) *jwt.NumericDate {
	if value.IsZero() {
		return nil
	}
	return jwt.NewNumericDate(value)
}

func signedFixtureToken(t *testing.T, privateKey *rsa.PrivateKey, options fixtureTokenOptions) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, fixtureClaims(options))
	kid := options.KID
	if kid == "" {
		kid = fixtureKID
	}
	token.Header["kid"] = kid
	raw, err := token.SignedString(privateKey)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func newTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	return key
}

type dependencyKeySource struct{}

func (dependencyKeySource) Key(context.Context, string) (crypto.PublicKey, error) {
	return nil, ErrAuthenticationDependency
}
