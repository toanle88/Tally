import { useMemo, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'

import { coaMaintainSegmentDefinitions, coaMaintainSegmentValues, coaValidateSegmentCombinations } from '@/generated/api/sdk.gen'
import { Button, DataTable, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { ValidationSummary } from '@/components/validation-summary'
import type { ValidationIssue } from '@/components/workflow-context'
import { useScopeContext } from '@/lib/scope/scope-context'

export type SegmentDefinitionStatus = 'draft' | 'active' | 'suspended' | 'retired'
export type SegmentValueStatus = SegmentDefinitionStatus

export type SegmentDefinitionRecord = {
  id: string
  scopeId: string
  segmentType: string
  code: string
  name: string
  status: SegmentDefinitionStatus
  effectiveDateFrom: string
  effectiveDateTo?: string
  approvalStatus: 'not-required'
  validationOutcome: 'valid' | 'invalid'
  nextAction: string
  version: number
  revisionNumber: number
}

export type SegmentValueRecord = {
  id: string
  segmentDefinitionId: string
  scopeId: string
  value: string
  description: string
  status: SegmentValueStatus
  effectiveDateFrom: string
  effectiveDateTo?: string
  approvalStatus: 'not-required'
  validationOutcome: 'valid' | 'invalid'
  nextAction: string
  version: number
  revisionNumber: number
}

export type SegmentChangeRequestRecord = {
  id: string
  scopeId: string
  changeType: 'definition' | 'value'
  subjectId: string
  subjectVersion: number
  requestedEffectiveDate: string
  approvalRequestId: string
  proposedFingerprint?: string
  approvalDecisionId?: string
  approvalPolicyVersion?: string
  approvalDecisionVersion?: number
  approvalSubjectVersion?: number
  approvalCandidateFingerprint?: string
  approvalApproverUserId?: string
  approvalDecidedAt?: string
  approvalAppliedAt?: string
  resultingSubjectVersion?: number
  appliedSubjectVersion?: number
  effectiveDateResult?: string
  approvalStatus: string
  applicationStatus: string
  validationOutcome: string
  conflictCode?: string
  rejectionReason?: string
  nextAction: string
  proposedChange: Record<string, unknown>
  version: number
  revisionNumber: number
}

const initialSegmentDefinitions: SegmentDefinitionRecord[] = [
  {
    id: 'segment-department-operations',
    scopeId: 'scope-vietnam-statutory',
    segmentType: 'department',
    code: 'D-001',
    name: 'Operations',
    status: 'active',
    effectiveDateFrom: '2026-01-01',
    approvalStatus: 'not-required',
    validationOutcome: 'valid',
    nextAction: 'maintain',
    version: 3,
    revisionNumber: 3,
  },
  {
    id: 'segment-department-shared-services',
    scopeId: 'scope-vietnam-statutory',
    segmentType: 'department',
    code: 'D-002',
    name: 'Shared Services',
    status: 'draft',
    effectiveDateFrom: '2026-07-01',
    effectiveDateTo: '2026-12-31',
    approvalStatus: 'not-required',
    validationOutcome: 'valid',
    nextAction: 'maintain',
    version: 1,
    revisionNumber: 1,
  },
  {
    id: 'segment-cost-centre-singapore',
    scopeId: 'scope-singapore-management',
    segmentType: 'cost-centre',
    code: 'CC-100',
    name: 'Regional Management',
    status: 'suspended',
    effectiveDateFrom: '2026-01-01',
    approvalStatus: 'not-required',
    validationOutcome: 'valid',
    nextAction: 'maintain',
    version: 2,
    revisionNumber: 2,
  },
]

const initialSegmentValues: SegmentValueRecord[] = [
  {
    id: 'segment-value-operations-1000',
    segmentDefinitionId: 'segment-department-operations',
    scopeId: 'scope-vietnam-statutory',
    value: '1000',
    description: 'Operations',
    status: 'active',
    effectiveDateFrom: '2026-01-01',
    approvalStatus: 'not-required',
    validationOutcome: 'valid',
    nextAction: 'maintain',
    version: 3,
    revisionNumber: 3,
  },
  {
    id: 'segment-value-operations-2000',
    segmentDefinitionId: 'segment-department-operations',
    scopeId: 'scope-vietnam-statutory',
    value: '2000',
    description: 'Secondary operations',
    status: 'active',
    effectiveDateFrom: '2026-01-01',
    approvalStatus: 'not-required',
    validationOutcome: 'valid',
    nextAction: 'maintain',
    version: 3,
    revisionNumber: 3,
  },
]

const safeReadAdapter = new Map(initialSegmentDefinitions.map((record) => [record.id, record]))
const safeValueReadAdapter = new Map(initialSegmentValues.map((record) => [record.id, record]))
const safeChangeRequestReadAdapter = new Map<string, SegmentChangeRequestRecord>()

export function readSafeSegmentDefinitions(scopeId: string): SegmentDefinitionRecord[] {
  return Array.from(safeReadAdapter.values())
    .filter((record) => record.scopeId === scopeId)
    .sort((left, right) => left.code.localeCompare(right.code))
}

function upsertSafeSegmentDefinition(record: SegmentDefinitionRecord) {
  safeReadAdapter.set(record.id, record)
  safeValueReadAdapter.forEach((value) => {
    if (value.segmentDefinitionId === record.id) {
      safeValueReadAdapter.set(value.id, { ...value, scopeId: record.scopeId, version: record.version, revisionNumber: record.revisionNumber })
    }
  })
}

export function readSafeSegmentValues(scopeId: string, segmentDefinitionId?: string): SegmentValueRecord[] {
  return Array.from(safeValueReadAdapter.values())
    .filter((record) => record.scopeId === scopeId && (!segmentDefinitionId || record.segmentDefinitionId === segmentDefinitionId))
    .sort((left, right) => left.value.localeCompare(right.value) || left.effectiveDateFrom.localeCompare(right.effectiveDateFrom))
}

export function readSafeSegmentChangeRequests(scopeId: string): SegmentChangeRequestRecord[] {
  return Array.from(safeChangeRequestReadAdapter.values()).filter((record) => record.scopeId === scopeId).sort((left, right) => right.requestedEffectiveDate.localeCompare(left.requestedEffectiveDate) || left.id.localeCompare(right.id))
}

export function upsertSafeSegmentChangeRequest(record: SegmentChangeRequestRecord) {
  safeChangeRequestReadAdapter.set(record.id, { ...record, proposedChange: { ...record.proposedChange } })
}

function upsertSafeSegmentValue(record: SegmentValueRecord) {
  safeValueReadAdapter.set(record.id, record)
}

const statusState: Record<SegmentDefinitionStatus, SemanticState> = {
  draft: 'pending',
  active: 'success',
  suspended: 'warning',
  retired: 'disabled',
}

function segmentStatusLabel(status: SegmentDefinitionStatus) {
  return status.replace('-', ' ')
}

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') {
    return error.detail
  }
  return 'The COA mutation could not be completed. Review the current version and retry safely.'
}

