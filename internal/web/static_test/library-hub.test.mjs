import assert from 'node:assert/strict'
import test from 'node:test'
import { hubCardsHTML, hubCount, hubKindWord } from '../static/library-hub.js'

const escapeHTML = (value) => String(value).replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' }[c]))

test('a format card says what kind of shelf it is, in words a reader uses', () => {
  assert.equal(hubKindWord({ id: 'books', kind: 'books' }), 'Books')
  assert.equal(hubKindWord({ id: 'manga', kind: 'manga' }), 'Manga')
  assert.equal(hubKindWord({ id: 'audiobooks', kind: 'audio' }), 'Audiobooks')
  assert.equal(hubKindWord({ id: 'movies', kind: 'video' }), 'Video')
  assert.equal(hubKindWord({ id: 'comics' }), 'Comics')
})

test('counts come from the server or not at all -- nothing is invented', () => {
  assert.equal(hubCount({ count: 112 }), '112 items')
  assert.equal(hubCount({ count: 1 }), '1 item')
  assert.equal(hubCount({}), '')
  assert.equal(hubCount({ count: 0 }), '')
  assert.equal(hubCount({ count: 2140 }), '2,140 items')
})

test('each shelf becomes one card, with its name and its icon', () => {
  const html = hubCardsHTML([
    { id: 'books', name: 'Books', kind: 'books', count: 112 },
    { id: 'manga', name: 'Manga', kind: 'manga' },
  ], escapeHTML)
  assert.equal((html.match(/class="hub-card"/g) || []).length, 2)
  assert.match(html, /data-library-id="books"/)
  assert.match(html, /112 items/)
  // a shelf without a count falls back to its kind rather than printing "undefined"
  assert.match(html, /Manga/)
  assert.doesNotMatch(html, /undefined/)
})

test('names are escaped, because a library name is user input', () => {
  const html = hubCardsHTML([{ id: 'x', name: '<script>x</script>', kind: 'books' }], escapeHTML)
  assert.doesNotMatch(html, /<script>/)
  assert.match(html, /&lt;script&gt;/)
})
