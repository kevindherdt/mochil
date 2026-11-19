import assert from 'node:assert/strict'
import test from 'node:test'
import { createPodcastCatalogue } from '../static/podcast-catalogue.js'

const headers = values => ({ get: name => values[name] ?? null })

test('podcast catalogue owns show selection, filters, and cursor paging', async () => {
  const requests = []
  const catalogue = createPodcastCatalogue({
    pageSize: 2,
    requestJSON: async (url, options) => {
      requests.push({ url, options })
      return requests.length === 1
        ? { data: [{ id: 'one' }, { id: 'two' }], response: { headers: headers({ 'X-Total-Count': '3', 'X-Next-Cursor': 'next' }) } }
        : { data: [{ id: 'three' }], response: { headers: headers({ 'X-Total-Count': '3' }) } }
    },
  })

  assert.equal(await catalogue.loadShow('The Show', { library: 'spoken audio', search: 'arrival', sort: 'oldest', played: 'unplayed' }), true)
  assert.deepEqual(catalogue.episodes.map(item => item.id), ['one', 'two'])
  assert.equal(catalogue.total, 3)
  assert.match(requests[0].url, /library=spoken\+audio/)
  assert.match(requests[0].url, /show=The\+Show/)
  assert.match(requests[0].url, /sort=oldest/)
  assert.match(requests[0].url, /played=unplayed/)
  assert.match(requests[0].url, /q=arrival/)

  assert.equal(await catalogue.loadShow('The Show', { library: 'spoken audio', reset: false }), true)
  assert.deepEqual(catalogue.episodes.map(item => item.id), ['one', 'two', 'three'])
  assert.match(requests[1].url, /cursor=next/)
})

test('superseded podcast responses cannot replace the active show', async () => {
  const pending = []
  const catalogue = createPodcastCatalogue({
    requestJSON: (url, options) => new Promise(resolve => pending.push({ url, options, resolve })),
  })
  const first = catalogue.loadShow('Old', { library: 'podcasts' })
  const second = catalogue.loadShow('Current', { library: 'podcasts' })
  assert.equal(pending[0].options.signal.aborted, true)
  pending[1].resolve({ data: [{ id: 'current' }], response: { headers: headers({}) } })
  assert.equal(await second, true)
  pending[0].resolve({ data: [{ id: 'stale' }], response: { headers: headers({}) } })
  assert.equal(await first, false)
  assert.equal(catalogue.show, 'Current')
  assert.deepEqual(catalogue.episodes, [{ id: 'current' }])
})

test('a changed podcast index restarts paging once and close clears the show', async () => {
  let calls = 0
  const catalogue = createPodcastCatalogue({
    pageSize: 1,
    requestJSON: async () => {
      calls += 1
      if (calls === 2) throw Object.assign(new Error('changed'), { status: 409 })
      return { data: [{ id: String(calls) }], response: { headers: headers({ 'X-Total-Count': '2' }) } }
    },
  })
  await catalogue.loadShow('Show', { library: 'podcasts' })
  assert.equal(await catalogue.loadShow('Show', { library: 'podcasts', reset: false }), true)
  assert.equal(calls, 3)
  assert.deepEqual(catalogue.episodes, [{ id: '3' }])
  catalogue.closeShow()
  assert.equal(catalogue.show, '')
  assert.deepEqual(catalogue.episodes, [])
})
