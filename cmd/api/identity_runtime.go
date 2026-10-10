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
	"github.com/toanle88/Tally/internal/coa"
	"github.com/toanle88/Tally/internal/gl"
	"github.com/toanle88/Tally/internal/identity"
	"github.com/toanle88/Tally/internal/organization"
	"github.com/toanle88/Tally/internal/platform/httpapi"
	"github.com/toanle88/Tally/internal/platform/httpapi/generated"
	platformidempotency "github.com/toanle88/Tally/internal/platform/idempotency"
	"github.com/toanle88/Tally/internal/platform/telemetry"
)

const (
	identityOperationID                 = "identity.manage-users.v1"
	roleOperationID                     = "identity.manage-roles.v1"
	segregationOperationID              = "identity.manage-segregation-rules.v1"
	emergencyAccessOperationID          = "identity.emergency-access.v1"
	organizationOperationID             = "organization.maintain-legal-entities.v1"
	partyOperationID                    = "organization.maintain-parties.v1"
	customerProfileOperationID          = "organization.maintain-customer-profiles.v1"
	vendorProfileOperationID            = "organization.maintain-vendor-profiles.v1"
	fiscalCalendarOperationID           = "organization.maintain-fiscal-calendars.v1"
	publicationOperationID              = "organization.publish-approved-master-data-changes.v1"
	coaSegmentDefinitionOperationID     = "coa.maintain-segment-definitions.v1"
	coaSegmentValueOperationID          = "coa.maintain-segment-values.v1"
	coaSegmentChangeRequestOperationID  = "coa.request-segment-changes.v1"
	coaSegmentChangeApprovalOperationID = "coa.apply-segment-change-approval-decision.v1"
	coaSegmentValidationOperationID     = "coa.validate-segment-combinations.v1"
	glLedgerOperationID                 = "gl.maintain-ledgers.v1"
	glAccountingBookOperationID         = "gl.maintain-accounting-books.v1"
	glChartOfAccountsOperationID        = "gl.maintain-charts-of-accounts.v1"
	glAccountOperationID                = "gl.maintain-accounts-and-reporting-mappings.v1"
)

func newIdentityAPIServerWithPostgres(getenv func(string) string, pool *pgxpool.Pool, auditWriter identity.PostgresAuditWriter, instrumentation ...*telemetry.Instrumentation) (http.Handler, *organization.LegalEntityService) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if pool == nil || auditWriter == nil {
		return identityUnavailableHandler("identity persistence or audit integration unavailable"), nil
	}
	repository, err := identity.NewPostgresUserRepositoryWithAudit(pool, auditWriter)
	if err != nil {
		return identityUnavailableHandler("identity persistence unavailable"), nil
	}
	roleRepository, err := identity.NewPostgresRoleRepositoryWithAudit(pool, postgresRoleAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("identity role persistence unavailable"), nil
	}
	return newIdentityAPIServerWithPostgresRepository(getenv, pool, repository, roleRepository, auditWriter, instrumentation...)
}

