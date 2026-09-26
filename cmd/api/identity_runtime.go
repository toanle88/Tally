package main

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/platform/httpapi"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	identityOperationID = "identity.manage-users.v1"
	roleOperationID     = "identity.manage-roles.v1"
)

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
	roleRepository, err := identity.NewPostgresRoleRepositoryWithAudit(pool, postgresRoleAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("identity role persistence unavailable")
	}
	return newIdentityAPIServerWithPostgresRepository(getenv, pool, repository, roleRepository)
}

func newIdentityAPIServerWithPostgresRepository(getenv func(string) string, pool *pgxpool.Pool, repository *identity.PostgresUserRepository, roleRepository *identity.PostgresRoleRepository) http.Handler {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if pool == nil || repository == nil || roleRepository == nil {
		return identityUnavailableHandler("identity persistence or role integration unavailable")
	}
	userService, err := identity.NewUserServiceWithDurableIdempotency(
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
		roleRepository,
	)
	if err != nil {
		return identityUnavailableHandler("identity user service unavailable")
	}
	roleService, err := identity.NewRoleServiceWithDurableIdempotency(
		roleRepository,
		environmentRoleAuthorizer{getenv: getenv},
		environmentRoleApproval{getenv: getenv},
		environmentRoleSegregation{getenv: getenv},
		&identity.MemoryRoleAuditRecorder{},
		time.Now,
		identity.DurableRoleServiceConfig{
			Database:    pool,
			Coordinator: platformidempotency.NewPostgresCoordinator(),
			Policy: platformidempotency.IdempotencyPolicy{
				RecordTTL: 24 * time.Hour,
				LeaseTTL:  5 * time.Minute,
			},
			OperationID: roleOperationID,
		},
	)
	if err != nil {
		return identityUnavailableHandler("identity role service unavailable")
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{Service: userService, RoleService: roleService},
		apiBearerSecurityHandler{},
	)
	if err != nil {
		return identityUnavailableHandler("identity API unavailable")
	}
	return server
}

func postgresRoleAuditWriter(auditWriter identity.PostgresAuditWriter) identity.PostgresRoleAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record identity.RoleAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, identity.ErrRoleAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        record.RoleID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorAuthenticationRef,
			Action:                        record.Action,
			ScopeIDs:                      append([]string(nil), record.ScopeIDs...),
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyReference,
			DecisionReference:             record.DecisionReference,
			ApprovalRequestID:             record.ApprovalRequestID,
			ApprovalDecisionID:            record.ApprovalDecisionID,
			ApproverUserID:                record.ApproverUserID,
			RevisionVersion:               record.RoleVersion,
			BeforeFingerprint:             record.BeforeFingerprint,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
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
	roleRepository := identity.NewMemoryRoleRepository()
	userService, err := identity.NewUserService(
		repository,
		environmentUserAuthorizer{getenv: getenv},
		&identity.MemoryAuditRecorder{},
		time.Now,
		roleRepository,
	)
	if err != nil {
		return identityUnavailableHandler("identity service unavailable")
	}
	roleService, err := identity.NewRoleService(
		roleRepository,
		environmentRoleAuthorizer{getenv: getenv},
		identity.AllowAllRoleApprovalPort{},
		identity.AllowAllRoleSegregationPort{},
		&identity.MemoryRoleAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("identity role service unavailable")
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{Service: userService, RoleService: roleService},
		apiBearerSecurityHandler{},
	)
	if err != nil {
		return identityUnavailableHandler("identity API unavailable")
	}
	return server
}

type apiBearerSecurityHandler struct{}

func (apiBearerSecurityHandler) HandleBearerAuth(ctx context.Context, _ generated.OperationName, token generated.BearerAuth) (context.Context, error) {
	if strings.TrimSpace(token.Token) == "" {
		return ctx, errors.New("bearer token is empty")
	}
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
	return identity.AuthorizationDecision{
		Allowed:           true,
		Permission:        identity.UserManagementPermission,
		ApprovedScopeIDs:  configuredScopes(authorizer.getenv),
		PolicyReference:   "environment-configured-user-management-policy",
		DecisionReference: uuid.New(),
	}, nil
}

type environmentRoleAuthorizer struct {
	getenv func(string) string
}

func (authorizer environmentRoleAuthorizer) AuthorizeRoleManagement(_ context.Context, _ identity.ApplicationActor, _ identity.RoleCommand, _ *identity.Role) (identity.AuthorizationDecision, error) {
	allowed := strings.EqualFold(strings.TrimSpace(authorizer.getenv("TALLY_IAM_ALLOW_ROLE_MANAGEMENT")), "true")
	if !allowed {
		return identity.AuthorizationDecision{
			Allowed:           false,
			Permission:        identity.RoleManagementPermission,
			DecisionReference: uuid.New(),
			Reason:            "role-management policy is not enabled",
		}, nil
	}
	return identity.AuthorizationDecision{
		Allowed:           true,
		Permission:        identity.RoleManagementPermission,
		ApprovedScopeIDs:  configuredScopes(authorizer.getenv),
		PolicyReference:   "environment-configured-role-management-policy",
		DecisionReference: uuid.New(),
	}, nil
}

type environmentRoleApproval struct {
	getenv func(string) string
}

func (approval environmentRoleApproval) ValidateRoleApproval(ctx context.Context, actor identity.ApplicationActor, command identity.RoleCommand, current *identity.Role, fingerprint string) error {
	if !strings.EqualFold(strings.TrimSpace(approval.getenv("TALLY_IAM_ALLOW_ROLE_APPROVAL")), "true") {
		return identity.ErrApprovalUnavailable
	}
	return identity.AllowAllRoleApprovalPort{}.ValidateRoleApproval(ctx, actor, command, current, fingerprint)
}

type environmentRoleSegregation struct {
	getenv func(string) string
}

func (segregation environmentRoleSegregation) ValidateRoleGrants(_ context.Context, _ []identity.PermissionGrant) error {
	if !strings.EqualFold(strings.TrimSpace(segregation.getenv("TALLY_IAM_ALLOW_ROLE_SEGREGATION")), "true") {
		return identity.ErrSegregationUnavailable
	}
	return nil
}

func configuredScopes(getenv func(string) string) []string {
	rawScopes := strings.TrimSpace(getenv("TALLY_IAM_APPROVED_SCOPES"))
	if rawScopes == "" {
		return []string{"*"}
	}
	scopes := make([]string, 0)
	for _, scope := range strings.Split(rawScopes, ",") {
		if value := strings.TrimSpace(scope); value != "" {
			scopes = append(scopes, value)
		}
	}
	return scopes
}
