import assert from 'node:assert/strict'
import test from 'node:test'
import { createCollections } from '../static/collections.js'

const tick = () => new Promise(resolve => setImmediate(resolve))
// The detail fetches one bounded page at a time, so its URL carries limit/offset.
const booksURL = (id, offset = 0) => `/api/collections/${id}/books?limit=120&offset=${offset}`
const collections = [
  { id: 'a', name: 'Alpha', count: 1, members: [] },
  { id: 'b', name: 'Beta', count: 1, members: [] },
]

function fixture() {
  const nodes = new Map(), buttons = collections.map(({ id }) => ({ dataset: { id } }))
  const node = selector => {
    if (!nodes.has(selector)) nodes.set(selector, {
      dataset: {}, textContent: '', innerHTML: '', value: '', hidden: false,
      querySelectorAll: () => [],
    })
    return nodes.get(selector)
  }
  const requests = [], events = []
  let active = true, library = 'books', layout = 'covers'
  const api = createCollections({
    document: { querySelector: () => ({}), defaultView: { dispatchEvent: event => events.push(event.type) } },
    $: node, $$: selector => selector.includes('.collection-grid') ? buttons : [],
    escapeHTML: String, requestJSON: url => new Promise((resolve, reject) => requests.push({ url, resolve, reject })),
    bookMarkup: book => `<article>${book.title}</article>`, installCoverFallbacks() {},
    getSelectedLibrary: () => library, getItemLayout: () => layout, isActive: () => active, abortBrowse() {},
  })
  const resolve = (index, data) => requests[index].resolve({ data })
  return { api, buttons, node, requests, events, resolve, setActive: value => { active = value }, setLibrary: value => { library = value }, setLayout: value => { layout = value } }
}

async function overview(f) {
  const loading = f.api.load()
  f.resolve(0, collections)
  await loading
}

test('a late collection response cannot replace the newer selected collection', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.buttons[1].onclick()
  assert.equal(f.requests[1].url, booksURL('a'))
  assert.equal(f.requests[2].url, booksURL('b'))
  f.resolve(2, [{ title: 'Beta book' }]); await tick()
  assert.equal(f.api.openId(), 'b')
  assert.match(f.node('#collection-index').innerHTML, /Beta book/)
  f.resolve(1, [{ title: 'Alpha book' }]); await tick()
  assert.equal(f.api.openId(), 'b')
  assert.match(f.node('#collection-index').innerHTML, /Beta book/)
  assert.doesNotMatch(f.node('#collection-index').innerHTML, /Alpha book/)
  assert.equal(f.node('#status').textContent, '1 book')
  assert.equal(f.events.length, 2)
})

test('stale collection failures do not replace newer status', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.buttons[1].onclick()
  f.requests[1].reject(new Error('Old failure')); await tick()
  assert.equal(f.node('#status').textContent, 'Opening Beta…')
  f.resolve(2, [{ title: 'Beta book' }]); await tick()
  assert.equal(f.node('#status').textContent, '1 book')
})

test('leaving collections discards pending success and failure', async () => {
  for (const outcome of ['success', 'failure']) {
    const f = fixture()
    await overview(f)
    f.buttons[0].onclick()
    f.setActive(false)
    f.node('#collection-index').hidden = true
    f.node('#status').textContent = 'Different view'
    if (outcome === 'success') f.resolve(1, [{ title: 'Stale book' }])
    else f.requests[1].reject(new Error('Stale failure'))
    await tick()
    assert.equal(f.node('#status').textContent, 'Different view')
    assert.doesNotMatch(f.node('#collection-index').innerHTML, /Stale book/)
    assert.equal(f.events.length, 1)
  }
})

test('returning to collections cannot revive the previous visit request', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.setActive(false)
  f.node('#collection-index').hidden = true
  f.setActive(true)
  const reentry = f.api.load()
  f.resolve(2, collections); await reentry
  f.buttons[1].onclick()
  f.resolve(3, [{ title: 'Beta book' }]); await tick()
  f.resolve(1, [{ title: 'Old visit' }]); await tick()
  assert.equal(f.api.openId(), 'b')
  assert.match(f.node('#collection-index').innerHTML, /Beta book/)
  assert.doesNotMatch(f.node('#collection-index').innerHTML, /Old visit/)
})

test('changing libraries discards the previous collection response', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.setLibrary('other')
  f.node('#status').textContent = 'Other library'
  f.resolve(1, [{ title: 'Wrong library' }]); await tick()
  assert.equal(f.node('#status').textContent, 'Other library')
  assert.doesNotMatch(f.node('#collection-index').innerHTML, /Wrong library/)
})

test('an older overview cannot replace a newer overview or poison its collection choices', async () => {
  const f = fixture()
  const old = f.api.load()
  const latest = f.api.load()
  f.resolve(1, collections); await latest
  f.resolve(0, [{ id: 'old', name: 'Old collection', count: 0, members: [] }]); await old
  assert.match(f.node('#collection-index').innerHTML, /Alpha/)
  assert.doesNotMatch(f.node('#collection-index').innerHTML, /Old collection/)
  f.buttons[1].onclick()
  assert.equal(f.node('#status').textContent, 'Opening Beta…')
  f.resolve(2, [{ title: 'Beta book' }]); await tick()
  assert.equal(f.api.openId(), 'b')
})

