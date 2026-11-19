const ITEM_MENU_SELECTOR = '.audio-more,.book-more,.document-more'
const OPEN_ITEM_MENU_SELECTOR = ':is(.audio-more,.book-more,.document-more)[open]'
const ITEM_OWNER_SELECTOR = '.audio-item,.book,.document-card,.document-row'

function setMenuOwnerOpen(menu, open) {
  menu.closest(ITEM_OWNER_SELECTOR)?.classList.toggle('viewport-menu-owner', open)
}

export function viewportMenuPosition(trigger, panel, viewport = { width: innerWidth, height: innerHeight }) {
  const anchor = trigger.getBoundingClientRect()
  const menu = panel.getBoundingClientRect()
  const gap = 6
  const margin = 10
  const width = Math.min(menu.width || 230, viewport.width - margin * 2)
  const left = Math.max(margin, Math.min(anchor.right - width, viewport.width - width - margin))
  const roomBelow = viewport.height - anchor.bottom - margin
  const top = roomBelow >= menu.height || roomBelow >= anchor.top
    ? Math.min(anchor.bottom + gap, viewport.height - menu.height - margin)
    : Math.max(margin, anchor.top - menu.height - gap)
  return { left, top: Math.max(margin, top), width }
}

/** Keeps item action menus beside the invoking item, independent of document scroll. */
export function installViewportActionMenus({ document, window }) {
  let frame = 0
  const position = menu => {
    if (!menu?.open || window.matchMedia('(max-width:680px)').matches) return
    const trigger = menu.querySelector(':scope > summary')
    const panel = menu.querySelector(':scope > div')
    if (!trigger || !panel) return
    menu.classList.add('viewport-menu')
    setMenuOwnerOpen(menu, true)
    const value = viewportMenuPosition(trigger, panel, { width: window.innerWidth, height: window.innerHeight })
    panel.style.setProperty('--menu-left', `${value.left}px`)
    panel.style.setProperty('--menu-top', `${value.top}px`)
    panel.style.setProperty('--menu-width', `${value.width}px`)
  }
  const refresh = () => {
    cancelAnimationFrame(frame)
    frame = requestAnimationFrame(() => document.querySelectorAll(OPEN_ITEM_MENU_SELECTOR).forEach(position))
  }
  const toggle = event => {
    const menu = event.target.closest?.(ITEM_MENU_SELECTOR)
    if (!menu) return
    if (!menu.open) {
      menu.classList.remove('viewport-menu')
      setMenuOwnerOpen(menu, false)
    }
    else refresh()
  }
  document.addEventListener('toggle', toggle, true)
  // Element scroll events do not bubble: capture also covers nested library panels.
  window.addEventListener('scroll', refresh, { passive: true, capture: true })
  window.addEventListener('resize', refresh, { passive: true })
  return () => {
    cancelAnimationFrame(frame)
    document.removeEventListener('toggle', toggle, true)
    window.removeEventListener('scroll', refresh, true)
    window.removeEventListener('resize', refresh)
    document.querySelectorAll(ITEM_MENU_SELECTOR).forEach(menu => {
      menu.classList.remove('viewport-menu')
      setMenuOwnerOpen(menu, false)
    })
  }
}
