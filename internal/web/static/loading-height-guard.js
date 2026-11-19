/** Prevents an asynchronously replaced catalogue from collapsing the page mid-navigation. */
export function createLoadingHeightGuard(element) {
  let locked = false

  function lock() {
    if (locked || !element) return
    const height = Math.max(0, Number(element.offsetHeight) || 0)
    if (height > 0) element.style.minHeight = `${height}px`
    locked = true
  }

  function release() {
    if (!locked || !element) return
    element.style.removeProperty('min-height')
    locked = false
  }

  return { lock, release }
}
