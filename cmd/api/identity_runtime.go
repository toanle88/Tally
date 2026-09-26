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
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

const (
	identityOperationID    = "identity.manage-users.v1"
	roleOperationID        = "identity.manage-roles.v1"
	segregationOperationID = "identity.manage-segregation-rules.v1"
)

func newIdentityAPIServerWithPostgres(getenv func(string) string, pool *pgxpool.Pool, auditWriter identity.PostgresAuditWriter, instrumentation ...*telemetry.Instrumentation) http.Handler {
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
	return newIdentityAPIServerWithPostgresRepository(getenv, pool, repository, roleRepository, auditWriter, instrumentation...)
}

func newIdentityAPIServerWithPostgresRepository(getenv func(string) string, pool *pgxpool.Pool, repository *identity.PostgresUserRepository, roleRepository *identity.PostgresRoleRepository, auditWriter identity.PostgresAuditWriter, instrumentation ...*telemetry.Instrumentation) http.Handler {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if pool == nil || repository == nil || roleRepository == nil || auditWriter == nil {
		return identityUnavailableHandler("identity persistence or role integration unavailable")
	}
	policyStore, err := identity.NewPostgresAccessPolicyStore(pool)
	if err != nil {
		return identityUnavailableHandler("identity policy persistence unavailable")
	}
	policyEvaluator, err := identity.NewPolicyEvaluator(policyStore, time.Now, authorizationDecisionObserver(instrumentation...))
	if err != nil {
		return identityUnavailableHandler("identity policy evaluator unavailable")
	}
	authorizer := evaluatorIdentityAuthorizer{evaluator: policyEvaluator}
	segregationRepository, err := identity.NewPostgresSegregationRuleRepositoryWithAudit(pool, postgresSegregationRuleAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule persistence unavailable")
	}
	segregationEvaluator, err := identity.NewSegregationEvaluator(segregationRepository, time.Now, segregationDecisionObserver(instrumentation...))
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule evaluator unavailable")
	}
	segregationAudit := &identity.MemorySegregationRuleAuditRecorder{}
	segregationService, err := identity.NewSegregationRuleServiceWithDurableIdempotency(segregationRepository, authorizer, identity.AllowAllSegregationRuleApprovalPort{}, segregationAudit, time.Now, identity.DurableSegregationRuleServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: segregationOperationID})
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule service unavailable")
	}
	userService, err := identity.NewUserServiceWithDurableIdempotency(
		repository,
		authorizer,
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
		authorizer,
		environmentRoleApproval{getenv: getenv},
		segregationEvaluator,
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
		httpapi.IdentityHandler{Service: userService, RoleService: roleService, SegregationRuleService: segregationService},
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
			PolicyVersion:                 record.PolicyVersion,
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

func postgresSegregationRuleAuditWriter(auditWriter identity.PostgresAuditWriter) identity.PostgresSegregationRuleAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record identity.SegregationRuleAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, identity.ErrSegregationRuleAudit
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.RuleID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorAuthenticationRef, Action: record.Action, ScopeIDs: append([]string(nil), record.ScopeIDs...), Permission: identity.SegregationRuleManagementPermission, PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference, ApprovalRequestID: record.ApprovalRequestID, ApprovalDecisionID: record.ApprovalDecisionID, ApproverUserID: record.ApproverUserID, RevisionVersion: record.RuleVersion, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint, CorrelationID: record.CorrelationID, CausationID: record.CausationID,
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

func newIdentityAPIServerWithRepository(getenv func(string) string, repository identity.UserRepository, instrumentation ...*telemetry.Instrumentation) http.Handler {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	roleRepository := identity.NewMemoryRoleRepository()
	authorizer, err := newEnvironmentIdentityAuthorizer(getenv, instrumentation...)
	if err != nil {
		return identityUnavailableHandler("identity policy evaluator unavailable")
	}
	segregationRepository, err := identity.NewMemorySegregationRuleRepository(identity.DefaultSegregationRules(time.Now())...)
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule persistence unavailable")
	}
	segregationEvaluator, err := identity.NewSegregationEvaluator(segregationRepository, time.Now, segregationDecisionObserver(instrumentation...))
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule evaluator unavailable")
	}
	segregationAudit := &identity.MemorySegregationRuleAuditRecorder{}
	segregationService, err := identity.NewSegregationRuleService(segregationRepository, authorizer, identity.AllowAllSegregationRuleApprovalPort{}, segregationAudit, time.Now)
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule service unavailable")
	}
	userService, err := identity.NewUserService(
		repository,
		authorizer,
		&identity.MemoryAuditRecorder{},
		time.Now,
		roleRepository,
	)
	if err != nil {
		return identityUnavailableHandler("identity service unavailable")
	}
	roleService, err := identity.NewRoleService(
		roleRepository,
		authorizer,
		identity.AllowAllRoleApprovalPort{},
		segregationEvaluator,
		&identity.MemoryRoleAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("identity role service unavailable")
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{Service: userService, RoleService: roleService, SegregationRuleService: segregationService},
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

type evaluatorIdentityAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

func (authorizer evaluatorIdentityAuthorizer) AuthorizeUserManagement(ctx context.Context, actor identity.ApplicationActor, command identity.UserCommand, current *identity.User) (identity.AuthorizationDecision, error) {
	assignments := command.Assignments
	if assignments == nil && current != nil {
		assignments = current.Assignments
	}
	return authorizer.evaluate(ctx, actor, identity.UserManagementPermission, requestedScopesFromAssignments(assignments))
}

func (authorizer evaluatorIdentityAuthorizer) AuthorizeRoleManagement(ctx context.Context, actor identity.ApplicationActor, command identity.RoleCommand, current *identity.Role) (identity.AuthorizationDecision, error) {
	grants := command.Grants
	if command.Action == identity.RoleActionRetire && current != nil {
		grants = current.Grants
	}
	return authorizer.evaluate(ctx, actor, identity.RoleManagementPermission, requestedScopesFromGrants(grants))
}

func (authorizer evaluatorIdentityAuthorizer) AuthorizeSegregationRuleManagement(ctx context.Context, actor identity.ApplicationActor, command identity.SegregationRuleCommand, current *identity.SegregationRule) (identity.AuthorizationDecision, error) {
	scopes := append([]string(nil), command.ScopeIDs...)
	if len(scopes) == 0 && current != nil {
		scopes = append(scopes, current.ScopeIDs...)
	}
	return authorizer.evaluate(ctx, actor, identity.SegregationRuleManagementPermission, scopes)
}

func (authorizer evaluatorIdentityAuthorizer) evaluate(ctx context.Context, actor identity.ApplicationActor, permission string, scopes []string) (identity.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return identity.AuthorizationDecision{Outcome: identity.AuthorizationUnavailable, Permission: permission, DecisionReference: uuid.New(), ReasonCode: identity.AuthorizationReasonPolicyUnavailable}, nil
	}
	return authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{
		ActorID:           actor.UserID,
		Permission:        permission,
		RequestedScopeIDs: scopes,
	})
}

