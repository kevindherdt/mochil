import test from 'node:test'
import assert from 'node:assert/strict'
import { applyCoverTint } from '../static/cover-tint.js'

test('with no cover or no picture the page keeps its colours', () => {
  const style = new Map(), target = { style: { setProperty: (k, v) => style.set(k, v) } }
  applyCoverTint(target, null)
  applyCoverTint(null, {})
  const img = { complete: false, naturalWidth: 0, addEventListener() {} }
  applyCoverTint(target, img, { createElement() { throw new Error('should not draw') } })
  assert.equal(style.size, 0)
})

test('a loaded cover sets --tint from the player picker, and a failing canvas leaves it unset', () => {
  const style = new Map(), target = { style: { setProperty: (k, v) => style.set(k, v) } }
  const img = { complete: true, naturalWidth: 10, addEventListener() {} }
  const doc = { createElement: () => ({ getContext: () => ({ drawImage() {}, getImageData: () => ({ data: new Uint8ClampedArray([40, 90, 200, 255]) }) }) }) }
  applyCoverTint(target, img, doc)
  assert.match(style.get('--tint'), /^rgb\(/)
  const broken = new Map()
  applyCoverTint({ style: { setProperty: (k, v) => broken.set(k, v) } }, img, { createElement: () => { throw new Error('tainted') } })
  assert.equal(broken.size, 0)
})
