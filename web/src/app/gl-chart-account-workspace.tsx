import { useMemo, useState } from 'react'

import { glMaintainAccountsAndReportingMappings, glMaintainChartsOfAccounts } from '@/generated/api/sdk.gen'
import { Button, DataTable, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

import type { GlLifecycleStatus } from './gl-ledger-book-workspace'

export type GlChartRecord = {
  id: string
  scopeId: string
  ledgerId: string
  accountCodePolicy: string
  lifecycleStatus: GlLifecycleStatus
  effectiveDateFrom: string
  effectiveDateTo?: string
  approvalStatus: string
  validationOutcome: string
  nextAction: string
  version: number
  revisionNumber: number
}

export type GlAccountRecord = {
  id: string
  scopeId: string
  chartOfAccountsId: string
  accountCode: string
  accountName: string
  accountType: string
  normalBalance: 'debit' | 'credit'
  lifecycleStatus: GlLifecycleStatus
  restrictions: string[]
  currencyPolicy: string
  reportingMappings: Array<{
    reportingDefinitionId: string
    reportingLineCode: string
    approved: boolean
    effectiveDateFrom?: string
    effectiveDateTo?: string
  }>
  reportingMappingCount: number
  approvedReportingMappingCount: number
  effectiveDateFrom: string
  effectiveDateTo?: string
  approvalStatus: string
  validationOutcome: string
  nextAction: string
  version: number
  revisionNumber: number
}

const statusState: Record<GlLifecycleStatus, SemanticState> = { draft: 'pending', active: 'success', suspended: 'warning', retired: 'disabled' }

const initialCharts: GlChartRecord[] = [
  { id: 'chart-vietnam', scopeId: 'scope-vietnam-statutory', ledgerId: 'ledger-vietnam', accountCodePolicy: 'NNNN-NN', lifecycleStatus: 'active', effectiveDateFrom: '2026-01-01', approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 2, revisionNumber: 2 },
  { id: 'chart-singapore', scopeId: 'scope-singapore-management', ledgerId: 'ledger-singapore', accountCodePolicy: 'NNNN', lifecycleStatus: 'draft', effectiveDateFrom: '2026-02-01', effectiveDateTo: '2026-12-31', approvalStatus: 'pending', validationOutcome: 'valid', nextAction: 'submit for approval', version: 1, revisionNumber: 1 },
]

const initialAccounts: GlAccountRecord[] = [
  { id: 'account-vietnam-cash', scopeId: 'scope-vietnam-statutory', chartOfAccountsId: 'chart-vietnam', accountCode: '1111', accountName: 'Cash and bank', accountType: 'asset', normalBalance: 'debit', lifecycleStatus: 'active', restrictions: ['postable'], currencyPolicy: 'functional-or-transaction', reportingMappings: [{ reportingDefinitionId: 'report-definition-vietnam', reportingLineCode: 'cash', approved: true, effectiveDateFrom: '2026-01-01' }], reportingMappingCount: 1, approvedReportingMappingCount: 1, effectiveDateFrom: '2026-01-01', approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 2, revisionNumber: 2 },
  { id: 'account-vietnam-revenue', scopeId: 'scope-vietnam-statutory', chartOfAccountsId: 'chart-vietnam', accountCode: '5111', accountName: 'Revenue', accountType: 'revenue', normalBalance: 'credit', lifecycleStatus: 'active', restrictions: ['postable'], currencyPolicy: 'functional', reportingMappings: [{ reportingDefinitionId: 'report-definition-vietnam', reportingLineCode: 'revenue', approved: true, effectiveDateFrom: '2026-01-01' }], reportingMappingCount: 1, approvedReportingMappingCount: 1, effectiveDateFrom: '2026-01-01', approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 1, revisionNumber: 1 },
]

const chartReadAdapter = new Map(initialCharts.map((record) => [record.id, record]))
const accountReadAdapter = new Map(initialAccounts.map((record) => [record.id, record]))

function readSafeCharts(scopeId: string) {
  return Array.from(chartReadAdapter.values()).filter((record) => record.scopeId === scopeId).sort((left, right) => left.id.localeCompare(right.id))
}

function readSafeAccounts(scopeId: string, chartId?: string) {
  return Array.from(accountReadAdapter.values()).filter((record) => record.scopeId === scopeId && (!chartId || record.chartOfAccountsId === chartId)).sort((left, right) => left.accountCode.localeCompare(right.accountCode))
}

function dateValue(value?: string | null) {
  return value ? value.slice(0, 10) : undefined
}

function rangesOverlap(leftFrom: string, leftTo: string, rightFrom: string, rightTo: string) {
  const leftEnd = leftTo || '9999-12-31'
  const rightEnd = rightTo || '9999-12-31'
  return leftFrom <= rightEnd && rightFrom <= leftEnd
}

function lifecycleTransitionIsValid(current: GlLifecycleStatus | undefined, next: GlLifecycleStatus) {
  if (!current || current === next) return true
  if (current === 'draft') return next === 'active' || next === 'retired'
  if (current === 'active') return next === 'suspended' || next === 'retired'
  if (current === 'suspended') return next === 'active' || next === 'retired'
  return false
}

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The general-ledger configuration command could not be completed. Review the current version and retry safely.'
}

