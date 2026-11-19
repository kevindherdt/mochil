import test from 'node:test'
import assert from 'node:assert/strict'
import { mountReadiumNarration } from '../static/readium-narration.js?v=14'

// Narration becomes available asynchronously (a fetch round-trip) after the settings
// dialog may already be open showing some other tab. Mounting it must not pop its
// panel open on top of whatever panel is currently selected: at most one tabpanel
// should ever be visible at once, matching the Text/Layout/etc. tab behavior.

function fakeElement(overrides = {}) {
  return { hidden: false, dataset: {}, attrs: {}, setAttribute(k, v) { this.attrs[k] = v }, getAttribute(k) { return this.attrs[k] }, addEventListener() {}, closest: () => ({ hidden: false }), ...overrides }
}

function fakeDollar(extra = {}) {
  const elements = {
    '#narrate': fakeElement({ hidden: true }),
    '#narration-settings': fakeElement({ hidden: true }),
    '#reader-settings': fakeElement(),
    '#rs-tab-narration': fakeElement({ hidden: true, attrs: { 'aria-selected': 'false' } }),
    '#location': fakeElement({ textContent: '' }),
    '#narration-voice': fakeElement({ innerHTML: '', options: [], value: '' }),
    '#narration-language': null,
    '#narration-speed': fakeElement({ innerHTML: '', value: '' }),
    '#narration-sleep': fakeElement({ value: '0', onchange: null }),
    '#narration-sleep-left': fakeElement({ textContent: '' }),
    '#settings-open': fakeElement(),
    ...extra,
  }
  return selector => elements[selector]
}

function fakeNavigator() { return { currentLocator: {}, _cframes: [] } }

function setup({ narrationTabSelected }) {
  globalThis.fetch = async () => ({ ok: true, json: async () => ({ available: true, voices: [], voice: '', loaded: true }) })
  globalThis.Audio = class { addEventListener() {} removeEventListener() {} pause() {} play() { return Promise.resolve() } }
  globalThis.location = globalThis.location || { pathname: '/book' }
  // mountReadiumNarration starts a 15s status poll; keep the test process from hanging on it.
  const realSetInterval = globalThis.setInterval
  globalThis.setInterval = (fn, ms) => realSetInterval(fn, ms).unref?.() || { unref() {} }
  const storage = { getItem: () => null, setItem() {} }
  const narrationTab = fakeElement({ hidden: true, attrs: { 'aria-selected': narrationTabSelected ? 'true' : 'false' } })
  const $ = fakeDollar({ '#rs-tab-narration': narrationTab })
  return { $, storage, narrationTab }
}

test('mounting narration keeps its panel hidden when another settings tab is selected', async () => {
  const { $, storage } = setup({ narrationTabSelected: false })
  await mountReadiumNarration({ $, navigator: fakeNavigator(), order: [], step: async () => {}, Locator: {}, publication: {}, title: () => '', storage })
  assert.equal($('#narration-settings').hidden, true, 'the narration panel must stay hidden; Text is the active panel, not Read aloud')
  assert.equal($('#narrate').hidden, false, 'the play button still appears once narration is available')
  assert.equal($('#rs-tab-narration').hidden, false, 'the tab itself becomes selectable')
})

test('mounting narration reveals its panel only if its tab was already the selected one', async () => {
  const { $, storage } = setup({ narrationTabSelected: true })
  await mountReadiumNarration({ $, navigator: fakeNavigator(), order: [], step: async () => {}, Locator: {}, publication: {}, title: () => '', storage })
  assert.equal($('#narration-settings').hidden, false, 'the panel may show when its own tab is the active one')
})
