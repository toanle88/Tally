package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const identityOperationID = "identity.manage-users.v1"

func newIdentityAPIServerWithPostgres(getenv func(string) string, pool *pgxpool.Pool, auditWriter identity.PostgresAuditWriter) http.Handler {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if pool == nil || auditWriter == nil {
		return identityUnavailableHandler("identity persistence or audit integration unavailable")
	}
	repository, err := identity.NewPostgresUserRepositoryWithAudit(pool, auditWriter)
	if err != nil {
		return identityUnavailableHandler("identity persistence unavailable")
	}
	return newIdentityAPIServerWithPostgresRepository(getenv, pool, repository, nil)
}

func newIdentityAPIServerWithPostgresRepository(getenv func(string) string, pool *pgxpool.Pool, repository *identity.PostgresUserRepository, rolePolicy identity.RolePolicyLookup) http.Handler {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if pool == nil || repository == nil {
		return identityUnavailableHandler("identity persistence unavailable")
	}
	if rolePolicy == nil {
		return identityUnavailableHandler("identity role and policy integration unavailable")
	}
	service, err := identity.NewUserServiceWithDurableIdempotency(
		repository,
		environmentUserAuthorizer{getenv: getenv},
		&identity.MemoryAuditRecorder{},
		time.Now,
		identity.DurableUserServiceConfig{
			Database:    pool,
			Coordinator: platformidempotency.NewPostgresCoordinator(),
			Policy: platformidempotency.IdempotencyPolicy{
				RecordTTL: 24 * time.Hour,
				LeaseTTL:  5 * time.Minute,
			},
			OperationID: identityOperationID,
		},
		rolePolicy,
	)
	if err != nil {
		return identityUnavailableHandler("identity service unavailable")
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{Service: service},
		apiBearerSecurityHandler{},
	)
	if err != nil {
		return identityUnavailableHandler("identity API unavailable")
	}
	return server
}

func identityUnavailableHandler(message string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, message, http.StatusServiceUnavailable)
	})
}

func newIdentityAPIServer(getenv func(string) string) http.Handler {
	return newIdentityAPIServerWithRepository(getenv, identity.NewMemoryUserRepository())
}

func newIdentityAPIServerWithRepository(getenv func(string) string, repository identity.UserRepository) http.Handler {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	service, err := identity.NewUserService(
		repository,
		environmentUserAuthorizer{getenv: getenv},
		&identity.MemoryAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "identity service unavailable", http.StatusServiceUnavailable)
		})
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{Service: service},
		apiBearerSecurityHandler{},
	)
	if err != nil {
		return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			http.Error(writer, "identity API unavailable", http.StatusServiceUnavailable)
		})
	}
	return server
}

type apiBearerSecurityHandler struct{}

func (apiBearerSecurityHandler) HandleBearerAuth(ctx context.Context, _ generated.OperationName, token generated.BearerAuth) (context.Context, error) {
	if strings.TrimSpace(token.Token) == "" {
		return ctx, errors.New("bearer token is empty")
	}
	// The authentication middleware has already validated the token and
	// resolved the application actor before the generated server is reached.
	return ctx, nil
}

type environmentUserAuthorizer struct {
	getenv func(string) string
}

func (authorizer environmentUserAuthorizer) AuthorizeUserManagement(_ context.Context, _ identity.ApplicationActor, _ identity.UserCommand, _ *identity.User) (identity.AuthorizationDecision, error) {
	allowed := strings.EqualFold(strings.TrimSpace(authorizer.getenv("TALLY_IAM_ALLOW_USER_MANAGEMENT")), "true")
	if !allowed {
		return identity.AuthorizationDecision{
			Allowed:           false,
			Permission:        identity.UserManagementPermission,
			DecisionReference: uuid.New(),
			Reason:            "user-management policy is not enabled",
		}, nil
	}
	rawScopes := strings.TrimSpace(authorizer.getenv("TALLY_IAM_APPROVED_SCOPES"))
	scopes := []string{"*"}
	if rawScopes != "" {
		scopes = nil
		for _, scope := range strings.Split(rawScopes, ",") {
			if value := strings.TrimSpace(scope); value != "" {
				scopes = append(scopes, value)
			}
		}
	}
	return identity.AuthorizationDecision{
		Allowed:           true,
		Permission:        identity.UserManagementPermission,
		ApprovedScopeIDs:  scopes,
		PolicyReference:   "environment-configured-user-management-policy",
		DecisionReference: uuid.New(),
	}, nil
}
