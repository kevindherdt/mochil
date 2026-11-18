import assert from 'node:assert/strict'
import test from 'node:test'
import { createPlayerRestore, hasSavedPlayer } from '../static/audio-player-restore.js'

const memory = () => { const data = new Map(); return { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, String(v)), removeItem: k => data.delete(k) } }
const track = id => ({ id, library_id: 'music', title: `Song ${id}` })
const key = item => `${item.library_id}/${item.id}`

function harness(storage = memory(), clock = { t: 1_000_000 }) {
  const opened = []
  const audio = { currentTime: 0, listeners: [], addEventListener(name, fn) { this.listeners.push([name, fn]) } }
  const state = { active: null, queue: [] }
  const music = { get queue() { return state.queue }, setQueue: q => { state.queue = q } }
  const restore = createPlayerRestore({ storage, now: () => clock.t, windowObject: null, getAudio: () => audio, getActive: () => state.active, key, music, open: (item, fromQueue) => { opened.push([item, fromQueue]); state.active = item } })
  return { restore, opened, audio, state, clock, storage }
}

const play = (h, item) => { h.state.active = item; h.restore.touch() }

test('the playing track and the ones after it come back, paused, after a reload', () => {
  const first = harness()
  first.state.queue = ['a', 'b', 'c', 'd'].map(track)
  first.audio.currentTime = 73.6
  play(first, track('b'))

  const second = harness(first.storage, first.clock)
  assert.equal(second.restore.restore(), true)
  assert.equal(second.opened[0][0].id, 'b')
  assert.equal(second.opened[0][1], true, 'it must not rebuild the queue from the visible list')
  assert.deepEqual(second.state.queue.map(item => item.id), ['b', 'c', 'd'])
  const [, seek] = second.audio.listeners[0]
  seek()
  assert.equal(second.audio.currentTime, 73)
})

test('nothing comes back after the player was closed', () => {
  const first = harness()
  play(first, track('a'))
  first.state.active = null
  first.restore.touch()
  assert.equal(harness(first.storage, first.clock).restore.restore(), false)
})

test('an old session is not brought back', () => {
  const first = harness()
  play(first, track('a'))
  first.clock.t += 7 * 60 * 60 * 1000
  assert.equal(harness(first.storage, first.clock).restore.restore(), false)
})

test('a player that is already open is left alone, and damaged saved data is ignored', () => {
  const first = harness()
  play(first, track('a'))
  const second = harness(first.storage, first.clock)
  second.state.active = track('z')
  assert.equal(second.restore.restore(), false)
  const broken = memory()
  broken.setItem('bobobooks-audio-resume', '{not json')
  assert.equal(harness(broken).restore.restore(), false)
})

test('a single track with no queue still comes back', () => {
  const first = harness()
  play(first, track('solo'))
  const second = harness(first.storage, first.clock)
  second.restore.restore()
  assert.deepEqual(second.state.queue.map(item => item.id), ['solo'])
})

test('the page shell can tell cheaply whether there is anything to restore', () => {
  const first = harness()
  assert.equal(hasSavedPlayer(first.storage), false)
  play(first, track('a'))
  assert.equal(hasSavedPlayer(first.storage), true)
})
