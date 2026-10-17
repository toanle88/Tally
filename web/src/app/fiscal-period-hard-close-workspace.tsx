import { useRef, useState, type FormEvent } from 'react'

import { fpmAbortCloseRun, fpmApplyCloseApprovalDecision, fpmApplyCloseExceptionApprovalDecision, fpmExtendCloseException, fpmResumeCloseRun, fpmStartHardClose } from '@/generated/api/sdk.gen'
import type { EstablishedResult } from '@/generated/api/types.gen'
import { ProcessProgressPanel } from '@/components/process-progress-panel'
import type { ProcessState, ProcessStepStatus } from '@/components/operational-context'
import { Button, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

type ScreenId = 'FPM-WS-01' | 'FPM-SCR-02' | 'FPM-SCR-05'

type CommandIdentity = {
  fingerprint: string
  commandId: string
  idempotencyKey: string
  correlationId: string
}

type CloseCommandResponse = {
  data?: EstablishedResult
  error?: unknown
}

type CloseCommandInvoker = (identity: CommandIdentity) => Promise<CloseCommandResponse>

type CloseEvidence = {
  data: Record<string, unknown>
  expectedPeriodVersion: number
}

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `ui-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function localDateTimeValue(date: Date) {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function rfc3339(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toISOString()
}

function resultValue<T>(result: EstablishedResult, key: string): T | undefined {
  return result.data[key] as T | undefined
}

function displayValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function exceptionOwner(row: Record<string, unknown>) {
  if (typeof row.ownerId === 'string' && row.ownerId.trim() !== '') return row.ownerId
  const owner = row.owner
  if (owner && typeof owner === 'object' && !Array.isArray(owner)) {
    const ownerRecord = owner as Record<string, unknown>
    if (typeof ownerRecord.userId === 'string' && ownerRecord.userId.trim() !== '') return ownerRecord.userId
    if (typeof ownerRecord.subjectReference === 'string' && ownerRecord.subjectReference.trim() !== '') return ownerRecord.subjectReference
  }
  return '—'
}

function exceptionAge(row: Record<string, unknown>) {
  const reported = row.blockerAge ?? row.age
  if (reported !== undefined && reported !== null && reported !== '') return reported
  if (typeof row.createdAt !== 'string') return 'Not reported'
  const createdAt = Date.parse(row.createdAt)
  if (!Number.isFinite(createdAt)) return 'Not reported'
  const elapsed = Math.max(0, Date.now() - createdAt)
  if (elapsed < 60 * 60 * 1000) return 'Under 1 hour'
  if (elapsed < 24 * 60 * 60 * 1000) return `${Math.floor(elapsed / (60 * 60 * 1000))} hours`
  return `${Math.floor(elapsed / (24 * 60 * 60 * 1000))} days`
}

function errorCode(error: unknown) {
  if (error && typeof error === 'object' && 'code' in error && typeof error.code === 'string') return error.code
  return ''
}

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The authoritative close-run result could not be established. Review the current period state before retrying.'
}

function stateFor(result: EstablishedResult | null, saving: boolean, blocked: boolean): SemanticState {
  const periodState = result ? resultValue<string>(result, 'periodState') : undefined
  const closeRunStatus = result ? resultValue<string>(result, 'closeRunStatus') : undefined
  const sealStatus = result ? resultValue<string>(result, 'sealStatus') : undefined
  if (saving) return 'pending'
  if (periodState === 'HardClosed') return sealStatus === 'SealFailed' ? 'warning' : 'success'
  if (closeRunStatus === 'Failed') return 'error'
  if (closeRunStatus === 'Finalizing') return 'pending'
  if (blocked) return 'warning'
  if (result && periodState === 'Closing') return 'success'
  return 'restricted'
}

function resultRows(result: EstablishedResult | null, key: string): Array<Record<string, unknown>> {
  if (!result) return []
  const value = resultValue<unknown>(result, key)
  return Array.isArray(value) ? value.filter((item): item is Record<string, unknown> => typeof item === 'object' && item !== null && !Array.isArray(item)) : []
}

function rowString(row: Record<string, unknown>, key: string, fallback = '') {
  const value = row[key]
  return value === undefined || value === null ? fallback : String(value)
}

function rowNumber(row: Record<string, unknown>, key: string, fallback = 0) {
  const value = row[key]
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) ? parsed : fallback
}

function rowExpired(row: Record<string, unknown>) {
  const value = row.expiresAt
  if (typeof value !== 'string' || value.trim() === '') return false
  const expiresAt = Date.parse(value)
  return Number.isFinite(expiresAt) && expiresAt <= Date.now()
}

function resultNumber(result: EstablishedResult, key: string, fallback: number) {
  const value = resultValue<unknown>(result, key)
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isFinite(parsed) && parsed > 0 ? parsed : fallback
}

function processStepStatus(status: string): ProcessStepStatus {
  switch (status) {
    case 'Running': return 'current'
    case 'Completed': return 'terminal'
    case 'Failed': return 'failed'
    case 'Blocked': return 'partially-completed'
    default: return 'pending'
  }
}

function closeProcess(result: EstablishedResult | null): ProcessState {
  const rawSteps = resultRows(result, 'steps')
  const steps = rawSteps.length > 0 ? rawSteps.map((step) => ({
    id: rowString(step, 'id'),
    label: rowString(step, 'stage', 'Close task'),
    owner: { id: rowString(step, 'owner', 'FPM'), label: rowString(step, 'owner', 'FPM') },
    status: processStepStatus(rowString(step, 'status')),
    detail: rowString(step, 'blocker') || `${rowNumber(step, 'progressCount')}/${rowNumber(step, 'progressTotal')} reported`,
  })) : [{
    id: 'close-run-pending',
    label: 'Close-task coordination',
    owner: { id: 'fpm', label: 'Fiscal Period Management' },
    status: 'pending' as const,
    detail: 'No task progress has been returned by the owning close-task capabilities yet.',
  }]
  const exceptions = resultRows(result, 'exceptions')
  const unresolved = exceptions.find((exception) => rowString(exception, 'status') !== 'Approved' || rowExpired(exception))
  const closeRunStatus = result ? resultValue<string>(result, 'closeRunStatus') ?? 'Unknown' : 'Not started'
  const closeRunId = result ? resultValue<string>(result, 'closeRunId') ?? 'close-run' : 'close-run'
  return {
    id: closeRunId,
    label: 'Hard-close execution',
    age: 'Current authoritative result',
    currentState: { value: closeRunStatus, label: closeRunStatus, semanticState: stateFor(result, false, false) },
    steps,
    exception: unresolved ? { value: rowExpired(unresolved) ? 'Expired' : rowString(unresolved, 'status'), label: `${rowString(unresolved, 'code', 'Close exception')} · ${rowExpired(unresolved) ? 'Expired' : rowString(unresolved, 'status')}`, semanticState: 'warning', detail: rowString(unresolved, 'blocker') || 'Workflow approval remains required.' } : undefined,
  }
}

function closeEvidence(result: EstablishedResult, currentScope: AccountingScopeFixture, fallbacks: { periodId: string; softCloseRunId: string; controlEpoch: number; policyVersion: string; expectedGateVersion: number }): CloseEvidence {
  const expectedPeriodVersion = resultNumber(result, 'periodAggregateVersion', result.aggregateVersion)
  const expectedCloseRunVersion = resultNumber(result, 'closeRunAggregateVersion', result.aggregateVersion)
  const data: Record<string, unknown> = {
    fiscalPeriodId: resultValue<string>(result, 'periodId') ?? fallbacks.periodId,
    softCloseRunId: resultValue<string>(result, 'softCloseRunId') ?? fallbacks.softCloseRunId,
    closeRunId: resultValue<string>(result, 'closeRunId') ?? '',
    softCloseControlEpoch: resultNumber(result, 'controlEpoch', fallbacks.controlEpoch),
    policyVersion: resultValue<string>(result, 'policyVersion') ?? fallbacks.policyVersion,
    approvalPolicyVersion: resultValue<string>(result, 'approvalPolicyVersion') ?? 'close-policy-v1',
    expectedGateVersion: resultNumber(result, 'gateVersion', fallbacks.expectedGateVersion),
    expectedCloseRunVersion,
    accountingScopeId: currentScope.id,
    candidateFingerprint: resultValue<string>(result, 'closeRunCommandFingerprint') ?? '',
  }
  return { data, expectedPeriodVersion }
}

function closeActionReason(code: string, label: string) {
  if (code === 'DEPENDENCY_UNAVAILABLE') return `${label} is unavailable. The period remains restrictive; retry with the same command identity or reconcile the authoritative state.`
  if (code === 'CLOSE_TASK_FAILED') return 'A mandatory task failed. Resolve the owning task and resume the existing close-run identity.'
  if (code === 'APPROVAL_INVALIDATED') return 'The approval evidence is stale or invalidated. Refresh the close-run result and obtain a current Workflow decision.'
  if (code === 'FINALIZATION_PENDING') return 'GL finalization is pending. Resume the existing close run after confirming the authoritative gate status.'
  return `${label} was rejected. The period remains restrictive; refresh the authoritative close-run state before retrying.`
}

export function FiscalPeriodHardCloseWorkspace({ screen = 'FPM-SCR-02' }: { screen?: ScreenId }) {
  const { currentScope } = useScopeContext()
  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">{screen}</p><h2 className="mt-1 text-2xl font-semibold">Hard-close barrier</h2><p className="mt-2">Select an accounting scope before starting hard close.</p></section>
  }

  return <FiscalPeriodHardCloseScopeWorkspace key={`${screen}:${currentScope.id}`} currentScope={currentScope} screen={screen} />
}

function FiscalPeriodHardCloseScopeWorkspace({ currentScope, screen }: { currentScope: AccountingScopeFixture; screen: ScreenId }) {
  const periodId = currentScope.period?.id
  const [softCloseRunId, setSoftCloseRunId] = useState('soft-close-run-1')
  const [controlEpoch, setControlEpoch] = useState('1')
  const [periodVersion, setPeriodVersion] = useState('1')
  const [expectedGateVersion, setExpectedGateVersion] = useState('1')
  const [policyVersion, setPolicyVersion] = useState('soft-close-policy-v1')
  const [result, setResult] = useState<EstablishedResult | null>(null)
  const [notice, setNotice] = useState('The period is expected to be SoftClosed with the soft-close run retaining admission ownership until GL confirms the barrier.')
  const [blockedReason, setBlockedReason] = useState('')
  const [validationError, setValidationError] = useState('')
  const [lastErrorCode, setLastErrorCode] = useState('')
  const [saving, setSaving] = useState(false)
  const [abortReason, setAbortReason] = useState('Close-task dependency cannot complete within the current close window.')
  const [exceptionId, setExceptionId] = useState('')
  const [approvalRequestId, setApprovalRequestId] = useState('')
  const [decisionId, setDecisionId] = useState('')
  const [decisionReference, setDecisionReference] = useState('')
  const [approverUserId, setApproverUserId] = useState('')
  const [separationOfDutiesReference, setSeparationOfDutiesReference] = useState('')
  const [candidateFingerprint, setCandidateFingerprint] = useState('')
  const [subjectVersion, setSubjectVersion] = useState('')
  const [approvalPolicyVersion, setApprovalPolicyVersion] = useState('')
  const [decisionReason, setDecisionReason] = useState('Close evidence reviewed by the independent approver.')
  const [decisionOutcome, setDecisionOutcome] = useState<'approved' | 'rejected'>('approved')
  const [finalLedgerWatermark, setFinalLedgerWatermark] = useState('')
  const [extensionJustification, setExtensionJustification] = useState('The owning capability needs additional time to resolve the recorded blocker.')
  const [extensionDueAt, setExtensionDueAt] = useState(() => localDateTimeValue(new Date(Date.now() + 2 * 60 * 60 * 1000)))
  const [extensionExpiresAt, setExtensionExpiresAt] = useState(() => localDateTimeValue(new Date(Date.now() + 4 * 60 * 60 * 1000)))
  const [extensionEvidenceReference, setExtensionEvidenceReference] = useState('')
  const [extensionCandidateFingerprint, setExtensionCandidateFingerprint] = useState('')
  const commandIdentity = useRef<CommandIdentity | null>(null)
  const closeActionIdentities = useRef<Record<string, CommandIdentity>>({})

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setValidationError('')
    setBlockedReason('')
    setLastErrorCode('')
    if (saving) return
    if (!periodId) {
      setValidationError('A current fiscal period is required before hard close can start.')
      return
    }

    const parsedPeriodVersion = Number(periodVersion)
    const parsedGateVersion = Number(expectedGateVersion)
    const parsedEpoch = Number(controlEpoch)
    if (!softCloseRunId.trim() || !policyVersion.trim() || !Number.isInteger(parsedPeriodVersion) || parsedPeriodVersion < 1 || !Number.isInteger(parsedGateVersion) || parsedGateVersion < 1 || !Number.isInteger(parsedEpoch) || parsedEpoch < 1) {
      setValidationError('The soft-close run, policy, control epoch, and expected versions must be present and valid.')
      return
    }

    const fingerprint = JSON.stringify({ scopeId: currentScope.id, periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, periodVersion: parsedPeriodVersion, expectedGateVersion: parsedGateVersion, policyVersion: policyVersion.trim() })
    const identity = commandIdentity.current?.fingerprint === fingerprint
      ? commandIdentity.current
      : { fingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    commandIdentity.current = identity
    setSaving(true)
    setNotice('Submitting the hard-close command. Waiting for the authoritative GL posting-barrier result…')
    try {
      const response = await fpmStartHardClose({
        body: {
          commandId: identity.commandId,
          expectedVersion: parsedPeriodVersion,
          accountingScopeId: currentScope.id,
          data: {
            fiscalPeriodId: periodId,
            softCloseRunId: softCloseRunId.trim(),
            softCloseControlEpoch: parsedEpoch,
            policyVersion: policyVersion.trim(),
            expectedGateVersion: parsedGateVersion,
          },
        },
        headers: {
          'Idempotency-Key': identity.idempotencyKey,
          'X-Accounting-Scope-Id': currentScope.id,
          'X-Correlation-Id': identity.correlationId,
        },
      })
      if (response.error) {
        const dependencyUnavailable = errorCode(response.error) === 'DEPENDENCY_UNAVAILABLE'
        const detail = errorDetail(response.error)
        setLastErrorCode(errorCode(response.error))
        setBlockedReason(dependencyUnavailable ? 'GL is unavailable. The period remains SoftClosed and the existing soft-close owner remains restrictive.' : 'The barrier request was rejected. The period remains SoftClosed; refresh the authoritative state before retrying.')
        setNotice(detail)
        return
      }
      if (!response.data) {
        setBlockedReason('No authoritative result was returned. No local transition is assumed.')
        setNotice('The command completed without an established result. Reconcile the authoritative posting gate before retrying.')
        return
      }
      setResult(response.data)
      setNotice(resultValue<boolean>(response.data, 'replayed') ? 'The existing authoritative barrier result was returned safely.' : 'Hard close started and the authoritative posting barrier was acquired.')
    } catch (error) {
      setBlockedReason('The barrier result could not be confirmed. The period remains SoftClosed until reconciliation succeeds.')
      setNotice(error instanceof Error ? error.message : 'The hard-close command could not reach the API. Retry with the same command identity.')
    } finally {
      setSaving(false)
    }
  }

  const executeCloseAction = async (actionId: string, label: string, data: Record<string, unknown>, expectedPeriodVersion: number, invoke: CloseCommandInvoker) => {
    if (saving) return
    const fingerprint = JSON.stringify({ scopeId: currentScope.id, actionId, expectedPeriodVersion, data })
    const existing = closeActionIdentities.current[actionId]
    const identity = existing?.fingerprint === fingerprint
      ? existing
      : { fingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    closeActionIdentities.current[actionId] = identity
    setSaving(true)
    setValidationError('')
    setBlockedReason('')
    setLastErrorCode('')
    setNotice(`${label} submitted. Waiting for the authoritative close-run result…`)
    try {
      const response = await invoke(identity)
      if (response.error) {
        const code = errorCode(response.error)
        setLastErrorCode(code)
        setBlockedReason(closeActionReason(code, label))
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setBlockedReason('No established close-run result was returned. No local transition is assumed.')
        setNotice('Reconcile the authoritative period and GL gate state before retrying.')
        return
      }
      setResult(response.data)
      setNotice(resultValue<boolean>(response.data, 'replayed') ? `${label} returned the existing established result safely.` : `${label} returned an established result.`)
    } catch (error) {
      setBlockedReason(`${label} could not be confirmed. The period remains restrictive until reconciliation succeeds.`)
      setNotice(error instanceof Error ? error.message : `${label} could not reach the API. Retry with the same command identity.`)
    } finally {
      setSaving(false)
    }
  }

  const periodState = result ? resultValue<string>(result, 'periodState') : 'SoftClosed'
  const gateMode = result ? resultValue<string>(result, 'gateMode') : 'SoftClosePolicy'
  const ownerId = result ? resultValue<string>(result, 'closeRunId') : softCloseRunId
  const closeRunStatus = result ? resultValue<string>(result, 'closeRunStatus') : 'Not yet created'
  const frozenSummary = result ? resultValue<unknown>(result, 'frozenAdmissionSummary') : undefined
  const statusLabel = saving ? 'Awaiting authoritative close-run result' : `${periodState} · ${gateMode}`
  const evidence = result && periodId ? closeEvidence(result, currentScope, { periodId, softCloseRunId, controlEpoch: Number(controlEpoch), policyVersion, expectedGateVersion: Number(expectedGateVersion) }) : null
  const exceptions = resultRows(result, 'exceptions')
  const steps = resultRows(result, 'steps')
  const firstException = exceptions[0]
  const approvalRequests = result ? resultValue<unknown>(result, 'approvalRequestIds') : undefined
  const firstApprovalRequest = Array.isArray(approvalRequests) && typeof approvalRequests[0] === 'string' ? approvalRequests[0] : ''
  const effectiveExceptionId = exceptionId || (firstException ? rowString(firstException, 'id') : '')
  const effectiveApprovalRequestId = approvalRequestId || (firstException ? rowString(firstException, 'approvalRequestId') : '') || firstApprovalRequest
  const effectiveCandidateFingerprint = candidateFingerprint || (result ? resultValue<string>(result, 'closeRunCommandFingerprint') ?? '' : '')
  const effectiveSubjectVersion = subjectVersion || (result ? String(resultNumber(result, 'closeRunAggregateVersion', result.aggregateVersion)) : '')
  const effectiveFinalWatermark = finalLedgerWatermark || (result ? String(resultNumber(result, 'finalLedgerWatermark', resultNumber(result, 'barrierLedgerPosition', 0))) : '')
  const effectivePolicyVersion = approvalPolicyVersion || (result ? resultValue<string>(result, 'approvalPolicyVersion') ?? 'close-policy-v1' : 'close-policy-v1')
  const approvalReady = Boolean(result && effectiveApprovalRequestId && decisionId && decisionReference && approverUserId && separationOfDutiesReference && effectiveCandidateFingerprint && effectiveSubjectVersion && effectivePolicyVersion && decisionReason)
  const exceptionApprovalReady = Boolean(approvalReady && effectiveExceptionId)
  const finalApprovalReady = Boolean(approvalReady && (decisionOutcome === 'rejected' || Number(effectiveFinalWatermark) >= 1))
  const extensionDue = new Date(extensionDueAt)
  const extensionExpires = new Date(extensionExpiresAt)
  const extensionWindowReady = Number.isFinite(extensionDue.getTime()) && Number.isFinite(extensionExpires.getTime()) && extensionDue.getTime() > Date.now() && extensionExpires.getTime() > Date.now() && extensionDue.getTime() <= extensionExpires.getTime()
  const extensionApprovalReady = Boolean(screen === 'FPM-SCR-05' && result && effectiveExceptionId && effectiveApprovalRequestId && decisionId && decisionReference && approverUserId && separationOfDutiesReference && extensionCandidateFingerprint && effectiveSubjectVersion && effectivePolicyVersion && decisionReason && extensionJustification.trim() && extensionEvidenceReference.trim() && rfc3339(extensionDueAt) && rfc3339(extensionExpiresAt) && extensionWindowReady)
  const safeNextAction = result ? resultValue<string>(result, 'nextAction') : undefined
  const title = screen === 'FPM-SCR-05' ? 'Review close exceptions and approvals' : screen === 'FPM-SCR-02' ? 'Start hard close' : 'Execute and recover hard close'

  const commandFor = (identity: CommandIdentity, data: Record<string, unknown>, expectedPeriodVersion: number) => ({ body: { commandId: identity.commandId, expectedVersion: expectedPeriodVersion, accountingScopeId: currentScope.id, data }, headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': currentScope.id, 'X-Correlation-Id': identity.correlationId } })
  const startCloseData = evidence?.data ?? {}

  const process = closeProcess(result)
  const taskFailed = steps.some((step) => rowString(step, 'status') === 'Failed') || closeRunStatus === 'Failed' && lastErrorCode === 'CLOSE_TASK_FAILED'
  const unresolvedException = exceptions.some((exception) => ['Unresolved', 'Expired', 'Rejected'].includes(rowString(exception, 'status')) || rowExpired(exception))
  const lifecycleSignals = [
    taskFailed ? { state: 'error' as const, label: 'Task failure', detail: 'A mandatory close task requires resolution before final approval.' } : null,
    unresolvedException ? { state: 'warning' as const, label: 'Unresolved or expired exception', detail: 'Exception approval evidence is required before finalization.' } : null,
    lastErrorCode === 'APPROVAL_INVALIDATED' ? { state: 'error' as const, label: 'Approval invalidated', detail: 'The current Workflow decision no longer matches the close-run evidence.' } : null,
    lastErrorCode === 'DEPENDENCY_UNAVAILABLE' ? { state: 'warning' as const, label: 'Dependency unavailable', detail: 'The current restrictive state is preserved and the action is retryable.' } : null,
    closeRunStatus === 'Finalizing' || lastErrorCode === 'FINALIZATION_PENDING' ? { state: 'pending' as const, label: 'Finalization pending', detail: 'GL finalization must be confirmed before the period transition is complete.' } : null,
    resultValue<string>(result ?? { data: {} } as EstablishedResult, 'sealStatus') === 'SealPending' ? { state: 'pending' as const, label: 'Seal pending', detail: 'Accounting close is established; Audit seal processing remains separate.' } : null,
    resultValue<string>(result ?? { data: {} } as EstablishedResult, 'sealStatus') === 'SealFailed' ? { state: 'warning' as const, label: 'Seal failed', detail: 'The period remains HardClosed. Retry the audit seal without reopening accounting.' } : null,
  ].flatMap((signal) => signal ? [signal] : [])

  return <section aria-labelledby="fpm-hard-close-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">{screen}</p><h2 id="fpm-hard-close-title" className="mt-1 text-2xl font-semibold">{title}</h2><p className="mt-2 max-w-4xl text-base-content/75">Create or recover one close-run identity, execute owned close tasks, apply current Workflow evidence, and finalize only through the authoritative GL posting gate. A pending dependency or audit seal never implies an open period.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">A pending or failed dependency keeps the period restrictive. Resume and abort commands reuse the visible process identity and idempotency evidence; they never create a second posting-gate owner.</p>
    <Panel title="Period-control status" description="The display distinguishes the restrictive FPM state from the authoritative GL gate and close-run result." state={stateFor(result, saving, Boolean(blockedReason))}>
      <div className="flex flex-wrap items-center gap-3"><StatusBadge state={stateFor(result, saving, Boolean(blockedReason))} label={statusLabel} announce /><span className="text-sm text-base-content/70">Scope {currentScope.id} · period {displayValue(periodId)}</span></div>
      <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Period state</dt><dd>{displayValue(periodState)}</dd></div><div><dt className="text-sm text-base-content/65">Gate mode</dt><dd>{displayValue(gateMode)}</dd></div><div><dt className="text-sm text-base-content/65">Gate version</dt><dd>{displayValue(result ? resultValue<number>(result, 'gateVersion') : expectedGateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Active owner</dt><dd className="break-all font-mono">{displayValue(ownerId)}</dd></div><div><dt className="text-sm text-base-content/65">Close-run status</dt><dd>{displayValue(closeRunStatus)}</dd></div><div><dt className="text-sm text-base-content/65">Barrier ledger position</dt><dd>{displayValue(result ? resultValue<number>(result, 'barrierLedgerPosition') : undefined)}</dd></div><div><dt className="text-sm text-base-content/65">Final ledger watermark</dt><dd>{displayValue(result ? resultValue<number>(result, 'finalLedgerWatermark') : undefined)}</dd></div><div><dt className="text-sm text-base-content/65">Audit seal</dt><dd>{displayValue(result ? resultValue<string>(result, 'sealStatus') : 'NotRequested')}</dd></div></dl>
      {blockedReason ? <p role="alert" className="mt-5 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{blockedReason}</p> : null}
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
    </Panel>
    <Panel title="Hard-close command" description="These values identify the active soft-close epoch and the expected GL gate version. GL remains the owner of posting admission.">
      <form onSubmit={(event) => void submit(event)}>
        <div className="grid gap-4 md:grid-cols-2"><Field id="fpm-hard-close-period-id" label="Fiscal period ID" value={periodId ?? ''} readOnly disabled description="Taken from the selected accounting scope." /><Field id="fpm-hard-close-soft-close-run-id" label="Active soft-close run ID" value={softCloseRunId} onChange={(event) => setSoftCloseRunId(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-control-epoch" label="Soft-close control epoch" type="number" min="1" value={controlEpoch} onChange={(event) => setControlEpoch(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-period-version" label="Expected period version" type="number" min="1" value={periodVersion} onChange={(event) => setPeriodVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-gate-version" label="Expected GL gate version" type="number" min="1" value={expectedGateVersion} onChange={(event) => setExpectedGateVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-policy-version" label="Soft-close policy version" value={policyVersion} onChange={(event) => setPolicyVersion(event.target.value)} required disabled={saving || Boolean(result)} /></div>
        {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
        <div className="mt-4 flex flex-wrap items-center gap-3"><Button type="submit" loading={saving} disabled={Boolean(result)}>Start hard close and acquire barrier</Button>{result ? <span className="text-sm text-base-content/70">The authoritative result is loaded. Further close work belongs to the close-run workflow below.</span> : null}</div>
      </form>
    </Panel>
    <div className="grid min-w-0 items-start gap-6 xl:grid-cols-2">
      <Panel title="Close-run progress and lifecycle signals" description="Progress is retained by FPM while each owning capability remains responsible for its task result." state={lifecycleSignals.length > 0 ? 'warning' : 'info'}>
        <ProcessProgressPanel process={process} />
        <div className="mt-5 space-y-3"><h3 className="font-semibold">Recovery and approval signals</h3>{lifecycleSignals.length === 0 ? <p className="text-sm text-base-content/70">No task, exception, approval, finalization, dependency, or seal exception is currently reported.</p> : <ul className="space-y-3">{lifecycleSignals.map((signal) => <li key={signal.label} className="rounded-box border border-base-300 p-3"><div className="flex flex-wrap items-center gap-2"><StatusBadge state={signal.state} label={signal.label} /><span className="text-sm">{signal.detail}</span></div></li>)}</ul>}</div>
      </Panel>
      <Panel title="Recovery actions" description="Use the existing close-run identity. Server-side authorization, GL ownership, version checks, and idempotency remain authoritative.">
        <div className="flex flex-wrap gap-3"><Button disabled={!result || saving} loading={saving && closeRunStatus !== 'Finalizing'} onClick={() => { if (!result || !evidence) return; void executeCloseAction('resume', 'Resume close run', startCloseData, evidence.expectedPeriodVersion, (identity) => fpmResumeCloseRun(commandFor(identity, startCloseData, evidence.expectedPeriodVersion))) }}>Resume close run</Button><Button variant="danger" disabled={!result || saving || periodState === 'HardClosed' || closeRunStatus === 'Completed'} onClick={() => { if (!result || !evidence || !abortReason.trim()) return; void executeCloseAction('abort', 'Abort close run', { ...startCloseData, reason: abortReason.trim() }, evidence.expectedPeriodVersion, (identity) => fpmAbortCloseRun(commandFor(identity, { ...startCloseData, reason: abortReason.trim() }, evidence.expectedPeriodVersion))) }}>Abort initial close run</Button></div>
        <Field id="fpm-hard-close-abort-reason" label="Abort reason" value={abortReason} onChange={(event) => setAbortReason(event.target.value)} disabled={!result || saving || periodState === 'HardClosed'} description="An initial barrier can be released only when GL confirms that no close posting was admitted." />
        <p className="mt-4 text-sm text-base-content/70">Abort is intentionally unavailable after finalization, after admitted close posting, or for a reclose run. A rejected abort leaves the current restrictive owner unchanged.</p>
      </Panel>
    </div>
    <Panel title="Exception and final approval evidence" description="Supply immutable Workflow decision references against the current close-run fingerprint and aggregate version. The final action also requires independent SoD evidence and a ledger watermark.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="fpm-close-exception-id" label="Close exception ID" value={exceptionId || effectiveExceptionId} onChange={(event) => setExceptionId(event.target.value)} disabled={!result || saving} description="Defaults to the first returned exception." /><Field id="fpm-close-approval-request-id" label="Approval request ID" value={approvalRequestId || effectiveApprovalRequestId} onChange={(event) => setApprovalRequestId(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-decision-id" label="Decision ID" value={decisionId} onChange={(event) => setDecisionId(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-decision-reference" label="Decision reference" value={decisionReference} onChange={(event) => setDecisionReference(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-approver-user-id" label="Approver user ID" value={approverUserId} onChange={(event) => setApproverUserId(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-sod-reference" label="Separation-of-duties reference" value={separationOfDutiesReference} onChange={(event) => setSeparationOfDutiesReference(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-candidate-fingerprint" label="Candidate command fingerprint" value={candidateFingerprint || effectiveCandidateFingerprint} onChange={(event) => setCandidateFingerprint(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-subject-version" label="Close-run subject version" type="number" min="1" value={subjectVersion || effectiveSubjectVersion} onChange={(event) => setSubjectVersion(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-approval-policy-version" label="Approval policy version" value={approvalPolicyVersion || effectivePolicyVersion} onChange={(event) => setApprovalPolicyVersion(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-decision-reason" label="Decision reason" value={decisionReason} onChange={(event) => setDecisionReason(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-ledger-watermark" label="Final ledger watermark" type="number" min="1" value={finalLedgerWatermark || effectiveFinalWatermark} onChange={(event) => setFinalLedgerWatermark(event.target.value)} disabled={!result || saving} /><div><label className="label cursor-pointer justify-start gap-2" htmlFor="fpm-close-decision-outcome"><span className="font-medium text-base-content">Decision outcome</span></label><Select id="fpm-close-decision-outcome" value={decisionOutcome} onChange={(event) => setDecisionOutcome(event.target.value as 'approved' | 'rejected')} disabled={!result || saving}><option value="approved">Approved</option><option value="rejected">Rejected</option></Select></div></div>
      <div className="mt-5 flex flex-wrap gap-3"><Button disabled={!exceptionApprovalReady || saving} onClick={() => { if (!result || !evidence || !exceptionApprovalReady) return; const data = { ...startCloseData, exceptionId: effectiveExceptionId, approvalRequestId: effectiveApprovalRequestId, decisionId, decisionVersion: 1, outcome: decisionOutcome, decisionReason, approvalPolicyVersion: effectivePolicyVersion, subjectVersion: Number(effectiveSubjectVersion), candidateFingerprint: effectiveCandidateFingerprint, decisionReference, approverUserId, separationOfDutiesReference, recordedAt: new Date().toISOString() }; void executeCloseAction('exception-approval', 'Apply close-exception approval decision', data, evidence.expectedPeriodVersion, (identity) => fpmApplyCloseExceptionApprovalDecision(commandFor(identity, data, evidence.expectedPeriodVersion))) }}>Apply close-exception approval decision</Button><Button disabled={!finalApprovalReady || saving} onClick={() => { if (!result || !evidence || !finalApprovalReady) return; const data = { ...startCloseData, approvalRequestId: effectiveApprovalRequestId, decisionId, decisionVersion: 1, outcome: decisionOutcome, decisionReason, approvalPolicyVersion: effectivePolicyVersion, subjectVersion: Number(effectiveSubjectVersion), candidateFingerprint: effectiveCandidateFingerprint, decisionReference, approverUserId, separationOfDutiesReference, finalLedgerWatermark: Number(effectiveFinalWatermark), recordedAt: new Date().toISOString() }; void executeCloseAction('final-approval', 'Apply final close approval decision', data, evidence.expectedPeriodVersion, (identity) => fpmApplyCloseApprovalDecision(commandFor(identity, data, evidence.expectedPeriodVersion))) }}>Apply final close approval decision</Button></div>
      {screen === 'FPM-SCR-05' ? <div className="mt-6 rounded-box border border-base-300 p-4"><h3 className="font-semibold">Exception extension review</h3><p className="mt-1 text-sm text-base-content/70">An extension preserves the current exception as unresolved and stores a separate approval decision. It cannot finalize the close or weaken the GL gate.</p>{firstException ? <dl className="mt-4 grid gap-3 sm:grid-cols-2"><div><dt className="text-sm text-base-content/65">Owner</dt><dd>{exceptionOwner(firstException)}</dd></div><div><dt className="text-sm text-base-content/65">Impact</dt><dd>{displayValue(firstException.impact)}</dd></div><div><dt className="text-sm text-base-content/65">Current due time</dt><dd>{displayValue(firstException.dueAt)}</dd></div><div><dt className="text-sm text-base-content/65">Blocker age</dt><dd>{displayValue(exceptionAge(firstException))}</dd></div><div className="sm:col-span-2"><dt className="text-sm text-base-content/65">Justification</dt><dd>{displayValue(firstException.justification ?? firstException.blocker)}</dd></div><div className="sm:col-span-2"><dt className="text-sm text-base-content/65">Extension approval evidence/history</dt><dd>{displayValue(firstException.extensionHistory)}</dd></div></dl> : <p role="note" className="mt-4 text-sm text-base-content/70">Load a current close-run result before extending an exception.</p>}<p role="note" className="mt-4 text-sm text-base-content/75">Safe next action: {displayValue(safeNextAction ?? 'Resolve the blocker or obtain an independent extension approval; keep the restrictive gate unchanged.')}</p><div className="mt-4 grid gap-4 md:grid-cols-2"><Field id="fpm-close-extension-justification" label="Extension justification" value={extensionJustification} onChange={(event) => setExtensionJustification(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-extension-evidence" label="Extension evidence reference" value={extensionEvidenceReference} onChange={(event) => setExtensionEvidenceReference(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-extension-due-at" label="New due time" type="datetime-local" value={extensionDueAt} onChange={(event) => setExtensionDueAt(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-extension-expires-at" label="New approval expiry" type="datetime-local" value={extensionExpiresAt} onChange={(event) => setExtensionExpiresAt(event.target.value)} disabled={!result || saving} /><Field id="fpm-close-extension-candidate" label="Extension candidate fingerprint" value={extensionCandidateFingerprint} onChange={(event) => setExtensionCandidateFingerprint(event.target.value)} disabled={!result || saving} description="This fingerprint is independent from the original close approval candidate." /></div><div className="mt-4 flex flex-wrap gap-3"><Button disabled={!extensionApprovalReady || saving} onClick={() => { if (!result || !evidence || !extensionApprovalReady) return; const dueAt = rfc3339(extensionDueAt); const expiresAt = rfc3339(extensionExpiresAt); const data = { fiscalPeriodId: periodId, closeRunId: resultValue<string>(result, 'closeRunId') ?? '', exceptionId: effectiveExceptionId, expectedCloseRunVersion: resultNumber(result, 'closeRunAggregateVersion', evidence.expectedPeriodVersion), expectedGateVersion: resultNumber(result, 'gateVersion', Number(expectedGateVersion)), expectedControlAuthorityVersion: resultNumber(result, 'authorityVersion', 1), expectedControlAuthorityEpoch: resultNumber(result, 'authorityEpoch', Number(controlEpoch)), justification: extensionJustification.trim(), newDueAt: dueAt, newExpiresAt: expiresAt, evidenceReference: extensionEvidenceReference.trim(), approvalRequestId: effectiveApprovalRequestId, decisionId, decisionVersion: 1, outcome: decisionOutcome, decisionReason, approvalPolicyVersion: effectivePolicyVersion, subjectVersion: Number(effectiveSubjectVersion), candidateFingerprint: extensionCandidateFingerprint.trim(), decisionReference, approverUserId, separationOfDutiesReference, recordedAt: new Date().toISOString() }; void executeCloseAction('exception-extension', 'Extend close exception', data, evidence.expectedPeriodVersion, (identity) => fpmExtendCloseException(commandFor(identity, data, evidence.expectedPeriodVersion))) }}>Extend current close exception</Button></div>{result && !extensionApprovalReady ? <p role="note" className="mt-3 text-sm text-base-content/70">Extension requires a current exception, future due/expiry values in order, a separate candidate fingerprint, and independent Workflow evidence.</p> : null}</div> : null}
      {result && !approvalReady ? <p role="note" className="mt-4 rounded-box border border-warning/40 bg-warning/5 p-3 text-sm">Workflow evidence is incomplete. Current exception/approval identifiers, decision references, the close-run fingerprint and subject version, independent approver, SoD reference, and reason are required before an approval action is enabled.</p> : null}
      {result && steps.length === 0 && exceptions.length === 0 ? <p role="note" className="mt-4 text-sm text-base-content/70">No close exceptions or task evidence have been returned yet. Resume the run to obtain the owning capability progress projection.</p> : null}
    </Panel>
    <Panel title="Authoritative barrier and close evidence" description="Only the committed GL result can move the period from SoftClosed to Closing or HardClosed." state={result ? 'success' : 'info'}>
      {!result ? <p role="note" className="text-sm text-base-content/75">No barrier result is loaded. The candidate close run is not a posting-gate owner.</p> : <><dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Result state</dt><dd>{displayValue(resultValue<string>(result, 'validationOutcome'))}</dd></div><div><dt className="text-sm text-base-content/65">Frozen summary</dt><dd>{frozenSummary ? 'Recorded' : 'Missing'}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="break-all font-mono">{displayValue(resultValue<string>(result, 'auditReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Decision reference</dt><dd className="break-all font-mono">{displayValue(resultValue<string>(result, 'decisionReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Period aggregate version</dt><dd>{displayValue(result.aggregateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Close-run aggregate version</dt><dd>{displayValue(resultValue<number>(result, 'closeRunAggregateVersion'))}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{resultValue<boolean>(result, 'replayed') ? 'Existing result returned' : 'First establishment'}</dd></div><div><dt className="text-sm text-base-content/65">Seal request</dt><dd className="break-all font-mono">{displayValue(resultValue<string>(result, 'sealRequestId'))}</dd></div></dl><div className="mt-5 rounded-box border border-base-300 bg-base-200/30 p-4"><h3 className="font-semibold">Frozen admission summary</h3><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-xs text-base-content/75">{frozenSummary ? JSON.stringify(frozenSummary, null, 2) : '—'}</pre></div><p role="note" className="mt-4 text-sm text-base-content/75">The soft-close handoff is pending and the close run owns the CloseOnly barrier. Ordinary posting must use the authoritative GL gate rules. A failed or pending audit seal never reopens a HardClosed accounting period.</p></>}
    </Panel>
  </section>
}
