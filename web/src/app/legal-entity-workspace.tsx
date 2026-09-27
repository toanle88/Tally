import { useMemo, useState } from 'react'
import { Link as RouterLink, useSearchParams } from 'react-router-dom'

import { SensitiveDataGuard } from '@/components/sensitive-data-guard'
import { Button, DataTable, Field, Panel, StatusBadge, type SemanticState } from '@/components/ui'
import { initialParties, type PartyFixture } from '@/app/party-workspace'
import { initialCustomerProfiles, profileStatusState, type CustomerProfileFixture } from '@/app/customer-profile-workspace'

type LegalEntityStatus = 'draft' | 'active' | 'end_dated'
type LegalEntity = {
  id: string
  scopeId: string
  legalName: string
  functionalCurrency: string
  presentationCurrency: string
  taxRegistrationMasked?: string
  status: LegalEntityStatus
  effectiveFrom: string
  effectiveTo?: string
  version: number
  approvalStatus: 'not-required' | 'approved' | 'pending'
  nextAction: string
  registrations: Array<{ type: string; identifierMasked: string; jurisdiction: string; effectiveFrom: string; effectiveTo?: string }>
  addresses: Array<{ type: string; line1: string; locality: string; region?: string; postalCode: string; countryCode: string }>
  ownershipInterests: Array<{ ownerReference: string; percentage: string }>
}

const initialLegalEntities: LegalEntity[] = [
  { id: 'entity-vietnam', scopeId: 'scope-vietnam-statutory', legalName: 'Acme Vietnam Co., Ltd.', functionalCurrency: 'VND', presentationCurrency: 'VND', taxRegistrationMasked: '••••••••', status: 'active', effectiveFrom: '2026-01-01', version: 4, approvalStatus: 'approved', nextAction: 'maintain', registrations: [{ type: 'company', identifierMasked: '••••6789', jurisdiction: 'VN', effectiveFrom: '2026-01-01' }], addresses: [{ type: 'registered', line1: '1 Main Street', locality: 'Hanoi', postalCode: '100000', countryCode: 'VN' }], ownershipInterests: [{ ownerReference: 'Acme Holdings', percentage: '100.000000' }] },
  { id: 'entity-singapore', scopeId: 'scope-singapore-management', legalName: 'Acme Singapore Pte. Ltd.', functionalCurrency: 'SGD', presentationCurrency: 'SGD', status: 'draft', effectiveFrom: '2026-02-01', version: 2, approvalStatus: 'pending', nextAction: 'submit for approval', registrations: [{ type: 'company', identifierMasked: '••••4321', jurisdiction: 'SG', effectiveFrom: '2026-02-01' }], addresses: [{ type: 'registered', line1: '2 Market Street', locality: 'Singapore', postalCode: '048940', countryCode: 'SG' }], ownershipInterests: [{ ownerReference: 'Acme Holdings', percentage: '100.000000' }] },
  { id: 'entity-legacy', scopeId: 'scope-vietnam-statutory', legalName: 'Acme Legacy Trading Co.', functionalCurrency: 'VND', presentationCurrency: 'USD', status: 'end_dated', effectiveFrom: '2020-01-01', effectiveTo: '2025-12-31', version: 8, approvalStatus: 'approved', nextAction: 'view history', registrations: [{ type: 'company', identifierMasked: '••••1122', jurisdiction: 'VN', effectiveFrom: '2020-01-01', effectiveTo: '2025-12-31' }], addresses: [{ type: 'registered', line1: '3 Old Road', locality: 'Da Nang', postalCode: '550000', countryCode: 'VN' }], ownershipInterests: [{ ownerReference: 'Acme Holdings', percentage: '100.000000' }] },
]

const statusState: Record<LegalEntityStatus, SemanticState> = { draft: 'pending', active: 'success', end_dated: 'disabled' }

