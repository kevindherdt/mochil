import test from 'node:test'
import assert from 'node:assert/strict'
import { createSeriesCatalogue } from '../static/series-catalogue.js'

// Same window as the author list: rows trimmed off the top used to leave a blank
// spacer with no way to bring them back.
test('scrolling back up in the series list reloads the rows the window dropped', async () => {
  const previousWindow = globalThis.window
  globalThis.window = { scrollY: 0, dispatchEvent() {} }
  try {
    const requests = []
    const grid = { innerHTML: '', insertAdjacentHTML(_where, markup) { this.innerHTML += markup } }
    const index = {
      hidden: true,
      innerHTML: '',
      querySelector(selector) {
        if (selector === '.series-cover-grid') return grid
        if (selector === '[data-earlier-sentinel]') return {}
        if (selector === '[data-earlier-more]') return { hidden: true }
        if (selector === '[data-top-spacer]') return { style: {} }
        if (selector === '[data-bottom-spacer]') return { style: {} }
        return null
      },
      querySelectorAll: () => [],
    }
    const elements = {
      '#series-index': index,
      '#book-catalog': {},
      '#status': { textContent: '' },
      '#search': { value: '' },
      '#series-sentinel': {},
      '#series-load-more': {},
      '#alphabet': {},
    }
    let trimNext = 0
    const pagers = []
    const catalogue = createSeriesCatalogue({
      $: selector => elements[selector],
      document: {},
      escapeHTML: String,
      requestJSON: async url => {
        requests.push(url)
        const params = new URL(url, 'https://example.invalid').searchParams
        const offset = Number(params.get('offset') || 0)
        const limit = Number(params.get('limit') || 80)
        const data = Array.from({ length: limit }, (_, i) => ({ name: `Series ${offset + i}`, count: 1, library_id: 'books', bucket: 'A', authors: [] }))
        return { data, response: { headers: { get: () => '400' } } }
      },
      createBoundedWindow: () => ({
        reset() {},
        begin: mode => ({ mode }),
        finish: () => {
          const start = trimNext
          trimNext = 0
          return { start, end: 0 }
        },
      }),
      createContinuousPager: options => {
        pagers.push(options)
        return { refresh() {}, disconnect() {} }
      },
      cards: { markup: book => `<article>${book.title}</article>`, installCoverFallbacks() {}, releaseCovers() {}, bookmarkIcon: '' },
      collections: { installActions() {}, syncSaveSearch() {} },
      libraryState: { selected: () => 'books' },
      getBrowseMode: () => 'series',
      getSeriesLayout: () => 'covers',
      getItemLayout: () => 'covers',
      getActiveBucket: () => '',
      abortBooks() {},
    })
    catalogue.bind()
    await catalogue.load()

    const earlier = pagers.find(pager => pager.label === 'earlier series')
    assert.ok(earlier, 'the list installs a pager for the rows above the window')
    assert.equal(earlier.canLoad(), false)

    trimNext = 40
    await pagers.find(pager => pager.label === 'series').load()
    assert.equal(earlier.canLoad(), true, 'a trimmed window can load what it dropped')

    const before = requests.length
    assert.equal(await earlier.load(), true)
    assert.equal(requests.length, before + 1)
    const request = new URL(requests.at(-1), 'https://example.invalid')
    assert.equal(request.pathname, '/api/series')
    assert.equal(request.searchParams.get('offset'), '0')
    assert.equal(request.searchParams.get('limit'), '40')
    assert.match(grid.innerHTML, /Series 0</)
  } finally {
    globalThis.window = previousWindow
  }
})

// Bug: loadEarlier asked for the rows above the window and then wrote them into
// whatever list was on screen when the answer arrived. Switching shelf or search
// while it was in flight put the old library's rows at the top of the new one, and
// moved windowStart away from 0 so the next "load more" skipped rows of the new shelf.
test('an earlier page that arrives after the shelf changed is dropped', async () => {
  const previousWindow = globalThis.window
  globalThis.window = { scrollY: 0, dispatchEvent() {}, scrollBy() {} }
  try {
    let library = 'books'
    const pending = []
    const grid = { innerHTML: '', insertAdjacentHTML(_where, markup) { this.innerHTML += markup } }
    const index = {
      hidden: true,
      innerHTML: '',
      querySelector(selector) {
        if (selector === '.series-cover-grid') return grid
        if (selector === '[data-earlier-sentinel]') return {}
        if (selector === '[data-earlier-more]') return { hidden: true }
        if (selector === '[data-top-spacer]') return { style: {} }
        if (selector === '[data-bottom-spacer]') return { style: {} }
        return null
      },
      querySelectorAll: () => [],
    }
    const elements = {
      '#series-index': index,
      '#book-catalog': {},
      '#status': { textContent: '' },
      '#search': { value: '' },
      '#series-sentinel': {},
      '#series-load-more': {},
    }
    let trimNext = 0
    let deferNext = false
    const pagers = []
    const catalogue = createSeriesCatalogue({
      $: selector => elements[selector],
      document: {},
      escapeHTML: String,
      requestJSON: async url => {
        const params = new URL(url, 'https://example.invalid').searchParams
        const data = Array.from({ length: Number(params.get('limit') || 80) }, (_, i) => ({
          name: `${params.get('library')} ${Number(params.get('offset') || 0) + i}`,
          count: 1,
          library_id: params.get('library'),
          bucket: 'A',
          authors: [],
        }))
        const result = { data, response: { headers: { get: () => '400' } } }
        if (!deferNext) return result
        deferNext = false
        const deferred = {}
        deferred.promise = new Promise(resolve => { deferred.resolve = () => resolve(result) })
        pending.push(deferred)
        return deferred.promise
      },
      createBoundedWindow: () => ({
        reset() {},
        begin: mode => ({ mode }),
        finish: () => {
          const start = trimNext
          trimNext = 0
          return { start, end: 0 }
        },
      }),
      createContinuousPager: options => {
        pagers.push(options)
        return { refresh() {}, disconnect() {} }
      },
      cards: { markup: book => `<article>${book.title}</article>`, installCoverFallbacks() {}, releaseCovers() {}, bookmarkIcon: '' },
      collections: { installActions() {}, syncSaveSearch() {} },
      libraryState: { selected: () => library },
      getBrowseMode: () => 'series',
      getSeriesLayout: () => 'covers',
      getItemLayout: () => 'covers',
      getActiveBucket: () => '',
      abortBooks() {},
    })
    catalogue.bind()
    await catalogue.load()

    const earlier = pagers.find(pager => pager.label === 'earlier series')
    trimNext = 40
    await pagers.find(pager => pager.label === 'series').load()
    assert.equal(earlier.canLoad(), true)

    deferNext = true
    const stale = earlier.load()
    library = 'manga'
    const fresh = catalogue.load()
    assert.equal(await fresh, true, 'the new shelf loads while the earlier page is still in flight')
    pending.at(-1).resolve()

    assert.equal(await stale, false, 'the earlier page belongs to the shelf it was asked for')
    assert.doesNotMatch(grid.innerHTML, /books 0</, 'the old shelf cannot prepend onto the new one')
    assert.match(grid.innerHTML, /manga 0</, 'the new shelf is what is on screen')
  } finally {
    globalThis.window = previousWindow
  }
})
