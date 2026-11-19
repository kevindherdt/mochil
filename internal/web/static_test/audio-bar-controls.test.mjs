import test from 'node:test'
import assert from 'node:assert/strict'
import { afterEnded, nextRepeatMode } from '../static/audio-bar-controls.js'

test('repeat cycles off, all, one and back', () => {
  assert.deepEqual(['off', 'all', 'one'].map(nextRepeatMode), ['all', 'one', 'off'])
})

test('a finished track moves on, wraps, or restarts depending on repeat', () => {
  assert.deepEqual(afterEnded('off', 'b', 'a'), { item: 'b' })
  assert.deepEqual(afterEnded('off', undefined, 'a'), { item: undefined })
  assert.deepEqual(afterEnded('all', undefined, 'a'), { item: 'a' })
  assert.deepEqual(afterEnded('all', 'b', 'a'), { item: 'b' })
  assert.deepEqual(afterEnded('one', 'b', 'a'), { restart: true })
})