function statusTransitionIsValid(current: SegmentDefinitionStatus | undefined, next: SegmentDefinitionStatus) {
  if (!current || current === next) return true
  if (current === 'draft') return next === 'active' || next === 'retired'
  if (current === 'active') return next === 'suspended' || next === 'retired'
  if (current === 'suspended') return next === 'active' || next === 'retired'
  return false
}

function rangesOverlap(leftFrom: string, leftTo: string, rightFrom: string, rightTo: string) {
  const leftEnd = leftTo || '9999-12-31'
  const rightEnd = rightTo || '9999-12-31'
  return leftFrom <= rightEnd && rightFrom <= leftEnd
}

export function CoaSegmentWorklist() {
  const { currentScope } = useScopeContext()
  const [search, setSearch] = useState('')
  const records = useMemo(() => currentScope ? readSafeSegmentDefinitions(currentScope.id) : [], [currentScope?.id])
  const changeRequests = useMemo(() => currentScope ? readSafeSegmentChangeRequests(currentScope.id) : [], [currentScope?.id])
  const rows = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (!query) return records
    return records.filter((record) => [record.segmentType, record.code, record.name, record.status].some((value) => value.toLowerCase().includes(query)) || readSafeSegmentValues(record.scopeId, record.id).some((value) => [value.value, value.description, value.status].some((field) => field.toLowerCase().includes(query))))
  }, [records, search])
  const columns = useMemo(() => [
    { key: 'definition', header: 'Segment definition', rowHeader: true, render: (record: SegmentDefinitionRecord) => <RouterLink className="link link-primary font-semibold" to={`/coa-segments/coa-scr-01?segmentDefinitionId=${record.id}`}>{record.name}</RouterLink> },
    { key: 'type', header: 'Type', render: (record: SegmentDefinitionRecord) => record.segmentType },
    { key: 'code', header: 'Code', render: (record: SegmentDefinitionRecord) => <span className="font-mono">{record.code}</span> },
    { key: 'status', header: 'State', render: (record: SegmentDefinitionRecord) => <StatusBadge state={statusState[record.status]} label={segmentStatusLabel(record.status)} /> },
    { key: 'effective', header: 'Effective interval', render: (record: SegmentDefinitionRecord) => `${record.effectiveDateFrom} — ${record.effectiveDateTo ?? 'open'}` },
    { key: 'values', header: 'Values', render: (record: SegmentDefinitionRecord) => { const values = readSafeSegmentValues(record.scopeId, record.id); return values.length ? <div className="flex flex-wrap gap-x-3 gap-y-1">{values.map((value) => <RouterLink key={value.id} className="link link-primary font-mono" to={`/coa-segments/coa-scr-02?segmentDefinitionId=${record.id}&segmentValueId=${value.id}`}>{value.value}</RouterLink>)}</div> : <RouterLink className="link link-primary" to={`/coa-segments/coa-scr-02?segmentDefinitionId=${record.id}&new=true`}>Add value</RouterLink> } },
    { key: 'version', header: 'Version', render: (record: SegmentDefinitionRecord) => `v${record.version}` },
    { key: 'nextAction', header: 'Next action', render: (record: SegmentDefinitionRecord) => record.nextAction },
  ] as const, [])

  return <section aria-labelledby="coa-worklist-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">COA-WS-01</p><h2 id="coa-worklist-title" className="mt-1 text-2xl font-semibold">Segment administration worklist</h2><p className="mt-2 max-w-3xl text-base-content/75">Review safe local/read-adapter projections for the selected accounting scope. Open a definition or one of its values to maintain an authoritative record.</p></div>
      <div className="flex flex-wrap gap-3"><RouterLink className="btn btn-outline min-h-11 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-scr-03">Validate combination</RouterLink><RouterLink className="btn btn-outline min-h-11 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-scr-04">Request segment change</RouterLink><RouterLink className="btn btn-primary min-h-11 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-scr-01?new=true">Create segment definition</RouterLink></div>
    </div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter with synthetic records. No approved COA read endpoint exists yet; successful live mutations update this adapter with the returned safe projection.</p>
    <Panel title="Definitions and values in the selected scope" description="Segment type and code must remain unique across overlapping inclusive effective-date ranges. Values are unique within a parent definition across overlapping inclusive ranges.">
      <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-end"><Field id="coa-segment-search" label="Search segment definitions" placeholder="Type, code, name, or value" value={search} onChange={(event) => setSearch(event.target.value)} /><p className="text-sm text-base-content/70" role="status" aria-live="polite">{rows.length} record{rows.length === 1 ? '' : 's'} found.</p></div>
      <div className="mt-6"><DataTable caption="Safe COA segment-definition projections" columns={columns} rows={rows} getRowKey={(record) => record.id} emptyMessage="No segment definitions match this scope and search." /></div>
    </Panel>
    <Panel title="Governed segment-change decisions" description="Safe local/read-adapter projections returned by the request and approval-application commands. No restricted Workflow values are shown.">
      {changeRequests.length ? <ul className="divide-y divide-base-300" aria-label="Governed segment-change decisions">{changeRequests.map((request) => <li key={request.id} className="flex flex-wrap items-center justify-between gap-4 py-4"><div><RouterLink className="link link-primary font-semibold" to={`/coa-segments/coa-scr-04?requestId=${request.id}&changeType=${request.changeType}&subjectId=${request.subjectId}`}>{request.id}</RouterLink><p className="text-sm text-base-content/70">{request.changeType} · subject v{request.subjectVersion} · requested {request.requestedEffectiveDate}</p></div><div className="text-right text-sm"><p>{request.approvalStatus} · {request.applicationStatus}</p><p className="text-base-content/70">Effective-date result: {request.effectiveDateResult ?? 'awaiting decision'}</p><p className="text-base-content/70">Next action: {request.nextAction}</p></div></li>)}</ul> : <p className="text-sm text-base-content/70">No governed segment-change decisions are present in this local safe adapter.</p>}
    </Panel>
  </section>
}

