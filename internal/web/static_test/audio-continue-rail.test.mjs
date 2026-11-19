import assert from 'node:assert/strict'
import test from 'node:test'
import { continuedAudioItems } from '../static/audio-continue-rail.js'

const item = (id, kind = 'podcasts', progress = {}) => ({ id, library_id: `library-${id}`, library_kind: kind, progress })

test('continue listening keeps only visible unfinished spoken media', () => {
  const values = [
    item('podcast'),
    item('audiobook', 'audiobooks'),
    item('music', 'music'),
    item('finished', 'podcasts', { finished: true }),
    item('dismissed', 'podcasts', { dismissed: true }),
    item('hidden'),
    { ...item('missing-progress'), progress: null },
  ]

  assert.deepEqual(
    continuedAudioItems(values, library => library !== 'library-hidden').map(value => value.id),
    ['podcast', 'audiobook'],
  )
})

test('continue listening remains bounded to four items', () => {
  assert.deepEqual(
    continuedAudioItems(Array.from({ length: 8 }, (_, index) => item(String(index))), () => true).map(value => value.id),
    ['0', '1', '2', '3'],
  )
})