func requestedScopesFromAssignments(assignments []identity.RoleAssignment) []string {
	seen := make(map[string]struct{})
	var scopes []string
	for _, assignment := range assignments {
		for _, scope := range assignment.Scopes {
			value := strings.TrimSpace(scope.ScopeID)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			scopes = append(scopes, value)
		}
	}
	return scopes
}

func requestedScopesFromGrants(grants []identity.PermissionGrant) []string {
	seen := make(map[string]struct{})
	var scopes []string
	for _, grant := range grants {
		for _, scope := range grant.ScopeIDs {
			value := strings.TrimSpace(scope)
			if value == "" {
				continue
			}
			if _, exists := seen[value]; exists {
				continue
			}
			seen[value] = struct{}{}
			scopes = append(scopes, value)
		}
	}
	return scopes
}

func authorizationDecisionObserver(instrumentation ...*telemetry.Instrumentation) identity.AuthorizationDecisionObserver {
	if len(instrumentation) == 0 || instrumentation[0] == nil {
		return nil
	}
	return func(ctx context.Context, decision identity.AuthorizationDecision, err error) {
		outcome := string(decision.Outcome)
		if outcome == "" {
			outcome = string(identity.AuthorizationDenied)
		}
		if err != nil {
			outcome = "internal_failure"
		}
		instrumentation[0].RecordAuthorizationDecision(ctx, outcome)
	}
}

func segregationDecisionObserver(instrumentation ...*telemetry.Instrumentation) identity.SegregationDecisionObserver {
	if len(instrumentation) == 0 || instrumentation[0] == nil {
		return nil
	}
	return func(ctx context.Context, decision identity.SegregationDecision, err error) {
		outcome := string(decision.Outcome)
		if outcome == "" {
			outcome = string(identity.AuthorizationDenied)
		}
		if err != nil {
			outcome = "internal_failure"
		}
		instrumentation[0].RecordAuthorizationDecision(ctx, outcome)
	}
}

func newEnvironmentIdentityAuthorizer(getenv func(string) string, instrumentation ...*telemetry.Instrumentation) (evaluatorIdentityAuthorizer, error) {
	now := time.Now().UTC().Add(-time.Minute)
	policies := make([]identity.AccessPolicy, 0, 2)
	if strings.EqualFold(strings.TrimSpace(getenv("TALLY_IAM_ALLOW_USER_MANAGEMENT")), "true") {
		policies = append(policies, identity.AccessPolicy{
			ID:            uuid.New(),
			Version:       "environment-user-management-v1",
			Status:        identity.AccessPolicyStatusActive,
			Permissions:   []string{identity.UserManagementPermission},
			EffectiveFrom: now,
			Rules:         []identity.AccessRule{{ScopeIDs: configuredScopes(getenv)}},
		})
	}
	if strings.EqualFold(strings.TrimSpace(getenv("TALLY_IAM_ALLOW_ROLE_MANAGEMENT")), "true") {
		policies = append(policies, identity.AccessPolicy{
			ID:            uuid.New(),
			Version:       "environment-role-management-v1",
			Status:        identity.AccessPolicyStatusActive,
			Permissions:   []string{identity.RoleManagementPermission},
			EffectiveFrom: now,
			Rules:         []identity.AccessRule{{ScopeIDs: configuredScopes(getenv)}},
		})
	}
	if strings.EqualFold(strings.TrimSpace(getenv("TALLY_IAM_ALLOW_SEGREGATION_RULE_MANAGEMENT")), "true") {
		policies = append(policies, identity.AccessPolicy{
			ID: uuid.New(), Version: "environment-segregation-rule-management-v1", Status: identity.AccessPolicyStatusActive,
			Permissions: []string{identity.SegregationRuleManagementPermission}, EffectiveFrom: now,
			Rules: []identity.AccessRule{{ScopeIDs: configuredScopes(getenv)}},
		})
	}
	store, err := identity.NewMemoryAccessPolicyStore(policies...)
	if err != nil {
		return evaluatorIdentityAuthorizer{}, err
	}
	evaluator, err := identity.NewPolicyEvaluator(store, time.Now, authorizationDecisionObserver(instrumentation...))
	if err != nil {
		return evaluatorIdentityAuthorizer{}, err
	}
	return evaluatorIdentityAuthorizer{evaluator: evaluator}, nil
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
