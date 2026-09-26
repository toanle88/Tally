import { useId, useState } from 'react'

import { Button } from '@/components/ui'
import type { SensitiveAccess, SensitiveAccessAuditEvent, SensitiveAction } from '@/components/record-context'

export interface SensitiveDataGuardProps {
  label: string
  classification: string
  value?: string
  access: SensitiveAccess
  canReveal: boolean
  canExport: boolean
  actorReference: string
  targetReference: string
  scopeReference: string
  purpose: string
  decisionReference: string
  policyVersion: string
  onAccess: (event: SensitiveAccessAuditEvent) => boolean
  onExport?: () => void
  onAccessDenied: (action: SensitiveAction) => void
}
export function SensitiveDataGuard({ label, classification, value, access, canReveal, canExport, actorReference, targetReference, scopeReference, purpose, decisionReference, policyVersion, onAccess, onExport, onAccessDenied }: SensitiveDataGuardProps) {
  const sectionId = `${useId()}-sensitive-data`
  const [revealed, setRevealed] = useState(false)
  const [accessMessage, setAccessMessage] = useState('')
  const revealAllowed = access === 'authorized' && canReveal && value !== undefined
  const exportAllowed = access === 'authorized' && canExport && value !== undefined && onExport !== undefined
  const maskedValue = value === undefined ? 'No value available' : '••••••••'

  const recordAccess = (action: SensitiveAction, outcome: SensitiveAccessAuditEvent['outcome']) => {
    try {
      return onAccess({ action, outcome, actorReference, targetReference, scopeReference, purpose, classification, decisionReference, policyVersion })
    } catch {
      return false
    }
  }

  const handleDenied = (action: SensitiveAction) => {
    const recorded = recordAccess(action, 'denied')
    onAccessDenied(action)
    if (!recorded) setAccessMessage('Access evidence unavailable; the protected value remains masked.')
  }

  const handleReveal = () => {
    if (!revealAllowed) return handleDenied('reveal')
    if (revealed) return setRevealed(false)
    if (recordAccess('reveal', 'allowed')) setRevealed(true)
    else setAccessMessage('Access evidence unavailable; the protected value remains masked.')
  }

  const handleExport = () => {
    if (!exportAllowed) return handleDenied('export')
    if (recordAccess('export', 'allowed')) onExport()
    else setAccessMessage('Access evidence unavailable; export remains unavailable.')
  }

  return (
    <section aria-labelledby={sectionId} className="rounded-box border border-warning/40 bg-base-100 p-4">
      <h3 id={sectionId} className="font-semibold">{label}</h3>
      <p className="mt-1 text-sm text-base-content/70">Classification: {classification}</p>
      <p role="status" className="mt-3 font-mono">{access === 'restricted' ? maskedValue : revealed && value !== undefined ? value : maskedValue}</p>
      {value === undefined ? <p className="mt-2 text-sm text-base-content/70">No value available.</p> : <div className="mt-3 flex flex-wrap gap-2">
        <Button variant="ghost" aria-disabled={!revealAllowed} onClick={handleReveal}>{revealAllowed && revealed ? 'Hide value' : revealAllowed ? 'Reveal value' : 'Reveal unavailable'}</Button>
        <Button variant="ghost" aria-disabled={!exportAllowed} onClick={handleExport}>{exportAllowed ? 'Export value' : 'Export unavailable'}</Button>
      </div>}
      {access === 'restricted' ? <p className="mt-3 text-sm text-base-content/75">Access restricted. The protected value remains masked and export is unavailable.</p> : null}
      {accessMessage ? <p role="status" className="mt-3 text-sm text-base-content/75">{accessMessage}</p> : null}
    </section>
  )
}
