import test from 'node:test'
import assert from 'node:assert/strict'
globalThis.window ??= { matchMedia: () => ({ matches: false, addEventListener() {} }), addEventListener() {}, devicePixelRatio: 1 }
const { createBookCards } = await import('../static/book-cards.js')

const escapeHTML = value => String(value).replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[ch])
const book = extra => ({ id: 'b', library_id: 'l', title: 'Neuromancer', authors: ['W. Gibson'], read_url: '/reader/l/b', download_url: '/books/l/b/x.epub', cover_url: '/covers/b', ...extra })

test('a book with its own recording is marked on its cover and in a list row', () => {
  const { bookMarkup } = createBookCards({ escapeHTML })
  assert.match(bookMarkup(book({ read_along: true })), /class="book-read-along"/)
  assert.match(bookMarkup(book({ read_along: true }), 'list'), /book-read-along-label/)
})

test('a plain book is not marked', () => {
  const { bookMarkup } = createBookCards({ escapeHTML })
  assert.doesNotMatch(bookMarkup(book({})), /read-along/)
  assert.doesNotMatch(bookMarkup(book({}), 'list'), /read-along/)
})

test('the home page rows mark a read-along book too', async () => {
  const { readAlongMarkup } = await import('../static/read-along-badge.js')
  assert.match(readAlongMarkup({ read_along: true }, 'cover'), /book-read-along/)
  assert.match(readAlongMarkup({ read_along: true }, 'label'), /Read-along/)
  assert.equal(readAlongMarkup({}, 'cover'), '')
})
