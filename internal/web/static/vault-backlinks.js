// "Linked from": the pages whose [[links]] point at the page being read.
import { pageName } from './vault-find.js?v=1'

export function backlinksHTML(pages, escapeHTML) {
  if (!pages.length) return ''
  const rows = pages.map(page => {
    const folder = pageName(page).split('/').slice(0, -1).join(' / ')
    return `<li><button type="button" data-document="${escapeHTML(page.id)}"><span>${escapeHTML(page.title)}</span>${folder ? `<small>${escapeHTML(folder)}</small>` : ''}</button></li>`
  }).join('')
  return `<section class="vault-backlinks" aria-label="Linked from"><h2>Linked from <span>${pages.length}</span></h2><ul>${rows}</ul></section>`
}

export async function mountBacklinks({ pane, item, library, requestJSON, escapeHTML, isCurrent }) {
  try {
    const { data } = await requestJSON(`/api/documents/${encodeURIComponent(library)}/${encodeURIComponent(item.id)}/backlinks`)
    if (!isCurrent() || !Array.isArray(data)) return
    pane.querySelector('.vault-backlinks')?.remove()
    pane.insertAdjacentHTML('beforeend', backlinksHTML(data, escapeHTML))
  } catch {
    // Backlinks are a nicety; a failure never gets in the way of reading.
  }
}
