import { Panel, StatusBadge } from '@/components/ui'
import type { RecordIdentity } from '@/components/record-context'

export interface RecordIdentityHeaderProps {
  identity: RecordIdentity
}
export function RecordIdentityHeader({ identity }: RecordIdentityHeaderProps) {
  const sensitivityState = identity.sensitivity.classification === 'standard' ? 'info' : 'restricted'

  return (
    <Panel title="Record identity" description="Authoritative record context and ownership.">
      <dl className="mt-4 grid gap-3 sm:grid-cols-2">
        <IdentityValue label="Record" value={`${identity.recordType} — ${identity.recordId}`} />
        <IdentityValue label="Authoritative capability" value={identity.capability} />
        <IdentityValue label="Scope" value={identity.scope.label} />
        <IdentityValue label="Version" value={`v${identity.version}`} />
        <IdentityValue label="Source" value={`${identity.source.label} (${identity.source.reference})`} />
        <IdentityValue label="Owner" value={`${identity.owner.label}${identity.owner.reference ? ` (${identity.owner.reference})` : ''}`} />
        <div>
          <dt className="text-sm font-medium text-base-content/70">Lifecycle state</dt>
          <dd className="mt-1"><StatusBadge state={identity.lifecycleSemanticState} label={identity.lifecycleState} /></dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-base-content/70">Sensitivity</dt>
          <dd className="mt-1"><StatusBadge state={sensitivityState} label={identity.sensitivity.label} /></dd>
        </div>
        <div>
          <dt className="text-sm font-medium text-base-content/70">Last material change</dt>
          <dd className="mt-1"><time dateTime={identity.lastMaterialChange}>{identity.lastMaterialChange}</time></dd>
        </div>
      </dl>
    </Panel>
  )
}

function IdentityValue({ label, value }: { label: string; value: string }) {
  return <div><dt className="text-sm font-medium text-base-content/70">{label}</dt><dd className="mt-1 break-words">{value}</dd></div>
}
