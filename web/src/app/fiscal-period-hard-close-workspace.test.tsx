import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { fpmStartHardClose } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { FiscalPeriodHardCloseWorkspace } from './fiscal-period-hard-close-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
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
  beforeEach(() => vi.mocked(fpmStartHardClose).mockReset())

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
})
