import { test } from '@playwright/test'

import { expectNoAccessibilityViolations } from '../a11y/accessibility-assertions'

test('A11Y_NEGATIVE_SENTINEL: an unlabeled button fails the axe assertion', async ({ page }) => {
  await page.setContent('<main><button></button></main>')
  await expectNoAccessibilityViolations(page, 'A11Y_NEGATIVE_SENTINEL')
})
