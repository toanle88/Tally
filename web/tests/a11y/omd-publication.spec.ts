import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

test('OMD-SCR-05 exposes approved publication state and an explicit safe action', async ({ page }) => {
  await page.goto('/master-data/omd-scr-05')

  await expect(page.getByRole('heading', { name: 'Approved master-data publication review', level: 2 })).toBeVisible()
  await expect(page.getByRole('table', { name: 'Master-data publication records' })).toBeVisible()
  await expect(page.getByText('pending approval', { exact: true })).toBeVisible()
  await expect(page.getByText('published', { exact: true })).toBeVisible()
  await expect(page.getByText('rejected', { exact: true })).toBeVisible()
  await expect(page.getByText('stale', { exact: true })).toBeVisible()
  await expect(page.getByText('unavailable', { exact: true }).first()).toBeVisible()
  await expect(page.getByRole('button', { name: 'Publish approved version' })).toHaveCount(1)
  await expectNoAccessibilityViolations(page, 'OMD-SCR-05 publication review')

  await page.getByRole('button', { name: 'Publish approved version' }).click()
  await expect(page.locator('p[role="status"]')).toContainText('was published')
  await expect(page.getByText('LegalEntityPublished v1')).toBeVisible()
  await expectNoAccessibilityViolations(page, 'OMD-SCR-05 published state')
})
