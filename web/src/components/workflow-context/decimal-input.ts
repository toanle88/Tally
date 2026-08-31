function localeSeparators(locale: string) {
  const parts = new Intl.NumberFormat(locale).formatToParts(1234567.89)
  return {
    group: parts.find((part) => part.type === 'group')?.value ?? ',',
    decimal: parts.find((part) => part.type === 'decimal')?.value ?? '.',
  }
}

export function normalizeDecimalInput(value: string, locale: string): string | null {
  const { group, decimal } = localeSeparators(locale)
  const normalized = value.trim().split(group).join('').replace(decimal, '.')
  if (!/^-?\d+(?:\.\d+)?$/.test(normalized)) return null

  const sign = normalized.startsWith('-') ? '-' : ''
  const unsigned = sign ? normalized.slice(1) : normalized
  const [integer, fraction] = unsigned.split('.')
  const canonicalInteger = integer.replace(/^0+(?=\d)/, '')
  return `${sign}${canonicalInteger}${fraction === undefined ? '' : `.${fraction}`}`
}

export function formatDecimalInput(canonical: string, locale: string): string {
  const normalized = normalizeDecimalInput(canonical, 'en-US')
  if (!normalized) return canonical

  const { group, decimal } = localeSeparators(locale)
  const sign = normalized.startsWith('-') ? '-' : ''
  const unsigned = sign ? normalized.slice(1) : normalized
  const [integer, fraction] = unsigned.split('.')
  const grouped = integer.replace(/\B(?=(\d{3})+(?!\d))/g, group)
  return `${sign}${grouped}${fraction === undefined ? '' : `${decimal}${fraction}`}`
}
