// Pressing Library should show the library, not whichever shelf happened to be open last: a hub with the shelves
// as pills, a card per format, and the books you are part-way through. Choosing a pill or a card opens that shelf.
import { libraryIcon, libraryPillLabel } from './library-pills.js?v=1'

const KIND_WORD = {
  books: 'Books',
  audio: 'Audiobooks',
  video: 'Video',
  manga: 'Manga',
  documents: 'Documents',
  comics: 'Comics',
  mixed: 'Mixed',
}

export function hubKindWord(library) {
  const kind = String(library?.kind || '').toLowerCase()
  for (const key of Object.keys(KIND_WORD)) {
    if (kind.includes(key)) return KIND_WORD[key]
  }
  const id = String(library?.id || '').toLowerCase()
  if (/movie|series|tv|anime/.test(id)) return KIND_WORD.video
  if (/audio|music|podcast/.test(id)) return KIND_WORD.audio
  if (/comic/.test(id)) return KIND_WORD.comics
  if (/manga/.test(id)) return KIND_WORD.manga
  if (/note|wiki|document/.test(id)) return KIND_WORD.documents
  return KIND_WORD.books
}

/** "112 items", or nothing when the server does not report a count: no invented numbers. */
export function hubCount(library) {
  const total = Number(library?.count ?? library?.total ?? library?.items)
  if (!Number.isFinite(total) || total <= 0) return ''
  return `${total.toLocaleString()} ${total === 1 ? 'item' : 'items'}`
}

export function hubCardsHTML(libraries, escapeHTML) {
  return (libraries || []).map((library) => {
    const count = hubCount(library)
    return `<button type="button" class="hub-card" data-library-id="${escapeHTML(library.id)}">`
      + `<span class="hub-card-mark"><svg viewBox="0 0 24 24" aria-hidden="true">${libraryIcon(library)}</svg></span>`
      + `<span class="hub-card-copy"><strong>${escapeHTML(libraryPillLabel(library))}</strong>`
      + `<small>${escapeHTML(count || hubKindWord(library))}</small></span></button>`
  }).join('')
}

/**
 * The hub lives beside the catalogue and is shown instead of it. `show()`/`hide()` only toggle visibility, so a
 * reader who steps into a shelf and comes back does not lose the grid they were in.
 */
export function mountLibraryHub({ document, escapeHTML, libraries, onPick }) {
  // Looked up on use, not at mount: this controller is built while the page is still parsing.
  const sectionOf = () => document?.getElementById?.('library-hub')
  const viewOf = () => document?.getElementById?.('library-view')

  function render() {
    const section = sectionOf()
    if (!section) return
    const list = (libraries() || []).filter((library) => !/wiki|documents|vault|notes/i.test(library.id || ''))
    section.innerHTML = `<h1 class="hub-title">Library</h1>`
      + `<h2 class="hub-heading">Reading formats</h2>`
      + `<div class="hub-grid">${hubCardsHTML(list, escapeHTML)}</div>`
    section.querySelectorAll('[data-library-id]').forEach((button) => {
      button.onclick = () => { hide(); onPick?.(button.dataset.libraryId) }
    })
  }

  function show() {
    const section = sectionOf()
    if (!section) return
    render()
    section.hidden = false
    // #library-view is re-shown by the view controller on every navigation, so the hub masks its contents with a
    // class instead of toggling the container: nothing the catalogue re-renders can undo it.
    document?.body?.classList?.add('library-hub-open')
  }

  function hide() {
    const section = sectionOf()
    if (!section) return
    section.hidden = true
    document?.body?.classList?.remove('library-hub-open')
  }

  return { show, hide, render, isOpen: () => !!sectionOf() && !sectionOf().hidden }
}
