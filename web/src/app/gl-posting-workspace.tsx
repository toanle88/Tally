import { useRef, useState, type FormEvent } from 'react'
import { NavLink, useLocation, useNavigate } from 'react-router-dom'

import { glApplyJournalApprovalDecision, glReverseJournalEntry, glSubmitPostingRequest } from '@/generated/api/sdk.gen'
import type { EstablishedResult, GlPostingRequestLine, GlSubmitPostingRequestEstablishedResult } from '@/generated/api/types.gen'
import { Button, Field, Panel, Select, StatusBadge, type SemanticState } from '@/components/ui'
import type { AccountingScopeFixture } from '@/lib/auth/auth-scope-adapter'
import { useScopeContext } from '@/lib/scope/scope-context'

export type GlPostingView = 'workbench' | 'request' | 'result'

type PostingResult = GlSubmitPostingRequestEstablishedResult
type ApprovalResult = EstablishedResult
type ReversalResult = EstablishedResult

type PostingLineDraft = Pick<GlPostingRequestLine, 'debitOrCredit' | 'lineCurrencyMode' | 'transactionAmount' | 'functionalAmount'> & {
  accountId: string
  segmentCombinationId: string
  lineReference: string
}

type PostingSubmissionIdentity = {
  contentFingerprint: string
  commandId: string
  requestId: string
  sourceAggregateId: string
  idempotencyKey: string
  correlationId: string
  conversionEvidence?: {
    rateSetId: string
    rateType: string
    conversionDate: string
    conversionTimestamp: string
  }
}

const initialLines: PostingLineDraft[] = [
  { accountId: 'account-cash', segmentCombinationId: 'combination-cash', debitOrCredit: 'debit', lineCurrencyMode: 'TransactionAndFunctional', transactionAmount: '100.00', functionalAmount: '100.00', lineReference: 'Manual debit' },
  { accountId: 'account-revenue', segmentCombinationId: 'combination-revenue', debitOrCredit: 'credit', lineCurrencyMode: 'TransactionAndFunctional', transactionAmount: '100.00', functionalAmount: '100.00', lineReference: 'Manual credit' },
]

