import { useRef, useState, type FormEvent } from 'react'

import { wfaMaintainApprovalPolicies } from '@/generated/api/sdk.gen'
import type { WfaApprovalPolicyProjection, WfaMaintainApprovalPoliciesEstablishedResult, WfaMaintainApprovalPoliciesCommandData } from '@/generated/api/types.gen'
import { Button, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `00000000-0000-4000-8000-${Math.random().toString(16).slice(2).padEnd(12, '0').slice(0, 12)}`
}

function dateTimeForApi(value: string) {
  return `${value}T00:00:00.000Z`
}

function decimalGreater(left: string, right: string) {
  const parse = (value: string) => {
    const match = /^(\d+)(?:\.(\d+))?$/.exec(value)
    if (!match) return null
    const fraction = match[2] ?? ''
    return { unscaled: BigInt(`${match[1]}${fraction}`), scale: fraction.length }
  }
  const leftValue = parse(left)
  const rightValue = parse(right)
  if (!leftValue || !rightValue) return false
  const scale = Math.max(leftValue.scale, rightValue.scale)
  const factor = (value: { unscaled: bigint; scale: number }) => value.unscaled * (BigInt(10) ** BigInt(scale - value.scale))
  return factor(leftValue) > factor(rightValue)
}

function errorDetail(error: unknown) {
  if (!error || typeof error !== 'object') return 'The policy command could not be established. Review authorization and retry safely.'
  const value = error as { code?: unknown; detail?: unknown }
  if (typeof value.detail === 'string') return `${typeof value.code === 'string' ? `${value.code}: ` : ''}${value.detail}`
  return 'The policy command could not be established. Review authorization and retry safely.'
}

function resultState(status: string): SemanticState {
  if (status === 'Active') return 'success'
  if (status === 'Superseded') return 'disabled'
  if (status === 'PendingApproval') return 'pending'
  return 'info'
}

type SubmissionIdentity = {
  fingerprint: string
  commandId: string
  idempotencyKey: string
  correlationId: string
}

const initialDate = new Date().toISOString().slice(0, 10)

