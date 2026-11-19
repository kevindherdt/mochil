import assert from 'node:assert/strict'
import test from 'node:test'
import { replayGainFactor } from '../static/replaygain.js'

test('track gain converts decibels to a linear factor', () => {
  assert.ok(Math.abs(replayGainFactor({ replaygain_track_db: -6 }, 'track') - 0.501187) < 0.00001)
})

test('album mode falls back to track metadata', () => {
  assert.ok(Math.abs(replayGainFactor({ replaygain_track_db: -3 }, 'album') - 0.707946) < 0.00001)
})

test('peak metadata prevents clipping', () => {
  assert.equal(replayGainFactor({ replaygain_track_db: 6, replaygain_track_peak: 0.8 }, 'track'), 1.25)
})

test('gain is bounded and off remains neutral', () => {
  assert.equal(replayGainFactor({ replaygain_track_db: 30 }, 'track'), 10 ** (12 / 20))
  assert.equal(replayGainFactor({ replaygain_track_db: -8 }, 'off'), 1)
  assert.equal(replayGainFactor({}, 'track'), 1)
})
