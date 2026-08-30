import type { SemanticState } from '@/components/ui'

export type ScopeReference = { id: string; label: string }

export type RecordSensitivity = 'standard' | 'restricted' | 'highly-restricted'

export interface RecordIdentity {
  recordId: string
  recordType: string
  capability: string
  scope: ScopeReference
  lifecycleState: string
  lifecycleSemanticState: SemanticState
  version: number
  source: { label: string; reference: string }
  owner: { label: string; reference?: string }
  sensitivity: { label: string; classification: RecordSensitivity }
  lastMaterialChange: string
}

export interface LineageReference {
  kind: CorrectionKind
  recordId: string
  label: string
  href: string
  state?: string
}

export type CorrectionKind =
  | 'reversal'
  | 'amendment'
  | 'return'
  | 'unapplication'
  | 'replacement'
  | 'compensation'
  | 'supersession'

export interface LifecycleTransition {
  id: string
  state: string
  semanticState: SemanticState
  occurredAt: string
  actor: string
  decision?: string
  sequence: number
  references: readonly LineageReference[]
}

export type MoneyRole = 'transaction' | 'functional' | 'presentation'

export interface RateEvidence {
  rateSet: string
  rateType: string
  asOf: string
  sourceReference: string
  rateValue?: string
}

export interface MoneyAmount {
  role: MoneyRole
  amount: string
  currency: string
  roundingLabel: string
  rateEvidence?: RateEvidence
}

export type MoneyByRole = {
  [Role in MoneyRole]?: MoneyAmount & { role: Role }
}

export interface SignedGainLoss {
  amount: string
  currency: string
  signConvention: string
}

export type EvidenceKind =
  | 'source'
  | 'approval'
  | 'posting'
  | 'provider-authority'
  | 'reconciliation'
  | 'close'
  | 'statement'
  | 'audit'

export type EvidenceAccess = 'available' | 'restricted' | 'unavailable'

export interface EvidenceLink {
  id: string
  kind: EvidenceKind
  label: string
  reference: string
  access: EvidenceAccess
  href?: string
  restrictionReason?: string
}

export type SensitiveAction = 'reveal' | 'export'
export type SensitiveAccess = 'authorized' | 'restricted'
export type LegalHoldStatus = 'active' | 'released' | 'none'
