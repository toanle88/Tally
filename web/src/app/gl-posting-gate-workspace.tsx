import { useEffect, useRef, useState, type FormEvent } from 'react'

import { glEnterSoftCloseGate, glExitSoftCloseGate } from '@/generated/api/sdk.gen'
import type { EstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'
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

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The soft-close gate command could not be established. Review the current scope and retry safely.'
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
  return resultValue<string>(result, 'gateMode') === 'Open' ? 'success' : 'restricted'
}

export function GlPostingGateWorkspace() {
  const { currentScope } = useScopeContext()
  const [softCloseRunId, setSoftCloseRunId] = useState('soft-close-run-1')
  const [controlEpoch, setControlEpoch] = useState('1')
  const [expectedVersion, setExpectedVersion] = useState('1')
  const [policyVersion, setPolicyVersion] = useState('soft-close-policy-v1')
  const [restrictions, setRestrictions] = useState('{"ordinaryPosting":"evaluator-controlled"}')
  const [result, setResult] = useState<EstablishedResult | null>(null)
  const [notice, setNotice] = useState('No authoritative gate result has been received for this scope.')
  const [saving, setSaving] = useState(false)
  const [validationError, setValidationError] = useState('')
  const commandIdentity = useRef<CommandIdentity | null>(null)

  useEffect(() => {
    setResult(null)
    setExpectedVersion('1')
    setNotice('No authoritative gate result has been received for this scope.')
    commandIdentity.current = null
  }, [currentScope?.id])

  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-03</p><h2 className="mt-1 text-2xl font-semibold">Posting-gate monitor</h2><p className="mt-2">Select an accounting scope before entering or exiting a soft-close gate.</p></section>
  }

  const periodId = currentScope.period?.id
  const establishedGateVersion = resultValue<number>(result ?? ({ data: {} } as EstablishedResult), 'gateVersion')
  const action = resultValue<string>(result ?? ({ data: {} } as EstablishedResult), 'gateMode') === 'SoftClosePolicy' ? 'exit' : 'enter'

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setValidationError('')
    if (!periodId) {
      setValidationError('A current fiscal period is required before a soft-close gate can be changed.')
      return
    }
    const parsedEpoch = Number(controlEpoch)
    const parsedExpectedVersion = action === 'exit' && typeof establishedGateVersion === 'number' ? establishedGateVersion : Number(expectedVersion)
    if (!softCloseRunId.trim() || !Number.isInteger(parsedEpoch) || parsedEpoch < 1 || !Number.isInteger(parsedExpectedVersion) || parsedExpectedVersion < 1) {
      setValidationError('The soft-close run, control epoch, and expected gate version are required positive values.')
      return
    }
    let parsedRestrictions: unknown = undefined
    if (action === 'enter') {
      try {
        parsedRestrictions = JSON.parse(restrictions)
      } catch {
        setValidationError('Restrictions must be valid JSON.')
        return
      }
      if (!policyVersion.trim()) {
        setValidationError('A soft-close policy version is required for entry.')
        return
      }
    }

    const fingerprint = JSON.stringify({ action, scopeId: currentScope.id, periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, expectedVersion: parsedExpectedVersion, policyVersion: policyVersion.trim(), restrictions: parsedRestrictions })
    const identity = commandIdentity.current?.fingerprint === fingerprint
      ? commandIdentity.current
      : { fingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    commandIdentity.current = identity
    setSaving(true)
    setNotice(`Submitting the ${action} soft-close gate command and waiting for the authoritative result…`)
    try {
      const body = {
        commandId: identity.commandId,
        expectedVersion: parsedExpectedVersion,
        accountingScopeId: currentScope.id,
        data: action === 'enter'
          ? { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), restrictions: parsedRestrictions }
          : { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch },
      }
      const response = action === 'enter'
        ? await glEnterSoftCloseGate({ body, headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': currentScope.id, 'X-Correlation-Id': identity.correlationId } })
        : await glExitSoftCloseGate({ body, headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': currentScope.id, 'X-Correlation-Id': identity.correlationId } })
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
      setNotice(`${action === 'enter' ? 'Soft-close entry' : 'Soft-close exit'} established by the GL action result.`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The soft-close gate command could not reach the live API.')
    } finally {
      setSaving(false)
    }
  }

  const gateMode = result ? resultValue<string>(result, 'gateMode') : undefined
  const ownerId = result ? resultValue<string | null>(result, 'ownerId') : undefined
  const resultRestrictions = result ? resultValue<unknown>(result, 'restrictions') : undefined
  const counters = result ? resultValue<unknown>(result, 'activeAdmissionCounters') : undefined
  const frozenSummary = result ? resultValue<unknown>(result, 'lastFrozenAdmissionSummary') : undefined
  const ordinaryPostingAdmission = result ? resultValue<unknown>(result, 'ordinaryPostingAdmission') : undefined

  return <section aria-labelledby="gl-scr-03-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-03</p><h2 id="gl-scr-03-title" className="mt-1 text-2xl font-semibold">Posting-gate monitor</h2><p className="mt-2 max-w-4xl text-base-content/75">Enter or exit the GL-owned soft-close gate with the matching run and control epoch. The monitor displays the latest authoritative action result for the selected accounting scope.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">This screen is action-result driven. It does not issue a posting-gate status read, so it never substitutes a local projection for the GL result.</p>
    <Panel title="Soft-close gate command" description="Restrictions are carried as an opaque canonical policy payload and evaluated by the owning GL policy boundary.">
      <form onSubmit={(event) => void submit(event)}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field id="gl-soft-close-run-id" label="Soft-close run ID" value={softCloseRunId} onChange={(event) => setSoftCloseRunId(event.target.value)} required />
          <Field id="gl-soft-close-control-epoch" label="Control epoch" type="number" min="1" value={controlEpoch} onChange={(event) => setControlEpoch(event.target.value)} required />
          <Field id="gl-soft-close-expected-version" label="Expected gate version" type="number" min="1" value={action === 'exit' && typeof establishedGateVersion === 'number' ? establishedGateVersion : expectedVersion} onChange={(event) => setExpectedVersion(event.target.value)} required />
          <Field id="gl-soft-close-policy-version" label="Policy version" value={policyVersion} onChange={(event) => setPolicyVersion(event.target.value)} required={action === 'enter'} />
        </div>
        <label className="form-control mt-4 w-full gap-2" htmlFor="gl-soft-close-restrictions"><span className="label font-medium">Restrictions JSON</span><textarea id="gl-soft-close-restrictions" className="textarea textarea-bordered min-h-28 w-full border-2 border-base-300 bg-base-100 p-3 font-mono text-sm focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" value={restrictions} onChange={(event) => setRestrictions(event.target.value)} disabled={action === 'exit'} /></label>
        {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
        <div className="mt-4 flex flex-wrap gap-3"><Button type="submit" loading={saving}>{action === 'enter' ? 'Enter soft-close gate' : 'Exit soft-close gate'}</Button>{result ? <span className="self-center text-sm text-base-content/70">Next command uses the latest gate result at version {displayValue(establishedGateVersion)}.</span> : null}</div>
      </form>
    </Panel>
    <Panel title="Latest authoritative gate result" description="Counters, restrictions, ownership, frozen summaries, audit evidence, and next action come from the latest established result.">
      {!result ? <p role="note" className="text-sm text-base-content/75">No result is loaded. Establish an entry or exit command to inspect the gate boundary.</p> : <>
        <div className="flex flex-wrap items-center gap-3"><StatusBadge state={gateState(result)} label={gateMode ?? 'Unknown'} announce /><span className="text-sm text-base-content/70">Aggregate {result.aggregateId} · version {result.aggregateVersion}</span></div>
        <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Gate mode</dt><dd>{displayValue(gateMode)}</dd></div><div><dt className="text-sm text-base-content/65">Gate version</dt><dd>{displayValue(establishedGateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Owner</dt><dd className="font-mono break-all">{displayValue(ownerId)}</dd></div><div><dt className="text-sm text-base-content/65">Control epoch</dt><dd>{displayValue(resultValue<number>(result, 'controlEpoch'))}</dd></div><div><dt className="text-sm text-base-content/65">Policy</dt><dd>{displayValue(resultValue<string>(result, 'policyVersion'))}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="font-mono break-all">{displayValue(resultValue<string>(result, 'auditReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Next action</dt><dd>{displayValue(resultValue<string>(result, 'nextAction'))}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{resultValue<boolean>(result, 'replayed') ? 'Existing result returned' : 'First establishment'}</dd></div></dl>
        <div className="mt-5 grid gap-4 lg:grid-cols-2"><EvidenceBlock label="Restrictions" value={resultRestrictions} /><EvidenceBlock label="Ordinary posting admission" value={ordinaryPostingAdmission} /><EvidenceBlock label="Active admission counters" value={counters} /><EvidenceBlock label="Last frozen admission summary" value={frozenSummary} /></div>
      </>}
    </Panel>
  </section>
}

function EvidenceBlock({ label, value }: { label: string; value: unknown }) {
  return <div className="rounded-box border border-base-300 bg-base-200/30 p-4"><h3 className="font-semibold">{label}</h3><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-xs text-base-content/75">{value === undefined || value === null ? '—' : JSON.stringify(value, null, 2)}</pre></div>
}
