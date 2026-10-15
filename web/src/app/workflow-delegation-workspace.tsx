import { useRef, useState, type FormEvent } from 'react'

import { wfaDelegateApproval } from '@/generated/api/sdk.gen'
import type { WfaDelegateApprovalEstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, StatusBadge } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `00000000-0000-4000-8000-${Math.random().toString(16).slice(2).padEnd(12, '0').slice(0, 12)}`
}

function errorDetail(error: unknown) {
  if (!error || typeof error !== 'object') return 'The approval delegation could not be established. Review authorization and retry safely.'
  const value = error as { code?: unknown; detail?: unknown }
  if (typeof value.detail === 'string') return `${typeof value.code === 'string' ? `${value.code}: ` : ''}${value.detail}`
  return 'The approval delegation could not be established. Review authorization and retry safely.'
}

function localDateTimeValue(date: Date) {
  const offset = date.getTimezoneOffset() * 60_000
  return new Date(date.getTime() - offset).toISOString().slice(0, 16)
}

function initialEffectiveFrom() {
  const now = new Date()
  now.setSeconds(0, 0)
  return localDateTimeValue(now)
}

function initialEffectiveTo() {
  const tomorrow = new Date()
  tomorrow.setDate(tomorrow.getDate() + 1)
  tomorrow.setSeconds(0, 0)
  return localDateTimeValue(tomorrow)
}

function toUtc(value: string) {
  const parsed = new Date(value)
  return Number.isNaN(parsed.getTime()) ? null : parsed.toISOString()
}

function isUuid(value: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value)
}

type DelegationForm = {
  delegateUserId: string
  policyId: string
  policyVersion: string
  effectiveFrom: string
  effectiveTo: string
}

type SubmissionIdentity = {
  fingerprint: string
  commandId: string
  delegationId: string
  idempotencyKey: string
  correlationId: string
}

