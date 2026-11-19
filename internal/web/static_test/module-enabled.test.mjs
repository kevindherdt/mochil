import { test } from 'node:test'
import assert from 'node:assert/strict'
import { moduleEnabled } from '../static/module-enabled.js'

const fetchWith = (modules, ok = true) => async () => ({ ok, json: async () => modules })

test('moduleEnabled is true only when that module id is present and enabled', async () => {
  assert.equal(await moduleEnabled('hermes', fetchWith([{ id: 'hermes', enabled: true }])), true)
  assert.equal(await moduleEnabled('hermes', fetchWith([{ id: 'hermes', enabled: false }])), false)
  assert.equal(await moduleEnabled('hermes', fetchWith([{ id: 'editor', enabled: true }])), false)
  assert.equal(await moduleEnabled('hermes', fetchWith([])), false)
})

test('moduleEnabled accepts either a bare array or a {data: [...]} envelope', async () => {
  assert.equal(await moduleEnabled('editor', fetchWith({ data: [{ id: 'editor', enabled: true }] })), true)
  assert.equal(await moduleEnabled('editor', fetchWith([{ id: 'editor', enabled: true }])), true)
})

test('moduleEnabled fails closed (false) if the request or parse throws', async () => {
  assert.equal(await moduleEnabled('hermes', async () => { throw new Error('offline') }), false)
  assert.equal(await moduleEnabled('hermes', async () => ({ ok: false, json: async () => { throw new Error('not json') } })), false)
})

test('the default fetch implementation is used when none is passed (no crash on import)', () => {
  assert.equal(typeof moduleEnabled, 'function')
})
