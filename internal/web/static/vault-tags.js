// Tags: the pages carrying one tag, and the list of all tags for the quick switcher.
import { pageName } from './vault-find.js?v=1'

export async function loadTags(library, requestJSON) {
  const { data } = await requestJSON(`/api/documents/${encodeURIComponent(library)}/tags`)
  return Array.isArray(data) ? data : []
}

export function tagPageHTML({ tag, pages, libraryName, escapeHTML, heading = `#${tag}`, crumb = 'Tags', none = 'No pages use this tag.' }) {
  const rows = pages.map(page => {
    const folder = pageName(page).split('/').slice(0, -1).join(' / ')
    return `<li><button type="button" data-document="${escapeHTML(page.id)}"><span>${escapeHTML(page.title)}</span>${folder ? `<small>${escapeHTML(folder)}</small>` : ''}</button></li>`
  }).join('')
  return `<header class="vault-notebar"><div class="vault-notebar-column"><nav class="vault-crumbs" aria-label="Location"><span>${escapeHTML(libraryName || 'Vault')}</span><i aria-hidden="true">/</i><span>${escapeHTML(crumb)}</span></nav></div></header><h1 class="vault-title">${escapeHTML(heading)}</h1><section class="vault-backlinks vault-tag-pages"><h2>Pages <span>${pages.length}</span></h2>${pages.length ? `<ul>${rows}</ul>` : `<p class="vault-empty">${escapeHTML(none)}</p>`}</section>`
}

export async function openTagPage({ pane, tag, library, libraryName, items, requestJSON, escapeHTML }) {
  const wanted = tag.toLowerCase()
  // Own the view before the request, so a slow tag list cannot land on top of a
  // note the reader opened in the meantime.
  pane.className = 'vault-pane document-reader vault-document'
  pane.innerHTML = tagPageHTML({ tag: wanted, pages: [], libraryName, escapeHTML })
  pane.scrollTop = 0
  let groups
  try {
    groups = await loadTags(library, requestJSON)
  } catch (error) {
    if (!pane.querySelector('.vault-tag-pages')) return
    pane.innerHTML = `<p class="vault-empty" role="alert">Could not load tags: ${escapeHTML(error.message)}</p>`
    return
  }
  if (!pane.querySelector('.vault-tag-pages')) return
  const ids = new Set(groups.find(group => group.tag === wanted)?.ids || [])
  const pages = items.filter(item => ids.has(item.id)).sort((a, b) => a.title.localeCompare(b.title, undefined, { numeric: true }))
  pane.innerHTML = tagPageHTML({ tag: wanted, pages, libraryName, escapeHTML })
}

/** Tags matching what was typed after "#", most used first. */
export function rankTags(groups, query, limit = 12) {
  const needle = String(query).replace(/^#/, '').trim().toLowerCase()
  return groups
    .filter(group => group.tag.includes(needle))
    .sort((a, b) => Number(b.tag.startsWith(needle)) - Number(a.tag.startsWith(needle)) || b.ids.length - a.ids.length || a.tag.localeCompare(b.tag))
    .slice(0, limit)
}
