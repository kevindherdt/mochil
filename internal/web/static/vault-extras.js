// Wiring for the wiki's navigation helpers: quick switcher, tag pages and
// drag-to-move. The library owns the data; this only connects the pieces.
import { mountVaultSwitcher } from './vault-switcher.js?v=4'
import { mountVaultDrag } from './vault-dnd.js?v=1'
import { openTagPage, loadTags } from './vault-tags.js?v=5'
import { openPropertyPage, parsePropertyHref } from './vault-properties.js?v=3'
import { editableRows, openPropertiesDialog, withProperties } from './vault-properties-edit.js?v=3'
import { newPagePath } from './vault-find.js?v=1'
import { createVaultTabs } from './vault-tabs.js?v=1'
import { vaultHomeHTML, tagRowHTML } from './vault-home.js?v=3'
import { isVaultPage } from './vault-pages.js?v=1'
import { dailyPath, defaultDaily, dailyTemplate, templatePages, expandTemplate, formatDate, DAILY_FOLDER } from './vault-daily.js?v=1'

export const graphButtonHTML = '<button type="button" class="vault-graph-open" data-vault-graph aria-label="Graph view" title="Graph view"><svg viewBox="0 0 24 24" aria-hidden="true"><circle cx="6" cy="7" r="2.2"/><circle cx="18" cy="6" r="2.2"/><circle cx="12" cy="18" r="2.2"/><path d="m8 7.6 8-1.2M7 9l4 7m6-8-4 8"/></svg></button>'

