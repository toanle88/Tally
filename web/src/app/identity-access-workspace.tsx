import { useMemo, useState } from 'react'

import { SensitiveDataGuard } from '@/components/sensitive-data-guard'
import type { SensitiveAccessAuditEvent } from '@/components/record-context'
import { ValidationSummary } from '@/components/validation-summary'
import { VersionConflictDialog } from '@/components/version-conflict-dialog'
import type { VersionConflictSnapshot, ValidationIssue } from '@/components/workflow-context'
import { Button, DataTable, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'

type ManagedUser = {
  id: string
  label: string
  subjectReference: string
  status: 'inactive' | 'active' | 'suspended' | 'terminated'
  roles: string[]
  scopes: string[]
  version: number
  reviewEvidence: string
}

type ManagedRole = {
  id: string
  name: string
  status: 'active' | 'retired'
  version: number
  approvalStatus: 'approved' | 'pending' | 'rejected'
  grants: Array<{ permission: string; scopes: string[]; effectiveFrom: string; effectiveTo?: string }>
}

type ManagedSegregationRule = {
  id: string
  code: string
  name: string
  status: "active" | "retired"
  version: number
  mode: "block" | "exception-required"
  permissions: string[]
  scopes: string[]
  threshold?: string
  coolingOffHours?: number
  approvalStatus: "approved" | "pending" | "rejected"
}

type EmergencyAccessGrant = {
  id: string
  targetActor: string
  status: 'active' | 'revoked' | 'expired'
  permissions: string[]
  scopes: string[]
  reasonCode: string
  approver: string
  startsAt: string
  expiresAt: string
  policyVersion: string
  reviewStatus: 'pending' | 'completed' | 'overdue'
  reviewOutcome?: string
  reviewDueAt: string
  version: number
}

type EmergencyAccessOutcome = 'ready' | 'denied' | 'duplicate' | 'conflict' | 'step-up-required'

type SegregationDecisionOutcome = "allowed" | "conflict" | "exception-required" | "stale" | "unavailable"
type SegregationDecisionExplanation = {
  outcome: SegregationDecisionOutcome
  reason: string
  resolution: string
  ruleVersion: string
  decisionReference: string
}

const initialSegregationRules: ManagedSegregationRule[] = [
  { id: "rule-001", code: "payment-batch-preparation-approval", name: "Payment batch preparer and approver", status: "active", version: 1, mode: "exception-required", permissions: ["finance.pcm.prepare.payment.batch", "finance.pcm.apply.payment.batch.approval.decision"], scopes: ["all accounting scopes"], approvalStatus: "approved" },
  { id: "rule-002", code: "fiscal-period-reopen-request-approval", name: "Fiscal-period reopen requester and approver", status: "active", version: 1, mode: "block", permissions: ["finance.fpm.request.reopen", "finance.fpm.apply.reopen.approval.decision"], scopes: ["all accounting scopes"], approvalStatus: "approved" },
  { id: "rule-003", code: "vendor-bank-detail-payment-release-cooling-off", name: "Vendor bank-detail cooling-off", status: "active", version: 1, mode: "block", permissions: ["finance.omd.maintain.vendor.profiles", "finance.pcm.submit.payment.instruction"], scopes: ["all accounting scopes"], coolingOffHours: 24, approvalStatus: "approved" },
  { id: "rule-004", code: "manual-journal-self-approval-threshold", name: "Manual-journal self approval threshold", status: "active", version: 1, mode: "block", permissions: ["finance.gl.submit.posting.request", "finance.gl.apply.journal.approval.decision"], scopes: ["all accounting scopes"], threshold: "10000.00", approvalStatus: "approved" },
  { id: "rule-005", code: "payroll-detail-summary-ledger", name: "Payroll detail and summary ledger", status: "active", version: 1, mode: "block", permissions: ["finance.payr.maintain.employee.payroll.profiles", "finance.rpt.generate.and.publish.ledger.financial.statements"], scopes: ["all accounting scopes"], approvalStatus: "approved" },
  { id: "rule-006", code: "independent-policy-approval", name: "Independent policy approval", status: "active", version: 1, mode: "block", permissions: ["finance.iam.manage.access.policies", "finance.wfa.decide.approval.request"], scopes: ["all accounting scopes"], approvalStatus: "approved" },
]

const segregationDecisionFixtures: Record<SegregationDecisionOutcome, SegregationDecisionExplanation> = {
  allowed: { outcome: "allowed", reason: "No active rule matched the requested action and actor history.", resolution: "Continue with the requested action.", ruleVersion: "not applicable", decisionReference: "decision-sod-allowed-001" },
  conflict: { outcome: "conflict", reason: "The actor history shows preparation and approval by the same actor.", resolution: "Use an independent actor or request an approved exception where the rule permits one.", ruleVersion: "payment-batch-preparation-approval v1", decisionReference: "decision-sod-conflict-002" },
  "exception-required": { outcome: "exception-required", reason: "This conflict requires a current, independently approved exception.", resolution: "Request an exception with a reason, approver, expiry, and matching rule version.", ruleVersion: "payment-batch-preparation-approval v1", decisionReference: "decision-sod-exception-003" },
  stale: { outcome: "stale", reason: "The rule or policy version changed after the action was prepared.", resolution: "Refresh current IAM policy state and retry.", ruleVersion: "payment-batch-preparation-approval v2", decisionReference: "decision-sod-stale-004" },
  unavailable: { outcome: "unavailable", reason: "Required rule or actor-history state is unavailable.", resolution: "Retry after the IAM policy dependency is available; no side effect was applied.", ruleVersion: "unavailable", decisionReference: "decision-sod-unavailable-005" },
}

const segregationDecisionState: Record<SegregationDecisionOutcome, SemanticState> = {
  allowed: "success", conflict: "error", "exception-required": "pending", stale: "pending", unavailable: "warning",
}

type AccessDecisionOutcome = 'allowed' | 'denied' | 'expired' | 'unavailable' | 'stale'
type AccessDecisionExplanation = {
  outcome: AccessDecisionOutcome
  policyVersion: string
  decisionReference: string
  matchedDimension: string
  nextAction: string
  recordAccess: boolean
}

const accessDecisionFixtures: Record<AccessDecisionOutcome, AccessDecisionExplanation> = {
  allowed: { outcome: 'allowed', policyVersion: 'policy-2026.09-v4', decisionReference: 'decision-allowed-001', matchedDimension: 'Legal entity + segment + action', nextAction: 'Continue with the requested record action.', recordAccess: true },
  denied: { outcome: 'denied', policyVersion: 'policy-2026.09-v4', decisionReference: 'decision-denied-002', matchedDimension: 'Requested scope', nextAction: 'Choose a scope included in the current policy decision.', recordAccess: false },
  expired: { outcome: 'expired', policyVersion: 'policy-2026.08-v3', decisionReference: 'decision-expired-003', matchedDimension: 'Effective date', nextAction: 'Request a current policy revision before retrying.', recordAccess: false },
  unavailable: { outcome: 'unavailable', policyVersion: 'policy-2026.09-v4', decisionReference: 'decision-unavailable-004', matchedDimension: 'Policy dependency', nextAction: 'Retry after the identity policy dependency is available.', recordAccess: false },
  stale: { outcome: 'stale', policyVersion: 'policy-2026.09-v5', decisionReference: 'decision-stale-005', matchedDimension: 'Expected policy version', nextAction: 'Refresh policy state and retry with the current version.', recordAccess: false },
}

const accessDecisionState: Record<AccessDecisionOutcome, SemanticState> = {
  allowed: 'success',
  denied: 'error',
  expired: 'disabled',
  unavailable: 'warning',
  stale: 'pending',
}

const initialUsers: ManagedUser[] = [
  { id: 'user-001', label: 'Finance operator 01', subjectReference: 'subject-ref-01', status: 'active', roles: ['finance.viewer', 'close.reviewer'], scopes: ['entity-vietnam', 'entity-singapore'], version: 8, reviewEvidence: 'Policy review 2026-09-20' },
  { id: 'user-002', label: 'AP operator 02', subjectReference: 'subject-ref-02', status: 'suspended', roles: ['payables.operator'], scopes: ['entity-vietnam'], version: 5, reviewEvidence: 'Suspended by access review' },
  { id: 'user-003', label: 'New application user', subjectReference: 'subject-ref-03', status: 'inactive', roles: [], scopes: [], version: 1, reviewEvidence: 'Awaiting activation' },
]

const initialRoles: ManagedRole[] = [
  { id: 'role-001', name: 'Scoped finance operator', status: 'active', version: 4, approvalStatus: 'approved', grants: [{ permission: 'finance.gl.submit.posting.request', scopes: ['entity-vietnam'], effectiveFrom: '2026-10-01' }, { permission: 'finance.ap.register.vendor.invoice', scopes: ['entity-vietnam'], effectiveFrom: '2026-10-01', effectiveTo: '2027-01-01' }] },
  { id: 'role-002', name: 'Close reviewer', status: 'active', version: 2, approvalStatus: 'pending', grants: [{ permission: 'finance.gl.apply.journal.approval.decision', scopes: ['entity-singapore'], effectiveFrom: '2026-09-01' }] },
  { id: 'role-003', name: 'Retired legacy role', status: 'retired', version: 7, approvalStatus: 'approved', grants: [] },
]

const initialEmergencyAccessGrants: EmergencyAccessGrant[] = [
  { id: 'grant-001', targetActor: 'Finance operator 01', status: 'active', permissions: ['finance.gl.submit.posting.request'], scopes: ['entity-vietnam'], reasonCode: 'break-fix', approver: 'Access approver 02', startsAt: '2026-09-27T08:00:00Z', expiresAt: '2026-09-27T10:00:00Z', policyVersion: 'emergency-policy-v1', reviewStatus: 'pending', reviewDueAt: '2026-09-28T08:00:00Z', version: 1 },
  { id: 'grant-002', targetActor: 'AP operator 02', status: 'revoked', permissions: ['finance.pcm.submit.payment.instruction'], scopes: ['entity-vietnam'], reasonCode: 'supplier-payment-recovery', approver: 'Access approver 02', startsAt: '2026-09-24T09:00:00Z', expiresAt: '2026-09-24T11:00:00Z', policyVersion: 'emergency-policy-v1', reviewStatus: 'completed', reviewOutcome: 'review-code-001', reviewDueAt: '2026-09-25T09:00:00Z', version: 3 },
  { id: 'grant-003', targetActor: 'Close reviewer', status: 'expired', permissions: ['finance.gl.apply.journal.approval.decision'], scopes: ['entity-singapore'], reasonCode: 'close-window-support', approver: 'Access approver 03', startsAt: '2026-09-20T06:00:00Z', expiresAt: '2026-09-20T10:00:00Z', policyVersion: 'emergency-policy-v1', reviewStatus: 'overdue', reviewDueAt: '2026-09-21T06:00:00Z', version: 1 },
]

const roleStatusState: Record<ManagedRole['status'], SemanticState> = {
  active: 'success',
  retired: 'disabled',
}

const emergencyStatusState: Record<EmergencyAccessGrant['status'], SemanticState> = {
  active: 'success', revoked: 'disabled', expired: 'error',
}

const emergencyReviewState: Record<EmergencyAccessGrant['reviewStatus'], SemanticState> = {
  pending: 'pending', completed: 'success', overdue: 'warning',
}

const statusState: Record<ManagedUser['status'], SemanticState> = {
  inactive: 'pending',
  active: 'success',
  suspended: 'disabled',
  terminated: 'error',
}

const conflictFixture: VersionConflictSnapshot = {
  kind: 'version',
  attemptedAction: 'Replace complete assignment set',
  record: { label: 'Finance operator 01', href: '/administration/identity-access' },
  scope: { id: 'identity-administration', label: 'Identity administration' },
  expected: { version: 8, state: 'active' },
  current: { version: 9, state: 'active' },
  changedValues: [
    { id: 'roles', label: 'Role assignments', reviewedValue: 'finance.viewer', currentValue: 'finance.viewer, close.reviewer' },
    { id: 'review', label: 'Review evidence', reviewedValue: 'Pending', currentValue: 'Policy review 2026-09-20' },
  ],
  currentOwner: { id: 'identity', label: 'Identity and access' },
  processEpoch: 'identity-read-2026-09-25',
  safeRetry: { permitted: false, reason: 'Refresh the authoritative record before retrying the replacement.' },
}

export function IdentityAccessWorkspace() {
  const [users, setUsers] = useState(initialUsers)
  const [selectedId, setSelectedId] = useState(initialUsers[0].id)
  const [role, setRole] = useState('finance.viewer')
  const [scope, setScope] = useState('entity-vietnam')
  const [issues, setIssues] = useState<ValidationIssue[]>([])
  const [notice, setNotice] = useState('Authoritative status, role, and scope values are re-evaluated before each material action.')
  const [conflictOpen, setConflictOpen] = useState(false)
  const [roles, setRoles] = useState(initialRoles)
  const [selectedRoleId, setSelectedRoleId] = useState(initialRoles[0].id)
  const [rolePermission, setRolePermission] = useState('finance.gl.submit.posting.request')
  const [roleScope, setRoleScope] = useState('entity-vietnam')
  const [roleIssues, setRoleIssues] = useState<ValidationIssue[]>([])
  const [roleNotice, setRoleNotice] = useState('Approval, segregation, scope containment, and version checks run before a role revision becomes active.')
  const [roleConflictOpen, setRoleConflictOpen] = useState(false)
  const [emergencyGrants, setEmergencyGrants] = useState(initialEmergencyAccessGrants)
  const [selectedEmergencyGrantId, setSelectedEmergencyGrantId] = useState(initialEmergencyAccessGrants[0].id)
  const [emergencyTarget, setEmergencyTarget] = useState('Finance operator 01')
  const [emergencyPermission, setEmergencyPermission] = useState('finance.gl.submit.posting.request')
  const [emergencyScope, setEmergencyScope] = useState('entity-vietnam')
  const [emergencyReason, setEmergencyReason] = useState('break-fix')
  const [emergencyReviewOutcome, setEmergencyReviewOutcome] = useState('review-code-001')
  const [emergencyIssues, setEmergencyIssues] = useState<ValidationIssue[]>([])
  const [emergencyNotice, setEmergencyNotice] = useState('Emergency grants are time-bound, independently approved, and reviewed after use.')
  const [emergencyOutcome, setEmergencyOutcome] = useState<EmergencyAccessOutcome>('ready')
  const [emergencyConflictOpen, setEmergencyConflictOpen] = useState(false)
  const [decisionOutcome, setDecisionOutcome] = useState<AccessDecisionOutcome>('allowed')
  const [segregationRules, setSegregationRules] = useState(initialSegregationRules)
  const [selectedSegregationRuleId, setSelectedSegregationRuleId] = useState(initialSegregationRules[0].id)
  const [segregationOutcome, setSegregationOutcome] = useState<SegregationDecisionOutcome>('allowed')
  const [segregationNotice, setSegregationNotice] = useState('Rule revisions require independent approval, expected version, idempotency, and audit linkage.')

  const selected = users.find((user) => user.id === selectedId) ?? users[0]
  const selectedRole = roles.find((role) => role.id === selectedRoleId) ?? roles[0]
  const selectedEmergencyGrant = emergencyGrants.find((grant) => grant.id === selectedEmergencyGrantId) ?? emergencyGrants[0]
  const selectedSegregationRule = segregationRules.find((rule) => rule.id === selectedSegregationRuleId) ?? segregationRules[0]
  const recordSensitiveAccess = (event: SensitiveAccessAuditEvent) => {
    setNotice(`${event.outcome === 'allowed' ? 'Allowed' : 'Denied'} ${event.action} evidence recorded for decision ${event.decisionReference}.`)
    return true
  }
  const segregationColumns = useMemo(() => [
    { key: "rule", header: "Rule", rowHeader: true, render: (rule: ManagedSegregationRule) => <button type="button" className="link link-primary text-left font-semibold" onClick={() => setSelectedSegregationRuleId(rule.id)}>{rule.name}</button> },
    { key: "mode", header: "Mode", render: (rule: ManagedSegregationRule) => rule.mode },
    { key: "status", header: "Status", render: (rule: ManagedSegregationRule) => <StatusBadge state={rule.status === "active" ? "success" : "disabled"} label={rule.status} /> },
    { key: "version", header: "Version", render: (rule: ManagedSegregationRule) => <>v{rule.version}</> },
    { key: "approval", header: "Approval", render: (rule: ManagedSegregationRule) => <StatusBadge state={rule.approvalStatus === "approved" ? "success" : rule.approvalStatus === "pending" ? "pending" : "error"} label={rule.approvalStatus} /> },
    { key: "action", header: "Action", render: (rule: ManagedSegregationRule) => <Button size="sm" variant="secondary" onClick={() => setSelectedSegregationRuleId(rule.id)}>Review</Button> },
  ] as const, [])

  const roleColumns = useMemo(() => [
    { key: 'role', header: 'Role', rowHeader: true, render: (role: ManagedRole) => <button type="button" className="link link-primary text-left font-semibold" onClick={() => setSelectedRoleId(role.id)}>{role.name}</button> },
    { key: 'status', header: 'Status', render: (role: ManagedRole) => <StatusBadge state={roleStatusState[role.status]} label={role.status} /> },
    { key: 'grants', header: 'Grants', render: (role: ManagedRole) => role.grants.length ? role.grants.length : 'No grants' },
    { key: 'approval', header: 'Approval', render: (role: ManagedRole) => <StatusBadge state={role.approvalStatus === 'approved' ? 'success' : role.approvalStatus === 'pending' ? 'pending' : 'error'} label={role.approvalStatus} /> },
    { key: 'version', header: 'Version', render: (role: ManagedRole) => <>v{role.version}</> },
    { key: 'action', header: 'Action', render: (role: ManagedRole) => <Button size="sm" variant="secondary" onClick={() => setSelectedRoleId(role.id)}>Review</Button> },
  ] as const, [])

  const columns = useMemo(() => [
    { key: 'user', header: 'User', rowHeader: true, render: (user: ManagedUser) => <button type="button" className="link link-primary text-left font-semibold" onClick={() => setSelectedId(user.id)}>{user.label}</button> },
    { key: 'status', header: 'Status', render: (user: ManagedUser) => <StatusBadge state={statusState[user.status]} label={user.status} /> },
    { key: 'roles', header: 'Roles', render: (user: ManagedUser) => user.roles.length ? user.roles.join(', ') : 'No assignments' },
    { key: 'scopes', header: 'Scopes', render: (user: ManagedUser) => user.scopes.length ? user.scopes.join(', ') : 'No approved scope' },
    { key: 'version', header: 'Version', render: (user: ManagedUser) => <>v{user.version}</> },
    { key: 'review', header: 'Review evidence', render: (user: ManagedUser) => user.reviewEvidence },
    { key: 'action', header: 'Action', render: (user: ManagedUser) => <Button size="sm" variant="secondary" onClick={() => setSelectedId(user.id)}>Review</Button> },
  ] as const, [])

  const emergencyColumns = useMemo(() => [
    { key: 'grant', header: 'Grant reference', rowHeader: true, render: (grant: EmergencyAccessGrant) => <button type="button" className="link link-primary text-left font-semibold" onClick={() => setSelectedEmergencyGrantId(grant.id)}>{grant.id}</button> },
    { key: 'target', header: 'Target actor', render: (grant: EmergencyAccessGrant) => grant.targetActor },
    { key: 'status', header: 'Access state', render: (grant: EmergencyAccessGrant) => <StatusBadge state={emergencyStatusState[grant.status]} label={grant.status} /> },
    { key: 'expiry', header: 'Expiry', render: (grant: EmergencyAccessGrant) => <time dateTime={grant.expiresAt}>{grant.expiresAt}</time> },
    { key: 'review', header: 'Post-use review', render: (grant: EmergencyAccessGrant) => <StatusBadge state={emergencyReviewState[grant.reviewStatus]} label={grant.reviewStatus} /> },
    { key: 'action', header: 'Action', render: (grant: EmergencyAccessGrant) => <Button size="sm" variant="secondary" onClick={() => setSelectedEmergencyGrantId(grant.id)}>Review</Button> },
  ] as const, [])

  const retireSegregationRule = () => {
    if (!selectedSegregationRule || selectedSegregationRule.status === 'retired') return
    setSegregationRules((current) => current.map((rule) => rule.id === selectedSegregationRule.id ? { ...rule, status: 'retired', version: rule.version + 1, approvalStatus: 'approved' } : rule))
    setSegregationNotice(selectedSegregationRule.name + ' was retired non-destructively; historical revisions remain available.')
  }

  const runLifecycleAction = (nextStatus: ManagedUser['status']) => {
    if (!selected || selected.status === 'terminated') return
    setUsers((current) => current.map((user) => user.id === selected.id ? { ...user, status: nextStatus, version: user.version + 1, reviewEvidence: 'Reviewed locally at version ' + (user.version + 1) } : user))
    setNotice(selected.label + ' is now ' + nextStatus + '. The next authorization check must use this current status.')
  }

  const replaceAssignments = () => {
    const nextIssues: ValidationIssue[] = []
    if (!role.trim()) nextIssues.push({ id: 'role', category: 'field', code: 'required', message: 'Enter an approved role identity.', targetId: 'role-assignment', targetLabel: 'Role identity' })
    if (!scope.trim()) nextIssues.push({ id: 'scope', category: 'field', code: 'required', message: 'Enter an approved entity scope.', targetId: 'entity-scope', targetLabel: 'Entity scope' })
    if (role.trim() === 'duplicate') nextIssues.push({ id: 'duplicate', category: 'business-rule', code: 'duplicate-assignment', message: 'This result is a safe duplicate; no assignment was changed.', targetId: 'role-assignment', targetLabel: 'Role identity', nextAction: 'Use a new idempotency key only when the command data changes intentionally.' })
    if (scope.trim() === 'outside-approved-scope') nextIssues.push({ id: 'scope-denied', category: 'authorization', code: 'authorization-denied', message: 'The complete assignment set is outside the administering actor scope.', targetId: 'entity-scope', targetLabel: 'Entity scope', nextAction: 'Choose a scope included in the current policy decision.' })
    setIssues(nextIssues)
    if (nextIssues.length) {
      setNotice('The command was not sent and no assignment side effect was applied.')
      return
    }
    if (!selected || selected.status === 'terminated') return
    setUsers((current) => current.map((user) => user.id === selected.id ? { ...user, roles: [role.trim()], scopes: [scope.trim()], version: user.version + 1, reviewEvidence: 'Assignment replacement reviewed at version ' + (user.version + 1) } : user))
    setNotice('Complete assignment set replaced for ' + selected.label + '.')
  }

  const replaceRoleGrant = () => {
    const nextIssues: ValidationIssue[] = []
    if (!rolePermission.trim()) nextIssues.push({ id: 'role-permission', category: 'field', code: 'required', message: 'Enter a permission from the approved catalogue.', targetId: 'role-permission', targetLabel: 'Permission' })
    if (!roleScope.trim()) nextIssues.push({ id: 'role-scope', category: 'field', code: 'required', message: 'Enter an opaque scope reference.', targetId: 'role-scope', targetLabel: 'Scope reference' })
    if (rolePermission.trim() === 'duplicate') nextIssues.push({ id: 'role-duplicate', category: 'business-rule', code: 'duplicate-grant', message: 'The grant duplicates an existing permission, scope, and effective-date tuple.', targetId: 'role-permission', targetLabel: 'Permission', nextAction: 'Choose a distinct approved grant or review the existing revision.' })
    if (roleScope.trim() === 'outside-approved-scope') nextIssues.push({ id: 'role-scope-denied', category: 'authorization', code: 'authorization-denied', message: 'The grant scope is outside the administering actor scope.', targetId: 'role-scope', targetLabel: 'Scope reference', nextAction: 'Choose a scope included in the current policy decision.' })
    setRoleIssues(nextIssues)
    if (nextIssues.length || !selectedRole || selectedRole.status === 'retired') {
      setRoleNotice(nextIssues.length ? 'The candidate revision was rejected and no role state changed.' : 'Retired roles remain historical and cannot receive new grants.')
      return
    }
    setRoles((current) => current.map((role) => role.id === selectedRole.id ? { ...role, version: role.version + 1, approvalStatus: 'approved', grants: [{ permission: rolePermission.trim(), scopes: [roleScope.trim()], effectiveFrom: '2026-10-01' }] } : role))
    setRoleNotice(selectedRole.name + ' updated as a complete grant-set replacement at version ' + (selectedRole.version + 1) + '.')
  }

  const retireRole = () => {
    if (!selectedRole || selectedRole.status === 'retired') return
    setRoles((current) => current.map((role) => role.id === selectedRole.id ? { ...role, status: 'retired', version: role.version + 1, grants: [], approvalStatus: 'approved' } : role))
    setRoleNotice(selectedRole.name + ' was retired non-destructively; historical revisions remain available for evidence.')
  }

  const grantEmergencyAccess = () => {
    const nextIssues: ValidationIssue[] = []
    if (!emergencyTarget.trim()) nextIssues.push({ id: 'emergency-target', category: 'field', code: 'required', message: 'Choose the application actor receiving emergency access.', targetId: 'emergency-target', targetLabel: 'Target actor' })
    if (!emergencyPermission.trim()) nextIssues.push({ id: 'emergency-permission', category: 'field', code: 'required', message: 'Enter a permission from the approved catalogue.', targetId: 'emergency-permission', targetLabel: 'Permission' })
    if (!emergencyScope.trim()) nextIssues.push({ id: 'emergency-scope', category: 'field', code: 'required', message: 'Enter an approved scope reference.', targetId: 'emergency-scope', targetLabel: 'Scope' })
    if (!emergencyReason.trim()) nextIssues.push({ id: 'emergency-reason', category: 'field', code: 'required', message: 'Enter a reason code for the emergency grant.', targetId: 'emergency-reason', targetLabel: 'Reason code' })
    if (emergencyOutcome === 'step-up-required') nextIssues.push({ id: 'emergency-step-up', category: 'authorization', code: 'step-up-required', message: 'Recent authentication assurance or an explicit step-up challenge is required.', targetId: 'emergency-reason', targetLabel: 'Authentication assurance', nextAction: 'Complete step-up and retry with the same reviewed request.' })
    if (emergencyOutcome === 'denied') nextIssues.push({ id: 'emergency-denied', category: 'authorization', code: 'authorization-denied', message: 'The requested permission or scope is outside the current policy decision.', targetId: 'emergency-scope', targetLabel: 'Approved scope', nextAction: 'Choose a permitted scope or request a new approved decision.' })
    if (emergencyOutcome === 'duplicate') nextIssues.push({ id: 'emergency-duplicate', category: 'business-rule', code: 'duplicate-submission', message: 'This command is a safe duplicate; no second grant was created.', targetId: 'emergency-reason', targetLabel: 'Command identity', nextAction: 'Reuse the established grant reference or submit an intentional new command identity.' })
    setEmergencyIssues(nextIssues)
    if (nextIssues.length) {
      setEmergencyNotice('The emergency grant was not established and no privileged access side effect was applied.')
      return
    }
    const nextVersion = emergencyGrants.length + 1
    const id = 'grant-' + String(nextVersion).padStart(3, '0')
    const start = '2026-09-27T08:00:00Z'
    const expiry = '2026-09-27T10:00:00Z'
    setEmergencyGrants((current) => [...current, { id, targetActor: emergencyTarget.trim(), status: 'active', permissions: [emergencyPermission.trim()], scopes: [emergencyScope.trim()], reasonCode: emergencyReason.trim(), approver: 'Independent approver fixture', startsAt: start, expiresAt: expiry, policyVersion: 'emergency-policy-v1', reviewStatus: 'pending', reviewDueAt: '2026-09-28T08:00:00Z', version: 1 }])
    setSelectedEmergencyGrantId(id)
    setEmergencyOutcome('ready')
    setEmergencyNotice(id + ' was established with a four-hour maximum window and pending post-use review.')
  }

  const revokeEmergencyAccess = () => {
    if (!selectedEmergencyGrant || selectedEmergencyGrant.status !== 'active') {
      setEmergencyNotice('Only active emergency grants can be revoked; expired and revoked grants remain historical evidence.')
      return
    }
    setEmergencyGrants((current) => current.map((grant) => grant.id === selectedEmergencyGrant.id ? { ...grant, status: 'revoked', version: grant.version + 1, reviewStatus: 'pending' } : grant))
    setEmergencyNotice(selectedEmergencyGrant.id + ' was revoked. All uses must now be denied and the post-use review remains pending.')
  }

  const completeEmergencyReview = () => {
    if (!selectedEmergencyGrant || selectedEmergencyGrant.status !== 'revoked' || (selectedEmergencyGrant.reviewStatus !== 'pending' && selectedEmergencyGrant.reviewStatus !== 'overdue')) return
    setEmergencyGrants((current) => current.map((grant) => grant.id === selectedEmergencyGrant.id ? { ...grant, reviewStatus: 'completed', reviewOutcome: emergencyReviewOutcome, version: grant.version + 1 } : grant))
    setEmergencyNotice(selectedEmergencyGrant.id + ' review was completed with outcome ' + emergencyReviewOutcome + '.')
  }

  return (
    <section className="space-y-6" aria-labelledby="iam-worklist-title">
      <div>
        <p className="text-sm font-semibold uppercase tracking-wide text-primary">IAM-WS-01 · EP-IAM-001</p>
        <h2 id="iam-worklist-title" className="mt-1 text-2xl font-semibold">Users and access assignments</h2>
        <p className="mt-2 max-w-4xl text-base-content/75">Manage application-owned lifecycle state and scoped assignments. Authentication subjects remain immutable references; role and policy administration stay with their owning stories.</p>
      </div>

      <Panel title="User access worklist" description="Masked identity data, current status, assignment set, and review evidence.">
        <DataTable caption="Application users" columns={columns} rows={users} getRowKey={(user) => user.id} emptyMessage="No application users match the current review." />
      </Panel>

      <Panel title="Role and permission worklist" description="IAM-SCR-02 - Versioned grant revisions, opaque scopes, effective dates, and approval state.">
        <DataTable caption="Application roles" columns={roleColumns} rows={roles} getRowKey={(role) => role.id} emptyMessage="No roles match the current review." />
      </Panel>

      <Panel title="IAM-SCR-04 · Emergency access" description="Grant, revoke, expiry, authorization use, and post-use review. Sensitive actor values remain masked in this synthetic workflow.">
        <DataTable caption="Emergency access grants" columns={emergencyColumns} rows={emergencyGrants} getRowKey={(grant) => grant.id} emptyMessage="No emergency access grants are available." />
        {selectedEmergencyGrant ? <div className="mt-5 rounded-box border border-base-300 p-4">
          <div className="grid gap-4 lg:grid-cols-4">
            <div><p className="text-sm text-base-content/70">Grant reference</p><p className="mt-1 font-mono font-semibold">{selectedEmergencyGrant.id}</p><p className="mt-1 text-sm text-base-content/70">Aggregate version {selectedEmergencyGrant.version}</p></div>
            <div><p className="text-sm text-base-content/70">Access state</p><div className="mt-2"><StatusBadge state={emergencyStatusState[selectedEmergencyGrant.status]} label={selectedEmergencyGrant.status} announce /></div><p className="mt-2 text-sm text-base-content/70">Expiry remains deny-by-default after cleanup delay.</p></div>
            <div><p className="text-sm text-base-content/70">Independent approval</p><p className="mt-1">{selectedEmergencyGrant.approver}</p><p className="mt-1 font-mono text-sm text-base-content/70">{selectedEmergencyGrant.policyVersion}</p></div>
            <div><p className="text-sm text-base-content/70">Post-use review</p><div className="mt-2"><StatusBadge state={emergencyReviewState[selectedEmergencyGrant.reviewStatus]} label={selectedEmergencyGrant.reviewStatus} announce /></div><p className="mt-1 text-sm text-base-content/70">Due {selectedEmergencyGrant.reviewDueAt}</p></div>
          </div>
          <dl className="mt-5 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <div><dt className="text-sm text-base-content/70">Target actor</dt><dd className="mt-1">{selectedEmergencyGrant.targetActor}</dd></div>
            <div><dt className="text-sm text-base-content/70">Permissions</dt><dd className="mt-1 text-sm">{selectedEmergencyGrant.permissions.join(', ')}</dd></div>
            <div><dt className="text-sm text-base-content/70">Opaque scopes</dt><dd className="mt-1 text-sm">{selectedEmergencyGrant.scopes.join(', ')}</dd></div>
            <div><dt className="text-sm text-base-content/70">Reason code</dt><dd className="mt-1 font-mono">{selectedEmergencyGrant.reasonCode}</dd></div>
            <div><dt className="text-sm text-base-content/70">Starts at</dt><dd className="mt-1"><time dateTime={selectedEmergencyGrant.startsAt}>{selectedEmergencyGrant.startsAt}</time></dd></div>
            <div><dt className="text-sm text-base-content/70">Expires at</dt><dd className="mt-1"><time dateTime={selectedEmergencyGrant.expiresAt}>{selectedEmergencyGrant.expiresAt}</time></dd></div>
            <div><dt className="text-sm text-base-content/70">Review outcome</dt><dd className="mt-1">{selectedEmergencyGrant.reviewOutcome ?? 'Pending'}</dd></div>
            <div><dt className="text-sm text-base-content/70">Grant reference propagation</dt><dd className="mt-1 text-sm">{selectedEmergencyGrant.id} is carried into authorization and audit context.</dd></div>
          </dl>
        </div> : null}
        <div className="mt-5 rounded-box border border-base-300 p-4">
          <h3 className="font-semibold">Grant emergency access</h3>
          <p className="mt-1 text-sm text-base-content/70">The fixture enforces an independent approver, recent high-risk assurance or explicit step-up, and a maximum four-hour window.</p>
          <div className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
            <Field id="emergency-target" label="Target actor" value={emergencyTarget} onChange={(event) => setEmergencyTarget(event.target.value)} />
            <Field id="emergency-permission" label="Permission" value={emergencyPermission} onChange={(event) => setEmergencyPermission(event.target.value)} />
            <Field id="emergency-scope" label="Opaque scope reference" value={emergencyScope} onChange={(event) => setEmergencyScope(event.target.value)} />
            <Field id="emergency-reason" label="Reason code" value={emergencyReason} onChange={(event) => setEmergencyReason(event.target.value)} />
          </div>
          {emergencyIssues.length ? <div className="mt-4"><ValidationSummary issues={emergencyIssues} /></div> : null}
          <div className="mt-4 flex flex-wrap gap-2">
            <Button onClick={grantEmergencyAccess}>Grant emergency access</Button>
            <Button variant="danger" onClick={revokeEmergencyAccess} disabled={!selectedEmergencyGrant || selectedEmergencyGrant.status !== 'active'}>Revoke selected grant</Button>
            <Button variant="secondary" onClick={completeEmergencyReview} disabled={!selectedEmergencyGrant || selectedEmergencyGrant.status !== 'revoked' || (selectedEmergencyGrant.reviewStatus !== 'pending' && selectedEmergencyGrant.reviewStatus !== 'overdue')}>Complete review</Button>
            <Button variant="ghost" onClick={() => setEmergencyConflictOpen(true)}>Simulate emergency version conflict</Button>
          </div>
          <div className="mt-4 flex flex-wrap gap-2" role="group" aria-label="Emergency access outcome fixtures">
            <Button size="sm" variant={emergencyOutcome === 'denied' ? 'primary' : 'ghost'} aria-pressed={emergencyOutcome === 'denied'} onClick={() => { setEmergencyOutcome('denied'); setEmergencyIssues([]) }}>Simulate denial</Button>
            <Button size="sm" variant={emergencyOutcome === 'duplicate' ? 'primary' : 'ghost'} aria-pressed={emergencyOutcome === 'duplicate'} onClick={() => { setEmergencyOutcome('duplicate'); setEmergencyIssues([]) }}>Simulate duplicate</Button>
            <Button size="sm" variant={emergencyOutcome === 'step-up-required' ? 'primary' : 'ghost'} aria-pressed={emergencyOutcome === 'step-up-required'} onClick={() => { setEmergencyOutcome('step-up-required'); setEmergencyIssues([]) }}>Require step-up</Button>
            <Button size="sm" variant="ghost" onClick={() => { setEmergencyOutcome('ready'); setEmergencyIssues([]); setEmergencyNotice('Emergency grant command fixtures reset; retry after reviewing the current grant state.') }}>Reset outcome fixture</Button>
          </div>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <Field id="emergency-review-outcome" label="Opaque review outcome code" value={emergencyReviewOutcome} onChange={(event) => setEmergencyReviewOutcome(event.target.value)} />
            <div className="rounded-box border border-base-300 p-3 text-sm text-base-content/75"><p className="font-semibold text-base-content">Review deadline</p><p className="mt-1">One business day after the emergency grant is recorded; overdue is derived deterministically and never grants access.</p></div>
          </div>
        </div>
        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{emergencyNotice}</p>
      </Panel>

      <Panel title="IAM-SCR-03 · Segregation rule administration" description="IAM-owned versioned rules, independent approval evidence, and safe maintenance outcomes. Finance action screens remain in their owning modules.">
        <DataTable caption="Segregation rules" columns={segregationColumns} rows={segregationRules} getRowKey={(rule) => rule.id} emptyMessage="No segregation rules are available." />
        {selectedSegregationRule ? <div className="mt-5 rounded-box border border-base-300 p-4">
          <div className="grid gap-4 lg:grid-cols-4">
            <div><p className="text-sm text-base-content/70">Rule</p><p className="mt-1 font-semibold">{selectedSegregationRule.name}</p><p className="mt-1 text-sm text-base-content/70">{selectedSegregationRule.code}</p></div>
            <div><p className="text-sm text-base-content/70">Enforcement</p><p className="mt-1">{selectedSegregationRule.mode}</p><p className="mt-1 text-sm text-base-content/70">Version {selectedSegregationRule.version}</p></div>
            <div><p className="text-sm text-base-content/70">Conflict permissions</p><p className="mt-1 text-sm">{selectedSegregationRule.permissions.join(" + ")}</p></div>
            <div><p className="text-sm text-base-content/70">Threshold / cooling-off</p><p className="mt-1 text-sm">{selectedSegregationRule.threshold ? selectedSegregationRule.threshold : selectedSegregationRule.coolingOffHours ? selectedSegregationRule.coolingOffHours + " hours" : "Not applicable"}</p></div>
          </div>
          <div className="mt-5 flex flex-wrap gap-2">
            <Button size="sm" variant={segregationOutcome === "conflict" ? "primary" : "ghost"} onClick={() => setSegregationOutcome("conflict")}>Review conflict explanation</Button>
            <Button size="sm" variant={segregationOutcome === "exception-required" ? "primary" : "ghost"} onClick={() => setSegregationOutcome("exception-required")}>Require approved exception</Button>
            <Button size="sm" variant={segregationOutcome === "stale" ? "primary" : "ghost"} onClick={() => setSegregationOutcome("stale")}>Simulate stale rule</Button>
            <Button size="sm" variant={segregationOutcome === "unavailable" ? "primary" : "ghost"} onClick={() => setSegregationOutcome("unavailable")}>Simulate unavailable policy</Button>
            <Button size="sm" variant="danger" onClick={retireSegregationRule} disabled={selectedSegregationRule.status === "retired"}>Retire rule revision</Button>
          </div>
          {(() => {
            const explanation = segregationDecisionFixtures[segregationOutcome]
            return <div className="mt-5 rounded-box border border-base-300 p-4">
              <div className="flex flex-wrap items-center gap-3"><StatusBadge state={segregationDecisionState[explanation.outcome]} label={explanation.outcome} announce /><span className="text-sm text-base-content/75">Decision reference: {explanation.decisionReference}</span></div>
              <dl className="mt-4 grid gap-4 md:grid-cols-3"><div><dt className="text-sm text-base-content/70">Non-sensitive reason</dt><dd className="mt-1">{explanation.reason}</dd></div><div><dt className="text-sm text-base-content/70">Rule / policy version</dt><dd className="mt-1 font-mono">{explanation.ruleVersion}</dd></div><div><dt className="text-sm text-base-content/70">Permitted resolution</dt><dd className="mt-1">{explanation.resolution}</dd></div></dl>
            </div>
          })()}
          <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{segregationNotice}</p>
        </div> : null}
      </Panel>

      <Panel title="IAM-SCR-05 · Access decision explanation" description="Synthetic policy evaluation evidence. This explanation never exposes policy rules, token claims, or sensitive values.">
        <div className="flex flex-wrap gap-2" role="group" aria-label="Synthetic authorization outcomes">
          {(Object.keys(accessDecisionFixtures) as AccessDecisionOutcome[]).map((outcome) => <Button key={outcome} size="sm" variant={decisionOutcome === outcome ? 'primary' : 'ghost'} aria-pressed={decisionOutcome === outcome} onClick={() => setDecisionOutcome(outcome)}>{outcome}</Button>)}
        </div>
        {(() => {
          const explanation = accessDecisionFixtures[decisionOutcome]
          return <>
            <div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={accessDecisionState[explanation.outcome]} label={explanation.outcome} announce /><span className="text-sm text-base-content/75">Record access: {explanation.recordAccess ? 'permitted' : 'not permitted'}</span></div>
            <dl className="mt-4 grid gap-4 md:grid-cols-2 lg:grid-cols-4">
              <div><dt className="text-sm text-base-content/70">Policy version</dt><dd className="mt-1 font-mono">{explanation.policyVersion}</dd></div>
              <div><dt className="text-sm text-base-content/70">Decision reference</dt><dd className="mt-1 font-mono">{explanation.decisionReference}</dd></div>
              <div><dt className="text-sm text-base-content/70">Matched dimension category</dt><dd className="mt-1">{explanation.matchedDimension}</dd></div>
              <div><dt className="text-sm text-base-content/70">Next action</dt><dd className="mt-1">{explanation.nextAction}</dd></div>
            </dl>
            <div className="mt-5 grid gap-4 lg:grid-cols-2">
              <div className="rounded-box border border-base-300 p-4"><h3 className="font-semibold">Record decision</h3><p className="mt-2 text-sm">The record decision and each field decision are evaluated independently.</p><p className="mt-3 text-sm">Policy outcome: {explanation.outcome}</p></div>
              <SensitiveDataGuard label="Restricted account detail" classification="Financial-sensitive" value="account-detail-fixture" access={explanation.recordAccess ? 'authorized' : 'restricted'} canReveal={false} canExport={false} actorReference="actor-fixture" targetReference="record-fixture-42" scopeReference="entity-vietnam" purpose="review-field-access" decisionReference={explanation.decisionReference} policyVersion={explanation.policyVersion} onAccess={recordSensitiveAccess} onAccessDenied={() => setNotice('Field reveal and export remain unavailable; record access does not grant field access.')} />
            </div>
          </>
        })()}
      </Panel>

      {selectedRole ? <Panel title="IAM-SCR-02 - Role detail" description="A role revision is a complete grant-set replacement. Retirement is non-destructive and preserves evidence.">
        <div className="grid gap-4 lg:grid-cols-4">
          <div><p className="text-sm text-base-content/70">Role</p><p className="mt-1 font-semibold">{selectedRole.name}</p><p className="mt-1 text-sm text-base-content/70">Aggregate {selectedRole.id} - version {selectedRole.version}</p></div>
          <div><p className="text-sm text-base-content/70">Status</p><div className="mt-2"><StatusBadge state={roleStatusState[selectedRole.status]} label={selectedRole.status} announce /></div></div>
          <div><p className="text-sm text-base-content/70">Approval evidence</p><div className="mt-2"><StatusBadge state={selectedRole.approvalStatus === 'approved' ? 'success' : selectedRole.approvalStatus === 'pending' ? 'pending' : 'error'} label={selectedRole.approvalStatus} /></div></div>
          <div><p className="text-sm text-base-content/70">Revision policy</p><p className="mt-1 text-sm">Expected version + idempotency key</p></div>
        </div>
        <div className="mt-5 overflow-x-auto rounded-box border border-base-300">
          <table className="table"><caption className="sr-only">Permission grants for {selectedRole.name}</caption><thead><tr><th scope="col">Permission</th><th scope="col">Scopes</th><th scope="col">Effective from</th><th scope="col">Effective to</th></tr></thead><tbody>{selectedRole.grants.length ? selectedRole.grants.map((grant) => <tr key={grant.permission + grant.effectiveFrom}><th scope="row">{grant.permission}</th><td>{grant.scopes.join(', ')}</td><td><time dateTime={grant.effectiveFrom}>{grant.effectiveFrom}</time></td><td>{grant.effectiveTo ? <time dateTime={grant.effectiveTo}>{grant.effectiveTo}</time> : 'Open ended'}</td></tr>) : <tr><td colSpan={4}>No active grants; the retired role remains queryable for historical evidence.</td></tr>}</tbody></table>
        </div>
        <div className="mt-5 rounded-box border border-base-300 p-4">
          <h3 className="font-semibold">Replace complete grant set</h3>
          <p className="mt-1 text-sm text-base-content/70">Permission identifiers are catalogue-validated; scope references stay opaque and are checked against the administering actor scope.</p>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <Field id="role-permission" label="Permission" value={rolePermission} onChange={(event) => setRolePermission(event.target.value)} />
            <Field id="role-scope" label="Opaque scope reference" value={roleScope} onChange={(event) => setRoleScope(event.target.value)} />
          </div>
          {roleIssues.length ? <div className="mt-4"><ValidationSummary issues={roleIssues} /></div> : null}
          <div className="mt-4 flex flex-wrap gap-2">
            <Button onClick={replaceRoleGrant} disabled={selectedRole.status === 'retired'}>Replace grant set</Button>
            <Button variant="danger" onClick={retireRole} disabled={selectedRole.status === 'retired'}>Retire role</Button>
            <Button variant="ghost" onClick={() => setRoleConflictOpen(true)}>Simulate role version conflict</Button>
            <Button variant="ghost" onClick={() => { setRolePermission('duplicate'); setRoleNotice('Enter duplicate to review the safe duplicate result.') }}>Review duplicate grant</Button>
            <Button variant="ghost" onClick={() => { setRoleScope('outside-approved-scope'); setRoleNotice('Enter the outside-scope fixture to review authorization denial.') }}>Review scope denial</Button>
          </div>
        </div>
        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{roleNotice}</p>
      </Panel> : null}

      {selected ? <Panel title="IAM-SCR-01 · User access record" description="Review authoritative state before a lifecycle or assignment mutation.">
        <div className="grid gap-4 lg:grid-cols-3">
          <div>
            <p className="text-sm text-base-content/70">User</p>
            <p className="mt-1 font-semibold">{selected.label}</p>
            <p className="mt-1 text-sm text-base-content/70">Aggregate {selected.id} · version {selected.version}</p>
          </div>
          <div>
            <p className="text-sm text-base-content/70">Lifecycle status</p>
            <div className="mt-2"><StatusBadge state={statusState[selected.status]} label={selected.status} announce /></div>
          </div>
          <div>
            <p className="text-sm text-base-content/70">Review evidence</p>
            <p className="mt-1">{selected.reviewEvidence}</p>
          </div>
        </div>

        <div className="mt-5 grid gap-4 lg:grid-cols-2">
          <SensitiveDataGuard label="Authentication subject reference" classification="Identity-sensitive" value={selected.subjectReference} access="restricted" canReveal={false} canExport={false} actorReference="actor-fixture" targetReference={selected.id} scopeReference="identity-administration" purpose="review-user-access" decisionReference="decision-user-access-001" policyVersion="iam-ui-fixture-v7" onAccess={recordSensitiveAccess} onAccessDenied={() => setNotice('Authentication subject data remains masked for privacy.')} />
          <div className="rounded-box border border-base-300 p-4">
            <h3 className="font-semibold">Current assignment set</h3>
            <p className="mt-2 text-sm">Roles: {selected.roles.length ? selected.roles.join(', ') : 'No assignments'}</p>
            <p className="mt-2 text-sm">Scopes: {selected.scopes.length ? selected.scopes.join(', ') : 'No approved scope'}</p>
            <p className="mt-3 text-sm text-base-content/70">Replacement is atomic: the submitted set becomes the complete authoritative set.</p>
          </div>
        </div>

        <div className="mt-5 flex flex-wrap gap-2">
          {selected.status !== 'active' && selected.status !== 'terminated' ? <Button onClick={() => runLifecycleAction('active')}>Activate</Button> : null}
          {selected.status === 'active' ? <Button variant="secondary" onClick={() => runLifecycleAction('suspended')}>Suspend</Button> : null}
          {selected.status !== 'terminated' ? <Button variant="danger" onClick={() => runLifecycleAction('terminated')}>Terminate</Button> : null}
          <Button variant="ghost" onClick={() => setConflictOpen(true)}>Simulate version conflict</Button>
        </div>

        <div className="mt-5 rounded-box border border-base-300 p-4">
          <h3 className="font-semibold">Replace complete assignments</h3>
          <p className="mt-1 text-sm text-base-content/70">Use approved role and entity-scope identities. This fixture makes denial and duplicate states reviewable without exposing raw identity data.</p>
          <div className="mt-4 grid gap-4 md:grid-cols-2">
            <Field id="role-assignment" label="Role identity" value={role} onChange={(event) => setRole(event.target.value)} />
            <Field id="entity-scope" label="Entity scope" value={scope} onChange={(event) => setScope(event.target.value)} />
          </div>
          {issues.length ? <div className="mt-4"><ValidationSummary issues={issues} /></div> : null}
          <div className="mt-4 flex flex-wrap gap-2">
            <Button onClick={replaceAssignments} disabled={selected.status === 'terminated'}>Replace assignments</Button>
            <Button variant="ghost" onClick={() => { setRole('duplicate'); setNotice('Enter duplicate to review the safe duplicate result.') }}>Review duplicate result</Button>
            <Button variant="ghost" onClick={() => { setScope('outside-approved-scope'); setNotice('Enter the outside-scope fixture to review authorization denial.') }}>Review denial result</Button>
          </div>
        </div>

        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
      </Panel> : null}

      {conflictOpen ? <VersionConflictDialog conflict={conflictFixture} onClose={() => setConflictOpen(false)} onRefresh={() => { setConflictOpen(false); setNotice('Authoritative user state refreshed. Review the current version before retrying.') }} onRetry={() => { setConflictOpen(false); setNotice('Retry remains blocked until the refreshed assignment set is reviewed.') }} /> : null}
      {roleConflictOpen ? <VersionConflictDialog conflict={{ ...conflictFixture, attemptedAction: 'Replace complete permission grant set', record: { label: selectedRole?.name ?? 'Role', href: '/administration/identity-access' }, expected: { version: selectedRole?.version ?? 1, state: selectedRole?.status ?? 'active' }, current: { version: (selectedRole?.version ?? 1) + 1, state: selectedRole?.status ?? 'active' }, changedValues: [{ id: 'grants', label: 'Permission grants', reviewedValue: 'finance.gl.submit.posting.request', currentValue: 'finance.gl.apply.journal.approval.decision' }] }} onClose={() => setRoleConflictOpen(false)} onRefresh={() => { setRoleConflictOpen(false); setRoleNotice('Authoritative role state refreshed. Review the current version before retrying.') }} onRetry={() => { setRoleConflictOpen(false); setRoleNotice('Retry remains blocked until the refreshed role revision is reviewed.') }} /> : null}
      {emergencyConflictOpen ? <VersionConflictDialog conflict={{ ...conflictFixture, attemptedAction: 'Revoke emergency access grant', record: { label: selectedEmergencyGrant?.id ?? 'Emergency grant', href: '/identity-access/iam-scr-04' }, expected: { version: selectedEmergencyGrant?.version ?? 1, state: selectedEmergencyGrant?.status ?? 'active' }, current: { version: (selectedEmergencyGrant?.version ?? 1) + 1, state: selectedEmergencyGrant?.status ?? 'active' }, changedValues: [{ id: 'lifecycle', label: 'Grant lifecycle state', reviewedValue: 'active', currentValue: selectedEmergencyGrant?.status ?? 'revoked' }, { id: 'review', label: 'Post-use review', reviewedValue: 'pending', currentValue: selectedEmergencyGrant?.reviewStatus ?? 'completed' }] }} onClose={() => setEmergencyConflictOpen(false)} onRefresh={() => { setEmergencyConflictOpen(false); setEmergencyNotice('Authoritative emergency grant state refreshed. Review the current version before retrying.') }} onRetry={() => { setEmergencyConflictOpen(false); setEmergencyNotice('Retry remains blocked until the refreshed grant and review state are confirmed.') }} /> : null}
    </section>
  )
}