func newIdentityAPIServerWithPostgresRepository(getenv func(string) string, pool *pgxpool.Pool, repository *identity.PostgresUserRepository, roleRepository *identity.PostgresRoleRepository, auditWriter identity.PostgresAuditWriter, instrumentation ...*telemetry.Instrumentation) (http.Handler, *organization.LegalEntityService) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	if pool == nil || repository == nil || roleRepository == nil || auditWriter == nil {
		return identityUnavailableHandler("identity persistence or role integration unavailable"), nil
	}
	policyStore, err := identity.NewPostgresAccessPolicyStore(pool)
	if err != nil {
		return identityUnavailableHandler("identity policy persistence unavailable"), nil
	}
	policyEvaluator, err := identity.NewPolicyEvaluator(policyStore, time.Now, authorizationDecisionObserver(instrumentation...))
	if err != nil {
		return identityUnavailableHandler("identity policy evaluator unavailable"), nil
	}
	authorizer := evaluatorIdentityAuthorizer{evaluator: policyEvaluator}
	segregationRepository, err := identity.NewPostgresSegregationRuleRepositoryWithAudit(pool, postgresSegregationRuleAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule persistence unavailable"), nil
	}
	emergencyRepository, err := identity.NewPostgresEmergencyAccessRepositoryWithAudit(pool, postgresEmergencyAccessAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("identity emergency-access persistence unavailable"), nil
	}
	segregationEvaluator, err := identity.NewSegregationEvaluator(segregationRepository, time.Now, segregationDecisionObserver(instrumentation...))
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule evaluator unavailable"), nil
	}
	segregationAudit := &identity.MemorySegregationRuleAuditRecorder{}
	segregationService, err := identity.NewSegregationRuleServiceWithDurableIdempotency(segregationRepository, authorizer, identity.AllowAllSegregationRuleApprovalPort{}, segregationAudit, time.Now, identity.DurableSegregationRuleServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: segregationOperationID})
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule service unavailable"), nil
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
		return identityUnavailableHandler("identity user service unavailable"), nil
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
		return identityUnavailableHandler("identity role service unavailable"), nil
	}
	emergencyAccessService, err := identity.NewEmergencyAccessServiceWithDurableIdempotency(
		emergencyRepository,
		authorizer,
		environmentEmergencyAccessApproval{getenv: getenv},
		&identity.MemoryEmergencyAccessAuditRecorder{},
		identity.WeekdayEmergencyAccessCalendar{},
		time.Now,
		identity.DurableEmergencyAccessServiceConfig{
			Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(),
			Policy:      platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute},
			OperationID: emergencyAccessOperationID,
		},
	)
	if err != nil {
		return identityUnavailableHandler("identity emergency-access service unavailable"), nil
	}
	organizationRepository, err := organization.NewPostgresLegalEntityRepository(pool, postgresOrganizationAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("organization persistence unavailable"), nil
	}
	organizationService, err := organization.NewLegalEntityServiceWithDurableIdempotency(
		organizationRepository,
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		organization.AllowAllApprovalValidator{},
		&organization.MemoryAuditRecorder{},
		time.Now,
		organization.DurableLegalEntityServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: organizationOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("organization service unavailable"), nil
	}
	partyRepository, err := organization.NewPostgresPartyRepository(pool, postgresOrganizationPartyAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("organization party persistence unavailable"), nil
	}
	partyService, err := organization.NewPartyServiceWithDurableIdempotency(
		partyRepository,
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		unavailablePartyBankReferenceValidator{},
		unavailablePartyBankControlEvaluator{},
		&organization.MemoryPartyAuditRecorder{},
		time.Now,
		organization.DurablePartyServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: partyOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("organization party service unavailable"), nil
	}
	customerProfileRepository, err := organization.NewPostgresCustomerProfileRepository(pool, postgresOrganizationCustomerProfileAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("organization customer-profile persistence unavailable"), nil
	}
	customerProfileService, err := organization.NewCustomerProfileServiceWithDurableIdempotency(
		customerProfileRepository,
		partyRepository,
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		organization.AllowAllCustomerProfileApprovalValidator{},
		&organization.MemoryCustomerProfileAuditRecorder{},
		time.Now,
		organization.DurableCustomerProfileServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: customerProfileOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("organization customer-profile service unavailable"), nil
	}
	vendorProfileRepository, err := organization.NewPostgresVendorProfileRepository(pool, postgresOrganizationVendorProfileAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("organization vendor-profile persistence unavailable"), nil
	}
	vendorProfileService, err := organization.NewVendorProfileServiceWithDurableIdempotency(
		vendorProfileRepository,
		partyRepository,
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		organization.AllowAllVendorProfileApprovalValidator{},
		&organization.MemoryVendorProfileAuditRecorder{},
		time.Now,
		organization.DurableVendorProfileServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: vendorProfileOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("organization vendor-profile service unavailable"), nil
	}
	fiscalCalendarRepository, err := organization.NewPostgresFiscalCalendarRepository(pool, postgresOrganizationFiscalCalendarAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("organization fiscal-calendar persistence unavailable"), nil
	}
	fiscalCalendarService, err := organization.NewFiscalCalendarServiceWithDurableIdempotency(
		fiscalCalendarRepository,
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		organization.AllowAllFiscalCalendarApprovalValidator{},
		&organization.MemoryFiscalCalendarAuditRecorder{},
		organization.UnavailableFiscalCalendarImpactReader{},
		time.Now,
		organization.DurableFiscalCalendarServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: fiscalCalendarOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("organization fiscal-calendar service unavailable"), nil
	}
	publicationRepository, err := organization.NewPostgresMasterDataPublicationRepository(pool, postgresOrganizationMasterDataPublicationAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("organization master-data publication persistence unavailable"), nil
	}
	publicationService, err := organization.NewMasterDataPublicationServiceWithDurableIdempotency(
		publicationRepository,
		evaluatorOrganizationAuthorizer{evaluator: policyEvaluator},
		&organization.MemoryMasterDataPublicationAuditRecorder{},
		time.Now,
		organization.DurableMasterDataPublicationServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: publicationOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("organization master-data publication service unavailable"), nil
	}
	coaRepository, err := coa.NewPostgresSegmentDefinitionRepository(pool, postgresCoaAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("coa segment-definition persistence unavailable"), nil
	}
	coaService, err := coa.NewSegmentDefinitionServiceWithDurableIdempotency(
		coaRepository,
		evaluatorCoaAuthorizer{evaluator: policyEvaluator},
		&coa.MemoryAuditRecorder{},
		time.Now,
		coa.DurableSegmentDefinitionServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: coaSegmentDefinitionOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-definition service unavailable"), nil
	}
	coaValueService, err := coa.NewSegmentValueServiceWithDurableIdempotency(
		coaRepository,
		evaluatorCoaAuthorizer{evaluator: policyEvaluator},
		&coa.MemoryAuditRecorder{},
		time.Now,
		coa.DurableSegmentValueServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: coaSegmentValueOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-value service unavailable"), nil
	}
	coaChangeRequestRepository, err := coa.NewPostgresSegmentChangeRequestRepository(pool, postgresCoaAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("coa segment-change request persistence unavailable"), nil
	}
	coaChangeRequestService, err := coa.NewSegmentChangeRequestServiceWithDurableIdempotency(
		coaChangeRequestRepository,
		coaRepository,
		evaluatorCoaAuthorizer{evaluator: policyEvaluator},
		&coa.MemoryAuditRecorder{},
		time.Now,
		coa.DurableSegmentChangeRequestServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: coaSegmentChangeRequestOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-change request service unavailable"), nil
	}
	coaChangeApprovalService, err := coa.NewSegmentChangeApprovalDecisionServiceWithDurableIdempotency(
		coaChangeRequestRepository,
		coaChangeRequestRepository,
		evaluatorCoaAuthorizer{evaluator: policyEvaluator},
		time.Now,
		coa.DurableSegmentChangeApprovalServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: coaSegmentChangeApprovalOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-change approval application service unavailable"), nil
	}
	coaValidationService, err := coa.NewSegmentCombinationValidationServiceWithDurableIdempotency(
		coaRepository,
		evaluatorCoaAuthorizer{evaluator: policyEvaluator},
		coa.DurableSegmentCombinationValidationServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: coaSegmentValidationOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-combination validation service unavailable"), nil
	}
	glRepository, err := gl.NewPostgresConfigurationRepository(pool, postgresGLLedgerAuditWriter(auditWriter), postgresGLAccountingBookAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("general-ledger configuration persistence unavailable"), nil
	}
	glLedgerService, err := gl.NewLedgerServiceWithDurableIdempotency(
		glRepository,
		evaluatorGLLedgerAuthorizer{evaluator: policyEvaluator},
		omdGLReferenceValidator{legalEntities: organizationService, fiscalCalendars: fiscalCalendarService},
		gl.UnavailableApprovalValidator{},
		&gl.MemoryLedgerAuditRecorder{},
		time.Now,
		gl.DurableLedgerServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: glLedgerOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("general-ledger service unavailable"), nil
	}
	glAccountingBookService, err := gl.NewAccountingBookServiceWithDurableIdempotency(
		glRepository,
		evaluatorGLAccountingBookAuthorizer{evaluator: policyEvaluator},
		omdGLReferenceValidator{legalEntities: organizationService, fiscalCalendars: fiscalCalendarService},
		gl.UnavailableApprovalValidator{},
		&gl.MemoryAccountingBookAuditRecorder{},
		time.Now,
		gl.DurableAccountingBookServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: glAccountingBookOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("accounting-book service unavailable"), nil
	}
	glChartAccountRepository, err := gl.NewPostgresChartAccountRepository(pool, postgresGLChartOfAccountsAuditWriter(auditWriter), postgresGLAccountAuditWriter(auditWriter))
	if err != nil {
		return identityUnavailableHandler("general-ledger chart/account persistence unavailable"), nil
	}
	glChartOfAccountsService, err := gl.NewChartOfAccountsServiceWithDurableIdempotency(
		glChartAccountRepository,
		evaluatorGLChartOfAccountsAuthorizer{evaluator: policyEvaluator},
		runtimeGLChartAccountReferenceValidator{},
		gl.UnavailableChartAccountApprovalValidator{},
		&gl.MemoryChartOfAccountsAuditRecorder{},
		time.Now,
		gl.DurableChartOfAccountsServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: glChartOfAccountsOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("chart-of-accounts service unavailable"), nil
	}
	glAccountService, err := gl.NewAccountServiceWithDurableIdempotency(
		glChartAccountRepository,
		evaluatorGLAccountAuthorizer{evaluator: policyEvaluator},
		runtimeGLChartAccountReferenceValidator{},
		gl.UnavailableChartAccountApprovalValidator{},
		&gl.MemoryAccountAuditRecorder{},
		time.Now,
		gl.DurableAccountServiceConfig{Database: pool, Coordinator: platformidempotency.NewPostgresCoordinator(), Policy: platformidempotency.IdempotencyPolicy{RecordTTL: 24 * time.Hour, LeaseTTL: 5 * time.Minute}, OperationID: glAccountOperationID},
	)
	if err != nil {
		return identityUnavailableHandler("account service unavailable"), nil
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{SegmentDefinitionService: coaService, SegmentValueService: coaValueService, SegmentChangeRequestService: coaChangeRequestService, SegmentChangeApprovalDecisionService: coaChangeApprovalService, SegmentCombinationValidationService: coaValidationService, LedgerService: glLedgerService, AccountingBookService: glAccountingBookService, ChartOfAccountsService: glChartOfAccountsService, AccountService: glAccountService, Service: userService, RoleService: roleService, SegregationRuleService: segregationService, EmergencyAccessService: emergencyAccessService, OrganizationService: organizationService, PartyService: partyService, CustomerProfileService: customerProfileService, VendorProfileService: vendorProfileService, FiscalCalendarService: fiscalCalendarService, PublicationService: publicationService, Instrumentation: optionalInstrumentation(instrumentation...)},
		apiBearerSecurityHandler{},
	)
	if err != nil {
		return identityUnavailableHandler("identity API unavailable"), nil
	}
	return server, organizationService
}

