import assert from 'node:assert/strict'
import test from 'node:test'

import { createLoadingHeightGuard } from '../static/loading-height-guard.js'

function surface(height) {
  const values = new Map()
  return {
    offsetHeight: height,
    style: {
      set minHeight(value) { values.set('min-height', value) },
      get minHeight() { return values.get('min-height') || '' },
      removeProperty(name) { values.delete(name) },
    },
  }
}

test('loading guard preserves and then releases the rendered catalogue height', () => {
  const element = surface(840)
  const guard = createLoadingHeightGuard(element)

  guard.lock()
  assert.equal(element.style.minHeight, '840px')
  element.offsetHeight = 0
  guard.lock()
  assert.equal(element.style.minHeight, '840px')
  guard.release()
  assert.equal(element.style.minHeight, '')
})

test('an initially empty catalogue does not gain an artificial height', () => {
  const element = surface(0)
  const guard = createLoadingHeightGuard(element)
  guard.lock()
  assert.equal(element.style.minHeight, '')
  guard.release()
})
