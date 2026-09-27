import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

test('OMD-WS-01 and OMD-SCR-03 keep customer-profile references safe and keyboard-accessible', async ({ page }) => {
  await page.goto('/master-data/omd-ws-01')

  await expect(page.getByRole('heading', { name: 'Legal-entity master-data worklist', level: 2 })).toBeVisible()
  await page.getByLabel('Search master-data records').fill('Northwind')
  const customerProfileRecords = page.getByLabel('Safe customer-profile records')
  await expect(customerProfileRecords.getByRole('link', { name: 'Northwind Distribution' })).toBeVisible()
  await expect(page.getByText('250000 SGD')).toBeVisible()
  await expect(page.locator('body')).not.toContainText('accountNumber')
  await expectNoAccessibilityViolations(page, 'OMD-WS-01 customer-profile search')

  await customerProfileRecords.getByRole('link', { name: 'Northwind Distribution' }).click()
  await expect(page.getByRole('heading', { name: 'Customer profile record', level: 2 })).toBeVisible()
  await expect(page.getByLabel('Authoritative Party')).toHaveValue('Northwind Distribution')
  await expect(page.getByLabel('Party version')).toHaveValue('v2')
  await expect(page.getByRole('button', { name: 'Save customer-profile revision' })).toBeVisible()
  await expectNoAccessibilityViolations(page, 'OMD-SCR-03 customer-profile record')
})
