// Package authentication contains the technical bearer-token boundary. It
// validates enterprise access tokens and delegates application-user mapping
// to the owning identity module.
package authentication

import (
	"context"
	"crypto"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/toanle88/Tally/internal/identity"
)

const ApprovedClockSkew = 120 * time.Second

var (
	ErrInvalidToken                 = errors.New("invalid access token")
	ErrAuthenticationDependency     = errors.New("authentication dependency unavailable")
	ErrInvalidAuthenticationConfig  = errors.New("invalid authentication configuration")
	ErrAuthenticationKeyUnavailable = errors.New("authentication signing key unavailable")
)

type KeySource interface {
	Key(context.Context, string) (crypto.PublicKey, error)
}

type IssuerSource interface {
	Issuer(context.Context) (string, error)
}

type Config struct {
	Issuer            string
	IssuerSource      IssuerSource
	TenantID          string
	Audience          string
	AllowedAlgorithms []string
	Clock             func() time.Time
}

type Validator struct {
	config     Config
	keys       KeySource
	issuer     IssuerSource
	parser     *jwt.Parser
	algorithms map[string]struct{}
}

type accessTokenClaims struct {
	OID      string           `json:"oid"`
	TID      string           `json:"tid"`
	ACR      string           `json:"acr"`
	AMR      []string         `json:"amr"`
	AuthTime *jwt.NumericDate `json:"auth_time,omitempty"`
	jwt.RegisteredClaims
}

func NewValidator(config Config, keys KeySource) (*Validator, error) {
	config.Issuer = strings.TrimSpace(config.Issuer)
	config.TenantID = strings.TrimSpace(config.TenantID)
	config.Audience = strings.TrimSpace(config.Audience)
	if (config.Issuer == "") == (config.IssuerSource == nil) || config.TenantID == "" || config.Audience == "" || keys == nil {
		return nil, ErrInvalidAuthenticationConfig
	}
	if config.Clock == nil {
		config.Clock = time.Now
	}
	if len(config.AllowedAlgorithms) == 0 {
		config.AllowedAlgorithms = []string{jwt.SigningMethodRS256.Alg()}
	}

	algorithms := make(map[string]struct{}, len(config.AllowedAlgorithms))
	for _, algorithm := range config.AllowedAlgorithms {
		algorithm = strings.TrimSpace(algorithm)
		if algorithm == "" {
			return nil, ErrInvalidAuthenticationConfig
		}
		algorithms[algorithm] = struct{}{}
	}

	parserOptions := []jwt.ParserOption{
		jwt.WithValidMethods(config.AllowedAlgorithms),
		jwt.WithAudience(config.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(ApprovedClockSkew),
		jwt.WithTimeFunc(config.Clock),
	}
	return &Validator{
		issuer:     config.IssuerSource,
		config:     config,
		keys:       keys,
		parser:     jwt.NewParser(parserOptions...),
		algorithms: algorithms,
	}, nil
}

func (validator *Validator) ValidateAccessToken(ctx context.Context, raw string) (identity.AuthenticationSubject, error) {
	if validator == nil || validator.parser == nil || validator.keys == nil {
		return identity.AuthenticationSubject{}, ErrInvalidAuthenticationConfig
	}
	if ctx == nil {
		ctx = context.Background()
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return identity.AuthenticationSubject{}, ErrInvalidToken
	}
	expectedIssuer := validator.config.Issuer
	if validator.issuer != nil {
		var err error
		expectedIssuer, err = validator.issuer.Issuer(ctx)
		if err != nil || strings.TrimSpace(expectedIssuer) == "" {
			return identity.AuthenticationSubject{}, ErrAuthenticationDependency
		}
	}

	dependencyUnavailable := false
	claims := &accessTokenClaims{}
	token, err := validator.parser.ParseWithClaims(raw, claims, func(token *jwt.Token) (any, error) {
		if _, allowed := validator.algorithms[token.Method.Alg()]; !allowed {
			return nil, ErrInvalidToken
		}
		kid, ok := token.Header["kid"].(string)
		if !ok || strings.TrimSpace(kid) == "" {
			return nil, ErrInvalidToken
		}
		key, keyErr := validator.keys.Key(ctx, kid)
		if errors.Is(keyErr, ErrAuthenticationDependency) {
			dependencyUnavailable = true
		}
		if keyErr != nil {
			return nil, ErrAuthenticationKeyUnavailable
		}
		return key, nil
	})
	if dependencyUnavailable {
		return identity.AuthenticationSubject{}, ErrAuthenticationDependency
	}
	if err != nil || token == nil || !token.Valid {
		return identity.AuthenticationSubject{}, ErrInvalidToken
	}
	if claims.Issuer != expectedIssuer || claims.TID != validator.config.TenantID {
		return identity.AuthenticationSubject{}, ErrInvalidToken
	}
	subject, err := identity.NewAuthenticationSubject(claims.OID, claims.TID, claims.Subject)
	if err != nil {
		return identity.AuthenticationSubject{}, ErrInvalidToken
	}
	if claims.AuthTime != nil {
		subject.Assurance.AuthenticatedAt = claims.AuthTime.Time.UTC()
	}
	subject.Assurance.AssuranceLevel = strings.TrimSpace(claims.ACR)
	subject.Assurance.Methods = strings.Join(claims.AMR, ",")
	return subject, nil
}
