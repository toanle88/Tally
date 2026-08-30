import type { EvidenceLink } from '@/components/record-context'
import type { ActionState, ExceptionSnapshot, OperationalLabel, OperationalOwner, ProcessState, ResultLookupOutcome, SavedWorklistView, SettlementSnapshot, WorklistItem } from '@/components/operational-context'

const vietnamScope = { id: 'scope-vietnam-statutory', label: 'Acme Vietnam · Statutory Book · VND · 2026-08' }
const singaporeScope = { id: 'scope-singapore-management', label: 'Acme Singapore · Management Book · SGD · 2026-08' }
const cashOwner: OperationalOwner = { id: 'cash-management', label: 'Payments & Cash Management', reference: 'PCM-FIXTURE' }
const arOwner: OperationalOwner = { id: 'accounts-receivable', label: 'Accounts Receivable', reference: 'AR-FIXTURE' }

const labels = {
  pending: { value: 'pending', label: 'Pending', semanticState: 'pending' },
  exception: { value: 'exception', label: 'Exception', semanticState: 'warning' },
  reconciled: { value: 'reconciled', label: 'Reconciled', semanticState: 'reconciled' },
  rejected: { value: 'rejected', label: 'Rejected', semanticState: 'error' },
  approvalPending: { value: 'approval-pending', label: 'Approval pending', semanticState: 'pending' },
  approved: { value: 'approved', label: 'Approved', semanticState: 'success' },
} as const satisfies Record<string, OperationalLabel>

export const operationalWorklistFixture: readonly WorklistItem[] = [
  { id: 'work-001', authoritativeCapability: 'Accounts Receivable', record: { label: 'Receipt RCPT-FIX-00042', href: '#record-detail-context-example' }, scope: vietnamScope, state: labels.pending, owner: arOwner, date: '2026-08-30', age: '2h 15m', amount: { amount: '1000.00', currency: 'USD' }, approval: labels.approvalPending, nextAction: 'Review owner application', bulkEligible: true },
  { id: 'work-002', authoritativeCapability: 'Payments & Cash Management', record: { label: 'Settlement SET-FIX-00017', href: '#settlement-example' }, scope: vietnamScope, state: labels.exception, owner: cashOwner, date: '2026-08-29', age: '1d 4h', amount: { amount: '24500000.00', currency: 'VND' }, exception: labels.exception, approval: labels.approved, nextAction: 'Resolve owner application exception', bulkEligible: false },
  { id: 'work-003', authoritativeCapability: 'Bank Feeds & Reconciliation', record: { label: 'Statement STMT-FIX-00008', href: '#process-example' }, scope: singaporeScope, state: labels.reconciled, owner: { id: 'bank-reconciliation', label: 'Bank Feeds & Reconciliation' }, date: '2026-08-28', age: '2d 6h', amount: { amount: '4500.00', currency: 'SGD' }, approval: labels.approved, nextAction: 'Open reconciled detail', bulkEligible: true },
  { id: 'work-004', authoritativeCapability: 'Accounts Receivable', record: { label: 'Receipt RCPT-FIX-00043', href: '#record-detail-context-example' }, scope: singaporeScope, state: labels.rejected, owner: arOwner, date: '2026-08-27', age: '3d 1h', amount: { amount: '200.00', currency: 'SGD' }, exception: { value: 'domain-rejection', label: 'Domain rejection', semanticState: 'error' }, nextAction: 'Correct source evidence', bulkEligible: false },
]

export const operationalSavedViews: readonly SavedWorklistView[] = [
  { id: 'my-pending', label: 'My pending work', filters: { state: 'pending', ownerId: 'accounts-receivable' }, visibleColumns: ['record', 'state', 'owner', 'age', 'amount', 'nextAction'] },
  { id: 'exceptions', label: 'Open exceptions', filters: { exception: 'exception' }, visibleColumns: ['record', 'scope', 'state', 'owner', 'age', 'exception', 'nextAction'] },
]

export const operationalActions: readonly ActionState[] = [
  { id: 'review', label: 'Review result', permitted: true, material: false },
  { id: 'apply', label: 'Apply settlement', permitted: false, material: true, blockingReason: 'Owner acknowledgement is required before application.', safeNextAction: 'Review the owner acknowledgement exception.' },
]

