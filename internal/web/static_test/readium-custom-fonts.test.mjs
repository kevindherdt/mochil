import test from 'node:test'
import assert from 'node:assert/strict'
import { createCustomFontManager } from '../static/readium-custom-fonts.js'
import { loadReaderSettings } from '../static/reader-settings.js'

test('custom font selection is restored only for safe generated identifiers', () => {
  const storage = { getItem(key) { return key === 'mochil-readium-settings' ? JSON.stringify({fontFamily: 'BoboFont0123456789abcdef'}) : null } }
  assert.equal(loadReaderSettings(storage).fontFamily, 'BoboFont0123456789abcdef')
})

test('adding the first font installs it into already opened chapters', async () => {
  const records = new Map()
  const store = { async list() { return [] }, async put(f) { records.set(f.id, f) }, async remove(id) { records.delete(id) } }
  const installed = []
  class Face { constructor(name) { this.family = name } async load() { return this } }
  const manager = createCustomFontManager({store, FontFace: Face, randomId: () => '0123456789abcdef'})
  await manager.install({ fonts: { add: face => installed.push(face.family) } })
  await manager.add({ name: 'Test.ttf', size: 4, arrayBuffer: async () => new ArrayBuffer(4) })
  assert.deepEqual(installed, ['BoboFont0123456789abcdef'])
})

test('uploaded font persists, restores and is installed in chapter documents; removal falls back safely', async () => {
  const records = new Map()
  const store = { async list() { return [...records.values()] }, async put(font) { records.set(font.id, font) }, async remove(id) { records.delete(id) } }
  const installed = []
  class Face { constructor(name, bytes) { this.family = name; this.bytes = bytes } async load() { return this } }
  const doc = () => ({ fonts: { add(face) { installed.push(face.family) }, delete(face) { installed.splice(installed.indexOf(face.family), 1) } } })
  const first = createCustomFontManager({ store, FontFace: Face, randomId: () => '0123456789abcdef' })
  const file = { name: 'Sample.ttf', size: 4, arrayBuffer: async () => new Uint8Array([0,1,0,0]).buffer }
  const font = await first.add(file)
  assert.equal(font.id, 'BoboFont0123456789abcdef')
  const restored = createCustomFontManager({ store, FontFace: Face })
  await restored.load()
  assert.deepEqual(restored.list().map(f => f.id), [font.id])
  const frame = doc()
  await restored.install(frame)
  assert.deepEqual(installed, [font.id])
  await restored.remove(font.id)
  assert.deepEqual(installed, [])
  assert.equal(records.size, 0)
  assert.deepEqual(restored.list(), [])
})


test('rejects fonts beyond the storage count limit without writing', async () => {
  const records = new Map()
  const store = { async list() { return [...records.values()] }, async put(f) { records.set(f.id, f) }, async remove(id) { records.delete(id) } }
  class Face { async load() { return this } }
  let n = 0
  const manager = createCustomFontManager({ store, FontFace: Face, randomId: () => (++n).toString(16).padStart(16, '0') })
  const file = { name: 'a.ttf', size: 1, arrayBuffer: async () => new ArrayBuffer(1) }
  for (let i = 0; i < 10; i++) await manager.add(file)
  await assert.rejects(manager.add(file), /limit|maximum/i)
  assert.equal(records.size, 10)
})


test('a forgotten chapter cannot receive a pending decoded font', async () => {
  let finish
  const store = { async list() { return [{ id: 'BoboFont0123456789abcdef', name: 'a', bytes: new ArrayBuffer(1) }] } }
  class Face { load() { return new Promise(resolve => { finish = resolve }) } }
  const manager = createCustomFontManager({ store, FontFace: Face })
  await manager.load()
  const installed = []
  const doc = { fonts: { add: face => installed.push(face) } }
  const pending = manager.install(doc)
  manager.forget(doc)
  finish()
  await pending
  assert.deepEqual(installed, [])
})


test('tracks every attached chapter and forgets replaced or detached documents', async () => {
  const { observeFontFrames } = await import('../static/readium-custom-fonts.js')
  let changed, loaded
  class Observer { constructor(callback) { changed = callback } observe() {} disconnect() {} }
  const a = { fonts: {} }, b = { fonts: {} }, c = { fonts: {} }
  const first = { contentDocument: a }, second = { contentDocument: b }
  let frames = [first, second]
  const host = { querySelectorAll: () => frames, addEventListener(type, callback) { loaded = callback }, removeEventListener() {} }
  const events = []
  const stop = observeFontFrames(host, { install: async doc => events.push(['install', doc]), forget: doc => events.push(['forget', doc]) }, () => {}, Observer)
  first.contentDocument = c
  loaded({ target: first })
  frames = [first]
  changed()
  stop()
  assert.deepEqual(events, [['install', a], ['install', b], ['forget', a], ['install', c], ['forget', b], ['forget', c]])
})