export function mountVaultExtras({ document, $, escapeHTML, requestJSON, library, items, entries, writable, active, openText, reload, expandFolder = () => {} }) {
  const inVault = () => document.body?.classList.contains('vault-reading')
  const pageById = id => items().find(entry => entry.id === id)
  const homeHTML = () => vaultHomeHTML(items(), { message: 'Choose a page from the vault.', writable: writable(), escapeHTML })

  function showTag(tag) {
    const pane = $('#documents .vault-pane')
    if (pane) openTagPage({ pane, tag, library: library(), libraryName: items()[0]?.library_name, items: items(), requestJSON, escapeHTML })
      .catch(error => { pane.innerHTML = `<section class="document-reader-error"><p>${escapeHTML(error.message)}</p></section>` })
  }

  function showProperty({ key, value }) {
    const pane = $('#documents .vault-pane')
    if (pane) openPropertyPage({ pane, key, value, library: library(), libraryName: items()[0]?.library_name, items: items(), requestJSON, escapeHTML })
      .catch(error => { pane.innerHTML = `<section class="document-reader-error"><p>${escapeHTML(error.message)}</p></section>` })
  }

  const switcher = mountVaultSwitcher({
    document,
    escapeHTML,
    source: () => inVault()
      ? { pages: entries().filter(isVaultPage), writable: writable(), tags: () => loadTags(library(), requestJSON) }
      : null,
    onOpen: page => openText(pageById(page.id) || page),
    onTag: showTag,
    onCreate: async path => {
      await requestJSON(`/api/vaults/${encodeURIComponent(library())}/pages`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ path: newPagePath(path), content: '' }),
      })
      await reload()
      const created = items().find(entry => entry.path === path)
      if (created) openText(created)
    },
  })

  const root = $('#documents')
  root.addEventListener('click', async event => {
    if (!event.target.closest?.('[data-edit-properties]') || !writable()) return
    event.preventDefault()
    const item = items().find(entry => entry.id === active())
    if (!item) return
    const { data } = await requestJSON(item.text_url)
    const rows = editableRows(data.content)
    if (!rows) return
    openPropertiesDialog({
      document, rows, escapeHTML,
      save: async next => {
        await requestJSON(`/api/vaults/${encodeURIComponent(item.library_id)}/pages/${encodeURIComponent(item.id)}`, {
          method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ revision: data.edit_revision, content: withProperties(data.content, next) }),
        })
        openText(item)
      },
    })
  })
  root.addEventListener('click', event => {
    const link = event.target.closest?.('a[href^="#prop:"]')
    if (!link) return
    event.preventDefault()
    showProperty(parsePropertyHref(link.getAttribute('href')))
  })
  // The home page lists the vault's tags once they arrive.
  const fillHomeTags = () => {
    const box = root.querySelector('[data-home-tags]:not([data-filled])')
    if (!box) return
    box.dataset.filled = 'true'
    loadTags(library(), requestJSON).then(groups => { const html = tagRowHTML(groups, escapeHTML); if (html && box.isConnected) { box.innerHTML = html; box.hidden = false } }, () => {})
  }
  new MutationObserver(fillHomeTags).observe(root, { childList: true, subtree: true })
  // <details> toggles do not bubble, so listen while the event travels down.
  root.addEventListener('toggle', event => { if (event.target.open && event.target.matches?.('.vault-folder')) expandFolder(event.target) }, true)
  const say = message => {
    const aside = root.querySelector('.vault-workspace aside')
    if (!aside) return
    aside.querySelector('.vault-drag-status')?.remove()
    aside.insertAdjacentHTML('afterbegin', `<p class="vault-drag-status" role="status">${escapeHTML(message)}</p>`)
    setTimeout(() => aside.querySelector('.vault-drag-status')?.remove(), 6000)
  }
  mountVaultDrag({
    root,
    canMove: () => inVault() && writable(),
    findPage: pageById,
    library,
    requestJSON,
    onMoved: async (page, destination) => {
      const wasOpen = active() === page.id
      await reload()
      const moved = items().find(entry => entry.path === destination)
      if (wasOpen && moved) openText(moved)
    },
    onError: say,
  })

  // Folder rename: the pencil on a folder swaps its name for a text box.
  root.addEventListener('click', event => {
    const pencil = event.target.closest?.('[data-vault-rename]')
    if (!pencil || !writable()) return
    event.preventDefault()
    const summary = pencil.closest('summary')
    const label = summary.querySelector('span')
    const path = pencil.dataset.vaultRename
    const parent = path.split('/').slice(0, -1).join('/')
    const input = Object.assign(document.createElement('input'), { value: path.split('/').at(-1), className: 'vault-folder-input', maxLength: 120 })
    input.setAttribute('aria-label', 'Folder name')
    label.replaceWith(input)
    input.focus()
    input.select()
    let finished = false
    const finish = async commit => {
      if (finished) return
      finished = true
      const name = input.value.trim()
      if (!commit || !name || name === path.split('/').at(-1)) { input.replaceWith(label); return }
      const target = parent ? `${parent}/${name}` : name
      try {
        await requestJSON(`/api/vaults/${encodeURIComponent(library())}/folders`, {
          method: 'PATCH', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ from: path, path: target }),
        })
      } catch (error) {
        input.replaceWith(label)
        say(error.message || 'Could not rename that folder.')
        return
      }
      const open = items().find(entry => entry.id === active())
      await reload()
      const moved = open?.path.startsWith(`${path}/`) && items().find(entry => entry.path === target + open.path.slice(path.length))
      if (moved) openText(moved)
    }
    input.addEventListener('keydown', key => {
      if (key.key === 'Enter') { key.preventDefault(); finish(true) }
      else if (key.key === 'Escape') { key.preventDefault(); finish(false) }
      else key.stopPropagation()
    })
    input.addEventListener('blur', () => finish(true))
    input.addEventListener('click', click => { click.preventDefault(); click.stopPropagation() })
  })

  const jsonPost = (path, body) => requestJSON(`/api/vaults/${encodeURIComponent(library())}/${path}`, {
    method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body),
  })
  // Today's note: opened if it exists, otherwise made from Templates/Daily.md (or a plain heading).
  async function openDaily() {
    const today = new Date(), path = dailyPath(today)
    try {
      if (!items().find(entry => entry.path === path)) {
        const template = dailyTemplate(entries())
        const body = template
          ? expandTemplate((await requestJSON(template.text_url)).data.content, { date: today, title: formatDate(today) })
          : defaultDaily(today)
        await jsonPost('folders', { path: DAILY_FOLDER }).catch(error => { if (error.status !== 409) throw error })
        await jsonPost('pages', { path, content: body })
        await reload()
      }
      const page = items().find(entry => entry.path === path)
      if (page) openText(page)
    } catch (error) {
      say(error.message || "Could not open today's note.")
    }
  }
  root.addEventListener('click', event => {
    if (event.target.closest?.('[data-vault-daily]') && writable()) openDaily()
  })

  // The editor's "insert template" button: pick one, get its text with the placeholders filled in.
  const pickTemplate = insert => switcher.open({
    pages: templatePages(entries()),
    placeholder: 'Insert a template…',
    onPick: async page => {
      try {
        const { data } = await requestJSON(page.text_url)
        insert(expandTemplate(data.content, { title: items().find(entry => entry.id === active())?.title || '' }))
      } catch (error) { say(error.message || 'Could not read that template.') }
    },
  })
  pickTemplate.available = () => templatePages(entries()).length > 0

  // Tabs: a plain click replaces the current tab, Ctrl/Cmd-click or middle-click opens a new one.
  const tabs = createVaultTabs({ escapeHTML })
  const pane = () => $('#documents .vault-pane')
  const pageByPath = path => items().find(entry => entry.path === path)
  const pointer = event => { if (event.target.closest?.('[data-document], a[href^="#vault:"]')) tabs.intent(event) }
  root.addEventListener('click', pointer, true)
  root.addEventListener('auxclick', event => {
    const button = event.target.closest?.('[data-document]')
    if (event.button !== 1 || !button) return
    event.preventDefault()
    tabs.intent(event)
    const page = pageById(button.dataset.document)
    if (page) openText(page)
  })
  root.addEventListener('click', event => {
    const close = event.target.closest?.('[data-tab-close]')
    const tab = event.target.closest?.('[data-tab-path]')
    if (close) {
      const wasActive = tabs.active === close.dataset.tabClose
      const next = tabs.close(close.dataset.tabClose)
      if (wasActive && next && pageByPath(next)) openText(pageByPath(next))
      else if (wasActive) { pane().innerHTML = homeHTML(); tabs.mount(pane()) }
      else tabs.mount(pane())
    } else if (tab && pageByPath(tab.dataset.tabPath)) openText(pageByPath(tab.dataset.tabPath))
  })
  /** Called by the library each time a page is shown. */
  function noteOpened(page, shownPane) {
    tabs.use(library())
    tabs.reconcile(entries().filter(isVaultPage))
    tabs.visit(page)
    tabs.mount(shownPane)
  }
  async function openGraph() {
    const target = pane()
    if (!target) return
    const { openGraph: draw } = await import('./vault-graph.js?v=5')
    tabs.use(library())
    await draw({
      pane: target, library: library(), libraryName: items()[0]?.library_name, activeId: active(), requestJSON, escapeHTML,
      onOpen: node => { const page = pageById(node.id); if (page) openText(page) },
      onTabs: shown => tabs.mount(shown),
    })
  }
  root.addEventListener('click', event => { if (event.target.closest?.('[data-vault-graph]')) openGraph() })
  return { openSwitcher: switcher.open, showTag, openDaily, pickTemplate, noteOpened, openGraph }
}
