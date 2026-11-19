import test from 'node:test'
import assert from 'node:assert/strict'
import { createAudioRootCatalogue } from '../static/audio-root-catalogue.js'

function response(headers = {}) {
  const normalized = new Map(Object.entries(headers).map(([key, value]) => [key.toLowerCase(), value]))
  return { headers: { get: key => normalized.get(key.toLowerCase()) ?? null } }
}

test('root audio catalogue builds cursor requests and preserves server paging facts', async () => {
  const calls = []
  const signal = new AbortController().signal
  const catalogue = createAudioRootCatalogue({ requestJSON: async (path, options) => {
    calls.push({ path, options })
    return { data: [{ id: 'track' }], response: response({ 'X-Total-Count': '1417', 'X-Next-Cursor': 'next page' }) }
  } })

  const result = await catalogue.page({ library: 'my music', kind: 'music', search: 'Paramore & Bôa', sort: 'artist', played: 'continue', cursor: 'opaque cursor', offset: 99, signal })

  const url = new URL(calls[0].path, 'https://example.invalid')
  assert.equal(url.pathname, '/api/audio')
  assert.deepEqual(Object.fromEntries(url.searchParams), {
    library: 'my music', limit: '72', sort: 'artist', played: 'continue',
    cursor: 'opaque cursor', q: 'Paramore & Bôa',
  })
  assert.equal(calls[0].options.signal, signal)
  assert.deepEqual(result, { items: [{ id: 'track' }], total: 1417, nextCursor: 'next page' })
})

test('root audio catalogue uses offsets for spoken media and distrusts malformed headers', async () => {
  const calls = []
  const catalogue = createAudioRootCatalogue({ requestJSON: async path => {
    calls.push(path)
    return { data: [{ id: 'one' }, { id: 'two' }], response: response({ 'X-Total-Count': 'not-a-number' }) }
  } })

  const result = await catalogue.page({ library: 'audio', kind: 'audiobooks', offset: 240 })

  const url = new URL(calls[0], 'https://example.invalid')
  assert.equal(url.searchParams.get('limit'), '120')
  assert.equal(url.searchParams.get('offset'), '240')
  assert.equal(url.searchParams.has('cursor'), false)
  assert.equal(result.total, 2)
  assert.equal(result.nextCursor, '')
})

test('root audio catalogue preserves an authoritative empty total', async () => {
  const catalogue = createAudioRootCatalogue({ requestJSON: async () => ({
    data: [], response: response({ 'X-Total-Count': '0' }),
  }) })

  assert.equal((await catalogue.page({ library: 'empty', kind: 'music' })).total, 0)
})

