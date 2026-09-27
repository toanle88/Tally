import { useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { SensitiveDataGuard } from '@/components/sensitive-data-guard'
import { Button, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'

export type PartyFixture = {
  id: string
  scopeId: string
  name: string
  partyType: string
  status: string
  taxIdentifierMasked?: string
  version: number
  revisionNumber: number
  contactMethods: Array<{ type: string; valueMasked: string }>
  addresses: Array<{ type: string; line1: string; locality: string; region?: string; postalCode: string; countryCode: string }>
  classifications: Array<{ code: string; value: string }>
  bankDetailReferences: Array<{ reference: string; providerCode: string; status: 'approved' | 'pending'; coolingOffUntil?: string }>
}

export const initialParties: PartyFixture[] = [
  {
    id: 'party-acme-vendor', scopeId: 'scope-vietnam-statutory', name: 'Acme Industrial Supplies', partyType: 'vendor', status: 'active', taxIdentifierMasked: '••••4321', version: 5, revisionNumber: 5,
    contactMethods: [{ type: 'email', valueMasked: '••••@acme-industrial.example' }, { type: 'phone', valueMasked: '••••••••' }],
    addresses: [{ type: 'registered', line1: '4 Commerce Avenue', locality: 'Ho Chi Minh City', postalCode: '700000', countryCode: 'VN' }],
    classifications: [{ code: 'segment', value: 'supplier' }, { code: 'risk', value: 'standard' }],
    bankDetailReferences: [{ reference: 'provider-ref-001', providerCode: 'provider-a', status: 'approved' }],
  },
  {
    id: 'party-northwind-customer', scopeId: 'scope-singapore-management', name: 'Northwind Distribution', partyType: 'customer', status: 'pending', version: 2, revisionNumber: 2,
    contactMethods: [{ type: 'email', valueMasked: '••••@northwind.example' }],
    addresses: [{ type: 'billing', line1: '8 Harbour Road', locality: 'Singapore', postalCode: '018956', countryCode: 'SG' }],
    classifications: [{ code: 'segment', value: 'customer' }],
    bankDetailReferences: [{ reference: 'provider-ref-002', providerCode: 'provider-b', status: 'pending', coolingOffUntil: '2026-10-04' }],
  },
]

const partyStatusState: Record<string, SemanticState> = { active: 'success', pending: 'pending', inactive: 'disabled' }

export function PartyRecord() {
  const [params] = useSearchParams()
  const selected = initialParties.find((party) => party.id === params.get('partyId')) ?? initialParties[0]
  const [party, setParty] = useState(selected)
  const [name, setName] = useState(selected.name)
  const [status, setStatus] = useState(selected.status)
  const [revisionHistory, setRevisionHistory] = useState([selected.version])
  const [notice, setNotice] = useState('Review the authoritative version before making a material party change.')

  const save = () => {
    const nextVersion = party.version + 1
    setParty((current) => ({ ...current, name: name.trim(), status: status.trim().toLowerCase(), version: nextVersion, revisionNumber: current.revisionNumber + 1 }))
    setRevisionHistory((current) => [...current, nextVersion])
    setNotice(`Accepted fixture revision v${nextVersion}. The API remains authoritative for the durable mutation and audit evidence.`)
  }

  const recordAccess = () => {
    setNotice('Restricted-field access evidence was recorded; protected values remain masked in this fixture.')
    return true
  }

  return <section aria-labelledby="omd-party-record-title" className="space-y-6">
    <div className="flex flex-wrap items-start justify-between gap-4">
      <div><p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-SCR-02</p><h2 id="omd-party-record-title" className="mt-1 text-2xl font-semibold">Party record</h2><p className="mt-2 max-w-3xl text-base-content/75">Fixture-backed maintenance for identity, status, contacts, addresses, classifications, and safe bank-control references. The API is authoritative.</p></div>
      <RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/master-data/omd-ws-01">Back to worklist</RouterLink>
    </div>
    <p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p>
    <Panel title="Identity and lifecycle" description="A save creates a local fixture revision and represents the API command boundary using optimistic concurrency.">
      <div className="grid gap-4 md:grid-cols-2"><Field id="party-name" label="Party name" value={name} onChange={(event) => setName(event.target.value)} /><Field id="party-id" label="Record identifier" value={party.id} readOnly /><Field id="party-type" label="Party type" value={party.partyType} readOnly /><Field id="party-status" label="Status code" value={status} onChange={(event) => setStatus(event.target.value)} /><Field id="party-version" label="Aggregate version" value={`v${party.version}`} readOnly /><Field id="party-revision" label="Revision number" value={String(party.revisionNumber)} readOnly /></div>
      <div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={partyStatusState[party.status] ?? 'pending'} label={party.status} announce /><span className="text-sm text-base-content/70">Party type: {party.partyType} · Scope: {party.scopeId}</span></div>
      <div className="mt-5 flex flex-wrap gap-3"><Button onClick={save}>Save party fixture revision</Button></div>
      <p className="mt-4 text-sm text-base-content/70">Local revision history: {revisionHistory.map((version) => `v${version}`).join(' → ')}</p>
    </Panel>
    <div className="grid gap-6 lg:grid-cols-2">
      <Panel title="Contact methods" description="Personal contact values are masked in the safe fixture projection."><ul className="space-y-3">{party.contactMethods.map((contact) => <li key={contact.type} className="rounded-box border border-base-300 p-3"><p className="font-semibold">{contact.type}</p><p className="mt-1 font-mono text-sm">{contact.valueMasked}</p></li>)}</ul></Panel>
      <Panel title="Classifications" description="Classification codes remain open canonical values owned by OMD."><ul className="space-y-3">{party.classifications.map((classification) => <li key={classification.code} className="flex justify-between rounded-box border border-base-300 p-3"><span>{classification.code}</span><span className="font-mono">{classification.value}</span></li>)}</ul></Panel>
    </div>
    <Panel title="Addresses"><ul className="grid gap-3 md:grid-cols-2">{party.addresses.map((address) => <li key={address.type} className="rounded-box border border-base-300 p-3"><p className="font-semibold">{address.type}</p><p className="mt-1">{address.line1}, {address.locality}</p><p className="text-sm text-base-content/70">{address.region ? `${address.region}, ` : ''}{address.postalCode}, {address.countryCode}</p></li>)}</ul></Panel>
    <Panel title="Bank-control references" description="Only opaque approved references and bounded approval/cooling-off state are shown; account data and provider credentials are never accepted here."><ul className="space-y-3">{party.bankDetailReferences.map((reference) => <li key={reference.reference} className="rounded-box border border-base-300 p-3"><div className="flex flex-wrap items-center justify-between gap-3"><span className="font-mono">{reference.reference}</span><StatusBadge state={reference.status === 'approved' ? 'success' : 'pending'} label={reference.status} /></div><p className="mt-1 text-sm text-base-content/70">Provider: {reference.providerCode}{reference.coolingOffUntil ? ` · Cooling off until ${reference.coolingOffUntil}` : ''}</p></li>)}</ul></Panel>
    <div className="grid gap-6 lg:grid-cols-2"><SensitiveDataGuard label="Restricted tax identifier" classification="highly-restricted" value={party.taxIdentifierMasked} access="restricted" canReveal={false} canExport={false} actorReference="current-actor" targetReference={party.id} scopeReference={party.scopeId} purpose="party maintenance" decisionReference="omd-party-read-decision" policyVersion="current" onAccess={recordAccess} onAccessDenied={() => setNotice('Restricted tax data remains masked; field-level access was not granted.')} /><SensitiveDataGuard label="Bank-control access evidence" classification="highly-restricted" access="restricted" canReveal={false} canExport={false} actorReference="current-actor" targetReference={party.id} scopeReference={party.scopeId} purpose="party bank-control review" decisionReference="omd-party-bank-decision" policyVersion="current" onAccess={recordAccess} onAccessDenied={() => setNotice('Bank-control access remains limited to opaque references and status.')}/></div>
  </section>
}
