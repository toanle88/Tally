import { useRef, useState, type FormEvent } from 'react'

import { fpmTakeOverPeriodControl } from '@/generated/api/sdk.gen'
import type { EstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

type CommandIdentity = {
  fingerprint: string
  commandId: string
  idempotencyKey: string
  correlationId: string
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

function resultValue<T>(result: EstablishedResult | null, key: string): T | undefined {
  return result ? result.data[key] as T | undefined : undefined
}

function displayValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function errorCode(error: unknown) {
  return error && typeof error === 'object' && 'code' in error && typeof error.code === 'string' ? error.code : ''
}

function errorDetail(error: unknown) {
  return error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string'
    ? error.detail
    : 'The authoritative recovery result could not be established. Refresh the period and GL gate state before retrying.'
}

function semanticState(result: EstablishedResult | null, saving: boolean, blocked: boolean): SemanticState {
  if (saving) return 'pending'
  if (blocked) return 'warning'
  if (resultValue<string>(result, 'pendingReconciliation') === 'true' || resultValue<boolean>(result, 'pendingReconciliation')) return 'warning'
  return result ? 'success' : 'restricted'
}

export function FiscalPeriodControlRecoveryWorkspace() {
  const { currentScope } = useScopeContext()
  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">FPM-SCR-04</p><h2 className="mt-1 text-2xl font-semibold">Recover period control</h2><p className="mt-2">Select an accounting scope before recovering an interrupted period-control process.</p></section>
  }
  return <FiscalPeriodControlRecoveryScopeWorkspace key={currentScope.id} currentScope={currentScope} />
}

function FiscalPeriodControlRecoveryScopeWorkspace({ currentScope }: { currentScope: AccountingScopeFixture }) {
  const periodID = currentScope.period?.id ?? ''
  const [target, setTarget] = useState<'softClose' | 'closeRun'>('softClose')
  const [processID, setProcessID] = useState('soft-close-run-1')
  const [periodVersion, setPeriodVersion] = useState('1')
  const [processVersion, setProcessVersion] = useState('1')
  const [gateVersion, setGateVersion] = useState('1')
  const [authorityVersion, setAuthorityVersion] = useState('1')
  const [authorityEpoch, setAuthorityEpoch] = useState('1')
  const [reason, setReason] = useState('Recover an interrupted period-control process after the prior authority expired.')
  const [evidenceReference, setEvidenceReference] = useState('')
  const [newExpiresAt, setNewExpiresAt] = useState(() => localDateTimeValue(new Date(Date.now() + 60 * 60 * 1000)))
  const [approvalRequestID, setApprovalRequestID] = useState('')
  const [decisionID, setDecisionID] = useState('')
  const [decisionVersion, setDecisionVersion] = useState('1')
  const [decisionOutcome, setDecisionOutcome] = useState<'approved' | 'rejected'>('approved')
  const [decisionReason, setDecisionReason] = useState('Independent recovery review completed.')
  const [approvalPolicyVersion, setApprovalPolicyVersion] = useState('period-control-recovery-v1')
  const [subjectVersion, setSubjectVersion] = useState('1')
  const [candidateFingerprint, setCandidateFingerprint] = useState('')
  const [decisionReference, setDecisionReference] = useState('')
  const [approverUserID, setApproverUserID] = useState('')
  const [separationReference, setSeparationReference] = useState('sod-period-control-recovery-1')
  const [recordedAt, setRecordedAt] = useState(() => localDateTimeValue(new Date()))
  const [result, setResult] = useState<EstablishedResult | null>(null)
  const [saving, setSaving] = useState(false)
  const [validationError, setValidationError] = useState('')
  const [blockedReason, setBlockedReason] = useState('')
  const [pendingReconciliation, setPendingReconciliation] = useState(false)
  const [notice, setNotice] = useState('Recovery is read-only with respect to GL: FPM will reconcile through GetPostingGateStatus before changing ownership.')
  const commandIdentity = useRef<CommandIdentity | null>(null)

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (saving) return
    setValidationError('')
    setBlockedReason('')
    if (!periodID || !processID.trim() || !reason.trim() || !evidenceReference.trim() || !newExpiresAt || !approvalRequestID.trim() || !decisionID.trim() || !decisionReference.trim() || !approverUserID.trim() || !candidateFingerprint.trim() || !separationReference.trim() || !decisionReason.trim()) {
      setValidationError('The process, reason, evidence reference, future expiry, and complete independent approval evidence are required.')
      return
    }
    const numbers = [periodVersion, processVersion, gateVersion, authorityVersion, authorityEpoch, decisionVersion, subjectVersion].map(Number)
    if (numbers.some((value) => !Number.isInteger(value) || value < 1) || !rfc3339(newExpiresAt) || !rfc3339(recordedAt)) {
      setValidationError('Expected versions, approval versions, expiry, and recorded-at evidence must be valid.')
      return
    }
    const data = {
      fiscalPeriodId: periodID,
      target,
      processId: processID.trim(),
      expectedProcessVersion: numbers[1],
      expectedGateVersion: numbers[2],
      expectedControlAuthorityVersion: numbers[3],
      expectedControlAuthorityEpoch: numbers[4],
      reason: reason.trim(),
      evidenceReference: evidenceReference.trim(),
      newAuthorityExpiresAt: rfc3339(newExpiresAt),
      approvalRequestId: approvalRequestID.trim(),
      decisionId: decisionID.trim(),
      decisionVersion: numbers[5],
      outcome: decisionOutcome,
      decisionReason: decisionReason.trim(),
      approvalPolicyVersion: approvalPolicyVersion.trim(),
      subjectVersion: numbers[6],
      candidateFingerprint: candidateFingerprint.trim(),
      decisionReference: decisionReference.trim(),
      approverUserId: approverUserID.trim(),
      separationOfDutiesReference: separationReference.trim(),
      recordedAt: rfc3339(recordedAt),
    }
    const fingerprint = JSON.stringify({ scopeID: currentScope.id, expectedPeriodVersion: Number(periodVersion), data })
    const existing = commandIdentity.current
    const identity = existing?.fingerprint === fingerprint
      ? existing
      : { fingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    commandIdentity.current = identity
    setSaving(true)
    setNotice('Submitting the takeover. Waiting for the authoritative GL gate status…')
    try {
      const response = await fpmTakeOverPeriodControl({
        body: { commandId: identity.commandId, expectedVersion: Number(periodVersion), accountingScopeId: currentScope.id, data },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': currentScope.id, 'X-Correlation-Id': identity.correlationId },
      })
      if (response.error) {
        const code = errorCode(response.error)
        setPendingReconciliation(code === 'DEPENDENCY_UNAVAILABLE')
        setBlockedReason(code === 'DEPENDENCY_UNAVAILABLE' ? 'The authoritative gate status is unavailable. The period remains restrictive and no takeover is assumed.' : 'The takeover was not applied. Refresh the owner, authority, and gate evidence before retrying.')
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setPendingReconciliation(true)
        setBlockedReason('No established result was returned. No ownership transition is assumed.')
        return
      }
      setPendingReconciliation(false)
      setResult(response.data)
      setPeriodVersion(String(resultValue<number>(response.data, 'periodAggregateVersion') ?? Number(periodVersion)))
      setProcessVersion(String(resultValue<number>(response.data, 'closeRunAggregateVersion') ?? resultValue<number>(response.data, 'processAggregateVersion') ?? Number(processVersion)))
      setGateVersion(String(resultValue<number>(response.data, 'gateVersion') ?? Number(gateVersion)))
      setAuthorityVersion(String(resultValue<number>(response.data, 'authorityVersion') ?? Number(authorityVersion)))
      setAuthorityEpoch(String(resultValue<number>(response.data, 'authorityEpoch') ?? Number(authorityEpoch)))
      setNotice(resultValue<boolean>(response.data, 'replayed') ? 'The existing takeover result was replayed safely.' : 'Control authority was recovered and the process identity was preserved.')
    } catch (error) {
      setPendingReconciliation(true)
      setBlockedReason('The takeover outcome could not be confirmed. The period remains restrictive until authoritative state is reconciled.')
      setNotice(error instanceof Error ? error.message : 'The takeover could not reach the API. Retry with the same command identity.')
    } finally {
      setSaving(false)
    }
  }

  const authority = resultValue<Record<string, unknown>>(result, 'controlAuthority')
  const history = resultValue<unknown[]>(result, 'ownershipHistory')
  const gate = resultValue<Record<string, unknown>>(result, 'authoritativeGateStatus')
  const state = semanticState(result, saving, Boolean(blockedReason))
  const statusLabel = saving ? 'Awaiting authoritative gate status' : result ? `${displayValue(resultValue<string>(result, 'periodState'))} · ${displayValue(resultValue<string>(result, 'replayStatus'))}` : 'Recovery not submitted'
  const reportedPendingReconciliation = resultValue<boolean>(result, 'pendingReconciliation')

  return <section aria-labelledby="fpm-recovery-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">FPM-SCR-04</p><h1 id="fpm-recovery-title" className="mt-1 text-2xl font-semibold">Recover period control</h1><p className="mt-2 max-w-4xl text-base-content/75">Take over an eligible interrupted SoftCloseRun or CloseRun while preserving process identity, epoch lineage, cutoff, previous owners, and audit evidence. GL is never opened, released, or weakened by this workspace.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">A missing owner/authority record, active authority, stale version, ambiguous gate, or unavailable dependency keeps the period restrictive. Use the same command identity when retrying.</p>
    <Panel title="Authoritative recovery status" description="The fields below are established-result projections. Gate state is reconciled only through GetPostingGateStatus." state={state}>
      <div className="flex flex-wrap items-center gap-3"><StatusBadge state={state} label={statusLabel} announce /><span className="text-sm text-base-content/70">Scope {currentScope.id} · period {displayValue(periodID)}</span></div>
      <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Process identity</dt><dd className="break-all font-mono">{displayValue(result?.processId ?? resultValue<string>(result, 'processId') ?? processID)}</dd></div><div><dt className="text-sm text-base-content/65">Current owner</dt><dd className="break-all font-mono">{displayValue(resultValue<string>(result, 'ownerId') ?? authority?.ownerId)}</dd></div><div><dt className="text-sm text-base-content/65">Authority epoch</dt><dd>{displayValue(resultValue<number>(result, 'authorityEpoch') ?? authority?.authorityEpoch)}</dd></div><div><dt className="text-sm text-base-content/65">Authority expiry</dt><dd>{displayValue(resultValue<string>(result, 'authorityExpiresAt') ?? authority?.expiresAt)}</dd></div><div><dt className="text-sm text-base-content/65">Immutable cutoff</dt><dd>{displayValue(resultValue<string>(result, 'controlCutoffAt') ?? authority?.cutoffAt)}</dd></div><div><dt className="text-sm text-base-content/65">Gate mode/version</dt><dd>{displayValue(gate?.gateMode ?? resultValue<string>(result, 'gateMode'))} / {displayValue(gate?.gateVersion ?? resultValue<number>(result, 'gateVersion'))}</dd></div><div><dt className="text-sm text-base-content/65">Pending reconciliation</dt><dd>{pendingReconciliation || reportedPendingReconciliation ? 'Pending — restrictive' : 'None established'}</dd></div><div><dt className="text-sm text-base-content/65">Next safe action</dt><dd>{displayValue(resultValue<string>(result, 'nextAction') ?? 'Confirm current authority and gate status')}</dd></div><div><dt className="text-sm text-base-content/65">Replay status</dt><dd>{displayValue(resultValue<string>(result, 'replayStatus'))}</dd></div></dl>
      {history && history.length > 0 ? <div className="mt-5"><h3 className="font-semibold">Ownership history</h3><pre className="mt-2 max-h-52 overflow-auto whitespace-pre-wrap break-words rounded-box border border-base-300 bg-base-200/30 p-3 text-xs">{JSON.stringify(history, null, 2)}</pre></div> : null}
      {blockedReason ? <p role="alert" className="mt-5 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{blockedReason}</p> : null}
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
    </Panel>
    <Panel title="Takeover command" description="All expected versions and the independent Workflow evidence are revalidated by FPM before ownership changes.">
      <form onSubmit={(event) => void submit(event)}>
        <div className="grid gap-4 md:grid-cols-2"><div><label className="label cursor-pointer justify-start gap-2" htmlFor="fpm-recovery-target"><span className="font-medium text-base-content">Process target</span></label><Select id="fpm-recovery-target" value={target} onChange={(event) => { const value = event.target.value as 'softClose' | 'closeRun'; setTarget(value); setProcessID(value === 'softClose' ? 'soft-close-run-1' : 'close-run-1') }} disabled={saving}><option value="softClose">SoftCloseRun</option><option value="closeRun">CloseRun</option></Select></div><Field id="fpm-recovery-period-id" label="Fiscal period ID" value={periodID} readOnly disabled description="Taken from the selected accounting scope." /><Field id="fpm-recovery-process-id" label="Interrupted process ID" value={processID} onChange={(event) => setProcessID(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-period-version" label="Expected period version" type="number" min="1" value={periodVersion} onChange={(event) => setPeriodVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-process-version" label="Expected process version" type="number" min="1" value={processVersion} onChange={(event) => setProcessVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-gate-version" label="Expected gate version" type="number" min="1" value={gateVersion} onChange={(event) => setGateVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-authority-version" label="Expected authority version" type="number" min="1" value={authorityVersion} onChange={(event) => setAuthorityVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-authority-epoch" label="Expected authority epoch" type="number" min="1" value={authorityEpoch} onChange={(event) => setAuthorityEpoch(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-new-expiry" label="New authority expiry" type="datetime-local" value={newExpiresAt} onChange={(event) => setNewExpiresAt(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-reason" label="Recovery reason" value={reason} onChange={(event) => setReason(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-evidence" label="Evidence reference" value={evidenceReference} onChange={(event) => setEvidenceReference(event.target.value)} required disabled={saving || Boolean(result)} /></div>
        <h3 className="mt-6 font-semibold">Independent Workflow approval evidence</h3>
        <div className="mt-3 grid gap-4 md:grid-cols-2"><Field id="fpm-recovery-approval-request" label="Approval request ID" value={approvalRequestID} onChange={(event) => setApprovalRequestID(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-decision" label="Decision ID" value={decisionID} onChange={(event) => setDecisionID(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-decision-version" label="Decision version" type="number" min="1" value={decisionVersion} onChange={(event) => setDecisionVersion(event.target.value)} required disabled={saving || Boolean(result)} /><div><label className="label cursor-pointer justify-start gap-2" htmlFor="fpm-recovery-outcome"><span className="font-medium text-base-content">Decision outcome</span></label><Select id="fpm-recovery-outcome" value={decisionOutcome} onChange={(event) => setDecisionOutcome(event.target.value as 'approved' | 'rejected')} disabled={saving || Boolean(result)}><option value="approved">Approved</option><option value="rejected">Rejected</option></Select></div><Field id="fpm-recovery-decision-reason" label="Decision reason" value={decisionReason} onChange={(event) => setDecisionReason(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-policy-version" label="Approval policy version" value={approvalPolicyVersion} onChange={(event) => setApprovalPolicyVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-subject-version" label="Approval subject version" type="number" min="1" value={subjectVersion} onChange={(event) => setSubjectVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-candidate-fingerprint" label="Candidate fingerprint" value={candidateFingerprint} onChange={(event) => setCandidateFingerprint(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-decision-reference" label="Decision reference" value={decisionReference} onChange={(event) => setDecisionReference(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-approver" label="Approver user ID" value={approverUserID} onChange={(event) => setApproverUserID(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-sod-reference" label="Separation-of-duties reference" value={separationReference} onChange={(event) => setSeparationReference(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-recovery-recorded-at" label="Decision recorded at" type="datetime-local" value={recordedAt} onChange={(event) => setRecordedAt(event.target.value)} required disabled={saving || Boolean(result)} /></div>
        {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
        <div className="mt-5 flex flex-wrap items-center gap-3"><Button type="submit" loading={saving} disabled={Boolean(result)}>Take over period control</Button>{result ? <span className="text-sm text-base-content/70">The established result is loaded. Use the same process identity for later reconciliation.</span> : null}</div>
      </form>
    </Panel>
  </section>
}
