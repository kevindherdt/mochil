import test from 'node:test'
import assert from 'node:assert/strict'
import { heldBooks, heldCardHTML } from '../static/held-rail.js'

const esc = text => String(text)

test('only shelved books on visible libraries are listed', () => {
  const books = [{ id: '1', library_id: 'a', status: 'on_hold' }, { id: '2', library_id: 'b', status: 'dropped' }, { id: '3', library_id: 'a' }]
  assert.deepEqual(heldBooks(books, id => id === 'a').map(book => book.id), ['1'])
})

test('a held card says what it is and offers Resume', () => {
  const html = heldCardHTML({ id: 'x', library_id: 'lib', title: 'Dune', authors: [], status: 'dropped', fraction: 0.2 }, esc)
  assert.match(html, /Dropped/)
  assert.match(html, /class="held-resume" data-library="lib" data-book="x"/)
})
