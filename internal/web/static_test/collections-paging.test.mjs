import assert from 'node:assert/strict'
import test from 'node:test'
import { createCollections } from '../static/collections.js'

const tick = () => new Promise(resolve => setImmediate(resolve))
const booksURL = (id, offset, limit = 120) => `/api/collections/${id}/books?limit=${limit}&offset=${offset}`
const books = (from, count) => Array.from({ length: count }, (_, index) => ({ id: `b${from + index}`, title: `Book ${from + index}` }))
const page = (data, total) => ({ data, response: { headers: { get: name => name === 'X-Total-Count' ? String(total) : null } } })
const overview = { data: [{ id: 'c1', name: 'Everything', count: 300, members: [] }] }
// Both directions are pagers now, so tests name the one they mean instead of
// relying on creation order.
const forward = f => f.pagers.filter(pager => pager.label === 'books').at(-1)

// The detail writes one shell into #collection-index and then appends each page
// into the grid it finds there, so the fixture keeps the grid inside the root's
// markup the way the real DOM does.
function fixture({ responses, trims = [] } = {}) {
  const root = { dataset: {}, textContent: '', innerHTML: '', value: '', hidden: false }
  const grid = {
    querySelectorAll: () => [],
    get innerHTML() {
      const match = root.innerHTML.match(/<div class="grid[^"]*">([\s\S]*?)<\/div>\s*<div class="pagination-sentinel/)
      return match ? match[1] : ''
    },
    insertAdjacentHTML(_where, markup) {
      const anchor = '</div><div class="pagination-sentinel'
      root.innerHTML = root.innerHTML.replace(anchor, `${markup}${anchor}`)
    },
    get cards() { return (this.innerHTML.match(/<article/g) || []).length },
  }
  const spacer = () => ({ style: { height: '0px' } })
  const parts = {
    '.book-grid': grid,
    '.collection-top-spacer': spacer(),
    '.collection-bottom-spacer': spacer(),
    '.collection-sentinel': {},
    '.collection-more': { hidden: true },
    '.collection-earlier-sentinel': {},
    '.collection-earlier': { hidden: true },
    '.collection-detail-count': { textContent: '' },
  }
  root.querySelector = name => parts[name] ?? null
  root.querySelectorAll = () => []
  const nodes = new Map([['#collection-index', root]])
  const node = selector => {
    if (!nodes.has(selector)) nodes.set(selector, {
      dataset: {}, textContent: '', innerHTML: '', value: '', hidden: false,
      querySelector: () => null, querySelectorAll: () => [],
    })
    return nodes.get(selector)
  }
  const card = { dataset: { id: 'c1' } }
  const requests = []
  const pagers = []
  const windowOptions = []
  const releases = []
  let layout = 'covers'
  const api = createCollections({
    document: { querySelector: () => ({}), defaultView: { dispatchEvent() {} } },
    $: node,
    $$: selector => selector.includes('.collection-grid') ? [card] : [],
    escapeHTML: String,
    requestJSON: url => {
      requests.push(url)
      const next = responses.shift()
      if (!next) return Promise.reject(new Error(`unexpected request ${url}`))
      return Promise.resolve(next)
    },
    bookMarkup: book => `<article>${book.title}</article>`,
    installCoverFallbacks() {},
    releaseCovers: removed => releases.push(removed),
    createBoundedWindow: options => {
      windowOptions.push(options)
      return {
        reset() {},
        begin: mode => ({ mode }),
        finish: () => {
          const trim = trims.shift() || { start: 0, end: 0 }
          if (trim.start || trim.end) options.onRemove([{ trimmed: trim }])
          return trim
        },
      }
    },
    createContinuousPager: options => {
      pagers.push(options)
      return { refresh() {}, disconnect() { options.disconnected = true } }
    },
    getSelectedLibrary: () => 'books',
    getItemLayout: () => layout,
    isActive: () => true,
    abortBrowse() {},
  })
  return { api, root, node, card, grid, requests, pagers, windowOptions, releases, setLayout: value => { layout = value } }
}

async function openDetail(f) {
  const loading = f.api.load()
  await tick()
  f.card.onclick()
  await tick()
  await tick()
  return loading
}

test('the first page is one bounded request, not the whole collection', async () => {
  const f = fixture({ responses: [overview, page(books(0, 120), 300)] })
  await openDetail(f)
  assert.deepEqual(f.requests, ['/api/collections?library=books', booksURL('c1', 0)])
  assert.equal(f.grid.cards, 120)
  assert.equal(f.node('#status').textContent, '300 books')
  assert.equal(f.api.openId(), 'c1')
  // The same window the shelf uses: 600 cards at most, trimmed 120 at a time.
  assert.equal(f.windowOptions[0].maximum, 600)
  assert.equal(f.windowOptions[0].trim, 120)
  assert.equal(f.pagers.length, 1)
  assert.equal(f.pagers[0].canLoad(), true)
})

