import test from 'node:test'
import assert from 'node:assert/strict'
import { applyMusicTrackWindow, musicTrackMarkup, musicTrackRenderMode } from '../static/music-track-render.js'

test('render mode follows the append flag: true appends, "prepend" prepends, anything else replaces', () => {
  assert.equal(musicTrackRenderMode(true), 'append')
  assert.equal(musicTrackRenderMode('prepend'), 'prepend')
  assert.equal(musicTrackRenderMode(false), 'replace')
  assert.equal(musicTrackRenderMode(undefined), 'replace')
})

test('markup picks the cover card or the row renderer per item', () => {
  const renderCard = item => `<card>${item.id}</card>`
  const renderRow = item => `<row>${item.id}</row>`
  assert.equal(musicTrackMarkup([{ id: 1 }, { id: 2 }], true, renderCard, renderRow), '<card>1</card><card>2</card>')
  assert.equal(musicTrackMarkup([{ id: 1 }], false, renderCard, renderRow), '<row>1</row>')
})

function fakeContainer() {
  return { innerHTML: '', insertAdjacentHTML(_where, markup) { this.innerHTML += markup } }
}

test('a replace mode resets the window and shows an empty state when there is nothing to show', () => {
  const resets = []
  const trackWindow = { reset: () => resets.push(true), begin() { throw new Error('should not begin a replace') }, finish() { throw new Error('should not finish a replace') } }
  const container = fakeContainer()
  const removed = applyMusicTrackWindow({ mode: 'replace', markup: '', container, trackWindow })
  assert.deepEqual(resets, [true])
  assert.match(container.innerHTML, /Nothing found here yet/)
  assert.deepEqual(removed, { start: 0, end: 0 })
})

test('a replace mode with markup shows the markup instead of the empty state', () => {
  const trackWindow = { reset() {}, begin() {}, finish() {} }
  const container = fakeContainer()
  applyMusicTrackWindow({ mode: 'replace', markup: '<row>1</row>', container, trackWindow })
  assert.equal(container.innerHTML, '<row>1</row>')
})

test('an append mode inserts at the end and reports nodes the window trimmed from the front', () => {
  const container = fakeContainer()
  const trackWindow = {
    reset() {},
    begin(mode) { return { mode } },
    finish(insertion) { assert.equal(insertion.mode, 'append'); return { start: 3, end: 0 } },
  }
  const removed = applyMusicTrackWindow({ mode: 'append', markup: '<row>2</row>', container, trackWindow })
  assert.equal(container.innerHTML, '<row>2</row>')
  assert.deepEqual(removed, { start: 3, end: 0 })
})

test('a prepend mode inserts at the start', () => {
  const container = fakeContainer()
  let where
  container.insertAdjacentHTML = function (position, markup) { where = position; this.innerHTML = markup + this.innerHTML }
  const trackWindow = { reset() {}, begin(mode) { return { mode } }, finish() { return { start: 0, end: 2 } } }
  const removed = applyMusicTrackWindow({ mode: 'prepend', markup: '<row>0</row>', container, trackWindow })
  assert.equal(where, 'afterbegin')
  assert.deepEqual(removed, { start: 0, end: 2 })
})
