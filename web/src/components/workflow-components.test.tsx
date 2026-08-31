import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

import { ApprovalPanel } from './approval-panel'
import { PostingPanel } from './posting-panel'
import { ValidationSummary } from './validation-summary'
import { VersionConflictDialog } from './version-conflict-dialog'
import { approvalFixture, conflictFixture, postingFixture, workflowOutcomeFixtures } from '@/app/workflow-fixtures'
import { formatDecimalInput, normalizeDecimalInput } from './workflow-context'

describe('Story 5 workflow components', () => {
  it('normalizes locale input to a canonical decimal string without numeric coercion', () => {
    expect(normalizeDecimalInput('1.250,50', 'de-DE')).toBe('1250.50')
    expect(formatDecimalInput('1250.50', 'de-DE')).toBe('1.250,50')
  })

  it('links validation issues to their field targets', () => {
    render(<ValidationSummary issues={[{ id: 'amount', category: 'field', code: 'required', message: 'Enter an amount.', targetId: 'amount', targetLabel: 'Amount' }]} />)
    expect(screen.getByRole('link', { name: /Amount: Enter an amount/ })).toHaveAttribute('href', '#amount')
  })

  it('focuses the validation summary when issues arrive', () => {
    render(<ValidationSummary issues={[{ id: 'amount', category: 'field', code: 'required', message: 'Enter an amount.', targetId: 'amount', targetLabel: 'Amount' }]} />)
    expect(document.activeElement).toHaveAttribute('role', 'alert')
  })

  it('keeps decided approval separate from application', () => {
    render(<ApprovalPanel approval={approvalFixture} />)
    expect(screen.getAllByText('decided', { selector: '[data-state]' })).toHaveLength(2)
    expect(screen.getByText(/Decision exists but is not applied/)).toBeInTheDocument()
  })

  it('shows posting retry only when the fixture permits deliberate retry', () => {
    const retry = vi.fn()
    render(<PostingPanel posting={postingFixture} onRetry={retry} />)
    expect(screen.getByRole('button', { name: 'Retry deliberately' })).toBeInTheDocument()
  })

  it('renders conflict evidence and safe recovery controls', () => {
    render(<VersionConflictDialog conflict={conflictFixture} onClose={vi.fn()} onRefresh={vi.fn()} onRetry={vi.fn()} />)
    expect(screen.getByRole('dialog')).toHaveAttribute('aria-modal', 'true')
    expect(screen.getByText('settlement-epoch-2026-08-31-03')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Refresh authoritative state' })).toBeInTheDocument()
  })

  it('provides every typed outcome fixture category', () => {
    expect(Object.keys(workflowOutcomeFixtures)).toEqual(expect.arrayContaining(['success', 'domain-rejection', 'authorization-denial', 'version-conflict', 'idempotency-conflict', 'dependency-unavailable', 'ambiguous-outcome', 'unexpected-failure']))
  })
})
