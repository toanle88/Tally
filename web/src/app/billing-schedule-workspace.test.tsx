import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { invConfigureBillingSchedules, invGetBillingSchedule, invListBillingSchedules } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { BillingScheduleEditor, BillingScheduleWorklist } from './billing-schedule-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  invConfigureBillingSchedules: vi.fn(),
  invGetBillingSchedule: vi.fn(),
  invListBillingSchedules: vi.fn(),
}))

const schedule = {
  id: '11111111-1111-4111-8111-111111111111',
  accountingScopeId: 'scope-vietnam-statutory',
  customerId: 'customer-001',
  contractReference: 'CONTRACT-001',
  charges: [{ id: '22222222-2222-4222-8222-222222222222', templateId: '33333333-3333-4333-8333-333333333333', templateVersion: 2, templateLineItemId: '44444444-4444-4444-8444-444444444444', quantity: '1', amount: '12.5', currency: 'USD' }],
  nextBillingDate: '2026-04-01',
  timeZone: 'Asia/Ho_Chi_Minh',
  status: 'configured',
  effectiveFrom: '2026-01-01',
  effectiveTo: '2026-12-31',
  version: 2,
  revisionNumber: 2,
  validationOutcome: 'valid',
  revisions: [{ revisionNumber: 1, capturedAt: '2026-01-01T00:00:00Z', fingerprint: 'sha256:one', version: 1 }, { revisionNumber: 2, capturedAt: '2026-02-01T00:00:00Z', fingerprint: 'sha256:two', version: 2 }],
}

