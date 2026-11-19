import assert from 'node:assert/strict'
import test from 'node:test'
import { createActivityMonitor, refreshDelay } from '../static/activity-monitor.js'

test('scans are watched closely while they run and rarely when nobody is looking', () => {
  assert.equal(refreshDelay([{ status: 'running' }], true), 1000)
  assert.equal(refreshDelay([{ status: 'queued' }], false), 1000)
  assert.equal(refreshDelay([{ status: 'done' }], true), 15000)
  assert.equal(refreshDelay([], false), 5000)
})

test('a hidden tab stops asking and a returning tab asks at once', async () => {
  let asked = 0
  const listeners = {}
  const timers = []
  const realSet = globalThis.setTimeout
  globalThis.window = { setTimeout: (fn, ms) => (timers.push(ms), 1) }
  const element = () => new Proxy({ classList: { toggle() {}, add() {}, remove() {} }, contains: () => false, addEventListener() {}, setAttribute() {} }, { get: (t, k) => (k in t ? t[k] : ''), set: (t, k, v) => ((t[k] = v), true) })
  const document = { hidden: true, querySelector: () => ({}), addEventListener: (name, fn) => (listeners[name] = fn) }
  createActivityMonitor({ document, $: element, escapeHTML: String, requestJSON: async () => (asked++, { data: [] }) })
  await new Promise((resolve) => realSet(resolve, 20))
  assert.equal(asked, 1)
  assert.deepEqual(timers, [], 'a hidden tab must not schedule another poll')
  document.hidden = false
  listeners.visibilitychange?.()
  await new Promise((resolve) => realSet(resolve, 20))
  assert.equal(asked, 2)
  assert.equal(timers.length, 1)
  delete globalThis.window
})
