import { expect, type Locator } from '@playwright/test'

export async function expectVisibleFocus(locator: Locator) {
  await expect(locator).toBeFocused()

  const hasVisibleIndicator = await locator.evaluate((element) => {
    const style = element.ownerDocument.defaultView?.getComputedStyle(element)
    if (!style) return false
    const hasOutline = style.outlineStyle !== 'none' && style.outlineWidth !== '0px'
    const hasShadow = style.boxShadow !== 'none'
    return hasOutline || hasShadow
  })

  expect(hasVisibleIndicator, `Expected ${await locator.getAttribute('aria-label')} to have a visible focus indicator`).toBe(true)
}