function renderWithScope(ui: ReactNode, initialEntry: string) {
  return render(<MemoryRouter initialEntries={[initialEntry]}><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
}

describe('billing-schedule workspace', () => {
  beforeEach(() => {
    vi.mocked(invConfigureBillingSchedules).mockReset()
    vi.mocked(invGetBillingSchedule).mockReset()
    vi.mocked(invListBillingSchedules).mockReset()
  })

  it('loads the authoritative worklist and links to the schedule editor', async () => {
    vi.mocked(invListBillingSchedules).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: schedule.version, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: '/api/v1/invoicing/reference/billing-schedules' }, data: { billingSchedules: [schedule], page: { size: 50, hasMore: false } } }, error: undefined } as never)
    renderWithScope(<BillingScheduleWorklist />, '/invoicing/inv-ws-01')

    expect(await screen.findByRole('link', { name: 'customer-001' })).toHaveAttribute('href', expect.stringContaining(`billingScheduleId=${schedule.id}`))
    expect(screen.getByText('v2 / r2')).toBeInTheDocument()
    expect(invListBillingSchedules).toHaveBeenCalledWith(expect.objectContaining({ headers: { 'X-Accounting-Scope-Id': 'scope-vietnam-statutory' } }))
  })

  it('does not turn a malformed authoritative record into an empty worklist', async () => {
    vi.mocked(invListBillingSchedules).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: 0, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: '/api/v1/invoicing/reference/billing-schedules' }, data: { billingSchedules: [{ ...schedule, charges: [{ ...schedule.charges[0], amount: 12.5 }] }], page: { size: 50, hasMore: false } } }, error: undefined } as never)
    renderWithScope(<BillingScheduleWorklist />, '/invoicing/inv-ws-01')

    expect(await screen.findByRole('status')).toHaveTextContent('unusable')
    expect(screen.queryByText('No authoritative billing schedules are configured for this accounting scope.')).not.toBeInTheDocument()
  })

  it('validates required fields and submits canonical exact-decimal values', async () => {
    vi.mocked(invConfigureBillingSchedules).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: 1, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: `/api/v1/invoicing/reference/billing-schedules/${schedule.id}` }, data: { billingSchedule: { ...schedule, version: 1, revisionNumber: 1, charges: [{ ...schedule.charges[0], amount: '10', quantity: '2' }] }, replayed: false } }, error: undefined } as never)
    renderWithScope(<BillingScheduleEditor />, '/invoicing/inv-scr-02?new=true')

    fireEvent.click(screen.getByRole('button', { name: 'Create billing schedule' }))
    expect(await screen.findByRole('alert')).toHaveTextContent('Customer identity')
    expect(invConfigureBillingSchedules).not.toHaveBeenCalled()

    fireEvent.change(document.getElementById('billing-schedule-customer')!, { target: { value: 'customer-001' } })
    fireEvent.change(document.getElementById('billing-schedule-contract')!, { target: { value: ' CONTRACT-001 ' } })
    fireEvent.change(document.getElementById('billing-schedule-effective-from')!, { target: { value: '2026-01-01' } })
    fireEvent.change(document.getElementById('billing-schedule-next-date')!, { target: { value: '2026-04-01' } })
    fireEvent.change(document.getElementById('billing-schedule-template')!, { target: { value: '33333333-3333-4333-8333-333333333333' } })
    fireEvent.change(document.getElementById('billing-schedule-template-line')!, { target: { value: '44444444-4444-4444-8444-444444444444' } })
    fireEvent.change(document.getElementById('billing-schedule-amount')!, { target: { value: '00010.00' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create billing schedule' }))

    await waitFor(() => expect(invConfigureBillingSchedules).toHaveBeenCalledTimes(1))
    const request = vi.mocked(invConfigureBillingSchedules).mock.calls[0][0]
    expect(request.body.data.contractReference).toBe('CONTRACT-001')
    expect(request.body.data.charges[0]).toMatchObject({ amount: '10', quantity: '1', currency: 'USD' })
    expect(request.headers['Idempotency-Key']).toBeTruthy()
  })

  it('retains every authoritative scheduled charge when maintaining a schedule', async () => {
    const secondCharge = { id: '77777777-7777-4777-8777-777777777777', templateId: '88888888-8888-4888-8888-888888888888', templateVersion: 4, templateLineItemId: '99999999-9999-4999-8999-999999999999', quantity: '3', amount: '25', currency: 'EUR' }
    const multiChargeSchedule = { ...schedule, charges: [...schedule.charges, secondCharge] }
    vi.mocked(invGetBillingSchedule).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: schedule.version, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: `/api/v1/invoicing/reference/billing-schedules/${schedule.id}` }, data: { billingSchedule: multiChargeSchedule } }, error: undefined } as never)
    vi.mocked(invConfigureBillingSchedules).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: 3, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: `/api/v1/invoicing/reference/billing-schedules/${schedule.id}` }, data: { billingSchedule: { ...multiChargeSchedule, version: 3, revisionNumber: 3 }, replayed: false } }, error: undefined } as never)
    renderWithScope(<BillingScheduleEditor />, `/invoicing/inv-scr-02?billingScheduleId=${schedule.id}`)

    expect(await screen.findByDisplayValue('CONTRACT-001')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save billing-schedule revision' }))

    await waitFor(() => expect(invConfigureBillingSchedules).toHaveBeenCalledTimes(1))
    const request = vi.mocked(invConfigureBillingSchedules).mock.calls[0][0]
    expect(request.body.data.charges).toHaveLength(2)
    expect(request.body.data.charges.map((charge) => charge.id)).toEqual([schedule.charges[0].id, secondCharge.id])
  })

  it('shows a version conflict while preserving the reviewed schedule form', async () => {
    vi.mocked(invGetBillingSchedule).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: schedule.version, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: `/api/v1/invoicing/reference/billing-schedules/${schedule.id}` }, data: { billingSchedule: schedule } }, error: undefined } as never)
    vi.mocked(invConfigureBillingSchedules).mockResolvedValue({ data: undefined, error: { status: 409, code: 'VERSION_CONFLICT', detail: 'The billing schedule changed.', currentVersion: 3 } } as never)
    renderWithScope(<BillingScheduleEditor />, `/invoicing/inv-scr-02?billingScheduleId=${schedule.id}`)

    expect(await screen.findByDisplayValue('CONTRACT-001')).toBeInTheDocument()
    fireEvent.change(screen.getByDisplayValue('CONTRACT-001'), { target: { value: 'Reviewed replacement' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save billing-schedule revision' }))

    expect(await screen.findByRole('dialog')).toHaveTextContent('record changed before this action')
    expect(screen.getByDisplayValue('Reviewed replacement')).toBeInTheDocument()
    expect(screen.getByText(/Refresh the authoritative schedule/)).toBeInTheDocument()
  })

  it('starts a fresh idempotency identity after refreshing a version conflict', async () => {
    vi.mocked(invGetBillingSchedule).mockResolvedValue({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: schedule.version, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: `/api/v1/invoicing/reference/billing-schedules/${schedule.id}` }, data: { billingSchedule: schedule } }, error: undefined } as never)
    vi.mocked(invConfigureBillingSchedules)
      .mockResolvedValueOnce({ data: undefined, error: { status: 409, code: 'VERSION_CONFLICT', detail: 'The billing schedule changed.', currentVersion: 3 } } as never)
      .mockResolvedValueOnce({ data: { status: 'established', aggregateId: schedule.id, aggregateVersion: 3, correlationId: '55555555-5555-4555-8555-555555555555', links: { self: `/api/v1/invoicing/reference/billing-schedules/${schedule.id}` }, data: { billingSchedule: { ...schedule, version: 3, revisionNumber: 3 } }, replayed: false }, error: undefined } as never)
    renderWithScope(<BillingScheduleEditor />, `/invoicing/inv-scr-02?billingScheduleId=${schedule.id}`)

    expect(await screen.findByDisplayValue('CONTRACT-001')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save billing-schedule revision' }))
    expect(await screen.findByRole('dialog')).toBeInTheDocument()
    const firstKey = vi.mocked(invConfigureBillingSchedules).mock.calls[0][0].headers['Idempotency-Key']

    fireEvent.click(screen.getByRole('button', { name: 'Refresh authoritative state' }))
    await waitFor(() => expect(invGetBillingSchedule).toHaveBeenCalledTimes(2))
    fireEvent.click(screen.getByRole('button', { name: 'Save billing-schedule revision' }))
    await waitFor(() => expect(invConfigureBillingSchedules).toHaveBeenCalledTimes(2))
    const secondKey = vi.mocked(invConfigureBillingSchedules).mock.calls[1][0].headers['Idempotency-Key']
    expect(secondKey).toBeTruthy()
    expect(secondKey).not.toBe(firstKey)
  })
})
