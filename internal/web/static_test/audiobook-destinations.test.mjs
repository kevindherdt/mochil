import assert from 'node:assert/strict'
import test from 'node:test'

import { createAudiobookDestinations } from '../static/audiobook-destinations.js'

test('audiobook destinations expose spoken-media concepts only', () => {
  const previousFrame = globalThis.requestAnimationFrame
  globalThis.requestAnimationFrame = callback => { callback(); return 1 }
  const element = {
    hidden: true,
    innerHTML: '',
    querySelectorAll: () => [],
    addEventListener() {},
    classList: { add() {}, toggle() {}, remove() {} },
    scrollLeft: 0,
    clientWidth: 500,
    scrollWidth: 500,
  }
  const select = { value: 'series' }
  const rail = createAudiobookDestinations({ element, select, onChange() {} })
  for (const label of ['Continue', 'Recent', 'Series', 'Authors', 'All']) assert.match(element.innerHTML, new RegExp(`>${label}<`))
  assert.doesNotMatch(element.innerHTML, /Albums|Artists|Playlists|Tracks/)
  rail.setVisible(true)
  assert.equal(element.hidden, false)
  globalThis.requestAnimationFrame = previousFrame
})
