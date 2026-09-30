import { useEffect, useMemo, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'

import { coaMaintainSegmentDefinitions } from '@/generated/api/sdk.gen'
import { Button, DataTable, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

export type SegmentDefinitionStatus = 'draft' | 'active' | 'suspended' | 'retired'

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

const safeReadAdapter = new Map(initialSegmentDefinitions.map((record) => [record.id, record]))

export function readSafeSegmentDefinitions(scopeId: string): SegmentDefinitionRecord[] {
  return Array.from(safeReadAdapter.values())
    .filter((record) => record.scopeId === scopeId)
    .sort((left, right) => left.code.localeCompare(right.code))
}

function upsertSafeSegmentDefinition(record: SegmentDefinitionRecord) {
  safeReadAdapter.set(record.id, record)
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
  const rows = useMemo(() => {
    const query = search.trim().toLowerCase()
    if (!query) return records
    return records.filter((record) => [record.segmentType, record.code, record.name, record.status].some((value) => value.toLowerCase().includes(query)))
  }, [records, search])
  const columns = useMemo(() => [
    { key: 'definition', header: 'Segment definition', rowHeader: true, render: (record: SegmentDefinitionRecord) => <RouterLink className="link link-primary font-semibold" to={`/coa-segments/coa-scr-01?segmentDefinitionId=${record.id}`}>{record.name}</RouterLink> },
    { key: 'type', header: 'Type', render: (record: SegmentDefinitionRecord) => record.segmentType },
    { key: 'code', header: 'Code', render: (record: SegmentDefinitionRecord) => <span className="font-mono">{record.code}</span> },
    { key: 'status', header: 'State', render: (record: SegmentDefinitionRecord) => <StatusBadge state={statusState[record.status]} label={segmentStatusLabel(record.status)} /> },
    { key: 'effective', header: 'Effective interval', render: (record: SegmentDefinitionRecord) => `${record.effectiveDateFrom} — ${record.effectiveDateTo ?? 'open'}` },
    { key: 'version', header: 'Version', render: (record: SegmentDefinitionRecord) => `v${record.version}` },
    { key: 'nextAction', header: 'Next action', render: (record: SegmentDefinitionRecord) => record.nextAction },
  ] as const, [])

  return <section aria-labelledby="coa-worklist-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">COA-WS-01</p><h2 id="coa-worklist-title" className="mt-1 text-2xl font-semibold">Segment administration worklist</h2><p className="mt-2 max-w-3xl text-base-content/75">Review safe local/read-adapter projections for the selected accounting scope. Authoritative changes are submitted only from the segment-definition record.</p></div>
      <RouterLink className="btn btn-primary min-h-11 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/coa-segments/coa-scr-01?new=true">Create segment definition</RouterLink>
    </div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter with synthetic records. No approved COA read endpoint exists yet; successful live mutations update this adapter with the returned safe projection.</p>
    <Panel title="Definitions in the selected scope" description="Segment type and code must remain unique across overlapping inclusive effective-date ranges.">
      <div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-end"><Field id="coa-segment-search" label="Search segment definitions" placeholder="Type, code, or name" value={search} onChange={(event) => setSearch(event.target.value)} /><p className="text-sm text-base-content/70" role="status" aria-live="polite">{rows.length} record{rows.length === 1 ? '' : 's'} found.</p></div>
      <div className="mt-6"><DataTable caption="Safe COA segment-definition projections" columns={columns} rows={rows} getRowKey={(record) => record.id} emptyMessage="No segment definitions match this scope and search." /></div>
    </Panel>
  </section>
}

export function CoaSegmentDefinitionRecord() {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [params, setParams] = useSearchParams()
  const navigate = useNavigate()
  const isNew = params.get('new') === 'true'
  const selected = currentScope ? readSafeSegmentDefinitions(currentScope.id).find((record) => record.id === params.get('segmentDefinitionId')) : undefined
  const initial = selected ?? (isNew ? undefined : currentScope ? readSafeSegmentDefinitions(currentScope.id)[0] : undefined)
  const [record, setRecord] = useState<SegmentDefinitionRecord | undefined>(initial)
  const [segmentType, setSegmentType] = useState(initial?.segmentType ?? 'department')
  const [code, setCode] = useState(initial?.code ?? '')
  const [name, setName] = useState(initial?.name ?? '')
  const [status, setStatus] = useState<SegmentDefinitionStatus>(initial?.status ?? 'draft')
  const [effectiveDateFrom, setEffectiveDateFrom] = useState(initial?.effectiveDateFrom ?? '2026-01-01')
  const [effectiveDateTo, setEffectiveDateTo] = useState(initial?.effectiveDateTo ?? '')
  const [notice, setNotice] = useState('Review the current safe projection and aggregate version before submitting a material change.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  useEffect(() => {
    setRecord(initial)
    setSegmentType(initial?.segmentType ?? 'department')
    setCode(initial?.code ?? '')
    setName(initial?.name ?? '')
    setStatus(initial?.status ?? 'draft')
    setEffectiveDateFrom(initial?.effectiveDateFrom ?? '2026-01-01')
    setEffectiveDateTo(initial?.effectiveDateTo ?? '')
  }, [initial?.id, initial?.version, currentScope?.id])

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
    <Panel title="History and boundary" description="Established facts retain the COA source version used when they were created."><p className="text-sm text-base-content/75">Current safe projection: version {record?.version ?? 0}, revision {record?.revisionNumber ?? 0}. Segment values, combinations, assignments, approval workflows, and downstream GL effects are outside this story.</p></Panel>
  </section>
}
