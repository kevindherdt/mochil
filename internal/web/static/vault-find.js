// Finding wiki pages by name: shared by the quick switcher and [[ suggestions.
export const pageName = entry => String(entry.path || '').replace(/\.(md|markdown)$/i, '')

const folded = value => String(value || '').toLowerCase()

function subsequence(text, query) {
  let at = 0
  for (const char of text) if (char === query[at] && ++at === query.length) return true
  return false
}

// Lower is better; null means the page does not match.
function score(entry, query) {
  const title = folded(entry.title || pageName(entry).split('/').at(-1))
  const path = folded(pageName(entry))
  if (title === query) return 0
  if (title.startsWith(query)) return 1
  if (title.split(/[\s\-_.]+/).some(word => word.startsWith(query))) return 2
  if (title.includes(query)) return 3
  if (path.includes(query)) return 4
  if (subsequence(title, query)) return 5
  return null
}

export function rankPages(pages, query, limit = 12) {
  const needle = folded(query).trim()
  const ranked = []
  for (const entry of pages) {
    const value = needle ? score(entry, needle) : 0
    if (value !== null) ranked.push({ entry, value })
  }
  ranked.sort((a, b) => a.value - b.value || pageName(a.entry).length - pageName(b.entry).length
    || pageName(a.entry).localeCompare(pageName(b.entry)))
  return ranked.slice(0, limit).map(row => row.entry)
}

/** The path a typed name would create, or null when it is not a usable page name. */
export function newPagePath(typed) {
  const path = String(typed).trim().replaceAll('\\', '/').replace(/^\/+/, '')
  if (!path || path.split('/').some(part => !part.trim() || part === '.' || part === '..')) return null
  return /\.(md|markdown)$/i.test(path) ? path : `${path}.md`
}

export function hasPageNamed(pages, typed) {
  const path = newPagePath(typed)
  if (!path) return true
  const wanted = folded(path)
  return pages.some(entry => folded(entry.path) === wanted
    || (!path.includes('/') && folded(entry.title) === folded(path.replace(/\.(md|markdown)$/i, ''))))
}
