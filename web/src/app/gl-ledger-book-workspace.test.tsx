import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import { glMaintainAccountingBooks, glMaintainLedgers } from '@/generated/api/sdk.gen'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { developmentScopes } from './app'
import { GlLedgerBookWorkspace } from './gl-ledger-book-workspace'

vi.mock('@/generated/api/sdk.gen', () => ({
  glMaintainAccountingBooks: vi.fn(),
  glMaintainLedgers: vi.fn(),
}))

function renderWithScope(ui: ReactNode) {
  return render(<MemoryRouter><ScopeProvider availableScopes={developmentScopes} initialScopeId={developmentScopes[0].id} effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>{ui}</ScopeProvider></MemoryRouter>)
}

describe('GL ledger and accounting-book workspace', () => {
  beforeEach(() => {
    vi.mocked(glMaintainLedgers).mockReset()
    vi.mocked(glMaintainAccountingBooks).mockReset()
  })

  it('shows scoped safe projections and blocks an invalid lifecycle transition', () => {
    renderWithScope(<GlLedgerBookWorkspace />)

    expect(screen.getByRole('heading', { name: 'Ledger and accounting-book configuration' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('local safe adapter')
    expect(screen.getByRole('button', { name: 'primary' })).toBeInTheDocument()
    fireEvent.change(screen.getByRole('combobox', { name: 'Lifecycle status' }), { target: { value: 'draft' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save ledger revision' }))

    expect(screen.getByRole('alert')).toHaveTextContent('The lifecycle transition from active to draft is not allowed.')
    expect(glMaintainLedgers).not.toHaveBeenCalled()
  })

  it('submits typed live commands and refreshes safe projections from the result', async () => {
    vi.mocked(glMaintainLedgers).mockResolvedValue({ data: { status: 'established', aggregateId: 'ledger-vietnam', aggregateVersion: 3, correlationId: 'correlation', links: { self: '/api/v1/general-ledger/configuration/maintain-ledgers' }, data: { ledger: { id: 'ledger-vietnam', accountingScopeId: 'scope-vietnam-statutory', legalEntityId: 'entity-vietnam', ledgerType: 'primary', functionalCurrency: 'VND', fiscalCalendarId: 'fiscal-calendar-vietnam', lifecycleStatus: 'active', effectiveDateFrom: '2026-01-01', effectiveDateTo: null, approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 3, revisionNumber: 3 }, validationOutcome: 'valid', approvalStatus: 'approved' } }, error: undefined } as never)
    vi.mocked(glMaintainAccountingBooks).mockResolvedValue({ data: { status: 'established', aggregateId: 'book-vietnam-statutory', aggregateVersion: 3, correlationId: 'correlation', links: { self: '/api/v1/general-ledger/configuration/maintain-accounting-books' }, data: { accountingBook: { id: 'book-vietnam-statutory', accountingScopeId: 'scope-vietnam-statutory', ledgerId: 'ledger-vietnam', bookType: 'statutory', accountingBasis: 'accrual', postingPolicyVersion: 'posting-v1', lifecycleStatus: 'active', effectiveDateFrom: '2026-01-01', effectiveDateTo: null, approvalStatus: 'approved', validationOutcome: 'valid', nextAction: 'maintain', version: 3, revisionNumber: 3 }, validationOutcome: 'valid', approvalStatus: 'approved' } }, error: undefined } as never)

    renderWithScope(<GlLedgerBookWorkspace />)
    fireEvent.click(screen.getByRole('button', { name: 'Save ledger revision' }))
    await waitFor(() => expect(glMaintainLedgers).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/Accepted ledger revision v3/)).toBeInTheDocument()
    expect(screen.getAllByText('v3').length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('button', { name: 'Save accounting-book revision' }))
    await waitFor(() => expect(glMaintainAccountingBooks).toHaveBeenCalledTimes(1))
    expect(await screen.findByText(/Accepted accounting-book revision v3/)).toBeInTheDocument()
  })
})
