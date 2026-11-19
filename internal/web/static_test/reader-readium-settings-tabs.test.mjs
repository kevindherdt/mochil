import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'
import { runInNewContext } from 'node:vm'

const htmlPath = new URL('../static/reader-readium.html', import.meta.url)
const jsPath = new URL('../static/reader-readium.js', import.meta.url)

const ALL_CONTROL_IDS = [
  'size-down', 'size-up', 'size-readout', 'reading-font', 'reading-font-upload', 'reading-font-remove', 'reading-font-status', 'reading-line-height',
  'reading-flow', 'reading-columns', 'reading-margin',
  'reading-theme', 'reading-warmth', 'reading-warmth-value',
  'tap-zones-select',
  'narration-language', 'narration-voice', 'narration-speed', 'narration-sleep', 'narration-sleep-left',
]

test('every pre-existing settings control id is still present after the reorganization', async () => {
  const html = await readFile(htmlPath, 'utf8')
  for (const id of ALL_CONTROL_IDS) assert.ok(html.includes(`id="${id}"`), `${id} is still present`)
})

test('the settings dialog is organized into a tablist with one tab per former section', async () => {
  const html = await readFile(htmlPath, 'utf8')
  assert.match(html, /role="tablist"/)
  for (const label of ['Text', 'Layout', 'Appearance', 'Page turning', 'Read aloud']) {
    assert.match(html, new RegExp(`<button[^>]*role="tab"[^>]*>${label}</button>`), `a tab button exists for ${label}`)
  }
  const tabpanelCount = (html.match(/role="tabpanel"/g) || []).length
  assert.equal(tabpanelCount, 5, 'five tab panels (Text, Layout, Appearance, Page turning, Read aloud)')
})

test('only the first tab panel is visible until a different tab is chosen', async () => {
  const html = await readFile(htmlPath, 'utf8')
  const panels = [...html.matchAll(/<section[^>]*role="tabpanel"[^>]*>/g)].map(m => m[0])
  assert.equal(panels.length, 5)
  assert.ok(!/\shidden/.test(panels[0]), 'first panel starts visible')
  for (const panel of panels.slice(1)) assert.ok(/\shidden/.test(panel), 'later panels start hidden')
})

test('the read-aloud tab stays hidden until narration mounts, mirroring its panel', async () => {
  const html = await readFile(htmlPath, 'utf8')
  const narrationTab = html.match(/<button[^>]*data-rs-tab="narration"[^>]*>/)[0]
  assert.match(narrationTab, /\shidden/, 'the narration tab button starts hidden like its panel')
})

// Minimal fake DOM: enough for the tab-wiring block to run against, modeled after the
// querySelector/dataset/classList surface the real elements expose.
function fakeTabElements() {
  const tabs = ['text', 'layout', 'appearance', 'paging', 'narration'].map(name => ({
    dataset: { rsTab: name },
    attrs: {},
    hidden: name === 'narration',
    tabIndex: 0,
    listeners: {},
    setAttribute(key, value) { this.attrs[key] = value },
    getAttribute(key) { return this.attrs[key] },
    addEventListener(type, fn) { this.listeners[type] = fn },
    onclick: null,
    focus() { this.focused = true },
  }))
  const panels = ['text', 'layout', 'appearance', 'paging', 'narration'].map(name => ({ dataset: { rsPanel: name }, hidden: name !== 'text' }))
  return { tabs, panels }
}

test('clicking a tab shows only its panel and marks it selected', async () => {
  const source = await readFile(jsPath, 'utf8')
  const block = source.match(/\/\/ Settings dialog tabs[\s\S]*?\n(?=\s*\$\('#settings-open'\)\.onclick)/)
  assert.ok(block, 'the tab-wiring block exists in reader-readium.js')
  const { tabs, panels } = fakeTabElements()
  const document = {
    querySelectorAll: selector => (selector.includes('data-rs-tab') ? tabs : panels),
  }
  const sessionData = new Map()
  const sessionStorage = { getItem: k => sessionData.get(k) ?? null, setItem: (k, v) => sessionData.set(k, v) }
  runInNewContext(block[0], { document, sessionStorage })
  const layoutTab = tabs[1]
  layoutTab.onclick()
  assert.equal(panels.find(p => p.dataset.rsPanel === 'text').hidden, true)
  assert.equal(panels.find(p => p.dataset.rsPanel === 'layout').hidden, false)
  assert.equal(layoutTab.attrs['aria-selected'], 'true')
  assert.equal(tabs[0].attrs['aria-selected'], 'false')
})

test('arrow keys move selection to the next or previous tab', async () => {
  const source = await readFile(jsPath, 'utf8')
  const block = source.match(/\/\/ Settings dialog tabs[\s\S]*?\n(?=\s*\$\('#settings-open'\)\.onclick)/)
  const { tabs, panels } = fakeTabElements()
  const document = { querySelectorAll: selector => (selector.includes('data-rs-tab') ? tabs : panels) }
  const sessionStorage = { getItem: () => null, setItem() {} }
  runInNewContext(block[0], { document, sessionStorage })
  tabs[0].listeners.keydown({ key: 'ArrowRight', preventDefault() {} })
  assert.equal(tabs[1].focused, true)
  assert.equal(panels.find(p => p.dataset.rsPanel === 'layout').hidden, false)
})
