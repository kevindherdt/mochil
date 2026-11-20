import test from 'node:test'
import assert from 'node:assert/strict'
import { requestJSON } from '../static/http-json.js'
import { vaultMutationCommitted } from '../static/vault-mutation-error.js'

function response(body, options = {}) {
  return new Response(body, options)
}

test('requestJSON returns parsed API data', async t => {
  t.mock.method(globalThis, 'fetch', async () => response('{"connected":true}', {
    status: 200,
    headers: { 'content-type': 'application/json' }
  }))
  const result = await requestJSON('/api/source-connection')
  assert.deepEqual(result.data, { connected: true })
})

test('requestJSON presents JSON API errors without parser noise', async t => {
  t.mock.method(globalThis, 'fetch', async () => response('{"error":"The source rejected its credentials"}', {
    status: 502,
    headers: { 'content-type': 'application/json' }
  }))
  await assert.rejects(requestJSON('/api/source-connection'), {
    message: 'The source rejected its credentials',
    status: 502
  })
})

test('requestJSON explains an HTML sign-in response', async t => {
  t.mock.method(globalThis, 'fetch', async () => response('<!doctype html><title>Sign in</title>', {
    status: 401,
    headers: { 'content-type': 'text/html' }
  }))
  await assert.rejects(requestJSON('/api/source-connection'), {
    message: 'Mochil received a sign-in or web page instead of API data. Reload this page and sign in again, then retry.',
    status: 401
  })
})

test('requestJSON explains a plain unauthorized session response', async t => {
  t.mock.method(globalThis, 'fetch', async () => new Response('Unauthorized\n', { status: 401, headers: { 'content-type': 'text/plain' } }))
  await assert.rejects(
    requestJSON('/api/settings'),
    { message: 'Your Mochil web session expired. Reload this page and sign in again, then retry.', status: 401 },
  )
})

test('requestJSON supports successful empty responses', async t => {
  t.mock.method(globalThis, 'fetch', async () => response(null, { status: 204 }))
  const result = await requestJSON('/api/source-library')
  assert.equal(result.data, null)
})

test('requestJSON preserves the explicit committed vault outcome', async t => {
  t.mock.method(globalThis, 'fetch', async () => response(JSON.stringify({
    error: 'Saved, but indexing failed', code: 'vault_index_refresh_failed',
    committed: true, index_refreshed: false,
  }), {status: 500}))
  await assert.rejects(requestJSON('/api/vaults/test/pages'), error => {
    assert.equal(vaultMutationCommitted(error), true)
    assert.equal(error.message, 'Saved, but indexing failed')
    return true
  })
})

test('vault committed outcome rejects ambiguous or malformed error fields', () => {
  const valid = {status: 500, code: 'vault_index_refresh_failed', committed: true, indexRefreshed: false}
  for (const patch of [{status: 409}, {code: ''}, {committed: 'true'}, {committed: false}, {indexRefreshed: 'false'}, {indexRefreshed: undefined}]) {
    assert.equal(vaultMutationCommitted({...valid, ...patch}), false)
  }
  assert.equal(vaultMutationCommitted(new Error('Offline')), false)
  assert.equal(vaultMutationCommitted(null), false)
})
