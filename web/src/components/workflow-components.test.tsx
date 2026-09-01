import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { useState } from 'react'
import { describe, expect, it, vi } from 'vitest'

import { ApprovalPanel } from './approval-panel'
import { PostingPanel } from './posting-panel'
import { ValidationSummary } from './validation-summary'
import { VersionConflictDialog } from './version-conflict-dialog'
import { approvalFixture, conflictFixture, postingFixture, workflowOutcomeFixtures } from '@/app/workflow-fixtures'
import { ConfirmationSurface } from './ui'
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

  it('focuses dynamic confirmations and restores their triggering control', async () => {
    function ConfirmationHarness() {
      const [open, setOpen] = useState(false)
      return <><button onClick={() => setOpen(true)}>Open confirmation</button>{open ? <ConfirmationSurface autoFocus title="Confirm fixture" description="Confirm fixture action." confirmLabel="Confirm" cancelLabel="Cancel" onConfirm={() => setOpen(false)} onCancel={() => setOpen(false)} /> : null}</>
    }

    render(<ConfirmationHarness />)
    const trigger = screen.getByRole('button', { name: 'Open confirmation' })
    trigger.focus()
    fireEvent.click(trigger)
    await waitFor(() => expect(screen.getByRole('button', { name: 'Cancel' })).toHaveFocus())
    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(trigger).toHaveFocus())
  })

  it('traps conflict-dialog focus and restores the opener on Escape', async () => {
    function ConflictHarness() {
      const [open, setOpen] = useState(false)
      return <><button onClick={() => setOpen(true)}>Open conflict</button>{open ? <VersionConflictDialog conflict={conflictFixture} onClose={() => setOpen(false)} onRefresh={() => undefined} onRetry={() => setOpen(false)} /> : null}</>
    }

    render(<ConflictHarness />)
    const trigger = screen.getByRole('button', { name: 'Open conflict' })
    trigger.focus()
    fireEvent.click(trigger)
    const dialog = await screen.findByRole('dialog')
    const close = screen.getByRole('button', { name: 'Close' })
    const retry = screen.getByRole('button', { name: 'Retry after review' })
    await waitFor(() => expect(close).toHaveFocus())

    fireEvent.keyDown(dialog, { key: 'Tab', shiftKey: true })
    expect(retry).toHaveFocus()
    fireEvent.keyDown(dialog, { key: 'Escape' })
    await waitFor(() => expect(trigger).toHaveFocus())
  })

  it('provides every typed outcome fixture category', () => {
    expect(Object.keys(workflowOutcomeFixtures)).toEqual(expect.arrayContaining(['success', 'domain-rejection', 'authorization-denial', 'version-conflict', 'idempotency-conflict', 'dependency-unavailable', 'ambiguous-outcome', 'unexpected-failure']))
  })
})
