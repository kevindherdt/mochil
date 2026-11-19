import assert from 'node:assert/strict'
import test from 'node:test'
import { offerRestart } from '../static/restart-prompt.js'

function fixture() {
  const status = { textContent: '', after(node) { status.next = node } }
  const found = {}
  const document = {
    querySelector: (selector) => found[selector] || null,
    createElement: () => { const node = { hidden: true }; found['#restart-now'] = node; return node },
  }
  return { status, document }
}

test('the button appears next to the message, and restarting waits for the server then reloads', async () => {
  const { status, document } = fixture()
  const calls = []
  let up = 0
  let reloaded = 0
  const fetchImpl = async (url, options = {}) => {
    calls.push(`${options.method || 'GET'} ${url}`)
    if (url === '/api/libraries') return { ok: ++up >= 3 }
    return { ok: true }
  }
  const button = offerRestart({ document, status, fetchImpl, wait: async () => {}, reload: () => { reloaded++ } })
  assert.match(status.textContent, /after a restart/)
  assert.equal(button.textContent, 'Restart now')
  assert.equal(status.next, button)
  await button.onclick()
  assert.equal(calls[0], 'POST /api/restart')
  assert.equal(reloaded, 1)
  assert.equal(calls.filter((c) => c === 'GET /api/libraries').length, 3, 'polls until the server answers')
})

test('a refused restart says so and can be tried again', async () => {
  const { status, document } = fixture()
  const button = offerRestart({ document, status, fetchImpl: async () => ({ ok: false, status: 501 }), wait: async () => {}, reload: () => { throw new Error('must not reload') } })
  await button.onclick()
  assert.match(status.textContent, /Could not restart: HTTP 501/)
  assert.equal(button.disabled, false)
  assert.equal(button.textContent, 'Restart now')
})

test('offering twice reuses the same button', () => {
  const { status, document } = fixture()
  const first = offerRestart({ document, status })
  const second = offerRestart({ document, status })
  assert.equal(first, second)
})

test('a server that never comes back says to reload', async () => {
  const { status, document } = fixture()
  const button = offerRestart({ document, status, tries: 3, wait: async () => {}, fetchImpl: async (url) => { if (url === '/api/libraries') throw new Error('down'); return { ok: true } }, reload: () => { throw new Error('must not reload') } })
  await button.onclick()
  assert.match(status.textContent, /Reload this page/)
})
