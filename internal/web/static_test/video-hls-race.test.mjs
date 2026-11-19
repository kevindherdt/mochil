import assert from 'node:assert/strict'
import test from 'node:test'
import { createVideoPlayer } from '../static/video-player.js'

test('closing and switching audio during HLS client loading only attaches the latest stream', async () => {
  const nodes = new Map()
  function node(id) {
    if (!nodes.has(id)) nodes.set(id, {
      style: {setProperty() {}}, classList: {toggle() {}, add() {}, remove() {}},
      addEventListener() {}, setAttribute() {}, removeAttribute(name) { if (name === 'src') this.src = '' },
      querySelectorAll() { return [] }, load() {}, pause() {}, play() { return Promise.resolve() },
      showModal() { this.open = true }, close() { this.open = false },
      canPlayType() { return '' }, paused: true, currentTime: 0, duration: 100, volume: 1,
    })
    return nodes.get(id)
  }
  let script
  const document = { getElementById: () => node('#video-player'), createElement: () => ({}),
    head: {append(value) { script = value }}, addEventListener() {}, documentElement: {} }
  const old = {document: globalThis.document, window: globalThis.window, localStorage: globalThis.localStorage, screen: globalThis.screen, fetch: globalThis.fetch}
  Object.assign(globalThis, { document, window: {addEventListener() {}},
    localStorage: {getItem() { return 'device' }, setItem() {}}, screen: {}, fetch: async () => ({}) })
  try {
    const attached = []
    class Hls {
      static isSupported() { return true }
      static Events = {MEDIA_ATTACHED: 'attached', MANIFEST_PARSED: 'manifest', LEVEL_UPDATED: 'level', ERROR: 'error', FRAG_LOADED: 'frag'}
      attachMedia() { attached.push(this) }
      on(event, callback) { if (event === 'attached') callback() }
      loadSource(source) { this.source = source }
      destroy() {}
    }
    const player = createVideoPlayer({document, $: node, escapeHTML: x => x,
      requestJSON: async url => ({data: url.startsWith('/api/video-preferences') ? {effective: {}} : null})})
    const item = id => ({library_id: 'tv', id, hls_url: 'yes', duration_seconds: 100, audio_streams: [{index: 1}, {index: 2}]})
    await player.open(item('old'), 'Old')
    assert.ok(script, 'HLS script is waiting to load')
    node('#video-close').onclick()
    await player.open(item('new'), 'New')
    node('#audio-track').value = '2'
    node('#audio-track').onchange()
    window.Hls = Hls
    script.onload()
    await new Promise(resolve => setImmediate(resolve))
    assert.equal(attached.length, 1, 'only the newest audio selection attaches')
    assert.match(attached[0].source, /\/new\/2\//)
  } finally { Object.assign(globalThis, old) }
})
