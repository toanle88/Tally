import { expect, test } from '@playwright/test'

import { expectVisibleFocus } from './keyboard-assertions'

test.describe('keyboard: shared synthetic examples', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/development/examples')
  })

  test('keeps navigation and form controls in a logical tab order', async ({ page }) => {
    const home = page.getByRole('link', { name: 'Home', exact: true })
    await home.focus()
    await expectVisibleFocus(home)

    await page.keyboard.press('Tab')
    await expect(page.getByRole('link', { name: 'Work', exact: true })).toBeFocused()
    await page.getByRole('link', { name: 'Work', exact: true }).press('Enter')
    await expect(page).toHaveURL(/\/work$/)
    await expect(page.getByRole('heading', { name: 'Work', exact: true })).toBeVisible()

    await page.goto('/development/examples')
    const businessIdentity = page.getByLabel('Business identity')
    const description = page.getByLabel('Intended action')
    const amount = page.getByRole('textbox', { name: 'Amount' }).last()
    const currency = page.locator('#currency')
    const lineReference = page.getByLabel('Line reference')
    const review = page.getByRole('button', { name: 'Review and continue' })

    await businessIdentity.focus()
    await expectVisibleFocus(businessIdentity)
    for (const control of [description, amount, currency, lineReference, review]) {
      await page.keyboard.press('Tab')
      await expect(control).toBeFocused()
    }
  })

  test('changes accounting scope and handles unsaved-work confirmation without a pointer', async ({ page }) => {
    const startWork = page.getByRole('button', { name: 'Start scoped work' })
    await startWork.focus()
    await startWork.press('Enter')

    const scope = page.getByRole('combobox', { name: 'Accounting scope', exact: true })
    await scope.focus()
    await scope.press('ArrowDown')

    const cancel = page.getByRole('button', { name: 'Return to current work' })
    await expect(cancel).toBeVisible()
    await expectVisibleFocus(cancel)
    await cancel.press('Enter')
    await expectVisibleFocus(scope)
    await expect(scope).toHaveValue('scope-vietnam-statutory')

    await scope.press('ArrowDown')
    const discard = page.getByRole('button', { name: 'Discard and switch' })
    await expectVisibleFocus(page.getByRole('button', { name: 'Return to current work' }))
    await discard.press('Enter')
    await expectVisibleFocus(scope)
    await expect(scope).toHaveValue('scope-singapore-management')
    await expect(page.getByRole('region', { name: 'Accounting scope context' }).getByText('Acme Singapore Pte. Ltd. (fixture)', { exact: true })).toBeVisible()
  })

  test('filters, sorts, selects, paginates, and opens worklist rows from the keyboard', async ({ page }) => {
    const scopeFilter = page.getByRole('combobox', { name: 'Scope', exact: true })
    await scopeFilter.focus()
    await scopeFilter.press('Home')
    await expect(scopeFilter).toHaveValue('')

    const stateFilter = page.getByRole('combobox', { name: 'State', exact: true })
    await stateFilter.focus()
    await stateFilter.press('End')
    await expect(page.getByRole('link', { name: 'Receipt RCPT-FIX-00043' })).toBeVisible()
    await stateFilter.press('Home')

    const recordSort = page.getByRole('button', { name: 'Record', exact: true })
    await recordSort.focus()
    await expectVisibleFocus(recordSort)
    await recordSort.press('Enter')
    await expect(page.getByRole('button', { name: /Record \(ascending\)/ })).toBeFocused()

    const firstSelection = page.getByRole('checkbox', { name: 'Select Receipt RCPT-FIX-00042' })
    await firstSelection.focus()
    await expectVisibleFocus(firstSelection)
    await firstSelection.press('Space')
    await expect(page.locator('tr[data-selected="true"]')).toHaveCount(1)

    const bulkAction = page.getByRole('button', { name: 'Assign selected work' })
    await expect(bulkAction).toBeEnabled()
    await bulkAction.focus()
    await bulkAction.press('Enter')
    await expect(page.getByText('Fixture bulk selection accepted for 1 eligible row(s).', { exact: false })).toBeVisible()
    await expect(bulkAction).toBeFocused()

    const nextPage = page.getByRole('button', { name: 'Next page' })
    await nextPage.focus()
    await nextPage.press('Enter')
    await expect(page.getByRole('link', { name: 'Statement STMT-FIX-00008' })).toBeVisible()

    const rowAction = page.getByRole('link', { name: 'Statement STMT-FIX-00008' })
    await rowAction.focus()
    await rowAction.press('Enter')
    await expect(page).toHaveURL(/\/development\/examples#process-example$/)
  })

  test('keeps blocked actions understandable and leaves focus in place for status updates', async ({ page }) => {
    const blockedAction = page.getByRole('button', { name: 'Apply settlement' })
    await expect(blockedAction).toBeDisabled()
    await expect(page.getByText('Owner acknowledgement is required before application.', { exact: true })).toBeVisible()

    const blockedResolution = page.getByRole('button', { name: 'Reclassify clearing' })
    await expect(blockedResolution).toBeDisabled()
    await expect(page.getByText('Independent approval is not recorded. Next: Request the required approval.', { exact: true })).toBeVisible()

    const reviewResult = page.getByRole('button', { name: 'Review result' })
    await reviewResult.focus()
    await reviewResult.press('Enter')
    await expect(reviewResult).toBeFocused()
    await expect(page.getByText('Fixture action selected: Review result. No financial state changed.', { exact: true })).toBeVisible()

    const retry = page.getByRole('button', { name: 'Retry deliberately' })
    await retry.focus()
    await retry.press('Enter')
    await expect(retry).toBeFocused()
    await expect(page.getByText('Retry requires deliberate user action after request lookup; no automatic retry was performed.', { exact: true })).toBeVisible()
    await expect(page.getByRole('heading', { name: 'Approval', exact: true })).toBeVisible()
  })

  test('focuses validation errors, preserves valid input, and follows summary links', async ({ page }) => {
    const businessIdentity = page.getByLabel('Business identity')
    const description = page.getByLabel('Intended action')
    const originalDescription = await description.inputValue()

    await businessIdentity.focus()
    await businessIdentity.press('Control+A')
    await businessIdentity.press('Backspace')
    const review = page.getByRole('button', { name: 'Review and continue' })
    await review.focus()
    await review.press('Enter')

    const summary = page.locator('[role="alert"]').filter({ hasText: 'Business identity' })
    await expectVisibleFocus(summary)
    await expect(description).toHaveValue(originalDescription)
    await expect(page.getByLabel('Amount').last()).toHaveValue('1,250.00')

    const summaryLink = page.getByRole('link', { name: /Business identity:/ }).first()
    await summaryLink.focus()
    await summaryLink.press('Enter')
    await expectVisibleFocus(businessIdentity)
  })

  test('completes confirmation and conflict flows with trapping, cancellation, Escape, and restoration', async ({ page }) => {
    const review = page.getByRole('button', { name: 'Review and continue' })
    await review.focus()
    await review.press('Enter')

    const cancel = page.getByRole('button', { name: 'Cancel and edit' })
    const confirm = page.getByRole('button', { name: 'Confirm and submit' })
    await expectVisibleFocus(cancel)
    await cancel.press('Enter')
    await expectVisibleFocus(review)

    await review.press('Enter')
    await cancel.press('Tab')
    await expectVisibleFocus(confirm)
    await confirm.press('Enter')

    const dialog = page.getByRole('dialog')
    const close = dialog.getByRole('button', { name: 'Close' })
    const refresh = dialog.getByRole('button', { name: 'Refresh authoritative state' })
    const retry = dialog.getByRole('button', { name: 'Retry after review' })
    await expectVisibleFocus(close)

    await close.press('Shift+Tab')
    await expectVisibleFocus(retry)
    await retry.press('Tab')
    await expectVisibleFocus(close)

    await refresh.focus()
    await refresh.press('Enter')
    await expectVisibleFocus(refresh)
    await expect(page.getByText('Authoritative state refreshed: current version 8 is ready for review.', { exact: true })).toBeVisible()

    await page.keyboard.press('Escape')
    await expect(dialog).not.toBeVisible()
    await expectVisibleFocus(review)
  })
})