export function CoaSegmentDefinitionRecord() {
  const { currentScope } = useScopeContext()
  const [params] = useSearchParams()
  const isNew = params.get('new') === 'true'
  const selected = currentScope ? readSafeSegmentDefinitions(currentScope.id).find((record) => record.id === params.get('segmentDefinitionId')) : undefined
  const initial = selected ?? (isNew ? undefined : currentScope ? readSafeSegmentDefinitions(currentScope.id)[0] : undefined)
  const [notice, setNotice] = useState('Review the current safe projection and aggregate version before submitting a material change.')

  return <CoaSegmentDefinitionRecordForm
    key={`${currentScope?.id ?? 'none'}:${initial?.id ?? 'new'}:${initial?.version ?? 0}`}
    initial={initial}
    notice={notice}
    setNotice={setNotice}
  />
}

type CoaSegmentDefinitionRecordFormProps = {
  initial?: SegmentDefinitionRecord
  notice: string
  setNotice: (notice: string) => void
}

function CoaSegmentDefinitionRecordForm({ initial, notice, setNotice }: CoaSegmentDefinitionRecordFormProps) {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [, setParams] = useSearchParams()
  const navigate = useNavigate()
  const [record, setRecord] = useState<SegmentDefinitionRecord | undefined>(initial)
  const [segmentType, setSegmentType] = useState(initial?.segmentType ?? 'department')
  const [code, setCode] = useState(initial?.code ?? '')
  const [name, setName] = useState(initial?.name ?? '')
  const [status, setStatus] = useState<SegmentDefinitionStatus>(initial?.status ?? 'draft')
  const [effectiveDateFrom, setEffectiveDateFrom] = useState(initial?.effectiveDateFrom ?? '2026-01-01')
  const [effectiveDateTo, setEffectiveDateTo] = useState(initial?.effectiveDateTo ?? '')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setValidationError('')
    if (!currentScope) {
      setNotice('Select an accounting scope before submitting a segment definition.')
      return
    }
    const normalizedType = segmentType.trim()
    const normalizedCode = code.trim()
    const normalizedName = name.trim()
    if (!normalizedType || !normalizedCode || !normalizedName || !effectiveDateFrom || (effectiveDateTo && effectiveDateTo < effectiveDateFrom)) {
      setValidationError('Enter segment type, code, name, and a valid inclusive effective-date range.')
      return
    }
    if (!statusTransitionIsValid(record?.status, status)) {
      setValidationError(`The lifecycle transition from ${record?.status} to ${status} is not allowed.`)
      return
    }
    const overlap = readSafeSegmentDefinitions(currentScope.id).some((candidate) => candidate.id !== record?.id && candidate.segmentType === normalizedType && candidate.code === normalizedCode && rangesOverlap(effectiveDateFrom, effectiveDateTo, candidate.effectiveDateFrom, candidate.effectiveDateTo ?? ''))
    if (overlap) {
      setValidationError('Another segment definition uses this type and code in an overlapping effective-date range.')
      return
    }
    setSaving(true)
    setNotice('Submitting the idempotent COA command…')
    try {
      const response = await coaMaintainSegmentDefinitions({
        body: {
          commandId: crypto.randomUUID(),
          expectedVersion: record?.version,
          accountingScopeId: currentScope.id,
          data: {
            action: record ? 'update' : 'create',
            segmentDefinitionId: record?.id,
            segmentType: normalizedType,
            code: normalizedCode,
            name: normalizedName,
            status,
            effectiveDateFrom,
            effectiveDateTo: effectiveDateTo || undefined,
          },
        },
        headers: {
          'Idempotency-Key': crypto.randomUUID(),
          'If-Match': record ? `"${record.version}"` : undefined,
        },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      const projection = response.data?.data?.segmentDefinition
      if (!projection || typeof projection !== 'object') {
        setNotice('The COA command returned no safe segment-definition projection.')
        return
      }
      const safe = projection as Partial<SegmentDefinitionRecord>
      if (typeof safe.id !== 'string' || typeof safe.scopeId !== 'string' || typeof safe.version !== 'number' || typeof safe.revisionNumber !== 'number') {
        setNotice('The COA command returned an invalid safe projection.')
        return
      }
      const nextRecord: SegmentDefinitionRecord = {
        id: safe.id,
        scopeId: safe.scopeId,
        segmentType: typeof safe.segmentType === 'string' ? safe.segmentType : normalizedType,
        code: typeof safe.code === 'string' ? safe.code : normalizedCode,
        name: typeof safe.name === 'string' ? safe.name : normalizedName,
        status: safe.status === 'active' || safe.status === 'suspended' || safe.status === 'retired' ? safe.status : 'draft',
        effectiveDateFrom: typeof safe.effectiveDateFrom === 'string' ? safe.effectiveDateFrom.slice(0, 10) : effectiveDateFrom,
        effectiveDateTo: typeof safe.effectiveDateTo === 'string' ? safe.effectiveDateTo.slice(0, 10) : effectiveDateTo || undefined,
        approvalStatus: 'not-required',
        validationOutcome: 'valid',
        nextAction: typeof safe.nextAction === 'string' ? safe.nextAction : 'maintain',
        version: safe.version,
        revisionNumber: safe.revisionNumber,
      }
      upsertSafeSegmentDefinition(nextRecord)
      setRecord(nextRecord)
      setHasUnsavedChanges(false)
      setNotice(`Accepted segment-definition revision v${nextRecord.version}. Safe local/read-adapter state was refreshed from the live mutation response.`)
      if (!record) {
        setParams({ segmentDefinitionId: nextRecord.id }, { replace: true })
      }
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The COA command could not reach the live API.')
    } finally {
      setSaving(false)
    }
  }

  const disabled = record?.status === 'retired'
  return <section aria-labelledby="coa-record-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">COA-SCR-01</p><h2 id="coa-record-title" className="mt-1 text-2xl font-semibold">Segment definition</h2><p className="mt-2 max-w-3xl text-base-content/75">Maintain type, code, name, lifecycle status, and inclusive effective dates. Established revisions retain their source version.</p></div><RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-ws-01">Back to worklist</RouterLink></div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter. Mutation source: live `coaMaintainSegmentDefinitions` API. The adapter is not an authoritative COA read model.</p>
    <Panel title="Authoritative identity and lifecycle" description="The live command enforces authorization, idempotency, optimistic concurrency, audit evidence, and scoped effective-date uniqueness.">
      <div className="grid gap-4 md:grid-cols-2">
        <Field id="coa-segment-type" label="Segment type" value={segmentType} onChange={(event) => { setSegmentType(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <Field id="coa-segment-code" label="Code" value={code} onChange={(event) => { setCode(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <Field id="coa-segment-name" label="Name" value={name} onChange={(event) => { setName(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <div className="form-control w-full gap-2"><label className="label cursor-pointer justify-start gap-2" htmlFor="coa-segment-status"><span className="font-medium text-base-content">Lifecycle status</span><span aria-hidden="true" className="text-error">*</span></label><Select id="coa-segment-status" aria-required="true" value={status} onChange={(event) => { setStatus(event.target.value as SegmentDefinitionStatus); setHasUnsavedChanges(true) }} disabled={disabled}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></div>
        <Field id="coa-segment-effective-from" label="Effective from" type="date" value={effectiveDateFrom} onChange={(event) => { setEffectiveDateFrom(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <Field id="coa-segment-effective-to" label="Effective to" type="date" value={effectiveDateTo} onChange={(event) => { setEffectiveDateTo(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} description="Leave blank for an open-ended range." />
        <Field id="coa-segment-id" label="Record identifier" value={record?.id ?? 'New definition'} readOnly />
        <Field id="coa-segment-scope" label="Accounting scope" value={currentScope?.id ?? 'Select a scope'} readOnly />
      </div>
      {validationError ? <div role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-4 text-sm text-error">{validationError}</div> : null}
      <div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[status]} label={segmentStatusLabel(status)} announce /><span className="text-sm text-base-content/70">Version {record?.version ?? 0} · Approval: {record?.approvalStatus ?? 'not-required'} · Revision {record?.revisionNumber ?? 0} · Next action: {record?.nextAction ?? 'create'}</span></div>
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={() => void save()} loading={saving} disabled={disabled}>{record ? 'Save segment-definition revision' : 'Create segment definition'}</Button>{record ? <Button variant="ghost" onClick={() => navigate('/coa-segments/coa-ws-01')}>Cancel</Button> : null}</div>
    </Panel>
    <Panel title="History and boundary" description="Established facts retain the COA source version used when they were created."><p className="text-sm text-base-content/75">Current safe projection: version {record?.version ?? 0}, revision {record?.revisionNumber ?? 0}. Values are maintained as children of this segment-definition aggregate.</p>{record ? <div className="mt-3 flex flex-wrap gap-4"><RouterLink className="link link-primary" to={`/coa-segments/coa-scr-02?segmentDefinitionId=${record.id}`}>Maintain values for this definition</RouterLink><RouterLink className="link link-primary" to={`/coa-segments/coa-scr-04?changeType=definition&subjectId=${record.id}`}>Request a governed change</RouterLink></div> : null}</Panel>
  </section>
}

export function CoaSegmentValueRecord() {
  const { currentScope } = useScopeContext()
  const [params] = useSearchParams()
  const isNew = params.get('new') === 'true'
  const requestedDefinitionId = params.get('segmentDefinitionId')
  const selectedValue = currentScope ? readSafeSegmentValues(currentScope.id, requestedDefinitionId || undefined).find((record) => record.id === params.get('segmentValueId')) : undefined
  const parent = currentScope ? readSafeSegmentDefinitions(currentScope.id).find((record) => record.id === (selectedValue?.segmentDefinitionId ?? requestedDefinitionId)) ?? readSafeSegmentDefinitions(currentScope.id)[0] : undefined
  const initial = selectedValue ?? (isNew ? undefined : parent ? readSafeSegmentValues(parent.scopeId, parent.id)[0] : undefined)
  const [notice, setNotice] = useState('Review the parent definition and aggregate version before submitting a material value change.')

  return <CoaSegmentValueRecordForm
    key={`${currentScope?.id ?? 'none'}:${parent?.id ?? 'none'}:${initial?.id ?? 'new'}:${initial?.version ?? 0}`}
    initial={initial}
    parent={parent}
    notice={notice}
    setNotice={setNotice}
  />
}

type CoaSegmentValueRecordFormProps = {
  initial?: SegmentValueRecord
  parent?: SegmentDefinitionRecord
  notice: string
  setNotice: (notice: string) => void
}

function CoaSegmentValueRecordForm({ initial, parent, notice, setNotice }: CoaSegmentValueRecordFormProps) {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [, setParams] = useSearchParams()
  const navigate = useNavigate()
  const [record, setRecord] = useState<SegmentValueRecord | undefined>(initial)
  const [value, setValue] = useState(initial?.value ?? '')
  const [description, setDescription] = useState(initial?.description ?? '')
  const [status, setStatus] = useState<SegmentValueStatus>(initial?.status ?? 'draft')
  const [effectiveDateFrom, setEffectiveDateFrom] = useState(initial?.effectiveDateFrom ?? parent?.effectiveDateFrom ?? '2026-01-01')
  const [effectiveDateTo, setEffectiveDateTo] = useState(initial?.effectiveDateTo ?? '')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setValidationError('')
    if (!currentScope || !parent) {
      setNotice('Select an accounting scope and parent segment definition before submitting a segment value.')
      return
    }
    const normalizedValue = value.trim()
    const normalizedDescription = description.trim()
    if (!normalizedValue || !effectiveDateFrom || effectiveDateTo && effectiveDateTo < effectiveDateFrom) {
      setValidationError('Enter a value and a valid inclusive effective-date range.')
      return
    }
    if (effectiveDateFrom < parent.effectiveDateFrom || parent.effectiveDateTo && (!effectiveDateTo || effectiveDateTo > parent.effectiveDateTo)) {
      setValidationError('The value effective interval must be contained within the parent definition interval.')
      return
    }
    if (!statusTransitionIsValid(record?.status, status)) {
      setValidationError(`The lifecycle transition from ${record?.status} to ${status} is not allowed.`)
      return
    }
    const overlap = readSafeSegmentValues(currentScope.id, parent.id).some((candidate) => candidate.id !== record?.id && candidate.value.trim() === normalizedValue && rangesOverlap(effectiveDateFrom, effectiveDateTo, candidate.effectiveDateFrom, candidate.effectiveDateTo ?? ''))
    if (overlap) {
      setValidationError('Another value uses this normalized value in an overlapping effective-date range for the parent definition.')
      return
    }
    setSaving(true)
    setNotice('Submitting the idempotent COA value command…')
    try {
      const response = await coaMaintainSegmentValues({
        body: {
          commandId: crypto.randomUUID(),
          expectedVersion: parent.version,
          accountingScopeId: currentScope.id,
          data: {
            action: record ? 'update' : 'create',
            segmentDefinitionId: parent.id,
            segmentValueId: record?.id,
            value: normalizedValue,
            description: normalizedDescription,
            status,
            effectiveDateFrom,
            effectiveDateTo: effectiveDateTo || undefined,
          },
        },
        headers: {
          'Idempotency-Key': crypto.randomUUID(),
          'If-Match': `"${parent.version}"`,
        },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      const projection = response.data?.data?.segmentValue
      if (!projection || typeof projection !== 'object') {
        setNotice('The COA command returned no safe segment-value projection.')
        return
      }
      const safe = projection as Partial<SegmentValueRecord>
      if (typeof safe.id !== 'string' || typeof safe.segmentDefinitionId !== 'string' || typeof safe.scopeId !== 'string' || typeof safe.version !== 'number' || typeof safe.revisionNumber !== 'number') {
        setNotice('The COA command returned an invalid safe segment-value projection.')
        return
      }
      const nextRecord: SegmentValueRecord = {
        id: safe.id,
        segmentDefinitionId: safe.segmentDefinitionId,
        scopeId: safe.scopeId,
        value: typeof safe.value === 'string' ? safe.value : normalizedValue,
        description: typeof safe.description === 'string' ? safe.description : normalizedDescription,
        status: safe.status === 'active' || safe.status === 'suspended' || safe.status === 'retired' ? safe.status : 'draft',
        effectiveDateFrom: typeof safe.effectiveDateFrom === 'string' ? safe.effectiveDateFrom.slice(0, 10) : effectiveDateFrom,
        effectiveDateTo: typeof safe.effectiveDateTo === 'string' ? safe.effectiveDateTo.slice(0, 10) : effectiveDateTo || undefined,
        approvalStatus: 'not-required',
        validationOutcome: 'valid',
        nextAction: typeof safe.nextAction === 'string' ? safe.nextAction : 'maintain',
        version: safe.version,
        revisionNumber: safe.revisionNumber,
      }
      upsertSafeSegmentValue(nextRecord)
      const currentParent = readSafeSegmentDefinitions(currentScope.id).find((candidate) => candidate.id === parent.id)
      if (currentParent) upsertSafeSegmentDefinition({ ...currentParent, version: nextRecord.version, revisionNumber: nextRecord.revisionNumber })
      setRecord(nextRecord)
      setHasUnsavedChanges(false)
      setNotice(`Accepted segment-value revision v${nextRecord.version}. Safe local/read-adapter state was refreshed from the live mutation response.`)
      if (!record) {
        setParams({ segmentDefinitionId: nextRecord.segmentDefinitionId, segmentValueId: nextRecord.id }, { replace: true })
      }
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The COA command could not reach the live API.')
    } finally {
      setSaving(false)
    }
  }

  const disabled = record?.status === 'retired' || parent?.status === 'retired'
  return <section aria-labelledby="coa-value-record-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">COA-SCR-02</p><h2 id="coa-value-record-title" className="mt-1 text-2xl font-semibold">Segment value</h2><p className="mt-2 max-w-3xl text-base-content/75">Maintain a value and description inside the parent definition’s inclusive effective interval. The parent aggregate version is required for every mutation.</p></div><div className="flex flex-wrap gap-4"><RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to={parent ? `/coa-segments/coa-scr-01?segmentDefinitionId=${parent.id}` : '/coa-segments/coa-ws-01'}>Parent definition</RouterLink><RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-ws-01">Back to worklist</RouterLink></div></div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter. Mutation source: live `coaMaintainSegmentValues` API. The adapter is not an authoritative COA read model.</p>
    <Panel title="Value identity and lifecycle" description="The live command enforces parent-scope authorization, idempotency, optimistic concurrency, audit evidence, parent interval containment, and scoped value uniqueness.">
      <div className="grid gap-4 md:grid-cols-2">
        <Field id="coa-segment-value" label="Value" value={value} onChange={(event) => { setValue(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <Field id="coa-segment-value-description" label="Description" value={description} onChange={(event) => { setDescription(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <div className="form-control w-full gap-2"><label className="label cursor-pointer justify-start gap-2" htmlFor="coa-segment-value-status"><span className="font-medium text-base-content">Lifecycle status</span><span aria-hidden="true" className="text-error">*</span></label><Select id="coa-segment-value-status" aria-required="true" value={status} onChange={(event) => { setStatus(event.target.value as SegmentValueStatus); setHasUnsavedChanges(true) }} disabled={disabled}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></div>
        <Field id="coa-segment-value-effective-from" label="Effective from" type="date" value={effectiveDateFrom} onChange={(event) => { setEffectiveDateFrom(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} required />
        <Field id="coa-segment-value-effective-to" label="Effective to" type="date" value={effectiveDateTo} onChange={(event) => { setEffectiveDateTo(event.target.value); setHasUnsavedChanges(true) }} disabled={disabled} description="Leave blank only when the parent definition is open-ended." />
        <Field id="coa-segment-value-id" label="Record identifier" value={record?.id ?? 'New value'} readOnly />
        <Field id="coa-segment-value-parent" label="Parent definition" value={parent ? `${parent.name} (${parent.code})` : 'Select a parent definition'} readOnly />
        <Field id="coa-segment-value-scope" label="Accounting scope" value={currentScope?.id ?? 'Select a scope'} readOnly />
        <Field id="coa-segment-value-parent-version" label="Parent aggregate version" value={`v${parent?.version ?? 0}`} readOnly />
      </div>
      {validationError ? <div role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-4 text-sm text-error">{validationError}</div> : null}
      <div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[status]} label={segmentStatusLabel(status)} announce /><span className="text-sm text-base-content/70">Version {record?.version ?? parent?.version ?? 0} · Approval: {record?.approvalStatus ?? 'not-required'} · Revision {record?.revisionNumber ?? parent?.revisionNumber ?? 0} · Next action: {record?.nextAction ?? 'create'}</span></div>
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={() => void save()} loading={saving} disabled={disabled || !parent}>{record ? 'Save segment-value revision' : 'Create segment value'}</Button>{record ? <Button variant="ghost" onClick={() => navigate(`/coa-segments/coa-scr-01?segmentDefinitionId=${record.segmentDefinitionId}`)}>Cancel</Button> : null}</div>
    </Panel>
    <Panel title="History and boundary" description="A value mutation establishes a new revision of the owning segment-definition aggregate."><p className="text-sm text-base-content/75">The parent definition interval is the boundary for this value. Historical non-overlapping ranges are allowed; established value facts are corrected through a new lifecycle or effective-date revision.</p>{record ? <RouterLink className="link link-primary mt-3 inline-block" to={`/coa-segments/coa-scr-04?changeType=value&subjectId=${record.id}`}>Request a governed change</RouterLink> : null}</Panel>
  </section>
}

type CombinationValidationIssuePayload = {
  segmentDefinitionId?: string
  segmentValueId?: string
  reason?: string
}

type CombinationValidationView = {
  validationStatus: 'valid' | 'invalid'
  effectiveDateResult: string
  sourceVersions: Array<{ segmentDefinitionId?: string; segmentValueId?: string; segmentDefinitionVersion?: number; segmentDefinitionRevision?: number }>
  invalidValues: CombinationValidationIssuePayload[]
  restrictions: string[]
  rejectionReasons: CombinationValidationIssuePayload[]
  nextAction: string
  replayed?: boolean
}

function combinationPayload(value: unknown): CombinationValidationView | undefined {
  if (!value || typeof value !== 'object') return undefined
  const candidate = value as Record<string, unknown>
  const status = candidate.validationStatus === 'valid' ? 'valid' : candidate.validationStatus === 'invalid' ? 'invalid' : undefined
  if (!status) return undefined
  const arrayOfIssues = (entry: unknown): CombinationValidationIssuePayload[] => Array.isArray(entry) ? entry.filter((item): item is CombinationValidationIssuePayload => Boolean(item && typeof item === 'object')).map((item) => {
    const issue = item as Record<string, unknown>
    return {
      segmentDefinitionId: typeof issue.segmentDefinitionId === 'string' ? issue.segmentDefinitionId : undefined,
      segmentValueId: typeof issue.segmentValueId === 'string' ? issue.segmentValueId : undefined,
      reason: typeof issue.reason === 'string' ? issue.reason : undefined,
    }
  }) : []
  const sourceVersions = Array.isArray(candidate.sourceVersions) ? candidate.sourceVersions.filter((item): item is Record<string, unknown> => Boolean(item && typeof item === 'object')).map((item) => ({
    segmentDefinitionId: typeof item.segmentDefinitionId === 'string' ? item.segmentDefinitionId : undefined,
    segmentValueId: typeof item.segmentValueId === 'string' ? item.segmentValueId : undefined,
    segmentDefinitionVersion: typeof item.segmentDefinitionVersion === 'number' ? item.segmentDefinitionVersion : undefined,
    segmentDefinitionRevision: typeof item.segmentDefinitionRevision === 'number' ? item.segmentDefinitionRevision : undefined,
  })) : []
  return {
    validationStatus: status,
    effectiveDateResult: typeof candidate.effectiveDateResult === 'string' ? candidate.effectiveDateResult : 'not-effective',
    sourceVersions,
    invalidValues: arrayOfIssues(candidate.invalidValues),
    restrictions: Array.isArray(candidate.restrictions) ? candidate.restrictions.filter((entry): entry is string => typeof entry === 'string') : [],
    rejectionReasons: arrayOfIssues(candidate.rejectionReasons),
    nextAction: typeof candidate.nextAction === 'string' ? candidate.nextAction : 'correct-and-revalidate',
    replayed: candidate.replayed === true,
  }
}

function combinationErrorIssue(message: string, category: ValidationIssue['category'] = 'dependency'): ValidationIssue {
  return { id: 'combination-request', category, code: 'validation-request', message, targetId: 'coa-combination-date', targetLabel: 'Validation request', nextAction: 'Review the current source state and retry with the same or a new idempotency identity.' }
}

function initialCombinationSelections(definitions: SegmentDefinitionRecord[]) {
  const selectedDefinitions: Record<string, boolean> = {}
  const selectedValues: Record<string, string> = {}
  definitions.forEach((definition) => {
    const values = readSafeSegmentValues(definition.scopeId, definition.id)
    selectedDefinitions[definition.id] = values.length > 0
    if (values[0]) selectedValues[definition.id] = values[0].id
  })
  return { selectedDefinitions, selectedValues }
}

export function CoaSegmentCombinationValidator() {
  const { currentScope } = useScopeContext()
  const definitions = useMemo(() => currentScope ? readSafeSegmentDefinitions(currentScope.id) : [], [currentScope?.id])
  return <CoaSegmentCombinationValidatorForm
    key={currentScope?.id ?? 'none'}
    scopeId={currentScope?.id}
    definitions={definitions}
  />
}

type CoaSegmentCombinationValidatorFormProps = {
  scopeId?: string
  definitions: SegmentDefinitionRecord[]
}

function CoaSegmentCombinationValidatorForm({ scopeId, definitions }: CoaSegmentCombinationValidatorFormProps) {
  const initialSelections = initialCombinationSelections(definitions)
  const [selectedDefinitions, setSelectedDefinitions] = useState<Record<string, boolean>>(initialSelections.selectedDefinitions)
  const [selectedValues, setSelectedValues] = useState<Record<string, string>>(initialSelections.selectedValues)
  const [businessDate, setBusinessDate] = useState('2026-01-01')
  const [result, setResult] = useState<CombinationValidationView | undefined>()
  const [localIssues, setLocalIssues] = useState<ValidationIssue[]>([])
  const [notice, setNotice] = useState('Select the proposed segment values and requested effective date before validating.')
  const [validating, setValidating] = useState(false)

  const selected = useMemo(() => definitions.filter((definition) => selectedDefinitions[definition.id]), [definitions, selectedDefinitions])
  const resultIssues = useMemo<ValidationIssue[]>(() => {
    if (!result) return []
    return result.rejectionReasons.map((issue, index) => {
      const targetId = issue.segmentDefinitionId ? `coa-combination-value-${issue.segmentDefinitionId}` : 'coa-combination-date'
      const definition = issue.segmentDefinitionId ? definitions.find((candidate) => candidate.id === issue.segmentDefinitionId) : undefined
      return {
        id: `combination-${index}-${issue.segmentDefinitionId ?? 'request'}`,
        category: 'business-rule',
        code: 'combination-restriction',
        message: issue.reason ?? 'The proposed combination does not satisfy a COA validation rule.',
        targetId,
        targetLabel: definition?.name ?? 'Proposed combination',
        nextAction: result.nextAction,
      }
    })
  }, [definitions, result])
  const issues = localIssues.length ? localIssues : resultIssues

  const focusIssue = (targetId: string) => {
    document.getElementById(targetId)?.focus()
  }

  const validate = async () => {
    setLocalIssues([])
    setResult(undefined)
    if (!scopeId) {
      setLocalIssues([combinationErrorIssue('Select an accounting scope before validating a segment combination.', 'authorization')])
      return
    }
    const chosen = selected.map((definition) => ({ definition, valueId: selectedValues[definition.id] }))
    const nextIssues: ValidationIssue[] = []
    if (!businessDate) nextIssues.push({ id: 'combination-date', category: 'field', code: 'required', message: 'Enter the requested effective date.', targetId: 'coa-combination-date', targetLabel: 'Requested effective date' })
    if (chosen.length === 0) nextIssues.push({ id: 'combination-values', category: 'field', code: 'required', message: 'Select at least one segment definition and value.', targetId: 'coa-combination-definitions', targetLabel: 'Proposed segment values' })
    chosen.forEach(({ definition, valueId }) => {
      if (!valueId) nextIssues.push({ id: `combination-value-${definition.id}`, category: 'field', code: 'required', message: 'Select a value for this definition.', targetId: `coa-combination-value-${definition.id}`, targetLabel: definition.name })
    })
    if (nextIssues.length) {
      setLocalIssues(nextIssues)
      return
    }

    setValidating(true)
    setNotice('Validating the proposed combination against the current COA source snapshot…')
    try {
      const response = await coaValidateSegmentCombinations({
        body: {
          commandId: crypto.randomUUID(),
          accountingScopeId: scopeId,
          businessDate,
          data: {
            segmentValues: chosen.map(({ definition, valueId }) => ({ segmentDefinitionId: definition.id, segmentValueId: valueId })),
          },
        },
        headers: { 'Idempotency-Key': crypto.randomUUID() },
      })
      if (response.error) {
        setLocalIssues([combinationErrorIssue(errorDetail(response.error), response.error.code === 'AUTHORIZATION_DENIED' ? 'authorization' : response.error.code === 'IDEMPOTENCY_CONFLICT' || response.error.code === 'COMMAND_IN_PROGRESS' ? 'conflict' : 'dependency')])
        setNotice('The validation request did not establish a result. Review the issue and retry safely.')
        return
      }
      const nextResult = combinationPayload(response.data?.data)
      if (!nextResult) {
        setLocalIssues([combinationErrorIssue('The validation response did not contain a safe established result.')])
        setNotice('The validation response was incomplete.')
        return
      }
      setResult(nextResult)
      setNotice(nextResult.validationStatus === 'valid' ? 'The proposed combination is valid for the requested date.' : 'The proposed combination is invalid. Review each rejection reason before relying on it.')
    } catch (error) {
      setLocalIssues([combinationErrorIssue(error instanceof Error ? error.message : 'The validation request could not reach the live API.')])
      setNotice('The validation request could not be completed.')
    } finally {
      setValidating(false)
    }
  }

  const status = result?.validationStatus
  return <section aria-labelledby="coa-combination-validator-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">COA-SCR-03</p><h2 id="coa-combination-validator-title" className="mt-1 text-2xl font-semibold">Segment combination validator</h2><p className="mt-2 max-w-3xl text-base-content/75">Check whether a proposed set of segment values is allowed and effective before another business action relies on it.</p></div><RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-ws-01">Back to worklist</RouterLink></div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter with synthetic records. Validation source: live <code>coaValidateSegmentCombinations</code> API. Validation is read-only and does not reserve or change authoritative COA state.</p>
    <Panel title="Proposed segment values" description="Select the definitions that belong to this proposed combination, then choose one value for each selected definition.">
      <fieldset id="coa-combination-definitions" className="space-y-4"><legend className="font-medium text-base-content">Segment definitions</legend>{definitions.map((definition) => {
        const values = readSafeSegmentValues(definition.scopeId, definition.id)
        const selectedDefinition = Boolean(selectedDefinitions[definition.id])
        return <div key={definition.id} className="rounded-box border border-base-300 p-4">
          <label className="flex items-start gap-3">
            <input type="checkbox" className="checkbox mt-1" checked={selectedDefinition} onChange={(event) => { setSelectedDefinitions((current) => ({ ...current, [definition.id]: event.target.checked })); setResult(undefined) }} />
            <span><span className="font-medium">{definition.name}</span><span className="ml-2 font-mono text-sm text-base-content/70">{definition.code}</span><span className="block text-sm text-base-content/70">{definition.status} · v{definition.version} · {definition.effectiveDateFrom} — {definition.effectiveDateTo ?? 'open'}</span></span>
          </label>
          {selectedDefinition ? <div className="mt-3">
            <label className="label" htmlFor={`coa-combination-value-${definition.id}`}><span className="font-medium text-base-content">{definition.name} value</span><span aria-hidden="true" className="text-error">*</span></label>
            <Select id={`coa-combination-value-${definition.id}`} aria-required="true" value={selectedValues[definition.id] ?? ''} onChange={(event) => { setSelectedValues((current) => ({ ...current, [definition.id]: event.target.value })); setResult(undefined) }}><option value="">Select a value</option>{values.map((value) => <option key={value.id} value={value.id}>{value.value} — {value.description} ({value.status})</option>)}</Select>
          </div> : null}
        </div>
      })}</fieldset>
      <div className="mt-5 max-w-sm"><Field id="coa-combination-date" label="Requested effective date" type="date" value={businessDate} onChange={(event) => { setBusinessDate(event.target.value); setResult(undefined) }} required /></div>
      {issues.length ? <div className="mt-5"><ValidationSummary issues={issues} onFocusTarget={focusIssue} /></div> : null}
      <div className="mt-5 flex flex-wrap items-center gap-3"><Button onClick={() => void validate()} loading={validating}>Validate combination</Button><StatusBadge state={status === 'valid' ? 'success' : status === 'invalid' ? 'error' : 'info'} label={status ? `${status} · ${result?.effectiveDateResult ?? 'not-effective'}` : 'not validated'} announce /></div>
    </Panel>
    {result ? <Panel title="Validation result" description="This established result is a read-only assessment of the source snapshot used for validation.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="coa-combination-result-status" label="Validation status" value={result.validationStatus} readOnly /><Field id="coa-combination-result-date" label="Effective-date result" value={result.effectiveDateResult} readOnly /><Field id="coa-combination-next-action" label="Next action" value={result.nextAction} readOnly /><Field id="coa-combination-replay" label="Result identity" value={result.replayed ? 'Replayed established result' : 'New established result'} readOnly /></div>
      <div className="mt-5 grid gap-5 md:grid-cols-2"><div><h3 className="font-medium">Restrictions</h3>{result.restrictions.length ? <ul className="mt-2 list-disc space-y-1 pl-5 text-sm">{result.restrictions.map((restriction) => <li key={restriction}>{restriction}</li>)}</ul> : <p className="mt-2 text-sm text-base-content/70">No v1 restrictions were reported.</p>}</div><div><h3 className="font-medium">Source versions</h3><ul className="mt-2 space-y-1 text-sm">{result.sourceVersions.map((source) => <li key={`${source.segmentDefinitionId}-${source.segmentValueId}`} className="font-mono">{source.segmentDefinitionId} / {source.segmentValueId}: v{source.segmentDefinitionVersion ?? 0}, revision {source.segmentDefinitionRevision ?? 0}</li>)}</ul></div></div>
      {result.invalidValues.length ? <div className="mt-5"><h3 className="font-medium">Invalid values</h3><ul className="mt-2 list-disc space-y-1 pl-5 text-sm">{result.invalidValues.map((issue, index) => <li key={`${issue.segmentValueId ?? issue.segmentDefinitionId}-${index}`}>{issue.segmentValueId ?? issue.segmentDefinitionId}: {issue.reason}</li>)}</ul></div> : null}
    </Panel> : null}
  </section>
}
