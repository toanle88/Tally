import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { glAcquirePostingBarrier, glCloseScopedReopenGate, glEnterSoftCloseGate, glExitSoftCloseGate, glFinalizePostingGate, glOpenScopedReopenGate, glReleasePostingBarrier } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { GlPostingGateWorkspace } from './gl-posting-gate-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  glAcquirePostingBarrier: vi.fn(),
  glCloseScopedReopenGate: vi.fn(),
  glEnterSoftCloseGate: vi.fn(),
  glExitSoftCloseGate: vi.fn(),
  glFinalizePostingGate: vi.fn(),
  glOpenScopedReopenGate: vi.fn(),
  glReleasePostingBarrier: vi.fn(),
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

const acquiredResult = {
  ...enteredResult,
  aggregateVersion: 4,
  correlationId: 'correlation-acquire',
  links: { self: '/api/v1/general-ledger/actions/acquire-posting-barrier' },
  data: {
    ...enteredResult.data,
    gateMode: 'CloseOnly',
    gateVersion: 4,
    ownerType: 'CloseRun',
    ownerId: 'close-run-1',
    closeType: 'InitialHardClose',
    barrierLedgerPosition: 10,
    priorOwnerId: 'run-1',
    activeAdmissionCounters: { controlOwnerType: 'CloseRun', admittedPostingCount: 0 },
    nextAction: 'Finalize or release the posting barrier under the authoritative rules.',
  },
}

const finalizedResult = {
  ...acquiredResult,
  aggregateVersion: 5,
  correlationId: 'correlation-finalize',
  links: { self: '/api/v1/general-ledger/actions/finalize-posting-gate' },
  data: {
    ...acquiredResult.data,
    gateMode: 'HardClosed',
    gateVersion: 5,
    ownerId: null,
    finalLedgerWatermark: 12,
    finalizedCloseRunId: 'close-run-1',
    nextAction: 'Use the final watermark for downstream close and sealing work.',
  },
}

const scopedReopenedResult = {
  ...finalizedResult,
  aggregateVersion: 6,
  correlationId: 'correlation-reopen-open',
  links: { self: '/api/v1/general-ledger/actions/open-scoped-reopen-gate' },
  data: {
    ...finalizedResult.data,
    gateMode: 'ScopedReopen',
    gateVersion: 6,
    ownerType: 'ReopenRequest',
    ownerId: 'reopen-request-1',
    reopenRequestId: 'reopen-request-1',
    authorizationScope: { accountIds: ['account-1'], transactionClasses: ['manual-journal'] },
    authorizationExpiry: '2026-12-31T23:59:00Z',
    controlAuthorityEpoch: 3,
    activeAdmissionCounters: { controlOwnerType: 'ReopenRequest', admittedPostingCount: 0 },
    nextAction: 'Close the scoped reopen gate when corrections are complete.',
  },
}

const scopedReopenClosedResult = {
  ...scopedReopenedResult,
  aggregateVersion: 7,
  correlationId: 'correlation-reopen-close',
  links: { self: '/api/v1/general-ledger/actions/close-scoped-reopen-gate' },
  data: {
    ...scopedReopenedResult.data,
    gateMode: 'CloseOnly',
    gateVersion: 7,
    lastFrozenAdmissionSummary: { controlOwnerType: 'ReopenRequest', admittedPostingCount: 1 },
    nextAction: 'Mandatory reclose handoff is required.',
  },
}

