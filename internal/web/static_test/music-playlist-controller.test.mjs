import test from 'node:test'
import assert from 'node:assert/strict'

import { createMusicPlaylistController } from '../static/music-playlist-controller.js'

function harness({ confirmAction = () => true } = {}) {
  const elements = {
    '#playlist-create': { hidden: true },
    '#playlist-name': { value: '', focus() {} },
    '#playlist-status': { textContent: '' },
    '#playlist-create-cancel': {},
    '[data-playlist-rename-form]': { hidden: true, input: { value: 'Night' }, querySelector() { return this.input } },
    '[data-playlist-rename-form] p': { textContent: '' },
  }
  const container = { addEventListener(type, callback) { this[type] = callback } }
  const calls = { rendered: 0, opened: [], queues: [], deleted: [], saved: [], added: [] }
  const music = {
    playlists: [{ name: 'Night', items: [{ id: 'one' }, { id: 'two' }] }],
    setQueue: items => calls.queues.push(items),
    deletePlaylist: async name => calls.deleted.push(name),
    editPlaylist: async (name, transform) => calls.saved.push({ name, items:transform(music.playlists.find(list => list.name === name).items) }),
    addPlaylistItem: async (name, item) => calls.added.push({ name, item }),
    createPlaylist: async () => {},
    renamePlaylist: async () => {},
  }
  const controller = createMusicPlaylistController({
    $: selector => elements[selector], container, music,
    key: item => item.id,
    render: () => { calls.rendered++ },
    open: (...values) => calls.opened.push(values),
    confirmAction,
  })
  return { controller, music, calls, container, elements }
}

function eventWith(matches) {
  return { target: { closest: selector => matches[selector] || null } }
}

function mockRenameData(t) {
  const descriptor = Object.getOwnPropertyDescriptor(globalThis, 'FormData')
  Object.defineProperty(globalThis, 'FormData', { configurable: true, writable: true, value: class { get() { return 'Renamed' } } })
  t.after(() => {
    if (descriptor) Object.defineProperty(globalThis, 'FormData', descriptor)
    else delete globalThis.FormData
  })
}

test('opening a playlist selects it without leaking selection into the catalogue', async () => {
  const { controller, calls } = harness()
  const button = { dataset: { openPlaylist: 'Night' } }
  assert.equal(await controller.handleClick(eventWith({ '[data-open-playlist]': button })), true)
  assert.equal(controller.selected, 'Night')
  assert.equal(calls.rendered, 1)
})

test('playlist playback installs the complete queue before opening its first track', async () => {
  const { controller, music, calls } = harness()
  const group = { dataset: { playlist: 'Night' } }
  await controller.handleClick(eventWith({ '[data-play-playlist]': {}, '[data-playlist]': group }))
  assert.deepEqual(calls.queues, [music.playlists[0].items])
  assert.deepEqual(calls.opened, [[music.playlists[0].items[0], true]])
})

test('cancelled playlist deletion never mutates music state', async () => {
  const { controller, calls } = harness({ confirmAction: () => false })
  const group = { dataset: { playlist: 'Night' } }
  await controller.handleClick(eventWith({ '[data-delete-playlist]': {}, '[data-playlist]': group }))
  assert.deepEqual(calls.deleted, [])
})

test('adding an existing track to a playlist remains idempotent', async () => {
  const { controller, music, calls } = harness()
  const wrapper = { querySelector: () => ({ value: 'Night' }) }
  const event = eventWith({
    '[data-action]': { dataset: { action: 'playlist' } },
    '.audio-playlist-add': wrapper,
  })
  await controller.handleItemAction(event, music.playlists[0].items[0])
  assert.deepEqual(calls.saved, [])
  assert.deepEqual(calls.added, [])
})

test('adding a new track submits one item rather than a replacement list', async () => {
  const {controller, calls} = harness()
  const item = {id:'new'}
  await controller.handleItemAction(eventWith({
    '[data-action]': {dataset:{action:'playlist'}},
    '.audio-playlist-add': {querySelector:()=>({value:'Night'})},
  }), item)
  assert.deepEqual(calls.added, [{name:'Night',item}])
  assert.deepEqual(calls.saved, [])
})

test('late failed rename cannot restore a playlist after navigation', async t => {
  mockRenameData(t)
  const h = harness()
  h.controller.setSelected('Night')
  let reject
  h.music.renamePlaylist = () => new Promise((_, fail) => { reject = fail })
  const form = h.elements['[data-playlist-rename-form]']
  const pending = h.container.submit({ ...eventWith({ '[data-playlist-rename-form]': form }), preventDefault() {}, submitter: null })
  h.controller.reset()
  h.controller.setSelected('Night') // Returning to the same name is still a new visit.
  reject(new Error('offline'))
  await pending
  assert.equal(h.controller.selected, 'Night')
  assert.equal(h.calls.rendered, 0)
  assert.equal(h.elements['[data-playlist-rename-form] p'].textContent, '')
})

test('current rename failure keeps its retry form and message visible', async t => {
  mockRenameData(t)
  const h = harness()
  h.controller.setSelected('Night')
  h.music.renamePlaylist = async () => { throw new Error('Could not save') }
  const form = h.elements['[data-playlist-rename-form]']
  await h.container.submit({ ...eventWith({ '[data-playlist-rename-form]': form }), preventDefault() {}, submitter: null })
  assert.equal(h.controller.selected, 'Night')
  assert.equal(form.hidden, false)
  assert.equal(form.input.value, 'Renamed')
  assert.equal(h.elements['[data-playlist-rename-form] p'].textContent, 'Could not save')
})

test('playlist creation handles submissions without a submitter', async () => {
  const h = harness()
  h.elements['#playlist-name'].value = 'New playlist'
  await h.elements['#playlist-create'].onsubmit({ preventDefault() {}, submitter: null })
  assert.equal(h.elements['#playlist-create'].hidden, true)
  assert.equal(h.elements['#playlist-name'].value, '')
})
