/** Decode only our generated wiki-link payload; malformed links stay inert. */
export function parseVaultReference(href) {
  if (!String(href).startsWith('#vault:')) return null
  let target
  try { target = decodeURIComponent(href.slice(7)) } catch { return null }
  const hash = target.indexOf('#')
  return hash < 0 ? {page: target, heading: ''}
    : {page: target.slice(0, hash), heading: target.slice(hash + 1)}
}

const normalizedHeading = value => String(value).normalize('NFC').trim().replace(/\s+/g, ' ').toLowerCase()

/** Search rendered heading text, never interpolate a note's text into a selector. */
export function jumpToVaultHeading(pane, heading) {
  const wanted = normalizedHeading(heading)
  if (!pane || !wanted) return false
  const target = [...pane.querySelectorAll('article h1, article h2, article h3, article h4, article h5, article h6')]
    .find(element => normalizedHeading(element.textContent) === wanted)
  if (!target) return false
  target.setAttribute('tabindex', '-1')
  target.scrollIntoView({block: 'start', behavior: 'instant'})
  target.focus({preventScroll: true})
  return true
}
