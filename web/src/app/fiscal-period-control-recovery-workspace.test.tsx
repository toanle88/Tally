import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { fpmTakeOverPeriodControl } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { FiscalPeriodControlRecoveryWorkspace } from './fiscal-period-control-recovery-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  fpmTakeOverPeriodControl: vi.fn(),
}))

const establishedResult = {
  status: 'established' as const,
  aggregateId: 'period-1',
  aggregateVersion: 6,
  processId: 'soft-close-run-1',
  correlationId: 'correlation-1',
  links: { self: '/api/v1/fiscal-periods/actions/take-over-period-control' },
  data: {
    periodState: 'SoftClosed',
    periodAggregateVersion: 6,
    processId: 'soft-close-run-1',
    ownerId: 'recovery-operator-1',
    authorityEpoch: 5,
    authorityVersion: 3,
    authorityExpiresAt: '2026-10-17T18:00:00Z',
    controlCutoffAt: '2026-10-17T12:00:00Z',
    controlAuthority: { authorityEpoch: 5, version: 3 },
    ownershipHistory: [{ priorOwner: 'owner-1', newOwner: 'recovery-operator-1' }],
    authoritativeGateStatus: { gateMode: 'SoftClosePolicy', gateVersion: 3 },
    gateMode: 'SoftClosePolicy',
    gateVersion: 3,
    pendingReconciliation: false,
    nextAction: 'resume-soft-close',
    replayStatus: 'established',
    replayed: false,
  },
}

function renderWithScope() {
  return render(<MemoryRouter><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}><FiscalPeriodControlRecoveryWorkspace /></ScopeProvider></MemoryRouter>)
}

describe('fiscal-period control recovery workspace', () => {
  beforeEach(() => vi.mocked(fpmTakeOverPeriodControl).mockReset())

  it('submits dedicated approval evidence and shows owner lineage and gate status', async () => {
    vi.mocked(fpmTakeOverPeriodControl).mockResolvedValue({ data: establishedResult, error: undefined } as never)
    renderWithScope()

    fireEvent.change(screen.getByLabelText(/Evidence reference/), { target: { value: 'incident-42' } })
    fireEvent.change(screen.getByLabelText(/Approval request ID/), { target: { value: 'approval-1' } })
    fireEvent.change(screen.getByLabelText(/Decision ID/), { target: { value: 'decision-1' } })
    fireEvent.change(screen.getByLabelText(/Candidate fingerprint/), { target: { value: 'sha256:recovery-candidate' } })
    fireEvent.change(screen.getByLabelText(/Decision reference/), { target: { value: 'decision-reference-1' } })
    fireEvent.change(screen.getByLabelText(/Approver user ID/), { target: { value: 'approver-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Take over period control' }))

    await waitFor(() => expect(fpmTakeOverPeriodControl).toHaveBeenCalledTimes(1))
    const request = vi.mocked(fpmTakeOverPeriodControl).mock.calls[0][0]
    expect(request.body.data.target).toBe('softClose')
    expect(request.body.data.expectedControlAuthorityEpoch).toBe(1)
    expect(request.body.data.evidenceReference).toBe('incident-42')
    expect(request.headers['Idempotency-Key']).toBeTruthy()
    expect(screen.getByRole('status', { name: /SoftClosed · established/ })).toBeInTheDocument()
    expect(screen.getByText('recovery-operator-1')).toBeInTheDocument()
    expect(screen.getByText(/SoftClosePolicy \/ 3/)).toBeInTheDocument()
    expect(screen.getByText(/Ownership history/)).toBeInTheDocument()
  })

  it('keeps the period restrictive when gate reconciliation is unavailable', async () => {
    vi.mocked(fpmTakeOverPeriodControl).mockResolvedValue({ data: undefined, error: { code: 'DEPENDENCY_UNAVAILABLE', detail: 'GL status unavailable' } } as never)
    renderWithScope()

    fireEvent.change(screen.getByLabelText(/Evidence reference/), { target: { value: 'incident-42' } })
    fireEvent.change(screen.getByLabelText(/Approval request ID/), { target: { value: 'approval-1' } })
    fireEvent.change(screen.getByLabelText(/Decision ID/), { target: { value: 'decision-1' } })
    fireEvent.change(screen.getByLabelText(/Candidate fingerprint/), { target: { value: 'sha256:recovery-candidate' } })
    fireEvent.change(screen.getByLabelText(/Decision reference/), { target: { value: 'decision-reference-1' } })
    fireEvent.change(screen.getByLabelText(/Approver user ID/), { target: { value: 'approver-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Take over period control' }))

    await waitFor(() => expect(fpmTakeOverPeriodControl).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('alert')).toHaveTextContent(/remains restrictive/)
    expect(screen.getByText('Pending — restrictive')).toBeInTheDocument()
    expect(screen.getByText('GL status unavailable')).toBeInTheDocument()
  })
})