test('an older overview failure does not replace the latest loading status', async () => {
  const f = fixture()
  const old = f.api.load()
  const latest = f.api.load()
  f.requests[0].reject(new Error('Old failure')); await old
  assert.equal(f.node('#status').textContent, 'Opening collections…')
  f.resolve(1, collections); await latest
})

test('an overview response cannot take back the view after a collection is opened', async () => {
  const f = fixture()
  await overview(f)
  const refreshing = f.api.load()
  f.buttons[1].onclick()
  f.resolve(2, [{ title: 'Beta book' }]); await tick()
  f.resolve(1, [{ id: 'old', name: 'Old list', count: 0, members: [] }]); await refreshing
  assert.equal(f.api.openId(), 'b')
  assert.match(f.node('#collection-index').innerHTML, /Beta book/)
})

test('opening the overview invalidates a pending detail request', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  const returning = f.api.load()
  f.resolve(2, collections); await returning
  f.resolve(1, [{ title: 'Old detail' }]); await tick()
  assert.equal(f.api.openId(), '')
  assert.match(f.node('#collection-index').innerHTML, /Beta/)
  assert.doesNotMatch(f.node('#collection-index').innerHTML, /Old detail/)
})

for (const first of ['pending Beta', 'reflow request']) {
  test(`reflow while switching collections preserves Beta when ${first} resolves first`, async () => {
    const f = fixture()
    await overview(f)
    f.buttons[0].onclick()
    f.resolve(1, [{ title: 'Alpha book' }]); await tick()
    f.buttons[1].onclick()
    assert.equal(f.api.openId(), 'a')
    f.setLayout('list')
    assert.equal(f.api.reflow(), true)
    assert.equal(f.api.openId(), 'a')
    assert.equal(f.requests[2].url, booksURL('b'))
    assert.equal(f.requests[3].url, booksURL('b'))
    const order = first === 'pending Beta' ? [2, 3] : [3, 2]
    f.resolve(order[0], [{ title: 'Beta book' }]); await tick()
    f.resolve(order[1], [{ title: 'Beta book' }]); await tick()
    assert.equal(f.api.openId(), 'b')
    assert.match(f.node('#collection-index').innerHTML, /Beta book/)
    assert.match(f.node('#collection-index').innerHTML, /book-list/)
    assert.doesNotMatch(f.node('#collection-index').innerHTML, /Alpha book/)
    assert.equal(f.events.length, 3)
  })
}

test('reflow during Back preserves the requested overview', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.resolve(1, [{ title: 'Alpha book' }]); await tick()
  const back = f.node('.collection-index .back-button').onclick()
  f.setLayout('list')
  assert.equal(f.api.reflow(), true)
  assert.equal(f.requests.length, 3)
  f.resolve(2, collections); await back
  assert.equal(f.api.openId(), '')
  assert.match(f.node('#collection-index').innerHTML, /collection-list/)
})

test('reflow after a failed Back reloads the displayed detail', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.resolve(1, [{ title: 'Alpha book' }]); await tick()
  const back = f.node('.collection-index .back-button').onclick()
  f.requests[2].reject(new Error('Offline')); await back
  f.setLayout('list')
  assert.equal(f.api.reflow(), true)
  assert.equal(f.requests[3].url, booksURL('a'))
  f.resolve(3, [{ title: 'Alpha book' }]); await tick()
  assert.equal(f.api.openId(), 'a')
  assert.match(f.node('#collection-index').innerHTML, /book-list/)
})

test('failed Back keeps the displayed collection identity until the overview succeeds', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.resolve(1, [{ title: 'Alpha book' }]); await tick()
  const markup = f.node('#collection-index').innerHTML
  const events = f.events.length
  const back = f.node('.collection-index .back-button').onclick()
  assert.equal(f.api.openId(), 'a')
  f.requests[2].reject(new Error('Offline')); await back
  assert.equal(f.node('#collection-index').innerHTML, markup)
  assert.equal(f.api.openId(), 'a')
  assert.equal(f.events.length, events)
  assert.equal(f.node('#status').textContent, 'Could not open collections: Offline')
  const retry = f.node('.collection-index .back-button').onclick()
  f.resolve(3, collections); await retry
  assert.equal(f.api.openId(), '')
  assert.match(f.node('#collection-index').innerHTML, /Beta/)
  assert.equal(f.events.length, events + 1)
})

test('current collection and overview failures remain visible', async () => {
  const f = fixture()
  await overview(f)
  f.buttons[0].onclick()
  f.requests[1].reject(new Error('Current detail failure')); await tick()
  assert.equal(f.node('#status').textContent, 'Current detail failure')
  const loading = f.api.load()
  f.requests[2].reject(new Error('Current overview failure')); await loading
  assert.equal(f.node('#status').textContent, 'Could not open collections: Current overview failure')
})
