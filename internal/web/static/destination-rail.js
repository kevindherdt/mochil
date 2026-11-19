export function railScrollTarget(rail, item, padding = 32) {
  const start = rail.scrollLeft
  const end = start + rail.clientWidth
  const itemStart = item.offsetLeft - padding
  const itemEnd = item.offsetLeft + item.offsetWidth + padding
  if (itemStart < start) return Math.max(0, itemStart)
  if (itemEnd > end) return Math.max(0, itemEnd - rail.clientWidth)
  return start
}

export function railOverflow(scrollLeft, clientWidth, scrollWidth, tolerance = 2) {
  const start = Number(scrollLeft) || 0
  const viewport = Math.max(0, Number(clientWidth) || 0)
  const content = Math.max(0, Number(scrollWidth) || 0)
  return { before: start > tolerance, after: start + viewport < content - tolerance }
}

export function destinationAfterSwipe(destinations, current, deltaX, threshold = 56) {
  const index = destinations.findIndex(([value]) => value === current)
  if (index < 0 || Math.abs(deltaX) < threshold) return current
  const next = Math.max(0, Math.min(destinations.length - 1, index + (deltaX < 0 ? 1 : -1)))
  return destinations[next][0]
}

export function destinationAfterKey(destinations, current, key) {
  const index = destinations.findIndex(([value]) => value === current)
  if (index < 0) return current
  if (key === 'Home') return destinations[0][0]
  if (key === 'End') return destinations[destinations.length - 1][0]
  if (key === 'ArrowLeft') return destinations[Math.max(0, index - 1)][0]
  if (key === 'ArrowRight') return destinations[Math.min(destinations.length - 1, index + 1)][0]
  return current
}

export function createDestinationRail({ element, destinations, attribute, getValue, setValue, onChange, gestureSurface, swipeExclusion = 'button, a, input, select, textarea, summary, [role=slider]' }) {
  element.classList.add('destination-rail')
  // A destination may carry an icon path as a third item: the button then shows the icon, and the label beside it only when
  // it is the current one or there is room (see destination-rail.css).
  element.innerHTML = destinations.map(([value, label, icon]) => icon
    ? `<button type="button" ${attribute}="${value}" aria-pressed="false" aria-label="${label}" title="${label}" class="has-icon"><svg viewBox="0 0 24 24" aria-hidden="true">${icon}</svg><span>${label}</span></button>`
    : `<button type="button" ${attribute}="${value}" aria-pressed="false">${label}</button>`).join('')
  const selector = `[${attribute}]`
  function updateOverflow() {
    const overflow = railOverflow(element.scrollLeft, element.clientWidth, element.scrollWidth)
    element.classList.toggle('has-destinations-before', overflow.before)
    element.classList.toggle('has-destinations-after', overflow.after)
  }
  function sync() {
    element.querySelectorAll(selector).forEach(button => {
      const active = button.getAttribute(attribute) === getValue()
      button.classList.toggle('active', active)
      button.setAttribute('aria-pressed', String(active))
      button.tabIndex = active ? 0 : -1
      if (active) {
        button.setAttribute('aria-current', 'page')
        const left = railScrollTarget(element, button)
        if (left !== element.scrollLeft) element.scrollTo({ left, behavior: 'auto' })
      } else button.removeAttribute('aria-current')
    })
    requestAnimationFrame(updateOverflow)
  }
  async function activate(value) {
    if (!value || value === getValue()) return
    setValue(value)
    sync()
    await onChange()
  }
  element.addEventListener('scroll', updateOverflow, { passive: true })
  globalThis.ResizeObserver?.prototype && new ResizeObserver(updateOverflow).observe(element)
  element.addEventListener('click', event => activate(event.target.closest(selector)?.getAttribute(attribute)))
  element.addEventListener('keydown', async event => {
    if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
    event.preventDefault()
    const value = destinationAfterKey(destinations, getValue(), event.key)
    const button = [...element.querySelectorAll(selector)].find(item => item.getAttribute(attribute) === value)
    button?.focus()
    await activate(value)
  })
  if (gestureSurface) {
    let gesture
    gestureSurface.addEventListener('touchstart', event => {
      const touch = event.touches[0]
      gesture = event.touches.length === 1 && !event.target.closest(swipeExclusion) ? { x: touch.clientX, y: touch.clientY, at: Date.now() } : null
    }, { passive: true })
    gestureSurface.addEventListener('touchend', async event => {
      const start = gesture
      gesture = null
      const touch = event.changedTouches[0]
      if (!start || !touch || Date.now() - start.at > 700) return
      const dx = touch.clientX - start.x
      const dy = touch.clientY - start.y
      if (Math.abs(dx) <= Math.abs(dy) * 1.25) return
      await activate(destinationAfterSwipe(destinations, getValue(), dx))
    }, { passive: true })
    gestureSurface.addEventListener('touchcancel', () => { gesture = null })
  }
  return {
    sync,
    setVisible(visible) {
      element.hidden = !visible
      if (visible) sync()
      else element.classList.remove('has-destinations-before', 'has-destinations-after')
    },
  }
}
