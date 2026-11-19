import assert from 'node:assert/strict'
import test from 'node:test'
import { createGaplessPlayback } from '../static/gapless-playback.js'

function deck() {
  return { src: '', preload: '', readyState: 0, paused: 0, loaded: 0, pause() { this.paused++ }, load() { this.loaded++ }, removeAttribute(name) { if (name === 'src') this.src = '' } }
}

const key = item => item.id

test('preloads one bounded next track and swaps only when ready', () => {
  const primary = deck(), secondary = deck()
  const gapless = createGaplessPlayback({ primary, secondary, key })
  assert.equal(gapless.reset({ id: 'one', stream_url: '/one' }), primary)
  assert.equal(gapless.prepare({ id: 'two', stream_url: '/two', size: 10 }), true)
  assert.equal(secondary.src, '/two')
  assert.equal(gapless.take({ id: 'two' }), null)
  secondary.readyState = 2
  assert.equal(gapless.take({ id: 'two' }), secondary)
  assert.equal(gapless.active(), secondary)
	assert.equal(gapless.prepare({ id: 'three', stream_url: '/three', size: 10 }), true)
	primary.readyState = 3
	assert.equal(gapless.take({ id: 'three' }), primary)
	assert.equal(gapless.active(), primary)
})

test('rejects oversized tracks and mismatched queue entries', () => {
  const primary = deck(), secondary = deck()
  const gapless = createGaplessPlayback({ primary, secondary, key, maximumBytes: 100 })
  gapless.reset({ id: 'one', stream_url: '/one' })
  assert.equal(gapless.prepare({ id: 'large', stream_url: '/large', size: 101 }), false)
  assert.equal(secondary.src, '')
  assert.equal(gapless.prepare({ id: 'two', stream_url: '/two', size: 100 }), true)
  secondary.readyState = 4
  assert.equal(gapless.take({ id: 'three' }), null)
})

test('manual reset cancels both decks and returns to primary', () => {
  const primary = deck(), secondary = deck()
  const gapless = createGaplessPlayback({ primary, secondary, key })
  gapless.reset({ id: 'one', stream_url: '/one' })
  gapless.prepare({ id: 'two', stream_url: '/two' })
  secondary.readyState = 4
  gapless.take({ id: 'two' })
  assert.equal(gapless.reset({ id: 'three', stream_url: '/three' }), primary)
  assert.equal(primary.src, '/three')
  assert.equal(secondary.src, '')
})
