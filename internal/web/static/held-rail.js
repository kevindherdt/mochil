import { jumpCardHTML } from './jump-back-in.js?v=24'

/** Books put on hold or dropped, on Home under Finished: folded, and only there when something is shelved. */
export function heldBooks(books, isLibraryVisible = () => true, limit = 48) {
  return books.filter(item => item.status && isLibraryVisible(item.library_id)).slice(0, limit)
}

export function heldCardHTML(item, escapeHTML) {
  const title = escapeHTML(item.title || 'Untitled')
  return `<article class="jump-item rail-card">${jumpCardHTML({ type: 'book', item }, escapeHTML)}<button type="button" class="held-resume" data-library="${escapeHTML(item.library_id)}" data-book="${escapeHTML(item.id)}" aria-label="Move ${title} back to reading">Resume</button></article>`
}

export function createHeldRail({ document, $, escapeHTML, requestJSON, isLibraryVisible, releaseCovers, installCoverFallbacks }) {
  let request = 0
  async function load() {
    const mine = ++request
    const section = $('#held-rail'), track = $('#held-track')
    if (!section || !track) return
    const books = await requestJSON('/api/reading?held=true').then(result => (Array.isArray(result?.data) ? result.data : []), () => [])
    if (mine !== request) return
    const items = heldBooks(books, isLibraryVisible)
    section.hidden = items.length < 1
    $('#held-count').textContent = String(items.length)
    releaseCovers(track)
    track.innerHTML = items.map(item => heldCardHTML(item, escapeHTML)).join('')
    installCoverFallbacks(track)
  }
  document.addEventListener('click', async event => {
    const button = event.target.closest?.('.held-resume')
    if (!button) return
    button.disabled = true
    const response = await fetch(`/api/reading/${encodeURIComponent(button.dataset.library)}/${encodeURIComponent(button.dataset.book)}/status`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ status: 'reading' }) }).catch(() => null)
    if (response?.ok) await load(); else button.disabled = false
  })
  return { load }
}
