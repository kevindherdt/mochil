// Pages sharing one property value ("status: draft"): the page a property in a note's table links to.
import { tagPageHTML } from './vault-tags.js?v=5'

export async function openPropertyPage({ pane, key, value, library, libraryName, items, requestJSON, escapeHTML }) {
  const { data } = await requestJSON(`/api/documents/${encodeURIComponent(library)}/properties`)
  const group = (Array.isArray(data) ? data : []).find(entry => entry.key === key && entry.value === value.toLowerCase())
  const ids = new Set(group?.ids || [])
  const pages = items.filter(item => ids.has(item.id)).sort((a, b) => a.title.localeCompare(b.title, undefined, { numeric: true }))
  pane.className = 'vault-pane document-reader vault-document'
  pane.innerHTML = tagPageHTML({ tag: value, pages, libraryName, escapeHTML, heading: `${key}: ${value}`, crumb: 'Properties', none: 'No pages have this value.' })
  pane.scrollTop = 0
}

export const propertyHref = (key, value) => `#prop:${encodeURIComponent(key.toLowerCase())}=${encodeURIComponent(value)}`

export function parsePropertyHref(href) {
  const [key, ...rest] = String(href).slice('#prop:'.length).split('=')
  return { key: decodeURIComponent(key || ''), value: decodeURIComponent(rest.join('=')) }
}
