package authentication

import (
	"context"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const maxOIDCMetadataBodyBytes = 1 << 20

type OIDCDiscoveryConfig struct {
	URL    string
	Client *http.Client
	TTL    time.Duration
	Clock  func() time.Time
}

type oidcMetadataDocument struct {
	Issuer  string `json:"issuer"`
	JWKSURI string `json:"jwks_uri"`
}

// OIDCKeySource discovers the trusted issuer and signing-key endpoint from a
// single configured OpenID Connect metadata URL.
type OIDCKeySource struct {
	url    string
	client *http.Client
	ttl    time.Duration
	clock  func() time.Time

	mu                 sync.Mutex
	metadata           oidcMetadataDocument
	fetchedAt          time.Time
	lastRefreshAttempt time.Time
	lastRefreshFailed  bool
	jwksURL            string
	keys               *JWKSKeySource
}

func NewOIDCKeySource(config OIDCDiscoveryConfig) (*OIDCKeySource, error) {
	config.URL = strings.TrimSpace(config.URL)
	if !validOIDCURL(config.URL) {
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
	return &OIDCKeySource{
		url:    config.URL,
		client: config.Client,
		ttl:    config.TTL,
		clock:  config.Clock,
	}, nil
}

func (source *OIDCKeySource) Issuer(ctx context.Context) (string, error) {
	metadata, err := source.getMetadata(ctx)
	if err != nil {
		return "", err
	}
	return metadata.Issuer, nil
}

func (source *OIDCKeySource) Key(ctx context.Context, kid string) (crypto.PublicKey, error) {
	metadata, err := source.getMetadata(ctx)
	if err != nil {
		return nil, err
	}

	source.mu.Lock()
	keys := source.keys
	if keys == nil || source.jwksURL != metadata.JWKSURI {
		keys, err = NewJWKSKeySource(JWKSConfig{
			URL:    metadata.JWKSURI,
			Client: source.client,
			TTL:    source.ttl,
			Clock:  source.clock,
		})
		if err == nil {
			source.jwksURL = metadata.JWKSURI
			source.keys = keys
		}
	}
	source.mu.Unlock()
	if err != nil {
		return nil, ErrAuthenticationDependency
	}
	return keys.Key(ctx, kid)
}

func (source *OIDCKeySource) getMetadata(ctx context.Context) (oidcMetadataDocument, error) {
	if source == nil {
		return oidcMetadataDocument{}, ErrAuthenticationDependency
	}
	if ctx == nil {
		ctx = context.Background()
	}
	now := source.clock()

	source.mu.Lock()
	defer source.mu.Unlock()
	if source.metadata.Issuer != "" && now.Sub(source.fetchedAt) < source.ttl {
		return source.metadata, nil
	}
	if !source.lastRefreshAttempt.IsZero() && now.Sub(source.lastRefreshAttempt) < source.ttl && source.lastRefreshFailed {
		return oidcMetadataDocument{}, ErrAuthenticationDependency
	}
	source.lastRefreshAttempt = now
	metadata, err := source.fetchMetadata(ctx)
	if err != nil {
		source.lastRefreshFailed = true
		return oidcMetadataDocument{}, fmt.Errorf("%w: %v", ErrAuthenticationDependency, err)
	}
	if source.jwksURL != "" && source.jwksURL != metadata.JWKSURI {
		source.keys = nil
	}
	source.metadata = metadata
	source.jwksURL = metadata.JWKSURI
	source.fetchedAt = now
	source.lastRefreshFailed = false
	return metadata, nil
}

func (source *OIDCKeySource) fetchMetadata(ctx context.Context) (oidcMetadataDocument, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, source.url, nil)
	if err != nil {
		return oidcMetadataDocument{}, err
	}
	response, err := source.client.Do(request)
	if err != nil {
		return oidcMetadataDocument{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return oidcMetadataDocument{}, fmt.Errorf("metadata returned status %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxOIDCMetadataBodyBytes+1))
	if err != nil {
		return oidcMetadataDocument{}, err
	}
	if len(body) > maxOIDCMetadataBodyBytes {
		return oidcMetadataDocument{}, errors.New("metadata document exceeds size limit")
	}
	var metadata oidcMetadataDocument
	if err := json.Unmarshal(body, &metadata); err != nil {
		return oidcMetadataDocument{}, errors.New("invalid metadata document")
	}
	metadata.Issuer = strings.TrimSpace(metadata.Issuer)
	metadata.JWKSURI = strings.TrimSpace(metadata.JWKSURI)
	if !validOIDCURL(metadata.Issuer) || !validOIDCURL(metadata.JWKSURI) {
		return oidcMetadataDocument{}, errors.New("metadata issuer or signing-key URL is invalid")
	}
	return metadata, nil
}

func validOIDCURL(value string) bool {
	parsed, err := url.ParseRequestURI(strings.TrimSpace(value))
	if err != nil || parsed.Host == "" || parsed.User != nil || parsed.Fragment != "" {
		return false
	}
	if parsed.Scheme == "https" {
		return true
	}
	if parsed.Scheme != "http" {
		return false
	}
	host := parsed.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
