import test from 'node:test'
import assert from 'node:assert/strict'
import { createMediaSectionNavigator, currentMediaEntry } from '../static/media-section-navigator.js'

const positionedEntry = (label, top, bottom, connected = true) => ({
  label,
  element: { isConnected: connected, getBoundingClientRect: () => ({ top, bottom }) }
})

test('visible media entry follows the newest row crossing the sticky header', () => {
  const tallEarlierCard = positionedEntry('Aimer', -240, 260)
  const currentRow = positionedEntry('Muse', 104, 320)
  assert.equal(currentMediaEntry([tallEarlierCard, currentRow], 108), currentRow)
})

test('visible media entry uses the nearest upcoming row before content reaches the header', () => {
  const later = positionedEntry('Muse', 420, 640)
  const nearer = positionedEntry('Aimer', 180, 400)
  assert.equal(currentMediaEntry([later, nearer], 108), nearer)
})

test('visible media entry ignores detached and fully scrolled cards', () => {
  const detached = positionedEntry('Aimer', 100, 300, false)
  const gone = positionedEntry('Bowie', -300, 80)
  assert.equal(currentMediaEntry([detached, gone], 108), null)
})

test('uses server navigation to expose unloaded media letters', () => {
  global.window = { addEventListener() {} }
  global.requestAnimationFrame = () => 1
  global.cancelAnimationFrame = () => {}
  const element = {
    hidden: true, innerHTML: '', classList: { toggle() {} },
    addEventListener() {}, querySelectorAll() { return [] }
  }
  const navigator = createMediaSectionNavigator(element)
  navigator.show([], { navigation: [{ bucket: 'A' }, { bucket: 'M' }] })
  assert.equal(element.hidden, false)
  assert.match(element.innerHTML, /data-bucket="M"/)
  assert.match(element.innerHTML, /data-bucket="B"[^>]*disabled/)
})

test('restores the visible letter immediately after a catalogue render', () => {
  global.window = { addEventListener() {} }
  global.requestAnimationFrame = () => 1
  global.cancelAnimationFrame = () => {}
  global.document = { documentElement: {} }
  global.getComputedStyle = () => ({ getPropertyValue: () => '100px' })
  const buttons = new Map()
  const element = {
    hidden: true, innerHTML: '', classList: { toggle() {} },
    addEventListener() {},
    querySelectorAll() { return [...buttons.values()] }
  }
  const entry = (label, bottom) => ({
    label,
    element: { isConnected: true, getBoundingClientRect: () => ({ top: bottom - 120, bottom }) }
  })
  for (const bucket of ['A', 'M']) buttons.set(bucket, {
    dataset: { bucket }, classList: { toggle(_name, active) { this.active = active } },
    setAttribute(_name, value) { this.pressed = value }
  })

  try {
    const navigator = createMediaSectionNavigator(element)
    navigator.show([entry('Aimer', 80), entry('Muse', 320)])
    assert.equal(buttons.get('A').classList.active, false)
    assert.equal(buttons.get('M').classList.active, true)
    assert.equal(buttons.get('M').pressed, 'true')
  } finally {
    delete global.document
    delete global.getComputedStyle
  }
})
