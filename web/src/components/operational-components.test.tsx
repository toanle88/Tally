import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { ExceptionResolutionPanel } from '@/components/exception-resolution-panel'
import { ProcessProgressPanel } from '@/components/process-progress-panel'
import { ResultLookup } from '@/components/result-lookup'
import { SettlementAndReconciliationPanel } from '@/components/settlement-and-reconciliation-panel'
import { StateAwareActionBar } from '@/components/state-aware-action-bar'
import { WorklistAndSavedFilters } from '@/components/worklist-and-saved-filters'
import { exceptionFixture, operationalActions, operationalSavedViews, operationalWorklistFixture, processFixture, resultLookupFixtures, settlementFixture } from '@/app/operational-fixtures'

describe('operational components', () => {
  it('filters exact amount strings, keeps ineligible rows disabled, and invokes eligible bulk work', () => {
    const onBulkAction = vi.fn()
    render(<WorklistAndSavedFilters items={operationalWorklistFixture} status="ready" savedViews={operationalSavedViews} initialFilters={{ scopeId: 'scope-vietnam-statutory' }} pageSize={2} bulkAction={{ id: 'assign', label: 'Assign selected work', permitted: true, material: false }} onBulkAction={onBulkAction} />)

    expect(screen.getByRole('link', { name: 'Receipt RCPT-FIX-00042' })).toBeInTheDocument()
    expect(screen.getByRole('checkbox', { name: 'Select Receipt RCPT-FIX-00042' })).toBeEnabled()
    expect(screen.getByRole('checkbox', { name: 'Select Settlement SET-FIX-00017' })).toBeDisabled()
    fireEvent.change(screen.getByRole('textbox', { name: 'Amount' }), { target: { value: '1000.00' } })
    expect(screen.getByRole('link', { name: 'Receipt RCPT-FIX-00042' })).toBeInTheDocument()
    expect(screen.queryByRole('link', { name: 'Settlement SET-FIX-00017' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('checkbox', { name: 'Select Receipt RCPT-FIX-00042' }))
    fireEvent.click(screen.getByRole('button', { name: 'Assign selected work' }))
    expect(onBulkAction).toHaveBeenCalledWith([operationalWorklistFixture[0]])
  })

  it('announces unavailable and rejected worklist states without presenting stale rows as ready', () => {
    for (const status of ['loading', 'unavailable', 'rejected'] as const) {
      const { unmount } = render(<WorklistAndSavedFilters items={operationalWorklistFixture} status={status} savedViews={[]} />)
      expect(screen.getByText(status === 'loading' ? /Worklist is loading/ : status === 'unavailable' ? /dependency unavailable/ : /request was rejected/)).toBeInTheDocument()
      expect(screen.queryByRole('table')).not.toBeInTheDocument()
      unmount()
    }
  })

  it('renders empty, partial, and reconciled worklist states with explicit empty-result behavior', () => {
    const { unmount } = render(<WorklistAndSavedFilters items={[]} status="ready" savedViews={[]} />)
    expect(screen.getByText('No matching work items.')).toBeInTheDocument()
    unmount()

    for (const status of ['partial', 'reconciled'] as const) {
      const view = render(<WorklistAndSavedFilters items={operationalWorklistFixture} status={status} savedViews={[]} />)
      expect(screen.getByRole('table', { name: 'Operational work items' })).toBeInTheDocument()
      expect(screen.getByRole('status', { name: status === 'partial' ? 'Partial result' : 'Reconciled' })).toBeInTheDocument()
      view.unmount()
    }
  })

  it('explains blocked actions and exposes the material result refresh announcement', () => {
    const onAction = vi.fn()
    render(<StateAwareActionBar actions={operationalActions} lastMaterialResult={{ actionId: 'settle', resultLabel: 'Apply settlement', stateLabel: 'Pending owner result', message: 'Refresh complete.', refreshedAt: '2026-08-30T11:45:00Z' }} onAction={onAction} />)

    const blocked = screen.getByRole('button', { name: /Apply settlement/ })
    expect(blocked).toBeDisabled()
    expect(screen.getByText(/Review the owner acknowledgement exception/)).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Refresh complete.')
    fireEvent.click(screen.getByRole('button', { name: 'Review result' }))
    expect(onAction).toHaveBeenCalledWith(operationalActions[0])
  })

  it('renders settlement balances verbatim and authorized exception resolution with evidence access states', () => {
    const onResolve = vi.fn()
    render(<><SettlementAndReconciliationPanel settlement={settlementFixture} /><ExceptionResolutionPanel exception={exceptionFixture} onResolve={onResolve} /></>)

    expect(screen.getByText('1000.00 USD')).toBeInTheDocument()
    expect(screen.getByText('125.00 USD')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Bank allocation evidence/ })).toBeInTheDocument()
    expect(screen.getByText(/Evidence restricted/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Apply corrected owner reference' })).toBeEnabled()
    expect(screen.getByRole('button', { name: /Reclassify/ })).toBeDisabled()
    fireEvent.click(screen.getByRole('button', { name: 'Apply corrected owner reference' }))
    expect(onResolve).toHaveBeenCalledWith(exceptionFixture.authorizedResolutions[0])
  })

  it('shows safe duplicate, ambiguous, and identity-content conflict outcomes', () => {
    for (const outcome of resultLookupFixtures) {
      const { unmount } = render(<ResultLookup query={outcome.query} status="found" outcome={outcome} onLookup={vi.fn()} />)
      expect(screen.getByText(outcome.status.label)).toBeInTheDocument()
      if (outcome.establishedResult) expect(screen.getByRole('link', { name: outcome.establishedResult.label })).toBeInTheDocument()
      if (outcome.kind === 'identity-content-conflict') expect(screen.getByRole('link', { name: outcome.requiredNewBusinessIdentity.label })).toBeInTheDocument()
      unmount()
    }
  })

  it('does not display a stale result while lookup is unavailable', () => {
    render(<ResultLookup query={resultLookupFixtures[0].query} status="unavailable" outcome={resultLookupFixtures[0]} onLookup={vi.fn()} />)

    expect(screen.getByText(/Result lookup is unavailable/)).toBeInTheDocument()
    expect(screen.queryByText(/Safe duplicate/)).not.toBeInTheDocument()
  })

  it('renders process progress, owners, partial failure, and established result links', () => {
    render(<ProcessProgressPanel process={processFixture} />)

    expect(screen.getByText('Partially completed')).toBeInTheDocument()
    expect(screen.getByText('Cash posting')).toBeInTheDocument()
    expect(screen.getAllByText('Payments & Cash Management', { exact: false }).length).toBeGreaterThan(0)
    expect(screen.getByRole('link', { name: 'Receipt detail' })).toHaveAttribute('href', '#record-detail-context-example')
    expect(screen.getByText(/Resolution remains pending/)).toBeInTheDocument()
  })
})
