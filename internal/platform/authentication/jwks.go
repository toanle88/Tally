package authentication

import (
	"context"
	"crypto"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	defaultJWKSCacheTTL       = 15 * time.Minute
	defaultJWKSRequestTimeout = 5 * time.Second
	maxJWKSBodyBytes          = 1 << 20
)

var (
	ErrInvalidJWKS = errors.New("invalid jwks document")
	ErrUnknownKey  = errors.New("unknown signing key")
)

type JWKSConfig struct {
	URL    string
	Client *http.Client
	TTL    time.Duration
	Clock  func() time.Time
}

type JWKSKeySource struct {
	url    string
	client *http.Client
	ttl    time.Duration
	clock  func() time.Time

	mu                 sync.Mutex
	keys               map[string]crypto.PublicKey
	fetchedAt          time.Time
	lastRefreshAttempt time.Time
	lastRefreshFailed  bool
}

type jwksDocument struct {
	Keys []jwkKey `json:"keys"`
}

type jwkKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Use string `json:"use"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func NewJWKSKeySource(config JWKSConfig) (*JWKSKeySource, error) {
	config.URL = strings.TrimSpace(config.URL)
	if config.URL == "" {
		return nil, ErrInvalidAuthenticationConfig
	}
	if config.Client == nil {
		config.Client = &http.Client{Timeout: defaultJWKSRequestTimeout}
	}
	if config.TTL <= 0 {
		config.TTL = defaultJWKSCacheTTL
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	return &JWKSKeySource{
		url:    config.URL,
		client: config.Client,
		ttl:    config.TTL,
		clock:  config.Clock,
		keys:   make(map[string]crypto.PublicKey),
	}, nil
}

func (source *JWKSKeySource) Key(ctx context.Context, kid string) (crypto.PublicKey, error) {
	if source == nil || strings.TrimSpace(kid) == "" {
		return nil, ErrUnknownKey
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := source.clock()

	source.mu.Lock()
	if key, ok := source.keys[kid]; ok {
		source.mu.Unlock()
		return key, nil
	}
	if !source.lastRefreshAttempt.IsZero() && now.Sub(source.lastRefreshAttempt) < source.ttl {
		if source.lastRefreshFailed {
			source.mu.Unlock()
			return nil, ErrAuthenticationDependency
		}
		source.mu.Unlock()
		return nil, ErrUnknownKey
	}
	source.lastRefreshAttempt = now
	source.mu.Unlock()

	keys, err := source.fetch(ctx)
	if err != nil {
		source.mu.Lock()
		source.lastRefreshFailed = true
		source.mu.Unlock()
		return nil, fmt.Errorf("%w: %v", ErrAuthenticationDependency, err)
	}
	source.mu.Lock()
	source.keys = keys
	source.fetchedAt = now
	source.lastRefreshFailed = false
	key, ok := source.keys[kid]
	source.mu.Unlock()
	if !ok {
		return nil, ErrUnknownKey
	}
	return key, nil
}

func (source *JWKSKeySource) fetch(ctx context.Context) (map[string]crypto.PublicKey, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source.url, nil)
	if err != nil {
		return nil, err
	}
	response, err := source.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxJWKSBodyBytes+1))
	if err != nil {
		return nil, err
	}
	if len(body) > maxJWKSBodyBytes {
		return nil, ErrInvalidJWKS
	}
	var document jwksDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidJWKS, err)
	}
	keys := make(map[string]crypto.PublicKey, len(document.Keys))
	for _, key := range document.Keys {
		parsed, err := parseRSAKey(key)
		if err != nil {
			return nil, err
		}
		if _, exists := keys[key.Kid]; exists {
			return nil, fmt.Errorf("%w: duplicate kid", ErrInvalidJWKS)
		}
		keys[key.Kid] = parsed
	}
	if len(keys) == 0 {
		return nil, ErrInvalidJWKS
	}
	return keys, nil
}

func parseRSAKey(key jwkKey) (*rsa.PublicKey, error) {
	if key.Kid == "" || key.Kty != "RSA" || key.N == "" || key.E == "" {
		return nil, ErrInvalidJWKS
	}
	modulus, err := base64.RawURLEncoding.DecodeString(key.N)
	if err != nil || len(modulus) == 0 {
		return nil, ErrInvalidJWKS
	}
	exponent, err := base64.RawURLEncoding.DecodeString(key.E)
	if err != nil || len(exponent) == 0 || len(exponent) > 4 {
		return nil, ErrInvalidJWKS
	}
	exponentInt := 0
	for _, value := range exponent {
		exponentInt = exponentInt<<8 | int(value)
	}
	if exponentInt < 2 {
		return nil, ErrInvalidJWKS
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(modulus), E: exponentInt}, nil
}

type StaticKeySource map[string]crypto.PublicKey

func (source StaticKeySource) Key(_ context.Context, kid string) (crypto.PublicKey, error) {
	key, ok := source[kid]
	if !ok {
		return nil, ErrUnknownKey
	}
	return key, nil
}
