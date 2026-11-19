import test from 'node:test'
import assert from 'node:assert/strict'

test('the music-names observer is scoped to #audio-items, not the whole document', async () => {
  const audioItems = { id: 'audio-items', scoped: true }
  const body = { id: 'body', scoped: true }
  let observed = null
  class FakeObserver {
    constructor(fn) { this.fn = fn }
    observe(root, options) { observed = { root, options } }
  }
  globalThis.MutationObserver = FakeObserver
  globalThis.document = {
    getElementById: (id) => (id === 'audio-items' ? audioItems : null),
    body,
    querySelectorAll: () => [],
  }
  const { installMusicNames } = await import('../static/music-names.js?scoped-test=1')
  installMusicNames()
  assert.ok(observed, 'installMusicNames must start an observer')
  assert.equal(observed.root, audioItems, 'the observer must be scoped to #audio-items, not document.body')
  assert.deepEqual(observed.options, { childList: true, subtree: true })
})

test('the music-names observer falls back to document.body if #audio-items is not there yet', async () => {
  const body = { id: 'body' }
  let observed = null
  class FakeObserver {
    constructor(fn) { this.fn = fn }
    observe(root, options) { observed = { root, options } }
  }
  globalThis.MutationObserver = FakeObserver
  globalThis.document = {
    getElementById: () => null,
    body,
    querySelectorAll: () => [],
  }
  const { installMusicNames } = await import('../static/music-names.js?fallback-test=1')
  installMusicNames()
  assert.ok(observed, 'installMusicNames must start an observer')
  assert.equal(observed.root, body)
})
