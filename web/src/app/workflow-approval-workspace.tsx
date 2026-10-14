import { useEffect, useRef, useState, type FormEvent } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { wfaCreateApprovalRequest, wfaGetApprovalRequest, wfaListApprovalRequests } from '@/generated/api/sdk.gen'
import type {
  WfaApprovalRequestProjection,
  WfaApprovalRequestStep,
  WfaCreateApprovalRequestEstablishedResult,
} from '@/generated/api/types.gen'
import { Button, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `00000000-0000-4000-8000-${Math.random().toString(16).slice(2).padEnd(12, '0').slice(0, 12)}`
}

function errorDetail(error: unknown) {
  if (!error || typeof error !== 'object') return 'The approval-request operation could not be established. Review authorization and retry safely.'
  const value = error as { code?: unknown; detail?: unknown }
  if (typeof value.detail === 'string') return `${typeof value.code === 'string' ? `${value.code}: ` : ''}${value.detail}`
  return 'The approval-request operation could not be established. Review authorization and retry safely.'
}

function requestState(value: string): SemanticState {
  if (value === 'Approved') return 'success'
  if (value === 'Rejected') return 'error'
  return 'pending'
}

function formatDate(value?: string) {
  if (!value) return '—'
  return new Date(value).toLocaleString()
}

type CreateForm = {
  subjectId: string
  subjectVersion: string
  subjectFingerprint: string
  label: string
  reference: string
  summary: string
  sourceContext: string
  subjectType: string
  operation: string
  transactionType: string
  dataClassification: string
  amount: string
  currency: string
}

function initialCreateForm(currency: string): CreateForm {
  return {
    subjectId: 'posting-request-001',
    subjectVersion: '1',
    subjectFingerprint: 'sha256:manual-subject-v1',
    label: 'Manual approval subject',
    reference: 'posting-request-001',
    summary: 'Safe review snapshot supplied by the originating workflow.',
    sourceContext: 'general-ledger',
    subjectType: 'posting-request',
    operation: 'submit',
    transactionType: '',
    dataClassification: 'internal',
    amount: '100.00',
    currency,
  }
}

type SubmissionIdentity = {
  fingerprint: string
  commandId: string
  requestId: string
  idempotencyKey: string
  correlationId: string
}

export function WorkflowApprovalWorkspace({ detail = false }: { detail?: boolean }) {
  return detail ? <ApprovalRequestDetail /> : <ApprovalRequestWorklist />
}

