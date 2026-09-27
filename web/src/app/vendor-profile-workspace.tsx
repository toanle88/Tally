import { useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'

export type VendorProfileFixture = {
  id: string
  scopeId: string
  partyId: string
  partyName: string
  partyVersion: number
  paymentTerms: string
  withholdingTreatment: string
  remittancePreference: string
  status: 'draft' | 'active' | 'end_dated'
  effectiveFrom: string
  effectiveTo?: string
  approvalStatus: 'not-required' | 'approved' | 'pending'
  version: number
  revisionNumber: number
  nextAction: string
  partyBankControl: 'approved' | 'pending' | 'unavailable'
}

export const initialVendorProfiles: VendorProfileFixture[] = [
  {
    id: 'vendor-profile-acme-industrial', scopeId: 'scope-vietnam-statutory', partyId: 'party-acme-vendor', partyName: 'Acme Industrial Supplies', partyVersion: 5,
    paymentTerms: 'net_30', withholdingTreatment: 'standard', remittancePreference: 'bank_transfer', status: 'active', effectiveFrom: '2026-01-01', approvalStatus: 'approved', version: 4, revisionNumber: 4, nextAction: 'maintain', partyBankControl: 'approved',
  },
  {
    id: 'vendor-profile-northwind', scopeId: 'scope-singapore-management', partyId: 'party-northwind-vendor', partyName: 'Northwind Components', partyVersion: 2,
    paymentTerms: 'net_45', withholdingTreatment: 'reduced', remittancePreference: 'manual_review', status: 'draft', effectiveFrom: '2026-02-01', approvalStatus: 'pending', version: 1, revisionNumber: 1, nextAction: 'submit for approval', partyBankControl: 'pending',
  },
]

export const vendorProfileStatusState: Record<VendorProfileFixture['status'], SemanticState> = { draft: 'pending', active: 'success', end_dated: 'disabled' }

export function VendorProfileRecord() {
  const [params] = useSearchParams()
  const selected = initialVendorProfiles.find((profile) => profile.id === params.get('vendorProfileId')) ?? initialVendorProfiles[0]
  const [profile, setProfile] = useState(selected)
  const [paymentTerms, setPaymentTerms] = useState(selected.paymentTerms)
  const [withholdingTreatment, setWithholdingTreatment] = useState(selected.withholdingTreatment)
  const [remittancePreference, setRemittancePreference] = useState(selected.remittancePreference)
  const [effectiveTo, setEffectiveTo] = useState(selected.effectiveTo ?? '')
  const [notice, setNotice] = useState('Review the authoritative Party and profile versions before making a material change.')

  const save = () => {
    const nextVersion = profile.version + 1
    const nextEffectiveTo = effectiveTo.trim() || undefined
    const nextStatus = nextEffectiveTo ? 'end_dated' : profile.status === 'draft' ? 'active' : profile.status
    setProfile((current) => ({ ...current, paymentTerms: paymentTerms.trim().toLowerCase(), withholdingTreatment: withholdingTreatment.trim().toLowerCase(), remittancePreference: remittancePreference.trim().toLowerCase(), effectiveTo: nextEffectiveTo, status: nextStatus, version: nextVersion, revisionNumber: current.revisionNumber + 1, nextAction: nextStatus === 'end_dated' ? 'view history' : 'maintain' }))
    setNotice(`Accepted vendor-profile revision v${nextVersion}. The authoritative Party reference remains at v${profile.partyVersion}; audit evidence was recorded.`)
  }

  return <section aria-labelledby="omd-vendor-profile-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-SCR-03</p><h2 id="omd-vendor-profile-title" className="mt-1 text-2xl font-semibold">Vendor profile record</h2><p className="mt-2 max-w-3xl text-base-content/75">The authoritative mutation surface for vendor payment terms, withholding treatment, remittance preference, and effective-dated profile history.</p></div>
      <RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/master-data/omd-ws-01">Back to worklist</RouterLink>
    </div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <Panel title="Authoritative reference and lifecycle" description="Changes use optimistic concurrency, Party-version revalidation, and create an immutable profile revision. The API is authoritative for durable state.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="vendor-profile-party" label="Authoritative Party" value={profile.partyName} readOnly /><Field id="vendor-profile-party-id" label="Party identifier" value={profile.partyId} readOnly /><Field id="vendor-profile-party-version" label="Party version" value={`v${profile.partyVersion}`} readOnly /><Field id="vendor-profile-id" label="Profile identifier" value={profile.id} readOnly /><Field id="vendor-profile-version" label="Profile version" value={`v${profile.version}`} readOnly /><Field id="vendor-profile-effective-from" label="Effective from" value={profile.effectiveFrom} readOnly /><Field id="vendor-profile-effective-to" label="Effective to" type="date" value={effectiveTo} onChange={(event) => setEffectiveTo(event.target.value)} disabled={profile.status === 'end_dated'} /></div>
      <div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={vendorProfileStatusState[profile.status]} label={profile.status} announce /><span className="text-sm text-base-content/70">Approval: {profile.approvalStatus} · Next action: {profile.nextAction} · Revision {profile.revisionNumber}</span></div>
    </Panel>
    <Panel title="Vendor settlement policy" description="Values are shown from the field-filtered safe projection. Bank details remain Party-owned and are not copied into this profile.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="vendor-profile-payment-terms" label="Payment terms" value={paymentTerms} onChange={(event) => setPaymentTerms(event.target.value)} disabled={profile.status === 'end_dated'} /><Field id="vendor-profile-withholding-treatment" label="Withholding treatment" value={withholdingTreatment} onChange={(event) => setWithholdingTreatment(event.target.value)} disabled={profile.status === 'end_dated'} /><Field id="vendor-profile-remittance-preference" label="Remittance preference" value={remittancePreference} onChange={(event) => setRemittancePreference(event.target.value)} disabled={profile.status === 'end_dated'} /></div>
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={save} disabled={profile.status === 'end_dated'}>Save vendor-profile revision</Button></div>
    </Panel>
    <Panel title="Party-owned remittance boundary" description="Only the authorized Party projection supplies bank-control state. Raw account numbers, credentials, provider tokens, and bank details are never exposed here."><div className="flex flex-wrap items-center gap-3"><StatusBadge state={profile.partyBankControl === 'approved' ? 'success' : profile.partyBankControl === 'pending' ? 'pending' : 'disabled'} label={`Party bank control: ${profile.partyBankControl}`} announce /><span className="text-sm text-base-content/70">Remittance preference: {profile.remittancePreference}. Payment execution and payment-instruction actions are outside this maintenance screen.</span></div></Panel>
    <Panel title="History and downstream boundary" description="Established profile revisions remain available for historical references. This screen does not create invoices, liabilities, payment requests, or payment instructions."><p className="text-sm text-base-content/75">Current profile revision: v{profile.version}. Historical Party and profile references remain versioned when the profile changes; downstream contexts consume identifiers and approved snapshots through their own integration boundary.</p></Panel>
  </section>
}
