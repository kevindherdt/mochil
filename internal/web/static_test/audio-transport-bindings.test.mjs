import test from 'node:test'
import assert from 'node:assert/strict'
import { applyAudioShuffle, clearClosedAudioSession, openNextAudioItem } from '../static/audio-transport-bindings.js'

const button = () => ({
  classList: { values: new Set(), toggle(name, enabled) { enabled ? this.values.add(name) : this.values.delete(name) } },
  attributes: {},
  setAttribute(name, value) { this.attributes[name] = value },
})

test('expanded transport reflects shuffle state', () => {
  const controls = [button()]
  const playbackOrder = { setShuffled: value => Boolean(value) }
  assert.equal(applyAudioShuffle(true, playbackOrder, controls), true)
  for (const control of controls) {
    assert.equal(control.classList.values.has('active'), true)
    assert.equal(control.attributes['aria-pressed'], 'true')
  }
})

test('ended converted audio opens the next item with autoplay intent', () => {
  const next = { id: 'next' }
  const calls = []

  assert.equal(openNextAudioItem(next, null, (...args) => calls.push(args)), true)
  assert.deepEqual(calls, [[next, true]])
})

test('ended direct audio preserves a prepared gapless source', () => {
  const next = { id: 'next' }
  const prepared = { audio: 'secondary deck' }
  const calls = []

  assert.equal(openNextAudioItem(next, prepared, (...args) => calls.push(args)), true)
  assert.deepEqual(calls, [[next, true, prepared]])
  assert.equal(openNextAudioItem(null, null, () => assert.fail('must not open')), false)
})

test('closing audio clears stale operating-system media state', () => {
  const mediaSession = { metadata: { title: 'Old track' }, playbackState: 'playing' }

  clearClosedAudioSession(mediaSession)

  assert.equal(mediaSession.metadata, null)
  assert.equal(mediaSession.playbackState, 'none')
  assert.doesNotThrow(() => clearClosedAudioSession(null))
})
