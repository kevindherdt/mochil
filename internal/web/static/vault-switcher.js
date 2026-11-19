// Quick switcher: jump to any page by typing part of its name, or create one.
import { rankPages, pageName, newPagePath, hasPageNamed } from './vault-find.js?v=1'
import { rankTags } from './vault-tags.js?v=5'

export function mountVaultSwitcher({ document, escapeHTML, source, onOpen, onCreate, onTag }) {
  let overlay = null

  function close() {
    if (!overlay) return
    const { returnTo } = overlay
    overlay.node.remove()
    overlay = null
    returnTo?.focus?.({ preventScroll: true })
  }

  // `custom` turns the box into a picker over its own list of pages: { pages, placeholder, onPick }.
  function open(custom) {
    const context = custom ? { pages: custom.pages, writable: false } : source()
    if (!context || overlay) return
    const node = document.createElement('div')
    node.className = 'vault-switcher'
    node.innerHTML = `<div class="vault-switcher-box" role="dialog" aria-modal="true" aria-label="Jump to page">
      <input type="text" role="combobox" aria-expanded="true" aria-controls="vault-switcher-list" aria-autocomplete="list" autocomplete="off" spellcheck="false" placeholder="Jump to a page, or type # for tags…" />
      <ul id="vault-switcher-list" role="listbox"></ul>
      <p class="vault-switcher-hint" role="status"></p></div>`
    document.body.append(node)
    overlay = { node, returnTo: document.activeElement }
    node.querySelector('input').placeholder = custom?.placeholder || (context.tags ? 'Jump to a page, or type # for tags…' : 'Jump to a page…')
    const input = node.querySelector('input'), list = node.querySelector('ul'), hint = node.querySelector('p')
    let rows = [], chosen = 0, busy = false, tagGroups = null
    const defaultHint = 'Enter opens · Esc closes'

    function draw() {
      const typed = input.value
      if (context.tags && typed.startsWith('#')) {
        if (!tagGroups) {
          tagGroups = []
          context.tags().then(groups => { tagGroups = groups; draw() }).catch(() => { hint.textContent = 'Could not load tags.' })
        }
        rows = rankTags(tagGroups, typed).map(group => ({ tag: group }))
      } else rows = rankPages(context.pages, typed).map(entry => ({ entry }))
      if (!custom && !typed.startsWith('#') && context.writable && typed.trim() && !hasPageNamed(context.pages, typed) && newPagePath(typed)) rows.push({ create: newPagePath(typed) })
      chosen = Math.min(chosen, Math.max(rows.length - 1, 0))
      list.innerHTML = rows.length ? rows.map((row, index) => {
        const selected = index === chosen
        const body = row.tag
          ? `<span>#${escapeHTML(row.tag.tag)}</span><small>${row.tag.ids.length} ${row.tag.ids.length === 1 ? 'page' : 'pages'}</small>`
          : row.create
          ? `<span>Create “${escapeHTML(row.create.replace(/\.md$/i, ''))}”</span><small>New page</small>`
          : `<span>${escapeHTML(row.entry.title)}</span><small>${escapeHTML(pageName(row.entry).split('/').slice(0, -1).join(' / '))}</small>`
        return `<li role="option" id="vault-switcher-${index}" data-row="${index}" aria-selected="${selected}"${row.create ? ' class="create"' : ''}>${body}</li>`
      }).join('') : '<li class="empty" role="presentation">No matching pages</li>'
      input.setAttribute('aria-activedescendant', rows.length ? `vault-switcher-${chosen}` : '')
      list.querySelector('[aria-selected="true"]')?.scrollIntoView({ block: 'nearest' })
    }

    async function choose(index) {
      const row = rows[index]
      if (!row || busy) return
      if (row.tag) { close(); onTag(row.tag.tag); return }
      if (!row.create) { close(); (custom ? custom.onPick : onOpen)(row.entry); return }
      busy = true
      hint.textContent = 'Creating…'
      try {
        await onCreate(row.create)
        close()
      } catch (error) {
        busy = false
        hint.textContent = error.message || 'Could not create that page.'
      }
    }

    input.addEventListener('input', () => { chosen = 0; hint.textContent = defaultHint; draw() })
    node.addEventListener('keydown', event => {
      if (event.key === 'Escape') { event.preventDefault(); close() }
      else if (event.key === 'ArrowDown' || event.key === 'ArrowUp') {
        event.preventDefault()
        if (rows.length) chosen = (chosen + (event.key === 'ArrowDown' ? 1 : rows.length - 1)) % rows.length
        draw()
      } else if (event.key === 'Enter') { event.preventDefault(); choose(chosen) }
      else if (event.key === 'Tab') event.preventDefault()
    })
    node.addEventListener('click', event => {
      const row = event.target.closest('[data-row]')
      if (row) choose(Number(row.dataset.row))
      else if (event.target === node) close()
    })
    hint.textContent = defaultHint
    draw()
    input.focus()
  }

  // Ctrl/Cmd+O everywhere; Ctrl/Cmd+K too, except in the editor where it inserts a link.
  document.addEventListener('keydown', event => {
    if (!(event.metaKey || event.ctrlKey) || event.shiftKey || event.altKey) return
    const key = event.key.toLowerCase()
    const inEditor = event.target.closest?.('.cm-editor')
    if (key !== 'o' && !(key === 'k' && !inEditor)) return
    if (!source()) return
    event.preventDefault()
    open()
  })
  return { open, close }
}
