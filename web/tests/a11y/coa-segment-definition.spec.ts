import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'
import { expectVisibleFocus } from './keyboard-assertions'
import { expectNoDocumentHorizontalOverflow } from './visual-assertions'

test('COA-WS-01 exposes the scoped safe-adapter worklist accessibly', async ({ page }) => {
  await page.goto('/coa-segments/coa-ws-01')

  await expect(page.getByRole('heading', { name: 'Segment administration worklist', level: 2 })).toBeVisible()
  await expect(page.getByRole('note')).toContainText('local safe adapter')
  await expect(page.getByRole('table', { name: 'Safe COA segment-definition projections' })).toBeVisible()
  await expect(page.getByRole('link', { name: 'Create segment definition' })).toBeVisible()
  await expectNoDocumentHorizontalOverflow(page, 'COA-WS-01')
  await expectNoAccessibilityViolations(page, 'COA-WS-01 worklist')
})

test('COA-SCR-01 preserves keyboard order and announces validation safely', async ({ page }) => {
  await page.goto('/coa-segments/coa-scr-01?new=true')

  const type = page.getByLabel('Segment type')
  const code = page.getByLabel('Code')
  const name = page.getByRole('textbox', { name: 'Name' })
  const status = page.getByLabel('Lifecycle status')
  const from = page.getByLabel('Effective from')
  const to = page.getByLabel('Effective to')
  const create = page.getByRole('button', { name: 'Create segment definition' })

  await type.focus()
  await expectVisibleFocus(type)
  for (const control of [code, name, status, from]) {
    await page.keyboard.press('Tab')
    await expect(control).toBeFocused()
  }
  await to.focus()
  await expectVisibleFocus(to)

  await code.fill('')
  await name.fill('')
  await create.click()

  await expect(page.getByRole('alert')).toContainText('Enter segment type, code, name, and a valid inclusive effective-date range.')
  await expectNoDocumentHorizontalOverflow(page, 'COA-SCR-01 validation')
  await expectNoAccessibilityViolations(page, 'COA-SCR-01 validation')
})

test('COA-SCR-02 preserves keyboard order and announces parent-boundary validation safely', async ({ page }) => {
  await page.goto('/coa-segments/coa-scr-02?segmentDefinitionId=segment-department-shared-services&new=true')

  await expect(page.getByRole('heading', { name: 'Segment value', level: 2 })).toBeVisible()
  await expect(page.getByRole('note')).toContainText('coaMaintainSegmentValues')
  await expect(page.getByLabel('Parent aggregate version')).toHaveValue('v1')

  const value = page.getByRole('textbox', { name: 'Value' })
  const description = page.getByRole('textbox', { name: 'Description' })
  const status = page.getByLabel('Lifecycle status')
  const from = page.getByLabel('Effective from')
  const to = page.getByLabel('Effective to')
  const create = page.getByRole('button', { name: 'Create segment value' })

  await value.focus()
  await expectVisibleFocus(value)
  for (const control of [description, status, from]) {
    await page.keyboard.press('Tab')
    await expect(control).toBeFocused()
  }
  await to.focus()
  await expectVisibleFocus(to)

  await value.fill('3000')
  await description.fill('Future services')
  await create.click()

  await expect(page.getByRole('alert')).toContainText('must be contained within the parent definition interval')
  await expectNoDocumentHorizontalOverflow(page, 'COA-SCR-02 validation')
  await expectNoAccessibilityViolations(page, 'COA-SCR-02 validation')
})

