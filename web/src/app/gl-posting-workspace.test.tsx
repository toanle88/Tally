import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { glApplyJournalApprovalDecision, glReverseJournalEntry, glSubmitPostingRequest } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { GlPostingWorkspace } from './gl-posting-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  glApplyJournalApprovalDecision: vi.fn(),
  glReverseJournalEntry: vi.fn(),
  glSubmitPostingRequest: vi.fn(),
}))

function renderWithScope(ui: ReactNode, initialEntries?: Array<string | { pathname: string; state?: unknown }>) {
  return render(<MemoryRouter initialEntries={initialEntries}><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
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
    vi.mocked(glApplyJournalApprovalDecision).mockReset()
    vi.mocked(glReverseJournalEntry).mockReset()
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
    expect(screen.getAllByText('JE-2026-000001')).toHaveLength(2)
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

  it('shows immutable decision evidence and applies a pending journal through the generic command route', async () => {
    const pendingResult = {
      ...establishedResult,
      aggregateVersion: 1,
      data: {
        ...establishedResult.data,
        outcome: 'PostingPendingApproval',
        lifecycleStatus: 'PendingApproval',
        journalNumber: 'PJE-2026-000001',
        journalVersion: 1,
        ledgerPosition: undefined,
        approvalRequestId: 'approval-request-1',
        policyReference: 'workflow-policy',
        policyVersion: '1',
      },
    }
    const appliedResult = {
      status: 'established',
      aggregateId: 'journal-1',
      aggregateVersion: 2,
      correlationId: 'correlation-apply-1',
      links: { self: '/api/v1/general-ledger/actions/apply-journal-approval-decision' },
      data: { outcome: 'JournalEntryPosted', lifecycleStatus: 'Posted', approvalDecisionId: 'decision-1', subjectVersion: 1, journalVersion: 2, nextAction: 'No action required' },
    }
    vi.mocked(glApplyJournalApprovalDecision).mockResolvedValue({ data: appliedResult, error: undefined } as never)
    renderWithScope(<GlPostingWorkspace view="result" />, [{ pathname: '/general-ledger/gl-scr-02', state: { postingResult: pendingResult } }])

    expect(screen.getByText('Apply journal approval decision')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('textbox', { name: 'Candidate fingerprint' }), { target: { value: 'sha256:pending-candidate' } })
    fireEvent.change(screen.getByRole('textbox', { name: 'Approver user ID' }), { target: { value: 'approver-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Apply approval' }))

    await waitFor(() => expect(glApplyJournalApprovalDecision).toHaveBeenCalledTimes(1))
    const request = vi.mocked(glApplyJournalApprovalDecision).mock.calls[0][0] as { body: { commandId: string; expectedVersion: number; data: { journalId: string; outcome: string; approvalRequestId: string; candidateFingerprint: string } }; headers: { 'Idempotency-Key': string } }
    expect(request.body.expectedVersion).toBe(1)
    expect(request.body.data.journalId).toBe('journal-1')
    expect(request.body.data.outcome).toBe('approved')
    expect(request.body.data.approvalRequestId).toBe('approval-request-1')
    expect(request.body.data.candidateFingerprint).toBe('sha256:pending-candidate')
    expect(request.headers['Idempotency-Key']).toBeTruthy()
    expect(screen.getByText('Decision applied')).toBeInTheDocument()
  })

  it('previews and submits a linked reversal with explicit reason and source evidence', async () => {
    const reversalResult = {
      status: 'established',
      aggregateId: 'reversal-1',
      aggregateVersion: 1,
      correlationId: 'reversal-correlation-1',
      links: { self: '/api/v1/general-ledger/actions/reverse-journal-entry' },
      data: { outcome: 'JournalEntryReversed', originalJournalId: 'journal-1', reversalJournalId: 'reversal-1', reversalJournalNumber: 'JE-2026-000002', correctionReason: 'Correct classification', auditReference: 'audit-reversal-1', replayed: false },
    }
    vi.mocked(glReverseJournalEntry).mockResolvedValue({ data: reversalResult, error: undefined } as never)
    const postedResult = {
      ...establishedResult,
      data: {
        ...establishedResult.data,
        lines: [
          { accountId: 'account-cash', debitOrCredit: 'debit', lineCurrencyMode: 'TransactionAndFunctional', transactionAmount: '100.00', functionalAmount: '100.00', segmentCombinationId: 'combination-cash' },
          { accountId: 'account-revenue', debitOrCredit: 'credit', lineCurrencyMode: 'TransactionAndFunctional', transactionAmount: '100.00', functionalAmount: '100.00', segmentCombinationId: 'combination-revenue' },
        ],
      },
    }
    renderWithScope(<GlPostingWorkspace view="result" />, [{ pathname: '/general-ledger/gl-scr-02', state: { postingResult: postedResult } }])

    expect(screen.getByText('Equal-and-opposite preview')).toBeInTheDocument()
    fireEvent.change(screen.getByRole('textbox', { name: 'Correction reason' }), { target: { value: 'Correct classification' } })
    fireEvent.change(screen.getByRole('textbox', { name: 'Authorization evidence ID' }), { target: { value: 'authorization-1' } })
    fireEvent.click(screen.getByRole('button', { name: 'Reverse posted journal' }))

    await waitFor(() => expect(glReverseJournalEntry).toHaveBeenCalledTimes(1))
	const request = vi.mocked(glReverseJournalEntry).mock.calls[0][0] as unknown as { body: { expectedVersion: number; data: { journalId: string; reason: string; postingAuthorizationId: string; sourceContext: string; sourceVersion: number } }; headers: { 'Idempotency-Key': string } }
    expect(request.body.expectedVersion).toBe(1)
    expect(request.body.data.journalId).toBe('journal-1')
    expect(request.body.data.reason).toBe('Correct classification')
    expect(request.body.data.postingAuthorizationId).toBe('authorization-1')
    expect(request.body.data.sourceContext).toBe('general-ledger')
    expect(request.body.data.sourceVersion).toBe(1)
    expect(request.headers['Idempotency-Key']).toBeTruthy()
    expect(screen.getByText('Linked reversal established')).toBeInTheDocument()
    expect(screen.getByText('JE-2026-000002')).toBeInTheDocument()
  })
})
