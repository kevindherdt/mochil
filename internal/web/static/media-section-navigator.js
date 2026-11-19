import { createCatalogAlphabet } from './catalog-alphabet.js?v=5'
import { catalogTitleBucket } from './catalog-title-bucket.js'

// Pick the newest row that has crossed the sticky header's reading line. A
// simple "first card whose bottom is visible" check lets a tall card from an
// earlier row retain ownership after later rows have reached the header,
// causing the alphabet highlight to jump backwards in mixed-height grids.
export function currentMediaEntry(items, line) {
  const positioned = items
    .filter(item => item?.element?.isConnected)
    .map((item, order) => ({ item, order, rect: item.element.getBoundingClientRect() }))
    .filter(candidate => Number.isFinite(candidate.rect.top) && Number.isFinite(candidate.rect.bottom) && candidate.rect.bottom > line)
  if (!positioned.length) return null
  const crossed = positioned.filter(candidate => candidate.rect.top <= line)
  if (crossed.length) {
    return crossed.reduce((best, candidate) =>
      candidate.rect.top > best.rect.top + 1 ||
      (Math.abs(candidate.rect.top - best.rect.top) <= 1 && candidate.order < best.order)
        ? candidate : best
    ).item
  }
  return positioned.reduce((best, candidate) =>
    candidate.rect.top < best.rect.top ||
    (candidate.rect.top === best.rect.top && candidate.order < best.order)
      ? candidate : best
  ).item
}

export function createMediaSectionNavigator(element) {
  let entries = []
  let select
  let layoutFrame = 0
  const alphabet = createCatalogAlphabet({
    element,
    onFilter() {},
    onJump: selected => {
      const match = entries.find(entry => entry.bucket === selected)
      if (match) match.element.scrollIntoView({ block: 'start' })
      else select?.(selected)
      alphabet.setCurrent(selected)
    }
  })

  function show(items, options = {}) {
    entries = items.filter(item => item?.element).map(item => ({ ...item, bucket: catalogTitleBucket(item.label) }))
    select = options.onSelect
    const navigation = options.navigation || [...new Set(entries.map(item => item.bucket))].map(bucket => ({ bucket }))
    element.hidden = navigation.length < 2
    if (!element.hidden) {
      alphabet.configure({ jump: true, navigation })
      syncCurrent()
      if (layoutFrame) cancelAnimationFrame(layoutFrame)
      layoutFrame = requestAnimationFrame(() => {
        layoutFrame = 0
        if (!element.hidden) syncCurrent()
      })
    }
  }

  function hide() {
    entries = []
    select = null
    if (layoutFrame) cancelAnimationFrame(layoutFrame)
    layoutFrame = 0
    element.hidden = true
  }

  let frame = 0
  function syncCurrent() {
    const root = globalThis.document?.documentElement
    const styles = root && globalThis.getComputedStyle ? globalThis.getComputedStyle(root) : null
    const top = Number.parseFloat(styles?.getPropertyValue('--catalog-sticky-top')) || 0
    const visible = currentMediaEntry(entries, top + 8)
    if (visible) alphabet.setCurrent(visible.bucket)
  }

  window.addEventListener('scroll', () => {
    if (frame || element.hidden) return
    frame = requestAnimationFrame(() => {
      frame = 0
      syncCurrent()
    })
  }, { passive: true })

  return { show, hide }
}
