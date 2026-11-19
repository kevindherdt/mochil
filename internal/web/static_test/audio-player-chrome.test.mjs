import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioPlayerChrome, nextSleepMinutes } from '../static/audio-player-chrome.js'

function fixture() {
  let iconWrites = 0
  const icon = { set innerHTML(value) { iconWrites++ } }
  const nodes = new Map()
  const node = () => ({
    dataset: {}, style: { setProperty() {} }, classList: { toggle() {} },
    setAttribute(key, value) { this[key] = value },
    querySelector() { return icon }, textContent: '',
  })
  const $ = id => { if (!nodes.has(id)) nodes.set(id, node()); return nodes.get(id) }
  const seek = node()
  const chrome = createAudioPlayerChrome({ document: { body: node() }, $, player: node(), seek, clock: n => String(n) })
  return { chrome, $, seek, writes: () => iconWrites }
}

test('steady playback updates timeline without replacing play icons each tick', () => {
  const f = fixture()
  for (let i = 0; i < 100; i++) f.chrome.updateTimeline({ duration: 300, currentTime: i / 4, paused: false }, {})
  assert.equal(f.writes(), 2)
  assert.equal(f.seek['aria-valuetext'], '24.75 of 300')
  f.chrome.updateTimeline({ duration: 300, currentTime: 25, paused: true }, {})
  assert.equal(f.writes(), 4)
  assert.equal(f.$('#audio-play')['aria-label'], 'Play')
})

test('seek presentation stays bounded and explains unavailable duration', () => {
  const f = fixture()
  f.chrome.updateTimeline({ duration: NaN, currentTime: Infinity, paused: true }, { duration_seconds: Infinity })
  assert.equal(f.seek.value, 0)
  assert.equal(f.seek['aria-valuetext'], '0, duration unavailable')
  f.chrome.updateTimeline({ duration: 10, currentTime: 11, paused: true }, {})
  assert.equal(f.seek.value, 1000)
})

test('sleep timer keeps its accessible purpose and resets on expiry', t => {
  t.mock.timers.enable({ apis: ['setTimeout'] })
  const f = fixture()
  let pauses = 0
  f.chrome.cycleSleep({ pause() { pauses++ } })
  assert.equal(f.$('#audio-sleep')['aria-label'], 'Sleep timer: 15 minutes')
  t.mock.timers.tick(15 * 60_000)
  assert.equal(pauses, 1)
  assert.equal(f.$('#audio-sleep')['aria-label'], 'Sleep timer: off')
  assert.equal(f.$('#audio-sleep').dataset.minutes, 0)
})

test('audio sleep timer cycles through bounded useful choices', () => {
  assert.deepEqual([0, 15, 30, 60, 0].map(nextSleepMinutes), [15, 30, 60, 0, 15])
  assert.equal(nextSleepMinutes('damaged'), 15)
})