func postgresCoaAuditWriter(auditWriter identity.PostgresAuditWriter) coa.PostgresSegmentDefinitionAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record coa.AuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			if record.SegmentChangeRequestID != uuid.Nil {
				return uuid.Nil, coa.ErrSegmentChangeRequestAuditUnavailable
			}
			if record.SegmentValueID != uuid.Nil {
				return uuid.Nil, coa.ErrSegmentValueAuditUnavailable
			}
			return uuid.Nil, coa.ErrSegmentDefinitionAuditUnavailable
		}
		userID := record.SegmentChangeRequestID
		if userID == uuid.Nil {
			userID = record.SegmentDefinitionID
			if record.SegmentValueID != uuid.Nil {
				userID = record.SegmentValueID
			}
		}
		if userID == uuid.Nil {
			return uuid.Nil, coa.ErrSegmentChangeRequestAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        userID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action:                        record.Action,
			ScopeIDs:                      []string{record.ScopeID.String()},
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyReference,
			PolicyVersion:                 record.PolicyVersion,
			DecisionReference:             record.DecisionReference,
			ApprovalRequestID:             record.ApprovalRequestID,
			ApprovalDecisionID:            record.ApprovalDecisionID,
			ApproverUserID:                record.ApproverUserID,
			RevisionVersion:               record.RevisionNumber,
			BeforeFingerprint:             record.BeforeFingerprint,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
}

func postgresGLLedgerAuditWriter(auditWriter identity.PostgresAuditWriter) gl.PostgresLedgerAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record gl.LedgerAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, gl.ErrLedgerAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        record.LedgerID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action:                        record.Action,
			ScopeIDs:                      []string{record.AccountingScopeID.String()},
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyReference,
			PolicyVersion:                 record.PolicyVersion,
			DecisionReference:             record.DecisionReference,
			ApprovalRequestID:             record.ApprovalRequestID,
			ApprovalDecisionID:            record.ApprovalDecisionID,
			ApproverUserID:                record.ApproverUserID,
			RevisionVersion:               record.RevisionNumber,
			BeforeFingerprint:             record.BeforeFingerprint,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
}

func postgresGLAccountingBookAuditWriter(auditWriter identity.PostgresAuditWriter) gl.PostgresAccountingBookAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record gl.AccountingBookAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, gl.ErrAccountingBookAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID:                        record.AccountingBookID,
			ActorUserID:                   record.ActorUserID,
			ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action:                        record.Action,
			ScopeIDs:                      []string{record.AccountingScopeID.String()},
			Permission:                    record.Permission,
			PolicyReference:               record.PolicyReference,
			PolicyVersion:                 record.PolicyVersion,
			DecisionReference:             record.DecisionReference,
			ApprovalRequestID:             record.ApprovalRequestID,
			ApprovalDecisionID:            record.ApprovalDecisionID,
			ApproverUserID:                record.ApproverUserID,
			RevisionVersion:               record.RevisionNumber,
			BeforeFingerprint:             record.BeforeFingerprint,
			AfterFingerprint:              record.AfterFingerprint,
			CorrelationID:                 record.CorrelationID,
			CausationID:                   record.CausationID,
		})
	}
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

func postgresOrganizationAuditWriter(auditWriter identity.PostgresAuditWriter) organization.PostgresAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record organization.AuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, organization.ErrLegalEntityAuditUnavailable
		}
		var approvalRequestID, approvalDecisionID, approverUserID uuid.UUID
		if record.Approval != nil {
			approvalRequestID, approvalDecisionID, approverUserID = record.Approval.ApprovalRequestID, record.Approval.DecisionID, record.Approval.ApproverUserID
		}
		return auditWriter(ctx, tx, identity.AuditRecord{UserID: record.LegalEntityID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference, Action: record.Action, ScopeIDs: []string{record.ScopeID.String()}, Permission: record.Permission, PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference, ApprovalRequestID: approvalRequestID, ApprovalDecisionID: approvalDecisionID, ApproverUserID: approverUserID, RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint, CorrelationID: record.CorrelationID, CausationID: record.CausationID})
	}
}

