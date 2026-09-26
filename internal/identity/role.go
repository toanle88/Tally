package identity

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/toanle88/Tally/internal/platform/aggregateversion"
)

var (
	ErrInvalidRole              = errors.New("invalid identity role")
	ErrInvalidRoleCommand       = errors.New("invalid identity role command")
	ErrRoleNotFound             = errors.New("identity role not found")
	ErrRoleRetired              = errors.New("identity role is retired")
	ErrInvalidPermissionGrant   = errors.New("invalid permission grant")
	ErrDuplicatePermissionGrant = errors.New("duplicate permission grant")
	ErrApprovalRequired         = errors.New("role approval is required")
	ErrApprovalRejected         = errors.New("role approval was rejected")
	ErrApprovalUnavailable      = errors.New("role approval is unavailable")
	ErrSegregationConflict      = errors.New("role segregation conflict")
	ErrSegregationUnavailable   = errors.New("role segregation evaluation unavailable")
	ErrRoleAuthorizationDenied  = errors.New("identity role authorization denied")
	ErrInvalidRoleService       = errors.New("invalid identity role service")
	ErrRoleIdempotencyConflict  = errors.New("identity role command idempotency conflict")
	ErrRoleCommandInProgress    = errors.New("identity role command is already in progress")
	ErrRoleDurableCommandFailed = errors.New("identity role command previously failed")
	ErrRoleAuditUnavailable     = errors.New("identity role audit recorder unavailable")
)

const (
	RoleStatusActive  = "active"
	RoleStatusRetired = "retired"

	RoleActionCreate = "create"
	RoleActionUpdate = "update"
	RoleActionRetire = "retire"

	RoleManagementPermission = "finance.iam.manage.roles"
)

type PermissionGrant struct {
	Permission    string
	ScopeIDs      []string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
}

func (grant PermissionGrant) Validate() error {
	permission := strings.TrimSpace(grant.Permission)
	if !isKnownPermission(permission) {
		return fmt.Errorf("%w: permission %q", ErrInvalidPermissionGrant, permission)
	}
	if grant.EffectiveFrom.IsZero() {
		return fmt.Errorf("%w: effective-from is required", ErrInvalidPermissionGrant)
	}
	if grant.EffectiveTo != nil && !grant.EffectiveTo.After(grant.EffectiveFrom) {
		return fmt.Errorf("%w: effective-to must be after effective-from", ErrInvalidPermissionGrant)
	}
	if len(grant.ScopeIDs) == 0 {
		return fmt.Errorf("%w: at least one scope is required", ErrInvalidPermissionGrant)
	}
	seen := make(map[string]struct{}, len(grant.ScopeIDs))
	for _, scopeID := range grant.ScopeIDs {
		scopeID = strings.TrimSpace(scopeID)
		if scopeID == "" {
			return fmt.Errorf("%w: scope is required", ErrInvalidPermissionGrant)
		}
		if _, exists := seen[scopeID]; exists {
			return fmt.Errorf("%w: scope %q", ErrDuplicatePermissionGrant, scopeID)
		}
		seen[scopeID] = struct{}{}
	}
	return nil
}

func normalizePermissionGrants(grants []PermissionGrant) ([]PermissionGrant, error) {
	result := make([]PermissionGrant, len(grants))
	seen := make(map[string]struct{})
	for index, grant := range grants {
		if err := grant.Validate(); err != nil {
			return nil, err
		}
		copyGrant := PermissionGrant{
			Permission:    strings.TrimSpace(grant.Permission),
			ScopeIDs:      append([]string(nil), grant.ScopeIDs...),
			EffectiveFrom: grant.EffectiveFrom.UTC(),
		}
		sort.Strings(copyGrant.ScopeIDs)
		if grant.EffectiveTo != nil {
			value := grant.EffectiveTo.UTC()
			copyGrant.EffectiveTo = &value
		}
		for _, scopeID := range copyGrant.ScopeIDs {
			key := copyGrant.Permission + "|" + scopeID + "|" + copyGrant.EffectiveFrom.Format(time.RFC3339Nano)
			if _, exists := seen[key]; exists {
				return nil, fmt.Errorf("%w: %s", ErrDuplicatePermissionGrant, key)
			}
			seen[key] = struct{}{}
		}
		result[index] = copyGrant
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Permission != result[right].Permission {
			return result[left].Permission < result[right].Permission
		}
		return result[left].EffectiveFrom.Before(result[right].EffectiveFrom)
	})
	return result, nil
}

