import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { glSubmitPostingRequest } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { GlPostingWorkspace } from './gl-posting-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  glSubmitPostingRequest: vi.fn(),
}))

function renderWithScope(ui: ReactNode) {
  return render(<MemoryRouter><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
}

const establishedResult = {
  status: 'established',
  aggregateId: 'journal-request-1',
  aggregateVersion: 1,
  correlationId: 'correlation-1',
  links: { self: '/api/v1/general-ledger/actions/submit-posting-request' },
  data: {
    outcome: 'JournalEntryPosted',
    lifecycleStatus: 'Posted',
    journalId: 'journal-1',
    journalNumber: 'JE-2026-000001',
    journalVersion: 1,
    ledgerPosition: 42,
    validationOutcome: 'valid',
    approvalStatus: 'not-required',
    sourceReference: 'posting-request:request-1:v1',
    gateEvidence: { fiscalPeriodId: 'period-2026-08', periodStateVersion: 1, postingGateVersion: 1, gateMode: 'Open' },
    auditReference: 'audit-1',
    replayed: false,
  },
}

describe('GL posting workbench', () => {
  beforeEach(() => {
    vi.mocked(glSubmitPostingRequest).mockReset()
  })

  it('blocks unbalanced exact-decimal lines before sending a command', () => {
    renderWithScope(<GlPostingWorkspace view="request" />)
    fireEvent.change(screen.getByRole('textbox', { name: /Functional amount · line 2/ }), { target: { value: '99.00' } })
    fireEvent.click(screen.getByRole('button', { name: 'Submit posting request' }))

    expect(screen.getByRole('alert')).toHaveTextContent('Debit and credit totals must balance')
    expect(glSubmitPostingRequest).not.toHaveBeenCalled()
  })

  it('submits the typed version-2 command and renders the established result', async () => {
    vi.mocked(glSubmitPostingRequest).mockResolvedValue({ data: establishedResult, error: undefined } as never)
    renderWithScope(<GlPostingWorkspace view="request" />)
    fireEvent.click(screen.getByRole('button', { name: 'Submit posting request' }))

    await waitFor(() => expect(glSubmitPostingRequest).toHaveBeenCalledTimes(1))
    const request = vi.mocked(glSubmitPostingRequest).mock.calls[0][0] as { body: { data: { contractVersion: number; lines: unknown[] }; accountingScopeId: string }; headers: { 'Idempotency-Key': string } }
    expect(request.body.data.contractVersion).toBe(2)
    expect(request.body.data.lines).toHaveLength(2)
    expect(request.body.accountingScopeId).toBe('scope-vietnam-statutory')
    expect(request.headers['Idempotency-Key']).toBeTruthy()
    expect(screen.getByText('JE-2026-000001')).toBeInTheDocument()
    expect(screen.getByText('JE-2026-000001')).toBeInTheDocument()
  })

  it('reuses command identity and conversion evidence after an ambiguous retry', async () => {
    vi.mocked(glSubmitPostingRequest)
      .mockResolvedValueOnce({ data: undefined, error: { detail: 'temporary gateway failure' } } as never)
      .mockResolvedValueOnce({ data: establishedResult, error: undefined } as never)
    renderWithScope(<GlPostingWorkspace view="request" />)
    fireEvent.change(screen.getByRole('combobox', { name: 'Transaction currency' }), { target: { value: 'EUR' } })

    fireEvent.click(screen.getByRole('button', { name: 'Submit posting request' }))
    await waitFor(() => expect(glSubmitPostingRequest).toHaveBeenCalledTimes(1))
    fireEvent.click(screen.getByRole('button', { name: 'Submit posting request' }))
    await waitFor(() => expect(glSubmitPostingRequest).toHaveBeenCalledTimes(2))

    type SubmissionRequest = {
      body: { commandId: string; data: { requestId: string; sourceAggregateId: string; conversionEvidence?: { rateSetId: string } } }
      headers: { 'Idempotency-Key': string; 'X-Correlation-Id': string }
    }
    const first = vi.mocked(glSubmitPostingRequest).mock.calls[0][0] as SubmissionRequest
    const second = vi.mocked(glSubmitPostingRequest).mock.calls[1][0] as SubmissionRequest
    expect(second.body.commandId).toBe(first.body.commandId)
    expect(second.body.data.requestId).toBe(first.body.data.requestId)
    expect(second.body.data.sourceAggregateId).toBe(first.body.data.sourceAggregateId)
    expect(second.body.data.conversionEvidence?.rateSetId).toBe(first.body.data.conversionEvidence?.rateSetId)
    expect(second.headers['Idempotency-Key']).toBe(first.headers['Idempotency-Key'])
    expect(second.headers['X-Correlation-Id']).toBe(first.headers['X-Correlation-Id'])
  })
})
