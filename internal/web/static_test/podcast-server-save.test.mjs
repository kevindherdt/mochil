import assert from 'node:assert/strict'
import test from 'node:test'
import { createServerSave, serverSaveHTML, serverSaveState } from '../static/podcast-server-save.js'
import { requestJSON } from '../static/http-json.js'

const escapeHTML = value => String(value).replace(/[&<>"]/g, c => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c])

// A tiny stand-in for the shelf: one row with a button, clicks delegated to the container.
function shelf(state = 'idle') {
  const svg = { innerHTML: '' }
  const attrs = {}
  const button = {
    dataset: { action: 'server-save', state },
    title: '', closest(selector) { return selector === '[data-item]' ? article : selector === '[data-action="server-save"]' ? button : null },
    querySelector() { return svg }, setAttribute(name, value) { attrs[name] = value },
  }
  const article = { dataset: { item: 'ep9' }, querySelector: () => ({ textContent: 'Episode 9' }) }
  const handlers = []
  const container = {
    addEventListener: (type, fn) => handlers.push(fn),
    querySelectorAll: () => [button],
  }
  const status = { textContent: '' }
  const click = () => {
    const event = { target: { closest: selector => selector === '[data-action="server-save"]' ? button : null }, stopImmediatePropagation() { this.stopped = true }, preventDefault() {} }
    handlers.forEach(fn => fn(event))
    return event
  }
  return { button, svg, attrs, container, status, click }
}

const sequence = (...answers) => { const calls = []; const fn = async (url, options) => { calls.push([url, options?.method || 'GET']); const next = answers.shift(); if (next instanceof Error) throw next; return { data: next } }; fn.calls = calls; return fn }
const settle = () => new Promise(done => setTimeout(done, 5))

test('a row shows whether the server already has the episode', () => {
  assert.equal(serverSaveState({ server_downloaded: true }), 'ready')
  assert.equal(serverSaveState({}), 'idle')
  assert.match(serverSaveHTML({ title: 'A "title"', server_downloaded: true }, escapeHTML), /data-state="ready".*Saved on the server: A &quot;title&quot;/)
  assert.match(serverSaveHTML({ title: 'Ep' }, escapeHTML), /data-state="idle".*Save to the server: Ep/)
})

test('pressing the arrow asks the server to download, shows progress, and says when it is saved', async () => {
  const s = shelf()
  const request = sequence({ status: 'queued' }, { status: 'downloading' }, { status: 'ready' })
  createServerSave({ container: s.container, requestJSON: request, status: s.status, wait: async () => {}, MutationObserverClass: undefined })
  const event = s.click()
  assert.equal(event.stopped, true, 'the shelf must not also open the browser download')
  assert.equal(s.button.dataset.state, 'saving')
  assert.match(s.status.textContent, /Saving “Episode 9” to the server/)
  await settle()
  assert.deepEqual(request.calls, [['/api/podcast-downloads/ep9', 'POST'], ['/api/podcast-downloads/ep9', 'GET'], ['/api/podcast-downloads/ep9', 'GET']])
  assert.equal(s.button.dataset.state, 'ready')
  assert.match(s.status.textContent, /“Episode 9” is saved on the server/)
})

test('a failed download says why and can be tried again', async () => {
  const s = shelf()
  const request = sequence({ status: 'failed', error: 'the feed returned 404' }, { status: 'ready' })
  createServerSave({ container: s.container, requestJSON: request, status: s.status, wait: async () => {}, MutationObserverClass: undefined })
  s.click(); await settle()
  assert.equal(s.button.dataset.state, 'failed')
  assert.match(s.status.textContent, /Could not save “Episode 9” to the server: the feed returned 404/)
  s.click(); await settle()
  assert.equal(s.button.dataset.state, 'ready')
})

test('a rejected HTTP admission shows its retry message and retries only on another click', async t => {
  const s = shelf()
  let requests = 0
  t.mock.method(globalThis, 'fetch', async () => {
    requests++
    return requests === 1
      ? new Response(JSON.stringify({ status: 'failed', error: 'The previous download is still stopping. Please try again shortly.' }), { status: 409, headers: { 'content-type': 'application/json' } })
      : new Response(JSON.stringify({ status: 'ready' }), { status: 200, headers: { 'content-type': 'application/json' } })
  })
  createServerSave({ container: s.container, requestJSON, status: s.status, wait: async () => {}, MutationObserverClass: undefined })
  s.click(); await settle()
  assert.equal(s.button.dataset.state, 'failed')
  assert.match(s.status.textContent, /previous download is still stopping.*try again shortly/i)
  assert.equal(requests, 1)
  s.click(); await settle()
  assert.equal(requests, 2)
  assert.equal(s.button.dataset.state, 'ready')
})

test('an unreachable or switched-off server is explained, not silent', async () => {
  const offline = shelf()
  createServerSave({ container: offline.container, requestJSON: sequence(new TypeError('Failed to fetch')), status: offline.status, MutationObserverClass: undefined })
  offline.click(); await settle()
  assert.match(offline.status.textContent, /the server could not be reached/)
  const off = shelf()
  const notFound = Object.assign(new Error('404 page not found'), { status: 404 })
  createServerSave({ container: off.container, requestJSON: sequence(notFound), status: off.status, MutationObserverClass: undefined })
  off.click(); await settle()
  assert.match(off.status.textContent, /server downloads are not available/)
})

test('pressing it again while saving, or once saved, does not start another download', async () => {
  const saved = shelf('ready')
  const request = sequence()
  createServerSave({ container: saved.container, requestJSON: request, status: saved.status, MutationObserverClass: undefined })
  saved.click()
  assert.equal(request.calls.length, 0)
  assert.match(saved.status.textContent, /already saved on the server/)
  const busy = shelf()
  const slow = sequence({ status: 'downloading' }, { status: 'ready' })
  createServerSave({ container: busy.container, requestJSON: slow, status: busy.status, wait: () => new Promise(() => {}), MutationObserverClass: undefined })
  busy.click(); busy.click(); await settle()
  assert.equal(slow.calls.filter(([, method]) => method === 'POST').length, 1)
  assert.match(busy.status.textContent, /already being saved/)
})

test('a download that never finishes is given up on with a reason', async () => {
  const s = shelf()
  const forever = async () => ({ data: { status: 'downloading' } })
  createServerSave({ container: s.container, requestJSON: forever, status: s.status, wait: async () => {}, pollMs: 10, giveUpAfterMs: 50, MutationObserverClass: undefined })
  s.click(); await settle()
  assert.equal(s.button.dataset.state, 'failed')
  assert.match(s.status.textContent, /taking too long/)
})

test('a redrawn row is painted from what is known, not from the stale markup', async () => {
  const s = shelf()
  let redraw
  class Fake { constructor(fn) { redraw = fn } observe() {} }
  createServerSave({ container: s.container, requestJSON: sequence({ status: 'ready' }), status: s.status, MutationObserverClass: Fake })
  s.click(); await settle()
  s.button.dataset.state = 'idle'; s.button.dataset.paintedState = ''
  redraw()
  assert.equal(s.button.dataset.state, 'ready')
})