var approvedPermissionCatalogue = map[string]struct{}{
	"finance.ap.apply.asset.clearing.classification":                  struct{}{},
	"finance.ap.apply.incoming.settlement":                            struct{}{},
	"finance.ap.apply.payment.return":                                 struct{}{},
	"finance.ap.apply.vendor.invoice.approval.decision":               struct{}{},
	"finance.ap.dispute.vendor.invoice":                               struct{}{},
	"finance.ap.register.vendor.invoice":                              struct{}{},
	"finance.ap.request.payment":                                      struct{}{},
	"finance.ap.reverse.incoming.settlement.application":              struct{}{},
	"finance.ap.validate.vendor.invoice":                              struct{}{},
	"finance.ap.void.vendor.invoice":                                  struct{}{},
	"finance.ar.apply.customer.refund.approval.decision":              struct{}{},
	"finance.ar.apply.customer.refund.payment.result":                 struct{}{},
	"finance.ar.apply.payment.return":                                 struct{}{},
	"finance.ar.apply.receipt":                                        struct{}{},
	"finance.ar.cancel.customer.refund.payment":                       struct{}{},
	"finance.ar.cancel.customer.refund.request":                       struct{}{},
	"finance.ar.create.customer.refund.request":                       struct{}{},
	"finance.ar.issue.credit.note":                                    struct{}{},
	"finance.ar.issue.customer.invoice":                               struct{}{},
	"finance.ar.record.customer.chargebacks":                          struct{}{},
	"finance.ar.record.receipt":                                       struct{}{},
	"finance.ar.record.receivable.write.offs":                         struct{}{},
	"finance.ar.request.customer.refund.payment":                      struct{}{},
	"finance.ar.resolve.customer.overpayments":                        struct{}{},
	"finance.ar.rollback.unposted.application.batch":                  struct{}{},
	"finance.ar.unapply.receipt":                                      struct{}{},
	"finance.aud.append.auditable.event":                              struct{}{},
	"finance.aud.create.audit.seal":                                   struct{}{},
	"finance.aud.escalate.integrity.incident":                         struct{}{},
	"finance.aud.rotate.verification.credential":                      struct{}{},
	"finance.aud.verify.proof":                                        struct{}{},
	"finance.bfr.complete.reconciliation":                             struct{}{},
	"finance.bfr.confirm.match":                                       struct{}{},
	"finance.bfr.import.statement":                                    struct{}{},
	"finance.bfr.maintain.bank.feed.connections":                      struct{}{},
	"finance.bfr.propose.match":                                       struct{}{},
	"finance.bfr.unmatch":                                             struct{}{},
	"finance.coa.apply.segment.change.approval.decision":              struct{}{},
	"finance.coa.maintain.segment.definitions":                        struct{}{},
	"finance.coa.maintain.segment.values":                             struct{}{},
	"finance.coa.request.segment.changes":                             struct{}{},
	"finance.coa.validate.segment.combinations":                       struct{}{},
	"finance.fa.apply.asset.disposal.approval.decision":               struct{}{},
	"finance.fa.apply.asset.settlement.result":                        struct{}{},
	"finance.fa.apply.asset.supplier.liability.result":                struct{}{},
	"finance.fa.apply.impairment.approval.decision":                   struct{}{},
	"finance.fa.apply.incoming.settlement":                            struct{}{},
	"finance.fa.apply.payment.return":                                 struct{}{},
	"finance.fa.cancel.unposted.asset.disposal":                       struct{}{},
	"finance.fa.capitalize.asset":                                     struct{}{},
	"finance.fa.compensate.failed.disposal.posting":                   struct{}{},
	"finance.fa.correct.posted.asset.disposals":                       struct{}{},
	"finance.fa.create.asset.acquisition.clearing":                    struct{}{},
	"finance.fa.create.disposal.settlement.clearing":                  struct{}{},
	"finance.fa.dispose.asset":                                        struct{}{},
	"finance.fa.reclassify.disposal.cost.for.payment":                 struct{}{},
	"finance.fa.record.impairment.assessments":                        struct{}{},
	"finance.fa.request.disposal.cost.payment":                        struct{}{},
	"finance.fa.request.disposal.cost.payment.replacement":            struct{}{},
	"finance.fa.reverse.incoming.settlement.application":              struct{}{},
	"finance.fa.run.depreciation":                                     struct{}{},
	"finance.fa.split.assets.or.components":                           struct{}{},
	"finance.fa.transfer.assets.or.components":                        struct{}{},
	"finance.fpm.abort.close.run":                                     struct{}{},
	"finance.fpm.apply.close.approval.decision":                       struct{}{},
	"finance.fpm.apply.close.exception.approval.decision":             struct{}{},
	"finance.fpm.apply.posting.gate.result":                           struct{}{},
	"finance.fpm.apply.reopen.approval.decision":                      struct{}{},
	"finance.fpm.end.soft.close":                                      struct{}{},
	"finance.fpm.extend.close.exception":                              struct{}{},
	"finance.fpm.request.reopen":                                      struct{}{},
	"finance.fpm.resume.close.run":                                    struct{}{},
	"finance.fpm.start.hard.close":                                    struct{}{},
	"finance.fpm.start.reclose":                                       struct{}{},
	"finance.fpm.start.soft.close":                                    struct{}{},
	"finance.fpm.take.over.period.control":                            struct{}{},
	"finance.fx.apply.revaluation.approval.decision":                  struct{}{},
	"finance.fx.post.revaluation.run":                                 struct{}{},
	"finance.fx.publish.rate.set":                                     struct{}{},
	"finance.fx.run.revaluation":                                      struct{}{},
	"finance.fx.run.translation":                                      struct{}{},
	"finance.gl.acquire.posting.barrier":                              struct{}{},
	"finance.gl.apply.journal.approval.decision":                      struct{}{},
	"finance.gl.begin.reclose.gate":                                   struct{}{},
	"finance.gl.close.operational.reopen.gate":                        struct{}{},
	"finance.gl.close.scoped.reopen.gate":                             struct{}{},
	"finance.gl.enter.soft.close.gate":                                struct{}{},
	"finance.gl.exit.soft.close.gate":                                 struct{}{},
	"finance.gl.finalize.posting.gate":                                struct{}{},
	"finance.gl.get.posting.gate.status":                              struct{}{},
	"finance.gl.maintain.accounting.books":                            struct{}{},
	"finance.gl.maintain.accounts.and.reporting.mappings":             struct{}{},
	"finance.gl.maintain.charts.of.accounts":                          struct{}{},
	"finance.gl.maintain.ledgers":                                     struct{}{},
	"finance.gl.open.operational.reopen.gate":                         struct{}{},
	"finance.gl.open.scoped.reopen.gate":                              struct{}{},
	"finance.gl.release.posting.barrier":                              struct{}{},
	"finance.gl.reverse.journal.entry":                                struct{}{},
	"finance.gl.submit.posting.request":                               struct{}{},
	"finance.iam.grant.emergency.access":                              struct{}{},
	"finance.iam.manage.access.policies":                              struct{}{},
	"finance.iam.manage.roles":                                        struct{}{},
	"finance.iam.manage.segregation.rules":                            struct{}{},
	"finance.iam.manage.users":                                        struct{}{},
	"finance.iam.revoke.emergency.access":                             struct{}{},
	"finance.ic.apply.incoming.settlement":                            struct{}{},
	"finance.ic.apply.payment.return":                                 struct{}{},
	"finance.ic.apply.residual.approval.decision":                     struct{}{},
	"finance.ic.complete.settlement.run":                              struct{}{},
	"finance.ic.create.settlement.instructions":                       struct{}{},
	"finance.ic.maintain.intercompany.agreements":                     struct{}{},
	"finance.ic.match.intercompany.items":                             struct{}{},
	"finance.ic.record.intercompany.transactions":                     struct{}{},
	"finance.ic.reverse.incoming.settlement.application":              struct{}{},
	"finance.ic.run.elimination":                                      struct{}{},
	"finance.ic.start.settlement":                                     struct{}{},
	"finance.inv.cancel.unfinalized.invoices":                         struct{}{},
	"finance.inv.configure.billing.schedules":                         struct{}{},
	"finance.inv.configure.invoice.templates":                         struct{}{},
	"finance.inv.finalize.generated.invoices":                         struct{}{},
	"finance.inv.generate.invoices":                                   struct{}{},
	"finance.inv.recalculate.unfinalized.invoices":                    struct{}{},
	"finance.omd.maintain.customer.profiles":                          struct{}{},
	"finance.omd.maintain.fiscal.calendars":                           struct{}{},
	"finance.omd.maintain.legal.entities":                             struct{}{},
	"finance.omd.maintain.parties":                                    struct{}{},
	"finance.omd.maintain.vendor.profiles":                            struct{}{},
	"finance.omd.publish.approved.master.data.changes":                struct{}{},
	"finance.payr.apply.payment.return":                               struct{}{},
	"finance.payr.apply.payroll.run.approval.decision":                struct{}{},
	"finance.payr.calculate.payroll.run":                              struct{}{},
	"finance.payr.create.payroll.correction":                          struct{}{},
	"finance.payr.maintain.employee.payroll.profiles":                 struct{}{},
	"finance.payr.maintain.payroll.tax.filing.records":                struct{}{},
	"finance.payr.post.payroll.run":                                   struct{}{},
	"finance.pcm.acknowledge.incoming.settlement":                     struct{}{},
	"finance.pcm.acknowledge.payment.return":                          struct{}{},
	"finance.pcm.apply.payment.batch.approval.decision":               struct{}{},
	"finance.pcm.apply.payment.instruction.exception.decision":        struct{}{},
	"finance.pcm.cancel.expected.incoming.settlement":                 struct{}{},
	"finance.pcm.cancel.payment.batch":                                struct{}{},
	"finance.pcm.cancel.payment.instruction":                          struct{}{},
	"finance.pcm.cancel.unposted.payment.return":                      struct{}{},
	"finance.pcm.cancel.unposted.settlement.receipt":                  struct{}{},
	"finance.pcm.close.expected.incoming.settlement":                  struct{}{},
	"finance.pcm.create.payment.instruction.from.obligation":          struct{}{},
	"finance.pcm.maintain.bank.accounts":                              struct{}{},
	"finance.pcm.prepare.payment.batch":                               struct{}{},
	"finance.pcm.record.incoming.settlement":                          struct{}{},
	"finance.pcm.record.payment.return":                               struct{}{},
	"finance.pcm.record.unallocated.incoming.settlement":              struct{}{},
	"finance.pcm.register.expected.incoming.settlement":               struct{}{},
	"finance.pcm.resolve.expected.incoming.settlement.exception":      struct{}{},
	"finance.pcm.resolve.incoming.settlement.owner.exception":         struct{}{},
	"finance.pcm.resolve.payment.return.exception":                    struct{}{},
	"finance.pcm.resolve.settlement.receipt.validation.exception":     struct{}{},
	"finance.pcm.resolve.unallocated.incoming.settlement":             struct{}{},
	"finance.pcm.retry.payment.instruction":                           struct{}{},
	"finance.pcm.reverse.incoming.settlement":                         struct{}{},
	"finance.pcm.submit.payment.instruction":                          struct{}{},
	"finance.rev.apply.contract.modification.approval.decision":       struct{}{},
	"finance.rev.apply.revenue.schedule.approval.decision":            struct{}{},
	"finance.rev.assess.contract":                                     struct{}{},
	"finance.rev.modify.contract":                                     struct{}{},
	"finance.rev.publish.revenue.accounting.profile":                  struct{}{},
	"finance.rev.run.recognition":                                     struct{}{},
	"finance.rpt.apply.consolidation.approval.decision":               struct{}{},
	"finance.rpt.apply.translation.result":                            struct{}{},
	"finance.rpt.generate.and.publish.ledger.financial.statements":    struct{}{},
	"finance.rpt.maintain.report.definitions":                         struct{}{},
	"finance.rpt.publish.consolidated.statement":                      struct{}{},
	"finance.rpt.run.consolidation":                                   struct{}{},
	"finance.tax.apply.incoming.settlement":                           struct{}{},
	"finance.tax.apply.payment.return":                                struct{}{},
	"finance.tax.apply.return.level.tax.adjustment.approval.decision": struct{}{},
	"finance.tax.apply.tax.amendment.approval.decision":               struct{}{},
	"finance.tax.apply.tax.return.approval.decision":                  struct{}{},
	"finance.tax.create.return.level.tax.adjustment":                  struct{}{},
	"finance.tax.create.tax.amendment":                                struct{}{},
	"finance.tax.determine.tax":                                       struct{}{},
	"finance.tax.maintain.tax.configurations":                         struct{}{},
	"finance.tax.post.return.level.tax.adjustment":                    struct{}{},
	"finance.tax.prepare.tax.return":                                  struct{}{},
	"finance.tax.record.tax.payment.settlement":                       struct{}{},
	"finance.tax.request.tax.payment":                                 struct{}{},
	"finance.tax.reverse.incoming.settlement.application":             struct{}{},
	"finance.tax.submit.tax.amendment":                                struct{}{},
	"finance.tax.submit.tax.return":                                   struct{}{},
	"finance.wfa.create.approval.request":                             struct{}{},
	"finance.wfa.decide.approval.request":                             struct{}{},
	"finance.wfa.delegate.approval":                                   struct{}{},
	"finance.wfa.escalate.approval":                                   struct{}{},
	"finance.wfa.maintain.approval.policies":                          struct{}{},
}

