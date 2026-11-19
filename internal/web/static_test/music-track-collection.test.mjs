import assert from 'node:assert/strict'
import test from 'node:test'
import { musicTrackCollectionHTML } from '../static/music-track-collection.js'

const items = [{ title: 'One' }, { title: 'Two' }]
const row = item => `<article>${item.title}</article>`
const card = item => `<button>${item.title}</button>`

test('music track collections share cover and list rendering', () => {
  assert.equal(
    musicTrackCollectionHTML(items, { layout: 'list', row, card }),
    '<article>One</article><article>Two</article>',
  )
  assert.equal(
    musicTrackCollectionHTML(items, { layout: 'covers', row, card }),
    '<div class="music-track-grid"><button>One</button><button>Two</button></div>',
  )
})

test('music track collections use a caller-owned empty message', () => {
  assert.equal(
    musicTrackCollectionHTML([], { row, card, empty: 'Play something first.' }),
    '<p class="empty-state">Play something first.</p>',
  )
})
