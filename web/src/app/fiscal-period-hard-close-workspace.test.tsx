import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { fpmExtendCloseException, fpmStartHardClose } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { FiscalPeriodHardCloseWorkspace } from './fiscal-period-hard-close-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  fpmExtendCloseException: vi.fn(),
  fpmStartHardClose: vi.fn(),
}))

function renderWithScope(ui: ReactNode) {
  return render(<MemoryRouter><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
}

const establishedResult = {
  status: 'established' as const,
  aggregateId: 'period-1',
  aggregateVersion: 5,
  processId: 'close-run-1',
  correlationId: 'correlation-1',
  links: { self: '/api/v1/fiscal-periods/actions/start-hard-close' },
  data: {
    periodState: 'Closing',
    periodStateVersion: 2,
    periodAggregateVersion: 5,
    softCloseRunId: 'soft-close-run-1',
    softCloseStatus: 'HandoffPending',
    controlEpoch: 1,
    controlEpochStatus: 'HandoffPending',
    closeRunId: 'close-run-1',
    closeRunStatus: 'BarrierAcquired',
    gateMode: 'CloseOnly',
    gateVersion: 2,
    barrierLedgerPosition: 17,
    frozenAdmissionSummary: { admittedPostingCount: 0, controlOwnerType: 'SoftCloseRun' },
    validationOutcome: 'accepted',
    decisionReference: 'decision-1',
    auditReference: 'audit-1',
    replayed: false,
  },
}

describe('fiscal-period hard-close workspace', () => {
  beforeEach(() => {
    vi.mocked(fpmExtendCloseException).mockReset()
    vi.mocked(fpmStartHardClose).mockReset()
  })

  it('submits the scoped soft-close handoff and displays the authoritative barrier', async () => {
    vi.mocked(fpmStartHardClose).mockResolvedValue({ data: establishedResult, error: undefined } as never)
    renderWithScope(<FiscalPeriodHardCloseWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Start hard close and acquire barrier' }))
    await waitFor(() => expect(fpmStartHardClose).toHaveBeenCalledTimes(1))

    const request = vi.mocked(fpmStartHardClose).mock.calls[0][0]
    expect(request.body.data.softCloseRunId).toBe('soft-close-run-1')
    expect(request.body.data.expectedGateVersion).toBe(1)
    expect(request.headers['Idempotency-Key']).toBeTruthy()
    expect(screen.getByRole('status', { name: 'Closing · CloseOnly' })).toBeInTheDocument()
    expect(screen.getByText('17')).toBeInTheDocument()
    expect(screen.getByText(/barrier was acquired/)).toBeInTheDocument()
  })

  it('keeps the period restrictive and explains dependency unavailability', async () => {
    vi.mocked(fpmStartHardClose).mockResolvedValue({ data: undefined, error: { code: 'DEPENDENCY_UNAVAILABLE', detail: 'GL gateway unavailable' } } as never)
    renderWithScope(<FiscalPeriodHardCloseWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Start hard close and acquire barrier' }))
    await waitFor(() => expect(fpmStartHardClose).toHaveBeenCalledTimes(1))

    expect(screen.getByRole('alert')).toHaveTextContent(/remains SoftClosed/)
    expect(screen.getByRole('status', { name: 'SoftClosed · SoftClosePolicy' })).toBeInTheDocument()
    expect(screen.getByText('GL gateway unavailable')).toBeInTheDocument()
  })

  it('submits a separately fingerprinted current exception extension from FPM-SCR-05', async () => {
    const closeResult = {
      ...establishedResult,
      data: {
        ...establishedResult.data,
        closeRunStatus: 'InProgress',
        closeRunAggregateVersion: 2,
        authorityVersion: 1,
        authorityEpoch: 1,
        approvalPolicyVersion: 'close-policy-v1',
        closeRunCommandFingerprint: 'sha256:close-candidate',
        approvalRequestIds: ['approval-1'],
        exceptions: [{ id: 'exception-1', closeRunId: 'close-run-1', status: 'Unresolved', ownerId: 'owner-1', impact: { materiality: 'low' }, dueAt: '2026-10-17T20:00:00Z', blocker: 'Awaiting evidence' }],
      },
    }
    vi.mocked(fpmStartHardClose).mockResolvedValue({ data: closeResult, error: undefined } as never)
    vi.mocked(fpmExtendCloseException).mockResolvedValue({ data: closeResult, error: undefined } as never)
    renderWithScope(<FiscalPeriodHardCloseWorkspace screen="FPM-SCR-05" />)

    fireEvent.click(screen.getByRole('button', { name: 'Start hard close and acquire barrier' }))
    await waitFor(() => expect(fpmStartHardClose).toHaveBeenCalledTimes(1))
    fireEvent.change(screen.getByLabelText(/Decision ID/), { target: { value: 'decision-1' } })
    fireEvent.change(screen.getByLabelText(/Decision reference/), { target: { value: 'decision-reference-1' } })
    fireEvent.change(screen.getByLabelText(/Approver user ID/), { target: { value: 'approver-1' } })
    fireEvent.change(screen.getByLabelText(/Separation-of-duties reference/), { target: { value: 'sod-1' } })
    fireEvent.change(screen.getByLabelText(/Extension evidence reference/), { target: { value: 'incident-42' } })
    fireEvent.change(screen.getByLabelText(/Extension candidate fingerprint/), { target: { value: 'sha256:extension-candidate' } })
    fireEvent.change(screen.getByLabelText(/New approval expiry/), { target: { value: (screen.getByLabelText(/New due time/) as HTMLInputElement).value } })
    expect(screen.getByRole('button', { name: 'Extend current close exception' })).toBeEnabled()
    fireEvent.click(screen.getByRole('button', { name: 'Extend current close exception' }))

    await waitFor(() => expect(fpmExtendCloseException).toHaveBeenCalledTimes(1))
    const request = vi.mocked(fpmExtendCloseException).mock.calls[0][0]
    expect(request.body.data.exceptionId).toBe('exception-1')
    expect(request.body.data.candidateFingerprint).toBe('sha256:extension-candidate')
    expect(request.body.data.evidenceReference).toBe('incident-42')
    expect(request.body.data.newDueAt).toBeTruthy()
  })
})