func isKnownPermission(permission string) bool {
	_, ok := approvedPermissionCatalogue[permission]
	return ok
}

type ApprovalDecisionReference struct {
	ApprovalRequestID    uuid.UUID
	DecisionID           uuid.UUID
	PolicyVersion        string
	DecisionVersion      int64
	SubjectVersion       int64
	CandidateFingerprint string
	ApproverUserID       uuid.UUID
}

func (reference ApprovalDecisionReference) Validate() error {
	if reference.ApprovalRequestID == uuid.Nil || reference.DecisionID == uuid.Nil || reference.ApproverUserID == uuid.Nil {
		return ErrApprovalRequired
	}
	if strings.TrimSpace(reference.PolicyVersion) == "" || strings.TrimSpace(reference.CandidateFingerprint) == "" ||
		reference.DecisionVersion < 1 || reference.SubjectVersion < 1 {
		return ErrApprovalRequired
	}
	return nil
}

type Role struct {
	ID             uuid.UUID
	Name           string
	Status         string
	Grants         []PermissionGrant
	Version        aggregateversion.AggregateVersion
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Approval       ApprovalDecisionReference
	AuditReference uuid.UUID
}

func NewRole(id uuid.UUID, name string, grants []PermissionGrant, approval ApprovalDecisionReference, now time.Time) (Role, error) {
	if id == uuid.Nil {
		return Role{}, fmt.Errorf("%w: id is required", ErrInvalidRole)
	}
	normalizedGrants, err := normalizePermissionGrants(grants)
	if err != nil {
		return Role{}, err
	}
	role := Role{
		ID: id, Name: strings.TrimSpace(name), Status: RoleStatusActive, Grants: normalizedGrants,
		Version: aggregateversion.Initial(), CreatedAt: now.UTC(), UpdatedAt: now.UTC(), Approval: approval,
	}
	if err := role.Validate(); err != nil {
		return Role{}, err
	}
	return role, nil
}

