import { useEffect, useId, useRef } from 'react'

import { Button, Link, StatusBadge } from '@/components/ui'
import type { VersionConflictSnapshot } from '@/components/workflow-context'

export function VersionConflictDialog({ conflict, onClose, onRetry, onRefresh, onNewIdentity }: { conflict: VersionConflictSnapshot; onClose: () => void; onRetry?: () => void; onRefresh?: () => void; onNewIdentity?: () => void }) {
  const dialogRef = useRef<HTMLDivElement>(null)
  const onCloseRef = useRef(onClose)
  const previousFocusRef = useRef<HTMLElement | null>(null)
  const titleId = `${useId()}-conflict-title`
  const descriptionId = `${useId()}-conflict-description`

  useEffect(() => {
    onCloseRef.current = onClose
  }, [onClose])

  useEffect(() => {
    const dialog = dialogRef.current
    previousFocusRef.current = document.activeElement instanceof HTMLElement
      ? document.activeElement
      : null
    dialog?.querySelector<HTMLButtonElement>('button:not([disabled])')?.focus()

    const handler = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault()
        onCloseRef.current()
        return
      }
      if (event.key !== 'Tab' || !dialog) return
      const focusable = Array.from(dialog.querySelectorAll<HTMLElement>(
        'button:not([disabled]), a[href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
      )).filter((element) => !element.hasAttribute('hidden') && element.getAttribute('aria-hidden') !== 'true')
      if (focusable.length === 0) return
      const first = focusable[0]
      const last = focusable[focusable.length - 1]
      if (event.shiftKey && document.activeElement === first) { event.preventDefault(); last.focus() }
      else if (!event.shiftKey && document.activeElement === last) { event.preventDefault(); first.focus() }
    }

    dialog?.addEventListener('keydown', handler)
    return () => {
      dialog?.removeEventListener('keydown', handler)
      if (previousFocusRef.current?.isConnected) previousFocusRef.current.focus()
    }
  }, [])

  return <div className="fixed inset-0 z-50 grid place-items-center bg-neutral/50 p-4" role="presentation" onMouseDown={(event) => { if (event.target === event.currentTarget) onCloseRef.current() }}>
    <div ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby={titleId} aria-describedby={descriptionId} className="max-h-[90vh] w-full max-w-3xl overflow-y-auto rounded-box bg-base-100 p-6 shadow-xl">
      <div className="flex items-start justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-wide text-base-content/75">CON-SCR-01</p><h2 id={titleId} className="mt-1 text-xl font-semibold">The record changed before this action</h2></div><Button variant="ghost" size="sm" onClick={() => onCloseRef.current()}>Close</Button></div>
      <p id={descriptionId} className="mt-3 text-sm text-base-content/75">Review the authoritative current state before deciding whether to retry. No automatic retry was performed.</p>
      <div className="mt-4 grid gap-3 sm:grid-cols-2 text-sm"><div><strong>Expected</strong><p>v{conflict.expected.version} · {conflict.expected.state}</p></div><div><strong>Current</strong><p>v{conflict.current.version} · {conflict.current.state}</p></div><div><strong>Current owner</strong><p>{conflict.currentOwner.label}</p></div><div><strong>Process epoch</strong><p>{conflict.processEpoch}</p></div></div>
      <div className="mt-4"><h3 className="font-medium">Changed values</h3><ul className="mt-2 space-y-2 text-sm">{conflict.changedValues.map((change) => <li key={change.id} className="rounded-box border border-base-300 p-3"><strong>{change.label}</strong><p>Reviewed: {change.reviewedValue}</p><p>Current: {change.currentValue}</p></li>)}</ul></div>
      {conflict.establishedResult ? <p className="mt-4 text-sm">Established result: <Link href={conflict.establishedResult.href}>{conflict.establishedResult.label}</Link></p> : null}
      <div className="mt-5 flex flex-wrap gap-2"><Button variant="secondary" onClick={onRefresh}>Refresh authoritative state</Button>{conflict.safeRetry.permitted ? <Button onClick={onRetry}>Retry after review</Button> : <StatusBadge state="disabled" label="Retry unavailable" />}{conflict.requiredNewBusinessIdentity ? <Button variant="secondary" onClick={onNewIdentity}>Start new business identity</Button> : null}</div>
      <p className="mt-3 text-sm text-base-content/70">{conflict.safeRetry.reason}{conflict.kind === 'identity-content' ? ' Identity-content conflicts require a new business identity.' : ''}</p>
    </div>
  </div>
}
