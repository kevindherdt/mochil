// The dock: the phone's four places, always within thumb reach. It answers the same three questions the
// masthead does — where am I, where can I go, what is playing — without spending a row at the top of the
// screen on it.
const PLACES = [
  ['home', 'Home', '<path d="M4 11 12 4l8 7v9h-5v-6H9v6H4z" />'],
  ['library', 'Library', '<path d="M5 5h4v14H5zM11 5h4v14h-4zM17 6l3 13-3 1-3-13z" />'],
  ['activity', 'Activity', '<path d="M12 7v5l3 2" /><circle cx="12" cy="12" r="8" />'],
  ['settings', 'Settings', '<path d="M4 7h10M18 7h2M4 17h2M10 17h10" /><circle cx="16" cy="7" r="2" /><circle cx="8" cy="17" r="2" />'],
]

export function createDock({ document: doc = globalThis.document, onPlace, getView, getLibrary }) {
  if (!doc) return { sync() {}, setVisible() {} }
  const nav = doc.getElementById('dock')
  if (!nav) return { sync() {}, setVisible() {} }

  nav.innerHTML = PLACES.map(([place, label, icon]) => `
    <button type="button" class="dock-place" data-dock="${place}" aria-label="${label}">
      <svg viewBox="0 0 24 24" aria-hidden="true">${icon}</svg>
      <span>${label}</span>
    </button>`
  ).join('')

  nav.addEventListener('click', (event) => {
    const button = event.target.closest('[data-dock]')
    if (!button) return
    onPlace(button.dataset.dock)
  })

  function sync() {
    const view = getView?.() ?? 'dashboard'
    const here = view === 'library' ? 'library' : view === 'settings' ? 'settings' : 'home'
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
  }

  function setVisible(visible) {
    nav.hidden = !visible
    measure()
  }

  if (typeof ResizeObserver !== 'undefined') new ResizeObserver(measure).observe(nav)

  sync()
  measure()
  return { sync, setVisible, measure }
}

/**
 * Wire the dock to the app: what each place does, and when it should exist at all. Keeping this here means
 * app.js grows by one call instead of by a dozen lines every time a place is added.
 */
export function mountDock({ getView, getLibrary, go, openSettings, activity, window: win = globalThis.window }) {
  const dock = createDock({
    getView,
    getLibrary,
    onPlace: (place) => {
      if (place === "home") go("dashboard")
      else if (place === "library") go("library")
      else if (place === "settings") openSettings()
      else if (place === "activity") activity()?.click()
      dock.sync()
    },
  })
  const phones = win.matchMedia("(max-width: 760px)")
  dock.setVisible(phones.matches)
  phones.addEventListener("change", (event) => dock.setVisible(event.matches))
  return dock
}
