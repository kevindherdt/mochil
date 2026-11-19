import test from 'node:test'
import assert from 'node:assert/strict'
import { readFile } from 'node:fs/promises'

test('reading settings expose layout and typography controls and persist independently of position', async () => {
  const html = await readFile(new URL('../static/reader-readium.html', import.meta.url), 'utf8')
  for (const id of ['reading-flow', 'reading-font', 'reading-line-height', 'reading-margin', 'reading-theme']) assert.ok(html.includes(`id="${id}"`), `${id} is available`)
  const { loadReaderSettings, saveReaderSettings, readiumPreferences } = await import('../static/reader-settings.js')
  const data = new Map([['mochil-readium-font-size', '1.2'], ['mochil-position:books:1', 'saved-position']])
  const storage = { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v) }
  const initial = loadReaderSettings(storage)
  assert.equal(initial.fontSize, 1.2)
  const changed = { ...initial, flow: 'scrolled', fontFamily: 'sans-serif', lineHeight: 1.8, pageGutter: 24, theme: 'sepia' }
  saveReaderSettings(storage, changed)
  assert.deepEqual(loadReaderSettings(storage), changed)
  assert.equal(data.get('mochil-position:books:1'), 'saved-position')
  const prefs = readiumPreferences(changed, 'dark')
  assert.equal(prefs.scroll, true)
  assert.equal(prefs.fontFamily, 'sans-serif')
  assert.equal(prefs.lineHeight, 1.8)
  assert.equal(prefs.pageGutter, 24)
  assert.equal(prefs.backgroundColor, '#f4ecd8')
})

test('page count and scroll margins reach the Readium EPUB preference API', async () => {
  const html = await readFile(new URL('../static/reader-readium.html', import.meta.url), 'utf8')
  assert.ok(html.includes('id="reading-columns"'))
  const { loadReaderSettings, saveReaderSettings, readiumPreferences } = await import('../static/reader-settings.js')
  const data = new Map()
  const storage = { getItem: k => data.get(k) ?? null, setItem: (k, v) => data.set(k, v) }
  const settings = { ...loadReaderSettings(storage), columnCount: 2, pageGutter: 24, flow: 'scrolled' }
  saveReaderSettings(storage, settings)
  const restored = loadReaderSettings(storage)
  assert.equal(restored.columnCount, 2)
  const prefs = readiumPreferences(restored, 'light')
  assert.equal(prefs.columnCount, 2)
  assert.equal(prefs.scrollPaddingLeft, 24)
  assert.equal(prefs.scrollPaddingRight, 24)
})


test('settings remain usable when storage reads or writes fail', async () => {
  const { loadReaderSettings, saveReaderSettings } = await import('../static/reader-settings.js')
  const blocked = { getItem() { throw new Error('blocked') }, setItem() { throw new Error('blocked') } }
  const settings = loadReaderSettings(blocked)
  assert.equal(settings.fontSize, 1)
  settings.fontSize = 1.5
  assert.doesNotThrow(() => saveReaderSettings(blocked, settings))
})


test('page shortcuts do not steal keys from forms or open dialogs', async () => {
  const { runInNewContext } = await import('node:vm')
  const source = await readFile(new URL('../static/reader-readium.js', import.meta.url), 'utf8')
  const handler = source.match(/function handleKey\(event\) \{[\s\S]*?\n\}/)[0]
  const steps = []
  let open = false
  const document = { querySelector: () => open ? {} : null }
  const handleKey = runInNewContext(`(${handler})`, { document, step: n => steps.push(n), $: () => ({ click() {} }) })
  const target = selector => ({ closest: () => selector })
  handleKey({ key: 'ArrowRight', target: target(true) })
  open = true
  handleKey({ key: 'PageDown', target: target(false) })
  assert.deepEqual(steps, [])
  open = false
  handleKey({ key: 'ArrowRight', target: target(false) })
  assert.deepEqual(steps, [1])
})


test('a failed device ID write does not prevent the reader starting', async () => {
  const { runInNewContext } = await import('node:vm')
  const source = await readFile(new URL('../static/reader-readium.js', import.meta.url), 'utf8')
  const setup = source.slice(source.indexOf('const deviceKey ='), source.indexOf('const deviceName ='))
  const localStorage = { getItem: () => null, setItem() { throw Error('quota') } }
  assert.equal(runInNewContext(`${setup}\ndeviceID`, { localStorage, crypto: { randomUUID: () => 'device' } }), 'device')
})


test('column control is unavailable while scrolling and restores the saved preference', async () => {
  const { runInNewContext } = await import('node:vm')
  const source = await readFile(new URL('../static/reader-readium.js', import.meta.url), 'utf8')
  const setup = source.slice(source.indexOf('  for (const [id, key, numeric]'), source.indexOf("  $('#smaller').onclick"))
  const fields = new Map()
  const $ = selector => { if (!fields.has(selector)) fields.set(selector, {}); return fields.get(selector) }
  const readerSettings = { flow: 'scrolled', columnCount: 2 }
  runInNewContext(setup, { $, readerSettings, localStorage: {}, saveReaderSettings() {}, applyPreferences() {} })
  assert.equal($('#reading-columns').disabled, true)
  $('#reading-flow').value = 'paginated'
  $('#reading-flow').onchange()
  assert.equal($('#reading-columns').disabled, false)
  assert.equal($('#reading-columns').value, '2')
})

test('two-page choice communicates screen width constraints', async () => {
  const html = await readFile(new URL('../static/reader-readium.html', import.meta.url), 'utf8')
  assert.match(html, /Two pages[^<]*wide screens/i)
})


test('blocked theme storage falls back to the system theme', async () => {
  const { runInNewContext } = await import('node:vm')
  const source = await readFile(new URL('../static/reader-readium.js', import.meta.url), 'utf8')
  const setup = source.slice(source.indexOf('const savedTheme ='), source.indexOf('try { applyMotion'))
  const document = { documentElement: { dataset: {} } }
  runInNewContext(setup, { localStorage: { getItem() { throw Error('blocked') } }, document, matchMedia: () => ({ matches: true }) })
  assert.equal(document.documentElement.dataset.theme, 'light')
})
