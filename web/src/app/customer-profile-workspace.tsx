import { useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'

export type CustomerProfileFixture = {
  id: string
  scopeId: string
  partyId: string
  partyName: string
  partyVersion: number
  creditTerms: string
  creditLimit: { amount: string; currency: string }
  billingPreference: string
  taxTreatment: string
  status: 'draft' | 'active' | 'end_dated'
  effectiveFrom: string
  effectiveTo?: string
  approvalStatus: 'not-required' | 'approved' | 'pending'
  version: number
  revisionNumber: number
  nextAction: string
}

export const initialCustomerProfiles: CustomerProfileFixture[] = [
  {
    id: 'customer-profile-northwind', scopeId: 'scope-singapore-management', partyId: 'party-northwind-customer', partyName: 'Northwind Distribution', partyVersion: 2,
    creditTerms: 'net_30', creditLimit: { amount: '250000', currency: 'SGD' }, billingPreference: 'invoice', taxTreatment: 'standard', status: 'active', effectiveFrom: '2026-01-01', approvalStatus: 'approved', version: 3, revisionNumber: 3, nextAction: 'maintain',
  },
  {
    id: 'customer-profile-acme', scopeId: 'scope-vietnam-statutory', partyId: 'party-acme-customer', partyName: 'Acme Retail Customer', partyVersion: 4,
    creditTerms: 'prepaid', creditLimit: { amount: '0', currency: 'VND' }, billingPreference: 'cash', taxTreatment: 'exempt', status: 'draft', effectiveFrom: '2026-02-01', approvalStatus: 'pending', version: 1, revisionNumber: 1, nextAction: 'submit for approval',
  },
]

export const profileStatusState: Record<CustomerProfileFixture['status'], SemanticState> = { draft: 'pending', active: 'success', end_dated: 'disabled' }

export function CustomerProfileRecord() {
  const [params] = useSearchParams()
  const selected = initialCustomerProfiles.find((profile) => profile.id === params.get('customerProfileId')) ?? initialCustomerProfiles[0]
  const [profile, setProfile] = useState(selected)
  const [creditTerms, setCreditTerms] = useState(selected.creditTerms)
  const [creditLimit, setCreditLimit] = useState(selected.creditLimit.amount)
  const [billingPreference, setBillingPreference] = useState(selected.billingPreference)
  const [taxTreatment, setTaxTreatment] = useState(selected.taxTreatment)
  const [notice, setNotice] = useState('Review the authoritative Party and profile versions before making a material change.')

  const save = () => {
    const nextVersion = profile.version + 1
    setProfile((current) => ({ ...current, creditTerms: creditTerms.trim().toLowerCase(), creditLimit: { amount: creditLimit.trim(), currency: current.creditLimit.currency }, billingPreference: billingPreference.trim().toLowerCase(), taxTreatment: taxTreatment.trim().toLowerCase(), version: nextVersion, revisionNumber: current.revisionNumber + 1, nextAction: 'maintain' }))
    setNotice(`Accepted customer-profile revision v${nextVersion}. The authoritative Party reference remains at v${profile.partyVersion}; audit evidence was recorded.`)
  }

  return <section aria-labelledby="omd-customer-profile-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-SCR-03</p><h2 id="omd-customer-profile-title" className="mt-1 text-2xl font-semibold">Customer profile record</h2><p className="mt-2 max-w-3xl text-base-content/75">The authoritative mutation surface for customer terms, exact credit limits, billing preference, tax treatment, and effective-dated profile history.</p></div>
      <RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/master-data/omd-ws-01">Back to worklist</RouterLink>
    </div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <Panel title="Authoritative reference and lifecycle" description="Changes use optimistic concurrency, Party-version revalidation, and create an immutable profile revision. The API is authoritative for durable state.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="customer-profile-party" label="Authoritative Party" value={profile.partyName} readOnly /><Field id="customer-profile-party-id" label="Party identifier" value={profile.partyId} readOnly /><Field id="customer-profile-party-version" label="Party version" value={`v${profile.partyVersion}`} readOnly /><Field id="customer-profile-id" label="Profile identifier" value={profile.id} readOnly /><Field id="customer-profile-version" label="Profile version" value={`v${profile.version}`} readOnly /><Field id="customer-profile-effective-from" label="Effective from" value={profile.effectiveFrom} readOnly /><Field id="customer-profile-effective-to" label="Effective to" value={profile.effectiveTo ?? 'open'} readOnly /></div>
      <div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={profileStatusState[profile.status]} label={profile.status} announce /><span className="text-sm text-base-content/70">Approval: {profile.approvalStatus} · Next action: {profile.nextAction} · Revision {profile.revisionNumber}</span></div>
    </Panel>
    <Panel title="Customer terms and controls" description="Values are shown from the field-filtered safe projection. Credit limits remain exact decimal strings with an explicit currency.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="customer-profile-credit-terms" label="Credit terms" value={creditTerms} onChange={(event) => setCreditTerms(event.target.value)} disabled={profile.status === 'end_dated'} /><Field id="customer-profile-credit-limit" label={`Credit limit (${profile.creditLimit.currency})`} value={creditLimit} onChange={(event) => setCreditLimit(event.target.value)} disabled={profile.status === 'end_dated'} /><Field id="customer-profile-billing-preference" label="Billing preference" value={billingPreference} onChange={(event) => setBillingPreference(event.target.value)} disabled={profile.status === 'end_dated'} /><Field id="customer-profile-tax-treatment" label="Tax treatment" value={taxTreatment} onChange={(event) => setTaxTreatment(event.target.value)} disabled={profile.status === 'end_dated'} /></div>
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={save} disabled={profile.status === 'end_dated'}>Save customer-profile revision</Button></div>
    </Panel>
    <Panel title="History and downstream boundary" description="Established profile revisions remain available for historical references. This screen does not create invoices, receivables, credit state, or collection state."><p className="text-sm text-base-content/75">Current profile revision: v{profile.version}. Historical reference versions are retained when the profile changes; downstream contexts consume identifiers and approved snapshots through their own integration boundary.</p></Panel>
  </section>
}
