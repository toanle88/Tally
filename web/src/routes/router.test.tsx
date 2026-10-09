import { fireEvent, render, screen } from '@testing-library/react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { describe, expect, it } from 'vitest'

import { developmentScopes } from '@/app/app'
import { ScopeProvider } from '@/lib/scope/scope-context'

import { AuthScopeProvider, createAppRoutes } from './router'

const resolution = {
  status: 'authenticated' as const,
  actorLabel: 'Fixture actor',
  availableScopes: developmentScopes,
  initialScopeId: developmentScopes[0].id,
}

function renderRoute(initialEntry: string) {
  const router = createMemoryRouter(createAppRoutes(), { initialEntries: [initialEntry] })
  render(
    <AuthScopeProvider resolution={resolution}>
      <ScopeProvider availableScopes={developmentScopes} initialScopeId={resolution.initialScopeId} effects={{ cancelInFlightWork: () => {}, clearScopeBoundQueryState: () => {} }}>
        <RouterProvider router={router} />
      </ScopeProvider>
    </AuthScopeProvider>,
  )
  return router
}

describe('application router', () => {
  it('supports direct operational-route entry and unknown-route handling', async () => {
    renderRoute('/operations/xct-ws-01')
    expect(await screen.findByRole('heading', { name: 'Cross-context event exception worklist' })).toBeInTheDocument()

    renderRoute('/unknown')
    expect(await screen.findByRole('heading', { name: 'Page not found' })).toBeInTheDocument()
  })

  it('supports the OMD worklist and record mutation boundary', async () => {
    renderRoute('/master-data/omd-ws-01')
    expect(await screen.findByRole('heading', { name: 'Legal-entity master-data worklist' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('link', { name: 'Acme Vietnam Co., Ltd.' }))
    expect(await screen.findByRole('heading', { name: 'Legal-entity record' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save legal-entity revision' })).toBeInTheDocument()
    expect(screen.getAllByText('Access restricted. The protected value remains masked and export is unavailable.').length).toBeGreaterThan(0)
  })

  it('supports the Party maintenance screen with safe bank-control data', async () => {
    renderRoute('/master-data/omd-scr-02?partyId=party-acme-vendor')
    expect(await screen.findByRole('heading', { name: 'Party record' })).toBeInTheDocument()
    expect(screen.getByDisplayValue('Acme Industrial Supplies')).toBeInTheDocument()
    expect(screen.getByText('provider-ref-001')).toBeInTheDocument()
    expect(screen.getAllByText('Access restricted. The protected value remains masked and export is unavailable.').length).toBeGreaterThan(0)
    fireEvent.click(screen.getByRole('button', { name: 'Save party fixture revision' }))
    expect(screen.getByText(/Accepted fixture revision v6/)).toBeInTheDocument()
  })

  it('supports the customer-profile maintenance screen with Party and profile versions', async () => {
    renderRoute('/master-data/omd-scr-03?customerProfileId=customer-profile-northwind')
    expect(await screen.findByRole('heading', { name: 'Customer profile record' })).toBeInTheDocument()
    expect(screen.getByDisplayValue('Northwind Distribution')).toBeInTheDocument()
    expect(screen.getByDisplayValue('v2')).toBeInTheDocument()
    expect(screen.getByDisplayValue('250000')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save customer-profile revision' }))
    expect(screen.getByText(/Accepted customer-profile revision v4/)).toBeInTheDocument()
  })

  it('supports the vendor-profile maintenance screen with safe Party-owned remittance state', async () => {
    renderRoute('/master-data/omd-scr-03?vendorProfileId=vendor-profile-acme-industrial')
    expect(await screen.findByRole('heading', { name: 'Vendor profile record' })).toBeInTheDocument()
    expect(screen.getByDisplayValue('Acme Industrial Supplies')).toBeInTheDocument()
    expect(screen.getByDisplayValue('v5')).toBeInTheDocument()
    expect(screen.getByDisplayValue('net_30')).toBeInTheDocument()
    expect(screen.getByText(/Party bank control: approved/)).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save vendor-profile revision' }))
    expect(screen.getByText(/Accepted vendor-profile revision v5/)).toBeInTheDocument()
  })

  it('supports the COA segment worklist and record mutation boundary', async () => {
    renderRoute('/coa-segments/coa-ws-01')
    expect(await screen.findByRole('heading', { name: 'Segment administration worklist' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('link', { name: 'Create segment definition' }))
    expect(await screen.findByRole('heading', { name: 'Segment definition' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create segment definition' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('local safe adapter')
  })

  it('supports direct entry to the COA combination validator', async () => {
    renderRoute('/coa-segments/coa-scr-03')
    expect(await screen.findByRole('heading', { name: 'Segment combination validator' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Validate combination' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('Validation is read-only')
  })

  it('supports direct entry to the COA segment change request screen', async () => {
    renderRoute('/coa-segments/coa-scr-04')
    expect(await screen.findByRole('heading', { name: 'Segment change request' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Request definition change' })).toBeInTheDocument()
    expect(screen.getByRole('note')).toHaveTextContent('does not mutate the subject')
  })

  it('updates active navigation and supports back navigation', async () => {
    const router = renderRoute('/')
    fireEvent.click(screen.getByRole('link', { name: 'Exceptions' }))
    expect(await screen.findByRole('heading', { name: 'Exceptions' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Exceptions' })).toHaveAttribute('aria-current', 'page')

    await router.navigate(-1)
    expect(await screen.findByRole('heading', { name: 'Home' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')
  })
})
