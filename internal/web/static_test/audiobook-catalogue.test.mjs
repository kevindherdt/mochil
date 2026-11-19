import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudiobookCatalogue } from '../static/audiobook-catalogue.js'

const headers = values => ({ get: name => values[name] ?? null })

test('audiobook catalogue owns series selection and cursor paging', async () => {
  const requests = []
  const requestJSON = async (url, options) => {
    requests.push({ url, options })
    if (url.startsWith('/api/audiobook-series?')) return { data: [{ name: 'Saga', count: 3 }] }
    const page = requests.filter(request => request.url.startsWith('/api/audio?')).length
    return page === 1
      ? { data: [{ id: 'one' }, { id: 'two' }], response: { headers: headers({ 'X-Total-Count': '3', 'X-Next-Cursor': 'next-page' }) } }
      : { data: [{ id: 'three' }], response: { headers: headers({ 'X-Total-Count': '3' }) } }
  }
  const catalogue = createAudiobookCatalogue({ requestJSON, pageSize: 2 })

  assert.equal(await catalogue.loadSeries({ library: 'audio books', search: 'hero', sort: 'title' }), true)
  assert.deepEqual(catalogue.series, [{ name: 'Saga', count: 3 }])
  assert.match(requests[0].url, /library=audio\+books/)
  assert.match(requests[0].url, /q=hero/)

  assert.equal(await catalogue.loadGroup('Saga', { library: 'audio books' }), true)
  assert.deepEqual(catalogue.books.map(book => book.id), ['one', 'two'])
  assert.equal(catalogue.total, 3)
  assert.equal(await catalogue.loadGroup('Saga', { library: 'audio books', reset: false }), true)
  assert.deepEqual(catalogue.books.map(book => book.id), ['one', 'two', 'three'])
  assert.match(requests.at(-1).url, /cursor=next-page/)
  assert.equal(catalogue.loading, false)
})

test('superseded audiobook series responses cannot replace newer state', async () => {
  const pending = []
  const catalogue = createAudiobookCatalogue({
    requestJSON: (url, options) => new Promise(resolve => pending.push({ url, options, resolve })),
  })
  const first = catalogue.loadSeries({ library: 'old' })
  const second = catalogue.loadSeries({ library: 'new' })
  assert.equal(pending[0].options.signal.aborted, true)
  pending[1].resolve({ data: [{ name: 'Current' }] })
  assert.equal(await second, true)
  pending[0].resolve({ data: [{ name: 'Stale' }] })
  assert.equal(await first, false)
  assert.deepEqual(catalogue.series, [{ name: 'Current' }])
})

test('a changed audiobook index restarts paging once', async () => {
  let calls = 0
  const catalogue = createAudiobookCatalogue({
    requestJSON: async () => {
      calls += 1
      if (calls === 2) throw Object.assign(new Error('changed'), { status: 409 })
      return { data: [{ id: String(calls) }], response: { headers: headers({ 'X-Total-Count': '2' }) } }
    },
    pageSize: 1,
  })
  await catalogue.loadGroup('Saga', { library: 'books' })
  assert.equal(await catalogue.loadGroup('Saga', { library: 'books', reset: false }), true)
  assert.equal(calls, 3)
  assert.deepEqual(catalogue.books, [{ id: '3' }])
})

test('audiobook authors use the dedicated author grouping endpoint', async () => {
  const requests = []
  const requestJSON = async url => {
    requests.push(url)
    if (url.startsWith('/api/audio-groups?')) return { data: [{ id: 'author-id', name: 'Ursula Le Guin', count: 4, artwork_item_id: 'cover' }] }
    return { data: [{ id: 'book' }], response: { headers: headers({}) } }
  }
  const catalogue = createAudiobookCatalogue({ requestJSON })
  assert.equal(await catalogue.loadAuthors({ library: 'spoken' }), true)
  assert.equal(catalogue.groupType, 'authors')
  assert.match(catalogue.series[0].artwork_url, /spoken\/cover/)
  assert.equal(await catalogue.loadGroup('Ursula Le Guin', { library: 'spoken' }), true)
  assert.match(requests.at(-1), /^\/api\/audio-group-tracks\?/)
  assert.match(requests.at(-1), /group_type=authors/)
  assert.match(requests.at(-1), /group_id=author-id/)
})