type SelectionState = { scopeId: string | null; id: string | null }

function resolveSelectionId<T extends { id: string }>(selection: SelectionState, scopeId: string | null, records: T[]) {
  if (selection.scopeId !== scopeId) return records[0]?.id ?? null
  if (selection.id === null) return null
  return records.some((record) => record.id === selection.id) ? selection.id : records[0]?.id ?? null
}

export function GlChartAccountWorkspace() {
  const { currentScope } = useScopeContext()
  const [adapterRevision, setAdapterRevision] = useState(0)
  const charts = useMemo(() => currentScope ? readSafeCharts(currentScope.id) : [], [currentScope?.id, adapterRevision])
  const accounts = useMemo(() => currentScope ? readSafeAccounts(currentScope.id) : [], [currentScope?.id, adapterRevision])
  const scopeId = currentScope?.id ?? null
  const [chartSelection, setChartSelection] = useState<SelectionState>({ scopeId: null, id: null })
  const [accountSelection, setAccountSelection] = useState<SelectionState>({ scopeId: null, id: null })
  const selectedChartId = resolveSelectionId(chartSelection, scopeId, charts)
  const selectedAccountId = resolveSelectionId(accountSelection, scopeId, accounts)
  const selectChart = (id: string | null) => setChartSelection({ scopeId, id })
  const selectAccount = (id: string | null) => setAccountSelection({ scopeId, id })
  const selectedChart = charts.find((record) => record.id === selectedChartId)
  const selectedAccount = accounts.find((record) => record.id === selectedAccountId)

  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-05</p><h2 className="mt-1 text-2xl font-semibold">Chart and account configuration</h2><p className="mt-2">Select an accounting scope before maintaining charts and accounts.</p></section>
  }

  const chartColumns = [
    { key: 'policy', header: 'Chart / code policy', rowHeader: true, render: (record: GlChartRecord) => <button type="button" className="link link-primary font-semibold" onClick={() => selectChart(record.id)}>{record.id} · {record.accountCodePolicy}</button> },
    { key: 'ledger', header: 'Ledger', render: (record: GlChartRecord) => record.ledgerId },
    { key: 'status', header: 'State', render: (record: GlChartRecord) => <StatusBadge state={statusState[record.lifecycleStatus]} label={record.lifecycleStatus} /> },
    { key: 'effective', header: 'Effective interval', render: (record: GlChartRecord) => `${record.effectiveDateFrom} — ${record.effectiveDateTo ?? 'open'}` },
    { key: 'version', header: 'Version', render: (record: GlChartRecord) => `v${record.version}` },
  ] as const
  const accountColumns = [
    { key: 'code', header: 'Account', rowHeader: true, render: (record: GlAccountRecord) => <button type="button" className="link link-primary font-semibold" onClick={() => selectAccount(record.id)}>{record.accountCode} · {record.accountName}</button> },
    { key: 'type', header: 'Type / normal', render: (record: GlAccountRecord) => `${record.accountType} / ${record.normalBalance}` },
    { key: 'controls', header: 'Restrictions / mappings', render: (record: GlAccountRecord) => `${record.restrictions.length} / ${record.approvedReportingMappingCount} approved` },
    { key: 'status', header: 'State', render: (record: GlAccountRecord) => <StatusBadge state={statusState[record.lifecycleStatus]} label={record.lifecycleStatus} /> },
    { key: 'version', header: 'Version', render: (record: GlAccountRecord) => `v${record.version}` },
  ] as const

  const refresh = () => setAdapterRevision((revision) => revision + 1)
  return <section aria-labelledby="gl-scr-05-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-05</p><h2 id="gl-scr-05-title" className="mt-1 text-2xl font-semibold">Chart and account configuration</h2><p className="mt-2 max-w-4xl text-base-content/75">Maintain versioned charts of accounts, account-code policy, account restrictions, currency policy, and approved reporting mappings within the selected accounting scope.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter for the selected scope. There is no approved GL read endpoint yet; accepted mutation results refresh these safe projections. Reporting mapping references are submitted only when their approval evidence is available.</p>
    <Panel title="Charts of accounts in the selected scope" description="Chart identity is unique for the owning ledger, code policy, and overlapping inclusive effective dates. Every accepted change retains its prior revision.">
      <DataTable caption="Safe chart-of-accounts projections" columns={chartColumns} rows={charts} getRowKey={(record) => record.id} emptyMessage="No charts of accounts are available in this scope." />
      <div className="mt-4 flex flex-wrap gap-3"><Button variant="secondary" onClick={() => selectChart(null)}>New chart</Button><span className="self-center text-sm text-base-content/70">Selected revision: {selectedChart ? `v${selectedChart.version}` : 'new'}</span></div>
    </Panel>
    <ChartEditor key={`${currentScope.id}:${selectedChart?.id ?? 'new'}`} scopeId={currentScope.id} initial={selectedChart} charts={charts} onAccepted={(record) => { chartReadAdapter.set(record.id, record); refresh(); selectChart(record.id) }} />
    <Panel title="Accounts and reporting mappings" description="Accounts preserve code, name, type, normal balance, restrictions, currency policy, approved reporting mappings, dates, and revision evidence.">
      <DataTable caption="Safe account projections" columns={accountColumns} rows={accounts} getRowKey={(record) => record.id} emptyMessage="No accounts are available in this scope." />
      <div className="mt-4 flex flex-wrap gap-3"><Button variant="secondary" onClick={() => selectAccount(null)}>New account</Button><span className="self-center text-sm text-base-content/70">Selected revision: {selectedAccount ? `v${selectedAccount.version}` : 'new'}</span></div>
    </Panel>
    <AccountEditor key={`${currentScope.id}:${selectedAccount?.id ?? 'new'}`} scopeId={currentScope.id} charts={charts} initial={selectedAccount} onAccepted={(record) => { accountReadAdapter.set(record.id, record); refresh(); selectAccount(record.id) }} />
    <Panel title="Configuration control status" description="Current state, dependent impact, approval, validation, blocked actions, and safe recovery remain visible before the next material change.">
      <div className="grid gap-4 lg:grid-cols-2">
        {selectedChart ? <ControlStatus title={`Chart · ${selectedChart.id}`} ownerLabel="Owning ledger" owner={selectedChart.ledgerId} status={selectedChart.lifecycleStatus} from={selectedChart.effectiveDateFrom} to={selectedChart.effectiveDateTo} approval={selectedChart.approvalStatus} validation={selectedChart.validationOutcome} version={selectedChart.version} revision={selectedChart.revisionNumber} nextAction={selectedChart.nextAction} /> : <p className="text-sm text-base-content/70">Select a chart to review its control status.</p>}
        {selectedAccount ? <ControlStatus title={`Account · ${selectedAccount.accountCode}`} ownerLabel="Owning chart" owner={selectedAccount.chartOfAccountsId} status={selectedAccount.lifecycleStatus} from={selectedAccount.effectiveDateFrom} to={selectedAccount.effectiveDateTo} approval={selectedAccount.approvalStatus} validation={selectedAccount.validationOutcome} version={selectedAccount.version} revision={selectedAccount.revisionNumber} nextAction={selectedAccount.nextAction} /> : <p className="text-sm text-base-content/70">Select an account to review its control status.</p>}
      </div>
    </Panel>
  </section>
}

