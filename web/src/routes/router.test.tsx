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
