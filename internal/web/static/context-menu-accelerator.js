export function installContextMenuAccelerator({ container, matchFinePointer = () => matchMedia('(hover:hover) and (pointer:fine)').matches }) {
  const open = event => {
    if (!matchFinePointer() || event.target.closest('.audio-more')) return
    const item = event.target.closest('.audio-item')
    if (!item || !container.contains(item)) return
    const menu = item.querySelector(':scope > .audio-more')
    if (!menu) return
    event.preventDefault()
    container.querySelectorAll('.audio-more[open]').forEach(value => { if (value !== menu) value.open = false })
    menu.open = true
    menu.querySelector(':scope > summary')?.focus({ preventScroll: true })
  }
  container.addEventListener('contextmenu', open)
  return () => container.removeEventListener('contextmenu', open)
}
