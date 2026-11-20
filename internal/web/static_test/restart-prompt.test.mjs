import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'
import { offerRestart, withdrawRestart } from '../static/restart-prompt.js'

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

// The offer is created once and only ever switched on, so a later save that needs no restart left the
// button from the earlier one standing, asking for a restart nothing was waiting for.
test('a save that needs no restart withdraws the earlier offer', () => {
  const button = { hidden: false, disabled: false, onclick: () => {} }
  const document = { querySelector: (selector) => (selector === '#restart-now' ? button : null) }
  withdrawRestart(document)
  assert.equal(button.hidden, true, 'the offer is still on screen')
  assert.equal(button.disabled, true, 'the offer can still be pressed')
  assert.equal(button.onclick, null, 'the withdrawn button kept its restart handler')
})

test('withdrawing an offer that was never made is harmless', () => {
  assert.doesNotThrow(() => withdrawRestart({ querySelector: () => null }))
})

// The two settings-page fixes need a full DOM to drive; this reads the source instead, so a later edit
// that reintroduces either shape fails here.
test('settings withdraws the offer, and a failed module fetch is not an empty module list', () => {
  const source = readFileSync(new URL('../static/settings.js', import.meta.url), 'utf8')
  assert.match(source, /withdrawRestart\(document\)/, 'a no-restart save no longer withdraws the offer')
  assert.match(source, /async function loadModules\(\)/, 'the module fetch is not guarded any more')
  assert.ok(
    !source.includes('requestJSON("/api/modules").catch(() => ({ data: [] }))'),
    'a failed module fetch is being treated as an empty module list again',
  )
})
