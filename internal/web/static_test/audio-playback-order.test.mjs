import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioPlaybackOrder } from '../static/audio-playback-order.js'

const key = item => item.id

test('shuffle preserves the active track and a stable order', () => {
  const queue = [{ id: 'one' }, { id: 'two' }, { id: 'three' }]
  let active = queue[1]
  const playback = createAudioPlaybackOrder({ key, getQueue: () => queue, getActive: () => active, random: () => 0 })

  playback.setShuffled(true)
  assert.equal(playback.ordered()[0], active)
  assert.deepEqual(playback.ordered(), playback.ordered())
  assert.equal(playback.adjacent(1), playback.ordered()[1])
})

test('queue mutations rebuild shuffle without retaining removed entries', () => {
  const queue = [{ id: 'one' }, { id: 'two' }, { id: 'three' }]
  let active = queue[0]
  const playback = createAudioPlaybackOrder({ key, getQueue: () => queue, getActive: () => active, random: () => 0.5 })

  playback.setShuffled(true)
  queue.splice(1, 1)
  const current = playback.ordered()
  assert.deepEqual(new Set(current.map(key)), new Set(['one', 'three']))
  assert.equal(playback.adjacent(1), current[1])
})

test('ordinary playback follows queue order and has bounded edges', () => {
  const queue = [{ id: 'one' }, { id: 'two' }]
  let active = queue[0]
  const playback = createAudioPlaybackOrder({ key, getQueue: () => queue, getActive: () => active })

  assert.equal(playback.adjacent(1), queue[1])
  assert.equal(playback.adjacent(-1), undefined)
  active = queue[1]
  assert.equal(playback.adjacent(1), undefined)
})
