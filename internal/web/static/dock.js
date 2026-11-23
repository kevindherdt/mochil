// The dock: a phone's places, always within thumb reach. It answers the same question the masthead does --
// where am I, where can I go -- without spending a row at the top of a small screen on it.
//
// Navigation goes through the shelf menu (#filters) instead of calling four controllers: that menu already knows
// how to open a library, Home, Settings, the wiki and the chat page, and it is the path the rest of the app has
// always used. Calling controllers directly is how Settings and Activity ended up as dead buttons here.
const PLACES = [
  ['library', 'Library', '', '<path d="M5 5h4v14H5zM11 5h4v14h-4zM17 6l3 13-3 1-3-13z" />'],
  ['discover', 'Discover', '__discover__', '<path d="M12 4a8 8 0 1 0 8 8" /><path d="M20 4l-8 8" /><path d="M20 4v5h-5" />'],
  ['hermes', 'Lyana AI', '__hermes__', '<path d="M12 3l2.2 6.3L21 12l-6.8 2.7L12 21l-2.2-6.3L3 12l6.8-2.7z" />'],
  ['notes', 'Notes', '__notes__', '<path d="M6 3h9l4 4v14H6z" /><path d="M9 12h7M9 16h5" />'],
  ['settings', 'Settings', '__settings__', '<path d="M4 7h10M18 7h2M4 17h2M10 17h10" /><circle cx="16" cy="7" r="2" /><circle cx="8" cy="17" r="2" />'],
]

/** Which places the dock has. Lyana is not part of the product: without the module (or without a chat page to
 *  open) the dock stays four buttons and never advertises something that is not there. */
export function dockPlaces({ hermes = false } = {}) {
  return PLACES.filter(([id]) => id !== 'hermes' || hermes)
}

