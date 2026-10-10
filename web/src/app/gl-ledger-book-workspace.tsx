import { useMemo, useState } from 'react'

import { glMaintainAccountingBooks, glMaintainLedgers } from '@/generated/api/sdk.gen'
import { Button, DataTable, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import { useScopeContext } from '@/lib/scope/scope-context'

export type GlLifecycleStatus = 'draft' | 'active' | 'suspended' | 'retired'

export type GlLedgerRecord = {
  id: string
  scopeId: string
  legalEntityId: string
  ledgerType: string
  functionalCurrency: string
  fiscalCalendarId: string
  lifecycleStatus: GlLifecycleStatus
  effectiveDateFrom: string
  effectiveDateTo?: string
  approvalStatus: string
  validationOutcome: string
  nextAction: string
  version: number
  revisionNumber: number
}

export type GlAccountingBookRecord = {
  id: string
  scopeId: string
  ledgerId: string
  bookType: string
  accountingBasis: string
  postingPolicyVersion: string
  lifecycleStatus: GlLifecycleStatus
  effectiveDateFrom: string
  effectiveDateTo?: string
  approvalStatus: string
  validationOutcome: string
  nextAction: string
  version: number
  revisionNumber: number
}

const initialLedgers: GlLedgerRecord[] = [
  { id: 'ledger-vietnam', scopeId: 'scope-vietnam-statutory', legalEntityId: 'entity-vietnam', ledgerType: 'primary', functionalCurrency: 'VND', fiscalCalendarId: 'fiscal-calendar-vietnam', lifecycleStatus: 'active', effectiveDateFrom: '2026-01-01', approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 2, revisionNumber: 2 },
  { id: 'ledger-singapore', scopeId: 'scope-singapore-management', legalEntityId: 'entity-singapore', ledgerType: 'regional', functionalCurrency: 'SGD', fiscalCalendarId: 'fiscal-calendar-singapore', lifecycleStatus: 'draft', effectiveDateFrom: '2026-02-01', effectiveDateTo: '2026-12-31', approvalStatus: 'pending', validationOutcome: 'valid', nextAction: 'submit for approval', version: 1, revisionNumber: 1 },
]

const initialBooks: GlAccountingBookRecord[] = [
  { id: 'book-vietnam-statutory', scopeId: 'scope-vietnam-statutory', ledgerId: 'ledger-vietnam', bookType: 'statutory', accountingBasis: 'accrual', postingPolicyVersion: 'posting-v1', lifecycleStatus: 'active', effectiveDateFrom: '2026-01-01', approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 2, revisionNumber: 2 },
  { id: 'book-singapore-management', scopeId: 'scope-singapore-management', ledgerId: 'ledger-singapore', bookType: 'management', accountingBasis: 'accrual', postingPolicyVersion: 'posting-v1', lifecycleStatus: 'draft', effectiveDateFrom: '2026-02-01', effectiveDateTo: '2026-12-31', approvalStatus: 'pending', validationOutcome: 'valid', nextAction: 'submit for approval', version: 1, revisionNumber: 1 },
]

const ledgerReadAdapter = new Map(initialLedgers.map((record) => [record.id, record]))
const bookReadAdapter = new Map(initialBooks.map((record) => [record.id, record]))

export function readSafeGlLedgers(scopeId: string) {
  return Array.from(ledgerReadAdapter.values()).filter((record) => record.scopeId === scopeId).sort((left, right) => left.ledgerType.localeCompare(right.ledgerType) || left.id.localeCompare(right.id))
}

export function readSafeGlAccountingBooks(scopeId: string) {
  return Array.from(bookReadAdapter.values()).filter((record) => record.scopeId === scopeId).sort((left, right) => left.bookType.localeCompare(right.bookType) || left.id.localeCompare(right.id))
}

function upsertLedger(record: GlLedgerRecord) {
  ledgerReadAdapter.set(record.id, record)
}

function upsertBook(record: GlAccountingBookRecord) {
  bookReadAdapter.set(record.id, record)
}

const statusState: Record<GlLifecycleStatus, SemanticState> = { draft: 'pending', active: 'success', suspended: 'warning', retired: 'disabled' }

function lifecycleTransitionIsValid(current: GlLifecycleStatus | undefined, next: GlLifecycleStatus) {
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

function errorDetail(error: unknown) {
  if (error && typeof error === 'object' && 'detail' in error && typeof error.detail === 'string') return error.detail
  return 'The general-ledger command could not be completed. Review the current version and retry safely.'
}

export function GlLedgerBookWorkspace() {
  const { currentScope } = useScopeContext()
  const [adapterRevision, setAdapterRevision] = useState(0)
  const ledgers = useMemo(() => currentScope ? readSafeGlLedgers(currentScope.id) : [], [currentScope?.id, adapterRevision])
  const books = useMemo(() => currentScope ? readSafeGlAccountingBooks(currentScope.id) : [], [currentScope?.id, adapterRevision])
  const [selectedLedgerId, setSelectedLedgerId] = useState<string | null>(ledgers[0]?.id ?? null)
  const [selectedBookId, setSelectedBookId] = useState<string | null>(books[0]?.id ?? null)
  const selectedLedger = ledgers.find((record) => record.id === selectedLedgerId)
  const selectedBook = books.find((record) => record.id === selectedBookId)

  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-04</p><h2 className="mt-1 text-2xl font-semibold">Ledger and accounting-book configuration</h2><p className="mt-2">Select an accounting scope before maintaining configuration.</p></section>
  }

  const ledgerColumns = [
    { key: 'type', header: 'Ledger', rowHeader: true, render: (record: GlLedgerRecord) => <button type="button" className="link link-primary font-semibold" onClick={() => setSelectedLedgerId(record.id)}>{record.ledgerType}</button> },
    { key: 'currency', header: 'Currency', render: (record: GlLedgerRecord) => record.functionalCurrency },
    { key: 'status', header: 'State', render: (record: GlLedgerRecord) => <StatusBadge state={statusState[record.lifecycleStatus]} label={record.lifecycleStatus} /> },
    { key: 'effective', header: 'Effective interval', render: (record: GlLedgerRecord) => `${record.effectiveDateFrom} — ${record.effectiveDateTo ?? 'open'}` },
    { key: 'version', header: 'Version', render: (record: GlLedgerRecord) => `v${record.version}` },
  ] as const
  const bookColumns = [
    { key: 'type', header: 'Book', rowHeader: true, render: (record: GlAccountingBookRecord) => <button type="button" className="link link-primary font-semibold" onClick={() => setSelectedBookId(record.id)}>{record.bookType}</button> },
    { key: 'basis', header: 'Basis', render: (record: GlAccountingBookRecord) => record.accountingBasis },
    { key: 'status', header: 'State', render: (record: GlAccountingBookRecord) => <StatusBadge state={statusState[record.lifecycleStatus]} label={record.lifecycleStatus} /> },
    { key: 'effective', header: 'Effective interval', render: (record: GlAccountingBookRecord) => `${record.effectiveDateFrom} — ${record.effectiveDateTo ?? 'open'}` },
    { key: 'version', header: 'Version', render: (record: GlAccountingBookRecord) => `v${record.version}` },
  ] as const

  return <section aria-labelledby="gl-scr-04-title" className="space-y-6">
    <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">GL-SCR-04</p><h2 id="gl-scr-04-title" className="mt-1 text-2xl font-semibold">Ledger and accounting-book configuration</h2><p className="mt-2 max-w-4xl text-base-content/75">Maintain the legal-entity ledger and its accounting books with explicit effective dates, lifecycle state, posting policy, optimistic version, authorization, and audit evidence.</p></div>
    <p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Read source: local safe adapter for the selected scope. No approved GL read endpoint exists yet; successful live mutation responses refresh these projections without exposing hidden financial data.</p>
    <Panel title="Ledgers in the selected scope" description="Ledger identity is unique across overlapping inclusive effective-date ranges for the legal entity, type, currency, and fiscal calendar.">
      <DataTable caption="Safe ledger projections" columns={ledgerColumns} rows={ledgers} getRowKey={(record) => record.id} emptyMessage="No ledgers are available in this scope." />
      <div className="mt-4 flex flex-wrap gap-3"><Button variant="secondary" onClick={() => setSelectedLedgerId(null)}>New ledger</Button><span className="self-center text-sm text-base-content/70">Selected revision: {selectedLedger ? `v${selectedLedger.version}` : 'new'}</span></div>
    </Panel>
    <LedgerEditor key={`${currentScope.id}:${selectedLedger?.id ?? 'new'}`} scopeId={currentScope.id} legalEntityId={currentScope.legalEntity.id} initial={selectedLedger} onAccepted={(record) => { upsertLedger(record); setAdapterRevision((revision) => revision + 1); setSelectedLedgerId(record.id) }} />
    <Panel title="Accounting books in the selected scope" description="Each book remains related to an owning ledger and keeps basis, book type, posting-policy version, lifecycle, dates, and revision history explicit.">
      <DataTable caption="Safe accounting-book projections" columns={bookColumns} rows={books} getRowKey={(record) => record.id} emptyMessage="No accounting books are available in this scope." />
      <div className="mt-4 flex flex-wrap gap-3"><Button variant="secondary" onClick={() => setSelectedBookId(null)}>New accounting book</Button><span className="self-center text-sm text-base-content/70">Selected revision: {selectedBook ? `v${selectedBook.version}` : 'new'}</span></div>
    </Panel>
    <AccountingBookEditor key={`${currentScope.id}:${selectedBook?.id ?? 'new'}`} scopeId={currentScope.id} ledgers={ledgers} initial={selectedBook} onAccepted={(record) => { upsertBook(record); setAdapterRevision((revision) => revision + 1); setSelectedBookId(record.id) }} />
    <Panel title="Configuration control status" description="The selected safe projection keeps lifecycle, ownership, evidence, blocking, and recovery guidance visible before the next material action.">
      <div className="grid gap-4 lg:grid-cols-2">
        {selectedLedger ? <ConfigurationControlStatus title={`Ledger · ${selectedLedger.ledgerType}`} ownerLabel="Legal entity" owner={selectedLedger.legalEntityId} status={selectedLedger.lifecycleStatus} effectiveDateFrom={selectedLedger.effectiveDateFrom} effectiveDateTo={selectedLedger.effectiveDateTo} approvalStatus={selectedLedger.approvalStatus} validationOutcome={selectedLedger.validationOutcome} version={selectedLedger.version} revisionNumber={selectedLedger.revisionNumber} nextAction={selectedLedger.nextAction} /> : <p className="text-sm text-base-content/70">Select a ledger to review its control status.</p>}
        {selectedBook ? <ConfigurationControlStatus title={`Accounting book · ${selectedBook.bookType}`} ownerLabel="Owning ledger" owner={selectedBook.ledgerId} status={selectedBook.lifecycleStatus} effectiveDateFrom={selectedBook.effectiveDateFrom} effectiveDateTo={selectedBook.effectiveDateTo} approvalStatus={selectedBook.approvalStatus} validationOutcome={selectedBook.validationOutcome} version={selectedBook.version} revisionNumber={selectedBook.revisionNumber} nextAction={selectedBook.nextAction} /> : <p className="text-sm text-base-content/70">Select an accounting book to review its control status.</p>}
      </div>
    </Panel>
  </section>
}

function ConfigurationControlStatus({ title, ownerLabel, owner, status, effectiveDateFrom, effectiveDateTo, approvalStatus, validationOutcome, version, revisionNumber, nextAction }: { title: string; ownerLabel: string; owner: string; status: GlLifecycleStatus; effectiveDateFrom: string; effectiveDateTo?: string; approvalStatus: string; validationOutcome: string; version: number; revisionNumber: number; nextAction: string }) {
  const blocked = status === 'retired' || validationOutcome !== 'valid'
  const blockedAction = status === 'retired' ? 'Further maintenance' : validationOutcome !== 'valid' ? 'Submit configuration change' : 'None'
  const blockingReason = status === 'retired' ? 'Retired configuration is immutable.' : validationOutcome !== 'valid' ? `Validation outcome: ${validationOutcome}.` : 'No blocking reason reported.'
  return <article aria-label={`${title} control status`} className="rounded-box border border-base-300 bg-base-200/30 p-4">
    <h3 className="font-semibold">{title}</h3>
    <dl className="mt-3 grid gap-2 text-sm sm:grid-cols-2">
      <div><dt className="text-base-content/65">Lifecycle</dt><dd className="font-medium">{status}</dd></div>
      <div><dt className="text-base-content/65">Effective interval</dt><dd className="font-medium">{effectiveDateFrom} — {effectiveDateTo ?? 'open'}</dd></div>
      <div><dt className="text-base-content/65">{ownerLabel}</dt><dd className="font-medium">{owner}</dd></div>
      <div><dt className="text-base-content/65">Version / revision</dt><dd className="font-medium">v{version} / r{revisionNumber}</dd></div>
      <div><dt className="text-base-content/65">Approval</dt><dd className="font-medium">{approvalStatus}</dd></div>
      <div><dt className="text-base-content/65">Validation</dt><dd className="font-medium">{validationOutcome}</dd></div>
      <div><dt className="text-base-content/65">Blocked action</dt><dd className="font-medium">{blocked ? blockedAction : 'None'}</dd></div>
      <div><dt className="text-base-content/65">Blocking reason</dt><dd className="font-medium">{blockingReason}</dd></div>
    </dl>
    <p className="mt-3 text-sm"><span className="font-semibold">Recovery / next action:</span> {nextAction}</p>
  </article>
}

function LedgerEditor({ scopeId, legalEntityId, initial, onAccepted }: { scopeId: string; legalEntityId: string; initial?: GlLedgerRecord; onAccepted: (record: GlLedgerRecord) => void }) {
  const { setHasUnsavedChanges } = useScopeContext()
  const [ledgerType, setLedgerType] = useState(initial?.ledgerType ?? 'primary')
  const [currency, setCurrency] = useState(initial?.functionalCurrency ?? 'USD')
  const [status, setStatus] = useState<GlLifecycleStatus>(initial?.lifecycleStatus ?? 'draft')
  const [effectiveFrom, setEffectiveFrom] = useState(initial?.effectiveDateFrom ?? '2026-01-01')
  const [effectiveTo, setEffectiveTo] = useState(initial?.effectiveDateTo ?? '')
  const [notice, setNotice] = useState('Review the current ledger version before submitting a material change.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setValidationError('')
    const normalizedType = ledgerType.trim().toLowerCase()
    const normalizedCurrency = currency.trim().toUpperCase()
    if (!normalizedType || !/^[A-Z]{3}$/.test(normalizedCurrency) || !effectiveFrom || (effectiveTo && effectiveTo < effectiveFrom)) {
      setValidationError('Enter a ledger type, a three-letter currency, and a valid inclusive effective-date range.')
      return
    }
    if (!lifecycleTransitionIsValid(initial?.lifecycleStatus, status)) {
      setValidationError(`The lifecycle transition from ${initial?.lifecycleStatus} to ${status} is not allowed.`)
      return
    }
    const overlap = readSafeGlLedgers(scopeId).some((candidate) => candidate.id !== initial?.id && candidate.legalEntityId === legalEntityId && candidate.ledgerType === normalizedType && candidate.functionalCurrency === normalizedCurrency && rangesOverlap(effectiveFrom, effectiveTo, candidate.effectiveDateFrom, candidate.effectiveDateTo ?? ''))
    if (overlap) {
      setValidationError('Another ledger uses this legal-entity identity in an overlapping effective-date range.')
      return
    }
    setSaving(true)
    setNotice('Submitting the idempotent ledger command…')
    try {
      const response = await glMaintainLedgers({
        body: { commandId: crypto.randomUUID(), expectedVersion: initial?.version, accountingScopeId: scopeId, data: { action: initial ? 'update' : 'create', ledgerId: initial?.id, legalEntityId, ledgerType: normalizedType, functionalCurrency: normalizedCurrency, fiscalCalendarId: initial?.fiscalCalendarId ?? 'fiscal-calendar-pending', lifecycleStatus: status, effectiveDateFrom: effectiveFrom, effectiveDateTo: effectiveTo || undefined } },
        headers: { 'Idempotency-Key': crypto.randomUUID(), 'If-Match': initial ? `"${initial.version}"` : undefined, 'X-Accounting-Scope-Id': scopeId },
      })
      if (response.error) { setNotice(errorDetail(response.error)); return }
      const projection = response.data?.data?.ledger
      if (!projection) { setNotice('The ledger command returned no safe ledger projection.'); return }
      const nextRecord: GlLedgerRecord = { id: projection.id, scopeId: projection.accountingScopeId, legalEntityId: projection.legalEntityId, ledgerType: projection.ledgerType, functionalCurrency: projection.functionalCurrency, fiscalCalendarId: projection.fiscalCalendarId, lifecycleStatus: projection.lifecycleStatus === 'active' || projection.lifecycleStatus === 'suspended' || projection.lifecycleStatus === 'retired' ? projection.lifecycleStatus : 'draft', effectiveDateFrom: projection.effectiveDateFrom.slice(0, 10), effectiveDateTo: projection.effectiveDateTo?.slice(0, 10) || undefined, approvalStatus: projection.approvalStatus, validationOutcome: projection.validationOutcome, nextAction: projection.nextAction, version: projection.version, revisionNumber: projection.revisionNumber }
      onAccepted(nextRecord)
      setHasUnsavedChanges(false)
      setNotice(`Accepted ledger revision v${nextRecord.version}. Safe local/read-adapter state was refreshed from the live mutation response.`)
    } catch (error) { setNotice(error instanceof Error ? error.message : 'The ledger command could not reach the live API.') } finally { setSaving(false) }
  }

  return <Panel title={initial ? `Maintain ledger: ${initial.ledgerType}` : 'Create ledger'} description="The command includes legal entity, functional currency, fiscal calendar, lifecycle, effective dates, correlation, idempotency, and If-Match version evidence.">
    <div className="grid gap-4 md:grid-cols-2"><Field id="gl-ledger-type" label="Ledger type" value={ledgerType} onChange={(event) => { setLedgerType(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-ledger-currency" label="Functional currency" value={currency} onChange={(event) => { setCurrency(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-ledger-legal-entity" label="Legal entity" value={legalEntityId} readOnly /><Field id="gl-ledger-fiscal-calendar" label="Fiscal calendar" value={initial?.fiscalCalendarId ?? 'Selected calendar reference'} readOnly /><label className="form-control w-full gap-2"><span className="label font-medium">Lifecycle status</span><Select aria-label="Lifecycle status" value={status} onChange={(event) => { setStatus(event.target.value as GlLifecycleStatus); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></label><Field id="gl-ledger-effective-from" label="Effective from" type="date" value={effectiveFrom} onChange={(event) => { setEffectiveFrom(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="gl-ledger-effective-to" label="Effective to" type="date" value={effectiveTo} onChange={(event) => { setEffectiveTo(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /></div>
    <div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[status]} label={`${status} · v${initial?.version ?? 1}`} /><span className="text-sm text-base-content/70">Approval: {initial?.approvalStatus ?? 'not-required'} · Revision {initial?.revisionNumber ?? 1}</span></div>
    {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}<p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p><div className="mt-4"><Button onClick={() => void save()} loading={saving} disabled={initial?.lifecycleStatus === 'retired'}>{initial ? 'Save ledger revision' : 'Create ledger'}</Button></div>
  </Panel>
}

function AccountingBookEditor({ scopeId, ledgers, initial, onAccepted }: { scopeId: string; ledgers: GlLedgerRecord[]; initial?: GlAccountingBookRecord; onAccepted: (record: GlAccountingBookRecord) => void }) {
  const { setHasUnsavedChanges } = useScopeContext()
  const [ledgerId, setLedgerId] = useState(initial?.ledgerId ?? ledgers[0]?.id ?? '')
  const [bookType, setBookType] = useState(initial?.bookType ?? 'statutory')
  const [basis, setBasis] = useState(initial?.accountingBasis ?? 'accrual')
  const [policyVersion, setPolicyVersion] = useState(initial?.postingPolicyVersion ?? 'posting-v1')
  const [status, setStatus] = useState<GlLifecycleStatus>(initial?.lifecycleStatus ?? 'draft')
  const [effectiveFrom, setEffectiveFrom] = useState(initial?.effectiveDateFrom ?? '2026-01-01')
  const [effectiveTo, setEffectiveTo] = useState(initial?.effectiveDateTo ?? '')
  const [notice, setNotice] = useState('Review the owning ledger and current book version before submitting a material change.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)

  const save = async () => {
    setValidationError('')
    const normalizedBookType = bookType.trim().toLowerCase()
    const normalizedBasis = basis.trim().toLowerCase()
    const normalizedPolicy = policyVersion.trim()
    if (!ledgerId || !normalizedBookType || !normalizedBasis || !normalizedPolicy || !effectiveFrom || (effectiveTo && effectiveTo < effectiveFrom)) { setValidationError('Enter an owning ledger, book type, accounting basis, posting-policy version, and valid inclusive effective-date range.'); return }
    if (!lifecycleTransitionIsValid(initial?.lifecycleStatus, status)) { setValidationError(`The lifecycle transition from ${initial?.lifecycleStatus} to ${status} is not allowed.`); return }
    const overlap = readSafeGlAccountingBooks(scopeId).some((candidate) => candidate.id !== initial?.id && candidate.ledgerId === ledgerId && candidate.bookType === normalizedBookType && candidate.accountingBasis === normalizedBasis && rangesOverlap(effectiveFrom, effectiveTo, candidate.effectiveDateFrom, candidate.effectiveDateTo ?? ''))
    if (overlap) { setValidationError('Another accounting book uses this ledger, type, and basis in an overlapping effective-date range.'); return }
    setSaving(true)
    setNotice('Submitting the idempotent accounting-book command…')
    try {
      const response = await glMaintainAccountingBooks({
        body: { commandId: crypto.randomUUID(), expectedVersion: initial?.version, accountingScopeId: scopeId, data: { action: initial ? 'update' : 'create', accountingBookId: initial?.id, ledgerId, bookType: normalizedBookType, accountingBasis: normalizedBasis, postingPolicyVersion: normalizedPolicy, lifecycleStatus: status, effectiveDateFrom: effectiveFrom, effectiveDateTo: effectiveTo || undefined } },
        headers: { 'Idempotency-Key': crypto.randomUUID(), 'If-Match': initial ? `"${initial.version}"` : undefined, 'X-Accounting-Scope-Id': scopeId },
      })
      if (response.error) { setNotice(errorDetail(response.error)); return }
      const projection = response.data?.data?.accountingBook
      if (!projection) { setNotice('The accounting-book command returned no safe accounting-book projection.'); return }
      const nextRecord: GlAccountingBookRecord = { id: projection.id, scopeId: projection.accountingScopeId, ledgerId: projection.ledgerId, bookType: projection.bookType, accountingBasis: projection.accountingBasis, postingPolicyVersion: projection.postingPolicyVersion, lifecycleStatus: projection.lifecycleStatus === 'active' || projection.lifecycleStatus === 'suspended' || projection.lifecycleStatus === 'retired' ? projection.lifecycleStatus : 'draft', effectiveDateFrom: projection.effectiveDateFrom.slice(0, 10), effectiveDateTo: projection.effectiveDateTo?.slice(0, 10) || undefined, approvalStatus: projection.approvalStatus, validationOutcome: projection.validationOutcome, nextAction: projection.nextAction, version: projection.version, revisionNumber: projection.revisionNumber }
      onAccepted(nextRecord)
      setHasUnsavedChanges(false)
      setNotice(`Accepted accounting-book revision v${nextRecord.version}. Safe local/read-adapter state was refreshed from the live mutation response.`)
    } catch (error) { setNotice(error instanceof Error ? error.message : 'The accounting-book command could not reach the live API.') } finally { setSaving(false) }
  }

  return <Panel title={initial ? `Maintain accounting book: ${initial.bookType}` : 'Create accounting book'} description="The book keeps its ledger relationship, basis, book type, posting-policy version, lifecycle, dates, correlation, idempotency, and If-Match version explicit.">
    <div className="grid gap-4 md:grid-cols-2"><label className="form-control w-full gap-2"><span className="label font-medium">Owning ledger</span><Select aria-label="Owning ledger" value={ledgerId} onChange={(event) => { setLedgerId(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}>{ledgers.map((ledger) => <option key={ledger.id} value={ledger.id}>{ledger.ledgerType} · {ledger.functionalCurrency}</option>)}</Select></label><Field id="gl-book-type" label="Book type" value={bookType} onChange={(event) => { setBookType(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-book-basis" label="Accounting basis" value={basis} onChange={(event) => { setBasis(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><Field id="gl-book-policy" label="Posting-policy version" value={policyVersion} onChange={(event) => { setPolicyVersion(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} required /><label className="form-control w-full gap-2"><span className="label font-medium">Lifecycle status</span><Select aria-label="Book lifecycle status" value={status} onChange={(event) => { setStatus(event.target.value as GlLifecycleStatus); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'}><option value="draft">Draft</option><option value="active">Active</option><option value="suspended">Suspended</option><option value="retired">Retired</option></Select></label><Field id="gl-book-effective-from" label="Effective from" type="date" value={effectiveFrom} onChange={(event) => { setEffectiveFrom(event.target.value); setHasUnsavedChanges(true) }} required /><Field id="gl-book-effective-to" label="Effective to" type="date" value={effectiveTo} onChange={(event) => { setEffectiveTo(event.target.value); setHasUnsavedChanges(true) }} disabled={initial?.lifecycleStatus === 'retired'} /></div>
    <div className="mt-4 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[status]} label={`${status} · v${initial?.version ?? 1}`} /><span className="text-sm text-base-content/70">Approval: {initial?.approvalStatus ?? 'not-required'} · Revision {initial?.revisionNumber ?? 1}</span></div>
    {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}<p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p><div className="mt-4"><Button onClick={() => void save()} loading={saving} disabled={initial?.lifecycleStatus === 'retired' || !ledgers.length}>{initial ? 'Save accounting-book revision' : 'Create accounting book'}</Button></div>
  </Panel>
}
