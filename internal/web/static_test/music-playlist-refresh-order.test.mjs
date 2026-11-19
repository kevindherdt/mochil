import test from 'node:test'
import assert from 'node:assert/strict'
import {createMusicState} from '../static/music-state.js'

const tick = () => new Promise(resolve => setImmediate(resolve))

test('external playlist refresh waits for every queued create before reading the overview', async () => {
  const writes = [], stored = [], reads = []
  const state = createMusicState({key: item => item.id, onChanged() {}, requestJSON: async (path, options = {}) => {
    if (path === '/api/audio-favorites') return {data: []}
    if (options.method === 'POST') {
      const {name} = JSON.parse(options.body)
      await new Promise(resolve => writes.push(() => { stored.push({name, count: 0}); resolve() }))
      return {data: null}
    }
    reads.push(stored.map(item => item.name))
    return {data: stored.map(item => ({...item}))}
  }})
  await state.load()
  const first = state.createPlaylist('First')
  await tick()
  const refresh = state.load({force: true})
  const second = state.createPlaylist('Second')
  await tick()
  assert.equal(reads.length, 1)
  writes[0]()
  await first
  await tick()
  assert.equal(reads.length, 1)
  writes[1]()
  await Promise.all([second, refresh])
  assert.deepEqual(reads, [[], ['First', 'Second']])
  assert.deepEqual(state.playlists.map(item => item.name), ['First', 'Second'])
})

test('failed playlist edit does not prevent a waiting refresh', async () => {
  let rejectWrite
  const state = createMusicState({key: item => item.id, onChanged() {}, requestJSON: async (path, options = {}) => {
    if (options.method) await new Promise((_, reject) => { rejectWrite = reject })
    return {data: []}
  }})
  const write = state.createPlaylist('Failed')
  const failure = assert.rejects(write, /Offline/)
  await tick()
  const refresh = state.load({force: true})
  rejectWrite(new Error('Offline'))
  await Promise.all([failure, refresh])
  assert.deepEqual(state.playlists, [])
})
