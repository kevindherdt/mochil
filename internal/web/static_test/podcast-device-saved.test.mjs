import test from 'node:test'
import assert from 'node:assert/strict'
import { podcastBadgesHTML, createDeviceSaved } from '../static/podcast-device-saved.js'

const esc = text => String(text)

test('badges show only the copies that exist', () => {
  assert.equal(podcastBadgesHTML({ id: '1' }, esc), '')
  assert.match(podcastBadgesHTML({ id: '1', server_downloaded: true }, esc), /On server/)
})

test('starting a device download remembers the episode and badges the row', () => {
  const store = {}
  globalThis.localStorage = { getItem: key => store[key] ?? null, setItem: (key, value) => { store[key] = value } }
  const added = []
  const badges = { insertAdjacentHTML: (_, html) => added.push(html) }
  const link = { dataset: {} }
  const row = { dataset: { item: '7' }, querySelector: selector => selector === '.episode-badges' ? badges : null }
  link.closest = () => row
  let handler
  createDeviceSaved({ container: { addEventListener: (_, fn) => { handler = fn } } })
  handler({ target: { closest: () => link } })
  assert.deepEqual(JSON.parse(store['mochil-device-saved']), ['7'])
  assert.equal(link.dataset.state, 'ready')
  assert.match(added[0], /On this device/)
  assert.match(podcastBadgesHTML({ id: '7' }, esc), /On this device/)
  delete globalThis.localStorage
})
