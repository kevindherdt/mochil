// On iPad and iPhone the on-screen keyboard covers the page without shrinking it, so the browser scrolls the whole page
// up to keep the caret in view and the top bar (with Save/Done) slides under the header. This tells the stylesheet how
// tall the visible area really is, so a full-height view can fit it instead.
const KEYBOARD_MIN = 120

export function keyboardHeight(viewport, layoutHeight) {
  if (!viewport) return 0
  const covered = layoutHeight - viewport.height - viewport.offsetTop
  return covered > KEYBOARD_MIN ? covered : 0
}

export function followKeyboard(win = globalThis.window) {
  const viewport = win?.visualViewport
  if (!viewport) return () => {}
  const root = win.document.documentElement
  const apply = () => {
    const covered = keyboardHeight(viewport, win.innerHeight)
    if (covered) {
      root.style.setProperty('--visible-height', `${Math.round(viewport.height)}px`)
      win.scrollTo(0, 0)
    } else root.style.removeProperty('--visible-height')
  }
  viewport.addEventListener('resize', apply)
  viewport.addEventListener('scroll', apply)
  apply()
  return () => { viewport.removeEventListener('resize', apply); viewport.removeEventListener('scroll', apply); root.style.removeProperty('--visible-height') }
}
