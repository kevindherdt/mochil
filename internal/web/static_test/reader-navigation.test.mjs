import assert from 'node:assert/strict'
import test from 'node:test'
import { bindReaderNavigation } from '../static/reader-navigation.js'

const event = (key, overrides = {}) => ({ key, target: { matches: () => false }, preventDefault() { this.defaultPrevented = true }, ...overrides })

test('reader controls use semantic navigation while arrows remain physical', () => {
  const calls = []
  const document = { addEventListener(_name, handler) { this.handler = handler } }
  const view = {
    prev: () => calls.push('prev'), next: () => calls.push('next'),
    goLeft: () => calls.push('left'), goRight: () => calls.push('right'),
    addEventListener() {}, renderer: { getContents: () => [] },
  }
  const previousButton = {}, nextButton = {}
  bindReaderNavigation({ document, view, previousButton, nextButton })
  previousButton.onclick(); nextButton.onclick()
  for (const key of ['PageUp', 'PageDown', 'ArrowLeft', 'ArrowRight']) document.handler(event(key))
  assert.deepEqual(calls, ['prev', 'next', 'prev', 'next', 'left', 'right'])
})

test('reader navigation ignores editable targets and modified shortcuts', () => {
  let calls = 0
  const document = { addEventListener(_name, handler) { this.handler = handler } }
  const view = { prev() { calls++ }, next() { calls++ }, goLeft() { calls++ }, goRight() { calls++ }, addEventListener() {}, renderer: { getContents: () => [] } }
  bindReaderNavigation({ document, view, previousButton: {}, nextButton: {} })
  document.handler(event('PageDown', { target: { matches: () => true } }))
  document.handler(event('PageDown', { ctrlKey: true }))
  assert.equal(calls, 0)
})

test('pointer activity inside the book wakes the reader controls', () => {
  const listeners = {}
  const bookDoc = { addEventListener: (type, fn) => (listeners[type] = fn) }
  const view = { addEventListener: (type, fn) => (view[type] = fn), renderer: { getContents: () => [] }, prev() {}, next() {} }
  let woke = 0
  bindReaderNavigation({
    document: { addEventListener() {} },
    view,
    previousButton: {},
    nextButton: {},
    onActivity: () => woke++,
  })
  view.load({ detail: { doc: bookDoc } })
  listeners.pointermove()
  listeners.pointerdown()
  assert.equal(woke, 2)
})

import { tapZone } from "../static/reader-navigation.js";

test("taps on the outer edges turn pages and the middle toggles controls", () => {
  assert.equal(tapZone(10, 400), "left");
  assert.equal(tapZone(110, 400), "left");
  assert.equal(tapZone(200, 400), "toggle");
  assert.equal(tapZone(300, 400), "right");
  assert.equal(tapZone(399, 400), "right");
  assert.equal(tapZone(50, 0), "toggle", "an unmeasured page never turns by accident");
});
