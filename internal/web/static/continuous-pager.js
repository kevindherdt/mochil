export function createContinuousPager({ sentinel, button, canLoad, load, autoLoad = () => true, label = 'items', rootMargin = '800px' }) {
  let busy = false
  let failed = false
  let scrollFrame = 0
  const supported = 'IntersectionObserver' in window
  const preload = Number.parseInt(rootMargin, 10) || 800
  const schedule = window.requestAnimationFrame?.bind(window) || (callback => setTimeout(callback, 0))

  const refresh = () => {
    const available = canLoad()
    button.hidden = !available || supported && autoLoad() && !failed
    button.disabled = busy
    if (!busy) button.textContent = `Load more ${label}`
  }

  const request = async (manual = false) => {
    if (busy || !canLoad()) return
    // A failed page waits for the reader: the visible button clears the failure,
    // while scroll, intersection and the interval must not keep retrying it.
    if (failed && !manual) return
    busy = true
    failed = false
    button.disabled = true
    button.textContent = 'Loading…'
    let operation
    try {
      operation = load()
    } catch {
      failed = true
    }
    try {
      if (await operation === false) failed = true
    } catch {
      failed = true
    } finally {
      busy = false
      refresh()
      if (!failed) schedule(checkPosition)
    }
  }

  button.onclick = () => request(true)
  const observer = supported ? new IntersectionObserver(entries => {
    if (failed) return
    if (autoLoad() && entries.some(entry => entry.isIntersecting)) request()
  }, { rootMargin }) : null
  const checkPosition = () => {
    scrollFrame = 0
    if (failed || !autoLoad() || !canLoad()) return
    const bounds = sentinel.getBoundingClientRect()
    if (bounds.top <= window.innerHeight + preload && bounds.bottom >= -preload) request()
  }
  const onScroll = () => { if (!scrollFrame) scrollFrame = schedule(checkPosition) }
  observer?.observe(sentinel)
  window.addEventListener?.('scroll', onScroll, { passive: true })
  const positionTimer = setInterval(checkPosition, 500)
  refresh()

  return { refresh, retry: () => request(true), disconnect: () => { observer?.disconnect(); clearInterval(positionTimer); window.removeEventListener?.('scroll', onScroll) } }
}
