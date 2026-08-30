import { Link, Panel } from '@/components/ui'
import type { LineageReference } from '@/components/record-context'

export interface OriginalEstablishedFact {
  recordId: string
  label: string
  href: string
  establishedAt: string
}
export interface CorrectionLineagePanelProps {
  original: OriginalEstablishedFact
  corrections: readonly LineageReference[]
}

export function CorrectionLineagePanel({ original, corrections }: CorrectionLineagePanelProps) {
  return (
    <Panel title="Correction lineage" description="Established facts remain visible while corrections are linked as new records.">
      <div className="mt-4 space-y-4">
        <div className="rounded-box border border-primary/30 bg-primary/5 p-3"><p className="text-sm font-medium">Original established fact</p><Link href={original.href}>{original.label} ({original.recordId})</Link><time className="mt-1 block text-sm text-base-content/70" dateTime={original.establishedAt}>Established {original.establishedAt}</time></div>
        {corrections.length === 0 ? <p className="text-sm text-base-content/70">No linked corrections or replacements.</p> : <ul className="space-y-2">{corrections.map((correction) => <li key={correction.recordId} className="rounded-box border border-base-300 p-3"><span className="mr-2 text-sm font-medium capitalize">{correction.kind}</span><Link href={correction.href}>{correction.label} ({correction.recordId})</Link>{correction.state ? <span className="ml-2 text-sm text-base-content/70">— {correction.state}</span> : null}</li>)}</ul>}
      </div>
    </Panel>
  )
}
