import assert from 'node:assert/strict'
import test from 'node:test'
import { libraryIcon, libraryPillLabel, libraryPillsHTML } from '../static/library-pills.js'

const escapeHTML = (value) => String(value).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]))
const shelves = [
  { id: 'books', name: 'Books', kind: 'books' },
  { id: 'audiobooks', name: 'Audiobooks', kind: 'audio' },
  { id: 'movies', name: 'TV & Movies', kind: 'video' },
  { id: 'wiki', name: 'wiki', kind: 'vault' },
]

test('every shelf becomes a pill with its own name', () => {
  const html = libraryPillsHTML(shelves, 'books', escapeHTML)
  assert.match(html, /Books/)
  assert.match(html, /Audiobooks/)
  assert.match(html, /TV &amp; Movies/)
  assert.equal((html.match(/class="library-pill/g) || []).length, 4)
})

test('only the shelf you are in is filled', () => {
  const html = libraryPillsHTML(shelves, 'movies', escapeHTML)
  const active = html.split('library-pill').filter((part) => part.startsWith(' is-active'))
  assert.equal(active.length, 1)
  assert.match(active[0], /data-library-id="movies"/)
})

test('the icon follows the kind, which is what the server knows', () => {
  assert.match(libraryIcon({ id: 'x', kind: 'audio' }), /M4 14v-2/)
  assert.match(libraryIcon({ id: 'x', kind: 'video' }), /M3 6h12/)
  assert.match(libraryIcon({ id: 'wiki', kind: 'vault' }), /M6 3h9/)
  assert.match(libraryIcon({ id: 'books', kind: 'books' }), /M5 5h4/)
})

test('a shelf with no kind still gets a sensible icon from its id', () => {
  assert.match(libraryIcon({ id: 'movies' }), /M3 6h12/)
  assert.match(libraryIcon({ id: 'podcasts' }), /M4 14v-2/)
  assert.match(libraryIcon({ id: 'documents' }), /M6 3h9/)
})

test('the label is the name the reader knows, never the raw id', () => {
  assert.equal(libraryPillLabel({ id: 'movies', name: 'TV & Movies' }), 'TV & Movies')
  assert.equal(libraryPillLabel({ id: 'books' }), 'books')
})
