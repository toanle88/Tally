package main

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/invoicing"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
)

const (
	invoiceTemplateOperationID = "invoicing.configure-invoice-templates.v1"
	billingScheduleOperationID = "invoicing.configure-billing-schedules.v1"
)

func newMemoryBillingScheduleService(authorizer identity.AuthorizationEvaluator) (*invoicing.BillingScheduleService, error) {
	repository := invoicing.NewMemoryBillingScheduleRepository()
	return invoicing.NewBillingScheduleService(
		repository,
		evaluatorBillingScheduleAuthorizer{evaluator: authorizer},
		invoicing.AllowAllBillingScheduleReferenceValidator{},
		&invoicing.MemoryBillingScheduleAuditRecorder{},
		time.Now,
	)
}

func newPostgresBillingScheduleService(pool *pgxpool.Pool, auditWriter identity.PostgresAuditWriter, authorizer identity.AuthorizationEvaluator) (*invoicing.BillingScheduleService, error) {
	repository, err := invoicing.NewPostgresBillingScheduleRepository(pool, postgresBillingScheduleAuditWriter(auditWriter))
	if err != nil {
		return nil, err
	}
	return invoicing.NewBillingScheduleServiceWithDurableIdempotency(
		repository,
		evaluatorBillingScheduleAuthorizer{evaluator: authorizer},
		invoicing.UnavailableBillingScheduleReferenceValidator{},
		&invoicing.MemoryBillingScheduleAuditRecorder{},
		time.Now,
		invoicing.DurableBillingScheduleServiceConfig{
			Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(),
			Policy:      platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute},
			OperationID: billingScheduleOperationID,
		},
	)
}

func newMemoryInvoiceTemplateService(authorizer identity.AuthorizationEvaluator) (*invoicing.InvoiceTemplateService, error) {
	repository := invoicing.NewMemoryInvoiceTemplateRepository()
	return invoicing.NewInvoiceTemplateService(
		repository,
		evaluatorInvoiceTemplateAuthorizer{evaluator: authorizer},
		invoicing.AllowAllInvoiceTemplateReferenceValidator{},
		&invoicing.MemoryInvoiceTemplateAuditRecorder{},
		time.Now,
	)
}

func newPostgresInvoiceTemplateService(pool *pgxpool.Pool, auditWriter identity.PostgresAuditWriter, authorizer identity.AuthorizationEvaluator) (*invoicing.InvoiceTemplateService, error) {
	repository, err := invoicing.NewPostgresInvoiceTemplateRepository(pool, postgresInvoiceTemplateAuditWriter(auditWriter))
	if err != nil {
		return nil, err
	}
	return invoicing.NewInvoiceTemplateServiceWithDurableIdempotency(
		repository,
		evaluatorInvoiceTemplateAuthorizer{evaluator: authorizer},
		invoicing.UnavailableInvoiceTemplateReferenceValidator{},
		&invoicing.MemoryInvoiceTemplateAuditRecorder{},
		time.Now,
		invoicing.DurableInvoiceTemplateServiceConfig{
			Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(),
			Policy:      platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute},
			OperationID: invoiceTemplateOperationID,
		},
	)
}

type evaluatorInvoiceTemplateAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

type evaluatorBillingScheduleAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

func (authorizer evaluatorBillingScheduleAuthorizer) AuthorizeBillingSchedule(ctx context.Context, actor invoicing.Actor, command invoicing.BillingScheduleCommand, _ *invoicing.BillingSchedule) (invoicing.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return invoicing.AuthorizationDecision{}, invoicing.ErrBillingScheduleAuthorizationUnavailable
	}
	var scopes []string
	if command.AccountingScopeID != uuid.Nil {
		scopes = []string{command.AccountingScopeID.String()}
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: invoicing.BillingScheduleManagementPermission, RequestedScopeIDs: scopes})
	if err != nil {
		return invoicing.AuthorizationDecision{}, invoicing.ErrBillingScheduleAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return invoicing.AuthorizationDecision{}, invoicing.ErrBillingScheduleAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return invoicing.AuthorizationDecision{}, invoicing.ErrBillingScheduleAuthorizationStale
	}
	result := invoicing.AuthorizationDecision{
		Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission,
		PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion,
		DecisionReference: decision.DecisionReference,
	}
	for _, value := range decision.ApprovedScopeIDs {
		if value == "*" {
			result.ApprovedScopeIDs = append(result.ApprovedScopeIDs, uuid.Nil)
			continue
		}
		parsed, parseErr := uuid.Parse(value)
		if parseErr == nil {
			result.ApprovedScopeIDs = append(result.ApprovedScopeIDs, parsed)
		}
	}
	return result, nil
}

func (authorizer evaluatorInvoiceTemplateAuthorizer) AuthorizeInvoiceTemplate(ctx context.Context, actor invoicing.Actor, command invoicing.InvoiceTemplateCommand, _ *invoicing.InvoiceTemplate) (invoicing.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return invoicing.AuthorizationDecision{}, invoicing.ErrInvoiceTemplateAuthorizationUnavailable
	}
	var scopes []string
	if command.AccountingScopeID != uuid.Nil {
		scopes = []string{command.AccountingScopeID.String()}
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: invoicing.InvoiceTemplateManagementPermission, RequestedScopeIDs: scopes})
	if err != nil {
		return invoicing.AuthorizationDecision{}, invoicing.ErrInvoiceTemplateAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return invoicing.AuthorizationDecision{}, invoicing.ErrInvoiceTemplateAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return invoicing.AuthorizationDecision{}, invoicing.ErrInvoiceTemplateAuthorizationStale
	}
	result := invoicing.AuthorizationDecision{
		Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission,
		PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion,
		DecisionReference: decision.DecisionReference,
	}
	for _, value := range decision.ApprovedScopeIDs {
		if value == "*" {
			result.ApprovedScopeIDs = append(result.ApprovedScopeIDs, uuid.Nil)
			continue
		}
		parsed, parseErr := uuid.Parse(value)
		if parseErr == nil {
			result.ApprovedScopeIDs = append(result.ApprovedScopeIDs, parsed)
		}
	}
	return result, nil
}

func postgresInvoiceTemplateAuditWriter(auditWriter identity.PostgresAuditWriter) invoicing.PostgresInvoiceTemplateAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record invoicing.InvoiceTemplateAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, invoicing.ErrInvoiceTemplateAuditUnavailable
		}
		var expectedVersion *int64
		if record.ExpectedVersion != nil {
			value := record.ExpectedVersion.Value()
			expectedVersion = &value
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.TemplateID, ActorUserID: record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorSubjectReference, Action: record.Action,
			ScopeIDs: []string{record.AccountingScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion,
			DecisionReference: record.DecisionReference, ExpectedVersion: expectedVersion, RevisionVersion: record.RevisionNumber,
			BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}

func postgresBillingScheduleAuditWriter(auditWriter identity.PostgresAuditWriter) invoicing.PostgresBillingScheduleAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record invoicing.BillingScheduleAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, invoicing.ErrBillingScheduleAuditUnavailable
		}
		var expectedVersion *int64
		if record.ExpectedVersion != nil {
			value := record.ExpectedVersion.Value()
			expectedVersion = &value
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.BillingScheduleID, ActorUserID: record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorSubjectReference, Action: record.Action,
			ScopeIDs: []string{record.AccountingScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion,
			DecisionReference: record.DecisionReference, ExpectedVersion: expectedVersion, RevisionVersion: record.RevisionNumber,
			BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}
