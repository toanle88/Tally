import type { EvidenceLink, ScopeReference } from '@/components/record-context'
import type { SemanticState } from '@/components/ui'

export interface ApiProblem {
  type: string
  title: string
  status: number
  code: string
  detail: string
  correlationId: string
  currentVersion?: number
  fieldErrors?: Array<{ field: string; code: string; message: string }>
}

export interface OperationalLabel {
  value: string
  label: string
  semanticState: SemanticState
}

export interface OperationalOwner {
  id: string
  label: string
  reference?: string
}

export interface ActionState {
  id: string
  label: string
  permitted: boolean
  material: boolean
  blockingReason?: string
  safeNextAction?: string
  requiredConfirmation?: string
}

export interface MaterialActionResult {
  actionId: string
  resultLabel: string
  stateLabel: string
  message: string
  refreshedAt: string
}

export type WorklistStatus =
  | 'loading'
  | 'ready'
  | 'unavailable'
  | 'rejected'
  | 'partial'
  | 'reconciled'

export interface WorklistItem {
  id: string
  authoritativeCapability: string
  record: { label: string; href: string }
  scope: ScopeReference
  state: OperationalLabel
  owner: OperationalOwner
  date: string
  age: string
  amount?: { amount: string; currency: string }
  exception?: OperationalLabel
  approval?: OperationalLabel
  nextAction: string
  bulkEligible: boolean
}

export interface WorklistFilters {
  scopeId: string
  state: string
  ownerId: string
  date: string
  amount: string
  currency: string
  exception: string
  approval: string
}

export type WorklistColumnId =
  | 'record'
  | 'scope'
  | 'state'
  | 'owner'
  | 'age'
  | 'amount'
  | 'exception'
  | 'approval'
  | 'nextAction'

export interface SavedWorklistView {
  id: string
  label: string
  filters: Partial<WorklistFilters>
  visibleColumns: readonly WorklistColumnId[]
}

export interface WorklistExportContext {
  rows: readonly WorklistItem[]
  filters: WorklistFilters
  scopeId: string
  visibleColumns: readonly WorklistColumnId[]
}

export interface WorklistExportState {
  permitted: boolean
  blockedReason?: string
}

export interface SettlementAmount {
  amount: string
  currency: string
}

export interface SettlementSnapshot {
  gross: SettlementAmount
  returned: SettlementAmount
  reversed: SettlementAmount
  cancelled: SettlementAmount
  remaining: SettlementAmount
  net: SettlementAmount
  ownerAcknowledgement: OperationalLabel
  reconciliation: OperationalLabel
  exception?: OperationalLabel & { detail?: string }
}

export interface ExceptionSnapshot {
  id: string
  type: string
  scope: ScopeReference
  amount?: SettlementAmount
  owner: OperationalOwner
  evidence: readonly EvidenceLink[]
  age: string
  authorizedResolutions: readonly ActionState[]
  resultingState: OperationalLabel
}

export interface ExceptionResolutionResult {
  resolutionId: string
  state: OperationalLabel
  message: string
}

export interface EstablishedResultLink {
  label: string
  href: string
}

export interface ResultLookupQuery {
  businessIdentity: string
  requestFingerprint: string
}

export type ResultLookupOutcome =
  | {
      kind: 'safe-duplicate'
      query: ResultLookupQuery
      status: OperationalLabel
      establishedResult: EstablishedResultLink
      nextAction: string
    }
  | {
      kind: 'ambiguous-outcome'
      query: ResultLookupQuery
      status: OperationalLabel
      establishedResult?: EstablishedResultLink
      nextAction: string
    }
  | {
      kind: 'identity-content-conflict'
      query: ResultLookupQuery
      status: OperationalLabel
      establishedFingerprint: string
      establishedResult?: EstablishedResultLink
      requiredNewBusinessIdentity: EstablishedResultLink
      nextAction: string
    }

export type LookupStatus = 'idle' | 'loading' | 'found' | 'unavailable'

export type ProcessStepStatus =
  | 'current'
  | 'pending'
  | 'failed'
  | 'partially-completed'
  | 'reconciled'
  | 'terminal'

export interface ProcessStep {
  id: string
  label: string
  owner: OperationalOwner
  status: ProcessStepStatus
  detail?: string
  authoritativeRecord?: EstablishedResultLink
  evidence?: readonly EvidenceLink[]
}

export interface ProcessState {
  id: string
  label: string
  age: string
  currentState: OperationalLabel
  steps: readonly ProcessStep[]
  exception?: OperationalLabel & { detail?: string }
}
