package authentication

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestJWKSKeySourceFetchesAndCachesRSAKeys(t *testing.T) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(jwksJSON("fixture-key", &privateKey.PublicKey)))
	}))
	defer server.Close()

	source, err := NewJWKSKeySource(JWKSConfig{URL: server.URL, TTL: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	first, err := source.Key(context.Background(), "fixture-key")
	if err != nil {
		t.Fatalf("first Key() error = %v", err)
	}
	second, err := source.Key(context.Background(), "fixture-key")
	if err != nil {
		t.Fatalf("second Key() error = %v", err)
	}
	if first == nil || second == nil || requests.Load() != 1 {
		t.Fatalf("first = %T, second = %T, requests = %d", first, second, requests.Load())
	}
	if _, err := source.Key(context.Background(), "unknown-key"); !errors.Is(err, ErrUnknownKey) {
		t.Fatalf("unknown key error = %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("unknown key unexpectedly refreshed JWKS, requests = %d", requests.Load())
	}
}

func TestJWKSKeySourceReturnsDependencyErrorOnUnavailableDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, "provider failure", http.StatusBadGateway)
	}))
	defer server.Close()

	source, err := NewJWKSKeySource(JWKSConfig{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.Key(context.Background(), "fixture-key")
	if !errors.Is(err, ErrAuthenticationDependency) {
		t.Fatalf("error = %v, want dependency error", err)
	}
	if len(err.Error()) > 256 || err.Error() == "" {
		t.Fatalf("unexpected dependency error detail length: %d", len(err.Error()))
	}
}

func TestJWKSKeySourceRejectsInvalidDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = writer.Write([]byte(`{"keys":[{"kid":"duplicate","kty":"RSA","n":"AQ","e":"AQAB"},{"kid":"duplicate","kty":"RSA","n":"AQ","e":"AQAB"}]}`))
	}))
	defer server.Close()

	source, err := NewJWKSKeySource(JWKSConfig{URL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = source.Key(context.Background(), "duplicate")
	if !errors.Is(err, ErrAuthenticationDependency) {
		t.Fatalf("error = %v, want dependency error for invalid provider document", err)
	}
}

func jwksJSON(kid string, key *rsa.PublicKey) string {
	modulus := base64.RawURLEncoding.EncodeToString(key.N.Bytes())
	exponent := base64.RawURLEncoding.EncodeToString([]byte{byte(key.E >> 16), byte(key.E >> 8), byte(key.E)})
	return `{"keys":[{"kid":"` + kid + `","kty":"RSA","use":"sig","alg":"RS256","n":"` + modulus + `","e":"` + exponent + `"}]}`
}
