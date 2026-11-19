import test from 'node:test'
import assert from 'node:assert/strict'
import { createDestinationScrollMemory, createMusicDestinationState, restoreAudioDestination } from '../static/destination-scroll-memory.js'

test('restores an independent position for each destination', () => {
  const calls = []
  const viewport = { scrollY: 180, scrollTo: value => calls.push(value) }
  const memory = createDestinationScrollMemory({ viewport })
  memory.remember('music:albums')
  viewport.scrollY = 42
  memory.remember('music:artists')

  assert.equal(memory.restore('music:albums'), true)
  assert.deepEqual(calls.at(-1), { top: 180, behavior: 'auto' })
  assert.equal(memory.restore('music:unknown'), false)
})

test('bounds retained destination positions', () => {
  const calls = []
  const viewport = { scrollY: 1, scrollTo: value => calls.push(value) }
  const memory = createDestinationScrollMemory({ viewport, maximum: 2 })
  memory.remember('one')
  memory.remember('two')
  memory.remember('three')

  assert.equal(memory.restore('one'), false)
  assert.equal(memory.restore('two'), true)
})

test('waits for two frames when the viewport can schedule them', () => {
  const frames = []
  const calls = []
  const viewport = {
    scrollY: 90,
    scrollTo: value => calls.push(value),
    requestAnimationFrame: callback => frames.push(callback),
  }
  const memory = createDestinationScrollMemory({ viewport })
  memory.remember('music:tracks')
  memory.restore('music:tracks')

  assert.equal(calls.length, 0)
  frames.shift()()
  assert.equal(calls.length, 0)
  frames.shift()()
  assert.deepEqual(calls, [{ top: 90, behavior: 'auto' }])
})

test('music destination state persists selection and restores its browsing context', async () => {
  const calls = []
  const stored = []
  const select = { value: 'albums' }
  const sort = { value: 'title' }
  const state = createMusicDestinationState({
    select, sort,
    storage: { setItem: (...args) => stored.push(args) },
    viewport: { scrollY: 160, scrollTo: value => calls.push(value) },
    getLibrary: () => 'music',
    closeMenu() {}, resetWindow() {}, resetPlaylist() {}, hideCreate() {}, async reload() {},
  })
  select.value = 'artists'
  await state.change()
  select.value = 'albums'
  await state.change()
  assert.deepEqual(stored.at(-1), ['mochil-music-browse', 'albums'])
  assert.deepEqual(calls.at(-1), { top: 160, behavior: 'auto' })
  state.select('tracks')
  state.selectSort('recent')
  assert.equal(select.value, 'tracks')
  assert.equal(sort.value, 'recent')
  assert.deepEqual(stored.slice(-2), [
    ['mochil-music-browse', 'tracks'],
    ['mochil-audio-sort-music', 'recent'],
  ])
})

test('restores media-specific destination and a valid saved sort', () => {
  const sort = { value: '', options: [{ value: 'artist' }] }
  const music = { value: '' }
  const audiobooks = { value: '' }
  let current = ''
  restoreAudioDestination({ kind: 'music', sort, savedSort: 'artist', music, audiobooks, storage: { getItem: () => 'favorites' }, onMusic: value => { current = value } })
  assert.equal(sort.value, 'artist')
  assert.equal(music.value, 'favorites')
  assert.equal(current, 'favorites')
})
