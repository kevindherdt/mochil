import assert from 'node:assert/strict'
import test from 'node:test'
import { backdropURL, createPlayerTint, dominantColour } from '../static/audio-player-tint.js'

const pixels = (...colours) => Uint8ClampedArray.from(colours.flatMap(([r, g, b, a = 255]) => [r, g, b, a]))

test('a coloured cover gives its colour', () => {
  assert.equal(dominantColour(pixels([200, 40, 40], [200, 40, 40])), 'rgb(200 40 40)')
})

test('greys, blacks, whites and transparent pixels do not count', () => {
  assert.equal(dominantColour(pixels([0, 0, 0], [255, 255, 255], [120, 120, 120], [255, 0, 0, 0])), '')
  assert.equal(dominantColour(pixels([255, 255, 255], [30, 90, 200])), 'rgb(30 90 200)')
})

test('the tint follows the cover that is in the player and goes when there is none', () => {
  const props = new Map()
  const player = { style: { setProperty: (k, v) => props.set(k, v), removeProperty: k => props.delete(k) } }
  let current = null
  const art = { querySelector: () => current }
  let observe
  class Observer { constructor(fn) { observe = fn } observe() {} }
  const canvas = { getContext: () => ({ drawImage() {}, getImageData: () => ({ data: pixels([20, 120, 60]) }) }) }
  createPlayerTint({ player, art, document: { createElement: () => canvas }, MutationObserverClass: Observer })
  assert.equal(props.has('--art-tint'), false)
  current = { complete: true, naturalWidth: 300, src: 'http://x/a.jpg' }
  observe()
  assert.equal(props.get('--art-tint'), 'rgb(20 120 60)')
  assert.equal(props.get('--art-image'), 'url("http://x/a.jpg")')
  current = null
  observe()
  assert.equal(props.has('--art-tint'), false)
  assert.equal(props.has('--art-image'), false)
})

test('the backdrop address cannot break out of its url()', () => {
  assert.equal(backdropURL('/a\\"b.jpg'), 'url("/a%5C%22b.jpg")')
})
