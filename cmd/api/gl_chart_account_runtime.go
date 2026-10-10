package main

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
)

type evaluatorGLChartOfAccountsAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

func (authorizer evaluatorGLChartOfAccountsAuthorizer) AuthorizeChartOfAccounts(ctx context.Context, actor gl.Actor, command gl.ChartOfAccountsCommand, _ *gl.ChartOfAccounts) (gl.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return gl.AuthorizationDecision{}, gl.ErrChartOfAccountsAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: gl.ChartOfAccountsManagementPermission, RequestedScopeIDs: []string{command.AccountingScopeID.String()}})
	if err != nil {
		return gl.AuthorizationDecision{}, gl.ErrChartOfAccountsAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return gl.AuthorizationDecision{}, gl.ErrChartOfAccountsAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return gl.AuthorizationDecision{}, gl.ErrChartOfAccountsAuthorizationStale
	}
	return glAuthorizationDecision(decision), nil
}

type evaluatorGLAccountAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

func (authorizer evaluatorGLAccountAuthorizer) AuthorizeAccount(ctx context.Context, actor gl.Actor, command gl.AccountCommand, _ *gl.Account) (gl.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return gl.AuthorizationDecision{}, gl.ErrAccountAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: gl.AccountManagementPermission, RequestedScopeIDs: []string{command.AccountingScopeID.String()}})
	if err != nil {
		return gl.AuthorizationDecision{}, gl.ErrAccountAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return gl.AuthorizationDecision{}, gl.ErrAccountAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return gl.AuthorizationDecision{}, gl.ErrAccountAuthorizationStale
	}
	return glAuthorizationDecision(decision), nil
}

// runtimeGLChartAccountReferenceValidator keeps cross-context checks behind
// an application boundary. The current repository has no reporting-definition
// module, so a submitted reporting mapping fails closed until that adapter is
// available. GL-owned parent relationships are checked transactionally by the
// GL repository.
type runtimeGLChartAccountReferenceValidator struct {
	reportingMappingsAvailable bool
}

func (validator runtimeGLChartAccountReferenceValidator) ValidateChartOfAccountsReferences(context.Context, gl.Actor, gl.ChartOfAccountsCommand) error {
	return nil
}

func (validator runtimeGLChartAccountReferenceValidator) ValidateAccountReferences(_ context.Context, _ gl.Actor, command gl.AccountCommand) error {
	if len(command.ReportingMappings) > 0 && !validator.reportingMappingsAvailable {
		return gl.ErrAccountReferenceUnavailable
	}
	return nil
}

type permissiveGLChartOfAccountsAuthorizer struct{}

func (permissiveGLChartOfAccountsAuthorizer) AuthorizeChartOfAccounts(context.Context, gl.Actor, gl.ChartOfAccountsCommand, *gl.ChartOfAccounts) (gl.AuthorizationDecision, error) {
	return gl.AuthorizationDecision{Allowed: true, Permission: gl.ChartOfAccountsManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

type permissiveGLAccountAuthorizer struct{}

func (permissiveGLAccountAuthorizer) AuthorizeAccount(context.Context, gl.Actor, gl.AccountCommand, *gl.Account) (gl.AuthorizationDecision, error) {
	return gl.AuthorizationDecision{Allowed: true, Permission: gl.AccountManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func postgresGLChartOfAccountsAuditWriter(auditWriter identity.PostgresAuditWriter) gl.PostgresChartOfAccountsAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record gl.ChartOfAccountsAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, gl.ErrChartOfAccountsAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{UserID: record.ChartOfAccountsID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference, Action: record.Action, ScopeIDs: []string{record.AccountingScopeID.String()}, Permission: record.Permission, PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference, ApprovalRequestID: record.ApprovalRequestID, ApprovalDecisionID: record.ApprovalDecisionID, ApproverUserID: record.ApproverUserID, RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint, CorrelationID: record.CorrelationID, CausationID: record.CausationID})
	}
}

func postgresGLAccountAuditWriter(auditWriter identity.PostgresAuditWriter) gl.PostgresAccountAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record gl.AccountAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, gl.ErrAccountAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{UserID: record.AccountID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference, Action: record.Action, ScopeIDs: []string{record.AccountingScopeID.String()}, Permission: record.Permission, PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference, ApprovalRequestID: record.ApprovalRequestID, ApprovalDecisionID: record.ApprovalDecisionID, ApproverUserID: record.ApproverUserID, RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint, CorrelationID: record.CorrelationID, CausationID: record.CausationID})
	}
}