export function WorkflowDelegationWorkspace() {
  const { currentScope } = useScopeContext()
  const [form, setForm] = useState<DelegationForm>(() => ({
    delegateUserId: '',
    policyId: '',
    policyVersion: '',
    effectiveFrom: initialEffectiveFrom(),
    effectiveTo: initialEffectiveTo(),
  }))
  const [submitting, setSubmitting] = useState(false)
  const [notice, setNotice] = useState('Enter the delegate, policy binding, and finite UTC effective interval.')
  const [result, setResult] = useState<WfaDelegateApprovalEstablishedResult | null>(null)
  const submissionIdentity = useRef<SubmissionIdentity | null>(null)

  const update = (key: keyof DelegationForm, value: string) => {
    setForm((current) => ({ ...current, [key]: value }))
    setResult(null)
  }

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!currentScope) {
      setNotice('Select an accounting scope before delegating approval responsibility.')
      return
    }

    const delegateUserId = form.delegateUserId.trim()
    const policyId = form.policyId.trim()
    const policyVersion = form.policyVersion.trim()
    const effectiveFrom = toUtc(form.effectiveFrom)
    const effectiveTo = toUtc(form.effectiveTo)
    if (!isUuid(delegateUserId) || !isUuid(policyId)) {
      setNotice('Delegate user ID and policy ID must be valid UUIDs.')
      return
    }
    if (!policyVersion || !effectiveFrom || !effectiveTo) {
      setNotice('Policy version and both finite effective dates are required.')
      return
    }
    if (new Date(effectiveFrom).getTime() >= new Date(effectiveTo).getTime()) {
      setNotice('Effective from must be earlier than effective to. The interval is finite and uses UTC.')
      return
    }

    const data = {
      delegationId: submissionIdentity.current?.delegationId ?? newIdentifier(),
      delegateUserId,
      policyId,
      policyVersion,
      scope: { scopeIds: [currentScope.id] },
      effectiveFrom,
      effectiveTo,
    }
    const fingerprint = JSON.stringify(data)
    const identity = submissionIdentity.current?.fingerprint === fingerprint
      ? submissionIdentity.current
      : { fingerprint, commandId: newIdentifier(), delegationId: data.delegationId, idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    submissionIdentity.current = identity
    setSubmitting(true)
    setNotice('Establishing the immutable approval delegation and waiting for the authoritative result…')
    try {
      const response = await wfaDelegateApproval({
        body: { commandId: identity.commandId, data },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Correlation-Id': identity.correlationId },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setNotice('No established result was returned. No local delegation state was substituted.')
        return
      }
      setResult(response.data)
      setNotice(response.data.data.replayed ? 'The established delegation result was safely replayed from the idempotency key.' : 'Approval responsibility delegated. The effective interval is enforced by Workflow.')
      submissionIdentity.current = null
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The delegation command could not reach the live API. The idempotency identity is retained for safe retry.')
    } finally {
      setSubmitting(false)
    }
  }

  return <div className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">WFA-SCR-04</p><h1 className="mt-1 text-2xl font-semibold">Delegate approval responsibility</h1><p className="mt-2 max-w-3xl text-base-content/75">Create a bounded delegation for a compatible approval policy. Workflow rechecks authority, segregation of duties, policy version, scope, and the finite UTC interval when the command is established and when a decision is made.</p></div>
    <Panel title="Delegation command" description={notice}>
      <form onSubmit={(event) => void submit(event)} noValidate className="space-y-5">
        <div className="grid gap-4 md:grid-cols-2"><Field id="wfa-delegate-user-id" label="Delegate user ID" value={form.delegateUserId} onChange={(event) => update('delegateUserId', event.target.value)} required /><Field id="wfa-delegate-policy-id" label="Policy ID" value={form.policyId} onChange={(event) => update('policyId', event.target.value)} required /><Field id="wfa-delegate-policy-version" label="Policy version" value={form.policyVersion} onChange={(event) => update('policyVersion', event.target.value)} required description="The delegation is bound to this immutable policy version." /></div>
        <fieldset className="rounded-box border border-base-300 p-4"><legend className="px-2 font-semibold">Finite effective interval</legend><div className="grid gap-4 md:grid-cols-2"><Field id="wfa-delegate-effective-from" label="Effective from" type="datetime-local" value={form.effectiveFrom} onChange={(event) => update('effectiveFrom', event.target.value)} required /><Field id="wfa-delegate-effective-to" label="Effective to" type="datetime-local" value={form.effectiveTo} onChange={(event) => update('effectiveTo', event.target.value)} required /></div><p className="mt-3 text-sm text-base-content/70">The browser value is converted to an explicit UTC timestamp. Expiry is the only termination mechanism in this delivery slice; no revoke or end control is exposed.</p></fieldset>
        <div className="rounded-box border border-primary/30 bg-primary/5 p-4"><h2 className="font-semibold">Scope boundary</h2><p className="mt-1 text-sm">{currentScope ? `${currentScope.legalEntity.name} · ${currentScope.ledger.name} · ${currentScope.accountingBook.name}` : 'No accounting scope selected.'}</p><p className="mt-1 break-all font-mono text-xs text-base-content/70">{currentScope?.id ?? 'Select a scope before submitting'}</p></div>
        <div className="flex flex-wrap items-center gap-3"><Button type="submit" loading={submitting}>Delegate approval responsibility</Button><span className="text-sm text-base-content/70">Retry identity is retained after transport or timeout failures.</span></div>
      </form>
    </Panel>
    {result ? <Panel title="Established delegation result" state="success" description="This projection came from the authoritative command result and includes the recorded delegation boundary."><div className="flex flex-wrap items-center gap-3"><StatusBadge state="success" label={result.data.replayed ? 'Replayed' : 'Established'} announce /><span className="font-mono text-sm break-all">{result.aggregateId} · aggregate v{result.aggregateVersion}</span></div><dl className="mt-5 grid gap-4 text-sm sm:grid-cols-2"><div><dt className="font-medium text-base-content/70">Delegator</dt><dd className="mt-1 break-all font-mono">{result.data.delegation.delegatorUserId}</dd></div><div><dt className="font-medium text-base-content/70">Delegate</dt><dd className="mt-1 break-all font-mono">{result.data.delegation.delegateUserId}</dd></div><div><dt className="font-medium text-base-content/70">Policy binding</dt><dd className="mt-1 break-all font-mono">{result.data.delegation.policyId} / {result.data.delegation.policyVersion}</dd></div><div><dt className="font-medium text-base-content/70">Scope</dt><dd className="mt-1 break-all font-mono">{result.data.delegation.scope.scopeIds?.join(', ') || '—'}</dd></div><div><dt className="font-medium text-base-content/70">Effective from</dt><dd className="mt-1">{result.data.delegation.effectiveFrom}</dd></div><div><dt className="font-medium text-base-content/70">Effective to</dt><dd className="mt-1">{result.data.delegation.effectiveTo}</dd></div><div><dt className="font-medium text-base-content/70">Validation</dt><dd className="mt-1">{result.data.validationStatus}</dd></div><div><dt className="font-medium text-base-content/70">Audit reference</dt><dd className="mt-1 break-all font-mono">{result.data.auditReference ?? 'Recorded with the command evidence'}</dd></div></dl></Panel> : null}
  </div>
}
