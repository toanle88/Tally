import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import {
  Button,
  ConfirmationSurface,
  DataTable,
  Field,
  Heading,
  Link,
  Panel,
  StatusBadge,
  type SemanticState,
} from './index'

const states: readonly SemanticState[] = [
  'success',
  'warning',
  'error',
  'info',
  'pending',
  'reconciled',
  'restricted',
  'disabled',
]

describe('shared UI primitives', () => {
  it.each(states)('renders the %s state with visible meaning and an accessible name', (state) => {
    render(<StatusBadge state={state} announce />)

    const badge = screen.getByRole('status')
    expect(badge).toHaveAttribute('data-state', state)
    expect(badge).toHaveAccessibleName(badge.textContent ?? '')
    expect(badge).toHaveTextContent(badge.textContent ?? '')
    expect(badge).toHaveClass('min-w-16', 'justify-center', 'whitespace-nowrap')
  })

  it('preserves semantic controls and associations across primitives', () => {
    render(
      <>
        <Heading level={3}>Primitive heading</Heading>
        <Field id="test-field" label="Test field" description="Helpful text" error="Required value" required />
        <Button>Continue</Button>
        <Link href="/records">Records</Link>
        <Panel title="Test panel">Panel content</Panel>
        <DataTable
          caption="Test table"
          columns={[{ key: 'name', header: 'Name', render: (row: { name: string }) => row.name }]}
          rows={[{ name: 'Example' }]}
          getRowKey={(row) => row.name}
        />
        <ConfirmationSurface
          title="Confirm test"
          description="Confirm this presentation example."
          confirmLabel="Confirm"
          cancelLabel="Cancel"
          onConfirm={() => undefined}
          onCancel={() => undefined}
        />
      </>,
    )

    expect(screen.getByRole('heading', { name: 'Primitive heading', level: 3 })).toBeInTheDocument()
    expect(screen.getByRole('textbox', { name: /Test field/ })).toHaveAttribute('aria-invalid', 'true')
    expect(screen.getByRole('textbox', { name: /Test field/ })).toHaveAttribute(
      'aria-describedby',
      'test-field-description test-field-error',
    )
    expect(screen.getByRole('textbox', { name: /Test field/ })).toHaveAttribute(
      'aria-errormessage',
      'test-field-error',
    )
    expect(screen.getByRole('link', { name: 'Records' })).toHaveAttribute('href', '/records')
    expect(screen.getByRole('table', { name: 'Test table' })).toBeInTheDocument()
    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument()
    expect(screen.getByRole('region', { name: 'Confirm test' })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue' }))
      .toHaveAttribute('type', 'button')
    expect(screen.getByRole('button', { name: 'Continue' })).toHaveClass('min-h-11')
    expect(screen.getByRole('region', { name: 'Confirm test' })).toHaveAttribute(
      'aria-describedby',
    )
  })

  it('only exposes live semantics when a status badge should be announced', () => {
    const { container } = render(
      <>
        <StatusBadge state="info" />
        <StatusBadge state="success" announce />
      </>,
    )

    expect(container.querySelectorAll('[role="status"]')).toHaveLength(1)
  })

  it('keeps generated region IDs unique for repeated titled surfaces', () => {
    render(
      <>
        <Panel title="Repeated panel">First panel</Panel>
        <Panel title="Repeated panel">Second panel</Panel>
        <ConfirmationSurface
          title="Repeated confirmation"
          description="First confirmation."
          confirmLabel="Confirm first"
          cancelLabel="Cancel first"
          onConfirm={() => undefined}
          onCancel={() => undefined}
        />
        <ConfirmationSurface
          title="Repeated confirmation"
          description="Second confirmation."
          confirmLabel="Confirm second"
          cancelLabel="Cancel second"
          onConfirm={() => undefined}
          onCancel={() => undefined}
        />
      </>,
    )

    const labelledRegions = screen.getAllByRole('region')
    const labelledBy = labelledRegions.map((region) => region.getAttribute('aria-labelledby'))
    expect(new Set(labelledBy).size).toBe(labelledBy.length)
  })
})