test('temporary storage failure preserves the selected custom font', async () => {
  const { restoreFontSelection } = await import('../static/readium-custom-fonts.js')
  const settings = { fontFamily: 'BoboFont0123456789abcdef' }
  await assert.rejects(restoreFontSelection(settings, { load: async () => { throw new Error('storage') } }), /storage/)
  assert.equal(settings.fontFamily, 'BoboFont0123456789abcdef')
})

test('removing a font does not overwrite a later selection', async () => {
  const { removeSelectedFont } = await import('../static/readium-custom-fonts.js')
  const settings = { fontFamily: 'BoboFont0123456789abcdef' }
  let finish
  const pending = removeSelectedFont(settings, { remove: () => new Promise(resolve => { finish = resolve }) })
  settings.fontFamily = 'serif'
  finish()
  await pending
  assert.equal(settings.fontFamily, 'serif')
})


test('simultaneous uploads cannot exceed the font limit', async () => {
  const records = new Map()
  const store = { async list() { return [...records.values()] }, async put(f) { await new Promise(resolve => setTimeout(resolve, 1)); records.set(f.id, f) } }
  class Face { async load() { return this } }
  let n = 0
  const manager = createCustomFontManager({ store, FontFace: Face, randomId: () => (++n).toString(16).padStart(16, '0') })
  const file = { name: 'a.ttf', size: 1, arrayBuffer: async () => new ArrayBuffer(1) }
  const results = await Promise.allSettled(Array.from({length: 11}, () => manager.add(file)))
  assert.equal(results.filter(result => result.status === 'fulfilled').length, 10)
  assert.equal(records.size, 10)
})


test('a chapter installation failure does not silently leave a stored upload', async () => {
  const records = new Map()
  const store = { async put(f) { records.set(f.id, f) }, async remove(id) { records.delete(id) } }
  let loads = 0
  class Face { async load() { if (++loads > 1) throw new Error('chapter decode failed'); return this } }
  const manager = createCustomFontManager({ store, FontFace: Face, randomId: () => '0123456789abcdef' })
  await manager.install({ fonts: { add() {} } })
  await assert.rejects(manager.add({name:'a.ttf', size:1, arrayBuffer:async()=>new ArrayBuffer(1)}), /chapter decode failed/)
  assert.equal(records.size, 0)
  assert.deepEqual(manager.list(), [])
})


test('a later font choice beats a slow upload and concurrent uploads', async () => {
  const { createFontSelectionActions } = await import('../static/readium-custom-fonts.js')
  const settings = { fontFamily: '' }, pending = []
  const actions = createFontSelectionActions(settings, { add: () => new Promise(resolve => pending.push(resolve)) })
  const first = actions.upload({}), second = actions.upload({})
  pending[1]({id: 'second'}); await second
  pending[0]({id: 'first'}); await first
  assert.equal(settings.fontFamily, 'second')
  const third = actions.upload({}); actions.select('serif')
  pending[2]({id: 'third'}); await third
  assert.equal(settings.fontFamily, 'serif')
})


test('font restoration times out when storage hangs and preserves selection', async () => {
  const { restoreFontSelection } = await import('../static/readium-custom-fonts.js')
  const settings = { fontFamily: 'BoboFont0123456789abcdef' }
  const result = await Promise.race([restoreFontSelection(settings, { load: () => new Promise(() => {}) }, 10).then(() => 'ok', () => 'timeout'), new Promise(resolve => setTimeout(() => resolve('hung'), 100))])
  assert.equal(result, 'timeout')
  assert.equal(settings.fontFamily, 'BoboFont0123456789abcdef')
})


test('another tab reloads added and removed fonts in existing chapters', async () => {
  const records = new Map(), channels = []
  class Channel { constructor() { channels.push(this) } postMessage() { for (const c of channels) if (c !== this) c.onmessage?.({data: 'changed'}) } close() {} }
  const store = { async list() { return [...records.values()] }, async put(f) { records.set(f.id, f) }, async remove(id) { records.delete(id) } }
  class Face { constructor(id) { this.family = id } async load() { return this } }
  const a = createCustomFontManager({store, FontFace: Face, randomId: () => '0123456789abcdef', BroadcastChannel: Channel})
  const b = createCustomFontManager({store, FontFace: Face, BroadcastChannel: Channel})
  const installed = new Set(); await b.install({ fonts: { add: f => installed.add(f.family), delete: f => installed.delete(f.family) } })
  let changed; b.onchange = () => changed?.()
  const next = () => new Promise(resolve => { changed = resolve })
  let updated = next(); const font = await a.add({name:'a.ttf', size:1, arrayBuffer:async()=>new ArrayBuffer(1)}); await updated
  assert.deepEqual(b.list().map(f => f.id), [font.id]); assert.ok(installed.has(font.id))
  updated = next(); await a.remove(font.id); await updated
  assert.deepEqual(b.list(), []); assert.equal(installed.size, 0)
})
