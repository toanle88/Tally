import { expect, test } from '@playwright/test'

import { expectNoAccessibilityViolations } from './accessibility-assertions'

test('OMD-SCR-04 keeps explicit fiscal-calendar maintenance and impact boundary accessible', async ({ page }) => {
  await page.goto('/master-data/omd-scr-04?fiscalCalendarId=fiscal-calendar-vietnam-gregorian')

  await expect(page.getByRole('heading', { name: 'Fiscal-calendar editor', level: 2 })).toBeVisible()
  await expect(page.getByLabel('Calendar type')).toHaveValue('gregorian')
  await expect(page.getByLabel('Period pattern metadata')).toHaveValue('quarterly')
  await expect(page.getByLabel('Period reference').first()).toHaveValue('q1')
  await expect(page.getByText('Impact: unavailable')).toBeVisible()
  await expect(page.getByText('No FPM FiscalPeriod, posting gate, close, or GL record is created or changed by this OMD operation.')).toBeVisible()
  await expectNoAccessibilityViolations(page, 'OMD-SCR-04 fiscal-calendar editor')
})