test('COA-SCR-03 displays a read-only validation result with source versions', async ({ page }) => {
  await page.route('**/api/v1/coa-segments/actions/validate-segment-combinations', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'established',
        aggregateId: '00000000-0000-0000-0000-000000000000',
        aggregateVersion: 0,
        processId: null,
        correlationId: '00000000-0000-0000-0000-000000000001',
        links: { self: '/api/v1/coa-segments/actions/validate-segment-combinations' },
        data: {
          validationStatus: 'valid',
          effectiveDateResult: 'effective',
          sourceVersions: [{ segmentDefinitionId: 'segment-department-operations', segmentValueId: 'segment-value-operations-1000', segmentDefinitionVersion: 3, segmentDefinitionRevision: 3 }],
          invalidValues: [],
          restrictions: [],
          rejectionReasons: [],
          nextAction: 'proceed',
        },
      }),
    })
  })
  await page.goto('/coa-segments/coa-scr-03')

  await expect(page.getByRole('heading', { name: 'Segment combination validator', level: 2 })).toBeVisible()
  await expect(page.getByRole('note')).toContainText('Validation is read-only')
  const operations = page.getByRole('checkbox', { name: /Operations/ })
  await operations.focus()
  await expectVisibleFocus(operations)
  await page.getByRole('button', { name: 'Validate combination' }).click()

  await expect(page.getByRole('textbox', { name: 'Validation status' })).toHaveValue('valid')
  await expect(page.getByRole('textbox', { name: 'Effective-date result' })).toHaveValue('effective')
  await expect(page.getByText(/segment-department-operations \/ segment-value-operations-1000: v3, revision 3/)).toBeVisible()
  await expectNoDocumentHorizontalOverflow(page, 'COA-SCR-03 result')
  await expectNoAccessibilityViolations(page, 'COA-SCR-03 result')
})

test('COA-SCR-04 captures the Workflow reference and exposes a pending safe request', async ({ page }) => {
  await page.route('**/api/v1/coa-segments/actions/request-segment-changes', async (route) => {
    await route.fulfill({
      status: 200,
      contentType: 'application/json',
      body: JSON.stringify({
        status: 'established',
        aggregateId: 'request-001',
        aggregateVersion: 1,
        processId: null,
        correlationId: '00000000-0000-0000-0000-000000000001',
        links: { self: '/api/v1/coa-segments/actions/request-segment-changes' },
        data: {
          segmentChangeRequest: {
            id: 'request-001',
            scopeId: 'scope-vietnam-statutory',
            changeType: 'definition',
            subjectId: 'segment-department-operations',
            subjectVersion: 3,
            requestedEffectiveDate: '2026-01-01',
            approvalRequestId: '11111111-1111-4111-8111-111111111111',
            approvalStatus: 'pending',
            applicationStatus: 'not-applied',
            validationOutcome: 'valid',
            nextAction: 'await-approval',
            proposedChange: { name: 'Operations and Shared Services' },
            version: 1,
            revisionNumber: 1,
          },
        },
      }),
    })
  })
  await page.goto('/coa-segments/coa-scr-04?changeType=definition&subjectId=segment-department-operations')

  await expect(page.getByRole('heading', { name: 'Segment change request', level: 2 })).toBeVisible()
  await expect(page.getByRole('note')).toContainText('does not mutate the subject')
  await page.getByRole('button', { name: 'Request definition change' }).click()
  await expect(page.getByRole('alert').filter({ hasText: 'valid Workflow approval-request reference' }).first()).toBeVisible()

  const approvalReference = page.getByRole('textbox', { name: 'Workflow approval-request reference' })
  await approvalReference.focus()
  await expectVisibleFocus(approvalReference)
  await approvalReference.fill('11111111-1111-4111-8111-111111111111')
  await page.getByRole('textbox', { name: 'Name' }).fill('Operations and Shared Services')
  await page.getByRole('button', { name: 'Request definition change' }).click()

  await expect(page.getByRole('textbox', { name: 'Segment-change request reference' })).toHaveValue('request-001')
  await expect(page.getByRole('textbox', { name: 'Approval status' })).toHaveValue('pending')
  await expect(page.getByRole('textbox', { name: 'Application status' })).toHaveValue('not-applied')
  await expect(page.locator('p[role="status"]')).toContainText('subject remains unchanged')
  await expectNoDocumentHorizontalOverflow(page, 'COA-SCR-04 request')
  await expectNoAccessibilityViolations(page, 'COA-SCR-04 request')
})
