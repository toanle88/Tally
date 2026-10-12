import { useRef, useState, type FormEvent } from 'react'

import { glAcquirePostingBarrier, glBeginRecloseGate, glCloseOperationalReopenGate, glCloseScopedReopenGate, glEnterSoftCloseGate, glExitSoftCloseGate, glFinalizePostingGate, glOpenOperationalReopenGate, glOpenScopedReopenGate, glReleasePostingBarrier } from '@/generated/api/sdk.gen'
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

type GateAction = 'enter' | 'exit' | 'acquire' | 'release' | 'finalize' | 'openScopedReopen' | 'closeScopedReopen' | 'openOperationalReopen' | 'closeOperationalReopen' | 'beginReclose'

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
    case 'ScopedReopen': return 'closeScopedReopen'
    case 'OperationalReopen': return 'closeOperationalReopen'
    case 'CloseOnly': return resultValue<string>(result, 'ownerType') === 'ReopenRequest' ? 'beginReclose' : 'finalize'
    case 'HardClosed': return 'openScopedReopen'
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
    case 'openScopedReopen': return 'Open scoped reopen gate'
    case 'closeScopedReopen': return 'Close scoped reopen gate'
    case 'openOperationalReopen': return 'Open operational reopen gate'
    case 'closeOperationalReopen': return 'Close operational reopen gate'
    case 'beginReclose': return 'Begin mandatory reclose'
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
  const [reopenRequestId, setReopenRequestId] = useState('reopen-request-1')
  const [recloseRunId, setRecloseRunId] = useState('reclose-run-1')
  const [authorizationScope, setAuthorizationScope] = useState('{"accountIds":[],"transactionClasses":["manual-journal"]}')
  const [operationalAuthorizationScope, setOperationalAuthorizationScope] = useState('{"actorIds":["00000000-0000-0000-0000-000000000001"],"transactionClasses":["operational-recovery"]}')
  const [authorizationExpiry, setAuthorizationExpiry] = useState('2026-12-31T23:59')
  const [reopenAuthorityEpoch, setReopenAuthorityEpoch] = useState('1')
  const [reopenApproval, setReopenApproval] = useState('{"approvalRequestId":"00000000-0000-0000-0000-000000000001","decisionId":"00000000-0000-0000-0000-000000000002","policyVersion":"reopen-policy-v1","decisionVersion":1,"subjectVersion":1,"candidateFingerprint":"sha256:approved","approverUserId":"00000000-0000-0000-0000-000000000003"}')
  const [noChangeClosePermitted, setNoChangeClosePermitted] = useState(true)
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
  const ownerType = result ? resultValue<string>(result, 'ownerType') : undefined
  const mandatoryReclose = gateMode === 'CloseOnly' && ownerType === 'ReopenRequest'
  const closeType = result ? resultValue<string>(result, 'closeType') : undefined
  const recloseActive = barrierActive && closeType === 'Reclose'
  const actionNeedsReopenRequest = action === 'openScopedReopen' || action === 'closeScopedReopen' || action === 'openOperationalReopen' || action === 'closeOperationalReopen' || action === 'beginReclose'

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
    const needsReopenRequest = requestedAction === 'openScopedReopen' || requestedAction === 'closeScopedReopen' || requestedAction === 'openOperationalReopen' || requestedAction === 'closeOperationalReopen' || requestedAction === 'beginReclose'
    if ((needsSoftClose && !softCloseRunId.trim()) || (needsCloseRun && !closeRunId.trim()) || (needsReopenRequest && !reopenRequestId.trim()) || (requestedAction === 'beginReclose' && !recloseRunId.trim()) || !Number.isInteger(parsedExpectedVersion) || parsedExpectedVersion < 1) {
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

    let parsedReopenScope: unknown = undefined
    let parsedApproval: unknown = undefined
    let parsedReopenExpiry: string | undefined
    if (requestedAction === 'openScopedReopen' || requestedAction === 'openOperationalReopen') {
      try {
        parsedReopenScope = JSON.parse(requestedAction === 'openOperationalReopen' ? operationalAuthorizationScope : authorizationScope)
        parsedApproval = JSON.parse(reopenApproval)
        const expiry = new Date(authorizationExpiry)
        if (Number.isNaN(expiry.getTime())) throw new Error('invalid expiry')
        parsedReopenExpiry = expiry.toISOString()
      } catch {
        setValidationError('The approved reopen scope, approval evidence, and expiry must be valid.')
        return
      }
      const parsedReopenEpoch = Number(reopenAuthorityEpoch)
      if (!Number.isInteger(parsedReopenEpoch) || parsedReopenEpoch < 1) {
        setValidationError('The reopen authority epoch must be a positive whole number.')
        return
      }
    }

    const fingerprint = JSON.stringify({ action: requestedAction, scopeId: currentScope.id, periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), restrictions: parsedRestrictions, closeRunId: closeRunId.trim(), recloseRunId: recloseRunId.trim(), finalLedgerWatermark: parsedWatermark, reopenRequestId: reopenRequestId.trim(), authorizationScope: parsedReopenScope, authorizationExpiry: parsedReopenExpiry, reopenAuthorityEpoch: Number(reopenAuthorityEpoch), approval: parsedApproval, noChangeClosePermitted, expectedVersion: parsedExpectedVersion })
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
      const response = requestedAction === 'openOperationalReopen'
        ? await glOpenOperationalReopenGate({ body: { ...base, data: { fiscalPeriodId: periodId, reopenRequestId: reopenRequestId.trim(), authorizationScope: parsedReopenScope, authorizationExpiry: parsedReopenExpiry, controlAuthorityEpoch: Number(reopenAuthorityEpoch), approval: parsedApproval, noChangeClosePermitted } }, headers })
        : requestedAction === 'closeOperationalReopen'
          ? await glCloseOperationalReopenGate({ body: { ...base, data: { fiscalPeriodId: periodId, reopenRequestId: reopenRequestId.trim() } }, headers })
          : requestedAction === 'beginReclose'
            ? await glBeginRecloseGate({ body: { ...base, data: { fiscalPeriodId: periodId, reopenRequestId: reopenRequestId.trim(), recloseRunId: recloseRunId.trim() } }, headers })
            : requestedAction === 'openScopedReopen'
        ? await glOpenScopedReopenGate({ body: { ...base, data: { fiscalPeriodId: periodId, reopenRequestId: reopenRequestId.trim(), authorizationScope: parsedReopenScope, authorizationExpiry: parsedReopenExpiry, controlAuthorityEpoch: Number(reopenAuthorityEpoch), approval: parsedApproval, noChangeClosePermitted } }, headers })
        : requestedAction === 'closeScopedReopen'
          ? await glCloseScopedReopenGate({ body: { ...base, data: { fiscalPeriodId: periodId, reopenRequestId: reopenRequestId.trim() } }, headers })
          : requestedAction === 'enter'
        ? await glEnterSoftCloseGate({ body: { ...base, data: { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), restrictions: parsedRestrictions } }, headers })
        : requestedAction === 'exit'
          ? await glExitSoftCloseGate({ body: { ...base, data: { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), controlEpoch: parsedEpoch } }, headers })
          : requestedAction === 'acquire'
            ? await glAcquirePostingBarrier({ body: { ...base, data: { fiscalPeriodId: periodId, softCloseRunId: softCloseRunId.trim(), softCloseControlEpoch: parsedEpoch, policyVersion: policyVersion.trim(), closeRunId: closeRunId.trim(), closeType: 'InitialHardClose' } }, headers })
            : requestedAction === 'release'
              ? await glReleasePostingBarrier({ body: { ...base, data: { fiscalPeriodId: periodId, closeRunId: closeRunId.trim() } }, headers })
              : requestedAction === 'finalize'
                ? await glFinalizePostingGate({ body: { ...base, data: { fiscalPeriodId: periodId, closeRunId: closeRunId.trim(), finalLedgerWatermark: parsedWatermark } }, headers })
                : { data: undefined, error: { detail: 'The selected gate action is unavailable.' } }
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
  const reopenAdmission = result ? resultValue<unknown>(result, 'reopenPostingAdmission') : undefined

  return <section aria-labelledby="gl-scr-03-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-03</p><h2 id="gl-scr-03-title" className="mt-1 text-2xl font-semibold">Posting-gate monitor</h2><p className="mt-2 max-w-4xl text-base-content/75">Acquire, release, and finalize the GL-owned posting barrier with matching process identities and expected versions. The monitor displays the latest authoritative action result for the selected accounting scope.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">This screen is action-result driven. It does not issue a posting-gate status read, so it never substitutes a local projection for the GL result. If a finalization response is ambiguous, use the same command identity for retry or the authoritative recovery workflow.</p>
    <Panel title="Posting-gate command" description="Each command carries its process identity, expected gate version, and a stable retry identity.">
      <form onSubmit={(event: FormEvent<HTMLFormElement>) => { event.preventDefault(); void submitAction(action) }}>
        <div className="grid gap-4 md:grid-cols-2">
          <Field id="gl-soft-close-run-id" label="Soft-close run ID" value={softCloseRunId} onChange={(event) => setSoftCloseRunId(event.target.value)} required={action === 'enter' || action === 'exit' || action === 'acquire'} disabled={action === 'release' || action === 'finalize'} />
          <Field id="gl-soft-close-control-epoch" label="Soft-close control epoch" type="number" min="1" value={controlEpoch} onChange={(event) => setControlEpoch(event.target.value)} required={action === 'enter' || action === 'exit' || action === 'acquire'} disabled={action === 'release' || action === 'finalize'} />
          <Field id="gl-posting-close-run-id" label="Close run ID" value={closeRunId} onChange={(event) => setCloseRunId(event.target.value)} required={action === 'acquire' || action === 'release' || action === 'finalize'} disabled={action === 'enter' || action === 'exit' || action === 'openScopedReopen' || action === 'closeScopedReopen' || action === 'openOperationalReopen' || action === 'closeOperationalReopen' || action === 'beginReclose'} />
          <Field id="gl-posting-reclose-run-id" label="Reclose run ID" value={recloseRunId} onChange={(event) => setRecloseRunId(event.target.value)} required={action === 'beginReclose'} disabled={action !== 'beginReclose'} description="The new exclusive owner created by the fiscal-period reclose workflow." />
          <Field id="gl-soft-close-expected-version" label="Expected gate version" type="number" min="1" value={expectedVersionValue} onChange={(event) => setExpectedVersion(event.target.value)} required disabled={action !== 'enter' && typeof establishedGateVersion === 'number'} />
          <Field id="gl-soft-close-policy-version" label="Policy version" value={policyVersion} onChange={(event) => setPolicyVersion(event.target.value)} required={action === 'enter' || action === 'acquire'} disabled={action !== 'enter' && action !== 'acquire'} />
          <Field id="gl-posting-final-ledger-watermark" label="Final ledger watermark" type="number" min="0" value={finalLedgerWatermark} onChange={(event) => setFinalLedgerWatermark(event.target.value)} required={action === 'finalize'} disabled={action !== 'finalize'} />
          <Field id="gl-reopen-request-id" label="Reopen request ID" value={reopenRequestId} onChange={(event) => setReopenRequestId(event.target.value)} required={actionNeedsReopenRequest} disabled={!actionNeedsReopenRequest} description="The approved request remains the sole reopen owner until closure or reclose handoff." />
          <Field id="gl-reopen-authorization-expiry" label="Authorization expiry" type="datetime-local" value={authorizationExpiry} onChange={(event) => setAuthorizationExpiry(event.target.value)} required={action === 'openScopedReopen' || action === 'openOperationalReopen'} disabled={action !== 'openScopedReopen' && action !== 'openOperationalReopen'} />
          <Field id="gl-reopen-authority-epoch" label="Control authority epoch" type="number" min="1" value={reopenAuthorityEpoch} onChange={(event) => setReopenAuthorityEpoch(event.target.value)} required={action === 'openScopedReopen' || action === 'openOperationalReopen'} disabled={action !== 'openScopedReopen' && action !== 'openOperationalReopen'} />
        </div>
        <label className="form-control mt-4 w-full gap-2" htmlFor="gl-soft-close-restrictions"><span className="label font-medium">Restrictions JSON</span><textarea id="gl-soft-close-restrictions" className="textarea textarea-bordered min-h-28 w-full border-2 border-base-300 bg-base-100 p-3 font-mono text-sm focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" value={restrictions} onChange={(event) => setRestrictions(event.target.value)} disabled={action !== 'enter'} /></label>
        <div className="mt-4 grid gap-4 md:grid-cols-2">
          <label className="form-control w-full gap-2" htmlFor="gl-reopen-authorization-scope"><span className="label font-medium">Approved scoped-reopen JSON</span><textarea id="gl-reopen-authorization-scope" className="textarea textarea-bordered min-h-28 w-full border-2 border-base-300 bg-base-100 p-3 font-mono text-sm focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" value={authorizationScope} onChange={(event) => setAuthorizationScope(event.target.value)} disabled={action !== 'openScopedReopen'} /></label>
          <label className="form-control w-full gap-2" htmlFor="gl-operational-reopen-authorization-scope"><span className="label font-medium">Approved operational scope JSON</span><textarea id="gl-operational-reopen-authorization-scope" className="textarea textarea-bordered min-h-28 w-full border-2 border-base-300 bg-base-100 p-3 font-mono text-sm focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" value={operationalAuthorizationScope} onChange={(event) => setOperationalAuthorizationScope(event.target.value)} disabled={action !== 'openOperationalReopen'} /><span className="text-xs text-base-content/65">Both actorIds and transactionClasses are required and matched together at GL admission.</span></label>
          <label className="form-control w-full gap-2" htmlFor="gl-reopen-approval"><span className="label font-medium">Approval evidence JSON</span><textarea id="gl-reopen-approval" className="textarea textarea-bordered min-h-28 w-full border-2 border-base-300 bg-base-100 p-3 font-mono text-sm focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" value={reopenApproval} onChange={(event) => setReopenApproval(event.target.value)} disabled={action !== 'openScopedReopen' && action !== 'openOperationalReopen'} /></label>
        </div>
        {action === 'openScopedReopen' || action === 'openOperationalReopen' ? <label className="mt-4 flex items-center gap-3 text-sm" htmlFor="gl-reopen-no-change-policy"><input id="gl-reopen-no-change-policy" type="checkbox" className="checkbox checkbox-primary" checked={noChangeClosePermitted} onChange={(event) => setNoChangeClosePermitted(event.target.checked)} />The approved handoff permits an audited no-change closure.</label> : null}
        {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
        <div className="mt-4 flex flex-wrap gap-3">
          {!hardClosed || action === 'openScopedReopen' || action === 'openOperationalReopen' ? <Button type="submit" loading={saving}>{actionLabel(action)}</Button> : <p role="note" className="self-center text-sm text-base-content/75">The gate is hard closed. Open a scoped or operational reopen window only with approved request, scope, expiry, and authority evidence.</p>}
          {hardClosed && !saving ? <Button type="button" variant="secondary" onClick={() => void submitAction('openOperationalReopen')}>Open operational reopen gate</Button> : null}
          {gateMode === 'SoftClosePolicy' && !saving ? <Button type="button" variant="secondary" onClick={() => void submitAction('acquire')}>Acquire posting barrier</Button> : null}
          {barrierActive && !mandatoryReclose && !recloseActive && !saving ? <><Button type="button" variant="secondary" onClick={() => void submitAction('release')}>Release posting barrier</Button>{action !== 'finalize' ? <Button type="button" variant="secondary" onClick={() => void submitAction('finalize')}>Finalize posting gate</Button> : null}</> : null}
          {recloseActive && !saving && action !== 'finalize' ? <Button type="button" variant="secondary" onClick={() => void submitAction('finalize')}>Finalize reclose gate</Button> : null}
          {mandatoryReclose ? <p role="note" className="self-center text-sm font-semibold text-warning">Corrections were admitted. Mandatory reclose handoff is required before finalization.</p> : null}
          {recloseActive ? <p role="note" className="self-center text-sm font-semibold text-warning">Reclose owns the barrier. Release is unavailable until the reclose run finalizes.</p> : null}
          {result ? <span className="self-center text-sm text-base-content/70">Next command uses the latest gate result at version {displayValue(establishedGateVersion)}.</span> : null}
        </div>
      </form>
    </Panel>
    <Panel title="Latest authoritative gate result" description="Ownership, barrier position, counters, frozen summaries, final watermark, audit evidence, and next action come from the latest established result.">
      {!result ? <p role="note" className="text-sm text-base-content/75">No result is loaded. Establish a gate command to inspect the posting barrier.</p> : <>
        <div className="flex flex-wrap items-center gap-3"><StatusBadge state={gateState(result)} label={gateMode ?? 'Unknown'} announce /><span className="text-sm text-base-content/70">Aggregate {result.aggregateId} · version {result.aggregateVersion}</span></div>
        <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Gate mode</dt><dd>{displayValue(gateMode)}</dd></div><div><dt className="text-sm text-base-content/65">Gate version</dt><dd>{displayValue(establishedGateVersion)}</dd></div><div><dt className="text-sm text-base-content/65">Owner</dt><dd className="font-mono break-all">{displayValue(ownerId)}</dd></div><div><dt className="text-sm text-base-content/65">Close type</dt><dd>{displayValue(resultValue<string>(result, 'closeType'))}</dd></div><div><dt className="text-sm text-base-content/65">Barrier position</dt><dd>{displayValue(resultValue<number>(result, 'barrierLedgerPosition'))}</dd></div><div><dt className="text-sm text-base-content/65">Final watermark</dt><dd>{displayValue(resultValue<number | null>(result, 'finalLedgerWatermark'))}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="font-mono break-all">{displayValue(resultValue<string>(result, 'auditReference'))}</dd></div><div><dt className="text-sm text-base-content/65">Next action</dt><dd>{displayValue(resultValue<string>(result, 'nextAction'))}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{resultValue<boolean>(result, 'replayed') ? 'Existing result returned' : 'First establishment'}</dd></div></dl>
        <div className="mt-5 grid gap-4 lg:grid-cols-2"><EvidenceBlock label="Prior owner" value={{ type: resultValue<unknown>(result, 'priorOwnerType'), id: resultValue<unknown>(result, 'priorOwnerId'), epoch: resultValue<unknown>(result, 'priorControlEpoch'), mode: resultValue<unknown>(result, 'priorGateMode') }} /><EvidenceBlock label="Restrictions" value={resultRestrictions} /><EvidenceBlock label="Reopen authorization by mode" value={{ gateMode, requestId: resultValue<unknown>(result, 'reopenRequestId'), scope: resultValue<unknown>(result, 'authorizationScope'), expiry: resultValue<unknown>(result, 'authorizationExpiry'), authorityEpoch: resultValue<unknown>(result, 'controlAuthorityEpoch') }} /><EvidenceBlock label="Ordinary posting admission" value={ordinaryPostingAdmission} /><EvidenceBlock label="Reopen admission by mode" value={reopenAdmission} /><EvidenceBlock label="Active admission counters" value={counters} /><EvidenceBlock label="Last frozen admission summary" value={frozenSummary} /><EvidenceBlock label="Finalization fingerprint" value={resultValue<string>(result, 'finalizationCommandFingerprint')} /></div>
      </>}
    </Panel>
  </section>
}

function EvidenceBlock({ label, value }: { label: string; value: unknown }) {
  return <div className="rounded-box border border-base-300 bg-base-200/30 p-4"><h3 className="font-semibold">{label}</h3><pre className="mt-2 max-h-48 overflow-auto whitespace-pre-wrap break-words text-xs text-base-content/75">{value === undefined || value === null ? '—' : JSON.stringify(value, null, 2)}</pre></div>
}
