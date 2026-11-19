import assert from 'node:assert/strict'
import test from 'node:test'
import { createContinuousPager } from '../static/continuous-pager.js'

class Observer {
  static latest
  constructor(callback) { this.callback = callback; Observer.latest = this }
  observe(target) { this.target = target }
  unobserve() {}
  disconnect() {}
  intersect() { this.callback([{ isIntersecting: true, target: this.target }]) }
}

test('loads once at the sentinel and hides the manual fallback', async () => {
  globalThis.window = { IntersectionObserver: Observer, addEventListener() {}, removeEventListener() {} }
  globalThis.IntersectionObserver = Observer
  const button = {}
  let remaining = 2
  const sentinel = { getBoundingClientRect: () => ({ top: 9999, bottom: 9999 }) }
  const pager = createContinuousPager({ sentinel, button, canLoad: () => remaining > 0, load: async () => { remaining-- } })
  assert.equal(button.hidden, true)
  Observer.latest.intersect()
  await new Promise(resolve => setTimeout(resolve, 10))
  assert.equal(remaining, 1)
  assert.equal(button.disabled, false)
  pager.disconnect()
})

test('reveals the fallback after an automatic load failure', async () => {
  globalThis.window = { IntersectionObserver: Observer, addEventListener() {}, removeEventListener() {} }
  globalThis.IntersectionObserver = Observer
  const button = {}
  const sentinel = { getBoundingClientRect: () => ({ top: 9999, bottom: 9999 }) }
  const pager = createContinuousPager({ sentinel, button, canLoad: () => true, load: async () => false })
  Observer.latest.intersect()
  await new Promise(resolve => setTimeout(resolve, 10))
  assert.equal(button.hidden, false)
  pager.disconnect()
})

test('coalesces repeated intersections while a page is loading', async () => {
  globalThis.window = { IntersectionObserver: Observer, addEventListener() {}, removeEventListener() {} }
  globalThis.IntersectionObserver = Observer
  let finish
  const pending = new Promise(resolve => { finish = resolve })
  let calls = 0
  const button = {}
  const sentinel = { getBoundingClientRect: () => ({ top: 9999, bottom: 9999 }) }
  const pager = createContinuousPager({ sentinel, button, canLoad: () => true, load: () => { calls++; return pending } })
  Observer.latest.intersect()
  Observer.latest.intersect()
  assert.equal(calls, 1)
  assert.equal(button.disabled, true)
  finish(true)
  await new Promise(resolve => setTimeout(resolve, 10))
  assert.equal(button.disabled, false)
  pager.disconnect()
})
