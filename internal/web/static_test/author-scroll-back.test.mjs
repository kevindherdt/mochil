import test from 'node:test'
import assert from 'node:assert/strict'
import { createAuthorCatalogue } from '../static/author-catalogue.js'

// The author list keeps a bounded window of cards. When it drops the leading rows
// there was no way back: scrolling up showed the spacer where they had been.
test('scrolling back up in the author list reloads the rows the window dropped', async () => {
  const previousWindow = globalThis.window
  globalThis.window = { scrollY: 0, dispatchEvent() {} }
  try {
    const requests = []
    const grid = { innerHTML: '', insertAdjacentHTML(_where, markup) { this.innerHTML += markup } }
    const index = {
      hidden: true,
      innerHTML: '',
      querySelector(selector) {
        if (selector === '.author-cover-grid') return grid
        if (selector === '[data-earlier-sentinel]') return {}
        if (selector === '[data-earlier-more]') return { hidden: true }
        if (selector === '.back-button') return {}
        return null
      },
      querySelectorAll: () => [],
    }
    const elements = {
      '#series-index': index,
      '#book-catalog': {},
      '#collection-index': {},
      '#status': { textContent: '' },
      '#search': { value: '' },
      '#series-sentinel': {},
      '#series-load-more': {},
      '#alphabet': {},
    }
    let trimNext = 0
    const pagers = []
    const catalogue = createAuthorCatalogue({
      $: selector => elements[selector],
      document: {},
      escapeHTML: String,
      requestJSON: async url => {
        requests.push(url)
        const params = new URL(url, 'https://example.invalid').searchParams
        const offset = Number(params.get('offset') || 0)
        const limit = Number(params.get('limit') || 80)
        const data = Array.from({ length: limit }, (_, i) => ({ name: `Author ${offset + i}`, count: 1, library_id: 'books', bucket: 'A' }))
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
      cards: { markup: book => `<article>${book.title}</article>`, installCoverFallbacks() {}, releaseCovers() {} },
      collections: { installActions() {} },
      libraryState: { selected: () => 'books' },
      getBrowseMode: () => 'authors',
      getItemLayout: () => 'covers',
      getGroupLayout: () => 'covers',
      abortOthers() {},
      getActiveBucket: () => '',
    })
    catalogue.bind()
    await catalogue.load()

    const earlier = pagers.find(pager => pager.label === 'earlier authors')
    assert.ok(earlier, 'the list installs a pager for the rows above the window')
    assert.equal(earlier.canLoad(), false, 'nothing to reload while the list starts at the top')

    // The next page trims the leading 40 rows, so the window now starts at 40.
    trimNext = 40
    await pagers.find(pager => pager.label === 'authors').load()
    assert.equal(earlier.canLoad(), true, 'a trimmed window can load what it dropped')

    const before = requests.length
    assert.equal(await earlier.load(), true, 'the rows above the window load again')
    assert.equal(requests.length, before + 1)
    const request = new URL(requests.at(-1), 'https://example.invalid')
    assert.equal(request.pathname, '/api/authors')
    assert.equal(request.searchParams.get('offset'), '0', 'it asks for exactly the dropped rows')
    assert.equal(request.searchParams.get('limit'), '40')
    assert.match(grid.innerHTML, /Author 0</)
    assert.equal(earlier.canLoad(), false, 'the start of the list is loaded again')
  } finally {
    globalThis.window = previousWindow
  }
})
