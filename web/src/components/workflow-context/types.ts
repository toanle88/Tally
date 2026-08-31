import type { EstablishedResultLink, ApiProblem, OperationalOwner } from '@/components/operational-context'
import type { ScopeReference } from '@/components/record-context'
import type { SemanticState } from '@/components/ui'

export type ValidationIssueCategory =
  | 'field'
  | 'line'
  | 'business-rule'
  | 'authorization'
  | 'dependency'
  | 'conflict'

export interface ValidationIssue {
  id: string
  category: ValidationIssueCategory
  code: string
  message: string
  targetId: string
  targetLabel: string
  nextAction?: string
}

export type ApprovalStatus =
  | 'requested'
  | 'pending'
  | 'delegated'
  | 'escalated'
  | 'decided'
  | 'applied'
  | 'rejected'
  | 'expired'
  | 'invalidated'

export interface ApprovalStep {
  id: string
  label: string
  owner: OperationalOwner
  status: ApprovalStatus
  decision?: string
  decidedAt?: string
}

export type ApprovalApplicationStatus =
  | 'not-applied'
  | 'awaiting-revalidation'
  | 'applied'
  | 'rejected'
  | 'invalidated'

export interface ApprovalSnapshot {
  requestId: string
  subject: { label: string; reference: string; version: number; snapshotLabel: string }
  scope: ScopeReference
  policy: { label: string; version: string }
  submittedBy: string
  requestedAt: string
  status: ApprovalStatus
  steps: readonly ApprovalStep[]
  decision?: { id: string; outcome: string; decidedAt: string; decisionVersion: number }
  application: {
    status: ApprovalApplicationStatus
    requiredSubjectVersion: number
    appliedSubjectVersion?: number
    reason?: string
  }
  delegation?: string
  escalation?: string
}

export type PostingStatus =
  | 'request'
  | 'pending'
  | 'established-result'
  | 'rejected'
  | 'failure'
  | 'reversal'
  | 'reconciliation'

export interface PostingSnapshot {
  postingId: string
  sourceRecord: EstablishedResultLink
  scope: ScopeReference
  sourceVersion: number
  purpose: string
  accountingOwner: string
  status: PostingStatus
  requestReference: string
  establishedResult?: EstablishedResultLink
  journalReference?: EstablishedResultLink
  periodGateEvidence?: {
    fiscalPeriodId: string
    periodStateVersion: number
    postingGateVersion: number
    admission: string
  }
  failure?: { detail: string; retryPermitted: boolean; nextAction: string }
  reversal?: EstablishedResultLink
  reconciliation?: { state: string; evidence?: EstablishedResultLink }
}

export type ConflictKind = 'version' | 'identity-content'

export interface ConflictValueChange {
  id: string
  label: string
  reviewedValue: string
  currentValue: string
}

export interface VersionConflictSnapshot {
  kind: ConflictKind
  attemptedAction: string
  record: EstablishedResultLink
  scope: ScopeReference
  expected: { version: number; state: string }
  current: { version: number; state: string }
  changedValues: readonly ConflictValueChange[]
  currentOwner: OperationalOwner
  processEpoch: string
  establishedResult?: EstablishedResultLink
  safeRetry: { permitted: boolean; reason: string }
  requiredNewBusinessIdentity?: EstablishedResultLink
}

export interface MaterialActionFormValues {
  businessIdentity: string
  description: string
  amount: string
  currency: string
  lineReference: string
}

export interface MaterialActionSubmission {
  businessIdentity: string
  description: string
  amount: string
  currency: string
  lineReference: string
  scopeId: string
  reviewedVersion: number
}

export type ProblemFixtureKind =
  | 'domain-rejection'
  | 'authorization-denial'
  | 'version-conflict'
  | 'idempotency-conflict'
  | 'dependency-unavailable'
  | 'ambiguous-outcome'
  | 'unexpected-failure'

export type FixtureSubmissionOutcome =
  | { kind: 'success'; result: { label: string; state: string; correlationId: string } }
  | { kind: 'problem'; problemKind: ProblemFixtureKind; problem: ApiProblem; conflict?: VersionConflictSnapshot }

export interface WorkflowLabel {
  label: string
  semanticState: SemanticState
}
