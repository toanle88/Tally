import { useState } from 'react'

import { CorrectionLineagePanel } from '@/components/correction-lineage-panel'
import { EvidenceDrawer } from '@/components/evidence-drawer'
import { LegalHoldIndicator } from '@/components/legal-hold-indicator'
import { LifecycleTimeline } from '@/components/lifecycle-timeline'
import { MoneyAndCurrencyPanel } from '@/components/money-and-currency-panel'
import { RecordIdentityHeader } from '@/components/record-identity-header'
import { SensitiveDataGuard } from '@/components/sensitive-data-guard'
import { Heading, Panel } from '@/components/ui'

import { evidenceFixture, lifecycleFixture, lineageReferences, moneyFixture, recordIdentityFixture } from './record-detail-fixtures'

export function RecordDetailExample() {
  const [evidenceOpen, setEvidenceOpen] = useState(false)
  const [privacyMessage, setPrivacyMessage] = useState('No restricted-data action has been attempted.')

  return (
    <section id="record-detail-context-example" aria-label="Record detail context example" className="space-y-6">
      <div>
        <Heading level={2}>Record detail context example</Heading>
        <p className="mt-2 max-w-3xl text-base-content/75">Synthetic presentation-only detail. It demonstrates immutable lineage, amount meaning, evidence access, privacy guards, and legal hold state without establishing a financial fact.</p>
      </div>

      <RecordIdentityHeader identity={recordIdentityFixture} />
      <div className="grid items-start gap-6 xl:grid-cols-2"><MoneyAndCurrencyPanel amounts={moneyFixture} gainLoss={{ amount: '-25.00', currency: 'USD', signConvention: 'Negative means loss; positive means gain.' }} /><LegalHoldIndicator status="active" holdReference="HOLD-FIX-00042" appliedAt="2026-08-30T09:35:00Z" destructionAction={{ label: 'Destroy retained content', onActivate: () => setPrivacyMessage('Fixture destruction action accepted after hold release.') }} /></div>
      <LifecycleTimeline transitions={lifecycleFixture} />
      <CorrectionLineagePanel original={{ recordId: recordIdentityFixture.recordId, label: 'Original established customer receipt', href: '#original', establishedAt: recordIdentityFixture.lastMaterialChange }} corrections={lineageReferences} />
      <Panel title="Evidence access" description="The drawer contains only fixture-supplied references and access decisions."><EvidenceDrawer open={evidenceOpen} evidence={evidenceFixture} onOpenChange={setEvidenceOpen} /></Panel>
      <Panel title="Sensitive data examples" description="Restricted values remain masked; authorized reveal and export require explicit fixture permission.">
        <div className="mt-4 grid gap-4 xl:grid-cols-2">
          <SensitiveDataGuard label="Restricted provider reference" classification="Bank-sensitive" value="SYNTHETIC-RESTRICTED-REF" access="restricted" canReveal={true} canExport={true} onAccessDenied={(action) => setPrivacyMessage(`Denied ${action} access to the restricted fixture value.`)} />
          <SensitiveDataGuard label="Authorized detail reference" classification="Synthetic personal reference" value="SYNTHETIC-AUTHORIZED-REF" access="authorized" canReveal={true} canExport={true} onExport={() => setPrivacyMessage('Fixture export permitted; no data was downloaded.')} onAccessDenied={(action) => setPrivacyMessage(`Denied ${action} access to the authorized fixture value.`)} />
        </div>
        <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{privacyMessage}</p>
      </Panel>
    </section>
  )
}
