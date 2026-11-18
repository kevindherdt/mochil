import test from 'node:test'
import assert from 'node:assert/strict'
import { createAuthorCatalogue } from '../static/author-catalogue.js'

function windowedElement() {
  return { innerHTML: '', insertAdjacentHTML(_where, markup) { this.innerHTML += markup } }
}

function fixture(requestJSON, { createBoundedWindow, getGroupLayout = () => 'covers', releaseCovers = () => {} } = {}) {
  const back = {}
  const books = { innerHTML: '' }
  const authorList = windowedElement()
  const authorCoverGrid = windowedElement()
  const topSpacer = windowedElement()
  const bottomSpacer = windowedElement()
  const index = {
    hidden: true,
    innerHTML: '',
    querySelector(selector) {
      if (selector === '.back-button') return back
      if (selector === '.series-focus-books') return books
      if (selector === '.author-list') return authorList
      if (selector === '.author-cover-grid') return authorCoverGrid
      if (selector === '[data-top-spacer]') return topSpacer
      if (selector === '[data-bottom-spacer]') return bottomSpacer
      return null
    },
    querySelectorAll: () => [],
  }
  const elements = {
    '#series-index': index,
    '#book-catalog': {},
    '#collection-index': {},
    '#status': {},
    '#search': { value: '' },
    '#series-sentinel': {},
    '#series-load-more': {},
  }
  const defaultBoundedWindow = () => ({
    reset() {},
    begin() { return {} },
    finish() { return { start: 0, end: 0 } },
  })
  const catalogue = createAuthorCatalogue({
    $: selector => elements[selector],
    document: {},
    escapeHTML: String,
    requestJSON,
    createBoundedWindow: createBoundedWindow || defaultBoundedWindow,
    createContinuousPager: () => ({ refresh() {} }),
    cards: {
      markup: book => `<article>${book.title}</article>`,
      installCoverFallbacks() {},
      releaseCovers,
    },
    collections: { installActions() {} },
    libraryState: { selected: () => 'books' },
    getBrowseMode: () => 'authors',
    getItemLayout: () => 'covers',
    getGroupLayout,
    abortOthers() {},
    getActiveBucket: () => '',
  })
  catalogue.bind()
  return { catalogue, index, books, back, status: elements['#status'], authorList, authorCoverGrid, topSpacer, bottomSpacer }
}

const response = data => ({ data, response: { headers: { get: () => String(data.length) } } })

test('restoring an author missing from the lookup shows their books without claiming a count', async () => {
  const previousWindow = globalThis.window
  globalThis.window = { scrollY: 0, dispatchEvent() {} }
  try {
    const requests = []
    const { catalogue, index, books, status } = fixture(async url => {
      requests.push(url)
      if (url.startsWith('/api/authors?')) return response([])
      if (url.startsWith('/api/books?')) return response([{ title: 'Example book' }])
      throw new Error(`Unexpected request: ${url}`)
    })
    catalogue.openOnLoad('Missing Author')
    assert.equal(await catalogue.load(), true)
    assert.equal(catalogue.opened(), 'Missing Author')
    assert.match(index.innerHTML, /<h2>Missing Author<\/h2>/)
    assert.doesNotMatch(index.innerHTML, /\b0 books\b/)
    assert.match(books.innerHTML, /Example book/)
    assert.equal(status.textContent, '')
    const request = new URL(requests[1], 'https://example.invalid')
    assert.equal(requests.length, 2)
    assert.equal(request.pathname, '/api/books')
    assert.equal(request.searchParams.get('author'), 'Missing Author')
    assert.equal(request.searchParams.get('library'), 'books')
  } finally {
    globalThis.window = previousWindow
  }
})

test('the author grid is wrapped in a bounded window that releases trimmed covers', async () => {
  const boundedWindowCalls = []
  const releasedNodes = []
  let onRemove
  const windowStub = {
    reset() {},
    begin(mode) { return { mode } },
    finish() {
      onRemove(['trimmed-author-card'])
      return { start: 1, end: 0 }
    },
  }
  const { catalogue, authorCoverGrid, topSpacer, bottomSpacer } = fixture(
    async url => {
      if (url.startsWith('/api/authors?')) return response([{ name: 'Ada', count: 2, library_id: 'books', bucket: 'A' }])
      throw new Error(`Unexpected request: ${url}`)
    },
    {
      createBoundedWindow: options => {
        boundedWindowCalls.push(options)
        onRemove = options.onRemove
        return windowStub
      },
      releaseCovers: node => releasedNodes.push(node),
    },
  )
  assert.equal(await catalogue.load(), true)
  assert.equal(boundedWindowCalls.length, 1, 'the author catalogue hands its grid to createBoundedWindow')
  assert.equal(boundedWindowCalls[0].container, authorCoverGrid)
  assert.equal(boundedWindowCalls[0].topSpacer, topSpacer)
  assert.equal(boundedWindowCalls[0].bottomSpacer, bottomSpacer)
  assert.match(authorCoverGrid.innerHTML, /Ada/)
  assert.ok(releasedNodes.includes('trimmed-author-card'), 'nodes trimmed by the window release their cover resources')
})
