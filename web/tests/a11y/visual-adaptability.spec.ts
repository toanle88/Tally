/// <reference lib="dom" />

import { expect, test, type Page } from '@playwright/test'

import { expectNoContrastViolations, expectNoDocumentHorizontalOverflow, expectLocatorFitsViewport, expectVisibleInteractiveElementsFitViewport } from './visual-assertions'

const themes = ['finance-light', 'finance-dark'] as const
const adaptationCases = [
  { id: '200-percent-text', width: 1280, height: 720, rootFontSize: '32px' },
  { id: '400-percent-browser-zoom-equivalent', width: 320, height: 900, rootFontSize: '' },
] as const

const surfaceNames = [
  'Worklist and saved filters',
  'Material action form',
  'Approval',
  'Posting',
  'Exception resolution',
  'Result lookup outcome',
  'Process progress',
  'Settlement and reconciliation',
] as const

test.describe('visual adaptability and motion preferences', () => {
  test('covers text and interactive contrast in both finance themes', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 })
    await page.goto('/development/examples')

    for (const theme of themes) {
      await setPageTheme(page, theme)
      await expect(page.getByTestId('app-shell')).toHaveAttribute('data-theme', theme)
      await expect(page.locator('[data-state="restricted"]').first()).toBeVisible()
      await expect(page.getByRole('button', { name: 'Apply settlement' })).toBeDisabled()
      await expect(page.getByRole('button', { name: 'Reclassify clearing' })).toBeDisabled()
      await expectNoContrastViolations(page, `development examples / contrast / ${theme}`)
    }
  })

  for (const adaptationCase of adaptationCases) {
    test(`preserves content and actions at ${adaptationCase.id}`, async ({ page }) => {
      await page.setViewportSize({ width: adaptationCase.width, height: adaptationCase.height })
      await page.goto('/development/examples')
      await page.evaluate((rootFontSize) => {
        document.documentElement.style.fontSize = rootFontSize
      }, adaptationCase.rootFontSize)

      const target = `development examples / ${adaptationCase.id}`
      await expectNoDocumentHorizontalOverflow(page, target)
      await expectVisibleInteractiveElementsFitViewport(page, target)

      for (const surfaceName of surfaceNames) {
        await expectLocatorFitsViewport(page.getByRole('region', { name: surfaceName, exact: true }), `${target} / ${surfaceName}`)
      }

      await expect(page.getByRole('heading', { name: 'Material action form', level: 2 })).toBeAttached()
      await expect(page.getByRole('button', { name: 'Review and continue' })).toBeAttached()
      await expect(page.getByText('Partially completed', { exact: true }).last()).toBeAttached()
      await expect(page.getByText('Reconciliation exception', { exact: true }).last()).toBeAttached()

      const foundationTable = page.getByRole('table', { name: 'Foundation surfaces' })
      const worklist = page.getByRole('table', { name: 'Operational work items' })
      await expect(foundationTable.getByRole('columnheader', { name: 'Surface' })).toBeAttached()
      await expect(foundationTable.getByRole('rowheader', { name: 'Status badge' })).toBeAttached()
      await expect(worklist.getByRole('columnheader', { name: 'Record' })).toBeAttached()
      await expect(worklist.getByRole('rowheader', { name: 'Receipt RCPT-FIX-00042' })).toBeAttached()

      const scrollRegions = page.locator('[data-a11y-scroll-region]')
      await expect(scrollRegions).toHaveCount(3)
      const scrollRegionAttributes = await scrollRegions.evaluateAll((regions) => regions.map((region) => ({
        role: region.getAttribute('role'),
        tabIndex: (region as HTMLElement).tabIndex,
      })))
      expect(scrollRegionAttributes).toEqual([
        { role: 'region', tabIndex: 0 },
        { role: 'region', tabIndex: 0 },
        { role: 'region', tabIndex: 0 },
      ])
      await expect(page.getByRole('region', { name: 'Foundation surfaces', exact: true })).toBeAttached()
      await expect(page.getByRole('region', { name: 'Operational work items', exact: true })).toBeAttached()

      await page.getByRole('button', { name: 'Open conflict example' }).click()
      const dialog = page.getByRole('dialog')
      await expectLocatorFitsViewport(dialog, `${target} / conflict dialog`)
      await expect(dialog.getByRole('button', { name: 'Close' })).toBeVisible()
      await expect(dialog.getByRole('button', { name: 'Refresh authoritative state' })).toBeAttached()
      await expectNoDocumentHorizontalOverflow(page, `${target} / conflict dialog`)
    })
  }

  test('reduces nonessential motion without removing status meaning', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 720 })
    await page.emulateMedia({ reducedMotion: 'no-preference' })
    await page.goto('/development/examples')
    const normalMotion = await readMotionProbe(page)

    await page.emulateMedia({ reducedMotion: 'reduce' })
    const reducedMotion = await readMotionProbe(page)

    expect(reducedMotion.matchesReducedMotion).toBe(true)
    expect(reducedMotion.transitionDurationMs).toBeLessThanOrEqual(0.1)
    expect(reducedMotion.animationDurationMs).toBeLessThanOrEqual(0.1)
    expect(reducedMotion.transitionDurationMs).toBeLessThan(normalMotion.transitionDurationMs)
    expect(reducedMotion.animationDurationMs).toBeLessThan(normalMotion.animationDurationMs)
    expect(reducedMotion.scrollBehavior).toBe('auto')

    await expect(page.getByRole('heading', { name: 'Process progress', level: 2 })).toBeVisible()
    await expect(page.getByText('Partially completed', { exact: true }).last()).toBeVisible()
    await expect(page.getByRole('button', { name: 'Review and continue' })).toBeVisible()
  })
})

async function setPageTheme(page: Page, theme: typeof themes[number]) {
  await page.getByTestId('app-shell').evaluate((shell: HTMLElement, selectedTheme: string) => {
    shell.setAttribute('data-theme', selectedTheme)
    shell.querySelectorAll<HTMLElement>('[data-theme]').forEach((element) => {
      element.setAttribute('data-theme', selectedTheme)
    })
  }, theme)
}

async function readMotionProbe(page: Page) {
  return page.evaluate(() => {
    const keyframes = document.createElement('style')
    keyframes.textContent = '@keyframes a11y-motion-probe { from { opacity: 0.5; } to { opacity: 1; } }'
    document.head.append(keyframes)

    const probe = document.createElement('span')
    probe.className = 'transition-colors'
    probe.style.animation = 'a11y-motion-probe 1s linear infinite'
    probe.setAttribute('aria-hidden', 'true')
    document.body.append(probe)

    const style = getComputedStyle(probe)
    const parseDuration = (value: string) => {
      const firstValue = value.split(',')[0].trim()
      if (firstValue.endsWith('ms')) return Number.parseFloat(firstValue)
      if (firstValue.endsWith('s')) return Number.parseFloat(firstValue) * 1000
      return 0
    }
    const result = {
      matchesReducedMotion: window.matchMedia('(prefers-reduced-motion: reduce)').matches,
      transitionDurationMs: parseDuration(style.transitionDuration),
      animationDurationMs: parseDuration(style.animationDuration),
      scrollBehavior: getComputedStyle(document.documentElement).scrollBehavior,
    }

    probe.remove()
    keyframes.remove()
    return result
  })
}
