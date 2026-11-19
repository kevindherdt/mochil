// "Polish" on a wiki note: Hermes tidies the formatting as a job on the server, you compare before and after, and nothing
// is saved until you press Apply. The job keeps running if you close the panel; the Polish button then says so, and the
// result waits there. Loaded only when a note opens for someone who can edit it.
import { diffLines, foldUnchanged, countChanges } from './line-diff.js?v=1'

const sparkle = '<svg viewBox="0 0 24 24" aria-hidden="true"><path d="m12 3 1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8zM19 16l.8 2.2L22 19l-2.2.8L19 22l-.8-2.2L16 19l2.2-.8z"/></svg>'
const POLL_MS = 2000

let styled = false
function ensureStyles(doc) {
  if (styled) return
  styled = true
  const link = doc.createElement('link')
  link.rel = 'stylesheet'
  link.href = '/assets/vault-polish.css?v=3'
  doc.head.append(link)
}

async function polishAvailable(fetchImpl) {
  try {
    const response = await fetchImpl('/api/hermes/status')
    if (!response.ok) return false // not an administrator, or the Hermes module is off
    const status = await response.json()
    return status.configured === true && status.features?.polish !== false
  } catch { return false }
}

const jobURL = item => `/api/hermes/polish?library=${encodeURIComponent(item.library_id)}&page=${encodeURIComponent(item.id)}`
const labelFor = state => (state?.status === 'running' ? 'Polishing…' : state?.status === 'ready' ? 'Review polish' : 'Polish')

export async function mountVaultPolish({ pane, item, requestJSON, onDone, fetchImpl = (...args) => fetch(...args), document: doc = pane.ownerDocument }) {
  const actions = pane.querySelector('.vault-page-actions')
  if (!actions || actions.querySelector('[data-vault-polish]')) return null
  if (!(await polishAvailable(fetchImpl)) || !actions.isConnected) return null
  const button = doc.createElement('button')
  button.type = 'button'
  button.dataset.vaultPolish = ''
  const paint = state => {
    button.innerHTML = `${sparkle}<span>${labelFor(state)}</span>`
    button.title = state?.status === 'running' ? 'Hermes is still working on this note. Open to watch or stop it.'
      : state?.status === 'ready' ? 'Hermes has finished: open to compare and apply.'
      : 'Format and polish this note with Hermes. You review the changes first.'
  }
  paint(null)
  actions.prepend(button)
  // A job from an earlier visit (or a closed panel) is still there: say so on the button.
  requestJSON(jobURL(item)).then(({ data }) => { if (button.isConnected) paint(data) }).catch(() => {})
  button.addEventListener('click', () => openPanel({ doc, item, requestJSON, onDone, paint }))
  return button
}

