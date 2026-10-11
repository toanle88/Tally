import { useRef, useState, type FormEvent } from 'react'

import { glAcquirePostingBarrier, glEnterSoftCloseGate, glExitSoftCloseGate, glFinalizePostingGate, glReleasePostingBarrier } from '@/generated/api/sdk.gen'
import type { EstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'
import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

type CommandIdentity = {
  fingerprint: string
  commandId: string
  idempotencyKey: string
  correlationId: string
}

type GateAction = 'enter' | 'exit' | 'acquire' | 'release' | 'finalize'

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `ui-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The posting-gate command could not be established. Review the authoritative result and retry safely.'
}

function resultValue<T>(result: EstablishedResult, key: string): T | undefined {
  return result.data[key] as T | undefined
}

function displayValue(value: unknown) {
  if (value === undefined || value === null || value === '') return '—'
  if (typeof value === 'object') return JSON.stringify(value)
  return String(value)
}

function gateState(result: EstablishedResult): SemanticState {
  const mode = resultValue<string>(result, 'gateMode')
  return mode === 'Open' || mode === 'HardClosed' ? 'success' : 'restricted'
}

function nextActionForResult(result: EstablishedResult): GateAction {
  switch (resultValue<string>(result, 'gateMode')) {
    case 'SoftClosePolicy': return 'exit'
    case 'CloseOnly': return 'finalize'
    default: return 'enter'
  }
}

function actionLabel(action: GateAction) {
  switch (action) {
    case 'enter': return 'Enter soft-close gate'
    case 'exit': return 'Exit soft-close gate'
    case 'acquire': return 'Acquire posting barrier'
    case 'release': return 'Release posting barrier'
    case 'finalize': return 'Finalize posting gate'
  }
}

export function GlPostingGateWorkspace() {
  const { currentScope } = useScopeContext()
  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-03</p><h2 className="mt-1 text-2xl font-semibold">Posting-gate monitor</h2><p className="mt-2">Select an accounting scope before changing the posting barrier.</p></section>
  }

  return <GlPostingGateScopeWorkspace key={currentScope.id} currentScope={currentScope} />
}

function GlPostingGateScopeWorkspace({ currentScope }: { currentScope: AccountingScopeFixture }) {
  const [softCloseRunId, setSoftCloseRunId] = useState('soft-close-run-1')
  const [controlEpoch, setControlEpoch] = useState('1')
  const [expectedVersion, setExpectedVersion] = useState('1')
  const [policyVersion, setPolicyVersion] = useState('soft-close-policy-v1')
  const [restrictions, setRestrictions] = useState('{"ordinaryPosting":"evaluator-controlled"}')
  const [closeRunId, setCloseRunId] = useState('close-run-1')
  const [finalLedgerWatermark, setFinalLedgerWatermark] = useState('0')
  const [result, setResult] = useState<EstablishedResult | null>(null)
  const [selectedAction, setSelectedAction] = useState<GateAction>('enter')
  const [notice, setNotice] = useState('No authoritative gate result has been received for this scope.')
  const [saving, setSaving] = useState(false)
  const [validationError, setValidationError] = useState('')
  const commandIdentity = useRef<CommandIdentity | null>(null)

  const periodId = currentScope.period?.id
  const establishedGateVersion = result ? resultValue<number>(result, 'gateVersion') : undefined
  const gateMode = result ? resultValue<string>(result, 'gateMode') : undefined
  const action = selectedAction
  const expectedVersionValue = action !== 'enter' && typeof establishedGateVersion === 'number' ? String(establishedGateVersion) : expectedVersion
  const barrierActive = gateMode === 'CloseOnly'
  const hardClosed = gateMode === 'HardClosed'

  const submitAction = async (requestedAction: GateAction) => {
    setValidationError('')
    if (saving) return
    if (!periodId) {
      setValidationError('A current fiscal period is required before the posting gate can be changed.')
      return
    }
    const parsedExpectedVersion = Number(expectedVersionValue)
    const parsedEpoch = Number(controlEpoch)
    const parsedWatermark = Number(finalLedgerWatermark)
    const needsSoftClose = requestedAction === 'enter' || requestedAction === 'exit' || requestedAction === 'acquire'
    const needsCloseRun = requestedAction === 'acquire' || requestedAction === 'release' || requestedAction === 'finalize'
    if ((needsSoftClose && !softCloseRunId.trim()) || (needsCloseRun && !closeRunId.trim()) || !Number.isInteger(parsedExpectedVersion) || parsedExpectedVersion < 1) {
      setValidationError('The required process identifiers and expected gate version must be present and valid.')
      return
    }
    if ((requestedAction === 'enter' || requestedAction === 'exit' || requestedAction === 'acquire') && (!Number.isInteger(parsedEpoch) || parsedEpoch < 1)) {
      setValidationError('The soft-close control epoch must be a positive whole number.')
      return
    }
    if ((requestedAction === 'enter' || requestedAction === 'acquire') && !policyVersion.trim()) {
      setValidationError('A soft-close policy version is required for this action.')
      return
    }
    if (requestedAction === 'finalize' && (!Number.isInteger(parsedWatermark) || parsedWatermark < 0)) {
      setValidationError('The final ledger watermark must be a non-negative whole number.')
      return
    }

    let parsedRestrictions: unknown = undefined
    if (requestedAction === 'enter') {
      try {
        parsedRestrictions = JSON.parse(restrictions)
      } catch {
        setValidationError('Restrictions must be valid JSON.')
        return
      }
    }

    const fingerprint = JSON.stringify({ action: requestedAction, scopeId: currentScope.id, periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), restrictions: parsedRestrictions, closeRunId: closeRunId.trim(), finalLedgerWatermark: parsedWatermark, expectedVersion: parsedExpectedVersion })
    const identity = commandIdentity.current?.fingerprint === fingerprint
      ? commandIdentity.current
      : { fingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    commandIdentity.current = identity
    setSelectedAction(requestedAction)
    setSaving(true)
    setNotice(`Submitting the ${actionLabel(requestedAction).toLowerCase()} command and waiting for the authoritative result…`)
    try {
      const base = { commandId: identity.commandId, expectedVersion: parsedExpectedVersion, accountingScopeId: currentScope.id }
      const headers = { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': currentScope.id, 'X-Correlation-Id': identity.correlationId }
      const response = requestedAction === 'enter'
        ? await glEnterSoftCloseGate({ body: { ...base, data: { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), restrictions: parsedRestrictions } }, headers })
        : requestedAction === 'exit'
          ? await glExitSoftCloseGate({ body: { ...base, data: { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch } }, headers })
          : requestedAction === 'acquire'
            ? await glAcquirePostingBarrier({ body: { ...base, data: { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), softCloseControlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), closeRunId: closeRunId.trim(), closeType: 'InitialHardClose' } }, headers })
            : requestedAction === 'release'
              ? await glReleasePostingBarrier({ body: { ...base, data: { fiscalPeriodId: periodId, closeRunId: closeRunId.trim() } }, headers })
              : await glFinalizePostingGate({ body: { ...base, data: { fiscalPeriodId: periodId, closeRunId: closeRunId.trim(), finalLedgerWatermark: parsedWatermark } }, headers })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setNotice('The command returned no established result. No local gate state was changed.')
        return
      }
      setResult(response.data)
      const nextVersion = resultValue<number>(response.data, 'gateVersion')
      if (typeof nextVersion === 'number') setExpectedVersion(String(nextVersion))
      setSelectedAction(nextActionForResult(response.data))
      const resultLabel = requestedAction === 'enter' ? 'Soft-close entry' : requestedAction === 'exit' ? 'Soft-close exit' : actionLabel(requestedAction)
      setNotice(`${resultLabel} established by the GL action result.`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The posting-gate command could not reach the live API. Retry with the same command identity.')
    } finally {
      setSaving(false)
    }
  }

  const ownerId = result ? resultValue<string | null>(result, 'ownerId') : undefined
  const resultRestrictions = result ? resultValue<unknown>(result, 'restrictions') : undefined
  const counters = result ? resultValue<unknown>(result, 'activeAdmissionCounters') : undefined
  const frozenSummary = result ? resultValue<unknown>(result, 'lastFrozenAdmissionSummary') : undefined
  const ordinaryPostingAdmission = result ? resultValue<unknown>(result, 'ordinaryPostingAdmission') : undefined

  return <section aria-labelledby="gl-scr-03-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-03</p><h2 id="gl-scr-03-title" className="mt-1 text-2xl font-semibold">Posting-gate monitor</h2><p className="mt-2 max-w-4xl text-base-content/75">Acquire, release, and finalize the GL-owned posting barrier with matching process identities and expected versions. The monitor displays the latest authoritative action result for the selected accounting scope.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">This screen is action-result driven. It does not issue a posting-gate status read, so it never substitutes a local projection for the GL result. If a finalization response is ambiguous, use the same command identity for retry or the authoritative recovery workflow.</p>
    <Panel title="Posting-gate command" description="Each command carries its process identity, expected gate version, and a stable retry identity.">
      <form onSubmit={(event: FormEvent<HTMLFormElement>) => { event.preventDefault(); void submitAction(action) }}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field id="gl-soft-close-run-id" label="Soft-close run ID" value={softCloseRunId} onChange={(event) => setSoftCloseRunId(event.target.value)} required={action === 'enter' || action === 'exit' || action === 'acquire'} disabled={action === 'release' || action === 'finalize'} />
          <Field id="gl-soft-close-control-epoch" label="Soft-close control epoch" type="number" min="1" value={controlEpoch} onChange={(event) => setControlEpoch(event.target.value)} required={action === 'enter' || action === 'exit' || action === 'acquire'} disabled={action === 'release' || action === 'finalize'} />
          <Field id="gl-posting-close-run-id" label="Close run ID" value={closeRunId} onChange={(event) => setCloseRunId(event.target.value)} required={action === 'acquire' || action === 'release' || action === 'finalize'} disabled={action === 'enter' || action === 'exit'} />
          <Field id="gl-soft-close-expected-version" label="Expected gate version" type="number" min="1" value={expectedVersionValue} onChange={(event) => setExpectedVersion(event.target.value)} required disabled={action !== 'enter' && typeof establishedGateVersion === 'number'} />
          <Field id="gl-soft-close-policy-version" label="Policy version" value={policyVersion} onChange={(event) => setPolicyVersion(event.target.value)} required={action === 'enter' || action === 'acquire'} disabled={action !== 'enter' && action !== 'acquire'} />
          <Field id="gl-posting-final-ledger-watermark" label="Final ledger watermark" type="number" min="0" value={finalLedgerWatermark} onChange={(event) => setFinalLedgerWatermark(event.target.value)} required={action === 'finalize'} disabled={action !== 'finalize'} />
        </div>
        <label className="form-control mt-4 w-full gap-2" htmlFor="gl-soft-close-restrictions"><span className="label font-medium">Restrictions JSON</span><textarea id="gl-soft-close-restrictions" className="textarea textarea-bordered min-h-28 w-full border-2 border-base-300 bg-base-100 p-3 font-mono text-sm focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" value={restrictions} onChange={(event) => setRestrictions(event.target.value)} disabled={action !== 'enter'} /></label>
        {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
        <div className="mt-4 flex flex-wrap gap-3">
          {!hardClosed ? <Button type="submit" loading={saving}>{actionLabel(action)}</Button> : <p role="note" className="self-center text-sm text-base-content/75">The gate is hard closed. No further barrier action is available from this workflow.</p>}
          {gateMode === 'SoftClosePolicy' && !saving ? <Button type="button" variant="secondary" onClick={() => void submitAction('acquire')}>Acquire posting barrier</Button> : null}
          {barrierActive && !saving ? <><Button type="button" variant="secondary" onClick={() => void submitAction('release')}>Release posting barrier</Button>{action !== 'finalize' ? <Button type="button" variant="secondary" onClick={() => void submitAction('finalize')}>Finalize posting gate</Button> : null}</> : null}
          {result ? <span className="self-center text-sm text-base-content/70">Next command uses the latest gate result at version {displayValue(establishedGateVersion)}.</span> : null}
        </div>
      </form>
    </Panel>
    <Panel title="Latest authoritative gate result" description="Ownership, barrier position, counters, frozen summaries, final watermark, audit evidence, and next action come from the latest established result.">
      {!result ? <p role="note" className="text-sm text-base-content/75">No result is loaded. Establish a gate command to inspect the posting barrier.</p> : <>
        <div className="flex flex-wrap items-center gap-3"><StatusBadge state={gateState(result)} label={gateMode ?? 'Unknown'} announce /><span className="text-sm text-base-content/70">Aggregate {result.aggregateId} · version {result.aggregateVersion}</span></div>
        <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Gate mode</dt><dd>{displayValue(gateMode)}</dd></div><div><dt className="text-sm text-base-content/65">Gate version</dt><dd>{displayValue(establishedGateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Owner</dt><dd className="font-mono break-all">{displayValue(ownerId)}</dd></div><div><dt className="text-sm text-base-content/65">Close type</dt><dd>{displayValue(resultValue<string>(result, 'closeType'))}</dd></div><div><dt className="text-sm text-base-content/65">Barrier position</dt><dd>{displayValue(resultValue<number>(result, 'barrierLedgerPosition'))}</dd></div><div><dt className="text-sm text-base-content/65">Final watermark</dt><dd>{displayValue(resultValue<number | null>(result, 'finalLedgerWatermark'))}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="font-mono break-all">{displayValue(resultValue<string>(result, 'auditReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Next action</dt><dd>{displayValue(resultValue<string>(result, 'nextAction'))}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{resultValue<boolean>(result, 'replayed') ? 'Existing result returned' : 'First establishment'}</dd></div></dl>
        <div className="mt-5 grid gap-4 lg:grid-cols-2"><EvidenceBlock label="Prior owner" value={{ type: resultValue<unknown>(result, 'priorOwnerType'), id: resultValue<unknown>(result, 'priorOwnerId'), epoch: resultValue<unknown>(result, 'priorControlEpoch'), mode: resultValue<unknown>(result, 'priorGateMode') }} /><EvidenceBlock label="Restrictions" value={resultRestrictions} /><EvidenceBlock label="Ordinary posting admission" value={ordinaryPostingAdmission} /><EvidenceBlock label="Active admission counters" value={counters} /><EvidenceBlock label="Last frozen admission summary" value={frozenSummary} /><EvidenceBlock label="Finalization fingerprint" value={resultValue<string>(result, 'finalizationCommandFingerprint')} /></div>
      </>}
    </Panel>
  </section>
}

function EvidenceBlock({ label, value }: { label: string; value: unknown }) {
  return <div className="rounded-box border border-base-300 bg-base-200/30 p-4"><h3 className="font-semibold">{label}</h3><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-xs text-base-content/75">{value === undefined || value === null ? '—' : JSON.stringify(value, null, 2)}</pre></div>
}
