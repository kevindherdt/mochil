/** Presentation only: preserve YAML verbatim, never evaluate or rewrite it. */
export function splitVaultFrontmatter(value) {
  const source = String(value ?? '')
  const opening = source.match(/^\uFEFF?---(?:\r\n|\r|\n)/)
  if (!opening) return {properties: null, body: source}
  const remainder = source.slice(opening[0].length)
  const closing = /^---(?:\r\n|\r|\n|$)/m.exec(remainder)
  if (!closing) return {properties: null, body: source}
  return {
    properties: remainder.slice(0, closing.index),
    body: remainder.slice(closing.index + closing[0].length),
  }
}

const unquote = value => value.replace(/^(["'])(.*)\1$/, '$2')

/**
 * Read simple front matter (key: value, key: [a, b], or a "- item" list) into rows
 * for display. Anything fancier returns null so the page shows the YAML as written.
 */
export function parseProperties(text) {
  const rows = []
  let open = null
  for (const raw of String(text ?? '').split(/\r\n|\r|\n/)) {
    if (!raw.trim() || raw.trim().startsWith('#')) continue
    const item = /^\s+-\s*(.*)$/.exec(raw) || (open && /^-\s*(.*)$/.exec(raw))
    if (item) {
      if (!open) return null
      open.values.push(unquote(item[1].trim()))
      continue
    }
    const pair = /^([^\s:#][^:]*):(?:\s+(.*)|\s*)$/.exec(raw)
    if (!pair) return null
    const value = (pair[2] ?? '').trim()
    open = null
    if (!value) { open = { key: pair[1].trim(), values: [], list: true }; rows.push(open) }
    else if (/^\[.*\]$/.test(value)) {
      const inner = value.slice(1, -1).trim()
      rows.push({ key: pair[1].trim(), list: true, values: inner ? inner.split(',').map(part => unquote(part.trim())) : [] })
    } else if (/^[|>{&*!]/.test(value)) return null
    else rows.push({ key: pair[1].trim(), list: false, values: [unquote(value)] })
  }
  return rows.length ? rows : null
}
