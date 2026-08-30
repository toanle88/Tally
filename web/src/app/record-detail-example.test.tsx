import { fireEvent, render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'

import { RecordDetailExample } from './record-detail-example'

describe('record detail example', () => {
  it('combines identity, lifecycle, money, lineage, evidence, privacy, and legal hold surfaces', () => {
    render(<RecordDetailExample />)

    expect(screen.getByRole('heading', { name: 'Record detail context example', level: 2 })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Record identity', level: 2 })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Lifecycle timeline', level: 2 })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Money and currency', level: 2 })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Correction lineage', level: 2 })).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Legal hold', level: 2 })).toBeInTheDocument()
    expect(screen.queryByText('SYNTHETIC-RESTRICTED-REF')).not.toBeInTheDocument()

    fireEvent.click(screen.getByRole('button', { name: 'Open evidence drawer' }))
    expect(screen.getByRole('link', { name: /Source receipt observation/ })).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Reveal value' }))
    expect(screen.getByText('SYNTHETIC-AUTHORIZED-REF')).toBeInTheDocument()
  })
})
