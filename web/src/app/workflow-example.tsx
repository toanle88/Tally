import { useState } from 'react'
import { z } from 'zod'
import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'

import { ApprovalPanel } from '@/components/approval-panel'
import { PostingPanel } from '@/components/posting-panel'
import { ValidationSummary } from '@/components/validation-summary'
import { VersionConflictDialog } from '@/components/version-conflict-dialog'
import { Button, ConfirmationSurface, Field, Heading, Panel, Select, StatusBadge } from '@/components/ui'
import { formatDecimalInput, normalizeDecimalInput } from '@/components/workflow-context'
import type { MaterialActionFormValues, ValidationIssue } from '@/components/workflow-context'

import { approvalFixture, conflictFixture, postingFixture, submitWorkflowFixture, workflowScope } from './workflow-fixtures'

const schema = z.object({ businessIdentity: z.string().trim().min(1, 'Enter a business identity.'), description: z.string().trim().min(5, 'Describe the intended action.'), amount: z.string().trim().min(1, 'Enter an amount.').refine((value) => { const canonical = normalizeDecimalInput(value, 'en-US'); return canonical !== null && !/^0+(?:\.0+)?$/.test(canonical) }, 'Enter a positive decimal amount.'), currency: z.string().min(1, 'Select a currency.'), lineReference: z.string().trim().min(1, 'Enter a line reference.') })
const fieldLabels: Record<keyof MaterialActionFormValues, string> = { businessIdentity: 'Business identity', description: 'Intended action', amount: 'Amount', currency: 'Currency', lineReference: 'Line reference' }

export function WorkflowExample() {
  const [confirmation, setConfirmation] = useState<MaterialActionFormValues | null>(null)
  const [outcome, setOutcome] = useState<string>('No fixture submission has been attempted.')
  const [conflictOpen, setConflictOpen] = useState(false)
  const [issues, setIssues] = useState<ValidationIssue[]>([])
  const { register, handleSubmit, setError, setFocus, formState: { errors } } = useForm<MaterialActionFormValues>({ resolver: zodResolver(schema), shouldFocusError: false, defaultValues: { businessIdentity: 'SET-1042', description: 'Apply approved supplier settlement adjustment', amount: '1,250.00', currency: 'THB', lineReference: 'SET-1042-L1' } })
  const formIssues = Object.entries(errors).map(([field, error]) => ({ id: `field-${field}`, category: 'field' as const, code: 'INVALID_FIELD', message: error?.message ?? 'Review this field.', targetId: field, targetLabel: fieldLabels[field as keyof MaterialActionFormValues] ?? field }))
  const allIssues = [...formIssues, ...issues]
  const submit = (values: MaterialActionFormValues) => { setIssues([]); setConfirmation({ ...values, amount: normalizeDecimalInput(values.amount, 'en-US') ?? values.amount }) }
  const confirm = () => { setConfirmation(null); const result = submitWorkflowFixture(); if (result.kind === 'problem') { result.problem.fieldErrors?.forEach((fieldError) => setError(fieldError.field as keyof MaterialActionFormValues, { type: fieldError.code, message: fieldError.message })); setOutcome(`${result.problem.title}: ${result.problem.detail} Correlation ${result.problem.correlationId}.`); setConflictOpen(true) } }
  const focusIssue = (targetId: string) => { setFocus(targetId as keyof MaterialActionFormValues); document.getElementById(targetId)?.scrollIntoView({ block: 'center' }) }
  return <div className="space-y-6">
    <div><Heading level={2}>Material action workflow</Heading><p className="mt-2 max-w-3xl text-base-content/75">Synthetic Story 5 example: form validation, explicit confirmation, approval revalidation, posting outcome, and deliberate conflict recovery.</p></div>
    <Panel title="Material action form" description="Amounts remain strings at the form boundary; the server remains authoritative for acceptance.">
      <form className="mt-4 space-y-5" noValidate onSubmit={handleSubmit(submit, () => { setIssues([]) })}>
        {allIssues.length ? <ValidationSummary issues={allIssues} onFocusTarget={focusIssue} /> : null}
        <div className="grid gap-5 md:grid-cols-2"><Field id="businessIdentity" label="Business identity" required {...register('businessIdentity')} error={errors.businessIdentity?.message} /><Field id="description" label="Intended action" required {...register('description')} error={errors.description?.message} /><Field id="amount" label="Amount" inputMode="decimal" required {...register('amount')} onBlur={(event) => { event.currentTarget.value = formatDecimalInput(normalizeDecimalInput(event.currentTarget.value, 'en-US') ?? event.currentTarget.value, 'en-US') }} error={errors.amount?.message} /><div className="form-control w-full gap-2"><label className="label" htmlFor="currency"><span className="label-text font-medium text-base-content">Currency</span><span aria-hidden="true" className="text-error">*</span></label><Select id="currency" {...register('currency')} aria-invalid={errors.currency ? true : undefined}><option value="">Select currency</option><option value="THB">THB — Thai baht</option><option value="USD">USD — US dollar</option></Select>{errors.currency ? <p className="text-sm text-error">{errors.currency.message}</p> : null}</div></div>
        <Field id="lineReference" label="Line reference" required {...register('lineReference')} error={errors.lineReference?.message} />
        <div className="flex flex-wrap gap-2"><Button type="submit">Review and continue</Button><Button variant="ghost" type="button" onClick={() => { setIssues([]); setOutcome('Draft preserved; no submission attempted.') }}>Keep draft</Button></div>
      </form>
    </Panel>
    {confirmation ? <ConfirmationSurface autoFocus title="Confirm material action" description="Review the complete action context before the owning capability receives the request." details={[{ label: 'Record', value: confirmation.businessIdentity }, { label: 'Reviewed version', value: '7' }, { label: 'Scope', value: workflowScope.label }, { label: 'Amount', value: confirmation.amount }, { label: 'Currency', value: confirmation.currency }, { label: 'Intended state change', value: confirmation.description }, { label: 'Accounting owner', value: 'General Ledger' }, { label: 'Approval status / reference', value: `decided · ${approvalFixture.requestId}` }, { label: 'Lineage effect', value: 'Creates an adjustment lineage reference; established facts remain immutable.' }]} confirmLabel="Confirm and submit" cancelLabel="Cancel and edit" onConfirm={confirm} onCancel={() => setConfirmation(null)} /> : null}
    <div className="grid items-start gap-6 xl:grid-cols-2"><ApprovalPanel approval={approvalFixture} /><PostingPanel posting={postingFixture} onRetry={() => setOutcome('Retry requires deliberate user action after request lookup; no automatic retry was performed.')} /></div>
    <Panel title="Typed outcome" description="Every fixture outcome retains a correlation reference and an actionable next step."><p role="status" aria-live="polite" className="text-sm">{outcome}</p><Button id="workflow-conflict-trigger" className="mt-3" variant="secondary" onClick={() => setConflictOpen(true)}>Open conflict example</Button><span className="ml-2"><StatusBadge state="warning" label="Synthetic fixture" /></span></Panel>
    {conflictOpen ? <VersionConflictDialog conflict={conflictFixture} onClose={() => setConflictOpen(false)} onRefresh={() => setOutcome('Authoritative state refreshed: current version 8 is ready for review.')} onRetry={() => { setConflictOpen(false); setOutcome('Retry blocked until the refreshed version is reviewed.') }} onNewIdentity={() => { setConflictOpen(false); setOutcome('A new business identity must be created for an identity-content conflict.') }} /> : null}
  </div>
}