function ControlStatus({ title, ownerLabel, owner, status, from, to, approval, validation, version, revision, nextAction }: { title: string; ownerLabel: string; owner: string; status: GlLifecycleStatus; from: string; to?: string; approval: string; validation: string; version: number; revision: number; nextAction: string }) {
  const blocked = status === 'retired' || validation !== 'valid'
  const dependentImpact = title.startsWith('Chart') ? 'Future account eligibility and journal validation use the accepted chart version.' : 'Future journal validation uses this accepted account version; established journal facts are unchanged.'
  return <article aria-label={`${title} control status`} className="rounded-box border border-base-300 bg-base-200/30 p-4"><h3 className="font-semibold">{title}</h3><dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2"><div><dt className="text-base-content/65">Lifecycle</dt><dd className="font-medium">{status}</dd></div><div><dt className="text-base-content/65">Effective interval</dt><dd className="font-medium">{from} — {to ?? 'open'}</dd></div><div><dt className="text-base-content/65">{ownerLabel}</dt><dd className="font-medium">{owner}</dd></div><div><dt className="text-base-content/65">Version / revision</dt><dd className="font-medium">v{version} / r{revision}</dd></div><div><dt className="text-base-content/65">Approval</dt><dd className="font-medium">{approval}</dd></div><div><dt className="text-base-content/65">Validation</dt><dd className="font-medium">{validation}</dd></div><div><dt className="text-base-content/65">Dependent impact</dt><dd className="font-medium">{dependentImpact}</dd></div><div><dt className="text-base-content/65">Blocked action</dt><dd className="font-medium">{blocked ? 'Further maintenance' : 'None'}</dd></div><div><dt className="text-base-content/65">Blocking reason</dt><dd className="font-medium">{blocked ? (status === 'retired' ? 'Retired configuration is immutable.' : `Validation outcome: ${validation}.`) : 'No blocking reason reported.'}</dd></div></dl><p className="mt-3 text-sm"><span className="font-semibold">Recovery / next action:</span> {nextAction}</p></article>
}

