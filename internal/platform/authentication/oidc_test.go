package authentication

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestOIDCKeySourceDiscoversIssuerAndKeysForValidation(t *testing.T) {
	privateKey := newTestRSAKey(t)
	issuer := "https://tenant.example.ciamlogin.com/tenant-id/v2.0"
	var metadataRequests atomic.Int32
	var jwksRequests atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Type", "application/json")
		switch request.URL.Path {
		case "/.well-known/openid-configuration":
			metadataRequests.Add(1)
			_, _ = fmt.Fprintf(writer, `{"issuer":%q,"jwks_uri":%q}`, issuer, server.URL+"/keys")
		case "/keys":
			jwksRequests.Add(1)
			_, _ = writer.Write([]byte(jwksJSON(fixtureKID, &privateKey.PublicKey)))
		default:
			http.NotFound(writer, request)
		}
	}))
	defer server.Close()

	source, err := NewOIDCKeySource(OIDCDiscoveryConfig{
		URL: server.URL + "/.well-known/openid-configuration",
		TTL: time.Hour,
	})
	if err != nil {
		t.Fatal(err)
	}
	validator, err := NewValidator(Config{
		IssuerSource: source,
		TenantID:     "tenant-id",
		Audience:     fixtureAudience,
		Clock:        func() time.Time { return time.Date(2026, 9, 23, 8, 30, 0, 0, time.UTC) },
	}, source)
	if err != nil {
		t.Fatal(err)
	}

	raw := signedFixtureToken(t, privateKey, fixtureTokenOptions{
		Issuer: issuer,
		TID:    "tenant-id",
		Exp:    time.Date(2026, 9, 23, 8, 40, 0, 0, time.UTC),
	})
	subject, err := validator.ValidateAccessToken(context.Background(), raw)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if subject.OID != "oid-fixture" || subject.TID != "tenant-id" || subject.Sub != "subject-fixture" {
		t.Fatalf("subject = %#v", subject)
	}
	if _, err := source.Issuer(context.Background()); err != nil {
		t.Fatalf("second Issuer() error = %v", err)
	}
	if _, err := source.Key(context.Background(), fixtureKID); err != nil {
		t.Fatalf("second Key() error = %v", err)
	}
	if metadataRequests.Load() != 1 || jwksRequests.Load() != 1 {
		t.Fatalf("metadata requests = %d, JWKS requests = %d", metadataRequests.Load(), jwksRequests.Load())
	}
}

func TestOIDCKeySourceFailsClosedOnUnavailableOrInvalidMetadata(t *testing.T) {
	tests := []struct {
		name string
		body string
		code int
	}{
		{name: "provider unavailable", body: "provider failure", code: http.StatusBadGateway},
		{name: "invalid JSON", body: "not-json", code: http.StatusOK},
		{name: "missing signing key URL", body: `{"issuer":"https://issuer.example"}`, code: http.StatusOK},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
				writer.WriteHeader(test.code)
				_, _ = writer.Write([]byte(test.body))
			}))
			defer server.Close()
			source, err := NewOIDCKeySource(OIDCDiscoveryConfig{URL: server.URL})
			if err != nil {
				t.Fatal(err)
			}
			_, err = source.Issuer(context.Background())
			if !errors.Is(err, ErrAuthenticationDependency) {
				t.Fatalf("error = %v, want dependency error", err)
			}
			if len(err.Error()) > 256 || err.Error() == "" {
				t.Fatalf("unsafe dependency error length: %d", len(err.Error()))
			}
		})
	}
}

func TestNewOIDCKeySourceRejectsNonLoopbackHTTP(t *testing.T) {
	if _, err := NewOIDCKeySource(OIDCDiscoveryConfig{URL: "http://provider.example/.well-known/openid-configuration"}); !errors.Is(err, ErrInvalidAuthenticationConfig) {
		t.Fatalf("error = %v, want invalid configuration", err)
	}
}
