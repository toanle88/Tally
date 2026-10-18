import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'

import { invConfigureBillingSchedules, invGetBillingSchedule, invListBillingSchedules } from '@/generated/api/sdk.gen'
import type { InvConfigureBillingSchedulesCommandRequest } from '@/generated/api/types.gen'
import { Button, DataTable, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { ValidationSummary } from '@/components/validation-summary'
import { VersionConflictDialog } from '@/components/version-conflict-dialog'
import type { EstablishedResultLink, OperationalOwner } from '@/components/operational-context'
import type { ScopeReference } from '@/components/record-context'
import type { ValidationIssue, VersionConflictSnapshot } from '@/components/workflow-context'
import { useScopeContext } from '@/lib/scope/scope-context'
import { InvoiceTemplateWorklist } from '@/app/invoice-template-workspace'

type ScheduledChargeProjection = {
  id: string
  templateId: string
  templateVersion: number
  templateLineItemId: string
  quantity: string
  amount: string
  currency: string
}

type BillingScheduleProjection = {
  id: string
  accountingScopeId: string
  customerId: string
  contractReference: string
  charges: ScheduledChargeProjection[]
  nextBillingDate: string
  timeZone: string
  status: 'configured'
  effectiveFrom: string
  effectiveTo?: string
  version: number
  revisionNumber: number
  revisions?: Array<{ revisionNumber: number; capturedAt: string; fingerprint: string; version: number }>
  validationOutcome?: string
}

type ScheduleFormChargeState = {
	 id: string
	 templateId: string
	 templateVersion: string
	 templateLineItemId: string
	 quantity: string
	 amount: string
	 currency: string
}

type ScheduleFormState = {
  customerId: string
  contractReference: string
	 charges: ScheduleFormChargeState[]
  nextBillingDate: string
  timeZone: string
  effectiveFrom: string
  effectiveTo: string
}

const exactDecimalPattern = /^(0|[1-9][0-9]*)(\.[0-9]+)?$/
const decimalInputPattern = /^\d+(?:\.\d+)?$/
let fallbackIdentityCounter = 0

function newClientIdentity() {
	const cryptoObject = globalThis.crypto
	if (cryptoObject?.randomUUID) return cryptoObject.randomUUID()
	const bytes = new Uint8Array(16)
	if (cryptoObject?.getRandomValues) {
		cryptoObject.getRandomValues(bytes)
	} else {
		const seed = `${Date.now()}-${fallbackIdentityCounter++}-${Math.random()}`
		for (let index = 0; index < bytes.length; index += 1) {
			bytes[index] = (seed.charCodeAt(index % seed.length) + index * 31 + Math.floor(Math.random() * 256)) & 0xff
		}
	}
	bytes[6] = (bytes[6] & 0x0f) | 0x40
	bytes[8] = (bytes[8] & 0x3f) | 0x80
	return Array.from(bytes, (value) => value.toString(16).padStart(2, '0')).join('').replace(/^(.{8})(.{4})(.{4})(.{4})(.{12})$/, '$1-$2-$3-$4-$5')
}

function emptyChargeForm(): ScheduleFormChargeState {
	return { id: newClientIdentity(), templateId: '', templateVersion: '1', templateLineItemId: '', quantity: '1', amount: '', currency: 'USD' }
}

function emptyForm(): ScheduleFormState {
  return {
		customerId: '', contractReference: '', charges: [emptyChargeForm()], nextBillingDate: '', timeZone: 'Asia/Ho_Chi_Minh', effectiveFrom: '', effectiveTo: '',
  }
}

function asRecord(value: unknown): Record<string, unknown> | null {
  return value !== null && typeof value === 'object' && !Array.isArray(value) ? value as Record<string, unknown> : null
}

function stringValue(value: unknown): string | undefined {
  return typeof value === 'string' ? value : undefined
}

function numberValue(value: unknown): number | undefined {
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function integerValue(value: unknown): number | undefined {
	return typeof value === 'number' && Number.isInteger(value) ? value : undefined
}

function dateInputValue(value: unknown) {
  const text = stringValue(value)
  return text ? text.slice(0, 10) : ''
}

function canonicalDecimal(value: string) {
  const trimmed = value.trim()
  if (!decimalInputPattern.test(trimmed)) return trimmed
  const [integerPart, fractionPart] = trimmed.split('.')
  const integer = integerPart.replace(/^0+(?=\d)/, '')
  const fraction = fractionPart?.replace(/0+$/, '')
  return fraction ? `${integer}.${fraction}` : integer
}

function parseCharge(value: unknown): ScheduledChargeProjection | null {
  const record = asRecord(value)
  const id = stringValue(record?.id)
  const templateId = stringValue(record?.templateId)
	const templateVersion = integerValue(record?.templateVersion)
  const templateLineItemId = stringValue(record?.templateLineItemId)
  const quantity = stringValue(record?.quantity)
  const amount = stringValue(record?.amount)
  const currency = stringValue(record?.currency)
  if (!id || !templateId || templateVersion === undefined || !templateLineItemId || !quantity || !amount || !currency) return null
  return { id, templateId, templateVersion, templateLineItemId, quantity, amount, currency }
}

function parseProjection(value: unknown): BillingScheduleProjection | null {
  const record = asRecord(value)
  const id = stringValue(record?.id)
  const accountingScopeId = stringValue(record?.accountingScopeId)
  const customerId = stringValue(record?.customerId)
  const contractReference = stringValue(record?.contractReference)
	const rawCharges = record?.charges
	if (!Array.isArray(rawCharges) || rawCharges.length === 0) return null
	const parsedCharges = rawCharges.map(parseCharge)
	if (parsedCharges.some((charge) => charge === null)) return null
	const charges = parsedCharges as ScheduledChargeProjection[]
  const nextBillingDate = stringValue(record?.nextBillingDate)
  const timeZone = stringValue(record?.timeZone)
  const status = stringValue(record?.status)
  const effectiveFrom = stringValue(record?.effectiveFrom)
	const version = integerValue(record?.version)
	const revisionNumber = integerValue(record?.revisionNumber)
	if ((record?.effectiveTo !== undefined && typeof record.effectiveTo !== 'string') || (record?.validationOutcome !== undefined && typeof record.validationOutcome !== 'string')) return null
	if (!id || !accountingScopeId || !customerId || !contractReference || !nextBillingDate || !timeZone || status !== 'configured' || !effectiveFrom || version === undefined || revisionNumber === undefined) return null
	let revisions: BillingScheduleProjection['revisions']
	if (record?.revisions !== undefined) {
		if (!Array.isArray(record.revisions)) return null
		const parsedRevisions = record.revisions.map((revision) => {
			const item = asRecord(revision)
			const revisionNumber = integerValue(item?.revisionNumber)
			const capturedAt = stringValue(item?.capturedAt)
			const fingerprint = stringValue(item?.fingerprint)
			const version = integerValue(item?.version)
			return revisionNumber !== undefined && capturedAt && fingerprint && version !== undefined ? { revisionNumber, capturedAt, fingerprint, version } : null
		})
		if (parsedRevisions.some((revision) => revision === null)) return null
		revisions = parsedRevisions as NonNullable<BillingScheduleProjection['revisions']>
	}
  return { id, accountingScopeId, customerId, contractReference, charges, nextBillingDate, timeZone, status, effectiveFrom, effectiveTo: stringValue(record?.effectiveTo), version, revisionNumber, revisions, validationOutcome: stringValue(record?.validationOutcome) ?? 'valid' }
}

function projectionToForm(schedule: BillingScheduleProjection): ScheduleFormState {
  return {
		customerId: schedule.customerId, contractReference: schedule.contractReference,
		charges: schedule.charges.map((charge) => ({ id: charge.id, templateId: charge.templateId, templateVersion: String(charge.templateVersion), templateLineItemId: charge.templateLineItemId, quantity: charge.quantity, amount: charge.amount, currency: charge.currency })),
		nextBillingDate: dateInputValue(schedule.nextBillingDate), timeZone: schedule.timeZone,
    effectiveFrom: dateInputValue(schedule.effectiveFrom), effectiveTo: dateInputValue(schedule.effectiveTo),
  }
}

function errorDetails(error: unknown) {
  const value = asRecord(error)
  return {
    status: numberValue(value?.status), code: stringValue(value?.code), detail: stringValue(value?.detail) ?? 'The billing-schedule operation could not be completed. Review the authoritative state and retry safely.',
    currentVersion: numberValue(value?.currentVersion),
  }
}

function scopeHeader(scopeId: string) {
  return { 'X-Accounting-Scope-Id': scopeId }
}

function validationIssue(id: string, targetId: string, targetLabel: string, message: string, category: ValidationIssue['category'] = 'field'): ValidationIssue {
  return { id, targetId, targetLabel, message, category, code: id, nextAction: 'Correct the value and review the other highlighted fields.' }
}

function validateForm(form: ScheduleFormState): ValidationIssue[] {
  const issues: ValidationIssue[] = []
  if (!form.customerId.trim()) issues.push(validationIssue('customer-required', 'billing-schedule-customer', 'Customer identity', 'Enter the authoritative customer reference.'))
  if (!form.contractReference.trim()) issues.push(validationIssue('contract-required', 'billing-schedule-contract', 'Contract reference', 'Enter the source contract reference.'))
	if (form.charges.length === 0) issues.push(validationIssue('charge-required', 'billing-schedule-add-charge', 'Scheduled charges', 'Add at least one template-linked scheduled charge.', 'business-rule'))
	form.charges.forEach((charge, index) => {
		const suffix = index === 0 ? '' : `-${index}`
		if (!charge.templateId.trim()) issues.push(validationIssue(`template-required${suffix}`, `billing-schedule-template${suffix}`, `Invoice template ${index + 1}`, 'Enter the source invoice-template identity.', 'dependency'))
		if (!charge.templateLineItemId.trim()) issues.push(validationIssue(`template-line-required${suffix}`, `billing-schedule-template-line${suffix}`, `Template line ${index + 1}`, 'Enter the source template-line identity.', 'dependency'))
		if (!/^\d+$/.test(charge.templateVersion.trim()) || Number(charge.templateVersion) < 1) issues.push(validationIssue(`template-version${suffix}`, `billing-schedule-template-version${suffix}`, `Template version ${index + 1}`, 'Enter a positive source template version.', 'dependency'))
		if (!isNonNegativeDecimal(charge.quantity) || Number(charge.quantity) <= 0) issues.push(validationIssue(`quantity-invalid${suffix}`, `billing-schedule-quantity${suffix}`, `Quantity ${index + 1}`, 'Enter a positive exact decimal quantity.', 'business-rule'))
		if (!isNonNegativeDecimal(charge.amount)) issues.push(validationIssue(`amount-invalid${suffix}`, `billing-schedule-amount${suffix}`, `Amount ${index + 1}`, 'Enter a non-negative exact decimal amount.', 'business-rule'))
		if (!/^[A-Z]{3}$/.test(charge.currency.trim().toUpperCase())) issues.push(validationIssue(`currency-invalid${suffix}`, `billing-schedule-currency${suffix}`, `Currency ${index + 1}`, 'Enter a three-letter currency code.', 'business-rule'))
	})
  if (!form.effectiveFrom) issues.push(validationIssue('effective-from-required', 'billing-schedule-effective-from', 'Effective from', 'Choose the first date on which this schedule applies.'))
  if (!form.nextBillingDate) issues.push(validationIssue('next-date-required', 'billing-schedule-next-date', 'Next billing date', 'Choose the next billing date.'))
  if (form.effectiveTo && form.effectiveTo < form.effectiveFrom) issues.push(validationIssue('effective-range', 'billing-schedule-effective-to', 'Effective to', 'The inclusive end date cannot precede the start date.', 'business-rule'))
  if (form.nextBillingDate && form.effectiveFrom && form.nextBillingDate < form.effectiveFrom) issues.push(validationIssue('next-date-range', 'billing-schedule-next-date', 'Next billing date', 'The next billing date cannot precede the effective start date.', 'business-rule'))
  if (form.effectiveTo && form.nextBillingDate > form.effectiveTo) issues.push(validationIssue('next-date-end', 'billing-schedule-next-date', 'Next billing date', 'The next billing date must be within the effective interval.', 'business-rule'))
  if (!form.timeZone.trim()) issues.push(validationIssue('timezone-required', 'billing-schedule-time-zone', 'IANA time zone', 'Enter an IANA time-zone identifier.'))
  else {
    try { new Intl.DateTimeFormat('en-US', { timeZone: form.timeZone.trim() }).format() } catch { issues.push(validationIssue('timezone-invalid', 'billing-schedule-time-zone', 'IANA time zone', 'Enter a valid IANA time-zone identifier.', 'business-rule')) }
  }
  return issues
}

function isNonNegativeDecimal(value: string) {
  return exactDecimalPattern.test(canonicalDecimal(value))
}

function scheduleStatusState(schedule: BillingScheduleProjection): SemanticState {
  return schedule.validationOutcome === 'valid' ? 'success' : 'warning'
}

function conflictSnapshot(scope: ScopeReference, schedule: BillingScheduleProjection | null, form: ScheduleFormState, error: unknown): VersionConflictSnapshot {
  const details = errorDetails(error)
  const expectedVersion = schedule?.version ?? 0
  const currentVersion = details.currentVersion ?? Math.max(expectedVersion + 1, 1)
  const record: EstablishedResultLink = { label: form.contractReference || 'Billing schedule', href: schedule ? `/invoicing/inv-scr-02?billingScheduleId=${schedule.id}` : '/invoicing/inv-ws-01' }
  const owner: OperationalOwner = { id: 'invoicing', label: 'Billing Operations', reference: 'finance.inv.configure.billing.schedules' }
  return {
    kind: 'version', attemptedAction: schedule ? 'Update billing-schedule version' : 'Create billing schedule', record, scope,
    expected: { version: expectedVersion, state: 'reviewed' }, current: { version: currentVersion, state: 'authoritative current' },
    changedValues: [
      { id: 'customer', label: 'Customer', reviewedValue: form.customerId, currentValue: schedule?.customerId ?? 'Unavailable until refresh' },
      { id: 'contract', label: 'Contract', reviewedValue: form.contractReference, currentValue: schedule?.contractReference ?? 'Unavailable until refresh' },
      { id: 'next-date', label: 'Next billing date', reviewedValue: form.nextBillingDate, currentValue: schedule ? dateInputValue(schedule.nextBillingDate) : 'Unavailable until refresh' },
    ], currentOwner: owner, processEpoch: `billing-schedule-version-${currentVersion}`, safeRetry: { permitted: false, reason: 'Refresh the authoritative schedule, compare the reviewed changes, then submit a newly reviewed version.' },
  }
}

export function BillingScheduleWorklist() {
  const { currentScope } = useScopeContext()
  const navigate = useNavigate()
  const [schedules, setSchedules] = useState<BillingScheduleProjection[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'unavailable'>('loading')
  const [notice, setNotice] = useState('Loading the authoritative billing-schedule worklist…')

  const load = useCallback(async () => {
    if (!currentScope) {
      setSchedules([]); setStatus('unavailable'); setNotice('Select an accounting scope before loading billing schedules.'); return
    }
    setStatus('loading'); setNotice('Loading the authoritative billing-schedule worklist…')
    try {
      const response = await invListBillingSchedules({ headers: scopeHeader(currentScope.id), query: { 'page[size]': 50 } })
      if (response.error) { setSchedules([]); setStatus('unavailable'); setNotice(errorDetails(response.error).detail); return }
		const rawSchedules = response.data?.data?.billingSchedules
		if (!Array.isArray(rawSchedules)) { setSchedules([]); setStatus('unavailable'); setNotice('The authoritative billing-schedule response was malformed. No local fixture records are shown.'); return }
		const parsedSchedules = rawSchedules.map(parseProjection)
		if (parsedSchedules.some((schedule) => schedule === null)) { setSchedules([]); setStatus('unavailable'); setNotice('The authoritative billing-schedule response contained an unusable record. No local fixture records are shown.'); return }
		const values = parsedSchedules as BillingScheduleProjection[]
      setSchedules(values); setStatus('ready'); setNotice(values.length > 0 ? `Loaded ${values.length} authoritative billing schedule${values.length === 1 ? '' : 's'} for ${currentScope.legalEntity.name}.` : 'No authoritative billing schedules are configured for this accounting scope.')
    } catch (error) { setSchedules([]); setStatus('unavailable'); setNotice(error instanceof Error ? error.message : 'The authoritative billing-schedule list is unavailable.') }
  }, [currentScope])

  useEffect(() => { void load() }, [load])

  const columns = useMemo(() => [
    { key: 'customer', header: 'Customer', rowHeader: true, render: (schedule: BillingScheduleProjection) => <RouterLink className="link link-primary font-semibold" to={`/invoicing/inv-scr-02?billingScheduleId=${schedule.id}`}>{schedule.customerId}</RouterLink> },
    { key: 'contract', header: 'Contract', render: (schedule: BillingScheduleProjection) => schedule.contractReference },
    { key: 'next-date', header: 'Next billing date', render: (schedule: BillingScheduleProjection) => `${dateInputValue(schedule.nextBillingDate)} (${schedule.timeZone})` },
    { key: 'charges', header: 'Charges', render: (schedule: BillingScheduleProjection) => `${schedule.charges.length} template-linked` },
    { key: 'version', header: 'Version', render: (schedule: BillingScheduleProjection) => `v${schedule.version} / r${schedule.revisionNumber}` },
    { key: 'status', header: 'Status', render: (schedule: BillingScheduleProjection) => <StatusBadge state={scheduleStatusState(schedule)} label={schedule.status} /> },
  ] as const, [])

  return <section aria-labelledby="billing-schedule-worklist-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">INV-WS-01</p><h1 id="billing-schedule-worklist-title" className="mt-1 text-2xl font-semibold">Billing operations worklist</h1><p className="mt-2 max-w-4xl text-base-content/75">Review authoritative customer billing schedules in the selected accounting scope. Schedule identity, source contract, next action, status, and retained versions come from Invoicing.</p></div>
    <Panel title="Selected accounting scope" description="Scope is sent on every read and mutation; client-side selection is never treated as an authorization decision."><dl className="grid gap-2 text-sm sm:grid-cols-2"><div><dt className="text-base-content/65">Legal entity</dt><dd className="font-medium">{currentScope?.legalEntity.name ?? 'Not selected'}</dd></div><div><dt className="text-base-content/65">Accounting book</dt><dd className="font-medium">{currentScope?.accountingBook.name ?? 'Not selected'}</dd></div><div><dt className="text-base-content/65">Owner</dt><dd className="font-medium">Billing Operations / Invoicing</dd></div><div><dt className="text-base-content/65">Permission</dt><dd className="font-medium">finance.inv.configure.billing.schedules</dd></div></dl></Panel>
    <Panel title="Customer billing schedules" description="Configured schedules retain immutable revision evidence; later maintenance does not rewrite a prior source version."><div className="flex flex-wrap items-center justify-between gap-3"><p role="status" aria-live="polite" className="text-sm text-base-content/75">{notice}</p><div className="flex flex-wrap gap-2"><Button variant="secondary" onClick={() => void load()} loading={status === 'loading'}>Refresh authoritative list</Button><Button onClick={() => navigate('/invoicing/inv-scr-02?new=true')}>Configure billing schedule</Button></div></div>{status === 'unavailable' ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">The authoritative billing-schedule list is unavailable or was denied. No local fixture records are shown.</p> : null}{status === 'ready' ? <div className="mt-4"><DataTable caption="Authoritative customer billing schedules" columns={columns} rows={schedules} getRowKey={(schedule) => schedule.id} emptyMessage="No billing schedules are configured for this scope." /></div> : null}</Panel>
  </section>
}

export function BillingScheduleEditor() {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const scheduleId = searchParams.get('billingScheduleId')
  const [record, setRecord] = useState<BillingScheduleProjection | null>(null)
  const [form, setForm] = useState<ScheduleFormState>(() => emptyForm())
  const [issues, setIssues] = useState<ValidationIssue[]>([])
  const [loading, setLoading] = useState(Boolean(scheduleId))
  const [saving, setSaving] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [notice, setNotice] = useState(scheduleId ? 'Loading the authoritative billing-schedule version…' : 'Enter the complete schedule configuration, then submit it for server validation.')
  const [conflict, setConflict] = useState<VersionConflictSnapshot | null>(null)
  const idempotencyKey = useRef<string | null>(null)

  const loadSchedule = useCallback(async () => {
    if (!currentScope || !scheduleId) {
      setLoading(false)
			if (!scheduleId) { setRecord(null); setForm(emptyForm()); idempotencyKey.current = null; setNotice('Create mode: the first established version will receive its server-owned schedule identity.') }
      return
    }
    setLoading(true); setLoadError('')
    try {
      const response = await invGetBillingSchedule({ path: { billingScheduleId: scheduleId }, headers: scopeHeader(currentScope.id) })
      if (response.error) { setLoadError(errorDetails(response.error).detail); setNotice('The editor could not load the authoritative billing-schedule version.'); return }
      const loaded = parseProjection(response.data?.data?.billingSchedule)
      if (!loaded) { setLoadError('The authoritative response did not contain a usable billing-schedule projection.'); return }
		setRecord(loaded); setForm(projectionToForm(loaded)); setIssues([]); idempotencyKey.current = null; setNotice(`Loaded customer ${loaded.customerId} at authoritative version ${loaded.version}; retained revisions: ${loaded.revisions?.length ?? loaded.revisionNumber}.`); setHasUnsavedChanges(false)
    } catch (error) { setLoadError(error instanceof Error ? error.message : 'The authoritative billing-schedule detail is unavailable.'); setNotice('The editor could not load the authoritative billing-schedule version.') } finally { setLoading(false) }
  }, [currentScope, scheduleId, setHasUnsavedChanges])

  useEffect(() => { void loadSchedule() }, [loadSchedule])

  const updateForm = useCallback((update: (current: ScheduleFormState) => ScheduleFormState) => { setForm(update); idempotencyKey.current = null; setHasUnsavedChanges(true) }, [setHasUnsavedChanges])
  const updateCharge = useCallback((index: number, update: (current: ScheduleFormChargeState) => ScheduleFormChargeState) => {
    updateForm((current) => ({ ...current, charges: current.charges.map((charge, chargeIndex) => chargeIndex === index ? update(charge) : charge) }))
  }, [updateForm])
  const addCharge = useCallback(() => { updateForm((current) => ({ ...current, charges: [...current.charges, emptyChargeForm()] })) }, [updateForm])
  const removeCharge = useCallback((index: number) => { updateForm((current) => ({ ...current, charges: current.charges.filter((_, chargeIndex) => chargeIndex !== index) })) }, [updateForm])
  const fieldError = (targetId: string) => issues.find((issue) => issue.targetId === targetId)?.message
  const focusTarget = (targetId: string) => { const target = document.getElementById(targetId); if (target instanceof HTMLElement) { target.focus(); target.scrollIntoView?.({ block: 'center' }) } }

  const submit = async () => {
    const localIssues = validateForm(form); setIssues(localIssues)
    if (localIssues.length > 0) { setNotice('Resolve the highlighted fields before sending the command.'); focusTarget(localIssues[0].targetId); return }
    if (!currentScope || (scheduleId && !record)) { setNotice('Load an authoritative billing schedule before submitting.'); return }
    const data: InvConfigureBillingSchedulesCommandRequest['data'] = {
      action: record ? 'update' : 'create', billingScheduleId: record?.id, customerId: form.customerId.trim(), contractReference: form.contractReference.trim(),
			charges: form.charges.map((charge) => ({ id: charge.id || newClientIdentity(), templateId: charge.templateId.trim(), templateVersion: Number(charge.templateVersion), templateLineItemId: charge.templateLineItemId.trim(), quantity: canonicalDecimal(charge.quantity), amount: canonicalDecimal(charge.amount), currency: charge.currency.trim().toUpperCase() })),
      nextBillingDate: form.nextBillingDate, timeZone: form.timeZone.trim(), status: 'configured', effectiveFrom: form.effectiveFrom, effectiveTo: form.effectiveTo || undefined,
    }
    const key = idempotencyKey.current ?? newClientIdentity(); idempotencyKey.current = key; setSaving(true); setNotice('Submitting the idempotent billing-schedule command…')
    try {
      const response = await invConfigureBillingSchedules({ body: { commandId: newClientIdentity(), expectedVersion: record?.version, accountingScopeId: currentScope.id, data }, headers: { ...scopeHeader(currentScope.id), 'Idempotency-Key': key, 'If-Match': record ? `"${record.version}"` : undefined } })
      if (response.error) {
        const details = errorDetails(response.error)
        if (details.status === 409 && (details.code === 'VERSION_CONFLICT' || details.code === 'SCOPE_CONFLICT')) setConflict(conflictSnapshot({ id: currentScope.id, label: currentScope.legalEntity.name }, record, form, response.error))
        setNotice(details.detail); return
      }
      const established = parseProjection(response.data?.data?.billingSchedule)
      if (!established) { setNotice('The established result did not contain a usable billing-schedule projection; refresh the authoritative detail before retrying.'); return }
      setRecord(established); setForm(projectionToForm(established)); setIssues([]); idempotencyKey.current = null; setHasUnsavedChanges(false); setNotice(response.data?.data?.replayed ? `The established schedule result was replayed safely at version ${established.version}.` : `Billing schedule for ${established.customerId} established at version ${established.version}.`)
      if (!scheduleId) navigate(`/invoicing/inv-scr-02?billingScheduleId=${established.id}`, { replace: true })
    } catch (error) { setNotice(error instanceof Error ? error.message : 'The billing-schedule command could not reach the live API. Retry with the same idempotency identity.') } finally { setSaving(false) }
  }

  const scopeLabel = currentScope?.legalEntity.name ?? 'No accounting scope selected'
  return <section aria-labelledby="billing-schedule-editor-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">INV-SCR-02</p><h1 id="billing-schedule-editor-title" className="mt-1 text-2xl font-semibold">{record ? 'Maintain customer billing schedule' : 'Configure customer billing schedule'}</h1><p className="mt-2 max-w-4xl text-base-content/75">Maintain an effective-dated, template-linked schedule for one accounting scope. Established versions are immutable historical facts; an update creates the next version after server authorization and validation.</p></div>
    <p role="status" aria-live="polite" className="rounded-box border border-base-300 bg-base-200/30 p-3 text-sm">{notice}</p>
    {loadError ? <p role="alert" className="rounded-box border border-error/40 bg-error/10 p-3 text-sm">{loadError} No local fixture was substituted.</p> : null}
    <Panel title="Authoritative record context" description="The displayed identity, version, scope, owner, validation state, and next action are loaded from or returned by the Invoicing API."><dl className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4"><div><dt className="text-base-content/65">Accounting scope</dt><dd className="font-medium">{scopeLabel}</dd></div><div><dt className="text-base-content/65">Schedule identity</dt><dd className="font-medium">{record?.id ?? 'Server assigned on create'}</dd></div><div><dt className="text-base-content/65">Version / revision</dt><dd className="font-medium">{record ? `v${record.version} / r${record.revisionNumber}` : 'New version'}</dd></div><div><dt className="text-base-content/65">Owner / permission</dt><dd className="font-medium">Billing Operations · finance.inv.configure.billing.schedules</dd></div><div><dt className="text-base-content/65">Status</dt><dd><StatusBadge state={record ? scheduleStatusState(record) : 'pending'} label={record?.status ?? 'configured after validation'} /></dd></div><div><dt className="text-base-content/65">Next action</dt><dd className="font-medium">{record ? 'Review and maintain when approved' : 'Complete and submit configuration'}</dd></div></dl>{record?.revisions?.length ? <p className="mt-3 text-sm text-base-content/70">Retained immutable revisions: {record.revisions.map((revision) => `v${revision.version} / r${revision.revisionNumber}`).join(', ')}.</p> : null}</Panel>
    <ValidationSummary issues={issues} onFocusTarget={focusTarget} />
    <Panel title="Schedule configuration" description="Charges are linked to a template and line-item source version. Decimal values remain strings end to end; the owning Invoicing module revalidates all references and boundaries.">
      <form noValidate onSubmit={(event) => { event.preventDefault(); void submit() }} className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2">
          <Field id="billing-schedule-customer" label="Customer reference" value={form.customerId} onChange={(event) => updateForm((current) => ({ ...current, customerId: event.target.value }))} error={fieldError('billing-schedule-customer')} required disabled={loading} />
          <Field id="billing-schedule-contract" label="Contract reference" value={form.contractReference} onChange={(event) => updateForm((current) => ({ ...current, contractReference: event.target.value }))} error={fieldError('billing-schedule-contract')} required disabled={loading} />
          <Field id="billing-schedule-effective-from" label="Effective from" type="date" value={form.effectiveFrom} onChange={(event) => updateForm((current) => ({ ...current, effectiveFrom: event.target.value }))} error={fieldError('billing-schedule-effective-from')} required disabled={loading} />
          <Field id="billing-schedule-effective-to" label="Effective to" type="date" value={form.effectiveTo} onChange={(event) => updateForm((current) => ({ ...current, effectiveTo: event.target.value }))} description="Leave blank for an open-ended interval." disabled={loading} />
          <Field id="billing-schedule-next-date" label="Next billing date" type="date" value={form.nextBillingDate} onChange={(event) => updateForm((current) => ({ ...current, nextBillingDate: event.target.value }))} error={fieldError('billing-schedule-next-date')} required disabled={loading} />
          <Field id="billing-schedule-time-zone" label="IANA time zone" value={form.timeZone} onChange={(event) => updateForm((current) => ({ ...current, timeZone: event.target.value }))} error={fieldError('billing-schedule-time-zone')} description="Example: Asia/Ho_Chi_Minh or UTC." required disabled={loading} />
          <label className="form-control w-full gap-2"><span className="label font-medium">Schedule status</span><Select id="billing-schedule-status" aria-label="Schedule status" value="configured" disabled><option value="configured">configured</option></Select></label>
        </div>
        <div className="space-y-4 rounded-box border border-base-300 p-4">
          <div className="flex flex-wrap items-start justify-between gap-3"><div><h2 className="font-semibold">Template-linked scheduled charges</h2><p className="mt-1 text-sm text-base-content/70">Each source template, line, and version is retained as evidence for later invoice generation.</p></div><Button type="button" id="billing-schedule-add-charge" variant="secondary" onClick={addCharge} disabled={loading}>Add scheduled charge</Button></div>
          {form.charges.map((charge, index) => {
            const suffix = index === 0 ? '' : `-${index}`
            return <div key={charge.id} className="rounded-box border border-base-300/70 p-4">
              <div className="flex items-center justify-between gap-3"><h3 className="font-medium">Scheduled charge {index + 1}</h3><Button type="button" variant="ghost" onClick={() => removeCharge(index)} disabled={loading}>Remove</Button></div>
              <div className="mt-4 grid gap-4 md:grid-cols-2">
                <Field id={`billing-schedule-template${suffix}`} label="Template identity" value={charge.templateId} onChange={(event) => updateCharge(index, (current) => ({ ...current, templateId: event.target.value }))} error={fieldError(`billing-schedule-template${suffix}`)} required disabled={loading} />
                <Field id={`billing-schedule-template-version${suffix}`} label="Template version" type="number" min="1" step="1" value={charge.templateVersion} onChange={(event) => updateCharge(index, (current) => ({ ...current, templateVersion: event.target.value }))} error={fieldError(`billing-schedule-template-version${suffix}`)} required disabled={loading} />
                <Field id={`billing-schedule-template-line${suffix}`} label="Template line identity" value={charge.templateLineItemId} onChange={(event) => updateCharge(index, (current) => ({ ...current, templateLineItemId: event.target.value }))} error={fieldError(`billing-schedule-template-line${suffix}`)} required disabled={loading} />
                <Field id={`billing-schedule-quantity${suffix}`} label="Quantity" inputMode="decimal" value={charge.quantity} onChange={(event) => updateCharge(index, (current) => ({ ...current, quantity: event.target.value }))} error={fieldError(`billing-schedule-quantity${suffix}`)} required disabled={loading} />
                <Field id={`billing-schedule-amount${suffix}`} label="Amount" inputMode="decimal" value={charge.amount} onChange={(event) => updateCharge(index, (current) => ({ ...current, amount: event.target.value }))} error={fieldError(`billing-schedule-amount${suffix}`)} required disabled={loading} />
                <Field id={`billing-schedule-currency${suffix}`} label="Currency" value={charge.currency} onChange={(event) => updateCharge(index, (current) => ({ ...current, currency: event.target.value }))} error={fieldError(`billing-schedule-currency${suffix}`)} required disabled={loading} />
              </div>
            </div>
          })}
        </div>
        <div className="flex flex-wrap items-center gap-3"><Button type="submit" loading={saving} disabled={loading || saving || Boolean(scheduleId && !record)}>{record ? 'Save billing-schedule revision' : 'Create billing schedule'}</Button><Button type="button" variant="secondary" onClick={() => navigate('/invoicing/inv-ws-01')} disabled={saving}>Back to worklist</Button><span className="text-sm text-base-content/70">Retries keep the same idempotency identity until the form changes or an established result is returned.</span></div>
      </form>
    </Panel>
    {conflict ? <VersionConflictDialog conflict={conflict} onClose={() => setConflict(null)} onRefresh={() => { setConflict(null); void loadSchedule() }} /> : null}
  </section>
}

export function BillingOperationsWorklist() {
  return <div className="space-y-10"><BillingScheduleWorklist /><InvoiceTemplateWorklist /></div>
}
