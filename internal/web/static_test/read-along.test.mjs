import assert from 'node:assert/strict'
import test from 'node:test'
import { firstClipOnPage, flattenOverlay, isCoarse, loadReadAlong } from '../static/read-along.js'

const overlay = {
  audio: ['a.mp3', 'b.mp3'],
  chapters: [
    { href: 'c1.html', clips: [['s0', 0, 0, 2], ['s1', 0, 2, 5], ['bad', 0, 5, 5]] },
    { href: 'c2.html', clips: [['t0', 1, 0, 3]] },
  ],
}

test('overlay clips flatten in reading order and drop empty ranges', () => {
  const clips = flattenOverlay(overlay)
  assert.deepEqual(clips.map(clip => clip.id), ['s0', 's1', 't0'])
  assert.equal(clips[2].audio, 'b.mp3')
  assert.deepEqual(flattenOverlay(null), [])
})

test('playback starts at the first narrated text on the page, else the chapter start', () => {
  const clips = flattenOverlay(overlay)
  assert.equal(firstClipOnPage(clips, 'c1.html', id => id === 's1'), 1)
  assert.equal(firstClipOnPage(clips, 'c1.html', () => false), 0)
  assert.equal(firstClipOnPage(clips, 'missing.html', () => true), -1)
})

test('a book without narration or a failed request offers no read-along', async () => {
  assert.equal(await loadReadAlong('x', async () => ({ ok: false })), null)
  assert.equal(await loadReadAlong('x', async () => ({ ok: true, json: async () => ({ audio: [], chapters: [] }) })), null)
  assert.equal(await loadReadAlong('x', async () => { throw new Error('offline') }), null)
  assert.equal((await loadReadAlong('x', async () => ({ ok: true, json: async () => overlay }))).length, 3)
})

test('a stretch the aligner could not split is not treated as one sentence', () => {
  assert.equal(isCoarse({ begin: 0, end: 12 }), false)
  assert.equal(isCoarse({ begin: 100, end: 190 }), false)
  assert.equal(isCoarse({ begin: 119.292, end: 2338.13 }), true)
})
