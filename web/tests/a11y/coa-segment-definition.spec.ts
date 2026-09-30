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
