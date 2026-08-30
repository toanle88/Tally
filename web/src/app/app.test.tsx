import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import App from './app'

describe('App', () => {
  it('renders the routed shell and accounting scope context', () => {
    const { container } = render(<App />)

    expect(screen.getByRole('heading', { name: 'TALLY', level: 1 })).toBeInTheDocument()
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
