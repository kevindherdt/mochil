const imageFormats = new Set(['png', 'jpg', 'jpeg', 'webp', 'gif', 'avif'])

function normalizePath(value) {
  const parts = []
  for (const part of value.split('/')) {
    if (!part || part === '.') continue
    if (part === '..') {
      if (!parts.length) return null
      parts.pop()
    } else parts.push(part)
  }
  return parts.join('/') || null
}

/** One index per vault; never fetch a URL supplied directly by Markdown. */
export function createVaultAssetResolver(entries, libraryId) {
  const exactPaths = new Map(), paths = new Map(), names = new Map()
  const add = (index, key, entry) => {
    if (!index.has(key)) index.set(key, entry)
    else if (index.get(key)?.id !== entry.id) index.set(key, null)
  }
  entries.forEach(entry => {
    if (entry.kind === 'folder' || !imageFormats.has(String(entry.format).toLowerCase())) return
    if (entry.library_id !== libraryId || !entry.id || !entry.path) return
    const path = entry.path.toLowerCase(), name = path.split('/').at(-1)
    add(exactPaths, entry.path, entry)
    add(paths, path, entry)
    add(names, name, entry)
  })
  return (pagePath, target) => {
    const clean = String(target).split('#')[0].split('|')[0].trim().replaceAll('\\', '/')
    if (!clean || clean.startsWith('/') || /^[a-z][a-z0-9+.-]*:/i.test(clean) || /[\u0000-\u001f\u007f]/.test(clean)) return null
    const parent = String(pagePath).split('/').slice(0, -1).join('/')
    const relative = normalizePath(parent ? `${parent}/${clean}` : clean)
    if (!relative) return null
    const candidates = clean.startsWith('./') || clean.startsWith('../')
      ? [relative] : [relative, normalizePath(clean)].filter(Boolean)
    let entry = null, matched = false
    for (const path of candidates) {
      if (exactPaths.has(path)) {
        entry = exactPaths.get(path)
        matched = true
        break
      }
      if (!paths.has(path.toLowerCase())) continue
      entry = paths.get(path.toLowerCase())
      matched = true
      break
    }
    if (!matched && !clean.includes('/')) entry = names.get(clean.toLowerCase())
    if (!entry) return null
    // Construct the authenticated same-origin route; ignore untrusted open_url fields.
    const components = [libraryId, entry.id, entry.path.split('/').at(-1)]
    if (components.some(value => !value || value === '.' || value === '..')) return null
    return {entry, url: `/documents/${components.map(encodeURIComponent).join('/')}`}
  }
}
