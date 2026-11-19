import { isVaultPage } from './vault-pages.js?v=1'

const DAY = 86400000

/** "today", "yesterday", "3 days ago", then the date. */
export function sinceText(modified, now = Date.now()) {
  const time = Date.parse(modified)
  if (!Number.isFinite(time)) return ''
  const days = Math.floor((now - time) / DAY)
  if (days <= 0) return 'Today'
  if (days === 1) return 'Yesterday'
  if (days < 7) return `${days} days ago`
  return new Date(time).toLocaleDateString(undefined, { month: 'short', day: 'numeric', year: days > 300 ? 'numeric' : undefined })
}

export function recentPages(entries, limit = 6) {
  return entries.filter(isVaultPage).slice().sort((a, b) => String(b.modified || '').localeCompare(String(a.modified || ''))).slice(0, limit)
}

const ICONS = {
  page: '<path d="M6 3h8l4 4v14H6zM14 3v5h5M12 11v6m-3-3h6"/>',
  daily: '<path d="M5 5h14v15H5zM5 9h14M9 3v4m6-4v4"/>',
  jump: '<circle cx="11" cy="11" r="6"/><path d="m16 16 4 4"/>',
  graph: '<circle cx="6" cy="6" r="2"/><circle cx="18" cy="9" r="2"/><circle cx="9" cy="18" r="2"/><path d="m8 7 8 1.5M7 8l2 8m3-6 5-1"/>',
}
const action = (attrs, icon, label) => `<button type="button" ${attrs}><svg viewBox="0 0 24 24" aria-hidden="true">${ICONS[icon]}</svg><span>${label}</span></button>`

/** What the vault shows before a page is open: ways in, and the pages touched last. With no pages it keeps the plain message. */
export function vaultHomeHTML(entries, { message, writable, escapeHTML, now }) {
  const recent = recentPages(entries)
  if (!recent.length) return `<p class="vault-empty">${message}</p>`
  const actions = [
    writable ? action('data-vault-create="page"', 'page', 'New page') : '',
    writable ? action('data-vault-daily', 'daily', 'Today’s note') : '',
    action('data-vault-switch', 'jump', 'Jump to page'),
    action('data-vault-graph', 'graph', 'Graph'),
  ].join('')
  const rows = recent.map(page => {
    const folder = String(page.path || '').split('/').slice(0, -1).join(' / ')
    return `<li><button type="button" data-document="${escapeHTML(page.id)}"><strong>${escapeHTML(page.title)}</strong><small>${escapeHTML(folder)}</small><time>${escapeHTML(sinceText(page.modified, now))}</time></button></li>`
  }).join('')
  return `<section class="vault-home"><h2>Pick up your notes</h2><div class="vault-home-actions">${actions}</div><div class="vault-home-tags" data-home-tags hidden></div><h3>Recently edited</h3><ul class="vault-recent">${rows}</ul></section>`
}

/** Fills the home's tag row once the tags have loaded: the most used first, each opening the pages that carry it. */
export function tagRowHTML(groups, escapeHTML, limit = 24) {
  const top = groups.slice().sort((a, b) => b.ids.length - a.ids.length || a.tag.localeCompare(b.tag)).slice(0, limit)
  if (!top.length) return ''
  return `<h3>Tags</h3><p>${top.map(group => `<a class="vault-tag" href="#tag:${encodeURIComponent(group.tag)}">#${escapeHTML(group.tag)} <small>${group.ids.length}</small></a>`).join('')}</p>`
}
