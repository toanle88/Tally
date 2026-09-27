import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

test('OMD-WS-01 and OMD-SCR-02 keep Party fixtures safe and keyboard-accessible', async ({ page }) => {
  await page.goto('/master-data/omd-ws-01')

  await expect(page.getByRole('heading', { name: 'Legal-entity master-data worklist', level: 2 })).toBeVisible()
  await page.getByLabel('Search master-data records').fill('Acme Industrial')
  const partyRecords = page.getByLabel('Safe party records')
  await expect(partyRecords.getByRole('link', { name: 'Acme Industrial Supplies' })).toBeVisible()
  await expect(page.locator('body')).not.toContainText('accountNumber')
  await expectNoAccessibilityViolations(page, 'OMD-WS-01 party search')

  await partyRecords.getByRole('link', { name: 'Acme Industrial Supplies' }).click()
  await expect(page.getByRole('heading', { name: 'Party record', level: 2 })).toBeVisible()
  await expect(page.getByText('provider-ref-001')).toBeVisible()
  await expect(page.getByText('approved', { exact: true }).last()).toBeVisible()
  await expect(page.getByRole('region', { name: 'Restricted tax identifier' }).getByText('••••••••')).toBeVisible()
  await expect(page.locator('body')).not.toContainText('accountNumber')
  await expectNoAccessibilityViolations(page, 'OMD-SCR-02 party record')
})
