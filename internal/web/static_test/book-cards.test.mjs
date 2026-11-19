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