function ChartEditor({ scopeId, initial, charts, onAccepted }: { scopeId: string; initial?: GlChartRecord; charts: GlChartRecord[]; onAccepted: (record: GlChartRecord) => void }) {
  const { setHasUnsavedChanges } = useScopeContext()
  const [ledgerId, setLedgerId] = useState(initial?.ledgerId ?? 'ledger-vietnam')
  const [policy, setPolicy] = useState(initial?.accountCodePolicy ?? 'NNNN-NN')
  const [status, setStatus] = useState<GlLifecycleStatus>(initial?.lifecycleStatus ?? 'draft')
  const [from, setFrom] = useState(initial?.effectiveDateFrom ?? '2026-01-01')
  const [to, setTo] = useState(initial?.effectiveDateTo ?? '')
  const [notice, setNotice] = useState('Review the current chart version before submitting a material change.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setValidationError('')
    const normalizedPolicy = policy.trim()
    if (!ledgerId || !normalizedPolicy || !from || (to && to < from)) { setValidationError('Enter an owning ledger, account-code policy, and valid inclusive effective-date range.'); return }
    if (!lifecycleTransitionIsValid(initial?.lifecycleStatus, status)) { setValidationError(`The lifecycle transition from ${initial?.lifecycleStatus} to ${status} is not allowed.`); return }
    if (charts.some((candidate) => candidate.id !== initial?.id && candidate.ledgerId === ledgerId && candidate.accountCodePolicy === normalizedPolicy && rangesOverlap(from, to, candidate.effectiveDateFrom, candidate.effectiveDateTo ?? ''))) { setValidationError('Another chart uses this ledger and account-code policy in an overlapping effective-date range.'); return }
    setSaving(true); setNotice('Submitting the idempotent chart command…')
    try {
      const response = await glMaintainChartsOfAccounts({ body: { commandId: crypto.randomUUID(), expectedVersion: initial?.version, accountingScopeId: scopeId, data: { action: initial ? 'update' : 'create', chartOfAccountsId: initial?.id, ledgerId, accountCodePolicy: normalizedPolicy, lifecycleStatus: status, effectiveDateFrom: from, effectiveDateTo: to || undefined } }, headers: { 'Idempotency-Key': crypto.randomUUID(), 'If-Match': initial ? `"${initial.version}"` : undefined, 'X-Accounting-Scope-Id': scopeId } })
      if (response.error) { setNotice(errorDetail(response.error)); return }
      const projection = response.data?.data?.chartOfAccounts
      if (!projection) { setNotice('The chart command returned no safe chart projection.'); return }
      const record: GlChartRecord = { id: projection.id, scopeId: projection.accountingScopeId, ledgerId: projection.ledgerId, accountCodePolicy: projection.accountCodePolicy, lifecycleStatus: projection.lifecycleStatus === 'active' || projection.lifecycleStatus === 'suspended' || projection.lifecycleStatus === 'retired' ? projection.lifecycleStatus : 'draft', effectiveDateFrom: projection.effectiveDateFrom.slice(0, 10), effectiveDateTo: dateValue(projection.effectiveDateTo), approvalStatus: projection.approvalStatus, validationOutcome: projection.validationOutcome, nextAction: projection.nextAction, version: projection.version, revisionNumber: projection.revisionNumber }
      onAccepted(record); setHasUnsavedChanges(false); setNotice(`Accepted chart revision v${record.version}. Safe local/read-adapter state was refreshed from the live mutation response.`)
    } catch (error) { setNotice(error instanceof Error ? error.message : 'The chart command could not reach the live API.') } finally { setSaving(false) }
  }

  return <Panel title={initial ? `Maintain chart: ${initial.id}` : 'Create chart of accounts'} description="The command includes owning ledger, account-code policy, lifecycle, effective dates, authorization, idempotency, and If-Match version evidence."><div className="grid gap-4 md:grid-cols-2"><Field id="gl-chart-ledger" label="Owning ledger" value={ledgerId} onChange={(event) => { setLedgerId(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-chart-policy" label="Account-code policy" value={policy} onChange={(event) => { setPolicy(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><label className="form-control w-full gap-2"><span className="label font-medium">Lifecycle status</span><Select aria-label="Chart lifecycle status" value={status} onChange={(event) => { setStatus(event.target.value as GlLifecycleStatus); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></label><Field id="gl-chart-effective-from" label="Effective from" type="date" value={from} onChange={(event) => { setFrom(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="gl-chart-effective-to" label="Effective to" type="date" value={to} onChange={(event) => { setTo(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /></div><div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[status]} label={`${status} · v${initial?.version ?? 1}`} /><span className="text-sm text-base-content/70">Approval: {initial?.approvalStatus ?? 'not-required'} · Revision {initial?.revisionNumber ?? 1}</span></div>{validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}<p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p><div className="mt-4"><Button onClick={() => void save()} loading={saving} disabled={initial?.lifecycleStatus === 'retired'}>{initial ? 'Save chart revision' : 'Create chart'}</Button></div></Panel>
}

function AccountEditor({ scopeId, charts, initial, onAccepted }: { scopeId: string; charts: GlChartRecord[]; initial?: GlAccountRecord; onAccepted: (record: GlAccountRecord) => void }) {
  const { setHasUnsavedChanges } = useScopeContext()
  const [chartId, setChartId] = useState(initial?.chartOfAccountsId ?? charts[0]?.id ?? '')
  const [code, setCode] = useState(initial?.accountCode ?? '1000')
  const [name, setName] = useState(initial?.accountName ?? 'Cash')
  const [type, setType] = useState(initial?.accountType ?? 'asset')
  const [normalBalance, setNormalBalance] = useState<'debit' | 'credit'>(initial?.normalBalance ?? 'debit')
  const [status, setStatus] = useState<GlLifecycleStatus>(initial?.lifecycleStatus ?? 'draft')
  const [restrictions, setRestrictions] = useState(initial?.restrictions.join(', ') ?? 'postable')
  const [currencyPolicy, setCurrencyPolicy] = useState(initial?.currencyPolicy ?? 'functional-or-transaction')
  const [reportDefinitionId, setReportDefinitionId] = useState(initial?.reportingMappings[0]?.reportingDefinitionId ?? '')
  const [reportLineCode, setReportLineCode] = useState(initial?.reportingMappings[0]?.reportingLineCode ?? '')
  const [mappingApproved, setMappingApproved] = useState(initial?.reportingMappings[0]?.approved ?? false)
  const [from, setFrom] = useState(initial?.effectiveDateFrom ?? '2026-01-01')
  const [to, setTo] = useState(initial?.effectiveDateTo ?? '')
  const [notice, setNotice] = useState('Review restrictions and approved reporting references before submitting a material change.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setValidationError('')
    const normalizedCode = code.trim()
    const normalizedName = name.trim()
    const normalizedType = type.trim().toLowerCase()
    const normalizedRestrictions = restrictions.split(',').map((value) => value.trim()).filter(Boolean)
    if (!chartId || !normalizedCode || !normalizedName || !normalizedType || !currencyPolicy.trim() || !from || (to && to < from)) { setValidationError('Enter a chart, account code/name/type, currency policy, and valid inclusive effective-date range.'); return }
    if (!lifecycleTransitionIsValid(initial?.lifecycleStatus, status)) { setValidationError(`The lifecycle transition from ${initial?.lifecycleStatus} to ${status} is not allowed.`); return }
    const hasMappingInput = reportDefinitionId.trim() || reportLineCode.trim()
    if (hasMappingInput && (!reportDefinitionId.trim() || !reportLineCode.trim() || !mappingApproved)) { setValidationError('A reporting mapping requires a reporting definition, line code, and explicit approval evidence.'); return }
    const mapping = hasMappingInput ? [{ reportingDefinitionId: reportDefinitionId.trim(), reportingLineCode: reportLineCode.trim(), approved: mappingApproved, effectiveDateFrom: from, effectiveDateTo: to || undefined }] : undefined
    setSaving(true); setNotice('Submitting the idempotent account and mapping command…')
    try {
      const response = await glMaintainAccountsAndReportingMappings({ body: { commandId: crypto.randomUUID(), expectedVersion: initial?.version, accountingScopeId: scopeId, data: { action: initial ? 'update' : 'create', accountId: initial?.id, chartOfAccountsId: chartId, accountCode: normalizedCode, accountName: normalizedName, accountType: normalizedType, normalBalance, lifecycleStatus: status, restrictions: normalizedRestrictions.map((restrictionCode) => ({ restrictionCode })), currencyPolicy: currencyPolicy.trim(), reportingMappings: mapping, effectiveDateFrom: from, effectiveDateTo: to || undefined } }, headers: { 'Idempotency-Key': crypto.randomUUID(), 'If-Match': initial ? `"${initial.version}"` : undefined, 'X-Accounting-Scope-Id': scopeId } })
      if (response.error) { setNotice(errorDetail(response.error)); return }
      const projection = response.data?.data?.account
      if (!projection) { setNotice('The account command returned no safe account projection.'); return }
      const reportingMappings = projection.reportingMappings.map((mapping) => ({ reportingDefinitionId: mapping.reportingDefinitionId, reportingLineCode: mapping.reportingLineCode, approved: mapping.approved, effectiveDateFrom: mapping.effectiveDateFrom, effectiveDateTo: mapping.effectiveDateTo ?? undefined }))
      const record: GlAccountRecord = { id: projection.id, scopeId: projection.accountingScopeId, chartOfAccountsId: projection.chartOfAccountsId, accountCode: projection.accountCode, accountName: projection.accountName, accountType: projection.accountType, normalBalance: projection.normalBalance, lifecycleStatus: projection.lifecycleStatus === 'active' || projection.lifecycleStatus === 'suspended' || projection.lifecycleStatus === 'retired' ? projection.lifecycleStatus : 'draft', restrictions: projection.restrictions.map((restriction) => restriction.restrictionCode), currencyPolicy: projection.currencyPolicy, reportingMappings, reportingMappingCount: reportingMappings.length, approvedReportingMappingCount: reportingMappings.filter((candidate) => candidate.approved).length, effectiveDateFrom: projection.effectiveDateFrom.slice(0, 10), effectiveDateTo: dateValue(projection.effectiveDateTo), approvalStatus: projection.approvalStatus, validationOutcome: projection.validationOutcome, nextAction: projection.nextAction, version: projection.version, revisionNumber: projection.revisionNumber }
      onAccepted(record); setHasUnsavedChanges(false); setNotice(`Accepted account revision v${record.version}. Safe local/read-adapter state was refreshed from the live mutation response.`)
    } catch (error) { setNotice(error instanceof Error ? error.message : 'The account command could not reach the live API.') } finally { setSaving(false) }
  }

  return <Panel title={initial ? `Maintain account: ${initial.accountCode}` : 'Create account'} description="The command validates code uniqueness, type/normal balance, restrictions, currency policy, effective dates, and approved reporting mappings before commit."><div className="grid gap-4 md:grid-cols-2"><label className="form-control w-full gap-2"><span className="label font-medium">Chart of accounts</span><Select aria-label="Account chart of accounts" value={chartId} onChange={(event) => { setChartId(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}>{charts.map((chart) => <option key={chart.id} value={chart.id}>{chart.id} · {chart.accountCodePolicy}</option>)}</Select></label><Field id="gl-account-code" label="Account code" value={code} onChange={(event) => { setCode(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-account-name" label="Account name" value={name} onChange={(event) => { setName(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-account-type" label="Account type" value={type} onChange={(event) => { setType(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><label className="form-control w-full gap-2"><span className="label font-medium">Normal balance</span><Select aria-label="Account normal balance" value={normalBalance} onChange={(event) => { setNormalBalance(event.target.value as 'debit' | 'credit'); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}><option value="debit">Debit</option><option value="credit">Credit</option></Select></label><label className="form-control w-full gap-2"><span className="label font-medium">Lifecycle status</span><Select aria-label="Account lifecycle status" value={status} onChange={(event) => { setStatus(event.target.value as GlLifecycleStatus); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></label><Field id="gl-account-restrictions" label="Restrictions (comma separated)" value={restrictions} onChange={(event) => { setRestrictions(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /><Field id="gl-account-currency-policy" label="Currency policy" value={currencyPolicy} onChange={(event) => { setCurrencyPolicy(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-account-report-definition" label="Reporting definition ID" value={reportDefinitionId} onChange={(event) => { setReportDefinitionId(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /><Field id="gl-account-report-line" label="Reporting line code" value={reportLineCode} onChange={(event) => { setReportLineCode(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /><label className="form-control w-full gap-2"><span className="label font-medium">Mapping approval</span><Select aria-label="Reporting mapping approval" value={mappingApproved ? 'approved' : 'not-approved'} onChange={(event) => { setMappingApproved(event.target.value === 'approved'); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}><option value="not-approved">Not approved</option><option value="approved">Approved</option></Select></label><Field id="gl-account-effective-from" label="Effective from" type="date" value={from} onChange={(event) => { setFrom(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="gl-account-effective-to" label="Effective to" type="date" value={to} onChange={(event) => { setTo(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /></div><div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[status]} label={`${status} · v${initial?.version ?? 1}`} /><span className="text-sm text-base-content/70">Approval: {initial?.approvalStatus ?? 'not-required'} · Revision {initial?.revisionNumber ?? 1}</span></div>{validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}<p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p><div className="mt-4"><Button onClick={() => void save()} loading={saving} disabled={initial?.lifecycleStatus === 'retired' || !charts.length}>{initial ? 'Save account revision' : 'Create account'}</Button></div></Panel>
}

