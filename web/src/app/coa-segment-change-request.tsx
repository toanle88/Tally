import { useEffect, useMemo, useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { Button, Field, Panel, Select } from '@/components/ui'
import { ValidationSummary } from '@/components/validation-summary'
import { coaRequestSegmentChanges } from '@/generated/api/sdk.gen'
import { useScopeContext } from '@/lib/scope/scope-context'

import {
  readSafeSegmentDefinitions,
  readSafeSegmentValues,
  type SegmentChangeRequestRecord,
  type SegmentDefinitionRecord,
  type SegmentDefinitionStatus,
  type SegmentValueRecord,
} from './coa-segment-workspace'

function isUuid(value: string) {
  return /^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i.test(value.trim())
}

function statusTransitionIsValid(current: SegmentDefinitionStatus | undefined, next: SegmentDefinitionStatus) {
  if (!current || current === next) return true
  if (current === 'draft') return next === 'active' || next === 'retired'
  if (current === 'active') return next === 'suspended' || next === 'retired'
  if (current === 'suspended') return next === 'active' || next === 'retired'
  return false
}

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The COA change request could not be completed. Review the current subject version and retry safely.'
}

function requestProjection(value: unknown): SegmentChangeRequestRecord | undefined {
  if (!value || typeof value !== 'object') return undefined
  const candidate = value as Partial<SegmentChangeRequestRecord>
  if (typeof candidate.id !== 'string' || typeof candidate.scopeId !== 'string' || (candidate.changeType !== 'definition' && candidate.changeType !== 'value') || typeof candidate.subjectId !== 'string' || typeof candidate.subjectVersion !== 'number' || typeof candidate.requestedEffectiveDate !== 'string' || typeof candidate.approvalRequestId !== 'string' || typeof candidate.approvalStatus !== 'string' || typeof candidate.applicationStatus !== 'string' || typeof candidate.validationOutcome !== 'string' || typeof candidate.nextAction !== 'string' || typeof candidate.version !== 'number' || typeof candidate.revisionNumber !== 'number') return undefined
  return {
    id: candidate.id,
    scopeId: candidate.scopeId,
    changeType: candidate.changeType,
    subjectId: candidate.subjectId,
    subjectVersion: candidate.subjectVersion,
    requestedEffectiveDate: candidate.requestedEffectiveDate.slice(0, 10),
    approvalRequestId: candidate.approvalRequestId,
    approvalStatus: candidate.approvalStatus,
    applicationStatus: candidate.applicationStatus,
    validationOutcome: candidate.validationOutcome,
    conflictCode: candidate.conflictCode,
    rejectionReason: candidate.rejectionReason,
    nextAction: candidate.nextAction,
    proposedChange: candidate.proposedChange && typeof candidate.proposedChange === 'object' ? candidate.proposedChange as Record<string, unknown> : {},
    version: candidate.version,
    revisionNumber: candidate.revisionNumber,
  }
}

