// Tabs for open wiki pages. Plain clicks reuse the current tab, Ctrl/Cmd-click
// or middle-click opens a new one, and the set is remembered per vault.
const MAX_TABS = 12
const key = library => `bobobooks.vault.tabs.${library}`

export function createVaultTabs({ storage = globalThis.localStorage, escapeHTML }) {
  let library = '', tabs = [], active = '', wantNew = false

  const save = () => { try { storage?.setItem(key(library), JSON.stringify({ tabs, active })) } catch { /* private mode */ } }

  /** Switches to a vault, bringing back the tabs it had. */
  function use(libraryId) {
    if (libraryId === library) return
    library = libraryId
    tabs = []; active = ''
    try {
      const saved = JSON.parse(storage?.getItem(key(libraryId)) || 'null')
      if (Array.isArray(saved?.tabs)) tabs = saved.tabs.filter(tab => typeof tab?.path === 'string').slice(0, MAX_TABS)
      if (typeof saved?.active === 'string') active = saved.active
    } catch { /* unreadable: start empty */ }
  }

  /** Drops tabs for pages that no longer exist and refreshes titles. */
  function reconcile(pages) {
    const byPath = new Map(pages.map(page => [page.path, page]))
    tabs = tabs.filter(tab => byPath.has(tab.path)).map(tab => ({ path: tab.path, title: byPath.get(tab.path).title }))
    if (!tabs.some(tab => tab.path === active)) active = ''
  }

  const intent = (event) => { wantNew = Boolean(event && (event.ctrlKey || event.metaKey || event.button === 1)) }

  /** Records that a page was opened: in a new tab, the same tab, or an existing one. */
  function visit(page) {
    const open = tabs.find(tab => tab.path === page.path)
    if (open) active = open.path
    else {
      const entry = { path: page.path, title: page.title }
      const at = tabs.findIndex(tab => tab.path === active)
      if (wantNew || at < 0) tabs.push(entry)
      else tabs[at] = entry
      active = entry.path
      if (tabs.length > MAX_TABS) tabs.splice(tabs.findIndex(tab => tab.path !== active), 1)
    }
    wantNew = false
    save()
  }

  /** Closes a tab; returns the path to show next ('' when none are left). */
  function close(path) {
    const at = tabs.findIndex(tab => tab.path === path)
    if (at < 0) return active
    tabs.splice(at, 1)
    if (active === path) active = (tabs[at] || tabs[at - 1])?.path || ''
    save()
    return active
  }

  function html() {
    if (!tabs.length) return ''
    return `<div class="vault-tabs" role="tablist" aria-label="Open pages">${tabs.map(tab => {
      const selected = tab.path === active
      return `<span class="vault-tab${selected ? ' active' : ''}"><button type="button" role="tab" aria-selected="${selected}" data-tab-path="${escapeHTML(tab.path)}" title="${escapeHTML(tab.path)}">${escapeHTML(tab.title)}</button><button type="button" class="vault-tab-close" data-tab-close="${escapeHTML(tab.path)}" aria-label="Close ${escapeHTML(tab.title)}">×</button></span>`
    }).join('')}</div>`
  }

  /** Puts the strip at the top of a pane. */
  function mount(pane) {
    pane.querySelector('.vault-tabs')?.remove()
    pane.classList.toggle('has-tabs', tabs.length > 0)
    if (tabs.length) pane.insertAdjacentHTML('afterbegin', html())
    pane.querySelector('.vault-tab.active')?.scrollIntoView?.({ block: 'nearest', inline: 'nearest' })
  }

  return { use, reconcile, intent, visit, close, html, mount, get paths() { return tabs.map(tab => tab.path) }, get active() { return active } }
}