describe('GL posting-gate monitor', () => {
  beforeEach(() => {
    vi.mocked(glEnterSoftCloseGate).mockReset()
    vi.mocked(glExitSoftCloseGate).mockReset()
    vi.mocked(glAcquirePostingBarrier).mockReset()
    vi.mocked(glFinalizePostingGate).mockReset()
    vi.mocked(glReleasePostingBarrier).mockReset()
    vi.mocked(glOpenScopedReopenGate).mockReset()
    vi.mocked(glCloseScopedReopenGate).mockReset()
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
    const request = vi.mocked(glExitSoftCloseGate).mock.calls[0][0]
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

  it('acquires and finalizes the barrier from authoritative command results', async () => {
    vi.mocked(glEnterSoftCloseGate).mockResolvedValue({ data: enteredResult, error: undefined } as never)
    vi.mocked(glAcquirePostingBarrier).mockResolvedValue({ data: acquiredResult, error: undefined } as never)
    vi.mocked(glFinalizePostingGate).mockResolvedValue({ data: finalizedResult, error: undefined } as never)
    renderWithScope(<GlPostingGateWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Enter soft-close gate' }))
    await waitFor(() => expect(glEnterSoftCloseGate).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('button', { name: 'Acquire posting barrier' }))
    await waitFor(() => expect(glAcquirePostingBarrier).toHaveBeenCalledTimes(1))
    const acquire = vi.mocked(glAcquirePostingBarrier).mock.calls[0][0]
    expect(acquire.body.expectedVersion).toBe(2)
    expect(acquire.body.data.closeRunId).toBe('close-run-1')
    expect(acquire.body.data.closeType).toBe('InitialHardClose')
    expect(screen.getByRole('status', { name: 'CloseOnly' })).toBeInTheDocument()
    const finalWatermark = screen.getByDisplayValue('0')
    expect(finalWatermark).not.toBeDisabled()
    fireEvent.change(finalWatermark, { target: { value: '12' } })
    fireEvent.click(screen.getByRole('button', { name: 'Finalize posting gate' }))
    await waitFor(() => expect(glFinalizePostingGate).toHaveBeenCalledTimes(1))
    const finalize = vi.mocked(glFinalizePostingGate).mock.calls[0][0]
    expect(finalize.body.expectedVersion).toBe(4)
    expect(finalize.body.data.finalLedgerWatermark).toBe(12)
    expect(screen.getByRole('status', { name: 'HardClosed' })).toBeInTheDocument()
    expect(screen.getAllByText(/final watermark/).length).toBeGreaterThan(0)
  })

  it('opens and closes a scoped reopen gate and shows the mandatory reclose handoff', async () => {
    vi.mocked(glEnterSoftCloseGate).mockResolvedValue({ data: enteredResult, error: undefined } as never)
    vi.mocked(glAcquirePostingBarrier).mockResolvedValue({ data: acquiredResult, error: undefined } as never)
    vi.mocked(glFinalizePostingGate).mockResolvedValue({ data: finalizedResult, error: undefined } as never)
    vi.mocked(glOpenScopedReopenGate).mockResolvedValue({ data: scopedReopenedResult, error: undefined } as never)
    vi.mocked(glCloseScopedReopenGate).mockResolvedValue({ data: scopedReopenClosedResult, error: undefined } as never)
    renderWithScope(<GlPostingGateWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Enter soft-close gate' }))
    await waitFor(() => expect(glEnterSoftCloseGate).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('button', { name: 'Acquire posting barrier' }))
    await waitFor(() => expect(glAcquirePostingBarrier).toHaveBeenCalledTimes(1))
    fireEvent.change(screen.getByDisplayValue('0'), { target: { value: '12' } })
    fireEvent.click(screen.getByRole('button', { name: 'Finalize posting gate' }))
    await waitFor(() => expect(glFinalizePostingGate).toHaveBeenCalledTimes(1))

    fireEvent.click(screen.getByRole('button', { name: 'Open scoped reopen gate' }))
    await waitFor(() => expect(glOpenScopedReopenGate).toHaveBeenCalledTimes(1))
    const openRequest = vi.mocked(glOpenScopedReopenGate).mock.calls[0][0]
    expect(openRequest.body.data.reopenRequestId).toBe('reopen-request-1')
    expect(openRequest.body.data.authorizationScope).toEqual({ accountIds: [], transactionClasses: ['manual-journal'] })
    expect(screen.getByRole('status', { name: 'ScopedReopen' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Close scoped reopen gate' }))
    await waitFor(() => expect(glCloseScopedReopenGate).toHaveBeenCalledTimes(1))
    const closeRequest = vi.mocked(glCloseScopedReopenGate).mock.calls[0][0]
    expect(closeRequest.body.expectedVersion).toBe(6)
    expect(closeRequest.body.data.reopenRequestId).toBe('reopen-request-1')
    expect(screen.getByText('Corrections were admitted. Mandatory reclose handoff is required before finalization.', { selector: 'p' })).toBeInTheDocument()
  })
})
