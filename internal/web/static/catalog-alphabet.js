import { railOverflow } from './destination-rail.js?v=4'

const buckets = ['#', ...'ABCDEFGHIJKLMNOPQRSTUVWXYZ']

export function createCatalogAlphabet({ element, onFilter, onJump }) {
  let jumpMode = false
  let current = ''
  let available = new Set(buckets)

  // At 390px the full A-Z bar is ~3x the viewport (scrollWidth ~1222px vs
  // clientWidth ~386px). overflow-x:auto alone gives no clue more letters are
  // off screen, so flag it on the element for CSS to render an edge fade.
  function updateOverflow() {
    const overflow = railOverflow(element.scrollLeft, element.clientWidth, element.scrollWidth)
    element.classList.toggle('has-letters-before', overflow.before)
    element.classList.toggle('has-letters-after', overflow.after)
  }
  element.addEventListener('scroll', updateOverflow, { passive: true })
  globalThis.ResizeObserver?.prototype && new ResizeObserver(updateOverflow).observe(element)

  function render() {
    element.classList.toggle('alphabet-jump', jumpMode)
    const all = jumpMode ? '' : `<button class="letter ${current === '' ? 'active' : ''}" data-bucket="" aria-pressed="${current === ''}">All</button>`
    element.innerHTML = all + buckets.map(bucket => {
      const enabled = available.has(bucket)
      const label = bucket === '#' ? 'Numbers and symbols' : bucket
      return `<button class="letter ${current === bucket ? 'active' : ''}" data-bucket="${bucket}" aria-label="${label}" aria-pressed="${current === bucket}" ${enabled ? '' : 'disabled'}>${bucket}</button>`
    }).join('')
    updateOverflow()
  }

  element.addEventListener('click', event => {
    const button = event.target.closest('.letter')
    if (!button || button.disabled) return
    // Filtering supports toggling a bucket off to show everything. A jump rail
    // has no "all" destination, so its active letter must remain selected when
    // it is clicked again.
    const selected = jumpMode ? button.dataset.bucket : button.dataset.bucket === current ? '' : button.dataset.bucket
    jumpMode ? onJump(selected) : onFilter(selected)
  })

  // The fixed jump rail is navigation, not a second scrolling surface. Forward
  // wheel movement to the page so its letters remain spatially stable. Wheel
  // events can fire many times per frame, so the actual scroll is batched to
  // at most once per animation frame instead of once per tick.
  let pendingScroll = 0
  let scrollScheduled = false
  function flushPendingScroll() {
    scrollScheduled = false
    const amount = pendingScroll
    pendingScroll = 0
    if (amount) globalThis.window?.scrollBy(0, amount)
  }
  element.addEventListener('wheel', event => {
    if (!jumpMode) return
    event.preventDefault()
    const multiplier = event.deltaMode === 1 ? 16 : event.deltaMode === 2 ? globalThis.window?.innerHeight || 1 : 1
    pendingScroll += event.deltaY * multiplier
    if (!scrollScheduled) {
      scrollScheduled = true
      const schedule = globalThis.window?.requestAnimationFrame || globalThis.requestAnimationFrame
      if (schedule) schedule(flushPendingScroll)
      else flushPendingScroll()
    }
  }, { passive: false })

  function configure({ jump, active = '', navigation = [] }) {
    jumpMode = jump
    current = active
    available = jump ? new Set(navigation.map(item => item.bucket)) : new Set(buckets)
    render()
    if (jumpMode) element.scrollTop = 0
  }

  function setCurrent(bucket) {
    if (!jumpMode || bucket === current) return
    current = bucket
    element.querySelectorAll('.letter').forEach(button => {
      const active = button.dataset.bucket === bucket
      button.classList.toggle('active', active)
      button.setAttribute('aria-pressed', String(active))
    })
  }

  return { configure, setCurrent }
}
