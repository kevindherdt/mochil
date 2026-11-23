// The design's top row is a line of library pills -- Books, Audiobooks, TV & Movies, Music, Manga -- with the
// active one filled and no chevron anywhere. The dropdown we shipped is the old button, so on a phone the
// libraries become pills instead of a menu.
import { isNotesShelf } from './shelf-targets.js?v=2'

const KIND_ICON = {
  books: '<path d="M5 5h4v14H5zM11 5h4v14h-4zM17 6l3 13-3 1-3-13z" />',
  audio: '<path d="M4 14v-2a8 8 0 0 1 16 0v2M4 14h3v6H5a1 1 0 0 1-1-1zM20 14h-3v6h2a1 1 0 0 0 1-1z" />',
  video: '<path d="M3 6h12v12H3zM15 10l6-4v12l-6-4z" />',
  manga: '<path d="M4 5h7v15H4zM13 5h7v15h-7z" />',
  documents: '<path d="M6 3h9l4 4v14H6z" /><path d="M9 12h7M9 16h5" />',
}

/** Which icon belongs to a shelf, by kind first and id second: kind is what the server actually knows. */
export function libraryIcon(library) {
  const key = String(library?.kind || '').toLowerCase()
  const id = String(library?.id || '').toLowerCase()
  for (const name of ['video', 'audio', 'manga', 'documents', 'books']) {
    if (key.includes(name)) return KIND_ICON[name]
  }
  if (/movie|series|tv|anime/.test(id)) return KIND_ICON.video
  if (/audio|music|podcast/.test(id)) return KIND_ICON.audio
  if (/manga|comic/.test(id)) return KIND_ICON.manga
  if (isNotesShelf(id)) return KIND_ICON.documents
  return KIND_ICON.books
}

/** A shelf's pill label: the server's name, which is what the reader recognises. */
export function libraryPillLabel(library) {
  return String(library?.name || library?.id || '').trim()
}

export function libraryPillsHTML(libraries, selectedId, escapeHTML) {
  return (libraries || []).map((library) => {
    const active = library.id === selectedId
    return `<button type="button" class="library-pill${active ? ' is-active' : ''}" data-library-id="${escapeHTML(library.id)}" aria-pressed="${active}">`
      + `<svg viewBox="0 0 24 24" aria-hidden="true">${libraryIcon(library)}</svg>`
      + `<span>${escapeHTML(libraryPillLabel(library))}</span></button>`
  }).join('')
}

/** Renders the row and keeps it in step with the shelf you are in. Returns a render() the app can call. */
export function mountLibraryPills({ document, escapeHTML, libraries, selected, onPick }) {
  const nav = document?.getElementById?.('library-pills')
  let lastKey = ''
  function render() {
    if (!nav) return
    const list = libraries() || []
    const active = selected() || ''
    const key = list.map((library) => library.id).join(',') + '|' + active
    if (key === lastKey) return
    lastKey = key
    nav.innerHTML = libraryPillsHTML(list, active, escapeHTML)
    nav.hidden = list.length < 2
  }
  nav?.addEventListener('click', (event) => {
    const button = event.target.closest?.('[data-library-id]')
    if (!button) return
    onPick?.(button.dataset.libraryId)
  })
  return { render }
}
