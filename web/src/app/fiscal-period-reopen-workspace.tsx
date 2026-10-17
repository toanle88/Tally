import { useState, type FormEvent } from 'react'

import { fpmApplyReopenApprovalDecision, fpmRequestReopen, fpmStartReclose } from '@/generated/api/sdk.gen'
import type { EstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `ui-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function resultValue<T>(result: EstablishedResult | null, key: string): T | undefined {
  return result ? result.data[key] as T | undefined : undefined
}

function errorCode(error: unknown) {
  return error && typeof error === 'object' && 'code' in error && typeof error.code === 'string' ? error.code : ''
}

function errorDetail(error: unknown) {
  return error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string'
    ? error.detail
    : 'The reopen outcome could not be established. Refresh the authoritative period and gate state before retrying.'
}

function semanticState(result: EstablishedResult | null, saving: boolean, blocked: boolean): SemanticState {
  if (saving) return 'pending'
  if (blocked) return 'warning'
  const status = resultValue<string>(result, 'reopenStatus')
  const periodState = resultValue<string>(result, 'periodState')
  if (status === 'Rejected') return 'error'
  if (status === 'Active' && periodState === 'Reopening') return 'success'
  if (status === 'CompletedNoChange' || status === 'Completed') return 'success'
  if (status === 'PendingApproval' || status === 'Approved' || status === 'GateOpening') return 'pending'
  return 'restricted'
}

function listValue(value: string) {
  return value.split(/[\s,]+/).map((item) => item.trim()).filter(Boolean)
}

function localDateTimeToRFC3339(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? '' : parsed.toISOString()
}

function localDateTimeValue(date: Date) {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function initialAuthorizationExpiry() {
  return localDateTimeValue(new Date(Date.now() + 60 * 60 * 1000))
}

function initialRecordedAt() {
  const now = new Date()
  now.setSeconds(0, 0)
  return localDateTimeValue(now)
}

function jsonValue(value: string) {
  return JSON.parse(value) as unknown
}

function commandHeaders(scopeID: string, idempotencyKey: string, correlationID: string) {
  return { 'Idempotency-Key': idempotencyKey, 'X-Accounting-Scope-Id': scopeID, 'X-Correlation-Id': correlationID }
}

export function FiscalPeriodReopenWorkspace() {
  const { currentScope } = useScopeContext()
  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">FPM-SCR-03</p><h2 className="mt-1 text-2xl font-semibold">Request and authorize reopen</h2><p className="mt-2">Select an accounting scope before requesting a reopen.</p></section>
  }
  return <FiscalPeriodReopenScopeWorkspace key={currentScope.id} currentScope={currentScope} />
}

function FiscalPeriodReopenScopeWorkspace({ currentScope }: { currentScope: AccountingScopeFixture }) {
  const periodID = currentScope.period?.id ?? ''
  const [mode, setMode] = useState<'ScopedCorrection' | 'Operational'>('ScopedCorrection')
  const [reason, setReason] = useState('Correct a bounded post-close classification error.')
  const [accountIDs, setAccountIDs] = useState('')
  const [transactionClasses, setTransactionClasses] = useState('ReopenCorrection')
  const [actorIDs, setActorIDs] = useState('')
  const [proposedCorrections, setProposedCorrections] = useState('{"replacement":"classification"}')
  const [impact, setImpact] = useState('{"materiality":"low"}')
  const [authorizationExpiry, setAuthorizationExpiry] = useState(initialAuthorizationExpiry)
  const [periodVersion, setPeriodVersion] = useState('1')
  const [gateVersion, setGateVersion] = useState('1')
  const [authorityEpoch, setAuthorityEpoch] = useState('1')
  const [result, setResult] = useState<EstablishedResult | null>(null)
  const [approvalRequestID, setApprovalRequestID] = useState('')
  const [decisionID, setDecisionID] = useState('')
  const [decisionVersion, setDecisionVersion] = useState('1')
  const [decisionOutcome, setDecisionOutcome] = useState<'approved' | 'rejected'>('approved')
  const [decisionReason, setDecisionReason] = useState('Independent review completed.')
  const [policyVersion, setPolicyVersion] = useState('reopen-policy-v1')
  const [subjectVersion, setSubjectVersion] = useState('1')
  const [candidateFingerprint, setCandidateFingerprint] = useState('')
  const [decisionReference, setDecisionReference] = useState('')
  const [approverUserID, setApproverUserID] = useState('')
  const [separationReference, setSeparationReference] = useState('sod-reopen-review-1')
  const [recordedAt, setRecordedAt] = useState(initialRecordedAt)
  const [expectedRequestVersion, setExpectedRequestVersion] = useState('1')
  const [saving, setSaving] = useState(false)
  const [notice, setNotice] = useState('A pending request does not change the period state and cannot admit postings.')
  const [blockedReason, setBlockedReason] = useState('')
  const [lastErrorCode, setLastErrorCode] = useState('')

  const submitRequest = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (saving) return
    setBlockedReason('')
    setLastErrorCode('')
    if (!periodID || !reason.trim() || !authorizationExpiry || Number(periodVersion) < 1 || Number(gateVersion) < 1 || Number(authorityEpoch) < 1) {
      setBlockedReason('The period, reason, expiry, and expected versions are required.')
      return
    }
    let corrections: unknown
    let impactValue: unknown
    try {
      corrections = jsonValue(proposedCorrections)
      impactValue = jsonValue(impact)
    } catch {
      setBlockedReason('Proposed corrections and impact must be valid JSON documents.')
      return
    }
    const commandID = newIdentifier()
    const correlationID = newIdentifier()
    setSaving(true)
    setNotice('Submitting the reopen request. The period remains HardClosed until current Workflow approval and the authoritative GL gate result are applied…')
    try {
      const response = await fpmRequestReopen({
        body: {
          commandId: commandID,
          expectedVersion: Number(periodVersion),
          accountingScopeId: currentScope.id,
          data: {
            fiscalPeriodId: periodID,
            reopenMode: mode,
            reason: reason.trim(),
            accountIds: listValue(accountIDs),
            transactionClasses: listValue(transactionClasses),
            actorIds: listValue(actorIDs),
            proposedCorrections: corrections,
            impact: impactValue,
            authorizationExpiry: localDateTimeToRFC3339(authorizationExpiry),
            expectedGateVersion: Number(gateVersion),
            controlAuthorityEpoch: Number(authorityEpoch),
          },
        },
        headers: commandHeaders(currentScope.id, `fpm-reopen-request-${commandID}`, correlationID),
      })
      if (response.error) {
        setLastErrorCode(errorCode(response.error))
        setBlockedReason(errorCode(response.error) === 'DEPENDENCY_UNAVAILABLE' ? 'A required dependency is unavailable. The period remains restrictive; retry with the same request identity.' : 'The reopen request was rejected. The period remains HardClosed and no GL gate was opened.')
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setBlockedReason('No established result was returned. No local period transition is assumed.')
        return
      }
      setResult(response.data)
      setPeriodVersion(String(resultValue<number>(response.data, 'periodAggregateVersion') ?? Number(periodVersion)))
      setGateVersion(String(resultValue<number>(response.data, 'gateVersion') ?? Number(gateVersion)))
      setApprovalRequestID(resultValue<string>(response.data, 'approvalRequestId') ?? '')
      const reopenRequest = resultValue<Record<string, unknown>>(response.data, 'reopenRequest')
      setCandidateFingerprint(String(reopenRequest?.requestFingerprint ?? ''))
      setNotice('Reopen request recorded as PendingApproval. Obtain an independent current Workflow decision before applying it.')
    } catch (error) {
      setBlockedReason('The reopen request could not be confirmed. The period remains HardClosed until the authoritative result is reconciled.')
      setNotice(error instanceof Error ? error.message : 'The reopen request could not reach the API. Retry with the same command identity.')
    } finally {
      setSaving(false)
    }
  }

  const applyApproval = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (saving) return
    setBlockedReason('')
    setLastErrorCode('')
    const requestID = resultValue<string>(result, 'reopenRequestId')
    if (!requestID || !approvalRequestID || !decisionID || !decisionReference || !approverUserID || !candidateFingerprint || !separationReference || !decisionReason || Number(expectedRequestVersion) < 1) {
      setBlockedReason('Current reopen request and immutable Workflow decision evidence are required.')
      return
    }
    const commandID = newIdentifier()
    const correlationID = newIdentifier()
    setSaving(true)
    setNotice('Applying current Workflow approval evidence. GL will be asked to open a bounded gate only after FPM revalidation succeeds…')
    try {
      const response = await fpmApplyReopenApprovalDecision({
        body: {
          commandId: commandID,
          expectedVersion: Number(periodVersion),
          accountingScopeId: currentScope.id,
          data: {
            fiscalPeriodId: periodID,
            reopenRequestId: requestID,
            approvalRequestId: approvalRequestID,
            decisionId: decisionID,
            decisionVersion: Number(decisionVersion),
            outcome: decisionOutcome,
            decisionReason: decisionReason.trim(),
            policyVersion: policyVersion.trim(),
            subjectVersion: Number(subjectVersion),
            candidateFingerprint: candidateFingerprint.trim(),
            decisionReference,
            approverUserId: approverUserID,
            separationOfDutiesReference: separationReference.trim(),
            recordedAt: localDateTimeToRFC3339(recordedAt),
            expectedReopenRequestVersion: Number(expectedRequestVersion),
            expectedGateVersion: Number(gateVersion),
          },
        },
        headers: commandHeaders(currentScope.id, `fpm-reopen-approval-${commandID}`, correlationID),
      })
      if (response.error) {
        setLastErrorCode(errorCode(response.error))
        setBlockedReason(errorCode(response.error) === 'DEPENDENCY_UNAVAILABLE' ? 'Workflow or GL is unavailable. The period remains restrictive and the approval can be retried safely.' : 'Approval evidence was rejected or became stale. Refresh the current request before retrying.')
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setBlockedReason('No established approval result was returned. No gate transition is assumed.')
        return
      }
      setResult(response.data)
      setPeriodVersion(String(resultValue<number>(response.data, 'periodAggregateVersion') ?? Number(periodVersion)))
      setGateVersion(String(resultValue<number>(response.data, 'gateVersion') ?? Number(gateVersion)))
      const reopenRequest = resultValue<Record<string, unknown>>(response.data, 'reopenRequest')
      setExpectedRequestVersion(String(Number(reopenRequest?.aggregateVersion ?? Number(expectedRequestVersion))))
      const status = resultValue<string>(response.data, 'reopenStatus') ?? 'Unknown'
      setNotice(status === 'Active' ? 'The approved reopen is active. Only the recorded scope may be admitted before the recorded expiry; mandatory reclose remains a later workflow.' : `The reopen request is now ${status}.`)
    } catch (error) {
      setBlockedReason('The approval outcome could not be confirmed. The period remains restrictive until Workflow and GL state are reconciled.')
      setNotice(error instanceof Error ? error.message : 'The approval command could not reach the API. Retry with the same command identity.')
    } finally {
      setSaving(false)
    }
  }

  const startReclose = async () => {
    if (saving) return
    setBlockedReason('')
    setLastErrorCode('')
    const requestID = resultValue<string>(result, 'reopenRequestId')
    const reopenRequest = resultValue<Record<string, unknown>>(result, 'reopenRequest')
    const expectedPeriod = Number(periodVersion)
    const expectedRequest = Number(reopenRequest?.aggregateVersion ?? Number(expectedRequestVersion))
    const expectedGate = Number(resultValue<number>(result, 'gateVersion') ?? Number(gateVersion))
    if (!periodID || !requestID || expectedPeriod < 1 || expectedRequest < 1 || expectedGate < 1) {
      setBlockedReason('The current period, approved reopen request, and authoritative versions are required before closure.')
      return
    }
    const commandID = newIdentifier()
    const correlationID = newIdentifier()
    setSaving(true)
    setNotice('Closing the approved reopen gate. FPM will use the authoritative GL admission summary to determine no-change completion or mandatory reclose…')
    try {
      const response = await fpmStartReclose({
        body: {
          commandId: commandID,
          expectedVersion: expectedPeriod,
          accountingScopeId: currentScope.id,
          data: {
            fiscalPeriodId: periodID,
            reopenRequestId: requestID,
            expectedPeriodVersion: expectedPeriod,
            expectedReopenRequestVersion: expectedRequest,
            expectedGateVersion: expectedGate,
          },
        },
        headers: commandHeaders(currentScope.id, `fpm-start-reclose-${commandID}`, correlationID),
      })
      if (response.error) {
        setLastErrorCode(errorCode(response.error))
        setBlockedReason(errorCode(response.error) === 'DEPENDENCY_UNAVAILABLE' ? 'GL or another required dependency is unavailable. The current gate remains restrictive; reconcile authoritative state before retrying.' : 'The reopen closure was rejected or became stale. Refresh the current period and gate state before retrying.')
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setBlockedReason('No established closure result was returned. No local no-change or reclose outcome is assumed.')
        return
      }
      setResult(response.data)
      setPeriodVersion(String(resultValue<number>(response.data, 'periodAggregateVersion') ?? expectedPeriod))
      setGateVersion(String(resultValue<number>(response.data, 'gateVersion') ?? expectedGate))
      const updatedRequest = resultValue<Record<string, unknown>>(response.data, 'reopenRequest')
      setExpectedRequestVersion(String(Number(updatedRequest?.aggregateVersion ?? expectedRequest)))
      const status = resultValue<string>(response.data, 'reopenStatus') ?? 'Unknown'
      if (status === 'CompletedNoChange') {
        setNotice('GL authoritatively recorded zero admitted reopen postings. The request completed as no-change, the prior watermark and seal remain authoritative, and no reclose run was created.')
      } else if (status === 'RecloseInProgress') {
        setNotice('GL authoritatively recorded admitted reopen work. Reclose is mandatory; the period is Closing under a new non-releasable reclose run.')
      } else {
        setNotice(`The approved reopen closure is now ${status}.`)
      }
    } catch (error) {
      setBlockedReason('The closure outcome could not be confirmed. The current gate remains restrictive until authoritative FPM and GL state are reconciled.')
      setNotice(error instanceof Error ? error.message : 'The closure command could not reach the API. Retry with the same command identity.')
    } finally {
      setSaving(false)
    }
  }

  const periodState = resultValue<string>(result, 'periodState') ?? 'HardClosed'
  const reopenStatus = resultValue<string>(result, 'reopenStatus') ?? 'Not requested'
  const gateMode = resultValue<string>(result, 'gateMode') ?? 'HardClosed'
  const expiry = resultValue<string>(result, 'authorizationExpiry') ?? authorizationExpiry
  const scopeEvidence = resultValue<unknown>(result, 'reopenScope')
  const state = semanticState(result, saving, Boolean(blockedReason))
  const canStartReclose = reopenStatus === 'Active' || reopenStatus === 'ExpiredPendingClosure' || reopenStatus === 'GateClosing' || reopenStatus === 'RecloseHandoff'

  return <section aria-labelledby="fpm-reopen-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">FPM-SCR-03</p><h2 id="fpm-reopen-title" className="mt-1 text-2xl font-semibold">Request and authorize reopen</h2><p className="mt-2 max-w-4xl text-base-content/75">Request a bounded scoped correction or operational reopen. FPM records the approval-bearing request; Workflow owns the decision; GL owns the authoritative reopen gate and posting admission.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Pending, rejected, stale, or unavailable approval evidence never opens a gate. The period stays HardClosed until an approved current-state decision and authoritative GL result are both established.</p>
    <Panel title="Reopen control status" description="The status explains the period, request, gate, scope, and cutoff without implying that a request itself grants posting authority." state={state}>
      <div className="flex flex-wrap items-center gap-3"><StatusBadge state={state} label={`${periodState} · ${reopenStatus} · ${gateMode}`} announce /><span className="text-sm text-base-content/70">Scope {currentScope.id} · period {periodID || '—'}</span></div>
      <dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Period state</dt><dd>{periodState}</dd></div><div><dt className="text-sm text-base-content/65">Reopen status</dt><dd>{reopenStatus}</dd></div><div><dt className="text-sm text-base-content/65">Gate mode</dt><dd>{gateMode}</dd></div><div><dt className="text-sm text-base-content/65">Gate version</dt><dd>{resultValue<number>(result, 'gateVersion') ?? gateVersion}</dd></div><div><dt className="text-sm text-base-content/65">Authorization expiry</dt><dd>{expiry || '—'}</dd></div><div><dt className="text-sm text-base-content/65">Reopen request ID</dt><dd className="break-all font-mono">{resultValue<string>(result, 'reopenRequestId') ?? '—'}</dd></div></dl>
      {scopeEvidence ? <div className="mt-5 rounded-box border border-base-300 bg-base-200/50 p-4"><h3 className="font-semibold">Authorized scope</h3><pre className="mt-2 overflow-auto whitespace-pre-wrap text-sm">{JSON.stringify(scopeEvidence, null, 2)}</pre></div> : null}
      {blockedReason ? <p role="alert" className="mt-5 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{blockedReason}</p> : null}
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}{lastErrorCode ? ` (${lastErrorCode})` : ''}</p>
    </Panel>
    <Panel title="Request package" description="The request is immutable evidence for the later Workflow decision. ScopedCorrection accepts accounts or transaction classes; Operational requires actors and transaction classes.">
      <form onSubmit={(event) => void submitRequest(event)} className="space-y-4">
        <div className="grid gap-4 md:grid-cols-2"><div className="form-control w-full gap-2"><label className="label" htmlFor="fpm-reopen-mode"><span className="font-medium">Reopen mode</span></label><Select id="fpm-reopen-mode" value={mode} onChange={(event) => setMode(event.target.value as typeof mode)} disabled={saving || Boolean(result)}><option value="ScopedCorrection">Scoped correction</option><option value="Operational">Operational</option></Select></div><Field id="fpm-reopen-period-id" label="Fiscal period ID" value={periodID} readOnly disabled description="Taken from the selected accounting scope." /><Field id="fpm-reopen-period-version" label="Expected period version" type="number" min="1" value={periodVersion} onChange={(event) => setPeriodVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-reopen-gate-version" label="Expected GL gate version" type="number" min="1" value={gateVersion} onChange={(event) => setGateVersion(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-reopen-authority-epoch" label="Control authority epoch" type="number" min="1" value={authorityEpoch} onChange={(event) => setAuthorityEpoch(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-reopen-expiry" label="Authorization expiry" type="datetime-local" value={authorizationExpiry} onChange={(event) => setAuthorizationExpiry(event.target.value)} required disabled={saving || Boolean(result)} /></div>
        <Field id="fpm-reopen-reason" label="Reason" value={reason} onChange={(event) => setReason(event.target.value)} required disabled={saving || Boolean(result)} />
        <div className="grid gap-4 md:grid-cols-2"><Field id="fpm-reopen-account-ids" label="Account IDs" value={accountIDs} onChange={(event) => setAccountIDs(event.target.value)} disabled={saving || Boolean(result)} description="Comma- or space-separated IDs." /><Field id="fpm-reopen-transaction-classes" label="Transaction classes" value={transactionClasses} onChange={(event) => setTransactionClasses(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-reopen-actor-ids" label="Actor IDs" value={actorIDs} onChange={(event) => setActorIDs(event.target.value)} disabled={saving || Boolean(result)} description="Required for Operational mode." /><Field id="fpm-reopen-proposed-corrections" label="Proposed corrections JSON" value={proposedCorrections} onChange={(event) => setProposedCorrections(event.target.value)} required disabled={saving || Boolean(result)} /><Field id="fpm-reopen-impact" label="Impact JSON" value={impact} onChange={(event) => setImpact(event.target.value)} required disabled={saving || Boolean(result)} /></div>
        <div className="flex flex-wrap items-center gap-3"><Button type="submit" loading={saving} disabled={Boolean(result)}>Request reopen approval</Button>{result ? <span className="text-sm text-base-content/70">The request is recorded. Apply only an independent current Workflow decision.</span> : null}</div>
      </form>
    </Panel>
    {result ? <Panel title="Apply Workflow decision" description="Supply the immutable approval reference, current candidate fingerprint, subject/policy versions, and separation-of-duties evidence. FPM revalidates all evidence before asking GL to open the bounded gate.">
      <form onSubmit={(event) => void applyApproval(event)} className="space-y-4"><div className="grid gap-4 md:grid-cols-2"><Field id="fpm-reopen-request-id" label="Approval request ID" value={approvalRequestID} onChange={(event) => setApprovalRequestID(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-decision-id" label="Decision ID" value={decisionID} onChange={(event) => setDecisionID(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-decision-version" label="Decision version" type="number" min="1" value={decisionVersion} onChange={(event) => setDecisionVersion(event.target.value)} required disabled={saving} /><div className="form-control w-full gap-2"><label className="label" htmlFor="fpm-reopen-decision-outcome"><span className="font-medium">Decision outcome</span></label><Select id="fpm-reopen-decision-outcome" value={decisionOutcome} onChange={(event) => setDecisionOutcome(event.target.value as typeof decisionOutcome)} disabled={saving}><option value="approved">Approved</option><option value="rejected">Rejected</option></Select></div><Field id="fpm-reopen-decision-reason" label="Decision reason" value={decisionReason} onChange={(event) => setDecisionReason(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-policy-version" label="Policy version" value={policyVersion} onChange={(event) => setPolicyVersion(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-subject-version" label="Subject version" type="number" min="1" value={subjectVersion} onChange={(event) => setSubjectVersion(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-candidate-fingerprint" label="Candidate fingerprint" value={candidateFingerprint} onChange={(event) => setCandidateFingerprint(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-decision-reference" label="Decision reference" value={decisionReference} onChange={(event) => setDecisionReference(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-approver-user-id" label="Approver user ID" value={approverUserID} onChange={(event) => setApproverUserID(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-sod-reference" label="Separation-of-duties reference" value={separationReference} onChange={(event) => setSeparationReference(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-recorded-at" label="Recorded at" type="datetime-local" value={recordedAt} onChange={(event) => setRecordedAt(event.target.value)} required disabled={saving} /><Field id="fpm-reopen-expected-request-version" label="Expected reopen-request version" type="number" min="1" value={expectedRequestVersion} onChange={(event) => setExpectedRequestVersion(event.target.value)} required disabled={saving} /></div><div className="flex flex-wrap items-center gap-3"><Button type="submit" loading={saving}>Apply approval decision</Button><span className="text-sm text-base-content/70">A rejected or stale decision leaves the period HardClosed.</span></div></form>
    </Panel> : null}
    {canStartReclose ? <Panel title="Close approved reopen" description="FPM closes the approved GL gate and evaluates the authoritative admission summary. The controller cannot choose no-change versus reclose: zero admitted postings complete as no-change; any admitted correction requires mandatory reclose.">
      <div className="space-y-4"><p className="text-sm text-base-content/75">The gate remains restrictive while this outcome is established. A successful no-change result retains the prior watermark and seal; a positive result creates a new non-releasable reclose run.</p><Button type="button" loading={saving} onClick={() => void startReclose()}>Close reopen and determine outcome</Button></div>
    </Panel> : null}
  </section>
}
