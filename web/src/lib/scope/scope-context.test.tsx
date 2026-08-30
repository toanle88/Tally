import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { useRef } from 'react'

import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'

import { ScopeProvider, useScopeContext } from './scope-context'

const scopes: AccountingScopeFixture[] = [
  {
    id: 'one',
    tenant: { id: 'tenant', name: 'Tenant' },
    legalEntity: { id: 'entity-one', name: 'Entity One' },
    ledger: { id: 'ledger', name: 'Ledger' },
    accountingBook: { id: 'book-one', name: 'Book One' },
    functionalCurrency: 'VND',
  },
  {
    id: 'two',
    tenant: { id: 'tenant', name: 'Tenant' },
    legalEntity: { id: 'entity-two', name: 'Entity Two' },
    ledger: { id: 'ledger', name: 'Ledger' },
    accountingBook: { id: 'book-two', name: 'Book Two' },
    functionalCurrency: 'SGD',
  },
]

function Harness() {
  const context = useScopeContext()
  const snapshot = useRef(context.captureScopeSnapshot())
  return (
    <>
      <output data-testid="current-scope">{context.currentScope?.id}</output>
      <output data-testid="revision">{context.scopeRevision}</output>
      <output data-testid="status">{context.statusMessage}</output>
      <button onClick={() => context.setHasUnsavedChanges(true)}>Make dirty</button>
      <button onClick={() => context.requestScopeChange('two')}>Request second</button>
      <button onClick={context.confirmScopeChange}>Confirm</button>
      <button onClick={context.cancelScopeChange}>Cancel</button>
      <button onClick={() => context.runIfCurrentScope(snapshot.current, () => undefined)}>Submit captured work</button>
    </>
  )
}

describe('ScopeProvider', () => {
  it('runs the scope-change integration effects and increments the revision', () => {
    const cancelInFlightWork = vi.fn()
    const clearScopeBoundQueryState = vi.fn()

    render(
      <ScopeProvider availableScopes={scopes} initialScopeId="one" effects={{ cancelInFlightWork, clearScopeBoundQueryState }}>
        <Harness />
      </ScopeProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Request second' }))

    expect(screen.getByTestId('current-scope')).toHaveTextContent('two')
    expect(screen.getByTestId('revision')).toHaveTextContent('1')
    expect(cancelInFlightWork).toHaveBeenCalledOnce()
    expect(clearScopeBoundQueryState).toHaveBeenCalledOnce()
  })

  it('requires explicit discard when current work is dirty', () => {
    render(
      <ScopeProvider availableScopes={scopes} initialScopeId="one" effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>
        <Harness />
      </ScopeProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Make dirty' }))
    fireEvent.click(screen.getByRole('button', { name: 'Request second' }))
    expect(screen.getByTestId('current-scope')).toHaveTextContent('one')

    fireEvent.click(screen.getByRole('button', { name: 'Confirm' }))
    expect(screen.getByTestId('current-scope')).toHaveTextContent('two')
  })

  it('rejects submission captured under an older scope revision', () => {
    render(
      <ScopeProvider availableScopes={scopes} initialScopeId="one" effects={{ cancelInFlightWork: vi.fn(), clearScopeBoundQueryState: vi.fn() }}>
        <Harness />
      </ScopeProvider>,
    )

    fireEvent.click(screen.getByRole('button', { name: 'Request second' }))
    fireEvent.click(screen.getByRole('button', { name: 'Submit captured work' }))

    expect(screen.getByTestId('status')).toHaveTextContent('older accounting scope')
  })
})
