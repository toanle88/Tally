import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { IdentityAccessWorkspace } from './identity-access-workspace'

describe('identity access workspace', () => {
  it('shows masked identity data and current assignment evidence', () => {
    render(<IdentityAccessWorkspace />)

    expect(screen.getByRole('heading', { name: 'Users and access assignments' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'User access worklist' })).toBeInTheDocument()
    expect(screen.getByText('Authentication subject reference')).toBeInTheDocument()
    expect(screen.queryByText('subject-ref-01')).not.toBeInTheDocument()
    expect(screen.getByText('••••••••')).toBeInTheDocument()
  })

  it('supports lifecycle actions and safe validation states', () => {
    render(<IdentityAccessWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Suspend' }))
    expect(screen.getByRole('status', { name: 'suspended' })).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Review duplicate result' }))
    fireEvent.click(screen.getByRole('button', { name: 'Replace assignments' }))
    expect(screen.getByRole('alert')).toHaveTextContent('safe duplicate')
    expect(screen.getByText('The command was not sent and no assignment side effect was applied.')).toBeInTheDocument()
  })

  it('opens a version conflict recovery dialog', () => {
    render(<IdentityAccessWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Simulate version conflict' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'The record changed before this action' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Refresh authoritative state' })).toBeInTheDocument()
  })
  it('shows versioned role grants, approval state, and effective dates', () => {
    render(<IdentityAccessWorkspace />)

    expect(screen.getByRole('heading', { name: 'Role and permission worklist' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'IAM-SCR-02 - Role detail' })).toBeInTheDocument()
    expect(screen.getByText('finance.gl.submit.posting.request')).toBeInTheDocument()
    expect(screen.getAllByText('2026-10-01').length).toBeGreaterThan(0)
    expect(screen.getAllByText('approved').length).toBeGreaterThan(0)
  })

  it('shows safe duplicate validation and non-destructive retirement', () => {
    render(<IdentityAccessWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Review duplicate grant' }))
    fireEvent.click(screen.getByRole('button', { name: 'Replace grant set' }))
    expect(screen.getByRole('alert')).toHaveTextContent('duplicates an existing permission')
    expect(screen.getByText('The candidate revision was rejected and no role state changed.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Retire role' }))
    expect(screen.getAllByRole('status', { name: 'retired' }).length).toBeGreaterThan(0)
    expect(screen.getByText(/retired non-destructively/)).toBeInTheDocument()
  })

  it('opens accessible role version conflict recovery', () => {
    render(<IdentityAccessWorkspace />)

    fireEvent.click(screen.getByRole('button', { name: 'Simulate role version conflict' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Permission grants')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Refresh authoritative state' })).toBeInTheDocument()
  })

})
