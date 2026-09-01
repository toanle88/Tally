import AxeBuilder from '@axe-core/playwright'
import type { Page } from '@playwright/test'

export async function expectNoAccessibilityViolations(page: Page, target: string) {
  const results = await new AxeBuilder({ page }).analyze()

  if (results.violations.length === 0) return

  const details = results.violations.flatMap((violation) => violation.nodes.map((node) => (
    `rule=${violation.id} impact=${violation.impact ?? 'unknown'} help=${violation.help} selectors=${node.target.map(String).join(' ')}`
  )))

  throw new Error([
    `A11Y_FAILURE target=${target}`,
    ...details,
  ].join('\n'))
}
