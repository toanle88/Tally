import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { createFixtureAuthClient } from '@/lib/auth/auth-client'

import App, { developmentScopes } from './app'

describe('App', () => {
  it('renders the routed shell and accounting scope context', () => {
    const authClient = createFixtureAuthClient({ actorLabel: 'Test fixture user', availableScopes: developmentScopes, initialScopeId: developmentScopes[0].id })
    const logout = vi.fn().mockResolvedValue(undefined)
    const appAuthClient = { ...authClient, logout }
    const { container } = render(<App authClient={appAuthClient} />)

    expect(screen.getByRole('heading', { name: 'TALLY', level: 1 })).toBeInTheDocument()
    expect(screen.getByLabelText('Signed-in username')).toHaveTextContent('Test fixture user')
    expect(screen.getByRole('button', { name: 'Sign out' })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Sign out' }))
    expect(logout).toHaveBeenCalledOnce()
    expect(screen.getByRole('navigation', { name: 'Global navigation' })).toBeInTheDocument()
    expect(screen.getByRole('main', { name: 'Application content' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Accounting scope context' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Status feedback' })).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Home' })).toHaveAttribute('aria-current', 'page')
    expect(screen.getByRole('combobox', { name: 'Accounting scope' })).toHaveValue('scope-vietnam-statutory')
    expect(screen.getByText('Acme Vietnam Co., Ltd. (fixture)')).toBeInTheDocument()
    expect(screen.getByText('VND')).toBeInTheDocument()
    expect(container.querySelector('[data-theme="finance-light"]')).toBeInTheDocument()
  })
})