func postgresOrganizationPartyAuditWriter(auditWriter identity.PostgresAuditWriter) organization.PostgresPartyAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record organization.PartyAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, organization.ErrPartyAuditUnavailable
		}
		var approvalRequestID, approvalDecisionID, approverUserID uuid.UUID
		if record.Approval != nil {
			approvalRequestID, approvalDecisionID, approverUserID = record.Approval.ApprovalRequestID, record.Approval.DecisionID, record.Approval.ApproverUserID
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.PartyID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action: record.Action, ScopeIDs: []string{record.ScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference,
			ApprovalRequestID: approvalRequestID, ApprovalDecisionID: approvalDecisionID, ApproverUserID: approverUserID,
			RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}

func postgresOrganizationCustomerProfileAuditWriter(auditWriter identity.PostgresAuditWriter) organization.PostgresCustomerProfileAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record organization.CustomerProfileAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, organization.ErrCustomerProfileAuditUnavailable
		}
		var approvalRequestID, approvalDecisionID, approverUserID uuid.UUID
		if record.Approval != nil {
			approvalRequestID, approvalDecisionID, approverUserID = record.Approval.ApprovalRequestID, record.Approval.DecisionID, record.Approval.ApproverUserID
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.CustomerProfileID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action: record.Action, ScopeIDs: []string{record.ScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference,
			ApprovalRequestID: approvalRequestID, ApprovalDecisionID: approvalDecisionID, ApproverUserID: approverUserID,
			RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}

func postgresOrganizationVendorProfileAuditWriter(auditWriter identity.PostgresAuditWriter) organization.PostgresVendorProfileAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record organization.VendorProfileAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, organization.ErrVendorProfileAuditUnavailable
		}
		var approvalRequestID, approvalDecisionID, approverUserID uuid.UUID
		if record.Approval != nil {
			approvalRequestID, approvalDecisionID, approverUserID = record.Approval.ApprovalRequestID, record.Approval.DecisionID, record.Approval.ApproverUserID
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.VendorProfileID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action: record.Action, ScopeIDs: []string{record.ScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference,
			ApprovalRequestID: approvalRequestID, ApprovalDecisionID: approvalDecisionID, ApproverUserID: approverUserID,
			RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}

func postgresOrganizationFiscalCalendarAuditWriter(auditWriter identity.PostgresAuditWriter) organization.PostgresFiscalCalendarAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record organization.FiscalCalendarAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, organization.ErrFiscalCalendarAuditUnavailable
		}
		var approvalRequestID, approvalDecisionID, approverUserID uuid.UUID
		if record.Approval != nil {
			approvalRequestID, approvalDecisionID, approverUserID = record.Approval.ApprovalRequestID, record.Approval.DecisionID, record.Approval.ApproverUserID
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.FiscalCalendarID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action: record.Action, ScopeIDs: []string{record.ScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference,
			ApprovalRequestID: approvalRequestID, ApprovalDecisionID: approvalDecisionID, ApproverUserID: approverUserID,
			RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}

func postgresOrganizationMasterDataPublicationAuditWriter(auditWriter identity.PostgresAuditWriter) organization.PostgresMasterDataPublicationAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record organization.MasterDataPublicationAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, organization.ErrMasterDataPublicationAuditUnavailable
		}
		var approvalRequestID, approvalDecisionID, approverUserID uuid.UUID
		if record.Approval != nil {
			approvalRequestID, approvalDecisionID, approverUserID = record.Approval.ApprovalRequestID, record.Approval.DecisionID, record.Approval.ApproverUserID
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.PublicationID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorSubjectReference,
			Action: record.Action, ScopeIDs: []string{record.ScopeID.String()}, Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference,
			ApprovalRequestID: approvalRequestID, ApprovalDecisionID: approvalDecisionID, ApproverUserID: approverUserID,
			RevisionVersion: record.RevisionNumber, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
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

func postgresEmergencyAccessAuditWriter(auditWriter identity.PostgresAuditWriter) identity.PostgresEmergencyAccessAuditWriter {
	return func(ctx context.Context, tx pgx.Tx, record identity.EmergencyAccessAuditRecord) (uuid.UUID, error) {
		if auditWriter == nil {
			return uuid.Nil, identity.ErrEmergencyAccessAuditUnavailable
		}
		return auditWriter(ctx, tx, identity.AuditRecord{
			UserID: record.GrantID, ActorUserID: record.ActorUserID, ActorAuthenticationSubjectRef: record.ActorAuthenticationRef,
			Action: record.Action, ScopeIDs: append([]string(nil), record.ScopeIDs...), Permission: record.Permission,
			PolicyReference: record.PolicyReference, PolicyVersion: record.PolicyVersion, DecisionReference: record.DecisionReference,
			ApprovalRequestID: record.ApprovalRequestID, ApprovalDecisionID: record.ApprovalDecisionID, ApproverUserID: record.ApproverUserID,
			RevisionVersion: record.GrantVersion, BeforeFingerprint: record.BeforeFingerprint, AfterFingerprint: record.AfterFingerprint,
			CorrelationID: record.CorrelationID, CausationID: record.CausationID,
		})
	}
}

func identityUnavailableHandler(message string) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		http.Error(writer, message, http.StatusServiceUnavailable)
	})
}

func newIdentityAPIServer(getenv func(string) string) (http.Handler, *organization.LegalEntityService) {
	return newIdentityAPIServerWithRepository(getenv, identity.NewMemoryUserRepository())
}