function ApprovalRequestWorklist() {
  const { currentScope } = useScopeContext()
  const [assignment, setAssignment] = useState<'assigned' | 'all'>('assigned')
  const [due, setDue] = useState<'due' | 'all'>('all')
  const [state, setState] = useState<'all' | 'blocked' | 'delegated' | 'escalated' | 'completed'>('all')
  const [requests, setRequests] = useState<WfaApprovalRequestProjection[]>([])
  const [loading, setLoading] = useState(true)
  const [notice, setNotice] = useState('Loading the authorized WFA worklist.')
  const [createForm, setCreateForm] = useState<CreateForm>(() => initialCreateForm(currentScope?.functionalCurrency ?? 'USD'))
  const [creating, setCreating] = useState(false)
  const [createdResult, setCreatedResult] = useState<WfaCreateApprovalRequestEstablishedResult | null>(null)
  const submissionIdentity = useRef<SubmissionIdentity | null>(null)

  const load = async () => {
    setLoading(true)
    try {
      const response = await wfaListApprovalRequests({
        query: { assignment, due, state, 'page[size]': 50 },
        headers: { 'X-Correlation-Id': newIdentifier() },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      setRequests(response.data?.data.requests ?? [])
      setNotice(response.data ? `Loaded ${response.data.data.requests.length} authorized approval request(s).` : 'No established worklist result was returned.')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The approval-request worklist could not reach the live API.')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => {
    void load()
    // The filter values are the authoritative read query; load is intentionally
    // recreated when a filter changes so no stale worklist is implied.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [assignment, due, state])

  const updateCreateForm = (key: keyof CreateForm, value: string) => {
    setCreateForm((current) => ({ ...current, [key]: value }))
    submissionIdentity.current = null
  }

  const submitCreate = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!currentScope) {
      setNotice('Select an accounting scope before creating an approval request.')
      return
    }
    if (!/^[A-Z]{3}$/.test(createForm.currency) || !/^\d+(?:\.\d+)?$/.test(createForm.amount)) {
      setNotice('Amount must be an exact non-negative decimal string and currency must be an uppercase three-letter code.')
      return
    }
    const bodyData = {
      requestId: submissionIdentity.current?.requestId ?? newIdentifier(),
      subjectId: createForm.subjectId.trim(),
      subjectVersion: Number(createForm.subjectVersion),
      subjectFingerprint: createForm.subjectFingerprint.trim(),
      subjectSnapshot: {
        label: createForm.label.trim(),
        reference: createForm.reference.trim() || undefined,
        summary: createForm.summary.trim() || undefined,
        amount: createForm.amount,
        currency: createForm.currency,
      },
      applicability: {
        sourceContext: createForm.sourceContext.trim(),
        subjectType: createForm.subjectType.trim(),
        operation: createForm.operation.trim(),
        transactionTypes: createForm.transactionType.trim() ? [createForm.transactionType.trim()] : undefined,
        currencies: [createForm.currency],
        dataClassifications: createForm.dataClassification.trim() ? [createForm.dataClassification.trim()] : undefined,
      },
      scope: { scopeIds: [currentScope.id] },
      transactionType: createForm.transactionType.trim() || undefined,
      dataClassification: createForm.dataClassification.trim() || undefined,
      amount: createForm.amount,
      currency: createForm.currency,
    }
    const fingerprint = JSON.stringify(bodyData)
    const identity = submissionIdentity.current?.fingerprint === fingerprint
      ? submissionIdentity.current
      : { fingerprint, commandId: newIdentifier(), requestId: bodyData.requestId, idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    submissionIdentity.current = identity
    setCreating(true)
    setNotice('Submitting the typed approval-request command and waiting for the authoritative result…')
    try {
      const response = await wfaCreateApprovalRequest({
        body: { commandId: identity.commandId, data: bodyData },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Correlation-Id': identity.correlationId },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setNotice('No established result was returned. No local request state was substituted.')
        return
      }
      setCreatedResult(response.data)
      setNotice(response.data.data.replayed ? 'The established result was safely replayed from the idempotency key.' : 'Approval request established. The WFA worklist will be refreshed from the authorized read model.')
      submissionIdentity.current = null
      await load()
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The approval-request command could not reach the live API.')
    } finally {
      setCreating(false)
    }
  }

  return <div className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">WFA-WS-01</p><h1 className="mt-1 text-2xl font-semibold">Approval-request worklist</h1><p className="mt-2 max-w-3xl text-base-content/75">Review only authorized approval requests. The worklist uses role-based responsibility evidence and keeps policy, subject, version, and scope evidence visible.</p></div>
    <Panel title="Worklist filters" description="Filters are sent to the protected reference route; an empty or stale response is never treated as local truth.">
      <div className="grid gap-4 sm:grid-cols-3">
        <label className="form-control gap-2"><span className="font-medium">Assignment</span><Select aria-label="Assignment" value={assignment} onChange={(event) => setAssignment(event.target.value as typeof assignment)}><option value="assigned">Assigned role work</option><option value="all">All authorized work</option></Select></label>
        <label className="form-control gap-2"><span className="font-medium">Due</span><Select aria-label="Due" value={due} onChange={(event) => setDue(event.target.value as typeof due)}><option value="all">All due states</option><option value="due">Due or overdue</option></Select></label>
        <label className="form-control gap-2"><span className="font-medium">State</span><Select aria-label="State" value={state} onChange={(event) => setState(event.target.value as typeof state)}><option value="all">All states</option><option value="blocked">Blocked</option><option value="delegated">Delegated</option><option value="escalated">Escalated</option><option value="completed">Completed</option></Select></label>
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={loading ? 'pending' : 'success'} label={loading ? 'Loading' : 'Authorized result'} announce /><span className="text-sm text-base-content/70">{notice}</span></div>
      {loading ? <p role="status" aria-busy="true" className="mt-4 rounded-box border border-info/30 bg-info/5 p-4 text-sm">Loading approval requests. No result is implied.</p> : null}
      {!loading && requests.length === 0 ? <p className="mt-4 rounded-box border border-base-300 p-4 text-sm">No matching authorized approval requests.</p> : null}
      {!loading && requests.length > 0 ? <div className="mt-4 overflow-x-auto"><table className="table min-w-full"><caption className="caption-top pb-3 text-left text-sm font-semibold">Authorized approval requests</caption><thead><tr><th scope="col">Subject</th><th scope="col">State</th><th scope="col">Owner roles</th><th scope="col">Amount</th><th scope="col">Due</th></tr></thead><tbody>{requests.map((request) => <tr key={request.id}><th scope="row"><RouterLink className="link link-primary" to={`/approvals/wfa-scr-01?approvalRequestId=${encodeURIComponent(request.id)}`}>{request.subjectSnapshot?.label ?? request.subjectId}</RouterLink><div className="text-xs text-base-content/60">{request.subjectType} · v{request.subjectVersion}</div></th><td><StatusBadge state={requestState(request.status)} label={request.status} /></td><td>{request.responsibleRoles.join(', ') || '—'}</td><td className="font-mono">{request.amount ? `${request.amount} ${request.currency ?? ''}` : '—'}</td><td>{formatDate(request.dueAt)}</td></tr>)}</tbody></table></div> : null}
    </Panel>
    <Panel title="Create approval request" description="The command carries a bounded safe subject snapshot. The server resolves exactly one effective policy and binds that policy version immutably.">
      <form onSubmit={(event) => void submitCreate(event)} noValidate className="space-y-5">
        <div className="grid gap-4 md:grid-cols-2"><Field id="wfa-request-subject-id" label="Subject ID" value={createForm.subjectId} onChange={(event) => updateCreateForm('subjectId', event.target.value)} required /><Field id="wfa-request-subject-version" label="Subject version" type="number" min="1" value={createForm.subjectVersion} onChange={(event) => updateCreateForm('subjectVersion', event.target.value)} required /><Field id="wfa-request-subject-fingerprint" label="Subject fingerprint" value={createForm.subjectFingerprint} onChange={(event) => updateCreateForm('subjectFingerprint', event.target.value)} required /><Field id="wfa-request-label" label="Review label" value={createForm.label} onChange={(event) => updateCreateForm('label', event.target.value)} required /></div>
        <div className="grid gap-4 md:grid-cols-2"><Field id="wfa-request-reference" label="Safe reference" value={createForm.reference} onChange={(event) => updateCreateForm('reference', event.target.value)} /><Field id="wfa-request-summary" label="Safe summary" value={createForm.summary} onChange={(event) => updateCreateForm('summary', event.target.value)} /></div>
        <fieldset className="rounded-box border border-base-300 p-4"><legend className="px-2 font-semibold">Policy applicability</legend><div className="grid gap-4 md:grid-cols-3"><Field id="wfa-request-source-context" label="Source context" value={createForm.sourceContext} onChange={(event) => updateCreateForm('sourceContext', event.target.value)} required /><Field id="wfa-request-subject-type" label="Subject type" value={createForm.subjectType} onChange={(event) => updateCreateForm('subjectType', event.target.value)} required /><Field id="wfa-request-operation" label="Operation" value={createForm.operation} onChange={(event) => updateCreateForm('operation', event.target.value)} required /><Field id="wfa-request-transaction-type" label="Transaction type" value={createForm.transactionType} onChange={(event) => updateCreateForm('transactionType', event.target.value)} /><Field id="wfa-request-classification" label="Data classification" value={createForm.dataClassification} onChange={(event) => updateCreateForm('dataClassification', event.target.value)} /></div></fieldset>
        <div className="grid gap-4 md:grid-cols-2"><Field id="wfa-request-amount" label="Amount" inputMode="decimal" value={createForm.amount} onChange={(event) => updateCreateForm('amount', event.target.value)} required description="Exact decimal text; binary floating point is not used." /><Field id="wfa-request-currency" label="Currency" value={createForm.currency} onChange={(event) => updateCreateForm('currency', event.target.value.toUpperCase())} required maxLength={3} /></div>
        <div className="flex flex-wrap items-center gap-3"><Button type="submit" loading={creating}>Create approval request</Button><span className="text-sm text-base-content/70">Scope: {currentScope?.id ?? 'Select a scope'}</span></div>
      </form>
    </Panel>
    {createdResult ? <Panel title="Established result" state="success" description="This is the server result returned by the create command."><div className="grid gap-3 text-sm sm:grid-cols-2"><div><span className="font-medium">Request ID</span><p className="break-all">{createdResult.aggregateId}</p></div><div><span className="font-medium">Policy version</span><p>{createdResult.data.request.policyVersion}</p></div><div><span className="font-medium">Approval status</span><p>{createdResult.data.approvalStatus}</p></div><div><span className="font-medium">Responsible roles</span><p>{createdResult.data.responsibleRoles.join(', ') || '—'}</p></div></div><RouterLink className="btn btn-secondary mt-4" to={`/approvals/wfa-scr-01?approvalRequestId=${encodeURIComponent(createdResult.aggregateId)}`}>Open read-only detail</RouterLink></Panel> : null}
  </div>
}

function ApprovalRequestDetail() {
  const [searchParams] = useSearchParams()
  const requestID = searchParams.get('approvalRequestId') ?? ''
  const [request, setRequest] = useState<WfaApprovalRequestProjection | null>(null)
  const [loading, setLoading] = useState(Boolean(requestID))
  const [notice, setNotice] = useState(requestID ? 'Loading the authorized approval-request detail.' : 'Choose an approval request from the worklist.')

  useEffect(() => {
    if (!requestID) {
      setLoading(false)
      setRequest(null)
      return
    }
    let cancelled = false
    setLoading(true)
    void wfaGetApprovalRequest({ path: { approvalRequestId: requestID }, headers: { 'X-Correlation-Id': newIdentifier() } }).then((response) => {
      if (cancelled) return
      if (response.error) {
        setNotice(errorDetail(response.error))
        setRequest(null)
      } else if (response.data) {
        setRequest(response.data.data.request)
        setNotice('Authorized read-only detail loaded.')
      } else {
        setNotice('No established detail result was returned.')
        setRequest(null)
      }
    }).catch((error: unknown) => {
      if (!cancelled) setNotice(error instanceof Error ? error.message : 'The approval-request detail could not reach the live API.')
    }).finally(() => {
      if (!cancelled) setLoading(false)
    })
    return () => { cancelled = true }
  }, [requestID])

  return <div className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">WFA-SCR-01</p><h1 className="mt-1 text-2xl font-semibold">Approval-request detail</h1><p className="mt-2 max-w-3xl text-base-content/75">Read-only evidence for the established subject snapshot, policy binding, responsibility, and lifecycle state.</p></div>
    <Panel title="Authorized detail" description={notice}>
      {loading ? <p role="status" aria-busy="true">Loading authoritative detail. No local result is implied.</p> : null}
      {!loading && !request ? <p className="rounded-box border border-base-300 p-4 text-sm">{requestID ? 'The detail could not be loaded.' : 'Return to the WFA worklist and choose a request.'}</p> : null}
      {request ? <div className="space-y-5"><div className="flex flex-wrap items-center gap-3"><StatusBadge state={requestState(request.status)} label={request.status} announce /><span className="text-sm text-base-content/70">Aggregate version {request.aggregateVersion}</span></div><div className="grid gap-4 md:grid-cols-2"><Evidence label="Request ID" value={request.id} /><Evidence label="Subject" value={`${request.subjectType} / ${request.subjectId} / v${request.subjectVersion}`} /><Evidence label="Subject fingerprint" value={request.subjectFingerprint} /><Evidence label="Snapshot fingerprint" value={request.snapshotFingerprint ?? '—'} /><Evidence label="Source operation" value={`${request.sourceContext ?? '—'} / ${request.operation ?? '—'}`} /><Evidence label="Policy binding" value={`${request.policyId} / ${request.policyVersion}`} /><Evidence label="Responsible roles" value={request.responsibleRoles.join(', ') || '—'} /><Evidence label="Due at" value={formatDate(request.dueAt)} /></div><Panel title="Safe subject snapshot" description="Allow-listed review evidence only."><dl className="grid gap-3 text-sm sm:grid-cols-2"><Evidence label="Label" value={request.subjectSnapshot?.label ?? '—'} /><Evidence label="Reference" value={request.subjectSnapshot?.reference ?? '—'} /><Evidence label="Summary" value={request.subjectSnapshot?.summary ?? '—'} /><Evidence label="Amount" value={request.subjectSnapshot?.amount ? `${request.subjectSnapshot.amount} ${request.subjectSnapshot.currency ?? ''}` : '—'} /></dl></Panel><Panel title="Copied approval steps" description="Policy-defined step configuration copied at request creation."><ol className="space-y-3">{request.steps.map((step) => <ApprovalStep key={step.id} step={step} />)}</ol></Panel><div className="flex flex-wrap gap-3"><RouterLink className="btn btn-secondary" to="/approvals/wfa-ws-01">Back to worklist</RouterLink></div></div> : null}
    </Panel>
  </div>
}

function Evidence({ label, value }: { label: string; value: string }) {
  return <div><dt className="font-medium text-base-content/70">{label}</dt><dd className="mt-1 break-words font-mono text-sm">{value}</dd></div>
}

function ApprovalStep({ step }: { step: WfaApprovalRequestStep }) {
  return <li className="rounded-box border border-base-300 p-4"><div className="flex flex-wrap items-center gap-3"><span className="font-semibold">{step.name ?? `Step ${step.sequence}`}</span><StatusBadge state={requestState(step.status === 'Approved' ? 'Approved' : step.status === 'Rejected' ? 'Rejected' : 'Pending')} label={step.status} /></div><p className="mt-2 text-sm text-base-content/70">Roles: {step.requiredRoles.map((role) => role.role).join(', ') || '—'} · Minimum approvals: {step.minimumApprovals}</p></li>
}
