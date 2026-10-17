import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { fpmApplyReopenApprovalDecision, fpmRequestReopen, fpmStartReclose } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { FiscalPeriodReopenWorkspace } from './fiscal-period-reopen-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  fpmApplyReopenApprovalDecision: vi.fn(),
  fpmRequestReopen: vi.fn(),
  fpmStartReclose: vi.fn(),
}))

function renderWithScope() {
  return render(<MemoryRouter><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}><FiscalPeriodReopenWorkspace /></ScopeProvider></MemoryRouter>)
}

const pendingResult = {
  status: 'established' as const,
  aggregateId: 'period-1',
  aggregateVersion: 2,
  processId: 'reopen-1',
  correlationId: 'correlation-1',
  links: { self: '/api/v1/fiscal-periods/actions/request-reopen' },
  data: {
    periodState: 'HardClosed',
    periodAggregateVersion: 2,
    reopenRequestId: 'reopen-1',
    reopenStatus: 'PendingApproval',
    reopenMode: 'ScopedCorrection',
    gateMode: 'HardClosed',
    gateVersion: 4,
    authorizationExpiry: '2026-10-17T18:00:00Z',
    approvalRequestId: 'approval-1',
    reopenRequest: { requestFingerprint: 'sha256:reopen-candidate', aggregateVersion: 1 },
    reopenScope: { transactionClasses: ['ReopenCorrection'] },
  },
}

const activeResult = {
  ...pendingResult,
  aggregateVersion: 3,
  links: { self: '/api/v1/fiscal-periods/actions/apply-reopen-approval-decision' },
  data: {
    ...pendingResult.data,
    periodState: 'Reopening',
    periodAggregateVersion: 3,
    reopenStatus: 'Active',
    gateMode: 'ScopedReopen',
    gateVersion: 5,
    reopenRequest: { requestFingerprint: 'sha256:reopen-candidate', aggregateVersion: 3 },
  },
}

const completedNoChangeResult = {
  ...activeResult,
  aggregateVersion: 4,
  links: { self: '/api/v1/fiscal-periods/actions/start-reclose' },
  data: {
    ...activeResult.data,
    periodState: 'HardClosed',
    periodAggregateVersion: 4,
    reopenStatus: 'CompletedNoChange',
    gateMode: 'HardClosed',
    gateVersion: 6,
    reopenRequest: { requestFingerprint: 'sha256:reopen-candidate', aggregateVersion: 4 },
  },
}

describe('fiscal-period reopen workspace', () => {
  beforeEach(() => {
    vi.mocked(fpmRequestReopen).mockReset()
    vi.mocked(fpmApplyReopenApprovalDecision).mockReset()
    vi.mocked(fpmStartReclose).mockReset()
  })

  it('keeps a requested reopen pending and restrictive until approval', async () => {
    vi.mocked(fpmRequestReopen).mockResolvedValue({ data: pendingResult, error: undefined } as never)
    renderWithScope()

    fireEvent.click(screen.getByRole('button', { name: 'Request reopen approval' }))
    await waitFor(() => expect(fpmRequestReopen).toHaveBeenCalledTimes(1))

    expect(screen.getByRole('status', { name: /HardClosed · PendingApproval · HardClosed/ })).toBeInTheDocument()
    expect(screen.getByText(/never opens a gate/)).toBeInTheDocument()
    expect(screen.getByText(/recorded as PendingApproval/)).toBeInTheDocument()
  })

  it('applies current approval evidence and shows the bounded active gate', async () => {
    vi.mocked(fpmRequestReopen).mockResolvedValue({ data: pendingResult, error: undefined } as never)
    vi.mocked(fpmApplyReopenApprovalDecision).mockResolvedValue({ data: activeResult, error: undefined } as never)
    renderWithScope()

    fireEvent.click(screen.getByRole('button', { name: 'Request reopen approval' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Apply approval decision' })).toBeInTheDocument())
    fireEvent.change(screen.getByLabelText(/Decision ID/), { target: { value: 'decision-1' } })
    fireEvent.change(screen.getByLabelText(/Decision reference/), { target: { value: 'decision-reference-1' } })
    fireEvent.change(screen.getByLabelText(/Approver user ID/), { target: { value: 'approver-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply approval decision' }))
    await waitFor(() => expect(fpmApplyReopenApprovalDecision).toHaveBeenCalledTimes(1))

    expect(screen.getByRole('status', { name: /Reopening · Active · ScopedReopen/ })).toBeInTheDocument()
    expect(screen.getByText(/Only the recorded scope may be admitted/)).toBeInTheDocument()
    expect(vi.mocked(fpmApplyReopenApprovalDecision).mock.calls[0][0].body.expectedVersion).toBe(2)
    expect(vi.mocked(fpmApplyReopenApprovalDecision).mock.calls[0][0].body.data.expectedReopenRequestVersion).toBe(1)
  })

  it('lets GL determine no-change versus mandatory reclose without a user choice', async () => {
    vi.mocked(fpmRequestReopen).mockResolvedValue({ data: pendingResult, error: undefined } as never)
    vi.mocked(fpmApplyReopenApprovalDecision).mockResolvedValue({ data: activeResult, error: undefined } as never)
    vi.mocked(fpmStartReclose).mockResolvedValue({ data: completedNoChangeResult, error: undefined } as never)
    renderWithScope()

    fireEvent.click(screen.getByRole('button', { name: 'Request reopen approval' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Apply approval decision' })).toBeInTheDocument())
    fireEvent.change(screen.getByLabelText(/Decision ID/), { target: { value: 'decision-1' } })
    fireEvent.change(screen.getByLabelText(/Decision reference/), { target: { value: 'decision-reference-1' } })
    fireEvent.change(screen.getByLabelText(/Approver user ID/), { target: { value: 'approver-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply approval decision' }))
    await waitFor(() => expect(screen.getByRole('button', { name: 'Close reopen and determine outcome' })).toBeInTheDocument())

    expect(screen.getByText(/cannot choose no-change versus reclose/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Close reopen and determine outcome' }))
    await waitFor(() => expect(fpmStartReclose).toHaveBeenCalledTimes(1))

    const request = vi.mocked(fpmStartReclose).mock.calls[0][0]
    expect(request.body.expectedVersion).toBe(3)
    expect(request.body.data.expectedReopenRequestVersion).toBe(3)
    expect(request.body.data.expectedGateVersion).toBe(5)
    expect(screen.getByRole('status', { name: /HardClosed · CompletedNoChange · HardClosed/ })).toBeInTheDocument()
    expect(screen.getByText(/zero admitted reopen postings/)).toBeInTheDocument()
  })
})
