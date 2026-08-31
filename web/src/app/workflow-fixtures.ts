import type { ApiProblem } from '@/components/operational-context'
import type { ApprovalSnapshot, FixtureSubmissionOutcome, PostingSnapshot, ProblemFixtureKind, VersionConflictSnapshot } from '@/components/workflow-context'

export const workflowScope = { id: 'scope-acme-th', label: 'Acme Thailand Co., Ltd.' }

export const approvalFixture: ApprovalSnapshot = {
  requestId: 'APR-2026-0042', subject: { label: 'Supplier settlement adjustment', reference: 'SET-1042', version: 7, snapshotLabel: 'Reviewed snapshot' }, scope: workflowScope, policy: { label: 'Settlement adjustment policy', version: '3.2' }, submittedBy: 'Narin K. · Finance Operations', requestedAt: '2026-08-31T09:15:00+07:00', status: 'decided', decision: { id: 'DEC-778', outcome: 'Approved', decidedAt: '2026-08-31T09:32:00+07:00', decisionVersion: 7 }, steps: [{ id: 'review', label: 'Finance review', owner: { id: 'fin-ops', label: 'Finance Operations' }, status: 'applied', decision: 'Approved' }, { id: 'controller', label: 'Controller approval', owner: { id: 'controller', label: 'Regional Controller' }, status: 'decided', decision: 'Approved' }], application: { status: 'awaiting-revalidation', requiredSubjectVersion: 7, reason: 'Decision exists but is not applied until the current record is revalidated.' }, delegation: 'No delegation on this request.', escalation: 'No escalation required.'
}

export const postingFixture: PostingSnapshot = {
  postingId: 'PST-2026-0191', sourceRecord: { label: 'Supplier settlement adjustment', href: '#record' }, scope: workflowScope, sourceVersion: 7, purpose: 'Establish approved adjustment in the general ledger', accountingOwner: 'General Ledger', status: 'failure', requestReference: 'REQ-POST-0191', periodGateEvidence: { fiscalPeriodId: '2026-08', periodStateVersion: 12, postingGateVersion: 4, admission: 'Admitted before provider timeout' }, failure: { detail: 'The posting provider did not return a definitive outcome.', retryPermitted: true, nextAction: 'Look up the request reference before deliberately retrying.' }, reconciliation: { state: 'Pending outcome lookup' }
}

export const conflictFixture: VersionConflictSnapshot = {
  kind: 'version', attemptedAction: 'Apply approved settlement adjustment', record: { label: 'Supplier settlement adjustment', href: '#record' }, scope: workflowScope, expected: { version: 7, state: 'Approved' }, current: { version: 8, state: 'Revalidation required' }, changedValues: [{ id: 'amount', label: 'Amount', reviewedValue: '1,250.00', currentValue: '1,275.00' }, { id: 'owner', label: 'Current owner', reviewedValue: 'Finance Operations', currentValue: 'Regional Controller' }], currentOwner: { id: 'controller', label: 'Regional Controller', reference: 'owner-22' }, processEpoch: 'settlement-epoch-2026-08-31-03', safeRetry: { permitted: true, reason: 'Retry is permitted only after refreshing and reviewing the current version.' }
}

export const identityContentConflictFixture: VersionConflictSnapshot = {
  kind: 'identity-content', attemptedAction: 'Submit settlement adjustment', record: { label: 'Supplier settlement adjustment', href: '#record' }, scope: workflowScope, expected: { version: 7, state: 'Draft' }, current: { version: 7, state: 'Draft' }, changedValues: [{ id: 'description', label: 'Description', reviewedValue: 'Apply approved adjustment', currentValue: 'Apply replacement adjustment' }], currentOwner: { id: 'fin-ops', label: 'Finance Operations' }, processEpoch: 'settlement-epoch-2026-08-31-04', safeRetry: { permitted: false, reason: 'The same business identity cannot be reused with different content.' }, requiredNewBusinessIdentity: { label: 'Create a new adjustment identity', href: '#new-business-identity' }
}

const problem = (problemKind: ProblemFixtureKind, code: string, title: string, status: number, detail: string): FixtureSubmissionOutcome => ({ kind: 'problem', problemKind, problem: { type: `https://tally.example/problems/${problemKind}`, title, status, code, detail, correlationId: `corr-us5-${problemKind}` } satisfies ApiProblem })

export const workflowOutcomeFixtures: Record<'success' | ProblemFixtureKind, FixtureSubmissionOutcome> = {
  success: { kind: 'success', result: { label: 'Adjustment established', state: 'Approved', correlationId: 'corr-us5-success' } },
  'domain-rejection': problem('domain-rejection', 'DOMAIN_REJECTION', 'Domain rejection', 422, 'The requested transition is not valid for the current state.'),
  'authorization-denial': problem('authorization-denial', 'AUTHORIZATION_DENIED', 'Authorization denied', 403, 'The current actor is not authorized for this material action.'),
  'version-conflict': { kind: 'problem', problemKind: 'version-conflict', problem: { type: 'https://tally.example/problems/version-conflict', title: 'Version conflict', status: 409, code: 'VERSION_CONFLICT', detail: 'The record changed before the material action was applied.', correlationId: 'corr-us5-409', currentVersion: 8 }, conflict: conflictFixture },
  'idempotency-conflict': { kind: 'problem', problemKind: 'idempotency-conflict', problem: { type: 'https://tally.example/problems/idempotency-conflict', title: 'Identity-content conflict', status: 409, code: 'IDEMPOTENCY_CONFLICT', detail: 'The business identity was already used with different content.', correlationId: 'corr-us5-idempotency-conflict' }, conflict: identityContentConflictFixture },
  'dependency-unavailable': problem('dependency-unavailable', 'DEPENDENCY_UNAVAILABLE', 'Dependency unavailable', 503, 'The owning dependency is unavailable.'),
  'ambiguous-outcome': problem('ambiguous-outcome', 'AMBIGUOUS_OUTCOME', 'Ambiguous outcome', 202, 'The request outcome must be looked up before any repeat.'),
  'unexpected-failure': problem('unexpected-failure', 'UNEXPECTED_FAILURE', 'Unexpected failure', 500, 'The owning capability returned an unexpected failure.'),
}

export function submitWorkflowFixture(): FixtureSubmissionOutcome {
  return workflowOutcomeFixtures['version-conflict']
}