test('later pages append at the loaded offset and stop at the total', async () => {
  const f = fixture({ responses: [overview, page(books(0, 120), 300), page(books(120, 120), 300), page(books(240, 60), 300)] })
  await openDetail(f)
  assert.equal(await f.pagers[0].load(), true)
  assert.equal(f.requests[2], booksURL('c1', 120))
  assert.equal(f.grid.cards, 240)
  assert.match(f.grid.innerHTML, /<article>Book 239<\/article>/)
  assert.equal(f.pagers[0].canLoad(), true)
  assert.equal(await f.pagers[0].load(), true)
  assert.equal(f.requests[3], booksURL('c1', 240))
  assert.equal(f.grid.cards, 300)
  // Everything is on screen: no more requests, however often the pager asks.
  assert.equal(f.pagers[0].canLoad(), false)
  assert.equal(await f.pagers[0].load(), false)
  assert.equal(f.requests.length, 4)
})

test('a trimmed window advances the start offset so the next page is still the right one', async () => {
  const f = fixture({ responses: [overview, page(books(0, 120), 300), page(books(120, 120), 300), page(books(240, 60), 300)], trims: [{ start: 120, end: 0 }] })
  await openDetail(f)
  assert.equal(await f.pagers[0].load(), true)
  // The first 120 cards were dropped from the DOM and their covers released, so
  // the loaded window now starts at b120 and the next request continues at 240.
  assert.deepEqual(f.releases.at(-1), { trimmed: { start: 120, end: 0 } })
  assert.equal(await f.pagers[0].load(), true)
  assert.equal(f.requests[3], booksURL('c1', 240))
  assert.equal(f.requests.length, 4)
})

// A layout switch redraws the pages already loaded; reopening from offset 0
// would throw away everything the reader had scrolled through.
test('changing the layout keeps the pages already loaded', async () => {
  const f = fixture({ responses: [overview, page(books(0, 120), 300), page(books(120, 120), 300), page(books(0, 120), 300)] })
  await openDetail(f)
  assert.equal(await f.pagers[0].load(), true)
  assert.equal(f.grid.cards, 240)
  const requests = f.requests.length
  f.setLayout('list')
  assert.equal(f.api.reflow(), true)
  await tick()
  assert.equal(f.requests.length, requests, 'a layout switch must not refetch the first page')
  assert.equal(f.grid.cards, 240)
  assert.match(f.grid.innerHTML, /<article>Book 239<\/article>/)
  assert.match(f.node('#collection-index').innerHTML, /book-list/)
  assert.equal(f.pagers[0].disconnected, true)
  assert.equal(forward(f).canLoad(), true)
})

test('a layout switch keeps a trimmed window starting where it was', async () => {
  const f = fixture({ responses: [overview, page(books(0, 120), 300), page(books(120, 120), 300), page(books(240, 60), 300)], trims: [{ start: 120, end: 0 }] })
  await openDetail(f)
  assert.equal(await f.pagers[0].load(), true)
  f.setLayout('list')
  assert.equal(f.api.reflow(), true)
  await tick()
  assert.equal(f.requests.length, 3)
  assert.equal(f.grid.cards, 120)
  assert.match(f.grid.innerHTML, /<article>Book 120<\/article>/)
  // The window still describes b120..b239, so the next page is 240, not 120.
  assert.equal(await forward(f).load(), true)
  assert.equal(f.requests[3], booksURL('c1', 240))
})

// Trimming drops cards from the top of the DOM. Scrolling back up used to show a
// blank spacer where they had been, because the detail only ever paged forward.
test('scrolling back up loads the cards the window trimmed away', async () => {
  const f = fixture({
    responses: [overview, page(books(0, 120), 300), page(books(120, 120), 300), page(books(0, 120), 300)],
    trims: [{ start: 120, end: 0 }],
  })
  await openDetail(f)
  assert.equal(f.pagers.length, 1, 'nothing to load back to before the window starts at 0')

  assert.equal(await f.pagers[0].load(), true)
  assert.equal(f.pagers.length, 2, 'after a trim the detail can page both ways')
  assert.equal(f.pagers[1].canLoad(), true, 'the trimmed window can load what it dropped')

  assert.equal(await f.pagers[1].load(), true)
  assert.equal(f.requests.at(-1), booksURL('c1', 0))
  assert.match(f.grid.innerHTML, /<article>Book 0<\/article>/)
  assert.equal(f.pagers[1].canLoad(), false, 'the start of the collection is loaded again')
})
