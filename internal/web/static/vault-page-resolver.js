import {isVaultPage} from './vault-pages.js?v=1'

function canonical(value) {
  const parts = []
  for (const part of value.split('/')) {
    if (!part || part === '.') continue
    if (part === '..') {
      if (!parts.length) return null
      parts.pop()
    } else parts.push(part)
  }
  return parts.join('/').replace(/\.(md|markdown)$/i, '') || null
}

function add(index, key, entry) {
  if (!index.has(key)) index.set(key, entry)
  else if (index.get(key)?.id !== entry.id) index.set(key, null)
}

/** Resolve against indexed pages only. Ambiguous fallback names never pick by order. */
export function createVaultPageResolver(entries, libraryId) {
  const exact = new Map(), folded = new Map(), exactNames = new Map(), names = new Map()
  for (const entry of entries) {
    if (!isVaultPage(entry) || entry.library_id !== libraryId || !entry.id || !entry.path) continue
    const path = canonical(entry.path)
    if (!path) continue
    add(exact, path, entry)
    add(folded, path.toLowerCase(), entry)
    add(exactNames, path.split('/').at(-1), entry)
    add(names, path.split('/').at(-1).toLowerCase(), entry)
  }
  return (pagePath, reference) => {
    const target = String(reference).trim().replaceAll('\\', '/')
    if (!target || target.startsWith('/') || /^[a-z][a-z0-9+.-]*:/i.test(target) || /[\u0000-\u001f\u007f]/.test(target)) return null
    const parent = String(pagePath).split('/').slice(0, -1).join('/')
    const relative = canonical(parent ? `${parent}/${target}` : target)
    if (!relative) return null
    const candidates = target.startsWith('./') || target.startsWith('../')
      ? [relative] : [...new Set([relative, canonical(target)].filter(Boolean))]
    for (const candidate of candidates) {
      if (exact.has(candidate)) return exact.get(candidate)
      if (folded.has(candidate.toLowerCase())) return folded.get(candidate.toLowerCase())
    }
    if (target.includes('/')) return null
    const name = canonical(target)
    if (exactNames.has(name)) return exactNames.get(name)
    return names.get(name?.toLowerCase()) || null
  }
}
