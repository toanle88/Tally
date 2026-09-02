import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

test.describe('semantic and screen-reader review proxy coverage', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/development/examples')
  })

  test('exposes landmarks, labels, table identity, states, and debit/credit text', async ({ page }) => {
    await expect(page.getByRole('main')).toBeVisible()
    await expect(page.getByRole('navigation')).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Development examples', level: 2 })).toBeVisible()
    await expect(page.getByRole('textbox', { name: 'Example field' })).toHaveAttribute('aria-describedby', /foundation-example-description/)

    const foundationTable = page.getByRole('table', { name: 'Foundation surfaces' })
    await expect(foundationTable.getByRole('columnheader', { name: 'Surface' })).toBeVisible()
    await expect(foundationTable.getByRole('rowheader', { name: 'Status badge' })).toBeVisible()

    const worklist = page.getByRole('table', { name: 'Operational work items' })
    await expect(worklist.getByRole('columnheader', { name: 'Record' })).toBeVisible()
    await expect(worklist.getByRole('rowheader', { name: 'Receipt RCPT-FIX-00042' })).toBeVisible()
    const selection = page.getByRole('checkbox', { name: 'Select Receipt RCPT-FIX-00042' })
    await expect(selection).not.toBeChecked()
    await selection.check()
    await expect(selection).toBeChecked()

    for (const label of ['Success', 'Warning', 'Error', 'Pending', 'Reconciled', 'Restricted']) {
      await expect(page.getByText(label, { exact: true }).first()).toBeVisible()
    }
    await expect(page.getByRole('cell', { name: 'Approval pending' })).toBeVisible()
    await expect(page.getByRole('cell', { name: 'Approved' }).first()).toBeVisible()
    await expect(page.getByText('Reconciliation exception', { exact: true }).last()).toBeVisible()
    await expect(page.getByText('Negative means loss; positive means gain.')).toBeVisible()
    await expect(page.getByRole('rowheader', { name: 'Debit' })).toBeVisible()
    await expect(page.getByRole('rowheader', { name: 'Credit' })).toBeVisible()
    await expect(page.getByText(/Debit line direction is stated explicitly in text/)).toBeVisible()
    await expect(page.getByText(/Credit line direction is stated explicitly in text/)).toBeVisible()

    await expectNoAccessibilityViolations(page, 'semantic development examples')
  })

  test('keeps restricted values out of accessible names, errors, and empty states', async ({ page }) => {
    const restrictedRegion = page.getByRole('region', { name: 'Restricted provider reference' })
    await expect(restrictedRegion).toContainText('••••••••')
    await expect(restrictedRegion).not.toContainText('SYNTHETIC-RESTRICTED-REF')
    await expect(page.getByRole('button', { name: 'Reveal unavailable' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Export unavailable' })).toBeVisible()

    await page.locator('#worklist-amount-filter').fill('999.00')
    await expect(page.getByText('No matching work items.')).toBeVisible()
    await expect(page.locator('body')).not.toContainText('SYNTHETIC-RESTRICTED-REF')

    await page.getByLabel('Business identity').fill('')
    await page.getByRole('button', { name: 'Review and continue' }).click()
    const validation = page.getByRole('alert').filter({ hasText: 'Business identity' })
    await expect(validation).toBeVisible()
    await expect(validation).not.toContainText('SYNTHETIC-RESTRICTED-REF')

    await expectNoAccessibilityViolations(page, 'restricted value privacy')
  })

  test('announces dynamic status and exposes dialog and progress semantics', async ({ page }) => {
    await expect(page.locator('[aria-live]')).not.toHaveCount(0)
    await page.getByRole('button', { name: 'Review result' }).click()
    await expect(page.getByText('Fixture action selected: Review result.')).toBeVisible()

    await page.getByRole('button', { name: 'Open conflict example' }).click()
    const dialog = page.getByRole('dialog', { name: 'The record changed before this action' })
    await expect(dialog).toBeVisible()
    await expect(dialog).toHaveAttribute('aria-describedby', /-conflict-description/)
    await expect(dialog.getByText('Expected')).toBeVisible()
    await expect(dialog.getByText('Current: 1,275.00')).toBeVisible()

    await expect(page.getByRole('heading', { name: 'Process progress', level: 2 })).toBeVisible()
    await expect(page.getByText('Current', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('Failed', { exact: true }).first()).toBeVisible()
    await expect(page.getByText('Reconciled', { exact: true }).first()).toBeVisible()

    await expectNoAccessibilityViolations(page, 'dynamic status, dialog, and progress')
  })
})