export function LegalEntityWorklist() {
  const [search, setSearch] = useState('')
  const rows = useMemo(() => initialLegalEntities.filter((entity) => entity.legalName.toLowerCase().includes(search.toLowerCase()) || entity.id.includes(search.toLowerCase())), [search])
  const partyRows = useMemo(() => initialParties.filter((party) => party.name.toLowerCase().includes(search.toLowerCase()) || party.id.includes(search.toLowerCase())), [search])
  const customerProfileRows = useMemo(() => initialCustomerProfiles.filter((profile) => profile.partyName.toLowerCase().includes(search.toLowerCase()) || profile.id.includes(search.toLowerCase()) || profile.creditTerms.includes(search.toLowerCase())), [search])
  const columns = useMemo(() => [
    { key: 'name', header: 'Legal entity', rowHeader: true, render: (entity: LegalEntity) => <RouterLink className="link link-primary font-semibold" to={`/master-data/omd-scr-01?legalEntityId=${entity.id}`}>{entity.legalName}</RouterLink> },
    { key: 'status', header: 'State', render: (entity: LegalEntity) => <StatusBadge state={statusState[entity.status]} label={entity.status} /> },
    { key: 'effective', header: 'Effective interval', render: (entity: LegalEntity) => `${entity.effectiveFrom} — ${entity.effectiveTo ?? 'open'}` },
    { key: 'approval', header: 'Approval', render: (entity: LegalEntity) => <StatusBadge state={entity.approvalStatus === 'approved' || entity.approvalStatus === 'not-required' ? 'success' : 'pending'} label={entity.approvalStatus} /> },
    { key: 'version', header: 'Version', render: (entity: LegalEntity) => `v${entity.version}` },
    { key: 'nextAction', header: 'Next action', render: (entity: LegalEntity) => entity.nextAction },
  ] as const, [])

  const partyColumns = useMemo(() => [
    { key: 'name', header: 'Party', rowHeader: true, render: (party: PartyFixture) => <RouterLink className="link link-primary font-semibold" to={`/master-data/omd-scr-02?partyId=${party.id}`}>{party.name}</RouterLink> },
    { key: 'type', header: 'Type', render: (party: PartyFixture) => party.partyType },
    { key: 'status', header: 'State', render: (party: PartyFixture) => <StatusBadge state={party.status === 'active' ? 'success' : 'pending'} label={party.status} /> },
    { key: 'bankControl', header: 'Bank control', render: (party: PartyFixture) => <StatusBadge state={party.bankDetailReferences.every((reference) => reference.status === 'approved') ? 'success' : 'pending'} label={party.bankDetailReferences.every((reference) => reference.status === 'approved') ? 'approved' : 'pending'} /> },
    { key: 'version', header: 'Version', render: (party: PartyFixture) => `v${party.version}` },
  ] as const, [])

  const customerProfileColumns = useMemo(() => [
    { key: 'party', header: 'Customer Party', rowHeader: true, render: (profile: CustomerProfileFixture) => <RouterLink className="link link-primary font-semibold" to={`/master-data/omd-scr-03?customerProfileId=${profile.id}`}>{profile.partyName}</RouterLink> },
    { key: 'terms', header: 'Credit terms', render: (profile: CustomerProfileFixture) => profile.creditTerms },
    { key: 'limit', header: 'Credit limit', render: (profile: CustomerProfileFixture) => `${profile.creditLimit.amount} ${profile.creditLimit.currency}` },
    { key: 'billing', header: 'Billing', render: (profile: CustomerProfileFixture) => profile.billingPreference },
    { key: 'tax', header: 'Tax treatment', render: (profile: CustomerProfileFixture) => profile.taxTreatment },
    { key: 'status', header: 'State', render: (profile: CustomerProfileFixture) => <StatusBadge state={profileStatusState[profile.status]} label={profile.status} /> },
    { key: 'version', header: 'Version', render: (profile: CustomerProfileFixture) => `v${profile.version} · Party v${profile.partyVersion}` },
    { key: 'nextAction', header: 'Next action', render: (profile: CustomerProfileFixture) => profile.nextAction },
  ] as const, [])

  const recordCount = rows.length + partyRows.length + customerProfileRows.length
  return <section aria-labelledby="omd-worklist-title" className="space-y-6"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-WS-01</p><h2 id="omd-worklist-title" className="mt-1 text-2xl font-semibold">Legal-entity master-data worklist</h2><p className="mt-2 max-w-3xl text-base-content/75">Search and review authoritative legal-entity, party, and customer-profile records. Maintenance is available only on the record screens.</p></div><Panel title="Search and review" description="Results are scoped to the selected accounting context and show only safe projections."><div className="grid gap-4 md:grid-cols-[minmax(0,1fr)_auto] md:items-end"><Field id="omd-search" label="Search master-data records" placeholder="Name or record identifier" value={search} onChange={(event) => setSearch(event.target.value)} /><p className="text-sm text-base-content/70" role="status" aria-live="polite">{recordCount} record{recordCount === 1 ? '' : 's'} found.</p></div><div className="mt-6"><DataTable caption="Authoritative legal-entity records" columns={columns} rows={rows} getRowKey={(entity) => entity.id} emptyMessage="No authorized legal entities match this search." /></div><div className="mt-8"><DataTable caption="Safe party records" columns={partyColumns} rows={partyRows} getRowKey={(party) => party.id} emptyMessage="No authorized parties match this search." /></div><div className="mt-8"><DataTable caption="Safe customer-profile records" columns={customerProfileColumns} rows={customerProfileRows} getRowKey={(profile) => profile.id} emptyMessage="No authorized customer profiles match this search." /></div></Panel></section>
}