function newIdentifier() {
  return globalThis.crypto?.randomUUID?.() ?? `ui-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

function decimalParts(value: string) {
  const normalized = value.trim()
  const match = /^(\d+)(?:\.(\d+))?$/.exec(normalized)
  if (!match) return null
  const fraction = match[2] ?? ''
  return { unscaled: BigInt(`${match[1]}${fraction}`), scale: fraction.length }
}

function totalsBalance(left: string[], right: string[]) {
  const leftParts = left.map(decimalParts)
  const rightParts = right.map(decimalParts)
  if (leftParts.some((part) => part === null) || rightParts.some((part) => part === null)) return false
  const parts = [...leftParts, ...rightParts] as Array<{ unscaled: bigint; scale: number }>
  const scale = Math.max(...parts.map((part) => part.scale), 0)
  const scaleFactor = BigInt(10) ** BigInt(scale)
  const total = (values: Array<{ unscaled: bigint; scale: number }>) => values.reduce((sum, part) => sum + part.unscaled * (scaleFactor / (BigInt(10) ** BigInt(part.scale))), BigInt(0))
  return total(leftParts as Array<{ unscaled: bigint; scale: number }>) === total(rightParts as Array<{ unscaled: bigint; scale: number }>)
}

function errorDetail(error: unknown) {
  if (!error || typeof error !== 'object') return 'The posting request could not be established. Review the current scope and retry safely.'
  const candidate = error as { detail?: unknown; fieldErrors?: unknown }
  if (typeof candidate.detail === 'string') {
    const fields = Array.isArray(candidate.fieldErrors)
      ? candidate.fieldErrors
        .filter((field): field is { field?: unknown; message?: unknown } => typeof field === 'object' && field !== null)
        .map((field) => `${typeof field.field === 'string' ? field.field : 'request'}: ${typeof field.message === 'string' ? field.message : 'invalid'}`)
      : []
    return fields.length ? `${candidate.detail} ${fields.join(' ')}` : candidate.detail
  }
  return 'The posting request could not be established. Review the current scope and retry safely.'
}

function postingLineFromDraft(line: PostingLineDraft): GlPostingRequestLine {
  return {
    accountId: line.accountId,
    debitOrCredit: line.debitOrCredit,
    lineCurrencyMode: line.lineCurrencyMode,
    transactionAmount: line.transactionAmount.trim(),
    functionalAmount: line.functionalAmount.trim(),
    segmentCombinationId: line.segmentCombinationId,
    lineReference: line.lineReference.trim() || undefined,
  }
}

function PostingRequestForm({ scope, onAccepted }: { scope: AccountingScopeFixture; onAccepted: (result: PostingResult) => void }) {
  const [postingDate, setPostingDate] = useState('2026-08-15')
  const [transactionCurrency, setTransactionCurrency] = useState(scope.functionalCurrency)
  const [purpose, setPurpose] = useState<'Ordinary' | 'Close' | 'ReopenCorrection' | 'OperationalReopen' | 'PolicyAdjustment'>('Ordinary')
  const [description, setDescription] = useState('Manual posting request')
  const [lines, setLines] = useState<PostingLineDraft[]>(initialLines)
  const [notice, setNotice] = useState('Validate the request before submitting an idempotent command.')
  const [validationError, setValidationError] = useState('')
  const [saving, setSaving] = useState(false)
  const submissionIdentity = useRef<PostingSubmissionIdentity | null>(null)

  const updateLine = <K extends keyof PostingLineDraft>(index: number, key: K, value: PostingLineDraft[K]) => {
    setLines((current) => current.map((line, candidateIndex) => candidateIndex === index ? { ...line, [key]: value } : line))
  }

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    setValidationError('')
    if (!scope.period) {
      setValidationError('A current fiscal period is required before a posting request can be submitted.')
      return
    }
    if (lines.length < 2) {
      setValidationError('A posting request requires at least two lines.')
      return
    }
    if (lines.some((line) => !line.accountId.trim() || !line.segmentCombinationId.trim() || !decimalParts(line.transactionAmount) || !decimalParts(line.functionalAmount))) {
      setValidationError('Each line requires an account, segment combination, and non-negative exact-decimal amounts.')
      return
    }
    const debitTransaction = lines.filter((line) => line.debitOrCredit === 'debit').map((line) => line.transactionAmount)
    const creditTransaction = lines.filter((line) => line.debitOrCredit === 'credit').map((line) => line.transactionAmount)
    const debitFunctional = lines.filter((line) => line.debitOrCredit === 'debit').map((line) => line.functionalAmount)
    const creditFunctional = lines.filter((line) => line.debitOrCredit === 'credit').map((line) => line.functionalAmount)
    if (!totalsBalance(debitTransaction, creditTransaction) || !totalsBalance(debitFunctional, creditFunctional)) {
      setValidationError('Debit and credit totals must balance in both transaction and functional currency.')
      return
    }

    const contentFingerprint = JSON.stringify({
      scopeId: scope.id,
      tenantId: scope.tenant.id,
      legalEntityId: scope.legalEntity.id,
      ledgerId: scope.ledger.id,
      accountingBookId: scope.accountingBook.id,
      functionalCurrency: scope.functionalCurrency,
      periodId: scope.period.id,
      postingDate,
      transactionCurrency,
      purpose,
      description: description.trim(),
      lines: lines.map(postingLineFromDraft),
    })
    const identity = submissionIdentity.current?.contentFingerprint === contentFingerprint
      ? submissionIdentity.current
      : {
          contentFingerprint,
          commandId: newIdentifier(),
          requestId: newIdentifier(),
          sourceAggregateId: newIdentifier(),
          idempotencyKey: newIdentifier(),
          correlationId: newIdentifier(),
          conversionEvidence: transactionCurrency === scope.functionalCurrency ? undefined : { rateSetId: newIdentifier(), rateType: 'spot', conversionDate: postingDate, conversionTimestamp: `${postingDate}T00:00:00Z` },
        }
    submissionIdentity.current = identity
    setSaving(true)
    setNotice('Submitting the typed posting command and waiting for the authoritative result…')
    try {
      const response = await glSubmitPostingRequest({
        body: {
          commandId: identity.commandId,
          accountingScopeId: scope.id,
          data: {
            contractVersion: 2,
            requestId: identity.requestId,
            sourceContext: 'manual-entry',
            sourceAggregateType: 'posting-request',
            sourceAggregateId: identity.sourceAggregateId,
            sourceVersion: 1,
            tenantId: scope.tenant.id,
            legalEntityId: scope.legalEntity.id,
            ledgerId: scope.ledger.id,
            accountingBookId: scope.accountingBook.id,
            functionalCurrency: scope.functionalCurrency,
            postingDate,
            fiscalPeriodId: scope.period.id,
            periodStateVersion: 1,
            postingGateVersion: 1,
            postingPurpose: purpose,
            transactionCurrency,
            conversionEvidence: identity.conversionEvidence,
            description: description.trim() || undefined,
            lines: lines.map(postingLineFromDraft),
          },
        },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': scope.id, 'X-Correlation-Id': identity.correlationId },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setNotice('The posting command returned no established result. No local journal projection was created.')
        return
      }
      submissionIdentity.current = null
      onAccepted(response.data)
      setNotice(`Authoritative result established: ${response.data.data.outcome}.`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The posting request could not reach the live API.')
    } finally {
      setSaving(false)
    }
  }

  return <Panel title="Submit posting request" description="The request carries source ownership, full accounting scope, period and gate evidence, exact-decimal lines, currency conversion evidence, correlation, and an idempotency key.">
    <form onSubmit={(event) => void submit(event)}>
      <div className="grid gap-4 md:grid-cols-2">
        <Field id="gl-posting-date" label="Posting date" type="date" value={postingDate} onChange={(event) => setPostingDate(event.target.value)} required />
        <label className="form-control w-full gap-2"><span className="label font-medium">Posting purpose</span><Select aria-label="Posting purpose" value={purpose} onChange={(event) => setPurpose(event.target.value as typeof purpose)}><option value="Ordinary">Ordinary</option><option value="Close">Close</option><option value="ReopenCorrection">Reopen correction</option><option value="OperationalReopen">Operational reopen</option><option value="PolicyAdjustment">Policy adjustment</option></Select></label>
        <label className="form-control w-full gap-2"><span className="label font-medium">Transaction currency</span><Select aria-label="Transaction currency" value={transactionCurrency} onChange={(event) => setTransactionCurrency(event.target.value)}><option value={scope.functionalCurrency}>{scope.functionalCurrency} · functional</option><option value="USD">USD</option><option value="EUR">EUR</option><option value="GBP">GBP</option></Select></label>
        <Field id="gl-posting-description" label="Description" value={description} onChange={(event) => setDescription(event.target.value)} />
      </div>
      <div className="mt-6 space-y-4">
        <div><h3 className="font-semibold">Posting lines</h3><p className="text-sm text-base-content/70">At least two lines are required. Amounts remain strings so the browser does not introduce binary floating-point rounding.</p></div>
        {lines.map((line, index) => <div key={`posting-line-${index}`} className="rounded-box border border-base-300 p-4"><div className="mb-3 flex flex-wrap items-center justify-between gap-2"><h4 className="font-semibold">Line {index + 1}</h4>{lines.length > 2 ? <Button type="button" variant="ghost" size="sm" onClick={() => setLines((current) => current.filter((_, candidateIndex) => candidateIndex !== index))}>Remove line</Button> : null}</div><div className="grid gap-4 md:grid-cols-2"><Field id={`gl-posting-account-${index}`} label={`Account ID · line ${index + 1}`} value={line.accountId} onChange={(event) => updateLine(index, 'accountId', event.target.value)} required /><Field id={`gl-posting-combination-${index}`} label={`Segment combination ID · line ${index + 1}`} value={line.segmentCombinationId} onChange={(event) => updateLine(index, 'segmentCombinationId', event.target.value)} required /><label className="form-control w-full gap-2"><span className="label font-medium">Direction · line {index + 1}</span><Select aria-label={`Direction · line ${index + 1}`} value={line.debitOrCredit} onChange={(event) => updateLine(index, 'debitOrCredit', event.target.value as PostingLineDraft['debitOrCredit'])}><option value="debit">Debit</option><option value="credit">Credit</option></Select></label><label className="form-control w-full gap-2"><span className="label font-medium">Currency mode · line {index + 1}</span><Select aria-label={`Currency mode · line ${index + 1}`} value={line.lineCurrencyMode} onChange={(event) => updateLine(index, 'lineCurrencyMode', event.target.value as PostingLineDraft['lineCurrencyMode'])}><option value="TransactionAndFunctional">Transaction and functional</option><option value="FunctionalOnlyAdjustment">Functional-only adjustment</option></Select></label><Field id={`gl-posting-transaction-amount-${index}`} label={`Transaction amount · line ${index + 1}`} inputMode="decimal" value={line.transactionAmount} onChange={(event) => updateLine(index, 'transactionAmount', event.target.value)} required /><Field id={`gl-posting-functional-amount-${index}`} label={`Functional amount · line ${index + 1}`} inputMode="decimal" value={line.functionalAmount} onChange={(event) => updateLine(index, 'functionalAmount', event.target.value)} required /><Field id={`gl-posting-reference-${index}`} label={`Line reference · line ${index + 1}`} value={line.lineReference} onChange={(event) => updateLine(index, 'lineReference', event.target.value)} /></div></div>)}
        <Button type="button" variant="secondary" onClick={() => setLines((current) => [...current, { ...initialLines[0], accountId: '', segmentCombinationId: '', transactionAmount: '0.00', functionalAmount: '0.00', lineReference: '' }])}>Add posting line</Button>
      </div>
      {validationError ? <p role="alert" className="mt-4 rounded-box border border-error/40 bg-error/10 p-3 text-sm">{validationError}</p> : null}
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
      <div className="mt-4"><Button type="submit" loading={saving}>Submit posting request</Button></div>
    </form>
  </Panel>
}

function PostingResultPanel({ result }: { result: PostingResult | null }) {
  if (!result) return <Panel title="Posting result" description="This screen receives the established result from a submitted request."><p role="note" className="text-sm text-base-content/75">No result is loaded. Submit a request from <NavLink className="link link-primary" to="/general-ledger/gl-scr-01">GL-SCR-01</NavLink>, then return here using the established result link.</p></Panel>
  const state: SemanticState = result.data.lifecycleStatus === 'Posted' ? 'success' : result.data.lifecycleStatus === 'Cancelled' ? 'error' : 'pending'
  return <Panel title="Posting result" description="This is the authoritative result returned by the posting command; no local journal projection is substituted."><div className="flex flex-wrap items-center gap-3"><StatusBadge state={state} label={`${result.data.outcome} · ${result.data.lifecycleStatus}`} announce /><span className="text-sm text-base-content/70">Request {result.aggregateId} · version {result.aggregateVersion}</span></div><dl className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"><div><dt className="text-sm text-base-content/65">Journal</dt><dd className="font-mono">{result.data.journalNumber ?? result.data.journalId ?? 'Not established'}</dd></div><div><dt className="text-sm text-base-content/65">Journal version</dt><dd>{result.data.journalVersion ?? '—'}</dd></div><div><dt className="text-sm text-base-content/65">Ledger position</dt><dd>{result.data.ledgerPosition ?? 'No ledger effect'}</dd></div><div><dt className="text-sm text-base-content/65">Approval reference</dt><dd className="font-mono">{result.data.approvalRequestId ?? 'Not required'}</dd></div><div><dt className="text-sm text-base-content/65">Approval policy</dt><dd>{result.data.policyReference ? `${result.data.policyReference} · v${result.data.policyVersion ?? '—'}` : 'Not required'}</dd></div><div><dt className="text-sm text-base-content/65">Gate evidence</dt><dd>{result.data.gateEvidence.gateMode} · period v{result.data.gateEvidence.periodStateVersion} · gate v{result.data.gateEvidence.postingGateVersion}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="font-mono">{result.data.auditReference ?? 'Unavailable in result'}</dd></div><div><dt className="text-sm text-base-content/65">Source reference</dt><dd className="font-mono break-all">{result.data.sourceReference}</dd></div><div><dt className="text-sm text-base-content/65">Next action</dt><dd>{result.data.nextAction ?? 'None'}</dd></div><div><dt className="text-sm text-base-content/65">Replay</dt><dd>{result.data.replayed ? 'Existing result returned' : 'First establishment'}</dd></div></dl>{result.data.issues?.length ? <div className="mt-5 rounded-box border border-warning/40 bg-warning/10 p-4"><h3 className="font-semibold">Validation evidence</h3><ul className="mt-2 space-y-2 text-sm">{result.data.issues.map((issue) => <li key={`${issue.code}-${issue.field}`}><span className="font-mono">{issue.code}</span> · {issue.field}: {issue.message}</li>)}</ul></div> : null}</Panel>
}

function approvalDataString(result: ApprovalResult | null, key: string) {
  const value = result?.data[key]
  return typeof value === 'string' ? value : value == null ? '—' : String(value)
}

function PostingApprovalDecisionPanel({ result, scope, appliedResult, onApplied }: { result: PostingResult | null; scope: AccountingScopeFixture; appliedResult: ApprovalResult | null; onApplied: (result: ApprovalResult) => void }) {
  const [outcome, setOutcome] = useState<'approved' | 'rejected'>('approved')
  const [approvalRequestId, setApprovalRequestId] = useState(result?.data.approvalRequestId ?? '')
  const [decisionId, setDecisionId] = useState<string>(newIdentifier())
  const [policyVersion] = useState(result?.data.policyVersion ?? '')
  const [decisionVersion, setDecisionVersion] = useState('1')
  const [subjectVersion, setSubjectVersion] = useState(String(result?.data.journalVersion ?? result?.aggregateVersion ?? 1))
  const [candidateFingerprint, setCandidateFingerprint] = useState('')
  const [approverUserId, setApproverUserId] = useState('')
  const [reason, setReason] = useState('')
  const [notice, setNotice] = useState('The pending journal has no ledger effect until an immutable approval decision is verified and applied.')
  const [saving, setSaving] = useState(false)
  const submissionIdentity = useRef<{ contentFingerprint: string; commandId: string; idempotencyKey: string; correlationId: string } | null>(null)

  if (!result || result.data.lifecycleStatus !== 'PendingApproval') return null

  const journalId = result.data.journalId ?? result.aggregateId
  const currentVersion = result.data.journalVersion ?? result.aggregateVersion
  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!approvalRequestId.trim() || !decisionId.trim() || !policyVersion.trim() || !candidateFingerprint.trim() || !approverUserId.trim() || !Number(decisionVersion) || !Number(subjectVersion)) {
      setNotice('Approval request, decision, policy, subject-version, candidate-fingerprint, and approver references are required.')
      return
    }
    const contentFingerprint = JSON.stringify({ journalId, outcome, approvalRequestId, decisionId, policyVersion, decisionVersion, subjectVersion, candidateFingerprint, approverUserId, reason })
    const identity = submissionIdentity.current?.contentFingerprint === contentFingerprint
      ? submissionIdentity.current
      : { contentFingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    submissionIdentity.current = identity
    setSaving(true)
    setNotice('Verifying the immutable Workflow decision and revalidating the current journal and gate…')
    try {
      const response = await glApplyJournalApprovalDecision({
        body: {
          commandId: identity.commandId,
          expectedVersion: currentVersion,
          accountingScopeId: scope.id,
          data: {
            journalId,
            outcome,
            approvalRequestId,
            decisionId,
            policyVersion,
            decisionVersion: Number(decisionVersion),
            subjectVersion: Number(subjectVersion),
            candidateFingerprint,
            approverUserId,
            reason: reason.trim() || undefined,
          },
        },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': scope.id, 'X-Correlation-Id': identity.correlationId },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setNotice('The approval decision returned no established result. Keep the command identity and retry safely.')
        return
      }
      submissionIdentity.current = null
      onApplied(response.data)
      setNotice(`Authoritative approval result established: ${approvalDataString(response.data, 'outcome')}.`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The approval decision could not reach the live API. Keep the command identity and retry.')
    } finally {
      setSaving(false)
    }
  }

  return <Panel title="Apply journal approval decision" description="This panel accepts immutable Workflow decision evidence only. It does not edit approval policy or replace the submitted journal facts.">
    <div className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm"><p><span className="font-semibold">Current journal:</span> <span className="font-mono">{journalId}</span> · version {currentVersion}</p><p className="mt-1"><span className="font-semibold">Approval request:</span> <span className="font-mono">{approvalRequestId || 'Not returned by the submit result'}</span></p><p className="mt-1">The pending journal has no ledger position. GL will revalidate scope, references, currency, balance, period, gate, authorization, and decision identity before applying it.</p></div>
    {appliedResult ? <div role="status" className={`mt-4 rounded-box border p-4 ${approvalDataString(appliedResult, 'lifecycleStatus') === 'Cancelled' ? 'border-error/40 bg-error/10' : 'border-success/40 bg-success/10'}`}><p className="font-semibold">Decision applied</p><dl className="mt-2 grid gap-2 sm:grid-cols-2"><div><dt className="text-sm text-base-content/65">Outcome</dt><dd>{approvalDataString(appliedResult, 'outcome')} · {approvalDataString(appliedResult, 'lifecycleStatus')}</dd></div><div><dt className="text-sm text-base-content/65">Decision reference</dt><dd className="font-mono">{approvalDataString(appliedResult, 'approvalDecisionId')}</dd></div><div><dt className="text-sm text-base-content/65">Applied subject version</dt><dd>{approvalDataString(appliedResult, 'subjectVersion')}</dd></div><div><dt className="text-sm text-base-content/65">Next action</dt><dd>{approvalDataString(appliedResult, 'nextAction')}</dd></div></dl></div> : <form className="mt-5" onSubmit={(event) => void submit(event)}>
      <div className="grid gap-4 md:grid-cols-2">
        <label className="form-control w-full gap-2"><span className="label font-medium">Decision outcome</span><Select aria-label="Decision outcome" value={outcome} onChange={(event) => setOutcome(event.target.value as typeof outcome)}><option value="approved">Approve</option><option value="rejected">Reject</option></Select></label>
        <Field id="gl-approval-request-id" label="Approval request ID" value={approvalRequestId} onChange={(event) => setApprovalRequestId(event.target.value)} required />
        <Field id="gl-approval-decision-id" label="Decision ID" value={decisionId} onChange={(event) => setDecisionId(event.target.value)} required />
        <Field id="gl-approval-policy-reference" label="Approval policy" value={result.data.policyReference ?? 'Unavailable from submit result'} readOnly />
        <Field id="gl-approval-policy-version" label="Policy version" value={policyVersion || 'Unavailable from submit result'} readOnly required />
        <Field id="gl-approval-decision-version" label="Decision version" type="number" min="1" value={decisionVersion} onChange={(event) => setDecisionVersion(event.target.value)} required />
        <Field id="gl-approval-subject-version" label="Subject version" type="number" min="1" value={subjectVersion} onChange={(event) => setSubjectVersion(event.target.value)} required />
        <Field id="gl-approval-candidate-fingerprint" label="Candidate fingerprint" value={candidateFingerprint} onChange={(event) => setCandidateFingerprint(event.target.value)} required />
        <Field id="gl-approval-approver-user-id" label="Approver user ID" value={approverUserId} onChange={(event) => setApproverUserId(event.target.value)} required />
        <Field id="gl-approval-reason" label="Decision reason" value={reason} onChange={(event) => setReason(event.target.value)} />
      </div>
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p>
      <div className="mt-4"><Button type="submit" loading={saving}>{outcome === 'approved' ? 'Apply approval' : 'Apply rejection'}</Button></div>
    </form>}
  </Panel>
}

function PostingEvidencePanel({ result }: { result: PostingResult | null }) {
  const permission = result?.data.lifecycleStatus === 'PendingApproval' ? 'finance.gl.apply.journal.approval.decision' : 'finance.gl.submit.posting.request'
  return <Panel title="Validation, ownership, and recovery" description="Every state shown here remains tied to the command result or an explicit dependency boundary."><dl className="grid gap-4 sm:grid-cols-2"><div><dt className="text-sm text-base-content/65">State</dt><dd className="font-medium">{result?.data.lifecycleStatus ?? 'Not submitted'}</dd></div><div><dt className="text-sm text-base-content/65">Owner</dt><dd className="font-medium">General Ledger</dd></div><div><dt className="text-sm text-base-content/65">Validation</dt><dd className="font-medium">{result?.data.validationOutcome ?? 'Awaiting authoritative validation'}</dd></div><div><dt className="text-sm text-base-content/65">Authorization</dt><dd className="font-mono text-sm">{permission}</dd></div><div><dt className="text-sm text-base-content/65">Evidence</dt><dd>{result ? 'Gate, source, audit, and idempotency evidence returned by GL.' : 'Current scope, period, and typed request fields are visible before submission.'}</dd></div><div><dt className="text-sm text-base-content/65">Correction / recovery</dt><dd>{result?.data.nextAction ?? 'Correct the identified field or dependency, then retry with the same request identity only when the business content is unchanged.'}</dd></div></dl></Panel>
}

function oppositeDirection(direction: GlPostingRequestLine['debitOrCredit']) {
  return direction === 'debit' ? 'credit' : 'debit'
}

function reversalDataString(result: ReversalResult | null, key: string) {
  const value = result?.data[key]
  return typeof value === 'string' || typeof value === 'number' ? String(value) : value == null ? '—' : JSON.stringify(value)
}

function PostingReversalPanel({ result, scope, reversalResult, onReversed }: { result: PostingResult | null; scope: AccountingScopeFixture; reversalResult: ReversalResult | null; onReversed: (result: ReversalResult) => void }) {
  const [reason, setReason] = useState('')
  const [effectivePostingDate, setEffectivePostingDate] = useState('2026-08-20')
  const [postingAuthorizationId, setPostingAuthorizationId] = useState('')
  const [sourceContext, setSourceContext] = useState('general-ledger')
  const [sourceAggregateType, setSourceAggregateType] = useState('journal-correction')
  const [sourceAggregateId, setSourceAggregateId] = useState<string>(() => newIdentifier())
  const [sourceVersion, setSourceVersion] = useState('1')
  const [notice, setNotice] = useState('Review the exact original lines before submitting the linked correction.')
  const [saving, setSaving] = useState(false)
  const submissionIdentity = useRef<{ contentFingerprint: string; commandId: string; idempotencyKey: string; correlationId: string } | null>(null)

  if (!result || result.data.lifecycleStatus !== 'Posted') return null
  const journalId = result.data.journalId ?? result.aggregateId
  const journalVersion = result.data.journalVersion ?? result.aggregateVersion
  const lines = result.data.lines ?? []
  const gateEvidence = result.data.gateEvidence

  const submit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    if (!scope.period || !reason.trim() || !postingAuthorizationId.trim() || !sourceContext.trim() || !sourceAggregateType.trim() || !sourceAggregateId.trim() || !Number(sourceVersion) || Number(sourceVersion) < 1) {
      setNotice('Reason, effective posting date, authorization evidence, source identity, source ID, and positive source version are required.')
      return
    }
    const contentFingerprint = JSON.stringify({ journalId, journalVersion, reason: reason.trim(), effectivePostingDate, postingAuthorizationId: postingAuthorizationId.trim(), sourceContext: sourceContext.trim(), sourceAggregateType: sourceAggregateType.trim(), sourceAggregateId: sourceAggregateId.trim(), sourceVersion: Number(sourceVersion) })
    const identity = submissionIdentity.current?.contentFingerprint === contentFingerprint
      ? submissionIdentity.current
      : { contentFingerprint, commandId: newIdentifier(), idempotencyKey: newIdentifier(), correlationId: newIdentifier() }
    submissionIdentity.current = identity
    setSaving(true)
    setNotice('Submitting the reversal command for current gate validation and authoritative linkage…')
    try {
      const response = await glReverseJournalEntry({
        body: {
          commandId: identity.commandId,
          expectedVersion: journalVersion,
          accountingScopeId: scope.id,
          data: {
            journalId,
            reason: reason.trim(),
            effectivePostingDate,
            fiscalPeriodId: gateEvidence.fiscalPeriodId,
            periodStateVersion: gateEvidence.periodStateVersion,
            postingGateVersion: gateEvidence.postingGateVersion,
            postingAuthorizationId: postingAuthorizationId.trim(),
            sourceContext: sourceContext.trim(),
            sourceAggregateType: sourceAggregateType.trim(),
            sourceAggregateId: sourceAggregateId.trim(),
            sourceVersion: Number(sourceVersion),
          },
        },
        headers: { 'Idempotency-Key': identity.idempotencyKey, 'X-Accounting-Scope-Id': scope.id, 'X-Correlation-Id': identity.correlationId },
      })
      if (response.error) {
        setNotice(errorDetail(response.error))
        return
      }
      if (!response.data) {
        setNotice('The reversal returned no established result. Keep the command identity and retry safely.')
        return
      }
      submissionIdentity.current = null
      onReversed(response.data)
      setNotice(`Authoritative reversal established: ${reversalDataString(response.data, 'outcome')}.`)
    } catch (error) {
      setNotice(error instanceof Error ? error.message : 'The reversal could not reach the live API. Keep the command identity and retry.')
    } finally {
      setSaving(false)
    }
  }

  return <Panel title="Reverse posted journal" description="The original posted fact remains immutable. GL will append a separate PolicyAdjustment journal with the same exact amounts and inverted debit/credit directions, linked to this journal and the correction owner’s source identity.">
    <div className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm"><p><span className="font-semibold">Original journal:</span> <span className="font-mono">{result.data.journalNumber ?? journalId}</span> · version {journalVersion}</p><p className="mt-1"><span className="font-semibold">Gate evidence:</span> {gateEvidence.gateMode} · period v{gateEvidence.periodStateVersion} · gate v{gateEvidence.postingGateVersion}</p><p className="mt-1">The correction owner supplies a reason, authorization reference, effective date, and explicit source identity. A successful result returns the original and linked reversal lineage.</p></div>
    <div className="mt-5"><h3 className="font-semibold">Equal-and-opposite preview</h3><p className="mt-1 text-sm text-base-content/70">Amounts are displayed as exact strings; only debit/credit direction changes.</p><div className="mt-3 overflow-x-auto rounded-box border border-base-300"><table className="table table-sm"><caption className="sr-only">Original and proposed reversal lines</caption><thead><tr><th>Account</th><th>Original</th><th>Reversal</th><th>Transaction</th><th>Functional</th></tr></thead><tbody>{lines.length ? lines.map((line, index) => <tr key={`${line.accountId}-${index}`}><td className="font-mono">{line.accountId}</td><td>{line.debitOrCredit}</td><td>{oppositeDirection(line.debitOrCredit)}</td><td className="font-mono">{line.transactionAmount}</td><td className="font-mono">{line.functionalAmount}</td></tr>) : <tr><td colSpan={5}>The established result did not include lines; refresh the authoritative journal before reversing.</td></tr>}</tbody></table></div></div>
    {reversalResult ? <div role="status" className="mt-5 rounded-box border border-success/40 bg-success/10 p-4"><p className="font-semibold">Linked reversal established</p><dl className="mt-2 grid gap-2 sm:grid-cols-2"><div><dt className="text-sm text-base-content/65">Reversal journal</dt><dd className="font-mono">{reversalDataString(reversalResult, 'reversalJournalNumber')}</dd></div><div><dt className="text-sm text-base-content/65">Reversal ID</dt><dd className="font-mono">{reversalDataString(reversalResult, 'reversalJournalId')}</dd></div><div><dt className="text-sm text-base-content/65">Reason</dt><dd>{reversalDataString(reversalResult, 'correctionReason')}</dd></div><div><dt className="text-sm text-base-content/65">Audit reference</dt><dd className="font-mono">{reversalDataString(reversalResult, 'auditReference')}</dd></div></dl></div> : <form className="mt-5" onSubmit={(event) => void submit(event)}>
      <div className="grid gap-4 md:grid-cols-2"><Field id="gl-reversal-reason" label="Correction reason" value={reason} onChange={(event) => setReason(event.target.value)} required /><Field id="gl-reversal-date" label="Effective posting date" type="date" value={effectivePostingDate} onChange={(event) => setEffectivePostingDate(event.target.value)} required /><Field id="gl-reversal-authorization" label="Authorization evidence ID" value={postingAuthorizationId} onChange={(event) => setPostingAuthorizationId(event.target.value)} required /><Field id="gl-reversal-source-context" label="Source context" value={sourceContext} onChange={(event) => setSourceContext(event.target.value)} required /><Field id="gl-reversal-source-type" label="Source aggregate type" value={sourceAggregateType} onChange={(event) => setSourceAggregateType(event.target.value)} required /><Field id="gl-reversal-source-id" label="Source aggregate ID" value={sourceAggregateId} onChange={(event) => setSourceAggregateId(event.target.value)} required /><Field id="gl-reversal-source-version" label="Source version" type="number" min="1" value={sourceVersion} onChange={(event) => setSourceVersion(event.target.value)} required /></div>
      <p role="status" aria-live="polite" className="mt-4 text-sm text-base-content/75">{notice}</p><div className="mt-4"><Button type="submit" loading={saving}>Reverse posted journal</Button></div>
    </form>}
  </Panel>
}

export function GlPostingWorkspace({ view }: { view: GlPostingView }) {
  const { currentScope } = useScopeContext()
  const location = useLocation()
  const navigate = useNavigate()
  const locationState = location.state as { postingResult?: PostingResult; approvalResult?: ApprovalResult; reversalResult?: ReversalResult } | null
  const [result, setResult] = useState<PostingResult | null>(locationState?.postingResult ?? null)
  const [approvalResult, setApprovalResult] = useState<ApprovalResult | null>(locationState?.approvalResult ?? null)
  const [reversalResult, setReversalResult] = useState<ReversalResult | null>(locationState?.reversalResult ?? null)
  const [recentResults, setRecentResults] = useState<PostingResult[]>([])

  if (!currentScope) {
    return <section><p className="text-sm font-semibold uppercase tracking-wide text-primary">{view === 'workbench' ? 'GL-WS-01' : view === 'request' ? 'GL-SCR-01' : 'GL-SCR-02'}</p><h1 className="mt-1 text-2xl font-semibold">Posting request</h1><p className="mt-2">Select an accounting scope before submitting or reviewing a posting request.</p></section>
  }

  const screenId = view === 'workbench' ? 'GL-WS-01' : view === 'request' ? 'GL-SCR-01' : 'GL-SCR-02'
  const title = view === 'workbench' ? 'Posting request workbench' : view === 'request' ? 'Posting request' : 'Posting result'
  const onAccepted = (nextResult: PostingResult) => {
    setResult(nextResult)
    setRecentResults((current) => [nextResult, ...current].slice(0, 5))
    if (view === 'request') {
      void navigate('/general-ledger/gl-scr-02', { state: { postingResult: nextResult } })
    }
  }

  return <section aria-labelledby="gl-posting-title" className="space-y-6"><div><p className="text-sm font-semibold uppercase tracking-wide text-primary">{screenId}</p><h1 id="gl-posting-title" className="mt-1 text-2xl font-semibold">{title}</h1><p className="mt-2 max-w-3xl text-base-content/75">Submit and validate one balanced posting request in the selected accounting scope. GL remains the owner of validation, authorization, journal establishment, audit evidence, and safe retry outcomes.</p><nav aria-label="Posting screens" className="mt-4 flex flex-wrap gap-3 text-sm"><NavLink className="link link-primary" to="/general-ledger/gl-ws-01">GL-WS-01 workbench</NavLink><NavLink className="link link-primary" to="/general-ledger/gl-scr-01">GL-SCR-01 request</NavLink><NavLink className="link link-primary" to="/general-ledger/gl-scr-02">GL-SCR-02 result</NavLink></nav></div><p role="note" className="rounded-box border border-warning/40 bg-warning/10 p-4 text-sm">Scope: {currentScope.legalEntity.name} · {currentScope.ledger.name} · {currentScope.accountingBook.name} · {currentScope.functionalCurrency} · {currentScope.period?.label ?? 'No fiscal period selected'}. Read source: selected-scope fixture; mutation source: typed <code>glSubmitPostingRequest</code> API. No local journal is treated as authoritative.</p>{view !== 'result' ? <PostingRequestForm scope={currentScope} onAccepted={onAccepted} /> : null}{view === 'workbench' ? <Panel title="Recent established results" description="This local list is a safe session view of returned results, not a substitute for a GL journal read endpoint.">{recentResults.length ? <ul className="space-y-3">{recentResults.map((entry) => <li key={`${entry.aggregateId}-${entry.correlationId}`} className="rounded-box border border-base-300 p-3"><p className="font-semibold">{entry.data.outcome} · {entry.data.journalNumber ?? entry.data.journalId ?? 'pending journal'}</p><p className="mt-1 text-sm text-base-content/70">{entry.data.replayed ? 'Replayed existing result' : 'New established result'} · source {entry.data.sourceReference}</p></li>)}</ul> : <p className="text-sm text-base-content/70">No posting result has been returned in this session.</p>}</Panel> : null}<PostingResultPanel result={result} /><PostingApprovalDecisionPanel result={result} scope={currentScope} appliedResult={approvalResult} onApplied={(nextResult) => { setApprovalResult(nextResult); void navigate('/general-ledger/gl-scr-02', { state: { postingResult: result, approvalResult: nextResult } }) }} /><PostingReversalPanel result={result} scope={currentScope} reversalResult={reversalResult} onReversed={(nextResult) => { setReversalResult(nextResult); void navigate('/general-ledger/gl-scr-02', { state: { postingResult: result, reversalResult: nextResult } }) }} /><PostingEvidencePanel result={result} /></section>
}