export const settlementFixture: SettlementSnapshot = {
  gross: { amount: '1000.00', currency: 'USD' },
  returned: { amount: '100.00', currency: 'USD' },
  reversed: { amount: '25.00', currency: 'USD' },
  cancelled: { amount: '0.00', currency: 'USD' },
  remaining: { amount: '125.00', currency: 'USD' },
  net: { amount: '875.00', currency: 'USD' },
  ownerAcknowledgement: { value: 'awaiting-owner', label: 'Awaiting owner acknowledgement', semanticState: 'pending' },
  reconciliation: { value: 'exception', label: 'Reconciliation exception', semanticState: 'warning' },
  exception: { value: 'owner-application', label: 'Owner application exception', semanticState: 'warning', detail: 'Cash remains visible in clearing until an authorized resolution is established.' },
}

const exceptionEvidence: readonly EvidenceLink[] = [
  { id: 'exception-source', kind: 'source', label: 'Bank allocation evidence', reference: 'BANK-FIX-00017', access: 'available', href: '#source' },
  { id: 'exception-restricted', kind: 'provider-authority', label: 'Provider evidence', reference: 'PROVIDER-FIX-00017', access: 'restricted', restrictionReason: 'Provider detail requires explicit access.' },
]

export const exceptionFixture: ExceptionSnapshot = {
  id: 'exception-00017',
  type: 'Incoming settlement owner application exception',
  scope: vietnamScope,
  amount: { amount: '100.00', currency: 'USD' },
  owner: cashOwner,
  evidence: exceptionEvidence,
  age: '1d 4h',
  authorizedResolutions: [
    { id: 'corrected-application', label: 'Apply corrected owner reference', permitted: true, material: true },
    { id: 'reclassify', label: 'Reclassify clearing', permitted: false, material: true, blockingReason: 'Independent approval is not recorded.', safeNextAction: 'Request the required approval.' },
  ],
  resultingState: { value: 'awaiting-owner', label: 'Awaiting owner acknowledgement', semanticState: 'pending' },
}

export const resultLookupFixtures: readonly ResultLookupOutcome[] = [
  { kind: 'safe-duplicate', query: { businessIdentity: 'RCPT-FIX-00042', requestFingerprint: 'fp-receipt-00042' }, status: { value: 'established', label: 'Established result found', semanticState: 'success' }, establishedResult: { label: 'Open established receipt result', href: '#record-detail-context-example' }, nextAction: 'Open the established result; do not resubmit.' },
  { kind: 'ambiguous-outcome', query: { businessIdentity: 'SET-FIX-00017', requestFingerprint: 'fp-settlement-00017' }, status: { value: 'ambiguous', label: 'Ambiguous outcome', semanticState: 'warning' }, nextAction: 'Refresh status lookup and wait for authoritative reconciliation before retrying.' },
  { kind: 'identity-content-conflict', query: { businessIdentity: 'RCPT-FIX-00042', requestFingerprint: 'fp-changed-content' }, status: { value: 'conflict', label: 'Identity-content conflict', semanticState: 'error' }, establishedFingerprint: 'fp-receipt-00042', establishedResult: { label: 'Review established identity', href: '#record-detail-context-example' }, requiredNewBusinessIdentity: { label: 'Create new identity RCPT-FIX-00044', href: '#new-business-identity' }, nextAction: 'Create a new business identity; do not reuse the existing one.' },
]

export const processFixture: ProcessState = {
  id: 'process-00042',
  label: 'Customer receipt settlement process',
  age: '2h 15m',
  currentState: { value: 'partial', label: 'Partially completed', semanticState: 'warning' },
  steps: [
    { id: 'process-source', label: 'Source receipt recorded', owner: arOwner, status: 'terminal', detail: 'The authoritative receipt record is established.', authoritativeRecord: { label: 'Receipt detail', href: '#record-detail-context-example' } },
    { id: 'process-posting', label: 'Cash posting', owner: cashOwner, status: 'reconciled', detail: 'The cash posting result is reconciled.', authoritativeRecord: { label: 'Settlement detail', href: '#settlement-example' } },
    { id: 'process-owner', label: 'Owner application', owner: arOwner, status: 'current', detail: 'Current owner action is required before the process can complete.' },
    { id: 'process-exception', label: 'Exception resolution', owner: cashOwner, status: 'pending', detail: 'Resolution remains pending with the owning capability.' },
    { id: 'process-close', label: 'Final reconciliation', owner: { id: 'bank-reconciliation', label: 'Bank Feeds & Reconciliation' }, status: 'failed', detail: 'Downstream reconciliation cannot complete while the exception remains open.' },
  ],
  exception: { value: 'open-exception', label: 'Open managed exception', semanticState: 'warning', detail: 'The process is not terminal.' },
}
