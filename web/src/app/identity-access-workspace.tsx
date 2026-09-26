import { useMemo, useState } from 'react'

import { SensitiveDataGuard } from '@/components/sensitive-data-guard'
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

const initialUsers: ManagedUser[] = [
  { id: 'user-001', label: 'Finance operator 01', subjectReference: 'subject-ref-01', status: 'active', roles: ['finance.viewer', 'close.reviewer'], scopes: ['entity-vietnam', 'entity-singapore'], version: 8, reviewEvidence: 'Policy review 2026-09-20' },
  { id: 'user-002', label: 'AP operator 02', subjectReference: 'subject-ref-02', status: 'suspended', roles: ['payables.operator'], scopes: ['entity-vietnam'], version: 5, reviewEvidence: 'Suspended by access review' },
  { id: 'user-003', label: 'New application user', subjectReference: 'subject-ref-03', status: 'inactive', roles: [], scopes: [], version: 1, reviewEvidence: 'Awaiting activation' },
]

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

  const selected = users.find((user) => user.id === selectedId) ?? users[0]
  const columns = useMemo(() => [
    { key: 'user', header: 'User', rowHeader: true, render: (user: ManagedUser) => <button type="button" className="link link-primary text-left font-semibold" onClick={() => setSelectedId(user.id)}>{user.label}</button> },
    { key: 'status', header: 'Status', render: (user: ManagedUser) => <StatusBadge state={statusState[user.status]} label={user.status} /> },
    { key: 'roles', header: 'Roles', render: (user: ManagedUser) => user.roles.length ? user.roles.join(', ') : 'No assignments' },
    { key: 'scopes', header: 'Scopes', render: (user: ManagedUser) => user.scopes.length ? user.scopes.join(', ') : 'No approved scope' },
    { key: 'version', header: 'Version', render: (user: ManagedUser) => <>v{user.version}</> },
    { key: 'review', header: 'Review evidence', render: (user: ManagedUser) => user.reviewEvidence },
    { key: 'action', header: 'Action', render: (user: ManagedUser) => <Button size="sm" variant="secondary" onClick={() => setSelectedId(user.id)}>Review</Button> },
  ] as const, [])

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
          <SensitiveDataGuard label="Authentication subject reference" classification="Identity-sensitive" value={selected.subjectReference} access="restricted" canReveal={false} canExport={false} onAccessDenied={() => setNotice('Authentication subject data remains masked for privacy.')} />
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
    </section>
  )
}