export function CoaSegmentChangeRequest() {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [params] = useSearchParams()
  const [changeType, setChangeType] = useState<'definition' | 'value'>(params.get('changeType') === 'value' ? 'value' : 'definition')
  const definitions = useMemo(() => currentScope ? readSafeSegmentDefinitions(currentScope.id) : [], [currentScope?.id])
  const values = useMemo(() => currentScope ? definitions.flatMap((definition) => readSafeSegmentValues(currentScope.id, definition.id)) : [], [currentScope?.id, definitions])
  const [subjectId, setSubjectId] = useState(params.get('subjectId') ?? '')
  const subjectValue = changeType === 'value' ? values.find((record) => record.id === subjectId) : undefined
  const subjectDefinition = changeType === 'definition' ? definitions.find((record) => record.id === subjectId) : definitions.find((record) => record.id === subjectValue?.segmentDefinitionId)
  const subject = changeType === 'definition' ? subjectDefinition : subjectValue
  const [segmentType, setSegmentType] = useState(subjectDefinition?.segmentType ?? 'department')
  const [code, setCode] = useState(subjectDefinition?.code ?? '')
  const [name, setName] = useState(subjectDefinition?.name ?? '')
  const [value, setValue] = useState(subjectValue?.value ?? '')
  const [description, setDescription] = useState(subjectValue?.description ?? '')
  const [status, setStatus] = useState<SegmentDefinitionStatus>(subject?.status ?? 'draft')
  const [effectiveDateFrom, setEffectiveDateFrom] = useState(subject?.effectiveDateFrom ?? '2026-01-01')
  const [effectiveDateTo, setEffectiveDateTo] = useState(subject?.effectiveDateTo ?? '')
  const [requestedEffectiveDate, setRequestedEffectiveDate] = useState(subject?.effectiveDateFrom ?? '2026-01-01')
  const [approvalRequestId, setApprovalRequestId] = useState('')
  const [notice, setNotice] = useState('Select an existing definition or value, provide the Workflow approval-request reference, and review the source version before submitting.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)
  const [result, setResult] = useState<SegmentChangeRequestRecord | undefined>()

  useEffect(() => {
    const candidates = changeType === 'definition' ? definitions : values
    if (!candidates.some((candidate) => candidate.id === subjectId)) setSubjectId(candidates[0]?.id ?? '')
  }, [changeType, definitions, values, subjectId])

  useEffect(() => {
    setSegmentType(subjectDefinition?.segmentType ?? 'department')
    setCode(subjectDefinition?.code ?? '')
    setName(subjectDefinition?.name ?? '')
    setValue(subjectValue?.value ?? '')
    setDescription(subjectValue?.description ?? '')
    setStatus(subject?.status ?? 'draft')
    setEffectiveDateFrom(subject?.effectiveDateFrom ?? '2026-01-01')
    setEffectiveDateTo(subject?.effectiveDateTo ?? '')
    setRequestedEffectiveDate(subject?.effectiveDateFrom ?? '2026-01-01')
    setResult(undefined)
    setValidationError('')
  }, [subject?.id, subject?.version, changeType])

  const selectSubjectType = (nextType: 'definition' | 'value') => {
    setChangeType(nextType)
    const candidates = nextType === 'definition' ? definitions : values
    setSubjectId(candidates[0]?.id ?? '')
    setHasUnsavedChanges(false)
  }

  const save = async () => {
    setValidationError('')
    if (!currentScope || !subject) {
      setNotice('Select an accounting scope and an existing definition or value before submitting a change request.')
      return
    }
    if (!isUuid(approvalRequestId)) {
      setValidationError('Enter the valid Workflow approval-request reference supplied by the approval process.')
      return
    }
    if (!requestedEffectiveDate || !effectiveDateFrom || (effectiveDateTo && effectiveDateTo < effectiveDateFrom)) {
      setValidationError('Enter a requested effective date and a valid inclusive proposed effective-date range.')
      return
    }
    if (!statusTransitionIsValid(subject.status, status)) {
      setValidationError(`The lifecycle transition from ${subject.status} to ${status} is not allowed.`)
      return
    }
    if (changeType === 'definition' && (!segmentType.trim() || !code.trim() || !name.trim())) {
      setValidationError('Enter segment type, code, and name for the proposed definition change.')
      return
    }
    if (changeType === 'value' && (!value.trim() || !description.trim())) {
      setValidationError('Enter value and description for the proposed value change.')
      return
    }
    if (changeType === 'value' && (!subjectDefinition || effectiveDateFrom < subjectDefinition.effectiveDateFrom || (subjectDefinition.effectiveDateTo && (!effectiveDateTo || effectiveDateTo > subjectDefinition.effectiveDateTo)))) {
      setValidationError('The value effective interval must be contained within the parent definition interval.')
      return
    }
    setSaving(true)
    setNotice('Submitting the idempotent segment-change request…')
    try {
      const response = await coaRequestSegmentChanges({
        body: {
          commandId: crypto.randomUUID(),
          accountingScopeId: currentScope.id,
          data: {
            changeType,
            subjectId: subject.id,
            subjectVersion: subject.version,
            requestedEffectiveDate,
            approvalRequestId: approvalRequestId.trim(),
            proposedChange: changeType === 'definition' ? {
              action: 'request', segmentDefinitionId: subject.id, segmentType: segmentType.trim(), code: code.trim(), name: name.trim(), status,
              effectiveDateFrom, effectiveDateTo: effectiveDateTo || undefined,
            } : {
              action: 'request', segmentValueId: subject.id, segmentDefinitionId: subjectValue?.segmentDefinitionId ?? '', value: value.trim(), description: description.trim(), status,
              effectiveDateFrom, effectiveDateTo: effectiveDateTo || undefined,
            },
          },
        },
        headers: { 'Idempotency-Key': crypto.randomUUID() },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      const safe = requestProjection(response.data?.data?.segmentChangeRequest)
      if (!safe) {
        setNotice('The COA command returned no safe segment-change request projection.')
        return
      }
      setResult(safe)
      setHasUnsavedChanges(false)
      setNotice(`Created segment-change request ${safe.id}. The ${safe.approvalStatus} approval state is recorded; the subject remains unchanged until Story 5 applies a Workflow decision.`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The segment-change request could not reach the live API.')
    } finally {
      setSaving(false)
    }
  }

  const subjectLabel = changeType === 'definition' ? subjectDefinition ? `${subjectDefinition.name} (${subjectDefinition.code})` : 'No definition available' : subjectValue ? `${subjectValue.value} — ${subjectValue.description}` : 'No value available'
  return <section aria-labelledby="coa-change-request-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">COA-SCR-04</p><h2 id="coa-change-request-title" className="mt-1 text-2xl font-semibold">Segment change request</h2><p className="mt-2 max-w-3xl text-base-content/75">Submit a governed definition or value proposal with the current source version and the Workflow approval request that will own the decision.</p></div><RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-ws-01">Back to worklist</RouterLink></div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter. Mutation source: live <code>coaRequestSegmentChanges</code> API. This screen creates a COA request only; it does not mutate the subject or create a Workflow approval policy.</p>
    <Panel title="Request identity and approval handoff" description="The approval-request reference must be supplied by the Workflow-owned process. Definition, value, combination, and assignment are not interchangeable subject types.">
      <div className="grid gap-4 md:grid-cols-2">
        <div className="form-control w-full gap-2"><label className="label" htmlFor="coa-change-type"><span className="font-medium text-base-content">Subject type</span><span aria-hidden="true" className="text-error">*</span></label><Select id="coa-change-type" aria-required="true" value={changeType} onChange={(event) => selectSubjectType(event.target.value as 'definition' | 'value')}><option value="definition">Segment definition</option><option value="value">Segment value</option></Select></div>
        <div className="form-control w-full gap-2"><label className="label" htmlFor="coa-change-subject"><span className="font-medium text-base-content">Existing subject</span><span aria-hidden="true" className="text-error">*</span></label><Select id="coa-change-subject" aria-required="true" value={subjectId} onChange={(event) => setSubjectId(event.target.value)}><option value="">Select a subject</option>{(changeType === 'definition' ? definitions : values).map((candidate) => <option key={candidate.id} value={candidate.id}>{changeType === 'definition' ? `${(candidate as SegmentDefinitionRecord).name} — ${(candidate as SegmentDefinitionRecord).code}` : `${(candidate as SegmentValueRecord).value} — ${(candidate as SegmentValueRecord).description}`}</option>)}</Select></div>
        <Field id="coa-change-approval-request" label="Workflow approval-request reference" value={approvalRequestId} onChange={(event) => { setApprovalRequestId(event.target.value); setHasUnsavedChanges(true) }} placeholder="UUID supplied by Workflow" required description="COA validates and records this reference; Workflow owns its lifecycle and decision." />
        <Field id="coa-change-subject-version" label="Source subject version" value={`v${subject?.version ?? 0}`} readOnly />
        <Field id="coa-change-subject-identity" label="Subject identity" value={subject?.id ?? 'Select a subject'} readOnly />
        <Field id="coa-change-subject-label" label="Subject" value={subjectLabel} readOnly />
        <Field id="coa-change-requested-date" label="Requested effective date" type="date" value={requestedEffectiveDate} onChange={(event) => { setRequestedEffectiveDate(event.target.value); setHasUnsavedChanges(true) }} required />
        <Field id="coa-change-scope" label="Accounting scope" value={currentScope?.id ?? 'Select a scope'} readOnly />
      </div>
    </Panel>
    <Panel title="Proposed change" description="The proposal reuses the existing maintenance fields. It is stored as a safe request snapshot and is applied only after an approved decision is handled by COA.">
      <div className="grid gap-4 md:grid-cols-2">
        {changeType === 'definition' ? <><Field id="coa-change-segment-type" label="Segment type" value={segmentType} onChange={(event) => { setSegmentType(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="coa-change-code" label="Code" value={code} onChange={(event) => { setCode(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="coa-change-name" label="Name" value={name} onChange={(event) => { setName(event.target.value); setHasUnsavedChanges(true) }} required /></> : <><Field id="coa-change-value" label="Value" value={value} onChange={(event) => { setValue(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="coa-change-description" label="Description" value={description} onChange={(event) => { setDescription(event.target.value); setHasUnsavedChanges(true) }} required /></>}
        <div className="form-control w-full gap-2"><label className="label cursor-pointer justify-start gap-2" htmlFor="coa-change-status"><span className="font-medium text-base-content">Proposed lifecycle status</span><span aria-hidden="true" className="text-error">*</span></label><Select id="coa-change-status" aria-required="true" value={status} onChange={(event) => { setStatus(event.target.value as SegmentDefinitionStatus); setHasUnsavedChanges(true) }}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></div>
        <Field id="coa-change-effective-from" label="Proposed effective from" type="date" value={effectiveDateFrom} onChange={(event) => { setEffectiveDateFrom(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="coa-change-effective-to" label="Proposed effective to" type="date" value={effectiveDateTo} onChange={(event) => { setEffectiveDateTo(event.target.value); setHasUnsavedChanges(true) }} description="Leave blank for an open-ended range only when the subject rules allow it." />
      </div>
      {validationError ? <div role="alert" className="mt-4"><ValidationSummary issues={[{ id: 'coa-change-request', category: 'field', code: 'invalid', message: validationError, targetId: 'coa-change-subject', targetLabel: 'Segment change request' }]} /></div> : null}
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={() => void save()} loading={saving} disabled={!subject}>{changeType === 'definition' ? 'Request definition change' : 'Request value change'}</Button><RouterLink className="btn btn-ghost min-h-11" to="/coa-segments/coa-ws-01">Cancel</RouterLink></div>
    </Panel>
    {result ? <Panel title="Request established" description="This safe projection records the COA request reference and handoff state. The subject configuration was not changed."><div className="grid gap-4 md:grid-cols-2"><Field id="coa-change-request-id" label="Segment-change request reference" value={result.id} readOnly /><Field id="coa-change-request-approval-status" label="Approval status" value={result.approvalStatus} readOnly /><Field id="coa-change-request-application-status" label="Application status" value={result.applicationStatus} readOnly /><Field id="coa-change-request-next-action" label="Next action" value={result.nextAction} readOnly /><Field id="coa-change-request-version" label="Request version" value={`v${result.version}`} readOnly /><Field id="coa-change-request-impact" label="Impacted-record information" value="Subject snapshot captured; impact analysis is not present in the approved v1 request contract." readOnly /></div>{result.conflictCode || result.rejectionReason ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-4 text-sm">{result.conflictCode ?? result.rejectionReason}</p> : null}</Panel> : null}
  </section>
}
