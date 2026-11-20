// "Save to server" for a podcast episode: the server downloads the episode itself and keeps it, so it plays without
// the feed and survives the show dropping it. The arrow on each row shows where that stands: not saved, saving,
// saved, or failed (with the reason in the status line under the list).
const cloud = '<path d="M7 18a4 4 0 0 1-.5-7.97A5.5 5.5 0 0 1 17 8.5 4.5 4.5 0 0 1 17 18"/>'
const icons = {
  idle: `${cloud}<path d="M12 11v6m-3-3 3 3 3-3"/>`,
  saving: '<circle class="audio-server-spinner" cx="12" cy="12" r="8" stroke-dasharray="30 20"/>',
  ready: `${cloud}<path d="m9.5 13.5 2 2 3.5-3.5"/>`,
  failed: '<path d="M12 4 3 20h18Z"/><path d="M12 10v4m0 3v.01"/>',
}
const labels = {
  idle: 'Save to the server',
  saving: 'Saving to the server…',
  ready: 'Saved on the server',
  failed: 'Saving failed. Try again',
}

export const serverSaveState = item => item.server_downloaded ? 'ready' : 'idle'

export function serverSaveHTML(item, escapeHTML) {
  const state = serverSaveState(item)
  return `<button class="audio-quick-download audio-server-save" data-action="server-save" data-state="${state}" aria-label="${labels[state]}: ${escapeHTML(item.title)}" title="${labels[state]}"><svg viewBox="0 0 24 24" aria-hidden="true">${icons[state]}</svg></button>`
}

/** Owns the clicks on those buttons; the rows stay owned by the shelf, which may redraw them at any time. */
export function createServerSave({ container, requestJSON, status, wait = ms => new Promise(done => setTimeout(done, ms)), MutationObserverClass = globalThis.MutationObserver, pollMs = 2000, giveUpAfterMs = 15 * 60_000 }) {
  const states = new Map() // episode id -> {state, error}
  const say = text => { if (status) status.textContent = text }
  const titleOf = article => article?.querySelector('strong')?.textContent || 'this episode'

  function paint(button) {
    const id = button.closest('[data-item]')?.dataset.item
    const state = states.get(id)?.state || button.dataset.state
    if (!labels[state]) return
    if (button.dataset.paintedState === state) return
    button.dataset.state = button.dataset.paintedState = state
    button.setAttribute('aria-label', `${labels[state]}: ${titleOf(button.closest('[data-item]'))}`)
    button.title = labels[state]
    button.querySelector('svg').innerHTML = icons[state]
  }
  const paintAll = () => container.querySelectorAll('[data-action="server-save"]').forEach(paint)
  if (MutationObserverClass) new MutationObserverClass(paintAll).observe(container, { childList: true, subtree: true })

  function finish(id, title, state, error = '') {
    states.set(id, { state, error })
    paintAll()
    say(state === 'ready' ? `“${title}” is saved on the server.` : `Could not save “${title}” to the server${error ? `: ${error}` : '.'}`)
  }

  async function save(id, title) {
    states.set(id, { state: 'saving' })
    paintAll()
    say(`Saving “${title}” to the server…`)
    const url = `/api/podcast-downloads/${encodeURIComponent(id)}`
    try {
      let { data } = await requestJSON(url, { method: 'POST' })
      for (let waited = 0; data.status !== 'ready' && data.status !== 'failed'; waited += pollMs) {
        if (waited >= giveUpAfterMs) return finish(id, title, 'failed', 'it is taking too long; check the server')
        await wait(pollMs)
        ;({ data } = await requestJSON(url))
      }
      if (data.status === 'ready') finish(id, title, 'ready')
      else finish(id, title, 'failed', data.error || '')
    } catch (error) {
      finish(id, title, 'failed', error?.status === 404 ? 'server downloads are not available' : error?.status ? error.message : 'the server could not be reached')
    }
  }

  container.addEventListener('click', event => {
    const button = event.target.closest('[data-action="server-save"]')
    if (!button) return
    event.stopImmediatePropagation()
    event.preventDefault()
    const article = button.closest('[data-item]')
    const id = article?.dataset.item, title = titleOf(article)
    if (!id) return
    const state = states.get(id)?.state || button.dataset.state
    if (state === 'saving') return say(`“${title}” is already being saved.`)
    if (state === 'ready') return say(`“${title}” is already saved on the server.`)
    void save(id, title)
  }, true)

  return { states, paintAll }
}
