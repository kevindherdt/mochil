import test from 'node:test'
import assert from 'node:assert/strict'
import { createCatalogAlphabet } from '../static/catalog-alphabet.js'

function element() {
  return {
    classList: { toggle() {} }, innerHTML: '', scrollTop: 0, listeners: {}, addEventListener(type, listener) { this.listeners[type] = listener },
    querySelectorAll() { return [] }
  }
}

// 390px-wide phones show the full A-Z bar, which is ~3x the viewport (scrollWidth
// 1222 vs clientWidth 386) with no fade/arrow/visible-scrollbar affordance today,
// so most letters are undiscoverable. The rail must flag that overflow so CSS can
// render a scroll indicator, and keep the flag correct as the user scrolls.
function overflowElement({ scrollWidth = 1222, clientWidth = 386, scrollLeft = 0 } = {}) {
  const classes = new Set()
  return {
    classList: {
      toggle(name, on) { on ? classes.add(name) : classes.delete(name) },
      contains(name) { return classes.has(name) },
    },
    classes,
    innerHTML: '',
    scrollTop: 0,
    scrollWidth,
    clientWidth,
    scrollLeft,
    listeners: {},
    addEventListener(type, listener) { this.listeners[type] = listener },
    querySelectorAll() { return [] },
  }
}

test('flags overflow on a narrow viewport so a scroll affordance can render', () => {
  const root = overflowElement({ scrollWidth: 1222, clientWidth: 386, scrollLeft: 0 })
  createCatalogAlphabet({ element: root, onFilter() {}, onJump() {} }).configure({ jump: false })
  assert.equal(root.classes.has('has-letters-after'), true, 'more letters are scrolled off to the right')
  assert.equal(root.classes.has('has-letters-before'), false, 'already at the start, nothing hidden to the left')
})

test('clears the overflow flag once every letter fits on screen', () => {
  const root = overflowElement({ scrollWidth: 300, clientWidth: 386, scrollLeft: 0 })
  createCatalogAlphabet({ element: root, onFilter() {}, onJump() {} }).configure({ jump: false })
  assert.equal(root.classes.has('has-letters-after'), false)
  assert.equal(root.classes.has('has-letters-before'), false)
})

test('re-evaluates overflow as the rail is scrolled', () => {
  const root = overflowElement({ scrollWidth: 1222, clientWidth: 386, scrollLeft: 0 })
  createCatalogAlphabet({ element: root, onFilter() {}, onJump() {} }).configure({ jump: false })
  assert.equal(root.classes.has('has-letters-before'), false)
  root.scrollLeft = 900
  root.listeners.scroll()
  assert.equal(root.classes.has('has-letters-before'), true, 'scrolled past the start, earlier letters are hidden')
  assert.equal(root.classes.has('has-letters-after'), false, 'scrolled to the end, nothing left to reveal')
})

test('disables unavailable jump letters and keeps available letters actionable', () => {
  const root = element()
  createCatalogAlphabet({ element: root, onFilter() {}, onJump() {} }).configure({ jump: true, navigation: [{ bucket: 'A' }, { bucket: 'M' }] })
  assert.match(root.innerHTML, /data-bucket="A"[^>]*>A/)
  assert.match(root.innerHTML, /data-bucket="B"[^>]*disabled>B/)
  assert.doesNotMatch(root.innerHTML, />All<\/button>/)
})

test('clicking the active jump letter keeps the destination selected', () => {
  const root = element(), selected = []
  const alphabet = createCatalogAlphabet({ element: root, onFilter() {}, onJump: bucket => selected.push(bucket) })
  alphabet.configure({ jump: true, active: 'M', navigation: [{ bucket: 'M' }] })
  root.listeners.click({ target: { closest: () => ({ disabled: false, dataset: { bucket: 'M' } }) } })
  assert.deepEqual(selected, ['M'])
})

test('clicking the active filter letter still clears the filter', () => {
  const root = element(), selected = []
  const alphabet = createCatalogAlphabet({ element: root, onFilter: bucket => selected.push(bucket), onJump() {} })
  alphabet.configure({ jump: false, active: 'M' })
  root.listeners.click({ target: { closest: () => ({ disabled: false, dataset: { bucket: 'M' } }) } })
  assert.deepEqual(selected, [''])
})

test('the jump rail forwards wheel movement instead of scrolling its letters', () => {
  const root = element()
  const originalWindow = globalThis.window
  const movement = []
  globalThis.window = { innerHeight: 900, scrollBy: (...args) => movement.push(args) }
  try {
    const alphabet = createCatalogAlphabet({ element: root, onFilter() {}, onJump() {} })
    alphabet.configure({ jump: true, navigation: [{ bucket: 'A' }] })
    let prevented = false
    root.listeners.wheel({ deltaMode: 0, deltaY: 48, preventDefault: () => { prevented = true } })
    assert.equal(prevented, true)
    assert.deepEqual(movement, [[0, 48]])
  } finally {
    globalThis.window = originalWindow
  }
})

test('rapid wheel ticks are batched into a single scroll per animation frame', () => {
  const root = element()
  const originalWindow = globalThis.window
  const movement = []
  let frame
  globalThis.window = {
    innerHeight: 900,
    scrollBy: (...args) => movement.push(args),
    requestAnimationFrame: callback => { frame = callback },
  }
  try {
    const alphabet = createCatalogAlphabet({ element: root, onFilter() {}, onJump() {} })
    alphabet.configure({ jump: true, navigation: [{ bucket: 'A' }] })
    root.listeners.wheel({ deltaMode: 0, deltaY: 20, preventDefault() {} })
    root.listeners.wheel({ deltaMode: 0, deltaY: 30, preventDefault() {} })
    root.listeners.wheel({ deltaMode: 0, deltaY: -5, preventDefault() {} })
    // Nothing scrolls synchronously: every tick only queues the movement.
    assert.deepEqual(movement, [])
    frame()
    assert.deepEqual(movement, [[0, 45]])
    // A tick after the flush schedules its own, separate frame.
    root.listeners.wheel({ deltaMode: 0, deltaY: 10, preventDefault() {} })
    frame()
    assert.deepEqual(movement, [[0, 45], [0, 10]])
  } finally {
    globalThis.window = originalWindow
  }
})
