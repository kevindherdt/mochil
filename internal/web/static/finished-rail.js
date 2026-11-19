import { jumpCardHTML } from './jump-back-in.js?v=24'
import { createHeldRail } from './held-rail.js?v=6'

/** Books you have finished, on Home under "Jump back in": a folded list that opens on demand, so Home stays about what is next. */
export function finishedBooks(books, isLibraryVisible = () => true, limit = 24) {
  return books.filter(item => item.finished && !item.dismissed && isLibraryVisible(item.library_id)).slice(0, limit)
}

export function finishedCardHTML(item, escapeHTML) {
  const title = escapeHTML(item.title || 'Untitled')
  return `<article class="jump-item finished-item">${jumpCardHTML({ type: 'book', item }, escapeHTML)}<button type="button" class="finished-restore" data-library="${escapeHTML(item.library_id)}" data-book="${escapeHTML(item.id)}" aria-label="Move ${title} back to in progress">Restore</button></article>`
}

export function createFinishedRail({ document, $, escapeHTML, requestJSON, isLibraryVisible, releaseCovers, installCoverFallbacks }) {
  let request = 0
  const held = createHeldRail({ document, $, escapeHTML, requestJSON, isLibraryVisible, releaseCovers, installCoverFallbacks })
  async function loadFinished() {
    const mine = ++request
    const section = $('#finished-rail'), track = $('#finished-track')
    if (!section || !track) return
    const books = await requestJSON('/api/reading?finished=true&limit=100').then(result => (Array.isArray(result?.data) ? result.data : []), () => [])
    if (mine !== request) return
    const items = finishedBooks(books, isLibraryVisible)
    section.hidden = items.length < 1
    $('#finished-count').textContent = String(items.length)
    releaseCovers(track)
    track.innerHTML = items.map(item => finishedCardHTML(item, escapeHTML)).join('')
    installCoverFallbacks(track)
  }
  {
    document.addEventListener('click', async event => {
      const button = event.target.closest?.('.finished-restore')
      if (!button) return
      button.disabled = true
      const response = await fetch(`/api/reading/${encodeURIComponent(button.dataset.library)}/${encodeURIComponent(button.dataset.book)}/restore`, { method: 'POST' }).catch(() => null)
      if (response?.ok) await loadFinished(); else button.disabled = false
    })
  }
  const load = () => Promise.all([loadFinished(), held.load()])
  return { load }
}
