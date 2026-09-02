/// <reference lib="dom" />

import AxeBuilder from '@axe-core/playwright'
import { expect, type Locator, type Page } from '@playwright/test'

export async function expectNoContrastViolations(page: Page, target: string) {
  const results = await new AxeBuilder({ page })
    .withRules(['color-contrast'])
    .analyze()

  if (results.violations.length === 0) return

  const details = results.violations.flatMap((violation) => violation.nodes.map((node) => (
    `rule=${violation.id} impact=${violation.impact ?? 'unknown'} help=${violation.help} selectors=${node.target.map(String).join(' ')}`
  )))

  throw new Error([
    `VISUAL_CONTRAST_FAILURE target=${target}`,
    ...details,
  ].join('\n'))
}

export async function expectNoDocumentHorizontalOverflow(page: Page, target: string) {
  const metrics = await page.evaluate(() => {
    const viewportWidth = document.documentElement.clientWidth
    const overflowingElements = Array.from(document.querySelectorAll<HTMLElement>('body *'))
      .map((element) => ({ element, rect: element.getBoundingClientRect() }))
      .filter(({ element, rect }) => (
        rect.width > 0
        && rect.right > viewportWidth + 1
        && !element.closest('[data-a11y-scroll-region]')
      ))
      .sort((left, right) => right.rect.right - left.rect.right)
      .slice(0, 5)
      .map(({ element, rect }) => ({
        tag: element.tagName.toLowerCase(),
        className: element.className,
        text: element.textContent?.trim().slice(0, 80) ?? '',
        right: Math.round(rect.right),
      }))

    return {
      documentWidth: document.documentElement.scrollWidth,
      bodyWidth: document.body.scrollWidth,
      viewportWidth,
      overflowingElements,
    }
  })

  expect(metrics.documentWidth, `${target}: document horizontal overflow ${JSON.stringify(metrics.overflowingElements)}`).toBeLessThanOrEqual(metrics.viewportWidth + 1)
  expect(metrics.bodyWidth, `${target}: body horizontal overflow ${JSON.stringify(metrics.overflowingElements)}`).toBeLessThanOrEqual(metrics.viewportWidth + 1)
}

export async function expectVisibleInteractiveElementsFitViewport(page: Page, target: string) {
  const interactive = page.locator('button:visible, input:visible, select:visible, textarea:visible, a:visible')
  const violations = await interactive.evaluateAll((elements) => {
    const viewportWidth = document.documentElement.clientWidth

    return elements.flatMap((element) => {
      if (element.closest('[data-a11y-scroll-region]')) return []

      const rect = element.getBoundingClientRect()
      if (rect.width === 0 || rect.height === 0) return []
      if (rect.left >= -1 && rect.right <= viewportWidth + 1) return []

      return [{
        tag: element.tagName.toLowerCase(),
        label: element.getAttribute('aria-label') ?? element.textContent?.trim().slice(0, 80) ?? '',
        left: Math.round(rect.left),
        right: Math.round(rect.right),
        viewportWidth,
      }]
    })
  })

  expect(violations, `${target}: visible interactive element outside the viewport`).toEqual([])
}

export async function expectLocatorFitsViewport(locator: Locator, target: string) {
  const details = await locator.evaluate((element) => {
    const rect = element.getBoundingClientRect()
    const viewportWidth = document.documentElement.clientWidth
    return {
      left: Math.round(rect.left),
      right: Math.round(rect.right),
      viewportWidth,
      width: Math.round(rect.width),
    }
  })

  expect(details.width, `${target}: expected a visible layout box`).toBeGreaterThan(0)
  expect(details.left, `${target}: left edge is clipped`).toBeGreaterThanOrEqual(-1)
  expect(details.right, `${target}: right edge is clipped`).toBeLessThanOrEqual(details.viewportWidth + 1)
}
