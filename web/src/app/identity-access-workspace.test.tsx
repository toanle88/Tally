import { fireEvent, render, screen, within } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { IdentityAccessWorkspace } from './identity-access-workspace'

describe('identity access workspace', () => {
  it('shows masked identity data and current assignment evidence', () => {
    render(<IdentityAccessWorkspace />)

    expect(screen.getByRole('heading', { name: 'Users and access assignments' })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'User access worklist' })).toBeInTheDocument()
    expect(screen.getByText('Authentication subject reference')).toBeInTheDocument()
    expect(screen.queryByText('subject-ref-01')).not.toBeInTheDocument()
    expect(screen.getAllByText('••••••••').length).toBeGreaterThan(0)
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
    const roleGrants = screen.getByRole('table', { name: 'Permission grants for Scoped finance operator' })
    expect(within(roleGrants).getByText('finance.gl.submit.posting.request')).toBeInTheDocument()
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

  it('explains distinct scoped access outcomes and keeps field access independent', () => {
    render(<IdentityAccessWorkspace />)

    expect(screen.getByRole('heading', { name: 'IAM-SCR-05 · Access decision explanation' })).toBeInTheDocument()
    expect(screen.getByText('policy-2026.09-v4')).toBeInTheDocument()
    expect(screen.getByText('decision-allowed-001')).toBeInTheDocument()
    expect(screen.getByText('Legal entity + segment + action')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'stale' }))
    expect(screen.getByRole('status', { name: 'stale' })).toBeInTheDocument()
    expect(screen.getByText('Refresh policy state and retry with the current version.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'expired' }))
    expect(screen.getByRole('status', { name: 'expired' })).toBeInTheDocument()
    expect(screen.getByText('Request a current policy revision before retrying.')).toBeInTheDocument()

    fireEvent.click(screen.getAllByRole('button', { name: 'Reveal unavailable' })[0])
    expect(screen.getByText('Field reveal and export remain unavailable; record access does not grant field access.')).toBeInTheDocument()
  })

  it('administers versioned segregation rules and explains safe outcomes', () => {
    render(<IdentityAccessWorkspace />)

    expect(screen.getByRole('heading', { name: 'IAM-SCR-03 · Segregation rule administration' })).toBeInTheDocument()
    expect(screen.getAllByText('Payment batch preparer and approver').length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: 'Review conflict explanation' }))
    expect(screen.getByRole('status', { name: 'conflict' })).toBeInTheDocument()
    expect(screen.getByText('Use an independent actor or request an approved exception where the rule permits one.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Simulate stale rule' }))
    expect(screen.getByRole('status', { name: 'stale' })).toBeInTheDocument()
    expect(screen.getByText('Refresh current IAM policy state and retry.')).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Retire rule revision' }))
    expect(screen.getByText(/historical revisions remain available/)).toBeInTheDocument()
  })

  it('grants, revokes, expires, and reviews emergency access with safe denial fixtures', () => {
    render(<IdentityAccessWorkspace />)

    expect(screen.getByRole('heading', { name: 'IAM-SCR-04 · Emergency access' })).toBeInTheDocument()
    expect(screen.getByText('grant-003')).toBeInTheDocument()
    expect(screen.getAllByText('overdue').length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: 'Revoke selected grant' }))
    expect(screen.getByText(/grant-001 was revoked/)).toBeInTheDocument()
    expect(screen.getAllByRole('status', { name: 'revoked' }).length).toBeGreaterThan(0)

    fireEvent.click(screen.getByRole('button', { name: 'Complete review' }))
    expect(screen.getByText(/review was completed with outcome review-code-001/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Simulate denial' }))
    fireEvent.click(screen.getByRole('button', { name: 'Grant emergency access' }))
    expect(screen.getByRole('alert')).toHaveTextContent('outside the current policy decision')
    expect(screen.getByText(/no privileged access side effect/)).toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Require step-up' }))
    fireEvent.click(screen.getByRole('button', { name: 'Grant emergency access' }))
    expect(screen.getByRole('alert')).toHaveTextContent('step-up challenge')

    fireEvent.click(screen.getByRole('button', { name: 'Simulate emergency version conflict' }))
    expect(screen.getByRole('dialog')).toBeInTheDocument()
    expect(screen.getByText('Grant lifecycle state')).toBeInTheDocument()
  })
})
