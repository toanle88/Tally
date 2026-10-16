import { useRef, useState, type FormEvent } from 'react'

import { fpmStartHardClose } from '@/generated/api/sdk.gen'
import type { EstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'
import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

type ScreenId = 'FPM-WS-01' | 'FPM-SCR-02'

type CommandIdentity = {
  fingerprint: string
  commandId: string
  idempotencyKey: string
  correlationId: string
}

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `ui-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function resultValue<T>(result: EstablishedResult, key: string): T | undefined {
  return result.data[key] as T | undefined
}

function displayValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function errorCode(error: unknown) {
  if (error && typeof error === 'object' && 'code' in error && typeof error.code === 'string') return error.code
  return ''
}

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The authoritative posting barrier could not be established. Review the current period state before retrying.'
}

function stateFor(result: EstablishedResult | null, saving: boolean, blocked: boolean): SemanticState {
  if (saving) return 'pending'
  if (blocked) return 'warning'
  if (result && resultValue<string>(result, 'periodState') === 'Closing') return 'success'
  return 'restricted'
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
  const [saving, setSaving] = useState(false)
  const commandIdentity = useRef<CommandIdentity | null>(null)

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setValidationError('')
    setBlockedReason('')
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

  const periodState = result ? resultValue<string>(result, 'periodState') : 'SoftClosed'
  const gateMode = result ? resultValue<string>(result, 'gateMode') : 'SoftClosePolicy'
  const ownerId = result ? resultValue<string>(result, 'closeRunId') : softCloseRunId
  const closeRunStatus = result ? resultValue<string>(result, 'closeRunStatus') : 'Not yet created'
  const frozenSummary = result ? resultValue<unknown>(result, 'frozenAdmissionSummary') : undefined
  const statusLabel = saving ? 'Awaiting authoritative barrier' : `${periodState} · ${gateMode}`

  return <section aria-labelledby="fpm-hard-close-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">{screen}</p><h2 id="fpm-hard-close-title" className="mt-1 text-2xl font-semibold">Start hard close</h2><p className="mt-2 max-w-4xl text-base-content/75">Create a non-owning close candidate, ask GL for the posting barrier, and transition the period only from the authoritative result. The soft-close owner remains visible until that result is committed.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">A pending or failed barrier request never implies an open period. If the dependency is unavailable, retry with the same command identity or use the authoritative reconciliation workflow.</p>
    <Panel title="Period-control status" description="The display distinguishes the current restrictive state from the GL-confirmed close barrier." state={stateFor(result, saving, Boolean(blockedReason))}>
      <div className="flex flex-wrap items-center gap-3"><StatusBadge state={stateFor(result, saving, Boolean(blockedReason))} label={statusLabel} announce /><span className="text-sm text-base-content/70">Scope {currentScope.id} · period {displayValue(periodId)}</span></div>
      <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Period state</dt><dd>{displayValue(periodState)}</dd></div><div><dt className="text-sm text-base-content/65">Gate mode</dt><dd>{displayValue(gateMode)}</dd></div><div><dt className="text-sm text-base-content/65">Gate version</dt><dd>{displayValue(result ? resultValue<number>(result, 'gateVersion') : expectedGateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Active owner</dt><dd className="break-all font-mono">{displayValue(ownerId)}</dd></div><div><dt className="text-sm text-base-content/65">Close-run status</dt><dd>{displayValue(closeRunStatus)}</dd></div><div><dt className="text-sm text-base-content/65">Barrier ledger position</dt><dd>{displayValue(result ? resultValue<number>(result, 'barrierLedgerPosition') : undefined)}</dd></div></dl>
      {blockedReason ? <p role="alert" className="mt-5 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{blockedReason}</p> : null}
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
    </Panel>
    <Panel title="Hard-close command" description="These values identify the active soft-close epoch and the expected GL gate version. GL remains the owner of posting admission.">
      <form onSubmit={(event) => void submit(event)}>
        <div className="grid gap-4 md:grid-cols-2"><Field id="fpm-hard-close-period-id" label="Fiscal period ID" value={periodId ?? ''} readOnly disabled description="Taken from the selected accounting scope." /><Field id="fpm-hard-close-soft-close-run-id" label="Active soft-close run ID" value={softCloseRunId} onChange={(event) => setSoftCloseRunId(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-control-epoch" label="Soft-close control epoch" type="number" min="1" value={controlEpoch} onChange={(event) => setControlEpoch(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-period-version" label="Expected period version" type="number" min="1" value={periodVersion} onChange={(event) => setPeriodVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-gate-version" label="Expected GL gate version" type="number" min="1" value={expectedGateVersion} onChange={(event) => setExpectedGateVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-hard-close-policy-version" label="Soft-close policy version" value={policyVersion} onChange={(event) => setPolicyVersion(event.target.value)} required disabled={saving || Boolean(result)} /></div>
        {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
        <div className="mt-4 flex flex-wrap items-center gap-3"><Button type="submit" loading={saving} disabled={Boolean(result)}>Start hard close and acquire barrier</Button>{result ? <span className="text-sm text-base-content/70">The authoritative result is loaded. Further close work belongs to the close-run workflow.</span> : null}</div>
      </form>
    </Panel>
    <Panel title="Authoritative barrier evidence" description="Only the committed GL result can move the period from SoftClosed to Closing." state={result ? 'success' : 'info'}>
      {!result ? <p role="note" className="text-sm text-base-content/75">No barrier result is loaded. The candidate close run is not a posting-gate owner.</p> : <><dl className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Result state</dt><dd>{displayValue(resultValue<string>(result, 'validationOutcome'))}</dd></div><div><dt className="text-sm text-base-content/65">Frozen summary</dt><dd>{frozenSummary ? 'Recorded' : 'Missing'}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="break-all font-mono">{displayValue(resultValue<string>(result, 'auditReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Decision reference</dt><dd className="break-all font-mono">{displayValue(resultValue<string>(result, 'decisionReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Period aggregate version</dt><dd>{displayValue(result.aggregateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{resultValue<boolean>(result, 'replayed') ? 'Existing result returned' : 'First establishment'}</dd></div></dl><div className="mt-5 rounded-box border border-base-300 bg-base-200/30 p-4"><h3 className="font-semibold">Frozen admission summary</h3><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-xs text-base-content/75">{frozenSummary ? JSON.stringify(frozenSummary, null, 2) : '—'}</pre></div><p role="note" className="mt-4 text-sm text-base-content/75">The soft-close handoff is pending and the close run now owns the CloseOnly barrier. Ordinary posting must use the authoritative GL gate rules.</p></>}
    </Panel>
  </section>
}