func (role Role) Validate() error {
	if role.ID == uuid.Nil || strings.TrimSpace(role.Name) == "" || len([]rune(role.Name)) > 120 {
		return fmt.Errorf("%w: role identity and name are required", ErrInvalidRole)
	}
	if role.Status != RoleStatusActive && role.Status != RoleStatusRetired {
		return fmt.Errorf("%w: unsupported status %q", ErrInvalidRole, role.Status)
	}
	if role.Version.Value() < 1 || role.CreatedAt.IsZero() || role.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: version and timestamps are required", ErrInvalidRole)
	}
	if err := role.Approval.Validate(); err != nil {
		return err
	}
	normalized, err := normalizePermissionGrants(role.Grants)
	if err != nil {
		return err
	}
	if role.Status == RoleStatusActive && len(normalized) == 0 {
		return fmt.Errorf("%w: active role requires a permission grant", ErrInvalidRole)
	}
	return nil
}

func (role *Role) Replace(name string, grants []PermissionGrant, approval ApprovalDecisionReference, now time.Time) error {
	if role == nil {
		return ErrInvalidRole
	}
	if role.Status == RoleStatusRetired {
		return ErrRoleRetired
	}
	normalized, err := normalizePermissionGrants(grants)
	if err != nil {
		return err
	}
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%w: role name is required", ErrInvalidRole)
	}
	role.Name = strings.TrimSpace(name)
	role.Grants = normalized
	role.Approval = approval
	role.UpdatedAt = now.UTC()
	return nil
}

func (role *Role) Retire(approval ApprovalDecisionReference, now time.Time) error {
	if role == nil {
		return ErrInvalidRole
	}
	if role.Status == RoleStatusRetired {
		return ErrRoleRetired
	}
	role.Status = RoleStatusRetired
	role.Grants = nil
	role.Approval = approval
	role.UpdatedAt = now.UTC()
	return nil
}

func clonePermissionGrants(grants []PermissionGrant) []PermissionGrant {
	result := make([]PermissionGrant, len(grants))
	for index, grant := range grants {
		result[index] = PermissionGrant{
			Permission:    grant.Permission,
			ScopeIDs:      append([]string(nil), grant.ScopeIDs...),
			EffectiveFrom: grant.EffectiveFrom,
		}
		if grant.EffectiveTo != nil {
			value := *grant.EffectiveTo
			result[index].EffectiveTo = &value
		}
	}
	return result
}

func cloneRole(role Role) Role {
	role.Grants = clonePermissionGrants(role.Grants)
	return role
}