function openPanel({ doc, item, requestJSON, onDone, paint }) {
  ensureStyles(doc)
  const overlay = doc.createElement('div')
  overlay.className = 'polish-overlay'
  overlay.setAttribute('role', 'dialog')
  overlay.setAttribute('aria-modal', 'true')
  overlay.setAttribute('aria-label', 'Polish note')
  const card = doc.createElement('div')
  card.className = 'polish-card'
  const head = doc.createElement('header')
  const title = doc.createElement('strong')
  title.textContent = `Polish “${item.title}”`
  const close = doc.createElement('button')
  close.type = 'button'
  close.className = 'polish-close'
  close.setAttribute('aria-label', 'Close (Hermes keeps working)')
  close.textContent = '×'
  head.append(title, close)
  const body = doc.createElement('div')
  body.className = 'polish-body'
  body.setAttribute('aria-live', 'polite')
  const foot = doc.createElement('footer')
  card.append(head, body, foot)
  overlay.append(card)
  doc.body.append(overlay)

  let timer = null
  let closed = false
  const hide = () => { closed = true; clearTimeout(timer); doc.removeEventListener('keydown', onKey); overlay.remove() }
  const onKey = event => { if (event.key === 'Escape') hide() }
  doc.addEventListener('keydown', onKey)
  close.onclick = hide
  overlay.addEventListener('click', event => { if (event.target === overlay) hide() })

  const buttons = (...items) => {
    foot.replaceChildren(...items.map(([label, handler, primary]) => {
      const b = doc.createElement('button')
      b.type = 'button'
      b.textContent = label
      if (primary) b.className = 'polish-primary'
      b.onclick = handler
      return b
    }))
  }
  const say = (text, kind = '') => { const p = doc.createElement('p'); p.textContent = text; if (kind) p.className = kind; body.replaceChildren(p) }
  const post = (method) => requestJSON('/api/hermes/polish', { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ library: item.library_id, page: item.id }) })

  async function stop() {
    clearTimeout(timer)
    try { await post('DELETE') } catch { /* nothing running */ }
    paint(null)
    hide()
  }

  function render(state) {
    paint(state)
    if (closed) return
    if (state.status === 'running') {
      say(`Hermes is tidying the note… ${state.elapsed} s. On a small model this can take several minutes. You can close this: it keeps working, and the Polish button will say when it is done.`)
      buttons(['Close, keep working', hide], ['Stop', stop])
      timer = setTimeout(poll, POLL_MS)
    } else if (state.status === 'ready') show(state.result)
    else if (state.status === 'failed') {
      say(state.error || 'Could not polish this note.', 'polish-error')
      buttons(['Close', stop], ['Try again', start, true])
    } else start()
  }

  async function poll() {
    try { render((await requestJSON(`/api/hermes/polish?library=${encodeURIComponent(item.library_id)}&page=${encodeURIComponent(item.id)}`)).data) }
    catch (error) {
      if (closed) return
      // The server may be restarting or the connection blipped; the job (if it survived) is picked up again.
      say('Waiting for the server… ' + (error.message || ''))
      buttons(['Close', hide])
      timer = setTimeout(poll, POLL_MS * 2)
    }
  }

  async function start() {
    say('Starting…')
    buttons(['Close', hide])
    try { render((await post('POST')).data) }
    catch (error) {
      if (closed) return
      say(error.message || 'Could not start.', 'polish-error')
      buttons(['Close', hide], ['Try again', start, true])
    }
  }

  function show(data) {
    if (data.same) { say('Nothing to change: this note already looks tidy.'); buttons(['Close', stop]); return }
    const ops = diffLines(data.original, data.proposal)
    const { added, removed } = countChanges(ops)
    const summary = doc.createElement('p')
    summary.className = 'polish-summary'
    summary.textContent = `${added} line${added === 1 ? '' : 's'} added, ${removed} removed. Nothing is saved until you press Apply.`
    const parts = [summary]
    if (data.lost?.length) {
      const warning = doc.createElement('div')
      warning.className = 'polish-warning'
      const lead = doc.createElement('strong')
      lead.textContent = 'Check this before applying: these were in your note but are missing or changed in the result.'
      const list = doc.createElement('ul')
      for (const line of data.lost) { const li = doc.createElement('li'); li.textContent = line; list.append(li) }
      warning.append(lead, list)
      parts.push(warning)
    }
    const diff = doc.createElement('pre')
    diff.className = 'polish-diff'
    diff.tabIndex = 0
    for (const op of foldUnchanged(ops)) {
      const row = doc.createElement('span')
      row.className = `polish-line polish-${op.type}`
      row.textContent = op.type === 'fold' ? `… ${op.count} unchanged line${op.count === 1 ? '' : 's'} …` : `${op.type === 'add' ? '+' : op.type === 'del' ? '−' : ' '} ${op.text}`
      diff.append(row)
    }
    const text = (label, value) => {
      const box = doc.createElement('section')
      box.className = 'polish-text'
      const h = doc.createElement('h4')
      h.textContent = label
      const pre = doc.createElement('pre')
      pre.tabIndex = 0
      pre.textContent = value
      box.append(h, pre)
      return box
    }
    const views = {
      changes: diff,
      before: text('Before (your note now)', data.original),
      after: text('After (what Apply saves)', data.proposal),
    }
    const side = doc.createElement('div')
    side.className = 'polish-side'
    side.append(text('Before (your note now)', data.original), text('After (what Apply saves)', data.proposal))
    views.side = side
    const tabs = doc.createElement('div')
    tabs.className = 'polish-tabs'
    tabs.setAttribute('role', 'tablist')
    const stage = doc.createElement('div')
    stage.className = 'polish-stage'
    const pick = id => {
      for (const [name, el] of Object.entries(views)) el.hidden = name !== id
      for (const t of tabs.children) t.setAttribute('aria-selected', String(t.dataset.view === id))
    }
    for (const [id, label] of [['changes', 'Changes'], ['before', 'Before'], ['after', 'After'], ['side', 'Side by side']]) {
      const t = doc.createElement('button')
      t.type = 'button'
      t.setAttribute('role', 'tab')
      t.dataset.view = id
      if (id === 'side') t.className = 'polish-wide-only'
      t.textContent = label
      t.onclick = () => pick(id)
      tabs.append(t)
      stage.append(views[id])
    }
    pick('changes')
    parts.push(tabs, stage)
    body.replaceChildren(...parts)
    const apply = async () => {
      buttons(['Saving…', () => {}, true])
      try {
        await requestJSON(`/api/vaults/${encodeURIComponent(item.library_id)}/pages/${encodeURIComponent(item.id)}`, {
          method: 'PUT', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ revision: data.revision, content: data.proposal }),
        })
        try { await post('DELETE') } catch { /* already gone */ }
        paint(null)
        hide()
        onDone?.()
      } catch (error) {
        say(error.message || 'Could not save.', 'polish-error')
        buttons(['Close', hide])
      }
    }
    buttons(['Discard', stop], ['Close, decide later', hide], [data.lost?.length ? 'Apply anyway' : 'Apply', apply, true])
  }

  poll()
}
