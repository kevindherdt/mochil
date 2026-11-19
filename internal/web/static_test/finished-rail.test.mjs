import test from 'node:test'
import assert from 'node:assert/strict'
import { finishedBooks, finishedCardHTML } from '../static/finished-rail.js'

const esc = value => String(value).replace(/[&<>"']/g, ch => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' })[ch])

test('only finished, not dismissed books from visible libraries are listed, up to the limit', () => {
  const books = [
    { id: 'a', library_id: 'x', finished: true }, { id: 'b', library_id: 'x', finished: false },
    { id: 'c', library_id: 'hidden', finished: true }, { id: 'd', library_id: 'x', finished: true, dismissed: true }, { id: 'e', library_id: 'x', finished: true },
  ]
  assert.deepEqual(finishedBooks(books, id => id !== 'hidden').map(book => book.id), ['a', 'e'])
  assert.equal(finishedBooks(books, () => true, 1).length, 1)
})

test('a card opens the book, offers Restore, and escapes the title', () => {
  const html = finishedCardHTML({ id: '1', library_id: 'l', title: '<b>X</b>', authors: ['A'], read_url: '/read?x=1', cover_url: '/c.jpg' }, esc)
  assert.match(html, /href="\/read\?x=1"/)
  assert.match(html, /finished-restore/)
  assert.ok(!html.includes('<b>X</b>'))
})

test('a finished card is the Jump back in card, labelled Finished', () => {
  const html = finishedCardHTML({ id: '1', library_id: 'l', title: 'X', finished: true, read_url: '/r', cover_url: '/c.jpg' }, esc)
  assert.match(html, /class="jump-card"/)
  assert.match(html, /Finished/)
})