export function LegalEntityRecord() {
  const [params] = useSearchParams()
  const selected = initialLegalEntities.find((entity) => entity.id === params.get('legalEntityId')) ?? initialLegalEntities[0]
  const [entity, setEntity] = useState(selected)
  const [notice, setNotice] = useState('Review the authoritative version before making a material change.')
  const [name, setName] = useState(entity.legalName)
  const [effectiveTo, setEffectiveTo] = useState(entity.effectiveTo ?? '')

  const save = () => { setEntity((current) => ({ ...current, legalName: name.trim(), version: current.version + 1, nextAction: 'maintain' })); setNotice(`Accepted revision v${entity.version + 1}. Audit evidence and the effective interval were recorded.`) }
  const endDate = () => { if (!effectiveTo) { setNotice('Enter an effective end date before end-dating this entity.'); return }; setEntity((current) => ({ ...current, status: 'end_dated', effectiveTo, version: current.version + 1, nextAction: 'view history' })); setNotice(`Accepted end-dated revision v${entity.version + 1}. The previous revision remains available in history.`) }
  const recordAccess = () => { setNotice('Restricted-value access evidence recorded; the tax registration remains masked.'); return true }

  return <section aria-labelledby="omd-record-title" className="space-y-6"><div className="flex flex-wrap items-start justify-between gap-4"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">OMD-SCR-01</p><h2 id="omd-record-title" className="mt-1 text-2xl font-semibold">Legal-entity record</h2><p className="mt-2 max-w-3xl text-base-content/75">The authoritative mutation surface for identity, effective dates, registrations, addresses, and ownership interests.</p></div><RouterLink className="link link-primary focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-primary" to="/master-data/omd-ws-01">Back to worklist</RouterLink></div><p role="status" aria-live="polite" className="rounded-box border border-info/30 bg-info/5 p-4 text-sm">{notice}</p><Panel title="Identity and lifecycle" description="Changes use optimistic concurrency and create a new immutable revision."><div className="grid gap-4 md:grid-cols-2"><Field id="legal-entity-name" label="Legal name" value={name} onChange={(event) => setName(event.target.value)} disabled={entity.status === 'end_dated'} /><Field id="legal-entity-id" label="Record identifier" value={entity.id} readOnly /><Field id="functional-currency" label="Functional currency" value={entity.functionalCurrency} readOnly /><Field id="presentation-currency" label="Presentation currency" value={entity.presentationCurrency} readOnly /><Field id="effective-from" label="Effective from" value={entity.effectiveFrom} readOnly /><Field id="effective-to" label="Effective to" type="date" value={effectiveTo} onChange={(event) => setEffectiveTo(event.target.value)} disabled={entity.status === 'end_dated'} /></div><div className="mt-5 flex flex-wrap items-center gap-3"><StatusBadge state={statusState[entity.status]} label={entity.status} announce /><span className="text-sm text-base-content/70">Version {entity.version} · Approval: {entity.approvalStatus} · Next action: {entity.nextAction}</span></div><div className="mt-5 flex flex-wrap gap-3"><Button onClick={save} disabled={entity.status === 'end_dated'}>Save legal-entity revision</Button><Button variant="danger" onClick={endDate} disabled={entity.status === 'end_dated'}>End-date entity</Button></div></Panel><div className="grid gap-6 lg:grid-cols-2"><Panel title="Registrations" description="Identifiers are masked in the safe projection."><ul className="space-y-3">{entity.registrations.map((registration) => <li key={`${registration.type}-${registration.jurisdiction}`} className="rounded-box border border-base-300 p-3"><p className="font-semibold">{registration.type} · {registration.jurisdiction}</p><p className="mt-1 font-mono text-sm">{registration.identifierMasked}</p><p className="mt-1 text-sm text-base-content/70">{registration.effectiveFrom} — {registration.effectiveTo ?? 'open'}</p></li>)}</ul></Panel><Panel title="Ownership interests" description="Exact decimal percentages are retained without binary floating point."><ul className="space-y-3">{entity.ownershipInterests.map((interest) => <li key={interest.ownerReference} className="flex justify-between rounded-box border border-base-300 p-3"><span>{interest.ownerReference}</span><span className="font-mono">{interest.percentage}%</span></li>)}</ul></Panel></div><Panel title="Addresses"><ul className="grid gap-3 md:grid-cols-2">{entity.addresses.map((address) => <li key={address.type} className="rounded-box border border-base-300 p-3"><p className="font-semibold">{address.type}</p><p className="mt-1">{address.line1}, {address.locality}</p><p className="text-sm text-base-content/70">{address.region ? `${address.region}, ` : ''}{address.postalCode}, {address.countryCode}</p></li>)}</ul></Panel><SensitiveDataGuard label="Tax registration" classification="highly-restricted" value={entity.taxRegistrationMasked} access="restricted" canReveal={false} canExport={false} actorReference="current-actor" targetReference={entity.id} scopeReference={entity.scopeId} purpose="legal-entity maintenance" decisionReference="omd-read-decision" policyVersion="current" onAccess={recordAccess} onAccessDenied={() => setNotice('Restricted-value access denied; the value remains masked.')} /></section>
}
