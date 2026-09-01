import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

const semanticLabels = [
  'Success',
  'Warning',
  'Error',
  'Info',
  'Pending',
  'Reconciled',
  'Restricted',
  'Disabled',
] as const

test('axe: routed application shell', async ({ page }) => {
  await page.goto('/')

  await expect(page.getByRole('heading', { name: 'TALLY', level: 1 })).toBeVisible()
  await expect(page.getByRole('navigation', { name: 'Global navigation' })).toBeVisible()
  await expect(page.getByRole('main', { name: 'Application content' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Accounting scope context' })).toBeVisible()
  await expect(page.getByRole('region', { name: 'Status feedback' })).toBeVisible()

  await expectNoAccessibilityViolations(page, 'application shell /')
})

test.describe('axe: synthetic integrated examples', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/development/examples')
  })

  test('default state covers shared regions, names, statuses, tables, and live regions', async ({ page }) => {
    await expect(page.getByRole('heading', { name: 'Development examples', level: 2 })).toBeVisible()
    await expect(page.getByRole('table', { name: 'Foundation surfaces' })).toBeVisible()
    await expect(page.getByRole('table', { name: 'Operational work items' })).toBeVisible()
    await expect(page.locator('[aria-live]').first()).toBeAttached()

    for (const label of semanticLabels) {
      await expect(page.getByText(label, { exact: true }).first()).toBeVisible()
    }

    await expectNoAccessibilityViolations(page, 'development examples / default')
  })

  test('confirmation state retains semantic content and accessible controls', async ({ page }) => {
    await page.getByRole('button', { name: 'Review and continue' }).click()

    await expect(page.getByRole('heading', { name: 'Confirm material action', level: 2 })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Confirm and submit' })).toBeVisible()
    await expectNoAccessibilityViolations(page, 'development examples / confirmation')
  })

  test('validation state preserves field associations and accessible error output', async ({ page }) => {
    const businessIdentity = page.getByLabel('Business identity')
    await businessIdentity.fill('')
    await page.getByRole('button', { name: 'Review and continue' }).click()

    await expect(page.locator('[role="alert"]').filter({ hasText: 'Business identity' })).toBeVisible()
    await expect(businessIdentity).toHaveAttribute('aria-invalid', 'true')
    await expect(businessIdentity).toHaveAttribute('aria-describedby', /businessIdentity-error/)
    await expect(businessIdentity).toHaveAttribute('aria-errormessage', 'businessIdentity-error')
    await expectNoAccessibilityViolations(page, 'development examples / validation error')
  })

  test('conflict dialog exposes modal semantics and a safe exit', async ({ page }) => {
    await page.getByRole('button', { name: 'Open conflict example' }).click()

    const dialog = page.getByRole('dialog')
    await expect(dialog).toHaveAttribute('aria-modal', 'true')
    await expect(dialog).toHaveAccessibleName('The record changed before this action')
    await expect(dialog.getByRole('button', { name: 'Close' })).toBeVisible()
    await expectNoAccessibilityViolations(page, 'development examples / conflict dialog')
  })
})
