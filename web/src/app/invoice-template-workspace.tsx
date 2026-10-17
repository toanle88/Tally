import { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { Link as RouterLink, useNavigate, useSearchParams } from 'react-router-dom'

import { invConfigureInvoiceTemplates, invGetInvoiceTemplate, invListInvoiceTemplates } from '@/generated/api/sdk.gen'
import type { InvConfigureInvoiceTemplatesCommandRequest } from '@/generated/api/types.gen'
import { Button, DataTable, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { ValidationSummary } from '@/components/validation-summary'
import { VersionConflictDialog } from '@/components/version-conflict-dialog'
import type { EstablishedResultLink, OperationalOwner } from '@/components/operational-context'
import type { ScopeReference } from '@/components/record-context'
import type { ValidationIssue, VersionConflictSnapshot } from '@/components/workflow-context'
import { useScopeContext } from '@/lib/scope/scope-context'

type ChargeKind = 'fixed' | 'per_unit' | 'percentage'
type BillingFrequency = 'one_time' | 'daily' | 'weekly' | 'monthly' | 'quarterly' | 'annual'
type PaymentMethodKind = 'bank_transfer' | 'card' | 'direct_debit' | 'cash' | 'other'
type ApplicabilityMode = 'all' | 'dimensions'

type TemplateLineItemProjection = {
  id: string
  code: string
  description: string
  chargeRule: { kind: ChargeKind; amount?: string; currency?: string; rate?: string }
  taxCategory: string
}

type InvoiceTemplateProjection = {
  id: string
  accountingScopeId: string
  code: string
  name: string
  billingFrequency: BillingFrequency
  lineItems: TemplateLineItemProjection[]
  paymentMethod: { method: PaymentMethodKind; termsDays: number; paymentReference?: string }
  applicability: { mode: ApplicabilityMode; dimensions?: Record<string, string> }
  effectiveFrom: string
  effectiveTo?: string
  version: number
  revisionNumber: number
  revisions?: Array<{ revisionNumber: number; capturedAt: string; fingerprint: string; version: number }>
  validationOutcome?: string
}

type FormLineItem = {
  id: string
  code: string
  description: string
  chargeKind: ChargeKind
  amount: string
  currency: string
  rate: string
  taxCategory: string
}

type TemplateFormState = {
  code: string
  name: string
  billingFrequency: BillingFrequency
  lineItems: FormLineItem[]
  paymentMethod: PaymentMethodKind
  termsDays: string
  paymentReference: string
  applicabilityMode: ApplicabilityMode
  dimensions: string
  effectiveFrom: string
  effectiveTo: string
}

const billingFrequencies: readonly BillingFrequency[] = ['one_time', 'daily', 'weekly', 'monthly', 'quarterly', 'annual']
const paymentMethods: readonly PaymentMethodKind[] = ['bank_transfer', 'card', 'direct_debit', 'cash', 'other']
const chargeKinds: readonly ChargeKind[] = ['fixed', 'per_unit', 'percentage']
const exactDecimalPattern = /^(0|[1-9][0-9]*)(\.[0-9]+)?$/
const decimalInputPattern = /^\d+(?:\.\d+)?$/

function newClientIdentity() {
  return globalThis.crypto?.randomUUID?.() ?? '00000000-0000-4000-8000-000000000000'
}

function emptyLineItem(): FormLineItem {
  return { id: newClientIdentity(), code: '', description: '', chargeKind: 'fixed', amount: '', currency: 'USD', rate: '', taxCategory: '' }
}

function emptyForm(): TemplateFormState {
  return {
    code: '', name: '', billingFrequency: 'monthly', lineItems: [emptyLineItem()], paymentMethod: 'bank_transfer',
    termsDays: '0', paymentReference: '', applicabilityMode: 'all', dimensions: '', effectiveFrom: '', effectiveTo: '',
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

function dateInputValue(value: unknown) {
  const text = stringValue(value)
  return text ? text.slice(0, 10) : ''
}

function parseLineItem(value: unknown, index: number): TemplateLineItemProjection | null {
  const record = asRecord(value)
  const chargeRule = asRecord(record?.chargeRule)
  const id = stringValue(record?.id)
  const code = stringValue(record?.code)
  const description = stringValue(record?.description)
  const kind = stringValue(chargeRule?.kind)
  const taxCategory = stringValue(record?.taxCategory)
  if (!id || !code || !description || !taxCategory || !kind || !chargeKinds.includes(kind as ChargeKind)) return null
  return {
    id: id || `line-${index}`,
    code,
    description,
    taxCategory,
    chargeRule: {
      kind: kind as ChargeKind,
      amount: stringValue(chargeRule?.amount),
      currency: stringValue(chargeRule?.currency),
      rate: stringValue(chargeRule?.rate),
    },
  }
}

function parseProjection(value: unknown): InvoiceTemplateProjection | null {
  const record = asRecord(value)
  const paymentMethod = asRecord(record?.paymentMethod)
  const applicability = asRecord(record?.applicability)
  const lines = Array.isArray(record?.lineItems) ? record.lineItems.map(parseLineItem).filter((line): line is TemplateLineItemProjection => line !== null) : []
  const id = stringValue(record?.id)
  const accountingScopeId = stringValue(record?.accountingScopeId)
  const code = stringValue(record?.code)
  const name = stringValue(record?.name)
  const billingFrequency = stringValue(record?.billingFrequency)
  const method = stringValue(paymentMethod?.method)
  const termsDays = numberValue(paymentMethod?.termsDays)
  const mode = stringValue(applicability?.mode)
  const effectiveFrom = stringValue(record?.effectiveFrom)
  const version = numberValue(record?.version)
  const revisionNumber = numberValue(record?.revisionNumber)
  if (!id || !accountingScopeId || !code || !name || !billingFrequency || !billingFrequencies.includes(billingFrequency as BillingFrequency) || !method || !paymentMethods.includes(method as PaymentMethodKind) || termsDays === undefined || !mode || !['all', 'dimensions'].includes(mode) || !effectiveFrom || version === undefined || revisionNumber === undefined) return null
  const revisions = Array.isArray(record?.revisions) ? record.revisions.map((revision) => {
    const item = asRecord(revision)
    const revisionNumber = numberValue(item?.revisionNumber)
    const capturedAt = stringValue(item?.capturedAt)
    const fingerprint = stringValue(item?.fingerprint)
    const version = numberValue(item?.version)
    return revisionNumber !== undefined && capturedAt && fingerprint && version !== undefined ? { revisionNumber, capturedAt, fingerprint, version } : null
  }).filter((revision): revision is NonNullable<typeof revision> => revision !== null) : undefined
  const dimensions = asRecord(applicability?.dimensions)
  return {
    id, accountingScopeId, code, name, billingFrequency: billingFrequency as BillingFrequency, lineItems: lines,
    paymentMethod: { method: method as PaymentMethodKind, termsDays, paymentReference: stringValue(paymentMethod?.paymentReference) },
    applicability: { mode: mode as ApplicabilityMode, dimensions: dimensions ? Object.fromEntries(Object.entries(dimensions).filter(([, item]) => typeof item === 'string')) as Record<string, string> : undefined },
    effectiveFrom, effectiveTo: stringValue(record?.effectiveTo), version, revisionNumber, revisions,
    validationOutcome: stringValue(record?.validationOutcome) ?? 'valid',
  }
}

function projectionToForm(template: InvoiceTemplateProjection): TemplateFormState {
  return {
    code: template.code,
    name: template.name,
    billingFrequency: template.billingFrequency,
    lineItems: template.lineItems.length > 0 ? template.lineItems.map((line) => ({
      id: line.id, code: line.code, description: line.description, chargeKind: line.chargeRule.kind,
      amount: line.chargeRule.amount ?? '', currency: line.chargeRule.currency ?? '', rate: line.chargeRule.rate ?? '', taxCategory: line.taxCategory,
    })) : [emptyLineItem()],
    paymentMethod: template.paymentMethod.method,
    termsDays: String(template.paymentMethod.termsDays),
    paymentReference: template.paymentMethod.paymentReference ?? '',
    applicabilityMode: template.applicability.mode,
    dimensions: Object.entries(template.applicability.dimensions ?? {}).map(([key, value]) => `${key}=${value}`).join('\n'),
    effectiveFrom: dateInputValue(template.effectiveFrom),
    effectiveTo: dateInputValue(template.effectiveTo),
  }
}

function canonicalDecimal(value: string) {
  const trimmed = value.trim()
  if (!decimalInputPattern.test(trimmed)) return trimmed
  const [integerPart, fractionPart] = trimmed.split('.')
  const integer = integerPart.replace(/^0+(?=\d)/, '')
  const fraction = fractionPart?.replace(/0+$/, '')
  return fraction ? `${integer}.${fraction}` : integer
}

function isPercentageRate(value: string) {
  const canonical = canonicalDecimal(value)
  if (!exactDecimalPattern.test(canonical)) return false
  const [integerPart, fractionPart = ''] = canonical.split('.')
  const integer = integerPart.replace(/^0+(?=\d)/, '')
  if (integer.length < 3) return true
  if (integer.length > 3 || integer !== '100') return false
  return /^0*$/.test(fractionPart)
}

function isNonNegativeDecimal(value: string) {
  return exactDecimalPattern.test(canonicalDecimal(value))
}

function parseDimensions(value: string) {
  return Object.fromEntries(value.split(/\r?\n/).map((line) => line.trim()).filter(Boolean).map((line) => {
    const separator = line.indexOf('=')
    return separator < 1 ? ['', ''] : [line.slice(0, separator).trim(), line.slice(separator + 1).trim()]
  }).filter(([key, item]) => key && item))
}

function errorDetails(error: unknown) {
  const value = asRecord(error)
  return {
    status: numberValue(value?.status),
    code: stringValue(value?.code),
    detail: stringValue(value?.detail) ?? 'The invoice-template operation could not be completed. Review the current authoritative state and retry safely.',
    currentVersion: numberValue(value?.currentVersion),
    fieldErrors: Array.isArray(value?.fieldErrors) ? value.fieldErrors.map((field) => asRecord(field)).filter((field): field is Record<string, unknown> => field !== null).map((field) => ({ field: stringValue(field.field) ?? '', code: stringValue(field.code) ?? 'invalid', message: stringValue(field.message) ?? 'Review this value.' })) : [],
  }
}

function validationIssue(id: string, targetId: string, targetLabel: string, message: string, category: ValidationIssue['category'] = 'field'): ValidationIssue {
  return { id, targetId, targetLabel, message, category, code: id, nextAction: 'Correct the value and review the other highlighted fields.' }
}

function validateForm(form: TemplateFormState): ValidationIssue[] {
  const issues: ValidationIssue[] = []
  if (!form.code.trim()) issues.push(validationIssue('code-required', 'invoice-template-code', 'Template code', 'Enter a stable template code.'))
  if (!form.name.trim()) issues.push(validationIssue('name-required', 'invoice-template-name', 'Template name', 'Enter a descriptive template name.'))
  if (!form.effectiveFrom) issues.push(validationIssue('effective-from-required', 'invoice-template-effective-from', 'Effective from', 'Choose the first date on which this version applies.'))
  if (form.effectiveTo && form.effectiveTo < form.effectiveFrom) issues.push(validationIssue('effective-range', 'invoice-template-effective-to', 'Effective to', 'The inclusive end date cannot precede the start date.', 'business-rule'))
  if (form.lineItems.length === 0) issues.push(validationIssue('line-items-required', 'invoice-template-line-items', 'Line items', 'Add at least one approved line item.', 'line'))
  form.lineItems.forEach((line, index) => {
    const prefix = `invoice-template-line-${index}`
    if (!line.code.trim()) issues.push(validationIssue(`${prefix}-code`, `${prefix}-code`, `Line ${index + 1} code`, 'Enter a stable line-item code.', 'line'))
    if (!line.description.trim()) issues.push(validationIssue(`${prefix}-description`, `${prefix}-description`, `Line ${index + 1} description`, 'Enter a line-item description.', 'line'))
    if (!line.taxCategory.trim()) issues.push(validationIssue(`${prefix}-tax-category`, `${prefix}-tax-category`, `Line ${index + 1} tax category`, 'Enter the owning Tax category reference.', 'dependency'))
    if (line.chargeKind === 'percentage') {
      if (!isPercentageRate(line.rate)) issues.push(validationIssue(`${prefix}-rate`, `${prefix}-rate`, `Line ${index + 1} rate`, 'Enter a non-negative exact decimal rate from 0 to 100.', 'business-rule'))
      if (line.amount || line.currency) issues.push(validationIssue(`${prefix}-percentage-shape`, `${prefix}-rate`, `Line ${index + 1} charge rule`, 'Percentage rules use rate only; clear amount and currency.', 'business-rule'))
    } else {
      if (!isNonNegativeDecimal(line.amount)) issues.push(validationIssue(`${prefix}-amount`, `${prefix}-amount`, `Line ${index + 1} amount`, 'Enter a non-negative exact decimal amount.', 'business-rule'))
      if (!/^[A-Z]{3}$/.test(line.currency.trim().toUpperCase())) issues.push(validationIssue(`${prefix}-currency`, `${prefix}-currency`, `Line ${index + 1} currency`, 'Enter a three-letter currency code.', 'business-rule'))
      if (line.rate) issues.push(validationIssue(`${prefix}-amount-shape`, `${prefix}-amount`, `Line ${index + 1} charge rule`, 'Fixed and per-unit rules use amount and currency; clear rate.', 'business-rule'))
    }
  })
  const terms = Number(form.termsDays)
  if (!/^\d+$/.test(form.termsDays.trim()) || !Number.isSafeInteger(terms) || terms < 0) issues.push(validationIssue('terms-days', 'invoice-template-terms-days', 'Payment terms', 'Enter a non-negative whole number of days.'))
  if (form.applicabilityMode === 'dimensions') {
    const entries = form.dimensions.split(/\r?\n/).map((line) => line.trim()).filter(Boolean)
    if (entries.length === 0 || entries.some((line) => !line.includes('=') || !line.split('=')[0].trim() || !line.split('=').slice(1).join('=').trim())) issues.push(validationIssue('dimensions-required', 'invoice-template-dimensions', 'Applicability dimensions', 'Enter one non-empty key=value pair per line.', 'business-rule'))
  }
  return issues
}

function targetForServerField(field: string) {
  if (field.includes('lineItems')) return 'invoice-template-line-items'
  if (field.includes('effective')) return 'invoice-template-effective-from'
  if (field.includes('payment')) return 'invoice-template-payment-method'
  if (field.includes('applicability')) return 'invoice-template-applicability'
  if (field.includes('name')) return 'invoice-template-name'
  return 'invoice-template-code'
}

function conflictSnapshot(scope: ScopeReference, template: InvoiceTemplateProjection | null, form: TemplateFormState, error: unknown): VersionConflictSnapshot {
  const details = errorDetails(error)
  const expectedVersion = template?.version ?? 0
  const currentVersion = details.currentVersion ?? Math.max(expectedVersion + 1, 1)
  const record: EstablishedResultLink = { label: template?.code || form.code || 'Invoice template', href: template ? `/invoicing/inv-scr-01?invoiceTemplateId=${template.id}` : '/invoicing/inv-ws-01' }
  const owner: OperationalOwner = { id: 'invoicing', label: 'Billing Operations', reference: 'finance.inv.configure.invoice.templates' }
  return {
    kind: 'version', attemptedAction: template ? 'Update invoice-template version' : 'Create invoice-template version', record, scope,
    expected: { version: expectedVersion, state: 'reviewed' }, current: { version: currentVersion, state: 'authoritative current' },
    changedValues: [
      { id: 'code', label: 'Code', reviewedValue: form.code, currentValue: template?.code ?? 'Unavailable until refresh' },
      { id: 'name', label: 'Name', reviewedValue: form.name, currentValue: template?.name ?? 'Unavailable until refresh' },
      { id: 'effective-from', label: 'Effective from', reviewedValue: form.effectiveFrom, currentValue: template ? dateInputValue(template.effectiveFrom) : 'Unavailable until refresh' },
    ], currentOwner: owner, processEpoch: `invoice-template-version-${currentVersion}`, safeRetry: { permitted: false, reason: 'Refresh the authoritative template, compare the reviewed changes, then submit a newly reviewed version.' },
  }
}

function scopeHeader(scopeId: string) {
  return { 'X-Accounting-Scope-Id': scopeId }
}

function templateStatusState(template: InvoiceTemplateProjection): SemanticState {
  return template.validationOutcome === 'valid' ? 'success' : 'warning'
}

export function InvoiceTemplateWorklist() {
  const { currentScope } = useScopeContext()
  const navigate = useNavigate()
  const [templates, setTemplates] = useState<InvoiceTemplateProjection[]>([])
  const [status, setStatus] = useState<'loading' | 'ready' | 'unavailable'>('loading')
  const [notice, setNotice] = useState('Loading the authoritative invoice-template worklist…')

  const load = useCallback(async () => {
    if (!currentScope) {
      setTemplates([])
      setStatus('unavailable')
      setNotice('Select an accounting scope before loading invoice templates.')
      return
    }
    setStatus('loading')
    setNotice('Loading the authoritative invoice-template worklist…')
    try {
      const response = await invListInvoiceTemplates({ headers: scopeHeader(currentScope.id), query: { 'page[size]': 50 } })
      if (response.error) {
        setTemplates([])
        setStatus('unavailable')
        setNotice(errorDetails(response.error).detail)
        return
      }
      const values = Array.isArray(response.data?.data?.templates) ? response.data.data.templates.map(parseProjection).filter((template): template is InvoiceTemplateProjection => template !== null) : []
      setTemplates(values)
      setStatus('ready')
      setNotice(values.length > 0 ? `Loaded ${values.length} authoritative invoice-template version${values.length === 1 ? '' : 's'} for ${currentScope.legalEntity.name}.` : 'No authoritative invoice templates are configured for this accounting scope.')
    } catch (error) {
      setTemplates([])
      setStatus('unavailable')
      setNotice(error instanceof Error ? error.message : 'The authoritative invoice-template list is unavailable.')
    }
  }, [currentScope])

  useEffect(() => { void load() }, [load])

  const columns = useMemo(() => [
    { key: 'code', header: 'Template', rowHeader: true, render: (template: InvoiceTemplateProjection) => <RouterLink className="link link-primary font-semibold" to={`/invoicing/inv-scr-01?invoiceTemplateId=${template.id}`}>{template.code}</RouterLink> },
    { key: 'name', header: 'Name', render: (template: InvoiceTemplateProjection) => template.name },
    { key: 'frequency', header: 'Frequency', render: (template: InvoiceTemplateProjection) => template.billingFrequency },
    { key: 'effective', header: 'Effective interval', render: (template: InvoiceTemplateProjection) => `${dateInputValue(template.effectiveFrom)} — ${dateInputValue(template.effectiveTo) || 'open'}` },
    { key: 'version', header: 'Version', render: (template: InvoiceTemplateProjection) => `v${template.version} / r${template.revisionNumber}` },
    { key: 'validation', header: 'Validation', render: (template: InvoiceTemplateProjection) => <StatusBadge state={templateStatusState(template)} label={template.validationOutcome ?? 'valid'} /> },
    { key: 'next-action', header: 'Next action', render: () => 'Review and publish when approved' },
  ] as const, [])

  return <section aria-labelledby="invoice-template-worklist-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">INV-WS-01</p><h1 id="invoice-template-worklist-title" className="mt-1 text-2xl font-semibold">Invoice-template worklist</h1><p className="mt-2 max-w-4xl text-base-content/75">Review effective-dated invoice-template versions in the selected accounting scope. The worklist and editor use the Invoicing reference API as the authoritative source.</p></div>
    <Panel title="Selected accounting scope" description="Scope is sent on every read and mutation; a client-side scope selection is never treated as an authorization decision.">
      <dl className="grid gap-2 text-sm sm:grid-cols-2"><div><dt className="text-base-content/65">Legal entity</dt><dd className="font-medium">{currentScope?.legalEntity.name ?? 'Not selected'}</dd></div><div><dt className="text-base-content/65">Accounting book</dt><dd className="font-medium">{currentScope?.accountingBook.name ?? 'Not selected'}</dd></div><div><dt className="text-base-content/65">Owner</dt><dd className="font-medium">Billing Operations / Invoicing</dd></div><div><dt className="text-base-content/65">Permission</dt><dd className="font-medium">finance.inv.configure.invoice.templates</dd></div></dl>
    </Panel>
    <Panel title="Configured versions" description="Historical revisions remain available from the detail screen and are not overwritten by later maintenance.">
      <div className="flex flex-wrap items-center justify-between gap-3"><p role="status" aria-live="polite" className="text-sm text-base-content/75">{notice}</p><div className="flex flex-wrap gap-2"><Button variant="secondary" onClick={() => void load()} loading={status === 'loading'}>Refresh authoritative list</Button><Button onClick={() => navigate('/invoicing/inv-scr-01?new=true')}>Configure new template</Button></div></div>
      {status === 'unavailable' ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">The authoritative invoice-template list is unavailable or was denied. No local fixture records are shown.</p> : null}
      {status === 'ready' ? <div className="mt-4"><DataTable caption="Authoritative invoice-template versions" columns={columns} rows={templates} getRowKey={(template) => template.id} emptyMessage="No invoice-template versions are configured for this scope." /></div> : null}
    </Panel>
  </section>
}

export function InvoiceTemplateEditor() {
  const { currentScope, setHasUnsavedChanges } = useScopeContext()
  const [searchParams] = useSearchParams()
  const navigate = useNavigate()
  const templateId = searchParams.get('invoiceTemplateId')
  const [record, setRecord] = useState<InvoiceTemplateProjection | null>(null)
  const [form, setForm] = useState<TemplateFormState>(() => emptyForm())
  const [issues, setIssues] = useState<ValidationIssue[]>([])
  const [loading, setLoading] = useState(Boolean(templateId))
  const [saving, setSaving] = useState(false)
  const [loadError, setLoadError] = useState('')
  const [notice, setNotice] = useState(templateId ? 'Loading the authoritative invoice-template version…' : 'Enter the complete replacement configuration, then submit it for server validation.')
  const [conflict, setConflict] = useState<VersionConflictSnapshot | null>(null)
  const idempotencyKey = useRef<string | null>(null)

  const loadTemplate = useCallback(async () => {
    if (!currentScope || !templateId) {
      setLoading(false)
      if (!templateId) {
        setRecord(null)
        setForm(emptyForm())
        setNotice('Create mode: the first established version will receive its server-owned template identity.')
      }
      return
    }
    setLoading(true)
    setLoadError('')
    try {
      const response = await invGetInvoiceTemplate({ path: { invoiceTemplateId: templateId }, headers: scopeHeader(currentScope.id) })
      if (response.error) {
        setLoadError(errorDetails(response.error).detail)
        setNotice('The editor could not load the authoritative invoice-template version.')
        return
      }
      const loaded = parseProjection(response.data?.data?.template)
      if (!loaded) {
        setLoadError('The authoritative response did not contain a usable invoice-template projection.')
        return
      }
      setRecord(loaded)
      setForm(projectionToForm(loaded))
      setIssues([])
      setNotice(`Loaded ${loaded.code} at authoritative version ${loaded.version}; retained revisions: ${loaded.revisions?.length ?? loaded.revisionNumber}.`)
      setHasUnsavedChanges(false)
    } catch (error) {
      setLoadError(error instanceof Error ? error.message : 'The authoritative invoice-template detail is unavailable.')
      setNotice('The editor could not load the authoritative invoice-template version.')
    } finally {
      setLoading(false)
    }
  }, [currentScope, setHasUnsavedChanges, templateId])

  useEffect(() => { void loadTemplate() }, [loadTemplate])

  const updateForm = useCallback((update: (current: TemplateFormState) => TemplateFormState) => {
    setForm(update)
    idempotencyKey.current = null
    setHasUnsavedChanges(true)
  }, [setHasUnsavedChanges])

  const fieldError = (targetId: string) => issues.find((issue) => issue.targetId === targetId)?.message

  const focusTarget = (targetId: string) => {
    const target = document.getElementById(targetId)
    if (target instanceof HTMLElement) {
      target.focus()
      if (typeof target.scrollIntoView === 'function') target.scrollIntoView({ block: 'center' })
    }
  }

  const submit = async () => {
    const localIssues = validateForm(form)
    setIssues(localIssues)
    if (localIssues.length > 0) {
      setNotice('Resolve the highlighted fields before sending the command.')
      focusTarget(localIssues[0].targetId)
      return
    }
    if (!currentScope || (templateId && !record)) {
      setNotice('Load an authoritative template version before submitting.')
      return
    }
    const dimensions = form.applicabilityMode === 'dimensions' ? parseDimensions(form.dimensions) : undefined
    const lineItems = form.lineItems.map((line) => ({
      id: line.id, code: line.code.trim(), description: line.description.trim(), taxCategory: line.taxCategory.trim(),
      chargeRule: line.chargeKind === 'percentage'
        ? { kind: line.chargeKind, rate: canonicalDecimal(line.rate.trim()) }
        : { kind: line.chargeKind, amount: canonicalDecimal(line.amount.trim()), currency: line.currency.trim().toUpperCase() },
    }))
    const data: InvConfigureInvoiceTemplatesCommandRequest['data'] = {
      action: record ? 'update' : 'create', templateId: record?.id, code: form.code.trim(), name: form.name.trim(), billingFrequency: form.billingFrequency,
      lineItems, paymentMethod: { method: form.paymentMethod, termsDays: Number(form.termsDays), paymentReference: form.paymentReference.trim() || undefined },
      applicability: form.applicabilityMode === 'dimensions' ? { mode: 'dimensions', dimensions } : { mode: 'all' },
      effectiveFrom: form.effectiveFrom, effectiveTo: form.effectiveTo || undefined,
    }
    const key = idempotencyKey.current ?? newClientIdentity()
    idempotencyKey.current = key
    setSaving(true)
    setNotice('Submitting the idempotent invoice-template command…')
    try {
      const response = await invConfigureInvoiceTemplates({
        body: { commandId: newClientIdentity(), expectedVersion: record?.version, accountingScopeId: currentScope.id, data },
        headers: { ...scopeHeader(currentScope.id), 'Idempotency-Key': key, 'If-Match': record ? `"${record.version}"` : undefined },
      })
      if (response.error) {
        const details = errorDetails(response.error)
        if (details.status === 409 && (details.code === 'VERSION_CONFLICT' || details.code === 'SCOPE_CONFLICT')) {
          setConflict(conflictSnapshot({ id: currentScope.id, label: currentScope.legalEntity.name }, record, form, response.error))
        }
        if (details.fieldErrors.length > 0) setIssues(details.fieldErrors.map((field, index) => validationIssue(`server-${index}-${field.code}`, targetForServerField(field.field), field.field || 'Template value', field.message, 'field')))
        setNotice(details.detail)
        return
      }
      const established = parseProjection(response.data?.data?.template)
      if (!established) {
        setNotice('The established result did not contain a usable invoice-template projection; refresh the authoritative detail before retrying.')
        return
      }
      setRecord(established)
      setForm(projectionToForm(established))
      setIssues([])
      idempotencyKey.current = null
      setHasUnsavedChanges(false)
      setNotice(response.data?.data?.replayed ? `The established result for ${established.code} was replayed safely at version ${established.version}.` : `Invoice template ${established.code} established at version ${established.version}.`)
      if (!templateId) navigate(`/invoicing/inv-scr-01?invoiceTemplateId=${established.id}`, { replace: true })
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The invoice-template command could not reach the live API. Retry with the same idempotency identity.')
    } finally {
      setSaving(false)
    }
  }

  const scopeLabel = currentScope?.legalEntity.name ?? 'No accounting scope selected'
  const updateLine = (index: number, update: (line: FormLineItem) => FormLineItem) => updateForm((current) => ({ ...current, lineItems: current.lineItems.map((line, lineIndex) => lineIndex === index ? update(line) : line) }))
  const removeLine = (index: number) => updateForm((current) => ({ ...current, lineItems: current.lineItems.filter((_, lineIndex) => lineIndex !== index) }))

  return <section aria-labelledby="invoice-template-editor-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">INV-SCR-01</p><h1 id="invoice-template-editor-title" className="mt-1 text-2xl font-semibold">{record ? `Maintain invoice template: ${record.code}` : 'Configure invoice template'}</h1><p className="mt-2 max-w-4xl text-base-content/75">Maintain a complete, effective-dated version for one accounting scope. Established versions are immutable historical facts; an update creates the next version after server authorization and validation.</p></div>
    <p role="status" aria-live="polite" className="rounded-box border border-base-300 bg-base-200/30 p-3 text-sm">{notice}</p>
    {loadError ? <p role="alert" className="rounded-box border border-error/40 bg-error/10 p-3 text-sm">{loadError} No local fixture was substituted.</p> : null}
    <Panel title="Authoritative record context" description="The displayed identity, version, scope, owner, validation state, and next action are either loaded from or returned by the Invoicing API.">
      <dl className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4"><div><dt className="text-base-content/65">Accounting scope</dt><dd className="font-medium">{scopeLabel}</dd></div><div><dt className="text-base-content/65">Template identity</dt><dd className="font-medium">{record?.id ?? 'Server assigned on create'}</dd></div><div><dt className="text-base-content/65">Version / revision</dt><dd className="font-medium">{record ? `v${record.version} / r${record.revisionNumber}` : 'New version'}</dd></div><div><dt className="text-base-content/65">Owner / permission</dt><dd className="font-medium">Billing Operations · finance.inv.configure.invoice.templates</dd></div><div><dt className="text-base-content/65">Validation</dt><dd><StatusBadge state={record ? templateStatusState(record) : 'pending'} label={record?.validationOutcome ?? 'pending server validation'} /></dd></div><div><dt className="text-base-content/65">Next action</dt><dd className="font-medium">{record ? 'Review and publish when approved' : 'Complete and submit configuration'}</dd></div></dl>
      {record?.revisions?.length ? <p className="mt-3 text-sm text-base-content/70">Retained immutable revisions: {record.revisions.map((revision) => `v${revision.version} / r${revision.revisionNumber}`).join(', ')}.</p> : null}
    </Panel>
    <ValidationSummary issues={issues} onFocusTarget={focusTarget} />
    <Panel title="Template settings" description="Canonical values are submitted as strings for exact decimals and are revalidated by the owning Invoicing module.">
      <form noValidate onSubmit={(event) => { event.preventDefault(); void submit() }} className="space-y-6">
        <div className="grid gap-4 md:grid-cols-2"><Field id="invoice-template-code" label="Template code" value={form.code} onChange={(event) => updateForm((current) => ({ ...current, code: event.target.value }))} error={fieldError('invoice-template-code')} required disabled={loading} /><Field id="invoice-template-name" label="Template name" value={form.name} onChange={(event) => updateForm((current) => ({ ...current, name: event.target.value }))} error={fieldError('invoice-template-name')} required disabled={loading} /><label className="form-control w-full gap-2"><span className="label font-medium">Billing frequency <span aria-hidden="true" className="text-error">*</span></span><Select id="invoice-template-billing-frequency" aria-label="Billing frequency" value={form.billingFrequency} onChange={(event) => updateForm((current) => ({ ...current, billingFrequency: event.target.value as BillingFrequency }))} disabled={loading}>{billingFrequencies.map((frequency) => <option key={frequency} value={frequency}>{frequency}</option>)}</Select></label><Field id="invoice-template-effective-from" label="Effective from" type="date" value={form.effectiveFrom} onChange={(event) => updateForm((current) => ({ ...current, effectiveFrom: event.target.value }))} error={fieldError('invoice-template-effective-from')} required disabled={loading} /><Field id="invoice-template-effective-to" label="Effective to" type="date" value={form.effectiveTo} onChange={(event) => updateForm((current) => ({ ...current, effectiveTo: event.target.value }))} error={fieldError('invoice-template-effective-to')} description="Leave blank for an open-ended inclusive interval." disabled={loading} /><label className="form-control w-full gap-2"><span className="label font-medium">Applicability</span><Select id="invoice-template-applicability" aria-label="Applicability" value={form.applicabilityMode} onChange={(event) => updateForm((current) => ({ ...current, applicabilityMode: event.target.value as ApplicabilityMode }))} disabled={loading}><option value="all">All records in scope</option><option value="dimensions">Matching dimensions</option></Select></label></div>
        {form.applicabilityMode === 'dimensions' ? <div><label className="form-control w-full gap-2"><span className="label font-medium">Applicability dimensions</span><textarea id="invoice-template-dimensions" value={form.dimensions} onChange={(event) => updateForm((current) => ({ ...current, dimensions: event.target.value }))} aria-invalid={fieldError('invoice-template-dimensions') ? true : undefined} className="textarea textarea-bordered min-h-24 w-full border-2 border-base-300 bg-base-100 p-3 focus:border-primary focus:outline-2 focus:outline-offset-1 focus:outline-primary" placeholder="customerTier=premium\nregion=APAC" disabled={loading} /></label>{fieldError('invoice-template-dimensions') ? <p className="mt-1 text-sm text-error">{fieldError('invoice-template-dimensions')}</p> : null}</div> : null}
        <div id="invoice-template-line-items" className="space-y-4"><div className="flex flex-wrap items-center justify-between gap-2"><div><h3 className="font-semibold">Approved line items</h3><p className="text-sm text-base-content/70">Each line item must contain a Tax-owned category reference and one exact-decimal charge rule.</p></div><Button variant="secondary" onClick={() => updateForm((current) => ({ ...current, lineItems: [...current.lineItems, emptyLineItem()] }))} disabled={loading}>Add line item</Button></div>{form.lineItems.map((line, index) => { const prefix = `invoice-template-line-${index}`; return <div key={line.id} className="rounded-box border border-base-300 p-4"><div className="mb-3 flex flex-wrap items-center justify-between gap-2"><h4 className="font-medium">Line item {index + 1}</h4><Button variant="ghost" size="sm" onClick={() => removeLine(index)} disabled={loading || form.lineItems.length <= 1}>Remove line item</Button></div><div className="grid gap-4 md:grid-cols-2"><Field id={`${prefix}-code`} label="Line code" value={line.code} onChange={(event) => updateLine(index, (current) => ({ ...current, code: event.target.value }))} error={fieldError(`${prefix}-code`)} required disabled={loading} /><Field id={`${prefix}-description`} label="Description" value={line.description} onChange={(event) => updateLine(index, (current) => ({ ...current, description: event.target.value }))} error={fieldError(`${prefix}-description`)} required disabled={loading} /><label className="form-control w-full gap-2"><span className="label font-medium">Charge rule</span><Select id={`${prefix}-charge-kind`} aria-label={`Line ${index + 1} charge rule`} value={line.chargeKind} onChange={(event) => updateLine(index, (current) => ({ ...current, chargeKind: event.target.value as ChargeKind, amount: event.target.value === 'percentage' ? '' : current.amount, currency: event.target.value === 'percentage' ? '' : current.currency, rate: event.target.value === 'percentage' ? current.rate : '' }))} disabled={loading}>{chargeKinds.map((kind) => <option key={kind} value={kind}>{kind}</option>)}</Select></label>{line.chargeKind === 'percentage' ? <Field id={`${prefix}-rate`} label="Rate" inputMode="decimal" value={line.rate} onChange={(event) => updateLine(index, (current) => ({ ...current, rate: event.target.value }))} error={fieldError(`${prefix}-rate`)} description="Exact decimal percentage from 0 to 100; do not use a percent sign." required disabled={loading} /> : <><Field id={`${prefix}-amount`} label="Amount" inputMode="decimal" value={line.amount} onChange={(event) => updateLine(index, (current) => ({ ...current, amount: event.target.value }))} error={fieldError(`${prefix}-amount`)} description="Exact decimal; binary floating point is not used." required disabled={loading} /><Field id={`${prefix}-currency`} label="Currency" value={line.currency} onChange={(event) => updateLine(index, (current) => ({ ...current, currency: event.target.value }))} error={fieldError(`${prefix}-currency`)} required disabled={loading} /></>}<Field id={`${prefix}-tax-category`} label="Tax category reference" value={line.taxCategory} onChange={(event) => updateLine(index, (current) => ({ ...current, taxCategory: event.target.value }))} error={fieldError(`${prefix}-tax-category`)} required disabled={loading} /></div></div> })}</div>
        <div className="grid gap-4 md:grid-cols-2"><label className="form-control w-full gap-2"><span className="label font-medium">Payment method</span><Select id="invoice-template-payment-method" aria-label="Payment method" value={form.paymentMethod} onChange={(event) => updateForm((current) => ({ ...current, paymentMethod: event.target.value as PaymentMethodKind }))} disabled={loading}>{paymentMethods.map((method) => <option key={method} value={method}>{method}</option>)}</Select></label><Field id="invoice-template-terms-days" label="Payment terms (days)" type="number" min="0" step="1" value={form.termsDays} onChange={(event) => updateForm((current) => ({ ...current, termsDays: event.target.value }))} error={fieldError('invoice-template-terms-days')} required disabled={loading} /><Field id="invoice-template-payment-reference" label="Payment reference" type="password" value={form.paymentReference} onChange={(event) => updateForm((current) => ({ ...current, paymentReference: event.target.value }))} description="Masked while editing; use an owning reference, never a secret payment credential." disabled={loading} /></div>
        <div className="flex flex-wrap items-center gap-3"><Button type="submit" loading={saving} disabled={loading || saving || Boolean(templateId && !record)}>{record ? 'Save invoice-template revision' : 'Create invoice template'}</Button><Button variant="secondary" onClick={() => navigate('/invoicing/inv-ws-01')} disabled={saving}>Back to worklist</Button><span className="text-sm text-base-content/70">Retries keep the same idempotency identity until the form changes or an established result is returned.</span></div>
      </form>
    </Panel>
    {conflict ? <VersionConflictDialog conflict={conflict} onClose={() => setConflict(null)} onRefresh={() => { setConflict(null); void loadTemplate() }} /> : null}
  </section>
}
