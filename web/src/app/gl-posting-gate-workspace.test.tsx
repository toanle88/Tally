import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { glEnterSoftCloseGate, glExitSoftCloseGate } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { GlPostingGateWorkspace } from './gl-posting-gate-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  glEnterSoftCloseGate: vi.fn(),
  glExitSoftCloseGate: vi.fn(),
}))

function renderWithScope(ui: ReactNode) {
  return render(<MemoryRouter><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
}

const enteredResult = {
  status: 'established' as const,
  aggregateId: 'gate-1',
  aggregateVersion: 2,
  correlationId: 'correlation-enter',
  links: { self: '/api/v1/general-ledger/actions/enter-soft-close-gate' },
  data: {
    gateMode: 'SoftClosePolicy',
    gateVersion: 2,
    controlEpoch: 1,
    ownerId: 'run-1',
    policyVersion: 'soft-close-policy-v1',
    restrictions: { ordinaryPosting: 'evaluator-controlled' },
    activeAdmissionCounters: { admittedPostingCount: 0 },
    lastFrozenAdmissionSummary: null,
    auditReference: 'audit-1',
    nextAction: 'Exit with the matching run and control epoch.',
    replayed: false,
  },
}

const exitedResult = {
  ...enteredResult,
  aggregateVersion: 3,
  correlationId: 'correlation-exit',
  links: { self: '/api/v1/general-ledger/actions/exit-soft-close-gate' },
  data: {
    ...enteredResult.data,
    gateMode: 'Open',
    gateVersion: 3,
    ownerId: null,
    controlEpoch: 0,
    lastFrozenAdmissionSummary: { frozenAtGateVersion: 3, admittedPostingCount: 0 },
    nextAction: 'Review the frozen soft-close admission summary.',
  },
}

describe('GL posting-gate monitor', () => {
  beforeEach(() => {
    vi.mocked(glEnterSoftCloseGate).mockReset()
    vi.mocked(glExitSoftCloseGate).mockReset()
  })

  it('renders only the latest action result and uses its version for exit', async () => {
    vi.mocked(glEnterSoftCloseGate).mockResolvedValue({ data: enteredResult, error: undefined } as never)
    vi.mocked(glExitSoftCloseGate).mockResolvedValue({ data: exitedResult, error: undefined } as never)
    renderWithScope(<GlPostingGateWorkspace />)

    expect(screen.getByText(/No authoritative gate result/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Enter soft-close gate' }))
    await waitFor(() => expect(glEnterSoftCloseGate).toHaveBeenCalledTimes(1))
    expect(screen.getByRole('status', { name: 'SoftClosePolicy' })).toBeInTheDocument()
    expect(screen.getByText(/Soft-close entry established/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Exit soft-close gate' }))
    await waitFor(() => expect(glExitSoftCloseGate).toHaveBeenCalledTimes(1))
    const request = vi.mocked(glExitSoftCloseGate).mock.calls[0][0] as { body: { expectedVersion: number; data: { softCloseRunId: string; controlEpoch: number } }; headers: { 'Idempotency-Key': string } }
    expect(request.body.expectedVersion).toBe(2)
    expect(request.body.data.softCloseRunId).toBe('soft-close-run-1')
    expect(request.body.data.controlEpoch).toBe(1)
    expect(request.headers['Idempotency-Key']).toBeTruthy()
    expect(screen.getByRole('status', { name: 'Open' })).toBeInTheDocument()
    expect(screen.getByText(/frozen soft-close admission summary/)).toBeInTheDocument()
  })

  it('reuses command identity after an ambiguous retry', async () => {
    vi.mocked(glEnterSoftCloseGate)
      .mockResolvedValueOnce({ data: undefined, error: { detail: 'temporary gateway failure' } } as never)
      .mockResolvedValueOnce({ data: enteredResult, error: undefined } as never)
    renderWithScope(<GlPostingGateWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Enter soft-close gate' }))
    await waitFor(() => expect(glEnterSoftCloseGate).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('button', { name: 'Enter soft-close gate' }))
    await waitFor(() => expect(glEnterSoftCloseGate).toHaveBeenCalledTimes(2))

    const first = vi.mocked(glEnterSoftCloseGate).mock.calls[0][0] as { body: { commandId: string }; headers: { 'Idempotency-Key': string; 'X-Correlation-Id': string } }
    const second = vi.mocked(glEnterSoftCloseGate).mock.calls[1][0] as { body: { commandId: string }; headers: { 'Idempotency-Key': string; 'X-Correlation-Id': string } }
    expect(second.body.commandId).toBe(first.body.commandId)
    expect(second.headers['Idempotency-Key']).toBe(first.headers['Idempotency-Key'])
    expect(second.headers['X-Correlation-Id']).toBe(first.headers['X-Correlation-Id'])
  })
})