export function createDock({ document: doc = globalThis.document, window: win = globalThis.window, places = dockPlaces(), getView, getLibrary, isNotes = () => false, notesLibrary = () => '', notesShelves = () => [] }) {
  if (!doc) return { sync() {}, setVisible() {}, measure() {}, setPlaces() {} }
  const nav = doc.getElementById('dock')
  if (!nav) return { sync() {}, setVisible() {}, measure() {}, setPlaces() {} }

  let current = places
  const markup = (list) => list.map(([id, label, , icon]) => `
    <button type="button" class="dock-place" data-dock="${id}" aria-label="${label}">
      <svg viewBox="0 0 24 24" aria-hidden="true">${icon}</svg>
      <span>${label}</span>
    </button>`
  ).join('')

  nav.innerHTML = markup(current)

  /** Open a place through the shelf menu, which owns every one of these routes. */
  function open(value) {
    const select = doc.getElementById('filters')
    if (!select) return false
    select.value = value
    select.dispatchEvent(new Event('change', { bubbles: true }))
    return true
  }

  /** Any navigation closes the vault list: it was staying on screen over the library grid. */
  function closeNotesPicker() {
    doc.getElementById('dock-notes-picker')?.remove()
  }

  doc.addEventListener('click', (event) => {
    if (event.target.closest?.('#dock-notes-picker')) return
    if (event.target.closest?.('[data-dock="notes"]')) return
    closeNotesPicker()
  })

  win?.addEventListener?.('hashchange', closeNotesPicker)

  /** Notes, when there is more than one vault to choose from: a short list rather than a guess. */
  function openNotes() {
    const shelves = notesShelves() || []
    const existing = doc.getElementById('dock-notes-picker')
    // A second tap closes the list; it must not also open the first vault behind it.
    if (existing) { existing.remove(); return }
    if (shelves.length <= 1) {
      const target = shelves[0]?.[0] || notesLibrary() || ''
      // A tap that opens nothing is the worst answer: say why instead of leaving the screen where it was.
      if (!target) {
        const status = doc.getElementById('status')
        if (status) status.textContent = 'No notes shelf is set up yet. Add one in Settings, Libraries.'
        return
      }
      open(target)
      return
    }
    const picker = doc.createElement('div')
    picker.id = 'dock-notes-picker'
    picker.className = 'dock-picker'
    picker.setAttribute('role', 'menu')
    picker.innerHTML = shelves.map(([id, name]) => `<button type="button" role="menuitem" data-shelf="${id}">${name}</button>`).join('')
    picker.addEventListener('click', (event) => {
      const choice = event.target.closest('[data-shelf]')
      if (!choice) return
      closeNotesPicker()
      open(choice.dataset.shelf)
      sync()
    })
    nav.before(picker)
  }

  nav.addEventListener('click', (event) => {
    const button = event.target.closest('[data-dock]')
    if (!button) return
    const place = current.find(([id]) => id === button.dataset.dock)
    if (!place) return
    if (button.dataset.dock === 'notes') { openNotes(); return }
    closeNotesPicker()
    let value = place[2]
    if (button.dataset.dock === 'discover') {
      // Discover is the same shelf ordered by what is new rather than by series. The chips own that choice, so the
      // dock opens the shelf first and then presses Discover -- after the re-render, or it presses a stale node.
      value = getLibrary?.() || 'books'
      const press = () => doc.querySelector?.('[data-publication-destination="discover"]')?.click?.()
      if (open(value)) { (win?.requestAnimationFrame || setTimeout)(press) ; sync(); return }
    }
    if (button.dataset.dock === 'library') {
      // Straight to the hub: the shelves are one row of pills there, so no guessing which one you meant.
      // Same event target the toolbar listens on: a CustomEvent fired at window never reaches document.
      doc?.dispatchEvent?.(new (win?.CustomEvent || CustomEvent)('mochil:library-hub'))
      value = getLibrary?.() || 'books'
    }
    if (open(value)) sync()
  })

  function sync() {
    const view = getView?.() ?? 'dashboard'
    // A notes shelf is its own place: opening it from Notes must not light up Library.
    const here = view === 'library' ? (isNotes() ? 'notes' : 'library')
      : view === 'settings' ? 'settings'
      : view === 'hermes' ? 'hermes'
      : view === 'dashboard' ? '' : ''
    for (const button of nav.querySelectorAll('[data-dock]')) {
      const active = button.dataset.dock === here
      button.classList.toggle('is-current', active)
      button.setAttribute('aria-current', active ? 'page' : 'false')
    }
    // The library button points at whatever shelf the reader was last in, so it is one tap back, not a menu.
    const library = nav.querySelector('[data-dock="library"]')
    const shelf = getLibrary?.()
    if (library) library.setAttribute('aria-label', shelf ? `Library, ${shelf}` : 'Library')
  }

  // The player bar is placed on the dock's top edge, so the dock publishes its own height rather than anyone
  // guessing a number that drifts the moment the safe-area inset changes.
  function measure() {
    const height = nav.hidden ? 0 : Math.round(nav.getBoundingClientRect().height)
    doc.documentElement.style.setProperty('--dock-h', `${height}px`)
    measureHiddenBottom()
  }

  // How much of the visible page the browser's own furniture covers at the bottom. Firefox Android puts its
  // navigation bar at the bottom of the screen and lays out the page behind it, so a dock pinned to `bottom: 0`
  // ends up half under the browser chrome while Chrome (toolbar at the top, gesture bar in the safe area) draws it
  // clean. The visual viewport knows the real visible edge in both; the layout viewport does not.
  function measureHiddenBottom() {
    const viewport = window.visualViewport
    const hidden = viewport
      ? Math.max(0, Math.round(window.innerHeight - (viewport.offsetTop + viewport.height)))
      : 0
    doc.documentElement.style.setProperty('--dock-bottom', `${hidden}px`)
    // The phone shell is a fixed frame exactly one screen tall, and `100dvh` is not the same number in every
    // browser: Firefox Android keeps its navigation bar at the bottom and reports the large viewport, so the frame
    // ran on behind the browser's own toolbar and the dock surfaced half-covered while the top of the page got
    // clipped. The visual viewport is the only measurement both engines agree on.
    if (viewport) doc.documentElement.style.setProperty('--visible-height', `${Math.round(viewport.height)}px`)
  }

  function setVisible(visible) {
    nav.hidden = !visible
    measure()
  }

  if (typeof ResizeObserver !== 'undefined') new ResizeObserver(measure).observe(nav)
  // The hidden strip changes as the browser's bar slides away on scroll, so follow the visual viewport too.
  window.visualViewport?.addEventListener('resize', measureHiddenBottom)
  window.visualViewport?.addEventListener('scroll', measureHiddenBottom)
  window.addEventListener('resize', measureHiddenBottom)


  /** A module can be switched on after boot (the availability check is a request), so the places can change. */
  function setPlaces(list) {
    current = list
    nav.innerHTML = markup(current)
    sync()
    measure()
  }

  sync()
  measure()
  return { sync, setVisible, measure, setPlaces }
}

/** Wire the dock to the app: which places exist, and when a phone shows it at all. */
export function mountDock({ hermes = false, notesLibrary, notesShelves, isNotes, getView, getLibrary, window: win = globalThis.window }) {
  const dock = createDock({ places: dockPlaces({ hermes }), getView, getLibrary, isNotes, notesLibrary, notesShelves, window: win })
  const phones = win.matchMedia('(max-width: 760px)')
  dock.setVisible(phones.matches)
  phones.addEventListener('change', (event) => dock.setVisible(event.matches))
  return dock
}
