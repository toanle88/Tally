import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

test('OMD-WS-01 and OMD-SCR-03 keep vendor-profile policy and Party boundary safe', async ({ page }) => {
  await page.goto('/master-data/omd-ws-01')

  await expect(page.getByRole('heading', { name: 'Legal-entity master-data worklist', level: 2 })).toBeVisible()
  const vendorProfileRecords = page.getByLabel('Safe vendor-profile records')
  await expect(vendorProfileRecords.getByRole('link', { name: 'Acme Industrial Supplies' })).toBeVisible()
  await expect(vendorProfileRecords.getByText('net_30')).toBeVisible()
  await expect(page.locator('body')).not.toContainText('accountNumber')
  await expectNoAccessibilityViolations(page, 'OMD-WS-01 vendor-profile search')

  await vendorProfileRecords.getByRole('link', { name: 'Acme Industrial Supplies' }).click()
  await expect(page.getByRole('heading', { name: 'Vendor profile record', level: 2 })).toBeVisible()
  await expect(page.getByLabel('Authoritative Party')).toHaveValue('Acme Industrial Supplies')
  await expect(page.getByLabel('Party version')).toHaveValue('v5')
  await expect(page.getByRole('button', { name: 'Save vendor-profile revision' })).toBeVisible()
  await expect(page.getByText('Party bank control: approved')).toBeVisible()
  await expect(page.locator('body')).not.toContainText('accountNumber')
  await expectNoAccessibilityViolations(page, 'OMD-SCR-03 vendor-profile record')
})
