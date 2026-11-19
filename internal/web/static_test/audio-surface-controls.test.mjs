import assert from 'node:assert/strict'
import test from 'node:test'
import { createAudioSurfaceControls } from '../static/audio-surface-controls.js'

function toggle() {
  const buttons = ['covers', 'list'].map(layout => ({ dataset: { audioLayout: layout }, pressed: '', setAttribute(name, value) { this.pressed = value } }))
  const listeners = {}
  return {
    hidden: true, buttons,
    addEventListener: (name, fn) => { listeners[name] = fn },
    ownerDocument: { addEventListener() {} },
    querySelectorAll: () => buttons,
    click: layout => listeners.click({ target: { closest: () => buttons.find(button => button.dataset.audioLayout === layout) } }),
  }
}

function fixture() {
  const elements = new Map()
  const $ = selector => {
    if (!elements.has(selector)) elements.set(selector, selector === '#audio-layout-toggle' ? toggle() : { hidden: false, value: 'covers', dispatchEvent() {} })
    return elements.get(selector)
  }
  const music = [], audiobooks = []
  const controls = createAudioSurfaceControls({
    $,
    musicDestinations: { setVisible: value => music.push(value) },
    audiobookDestinations: { setVisible: value => audiobooks.push(value) },
  })
  return { $, controls, music, audiobooks }
}

test('audio media families expose only their relevant controls', () => {
  const { $, controls, music, audiobooks } = fixture()
  controls.show('podcasts')
  assert.equal($('#podcast-manage-toggle').hidden, false)
  assert.equal($('#music-layout-field').hidden, true)
  assert.equal($('#audiobook-series-layout-field').hidden, true)
  controls.show('audiobooks')
  assert.equal($('#podcast-manage-toggle').hidden, true)
  assert.equal($('#audiobook-series-layout-field').hidden, false)
  assert.equal($('#audio-sort option[value="series"]').hidden, false)
  controls.show('music')
  assert.equal($('#music-layout-field').hidden, false)
  assert.equal($('#audiobook-series-layout-field').hidden, true)
  assert.deepEqual(music, [false, false, true])
  assert.deepEqual(audiobooks, [false, true, false])
})

test('route changes always dismiss transient playlist creation', () => {
  const { $, controls } = fixture()
  $('#playlist-create').hidden = false
  controls.show('music')
  assert.equal($('#playlist-create').hidden, true)
  assert.equal($('#music-browse-field').hidden, true)
})

test('the cover/list toggle drives the layout that fits what is on screen', () => {
  const { $, controls } = fixture()
  controls.show('music')
  assert.equal($('#audio-layout-toggle').hidden, false)
  $('#audio-layout-toggle').click('list')
  assert.equal($('#music-layout').value, 'list')
  assert.equal($('#audio-layout-toggle').buttons[1].pressed, 'true')
  controls.show('podcasts')
  assert.equal($('#podcast-layout').value, 'covers', 'podcasts keep their own layout')
})
