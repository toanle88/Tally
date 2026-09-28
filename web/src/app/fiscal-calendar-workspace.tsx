import { useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'

export type FiscalCalendarPeriodFixture = {
  id: string
  reference: string
  ordinal: number
  startDate: string
  endDate: string
}

export type FiscalCalendarFixture = {
  id: string
  scopeId: string
  calendarType: string
  periodPattern: string
  status: 'draft' | 'active' | 'end_dated'
  effectiveFrom: string
  effectiveTo?: string
  approvalStatus: 'not-required' | 'approved' | 'pending'
  version: number
  revisionNumber: number
  nextAction: string
  impactAvailability: 'available' | 'unavailable'
  periods: FiscalCalendarPeriodFixture[]
}

export const initialFiscalCalendars: FiscalCalendarFixture[] = [
  {
    id: 'fiscal-calendar-vietnam-gregorian', scopeId: 'scope-vietnam-statutory', calendarType: 'gregorian', periodPattern: 'quarterly',
    status: 'active', effectiveFrom: '2026-01-01', approvalStatus: 'approved', version: 3, revisionNumber: 3, nextAction: 'maintain', impactAvailability: 'unavailable',
    periods: [
      { id: 'vn-q1-2026', reference: 'q1', ordinal: 1, startDate: '2026-01-01', endDate: '2026-03-31' },
      { id: 'vn-q2-2026', reference: 'q2', ordinal: 2, startDate: '2026-04-01', endDate: '2026-06-30' },
      { id: 'vn-q3-2026', reference: 'q3', ordinal: 3, startDate: '2026-07-01', endDate: '2026-09-30' },
      { id: 'vn-q4-2026', reference: 'q4', ordinal: 4, startDate: '2026-10-01', endDate: '2026-12-31' },
    ],
  },
  {
    id: 'fiscal-calendar-singapore-445', scopeId: 'scope-singapore-management', calendarType: 'retail', periodPattern: '4-4-5',
    status: 'draft', effectiveFrom: '2026-02-01', approvalStatus: 'pending', version: 1, revisionNumber: 1, nextAction: 'submit for approval', impactAvailability: 'unavailable',
    periods: [
      { id: 'sg-p01-2026', reference: 'period_01', ordinal: 1, startDate: '2026-02-01', endDate: '2026-02-28' },
      { id: 'sg-p02-2026', reference: 'period_02', ordinal: 2, startDate: '2026-03-01', endDate: '2026-03-28' },
    ],
  },
]

export const fiscalCalendarStatusState: Record<FiscalCalendarFixture['status'], SemanticState> = { draft: 'pending', active: 'success', end_dated: 'disabled' }
const fiscalCalendarCodePattern = /^[a-z0-9][a-z0-9_-]{0,63}$/

function fiscalCalendarDefinitionIsValid(calendarType: string, periodPattern: string, effectiveFrom: string, effectiveTo: string, periods: FiscalCalendarPeriodFixture[]) {
  const normalizedType = calendarType.trim().toLowerCase()
  const normalizedPattern = periodPattern.trim().toLowerCase()
  if (!fiscalCalendarCodePattern.test(normalizedType) || !fiscalCalendarCodePattern.test(normalizedPattern) || !effectiveFrom || (effectiveTo && effectiveTo <= effectiveFrom)) return false
  if (!periods.length || periods.some((period) => !fiscalCalendarCodePattern.test(period.reference.trim().toLowerCase()) || !period.startDate || !period.endDate || period.endDate < period.startDate || period.startDate < effectiveFrom || (effectiveTo && period.endDate > effectiveTo))) return false
  const references = new Set<string>()
  return periods.every((period, index) => {
    if (references.has(period.reference.trim().toLowerCase()) || period.ordinal !== index + 1) return false
    references.add(period.reference.trim().toLowerCase())
    return index === 0 || period.startDate > periods[index - 1].endDate
  })
}

export function FiscalCalendarRecord() {
  const [params] = useSearchParams()
  const selected = initialFiscalCalendars.find((calendar) => calendar.id === params.get('fiscalCalendarId')) ?? initialFiscalCalendars[0]
  const [calendar, setCalendar] = useState(selected)
  const [calendarType, setCalendarType] = useState(selected.calendarType)
  const [periodPattern, setPeriodPattern] = useState(selected.periodPattern)
  const [effectiveTo, setEffectiveTo] = useState(selected.effectiveTo ?? '')
  const [periods, setPeriods] = useState(selected.periods)
  const [notice, setNotice] = useState('Review the explicit period definitions and current aggregate version before making a material change.')

  const updatePeriod = (index: number, field: keyof FiscalCalendarPeriodFixture, value: string) => {
    setPeriods((current) => current.map((period, periodIndex) => periodIndex === index ? { ...period, [field]: field === 'ordinal' ? Number(value) : value } : period))
  }

  const save = () => {
    if (!fiscalCalendarDefinitionIsValid(calendarType, periodPattern, calendar.effectiveFrom, effectiveTo.trim(), periods)) {
      setNotice('The calendar metadata, effective interval, and period definitions must be complete, ordered, uniquely referenced, in range, and non-overlapping before they can be accepted.')
      return
    }
    const nextVersion = calendar.version + 1
    const nextEffectiveTo = effectiveTo.trim() || undefined
    const nextStatus = nextEffectiveTo ? 'end_dated' : calendar.status === 'draft' ? 'active' : calendar.status
    setCalendar((current) => ({ ...current, calendarType: calendarType.trim().toLowerCase(), periodPattern: periodPattern.trim().toLowerCase(), effectiveTo: nextEffectiveTo, status: nextStatus, periods, version: nextVersion, revisionNumber: current.revisionNumber + 1, nextAction: nextStatus === 'end_dated' ? 'view history' : 'maintain' }))
    setNotice(`Accepted fiscal-calendar revision v${nextVersion}. Explicit period history was retained; downstream impact remains ${calendar.impactAvailability}.`)
  }

  return <section aria-labelledby="omd-fiscal-calendar-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-SCR-04</p><h2 id="omd-fiscal-calendar-title" className="mt-1 text-2xl font-semibold">Fiscal-calendar editor</h2><p className="mt-2 max-w-3xl text-base-content/75">Maintain the authoritative calendar pattern and explicitly supplied periods. OMD owns these definitions; downstream FiscalPeriod and ledger state are outside this screen.</p></div>
      <RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/master-data/omd-ws-01">Back to worklist</RouterLink>
    </div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <Panel title="Authoritative reference and lifecycle" description="Changes use optimistic concurrency, effective dates, approval evidence, and an immutable fiscal-calendar revision history.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="fiscal-calendar-type" label="Calendar type" value={calendarType} onChange={(event) => setCalendarType(event.target.value)} disabled={calendar.status === 'end_dated'} /><Field id="fiscal-calendar-pattern" label="Period pattern metadata" value={periodPattern} onChange={(event) => setPeriodPattern(event.target.value)} disabled={calendar.status === 'end_dated'} /><Field id="fiscal-calendar-id" label="Record identifier" value={calendar.id} readOnly /><Field id="fiscal-calendar-scope" label="Accounting scope" value={calendar.scopeId} readOnly /><Field id="fiscal-calendar-effective-from" label="Effective from" value={calendar.effectiveFrom} readOnly /><Field id="fiscal-calendar-effective-to" label="Effective to" type="date" value={effectiveTo} onChange={(event) => setEffectiveTo(event.target.value)} disabled={calendar.status === 'end_dated'} /></div>
      <div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={fiscalCalendarStatusState[calendar.status]} label={calendar.status} announce /><span className="text-sm text-base-content/70">Version v{calendar.version} · Approval: {calendar.approvalStatus} · Revision {calendar.revisionNumber} · Next action: {calendar.nextAction}</span></div>
    </Panel>
    <Panel title="Explicit period definitions" description="Every period is entered and validated before persistence. The editor does not generate periods from the pattern metadata.">
      <div className="space-y-4">{periods.map((period, index) => <div key={period.id} className="rounded-box border border-base-300 p-4"><div className="mb-3 flex flex-wrap items-center justify-between gap-2"><h3 className="font-semibold">Period {period.ordinal}</h3><span className="font-mono text-xs text-base-content/65">{period.id}</span></div><div className="grid gap-4 md:grid-cols-4"><Field id={`fiscal-period-${index}-reference`} label="Period reference" value={period.reference} onChange={(event) => updatePeriod(index, 'reference', event.target.value)} disabled={calendar.status === 'end_dated'} /><Field id={`fiscal-period-${index}-ordinal`} label="Ordinal" type="number" value={String(period.ordinal)} onChange={(event) => updatePeriod(index, 'ordinal', event.target.value)} disabled={calendar.status === 'end_dated'} /><Field id={`fiscal-period-${index}-start`} label="Start date" type="date" value={period.startDate} onChange={(event) => updatePeriod(index, 'startDate', event.target.value)} disabled={calendar.status === 'end_dated'} /><Field id={`fiscal-period-${index}-end`} label="End date" type="date" value={period.endDate} onChange={(event) => updatePeriod(index, 'endDate', event.target.value)} disabled={calendar.status === 'end_dated'} /></div></div>)}</div>
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={save} disabled={calendar.status === 'end_dated'}>Save fiscal-calendar revision</Button></div>
    </Panel>
    <Panel title="Dependent impact review" description="The impact port is read-only and currently unavailable because no approved downstream reader is configured."><div className="flex flex-wrap items-center gap-3"><StatusBadge state="disabled" label={`Impact: ${calendar.impactAvailability}`} announce /><span className="text-sm text-base-content/70">No FPM FiscalPeriod, posting gate, close, or GL record is created or changed by this OMD operation.</span></div></Panel>
    <Panel title="History and downstream boundary" description="Established calendar revisions remain available for historical references and effective-dated review."><p className="text-sm text-base-content/75">Current fiscal-calendar revision: v{calendar.version}. Historical period snapshots are retained when the calendar changes; downstream contexts consume approved references through their own integration boundary.</p></Panel>
  </section>
}
