// "File names" on an album header (administrators only): shows how the album's files would be named from their tags.
// It shows a plan first; files are renamed only after a confirm, and a batch can be undone. Loaded when the music view opens; it adds the button itself when an album opens.
let admin = null // a promise, so every header after the first reuses one check
const isAdmin = () => (admin ||= fetch('/api/hermes/status').then(response => response.ok).catch(() => false))

function ensureStyles() {
  if (document.querySelector('link[data-names-css]')) return
  const link = document.createElement('link')
  link.rel = 'stylesheet'
  link.href = '/assets/vault-polish.css?v=3'
  link.dataset.namesCss = ''
  document.head.append(link)
}

async function openPanel(library, groupId) {
  ensureStyles()
  const overlay = document.createElement('div')
  overlay.className = 'polish-overlay'
  overlay.setAttribute('role', 'dialog')
  overlay.setAttribute('aria-modal', 'true')
  overlay.setAttribute('aria-label', 'File names')
  overlay.innerHTML = '<div class="polish-card"><header><strong>File names</strong><button type="button" class="polish-close" aria-label="Close">×</button></header><div class="polish-body" aria-live="polite"><p>Reading the tags…</p></div><footer></footer></div>'
  document.body.append(overlay)
  const close = () => { document.removeEventListener('keydown', onKey); overlay.remove() }
  const onKey = event => { if (event.key === 'Escape') close() }
  document.addEventListener('keydown', onKey)
  overlay.addEventListener('click', event => { if (event.target === overlay || event.target.closest('.polish-close')) close() })
  const body = overlay.querySelector('.polish-body')
  const foot = overlay.querySelector('footer')
  const say = text => { const p = document.createElement('p'); p.textContent = text; body.replaceChildren(p) }
  const buttons = (...items) => foot.replaceChildren(...items.filter(Boolean).map(([label, handler, primary]) => {
    const b = document.createElement('button')
    b.type = 'button'
    b.textContent = label
    if (primary) b.className = 'polish-primary'
    b.onclick = handler
    return b
  }))
  const query = `library=${encodeURIComponent(library)}&group_id=${encodeURIComponent(groupId)}`
  const send = async (path, payload) => {
    const response = await fetch(`/api/hermes/music-names/${path}`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(payload) })
    const data = await response.json().catch(() => ({}))
    if (!response.ok) throw new Error(data.error || 'That did not work.')
    return data
  }
  const last = () => fetch(`/api/hermes/music-names/last?library=${encodeURIComponent(library)}`).then(r => r.json()).catch(() => ({}))

  async function show(tidy = false) {
    say(tidy ? 'Tidying the tags… with a helper model this can take up to a minute.' : 'Reading the tags…')
    buttons(['Close', close])
    let plan
    try {
      const response = await fetch(`/api/hermes/music-names?${query}${tidy ? '&tidy=1' : ''}`)
      plan = await response.json().catch(() => ({}))
      if (!response.ok) throw new Error(plan.error || 'Could not read this album.')
    } catch (error) { say(error.message); buttons(['Close', close], tidy ? ['Use the tags as they are', () => show(false)] : null); return }
    const summary = document.createElement('p')
    summary.textContent = `${plan.change} will be renamed, ${plan.same} already named this way, ${plan.problem} cannot be named from their tags. Only file names change, inside each file's own folder: tags, cover art and other files stay as they are.${plan.tidied ? ' Messy tags were tidied for these names only.' : ''}`
    const list = document.createElement('ul')
    list.className = 'names-list'
    for (const track of plan.tracks) {
      const item = document.createElement('li')
      item.className = `names-${track.status}`
      const from = document.createElement('div')
      from.className = 'names-from'
      from.textContent = track.from
      const to = document.createElement('div')
      to.className = 'names-to'
      to.textContent = track.status === 'change' ? `→ ${track.to}` : track.status === 'same' ? '✓ already named this way' : `✕ ${track.problem}`
      item.append(from, to)
      for (const note of track.tidy || []) { const n = document.createElement('div'); n.className = 'names-from'; n.textContent = `tidied ${note}`; item.append(n) }
      list.append(item)
    }
    body.replaceChildren(summary, ...(plan.model_note ? [Object.assign(document.createElement('p'), { textContent: plan.model_note })] : []), list)
    const changes = plan.tracks.filter(track => track.status === 'change')
    buttons(['Close', close], plan.undo ? ['Undo the last rename', undo] : null, !plan.tidied ? ['Tidy messy tags', () => show(true)] : null,
      changes.length ? [`Rename ${changes.length} ${changes.length === 1 ? 'file' : 'files'}…`, () => confirm(changes), true] : null)
  }

  function confirm(changes) {
    say(`Rename ${changes.length} ${changes.length === 1 ? 'file' : 'files'} in this album? Queue, favourites, playlists and play progress follow the new names, a record is kept, and you can undo it right after. If anything else reads this folder by file name (another player or sync), it will see the new names after its next scan.`)
    buttons(['Back', () => show(false)], ['Rename now', async () => {
      buttons(['Renaming…', () => {}, true])
      try {
        const done = await send('apply', { library, group_id: groupId, names: changes.map(track => ({ id: track.id, to: track.to })) })
        say(`Renamed ${done.renamed} ${done.renamed === 1 ? 'file' : 'files'}.${done.warning ? ' ' + done.warning : ' The library is rescanning; the album refreshes in a moment.'}`)
        buttons(['Close', close], ['Undo', undo])
      } catch (error) { say(error.message); buttons(['Close', close]) }
    }, true])
  }

  async function undo() {
    const info = await last()
    if (!info.ledger) { say('There is nothing to undo.'); buttons(['Close', close]); return }
    say(`Put ${info.files} ${info.files === 1 ? 'file' : 'files'} back under their old names?`)
    buttons(['Back', () => show(false)], ['Undo', async () => {
      buttons(['Restoring…', () => {}, true])
      try { await send('undo', { library, ledger: info.ledger }); say('Everything is back under its old names. The library is rescanning.'); buttons(['Close', close]) }
      catch (error) { say(error.message); buttons(['Close', close]) }
    }, true])
  }

  show(false)
}

function addButtons() {
  for (const actions of document.querySelectorAll('.music-group-heading.featured .music-group-actions')) {
    if (actions.querySelector('[data-music-names]')) continue
    let link
    try { link = new URL(actions.querySelector('a[href*="audio-group-download"]')?.href || '', location.href) } catch { continue }
    const library = link.searchParams.get('library')
    const groupId = link.searchParams.get('group_id')
    if (!library || !groupId) continue
    isAdmin().then(allowed => {
      if (!allowed || !actions.isConnected || actions.querySelector('[data-music-names]')) return
      const button = document.createElement('button')
      button.type = 'button'
      button.dataset.musicNames = ''
      button.textContent = 'File names'
      button.title = 'Review file names made from the tags. Nothing changes until you confirm, and it can be undone.'
      button.addEventListener('click', () => openPanel(library, groupId))
      actions.append(button)
    })
  }
}

let installed = false
export function installMusicNames() {
  if (installed || typeof MutationObserver === 'undefined' || typeof document === 'undefined') return
  installed = true
  // Scoped to #audio-items (the music/audio view's own item container, see audio-library-template.js) so
  // this doesn't re-scan the whole document on every app-wide DOM mutation (catalogue re-renders, toasts,
  // progress updates...) for the rest of the session -- only mutations inside the music view matter here.
  // Falls back to document.body only in the unlikely case this module runs before that container exists.
  new MutationObserver(addButtons).observe(document.getElementById('audio-items') || document.body, { childList: true, subtree: true })
  addButtons()
}
