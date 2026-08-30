import type {
  EvidenceLink,
  LifecycleTransition,
  LineageReference,
  MoneyByRole,
  RecordIdentity,
} from '@/components/record-context'

export const recordIdentityFixture: RecordIdentity = {
  recordId: 'RCPT-FIX-00042',
  recordType: 'Customer receipt',
  capability: 'Accounts Receivable',
  scope: { id: 'scope-vietnam-statutory', label: 'Acme Vietnam Co., Ltd. · Statutory Book · VND · 2026-08' },
  lifecycleState: 'Established',
  lifecycleSemanticState: 'success',
  version: 3,
  source: { label: 'Bank reconciliation fixture', reference: 'SRC-FIX-00042' },
  owner: { label: 'Accounts Receivable', reference: 'AR-FIXTURE' },
  sensitivity: { label: 'Standard business record', classification: 'standard' },
  lastMaterialChange: '2026-08-30T09:30:00Z',
}

export const lineageReferences: readonly LineageReference[] = [
  { kind: 'reversal', recordId: 'REV-FIX-00042', label: 'Receipt reversal fixture', href: '/development/examples#reversal', state: 'Established' },
  { kind: 'amendment', recordId: 'AMD-FIX-00042', label: 'Receipt amendment fixture', href: '/development/examples#amendment', state: 'Pending' },
  { kind: 'return', recordId: 'RET-FIX-00042', label: 'Receipt return fixture', href: '/development/examples#return', state: 'Observed' },
  { kind: 'unapplication', recordId: 'UNA-FIX-00042', label: 'Receipt unapplication fixture', href: '/development/examples#unapplication', state: 'Established' },
  { kind: 'replacement', recordId: 'RPL-FIX-00042', label: 'Replacement receipt fixture', href: '/development/examples#replacement', state: 'Established' },
  { kind: 'compensation', recordId: 'CMP-FIX-00042', label: 'Compensation fixture', href: '/development/examples#compensation', state: 'Established' },
  { kind: 'supersession', recordId: 'SUP-FIX-00042', label: 'Superseding receipt fixture', href: '/development/examples#supersession', state: 'Established' },
]
export const lifecycleFixture: readonly LifecycleTransition[] = [
  { id: 'life-3', state: 'Established', semanticState: 'success', occurredAt: '2026-08-30T09:30:00Z', actor: 'AR fixture process', decision: 'Authoritative receipt established', sequence: 3, references: lineageReferences.slice(0, 1) },
  { id: 'life-1', state: 'Draft', semanticState: 'info', occurredAt: '2026-08-30T09:00:00Z', actor: 'Fixture user', decision: 'Source observation recorded', sequence: 1, references: [] },
  { id: 'life-2', state: 'Validated', semanticState: 'pending', occurredAt: '2026-08-30T09:15:00Z', actor: 'AR validation fixture', decision: 'Receipt evidence validated', sequence: 2, references: [] },
]

export const moneyFixture: MoneyByRole = {
  transaction: { role: 'transaction', amount: '1000.00', currency: 'USD', roundingLabel: 'Source precision retained', rateEvidence: { rateSet: 'FX-FIX-2026-08', rateType: 'Spot', asOf: '2026-08-30T09:20:00Z', sourceReference: 'RATE-FIX-0008', rateValue: '24500.000000' } },
  functional: { role: 'functional', amount: '24500000.00', currency: 'VND', roundingLabel: 'Rounded to 2 decimal places by source fixture', rateEvidence: { rateSet: 'FX-FIX-2026-08', rateType: 'Spot', asOf: '2026-08-30T09:20:00Z', sourceReference: 'RATE-FIX-0008' } },
  presentation: { role: 'presentation', amount: '1000.00', currency: 'USD', roundingLabel: 'Presentation rounding only' },
}

export const evidenceFixture: readonly EvidenceLink[] = [
  { id: 'evidence-source', kind: 'source', label: 'Source receipt observation', reference: 'SRC-FIX-00042', access: 'available', href: '#source' },
  { id: 'evidence-approval', kind: 'approval', label: 'Approval decision', reference: 'APR-FIX-00042', access: 'available', href: '#approval' },
  { id: 'evidence-posting', kind: 'posting', label: 'Posting result', reference: 'POST-FIX-00042', access: 'available', href: '#posting' },
  { id: 'evidence-provider', kind: 'provider-authority', label: 'Provider reference', reference: 'PROVIDER-FIX-00042', access: 'restricted', restrictionReason: 'Provider detail requires explicit sensitive-data access.' },
  { id: 'evidence-reconciliation', kind: 'reconciliation', label: 'Reconciliation evidence', reference: 'REC-FIX-00042', access: 'unavailable' },
  { id: 'evidence-close', kind: 'close', label: 'Close evidence', reference: 'CLOSE-FIX-00042', access: 'available', href: '#close' },
  { id: 'evidence-statement', kind: 'statement', label: 'Statement evidence', reference: 'STMT-FIX-00042', access: 'unavailable' },
  { id: 'evidence-audit', kind: 'audit', label: 'Audit evidence', reference: 'AUD-FIX-00042', access: 'available', href: '#audit' },
]