export function ApprovalPolicyEditor() {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [action, setAction] = useState<WfaMaintainApprovalPoliciesCommandData['action']>('create')
  const [policyId, setPolicyId] = useState('')
  const [expectedVersion, setExpectedVersion] = useState('')
  const [code, setCode] = useState('manual-approval-policy')
  const [name, setName] = useState('Manual approval policy')
  const [policyVersion, setPolicyVersion] = useState('v1')
  const [sourceContext, setSourceContext] = useState('general-ledger')
  const [subjectType, setSubjectType] = useState('posting-request')
  const [operation, setOperation] = useState('submit')
  const [role, setRole] = useState('finance.approver')
  const [minimum, setMinimum] = useState('0.00')
  const [maximum, setMaximum] = useState('10000.00')
  const [currency, setCurrency] = useState(currentScope?.functionalCurrency ?? 'USD')
  const [effectiveFrom, setEffectiveFrom] = useState(initialDate)
  const [effectiveTo, setEffectiveTo] = useState('')
  const [ruleId, setRuleId] = useState<string>(newIdentifier)
  const [stepId, setStepId] = useState<string>(newIdentifier)
  const [approvalRequestId, setApprovalRequestId] = useState('')
  const [decisionId, setDecisionId] = useState('')
  const [decisionVersion, setDecisionVersion] = useState('')
  const [subjectVersion, setSubjectVersion] = useState('')
  const [candidateFingerprint, setCandidateFingerprint] = useState('')
  const [approverUserId, setApproverUserId] = useState('')
  const [submitForApproval, setSubmitForApproval] = useState(false)
  const [reason, setReason] = useState('Initial approval-policy configuration')
  const [validationErrors, setValidationErrors] = useState<string[]>([])
  const [notice, setNotice] = useState('Review the effective interval and scope before submitting the policy command.')
  const [saving, setSaving] = useState(false)
  const [result, setResult] = useState<WfaMaintainApprovalPoliciesEstablishedResult | null>(null)
  const submissionIdentity = useRef<SubmissionIdentity | null>(null)

  const markChanged = () => setHasUnsavedChanges(true)
  const scopeId = currentScope?.id ?? ''

  const validate = () => {
    const errors: string[] = []
    if (!currentScope) errors.push('Select an accounting scope before maintaining a policy.')
    if (!code.trim()) errors.push('Policy code is required.')
    if (!name.trim()) errors.push('Policy name is required.')
    if (!policyVersion.trim()) errors.push('Policy version is required for immutable history.')
    if (!sourceContext.trim() || !subjectType.trim() || !operation.trim()) errors.push('Applicability requires source context, subject type, and operation.')
    if (!role.trim()) errors.push('At least one approver role is required.')
    if (!/^\d+(?:\.\d+)?$/.test(minimum) || !/^\d+(?:\.\d+)?$/.test(maximum)) errors.push('Thresholds must be non-negative exact-decimal strings.')
    if (decimalGreater(minimum, maximum)) errors.push('Minimum threshold cannot exceed maximum threshold.')
    if (!/^[A-Z]{3}$/.test(currency)) errors.push('Currency must be an uppercase three-letter code.')
    if (!effectiveFrom) errors.push('Effective-from is required.')
    if (effectiveTo && effectiveTo <= effectiveFrom) errors.push('Effective-to must be after effective-from.')
    if (action !== 'create' && !policyId.trim()) errors.push('Policy ID is required when editing or changing lifecycle state.')
    if (action !== 'create' && (!expectedVersion || Number(expectedVersion) < 1)) errors.push('Expected aggregate version is required for stale-write protection.')
    if (action === 'activate') {
      if (!approvalRequestId.trim() || !decisionId.trim() || !approverUserId.trim()) errors.push('Activation requires the generic approval request, decision, and approver references.')
      if (!decisionVersion || Number(decisionVersion) < 1 || !subjectVersion || Number(subjectVersion) < 1) errors.push('Activation requires positive decision and subject versions.')
      if (!candidateFingerprint.trim()) errors.push('Activation requires the candidate fingerprint returned for the approved policy version.')
    }
    return errors
  }

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const errors = validate()
    setValidationErrors(errors)
    if (errors.length) {
      setNotice('Resolve the validation summary before submitting.')
      return
    }
    const bodyData: WfaMaintainApprovalPoliciesCommandData = {
      action,
      policyId: policyId.trim() || undefined,
      code: code.trim(),
      name: name.trim(),
      policyVersion: policyVersion.trim(),
      applicability: { sourceContext: sourceContext.trim(), subjectType: subjectType.trim(), operation: operation.trim(), currencies: [currency] },
      scope: { scopeIds: [scopeId] },
      rules: [{ id: ruleId, threshold: { minimum, maximum, currency }, requiredRoles: [{ role: role.trim(), minimumApprovals: 1, allowDelegation: false }], delegation: { enabled: false, requiresIndependentApproval: false }, escalation: { enabled: false } }],
      steps: [{ id: stepId, sequence: 1, name: 'Independent approval', requiredRoles: [{ role: role.trim(), minimumApprovals: 1, allowDelegation: false }], minimumApprovals: 1, delegation: { enabled: false, requiresIndependentApproval: false }, escalation: { enabled: false } }],
      effectiveFrom: dateTimeForApi(effectiveFrom),
      effectiveTo: effectiveTo ? dateTimeForApi(effectiveTo) : undefined,
      approval: action === 'activate' ? { approvalRequestId: approvalRequestId.trim(), decisionId: decisionId.trim(), policyVersion: policyVersion.trim(), decisionVersion: Number(decisionVersion), subjectVersion: Number(subjectVersion), candidateFingerprint: candidateFingerprint.trim(), approverUserId: approverUserId.trim() } : undefined,
      submitForApproval: submitForApproval || undefined,
      reason: reason.trim() || undefined,
      dependentWorkflowImpact: 'Existing requests retain their established policy version; new requests use the selected effective policy.',
    }
    // Expected-version changes are a new command fingerprint. This is
    // important after a stale-version response: retrying with the same policy
    // content but a refreshed version must receive a new idempotency identity.
    const fingerprint = JSON.stringify({ bodyData, expectedVersion: action === 'create' ? undefined : Number(expectedVersion) })
    const identity = submissionIdentity.current?.fingerprint === fingerprint
      ? submissionIdentity.current
      : { fingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    submissionIdentity.current = identity
    setSaving(true)
    setNotice('Submitting the policy command and waiting for the authoritative result…')
    try {
      const response = await wfaMaintainApprovalPolicies({
        body: { commandId: identity.commandId, expectedVersion: action === 'create' ? undefined : Number(expectedVersion), data: bodyData },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Correlation-Id': identity.correlationId, ...(action === 'create' ? {} : { 'If-Match': `"${expectedVersion}"` }) },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        if ((response.error as { status?: number }).status === 409) setNotice(`${errorDetail(response.error)} Refresh the expected version and retry with a new command identity.`)
        return
      }
      if (!response.data) {
        setNotice('No established result was returned. No local policy state was substituted.')
        return
      }
      setResult(response.data)
      setPolicyId(response.data.aggregateId)
      setExpectedVersion(String(response.data.aggregateVersion))
      setPolicyVersion(response.data.data.policyVersion)
      setCandidateFingerprint(response.data.data.policy.candidateFingerprint)
      setApprovalRequestId(response.data.data.approvalRequestId ?? response.data.data.policy.approvalRequestId ?? '')
      setRuleId(response.data.data.policy.rules[0]?.id ?? ruleId)
      setStepId(response.data.data.policy.steps[0]?.id ?? stepId)
      if (response.data.data.policy.approval) {
        setDecisionId(response.data.data.policy.approval.decisionId)
        setDecisionVersion(String(response.data.data.policy.approval.decisionVersion))
        setSubjectVersion(String(response.data.data.policy.approval.subjectVersion))
        setApproverUserId(response.data.data.policy.approval.approverUserId)
      }
      setAction('update')
      setSubmitForApproval(false)
      setHasUnsavedChanges(false)
      submissionIdentity.current = null
      setNotice(response.data.data.replayed ? 'The established result was safely replayed from the idempotency key.' : 'Policy result established. Review approval status and effective-date evidence below.')
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The policy command could not reach the live API.')
    } finally {
      setSaving(false)
    }
  }

  return <div className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">WFA-SCR-03</p><h1 className="mt-1 text-2xl font-semibold">Maintain versioned approval policies</h1><p className="mt-2 max-w-3xl text-base-content/75">Edit a candidate policy, review its effective interval and scope, and submit it through the generic approval workflow. Established versions are immutable.</p></div>
    <Panel title="Policy command" description="The editor sends one typed command to the approved maintain-approval-policies route. It does not maintain a local authoritative read model.">
      <form onSubmit={(event) => void submit(event)} noValidate>
        {validationErrors.length ? <div role="alert" aria-live="assertive" className="mb-5 rounded-box border border-error/40 bg-error/10 p-4"><h2 className="font-semibold">Review these fields before submitting</h2><ul className="mt-2 list-disc space-y-1 pl-5 text-sm">{validationErrors.map((error) => <li key={error}>{error}</li>)}</ul></div> : null}
        <div className="grid gap-4 md:grid-cols-2">
          <label className="form-control w-full gap-2"><span className="label font-medium">Command action</span><Select aria-label="Command action" value={action} onChange={(event) => { setAction(event.target.value as WfaMaintainApprovalPoliciesCommandData['action']); markChanged() }}><option value="create">Create draft</option><option value="update">Create next version</option><option value="submitForApproval">Submit for approval</option><option value="activate">Activate approved version</option></Select></label>
          <Field id="wfa-policy-id" label="Policy ID" value={policyId} onChange={(event) => { setPolicyId(event.target.value); markChanged() }} description="Required for update, submission, and activation; leave blank for a new family." />
          <Field id="wfa-expected-version" label="Expected aggregate version" type="number" min="1" value={expectedVersion} onChange={(event) => { setExpectedVersion(event.target.value); markChanged() }} description="Used with If-Match to reject stale edits." />
          <Field id="wfa-policy-code" label="Policy code" value={code} onChange={(event) => { setCode(event.target.value); markChanged() }} required />
          <Field id="wfa-policy-name" label="Policy name" value={name} onChange={(event) => { setName(event.target.value); markChanged() }} required />
          <Field id="wfa-policy-version" label="Policy version" value={policyVersion} onChange={(event) => { setPolicyVersion(event.target.value); markChanged() }} required description="A new value is required for every revision." />
        </div>
        <fieldset className="mt-6 rounded-box border border-base-300 p-4"><legend className="px-2 font-semibold">Applicability</legend><div className="grid gap-4 md:grid-cols-3"><Field id="wfa-source-context" label="Source context" value={sourceContext} onChange={(event) => { setSourceContext(event.target.value); markChanged() }} required /><Field id="wfa-subject-type" label="Subject type" value={subjectType} onChange={(event) => { setSubjectType(event.target.value); markChanged() }} required /><Field id="wfa-operation" label="Operation" value={operation} onChange={(event) => { setOperation(event.target.value); markChanged() }} required /></div></fieldset>
        <fieldset className="mt-6 rounded-box border border-base-300 p-4"><legend className="px-2 font-semibold">Approval rule</legend><div className="grid gap-4 md:grid-cols-3"><Field id="wfa-role" label="Required approver role" value={role} onChange={(event) => { setRole(event.target.value); markChanged() }} required /><Field id="wfa-minimum" label="Minimum threshold" inputMode="decimal" value={minimum} onChange={(event) => { setMinimum(event.target.value); markChanged() }} required /><Field id="wfa-maximum" label="Maximum threshold" inputMode="decimal" value={maximum} onChange={(event) => { setMaximum(event.target.value); markChanged() }} required /><Field id="wfa-currency" label="Threshold currency" value={currency} onChange={(event) => { setCurrency(event.target.value.toUpperCase()); markChanged() }} maxLength={3} required /></div></fieldset>
        <fieldset className="mt-6 rounded-box border border-base-300 p-4"><legend className="px-2 font-semibold">Effective-date review</legend><div className="grid gap-4 md:grid-cols-2"><Field id="wfa-effective-from" label="Effective from" type="date" value={effectiveFrom} onChange={(event) => { setEffectiveFrom(event.target.value); markChanged() }} required /><Field id="wfa-effective-to" label="Effective to (optional)" type="date" value={effectiveTo} onChange={(event) => { setEffectiveTo(event.target.value); markChanged() }} /></div><p className="mt-3 text-sm text-base-content/70">Inclusive interval endpoints are checked by the workflow aggregate. Overlapping selectable policies are rejected during the transaction recheck.</p></fieldset>
        {action === 'activate' ? <fieldset className="mt-6 rounded-box border border-base-300 p-4"><legend className="px-2 font-semibold">Independent approval evidence</legend><p className="mb-4 text-sm text-base-content/70">Paste the immutable generic WFA decision reference. The API revalidates the request, decision, policy version, subject version, fingerprint, and segregation boundary.</p><div className="grid gap-4 md:grid-cols-2"><Field id="wfa-approval-request-id" label="Approval request ID" value={approvalRequestId} onChange={(event) => { setApprovalRequestId(event.target.value); markChanged() }} required /><Field id="wfa-decision-id" label="Decision ID" value={decisionId} onChange={(event) => { setDecisionId(event.target.value); markChanged() }} required /><Field id="wfa-decision-version" label="Decision version" type="number" min="1" value={decisionVersion} onChange={(event) => { setDecisionVersion(event.target.value); markChanged() }} required /><Field id="wfa-subject-version" label="Approved subject version" type="number" min="1" value={subjectVersion} onChange={(event) => { setSubjectVersion(event.target.value); markChanged() }} required /><Field id="wfa-candidate-fingerprint" label="Candidate fingerprint" value={candidateFingerprint} onChange={(event) => { setCandidateFingerprint(event.target.value); markChanged() }} required /><Field id="wfa-approver-user-id" label="Approver user ID" value={approverUserId} onChange={(event) => { setApproverUserId(event.target.value); markChanged() }} required /></div></fieldset> : null}
        <div className="mt-6 grid gap-4 md:grid-cols-2"><Field id="wfa-reason" label="Change reason" value={reason} onChange={(event) => { setReason(event.target.value); markChanged() }} required /><label className="label cursor-pointer justify-start gap-3 self-end pb-2"><input className="checkbox checkbox-primary" type="checkbox" checked={submitForApproval} onChange={(event) => { setSubmitForApproval(event.target.checked); markChanged() }} /><span className="font-medium">Create the generic approval request now</span></label></div>
        <div className="mt-6 rounded-box border border-primary/30 bg-primary/5 p-4"><h2 className="font-semibold">Scope review</h2><p className="mt-1 text-sm">{currentScope ? `${currentScope.legalEntity.name} · ${currentScope.ledger.name} · ${currentScope.accountingBook.name} · ${currentScope.id}` : 'No accounting scope selected.'}</p><p className="mt-2 text-sm text-base-content/70">The command binds this policy to the selected scope. Authorization is evaluated by IAM; a denied, stale, or unavailable decision is not treated as success.</p></div>
        <p role="status" aria-live="polite" className="mt-5 text-sm text-base-content/75">{notice}</p><div className="mt-4"><Button type="submit" loading={saving}>{action === 'activate' ? 'Activate approved policy' : submitForApproval ? 'Submit policy for approval' : 'Establish policy command'}</Button></div>
      </form>
    </Panel>
    <PolicyResult result={result} />
  </div>
}

function PolicyResult({ result }: { result: WfaMaintainApprovalPoliciesEstablishedResult | null }) {
  if (!result) return <Panel title="Established policy result" description="The authoritative result appears here after a successful command."><p className="text-sm text-base-content/70">No policy result is loaded in this browser session.</p></Panel>
  const policy: WfaApprovalPolicyProjection = result.data.policy
  return <Panel title="Established policy result" description="This projection came from the command result; it is not a locally maintained policy read model." state={resultState(policy.status)}><div className="flex flex-wrap items-center gap-3"><StatusBadge state={resultState(policy.status)} label={`${policy.status} · ${result.data.approvalStatus}`} announce /><span className="font-mono text-sm">{result.aggregateId} · aggregate v{result.aggregateVersion}</span></div><dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Policy version</dt><dd>{policy.policyVersion}</dd></div><div><dt className="text-sm text-base-content/65">Scope</dt><dd className="font-mono break-all">{policy.scope.scopeIds?.join(', ') || 'Typed identity scope'}</dd></div><div><dt className="text-sm text-base-content/65">Effective interval</dt><dd>{policy.effectiveFrom} → {policy.effectiveTo ?? 'open-ended'}</dd></div><div><dt className="text-sm text-base-content/65">Activation outcome</dt><dd>{result.data.activationOutcome}</dd></div><div><dt className="text-sm text-base-content/65">Approval request</dt><dd className="font-mono break-all">{result.data.approvalRequestId ?? 'Not requested'}</dd></div><div><dt className="text-sm text-base-content/65">Decision reference</dt><dd className="font-mono break-all">{result.data.decisionReference ?? 'Not established'}</dd></div><div><dt className="text-sm text-base-content/65">Candidate fingerprint</dt><dd className="font-mono break-all">{policy.candidateFingerprint}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="font-mono break-all">{result.data.auditReference ?? 'Unavailable in result'}</dd></div><div><dt className="text-sm text-base-content/65">Dependent workflow impact</dt><dd>{result.data.dependentWorkflowImpact}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{result.data.replayed ? 'Established result replayed' : 'First establishment'}</dd></div></dl></Panel>
}

