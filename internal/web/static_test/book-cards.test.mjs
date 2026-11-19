import test from 'node:test'
import assert from 'node:assert/strict'
import { createBookCards } from '../static/book-cards.js'

globalThis.window = globalThis.window || {}
const { bookMarkup } = createBookCards({ escapeHTML: String })

test('list rows give screen readers the series and progress that sighted users see in the header columns', () => {
  const markup = bookMarkup({
    title: 'The Great Hunt',
    authors: ['Robert Jordan'],
    series: 'The Wheel of Time',
    series_index: 2,
    library_id: 'books',
    id: '1',
    read_url: '/read/books/1',
    reading_progress: { fraction: 0.4 },
  }, 'list')
  assert.match(markup, /book-list-series[^>]*aria-label="Series: The Wheel of Time #2"/)
  assert.match(markup, /book-list-meter[^>]*aria-label="40% read"/)
})

test('a book with no series or progress still announces that absence', () => {
  const markup = bookMarkup({
    title: 'Loose Pages',
    authors: ['Anon'],
    library_id: 'books',
    id: '2',
    read_url: '/read/books/2',
  }, 'list')
  assert.match(markup, /book-list-series[^>]*aria-label="No series"/)
  assert.match(markup, /book-list-meter[^>]*aria-label="Not started"/)
})

test('a finished book announces that it is read', () => {
  const markup = bookMarkup({
    title: 'Winter\u2019s Heart',
    authors: ['Robert Jordan'],
    library_id: 'books',
    id: '3',
    read_url: '/read/books/3',
    reading_progress: { finished: true },
  }, 'list')
  assert.match(markup, /book-list-meter[^>]*aria-label="Read"/)
})

// A grid that mixes several manga series only shows the chapter number as the title ("Chapter
// 122"), so there was no way to tell which series a card belonged to without opening it.
test('a manga chapter card on the covers grid names its series', () => {
  const markup = bookMarkup({
    title: 'Chapter 122',
    authors: ['YASUDA Kasumi'],
    series: 'One Piece',
    library_id: 'manga',
    library_kind: 'manga',
    id: '1',
    read_url: '/read/manga/1',
  }, 'covers')
  assert.match(markup, /<small class="book-series">One Piece<\/small>/)
})

test('a regular book on the covers grid does not repeat its series under the title', () => {
  const markup = bookMarkup({
    title: 'The Great Hunt',
    authors: ['Robert Jordan'],
    series: 'The Wheel of Time',
    library_id: 'books',
    library_kind: 'books',
    id: '1',
    read_url: '/read/books/1',
  }, 'covers')
  assert.doesNotMatch(markup, /book-series/)
})

test('a manga chapter with no series carries no empty series line', () => {
  const markup = bookMarkup({
    title: 'Oneshot',
    authors: ['Anon'],
    library_id: 'manga',
    library_kind: 'manga',
    id: '2',
    read_url: '/read/manga/2',
  }, 'covers')
  assert.doesNotMatch(markup, /book-series/)
})
