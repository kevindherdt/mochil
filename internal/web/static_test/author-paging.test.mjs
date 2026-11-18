import test from 'node:test'
import assert from 'node:assert/strict'
import { createAuthorCatalogue } from '../static/author-catalogue.js'

// The author detail used to ask for 500 books in one request and ignore the
// total, so an author with more than 500 could never show the rest.
test('an author with more books than one page can load the rest', async () => {
  const previousWindow = globalThis.window
  globalThis.window = { scrollY: 0, dispatchEvent() {} }
  try {
    const books = Array.from({ length: 120 }, (_, index) => ({ title: `Book ${index + 1}` }))
    const requests = []
    let moreButton
    const detail = {
      innerHTML: '',
      querySelector(selector) {
        if (selector === '[data-author-more]') {
          moreButton = moreButton || { disabled: false, textContent: '' }
          return moreButton
        }
        return null
      },
    }
    const index = {
      hidden: true,
      innerHTML: '',
      querySelector(selector) {
        if (selector === '.series-focus-books') return detail
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
    const catalogue = createAuthorCatalogue({
      $: selector => elements[selector],
      document: {},
      escapeHTML: String,
      requestJSON: async url => {
        requests.push(url)
        const offset = Number(new URL(url, 'https://example.invalid').searchParams.get('offset') || 0)
        const page = Array.from({ length: offset === 0 ? 120 : 80 }, (_, index) => ({ title: `Book ${offset + index + 1}` }))
        return { data: page, response: { headers: { get: () => '200' } } }
      },
      createBoundedWindow: () => ({ reset() {}, begin: () => ({}), finish: () => ({ start: 0, end: 0 }) }),
      createContinuousPager: () => ({ refresh() {}, disconnect() {} }),
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
    await catalogue.openOnLoad('Prolific Author')
    await catalogue.load()

    assert.equal(requests.filter(url => url.startsWith('/api/books?')).length, 1)
    assert.match(detail.innerHTML, /Load more books \(80 left\)/)
    assert.match(detail.innerHTML, /Book 120/)

    moreButton.onclick()
    await new Promise(resolve => setTimeout(resolve, 0))

    const pages = requests.filter(url => url.startsWith('/api/books?'))
    assert.equal(pages.length, 2)
    assert.equal(new URL(pages[1], 'https://example.invalid').searchParams.get('offset'), '120')
    assert.equal(new URL(pages[1], 'https://example.invalid').searchParams.get('limit'), '120')
    assert.match(detail.innerHTML, /Book 200/)
    assert.doesNotMatch(detail.innerHTML, /Load more books/)
  } finally {
    globalThis.window = previousWindow
  }
})
