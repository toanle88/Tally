import { useId, useState } from 'react'

import { Button } from '@/components/ui'
import type { SensitiveAccess, SensitiveAction } from '@/components/record-context'

export interface SensitiveDataGuardProps {
  label: string
  classification: string
  value?: string
  access: SensitiveAccess
  canReveal: boolean
  canExport: boolean
  onExport?: () => void
  onAccessDenied: (action: SensitiveAction) => void
}
export function SensitiveDataGuard({ label, classification, value, access, canReveal, canExport, onExport, onAccessDenied }: SensitiveDataGuardProps) {
  const sectionId = `${useId()}-sensitive-data`
  const [revealed, setRevealed] = useState(false)
  const revealAllowed = access === 'authorized' && canReveal && value !== undefined
  const exportAllowed = access === 'authorized' && canExport && value !== undefined && onExport !== undefined
  const maskedValue = value === undefined ? 'No value available' : '••••••••'

  const handleDenied = (action: SensitiveAction) => onAccessDenied(action)

  return (
    <section aria-labelledby={sectionId} className="rounded-box border border-warning/40 bg-base-100 p-4">
      <h3 id={sectionId} className="font-semibold">{label}</h3>
      <p className="mt-1 text-sm text-base-content/70">Classification: {classification}</p>
      <p role="status" className="mt-3 font-mono">{access === 'restricted' ? maskedValue : revealed && value !== undefined ? value : maskedValue}</p>
      {value === undefined ? <p className="mt-2 text-sm text-base-content/70">No value available.</p> : <div className="mt-3 flex flex-wrap gap-2">
        <Button variant="ghost" aria-disabled={!revealAllowed} onClick={() => revealAllowed ? setRevealed((current) => !current) : handleDenied('reveal')}>{revealAllowed && revealed ? 'Hide value' : revealAllowed ? 'Reveal value' : 'Reveal unavailable'}</Button>
        <Button variant="ghost" aria-disabled={!exportAllowed} onClick={() => exportAllowed ? onExport() : handleDenied('export')}>{exportAllowed ? 'Export value' : 'Export unavailable'}</Button>
      </div>}
      {access === 'restricted' ? <p className="mt-3 text-sm text-warning">Access restricted. The protected value remains masked and export is unavailable.</p> : null}
    </section>
  )
}