func newIdentityAPIServerWithRepository(getenv func(string) string, repository identity.UserRepository, instrumentation ...*telemetry.Instrumentation) (http.Handler, *organization.LegalEntityService) {
	if getenv == nil {
		getenv = func(string) string { return "" }
	}
	roleRepository := identity.NewMemoryRoleRepository()
	authorizer, err := newEnvironmentIdentityAuthorizer(getenv, instrumentation...)
	if err != nil {
		return identityUnavailableHandler("identity policy evaluator unavailable"), nil
	}
	segregationRepository, err := identity.NewMemorySegregationRuleRepository(identity.DefaultSegregationRules(time.Now())...)
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule persistence unavailable"), nil
	}
	segregationEvaluator, err := identity.NewSegregationEvaluator(segregationRepository, time.Now, segregationDecisionObserver(instrumentation...))
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule evaluator unavailable"), nil
	}
	emergencyRepository := identity.NewMemoryEmergencyAccessRepository()
	segregationAudit := &identity.MemorySegregationRuleAuditRecorder{}
	segregationService, err := identity.NewSegregationRuleService(segregationRepository, authorizer, identity.AllowAllSegregationRuleApprovalPort{}, segregationAudit, time.Now)
	if err != nil {
		return identityUnavailableHandler("identity segregation-rule service unavailable"), nil
	}
	userService, err := identity.NewUserService(
		repository,
		authorizer,
		&identity.MemoryAuditRecorder{},
		time.Now,
		roleRepository,
	)
	if err != nil {
		return identityUnavailableHandler("identity service unavailable"), nil
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
		return identityUnavailableHandler("identity role service unavailable"), nil
	}
	emergencyAccessService, err := identity.NewEmergencyAccessService(
		emergencyRepository,
		authorizer,
		environmentEmergencyAccessApproval{getenv: getenv},
		&identity.MemoryEmergencyAccessAuditRecorder{},
		identity.WeekdayEmergencyAccessCalendar{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("identity emergency-access service unavailable"), nil
	}
	organizationRepository := organization.NewMemoryLegalEntityRepository()
	organizationService, err := organization.NewLegalEntityService(
		organizationRepository,
		permissiveOrganizationAuthorizer{},
		organization.AllowAllApprovalValidator{},
		&organization.MemoryAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("organization service unavailable"), nil
	}
	partyRepository := organization.NewMemoryPartyRepository()
	partyService, err := organization.NewPartyService(
		partyRepository,
		permissiveOrganizationAuthorizer{},
		permissiveOrganizationAuthorizer{},
		unavailablePartyBankReferenceValidator{},
		unavailablePartyBankControlEvaluator{},
		&organization.MemoryPartyAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("organization party service unavailable"), nil
	}
	customerProfileRepository := organization.NewMemoryCustomerProfileRepository(partyRepository)
	customerProfileService, err := organization.NewCustomerProfileService(
		customerProfileRepository,
		partyRepository,
		permissiveOrganizationAuthorizer{},
		permissiveOrganizationAuthorizer{},
		organization.AllowAllCustomerProfileApprovalValidator{},
		&organization.MemoryCustomerProfileAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("organization customer-profile service unavailable"), nil
	}
	vendorProfileRepository := organization.NewMemoryVendorProfileRepository(partyRepository)
	vendorProfileService, err := organization.NewVendorProfileService(
		vendorProfileRepository,
		partyRepository,
		permissiveOrganizationAuthorizer{},
		permissiveOrganizationAuthorizer{},
		organization.AllowAllVendorProfileApprovalValidator{},
		&organization.MemoryVendorProfileAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("organization vendor-profile service unavailable"), nil
	}
	fiscalCalendarRepository := organization.NewMemoryFiscalCalendarRepository()
	fiscalCalendarService, err := organization.NewFiscalCalendarService(
		fiscalCalendarRepository,
		permissiveOrganizationAuthorizer{},
		organization.AllowAllFiscalCalendarApprovalValidator{},
		&organization.MemoryFiscalCalendarAuditRecorder{},
		organization.UnavailableFiscalCalendarImpactReader{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("organization fiscal-calendar service unavailable"), nil
	}
	publicationRepository := organization.NewMemoryMasterDataPublicationRepository()
	publicationService, err := organization.NewMasterDataPublicationService(
		publicationRepository,
		permissiveOrganizationAuthorizer{},
		&organization.MemoryMasterDataPublicationAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("organization master-data publication service unavailable"), nil
	}
	coaRepository := coa.NewMemorySegmentDefinitionRepository()
	coaService, err := coa.NewSegmentDefinitionService(
		coaRepository,
		permissiveCoaAuthorizer{},
		&coa.MemoryAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-definition service unavailable"), nil
	}
	coaValueService, err := coa.NewSegmentValueService(
		coaRepository,
		permissiveCoaAuthorizer{},
		&coa.MemoryAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-value service unavailable"), nil
	}
	coaChangeRequestRepository := coa.NewMemorySegmentChangeRequestRepository(coaRepository)
	coaChangeRequestService, err := coa.NewSegmentChangeRequestService(
		coaChangeRequestRepository,
		coaRepository,
		permissiveCoaAuthorizer{},
		&coa.MemoryAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-change request service unavailable"), nil
	}
	coaChangeApprovalService, err := coa.NewSegmentChangeApprovalDecisionService(
		coaChangeRequestRepository,
		coaChangeRequestRepository,
		permissiveCoaAuthorizer{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-change approval application service unavailable"), nil
	}
	coaValidationService, err := coa.NewSegmentCombinationValidationService(
		coaRepository,
		permissiveCoaAuthorizer{},
	)
	if err != nil {
		return identityUnavailableHandler("coa segment-combination validation service unavailable"), nil
	}
	glLedgerRepository := gl.NewMemoryLedgerRepository()
	glLedgerService, err := gl.NewLedgerService(
		glLedgerRepository,
		permissiveGLLedgerAuthorizer{},
		gl.AllowAllReferenceValidator{},
		gl.AllowAllApprovalValidator{},
		&gl.MemoryLedgerAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("general-ledger service unavailable"), nil
	}
	glAccountingBookRepository := gl.NewMemoryAccountingBookRepository(glLedgerRepository)
	glAccountingBookService, err := gl.NewAccountingBookService(
		glAccountingBookRepository,
		permissiveGLAccountingBookAuthorizer{},
		gl.AllowAllReferenceValidator{},
		gl.AllowAllApprovalValidator{},
		&gl.MemoryAccountingBookAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("accounting-book service unavailable"), nil
	}
	glChartAccountRepository := gl.NewMemoryChartOfAccountsRepository(glLedgerRepository)
	glChartOfAccountsService, err := gl.NewChartOfAccountsService(
		glChartAccountRepository,
		permissiveGLChartOfAccountsAuthorizer{},
		gl.AllowAllChartAccountReferenceValidator{},
		gl.AllowAllChartAccountApprovalValidator{},
		&gl.MemoryChartOfAccountsAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("chart-of-accounts service unavailable"), nil
	}
	glAccountRepository := gl.NewMemoryAccountRepository(glChartAccountRepository)
	glAccountService, err := gl.NewAccountService(
		glAccountRepository,
		permissiveGLAccountAuthorizer{},
		gl.AllowAllChartAccountReferenceValidator{},
		gl.AllowAllChartAccountApprovalValidator{},
		&gl.MemoryAccountAuditRecorder{},
		time.Now,
	)
	if err != nil {
		return identityUnavailableHandler("account service unavailable"), nil
	}
	postingService, err := newMemoryPostingService()
	if err != nil {
		return identityUnavailableHandler("posting service unavailable"), nil
	}
	server, err := generated.NewServer(
		httpapi.IdentityHandler{SegmentDefinitionService: coaService, SegmentValueService: coaValueService, SegmentChangeRequestService: coaChangeRequestService, SegmentChangeApprovalDecisionService: coaChangeApprovalService, SegmentCombinationValidationService: coaValidationService, LedgerService: glLedgerService, AccountingBookService: glAccountingBookService, ChartOfAccountsService: glChartOfAccountsService, AccountService: glAccountService, PostingService: postingService, Service: userService, RoleService: roleService, SegregationRuleService: segregationService, EmergencyAccessService: emergencyAccessService, OrganizationService: organizationService, PartyService: partyService, CustomerProfileService: customerProfileService, VendorProfileService: vendorProfileService, FiscalCalendarService: fiscalCalendarService, PublicationService: publicationService, Instrumentation: optionalInstrumentation(instrumentation...)},
		apiBearerSecurityHandler{},
	)
	if err != nil {
		return identityUnavailableHandler("identity API unavailable"), nil
	}
	return server, organizationService
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

type evaluatorOrganizationAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

type evaluatorCoaAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

type evaluatorGLLedgerAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

type evaluatorGLAccountingBookAuthorizer struct {
	evaluator identity.AuthorizationEvaluator
}

func (authorizer evaluatorGLLedgerAuthorizer) AuthorizeLedger(ctx context.Context, actor gl.Actor, command gl.LedgerCommand, _ *gl.Ledger) (gl.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return gl.AuthorizationDecision{}, gl.ErrLedgerAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: gl.LedgerManagementPermission, RequestedScopeIDs: []string{command.AccountingScopeID.String()}})
	if err != nil {
		return gl.AuthorizationDecision{}, gl.ErrLedgerAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return gl.AuthorizationDecision{}, gl.ErrLedgerAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return gl.AuthorizationDecision{}, gl.ErrLedgerAuthorizationStale
	}
	return glAuthorizationDecision(decision), nil
}

func (authorizer evaluatorGLAccountingBookAuthorizer) AuthorizeAccountingBook(ctx context.Context, actor gl.Actor, command gl.AccountingBookCommand, _ *gl.AccountingBook) (gl.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return gl.AuthorizationDecision{}, gl.ErrAccountingBookAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: gl.AccountingBookManagementPermission, RequestedScopeIDs: []string{command.AccountingScopeID.String()}})
	if err != nil {
		return gl.AuthorizationDecision{}, gl.ErrAccountingBookAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return gl.AuthorizationDecision{}, gl.ErrAccountingBookAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return gl.AuthorizationDecision{}, gl.ErrAccountingBookAuthorizationStale
	}
	return glAuthorizationDecision(decision), nil
}

func glAuthorizationDecision(decision identity.AuthorizationDecision) gl.AuthorizationDecision {
	result := gl.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference, Reason: decision.Reason}
	for _, value := range decision.ApprovedScopeIDs {
		if value == "*" {
			result.ApprovedScopeIDs = append(result.ApprovedScopeIDs, uuid.Nil)
			continue
		}
		parsed, err := uuid.Parse(value)
		if err == nil {
			result.ApprovedScopeIDs = append(result.ApprovedScopeIDs, parsed)
		}
	}
	return result
}

func (authorizer evaluatorCoaAuthorizer) AuthorizeSegmentDefinition(ctx context.Context, actor coa.Actor, command coa.SegmentDefinitionCommand, _ *coa.SegmentDefinition) (coa.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentDefinitionAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: coa.SegmentDefinitionManagementPermission, RequestedScopeIDs: []string{command.ScopeID.String()}})
	if err != nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentDefinitionAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return coa.AuthorizationDecision{}, coa.ErrSegmentDefinitionAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return coa.AuthorizationDecision{}, coa.ErrSegmentDefinitionAuthorizationStale
	}
	result := coa.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorCoaAuthorizer) AuthorizeSegmentValue(ctx context.Context, actor coa.Actor, command coa.SegmentValueCommand, _ *coa.SegmentDefinition, _ *coa.SegmentValue) (coa.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentValueAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: coa.SegmentValueManagementPermission, RequestedScopeIDs: []string{command.ScopeID.String()}})
	if err != nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentValueAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return coa.AuthorizationDecision{}, coa.ErrSegmentValueAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return coa.AuthorizationDecision{}, coa.ErrSegmentValueAuthorizationStale
	}
	result := coa.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorCoaAuthorizer) AuthorizeSegmentChangeRequest(ctx context.Context, actor coa.Actor, command coa.SegmentChangeRequestCommand, _ *coa.SegmentChangeRequestSubject) (coa.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeRequestAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: coa.SegmentChangeRequestPermission, RequestedScopeIDs: []string{command.ScopeID.String()}})
	if err != nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeRequestAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeRequestAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeRequestAuthorizationStale
	}
	result := coa.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorCoaAuthorizer) AuthorizeSegmentChangeApprovalDecision(ctx context.Context, actor coa.Actor, command coa.SegmentChangeApprovalDecisionCommand, request *coa.SegmentChangeRequest) (coa.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeApprovalDecisionAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: coa.SegmentChangeApprovalDecisionPermission, RequestedScopeIDs: []string{command.ScopeID.String()}})
	if err != nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeApprovalDecisionAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeApprovalDecisionAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return coa.AuthorizationDecision{}, coa.ErrSegmentChangeApprovalDecisionAuthorizationStale
	}
	result := coa.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
	if request != nil && request.ScopeID != command.ScopeID {
		result.Allowed = false
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

func (authorizer evaluatorCoaAuthorizer) AuthorizeSegmentCombinationValidation(ctx context.Context, actor coa.Actor, command coa.SegmentCombinationValidationCommand) (coa.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentCombinationValidationAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: coa.SegmentCombinationValidationPermission, RequestedScopeIDs: []string{command.ScopeID.String()}})
	if err != nil {
		return coa.AuthorizationDecision{}, coa.ErrSegmentCombinationValidationAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return coa.AuthorizationDecision{}, coa.ErrSegmentCombinationValidationAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return coa.AuthorizationDecision{}, coa.ErrSegmentCombinationValidationAuthorizationStale
	}
	result := coa.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeLegalEntity(ctx context.Context, actor organization.Actor, command organization.LegalEntityCommand, _ *organization.LegalEntity) (organization.AuthorizationDecision, error) {
	return authorizer.evaluate(ctx, actor, organization.LegalEntityManagementPermission, command.ScopeID)
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeLegalEntityRead(ctx context.Context, actor organization.Actor, scopeID uuid.UUID) (organization.AuthorizationDecision, error) {
	return authorizer.evaluate(ctx, actor, organization.LegalEntityReadPermission, scopeID)
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeParty(ctx context.Context, actor organization.Actor, command organization.PartyCommand, _ *organization.Party) (organization.AuthorizationDecision, error) {
	return authorizer.evaluateParty(ctx, actor, organization.PartyManagementPermission, command.ScopeID)
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizePartyFields(ctx context.Context, actor organization.Actor, command organization.PartyCommand, current *organization.Party) (organization.PartyFieldAuthorization, error) {
	decision, err := authorizer.evaluateParty(ctx, actor, organization.PartyManagementPermission, command.ScopeID)
	if err != nil {
		return organization.PartyFieldAuthorization{}, err
	}
	if !decision.Allowed {
		return organization.PartyFieldAuthorization{}, nil
	}
	return organization.PartyFieldAuthorization{Identity: true, RestrictedTaxIdentifier: true, PersonalData: true, Classifications: true, BankDetailReferences: true}, nil
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeCustomerProfile(ctx context.Context, actor organization.Actor, command organization.CustomerProfileCommand, _ *organization.CustomerProfile) (organization.AuthorizationDecision, error) {
	return authorizer.evaluateCustomerProfile(ctx, actor, organization.CustomerProfileManagementPermission, command.ScopeID)
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeCustomerProfileFields(ctx context.Context, actor organization.Actor, command organization.CustomerProfileCommand, current *organization.CustomerProfile) (organization.CustomerProfileFieldAuthorization, error) {
	decision, err := authorizer.evaluateCustomerProfile(ctx, actor, organization.CustomerProfileManagementPermission, command.ScopeID)
	if err != nil {
		return organization.CustomerProfileFieldAuthorization{}, err
	}
	if !decision.Allowed {
		return organization.CustomerProfileFieldAuthorization{}, nil
	}
	return organization.CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true}, nil
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeVendorProfile(ctx context.Context, actor organization.Actor, command organization.VendorProfileCommand, _ *organization.VendorProfile) (organization.AuthorizationDecision, error) {
	return authorizer.evaluateVendorProfile(ctx, actor, organization.VendorProfileManagementPermission, command.ScopeID)
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeVendorProfileFields(ctx context.Context, actor organization.Actor, command organization.VendorProfileCommand, _ *organization.VendorProfile) (organization.VendorProfileFieldAuthorization, error) {
	decision, err := authorizer.evaluateVendorProfile(ctx, actor, organization.VendorProfileManagementPermission, command.ScopeID)
	if err != nil {
		return organization.VendorProfileFieldAuthorization{}, err
	}
	if !decision.Allowed {
		return organization.VendorProfileFieldAuthorization{}, nil
	}
	return organization.VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}, nil
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeFiscalCalendar(ctx context.Context, actor organization.Actor, command organization.FiscalCalendarCommand, _ *organization.FiscalCalendar) (organization.AuthorizationDecision, error) {
	return authorizer.evaluateFiscalCalendar(ctx, actor, organization.FiscalCalendarManagementPermission, command.ScopeID)
}

func (authorizer evaluatorOrganizationAuthorizer) AuthorizeMasterDataPublication(ctx context.Context, actor organization.Actor, _ organization.MasterDataPublicationCommand, candidate organization.MasterDataPublicationCandidate) (organization.AuthorizationDecision, error) {
	decision, err := authorizer.evaluate(ctx, actor, organization.MasterDataPublicationPermission, candidate.ScopeID)
	if err != nil {
		return organization.AuthorizationDecision{}, organization.ErrMasterDataPublicationAuthorizationUnavailable
	}
	return decision, nil
}

func (authorizer evaluatorOrganizationAuthorizer) evaluate(ctx context.Context, actor organization.Actor, permission string, scopeID uuid.UUID) (organization.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return organization.AuthorizationDecision{}, organization.ErrLegalEntityAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: permission, RequestedScopeIDs: []string{scopeID.String()}})
	if err != nil {
		return organization.AuthorizationDecision{}, organization.ErrLegalEntityAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return organization.AuthorizationDecision{}, organization.ErrLegalEntityAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return organization.AuthorizationDecision{}, organization.ErrLegalEntityAuthorizationStale
	}
	result := organization.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorOrganizationAuthorizer) evaluateParty(ctx context.Context, actor organization.Actor, permission string, scopeID uuid.UUID) (organization.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return organization.AuthorizationDecision{}, organization.ErrPartyAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: permission, RequestedScopeIDs: []string{scopeID.String()}})
	if err != nil {
		return organization.AuthorizationDecision{}, organization.ErrPartyAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return organization.AuthorizationDecision{}, organization.ErrPartyAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return organization.AuthorizationDecision{}, organization.ErrPartyAuthorizationStale
	}
	result := organization.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorOrganizationAuthorizer) evaluateCustomerProfile(ctx context.Context, actor organization.Actor, permission string, scopeID uuid.UUID) (organization.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return organization.AuthorizationDecision{}, organization.ErrCustomerProfileAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: permission, RequestedScopeIDs: []string{scopeID.String()}})
	if err != nil {
		return organization.AuthorizationDecision{}, organization.ErrCustomerProfileAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return organization.AuthorizationDecision{}, organization.ErrCustomerProfileAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return organization.AuthorizationDecision{}, organization.ErrCustomerProfileAuthorizationStale
	}
	result := organization.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorOrganizationAuthorizer) evaluateVendorProfile(ctx context.Context, actor organization.Actor, permission string, scopeID uuid.UUID) (organization.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return organization.AuthorizationDecision{}, organization.ErrVendorProfileAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: permission, RequestedScopeIDs: []string{scopeID.String()}})
	if err != nil {
		return organization.AuthorizationDecision{}, organization.ErrVendorProfileAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return organization.AuthorizationDecision{}, organization.ErrVendorProfileAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return organization.AuthorizationDecision{}, organization.ErrVendorProfileAuthorizationStale
	}
	result := organization.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

func (authorizer evaluatorOrganizationAuthorizer) evaluateFiscalCalendar(ctx context.Context, actor organization.Actor, permission string, scopeID uuid.UUID) (organization.AuthorizationDecision, error) {
	if authorizer.evaluator == nil {
		return organization.AuthorizationDecision{}, organization.ErrFiscalCalendarAuthorizationUnavailable
	}
	decision, err := authorizer.evaluator.Evaluate(ctx, identity.DecisionInput{ActorID: actor.UserID, Permission: permission, RequestedScopeIDs: []string{scopeID.String()}})
	if err != nil {
		return organization.AuthorizationDecision{}, organization.ErrFiscalCalendarAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationUnavailable {
		return organization.AuthorizationDecision{}, organization.ErrFiscalCalendarAuthorizationUnavailable
	}
	if decision.Outcome == identity.AuthorizationStale {
		return organization.AuthorizationDecision{}, organization.ErrFiscalCalendarAuthorizationStale
	}
	result := organization.AuthorizationDecision{Allowed: decision.Allowed, Outcome: string(decision.Outcome), Permission: decision.Permission, PolicyReference: decision.PolicyReference, PolicyVersion: decision.PolicyVersion, DecisionReference: decision.DecisionReference}
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

type permissiveCoaAuthorizer struct{}

type permissiveGLLedgerAuthorizer struct{}

func (permissiveGLLedgerAuthorizer) AuthorizeLedger(context.Context, gl.Actor, gl.LedgerCommand, *gl.Ledger) (gl.AuthorizationDecision, error) {
	return gl.AuthorizationDecision{Allowed: true, Permission: gl.LedgerManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

type permissiveGLAccountingBookAuthorizer struct{}

func (permissiveGLAccountingBookAuthorizer) AuthorizeAccountingBook(context.Context, gl.Actor, gl.AccountingBookCommand, *gl.AccountingBook) (gl.AuthorizationDecision, error) {
	return gl.AuthorizationDecision{Allowed: true, Permission: gl.AccountingBookManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveCoaAuthorizer) AuthorizeSegmentDefinition(context.Context, coa.Actor, coa.SegmentDefinitionCommand, *coa.SegmentDefinition) (coa.AuthorizationDecision, error) {
	return coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentDefinitionManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveCoaAuthorizer) AuthorizeSegmentValue(context.Context, coa.Actor, coa.SegmentValueCommand, *coa.SegmentDefinition, *coa.SegmentValue) (coa.AuthorizationDecision, error) {
	return coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentValueManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveCoaAuthorizer) AuthorizeSegmentChangeRequest(context.Context, coa.Actor, coa.SegmentChangeRequestCommand, *coa.SegmentChangeRequestSubject) (coa.AuthorizationDecision, error) {
	return coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeRequestPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveCoaAuthorizer) AuthorizeSegmentChangeApprovalDecision(context.Context, coa.Actor, coa.SegmentChangeApprovalDecisionCommand, *coa.SegmentChangeRequest) (coa.AuthorizationDecision, error) {
	return coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentChangeApprovalDecisionPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveCoaAuthorizer) AuthorizeSegmentCombinationValidation(context.Context, coa.Actor, coa.SegmentCombinationValidationCommand) (coa.AuthorizationDecision, error) {
	return coa.AuthorizationDecision{Allowed: true, Permission: coa.SegmentCombinationValidationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

type permissiveOrganizationAuthorizer struct{}

func (permissiveOrganizationAuthorizer) AuthorizeLegalEntity(context.Context, organization.Actor, organization.LegalEntityCommand, *organization.LegalEntity) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.LegalEntityManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}
func (permissiveOrganizationAuthorizer) AuthorizeLegalEntityRead(context.Context, organization.Actor, uuid.UUID) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.LegalEntityReadPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeParty(context.Context, organization.Actor, organization.PartyCommand, *organization.Party) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.PartyManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizePartyFields(context.Context, organization.Actor, organization.PartyCommand, *organization.Party) (organization.PartyFieldAuthorization, error) {
	return organization.PartyFieldAuthorization{Identity: true, RestrictedTaxIdentifier: true, PersonalData: true, Classifications: true, BankDetailReferences: true}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeCustomerProfile(context.Context, organization.Actor, organization.CustomerProfileCommand, *organization.CustomerProfile) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.CustomerProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeCustomerProfileFields(context.Context, organization.Actor, organization.CustomerProfileCommand, *organization.CustomerProfile) (organization.CustomerProfileFieldAuthorization, error) {
	return organization.CustomerProfileFieldAuthorization{CreditTerms: true, CreditLimit: true, BillingPreference: true, TaxTreatment: true}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeVendorProfile(context.Context, organization.Actor, organization.VendorProfileCommand, *organization.VendorProfile) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.VendorProfileManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeVendorProfileFields(context.Context, organization.Actor, organization.VendorProfileCommand, *organization.VendorProfile) (organization.VendorProfileFieldAuthorization, error) {
	return organization.VendorProfileFieldAuthorization{PaymentTerms: true, WithholdingTreatment: true, RemittancePreference: true}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeFiscalCalendar(context.Context, organization.Actor, organization.FiscalCalendarCommand, *organization.FiscalCalendar) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.FiscalCalendarManagementPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

func (permissiveOrganizationAuthorizer) AuthorizeMasterDataPublication(context.Context, organization.Actor, organization.MasterDataPublicationCommand, organization.MasterDataPublicationCandidate) (organization.AuthorizationDecision, error) {
	return organization.AuthorizationDecision{Allowed: true, Permission: organization.MasterDataPublicationPermission, DecisionReference: uuid.New(), ApprovedScopeIDs: []uuid.UUID{uuid.Nil}}, nil
}

type unavailablePartyBankControlEvaluator struct{}

func (unavailablePartyBankControlEvaluator) EvaluatePartyBankControl(context.Context, organization.PartyBankControlRequest) (organization.BankDetailControlState, error) {
	return organization.BankDetailControlState{}, organization.ErrPartyBankControlUnavailable
}

type unavailablePartyBankReferenceValidator struct{}

func (unavailablePartyBankReferenceValidator) ValidateAndCanonicalizePartyBankReferences(context.Context, organization.PartyBankReferenceValidationRequest) ([]organization.PartyBankDetailReference, error) {
	return nil, organization.ErrPartyBankReferenceUnavailable
}

func optionalInstrumentation(values ...*telemetry.Instrumentation) *telemetry.Instrumentation {
	if len(values) == 0 {
		return nil
	}
	return values[0]
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

func (authorizer evaluatorIdentityAuthorizer) AuthorizeEmergencyAccess(ctx context.Context, actor identity.ApplicationActor, command identity.EmergencyAccessGrantCommand, current *identity.EmergencyAccessGrant) (identity.AuthorizationDecision, error) {
	scopes := append([]string(nil), command.ScopeIDs...)
	if len(scopes) == 0 && current != nil {
		scopes = append(scopes, current.ScopeIDs...)
	}
	permission := identity.EmergencyAccessRevokePermission
	if command.Action == identity.EmergencyAccessActionGrant {
		permission = identity.EmergencyAccessGrantPermission
	}
	return authorizer.evaluate(ctx, actor, permission, scopes)
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
	if strings.EqualFold(strings.TrimSpace(getenv("TALLY_IAM_ALLOW_EMERGENCY_ACCESS")), "true") {
		policies = append(policies,
			identity.AccessPolicy{ID: uuid.New(), Version: "environment-emergency-access-grant-v1", Status: identity.AccessPolicyStatusActive, Permissions: []string{identity.EmergencyAccessGrantPermission}, EffectiveFrom: now, Rules: []identity.AccessRule{{ScopeIDs: configuredScopes(getenv)}}},
			identity.AccessPolicy{ID: uuid.New(), Version: "environment-emergency-access-revoke-v1", Status: identity.AccessPolicyStatusActive, Permissions: []string{identity.EmergencyAccessRevokePermission}, EffectiveFrom: now, Rules: []identity.AccessRule{{ScopeIDs: configuredScopes(getenv)}}},
		)
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

type environmentEmergencyAccessApproval struct {
	getenv func(string) string
}

func (approval environmentEmergencyAccessApproval) ValidateEmergencyAccessApproval(ctx context.Context, actor identity.ApplicationActor, command identity.EmergencyAccessGrantCommand, current *identity.EmergencyAccessGrant, fingerprint string) error {
	if !strings.EqualFold(strings.TrimSpace(approval.getenv("TALLY_IAM_ALLOW_EMERGENCY_ACCESS_APPROVAL")), "true") {
		return identity.ErrEmergencyAccessApprovalUnavailable
	}
	return identity.AllowAllEmergencyAccessApprovalPort{}.ValidateEmergencyAccessApproval(ctx, actor, command, current, fingerprint)
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
